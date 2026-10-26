package grok

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// Test helpers, as the other provider packages' (0015-PLAN S7).

const okResponse = `{"id":"r","output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`

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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func httpResponse(request *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    request,
	}
}

// apiKey is the options for a key "k" against url, with the given model.
func apiKey(url, model string, opts ...llmprovider.Option) []llmprovider.Option {
	return append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModel(model),
		llmprovider.WithBaseURL(url)}, opts...)
}

// caches holds each running test's catalog cache, by *testing.T.
var caches sync.Map

// testCache is t's own catalog cache, the same for every call in t. httptest
// reuses ports, so nothing a test caches under its server's URL may reach the
// process's cache, where a later test's server could read it (0058-MADR D1).
// A test's servers stay open until it ends, so none of them shares a port
// with another within it; a server closed sooner gets a cache of its own.
func testCache(t *testing.T) llmprovider.Option {
	t.Helper()
	c, loaded := caches.LoadOrStore(t, new(catalog.Cache))
	if !loaded {
		t.Cleanup(func() { caches.Delete(t) })
	}
	return catalog.WithCache(c.(*catalog.Cache))
}

// build is New, failing the test on an error. t's own cache comes first, so
// that a test's own WithCache wins.
func build(t *testing.T, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	// A test that serves no models.dev document lists with it off, so that
	// none reaches models.opencode.ai (0065-MADR D1).
	if st, err := llmprovider.ResolveOptions(llmprovider.ProviderGrok, opts); err == nil && st.ModelMetadataURL() == "" {
		opts = append(opts, llmprovider.WithoutModelMetadata())
	}
	p, err := New(append([]llmprovider.Option{testCache(t)}, opts...)...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return p
}

// user is a user message.
func user(text string) llmprovider.MessageItem {
	return llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: text}
}

// items is a request with input, as the old GenerateItems sent.
func items(input ...llmprovider.Item) *llmprovider.Request {
	return &llmprovider.Request{Input: input}
}

// text is a request with one user message.
func text(prompt string) *llmprovider.Request { return items(user(prompt)) }

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

// effortOf is reasoning.effort in body, or nil.
func effortOf(body map[string]any) any {
	r, _ := body["reasoning"].(map[string]any)
	return r["effort"]
}
