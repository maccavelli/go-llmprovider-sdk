package opencode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from llmprovider's opencode_conventions_test.go (0015-PLAN S7).

// conventionsBody runs one call against a stub Go gateway and returns the
// request body it sent.
func conventionsBody(t *testing.T, model string, reasons bool, opts ...llmprovider.Option) map[string]any {
	t.Helper()
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/messages"):
			_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}]}`))
		default:
			_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`))
		}
	}))
	t.Cleanup(srv.Close)
	req := text("hi")
	if reasons {
		thinking(req)
	}
	if _, err := build(t, llmprovider.ProviderOpencodeGo, apiKey(srv.URL, model, opts...)...).Generate(context.Background(), req); err != nil {
		t.Fatalf("generate: %v", err)
	}
	return body
}

// TestOpencodeResponses_SendsStoreFalse: OpenCode's client sets store false
// for every @ai-sdk/openai model (transform.ts:1235-1243).
func TestOpencodeResponses_SendsStoreFalse(t *testing.T) {
	body := conventionsBody(t, "gpt-6-luna", false)
	if v, ok := body["store"]; !ok || v != false {
		t.Fatalf("store = %v (present %t), want false", v, ok)
	}
}

// TestOpencodeMessages_MinimaxM3ThinksAdaptive: MiniMax's Anthropic interface
// takes thinking.type "adaptive", with no budget, and the effort configured,
// which minimax-m3 accepted on 2026-10-25 (0062-MADR Q12 (a); R2 removed).
func TestOpencodeMessages_MinimaxM3ThinksAdaptive(t *testing.T) {
	body := conventionsBody(t, "minimax-m3", true, effort(llmprovider.EffortHigh))
	th, _ := body["thinking"].(map[string]any)
	if len(th) != 1 || th["type"] != "adaptive" {
		t.Fatalf("thinking = %v, want exactly {type: adaptive}", body["thinking"])
	}
	if cfg, _ := body["output_config"].(map[string]any); len(cfg) != 1 || cfg["effort"] != "high" {
		t.Errorf("output_config = %v, want {effort: high}", body["output_config"])
	}
}

// TestOpencodeMessages_OtherModelsThinkAdaptive: a non-Claude model on the
// Messages route, with no facts, thinks adaptively, which qwen3.8-flash
// accepted on 2026-10-25 (0062-MADR Q12 (a); R1 removed); a Budget keeps the
// enabled budget.
func TestOpencodeMessages_OtherModelsThinkAdaptive(t *testing.T) {
	body := conventionsBody(t, "qwen3.8-flash", true)
	if th, _ := body["thinking"].(map[string]any); len(th) != 1 || th["type"] != "adaptive" {
		t.Fatalf("thinking = %v, want exactly {type: adaptive}", body["thinking"])
	}
	body = conventionsBody(t, "qwen3.8-flash", true, llmprovider.WithReasoning(&llmprovider.Reasoning{Budget: 2000}))
	if th, _ := body["thinking"].(map[string]any); th["type"] != "enabled" || th["budget_tokens"] != float64(2000) {
		t.Fatalf("thinking = %v with a budget, want an enabled budget of 2000", body["thinking"])
	}
}
