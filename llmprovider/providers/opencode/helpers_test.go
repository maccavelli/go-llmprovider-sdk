package opencode

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Test helpers, as the other provider packages' (0015-PLAN S7).

// Per-route response fixtures. The responses one is the shape measured on Zen:
// summary is empty and the trace lives in encrypted_content, so the shared
// decoder yields a present-but-blank ReasoningItem.
const (
	fxResponses = `{"id":"resp_1","output":[
		{"type":"reasoning","summary":[],"encrypted_content":"Q-PaDgFF"},
		{"type":"message","content":[{"type":"output_text","text":"hello"}]}]}`
	fxMessages = `{"content":[{"type":"text","text":"hello"}]}`
	fxGoogle   = `{"candidates":[{"content":{"parts":[{"text":"hello"}]}}]}`
	fxChat     = `{"id":"router-1","choices":[{"message":{"role":"assistant","content":"hello"}}]}`

	deepSeekV4Pro = "deepseek-v4-pro"
)

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

// pathCapture records the request path and replies with a canned body.
func pathCapture(t *testing.T, lastPath *string, resp string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*lastPath = r.URL.Path
		_, _ = w.Write([]byte(resp))
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

// newFor is the constructor for gateway.
func newFor(gateway llmprovider.ProviderID) llmprovider.Factory {
	if gateway == llmprovider.ProviderOpencodeGo {
		return NewGo
	}
	return NewZen
}

// build is gateway's constructor, failing the test on an error.
func build(t *testing.T, gateway llmprovider.ProviderID, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := newFor(gateway)(append(metadataOff(), opts...)...)
	if err != nil {
		t.Fatalf("New %s: %v", gateway, err)
	}
	return p
}

// apiKey is the options for a key "k" against url, with the given model.
func apiKey(url, model string, opts ...llmprovider.Option) []llmprovider.Option {
	return append(append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModel(model),
		llmprovider.WithBaseURL(url)}, metadataOff()...), opts...)
}

// effort is WithReasoning with an effort, as the old WithReasoningEffort.
func effort(e llmprovider.Effort) llmprovider.Option {
	return llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: e})
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

// thinking adds reasoning at the provider's defaults, as GenerateThinking did.
func thinking(req *llmprovider.Request) *llmprovider.Request {
	req.Reasoning = &llmprovider.Reasoning{}
	return req
}

// deadlineTransport records how long each first request's context had left
// (-1 without a deadline), by method and path.
type deadlineTransport struct {
	mu   sync.Mutex
	seen map[string]time.Duration
}

func (d *deadlineTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	left := time.Duration(-1)
	if dl, ok := r.Context().Deadline(); ok {
		left = time.Until(dl)
	}
	key := r.Method + " " + r.URL.Path
	d.mu.Lock()
	if d.seen == nil {
		d.seen = map[string]time.Duration{}
	}
	if _, ok := d.seen[key]; !ok {
		d.seen[key] = left
	}
	d.mu.Unlock()
	return http.DefaultTransport.RoundTrip(r)
}

func (d *deadlineTransport) left(key string) (time.Duration, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	v, ok := d.seen[key]
	return v, ok
}

// headerRecorder records every request's path and headers, answering each
// route's shape with "ok".
type headerRecorder struct {
	srv  *httptest.Server
	mu   sync.Mutex
	seen []recorded
}

type recorded struct {
	path   string
	header http.Header
}

func newHeaderRecorder(t *testing.T) *headerRecorder {
	t.Helper()
	r := &headerRecorder{}
	r.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		r.seen = append(r.seen, recorded{path: req.URL.Path, header: req.Header.Clone()})
		r.mu.Unlock()
		switch {
		case req.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[{"id":"glm-5.3-flash"}]}`))
		case len(req.URL.Path) > 9 && req.URL.Path[len(req.URL.Path)-9:] == "/messages":
			_, _ = w.Write([]byte(fxMessages))
		default:
			_, _ = w.Write([]byte(fxChat))
		}
	}))
	t.Cleanup(r.srv.Close)
	return r
}

func (r *headerRecorder) requests() []recorded {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]recorded(nil), r.seen...)
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
