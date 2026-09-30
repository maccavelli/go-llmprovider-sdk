package claude

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from llmprovider's claude_items_test.go, claude_system_test.go,
// thinking_test.go, thinking_wire_test.go, provider_correctness_test.go,
// api_error_message_test.go and identification_test.go (0015-PLAN S7). Each
// old method call is the Request it sent: GenerateItems is Input,
// GenerateWithTool a forced tool, GenerateThinking a Reasoning;
// WithThinkingBudget and WithReasoningEffort are WithReasoning's Budget and
// Effort.

func TestClaude_TextBlock(t *testing.T) {
	srv := replyServer(t, `{"content":[{"type":"text","text":"hello from claude"}]}`)
	p := build(t, apiKey(srv.URL, "claude-haiku-4-5")...)
	resp, err := p.Generate(context.Background(), text("hi"))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.Output) != 1 {
		t.Fatalf("expected 1 output item, got %d", len(resp.Output))
	}
	m, ok := resp.Output[0].(llmprovider.MessageItem)
	if !ok || m.Text != "hello from claude" {
		t.Errorf("expected MessageItem with 'hello from claude', got %v", resp.Output[0])
	}
	if resp.OutputText() != "hello from claude" {
		t.Errorf("OutputText() = %q, want %q", resp.OutputText(), "hello from claude")
	}
	if resp.ID != "" {
		t.Errorf("Response.ID must be empty for Claude, got %q", resp.ID)
	}
}

// TestClaude_MultiBlock verifies thinking + text blocks are decoded into
// ReasoningItem and MessageItem, and OutputText() returns only text.
func TestClaude_MultiBlock(t *testing.T) {
	srv := replyServer(t, `{"content":[{"type":"thinking","thinking":"deep thought process"},{"type":"text","text":"the result"}]}`)
	p := build(t, apiKey(srv.URL, "claude-sonnet-5")...)
	resp, err := p.Generate(context.Background(), text("solve"))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.Output) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Output))
	}
	r, ok := resp.Output[0].(llmprovider.ReasoningItem)
	if !ok || r.Text != "deep thought process" {
		t.Errorf("output[0] should be ReasoningItem with 'deep thought process', got %v", resp.Output[0])
	}
	m, ok := resp.Output[1].(llmprovider.MessageItem)
	if !ok || m.Text != "the result" {
		t.Errorf("output[1] should be MessageItem with 'the result', got %v", resp.Output[1])
	}
	if resp.OutputText() != "the result" {
		t.Errorf("OutputText() = %q, want %q", resp.OutputText(), "the result")
	}
}

func TestClaude_ToolUse(t *testing.T) {
	srv := replyServer(t, `{"content":[{"type":"tool_use","id":"tool_call_99","name":"search","input":{"q":"golang"}}]}`)
	p := build(t, apiKey(srv.URL, "claude-haiku-4-5")...)
	tool := llmprovider.Tool{Name: "search", Description: "Search", Schema: map[string]any{"type": "object"}}
	resp, err := p.Generate(context.Background(), withTool(text("search"), tool))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.Output) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Output))
	}
	fc, ok := resp.Output[0].(llmprovider.FunctionCallItem)
	if !ok || fc.CallID != "tool_call_99" || fc.Name != "search" || fc.Arguments != `{"q":"golang"}` {
		t.Errorf("FunctionCallItem = %+v", resp.Output[0])
	}
	// Also verify the convenience function, which replaced GenerateWithTool.
	call, err := llmprovider.GenerateToolCall(context.Background(), p, &llmprovider.Request{
		Input: text("search").Input, Tools: []llmprovider.Tool{tool}})
	if err != nil {
		t.Fatalf("GenerateToolCall: %v", err)
	}
	if call.Arguments != `{"q":"golang"}` {
		t.Errorf("GenerateToolCall = %q, want %q", call.Arguments, `{"q":"golang"}`)
	}
}

// TestClaude_Capabilities is the old interface-satisfaction test in the new
// API's terms: the old type implemented the tool, thinking and item
// interfaces and ModelDiscoverer, and deliberately not Continuer.
func TestClaude_Capabilities(t *testing.T) {
	p := build(t, llmprovider.WithAPIKey("k"))
	want := llmprovider.Capabilities{Tools: llmprovider.Supported, ForcedToolChoice: llmprovider.Supported,
		Reasoning: llmprovider.Supported, Continuation: llmprovider.Unsupported, NativeStreaming: llmprovider.Unsupported}
	if got := p.Capabilities(); got != want {
		t.Fatalf("Capabilities() = %+v, want %+v", got, want)
	}
	if _, ok := p.(llmprovider.ModelLister); !ok {
		t.Fatal("the provider is not a ModelLister")
	}
	if p.ID() != llmprovider.ProviderClaude {
		t.Fatalf("ID() = %q", p.ID())
	}
}

