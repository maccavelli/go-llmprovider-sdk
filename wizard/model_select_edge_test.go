package wizard

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// TestConfigureLLM_FallbackPicksDeduped pins MADR 0013 C1: a repeated index
// in a fallback MultiSelect adds that model once.
func TestConfigureLLM_FallbackPicksDeduped(t *testing.T) {
	static := catalog.Static(llmprovider.ProviderClaude)
	f := newFake(t, fakePrompter{
		selects: []int{providerIdx(t, llmprovider.ProviderClaude), 0}, secrets: []string{testKey},
		inputs: []string{"", ""}, multiSelects: [][]int{{0, 0, 1}},
	})
	res, err := ConfigureLLM(context.Background(), f, Options{NeedFallbacks: true})
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if want := []string{static[1], static[2]}; !slices.Equal(res.Fallbacks, want) {
		t.Errorf("Fallbacks = %q, want %q", res.Fallbacks, want)
	}
}

// TestMultiSelect_RepeatedIndexCountsOnce pins MADR 0013 C1 in TextPrompter.
func TestMultiSelect_RepeatedIndexCountsOnce(t *testing.T) {
	p, _ := pipePrompter(t, "1,1,2\n")
	got, err := p.MultiSelect("Pick any", []Choice{{Label: "a"}, {Label: "b"}, {Label: "c"}}, nil)
	if err != nil {
		t.Fatalf("MultiSelect: %v", err)
	}
	if !slices.Equal(got, []int{0, 1}) {
		t.Errorf("MultiSelect(1,1,2) = %v, want [0 1]", got)
	}
}

// TestConfigureLLM_OtherDefaultsOnlyToSameProvider pins MADR 0013 C4 (MADR
// 0007 §4.3): the Other and "No models found" prompts default to the saved
// model only when it belongs to the chosen provider.
func TestConfigureLLM_OtherDefaultsOnlyToSameProvider(t *testing.T) {
	saved := Result{Provider: llmprovider.ProviderClaude, Model: "claude-opus-5"}
	t.Run("other provider, Other", func(t *testing.T) {
		gemini := len(catalog.Static(llmprovider.ProviderGemini))
		f := newFake(t, fakePrompter{
			selects: []int{providerIdx(t, llmprovider.ProviderGemini), gemini},
			secrets: []string{testKey}, inputs: []string{"", acceptDefault},
		})
		res, err := ConfigureLLM(context.Background(), f, Options{Existing: saved})
		if err == nil || !strings.Contains(err.Error(), "no model entered") {
			t.Errorf("Enter at Other saved provider=%s model=%q err=%v, want no model entered (no default from another provider)",
				res.Provider, res.Model, err)
		}
	})
	t.Run("other provider, no models found", func(t *testing.T) {
		f := newFake(t, fakePrompter{
			selects: []int{providerIdx(t, llmprovider.ProviderOllama)},
			inputs:  []string{"http://127.0.0.1:1", acceptDefault}, confirms: []bool{false},
		})
		res, err := ConfigureLLM(context.Background(), f, Options{Existing: saved})
		if err == nil || !strings.Contains(err.Error(), "none entered") {
			t.Errorf("Enter at No models found saved provider=%s model=%q err=%v, want none entered", res.Provider, res.Model, err)
		}
	})
	t.Run("same provider keeps its model", func(t *testing.T) {
		claude := len(catalog.Static(llmprovider.ProviderClaude))
		f := newFake(t, fakePrompter{
			selects: []int{providerIdx(t, llmprovider.ProviderClaude), claude + 1},
			secrets: []string{testKey}, inputs: []string{"", acceptDefault},
		})
		res, err := ConfigureLLM(context.Background(), f, Options{
			Existing: Result{Provider: llmprovider.ProviderClaude, Model: "my-model"},
		})
		if err != nil || res.Model != "my-model" {
			t.Errorf("Enter at Other: model=%q err=%v, want my-model", res.Model, err)
		}
	})
}

// TestConfigureLLM_FallbackExclusionIgnoresCase pins MADR 0013 C6: a primary
// typed in another case is not offered back as a fallback.
func TestConfigureLLM_FallbackExclusionIgnoresCase(t *testing.T) {
	static := catalog.Static(llmprovider.ProviderClaude)
	f := newFake(t, fakePrompter{
		selects: []int{providerIdx(t, llmprovider.ProviderClaude), len(static)},
		secrets: []string{testKey}, inputs: []string{"", "Claude-Haiku-5-5", ""}, multiSelects: [][]int{{}},
	})
	if _, err := ConfigureLLM(context.Background(), f, Options{NeedFallbacks: true}); err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if n := len(f.seenMultiSelectItems[0]); n != len(static)-1 {
		t.Errorf("fallback menu has %d rows, want %d (claude-haiku-5-5 is the primary)", n, len(static)-1)
	}
}

