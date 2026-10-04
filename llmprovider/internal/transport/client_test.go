package transport

import (
	"net/http"
	"testing"
	"time"
)

// Ported from llmprovider's timeouts_test.go (0015-PLAN S7b).

// TestDefaultClient_Timeouts pins 0021-MADR D2, amending MADR 0012 §1.3: a
// generation may take 300 s to its first byte, as the references allow, and a
// slow model must not be cut off at 30 s (0013 D6); there is no total timeout,
// since the idle limit bounds a body; a connection is made within 30 s, and
// HTTP/2 is kept (T12).
func TestDefaultClient_Timeouts(t *testing.T) {
	client := DefaultClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport is %T, want *http.Transport", client.Transport)
	}
	if transport.ResponseHeaderTimeout != 300*time.Second || client.Timeout != 0 {
		t.Fatalf("ResponseHeaderTimeout/Timeout = %s/%s, want 5m0s/0s",
			transport.ResponseHeaderTimeout, client.Timeout)
	}
	if transport.DialContext == nil || !transport.ForceAttemptHTTP2 {
		t.Errorf("DialContext set %v, ForceAttemptHTTP2 %v; want a bounded dialer and HTTP/2 kept",
			transport.DialContext != nil, transport.ForceAttemptHTTP2)
	}
	if dialer.Timeout != 30*time.Second || dialer.KeepAlive != 30*time.Second {
		t.Errorf("dialer Timeout/KeepAlive = %s/%s, want 30s/30s", dialer.Timeout, dialer.KeepAlive)
	}
	if transport.Proxy == nil {
		t.Error("Proxy is nil, want http.ProxyFromEnvironment (0016-MADR D8)")
	}
}
