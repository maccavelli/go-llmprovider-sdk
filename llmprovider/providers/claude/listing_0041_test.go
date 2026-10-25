package claude

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestListModels_DefaultBaseShape_0041 (0041-MADR A8): with a base URL in
// the provider's own form, ending in /v1 as its default does, ListModels
// lists live from /v1/models, and a request after it reads the listing's
// facts. The request's Budget gives an enabled budget with no facts; its
// listing's efforts give adaptive thinking (0062-MADR D2).
func TestListModels_DefaultBaseShape_0041(t *testing.T) {
	var mu sync.Mutex
	var paths []string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/v1/models":
			_, _ = w.Write([]byte(`{"data":[{"id":"claude-sonnet-4-5-0041","capabilities":` +
				`{"effort":{"supported":true,"low":{"supported":true}}}}],"has_more":false}`))
		case "/v1/messages":
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}]}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()
	p := build(t, apiKey(srv.URL+"/v1", "claude-sonnet-4-5-0041", llmprovider.WithModelProbes(false),
		llmprovider.WithModelMetadataURL(srv.URL+"/api.json"))...)
	models, err := list(t, p)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(models, "claude-sonnet-4-5-0041") {
		mu.Lock()
		t.Errorf("ListModels %q after %q; want the live listing's model", models, paths)
		mu.Unlock()
	}
	req := text("hi")
	req.Reasoning = &llmprovider.Reasoning{Effort: llmprovider.EffortLow, Budget: 2000}
	if _, err := p.Generate(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	thinking, _ := body["thinking"].(map[string]any)
	if thinking["type"] != "adaptive" {
		t.Errorf("thinking %v; want adaptive, from the listing's efforts", body["thinking"])
	}
}
