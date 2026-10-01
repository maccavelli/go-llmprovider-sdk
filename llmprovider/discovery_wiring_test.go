package llmprovider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// deadlineTransport records, for the first request of each method and path,
// how long its context had left (-1 without a deadline), then forwards it.
type deadlineTransport struct {
	mu   sync.Mutex
	seen map[string]time.Duration
}

func (d *deadlineTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	left := time.Duration(-1)
	if dl, ok := r.Context().Deadline(); ok {
		left = time.Until(dl)
	}
	key := r.Method + " " + r.URL.Path
	d.mu.Lock()
	if d.seen == nil {
		d.seen = map[string]time.Duration{}
	}
	if _, ok := d.seen[key]; !ok {
		d.seen[key] = left
	}
	d.mu.Unlock()
	return http.DefaultTransport.RoundTrip(r)
}

func (d *deadlineTransport) left(key string) (time.Duration, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	v, ok := d.seen[key]
	return v, ok
}

// discoverer is the DiscoverModels half of a provider.
type discoverer interface {
	DiscoverModels(context.Context) ([]string, error)
}

// TestDiscoverModels_ListingBounded pins MADR 0013 A5: every DiscoverModels
// listing runs under the 10 s bound ListModelCatalogWithSource applies, so a
// slow listing or metadata host cannot hold discovery past it.
func TestDiscoverModels_ListingBounded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	for _, tc := range []struct {
		name, listing string
		build         func(opts ...ProviderOption) (discoverer, error)
	}{
		{"ollama", "GET /api/tags", func(o ...ProviderOption) (discoverer, error) { return NewOllama("", "m", o...) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := &deadlineTransport{}
			p, err := tc.build(WithHTTPClient(&http.Client{Transport: rec}), WithBaseURL(srv.URL))
			if err != nil {
				t.Fatalf("construct: %v", err)
			}
			_, _ = p.DiscoverModels(context.Background())
			left, ok := rec.left(tc.listing)
			if !ok {
				t.Fatalf("no %s request was made", tc.listing)
			}
			switch {
			case left < 0:
				t.Errorf("%s ran without a deadline, want one within 10s", tc.listing)
			case left > 10*time.Second:
				t.Errorf("%s ran with %v left, want at most 10s", tc.listing, left.Round(time.Second))
			}
		})
	}
}
