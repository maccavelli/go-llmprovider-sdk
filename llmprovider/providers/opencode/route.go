package opencode

import (
	"fmt"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/opencoderoute"
)

// wireShapesProbedOn is the date every wire shape in this file was measured
// against the live gateway: the per-gateway route tables, which routes reject
// which models (a mismatch returns HTTP 500, not a typed error), and the
// response envelopes each route returns.
//
// The OpenCode gateways are remote and continuously deployed. They expose no
// version endpoint, so there is nothing to compare this against at runtime and
// nothing warns when it goes stale — unlike a local engine, which can report a
// version on boot (see magic-cli-remote/internal/provider/opencode/version.go).
// Re-validate with: go test -tags live_gateways ./llmprovider/ -run Live
const wireShapesProbedOn = "2026-09-27"

// Route identifies which wire format the OpenCode gateway expects for a given
// model; WithRoute pins one. OpenCode Zen and Go are multi-protocol gateways:
// they do not normalize to a single request shape, and sending a model to the
// wrong route fails with an opaque HTTP 500 rather than a typed error. It is
// a string, safe for concurrent use.
type Route string

// The four wire formats the OpenCode gateways dispatch to.
const (
	RouteResponses       Route = "responses"
	RouteMessages        Route = "messages"
	RouteChatCompletions Route = "chat_completions"
	RouteGoogle          Route = "google"
)

// path returns the request path suffix for the route. The Google route is
// model-scoped, so it takes the model id.
func (r Route) path(model string) string {
	switch r {
	case RouteResponses:
		return "/responses"
	case RouteMessages:
		return "/messages"
	case RouteGoogle:
		return fmt.Sprintf("/models/%s:generateContent", model)
	case RouteChatCompletions:
		return "/chat/completions"
	default:
		return "/chat/completions"
	}
}

// valid reports whether r is one of the four known routes.
func (r Route) valid() bool {
	switch r {
	case RouteResponses, RouteMessages,
		RouteChatCompletions, RouteGoogle:
		return true
	default:
		return false
	}
}

// routesSnapshot is OpenCode's route snapshot, embedded in
// internal/opencoderoute, which catalog reads too (0089-PLAN deviations 2
// and 4).
var routesSnapshot = opencoderoute.Snapshot

// routeTable maps gateway -> model id -> wire format: internal/opencoderoute's
// table, as Routes. It is the fallback for a request whose metadata is
// unavailable (MADR 0012 §3.1), built from routesSnapshot, each model routed
// by its provider.npm (routeForNPM), so the table and the snapshot cannot
// drift apart (0021-MADR C10).
//
// Routing is per-gateway, not per-model: the minimax-* family takes
// chat_completions on Zen and messages on Go.
var routeTable map[llmprovider.ProviderID]map[string]Route

func init() { routeTable = buildRouteTable(routesSnapshot) }

// buildRouteTable routes each model of a snapshot by its provider.npm, by
// internal/opencoderoute's rule. The snapshot is embedded, so a malformed
// one is a build defect, and panics;
// TestOpencodeRouteTable_MatchesMetadataSnapshot pins it.
func buildRouteTable(raw []byte) map[llmprovider.ProviderID]map[string]Route {
	built := opencoderoute.BuildTable(raw)
	table := make(map[llmprovider.ProviderID]map[string]Route, len(built))
	for gateway, models := range built {
		routes := make(map[string]Route, len(models))
		for model, r := range models {
			routes[model] = Route(r)
		}
		table[gateway] = routes
	}
	return table
}

// heuristicRoute infers a route from the model id prefix when the table
// has no entry, by internal/opencoderoute's heuristic: gpt/grok/muse to
// responses on both gateways, a qwen model as the table's qwen rows are,
// minimax to messages on Go only, gemini and claude to google and messages
// on Zen only, and Chat Completions otherwise.
func heuristicRoute(gateway llmprovider.ProviderID, model string) Route {
	return Route(opencoderoute.Heuristic(gateway, model))
}

// routeForNPM maps a model's provider.npm to its route, as OpenCode's
// client picks an AI SDK package (provider.ts:1274-1278); any other package,
// or none, is the openai-compatible chat route.
func routeForNPM(npm string) Route { return Route(opencoderoute.ForNPM(npm)) }

// resolveRoute picks the wire format for (gateway, model), honouring an
// explicit override first, then the published table, then the prefix heuristic.
func resolveRoute(gateway llmprovider.ProviderID, model string, override Route) (Route, error) {
	if override != "" {
		if !override.valid() {
			return "", fmt.Errorf("%w: unknown opencode route %q", llmprovider.ErrInvalidRequest, override)
		}
		return override, nil
	}
	return tableRoute(gateway, model), nil
}

// inRouteTable reports whether gateway's route table lists model.
func inRouteTable(gateway llmprovider.ProviderID, model string) bool {
	_, ok := routeTable[gateway][strings.ToLower(strings.TrimSpace(model))]
	return ok
}

// tableRoute is the published table's route for (gateway, model), else the
// prefix heuristic's.
func tableRoute(gateway llmprovider.ProviderID, model string) Route {
	if byModel, ok := routeTable[gateway]; ok {
		if r, ok := byModel[strings.ToLower(strings.TrimSpace(model))]; ok {
			return r
		}
	}
	return heuristicRoute(gateway, model)
}
