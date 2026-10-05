package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestLoadModelMetadata_StaleOnRefreshFailure pins MADR 0013 A6: when the
// cached document has expired and the refresh fails, the stale copy is used.
func TestLoadModelMetadata_StaleOnRefreshFailure(t *testing.T) {
	enableModelMetadata(t)
	var fail atomic.Bool
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		if fail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(smallMetadataDoc))
	}))
	t.Cleanup(srv.Close)
	cfg := testConfig(t, llmprovider.WithModelMetadataURL(srv.URL))
	if _, err := loadModelMetadata(context.Background(), cfg); err != nil {
		t.Fatalf("first load: %v", err)
	}
	modelMetadataMu.Lock()
	e := modelMetadataCache[srv.URL]
	e.fetched = time.Now().Add(-modelMetadataTTL - time.Minute)
	modelMetadataCache[srv.URL] = e
	modelMetadataMu.Unlock()
	fail.Store(true)
	doc, err := loadModelMetadata(context.Background(), cfg)
	if err != nil || doc[metadataKeyZen] == nil {
		t.Errorf("refresh failed: doc=%v err=%v, want the stale document", doc, err)
	}
	waitMetadataFetches() // the refresh runs in the background (0021-MADR C2)
	if n := hits.Load(); n != 2 {
		t.Errorf("requests = %d, want 2 (one fetch, one failed refresh)", n)
	}
}

// TestLoadModelMetadata_FailureCachedBriefly pins MADR 0013 A6: a failed
// fetch is remembered, so an immediate second load does not fetch again.
func TestLoadModelMetadata_FailureCachedBriefly(t *testing.T) {
	enableModelMetadata(t)
	srv, hits := metadataServer(t, http.StatusInternalServerError, "")
	cfg := testConfig(t, llmprovider.WithModelMetadataURL(srv.URL))
	for i := range 2 {
		if _, err := loadModelMetadata(context.Background(), cfg); err == nil {
			t.Fatalf("load %d: want the HTTP 500 error", i+1)
		}
	}
	if n := hits.Load(); n != 1 {
		t.Errorf("requests = %d, want 1 (the failure is cached)", n)
	}
}

// TestLookupMetadata_NilClientUsesDefault: a nil client means the default
// client, as an omitted WithHTTPClient does. On a cold cache it used to
// dereference the nil client (0009-PLAN, deviation of 2026-10-03).
func TestLookupMetadata_NilClientUsesDefault(t *testing.T) {
	enableModelMetadata(t)
	srv, hits := metadataServer(t, http.StatusOK, smallMetadataDoc)
	if _, err := LookupMetadata(context.Background(), srv.URL, nil); err != nil {
		t.Fatalf("LookupMetadata with a nil client: %v", err)
	}
	if n := hits.Load(); n != 1 {
		t.Errorf("requests = %d, want 1 (fetched through the default client)", n)
	}
}

// slowMetadataServer answers each request with smallMetadataDoc after delay,
// and counts requests.
func slowMetadataServer(t *testing.T, delay time.Duration) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		select {
		case <-time.After(delay):
		case <-r.Context().Done():
			return
		}
		_, _ = w.Write([]byte(smallMetadataDoc))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

// TestLookupMetadata_CallerDeadlineNotCached (0021-MADR C1): a lookup whose
// own deadline ends first fails, and that failure is not remembered: the next
// lookup, with time, gets the document.
func TestLookupMetadata_CallerDeadlineNotCached(t *testing.T) {
	enableModelMetadata(t)
	srv, _ := slowMetadataServer(t, 50*time.Millisecond)
	short, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if _, err := LookupMetadata(short, srv.URL, nil); err == nil {
		t.Fatal("a lookup under a 1 ms deadline succeeded")
	}
	m, err := LookupMetadata(context.Background(), srv.URL, nil)
	if err != nil || m.doc[metadataKeyZen] == nil {
		t.Fatalf("the next lookup = %v, %v; want the document, not the first caller's deadline", m.doc, err)
	}
}

// TestLookupMetadata_OneFetchForConcurrentLookups (0021-MADR C2): concurrent
// cold lookups of one URL share one fetch.
func TestLookupMetadata_OneFetchForConcurrentLookups(t *testing.T) {
	enableModelMetadata(t)
	srv, hits := slowMetadataServer(t, 100*time.Millisecond)
	var wg sync.WaitGroup
	errs := make([]error, 20)
	for i := range errs {
		wg.Go(func() { _, errs[i] = LookupMetadata(context.Background(), srv.URL, nil) })
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("lookup %d: %v", i, err)
		}
	}
	if n := hits.Load(); n != 1 {
		t.Errorf("20 concurrent lookups made %d requests, want 1", n)
	}
}

// TestLookupMetadata_StaleServedWhileRefreshing (0021-MADR C2): past the TTL,
// lookups return the cached document at once, and one refresh runs in the
// background.
func TestLookupMetadata_StaleServedWhileRefreshing(t *testing.T) {
	enableModelMetadata(t)
	srv, hits := slowMetadataServer(t, 300*time.Millisecond)
	if _, err := LookupMetadata(context.Background(), srv.URL, nil); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(modelMetadataTTL + time.Minute)
	metadataNow = func() time.Time { return later }
	t.Cleanup(func() { metadataNow = time.Now })
	start := time.Now()
	for i := range 10 {
		if m, err := LookupMetadata(context.Background(), srv.URL, nil); err != nil || m.doc[metadataKeyZen] == nil {
			t.Fatalf("lookup %d past the TTL = %v, %v; want the stale document", i, m.doc, err)
		}
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Errorf("10 lookups past the TTL took %v; want the stale document at once", elapsed)
	}
	waitMetadataFetches()
	if n := hits.Load(); n != 2 {
		t.Errorf("requests = %d, want 2: the first fetch and one background refresh", n)
	}
}
