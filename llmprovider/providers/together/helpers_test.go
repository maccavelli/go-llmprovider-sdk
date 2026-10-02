package together

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Test helpers, as the other provider packages' (0015-PLAN S7).

const togetherText = `{"id":"t-1","choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"hello"}}]}`

// weatherTool is the tool the tool tests offer.
var weatherTool = llmprovider.Tool{Name: "get_weather", Description: "Weather for a city",
	Schema: map[string]any{"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}}}

// togetherCapture records the last request a stub Together server received.
type togetherCapture struct {
	mu     sync.Mutex
	path   string
	header http.Header
	body   map[string]any
}

func (c *togetherCapture) last() (string, http.Header, map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.path, c.header, c.body
}

// togetherServer answers /models with listing and everything else with reply.
func togetherServer(t *testing.T, reply, listing string) (*httptest.Server, *togetherCapture) {
	t.Helper()
	c := &togetherCapture{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/models" {
			_, _ = io.WriteString(w, listing)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		c.mu.Lock()
		c.path, c.header, c.body = r.URL.Path, r.Header.Clone(), body
		c.mu.Unlock()
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	return srv, c
}

// metadataServer serves body with status at any path and counts requests.
func metadataServer(t *testing.T, status int, body string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

// apiKey is the options for a key "k" against url, with the given model.
func apiKey(url, model string, opts ...llmprovider.Option) []llmprovider.Option {
	return append(append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModel(model),
		llmprovider.WithBaseURL(url)}, metadataOff()...), opts...)
}

// build is New, failing the test on an error.
func build(t *testing.T, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := New(append(metadataOff(), opts...)...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return p
}

// user is a user message.
func user(text string) llmprovider.MessageItem {
	return llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: text}
}

// text is a request with one user message.
func text(prompt string) *llmprovider.Request {
	return &llmprovider.Request{Input: []llmprovider.Item{user(prompt)}}
}

func list(t *testing.T, p llmprovider.Provider) ([]string, error) {
	t.Helper()
	lister, ok := p.(llmprovider.ModelLister)
	if !ok {
		t.Fatal("the provider is not a ModelLister")
	}
	return lister.ListModels(t.Context())
}

// metadataOn is set by enableMetadata for one test, which serves its own
// metadata document. Otherwise the helpers turn the fetch off, so no test
// reaches models.opencode.ai (0015-PLAN S10). No test here runs in parallel.
var metadataOn bool

// enableMetadata leaves the metadata fetch on for one test.
func enableMetadata(t *testing.T) {
	t.Helper()
	metadataOn = true
	t.Cleanup(func() { metadataOn = false })
}

// metadataOff is WithoutModelMetadata, unless enableMetadata turned the fetch
// on.
func metadataOff() []llmprovider.Option {
	if metadataOn {
		return nil
	}
	return []llmprovider.Option{llmprovider.WithoutModelMetadata()}
}
