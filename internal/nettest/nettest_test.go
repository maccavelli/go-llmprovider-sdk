package nettest

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"syscall"
	"testing"
)

// TestRefusing: the client's error is a refused dial, as the OS gives one,
// and it never reaches a server that is listening (0045-MADR D1).
func TestRefusing(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits.Add(1) }))
	defer srv.Close()
	for _, target := range []string{RefusedURL, srv.URL} {
		resp, err := Refusing().Get(target)
		if err == nil {
			_ = resp.Body.Close()
			t.Fatalf("%s: a response; want a refused dial", target)
		}
		var urlErr *url.Error
		var netErr net.Error
		var opErr *net.OpError
		if !errors.As(err, &urlErr) || !errors.As(err, &netErr) || !errors.As(err, &opErr) || opErr.Op != "dial" ||
			!errors.Is(err, syscall.ECONNREFUSED) {
			t.Errorf("%s: %#v; want a *url.Error around a dial *net.OpError matching ECONNREFUSED", target, err)
		}
	}
	if hits.Load() != 0 {
		t.Errorf("the listening server got %d request(s); want none", hits.Load())
	}
}

// TestPort: distinct ports in the range, each free to bind on 127.0.0.1
// (0045-MADR D2).
func TestPort(t *testing.T) {
	seen := map[int]bool{}
	for range 50 {
		port := Port(t)
		if port < portLow || port >= portHigh || seen[port] {
			t.Fatalf("port %d: want a port in [%d, %d) not seen before", port, portLow, portHigh)
		}
		seen[port] = true
		l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		if err != nil {
			t.Fatalf("port %d does not bind: %v", port, err)
		}
		_ = l.Close()
	}
}

// heldPort is a port in the range that a listener holds.
func heldPort(t *testing.T) (int, net.Listener) {
	t.Helper()
	for port := portLow; port < portHigh; port++ {
		if l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port))); err == nil {
			return port, l
		}
	}
	t.Fatal("no port in the range binds")
	return 0, nil
}

// usePicks makes pickPort return ports in turn, the last one again and
// again.
func usePicks(t *testing.T, ports ...int) {
	t.Helper()
	original := pickPort
	t.Cleanup(func() { pickPort = original })
	n := 0
	pickPort = func() int {
		port := ports[min(n, len(ports)-1)]
		n++
		return port
	}
}

// TestPort_PassesOverHeldAndGiven: a port that does not bind, or was handed
// out already, is passed over.
func TestPort_PassesOverHeldAndGiven(t *testing.T) {
	held, l := heldPort(t)
	defer l.Close()
	given := Port(t)
	free := held + 1
	for portsGiven[free] || !bindable(free) {
		free++
	}
	usePicks(t, held, given, free)
	if got := Port(t); got != free {
		t.Errorf("Port = %d; want %d, past the held %d and the given %d", got, free, held, given)
	}
}

// TestPort_PassesOverIPv6Held: a port held on [::1] only is passed over,
// where the host has IPv6 loopback.
func TestPort_PassesOverIPv6Held(t *testing.T) {
	if !hasIPv6Loopback() {
		t.Skip("no IPv6 loopback on this host")
	}
	held := portLow
	var l net.Listener
	for ; held < portHigh; held++ {
		var err error
		if l, err = net.Listen("tcp6", net.JoinHostPort("::1", strconv.Itoa(held))); err == nil {
			break
		}
	}
	if l == nil {
		t.Fatal("no port in the range binds on [::1]")
	}
	defer l.Close()
	free := held + 1
	for portsGiven[free] || !bindable(free) {
		free++
	}
	usePicks(t, held, free)
	if got := Port(t); got != free {
		t.Errorf("Port = %d; want %d, past %d, held on [::1]", got, free, held)
	}
}

// failRecorder is a testing.TB that records Fatalf.
type failRecorder struct {
	testing.TB
	failed string
}

func (f *failRecorder) Helper() {}

func (f *failRecorder) Fatalf(format string, args ...any) { f.failed = fmt.Sprintf(format, args...) }

// TestPort_FailsAfterTries: with no free port, Port fails its test.
func TestPort_FailsAfterTries(t *testing.T) {
	held, l := heldPort(t)
	defer l.Close()
	usePicks(t, held)
	rec := &failRecorder{TB: t}
	if got := Port(rec); got != 0 || rec.failed == "" {
		t.Errorf("Port = %d, failure %q; want 0 and a failure", got, rec.failed)
	}
}
