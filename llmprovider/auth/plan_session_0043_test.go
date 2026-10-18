package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// A plan session's fields, kind, refresh, revocation and store record
// (0043-MADR D1–D3, A1, A2; 0043-PLAN Phase 1).

const (
	planTestIDToken = "planted-id-token-0043"
	planTestHostID  = "urn:uuid:4f1c6e2a-0b7d-4e55-9a31-0043aa0043bb"
)

// planTestSession is a plan session as LoginChatGPTPlan returns one.
func planTestSession(tokenURL string) *OAuthSession {
	return &OAuthSession{
		Provider: llmprovider.ProviderOpenAI, Access: "planted-access-0043", Refresh: "planted-refresh-0043",
		Expiry: time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC), Issuer: DefaultOpenAIIssuer, ClientID: "oaiapp_0043",
		TokenURL: tokenURL, Resource: planResource, Scope: "openid profile email offline_access resource.invoke chatgpt.tokens.use.direct",
		IDToken: planTestIDToken, HostID: planTestHostID,
	}
}

func TestPlanSession_StoreRoundTrip_0043(t *testing.T) {
	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	in := planTestSession(DefaultOpenAIIssuer + "/api/accounts/oauth/token")
	if err := store.Save(ctx, llmprovider.ProviderOpenAI, in); err != nil {
		t.Fatal(err)
	}
	out, err := store.Load(ctx, llmprovider.ProviderOpenAI)
	if err != nil || out == nil {
		t.Fatalf("Load = %v, %v", out, err)
	}
	if out.Resource != in.Resource || out.Scope != in.Scope || out.IDToken != in.IDToken || out.HostID != in.HostID ||
		out.ClientID != in.ClientID || out.TokenURL != in.TokenURL || out.Access != in.Access || out.Refresh != in.Refresh {
		t.Errorf("round trip lost a field:\n got %s %s %q %s\nwant %s %s %q %s", out.Resource, out.ClientID, out.Scope, out.HostID,
			in.Resource, in.ClientID, in.Scope, in.HostID)
	}
	if out.IDToken != planTestIDToken {
		t.Error("round trip lost the ID token")
	}
	raw, err := os.ReadFile(filepath.Join(store.Dir, "openai.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"resource"`, `"scope"`, `"id_token"`, `"host_id"`} {
		if !bytes.Contains(raw, []byte(key)) {
			t.Errorf("the record has no %s", key)
		}
	}
}

// TestPlanSession_OlderRecords_0043: a record without the new fields, as
// v1.14.0 writes one, loads as before, and a Codex-id session's record gains
// no new key (D3).
func TestPlanSession_OlderRecords_0043(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileTokenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	older := `{"provider":"openai","access":"a","refresh":"r","expiry":"2026-10-18T12:00:00Z","issuer":"https://auth.openai.com","client_id":"app_EMoamEEZ73f0CkXaXp7hrann","account_id":"acct","token_url":""}`
	if err := os.WriteFile(filepath.Join(dir, "openai.json"), []byte(older), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := store.Load(ctx, llmprovider.ProviderOpenAI)
	if err != nil || s == nil {
		t.Fatalf("Load = %v, %v", s, err)
	}
	if s.Resource != "" || s.Scope != "" || s.IDToken != "" || s.HostID != "" || !s.ChatGPT() || s.PlanUsage() {
		t.Errorf("an older record loaded as %s; want a Codex-id ChatGPT session", s)
	}
	if err := store.Save(ctx, llmprovider.ProviderOpenAI, s); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "openai.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"resource"`, `"scope"`, `"id_token"`, `"host_id"`} {
		if bytes.Contains(raw, []byte(key)) {
			t.Errorf("a Codex-id session's record gained %s:\n%s", key, raw)
		}
	}
}

func TestPlanSession_Kind_0043(t *testing.T) {
	for _, tc := range []struct {
		name          string
		s             *OAuthSession
		chatGPT, plan bool
	}{
		{"Codex id", &OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: DefaultOpenAIIssuer}, true, false},
		{"older Codex id, no issuer", &OAuthSession{Provider: llmprovider.ProviderOpenAI}, true, false},
		{"plan", planTestSession(""), false, true},
		{"plan, provider from the issuer", &OAuthSession{Issuer: DefaultOpenAIIssuer, Resource: planResource}, false, true},
		{"a resource on another provider's session", &OAuthSession{Provider: llmprovider.ProviderGrok, Issuer: DefaultGrokOAuthIssuer, Resource: planResource}, false, false},
		{"Grok", &OAuthSession{Provider: llmprovider.ProviderGrok, Issuer: DefaultGrokOAuthIssuer}, false, false},
	} {
		if got := tc.s.ChatGPT(); got != tc.chatGPT {
			t.Errorf("%s: ChatGPT() = %t, want %t", tc.name, got, tc.chatGPT)
		}
		if got := tc.s.PlanUsage(); got != tc.plan {
			t.Errorf("%s: PlanUsage() = %t, want %t", tc.name, got, tc.plan)
		}
	}
}

