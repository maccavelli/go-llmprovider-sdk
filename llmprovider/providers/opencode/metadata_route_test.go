package opencode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from llmprovider's opencode_metadata_route_test.go (0015-PLAN S7).
// TestIsUsableOpencodeModel_DeniesSystemone stays there, with the catalog.

// metadataKey is the document's key for a gateway.
func metadataKey(gateway llmprovider.ProviderID) string {
	if gateway == llmprovider.ProviderOpencodeGo {
		return "opencode-go"
	}
	return "opencode"
}

// routeServer serves a metadata document listing model under gateway's
// section, with provider.npm set when npm is not empty (listed=false serves a
// document without the model), and answers every generation route with "ok",
// recording the request paths.
func routeServer(t *testing.T, gateway llmprovider.ProviderID, model, npm string, listed bool) (*httptest.Server, func() []string) {
	t.Helper()
	enableMetadata(t)
	var (
		mu    sync.Mutex
		paths []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/api.json") {
			entry := `"other-model":{"id":"other-model"}`
			if listed {
				entry = `"` + model + `":{"id":"` + model + `"`
				if npm != "" {
					entry += `,"provider":{"npm":"` + npm + `"}`
				}
				entry += "}"
			}
			_, _ = w.Write([]byte(`{"` + metadataKey(gateway) + `":{"models":{` + entry + `}}}`))
			return
		}
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		switch {
		case strings.HasSuffix(r.URL.Path, "/messages"):
			_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}]}`))
		case strings.HasSuffix(r.URL.Path, "/responses"):
			_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`))
		case strings.HasSuffix(r.URL.Path, ":generateContent"):
			_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`))
		default:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), paths...)
	}
}

// generatePath runs one Generate and returns the single request path it sent.
func generatePath(t *testing.T, srv *httptest.Server, paths func() []string, gateway llmprovider.ProviderID, model string, opts ...llmprovider.Option) string {
	t.Helper()
	opts = append([]llmprovider.Option{llmprovider.WithModelMetadataURL(srv.URL + "/api.json")}, opts...)
	if _, err := build(t, gateway, apiKey(srv.URL, model, opts...)...).Generate(context.Background(), text("hi")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	got := paths()
	if len(got) != 1 {
		t.Fatalf("requests = %v, want exactly one", got)
	}
	return got[0]
}

// TestOpencode_RoutesFromMetadata: the metadata's provider.npm picks the
// route, as OpenCode's client does (provider.ts:1274-1278), for a model the
// table does not know and for one it routes differently.
func TestOpencode_RoutesFromMetadata(t *testing.T) {
	zen, goGW := llmprovider.ProviderID(llmprovider.ProviderOpencodeZen), llmprovider.ProviderID(llmprovider.ProviderOpencodeGo)
	tests := []struct {
		name    string
		gateway llmprovider.ProviderID
		model   string
		npm     string
		want    string
	}{
		{"anthropic", goGW, "brand-new-model", "@ai-sdk/anthropic", "/messages"},
		{"openai", goGW, "brand-new-model", "@ai-sdk/openai", "/responses"},
		{"google", zen, "brand-new-model", "@ai-sdk/google", "/models/brand-new-model:generateContent"},
		{"compatible", goGW, "brand-new-model", "@ai-sdk/openai-compatible", "/chat/completions"},
		{"unset", goGW, "brand-new-model", "", "/chat/completions"},
		// The table says messages; the metadata wins.
		{"over table", goGW, "minimax-m3", "@ai-sdk/openai-compatible", "/chat/completions"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv, paths := routeServer(t, tc.gateway, tc.model, tc.npm, true)
			if got := generatePath(t, srv, paths, tc.gateway, tc.model); got != tc.want {
				t.Errorf("path = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestOpencode_RouteOverrideBeatsMetadata: WithRoute wins over the metadata.
func TestOpencode_RouteOverrideBeatsMetadata(t *testing.T) {
	srv, paths := routeServer(t, llmprovider.ProviderOpencodeGo, "brand-new-model", "@ai-sdk/anthropic", true)
	got := generatePath(t, srv, paths, llmprovider.ProviderOpencodeGo, "brand-new-model", WithRoute(RouteChatCompletions))
	if got != "/chat/completions" {
		t.Errorf("path = %q, want the override's /chat/completions", got)
	}
}

// TestOpencode_RouteWithoutMetadataUsesTable: metadata that is disabled, or
// that does not list the model, leaves the table's route.
func TestOpencode_RouteWithoutMetadataUsesTable(t *testing.T) {
	t.Run("absent from document", func(t *testing.T) {
		srv, paths := routeServer(t, llmprovider.ProviderOpencodeGo, "minimax-m3", "", false)
		if got := generatePath(t, srv, paths, llmprovider.ProviderOpencodeGo, "minimax-m3"); got != "/messages" {
			t.Errorf("path = %q, want the table's /messages", got)
		}
	})
	t.Run("disabled", func(t *testing.T) {
		srv, paths := routeServer(t, llmprovider.ProviderOpencodeGo, "minimax-m3", "@ai-sdk/openai-compatible", true)
		t.Setenv(envDisableMetadata, "1")
		if got := generatePath(t, srv, paths, llmprovider.ProviderOpencodeGo, "minimax-m3"); got != "/messages" {
			t.Errorf("path = %q, want the table's /messages", got)
		}
	})
}

// TestOpencodeRouteTable_MatchesMetadataSnapshot: every active Zen and Go
// model in testdata/opencode-routes.json (provider.npm per model, from
// models.opencode.ai/api.json on 2026-09-27, deprecated models omitted)
// resolves without metadata to the route its npm selects. It pins the table's
// refresh; the oracle below is written independently of the production map.
func TestOpencodeRouteTable_MatchesMetadataSnapshot(t *testing.T) {
	raw, err := os.ReadFile("testdata/opencode-routes.json")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot map[string]map[string]string
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatal(err)
	}
	oracle := map[string]Route{
		"@ai-sdk/openai":    RouteResponses,
		"@ai-sdk/anthropic": RouteMessages,
		"@ai-sdk/google":    RouteGoogle,
	}
	gateways := map[string]llmprovider.ProviderID{"opencode": llmprovider.ProviderOpencodeZen, "opencode-go": llmprovider.ProviderOpencodeGo}
	for section, models := range snapshot {
		gateway, ok := gateways[section]
		if !ok || len(models) == 0 {
			t.Fatalf("snapshot section %q: unknown or empty", section)
		}
		for model, npm := range models {
			want, ok := oracle[npm]
			if !ok {
				want = RouteChatCompletions
			}
			got, err := resolveRoute(gateway, model, "")
			if err != nil || got != want {
				t.Errorf("%s %s (npm %q): route = %q, %v; want %q", gateway, model, npm, got, err, want)
			}
		}
	}
}
