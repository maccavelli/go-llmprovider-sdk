package opencode

import (
	_ "embed" // routes_snapshot.json
	"encoding/json"
	"fmt"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
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
// wrong route fails with an opaque HTTP 500 rather than a typed error.
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

// routesSnapshot is provider.npm for every model in the "opencode" and
// "opencode-go" sections of models.opencode.ai/api.json (retrieved
// 2026-09-27) whose status is not "deprecated".
//
//go:embed routes_snapshot.json
var routesSnapshot []byte

// routeSections maps the snapshot's sections to their gateways.
var routeSections = map[string]llmprovider.ProviderID{
	"opencode":    llmprovider.ProviderOpencodeZen,
	"opencode-go": llmprovider.ProviderOpencodeGo,
}

// routeTable maps gateway -> model id -> wire format. It is the fallback for
// a request whose metadata is unavailable (MADR 0012 §3.1). It is built from
// routesSnapshot, each model routed by its provider.npm (routeForNPM), so the
// table and the snapshot cannot drift apart (0021-MADR C10).
//
// Routing is per-gateway, not per-model: the minimax-* family takes
// chat_completions on Zen and messages on Go.
var routeTable map[llmprovider.ProviderID]map[string]Route

func init() { routeTable = buildRouteTable(routesSnapshot) }

// buildRouteTable routes each model of a snapshot by its provider.npm. The
// snapshot is embedded, so a malformed one is a build defect, and panics;
// TestOpencodeRouteTable_MatchesMetadataSnapshot pins it.
func buildRouteTable(raw []byte) map[llmprovider.ProviderID]map[string]Route {
	var snapshot map[string]map[string]string
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		panic(fmt.Sprintf("opencode: routes_snapshot.json: %v", err))
	}
	table := make(map[llmprovider.ProviderID]map[string]Route, len(routeSections))
	for section, models := range snapshot {
		gateway, ok := routeSections[section]
		if !ok {
			panic(fmt.Sprintf("opencode: routes_snapshot.json: unknown section %q", section))
		}
		routes := make(map[string]Route, len(models))
		for model, npm := range models {
			routes[strings.ToLower(model)] = routeForNPM(npm)
		}
		table[gateway] = routes
	}
	return table
}

// heuristicRoute infers a route from the model id prefix when the table
// has no entry. Both gateways send gpt/grok/muse to responses. A qwen model
// routes as the table's qwen rows do (qwenRoute). They differ on minimax (Go
// only) and gemini/claude (Zen only). Chat Completions is the default
// because it is the largest bucket on both.
func heuristicRoute(gateway llmprovider.ProviderID, model string) Route {
	m := strings.ToLower(strings.TrimSpace(model))

	switch {
	case strings.HasPrefix(m, "gpt-"), strings.HasPrefix(m, "grok-"),
		strings.HasPrefix(m, "muse-"):
		return RouteResponses
	case strings.HasPrefix(m, "qwen"):
		return qwenRoute(gateway, m)
	}

	if gateway == llmprovider.ProviderOpencodeZen {
		switch {
		case strings.HasPrefix(m, "claude-"):
			return RouteMessages
		case strings.HasPrefix(m, "gemini-"):
			return RouteGoogle
		}
	}
	if gateway == llmprovider.ProviderOpencodeGo && strings.HasPrefix(m, "minimax-") {
		return RouteMessages
	}

	return RouteChatCompletions
}

// qwenRoute is the route of a qwen model the table does not list, as the
// table's qwen rows are: -max to chat completions and -flash to messages on
// both gateways, and any other to messages on Zen and to chat completions on
// Go (0020-MADR F41).
func qwenRoute(gateway llmprovider.ProviderID, model string) Route {
	switch {
	case strings.HasSuffix(model, "-max"):
		return RouteChatCompletions
	case strings.HasSuffix(model, "-flash"):
		return RouteMessages
	case gateway == llmprovider.ProviderOpencodeGo:
		return RouteChatCompletions
	default:
		return RouteMessages
	}
}

// routeForNPM maps a model's provider.npm to its route, as OpenCode's
// client picks an AI SDK package (provider.ts:1274-1278); any other package,
// or none, is the openai-compatible chat route.
func routeForNPM(npm string) Route {
	switch npm {
	case "@ai-sdk/openai":
		return RouteResponses
	case "@ai-sdk/anthropic":
		return RouteMessages
	case "@ai-sdk/google":
		return RouteGoogle
	default:
		return RouteChatCompletions
	}
}

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
