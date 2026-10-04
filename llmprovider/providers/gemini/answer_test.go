package gemini

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestDecodeInteraction_ModelAndFinish (0020-MADR F11): the model that
// answered, and stop or tool_calls.
func TestDecodeInteraction_ModelAndFinish(t *testing.T) {
	for _, c := range []struct {
		status, step string
		want         llmprovider.FinishReason
	}{
		{"completed", `{"type":"model_output","content":[{"type":"text","text":"hi"}]}`, llmprovider.FinishStop},
		{"requires_action", `{"type":"function_call","id":"c","name":"f","arguments":{}}`, llmprovider.FinishToolCalls},
	} {
		body := `{"id":"i","model":"gemini-served","status":"` + c.status + `","steps":[` + c.step + `]}`
		res, err := decodeInteraction(strings.NewReader(body))
		if err != nil || res.Model != "gemini-served" || res.FinishReason != c.want {
			t.Errorf("%s: decode = %+v, %v; want model gemini-served, finish %q", c.status, res, err, c.want)
		}
	}
}

// TestDecodeInteraction_EmptyIsIncomplete (0020-MADR F9): a thought-only
// interaction has nothing to give, and says so by kind.
func TestDecodeInteraction_EmptyIsIncomplete(t *testing.T) {
	_, err := decodeInteraction(strings.NewReader(`{"id":"i","status":"completed","steps":[{"type":"thought","signature":"s"}]}`))
	if !errors.Is(err, llmprovider.ErrIncomplete) {
		t.Errorf("decode = %v, want ErrIncomplete", err)
	}
}
