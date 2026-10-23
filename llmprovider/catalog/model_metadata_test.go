package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// waitMetadataFetches waits for every metadata fetch in flight, such as a
// background refresh, to finish.
func waitMetadataFetches() { processStore.metadataFetching.Wait() }

// resetModelMetadataCache empties the in-process metadata cache, after any
// fetch in flight has finished. httptest reuses ports, so a cache keyed by
// URL must not outlive a test.
func resetModelMetadataCache() {
	waitMetadataFetches()
	processStore.metadataMu.Lock()
	defer processStore.metadataMu.Unlock()
	clear(processStore.metadata)
}

// resetProcessStore empties every map of the process's store keyed by a URL,
// the metadata, the listings and Ollama's show replies, after any fetch in
// flight has finished (0058-MADR, amendment of 2026-10-23).
func resetProcessStore() {
	resetModelMetadataCache()
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

// enableModelMetadata empties the process's store on both sides of a test
// that fetches a fixture document. The fetch is on unless
// WithoutModelMetadata turns it off (0015-PLAN S10).
func enableModelMetadata(t *testing.T) {
	t.Helper()
	metadataOn = true
	clearProcessStore(t)
	t.Cleanup(func() { metadataOn = false })
}

// metadataOn is set by enableModelMetadata for one test, which serves its own
// metadata document. Otherwise listT turns the fetch off, so no test reaches
// models.opencode.ai (0015-PLAN S10). No test here runs in parallel.
var metadataOn bool

// listT is List, with the metadata fetch off unless enableModelMetadata
// turned it on.
func listT(ctx context.Context, id llmprovider.ProviderID, src llmprovider.TokenSource, opts ...llmprovider.Option) (Catalog, error) {
	if !metadataOn {
		opts = append(opts, llmprovider.WithoutModelMetadata())
	}
	return List(ctx, id, src, opts...)
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

// TestModelMetadataURL_Precedence: the option, else OpenCode's document. The
// environment is not read (0015-PLAN S10).
func TestModelMetadataURL_Precedence(t *testing.T) {
	t.Setenv(envModelMetadataURL, "http://env")
	if got := modelMetadataURL(config{ModelMetadataURL: "http://opt"}); got != "http://opt" {
		t.Errorf("option: got %q, want http://opt", got)
	}
	if got := modelMetadataURL(config{}); got != "https://models.opencode.ai/api.json" {
		t.Errorf("default: got %q, want OpenCode's document, never the environment's", got)
	}
}

// TestLoadModelMetadata_Disabled: WithoutModelMetadata turns the fetch off
// whatever the URL, and the environment variable does not, by itself.
func TestLoadModelMetadata_Disabled(t *testing.T) {
	clearProcessStore(t)
	srv, hits := metadataServer(t, http.StatusOK, smallMetadataDoc)
	resetModelMetadataCache()
	_, err := loadModelMetadata(context.Background(),
		testConfig(t, llmprovider.WithModelMetadataURL(srv.URL), llmprovider.WithoutModelMetadata()))
	if !errors.Is(err, errModelMetadataDisabled) {
		t.Errorf("WithoutModelMetadata: err = %v, want errModelMetadataDisabled", err)
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("disabled fetch made %d requests, want 0", n)
	}
	t.Setenv(envDisableModelMetadata, "1")
	resetModelMetadataCache()
	if _, err := loadModelMetadata(context.Background(), testConfig(t, llmprovider.WithModelMetadataURL(srv.URL))); err != nil {
		t.Errorf("with only the variable set: err = %v, want the fetch to run", err)
	}
}

// TestOptionsFromEnv: the helper a caller opts into turns each variable into
// its option, and adds none for an unset or non-boolean one.
func TestOptionsFromEnv(t *testing.T) {
	clearProcessStore(t)
	srv, hits := metadataServer(t, http.StatusOK, smallMetadataDoc)
	t.Setenv(envModelMetadataURL, srv.URL)
	t.Setenv(envDisableModelMetadata, "")
	resetModelMetadataCache()
	if _, err := loadModelMetadata(context.Background(), testConfig(t, OptionsFromEnv()...)); err != nil || hits.Load() != 1 {
		t.Errorf("URL from the environment: err = %v, %d requests; want one fetch of it", err, hits.Load())
	}
	for _, v := range []string{"1", "true"} {
		t.Setenv(envDisableModelMetadata, v)
		resetModelMetadataCache()
		if _, err := loadModelMetadata(context.Background(), testConfig(t, OptionsFromEnv()...)); !errors.Is(err, errModelMetadataDisabled) {
			t.Errorf("%s=%q: err = %v, want errModelMetadataDisabled", envDisableModelMetadata, v, err)
		}
	}
	t.Setenv(envModelMetadataURL, "")
	t.Setenv(envDisableModelMetadata, "sometimes")
	if got := OptionsFromEnv(); len(got) != 0 {
		t.Errorf("OptionsFromEnv() = %d options, want none", len(got))
	}
}

func TestLoadModelMetadata_CachesSuccess(t *testing.T) {
	enableModelMetadata(t)
	srv, hits := metadataServer(t, http.StatusOK, smallMetadataDoc)
	cfg := testConfig(t, llmprovider.WithModelMetadataURL(srv.URL))
	first, err := loadModelMetadata(context.Background(), cfg)
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	second, err := loadModelMetadata(context.Background(), cfg)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if n := hits.Load(); n != 1 {
		t.Errorf("requests = %d, want 1 (second load cached)", n)
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("cached document differs: %v vs %v", first, second)
	}
}

// TestLoadModelMetadata_FailureRetriedAfterBackoff pins MADR 0013 A6: once
// modelMetadataRetryAfter has passed since a failure, the next load fetches.
// It replaces TestLoadModelMetadata_FailureNotCached (0009 PLAN §1.11 item 3).
func TestLoadModelMetadata_FailureRetriedAfterBackoff(t *testing.T) {
	enableModelMetadata(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hits.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(smallMetadataDoc))
	}))
	t.Cleanup(srv.Close)
	cfg := testConfig(t, llmprovider.WithModelMetadataURL(srv.URL))
	if _, err := loadModelMetadata(context.Background(), cfg); err == nil {
		t.Fatal("first load: want an error for HTTP 500")
	}
	processStore.metadataMu.Lock()
	e := processStore.metadata[srv.URL]
	e.failed = time.Now().Add(-modelMetadataRetryAfter - time.Second)
	processStore.metadata[srv.URL] = e
	processStore.metadataMu.Unlock()
	doc, err := loadModelMetadata(context.Background(), cfg)
	if err != nil || doc.all()[metadataKeyZen] == nil {
		t.Fatalf("load after the backoff: doc=%v err=%v, want a fresh fetch", doc, err)
	}
	if n := hits.Load(); n != 2 {
		t.Errorf("requests = %d, want 2", n)
	}
}

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
