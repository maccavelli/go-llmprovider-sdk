package transport

import (
	"net/http"
	"testing"
	"time"
)

// TestRetryAfter_HugeIsALongWait (0020-MADR F31): a delay too long for a
// Duration is the longest one, not an overflow. On amd64 the overflow was
// negative, which WithRetry reads as no delay; run with GOARCH=amd64 to see it.
func TestRetryAfter_HugeIsALongWait(t *testing.T) {
	for _, h := range []http.Header{
		{"Retry-After": {"99999999999"}},
		{"Retry-After-Ms": {"99999999999999"}},
	} {
		if got := RetryAfter(h); got < 24*time.Hour {
			t.Errorf("RetryAfter(%v) = %v, want a long positive wait", h, got)
		}
	}
}
