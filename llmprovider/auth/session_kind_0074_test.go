package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// The session kind (0074-MADR D5; 0074-PLAN Phase 2).

const siteTokenURL = "https://issuer.example.test/api/accounts/oauth/token"

// siteCase is a session as one kind's sign-in or an older record leaves it,
// and what the seven dispatch sites made of it at 773b84c, before Kind
// existed. refreshURL "" means the refresh gets no token URL.
type siteCase struct {
	name       string
	kind       SessionKind
	build      func() *OAuthSession
	owner      llmprovider.ProviderID
	chatGPT    bool
	plan       bool
	validates  bool
	refreshURL string
	json       bool
	// kindOnly cases check the refresh alone: their Owner, ChatGPT and
	// PlanUsage with a Kind set are the kind's, not the older rules'.
	refreshOnly bool
}

func siteSession(provider llmprovider.ProviderID, issuer string) *OAuthSession {
	return &OAuthSession{Provider: provider, Access: "a", Refresh: "r", ClientID: "c", Issuer: issuer,
		Expiry: time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)}
}

func siteCases() []siteCase {
	plan := func(provider llmprovider.ProviderID) func() *OAuthSession {
		return func() *OAuthSession {
			s := siteSession(provider, DefaultOpenAIIssuer)
			s.Resource, s.TokenURL = planResource, siteTokenURL
			return s
		}
	}
	return []siteCase{
		{name: "chatgpt", kind: SessionChatGPT, build: func() *OAuthSession { return siteSession(llmprovider.ProviderOpenAI, DefaultOpenAIIssuer) },
			owner: llmprovider.ProviderOpenAI, chatGPT: true, validates: true, refreshURL: DefaultOpenAIIssuer + "/oauth/token", json: true},
		{name: "chatgpt, no issuer", kind: SessionChatGPT, build: func() *OAuthSession { return siteSession(llmprovider.ProviderOpenAI, "") },
			owner: llmprovider.ProviderOpenAI, chatGPT: true, validates: true, refreshURL: DefaultOpenAIIssuer + "/oauth/token"},
		{name: "chatgpt, no provider", kind: SessionChatGPT, build: func() *OAuthSession { return siteSession("", DefaultOpenAIIssuer) },
			owner: llmprovider.ProviderOpenAI, chatGPT: true, validates: true, refreshURL: DefaultOpenAIIssuer + "/oauth/token", json: true},
		{name: "plan", kind: SessionChatGPTPlan, build: plan(llmprovider.ProviderOpenAI),
			owner: llmprovider.ProviderOpenAI, plan: true, validates: true, refreshURL: siteTokenURL},
		{name: "plan, no provider", kind: SessionChatGPTPlan, build: plan(""),
			owner: llmprovider.ProviderOpenAI, plan: true, validates: true, refreshURL: siteTokenURL},
		{name: "grok", kind: SessionGrok, build: func() *OAuthSession { return siteSession(llmprovider.ProviderGrok, DefaultGrokOAuthIssuer) },
			owner: llmprovider.ProviderGrok, validates: true, refreshURL: defaultGrokOAuthRefreshURL},
		{name: "grok, no issuer", kind: SessionGrok, build: func() *OAuthSession { return siteSession(llmprovider.ProviderGrok, "") },
			owner: llmprovider.ProviderGrok, validates: true, refreshURL: defaultGrokOAuthRefreshURL},
		{name: "chatgpt, access only", kind: SessionChatGPT, build: func() *OAuthSession {
			s := siteSession(llmprovider.ProviderOpenAI, DefaultOpenAIIssuer)
			s.Refresh, s.ClientID, s.Expiry = "", DefaultOpenAIClientID, time.Time{}
			return s
		}, owner: llmprovider.ProviderOpenAI, chatGPT: true, validates: true},
		{name: "kilo device", kind: SessionKiloDevice, build: func() *OAuthSession {
			s := siteSession(llmprovider.ProviderKilo, "https://kilo.example.test")
			s.Refresh, s.ClientID, s.Expiry = "", "", time.Time{}
			return s
		}, owner: llmprovider.ProviderKilo},
		{name: "openai, another issuer, no token URL", kind: SessionChatGPT, refreshOnly: true,
			build: func() *OAuthSession { return siteSession(llmprovider.ProviderOpenAI, "https://other.example.test") }},
		{name: "openai, another issuer", kind: SessionChatGPT,
			build: func() *OAuthSession {
				s := siteSession(llmprovider.ProviderOpenAI, "https://other.example.test")
				s.TokenURL = siteTokenURL
				return s
			}, owner: llmprovider.ProviderOpenAI, validates: true, refreshURL: siteTokenURL},
		{name: "chatgpt, no provider, another issuer", kind: SessionChatGPT,
			build: func() *OAuthSession {
				s := siteSession("", "https://other.example.test")
				s.TokenURL = siteTokenURL
				return s
			}, validates: true, refreshURL: siteTokenURL},
		{name: "grok, no provider, another issuer", kind: SessionGrok,
			build: func() *OAuthSession {
				s := siteSession("", "https://other.example.test")
				s.TokenURL = siteTokenURL
				return s
			}, validates: true, refreshURL: siteTokenURL},
		{name: "grok, another issuer, no token URL", kind: SessionGrok, refreshOnly: true,
			build: func() *OAuthSession { return siteSession(llmprovider.ProviderGrok, "https://other.example.test") }},
	}
}

