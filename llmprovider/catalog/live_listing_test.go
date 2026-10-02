//go:build live_gateways

package catalog

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Live tests of the listing, moved from llmprovider's live files (0015-PLAN
// S8, commit 2):
//
//	go test -tags live_gateways ./llmprovider/catalog -run Live -v

func liveCtx(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 90*time.Second)
}

// liveEnvKey skips unless name is set. Together's listing is billed, so it
// also needs LLMPROVIDER_LIVE_TOGETHER=1.
func liveEnvKey(t *testing.T, name string) string {
	t.Helper()
	if name == "TOGETHER_API_KEY" && os.Getenv("LLMPROVIDER_LIVE_TOGETHER") != "1" {
		t.Skip("LLMPROVIDER_LIVE_TOGETHER unset: live Together calls are billed")
	}
	key := os.Getenv(name)
	if key == "" {
		t.Skipf("%s unset", name)
	}
	return key
}

// TestLive_GrokListingTextOnly: xAI's live listing yields no media model.
func TestLive_GrokListingTextOnly(t *testing.T) {
	key := liveEnvKey(t, "XAI_API_KEY")
	ctx, cancel := liveCtx(t)
	defer cancel()
	cat, err := listT(ctx, llmprovider.ProviderGrok, llmprovider.NewStaticToken(key))
	if err != nil || !cat.Live {
		t.Skipf("listing unavailable: live=%t err=%v", cat.Live, err)
	}
	for _, id := range cat.Usable {
		for _, media := range []string{"imagine", "video", "image", "voice", "tts", "stt"} {
			if strings.Contains(id, media) {
				t.Errorf("usable %q is a media model", id)
			}
		}
	}
	t.Logf("usable: %v", cat.Usable)
}

// TestLive_TogetherListing confirms the bare-array listing: chat models come
// back, curated, and no generation is sent.
func TestLive_TogetherListing(t *testing.T) {
	key := liveEnvKey(t, "TOGETHER_API_KEY")
	ctx, cancel := liveCtx(t)
	defer cancel()
	usable, err := fetchTogetherUsable(ctx, llmprovider.Token{Value: key}, defaultConfig())
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(usable) == 0 {
		t.Fatal("no chat models listed")
	}
	t.Logf("%d chat models; first %v", len(usable), usable[:min(len(usable), 5)])
}

// TestLive_ModelMetadataDocument checks the shape MADR 0009 §2 depends on:
// models.opencode.ai still publishes the three sections, and a known Zen
// model still carries its reasoning flag.
func TestLive_ModelMetadataDocument(t *testing.T) {
	enableModelMetadata(t)
	ctx, cancel := liveCtx(t)
	defer cancel()
	doc, err := loadModelMetadata(ctx, defaultConfig())
	if err != nil {
		t.Skipf("metadata unreachable: %v", err)
	}
	for _, key := range []string{metadataKeyZen, metadataKeyGo, metadataKeyHF} {
		if len(doc[key]) == 0 {
			t.Errorf("DRIFT: %s no longer publishes section %q", defaultModelMetadataURL, key)
		}
	}
	m, ok := doc[metadataKeyZen]["glm-5.3-flash"]
	if !ok || m.Reasoning == nil || !*m.Reasoning {
		t.Errorf("DRIFT: %s/glm-5.3-flash reasoning = %v (present %v), want true", metadataKeyZen, m.Reasoning, ok)
	}
}
