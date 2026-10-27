package catalog

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// workspaceListing is an Anthropic listing of two pages, which records the
// anthropic-workspace-id of each request, "-" for none (0064-PLAN Phase 3).
type workspaceListing struct {
	*httptest.Server
	mu   sync.Mutex
	sent []string
}

func newWorkspaceListing(t *testing.T) *workspaceListing {
	t.Helper()
	s := &workspaceListing{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := "-"
		if v := r.Header.Values("anthropic-workspace-id"); len(v) > 0 {
			got = strings.Join(v, ",")
		}
		s.mu.Lock()
		s.sent = append(s.sent, got)
		s.mu.Unlock()
		if !strings.HasSuffix(r.URL.Path, "/models") {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("after_id") == "" {
			_, _ = w.Write([]byte(`{"data":[{"id":"claude-haiku-4-5","type":"model"}],"has_more":true,` +
				`"first_id":"claude-haiku-4-5","last_id":"claude-haiku-4-5"}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"claude-sonnet-4-5","type":"model"}],"has_more":false,` +
			`"first_id":"claude-sonnet-4-5","last_id":"claude-sonnet-4-5"}`))
	}))
	t.Cleanup(s.Close)
	return s
}

// take returns the workspaces sent since the last take.
func (s *workspaceListing) take() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	sent := s.sent
	s.sent = nil
	return sent
}

// TestClaudeWorkspace_Header_0064: with WithClaudeWorkspace, trimmed, every
// page of List and ListRaw carries anthropic-workspace-id; without it, none
// does (0064-MADR D2).
func TestClaudeWorkspace_Header_0064(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts []llmprovider.Option
		want string
	}{
		{"with", []llmprovider.Option{WithClaudeWorkspace(" wrkspc_A\n")}, "wrkspc_A"},
		{"without", nil, "-"},
		{"empty", []llmprovider.Option{WithClaudeWorkspace("")}, "-"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newWorkspaceListing(t)
			opts := append([]llmprovider.Option{testCache(t), llmprovider.WithBaseURL(s.URL)}, tc.opts...)
			if _, err := listT(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"), opts...); err != nil {
				t.Fatalf("List: %v", err)
			}
			if _, err := ListRaw(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"),
				append(opts, llmprovider.WithoutModelMetadata())...); err != nil {
				t.Fatalf("ListRaw: %v", err)
			}
			sent := s.take()
			if len(sent) != 4 {
				t.Fatalf("%d requests %q; want 4, two pages each for List and ListRaw", len(sent), sent)
			}
			for i, got := range sent {
				if got != tc.want {
					t.Errorf("request %d: workspace %q; want %q", i, got, tc.want)
				}
			}
		})
	}
}

// TestClaudeWorkspace_CacheKey_0064: one key's listings in two workspaces,
// and with none, are three entries; each is served again from the cache
// (0064-MADR D2, F3).
func TestClaudeWorkspace_CacheKey_0064(t *testing.T) {
	s := newWorkspaceListing(t)
	cache := testCache(t)
	list := func(opts ...llmprovider.Option) []string {
		t.Helper()
		opts = append([]llmprovider.Option{cache, llmprovider.WithBaseURL(s.URL)}, opts...)
		if _, err := listT(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"), opts...); err != nil {
			t.Fatalf("List: %v", err)
		}
		return s.take()
	}
	for round := range 2 {
		for _, tc := range []struct {
			opts []llmprovider.Option
			want string
		}{
			{[]llmprovider.Option{WithClaudeWorkspace("wrkspc_A")}, "wrkspc_A,wrkspc_A"},
			{[]llmprovider.Option{WithClaudeWorkspace("wrkspc_B")}, "wrkspc_B,wrkspc_B"},
			{nil, "-,-"},
		} {
			got := strings.Join(list(tc.opts...), ",")
			if round == 1 {
				tc.want = ""
			}
			if got != tc.want {
				t.Errorf("round %d: requests %q; want %q", round, got, tc.want)
			}
		}
	}
}

// TestClaudeWorkspace_Refused_0064: a workspace with a control character, and
// the option given to another provider's listing, are ErrInvalidRequest, and
// send nothing (0064-MADR D1, D2).
func TestClaudeWorkspace_Refused_0064(t *testing.T) {
	s := newWorkspaceListing(t)
	for _, tc := range []struct {
		name string
		id   llmprovider.ProviderID
		ws   string
		text string
	}{
		{"a control character", llmprovider.ProviderClaude, "wrkspc_\x00A", "control character"},
		{"an inner tab", llmprovider.ProviderClaude, "wrkspc_\tA", "control character"},
		{"another provider", llmprovider.ProviderOpenAI, "wrkspc_A", `catalog.WithClaudeWorkspace is for provider "claude", not "openai"`},
	} {
		_, err := listT(context.Background(), tc.id, llmprovider.NewStaticToken("k"), testCache(t),
			llmprovider.WithBaseURL(s.URL), WithClaudeWorkspace(tc.ws))
		if !errors.Is(err, llmprovider.ErrInvalidRequest) || !strings.Contains(err.Error(), tc.text) {
			t.Errorf("%s: %v; want ErrInvalidRequest naming %q", tc.name, err, tc.text)
		}
	}
	if sent := s.take(); len(sent) != 0 {
		t.Errorf("requests %q; want none", sent)
	}
}
