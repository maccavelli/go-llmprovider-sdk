package llmprovider

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestParseRetryAfter(t *testing.T) {
	if got := parseRetryAfter("120"); got != 120*time.Second {
		t.Errorf("seconds: got %v", got)
	}
	if got := parseRetryAfter(""); got != 0 {
		t.Errorf("empty: got %v", got)
	}
	if got := parseRetryAfter("not-a-number"); got != 0 {
		t.Errorf("garbage: got %v", got)
	}
}

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
