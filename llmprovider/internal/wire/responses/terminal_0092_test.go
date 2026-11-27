package responses

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// 0092-PLAN Phase 1 (0092-MADR ST1 to ST32): the Responses stream reader
// takes, at its terminal event, each item the terminal response carries that
// no output_item.done delivered, paired by id, then by output_index within
// one type, then in order within one type (D2). Every stream here is this
// file's own, in the shapes the MADR measured (M1 to M8); none is a recorded
// one.

// tfEvent is one server-sent event of kind, its payload's type kind and its
// other fields fields, each starting with a comma.
func tfEvent(kind, fields string) string {
	return "event: " + kind + "\ndata: {\"type\":\"" + kind + "\"" + fields + "}\n\n"
}

// tfCreated opens a stream.
var tfCreated = tfEvent("response.created", `,"response":{"id":"resp_t","model":"m","status":"in_progress","output":[]}`)

// tfID is an item's id field, or nothing for "".
func tfID(id string) string {
	if id == "" {
		return ""
	}
	return `"id":"` + id + `",`
}

// tfMsg is a message item with text, or with no content for "".
func tfMsg(id, text string) string {
	content := `[]`
	if text != "" {
		content = `[{"type":"output_text","text":"` + text + `"}]`
	}
	return `{` + tfID(id) + `"type":"message","role":"assistant","status":"completed","content":` + content + `}`
}

// tfCall is a function_call item; args is the arguments' JSON text.
func tfCall(id, callID, name, args string) string {
	return `{` + tfID(id) + `"type":"function_call","status":"completed","call_id":"` + callID + `","name":"` + name +
		`","arguments":` + js0092(args) + `}`
}

// tfReasoning is a reasoning item with a summary text and an encrypted part,
// either "" for none.
func tfReasoning(id, text, encrypted string) string {
	summary := `[]`
	if text != "" {
		summary = `[{"type":"summary_text","text":"` + text + `"}]`
	}
	enc := ""
	if encrypted != "" {
		enc = `,"encrypted_content":"` + encrypted + `"`
	}
	return `{` + tfID(id) + `"type":"reasoning","summary":` + summary + enc + `}`
}

