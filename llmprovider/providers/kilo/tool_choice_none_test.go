package kilo

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestKilo_ToolChoiceNone (0020-MADR F40): a model that takes tool_choice is
// sent "none"; one that does not gets no tools, so it cannot call one.
func TestKilo_ToolChoiceNone(t *testing.T) {
	tool := llmprovider.Tool{Name: "get_weather", Schema: map[string]any{"type": "object"}}
	for _, c := range []struct {
		name       string
		caps       []string
		wantTools  bool
		wantChoice any
	}{
		{"tool_choice accepted", []string{paramTools, paramToolChoice}, true, "none"},
		{"tool_choice not accepted", []string{paramTools}, false, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			var body map[string]any
			srv := captureServer(t, &body, fxKiloChat)
			req := withTool(text("hi"), tool)
			req.ToolChoice = llmprovider.ToolChoiceNone
			if _, err := build(t, apiKey(srv.URL, "kilo-auto/free", WithCapabilities(c.caps...))...).Generate(context.Background(), req); err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if _, ok := body[paramTools]; ok != c.wantTools {
				t.Errorf("tools present = %v, want %v", ok, c.wantTools)
			}
			if body[paramToolChoice] != c.wantChoice {
				t.Errorf("tool_choice = %v, want %v", body[paramToolChoice], c.wantChoice)
			}
		})
	}
}
