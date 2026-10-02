package wizard

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestConfigureLLM_KiloListsTheChosenOrganization (0015-PLAN S8b): the
// organization chosen after a Kilo device login reaches the listing through
// the wizard's one option list, in For(kilo, …).
func TestConfigureLLM_KiloListsTheChosenOrganization(t *testing.T) {
	stubKilo(t, llmprovider.KiloAccount{HasPersonalAccount: true,
		Organizations: []llmprovider.KiloOrganization{{ID: "org-1", Name: "Acme"}}}, nil)
	var mu sync.Mutex
	var listings []string
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		status, body := http.StatusNotFound, ""
		if strings.HasSuffix(r.URL.Path, "/models") {
			mu.Lock()
			listings = append(listings, r.URL.Path+" org="+r.Header.Get("X-KILOCODE-ORGANIZATIONID"))
			mu.Unlock()
			status, body = http.StatusOK, `{"data":[]}`
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	f := &fakePrompter{t: t, selects: []int{providerIdx(t, llmprovider.ProviderKilo), 1, 1, 0}}
	res, err := ConfigureLLM(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), Discover: true, HTTPClient: client})
	if err != nil {
		t.Fatalf("ConfigureLLM() error = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if res.Organization != "org-1" || len(listings) != 1 || listings[0] != "/api/organizations/org-1/models org=org-1" {
		t.Fatalf("organization %q, listings %q; want org-1's catalog with its header", res.Organization, listings)
	}
}

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

// TestConfigureLLM_KiloDeviceLogin (0017-MADR D2; 0016-MADR D11, A7): the
// device login's token is saved to the store, its only copy, and Result is an
// oauth credential with no token; an account with organizations chooses one,
// or the personal account.
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
			if res.Kind != CredOAuth || res.Organization != tc.wantOrg {
				t.Errorf("Result kind %q organization %q; want oauth, %q", res.Kind, res.Organization, tc.wantOrg)
			}
			if fields := resultFieldsHolding(res, kiloWizardToken); len(fields) != 0 {
				t.Errorf("Result fields %v hold the token; the store is the only copy", fields)
			}
			if saved == nil || saved.Access != kiloWizardToken || saved.Refresh != "" {
				t.Errorf("stored session %v, want the token with no refresh", saved)
			}
		})
	}
}
