package catalog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// openAIStyleListing renders a {"data":[{"id":…}]} listing, the shape shared by
// OpenAI, Anthropic, xAI and the OpenCode gateways.
func openAIStyleListing(ids ...string) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf(`{"id":%q}`, id))
	}
	return `{"data":[` + strings.Join(parts, ",") + `]}`
}

// ollamaTags renders an Ollama GET /api/tags body.
func ollamaTags(names ...string) string {
	parts := make([]string, 0, len(names))
	for _, n := range names {
		parts = append(parts, fmt.Sprintf(`{"name":%q}`, n))
	}
	return `{"models":[` + strings.Join(parts, ",") + `]}`
}

// hfListing renders n text→text, live, tool-capable Hugging Face models named
// org/m-01…, with throughput descending so listing order is metadata order.
func hfListing(n int) string {
	parts := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		parts = append(parts, fmt.Sprintf(`{"id":"org/m-%02d","architecture":{"input_modalities":["text"],"output_modalities":["text"]},`+
			`"providers":[{"provider":"a","status":"live","supports_tools":true,"throughput":%d,"first_token_latency_ms":100}]}`, i, 1000-i))
	}
	return `{"object":"list","data":[` + strings.Join(parts, ",") + `]}`
}

// kiloListing renders n text→text, tool-capable, non-training Kilo models named
// org/m-01…, with completion price ascending so listing order is price order.
func kiloListing(n int) string {
	parts := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		parts = append(parts, fmt.Sprintf(`{"id":"org/m-%02d","architecture":{"input_modalities":["text"],"output_modalities":["text"]},`+
			`"pricing":{"completion":"0.00000%d"},"supported_parameters":["tools"],"mayTrainOnYourPrompts":false}`, i, i))
	}
	return `{"data":[` + strings.Join(parts, ",") + `]}`
}

const geminiCatalogFixture = `{"models":[
 {"name":"models/gemini-3.7-flash","supportedGenerationMethods":["generateContent"]},
 {"name":"models/gemini-2.5-flash","supportedGenerationMethods":["generateContent"]},
 {"name":"models/gemini-embedding-001","supportedGenerationMethods":["embedContent"]},
 {"name":"models/gemini-3.5-flash-preview-09-2026","supportedGenerationMethods":["generateContent"]}]}`

// catalogCase is one provider with a good listing fixture.
type catalogCase struct {
	provider  llmprovider.ProviderID
	key, body string
}

func goodCatalogCases() []catalogCase {
	return []catalogCase{
		{llmprovider.ProviderGemini, "k", geminiCatalogFixture},
		{llmprovider.ProviderOpenAI, "k", openAIStyleListing("gpt-4.1-mini", "gpt-4o", "gpt-5.1")},
		{llmprovider.ProviderClaude, "k", openAIStyleListing("claude-haiku-4-5", "claude-sonnet-5")},
		{llmprovider.ProviderGrok, "k", openAIStyleListing("grok-4.6", "grok-3-mini")},
		{llmprovider.ProviderOpencodeZen, "k", opencodeListingFixture},
		{llmprovider.ProviderOpencodeGo, "k", opencodeListingFixture},
		{llmprovider.ProviderHuggingFace, "k", hfListingFixture},
		{llmprovider.ProviderKilo, "k", kiloListingFixture},
		{llmprovider.ProviderOllama, "", ollamaTags("llama3:latest", "mistral:7b")},
	}
}

