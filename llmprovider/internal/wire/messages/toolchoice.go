package messages

import (
	"fmt"
	"iter"
	"slices"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

const (
	keySystem       = "system"
	keyOutputConfig = "output_config"

	// The tool_choice types this wire sends.
	choiceAuto = "auto"
	choiceNone = "none"
)

// forcedChoiceRefused are the models that answer a forced tool_choice (any or
// tool) with HTTP 400, as Anthropic's define-tools and errors pages name them,
// read 2026-10-11 (0034-MADR D1). Exact ids, compared case-insensitively: a
// model Anthropic adds later is added by a record. Its models API carries no
// such capability.
var forcedChoiceRefused = []string{"claude-fable-5-1", "claude-mythos-5-1", "claude-opus-5-5", "claude-sonnet-5-5"}

// refusesForcedChoice reports whether model refuses a forced tool_choice on
// provider: its normalised id is in the table, or learned holds it, a nil one
// being the process's (0034-MADR D3, D9; 0049-MADR D3). The caller's override
// is Forcing's.
func refusesForcedChoice(learned *Refusals, provider, model string) bool {
	return slices.Contains(forcedChoiceRefused, normalizeModel(model)) || learned.learned(provider, model)
}

// ToolChoice is the Messages tool_choice for choice, or nil when none is sent
// (auto). On a model that refuses forcing, a named tool is none: AddToolChoice
// sends it as structured outputs (0034-MADR D8). Otherwise a forced choice
// (ForceTool or ToolChoiceRequired) is sent as auto with thinking, which
// Anthropic refuses to force (0020-MADR F43), and on a model that refuses
// forcing (0034-MADR D2, D3, D9).
func ToolChoice(choice llmprovider.ToolChoice, thinking bool, f Forcing) map[string]any {
	name, named := choice.Tool()
	forced := named || choice == llmprovider.ToolChoiceRequired
	switch {
	case named && f.refuses():
		return map[string]any{wire.KeyType: choiceNone}
	case forced && (thinking || f.refuses()):
		return map[string]any{wire.KeyType: choiceAuto}
	case named:
		return map[string]any{wire.KeyType: "tool", wire.KeyName: name}
	case choice == llmprovider.ToolChoiceRequired:
		return map[string]any{wire.KeyType: "any"}
	case choice == llmprovider.ToolChoiceNone:
		return map[string]any{wire.KeyType: choiceNone}
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
func AddToolChoice(body map[string]any, choice llmprovider.ToolChoice, thinking bool, f Forcing) {
	name, named := choice.Tool()
	if named && f.refuses() {
		if schema, ok := offeredSchema(body, name); ok {
			// The tool's arguments, as structured outputs: schema-valid JSON,
			// with the tools kept, uncalled, so the history stays valid and
			// the tool definitions stay the same from turn to turn
			// (0034-MADR D8; 0034-PLAN Phase 4b).
			body["tool_choice"] = map[string]any{wire.KeyType: choiceNone}
			cfg, ok := body[keyOutputConfig].(map[string]any)
			if !ok {
				cfg = map[string]any{}
			}
			cfg["format"] = map[string]any{wire.KeyType: "json_schema", "schema": closedSchema(schema)}
			body[keyOutputConfig] = cfg
			return
		}
		// A named tool the body does not offer has no schema to send: it is
		// degraded, as before D8.
		body["tool_choice"] = map[string]any{wire.KeyType: choiceAuto}
		appendTurnInstruction(body, turnInstruction(name))
		return
	}
	if c := ToolChoice(choice, thinking, f); c != nil {
		body["tool_choice"] = c
	}
	if !f.refuses() || choice != llmprovider.ToolChoiceRequired {
		return
	}
	appendTurnInstruction(body, turnInstruction(""))
}

// offeredSchema is the input_schema of the tool body offers under name, and
// whether it offers one.
func offeredSchema(body map[string]any, name string) (any, bool) {
	tools, ok := body["tools"].([]map[string]any)
	if !ok {
		return nil, false
	}
	for _, tool := range tools {
		if tool[wire.KeyName] == name {
			schema, ok := tool["input_schema"]
			return schema, ok
		}
	}
	return nil, false
}

// appendTurnInstruction adds text after the last user turn, as a
// mid-conversation system message, or, when the messages do not end with one,
// to the top-level system.
func appendTurnInstruction(body map[string]any, text string) {
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
// it is sent as auto, for f's model, and false when it is sent as
// asked (0034-MADR D7). It is decided as AddToolChoice decides: with
// thinking (0020-MADR F43), or on a model that refuses forcing. Called after
// the request is sent, it sees a refusal learned by its resend.
func ToolChoiceWarning(req *llmprovider.Request, thinking bool, f Forcing) (llmprovider.Warning, bool) {
	model := f.Model
	name, named := req.ToolChoice.Tool()
	if len(req.Tools) == 0 || (!named && req.ToolChoice != llmprovider.ToolChoiceRequired) {
		return llmprovider.Warning{}, false
	}
	target := "one of the tools"
	if named {
		target = "the " + name + " tool"
	}
	switch {
	case named && f.refuses() && offered(req.Tools, name):
		// Sent as structured outputs: not degraded (0034-MADR D8).
		return llmprovider.Warning{}, false
	case f.refuses():
		return llmprovider.Warning{Kind: llmprovider.WarningForcedChoiceDegraded, Model: model,
			Detail: model + " refuses a forced tool choice, so it was sent as auto, with an instruction to call " + target + "."}, true
	case thinking:
		return llmprovider.Warning{Kind: llmprovider.WarningForcedChoiceDegraded, Model: model,
			Detail: "A forced tool choice was sent as auto, since Anthropic refuses to force a tool while thinking; the model may not call " + target + "."}, true
	}
	return llmprovider.Warning{}, false
}

// WarnOnDone is seq with the warnings warns return, if any, added in order to
// its done event's Response. Each runs when the done event arrives, after any
// resend.
func WarnOnDone(seq iter.Seq2[llmprovider.Event, error], warns ...func() (llmprovider.Warning, bool)) iter.Seq2[llmprovider.Event, error] {
	return func(yield func(llmprovider.Event, error) bool) {
		for ev, err := range seq {
			if err == nil && ev.Type == llmprovider.EventDone && ev.Response != nil {
				for _, warn := range warns {
					if w, ok := warn(); ok {
						ev.Response.Warnings = append(ev.Response.Warnings, w)
					}
				}
			}
			if !yield(ev, err) {
				return
			}
		}
	}
}
