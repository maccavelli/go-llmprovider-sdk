package gemini

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from llmprovider's gemini_items_test.go (0015-PLAN S7).

// TestGemini_GenerateItems_TextParts verifies a model_output step is decoded to MessageItem.
func TestGemini_GenerateItems_TextParts(t *testing.T) {
	srv, _ := interactionServer(t, `{
		"id": "v1_int_text",
		"status": "completed",
		"steps": [{"type": "model_output", "content": [{"type": "text", "text": "hello from gemini"}]}]
	}`)
	p := build(t, apiKey(srv.URL, "gemini-3.7-flash")...)
	resp, err := p.Generate(context.Background(), text("hi"))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.Output) != 1 {
		t.Fatalf("expected 1 output item, got %d", len(resp.Output))
	}
	m, ok := resp.Output[0].(llmprovider.MessageItem)
	if !ok || m.Text != "hello from gemini" {
		t.Errorf("expected MessageItem with 'hello from gemini', got %v", resp.Output[0])
	}
	if resp.OutputText() != "hello from gemini" {
		t.Errorf("OutputText() = %q, want %q", resp.OutputText(), "hello from gemini")
	}
	if resp.ID != "v1_int_text" {
		t.Errorf("Response.ID = %q, want v1_int_text", resp.ID)
	}
}

// TestGemini_GenerateItems_FunctionCallPart verifies a function_call step is
// decoded, and that GenerateToolCall, which replaces GenerateWithTool, returns
// its arguments.
func TestGemini_GenerateItems_FunctionCallPart(t *testing.T) {
	srv, _ := interactionServer(t, `{
		"id": "v1_int_call",
		"status": "requires_action",
		"steps": [{"type": "function_call", "id": "call_1", "name": "lookup", "arguments": {"query": "weather"}}]
	}`)
	p := build(t, apiKey(srv.URL, "gemini-3.7-flash")...)
	tool := llmprovider.Tool{Name: "lookup", Description: "Lookup info", Schema: map[string]any{"type": "object"}}
	resp, err := p.Generate(context.Background(), withTool(text("find"), tool))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.Output) != 1 {
		t.Fatalf("expected 1 output item, got %d", len(resp.Output))
	}
	fc, ok := resp.Output[0].(llmprovider.FunctionCallItem)
	if !ok || fc.CallID != "call_1" || fc.Name != "lookup" || fc.Arguments != `{"query":"weather"}` {
		t.Errorf("FunctionCallItem = %+v", resp.Output[0])
	}

	call, err := llmprovider.GenerateToolCall(context.Background(), p, &llmprovider.Request{
		Input: []llmprovider.Item{user("find")}, Tools: []llmprovider.Tool{tool}})
	if err != nil {
		t.Fatalf("GenerateToolCall: %v", err)
	}
	if call.Arguments != `{"query":"weather"}` {
		t.Errorf("GenerateToolCall arguments = %q, want %q", call.Arguments, `{"query":"weather"}`)
	}
}

// TestGemini_GenerateItems_InterleavedParts verifies thought and model_output steps are decoded.
func TestGemini_GenerateItems_InterleavedParts(t *testing.T) {
	srv, _ := interactionServer(t, `{
		"id": "v1_int_thought",
		"status": "completed",
		"steps": [
			{"type": "thought", "signature": "sig", "summary": [{"type": "text", "text": "pondering the question"}]},
			{"type": "model_output", "content": [{"type": "text", "text": "the solution is 42"}]}
		]
	}`)
	p := build(t, apiKey(srv.URL, "gemini-3.7-flash")...)
	resp, err := p.Generate(context.Background(), text("question"))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.Output) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Output))
	}
	if _, ok := resp.Output[0].(llmprovider.ReasoningItem); !ok {
		t.Errorf("output[0] should be ReasoningItem, got %T", resp.Output[0])
	}
	if _, ok := resp.Output[1].(llmprovider.MessageItem); !ok {
		t.Errorf("output[1] should be MessageItem, got %T", resp.Output[1])
	}
	if resp.OutputText() != "the solution is 42" {
		t.Errorf("OutputText() = %q, want %q", resp.OutputText(), "the solution is 42")
	}
}

// TestGemini_Continue verifies a stored provider sends previous_interaction_id.
func TestGemini_Continue(t *testing.T) {
	srv, c := interactionServer(t, `{"id":"interaction_2","status":"completed","steps":[`+
		`{"type":"model_output","content":[{"type":"text","text":"more text"}]}]}`)
	p := build(t, apiKey(srv.URL, "gemini-3.7-flash", WithStore(true))...)
	req := text("go on")
	req.PreviousResponseID = "interaction_1"
	resp, err := p.Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	_, body := c.last()
	if body["previous_interaction_id"] != "interaction_1" || body["store"] != true {
		t.Errorf("previous_interaction_id = %v, store = %v; want interaction_1, true", body["previous_interaction_id"], body["store"])
	}
	if resp.ID != "interaction_2" {
		t.Errorf("Response.ID = %q, want interaction_2", resp.ID)
	}
}

// TestGemini_Capabilities is what TestGeminiInterfaceSatisfaction asserted:
// text, tools, thinking, items and listing are all offered, and continuation
// with WithStore(true) (TestGemini_ContinueChainsWhenStored).
func TestGemini_Capabilities(t *testing.T) {
	p := build(t, apiKey("http://127.0.0.1:0", "gemini-3.7-flash")...)
	caps := p.Capabilities()
	if caps.Tools != llmprovider.Supported || caps.ForcedToolChoice != llmprovider.Supported ||
		caps.Reasoning != llmprovider.Supported || caps.NativeStreaming != llmprovider.Unsupported {
		t.Errorf("Capabilities = %+v", caps)
	}
	if _, ok := p.(llmprovider.ModelLister); !ok {
		t.Error("the provider is not a ModelLister")
	}
}

func TestGemini_GenerateItems_FunctionCallOutput(t *testing.T) {
	srv, _ := interactionServer(t, `{"id":"v1_int_out","status":"completed","steps":[`+
		`{"type":"model_output","content":[{"type":"text","text":"received output"}]}]}`)
	p := build(t, apiKey(srv.URL, "gemini-3.7-flash")...)
	resp, err := p.Generate(context.Background(), items(user("compute"),
		llmprovider.FunctionCallOutputItem{CallID: "fn1", Output: "{\"res\":1}"}))
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if resp.OutputText() != "received output" {
		t.Errorf("OutputText = %q, want 'received output'", resp.OutputText())
	}
}

func TestGemini_ErrorClassification(t *testing.T) {
	tests := []struct {
		status int
		target error
	}{
		{429, llmprovider.ErrRateLimited},
		{401, llmprovider.ErrAuthFailure},
		{403, llmprovider.ErrAuthFailure},
		{500, llmprovider.ErrProviderUnavailable},
		{400, llmprovider.ErrInvalidRequest},
	}
	for _, tc := range tests {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
		}))
		p := build(t, apiKey(srv.URL, "gemini-3.7-flash")...)
		_, err := p.Generate(context.Background(), text("hi"))
		srv.Close()
		if err == nil || !errors.Is(err, tc.target) {
			t.Errorf("HTTP %d: got %v, want %v", tc.status, err, tc.target)
		}
	}
}
