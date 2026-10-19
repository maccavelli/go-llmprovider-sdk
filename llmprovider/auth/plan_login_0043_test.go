package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// LoginChatGPTPlan against a fake issuer (0043-MADR D4, A2; 0043-PLAN
// Phase 2).

const planTestAgent = "Gobble"

// planLogin is a fake plan issuer and the browser that signs in at it.
type planLogin struct {
	t   *testing.T
	srv *httptest.Server
	ti  *testIssuer

	mu        sync.Mutex
	authorize url.Values // the authorize request's query
	tokenForm url.Values // the token request's form
	tokens    atomic.Int32
	opened    atomic.Int32

	// callbackClient is the client_id the browser's callback carries; "-"
	// sends none.
	callbackClient string
	// tokenClient signs the ID token for this audience; empty is the
	// exchange's client.
	tokenClient string
	// scope is the token response's scope.
	scope string
	// discovery overrides discovery fields.
	discovery map[string]string
}

func newPlanLogin(t *testing.T) *planLogin {
	t.Helper()
	// Ports no other package can be given while they are free (0045-MADR D2).
	originalPorts := openaiLoopbackPorts
	openaiLoopbackPorts = []int{nettest.Port(t), nettest.Port(t)}
	t.Cleanup(func() { openaiLoopbackPorts = originalPorts })

	pl := &planLogin{t: t, callbackClient: "oaiapp_issued0043", scope: planScopes}
	mux := http.NewServeMux()
	pl.srv = httptest.NewServer(mux)
	t.Cleanup(pl.srv.Close)
	pl.ti = newTestIssuer(mux, pl.srv.URL)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		fields := map[string]string{
			"authorization_endpoint": pl.srv.URL + "/api/accounts/authorize",
			"token_endpoint":         pl.srv.URL + "/api/accounts/oauth/token",
			"revocation_endpoint":    pl.srv.URL + "/api/accounts/oauth/revoke",
		}
		for k, v := range pl.discovery {
			fields[k] = v
		}
		writeTestJSON(t, w, pl.ti.discovery(t, fields))
	})
	mux.HandleFunc("/api/accounts/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		pl.tokens.Add(1)
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse token form: %v", err)
		}
		pl.mu.Lock()
		pl.tokenForm = r.PostForm
		pl.mu.Unlock()
		audience := pl.tokenClient
		if audience == "" {
			audience = r.PostForm.Get("client_id")
		}
		var body map[string]any
		if err := json.Unmarshal([]byte(pl.ti.tokenResponse(t, "RS256", audience, "plan-access", "plan-refresh")), &body); err != nil {
			t.Fatal(err)
		}
		if pl.scope != "" {
			body["scope"] = pl.scope
		}
		raw, _ := json.Marshal(body)
		writeTestJSON(t, w, string(raw))
	})
	return pl
}

// callbackURL is the redirect the issuer sends the browser to for the
// authorize URL raw.
func (pl *planLogin) callbackURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	q := u.Query()
	pl.mu.Lock()
	pl.authorize = q
	pl.mu.Unlock()
	pl.ti.captureNonce(raw)
	back := url.Values{"code": {"plan-code"}, "state": {q.Get("state")}}
	if pl.callbackClient != "-" {
		back.Set("client_id", pl.callbackClient)
	}
	return q.Get("redirect_uri") + "?" + back.Encode(), nil
}

// openURL plays the browser: it follows the authorize URL to the callback.
func (pl *planLogin) openURL(raw string) error {
	pl.opened.Add(1)
	callback, err := pl.callbackURL(raw)
	if err != nil {
		return err
	}
	resp, err := http.Get(callback)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.Body.Close()
}

func (pl *planLogin) options() ChatGPTPlanOptions {
	return ChatGPTPlanOptions{AgentName: planTestAgent, HTTPClient: pl.srv.Client(), OpenURL: pl.openURL, issuer: pl.srv.URL}
}

