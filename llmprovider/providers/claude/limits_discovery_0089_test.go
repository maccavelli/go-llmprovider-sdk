package claude

import (
	"context"
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

// TestDiscover_RequestReadsDiscovery_0089 (0089-MADR DL72, DL75): Claude in
// a workspace: catalog.LookupLimits given catalog.WithClaudeWorkspace, then
// a request through a provider built with the same options, the same key
// and claude.WithWorkspace: the request reads the listing the lookup stored,
// its 128,000 cutting WithMaxTokens(200000), and sends no listing request.
// It is here, not in catalog, which cannot build a provider that imports it
// (0089-PLAN deviation 3).
func TestDiscover_RequestReadsDiscovery_0089(t *testing.T) {
	const workspace = "wrkspc_A"
	var mu sync.Mutex
	listings, workspaces := 0, map[string]bool{}
	var sent map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		workspaces[r.Header.Get("anthropic-workspace-id")] = true
		switch {
		case strings.HasSuffix(r.URL.Path, "/v1/models"):
			listings++
			_, _ = io.WriteString(w, `{"data":[{"id":"claude-opus-4-6","max_tokens":128000,"max_input_tokens":1000000}],"has_more":false}`)
		case strings.HasSuffix(r.URL.Path, "/messages"):
			raw, _ := io.ReadAll(r.Body)
			sent = map[string]any{}
			_ = json.Unmarshal(raw, &sent)
			_, _ = io.WriteString(w, okReply)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	cache := new(catalog.Cache)
	t.Cleanup(func() { _ = cache.Close(context.Background()) })
	shared := []llmprovider.Option{llmprovider.WithBaseURL(srv.URL + "/v1"),
		catalog.WithCache(cache), catalog.WithClaudeWorkspace(workspace)}

	l, err := catalog.LookupLimits(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"),
		"claude-opus-4-6", shared...)
	if err != nil || l.Output.Tokens != 128000 {
		t.Fatalf("LookupLimits = %+v, %v; want 128,000", l.Output, err)
	}
	mu.Lock()
	before := listings
	mu.Unlock()

	p := build(t, append(shared, llmprovider.WithAPIKey("k"), llmprovider.WithModel("claude-opus-4-6"),
		llmprovider.WithMaxTokens(200000), WithWorkspace(workspace))...)
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if listings != before {
		t.Errorf("listing requests during the request: %d; want none", listings-before)
	}
	if got := sent["max_tokens"]; got != float64(128000) {
		t.Errorf("max_tokens = %v; want 128000, the listing's limit the lookup stored", got)
	}
	if !workspaces[workspace] || workspaces[""] {
		t.Errorf("workspaces sent = %v; want only %s", workspaces, workspace)
	}
}
