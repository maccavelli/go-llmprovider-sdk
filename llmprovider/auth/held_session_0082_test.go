package auth

// 0082-PLAN Phase 1's tests in llmprovider/auth: a held session never adopts
// a stored session that is not the same sign-in (0082-MADR D1 to D5). Every
// issuer is an in-process server or a transport that fails the test.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
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

// heldStore0082 is an in-memory TokenStore whose record a test replaces, as
// another process writing the store would.
type heldStore0082 struct {
	mu    sync.Mutex
	rec   *OAuthSession
	saves int
}

func (m *heldStore0082) Load(context.Context, llmprovider.ProviderID) (*OAuthSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rec == nil {
		return nil, nil
	}
	r := m.rec
	return &OAuthSession{Provider: r.Provider, Access: r.Access, Refresh: r.Refresh, Expiry: r.Expiry,
		Issuer: r.Issuer, ClientID: r.ClientID, AccountID: r.AccountID, TokenURL: r.TokenURL,
		Resource: r.Resource, Scope: r.Scope, IDToken: r.IDToken, HostID: r.HostID, Kind: r.Kind,
		Consents: r.Consents}, nil
}

func (m *heldStore0082) Save(_ context.Context, _ llmprovider.ProviderID, s *OAuthSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saves++
	m.rec = s.snapshot()
	return nil
}

func (m *heldStore0082) Delete(context.Context, llmprovider.ProviderID) error { return nil }

func (m *heldStore0082) set(rec *OAuthSession) {
	m.mu.Lock()
	m.rec = rec
	m.mu.Unlock()
}

func (m *heldStore0082) current() *OAuthSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.rec
}

// noNetwork0082 fails the test on any request.
type noNetwork0082 struct{ t *testing.T }

func (n noNetwork0082) RoundTrip(r *http.Request) (*http.Response, error) {
	n.t.Errorf("unexpected request %s %s", r.Method, r.URL)
	return nil, errors.New("no network in this test")
}

// deadURL0082 is the token URL of every record a test writes that is not the
// held session's: a name that never resolves, so that a refresh sent with it
// reaches nothing even if a client could dial (RFC 6761 reserves .invalid).
// An empty one would mean OpenAI's real endpoint.
const deadURL0082 = "https://issuer.invalid/never"

// loopbackOnly0082 refuses, without dialing, any request whose host is not a
// loopback address, and fails the test that sent it: a refresh a planted break
// or a bug sends to another host never leaves the process (0065-MADR).
type loopbackOnly0082 struct {
	t    *testing.T
	base http.RoundTripper
}

func (l loopbackOnly0082) RoundTrip(r *http.Request) (*http.Response, error) {
	if ip := net.ParseIP(r.URL.Hostname()); ip == nil || !ip.IsLoopback() {
		l.t.Errorf("a request left for %q: %s %s", r.URL.Hostname(), r.Method, r.URL.Redacted())
		return nil, errors.New("0082 fixtures: no request off the machine")
	}
	return l.base.RoundTrip(r)
}

// local0082 is srv's client, restricted to loopback.
func local0082(t *testing.T, srv *httptest.Server) *http.Client {
	return &http.Client{Transport: loopbackOnly0082{t: t, base: srv.Client().Transport}}
}

