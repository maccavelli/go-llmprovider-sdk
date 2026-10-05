package wizard

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/claude"
)

// providerIdx returns the index of a provider in the default registry's menu
// order, so tests script menu positions without hard-coding them.
func providerIdx(t *testing.T, id llmprovider.ProviderID) int {
	t.Helper()
	for i, d := range providers.Default().Descriptors() {
		if d.ID == id {
			return i
		}
	}
	t.Fatalf("provider %q not in providers.Default()", id)
	return -1
}

// envOf is a LookupEnv over vals, so a test never reads the process
// environment (0015-MADR D9).
func envOf(vals map[string]string) func(string) string {
	return func(k string) string { return vals[k] }
}

const testKey = "sk-super-secret-key-1234"

func TestConfigureLLM_EnvKeyPrecedence(t *testing.T) {
	f := &fakePrompter{
		t:             t,
		blankSearches: true,
		selects:       []int{providerIdx(t, llmprovider.ProviderClaude), 0},
		confirms:      []bool{true}, // yes, use the env key
	}
	res, err := ConfigureLLM(context.Background(), f, Options{AllowEnv: true,
		LookupEnv: envOf(map[string]string{"ANTHROPIC_API_KEY": testKey})})
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if res.APIKey != testKey {
		t.Errorf("APIKey = %q, want the env value", res.APIKey)
	}
	if len(f.seenSecret) != 0 {
		t.Errorf("Secret must not be prompted when the env key is accepted: %v", f.seenSecret)
	}
}

func TestConfigureLLM_KeepExisting(t *testing.T) {
	f := &fakePrompter{
		t:             t,
		blankSearches: true,
		selects:       []int{providerIdx(t, llmprovider.ProviderClaude), 0},
		confirms:      []bool{true}, // keep existing
	}
	res, err := ConfigureLLM(context.Background(), f, Options{
		Existing: Result{Provider: llmprovider.ProviderClaude, APIKey: "existing-key-abcd"},
	})
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if res.APIKey != "existing-key-abcd" {
		t.Errorf("APIKey = %q, want the existing key", res.APIKey)
	}
	if len(f.seenSecret) != 0 {
		t.Error("Secret must not be prompted when the existing key is kept")
	}
}

func TestConfigureLLM_PromptsWhenNothingAvailable(t *testing.T) {
	f := &fakePrompter{
		t:             t,
		blankSearches: true,
		selects:       []int{providerIdx(t, llmprovider.ProviderClaude), 0},
		secrets:       []string{testKey},
	}
	res, err := ConfigureLLM(context.Background(), f, Options{AllowEnv: true})
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if res.APIKey != testKey {
		t.Errorf("APIKey = %q", res.APIKey)
	}
	if len(f.seenSecret) != 1 {
		t.Errorf("Secret prompted %d times, want 1", len(f.seenSecret))
	}
}

// TestConfigureLLM_LocalProviderSkipsKey: Ollama needs no credential, which is
// why ProviderDescriptor has RequiresAPIKey.
func TestConfigureLLM_LocalProviderSkipsKey(t *testing.T) {
	f := &fakePrompter{
		t:       t,
		selects: []int{providerIdx(t, llmprovider.ProviderOllama)},
		// Two inputs: the endpoint, then the manual model id (Ollama has no
		// static catalog, so the flow falls through to manual entry).
		inputs: []string{"http://localhost:11434", "llama3.2:latest"},
		// Unreachable Ollama (CI) asks to try a different endpoint. A running
		// local daemon never issues that Confirm; leftover scripted answers
		// are ignored.
		confirms: []bool{false},
	}
	res, err := ConfigureLLM(context.Background(), f, Options{AllowEnv: true})
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if res.APIKey != "" {
		t.Errorf("APIKey = %q, want empty for a local provider", res.APIKey)
	}
	if res.Kind != CredNone {
		t.Errorf("Kind = %q, want CredNone for a local provider", res.Kind)
	}
	if len(f.seenSecret) != 0 {
		t.Error("Secret must never be prompted for a provider that needs no key")
	}
	// Ollama has no static catalog, so the flow must fall through to a manual
	// model entry rather than dead-ending.
	if res.Model != "llama3.2:latest" {
		t.Errorf("Model = %q, want the manually entered id", res.Model)
	}
}

