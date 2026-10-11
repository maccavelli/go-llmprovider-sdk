package opencode

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

const (
	refusal0034   = `{"type":"error","error":{"type":"invalid_request_error","message":"tool_choice: type \"tool\" and \"any\" are not supported for this model."}}`
	callReply0034 = `{"stop_reason":"tool_use","content":[{"type":"tool_use","id":"toolu_1","name":"get_weather","input":{}}]}`
)

// refusingServer answers a forced tool_choice with errBody (or, with always,
// every request) and any other with reply, and records each tool_choice type.
func refusingServer(t *testing.T, reply, errBody string, always bool) (*httptest.Server, func() string) {
	t.Helper()
	var mu sync.Mutex
	var choices []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body struct {
			ToolChoice map[string]any `json:"tool_choice"`
		}
		_ = json.Unmarshal(raw, &body)
		choice, _ := body.ToolChoice["type"].(string)
		mu.Lock()
		choices = append(choices, choice)
		mu.Unlock()
		if always || choice == "tool" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, errBody)
			return
		}
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	return srv, func() string {
		mu.Lock()
		defer mu.Unlock()
		return strings.Join(choices, ",")
	}
}

func weatherRequest0034() *llmprovider.Request {
	req := text("hi")
	req.Tools = []llmprovider.Tool{{Name: "get_weather", Schema: map[string]any{"type": "object"}}}
	return req
}

// TestGenerate_MessagesRouteLearnsARefusal_0034 (0034-MADR D3): on Zen's
// messages route, an unlisted claude- model that refuses a forced choice is
// sent once more, degraded, and remembered.
func TestGenerate_MessagesRouteLearnsARefusal_0034(t *testing.T) {
	srv, seen := refusingServer(t, callReply0034, refusal0034, false)
	p := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "claude-future-zen")...)
	for range 2 {
		if _, err := llmprovider.GenerateToolCall(context.Background(), p, weatherRequest0034()); err != nil {
			t.Fatalf("GenerateToolCall: %v", err)
		}
	}
	if got := seen(); got != "tool,none,none" {
		t.Errorf("tool_choice %s, want tool, none, then none at once: structured outputs (D8)", got)
	}
}

// TestGenerate_MessagesRouteOtherBadRequest_0034 (0034-MADR D3): a 400 that is
// not the refusal is returned, with one request sent.
func TestGenerate_MessagesRouteOtherBadRequest_0034(t *testing.T) {
	srv, seen := refusingServer(t, callReply0034,
		`{"type":"error","error":{"type":"invalid_request_error","message":"max_tokens: 999999 > 128000"}}`, true)
	_, err := llmprovider.GenerateToolCall(context.Background(),
		build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "claude-future-zen-other")...), weatherRequest0034())
	if !errors.Is(err, llmprovider.ErrInvalidRequest) || seen() != "tool" {
		t.Errorf("err %v, tool_choice %s; want ErrInvalidRequest after one request", err, seen())
	}
}

// TestStream_MessagesRouteLearnsARefusal_0034 (0034-MADR D3): a stream on
// Zen's messages route refused for forcing is opened once more, degraded.
func TestStream_MessagesRouteLearnsARefusal_0034(t *testing.T) {
	srv, seen := refusingServer(t, llmtestMessagesStream, refusal0034, false)
	p := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "claude-future-zen-streams")...)
	req := weatherRequest0034()
	req.ToolChoice = llmprovider.ForceTool("get_weather")
	var done bool
	for ev, err := range p.(llmprovider.Streamer).Stream(context.Background(), req) {
		if err != nil {
			t.Fatalf("Stream: %v", err)
		}
		done = done || ev.Type == llmprovider.EventDone
	}
	if got := seen(); got != "tool,none" || !done {
		t.Errorf("tool_choice %s, done %v; want tool then none, and EventDone", got, done)
	}
}
