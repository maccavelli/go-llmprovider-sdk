package llmprovider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// captureServer returns an httptest server that records the decoded JSON request body of
// the last call and replies with the given canned response body.
func captureServer(t *testing.T, lastBody *map[string]any, respBody string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		m := map[string]any{}
		_ = json.Unmarshal(raw, &m)
		*lastBody = m
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, respBody)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestGoogleRouteThinking_RequestBody verifies a thinkingConfig is nested in
// generationConfig with the configured budget on OpenCode's google route, and
// that the default path omits it. GeminiProvider has no budget (MADR 0014).
func TestGoogleRouteThinking_RequestBody(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`)

	p, _ := NewOpencode(ProviderOpencodeZen, "k", "gemini-x", WithBaseURL(srv.URL), WithThinkingBudget(1234))
	if _, err := p.GenerateThinking(context.Background(), "hi"); err != nil {
		t.Fatal(err)
	}
	gc, ok := body["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("missing generationConfig: %v", body)
	}
	tc, ok := gc["thinkingConfig"].(map[string]any)
	if !ok {
		t.Fatalf("missing thinkingConfig: %v", gc)
	}
	if b := tc["thinkingBudget"].(float64); int(b) != 1234 {
		t.Errorf("thinkingBudget = %v, want 1234", b)
	}

	// Default path: no thinkingConfig.
	body = nil
	if _, err := p.Generate(context.Background(), "hi"); err != nil {
		t.Fatal(err)
	}
	gc2 := body["generationConfig"].(map[string]any)
	if _, present := gc2["thinkingConfig"]; present {
		t.Errorf("non-thinking request must not contain thinkingConfig: %v", gc2)
	}
}

// TestGoogleRouteThinking_DynamicBudgetDefault verifies an unset budget maps to
// -1 (dynamic) on OpenCode's google route.
func TestGoogleRouteThinking_DynamicBudgetDefault(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`)

	p, _ := NewOpencode(ProviderOpencodeZen, "k", "gemini-x", WithBaseURL(srv.URL))
	if _, err := p.GenerateThinking(context.Background(), "hi"); err != nil {
		t.Fatal(err)
	}
	gc := body["generationConfig"].(map[string]any)
	tc := gc["thinkingConfig"].(map[string]any)
	if b := tc["thinkingBudget"].(float64); int(b) != dynamicGeminiThinkingBudget {
		t.Errorf("thinkingBudget = %v, want %d (dynamic)", b, dynamicGeminiThinkingBudget)
	}
}

func TestGeminiThinkingTool(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"id":"v1_t","status":"requires_action","steps":[{"type":"thought","signature":"s"},`+
		`{"type":"function_call","id":"c1","name":"lookup","arguments":{"q":"test"}}]}`)

	p, _ := NewGemini(context.Background(), "k", "gemini-3.7-flash", WithBaseURL(srv.URL), WithReasoningEffort(effortHigh))
	tool := Tool{Name: "lookup", Description: "lookup", Schema: map[string]any{"type": "object"}}
	args, err := p.GenerateWithToolThinking(context.Background(), "hi", tool)
	if err != nil {
		t.Fatalf("GenerateWithToolThinking error: %v", err)
	}
	if args != `{"q":"test"}` {
		t.Errorf("expected '{\"q\":\"test\"}', got %q", args)
	}
	gc, _ := body["generation_config"].(map[string]any)
	if gc["thinking_level"] != effortHigh || gc["tool_choice"] == nil {
		t.Errorf("generation_config = %v, want thinking_level high and a tool_choice", gc)
	}
}

func TestGrokThinkingTool(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"output":[{"type":"function_call","call_id":"c1","name":"query","arguments":"{\"a\":2}"}]}`)

	p, _ := NewGrok("k", "grok-4.5", WithBaseURL(srv.URL), WithReasoningEffort("high"))
	tool := Tool{Name: "query", Description: "query", Schema: map[string]any{"type": "object"}}
	args, err := p.GenerateWithToolThinking(context.Background(), "hi", tool)
	if err != nil {
		t.Fatalf("GenerateWithToolThinking error: %v", err)
	}
	if args != `{"a":2}` {
		t.Errorf("expected '{\"a\":2}', got %q", args)
	}
}

func TestProviderNamesAndConstructors(t *testing.T) {
	gp, err := NewGemini(context.Background(), "key", "gemini-x")
	if err != nil || gp.Name() != ProviderGemini {
		t.Errorf("Gemini name = %v, err = %v", gp.Name(), err)
	}

	xaiP, err := NewGrok("key", "grok-x")
	if err != nil || xaiP.Name() != ProviderGrok {
		t.Errorf("Grok name = %v, err = %v", xaiP.Name(), err)
	}
	if _, err := NewGrok("", "grok-x"); err == nil {
		t.Error("expected error for empty Grok API key")
	}
}

// TestProvidersImplementThinkingInterfaces is a compile-time guarantee that all
// providers satisfy the optional thinking interfaces.
func TestProvidersImplementThinkingInterfaces(t *testing.T) {
	var _ ThinkingProvider = (*GeminiProvider)(nil)
	var _ ThinkingProvider = (*GrokProvider)(nil)
	var _ ThinkingToolProvider = (*GeminiProvider)(nil)
	var _ ThinkingToolProvider = (*GrokProvider)(nil)
	var _ ItemProvider = (*GrokProvider)(nil)
	var _ Continuer = (*GrokProvider)(nil)
}
