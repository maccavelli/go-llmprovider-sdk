package grok

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0040-MADR D5: Grok's deferred tools go through grok-build's use_tool
// wrapper. The tool list stays fixed, a result that loads tools carries their
// definitions as JSON, and the caller sees ordinary calls both ways.

// weatherSchema is get_weather's input schema.
var weatherSchema = map[string]any{"type": "object",
	"properties": map[string]any{"location": map[string]any{"type": "string"}}, "required": []any{"location"}}

// dispatchTools has two deferred tools behind a search.
func dispatchTools() []llmprovider.Tool {
	return []llmprovider.Tool{
		{Name: "read_file", Description: "Reads a file."},
		{Name: "tool_search", Description: "Finds tools.", Search: true},
		{Name: "get_weather", Description: "Current weather.", Schema: weatherSchema, Deferred: true},
		{Name: "lookup_stock", Description: "A stock price.", Deferred: true},
	}
}

// useToolArgs is the model's call to the wrapper, as the probe saw it
// (0040-MADR A4).
const useToolArgs = `{"tool_name":"get_weather","tool_input":{"location":"Paris"}}`

// callReply answers with one function call, by name and arguments.
func callReply(name, args string) string {
	quoted, _ := json.Marshal(args)
	return `{"id":"r","output":[{"type":"function_call","call_id":"c2","name":"` + name + `","arguments":` + string(quoted) + `}]}`
}

// callStream is callReply as a Responses event stream.
func callStream(name, args string) string {
	quoted, _ := json.Marshal(args)
	return `data: {"type":"response.output_item.done","item":{"type":"function_call","call_id":"c2","name":"` + name +
		`","arguments":` + string(quoted) + "}}\n\n" + `data: {"type":"response.completed","response":{"id":"r"}}` + "\n\n"
}

// toolNames is the names in a body's tools.
func toolNames(body map[string]any) []string {
	var out []string
	list, _ := body["tools"].([]any)
	for _, raw := range list {
		tool, _ := raw.(map[string]any)
		name, _ := tool["name"].(string)
		out = append(out, name)
	}
	return out
}

// TestGrokDispatch_Tools_0040: the tools sent are the ones not deferred, then
// use_tool, byte for byte the same before and after each load; the response
// says the tools were dispatched.
func TestGrokDispatch_Tools_0040(t *testing.T) {
	search := []llmprovider.Item{user("weather and a stock"),
		llmprovider.FunctionCallItem{CallID: "c1", Name: "tool_search", Arguments: `{"query":"weather"}`},
		llmprovider.FunctionCallOutputItem{CallID: "c1", Output: "Loaded."}}
	inputs := [][]llmprovider.Item{
		{user("weather and a stock")},
		append(slices.Clone(search), llmprovider.ToolLoadItem{CallID: "c1", Name: "get_weather"}),
		append(slices.Clone(search), llmprovider.ToolLoadItem{CallID: "c1", Name: "get_weather"},
			llmprovider.ToolLoadItem{CallID: "c1", Name: "lookup_stock"}),
	}
	var first string
	for i, input := range inputs {
		var body map[string]any
		srv := captureServer(t, &body, okResponse)
		p := build(t, apiKey(srv.URL, "grok-4.6")...)
		resp, err := p.Generate(context.Background(), &llmprovider.Request{Tools: dispatchTools(), Input: input})
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		if got, want := toolNames(body), []string{"read_file", "tool_search", "use_tool"}; !slices.Equal(got, want) {
			t.Errorf("request %d: tools %v, want %v", i, got, want)
		}
		raw, _ := json.Marshal(body["tools"])
		if i == 0 {
			first = string(raw)
		} else if string(raw) != first {
			t.Errorf("request %d: the tool list changed on a load:\n%s\nwant\n%s", i, raw, first)
		}
		var warned bool
		for _, w := range resp.Warnings {
			warned = warned || w.Kind == llmprovider.WarningDeferralEmulated && strings.Contains(w.Detail, "use_tool")
		}
		if !warned {
			t.Errorf("request %d: warnings %+v; want WarningDeferralEmulated naming use_tool", i, resp.Warnings)
		}
	}
	for _, want := range []string{`"name":"use_tool"`, `"tool_name":{`, `"tool_input":{`, `"required":["tool_name","tool_input"]`} {
		if !strings.Contains(first, want) {
			t.Errorf("tools %s; want %s", first, want)
		}
	}
}

