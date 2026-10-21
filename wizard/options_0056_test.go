package wizard

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// 0056-MADR D2 and D4: Options.KiloDeviceKey, SkipMethods and
// NoEndpointPrompt.

// kiloAccount0056 has organizations, so that a Kilo sign-in that read the
// profile would ask which to use.
var kiloAccount0056 = auth.KiloAccount{HasPersonalAccount: true,
	Organizations: []auth.KiloOrganization{{ID: "org-1", Name: "Acme"}}}

// countKiloProfiles counts the profile reads after stubKilo, whose cleanup
// restores the original.
func countKiloProfiles(t *testing.T) *int {
	t.Helper()
	reads, stubbed := 0, kiloProfile
	kiloProfile = func(ctx context.Context, token string, opts ...llmprovider.Option) (auth.KiloAccount, error) {
		reads++
		return stubbed(ctx, token, opts...)
	}
	return &reads
}

// menuLabels is the labels of the Select call titled title.
func menuLabels(f *fakePrompter, title string) []string {
	for i, seen := range f.seenSelect {
		if seen == title {
			var labels []string
			for _, c := range f.seenSelectItems[i] {
				labels = append(labels, c.Label)
			}
			return labels
		}
	}
	return nil
}

func descriptor0056(t *testing.T, id llmprovider.ProviderID) llmprovider.Descriptor {
	t.Helper()
	d, ok := (Options{}).registry().Descriptor(id)
	if !ok {
		t.Fatalf("no descriptor for %q", id)
	}
	return d
}

// TestKiloDeviceKey_ReturnsTheKey_0056: with a store, the device sign-in in
// key mode is CredAPIKey with the token, saves nothing, reads no profile and
// asks no organization.
func TestKiloDeviceKey_ReturnsTheKey_0056(t *testing.T) {
	stubKilo(t, kiloAccount0056, nil)
	profiles := countKiloProfiles(t)
	store := newMemoryTokenStore()
	f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, selects: []int{1}})
	res, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: store, KiloDeviceKey: true},
		llmprovider.ProviderKilo)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != CredAPIKey || res.APIKey != kiloWizardToken || res.Organization != "" {
		t.Errorf("Result kind %q, key %q, organization %q; want the token as the key, no organization",
			res.Kind, res.APIKey, res.Organization)
	}
	if store.saves != 0 || *profiles != 0 || slices.Contains(f.seenSelect, "Use Kilo as:") {
		t.Errorf("%d saves, %d profile reads, selects %q; want none, none, and no organization question",
			store.saves, *profiles, f.seenSelect)
	}
}

// TestKiloDeviceKey_NoStoreNeeded_0056: in key mode the device sign-in is
// offered with no store; without it, only the API key is.
func TestKiloDeviceKey_NoStoreNeeded_0056(t *testing.T) {
	stubKilo(t, kiloAccount0056, nil)
	f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, selects: []int{1}})
	res, _, err := ResolveCredential(context.Background(), f, Options{KiloDeviceKey: true}, llmprovider.ProviderKilo)
	if err != nil {
		t.Fatal(err)
	}
	labels := menuLabels(f, "Choose how to authenticate with "+descriptor0056(t, llmprovider.ProviderKilo).Label+":")
	if !slices.Contains(labels, "Sign in with Kilo (device code)") || res.APIKey != kiloWizardToken {
		t.Errorf("menu %q, key %q; want the device sign-in offered with no store, and its token", labels, res.APIKey)
	}
	plain := newFake(t, fakePrompter{inputs: []string{acceptDefault}, secrets: []string{"kilo-key-0056"}})
	if _, _, err := ResolveCredential(context.Background(), plain, Options{}, llmprovider.ProviderKilo); err != nil {
		t.Fatal(err)
	}
	if len(plain.seenSelect) != 0 {
		t.Errorf("selects %q without key mode; want none, the API key alone", plain.seenSelect)
	}
}

// TestKiloDeviceKey_Refused_0056: a device token the listing refuses ends
// the run, saying to sign in again, and matches ErrAuthFailure.
func TestKiloDeviceKey_Refused_0056(t *testing.T) {
	stubKilo(t, kiloAccount0056, nil)
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		status, body := http.StatusNotFound, `{}`
		if strings.HasSuffix(r.URL.Path, "/models") {
			status, body = http.StatusUnauthorized, `{"error":{"message":"invalid token"}}`
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, selects: []int{1}})
	_, _, err := ResolveCredential(context.Background(), f, Options{KiloDeviceKey: true, Discover: true, HTTPClient: client},
		llmprovider.ProviderKilo)
	if !errors.Is(err, llmprovider.ErrAuthFailure) || err == nil || !strings.Contains(err.Error(), "refused the device sign-in") {
		t.Errorf("error %v; want ErrAuthFailure, refused the device sign-in", err)
	}
}

