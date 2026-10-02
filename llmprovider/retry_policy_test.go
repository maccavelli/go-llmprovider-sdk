package llmprovider

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestParseRetryAfter_MillisAndFractional(t *testing.T) {
	for _, header := range []http.Header{
		{"Retry-After-Ms": {"1500"}},
		{"Retry-After": {"1.5"}},
	} {
		var rl *APIError
		err := classifyFixture("kilo", http.StatusTooManyRequests, `{}`, header)
		if !errors.As(err, &rl) || rl.RetryAfter != 1500*time.Millisecond {
			t.Errorf("header %v: err = %v, want retry-after 1.5s", header, err)
		}
	}
}

// TestWithRetry_Retries408 was TestRetry_408IsRetried: a classified request
// timeout is transient (0013 B4), though it matches ErrInvalidRequest.
func TestWithRetry_Retries408(t *testing.T) {
	timeout := classifyFixture("huggingface", http.StatusRequestTimeout, `{"error":"timeout"}`, nil)
	p, calls := failing(timeout)
	resp, err := WithRetry(p, fastRetry).Generate(context.Background(), &Request{})
	if err != nil || resp.ID != "ok" || calls.Load() != 2 {
		t.Fatalf("calls = %d, resp = %+v, err = %v; want a retry that succeeds", calls.Load(), resp, err)
	}
}

// TestWithRetry_ShouldRetryFalseIsFinal was
// TestRetry_ShouldRetryFalseIsTerminal: x-should-retry: false makes an
// otherwise retryable error final.
func TestWithRetry_ShouldRetryFalseIsFinal(t *testing.T) {
	unavailable := classifyFixture("openai", http.StatusServiceUnavailable, `{}`, http.Header{"X-Should-Retry": {"false"}})
	p, calls := failing(unavailable, unavailable)
	_, err := WithRetry(p, fastRetry).Generate(context.Background(), &Request{})
	if calls.Load() != 1 || !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("calls = %d, err = %v; want 1 call", calls.Load(), err)
	}
}

// TestRetryPolicy_SubNanosNoPanic was TestGenerateWithRetry_SubNanosNoPanic:
// a delay too small to take a quarter of guards rand.N(0).
func TestRetryPolicy_SubNanosNoPanic(t *testing.T) {
	p := RetryPolicy{MaxAttempts: 2, BaseDelay: time.Nanosecond, MaxDelay: time.Second}
	if got, ok := p.wait(1, errors.New("dial")); !ok || got != time.Nanosecond {
		t.Fatalf("wait(1) = %v, %v; want 1ns", got, ok)
	}
}

// TestRetryPolicy_NegativeIsDefault was TestRetry_NegativeRetriesMakesOneCall
// (0013 B5): a negative value takes the default, never zero attempts.
func TestRetryPolicy_NegativeIsDefault(t *testing.T) {
	got := RetryPolicy{MaxAttempts: -1, BaseDelay: -1, MaxDelay: -1}.withDefaults()
	if got != (RetryPolicy{}).withDefaults() {
		t.Fatalf("withDefaults = %+v, want the defaults", got)
	}
}