// TestGrokDispatch_Decode_0040: a use_tool call naming a deferred tool is that
// tool's call, by Generate and by Stream; one naming anything else, or made
// in a request with nothing deferred, stays a use_tool call.
func TestGrokDispatch_Decode_0040(t *testing.T) {
	unknown := `{"tool_name":"send_fax","tool_input":{"to":"1"}}`
	stringInput := `{"tool_name":"get_weather","tool_input":"{\"location\":\"Paris\"}"}`
	badInput := `{"tool_name":"get_weather","tool_input":"Paris"}`
	numberInput := `{"tool_name":"get_weather","tool_input":5}`
	for _, tc := range []struct {
		name  string
		tools []llmprovider.Tool
		args  string
		want  llmprovider.FunctionCallItem
	}{
		{"a deferred tool", dispatchTools(), useToolArgs,
			llmprovider.FunctionCallItem{CallID: "c2", Name: "get_weather", Arguments: `{"location":"Paris"}`}},
		{"its input as JSON text", dispatchTools(), stringInput,
			llmprovider.FunctionCallItem{CallID: "c2", Name: "get_weather", Arguments: `{"location":"Paris"}`}},
		{"no input", dispatchTools(), `{"tool_name":"lookup_stock"}`,
			llmprovider.FunctionCallItem{CallID: "c2", Name: "lookup_stock", Arguments: `{}`}},
		{"an unknown tool", dispatchTools(), unknown,
			llmprovider.FunctionCallItem{CallID: "c2", Name: "use_tool", Arguments: unknown}},
		{"input that is not an object", dispatchTools(), badInput,
			llmprovider.FunctionCallItem{CallID: "c2", Name: "use_tool", Arguments: badInput}},
		{"input that is JSON but not an object", dispatchTools(), numberInput,
			llmprovider.FunctionCallItem{CallID: "c2", Name: "use_tool", Arguments: numberInput}},
		{"nothing deferred", []llmprovider.Tool{{Name: "use_tool"}, {Name: "get_weather"}}, useToolArgs,
			llmprovider.FunctionCallItem{CallID: "c2", Name: "use_tool", Arguments: useToolArgs}},
	} {
		req := func() *llmprovider.Request {
			return &llmprovider.Request{Tools: tc.tools, Input: []llmprovider.Item{user("weather in Paris")}}
		}
		srv := replyServer(t, callReply("use_tool", tc.args))
		resp, err := build(t, apiKey(srv.URL, "grok-4.6")...).Generate(context.Background(), req())
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if len(resp.Output) != 1 || resp.Output[0] != tc.want {
			t.Errorf("%s: Generate output %+v, want %+v", tc.name, resp.Output, tc.want)
		}
		streamed, _ := sseServer(t, callStream("use_tool", tc.args))
		var item llmprovider.Item
		var done *llmprovider.Response
		for ev, err := range llmprovider.Stream(context.Background(), build(t, apiKey(streamed.URL, "grok-4.6")...), req()) {
			if err != nil {
				t.Fatalf("%s: stream: %v", tc.name, err)
			}
			switch ev.Type {
			case llmprovider.EventItem:
				item = ev.Item
			case llmprovider.EventDone:
				done = ev.Response
			}
		}
		if item != tc.want || done == nil || len(done.Output) != 1 || done.Output[0] != tc.want {
			t.Errorf("%s: stream item %+v, done %+v; want %+v", tc.name, item, done, tc.want)
		}
	}
}

