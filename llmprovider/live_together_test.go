//go:build live_gateways

package llmprovider_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/together"
)

// Together's live generation tests, through its own package (0015-PLAN S7).
// They need LLMPROVIDER_LIVE_TOGETHER=1 and TOGETHER_API_KEY: every call is
// billed. The listing test, which reads the listing directly, is in
// live_together_listing_test.go.

var togetherWeatherTool = llmprovider.Tool{Name: "get_weather", Description: "Weather for a city",
	Schema: map[string]any{"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}}}

func liveTogether(t *testing.T, model string, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := together.New(append([]llmprovider.Option{llmprovider.WithAPIKey(llmprovider.LiveTogetherKey(t)),
		llmprovider.WithModel(model), llmprovider.WithMaxTokens(512)}, opts...)...)
	if err != nil {
		t.Fatalf("together.New: %v", err)
	}
	return p
}

// TestLive_TogetherWire confirms MADR 0017 D1's wire shapes against the
// service: text and a forced tool, reasoning {"enabled": true} on a
// toggleable model, and reasoning_effort on gpt-oss.
func TestLive_TogetherWire(t *testing.T) {
	for _, tc := range []struct {
		name, model string
		reasoning   *llmprovider.Reasoning
		tool        bool
	}{
		{name: "text", model: "openai/gpt-oss-120b"},
		{name: "forced tool", model: "openai/gpt-oss-120b", tool: true},
		{name: "thinking toggle", model: "deepseek-ai/DeepSeek-V4.1-Flash", reasoning: &llmprovider.Reasoning{}},
		{name: "thinking with effort", model: "openai/gpt-oss-120b", reasoning: &llmprovider.Reasoning{Effort: llmprovider.EffortHigh}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			p := liveTogether(t, tc.model)
			var out string
			var err error
			if tc.tool {
				var call llmprovider.FunctionCallItem
				call, err = llmprovider.GenerateToolCall(ctx, p, &llmprovider.Request{Input: []llmprovider.Item{
					llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "What is the weather in Paris? Use the tool."}},
					Tools: []llmprovider.Tool{togetherWeatherTool}})
				out = call.Arguments
			} else {
				req := userText("Reply with only the word ALPHA")
				req.Reasoning = tc.reasoning
				out, err = llmprovider.GenerateText(ctx, p, req)
			}
			if errors.Is(err, llmprovider.ErrRateLimited) {
				t.Skipf("rate limited: %v", err)
			}
			if err != nil {
				t.Fatalf("%s: %v", tc.model, err)
			}
			if tc.tool && !strings.Contains(out, "Paris") {
				t.Errorf("tool arguments %q do not name Paris", out)
			}
			if !tc.tool && !strings.Contains(strings.ToUpper(out), "ALPHA") {
				t.Errorf("reply %q does not contain ALPHA", out)
			}
		})
	}
}

// TestLive_TogetherToolChoices pins the tool choices: "required" makes a call,
// and "none" makes none (0015-PLAN S7). "required" runs on
// DeepSeek-V4.1-Flash: gpt-oss-120b answers it with HTTP 500
// (TestLive_TogetherRequiredFailsOnGptOss; 0017-PLAN, deviation of
// 2026-10-03).
func TestLive_TogetherToolChoices(t *testing.T) {
	for _, tc := range []struct {
		choice   llmprovider.ToolChoice
		model    string
		wantCall bool
	}{
		{llmprovider.ToolChoiceRequired, "deepseek-ai/DeepSeek-V4.1-Flash", true},
		{llmprovider.ToolChoiceNone, "openai/gpt-oss-120b", false},
	} {
		t.Run(string(tc.choice), func(t *testing.T) {
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			req := userText("What is the weather in Paris?")
			req.Tools, req.ToolChoice = []llmprovider.Tool{togetherWeatherTool}, tc.choice
			res, err := liveTogether(t, tc.model).Generate(ctx, req)
			if errors.Is(err, llmprovider.ErrRateLimited) {
				t.Skipf("rate limited: %v", err)
			}
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

// TestLive_TogetherRequiredFailsOnGptOss pins a measured failure (2026-10-03,
// 4 runs of 4): Together answers tool_choice "required" on openai/gpt-oss-120b
// with HTTP 500, ErrProviderUnavailable, while it honours a named tool there
// and "required" on other models. It fails once Together fixes it, so the
// degradation in together's package doc can be removed.
func TestLive_TogetherRequiredFailsOnGptOss(t *testing.T) {
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	req := userText("What is the weather in Paris?")
	req.Tools, req.ToolChoice = []llmprovider.Tool{togetherWeatherTool}, llmprovider.ToolChoiceRequired
	_, err := liveTogether(t, "openai/gpt-oss-120b").Generate(ctx, req)
	if errors.Is(err, llmprovider.ErrRateLimited) {
		t.Skipf("rate limited: %v", err)
	}
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Fatalf("Generate = %v, want the HTTP 500, an error matching ErrProviderUnavailable", err)
	}
}
