package catalog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0028-MADR D-A1: a cold cache does not hold a request up, a refresh
// revalidates with the document's ETag, and the document is decoded as it
// streams in, under the same limit.

// TestCachedMetadataWith_ColdStartsFetchWithoutWaiting: with nothing cached,
// CachedMetadataWith reports false at once and starts the fetch; once the
// fetch is done, it returns the document.
func TestCachedMetadataWith_ColdStartsFetchWithoutWaiting(t *testing.T) {
	enableModelMetadata(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		time.Sleep(450 * time.Millisecond)
		_, _ = w.Write([]byte(smallMetadataDoc))
	}))
	t.Cleanup(srv.Close)
	opt := llmprovider.WithModelMetadataURL(srv.URL)
	start := time.Now()
	if _, ok := CachedMetadataWith(context.Background(), llmprovider.ProviderOpencodeZen, opt); ok {
		t.Fatal("cold cache: ok = true; want false")
	}
	if took := time.Since(start); took > 50*time.Millisecond {
		t.Errorf("cold cache: returned after %v; want it not to wait for the fetch", took)
	}
	waitMetadataFetches()
	meta, ok := CachedMetadataWith(context.Background(), llmprovider.ProviderOpencodeZen, opt)
	if !ok {
		t.Fatal("after the fetch: ok = false; want the document")
	}
	if _, listed := meta.NPM(llmprovider.ProviderOpencodeZen, "glm-5.3-flash"); !listed {
		t.Error("after the fetch: the document does not list glm-5.3-flash")
	}
	if n := hits.Load(); n != 1 {
		t.Errorf("requests = %d; want 1", n)
	}
}

// TestCachedMetadataWith_Unavailable: metadata that is off, a recent failure,
// and an option CachedMetadataWith refuses all report false without a fetch.
func TestCachedMetadataWith_Unavailable(t *testing.T) {
	enableModelMetadata(t)
	srv, hits := metadataServer(t, http.StatusInternalServerError, "")
	opt := llmprovider.WithModelMetadataURL(srv.URL)
	if _, ok := CachedMetadataWith(context.Background(), llmprovider.ProviderOpencodeZen, opt,
		llmprovider.WithoutModelMetadata()); ok {
		t.Error("metadata off: ok = true")
	}
	if _, ok := CachedMetadataWith(context.Background(), llmprovider.ProviderOpencodeZen, opt,
		llmprovider.For(llmprovider.ProviderKilo, WithKiloOrganization("org"))); ok {
		t.Error("an option for another provider: ok = true")
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("requests = %d; want none yet", n)
	}
	_, _ = CachedMetadataWith(context.Background(), llmprovider.ProviderOpencodeZen, opt)
	waitMetadataFetches()
	if _, ok := CachedMetadataWith(context.Background(), llmprovider.ProviderOpencodeZen, opt); ok {
		t.Error("after a failed fetch: ok = true")
	}
	if n := hits.Load(); n != 1 {
		t.Errorf("requests = %d; want 1, the failure remembered", n)
	}
}

// TestLoadModelMetadata_RevalidatesWithETag: past the TTL the refresh sends
// the document's ETag, and a 304 keeps the document, renews its time, and
// decodes nothing.
func TestLoadModelMetadata_RevalidatesWithETag(t *testing.T) {
	enableModelMetadata(t)
	var hits atomic.Int32
	var sentTag atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if tag := r.Header.Get("If-None-Match"); tag != "" {
			sentTag.Store(tag)
			if tag == `"v1"` {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
		w.Header().Set("ETag", `"v1"`)
		_, _ = w.Write([]byte(smallMetadataDoc))
	}))
	t.Cleanup(srv.Close)
	cfg := testConfig(t, llmprovider.WithModelMetadataURL(srv.URL))
	first, err := loadModelMetadata(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	decodes := metadataDecodes.Load()
	later := time.Now().Add(modelMetadataTTL + time.Second)
	metadataNow = func() time.Time { return later }
	t.Cleanup(func() { metadataNow = time.Now })
	if _, err := loadModelMetadata(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	waitMetadataFetches()
	if n := hits.Load(); n != 2 {
		t.Fatalf("requests = %d; want 2", n)
	}
	if tag, _ := sentTag.Load().(string); tag != `"v1"` {
		t.Errorf("If-None-Match = %q; want %q", tag, `"v1"`)
	}
	modelMetadataMu.Lock()
	e := modelMetadataCache[srv.URL]
	modelMetadataMu.Unlock()
	if !e.fetched.Equal(later) || e.etag != `"v1"` || fmt.Sprint(e.doc) != fmt.Sprint(first) {
		t.Errorf("entry: fetched %v, etag %q; want the renewed time %v, the same ETag and document", e.fetched, e.etag, later)
	}
	if n := metadataDecodes.Load() - decodes; n != 0 {
		t.Errorf("decodes after the 304 = %d; want 0", n)
	}
}

// TestDecodeModelMetadata_StreamedLimit: a body one byte past the limit is
// refused, naming it; a valid body decodes; trailing data is still refused,
// as json.Unmarshal refused it.
func TestDecodeModelMetadata_StreamedLimit(t *testing.T) {
	const head, tail = `{"opencode":{"models":{}},"pad":"`, `"}`
	body := head + strings.Repeat("x", metadataLimit+1-len(head)-len(tail)) + tail
	if len(body) != metadataLimit+1 {
		t.Fatalf("body is %d bytes; want %d", len(body), metadataLimit+1)
	}
	if _, err := decodeModelMetadata(strings.NewReader(body)); err == nil || !strings.Contains(err.Error(), "32 MiB") {
		t.Errorf("a body past the limit: err = %v; want one naming 32 MiB", err)
	}
	exact := head + strings.Repeat("x", metadataLimit-len(head)-len(tail)) + tail
	if _, err := decodeModelMetadata(strings.NewReader(exact)); err != nil {
		t.Errorf("a body of exactly the limit: %v; want it decoded", err)
	}
	doc, err := decodeModelMetadata(strings.NewReader(smallMetadataDoc))
	if err != nil || doc[metadataKeyZen] == nil {
		t.Errorf("a valid body: %v, %v; want it decoded", doc, err)
	}
	if _, err := decodeModelMetadata(strings.NewReader(smallMetadataDoc + ` {}`)); err == nil {
		t.Error("trailing data: no error; want one")
	}
}
