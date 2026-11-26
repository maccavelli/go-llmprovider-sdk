package responses

import (
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// The shapes of a reasoning item that carries its text in content as
// reasoning_text, with an empty summary, as Kilo, Together and Hugging Face
// sent it on 2026-11-23 (0091-MADR PR1, PR3, PR4 P6), and an OpenAI-shaped
// item whose summary holds the text.
const (
	textReasoning0091    = `{"type":"reasoning","id":"rs_1","summary":[],"content":[{"type":"reasoning_text","text":"thinking aloud"}]}`
	summaryReasoning0091 = `{"type":"reasoning","id":"rs_2","summary":[{"type":"summary_text","text":"from the summary"}],` +
		`"content":[{"type":"reasoning_text","text":"from the content"}]}`
	message0091 = `{"type":"message","id":"msg_1","content":[{"type":"output_text","text":"ok"}]}`
)

// reasoningOf is the reasoning items of output.
func reasoningOf(output []llmprovider.Item) []llmprovider.ReasoningItem {
	var out []llmprovider.ReasoningItem
	for _, item := range output {
		if r, ok := item.(llmprovider.ReasoningItem); ok {
			out = append(out, r)
		}
	}
	return out
}

// TestResponses_ReasoningText_0091 (0091-MADR ST10, ST63): a reasoning item
// whose summary holds no text is read from its reasoning_text content, and
// its stream's reasoning_text delta is EventReasoningDelta; an OpenAI-shaped
// item whose summary holds text decodes from the summary, unchanged. Neither
// carries encrypted content.
func TestResponses_ReasoningText_0091(t *testing.T) {
	want := []llmprovider.ReasoningItem{
		{Text: "thinking aloud", Format: wire.FormatResponses},
		{Text: "from the summary", Format: wire.FormatResponses},
	}
	t.Run("plain", func(t *testing.T) {
		resp, err := Decode(strings.NewReader(`{"id":"r1","status":"completed","output":[` +
			textReasoning0091 + `,` + summaryReasoning0091 + `,` + message0091 + `]}`))
		if err != nil {
			t.Fatal(err)
		}
		if got := reasoningOf(resp.Output); !reflect.DeepEqual(got, want) {
			t.Errorf("reasoning %+v; want %+v", got, want)
		}
	})
	t.Run("stream", func(t *testing.T) {
		events, err := collectEvents(t, sse(
			`{"type":"response.reasoning_text.delta","delta":"thinking "}`,
			`{"type":"response.reasoning_text.delta","delta":"aloud"}`,
			`{"type":"response.output_item.done","item":`+textReasoning0091+`}`,
			`{"type":"response.reasoning_summary_text.delta","delta":"from the summary"}`,
			`{"type":"response.output_item.done","item":`+summaryReasoning0091+`}`,
			`{"type":"response.output_text.delta","delta":"ok"}`,
			`{"type":"response.output_item.done","item":`+message0091+`}`,
			`{"type":"response.completed","response":{"id":"r1"}}`,
		))
		if err != nil {
			t.Fatal(err)
		}
		kinds := []string{"reasoning_delta:thinking ", "reasoning_delta:aloud", "item:ReasoningItem",
			"reasoning_delta:from the summary", "item:ReasoningItem", "text_delta:ok", "item:MessageItem", "done"}
		if got := eventKinds(events); !reflect.DeepEqual(got, kinds) {
			t.Fatalf("events %v; want %v", got, kinds)
		}
		if got := reasoningOf(events[len(events)-1].Response.Output); !reflect.DeepEqual(got, want) {
			t.Errorf("reasoning %+v; want %+v", got, want)
		}
	})
}
