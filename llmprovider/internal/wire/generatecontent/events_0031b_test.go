package generatecontent

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// sse frames each payload as one event, as Google's alt=sse stream sends it:
// no event line, and no [DONE].
func sse(payloads ...string) string {
	var b strings.Builder
	for _, p := range payloads {
		b.WriteString("data: " + p + "\n\n")
	}
	return b.String()
}

// chunk is one streamed GenerateContentResponse with parts, and a finish
// reason and usage when finish is set.
func chunk(parts, finish string) string {
	tail := ""
	if finish != "" {
		tail = `,"finishReason":"` + finish + `"`
	}
	usage := ""
	if finish != "" {
		usage = `,"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":4,"thoughtsTokenCount":2}`
	}
	return `{"candidates":[{"content":{"role":"model","parts":` + parts + `}` + tail + `,"index":0}],"modelVersion":"g"` + usage + `}`
}

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
// JSON reply, where consecutive text parts are one part, or both fail alike.
func TestEvents_EqualsDecode(t *testing.T) {
	usage := `,"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":4,"thoughtsTokenCount":2}`
	cases := []struct {
		name   string
		stream string
		reply  string
		want   []string
	}{
		{"text", sse(chunk(`[{"text":"hel"}]`, ""), chunk(`[{"text":"lo","thoughtSignature":"sig"}]`, "STOP")),
			`{"candidates":[{"content":{"parts":[{"text":"hello","thoughtSignature":"sig"}]},"finishReason":"STOP"}],"modelVersion":"g"` + usage + `}`,
			[]string{"text_delta:hel", "text_delta:lo", "item:MessageItem", "done"}},
		{"thought, then text", sse(chunk(`[{"text":"thi","thought":true}]`, ""), chunk(`[{"text":"nk","thought":true}]`, ""),
			chunk(`[{"text":"ok"}]`, "STOP")),
			`{"candidates":[{"content":{"parts":[{"text":"think","thought":true},{"text":"ok"}]},"finishReason":"STOP"}],"modelVersion":"g"` + usage + `}`,
			[]string{"text_delta:ok", "item:ReasoningItem", "item:MessageItem", "done"}},
		{"calls", sse(chunk(`[{"functionCall":{"id":"f1","name":"record_city","args":{"city":"Paris"}},"thoughtSignature":"sig"}]`, ""),
			chunk(`[{"functionCall":{"name":"now"}}]`, "STOP")),
			`{"candidates":[{"content":{"parts":[{"functionCall":{"id":"f1","name":"record_city","args":{"city":"Paris"}},"thoughtSignature":"sig"},` +
				`{"functionCall":{"name":"now"}}]},"finishReason":"STOP"}],"modelVersion":"g"` + usage + `}`,
			[]string{"item:FunctionCallItem", "item:FunctionCallItem", "done"}},
		{"filtered", sse(chunk(`[{"text":"no"}]`, "SAFETY")),
			`{"candidates":[{"content":{"parts":[{"text":"no"}]},"finishReason":"SAFETY"}],"modelVersion":"g"` + usage + `}`,
			[]string{"text_delta:no", "item:MessageItem", "done"}},
		{"cut while thinking", sse(chunk(`[{"text":"hmm","thought":true}]`, "MAX_TOKENS")),
			`{"candidates":[{"content":{"parts":[{"text":"hmm","thought":true}]},"finishReason":"MAX_TOKENS"}],"modelVersion":"g"` + usage + `}`,
			nil},
		{"call cut by the limit", sse(chunk(`[{"functionCall":{"name":"record_city","args":{}}}]`, "MAX_TOKENS")),
			`{"candidates":[{"content":{"parts":[{"functionCall":{"name":"record_city","args":{}}}]},"finishReason":"MAX_TOKENS"}],"modelVersion":"g"` + usage + `}`,
			nil},
		{"blocked prompt", sse(`{"promptFeedback":{"blockReason":"SAFETY"},"modelVersion":"g"}`),
			`{"promptFeedback":{"blockReason":"SAFETY"},"modelVersion":"g"}`,
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

// TestEvents_ErrorInsideTheStream (0031-MADR D4, E3): a delta, then Google's
// error body in the stream: the delta, then the error classified by the
// status its code names, and no EventDone.
func TestEvents_ErrorInsideTheStream(t *testing.T) {
	_, kinds, err := collect(t, sse(chunk(`[{"text":"par"}]`, ""),
		`{"error":{"code":503,"message":"The model is overloaded.","status":"UNAVAILABLE"}}`))
	if !reflect.DeepEqual(kinds, []string{"text_delta:par"}) || !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Fatalf("events %v, error %v; want the delta, then ErrProviderUnavailable", kinds, err)
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
	stream := strings.Repeat(sse(chunk(`[{"text":"`+strings.Repeat("x", 200)+`"}]`, "")), 10000)
	body := &countingReader{r: strings.NewReader(stream)}
	calls := 0
	err := Events("p", body, func(llmprovider.Event) bool { calls++; return false })
	if err != nil || calls != 1 || body.n >= len(stream)/2 {
		t.Fatalf("stopped after %d calls, %v, having read %d of %d bytes", calls, err, body.n, len(stream))
	}
}

// TestEvents_EndsEarly: a stream with no finishReason is retryable once.
func TestEvents_EndsEarly(t *testing.T) {
	_, _, err := collect(t, sse(chunk(`[{"text":"x"}]`, "")))
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) || !strings.Contains(err.Error(), "finishReason") {
		t.Fatalf("an early end: %v; want ErrProviderUnavailable naming finishReason", err)
	}
}
