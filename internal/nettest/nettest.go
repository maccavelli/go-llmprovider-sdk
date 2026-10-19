// Package nettest gives tests a connection that fails without dialling, and
// a loopback port that no other test can be given, so that no test depends
// on a freed port staying unused (0045-MADR). Only tests import it.
//
// go test runs packages in parallel, and the OS hands a freed ephemeral port
// to the next listener that asks. A test that dials a closed server's
// address, or binds a port it chose by binding :0 and closing it, can meet
// another package's server there.
package nettest

import (
	"context"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"sync"
	"syscall"
	"testing"
)

// RefusedURL is an address for a client from Refusing, which never dials
// it. Port 1 is privileged and outside every ephemeral range, so even a
// client that dials it meets no other test's server.
const RefusedURL = "http://127.0.0.1:1"

// Refusing returns a client whose every connection is refused at dial,
// without dialling. net/http wraps the refusal as it wraps one from the OS,
// so a caller sees a *url.Error around a *net.OpError that matches
// syscall.ECONNREFUSED, before any byte of the request is written (0045-MADR
// D1).
func Refusing() *http.Client {
	return &http.Client{Transport: &http.Transport{
		DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
			return nil, &net.OpError{Op: "dial", Net: network, Err: syscall.ECONNREFUSED}
		},
	}}
}

// The ports Port hands out: below the ephemeral ranges of Linux (32768 up)
// and of macOS and Windows (49152 up), so that no :0 bind is given one
// (0045-MADR D2).
const (
	portLow   = 20000
	portHigh  = 30000
	portTries = 100
)

var (
	portsMu    sync.Mutex
	portsGiven = map[int]bool{}
	// pickPort is a candidate port; a test replaces it.
	//nolint:gosec // G404: a test's port need not be unpredictable
	pickPort = func() int { return portLow + rand.N(portHigh-portLow) }
	ipv6Once sync.Once
	ipv6     bool
)

// Port returns a loopback port in [20000, 30000) that binds on 127.0.0.1,
// and on [::1] when the host has IPv6 loopback, for code under test to bind.
// It is released before Port returns, and never handed out again in this
// process, so parallel tests of one package do not share one. A port that
// does not bind, such as one Windows reserves, is passed over. Port fails t
// after 100 tries.
func Port(t testing.TB) int {
	t.Helper()
	portsMu.Lock()
	defer portsMu.Unlock()
	for range portTries {
		port := pickPort()
		if portsGiven[port] || !bindable(port) {
			continue
		}
		portsGiven[port] = true
		return port
	}
	t.Fatalf("nettest: no free loopback port in [%d, %d) after %d tries", portLow, portHigh, portTries)
	return 0
}

// bindable reports whether port binds on each loopback address at once, and
// is released cleanly.
func bindable(port int) bool {
	hosts := []string{"127.0.0.1"}
	if hasIPv6Loopback() {
		hosts = append(hosts, "::1")
	}
	var held []net.Listener
	ok := true
	for _, host := range hosts {
		l, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			ok = false
			break
		}
		held = append(held, l)
	}
	for _, l := range held {
		if err := l.Close(); err != nil {
			ok = false
		}
	}
	return ok
}

// hasIPv6Loopback reports whether [::1] can be bound, and released, on this
// host.
func hasIPv6Loopback() bool {
	ipv6Once.Do(func() {
		if l, err := net.Listen("tcp6", "[::1]:0"); err == nil {
			ipv6 = l.Close() == nil
		}
	})
	return ipv6
}
