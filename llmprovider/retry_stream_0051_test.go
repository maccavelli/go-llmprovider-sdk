package llmprovider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// attempt0051 is one call's script: its events, then err when it is set.
type attempt0051 struct {
	events []Event
	err    error
}

// streamScript0051 streams natively: each call to Stream or Generate plays
// the next attempt, and the last repeats. With hold set, Stream waits on it
// after its first event, for at most two seconds.
type streamScript0051 struct {
	attempts []attempt0051
	hold     chan struct{}
	calls    atomic.Int32
	yielded  atomic.Int32 // events Stream has yielded, in every call
	stopped  atomic.Bool  // a yield returned false
}

func (s *streamScript0051) ID() ProviderID { return "script" }

func (s *streamScript0051) Capabilities() Capabilities {
	return Capabilities{Tools: Supported, NativeStreaming: Supported}
}

func (s *streamScript0051) next() attempt0051 {
	n := int(s.calls.Add(1))
	return s.attempts[min(n, len(s.attempts))-1]
}

func (s *streamScript0051) Generate(context.Context, *Request) (*Response, error) {
	a := s.next()
	if a.err != nil {
		return nil, a.err
	}
	return a.events[len(a.events)-1].Response, nil
}

func (s *streamScript0051) Stream(context.Context, *Request) iter.Seq2[Event, error] {
	a := s.next()
	return func(yield func(Event, error) bool) {
		for i, ev := range a.events {
			s.yielded.Add(1)
			if !yield(ev, nil) {
				s.stopped.Store(true)
				return
			}
			if i == 0 && s.hold != nil {
				select {
				case <-s.hold:
				case <-time.After(2 * time.Second):
				}
			}
		}
		if a.err != nil {
			yield(Event{}, a.err)
		}
	}
}

// listingScript0051 is a streamScript0051 that lists models.
type listingScript0051 struct{ *streamScript0051 }

func (listingScript0051) ListModels(context.Context) ([]string, error) {
	return []string{"m-0051"}, nil
}

// hello0051 is a reply "hello", in two text deltas, its item, then EventDone.
func hello0051() []Event {
	item := MessageItem{Role: RoleAssistant, Text: "hello"}
	return []Event{
		{Type: EventTextDelta, Text: "hel"},
		{Type: EventTextDelta, Text: "lo"},
		{Type: EventItem, Item: item},
		{Type: EventDone, Response: &Response{ID: "r-0051", Output: []Item{item}}},
	}
}

var (
	unavailable0051 = &APIError{Status: 503, Kind: ErrProviderUnavailable}
	errCut0051      = fmt.Errorf("%w: p: read reply: %w", ErrProviderUnavailable, transport.AfterReply(io.ErrUnexpectedEOF))
)

// collect0051 ranges over a stream, returning its events and its errors.
func collect0051(seq iter.Seq2[Event, error]) ([]Event, []error) {
	var events []Event
	var errs []error
	for ev, err := range seq {
		if err != nil {
			errs = append(errs, err)
			continue
		}
		events = append(events, ev)
	}
	return events, errs
}

// TestStreamRetry_Live_0051 (0051-MADR D2): the caller has the first delta
// before the inner stream has produced the second, so nothing is held back;
// the deltas, two, join to EventDone's text.
func TestStreamRetry_Live_0051(t *testing.T) {
	s := &streamScript0051{attempts: []attempt0051{{events: hello0051()}}, hold: make(chan struct{})}
	p := WithStreamRetry(s, fastRetry, nil)
	if _, ok := p.(Streamer); !ok {
		t.Fatal("WithStreamRetry over a Streamer is not a Streamer")
	}
	var text string
	deltas := 0
	var done *Response
	for ev, err := range Stream(context.Background(), p, &Request{}) {
		if err != nil {
			t.Fatalf("Stream: %v", err)
		}
		switch ev.Type {
		case EventTextDelta:
			deltas++
			if deltas == 1 {
				if n := s.yielded.Load(); n != 1 {
					t.Errorf("the first delta reached the caller after the inner stream yielded %d events; want 1: the stream is held back", n)
				}
				close(s.hold)
			}
			text += ev.Text
		case EventDone:
			done = ev.Response
		default:
		}
	}
	if deltas < 2 || done == nil || text != done.Output[0].(MessageItem).Text {
		t.Errorf("deltas %d joined %q, EventDone %+v; want two or more joined to its text", deltas, text, done)
	}
}

