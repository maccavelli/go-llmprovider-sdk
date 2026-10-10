package responses

import (
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// collectEvents runs Events over stream and returns every event it yielded,
// and its error.
func collectEvents(t *testing.T, stream string) ([]llmprovider.Event, error) {
	t.Helper()
	var events []llmprovider.Event
	err := Events("p", strings.NewReader(stream), func(ev llmprovider.Event) bool {
		events = append(events, ev)
		return true
	})
	return events, err
}

// eventKinds is each event's type, with a delta's text, for comparison.
func eventKinds(events []llmprovider.Event) []string {
	kinds := make([]string, 0, len(events))
	for _, ev := range events {
		switch ev.Type {
		case llmprovider.EventTextDelta, llmprovider.EventReasoningDelta:
			kinds = append(kinds, string(ev.Type)+":"+ev.Text)
		case llmprovider.EventItem:
			kinds = append(kinds, string(ev.Type)+":"+reflect.TypeOf(ev.Item).Name())
		default:
			kinds = append(kinds, string(ev.Type))
		}
	}
	return kinds
}

// TestEvents_Goldens (0031-MADR D2): each ChatGPT golden's events are pinned;
// one EventDone comes last, its Response is ReadStream's, its items are the
// EventItems in order, and its message text is the text deltas joined.
func TestEvents_Goldens(t *testing.T) {
	for name, want := range map[string][]string{
		"chatgpt-text.sse":      {"text_delta:ok", "item:MessageItem", "done"},
		"chatgpt-reasoning.sse": {"text_delta:391", "item:MessageItem", "done"},
		"chatgpt-tool.sse":      {"item:FunctionCallItem", "done"},
	} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile("../../../providers/openai/testdata/" + name)
			if err != nil {
				t.Fatal(err)
			}
			events, err := collectEvents(t, string(raw))
			if err != nil {
				t.Fatal(err)
			}
			if got := eventKinds(events); !reflect.DeepEqual(got, want) {
				t.Fatalf("events %v, want %v", got, want)
			}
			done := events[len(events)-1].Response
			read, err := ReadStream("p", strings.NewReader(string(raw)))
			if err != nil || !reflect.DeepEqual(done, read) {
				t.Fatalf("EventDone's Response %+v, ReadStream's %+v (%v)", done, read, err)
			}
			var items []llmprovider.Item
			var deltas, texts strings.Builder
			for _, ev := range events {
				switch ev.Type {
				case llmprovider.EventItem:
					items = append(items, ev.Item)
				case llmprovider.EventTextDelta:
					deltas.WriteString(ev.Text)
				}
			}
			for _, item := range done.Output {
				if m, ok := item.(llmprovider.MessageItem); ok {
					texts.WriteString(m.Text)
				}
			}
			if !reflect.DeepEqual(items, done.Output) || deltas.String() != texts.String() {
				t.Fatalf("items %+v, deltas %q; want %+v and %q", items, deltas.String(), done.Output, texts.String())
			}
		})
	}
}

// TestEvents_ReasoningDelta (0031-PLAN Phase 1, rule 3): Grok's reasoning
// summary delta is EventReasoningDelta; the unmeasured reasoning text delta
// yields nothing.
func TestEvents_ReasoningDelta(t *testing.T) {
	events, err := collectEvents(t, sse(
		`{"type":"response.reasoning_summary_text.delta","delta":"think"}`,
		`{"type":"response.reasoning_text.delta","delta":"unmapped"}`,
		`{"type":"response.output_item.done","item":{"type":"reasoning","summary":[{"type":"summary_text","text":"think"}]}}`,
		`{"type":"response.output_text.delta","delta":"ok"}`,
		`{"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"ok"}]}}`,
		`{"type":"response.completed","response":{"id":"r1"}}`,
	))
	want := []string{"reasoning_delta:think", "item:ReasoningItem", "text_delta:ok", "item:MessageItem", "done"}
	if got := eventKinds(events); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("events %v (%v), want %v", got, err, want)
	}
}

// TestEvents_CallArgumentDeltasSilent (0031-MADR Q4): call-argument deltas
// yield nothing; the call arrives as one EventItem.
func TestEvents_CallArgumentDeltasSilent(t *testing.T) {
	raw, err := os.ReadFile("../../../providers/openai/testdata/chatgpt-tool.sse")
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(raw), `"type":"response.function_call_arguments.delta"`); n != 7 {
		t.Fatalf("the golden has %d argument deltas, want 7", n)
	}
	events, err := collectEvents(t, string(raw))
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range events {
		if ev.Type == llmprovider.EventTextDelta || ev.Type == llmprovider.EventReasoningDelta {
			t.Fatalf("an argument delta yielded %+v", ev)
		}
	}
}

// countingReader counts the bytes read through it.
type countingReader struct {
	r io.Reader
	n int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	return n, err
}

// TestEvents_StopEarly (0031-MADR D3): Events returns nil as soon as yield
// returns false, and reads no further.
func TestEvents_StopEarly(t *testing.T) {
	stream := longStream(10000)
	body := &countingReader{r: strings.NewReader(stream)}
	calls := 0
	err := Events("p", body, func(llmprovider.Event) bool {
		calls++
		return false
	})
	if err != nil || calls != 1 {
		t.Fatalf("stopped at once: %d calls, %v; want 1 call and nil", calls, err)
	}
	if body.n >= len(stream)/2 {
		t.Fatalf("read %d of %d bytes after the caller stopped", body.n, len(stream))
	}
}

// TestEvents_FailureAfterDelta (0031-MADR D4): a delta, then response.failed:
// the delta, then the classified error, and no EventDone.
func TestEvents_FailureAfterDelta(t *testing.T) {
	events, err := collectEvents(t, sse(
		`{"type":"response.output_text.delta","delta":"par"}`,
		`{"type":"response.failed","response":{"error":{"code":"rate_limit_exceeded","message":"slow down"}}}`,
	))
	if got, want := eventKinds(events), []string{"text_delta:par"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("events %v, want %v", got, want)
	}
	if !errors.Is(err, llmprovider.ErrRateLimited) {
		t.Fatalf("error %v, want ErrRateLimited", err)
	}
}
