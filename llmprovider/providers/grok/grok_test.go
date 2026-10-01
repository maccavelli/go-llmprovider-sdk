package grok

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from llmprovider's grok_test.go and grok_tool_description_test.go,
// and the Grok parts of thinking_test.go, responses_store_test.go,
// api_error_message_test.go and identification_test.go (0015-PLAN S7). Each
// old method is the Request it sent.

// TestGrok_Generate verifies GenerateText returns text from a Responses API fixture.
func TestGrok_Generate(t *testing.T) {
	srv := replyServer(t, `{"id":"resp_1","output":[{"type":"message","content":[{"type":"output_text","text":"hello world"}]}]}`)
	out, err := llmprovider.GenerateText(context.Background(), build(t, apiKey(srv.URL, "grok-4")...), text("hi"))
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}
	if out != "hello world" {
		t.Errorf("output = %q, want %q", out, "hello world")
	}
}

// TestGrok_GenerateItems_MessageAndReasoning verifies interleaved output items
// are parsed correctly and OutputText() returns only message text.
func TestGrok_GenerateItems_MessageAndReasoning(t *testing.T) {
	srv := replyServer(t, `{
		"id": "resp_2",
		"output": [
			{"type": "reasoning", "summary": [{"type": "summary_text", "text": "thinking hard"}]},
			{"type": "message", "content": [{"type": "output_text", "text": "the answer"}]}
		]
	}`)
	resp, err := build(t, apiKey(srv.URL, "grok-4")...).Generate(context.Background(), text("hi"))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.Output) != 2 {
		t.Fatalf("expected 2 output items, got %d", len(resp.Output))
	}
	if _, ok := resp.Output[0].(llmprovider.ReasoningItem); !ok {
		t.Errorf("output[0] should be ReasoningItem, got %T", resp.Output[0])
	}
	if _, ok := resp.Output[1].(llmprovider.MessageItem); !ok {
		t.Errorf("output[1] should be MessageItem, got %T", resp.Output[1])
	}
	if got := resp.OutputText(); got != "the answer" {
		t.Errorf("OutputText() = %q, want %q", got, "the answer")
	}
	if resp.ID != "resp_2" {
		t.Errorf("ID = %q, want %q", resp.ID, "resp_2")
	}
}

// TestGrok_GenerateItemsWithTool verifies function_call output is parsed
// correctly, and that GenerateToolCall, which replaces GenerateWithTool,
// returns its arguments.
func TestGrok_GenerateItemsWithTool(t *testing.T) {
	srv := replyServer(t, `{
		"id": "resp_3",
		"output": [
			{"type": "function_call", "call_id": "call_1", "name": "get_weather", "arguments": "{\"city\":\"SF\"}"}
		]
	}`)
	p := build(t, apiKey(srv.URL, "grok-4")...)
	tool := llmprovider.Tool{Name: "get_weather", Description: "Get weather", Schema: map[string]any{"type": "object"}}
	resp, err := p.Generate(context.Background(), withTool(text("weather?"), tool))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.Output) != 1 {
		t.Fatalf("expected 1 output item, got %d", len(resp.Output))
	}
	fc, ok := resp.Output[0].(llmprovider.FunctionCallItem)
	if !ok {
		t.Fatalf("output[0] should be FunctionCallItem, got %T", resp.Output[0])
	}
	if fc.CallID != "call_1" || fc.Name != "get_weather" || fc.Arguments != `{"city":"SF"}` {
		t.Errorf("FunctionCallItem = %+v", fc)
	}

	call, err := llmprovider.GenerateToolCall(context.Background(), p, &llmprovider.Request{
		Input: []llmprovider.Item{user("weather?")}, Tools: []llmprovider.Tool{tool}})
	if err != nil {
		t.Fatalf("GenerateToolCall: %v", err)
	}
	if call.Arguments != `{"city":"SF"}` {
		t.Errorf("GenerateToolCall arguments = %q, want %q", call.Arguments, `{"city":"SF"}`)
	}
}

// TestGrok_GenerateThinking_ReasoningEffortSent verifies reasoning.effort is
// present in the request body for a model that supports it (grok-4.5).
func TestGrok_GenerateThinking_ReasoningEffortSent(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, okResponse)
	p := build(t, apiKey(srv.URL, "grok-4.5", llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: llmprovider.EffortHigh}))...)
	if _, err := p.Generate(context.Background(), thinking(text("hi"))); err != nil {
		t.Fatal(err)
	}
	if got := effortOf(body); got != "high" {
		t.Errorf("reasoning.effort = %v, want high (body %v)", got, body)
	}
}

// TestGrok_GenerateThinking_ReasoningEffortOmitted verifies reasoning key is
// absent for a model that rejects it (grok-4).
func TestGrok_GenerateThinking_ReasoningEffortOmitted(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, okResponse)
	p := build(t, apiKey(srv.URL, "grok-4", llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: llmprovider.EffortHigh}))...)
	if _, err := p.Generate(context.Background(), thinking(text("hi"))); err != nil {
		t.Fatal(err)
	}
	if _, present := body["reasoning"]; present {
		t.Errorf("reasoning must be omitted for grok-4: %v", body)
	}
}