func TestClaude_FunctionCallOutput(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"content":[{"type":"text","text":"got result"}]}`)
	p := build(t, apiKey(srv.URL, "claude-haiku-4-5")...)
	resp, err := p.Generate(context.Background(), &llmprovider.Request{Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: "user", Text: "compute"},
		llmprovider.FunctionCallOutputItem{CallID: "call_abc", Output: "{\"answer\":42}"},
	}})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if resp.OutputText() != "got result" {
		t.Errorf("OutputText = %q, want 'got result'", resp.OutputText())
	}
}

func TestClaude_ErrorClassification(t *testing.T) {
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
		p := build(t, apiKey(srv.URL, "claude-haiku-4-5")...)
		_, err := p.Generate(context.Background(), text("hi"))
		srv.Close()
		if err == nil || !errors.Is(err, tc.target) {
			t.Errorf("HTTP %d: got %v, want %v", tc.status, err, tc.target)
		}
	}
}

// TestClaude_ErrorCarriesServiceMessage: the error keeps the service's own
// explanation (MADR 0012 §1.1, 0013 B3), and still matches the sentinel its
// status always mapped to.
func TestClaude_ErrorCarriesServiceMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"model is unavailable"}}`))
	}))
	t.Cleanup(srv.Close)
	p := build(t, apiKey(srv.URL, "claude-haiku-4-5")...)
	_, err := p.Generate(context.Background(), text("hello"))
	if err == nil || !strings.Contains(err.Error(), "model is unavailable") {
		t.Fatalf("error = %v, want it to carry the service's message", err)
	}
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("error = %v, want it to still match ErrInvalidRequest", err)
	}
}

var userAgentPattern = regexp.MustCompile(`^go-llmprovider-sdk/\S+ \(\w+; \w+\) go-llmprovider-sdk/\S+$`)

// TestClaude_UserAgent: a generation request names this module honestly
// (MADR 0012 §1.4), never Go's default agent.
func TestClaude_UserAgent(t *testing.T) {
	var ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	p := build(t, apiKey(srv.URL, "claude-haiku-4-5")...)
	_, _ = p.Generate(context.Background(), text("hello"))
	if !userAgentPattern.MatchString(ua) {
		t.Errorf("User-Agent = %q, want go-llmprovider-sdk/<version> (<os>; <arch>) go-llmprovider-sdk/<version>", ua)
	}
}

// TestClaude_MultiBlockContent is the regression for the Content[0] bug: a
// leading non-text (thinking) block must not cause an empty result.
func TestClaude_MultiBlockContent(t *testing.T) {
	srv := replyServer(t, `{"content":[{"type":"thinking","text":""},{"type":"text","text":"the answer"}]}`)
	p := build(t, apiKey(srv.URL, "claude-x")...)
	out, err := llmprovider.GenerateText(context.Background(), p, text("hi"))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if out != "the answer" {
		t.Errorf("expected concatenated text block, got %q", out)
	}
}

// TestClaude_SystemMessageIsTopLevel: system items are joined into the
// request's system field and never sent as an assistant turn (MADR 0012 §2).
func TestClaude_SystemMessageIsTopLevel(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"content":[{"type":"text","text":"Bonjour"}]}`)
	p := build(t, apiKey(srv.URL, "claude-haiku-4-5")...)
	_, err := p.Generate(context.Background(), &llmprovider.Request{Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: "system", Text: "Always answer in French."},
		llmprovider.MessageItem{Role: "system", Text: "Be brief."},
		llmprovider.MessageItem{Role: "user", Text: "Say hello."},
	}})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if body["system"] != "Always answer in French.\n\nBe brief." {
		t.Errorf("system = %#v, want the two system items joined", body["system"])
	}
	want := []any{map[string]any{"role": "user", "content": "Say hello."}}
	if !reflect.DeepEqual(body["messages"], want) {
		got, _ := json.Marshal(body["messages"])
		t.Errorf("messages = %s, want only the user turn", got)
	}
}

// TestClaudeThinking_RequestBody verifies the thinking path emits a thinking
// block and raises max_tokens above the budget (Anthropic requires
// max_tokens > budget_tokens).
func TestClaudeThinking_RequestBody(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, okReply)
	// maxTokens (4096) <= budget (8000) must force the ceiling above the budget.
	p := build(t, apiKey(srv.URL, "claude-haiku-4-5", llmprovider.WithMaxTokens(4096),
		llmprovider.WithReasoning(&llmprovider.Reasoning{Budget: 8000}))...)
	if _, err := p.Generate(context.Background(), thinking(text("hi"))); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	think, ok := body["thinking"].(map[string]any)
	if !ok {
		t.Fatalf("expected thinking block, body=%v", body)
	}
	if think["type"] != "enabled" {
		t.Errorf("thinking.type = %v, want enabled", think["type"])
	}
	if budget := think["budget_tokens"].(float64); int(budget) != 8000 {
		t.Errorf("budget_tokens = %v, want 8000", budget)
	}
	if mt := body["max_tokens"].(float64); !(int(mt) > 8000) {
		t.Errorf("max_tokens = %v, must exceed budget 8000", mt)
	}
}

// TestClaudeNonThinking_NoThinkingBlock verifies the default path is unchanged.
func TestClaudeNonThinking_NoThinkingBlock(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, okReply)
	p := build(t, apiKey(srv.URL, "claude-x")...)
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatal(err)
	}
	if _, present := body["thinking"]; present {
		t.Errorf("non-thinking request must not contain a thinking block: %v", body)
	}
}

// TestClaudeThinkingTool_AutoChoice verifies extended thinking drops the
// forced tool_choice (Anthropic forbids forcing a tool while thinking is
// enabled).
func TestClaudeThinkingTool_AutoChoice(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"content":[{"type":"tool_use","name":"emit","input":{"x":1}}]}`)
	p := build(t, apiKey(srv.URL, "claude-x", llmprovider.WithReasoning(&llmprovider.Reasoning{Budget: 2048}))...)
	tool := llmprovider.Tool{Name: "emit", Description: "d", Schema: map[string]any{"type": "object"}}
	if _, err := llmprovider.GenerateToolCall(context.Background(), p, &llmprovider.Request{
		Input: text("hi").Input, Tools: []llmprovider.Tool{tool}, Reasoning: &llmprovider.Reasoning{}}); err != nil {
		t.Fatal(err)
	}
	tc, ok := body["tool_choice"].(map[string]any)
	if !ok || tc["type"] != "auto" {
		t.Errorf("thinking tool_choice = %v, want type=auto", body["tool_choice"])
	}
}

