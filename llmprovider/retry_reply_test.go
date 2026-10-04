package llmprovider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// TestWithRetry_AfterReplyOnce (0021-MADR D1, T5): a failure while reading a
// reply is retried once per Generate, whatever MaxAttempts allows.
func TestWithRetry_AfterReplyOnce(t *testing.T) {
	cut := fmt.Errorf("%w: p: read reply: %w", ErrProviderUnavailable, transport.AfterReply(io.ErrUnexpectedEOF))
	p, calls := failing(cut, cut, cut, cut, cut)
	_, err := WithRetry(p, RetryPolicy{MaxAttempts: 5, BaseDelay: time.Millisecond}).Generate(context.Background(), &Request{})
	if calls.Load() != 2 || !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("calls = %d, err = %v; want 2 calls: retried once", calls.Load(), err)
	}
	// A failure before the reply keeps the usual budget after one after it.
	unavailable := &APIError{Status: 503, Kind: ErrProviderUnavailable}
	p, calls = failing(cut, unavailable, unavailable)
	if _, err := WithRetry(p, RetryPolicy{MaxAttempts: 5, BaseDelay: time.Millisecond}).Generate(context.Background(), &Request{}); err != nil || calls.Load() != 4 {
		t.Fatalf("calls = %d, err = %v; want 4 calls and an answer", calls.Load(), err)
	}
	// With no retries allowed, nothing is retried.
	p, calls = failing(cut)
	if _, err := WithRetry(p, RetryPolicy{MaxAttempts: 1}).Generate(context.Background(), &Request{}); err == nil || calls.Load() != 1 {
		t.Fatalf("MaxAttempts 1: calls = %d, err = %v; want one call", calls.Load(), err)
	}
}

// TestWithRetry_TooLargeNotRetried (0021-MADR W6): a reply over the limit was
// sent whole, and is not bought again.
func TestWithRetry_TooLargeNotRetried(t *testing.T) {
	p, calls := failing(fmt.Errorf("%w: p: reply over 16 MiB", ErrIncomplete))
	if _, err := WithRetry(p, fastRetry).Generate(context.Background(), &Request{}); !errors.Is(err, ErrIncomplete) || calls.Load() != 1 {
		t.Fatalf("calls = %d, err = %v; want one call", calls.Load(), err)
	}
}

// TestWithRetry_WaitPastDeadlineReturnsAtOnce (0021-MADR T6): a wait the
// caller's deadline cannot hold is not begun, and the caller keeps the error's
// kind and RetryAfter.
func TestWithRetry_WaitPastDeadlineReturnsAtOnce(t *testing.T) {
	limited := &APIError{Status: 429, Kind: ErrRateLimited, RetryAfter: 5 * time.Second}
	p, calls := failing(limited, limited)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	start := time.Now()
	_, err := WithRetry(p, RetryPolicy{MaxDelay: time.Minute}).Generate(ctx, &Request{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.RetryAfter != 5*time.Second || calls.Load() != 1 {
		t.Fatalf("err = %v, calls = %d; want the 429 with its RetryAfter after one call", err, calls.Load())
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("returned after %s, want at once", elapsed)
	}
}

// TestWithRetry_CancelledWaitKeepsError (0021-MADR T6): a wait cut short
// returns an error that is both the context's and the last failure.
func TestWithRetry_CancelledWaitKeepsError(t *testing.T) {
	unavailable := &APIError{Status: 503, Kind: ErrProviderUnavailable}
	p, _ := failing(unavailable, unavailable)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(20*time.Millisecond, cancel)
	_, err := WithRetry(p, RetryPolicy{BaseDelay: time.Minute, MaxDelay: time.Minute}).Generate(ctx, &Request{})
	var apiErr *APIError
	if !errors.Is(err, context.Canceled) || !errors.As(err, &apiErr) || apiErr.Status != 503 {
		t.Fatalf("err = %v; want context.Canceled and the 503", err)
	}
}

// TestRetryPolicy_ServerWaitJitter (0021-MADR T7): a wait the service asked
// for gets up to a tenth more, at least 250 ms, so callers limited together
// spread out.
func TestRetryPolicy_ServerWaitJitter(t *testing.T) {
	p := RetryPolicy{}.withDefaults()
	limited := &APIError{Status: 429, Kind: ErrRateLimited, RetryAfter: time.Second}
	seen := map[time.Duration]bool{}
	for range 1000 {
		got, ok := p.wait(1, limited)
		if !ok || got < time.Second || got >= time.Second+250*time.Millisecond {
			t.Fatalf("wait = %v, %v; want in [1s, 1.25s)", got, ok)
		}
		seen[got] = true
	}
	if len(seen) < 2 {
		t.Fatal("every wait was the same: no jitter")
	}
}

// TestRetryPolicy_BackoffEqualJitterAfterCap (0021-MADR T7): at the cap, waits
// spread over the upper half instead of all being MaxDelay.
func TestRetryPolicy_BackoffEqualJitterAfterCap(t *testing.T) {
	p := RetryPolicy{BaseDelay: time.Second, MaxDelay: 4 * time.Second}.withDefaults()
	seen := map[time.Duration]bool{}
	for range 1000 {
		got, ok := p.wait(10, errors.New("dial"))
		if !ok || got < 2*time.Second || got > 4*time.Second {
			t.Fatalf("wait = %v, %v; want in [2s, 4s]", got, ok)
		}
		seen[got] = true
	}
	if len(seen) < 2 {
		t.Fatal("every wait at the cap was the same")
	}
}

// TestClassifyHTTPError_ShouldRetryTrue (0021-MADR T14): x-should-retry: true
// makes any status retryable, as the OpenAI and Anthropic SDKs obey it; the
// kind is unchanged.
func TestClassifyHTTPError_ShouldRetryTrue(t *testing.T) {
	err := classifyFixture("openai", http.StatusBadRequest, `{}`, http.Header{"X-Should-Retry": {"true"}})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !apiErr.Retryable() || !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("err = %v; want a retryable ErrInvalidRequest", err)
	}
	p, calls := failing(err)
	if _, err := WithRetry(p, fastRetry).Generate(context.Background(), &Request{}); err != nil || calls.Load() != 2 {
		t.Fatalf("calls = %d, err = %v; want a retry that succeeds", calls.Load(), err)
	}
}

// TestClassifyHTTPError_Conflict (0021-MADR T14): a 409 is a lock timeout to
// OpenAI and Anthropic, and retryable; elsewhere it stays final.
func TestClassifyHTTPError_Conflict(t *testing.T) {
	for provider, retryable := range map[string]bool{"openai": true, "claude": true, "grok": false} {
		err := classifyFixture(provider, http.StatusConflict, `{}`, nil)
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.Retryable() != retryable {
			t.Errorf("%s 409: %v, retryable %v; want %v", provider, err, apiErr != nil && apiErr.Retryable(), retryable)
		}
		if retryable && !errors.Is(err, ErrProviderUnavailable) {
			t.Errorf("%s 409: %v; want ErrProviderUnavailable", provider, err)
		}
		if !retryable && !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s 409: %v; want ErrInvalidRequest", provider, err)
		}
	}
}
