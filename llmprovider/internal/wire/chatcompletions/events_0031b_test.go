package chatcompletions

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// chunks frames each payload as one event, then [DONE], as the gateways send
// a stream.
func chunks(payloads ...string) string {
	var b strings.Builder
	for _, p := range payloads {
		b.WriteString("data: " + p + "\n\n")
	}
	return b.String() + "data: [DONE]\n\n"
}

// d is a chunk with one choice's delta, and a finish reason when finish is
// set.
func d(delta, finish string) string {
	f := "null"
	if finish != "" {
		f = `"` + finish + `"`
	}
	return `{"id":"c1","model":"m","object":"chat.completion.chunk","choices":[{"index":0,"delta":` + delta + `,"finish_reason":` + f + `}]}`
}

const usageChunk = `{"id":"c1","model":"m","choices":[],"usage":{"prompt_tokens":9,"completion_tokens":4,"prompt_tokens_details":{"cached_tokens":2}}}`

// collect runs Events over stream.
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

// TestEvents_EqualsDecode (0031-MADR E3; rule 11): each stream's events are
// pinned, and its EventDone's Response is Decode's on the same answer as one
// JSON reply, or both fail alike; the text deltas joined are the message
// text.
func TestEvents_EqualsDecode(t *testing.T) {
	usage := `"usage":{"prompt_tokens":9,"completion_tokens":4,"prompt_tokens_details":{"cached_tokens":2}}`
	cases := []struct {
		name   string
		stream string
		reply  string
		want   []string
	}{
		{"text", chunks(d(`{"role":"assistant","content":""}`, ""), d(`{"content":"hel"}`, ""), d(`{"content":"lo"}`, ""),
			d(`{}`, "stop"), usageChunk),
			`{"id":"c1","model":"m","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"hello"}}],` + usage + `}`,
			[]string{"text_delta:hel", "text_delta:lo", "item:MessageItem", "done"}},
		{"reasoning_content", chunks(d(`{"role":"assistant","reasoning_content":"thi"}`, ""), d(`{"reasoning_content":"nk"}`, ""),
			d(`{"content":"ok"}`, ""), d(`{}`, "stop"), usageChunk),
			`{"id":"c1","model":"m","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"ok","reasoning_content":"think"}}],` + usage + `}`,
			[]string{"reasoning_delta:thi", "reasoning_delta:nk", "text_delta:ok", "item:ReasoningItem", "item:MessageItem", "done"}},
		{"reasoning with details (Kilo)", chunks(
			d(`{"role":"assistant","reasoning":"thi","reasoning_details":[{"type":"reasoning.text","text":"thi","format":"unknown","index":0}]}`, ""),
			d(`{"reasoning":"nk","reasoning_details":[{"type":"reasoning.text","text":"nk","format":"unknown","index":0}]}`, ""),
			d(`{"content":"ok"}`, ""), d(`{}`, "stop"), usageChunk),
			`{"id":"c1","model":"m","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"ok","reasoning":"think",` +
				`"reasoning_details":[{"format":"unknown","index":0,"text":"think","type":"reasoning.text"}]}}],` + usage + `}`,
			[]string{"reasoning_delta:thi", "reasoning_delta:nk", "text_delta:ok", "item:ReasoningItem", "item:MessageItem", "done"}},
		{"calls", chunks(
			d(`{"role":"assistant","tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"record_city","arguments":""}}]}`, ""),
			d(`{"tool_calls":[{"index":0,"function":{"arguments":"{\"city\":"}}]}`, ""),
			d(`{"tool_calls":[{"index":0,"function":{"arguments":"\"Paris\"}"}}]}`, ""),
			d(`{"tool_calls":[{"index":1,"id":"call_2","type":"function","function":{"name":"now","arguments":""}}]}`, ""),
			d(`{}`, "tool_calls"), usageChunk),
			`{"id":"c1","model":"m","choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","tool_calls":[` +
				`{"id":"call_1","function":{"name":"record_city","arguments":"{\"city\":\"Paris\"}"}},` +
				`{"id":"call_2","function":{"name":"now","arguments":""}}]}}],` + usage + `}`,
			[]string{"item:FunctionCallItem", "item:FunctionCallItem", "done"}},
		{"refusal", chunks(d(`{"role":"assistant","refusal":"No"}`, ""), d(`{"refusal":"."}`, ""), d(`{}`, "stop"), usageChunk),
			`{"id":"c1","model":"m","choices":[{"finish_reason":"stop","message":{"role":"assistant","refusal":"No."}}],` + usage + `}`,
			[]string{"text_delta:No", "text_delta:.", "item:MessageItem", "done"}},
		{"cut while reasoning", chunks(d(`{"role":"assistant","reasoning_content":"hmm"}`, ""), d(`{}`, "length"), usageChunk),
			`{"id":"c1","model":"m","choices":[{"finish_reason":"length","message":{"role":"assistant","reasoning_content":"hmm"}}],` + usage + `}`,
			[]string{"reasoning_delta:hmm"}},
		{"call cut by the limit", chunks(
			d(`{"role":"assistant","tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"record_city","arguments":"{\"ci"}}]}`, ""),
			d(`{}`, "length"), usageChunk),
			`{"id":"c1","model":"m","choices":[{"finish_reason":"length","message":{"role":"assistant","tool_calls":[` +
				`{"id":"call_1","function":{"name":"record_city","arguments":"{\"ci"}}]}}],` + usage + `}`,
			nil},
		{"no usage", chunks(d(`{"role":"assistant","content":"ok"}`, "stop")),
			`{"id":"c1","model":"m","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}]}`,
			[]string{"text_delta:ok", "item:MessageItem", "done"}},
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

// TestEvents_ErrorInsideTheStream (0021-MADR W5; 0031-MADR D4): a delta, then
// an error envelope inside the 200: the delta, then the error classified as
// Decode classifies it, and no EventDone.
func TestEvents_ErrorInsideTheStream(t *testing.T) {
	for name, failure := range map[string]string{
		"top level":     `{"error":{"message":"overloaded","code":503}}`,
		"in the choice": `{"choices":[{"index":0,"delta":{},"finish_reason":"error","error":{"message":"overloaded","code":503}}]}`,
	} {
		_, kinds, err := collect(t, chunks(d(`{"role":"assistant","content":"par"}`, ""), failure))
		if !reflect.DeepEqual(kinds, []string{"text_delta:par"}) || !errors.Is(err, llmprovider.ErrProviderUnavailable) {
			t.Errorf("%s: events %v, error %v; want the delta, then ErrProviderUnavailable", name, kinds, err)
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

// TestEvents_StopEarly: Events returns nil as soon as yield returns false.
func TestEvents_StopEarly(t *testing.T) {
	stream := strings.Repeat("data: "+d(`{"content":"`+strings.Repeat("x", 200)+`"}`, "")+"\n\n", 10000)
	body := &countingReader{r: strings.NewReader(stream)}
	calls := 0
	err := Events("p", body, func(llmprovider.Event) bool { calls++; return false })
	if err != nil || calls != 1 || body.n >= len(stream)/2 {
		t.Fatalf("stopped after %d calls, %v, having read %d of %d bytes", calls, err, body.n, len(stream))
	}
}

// TestEvents_EndsEarly: a stream with no [DONE] is retryable once.
func TestEvents_EndsEarly(t *testing.T) {
	_, _, err := collect(t, strings.TrimSuffix(chunks(d(`{"content":"x"}`, "stop")), "data: [DONE]\n\n"))
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) || !strings.Contains(err.Error(), "[DONE]") {
		t.Fatalf("an early end: %v; want ErrProviderUnavailable naming [DONE]", err)
	}
}
