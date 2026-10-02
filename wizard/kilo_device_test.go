package wizard

import (
	"context"
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

const kiloWizardToken = "kilo-device-token-0123456789"

// stubKilo replaces the Kilo device login and profile for one test.
func stubKilo(t *testing.T, account llmprovider.KiloAccount, profileErr error) {
	t.Helper()
	originalDevice, originalProfile := loginDeviceOAuth, kiloProfile
	t.Cleanup(func() { loginDeviceOAuth, kiloProfile = originalDevice, originalProfile })
	loginDeviceOAuth = func(_ context.Context, provider llmprovider.ProviderID, opts llmprovider.OAuthFlowOptions) (*llmprovider.OAuthSession, error) {
		if provider != llmprovider.ProviderKilo {
			t.Errorf("device login for %q", provider)
		}
		opts.NotifyDevice("https://app.kilo.ai/device", "KILO-1234")
		return &llmprovider.OAuthSession{Provider: llmprovider.ProviderKilo, Access: kiloWizardToken}, nil
	}
	kiloProfile = func(_ context.Context, token string, _ ...llmprovider.Option) (llmprovider.KiloAccount, error) {
		if token != kiloWizardToken {
			t.Errorf("profile read with %q", token)
		}
		return account, profileErr
	}
}

// TestConfigureLLM_KiloDeviceLogin (0017-MADR D2): the device login's token
// is saved to the store and returned as the Kilo API key; an account with
// organizations chooses one, or the personal account.
func TestConfigureLLM_KiloDeviceLogin(t *testing.T) {
	orgs := llmprovider.KiloAccount{HasPersonalAccount: true, SelectedOrganizationID: "org-2",
		Organizations: []llmprovider.KiloOrganization{{ID: "org-1", Name: "Acme"}, {ID: "org-2", Name: "Beta"}}}
	for _, tc := range []struct {
		name       string
		account    llmprovider.KiloAccount
		profileErr error
		selects    []int
		wantOrg    string
	}{
		{"personal only", llmprovider.KiloAccount{HasPersonalAccount: true}, nil, []int{1, 0}, ""},
		{"an organization", orgs, nil, []int{1, 1, 0}, "org-1"},
		{"the personal account", orgs, nil, []int{1, 0, 0}, ""},
		{"profile unreadable", llmprovider.KiloAccount{}, errors.New("down"), []int{1, 0}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stubKilo(t, tc.account, tc.profileErr)
			store := newMemoryTokenStore()
			f := &fakePrompter{t: t, selects: append([]int{providerIdx(t, llmprovider.ProviderKilo)}, tc.selects...)}
			res, err := ConfigureLLM(context.Background(), f, Options{TokenStore: store})
			if err != nil {
				t.Fatal(err)
			}
			saved := store.sessions[llmprovider.ProviderKilo]
			if res.Kind != CredAPIKey || res.APIKey != kiloWizardToken || res.Organization != tc.wantOrg {
				t.Errorf("Result kind %q key-set %v organization %q; want api_key, the token, %q",
					res.Kind, res.APIKey == kiloWizardToken, res.Organization, tc.wantOrg)
			}
			if saved == nil || saved.Access != kiloWizardToken || saved.Refresh != "" {
				t.Errorf("stored session %v, want the token with no refresh", saved)
			}
		})
	}
}
