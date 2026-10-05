package chatcompletions

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestDecode_ArgumentsAsObject (0021-MADR W1): a gateway that sends a call's
// arguments as an object, not a string, has them kept, not the reply failed.
func TestDecode_ArgumentsAsObject(t *testing.T) {
	res, err := Decode(strings.NewReader(`{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant",` +
		`"tool_calls":[{"id":"c1","function":{"name":"f","arguments":{"city": "Paris"}}}]}}]}`))
	if err != nil {
		t.Fatalf("an object's arguments: %v", err)
	}
	call, ok := res.Output[0].(llmprovider.FunctionCallItem)
	if !ok || call.Arguments != `{"city":"Paris"}` {
		t.Errorf("Output = %+v, want the arguments compacted", res.Output)
	}
}

// TestFinish_Chat (0021-MADR W3): a reply with a call finishes tool_calls, the
// answer is always the assistant's, and an empty answer keeps its reason.
func TestFinish_Chat(t *testing.T) {
	res, err := Decode(strings.NewReader(`{"choices":[{"finish_reason":"stop","message":{"role":"assistant",` +
		`"tool_calls":[{"id":"c1","function":{"name":"f","arguments":"{}"}}]}}]}`))
	if err != nil || res.FinishReason != llmprovider.FinishToolCalls {
		t.Errorf("stop with a call: %v, %v; want tool_calls", res, err)
	}
	res, err = Decode(strings.NewReader(`{"choices":[{"finish_reason":"stop","message":{"role":"Assistant","content":"hi"}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if msg, ok := res.Output[0].(llmprovider.MessageItem); !ok || msg.Role != llmprovider.RoleAssistant {
		t.Errorf("role Assistant: %+v; want RoleAssistant", res.Output)
	}
	var apiErr *llmprovider.APIError
	_, err = Decode(strings.NewReader(`{"choices":[{"finish_reason":"content_filter","message":{"role":"assistant"}}]}`))
	if !errors.As(err, &apiErr) || !errors.Is(err, llmprovider.ErrIncomplete) || apiErr.Reason != "content_filter" {
		t.Errorf("an empty content_filter answer: %v; want ErrIncomplete, reason content_filter", err)
	}
}

// TestDecode_GatewayErrorIn200 (0021-MADR W5): an error envelope in a 200, at
// the top level or in the choice, is classified with its message, not
// reported as an empty answer.
func TestDecode_GatewayErrorIn200(t *testing.T) {
	for name, body := range map[string]string{
		"top level": `{"error":{"message":"Upstream overloaded","code":502}}`,
		"in the choice": `{"choices":[{"finish_reason":"error","error":{"message":"Upstream overloaded","code":"502"},` +
			`"message":{"role":"assistant","content":""}}]}`,
	} {
		_, err := Decode(strings.NewReader(body))
		if !errors.Is(err, llmprovider.ErrProviderUnavailable) || err == nil || !strings.Contains(err.Error(), "Upstream overloaded") {
			t.Errorf("%s: %v; want ErrProviderUnavailable with the message", name, err)
		}
	}
	if _, err := Decode(strings.NewReader(`{"choices":[{"finish_reason":"error","message":{"role":"assistant"}}]}`)); !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Errorf("finish_reason error with no envelope: %v; want ErrProviderUnavailable", err)
	}
}