// TestSessionKind_EachSite_0074: Kind set and Kind empty take the branch the
// session took at 773b84c, at the seven sites.
func TestSessionKind_EachSite_0074(t *testing.T) {
	for _, tc := range siteCases() {
		for _, explicit := range []bool{true, false} {
			name := tc.name + ", Kind empty"
			if explicit {
				name = tc.name + ", Kind set"
			}
			t.Run(name, func(t *testing.T) {
				s := tc.build()
				if explicit {
					s.Kind = tc.kind
				}
				if !tc.refreshOnly {
					checkSiteFields(t, tc, s)
				}
				if s.Refresh != "" {
					checkSiteRefresh(t, tc, s)
				}
			})
		}
	}
}

// checkSiteFields checks Owner, ChatGPT, PlanUsage and validation.
func checkSiteFields(t *testing.T, tc siteCase, s *OAuthSession) {
	t.Helper()
	if got := s.Owner(); got != tc.owner {
		t.Errorf("Owner() = %q, want %q", got, tc.owner)
	}
	if got := s.ChatGPT(); got != tc.chatGPT {
		t.Errorf("ChatGPT() = %t, want %t", got, tc.chatGPT)
	}
	if got := s.PlanUsage(); got != tc.plan {
		t.Errorf("PlanUsage() = %t, want %t", got, tc.plan)
	}
	if err := ValidateOAuthSession(s); (err == nil) != tc.validates {
		t.Errorf("ValidateOAuthSession() = %v, want ok %t", err, tc.validates)
	}
}

// checkSiteRefresh checks the refresh request's body and token URL.
func checkSiteRefresh(t *testing.T, tc siteCase, s *OAuthSession) {
	t.Helper()
	req, err := newRefreshRequest(context.Background(), s.refreshState())
	switch {
	case tc.refreshURL == "":
		if !errors.Is(err, llmprovider.ErrAuthFailure) {
			t.Errorf("refresh request = %v, %v; want no token URL, an auth failure", req, err)
		}
	case err != nil:
		t.Errorf("refresh request: %v", err)
	default:
		wantType := "application/x-www-form-urlencoded"
		if tc.json {
			wantType = "application/json"
		}
		if req.URL.String() != tc.refreshURL || req.Header.Get("Content-Type") != wantType {
			t.Errorf("refresh request = %s as %s, want %s as %s", req.URL, req.Header.Get("Content-Type"), tc.refreshURL, wantType)
		}
	}
}

