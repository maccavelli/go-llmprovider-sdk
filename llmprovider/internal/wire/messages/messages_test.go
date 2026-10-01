package messages

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// The Messages wire at its own level (0015-PLAN S7b); the claude and opencode
// packages test it through Generate.

func asJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestFromItems: system items are left out, a role other than user is the
// assistant, and calls join the assistant's text while results share a user
// turn.
func TestFromItems(t *testing.T) {
	got := asJSON(t, FromItems([]llmprovider.Item{
		llmprovider.MessageItem{Role: "system", Text: "Be brief."},
		llmprovider.MessageItem{Text: "weather?"},
		llmprovider.MessageItem{Role: "model", Text: "Checking."},
		llmprovider.FunctionCallItem{CallID: "c1", Name: "get_weather", Arguments: `{"city":"Paris"}`},
		llmprovider.FunctionCallItem{CallID: "c2", Name: "now"},
		llmprovider.FunctionCallOutputItem{CallID: "c1", Output: "sunny"},
		llmprovider.FunctionCallOutputItem{CallID: "c2", Output: "noon"},
		llmprovider.ReasoningItem{Text: "not sent"},
	}))
	want := `[{"content":"weather?","role":"user"},` +
		`{"content":[{"text":"Checking.","type":"text"},` +
		`{"id":"c1","input":{"city":"Paris"},"name":"get_weather","type":"tool_use"},` +
		`{"id":"c2","input":{},"name":"now","type":"tool_use"}],"role":"assistant"},` +
		`{"content":[{"content":"sunny","tool_use_id":"c1","type":"tool_result"},` +
		`{"content":"noon","tool_use_id":"c2","type":"tool_result"}],"role":"user"}]`
	if got != want {
		t.Errorf("FromItems = %s\nwant        %s", got, want)
	}
	// A result after a user's text opens its own turn: the text stays a string.
	got = asJSON(t, FromItems([]llmprovider.Item{
		llmprovider.MessageItem{Role: "user", Text: "hi"},
		llmprovider.FunctionCallOutputItem{CallID: "c1", Output: "sunny"},
	}))
	if want := `[{"content":"hi","role":"user"},{"content":[{"content":"sunny","tool_use_id":"c1","type":"tool_result"}],"role":"user"}]`; got != want {
		t.Errorf("FromItems = %s\nwant        %s", got, want)
	}
}

// TestDecode: thinking (with text as its fallback), text and tool_use become
// their items; empty or unusable content and a bad body are errors.
func TestDecode(t *testing.T) {
	res, err := Decode(strings.NewReader(`{"content":[
		{"type":"thinking","thinking":"plan"},{"type":"thinking","text":"fallback"},
		{"type":"text","text":"hello"},{"type":"text","text":""},
		{"type":"tool_use","id":"t1","name":"get_weather","input":{"city":"Paris"}},
		{"type":"tool_use","id":"t2","name":"now"},
		{"type":"server_tool_use"}]}`))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	want := []llmprovider.Item{
		llmprovider.ReasoningItem{Text: "plan"},
		llmprovider.ReasoningItem{Text: "fallback"},
		llmprovider.MessageItem{Role: "assistant", Text: "hello"},
		llmprovider.FunctionCallItem{CallID: "t1", Name: "get_weather", Arguments: `{"city":"Paris"}`},
		llmprovider.FunctionCallItem{CallID: "t2", Name: "now"},
	}
	if res.ID != "" || fmt.Sprint(res.Output) != fmt.Sprint(want) {
		t.Errorf("Decode = %+v, want %v and no id", res.Output, want)
	}
	for name, body := range map[string]string{
		"empty content":   `{"content":[]}`,
		"no usable block": `{"content":[{"type":"text","text":""}]}`,
		"not JSON":        `not json`,
	} {
		if _, err := Decode(strings.NewReader(body)); err == nil {
			t.Errorf("%s: decoded", name)
		}
	}
}

// TestClaudeAdaptiveOnly: Claude 4.7 and later, and a claude- id with no
// readable version, refuse an enabled budget; earlier Claude ids and other
// models take one.
func TestClaudeAdaptiveOnly(t *testing.T) {
	for model, want := range map[string]bool{
		"claude-opus-4-8":          true,
		"claude-4.7-opus":          true,
		"claude-sonnet-5":          true,
		"claude-opus-4-6":          false,
		"claude-sonnet-4-20250514": false,
		"claude-3-5-haiku":         false,
		"claude-unversioned":       true,
		"glm-5.3":                  false,
	} {
		if got := claudeAdaptiveOnly(model); got != want {
			t.Errorf("claudeAdaptiveOnly(%q) = %t, want %t", model, got, want)
		}
	}
}

// TestAddThinking: MiniMax and adaptive-only Claude take thinking.type
// "adaptive" (with output_config.effort when an effort is set); others take
// an enabled budget, with max_tokens raised above it.
func TestAddThinking(t *testing.T) {
	for _, tc := range []struct {
		name, model, effort string
		budget, maxTokens   int
		want                string
		wantMax             int
	}{
		{"minimax", "MiniMax-M3", "high", 0, 100, `{"thinking":{"type":"adaptive"}}`, 100},
		{"adaptive, effort", "claude-opus-4-8", "high", 9000, 100, `{"output_config":{"effort":"high"},"thinking":{"type":"adaptive"}}`, 100},
		{"adaptive, no effort", "claude-opus-4-8", "", 0, 100, `{"thinking":{"type":"adaptive"}}`, 100},
		{"budget by default", "claude-opus-4-6", "", 0, 8192, `{"thinking":{"budget_tokens":4096,"type":"enabled"}}`, 8192},
		{"low effort", "claude-opus-4-6", "low", 0, 8192, `{"thinking":{"budget_tokens":1024,"type":"enabled"}}`, 8192},
		{"budget given", "glm-5.3", "high", 2000, 8192, `{"thinking":{"budget_tokens":2000,"type":"enabled"}}`, 8192},
		{"max_tokens raised", "claude-opus-4-6", "", 0, 4096, `{"thinking":{"budget_tokens":4096,"type":"enabled"}}`, 8192},
	} {
		body := map[string]any{}
		gotMax := AddThinking(body, tc.model, tc.effort, tc.budget, tc.maxTokens)
		if got := asJSON(t, body); got != tc.want || gotMax != tc.wantMax {
			t.Errorf("%s: body %s, max_tokens %d; want %s, %d", tc.name, got, gotMax, tc.want, tc.wantMax)
		}
	}
}
