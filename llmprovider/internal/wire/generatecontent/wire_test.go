package generatecontent

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// The generateContent wire at its own level (0015-PLAN S7b); the opencode
// package tests it through Generate.

func TestSystemInstruction(t *testing.T) {
	if got := SystemInstruction([]llmprovider.Item{llmprovider.MessageItem{Role: "user", Text: "hi"}}); got != nil {
		t.Errorf("no system items: %v, want nil", got)
	}
	got := SystemInstruction([]llmprovider.Item{llmprovider.MessageItem{Role: "system", Text: "Be brief."}})
	mustEqualJSON(t, got, `{"parts":[{"text":"Be brief."}]}`)
}

// TestContents_ResultWithoutItsCall: a result whose call is not in the
// history takes its call id as the name; a system item is left out.
func TestContents_ResultWithoutItsCall(t *testing.T) {
	got := Contents([]llmprovider.Item{
		llmprovider.MessageItem{Role: "system", Text: "Be brief."},
		llmprovider.MessageItem{Role: "assistant", Text: "earlier"},
		llmprovider.FunctionCallOutputItem{CallID: "c9", Output: "sunny"},
	})
	mustEqualJSON(t, got, `[{"role":"model","parts":[{"text":"earlier"}]},`+
		`{"role":"user","parts":[{"functionResponse":{"name":"c9","response":{"output":"sunny"}}}]}]`)
}

func TestDecode_Errors(t *testing.T) {
	for name, body := range map[string]string{
		"no candidates": `{"candidates":[]}`,
		"no parts":      `{"candidates":[{"content":{"parts":[]}}]}`,
		"not JSON":      `not json`,
	} {
		if _, err := Decode(strings.NewReader(body)); err == nil {
			t.Errorf("%s: decoded", name)
		}
	}
	res, err := Decode(strings.NewReader(`{"candidates":[{"content":{"parts":[{"thought":true,"text":""},{"text":"ok"}]}}]}`))
	if err != nil || len(res.Output) != 1 || res.OutputText() != "ok" {
		t.Errorf("an empty thought: %+v, %v; want only the text", res, err)
	}
}

// TestThinkingConfig: a budget wins; any effort but "low" is dynamic; "low" is
// a budget on Gemini 1.x and 2.x and a level on 3 and later.
func TestThinkingConfig(t *testing.T) {
	for _, tc := range []struct {
		name, model, effort string
		budget              int
		want                string
	}{
		{"budget", "gemini-3.7-flash", "low", 500, `{"includeThoughts":true,"thinkingBudget":500}`},
		{"high", "gemini-3.7-flash", "high", 0, `{"includeThoughts":true,"thinkingBudget":-1}`},
		{"none", "gemini-2.5-pro", "", 0, `{"includeThoughts":true,"thinkingBudget":-1}`},
		{"low, legacy", "gemini-2.5-flash", "low", 0, `{"includeThoughts":true,"thinkingBudget":1024}`},
		{"low, legacy with family", "gemini-flash-1.5", "low", 0, `{"includeThoughts":true,"thinkingBudget":1024}`},
		{"low, current", "gemini-3.7-flash", "low", 0, `{"includeThoughts":true,"thinkingLevel":"low"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mustEqualJSON(t, ThinkingConfig(tc.model, tc.effort, tc.budget), tc.want)
		})
	}
}