// TestGrok_GenerateThinking_ReasoningEffortClamped verifies grok-3-mini clamps
// "medium" to the nearest lower effort on its low/high menu (MADR 0012 §6).
func TestGrok_GenerateThinking_ReasoningEffortClamped(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, okResponse)
	p := build(t, apiKey(srv.URL, "grok-3-mini", llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: llmprovider.EffortMedium}))...)
	if _, err := p.Generate(context.Background(), thinking(text("hi"))); err != nil {
		t.Fatal(err)
	}
	if got := effortOf(body); got != "low" {
		t.Errorf("reasoning.effort = %v, want low (clamped from medium)", got)
	}
}

// TestGrokThinkingTool: a forced tool with reasoning returns the call's
// arguments. It was TestGrokThinkingTool in llmprovider's thinking_test.go.
func TestGrokThinkingTool(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"output":[{"type":"function_call","call_id":"c1","name":"query","arguments":"{\"a\":2}"}]}`)
	p := build(t, apiKey(srv.URL, "grok-4.5", llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: llmprovider.EffortHigh}))...)
	tool := llmprovider.Tool{Name: "query", Description: "query", Schema: map[string]any{"type": "object"}}
	call, err := llmprovider.GenerateToolCall(context.Background(), p, thinking(&llmprovider.Request{
		Input: []llmprovider.Item{user("hi")}, Tools: []llmprovider.Tool{tool}}))
	if err != nil {
		t.Fatalf("GenerateToolCall error: %v", err)
	}
	if call.Arguments != `{"a":2}` {
		t.Errorf("expected '{\"a\":2}', got %q", call.Arguments)
	}
}

// TestGrok_Continue verifies previous_response_id is sent in request body
// and response ID is captured.
func TestGrok_Continue(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"id":"resp_next","output":[{"type":"message","content":[{"type":"output_text","text":"continued"}]}]}`)
	req := text("go on")
	req.PreviousResponseID = "resp_prev"
	resp, err := build(t, apiKey(srv.URL, "grok-4")...).Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if body["previous_response_id"] != "resp_prev" {
		t.Errorf("previous_response_id = %v, want resp_prev", body["previous_response_id"])
	}
	if resp.ID != "resp_next" {
		t.Errorf("Response.ID = %q, want resp_next", resp.ID)
	}
}

// TestGrok_ErrorClassification verifies status codes map to correct sentinel errors.
func TestGrok_ErrorClassification(t *testing.T) {
	tests := []struct {
		status int
		target error
	}{
		{429, llmprovider.ErrRateLimited},
		{401, llmprovider.ErrAuthFailure},
		{403, llmprovider.ErrAuthFailure},
		{500, llmprovider.ErrProviderUnavailable},
		{503, llmprovider.ErrProviderUnavailable},
		{400, llmprovider.ErrInvalidRequest},
		{422, llmprovider.ErrInvalidRequest},
	}
	for _, tc := range tests {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
		}))
		_, err := build(t, apiKey(srv.URL, "grok-4")...).Generate(context.Background(), text("hi"))
		srv.Close()
		if !errors.Is(err, tc.target) {
			t.Errorf("HTTP %d: got %v, want %v", tc.status, err, tc.target)
		}
	}
}

// TestGrok_KeyInHeader verifies API key travels in Authorization header, not URL.
func TestGrok_KeyInHeader(t *testing.T) {
	const key = "test-api-key-grok-123"
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(okResponse))
	}))
	defer srv.Close()
	p := build(t, llmprovider.WithAPIKey(key), llmprovider.WithModel("grok-4"), llmprovider.WithBaseURL(srv.URL))
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer "+key {
		t.Errorf("Authorization = %q, want Bearer %s", gotAuth, key)
	}
}

// TestGrok_GenerateItems_RequestBody verifies the Responses API request shape.
func TestGrok_GenerateItems_RequestBody(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, okResponse)
	if _, err := build(t, apiKey(srv.URL, "grok-4", llmprovider.WithMaxTokens(4096))...).Generate(context.Background(), text("hello")); err != nil {
		t.Fatal(err)
	}
	if body["model"] != "grok-4" {
		t.Errorf("model = %v", body["model"])
	}
	if mt, ok := body["max_output_tokens"].(float64); !ok || int(mt) != 4096 {
		t.Errorf("max_output_tokens = %v", body["max_output_tokens"])
	}
	input, ok := body["input"].([]any)
	if !ok || len(input) == 0 {
		t.Fatalf("input missing or empty: %v", body["input"])
	}
}

