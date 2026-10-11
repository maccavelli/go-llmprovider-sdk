package messages

import (
	"fmt"
	"slices"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

const keySystem = "system"

// forcedChoiceRefused are the models that answer a forced tool_choice (any or
// tool) with HTTP 400, as Anthropic's define-tools and errors pages name them,
// read 2026-10-11 (0034-MADR D1). Exact ids, compared case-insensitively: a
// model Anthropic adds later is added by a record. Its models API carries no
// such capability.
var forcedChoiceRefused = []string{"claude-fable-5-1", "claude-mythos-5-1", "claude-opus-5-5", "claude-sonnet-5-5"}

// refusesForcedChoice reports whether model refuses a forced tool_choice.
func refusesForcedChoice(model string) bool {
	return slices.Contains(forcedChoiceRefused, strings.ToLower(model))
}

// ToolChoice is the Messages tool_choice for choice, or nil when none is sent
// (auto). A forced choice (ForceTool or ToolChoiceRequired) is sent as auto
// with thinking, which Anthropic refuses to force (0020-MADR F43), and on a
// model that refuses forcing (0034-MADR D2).
func ToolChoice(choice llmprovider.ToolChoice, thinking bool, model string) map[string]any {
	name, named := choice.Tool()
	forced := named || choice == llmprovider.ToolChoiceRequired
	switch {
	case forced && (thinking || refusesForcedChoice(model)):
		return map[string]any{wire.KeyType: "auto"}
	case named:
		return map[string]any{wire.KeyType: "tool", wire.KeyName: name}
	case choice == llmprovider.ToolChoiceRequired:
		return map[string]any{wire.KeyType: "any"}
	case choice == llmprovider.ToolChoiceNone:
		return map[string]any{wire.KeyType: "none"}
	default:
		return nil
	}
}

// AddToolChoice sets body's tool_choice for choice. On a model that refuses a
// forced choice, a forced choice is sent as auto with Anthropic's documented
// turn instruction naming the tool, or one of the tools (0034-MADR D2). The
// instruction is a mid-conversation system message after the last user turn;
// when the messages do not end with one, which that message must follow, it
// is added to the top-level system instead. body's messages are FromItems'.
func AddToolChoice(body map[string]any, choice llmprovider.ToolChoice, thinking bool, model string) {
	if c := ToolChoice(choice, thinking, model); c != nil {
		body["tool_choice"] = c
	}
	name, named := choice.Tool()
	if !refusesForcedChoice(model) || (!named && choice != llmprovider.ToolChoiceRequired) {
		return
	}
	if !named {
		name = "" // Tool's string is the choice itself, not a tool's name
	}
	text := turnInstruction(name)
	if msgs, ok := body["messages"].([]Message); ok && len(msgs) > 0 && msgs[len(msgs)-1].Role == wire.RoleUser {
		body["messages"] = append(msgs, Message{Role: wire.RoleSystem, Content: Content{text: &text}})
		return
	}
	if system, ok := body[keySystem].(string); ok && system != "" {
		text = system + "\n\n" + text
	}
	body[keySystem] = text
}

// turnInstruction is Anthropic's turn instruction for a required call to the
// named tool, or to one of the tools when name is empty (the Claude Fable 5.1
// migration guide, read 2026-10-11).
func turnInstruction(name string) string {
	if name == "" {
		return "Tool-use requirement for the current turn: the application requires a call to one of the tools in your " +
			"response to the user's latest message. Begin your response with that tool call. Do not reply with text only."
	}
	return fmt.Sprintf("Tool-use requirement for the current turn: the application requires a call to the %s tool in your "+
		"response to the user's latest message. Begin your response with the %s tool call. Do not reply with text only.", name, name)
}
