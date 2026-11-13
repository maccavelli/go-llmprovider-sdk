package auth

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// The consumer's identity on its sign-ins (0076-MADR D3; 0076-PLAN Phase 2).
// Each sign-in runs against its own fake issuer, which records every
// request's method, path and User-Agent.

const (
	probeName0076    = "probe-app"
	probeVersion0076 = "1.2.3"
	// planClient0076 is the client the plan's callback names; the other
	// browser sign-ins ignore it.
	planClient0076 = "oaiapp_issued0076"
	discovery0076  = "GET /.well-known/openid-configuration"
	keys0076       = "GET /keys"
)

// fake0076 is a fake issuer for every sign-in: OpenAI's and Grok's browser
// and device sign-ins, Kilo's, and the ChatGPT plan's, and Grok's and the
// plan's revocation.
type fake0076 struct {
	srv *httptest.Server
	ti  *testIssuer
	// keys is the path of the issuer's keys, new for each fake, so the keys'
	// process-wide cache has not seen its URL and fetches it (MADR C38).
	keys string

	mu         sync.Mutex
	seen       []string // "METHOD /path | User-Agent", one per request
	authorize  string   // the authorize URL the browser was sent to
	deviceForm url.Values
}

func newFake0076(t *testing.T) *fake0076 {
	t.Helper()
	f := &fake0076{keys: "/keys-0076-" + rand.Text()}
	mux := http.NewServeMux()
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.seen = append(f.seen, r.Method+" "+r.URL.Path+" | "+r.Header.Get("User-Agent"))
		f.mu.Unlock()
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(f.srv.Close)
	base := f.srv.URL
	f.ti = newTestIssuer(mux, base)
	mux.HandleFunc("GET "+f.keys, f.ti.serveJWKS)
	mux.HandleFunc(discovery0076, func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, f.ti.discovery(t, map[string]string{
			"authorization_endpoint":        base + "/authorize",
			"token_endpoint":                base + "/token",
			"device_authorization_endpoint": base + "/device",
			"revocation_endpoint":           base + "/revoke",
			"jwks_uri":                      base + f.keys,
		}))
	})
	token := func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse token form: %v", err)
		}
		var body map[string]any
		if err := json.Unmarshal([]byte(f.ti.tokenResponse(t, "RS256", r.PostForm.Get("client_id"), "access-0076", "refresh-0076")), &body); err != nil {
			t.Fatal(err)
		}
		body["scope"] = planScopes // the plan's grant; the other sign-ins ignore it
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		writeTestJSON(t, w, string(raw))
	}
	mux.HandleFunc("POST /token", token)
	mux.HandleFunc("POST /oauth/token", token) // OpenAI's, which discovery does not name
	mux.HandleFunc("POST /device", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse device form: %v", err)
		}
		f.mu.Lock()
		f.deviceForm = r.PostForm
		f.mu.Unlock()
		writeTestJSON(t, w, grokCode0048)
	})
	mux.HandleFunc("POST /api/accounts/deviceauth/usercode", answer0048(t, http.StatusOK, openAICode0048))
	mux.HandleFunc("POST "+openAIPoll0048, answer0048(t, http.StatusOK, `{"authorization_code":"code-0076","code_verifier":"verifier-0076"}`))
	mux.HandleFunc(kiloCodes0048, answer0048(t, http.StatusOK, kiloCode0048))
	mux.HandleFunc(kiloPoll0048, answer0048(t, http.StatusOK, `{"token":"kilo-token-0076"}`))
	mux.HandleFunc("POST /revoke", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	return f
}

// visit plays the browser: it records the authorize URL and follows it to
// the loopback callback, with a code and the plan's issued client.
func (f *fake0076) visit(rawURL string) error {
	f.mu.Lock()
	f.authorize = rawURL
	f.mu.Unlock()
	f.ti.captureNonce(rawURL)
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	q := u.Query()
	back := url.Values{"code": {"code-0076"}, "state": {q.Get("state")}, "client_id": {planClient0076}}
	resp, err := http.Get(q.Get("redirect_uri") + "?" + back.Encode())
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.Body.Close()
}

