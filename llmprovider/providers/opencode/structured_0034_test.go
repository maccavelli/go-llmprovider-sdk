package opencode

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestGenerateToolCall_MessagesRouteStructured_0034 (0034-MADR D8): on Zen's
// messages route, GenerateToolCall on a refusing model sends structured
// outputs and returns the call.
func TestGenerateToolCall_MessagesRouteStructured_0034(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"stop_reason":"end_turn","content":[{"type":"text","text":"{\"city\":\"Paris\"}"}]}`)
	req := weatherRequest0034()
	req.Tools[0].Schema = map[string]any{"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}},
		"required": []any{"city"}, "additionalProperties": false}
	call, err := llmprovider.GenerateToolCall(context.Background(),
		build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "claude-opus-5-5")...), req)
	if err != nil || call.Name != "get_weather" || call.Arguments != `{"city":"Paris"}` {
		t.Fatalf("GenerateToolCall = %+v, %v; want get_weather {\"city\":\"Paris\"}", call, err)
	}
	if got := fmt.Sprint(body["tool_choice"], " ", body["output_config"]); got !=
		"map[type:none] map[format:map[schema:map[additionalProperties:false properties:map[city:map[type:string]] required:[city] type:object] type:json_schema]]" {
		t.Errorf("sent %s, want tool_choice none and the tool's schema as the format", got)
	}
}

// TestMessagesRoute_FormatClosed_0034 (0034-MADR Q9): a schema that never says
// additionalProperties is sent closed in the format, and as it was in tools.
func TestMessagesRoute_FormatClosed_0034(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"stop_reason":"end_turn","content":[{"type":"text","text":"{\"city\":\"Paris\"}"}]}`)
	req := weatherRequest0034()
	req.Tools[0].Schema = map[string]any{"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}}
	if _, err := llmprovider.GenerateToolCall(context.Background(),
		build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "claude-opus-5-5")...), req); err != nil {
		t.Fatal(err)
	}
	format := fmt.Sprint(body["output_config"])
	tools := fmt.Sprint(body["tools"])
	if !strings.Contains(format, "additionalProperties:false") || strings.Contains(tools, "additionalProperties") {
		t.Errorf("format %s, tools %s; want the format closed and the tools as given", format, tools)
	}
}
