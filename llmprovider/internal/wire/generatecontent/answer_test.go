package generatecontent

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestDecode_FinishReasonAndModel (0020-MADR F3, F11): finishReason becomes
// FinishReason, modelVersion the model, and a call cut by MAX_TOKENS is
// ErrIncomplete.
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
	_, err := Decode(strings.NewReader(`{"candidates":[{"finishReason":"MAX_TOKENS","content":{"parts":[` + call + `]}}]}`))
	if !errors.Is(err, llmprovider.ErrIncomplete) {
		t.Errorf("a call cut by MAX_TOKENS = %v, want ErrIncomplete", err)
	}
}

// TestDecode_EmptyIsIncomplete (0020-MADR F9).
func TestDecode_EmptyIsIncomplete(t *testing.T) {
	if _, err := Decode(strings.NewReader(`{"candidates":[]}`)); !errors.Is(err, llmprovider.ErrIncomplete) {
		t.Errorf("Decode(no candidates) = %v, want ErrIncomplete", err)
	}
}
