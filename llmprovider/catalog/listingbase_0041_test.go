package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0041-MADR A8: the Claude lister takes a base with or without /v1, and the
// listing cache is keyed by the base the listing reads.

// claudeEffortPage is a Claude listing of one model whose facts list one
// effort.
const claudeEffortPage = `{"data":[{"id":"claude-sonnet-4-5-0041","capabilities":` +
	`{"effort":{"supported":true,"low":{"supported":true}}}}],"has_more":false}`

// baseServer answers /v1/models with page and anything else with {}, and
// records each request's URL as it was asked for, before any rewrite.
type baseServer struct {
	*httptest.Server
	mu   sync.Mutex
	urls []string
}

func newBaseServer(t *testing.T, page string) *baseServer {
	t.Helper()
	s := &baseServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			_, _ = w.Write([]byte(page))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(s.Close)
	return s
}

// RoundTrip records req's URL and sends it to s, keeping its path.
func (s *baseServer) RoundTrip(req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	s.urls = append(s.urls, req.URL.Scheme+"://"+req.URL.Host+req.URL.Path)
	s.mu.Unlock()
	u, err := url.Parse(s.URL)
	if err != nil {
		return nil, err
	}
	req = req.Clone(req.Context())
	req.URL.Scheme, req.URL.Host = u.Scheme, u.Host
	return http.DefaultTransport.RoundTrip(req)
}

// listings are the URLs s was asked for, the models.dev document's left out.
func (s *baseServer) listings() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, u := range s.urls {
		if !strings.HasSuffix(u, "/api.json") {
			out = append(out, u)
		}
	}
	return out
}

// client is an HTTP client whose every request goes to s.
func (s *baseServer) client() *http.Client { return &http.Client{Transport: s} }

// TestList_ClaudeBase_0041: a Claude base with or without /v1, and with or
// without a trailing slash, lists /v1/models, live; ListRaw names that URL.
func TestList_ClaudeBase_0041(t *testing.T) {
	for _, suffix := range []string{"", "/", "/v1", "/v1/"} {
		resetLookupCaches(t)
		s := newBaseServer(t, claudeEffortPage)
		opts := []llmprovider.Option{llmprovider.WithBaseURL(s.URL + suffix), llmprovider.WithHTTPClient(s.client()),
			llmprovider.WithModelMetadataURL(s.URL + "/api.json")}
		cat, err := List(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"), opts...)
		if err != nil {
			t.Fatal(err)
		}
		if got := s.listings(); cat.Source != ListLive || len(got) != 1 || got[0] != s.URL+"/v1/models" {
			t.Errorf("base %q: Source %v, read %q; want live from %s/v1/models", suffix, cat.Source, got, s.URL)
		}
		raw, err := ListRaw(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"), opts...)
		if err != nil || raw.Endpoint != s.URL+"/v1/models" {
			t.Errorf("base %q: ListRaw endpoint %q, %v; want %s/v1/models", suffix, raw.Endpoint, err, s.URL)
		}
	}
}

// providerDefaultBases are the base URLs the built-in providers pass the
// catalog when the caller names none: each provider package's default.
var providerDefaultBases = map[llmprovider.ProviderID]string{
	llmprovider.ProviderOpenAI:      "https://api.openai.com/v1",
	llmprovider.ProviderClaude:      "https://api.anthropic.com/v1",
	llmprovider.ProviderGemini:      "https://generativelanguage.googleapis.com/v1beta",
	llmprovider.ProviderGrok:        "https://api.x.ai/v1",
	llmprovider.ProviderOpencodeZen: "https://opencode.ai/zen/v1",
	llmprovider.ProviderOpencodeGo:  "https://opencode.ai/zen/go/v1",
	llmprovider.ProviderKilo:        "https://api.kilo.ai/api/gateway",
	llmprovider.ProviderHuggingFace: "https://router.huggingface.co/v1",
	llmprovider.ProviderTogether:    "https://api.together.ai/v1",
	llmprovider.ProviderOllama:      "http://localhost:11434",
}

// TestListingKey_DefaultBase_0041: no base URL and the provider's default
// are one cache key, for every provider; another base is another key.
func TestListingKey_DefaultBase_0041(t *testing.T) {
	token := llmprovider.Token{Value: "k"}
	for id, base := range providerDefaultBases {
		if got, want := listingKeyFor(id, config{}, token), listingKeyFor(id, config{BaseURL: base}, token); got != want {
			t.Errorf("%s: key with no base %+v; with %s %+v; want one key", id, got, base, want)
		}
		if listingKeyFor(id, config{}, token) == listingKeyFor(id, config{BaseURL: "http://proxy.test/v1"}, token) {
			t.Errorf("%s: no base and another base share a key", id)
		}
	}
}

// TestCachedFacts_ListWithoutBaseURL_0041: a catalog.List with no base URL
// warms the listing facts a Claude provider's request reads, under its
// default base.
func TestCachedFacts_ListWithoutBaseURL_0041(t *testing.T) {
	resetLookupCaches(t)
	s := newBaseServer(t, claudeEffortPage)
	opts := []llmprovider.Option{llmprovider.WithHTTPClient(s.client()), llmprovider.WithModelMetadataURL(s.URL + "/api.json")}
	if _, err := List(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"), opts...); err != nil {
		t.Fatal(err)
	}
	info, ok := CachedFacts(context.Background(), llmprovider.ProviderClaude, "claude-sonnet-4-5-0041",
		append(opts, llmprovider.WithAPIKey("k"), llmprovider.WithBaseURL(providerDefaultBases[llmprovider.ProviderClaude]))...)
	if !ok || !info.Has(FieldReasoningEfforts) || len(info.Reasoning.Efforts) != 1 {
		t.Errorf("CachedFacts: %t, efforts known %t %v; want the listing's [low]", ok, info.Has(FieldReasoningEfforts),
			info.Reasoning.Efforts)
	}
}

// TestListingBase_IsWhatListReads_0041: with no base URL, every provider's
// listing is read under listingBaseURL's default, so the key names the URL
// the listing reads.
func TestListingBase_IsWhatListReads_0041(t *testing.T) {
	for id := range providerDefaultBases {
		resetLookupCaches(t)
		s := newBaseServer(t, claudeEffortPage)
		_, _ = List(context.Background(), id, llmprovider.NewStaticToken("k"), llmprovider.WithHTTPClient(s.client()),
			llmprovider.WithModelMetadataURL(s.URL+"/api.json"))
		base := listingBaseURL(id, "")
		if got := s.listings(); len(got) == 0 || !strings.HasPrefix(got[0], base+"/") {
			t.Errorf("%s: read %q; want under %s", id, got, base)
		}
	}
}
