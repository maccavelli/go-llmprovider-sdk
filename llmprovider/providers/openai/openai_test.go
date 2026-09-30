package openai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from llmprovider's openai_items_test.go, thinking_test.go,
// provider_correctness_test.go, api_error_message_test.go,
// identification_test.go and responses_store_test.go (0015-PLAN S7). Each old
// method call is the Request that method sent: GenerateItems is Input,
// GenerateWithTool a forced tool, GenerateThinking a Reasoning, Continue a
// PreviousResponseID.

func apiKey(url string, opts ...llmprovider.Option) []llmprovider.Option {
	return append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithBaseURL(url)}, opts...)
}

func replyServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestOpenAI_MessageOutput(t *testing.T) {
	srv := replyServer(t, `{"id":"resp_oai_1","output":[{"type":"message","content":[{"type":"output_text","text":"hello from openai"}]}]}`)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("gpt-4o"))...)
	resp, err := p.Generate(context.Background(), text("hi"))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.Output) != 1 {
		t.Fatalf("expected 1 output item, got %d", len(resp.Output))
	}
	if m, ok := resp.Output[0].(llmprovider.MessageItem); !ok || m.Text != "hello from openai" {
		t.Errorf("expected MessageItem with text 'hello from openai', got %v", resp.Output[0])
	}
	if resp.OutputText() != "hello from openai" {
		t.Errorf("OutputText() = %q, want %q", resp.OutputText(), "hello from openai")
	}
}

func TestOpenAI_FunctionCallOutput(t *testing.T) {
	srv := replyServer(t, `{"id":"resp_oai_2","output":[{"type":"function_call","call_id":"call_123","name":"calc","arguments":"{\"x\":42}"}]}`)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("gpt-4o"))...)
	tool := llmprovider.Tool{Name: "calc", Description: "Calculator", Schema: map[string]any{"type": "object"}}
	resp, err := p.Generate(context.Background(), withTool(text("compute"), tool))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.Output) != 1 {
		t.Fatalf("expected 1 output item, got %d", len(resp.Output))
	}
	fc, ok := resp.Output[0].(llmprovider.FunctionCallItem)
	if !ok || fc.CallID != "call_123" || fc.Name != "calc" || fc.Arguments != `{"x":42}` {
		t.Errorf("FunctionCallItem = %+v", resp.Output[0])
	}
	// Also verify the convenience function, which replaced GenerateWithTool.
	call, err := llmprovider.GenerateToolCall(context.Background(), p, &llmprovider.Request{
		Input: text("compute").Input, Tools: []llmprovider.Tool{tool}})
	if err != nil {
		t.Fatalf("GenerateToolCall: %v", err)
	}
	if call.Arguments != `{"x":42}` {
		t.Errorf("GenerateToolCall = %q, want %q", call.Arguments, `{"x":42}`)
	}
}

func TestOpenAI_InterleavedItems(t *testing.T) {
	srv := replyServer(t, `{"id":"resp_oai_3","output":[`+
		`{"type":"reasoning","summary":[{"type":"summary_text","text":"let me think"}]},`+
		`{"type":"message","content":[{"type":"output_text","text":"result is 42"}]}]}`)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("o4"))...)
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
	if resp.OutputText() != "result is 42" {
		t.Errorf("OutputText() = %q, want %q", resp.OutputText(), "result is 42")
	}
}