// js0092 is s as a JSON string.
func js0092(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// tfDone is an output_item.done for item at index; a negative index sends no
// output_index.
func tfDone(index int, item string) string {
	if index < 0 {
		return tfEvent("response.output_item.done", `,"item":`+item)
	}
	return tfEvent("response.output_item.done", fmt.Sprintf(`,"output_index":%d,"item":%s`, index, item))
}

// tfAdded is an output_item.added for item at index.
func tfAdded(index int, item string) string {
	return tfEvent("response.output_item.added", fmt.Sprintf(`,"output_index":%d,"item":%s`, index, item))
}

// tfDelta is a text delta.
func tfDelta(text string) string {
	return tfEvent("response.output_text.delta", `,"output_index":0,"content_index":0,"delta":"`+text+`"`)
}

// tfCompleted is a response.completed whose output is items, with usage 3 in
// and 5 out.
func tfCompleted(items ...string) string {
	return tfEvent("response.completed", `,"response":{"id":"resp_t","model":"m","status":"completed","output":[`+
		strings.Join(items, ",")+`],"usage":{"input_tokens":3,"output_tokens":5}}`)
}

// tfCut is a response.incomplete for the output limit whose output is items.
func tfCut(items ...string) string {
	return tfEvent("response.incomplete", `,"response":{"id":"resp_t","model":"m","status":"incomplete",`+
		`"incomplete_details":{"reason":"max_output_tokens"},"output":[`+strings.Join(items, ",")+`],`+
		`"usage":{"input_tokens":3,"output_tokens":5}}`)
}

// tfRun is one stream read by d's Events and by its ReadStream.
type tfRun struct {
	events []llmprovider.Event
	err    error
	// done is EventDone's response, nil with none.
	done *llmprovider.Response
	// read and readErr are ReadStream's.
	read    *llmprovider.Response
	readErr error
}

// tfRead reads stream both ways with d.
func tfRead(d Decoder, stream string) tfRun {
	var r tfRun
	r.err = d.Events("p", strings.NewReader(stream), func(ev llmprovider.Event) bool {
		r.events = append(r.events, ev)
		if ev.Type == llmprovider.EventDone {
			r.done = ev.Response
		}
		return true
	})
	r.read, r.readErr = d.ReadStream("p", strings.NewReader(stream))
	return r
}

// kinds is the run's events as eventKinds writes them.
func (r tfRun) kinds() []string { return eventKinds(r.events) }

// items is the run's EventItems' items.
func (r tfRun) items() []llmprovider.Item {
	var out []llmprovider.Item
	for _, ev := range r.events {
		if ev.Type == llmprovider.EventItem {
			out = append(out, ev.Item)
		}
	}
	return out
}

// outputOf is the response's items, each as kind:text, for comparison.
func outputOf(resp *llmprovider.Response) []string {
	if resp == nil {
		return nil
	}
	out := make([]string, 0, len(resp.Output))
	for _, item := range resp.Output {
		switch v := item.(type) {
		case llmprovider.MessageItem:
			out = append(out, "message:"+v.Text)
		case llmprovider.FunctionCallItem:
			out = append(out, "call:"+v.Name+" "+v.Arguments)
		case llmprovider.ReasoningItem:
			out = append(out, "reasoning:"+v.Text+"/"+v.Encrypted)
		default:
			out = append(out, fmt.Sprintf("%T", item))
		}
	}
	return out
}

// check fails t unless both readers return a response whose output is want,
// finishing finish, and the EventItems are that output in order.
func (r tfRun) check(t *testing.T, name string, finish llmprovider.FinishReason, want []string) {
	t.Helper()
	if r.err != nil || r.done == nil || r.readErr != nil {
		t.Errorf("%s: Events err %v, EventDone %t, ReadStream err %v; want a response", name, r.err, r.done != nil, r.readErr)
		return
	}
	if got := outputOf(r.done); !reflect.DeepEqual(got, want) || r.done.FinishReason != finish {
		t.Errorf("%s: output %q finishing %q; want %q finishing %q", name, got, r.done.FinishReason, want, finish)
	}
	if !reflect.DeepEqual(r.done, r.read) {
		t.Errorf("%s: EventDone's response %+v, ReadStream's %+v; want them equal", name, r.done, r.read)
	}
	if items := r.items(); !reflect.DeepEqual(items, r.done.Output) {
		t.Errorf("%s: EventItems %+v; want the response's output in order, %+v", name, items, r.done.Output)
	}
}

// m1 is M1's shape: a text delta, no .done, the message only in the
// terminal output.
func m1() string {
	return tfCreated + tfAdded(0, tfMsg("msg_1", "")) + tfDelta("Hello") + tfCompleted(tfMsg("msg_1", "Hello"))
}

// TestTerminalFill_Text_0092 (ST1, ST17, ST31): M1's message, only in the
// terminal output, is taken by Events and ReadStream alike, finishing stop,
// with the terminal event's usage; a refusal part finishes content_filter.
func TestTerminalFill_Text_0092(t *testing.T) {
	r := tfRead(Decoder{}, m1())
	r.check(t, "M1", llmprovider.FinishStop, []string{"message:Hello"})
	if want := []string{"text_delta:Hello", "item:MessageItem", "done"}; !reflect.DeepEqual(r.kinds(), want) {
		t.Errorf("M1: events %v; want %v", r.kinds(), want)
	}
	if r.done != nil && (r.done.Usage.InputTokens != 3 || r.done.Usage.OutputTokens != 5 || r.done.ID != "resp_t") {
		t.Errorf("M1: usage %+v, id %q; want 3 in, 5 out, resp_t", r.done.Usage, r.done.ID)
	}

	refusal := `{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"refusal","refusal":"no"}]}`
	r = tfRead(Decoder{}, tfCreated+tfCompleted(refusal))
	r.check(t, "a refusal only in the terminal", llmprovider.FinishContentFilter, []string{"message:no"})
}

// TestTerminalFill_Call_0092 (ST2): a call only in the terminal output is
// taken, finishing tool_calls; so are a client tool search with a Search
// tool and Grok's dispatch wrapper, decoded as from a .done.
func TestTerminalFill_Call_0092(t *testing.T) {
	msg := tfAdded(0, tfMsg("msg_1", "")) + tfDelta("Hello") + tfDone(0, tfMsg("msg_1", "Hello"))
	r := tfRead(Decoder{}, tfCreated+msg+tfCompleted(tfMsg("msg_1", "Hello"), tfCall("fc_1", "call_1", "lookup", `{"x":1}`)))
	r.check(t, "M2, a call only in the terminal", llmprovider.FinishToolCalls, []string{"message:Hello", `call:lookup {"x":1}`})

	search := `{"id":"ts_1","type":"tool_search_call","execution":"client","call_id":"call_s","arguments":{"query":"x"}}`
	r = tfRead(Decoder{Search: "tool_search"}, tfCreated+msg+tfCompleted(tfMsg("msg_1", "Hello"), search))
	r.check(t, "a client tool search only in the terminal", llmprovider.FinishToolCalls,
		[]string{"message:Hello", `call:tool_search {"query":"x"}`})

	wrapped := tfCall("fc_2", "call_2", "use_tool", `{"tool_name":"read_file","tool_input":{"path":"a"}}`)
	r = tfRead(Decoder{Dispatch: map[string]bool{"read_file": true}}, tfCreated+msg+tfCompleted(tfMsg("msg_1", "Hello"), wrapped))
	r.check(t, "Grok's dispatch wrapper only in the terminal", llmprovider.FinishToolCalls,
		[]string{"message:Hello", `call:read_file {"path":"a"}`})
}

// TestTerminalFill_Reasoning_0092 (ST3): M4's encrypted reasoning item, only
// in the terminal output, is taken in the Responses format, and the next
// request's input carries it.
func TestTerminalFill_Reasoning_0092(t *testing.T) {
	stream := tfCreated + tfAdded(1, tfMsg("msg_1", "")) + tfDelta("Hello") + tfDone(1, tfMsg("msg_1", "Hello")) +
		tfCompleted(tfReasoning("rs_1", "think", "ENC"), tfMsg("msg_1", "Hello"))
	r := tfRead(Decoder{}, stream)
	r.check(t, "M4", llmprovider.FinishStop, []string{"message:Hello", "reasoning:think/ENC"})
	if r.done == nil || len(r.done.Output) != 2 {
		return
	}
	if rs, ok := r.done.Output[1].(llmprovider.ReasoningItem); !ok || rs.Format != wire.FormatResponses {
		t.Errorf("M4: the taken item %+v; want a ReasoningItem in the Responses format", r.done.Output[1])
	}
	next, err := json.Marshal(Input(r.done.Output))
	if err != nil || !strings.Contains(string(next), `"encrypted_content":"ENC"`) || !strings.Contains(string(next), `"type":"reasoning"`) {
		t.Errorf("M4: the next request's input %s (%v); want the reasoning item with its encrypted part", next, err)
	}
}

// TestTerminalFill_All_0092 (ST4, ST32): with no .done at all, the
// terminal's reasoning item, message and call are taken in its order and
// yielded in that order before EventDone, with no delta after the first
// taken item.
func TestTerminalFill_All_0092(t *testing.T) {
	stream := tfCreated + tfEvent("response.reasoning_summary_text.delta", `,"output_index":0,"delta":"think"`) +
		tfDelta("Hello") +
		tfCompleted(tfReasoning("rs_1", "think", "ENC"), tfMsg("msg_1", "Hello"), tfCall("fc_1", "call_1", "lookup", `{"x":1}`))
	r := tfRead(Decoder{}, stream)
	r.check(t, "no .done", llmprovider.FinishToolCalls, []string{"reasoning:think/ENC", "message:Hello", `call:lookup {"x":1}`})
	want := []string{"reasoning_delta:think", "text_delta:Hello", "item:ReasoningItem", "item:MessageItem",
		"item:FunctionCallItem", "done"}
	if got := r.kinds(); !reflect.DeepEqual(got, want) {
		t.Errorf("no .done: events %v; want %v", got, want)
	}
	taken := false
	for _, ev := range r.events {
		switch ev.Type {
		case llmprovider.EventItem:
			taken = true
		case llmprovider.EventTextDelta, llmprovider.EventReasoningDelta:
			if taken {
				t.Errorf("no .done: a delta %q after the first taken item", ev.Text)
			}
		default:
		}
	}
}

// TestTerminalFill_NoDuplicate_0092 (ST5): a compliant stream, every item by
// .done and the terminal output the same items, takes nothing.
func TestTerminalFill_NoDuplicate_0092(t *testing.T) {
	rs, msg, call := tfReasoning("rs_1", "think", "ENC"), tfMsg("msg_1", "Hello"), tfCall("fc_1", "call_1", "lookup", `{"x":1}`)
	stream := tfCreated + tfDone(0, rs) + tfAdded(1, tfMsg("msg_1", "")) + tfDelta("Hello") + tfDone(1, msg) +
		tfAdded(2, tfCall("fc_1", "call_1", "lookup", "")) + tfDone(2, call) + tfCompleted(rs, msg, call)
	r := tfRead(Decoder{}, stream)
	r.check(t, "a compliant stream", llmprovider.FinishToolCalls, []string{"reasoning:think/ENC", "message:Hello", `call:lookup {"x":1}`})
}

// TestTerminalFill_Match_0092 (ST6, ST13): D2's pairing, on a completed and
// a cut stream: (a) a .done with no id, the terminal's item with one (M5);
// (b) a .done skipped for item 0 and given, with no id, for item 1; (c) as
// (b) with no output_index, item 0 a reasoning item; (d) B by id, A with no
// id taken; (e) the index pass does not pair across types; (f) OQ5's case,
// nothing of that type taken.
func TestTerminalFill_Match_0092(t *testing.T) {
	for _, tc := range []struct {
		name     string
		dones    string
		terminal []string
		want     []string
	}{
		{"(a)", tfDone(0, tfMsg("", "Hello")), []string{tfMsg("msg_1", "Hello")}, []string{"message:Hello"}},
		{"(b)", tfDone(1, tfMsg("", "B")), []string{tfMsg("msg_a", "A"), tfMsg("msg_b", "B")}, []string{"message:B", "message:A"}},
		{"(c)", tfDone(-1, tfMsg("", "B")), []string{tfReasoning("rs_1", "", "ENC"), tfMsg("msg_b", "B")},
			[]string{"message:B", "reasoning:/ENC"}},
		{"(d)", tfDone(-1, tfMsg("msg_b", "B")), []string{tfMsg("", "A"), tfMsg("msg_b", "B")}, []string{"message:B", "message:A"}},
		{"(e)", tfDone(0, tfMsg("", "B")), []string{tfReasoning("rs_1", "", "ENC"), tfMsg("", "B")},
			[]string{"message:B", "reasoning:/ENC"}},
		{"(f)", tfDone(-1, tfMsg("", "B")), []string{tfMsg("", "A"), tfMsg("", "B")}, []string{"message:B"}},
	} {
		completed := tfRead(Decoder{}, tfCreated+tc.dones+tfCompleted(tc.terminal...))
		completed.check(t, tc.name+", completed", llmprovider.FinishStop, tc.want)
		cut := tfRead(Decoder{}, tfCreated+tc.dones+tfCut(tc.terminal...))
		cut.check(t, tc.name+", cut", llmprovider.FinishLength, tc.want)
	}
}

// TestTerminalFill_DoneWins_0092 (ST7): a terminal item that differs from
// the .done item of the same id (M8) leaves the delivered one standing, and
// an empty terminal output removes nothing.
func TestTerminalFill_DoneWins_0092(t *testing.T) {
	head := tfCreated + tfAdded(0, tfMsg("msg_1", "")) + tfDelta("Hello") + tfDone(0, tfMsg("msg_1", "Hello"))
	for name, terminal := range map[string]string{
		"M8, the terminal's text differs": tfCompleted(tfMsg("msg_1", "Hello, world")),
		"an empty terminal output":        tfCompleted(),
	} {
		r := tfRead(Decoder{}, head+terminal)
		r.check(t, name, llmprovider.FinishStop, []string{"message:Hello"})
	}
}

// TestTerminalFill_Deltas_0092 (ST8): for an item no .done delivered, the
// terminal's item is the response's; the deltas are not kept.
func TestTerminalFill_Deltas_0092(t *testing.T) {
	r := tfRead(Decoder{}, tfCreated+tfAdded(0, tfMsg("msg_1", ""))+tfDelta("Hel")+tfCompleted(tfMsg("msg_1", "Hello")))
	r.check(t, "deltas Hel, the terminal Hello", llmprovider.FinishStop, []string{"message:Hello"})
}

// TestTerminalFill_Order_0092 (ST9, ST14): the response's output is the
// delivered items in arrival order, then the taken ones in the terminal's
// order, and the EventItems come in that order; on a cut the take runs
// before the cut rules, so a taken last {} call is the cut call. Which item
// is last for the cut is the server's order, the terminal output's, not the
// response's (ST37, "Server's order decides the cut (Recommended)"): a
// delivered {} call the server sent last is the cut call, though a taken
// message follows it in the response, and is never yielded; a taken {} call
// the server did not send last is a whole call.
func TestTerminalFill_Order_0092(t *testing.T) {
	stream := tfCreated + tfAdded(1, tfMsg("msg_1", "")) + tfDelta("A") + tfDone(1, tfMsg("msg_1", "A")) +
		tfCompleted(tfReasoning("rs_1", "", "ENC"), tfMsg("msg_1", "A"), tfCall("fc_1", "call_1", "lookup", `{"x":1}`))
	r := tfRead(Decoder{}, stream)
	r.check(t, "delivered, then taken", llmprovider.FinishToolCalls, []string{"message:A", "reasoning:/ENC", `call:lookup {"x":1}`})

	cut := tfCreated + tfDone(0, tfMsg("msg_1", "A")) +
		tfCut(tfMsg("msg_1", "A"), tfCall("fc_1", "call_1", "lookup", `{"x":1}`), tfCall("fc_2", "call_z", "now", "{}"))
	r = tfRead(Decoder{}, cut)
	r.check(t, "a cut, the last taken call {}", llmprovider.FinishLength, []string{"message:A", `call:lookup {"x":1}`})
	if r.done != nil && (len(r.done.Warnings) != 1 || r.done.Warnings[0].Kind != llmprovider.WarningToolCallCut ||
		!strings.Contains(r.done.Warnings[0].Detail, "call_z")) {
		t.Errorf("a cut, the last taken call {}: warnings %+v; want one naming call_z", r.done.Warnings)
	}

	now := tfCall("fc_2", "call_z", "now", "{}")
	serverLast := tfCreated + tfAdded(0, tfMsg("msg_1", "")) + tfAdded(1, tfCall("fc_2", "call_z", "now", "")) +
		tfDone(1, now) + tfCut(tfMsg("msg_1", "Hello"), now)
	r = tfRead(Decoder{}, serverLast)
	r.check(t, "a cut, the server's last item a delivered {} call", llmprovider.FinishLength, []string{"message:Hello"})
	if r.done != nil && (len(r.done.Warnings) != 1 || r.done.Warnings[0].Kind != llmprovider.WarningToolCallCut ||
		!strings.Contains(r.done.Warnings[0].Detail, "call_z")) {
		t.Errorf("a cut, the server's last item a delivered {} call: warnings %+v; want one naming call_z", r.done.Warnings)
	}

	notLast := tfCreated + tfAdded(1, tfMsg("msg_1", "")) + tfDone(1, tfMsg("msg_1", "Hello")) +
		tfCut(now, tfMsg("msg_1", "Hello"))
	r = tfRead(Decoder{}, notLast)
	r.check(t, "a cut, a taken {} call not the server's last", llmprovider.FinishLength, []string{"message:Hello", "call:now {}"})
	if r.done != nil && len(r.done.Warnings) != 0 {
		t.Errorf("a cut, a taken {} call not the server's last: warnings %+v; want none, a whole call", r.done.Warnings)
	}

	// The cut applies only to a call not yet yielded (ST9, 0088-MADR ST109):
	// .done events out of order, the server's last {} call delivered first
	// and released by the next .done, is a whole call, kept as yielded.
	first, last := tfCall("fc_a", "call_a", "first", "{}"), tfCall("fc_b", "call_b", "last", "{}")
	outOfOrder := tfCreated + tfAdded(0, tfCall("fc_a", "call_a", "first", "")) + tfAdded(1, tfCall("fc_b", "call_b", "last", "")) +
		tfDone(1, last) + tfDone(0, first) + tfCut(first, last)
	r = tfRead(Decoder{}, outOfOrder)
	r.check(t, "a cut, .done out of order, the server's last call yielded", llmprovider.FinishLength,
		[]string{"call:last {}", "call:first {}"})
	if r.done != nil && len(r.done.Warnings) != 0 {
		t.Errorf("a cut, .done out of order, the server's last call yielded: warnings %+v; want none", r.done.Warnings)
	}
}

// TestTerminalFill_Held_0092 (ST10): on a completed stream, a held {} call
// is yielded at the end, then the taken calls, a taken {} call kept.
func TestTerminalFill_Held_0092(t *testing.T) {
	first := tfCall("fc_f", "call_f", "first", "{}")
	stream := tfCreated + tfDone(0, first) +
		tfCompleted(first, tfCall("fc_1", "call_1", "lookup", `{"x":1}`), tfCall("fc_2", "call_2", "now", "{}"))
	r := tfRead(Decoder{}, stream)
	r.check(t, "a held call, then taken calls", llmprovider.FinishToolCalls,
		[]string{"call:first {}", `call:lookup {"x":1}`, "call:now {}"})
}

// TestTerminalFill_EmptyDone_0092 (ST12): a .done that added nothing gives
// way to its terminal partner, taken after the delivered items (M7, and an
// empty message); a reasoning item kept with text and no encrypted part is
// not filled in.
func TestTerminalFill_EmptyDone_0092(t *testing.T) {
	m7 := tfCreated + tfDone(0, tfReasoning("rs_1", "", "")) + tfAdded(1, tfMsg("msg_1", "")) + tfDelta("Hello") +
		tfDone(1, tfMsg("msg_1", "Hello")) + tfCompleted(tfReasoning("rs_1", "", "ENC"), tfMsg("msg_1", "Hello"))
	tfRead(Decoder{}, m7).check(t, "M7", llmprovider.FinishStop, []string{"message:Hello", "reasoning:/ENC"})

	empty := tfCreated + tfAdded(0, tfMsg("msg_1", "")) + tfDelta("Hello") + tfDone(0, tfMsg("msg_1", "")) +
		tfCompleted(tfMsg("msg_1", "Hello"))
	tfRead(Decoder{}, empty).check(t, "an empty message .done", llmprovider.FinishStop, []string{"message:Hello"})

	kept := tfCreated + tfDone(0, tfReasoning("rs_1", "think", "")) + tfDone(1, tfMsg("msg_1", "Hello")) +
		tfCompleted(tfReasoning("rs_1", "think", "ENC"), tfMsg("msg_1", "Hello"))
	tfRead(Decoder{}, kept).check(t, "a reasoning item kept with text", llmprovider.FinishStop,
		[]string{"reasoning:think/", "message:Hello"})
}

// TestTerminalFill_Plain_0092 (ST16): the plain decode of M6's completed
// reply returns its message and call, as before.
func TestTerminalFill_Plain_0092(t *testing.T) {
	resp, err := DecodeFor("p")(strings.NewReader(`{"id":"resp_t","model":"m","status":"completed","output":[` +
		tfMsg("msg_1", "Hello") + "," + tfCall("fc_1", "call_1", "lookup", `{"x":1}`) + `]}`))
	if want := []string{"message:Hello", `call:lookup {"x":1}`}; err != nil || !reflect.DeepEqual(outputOf(resp), want) ||
		resp.FinishReason != llmprovider.FinishToolCalls {
		t.Errorf("M6: %+v (%v); want %q finishing tool_calls", resp, err, want)
	}
}

// TestTerminalFill_Dropped_0092 (ST28): a terminal-only item appendOutput
// drops is dropped, as its .done would be: the reply finishes as if it were
// absent.
func TestTerminalFill_Dropped_0092(t *testing.T) {
	head := tfCreated + tfAdded(0, tfMsg("msg_1", "")) + tfDelta("Hello") + tfDone(0, tfMsg("msg_1", "Hello"))
	for _, tc := range []struct {
		name string
		d    Decoder
		item string
	}{
		{"an empty message", Decoder{}, tfMsg("msg_2", "")},
		{"an empty reasoning item", Decoder{}, tfReasoning("rs_1", "", "")},
		{"a hosted tool search", Decoder{Search: "tool_search"},
			`{"id":"ts_1","type":"tool_search_call","execution":"server","call_id":"call_s","arguments":{"query":"x"}}`},
		{"a client tool search, no Search tool", Decoder{},
			`{"id":"ts_1","type":"tool_search_call","execution":"client","call_id":"call_s","arguments":{"query":"x"}}`},
		{"a custom_tool_call", Decoder{}, `{"id":"ct_1","type":"custom_tool_call","call_id":"call_c","name":"x","input":"y"}`},
	} {
		r := tfRead(tc.d, head+tfCompleted(tfMsg("msg_1", "Hello"), tc.item))
		r.check(t, tc.name+" only in the terminal", llmprovider.FinishStop, []string{"message:Hello"})
	}
}

// TestTerminalFill_Collect_0092 (ST30): a Generate read over the stream
// counts the deltas against StreamAnswerLimit, 32 MiB (0091-MADR ST71), not a
// taken message's text again: 15 MiB of reasoning deltas, 10 MiB of text
// deltas and a terminal message of the same 10 MiB, under EventLimit, return
// the answer, about 25 MiB counted; counting the message too, about 35 MiB,
// would be over the bound. The bound is checked first (0092-MADR ST36): moved
// out of [25 MiB, 35 MiB), the counts above would no longer straddle it, and
// the plant would pass unseen.
func TestTerminalFill_Collect_0092(t *testing.T) {
	// Wanted: 25<<20 <= wire.StreamAnswerLimit && wire.StreamAnswerLimit < 35<<20.
	bound := wire.StreamAnswerLimit
	if bound < 25<<20 || bound >= 35<<20 {
		t.Fatalf("StreamAnswerLimit is %d MiB; this test's counts, about 25 MiB and 35 MiB, need it in [25 MiB, 35 MiB)",
			bound>>20)
	}
	chunk := strings.Repeat("x", 64<<10)
	var b strings.Builder
	b.WriteString(tfCreated + tfAdded(0, tfMsg("msg_1", "")))
	for range 240 {
		b.WriteString(tfEvent("response.reasoning_summary_text.delta", `,"output_index":0,"delta":"`+chunk+`"`))
	}
	for range 160 {
		b.WriteString(tfDelta(chunk))
	}
	text := strings.Repeat(chunk, 160)
	b.WriteString(tfCompleted(tfMsg("msg_1", text)))
	stream := b.String()
	var seq iter.Seq2[llmprovider.Event, error] = func(yield func(llmprovider.Event, error) bool) {
		stopped := false
		err := Decoder{}.Events("p", strings.NewReader(stream), func(ev llmprovider.Event) bool {
			if !yield(ev, nil) {
				stopped = true
				return false
			}
			return true
		})
		if err != nil && !stopped {
			yield(llmprovider.Event{}, err)
		}
	}
	resp, err := wire.Collect("p", seq)
	if err != nil || resp == nil || len(resp.OutputText()) != len(text) {
		var n int
		if resp != nil {
			n = len(resp.OutputText())
		}
		t.Errorf("15 MiB of reasoning and 10 MiB of text deltas, then the terminal's message: %d bytes of text (%v); want the answer, %d bytes", n, err, len(text))
	}
}

// tfLog is a slog handler that keeps every record, each attribute as
// key=value.
type tfLog struct {
	mu      sync.Mutex
	records []string
	levels  []slog.Level
}

func (h *tfLog) Enabled(context.Context, slog.Level) bool { return true }

func (h *tfLog) Handle(_ context.Context, r slog.Record) error {
	line := r.Message
	r.Attrs(func(a slog.Attr) bool {
		line += fmt.Sprintf(" %s=%v", a.Key, a.Value)
		return true
	})
	h.mu.Lock()
	h.records = append(h.records, line)
	h.levels = append(h.levels, r.Level)
	h.mu.Unlock()
	return nil
}

func (h *tfLog) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *tfLog) WithGroup(string) slog.Handler      { return h }

