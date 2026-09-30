//go:build live_gateways

package llmprovider_test

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestLive_VendorCLISessionOpenAI generates through the Codex CLI's own
// login. It was the Codex row of TestLive_VendorCLISession.
func TestLive_VendorCLISessionOpenAI(t *testing.T) {
	s := llmprovider.LiveVendorSession(t, llmprovider.ProviderOpenAI, "LLMPROVIDER_LIVE_CHATGPT", "CODEX_HOME", ".codex")
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	out, err := llmprovider.GenerateText(ctx, liveOpenAI(t, s, "gpt-6-astra"), userText("Reply with only the word ALPHA"))
	llmprovider.SkipIfTransient(t, err)
	if err != nil || !strings.Contains(strings.ToUpper(out), "ALPHA") {
		t.Fatalf("Generate = %q, %v", out, err)
	}
}
