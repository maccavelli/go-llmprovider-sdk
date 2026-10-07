//go:build live_gateways

package llmprovider_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/claude"
)

// Claude's live tests, through its own package (0015-PLAN S7). They REQUIRE
// ANTHROPIC_API_KEY and skip without it.

func liveClaude(t *testing.T, key, model string, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := claude.New(append([]llmprovider.Option{llmprovider.WithAPIKey(key), llmprovider.WithModel(model)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// TestLive_StaticClaudeServed pins MADR 0013 B10: every static Claude id
// answers on the Messages API. The static catalog is what the wizard offers
// when the listing fails, so a retired id there is a dead end.
func TestLive_StaticClaudeServed(t *testing.T) {
	key := llmprovider.LiveEnvKey(t, "ANTHROPIC_API_KEY")
	for _, model := range catalog.Static(llmprovider.ProviderClaude) {
		t.Run(model, func(t *testing.T) {
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			_, err := llmprovider.GenerateText(ctx, liveClaude(t, key, model, llmprovider.WithMaxTokens(16)),
				userText("Reply with only the word ALPHA"))
			llmprovider.SkipIfTransient(t, err)
			if err != nil {
				t.Errorf("%s: %v", model, err)
			}
		})
	}
}

// TestLive_ClaudeThinkingShapes: a budget-only model (Haiku 4.5) and two
// adaptive-only models accept the utility and capable efforts.
func TestLive_ClaudeThinkingShapes(t *testing.T) {
	key := llmprovider.LiveEnvKey(t, "ANTHROPIC_API_KEY")
	for _, model := range []string{"claude-haiku-4-5", "claude-sonnet-5", "claude-opus-4-8"} {
		for _, effort := range []llmprovider.Effort{llmprovider.EffortLow, ""} {
			t.Run(model+"/"+string(effort), func(t *testing.T) {
				ctx, cancel := llmprovider.LiveCtx(t)
				defer cancel()
				p := liveClaude(t, key, model, llmprovider.WithMaxTokens(2048))
				req := userText("Reply with only the word ALPHA")
				req.Reasoning = &llmprovider.Reasoning{Effort: effort}
				out, err := llmprovider.GenerateText(ctx, p, req)
				llmprovider.SkipIfTransient(t, err)
				if err != nil {
					t.Fatalf("Generate: %v", err)
				}
				if !strings.Contains(strings.ToUpper(out), "ALPHA") {
					t.Errorf("output %q does not contain ALPHA", out)
				}
			})
		}
	}
}

// TestLive_ToolRoundTripClaude sends a completed tool call and its result,
// and requires the model to answer from the result (MADR 0012 §2). It was the
// claude row of TestLive_ToolRoundTrip.
func TestLive_ToolRoundTripClaude(t *testing.T) {
	key := llmprovider.LiveEnvKey(t, "ANTHROPIC_API_KEY")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	res, err := liveClaude(t, key, "claude-haiku-4-5").Generate(ctx, &llmprovider.Request{Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: "user", Text: "What is the weather in Paris? Use the tool."},
		llmprovider.FunctionCallItem{CallID: "call_rt_1", Name: "get_weather", Arguments: `{"city":"Paris"}`},
		llmprovider.FunctionCallOutputItem{CallID: "call_rt_1", Output: `{"forecast":"sunny, 21C"}`},
	}})
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if text := strings.ToLower(res.OutputText()); !strings.Contains(text, "sunny") && !strings.Contains(text, "21") {
		t.Fatalf("reply %q does not use the tool result", res.OutputText())
	}
}

// TestLive_SystemMessageClaude: a system instruction reaches the model, and
// the model follows it. It was the claude row of TestLive_SystemMessage.
func TestLive_SystemMessageClaude(t *testing.T) {
	key := llmprovider.LiveEnvKey(t, "ANTHROPIC_API_KEY")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	res, err := liveClaude(t, key, "claude-haiku-4-5").Generate(ctx, &llmprovider.Request{Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: "system", Text: "You only ever reply in French."},
		llmprovider.MessageItem{Role: "user", Text: "Say hello in one word, nothing else."},
	}})
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if text := strings.ToLower(res.OutputText()); !strings.Contains(text, "bonjour") && !strings.Contains(text, "salut") &&
		!strings.Contains(text, "coucou") {
		t.Fatalf("reply %q does not follow the system instruction", res.OutputText())
	}
}
