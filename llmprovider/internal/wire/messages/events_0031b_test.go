package messages

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// sse frames each payload as one event, with its type on the event line, as
// the Messages API sends it.
func sse(payloads ...string) string {
	var b strings.Builder
	for _, p := range payloads {
		b.WriteString("event: x\ndata: " + p + "\n\n")
	}
	return b.String()
}

// collect runs Events over stream, returning each event's kind (a delta with
// its text, an item with its type) and the stream's error.
func collect(t *testing.T, stream string) ([]llmprovider.Event, []string, error) {
	t.Helper()
	var events []llmprovider.Event
	var kinds []string
	err := Events("p", strings.NewReader(stream), func(ev llmprovider.Event) bool {
		events = append(events, ev)
		switch ev.Type {
		case llmprovider.EventTextDelta, llmprovider.EventReasoningDelta:
			kinds = append(kinds, string(ev.Type)+":"+ev.Text)
		case llmprovider.EventItem:
			kinds = append(kinds, string(ev.Type)+":"+reflect.TypeOf(ev.Item).Name())
		default:
			kinds = append(kinds, string(ev.Type))
		}
		return true
	})
	return events, kinds, err
}

const (
	start = `{"type":"message_start","message":{"model":"m","usage":{"input_tokens":10,"cache_read_input_tokens":2,"output_tokens":1}}}`
	ping  = `{"type":"ping"}`
	stop  = `{"type":"message_stop"}`
)

func blockStart(index, block string) string {
	return `{"type":"content_block_start","index":` + index + `,"content_block":` + block + `}`
}

func delta(index, d string) string {
	return `{"type":"content_block_delta","index":` + index + `,"delta":` + d + `}`
}

func blockStop(index string) string { return `{"type":"content_block_stop","index":` + index + `}` }

func messageDelta(stopReason string) string {
	return `{"type":"message_delta","delta":{"stop_reason":"` + stopReason + `"},"usage":{"output_tokens":5}}`
}

