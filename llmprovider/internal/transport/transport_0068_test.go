package transport

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 0068-MADR D1: the shared transport pings a silent HTTP/2 connection, so a
// dead pooled one fails its request within the ping timeouts, not at the
// header timeout.

// TestDefaultClient_HTTP2HealthCheck_0068: DefaultClient's transport pings
// after 30 s of silence and closes the connection after 15 s more.
func TestDefaultClient_HTTP2HealthCheck_0068(t *testing.T) {
	base := DefaultClient().Transport.(interface{ Base() *http.Transport }).Base()
	if base.HTTP2 == nil {
		t.Fatal("HTTP2 is nil; want a health check")
	}
	if base.HTTP2.SendPingTimeout != 30*time.Second || base.HTTP2.PingTimeout != 15*time.Second {
		t.Errorf("SendPingTimeout, PingTimeout = %v, %v; want 30s, 15s", base.HTTP2.SendPingTimeout, base.HTTP2.PingTimeout)
	}
}

// silencer is a listener whose connections, once silenced, stop reading and
// drop what they write: a peer that has slept or vanished.
type silencer struct {
	net.Listener
	silent atomic.Bool
	mu     sync.Mutex
	conns  []*silentConn
}

type silentConn struct {
	net.Conn
	s    *silencer
	once sync.Once
	stop chan struct{}
}

func (l *silencer) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	sc := &silentConn{Conn: c, s: l, stop: make(chan struct{})}
	l.mu.Lock()
	l.conns = append(l.conns, sc)
	l.mu.Unlock()
	return sc, nil
}

// release ends every blocked read, for the test's cleanup.
func (l *silencer) release() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, c := range l.conns {
		c.once.Do(func() { close(c.stop) })
	}
}

func (c *silentConn) Read(p []byte) (int, error) {
	if c.s.silent.Load() {
		<-c.stop
		return 0, net.ErrClosed
	}
	return c.Conn.Read(p)
}

func (c *silentConn) Write(p []byte) (int, error) {
	if c.s.silent.Load() {
		return len(p), nil
	}
	return c.Conn.Write(p)
}

func (c *silentConn) Close() error {
	c.once.Do(func() { close(c.stop) })
	return c.Conn.Close()
}

// TestDeadHTTP2Connection_FailsWithinPing_0068: a pooled HTTP/2 connection
// whose peer goes silent fails the next request within the health check's
// timeouts. The transport is defaultConfig's, made by its own manager, with
// a 5 s header timeout; the health check, when the manager set one, is
// shortened to 500 ms and 500 ms. Without one, the request waits for the
// header timeout.
func TestDeadHTTP2Connection_FailsWithinPing_0068(t *testing.T) {
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	srv.EnableHTTP2 = true
	l := &silencer{Listener: srv.Listener}
	srv.Listener = l
	srv.StartTLS()
	t.Cleanup(func() {
		l.release()
		srv.Close()
	})

	cfg := defaultConfig
	cfg.rootCAs = srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs
	cfg.ResponseHeaderTimeout = 5 * time.Second
	m := &manager{transports: map[Config]*http.Transport{}}
	tr := m.transport(cfg)
	t.Cleanup(tr.CloseIdleConnections)
	if tr.HTTP2 != nil {
		tr.HTTP2.SendPingTimeout, tr.HTTP2.PingTimeout = 500*time.Millisecond, 500*time.Millisecond
	}
	client := &http.Client{Transport: scoped{base: tr}}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.ProtoMajor != 2 {
		t.Fatalf("first request over %s; want HTTP/2", resp.Proto)
	}

	l.silent.Store(true)
	start := time.Now()
	resp, err = client.Get(srv.URL)
	elapsed := time.Since(start)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("second request succeeded over a silent connection")
	}
	if elapsed >= 3*time.Second {
		t.Errorf("second request failed after %v (%v); want within the 500 ms + 500 ms health check, "+
			"not at the 5 s header timeout", elapsed.Round(100*time.Millisecond), err)
	}
}
