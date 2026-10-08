package transport

import (
	"context"
	"errors"
	"io"
	"time"
)

// StreamIdleTimeout is how long a reply's body may send nothing before the
// request is given up, as the ChatGPT client waits for each stream event
// (Codex's DEFAULT_STREAM_IDLE_TIMEOUT_MS; 0021-MADR D2). It bounds every
// generation body, event stream or JSON, whatever client sends it.
const StreamIdleTimeout = 300 * time.Second

// idleTimeout is the error of a body that sent nothing for StreamIdleTimeout.
// It is a timeout, as a net.Error reports one.
type idleTimeout struct{}

func (idleTimeout) Error() string   { return "no data for 300s" }
func (idleTimeout) Timeout() bool   { return true }
func (idleTimeout) Temporary() bool { return true }

// ErrIdleTimeout is a reply body that sent nothing within the idle limit.
var ErrIdleTimeout error = idleTimeout{}

// ErrReplyTooLarge is a reply body longer than its reader's limit.
var ErrReplyTooLarge = errors.New("reply over the size limit")

// afterReply marks a failure after the service received the whole request:
// while reading a reply it had begun to send, or while waiting for one.
type afterReply struct{ err error }

func (e *afterReply) Error() string { return e.err.Error() }
func (e *afterReply) Unwrap() error { return e.err }

// AfterReply marks err as a failure after the service received the whole
// request: while reading a reply it had begun to send (0021-MADR D1), or
// while waiting for the reply's headers (0028-MADR D-H1). Such a request may
// have been generated and billed, so llmprovider.WithRetry retries it at most
// once.
func AfterReply(err error) error {
	if err == nil {
		return nil
	}
	return &afterReply{err: err}
}

// IsAfterReply reports whether err carries AfterReply's mark.
func IsAfterReply(err error) bool {
	var mark *afterReply
	return errors.As(err, &mark)
}

// ReplyReader reads a reply's body under an idle limit and a size limit, and
// keeps the first read failure, so a caller can tell a cut connection from a
// body that arrived whole but did not decode (0021-MADR D1, D2).
type ReplyReader struct {
	r       io.Reader
	ctx     context.Context
	limit   int64
	n       int64
	idle    time.Duration
	timer   *time.Timer
	readErr error
}

// NewReplyReader reads body. ctx is the request's context, and cancel its
// cancel function: when idle passes with no data, the request is cancelled
// with ErrIdleTimeout. A limit of 0 means no size limit, and an idle of 0 no
// idle limit. Stop must be called when the reading is done.
func NewReplyReader(ctx context.Context, cancel context.CancelCauseFunc, body io.Reader, limit int64, idle time.Duration) *ReplyReader {
	rr := &ReplyReader{r: body, ctx: ctx, limit: limit, idle: idle}
	if idle > 0 {
		rr.timer = time.AfterFunc(idle, func() { cancel(ErrIdleTimeout) })
	}
	return rr
}

// Read reads from the body. Past the size limit it returns ErrReplyTooLarge.
func (rr *ReplyReader) Read(p []byte) (int, error) {
	if rr.limit > 0 && int64(len(p)) > rr.limit-rr.n+1 {
		p = p[:rr.limit-rr.n+1]
	}
	k, err := rr.r.Read(p)
	if k > 0 && rr.timer != nil {
		rr.timer.Reset(rr.idle)
	}
	rr.n += int64(k)
	if rr.limit > 0 && rr.n > rr.limit {
		k -= int(rr.n - rr.limit)
		rr.n = rr.limit
		return k, ErrReplyTooLarge
	}
	if err != nil && !errors.Is(err, io.EOF) {
		if errors.Is(context.Cause(rr.ctx), ErrIdleTimeout) {
			err = ErrIdleTimeout
		}
		if rr.readErr == nil {
			rr.readErr = err
		}
	}
	return k, err
}

// ReadErr is the first read failure other than io.EOF, or nil. A body cut by
// the idle limit reports ErrIdleTimeout.
func (rr *ReplyReader) ReadErr() error { return rr.readErr }

// Stop ends the idle limit.
func (rr *ReplyReader) Stop() {
	if rr.timer != nil {
		rr.timer.Stop()
	}
}
