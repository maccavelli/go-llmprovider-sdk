// Package opencoderoute is OpenCode's route snapshot and table: which wire
// format the Zen and Go gateways expect for a model, without the models.dev
// document. providers/opencode routes its requests by it, and catalog reads
// it for Limits.Required, so the two cannot disagree (0089-MADR DL74; its
// PLAN's deviations 2 and 4). A route is its name: "responses",
// "messages", "chat_completions" or "google", as opencode.Route spells it.
package opencoderoute

import (
	_ "embed" // routes_snapshot.json
	"encoding/json"
	"fmt"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// The four routes, by name.
const (
	Responses       = "responses"
	Messages        = "messages"
	ChatCompletions = "chat_completions"
	Google          = "google"
)

// Snapshot is provider.npm for every model in the "opencode" and
// "opencode-go" sections of models.opencode.ai/api.json (retrieved
// 2026-09-27) whose status is not "deprecated".
//
//go:embed routes_snapshot.json
var Snapshot []byte

// sections maps the snapshot's sections to their gateways.
var sections = map[string]llmprovider.ProviderID{
	"opencode":    llmprovider.ProviderOpencodeZen,
	"opencode-go": llmprovider.ProviderOpencodeGo,
}

// Table maps gateway -> model id, lower case -> route. It is the fallback
// for a request whose metadata is unavailable (MADR 0012 §3.1), built from
// Snapshot, each model routed by its provider.npm (ForNPM), so the table and
// the snapshot cannot drift apart (0021-MADR C10).
//
// Routing is per-gateway, not per-model: the minimax-* family takes
// chat_completions on Zen and messages on Go.
var Table = BuildTable(Snapshot)

// BuildTable routes each model of a snapshot by its provider.npm. The
// snapshot is embedded, so a malformed one is a build defect, and panics.
func BuildTable(raw []byte) map[llmprovider.ProviderID]map[string]string {
	var snapshot map[string]map[string]string
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		panic(fmt.Sprintf("opencode: routes_snapshot.json: %v", err))
	}
	table := make(map[llmprovider.ProviderID]map[string]string, len(sections))
	for section, models := range snapshot {
		gateway, ok := sections[section]
		if !ok {
			panic(fmt.Sprintf("opencode: routes_snapshot.json: unknown section %q", section))
		}
		routes := make(map[string]string, len(models))
		for model, npm := range models {
			routes[strings.ToLower(model)] = ForNPM(npm)
		}
		table[gateway] = routes
	}
	return table
}

// Heuristic infers a route from the model id prefix when the table has no
// entry. Both gateways send gpt/grok/muse to responses. A qwen model routes
// as the table's qwen rows do (qwen). They differ on minimax (Go only) and
// gemini/claude (Zen only). Chat Completions is the default because it is
// the largest bucket on both.
func Heuristic(gateway llmprovider.ProviderID, model string) string {
	m := strings.ToLower(strings.TrimSpace(model))

	switch {
	case strings.HasPrefix(m, "gpt-"), strings.HasPrefix(m, "grok-"),
		strings.HasPrefix(m, "muse-"):
		return Responses
	case strings.HasPrefix(m, "qwen"):
		return qwen(gateway, m)
	}

	if gateway == llmprovider.ProviderOpencodeZen {
		switch {
		case strings.HasPrefix(m, "claude-"):
			return Messages
		case strings.HasPrefix(m, "gemini-"):
			return Google
		}
	}
	if gateway == llmprovider.ProviderOpencodeGo && strings.HasPrefix(m, "minimax-") {
		return Messages
	}

	return ChatCompletions
}

// qwen is the route of a qwen model the table does not list, as the table's
// qwen rows are: -max to chat completions and -flash to messages on both
// gateways, and any other to messages on Zen and to chat completions on Go
// (0020-MADR F41).
func qwen(gateway llmprovider.ProviderID, model string) string {
	switch {
	case strings.HasSuffix(model, "-max"):
		return ChatCompletions
	case strings.HasSuffix(model, "-flash"):
		return Messages
	case gateway == llmprovider.ProviderOpencodeGo:
		return ChatCompletions
	default:
		return Messages
	}
}

// ForNPM maps a model's provider.npm to its route, as OpenCode's client
// picks an AI SDK package (provider.ts:1274-1278); any other package, or
// none, is the openai-compatible chat route.
func ForNPM(npm string) string {
	switch npm {
	case "@ai-sdk/openai":
		return Responses
	case "@ai-sdk/anthropic":
		return Messages
	case "@ai-sdk/google":
		return Google
	default:
		return ChatCompletions
	}
}

// Of is the table's route for (gateway, model), else the heuristic's.
func Of(gateway llmprovider.ProviderID, model string) string {
	if byModel, ok := Table[gateway]; ok {
		if r, ok := byModel[strings.ToLower(strings.TrimSpace(model))]; ok {
			return r
		}
	}
	return Heuristic(gateway, model)
}
