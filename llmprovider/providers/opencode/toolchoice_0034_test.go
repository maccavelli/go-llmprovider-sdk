package opencode

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestGenerate_RefusingModelDegrades_0034 (0034-MADR D2): Zen routes
// claude-opus-5-5 to the messages route, where a forced choice is sent as auto
// with the turn instruction, as the claude provider sends it; auto and none
// are unchanged.
func TestGenerate_RefusingModelDegrades_0034(t *testing.T) {
	named := llmprovider.ForceTool("get_weather")
	for _, tc := range []struct {
		choice      llmprovider.ToolChoice
		toolChoice  string
		instruction string // in the last message, or "" for none appended
	}{
		{named, "map[type:auto]", "a call to the get_weather tool"},
		{llmprovider.ToolChoiceRequired, "map[type:auto]", "a call to one of the tools"},
		{llmprovider.ToolChoiceNone, "map[type:none]", ""},
		{llmprovider.ToolChoiceAuto, "<nil>", ""},
	} {
		var body map[string]any
		srv := captureServer(t, &body, fxMessages)
		req := text("hi")
		req.Tools = []llmprovider.Tool{{Name: "get_weather", Schema: map[string]any{"type": "object"}}}
		req.ToolChoice = tc.choice
		if _, err := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "claude-opus-5-5")...).Generate(context.Background(), req); err != nil {
			t.Fatalf("%q: Generate: %v", tc.choice, err)
		}
		if got := fmt.Sprint(body["tool_choice"]); got != tc.toolChoice {
			t.Errorf("%q: tool_choice = %s, want %s", tc.choice, got, tc.toolChoice)
		}
		msgs, _ := body["messages"].([]any)
		last := fmt.Sprint(msgs[len(msgs)-1])
		switch {
		case tc.instruction != "" && (!strings.Contains(last, "role:system") || !strings.Contains(last, tc.instruction)):
			t.Errorf("%q: last message = %s, want a system message holding %q", tc.choice, last, tc.instruction)
		case tc.instruction == "" && len(msgs) != 1:
			t.Errorf("%q: %d messages, want only the user's", tc.choice, len(msgs))
		}
	}
}