var hostIDPattern = regexp.MustCompile(`^urn:uuid:[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestLoginChatGPTPlan_FirstSignIn_0043(t *testing.T) {
	pl := newPlanLogin(t)
	session, err := LoginChatGPTPlan(context.Background(), pl.options())
	if err != nil {
		t.Fatal(err)
	}
	q := pl.authorize
	redirect := q.Get("redirect_uri")
	if !regexp.MustCompile(`^http://127\.0\.0\.1:\d+/auth/callback$`).MatchString(redirect) {
		t.Errorf("redirect_uri = %q, want http://127.0.0.1:<port>/auth/callback", redirect)
	}
	want := map[string]string{
		"response_type": "code", "client_id": planDynamicClientID, "agent_name_hint": planTestAgent,
		"scope":    "openid profile email offline_access resource.invoke chatgpt.tokens.use.direct",
		"resource": "https://api.openai.com/v1", "code_challenge_method": "S256",
	}
	for k, v := range want {
		if got := q.Get(k); got != v {
			t.Errorf("authorize %s = %q, want %q", k, got, v)
		}
	}
	for _, k := range []string{"code_challenge", "state", "nonce"} {
		if q.Get(k) == "" {
			t.Errorf("authorize has no %s", k)
		}
	}
	if q.Has("id_token_hint") || q.Has("originator") || q.Has("codex_cli_simplified_flow") {
		t.Errorf("authorize carries another flow's parameter: %v", q)
	}
	if !hostIDPattern.MatchString(q.Get("ext_agent_host_id")) {
		t.Errorf("ext_agent_host_id = %q, want a version 4 urn:uuid", q.Get("ext_agent_host_id"))
	}
	f := pl.tokenForm
	wantForm := url.Values{"grant_type": {"authorization_code"}, "code": {"plan-code"}, "redirect_uri": {redirect},
		"client_id": {"oaiapp_issued0043"}, "code_verifier": {f.Get("code_verifier")}, "resource": {"https://api.openai.com/v1"}}
	if f.Get("code_verifier") == "" || fmt.Sprint(f) != fmt.Sprint(wantForm) {
		t.Errorf("token form = %v, want %v", f, wantForm)
	}
	if !session.PlanUsage() || session.ChatGPT() || session.ClientID != "oaiapp_issued0043" || session.Provider != llmprovider.ProviderOpenAI ||
		session.Issuer != pl.srv.URL || session.TokenURL != pl.srv.URL+"/api/accounts/oauth/token" ||
		session.Resource != planResource || session.Scope != planScopes || session.IDToken == "" ||
		session.HostID != q.Get("ext_agent_host_id") || session.Access != "plan-access" || session.Refresh != "plan-refresh" {
		t.Errorf("session = %s", session)
	}
	if err := ValidateOAuthSession(session); err != nil {
		t.Errorf("the session does not validate: %v", err)
	}
}

func TestLoginChatGPTPlan_ReSignIn_0043(t *testing.T) {
	pl := newPlanLogin(t)
	pl.callbackClient = "oaiapp_prev"
	opts := pl.options()
	opts.Previous = &OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: pl.srv.URL, ClientID: "oaiapp_prev",
		Resource: planResource, IDToken: "previous-id-token", HostID: planTestHostID}
	session, err := LoginChatGPTPlan(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	q := pl.authorize
	if q.Get("client_id") != "oaiapp_prev" || q.Get("id_token_hint") != "previous-id-token" ||
		q.Get("ext_agent_host_id") != planTestHostID || q.Has("agent_name_hint") {
		t.Errorf("re-sign-in authorize = %v; want the issued client, the hint and the host id, without the app's name", q)
	}
	if pl.tokenForm.Get("client_id") != "oaiapp_prev" || session.ClientID != "oaiapp_prev" || session.HostID != planTestHostID {
		t.Errorf("token client %q, session client %q, host %q", pl.tokenForm.Get("client_id"), session.ClientID, session.HostID)
	}
}

// TestLoginChatGPTPlan_ReSignInWithoutCallbackClient_0043: a re-sign-in's
// callback need not repeat the client.
func TestLoginChatGPTPlan_ReSignInWithoutCallbackClient_0043(t *testing.T) {
	pl := newPlanLogin(t)
	pl.callbackClient = "-"
	opts := pl.options()
	opts.Previous = &OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: pl.srv.URL, ClientID: "oaiapp_prev", Resource: planResource}
	session, err := LoginChatGPTPlan(context.Background(), opts)
	if err != nil || session.ClientID != "oaiapp_prev" {
		t.Fatalf("session %v, err %v", session, err)
	}
}

func TestLoginChatGPTPlan_CallbackClientRefused_0043(t *testing.T) {
	for _, tc := range []struct {
		name, callback string
		previous       bool
	}{
		{"a first sign-in without an issued client", "-", false},
		{"a first sign-in naming the registration client", planDynamicClientID, false},
		{"a re-sign-in naming another client", "oaiapp_other", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pl := newPlanLogin(t)
			pl.callbackClient = tc.callback
			opts := pl.options()
			if tc.previous {
				opts.Previous = &OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: pl.srv.URL, ClientID: "oaiapp_prev", Resource: planResource}
			}
			session, err := LoginChatGPTPlan(context.Background(), opts)
			if err == nil || session != nil {
				t.Fatalf("session %v, err %v; want a refusal", session, err)
			}
			if pl.tokens.Load() != 0 {
				t.Errorf("the code was exchanged %d times before the refusal", pl.tokens.Load())
			}
		})
	}
}

func TestLoginChatGPTPlan_GrantWithoutPlanUsage_0043(t *testing.T) {
	for _, scope := range []string{"openid profile email offline_access", "-"} {
		pl := newPlanLogin(t)
		pl.scope = scope
		if scope == "-" {
			pl.scope = ""
		}
		session, err := LoginChatGPTPlan(context.Background(), pl.options())
		if !errors.Is(err, llmprovider.ErrNotPermitted) || session != nil {
			t.Errorf("scope %q: session %v, err %v; want ErrNotPermitted", scope, session, err)
		}
	}
}

