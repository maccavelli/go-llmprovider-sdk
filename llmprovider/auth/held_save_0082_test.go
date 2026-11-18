package auth

// 0082-PLAN Phase 1's second pass (the amendment of 2026-11-17, R-1 to R-4 and
// the review's findings): the refresh's own save is guarded, a record of
// another sign-in is refused with or without a refresh token, the consent
// read and the empty-issuer rule follow the same comparison, and a refusal
// leaves every field of the held session, its store and its text alone. Every
// issuer is an in-process server or a transport that fails the test.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// bytesStore0082 is an in-memory TokenStore whose record is kept as the JSON
// bytes FileTokenStore would write, so that a test can assert it is unchanged.
// It counts its loads and saves. It is not a RefreshLocker; lockedStore0082 is.
type bytesStore0082 struct {
	mu    sync.Mutex
	rec   []byte
	saves int
	loads int
	// drop, when set, edits the copy Save writes: a store that loses fields.
	drop func(*OAuthSession)
	// failSaves fails that many Saves before one writes; failAfterWrite makes
	// the next Save write and then report an error; failLoads fails that many
	// Loads, counted from loadsAfter more (a read that fails after a refresh).
	failSaves      int
	failAfterWrite int
	failLoadsFrom  int
}

func recBytes0082(s *OAuthSession) []byte {
	if s == nil {
		return nil
	}
	b, err := json.Marshal(fileRecord{Provider: s.Provider, Access: s.Access, Refresh: s.Refresh, Expiry: s.Expiry,
		Issuer: s.Issuer, ClientID: s.ClientID, AccountID: s.AccountID, FedRAMP: s.FedRAMP, TokenURL: s.TokenURL,
		Resource: s.Resource, Scope: s.Scope, IDToken: s.IDToken, HostID: s.HostID, Kind: s.Kind, Consents: s.Consents})
	if err != nil {
		panic(err)
	}
	return b
}

func (m *bytesStore0082) Load(context.Context, llmprovider.ProviderID) (*OAuthSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loads++
	if m.failLoadsFrom > 0 && m.loads >= m.failLoadsFrom {
		return nil, errors.New("bytesStore0082: read failed")
	}
	if m.rec == nil {
		return nil, nil
	}
	var r fileRecord
	if err := json.Unmarshal(m.rec, &r); err != nil {
		return nil, err
	}
	return &OAuthSession{Provider: r.Provider, Access: r.Access, Refresh: r.Refresh, Expiry: r.Expiry,
		Issuer: r.Issuer, ClientID: r.ClientID, AccountID: r.AccountID, FedRAMP: r.FedRAMP, TokenURL: r.TokenURL,
		Resource: r.Resource, Scope: r.Scope, IDToken: r.IDToken, HostID: r.HostID, Kind: r.Kind,
		Consents: r.Consents}, nil
}

func (m *bytesStore0082) Save(_ context.Context, _ llmprovider.ProviderID, s *OAuthSession) error {
	snap := s.snapshot()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failSaves > 0 {
		m.failSaves--
		return errors.New("bytesStore0082: save refused")
	}
	if m.drop != nil {
		m.drop(snap)
	}
	m.saves++
	m.rec = recBytes0082(snap)
	if m.failAfterWrite > 0 {
		m.failAfterWrite--
		return errors.New("bytesStore0082: wrote, then reported an error")
	}
	return nil
}

func (m *bytesStore0082) Delete(context.Context, llmprovider.ProviderID) error {
	m.mu.Lock()
	m.rec = nil
	m.mu.Unlock()
	return nil
}

// set replaces the record as another process writing the store would, with no
// lock and no count.
func (m *bytesStore0082) set(s *OAuthSession) {
	m.mu.Lock()
	m.rec = recBytes0082(s)
	m.mu.Unlock()
}

func (m *bytesStore0082) bytes() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]byte(nil), m.rec...)
}

func (m *bytesStore0082) counts() (loads, saves int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.loads, m.saves
}

// lockedStore0082 is a bytesStore0082 that is also a RefreshLocker.
type lockedStore0082 struct {
	*bytesStore0082
	lock sync.Mutex
}

func (m *lockedStore0082) LockRefresh(context.Context, llmprovider.ProviderID) (func(), error) {
	m.lock.Lock()
	return m.lock.Unlock, nil
}