// TestSessionKind_Revocation_0074: Kind set and Kind empty are revoked at the
// same place, in the same form.
func TestSessionKind_Revocation_0074(t *testing.T) {
	for _, tc := range []struct {
		name, path, contentType string
		kind                    SessionKind
		build                   func(issuer string) *OAuthSession
		unsupported             bool
	}{
		{"chatgpt", "/oauth/revoke", "application/json", SessionChatGPT, func(issuer string) *OAuthSession { return siteSession(llmprovider.ProviderOpenAI, issuer) }, false},
		{"grok", "/oauth2/revoke", "application/x-www-form-urlencoded", SessionGrok, func(issuer string) *OAuthSession { return siteSession(llmprovider.ProviderGrok, issuer) }, false},
		{"plan", "/oauth2/revoke", "application/x-www-form-urlencoded", SessionChatGPTPlan, func(issuer string) *OAuthSession {
			s := siteSession(llmprovider.ProviderOpenAI, issuer)
			s.Resource, s.TokenURL = planResource, issuer+"/oauth2/token"
			return s
		}, false},
		{"kilo device", "", "", SessionKiloDevice, func(issuer string) *OAuthSession {
			s := siteSession(llmprovider.ProviderKilo, issuer)
			s.Refresh = ""
			return s
		}, true},
	} {
		for _, explicit := range []bool{true, false} {
			t.Run(tc.name, func(t *testing.T) {
				srv, c := revokeIssuer(t, true)
				s := tc.build(srv.URL)
				s.HTTPClient = srv.Client()
				if explicit {
					s.Kind = tc.kind
				}
				err := RevokeOAuthSession(context.Background(), s)
				if tc.unsupported {
					if !errors.Is(err, llmprovider.ErrUnsupported) {
						t.Errorf("explicit %t: err = %v, want ErrUnsupported", explicit, err)
					}
					return
				}
				if err != nil || c.path != tc.path || c.contentType != tc.contentType {
					t.Errorf("explicit %t: err = %v, request %s as %s; want %s as %s", explicit, err, c.path, c.contentType, tc.path, tc.contentType)
				}
			})
		}
	}
}

// TestSessionKindAccessor_0074: SessionKind is Kind when set, else the kind
// the fields name, else empty, also while a refresh rewrites the session.
func TestSessionKindAccessor_0074(t *testing.T) {
	for _, tc := range siteCases() {
		if tc.name == "kilo device" || tc.refreshOnly || strings.Contains(tc.name, "another issuer") {
			continue
		}
		s := tc.build()
		if got := s.SessionKind(); got != tc.kind {
			t.Errorf("%s: Kind empty: SessionKind() = %q, want the inferred %q", tc.name, got, tc.kind)
		}
		s.Kind = SessionKiloDevice
		if got := s.SessionKind(); got != SessionKiloDevice {
			t.Errorf("%s: Kind set: SessionKind() = %q, want it", tc.name, got)
		}
	}
	for name, s := range map[string]*OAuthSession{
		"a Kilo session of no Kind":  siteSession(llmprovider.ProviderKilo, "https://kilo.example.test"),
		"another provider's session": siteSession("other", "https://other.example.test"),
		"an empty session":           {},
	} {
		if got := s.SessionKind(); got != "" {
			t.Errorf("%s: SessionKind() = %q, want empty", name, got)
		}
	}

	rr := &refreshRecorder{reply: `{"access_token":"next","refresh_token":"next-refresh","expires_in":3600}`}
	srv := rr.serve(t)
	s := siteSession(llmprovider.ProviderOpenAI, DefaultOpenAIIssuer)
	s.TokenURL, s.HTTPClient, s.Expiry = srv.URL+"/token", srv.Client(), time.Now().Add(-time.Minute)
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 50 {
				if got := s.SessionKind(); got != SessionChatGPT {
					t.Errorf("SessionKind() = %q during a refresh, want %q", got, SessionChatGPT)
					return
				}
			}
		})
	}
	if _, err := s.Token(context.Background()); err != nil {
		t.Error(err)
	}
	wg.Wait()
}

// deviceSignIn runs provider's device sign-in against a fake issuer.
func deviceSignIn(t *testing.T, provider llmprovider.ProviderID) *OAuthSession {
	t.Helper()
	clock := newFakeOAuthClock()
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	ti := newTestIssuer(mux, srv.URL)
	alg := "RS256"
	if provider == llmprovider.ProviderGrok {
		alg = "ES256"
		mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
			writeTestJSON(t, w, ti.discovery(t, map[string]string{"device_authorization_endpoint": srv.URL + "/device", "token_endpoint": srv.URL + "/token"}))
		})
		mux.HandleFunc("/device", func(w http.ResponseWriter, _ *http.Request) {
			writeTestJSON(t, w, `{"device_code":"device-secret","user_code":"ABCD-EFGH","verification_uri":"https://example.test/activate","expires_in":60,"interval":1}`)
		})
		mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
			writeTestJSON(t, w, ti.tokenResponse(t, alg, "test-client", "access", "refresh"))
		})
	} else {
		mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
			writeTestJSON(t, w, ti.discovery(t, nil))
		})
		mux.HandleFunc("/api/accounts/deviceauth/usercode", func(w http.ResponseWriter, _ *http.Request) {
			writeTestJSON(t, w, `{"device_auth_id":"device-auth-id","user_code":"OPEN-AI","interval":"2"}`)
		})
		mux.HandleFunc("/api/accounts/deviceauth/token", func(w http.ResponseWriter, _ *http.Request) {
			writeTestJSON(t, w, `{"authorization_code":"authorization-code","code_challenge":"challenge","code_verifier":"server-verifier"}`)
		})
		mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, _ *http.Request) {
			writeTestJSON(t, w, ti.tokenResponse(t, alg, "test-client", "access", "refresh"))
		})
	}
	session, err := LoginDeviceOAuth(context.Background(), provider, OAuthFlowOptions{
		HTTPClient: srv.Client(), ClientID: "test-client", Issuer: srv.URL, now: clock.now, sleep: clock.sleep,
	})
	if err != nil {
		t.Fatalf("LoginDeviceOAuth(%s): %v", provider, err)
	}
	return session
}

