package wire_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/chatcompletions"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/generatecontent"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/messages"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/responses"
)

// The item conversions of every shared format, side by side (MADR 0012 §2).
// Ported from llmprovider's item_fidelity_test.go (0015-PLAN S7b).

// roundTrip is the canonical tool round trip MADR 0012 §2 must survive.
func roundTrip() []llmprovider.Item {
	return []llmprovider.Item{
		llmprovider.MessageItem{Role: wire.RoleUser, Text: "weather?"},
		llmprovider.FunctionCallItem{CallID: "call_1", Name: "get_weather", Arguments: `{"city":"Paris"}`},
		llmprovider.FunctionCallOutputItem{CallID: "call_1", Output: `{"forecast":"sunny"}`},
	}
}

// twoCalls is a turn in which the model says something, then calls two tools.
func twoCalls() []llmprovider.Item {
	return []llmprovider.Item{
		llmprovider.MessageItem{Role: wire.RoleUser, Text: "weather in two cities?"},
		llmprovider.MessageItem{Role: wire.RoleAssistant, Text: "Checking both."},
		llmprovider.FunctionCallItem{CallID: "call_1", Name: "get_weather", Arguments: `{"city":"Paris"}`},
		llmprovider.FunctionCallItem{CallID: "call_2", Name: "get_weather", Arguments: `{"city":"Rome"}`},
		llmprovider.FunctionCallOutputItem{CallID: "call_1", Output: `{"forecast":"sunny"}`},
		llmprovider.FunctionCallOutputItem{CallID: "call_2", Output: `{"forecast":"rain"}`},
	}
}

// asJSON normalises a request fragment through JSON, as the wire sees it.
func asJSON(t *testing.T, v any) any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

func mustEqualJSON(t *testing.T, got any, want string) {
	t.Helper()
	var w any
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("bad want: %v", err)
	}
	if g := asJSON(t, got); !reflect.DeepEqual(g, w) {
		gj, _ := json.Marshal(g)
		t.Fatalf("got  %s\nwant %s", gj, want)
	}
}

func TestItemFidelity_ChatToolCall(t *testing.T) {
	mustEqualJSON(t, chatMessages(roundTrip()), `[
		{"role":"user","content":"weather?"},
		{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Paris\"}"}}]},
		{"role":"tool","tool_call_id":"call_1","content":"{\"forecast\":\"sunny\"}"}]`)
}

func TestItemFidelity_ChatGroupsCalls(t *testing.T) {
	mustEqualJSON(t, chatMessages(twoCalls()), `[
		{"role":"user","content":"weather in two cities?"},
		{"role":"assistant","content":"Checking both.","tool_calls":[
			{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Paris\"}"}},
			{"id":"call_2","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Rome\"}"}}]},
		{"role":"tool","tool_call_id":"call_1","content":"{\"forecast\":\"sunny\"}"},
		{"role":"tool","tool_call_id":"call_2","content":"{\"forecast\":\"rain\"}"}]`)
}

func TestItemFidelity_ResponsesToolCall(t *testing.T) {
	mustEqualJSON(t, responses.Input(roundTrip()), `[
		{"role":"user","content":"weather?"},
		{"type":"function_call","call_id":"call_1","name":"get_weather","arguments":"{\"city\":\"Paris\"}"},
		{"type":"function_call_output","call_id":"call_1","output":"{\"forecast\":\"sunny\"}"}]`)
}

func TestItemFidelity_AnthropicToolCall(t *testing.T) {
	mustEqualJSON(t, messages.FromItems(roundTrip()), `[
		{"role":"user","content":"weather?"},
		{"role":"assistant","content":[{"type":"tool_use","id":"call_1","name":"get_weather","input":{"city":"Paris"}}]},
		{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"{\"forecast\":\"sunny\"}"}]}]`)
}

func TestItemFidelity_AnthropicGroupsCallsAndResults(t *testing.T) {
	mustEqualJSON(t, messages.FromItems(twoCalls()), `[
		{"role":"user","content":"weather in two cities?"},
		{"role":"assistant","content":[
			{"type":"text","text":"Checking both."},
			{"type":"tool_use","id":"call_1","name":"get_weather","input":{"city":"Paris"}},
			{"type":"tool_use","id":"call_2","name":"get_weather","input":{"city":"Rome"}}]},
		{"role":"user","content":[
			{"type":"tool_result","tool_use_id":"call_1","content":"{\"forecast\":\"sunny\"}"},
			{"type":"tool_result","tool_use_id":"call_2","content":"{\"forecast\":\"rain\"}"}]}]`)
}

func TestItemFidelity_GeminiToolCall(t *testing.T) {
	mustEqualJSON(t, generatecontent.Contents(roundTrip()), `[
		{"role":"user","parts":[{"text":"weather?"}]},
		{"role":"model","parts":[{"functionCall":{"name":"get_weather","args":{"city":"Paris"}},"thoughtSignature":"skip_thought_signature_validator"}]},
		{"role":"user","parts":[{"functionResponse":{"name":"get_weather","response":{"output":"{\"forecast\":\"sunny\"}"}}}]}]`)
}

// TestItemFidelity_GeminiDecodesCallWithoutArgs: a call to a tool with no
// parameters has no args; it is still a call.
func TestItemFidelity_GeminiDecodesCallWithoutArgs(t *testing.T) {
	res, err := generatecontent.Decode(strings.NewReader(
		`{"candidates":[{"content":{"parts":[{"functionCall":{"name":"now"}}]}}]}`))
	if err != nil || len(res.Output) != 1 {
		t.Fatalf("decode = %+v/%v, want one function call", res, err)
	}
	if call, ok := res.Output[0].(llmprovider.FunctionCallItem); !ok || call.Name != "now" || call.Arguments != "{}" {
		t.Fatalf("item = %#v, want FunctionCallItem now with {}", res.Output[0])
	}
}

func TestItemFidelity_GeminiGroupsCallsAndResults(t *testing.T) {
	mustEqualJSON(t, generatecontent.Contents(twoCalls()), `[
		{"role":"user","parts":[{"text":"weather in two cities?"}]},
		{"role":"model","parts":[
			{"text":"Checking both."},
			{"functionCall":{"name":"get_weather","args":{"city":"Paris"}},"thoughtSignature":"skip_thought_signature_validator"},
			{"functionCall":{"name":"get_weather","args":{"city":"Rome"}},"thoughtSignature":"skip_thought_signature_validator"}]},
		{"role":"user","parts":[
			{"functionResponse":{"name":"get_weather","response":{"output":"{\"forecast\":\"sunny\"}"}}},
			{"functionResponse":{"name":"get_weather","response":{"output":"{\"forecast\":\"rain\"}"}}}]}]`)
}

// chatMessages is the messages Chat Completions sends for items.
func chatMessages(items []llmprovider.Item) any {
	return chatcompletions.Body("m", 1, items, chatcompletions.Opts{})["messages"]
}
