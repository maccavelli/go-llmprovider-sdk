package opencode

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from llmprovider's opencode_test.go (0015-PLAN S7). Each old method
// is the Request it sent; Route() is read from the request path.

// TestOpencode_RoutePaths asserts the per-gateway divergence at the HTTP layer:
// the same model id takes different paths on Zen and Go.
func TestOpencode_RoutePaths(t *testing.T) {
	tests := []struct {
		gateway                  llmprovider.ProviderID
		model, wantPath, fixture string
	}{
		{llmprovider.ProviderOpencodeZen, "gpt-5.5", "/responses", fxResponses},
		{llmprovider.ProviderOpencodeZen, "claude-sonnet-5", "/messages", fxMessages},
		{llmprovider.ProviderOpencodeZen, "gemini-3.7-flash", "/models/gemini-3.7-flash:generateContent", fxGoogle},
		{llmprovider.ProviderOpencodeZen, deepSeekV4Pro, "/chat/completions", fxChat},
		{llmprovider.ProviderOpencodeZen, "minimax-m3", "/chat/completions", fxChat},
		{llmprovider.ProviderOpencodeGo, "minimax-m3", "/messages", fxMessages},
	}
	for _, tc := range tests {
		t.Run(string(tc.gateway)+"/"+tc.model, func(t *testing.T) {
			var path string
			srv := pathCapture(t, &path, tc.fixture)
			if _, err := build(t, tc.gateway, apiKey(srv.URL, tc.model)...).Generate(context.Background(), text("hi")); err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if path != tc.wantPath {
				t.Errorf("path = %q, want %q", path, tc.wantPath)
			}
		})
	}
}

// TestOpencode_RequestModelPicksItsRoute: a Request naming another model is
// routed for that model, not the provider's (new with the Request API).
func TestOpencode_RequestModelPicksItsRoute(t *testing.T) {
	var path string
	srv := pathCapture(t, &path, fxMessages)
	req := text("hi")
	req.Model = "claude-sonnet-5"
	if _, err := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "gpt-5.5")...).Generate(context.Background(), req); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if path != "/messages" {
		t.Errorf("path = %q, want claude-sonnet-5's /messages", path)
	}
}

func TestOpencode_Generate_PerRoute(t *testing.T) {
	tests := []struct{ name, model, fixture string }{
		{"responses", "gpt-5.5", fxResponses},
		{"messages", "claude-sonnet-5", fxMessages},
		{"google", "gemini-3.7-flash", fxGoogle},
		{"chat", deepSeekV4Pro, fxChat},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var path string
			srv := pathCapture(t, &path, tc.fixture)
			resp, err := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, tc.model)...).Generate(context.Background(), text("hi"))
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if got := resp.OutputText(); got != "hello" {
				t.Errorf("OutputText() = %q, want hello", got)
			}
			switch tc.name {
			case "responses":
				// Zen sends summary:[] with encrypted_content, so the decoder
				// emits a present-but-blank ReasoningItem.
				r, ok := resp.Output[0].(llmprovider.ReasoningItem)
				if !ok {
					t.Fatalf("output[0] = %T, want ReasoningItem", resp.Output[0])
				}
				if r.Text != "" {
					t.Errorf("reasoning text = %q, want empty (summary is [])", r.Text)
				}
			case "messages":
				if resp.ID != "" {
					t.Errorf("messages route is stateless; ID = %q, want empty", resp.ID)
				}
			}
		})
	}
}

// TestOpencode_KeyInHeader pins MADR 0007 §1c: each Zen/Go route reads the key
// from its vendor's header, and no route reads the others.
func TestOpencode_KeyInHeader(t *testing.T) {
	keyHeaders := []string{"Authorization", "x-api-key", "x-goog-api-key"}
	tests := []struct{ model, fixture, header, value string }{
		{"gpt-5.5", fxResponses, "Authorization", "Bearer test-key"},
		{"claude-sonnet-5", fxMessages, "x-api-key", "test-key"},
		{"gemini-3.7-flash", fxGoogle, "x-goog-api-key", "test-key"},
		{deepSeekV4Pro, fxChat, "Authorization", "Bearer test-key"},
	}
	for _, tc := range tests {
		t.Run(tc.model, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for _, h := range keyHeaders {
					got := r.Header.Get(h)
					switch {
					case h == tc.header && got != tc.value:
						t.Errorf("%s = %q, want %q", h, got, tc.value)
					case h != tc.header && got != "":
						t.Errorf("%s must not be sent on this route, got %q", h, got)
					}
				}
				if r.URL.RawQuery != "" {
					t.Errorf("key must never reach the URL; RawQuery = %q", r.URL.RawQuery)
				}
				_, _ = w.Write([]byte(tc.fixture))
			}))
			defer srv.Close()
			p := build(t, llmprovider.ProviderOpencodeZen, llmprovider.WithAPIKey("test-key"),
				llmprovider.WithModel(tc.model), llmprovider.WithBaseURL(srv.URL))
			if _, err := p.Generate(context.Background(), text("hi")); err != nil {
				t.Fatalf("Generate: %v", err)
			}
		})
	}
}

