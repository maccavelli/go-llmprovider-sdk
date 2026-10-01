package providers

import (
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// Ported from llmprovider's descriptor_test.go (0015-PLAN S8, commit 1): the
// descriptors are each provider package's, and Default holds them.

// TestDescriptors_NoOAuthOnOtherProviders: only OpenAI and Grok offer OAuth,
// and Kilo only its device login (0017-MADR D2), which is not OAuth: its
// token never refreshes.
func TestDescriptors_NoOAuthOnOtherProviders(t *testing.T) {
	for _, descriptor := range Default().Descriptors() {
		if descriptor.ID == llmprovider.ProviderOpenAI || descriptor.ID == llmprovider.ProviderGrok {
			continue
		}
		for _, method := range descriptor.AuthMethods {
			if descriptor.ID == llmprovider.ProviderKilo && method.ID == llmprovider.AuthDeviceCode {
				continue
			}
			if method.ID == llmprovider.AuthBrowserOAuth || method.ID == llmprovider.AuthDeviceCode {
				t.Errorf("provider %q unexpectedly offers OAuth method %q", descriptor.ID, method.ID)
			}
		}
	}
}

// TestDescriptors_OpenAIAndGrokOfferOAuth pins both providers' methods, in
// menu order.
func TestDescriptors_OpenAIAndGrokOfferOAuth(t *testing.T) {
	want := map[llmprovider.ProviderID][]llmprovider.AuthMethod{
		llmprovider.ProviderOpenAI: {
			{
				ID:          llmprovider.AuthAPIKey,
				Label:       "OpenAI API key",
				Detail:      "Platform billing (`api.openai.com`)",
				Interactive: true,
				HeadlessOK:  true,
			},
			{
				ID:          llmprovider.AuthBrowserOAuth,
				Label:       "Sign in with ChatGPT",
				Detail:      "Plus/Pro/Business/Edu/Enterprise plan",
				Interactive: true,
			},
			{
				ID:          llmprovider.AuthDeviceCode,
				Label:       "Sign in with ChatGPT (device code)",
				Interactive: true,
				HeadlessOK:  true,
			},
			{
				ID:          llmprovider.AuthTokenStdin,
				Label:       "Paste a ChatGPT access token or API key",
				Interactive: true,
				HeadlessOK:  true,
			},
			{
				ID:          llmprovider.AuthImportVendorCLI,
				Label:       "Use the Codex CLI login (~/.codex/auth.json)",
				Interactive: true,
				HeadlessOK:  true,
			},
		},
		llmprovider.ProviderGrok: {
			{
				ID:          llmprovider.AuthAPIKey,
				Label:       "xAI API key",
				Detail:      "console.x.ai billing (`api.x.ai`)",
				Interactive: true,
				HeadlessOK:  true,
			},
			{
				ID:          llmprovider.AuthBrowserOAuth,
				Label:       "Sign in with xAI",
				Interactive: true,
			},
			{
				ID:          llmprovider.AuthDeviceCode,
				Label:       "Sign in with xAI (device code)",
				Interactive: true,
				HeadlessOK:  true,
			},
			{
				ID:          llmprovider.AuthTokenStdin,
				Label:       "Paste an xAI API key",
				Interactive: true,
				HeadlessOK:  true,
			},
			{
				ID:          llmprovider.AuthImportVendorCLI,
				Label:       "Use the Grok CLI login (~/.grok/auth.json)",
				Interactive: true,
				HeadlessOK:  true,
			},
		},
	}

	registry := Default()
	for provider, methods := range want {
		descriptor, ok := registry.Descriptor(provider)
		if !ok {
			t.Fatalf("Descriptor(%q) not found", provider)
		}
		if !reflect.DeepEqual(descriptor.AuthMethods, methods) {
			t.Errorf("%s AuthMethods = %#v, want %#v", provider, descriptor.AuthMethods, methods)
		}
	}
}

// TestDescriptors_CoverEveryRegisteredProvider is the load-bearing test of this
// design. If a provider can be constructed but has no descriptor, no wizard can
// offer it — which is exactly how Grok shipped in MADR 0003 and reached none of
// the three configuration wizards. Every id with a credential variable is in
// Default, and every descriptor that needs a key names its variable.
func TestDescriptors_CoverEveryRegisteredProvider(t *testing.T) {
	registry := Default()
	envVars := llmprovider.ProviderEnvVars()
	for id := range envVars {
		if _, ok := registry.Descriptor(llmprovider.ProviderID(id)); !ok {
			t.Errorf("provider %q has a credential variable but no descriptor in Default: no wizard can offer it", id)
		}
	}
	for _, d := range registry.Descriptors() {
		if d.RequiresAPIKey && d.EnvVar != envVars[string(d.ID)] {
			t.Errorf("%s: EnvVar = %q, want %q", d.ID, d.EnvVar, envVars[string(d.ID)])
		}
		if !reflect.DeepEqual(d.StaticModels, catalog.Static(string(d.ID))) {
			t.Errorf("%s: StaticModels = %v, want StaticModels()'s", d.ID, d.StaticModels)
		}
	}
	for _, id := range []llmprovider.ProviderID{llmprovider.ProviderOpenAI, llmprovider.ProviderGrok} {
		d, _ := registry.Descriptor(id)
		for _, method := range []llmprovider.AuthMethodID{llmprovider.AuthAPIKey, llmprovider.AuthBrowserOAuth} {
			found := false
			for _, m := range d.AuthMethods {
				found = found || m.ID == method
			}
			if !found {
				t.Errorf("descriptor %q does not offer required auth method %q", id, method)
			}
		}
	}
}

// TestDescriptors_MenuOrder: the wizard's menu keeps the order llmprovider's
// Descriptors() gave it: remote providers first, local last.
func TestDescriptors_MenuOrder(t *testing.T) {
	var got []llmprovider.ProviderID
	for _, d := range Default().Descriptors() {
		got = append(got, d.ID)
	}
	want := []llmprovider.ProviderID{llmprovider.ProviderGemini, llmprovider.ProviderOpenAI, llmprovider.ProviderClaude,
		llmprovider.ProviderGrok, llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo,
		llmprovider.ProviderHuggingFace, llmprovider.ProviderKilo, llmprovider.ProviderTogether, llmprovider.ProviderOllama}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("menu order = %v, want %v", got, want)
	}
}

