package kilo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from llmprovider's kilo_test.go (0015-PLAN S7). Its catalog tests
// stay there, in kilo_catalog_test.go. Each old method is the Request it sent.

func TestKilo_RequestShape(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		switch r.URL.Path {
		case "/responses", "/messages":
			t.Errorf("%s must never be requested: Kilo translates formats, so one "+
				"route reaches the whole catalog", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer kilo_test" {
			t.Errorf("Authorization = %q", got)
		}
		_, _ = w.Write([]byte(fxKiloChat))
	}))
	defer srv.Close()
	p := build(t, llmprovider.WithAPIKey("kilo_test"), llmprovider.WithModel("kilo-auto/free"), llmprovider.WithBaseURL(srv.URL))
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if path != "/chat/completions" {
		t.Errorf("path = %q, want /chat/completions", path)
	}
	if p.ID() != llmprovider.ProviderKilo {
		t.Errorf("ID() = %q, want %q", p.ID(), llmprovider.ProviderKilo)
	}
}

// assertKiloReasoning checks the two reasoning fields of a Kilo request body:
// wantEffort "" means reasoning_effort is absent, and a nil wantReasoning
// means the reasoning object is absent.
func assertKiloReasoning(t *testing.T, body map[string]any, wantEffort string, wantReasoning map[string]any) {
	t.Helper()
	effort, hasEffort := body[paramReasoningEffort]
	switch {
	case wantEffort == "" && hasEffort:
		t.Errorf("reasoning_effort = %v, want absent", effort)
	case wantEffort != "" && effort != wantEffort:
		t.Errorf("reasoning_effort = %v, want %q", effort, wantEffort)
	}
	reasoning, hasReasoning := body[paramReasoning]
	switch {
	case wantReasoning == nil && hasReasoning:
		t.Errorf("reasoning = %v, want absent", reasoning)
	case wantReasoning != nil && !reflect.DeepEqual(reasoning, wantReasoning):
		t.Errorf("reasoning = %v, want %v", reasoning, wantReasoning)
	}
}

// TestKilo_SupportedParameterGating covers the capability gating that makes
// Kilo's per-model supported_parameters actionable. With no effort
// configured, the reasoning path sends Kilo's reasoning object
// {"enabled": true} when "reasoning" is accepted (or capabilities are
// unknown), and reasoning_effort only when the model lists it and not
// "reasoning" (MADR 0009 §6).
func TestKilo_SupportedParameterGating(t *testing.T) {
	tool := llmprovider.Tool{Name: "get_weather", Schema: map[string]any{"type": "object"}}
	enabled := map[string]any{"enabled": true}
	tests := []struct {
		name                  string
		caps                  []string
		wantTools, wantChoice bool
		wantEffort            string
		wantReasoning         map[string]any
	}{
		{"nil caps sends the reasoning object", nil, true, true, "", enabled},
		{"tools only", []string{paramTools}, true, false, "", nil},
		{"tools and tool_choice", []string{paramTools, paramToolChoice}, true, true, "", nil},
		{"reasoning_effort without reasoning",
			[]string{paramTools, paramToolChoice, paramReasoningEffort}, true, true, "medium", nil},
		{"reasoning", []string{paramTools, paramReasoning}, true, false, "", enabled},
		{"reasoning and reasoning_effort",
			[]string{paramTools, paramReasoning, paramReasoningEffort}, true, false, "", enabled},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body map[string]any
			srv := captureServer(t, &body, fxKiloChat)
			var opts []llmprovider.Option
			if tc.caps != nil {
				opts = append(opts, WithCapabilities(tc.caps...))
			}
			// This test inspects the request body; the fixture returns no tool_calls.
			if _, err := build(t, apiKey(srv.URL, "kilo-auto/free", opts...)...).Generate(context.Background(),
				thinking(withTool(text("hi"), tool))); err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if _, ok := body[paramTools]; ok != tc.wantTools {
				t.Errorf("tools present = %v, want %v", ok, tc.wantTools)
			}
			if _, ok := body[paramToolChoice]; ok != tc.wantChoice {
				t.Errorf("tool_choice present = %v, want %v", ok, tc.wantChoice)
			}
			assertKiloReasoning(t, body, tc.wantEffort, tc.wantReasoning)
		})
	}
}

