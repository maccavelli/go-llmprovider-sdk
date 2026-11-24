//go:build live_gateways

package llmprovider_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLive_ClaudeWindowRefusal_0088 is 0088-PLAN's OQ10 measurement (its
// first amendment; 0088-MADR ST65). It runs only with LLMPROVIDER_LIVE_0088
// set, with LLMPROVIDER_LIVE_0088_MODEL naming a Claude model before 4.5, and
// ANTHROPIC_API_KEY in the environment. It sends one request to Anthropic's
// /v1/messages: about 184,000 input tokens (windowWords) with max_tokens
// 32,000, so that the input alone fits a 200,000-token window and the input
// plus the limit does not. It writes the reply's status, error type and
// message, redacted (recordWindow, whose offline twin is
// TestWindowRecord_Twin_0088), to
// providers/claude/testdata/window-0088/refusal.json, and logs one line: the
// status, the type, and whether the message states numbers. It never logs a
// body, the message or the key. Billed: not established for a refused
// request; an accepted one is asked to answer in one word.
func TestLive_ClaudeWindowRefusal_0088(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0088") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0088 to run 0088-PLAN's OQ10 measurement")
	}
	key, model := os.Getenv("ANTHROPIC_API_KEY"), os.Getenv("LLMPROVIDER_LIVE_0088_MODEL")
	switch {
	case strings.TrimSpace(key) == "":
		t.Fatal("ANTHROPIC_API_KEY unset")
	case model == "":
		t.Fatal("LLMPROVIDER_LIVE_0088_MODEL unset: name a Claude model before 4.5")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	r, err := recordWindow(ctx, &http.Client{}, "https://api.anthropic.com", key, model)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeWindowRecord(filepath.Join("providers", "claude", "testdata", "window-0088", "refusal.json"), r); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s: HTTP %d, type %q, stop reason %q, message states numbers: %t",
		model, r.Status, r.Type, r.StopReason, strings.ContainsAny(r.Message, "0123456789"))
}
