//go:build live_gateways

package llmprovider_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/gemini"
)

// Gemini's live tests, through its own package (0015-PLAN S7). They REQUIRE
// GEMINI_API_KEY and skip without it.

func liveGemini(t *testing.T, key, model string, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := gemini.New(append([]llmprovider.Option{llmprovider.WithAPIKey(key), llmprovider.WithModel(model)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// geminiRecorder keeps each POST's path and body, passing the request on.
type geminiRecorder struct {
	paths  []string
	bodies [][]byte
}

func (g *geminiRecorder) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method == http.MethodPost && r.Body != nil {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		g.paths, g.bodies = append(g.paths, r.URL.Path), append(g.bodies, b)
		r.Body = io.NopCloser(bytes.NewReader(b))
	}
	return http.DefaultTransport.RoundTrip(r)
}

// TestLive_GeminiInteractions: the provider calls the Interactions API with
// store false, and a system item changes the reply; with WithStore(true) a
// continuation recalls an earlier turn (MADR 0014 §1-§2). The system item is
// checked as a pair, with and without a neutral instruction, so an obeyed
// instruction is told from a reply that happens to match, and a model that
// declines an override does not fail it (0022-MADR).
func TestLive_GeminiInteractions(t *testing.T) {
	key := llmprovider.LiveEnvKey(t, "GEMINI_API_KEY")
	for _, tc := range []struct {
		name   string
		system string
		want   bool
	}{
		{"with", "Begin every reply with the word OMEGA.", true},
		{"without", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := &geminiRecorder{}
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			var input []llmprovider.Item
			if tc.system != "" {
				input = append(input, llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: tc.system})
			}
			input = append(input, llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Say hello."})
			res, err := liveGemini(t, key, "gemini-3.7-flash", llmprovider.WithHTTPClient(&http.Client{Transport: rec})).Generate(ctx, &llmprovider.Request{Input: input})
			llmprovider.SkipIfTransient(t, err)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if got := strings.Contains(strings.ToUpper(res.OutputText()), "OMEGA"); got != tc.want {
				t.Fatalf("Generate = %+v; contains OMEGA = %v, want %v", res, got, tc.want)
			}
			if len(rec.paths) != 1 || !strings.HasSuffix(rec.paths[0], "/interactions") || !bytes.Contains(rec.bodies[0], []byte(`"store":false`)) ||
				bytes.Contains(rec.bodies[0], []byte(`"system_instruction"`)) != tc.want {
				t.Fatalf("requests %v; want one /interactions call with store false, and a system_instruction = %v", rec.paths, tc.want)
			}
		})
	}

	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	client := &http.Client{Transport: &geminiRecorder{}}
	stored := liveGemini(t, key, "gemini-3.7-flash", llmprovider.WithHTTPClient(client), gemini.WithStore(true))
	first, err := stored.Generate(ctx, userText("Remember the number 41. Reply with only OK."))
	llmprovider.SkipIfTransient(t, err)
	if err != nil || first.ID == "" {
		t.Fatalf("first = %+v, %v", first, err)
	}
	req := userText("Which number did I ask you to remember? Reply with only the number.")
	req.PreviousResponseID = first.ID
	next, err := stored.Generate(ctx, req)
	llmprovider.SkipIfTransient(t, err)
	if err != nil || !strings.Contains(next.OutputText(), "41") {
		t.Fatalf("continuation = %+v, %v; want 41", next, err)
	}
}

// TestLive_StaticGeminiServed: every static Gemini id answers on the
// Interactions API (measured 2026-09-27).
func TestLive_StaticGeminiServed(t *testing.T) {
	key := llmprovider.LiveEnvKey(t, "GEMINI_API_KEY")
	for _, model := range catalog.Static(llmprovider.ProviderGemini) {
		t.Run(model, func(t *testing.T) {
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			out, err := llmprovider.GenerateText(ctx, liveGemini(t, key, model), userText("Reply with only the word ALPHA"))
			llmprovider.SkipIfTransient(t, err)
			if err != nil || !strings.Contains(strings.ToUpper(out), "ALPHA") {
				t.Fatalf("%s: GenerateText = %q, %v", model, out, err)
			}
		})
	}
}

// TestLive_GeminiThoughtSummary: a Gemini thinking call returns its thought
// summary as a ReasoningItem (MADR 0014).
func TestLive_GeminiThoughtSummary(t *testing.T) {
	key := llmprovider.LiveEnvKey(t, "GEMINI_API_KEY")
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	req := userText("What is 17 * 23? Reply with only the number.")
	req.Reasoning = &llmprovider.Reasoning{}
	res, err := liveGemini(t, key, "gemini-3.7-flash").Generate(ctx, req)
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var reasoning string
	for _, it := range res.Output {
		if r, ok := it.(llmprovider.ReasoningItem); ok {
			reasoning += r.Text
		}
	}
	if strings.TrimSpace(reasoning) == "" || !strings.Contains(res.OutputText(), "391") {
		t.Fatalf("reasoning %q, answer %q; want a thought summary and 391", reasoning, res.OutputText())
	}
}

var liveWeatherTool = llmprovider.Tool{Name: "get_weather", Description: "Current weather for a city",
	Schema: map[string]any{"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}},
		"required": []string{"city"}}}

// TestLive_GeminiToolRoundTrip sends a completed tool call and its result, and
// requires the model to answer from the result (MADR 0012 §2). It was the
// gemini row of TestLive_ToolRoundTrip.
func TestLive_GeminiToolRoundTrip(t *testing.T) {
	key := llmprovider.LiveEnvKey(t, "GEMINI_API_KEY")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	res, err := liveGemini(t, key, "gemini-3.7-flash").Generate(ctx, &llmprovider.Request{Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "What is the weather in Paris? Use the tool."},
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

// TestLive_GeminiReplaysRealCall: a call Gemini issued goes back with the
// thoughtSignature it came with, and Gemini answers from the result.
func TestLive_GeminiReplaysRealCall(t *testing.T) {
	key := llmprovider.LiveEnvKey(t, "GEMINI_API_KEY")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	p := liveGemini(t, key, "gemini-3.7-flash")
	ask := llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "What is the weather in Paris? Use the tool."}
	first, err := p.Generate(ctx, &llmprovider.Request{Input: []llmprovider.Item{ask},
		Tools: []llmprovider.Tool{liveWeatherTool}, ToolChoice: llmprovider.ForceTool(liveWeatherTool.Name)})
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("Generate with the tool: %v", err)
	}
	var call llmprovider.FunctionCallItem
	for _, item := range first.Output {
		if c, ok := item.(llmprovider.FunctionCallItem); ok {
			call = c
		}
	}
	if call.Name != "get_weather" {
		t.Fatalf("no get_weather call in %+v", first.Output)
	}
	res, err := p.Generate(ctx, &llmprovider.Request{Input: []llmprovider.Item{ask, call,
		llmprovider.FunctionCallOutputItem{CallID: call.CallID, Output: `{"forecast":"sunny, 21C"}`}}})
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if text := strings.ToLower(res.OutputText()); !strings.Contains(text, "sunny") && !strings.Contains(text, "21") {
		t.Fatalf("reply %q does not use the tool result", res.OutputText())
	}
}

// TestLive_GeminiToolChoiceNone pins the one unmeasured tool choice: "none"
// in generation_config.tool_choice is accepted, and no call comes back
// (0015-PLAN S7; MADR 0014 measured only "any" and allowed_tools).
func TestLive_GeminiToolChoiceNone(t *testing.T) {
	key := llmprovider.LiveEnvKey(t, "GEMINI_API_KEY")
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	req := userText("What is the weather in Paris?")
	req.Tools, req.ToolChoice = []llmprovider.Tool{liveWeatherTool}, llmprovider.ToolChoiceNone
	res, err := liveGemini(t, key, "gemini-3.7-flash").Generate(ctx, req)
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("Generate with tool_choice none: %v", err)
	}
	for _, item := range res.Output {
		if _, ok := item.(llmprovider.FunctionCallItem); ok {
			t.Fatalf("a call came back with tool_choice none: %+v", res.Output)
		}
	}
}

// TestLive_GeminiThinkingShapes: Gemini 2.5 and 3.x accept the utility and
// capable efforts.
func TestLive_GeminiThinkingShapes(t *testing.T) {
	key := llmprovider.LiveEnvKey(t, "GEMINI_API_KEY")
	for _, model := range []string{"gemini-2.5-flash", "gemini-3.7-flash"} {
		for _, effort := range []llmprovider.Effort{llmprovider.EffortLow, ""} {
			t.Run(model+"/"+string(effort), func(t *testing.T) {
				ctx, cancel := llmprovider.LiveCtx(t)
				defer cancel()
				req := userText("Reply with only the word ALPHA")
				req.Reasoning = &llmprovider.Reasoning{Effort: effort}
				out, err := llmprovider.GenerateText(ctx, liveGemini(t, key, model, llmprovider.WithMaxTokens(2048)), req)
				switch {
				case errors.Is(err, llmprovider.ErrProviderUnavailable):
					t.Skipf("Gemini overloaded: %v", err)
				case errors.Is(err, llmprovider.ErrRateLimited):
					t.Skipf("rate limited: %v", err)
				case err != nil:
					t.Fatalf("GenerateText: %v", err)
				case !strings.Contains(strings.ToUpper(out), "ALPHA"):
					t.Errorf("output %q does not contain ALPHA", out)
				}
			})
		}
	}
}
