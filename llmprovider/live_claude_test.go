//go:build live_gateways

package llmprovider_test

import (
	"context"
	"errors"
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

// TestLive_ClaudeThinkingShapes: a budget-only model (Sonnet 4.6) and two
// adaptive-only models accept the utility and capable efforts. Haiku 4.5,
// the budget model before, is retiring (0034-MADR D5).
func TestLive_ClaudeThinkingShapes(t *testing.T) {
	key := llmprovider.LiveEnvKey(t, "ANTHROPIC_API_KEY")
	for _, model := range []string{"claude-sonnet-4-6", "claude-sonnet-5", "claude-opus-4-8"} {
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
// claude row of TestLive_ToolRoundTrip, on Haiku 4.5 until 0034-MADR D5.
func TestLive_ToolRoundTripClaude(t *testing.T) {
	key := llmprovider.LiveEnvKey(t, "ANTHROPIC_API_KEY")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	res, err := liveClaude(t, key, "claude-haiku-5-5").Generate(ctx, &llmprovider.Request{Input: []llmprovider.Item{
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
// the model follows it. It was the claude row of TestLive_SystemMessage, on
// Haiku 4.5 until 0034-MADR D5.
func TestLive_SystemMessageClaude(t *testing.T) {
	key := llmprovider.LiveEnvKey(t, "ANTHROPIC_API_KEY")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	res, err := liveClaude(t, key, "claude-haiku-5-5").Generate(ctx, &llmprovider.Request{Input: []llmprovider.Item{
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

// refusingClaude0034 is 0034-MADR D1's table, written out here so that a table
// that lost a model fails this test rather than skipping it.
var refusingClaude0034 = map[string]bool{
	"claude-fable-5-1": true, "claude-mythos-5-1": true, "claude-opus-5-5": true, "claude-sonnet-5-5": true,
}

// TestLive_ClaudeForcedChoiceDegrades (0034-MADR D5, D8): a forced named tool,
// as GenerateToolCall sends it, on each menu model that refuses a forced tool
// choice, returns the call, or no call (ErrIncomplete), and never the 400.
// The SDK knew the model refuses before it sent, so nothing is learned: the
// response carries no WarningForcedChoiceLearned. Billed: one request per
// model, of at most 512 output tokens.
func TestLive_ClaudeForcedChoiceDegrades(t *testing.T) {
	key := llmprovider.LiveEnvKey(t, "ANTHROPIC_API_KEY")
	var models []string
	for _, model := range catalog.Static(llmprovider.ProviderClaude) {
		if refusingClaude0034[model] {
			models = append(models, model)
		}
	}
	if len(models) == 0 {
		t.Fatal("no model in the Claude menu refuses a forced tool choice; 0034-MADR D5 put two there")
	}
	tool := llmprovider.Tool{Name: "record_city", Description: "Records a city name.",
		Schema: map[string]any{"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}, "required": []any{"city"}}}
	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			req := userText("Record the city Paris.")
			req.Tools, req.ToolChoice = []llmprovider.Tool{tool}, llmprovider.ForceTool(tool.Name)
			resp, err := liveClaude(t, key, model, llmprovider.WithMaxTokens(512)).Generate(ctx, req)
			llmprovider.SkipIfTransient(t, err)
			if errors.Is(err, llmprovider.ErrInvalidRequest) {
				t.Fatalf("%s refused the forced tool: %v", model, err)
			}
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			for _, w := range resp.Warnings {
				if w.Kind == llmprovider.WarningForcedChoiceLearned {
					t.Errorf("%s was learned, not known: %s", model, w.Detail)
				}
			}
			call := false
			for _, item := range resp.Output {
				if c, ok := item.(llmprovider.FunctionCallItem); ok && c.Name == tool.Name {
					call = true
				}
			}
			t.Logf("%s: call %v, finish %s", model, call, resp.FinishReason)
		})
	}
}
