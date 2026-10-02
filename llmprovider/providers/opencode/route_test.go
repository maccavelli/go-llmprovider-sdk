package opencode

import (
	"errors"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from llmprovider's opencode_route_test.go (0015-PLAN S7). The base
// URL and identifier tests stay there, with the code they test.

// TestOpencodeRoute_Table pins the published endpoint tables, including the
// per-gateway divergence that makes a model-only key wrong.
func TestOpencodeRoute_Table(t *testing.T) {
	zen, goGW := llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo
	tests := []struct {
		name    string
		gateway llmprovider.ProviderID
		model   string
		want    Route
	}{
		// The divergence proof: same model id, different route per gateway.
		{"zen minimax is chat", zen, "minimax-m3", RouteChatCompletions},
		{"go minimax is messages", goGW, "minimax-m3", RouteMessages},

		{"zen gpt", zen, "gpt-5.5", RouteResponses},
		{"zen grok", zen, "grok-4.6", RouteResponses},
		{"zen muse", zen, "muse-spark-1.2", RouteResponses},
		{"zen claude", zen, "claude-opus-4-5", RouteMessages},
		{"zen qwen", zen, "qwen3.6-plus", RouteMessages},
		{"zen gemini", zen, "gemini-3.7-flash", RouteGoogle},
		{"zen deepseek", zen, deepSeekV4Pro, RouteChatCompletions},
		{"zen glm", zen, "glm-5.2", RouteChatCompletions},
		{"zen kimi", zen, "kimi-k3", RouteChatCompletions},
		{"zen big-pickle", zen, "big-pickle", RouteChatCompletions},

		{"go gpt", goGW, "gpt-5.6-luna", RouteResponses},
		{"go glm", goGW, "glm-5.3", RouteChatCompletions},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveRoute(tc.gateway, tc.model, "")
			if err != nil {
				t.Fatalf("resolveRoute(%q,%q): %v", tc.gateway, tc.model, err)
			}
			if got != tc.want {
				t.Errorf("resolveRoute(%q,%q) = %q, want %q", tc.gateway, tc.model, got, tc.want)
			}
		})
	}
}

// TestOpencodeRoute_Heuristic covers the ten model ids that are live on the
// gateways but absent from the published tables (0004-PLAN §3.3). This is the
// regression guard for catalog drift: every one must still resolve correctly
// through the prefix heuristic.
func TestOpencodeRoute_Heuristic(t *testing.T) {
	zen, goGW := llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo
	tests := []struct {
		gateway llmprovider.ProviderID
		model   string
		want    Route
	}{
		{zen, "claude-sonnet-9", RouteMessages},
		{zen, "deepseek-v4-flash-free", RouteChatCompletions},
		{zen, "laguna-s-2.1-free", RouteChatCompletions},
		{goGW, "kimi-k2.5", RouteChatCompletions},
		{goGW, "glm-5", RouteChatCompletions},
		{goGW, "mimo-v2-pro", RouteChatCompletions},
		{goGW, "mimo-v2-omni", RouteChatCompletions},
		{goGW, "hy3-preview", RouteChatCompletions},
		{goGW, "qwen3.5-plus", RouteMessages},
		{goGW, "grok-4.5", RouteResponses},
	}
	for _, tc := range tests {
		t.Run(string(tc.gateway)+"/"+tc.model, func(t *testing.T) {
			// Guard the premise: these must NOT be in the table, or the test
			// is asserting table lookups rather than the heuristic.
			if _, tabled := routeTable[tc.gateway][tc.model]; tabled {
				t.Fatalf("%q is in the route table for %q; move this case to TestOpencodeRoute_Table",
					tc.model, tc.gateway)
			}
			if got := heuristicRoute(tc.gateway, tc.model); got != tc.want {
				t.Errorf("heuristicRoute(%q,%q) = %q, want %q", tc.gateway, tc.model, got, tc.want)
			}
		})
	}
}

// TestOpencodeRoute_Override verifies an explicit route beats both the table
// and the heuristic, and that an unknown route is rejected.
func TestOpencodeRoute_Override(t *testing.T) {
	zen := llmprovider.ProviderOpencodeZen
	// gpt-5.5 is tabled as responses; the override must win.
	got, err := resolveRoute(zen, "gpt-5.5", RouteChatCompletions)
	if err != nil {
		t.Fatalf("override: %v", err)
	}
	if got != RouteChatCompletions {
		t.Errorf("override = %q, want %q", got, RouteChatCompletions)
	}
	if _, err := resolveRoute(zen, "gpt-5.5", Route("bogus")); err == nil {
		t.Error("expected error for invalid route override")
	} else if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("invalid override error = %v, want wrapping ErrInvalidRequest", err)
	}
}

// TestOpencodeRoute_Path pins the request path suffix per route.
func TestOpencodeRoute_Path(t *testing.T) {
	if got := RouteGoogle.path("gemini-3.7-flash"); got != "/models/gemini-3.7-flash:generateContent" {
		t.Errorf("google path = %q", got)
	}
	if got := RouteResponses.path("x"); got != "/responses" {
		t.Errorf("responses path = %q", got)
	}
	if got := RouteMessages.path("x"); got != "/messages" {
		t.Errorf("messages path = %q", got)
	}
	if got := RouteChatCompletions.path("x"); got != "/chat/completions" {
		t.Errorf("chat path = %q", got)
	}
}

// TestOpencodeBaseURLs_MatchTheDescriptors: the provider's own copies of the
// default base URLs. The descriptors are built from these constants since
// 0015-PLAN S8, commit 1, so only the values are pinned here.
func TestOpencodeBaseURLs_MatchTheDescriptors(t *testing.T) {
	if DescriptorZen().DefaultBaseURL != zenBaseURL || DescriptorGo().DefaultBaseURL != goBaseURL {
		t.Errorf("descriptor base URLs %q, %q", DescriptorZen().DefaultBaseURL, DescriptorGo().DefaultBaseURL)
	}
	if zenBaseURL != "https://opencode.ai/zen/v1" || goBaseURL != "https://opencode.ai/zen/go/v1" {
		t.Errorf("base URLs = %q, %q", zenBaseURL, goBaseURL)
	}
}

// TestOpencodeRouteTable_NoUnknownRoutes ensures a typo in the table cannot ship.
func TestOpencodeRouteTable_NoUnknownRoutes(t *testing.T) {
	for gateway, byModel := range routeTable {
		if len(byModel) == 0 {
			t.Errorf("route table for %q is empty", gateway)
		}
		for model, route := range byModel {
			if !route.valid() {
				t.Errorf("route table[%q][%q] = %q is not a known route", gateway, model, route)
			}
		}
	}
}

// TestWireShapesProbedOn is the opencode row of llmprovider's test: the probe
// date is a real date, and not in the future.
func TestWireShapesProbedOn(t *testing.T) {
	d, err := time.Parse(time.DateOnly, wireShapesProbedOn)
	if err != nil {
		t.Fatalf("wire-shape pin %q is not a YYYY-MM-DD date: %v", wireShapesProbedOn, err)
	}
	if d.After(time.Now()) {
		t.Errorf("wire-shape pin %q is in the future", wireShapesProbedOn)
	}
}