// TestOpenAI_Continue verifies previous_response_id is sent in the request body.
func TestOpenAI_Continue(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"id":"resp_oai_4","output":[{"type":"message","content":[{"type":"output_text","text":"continued"}]}]}`)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("gpt-4o"))...)
	req := text("next step")
	req.PreviousResponseID = "resp_prev_1"
	resp, err := p.Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if body["previous_response_id"] != "resp_prev_1" {
		t.Errorf("previous_response_id = %v, want resp_prev_1", body["previous_response_id"])
	}
	if resp.ID != "resp_oai_4" {
		t.Errorf("Response.ID = %q, want resp_oai_4", resp.ID)
	}
}

func TestOpenAI_ResponseID(t *testing.T) {
	srv := replyServer(t, `{"id":"resp_id_999","output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("gpt-4o"))...)
	resp, err := p.Generate(context.Background(), text("hi"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != "resp_id_999" {
		t.Errorf("Response.ID = %q, want resp_id_999", resp.ID)
	}
}

// TestOpenAI_Capabilities is the old interface-satisfaction test in the new
// API's terms: the old type implemented the tool, thinking, item and
// Continuer interfaces and ModelDiscoverer, so tools, forced tools, reasoning
// and continuation are Supported with an API key, and it is a ModelLister.
func TestOpenAI_Capabilities(t *testing.T) {
	p := build(t, llmprovider.WithAPIKey("k"))
	want := llmprovider.Capabilities{Tools: llmprovider.Supported, ForcedToolChoice: llmprovider.Supported,
		Reasoning: llmprovider.Supported, Continuation: llmprovider.Supported, NativeStreaming: llmprovider.Unsupported}
	if got := p.Capabilities(); got != want {
		t.Fatalf("Capabilities() = %+v, want %+v", got, want)
	}
	if _, ok := p.(llmprovider.ModelLister); !ok {
		t.Fatal("the provider is not a ModelLister")
	}
	if _, ok := p.(llmprovider.Streamer); ok {
		t.Fatal("the provider claims native streaming")
	}
	if p.ID() != llmprovider.ProviderOpenAI {
		t.Fatalf("ID() = %q", p.ID())
	}
}

func TestOpenAI_FunctionCallOutputItem(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"output":[{"type":"message","content":[{"type":"output_text","text":"received output"}]}]}`)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("gpt-4o"))...)
	resp, err := p.Generate(context.Background(), &llmprovider.Request{Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: "user", Text: "compute"},
		llmprovider.FunctionCallOutputItem{CallID: "call_abc", Output: "{\"res\":1}"},
	}})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if resp.OutputText() != "received output" {
		t.Errorf("OutputText = %q, want 'received output'", resp.OutputText())
	}
}

func TestOpenAI_ErrorClassification(t *testing.T) {
	for _, tc := range []struct {
		status int
		target error
	}{
		{429, llmprovider.ErrRateLimited},
		{401, llmprovider.ErrAuthFailure},
		{403, llmprovider.ErrAuthFailure},
		{500, llmprovider.ErrProviderUnavailable},
		{400, llmprovider.ErrInvalidRequest},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
		}))
		p := build(t, apiKey(srv.URL, llmprovider.WithModel("gpt-4o"))...)
		_, err := p.Generate(context.Background(), text("hi"))
		srv.Close()
		if err == nil || !errors.Is(err, tc.target) {
			t.Errorf("HTTP %d: got %v, want %v", tc.status, err, tc.target)
		}
	}
}

// TestOpenAI_ErrorCarriesServiceMessage: the error keeps the service's own
// explanation (MADR 0012 §1.1, 0013 B3), and still matches the sentinel its
// status always mapped to.
func TestOpenAI_ErrorCarriesServiceMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"model is unavailable"}}`))
	}))
	t.Cleanup(srv.Close)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("gpt-4.1-mini"))...)
	_, err := p.Generate(context.Background(), text("hello"))
	if err == nil || !strings.Contains(err.Error(), "model is unavailable") {
		t.Fatalf("error = %v, want it to carry the service's message", err)
	}
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("error = %v, want it to still match ErrInvalidRequest", err)
	}
}

var userAgentPattern = regexp.MustCompile(`^go-llmprovider-sdk/\S+ \(\w+; \w+\) go-llmprovider-sdk/\S+$`)

// TestOpenAI_UserAgent: a generation request names this module honestly
// (MADR 0012 §1.4), never Go's default agent.
func TestOpenAI_UserAgent(t *testing.T) {
	var ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("gpt-4.1-mini"))...)
	_, _ = p.Generate(context.Background(), text("hello"))
	if !userAgentPattern.MatchString(ua) {
		t.Errorf("User-Agent = %q, want go-llmprovider-sdk/<version> (<os>; <arch>) go-llmprovider-sdk/<version>", ua)
	}
}