// refreshRecorder serves a token endpoint and records the one request.
type refreshRecorder struct {
	contentType string
	form        url.Values
	json        map[string]string
	reply       string
}

func (rr *refreshRecorder) serve(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rr.contentType = r.Header.Get("Content-Type")
		if rr.contentType == "application/json" {
			_ = json.NewDecoder(r.Body).Decode(&rr.json)
		} else if err := r.ParseForm(); err == nil {
			rr.form = r.PostForm
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(rr.reply))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestPlanSession_Refresh_0043(t *testing.T) {
	for _, tc := range []struct {
		name, reply    string
		scope, idToken string
	}{
		{"the reply's scope and ID token are kept",
			`{"access_token":"next-access","refresh_token":"next-refresh","expires_in":3600,"scope":"openid chatgpt.tokens.use.direct","id_token":"next-id-token"}`,
			"openid chatgpt.tokens.use.direct", "next-id-token"},
		{"a reply without them keeps the session's",
			`{"access_token":"next-access","refresh_token":"next-refresh","expires_in":3600}`,
			planTestSession("").Scope, planTestIDToken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr := &refreshRecorder{reply: tc.reply}
			srv := rr.serve(t)
			s := planTestSession(srv.URL + "/api/accounts/oauth/token")
			s.Expiry, s.HTTPClient = time.Now().Add(-time.Minute), srv.Client()
			token, err := s.Token(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if token.Value != "next-access" || s.Refresh != "next-refresh" {
				t.Errorf("token %q, refresh %q; want the rotated pair", token.Value, s.Refresh)
			}
			if rr.contentType != "application/x-www-form-urlencoded" {
				t.Errorf("Content-Type = %q, want a form", rr.contentType)
			}
			want := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"planted-refresh-0043"},
				"client_id": {"oaiapp_0043"}, "resource": {planResource}}
			if fmt.Sprint(rr.form) != fmt.Sprint(want) {
				t.Errorf("refresh form = %v, want %v (and no scope)", rr.form, want)
			}
			if s.Scope != tc.scope || s.IDToken != tc.idToken || s.Resource != planResource || s.HostID != planTestHostID {
				t.Errorf("after refresh: scope %q, ID token kept %t, resource %q, host %q", s.Scope, s.IDToken == tc.idToken, s.Resource, s.HostID)
			}
		})
	}
}

// TestPlanSession_RefreshSaved_0043: the rotation a refresh saves keeps the
// plan session's fields.
func TestPlanSession_RefreshSaved_0043(t *testing.T) {
	rr := &refreshRecorder{reply: `{"access_token":"next-access","refresh_token":"next-refresh","expires_in":3600,"scope":"openid chatgpt.tokens.use.direct","id_token":"next-id-token"}`}
	srv := rr.serve(t)
	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	s := planTestSession(srv.URL + "/api/accounts/oauth/token")
	s.Expiry, s.HTTPClient = time.Now().Add(-time.Minute), srv.Client()
	if err := store.Save(ctx, llmprovider.ProviderOpenAI, s); err != nil {
		t.Fatal(err)
	}
	s, err = store.Load(ctx, llmprovider.ProviderOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	s.HTTPClient = srv.Client()
	if _, err := s.Token(ctx); err != nil {
		t.Fatal(err)
	}
	saved, err := store.Load(ctx, llmprovider.ProviderOpenAI)
	if err != nil || saved == nil {
		t.Fatalf("Load = %v, %v", saved, err)
	}
	if saved.Refresh != "next-refresh" || saved.Resource != planResource || saved.Scope != "openid chatgpt.tokens.use.direct" ||
		saved.IDToken != "next-id-token" || saved.HostID != planTestHostID {
		t.Errorf("the saved rotation: refresh %q, resource %q, scope %q, ID token rotated %t, host %q",
			saved.Refresh, saved.Resource, saved.Scope, saved.IDToken == "next-id-token", saved.HostID)
	}
}

// TestPlanSession_AdoptAndPersistable_0043: adopting a session from the store,
// as after another process's rotation, and copying one to retry its save,
// carry each plan field. A refresh keeps them unchanged, so only these show a
// field that is dropped.
func TestPlanSession_AdoptAndPersistable_0043(t *testing.T) {
	next := &OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "a2", Refresh: "r2", Resource: "https://resource.example/v1",
		Scope: "openid other", IDToken: "id-2", HostID: "urn:uuid:other"}
	s := planTestSession("")
	s.adopt(next)
	if s.Resource != next.Resource || s.Scope != next.Scope || s.IDToken != next.IDToken || s.HostID != next.HostID {
		t.Errorf("adopt: resource %q, scope %q, ID token %q, host %q; want next's", s.Resource, s.Scope, s.IDToken, s.HostID)
	}
	in := planTestSession("https://auth.openai.com/api/accounts/oauth/token")
	out := in.persistable()
	if out.Resource != in.Resource || out.Scope != in.Scope || out.IDToken != in.IDToken || out.HostID != in.HostID {
		t.Errorf("persistable: resource %q, scope %q, ID token kept %t, host %q", out.Resource, out.Scope, out.IDToken == in.IDToken, out.HostID)
	}
}

