package responses

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0040-MADR D3: a client tool_search_call decodes as a call to the request's
// Search tool, and replays as a tool_search_call; a search's result replays
// as a tool_search_output carrying the loaded definitions.

const searchReply = `{"id":"r1","status":"completed","output":[` +
	`{"type":"tool_search_call","execution":"client","call_id":"c1","status":"completed","arguments":{"query":"weather"}},` +
	`{"type":"function_call","call_id":"c2","name":"read_file","arguments":"{\"path\":\"a\"}"}]}`

// TestDecode_ToolSearchCall_0040: with the Search tool named, the call is a
// FunctionCallItem for it, its arguments the object as JSON text; without, it
// is dropped as before. A function_call's string arguments are unchanged.
func TestDecode_ToolSearchCall_0040(t *testing.T) {
	resp, err := Decoder{Search: "tool_search"}.DecodeFor("openai")(strings.NewReader(searchReply))
	if err != nil {
		t.Fatal(err)
	}
	want := []llmprovider.Item{
		llmprovider.FunctionCallItem{CallID: "c1", Name: "tool_search", Arguments: `{"query":"weather"}`},
		llmprovider.FunctionCallItem{CallID: "c2", Name: "read_file", Arguments: `{"path":"a"}`},
	}
	if len(resp.Output) != 2 || resp.Output[0] != want[0] || resp.Output[1] != want[1] {
		t.Errorf("with a search tool: %+v, want %+v", resp.Output, want)
	}
	resp, err = DecodeFor("openai")(strings.NewReader(searchReply))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Output) != 1 || resp.Output[0] != want[1] {
		t.Errorf("without a search tool: %+v, want only %+v", resp.Output, want[1])
	}
	// A hosted search (execution "server") is the service's own, not the
	// caller's: it is dropped even with a Search tool named (D8).
	hosted := strings.Replace(searchReply, `"execution":"client"`, `"execution":"server"`, 1)
	resp, err = Decoder{Search: "tool_search"}.DecodeFor("openai")(strings.NewReader(hosted))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Output) != 1 || resp.Output[0] != want[1] {
		t.Errorf("a hosted search: %+v, want only %+v", resp.Output, want[1])
	}
}

// TestEvents_ToolSearchCall_0040: the stream yields the same item.
func TestEvents_ToolSearchCall_0040(t *testing.T) {
	stream := "data: " + `{"type":"response.output_item.done","item":{"type":"tool_search_call","execution":"client","call_id":"c1","status":"completed","arguments":{"query":"weather"}}}` + "\n\n" +
		"data: " + `{"type":"response.completed","response":{"id":"r1","status":"completed"}}` + "\n\n"
	var items []llmprovider.Item
	err := Decoder{Search: "tool_search"}.Events("openai", strings.NewReader(stream), func(e llmprovider.Event) bool {
		if e.Type == llmprovider.EventItem {
			items = append(items, e.Item)
		}
		return true
	})
	if err != nil {
		t.Fatal(err)
	}
	want := llmprovider.FunctionCallItem{CallID: "c1", Name: "tool_search", Arguments: `{"query":"weather"}`}
	if len(items) != 1 || items[0] != want {
		t.Errorf("stream items %+v, want %+v", items, want)
	}
}

func deferralRequest() *llmprovider.Request {
	return &llmprovider.Request{
		Tools: []llmprovider.Tool{
			{Name: "read_file"},
			{Name: "tool_search", Search: true},
			{Name: "get_weather", Description: "Gets the weather.", Deferred: true},
			{Name: "lookup_stock", Deferred: true},
		},
		Input: []llmprovider.Item{
			llmprovider.MessageItem{Text: "hi"},
			llmprovider.FunctionCallItem{CallID: "c1", Name: "tool_search", Arguments: `{"query":"w"}`},
			llmprovider.FunctionCallOutputItem{CallID: "c1", Output: "Loaded get_weather."},
			llmprovider.ToolLoadItem{CallID: "c1", Name: "get_weather"},
			llmprovider.FunctionCallItem{CallID: "c2", Name: "get_weather", Arguments: `{"location":"Paris"}`},
			llmprovider.FunctionCallOutputItem{CallID: "c2", Output: "Sunny"},
		},
	}
}

// TestInputLoading_Replay_0040: the search call and its result, byte for
// byte; the result's text is not sent, and the loaded definition carries
// defer_loading.
func TestInputLoading_Replay_0040(t *testing.T) {
	req := deferralRequest()
	got, err := json.Marshal(InputLoading(req.Input, req))
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"content":"hi","role":"user"},` +
		`{"arguments":{"query":"w"},"call_id":"c1","execution":"client","status":"completed","type":"tool_search_call"},` +
		`{"call_id":"c1","execution":"client","status":"completed","tools":[` +
		`{"defer_loading":true,"description":"Gets the weather.","name":"get_weather","parameters":{"properties":{},"type":"object"},"type":"function"}],` +
		`"type":"tool_search_output"},` +
		`{"arguments":"{\"location\":\"Paris\"}","call_id":"c2","name":"get_weather","type":"function_call"},` +
		`{"call_id":"c2","output":"Sunny","type":"function_call_output"}]`
	if string(got) != want {
		t.Errorf("InputLoading:\n got  %s\n want %s", got, want)
	}
}

// TestInputLoading_Unchanged_0040: with nothing deferred, InputLoading is
// Input byte for byte, and Input sends no ToolLoadItem.
func TestInputLoading_Unchanged_0040(t *testing.T) {
	req := deferralRequest()
	for i := range req.Tools {
		req.Tools[i].Deferred, req.Tools[i].Search = false, false
	}
	plain := req.Input[:3]
	a, _ := json.Marshal(InputLoading(plain, &llmprovider.Request{Tools: req.Tools, Input: plain}))
	b, _ := json.Marshal(Input(plain))
	if string(a) != string(b) {
		t.Errorf("InputLoading %s\nInput        %s", a, b)
	}
	// Input knows nothing of deferral: the search is a plain function call,
	// and the ToolLoadItem is dropped, leaving five of the six items.
	full := deferralRequest().Input
	items := Input(full)
	withLoad, _ := json.Marshal(items)
	if len(items) != len(full)-1 || strings.Contains(string(withLoad), "tool_search_call") ||
		strings.Contains(string(withLoad), "tool_search_output") {
		t.Errorf("Input sent a deferral item: %s", withLoad)
	}
}

// TestInputLoading_BadArguments_0040: arguments that are not a JSON object
// replay as an empty object, which the API takes, rather than a string it
// refuses.
func TestInputLoading_BadArguments_0040(t *testing.T) {
	req := deferralRequest()
	req.Input[1] = llmprovider.FunctionCallItem{CallID: "c1", Name: "tool_search", Arguments: "not json"}
	got, _ := json.Marshal(InputLoading(req.Input, req))
	if !strings.Contains(string(got), `{"arguments":{},"call_id":"c1","execution":"client"`) {
		t.Errorf("bad arguments: %s", got)
	}
}
