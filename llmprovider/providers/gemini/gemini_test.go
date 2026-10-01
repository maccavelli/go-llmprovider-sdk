package gemini

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

// Ported from llmprovider's gemini_test.go, and the Gemini parts of
// provider_correctness_test.go, thinking_test.go, api_error_message_test.go
// and identification_test.go (0015-PLAN S7).

// TestGemini_KeyInHeaderNotURL is the #4 regression: the API key must travel in
// the x-goog-api-key header, never the URL query.
func TestGemini_KeyInHeaderNotURL(t *testing.T) {
	const key = "test-api-key-gemini-123"
	var gotKeyHeader, gotRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKeyHeader = r.Header.Get("x-goog-api-key")
		gotRawQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(interactionText))
	}))
	defer srv.Close()

	p := build(t, llmprovider.WithAPIKey(key), llmprovider.WithModel("gemini-x"), llmprovider.WithBaseURL(srv.URL))
	out, err := llmprovider.GenerateText(context.Background(), p, text("hi"))
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}
	if out != "ok" {
		t.Errorf("output: got %q", out)
	}
	if gotKeyHeader != key {
		t.Errorf("x-goog-api-key header: got %q want %q", gotKeyHeader, key)
	}
	if strings.Contains(gotRawQuery, key) || strings.Contains(gotRawQuery, "key=") {
		t.Errorf("key leaked into URL query: %q", gotRawQuery)
	}
}

// TestGemini_KeyNotInError confirms a transport error (*url.Error) does not
// embed the API key now that it is out of the URL.
func TestGemini_KeyNotInError(t *testing.T) {
	const key = "test-api-key-gemini-err"
	p := build(t, llmprovider.WithAPIKey(key), llmprovider.WithModel("gemini-x"), llmprovider.WithBaseURL("http://127.0.0.1:0"))
	_, err := p.Generate(context.Background(), text("hi"))
	if err == nil {
		t.Fatal("expected transport error")
	}
	if strings.Contains(err.Error(), key) {
		t.Errorf("API key leaked into error: %v", err)
	}
}

// TestGemini_WithMaxTokens is the regression for Gemini's generation_config.
func TestGemini_WithMaxTokens(t *testing.T) {
	srv, c := interactionServer(t, interactionText)
	p := build(t, apiKey(srv.URL, "gemini-x", llmprovider.WithMaxTokens(123))...)
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	_, body := c.last()
	gc := generationConfig(body)
	if gc == nil {
		t.Fatalf("generation_config missing: %v", body)
	}
	if mt, ok := gc["max_output_tokens"].(float64); !ok || int(mt) != 123 {
		t.Errorf("max_output_tokens not sent: %v", gc["max_output_tokens"])
	}
}

// TestGeminiThinkingTool: a forced tool with thinking returns the call's
// arguments and sends both thinking_level and a tool_choice.
func TestGeminiThinkingTool(t *testing.T) {
	srv, c := interactionServer(t, `{"id":"v1_t","status":"requires_action","steps":[{"type":"thought","signature":"s"},`+
		`{"type":"function_call","id":"c1","name":"lookup","arguments":{"q":"test"}}]}`)
	p := build(t, apiKey(srv.URL, "gemini-3.7-flash", llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: llmprovider.EffortHigh}))...)
	tool := llmprovider.Tool{Name: "lookup", Description: "lookup", Schema: map[string]any{"type": "object"}}
	call, err := llmprovider.GenerateToolCall(context.Background(), p, thinking(&llmprovider.Request{
		Input: []llmprovider.Item{user("hi")}, Tools: []llmprovider.Tool{tool}}))
	if err != nil {
		t.Fatalf("GenerateToolCall error: %v", err)
	}
	if call.Arguments != `{"q":"test"}` {
		t.Errorf("expected '{\"q\":\"test\"}', got %q", call.Arguments)
	}
	_, body := c.last()
	gc := generationConfig(body)
	if gc["thinking_level"] != string(llmprovider.EffortHigh) || gc["tool_choice"] == nil {
		t.Errorf("generation_config = %v, want thinking_level high and a tool_choice", gc)
	}
}

// TestGemini_ErrorCarriesServiceMessage: the error keeps the service's own
// explanation and still matches the sentinel its status maps to (MADR 0012
// §1.1, 0013 B3). It was the gemini row of
// TestProviders_ErrorCarriesServiceMessage.
func TestGemini_ErrorCarriesServiceMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"model is unavailable"}}`))
	}))
	t.Cleanup(srv.Close)
	_, err := llmprovider.GenerateText(context.Background(), build(t, apiKey(srv.URL, "gemini-3.7-flash")...), text("hello"))
	if err == nil || !strings.Contains(err.Error(), "model is unavailable") {
		t.Fatalf("error = %v, want it to carry the service's message", err)
	}
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("error = %v, want it to still match ErrInvalidRequest", err)
	}
}

var userAgentPattern = regexp.MustCompile(`^go-llmprovider-sdk/\S+ \(\w+; \w+\) go-llmprovider-sdk/\S+$`)

// TestGemini_UserAgent: generation names this module honestly (MADR 0012
// §1.4), never Go's default agent. It was the gemini row of
// TestIdentification_UserAgent.
func TestGemini_UserAgent(t *testing.T) {
	srv, c := interactionServer(t, interactionText)
	if _, err := build(t, apiKey(srv.URL, "gemini-3.7-flash")...).Generate(context.Background(), text("hello")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if ua := c.headers[0].Get("User-Agent"); !userAgentPattern.MatchString(ua) {
		t.Errorf("User-Agent = %q, want go-llmprovider-sdk/<version> (<os>; <arch>) go-llmprovider-sdk/<version>", ua)
	}
}

// TestGemini_ID is the Gemini half of TestProviderNamesAndConstructors.
func TestGemini_ID(t *testing.T) {
	if id := build(t, llmprovider.WithAPIKey("key"), llmprovider.WithModel("gemini-x")).ID(); id != llmprovider.ProviderGemini {
		t.Errorf("ID = %q, want %q", id, llmprovider.ProviderGemini)
	}
}