// TestConfigureLLM_NoModelsAndNoneEnteredErrors: a Result with an empty Model
// cannot generate anything, so the flow must fail rather than return it.
func TestConfigureLLM_NoModelsAndNoneEnteredErrors(t *testing.T) {
	f := &fakePrompter{
		t:        t,
		selects:  []int{providerIdx(t, llmprovider.ProviderOllama)},
		inputs:   []string{"http://localhost:11434", ""},
		confirms: []bool{false},
	}
	if _, err := ConfigureLLM(context.Background(), f, Options{}); err == nil {
		t.Error("expected an error when no model is available and none is entered")
	}
}

func TestConfigureLLM_EmptyDiscoveryFallsBackToStatic(t *testing.T) {
	f := &fakePrompter{
		t:             t,
		blankSearches: true,
		selects:       []int{providerIdx(t, llmprovider.ProviderClaude), 0},
		secrets:       []string{testKey},
	}
	// The listing's client refuses every request, so the listing fails
	// without reaching the network, and the static catalog must be offered
	// instead of the wizard dead-ending.
	var listed atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		listed.Add(1)
		return nil, errors.New("refused by the test")
	})}
	res, err := ConfigureLLM(context.Background(), f, Options{
		Discover:      true,
		DiscoverLimit: 2 * time.Second,
		HTTPClient:    client,
	})
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if listed.Load() == 0 {
		t.Fatal("the listing never used Options.HTTPClient")
	}
	static := catalog.Static(llmprovider.ProviderClaude)
	if len(static) == 0 || res.Model != static[0] {
		t.Errorf("Model = %q, want the first static model %v", res.Model, static)
	}
	if countContaining(f.seenNotify, "refused by the test") != 1 {
		t.Errorf("notices = %v, want the listing failure reported once", f.seenNotify)
	}
}

func TestConfigureLLM_Fallbacks(t *testing.T) {
	f := &fakePrompter{
		t:             t,
		blankSearches: true,
		selects:       []int{providerIdx(t, llmprovider.ProviderClaude), 0},
		secrets:       []string{testKey},
		multiSelects:  [][]int{{0, 1}},
	}
	res, err := ConfigureLLM(context.Background(), f, Options{NeedFallbacks: true})
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if len(res.Fallbacks) != 2 {
		t.Fatalf("Fallbacks = %v, want 2", res.Fallbacks)
	}
	for _, fb := range res.Fallbacks {
		if fb == res.Model {
			t.Errorf("fallbacks must exclude the primary model %q: %v", res.Model, res.Fallbacks)
		}
	}
}

// TestConfigureLLM_MaskedKeyNeverPrintsSecret is the guarantee that a
// credential cannot leak through a prompt. Every string the user could have
// seen is checked against the raw key.
func TestConfigureLLM_MaskedKeyNeverPrintsSecret(t *testing.T) {
	f := &fakePrompter{
		t:             t,
		blankSearches: true,
		selects:       []int{providerIdx(t, llmprovider.ProviderClaude), 0},
		confirms:      []bool{true},
	}
	if _, err := ConfigureLLM(context.Background(), f, Options{AllowEnv: true,
		LookupEnv: envOf(map[string]string{"ANTHROPIC_API_KEY": testKey})}); err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	var sawMask bool
	for _, s := range f.allText {
		if strings.Contains(s, testKey) {
			t.Errorf("a raw credential reached the user: %q", s)
		}
		if strings.Contains(s, "1234") && strings.Contains(s, "•") {
			sawMask = true
		}
	}
	if !sawMask {
		t.Error("expected the masked key to be shown so the user can identify it")
	}
}

