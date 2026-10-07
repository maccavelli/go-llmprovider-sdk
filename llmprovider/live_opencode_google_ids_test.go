//go:build live_gateways

package llmprovider_test

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// generatedCallID is the id the generateContent decoder makes for a call
// that came without one: "name#index" (0021-MADR W10).
var generatedCallID = regexp.MustCompile(`^[A-Za-z0-9_]+#\d+$`)

// googleAPIBase is Google's generateContent API, which OpenCode's google
// route forwards to.
const googleAPIBase = "https://generativelanguage.googleapis.com/v1beta"

// TestLive_OpencodeGoogleTwoCallRoundTrip (0026-MADR F65, live first): on a
// gemini-3.x model, through the google route's own encoder and decoder, two
// calls to one function in one reply, then both results. It records whether
// the service sent functionCall.id, and fails if it refuses results paired by
// name and position, or pairs them wrongly: then generateContent must send
// the ids back (F65's fix).
//
//   - google-direct sends the route's request to Google itself, with
//     GEMINI_API_KEY: the measurement the PLAN's deviation D6 chose.
//   - zen sends it through OpenCode Zen, with OPENCODE_API_KEY. Zen answered
//     402 for want of funds on 2026-10-07, which skips.
func TestLive_OpencodeGoogleTwoCallRoundTrip(t *testing.T) {
	t.Run("google-direct", func(t *testing.T) {
		key := llmprovider.LiveEnvKey(t, "GEMINI_API_KEY")
		p, err := opencode.NewZen(llmprovider.WithAPIKey(key), llmprovider.WithModel("gemini-3.8-flash"),
			llmprovider.WithBaseURL(googleAPIBase), opencode.WithRoute(opencode.RouteGoogle))
		if err != nil {
			t.Fatalf("NewZen: %v", err)
		}
		twoCallRoundTrip(t, p, "gemini-3.8-flash")
	})
	t.Run("zen", func(t *testing.T) {
		key := llmprovider.LiveOpencodeKey(t)
		model := llmprovider.LiveModel(t, llmprovider.ProviderOpencodeZen, "gemini-3.8-flash", "gemini-3.5-flash-lite")
		p, err := opencode.NewZen(llmprovider.WithAPIKey(key), llmprovider.WithModel(model), opencode.WithRoute(opencode.RouteGoogle))
		if err != nil {
			t.Fatalf("NewZen: %v", err)
		}
		twoCallRoundTrip(t, p, model)
	})
}

// twoCallRoundTrip asks p for two calls to one function in one reply, sends
// both results back, and checks the reply pairs each result with its call.
func twoCallRoundTrip(t *testing.T, p llmprovider.Provider, model string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tools := []llmprovider.Tool{{Name: "city_code", Description: "Returns the secret code of one city.",
		Schema: map[string]any{"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}},
			"required": []string{"city"}}}}
	input := []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser,
		Text: "Get the secret codes of Oslo and of Rome. Call city_code twice, once per city, both in this reply."}}
	first, err := p.Generate(ctx, &llmprovider.Request{Input: input, Tools: tools, ToolChoice: llmprovider.ToolChoiceRequired})
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("turn 1: %v", err)
	}
	var calls []llmprovider.FunctionCallItem
	for _, item := range first.Output {
		if c, ok := item.(llmprovider.FunctionCallItem); ok {
			calls = append(calls, c)
		}
	}
	if len(calls) != 2 {
		t.Skipf("%s made %d calls, not 2; the measurement needs two calls to one function in one reply", model, len(calls))
	}
	sentIDs := !generatedCallID.MatchString(calls[0].CallID) && !generatedCallID.MatchString(calls[1].CallID)
	t.Logf("%s: call ids %q, %q; service sent functionCall.id: %t", model, calls[0].CallID, calls[1].CallID, sentIDs)

	codes := map[string]string{"oslo": "OSLO-7731", "rome": "ROME-4412"}
	input = append(input, first.Output...)
	for _, c := range calls {
		city := "rome"
		if strings.Contains(strings.ToLower(c.Arguments), "oslo") {
			city = "oslo"
		}
		input = append(input, llmprovider.FunctionCallOutputItem{CallID: c.CallID, Output: `{"code":"` + codes[city] + `"}`})
	}
	input = append(input, llmprovider.MessageItem{Role: llmprovider.RoleUser,
		Text: "Reply on one line exactly as: Oslo=<code>; Rome=<code>"})
	second, err := p.Generate(ctx, &llmprovider.Request{Input: input, Tools: tools, ToolChoice: llmprovider.ToolChoiceNone})
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("turn 2, results paired by name and position: refused: %v", err)
	}
	reply := second.OutputText()
	t.Logf("%s: turn 2 reply %q", model, reply)
	if !strings.Contains(reply, "Oslo="+codes["oslo"]) || !strings.Contains(reply, "Rome="+codes["rome"]) {
		t.Fatalf("turn 2 reply %q pairs the results wrongly; want Oslo=%s; Rome=%s", reply, codes["oslo"], codes["rome"])
	}
}