// TestKiloDeviceKey_KeptNextRun_0056: the next run offers the device token,
// now the existing key, as any API key, and keeps it.
func TestKiloDeviceKey_KeptNextRun_0056(t *testing.T) {
	stubKilo(t, kiloAccount0056, nil)
	f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, selects: []int{0}, confirms: []bool{true}})
	res, _, err := ResolveCredential(context.Background(), f, Options{KiloDeviceKey: true,
		Existing: Result{Provider: llmprovider.ProviderKilo, Kind: CredAPIKey, APIKey: kiloWizardToken}}, llmprovider.ProviderKilo)
	if err != nil {
		t.Fatal(err)
	}
	if res.APIKey != kiloWizardToken || len(f.seenConfirm) != 1 || !strings.Contains(f.seenConfirm[0], "Keep the existing key") {
		t.Errorf("key %q, confirms %q; want the existing key offered and kept", res.APIKey, f.seenConfirm)
	}
}

// TestSkipMethods_LeavesThemOut_0056: skipped methods are not in the menu.
func TestSkipMethods_LeavesThemOut_0056(t *testing.T) {
	f := newFake(t, fakePrompter{selects: []int{0}, secrets: []string{"sk-proj-0056"}})
	_, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: newMemoryTokenStore(),
		SkipMethods: []llmprovider.AuthMethodID{llmprovider.AuthBrowserOAuth, llmprovider.AuthDeviceCode}}, llmprovider.ProviderOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	labels := menuLabels(f, "Choose how to authenticate with OpenAI:")
	if !slices.Contains(labels, "OpenAI API key") || slices.Contains(labels, "Sign in with ChatGPT") ||
		slices.Contains(labels, "Sign in with ChatGPT (device code)") {
		t.Errorf("menu %q; want the API key, and neither ChatGPT sign-in", labels)
	}
}

// TestSkipMethods_NoneLeft_0056: skipping a provider's only method, here
// Claude's implicit API key, is a plain error, with no prompt.
func TestSkipMethods_NoneLeft_0056(t *testing.T) {
	f := newFake(t, fakePrompter{})
	_, _, err := ResolveCredential(context.Background(), f, Options{SkipMethods: []llmprovider.AuthMethodID{llmprovider.AuthAPIKey}},
		llmprovider.ProviderClaude)
	if err == nil || !strings.Contains(err.Error(), "leaves no sign-in method for Claude") ||
		errors.Is(err, llmprovider.ErrInvalidProvider) || len(f.allText) != 0 {
		t.Errorf("error %v, shown %q; want the plain error, and no prompt", err, f.allText)
	}
}

// TestNoEndpointPrompt_0056: no endpoint is asked; the base URL is
// Existing's, else the default; a local one is checked, and one that cannot
// be reached is an error.
func TestNoEndpointPrompt_0056(t *testing.T) {
	kilo := descriptor0056(t, llmprovider.ProviderKilo)
	for _, tc := range []struct {
		name     string
		existing string
		want     string
	}{
		{"Kilo's default", "", kilo.DefaultBaseURL},
		{"Kilo's existing", "https://kilo.example.test/api", "https://kilo.example.test/api"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake(t, fakePrompter{secrets: []string{"kilo-key-0056"}})
			o := Options{NoEndpointPrompt: true}
			if tc.existing != "" {
				o.Existing = Result{Provider: llmprovider.ProviderKilo, BaseURL: tc.existing}
			}
			res, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
			if err != nil {
				t.Fatal(err)
			}
			if len(f.seenInput) != 0 || res.BaseURL != tc.want {
				t.Errorf("inputs %q, base URL %q; want no endpoint asked, %q", f.seenInput, res.BaseURL, tc.want)
			}
		})
	}
	t.Run("Ollama reachable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/version" {
				_, _ = io.WriteString(w, `{"version": "0.1.24"}`)
				return
			}
			http.NotFound(w, r)
		}))
		t.Cleanup(srv.Close)
		f := newFake(t, fakePrompter{})
		res, _, err := ResolveCredential(context.Background(), f, Options{NoEndpointPrompt: true,
			Existing: Result{Provider: llmprovider.ProviderOllama, BaseURL: srv.URL}}, llmprovider.ProviderOllama)
		if err != nil {
			t.Fatal(err)
		}
		if res.BaseURL != srv.URL || len(f.allText) != 0 {
			t.Errorf("base URL %q, shown %q; want the existing endpoint, nothing asked", res.BaseURL, f.allText)
		}
	})
	t.Run("Ollama unreachable", func(t *testing.T) {
		f := newFake(t, fakePrompter{})
		_, _, err := ResolveCredential(context.Background(), f, Options{NoEndpointPrompt: true, HTTPClient: nettest.Refusing(),
			Existing: Result{Provider: llmprovider.ProviderOllama, BaseURL: nettest.RefusedURL}}, llmprovider.ProviderOllama)
		if err == nil || !strings.Contains(err.Error(), "cannot reach") || len(f.allText) != 0 {
			t.Errorf("error %v, shown %q; want cannot reach, and no prompt", err, f.allText)
		}
	})
}
