package deferral

import (
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0040-MADR D1's table, A1 to A5: which form each wire and model takes, what
// each form sends, and which tools are loaded.

// TestForm_0040 is D1's table, with the model rules at their edges.
func TestForm_0040(t *testing.T) {
	for _, tc := range []struct {
		wire  Wire
		model string
		want  Form
	}{
		{Messages, "claude-haiku-5-5", MessagesNative},
		{Messages, "claude-haiku-4-5", MessagesNative},
		{Messages, "claude-haiku-4-5-20251001", MessagesNative},
		{Messages, "claude-sonnet-4-5", MessagesNative},
		{Messages, "claude-opus-4-1", Append},
		{Messages, "claude-opus-4-0", Append},
		{Messages, "claude-3-5-haiku", Append},
		{Messages, "claude-sonnet-5", MessagesNative},
		{Messages, "minimax-m3", Append},
		{Messages, "qwen3.8-flash", Append},
		{Messages, "claude-", Append},
		{Responses, "gpt-5.4", ResponsesNative},
		{Responses, "gpt-5.4-mini", ResponsesNative},
		{Responses, "gpt-6-luna", ResponsesNative},
		{Responses, "gpt-6.1-sol", ResponsesNative},
		{Responses, "gpt-5.3", Append},
		{Responses, "gpt-5", Append},
		{Responses, "gpt-4.1-mini", Append},
		{Responses, "o3-pro", Append},
		{Responses, "codex-auto-review", Append},
		{ChatGPT, "codex-auto-review", ResponsesNative},
		{ChatGPT, "gpt-5", ResponsesNative},
		{Grok, "grok-4.6", Dispatch},
		{Other, "claude-haiku-5-5", Append},
		{Other, "gpt-6-luna", Append},
	} {
		if got := Of(tc.wire, tc.model); got != tc.want {
			t.Errorf("Of(%v, %q) = %v, want %v", tc.wire, tc.model, got, tc.want)
		}
	}
}

func deferralTools() []llmprovider.Tool {
	return []llmprovider.Tool{
		{Name: "read_file"},
		{Name: "tool_search", Search: true},
		{Name: "get_weather", Deferred: true},
		{Name: "lookup_stock", Deferred: true},
		{Name: "send_email", Deferred: true},
	}
}

// history: a search loads lookup_stock then get_weather; an event result
// (read_file's) loads send_email; a repeat load of get_weather.
func history() []llmprovider.Item {
	return []llmprovider.Item{
		llmprovider.MessageItem{Text: "hi"},
		llmprovider.FunctionCallItem{CallID: "s1", Name: "tool_search", Arguments: "{}"},
		llmprovider.FunctionCallOutputItem{CallID: "s1", Output: "found"},
		llmprovider.ToolLoadItem{CallID: "s1", Name: "lookup_stock"},
		llmprovider.ToolLoadItem{CallID: "s1", Name: "get_weather"},
		llmprovider.FunctionCallItem{CallID: "r1", Name: "read_file", Arguments: "{}"},
		llmprovider.FunctionCallOutputItem{CallID: "r1", Output: "denied"},
		llmprovider.ToolLoadItem{CallID: "r1", Name: "send_email"},
		llmprovider.FunctionCallItem{CallID: "s2", Name: "tool_search", Arguments: "{}"},
		llmprovider.FunctionCallOutputItem{CallID: "s2", Output: "found"},
		llmprovider.ToolLoadItem{CallID: "s2", Name: "get_weather"},
	}
}

func names(ts []llmprovider.Tool) []string {
	out := make([]string, len(ts))
	for i, tool := range ts {
		out[i] = tool.Name
	}
	return out
}

// TestLoaded_0040: first-load order, without repeats, across results.
func TestLoaded_0040(t *testing.T) {
	want := []string{"lookup_stock", "get_weather", "send_email"}
	if got := Loaded(history()); !slices.Equal(got, want) {
		t.Errorf("Loaded = %v, want %v", got, want)
	}
	if got := Loaded(nil); len(got) != 0 {
		t.Errorf("Loaded(nil) = %v, want none", got)
	}
}

// TestTools_0040: what each form sends.
func TestTools_0040(t *testing.T) {
	req := &llmprovider.Request{Tools: deferralTools(), Input: history()}
	for _, tc := range []struct {
		name string
		form Form
		req  *llmprovider.Request
		want []string
	}{
		{"Messages sends every tool", MessagesNative, req,
			[]string{"read_file", "tool_search", "get_weather", "lookup_stock", "send_email"}},
		// The search's loads go in tool_search_output; the event's load
		// (read_file's result) is appended (D3, Q4).
		{"Responses appends only a non-search load", ResponsesNative, req,
			[]string{"read_file", "tool_search", "send_email"}},
		{"Dispatch sends no deferred tool, and the wrapper", Dispatch, req, []string{"read_file", "tool_search", UseTool}},
		{"Append adds the loaded, in load order", Append, req,
			[]string{"read_file", "tool_search", "lookup_stock", "get_weather", "send_email"}},
		{"Append with a continuation sends every tool", Append,
			&llmprovider.Request{Tools: deferralTools(), PreviousResponseID: "r"},
			[]string{"read_file", "tool_search", "get_weather", "lookup_stock", "send_email"}},
		{"Append, nothing loaded", Append, &llmprovider.Request{Tools: deferralTools()}, []string{"read_file", "tool_search"}},
	} {
		if got := names(Tools(tc.req, tc.form)); !slices.Equal(got, tc.want) {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
	// A request with nothing deferred is returned as it is, whatever the form.
	plain := []llmprovider.Tool{{Name: "a"}, {Name: "b"}}
	for _, form := range []Form{Append, MessagesNative, ResponsesNative, Dispatch} {
		got := Tools(&llmprovider.Request{Tools: plain}, form)
		if len(got) != 2 || &got[0] != &plain[0] {
			t.Errorf("form %v: a request with nothing deferred was copied or changed: %v", form, names(got))
		}
	}
}

// TestForRequest_0040: the wire and model's form, except that a forced
// choice of the Search tool on Responses appends, since tool_choice cannot
// name a tool_search entry; and a request with nothing deferred is Append,
// which sends it as before.
func TestForRequest_0040(t *testing.T) {
	req := &llmprovider.Request{Tools: deferralTools()}
	if got := ForRequest(Responses, "gpt-6-luna", req); got != ResponsesNative {
		t.Errorf("unforced: %v, want ResponsesNative", got)
	}
	req.ToolChoice = llmprovider.ForceTool("tool_search")
	if got := ForRequest(Responses, "gpt-6-luna", req); got != Append {
		t.Errorf("search forced on Responses: %v, want Append", got)
	}
	if got := ForRequest(Messages, "claude-haiku-5-5", req); got != MessagesNative {
		t.Errorf("search forced on Messages: %v, want MessagesNative", got)
	}
	req.ToolChoice = llmprovider.ForceTool("read_file")
	if got := ForRequest(Responses, "gpt-6-luna", req); got != ResponsesNative {
		t.Errorf("another tool forced: %v, want ResponsesNative", got)
	}
	plain := &llmprovider.Request{Tools: []llmprovider.Tool{{Name: "a"}}}
	if got := ForRequest(Responses, "gpt-6-luna", plain); got != Append {
		t.Errorf("nothing deferred: %v, want Append", got)
	}
}

// TestSearchLoads_0040: the loads a search result made, by call, for the
// Responses wire's tool_search_output.
func TestSearchLoads_0040(t *testing.T) {
	got := SearchLoads(&llmprovider.Request{Tools: deferralTools(), Input: history()})
	if !slices.Equal(got["s1"], []string{"lookup_stock", "get_weather"}) || !slices.Equal(got["s2"], []string{"get_weather"}) {
		t.Errorf("SearchLoads = %v", got)
	}
	if _, ok := got["r1"]; ok {
		t.Errorf("SearchLoads holds read_file's result, which is not the search's: %v", got)
	}
}
