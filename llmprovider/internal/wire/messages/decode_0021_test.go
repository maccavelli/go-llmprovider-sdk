package messages

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

func onlyCall(t *testing.T, res *llmprovider.Response) llmprovider.FunctionCallItem {
	t.Helper()
	for _, it := range res.Output {
		if call, ok := it.(llmprovider.FunctionCallItem); ok {
			return call
		}
	}
	t.Fatalf("no call in %+v", res.Output)
	return llmprovider.FunctionCallItem{}
}

// TestDecode_ToolInputKeepsPrecision (0021-MADR W1): a tool_use input comes
// back byte for byte: a large integer keeps its digits, the keys their order.
func TestDecode_ToolInputKeepsPrecision(t *testing.T) {
	res, err := Decode(strings.NewReader(`{"stop_reason":"tool_use","content":[` +
		`{"type":"tool_use","id":"t1","name":"f","input":{"z":1,"id":12345678901234567890}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := onlyCall(t, res).Arguments; got != `{"z":1,"id":12345678901234567890}` {
		t.Errorf("Arguments = %s, want the input exactly", got)
	}
}

// TestDecode_ToolInputMissingIsEmptyObject (0021-MADR W1): a call with no
// input has the empty object, as the other wires give it.
func TestDecode_ToolInputMissingIsEmptyObject(t *testing.T) {
	res, err := Decode(strings.NewReader(`{"stop_reason":"tool_use","content":[{"type":"tool_use","id":"t1","name":"now"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := onlyCall(t, res).Arguments; got != "{}" {
		t.Errorf("Arguments = %q, want {}", got)
	}
}

// TestFinish_Messages (0021-MADR W3): a context-window stop with a call is a
// cut call; an unknown reason is kept; a call finishes tool_calls; an empty
// refusal keeps its reason.
func TestFinish_Messages(t *testing.T) {
	_, err := Decode(strings.NewReader(`{"stop_reason":"model_context_window_exceeded","content":[` +
		`{"type":"tool_use","id":"t1","name":"f","input":{"a":`)) // never read whole: the decode fails first
	if err == nil {
		t.Fatal("a cut body decoded")
	}
	var apiErr *llmprovider.APIError
	_, err = Decode(strings.NewReader(`{"stop_reason":"model_context_window_exceeded","content":[` +
		`{"type":"tool_use","id":"t1","name":"f","input":{"a":1}}]}`))
	if !errors.As(err, &apiErr) || !errors.Is(err, llmprovider.ErrIncomplete) || apiErr.Reason != "length" {
		t.Errorf("model_context_window_exceeded with a call: %v; want ErrIncomplete, reason length", err)
	}
	res, err := Decode(strings.NewReader(`{"stop_reason":"new_reason","content":[{"type":"text","text":"hi"}]}`))
	if err != nil || res.FinishReason != "new_reason" {
		t.Errorf("an unknown stop_reason: %v, %v; want it kept", res, err)
	}
	res, err = Decode(strings.NewReader(`{"stop_reason":"end_turn","content":[{"type":"tool_use","id":"t1","name":"f","input":{}}]}`))
	if err != nil || res.FinishReason != llmprovider.FinishToolCalls {
		t.Errorf("end_turn with a call: %v, %v; want tool_calls", res, err)
	}
	_, err = Decode(strings.NewReader(`{"stop_reason":"refusal","content":[]}`))
	if !errors.As(err, &apiErr) || !errors.Is(err, llmprovider.ErrIncomplete) || apiErr.Reason != "content_filter" {
		t.Errorf("an empty refusal: %v; want ErrIncomplete, reason content_filter", err)
	}
}
