package messages

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// citySchema is a strict-ready schema, as 0034-PLAN Phase 4b measured.
var citySchema = map[string]any{"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}},
	"required": []any{"city"}, "additionalProperties": false}

// structuredBody is a body as the providers build it, offering get_weather.
func structuredBody() map[string]any {
	body := bodyFor(userHi)
	body["tools"] = wire.MessagesTools([]llmprovider.Tool{{Name: choiceTool, Schema: citySchema}, {Name: "get_time"}})
	return body
}

// TestAddToolChoice_StructuredOutput_0034 (0034-MADR D8): a forced named tool
// on a refusing model is sent as structured outputs with that tool's schema,
// the tools kept under tool_choice none, an effort kept, and no instruction.
func TestAddToolChoice_StructuredOutput_0034(t *testing.T) {
	for _, thinking := range []bool{false, true} {
		body := structuredBody()
		body["output_config"] = map[string]any{"effort": "low"}
		AddToolChoice(body, llmprovider.ForceTool(choiceTool), thinking, Forcing{Provider: "claude", Model: "claude-sonnet-5-5"})
		if got := body["tool_choice"]; !maps.Equal(got.(map[string]any), map[string]any{"type": "none"}) {
			t.Errorf("thinking %v: tool_choice = %v, want none", thinking, got)
		}
		cfg, _ := body["output_config"].(map[string]any)
		format, _ := cfg["format"].(map[string]any)
		raw, _ := json.Marshal(format["schema"])
		want, _ := json.Marshal(citySchema)
		if format["type"] != "json_schema" || string(raw) != string(want) || cfg["effort"] != "low" {
			t.Errorf("thinking %v: output_config = %v, want the effort and a json_schema format with get_weather's schema", thinking, cfg)
		}
		if n := len(body["messages"].([]Message)); n != 1 {
			t.Errorf("thinking %v: %d messages, want no instruction appended", thinking, n)
		}
		if _, ok := body["tools"]; !ok {
			t.Errorf("thinking %v: the tools were dropped", thinking)
		}
	}
}

// TestToolChoice_NamedOnRefusing_0034 (0034-MADR D8): the named choice maps
// to none on a refusing model, with or without thinking; on an accepting
// model it is unchanged.
func TestToolChoice_NamedOnRefusing_0034(t *testing.T) {
	named := llmprovider.ForceTool(choiceTool)
	for _, thinking := range []bool{false, true} {
		if got := ToolChoice(named, thinking, Forcing{Provider: "claude", Model: "claude-opus-5-5"}); !maps.Equal(got, map[string]any{"type": "none"}) {
			t.Errorf("thinking %v: %v, want none", thinking, got)
		}
	}
	if got := ToolChoice(named, false, Forcing{Provider: "claude", Model: "claude-sonnet-5"}); !maps.Equal(got, map[string]any{"type": "tool", "name": choiceTool}) {
		t.Errorf("accepting model: %v, want the named tool", got)
	}
}

// TestStructuredCall_0034 (0034-MADR D8): a structured-outputs reply's JSON
// text becomes a FunctionCallItem for the tool, finishing tool_calls; the
// reasoning stays; a reply that is not a JSON object is left as it came.
func TestStructuredCall_0034(t *testing.T) {
	req := &llmprovider.Request{Tools: []llmprovider.Tool{{Name: choiceTool}}, ToolChoice: llmprovider.ForceTool(choiceTool),
		Input: []llmprovider.Item{userHi}}
	f := Forcing{Provider: "claude", Model: "claude-sonnet-5-5"}
	resp := &llmprovider.Response{FinishReason: llmprovider.FinishStop, Output: []llmprovider.Item{
		llmprovider.ReasoningItem{Text: "thinking"},
		llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "{ \"city\": \"Paris\" }"},
	}}
	StructuredCall(req, f, resp)
	if resp.FinishReason != llmprovider.FinishToolCalls || len(resp.Output) != 2 {
		t.Fatalf("response = %+v, want the reasoning and one call, finishing tool_calls", resp)
	}
	call, ok := resp.Output[1].(llmprovider.FunctionCallItem)
	if !ok || call.Name != choiceTool || call.Arguments != `{"city":"Paris"}` || !strings.HasPrefix(call.CallID, "toolu_") {
		t.Errorf("call = %+v, want get_weather with compacted arguments and a toolu_ id", resp.Output[1])
	}
	if _, ok := resp.Output[0].(llmprovider.ReasoningItem); !ok {
		t.Errorf("first item = %T, want the reasoning kept", resp.Output[0])
	}

	text := &llmprovider.Response{FinishReason: llmprovider.FinishStop, Output: []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "I cannot do that."}}}
	StructuredCall(req, f, text)
	if text.FinishReason != llmprovider.FinishStop || len(text.Output) != 1 {
		t.Errorf("a text reply was changed: %+v", text)
	}
	accepting := &llmprovider.Response{Output: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: `{"city":"Paris"}`}}}
	StructuredCall(req, Forcing{Provider: "claude", Model: "claude-sonnet-5"}, accepting)
	if _, ok := accepting.Output[0].(llmprovider.MessageItem); !ok {
		t.Error("an accepting model's reply was converted")
	}
}