func TestOpencode_ToolCall_PerRoute(t *testing.T) {
	tool := llmprovider.Tool{Name: "get_weather", Description: "Get weather", Schema: map[string]any{"type": "object"}}
	tests := []struct{ name, model, fixture string }{
		{"responses", "gpt-5.5",
			`{"output":[{"type":"function_call","call_id":"c1","name":"get_weather","arguments":"{\"city\":\"SF\"}"}]}`},
		{"messages", "claude-sonnet-5",
			`{"content":[{"type":"tool_use","id":"c1","name":"get_weather","input":{"city":"SF"}}]}`},
		{"google", "gemini-3.7-flash",
			`{"candidates":[{"content":{"parts":[{"functionCall":{"name":"get_weather","args":{"city":"SF"}}}]}}]}`},
		{"chat", deepSeekV4Pro,
			`{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"SF\"}"}}]}}]}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var path string
			srv := pathCapture(t, &path, tc.fixture)
			call, err := llmprovider.GenerateToolCall(context.Background(), build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, tc.model)...),
				&llmprovider.Request{Input: []llmprovider.Item{user("weather?")}, Tools: []llmprovider.Tool{tool}})
			if err != nil {
				t.Fatalf("GenerateToolCall: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal([]byte(call.Arguments), &parsed); err != nil {
				t.Fatalf("arguments %q are not valid JSON: %v", call.Arguments, err)
			}
			if parsed["city"] != "SF" {
				t.Errorf("arguments = %q", call.Arguments)
			}
		})
	}
}

// TestOpencode_Thinking_PerRoute pins where each route carries reasoning, and
// that the chat route carries none at all.
func TestOpencode_Thinking_PerRoute(t *testing.T) {
	t.Run("responses uses reasoning.effort", func(t *testing.T) {
		var body map[string]any
		srv := captureServer(t, &body, fxResponses)
		if _, err := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "gpt-5.5")...).Generate(context.Background(), thinking(text("hi"))); err != nil {
			t.Fatalf("Generate: %v", err)
		}
		r, ok := body["reasoning"].(map[string]any)
		if !ok || r["effort"] != string(llmprovider.EffortMedium) {
			t.Errorf("reasoning = %v, want effort=medium", body["reasoning"])
		}
	})

	t.Run("messages uses thinking.budget_tokens and raises max_tokens", func(t *testing.T) {
		var body map[string]any
		srv := captureServer(t, &body, fxMessages)
		p := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "claude-haiku-4-5", llmprovider.WithMaxTokens(4096),
			llmprovider.WithReasoning(&llmprovider.Reasoning{Budget: 8000}))...)
		if _, err := p.Generate(context.Background(), thinking(text("hi"))); err != nil {
			t.Fatalf("Generate: %v", err)
		}
		th, ok := body["thinking"].(map[string]any)
		if !ok || th["budget_tokens"].(float64) != 8000 {
			t.Fatalf("thinking = %v", body["thinking"])
		}
		mt, ok := body["max_tokens"].(float64)
		if !ok || mt <= 8000 {
			t.Errorf("max_tokens = %v, must exceed budget_tokens", body["max_tokens"])
		}
	})

	t.Run("google uses generationConfig.thinkingConfig", func(t *testing.T) {
		var body map[string]any
		srv := captureServer(t, &body, fxGoogle)
		if _, err := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "gemini-3.7-flash")...).Generate(context.Background(), thinking(text("hi"))); err != nil {
			t.Fatalf("Generate: %v", err)
		}
		gc, ok := body["generationConfig"].(map[string]any)
		if !ok {
			t.Fatalf("generationConfig missing: %v", body)
		}
		if _, ok := gc["thinkingConfig"]; !ok {
			t.Errorf("thinkingConfig missing: %v", gc)
		}
	})

	t.Run("chat carries no reasoning key at all", func(t *testing.T) {
		var body map[string]any
		srv := captureServer(t, &body, fxChat)
		if _, err := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, deepSeekV4Pro)...).Generate(context.Background(), thinking(text("hi"))); err != nil {
			t.Fatalf("Generate: %v", err)
		}
		for _, k := range []string{"reasoning", "reasoning_effort", "thinking"} {
			if _, present := body[k]; present {
				t.Errorf("chat route must not send %q: %v", k, body[k])
			}
		}
	})
}

// TestOpencode_ChatReasoningEffort pins MADR 0009 §6 for the chat route:
// reasoning_effort is sent only on a reasoning call with an effort configured
// that the model's published reasoning_options list.
func TestOpencode_ChatReasoningEffort(t *testing.T) {
	enableMetadata(t)
	meta, _ := metadataServer(t, http.StatusOK, `{"opencode":{"models":{"`+deepSeekV4Pro+`":{`+
		`"reasoning":true,"reasoning_options":[{"type":"effort","values":["low","high","max"]}]}}}}`)
	tests := []struct {
		name, model     string
		effort          llmprovider.Effort
		plain, disabled bool
		want            string // "" means reasoning_effort is absent
	}{
		{"listed", deepSeekV4Pro, "low", false, false, "low"},
		{"unlisted value", deepSeekV4Pro, "medium", false, false, ""},
		{"uncovered model", "kimi-k2.6", "low", false, false, ""},
		{"no effort", deepSeekV4Pro, "", false, false, ""},
		{"plain call", deepSeekV4Pro, "low", true, false, ""},
		{"disabled", deepSeekV4Pro, "low", false, true, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body map[string]any
			srv := captureServer(t, &body, fxChat)
			opts := []llmprovider.Option{llmprovider.WithModelMetadataURL(meta.URL)}
			if tc.disabled {
				opts = append(opts, llmprovider.WithoutModelMetadata())
			}
			p := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, tc.model, opts...)...)
			req := text("hi")
			if !tc.plain {
				req.Reasoning = &llmprovider.Reasoning{Effort: tc.effort}
			}
			if _, err := p.Generate(context.Background(), req); err != nil {
				t.Fatalf("generate: %v", err)
			}
			got, present := body["reasoning_effort"]
			switch {
			case tc.want == "" && present:
				t.Errorf("reasoning_effort = %v, want absent", got)
			case tc.want != "" && got != tc.want:
				t.Errorf("reasoning_effort = %v, want %q", got, tc.want)
			}
			for _, k := range []string{"reasoning", "thinking"} {
				if _, ok := body[k]; ok {
					t.Errorf("chat route must not send %q: %v", k, body[k])
				}
			}
		})
	}
}

// TestOpencode_SessionHeader pins the x-opencode-session header pulled forward
// from MADR 0012 §1.4: OpenCode Go rejects requests without it (400
// MissingSessionID, 2026-09-26). Every generation request carries one id,
// fixed per provider instance, on both gateways and every route.
func TestOpencode_SessionHeader(t *testing.T) {
	var seen []string
	serve := func(fixture string) *httptest.Server {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seen = append(seen, r.Header.Get("x-opencode-session"))
			_, _ = w.Write([]byte(fixture))
		}))
		t.Cleanup(srv.Close)
		return srv
	}
	for _, tc := range []struct {
		gateway        llmprovider.ProviderID
		model, fixture string
	}{
		{llmprovider.ProviderOpencodeGo, "glm-5.3-flash", fxChat},
		{llmprovider.ProviderOpencodeZen, deepSeekV4Pro, fxChat},
		{llmprovider.ProviderOpencodeZen, "claude-sonnet-5", fxMessages},
	} {
		seen = nil
		srv := serve(tc.fixture)
		first := build(t, tc.gateway, apiKey(srv.URL, tc.model)...)
		second := build(t, tc.gateway, apiKey(srv.URL, tc.model)...)
		for _, p := range []llmprovider.Provider{first, first, second} {
			if _, err := p.Generate(context.Background(), text("hi")); err != nil {
				t.Fatalf("%s %s: Generate: %v", tc.gateway, tc.model, err)
			}
		}
		if len(seen) != 3 || seen[0] == "" {
			t.Fatalf("%s %s: sessions = %q, want 3 non-empty", tc.gateway, tc.model, seen)
		}
		if seen[0] != seen[1] {
			t.Errorf("%s %s: one provider sent two sessions %q and %q", tc.gateway, tc.model, seen[0], seen[1])
		}
		if seen[0] == seen[2] {
			t.Errorf("%s %s: two providers share session %q", tc.gateway, tc.model, seen[0])
		}
	}
}

func TestOpencode_ErrorClassification(t *testing.T) {
	tests := []struct {
		status  int
		wantErr error
	}{
		{http.StatusTooManyRequests, llmprovider.ErrRateLimited},
		{http.StatusUnauthorized, llmprovider.ErrAuthFailure},
		{http.StatusInternalServerError, llmprovider.ErrProviderUnavailable},
		{http.StatusBadRequest, llmprovider.ErrInvalidRequest},
	}
	for _, tc := range tests {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"type":"error","error":{"type":"AuthError","message":"x"}}`))
			}))
			defer srv.Close()
			_, err := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "claude-sonnet-5")...).Generate(context.Background(), text("hi"))
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want wrapping %v", err, tc.wantErr)
			}
			if tc.status == http.StatusTooManyRequests {
				var rl *llmprovider.APIError
				if !errors.As(err, &rl) || !errors.Is(rl.Kind, llmprovider.ErrRateLimited) || rl.RetryAfter != 0 {
					// The gateway sends no Retry-After (measured 2026-08-28).
					t.Errorf("err = %#v, want an *APIError of kind ErrRateLimited, RetryAfter 0", err)
				}
			}
			// Every classification names gateway/route, 429 included.
			if !strings.Contains(err.Error(), "opencode-zen/messages") {
				t.Errorf("error must name gateway/route for diagnosability: %v", err)
			}
		})
	}
}