// TestConfigureLLM_BlankModelIDRefused pins MADR 0013 C7 and covers
// model_select.go's empty-id branch: a blank or whitespace-only id is refused
// at Other and at "No models found".
func TestConfigureLLM_BlankModelIDRefused(t *testing.T) {
	claude := len(catalog.Static(llmprovider.ProviderClaude))
	for _, id := range []string{"", "   "} {
		f := newFake(t, fakePrompter{
			selects: []int{providerIdx(t, llmprovider.ProviderClaude), claude},
			secrets: []string{testKey}, inputs: []string{"", id},
		})
		if res, err := ConfigureLLM(context.Background(), f, Options{}); err == nil {
			t.Errorf("Other with %q: model=%q, want an error", id, res.Model)
		}
	}
	f := newFake(t, fakePrompter{
		selects: []int{providerIdx(t, llmprovider.ProviderOllama)},
		inputs:  []string{"http://127.0.0.1:1", "   "}, confirms: []bool{false},
	})
	if res, err := ConfigureLLM(context.Background(), f, Options{}); err == nil {
		t.Errorf("No models found with blanks: model=%q, want an error", res.Model)
	}
}

// TestConfigureLLM_ListingErrorWarns covers configure.go's listing-error
// warning: Ollama unreachable with Discover set.
func TestConfigureLLM_ListingErrorWarns(t *testing.T) {
	f := newFake(t, fakePrompter{
		selects: []int{providerIdx(t, llmprovider.ProviderOllama)},
		inputs:  []string{"http://127.0.0.1:1", "llama3"}, confirms: []bool{false},
	})
	res, err := ConfigureLLM(context.Background(), f, Options{Discover: true})
	if err != nil || res.Model != "llama3" {
		t.Fatalf("ConfigureLLM: model=%q err=%v, want llama3", res.Model, err)
	}
	if n := countContaining(f.seenNotify, "could not list models for Ollama"); n != 1 {
		t.Errorf("listing warning seen %d times, want 1: %v", n, f.seenNotify)
	}
}

// TestConfigureLLM_OllamaEmptyListing covers configure.go's empty live
// listing: a reachable Ollama with nothing installed falls through to manual
// entry without a warning.
func TestConfigureLLM_OllamaEmptyListing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			_, _ = w.Write([]byte(`{"models":[]}`))
		}
	}))
	t.Cleanup(srv.Close)
	f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderOllama)}, inputs: []string{srv.URL, "llama3"}})
	res, err := ConfigureLLM(context.Background(), f, Options{Discover: true})
	if err != nil || res.Model != "llama3" {
		t.Fatalf("ConfigureLLM: model=%q err=%v, want llama3", res.Model, err)
	}
	if len(f.seenNotify) != 0 {
		t.Errorf("notices = %v, want none for an empty install", f.seenNotify)
	}
}

// TestConfigureLLM_DefaultRowIsExistingModel covers model_select.go's default
// row: a saved model in the recommended list is the menu default.
func TestConfigureLLM_DefaultRowIsExistingModel(t *testing.T) {
	static := catalog.Static(llmprovider.ProviderClaude)
	f := newFake(t, fakePrompter{blankSearches: true, selects: []int{providerIdx(t, llmprovider.ProviderClaude), 2}, secrets: []string{testKey}})
	res, err := ConfigureLLM(context.Background(), f, Options{
		Existing: Result{Provider: llmprovider.ProviderClaude, Model: static[2]},
	})
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if f.seenSelectDefault[1] != 2 || res.Model != static[2] {
		t.Errorf("default = %d, model = %q; want 2 and %q", f.seenSelectDefault[1], res.Model, static[2])
	}
}

// TestConfigureLLM_BlankFallbackRoundWithNothingLeft covers model_select.go's
// blank round after search has taken every recommended model: it ends the
// loop without an empty MultiSelect.
func TestConfigureLLM_BlankFallbackRoundWithNothingLeft(t *testing.T) {
	srv := zenServer(t, http.StatusOK, zenListing(zenSearchIDs))
	f := newFake(t, fakePrompter{
		selects: []int{providerIdx(t, llmprovider.ProviderOpencodeZen), 0}, secrets: []string{testKey},
		inputs:       []string{srv.URL, "", "flash", ""},
		multiSelects: [][]int{{0, 1, 2, 3, 4}}, confirms: []bool{true},
	})
	opts := zenOptions(t)
	opts.NeedFallbacks = true
	opts.ModelLimit = catalog.MaxListed
	res, err := ConfigureLLM(context.Background(), f, opts)
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if len(res.Fallbacks) != 5 || len(f.seenMultiSelectItems) != 1 {
		t.Errorf("Fallbacks = %v after %d MultiSelects, want 5 after 1", res.Fallbacks, len(f.seenMultiSelectItems))
	}
}

