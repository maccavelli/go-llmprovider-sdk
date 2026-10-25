package chatcompat

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// Test helpers, as the other provider packages' (0015-PLAN S7).

// testID is the id the tests' providers take.
const testID llmprovider.ProviderID = "acme"

const fxChat = `{"id":"cmpl-acme","model":"llmtest-model","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"hello"}}]}`

// server records each request's method, path, Authorization header and JSON
// body, and answers by reply, else with fxChat.
type server struct {
	*httptest.Server
	mu       sync.Mutex
	requests []request
}

type request struct {
	method, path, authorization string
	body                        map[string]any
}

func newServer(t *testing.T, reply func(n int, r *http.Request) (int, string)) *server {
	t.Helper()
	s := &server{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		s.mu.Lock()
		s.requests = append(s.requests, request{r.Method, r.URL.Path, r.Header.Get("Authorization"), body})
		n := len(s.requests)
		s.mu.Unlock()
		status, text := http.StatusOK, fxChat
		if reply != nil {
			status, text = reply(n, r)
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, text)
	}))
	t.Cleanup(s.Close)
	return s
}

// seen is a copy of the requests so far.
func (s *server) seen() []request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]request(nil), s.requests...)
}

// caches holds each running test's catalog cache, by *testing.T.
var caches sync.Map

// testCache is t's own catalog cache, the same for every call in t
// (0058-MADR D1).
func testCache(t *testing.T) llmprovider.Option {
	t.Helper()
	c, loaded := caches.LoadOrStore(t, new(catalog.Cache))
	if !loaded {
		t.Cleanup(func() { caches.Delete(t) })
	}
	return catalog.WithCache(c.(*catalog.Cache))
}

// build is New with testID, base url and model "m", t's own cache first so
// that a test's own WithCache wins, failing the test on an error. It turns
// the models.dev document off, which a listing fetches since 0061-PLAN Phase
// 4c (decision 4), so that no test reaches models.opencode.ai.
func build(t *testing.T, url string, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := New(append([]llmprovider.Option{testCache(t), WithID(testID), llmprovider.WithBaseURL(url),
		llmprovider.WithModel("m"), llmprovider.WithoutModelMetadata()}, opts...)...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return p
}

// text is a request with one user message.
func text(prompt string) *llmprovider.Request {
	return &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: prompt}}}
}