// TestOpencode_WithRoute was TestOpencode_WithOpencodeRoute: the route option
// beats the table.
func TestOpencode_WithRoute(t *testing.T) {
	var path string
	srv := pathCapture(t, &path, fxChat)
	// gpt-5.5 is tabled as responses; the override must win.
	p := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "gpt-5.5", WithRoute(RouteChatCompletions))...)
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if path != "/chat/completions" {
		t.Errorf("path = %q, want /chat/completions", path)
	}
}

// TestOpencode_ConstructorErrors: an empty key is the gateway's public token,
// not an error (MADR 0012 §1.7); an unknown route is refused. The old
// unknown-gateway case has no successor: each constructor names its gateway.
func TestOpencode_ConstructorErrors(t *testing.T) {
	for _, gateway := range family {
		if _, err := newFor(gateway)(llmprovider.WithAPIKey(""), llmprovider.WithModel("m")); err != nil {
			t.Errorf("%s: empty api key: %v, want the public token", gateway, err)
		}
		if _, err := newFor(gateway)(llmprovider.WithModel("m")); err != nil {
			t.Errorf("%s: no credential: %v, want the public token", gateway, err)
		}
		_, err := newFor(gateway)(llmprovider.WithAPIKey("k"), WithRoute(Route("bogus")))
		if !errors.Is(err, llmprovider.ErrInvalidRequest) {
			t.Errorf("%s: invalid route override err = %v, want wrapping ErrInvalidRequest", gateway, err)
		}
	}
}

