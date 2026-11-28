package messages

import (
	"reflect"
	"testing"
)

// TestEvents_UnstoppedBlock_0095 (0095-MADR FU01): a block the stream never
// stopped, which the answer keeps, is yielded as EventItem before EventDone,
// as a stopped block is; a call the limit cut, which the answer drops, is
// not.
func TestEvents_UnstoppedBlock_0095(t *testing.T) {
	cases := []struct {
		name   string
		stream string
		want   []string
	}{
		{"a text block never stopped, on end_turn", sse(start, blockStart("0", `{"type":"text","text":""}`),
			delta("0", `{"type":"text_delta","text":"Hello"}`), messageDelta("end_turn"), stop),
			[]string{"text_delta:Hello", "item:MessageItem", "done"}},
		{"a thinking block stopped, then a text block never stopped", sse(start,
			blockStart("0", `{"type":"thinking","thinking":"","signature":""}`), delta("0", `{"type":"thinking_delta","thinking":"hm"}`),
			blockStop("0"), blockStart("1", `{"type":"text","text":""}`), delta("1", `{"type":"text_delta","text":"ok"}`),
			messageDelta("end_turn"), stop),
			[]string{"reasoning_delta:hm", "item:ReasoningItem", "text_delta:ok", "item:MessageItem", "done"}},
		{"text, then a call never stopped, cut on max_tokens", sse(start, blockStart("0", `{"type":"text","text":""}`),
			delta("0", `{"type":"text_delta","text":"Hi"}`), blockStop("0"),
			blockStart("1", `{"type":"tool_use","id":"toolu_1","name":"record_city","input":{}}`),
			delta("1", `{"type":"input_json_delta","partial_json":"{\"city\":"}`), messageDelta("max_tokens"), stop),
			[]string{"text_delta:Hi", "item:MessageItem", "done"}},
	}
	for _, tc := range cases {
		events, kinds, err := collect(t, tc.stream)
		if err != nil {
			t.Errorf("%s: err %v; want none", tc.name, err)
			continue
		}
		if !reflect.DeepEqual(kinds, tc.want) {
			t.Errorf("%s: events %q; want %q", tc.name, kinds, tc.want)
			continue
		}
		var items []any
		var output []any
		for _, ev := range events {
			if ev.Item != nil {
				items = append(items, ev.Item)
			}
			if ev.Response != nil {
				for _, it := range ev.Response.Output {
					output = append(output, it)
				}
			}
		}
		if !reflect.DeepEqual(items, output) {
			t.Errorf("%s: EventItems %+v; want EventDone's output in order, %+v", tc.name, items, output)
		}
	}
}
