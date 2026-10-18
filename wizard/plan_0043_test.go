package wizard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
)

const planTestApp = "test-app"

// planSessionFor is a plan session issued to client, as
// auth.LoginChatGPTPlan returns one.
func planSessionFor(client string) *auth.OAuthSession {
	return &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: auth.DefaultOpenAIIssuer,
		Access: "plan-access-" + client, Refresh: "plan-refresh-" + client, Expiry: time.Now().Add(time.Hour),
		ClientID: client, TokenURL: auth.DefaultOpenAIIssuer + "/api/accounts/oauth/token",
		Resource: openai.PlatformBaseURL, Scope: "openid chatgpt.tokens.use.direct", HostID: "urn:uuid:" + client,
		IDToken: "id-token-" + client}
}

// planMethodIdx is the plan sign-in's place in the menu that offers every
// OpenAI method.
func planMethodIdx(t *testing.T) int {
	t.Helper()
	d, _ := (Options{}).registry().Descriptor(llmprovider.ProviderOpenAI)
	i := slices.IndexFunc(d.AuthMethods, func(m llmprovider.AuthMethod) bool { return m.ID == llmprovider.AuthChatGPTPlan })
	if i < 0 {
		t.Fatal("the OpenAI descriptor has no ChatGPT plan method")
	}
	return i
}

// stubPlanLogin replaces the plan login for one test, counting its calls.
func stubPlanLogin(t *testing.T, login func(context.Context, auth.ChatGPTPlanOptions) (*auth.OAuthSession, error)) *int {
	t.Helper()
	original := loginChatGPTPlan
	t.Cleanup(func() { loginChatGPTPlan = original })
	calls := 0
	loginChatGPTPlan = func(ctx context.Context, opts auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		calls++
		return login(ctx, opts)
	}
	return &calls
}

// TestPlan_OfferedOnlyWithAppName_0043: the plan sign-in is offered only with
// a store and the consumer's app name, last (0043-MADR D7).
func TestPlan_OfferedOnlyWithAppName_0043(t *testing.T) {
	if llmprovider.AuthChatGPTPlan != "chatgpt_plan" {
		t.Errorf("AuthChatGPTPlan = %q; want D7's chatgpt_plan, which a consumer may store", llmprovider.AuthChatGPTPlan)
	}
	label := methodLabel(t, llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan)
	for _, c := range []struct {
		name  string
		o     Options
		offer bool
	}{
		{"a store and a name", Options{TokenStore: newMemoryTokenStore(), AgentName: planTestApp}, true},
		{"a store, no name", Options{TokenStore: newMemoryTokenStore()}, false},
		{"a store, a blank name", Options{TokenStore: newMemoryTokenStore(), AgentName: "  "}, false},
		{"a name, no store", Options{AgentName: planTestApp}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderOpenAI)}})
			_, _ = ConfigureLLM(context.Background(), f, c.o)
			if len(f.seenSelectItems) < 2 {
				t.Fatalf("Select titles = %q, want the method menu second", f.seenSelect)
			}
			menu := labels(f.seenSelectItems[1])
			if got := slices.Contains(menu, label); got != c.offer {
				t.Errorf("menu %q offers the plan sign-in: %v; want %v", menu, got, c.offer)
			}
			if c.offer && menu[len(menu)-1] != label {
				t.Errorf("menu %q; want the plan sign-in last", menu)
			}
		})
	}
}

