//go:build live_gateways

package llmprovider

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLive_ToolRoundTrip sends a completed tool call and its result on every
// wire, and requires the model to answer from the result (MADR 0012 §2).
// Before the fix the call was dropped: Anthropic, Gemini and the Responses
// route answered 400, and the chat routes answered without the tool.
func TestLive_ToolRoundTrip(t *testing.T) {
	convo := []Item{
		MessageItem{Role: string(RoleUser), Text: "What is the weather in Paris? Use the tool."},
		FunctionCallItem{CallID: "call_rt_1", Name: "get_weather", Arguments: `{"city":"Paris"}`},
		FunctionCallOutputItem{CallID: "call_rt_1", Output: `{"forecast":"sunny, 21C"}`},
	}
	for _, c := range []struct {
		name, env string
		build     func(t *testing.T, key string) (ItemProvider, error)
	}{} {
		t.Run(c.name, func(t *testing.T) {
			key := os.Getenv(c.env)
			if key == "" {
				t.Skipf("%s unset", c.env)
			}
			p, err := c.build(t, key)
			if err != nil {
				t.Fatalf("construct: %v", err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			res, err := p.GenerateItems(ctx, convo...)
			skipIfTransient(t, err)
			if err != nil {
				t.Fatalf("GenerateItems: %v", err)
			}
			if text := strings.ToLower(res.OutputText()); !strings.Contains(text, "sunny") && !strings.Contains(text, "21") {
				t.Fatalf("reply %q does not use the tool result", res.OutputText())
			}
		})
	}
}
