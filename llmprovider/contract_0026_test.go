package llmprovider

import (
	"encoding/json"
	"errors"
	"testing"
)

// TestValidate_UnmarshalableSchema (0026-MADR F20): a tool schema no request
// could carry is ErrInvalidRequest before any network call (R23, R25).
func TestValidate_UnmarshalableSchema(t *testing.T) {
	for name, schema := range map[string]any{
		"a channel":            map[string]any{"type": "object", "x": make(chan int)},
		"invalid raw JSON":     json.RawMessage(`{"type":`),
		"an unsupported value": func() {},
	} {
		req := &Request{Tools: []Tool{{Name: "f", Schema: schema}}}
		if err := (Capabilities{Tools: BestEffort}).Check(req); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s: Check = %v; want ErrInvalidRequest", name, err)
		}
	}
	ok := &Request{Tools: []Tool{{Name: "f", Schema: json.RawMessage(`{"type":"object"}`)}, {Name: "g"}}}
	if err := (Capabilities{Tools: BestEffort}).Check(ok); err != nil {
		t.Errorf("a valid schema and a nil one: Check = %v; want nil", err)
	}
}
