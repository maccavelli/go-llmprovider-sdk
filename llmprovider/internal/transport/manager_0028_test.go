package transport

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync/atomic"
	"testing"
)

// 0028-MADR D-A2: the default clients share one transport per
// configuration; one client's CloseIdleConnections closes nothing the others
// would reuse.

// tlsServer is a TLS test server, HTTP/2 or HTTP/1.1 only, counting the
// connections it accepts. The default configuration trusts its certificate
// for the test.
func tlsServer(t *testing.T, http2 bool) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var conns atomic.Int32
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	srv.EnableHTTP2 = http2
	srv.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			conns.Add(1)
		}
	}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	cfg := defaultConfig
	cfg.rootCAs = srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs
	previous := defaultConfig
	defaultConfig = cfg
	t.Cleanup(func() {
		defaultConfig = previous
		defaultManager.drop(cfg)
	})
	return srv, &conns
}

// call makes one request with c and reads the reply through.
func call(t *testing.T, c *http.Client, url string) {
	t.Helper()
	resp, err := c.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

// TestDefaultClient_SharesOneTransport: two default clients, one call each,
// open one connection between them, over HTTP/1.1 and over HTTP/2.
func TestDefaultClient_SharesOneTransport(t *testing.T) {
	for name, h2 := range map[string]bool{"HTTP/1.1": false, "HTTP/2": true} {
		t.Run(name, func(t *testing.T) {
			srv, conns := tlsServer(t, h2)
			call(t, DefaultClient(), srv.URL)
			call(t, DefaultClient(), srv.URL)
			if n := conns.Load(); n != 1 {
				t.Errorf("connections = %d; want 1, shared", n)
			}
		})
	}
}

// TestDefaultClient_CloseIdleConnectionsIsScoped: client A closing its idle
// connections does not close the one client B reuses.
func TestDefaultClient_CloseIdleConnectionsIsScoped(t *testing.T) {
	srv, conns := tlsServer(t, true)
	a, b := DefaultClient(), DefaultClient()
	call(t, a, srv.URL)
	a.CloseIdleConnections()
	call(t, b, srv.URL)
	if n := conns.Load(); n != 1 {
		t.Errorf("connections = %d; want 1, B reusing the connection A left idle", n)
	}
}

// TestManager_OneTransportPerConfig: the same configuration has one
// transport; a different one has its own.
func TestManager_OneTransportPerConfig(t *testing.T) {
	m := &manager{transports: map[Config]*http.Transport{}}
	other := defaultConfig
	other.MaxIdleConnsPerHost++
	first, second := m.transport(defaultConfig), m.transport(defaultConfig)
	if first != second {
		t.Error("one configuration, two transports")
	}
	if m.transport(defaultConfig) == m.transport(other) {
		t.Error("two configurations, one transport")
	}
	if tr := m.transport(other); tr.MaxIdleConnsPerHost != other.MaxIdleConnsPerHost ||
		tr.ResponseHeaderTimeout != other.ResponseHeaderTimeout {
		t.Errorf("transport %+v; want the configuration's settings", tr)
	}
}

// TestDefaultClients_GoroutinesBounded: 20 default clients, one call each,
// leave the goroutine count within the baseline and 8 (0020-MADR F28's
// measure), where 20 transports would hold 20 connections' loops.
func TestDefaultClients_GoroutinesBounded(t *testing.T) {
	srv, _ := tlsServer(t, true)
	baseline := runtime.NumGoroutine()
	for range 20 {
		call(t, DefaultClient(), srv.URL)
	}
	if n := runtime.NumGoroutine(); n > baseline+8 {
		t.Errorf("goroutines = %d; want at most %d, the baseline and 8", n, baseline+8)
	}
}

// TestDefaultClient_MaxIdleConnsPerHost: the shared transport keeps 16 idle
// connections per host (0028-MADR D-A2).
func TestDefaultClient_MaxIdleConnsPerHost(t *testing.T) {
	if n := DefaultClient().Transport.(interface{ Base() *http.Transport }).Base().MaxIdleConnsPerHost; n != 16 {
		t.Errorf("MaxIdleConnsPerHost = %d; want 16", n)
	}
}