// serveBody starts a server that answers every request with body.
func serveBody(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestListModelCatalog_RecommendedMatchesListAvailable(t *testing.T) {
	for _, tc := range goodCatalogCases() {
		t.Run(string(tc.provider), func(t *testing.T) {
			srv := serveBody(t, tc.body)
			cat, err := List(context.Background(), tc.provider, llmprovider.NewStaticToken(tc.key), llmprovider.WithBaseURL(srv.URL))
			if err != nil {
				t.Fatalf("ListModelCatalog: %v", err)
			}
			avail, err := listRecommended(context.Background(), tc.provider, llmprovider.NewStaticToken(tc.key), llmprovider.WithBaseURL(srv.URL))
			if err != nil {
				t.Fatalf("ListAvailableModels: %v", err)
			}
			if !slices.Equal(cat.Recommended, avail) {
				t.Errorf("Recommended = %v, ListAvailableModels = %v", cat.Recommended, avail)
			}
			if !cat.Live {
				t.Errorf("Live = false for a good fixture")
			}
		})
	}
}

func TestListModelCatalog_OpenAIRecommendedIsCurated(t *testing.T) {
	order := []string{"gpt-5.1", "o3", "gpt-4o", "o4-mini", "gpt-4.1", "gpt-4o-mini", "gpt-4.1-nano", "gpt-4.1-mini"}
	srv := serveBody(t, openAIStyleListing(order...))
	cat, err := List(context.Background(), llmprovider.ProviderOpenAI, llmprovider.NewStaticToken("k"), llmprovider.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("ListModelCatalog: %v", err)
	}
	if !slices.Equal(cat.Recommended, staticOpenAI) {
		t.Errorf("Recommended = %v, want staticOpenAI %v in catalog order", cat.Recommended, staticOpenAI)
	}
	if !slices.Equal(cat.Usable, order) {
		t.Errorf("Usable = %v, want listing order %v", cat.Usable, order)
	}
	if !cat.Live {
		t.Error("Live = false")
	}
}

func TestListModelCatalog_UsableIsUncapped(t *testing.T) {
	zen := make([]string, 0, 8)
	for i := 1; i <= 8; i++ {
		zen = append(zen, fmt.Sprintf("zen-model-%02d", i))
	}
	for _, tc := range []catalogCase{
		{llmprovider.ProviderHuggingFace, "k", hfListing(8)},
		{llmprovider.ProviderKilo, "k", kiloListing(8)},
		{llmprovider.ProviderOpencodeZen, "k", openAIStyleListing(zen...)},
	} {
		t.Run(string(tc.provider), func(t *testing.T) {
			srv := serveBody(t, tc.body)
			cat, err := List(context.Background(), tc.provider, llmprovider.NewStaticToken(tc.key), llmprovider.WithBaseURL(srv.URL))
			if err != nil {
				t.Fatalf("ListModelCatalog: %v", err)
			}
			if len(cat.Usable) != 8 {
				t.Errorf("len(Usable) = %d, want 8: %v", len(cat.Usable), cat.Usable)
			}
			if len(cat.Recommended) != MaxListed {
				t.Errorf("len(Recommended) = %d, want %d", len(cat.Recommended), MaxListed)
			}
		})
	}
}

func TestListModelCatalog_RecommendedSubsetOfUsable(t *testing.T) {
	for _, tc := range goodCatalogCases() {
		t.Run(string(tc.provider), func(t *testing.T) {
			srv := serveBody(t, tc.body)
			cat, err := List(context.Background(), tc.provider, llmprovider.NewStaticToken(tc.key), llmprovider.WithBaseURL(srv.URL))
			if err != nil {
				t.Fatalf("ListModelCatalog: %v", err)
			}
			for _, id := range cat.Recommended {
				if !slices.Contains(cat.Usable, id) {
					t.Errorf("Recommended id %q is not in Usable %v", id, cat.Usable)
				}
			}
		})
	}
}

func TestListModelCatalog_FiltersStillApply(t *testing.T) {
	for _, tc := range []struct {
		catalogCase
		absent []string
	}{
		{catalogCase{llmprovider.ProviderKilo, "k", kiloListingFixture}, []string{"org/trains", "org/no-tools"}},
		{catalogCase{llmprovider.ProviderHuggingFace, "k", hfListingFixture}, []string{"org/not-live"}},
		{catalogCase{llmprovider.ProviderOpencodeZen, "k", opencodeListingFixture}, []string{"deepseek-v4-flash-vision-exp"}},
		{catalogCase{llmprovider.ProviderGemini, "k", geminiCatalogFixture}, []string{"gemini-embedding-001", "gemini-3.5-flash-preview-09-2026"}},
	} {
		t.Run(string(tc.provider), func(t *testing.T) {
			srv := serveBody(t, tc.body)
			cat, err := List(context.Background(), tc.provider, llmprovider.NewStaticToken(tc.key), llmprovider.WithBaseURL(srv.URL))
			if err != nil {
				t.Fatalf("ListModelCatalog: %v", err)
			}
			if !cat.Live {
				t.Fatalf("Live = false; the fixture must be read, not replaced by static")
			}
			for _, id := range tc.absent {
				if slices.Contains(cat.Usable, id) {
					t.Errorf("filtered id %q is in Usable %v", id, cat.Usable)
				}
			}
		})
	}
}

func TestListModelCatalog_OllamaSplit(t *testing.T) {
	names := []string{"m1", "m2", "m3", "m4", "m5", "m6", "m7", "m8"}
	srv := serveBody(t, ollamaTags(names...))
	cat, err := List(context.Background(), llmprovider.ProviderOllama, llmprovider.NewStaticToken(""), llmprovider.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("ListModelCatalog: %v", err)
	}
	if !slices.Equal(cat.Usable, names) {
		t.Errorf("Usable = %v, want %v", cat.Usable, names)
	}
	if !slices.Equal(cat.Recommended, names[:MaxListed]) {
		t.Errorf("Recommended = %v, want %v", cat.Recommended, names[:MaxListed])
	}
	// Appending to Recommended must not write through into Usable.
	_ = append(cat.Recommended, "sentinel")
	if cat.Usable[MaxListed] != names[MaxListed] {
		t.Errorf("Usable[%d] = %q after appending to Recommended; the views alias", MaxListed, cat.Usable[MaxListed])
	}
}

func TestListModelCatalog_OllamaEmptyIsLive(t *testing.T) {
	srv := serveBody(t, `{"models":[]}`)
	cat, err := List(context.Background(), llmprovider.ProviderOllama, llmprovider.NewStaticToken(""), llmprovider.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("ListModelCatalog: %v", err)
	}
	if !cat.Live || len(cat.Recommended) != 0 || len(cat.Usable) != 0 {
		t.Errorf("catalog = %+v, want Live with both views empty", cat)
	}
}

func TestListModelCatalog_LiveFlag(t *testing.T) {
	for _, p := range []llmprovider.ProviderID{
		llmprovider.ProviderGemini, llmprovider.ProviderOpenAI, llmprovider.ProviderClaude, llmprovider.ProviderGrok,
		llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo, llmprovider.ProviderHuggingFace, llmprovider.ProviderKilo,
	} {
		t.Run(string(p), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer srv.Close()
			cat, err := List(context.Background(), p, llmprovider.NewStaticToken("k"), llmprovider.WithBaseURL(srv.URL))
			if err != nil {
				t.Fatalf("a failed listing must degrade, not error: %v", err)
			}
			if cat.Live {
				t.Error("Live = true for a failed listing")
			}
			static := Static(p)
			if !slices.Equal(cat.Recommended, static) || !slices.Equal(cat.Usable, static) {
				t.Errorf("catalog = %+v, want both views = %v", cat, static)
			}
		})
	}
}