// tokenServer0082 answers every refresh with fresh tokens and counts the
// refresh tokens it is sent.
func tokenServer0082(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var sent []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refresh := ""
		if r.Header.Get("Content-Type") == "application/json" {
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			refresh = body[oauthRefreshToken]
		} else if err := r.ParseForm(); err == nil {
			refresh = r.Form.Get(oauthRefreshToken)
		}
		mu.Lock()
		sent = append(sent, refresh)
		mu.Unlock()
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

func heldPlan0082(store TokenStore, tokenURL string, client *http.Client) *OAuthSession {
	return &OAuthSession{Provider: llmprovider.ProviderOpenAI, Kind: SessionChatGPTPlan,
		Access: "held-plan-access", Refresh: "held-plan-refresh", Expiry: time.Now().Add(time.Hour),
		Issuer: DefaultOpenAIIssuer, ClientID: "oaiapp_held", TokenURL: tokenURL,
		Resource: planResource, Scope: "openid", HostID: "host-held", Store: store, HTTPClient: client}
}

func otherChatGPT0082(expiry time.Time, tokenURL string) *OAuthSession {
	return &OAuthSession{Provider: llmprovider.ProviderOpenAI, Kind: SessionChatGPT,
		Access: "other-access", Refresh: "other-refresh", Expiry: expiry,
		Issuer: DefaultOpenAIIssuer, ClientID: "app_other", AccountID: "acct-other", TokenURL: tokenURL}
}

// refused0082 checks a refused token request: an auth failure, no token,
// and every field of the held session, its refresh bookkeeping included,
// exactly as before (before is fieldsOf0082, taken before the request).
func refused0082(t *testing.T, held *OAuthSession, before fields0082, tok llmprovider.Token, err error) {
	t.Helper()
	refusedFully0082(t, held, before, tok, err)
}

// T1 (MADR S1): another application's ChatGPT login with a current token is
// refused, with no request, and the held plan session stays itself.
func TestHeldSessionRefusesAnotherKind_0082(t *testing.T) {
	store := &heldStore0082{}
	held := heldPlan0082(store, "https://issuer.invalid/token", &http.Client{Transport: noNetwork0082{t}})
	store.set(otherChatGPT0082(time.Now().Add(time.Hour), deadURL0082))
	held.Invalidate() // as a provider does after a 401
	before := fieldsOf0082(held)
	tok, err := held.Token(context.Background())
	refused0082(t, held, before, tok, err)

	// The kind alone: a ChatGPT login of the same provider, issuer, client
	// and account as the held plan session.
	store = &heldStore0082{}
	held = heldPlan0082(store, "https://issuer.invalid/token", &http.Client{Transport: noNetwork0082{t}})
	same := otherChatGPT0082(time.Now().Add(time.Hour), deadURL0082)
	same.ClientID, same.AccountID = "oaiapp_held", ""
	store.set(same)
	held.Invalidate()
	before = fieldsOf0082(held)
	tok, err = held.Token(context.Background())
	refused0082(t, held, before, tok, err)
}

// T2 (MADR S2): with the other login's token due, its refresh token is never
// sent, and the store keeps the other entry as it was.
func TestHeldSessionNeverSpendsAnotherRefreshToken_0082(t *testing.T) {
	srv, sent := tokenServer0082(t)
	store := &heldStore0082{}
	held := heldPlan0082(store, srv.URL+"/held", local0082(t, srv))
	store.set(otherChatGPT0082(time.Now().Add(-time.Minute), srv.URL+"/other"))
	held.Invalidate()
	before := fieldsOf0082(held)
	tok, err := held.Token(context.Background())
	if got := sent(); len(got) != 0 {
		t.Errorf("refresh tokens sent %q, want none", got)
	}
	if rec := store.current(); rec.Refresh != "other-refresh" || store.saves != 0 {
		t.Errorf("store holds refresh %q after %d saves; want the other entry untouched", rec.Refresh, store.saves)
	}
	refused0082(t, held, before, tok, err)
}

// T3 (MADR S3): a Grok record in FileTokenStore's openai.json is refused by a
// held OpenAI session, which stays OpenAI's.
func TestHeldSessionRefusesAnotherProviderRecord_0082(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileTokenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(fileRecord{Provider: llmprovider.ProviderGrok, Access: "grok-access", Refresh: "grok-refresh",
		Expiry: time.Now().Add(time.Hour), Issuer: DefaultGrokOAuthIssuer, ClientID: "grok-client", Kind: SessionGrok, TokenURL: deadURL0082})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "openai.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	held := &OAuthSession{TokenURL: deadURL0082, Provider: llmprovider.ProviderOpenAI, Kind: SessionChatGPT, Access: "openai-access",
		Refresh: "openai-refresh", Expiry: time.Now().Add(time.Hour), Issuer: DefaultOpenAIIssuer,
		ClientID: "openai-client", Store: fs, HTTPClient: &http.Client{Transport: noNetwork0082{t}}}
	held.Invalidate()
	before := fieldsOf0082(held)
	tok, err := held.Token(context.Background())
	refused0082(t, held, before, tok, err)
	if owner := held.Owner(); owner != llmprovider.ProviderOpenAI {
		t.Errorf("Owner() = %q, want openai kept", owner)
	}
	if got, readErr := os.ReadFile(filepath.Join(dir, "openai.json")); readErr != nil || !bytes.Equal(got, raw) {
		t.Errorf("openai.json is %q, %v; want the Grok record byte for byte as it was", got, readErr)
	}

	// A record that names another provider and is otherwise this sign-in's:
	// the provider alone refuses it.
	store := &heldStore0082{}
	held = &OAuthSession{TokenURL: deadURL0082, Provider: llmprovider.ProviderOpenAI, Kind: SessionChatGPT, Access: "openai-access",
		Refresh: "openai-refresh", Expiry: time.Now().Add(time.Hour), Issuer: DefaultOpenAIIssuer,
		ClientID: "openai-client", Store: store, HTTPClient: &http.Client{Transport: noNetwork0082{t}}}
	store.set(&OAuthSession{TokenURL: deadURL0082, Provider: llmprovider.ProviderGrok, Kind: SessionChatGPT, Access: "other-access",
		Refresh: "other-refresh", Expiry: time.Now().Add(time.Hour), Issuer: DefaultOpenAIIssuer,
		ClientID: "openai-client"})
	held.Invalidate()
	before = fieldsOf0082(held)
	tok, err = held.Token(context.Background())
	refused0082(t, held, before, tok, err)
}

