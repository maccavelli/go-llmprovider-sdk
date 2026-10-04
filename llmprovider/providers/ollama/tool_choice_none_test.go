package ollama

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestOllama_ToolChoiceNoneSendsNoTools (0020-MADR F40): Ollama cannot take
// tool_choice, so "none" is honoured by leaving the tools out.
func TestOllama_ToolChoiceNoneSendsNoTools(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, fxOllamaChat)
	req := text("hi")
	req.Tools = []llmprovider.Tool{{Name: "get_weather", Schema: map[string]any{"type": "object"}}}
	req.ToolChoice = llmprovider.ToolChoiceNone
	if _, err := build(t, local(srv.URL, "llama3.2:latest")...).Generate(context.Background(), req); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if tools, ok := body["tools"]; ok {
		t.Errorf("tools = %v, want none: the model must not call one", tools)
	}
}
