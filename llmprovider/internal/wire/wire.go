// Package wire holds what the shared wire formats have in common (0015-MADR
// D2, amendment "S7b's import graph"): the JSON keys more than one format
// uses, and the item conversions more than one needs. Each format is a
// package below it: responses, chatcompletions, messages and generatecontent.
// The Interactions wire is Gemini's alone, in providers/gemini.
package wire

import (
	"encoding/json"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// JSON keys and roles more than one format uses.
const (
	KeyArguments = "arguments"
	KeyContent   = "content"
	KeyName      = "name"
	KeyOutput    = "output"
	KeyRole      = "role"
	KeyText      = "text"
	KeyType      = "type"

	RoleAssistant = "assistant"
	RoleSystem    = "system"
	RoleUser      = "user"
)

// The wire formats a ReasoningItem's Format names (0021-MADR D5).
const (
	FormatResponses       = "responses"
	FormatMessages        = "messages"
	FormatChatCompletions = "chatcompletions"
	FormatGenerateContent = "generatecontent"
)

// Replays reports whether a reasoning item's opaque parts, its Signature and
// Encrypted, may go to the wire named format: the wire that issued them, or
// any wire for an item with no Format (0021-MADR D5, W7).
func Replays(item llmprovider.ReasoningItem, format string) bool {
	return item.Format == "" || item.Format == format
}

// LowEffortThinkingBudget is the budget "low" maps to where a model takes only
// a budget, in the Messages and generateContent thinking shapes. It is
// Anthropic's minimum: 1023 is refused with HTTP 400 (measured 2026-09-27).
const LowEffortThinkingBudget = 1024

// ToolArguments is a call's JSON arguments as the object Anthropic's
// tool_use.input and Gemini's functionCall.args require. A JSON object is
// passed through as it is, compacted, so a large integer keeps its digits
// (0021-MADR W1). Empty arguments are an empty object; arguments that are not
// a JSON object are kept under "arguments" rather than dropped.
func ToolArguments(arguments string) any {
	trimmed := strings.TrimSpace(arguments)
	if trimmed == "" {
		return map[string]any{}
	}
	if strings.HasPrefix(trimmed, "{") && json.Valid([]byte(trimmed)) {
		if compact, err := CompactArguments(json.RawMessage(trimmed)); err == nil {
			return json.RawMessage(compact)
		}
	}
	return map[string]any{KeyArguments: arguments}
}

// SystemPrompt joins the system items, in order, for a wire's dedicated
// system field: the Messages API's system (MADR 0012 §2) and Gemini's
// system instruction (MADR 0014). The converters leave system items out.
func SystemPrompt(items []llmprovider.Item) string {
	var parts []string
	for _, item := range items {
		if m, ok := item.(llmprovider.MessageItem); ok && m.Role == RoleSystem && m.Text != "" {
			parts = append(parts, m.Text)
		}
	}
	return strings.Join(parts, "\n\n")
}