func TestListModelCatalogWithSource_ChatGPTListsCodexCatalog(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte(`{"models":[{"slug":"gpt-6-astra","visibility":"list","priority":1}]}`))
	}))
	defer srv.Close()

	session := &llmprovider.OAuthSession{
		Issuer: llmprovider.DefaultOpenAIIssuer,
		Access: "session-access",
		Expiry: time.Now().Add(time.Hour),
	}
	cat, err := List(context.Background(), llmprovider.ProviderOpenAI, session, llmprovider.WithHTTPClient(srv.Client()), llmprovider.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("ListModelCatalogWithSource() error = %v", err)
	}
	want := []string{"gpt-6-astra"}
	if !reflect.DeepEqual(cat.Recommended, want) || !reflect.DeepEqual(cat.Usable, want) || !cat.Live {
		t.Fatalf("catalog = %+v, want a live catalog with both views = %v", cat, want)
	}
	if !reflect.DeepEqual(paths, []string{"/models"}) {
		t.Fatalf("requests = %v, want one Codex /models listing", paths)
	}
	cat.Recommended[0] = "mutated"
	if cat.Usable[0] == "mutated" {
		t.Fatal("Recommended and Usable share a backing array")
	}
}

func TestListModelCatalog_Errors(t *testing.T) {
	if _, err := List(context.Background(), "unsupported", llmprovider.NewStaticToken("k")); err == nil {
		t.Error("unsupported provider: want an error")
	}
	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFound.Close()
	if _, err := List(context.Background(), llmprovider.ProviderOllama, llmprovider.NewStaticToken(""), llmprovider.WithBaseURL(notFound.URL)); err == nil {
		t.Error("ollama 404: want an error")
	}
	if _, err := List(context.Background(), llmprovider.ProviderGemini, nil); err == nil {
		t.Error("nil TokenSource: want an error")
	}
}

// TestListModelCatalog_InputModalityContainsText pins MADR 0007 §1b for both
// metadata-driven gateways: input containing text is admitted, output must be
// exactly text, and input without text is rejected.
func TestListModelCatalog_InputModalityContainsText(t *testing.T) {
	for _, tc := range []catalogCase{
		{llmprovider.ProviderHuggingFace, "k", hfListingFixture},
		{llmprovider.ProviderKilo, "k", kiloListingFixture},
	} {
		t.Run(string(tc.provider), func(t *testing.T) {
			srv := serveBody(t, tc.body)
			cat, err := List(context.Background(), tc.provider, llmprovider.NewStaticToken(tc.key), llmprovider.WithBaseURL(srv.URL))
			if err != nil {
				t.Fatalf("ListModelCatalog: %v", err)
			}
			if !slices.Contains(cat.Usable, "org/vlm") {
				t.Errorf("org/vlm (text+image in, text out) missing from Usable %v", cat.Usable)
			}
			for _, id := range []string{"org/painter", "org/listener"} {
				if slices.Contains(cat.Usable, id) {
					t.Errorf("%s must be rejected, Usable = %v", id, cat.Usable)
				}
			}
		})
	}
}