// flowOptions are the options of a sign-in at f, with a stepped clock, as set
// changes them.
func (f *fake0076) flowOptions(set func(*OAuthFlowOptions)) OAuthFlowOptions {
	clock := newFakeOAuthClock()
	opts := OAuthFlowOptions{HTTPClient: f.srv.Client(), OpenURL: f.visit, ClientID: "test-client", Issuer: f.srv.URL,
		now: clock.now, sleep: clock.sleep}
	if set != nil {
		set(&opts)
	}
	return opts
}

// requests are the requests f received from mark on, "METHOD /path |
// User-Agent", with the keys' path as /keys.
func (f *fake0076) requests(mark int) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.seen)-mark)
	for _, line := range f.seen[mark:] {
		out = append(out, strings.Replace(line, f.keys, "/keys", 1))
	}
	return out
}

func (f *fake0076) received() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.seen)
}

// loopbackPorts0076 gives OpenAI's browser sign-ins ports no other package
// can be given while they are free (0045-MADR D2).
func loopbackPorts0076(t *testing.T) {
	original := openaiLoopbackPorts
	openaiLoopbackPorts = []int{nettest.Port(t), nettest.Port(t)}
	t.Cleanup(func() { openaiLoopbackPorts = original })
}

// signIn0076 is one sign-in: the requests it sends, "METHOD /path", in
// order, and how it runs at a fake with the options set and setPlan change.
type signIn0076 struct {
	name     string
	requests []string
	run      func(t *testing.T, f *fake0076, set func(*OAuthFlowOptions), setPlan func(*ChatGPTPlanOptions)) (*OAuthSession, error)
}

