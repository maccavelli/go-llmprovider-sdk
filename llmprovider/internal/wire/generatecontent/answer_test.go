package generatecontent

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestDecode_FinishReasonAndModel (0020-MADR F3, F11): finishReason becomes
// FinishReason and modelVersion the model. A MAX_TOKENS reply whose only part
// is a call with {} keeps it, a whole call that takes no arguments, finishing
// length with no warning: a functionCall part arrives whole (0088-MADR ST107,
// ST111, which replace F3's ErrIncomplete for it).
func TestDecode_FinishReasonAndModel(t *testing.T) {
	text := `{"text":"hi"}`
	call := `{"functionCall":{"name":"f","args":{}}}`
	for _, c := range []struct {
		reason, part string
		want         llmprovider.FinishReason
	}{
		{"STOP", text, llmprovider.FinishStop},
		{"MAX_TOKENS", text, llmprovider.FinishLength},
		{"SAFETY", text, llmprovider.FinishContentFilter},
		{"STOP", call, llmprovider.FinishToolCalls},
	} {
		body := `{"modelVersion":"gemini-served","candidates":[{"finishReason":"` + c.reason +
			`","content":{"parts":[` + c.part + `]}}]}`
		res, err := Decode(strings.NewReader(body))
		if err != nil || res.FinishReason != c.want || res.Model != "gemini-served" {
			t.Errorf("%s %s: Decode = %+v, %v; want finish %q, model gemini-served", c.reason, c.part, res, err, c.want)
		}
	}
	res, err := Decode(strings.NewReader(`{"candidates":[{"finishReason":"MAX_TOKENS","content":{"parts":[` + call + `]}}]}`))
	if err != nil || res.FinishReason != llmprovider.FinishLength || len(res.Output) != 1 || len(res.Warnings) != 0 {
		t.Fatalf("a call with {} cut by MAX_TOKENS = %+v, %v; want the call f {} kept, finishing length, no warning", res, err)
	}
	if call, ok := res.Output[0].(llmprovider.FunctionCallItem); !ok || call.Name != "f" || call.Arguments != "{}" {
		t.Errorf("a call with {} cut by MAX_TOKENS: output %+v; want the call f {}", res.Output)
	}
}

// TestDecode_EmptyIsIncomplete (0020-MADR F9).
func TestDecode_EmptyIsIncomplete(t *testing.T) {
	if _, err := Decode(strings.NewReader(`{"candidates":[]}`)); !errors.Is(err, llmprovider.ErrIncomplete) {
		t.Errorf("Decode(no candidates) = %v, want ErrIncomplete", err)
	}
}
