package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// 0040-MADR D3 and D6 through the provider: GPT-5.4 and later with a key,
// and every model on a ChatGPT session, send a client tool_search and replay
// its call and output; an older model appends its loads.

func deferralRequest() *llmprovider.Request {
	return &llmprovider.Request{
		Tools: []llmprovider.Tool{
			{Name: "read_file"},
			{Name: "tool_search", Description: "Finds tools.", Search: true},
			{Name: "get_weather", Deferred: true},
			{Name: "lookup_stock", Deferred: true},
		},
		Input: []llmprovider.Item{
			llmprovider.MessageItem{Text: "weather in Paris"},
			llmprovider.FunctionCallItem{CallID: "c1", Name: "tool_search", Arguments: `{"query":"weather"}`},
			llmprovider.FunctionCallOutputItem{CallID: "c1", Output: "Loaded get_weather."},
			llmprovider.ToolLoadItem{CallID: "c1", Name: "get_weather"},
		},
	}
}

const searchCallReply = `{"id":"r2","status":"completed","output":[` +
	`{"type":"tool_search_call","execution":"client","call_id":"c9","status":"completed","arguments":{"query":"stock"}}]}`

// textReply is a completed answer with text.
const textReply = `{"id":"r","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`

// toolTypes is each sent tool's name, or its type when it has none.
func toolTypes(body map[string]any) []string {
	var out []string
	for _, raw := range body["tools"].([]any) {
		tool := raw.(map[string]any)
		if name, ok := tool["name"].(string); ok {
			out = append(out, name)
		} else {
			out = append(out, tool["type"].(string))
		}
	}
	return out
}

// inputTypes is each input item's type, or "message".
func inputTypes(body map[string]any) []string {
	var out []string
	for _, raw := range body["input"].([]any) {
		item := raw.(map[string]any)
		if typ, ok := item["type"].(string); ok {
			out = append(out, typ)
		} else {
			out = append(out, "message")
		}
	}
	return out
}

func equal(a, b []string) bool { return strings.Join(a, ",") == strings.Join(b, ",") }

// readBody reads a captured request's body.
func readBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}

// TestDeferral_KeyNative_0040: gpt-6-luna sends read_file and a client
// tool_search, no deferred tool at the top level, the search's call and
// output, and decodes a tool_search_call as a call to the Search tool.
func TestDeferral_KeyNative_0040(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, searchCallReply)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("gpt-6-luna"))...)
	resp, err := p.Generate(context.Background(), deferralRequest())
	if err != nil {
		t.Fatal(err)
	}
	if got := toolTypes(body); !equal(got, []string{"read_file", "tool_search"}) {
		t.Errorf("tools %v; want read_file and a tool_search entry", got)
	}
	if got := inputTypes(body); !equal(got, []string{"message", "tool_search_call", "tool_search_output"}) {
		t.Errorf("input %v; want the message, the search call and its output", got)
	}
	want := llmprovider.FunctionCallItem{CallID: "c9", Name: "tool_search", Arguments: `{"query":"stock"}`}
	if len(resp.Output) != 1 || resp.Output[0] != want {
		t.Errorf("output %+v, want %+v", resp.Output, want)
	}
}

// TestDeferral_KeyAppend_0040: gpt-4.1-mini, which has no tool search,
// appends the loaded tool and replays plain function items.
func TestDeferral_KeyAppend_0040(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, textReply)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("gpt-4.1-mini"))...)
	if _, err := p.Generate(context.Background(), deferralRequest()); err != nil {
		t.Fatal(err)
	}
	if got := toolTypes(body); !equal(got, []string{"read_file", "tool_search", "get_weather"}) {
		t.Errorf("tools %v; want read_file, tool_search and the loaded get_weather", got)
	}
	if got := inputTypes(body); !equal(got, []string{"message", "function_call", "function_call_output"}) {
		t.Errorf("input %v; want plain function items", got)
	}
}

// TestDeferral_KeyStream_0040: Stream decodes the search call too.
func TestDeferral_KeyStream_0040(t *testing.T) {
	stream := "data: " + `{"type":"response.output_item.done","item":{"type":"tool_search_call","execution":"client","call_id":"c9","status":"completed","arguments":{"query":"stock"}}}` + "\n\n" +
		"data: " + `{"type":"response.completed","response":{"id":"r","status":"completed"}}` + "\n\n"
	var body map[string]any
	srv := captureServer(t, &body, stream)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("gpt-6-luna"))...)
	var items []llmprovider.Item
	for event, err := range llmprovider.Stream(context.Background(), p, deferralRequest()) {
		if err != nil {
			t.Fatal(err)
		}
		if event.Type == llmprovider.EventItem {
			items = append(items, event.Item)
		}
	}
	want := llmprovider.FunctionCallItem{CallID: "c9", Name: "tool_search", Arguments: `{"query":"stock"}`}
	if len(items) != 1 || items[0] != want {
		t.Errorf("streamed %+v, want %+v", items, want)
	}
}

// TestDeferral_ChatGPT_0040: a ChatGPT session defers natively on a model
// the key path would not (gpt-5), and decodes the streamed search call.
func TestDeferral_ChatGPT_0040(t *testing.T) {
	stream := "data: " + `{"type":"response.output_item.done","item":{"type":"tool_search_call","execution":"client","call_id":"c9","status":"completed","arguments":{"query":"stock"}}}` + "\n\n" +
		"data: " + `{"type":"response.completed","response":{"id":"r","status":"completed"}}` + "\n\n"
	var sent map[string]any
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		raw, err := readBody(request)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &sent); err != nil {
			return nil, err
		}
		return httpResponse(request, http.StatusOK, stream), nil
	})}
	session := &auth.OAuthSession{Issuer: auth.DefaultOpenAIIssuer, Access: "sess", Refresh: "refresh",
		Expiry: time.Now().Add(time.Hour)}
	p := sessionProvider(t, session, "gpt-5", llmprovider.WithHTTPClient(client))
	resp, err := p.Generate(context.Background(), deferralRequest())
	if err != nil {
		t.Fatal(err)
	}
	if got := toolTypes(sent); !equal(got, []string{"read_file", "tool_search"}) {
		t.Errorf("tools %v; want read_file and a tool_search entry", got)
	}
	want := llmprovider.FunctionCallItem{CallID: "c9", Name: "tool_search", Arguments: `{"query":"stock"}`}
	if len(resp.Output) != 1 || resp.Output[0] != want {
		t.Errorf("output %+v, want %+v", resp.Output, want)
	}
}

// TestDeferral_ForcedSearch_0040: forcing the Search tool appends, so the
// choice can name it as a function.
func TestDeferral_ForcedSearch_0040(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, textReply)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("gpt-6-luna"))...)
	req := deferralRequest()
	req.ToolChoice = llmprovider.ForceTool("tool_search")
	if _, err := p.Generate(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if got := toolTypes(body); !equal(got, []string{"read_file", "tool_search", "get_weather"}) {
		t.Errorf("tools %v; want the append form", got)
	}
	if choice, _ := body["tool_choice"].(map[string]any); choice["name"] != "tool_search" {
		t.Errorf("tool_choice %v; want tool_search forced", body["tool_choice"])
	}
}
