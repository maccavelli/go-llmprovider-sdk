package openai

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// Test helpers, moved from llmprovider's tests with the provider (0015-PLAN
// S7). Their behaviour is unchanged.

const okResponse = `{"output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

// httpResponse answers request; a ChatGPT request (Accept: text/event-stream)
// gets a Responses body reframed as its event stream.
func httpResponse(request *http.Request, status int, body string) *http.Response {
	if status == http.StatusOK && request != nil && request.Header.Get("Accept") == "text/event-stream" {
		body = stream(body)
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    request,
	}
}

// stream reframes a Responses JSON body as the event stream a ChatGPT session
// reads (MADR 0012 §4.1): one output_item.done per output item, then
// response.completed.
func stream(body string) string {
	var parsed struct {
		ID     string            `json:"id"`
		Model  string            `json:"model"`
		Output []json.RawMessage `json:"output"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return body
	}
	var b strings.Builder
	for _, item := range parsed.Output {
		b.WriteString(`data: {"type":"response.output_item.done","item":` + string(item) + "}\n\n")
	}
	id, _ := json.Marshal(parsed.ID)
	model, _ := json.Marshal(parsed.Model)
	b.WriteString(`data: {"type":"response.completed","response":{"id":` + string(id) + `,"model":` + string(model) + "}}\n\n")
	return b.String()
}

func testJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal JWT claims: %v", err)
	}
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
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

// capture records what a stubbed ChatGPT backend received.
type capture struct {
	mu     sync.Mutex
	calls  int
	header http.Header
	body   map[string]any
}

// chatGPTClient answers every request with reply and no Content-Type, as the
// ChatGPT backend sends its event stream (gate G-C, 2026-09-27).
func chatGPTClient(t *testing.T, reply string) (*http.Client, *capture) {
	t.Helper()
	c := &capture{}
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		c.mu.Lock()
		c.calls++
		c.header, c.body = r.Header.Clone(), body
		c.mu.Unlock()
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{},
			Body: io.NopCloser(strings.NewReader(reply)), Request: r}, nil
	})}, c
}

func chatGPTSession() *auth.OAuthSession {
	return &auth.OAuthSession{Issuer: auth.DefaultOpenAIIssuer, Access: "sess", Refresh: "refresh",
		Expiry: time.Now().Add(time.Hour)}
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
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

// thinking adds reasoning at the provider's effort, as GenerateThinking did.
func thinking(req *llmprovider.Request) *llmprovider.Request {
	req.Reasoning = &llmprovider.Reasoning{}
	return req
}