// TestOpencode_NoContinuation was TestOpencode_NoContinuer: chaining via
// previous_response_id returns HTTP 400 (measured 2026-08-28), so
// continuation is Unsupported, and refused before any request.
func TestOpencode_NoContinuation(t *testing.T) {
	var path string
	srv := pathCapture(t, &path, fxResponses)
	p := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "gpt-5.5")...)
	if got := p.Capabilities().Continuation; got != llmprovider.Unsupported {
		t.Errorf("Continuation = %v, want Unsupported: the gateway rejects previous_response_id with HTTP 400", got)
	}
	req := text("hi")
	req.PreviousResponseID = "resp_prev"
	if _, err := p.Generate(context.Background(), req); !errors.Is(err, llmprovider.ErrUnsupported) || path != "" {
		t.Errorf("err = %v after request %q, want ErrUnsupported and none", err, path)
	}
}

// TestOpencode_RetryOnRateLimit: WithRetry, which replaces GenerateWithRetry,
// retries a 429.
func TestOpencode_RetryOnRateLimit(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(fxMessages))
	}))
	defer srv.Close()
	p := llmprovider.WithRetry(build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "claude-sonnet-5")...),
		llmprovider.RetryPolicy{MaxAttempts: 2, BaseDelay: time.Millisecond})
	out, err := llmprovider.GenerateText(context.Background(), p, text("hi"))
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}
	if out != "hello" || calls != 2 {
		t.Errorf("out = %q after %d calls", out, calls)
	}
}
