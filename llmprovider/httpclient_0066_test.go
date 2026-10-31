package llmprovider

import (
	"net/http"
	"testing"
	"time"
)

// 0066-MADR D2: NewHTTPClient is the bounded client a provider builds when
// given none, for a caller to wrap.

// baseOf is c's shared transport, behind the client's scoped view, as
// TestResolveOptions_DefaultTimeout reads it.
func baseOf(t *testing.T, c *http.Client) *http.Transport {
	t.Helper()
	base, ok := c.Transport.(interface{ Base() *http.Transport })
	if !ok {
		t.Fatalf("Transport %T has no Base", c.Transport)
	}
	return base.Base()
}

// TestNewHTTPClient_Bounds_0066: its transport is the very one a provider
// built with no client uses, with the documented bounds and no total
// timeout.
func TestNewHTTPClient_Bounds_0066(t *testing.T) {
	c := NewHTTPClient()
	st, err := ResolveOptions(ProviderOpenAI, nil)
	if err != nil {
		t.Fatal(err)
	}
	b := baseOf(t, c)
	if b != baseOf(t, st.HTTPClient()) {
		t.Error("NewHTTPClient's transport is not the one a provider builds with no client")
	}
	if b.ResponseHeaderTimeout != 300*time.Second || b.TLSHandshakeTimeout != 10*time.Second ||
		b.IdleConnTimeout != 90*time.Second || b.MaxIdleConnsPerHost != 16 || b.DialContext == nil || c.Timeout != 0 {
		t.Errorf("bounds: first byte %v, handshake %v, idle %v, per host %d, dialer set %v, total %v; "+
			"want 300s, 10s, 90s, 16, true, none", b.ResponseHeaderTimeout, b.TLSHandshakeTimeout, b.IdleConnTimeout,
			b.MaxIdleConnsPerHost, b.DialContext != nil, c.Timeout)
	}
}

// TestNewHTTPClient_NewEachCall_0066: two calls give two clients over one
// transport; a field set on one leaves the other as it was.
func TestNewHTTPClient_NewEachCall_0066(t *testing.T) {
	a, b := NewHTTPClient(), NewHTTPClient()
	if a == b || baseOf(t, a) != baseOf(t, b) {
		t.Fatalf("clients %p and %p; want two, over one transport", a, b)
	}
	transport := b.Transport
	a.Timeout, a.Transport = time.Second, nil
	if b.Timeout != 0 || b.Transport != transport {
		t.Errorf("b changed with a: Timeout %v, Transport %T", b.Timeout, b.Transport)
	}
}
