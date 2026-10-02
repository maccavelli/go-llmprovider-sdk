package ollama

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Test helpers, as the other provider packages' (0015-PLAN S7).

const fxOllamaChat = `{"id":"chatcmpl-1","object":"chat.completion","model":"llama3.2:latest",
"choices":[{"index":0,"message":{"role":"assistant","content":"ALPHA"}}]}`

// ollamaTags is a GET /api/tags body listing names.
func ollamaTags(names ...string) string {
	models := make([]map[string]string, len(names))
	for i, n := range names {
		models[i] = map[string]string{"name": n}
	}
	b, _ := json.Marshal(map[string]any{"models": models})
	return string(b)
}

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

// local is the options for the instance at url, with no key, and the given
// model.
func local(url, model string, opts ...llmprovider.Option) []llmprovider.Option {
	return append([]llmprovider.Option{llmprovider.WithModel(model), llmprovider.WithBaseURL(url)}, opts...)
}

// build is New, failing the test on an error.
func build(t *testing.T, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := New(opts...)
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