// TestStreamRetry_BeforeFirstEvent_0051 (0051-MADR D2): a 503 before the
// first event is retried, with one notice, and the caller sees only the
// attempt that succeeds.
func TestStreamRetry_BeforeFirstEvent_0051(t *testing.T) {
	s := &streamScript0051{attempts: []attempt0051{{err: unavailable0051}, {events: hello0051()}}}
	var n notices
	events, errs := collect0051(Stream(context.Background(), WithStreamRetry(s, fastRetry, n.notify), &Request{}))
	if len(errs) != 0 || !reflect.DeepEqual(events, hello0051()) {
		t.Fatalf("events %v, errors %v; want the second attempt's events and no error", events, errs)
	}
	if got := n.all(); s.calls.Load() != 2 || len(got) != 1 || got[0].Attempt != 1 || !errors.Is(got[0].Err, ErrProviderUnavailable) {
		t.Errorf("calls %d, notices %+v; want 2 calls and one notice for attempt 1", s.calls.Load(), got)
	}
}

// TestStreamRetry_AfterFirstEvent_0051 (0051-MADR D2): a failure after the
// first event ends the stream with that error, unchanged, and no EventDone;
// it is not retried, though it is retryable.
func TestStreamRetry_AfterFirstEvent_0051(t *testing.T) {
	s := &streamScript0051{attempts: []attempt0051{{events: hello0051()[:1], err: errCut0051}, {events: hello0051()}}}
	var n notices
	events, errs := collect0051(Stream(context.Background(), WithStreamRetry(s, fastRetry, n.notify), &Request{}))
	if !reflect.DeepEqual(events, hello0051()[:1]) || len(errs) != 1 || !sameError(errs[0], errCut0051) {
		t.Fatalf("events %v, errors %v; want the one delta, then the failure unchanged", events, errs)
	}
	if s.calls.Load() != 1 || len(n.all()) != 0 {
		t.Errorf("calls %d, notices %d; want one call and no notice", s.calls.Load(), len(n.all()))
	}
}

// TestStreamRetry_PolicyAsGenerate_0051 (0051-MADR D2): before the first
// event, Stream stops where Generate stops, after as many calls, with the
// same error.
func TestStreamRetry_PolicyAsGenerate_0051(t *testing.T) {
	policy := RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: time.Second}
	for _, c := range []struct {
		name  string
		err   error
		calls int32
	}{
		{"503 every time", unavailable0051, 3},
		{"a failure after the reply began, every time", errCut0051, 2},
		{"a wait over MaxDelay", &APIError{Status: 429, Kind: ErrRateLimited, RetryAfter: time.Minute}, 1},
		{"a failure that is not retried", fmt.Errorf("%w: no", ErrAuthFailure), 1},
	} {
		gen := &streamScript0051{attempts: []attempt0051{{err: c.err}}}
		_, genErr := WithStreamRetry(gen, policy, nil).Generate(context.Background(), &Request{})
		s := &streamScript0051{attempts: []attempt0051{{err: c.err}}}
		events, errs := collect0051(Stream(context.Background(), WithStreamRetry(s, policy, nil), &Request{}))
		if len(events) != 0 || len(errs) != 1 || genErr == nil {
			t.Errorf("%s: events %v, errors %v, Generate %v; want one error and no event, and Generate's error", c.name, events, errs, genErr)
			continue
		}
		if s.calls.Load() != c.calls || gen.calls.Load() != c.calls || errs[0].Error() != genErr.Error() {
			t.Errorf("%s: Stream %d calls, %v; Generate %d calls, %v; want %d calls each, the same error",
				c.name, s.calls.Load(), errs[0], gen.calls.Load(), genErr, c.calls)
		}
	}
}

// TestStreamRetry_Deadline_0051 (0021-MADR T6): a wait the caller's deadline
// cannot hold is not begun, and the 429 is returned at once.
func TestStreamRetry_Deadline_0051(t *testing.T) {
	limited := &APIError{Status: 429, Kind: ErrRateLimited, RetryAfter: 5 * time.Second}
	s := &streamScript0051{attempts: []attempt0051{{err: limited}}}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	start := time.Now()
	_, errs := collect0051(Stream(ctx, WithStreamRetry(s, RetryPolicy{MaxDelay: time.Minute}, nil), &Request{}))
	if len(errs) != 1 || !sameError(errs[0], limited) || s.calls.Load() != 1 || time.Since(start) > 50*time.Millisecond {
		t.Errorf("errors %v, calls %d after %s; want the 429 at once, after one call", errs, s.calls.Load(), time.Since(start))
	}
}

