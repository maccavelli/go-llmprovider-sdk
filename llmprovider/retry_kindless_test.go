package llmprovider

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
)

// TestWithRetry_KindlessRetriedOnlyFromTheNetwork (0020-MADR F9, Q2 a): an
// error without a kind is retried only when it comes from the network; any
// other, such as an answer that could not be read, was billed already and is
// not bought again.
func TestWithRetry_KindlessRetriedOnlyFromTheNetwork(t *testing.T) {
	for _, c := range []struct {
		name  string
		err   error
		calls int32
	}{
		{"a network read failure", &net.OpError{Op: "read", Err: errors.New("connection reset by peer")}, 2},
		{"an unreadable answer", io.ErrUnexpectedEOF, 1},
		{"any other kindless error", errors.New("chat completions: the answer has no choices"), 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			p, calls := failing(c.err)
			_, _ = WithRetry(p, fastRetry).Generate(context.Background(), &Request{})
			if got := calls.Load(); got != c.calls {
				t.Errorf("%d calls, want %d", got, c.calls)
			}
		})
	}
}