// TestConfigureLLM_FallbackSearchNoMatches covers model_select.go's no-match
// fallback search: it warns and asks again.
func TestConfigureLLM_FallbackSearchNoMatches(t *testing.T) {
	f := newFake(t, fakePrompter{
		selects: []int{providerIdx(t, llmprovider.ProviderClaude), 0}, secrets: []string{testKey},
		inputs: []string{"", "zzz-no-match", ""}, multiSelects: [][]int{{}},
	})
	if _, err := ConfigureLLM(context.Background(), f, Options{NeedFallbacks: true}); err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if n := countContaining(f.seenNotify, `match "zzz-no-match"`); n != 1 {
		t.Errorf("no-match warning seen %d times, want 1: %v", n, f.seenNotify)
	}
}

// TestConfigureLLM_ListingTokenFailureUsesStaticCatalog covers configure.go's
// listing-error branch for a provider that has a static catalog: a kept Grok
// session whose refresh cannot finish inside the listing's deadline fails the
// listing, so the wizard warns and offers the built-in catalog. The 1 ns limit
// has passed when the refresh starts, but the listing's fetch is detached from
// it and refreshes anyway (0053-MADR D4): the run's client refuses every
// request, where the default one reached auth.x.ai on Windows (0066-MADR,
// amendment "a leak V13's guard found on Windows").
func TestConfigureLLM_ListingTokenFailureUsesStaticCatalog(t *testing.T) {
	store := newMemoryTokenStore()
	static := catalog.Static(llmprovider.ProviderGrok)
	f := newFake(t, fakePrompter{blankSearches: true, selects: []int{providerIdx(t, llmprovider.ProviderGrok), 0}, confirms: []bool{true}})
	res, err := ConfigureLLM(context.Background(), f, Options{
		Existing: storedExisting(t, store, Result{
			Provider:    llmprovider.ProviderGrok,
			Kind:        CredOAuth,
			TokenExpiry: time.Now().Add(-time.Hour),
			Issuer:      auth.DefaultGrokOAuthIssuer,
			ClientID:    auth.DefaultGrokOAuthClientID,
		}, "expired-access-abcd", "refresh"),
		TokenStore:    store,
		Discover:      true,
		DiscoverLimit: time.Nanosecond,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, io.ErrUnexpectedEOF
		})},
		ProviderOptions: []llmprovider.Option{testCache(t), llmprovider.WithoutModelMetadata()},
	})
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if res.Model != static[0] {
		t.Errorf("Model = %q, want the first built-in model %q", res.Model, static[0])
	}
	// The deadline as "timed out", not the error's text (0056-MADR D3).
	if n := countContaining(f.seenNotify, "(timed out); using the built-in catalog"); n != 1 {
		t.Errorf("listing warning seen %d times, want 1: %v", n, f.seenNotify)
	}
}

