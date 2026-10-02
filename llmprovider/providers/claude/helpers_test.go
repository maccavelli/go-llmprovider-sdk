package claude

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Test helpers, as the openai package's (0015-PLAN S7).

const okReply = `{"content":[{"type":"text","text":"ok"}]}`

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

func replyServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// apiKey is the options for a key "k" against url, with the given model.
func apiKey(url, model string, opts ...llmprovider.Option) []llmprovider.Option {
	return append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModel(model),
		llmprovider.WithBaseURL(url)}, opts...)
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

// text is a request with one user message.
func text(prompt string) *llmprovider.Request {
	return &llmprovider.Request{Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: prompt}}}
}

// withTool adds tool to req and forces it, as the old GenerateWithTool did.
func withTool(req *llmprovider.Request, tool llmprovider.Tool) *llmprovider.Request {
	req.Tools = []llmprovider.Tool{tool}
	req.ToolChoice = llmprovider.ForceTool(tool.Name)
	return req
}

// thinking adds reasoning at the provider's defaults, as GenerateThinking did.
func thinking(req *llmprovider.Request) *llmprovider.Request {
	req.Reasoning = &llmprovider.Reasoning{}
	return req
}
