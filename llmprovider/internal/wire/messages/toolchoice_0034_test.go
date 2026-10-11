package messages

import (
	"encoding/json"
	"maps"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

const choiceTool = "get_weather"

// TestRefusesForcedChoice_0034 (0034-MADR D1): the four ids Anthropic names,
// in any case, and nothing else: no prefix, no version pattern.
func TestRefusesForcedChoice_0034(t *testing.T) {
	for _, id := range []string{"claude-fable-5-1", "claude-mythos-5-1", "claude-opus-5-5", "claude-sonnet-5-5", "Claude-Sonnet-5-5"} {
		if !refusesForcedChoice("claude", id) {
			t.Errorf("refusesForcedChoice(%q) = false, want true", id)
		}
	}
	for _, id := range []string{"claude-haiku-5-5", "claude-sonnet-5", "claude-opus-5-55", "claude-sonnet-5-5-x", "claude-fable-5", ""} {
		if refusesForcedChoice("claude", id) {
			t.Errorf("refusesForcedChoice(%q) = true, want false: exact ids only", id)
		}
	}
}

// choiceCase is a tool choice, with or without thinking, and the tool_choice
// it maps to; nil is none sent.
type choiceCase struct {
	choice   llmprovider.ToolChoice
	thinking bool
	want     map[string]any
}

// TestToolChoice_UnchangedOutsideTheTable_0034 (0034-MADR D2; R45): a model
// outside the table maps every choice as the providers' own functions did
// (0020-MADR F43).
func TestToolChoice_UnchangedOutsideTheTable_0034(t *testing.T) {
	auto, anyTool, none := map[string]any{"type": "auto"}, map[string]any{"type": "any"}, map[string]any{"type": "none"}
	named := map[string]any{"type": "tool", "name": choiceTool}
	for _, c := range []choiceCase{
		{"", false, nil}, {llmprovider.ToolChoiceAuto, false, nil},
		{llmprovider.ToolChoiceRequired, false, anyTool}, {llmprovider.ToolChoiceNone, false, none},
		{llmprovider.ForceTool(choiceTool), false, named},
		{llmprovider.ToolChoiceRequired, true, auto}, {llmprovider.ForceTool(choiceTool), true, auto},
		{llmprovider.ToolChoiceNone, true, none}, {llmprovider.ToolChoiceAuto, true, nil},
	} {
		got := ToolChoice(c.choice, c.thinking, Forcing{Provider: "claude", Model: "claude-sonnet-5"})
		if !maps.Equal(got, c.want) {
			t.Errorf("ToolChoice(%q, thinking %v) = %v, want %v", c.choice, c.thinking, got, c.want)
		}
	}
}

// TestToolChoice_RefusingModel_0034 (0034-MADR D2, D8): on a refusing model
// ToolChoiceRequired is auto, a named tool is none (its structured outputs are
// AddToolChoice's), and none and auto are unchanged.
func TestToolChoice_RefusingModel_0034(t *testing.T) {
	auto, none := map[string]any{"type": "auto"}, map[string]any{"type": "none"}
	for _, c := range []choiceCase{
		{llmprovider.ForceTool(choiceTool), false, none}, {llmprovider.ToolChoiceRequired, false, auto},
		{llmprovider.ForceTool(choiceTool), true, none}, {llmprovider.ToolChoiceNone, false, none},
		{llmprovider.ToolChoiceAuto, false, nil}, {"", false, nil},
	} {
		got := ToolChoice(c.choice, c.thinking, Forcing{Provider: "claude", Model: "claude-opus-5-5"})
		if !maps.Equal(got, c.want) {
			t.Errorf("ToolChoice(%q, thinking %v) on claude-opus-5-5 = %v, want %v", c.choice, c.thinking, got, c.want)
		}
	}
}

// bodyFor is a Messages body for items, as the providers build it.
func bodyFor(items ...llmprovider.Item) map[string]any {
	return map[string]any{"messages": FromItems(items)}
}

// lastMessage is the last entry of body's messages, as JSON.
func lastMessage(t *testing.T, body map[string]any) string {
	t.Helper()
	msgs := body["messages"].([]Message)
	raw, err := json.Marshal(msgs[len(msgs)-1])
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

var (
	userHi      = llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}
	assistantOK = llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "ok"}
)

// namedInstruction is 0034-MADR D2's text for a named tool.
const namedInstruction = "Tool-use requirement for the current turn: the application requires a call to the get_weather tool in " +
	"your response to the user's latest message. Begin your response with the get_weather tool call. Do not reply with text only."

