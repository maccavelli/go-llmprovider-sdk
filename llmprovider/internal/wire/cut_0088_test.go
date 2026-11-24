package wire

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0088-PLAN Phase 1b: the shared cut-answer helpers, measured in their own
// package (R47): CutAnswer and CutCall (0088-MADR ST75 to ST79, ST94; R1 of
// its tenth amendment), ToolCallCut (ST77) and HeldCalls (ST80; R3).

// TestCutAnswer_Helper_0088: CutAnswer drops each call CutCall reports and
// names it, keeps text and whole calls finishing length, and with nothing
// usable is ErrEmptyAnswer with the reason given.
func TestCutAnswer_Helper_0088(t *testing.T) {
	whole := llmprovider.FunctionCallItem{CallID: "a", Name: "lookup", Arguments: `{"x":1}`}
	empty := llmprovider.FunctionCallItem{CallID: "e", Name: "now", Arguments: "{}"}
	partial := llmprovider.FunctionCallItem{CallID: "p", Name: "fetch", Arguments: `{"path":"a"`}
	r := &llmprovider.Response{Model: "m", Output: []llmprovider.Item{
		llmprovider.MessageItem{Text: "partial"}, empty, whole, partial, llmprovider.FunctionCallItem{CallID: "z", Name: "last", Arguments: "{}"}}}
	got, err := CutAnswer("w", r, "length")
	if err != nil || got.FinishReason != llmprovider.FinishLength || len(got.Output) != 3 || len(got.Warnings) != 2 {
		t.Fatalf("CutAnswer = %+v, %v; want text, now and lookup, finishing length, two warnings", got, err)
	}
	for i, name := range []string{"fetch", "last"} {
		if w := got.Warnings[i]; w.Kind != llmprovider.WarningToolCallCut || w.Model != "m" || !strings.Contains(w.Detail, name) {
			t.Errorf("warning %d = %+v; want WarningToolCallCut on m naming %s", i, w, name)
		}
	}
	got, err = CutAnswer("w", &llmprovider.Response{Output: []llmprovider.Item{whole}}, "length")
	if err != nil || len(got.Output) != 1 {
		t.Errorf("a whole call alone: %+v, %v; want it returned", got, err)
	}
	for name, output := range map[string][]llmprovider.Item{
		"only reasoning":  {llmprovider.ReasoningItem{Text: "thinking"}},
		"only a cut call": {partial},
	} {
		got, err := CutAnswer("w", &llmprovider.Response{Output: output}, "max_output_tokens")
		apiErr, ok := errors.AsType[*llmprovider.APIError](err)
		if got != nil || !errors.Is(err, llmprovider.ErrEmptyAnswer) || !ok || apiErr.Reason != "max_output_tokens" {
			t.Errorf("%s: %+v, %v; want ErrEmptyAnswer with reason max_output_tokens", name, got, err)
		}
	}
}

// TestCutCall_0088: a call is cut when its arguments are not a JSON object,
// or are {} and it is the last item.
func TestCutCall_0088(t *testing.T) {
	for _, c := range []struct {
		args string
		last bool
		want bool
	}{
		{`{"x":1}`, true, false}, {`{"x":1}`, false, false}, {"{}", false, false}, {"{ }", true, true},
		{"{}", true, true}, {`{"path":"a"`, false, true}, {"[1]", false, true}, {"", false, true}, {`"s"`, true, true},
	} {
		if got := CutCall(c.args, c.last); got != c.want {
			t.Errorf("CutCall(%q, last %t) = %t; want %t", c.args, c.last, got, c.want)
		}
	}
}

// TestToolCallCut_0088: the warning names the call by its tool and its id
// where each is given.
func TestToolCallCut_0088(t *testing.T) {
	for _, c := range []struct{ name, id, want, not string }{
		{"lookup", "call_1", `"lookup" (id "call_1")`, ""},
		{"lookup", "", `"lookup" before`, "id"},
		{"", "call_1", `call "call_1"`, "to"},
		{"", "", "a tool call", `"`},
	} {
		w := ToolCallCut("m", c.name, c.id)
		if w.Kind != llmprovider.WarningToolCallCut || w.Model != "m" || !strings.Contains(w.Detail, c.want) ||
			c.not != "" && strings.Contains(w.Detail, c.not+" ") {
			t.Errorf("ToolCallCut(%q, %q) = %+v; want a detail holding %q", c.name, c.id, w, c.want)
		}
	}
}

// TestHeldCalls_0088: a call with {} or arguments not whole is held; a later
// item yields the {} one and keeps the other; Done yields those the response
// kept; a stopped yield stops each.
func TestHeldCalls_0088(t *testing.T) {
	empty := llmprovider.FunctionCallItem{CallID: "e", Name: "now", Arguments: "{}"}
	partial := llmprovider.FunctionCallItem{CallID: "p", Name: "fetch", Arguments: `{"path":"a"`}
	whole := llmprovider.FunctionCallItem{CallID: "a", Name: "lookup", Arguments: `{"x":1}`}
	var h HeldCalls
	if h.Hold(llmprovider.MessageItem{Text: "t"}) || h.Hold(whole) || !h.Hold(partial) || !h.Hold(empty) {
		t.Fatal("Hold: want only the partial and the {} calls held")
	}
	var got []string
	collect := func(ev llmprovider.Event) bool {
		got = append(got, ev.Item.(llmprovider.FunctionCallItem).Name)
		return true
	}
	if !h.Later(collect) || strings.Join(got, ",") != "now" {
		t.Errorf("Later yielded %q; want now alone, the partial call kept", got)
	}
	got = nil
	if !h.Done(&llmprovider.Response{Output: []llmprovider.Item{partial}}, collect) || strings.Join(got, ",") != "fetch" {
		t.Errorf("Done yielded %q; want fetch, which the response kept", got)
	}
	got = nil
	_ = h.Hold(empty)
	if !h.Done(&llmprovider.Response{}, collect) || got != nil {
		t.Errorf("Done yielded %q; want nothing: the response dropped the call", got)
	}

	stop := func(llmprovider.Event) bool { return false }
	_, _ = h.Hold(partial), h.Hold(empty)
	if h.Later(stop) || len(h.calls) != 1 {
		t.Errorf("Later with a stopped yield: held %v; want false and the partial call still held", h.calls)
	}
	if h.Done(&llmprovider.Response{Output: []llmprovider.Item{partial}}, stop) {
		t.Error("Done with a stopped yield: want false")
	}
}