// TestConfigureLLM_OffersEveryDescriptor: with no filter, the provider menu is
// exactly the canonical descriptor list. This is the property that keeps every
// wizard current when this module adds a provider.
func TestConfigureLLM_OffersEveryDescriptor(t *testing.T) {
	f := &fakePrompter{
		t:             t,
		blankSearches: true,
		selects:       []int{0, 0},
		secrets:       []string{testKey},
	}
	if _, err := ConfigureLLM(context.Background(), f, Options{}); err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if len(f.seenSelectItems) == 0 {
		t.Fatal("no Select was made")
	}
	got := len(f.seenSelectItems[0])
	want := len(providers.Default().Descriptors())
	if got != want {
		t.Errorf("provider menu offered %d choices, want all %d descriptors", got, want)
	}
}

// TestConfigureLLM_UsesTheRegistry: a caller's Registry is the menu, in its own
// order; nil is every built-in provider (0015-PLAN S8).
func TestConfigureLLM_UsesTheRegistry(t *testing.T) {
	reg := llmprovider.NewRegistry()
	if err := reg.Register(claude.Descriptor(), claude.New); err != nil {
		t.Fatal(err)
	}
	f := &fakePrompter{t: t, blankSearches: true, selects: []int{0, 0}, secrets: []string{testKey}}
	res, err := ConfigureLLM(context.Background(), f, Options{Registry: reg})
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if len(f.seenSelectItems) == 0 || len(f.seenSelectItems[0]) != 1 {
		t.Fatalf("provider menu = %v, want the registry's one provider", f.seenSelectItems)
	}
	if res.Provider != llmprovider.ProviderClaude {
		t.Errorf("Provider = %q, want claude", res.Provider)
	}
}

func TestConfigureLLM_ProviderFilter(t *testing.T) {
	f := &fakePrompter{t: t, blankSearches: true, selects: []int{0, 0, 0}, secrets: []string{testKey}}
	res, err := ConfigureLLM(context.Background(), f, Options{
		Providers: []llmprovider.ProviderID{llmprovider.ProviderGrok},
	})
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if res.Provider != llmprovider.ProviderGrok {
		t.Errorf("Provider = %q, want grok", res.Provider)
	}
	if n := len(f.seenSelectItems[0]); n != 1 {
		t.Errorf("filtered menu offered %d choices, want 1", n)
	}

	if _, err := ConfigureLLM(context.Background(), &fakePrompter{t: t},
		Options{Providers: []llmprovider.ProviderID{"nonexistent"}}); err == nil {
		t.Error("expected an error when no requested provider exists")
	}
}

// TestConfigureLLM_OtherModelEscapeHatch: a curated catalog and a live listing
// can both lag a newly released model, so the menu always ends with a manual
// entry. prepare-commit-msg's wizard had this before the migration.
func TestConfigureLLM_OtherModelEscapeHatch(t *testing.T) {
	static := catalog.Static(llmprovider.ProviderClaude)
	f := &fakePrompter{
		t: t,
		// provider, then the trailing "Other" entry
		selects: []int{providerIdx(t, llmprovider.ProviderClaude), len(static)},
		secrets: []string{testKey},
		// a blank search (MADR 0007 §4), then the manual model id
		inputs: []string{"", "my-custom-model"},
	}
	res, err := ConfigureLLM(context.Background(), f, Options{})
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if res.Model != "my-custom-model" {
		t.Errorf("Model = %q, want the manually entered id", res.Model)
	}
	last := f.seenSelectItems[1][len(f.seenSelectItems[1])-1].Label
	if last != otherModelLabel {
		t.Errorf("model menu must end with %q, got %q", otherModelLabel, last)
	}
}

