package providers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// pathRecorder serves G-wire's canned reply for every wire format and
// records each request's path.
type pathRecorder struct {
	mu    sync.Mutex
	paths []string
}

func (r *pathRecorder) serve(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		r.paths = append(r.paths, req.URL.Path)
		r.mu.Unlock()
		reply := wirecase.Case{}.Reply(req)
		w.WriteHeader(max(reply.Status, http.StatusOK))
		_, _ = w.Write([]byte(reply.Body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (r *pathRecorder) take() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.paths
	r.paths = nil
	return out
}

// firstModel is d's first static model, or a placeholder.
func firstModel(d llmprovider.Descriptor) string {
	if len(d.StaticModels) > 0 {
		return d.StaticModels[0]
	}
	return "test-model"
}

func hello() *llmprovider.Request {
	return &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}
}

// TestEveryProvider_TrailingSlashBaseURL (0020-MADR F30): a base URL ending
// in "/" requests the same paths as one without, on every provider's
// Generate and ListModels and on catalog's listing.
func TestEveryProvider_TrailingSlashBaseURL(t *testing.T) {
	rec := &pathRecorder{}
	srv := rec.serve(t)
	for _, d := range Default().Descriptors() {
		t.Run(string(d.ID), func(t *testing.T) {
			common := []llmprovider.Option{llmprovider.WithBaseURL(srv.URL + "/"), llmprovider.WithHTTPClient(srv.Client()),
				llmprovider.WithModelProbes(false), llmprovider.WithoutModelMetadata()}
			p, err := New(d.ID, append([]llmprovider.Option{llmprovider.WithAPIKey("slash-key"),
				llmprovider.WithModel(firstModel(d))}, common...)...)
			if err != nil {
				t.Fatal(err)
			}
			// Only the paths matter: a canned reply may not suit every call.
			_, _ = p.Generate(context.Background(), hello())
			if lister, ok := p.(llmprovider.ModelLister); ok {
				_, _ = lister.ListModels(context.Background())
			}
			_, _ = catalog.List(context.Background(), d.ID, llmprovider.NewStaticToken("slash-key"), common...)
			paths := rec.take()
			if len(paths) == 0 {
				t.Fatal("no request reached the server")
			}
			for _, path := range paths {
				if strings.Contains(path, "//") {
					t.Errorf("requested %q", path)
				}
			}
		})
	}
}

// TestEveryProvider_NilHTTPClient (0020-MADR F13): WithHTTPClient(nil) is the
// default client, so Generate reaches the service instead of panicking.
func TestEveryProvider_NilHTTPClient(t *testing.T) {
	rec := &pathRecorder{}
	srv := rec.serve(t)
	for _, d := range Default().Descriptors() {
		t.Run(string(d.ID), func(t *testing.T) {
			defer func() {
				if v := recover(); v != nil {
					t.Fatalf("Generate panicked: %v", v)
				}
			}()
			p, err := New(d.ID, llmprovider.WithAPIKey("nil-client-key"), llmprovider.WithModel(firstModel(d)),
				llmprovider.WithBaseURL(srv.URL), llmprovider.WithHTTPClient(nil), llmprovider.WithoutModelMetadata())
			if err != nil {
				t.Fatal(err)
			}
			_, _ = p.Generate(context.Background(), hello())
			if len(rec.take()) == 0 {
				t.Error("no request reached the server")
			}
		})
	}
}

// TestKeyedProviders_RefuseAnEmptyKey (0020-MADR F52): a provider with no
// anonymous access refuses an empty key at New. Kilo and the OpenCode
// gateways are left out: they send their anonymous key instead, by design.
func TestKeyedProviders_RefuseAnEmptyKey(t *testing.T) {
	for _, id := range []llmprovider.ProviderID{llmprovider.ProviderOpenAI, llmprovider.ProviderGemini,
		llmprovider.ProviderClaude, llmprovider.ProviderGrok, llmprovider.ProviderHuggingFace, llmprovider.ProviderTogether} {
		d, ok := Default().Descriptor(id)
		if !ok {
			t.Fatalf("no descriptor for %s", id)
		}
		if _, err := New(id, llmprovider.WithAPIKey(""), llmprovider.WithModel(firstModel(d))); !errors.Is(err, llmprovider.ErrInvalidRequest) {
			t.Errorf("%s: New with an empty key = %v, want ErrInvalidRequest", id, err)
		}
	}
}

// TestCapabilities_DegradedAreBestEffort (0020-MADR F43, R10): a capability
// whose request is degraded is BestEffort, not Supported.
func TestCapabilities_DegradedAreBestEffort(t *testing.T) {
	for _, c := range []struct {
		id   llmprovider.ProviderID
		name string
		get  func(llmprovider.Capabilities) llmprovider.Support
	}{
		// With reasoning, a forced tool is sent as "auto".
		{llmprovider.ProviderClaude, "ForcedToolChoice", func(c llmprovider.Capabilities) llmprovider.Support { return c.ForcedToolChoice }},
		// Reasoning is dropped for models off the Grok CLI's menu.
		{llmprovider.ProviderGrok, "Reasoning", func(c llmprovider.Capabilities) llmprovider.Support { return c.Reasoning }},
		// Only a named tool was measured.
		{llmprovider.ProviderHuggingFace, "ForcedToolChoice", func(c llmprovider.Capabilities) llmprovider.Support { return c.ForcedToolChoice }},
	} {
		d, ok := Default().Descriptor(c.id)
		if !ok {
			t.Fatalf("no descriptor for %s", c.id)
		}
		p, err := New(c.id, llmprovider.WithAPIKey("caps-key"), llmprovider.WithModel(firstModel(d)))
		if err != nil {
			t.Fatal(err)
		}
		if got := c.get(p.Capabilities()); got != llmprovider.BestEffort {
			t.Errorf("%s %s = %v, want BestEffort", c.id, c.name, got)
		}
	}
}
