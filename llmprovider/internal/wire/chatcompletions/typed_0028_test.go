package chatcompletions

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// 0028-MADR D-A6: the typed encoder sends the JSON the map encoder sent,
// byte for byte, on every case.

// messagesCase is one encoding: items, a replay field and the details flag.
type messagesCase struct {
	name    string
	items   []llmprovider.Item
	field   string
	details bool
}

// messagesCorpus is the wire cases' inputs under every replay setting, and
// 300 generated sequences (seed 28).
func messagesCorpus() []messagesCase {
	fields := []string{"", "reasoning_content", "reasoning_details", "reasoning",
		// Fields that are the message's own keys: never sent by a service's
		// metadata, kept to the map encoder's results all the same.
		"content", "role", "tool_call_id", "tool_calls"}
	var cases []messagesCase
	for _, field := range fields {
		for _, details := range []bool{false, true} {
			cases = append(cases, messagesCase{fmt.Sprintf("wirecase.Items/%q/%t", field, details), wirecase.Items(), field, details})
		}
	}
	r := rand.New(rand.NewPCG(28, 0))
	texts := []string{"", "hello", "It is <b>sunny</b> & warm", "quote \" and \\ slash", "日本語 ✓", "line\nbreak"}
	detail := []string{`{"type":"reasoning.text","text":"t","signature":"s"}`, ` {"format":"google-gemini-v1", "index":0} `,
		`{"type":`, `not json`, `[1,2]`, ""}
	for i := range 300 {
		var items []llmprovider.Item
		for range 1 + r.IntN(12) {
			text := texts[r.IntN(len(texts))]
			switch r.IntN(9) {
			case 0:
				items = append(items, llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: text})
			case 1:
				items = append(items, llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: text})
			case 2:
				items = append(items, llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: text})
			case 3:
				items = append(items, llmprovider.MessageItem{Text: text})
			case 4:
				items = append(items, llmprovider.FunctionCallItem{CallID: fmt.Sprintf("call_%d", r.IntN(4)),
					Name: "get_weather", Arguments: []string{`{"city":"Paris"}`, `{}`, ``, `not json`}[r.IntN(4)]})
			case 5:
				items = append(items, llmprovider.FunctionCallOutputItem{CallID: []string{"call_1", ""}[r.IntN(2)], Output: text})
			case 6:
				items = append(items, llmprovider.ReasoningItem{Text: text})
			case 7:
				items = append(items, llmprovider.ReasoningItem{Text: text, Encrypted: detail[r.IntN(len(detail))],
					Format: []string{wire.FormatChatCompletions, "responses"}[r.IntN(2)]})
			case 8:
				items = append(items, llmprovider.MessageItem{Role: "developer", Text: text})
			}
		}
		cases = append(cases, messagesCase{fmt.Sprintf("gen-%03d", i), items, fields[r.IntN(len(fields))], r.IntN(2) == 0})
	}
	return cases
}

// TestChatMessages_SameJSONAsReference: every case encodes to the reference's
// bytes.
func TestChatMessages_SameJSONAsReference(t *testing.T) {
	for _, c := range messagesCorpus() {
		got, err := json.Marshal(chatMessagesValue(itemsToChatMessagesReplaying(c.items, c.field, c.details)))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		want, err := json.Marshal(refItemsToChatMessagesReplaying(c.items, c.field, c.details))
		if err != nil {
			t.Fatalf("%s: reference: %v", c.name, err)
		}
		if string(got) != string(want) {
			t.Errorf("%s:\n  got  %s\n  want %s", c.name, got, want)
		}
	}
}

// TestToolList_SameJSONAsReference: tools, with and without a schema or a
// description, encode to the reference's bytes.
func TestToolList_SameJSONAsReference(t *testing.T) {
	weather := wirecase.WeatherTool()
	for name, tools := range map[string][]llmprovider.Tool{
		"none":           nil,
		"weather":        {weather},
		"nil schema":     {{Name: "noop"}},
		"raw schema":     {{Name: "raw", Description: "<raw> & schema", Schema: json.RawMessage(`{"type":"object"}`)}},
		"several":        {weather, {Name: "second", Description: "two"}},
		"empty schema":   {{Name: "empty", Schema: map[string]any{}}},
		"no description": {{Name: "plain", Schema: map[string]any{"type": "object"}}},
	} {
		got, err := json.Marshal(toolList(tools))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		want, err := json.Marshal(refToolList(tools))
		if err != nil {
			t.Fatalf("%s: reference: %v", name, err)
		}
		if string(got) != string(want) {
			t.Errorf("%s:\n  got  %s\n  want %s", name, got, want)
		}
	}
}