// TestEvents_EqualsDecode (0031-MADR E3; rule 11): each stream's events are
// pinned, its EventDone's Response is Decode's on the same answer as one JSON
// reply, or both fail alike, and its text deltas joined are the message text.
func TestEvents_EqualsDecode(t *testing.T) {
	usage := `"usage":{"input_tokens":10,"cache_read_input_tokens":2,"output_tokens":5}`
	cases := []struct {
		name   string
		stream string
		reply  string
		want   []string
	}{
		{"text", sse(start, ping, blockStart("0", `{"type":"text","text":""}`),
			delta("0", `{"type":"text_delta","text":"hel"}`), delta("0", `{"type":"text_delta","text":"lo"}`), blockStop("0"),
			messageDelta("end_turn"), stop),
			`{"model":"m","stop_reason":"end_turn","content":[{"type":"text","text":"hello"}],` + usage + `}`,
			[]string{"text_delta:hel", "text_delta:lo", "item:MessageItem", "done"}},
		{"reasoning", sse(start, blockStart("0", `{"type":"thinking","thinking":"","signature":""}`),
			delta("0", `{"type":"thinking_delta","thinking":"think"}`), delta("0", `{"type":"signature_delta","signature":"sig"}`),
			blockStop("0"), blockStart("1", `{"type":"text","text":""}`), delta("1", `{"type":"text_delta","text":"ok"}`),
			blockStop("1"), messageDelta("end_turn"), stop),
			`{"model":"m","stop_reason":"end_turn","content":[{"type":"thinking","thinking":"think","signature":"sig"},{"type":"text","text":"ok"}],` + usage + `}`,
			[]string{"reasoning_delta:think", "item:ReasoningItem", "text_delta:ok", "item:MessageItem", "done"}},
		{"redacted thinking", sse(start, blockStart("0", `{"type":"redacted_thinking","data":"enc"}`), blockStop("0"),
			blockStart("1", `{"type":"text","text":""}`), delta("1", `{"type":"text_delta","text":"ok"}`), blockStop("1"),
			messageDelta("end_turn"), stop),
			`{"model":"m","stop_reason":"end_turn","content":[{"type":"redacted_thinking","data":"enc"},{"type":"text","text":"ok"}],` + usage + `}`,
			[]string{"item:ReasoningItem", "text_delta:ok", "item:MessageItem", "done"}},
		{"call", sse(start, blockStart("0", `{"type":"tool_use","id":"toolu_1","name":"record_city","input":{}}`),
			delta("0", `{"type":"input_json_delta","partial_json":"{\"city\":"}`), delta("0", `{"type":"input_json_delta","partial_json":" \"Paris\"}"}`),
			blockStop("0"), messageDelta("tool_use"), stop),
			`{"model":"m","stop_reason":"tool_use","content":[{"type":"tool_use","id":"toolu_1","name":"record_city","input":{"city": "Paris"}}],` + usage + `}`,
			[]string{"item:FunctionCallItem", "done"}},
		{"call with no arguments", sse(start, blockStart("0", `{"type":"tool_use","id":"toolu_1","name":"now","input":{}}`),
			blockStop("0"), messageDelta("tool_use"), stop),
			`{"model":"m","stop_reason":"tool_use","content":[{"type":"tool_use","id":"toolu_1","name":"now","input":{}}],` + usage + `}`,
			[]string{"item:FunctionCallItem", "done"}},
		{"refusal", sse(start, blockStart("0", `{"type":"text","text":""}`), delta("0", `{"type":"text_delta","text":"No."}`),
			blockStop("0"), messageDelta("refusal"), stop),
			`{"model":"m","stop_reason":"refusal","content":[{"type":"text","text":"No."}],` + usage + `}`,
			[]string{"text_delta:No.", "item:MessageItem", "done"}},
		{"cut while thinking", sse(start, blockStart("0", `{"type":"thinking","thinking":"","signature":""}`),
			delta("0", `{"type":"thinking_delta","thinking":"hmm"}`), blockStop("0"), messageDelta("max_tokens"), stop),
			`{"model":"m","stop_reason":"max_tokens","content":[{"type":"thinking","thinking":"hmm","signature":""}],` + usage + `}`,
			[]string{"reasoning_delta:hmm", "item:ReasoningItem"}},
		{"call cut by the limit", sse(start, blockStart("0", `{"type":"tool_use","id":"toolu_1","name":"record_city","input":{}}`),
			delta("0", `{"type":"input_json_delta","partial_json":"{\"city\":"}`), blockStop("0"), messageDelta("max_tokens"), stop),
			`{"model":"m","stop_reason":"max_tokens","content":[{"type":"tool_use","id":"toolu_1","name":"record_city","input":{}}],` + usage + `}`,
			nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events, kinds, err := collect(t, tc.stream)
			want, wantErr := Decode(strings.NewReader(tc.reply))
			if (err == nil) != (wantErr == nil) || err != nil && err.Error() != wantErr.Error() {
				t.Fatalf("stream error %v; Decode's %v", err, wantErr)
			}
			if tc.want != nil && !reflect.DeepEqual(kinds, tc.want) {
				t.Fatalf("events %v, want %v", kinds, tc.want)
			}
			if err != nil {
				for _, ev := range events {
					if ev.Type == llmprovider.EventDone {
						t.Fatal("EventDone on a failed answer")
					}
				}
				return
			}
			done := events[len(events)-1].Response
			if !reflect.DeepEqual(done, want) {
				t.Fatalf("EventDone's Response %+v; Decode's %+v", done, want)
			}
			var deltas strings.Builder
			for _, ev := range events {
				if ev.Type == llmprovider.EventTextDelta {
					deltas.WriteString(ev.Text)
				}
			}
			if deltas.String() != done.OutputText() {
				t.Fatalf("text deltas %q; message text %q", deltas.String(), done.OutputText())
			}
		})
	}
}

// TestEvents_FailureAfterDelta (0031-MADR D4, E3): a delta, then the stream's
// error event: the delta, then the error classified by its type, and no
// EventDone.
func TestEvents_FailureAfterDelta(t *testing.T) {
	_, kinds, err := collect(t, sse(start, blockStart("0", `{"type":"text","text":""}`),
		delta("0", `{"type":"text_delta","text":"par"}`),
		`{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`))
	if !reflect.DeepEqual(kinds, []string{"text_delta:par"}) {
		t.Fatalf("events %v, want the one delta", kinds)
	}
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Fatalf("error %v, want ErrProviderUnavailable", err)
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

// TestEvents_StopEarly: Events returns nil as soon as yield returns false.
func TestEvents_StopEarly(t *testing.T) {
	stream := sse(start, blockStart("0", `{"type":"text","text":""}`)) +
		strings.Repeat(sse(delta("0", `{"type":"text_delta","text":"`+strings.Repeat("x", 200)+`"}`)), 10000)
	body := &countingReader{r: strings.NewReader(stream)}
	calls := 0
	err := Events("p", body, func(llmprovider.Event) bool { calls++; return false })
	if err != nil || calls != 1 || body.n >= len(stream)/2 {
		t.Fatalf("stopped after %d calls, %v, having read %d of %d bytes", calls, err, body.n, len(stream))
	}
}

// TestEvents_EndsEarly: a stream with no message_stop is retryable once.
func TestEvents_EndsEarly(t *testing.T) {
	_, _, err := collect(t, sse(start, blockStart("0", `{"type":"text","text":""}`), delta("0", `{"type":"text_delta","text":"x"}`)))
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) || !strings.Contains(err.Error(), "message_stop") {
		t.Fatalf("an early end: %v; want ErrProviderUnavailable naming message_stop", err)
	}
}
