package gemini

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Test helpers, as the claude and openai packages' (0015-PLAN S7).

// interactionText is an Interaction answering "ok".
const interactionText = `{"id":"v1_int_1","status":"completed","steps":[{"type":"model_output","content":[{"type":"text","text":"ok"}]}]}`

// interactionCapture records the requests a stub Interactions server received.
type interactionCapture struct {
	mu      sync.Mutex
	paths   []string
	bodies  []map[string]any
	headers []http.Header
}

func (c *interactionCapture) last() (string, map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.paths) == 0 {
		return "", nil
	}
	return c.paths[len(c.paths)-1], c.bodies[len(c.bodies)-1]
}

func (c *interactionCapture) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.paths)
}

// interactionServer answers every request with reply and records it.
func interactionServer(t *testing.T, reply string) (*httptest.Server, *interactionCapture) {
	t.Helper()
	c := &interactionCapture{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if raw, _ := io.ReadAll(r.Body); len(raw) > 0 {
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Errorf("decode: %v", err)
			}
		}
		c.mu.Lock()
		c.paths, c.bodies, c.headers = append(c.paths, r.URL.Path), append(c.bodies, body), append(c.headers, r.Header.Clone())
		c.mu.Unlock()
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	return srv, c
}

// stepTypes lists the input steps' types.
func stepTypes(body map[string]any) []string {
	var out []string
	steps, _ := body["input"].([]any)
	for _, s := range steps {
		out = append(out, fmt.Sprint(s.(map[string]any)["type"]))
	}
	return out
}

// generationConfig is the body's generation_config.
func generationConfig(body map[string]any) map[string]any {
	gen, _ := body["generation_config"].(map[string]any)
	return gen
}

var weatherTool = llmprovider.Tool{Name: "get_weather", Description: "Weather for a city",
	Schema: map[string]any{"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}}}

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

// user is a user message.
func user(text string) llmprovider.MessageItem {
	return llmprovider.MessageItem{Role: string(llmprovider.RoleUser), Text: text}
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
