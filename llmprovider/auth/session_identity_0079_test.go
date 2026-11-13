package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// The session's identity on its refresh and revocation (0079-MADR D2, D3;
// 0079-PLAN Phase 2). Requests are recorded by a client's transport, which
// answers some itself, so that no test sends a request off the machine.

const probeAgent0079 = probeName0076 + "/" + probeVersion0076 + " ("

type req0079 struct{ Method, URL, UA, Type, Body string }

// shape is the request without its host and User-Agent: what D3 says does not
// change.
func (r req0079) shape() string {
	u, err := url.Parse(r.URL)
	path := r.URL
	if err == nil {
		path = u.Path
	}
	return r.Method + " " + path + " | " + r.Type + " | " + r.Body
}

type wire0079 struct {
	mu   sync.Mutex
	reqs []req0079
}

func (w *wire0079) mark() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.reqs)
}

func (w *wire0079) since(n int) []req0079 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]req0079(nil), w.reqs[n:]...)
}

// transport0079 records each request, then answers it from answer when that
// says so, else sends it on to next.
type transport0079 struct {
	w      *wire0079
	next   http.RoundTripper
	answer func(*http.Request) (string, bool)
}

func (tr *transport0079) RoundTrip(r *http.Request) (*http.Response, error) {
	var body []byte
	if r.Body != nil {
		var err error
		if body, err = io.ReadAll(r.Body); err != nil {
			return nil, err
		}
		if err = r.Body.Close(); err != nil {
			return nil, err
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
	}
	tr.w.mu.Lock()
	tr.w.reqs = append(tr.w.reqs, req0079{r.Method, r.URL.String(), r.Header.Get("User-Agent"), r.Header.Get("Content-Type"), string(body)})
	tr.w.mu.Unlock()
	if tr.answer != nil {
		if reply, ok := tr.answer(r); ok {
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": {"application/json"}},
				Body: io.NopCloser(strings.NewReader(reply)), Request: r}, nil
		}
	}
	return tr.next.RoundTrip(r)
}

func (w *wire0079) client(next http.RoundTripper, answer func(*http.Request) (string, bool)) *http.Client {
	return &http.Client{Transport: &transport0079{w: w, next: next, answer: answer}}
}

func moduleAgent0079() string {
	sdk, _ := transport.BuildVersions()
	return "go-llmprovider-sdk/" + sdk + " (" + runtime.GOOS + "; " + runtime.GOARCH + ") go-llmprovider-sdk/" + sdk
}

// run0079 is a session signed in at a fake issuer, and the requests its
// client sent.
type run0079 struct {
	f       *fake0076
	w       *wire0079
	client  *http.Client
	session *OAuthSession
}

// signIn0079 signs in through s under id. The OpenAI revocation endpoint,
// which the fake does not serve, is answered by the client.
func signIn0079(t *testing.T, s signIn0076, id Identity) run0079 {
	t.Helper()
	f := newFake0076(t)
	w := &wire0079{}
	c := w.client(f.srv.Client().Transport, func(r *http.Request) (string, bool) {
		return "", r.Method == http.MethodPost && r.URL.Path == "/oauth/revoke"
	})
	session, err := s.run(t, f, func(o *OAuthFlowOptions) { o.Identity, o.HTTPClient = id, c },
		func(o *ChatGPTPlanOptions) { o.Identity, o.HTTPClient = id, c })
	if err != nil {
		t.Fatalf("%s: %v", s.name, err)
	}
	return run0079{f, w, c, session}
}

// refresh forces one refresh and returns the requests it sent.
func (r run0079) refresh(t *testing.T, name string) []req0079 {
	t.Helper()
	r.session.Invalidate()
	mark := r.w.mark()
	if _, err := r.session.Token(t.Context()); err != nil {
		t.Fatalf("%s: refresh: %v", name, err)
	}
	return r.w.since(mark)
}

// revoke revokes the session and returns the requests it sent.
func (r run0079) revoke(t *testing.T, name string) []req0079 {
	t.Helper()
	mark := r.w.mark()
	if err := RevokeOAuthSession(t.Context(), r.session); err != nil {
		t.Fatalf("%s: revoke: %v", name, err)
	}
	return r.w.since(mark)
}

// refreshables are the sign-ins whose sessions refresh (not Kilo's).
func refreshables0079() []signIn0076 {
	var out []signIn0076
	for _, s := range signIns0076() {
		if s.name != "Kilo device" {
			out = append(out, s)
		}
	}
	return out
}

func identityOf0079(s *OAuthSession) Identity {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.identity
}

