package messages

import (
	"fmt"
	"iter"
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

// refusesForcedChoice reports whether model refuses a forced tool_choice on
// provider: it is in the table, or this process learned it (0034-MADR D3).
func refusesForcedChoice(provider, model string) bool {
	return slices.Contains(forcedChoiceRefused, strings.ToLower(model)) || learnedRefusal(provider, model)
}

// ToolChoice is the Messages tool_choice for choice, or nil when none is sent
// (auto). A forced choice (ForceTool or ToolChoiceRequired) is sent as auto
// with thinking, which Anthropic refuses to force (0020-MADR F43), and on a
// model that refuses forcing on provider (0034-MADR D2, D3).
func ToolChoice(choice llmprovider.ToolChoice, thinking bool, provider, model string) map[string]any {
	name, named := choice.Tool()
	forced := named || choice == llmprovider.ToolChoiceRequired
	switch {
	case forced && (thinking || refusesForcedChoice(provider, model)):
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
func AddToolChoice(body map[string]any, choice llmprovider.ToolChoice, thinking bool, provider, model string) {
	if c := ToolChoice(choice, thinking, provider, model); c != nil {
		body["tool_choice"] = c
	}
	name, named := choice.Tool()
	if !refusesForcedChoice(provider, model) || (!named && choice != llmprovider.ToolChoiceRequired) {
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

// ToolChoiceWarning is the warning a request's forced tool choice earns when
// it is sent as auto, on model through provider, and false when it is sent as
// asked (0034-MADR D7). It is decided as AddToolChoice decides: with
// thinking (0020-MADR F43), or on a model that refuses forcing. Called after
// the request is sent, it sees a refusal learned by its resend.
func ToolChoiceWarning(req *llmprovider.Request, thinking bool, provider, model string) (llmprovider.Warning, bool) {
	name, named := req.ToolChoice.Tool()
	if len(req.Tools) == 0 || (!named && req.ToolChoice != llmprovider.ToolChoiceRequired) {
		return llmprovider.Warning{}, false
	}
	target := "one of the tools"
	if named {
		target = "the " + name + " tool"
	}
	switch {
	case refusesForcedChoice(provider, model):
		return llmprovider.Warning{Kind: llmprovider.WarningForcedChoiceDegraded, Model: model,
			Detail: model + " refuses a forced tool choice, so it was sent as auto, with an instruction to call " + target + "."}, true
	case thinking:
		return llmprovider.Warning{Kind: llmprovider.WarningForcedChoiceDegraded, Model: model,
			Detail: "A forced tool choice was sent as auto, since Anthropic refuses to force a tool while thinking; the model may not call " + target + "."}, true
	}
	return llmprovider.Warning{}, false
}

// WarnOnDone is seq with the warning warn returns, if any, added to its done
// event's Response. warn runs when the done event arrives, after any resend.
func WarnOnDone(seq iter.Seq2[llmprovider.Event, error], warn func() (llmprovider.Warning, bool)) iter.Seq2[llmprovider.Event, error] {
	return func(yield func(llmprovider.Event, error) bool) {
		for ev, err := range seq {
			if err == nil && ev.Type == llmprovider.EventDone && ev.Response != nil {
				if w, ok := warn(); ok {
					ev.Response.Warnings = append(ev.Response.Warnings, w)
				}
			}
			if !yield(ev, err) {
				return
			}
		}
	}
}
