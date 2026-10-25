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

// TestThinkingConfig: with no facts, a Budget is thinkingBudget, and an effort,
// or none, the level: thinkingLevel the effort, nothing for none (0062-MADR
// D2, amendment "Phase 3's google-route levels, and 0013's B9"; R4 removed).
func TestThinkingConfig(t *testing.T) {
	for _, tc := range []struct {
		name, effort string
		budget       int
		want         string
	}{
		{"budget", "low", 500, `{"includeThoughts":true,"thinkingBudget":500}`},
		{"high", "high", 0, `{"includeThoughts":true,"thinkingLevel":"high"}`},
		{"none", "", 0, `{"includeThoughts":true}`},
		{"low", "low", 0, `{"includeThoughts":true,"thinkingLevel":"low"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mustEqualJSON(t, ThinkingConfig(tc.effort, tc.budget), tc.want)
		})
	}
}
