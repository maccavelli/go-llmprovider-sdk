package opencode

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// TestWireGolden_ChatNoDocument_0096 pins 0096-MADR OC2 at the wire: Zen's
// chat route, with no document handed in and no layer, so that no source
// states glm-5.3's output limit, sends the text scenario with no max_tokens
// and no fallback in its body. Its golden is rewritten by -update, as the
// G-wire goldens are.
func TestWireGolden_ChatNoDocument_0096(t *testing.T) {
	c := wirecase.Case{Listing: zenListing}
	srv := wiretest.NewServer(t, c.Reply)
	p, err := NewZen(append([]llmprovider.Option{llmprovider.WithAPIKey("opencode-wire-key"), llmprovider.WithModel("glm-5.3"),
		catalog.WithCache(new(catalog.Cache))}, wirecase.Opts(srv.URL)...)...)
	if err != nil {
		t.Fatal(err)
	}
	got, err := llmprovider.GenerateText(context.Background(), p, &llmprovider.Request{
		Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: wirecase.Prompt}}})
	rec := wiretest.Record{Requests: srv.Requests(), Result: got}
	if err != nil {
		rec.Error = err.Error()
	}
	wiretest.Check(t, filepath.Join("testdata", "wire", "opencode-zen-chat-nodoc", "text.json"), rec, *updateWire)
}