func signIns0076() []signIn0076 {
	browser := func(provider llmprovider.ProviderID) func(*testing.T, *fake0076, func(*OAuthFlowOptions), func(*ChatGPTPlanOptions)) (*OAuthSession, error) {
		return func(t *testing.T, f *fake0076, set func(*OAuthFlowOptions), _ func(*ChatGPTPlanOptions)) (*OAuthSession, error) {
			loopbackPorts0076(t)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			return LoginBrowserOAuth(ctx, provider, f.flowOptions(set))
		}
	}
	device := func(provider llmprovider.ProviderID) func(*testing.T, *fake0076, func(*OAuthFlowOptions), func(*ChatGPTPlanOptions)) (*OAuthSession, error) {
		return func(t *testing.T, f *fake0076, set func(*OAuthFlowOptions), _ func(*ChatGPTPlanOptions)) (*OAuthSession, error) {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			login, err := StartDeviceOAuth(ctx, provider, f.flowOptions(set))
			if err != nil {
				return nil, err
			}
			return login.Wait(ctx)
		}
	}
	plan := func(t *testing.T, f *fake0076, _ func(*OAuthFlowOptions), setPlan func(*ChatGPTPlanOptions)) (*OAuthSession, error) {
		loopbackPorts0076(t)
		opts := ChatGPTPlanOptions{AgentName: planTestAgent, HTTPClient: f.srv.Client(), OpenURL: f.visit, issuer: f.srv.URL}
		if setPlan != nil {
			setPlan(&opts)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		return LoginChatGPTPlan(ctx, opts)
	}
	return []signIn0076{
		{"OpenAI browser", []string{discovery0076, "POST /oauth/token", keys0076}, browser(llmprovider.ProviderOpenAI)},
		{"Grok browser", []string{discovery0076, "POST /token", keys0076}, browser(llmprovider.ProviderGrok)},
		{"OpenAI device", []string{discovery0076, "POST /api/accounts/deviceauth/usercode", "POST " + openAIPoll0048,
			"POST /oauth/token", keys0076}, device(llmprovider.ProviderOpenAI)},
		{"Grok device", []string{discovery0076, "POST /device", "POST /token", keys0076}, device(llmprovider.ProviderGrok)},
		{"Kilo device", []string{"POST /api/device-auth/codes", "GET " + kiloPollPath0048}, device(llmprovider.ProviderKilo)},
		{"ChatGPT plan", []string{discovery0076, "POST /token", keys0076}, plan},
	}
}

// withAgent0076 is each of requests followed by " | " and agent.
func withAgent0076(requests []string, agent string) []string {
	out := make([]string, len(requests))
	for i, r := range requests {
		out[i] = r + " | " + agent
	}
	return out
}

var loopbackPort0076 = regexp.MustCompile(`127\.0\.0\.1:\d+`)

// normalized0076 is an authorize URL with what changes from run to run
// replaced: the fake issuer's URL, the loopback port, the state, the PKCE
// challenge and the nonce.
func normalized0076(t *testing.T, rawURL, issuer string) string {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	for _, key := range []string{"state", "code_challenge", "nonce"} {
		if q.Has(key) {
			q.Set(key, strings.ToUpper(key))
		}
	}
	q.Set("redirect_uri", loopbackPort0076.ReplaceAllString(q.Get("redirect_uri"), "127.0.0.1:PORT"))
	u.RawQuery = q.Encode()
	return strings.Replace(u.String(), issuer, "ISSUER", 1)
}

// The literals TestIdentityUnset_Unchanged_0076 pins, as 773b84c sends them.
const (
	openAIAuthorize0076 = "ISSUER/oauth/authorize?client_id=test-client&code_challenge=CODE_CHALLENGE&code_challenge_method=S256" +
		"&codex_cli_simplified_flow=true&id_token_add_organizations=true&originator=go-llmprovider-sdk" +
		"&redirect_uri=http%3A%2F%2F127.0.0.1%3APORT%2Fauth%2Fcallback&response_type=code" +
		"&scope=openid+profile+email+offline_access+api.connectors.read+api.connectors.invoke&state=STATE"
	grokAuthorize0076 = "ISSUER/authorize?client_id=test-client&code_challenge=CODE_CHALLENGE&code_challenge_method=S256" +
		"&nonce=NONCE&redirect_uri=http%3A%2F%2F127.0.0.1%3APORT%2Fcallback&referrer=go-llmprovider-sdk&response_type=code" +
		"&scope=openid+profile+email+offline_access+grok-cli%3Aaccess+api%3Aaccess&state=STATE"
	grokDeviceForm0076 = "client_id=test-client&referrer=go-llmprovider-sdk" +
		"&scope=openid+profile+email+offline_access+grok-cli%3Aaccess+api%3Aaccess"
)

// TestIdentityUnset_Unchanged_0076: with no Identity, OpenAI's and Grok's
// authorize URLs, Grok's device form and the User-Agent of every request of
// every sign-in are as 773b84c sends them: this module, at its version, on
// this platform (0076-MADR D3, "byte for byte").
func TestIdentityUnset_Unchanged_0076(t *testing.T) {
	sdk, _ := transport.BuildVersions()
	agent := "go-llmprovider-sdk/" + sdk + " (" + runtime.GOOS + "; " + runtime.GOARCH + ") go-llmprovider-sdk/" + sdk
	for _, s := range signIns0076() {
		f := newFake0076(t)
		if _, err := s.run(t, f, nil, nil); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		if got, want := f.requests(0), withAgent0076(s.requests, agent); !slices.Equal(got, want) {
			t.Errorf("%s: requests %q; want %q", s.name, got, want)
		}
		f.mu.Lock()
		authorize, form := f.authorize, f.deviceForm
		f.mu.Unlock()
		switch s.name {
		case "OpenAI browser":
			if got := normalized0076(t, authorize, f.srv.URL); got != openAIAuthorize0076 {
				t.Errorf("%s: authorize URL %s; want %s", s.name, got, openAIAuthorize0076)
			}
		case "Grok browser":
			if got := normalized0076(t, authorize, f.srv.URL); got != grokAuthorize0076 {
				t.Errorf("%s: authorize URL %s; want %s", s.name, got, grokAuthorize0076)
			}
		case "Grok device":
			if got := form.Encode(); got != grokDeviceForm0076 {
				t.Errorf("%s: device form %s; want %s", s.name, got, grokDeviceForm0076)
			}
		}
	}
}

// TestIdentity_Originator_0076: an Identity's Name is OpenAI's originator
// (0076-MADR D3).
func TestIdentity_Originator_0076(t *testing.T) {
	probe := func(o *OAuthFlowOptions) { o.Identity = Identity{Name: probeName0076, Version: probeVersion0076} }
	f := newFake0076(t)
	if _, err := signIns0076()[0].run(t, f, probe, nil); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	authorize := f.authorize
	f.mu.Unlock()
	u, err := url.Parse(authorize)
	if err != nil {
		t.Fatal(err)
	}
	if got := u.Query()["originator"]; !slices.Equal(got, []string{probeName0076}) {
		t.Errorf("originator = %q; want [%q]", got, probeName0076)
	}
}

// TestIdentity_Referrer_0076: an Identity's Name is Grok's referrer, on the
// authorize URL and on the device form (0076-MADR D3).
func TestIdentity_Referrer_0076(t *testing.T) {
	probe := func(o *OAuthFlowOptions) { o.Identity = Identity{Name: probeName0076, Version: probeVersion0076} }
	for _, s := range signIns0076() {
		if s.name != "Grok browser" && s.name != "Grok device" {
			continue
		}
		f := newFake0076(t)
		if _, err := s.run(t, f, probe, nil); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		f.mu.Lock()
		authorize, form := f.authorize, f.deviceForm
		f.mu.Unlock()
		got := form["referrer"]
		if s.name == "Grok browser" {
			u, err := url.Parse(authorize)
			if err != nil {
				t.Fatal(err)
			}
			got = u.Query()["referrer"]
		}
		if !slices.Equal(got, []string{probeName0076}) {
			t.Errorf("%s: referrer = %q; want [%q]", s.name, got, probeName0076)
		}
	}
}

// TestIdentity_UserAgent_0076: with an Identity, every request of every
// sign-in names the application first and this module after it: discovery,
// the token exchanges, the OpenAI and Grok device requests, Kilo's, the
// plan's and the fetch of the issuer's keys. A revocation of the signed-in
// session, its discovery included, names the application first and this
// module after it, and a session built with no identity names this module
// only (0079-MADR D3, which rewrote 0076-MADR D3's "this module only").
func TestIdentity_UserAgent_0076(t *testing.T) {
	probe := Identity{Name: probeName0076, Version: probeVersion0076}
	for _, s := range signIns0076() {
		f := newFake0076(t)
		session, err := s.run(t, f, func(o *OAuthFlowOptions) { o.Identity = probe }, func(o *ChatGPTPlanOptions) { o.Identity = probe })
		if err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		got := f.requests(0)
		if len(got) != len(s.requests) {
			t.Errorf("%s: requests %q; want %q", s.name, got, s.requests)
		}
		for i, line := range got {
			request, agent, _ := strings.Cut(line, " | ")
			if i < len(s.requests) && request != s.requests[i] {
				t.Errorf("%s: request %d is %q; want %q", s.name, i, request, s.requests[i])
			}
			if !strings.HasPrefix(agent, probeName0076+"/"+probeVersion0076+" (") || !strings.Contains(agent, "go-llmprovider-sdk/") {
				t.Errorf("%s: %s sent User-Agent %q; want the probe's first, then this module", s.name, request, agent)
			}
		}
		if s.name != "Grok browser" && s.name != "ChatGPT plan" {
			continue
		}
		mark := f.received()
		if err := RevokeOAuthSession(t.Context(), session); err != nil {
			t.Fatalf("%s: revoke: %v", s.name, err)
		}
		revoked := f.requests(mark)
		if len(revoked) != 2 {
			t.Errorf("%s: revocation sent %q; want its discovery and the revocation", s.name, revoked)
		}
		for _, line := range revoked {
			request, agent, _ := strings.Cut(line, " | ")
			if !strings.HasPrefix(agent, probeName0076+"/"+probeVersion0076+" (") || !strings.Contains(agent, "go-llmprovider-sdk/") {
				t.Errorf("%s: revocation's %s sent User-Agent %q; want the probe's first, then this module", s.name, request, agent)
			}
		}

		// A session built from the same fields, never signed in, has no
		// identity: its revocation names this module only.
		built := &OAuthSession{Provider: session.Provider, Access: session.Access, Refresh: session.Refresh,
			Issuer: session.Issuer, ClientID: session.ClientID, TokenURL: session.TokenURL, Resource: session.Resource,
			Kind: session.Kind, HTTPClient: session.HTTPClient}
		mark = f.received()
		if err := RevokeOAuthSession(t.Context(), built); err != nil {
			t.Fatalf("%s: revoke a session with no identity: %v", s.name, err)
		}
		unnamed := f.requests(mark)
		if len(unnamed) != 2 {
			t.Errorf("%s: revocation with no identity sent %q; want its discovery and the revocation", s.name, unnamed)
		}
		for _, line := range unnamed {
			request, agent, _ := strings.Cut(line, " | ")
			if !strings.HasPrefix(agent, "go-llmprovider-sdk/") || strings.Contains(agent, probeName0076) {
				t.Errorf("%s: revocation's %s with no identity sent User-Agent %q; want this module's only", s.name, request, agent)
			}
		}
	}
}
