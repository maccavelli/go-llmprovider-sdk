package ollama

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

// Ported from llmprovider's ollama_test.go, and the Ollama parts of
// api_error_message_test.go, identification_test.go and interface_test.go
// (0015-PLAN S7).

// TestOllama_NoAuthHeader pins the property that makes Ollama the only
// credential-free provider: it requires no key and ignores any sent, so this
// provider sends none at all — with no key, an empty one AND a non-empty one.
func TestOllama_NoAuthHeader(t *testing.T) {
	for name, opts := range map[string][]llmprovider.Option{
		"no key":    nil,
		"key=":      {llmprovider.WithAPIKey("")},
		"key=other": {llmprovider.WithAPIKey("ignored-by-ollama")},
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != "" {
					t.Errorf("Authorization must never be sent to Ollama, got %q", got)
				}
				if r.URL.Path != "/v1/chat/completions" {
					t.Errorf("path = %q, want /v1/chat/completions", r.URL.Path)
				}
				_, _ = w.Write([]byte(fxOllamaChat))
			}))
			defer srv.Close()
			if _, err := build(t, local(srv.URL, "llama3.2:latest", opts...)...).Generate(context.Background(), text("hi")); err != nil {
				t.Fatalf("Generate: %v", err)
			}
		})
	}
}

// TestOllama_NeverForcesToolChoice pins the documented limitation: Ollama
// supports tools but not tool_choice, so a forced call would 400.
func TestOllama_NeverForcesToolChoice(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, fxOllamaChat)
	tool := llmprovider.Tool{Name: "get_weather", Schema: map[string]any{"type": "object"}}
	req := text("hi")
	req.Tools, req.ToolChoice = []llmprovider.Tool{tool}, llmprovider.ForceTool(tool.Name)
	if _, err := build(t, local(srv.URL, "llama3.2:latest")...).Generate(context.Background(), req); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, ok := body["tools"]; !ok {
		t.Error("tools must be offered")
	}
	if _, ok := body["tool_choice"]; ok {
		t.Errorf("tool_choice must NEVER be sent: Ollama does not support it (got %v)", body["tool_choice"])
	}
}

// TestOllama_ClampsXHighEffort: Ollama's vocabulary tops out at "max", not the
// "xhigh" this package models. The effort was WithReasoningEffort's; it runs
// at construction and on the request now.
func TestOllama_ClampsXHighEffort(t *testing.T) {
	tests := []struct {
		configured llmprovider.Effort
		want       string
	}{
		{llmprovider.EffortXHigh, maxEffort},
		{llmprovider.EffortMedium, string(llmprovider.EffortMedium)},
		{llmprovider.EffortLow, string(llmprovider.EffortLow)},
		{"", string(llmprovider.EffortMedium)}, // default
	}
	for _, where := range []string{"construction", "request"} {
		for _, tc := range tests {
			var body map[string]any
			srv := captureServer(t, &body, fxOllamaChat)
			var opts []llmprovider.Option
			req := text("hi")
			if where == "construction" && tc.configured != "" {
				opts = append(opts, llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: tc.configured}))
				thinking(req)
			} else {
				req.Reasoning = &llmprovider.Reasoning{Effort: tc.configured}
			}
			if _, err := build(t, local(srv.URL, "llama3.2:latest", opts...)...).Generate(context.Background(), req); err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if got := body["reasoning_effort"]; got != tc.want {
				t.Errorf("at %s: configured %q -> reasoning_effort %v, want %q", where, tc.configured, got, tc.want)
			}
		}
	}
}

func TestOllama_Generate(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, fxOllamaChat)
	p := build(t, local(srv.URL, "llama3.2:latest")...)
	out, err := llmprovider.GenerateText(context.Background(), p, text("hi"))
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}
	if out != "ALPHA" {
		t.Errorf("output = %q", out)
	}
	if p.ID() != llmprovider.ProviderOllama {
		t.Errorf("ID() = %q", p.ID())
	}
	// Non-thinking path sends no reasoning parameter.
	if _, ok := body["reasoning_effort"]; ok {
		t.Error("reasoning_effort must be absent on the plain path")
	}
}

