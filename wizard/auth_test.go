package wizard

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// TestConfigureLLM_IgnoresOrchestratorEnv: the wizard has no orchestration
// concept (0002-MADR, sixth amendment), so MCP_ORCHESTRATOR_OWNED does not stop
// the flow before provider selection.
func TestConfigureLLM_IgnoresOrchestratorEnv(t *testing.T) {
	t.Setenv("MCP_ORCHESTRATOR_OWNED", "true")
	f := &fakePrompter{t: t}
	_, _ = ConfigureLLM(context.Background(), f, Options{})
	if len(f.seenSelect) != 1 {
		t.Fatalf("Select calls = %d, want 1: the flow must reach provider selection", len(f.seenSelect))
	}
}

func TestConfigureLLM_ChatGPTResultDoesNotPopulateAPIKey(t *testing.T) {
	store := newMemoryTokenStore()
	f := &fakePrompter{
		t:        t,
		selects:  []int{providerIdx(t, llmprovider.ProviderOpenAI)},
		confirms: []bool{true},
		inputs:   []string{acceptDefault},
	}
	res, err := ConfigureLLM(context.Background(), f, Options{
		Existing: storedExisting(t, store, Result{
			Provider:    llmprovider.ProviderOpenAI,
			Kind:        CredOAuth,
			TokenExpiry: time.Now().Add(time.Hour),
			Issuer:      auth.DefaultOpenAIIssuer,
			ClientID:    auth.DefaultOpenAIClientID,
			AccountID:   "acct_test",
			Model:       "kept-chatgpt-model",
		}, "existing-access-abcd", "existing-refresh"),
		TokenStore: store,
	})
	if err != nil {
		t.Fatalf("ConfigureLLM() error = %v", err)
	}
	if res.APIKey != "" || res.Kind != CredOAuth {
		t.Fatalf("APIKey/Kind = %q/%q, want empty/%q", res.APIKey, res.Kind, CredOAuth)
	}
	if fields := resultFieldsHolding(res, "existing-access-abcd"); len(fields) != 0 || res.Model != "kept-chatgpt-model" {
		t.Fatalf("token in %v, model %q; want no token and the kept model", fields, res.Model)
	}
	assertTextMasksSecret(t, f.allText, "existing-access-abcd")
}

func TestConfigureLLM_APIKeyKindUnchanged(t *testing.T) {
	tests := []struct {
		name       string
		provider   llmprovider.ProviderID
		selections []int
	}{
		{name: "gemini", provider: llmprovider.ProviderGemini},
		{name: "openai", provider: llmprovider.ProviderOpenAI, selections: []int{0}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selects := []int{providerIdx(t, test.provider)}
			selects = append(selects, test.selections...)
			selects = append(selects, 0)
			f := &fakePrompter{t: t, blankSearches: true, selects: selects, secrets: []string{testKey}}
			res, err := ConfigureLLM(context.Background(), f, Options{})
			if err != nil {
				t.Fatalf("ConfigureLLM() error = %v", err)
			}
			if res.Kind != CredAPIKey || res.APIKey != testKey {
				t.Fatalf("Kind/APIKey = %q/%q, want %q/key", res.Kind, res.APIKey, CredAPIKey)
			}
		})
	}
}

func TestConfigureLLM_DoesNotOfferClaudeOAuth(t *testing.T) {
	f := &fakePrompter{
		t:             t,
		blankSearches: true,
		selects:       []int{providerIdx(t, llmprovider.ProviderClaude), 0},
		secrets:       []string{testKey},
	}
	if _, err := ConfigureLLM(context.Background(), f, Options{}); err != nil {
		t.Fatalf("ConfigureLLM() error = %v", err)
	}
	if len(f.seenSelect) != 2 {
		t.Fatalf("Select calls = %d, want provider and model only", len(f.seenSelect))
	}
	for _, title := range f.seenSelect {
		if strings.Contains(strings.ToLower(title), "authentication") {
			t.Fatalf("Claude unexpectedly received auth menu %q", title)
		}
	}
}