// TestNew_Constructs: an API key builds the provider, and an empty one is
// refused. It was the Claude lines of TestProviderNamesAndConstructors.
func TestNew_Constructs(t *testing.T) {
	p, err := New(llmprovider.WithAPIKey("key"), llmprovider.WithModel("claude-x"))
	if err != nil || p.ID() != llmprovider.ProviderClaude {
		t.Errorf("New = %v, %v", p, err)
	}
	if _, err := New(llmprovider.WithAPIKey(""), llmprovider.WithModel("claude-x")); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("an empty key: err = %v, want ErrInvalidRequest", err)
	}
}

// thinkingCase is one thinking request and the thinking fields its body must
// carry. A nil want entry means the key must be absent.
type thinkingCase struct {
	model  string
	effort llmprovider.Effort
	budget int
	want   map[string]any
}

func assertThinkingFields(t *testing.T, body map[string]any, want map[string]any) {
	t.Helper()
	for k, v := range want {
		got, present := body[k]
		switch {
		case v == nil && present:
			t.Errorf("%s = %v, want it absent", k, got)
		case v != nil && !reflect.DeepEqual(got, v):
			t.Errorf("%s = %v, want %v", k, got, v)
		}
	}
}

// TestThinkingWire_Claude pins MADR 0013 Q1 and B9 on the Anthropic wire:
// Claude 4.7 and later take adaptive thinking with output_config.effort
// (thinking.type "enabled" is HTTP 400 there); older models take a budget,
// 1024 for "low". An explicit budget wins where a budget is accepted. Each
// case is sent twice: with the effort and budget set at construction, as the
// old test did, and on the request.
func TestThinkingWire_Claude(t *testing.T) {
	adaptive := map[string]any{"type": "adaptive"}
	for _, tc := range []thinkingCase{
		{"claude-sonnet-5", llmprovider.EffortLow, 0, map[string]any{"thinking": adaptive,
			"output_config": map[string]any{"effort": "low"}}},
		{"claude-opus-4-8", "", 0, map[string]any{"thinking": adaptive, "output_config": nil}},
		{"claude-sonnet-5", "", 9000, map[string]any{"thinking": adaptive, "max_tokens": float64(8192)}},
		{"claude-haiku-4-5", llmprovider.EffortLow, 0, map[string]any{"output_config": nil,
			"thinking": map[string]any{"type": "enabled", "budget_tokens": float64(1024)}}},
		{"claude-haiku-4-5", "", 0, map[string]any{
			"thinking": map[string]any{"type": "enabled", "budget_tokens": float64(4096)}}},
		{"claude-haiku-4-5", llmprovider.EffortLow, 2000, map[string]any{
			"thinking": map[string]any{"type": "enabled", "budget_tokens": float64(2000)}}},
		{"claude-sonnet-4-20250514", llmprovider.EffortLow, 0, map[string]any{
			"thinking": map[string]any{"type": "enabled", "budget_tokens": float64(1024)}}},
	} {
		reasoning := llmprovider.Reasoning{Effort: tc.effort, Budget: tc.budget}
		for _, via := range []string{"construction", "request"} {
			t.Run(tc.model+"/"+string(tc.effort)+"/"+via, func(t *testing.T) {
				var body map[string]any
				srv := captureServer(t, &body, okReply)
				opts := apiKey(srv.URL, tc.model)
				req := thinking(text("hi"))
				if via == "construction" {
					r := reasoning
					opts = append(opts, llmprovider.WithReasoning(&r))
				} else {
					r := reasoning
					req.Reasoning = &r
				}
				if _, err := build(t, opts...).Generate(context.Background(), req); err != nil {
					t.Fatalf("Generate: %v", err)
				}
				assertThinkingFields(t, body, tc.want)
			})
		}
	}
}
