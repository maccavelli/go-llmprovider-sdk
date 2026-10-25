package transport

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os"
	"runtime"
	"syscall"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
)

// timedOut is a dial cause that timed out.
type timedOut struct{}

func (timedOut) Error() string   { return "i/o timeout" }
func (timedOut) Timeout() bool   { return true }
func (timedOut) Temporary() bool { return true }

// sent wraps cause as the HTTP client returns a failure to send.
func sent(cause error) error {
	return &url.Error{Op: "Post", URL: "https://api.example.test/v1", Err: cause}
}

// dial is a failed dial with cause.
func dial(cause error) error { return &net.OpError{Op: "dial", Net: "tcp", Err: cause} }

// TestOffline_Rule_0061 (0061-MADR D2): what Offline counts, and what it does
// not.
func TestOffline_Rule_0061(t *testing.T) {
	_, refused := nettest.Refusing().Get(nettest.RefusedURL) //nolint:bodyclose,noctx // no response: the dial is refused
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"a refused dial, as the OS gives it", refused, true},
		{"a name that did not resolve", sent(dial(&net.DNSError{Err: "no such host", Name: "api.example.test", IsNotFound: true})), true},
		{"a dial that timed out", sent(dial(timedOut{})), true},
		{"network down", sent(dial(&os.SyscallError{Syscall: "connect", Err: syscall.ENETDOWN})), true},
		{"network unreachable", sent(dial(&os.SyscallError{Syscall: "connect", Err: syscall.ENETUNREACH})), true},
		{"host unreachable", sent(dial(&os.SyscallError{Syscall: "connect", Err: syscall.EHOSTUNREACH})), true},
		{"a plain error", errors.New("boom"), false},
		{"a TLS failure", sent(errors.New("tls: handshake failure")), false},
		{"a proxy's failure", sent(&net.OpError{Op: "proxyconnect", Net: "tcp", Err: syscall.ECONNREFUSED}), false},
		{"a reset read", sent(&net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET}), false},
		{"a dial with another cause", sent(dial(errors.New("permission denied"))), false},
		{"a refused dial outside a *url.Error", dial(syscall.ECONNREFUSED), false},
	}
	if runtime.GOOS == "windows" {
		cases = append(cases, struct {
			name string
			err  error
			want bool
		}{"Winsock's refused", sent(dial(&os.SyscallError{Syscall: "connectex", Err: syscall.Errno(10061)})), true})
	}
	if refused == nil {
		t.Fatal("nettest.Refusing gave no error")
	}
	for _, c := range cases {
		if got := Offline(c.err); got != c.want {
			t.Errorf("%s: Offline = %t; want %t (%v)", c.name, got, c.want, c.err)
		}
	}
}

// TestWrapOffline_Context_0061 (0061-MADR D2): a failure with no network is
// wrapped while the context is live, and not once it has ended.
func TestWrapOffline_Context_0061(t *testing.T) {
	offline := errors.New("offline")
	_, refused := nettest.Refusing().Get(nettest.RefusedURL) //nolint:bodyclose,noctx // no response: the dial is refused
	if err := WrapOffline(context.Background(), refused, offline); !errors.Is(err, offline) || !errors.Is(err, syscall.ECONNREFUSED) {
		t.Errorf("live context: %v; want it to match the sentinel and keep the cause", err)
	}
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	if err := WrapOffline(ended, refused, offline); errors.Is(err, offline) || err != refused { //nolint:errorlint // the same error, unwrapped
		t.Errorf("ended context: %v; want the error as it was", err)
	}
	if err := WrapOffline(context.Background(), nil, offline); err != nil {
		t.Errorf("nil: %v; want nil", err)
	}
	plain := errors.New("boom")
	if err := WrapOffline(context.Background(), plain, offline); err != plain { //nolint:errorlint // the same error, unwrapped
		t.Errorf("not offline: %v; want it as it was", err)
	}
}
