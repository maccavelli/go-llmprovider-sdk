package auth

// 0082-PLAN Phase 1's third pass (the second amendment of 2026-11-17: R-5, the
// own access token; the second review's missing tests B-2, N-2 and N-4). Every
// issuer is an in-process server, every foreign record names a dead token URL
// or the in-process issuer, and the package runs behind localonly's guard.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// rotatingServer0082 answers the n-th refresh with access "a-n" and refresh
// "r-n", records every refresh token it is sent, and runs hook, if set, with
// n and the token before it answers.
func rotatingServer0082(t *testing.T, hook func(n int, refresh string)) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var sent []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		refresh := ""
		var body map[string]string
		if json.Unmarshal(raw, &body) == nil {
			refresh = body[oauthRefreshToken]
		} else if form, err := parseForm0082(raw); err == nil {
			refresh = form
		}
		mu.Lock()
		sent = append(sent, refresh)
		n := len(sent)
		mu.Unlock()
		if hook != nil {
			hook(n, refresh)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"a-` + itoa0082(n) + `","refresh_token":"r-` + itoa0082(n) + `","expires_in":3600}`))
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), sent...)
	}
}

func itoa0082(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa0082(n/10) + string(rune('0'+n%10))
}

// loaded0082 is store's record as a second process would hold it: a session
// built from it, with the store and the client given, and the refresh token
// the store holds noted, as FileTokenStore.Load notes it.
func loaded0082(t *testing.T, store TokenStore, client *http.Client) *OAuthSession {
	t.Helper()
	s, err := store.Load(context.Background(), llmprovider.ProviderOpenAI)
	if err != nil || s == nil {
		t.Fatalf("Load = %v, %v", s, err)
	}
	s.Store, s.HTTPClient = store, client
	s.mu.Lock()
	s.storedRefresh = s.Refresh
	s.mu.Unlock()
	return s
}

// lossy0082 is a store that keeps neither the refresh token nor the client id
// (the review's N-1: a single process on such a store refreshed again and
// again before 0082).
func lossy0082(s *OAuthSession) { s.Refresh, s.ClientID = "", "" }

// R-5 (the second amendment): a store that keeps no refresh token and no
// client still serves its one process: the record it holds carries the
// session's own access token, which is its own, so every refresh meets it as
// the session's, adopts nothing and saves. Red before the exemption: the
// second refresh is refused, the record differing by client.
func TestLossyStoreSingleProcessKeepsRefreshing_0082(t *testing.T) {
	for name, mk := range storeVariants0082() {
		t.Run(name, func(t *testing.T) {
			srv, sent := rotatingServer0082(t, nil)
			store, raw := mk()
			raw.drop = lossy0082
			held := heldPlan0082(store, srv.URL+"/token", local0082(t, srv))
			for i := 1; i <= 3; i++ {
				held.Invalidate()
				tok, err := held.Token(context.Background())
				if err != nil || tok.Value != "a-"+itoa0082(i) {
					t.Fatalf("refresh %d: Token = %q, %v; want the rotation a-%d", i, tok.Value, err, i)
				}
				if err := held.Wait(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			if got := sent(); len(got) != 3 || got[0] != "held-plan-refresh" || got[1] != "r-1" || got[2] != "r-2" {
				t.Errorf("refresh tokens sent %q; want the session's own, each rotated one in turn", got)
			}
			if _, saves := raw.counts(); saves != 3 {
				t.Errorf("%d saves, want one per refresh", saves)
			}
		})
	}
}

// R-5, at the save: the retried save finds the rotation already written by a
// save that reported an error, in a store that keeps no refresh token: the
// record carries the rotated access token, the session's own then, and is
// written over again.
func TestRetriedSaveMeetsTheRotatedAccessToken_0082(t *testing.T) {
	srv, _ := rotatingServer0082(t, nil)
	store := &bytesStore0082{drop: lossy0082, failAfterWrite: 1}
	held := heldPlan0082(store, srv.URL+"/token", local0082(t, srv))
	held.Invalidate()
	if tok, err := held.Token(context.Background()); err != nil || tok.Value != "a-1" {
		t.Fatalf("Token = %q, %v", tok.Value, err)
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
	if _, saves := store.counts(); saves != 2 {
		t.Errorf("%d saves, want the first (which reported an error) and the retry over the rotation it wrote", saves)
	}
	if f := fieldsOf0082(held); f.StoredRefresh != "r-1" {
		t.Errorf("storedRefresh %q after the retry; want the rotation noted as the store's", f.StoredRefresh)
	}
}

// R-5, what the exemption means and what it does not: a record that carries
// the session's own current access token is the session's own, whatever else it
// says: nothing is adopted, and the session refreshes its own token and saves
// over it. A record with an empty access token never counts, on either side;
// a pasted access-only record of another sign-in carries another access token
// and is refused (TestAccessOnlyRecordOfAnotherSignInIsRefused_0082).
func TestOwnAccessTokenExemption_0082(t *testing.T) {
	t.Run("a record with the session's access token is its own", func(t *testing.T) {
		srv, sent := rotatingServer0082(t, nil)
		store := &bytesStore0082{}
		held := heldPlan0082(store, srv.URL+"/token", local0082(t, srv))
		rec := heldPlan0082(nil, deadURL0082, nil)
		rec.Refresh, rec.ClientID = "", "oaiapp_hybrid" // neither the session's token nor its client
		store.set(rec)                                  // its access token is the held one: "held-plan-access"
		held.Invalidate()
		tok, err := held.Token(context.Background())
		if err != nil || tok.Value != "a-1" {
			t.Fatalf("Token = %q, %v; want the session's own refresh", tok.Value, err)
		}
		if got := sent(); len(got) != 1 || got[0] != "held-plan-refresh" {
			t.Errorf("refresh tokens sent %q; want only the session's own", got)
		}
		if _, saves := store.counts(); saves != 1 {
			t.Errorf("%d saves, want the rotation saved over the record", saves)
		}
	})
	t.Run("an empty access token is never the session's", func(t *testing.T) {
		srv, sent := rotatingServer0082(t, nil)
		store := &bytesStore0082{}
		held := heldPlan0082(store, srv.URL+"/token", local0082(t, srv))
		held.Access = "" // nothing current: the token is due
		other := otherChatGPT0082(time.Now().Add(time.Hour), deadURL0082)
		other.Access = ""
		store.set(other)
		before := fieldsOf0082(held)
		tok, err := held.Token(context.Background())
		refusedFully0082(t, held, before, tok, err)
		if got := sent(); len(got) != 0 {
			t.Errorf("refresh tokens sent %q, want none", got)
		}
	})
	t.Run("another access token is another sign-in's", func(t *testing.T) {
		store := &bytesStore0082{}
		held := heldPlan0082(store, deadURL0082, &http.Client{Transport: noNetwork0082{t}})
		other := otherChatGPT0082(time.Now().Add(time.Hour), deadURL0082)
		other.Refresh = ""
		store.set(other)
		held.Invalidate()
		before := fieldsOf0082(held)
		tok, err := held.Token(context.Background())
		refusedFully0082(t, held, before, tok, err)
	})
}

// B-2 (the second review's Q1): two processes of one sign-in on a store with
// no refresh lock; B refreshes and saves during A's request. A's save finds
// the same sign-in with a token it does not know: it is not written. A keeps
// its rotation, its retried save stops, and its next due refresh adopts B's
// rotation, with no request of its own. Red when the save writes over a
// sibling's newer rotation.
func TestSiblingRotatedDuringRefreshOnAStoreWithNoLock_0082(t *testing.T) {
	store := &bytesStore0082{}
	var b *OAuthSession
	srv, sent := rotatingServer0082(t, func(n int, _ string) {
		if n == 1 { // A's request: B refreshes the same token, and saves, now
			b.Invalidate()
			if tok, err := b.Token(context.Background()); err != nil || tok.Value != "a-2" {
				t.Errorf("B: Token = %q, %v", tok.Value, err)
			}
		}
	})
	own := heldPlan0082(nil, srv.URL+"/token", nil)
	store.set(own)
	a := loaded0082(t, store, local0082(t, srv))
	b = loaded0082(t, store, local0082(t, srv))
	a.Invalidate()
	tok, err := a.Token(context.Background())
	if err != nil || tok.Value != "a-1" {
		t.Fatalf("A: Token = %q, %v; want its own rotation", tok.Value, err)
	}
	if rec, _ := store0082(store); rec == nil || rec.Refresh != "r-2" {
		t.Fatalf("the store holds %v; want B's rotation r-2, not written over by A's save", rec)
	}
	a.mu.Lock()
	a.nextResave = time.Time{}
	a.mu.Unlock()
	if _, err := a.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := a.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	if f := fieldsOf0082(a); f.StoredRefresh != "" {
		t.Errorf("A's storedRefresh %q after the store moved on; want it cleared, so the retry stops", f.StoredRefresh)
	}
	a.Invalidate()
	tok, err = a.Token(context.Background())
	if err != nil || tok.Value != "a-2" {
		t.Errorf("A's next refresh: Token = %q, %v; want B's rotation adopted", tok.Value, err)
	}
	if got := sent(); len(got) != 2 {
		t.Errorf("refresh tokens sent %q; want A's and B's, and none for the adoption", got)
	}
}

// N-2 (the second review): a read of the store that fails before the save
// does not skip the save, as the retried save never did: a persistent read
// error would otherwise lose every rotation. A known limit: with another
// sign-in written during the request and the read failing, it is overwritten.
// Red when a failed read skips the save.
func TestFailedReReadBeforeTheSaveStillSaves_0082(t *testing.T) {
	store := &bytesStore0082{}
	srv, _ := rotatingServer0082(t, func(n int, _ string) {
		if n == 1 {
			store.mu.Lock()
			store.failLoadsFrom = store.loads + 1 // every read from here on fails
			store.mu.Unlock()
		}
	})
	store.set(heldPlan0082(nil, srv.URL+"/token", nil))
	held := heldPlan0082(store, srv.URL+"/token", local0082(t, srv))
	held.Invalidate()
	if tok, err := held.Token(context.Background()); err != nil || tok.Value != "a-1" {
		t.Fatalf("Token = %q, %v", tok.Value, err)
	}
	if _, saves := store.counts(); saves != 1 {
		t.Errorf("%d saves, want the rotation saved though the re-read failed", saves)
	}
}

// N-4 (the second review, N19): the refresh after an earlier unsaved rotation
// saves at once, though the store still holds the token the first refresh
// spent: the save knows it as the store's. Red when the save's known list
// misses the token the store is known to hold.
func TestRefreshAfterAnUnsavedRotationSavesAtOnce_0082(t *testing.T) {
	for name, mk := range storeVariants0082() {
		t.Run(name, func(t *testing.T) {
			srv, _ := rotatingServer0082(t, nil)
			store, raw := mk()
			raw.set(heldPlan0082(nil, srv.URL+"/token", nil))
			held := loaded0082(t, store, local0082(t, srv))
			raw.mu.Lock()
			raw.failSaves = 1
			raw.mu.Unlock()
			held.Invalidate()
			if tok, err := held.Token(context.Background()); err != nil || tok.Value != "a-1" {
				t.Fatalf("first refresh: Token = %q, %v", tok.Value, err)
			}
			if _, saves := raw.counts(); saves != 0 {
				t.Fatalf("%d saves; want the first refused", saves)
			}
			held.Invalidate()
			if tok, err := held.Token(context.Background()); err != nil || tok.Value != "a-2" {
				t.Fatalf("second refresh: Token = %q, %v", tok.Value, err)
			}
			if err := held.Wait(context.Background()); err != nil {
				t.Fatal(err)
			}
			if rec, _ := store0082(raw); rec == nil || rec.Refresh != "r-2" {
				t.Errorf("the store holds %v; want the second rotation, saved over the first's spent token", rec)
			}
		})
	}
}

// N-4 (N29): the retried save finds that the store already holds the
// rotation, a save having written it and reported an error: the rotation is
// the record's own token and is written again, and the session notes it as
// the store's. Red when the retried save's known list misses the rotation.
func TestRetriedSaveMeetsItsOwnRotation_0082(t *testing.T) {
	srv, _ := rotatingServer0082(t, nil)
	// The store drops the access token, so that the rotation's refresh token is
	// all that tells the record from another's.
	store := &bytesStore0082{failAfterWrite: 1, drop: func(s *OAuthSession) { s.Access = "" }}
	store.set(heldPlan0082(nil, srv.URL+"/token", nil))
	held := loaded0082(t, store, local0082(t, srv))
	held.Invalidate()
	if tok, err := held.Token(context.Background()); err != nil || tok.Value != "a-1" {
		t.Fatalf("Token = %q, %v", tok.Value, err)
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
	if f := fieldsOf0082(held); f.StoredRefresh != "r-1" {
		t.Errorf("storedRefresh %q after the retry; want the rotation r-1 noted as the store's", f.StoredRefresh)
	}
}

// B-2 (the second review's RV_B3 and Q6): requests are counted across several
// retried saves while the store holds another sign-in, whose token URL is the
// in-process issuer, so that a refresh of its token would be seen: none is
// sent, and the store is not written.
func TestRetriesWhileAnotherSignInIsStoredSendNothing_0082(t *testing.T) {
	for name, mk := range storeVariants0082() {
		t.Run(name, func(t *testing.T) {
			store, raw := mk()
			var srvURL string
			srv, sent := hookServer0082(t, func() {
				other := otherChatGPT0082(time.Now().Add(-time.Minute), srvURL+"/other")
				raw.set(other)
			})
			srvURL = srv.URL
			held := heldPlan0082(store, srv.URL+"/token", local0082(t, srv))
			raw.set(heldPlan0082(nil, srv.URL+"/token", nil))
			held.Invalidate()
			if _, err := held.Token(context.Background()); err != nil {
				t.Fatal(err)
			}
			want := raw.bytes()
			for range 3 {
				held.mu.Lock()
				held.nextResave = time.Time{}
				held.mu.Unlock()
				if _, err := held.Token(context.Background()); err != nil {
					t.Fatal(err)
				}
				if err := held.Wait(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			held.Invalidate()
			if _, err := held.Token(context.Background()); err == nil {
				t.Error("a due refresh met the other sign-in without a refusal")
			}
			if got := sent(); len(got) != 1 || strings.Contains(strings.Join(got, ","), "other-refresh") {
				t.Errorf("refresh tokens sent %q; want only the first refresh's, none of the other sign-in's", got)
			}
			if !bytes.Equal(raw.bytes(), want) {
				t.Errorf("the store changed while it held the other sign-in")
			}
		})
	}
}