// TestKilo_ReasoningEffortConfigured pins MADR 0009 §6 with an effort set:
// {"effort": …} in the reasoning object, reasoning_effort only for models that
// list it and not "reasoning", and nothing on a plain call. Each case runs
// with the effort set at construction, as WithReasoningEffort did, and on the
// request.
func TestKilo_ReasoningEffortConfigured(t *testing.T) {
	low := map[string]any{"effort": "low"}
	tests := []struct {
		name          string
		caps          []string
		wantEffort    string
		wantReasoning map[string]any
	}{
		{"nil caps", nil, "", low},
		{"reasoning", []string{paramTools, paramReasoning}, "", low},
		{"reasoning_effort only", []string{paramTools, paramReasoningEffort}, "low", nil},
		{"neither", []string{paramTools}, "", nil},
	}
	for _, tc := range tests {
		for _, where := range []string{"construction", "request"} {
			t.Run(tc.name+"/"+where, func(t *testing.T) {
				var body map[string]any
				srv := captureServer(t, &body, fxKiloChat)
				var opts []llmprovider.Option
				if tc.caps != nil {
					opts = append(opts, WithCapabilities(tc.caps...))
				}
				req := text("hi")
				if where == "construction" {
					opts = append(opts, llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: llmprovider.EffortLow}))
					thinking(req)
				} else {
					req.Reasoning = &llmprovider.Reasoning{Effort: llmprovider.EffortLow}
				}
				if _, err := build(t, apiKey(srv.URL, "kilo-auto/free", opts...)...).Generate(context.Background(), req); err != nil {
					t.Fatalf("Generate: %v", err)
				}
				assertKiloReasoning(t, body, tc.wantEffort, tc.wantReasoning)
			})
		}
	}
	t.Run("plain call", func(t *testing.T) {
		var body map[string]any
		srv := captureServer(t, &body, fxKiloChat)
		p := build(t, apiKey(srv.URL, "kilo-auto/free")...)
		if _, err := p.Generate(context.Background(), text("hi")); err != nil {
			t.Fatalf("Generate: %v", err)
		}
		assertKiloReasoning(t, body, "", nil)
	})
}

// TestWithCapabilities was TestWithKiloCapabilities: the option, not an
// internal side effect, is what populates the capabilities.
func TestWithCapabilities(t *testing.T) {
	p := build(t, llmprovider.WithAPIKey("k"), WithCapabilities(paramTools)).(*provider)
	if !p.supports(paramTools) {
		t.Error("declared capability must be supported")
	}
	if p.supports(paramToolChoice) {
		t.Error("undeclared capability must not be supported when caps is set")
	}
	unknown := build(t, llmprovider.WithAPIKey("k")).(*provider)
	if !unknown.supports(paramTools) || !unknown.supports(paramToolChoice) {
		t.Error("nil caps means unknown: every parameter must be sent")
	}
}

func TestKilo_Generate(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, fxKiloChat)
	resp, err := build(t, apiKey(srv.URL, "kilo-auto/free")...).Generate(context.Background(), text("hi"))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.Output) != 2 {
		t.Fatalf("expected reasoning + message, got %d items", len(resp.Output))
	}
	if _, ok := resp.Output[0].(llmprovider.ReasoningItem); !ok {
		t.Errorf("output[0] = %T, want ReasoningItem (from message.reasoning)", resp.Output[0])
	}
	if got := resp.OutputText(); got != "ALPHA" {
		t.Errorf("OutputText() = %q", got)
	}
}

func TestKilo_ErrorClassification(t *testing.T) {
	// The measured 401 envelope, which is Kilo-specific rather than OpenAI-shaped.
	const kiloAuthErr = `{"error":{"code":"PAID_MODEL_AUTH_REQUIRED","message":"You need to sign in to use this model."},"error_type":"paid_model_auth_required"}`
	tests := []struct {
		status  int
		wantErr error
	}{
		{http.StatusUnauthorized, llmprovider.ErrAuthFailure},
		{http.StatusPaymentRequired, llmprovider.ErrInvalidRequest}, // documented "insufficient balance"
		{http.StatusTooManyRequests, llmprovider.ErrRateLimited},
		{http.StatusInternalServerError, llmprovider.ErrProviderUnavailable},
		{http.StatusBadRequest, llmprovider.ErrInvalidRequest},
	}
	for _, tc := range tests {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(kiloAuthErr))
			}))
			defer srv.Close()
			if _, err := build(t, apiKey(srv.URL, "kilo-auto/free")...).Generate(context.Background(), text("hi")); !errors.Is(err, tc.wantErr) {
				t.Errorf("err = %v, want wrapping %v", err, tc.wantErr)
			}
		})
	}
}

// TestKilo_NoContinuation was TestKilo_NoContinuer: Chat Completions is
// stateless, so continuation is Unsupported, and refused before any request.
func TestKilo_NoContinuation(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, fxKiloChat)
	p := build(t, apiKey(srv.URL, "kilo-auto/free")...)
	if got := p.Capabilities().Continuation; got != llmprovider.Unsupported {
		t.Errorf("Continuation = %v, want Unsupported", got)
	}
	req := text("hi")
	req.PreviousResponseID = "resp_prev"
	if _, err := p.Generate(context.Background(), req); !errors.Is(err, llmprovider.ErrUnsupported) || body != nil {
		t.Errorf("err = %v, body %v; want ErrUnsupported and no request", err, body)
	}
}
