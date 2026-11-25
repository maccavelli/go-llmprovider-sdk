package wire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// A reply the output limit cut is reported one way on every wire, keeping
// what is usable (0088-MADR ST75 to ST81, OQ11): a response finishing
// FinishLength with its text and its whole calls, each call the limit cut
// dropped and named in a WarningToolCallCut; with nothing usable, only
// reasoning or only a cut call, ErrEmptyAnswer with the service's reason.

// CutAnswer finishes r, a reply the output limit cut, read on the wire
// where: each call the limit cut (CutCall) is dropped from its output and
// named in a WarningToolCallCut; then it finishes FinishLength, keeping its
// output and warnings, when it holds text or a call, and otherwise it has
// nothing usable and is EmptyAnswer with the service's reason (0088-MADR
// ST75 to ST79, ST94). inPieces reports whether the reply's last call
// arrived in pieces, as on a wire that streams a call's arguments: only then
// is a last call with "{}" the one the limit cut. Where calls arrive whole,
// on generateContent and an Interaction's plain reply, it is a call that
// takes no arguments, and the wire passes false (ST107, ST110). A wire drops
// a call it knows the limit cut, such as a Messages block never closed,
// before it calls CutAnswer.
func CutAnswer(where string, r *llmprovider.Response, reason llmprovider.FinishReason, inPieces bool) (*llmprovider.Response, error) {
	kept := r.Output[:0:0]
	for i, item := range r.Output {
		if call, ok := item.(llmprovider.FunctionCallItem); ok && CutCall(call.Arguments, inPieces && i == len(r.Output)-1) {
			r.Warnings = append(r.Warnings, ToolCallCut(r.Model, call.Name, call.CallID))
			continue
		}
		kept = append(kept, item)
	}
	r.Output = kept
	r.FinishReason = llmprovider.FinishLength
	if r.OutputText() != "" {
		return r, nil
	}
	for _, item := range r.Output {
		if _, ok := item.(llmprovider.FunctionCallItem); ok {
			return r, nil
		}
	}
	return nil, EmptyAnswer(where, reason)
}

// CutCall reports whether a call of a reply the output limit cut, whose
// arguments as returned are args, is one the limit cut (0088-MADR ST78,
// ST94, ST107): its arguments are not whole, a JSON object, or last is set,
// for the reply's last item, where the limit fell, on a wire whose calls
// arrive in pieces, and they are empty, "{}", the form a call cut before its
// first argument takes once the SDK writes empty arguments as "{}"
// (wire.CompactArguments). An earlier call with empty arguments, or a last
// one that arrived whole, is a whole call that takes none.
func CutCall(args string, last bool) bool {
	return !WholeObject(args) || last && noArguments(args)
}

// HeldCalls are the calls a stream holds back, not yet yielded, since a cut
// at the output limit could drop them (0088-MADR ST80): a call CutCall would
// drop were it the reply's last item. A stream whose items arrive as they
// end, such as Messages' and Responses', holds each, releases it once a
// later item shows it was not the last and its arguments are whole, and
// yields the rest the finished response kept just before EventDone, so a
// call the limit cut is never yielded. The zero value is ready.
type HeldCalls struct {
	calls []llmprovider.FunctionCallItem
}

// Hold reports whether item is held: a call CutCall would drop were it the
// last item. A held item is not yielded now.
func (h *HeldCalls) Hold(item llmprovider.Item) bool {
	call, ok := item.(llmprovider.FunctionCallItem)
	if !ok || !CutCall(call.Arguments, true) {
		return false
	}
	h.calls = append(h.calls, call)
	return true
}

// Later yields, for a later item, each held call whose arguments are whole:
// not the last, it is not one the limit cut. It reports false when yield
// stopped the stream.
func (h *HeldCalls) Later(yield func(llmprovider.Event) bool) bool {
	kept := h.calls[:0]
	for i, call := range h.calls {
		if CutCall(call.Arguments, false) {
			kept = append(kept, call)
			continue
		}
		if !yield(llmprovider.Event{Type: llmprovider.EventItem, Item: call}) {
			h.calls = append(kept, h.calls[i+1:]...)
			return false
		}
	}
	h.calls = kept
	return true
}