// storeVariants0082 are the two stores every guard is run against.
func storeVariants0082() map[string]func() (TokenStore, *bytesStore0082) {
	return map[string]func() (TokenStore, *bytesStore0082){
		"a RefreshLocker": func() (TokenStore, *bytesStore0082) {
			s := &lockedStore0082{bytesStore0082: &bytesStore0082{}}
			return s, s.bytesStore0082
		},
		"no RefreshLocker": func() (TokenStore, *bytesStore0082) {
			s := &bytesStore0082{}
			return s, s
		},
	}
}

// fields0082 is every field of a held session that a refusal must leave alone,
// including the unexported bookkeeping of its refresh.
type fields0082 struct {
	Provider                                             llmprovider.ProviderID
	Access, Refresh, Issuer, ClientID, AccountID         string
	TokenURL, Resource, Scope, IDToken, HostID, Consents string
	Kind                                                 SessionKind
	Expiry                                               time.Time
	StoredRefresh                                        string
	NextRefresh, NextResave                              time.Time
	Store                                                TokenStore
}

func fieldsOf0082(s *OAuthSession) fields0082 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fields0082{s.Provider, s.Access, s.Refresh, s.Issuer, s.ClientID, s.AccountID, s.TokenURL, s.Resource,
		s.Scope, s.IDToken, s.HostID, s.Consents, s.Kind, s.Expiry, s.storedRefresh, s.nextRefresh, s.nextResave, s.Store}
}

