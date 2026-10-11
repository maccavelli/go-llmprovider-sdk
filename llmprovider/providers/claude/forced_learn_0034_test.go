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

// warningKinds is the kinds of ws, in order.
func warningKinds(ws []llmprovider.Warning) string {
	kinds := make([]string, len(ws))
	for i, w := range ws {
		kinds[i] = string(w.Kind)
	}
	return strings.Join(kinds, ",")
}

// TestGenerate_LearnedWarning_0034 (0034-MADR D3, D7): the response to a
// request that taught a refusal carries a WarningForcedChoiceLearned naming
// the model; the next request, prevented, carries none.
func TestGenerate_LearnedWarning_0034(t *testing.T) {
	srv, _ := newRefusingServer(t, `{"stop_reason":"end_turn","content":[{"type":"text","text":"{\"city\":\"Paris\"}"}]}`,
		false, http.StatusBadRequest, refusal0034)
	p := build(t, apiKey(srv.URL, "claude-future-warns")...)
	req := weatherRequest()
	req.ToolChoice = llmprovider.ForceTool("get_weather")
	resp, err := p.Generate(context.Background(), req)
	if err != nil || len(resp.Warnings) != 1 || resp.Warnings[0].Kind != llmprovider.WarningForcedChoiceLearned ||
		resp.Warnings[0].Model != "claude-future-warns" {
		t.Fatalf("Generate = %+v, %v; want one learned warning naming claude-future-warns", resp, err)
	}
	if resp, err := p.Generate(context.Background(), req); err != nil || len(resp.Warnings) != 0 {
		t.Errorf("second Generate = %+v, %v; want no warning", resp, err)
	}
}

// TestGenerate_LearnedRequiredRefuses_0034 (0034-MADR D3, D10): by default, a
// ToolChoiceRequired a model no layer knows refuses is learned, and returned
// as an error matching ErrUnsupported and the *APIError, with no resend; the
// next one fails before the network.
func TestGenerate_LearnedRequiredRefuses_0034(t *testing.T) {
	srv, rs := newRefusingServer(t, callReply0034, false, http.StatusBadRequest, refusal0034)
	p := build(t, apiKey(srv.URL, "claude-future-required")...)
	req := weatherRequest()
	req.ToolChoice = llmprovider.ToolChoiceRequired
	_, err := p.Generate(context.Background(), req)
	if !errors.Is(err, llmprovider.ErrUnsupported) || !errors.As(err, new(*llmprovider.APIError)) {
		t.Errorf("err = %v, want ErrUnsupported wrapping the *APIError", err)
	}
	if _, err := p.Generate(context.Background(), req); !errors.Is(err, llmprovider.ErrUnsupported) {
		t.Errorf("second err = %v, want ErrUnsupported", err)
	}
	if choices, _ := rs.seen(); strings.Join(choices, ",") != "any" {
		t.Errorf("tool_choice %v, want one request: no resend, and none for the second", choices)
	}
}

// TestGenerate_LearnedRequiredDegrades_0034 (0034-MADR D3, D10): under
// ForcedChoiceDegrade, it is resent once degraded, warned of as learned and
// as degraded.
func TestGenerate_LearnedRequiredDegrades_0034(t *testing.T) {
	srv, rs := newRefusingServer(t, callReply0034, false, http.StatusBadRequest, refusal0034)
	p := build(t, append(apiKey(srv.URL, "claude-future-degrades"), llmprovider.WithForcedChoiceFallback(llmprovider.ForcedChoiceDegrade))...)
	req := weatherRequest()
	req.ToolChoice = llmprovider.ToolChoiceRequired
	resp, err := p.Generate(context.Background(), req)
	if err != nil || warningKinds(resp.Warnings) != "forced_tool_choice_learned,forced_tool_choice_degraded" {
		t.Fatalf("Generate = %+v, %v; want the learned and the degraded warnings", resp, err)
	}
	if choices, _ := rs.seen(); strings.Join(choices, ",") != "any,auto" {
		t.Errorf("tool_choice %v, want any, then auto", choices)
	}
}

// TestStream_LearnedWarning_0034 (0034-MADR D3, D7): a stream that taught a
// refusal carries the learned warning on its done event.
func TestStream_LearnedWarning_0034(t *testing.T) {
	srv, _ := newRefusingServer(t, messagesStream, false, http.StatusBadRequest, refusal0034)
	p := build(t, apiKey(srv.URL, "claude-future-streams-warn")...)
	req := weatherRequest()
	req.ToolChoice = llmprovider.ForceTool("get_weather")
	var kinds string
	for ev, err := range p.(llmprovider.Streamer).Stream(context.Background(), req) {
		if err != nil {
			t.Fatalf("Stream: %v", err)
		}
		if ev.Type == llmprovider.EventDone {
			kinds = warningKinds(ev.Response.Warnings)
		}
	}
	if kinds != "forced_tool_choice_learned" {
		t.Errorf("done warnings %q, want the learned warning", kinds)
	}
}