// TestOpenAIThinking_RequestBody: the reasoning set at construction reaches
// the body, beside the output limit. It was WithReasoningEffort("high") and
// GenerateThinking.
func TestOpenAIThinking_RequestBody(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, okResponse)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("o4"), llmprovider.WithMaxTokens(5000),
		llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: llmprovider.EffortHigh}))...)
	if _, err := p.Generate(context.Background(), thinking(text("hi"))); err != nil {
		t.Fatal(err)
	}
	if mt, ok := body["max_output_tokens"].(float64); !ok || int(mt) != 5000 {
		t.Errorf("max_output_tokens = %v, want 5000", body["max_output_tokens"])
	}
	reasoning, ok := body["reasoning"].(map[string]any)
	if !ok || reasoning["effort"] != "high" {
		t.Errorf("reasoning.effort = %v, want high", body["reasoning"])
	}
}

// TestOpenAIThinking_DefaultEffort verifies the default effort is applied when unset.
func TestOpenAIThinking_DefaultEffort(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, okResponse)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("o4"))...)
	if _, err := p.Generate(context.Background(), thinking(text("hi"))); err != nil {
		t.Fatal(err)
	}
	reasoning, ok := body["reasoning"].(map[string]any)
	if !ok || reasoning["effort"] != string(llmprovider.EffortMedium) {
		t.Errorf("reasoning.effort = %v, want default %q", body["reasoning"], llmprovider.EffortMedium)
	}
}

// TestOpenAINonThinking_UsesMaxTokens verifies the default path uses
// max_output_tokens and omits reasoning.
func TestOpenAINonThinking_UsesMaxTokens(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, okResponse)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("gpt-x"), llmprovider.WithMaxTokens(321))...)
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatal(err)
	}
	if mt, ok := body["max_output_tokens"].(float64); !ok || int(mt) != 321 {
		t.Errorf("max_output_tokens = %v, want 321", body["max_output_tokens"])
	}
	if _, present := body["reasoning"]; present {
		t.Errorf("non-thinking request must not send reasoning: %v", body)
	}
}

func TestOpenAIThinkingTool(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"output":[{"type":"function_call","call_id":"c1","name":"calc","arguments":"{\"x\":1}"}]}`)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("o4"),
		llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: llmprovider.EffortHigh}))...)
	tool := llmprovider.Tool{Name: "calc", Description: "calc", Schema: map[string]any{"type": "object"}}
	call, err := llmprovider.GenerateToolCall(context.Background(), p, &llmprovider.Request{
		Input: text("hi").Input, Tools: []llmprovider.Tool{tool}, Reasoning: &llmprovider.Reasoning{}})
	if err != nil {
		t.Fatalf("GenerateToolCall error: %v", err)
	}
	if call.Arguments != `{"x":1}` {
		t.Errorf("expected arguments '{\"x\":1}', got %q", call.Arguments)
	}
	reasoning, ok := body["reasoning"].(map[string]any)
	if !ok || reasoning["effort"] != "high" {
		t.Errorf("expected reasoning effort high, got %v", body["reasoning"])
	}
}

// TestOpenAI_WithMaxTokens is the regression: WithMaxTokens must reach the body.
func TestOpenAI_WithMaxTokens(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, okResponse)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("gpt-x"), llmprovider.WithMaxTokens(123))...)
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if mt, ok := body["max_output_tokens"].(float64); !ok || int(mt) != 123 {
		t.Errorf("max_output_tokens not sent: %v", body["max_output_tokens"])
	}
}

// TestWithStore_APIKey: store is absent unless WithStore sets it.
func TestWithStore_APIKey(t *testing.T) {
	const reply = `{"id":"r","output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`
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
			srv := captureServer(t, &body, reply)
			p := build(t, apiKey(srv.URL, append([]llmprovider.Option{llmprovider.WithModel("gpt-6-luna")}, tc.opts...)...)...)
			if _, err := p.Generate(context.Background(), text("hi")); err != nil {
				t.Fatal(err)
			}
			v, ok := body["store"]
			if ok != tc.isSet || v != tc.want {
				t.Errorf("store = %v (present %t), want %v (present %t)", v, ok, tc.want, tc.isSet)
			}
		})
	}
}

// TestNew_Constructs: an API key builds the provider; its id is openai. It
// was the OpenAI line of TestProviderNamesAndConstructors.
func TestNew_Constructs(t *testing.T) {
	p, err := New(llmprovider.WithAPIKey("key"), llmprovider.WithModel("gpt-x"))
	if err != nil || p.ID() != llmprovider.ProviderOpenAI {
		t.Fatalf("New = %v, %v", p, err)
	}
}