// TestAddToolChoice_TurnInstruction_0034 (0034-MADR D2): a forced choice on a
// refusing model appends Anthropic's documented mid-conversation system
// message after the last user turn, naming the tool, or one of the tools. A
// named tool takes this form only when the body does not offer it, as here:
// offered, it is sent as structured outputs (D8,
// TestAddToolChoice_StructuredOutput_0034).
func TestAddToolChoice_TurnInstruction_0034(t *testing.T) {
	body := bodyFor(userHi)
	AddToolChoice(body, llmprovider.ForceTool(choiceTool), false, Forcing{Provider: "claude", Model: "claude-sonnet-5-5"})
	if want := `{"content":"` + namedInstruction + `","role":"system"}`; lastMessage(t, body) != want {
		t.Errorf("last message = %s, want %s", lastMessage(t, body), want)
	}
	if got := body["tool_choice"]; !maps.Equal(got.(map[string]any), map[string]any{"type": "auto"}) {
		t.Errorf("tool_choice = %v, want auto", got)
	}
	if n := len(body["messages"].([]Message)); n != 2 {
		t.Errorf("%d messages, want the user turn and the system message", n)
	}

	// With thinking too: the model refuses forcing whatever thinking does.
	body = bodyFor(userHi)
	AddToolChoice(body, llmprovider.ForceTool(choiceTool), true, Forcing{Provider: "claude", Model: "claude-sonnet-5-5"})
	if want := `{"content":"` + namedInstruction + `","role":"system"}`; lastMessage(t, body) != want {
		t.Errorf("with thinking, last message = %s, want %s", lastMessage(t, body), want)
	}

	body = bodyFor(userHi)
	AddToolChoice(body, llmprovider.ToolChoiceRequired, false, Forcing{Provider: "claude", Model: "claude-fable-5-1"})
	if got := lastMessage(t, body); !strings.Contains(got, "a call to one of the tools") || !strings.Contains(got, `"role":"system"`) {
		t.Errorf("ToolChoiceRequired's last message = %s, want the one-of-the-tools instruction", got)
	}
}

// TestAddToolChoice_Placement_0034 (0034-MADR D2): with an input that does not
// end with a user turn, the text goes to the top-level system, after any
// system text there, and no system message is appended.
func TestAddToolChoice_Placement_0034(t *testing.T) {
	body := bodyFor(userHi, assistantOK)
	AddToolChoice(body, llmprovider.ForceTool(choiceTool), false, Forcing{Provider: "claude", Model: "claude-opus-5-5"})
	if n := len(body["messages"].([]Message)); n != 2 {
		t.Errorf("%d messages, want 2: no system message after an assistant turn", n)
	}
	if got := body["system"]; got != namedInstruction {
		t.Errorf("system = %v, want the instruction", got)
	}

	body = bodyFor(userHi, assistantOK)
	body["system"] = "Be brief."
	AddToolChoice(body, llmprovider.ForceTool(choiceTool), false, Forcing{Provider: "claude", Model: "claude-opus-5-5"})
	if got := body["system"]; got != "Be brief.\n\n"+namedInstruction {
		t.Errorf("system = %q, want the caller's system, then the instruction", got)
	}
}

// TestAddToolChoice_NoInstructionOtherwise_0034 (0034-MADR D2; R45): a model
// outside the table, or a choice that does not force, adds no instruction.
func TestAddToolChoice_NoInstructionOtherwise_0034(t *testing.T) {
	for _, c := range []struct {
		choice llmprovider.ToolChoice
		model  string
	}{
		{llmprovider.ForceTool(choiceTool), "claude-sonnet-5"},
		{llmprovider.ToolChoiceRequired, "claude-haiku-5-5"},
		{llmprovider.ToolChoiceNone, "claude-sonnet-5-5"},
		{llmprovider.ToolChoiceAuto, "claude-sonnet-5-5"},
	} {
		body := bodyFor(userHi)
		AddToolChoice(body, c.choice, false, Forcing{Provider: "claude", Model: c.model})
		if n := len(body["messages"].([]Message)); n != 1 {
			t.Errorf("%s on %s: %d messages, want 1", c.choice, c.model, n)
		}
		if _, ok := body["system"]; ok {
			t.Errorf("%s on %s: a system field was added", c.choice, c.model)
		}
	}
}