// TestTerminalFill_Log_0092 (ST20): a stream that took items writes one
// Debug line, naming the provider, the terminal event and the count taken
// of each type, with no text or id; a compliant stream writes none.
func TestTerminalFill_Log_0092(t *testing.T) {
	log := &tfLog{}
	d := Decoder{Logger: slog.New(log)}
	tfRead(d, m1())
	// Events and ReadStream each read the stream once.
	if len(log.records) != 2 || log.records[0] != log.records[1] {
		t.Fatalf("M1, read twice: records %q; want one line a read", log.records)
	}
	line := log.records[0]
	for _, want := range []string{"provider=p", "event=response.completed", "message=1"} {
		if !strings.Contains(line, want) {
			t.Errorf("M1: the line %q; want it to hold %q", line, want)
		}
	}
	for _, leak := range []string{"Hello", "msg_1", "resp_t"} {
		if strings.Contains(line, leak) {
			t.Errorf("M1: the line %q holds %q", line, leak)
		}
	}
	if log.levels[0] != slog.LevelDebug {
		t.Errorf("M1: the line's level %v; want Debug", log.levels[0])
	}

	quiet := &tfLog{}
	msg := tfMsg("msg_1", "Hello")
	tfRead(Decoder{Logger: slog.New(quiet)}, tfCreated+tfDone(0, msg)+tfCompleted(msg))
	if len(quiet.records) != 0 {
		t.Errorf("a compliant stream: records %q; want none", quiet.records)
	}
	if r := tfRead(Decoder{}, m1()); r.err != nil {
		t.Errorf("M1 with no logger: %v; want the answer", r.err)
	}
}

