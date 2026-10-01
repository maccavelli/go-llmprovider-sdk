package huggingface

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from llmprovider's huggingface_test.go, less its catalog tests, which
// stay there in huggingface_catalog_test.go, and the Hugging Face parts of
// api_error_message_test.go, identification_test.go and interface_test.go
// (0015-PLAN S7).

// TestHuggingFace_RequestShape pins the single endpoint this provider uses and
// fails loudly if /responses is ever requested — that endpoint returns HTTP 200
// with status:"failed" on auth errors, which would decode as a silent success.
func TestHuggingFace_RequestShape(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if r.URL.Path == "/responses" {
			t.Error("/responses must never be requested: it returns 200 on auth failure")
		}
		if got := r.Header.Get("Authorization"); got != "Bearer hf_test" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer hf_test")
		}
		_, _ = w.Write([]byte(fxHFChat))
	}))
	defer srv.Close()
	p := build(t, llmprovider.WithAPIKey("hf_test"), llmprovider.WithModel("openai/gpt-oss-20b"), llmprovider.WithBaseURL(srv.URL))
	out, err := llmprovider.GenerateText(context.Background(), p, text("hi"))
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}
	if out != "hello" {
		t.Errorf("output = %q", out)
	}
	if path != "/chat/completions" {
		t.Errorf("path = %q, want /chat/completions", path)
	}
	if p.ID() != llmprovider.ProviderHuggingFace {
		t.Errorf("ID() = %q", p.ID())
	}
}

func TestHuggingFace_ReasoningEffort(t *testing.T) {
	t.Run("thinking defaults to medium", func(t *testing.T) {
		var body map[string]any
		srv := captureServer(t, &body, fxHFChat)
		if _, err := build(t, apiKey(srv.URL, "openai/gpt-oss-20b")...).Generate(context.Background(), thinking(text("hi"))); err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if body["reasoning_effort"] != string(llmprovider.EffortMedium) {
			t.Errorf("reasoning_effort = %v, want medium", body["reasoning_effort"])
		}
	})

	// It was "WithReasoningEffort honoured"; it runs both ways now.
	for _, where := range []string{"construction", "request"} {
		t.Run("the effort honoured, at "+where, func(t *testing.T) {
			var body map[string]any
			srv := captureServer(t, &body, fxHFChat)
			var opts []llmprovider.Option
			req := text("hi")
			if where == "construction" {
				opts = append(opts, llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: llmprovider.EffortXHigh}))
				thinking(req)
			} else {
				req.Reasoning = &llmprovider.Reasoning{Effort: llmprovider.EffortXHigh}
			}
			if _, err := build(t, apiKey(srv.URL, "openai/gpt-oss-20b", opts...)...).Generate(context.Background(), req); err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if body["reasoning_effort"] != string(llmprovider.EffortXHigh) {
				t.Errorf("reasoning_effort = %v, want xhigh", body["reasoning_effort"])
			}
		})
	}

	t.Run("non-thinking path sends no reasoning_effort", func(t *testing.T) {
		var body map[string]any
		srv := captureServer(t, &body, fxHFChat)
		if _, err := build(t, apiKey(srv.URL, "openai/gpt-oss-20b")...).Generate(context.Background(), text("hi")); err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if _, present := body["reasoning_effort"]; present {
			t.Errorf("reasoning_effort must be absent: %v", body["reasoning_effort"])
		}
	})
}

