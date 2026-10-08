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
	body, err := json.Marshal(c.Body)
	if err != nil {
		// The body is built from the caller's request, which no retry
		// changes (0026-MADR F20).
		return zero, fmt.Errorf("%w: %s: marshal request: %w", llmprovider.ErrInvalidRequest, c.Provider, err)
	}
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
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
		return zero, fmt.Errorf("%w: %s: %w", llmprovider.ErrInvalidRequest, c.Provider, err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.Prepare(req, token)

	resp, err := c.Client.Do(req)
	if err != nil {
		// A request no retry can send, such as a token with a newline, is an
		// invalid request; a failure to reach the service keeps its own
		// error, which WithRetry retries (0026-MADR F11). A failure after the
		// request was written, such as a timeout awaiting the reply's
		// headers, may come after a billed generation: it is marked so that
		// WithRetry resends it at most once (0028-MADR D-H1).
		if transport.Unsendable(err) {
			return zero, fmt.Errorf("%w: %s: %w", llmprovider.ErrInvalidRequest, c.Provider, err)
		}
		if wrote.Load() {
			return zero, transport.AfterReply(err)
		}
		return zero, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.Logger.Debug("llmprovider: "+c.Provider+": close response body", "error", err)
		}
	}()
	var limit int64 = ReplyLimit
	if c.Stream {
		limit = 0
	}
	// The idle limit covers an error reply's body too: without a total
	// timeout on the client, a body that stalls after its headers would
	// otherwise hold the call until the caller's deadline (0026-MADR F4).
	reader := transport.NewReplyReader(ctx, cancel, resp.Body, limit, idleTimeout)
	defer reader.Stop()
	resp.Body = struct {
		io.Reader
		io.Closer
	}{reader, resp.Body}
	if err := llmprovider.ClassifyHTTPError(c.Provider, resp); err != nil {
		return zero, err
	}
	out, err := decode(reader)
	return out, DecodeError(c.Provider, err, reader.ReadErr())
}
