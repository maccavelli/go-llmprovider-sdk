package claude

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

// refusal0034 is Anthropic's refusal of a forced tool choice, as 0034-PLAN
// Phase 1 measured it (M2).
const refusal0034 = `{"type":"error","error":{"type":"invalid_request_error","message":"tool_choice: type \"tool\" and \"any\" are not supported for this model."}}`

// callReply0034 is a reply calling get_weather.
const callReply0034 = `{"stop_reason":"tool_use","content":[{"type":"tool_use","id":"toolu_1","name":"get_weather","input":{"city":"Paris"}}]}`

// refusingServer refuses a forced tool_choice (or, with always, every
// request) with refusal0034, answers any other with reply, and records each
// request's tool_choice and last message role.
type refusingServer struct {
	mu      sync.Mutex
	choices []string
	lastMsg []string
}

func newRefusingServer(t *testing.T, reply string, always bool, status int, errBody string) (*httptest.Server, *refusingServer) {
	t.Helper()
	rs := &refusingServer{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body struct {
			ToolChoice map[string]any `json:"tool_choice"`
			Messages   []struct {
				Role string `json:"role"`
			} `json:"messages"`
		}
		_ = json.Unmarshal(raw, &body)
		choice, _ := body.ToolChoice["type"].(string)
		rs.mu.Lock()
		rs.choices = append(rs.choices, choice)
		rs.lastMsg = append(rs.lastMsg, body.Messages[len(body.Messages)-1].Role)
		rs.mu.Unlock()
		if always || choice == "tool" || choice == "any" {
			w.WriteHeader(status)
			_, _ = io.WriteString(w, errBody)
			return
		}
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	return srv, rs
}

func (rs *refusingServer) seen() ([]string, []string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return append([]string(nil), rs.choices...), append([]string(nil), rs.lastMsg...)
}

func weatherRequest() *llmprovider.Request {
	req := text("What is the weather in Paris?")
	req.Tools = []llmprovider.Tool{{Name: "get_weather", Schema: map[string]any{"type": "object"}}}
	return req
}

// TestGenerate_LearnsARefusal_0034 (0034-MADR D3): a model outside the table
// that refuses a forced choice is sent once more, degraded, and remembered,
// so its next forced request is degraded at once.
func TestGenerate_LearnsARefusal_0034(t *testing.T) {
	srv, rs := newRefusingServer(t, callReply0034, false, http.StatusBadRequest, refusal0034)
	p := build(t, apiKey(srv.URL, "claude-future-learns")...)
	call, err := llmprovider.GenerateToolCall(context.Background(), p, weatherRequest())
	if err != nil || call.Name != "get_weather" {
		t.Fatalf("GenerateToolCall = %+v, %v; want the call", call, err)
	}
	choices, last := rs.seen()
	if strings.Join(choices, ",") != "tool,none" || last[1] != "user" {
		t.Errorf("first call sent tool_choice %v, last roles %v; want tool, then none: the resend is structured outputs (D8)", choices, last)
	}
	if _, err := llmprovider.GenerateToolCall(context.Background(), p, weatherRequest()); err != nil {
		t.Fatalf("second GenerateToolCall: %v", err)
	}
	if choices, _ := rs.seen(); strings.Join(choices, ",") != "tool,none,none" {
		t.Errorf("tool_choice %v; want the second request degraded at once", choices)
	}
}

// TestGenerate_OtherBadRequestNotResent_0034 (0034-MADR D3): a 400 that is
// not the refusal is returned, with one request sent.
func TestGenerate_OtherBadRequestNotResent_0034(t *testing.T) {
	srv, rs := newRefusingServer(t, callReply0034, true, http.StatusBadRequest,
		`{"type":"error","error":{"type":"invalid_request_error","message":"max_tokens: 999999 > 128000"}}`)
	_, err := llmprovider.GenerateToolCall(context.Background(), build(t, apiKey(srv.URL, "claude-future-other")...), weatherRequest())
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("err = %v, want ErrInvalidRequest", err)
	}
	if choices, _ := rs.seen(); len(choices) != 1 {
		t.Errorf("%d requests, want 1", len(choices))
	}
}

// TestGenerate_RefusedResendReturned_0034 (0034-MADR D3): a refusal of the
// degraded resend is returned; there is no third request.
func TestGenerate_RefusedResendReturned_0034(t *testing.T) {
	srv, rs := newRefusingServer(t, callReply0034, true, http.StatusBadRequest, refusal0034)
	_, err := llmprovider.GenerateToolCall(context.Background(), build(t, apiKey(srv.URL, "claude-future-twice")...), weatherRequest())
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("err = %v, want ErrInvalidRequest", err)
	}
	if choices, _ := rs.seen(); strings.Join(choices, ",") != "tool,none" {
		t.Errorf("tool_choice %v, want tool then none, and no third request", choices)
	}
}

// TestStream_LearnsARefusal_0034 (0034-MADR D3): a stream refused for forcing
// before its first event is opened once more, degraded.
func TestStream_LearnsARefusal_0034(t *testing.T) {
	srv, rs := newRefusingServer(t, messagesStream, false, http.StatusBadRequest, refusal0034)
	p := build(t, apiKey(srv.URL, "claude-future-streams")...)
	req := weatherRequest()
	req.ToolChoice = llmprovider.ForceTool("get_weather")
	var done bool
	for ev, err := range p.(llmprovider.Streamer).Stream(context.Background(), req) {
		if err != nil {
			t.Fatalf("Stream: %v", err)
		}
		done = done || ev.Type == llmprovider.EventDone
	}
	if choices, _ := rs.seen(); strings.Join(choices, ",") != "tool,none" || !done {
		t.Errorf("tool_choice %v, done %v; want tool then none, and EventDone", choices, done)
	}
}
