package llmprovider

import (
	"encoding/json"
	"strings"
)

// ToolArguments decodes a call's JSON arguments into the object Anthropic's
// tool_use.input and Gemini's functionCall.args require. Empty arguments are
// an empty object; arguments that are not a JSON object are kept under
// "arguments" rather than dropped.
// Temporary export for the provider packages (0015-PLAN S7); S7b moves it to internal/wire.
func ToolArguments(arguments string) map[string]any {
	if strings.TrimSpace(arguments) == "" {
		return map[string]any{}
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(arguments), &args); err == nil && args != nil {
		return args
	}
	return map[string]any{jsonKeyArguments: arguments}
}

// SystemPrompt joins the system items, in order, for a wire's dedicated
// system field: the Messages API's system (MADR 0012 §2) and Gemini's
// system instruction (MADR 0014). The converters leave system items out.
// Temporary export for the provider packages (0015-PLAN S7); S7b moves it to internal/wire.
func SystemPrompt(items []Item) string {
	var parts []string
	for _, item := range items {
		if m, ok := item.(MessageItem); ok && m.Role == jsonRoleSystem && m.Text != "" {
			parts = append(parts, m.Text)
		}
	}
	return strings.Join(parts, "\n\n")
}
