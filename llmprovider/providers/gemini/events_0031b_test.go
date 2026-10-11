package gemini

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// iev frames one Interactions stream event as the API sends it: its type on
// the event line and in event_type (0031-PLAN-native-streaming-on-the-other-
// wires, Phase 1).
func iev(eventType, fields string) string {
	return "event: " + eventType + "\ndata: {\"event_type\":\"" + eventType + "\"" + fields + "}\n\n"
}

const (
	created  = `,"interaction":{"id":"int_1","model":"gemini-3.7-flash","object":"interaction","status":"in_progress"}`
	doneLine = "event: done\ndata: [DONE]\n\n"
	usage    = `"usage":{"total_input_tokens":9,"total_output_tokens":3,"total_thought_tokens":5}`
)

func completed(status string) string {
	return iev("interaction.completed", `,"interaction":{"id":"int_1","model":"gemini-3.7-flash","status":"`+status+`",`+usage+`}`)
}

func collectInteraction(t *testing.T, stream string) ([]llmprovider.Event, []string, error) {
	t.Helper()
	var events []llmprovider.Event
	var kinds []string
	err := interactionEvents("p", strings.NewReader(stream), func(ev llmprovider.Event) bool {
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

// TestInteractionEvents_EqualsDecode (0031-MADR E3; rule 11): each stream's
// events are pinned, and its EventDone's Response is decodeInteraction's on
// the same interaction as one reply, or both fail alike.
func TestInteractionEvents_EqualsDecode(t *testing.T) {
	thought := iev("step.start", `,"index":0,"step":{"type":"thought"}`) +
		iev("step.delta", `,"index":0,"delta":{"type":"thought_signature","signature":"sig"}`) +
		iev("step.stop", `,"index":0`)
	cases := []struct {
		name   string
		stream string
		reply  string
		want   []string
	}{
		{"text", iev("interaction.created", created) + thought +
			iev("step.start", `,"index":1,"step":{"type":"model_output"}`) +
			iev("step.delta", `,"index":1,"delta":{"type":"text","text":"hel"}`) +
			iev("step.delta", `,"index":1,"delta":{"type":"text","text":"lo"}`) +
			iev("step.stop", `,"index":1`) + completed("completed") + doneLine,
			`{"id":"int_1","model":"gemini-3.7-flash","status":"completed","steps":[{"type":"thought","signature":"sig"},` +
				`{"type":"model_output","content":[{"type":"text","text":"hello"}]}],` + usage + `}`,
			[]string{"text_delta:hel", "text_delta:lo", "item:MessageItem", "done"}},
		{"call, arguments in deltas", iev("interaction.created", created) + thought +
			iev("step.start", `,"index":1,"step":{"type":"function_call","id":"fc1","name":"record_city","signature":"own","arguments":{}}`) +
			iev("step.delta", `,"index":1,"delta":{"type":"arguments_delta","arguments":"{\"city\":"}`) +
			iev("step.delta", `,"index":1,"delta":{"type":"arguments_delta","arguments":" \"Paris\"}"}`) +
			iev("step.stop", `,"index":1`) + completed("requires_action") + doneLine,
			`{"id":"int_1","model":"gemini-3.7-flash","status":"requires_action","steps":[{"type":"thought","signature":"sig"},` +
				`{"type":"function_call","id":"fc1","name":"record_city","arguments":{"city": "Paris"}}],` + usage + `}`,
			[]string{"item:FunctionCallItem", "done"}},
		{"call, arguments whole", iev("interaction.created", created) + thought +
			iev("step.start", `,"index":1,"step":{"type":"function_call","id":"fc1","name":"record_city","arguments":{"city":"Paris"}}`) +
			iev("step.delta", `,"index":1,"delta":{"type":"arguments_delta","arguments":{"city":"Paris"}}`) +
			iev("step.stop", `,"index":1`) + completed("requires_action") + doneLine,
			`{"id":"int_1","model":"gemini-3.7-flash","status":"requires_action","steps":[{"type":"thought","signature":"sig"},` +
				`{"type":"function_call","id":"fc1","name":"record_city","arguments":{"city":"Paris"}}],` + usage + `}`,
			[]string{"item:FunctionCallItem", "done"}},
		{"incomplete", iev("interaction.created", created) + thought + completed("incomplete") + doneLine,
			`{"id":"int_1","model":"gemini-3.7-flash","status":"incomplete","steps":[{"type":"thought","signature":"sig"}],` + usage + `}`,
			nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events, kinds, err := collectInteraction(t, tc.stream)
			want, wantErr := decodeInteraction(strings.NewReader(tc.reply))
			if (err == nil) != (wantErr == nil) || err != nil && err.Error() != wantErr.Error() {
				t.Fatalf("stream error %v; decodeInteraction's %v", err, wantErr)
			}
			if tc.want != nil && !reflect.DeepEqual(kinds, tc.want) {
				t.Fatalf("events %v, want %v", kinds, tc.want)
			}
			if err != nil {
				for _, ev := range events {
					if ev.Type == llmprovider.EventDone {
						t.Fatal("EventDone on a failed interaction")
					}
				}
				return
			}
			done := events[len(events)-1].Response
			if !reflect.DeepEqual(done, want) {
				t.Fatalf("EventDone's Response %+v; decodeInteraction's %+v", done, want)
			}
		})
	}
}

// TestInteractionEvents_ErrorInsideTheStream (0031-MADR D4, E3): a delta,
// then an error event: the delta, then the error classified by the status
// its code names, and no EventDone.
func TestInteractionEvents_ErrorInsideTheStream(t *testing.T) {
	_, kinds, err := collectInteraction(t, iev("interaction.created", created)+
		iev("step.start", `,"index":0,"step":{"type":"model_output"}`)+
		iev("step.delta", `,"index":0,"delta":{"type":"text","text":"par"}`)+
		iev("error", `,"error":{"code":503,"message":"The model is overloaded.","status":"UNAVAILABLE"}`))
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

// TestInteractionEvents_StopEarly: the decoder returns as soon as yield
// returns false.
func TestInteractionEvents_StopEarly(t *testing.T) {
	stream := iev("step.start", `,"index":0,"step":{"type":"model_output"}`) +
		strings.Repeat(iev("step.delta", `,"index":0,"delta":{"type":"text","text":"`+strings.Repeat("x", 200)+`"}`), 10000)
	body := &countingReader{r: strings.NewReader(stream)}
	calls := 0
	err := interactionEvents("p", body, func(llmprovider.Event) bool { calls++; return false })
	if err != nil || calls != 1 || body.n >= len(stream)/2 {
		t.Fatalf("stopped after %d calls, %v, having read %d of %d bytes", calls, err, body.n, len(stream))
	}
}

// TestInteractionEvents_EndsEarly: a stream with no interaction.completed is
// retryable once.
func TestInteractionEvents_EndsEarly(t *testing.T) {
	_, _, err := collectInteraction(t, iev("interaction.created", created)+
		iev("step.start", `,"index":0,"step":{"type":"model_output"}`)+
		iev("step.delta", `,"index":0,"delta":{"type":"text","text":"x"}`))
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) || !strings.Contains(err.Error(), "interaction.completed") {
		t.Fatalf("an early end: %v; want ErrProviderUnavailable naming interaction.completed", err)
	}
}
