//go:build live_gateways

package llmprovider_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
)

// liveRoundTrip adapts a function to http.RoundTripper.
type liveRoundTrip func(*http.Request) (*http.Response, error)

func (fn liveRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

// liveOpenAI builds OpenAI from a session, or fails the test.
func liveOpenAI(t *testing.T, src llmprovider.TokenSource, model string, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := openai.New(append([]llmprovider.Option{llmprovider.WithTokenSource(src), llmprovider.WithModel(model)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func userText(text string) *llmprovider.Request {
	return &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: "user", Text: text}}}
}

// TestLive_ChatGPTGenerate is gate G-C's end-to-end check through this module: a
// text call, a forced tool call, and a tool round trip replayed with
// store:false, on the backend's current default model.
func TestLive_ChatGPTGenerate(t *testing.T) {
	session := liveChatGPTSession(t)
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	p := liveOpenAI(t, session, "gpt-6-astra")
	out, err := llmprovider.GenerateText(ctx, p, userText("Reply with only the word ALPHA"))
	llmprovider.SkipIfTransient(t, err)
	if err != nil || !strings.Contains(strings.ToUpper(out), "ALPHA") {
		t.Fatalf("Generate = %q, %v", out, err)
	}
	tool := llmprovider.Tool{Name: "get_weather", Description: "Get the weather for a city", Schema: map[string]any{
		"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}, "required": []string{"city"}}}
	req := userText("What is the weather in Paris?")
	req.Tools, req.ToolChoice = []llmprovider.Tool{tool}, llmprovider.ForceTool(tool.Name)
	res, err := p.Generate(ctx, req)
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("forced tool: %v", err)
	}
	var call *llmprovider.FunctionCallItem
	for _, it := range res.Output {
		if fc, ok := it.(llmprovider.FunctionCallItem); ok {
			call = &fc
		}
	}
	if call == nil || !strings.Contains(strings.ToLower(call.Arguments), "paris") {
		t.Fatalf("tool call = %+v", call)
	}
	final, err := p.Generate(ctx, &llmprovider.Request{Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: "user", Text: "What is the weather in Paris?"},
		*call, llmprovider.FunctionCallOutputItem{CallID: call.CallID, Output: `{"forecast":"sunny, 21C"}`}}})
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("round trip: %v", err)
	}
	if text := strings.ToLower(final.OutputText()); !strings.Contains(text, "sunny") && !strings.Contains(text, "21") {
		t.Errorf("reply %q does not use the tool result", final.OutputText())
	}
}
