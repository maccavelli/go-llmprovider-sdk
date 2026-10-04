package wire

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
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
		return zero, fmt.Errorf("llmprovider: %s: marshal request: %w", c.Provider, err)
	}
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return zero, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.Prepare(req, token)

	resp, err := c.Client.Do(req)
	if err != nil {
		return zero, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.Logger.Debug("llmprovider: "+c.Provider+": close response body", "error", err)
		}
	}()
	if err := llmprovider.ClassifyHTTPError(c.Provider, resp); err != nil {
		return zero, err
	}
	var limit int64 = ReplyLimit
	if c.Stream {
		limit = 0
	}
	reader := transport.NewReplyReader(ctx, cancel, resp.Body, limit, idleTimeout)
	defer reader.Stop()
	out, err := decode(reader)
	return out, DecodeError(c.Provider, err, reader.ReadErr())
}
