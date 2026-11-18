package auth

// 0082-PLAN Phase 1's fourth pass (the third amendment of 2026-11-17: R-11,
// the narrowed access-token exemption, and F-4, the store known to hold a
// previous access token; the third review's M02a and M02b). Every issuer is an
// in-process server, every foreign record names a dead token URL or the
// in-process issuer, and the package runs behind localonly's guard.

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// R-11 (the review's A1): a record of another sign-in, with its own client and
// its own refresh token, that also carries this session's access token (as a
// copy another application made would) is another sign-in's: the exemption
// belongs to a record with no refresh token of its own. It is refused, nothing
// is sent, and it is not written over. Red when the exemption takes it as
// the session's own.
func TestRecordWithItsOwnRefreshTokenAndOurAccessTokenIsAnotherSignIn_0082(t *testing.T) {
	for name, mk := range storeVariants0082() {
		t.Run(name, func(t *testing.T) {
			srv, sent := rotatingServer0082(t, nil)
			store, raw := mk()
			held := heldPlan0082(store, srv.URL+"/token", local0082(t, srv))
			other := heldPlan0082(nil, srv.URL+"/other", nil) // carries the held session's access token
			other.ClientID, other.AccountID, other.Refresh = "oaiapp_other", "acct-other", "other-refresh"
			raw.set(other)
			want := raw.bytes()
			held.Invalidate()
			before := fieldsOf0082(held)
			tok, err := held.Token(context.Background())
			refusedFully0082(t, held, before, tok, err)
			if got := sent(); len(got) != 0 {
				t.Errorf("refresh tokens sent %q, want none", got)
			}
			if _, saves := raw.counts(); saves != 0 || !bytes.Equal(raw.bytes(), want) {
				t.Errorf("the other sign-in's record was written over (%d saves)", saves)
			}
		})
	}
}

