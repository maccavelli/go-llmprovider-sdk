package wire

import (
	"reflect"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

var emptyObject = map[string]any{"type": "object", "properties": map[string]any{}}

// TestToolSchema (0021-MADR W11): a nil schema is an object with no
// properties; any other is sent as given.
func TestToolSchema(t *testing.T) {
	if got := ToolSchema(nil); !reflect.DeepEqual(got, emptyObject) {
		t.Errorf("ToolSchema(nil) = %#v", got)
	}
	schema := map[string]any{"type": "object"}
	if got := ToolSchema(schema); !reflect.DeepEqual(got, schema) {
		t.Errorf("ToolSchema(schema) = %#v", got)
	}
}

// TestAddResponsesTools: the Responses tool list, and tool_choice for each
// choice; no tools adds nothing.
func TestAddResponsesTools(t *testing.T) {
	tools := []llmprovider.Tool{{Name: "w", Description: "d"}}
	for _, c := range []struct {
		choice llmprovider.ToolChoice
		want   any
	}{
		{llmprovider.ToolChoiceAuto, nil},
		{llmprovider.ToolChoiceRequired, "required"},
		{llmprovider.ToolChoiceNone, "none"},
		{llmprovider.ForceTool("w"), map[string]any{"type": "function", "name": "w"}},
	} {
		body := map[string]any{}
		AddResponsesTools(body, tools, c.choice)
		wantTools := []map[string]any{{"type": "function", "name": "w", "description": "d", "parameters": emptyObject}}
		if !reflect.DeepEqual(body["tools"], wantTools) || !reflect.DeepEqual(body["tool_choice"], c.want) {
			t.Errorf("choice %q: %#v", c.choice, body)
		}
	}
	body := map[string]any{}
	AddResponsesTools(body, nil, llmprovider.ToolChoiceRequired)
	if len(body) != 0 {
		t.Errorf("no tools: %#v", body)
	}
}

// TestMessagesTools: the Messages tool list, with input_schema.
func TestMessagesTools(t *testing.T) {
	got := MessagesTools([]llmprovider.Tool{{Name: "w", Description: "d"}})
	want := []map[string]any{{"name": "w", "description": "d", "input_schema": emptyObject}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MessagesTools = %#v", got)
	}
}