// TestNew_NeedsNoKey was TestOllama_EmptyKeyAccepted: unlike every other
// provider, no key, or an empty one, is valid here.
func TestNew_NeedsNoKey(t *testing.T) {
	for name, opts := range map[string][]llmprovider.Option{
		"none":  {llmprovider.WithModel("llama3.2:latest")},
		"empty": {llmprovider.WithAPIKey(""), llmprovider.WithModel("llama3.2:latest")},
	} {
		if _, err := New(opts...); err != nil {
			t.Errorf("%s: New must succeed: %v", name, err)
		}
	}
}

func TestOllama_ErrorClassification(t *testing.T) {
	tests := []struct {
		status  int
		wantErr error
	}{
		{http.StatusTooManyRequests, llmprovider.ErrRateLimited},
		{http.StatusInternalServerError, llmprovider.ErrProviderUnavailable},
		{http.StatusBadRequest, llmprovider.ErrInvalidRequest},
	}
	for _, tc := range tests {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
			}))
			defer srv.Close()
			_, err := build(t, local(srv.URL, "llama3.2:latest")...).Generate(context.Background(), text("hi"))
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want wrapping %v", err, tc.wantErr)
			}
			if tc.status != http.StatusTooManyRequests && !strings.Contains(err.Error(), string(llmprovider.ProviderOllama)) {
				t.Errorf("error must name the provider: %v", err)
			}
		})
	}
}

// TestOllama_NoContinuation was TestOllama_NoContinuer: Chat Completions is
// stateless, so continuation is Unsupported, and refused before any request.
func TestOllama_NoContinuation(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, fxOllamaChat)
	p := build(t, local(srv.URL, "llama3.2:latest")...)
	if got := p.Capabilities().Continuation; got != llmprovider.Unsupported {
		t.Errorf("Continuation = %v, want Unsupported", got)
	}
	req := text("hi")
	req.PreviousResponseID = "resp_prev"
	if _, err := p.Generate(context.Background(), req); !errors.Is(err, llmprovider.ErrUnsupported) || body != nil {
		t.Errorf("err = %v, body %v; want ErrUnsupported and no request", err, body)
	}
}

// TestOllama_ErrorCarriesServiceMessage: the error keeps the service's own
// explanation and still matches the sentinel its status maps to (MADR 0012
// §1.1, 0013 B3). It was the ollama row of
// TestProviders_ErrorCarriesServiceMessage.
func TestOllama_ErrorCarriesServiceMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"model is unavailable"}}`))
	}))
	t.Cleanup(srv.Close)
	_, err := llmprovider.GenerateText(context.Background(), build(t, local(srv.URL, "llama3")...), text("hello"))
	if err == nil || !strings.Contains(err.Error(), "model is unavailable") {
		t.Fatalf("error = %v, want it to carry the service's message", err)
	}
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("error = %v, want it to still match ErrInvalidRequest", err)
	}
}

var userAgentPattern = regexp.MustCompile(`^go-llmprovider-sdk/\S+ \(\w+; \w+\) go-llmprovider-sdk/\S+$`)

// TestOllama_UserAgent: generation names this module honestly (MADR 0012
// §1.4). It was the ollama row of TestIdentification_UserAgent.
func TestOllama_UserAgent(t *testing.T) {
	var ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.UserAgent()
		_, _ = w.Write([]byte(fxOllamaChat))
	}))
	t.Cleanup(srv.Close)
	if _, err := build(t, local(srv.URL, "llama3")...).Generate(context.Background(), text("hello")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !userAgentPattern.MatchString(ua) {
		t.Errorf("User-Agent = %q, want go-llmprovider-sdk/<version> (<os>; <arch>) go-llmprovider-sdk/<version>", ua)
	}
}

// TestOllama_Capabilities is what the interface assertions declared.
func TestOllama_Capabilities(t *testing.T) {
	p := build(t, llmprovider.WithModel("m"))
	caps := p.Capabilities()
	if caps.Tools != llmprovider.Supported || caps.ForcedToolChoice != llmprovider.BestEffort ||
		caps.Reasoning != llmprovider.BestEffort || caps.Continuation != llmprovider.Unsupported ||
		caps.NativeStreaming != llmprovider.Unsupported {
		t.Errorf("Capabilities = %+v", caps)
	}
	if _, ok := p.(llmprovider.ModelLister); !ok {
		t.Error("the provider is not a ModelLister")
	}
}
