package llmprovider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// notices records what WithRetryNotify reports.
type notices struct {
	mu   sync.Mutex
	list []RetryNotice
}

func (n *notices) notify(_ context.Context, notice RetryNotice) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.list = append(n.list, notice)
}

func (n *notices) all() []RetryNotice {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]RetryNotice(nil), n.list...)
}

// TestRetryNotify_ServerDirected_0044: a 429 with RetryAfter, then a success,
// sends one notice with every field (0044-MADR D1).
func TestRetryNotify_ServerDirected_0044(t *testing.T) {
	const asked = 40 * time.Millisecond
	limited := &APIError{Status: 429, Kind: ErrRateLimited, RetryAfter: asked}
	p, calls := failing(limited)
	var got notices
	var at time.Duration
	start := time.Now()
	notify := func(ctx context.Context, n RetryNotice) {
		at = time.Since(start)
		got.notify(ctx, n)
	}
	resp, err := WithRetryNotify(p, fastRetry, notify).Generate(context.Background(), &Request{})
	if err != nil || resp.ID != "ok" || calls.Load() != 2 {
		t.Fatalf("resp %+v, err %v, calls %d", resp, err, calls.Load())
	}
	if at >= asked {
		t.Errorf("notified after %v; want before the %v wait", at, asked)
	}
	list := got.all()
	if len(list) != 1 {
		t.Fatalf("notices %+v; want one", list)
	}
	n := list[0]
	if n.Provider != "stub" || n.Attempt != 1 || n.MaxAttempts != 3 || !n.ServerDirected || !sameError(n.Err, limited) {
		t.Errorf("notice %+v; want stub, attempt 1 of 3, server-directed, the 429", n)
	}
	if n.Wait < asked || n.Wait > asked+minServerJitter {
		t.Errorf("Wait %v; want the %v asked for, plus at most %v", n.Wait, asked, minServerJitter)
	}
}

// TestRetryNotify_Backoff_0044: backoff's waits are reported, not
// server-directed, within the equal-jitter range, with the policy's
// defaulted MaxAttempts.
func TestRetryNotify_Backoff_0044(t *testing.T) {
	unavailable := &APIError{Status: 503, Kind: ErrProviderUnavailable}
	p, _ := failing(unavailable, unavailable)
	var got notices
	policy := RetryPolicy{BaseDelay: 2 * time.Millisecond, MaxDelay: time.Second}
	if _, err := WithRetryNotify(p, policy, got.notify).Generate(context.Background(), &Request{}); err != nil {
		t.Fatal(err)
	}
	list := got.all()
	if len(list) != 2 {
		t.Fatalf("notices %+v; want two", list)
	}
	for i, n := range list {
		full := policy.BaseDelay << i
		if n.Attempt != i+1 || n.MaxAttempts != 3 || n.ServerDirected || n.Wait < full/2 || n.Wait > full {
			t.Errorf("notice %d %+v; want attempt %d of 3, not server-directed, a wait in [%v, %v]", i, n, i+1, full/2, full)
		}
	}
}

// TestRetryNotify_NoWaitNoNotice_0044: where no wait follows, notify is not
// called, and Generate returns as WithRetry's does (0044-MADR D2).
func TestRetryNotify_NoWaitNoNotice_0044(t *testing.T) {
	cut := fmt.Errorf("%w: p: read reply: %w", ErrProviderUnavailable, transport.AfterReply(io.ErrUnexpectedEOF))
	unavailable := &APIError{Status: 503, Kind: ErrProviderUnavailable}
	for _, c := range []struct {
		name   string
		errs   []error
		policy RetryPolicy
		ctxFor time.Duration
		want   int
	}{
		{"a terminal error", []error{&APIError{Status: 400, Kind: ErrInvalidRequest}}, fastRetry, 0, 0},
		{"the last attempt", []error{unavailable, unavailable}, RetryPolicy{MaxAttempts: 2, BaseDelay: time.Millisecond}, 0, 1},
		{"a wait over MaxDelay", []error{&APIError{Status: 429, Kind: ErrRateLimited, RetryAfter: 5 * time.Second}},
			RetryPolicy{MaxDelay: time.Second}, 0, 0},
		{"a wait past the deadline", []error{&APIError{Status: 429, Kind: ErrRateLimited, RetryAfter: 2 * time.Second}},
			RetryPolicy{MaxDelay: time.Minute}, 500 * time.Millisecond, 0},
		{"a second failure after a reply", []error{cut, cut}, RetryPolicy{MaxAttempts: 5, BaseDelay: time.Millisecond}, 0, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			if c.ctxFor > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, c.ctxFor)
				defer cancel()
			}
			p, _ := failing(c.errs...)
			var got notices
			_, err := WithRetryNotify(p, c.policy, got.notify).Generate(ctx, &Request{})
			if err == nil || len(got.all()) != c.want {
				t.Errorf("err %v, notices %+v; want an error and %d notice(s)", err, got.all(), c.want)
			}
		})
	}
}

