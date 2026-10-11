package claude

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// structuredReply0034 is a structured-outputs reply: thinking, then the JSON.
const structuredReply0034 = `{"stop_reason":"end_turn","content":[{"type":"thinking","thinking":"t","signature":"s"},` +
	`{"type":"text","text":"{\"city\":\"Paris\"}"}]}`

// structuredStream0034 is the same reply streamed.
const structuredStream0034 = "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"model\":\"claude-opus-5-5\",\"usage\":{\"input_tokens\":9,\"output_tokens\":1}}}\n\n" +
	"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
	"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"{\\\"city\\\":\"}}\n\n" +
	"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"\\\"Paris\\\"}\"}}\n\n" +
	"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
	"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":7}}\n\n" +
	"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

// structuredServer records each request body and answers with reply.
func structuredServer(t *testing.T, reply string) (*httptest.Server, func() []map[string]any) {
	t.Helper()
	var mu sync.Mutex
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		mu.Lock()
		bodies = append(bodies, body)
		mu.Unlock()
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	return srv, func() []map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return append([]map[string]any(nil), bodies...)
	}
}

// TestGenerateToolCall_StructuredOutputs_0034 (0034-MADR D8): on a refusing
// model, GenerateToolCall sends structured outputs and returns the call, under
// either policy, with no warning.
func TestGenerateToolCall_StructuredOutputs_0034(t *testing.T) {
	for _, opts := range [][]llmprovider.Option{nil, {llmprovider.WithForcedChoiceFallback(llmprovider.ForcedChoiceDegrade)}} {
		srv, bodies := structuredServer(t, structuredReply0034)
		p := build(t, apiKey(srv.URL, "claude-sonnet-5-5", opts...)...)
		call, err := llmprovider.GenerateToolCall(context.Background(), p, weatherRequest())
		if err != nil || call.Name != "get_weather" || call.Arguments != `{"city":"Paris"}` {
			t.Fatalf("GenerateToolCall = %+v, %v; want get_weather {\"city\":\"Paris\"}", call, err)
		}
		sent := bodies()
		if len(sent) != 1 {
			t.Fatalf("%d requests, want 1", len(sent))
		}
		cfg, _ := sent[0]["output_config"].(map[string]any)
		format, _ := cfg["format"].(map[string]any)
		choice, _ := sent[0]["tool_choice"].(map[string]any)
		if format["type"] != "json_schema" || choice["type"] != "none" {
			t.Errorf("sent output_config %v, tool_choice %v; want a json_schema format and none", cfg, choice)
		}
	}
	srv, _ := structuredServer(t, structuredReply0034)
	req := weatherRequest()
	req.ToolChoice = llmprovider.ForceTool("get_weather")
	resp, err := build(t, apiKey(srv.URL, "claude-sonnet-5-5")...).Generate(context.Background(), req)
	if err != nil || len(resp.Warnings) != 0 || resp.FinishReason != llmprovider.FinishToolCalls {
		t.Errorf("Generate = %+v, %v; want a tool_calls finish and no warning", resp, err)
	}
}

// TestStream_StructuredOutputs_0034 (0034-MADR D8): the stream yields no text
// delta for the JSON, then the call as an item and in the done Response.
func TestStream_StructuredOutputs_0034(t *testing.T) {
	srv, _ := sseServer(t, structuredStream0034)
	req := weatherRequest()
	req.ToolChoice = llmprovider.ForceTool("get_weather")
	var deltas int
	var item llmprovider.Item
	var done *llmprovider.Response
	for ev, err := range build(t, apiKey(srv.URL, "claude-opus-5-5")...).(llmprovider.Streamer).Stream(context.Background(), req) {
		if err != nil {
			t.Fatalf("Stream: %v", err)
		}
		switch ev.Type {
		case llmprovider.EventTextDelta:
			deltas++
		case llmprovider.EventItem:
			item = ev.Item
		case llmprovider.EventDone:
			done = ev.Response
		}
	}
	call, ok := item.(llmprovider.FunctionCallItem)
	if deltas != 0 || !ok || call.Arguments != `{"city":"Paris"}` || done == nil || len(done.Output) != 1 || done.Output[0] != item {
		t.Errorf("deltas %d, item %+v, done %+v; want no delta and the call in both", deltas, item, done)
	}
}