// T4: the same kind from another client is refused.
func TestHeldSessionRefusesAnotherClient_0082(t *testing.T) {
	store := &heldStore0082{}
	held := heldPlan0082(store, "https://issuer.invalid/token", &http.Client{Transport: noNetwork0082{t}})
	other := heldPlan0082(nil, "https://issuer.invalid/token", nil)
	other.ClientID, other.Access, other.Refresh = "oaiapp_other", "other-access", "other-refresh"
	store.set(other)
	held.Invalidate()
	before := fieldsOf0082(held)
	tok, err := held.Token(context.Background())
	refused0082(t, held, before, tok, err)

	// The issuer alone: the same kind and client from another issuer.
	store = &heldStore0082{}
	held = heldPlan0082(store, "https://issuer.invalid/token", &http.Client{Transport: noNetwork0082{t}})
	other = heldPlan0082(nil, "https://issuer.invalid/token", nil)
	other.Issuer, other.Access, other.Refresh = "https://issuer.invalid", "other-access", "other-refresh"
	store.set(other)
	held.Invalidate()
	before = fieldsOf0082(held)
	tok, err = held.Token(context.Background())
	refused0082(t, held, before, tok, err)
}

// T5: the same kind and client for another account is refused (Q2).
func TestHeldSessionRefusesAnotherAccount_0082(t *testing.T) {
	store := &heldStore0082{}
	held := &OAuthSession{TokenURL: deadURL0082, Provider: llmprovider.ProviderOpenAI, Kind: SessionChatGPT, Access: "held-access",
		Refresh: "held-refresh", Expiry: time.Now().Add(time.Hour), Issuer: DefaultOpenAIIssuer,
		ClientID: "app_same", AccountID: "acct-held", Store: store, HTTPClient: &http.Client{Transport: noNetwork0082{t}}}
	store.set(&OAuthSession{TokenURL: deadURL0082, Provider: llmprovider.ProviderOpenAI, Kind: SessionChatGPT, Access: "other-access",
		Refresh: "other-refresh", Expiry: time.Now().Add(time.Hour), Issuer: DefaultOpenAIIssuer,
		ClientID: "app_same", AccountID: "acct-other"})
	held.Invalidate()
	before := fieldsOf0082(held)
	tok, err := held.Token(context.Background())
	refused0082(t, held, before, tok, err)
}

