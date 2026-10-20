package wizard

import (
	"context"
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// errLogin0049 is a sign-in's own failure, which ConfigureLLM returns as the
// login gave it.
var errLogin0049 = errors.New("oauth: the sign-in was abandoned")

// asGiven0049 fails t unless err is the login's error, unwrapped.
func asGiven0049(t *testing.T, what string, err error) {
	t.Helper()
	if !errors.Is(err, errLogin0049) || err.Error() != errLogin0049.Error() {
		t.Errorf("%s: ConfigureLLM = %v; want the login's error as it gave it", what, err)
	}
}

// TestConfigureLLM_BrowserLoginErrorBare_0049 (0049-MADR D4, F6): a failed
// browser sign-in's error is returned as the login gave it.
func TestConfigureLLM_BrowserLoginErrorBare_0049(t *testing.T) {
	original := loginBrowserOAuth
	t.Cleanup(func() { loginBrowserOAuth = original })
	loginBrowserOAuth = func(context.Context, llmprovider.ProviderID, auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		return nil, errLogin0049
	}
	// Grok, then its browser method.
	f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderGrok), 1}})
	_, err := ConfigureLLM(context.Background(), f, Options{TokenStore: newMemoryTokenStore()})
	asGiven0049(t, "browser", err)
}

// TestConfigureLLM_PlanLoginErrorBare_0049 (0049-MADR D4, F6): a failed
// ChatGPT plan sign-in's error is returned as the login gave it.
func TestConfigureLLM_PlanLoginErrorBare_0049(t *testing.T) {
	calls := stubPlanLogin(t, func(context.Context, auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		return nil, errLogin0049
	})
	f := newFake(t, fakePrompter{
		selects:  []int{providerIdx(t, llmprovider.ProviderOpenAI), planMethodIdx(t)},
		confirms: []bool{true},
	})
	_, err := ConfigureLLM(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), AgentName: planTestApp})
	if *calls != 1 {
		t.Fatalf("the plan login ran %d times; want once", *calls)
	}
	asGiven0049(t, "ChatGPT plan", err)
}
