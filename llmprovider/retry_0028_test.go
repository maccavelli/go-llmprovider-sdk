package llmprovider_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

var request0028 = &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}

// TestWithRetry_HeaderTimeoutSentTwice (0028-MADR D-H1): a generation that
// outlives the client's wait for headers was written whole and may be
// billed. Under WithRetry's three attempts it is sent at most twice, on
// HTTP/1.1 and on HTTP/2.
func TestWithRetry_HeaderTimeoutSentTwice(t *testing.T) {
	for _, h2 := range []bool{false, true} {
		name := "HTTP/1.1"
		if h2 {
			name = "HTTP/2"
		}
		t.Run(name, func(t *testing.T) {
			var got atomic.Int32
			srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				got.Add(1)
				time.Sleep(300 * time.Millisecond)
			}))
			srv.EnableHTTP2 = h2
			srv.StartTLS()
			defer srv.Close()
			base := srv.Client().Transport.(*http.Transport)
			client := &http.Client{Transport: &http.Transport{
				TLSClientConfig:       base.TLSClientConfig.Clone(),
				ForceAttemptHTTP2:     true,
				ResponseHeaderTimeout: 50 * time.Millisecond,
			}}
			p, err := providers.New(llmprovider.ProviderOpenAI, llmprovider.WithAPIKey("k"), llmprovider.WithModel("m"),
				llmprovider.WithBaseURL(srv.URL), llmprovider.WithHTTPClient(client))
			if err != nil {
				t.Fatal(err)
			}
			_, err = llmprovider.WithRetry(p, llmprovider.RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond}).
				Generate(context.Background(), request0028)
			if n := got.Load(); err == nil || n != 2 {
				t.Fatalf("the service received %d request(s), error %v; want 2 and an error", n, err)
			}
		})
	}
}

// TestWithRetry_ResetPastMaxDelayReturnsAtOnce (0028-MADR D-H5): a 429 whose
// reset header names a wait past MaxDelay (30 s by default) is returned at
// once, for the caller to reschedule, rather than retried on the backoff.
func TestWithRetry_ResetPastMaxDelayReturnsAtOnce(t *testing.T) {
	var got atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		got.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("x-ratelimit-reset-requests", "60s")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"error":{"message":"Rate limit reached","type":"rate_limit_exceeded"}}`)
	}))
	defer srv.Close()
	p, err := providers.New(llmprovider.ProviderTogether, llmprovider.WithAPIKey("k"), llmprovider.WithModel("m"),
		llmprovider.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = llmprovider.WithRetry(p, llmprovider.RetryPolicy{BaseDelay: time.Millisecond}).Generate(context.Background(), request0028)
	if n := got.Load(); err == nil || n != 1 {
		t.Fatalf("the service received %d request(s), error %v; want 1 and the rate limit", n, err)
	}
}
