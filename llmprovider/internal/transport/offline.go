package transport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"runtime"
	"slices"
	"syscall"
)

// Offline reports whether err is a request that could not be sent for want
// of a network: a *url.Error whose host's name did not resolve, or whose dial
// timed out or found the port refused, the network down or unreachable, or
// the host unreachable. A TLS failure, a proxy's failure and any HTTP
// response are not. It is gobble-cli's rule (0061-MADR D2), with the
// standard library's names for those errors on Windows beside Winsock's.
func Offline(err error) bool {
	if _, ok := errors.AsType[*url.Error](err); !ok { //nolint:errcheck // only whether the chain holds one matters
		return false
	}
	if _, ok := errors.AsType[*net.DNSError](err); ok { //nolint:errcheck // only whether the chain holds one matters
		return true
	}
	op, ok := errors.AsType[*net.OpError](err)
	if !ok || op.Op != "dial" {
		return false
	}
	if op.Timeout() {
		return true
	}
	return slices.ContainsFunc(unreachable(), func(e syscall.Errno) bool { return errors.Is(err, e) })
}

// unreachable are the dial errors Offline counts: refused, network down or
// unreachable, host unreachable. Windows reports them as Winsock's codes; the
// syscall package's names for them, which it gives an error built from them,
// are counted there too.
func unreachable() []syscall.Errno {
	named := []syscall.Errno{syscall.ECONNREFUSED, syscall.ENETDOWN, syscall.ENETUNREACH, syscall.EHOSTUNREACH}
	if runtime.GOOS == "windows" {
		const wsaENETDOWN, wsaENETUNREACH, wsaECONNREFUSED, wsaEHOSTUNREACH = 10050, 10051, 10061, 10065
		return append(named, wsaECONNREFUSED, wsaENETDOWN, wsaENETUNREACH, wsaEHOSTUNREACH)
	}
	return named
}

// WrapOffline returns err wrapped in offline when ctx has not ended and err
// is Offline, else err as it is. A dial the caller's context ended is not
// offline: it keeps the context's error. offline is the caller's sentinel,
// llmprovider.ErrOffline, which this package cannot import.
func WrapOffline(ctx context.Context, err, offline error) error {
	if err == nil || ctx.Err() != nil || !Offline(err) {
		return err
	}
	return fmt.Errorf("%w: %w", offline, err)
}
