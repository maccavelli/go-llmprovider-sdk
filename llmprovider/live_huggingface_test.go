//go:build live_gateways

package llmprovider_test

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/huggingface"
)

// Hugging Face's live tests, through its own package (0015-PLAN S7). They
// REQUIRE HF_TOKEN: Hugging Face reports is_free:false for all offerings, so
// there is no credential-free path. The raw-HTTP probes of the router's shapes
// stay in live_gateways_test.go.

// hfProbedOn is llmprovider's wireShapesProbedOnHuggingFace, for messages.
const hfProbedOn = "2026-08-29"

func liveHuggingFace(t *testing.T, model string, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := huggingface.New(append([]llmprovider.Option{llmprovider.WithAPIKey(llmprovider.LiveEnvKey(t, "HF_TOKEN")),
		llmprovider.WithModel(model)}, opts...)...)
	if err != nil {
		t.Fatalf("huggingface.New: %v", err)
	}
	return p
}

func TestLive_HuggingFaceChatCompletions(t *testing.T) {
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	out, err := llmprovider.GenerateText(ctx, liveHuggingFace(t, "openai/gpt-oss-20b"), userText("Reply with only the word ALPHA"))
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}
	if strings.TrimSpace(out) == "" {
		t.Errorf("empty output (probed %s)", hfProbedOn)
	}
}

// TestLive_HuggingFaceToolChoices pins the unmeasured tool choices: "required"
// makes a call, and "none" makes none (0015-PLAN S7).
func TestLive_HuggingFaceToolChoices(t *testing.T) {
	tool := llmprovider.Tool{Name: "get_weather", Description: "Get the weather for a city", Schema: map[string]any{
		"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}, "required": []string{"city"}}}
	for _, tc := range []struct {
		choice   llmprovider.ToolChoice
		wantCall bool
	}{{llmprovider.ToolChoiceRequired, true}, {llmprovider.ToolChoiceNone, false}} {
		t.Run(string(tc.choice), func(t *testing.T) {
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			req := userText("What is the weather in Paris?")
			req.Tools, req.ToolChoice = []llmprovider.Tool{tool}, tc.choice
			res, err := liveHuggingFace(t, "openai/gpt-oss-120b").Generate(ctx, req)
			llmprovider.SkipIfTransient(t, err)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			called := false
			for _, item := range res.Output {
				if _, ok := item.(llmprovider.FunctionCallItem); ok {
					called = true
				}
			}
			if called != tc.wantCall {
				t.Fatalf("a call came back: %t, want %t (%+v)", called, tc.wantCall, res.Output)
			}
		})
	}
}
