package messages

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestDecode_StopReason (0020-MADR F3, F11): stop_reason becomes
// FinishReason, the model is reported, and a tool call cut by max_tokens is
// ErrIncomplete, as Chat Completions does (MADR 0012 §1.5).
func TestDecode_StopReason(t *testing.T) {
	for _, c := range []struct {
		stop   string
		blocks string
		want   llmprovider.FinishReason
	}{
		{"end_turn", `{"type":"text","text":"hi"}`, llmprovider.FinishStop},
		{"stop_sequence", `{"type":"text","text":"hi"}`, llmprovider.FinishStop},
		{"max_tokens", `{"type":"text","text":"partial"}`, llmprovider.FinishLength},
		{"tool_use", `{"type":"tool_use","id":"t1","name":"f","input":{}}`, llmprovider.FinishToolCalls},
		{"refusal", `{"type":"text","text":"no"}`, llmprovider.FinishContentFilter},
	} {
		res, err := Decode(strings.NewReader(`{"model":"claude-served","stop_reason":"` + c.stop + `","content":[` + c.blocks + `]}`))
		if err != nil || res.FinishReason != c.want || res.Model != "claude-served" {
			t.Errorf("%s: Decode = %+v, %v; want finish %q, model claude-served", c.stop, res, err, c.want)
		}
	}
	_, err := Decode(strings.NewReader(`{"stop_reason":"max_tokens","content":[{"type":"tool_use","id":"t1","name":"f","input":{"subject":"fix: tru"}}]}`))
	if !errors.Is(err, llmprovider.ErrIncomplete) {
		t.Errorf("a tool call cut by max_tokens = %v, want ErrIncomplete", err)
	}
}

// TestDecode_EmptyIsIncomplete (0020-MADR F9): an answer with nothing in it
// has a kind, never retried as a failure to reach the service.
func TestDecode_EmptyIsIncomplete(t *testing.T) {
	for _, body := range []string{`{"stop_reason":"end_turn","content":[]}`, `{"content":[{"type":"text","text":""}]}`} {
		if _, err := Decode(strings.NewReader(body)); !errors.Is(err, llmprovider.ErrIncomplete) {
			t.Errorf("Decode(%s) = %v, want ErrIncomplete", body, err)
		}
	}
}

// TestThinking_SignatureIsReplayed (0020-MADR F7): a thinking block keeps its
// signature, a redacted one its data, and both go back ahead of the tool
// call in the next request, as Anthropic requires when thinking and tools
// are combined.
func TestThinking_SignatureIsReplayed(t *testing.T) {
	res, err := Decode(strings.NewReader(`{"stop_reason":"tool_use","content":[
		{"type":"thinking","thinking":"plan","signature":"SIG123"},
		{"type":"redacted_thinking","data":"ENCRYPTED"},
		{"type":"tool_use","id":"t1","name":"f","input":{"a":1}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []llmprovider.Item{
		llmprovider.ReasoningItem{Text: "plan", Signature: "SIG123", Format: "messages"},
		llmprovider.ReasoningItem{Encrypted: "ENCRYPTED", Format: "messages"},
		llmprovider.FunctionCallItem{CallID: "t1", Name: "f", Arguments: `{"a":1}`},
	}
	if !reflect.DeepEqual(res.Output, want) {
		t.Fatalf("Output = %#v, want %#v", res.Output, want)
	}
	items := append([]llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}, res.Output...)
	items = append(items, llmprovider.FunctionCallOutputItem{CallID: "t1", Output: "done"})
	msgs := FromItems(items)
	if len(msgs) != 3 {
		t.Fatalf("messages = %v, want user, assistant, user", msgs)
	}
	// Typed blocks (0028-PLAN D14, Phase 9b; D16's union).
	list := msgs[1].Content.blocks
	if list == nil || len(*list) != 3 {
		t.Fatalf("assistant content = %#v, want three blocks", msgs[1].Content)
	}
	blocks := *list
	if blocks[0].Type != "thinking" || deref(blocks[0].Signature) != "SIG123" ||
		blocks[1].Type != "redacted_thinking" || deref(blocks[1].Data) != "ENCRYPTED" || blocks[2].Type != "tool_use" {
		t.Errorf("assistant content = %#v, want thinking (signed), redacted_thinking, then tool_use", blocks)
	}
	// Unsigned reasoning, such as another service's, cannot be replayed.
	if got := FromItems([]llmprovider.Item{llmprovider.ReasoningItem{Text: "unsigned"}}); len(got) != 0 {
		t.Errorf("FromItems(unsigned reasoning) = %v, want nothing", got)
	}
}

// deref is *p, or "" for nil.
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