// T6: the re-read after a rejected refresh (0016-MADR A1) refuses another
// sign-in the same way: another program writes it while the held session's
// own refresh is rejected as reused.
func TestReReadAfterRejectedRefreshRefusesAnotherSignIn_0082(t *testing.T) {
	store := &heldStore0082{}
	var refreshes atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		refreshes.Add(1)
		store.set(otherChatGPT0082(time.Now().Add(time.Hour), deadURL0082))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"refresh_token_reused"}`))
	}))
	defer srv.Close()
	held := heldPlan0082(store, srv.URL+"/held", local0082(t, srv))
	own := heldPlan0082(nil, srv.URL+"/held", nil) // the store holds the held session's own record
	store.set(own)
	held.Invalidate()
	before := fieldsOf0082(held)
	tok, err := held.Token(context.Background())
	if n := refreshes.Load(); n != 1 {
		t.Errorf("%d refreshes, want the held session's own, once", n)
	}
	refused0082(t, held, before, tok, err)
	// The refusal is the held session's own, not the issuer's rejection of the
	// refresh that preceded the re-read: that one is an auth failure too.
	if err == nil || !strings.Contains(err.Error(), "another sign-in") {
		t.Errorf("err = %v; want the re-read's refusal naming another sign-in", err)
	}
}

// T9 (Q3): refused while the store holds another sign-in; adopted again once
// it holds this sign-in's rotation; refreshed on its own once it holds
// nothing.
func TestHeldSessionResumesWhenItsSignInReturns_0082(t *testing.T) {
	srv, sent := tokenServer0082(t)
	store := &heldStore0082{}
	held := heldPlan0082(store, srv.URL+"/held", local0082(t, srv))
	store.set(otherChatGPT0082(time.Now().Add(time.Hour), srv.URL+"/other"))
	held.Invalidate()
	before := fieldsOf0082(held)
	tok, err := held.Token(context.Background())
	refused0082(t, held, before, tok, err)

	rotated := heldPlan0082(nil, srv.URL+"/held", nil)
	rotated.Access, rotated.Refresh = "held-plan-access-2", "held-plan-refresh-2"
	store.set(rotated)
	if tok, err = held.Token(context.Background()); err != nil || tok.Value != "held-plan-access-2" {
		t.Fatalf("with its own sign-in back: Token = %q, %v; want the rotation adopted", tok.Value, err)
	}

	store.set(nil)
	held.Invalidate()
	if tok, err = held.Token(context.Background()); err != nil || tok.Value != "fresh-access" {
		t.Fatalf("with the store empty: Token = %q, %v; want its own refresh", tok.Value, err)
	}
	if got := sent(); len(got) != 1 || got[0] != "held-plan-refresh-2" {
		t.Errorf("refresh tokens sent %q, want only the held session's own", got)
	}
}

// T10: a sibling's rotation of the same sign-in is adopted: a record with no
// Kind (an older one), one whose scope, ID token and consents changed, and
// one with no account where the held session has one. A guard: green before
// and after the change.
func TestSiblingRotationOfTheSameSignInIsAdopted_0082(t *testing.T) {
	cases := map[string]func(*OAuthSession){
		"no kind":                      func(s *OAuthSession) { s.Kind = "" },
		"scope, id token and consents": func(s *OAuthSession) { s.Scope, s.IDToken, s.Consents = "openid email", "id-2", "x=granted" },
		"no account on one side":       func(s *OAuthSession) { s.AccountID = "" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			store := &heldStore0082{}
			held := heldPlan0082(store, "https://issuer.invalid/token", &http.Client{Transport: noNetwork0082{t}})
			held.AccountID = "acct-held"
			rotated := heldPlan0082(nil, "https://issuer.invalid/token", nil)
			rotated.Access, rotated.Refresh, rotated.AccountID = "sibling-access", "sibling-refresh", "acct-held"
			change(rotated)
			store.set(rotated)
			held.Invalidate()
			tok, err := held.Token(context.Background())
			if err != nil || tok.Value != "sibling-access" {
				t.Fatalf("Token = %q, %v; want the sibling's rotation adopted", tok.Value, err)
			}
		})
	}
}

// T11 (Q6): a Grok session refuses another client's Grok record.
func TestGrokHeldSessionRefusesAnotherClient_0082(t *testing.T) {
	store := &heldStore0082{}
	held := &OAuthSession{TokenURL: deadURL0082, Provider: llmprovider.ProviderGrok, Kind: SessionGrok, Access: "grok-held-access",
		Refresh: "grok-held-refresh", Expiry: time.Now().Add(time.Hour), Issuer: DefaultGrokOAuthIssuer,
		ClientID: "grok-held", Store: store, HTTPClient: &http.Client{Transport: noNetwork0082{t}}}
	store.set(&OAuthSession{TokenURL: deadURL0082, Provider: llmprovider.ProviderGrok, Kind: SessionGrok, Access: "grok-other-access",
		Refresh: "grok-other-refresh", Expiry: time.Now().Add(time.Hour), Issuer: DefaultGrokOAuthIssuer,
		ClientID: "grok-other"})
	held.Invalidate()
	before := fieldsOf0082(held)
	tok, err := held.Token(context.Background())
	refused0082(t, held, before, tok, err)
}

// T12 (Q4): the refusal is an auth failure whose text carries no token,
// refresh token or account id.
func TestRefusalNamesNoSecret_0082(t *testing.T) {
	store := &heldStore0082{}
	held := heldPlan0082(store, "https://issuer.invalid/token", &http.Client{Transport: noNetwork0082{t}})
	held.AccountID = "acct-held"
	store.set(otherChatGPT0082(time.Now().Add(time.Hour), deadURL0082))
	held.Invalidate()
	_, err := held.Token(context.Background())
	if err == nil || !errors.Is(err, llmprovider.ErrAuthFailure) {
		t.Fatalf("err = %v; want an error matching ErrAuthFailure", err)
	}
	for _, secret := range []string{"other-access", "other-refresh", "acct-other", "held-plan-access", "held-plan-refresh", "acct-held"} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("the refusal %q carries %q", err, secret)
		}
	}
}

// T13 (the edges of 0082-MADR D2, added at Phase 1): what the store keeps for
// a record with no kind and no issuer, shared by two sessions, is the same
// sign-in on both sides: a refresh keeps both as they were, so the second
// session adopts the first's rotation. No release wrote such a record (every
// sign-in sets the issuer, and the kind has been set since 0074); a hand-built
// or imported one is the case. A guard: green before and after the change.
func TestLegacyRecordRotationIsAdopted_0082(t *testing.T) {
	srv, sent := tokenServer0082(t)
	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := json.Marshal(map[string]any{"provider": "openai", "access": "old-access", "refresh": "old-refresh",
		"expiry": time.Now().Add(-time.Minute), "issuer": "", "client_id": "app_legacy", "account_id": "",
		"token_url": srv.URL + "/token"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.Dir, "openai.json"), legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	load := func() *OAuthSession {
		s, err := store.Load(context.Background(), llmprovider.ProviderOpenAI)
		if err != nil || s == nil {
			t.Fatalf("Load = %v, %v", s, err)
		}
		s.HTTPClient = local0082(t, srv)
		return s
	}
	first, second := load(), load()
	if tok, err := first.Token(context.Background()); err != nil || tok.Value != "fresh-access" {
		t.Fatalf("first: Token = %q, %v; want its own refresh", tok.Value, err)
	}
	saved, err := store.Load(context.Background(), llmprovider.ProviderOpenAI)
	if err != nil || saved == nil || saved.Refresh != "fresh-refresh" || saved.Kind != "" || saved.Issuer != "" || saved.ClientID != "app_legacy" {
		t.Fatalf("the store holds %v, %v after the refresh; want the rotation with the record's own empty kind and issuer", saved, err)
	}
	if tok, err := second.Token(context.Background()); err != nil || tok.Value != "fresh-access" {
		t.Fatalf("second: Token = %q, %v; want the first's rotation adopted", tok.Value, err)
	}
	if got := sent(); len(got) != 1 || got[0] != "old-refresh" {
		t.Errorf("refresh tokens sent %q, want old-refresh once", got)
	}
}

// T14: the rules of D2 that are not a plain comparison. A stored session that
// names an account where the held one names none, that names no provider, or
// whose issuer has a trailing slash or none, is the same sign-in and is
// adopted; one that names no client where the held session names one is
// another.
func TestSameSignInEdges_0082(t *testing.T) {
	adopted := map[string]func(*OAuthSession, *OAuthSession){
		"no account on the held side": func(held, stored *OAuthSession) { held.AccountID, stored.AccountID = "", "acct-sibling" },
		"no provider on the record":   func(_, stored *OAuthSession) { stored.Provider = "" },
		"trailing slash on the issuer": func(_, stored *OAuthSession) {
			stored.Issuer += "/"
		},
		// An empty issuer is the kind's built-in issuer (the amendment's R-4;
		// the first pass refused it).
		"no issuer on the record": func(_, stored *OAuthSession) { stored.Issuer = "" },
	}
	for name, change := range adopted {
		t.Run("adopted/"+name, func(t *testing.T) {
			store := &heldStore0082{}
			held := heldPlan0082(store, "https://issuer.invalid/token", &http.Client{Transport: noNetwork0082{t}})
			rotated := heldPlan0082(nil, "https://issuer.invalid/token", nil)
			rotated.Access, rotated.Refresh = "sibling-access", "sibling-refresh"
			change(held, rotated)
			store.set(rotated)
			held.Invalidate()
			tok, err := held.Token(context.Background())
			if err != nil || tok.Value != "sibling-access" {
				t.Fatalf("Token = %q, %v; want the sibling's rotation adopted", tok.Value, err)
			}
		})
	}
	refused := map[string]func(*OAuthSession){
		"no client on the record": func(stored *OAuthSession) { stored.ClientID = "" },
	}
	for name, change := range refused {
		t.Run("refused/"+name, func(t *testing.T) {
			store := &heldStore0082{}
			held := heldPlan0082(store, "https://issuer.invalid/token", &http.Client{Transport: noNetwork0082{t}})
			stored := heldPlan0082(nil, "https://issuer.invalid/token", nil)
			stored.Access, stored.Refresh = "other-access", "other-refresh"
			change(stored)
			store.set(stored)
			held.Invalidate()
			before := fieldsOf0082(held)
			tok, err := held.Token(context.Background())
			refused0082(t, held, before, tok, err)
		})
	}
}

// T15 of the first pass, the stored kind made plain in the refusal's text, is
// replaced by TestRefusalNamesTheFieldNotTheValue_0082: the text no longer
// carries any value the store holds (the amendment's N1).
