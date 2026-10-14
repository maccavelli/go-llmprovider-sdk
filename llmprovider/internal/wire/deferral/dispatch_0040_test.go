package deferral

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0040-MADR D5: what Dispatch needs from the shared form.

// TestForRequest_Dispatch_0040: Grok dispatches, unless a forced choice names
// a deferred tool, which only the append form holds in the list.
func TestForRequest_Dispatch_0040(t *testing.T) {
	req := &llmprovider.Request{Tools: deferralTools(), Input: history()}
	for _, tc := range []struct {
		choice llmprovider.ToolChoice
		want   Form
	}{
		{llmprovider.ToolChoiceAuto, Dispatch},
		{llmprovider.ToolChoiceRequired, Dispatch},
		{llmprovider.ForceTool("tool_search"), Dispatch},
		{llmprovider.ForceTool("read_file"), Dispatch},
		{llmprovider.ForceTool("get_weather"), Append},
	} {
		req.ToolChoice = tc.choice
		if got := ForRequest(Grok, "grok-4.6", req); got != tc.want {
			t.Errorf("choice %v: %v, want %v", tc.choice, got, tc.want)
		}
	}
}

// TestCheckDispatch_0040: a tool named use_tool, or a deferred schema that does
// not encode, is refused under Dispatch only.
func TestCheckDispatch_0040(t *testing.T) {
	clash := &llmprovider.Request{Tools: append(deferralTools(), llmprovider.Tool{Name: UseTool})}
	bad := deferralTools()
	bad[2].Schema = map[string]any{"f": func() {}}
	badSchema := &llmprovider.Request{Tools: bad}
	for _, tc := range []struct {
		name string
		form Form
		req  *llmprovider.Request
		want string
	}{
		{"use_tool, dispatched", Dispatch, clash, `"use_tool"`},
		{"use_tool, appended", Append, clash, ""},
		{"a schema that does not encode, dispatched", Dispatch, badSchema, `"get_weather": schema`},
		{"a schema that does not encode, appended", Append, badSchema, ""},
		{"neither", Dispatch, &llmprovider.Request{Tools: deferralTools()}, ""},
	} {
		err := CheckDispatch(tc.form, tc.req)
		if tc.want == "" && err != nil || tc.want != "" && (!errors.Is(err, llmprovider.ErrInvalidRequest) ||
			!strings.Contains(err.Error(), tc.want)) {
			t.Errorf("%s: %v; want %q", tc.name, err, tc.want)
		}
	}
}

// TestLoads_0040: every result's loads, by call, in order, once each.
func TestLoads_0040(t *testing.T) {
	got := Loads(history())
	for call, want := range map[string][]string{"s1": {"lookup_stock", "get_weather"}, "r1": {"send_email"}, "s2": {"get_weather"}} {
		if !slices.Equal(got[call], want) {
			t.Errorf("Loads[%s] = %v, want %v", call, got[call], want)
		}
	}
	twice := append(history(), llmprovider.ToolLoadItem{CallID: "s1", Name: "lookup_stock"})
	if got := Loads(twice)["s1"]; !slices.Equal(got, []string{"lookup_stock", "get_weather"}) {
		t.Errorf("a repeated load: %v", got)
	}
}

// TestDispatched_0040: the deferred tools, and nothing else.
func TestDispatched_0040(t *testing.T) {
	got := Dispatched(deferralTools())
	if len(got) != 3 || !got["get_weather"] || !got["lookup_stock"] || !got["send_email"] {
		t.Errorf("Dispatched = %v", got)
	}
}
