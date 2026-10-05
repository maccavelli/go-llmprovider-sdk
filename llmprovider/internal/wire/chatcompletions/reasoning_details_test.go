package chatcompletions

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// detailReply is a Kilo reply shaped as the 2.8a probe measured on
// google/gemini-3.8-flash: one reasoning.text entry, with the plain string
// alongside.
const detailReply = `{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant",` +
	`"reasoning":"Check the weather.","reasoning_details":[{"type":"reasoning.text","format":"google-gemini-v1",` +
	`"index":0,"text":"Check the weather."}],"tool_calls":[{"id":"c1","function":{"name":"w","arguments":"{}"}}]}}]}`

// TestDecode_ReasoningDetailsKept (0021-MADR W9): each reasoning_details entry
// is one reasoning item, with the entry itself kept to replay; the plain
// string is not decoded as well.
func TestDecode_ReasoningDetailsKept(t *testing.T) {
	res, err := Decode(strings.NewReader(detailReply))
	if err != nil {
		t.Fatal(err)
	}
	var reasoning []llmprovider.ReasoningItem
	for _, it := range res.Output {
		if r, ok := it.(llmprovider.ReasoningItem); ok {
			reasoning = append(reasoning, r)
		}
	}
	want := `{"type":"reasoning.text","format":"google-gemini-v1","index":0,"text":"Check the weather."}`
	if len(reasoning) != 1 || reasoning[0].Text != "Check the weather." || reasoning[0].Encrypted != want ||
		reasoning[0].Format != "chatcompletions" {
		t.Errorf("reasoning = %+v; want one item, the entry kept", reasoning)
	}
	if _, err := Decode(strings.NewReader(`{"choices":[{"message":{"content":"x","reasoning_details":[1]}}]}`)); err == nil {
		t.Error("an entry that is not an object decoded")
	}
}

// TestBody_ReplaysReasoningDetails (0021-MADR W9): with the option, the
// entries go back on the assistant message they precede, as they came; without
// it, and for another wire's reasoning, they do not.
func TestBody_ReplaysReasoningDetails(t *testing.T) {
	res, err := Decode(strings.NewReader(detailReply))
	if err != nil {
		t.Fatal(err)
	}
	input := append([]llmprovider.Item{llmprovider.MessageItem{Role: "user", Text: "umbrella?"}}, res.Output...)
	input = append(input, llmprovider.FunctionCallOutputItem{CallID: "c1", Output: "rain"},
		llmprovider.ReasoningItem{Encrypted: `{"type":"x"}`, Format: "responses"},
		llmprovider.MessageItem{Role: "assistant", Text: "take one"})
	encode := func(details bool) []map[string]any {
		raw, err := json.Marshal(Body("m", 10, input, Opts{ReplayReasoningDetails: details})[keyMessages])
		if err != nil {
			t.Fatal(err)
		}
		var messages []map[string]any
		if err := json.Unmarshal(raw, &messages); err != nil {
			t.Fatal(err)
		}
		return messages
	}
	messages := encode(true)
	details, _ := messages[1]["reasoning_details"].([]any)
	if len(details) != 1 || details[0].(map[string]any)["format"] != "google-gemini-v1" {
		t.Errorf("assistant message = %v; want the entry replayed", messages[1])
	}
	if _, ok := messages[3]["reasoning_details"]; ok {
		t.Errorf("another wire's reasoning was replayed: %v", messages[3])
	}
	for _, m := range encode(false) {
		if _, ok := m["reasoning_details"]; ok {
			t.Errorf("replayed without the option: %v", m)
		}
	}
}
