//go:build live_gateways

package llmprovider_test

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// The documents the live tests hand in, now that the SDK fetches none
// (0093-MADR D12, MH32): a committed fixture, read by hand against each
// assertion that needs it, or the live document a test fetches itself.
// fixtureDoc and extractDoc are in fixtures_0093_test.go, which the offline
// twins share.

// rankingDoc is the 2026-10-12 ranking fixture: the facts a live assertion
// reads, each checked by hand against it (0093-PLAN Phase 0, step 6).
func rankingDoc(t *testing.T) llmprovider.Option {
	return fixtureDoc(t, "ranking-2026-10-12/metadata.json")
}

// rankingFile is the ranking fixture's bytes, for a test that reads the
// document itself as well as handing it in.
func rankingFile(t *testing.T) io.Reader {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("catalog", "testdata", "ranking-2026-10-12", "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(raw)
}

// liveOpencodeDoc is the live document the test fetches itself, parsed, for
// an assertion that follows the live provider.npm; it skips when the
// document is unreachable.
func liveOpencodeDoc(t *testing.T) catalog.Metadata {
	t.Helper()
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://models.opencode.ai/api.json", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("models.opencode.ai unreachable: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	m, err := catalog.ParseMetadata(resp.Body)
	if err != nil {
		t.Skipf("models.opencode.ai's document: %v", err)
	}
	return m
}
