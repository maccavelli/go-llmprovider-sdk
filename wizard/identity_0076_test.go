package wizard

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// probe0076 is the Identity the caller gives the wizard.
var probe0076 = auth.Identity{Name: "probe-app", Version: "1.2.3"}

// stubDevice0076 replaces the device sign-in with one that records the
// options it was given and returns a session for provider.
func stubDevice0076(t *testing.T, got *[]auth.OAuthFlowOptions) {
	t.Helper()
	original := deviceLogin
	t.Cleanup(func() { deviceLogin = original })
	deviceLogin = func(_ context.Context, provider llmprovider.ProviderID, flow auth.OAuthFlowOptions, _ func(DeviceCode) error) (*auth.OAuthSession, error) {
		*got = append(*got, flow)
		if provider == llmprovider.ProviderKilo {
			return &auth.OAuthSession{Provider: provider, Access: kiloWizardToken}, nil
		}
		return testOAuthSession(provider), nil
	}
}

// TestWizardIdentity_0076 (0076-MADR D4): Options.Identity reaches every
// sign-in the wizard runs, the browser's, the device code's, Kilo's as a key
// included, and the ChatGPT plan's; the browser and device sign-ins are given
// no client id and no issuer, as before.
func TestWizardIdentity_0076(t *testing.T) {
	t.Run("browser", func(t *testing.T) {
		var got []auth.OAuthFlowOptions
		stubBrowserLogin(t, func(_ context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
			got = append(got, opts)
			return testOAuthSession(provider), nil
		})
		f := newFake(t, fakePrompter{selects: []int{grokBrowserIdx}})
		if _, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), NoEndpointPrompt: true,
			NoPastePrompt: true, Identity: probe0076}, llmprovider.ProviderGrok); err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Identity != probe0076 || got[0].ClientID != "" || got[0].Issuer != "" {
			t.Errorf("browser sign-ins given %+v; want one, with %+v and no client id or issuer", got, probe0076)
		}
	})
	t.Run("device", func(t *testing.T) {
		var got []auth.OAuthFlowOptions
		stubDevice0076(t, &got)
		f := newFake(t, fakePrompter{selects: []int{grokDeviceIdx}})
		if _, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), NoEndpointPrompt: true,
			Identity: probe0076}, llmprovider.ProviderGrok); err != nil {
			t.Fatal(err)
		}
		k := newFake(t, fakePrompter{inputs: []string{acceptDefault}, selects: []int{1}})
		if _, _, err := ResolveCredential(context.Background(), k, Options{TokenStore: newMemoryTokenStore(), KiloDeviceKey: true,
			Identity: probe0076}, llmprovider.ProviderKilo); err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("device sign-ins given %+v; want Grok's and Kilo's", got)
		}
		for _, flow := range got {
			if flow.Identity != probe0076 || flow.ClientID != "" || flow.Issuer != "" {
				t.Errorf("device sign-in given %+v; want %+v and no client id or issuer", flow, probe0076)
			}
		}
	})
	t.Run("plan", func(t *testing.T) {
		var got []auth.ChatGPTPlanOptions
		stubPlanLogin(t, func(_ context.Context, opts auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
			got = append(got, opts)
			return planSessionFor("oaiapp_0076"), nil
		})
		f := newFake(t, fakePrompter{selects: []int{planMethodIdx(t)}, confirms: []bool{true}})
		if _, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), AgentName: planTestApp,
			NoPastePrompt: true, Identity: probe0076}, llmprovider.ProviderOpenAI); err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Identity != probe0076 {
			t.Errorf("plan sign-ins given %+v; want one, with %+v", got, probe0076)
		}
	})
}