// TestTerminalFill_Bound_0092 (ST33): the pairing considers up to
// wire.MaxStreamIndex items; a terminal output of 4096 messages, one of them
// by .done, gives all 4096, and one of 4097 gives only the .done message.
func TestTerminalFill_Bound_0092(t *testing.T) {
	for _, n := range []int{wire.MaxStreamIndex, wire.MaxStreamIndex + 1} {
		items := make([]string, n)
		for i := range items {
			items[i] = tfMsg(fmt.Sprintf("msg_%d", i), "t")
		}
		r := tfRead(Decoder{}, tfCreated+tfDone(0, items[0])+tfCompleted(items...))
		want := []string{"message:t"}
		if n <= wire.MaxStreamIndex {
			want = slices.Repeat(want, n)
		}
		r.check(t, fmt.Sprintf("a terminal of %d items", n), llmprovider.FinishStop, want)
	}

	// Past the bound, ST11's check is not run either: a call announced and
	// never finished does not fail the stream.
	items := make([]string, wire.MaxStreamIndex+1)
	for i := range items {
		items[i] = tfMsg(fmt.Sprintf("msg_%d", i), "t")
	}
	lost := tfAdded(1, tfCall("fc_x", "call_x", "lookup", ""))
	r := tfRead(Decoder{}, tfCreated+tfDone(0, items[0])+lost+tfCompleted(items...))
	r.check(t, "past the bound, a call announced and never finished", llmprovider.FinishStop, []string{"message:t"})
}

