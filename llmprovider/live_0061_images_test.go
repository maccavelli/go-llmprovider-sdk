//go:build live_gateways

package llmprovider_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestLive_Images_0061 is 0061-PLAN Phase 7: an image by its bytes, an image
// by URL, and a text request at EffortMax, to each provider and OpenCode route
// (imagesLiveCases), each through checkImagesLive, whose offline twin is
// TestImagesLive_Twin_0061 (deviation 18). It runs only with
// LLMPROVIDER_LIVE_0061 set. A case runs when its provider's key variable is
// set, and is skipped, by name, when it is not. Billed: one generation per
// case at the provider's default output limit; an EffortMax case may send
// twice, when a refusal is learned.
//
// Ollama's cases run only when LLMPROVIDER_LIVE_0061_OLLAMA names the host's
// base URL and LLMPROVIDER_LIVE_0061_OLLAMA_MODEL a model installed there
// whose /api/show lists vision and thinking, as the EffortMax case needs. The ChatGPT session's cases (Q11 (a); the MADR's amendment "Q11's
// live result"; deviation 21) run only with
// LLMPROVIDER_LIVE_CHATGPT=1 and the Codex CLI's login, as
// liveChatGPTSession reads it; they spend the subscription. It logs each
// case's sends and warnings; never a reply's text, a body, a host or a key.
func TestLive_Images_0061(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0061") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0061 to run 0061-PLAN Phase 7's live cases")
	}
	keys := llmprovider.ProviderEnvVars()
	for _, c := range imagesLiveCases() {
		t.Run(c.id, func(t *testing.T) {
			key := os.Getenv(keys[c.provider])
			if key == "" {
				t.Skipf("%s unset", keys[c.provider])
			}
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			r, err := checkImagesLive(ctx, c, http.DefaultTransport, llmprovider.WithAPIKey(key))
			if err != nil {
				t.Fatal(err)
			}
			logImagesLive(t, c, r)
		})
	}
	host, model := os.Getenv("LLMPROVIDER_LIVE_0061_OLLAMA"), os.Getenv("LLMPROVIDER_LIVE_0061_OLLAMA_MODEL")
	for _, c := range ollamaImagesLive(model) {
		t.Run(c.id, func(t *testing.T) {
			if host == "" || model == "" {
				t.Skip("LLMPROVIDER_LIVE_0061_OLLAMA or LLMPROVIDER_LIVE_0061_OLLAMA_MODEL unset")
			}
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			r, err := checkImagesLive(ctx, c, http.DefaultTransport, llmprovider.WithBaseURL(host))
			if err != nil {
				t.Fatal(err)
			}
			logImagesLive(t, c, r)
		})
	}
	for _, c := range chatGPTImagesLive() {
		t.Run(c.id, func(t *testing.T) {
			session := liveChatGPTSession(t)
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			r, err := checkImagesLive(ctx, c, http.DefaultTransport, llmprovider.WithTokenSource(session))
			if err != nil {
				t.Fatal(err)
			}
			logImagesLive(t, c, r)
		})
	}
}

// logImagesLive logs a case's model, sends and warnings, each warning's
// detail included: a detail names a model, an effort or an input's index.
func logImagesLive(t *testing.T, c imagesLive, r imagesLiveResult) {
	t.Helper()
	t.Logf("%s (%s): %d send(s), warnings: %s", c.id, c.model, r.Sends, r.kinds())
	for _, w := range r.Warnings {
		t.Logf("  %s: %s", w.Kind, w.Detail)
	}
}
