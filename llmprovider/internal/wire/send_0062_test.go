package wire_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/responses"
)

// TestSend_Bound_0062 (0062-MADR D5, Q5 (a)): a check that always says resend
// is asked after each refusal until MaxSends sends are made, and the last
// refusal is returned; a forced-choice check's resend counts against the
// same bound; a stream's opening is bound the same way.
func TestSend_Bound_0062(t *testing.T) {
	// The owner's answer to Q5: at most 7 sends.
	const bound = 7
	if wire.MaxSends != bound {
		t.Fatalf("MaxSends = %d; want %d (0062-MADR Q5 (a))", wire.MaxSends, bound)
	}
	always := make([]int, 2*bound)
	for i := range always {
		always[i] = http.StatusBadRequest
	}
	want := strings.TrimSuffix(strings.Repeat("@t,", wire.MaxSends), ",")

	srv, seen := retryServer(t, replies("ok", always...))
	build, builds := builder(srv.URL, false)
	chk := &check{resend: true}
	_, err := wire.Send(context.Background(), "p", llmprovider.NewStaticToken("t"), build, readBody, chk.retry)
	if *builds != wire.MaxSends || len(chk.asked) != wire.MaxSends-1 || !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("always resend: %d builds, check asked %d times, %v; want %d, %d, the last refusal", *builds,
			len(chk.asked), err, wire.MaxSends, wire.MaxSends-1)
	}
	if got := stripBuilds(seen()); got != want {
		t.Errorf("requests %s; want %d", seen(), wire.MaxSends)
	}

	// A forced-choice check that resends once, then an effort check that
	// always does: one bound for both.
	srv, _ = retryServer(t, replies("ok", always...))
	build, builds = builder(srv.URL, false)
	forced, effort := 0, 0
	retry := wire.Retries(func(err error) (bool, error) {
		forced++
		return forced == 1, err
	}, func(err error) (bool, error) {
		effort++
		return true, nil
	})
	if _, err := wire.Send(context.Background(), "p", llmprovider.NewStaticToken("t"), build, readBody, retry); err == nil ||
		*builds != wire.MaxSends || forced != wire.MaxSends-1 || effort != wire.MaxSends-2 {
		t.Errorf("forced then effort: %d builds, forced asked %d, effort asked %d, %v; want %d, %d, %d and the refusal",
			*builds, forced, effort, err, wire.MaxSends, wire.MaxSends-1, wire.MaxSends-2)
	}

	srv, _ = retryServer(t, replies("ok", always...))
	build, builds = builder(srv.URL, true)
	chk = &check{resend: true}
	var streamErr error
	for _, err := range wire.StreamEventsRetry(context.Background(), "p", llmprovider.NewStaticToken("t"), build,
		responses.Events, chk.retry) {
		if err != nil {
			streamErr = err
		}
	}
	if *builds != wire.MaxSends || len(chk.asked) != wire.MaxSends-1 || !errors.Is(streamErr, llmprovider.ErrInvalidRequest) {
		t.Errorf("a stream: %d opens, check asked %d times, %v; want %d, %d, the last refusal", *builds, len(chk.asked),
			streamErr, wire.MaxSends, wire.MaxSends-1)
	}
}

// stripBuilds is a request log with each build number dropped: "@t,@t".
func stripBuilds(log string) string {
	parts := strings.Split(log, ",")
	for i, p := range parts {
		if at := strings.Index(p, "@"); at >= 0 {
			parts[i] = p[at:]
		}
	}
	return strings.Join(parts, ",")
}
