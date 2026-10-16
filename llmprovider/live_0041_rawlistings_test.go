//go:build live_gateways

package llmprovider_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// TestLive_RecordRawListings_0041 reads, through catalog.ListRaw, a ChatGPT
// session's listing and a local Ollama's, and writes each one's page,
// byte for byte, into LLMPROVIDER_RAW_RECORD_DIR, for 0041-PLAN Phase 8's
// fixtures. It is also ListRaw's live check on both: one page, models, each
// with its provider's id.
//
//   - chatgpt: LLMPROVIDER_LIVE_CHATGPT=1 and the Codex CLI's login; one
//     listing request, metered on the subscription.
//   - ollama: a local server answering on localhost:11434; free.
//
// Each skips without its service. It prints counts, never a body or a key.
func TestLive_RecordRawListings_0041(t *testing.T) {
	dir := os.Getenv("LLMPROVIDER_RAW_RECORD_DIR")
	if dir == "" {
		t.Skip("LLMPROVIDER_RAW_RECORD_DIR unset: nothing to record into")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	record := func(t *testing.T, name string, provider llmprovider.ProviderID, src llmprovider.TokenSource) {
		t.Helper()
		ctx, cancel := llmprovider.LiveCtx(t)
		defer cancel()
		raw, err := catalog.ListRaw(ctx, provider, src)
		llmprovider.SkipIfTransient(t, err)
		if err != nil {
			t.Fatalf("ListRaw: %v", err)
		}
		if len(raw.Pages) != 1 || len(raw.Models) == 0 {
			t.Fatalf("%d pages, %d models; want one page with models", len(raw.Pages), len(raw.Models))
		}
		for i, m := range raw.Models {
			if m.ID == "" {
				t.Errorf("model %d has no id", i)
			}
		}
		if err := os.WriteFile(filepath.Join(dir, name+".json"), raw.Pages[0], 0o600); err != nil {
			t.Fatal(err)
		}
		t.Logf("%s: %d models, %d bytes, recorded", name, len(raw.Models), len(raw.Pages[0]))
	}
	t.Run("chatgpt", func(t *testing.T) {
		record(t, "chatgpt", llmprovider.ProviderOpenAI,
			liveVendorSession(t, llmprovider.ProviderOpenAI, "LLMPROVIDER_LIVE_CHATGPT", "CODEX_HOME", ".codex"))
	})
	t.Run("ollama", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:11434/api/tags", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Skip("no local Ollama server answered")
		}
		_ = resp.Body.Close()
		record(t, "ollama", llmprovider.ProviderOllama, llmprovider.NewStaticToken(""))
	})
}
