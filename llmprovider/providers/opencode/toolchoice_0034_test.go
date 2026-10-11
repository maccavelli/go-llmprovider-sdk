package opencode

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestGenerate_RefusingModelDegrades_0034 (0034-MADR D2): Zen routes
// claude-opus-5-5 to the messages route, where a forced choice is sent as auto
// with the turn instruction, as the claude provider sends it; auto and none
// are unchanged. ToolChoiceRequired degrades only when the caller opts in
// (D10), which this test does; TestGenerate_RefusingModelStrict_0034 checks
// the default.
func TestGenerate_RefusingModelDegrades_0034(t *testing.T) {
	named := llmprovider.ForceTool("get_weather")
	for _, tc := range []struct {
		choice      llmprovider.ToolChoice
		toolChoice  string
		instruction string // in the last message, or "" for none appended
	}{
		{named, "map[type:none]", ""}, // structured outputs (D8)
		{llmprovider.ToolChoiceRequired, "map[type:auto]", "a call to one of the tools"},
		{llmprovider.ToolChoiceNone, "map[type:none]", ""},
		{llmprovider.ToolChoiceAuto, "<nil>", ""},
	} {
		var body map[string]any
		srv := captureServer(t, &body, fxMessages)
		req := text("hi")
		req.Tools = []llmprovider.Tool{{Name: "get_weather", Schema: map[string]any{"type": "object"}}}
		req.ToolChoice = tc.choice
		p := build(t, llmprovider.ProviderOpencodeZen,
			apiKey(srv.URL, "claude-opus-5-5", llmprovider.WithForcedChoiceFallback(llmprovider.ForcedChoiceDegrade))...)
		if _, err := p.Generate(context.Background(), req); err != nil {
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

// TestGenerate_RefusingModelStrict_0034 (0034-MADR D10; R11): by default,
// ToolChoiceRequired on a refusing model on the messages route fails before the
// network, with an error matching ErrUnsupported.
func TestGenerate_RefusingModelStrict_0034(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, fxMessages)
	req := text("hi")
	req.Tools = []llmprovider.Tool{{Name: "get_weather", Schema: map[string]any{"type": "object"}}}
	req.ToolChoice = llmprovider.ToolChoiceRequired
	_, err := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "claude-opus-5-5")...).Generate(context.Background(), req)
	if !errors.Is(err, llmprovider.ErrUnsupported) || body != nil {
		t.Errorf("err = %v, body sent %v; want ErrUnsupported and no request", err, body != nil)
	}
}