// F-4 (the review's A4): a store that keeps neither the refresh token nor the
// client, one save that fails, then the retried save and a later refresh: the
// record holds the session's previous access token, which the session knows as
// the store's, so the retry saves and the session is never stranded. Red when
// the previous access token is not kept.
func TestStoreThatKeepsNoRefreshTokenSurvivesOneFailedSave_0082(t *testing.T) {
	for name, mk := range storeVariants0082() {
		t.Run(name, func(t *testing.T) {
			srv, sent := rotatingServer0082(t, nil)
			store, raw := mk()
			raw.drop = lossy0082
			held := heldPlan0082(store, srv.URL+"/token", local0082(t, srv))
			held.Invalidate()
			if tok, err := held.Token(context.Background()); err != nil || tok.Value != "a-1" {
				t.Fatalf("refresh 1: Token = %q, %v", tok.Value, err)
			}
			raw.mu.Lock()
			raw.failSaves = 1
			raw.mu.Unlock()
			held.Invalidate()
			if tok, err := held.Token(context.Background()); err != nil || tok.Value != "a-2" {
				t.Fatalf("refresh 2: Token = %q, %v; want the rotation, kept in memory", tok.Value, err)
			}
			if _, saves := raw.counts(); saves != 1 {
				t.Fatalf("%d saves, want the first refresh's only", saves)
			}
			held.mu.Lock()
			held.nextResave = time.Time{}
			held.mu.Unlock()
			if _, err := held.Token(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err := held.Wait(context.Background()); err != nil {
				t.Fatal(err)
			}
			if rec, _ := store0082(raw); rec == nil || rec.Access != "a-2" {
				t.Errorf("the store holds %v after the retry; want the second rotation saved over the record of the previous access token", rec)
			}
			// A second failed save, after the retry's success alone: the store
			// holds what the retry wrote, and the session knows it.
			raw.mu.Lock()
			raw.failSaves = 1
			raw.mu.Unlock()
			held.Invalidate()
			if tok, err := held.Token(context.Background()); err != nil || tok.Value != "a-3" {
				t.Fatalf("refresh 3: Token = %q, %v; want the session never stranded", tok.Value, err)
			}
			held.mu.Lock()
			held.nextResave = time.Time{}
			held.mu.Unlock()
			if _, err := held.Token(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err := held.Wait(context.Background()); err != nil {
				t.Fatal(err)
			}
			if rec, _ := store0082(raw); rec == nil || rec.Access != "a-3" {
				t.Errorf("the store holds %v after the second retry; want the third rotation saved", rec)
			}
			held.Invalidate()
			if tok, err := held.Token(context.Background()); err != nil || tok.Value != "a-4" {
				t.Fatalf("refresh 4: Token = %q, %v; want the session never stranded", tok.Value, err)
			}
			if got := sent(); len(got) != 4 || got[0] != "held-plan-refresh" || got[1] != "r-1" || got[2] != "r-2" || got[3] != "r-3" {
				t.Errorf("refresh tokens sent %q; want the session's own, each rotated one in turn", got)
			}
		})
	}
}

// F-4, before the retry: a refresh that comes due while the rotation is still
// unsaved meets the record of the previous access token as the session's own,
// and saves at once.
func TestRefreshAfterAFailedSaveOnAStoreThatKeepsNoRefreshToken_0082(t *testing.T) {
	srv, _ := rotatingServer0082(t, nil)
	store := &bytesStore0082{drop: lossy0082}
	held := heldPlan0082(store, srv.URL+"/token", local0082(t, srv))
	held.Invalidate()
	if _, err := held.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	store.failSaves = 1
	store.mu.Unlock()
	held.Invalidate()
	if _, err := held.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	held.Invalidate()
	if tok, err := held.Token(context.Background()); err != nil || tok.Value != "a-3" {
		t.Fatalf("refresh 3 before any retry: Token = %q, %v", tok.Value, err)
	}
	if err := held.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	if rec, _ := store0082(store); rec == nil || rec.Access != "a-3" {
		t.Errorf("the store holds %v; want the third rotation saved", rec)
	}
}

// foreignAccessRecord0082 is a record of another sign-in with no refresh
// token, whose access token is access: with its labels foreign.
func foreignAccessRecord0082(access string) *OAuthSession {
	r := heldPlan0082(nil, deadURL0082, nil)
	r.ClientID, r.AccountID, r.Refresh, r.Access = "oaiapp_other", "acct-other", "", access
	return r
}

// N-c (the review's M02a): at the re-read after a rejected refresh, the
// session's own access token is the held session's, not that of the record the
// first read took up. The first read takes up a sibling's due rotation, its
// refresh is rejected, and the record then in the store carries no refresh
// token, foreign labels, and the sibling's access token: it is another
// sign-in's, and refused. Red when the adopted state's access token counts.
func TestReReadOwnAccessTokenIsTheHeldSessions_0082(t *testing.T) {
	store := &bytesStore0082{}
	var requests []string
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		mu.Lock()
		requests = append(requests, string(raw))
		mu.Unlock()
		store.set(foreignAccessRecord0082("sib-access"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"refresh_token_reused"}`))
	}))
	defer srv.Close()
	sib := heldPlan0082(nil, srv.URL+"/token", nil)
	sib.Access, sib.Refresh, sib.Expiry = "sib-access", "sib-refresh", time.Now().Add(-time.Minute)
	store.set(sib)
	held := heldPlan0082(store, srv.URL+"/token", local0082(t, srv))
	held.Invalidate()
	tok, err := held.Token(context.Background())
	if err == nil || !strings.Contains(err.Error(), "another sign-in") || tok.Value != "" {
		t.Errorf("Token = %q, %v; want the re-read's refusal naming another sign-in", tok.Value, err)
	}
	mu.Lock()
	n := len(requests)
	mu.Unlock()
	if n != 1 {
		t.Errorf("%d refresh requests, want the one the sibling's token made", n)
	}
}

// N-c (M02b): at the refresh's own save, the session's own access token is the
// held session's too. The first read takes up a sibling's due rotation, its
// refresh succeeds, and while the request was in flight the store came to hold
// a record with no refresh token, foreign labels and the sibling's access token:
// not the session's, so not written over.
func TestSaveOwnAccessTokenIsTheHeldSessions_0082(t *testing.T) {
	store := &bytesStore0082{}
	foreign := foreignAccessRecord0082("sib-access")
	srv, _ := rotatingServer0082(t, func(n int, _ string) {
		if n == 1 {
			store.set(foreign)
		}
	})
	sib := heldPlan0082(nil, srv.URL+"/token", nil)
	sib.Access, sib.Refresh, sib.Expiry = "sib-access", "sib-refresh", time.Now().Add(-time.Minute)
	store.set(sib)
	held := heldPlan0082(store, srv.URL+"/token", local0082(t, srv))
	held.Invalidate()
	if tok, err := held.Token(context.Background()); err != nil || tok.Value != "a-1" {
		t.Fatalf("Token = %q, %v; want the rotation, used in memory", tok.Value, err)
	}
	if want := recBytes0082(foreign); !bytes.Equal(store.bytes(), want) {
		t.Errorf("the record another sign-in wrote during the refresh was written over:\n%s", store.bytes())
	}
	if _, saves := store.counts(); saves != 0 {
		t.Errorf("%d saves, want none", saves)
	}
}

// F-4, a first save that fails before the session knows what the store holds
// (a session built by the caller, with no record noted): the store still holds
// the record the refresh started from, a record of the session's access token
// with no refresh token and no client. The retry saves over it. Red when the
// access token the first refresh spent is not noted as the store's.
func TestFailedFirstSaveOnAStoreThatKeepsNoRefreshToken_0082(t *testing.T) {
	srv, sent := rotatingServer0082(t, nil)
	store := &bytesStore0082{drop: lossy0082}
	held := heldPlan0082(store, srv.URL+"/token", local0082(t, srv))
	record := heldPlan0082(nil, deadURL0082, nil) // the held session's own access token
	record.Refresh, record.ClientID = "", ""
	store.set(record)
	store.mu.Lock()
	store.failSaves = 1
	store.mu.Unlock()
	held.Invalidate()
	if tok, err := held.Token(context.Background()); err != nil || tok.Value != "a-1" {
		t.Fatalf("Token = %q, %v; want the rotation, kept in memory", tok.Value, err)
	}
	held.mu.Lock()
	held.nextResave = time.Time{}
	held.mu.Unlock()
	if _, err := held.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := held.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	if rec, _ := store0082(store); rec == nil || rec.Access != "a-1" {
		t.Errorf("the store holds %v after the retry; want the rotation saved over the record the refresh started from", rec)
	}
	if got := sent(); len(got) != 1 {
		t.Errorf("refresh tokens sent %q, want the one", got)
	}
}

// F-4, the consent read: on a store that keeps no refresh token, the record
// holds the access token the store is known to hold, which is the session's
// own though no longer its current one; its answers are the session's. Red when
// the consent read does not know the access token the store is known to hold.
func TestStoredConsentOnAStoreThatKeepsThePreviousAccessToken_0082(t *testing.T) {
	const id = "chatgpt_plan.background_use"
	store := &bytesStore0082{}
	held := heldPlan0082(store, deadURL0082, nil)
	rec := foreignAccessRecord0082("previous-access") // no refresh token, foreign labels, a lossy store's record
	rec.Consents = withConsent("", id, true)
	store.set(rec)
	if granted, answered, err := held.StoredConsent(context.Background(), id); err != nil || answered || granted {
		t.Fatalf("before the store is known to hold it: StoredConsent = %v, %v, %v; want unanswered", granted, answered, err)
	}
	held.mu.Lock()
	held.storedAccess = "previous-access"
	held.mu.Unlock()
	if granted, answered, err := held.StoredConsent(context.Background(), id); err != nil || !answered || !granted {
		t.Errorf("StoredConsent = %v, %v, %v; want the record's answer, the store being known to hold its access token", granted, answered, err)
	}
}

// F-4, forgetting: when another process has saved a newer session of this
// sign-in, the access token the session believed the store held is gone with
// the refresh token it was noted with, and no record carrying it is the
// session's own: one that turns up with no refresh token and foreign labels is
// another sign-in's. Red when the stored access token outlives the store's
// record.
func TestStoredAccessIsForgottenWhenAnotherProcessSavedANewerSession_0082(t *testing.T) {
	store := &bytesStore0082{}
	var b *OAuthSession
	srv, _ := rotatingServer0082(t, func(n int, _ string) {
		if n == 1 {
			b.Invalidate()
			if _, err := b.Token(context.Background()); err != nil {
				t.Errorf("B: %v", err)
			}
		}
	})
	store.set(heldPlan0082(nil, srv.URL+"/token", nil))
	a := loaded0082(t, store, local0082(t, srv))
	b = loaded0082(t, store, local0082(t, srv))
	a.mu.Lock()
	a.storedAccess = "stale-access" // what A believed the store held
	a.mu.Unlock()
	a.Invalidate()
	if _, err := a.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	a.nextResave = time.Time{}
	a.mu.Unlock()
	if _, err := a.Token(context.Background()); err != nil { // the retried save finds B's newer rotation
		t.Fatal(err)
	}
	if err := a.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	if f := fieldsOf0082(a); f.StoredRefresh != "" {
		t.Fatalf("storedRefresh %q; the retry was meant to find the store moved on", f.StoredRefresh)
	}
	store.set(foreignAccessRecord0082("stale-access"))
	a.Invalidate()
	before := fieldsOf0082(a)
	tok, err := a.Token(context.Background())
	if err == nil || !strings.Contains(err.Error(), "another sign-in") || tok.Value != "" {
		t.Errorf("Token = %q, %v; want the record, which carries an access token the session no longer knows the store to hold, refused", tok.Value, err)
	}
	if after := fieldsOf0082(a); after.Access != before.Access {
		t.Errorf("the session's access token changed from %q to %q", before.Access, after.Access)
	}
}
