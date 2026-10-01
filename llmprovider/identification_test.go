package llmprovider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
)

// headerRecorder is a test server that records every request's headers and
// answers a listing and a metadata document, and a 500 to everything else.
type headerRecorder struct {
	mu   sync.Mutex
	seen []recordedRequest
	srv  *httptest.Server
}

type recordedRequest struct {
	path   string
	header http.Header
}

func newHeaderRecorder(t *testing.T) *headerRecorder {
	t.Helper()
	r := &headerRecorder{}
	r.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		r.seen = append(r.seen, recordedRequest{path: req.URL.Path, header: req.Header.Clone()})
		r.mu.Unlock()
		switch {
		case strings.HasSuffix(req.URL.Path, "/api.json"):
			_, _ = w.Write([]byte(`{}`))
		case strings.HasSuffix(req.URL.Path, "/models"):
			_, _ = w.Write([]byte(`{"data":[{"id":"glm-5.3-flash"},{"id":"qwen3.8-flash"}]}`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(r.srv.Close)
	return r
}

func (r *headerRecorder) requests() []recordedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]recordedRequest(nil), r.seen...)
}

var userAgentPattern = regexp.MustCompile(`^go-llmprovider-sdk/\S+ \(\w+; \w+\) go-llmprovider-sdk/\S+$`)

// TestIdentification_UserAgent: every generation and listing request names
// this module honestly (MADR 0012 §1.4), never Go's default agent.
func TestIdentification_UserAgent(t *testing.T) {
	rec := newHeaderRecorder(t)
	base := WithBaseURL(rec.srv.URL)
	build := map[string]func() (LegacyProvider, error){
		"ollama": func() (LegacyProvider, error) { return NewOllama("", "llama3", base) },
	}
	for name, newProvider := range build {
		p, err := newProvider()
		if err != nil {
			t.Fatalf("%s: construct: %v", name, err)
		}
		_, _ = p.Generate(context.Background(), "hello")
	}
	if _, err := ListModelCatalog(context.Background(), ProviderKilo, "k", base); err != nil {
		t.Fatalf("listing: %v", err)
	}
	for _, r := range rec.requests() {
		if ua := r.header.Get("User-Agent"); !userAgentPattern.MatchString(ua) {
			t.Errorf("%s: User-Agent = %q, want go-llmprovider-sdk/<version> (<os>; <arch>) go-llmprovider-sdk/<version>", r.path, ua)
		}
	}
}
