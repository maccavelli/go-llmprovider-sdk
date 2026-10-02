package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// resetModelMetadataCache empties the in-process metadata cache. httptest
// reuses ports, so a cache keyed by URL must not outlive a test.
func resetModelMetadataCache() {
	modelMetadataMu.Lock()
	defer modelMetadataMu.Unlock()
	clear(modelMetadataCache)
}

// enableModelMetadata empties the metadata cache on both sides of a test that
// fetches a fixture document. The fetch is on unless WithoutModelMetadata
// turns it off (0015-PLAN S10).
func enableModelMetadata(t *testing.T) {
	t.Helper()
	metadataOn = true
	resetModelMetadataCache()
	t.Cleanup(func() {
		metadataOn = false
		resetModelMetadataCache()
	})
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
	modelMetadataMu.Lock()
	e := modelMetadataCache[srv.URL]
	e.failed = time.Now().Add(-modelMetadataRetryAfter - time.Second)
	modelMetadataCache[srv.URL] = e
	modelMetadataMu.Unlock()
	doc, err := loadModelMetadata(context.Background(), cfg)
	if err != nil || doc[metadataKeyZen] == nil {
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
	for k := range doc {
		keys = append(keys, k)
	}
	if len(doc) != 3 || doc["opencode"] == nil || doc["opencode-go"] == nil || doc["huggingface"] == nil {
		t.Errorf("keys = %v, want exactly opencode, opencode-go, huggingface", keys)
	}
	doc, err = decodeModelMetadata(strings.NewReader(`{"huggingface":{"name":"x"}}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := doc["huggingface"]; ok {
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
	if c.status != "deprecated" || c.group != "glm-flash" || !c.small || c.context != 131072 {
		t.Errorf("status=%q group=%q small=%v context=%d", c.status, c.group, c.small, c.context)
	}
	u := metadataCandidate("org/flash", modelMetadata{}, false, refNow)
	want := rankCandidate{id: "org/flash", group: "org", small: true}
	if u != want {
		t.Errorf("uncovered candidate = %+v, want %+v", u, want)
	}
}
