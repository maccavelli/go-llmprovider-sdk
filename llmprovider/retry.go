package llmprovider

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"math/rand/v2"
	"net/url"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// RetryPolicy says how WithRetry retries (0015-MADR D8). It is safe for
// concurrent use only while no goroutine changes it.
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

// RetryNotice describes a retry that WithRetryNotify is about to wait for
// (0044-MADR D1). It is safe for concurrent use only while no goroutine
// changes it.
type RetryNotice struct {
	// Provider is the wrapped provider's ID.
	Provider ProviderID
	// Attempt is the try that failed, the first being 1.
	Attempt int
	// MaxAttempts is the policy's, after its defaults.
	MaxAttempts int
	// Wait is the wait about to begin, jitter included.
	Wait time.Duration
	// ServerDirected reports that the service asked for the wait, by
	// Retry-After or a rate-limit reset header (APIError.RetryAfter).
	ServerDirected bool
	// Err is the failure being retried, classified and redacted, as Generate
	// would return it if it stopped here.
	Err error
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
//     cut connection, a stream that ended early, or a failure the service
//     reports inside the stream (0053-MADR D3), is retried at most once per
//     Generate: the lost reply may have been generated and billed (0021-MADR
//     D1). So is a failure after the request was written, such as a timeout
//     awaiting the reply's headers (0028-MADR D-H1). That retry counts
//     against MaxAttempts.
//   - It waits as long as the service asks, plus up to a tenth more (at least
//     250 ms) so that callers limited together do not return together, and at
//     most MaxDelay.
//   - A wait that would outlast ctx's deadline is not begun: the last error is
//     returned at once, so the caller can reschedule. When ctx ends during a
//     wait, the error matches both ctx.Err() and the last failure.
//
// The result does not stream natively, even when p does; Stream falls back to
// its Generate. That buffers the stream so that it can be retried whole: a
// failure at any point is retried, and the caller receives events only from
// the attempt that succeeds, never text it already holds. A caller that wants
// text as it is generated streams from p itself, and handles a failure there
// (0031-MADR D6), or uses WithStreamRetry, which retries a failure only
// before the first event (0051-MADR). When p lists models, so does the
// result, through p (0020-MADR F32). WithRetryNotify also reports each retry
// before its wait.
func WithRetry(p Provider, policy RetryPolicy) Provider {
	return WithRetryNotify(p, policy, nil)
}

// WithRetryNotify is WithRetry, and calls notify before each wait, so that a
// caller can say that a retry is pending, why, and for how long (0044-MADR).
// It retries exactly as WithRetry does.
//
// notify is called only when a wait is about to begin: never for a success,
// a failure that is not retried, the last attempt, a wait longer than
// MaxDelay or than ctx's deadline allows, or a second failure after a reply.
// It runs on the goroutine of Generate, with that call's ctx. The wait begins
// when it returns, so it must not block, and a panic in it is not recovered.
// Through llmprovider.Stream the notices come before the first event, since
// the result buffers the stream. A nil notify is WithRetry.
func WithRetryNotify(p Provider, policy RetryPolicy, notify func(context.Context, RetryNotice)) Provider {
	r := &retrying{inner: p, policy: policy.withDefaults(), notify: notify}
	if lister, ok := p.(ModelLister); ok {
		return &retryingLister{retrying: r, lister: lister}
	}
	return r
}

type retrying struct {
	inner  Provider
	policy RetryPolicy
	// notify is WithRetryNotify's, or nil.
	notify func(context.Context, RetryNotice)
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
		if final := r.after(ctx, attempt, err, &retriedAfterReply); final != nil {
			return nil, final
		}
	}
}

