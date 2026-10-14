package responses

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0040-MADR D5: under dispatch, a call to a deferred tool replays as a call to
// use_tool, a result that loads tools carries their definitions as JSON, and
// a use_tool call naming a deferred tool decodes as that tool's call.

func dispatchRequest() *llmprovider.Request {
	return &llmprovider.Request{Tools: []llmprovider.Tool{
		{Name: "read_file"},
		{Name: "tool_search", Search: true},
		{Name: "get_weather", Description: "Current weather.", Deferred: true,
			Schema: map[string]any{"type": "object", "properties": map[string]any{"location": map[string]any{"type": "string"}}}},
		{Name: "lookup_stock", Deferred: true},
	}}
}

// TestInputDispatch_0040: the calls and results, byte for byte.
func TestInputDispatch_0040(t *testing.T) {
	req := dispatchRequest()
	items := []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"},
		llmprovider.FunctionCallItem{CallID: "s1", Name: "tool_search", Arguments: `{"q":"w"}`},
		// A result with no text carries the definitions alone; a repeated
		// load is sent once.
		llmprovider.FunctionCallOutputItem{CallID: "s1"},
		llmprovider.ToolLoadItem{CallID: "s1", Name: "get_weather"},
		llmprovider.ToolLoadItem{CallID: "s1", Name: "lookup_stock"},
		llmprovider.ToolLoadItem{CallID: "s1", Name: "get_weather"},
		llmprovider.FunctionCallItem{CallID: "c2", Name: "get_weather", Arguments: ` {"location": "Paris"} `},
		// Arguments that are not an object go as {}.
		llmprovider.FunctionCallItem{CallID: "c3", Name: "lookup_stock", Arguments: `"AAPL"`},
		llmprovider.FunctionCallItem{CallID: "c4", Name: "read_file", Arguments: `{"path":"a"}`},
		llmprovider.FunctionCallOutputItem{CallID: "c4", Output: "text"},
	}
	got, err := json.Marshal(InputDispatch(items, req))
	if err != nil {
		t.Fatal(err)
	}
	defs := `[{"name":"get_weather","description":"Current weather.","input_schema":{"properties":{"location":{"type":"string"}},"type":"object"}},` +
		`{"name":"lookup_stock","description":"","input_schema":{"properties":{},"type":"object"}}]`
	quoted, _ := json.Marshal(defs)
	// InputItem's fields are in 0028-MADR D-A6's order; tool_input is
	// compacted.
	want := `[{"content":"hi","role":"user"},` +
		`{"arguments":"{\"q\":\"w\"}","call_id":"s1","name":"tool_search","type":"function_call"},` +
		`{"call_id":"s1","output":` + string(quoted) + `,"type":"function_call_output"},` +
		`{"arguments":"{\"tool_name\":\"get_weather\",\"tool_input\":{\"location\":\"Paris\"}}","call_id":"c2","name":"use_tool","type":"function_call"},` +
		`{"arguments":"{\"tool_name\":\"lookup_stock\",\"tool_input\":{}}","call_id":"c3","name":"use_tool","type":"function_call"},` +
		`{"arguments":"{\"path\":\"a\"}","call_id":"c4","name":"read_file","type":"function_call"},` +
		`{"call_id":"c4","output":"text","type":"function_call_output"}]`
	if string(got) != want {
		t.Errorf("InputDispatch\n%s\nwant\n%s", got, want)
	}
}

// TestDecode_Dispatch_0040: the use_tool calls a reply may hold, by DecodeFor
// and by Events; with no Dispatch set, every use_tool call stays one.
func TestDecode_Dispatch_0040(t *testing.T) {
	d := Decoder{Dispatch: map[string]bool{"get_weather": true, "lookup_stock": true}}
	for _, tc := range []struct {
		name, args, wantName, wantArgs string
	}{
		{"an object", `{"tool_name":"get_weather","tool_input":{ "location" : "Paris" }}`, "get_weather", `{"location":"Paris"}`},
		{"JSON text", `{"tool_name":"get_weather","tool_input":"{\"location\":\"Paris\"}"}`, "get_weather", `{"location":"Paris"}`},
		{"null input", `{"tool_name":"lookup_stock","tool_input":null}`, "lookup_stock", `{}`},
		{"an unknown tool", `{"tool_name":"send_fax","tool_input":{}}`, "use_tool", `{"tool_name":"send_fax","tool_input":{}}`},
		{"an array", `{"tool_name":"get_weather","tool_input":[1]}`, "use_tool", `{"tool_name":"get_weather","tool_input":[1]}`},
		{"not JSON", `not json`, "use_tool", `not json`},
	} {
		quoted, _ := json.Marshal(tc.args)
		item := `{"type":"function_call","call_id":"c1","name":"use_tool","arguments":` + string(quoted) + `}`
		want := llmprovider.FunctionCallItem{CallID: "c1", Name: tc.wantName, Arguments: tc.wantArgs}
		resp, err := d.DecodeFor("grok")(strings.NewReader(`{"id":"r","output":[` + item + `]}`))
		if err != nil || len(resp.Output) != 1 || resp.Output[0] != want {
			t.Errorf("%s: DecodeFor %+v, %v; want %+v", tc.name, resp, err, want)
		}
		stream := "data: " + `{"type":"response.output_item.done","item":` + item + "}\n\n" +
			"data: " + `{"type":"response.completed","response":{"id":"r"}}` + "\n\n"
		var got llmprovider.Item
		if err := d.Events("grok", strings.NewReader(stream), func(e llmprovider.Event) bool {
			if e.Type == llmprovider.EventItem {
				got = e.Item
			}
			return true
		}); err != nil || got != want {
			t.Errorf("%s: Events %+v, %v; want %+v", tc.name, got, err, want)
		}
	}
	plain := `{"id":"r","output":[{"type":"function_call","call_id":"c1","name":"use_tool","arguments":"{\"tool_name\":\"get_weather\"}"}]}`
	resp, err := Decoder{}.DecodeFor("grok")(strings.NewReader(plain))
	if err != nil || resp.Output[0].(llmprovider.FunctionCallItem).Name != "use_tool" {
		t.Errorf("no Dispatch: %+v, %v; want the use_tool call", resp, err)
	}
}