func TestLoginChatGPTPlan_IDTokenRefused_0043(t *testing.T) {
	t.Run("another audience", func(t *testing.T) {
		pl := newPlanLogin(t)
		pl.tokenClient = "oaiapp_someone_else"
		if session, err := LoginChatGPTPlan(context.Background(), pl.options()); !errors.Is(err, errIDToken) || session != nil {
			t.Errorf("session %v, err %v; want the ID token refused", session, err)
		}
	})
	t.Run("another nonce", func(t *testing.T) {
		pl := newPlanLogin(t)
		opts := pl.options()
		opts.OpenURL = func(raw string) error {
			callback, err := pl.callbackURL(raw)
			if err != nil {
				return err
			}
			pl.ti.mu.Lock()
			pl.ti.nonce = "not-the-login-nonce"
			pl.ti.mu.Unlock()
			resp, err := http.Get(callback)
			if err != nil {
				return err
			}
			return resp.Body.Close()
		}
		if session, err := LoginChatGPTPlan(context.Background(), opts); !errors.Is(err, errIDToken) || session != nil {
			t.Errorf("session %v, err %v; want the ID token refused", session, err)
		}
	})
}

// TestLoginChatGPTPlan_InvalidOptions_0043: refused before any request.
func TestLoginChatGPTPlan_InvalidOptions_0043(t *testing.T) {
	client := &http.Client{Transport: failingTransport{}}
	for name, opts := range map[string]ChatGPTPlanOptions{
		"no app name":    {HTTPClient: client},
		"a blank name":   {AgentName: "  ", HTTPClient: client},
		"Previous Codex": {AgentName: planTestAgent, HTTPClient: client, Previous: &OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: DefaultOpenAIIssuer, ClientID: DefaultOpenAIClientID}},
		"Previous with no client": {AgentName: planTestAgent, HTTPClient: client,
			Previous: &OAuthSession{Provider: llmprovider.ProviderOpenAI, Resource: planResource}},
	} {
		// Options that pass by mistake reach a login that waits for a
		// browser; the deadline makes that a quick failure.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := LoginChatGPTPlan(ctx, opts)
		cancel()
		if !errors.Is(err, llmprovider.ErrInvalidRequest) {
			t.Errorf("%s: err = %v, want ErrInvalidRequest", name, err)
		}
	}
}

// TestLoginChatGPTPlan_OffOriginDiscovery_0043: a token endpoint on another
// host is refused before the browser opens (A2).
func TestLoginChatGPTPlan_OffOriginDiscovery_0043(t *testing.T) {
	pl := newPlanLogin(t)
	pl.discovery = map[string]string{"token_endpoint": "https://evil.example/api/accounts/oauth/token"}
	_, err := LoginChatGPTPlan(context.Background(), pl.options())
	if err == nil || !strings.Contains(err.Error(), "another origin") {
		t.Fatalf("err = %v, want a refusal naming another origin", err)
	}
	if pl.opened.Load() != 0 {
		t.Error("the browser opened")
	}
}

// TestLoginChatGPTPlan_PastedCallback_0043: the pasted callback URL carries
// the issued client.
func TestLoginChatGPTPlan_PastedCallback_0043(t *testing.T) {
	pl := newPlanLogin(t)
	authorize := make(chan string, 1)
	opts := pl.options()
	opts.OpenURL = func(raw string) error {
		authorize <- raw
		return nil
	}
	opts.InputCode = func(ctx context.Context) (string, error) {
		select {
		case raw := <-authorize:
			return pl.callbackURL(raw)
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	session, err := LoginChatGPTPlan(context.Background(), opts)
	if err != nil || session.ClientID != "oaiapp_issued0043" {
		t.Fatalf("session %v, err %v", session, err)
	}
}

func TestLoginChatGPTPlan_GivenHostID_0043(t *testing.T) {
	pl := newPlanLogin(t)
	opts := pl.options()
	opts.HostID = "urn:ietf:params:oauth:jwk-thumbprint:sha-256:0043"
	session, err := LoginChatGPTPlan(context.Background(), opts)
	if err != nil || session.HostID != opts.HostID || pl.authorize.Get("ext_agent_host_id") != opts.HostID {
		t.Fatalf("session %v, err %v, sent %q", session, err, pl.authorize.Get("ext_agent_host_id"))
	}
}

func TestParseOAuthCallback_ClientID_0043(t *testing.T) {
	got := parseOAuthCallback("http://127.0.0.1:1455/auth/callback?code=c&state=s&client_id=+oaiapp_x+", "s")
	if got.err != nil || got.code != "c" || got.clientID != "oaiapp_x" {
		t.Errorf("a pasted URL = %+v", got)
	}
	if got := parseOAuthCallback("bare-code", "s"); got.err != nil || got.code != "bare-code" || got.clientID != "" {
		t.Errorf("a bare code = %+v", got)
	}
}