type noticeKey struct{}

// TestRetryNotify_CallersContext_0044: notify gets the caller's ctx; one that
// cancels it ends the call with no further try, the error both the
// context's and the failure.
func TestRetryNotify_CallersContext_0044(t *testing.T) {
	unavailable := &APIError{Status: 503, Kind: ErrProviderUnavailable}
	p, calls := failing(unavailable)
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), noticeKey{}, "request-7"))
	defer cancel()
	var seen any
	_, err := WithRetryNotify(p, RetryPolicy{BaseDelay: time.Minute, MaxDelay: time.Minute},
		func(ctx context.Context, _ RetryNotice) {
			seen = ctx.Value(noticeKey{})
			cancel()
		}).Generate(ctx, &Request{})
	if seen != "request-7" {
		t.Errorf("notify's ctx value %v; want the caller's", seen)
	}
	if calls.Load() != 1 || !errors.Is(err, context.Canceled) || !errors.Is(err, ErrProviderUnavailable) {
		t.Errorf("calls %d, err %v; want one call, and an error that is both", calls.Load(), err)
	}
}

// TestRetryNotify_NilIsWithRetry_0044: a nil notify retries as WithRetry.
func TestRetryNotify_NilIsWithRetry_0044(t *testing.T) {
	p, calls := failing(&APIError{Status: 503, Kind: ErrProviderUnavailable})
	resp, err := WithRetryNotify(p, fastRetry, nil).Generate(context.Background(), &Request{})
	if err != nil || resp.ID != "ok" || calls.Load() != 2 {
		t.Fatalf("resp %+v, err %v, calls %d; want a retry and the answer", resp, err, calls.Load())
	}
}

// TestRetryNotify_KeepsLister_0044: the result lists when p does
// (0020-MADR F32; listingStub is hygiene_test.go's).
func TestRetryNotify_KeepsLister_0044(t *testing.T) {
	inner := &listingStub{stubProvider{id: "stub", caps: Capabilities{Tools: Supported}}}
	lister, ok := WithRetryNotify(inner, fastRetry, func(context.Context, RetryNotice) {}).(ModelLister)
	if !ok {
		t.Fatal("the result does not list models")
	}
	if models, err := lister.ListModels(context.Background()); err != nil || len(models) != 1 {
		t.Errorf("ListModels = %v, %v", models, err)
	}
}

// TestRetryNotify_Stream_0044: through Stream, the notice comes before the
// first event (0044-MADR D3).
func TestRetryNotify_Stream_0044(t *testing.T) {
	p, _ := failing(&APIError{Status: 503, Kind: ErrProviderUnavailable})
	var order []string
	notify := func(context.Context, RetryNotice) { order = append(order, "notice") }
	for _, err := range Stream(context.Background(), WithRetryNotify(p, fastRetry, notify), &Request{}) {
		if err != nil {
			t.Fatal(err)
		}
		order = append(order, "event")
	}
	if len(order) < 2 || order[0] != "notice" || order[1] != "event" {
		t.Errorf("order %q; want the notice, then the events", order)
	}
}