// TestDescriptors_DefensiveCopy: a caller mutating a returned descriptor
// cannot corrupt the next caller's.
func TestDescriptors_DefensiveCopy(t *testing.T) {
	registry := Default()
	a := registry.Descriptors()
	a[0].Label = "MUTATED"
	a[1].AuthMethods[0].Label = "MUTATED"
	if len(a[0].StaticModels) > 0 {
		a[0].StaticModels[0] = "MUTATED"
	}
	b := registry.Descriptors()
	if b[0].Label == "MUTATED" || b[1].AuthMethods[0].Label == "MUTATED" ||
		(len(b[0].StaticModels) > 0 && b[0].StaticModels[0] == "MUTATED") {
		t.Errorf("a caller's change reached the registry: %+v", b[:2])
	}
}

// TestDescriptor_Lookup was TestDescriptorFor: Kilo's descriptor is found by
// id, and an unknown id misses.
func TestDescriptor_Lookup(t *testing.T) {
	registry := Default()
	d, ok := registry.Descriptor(llmprovider.ProviderKilo)
	if !ok {
		t.Fatal("Descriptor(kilo) not found")
	}
	if d.EnvVar != "KILO_API_KEY" || !d.SupportsBaseURL {
		t.Errorf("kilo descriptor = %+v", d)
	}
	if _, ok := registry.Descriptor("nope"); ok {
		t.Error("Descriptor should miss on an unknown id")
	}
}

// TestDescriptors_NoStaleModels is the direct regression for the bug that
// motivated 0005-MADR: mcp-server-magictools recommended gemini-2.0-flash while
// models_catalog.go documents the 2.0 and 1.5 families as shut down.
func TestDescriptors_NoStaleModels(t *testing.T) {
	for _, d := range Default().Descriptors() {
		for _, m := range d.StaticModels {
			for _, dead := range []string{"gemini-2.0-", "gemini-1.5-"} {
				if strings.Contains(m, dead) {
					t.Errorf("%s offers %q, a shut-down model family (models_catalog.go:26)", d.ID, m)
				}
			}
		}
	}
}