func TestGrok_GenerateItems_FunctionCallOutput(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"output":[{"type":"message","content":[{"type":"output_text","text":"got output"}]}]}`)
	resp, err := build(t, apiKey(srv.URL, "grok-4")...).Generate(context.Background(), items(user("compute"),
		llmprovider.FunctionCallOutputItem{CallID: "call_abc", Output: "{\"res\":1}"}))
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if resp.OutputText() != "got output" {
		t.Errorf("OutputText = %q, want 'got output'", resp.OutputText())
	}
}

// TestGrok_SendsToolDescription: the tool's description reaches xAI, as the
// OpenAI and OpenCode Responses paths already send it.
func TestGrok_SendsToolDescription(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"id":"r","output":[{"type":"function_call","call_id":"c","name":"get_weather","arguments":"{}"}]}`)
	tool := llmprovider.Tool{Name: "get_weather", Description: "Get the weather for a city", Schema: map[string]any{"type": "object"}}
	if _, err := build(t, apiKey(srv.URL, "grok-4.6")...).Generate(context.Background(), withTool(text("weather?"), tool)); err != nil {
		t.Fatal(err)
	}
	tools, _ := body["tools"].([]any)
	if len(tools) != 1 || tools[0].(map[string]any)["description"] != tool.Description {
		t.Fatalf("tools = %v, want the description", body["tools"])
	}
}

// TestGrok_WithStore: store is absent unless WithStore sets it. It was the
// grok rows of TestWithStore_ResponsesProviders.
func TestGrok_WithStore(t *testing.T) {
	for _, tc := range []struct {
		name  string
		opts  []llmprovider.Option
		want  any
		isSet bool
	}{
		{"default", nil, nil, false},
		{"false", []llmprovider.Option{WithStore(false)}, false, true},
		{"true", []llmprovider.Option{WithStore(true)}, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body map[string]any
			srv := captureServer(t, &body, okResponse)
			if _, err := build(t, apiKey(srv.URL, "grok-4.6", tc.opts...)...).Generate(context.Background(), text("hi")); err != nil {
				t.Fatal(err)
			}
			v, ok := body["store"]
			if ok != tc.isSet || v != tc.want {
				t.Errorf("store = %v (present %t), want %v (present %t)", v, ok, tc.want, tc.isSet)
			}
		})
	}
}

// TestGrok_ErrorCarriesServiceMessage: the error keeps the service's own
// explanation and still matches the sentinel its status maps to (MADR 0012
// §1.1, 0013 B3). It was the grok row of TestProviders_ErrorCarriesServiceMessage.
func TestGrok_ErrorCarriesServiceMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"model is unavailable"}}`))
	}))
	t.Cleanup(srv.Close)
	_, err := llmprovider.GenerateText(context.Background(), build(t, apiKey(srv.URL, "grok-4.5")...), text("hello"))
	if err == nil || !strings.Contains(err.Error(), "model is unavailable") {
		t.Fatalf("error = %v, want it to carry the service's message", err)
	}
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("error = %v, want it to still match ErrInvalidRequest", err)
	}
}

var userAgentPattern = regexp.MustCompile(`^go-llmprovider-sdk/\S+ \(\w+; \w+\) go-llmprovider-sdk/\S+$`)

// TestGrok_Identification: generation names this module honestly and vouches
// for no other client (MADR 0012 §1.4). It was the grok rows of
// TestIdentification_UserAgent and TestIdentification_NoForbiddenHeaders.
func TestGrok_Identification(t *testing.T) {
	var mu sync.Mutex
	var seen http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = r.Header.Clone()
		mu.Unlock()
		_, _ = w.Write([]byte(okResponse))
	}))
	t.Cleanup(srv.Close)
	if _, err := build(t, apiKey(srv.URL, "grok-4.5")...).Generate(context.Background(), text("hello")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if ua := seen.Get("User-Agent"); !userAgentPattern.MatchString(ua) {
		t.Errorf("User-Agent = %q, want go-llmprovider-sdk/<version> (<os>; <arch>) go-llmprovider-sdk/<version>", ua)
	}
	for name := range seen {
		lower := strings.ToLower(name)
		if lower == "x-opencode-client" || lower == "x-xai-token-auth" || strings.HasPrefix(lower, "x-grok-") {
			t.Errorf("forbidden header %s", name)
		}
	}
	if ua := strings.ToLower(seen.Get("User-Agent")); strings.Contains(ua, "codex") ||
		strings.Contains(ua, "kilo-code") || strings.Contains(ua, "opencode/") {
		t.Errorf("User-Agent %q impersonates a reference client", ua)
	}
}

// TestGrok_IDAndCapabilities is the Grok half of TestProviderNamesAndConstructors
// and the Grok assertions of the interface tests: what those interfaces
// declared is now Capabilities() and ModelLister.
func TestGrok_IDAndCapabilities(t *testing.T) {
	p := build(t, llmprovider.WithAPIKey("key"), llmprovider.WithModel("grok-x"))
	if p.ID() != llmprovider.ProviderGrok {
		t.Errorf("ID = %q, want %q", p.ID(), llmprovider.ProviderGrok)
	}
	caps := p.Capabilities()
	if caps.Tools != llmprovider.Supported || caps.ForcedToolChoice != llmprovider.Supported ||
		caps.Reasoning != llmprovider.Supported || caps.Continuation != llmprovider.Supported ||
		caps.NativeStreaming != llmprovider.Unsupported {
		t.Errorf("Capabilities = %+v", caps)
	}
	if _, ok := p.(llmprovider.ModelLister); !ok {
		t.Error("the provider is not a ModelLister")
	}
}
