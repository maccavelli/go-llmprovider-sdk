package auth

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestOAuthRefresh_RefusedCertificateOnce_0068 (0068-MADR D2): a refresh
// whose issuer's certificate the client refuses is tried once, not three
// times, and its error holds the verification error.
func TestOAuthRefresh_RefusedCertificateOnce_0068(t *testing.T) {
	var conns atomic.Int32
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		refreshOK(w, "a-new")
	}))
	srv.Config.ErrorLog = log.New(io.Discard, "", 0)
	srv.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			conns.Add(1)
		}
	}
	srv.StartTLS()
	t.Cleanup(srv.Close)

	session := &OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "a-old", Refresh: "rt-old",
		Expiry: time.Now().Add(-time.Minute), Issuer: DefaultOpenAIIssuer, ClientID: "client-test",
		TokenURL: srv.URL, HTTPClient: &http.Client{Transport: &http.Transport{}}}
	_, err := session.Token(context.Background())
	if _, ok := errors.AsType[*tls.CertificateVerificationError](err); !ok {
		t.Fatalf("err = %v; want a certificate verification error", err)
	}
	if n := conns.Load(); n != 1 {
		t.Errorf("%d connections; want 1: a refused certificate is not retried", n)
	}
}