// TestConfigureLLM_UnusableListingNotice covers the static-catalog notice with
// no cause: a listing that succeeds but offers no usable model degrades
// without a ModelCatalog.Err.
func TestConfigureLLM_UnusableListingNotice(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"not-a-claude-model"}],"has_more":false}`)),
			Request:    r,
		}, nil
	})}
	f := newFake(t, fakePrompter{blankSearches: true, selects: []int{providerIdx(t, llmprovider.ProviderClaude), 0}, secrets: []string{testKey}})
	if _, err := ConfigureLLM(context.Background(), f, Options{Discover: true, HTTPClient: client,
		ProviderOptions: []llmprovider.Option{testCache(t)}}); err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	want := "live model listing for Claude (Anthropic) is unavailable; search covers the built-in catalog only"
	if n := countContaining(f.seenNotify, want); n != 1 {
		t.Errorf("notice %q seen %d times, want 1: %v", want, n, f.seenNotify)
	}
}

// ollamaShowServer answers GET /api/version, which the wizard checks first,
// GET /api/tags with the models, in order, and POST /api/show with each
// model's capabilities (0042-MADR D1).
func ollamaShowServer(t *testing.T, models ...[2]string) *httptest.Server {
	t.Helper()
	caps := map[string]string{}
	var tags []string
	for _, m := range models {
		caps[m[0]] = m[1]
		tags = append(tags, `{"name":"`+m[0]+`"}`)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/version":
			_, _ = io.WriteString(w, `{"version":"0.40.2"}`)
		case "/api/tags":
			_, _ = io.WriteString(w, `{"models":[`+strings.Join(tags, ",")+`]}`)
		case "/api/show":
			raw, _ := io.ReadAll(r.Body)
			for name, c := range caps {
				if strings.Contains(string(raw), `"`+name+`"`) {
					_, _ = io.WriteString(w, `{"model_info":{},"capabilities":`+c+`}`)
					return
				}
			}
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// ollama0042S1 is issue 1's install: an embedding model listed first.
var ollama0042S1 = [][2]string{
	{"granite-embedding:30m", `["embedding"]`},
	{"qwen3-vl:8b", `["completion","vision","tools","thinking"]`},
	{"llama3.1:8b-instruct-q5_K_M", `["completion","tools"]`},
	{"qwen2.5-coder:7b", `["completion","tools","insert"]`},
}

// TestConfigureLLM_OllamaLeavesOutEmbedding_0042: the wizard neither shows
// nor finds a model Ollama says writes no text, and an install of only
// those falls through to manual entry, as an empty one does (0042-MADR D1).
func TestConfigureLLM_OllamaLeavesOutEmbedding_0042(t *testing.T) {
	ollama := providerIdx(t, llmprovider.ProviderOllama)
	t.Run("menu", func(t *testing.T) {
		srv := ollamaShowServer(t, ollama0042S1...)
		f := newFake(t, fakePrompter{selects: []int{ollama, 0}, inputs: []string{srv.URL, ""}})
		res, err := ConfigureLLM(context.Background(), f, Options{Discover: true, ProviderOptions: []llmprovider.Option{testCache(t)}})
		if err != nil || res.Model != "qwen3-vl:8b" {
			t.Fatalf("ConfigureLLM: model=%q err=%v, want qwen3-vl:8b", res.Model, err)
		}
		menu := f.seenSelectItems[len(f.seenSelectItems)-1]
		for _, c := range menu {
			if strings.Contains(c.Label, "granite") {
				t.Errorf("the menu offers %q", c.Label)
			}
		}
		if len(menu) != 4 {
			t.Errorf("menu %v; want the three chat models and Other", menu)
		}
	})
	t.Run("search", func(t *testing.T) {
		srv := ollamaShowServer(t, ollama0042S1...)
		f := newFake(t, fakePrompter{selects: []int{ollama, 1, 0}, inputs: []string{srv.URL, "granite", "qwen2.5"}})
		res, err := ConfigureLLM(context.Background(), f, Options{Discover: true, ProviderOptions: []llmprovider.Option{testCache(t)}})
		if err != nil || res.Model != "qwen2.5-coder:7b" {
			t.Fatalf("ConfigureLLM: model=%q err=%v, want qwen2.5-coder:7b", res.Model, err)
		}
		if n := countContaining(f.seenNotify, `models match "granite"`); n != 1 {
			t.Errorf("search for granite: notices %v; want no match", f.seenNotify)
		}
	})
	t.Run("only embedding models", func(t *testing.T) {
		srv := ollamaShowServer(t, [2]string{"granite-embedding:30m", `["embedding"]`},
			[2]string{"nomic-embed-text:latest", `["embedding"]`})
		f := newFake(t, fakePrompter{selects: []int{ollama}, inputs: []string{srv.URL, "llama3"}})
		res, err := ConfigureLLM(context.Background(), f, Options{Discover: true, ProviderOptions: []llmprovider.Option{testCache(t)}})
		if err != nil || res.Model != "llama3" {
			t.Fatalf("ConfigureLLM: model=%q err=%v, want llama3", res.Model, err)
		}
		if len(f.seenNotify) != 0 {
			t.Errorf("notices = %v, want none, as for an empty install", f.seenNotify)
		}
	})
}

// TestConfigureLLM_OllamaToolless_0042: an install whose chat models take
// no tools offers them as the menu, with no "meets the profile" notice
// (0042-MADR D2).
func TestConfigureLLM_OllamaToolless_0042(t *testing.T) {
	srv := ollamaShowServer(t, [2]string{"gemma3:12b", `["completion","vision"]`},
		[2]string{"codellama:7b", `["completion","insert"]`})
	f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderOllama), 1}, inputs: []string{srv.URL, ""}})
	res, err := ConfigureLLM(context.Background(), f, Options{Discover: true, ProviderOptions: []llmprovider.Option{testCache(t)}})
	if err != nil || res.Model != "codellama:7b" {
		t.Fatalf("ConfigureLLM: model=%q err=%v, want codellama:7b", res.Model, err)
	}
	if len(f.seenNotify) != 0 {
		t.Errorf("notices = %v; want none", f.seenNotify)
	}
	if menu := f.seenSelectItems[len(f.seenSelectItems)-1]; len(menu) != 3 {
		t.Errorf("menu %v; want both models and Other", menu)
	}
}
