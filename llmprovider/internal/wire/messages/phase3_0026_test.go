package messages

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestDecode_ReasoningOnlyCutIsIncomplete (0026-MADR F7): thinking with no
// text or call, cut by max_tokens, is ErrIncomplete with Reason "length".
func TestDecode_ReasoningOnlyCutIsIncomplete(t *testing.T) {
	body := `{"stop_reason":"max_tokens","content":[{"type":"thinking","thinking":"thinking about it","signature":"s"}]}`
	res, err := Decode(strings.NewReader(body))
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	if !ok || !errors.Is(err, llmprovider.ErrIncomplete) || apiErr.Reason != string(llmprovider.FinishLength) {
		t.Fatalf("a reasoning-only cut answer: %+v, %v; want ErrIncomplete with Reason length", res, err)
	}
	cut := `{"stop_reason":"max_tokens","content":[{"type":"thinking","thinking":"t","signature":"s"},{"type":"text","text":"partial"}]}`
	if res, err := Decode(strings.NewReader(cut)); err != nil || res.OutputText() != "partial" || res.FinishReason != llmprovider.FinishLength {
		t.Fatalf("a cut text answer: %+v, %v; want its text and FinishLength", res, err)
	}
}