// TestStreamRetry_CancelledWait_0051 (0021-MADR T6): a wait cut short ends the
// stream with an error that is both the context's and the failure.
func TestStreamRetry_CancelledWait_0051(t *testing.T) {
	s := &streamScript0051{attempts: []attempt0051{{err: unavailable0051}}}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(20*time.Millisecond, cancel)
	_, errs := collect0051(Stream(ctx, WithStreamRetry(s, RetryPolicy{BaseDelay: time.Minute, MaxDelay: time.Minute}, nil), &Request{}))
	var apiErr *APIError
	if len(errs) != 1 || !errors.Is(errs[0], context.Canceled) || !errors.As(errs[0], &apiErr) || apiErr.Status != 503 {
		t.Errorf("errors %v; want one matching context.Canceled and the 503", errs)
	}
}

// TestStreamRetry_StopEarly_0051 (0031-MADR D3): a caller that stops after
// the first event stops the inner stream, which is not called again.
func TestStreamRetry_StopEarly_0051(t *testing.T) {
	s := &streamScript0051{attempts: []attempt0051{{events: hello0051()}}}
	for range Stream(context.Background(), WithStreamRetry(s, fastRetry, nil), &Request{}) {
		break
	}
	if !s.stopped.Load() || s.calls.Load() != 1 {
		t.Errorf("inner stopped %v after %d calls; want stopped after one", s.stopped.Load(), s.calls.Load())
	}
}

// TestStreamRetry_Methods_0051 (0051-MADR D3): the result's Capabilities are
// the inner provider's, NativeStreaming included; it lists when the inner
// provider lists, and only then; Generate retries the whole generation.
func TestStreamRetry_Methods_0051(t *testing.T) {
	s := &streamScript0051{attempts: []attempt0051{{err: unavailable0051}, {events: hello0051()}}}
	p := WithStreamRetry(s, fastRetry, nil)
	if !reflect.DeepEqual(p.Capabilities(), s.Capabilities()) || p.ID() != s.ID() {
		t.Errorf("Capabilities %+v, ID %q; want the inner provider's %+v, %q", p.Capabilities(), p.ID(), s.Capabilities(), s.ID())
	}
	if _, ok := p.(ModelLister); ok {
		t.Error("the result lists models, though the inner provider does not")
	}
	if resp, err := p.Generate(context.Background(), &Request{}); err != nil || resp.ID != "r-0051" || s.calls.Load() != 2 {
		t.Errorf("Generate = %v, %v after %d calls; want the response after 2", resp, err, s.calls.Load())
	}
	l, ok := WithStreamRetry(listingScript0051{s}, fastRetry, nil).(ModelLister)
	if !ok {
		t.Fatal("the result over a ModelLister does not list")
	}
	if ids, err := l.ListModels(context.Background()); err != nil || !reflect.DeepEqual(ids, []string{"m-0051"}) {
		t.Errorf("ListModels = %v, %v; want the inner provider's", ids, err)
	}
	if _, ok := l.(Streamer); !ok {
		t.Error("the listing result over a Streamer is not a Streamer")
	}
}

// TestStreamRetry_NotAStreamer_0051 (0051-MADR D3; R10): over a provider that
// does not stream natively, the result is WithRetryNotify's: no Streamer,
// NativeStreaming Unsupported, and Stream retried whole through Generate.
func TestStreamRetry_NotAStreamer_0051(t *testing.T) {
	inner, calls := failing(unavailable0051)
	p := WithStreamRetry(inner, fastRetry, nil)
	if _, ok := p.(Streamer); ok || p.Capabilities().NativeStreaming != Unsupported {
		t.Fatalf("over a provider that does not stream: Streamer %v, NativeStreaming %v; want neither", ok, p.Capabilities().NativeStreaming)
	}
	events, errs := collect0051(Stream(context.Background(), p, &Request{}))
	if len(errs) != 0 || calls.Load() != 2 || events[len(events)-1].Type != EventDone {
		t.Errorf("events %v, errors %v, calls %d; want EventDone after 2 calls", events, errs, calls.Load())
	}
}