func TestHuggingFace_ToolCall(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"choices":[{"message":{"role":"assistant","content":"",
		"tool_calls":[{"id":"c1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"SF\"}"}}]}}]}`)
	tool := llmprovider.Tool{Name: "get_weather", Description: "Get weather", Schema: map[string]any{"type": "object"}}
	call, err := llmprovider.GenerateToolCall(context.Background(), build(t, apiKey(srv.URL, "openai/gpt-oss-20b")...),
		&llmprovider.Request{Input: []llmprovider.Item{user("weather?")}, Tools: []llmprovider.Tool{tool}})
	if err != nil {
		t.Fatalf("GenerateToolCall: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(call.Arguments), &parsed); err != nil {
		t.Fatalf("arguments %q not valid JSON: %v", call.Arguments, err)
	}
	if parsed["city"] != "SF" {
		t.Errorf("arguments = %q", call.Arguments)
	}
	// Forced tool_choice must be present on this provider.
	tc, ok := body["tool_choice"].(map[string]any)
	if !ok {
		t.Fatalf("tool_choice missing: %v", body)
	}
	fn, _ := tc["function"].(map[string]any)
	if fn["name"] != "get_weather" {
		t.Errorf("tool_choice.function = %v", tc["function"])
	}
}

// TestHuggingFace_ErrorClassification uses the measured 401 envelope, whose
// `error` field is a plain string rather than the OpenAI object.
func TestHuggingFace_ErrorClassification(t *testing.T) {
	tests := []struct {
		status  int
		wantErr error
	}{
		{http.StatusUnauthorized, llmprovider.ErrAuthFailure},
		{http.StatusTooManyRequests, llmprovider.ErrRateLimited},
		{http.StatusInternalServerError, llmprovider.ErrProviderUnavailable},
		{http.StatusBadRequest, llmprovider.ErrInvalidRequest},
	}
	for _, tc := range tests {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"error":"Invalid username or password."}`))
			}))
			defer srv.Close()
			if _, err := build(t, apiKey(srv.URL, "openai/gpt-oss-20b")...).Generate(context.Background(), text("hi")); !errors.Is(err, tc.wantErr) {
				t.Errorf("err = %v, want wrapping %v", err, tc.wantErr)
			}
		})
	}
}

// TestHuggingFace_NoContinuation was TestHuggingFace_NoContinuer: Chat
// Completions is stateless, so continuation is Unsupported, and refused before
// any request.
func TestHuggingFace_NoContinuation(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, fxHFChat)
	p := build(t, apiKey(srv.URL, "openai/gpt-oss-20b")...)
	if got := p.Capabilities().Continuation; got != llmprovider.Unsupported {
		t.Errorf("Continuation = %v, want Unsupported", got)
	}
	req := text("hi")
	req.PreviousResponseID = "resp_prev"
	if _, err := p.Generate(context.Background(), req); !errors.Is(err, llmprovider.ErrUnsupported) || body != nil {
		t.Errorf("err = %v, body %v; want ErrUnsupported and no request", err, body)
	}
}

// TestHuggingFace_ErrorCarriesServiceMessage: the error keeps the service's
// own explanation and still matches the sentinel its status maps to (MADR 0012
// §1.1, 0013 B3). It was the huggingface row of
// TestProviders_ErrorCarriesServiceMessage.
func TestHuggingFace_ErrorCarriesServiceMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"model is unavailable"}}`))
	}))
	t.Cleanup(srv.Close)
	_, err := llmprovider.GenerateText(context.Background(), build(t, apiKey(srv.URL, "org/model")...), text("hello"))
	if err == nil || !strings.Contains(err.Error(), "model is unavailable") {
		t.Fatalf("error = %v, want it to carry the service's message", err)
	}
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("error = %v, want it to still match ErrInvalidRequest", err)
	}
}

var userAgentPattern = regexp.MustCompile(`^go-llmprovider-sdk/\S+ \(\w+; \w+\) go-llmprovider-sdk/\S+$`)

// TestHuggingFace_UserAgent: generation names this module honestly (MADR 0012
// §1.4). It was the huggingface row of TestIdentification_UserAgent.
func TestHuggingFace_UserAgent(t *testing.T) {
	var ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.UserAgent()
		_, _ = w.Write([]byte(fxHFChat))
	}))
	t.Cleanup(srv.Close)
	if _, err := build(t, apiKey(srv.URL, "org/model")...).Generate(context.Background(), text("hello")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !userAgentPattern.MatchString(ua) {
		t.Errorf("User-Agent = %q, want go-llmprovider-sdk/<version> (<os>; <arch>) go-llmprovider-sdk/<version>", ua)
	}
}

// TestHuggingFace_Capabilities is what the interface assertions declared.
func TestHuggingFace_Capabilities(t *testing.T) {
	p := build(t, llmprovider.WithAPIKey("k"), llmprovider.WithModel("m"))
	caps := p.Capabilities()
	if caps.Tools != llmprovider.Supported || caps.ForcedToolChoice != llmprovider.Supported ||
		caps.Reasoning != llmprovider.BestEffort || caps.Continuation != llmprovider.Unsupported ||
		caps.NativeStreaming != llmprovider.Unsupported {
		t.Errorf("Capabilities = %+v", caps)
	}
	if _, ok := p.(llmprovider.ModelLister); !ok {
		t.Error("the provider is not a ModelLister")
	}
}