// TestPlan_FirstSignIn_0043: with no plan session saved, the user confirms
// OpenAI's notice with the usage page, the login gets the app's name, the
// caller's client and the browser hooks, and the session is saved under
// openai; the paste instruction asks for the URL (0043-MADR D7, A3).
func TestPlan_FirstSignIn_0043(t *testing.T) {
	client := &http.Client{}
	calls := stubPlanLogin(t, func(_ context.Context, opts auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		if opts.AgentName != planTestApp || opts.Previous != nil || opts.HTTPClient != client ||
			opts.OpenURL == nil || opts.InputCode == nil {
			t.Errorf("login options %+v; want the app name, no Previous, the caller's client and both hooks", opts)
		}
		if err := opts.OpenURL("https://auth.test/authorize"); err != nil {
			t.Errorf("OpenURL: %v", err)
		}
		return planSessionFor("oaiapp_first"), nil
	})
	store := newMemoryTokenStore()
	f := newFake(t, fakePrompter{
		selects:  []int{providerIdx(t, llmprovider.ProviderOpenAI), planMethodIdx(t)},
		confirms: []bool{true},
		inputs:   []string{"gpt-6.1-sol"},
	})
	res, err := ConfigureLLM(context.Background(), f, Options{TokenStore: store, AgentName: planTestApp, HTTPClient: client})
	if err != nil {
		t.Fatal(err)
	}
	if *calls != 1 || res.Kind != CredOAuth || res.ClientID != "oaiapp_first" || res.Model != "gpt-6.1-sol" {
		t.Errorf("calls %d, result %+v", *calls, res)
	}
	saved := store.sessions[llmprovider.ProviderOpenAI]
	if saved == nil || !saved.PlanUsage() || saved.ClientID != "oaiapp_first" || saved.Store != store {
		t.Errorf("saved %v; want the plan session, bound to the store", saved)
	}
	if len(f.seenConfirm) != 1 || !strings.Contains(f.seenConfirm[0], "You're using your ChatGPT plan") ||
		!strings.Contains(f.seenConfirm[0], openai.ChatGPTUsageURL) {
		t.Errorf("confirms %q; want OpenAI's notice with the usage page", f.seenConfirm)
	}
	if !slices.Contains(f.seenNotify, pasteURLInstruction) || slices.Contains(f.seenNotify, pasteCodeInstruction) {
		t.Errorf("notices %q; want the paste instruction to ask for the URL", f.seenNotify)
	}
}

// TestPlan_PastePrompt_0043: the paste prompt of a first plan sign-in asks
// for the redirected URL, which names the issued client (0043-MADR A3).
func TestPlan_PastePrompt_0043(t *testing.T) {
	callback := "http://127.0.0.1:1455/auth/callback?code=c&state=s&client_id=oaiapp_x"
	f := newFake(t, fakePrompter{inputs: []string{callback}})
	flow, drain := browserFlowOptionsFor(f, Options{}, pasteURLInstruction, pasteURLPrompt)
	if err := flow.OpenURL("https://auth.test/authorize"); err != nil {
		t.Fatal(err)
	}
	got, err := flow.InputCode(context.Background())
	drain()
	if err != nil || got != callback || !slices.Equal(f.seenInput, []string{pasteURLPrompt}) {
		t.Errorf("InputCode = %q, %v; prompts %q; want the paste, asked with %q", got, err, f.seenInput, pasteURLPrompt)
	}
}

// TestPlan_NoticeDeclined_0043: declining the notice signs nobody in.
func TestPlan_NoticeDeclined_0043(t *testing.T) {
	calls := stubPlanLogin(t, func(context.Context, auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		return planSessionFor("oaiapp_first"), nil
	})
	store := newMemoryTokenStore()
	f := newFake(t, fakePrompter{
		selects:  []int{providerIdx(t, llmprovider.ProviderOpenAI), planMethodIdx(t)},
		confirms: []bool{false},
	})
	_, err := ConfigureLLM(context.Background(), f, Options{TokenStore: store, AgentName: planTestApp})
	if err == nil || !strings.Contains(err.Error(), "declined") || *calls != 0 || store.saves != 0 {
		t.Errorf("err %v, %d logins, %d saves; want declined, none, none", err, *calls, store.saves)
	}
}

// TestPlan_ReSignIn_0043: a saved plan session is signed in to again, with
// no notice, and the new session replaces it (0043-MADR D7).
func TestPlan_ReSignIn_0043(t *testing.T) {
	calls := stubPlanLogin(t, func(_ context.Context, opts auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		if opts.Previous == nil || opts.Previous.ClientID != "oaiapp_prev" || opts.Previous.HostID != "urn:uuid:oaiapp_prev" ||
			opts.Previous.IDToken != "id-token-oaiapp_prev" {
			t.Errorf("Previous %v; want the saved plan session", opts.Previous)
		}
		if err := opts.OpenURL("https://auth.test/authorize"); err != nil {
			t.Errorf("OpenURL: %v", err)
		}
		return planSessionFor("oaiapp_prev"), nil
	})
	store := newMemoryTokenStore()
	store.sessions[llmprovider.ProviderOpenAI] = planSessionFor("oaiapp_prev")
	f := newFake(t, fakePrompter{
		selects: []int{providerIdx(t, llmprovider.ProviderOpenAI), planMethodIdx(t)},
		inputs:  []string{"gpt-6.1-sol"},
	})
	if _, err := ConfigureLLM(context.Background(), f, Options{TokenStore: store, AgentName: planTestApp}); err != nil {
		t.Fatal(err)
	}
	if *calls != 1 || len(f.seenConfirm) != 0 || store.saves != 1 {
		t.Errorf("%d logins, confirms %q, %d saves; want one login, no notice, one save", *calls, f.seenConfirm, store.saves)
	}
	if !slices.Contains(f.seenNotify, pasteCodeInstruction) {
		t.Errorf("notices %q; want the usual paste instruction", f.seenNotify)
	}
}