// TestPlanSession_CodexRefreshUnchanged_0043: a Codex-id session still
// refreshes with Codex's JSON body, with no resource (V2).
func TestPlanSession_CodexRefreshUnchanged_0043(t *testing.T) {
	rr := &refreshRecorder{reply: `{"access_token":"next-access","refresh_token":"next-refresh","expires_in":3600}`}
	srv := rr.serve(t)
	s := &OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "a", Refresh: "r", Issuer: DefaultOpenAIIssuer,
		ClientID: DefaultOpenAIClientID, TokenURL: srv.URL + "/oauth/token", Expiry: time.Now().Add(-time.Minute), HTTPClient: srv.Client()}
	if _, err := s.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"grant_type": "refresh_token", "refresh_token": "r", "client_id": DefaultOpenAIClientID}
	if rr.contentType != "application/json" || fmt.Sprint(rr.json) != fmt.Sprint(want) {
		t.Errorf("Codex refresh = %s %v, want JSON %v", rr.contentType, rr.json, want)
	}
}

// planIssuer serves a plan issuer's discovery document, its revocation
// endpoint at revocation (an absolute URL, or a path on this server), and
// records the revocation form.
func planIssuer(t *testing.T, revocation string) (*httptest.Server, *revokeCapture) {
	t.Helper()
	c := &revokeCapture{}
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		doc := map[string]string{
			"issuer": srv.URL, "authorization_endpoint": srv.URL + "/api/accounts/authorize",
			"token_endpoint": srv.URL + "/api/accounts/oauth/token", "jwks_uri": srv.URL + "/jwks",
		}
		switch {
		case strings.HasPrefix(revocation, "/"):
			doc["revocation_endpoint"] = srv.URL + revocation
		case revocation != "":
			doc["revocation_endpoint"] = revocation
		}
		_ = json.NewEncoder(w).Encode(doc)
	})
	mux.HandleFunc("/api/accounts/oauth/revoke", func(w http.ResponseWriter, r *http.Request) {
		c.path, c.contentType = r.URL.Path, r.Header.Get("Content-Type")
		if err := r.ParseForm(); err == nil {
			c.form = r.PostForm
		}
		w.WriteHeader(http.StatusOK)
	})
	return srv, c
}

func TestPlanSession_Revoke_0043(t *testing.T) {
	srv, c := planIssuer(t, "/api/accounts/oauth/revoke")
	s := planTestSession(srv.URL + "/api/accounts/oauth/token")
	s.Issuer, s.HTTPClient = srv.URL, srv.Client()
	if err := RevokeOAuthSession(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"token": {"planted-refresh-0043"}, "token_type_hint": {"refresh_token"}, "client_id": {"oaiapp_0043"}}
	if c.path != "/api/accounts/oauth/revoke" || c.contentType != "application/x-www-form-urlencoded" || fmt.Sprint(c.form) != fmt.Sprint(want) {
		t.Errorf("revocation = %s %s %v, want a form %v at the discovered endpoint", c.path, c.contentType, c.form, want)
	}
}

// TestPlanSession_RevokeOffOrigin_0043: a revocation_endpoint on another
// host is refused, and that host is sent nothing (A2).
func TestPlanSession_RevokeOffOrigin_0043(t *testing.T) {
	var hits atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(other.Close)
	srv, c := planIssuer(t, other.URL+"/api/accounts/oauth/revoke")
	s := planTestSession(srv.URL + "/api/accounts/oauth/token")
	s.Issuer, s.HTTPClient = srv.URL, srv.Client()
	err := RevokeOAuthSession(context.Background(), s)
	if err == nil || !strings.Contains(err.Error(), "another origin") {
		t.Fatalf("err = %v, want a refusal naming another origin", err)
	}
	if hits.Load() != 0 || c.path != "" {
		t.Errorf("the token went somewhere: other host %d hits, issuer %q", hits.Load(), c.path)
	}
}