// sameShape fails the test unless got and want are the same requests but for
// their host and User-Agent, and each of got names agent by prefix.
func sameShape0079(t *testing.T, name string, got, want []req0079, prefix string, module bool) {
	t.Helper()
	if len(got) != len(want) || len(got) == 0 {
		t.Errorf("%s: sent %d requests %q; want the %d of the unnamed session: %q", name, len(got), got, len(want), want)
		return
	}
	for i := range got {
		if got[i].shape() != want[i].shape() {
			t.Errorf("%s: request %d is %q; the unnamed session sent %q", name, i, got[i].shape(), want[i].shape())
		}
		if !strings.HasPrefix(got[i].UA, prefix) || (module && !strings.Contains(got[i].UA, "go-llmprovider-sdk/")) {
			t.Errorf("%s: %s sent User-Agent %q; want it to begin %q and name this module", name, got[i].shape(), got[i].UA, prefix)
		}
	}
}

// TestSessionIdentity_Unset_0079: with no identity, and with a Version and no
// name, the refresh of a ChatGPT (JSON), Grok (form) and plan (form) session
// and the revocation of each, with Grok's and the plan's discovery, send this
// module's User-Agent, and each URL and body is what 5d08770 sends.
func TestSessionIdentity_Unset_0079(t *testing.T) {
	agent := moduleAgent0079()
	const discoveryBody = `{"authorization_endpoint":"https://auth.openai.com/oauth/authorize","token_endpoint":"https://auth.openai.com/oauth/token",` +
		`"revocation_endpoint":"https://auth.openai.com/oauth/revoke","jwks_uri":"https://auth.openai.com/keys"}`
	grokDiscovery := strings.NewReplacer("auth.openai.com/oauth/revoke", "auth.x.ai/oauth2/revoke", "auth.openai.com", "auth.x.ai").Replace(discoveryBody)
	answer := func(r *http.Request) (string, bool) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			if r.URL.Host == "auth.x.ai" {
				return grokDiscovery, true
			}
			return discoveryBody, true
		case "/oauth/token", "/oauth2/token":
			return `{"access_token":"a2","refresh_token":"r2","expires_in":3600}`, true
		case "/oauth/revoke", "/oauth2/revoke":
			return `{}`, true
		}
		return "", false
	}
	const form = "client_id=cid&grant_type=refresh_token&refresh_token=r1"
	// The refresh rotated the token to r2, which the revocation sends.
	const revokeForm = "client_id=cid&token=r2&token_type_hint=refresh_token"
	const urlEncoded = "application/x-www-form-urlencoded"
	cases := []struct {
		name    string
		build   func(*http.Client) *OAuthSession
		refresh []req0079
		revoke  []req0079
	}{
		{"ChatGPT", func(c *http.Client) *OAuthSession {
			return &OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "a1", Refresh: "r1", Expiry: time.Now().Add(-time.Minute),
				Issuer: DefaultOpenAIIssuer, ClientID: "cid", HTTPClient: c}
		},
			[]req0079{{"POST", "https://auth.openai.com/oauth/token", agent, "application/json", `{"client_id":"cid","grant_type":"refresh_token","refresh_token":"r1"}`}},
			[]req0079{{"POST", "https://auth.openai.com/oauth/revoke", agent, "application/json", `{"client_id":"cid","token":"r2","token_type_hint":"refresh_token"}`}}},
		{"Grok", func(c *http.Client) *OAuthSession {
			return &OAuthSession{Provider: llmprovider.ProviderGrok, Access: "a1", Refresh: "r1", Expiry: time.Now().Add(-time.Minute),
				Issuer: DefaultGrokOAuthIssuer, ClientID: "cid", HTTPClient: c}
		},
			[]req0079{{"POST", "https://auth.x.ai/oauth2/token", agent, urlEncoded, form}},
			[]req0079{{"GET", "https://auth.x.ai/.well-known/openid-configuration", agent, "", ""},
				{"POST", "https://auth.x.ai/oauth2/revoke", agent, urlEncoded, revokeForm}}},
		{"plan", func(c *http.Client) *OAuthSession {
			return &OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "a1", Refresh: "r1", Expiry: time.Now().Add(-time.Minute),
				Issuer: DefaultOpenAIIssuer, ClientID: "cid", Resource: "res1", TokenURL: "https://auth.openai.com/oauth/token", HTTPClient: c}
		},
			[]req0079{{"POST", "https://auth.openai.com/oauth/token", agent, urlEncoded, form + "&resource=res1"}},
			[]req0079{{"GET", "https://auth.openai.com/.well-known/openid-configuration", agent, "", ""},
				{"POST", "https://auth.openai.com/oauth/revoke", agent, urlEncoded, revokeForm}}},
	}
	for _, tc := range cases {
		w := &wire0079{}
		session := tc.build(w.client(nil, answer))
		if _, err := session.Token(t.Context()); err != nil {
			t.Fatalf("%s: refresh: %v", tc.name, err)
		}
		if got := w.since(0); fmt.Sprint(got) != fmt.Sprint(tc.refresh) {
			t.Errorf("%s: refresh sent %q; want %q", tc.name, got, tc.refresh)
		}
		mark := w.mark()
		if err := RevokeOAuthSession(t.Context(), session); err != nil {
			t.Fatalf("%s: revoke: %v", tc.name, err)
		}
		if got := w.since(mark); fmt.Sprint(got) != fmt.Sprint(tc.revoke) {
			t.Errorf("%s: revocation sent %q; want %q", tc.name, got, tc.revoke)
		}
	}

	// A Version and no name names this module, and sends what an unnamed
	// sign-in's session sends.
	for _, s := range refreshables0079() {
		plain, versioned := signIn0079(t, s, Identity{}), signIn0079(t, s, Identity{Version: "9.9.9"})
		sameShape0079(t, s.name+" refresh", versioned.refresh(t, s.name), plain.refresh(t, s.name), agent, false)
		sameShape0079(t, s.name+" revocation", versioned.revoke(t, s.name), plain.revoke(t, s.name), agent, false)
	}
}