// TestSessionKind_SetBySignIn_0074: each sign-in sets its kind.
func TestSessionKind_SetBySignIn_0074(t *testing.T) {
	if got := deviceSignIn(t, llmprovider.ProviderOpenAI).Kind; got != SessionChatGPT {
		t.Errorf("OpenAI device sign-in: Kind = %q, want %q", got, SessionChatGPT)
	}
	if got := deviceSignIn(t, llmprovider.ProviderGrok).Kind; got != SessionGrok {
		t.Errorf("Grok device sign-in: Kind = %q, want %q", got, SessionGrok)
	}

	pl := newPlanLogin(t)
	plan, err := LoginChatGPTPlan(context.Background(), pl.options())
	if err != nil {
		t.Fatal(err)
	}
	if plan.Kind != SessionChatGPTPlan {
		t.Errorf("LoginChatGPTPlan: Kind = %q, want %q", plan.Kind, SessionChatGPTPlan)
	}

	srv, _ := kiloDeviceServer(t, http.StatusOK, http.StatusOK)
	login, err := StartDeviceOAuth(context.Background(), llmprovider.ProviderKilo, kiloOptions(srv, newFakeOAuthClock()))
	if err != nil {
		t.Fatal(err)
	}
	kilo, err := login.Wait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if kilo.Kind != SessionKiloDevice {
		t.Errorf("Kilo device sign-in: Kind = %q, want %q", kilo.Kind, SessionKiloDevice)
	}
}

// TestSessionKind_StoreRoundTrip_0074: kind is stored as "kind"; an older
// record without it loads with Kind empty and infers as before.
func TestSessionKind_StoreRoundTrip_0074(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileTokenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	s := siteSession(llmprovider.ProviderGrok, DefaultGrokOAuthIssuer)
	s.Kind = SessionGrok
	if err := store.Save(ctx, llmprovider.ProviderGrok, s); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "grok.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"kind": "grok"`) {
		t.Errorf("the record = %s, want a \"kind\" field", raw)
	}
	loaded, err := store.Load(ctx, llmprovider.ProviderGrok)
	if err != nil || loaded.Kind != SessionGrok {
		t.Errorf("Load: Kind %q, err %v; want %q", loaded.Kind, err, SessionGrok)
	}

	// A record this release did not write carries a kind a file may name.
	written := `{"provider":"grok","access":"a","refresh":"r","expiry":"2026-11-12T12:00:00Z","issuer":"` +
		DefaultGrokOAuthIssuer + `","client_id":"c","account_id":"","token_url":"","kind":"chatgpt_plan","resource":"` + planResource + `"}`
	if err := os.WriteFile(filepath.Join(dir, "grok.json"), []byte(written), 0o600); err != nil {
		t.Fatal(err)
	}
	if loaded, err = store.Load(ctx, llmprovider.ProviderGrok); err != nil || loaded.Kind != SessionChatGPTPlan {
		t.Errorf("a record naming a kind: Kind %q, err %v; want %q", loaded.Kind, err, SessionChatGPTPlan)
	}

	old := `{"provider":"openai","access":"a","refresh":"r","expiry":"2026-11-12T12:00:00Z","issuer":"` +
		DefaultOpenAIIssuer + `","client_id":"c","account_id":"","token_url":""}`
	if err := os.WriteFile(filepath.Join(dir, "openai.json"), []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	if loaded, err = store.Load(ctx, llmprovider.ProviderOpenAI); err != nil || loaded.Kind != "" {
		t.Fatalf("an older record: Kind %q, err %v; want empty", loaded.Kind, err)
	}
	if got := loaded.SessionKind(); got != SessionChatGPT || !loaded.ChatGPT() {
		t.Errorf("an older record infers %q (ChatGPT %t), want %q", got, loaded.ChatGPT(), SessionChatGPT)
	}
}

