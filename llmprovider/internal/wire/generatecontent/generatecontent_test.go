package generatecontent

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// thoughtSummaryResponse is generateContent's shape with includeThoughts
// (gemini-3.7-flash, 2026-09-27): the summary part carries thought: true, a
// boolean, and its text in "text".
const thoughtSummaryResponse = `{"candidates":[{"content":{"parts":[
{"thought":true,"text":"Multiply 17 by 23."},
{"text":"391","thoughtSignature":"sig-abc"}]}}]}`

// TestGeminiDecode_ThoughtSummaryPart: the thought part is reasoning, the
// answer is the only message, and the response decodes (MADR 0014 §3).
func TestGeminiDecode_ThoughtSummaryPart(t *testing.T) {
	res, err := Decode(strings.NewReader(thoughtSummaryResponse))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(res.Output) != 2 {
		t.Fatalf("output = %#v, want a ReasoningItem and a MessageItem", res.Output)
	}
	if r, ok := res.Output[0].(llmprovider.ReasoningItem); !ok || r.Text != "Multiply 17 by 23." {
		t.Errorf("output[0] = %#v, want the thought summary", res.Output[0])
	}
	if res.OutputText() != "391" {
		t.Errorf("OutputText = %q, want 391", res.OutputText())
	}
}
