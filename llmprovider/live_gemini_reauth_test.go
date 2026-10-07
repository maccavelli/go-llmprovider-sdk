//go:build live_gateways

package llmprovider_test

import (
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/gemini"
)

// geminiInvalidKey is a key Gemini refuses.
const geminiInvalidKey = "not-a-gemini-key"

// TestLive_GeminiCommandTokenRerunsOnInvalidKey (0026-MADR F6, V4): Gemini
// refuses a key with HTTP 400 API_KEY_INVALID, not 401, and a CommandToken
// that printed it is rerun once, and its next key used. The command prints
// the invalid key on its first run and GEMINI_API_KEY, from its environment,
// after it. It REQUIRES GEMINI_API_KEY and a POSIX sh.
func TestLive_GeminiCommandTokenRerunsOnInvalidKey(t *testing.T) {
	llmprovider.LiveEnvKey(t, "GEMINI_API_KEY")
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh on PATH")
	}
	request := &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Say ok."}}}

	t.Run("refusal", func(t *testing.T) {
		ctx, cancel := llmprovider.LiveCtx(t)
		defer cancel()
		_, err := liveGemini(t, geminiInvalidKey, "gemini-2.5-flash-lite").Generate(ctx, request)
		llmprovider.SkipIfTransient(t, err)
		apiErr, ok := errors.AsType[*llmprovider.APIError](err)
		if !ok {
			t.Fatalf("Generate with an invalid key = %v, want an *APIError", err)
		}
		if apiErr.Status != http.StatusBadRequest || apiErr.Kind != llmprovider.ErrAuthFailure { //nolint:errorlint // the kind itself
			t.Fatalf("invalid key: status %d kind %v, want 400 and ErrAuthFailure", apiErr.Status, apiErr.Kind)
		}
	})

	t.Run("rerun", func(t *testing.T) {
		runs := filepath.Join(t.TempDir(), "runs")
		script := `echo run >> "$1"; if [ "$(wc -l < "$1")" -gt 1 ]; then printf %s "$GEMINI_API_KEY"; else printf %s "$2"; fi`
		src := llmprovider.NewCommandToken(sh, "-c", script, "sh", runs, geminiInvalidKey)
		p, err := gemini.New(llmprovider.WithTokenSource(src), llmprovider.WithModel("gemini-2.5-flash-lite"))
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := llmprovider.LiveCtx(t)
		defer cancel()
		_, err = p.Generate(ctx, request)
		llmprovider.SkipIfTransient(t, err)
		got, readErr := os.ReadFile(runs)
		if readErr != nil {
			t.Fatal(readErr)
		}
		n := strings.Count(string(got), "run\n")
		if err != nil || n != 2 {
			t.Fatalf("Generate = %v after %d command runs, want success after 2", err, n)
		}
	})
}
