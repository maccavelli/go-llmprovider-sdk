//go:build live_gateways

package llmprovider_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/kilo"
)

// Kilo's live tests, through its own package (0015-PLAN S7). They REQUIRE
// KILO_API_KEY: Kilo answers 401 for a placeholder key, even on free models.
// The raw-HTTP probes of the gateway's shapes stay in live_gateways_test.go.

// kiloProbedOn is llmprovider's wireShapesProbedOnKilo, for messages.
const kiloProbedOn = "2026-08-29"

func liveKilo(t *testing.T, candidates []string, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := kilo.New(append([]llmprovider.Option{llmprovider.WithAPIKey(llmprovider.LiveKiloKey(t)),
		llmprovider.WithModel(llmprovider.LiveModel(t, llmprovider.ProviderKilo, candidates...))}, opts...)...)
	if err != nil {
		t.Fatalf("kilo.New: %v", err)
	}
	return p
}

func TestLive_KiloChatCompletions(t *testing.T) {
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	out, err := llmprovider.GenerateText(ctx, liveKilo(t, llmprovider.LiveKiloFreeCollecting, kilo.WithDataCollection(true)),
		userText("Reply with only the word ALPHA"))
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}
	if strings.TrimSpace(out) == "" {
		t.Errorf("empty output (probed %s)", kiloProbedOn)
	}
}

var liveKiloWeather = llmprovider.Tool{Name: "get_weather", Description: "Get the weather for a city", Schema: map[string]any{
	"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}, "required": []string{"city"}}}

// TestLive_KiloToolCall is end-to-end tool calling on a free model.
func TestLive_KiloToolCall(t *testing.T) {
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	call, err := llmprovider.GenerateToolCall(ctx, liveKilo(t, llmprovider.LiveKiloFreeCollecting, llmprovider.WithMaxTokens(400),
		kilo.WithDataCollection(true)), &llmprovider.Request{Input: userText("What is the weather in Paris? Use the tool.").Input,
		Tools: []llmprovider.Tool{liveKiloWeather}})
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("GenerateToolCall: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(call.Arguments), &parsed); err != nil {
		t.Fatalf("tool arguments %q are not valid JSON (probed %s): %v", call.Arguments, kiloProbedOn, err)
	}
}

// TestLive_KiloReasoningShapes is MADR 0009 §6's gate (as amended 2026-09-26):
// on Kilo's first utility default, the gateway must accept both reasoning
// shapes and return reasoning. A 400 is a DRIFT failure here, not a skip. The
// gateway does not validate effort values, so this proves acceptance and that
// reasoning is on, not that {"effort":"low"} changes the effort.
func TestLive_KiloReasoningShapes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		effort llmprovider.Effort
	}{{"enabled", ""}, {"effort low", llmprovider.EffortLow}} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			req := userText("Reply with only the word ALPHA")
			req.Reasoning = &llmprovider.Reasoning{Effort: tc.effort}
			resp, err := liveKilo(t, llmprovider.LiveKiloNonTraining, llmprovider.WithMaxTokens(400)).Generate(ctx, req)
			if errors.Is(err, llmprovider.ErrRateLimited) || errors.Is(err, llmprovider.ErrProviderUnavailable) {
				t.Skipf("gateway transient: %v", err)
			}
			if err != nil {
				t.Fatalf("DRIFT (probed %s): gateway rejected reasoning shape %q: %v", kiloProbedOn, tc.name, err)
			}
			sawReasoning := false
			for _, item := range resp.Output {
				if _, ok := item.(llmprovider.ReasoningItem); ok {
					sawReasoning = true
				}
			}
			if !sawReasoning {
				t.Errorf("DRIFT (probed %s): no reasoning returned for shape %q", kiloProbedOn, tc.name)
			}
		})
	}
}

// TestLive_KiloDataCollectionDenied is gate G-K's assertion: with the default
// deny, a free model that requires data collection is refused with a
// terminal ErrNotPermitted carrying data_collection_required.
func TestLive_KiloDataCollectionDenied(t *testing.T) {
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	_, err := llmprovider.GenerateText(ctx, liveKilo(t, llmprovider.LiveKiloFreeCollecting), userText("Reply with only the word ALPHA"))
	if errors.Is(err, llmprovider.ErrRateLimited) || errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Skipf("transient: %v", err)
	}
	var apiErr *llmprovider.APIError
	if !errors.Is(err, llmprovider.ErrNotPermitted) || !errors.As(err, &apiErr) || apiErr.Code != "data_collection_required" {
		t.Fatalf("err = %v, want ErrNotPermitted with code data_collection_required", err)
	}
}

// TestLive_KiloToolRoundTrip sends a completed tool call and its result, and
// requires the model to answer from the result (MADR 0012 §2). It was the
// kilo row of TestLive_ToolRoundTrip.
func TestLive_KiloToolRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	res, err := liveKilo(t, llmprovider.LiveKiloNonTraining).Generate(ctx, &llmprovider.Request{Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: string(llmprovider.RoleUser), Text: "What is the weather in Paris? Use the tool."},
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

// TestLive_KiloToolChoices pins the unmeasured tool choices: "required" makes
// a call, and "none" makes none (0015-PLAN S7).
func TestLive_KiloToolChoices(t *testing.T) {
	for _, tc := range []struct {
		choice   llmprovider.ToolChoice
		wantCall bool
	}{{llmprovider.ToolChoiceRequired, true}, {llmprovider.ToolChoiceNone, false}} {
		t.Run(string(tc.choice), func(t *testing.T) {
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			req := userText("What is the weather in Paris?")
			req.Tools, req.ToolChoice = []llmprovider.Tool{liveKiloWeather}, tc.choice
			res, err := liveKilo(t, llmprovider.LiveKiloNonTraining).Generate(ctx, req)
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
