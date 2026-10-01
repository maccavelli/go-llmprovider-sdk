package llmprovider

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// failing returns a provider that fails with errs in turn, then succeeds, and
// counts its calls.
func failing(errs ...error) (*stubProvider, *atomic.Int32) {
	var calls atomic.Int32
	return &stubProvider{id: "stub", caps: Capabilities{Tools: Supported}, gen: func(context.Context, *Request) (*Response, error) {
		n := int(calls.Add(1))
		if n <= len(errs) {
			return nil, errs[n-1]
		}
		return &Response{ID: "ok"}, nil
	}}, &calls
}

var fastRetry = RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: time.Second}

// sameError reports whether got is want, returned without wrapping.
func sameError(got, want error) bool {
	return errors.Is(got, want) && got != nil && got.Error() == want.Error()
}

func TestWithRetry_RetriesByKind(t *testing.T) {
	for _, c := range []struct {
		name  string
		err   error
		calls int32
	}{
		{"a rate limit", &APIError{Status: 429, Kind: ErrRateLimited}, 2},
		{"an unavailable service", &APIError{Status: 503, Kind: ErrProviderUnavailable}, 2},
		{"a stream rate limit", &APIError{Kind: ErrRateLimited}, 2},
		{"a failure to reach the service", errors.New("dial tcp: connection refused"), 2},
		{"exhausted quota", &APIError{Status: 429, Kind: ErrQuotaExhausted, terminal: true}, 1},
		{"exhausted quota, not terminal", &APIError{Status: 402, Kind: ErrQuotaExhausted}, 1},
		{"a final unavailable service", &APIError{Status: 525, Kind: ErrProviderUnavailable, terminal: true}, 1},
		{"an invalid request", &APIError{Status: 400, Kind: ErrInvalidRequest}, 1},
		{"a context overflow", &APIError{Status: 400, Kind: ErrContextOverflow}, 1},
		{"an authentication failure", ErrAuthFailure, 1},
		{"a refusal", ErrNotPermitted, 1},
		{"a cut-short response", &APIError{Kind: ErrIncomplete, Reason: "length"}, 1},
		{"an unsupported need", ErrUnsupported, 1},
		{"an unknown provider", ErrInvalidProvider, 1},
		{"a quota sentinel", ErrQuotaExhausted, 1},
		{"an unavailable sentinel", ErrProviderUnavailable, 2},
	} {
		t.Run(c.name, func(t *testing.T) {
			p, calls := failing(c.err)
			_, err := WithRetry(p, fastRetry).Generate(context.Background(), &Request{})
			if got := calls.Load(); got != c.calls {
				t.Fatalf("%d calls, want %d (err %v)", got, c.calls, err)
			}
			if c.calls == 1 && !sameError(err, c.err) {
				t.Fatalf("err = %v, want the failure itself", err)
			}
		})
	}
}

func TestWithRetry_HonoursRetryAfter(t *testing.T) {
	const asked = 40 * time.Millisecond
	p, calls := failing(&APIError{Status: 429, Kind: ErrRateLimited, RetryAfter: asked})
	start := time.Now()
	resp, err := WithRetry(p, fastRetry).Generate(context.Background(), &Request{})
	if err != nil || resp.ID != "ok" || calls.Load() != 2 {
		t.Fatalf("resp %+v, err %v, calls %d", resp, err, calls.Load())
	}
	if waited := time.Since(start); waited < asked {
		t.Fatalf("waited %v, want at least the %v the service asked for", waited, asked)
	}
}

func TestWithRetry_ReturnsAtOnceWhenTheServiceAsksTooMuch(t *testing.T) {
	tooLong := &APIError{Status: 429, Kind: ErrRateLimited, RetryAfter: time.Hour}
	p, calls := failing(tooLong)
	start := time.Now()
	_, err := WithRetry(p, fastRetry).Generate(context.Background(), &Request{})
	if !sameError(err, tooLong) || calls.Load() != 1 || time.Since(start) > time.Second {
		t.Fatalf("err %v, calls %d, after %v; want the error at once", err, calls.Load(), time.Since(start))
	}
}

func TestWithRetry_StopsAfterMaxAttempts(t *testing.T) {
	unavailable := &APIError{Status: 503, Kind: ErrProviderUnavailable}
	p, calls := failing(unavailable, unavailable, unavailable, unavailable)
	_, err := WithRetry(p, fastRetry).Generate(context.Background(), &Request{})
	if calls.Load() != 3 || !errors.Is(err, ErrProviderUnavailable) || !strings.HasPrefix(err.Error(), "llmprovider: stub failed after 3 attempts: ") {
		t.Fatalf("calls %d, err %v", calls.Load(), err)
	}
}

func TestWithRetry_StopsWhenTheContextEnds(t *testing.T) {
	unavailable := &APIError{Status: 503, Kind: ErrProviderUnavailable}
	p, calls := failing(unavailable, unavailable)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(20*time.Millisecond, cancel)
	start := time.Now()
	_, err := WithRetry(p, RetryPolicy{BaseDelay: time.Minute, MaxDelay: time.Minute}).Generate(ctx, &Request{})
	if !errors.Is(err, context.Canceled) || calls.Load() != 1 || time.Since(start) > 5*time.Second {
		t.Fatalf("err %v, calls %d, after %v; want context.Canceled during the wait", err, calls.Load(), time.Since(start))
	}

	// A failure after the context ended is returned as it is.
	p, calls = failing(unavailable)
	ctx, cancel = context.WithCancel(context.Background())
	p.gen = func(context.Context, *Request) (*Response, error) {
		calls.Add(1)
		cancel()
		return nil, unavailable
	}
	if _, err := WithRetry(p, fastRetry).Generate(ctx, &Request{}); !sameError(err, unavailable) || calls.Load() != 1 {
		t.Fatalf("err %v, calls %d; want the failure after one call", err, calls.Load())
	}
}

func TestWithRetry_DelegatesIDAndCapabilities(t *testing.T) {
	p, _ := failing()
	r := WithRetry(p, RetryPolicy{})
	if r.ID() != "stub" || r.Capabilities() != p.caps {
		t.Fatalf("ID %q, capabilities %+v", r.ID(), r.Capabilities())
	}
}

func TestRetryPolicy_Defaults(t *testing.T) {
	got := RetryPolicy{}.withDefaults()
	if got.MaxAttempts != 3 || got.BaseDelay != time.Second || got.MaxDelay != 30*time.Second {
		t.Fatalf("defaults %+v", got)
	}
}

func TestRetryPolicy_WaitBackoffAndCap(t *testing.T) {
	p := RetryPolicy{MaxAttempts: 9, BaseDelay: 100 * time.Millisecond, MaxDelay: time.Second}
	plain := errors.New("dial")
	for attempt, lo := range map[int]time.Duration{1: 100 * time.Millisecond, 2: 200 * time.Millisecond, 3: 400 * time.Millisecond} {
		got, ok := p.wait(attempt, plain)
		if !ok || got < lo || got > lo+lo/4 {
			t.Errorf("wait(%d) = %v, %v; want %v plus up to a quarter", attempt, got, ok, lo)
		}
	}
	for _, attempt := range []int{5, 64} {
		if got, ok := p.wait(attempt, plain); !ok || got != time.Second {
			t.Errorf("wait(%d) = %v, %v; want the 1s cap", attempt, got, ok)
		}
	}
}