func TestConfigureLLM_TokenStdinClassification(t *testing.T) {
	tests := []struct {
		name       string
		provider   llmprovider.ProviderID
		secret     string
		wantKind   CredentialKind
		wantAPIKey string
		wantAccess string
		wantSaves  int
		wantErr    bool
	}{
		{
			name:       "openai platform key",
			provider:   llmprovider.ProviderOpenAI,
			secret:     "sk-platform",
			wantKind:   CredAPIKey,
			wantAPIKey: "sk-platform",
		},
		{
			name:       "openai access token",
			provider:   llmprovider.ProviderOpenAI,
			secret:     jwtShapedAccess,
			wantKind:   CredOAuth,
			wantAccess: jwtShapedAccess,
			wantSaves:  1,
		},
		{
			name:     "chatgpt-access fixture",
			provider: llmprovider.ProviderOpenAI,
			secret:   "chatgpt-access",
			wantErr:  true,
		},
		{
			name:       "grok always api key",
			provider:   llmprovider.ProviderGrok,
			secret:     "xai-key",
			wantKind:   CredAPIKey,
			wantAPIKey: "xai-key",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := newMemoryTokenStore()
			f := &fakePrompter{
				t:             t,
				blankSearches: true,
				selects:       []int{providerIdx(t, test.provider), 3, 0},
				secrets:       []string{test.secret},
			}
			if test.wantKind == CredOAuth {
				f.inputs = []string{"chatgpt-model"}
			}
			res, err := ConfigureLLM(context.Background(), f, Options{TokenStore: store})
			if test.wantErr {
				if err == nil || store.saves != 0 {
					t.Fatalf("ConfigureLLM() error = %v with %d saves, want an error and no save", err, store.saves)
				}
				return
			}
			if err != nil {
				t.Fatalf("ConfigureLLM() error = %v", err)
			}
			if res.Kind != test.wantKind || res.APIKey != test.wantAPIKey {
				t.Fatalf("result credential = %q/%q", res.Kind, res.APIKey)
			}
			if saved := store.sessions[test.provider]; test.wantAccess != "" && (saved == nil || saved.Access != test.wantAccess) {
				t.Fatalf("stored session %v, want access %q", saved, test.wantAccess)
			}
			if test.wantKind == CredOAuth && res.Model != "chatgpt-model" {
				t.Fatalf("oauth model = %q, want chatgpt-model", res.Model)
			}
			if store.saves != test.wantSaves {
				t.Fatalf("TokenStore saves = %d, want %d", store.saves, test.wantSaves)
			}
		})
	}
}

func TestConfigureLLM_BrowserAndDevicePersistSessions(t *testing.T) {
	originalBrowser := loginBrowserOAuth
	originalDevice := loginDeviceOAuth
	t.Cleanup(func() {
		loginBrowserOAuth = originalBrowser
		loginDeviceOAuth = originalDevice
	})

	loginBrowserOAuth = func(_ context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		if opts.OpenURL == nil {
			t.Fatal("browser OpenURL hook is nil")
		}
		if err := opts.OpenURL("https://authorize.test"); err != nil {
			t.Fatalf("OpenURL() error = %v", err)
		}
		return testOAuthSession(provider), nil
	}
	loginDeviceOAuth = func(_ context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		if opts.NotifyDevice == nil {
			t.Fatal("device NotifyDevice hook is nil")
		}
		opts.NotifyDevice("https://verify.test", "ABCD-EFGH")
		return testOAuthSession(provider), nil
	}

	for _, test := range []struct {
		name    string
		authIdx int
	}{
		{name: "browser", authIdx: 1},
		{name: "device", authIdx: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := newMemoryTokenStore()
			f := &fakePrompter{t: t, blankSearches: true, selects: []int{providerIdx(t, llmprovider.ProviderGrok), test.authIdx, 0}}
			res, err := ConfigureLLM(context.Background(), f, Options{TokenStore: store})
			if err != nil {
				t.Fatalf("ConfigureLLM() error = %v", err)
			}
			if res.Kind != CredOAuth || store.saves != 1 || store.sessions[llmprovider.ProviderGrok] == nil {
				t.Fatalf("Kind/saves/session = %q/%d/%v", res.Kind, store.saves, store.sessions[llmprovider.ProviderGrok])
			}
			if len(f.seenNotify) == 0 {
				t.Fatal("OAuth flow produced no user notification")
			}
		})
	}
}

func testOAuthSession(provider llmprovider.ProviderID) *auth.OAuthSession {
	issuer := auth.DefaultGrokOAuthIssuer
	clientID := auth.DefaultGrokOAuthClientID
	if provider == llmprovider.ProviderOpenAI {
		issuer = auth.DefaultOpenAIIssuer
		clientID = auth.DefaultOpenAIClientID
	}
	return &auth.OAuthSession{
		Provider: provider,
		Access:   "oauth-access",
		Refresh:  "oauth-refresh",
		Expiry:   time.Now().Add(time.Hour),
		Issuer:   issuer,
		ClientID: clientID,
	}
}

