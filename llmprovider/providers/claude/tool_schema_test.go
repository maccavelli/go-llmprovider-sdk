package claude

import (
	"context"
	"reflect"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestClaude_NilToolSchemaIsEmptyObject (0021-MADR W11): a tool with no schema
// is sent as an object with no properties; Anthropic refuses a null
// input_schema.
func TestClaude_NilToolSchemaIsEmptyObject(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, okReply)
	p := build(t, apiKey(srv.URL, "claude-test")...)
	if _, err := p.Generate(context.Background(), withTool(text("hi"), llmprovider.Tool{Name: "x"})); err != nil {
		t.Fatal(err)
	}
	tools, _ := body["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("tools = %v, want one", body["tools"])
	}
	got := tools[0].(map[string]any)["input_schema"]
	want := map[string]any{"type": "object", "properties": map[string]any{}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("input_schema = %#v, want %#v", got, want)
	}
}
