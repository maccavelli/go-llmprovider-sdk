package transport

import (
	"net/http"
	"testing"
	"time"
)

// Ported from llmprovider's timeouts_test.go (0015-PLAN S7b).

// TestDefaultClient_Timeouts pins MADR 0012 §1.3: the references allow a
// generation 300 s to first byte, and a slow model must not be cut off at 30 s
// (0013 D6).
func TestDefaultClient_Timeouts(t *testing.T) {
	client := DefaultClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport is %T, want *http.Transport", client.Transport)
	}
	if transport.ResponseHeaderTimeout != 300*time.Second || client.Timeout != 330*time.Second {
		t.Fatalf("ResponseHeaderTimeout/Timeout = %s/%s, want 5m0s/5m30s",
			transport.ResponseHeaderTimeout, client.Timeout)
	}
	if transport.Proxy == nil {
		t.Error("Proxy is nil, want http.ProxyFromEnvironment (0016-MADR D8)")
	}
}