// storedFieldValues sets every exported stored field of a session to a
// distinct value: all but Store, HTTPClient and Logger.
func storedFieldValues(t *testing.T) *OAuthSession {
	t.Helper()
	s := &OAuthSession{}
	v := reflect.ValueOf(s).Elem()
	for i := range v.NumField() {
		field := v.Type().Field(i)
		if !field.IsExported() || field.Name == "Store" || field.Name == "HTTPClient" || field.Name == "Logger" {
			continue
		}
		switch f := v.Field(i); {
		case field.Type == reflect.TypeFor[time.Time]():
			f.Set(reflect.ValueOf(time.Date(2026, 11, 12, 10+i, 0, 0, 0, time.UTC)))
		case f.Kind() == reflect.String:
			f.SetString("value-" + field.Name)
		case f.Kind() == reflect.Bool:
			f.SetBool(true)
		default:
			t.Fatalf("field %s has type %s: teach this test to set it", field.Name, field.Type)
		}
	}
	// Save keys the record by the provider it is given, a file name.
	s.Provider = "provider-0074"
	return s
}

// compareStored reports each exported stored field of got that differs from
// want, except those named in except.
func compareStored(t *testing.T, stage string, got, want *OAuthSession, except ...string) {
	t.Helper()
	g, w := reflect.ValueOf(got).Elem(), reflect.ValueOf(want).Elem()
	for i := range g.NumField() {
		field := g.Type().Field(i)
		if !field.IsExported() || field.Name == "Store" || field.Name == "HTTPClient" || field.Name == "Logger" {
			continue
		}
		skip := false
		for _, name := range except {
			skip = skip || name == field.Name
		}
		if !skip && !reflect.DeepEqual(g.Field(i).Interface(), w.Field(i).Interface()) {
			t.Errorf("%s: field %s = %v, want %v", stage, field.Name, g.Field(i).Interface(), w.Field(i).Interface())
		}
	}
}

// TestSessionFieldsCarried_0074: every exported stored field survives a store
// round trip, persistable, a refresh and adopt (MADR C27 to C31).
func TestSessionFieldsCarried_0074(t *testing.T) {
	ctx := context.Background()
	want := storedFieldValues(t)
	if want.Kind == "" {
		t.Fatal("the session has no Kind to carry")
	}

	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, want.Provider, want); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(ctx, want.Provider)
	if err != nil {
		t.Fatal(err)
	}
	compareStored(t, "Save and Load", loaded, want)
	compareStored(t, "persistable", want.persistable(), want)

	rr := &refreshRecorder{reply: `{"access_token":"next","refresh_token":"next-refresh","expires_in":3600}`}
	srv := rr.serve(t)
	state := want.refreshState()
	state.tokenURL, state.httpClient = srv.URL+"/token", srv.Client()
	next, _, err := refreshOAuthSession(ctx, state)
	if err != nil {
		t.Fatal(err)
	}
	// A refresh rotates the tokens and the expiry, and sets the token URL
	// this test pointed it at; nothing else.
	compareStored(t, "refreshState to the refreshed session", next, want, "Access", "Refresh", "Expiry", "TokenURL")

	adopted := &OAuthSession{}
	adopted.adopt(want)
	compareStored(t, "adopt", adopted, want)
}

// TestValidateOAuthSession_KindContradiction_0074: a Kind that disagrees with
// the fields is refused.
func TestValidateOAuthSession_KindContradiction_0074(t *testing.T) {
	plan := siteSession(llmprovider.ProviderOpenAI, DefaultOpenAIIssuer)
	plan.Kind, plan.TokenURL = SessionChatGPTPlan, siteTokenURL
	kilo := siteSession(llmprovider.ProviderKilo, "https://kilo.example.test")
	kilo.Kind = SessionKiloDevice
	for name, tc := range map[string]struct {
		s    *OAuthSession
		want string
	}{
		"a chatgpt_plan without a resource":  {plan, "chatgpt_plan"},
		"a kilo_device with a refresh token": {kilo, "kilo_device"},
	} {
		err := ValidateOAuthSession(tc.s)
		if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "contradicts") {
			t.Errorf("%s: err = %v, want a refusal naming %s and the contradiction", name, err, tc.want)
		}
	}
	plan.Resource = planResource
	if err := ValidateOAuthSession(plan); err != nil {
		t.Errorf("a plan session with its resource: %v", err)
	}
}