// TestPlan_ReplacesCodexSession_0043: a saved Codex-id session is not a
// previous plan session: the notice is shown, and the plan session replaces
// it, one OpenAI login per store (0043-MADR D3).
func TestPlan_ReplacesCodexSession_0043(t *testing.T) {
	stubPlanLogin(t, func(_ context.Context, opts auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		if opts.Previous != nil {
			t.Errorf("Previous %v; want none for a Codex-id session", opts.Previous)
		}
		return planSessionFor("oaiapp_first"), nil
	})
	store := newMemoryTokenStore()
	store.sessions[llmprovider.ProviderOpenAI] = testOAuthSession(llmprovider.ProviderOpenAI)
	f := newFake(t, fakePrompter{
		selects:  []int{providerIdx(t, llmprovider.ProviderOpenAI), planMethodIdx(t)},
		confirms: []bool{true},
		inputs:   []string{"gpt-6.1-sol"},
	})
	if _, err := ConfigureLLM(context.Background(), f, Options{TokenStore: store, AgentName: planTestApp}); err != nil {
		t.Fatal(err)
	}
	if saved := store.sessions[llmprovider.ProviderOpenAI]; saved == nil || !saved.PlanUsage() {
		t.Errorf("saved %v; want the plan session in the Codex session's place", saved)
	}
}

// TestPlan_ExistingDefault_0043: an Existing OpenAI session under an issued
// client defaults the menu to the plan sign-in; one under Codex's client, to
// the browser sign-in, as before (0020-MADR F19; 0043-MADR D7).
func TestPlan_ExistingDefault_0043(t *testing.T) {
	d, _ := (Options{}).registry().Descriptor(llmprovider.ProviderOpenAI)
	browser := slices.IndexFunc(d.AuthMethods, func(m llmprovider.AuthMethod) bool { return m.ID == llmprovider.AuthBrowserOAuth })
	for client, want := range map[string]int{"oaiapp_prev": planMethodIdx(t), auth.DefaultOpenAIClientID: browser} {
		f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderOpenAI)}})
		existing := Result{Provider: llmprovider.ProviderOpenAI, Kind: CredOAuth, ClientID: client}
		_, _ = ConfigureLLM(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), AgentName: planTestApp, Existing: existing})
		if len(f.seenSelectDefault) < 2 || f.seenSelectDefault[1] != want {
			t.Errorf("client %s: method defaults %v; want %d", client, f.seenSelectDefault, want)
		}
	}
}

// TestPlan_LogoutRevokes_0043: logout revokes a plan session at its issuer's
// discovered revocation endpoint, as a form with the issued client, then
// deletes it (0043-MADR D2, D7).
func TestPlan_LogoutRevokes_0043(t *testing.T) {
	var revoked []string
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{"issuer": srv.URL, "authorization_endpoint": srv.URL + "/authorize",
				"token_endpoint": srv.URL + "/token", "revocation_endpoint": srv.URL + "/revoke", "jwks_uri": srv.URL + "/jwks"})
		case "/revoke":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			revoked = append(revoked, r.PostForm.Get("token")+" "+r.PostForm.Get("client_id"))
		default:
			t.Errorf("request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	store := newMemoryTokenStore()
	session := planSessionFor("oaiapp_prev")
	session.Issuer, session.TokenURL = srv.URL, srv.URL+"/token"
	store.sessions[llmprovider.ProviderOpenAI] = session
	f := newFake(t, fakePrompter{confirms: []bool{true}})
	if err := Logout(context.Background(), f, Options{TokenStore: store}, llmprovider.ProviderOpenAI); err != nil {
		t.Fatal(err)
	}
	if want := []string{"plan-refresh-oaiapp_prev oaiapp_prev"}; !slices.Equal(revoked, want) {
		t.Errorf("revoked %q; want %q", revoked, want)
	}
	if store.sessions[llmprovider.ProviderOpenAI] != nil || slices.ContainsFunc(f.seenNotify, func(s string) bool {
		return strings.Contains(s, "could not revoke")
	}) {
		t.Errorf("session %v, notices %q; want it revoked and deleted", store.sessions[llmprovider.ProviderOpenAI], f.seenNotify)
	}
}