type memoryTokenStore struct {
	sessions map[llmprovider.ProviderID]*auth.OAuthSession
	saves    int
}

func newMemoryTokenStore() *memoryTokenStore {
	return &memoryTokenStore{sessions: make(map[llmprovider.ProviderID]*auth.OAuthSession)}
}

func (store *memoryTokenStore) Load(_ context.Context, provider llmprovider.ProviderID) (*auth.OAuthSession, error) {
	return store.sessions[provider], nil
}

func (store *memoryTokenStore) Save(_ context.Context, provider llmprovider.ProviderID, session *auth.OAuthSession) error {
	store.sessions[provider] = session
	store.saves++
	return nil
}

func (store *memoryTokenStore) Delete(_ context.Context, provider llmprovider.ProviderID) error {
	delete(store.sessions, provider)
	return nil
}

func assertTextMasksSecret(t *testing.T, displayed []string, secret string) {
	t.Helper()
	joined := strings.Join(displayed, "\n")
	if strings.Contains(joined, secret) {
		t.Fatalf("displayed text contains raw credential %q", secret)
	}
	if !strings.Contains(joined, "••••") {
		t.Fatalf("displayed text has no masked credential: %q", joined)
	}
}

// TestConfigureLLM_ChatGPTListingFailurePromptsForModel: after ChatGPT sign-in
// a failed Codex listing asks for a model id; it never offers the Platform
// catalog or a frozen ChatGPT list (MADR 0008 D11).
func TestConfigureLLM_ChatGPTListingFailurePromptsForModel(t *testing.T) {
	store := newMemoryTokenStore()
	var requests []string
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests = append(requests, r.URL.Host+r.URL.Path)
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    r,
		}, nil
	})}
	f := &fakePrompter{
		t:        t,
		selects:  []int{providerIdx(t, llmprovider.ProviderOpenAI), 1},
		confirms: []bool{true},
		inputs:   []string{"manual-chatgpt"},
	}
	res, err := ConfigureLLM(context.Background(), f, Options{
		Existing: storedExisting(t, store, Result{
			Provider:    llmprovider.ProviderOpenAI,
			Kind:        CredOAuth,
			TokenExpiry: time.Now().Add(time.Hour),
			Issuer:      auth.DefaultOpenAIIssuer,
			ClientID:    auth.DefaultOpenAIClientID,
			AccountID:   "acct_test",
		}, "existing-access-abcd", "existing-refresh"),
		TokenStore: store,
		Discover:   true,
		HTTPClient: client,
	})
	if err != nil {
		t.Fatalf("ConfigureLLM() error = %v", err)
	}
	if res.Model != "manual-chatgpt" {
		t.Fatalf("Model = %q, want the entered id", res.Model)
	}
	if len(requests) != 1 || strings.Contains(requests[0], "api.openai.com") {
		t.Fatalf("listing requests = %v, want one Codex /models call", requests)
	}
	for _, items := range f.seenSelectItems {
		for _, c := range items {
			if strings.Contains(c.Label, "gpt-5.4") || strings.Contains(c.Label, "gpt-4.1") {
				t.Fatalf("model menu offered %q after a failed ChatGPT listing", c.Label)
			}
		}
	}
	for _, n := range f.seenNotify {
		if strings.Contains(n, "built-in catalog") {
			t.Fatalf("notice %q claims a built-in catalog ChatGPT does not have", n)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// jwtShapedAccess is an access-only ChatGPT token as token_stdin receives it.
const jwtShapedAccess = "eyJhbGciOiJub25lIn0.e30.x"

func TestSaveOAuthCredential_RejectsFixture(t *testing.T) {
	store := newMemoryTokenStore()
	_, err := saveOAuthCredential(context.Background(), store, llmprovider.ProviderOpenAI, &auth.OAuthSession{
		Access: "chatgpt-access",
		Issuer: auth.DefaultOpenAIIssuer,
	})
	if err == nil || store.saves != 0 {
		t.Fatalf("saveOAuthCredential() error = %v with %d saves, want an error and no save", err, store.saves)
	}
	if msg := err.Error(); !strings.Contains(msg, "chatgpt-access") && !strings.Contains(msg, "stub") &&
		!strings.Contains(msg, "refresh") {
		t.Fatalf("error = %q, want it to name the fixture or the missing refresh", msg)
	}
}

// TestConfigureLLM_KeepRefusesStubSession: a saved stub is not kept (D7); the
// user must sign in again.
func TestConfigureLLM_KeepRefusesStubSession(t *testing.T) {
	store := newMemoryTokenStore()
	f := &fakePrompter{
		t: t, selects: []int{providerIdx(t, llmprovider.ProviderOpenAI), 1}, confirms: []bool{true},
		inputs: []string{"chatgpt-model"},
	}
	_, err := ConfigureLLM(context.Background(), f, Options{
		Existing: storedExisting(t, store, Result{
			Provider: llmprovider.ProviderOpenAI,
			Kind:     CredOAuth,
			Issuer:   auth.DefaultOpenAIIssuer,
			ClientID: auth.DefaultOpenAIClientID,
		}, "chatgpt-access", ""),
		TokenStore: store,
	})
	if err == nil || !strings.Contains(err.Error(), "sign in again") || store.saves != 1 {
		t.Fatalf("ConfigureLLM() error = %v with %d saves, want the stub refused and no save after the setup's", err, store.saves)
	}
}

// TestConfigureLLM_BrowserOAuthSetsInputCode: browser sign-in races a paste
// prompt, and the authorize URL and paste instruction are shown even when the
// consumer opens the browser itself (MADR 0008 D3).
func TestConfigureLLM_BrowserOAuthSetsInputCode(t *testing.T) {
	stubBrowserLogin(t, func(_ context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		if opts.InputCode == nil {
			t.Fatal("browser OAuth has no paste-code InputCode")
		}
		if err := opts.OpenURL("https://authorize.test"); err != nil {
			t.Fatalf("OpenURL() error = %v", err)
		}
		return testOAuthSession(provider), nil
	})
	opened := 0
	f := &fakePrompter{t: t, blankSearches: true, selects: []int{providerIdx(t, llmprovider.ProviderGrok), 1, 0}}
	_, err := ConfigureLLM(context.Background(), f, Options{
		TokenStore: newMemoryTokenStore(),
		OpenURL:    func(string) error { opened++; return nil },
	})
	if err != nil {
		t.Fatalf("ConfigureLLM() error = %v", err)
	}
	if opened != 1 {
		t.Errorf("consumer OpenURL called %d times, want 1", opened)
	}
	if countContaining(f.seenNotify, "https://authorize.test") != 1 {
		t.Errorf("notices = %v, want the authorize URL shown once", f.seenNotify)
	}
	pasteHint := false
	for _, n := range f.seenNotify {
		pasteHint = pasteHint || strings.Contains(strings.ToLower(n), "paste")
	}
	if !pasteHint {
		t.Errorf("notices = %v, want a paste instruction", f.seenNotify)
	}
}

// TestConfigureLLM_BrowserLoopbackWinDrainsPastePrompt: when the loopback wins
// while the paste prompt is still reading, the wizard asks for Enter and waits
// for that read before its next prompt, so two reads never share the input.
func TestConfigureLLM_BrowserLoopbackWinDrainsPastePrompt(t *testing.T) {
	p := &drainPrompter{
		// The paste read is answered by Enter, as the notice asks.
		fakePrompter: &fakePrompter{t: t, blankSearches: true, selects: []int{providerIdx(t, llmprovider.ProviderGrok), 1, 0},
			inputs: []string{""}},
		inputStarted: make(chan struct{}),
		release:      make(chan struct{}),
	}
	stubBrowserLogin(t, func(ctx context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		if opts.InputCode == nil {
			t.Fatal("browser OAuth has no paste-code InputCode")
		}
		if err := opts.OpenURL("https://authorize.test"); err != nil {
			t.Fatalf("OpenURL() error = %v", err)
		}
		go func() { _, _ = opts.InputCode(ctx) }()
		<-p.inputStarted
		return testOAuthSession(provider), nil // the loopback won
	})
	res, err := ConfigureLLM(context.Background(), p, Options{TokenStore: newMemoryTokenStore()})
	if err != nil {
		t.Fatalf("ConfigureLLM() error = %v", err)
	}
	if res.Kind != CredOAuth || countContaining(p.seenNotify, "press Enter to continue") != 1 {
		t.Fatalf("Kind = %q, notices = %v; want OAuth and one press-Enter notice", res.Kind, p.seenNotify)
	}
}

// drainPrompter's Input blocks like a terminal read until the wizard asks for
// the Enter that finishes it, and its Select fails a prompt that starts while
// that read is pending.
type drainPrompter struct {
	*fakePrompter
	inputStarted chan struct{}
	release      chan struct{}
	releaseOnce  sync.Once
	reading      atomic.Bool
}

func (d *drainPrompter) Input(prompt, def string) (string, error) {
	v, err := d.fakePrompter.Input(prompt, def)
	if prompt != pasteCodePrompt {
		return v, err
	}
	d.reading.Store(true)
	close(d.inputStarted)
	<-d.release
	d.reading.Store(false)
	return v, err
}

func (d *drainPrompter) Notify(level Level, format string, args ...any) {
	d.fakePrompter.Notify(level, format, args...)
	if strings.Contains(fmt.Sprintf(format, args...), "press Enter") {
		d.releaseOnce.Do(func() { close(d.release) })
	}
}

func (d *drainPrompter) Select(title string, choices []Choice, defaultIdx int) (int, error) {
	if d.reading.Load() {
		d.t.Errorf("Select(%q) started while the paste prompt was still reading", title)
	}
	return d.fakePrompter.Select(title, choices, defaultIdx)
}

func stubBrowserLogin(t *testing.T, login func(context.Context, llmprovider.ProviderID, auth.OAuthFlowOptions) (*auth.OAuthSession, error)) {
	t.Helper()
	original := loginBrowserOAuth
	t.Cleanup(func() { loginBrowserOAuth = original })
	loginBrowserOAuth = login
}

// TestConfigureLLM_NoTokenStoreOffersStorelessMethods (0020-MADR F51):
// without a TokenStore the caller cannot keep a session, so the menu offers
// only the methods that save none, in descriptor order.
func TestConfigureLLM_NoTokenStoreOffersStorelessMethods(t *testing.T) {
	for _, c := range []struct {
		provider llmprovider.ProviderID
		want     []llmprovider.AuthMethodID
	}{
		{llmprovider.ProviderOpenAI, []llmprovider.AuthMethodID{llmprovider.AuthAPIKey, llmprovider.AuthImportVendorCLI}},
		{llmprovider.ProviderGrok, []llmprovider.AuthMethodID{llmprovider.AuthAPIKey, llmprovider.AuthTokenStdin,
			llmprovider.AuthImportVendorCLI}},
	} {
		t.Run(string(c.provider), func(t *testing.T) {
			f := &fakePrompter{
				t:       t,
				selects: []int{providerIdx(t, c.provider), 0},
				secrets: []string{"sk-test-0123456789"},
			}
			_, _ = ConfigureLLM(context.Background(), f, Options{})
			if len(f.seenSelect) < 2 || !strings.Contains(f.seenSelect[1], "authenticate") {
				t.Fatalf("Select titles = %q, want the method menu second", f.seenSelect)
			}
			var want []string
			for _, id := range c.want {
				want = append(want, methodLabel(t, c.provider, id))
			}
			if got := labels(f.seenSelectItems[1]); !slices.Equal(got, want) {
				t.Errorf("menu = %q, want %q", got, want)
			}
			if len(f.seenSecret) != 1 {
				t.Fatalf("Secret prompts = %d, want the API-key prompt", len(f.seenSecret))
			}
		})
	}
}

// TestConfigureLLM_TokenStoreOffersAllMethods: with a TokenStore the menu lists
// every descriptor method, in descriptor order.
func TestConfigureLLM_TokenStoreOffersAllMethods(t *testing.T) {
	for _, provider := range []llmprovider.ProviderID{llmprovider.ProviderOpenAI, llmprovider.ProviderGrok} {
		t.Run(string(provider), func(t *testing.T) {
			d, ok := providers.Default().Descriptor(provider)
			if !ok {
				t.Fatalf("no descriptor for %s", provider)
			}
			f := &fakePrompter{t: t, selects: []int{providerIdx(t, provider)}}
			_, _ = ConfigureLLM(context.Background(), f, Options{TokenStore: newMemoryTokenStore()})
			if len(f.seenSelect) < 2 || !strings.Contains(f.seenSelect[1], "authenticate") {
				t.Fatalf("Select titles = %q, want the method menu second", f.seenSelect)
			}
			got := f.seenSelectItems[1]
			if len(got) != len(d.AuthMethods) {
				t.Fatalf("menu has %d methods, want %d", len(got), len(d.AuthMethods))
			}
			for i, m := range d.AuthMethods {
				if got[i].Label != m.Label {
					t.Errorf("menu[%d] = %q, want %q", i, got[i].Label, m.Label)
				}
			}
		})
	}
}
