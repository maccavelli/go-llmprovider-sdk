package opencode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0028-MADR D-A1: on a cold metadata cache, a model the route table knows is
// sent on the table's route at once, and the next request, once the document
// is cached, on the document's; a model the table does not know waits for
// the document, as before.

// slowRouteServer is routeServer whose metadata document takes delay to
// arrive. It records generation paths.
func slowRouteServer(t *testing.T, gateway llmprovider.ProviderID, model, npm string, delay time.Duration) (*httptest.Server, func() []string) {
	t.Helper()
	enableMetadata(t)
	var (
		mu    sync.Mutex
		paths []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/api.json") {
			time.Sleep(delay)
			_, _ = w.Write([]byte(`{"` + metadataKey(gateway) + `":{"models":{"` + model + `":{"id":"` + model +
				`","provider":{"npm":"` + npm + `"}}}}}`))
			return
		}
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		switch {
		case strings.HasSuffix(r.URL.Path, "/messages"):
			_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}]}`))
		default:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), paths...)
	}
}

// TestOpencode_ColdCacheUsesTableRoute: minimax-m3 is Messages in Go's table,
// and chat in the document, which takes 2 s to arrive.
func TestOpencode_ColdCacheUsesTableRoute(t *testing.T) {
	gw, model := llmprovider.ProviderOpencodeGo, "minimax-m3"
	srv, paths := slowRouteServer(t, gw, model, "@ai-sdk/openai-compatible", 2*time.Second)
	metaURL := llmprovider.WithModelMetadataURL(srv.URL + "/api.json")
	p := build(t, gw, apiKey(srv.URL, model, metaURL)...)
	start := time.Now()
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if took := time.Since(start); took > 200*time.Millisecond {
		t.Errorf("the first Generate took %v; want it not to wait for the document", took)
	}
	if _, err := catalog.LookupMetadataWith(context.Background(), gw, metaURL); err != nil {
		t.Fatalf("waiting for the document: %v", err)
	}
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got, want := paths(), []string{"/messages", "/chat/completions"}; strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("paths = %v; want the table's route, then the document's: %v", got, want)
	}
}

// TestOpencode_UnknownModelWaitsForMetadata: a model the table does not know
// waits for the document and takes its route.
func TestOpencode_UnknownModelWaitsForMetadata(t *testing.T) {
	gw, model := llmprovider.ProviderOpencodeGo, "brand-new-model"
	srv, paths := slowRouteServer(t, gw, model, "@ai-sdk/anthropic", 300*time.Millisecond)
	p := build(t, gw, apiKey(srv.URL, model, llmprovider.WithModelMetadataURL(srv.URL+"/api.json"))...)
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got := paths(); len(got) != 1 || got[0] != "/messages" {
		t.Errorf("paths = %v; want the document's /messages", got)
	}
}