// Done yields each held call resp kept, before its EventDone. It reports
// false when yield stopped the stream.
func (h *HeldCalls) Done(resp *llmprovider.Response, yield func(llmprovider.Event) bool) bool {
	for _, call := range h.calls {
		kept := slices.ContainsFunc(resp.Output, func(item llmprovider.Item) bool { return item == call })
		if kept && !yield(llmprovider.Event{Type: llmprovider.EventItem, Item: call}) {
			return false
		}
	}
	h.calls = nil
	return true
}

// noArguments reports whether args is the JSON object with no members.
func noArguments(args string) bool {
	var buf bytes.Buffer
	return json.Compact(&buf, []byte(args)) == nil && buf.String() == "{}"
}

// ToolCallCut is the WarningToolCallCut for a call to the tool name, whose
// id is callID, that the output limit cut before its arguments were whole,
// on a reply of model (0088-MADR ST77). The name and the id are the
// service's, so each is stripped and bounded; an empty one is left out.
func ToolCallCut(model, name, callID string) llmprovider.Warning {
	detail := "the output limit cut a tool call before its arguments were whole; it was dropped"
	switch name, callID = redact.Field(name), redact.Field(callID); {
	case name != "" && callID != "":
		detail = fmt.Sprintf("the output limit cut the call to %q (id %q) before its arguments were whole; it was dropped", name, callID)
	case name != "":
		detail = fmt.Sprintf("the output limit cut the call to %q before its arguments were whole; it was dropped", name)
	case callID != "":
		detail = fmt.Sprintf("the output limit cut the call %q before its arguments were whole; it was dropped", callID)
	}
	return llmprovider.Warning{Kind: llmprovider.WarningToolCallCut, Model: model, Detail: detail}
}

// WholeObject reports whether args, a call's arguments as JSON text, are
// whole: a JSON object (0088-MADR ST78).
func WholeObject(args string) bool {
	trimmed := bytes.TrimSpace([]byte(args))
	return len(trimmed) > 0 && trimmed[0] == '{' && json.Valid(trimmed)
}

// Finish maps a service's finish reason through table (0021-MADR W3). A value
// the table does not name is kept as the service sent it, as Response's
// contract says. A reply that carries a call and would finish "stop", or
// names no reason at all, finishes "tool_calls", as every wire reports a call
// (0021-MADR amendment "the empty finish reason and the cut answer's
// reason").
func Finish(raw string, table map[string]llmprovider.FinishReason, hasCall bool) llmprovider.FinishReason {
	finish, ok := table[raw]
	if !ok {
		finish = llmprovider.FinishReason(raw)
	}
	if hasCall && (finish == llmprovider.FinishStop || finish == "") {
		return llmprovider.FinishToolCalls
	}
	return finish
}

// EmptyAnswer is an answer with nothing usable in it: an APIError of kind
// ErrEmptyAnswer, which also matches ErrIncomplete, with the service's finish
// reason in Reason, so a refusal or a safety stop is not mistaken for a plain
// empty answer (0021-MADR W3; 0061-MADR D1). where names the wire. The
// Message is unchanged by 0061, for a caller that still matches it. The
// reason is the service's text, so it is stripped and bounded (0026-MADR F18).
func EmptyAnswer(where string, reason llmprovider.FinishReason) error {
	return &llmprovider.APIError{Kind: llmprovider.ErrEmptyAnswer, Reason: redact.Field(string(reason)),
		Message: where + ": the answer has no content"}
}

// CompactArguments is a call's JSON arguments as a string, kept exactly: not
// decoded and re-encoded, so a large integer keeps its digits and the keys
// their order (0021-MADR W1). Empty arguments, or null, are "{}".
func CompactArguments(raw json.RawMessage) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return "{}", nil
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return "", err
	}
	return buf.String(), nil
}
