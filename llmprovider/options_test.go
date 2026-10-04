package llmprovider

import (
	"net/http"
	"testing"
	"time"
)

// TestResolveOptions_DefaultTimeout is the #5 regression: the default client
// must never wait forever (the old http.DefaultClient had no bound). MADR 0012
// §1.3 set a 330 s total; 0021-MADR D2 bounds each phase instead: the
// connection, the first byte at 300 s, and the body by wire.Post's idle limit.
func TestResolveOptions_DefaultTimeout(t *testing.T) {
	st, err := ResolveOptions(ProviderOpenAI, nil)
	if err != nil {
		t.Fatal(err)
	}
	if st.HTTPClient() == nil {
		t.Fatal("default HTTPClient is nil")
	}
	transport, ok := st.HTTPClient().Transport.(*http.Transport)
	if !ok || transport.ResponseHeaderTimeout != 300*time.Second || transport.DialContext == nil {
		t.Errorf("default transport %T: want a 300s first-byte bound and a bounded dialer", st.HTTPClient().Transport)
	}
	if st.HTTPClient().Timeout != 0 {
		t.Errorf("default timeout: got %v want none; the idle limit bounds a body", st.HTTPClient().Timeout)
	}
}

func TestResolveOptions_WithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 5 * time.Second}
	st, err := ResolveOptions(ProviderOpenAI, []Option{WithHTTPClient(custom)})
	if err != nil || st.HTTPClient() != custom {
		t.Errorf("WithHTTPClient override not honored (err %v)", err)
	}
}
