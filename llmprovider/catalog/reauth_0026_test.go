package catalog

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// invalidatingSource counts its invalidations, and hands out "k<n>".
type invalidatingSource struct{ invalidated atomic.Int32 }

func (s *invalidatingSource) Token(context.Context) (llmprovider.Token, error) {
	return llmprovider.Token{Value: "k" + strconv.Itoa(int(s.invalidated.Load()))}, nil
}

func (s *invalidatingSource) Invalidate() { s.invalidated.Add(1) }

// TestList_RerunsCredentialOn401 (0026-MADR F36, Q3 a): a listing refused
// with 401 renews an invalidating source once and lists live, as Generate
// does. One still refused after the rerun degrades to the static catalog,
// with the refusal in Catalog.Err.
func TestList_RerunsCredentialOn401(t *testing.T) {
	for _, c := range []struct {
		name         string
		refusals     int32
		wantLive     bool
		wantRequests int32
	}{
		{"renewed", 1, true, 2},
		{"still refused", 2, false, 2},
	} {
		t.Run(c.name, func(t *testing.T) {
			var requests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if requests.Add(1) <= c.refusals {
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = io.WriteString(w, `{"error":{"message":"invalid api key"}}`)
					return
				}
				_, _ = io.WriteString(w, `[{"id":"meta-llama/Llama-4-Scout-17B-16E-Instruct","type":"chat"}]`)
			}))
			defer srv.Close()
			src := &invalidatingSource{}
			cat, err := List(context.Background(), llmprovider.ProviderTogether, src, llmprovider.WithBaseURL(srv.URL),
				llmprovider.WithoutModelMetadata())
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if cat.Live != c.wantLive || requests.Load() != c.wantRequests || src.invalidated.Load() != 1 {
				t.Fatalf("Live=%t requests=%d invalidations=%d Err=%v; want Live=%t, %d requests, 1 invalidation",
					cat.Live, requests.Load(), src.invalidated.Load(), cat.Err, c.wantLive, c.wantRequests)
			}
			if !c.wantLive && !errors.Is(cat.Err, llmprovider.ErrAuthFailure) {
				t.Errorf("Err = %v; want the refusal, ErrAuthFailure", cat.Err)
			}
		})
	}
	// A static key cannot renew: one request, and the static catalog.
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	cat, _ := List(context.Background(), llmprovider.ProviderTogether, llmprovider.NewStaticToken("k"),
		llmprovider.WithBaseURL(srv.URL), llmprovider.WithoutModelMetadata())
	if cat.Live || requests.Load() != 1 {
		t.Errorf("a static key: Live=%t after %d requests; want the static catalog after 1", cat.Live, requests.Load())
	}
}
