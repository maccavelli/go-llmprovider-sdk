package gemini

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestStreamIndex_0053 (0053-MADR D2): a step index outside 0 to 4095, on a
// start or a delta, ends the stream with ErrIncomplete naming the field,
// without a panic; 4095 is accepted.
func TestStreamIndex_0053(t *testing.T) {
	stream := func(event, index string) string {
		ev := map[string]string{
			"step.start": `{"event_type":"step.start","index":` + index + `,"step":{"type":"model_output"}}`,
			"step.delta": `{"event_type":"step.delta","index":` + index + `,"delta":{"type":"text","text":"x"}}`,
		}[event]
		return "event: interaction.created\ndata: {\"event_type\":\"interaction.created\",\"interaction\":{\"id\":\"v1\",\"model\":\"m\",\"status\":\"in_progress\"}}\n\n" +
			"event: " + event + "\ndata: " + ev + "\n\n"
	}
	run := func(s string) (panicked any, err error) {
		defer func() { panicked = recover() }()
		return nil, interactionEvents("p", strings.NewReader(s), func(llmprovider.Event) bool { return true })
	}
	for _, event := range []string{"step.start", "step.delta"} {
		for _, index := range []string{"-1", "4096", "100000"} {
			panicked, err := run(stream(event, index))
			if panicked != nil || !errors.Is(err, llmprovider.ErrIncomplete) || !strings.Contains(fmt.Sprint(err), "step") {
				t.Errorf("%s index %s: err %v, panic %v; want ErrIncomplete naming step, no panic", event, index, err, panicked)
			}
		}
		// 4095 passes the bound; this script ends without completing, so any
		// error is that, not the bound's.
		if panicked, err := run(stream(event, "4095")); panicked != nil || strings.Contains(fmt.Sprint(err), "out of range") {
			t.Errorf("%s index 4095: err %v, panic %v; want it past the bound", event, err, panicked)
		}
	}
}

// TestInteractionSignature_0053 (0053-MADR D1): a thought signature that
// arrives in deltas is joined after the one the step started with, and goes
// to the call that follows, as before.
func TestInteractionSignature_0053(t *testing.T) {
	stream := iev("interaction.created", created) +
		iev("step.start", `,"index":0,"step":{"type":"thought","signature":"a"}`) +
		iev("step.delta", `,"index":0,"delta":{"type":"thought_signature","signature":"b"}`) +
		iev("step.delta", `,"index":0,"delta":{"type":"thought_signature","signature":"c"}`) +
		iev("step.stop", `,"index":0`) +
		iev("step.start", `,"index":1,"step":{"type":"function_call","id":"fc1","name":"record_city","arguments":{}}`) +
		iev("step.delta", `,"index":1,"delta":{"type":"arguments_delta","arguments":"{\"city\":\"Paris\"}"}`) +
		iev("step.stop", `,"index":1`) + completed("requires_action") + doneLine
	events, _, err := collectInteraction(t, stream)
	if err != nil {
		t.Fatal(err)
	}
	var sig string
	for _, item := range events[len(events)-1].Response.Output {
		if c, ok := item.(llmprovider.FunctionCallItem); ok {
			sig = c.Signature
		}
	}
	if sig != "abc" {
		t.Errorf("the call's signature %q; want %q, the thought's start then its deltas", sig, "abc")
	}
}
