package wire

import (
	"encoding/json"
	"testing"
)

// TestToolSchema_TypedNil (0026-MADR F27): a nil map or a nil RawMessage is
// no schema, so it goes out as an empty object, never null, which Anthropic
// refuses (0021-MADR W11).
func TestToolSchema_TypedNil(t *testing.T) {
	for name, schema := range map[string]any{
		"nil":            nil,
		"nil map":        map[string]any(nil),
		"nil RawMessage": json.RawMessage(nil),
	} {
		got, err := json.Marshal(ToolSchema(schema))
		if err != nil || string(got) != `{"properties":{},"type":"object"}` {
			t.Errorf("%s: %s, %v; want an empty object schema", name, got, err)
		}
	}
	if got, _ := json.Marshal(ToolSchema(map[string]any{"type": "object"})); string(got) != `{"type":"object"}` {
		t.Errorf("a schema: %s; want it as given", got)
	}
}
