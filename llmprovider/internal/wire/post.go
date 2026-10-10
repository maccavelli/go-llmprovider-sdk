package wire

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptrace"
	"sync/atomic"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// idleTimeout is the idle limit on a reply's body (0021-MADR D2).
var idleTimeout = transport.StreamIdleTimeout

// SetIdleTimeout sets the idle limit on a reply's body and returns a function
// that restores it. It is for tests, which cannot wait 300 s.
func SetIdleTimeout(d time.Duration) (restore func()) {
	old := idleTimeout
	idleTimeout = d
	return func() { idleTimeout = old }
}

// Call is one generation request: where it goes, what it carries, and how it
// is labelled in errors.
type Call struct {
	// Provider labels errors, as ClassifyHTTPError and DecodeError take it.
	Provider string
	Client   *http.Client
	Logger   *slog.Logger
	URL      string
	// Body is marshalled with json.Marshal.
	Body any
	// Prepare sets the provider's headers and applies the token.
	Prepare func(*http.Request, llmprovider.Token)
	// Stream reads an event stream: no limit on the whole body, since the
	// decoder bounds each event.
	Stream bool
}

// Post sends c once with token, and decodes a 2xx reply with decode (0021-MADR
// W11). The reply is read under the idle limit and, unless c.Stream, under
// ReplyLimit; a non-2xx reply is classified by llmprovider.ClassifyHTTPError,
// and a failure to read the reply by DecodeError.
func Post[T any](ctx context.Context, c Call, token llmprovider.Token, decode func(io.Reader) (T, error)) (T, error) {
	var zero T
	var limit int64 = ReplyLimit
	if c.Stream {
		limit = 0
	}
	reply, err := open(ctx, c, token, limit)
	if err != nil {
		return zero, err
	}
	defer reply.Close()
	out, err := decode(reply.reader)
	return out, DecodeError(c.Provider, err, reply.reader.ReadErr())
}

// Stream is a 2xx reply held open while its events are read (0031-MADR D3).
type Stream struct {
	provider string
	logger   *slog.Logger
	body     io.Closer
	reader   *transport.ReplyReader
	cancel   context.CancelCauseFunc
	closed   bool
}

// OpenStream sends c once with token, as Post does, and returns the 2xx reply
// open, under the idle limit and with no limit on the whole body: the decoder
// bounds each event (0031-MADR D3). A non-2xx reply is classified by
// llmprovider.ClassifyHTTPError and closed, before any event, so Reauth can
// renew a refused credential and send again. The caller reads the reply with
// Read, or ends it with Close.
func OpenStream(ctx context.Context, c Call, token llmprovider.Token) (*Stream, error) {
	return open(ctx, c, token, 0)
}

// Read runs decode over the reply's body, then closes it, whether decode
// finished, failed or stopped early. A failure to read the body gets its kind
// from DecodeError, as Post's does.
func (s *Stream) Read(decode func(io.Reader) error) error {
	defer s.Close()
	return DecodeError(s.provider, decode(s.reader), s.reader.ReadErr())
}

// Close ends the reply: it stops the idle limit, closes the body and cancels
// the request. It may be called more than once.
func (s *Stream) Close() {
	if s.closed {
		return
	}
	s.closed = true
	s.reader.Stop()
	if err := s.body.Close(); err != nil {
		s.logger.Debug("llmprovider: "+s.provider+": close response body", "error", err)
	}
	s.cancel(nil)
}

// open sends c once with token and returns its 2xx reply, read under the idle
// limit and, when limit is above 0, under limit bytes.
func open(ctx context.Context, c Call, token llmprovider.Token, limit int64) (*Stream, error) {
	body, err := json.Marshal(c.Body)
	if err != nil {
		// The body is built from the caller's request, which no retry
		// changes (0026-MADR F20).
		return nil, fmt.Errorf("%w: %s: marshal request: %w", llmprovider.ErrInvalidRequest, c.Provider, err)
	}
	ctx, cancel := context.WithCancelCause(ctx)
	// wrote records that the whole request reached the connection, so a
	// failure after it is not a failure to send (0028-MADR D-H1).
	var wrote atomic.Bool
	ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			if info.Err == nil {
				wrote.Store(true)
			}
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		cancel(nil)
		return nil, fmt.Errorf("%w: %s: %w", llmprovider.ErrInvalidRequest, c.Provider, err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.Prepare(req, token)

	//nolint:bodyclose // the returned Stream owns the body; Read and Close close it (TestStream_StopClosesBody; 0031-PLAN D3)
	resp, err := c.Client.Do(req)
	if err != nil {
		cancel(nil)
		// A request no retry can send, such as a token with a newline, is an
		// invalid request; a failure to reach the service keeps its own
		// error, which WithRetry retries (0026-MADR F11). A failure after the
		// request was written, such as a timeout awaiting the reply's
		// headers, may come after a billed generation: it is marked so that
		// WithRetry resends it at most once (0028-MADR D-H1).
		if transport.Unsendable(err) {
			return nil, fmt.Errorf("%w: %s: %w", llmprovider.ErrInvalidRequest, c.Provider, err)
		}
		if wrote.Load() {
			return nil, transport.AfterReply(err)
		}
		return nil, err
	}
	// The idle limit covers an error reply's body too: without a total
	// timeout on the client, a body that stalls after its headers would
	// otherwise hold the call until the caller's deadline (0026-MADR F4).
	reply := &Stream{provider: c.Provider, logger: c.Logger, body: resp.Body, cancel: cancel,
		reader: transport.NewReplyReader(ctx, cancel, resp.Body, limit, idleTimeout)}
	resp.Body = struct {
		io.Reader
		io.Closer
	}{reply.reader, reply.body}
	if err := llmprovider.ClassifyHTTPError(c.Provider, resp); err != nil {
		reply.Close()
		return nil, err
	}
	return reply, nil
}