// hookServer0082 is an issuer that rotates on every refresh, records each
// refresh token it is sent, and runs onFirst before it answers the first.
func hookServer0082(t *testing.T, onFirst func()) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var sent []string
	var once sync.Once
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
		mu.Unlock()
		if onFirst != nil {
			once.Do(onFirst)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"fresh-access","refresh_token":"fresh-refresh","expires_in":3600}`))
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), sent...)
	}
}

func parseForm0082(raw []byte) (string, error) {
	r := &http.Request{Method: http.MethodPost, Header: http.Header{"Content-Type": {"application/x-www-form-urlencoded"}},
		Body: io.NopCloser(bytes.NewReader(raw))}
	if err := r.ParseForm(); err != nil {
		return "", err
	}
	return r.PostForm.Get(oauthRefreshToken), nil
}

func chatGPTHeld0082(store TokenStore, tokenURL string, client *http.Client) *OAuthSession {
	return &OAuthSession{Provider: llmprovider.ProviderOpenAI, Kind: SessionChatGPT,
		Access: "held-access", Refresh: "held-refresh", Expiry: time.Now().Add(time.Hour),
		Issuer: DefaultOpenAIIssuer, ClientID: "app_same", AccountID: "acct-held", TokenURL: tokenURL,
		Store: store, HTTPClient: client}
}

// refusedFully0082 checks a refused token request against the held session as
// it was: an auth failure, no token, and every field, bookkeeping included,
// unchanged.
func refusedFully0082(t *testing.T, held *OAuthSession, before fields0082, tok llmprovider.Token, err error) {
	t.Helper()
	if !errors.Is(err, llmprovider.ErrAuthFailure) || tok.Value != "" {
		t.Fatalf("Token = %q, %v; want no token and an error matching ErrAuthFailure", tok.Value, err)
	}
	if after := fieldsOf0082(held); after != before {
		t.Errorf("the refusal changed the held session:\n before %+v\n after  %+v", before, after)
	}
}

// R-4 (amendment of 2026-11-17): an empty issuer is the kind's built-in issuer
// on both sides, as refreshTokenURL, ChatGPT and inferKind read it; an empty
// client is never the same as a named one; a different issuer, and the same
// host in another case, are other issuers.
func TestEmptyIssuerIsTheBuiltInIssuer_0082(t *testing.T) {
	type build func() *OAuthSession
	chatgpt := func() *OAuthSession { return chatGPTHeld0082(nil, "https://issuer.invalid/token", nil) }
	grok := func() *OAuthSession {
		return &OAuthSession{Provider: llmprovider.ProviderGrok, Kind: SessionGrok, Access: "held-access",
			Refresh: "held-refresh", Expiry: time.Now().Add(time.Hour), Issuer: DefaultGrokOAuthIssuer,
			ClientID: "grok-client", TokenURL: "https://issuer.invalid/token"}
	}
	plan := func() *OAuthSession { return heldPlan0082(nil, "https://issuer.invalid/token", nil) }
	cases := []struct {
		name   string
		held   build
		change func(held, stored *OAuthSession)
		adopt  bool
	}{
		{"adopted/held ChatGPT with no issuer, record with the default", chatgpt, func(h, _ *OAuthSession) { h.Issuer = "" }, true},
		{"adopted/held with the default, record with no issuer", chatgpt, func(_, s *OAuthSession) { s.Issuer = "" }, true},
		{"adopted/both with no issuer", chatgpt, func(h, s *OAuthSession) { h.Issuer, s.Issuer = "", "" }, true},
		{"adopted/held Grok with no issuer, record with the default", grok, func(h, _ *OAuthSession) { h.Issuer = "" }, true},
		{"adopted/held plan session with no issuer, record with the default", plan, func(h, _ *OAuthSession) { h.Issuer = "" }, true},
		{"refused/held with no issuer, record from another issuer", chatgpt, func(h, s *OAuthSession) { h.Issuer, s.Issuer = "", "https://other.example" }, false},
		{"refused/the same host in another case", chatgpt, func(_, s *OAuthSession) { s.Issuer = "https://AUTH.openai.com" }, false},
		{"refused/no client on the record", chatgpt, func(_, s *OAuthSession) { s.ClientID = "" }, false},
		{"refused/no client on the held side", chatgpt, func(h, _ *OAuthSession) { h.ClientID = "" }, false},
		{"refused/held Grok with neither issuer nor client, record with both", grok, func(h, _ *OAuthSession) { h.Issuer, h.ClientID = "", "" }, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store := &bytesStore0082{}
			h := c.held()
			h.Store, h.HTTPClient = store, &http.Client{Transport: noNetwork0082{t}}
			stored := c.held()
			stored.Access, stored.Refresh = "sibling-access", "sibling-refresh"
			c.change(h, stored)
			store.set(stored)
			h.Invalidate()
			before := fieldsOf0082(h)
			tok, err := h.Token(context.Background())
			if c.adopt {
				if err != nil || tok.Value != "sibling-access" {
					t.Fatalf("Token = %q, %v; want the sibling's rotation adopted", tok.Value, err)
				}
				return
			}
			refusedFully0082(t, h, before, tok, err)
		})
	}
}

// R-2: a stored record with no refresh token is judged like any other. One of
// another sign-in, a pasted ChatGPT access token under a held plan session, is
// refused, with no request, and the store is not written over.
func TestAccessOnlyRecordOfAnotherSignInIsRefused_0082(t *testing.T) {
	pasted := func() *OAuthSession {
		return &OAuthSession{TokenURL: deadURL0082, Provider: llmprovider.ProviderOpenAI, Kind: SessionChatGPT, Access: "pasted-access",
			Issuer: DefaultOpenAIIssuer, ClientID: DefaultOpenAIClientID, AccountID: "acct-pasted",
			Expiry: time.Time{}}
	}
	for name, mk := range storeVariants0082() {
		t.Run(name, func(t *testing.T) {
			store, raw := mk()
			held := heldPlan0082(store, "https://issuer.invalid/token", &http.Client{Transport: noNetwork0082{t}})
			raw.set(pasted())
			want := raw.bytes()
			held.Invalidate()
			before := fieldsOf0082(held)
			tok, err := held.Token(context.Background())
			refusedFully0082(t, held, before, tok, err)
			if _, saves := raw.counts(); saves != 0 || !bytes.Equal(raw.bytes(), want) {
				t.Errorf("the access-only record was written over (%d saves)", saves)
			}
		})
	}
	t.Run("FileTokenStore", func(t *testing.T) {
		dir := t.TempDir()
		fs, err := NewFileTokenStore(dir)
		if err != nil {
			t.Fatal(err)
		}
		if err := fs.Save(context.Background(), llmprovider.ProviderOpenAI, pasted()); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "openai.json")
		want, _ := os.ReadFile(path)
		held := heldPlan0082(fs, "https://issuer.invalid/token", &http.Client{Transport: noNetwork0082{t}})
		held.Invalidate()
		before := fieldsOf0082(held)
		tok, err := held.Token(context.Background())
		refusedFully0082(t, held, before, tok, err)
		if got, _ := os.ReadFile(path); !bytes.Equal(got, want) {
			t.Errorf("openai.json changed:\n%s\nwant\n%s", got, want)
		}
	})
}

// R-2, what happens to an access-only record of this very sign-in: it is not
// adopted (adoption needs a refresh token), so the held session refreshes its
// own refresh token and saves over it. A guard: green before and after.
func TestAccessOnlyRecordOfTheSameSignInIsRefreshedOver_0082(t *testing.T) {
	for name, mk := range storeVariants0082() {
		t.Run(name, func(t *testing.T) {
			srv, sent := hookServer0082(t, nil)
			store, raw := mk()
			held := chatGPTHeld0082(store, srv.URL+"/token", local0082(t, srv))
			same := chatGPTHeld0082(nil, deadURL0082, nil)
			same.Access, same.Refresh, same.Expiry = "pasted-access", "", time.Time{}
			raw.set(same)
			held.Invalidate()
			tok, err := held.Token(context.Background())
			if err != nil || tok.Value != "fresh-access" {
				t.Fatalf("Token = %q, %v; want the held session's own refresh", tok.Value, err)
			}
			if got := sent(); len(got) != 1 || got[0] != "held-refresh" {
				t.Errorf("refresh tokens sent %q, want only the held session's own", got)
			}
			if _, saves := raw.counts(); saves != 1 {
				t.Errorf("%d saves, want the rotation saved once over the access-only record", saves)
			}
			if rec, _ := store.Load(context.Background(), llmprovider.ProviderOpenAI); rec == nil || rec.Refresh != "fresh-refresh" {
				t.Errorf("the store holds %v; want the rotation", rec)
			}
		})
	}
}

// A record that holds the held session's own refresh token is the session's
// whatever its other fields say: a store that drops fields still refreshes the
// session as before. A guard: green before and after.
func TestRecordWithTheHeldRefreshTokenIsTheHeldSessions_0082(t *testing.T) {
	srv, sent := hookServer0082(t, nil)
	store := &bytesStore0082{}
	held := heldPlan0082(store, srv.URL+"/token", local0082(t, srv))
	store.set(&OAuthSession{TokenURL: deadURL0082, Access: "stale-access", Refresh: "held-plan-refresh", Expiry: time.Now().Add(-time.Minute)})
	held.Invalidate()
	tok, err := held.Token(context.Background())
	if err != nil || tok.Value != "fresh-access" {
		t.Fatalf("Token = %q, %v; want its own refresh", tok.Value, err)
	}
	if got := sent(); len(got) != 1 || got[0] != "held-plan-refresh" {
		t.Errorf("refresh tokens sent %q, want the held session's own", got)
	}
	if _, saves := store.counts(); saves != 1 {
		t.Errorf("%d saves, want the rotation saved", saves)
	}
}

// skippedSave0082 is the state after a refresh whose save found another
// sign-in written to the store during the refresh's own request: the rotation
// is held in memory, unsaved, and the store holds the other sign-in.
type skippedSave0082 struct {
	t    *testing.T
	held *OAuthSession
	raw  *bytesStore0082
	sent func() []string
	want []byte // the other sign-in's record, as the store holds it
}

func newSkippedSave0082(t *testing.T, mk func() (TokenStore, *bytesStore0082)) *skippedSave0082 {
	t.Helper()
	newer := heldPlan0082(nil, deadURL0082, nil)
	newer.ClientID, newer.Access, newer.Refresh = "oaiapp_newclient", "newer-access", "newer-refresh"
	store, raw := mk()
	srv, sent := hookServer0082(t, func() { raw.set(newer) })
	held := heldPlan0082(store, srv.URL+"/token", local0082(t, srv))
	raw.set(heldPlan0082(nil, srv.URL+"/token", nil)) // the store holds the held session's own record
	held.Invalidate()
	tok, err := held.Token(context.Background())
	if err != nil || tok.Value != "fresh-access" {
		t.Fatalf("own refresh: Token = %q, %v; want the rotation, used in memory", tok.Value, err)
	}
	want := recBytes0082(newer)
	if got := raw.bytes(); !bytes.Equal(got, want) {
		t.Fatalf("the newer sign-in was overwritten by the refresh's save:\n%s\nwant\n%s", got, want)
	}
	if _, saves := raw.counts(); saves != 0 {
		t.Errorf("%d saves, want none: the store had moved on", saves)
	}
	if f := fieldsOf0082(held); f.Refresh != "fresh-refresh" || f.StoredRefresh != "held-plan-refresh" {
		t.Errorf("held refresh %q, storedRefresh %q; want the rotation held and the spent token still known as the store's", f.Refresh, f.StoredRefresh)
	}
	return &skippedSave0082{t: t, held: held, raw: raw, sent: sent, want: want}
}

// retry runs the retried save: the token is current and the rotation unsaved.
func (s *skippedSave0082) retry() {
	s.t.Helper()
	s.held.mu.Lock()
	s.held.nextResave = time.Time{}
	s.held.mu.Unlock()
	if tok, err := s.held.Token(context.Background()); err != nil || tok.Value != "fresh-access" {
		s.t.Fatalf("retried save: Token = %q, %v", tok.Value, err)
	}
	if err := s.held.Wait(context.Background()); err != nil {
		s.t.Fatal(err)
	}
}

// storeRefresh is the refresh token the store holds, "" for none.
func (s *skippedSave0082) storeRefresh() string {
	rec, _ := store0082(s.raw)
	if rec == nil {
		return ""
	}
	return rec.Refresh
}

// R-1 (the review's S3): another sign-in written to the store while the held
// session's own refresh request is in flight is not written over by the
// refresh's save. The session keeps the rotation in memory, unsaved, and its
// next due refresh meets the guard; the store is checked again under the same
// predicate as the retried save.
func TestSignInWrittenDuringRefreshIsNotOverwritten_0082(t *testing.T) {
	for name, mk := range storeVariants0082() {
		t.Run(name+"/the other sign-in stays", func(t *testing.T) {
			s := newSkippedSave0082(t, mk)
			s.retry()
			if !bytes.Equal(s.raw.bytes(), s.want) {
				t.Errorf("the retried save wrote over the other sign-in")
			}
			s.held.Invalidate()
			before := fieldsOf0082(s.held)
			tok, err := s.held.Token(context.Background())
			refusedFully0082(t, s.held, before, tok, err)
			if got := s.sent(); len(got) != 1 {
				t.Errorf("refresh tokens sent %q, want only the first refresh", got)
			}
			if !bytes.Equal(s.raw.bytes(), s.want) {
				t.Errorf("the store changed after the refusal")
			}
		})
		t.Run(name+"/the store returns to the spent token", func(t *testing.T) {
			s := newSkippedSave0082(t, mk)
			// A retry while the other sign-in is still there leaves the spent
			// token known as the store's, so that it is not adopted back below.
			s.retry()
			if f := fieldsOf0082(s.held); f.StoredRefresh != "held-plan-refresh" || !bytes.Equal(s.raw.bytes(), s.want) {
				t.Fatalf("after a retry against the other sign-in: storedRefresh %q, store changed %v; want the spent token still known and the store untouched",
					f.StoredRefresh, !bytes.Equal(s.raw.bytes(), s.want))
			}
			s.raw.set(heldPlan0082(nil, deadURL0082, nil)) // the held session's old record, whose token the refresh spent
			s.retry()
			if got := s.storeRefresh(); got != "fresh-refresh" {
				t.Errorf("the store holds refresh %q; want the rotation saved over the spent token it was restored to", got)
			}
			if got := s.sent(); len(got) != 1 {
				t.Errorf("refresh tokens sent %q: the spent token must never be adopted and sent again", got)
			}
		})
		t.Run(name+"/the store is emptied", func(t *testing.T) {
			s := newSkippedSave0082(t, mk)
			s.raw.set(nil)
			s.retry()
			if got := s.storeRefresh(); got != "fresh-refresh" {
				t.Errorf("the store holds refresh %q; want the rotation saved into the empty store", got)
			}
		})
	}
}

func store0082(raw *bytesStore0082) (*OAuthSession, error) {
	return raw.Load(context.Background(), llmprovider.ProviderOpenAI)
}

// R-1, in one process under -race: sessions refreshing against a FileTokenStore
// while another writer saves another sign-in the way a sign-in's save does,
// under the store's refresh lock. Whatever the interleaving, the writer's
// sign-in is what the store ends with, no request carries its tokens, and each
// Token is the session's own or an auth failure.
func TestSignInSaveRacingRefreshes_0082(t *testing.T) {
	srv, sent := hookServer0082(t, nil)
	fs, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	own := heldPlan0082(nil, srv.URL+"/token", nil)
	if err := fs.Save(ctx, llmprovider.ProviderOpenAI, own); err != nil {
		t.Fatal(err)
	}
	var sessions []*OAuthSession
	for range 3 {
		s := heldPlan0082(fs, srv.URL+"/token", local0082(t, srv))
		sessions = append(sessions, s)
	}
	newer := heldPlan0082(nil, srv.URL+"/token", nil)
	newer.ClientID, newer.Access, newer.Refresh = "oaiapp_newclient", "newer-access", "newer-refresh"

	// started counts the sessions that have begun, so that the writer saves
	// while they run.
	var started atomic.Int32
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for started.Load() < 3 {
			time.Sleep(100 * time.Microsecond)
		}
		unlock, err := fs.LockRefresh(ctx, llmprovider.ProviderOpenAI)
		if err != nil {
			t.Errorf("LockRefresh: %v", err)
			return
		}
		defer unlock()
		if err := fs.Save(ctx, llmprovider.ProviderOpenAI, newer); err != nil {
			t.Errorf("Save: %v", err)
		}
	}()
	var wg sync.WaitGroup
	for _, s := range sessions {
		wg.Add(1)
		go func() {
			defer wg.Done()
			started.Add(1)
			for range 60 {
				s.Invalidate()
				tok, err := s.Token(ctx)
				if err != nil && !errors.Is(err, llmprovider.ErrAuthFailure) {
					t.Errorf("Token err %v", err)
				}
				if strings.Contains(tok.Value, "newer") {
					t.Errorf("Token returned the newer sign-in's %q", tok.Value)
				}
				if s.SessionKind() != SessionChatGPTPlan {
					t.Errorf("kind changed to %q", s.SessionKind())
				}
			}
		}()
	}
	wg.Wait()
	<-writerDone
	for _, s := range sessions {
		_ = s.Wait(ctx)
	}
	rec, err := fs.Load(ctx, llmprovider.ProviderOpenAI)
	if err != nil || rec == nil || rec.Refresh != "newer-refresh" || rec.ClientID != "oaiapp_newclient" {
		t.Errorf("the store ends with %v, %v; want the sign-in the writer saved", rec, err)
	}
	for _, refresh := range sent() {
		if refresh == "newer-refresh" {
			t.Errorf("a refresh request carried the newer sign-in's refresh token")
		}
	}
}

// R-3 (the review's S4): the consent read judges the stored record by the same
// comparison. Another application's plan record, and another kind's, count as
// unanswered; this sign-in's record answers; a record that holds the session's
// own refresh token answers whatever its other fields say.
func TestStoredConsentIgnoresAnotherSignIn_0082(t *testing.T) {
	const id = "chatgpt_plan.background_use"
	granted := withConsent("", id, true)
	mk := func(change func(s *OAuthSession)) *OAuthSession {
		s := heldPlan0082(nil, "https://issuer.invalid/token", nil)
		s.AccountID, s.Consents = "acct-held", granted
		// Another record carries its own access token: one that carried the
		// session's would be the session's own (R-5), not another sign-in's.
		s.Access = "record-access"
		change(s)
		return s
	}
	cases := []struct {
		name         string
		stored       *OAuthSession
		wantAnswered bool
	}{
		{"another application's plan record", mk(func(s *OAuthSession) { s.ClientID, s.Refresh = "oaiapp_otherapp", "o-r" }), false},
		{"another account's record", mk(func(s *OAuthSession) { s.AccountID, s.Refresh = "acct-other", "o-r" }), false},
		{"another kind's record", mk(func(s *OAuthSession) {
			s.Kind, s.Resource, s.ClientID, s.Refresh = SessionChatGPT, "", DefaultOpenAIClientID, "o-r"
		}), false},
		{"another provider's record", mk(func(s *OAuthSession) { s.Provider, s.Refresh = llmprovider.ProviderGrok, "o-r" }), false},
		{"a record of this sign-in", mk(func(s *OAuthSession) { s.Refresh = "sibling-refresh" }), true},
		{"a record with the held refresh token and its labels dropped", &OAuthSession{TokenURL: deadURL0082, Refresh: "held-plan-refresh", Consents: granted}, true},
		{"a record with the held access token and its other fields foreign (R-5)", &OAuthSession{TokenURL: deadURL0082, Access: "held-plan-access", ClientID: "oaiapp_other", Consents: granted}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store := &bytesStore0082{}
			held := heldPlan0082(store, "https://issuer.invalid/token", nil)
			held.AccountID = "acct-held"
			store.set(c.stored)
			gotGranted, gotAnswered, err := held.StoredConsent(context.Background(), id)
			if err != nil || gotAnswered != c.wantAnswered || gotGranted != c.wantAnswered {
				t.Errorf("StoredConsent = %v, %v, %v; want granted and answered both %v", gotGranted, gotAnswered, err, c.wantAnswered)
			}
		})
	}
}

// N1: the refusal names the one field that differs, by its name, and no value
// the store holds: nothing the store holds reaches its text, control characters
// included.
func TestRefusalNamesTheFieldNotTheValue_0082(t *testing.T) {
	cases := []struct {
		field  string
		change func(s *OAuthSession)
		secret string
	}{
		{"provider", func(s *OAuthSession) { s.Provider = "prov-\x1b[2Jsecret" }, "prov-"},
		{"kind", func(s *OAuthSession) { s.Kind = "kind-\nsecret" }, "kind-"},
		{"issuer", func(s *OAuthSession) { s.Issuer = "https://issuer-secret.example" }, "issuer-secret"},
		{"client", func(s *OAuthSession) { s.ClientID = "client-secret-id" }, "client-secret"},
		{"account", func(s *OAuthSession) { s.AccountID = "acct-secret" }, "acct-secret"},
	}
	for _, c := range cases {
		t.Run(c.field, func(t *testing.T) {
			store := &bytesStore0082{}
			held := chatGPTHeld0082(store, "https://issuer.invalid/token", &http.Client{Transport: noNetwork0082{t}})
			stored := chatGPTHeld0082(nil, deadURL0082, nil)
			stored.Access, stored.Refresh = "other-access", "other-refresh"
			c.change(stored)
			store.set(stored)
			held.Invalidate()
			_, err := held.Token(context.Background())
			if !errors.Is(err, llmprovider.ErrAuthFailure) {
				t.Fatalf("err = %v; want an error matching ErrAuthFailure", err)
			}
			text := err.Error()
			if !strings.Contains(text, "(its "+c.field+" differs)") {
				t.Errorf("the refusal %q does not name the %s as the field that differs", text, c.field)
			}
			for _, bad := range []string{c.secret, "other-access", "other-refresh", "acct-held", "held-access", "held-refresh", "app_same"} {
				if strings.Contains(text, bad) {
					t.Errorf("the refusal %q carries %q", text, bad)
				}
			}
			if strings.ContainsAny(text, "\x1b\n") {
				t.Errorf("the refusal %q carries a control character", text)
			}
		})
	}
}

// S1 (the review): three token requests against one record of another sign-in
// (a ChatGPT login's current token under a held plan session): each is refused
// afresh, re-reading the store, and each leaves every field of the held
// session, its bookkeeping included, as it was; the store is not written;
// nothing is sent; no value of the other record reaches the text.
func TestRefusalLeavesEverythingAlone_0082(t *testing.T) {
	store := &bytesStore0082{}
	held := heldPlan0082(store, "https://issuer.invalid/token", &http.Client{Transport: noNetwork0082{t}})
	store.set(otherChatGPT0082(time.Now().Add(time.Hour), deadURL0082))
	want := store.bytes()
	held.Invalidate()
	before := fieldsOf0082(held)
	for i := range 3 {
		loadsBefore, _ := store.counts()
		tok, err := held.Token(context.Background())
		refusedFully0082(t, held, before, tok, err)
		if loadsAfter, _ := store.counts(); loadsAfter == loadsBefore {
			t.Errorf("call %d did not re-read the store", i)
		}
		for _, bad := range []string{"app_other", "acct-other", "other-access", "other-refresh"} {
			if strings.Contains(err.Error(), bad) {
				t.Errorf("call %d: the refusal %q carries %q", i, err, bad)
			}
		}
	}
	if _, saves := store.counts(); saves != 0 || !bytes.Equal(store.bytes(), want) {
		t.Errorf("the store was written (%d saves)", saves)
	}
}

// B1 (the review): an early refresh, with the held token still valid for more
// than 30 s, that meets another sign-in fails; it does not fall back to the
// held token, as it does for a failure that is not an auth failure.
func TestEarlyRefreshRefusalDoesNotFallBack_0082(t *testing.T) {
	store := &bytesStore0082{}
	held := heldPlan0082(store, "https://issuer.invalid/token", &http.Client{Transport: noNetwork0082{t}})
	held.Expiry = time.Now().Add(2 * time.Minute) // due within the skew, more than 30 s left
	store.set(otherChatGPT0082(time.Now().Add(time.Hour), deadURL0082))
	before := fieldsOf0082(held)
	for range 2 {
		tok, err := held.Token(context.Background())
		refusedFully0082(t, held, before, tok, err)
	}
}

// N3 (D2): a record that names no provider takes the held session's, for the
// kind it infers as for the provider it is compared on: a hand-built or
// imported record with no provider, no kind and no issuer is the same ChatGPT
// sign-in.
func TestRecordWithNoProviderTakesTheHeldProvider_0082(t *testing.T) {
	store := &bytesStore0082{}
	held := chatGPTHeld0082(store, "https://issuer.invalid/token", &http.Client{Transport: noNetwork0082{t}})
	store.set(&OAuthSession{TokenURL: deadURL0082, Access: "sibling-access", Refresh: "sibling-refresh", Expiry: time.Now().Add(time.Hour),
		ClientID: "app_same"})
	held.Invalidate()
	tok, err := held.Token(context.Background())
	if err != nil || tok.Value != "sibling-access" {
		t.Fatalf("Token = %q, %v; want the record, which names no provider, kind or issuer, adopted", tok.Value, err)
	}
}

// N3 (D2): a held session that names no provider, a caller-built one, takes the
// provider a record names for the kind it infers: an older OpenAI record with
// no kind and no issuer is the held ChatGPT sign-in's, whose Kind is set.
func TestRecordNamingTheProviderInfersItsKind_0082(t *testing.T) {
	store := &bytesStore0082{}
	held := chatGPTHeld0082(store, "https://issuer.invalid/token", &http.Client{Transport: noNetwork0082{t}})
	held.Provider = ""
	store.set(&OAuthSession{TokenURL: deadURL0082, Provider: llmprovider.ProviderOpenAI, Access: "sibling-access", Refresh: "sibling-refresh",
		Expiry: time.Now().Add(time.Hour), ClientID: "app_same"})
	held.Invalidate()
	tok, err := held.Token(context.Background())
	if err != nil || tok.Value != "sibling-access" {
		t.Fatalf("Token = %q, %v; want the OpenAI record, whose kind is inferred from its own provider, adopted", tok.Value, err)
	}
}

// N3 (D1): both reads compare with the held session's state, even after the
// first read took up a due rotation of this sign-in. The held session names no
// account, so a re-read record of an account the first read's record did not
// name does not differ from it by account: it is adopted.
func TestReReadComparesWithTheHeldSession_0082(t *testing.T) {
	store := &bytesStore0082{}
	var once sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		once.Do(func() {
			r := chatGPTHeld0082(nil, deadURL0082, nil)
			r.Access, r.Refresh, r.AccountID = "reread-access", "reread-refresh", "acct-B"
			store.set(r)
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"refresh_token_reused"}`))
	}))
	defer srv.Close()
	held := chatGPTHeld0082(store, srv.URL+"/token", local0082(t, srv))
	held.AccountID = ""
	first := chatGPTHeld0082(nil, srv.URL+"/token", nil)
	first.Access, first.Refresh, first.AccountID, first.Expiry = "first-access", "first-refresh", "acct-A", time.Now().Add(-time.Minute)
	store.set(first)
	held.Invalidate()
	tok, err := held.Token(context.Background())
	if err != nil || tok.Value != "reread-access" {
		t.Fatalf("Token = %q, %v; want the re-read record adopted, compared with the held session's own fields", tok.Value, err)
	}
}