// TestPlanSession_RevokeNoEndpoint_0043: another issuer that publishes no
// revocation_endpoint is unsupported, as Grok's is.
func TestPlanSession_RevokeNoEndpoint_0043(t *testing.T) {
	srv, _ := planIssuer(t, "")
	s := planTestSession(srv.URL + "/api/accounts/oauth/token")
	s.Issuer, s.HTTPClient = srv.URL, srv.Client()
	if err := RevokeOAuthSession(context.Background(), s); !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("err = %v, want errors.ErrUnsupported", err)
	}
}

// failingTransport fails every request, as an unreachable issuer does.
type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("unreachable")
}

// TestPlanOAuthEndpoints_BuiltinFallback_0043: OpenAI's issuer, unreachable,
// falls back to the endpoints read on 2026-10-18; another issuer fails.
func TestPlanOAuthEndpoints_BuiltinFallback_0043(t *testing.T) {
	client := &http.Client{Transport: failingTransport{}}
	got, err := planOAuthEndpoints(context.Background(), client, DefaultOpenAIIssuer)
	if err != nil || got.Authorization != planBuiltinEndpoints.Authorization || got.Token != planBuiltinEndpoints.Token ||
		got.Revocation != planBuiltinEndpoints.Revocation || got.JWKS != planBuiltinEndpoints.JWKS {
		t.Errorf("planOAuthEndpoints(OpenAI, unreachable) = %+v, %v; want the built-in endpoints", got, err)
	}
	if _, err := planOAuthEndpoints(context.Background(), client, "https://issuer.example"); err == nil {
		t.Error("another issuer, unreachable: no error")
	}
}

func TestRequireSameOrigin_0043(t *testing.T) {
	issuer := "https://auth.example"
	for _, tc := range []struct {
		endpoint string
		ok       bool
	}{
		{"https://auth.example/api/accounts/oauth/token", true},
		{"https://AUTH.example/revoke", true},
		{"", true},
		{"https://evil.example/token", false},
		{"http://auth.example/token", false},
		{"https://auth.example:8443/token", false},
		{"https://auth.example.evil.example/token", false},
	} {
		if err := requireSameOrigin(issuer, tc.endpoint); (err == nil) != tc.ok {
			t.Errorf("requireSameOrigin(%q) = %v, want ok %t", tc.endpoint, err, tc.ok)
		}
	}
	if err := requireSameOrigin("not a url", "https://auth.example/x"); err == nil {
		t.Error("a relative issuer: no error")
	}
}

func TestPlanSession_Validate_0043(t *testing.T) {
	if err := ValidateOAuthSession(planTestSession("https://auth.openai.com/api/accounts/oauth/token")); err != nil {
		t.Errorf("a plan session with a token URL: %v", err)
	}
	if err := ValidateOAuthSession(planTestSession("")); err == nil || !strings.Contains(err.Error(), "no token URL") {
		t.Errorf("a plan session without a token URL: %v, want no token URL", err)
	}
}

// TestPlanSession_Formatting_0043: the ID token is hidden wherever the tokens
// are (R33), and a session without plan fields formats as before.
func TestPlanSession_Formatting_0043(t *testing.T) {
	s := planTestSession("https://auth.openai.com/api/accounts/oauth/token")
	var logged bytes.Buffer
	slog.New(slog.NewJSONHandler(&logged, nil)).Info("session", "s", s)
	marshalled, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{
		"String": s.String(), "%v": fmt.Sprintf("%v", s), "%+v": fmt.Sprintf("%+v", s), "%#v": fmt.Sprintf("%#v", s),
		"MarshalJSON": string(marshalled), "slog": logged.String(),
	} {
		for _, secret := range []string{planTestIDToken, "planted-access-0043", "planted-refresh-0043"} {
			if strings.Contains(text, secret) {
				t.Errorf("%s shows a secret: %s", name, text)
			}
		}
		if !strings.Contains(text, planResource) || !strings.Contains(text, planTestHostID) {
			t.Errorf("%s does not show the resource and host id: %s", name, text)
		}
	}
	codex := &OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: DefaultOpenAIIssuer, ClientID: DefaultOpenAIClientID}
	want := "OAuthSession{Provider:openai Issuer:https://auth.openai.com ClientID:app_EMoamEEZ73f0CkXaXp7hrann AccountID: FedRAMP:false TokenURL: Expiry:none Access: Refresh:}"
	if got := codex.String(); got != want {
		t.Errorf("a Codex-id session's String changed:\n got %s\nwant %s", got, want)
	}
	if raw, _ := json.Marshal(codex); bytes.Contains(raw, []byte("resource")) || bytes.Contains(raw, []byte("id_token")) {
		t.Errorf("a Codex-id session's JSON gained plan fields: %s", raw)
	}
}
