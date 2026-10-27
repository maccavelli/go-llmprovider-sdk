package claude

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
)

// workspaceRequest is one request a workspaceServer answered: what it was
// (list, stream or post, a generation or a probe) and the workspace it named,
// "-" for none.
type workspaceRequest struct{ kind, workspace string }

// workspaceServer is Anthropic for 0064-PLAN Phase 3: a listing of two pages,
// an event stream, and a reply of "Hello" to any other message.
type workspaceServer struct {
	*httptest.Server
	mu   sync.Mutex
	seen []workspaceRequest
}

func newWorkspaceServer(t *testing.T) *workspaceServer {
	t.Helper()
	s := &workspaceServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := workspaceRequest{kind: "post", workspace: "-"}
		if v := r.Header.Values(headerWorkspace); len(v) > 0 {
			req.workspace = strings.Join(v, ",")
		}
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/models"):
			req.kind = "list"
		case r.Header.Get("Accept") == "text/event-stream":
			req.kind = "stream"
		}
		s.mu.Lock()
		s.seen = append(s.seen, req)
		s.mu.Unlock()
		switch req.kind {
		case "list":
			if r.URL.Query().Get("after_id") == "" {
				_, _ = w.Write([]byte(`{"data":[{"id":"claude-haiku-4-5","type":"model"}],"has_more":true,"last_id":"claude-haiku-4-5"}`))
				return
			}
			_, _ = w.Write([]byte(`{"data":[{"id":"claude-sonnet-4-5","type":"model"}],"has_more":false,"last_id":"claude-sonnet-4-5"}`))
		case "stream":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte(messagesStream))
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"model":"claude-haiku-4-5","stop_reason":"end_turn","content":[{"type":"text","text":"Hello"}]}`))
		}
	}))
	t.Cleanup(s.Close)
	return s
}

// take returns the requests answered since the last take.
func (s *workspaceServer) take() []workspaceRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := s.seen
	s.seen = nil
	return seen
}

// count is how many of seen are of kind, and how many of those named
// workspace.
func count(seen []workspaceRequest, kind, workspace string) (all, named int) {
	for _, r := range seen {
		if r.kind == kind {
			all++
			if r.workspace == workspace {
				named++
			}
		}
	}
	return all, named
}

// TestWorkspace_Header_0064: with WithWorkspace, trimmed, Generate, Stream,
// each probe and each listing page carry anthropic-workspace-id; without it,
// or with it empty, none does (0064-MADR D1, D2).
func TestWorkspace_Header_0064(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts []llmprovider.Option
		want string
	}{
		{"with", []llmprovider.Option{WithWorkspace("  wrkspc_A\n")}, "wrkspc_A"},
		{"without", nil, "-"},
		{"empty", []llmprovider.Option{WithWorkspace(" ")}, "-"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newWorkspaceServer(t)
			p := build(t, apiKey(s.URL, "claude-haiku-4-5", append(tc.opts, llmprovider.WithModelProbes(true))...)...)
			if _, err := p.Generate(context.Background(), text("hi")); err != nil {
				t.Fatalf("Generate: %v", err)
			}
			for _, err := range p.(llmprovider.Streamer).Stream(context.Background(), text("hi")) {
				if err != nil {
					t.Fatalf("Stream: %v", err)
				}
			}
			if models, err := list(t, p); err != nil || len(models) == 0 {
				t.Fatalf("ListModels = %v, %v; want models", models, err)
			}
			seen := s.take()
			for _, kind := range []string{"list", "stream", "post"} {
				all, named := count(seen, kind, tc.want)
				// Two listing pages; one stream; the generation and a probe
				// of each of the two listed models.
				want := map[string]int{"list": 2, "stream": 1, "post": 3}[kind]
				if all != want || named != all {
					t.Errorf("%s: %d requests, %d naming %q; want %d, all naming it (%v)", kind, all, named, tc.want, want, seen)
				}
			}
		})
	}
}

// TestWorkspace_ProbeAndListingScope_0064: one key's listings and probes in
// two workspaces, through one cache, are kept apart, and each is served from
// the cache when asked again (0064-MADR D2; its amendment "the probe cache").
func TestWorkspace_ProbeAndListingScope_0064(t *testing.T) {
	s := newWorkspaceServer(t)
	var cache catalog.Cache
	provider := func(workspace string) llmprovider.Provider {
		return build(t, apiKey(s.URL, "claude-haiku-4-5", WithWorkspace(workspace), llmprovider.WithModelProbes(true),
			catalog.WithCache(&cache))...)
	}
	a, b := provider("wrkspc_A"), provider("wrkspc_B")
	for i, step := range []struct {
		p         llmprovider.Provider
		workspace string
		fresh     bool
	}{{a, "wrkspc_A", true}, {b, "wrkspc_B", true}, {a, "wrkspc_A", false}, {b, "wrkspc_B", false}} {
		if models, err := list(t, step.p); err != nil || len(models) == 0 {
			t.Fatalf("step %d: ListModels = %v, %v; want models", i, models, err)
		}
		seen := s.take()
		lists, listsNamed := count(seen, "list", step.workspace)
		probes, probesNamed := count(seen, "post", step.workspace)
		switch {
		case step.fresh && (lists != 2 || listsNamed != 2 || probes != 2 || probesNamed != 2):
			t.Errorf("step %d (%s): %v; want its own two pages and two probes", i, step.workspace, seen)
		case !step.fresh && len(seen) != 0:
			t.Errorf("step %d (%s): %v; want none, both cached", i, step.workspace, seen)
		}
	}
}

// TestWorkspace_Refused_0064: a workspace holding a control character fails
// New with ErrInvalidRequest; the option given to another provider is
// refused as ScopedOption refuses (0064-MADR D1).
func TestWorkspace_Refused_0064(t *testing.T) {
	for _, ws := range []string{"wrkspc_\x00A", "wrkspc_\tA", "wrkspc_A\r\nX-Injected: 1"} {
		_, err := New(llmprovider.WithAPIKey("k"), WithWorkspace(ws))
		if !errors.Is(err, llmprovider.ErrInvalidRequest) || !strings.Contains(err.Error(), "control character") {
			t.Errorf("New(WithWorkspace(%q)) = %v; want ErrInvalidRequest naming the control character", ws, err)
		}
	}
	_, err := llmprovider.ResolveOptions(llmprovider.ProviderOpenAI, []llmprovider.Option{WithWorkspace("wrkspc_A")})
	if !errors.Is(err, llmprovider.ErrInvalidRequest) ||
		!strings.Contains(err.Error(), `claude.WithWorkspace is for provider "claude", not "openai"`) {
		t.Errorf("openai with claude.WithWorkspace: %v; want ErrInvalidRequest naming the option", err)
	}
}