// TestTerminalFill_AnnounceBound_0092 (ST33, ST34): the reader stops
// recording announced items once they pass wire.MaxStreamIndex, where ST11's
// check is off, so a long run of output_item.added events costs constant work
// each; 10,000 distinct announcements leave 4097 recorded.
func TestTerminalFill_AnnounceBound_0092(t *testing.T) {
	s := &streamState{result: &llmprovider.Response{}}
	for i := range 10000 {
		s.announce(outputItem{Type: itemTypeMessage, ID: fmt.Sprintf("msg_%d", i)}, i)
	}
	if got, want := len(s.announced), wire.MaxStreamIndex+1; got != want {
		t.Errorf("10000 announcements: %d recorded; want %d, one past the bound", got, want)
	}
}

// TestTerminalFill_Reannounced_0092 (ST34): a message announced twice by its
// id and delivered once is one item; the stream completes with it.
func TestTerminalFill_Reannounced_0092(t *testing.T) {
	stream := tfCreated + tfAdded(0, tfMsg("msg_1", "")) + tfAdded(0, tfMsg("msg_1", "")) + tfDelta("Hello") +
		tfDone(0, tfMsg("msg_1", "Hello")) + tfCompleted(tfMsg("msg_1", "Hello"))
	tfRead(Decoder{}, stream).check(t, "a repeated announcement", llmprovider.FinishStop, []string{"message:Hello"})

	// ST38, "Match by call ID, then position (Recommended)": with no id, a
	// repeat is matched by its call_id, else by its output_index and type.
	byIndex := tfCreated + tfAdded(0, tfMsg("", "")) + tfAdded(0, tfMsg("", "")) + tfDone(0, tfMsg("", "Hello")) +
		tfCompleted(tfMsg("", "Hello"))
	tfRead(Decoder{}, byIndex).check(t, "a repeat with no id, at one index", llmprovider.FinishStop, []string{"message:Hello"})

	call := tfCall("", "call_1", "lookup", `{"x":1}`)
	byCallID := tfCreated + tfAdded(0, tfCall("", "call_1", "lookup", "")) + tfAdded(1, tfCall("", "call_1", "lookup", "")) +
		tfDone(0, call) + tfCompleted(call)
	tfRead(Decoder{}, byCallID).check(t, "a repeat with no id, one call_id", llmprovider.FinishToolCalls,
		[]string{`call:lookup {"x":1}`})

	added := tfEvent("response.output_item.added", `,"item":`+tfMsg("", ""))
	two := tfCreated + added + added + tfDone(-1, tfMsg("", "Hello")) + tfCompleted(tfMsg("", "Hello"))
	r := tfRead(Decoder{}, two)
	if r.err == nil || !strings.Contains(r.err.Error(), "item_not_finished") {
		t.Errorf("two announcements with no id, call_id or index, one delivered: err %v; want item_not_finished", r.err)
	}

	// ST38 as narrowed (2026-11-27): position matches only an item with
	// neither id nor call_id, so one with an id and one with neither, at one
	// index, are two items; with only the first delivered, the second fails.
	mixed := tfCreated + tfAdded(0, tfMsg("msg_1", "")) + tfAdded(0, tfMsg("", "")) + tfDone(0, tfMsg("msg_1", "Hello")) +
		tfCompleted(tfMsg("msg_1", "Hello"))
	r = tfRead(Decoder{}, mixed)
	if r.err == nil || !strings.Contains(r.err.Error(), "item_not_finished") {
		t.Errorf("an id and an anonymous item at one index, one delivered: err %v; want item_not_finished", r.err)
	}
}

// TestTerminalFill_IndexForm_0092 (ST35): an output_index that is not an
// integer JSON number reads as none: a .done with no id against two messages
// with none is OQ5's case, nothing taken, and the stream is not failed.
func TestTerminalFill_IndexForm_0092(t *testing.T) {
	for _, index := range []string{`1.0`, `"1"`} {
		done := tfEvent("response.output_item.done", `,"output_index":`+index+`,"item":`+tfMsg("", "A"))
		r := tfRead(Decoder{}, tfCreated+done+tfCompleted(tfMsg("", "A"), tfMsg("", "B")))
		r.check(t, "output_index "+index, llmprovider.FinishStop, []string{"message:A"})
	}
}
