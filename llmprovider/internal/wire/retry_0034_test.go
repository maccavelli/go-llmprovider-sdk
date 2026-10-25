package wire_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/responses"
)

// errReplaced is what a check returns in a refusal's place.
var errReplaced = errors.New("replaced by the check")

// retryServer answers request n (from 1), sent with auth, with reply(n, auth),
// and records each as "<build>@<token>", the build read from the body's n.
func retryServer(t *testing.T, reply func(n int, auth string) (int, string)) (*httptest.Server, func() string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		mu.Lock()
		seen = append(seen, fmt.Sprintf("%s@%s", strings.Trim(string(raw), "{}\"n:"), auth))
		n := len(seen)
		mu.Unlock()
		status, body := reply(n, auth)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, func() string {
		mu.Lock()
		defer mu.Unlock()
		return strings.Join(seen, ",")
	}
}

// builder builds a call to url whose body counts the builds.
func builder(url string, stream bool) (func(llmprovider.Token) wire.Call, *int) {
	builds := 0
	return func(llmprovider.Token) wire.Call {
		builds++
		c := streamCall(url)
		c.Stream = stream
		c.Body = map[string]any{"n": builds}
		return c
	}, &builds
}

// check is a retry check that answers resend, and returns errReplaced
// wrapping the refusal when it does not; it records what it was asked.
type check struct {
	resend bool
	asked  []error
}

func (c *check) retry(err error) (bool, error) {
	c.asked = append(c.asked, err)
	if c.resend {
		return true, nil
	}
	return false, fmt.Errorf("%w: %w", errReplaced, err)
}

// replies answers the statuses in turn, 200 with ok past them.
func replies(ok string, statuses ...int) func(int, string) (int, string) {
	return func(n int, _ string) (int, string) {
		if n <= len(statuses) && statuses[n-1] != http.StatusOK {
			return statuses[n-1], `{"error":{"message":"refused"}}`
		}
		return http.StatusOK, ok
	}
}

// readBody is the reply body as a string.
func readBody(r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	return string(b), err
}

// TestSend_Retry_0034 (0034-MADR D3; 0034-PLAN Phase 4f): a refused send the
// check answers true is built again and sent once more; a check answering
// false has its error returned after one request; a refused resend is put to
// the check again, until it declines or MaxSends is met (0062-MADR D5, Q5
// (a); TestSend_Bound_0062); success never consults the check.
func TestSend_Retry_0034(t *testing.T) {
	for _, c := range []struct {
		name     string
		statuses []int
		resend   bool
		seen     string
		asked    int
		err      error
	}{
		{"refused, rebuilt, answered", []int{400}, true, "1@t,2@t", 1, nil},
		{"the check says no", []int{400}, false, "1@t", 1, errReplaced},
		{"the resend refused, and asked again", []int{400, 400}, true, "1@t,2@t,3@t", 2, nil},
		{"success", nil, true, "1@t", 0, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			srv, seen := retryServer(t, replies("ok", c.statuses...))
			build, _ := builder(srv.URL, false)
			chk := &check{resend: c.resend}
			out, err := wire.Send(context.Background(), "p", llmprovider.NewStaticToken("t"), build, readBody, chk.retry)
			if seen() != c.seen || len(chk.asked) != c.asked || !errors.Is(err, c.err) || (c.err == nil && out != "ok") {
				t.Errorf("requests %s, check asked %d times, %q, %v; want %s, %d, %v", seen(), len(chk.asked), out, err, c.seen, c.asked, c.err)
			}
			if errors.Is(c.err, errReplaced) && !errors.As(err, new(*llmprovider.APIError)) {
				t.Errorf("the check's error %v lost the refusal's *APIError", err)
			}
		})
	}
	srv, seen := retryServer(t, replies("ok", 400))
	build, _ := builder(srv.URL, false)
	if _, err := wire.Send(context.Background(), "p", llmprovider.NewStaticToken("t"), build, readBody, nil); err == nil || seen() != "1@t" {
		t.Errorf("no check: requests %s, err %v; want one request and the refusal", seen(), err)
	}
}

// TestSend_ReauthInsideEachSend_0034 (0020-MADR F2; 0034-PLAN Phase 4f): a 401
// is renewed inside the send, and the check sees only the refusal after it.
func TestSend_ReauthInsideEachSend_0034(t *testing.T) {
	srv, seen := retryServer(t, func(n int, auth string) (int, string) {
		switch {
		case auth != "fresh":
			return http.StatusUnauthorized, `{"error":{"message":"expired"}}`
		case n == 2:
			return http.StatusBadRequest, `{"error":{"message":"refused"}}`
		}
		return http.StatusOK, "ok"
	})
	build, _ := builder(srv.URL, false)
	chk := &check{resend: true}
	out, err := wire.Send(context.Background(), "p", &renewingSource{}, build, readBody, chk.retry)
	if err != nil || out != "ok" || seen() != "1@stale,2@fresh,3@fresh" {
		t.Fatalf("requests %s, %q, %v; want stale, fresh refused, fresh answered", seen(), out, err)
	}
	if apiErr, ok := errors.AsType[*llmprovider.APIError](chk.asked[0]); len(chk.asked) != 1 || !ok || apiErr.Status != http.StatusBadRequest {
		t.Errorf("the check was asked %v; want only the 400", chk.asked)
	}
}

// TestStreamEventsRetry_0034 (0034-MADR D3; 0034-PLAN Phase 4f): a stream
// refused when opened is built again and opened once more; a failure after
// an event is never resent, nor put to the check.
func TestStreamEventsRetry_0034(t *testing.T) {
	stream := sseEvent(textDelta) + sseEvent(itemDone) + sseEvent(completed)
	collect := func(seq func(func(llmprovider.Event, error) bool)) ([]llmprovider.EventType, error) {
		var types []llmprovider.EventType
		for ev, err := range seq {
			if err != nil {
				return types, err
			}
			types = append(types, ev.Type)
		}
		return types, nil
	}

	srv, seen := retryServer(t, replies(stream, 400))
	build, builds := builder(srv.URL, true)
	chk := &check{resend: true}
	types, err := collect(wire.StreamEventsRetry(context.Background(), "p", llmprovider.NewStaticToken("t"), build, responses.Events, chk.retry))
	if err != nil || len(types) != 3 || seen() != "1@t,2@t" || *builds != 2 {
		t.Errorf("refused at open: events %v, %v, requests %s, %d builds; want 3 events after a rebuilt resend", types, err, seen(), *builds)
	}

	srv, seen = retryServer(t, replies(stream, 400))
	build, _ = builder(srv.URL, true)
	chk = &check{}
	if _, err := collect(wire.StreamEventsRetry(context.Background(), "p", llmprovider.NewStaticToken("t"), build, responses.Events, chk.retry)); !errors.Is(err, errReplaced) || seen() != "1@t" {
		t.Errorf("the check says no: %v, requests %s; want its error after one request", err, seen())
	}

	srv, seen = retryServer(t, replies(sseEvent(textDelta)))
	build, _ = builder(srv.URL, true)
	chk = &check{resend: true}
	types, err = collect(wire.StreamEventsRetry(context.Background(), "p", llmprovider.NewStaticToken("t"), build, responses.Events, chk.retry))
	if err == nil || len(types) != 1 || seen() != "1@t" || len(chk.asked) != 0 {
		t.Errorf("a failure after an event: events %v, %v, requests %s, check asked %d; want the failure, one request, no check",
			types, err, seen(), len(chk.asked))
	}
}
