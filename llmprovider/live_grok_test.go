//go:build live_gateways

package llmprovider_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/grok"
)

// Grok's live tests, through its own package (0015-PLAN S7). Most REQUIRE
// XAI_API_KEY and skip without it; the CLI login's needs
// LLMPROVIDER_LIVE_GROK_CLI=1.

func liveGrok(t *testing.T, src llmprovider.TokenSource, model string, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := grok.New(append([]llmprovider.Option{llmprovider.WithTokenSource(src), llmprovider.WithModel(model)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// xaiRecorder keeps the last POST body sent, passing the request on.
type xaiRecorder struct{ sent []byte }

func (x *xaiRecorder) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method == http.MethodPost && r.Body != nil {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		x.sent = b
		r.Body = io.NopCloser(bytes.NewReader(b))
	}
	return http.DefaultTransport.RoundTrip(r)
}

func xaiKey(t *testing.T) llmprovider.TokenSource {
	t.Helper()
	return llmprovider.NewStaticToken(llmprovider.LiveEnvKey(t, "XAI_API_KEY"))
}

// TestLive_GrokEffortMenu: grok-4.5 asked for xhigh sends the CLI menu's high
// and answers. xAI accepted every effort on grok-4.5 on 2026-09-27, so this
// proves conformance to the CLI, not a rejection avoided.
func TestLive_GrokEffortMenu(t *testing.T) {
	rec := &xaiRecorder{}
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	req := userText("Reply with only the word ALPHA")
	req.Reasoning = &llmprovider.Reasoning{Effort: llmprovider.EffortXHigh}
	out, err := llmprovider.GenerateText(ctx, liveGrok(t, xaiKey(t), "grok-4.5", llmprovider.WithHTTPClient(&http.Client{Transport: rec})), req)
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}
	if !bytes.Contains(rec.sent, []byte(`"reasoning":{"effort":"high"}`)) {
		t.Fatalf("request did not clamp xhigh to high: %s", rec.sent)
	}
	if !strings.Contains(strings.ToUpper(out), "ALPHA") {
		t.Errorf("reply %q", out)
	}
}

var liveGrokWeather = llmprovider.Tool{Name: "get_weather", Description: "Get the weather for a city", Schema: map[string]any{
	"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}, "required": []string{"city"}}}

// TestLive_GrokToolDescription: xAI takes a tool with its description and
// calls it.
func TestLive_GrokToolDescription(t *testing.T) {
	rec := &xaiRecorder{}
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	call, err := llmprovider.GenerateToolCall(ctx, liveGrok(t, xaiKey(t), "grok-4.6", llmprovider.WithHTTPClient(&http.Client{Transport: rec})),
		&llmprovider.Request{Input: userText("What is the weather in Paris?").Input, Tools: []llmprovider.Tool{liveGrokWeather}})
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("GenerateToolCall: %v", err)
	}
	if !bytes.Contains(rec.sent, []byte(`"description":"Get the weather for a city"`)) {
		t.Fatalf("request did not send the tool description: %s", rec.sent)
	}
	if !strings.Contains(strings.ToLower(call.Arguments), "paris") {
		t.Errorf("arguments %q", call.Arguments)
	}
}

// TestLive_GrokStoreFalse: xAI accepts store:false from WithStore(false). It
// was the grok row of TestLive_ResponsesStoreFalse.
func TestLive_GrokStoreFalse(t *testing.T) {
	rec := &xaiRecorder{}
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	out, err := llmprovider.GenerateText(ctx, liveGrok(t, xaiKey(t), "grok-4.6", grok.WithStore(false),
		llmprovider.WithHTTPClient(&http.Client{Transport: rec})), userText("Reply with only the word ALPHA"))
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}
	if !bytes.Contains(rec.sent, []byte(`"store":false`)) || !strings.Contains(strings.ToUpper(out), "ALPHA") {
		t.Fatalf("out = %q, sent %s", out, rec.sent)
	}
}

// TestLive_GrokToolRoundTrip sends a completed tool call and its result, and
// requires the model to answer from the result (MADR 0012 §2). It was the
// grok row of TestLive_ToolRoundTrip.
func TestLive_GrokToolRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	res, err := liveGrok(t, xaiKey(t), "grok-4.5").Generate(ctx, &llmprovider.Request{Input: []llmprovider.Item{
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

// TestLive_GrokInstructions pins the instructions form: a Request's
// Instructions, sent as a leading system message, change the reply. The same
// request is sent with and without them, so an obeyed instruction is told
// from a reply that happens to match (0015-PLAN S7). The instruction is a
// neutral one: Grok declines one that overrides whatever the user says, in
// any position (0021-MADR, amendment "L2 is the test's prompt, not the
// wire").
func TestLive_GrokInstructions(t *testing.T) {
	for _, tc := range []struct {
		name         string
		instructions string
		want         bool
	}{
		{"with", "Begin every reply with the word OMEGA.", true},
		{"without", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			req := userText("Say hello.")
			req.Instructions = tc.instructions
			out, err := llmprovider.GenerateText(ctx, liveGrok(t, xaiKey(t), "grok-4.6"), req)
			llmprovider.SkipIfTransient(t, err)
			if err != nil {
				t.Fatalf("GenerateText: %v", err)
			}
			if got := strings.Contains(strings.ToUpper(out), "OMEGA"); got != tc.want {
				t.Fatalf("GenerateText = %q; contains OMEGA = %v, want %v", out, got, tc.want)
			}
		})
	}
}

// TestLive_GrokToolChoices pins the unmeasured tool choices: "required" makes
// a call, and "none" makes none (0015-PLAN S7).
func TestLive_GrokToolChoices(t *testing.T) {
	for _, tc := range []struct {
		choice   llmprovider.ToolChoice
		wantCall bool
	}{{llmprovider.ToolChoiceRequired, true}, {llmprovider.ToolChoiceNone, false}} {
		t.Run(string(tc.choice), func(t *testing.T) {
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			req := userText("What is the weather in Paris?")
			req.Tools, req.ToolChoice = []llmprovider.Tool{liveGrokWeather}, tc.choice
			res, err := liveGrok(t, xaiKey(t), "grok-4.6").Generate(ctx, req)
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

// TestLive_GrokVendorCLISession generates through the Grok CLI's own login.
// It was the grok row of TestLive_VendorCLISession.
func TestLive_GrokVendorCLISession(t *testing.T) {
	s := liveVendorSession(t, llmprovider.ProviderGrok, "LLMPROVIDER_LIVE_GROK_CLI", "GROK_HOME", ".grok")
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	out, err := llmprovider.GenerateText(ctx, liveGrok(t, s, "grok-4.6"), userText("Reply with only the word ALPHA"))
	llmprovider.SkipIfTransient(t, err)
	if err != nil || !strings.Contains(strings.ToUpper(out), "ALPHA") {
		t.Fatalf("GenerateText = %q, %v", out, err)
	}
}
