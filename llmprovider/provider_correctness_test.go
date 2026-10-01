package llmprovider

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestRateLimitError_Classification: RateLimitError unwraps to ErrRateLimited
// (retryable) and is honored by the retry layer.
func TestRateLimitError_Classification(t *testing.T) {
	rl := &RateLimitError{RetryAfter: time.Millisecond, Status: 429}
	if !errors.Is(rl, ErrRateLimited) {
		t.Error("RateLimitError must unwrap to ErrRateLimited")
	}

	f := &fakeProvider{errs: []error{rl}} // fail once with RateLimitError, then succeed
	out, err := GenerateWithRetry(context.Background(), f, "p", 3, time.Nanosecond)
	if err != nil || out != "ok" {
		t.Fatalf("expected retry+success, got %q err=%v", out, err)
	}
	if f.calls != 2 {
		t.Errorf("expected 2 calls (1 rate-limited + 1 ok), got %d", f.calls)
	}
}