// TestSessionIdentity_Refresh_0079: a session from each sign-in that signed
// in under an Identity refreshes under it, and sends the unnamed session's
// URL and body.
func TestSessionIdentity_Refresh_0079(t *testing.T) {
	probe := Identity{Name: probeName0076, Version: probeVersion0076}
	for _, s := range refreshables0079() {
		named, plain := signIn0079(t, s, probe), signIn0079(t, s, Identity{})
		sameShape0079(t, s.name, named.refresh(t, s.name), plain.refresh(t, s.name), probeAgent0079, true)
	}
}

// TestSessionIdentity_Revoke_0079: the same sessions' revocations, and for
// Grok and the plan the discovery before them, carry the name.
func TestSessionIdentity_Revoke_0079(t *testing.T) {
	probe := Identity{Name: probeName0076, Version: probeVersion0076}
	for _, s := range refreshables0079() {
		named, plain := signIn0079(t, s, probe), signIn0079(t, s, Identity{})
		got := named.revoke(t, s.name)
		if want := map[string]int{"OpenAI browser": 1, "OpenAI device": 1, "Grok browser": 2, "Grok device": 2, "ChatGPT plan": 2}[s.name]; len(got) != want {
			t.Errorf("%s: revocation sent %d requests %q; want %d", s.name, len(got), got, want)
		}
		sameShape0079(t, s.name, got, plain.revoke(t, s.name), probeAgent0079, true)
	}
}

// TestUseIdentity_0079: a session with none takes the first non-zero
// identity; a second does not replace it; a zero one changes nothing; a
// sign-in's is kept; and it is safe beside a refresh.
func TestUseIdentity_0079(t *testing.T) {
	first, second := Identity{Name: "first-app", Version: "1"}, Identity{Name: "second-app", Version: "2"}
	s := &OAuthSession{}
	steps := []struct{ give, want Identity }{
		{Identity{}, Identity{}}, {first, first}, {second, first}, {Identity{}, first},
	}
	for i, step := range steps {
		s.UseIdentity(step.give)
		if got := identityOf0079(s); got != step.want {
			t.Fatalf("step %d: UseIdentity(%+v) left %+v; want %+v", i, step.give, got, step.want)
		}
	}

	signed := signIn0079(t, signIns0076()[3], first) // Grok device
	signed.session.UseIdentity(second)
	if got := identityOf0079(signed.session); got != first {
		t.Errorf("a sign-in's identity became %+v; want %+v kept", got, first)
	}

	w := &wire0079{}
	c := w.client(nil, func(*http.Request) (string, bool) {
		return `{"access_token":"a2","refresh_token":"r2","expires_in":3600}`, true
	})
	racing := &OAuthSession{Provider: llmprovider.ProviderGrok, Access: "a1", Refresh: "r1", Issuer: DefaultGrokOAuthIssuer, ClientID: "cid", HTTPClient: c}
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			racing.UseIdentity(Identity{Name: fmt.Sprintf("app-%d", i)})
		}()
		go func() {
			defer wg.Done()
			racing.Invalidate()
			if _, err := racing.Token(context.Background()); err != nil {
				t.Errorf("Token: %v", err)
			}
		}()
	}
	wg.Wait()
	if got := identityOf0079(racing); !strings.HasPrefix(got.Name, "app-") {
		t.Errorf("identity after the race = %+v; want one of the callers'", got)
	}
}

