package llmprovider

import (
	"context"
	"errors"
	"iter"
	"testing"
)

func collect(t *testing.T, seq iter.Seq2[Event, error]) ([]Event, []error) {
	t.Helper()
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

func TestStream_FallbackEmitsTheResponse(t *testing.T) {
	resp := &Response{ID: "r1", Output: []Item{
		ReasoningItem{Text: "think"},
		MessageItem{Text: "hello"},
		FunctionCallItem{Name: "t", Arguments: "{}"},
	}}
	p := &stubProvider{id: "stub", gen: func(context.Context, *Request) (*Response, error) { return resp, nil }}
	events, errs := collect(t, Stream(context.Background(), p, &Request{}))
	if len(errs) != 0 {
		t.Fatalf("errors %v", errs)
	}
	want := []EventType{EventReasoningDelta, EventItem, EventTextDelta, EventItem, EventItem, EventDone}
	if len(events) != len(want) {
		t.Fatalf("%d events %+v, want %d", len(events), events, len(want))
	}
	for i, ev := range events {
		if ev.Type != want[i] {
			t.Errorf("event %d is %q, want %q", i, ev.Type, want[i])
		}
	}
	if events[0].Text != "think" || events[2].Text != "hello" {
		t.Errorf("deltas %q and %q, want think and hello", events[0].Text, events[2].Text)
	}
	if events[4].Item != resp.Output[2] {
		t.Errorf("the call arrived as %+v", events[4].Item)
	}
	if events[5].Response != resp {
		t.Error("EventDone does not carry the response")
	}
}

func TestStream_FallbackYieldsTheErrorOnce(t *testing.T) {
	p := &stubProvider{id: "stub", gen: func(context.Context, *Request) (*Response, error) { return nil, ErrRateLimited }}
	events, errs := collect(t, Stream(context.Background(), p, &Request{}))
	if len(events) != 0 || len(errs) != 1 || !errors.Is(errs[0], ErrRateLimited) {
		t.Fatalf("events %+v, errors %v; want one ErrRateLimited", events, errs)
	}
	p.gen = func(context.Context, *Request) (*Response, error) { return nil, nil }
	if _, errs := collect(t, Stream(context.Background(), p, &Request{})); len(errs) != 1 || !errors.Is(errs[0], ErrIncomplete) {
		t.Fatalf("a nil response gave %v, want one ErrIncomplete", errs)
	}
}

func TestStream_FallbackStopsWhenTheCallerStops(t *testing.T) {
	p := &stubProvider{id: "stub", gen: func(context.Context, *Request) (*Response, error) {
		return &Response{Output: []Item{MessageItem{Text: "a"}, MessageItem{Text: "b"}}}, nil
	}}
	n := 0
	for range Stream(context.Background(), p, &Request{}) {
		n++
		break
	}
	if n != 1 {
		t.Fatalf("%d events after break, want 1", n)
	}
}

// nativeStreamer is a provider with its own Stream.
type nativeStreamer struct{ stubProvider }

func (nativeStreamer) Stream(context.Context, *Request) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		yield(Event{Type: EventTextDelta, Text: "native"}, nil)
	}
}

func TestStream_UsesANativeStreamer(t *testing.T) {
	p := &nativeStreamer{stubProvider{id: "stub", gen: func(context.Context, *Request) (*Response, error) {
		t.Fatal("Stream called Generate on a native streamer")
		return nil, nil
	}}}
	events, _ := collect(t, Stream(context.Background(), p, &Request{}))
	if len(events) != 1 || events[0].Text != "native" {
		t.Fatalf("events %+v, want the native one", events)
	}
}
