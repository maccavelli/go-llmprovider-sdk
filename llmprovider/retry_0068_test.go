package llmprovider

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// 0068-MADR D2: what no retry can mend is not retried: a TLS certificate the
// client refuses, and HTTP 501 and 505.

// untrustedServer is a TLS test server whose certificate the default client
// does not trust, counting the connections it accepts.
func untrustedServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var conns atomic.Int32
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	srv.Config.ErrorLog = log.New(io.Discard, "", 0)
	srv.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			conns.Add(1)
		}
	}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv, &conns
}

// TestWithRetry_RefusedCertificateOnce_0068: a provider whose request meets a
// certificate the client refuses is tried once under MaxAttempts 3, as it
// is when the failure is wrapped as ErrProviderUnavailable, as an OAuth
// refresh wraps it.
func TestWithRetry_RefusedCertificateOnce_0068(t *testing.T) {
	srv, conns := untrustedServer(t)
	send := func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		if err != nil {
			return err
		}
		resp, err := transport.DefaultClient().Do(req)
		if err == nil {
			_ = resp.Body.Close()
			return errors.New("the untrusted server was trusted")
		}
		return err
	}
	for _, c := range []struct {
		name string
		wrap func(error) error
	}{
		{"as sent", func(err error) error { return err }},
		{"wrapped as ErrProviderUnavailable", func(err error) error {
			return fmt.Errorf("%w: oauth: refresh request: %w", ErrProviderUnavailable, err)
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			before := conns.Load()
			var calls atomic.Int32
			p := &stubProvider{id: "stub", gen: func(ctx context.Context, _ *Request) (*Response, error) {
				calls.Add(1)
				return nil, c.wrap(send(ctx))
			}}
			_, err := WithRetry(p, fastRetry).Generate(context.Background(), &Request{})
			if _, ok := errors.AsType[*tls.CertificateVerificationError](err); !ok {
				t.Fatalf("err = %v; want a certificate verification error", err)
			}
			if n, h := calls.Load(), conns.Load()-before; n != 1 || h != 1 {
				t.Errorf("%d attempts, %d connections; want 1 and 1: a refused certificate is not retried", n, h)
			}
		})
	}
}

// TestClassify_501And505Terminal_0068: 501 and 505 are ErrProviderUnavailable
// and not Retryable; 500 still is.
func TestClassify_501And505Terminal_0068(t *testing.T) {
	for _, c := range []struct {
		status    int
		retryable bool
	}{
		{http.StatusNotImplemented, false},
		{http.StatusHTTPVersionNotSupported, false},
		{http.StatusInternalServerError, true},
	} {
		resp := &http.Response{StatusCode: c.status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}
		err := ClassifyHTTPError(string(ProviderOpenAI), resp)
		apiErr, ok := errors.AsType[*APIError](err)
		if !ok || !errors.Is(err, ErrProviderUnavailable) {
			t.Fatalf("%d: err = %v; want an *APIError matching ErrProviderUnavailable", c.status, err)
		}
		if apiErr.Retryable() != c.retryable {
			t.Errorf("%d: Retryable = %v; want %v", c.status, apiErr.Retryable(), c.retryable)
		}
	}
}
