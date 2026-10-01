package llmprovider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
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
	cfg := ApplyOptions([]ProviderOption{WithModelMetadataURL(srv.URL)})
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
	if n := hits.Load(); n != 2 {
		t.Errorf("requests = %d, want 2 (one fetch, one failed refresh)", n)
	}
}

// TestLoadModelMetadata_FailureCachedBriefly pins MADR 0013 A6: a failed
// fetch is remembered, so an immediate second load does not fetch again.
func TestLoadModelMetadata_FailureCachedBriefly(t *testing.T) {
	enableModelMetadata(t)
	srv, hits := metadataServer(t, http.StatusInternalServerError, "")
	cfg := ApplyOptions([]ProviderOption{WithModelMetadataURL(srv.URL)})
	for i := range 2 {
		if _, err := loadModelMetadata(context.Background(), cfg); err == nil {
			t.Fatalf("load %d: want the HTTP 500 error", i+1)
		}
	}
	if n := hits.Load(); n != 1 {
		t.Errorf("requests = %d, want 1 (the failure is cached)", n)
	}
}
