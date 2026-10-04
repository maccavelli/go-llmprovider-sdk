package llmprovider

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/url"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// RetryPolicy says how WithRetry retries (0015-MADR D8).
type RetryPolicy struct {
	// MaxAttempts counts every try, the first included. Zero means 3; one
	// means no retry.
	MaxAttempts int
	// BaseDelay is the wait before the second try. Each later wait doubles,
	// up to MaxDelay; the wait taken is between half and all of it, at
	// random. Zero means one second.
	BaseDelay time.Duration
	// MaxDelay caps each wait. A service asking for a longer wait gets its
	// error returned at once, so the caller can reschedule. Zero means
	// 30 seconds.
	MaxDelay time.Duration
}

// withDefaults fills the zero fields.
func (p RetryPolicy) withDefaults() RetryPolicy {
	if p.MaxAttempts <= 0 {
		p.MaxAttempts = 3
	}
	if p.BaseDelay <= 0 {
		p.BaseDelay = time.Second
	}
	if p.MaxDelay <= 0 {
		p.MaxDelay = retryBackoffCap
	}
	return p
}

// WithRetry returns p with failed generations retried under policy
// (0015-MADR D8). It retries only what can succeed later: a rate limit other
// than exhausted quota, an unavailable service, or a failure to reach it.
//
//   - A failure while reading a reply the service had begun to send, such as a
//     cut connection or a stream that ended early, is retried at most once per
//     Generate: the lost reply may have been generated and billed (0021-MADR
//     D1). That retry counts against MaxAttempts.
//   - It waits as long as the service asks, plus up to a tenth more (at least
//     250 ms) so that callers limited together do not return together, and at
//     most MaxDelay.
//   - A wait that would outlast ctx's deadline is not begun: the last error is
//     returned at once, so the caller can reschedule. When ctx ends during a
//     wait, the error matches both ctx.Err() and the last failure.
//
// The result does not stream natively; Stream falls back to its Generate.
// When p lists models, so does the result, through p (0020-MADR F32).
func WithRetry(p Provider, policy RetryPolicy) Provider {
	r := &retrying{inner: p, policy: policy.withDefaults()}
	if lister, ok := p.(ModelLister); ok {
		return &retryingLister{retrying: r, lister: lister}
	}
	return r
}

type retrying struct {
	inner  Provider
	policy RetryPolicy
}

// retryingLister is retrying for a provider that lists models.
type retryingLister struct {
	*retrying
	lister ModelLister
}

// ListModels lists through the inner provider, once.
func (r *retryingLister) ListModels(ctx context.Context) ([]string, error) {
	return r.lister.ListModels(ctx)
}

func (r *retrying) ID() ProviderID { return r.inner.ID() }

// Capabilities is the inner provider's, without NativeStreaming: the wrapper
// has no Stream (0020-MADR F32).
func (r *retrying) Capabilities() Capabilities {
	caps := r.inner.Capabilities()
	caps.NativeStreaming = Unsupported
	return caps
}

func (r *retrying) Generate(ctx context.Context, req *Request) (*Response, error) {
	retriedAfterReply := false
	for attempt := 1; ; attempt++ {
		resp, err := r.inner.Generate(ctx, req)
		if err == nil {
			return resp, nil
		}
		if ctx.Err() != nil || !retryable(err) {
			return nil, err
		}
		if transport.IsAfterReply(err) {
			// Once per Generate (0021-MADR D1).
			if retriedAfterReply {
				return nil, err
			}
			retriedAfterReply = true
		}
		if attempt >= r.policy.MaxAttempts {
			return nil, fmt.Errorf("llmprovider: %s failed after %d attempts: %w", r.inner.ID(), attempt, err)
		}
		wait, ok := r.policy.wait(attempt, err)
		if !ok {
			return nil, err
		}
		if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < wait {
			// The wait would outlast the caller (0021-MADR T6).
			return nil, err
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, errors.Join(ctx.Err(), err)
		case <-timer.C:
		}
	}
}

// minServerJitter is the least spread added to a wait the service asked for.
const minServerJitter = 250 * time.Millisecond

// wait is the delay before the try after attempt, and false when the service
// asked for longer than MaxDelay (0021-MADR T7).
//
//   - A wait the service asked for gets up to a tenth more, at least 250 ms,
//     so that callers limited together spread out.
//   - Backoff doubles from BaseDelay up to MaxDelay, and the wait taken is
//     between half and all of it ("equal jitter"), chosen after the cap so
//     that callers at the cap do not all wait exactly MaxDelay.
func (p RetryPolicy) wait(attempt int, err error) (time.Duration, bool) {
	if server := serverRetryAfter(err); server > 0 {
		//nolint:gosec // G404: non-crypto jitter for retry spacing
		return server + rand.N(max(server/10, minServerJitter)), server <= p.MaxDelay
	}
	delay := p.BaseDelay << (attempt - 1)
	if delay <= 0 || delay > p.MaxDelay {
		delay = p.MaxDelay
	}
	if half := delay / 2; half > 0 {
		//nolint:gosec // G404: non-crypto jitter for retry backoff spacing
		return half + rand.N(delay-half+1), true
	}
	return delay, true
}

// retryable reports whether the same request can succeed later. A failure
// without a kind is retried only when it is a failure to send the request, a
// *url.Error from the HTTP client; any other was answered and billed already
// (0020-MADR F9, Q2 a). A failure to read a reply is ErrProviderUnavailable,
// marked so that Generate retries it once (0021-MADR D1).
func retryable(err error) bool {
	if apiErr, ok := errors.AsType[*APIError](err); ok {
		return apiErr.Retryable()
	}
	switch {
	case errors.Is(err, ErrQuotaExhausted):
		return false
	case errors.Is(err, ErrRateLimited), errors.Is(err, ErrProviderUnavailable):
		return true
	case errors.Is(err, ErrAuthFailure), errors.Is(err, ErrNotPermitted),
		errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrIncomplete),
		errors.Is(err, ErrUnsupported), errors.Is(err, ErrInvalidProvider):
		return false
	default:
		var urlErr *url.Error
		return errors.As(err, &urlErr)
	}
}
