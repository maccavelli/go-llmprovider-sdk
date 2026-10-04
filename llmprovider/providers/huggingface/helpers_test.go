package huggingface

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

const fxHFChat = `{"id":"cmpl-hf","model":"llmtest-model","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"hello"}}]}`

// hfListing is a router listing with one live, tool-capable text model.
const hfListing = `{"data":[{"id":"openai/gpt-oss-120b","architecture":{"input_modalities":["text"],"output_modalities":["text"]},` +
	`"providers":[{"status":"live","supports_tools":true,"throughput":100,"first_token_latency_ms":300}]}]}`

// captureServer records the decoded JSON body of the last request and
// replies with respBody.
func captureServer(t *testing.T, lastBody *map[string]any, respBody string) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		m := map[string]any{}
		_ = json.Unmarshal(raw, &m)
		mu.Lock()
		*lastBody = m
		mu.Unlock()
		_, _ = io.WriteString(w, respBody)
	}))
	t.Cleanup(srv.Close)
	return srv
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

// thinking adds reasoning at the provider's defaults, as GenerateThinking did.
func thinking(req *llmprovider.Request) *llmprovider.Request {
	req.Reasoning = &llmprovider.Reasoning{}
	return req
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
