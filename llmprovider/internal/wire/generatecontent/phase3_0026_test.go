package generatecontent

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// incompleteReason is err's Reason when err is an ErrIncomplete *APIError.
func incompleteReason(t *testing.T, err error) string {
	t.Helper()
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	if !ok || !errors.Is(err, llmprovider.ErrIncomplete) {
		t.Fatalf("err = %v; want an ErrIncomplete *APIError", err)
	}
	return apiErr.Reason
}

// TestDecode_ReasoningOnlyCutIsIncomplete (0026-MADR F7): a thought summary
// with no text or call, cut by MAX_TOKENS, is ErrIncomplete with Reason
// "length".
func TestDecode_ReasoningOnlyCutIsIncomplete(t *testing.T) {
	body := `{"candidates":[{"finishReason":"MAX_TOKENS","content":{"parts":[{"text":"thinking about it","thought":true}]}}]}`
	_, err := Decode(strings.NewReader(body))
	if got := incompleteReason(t, err); got != string(llmprovider.FinishLength) {
		t.Fatalf("Reason = %q; want length", got)
	}
}

// TestGenerateContent_BlockReason (0026-MADR F25): a prompt Gemini blocks has
// no candidates, and its promptFeedback.blockReason is the Reason.
func TestGenerateContent_BlockReason(t *testing.T) {
	_, err := Decode(strings.NewReader(`{"promptFeedback":{"blockReason":"SAFETY"},"modelVersion":"gemini-x"}`))
	if got := incompleteReason(t, err); got != "SAFETY" {
		t.Fatalf("Reason = %q; want SAFETY", got)
	}
}

// TestGenerateContent_ToolCallFailureReasons (0026-MADR F26): Gemini's
// reasons for a failed tool call are kept as sent, not reported as "stop".
func TestGenerateContent_ToolCallFailureReasons(t *testing.T) {
	for _, reason := range []string{"MALFORMED_FUNCTION_CALL", "UNEXPECTED_TOOL_CALL"} {
		_, err := Decode(strings.NewReader(`{"candidates":[{"finishReason":"` + reason + `","content":{}}]}`))
		if got := incompleteReason(t, err); got != reason {
			t.Errorf("Reason = %q; want %s", got, reason)
		}
	}
}