// TestGrokDispatch_Replay_0040: a deferred tool's call goes back as the
// use_tool call the model made, byte for byte, and a result that loads tools
// carries its text, then the loaded definitions as JSON.
func TestGrokDispatch_Replay_0040(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, okResponse)
	p := build(t, apiKey(srv.URL, "grok-4.6")...)
	req := &llmprovider.Request{Tools: dispatchTools(), Input: []llmprovider.Item{
		user("weather in Paris"),
		llmprovider.FunctionCallItem{CallID: "c1", Name: "tool_search", Arguments: `{"query":"weather"}`},
		llmprovider.FunctionCallOutputItem{CallID: "c1", Output: "Loaded get_weather."},
		llmprovider.ToolLoadItem{CallID: "c1", Name: "get_weather"},
		// What TestGrokDispatch_Decode_0040 decodes from useToolArgs.
		llmprovider.FunctionCallItem{CallID: "c2", Name: "get_weather", Arguments: `{"location":"Paris"}`},
		llmprovider.FunctionCallOutputItem{CallID: "c2", Output: "18C"},
	}}
	if _, err := p.Generate(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	loaded := "Loaded get_weather.\n\n" +
		`[{"name":"get_weather","description":"Current weather.","input_schema":` +
		`{"properties":{"location":{"type":"string"}},"required":["location"],"type":"object"}}]`
	quotedLoaded, _ := json.Marshal(loaded)
	quotedArgs, _ := json.Marshal(useToolArgs)
	want := `[{"content":"weather in Paris","role":"user"},` +
		`{"arguments":"{\"query\":\"weather\"}","call_id":"c1","name":"tool_search","type":"function_call"},` +
		`{"call_id":"c1","output":` + string(quotedLoaded) + `,"type":"function_call_output"},` +
		`{"arguments":` + string(quotedArgs) + `,"call_id":"c2","name":"use_tool","type":"function_call"},` +
		`{"call_id":"c2","output":"18C","type":"function_call_output"}]`
	got, _ := json.Marshal(body["input"])
	if string(got) != want {
		t.Errorf("input\n%s\nwant\n%s", got, want)
	}
}

// TestGrokDispatch_NameClash_0040: a tool of the caller's named use_tool, in a
// request with a deferred tool, is refused before anything is sent, by
// Generate and by Stream; with nothing deferred it is an ordinary tool.
func TestGrokDispatch_NameClash_0040(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, okResponse)
	p := build(t, apiKey(srv.URL, "grok-4.6")...)
	clash := func() *llmprovider.Request {
		return &llmprovider.Request{Tools: append(dispatchTools(), llmprovider.Tool{Name: "use_tool"}),
			Input: []llmprovider.Item{user("hi")}}
	}
	if _, err := p.Generate(context.Background(), clash()); !errors.Is(err, llmprovider.ErrInvalidRequest) ||
		!strings.Contains(fmt.Sprint(err), "use_tool") {
		t.Errorf("Generate: %v; want ErrInvalidRequest naming use_tool", err)
	}
	for _, err := range llmprovider.Stream(context.Background(), p, clash()) {
		if !errors.Is(err, llmprovider.ErrInvalidRequest) {
			t.Errorf("Stream: %v; want ErrInvalidRequest", err)
		}
		break
	}
	if body != nil {
		t.Errorf("a refused request was sent: %v", body)
	}
	plain := &llmprovider.Request{Tools: []llmprovider.Tool{{Name: "use_tool"}}, Input: []llmprovider.Item{user("hi")}}
	if _, err := p.Generate(context.Background(), plain); err != nil {
		t.Errorf("use_tool with nothing deferred: %v", err)
	}
}

// TestGrokDispatch_Forced_0040: tool_choice can name only a tool in the list,
// so a forced choice of a loaded deferred tool sends the request in the
// append form, where it is; a forced search keeps the wrapper.
func TestGrokDispatch_Forced_0040(t *testing.T) {
	input := []llmprovider.Item{user("weather"),
		llmprovider.FunctionCallItem{CallID: "c1", Name: "tool_search", Arguments: `{}`},
		llmprovider.FunctionCallOutputItem{CallID: "c1", Output: "Loaded."},
		llmprovider.ToolLoadItem{CallID: "c1", Name: "get_weather"}}
	for _, tc := range []struct {
		force string
		tools []string
	}{
		{"get_weather", []string{"read_file", "tool_search", "get_weather"}},
		{"tool_search", []string{"read_file", "tool_search", "use_tool"}},
	} {
		var body map[string]any
		srv := captureServer(t, &body, okResponse)
		req := &llmprovider.Request{Tools: dispatchTools(), Input: input, ToolChoice: llmprovider.ForceTool(tc.force)}
		if _, err := build(t, apiKey(srv.URL, "grok-4.6")...).Generate(context.Background(), req); err != nil {
			t.Fatalf("forcing %s: %v", tc.force, err)
		}
		if got := toolNames(body); !slices.Equal(got, tc.tools) {
			t.Errorf("forcing %s: tools %v, want %v", tc.force, got, tc.tools)
		}
		if choice, _ := json.Marshal(body["tool_choice"]); !strings.Contains(string(choice), `"name":"`+tc.force+`"`) {
			t.Errorf("forcing %s: tool_choice %s", tc.force, choice)
		}
	}
}