// TestConfigureLLM_InjectedLookupEnv: consumers drive the env-key branch
// deterministically in their own tests without touching the real environment.
func TestConfigureLLM_InjectedLookupEnv(t *testing.T) {
	f := &fakePrompter{
		t:             t,
		blankSearches: true,
		selects:       []int{providerIdx(t, llmprovider.ProviderGemini), 0},
		confirms:      []bool{true},
	}
	res, err := ConfigureLLM(context.Background(), f, Options{
		AllowEnv:  true,
		LookupEnv: func(k string) string { return map[string]string{"GEMINI_API_KEY": testKey}[k] },
	})
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if res.APIKey != testKey {
		t.Errorf("APIKey = %q, want the injected env value", res.APIKey)
	}
}

// TestConfigure_EmptyRecommendedOpensSearch (0021-MADR C13): a live listing
// with nothing recommended, here Kilo's tiers alone under the utility
// profile, keeps the live listing. A blank search says no model meets the
// profile and asks again, and a search finds a tier.
func TestConfigure_EmptyRecommendedOpensSearch(t *testing.T) {
	listing := `{"data":[` + kiloProfileEntry("kilo-auto/small", "-1", "-1", 0, 0) + "," +
		kiloProfileEntry("kilo-auto/balanced", "-1", "-1", 0, 0) + `]}`
	srv := zenServer(t, http.StatusOK, listing)
	f := &fakePrompter{
		t: t, selects: []int{providerIdx(t, llmprovider.ProviderKilo), 0},
		inputs: []string{srv.URL, "", "kilo-auto/small"}, secrets: []string{testKey},
	}
	res, err := ConfigureLLM(context.Background(), f, zenOptions())
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if res.Model != "kilo-auto/small" {
		t.Errorf("Model = %q, want kilo-auto/small, found by search", res.Model)
	}
	if countContaining(f.seenNotify, "meets the profile") != 1 {
		t.Errorf("notices = %v, want one that no model meets the profile", f.seenNotify)
	}
}

// TestResolveBaseURL_Validates (0021-MADR Z7): a remote provider's base URL
// must be http or https with a host, and carry no userinfo, query or
// fragment; plain http to a host that is not loopback asks first. A refused
// URL is asked for again. Kilo's base URL is resolved here too.
func TestResolveBaseURL_Validates(t *testing.T) {
	d := llmprovider.Descriptor{ID: llmprovider.ProviderKilo, Label: "Kilo", SupportsBaseURL: true}
	for _, c := range []struct {
		name     string
		inputs   []string
		confirms []bool
		want     string
	}{
		{"no scheme", []string{"api.example.com", "https://api.example.com"}, nil, "https://api.example.com"},
		{"userinfo", []string{"https://u:p@host", "https://host"}, nil, "https://host"},
		{"query", []string{"https://host/?q=1", "https://host"}, nil, "https://host"},
		{"fragment", []string{"https://host/#f", "https://host"}, nil, "https://host"},
		{"remote http refused", []string{"http://remote.example", "https://remote.example"}, []bool{false}, "https://remote.example"},
		{"remote http confirmed", []string{"http://remote.example"}, []bool{true}, "http://remote.example"},
		{"loopback http", []string{"http://localhost:11434"}, nil, "http://localhost:11434"},
		{"blank keeps the default", []string{""}, nil, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := &fakePrompter{t: t, inputs: c.inputs, confirms: c.confirms}
			got, err := resolveBaseURL(context.Background(), f, d, Options{})
			if err != nil || got != c.want {
				t.Fatalf("resolveBaseURL = %q, %v; want %q", got, err, c.want)
			}
			if len(f.inputs) != 0 || len(f.seenConfirm) != len(c.confirms) {
				t.Errorf("%d answers left and %d confirmations asked; want every answer used and %d asked",
					len(f.inputs), len(f.seenConfirm), len(c.confirms))
			}
		})
	}
}
