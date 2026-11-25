package catalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// resetProcessStore empties every map of the process's store keyed by a URL,
// the listings and Ollama's show replies, after any fetch in flight has
// finished (0058-MADR, amendment of 2026-10-23).
func resetProcessStore() {
	resetListingCache()
}

// clearProcessStore empties the process's store on both sides of a test
// that writes to it under its httptest server's URL.
func clearProcessStore(t *testing.T) {
	t.Helper()
	resetProcessStore()
	t.Cleanup(resetProcessStore)
}

// caches holds each running test's catalog cache, by *testing.T.
var caches sync.Map

// testCache is t's own catalog cache, the same for every call in t, for a
// test that does not reach the process's store itself (0058-MADR D1). A
// test's servers stay open until it ends, so none of them shares a port with
// another within it.
func testCache(t *testing.T) llmprovider.Option {
	t.Helper()
	c, loaded := caches.LoadOrStore(t, new(Cache))
	if !loaded {
		t.Cleanup(func() { caches.Delete(t) })
	}
	return WithCache(c.(*Cache))
}

// listT is List: no test reaches a host but its own servers, since the SDK
// fetches no document (0065-MADR D1; 0093-MADR D6).
func listT(ctx context.Context, id llmprovider.ProviderID, src llmprovider.TokenSource, opts ...llmprovider.Option) (Catalog, error) {
	return List(ctx, id, src, opts...)
}

// handed is body parsed and handed in with WithMetadata: a test that served a
// document by URL hands the same bytes in (0093-MADR D12, MH29).
func handed(t *testing.T, body string) llmprovider.Option {
	t.Helper()
	m, err := ParseMetadata(strings.NewReader(body))
	if err != nil {
		t.Fatalf("ParseMetadata: %v", err)
	}
	return WithMetadata(m)
}

// metadataServer answers every request with status and body, and counts
// requests.
func metadataServer(t *testing.T, status int, body string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

const smallMetadataDoc = `{"opencode":{"models":{"glm-5.3-flash":{"id":"glm-5.3-flash","reasoning":true}}}}`

func TestLoadModelMetadata_Decode(t *testing.T) {
	doc, err := decodeModelMetadata(strings.NewReader(`{
		"opencode":{"models":{"a":{}}},"opencode-go":{"models":{"b":{}}},
		"huggingface":{"models":{"c/d":{}}},"other":{"models":{"e":{}}}}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	var keys []string
	for k := range doc.all() {
		keys = append(keys, k)
	}
	if all := doc.all(); len(all) != 3 || all["opencode"] == nil || all["opencode-go"] == nil || all["huggingface"] == nil {
		t.Errorf("keys = %v, want exactly opencode, opencode-go, huggingface", keys)
	}
	doc, err = decodeModelMetadata(strings.NewReader(`{"huggingface":{"name":"x"}}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := doc.models("huggingface"); ok {
		t.Error("a section without models must be treated as absent")
	}
	if _, err := decodeModelMetadata(strings.NewReader(`{`)); err == nil ||
		!strings.Contains(err.Error(), "model metadata: decode") {
		t.Errorf("invalid JSON: err = %v, want a model metadata: decode error", err)
	}
}

func TestMetadataCandidate_Fields(t *testing.T) {
	var m modelMetadata
	if err := json.Unmarshal([]byte(`{"name":"GLM 5.3 Flash","family":"glm-flash",
		"release_date":"2026-07","status":"deprecated","limit":{"context":131072}}`), &m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	c := metadataCandidate("glm-5.3-flash", m, true, refNow)
	if !c.ageKnown || c.ageDays != 87 {
		t.Errorf("ageDays = %d/%v, want 87", c.ageDays, c.ageKnown)
	}
	if c.reasoningKnown || c.costKnown {
		t.Errorf("reasoningKnown=%v costKnown=%v, want both false (fields absent)", c.reasoningKnown, c.costKnown)
	}
	// The group is the family's leading letters (0021-MADR D3).
	if c.status != "deprecated" || c.group != "glm" || !c.small || c.context != 131072 {
		t.Errorf("status=%q group=%q small=%v context=%d", c.status, c.group, c.small, c.context)
	}
	u := metadataCandidate("org/flash", modelMetadata{}, false, refNow)
	want := rankCandidate{id: "org/flash", group: "org", small: true}
	if u != want {
		t.Errorf("uncovered candidate = %+v, want %+v", u, want)
	}
}