// TestStructuredStream_0034 (0034-MADR D8): the stream yields no text delta
// for the JSON, and yields the call as an item and in the done Response.
func TestStructuredStream_0034(t *testing.T) {
	req := &llmprovider.Request{Tools: []llmprovider.Tool{{Name: choiceTool}}, ToolChoice: llmprovider.ForceTool(choiceTool),
		Input: []llmprovider.Item{userHi}}
	text := llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: `{"city":"Paris"}`}
	events := []llmprovider.Event{
		{Type: llmprovider.EventReasoningDelta, Text: "thinking"},
		{Type: llmprovider.EventTextDelta, Text: `{"city":`},
		{Type: llmprovider.EventTextDelta, Text: `"Paris"}`},
		{Type: llmprovider.EventItem, Item: text},
		{Type: llmprovider.EventDone, Response: &llmprovider.Response{FinishReason: llmprovider.FinishStop, Output: []llmprovider.Item{text}}},
	}
	seq := func(yield func(llmprovider.Event, error) bool) {
		for _, ev := range events {
			if !yield(ev, nil) {
				return
			}
		}
	}
	var types []llmprovider.EventType
	var item, done llmprovider.Item
	for ev, err := range StructuredStream(req, Forcing{Provider: "claude", Model: "claude-opus-5-5"}, seq) {
		if err != nil {
			t.Fatal(err)
		}
		types = append(types, ev.Type)
		switch ev.Type {
		case llmprovider.EventItem:
			item = ev.Item
		case llmprovider.EventDone:
			done = ev.Response.Output[0]
		}
	}
	if slices.Contains(types, llmprovider.EventTextDelta) {
		t.Errorf("events %v hold a text delta", types)
	}
	call, ok := item.(llmprovider.FunctionCallItem)
	if !ok || call.Name != choiceTool || done != item {
		t.Errorf("item %+v, done output %+v; want the same call in both", item, done)
	}
}

// openSchema is a nested schema that never says additionalProperties, as most
// tool schemas do, with a property named "properties", one object left open
// on purpose, a list of objects, an alternative, and a definition.
func openSchema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{
		"place":      map[string]any{"type": "object", "properties": map[string]any{"country": map[string]any{"type": "string"}}},
		"properties": map[string]any{"type": "string"},
		"open":       map[string]any{"type": "object", "additionalProperties": true},
		"list":       map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{}}},
		"either":     map[string]any{"anyOf": []any{map[string]any{"type": "object"}, map[string]any{"type": "string"}}},
	}, "$defs": map[string]any{"d": map[string]any{"properties": map[string]any{"x": map[string]any{"type": "string"}}}}}
}

// TestClosedSchema_0034 (0034-MADR Q9): the format's schema closes every
// object that does not say otherwise, nested ones included; an object left
// open keeps it; a property named "properties" is not a schema; the caller's
// value is not changed.
func TestClosedSchema_0034(t *testing.T) {
	in := openSchema()
	before, _ := json.Marshal(in)
	got, ok := closedSchema(in).(map[string]any)
	if !ok {
		t.Fatalf("closedSchema = %T, want a map", closedSchema(in))
	}
	if after, _ := json.Marshal(in); string(after) != string(before) {
		t.Errorf("the caller's schema was changed: %s", after)
	}
	props := got["properties"].(map[string]any)
	for name, schema := range map[string]any{
		"the root":        got,
		"place":           props["place"],
		"list's items":    props["list"].(map[string]any)["items"],
		"either's object": props["either"].(map[string]any)["anyOf"].([]any)[0],
		"the definition":  got["$defs"].(map[string]any)["d"],
	} {
		if schema.(map[string]any)["additionalProperties"] != false {
			t.Errorf("%s is not closed: %v", name, schema)
		}
	}
	if props["open"].(map[string]any)["additionalProperties"] != true {
		t.Errorf("an object left open was closed: %v", props["open"])
	}
	if _, set := props["properties"].(map[string]any)["additionalProperties"]; set {
		t.Errorf("a property named properties was taken for a schema: %v", props["properties"])
	}
	if _, set := props["either"].(map[string]any)["anyOf"].([]any)[1].(map[string]any)["additionalProperties"]; set {
		t.Error("a string schema was closed")
	}
}

// TestAddToolChoice_FormatClosed_0034 (0034-MADR Q9): the format carries the
// closed copy; the tool definition in tools is the caller's, unchanged.
func TestAddToolChoice_FormatClosed_0034(t *testing.T) {
	body := bodyFor(userHi)
	body["tools"] = wire.MessagesTools([]llmprovider.Tool{{Name: choiceTool, Schema: openSchema()}})
	AddToolChoice(body, llmprovider.ForceTool(choiceTool), false, Forcing{Provider: "claude", Model: "claude-sonnet-5-5"})
	format := body["output_config"].(map[string]any)["format"].(map[string]any)["schema"].(map[string]any)
	if format["additionalProperties"] != false {
		t.Errorf("the format's schema is not closed: %v", format)
	}
	tool := body["tools"].([]map[string]any)[0]["input_schema"].(map[string]any)
	if _, set := tool["additionalProperties"]; set {
		t.Errorf("the tool definition was closed too: %v", tool)
	}
}
