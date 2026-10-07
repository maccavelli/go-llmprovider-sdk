package chatcompletions

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// TestDecode_ReasoningOnlyCutIsIncomplete (0026-MADR F7): an answer the
// token limit cut while it was still reasoning has nothing usable, so it is
// ErrIncomplete with Reason "length", as on the Responses wire. A cut text
// answer keeps its text (TestDecodeChat_LengthTextKeepsText).
func TestDecode_ReasoningOnlyCutIsIncomplete(t *testing.T) {
	body := `{"choices":[{"finish_reason":"length","message":{"role":"assistant","content":"","reasoning_content":"thinking about it"}}]}`
	res, err := Decode(strings.NewReader(body))
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	if !ok || !errors.Is(err, llmprovider.ErrIncomplete) || apiErr.Reason != string(llmprovider.FinishLength) {
		t.Fatalf("a reasoning-only cut answer: %+v, %v; want ErrIncomplete with Reason length", res, err)
	}
}

// TestChatCompletions_Refusal (0026-MADR F29): message.refusal is the answer's
// text, with FinishContentFilter, as the Responses wire keeps a refusal
// (0021-MADR W4).
func TestChatCompletions_Refusal(t *testing.T) {
	body := `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":null,"refusal":"I can't help with that."}}]}`
	res, err := Decode(strings.NewReader(body))
	if err != nil || res.OutputText() != "I can't help with that." || res.FinishReason != llmprovider.FinishContentFilter {
		t.Fatalf("a refusal: %+v, %v; want its text and FinishContentFilter", res, err)
	}
}

// TestChatCompletions_ContentParts (0026-MADR F30): content may be an array
// of parts; its text parts are the answer's text, joined.
func TestChatCompletions_ContentParts(t *testing.T) {
	body := `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":[` +
		`{"type":"text","text":"Hello, "},{"type":"image_url","image_url":{"url":"x"}},{"type":"text","text":"world"}]}}]}`
	res, err := Decode(strings.NewReader(body))
	if err != nil || res.OutputText() != "Hello, world" {
		t.Fatalf("content parts: %+v, %v; want \"Hello, world\"", res, err)
	}
	if _, err := Decode(strings.NewReader(`{"choices":[{"message":{"content":42}}]}`)); err == nil {
		t.Fatal("content 42: want a decode error")
	}
}

// TestChatCompletions_ReasoningNotCarriedAcrossTurns (0026-MADR F32):
// reasoning that no assistant message followed, such as an answer cut while
// reasoning, is not attached to a later turn's assistant message.
func TestChatCompletions_ReasoningNotCarriedAcrossTurns(t *testing.T) {
	detail := `{"type":"reasoning.text","text":"turn-1 reasoning","signature":"sig-1"}`
	msgs := itemsToChatMessagesReplaying([]llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "turn 1"},
		llmprovider.ReasoningItem{Text: "turn-1 reasoning", Encrypted: detail, Format: wire.FormatChatCompletions},
		llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "turn 2"},
		llmprovider.ReasoningItem{Text: "turn-2 reasoning"},
		llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "answer 2"},
	}, "reasoning_content", true)
	last := msgs[len(msgs)-1]
	if got := last["reasoning_content"]; got != "turn-2 reasoning" {
		t.Errorf("turn 2's reasoning_content = %q; want only turn 2's reasoning", got)
	}
	if got, ok := last[keyReasoningDetail]; ok {
		t.Errorf("turn 2 carries reasoning_details %s; want turn 1's left behind", got)
	}
}