// after decides attempt's failure err: it waits and returns nil when the
// request is to be sent again, or returns the error to give the caller.
// Generate and a streaming retry's Stream share it, so that both retry alike
// (0051-MADR D2). retriedAfterReply is the call's own.
func (r *retrying) after(ctx context.Context, attempt int, err error, retriedAfterReply *bool) error {
	if ctx.Err() != nil || !retryable(err) {
		return err
	}
	if transport.IsAfterReply(err) {
		// Once per Generate or Stream call (0021-MADR D1).
		if *retriedAfterReply {
			return err
		}
		*retriedAfterReply = true
	}
	if attempt >= r.policy.MaxAttempts {
		return fmt.Errorf("llmprovider: %s failed after %d attempts: %w", r.inner.ID(), attempt, err)
	}
	wait, ok := r.policy.wait(attempt, err)
	if !ok {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < wait {
		// The wait would outlast the caller (0021-MADR T6).
		return err
	}
	if r.notify != nil {
		// Only now is a wait certain (0044-MADR D2).
		r.notify(ctx, RetryNotice{Provider: r.inner.ID(), Attempt: attempt, MaxAttempts: r.policy.MaxAttempts,
			Wait: wait, ServerDirected: serverRetryAfter(err) > 0, Err: err})
	}
	timer := time.NewTimer(wait)
	select {
	case <-ctx.Done():
		timer.Stop()
		return errors.Join(ctx.Err(), err)
	case <-timer.C:
		return nil
	}
}

// WithStreamRetry is WithRetryNotify for a caller that wants text as it is
// generated (0051-MADR). Over a provider that streams natively, its result
// streams natively too, and its Stream retries only until the first event:
//
//   - A failure before the first event, when the caller has seen nothing, is
//     retried exactly as Generate retries one: the same kinds, the once-only
//     resend of a failure after the request was written or the reply begun
//     (once per Stream call), the waits, MaxDelay, ctx's deadline, and notify
//     before each wait, on the goroutine ranging the stream. The failure that
//     ends the stream there is yielded once, with a zero Event: the inner
//     stream's error, or the "failed after N attempts" error at the last
//     attempt, or one matching both ctx.Err() and the failure when ctx ends
//     during a wait.
//   - After the first event (a text or reasoning delta, or an item) every
//     event passes through as the inner stream yields it, and a failure ends
//     the stream with that error, unchanged, and no EventDone. It is not
//     retried, so a caller loses the generation where WithRetry would have
//     retried it; the events already yielded are not a response (0031-MADR
//     D4).
//
// A caller that stops ranging stops the inner stream. Generate retries the
// whole generation, as WithRetryNotify's does; Capabilities is p's,
// NativeStreaming included; ListModels lists through p when p lists. Over a
// provider that does not stream natively there is nothing to pass through:
// the result is WithRetryNotify(p, policy, notify)'s. A nil notify reports
// nothing. The result is safe for concurrent use, as p is.
func WithStreamRetry(p Provider, policy RetryPolicy, notify func(context.Context, RetryNotice)) Provider {
	s, ok := p.(Streamer)
	if !ok {
		// Not a Streamer, as R10 requires of a provider that does not stream
		// (0051-MADR D3).
		return WithRetryNotify(p, policy, notify)
	}
	r := &streamRetrying{retrying: &retrying{inner: p, policy: policy.withDefaults(), notify: notify}, streamer: s}
	if lister, ok := p.(ModelLister); ok {
		return &streamRetryingLister{streamRetrying: r, lister: lister}
	}
	return r
}

// streamRetrying is WithStreamRetry's result over a Streamer.
type streamRetrying struct {
	*retrying
	streamer Streamer
}

// streamRetryingLister is streamRetrying for a provider that lists models.
type streamRetryingLister struct {
	*streamRetrying
	lister ModelLister
}

// ListModels lists through the inner provider, once.
func (r *streamRetryingLister) ListModels(ctx context.Context) ([]string, error) {
	return r.lister.ListModels(ctx)
}

// Capabilities is the inner provider's, NativeStreaming included: the wrapper
// streams as it does (0051-MADR D3).
func (r *streamRetrying) Capabilities() Capabilities { return r.inner.Capabilities() }

// Stream retries a failure before the first event, and passes every event
// after it through (0051-MADR D2).
func (r *streamRetrying) Stream(ctx context.Context, req *Request) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		retriedAfterReply := false
		for attempt := 1; ; attempt++ {
			started := false
			var failure error
			for ev, err := range r.streamer.Stream(ctx, req) {
				if err != nil && !started {
					failure = err
					break
				}
				started = true
				if !yield(ev, err) {
					return
				}
			}
			if failure == nil {
				return
			}
			if final := r.after(ctx, attempt, failure, &retriedAfterReply); final != nil {
				yield(Event{}, final)
				return
			}
		}
	}
}

// minServerJitter is the least spread added to a wait the service asked for.
const minServerJitter = 250 * time.Millisecond

// wait is the delay before the try after attempt, and false when the service
// asked for longer than MaxDelay (0021-MADR T7).
//
//   - A wait the service asked for gets up to a tenth more, at least 250 ms,
//     so that callers limited together spread out, and is then capped at
//     MaxDelay (0026-MADR F22).
//   - Backoff doubles from BaseDelay up to MaxDelay, and the wait taken is
//     between half and all of it ("equal jitter"), chosen after the cap so
//     that callers at the cap do not all wait exactly MaxDelay.
func (p RetryPolicy) wait(attempt int, err error) (time.Duration, bool) {
	if server := serverRetryAfter(err); server > 0 {
		//nolint:gosec // G404: non-crypto jitter for retry spacing
		return min(server+rand.N(max(server/10, minServerJitter)), p.MaxDelay), server <= p.MaxDelay
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
// marked so that Generate retries it once (0021-MADR D1); so is a failure
// after the request was written, such as a timeout awaiting the reply's
// headers (0028-MADR D-H1).
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
		// A failure to send is retried, unless no retry can mend it
		// (0026-MADR F11).
		var urlErr *url.Error
		return errors.As(err, &urlErr) && !transport.Unsendable(err)
	}
}