// TestSessionIdentity_NotStored_0079: a named session saved by FileTokenStore
// writes the same bytes as the same session unnamed; loaded, its refresh
// names this module; its formatters equal the unnamed session's.
func TestSessionIdentity_NotStored_0079(t *testing.T) {
	probe := Identity{Name: probeName0076, Version: probeVersion0076}
	grokDevice := signIns0076()[3]
	named, plain := signIn0079(t, grokDevice, probe), signIn0079(t, grokDevice, Identity{})
	stores := [2]*FileTokenStore{}
	var saved [2][]byte
	for i, r := range []run0079{named, plain} {
		store, err := NewFileTokenStore(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Save(t.Context(), llmprovider.ProviderGrok, r.session); err != nil {
			t.Fatal(err)
		}
		if saved[i], err = os.ReadFile(store.path(llmprovider.ProviderGrok)); err != nil {
			t.Fatal(err)
		}
		stores[i] = store
	}
	// Each fake issuer has a port of its own.
	saved[0], saved[1] = bytes.ReplaceAll(saved[0], []byte(named.f.srv.URL), []byte("ISSUER")), bytes.ReplaceAll(saved[1], []byte(plain.f.srv.URL), []byte("ISSUER"))
	if !bytes.Equal(saved[0], saved[1]) {
		t.Errorf("the named session's record is %s; the unnamed one's is %s", saved[0], saved[1])
	}

	loaded, err := stores[0].Load(t.Context(), llmprovider.ProviderGrok)
	if err != nil {
		t.Fatal(err)
	}
	w := &wire0079{}
	loaded.HTTPClient = w.client(named.f.srv.Client().Transport, nil)
	loaded.Invalidate()
	if _, err := loaded.Token(t.Context()); err != nil {
		t.Fatalf("refresh of the loaded session: %v", err)
	}
	for _, r := range w.since(0) {
		if r.UA != moduleAgent0079() {
			t.Errorf("the loaded session's %s sent User-Agent %q; want this module's only", r.shape(), r.UA)
		}
	}

	forms := func(r run0079) [4]string {
		raw, err := json.Marshal(r.session)
		if err != nil {
			t.Fatal(err)
		}
		out := [4]string{r.session.String(), fmt.Sprintf("%#v", r.session), string(raw), r.session.LogValue().String()}
		for i := range out {
			out[i] = strings.ReplaceAll(out[i], r.f.srv.URL, "ISSUER")
		}
		return out
	}
	if a, b := forms(named), forms(plain); a != b {
		t.Errorf("the named session formats as %q; the unnamed one as %q", a, b)
	}
}

// TestSessionIdentity_Kept_0079: after a refresh, and after a rejected refresh
// adopts a sibling's rotation from the store (0016-MADR A1), the session's
// next refresh still names the application.
func TestSessionIdentity_Kept_0079(t *testing.T) {
	probe := Identity{Name: probeName0076, Version: probeVersion0076}
	signed := signIn0079(t, signIns0076()[3], probe)
	for i := range 2 {
		for _, r := range signed.refresh(t, "Grok device") {
			if !strings.HasPrefix(r.UA, probeAgent0079) {
				t.Errorf("refresh %d sent User-Agent %q; want the probe's first", i+1, r.UA)
			}
		}
	}

	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var agents, refreshes []string
	var tokenURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode refresh: %v", err)
		}
		mu.Lock()
		agents, refreshes = append(agents, r.Header.Get("User-Agent")), append(refreshes, body[oauthRefreshToken])
		mu.Unlock()
		if body[oauthRefreshToken] != "rt-old" {
			refreshOK(w, "a-refreshed")
			return
		}
		sibling := testSession("a-sibling", "rt-sibling", time.Now().Add(time.Hour))
		sibling.TokenURL = tokenURL
		if err := store.Save(r.Context(), llmprovider.ProviderOpenAI, sibling); err != nil {
			t.Errorf("sibling save: %v", err)
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"refresh_token_reused"}`))
	}))
	t.Cleanup(srv.Close)
	tokenURL = srv.URL
	old := testSession("a-old", "rt-old", time.Now().Add(-time.Minute))
	old.TokenURL = tokenURL
	if err := store.Save(t.Context(), llmprovider.ProviderOpenAI, old); err != nil {
		t.Fatal(err)
	}
	session, err := store.Load(t.Context(), llmprovider.ProviderOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	session.UseIdentity(probe)
	if tok, err := session.Token(t.Context()); err != nil || tok.Value != "a-sibling" {
		t.Fatalf("Token = %q, %v; want the sibling's session adopted", tok.Value, err)
	}
	session.Invalidate()
	if _, err := session.Token(t.Context()); err != nil {
		t.Fatalf("refresh after the adoption: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(agents) != 2 || refreshes[1] != "rt-sibling" {
		t.Fatalf("refreshes %q sent; want the rejected one and one with the sibling's token", refreshes)
	}
	for i, agent := range agents {
		if !strings.HasPrefix(agent, probeAgent0079) {
			t.Errorf("refresh %d (%s) sent User-Agent %q; want the probe's first", i+1, refreshes[i], agent)
		}
	}
}
