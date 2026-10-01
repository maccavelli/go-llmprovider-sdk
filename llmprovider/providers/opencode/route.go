package opencode

import (
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

// routeTable maps gateway -> model id -> wire format. It is the
// fallback for a request whose metadata is unavailable (MADR 0012 §3.1), and
// it is generated from the same document: every model in the "opencode" and
// "opencode-go" sections of models.opencode.ai/api.json (retrieved
// 2026-09-27) whose status is not "deprecated", routed by its provider.npm
// (routeForNPM). TestOpencodeRouteTable_MatchesMetadataSnapshot pins
// it to testdata/opencode-routes.json.
//
// Routing is per-gateway, not per-model: the minimax-* family takes
// chat_completions on Zen and messages on Go.
//
//nolint:goconst // model IDs are intentionally repeated across gateways and tests
var routeTable = map[llmprovider.ProviderID]map[string]Route{
	llmprovider.ProviderOpencodeZen: {
		// responses (@ai-sdk/openai)
		"gpt-5":                           RouteResponses,
		"gpt-5-codex":                     RouteResponses,
		"gpt-5-nano":                      RouteResponses,
		"gpt-5.1":                         RouteResponses,
		"gpt-5.1-codex":                   RouteResponses,
		"gpt-5.1-codex-max":               RouteResponses,
		"gpt-5.1-codex-mini":              RouteResponses,
		"gpt-5.2":                         RouteResponses,
		"gpt-5.2-codex":                   RouteResponses,
		"gpt-5.3-codex":                   RouteResponses,
		"gpt-5.3-codex-spark":             RouteResponses,
		"gpt-5.4":                         RouteResponses,
		"gpt-5.4-mini":                    RouteResponses,
		"gpt-5.4-nano":                    RouteResponses,
		"gpt-5.4-pro":                     RouteResponses,
		"gpt-5.5":                         RouteResponses,
		"gpt-5.5-pro":                     RouteResponses,
		"gpt-5.6-luna":                    RouteResponses,
		"gpt-5.6-sol":                     RouteResponses,
		"gpt-5.6-terra":                   RouteResponses,
		"gpt-6-astra":                     RouteResponses,
		"gpt-6-luna":                      RouteResponses,
		"gpt-6-sol":                       RouteResponses,
		"grok-4.5":                        RouteResponses,
		"grok-4.6":                        RouteResponses,
		"grok-4.7":                        RouteResponses,
		"grok-build-0.1":                  RouteResponses,
		"muse-spark-1.2":                  RouteResponses,
		"muse-spark-1.3":                  RouteResponses,
		"muse-spark-1.3-contributor-free": RouteResponses,

		// messages (@ai-sdk/anthropic)
		"claude-fable-5":    RouteMessages,
		"claude-fable-5-1":  RouteMessages,
		"claude-haiku-4-5":  RouteMessages,
		"claude-opus-4-5":   RouteMessages,
		"claude-opus-4-6":   RouteMessages,
		"claude-opus-4-7":   RouteMessages,
		"claude-opus-4-8":   RouteMessages,
		"claude-opus-5":     RouteMessages,
		"claude-opus-5-5":   RouteMessages,
		"claude-sonnet-4":   RouteMessages,
		"claude-sonnet-4-5": RouteMessages,
		"claude-sonnet-4-6": RouteMessages,
		"claude-sonnet-5":   RouteMessages,
		"qwen3.5-plus":      RouteMessages,
		"qwen3.6-plus":      RouteMessages,
		"qwen3.8-flash":     RouteMessages,

		// google (@ai-sdk/google)
		"gemini-3-flash":        RouteGoogle,
		"gemini-3.1-pro":        RouteGoogle,
		"gemini-3.5-flash":      RouteGoogle,
		"gemini-3.5-flash-lite": RouteGoogle,
		"gemini-3.6-flash":      RouteGoogle,
		"gemini-3.7-flash":      RouteGoogle,
		"gemini-3.8-flash":      RouteGoogle,

		// chat_completions (any other npm, or unset)
		"big-pickle":                   RouteChatCompletions,
		"deepseek-v4-flash":            RouteChatCompletions,
		"deepseek-v4-flash-vision-exp": RouteChatCompletions,
		"deepseek-v4-pro":              RouteChatCompletions,
		"deepseek-v4.1-flash":          RouteChatCompletions,
		"glm-5":                        RouteChatCompletions,
		"glm-5.1":                      RouteChatCompletions,
		"glm-5.2":                      RouteChatCompletions,
		"glm-5.3":                      RouteChatCompletions,
		"glm-5.3-flash":                RouteChatCompletions,
		"kimi-k2.5":                    RouteChatCompletions,
		"kimi-k2.6":                    RouteChatCompletions,
		"kimi-k2.7-code":               RouteChatCompletions,
		"kimi-k3":                      RouteChatCompletions,
		"ling-3.0-flash-fin-free":      RouteChatCompletions,
		"longcat-2.5-preview-free":     RouteChatCompletions,
		"mimo-v2.6-flash-free":         RouteChatCompletions,
		"minimax-m2.5":                 RouteChatCompletions,
		"minimax-m2.7":                 RouteChatCompletions,
		"minimax-m3":                   RouteChatCompletions,
		"nemotron-3-ultra-free":        RouteChatCompletions,
		"nemotron-3.5-lightning-free":  RouteChatCompletions,
		"qwen3.8-max":                  RouteChatCompletions,
		"space-bunny-free":             RouteChatCompletions,
	},
	llmprovider.ProviderOpencodeGo: {
		// responses (@ai-sdk/openai)
		"gpt-5.6-luna":               RouteResponses,
		"gpt-6-luna":                 RouteResponses,
		"grok-4.6":                   RouteResponses,
		"grok-4.7":                   RouteResponses,
		"muse-spark-1.2-contributor": RouteResponses,
		"muse-spark-1.3-contributor": RouteResponses,

		// messages (@ai-sdk/anthropic)
		"minimax-m2.7":  RouteMessages,
		"minimax-m3":    RouteMessages,
		"qwen3.8-flash": RouteMessages,

		// chat_completions (any other npm, or unset)
		"deepseek-v4-flash":            RouteChatCompletions,
		"deepseek-v4-flash-vision-exp": RouteChatCompletions,
		"deepseek-v4-pro":              RouteChatCompletions,
		"deepseek-v4.1-flash":          RouteChatCompletions,
		"glm-5.1":                      RouteChatCompletions,
		"glm-5.2":                      RouteChatCompletions,
		"glm-5.3":                      RouteChatCompletions,
		"glm-5.3-flash":                RouteChatCompletions,
		"hy3":                          RouteChatCompletions,
		"hy4-preview":                  RouteChatCompletions,
		"kimi-k2.6":                    RouteChatCompletions,
		"kimi-k2.7-code":               RouteChatCompletions,
		"kimi-k3":                      RouteChatCompletions,
		"longcat-2.0":                  RouteChatCompletions,
		"longcat-2.5-preview-free":     RouteChatCompletions,
		"mimo-v2.5":                    RouteChatCompletions,
		"mimo-v2.5-pro":                RouteChatCompletions,
		"mimo-v2.6-flash":              RouteChatCompletions,
		"mimo-v2.6-pro":                RouteChatCompletions,
		"qwen3.6-plus":                 RouteChatCompletions,
		"qwen3.7-max":                  RouteChatCompletions,
		"qwen3.7-plus":                 RouteChatCompletions,
		"qwen3.8-max":                  RouteChatCompletions,
		"space-bunny-free":             RouteChatCompletions,
	},
}

// heuristicRoute infers a route from the model id prefix when the table
// has no entry. Both gateways send gpt/grok/muse to responses and qwen to
// messages; they differ on minimax (Go only) and gemini/claude (Zen only).
// Chat Completions is the default because it is the largest bucket on both.
func heuristicRoute(gateway llmprovider.ProviderID, model string) Route {
	m := strings.ToLower(strings.TrimSpace(model))

	switch {
	case strings.HasPrefix(m, "gpt-"), strings.HasPrefix(m, "grok-"),
		strings.HasPrefix(m, "muse-"):
		return RouteResponses
	case strings.HasPrefix(m, "qwen"):
		return RouteMessages
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
