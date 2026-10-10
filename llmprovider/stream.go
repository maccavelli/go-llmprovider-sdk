package llmprovider

import (
	"context"
	"fmt"
	"iter"
)

// EventType names what an Event carries.
type EventType string

const (
	// EventTextDelta carries output text in Event.Text.
	EventTextDelta EventType = "text_delta"
	// EventReasoningDelta carries reasoning text in Event.Text.
	EventReasoningDelta EventType = "reasoning_delta"
	// EventItem carries one finished output item in Event.Item.
	EventItem EventType = "item"
	// EventDone carries the whole response in Event.Response. It is the last
	// event of a stream that succeeds.
	EventDone EventType = "done"
)

// Event is one step of a streamed response (0015-MADR D4). Code that switches
// on Type has a default case: new types may be added.
type Event struct {
	Type     EventType
	Text     string
	Item     Item
	Response *Response
}

// Streamer is implemented by a provider that streams natively.
type Streamer interface {
	Stream(ctx context.Context, req *Request) iter.Seq2[Event, error]
}

// Stream streams req from p, for every provider (0015-MADR D4). It uses p's
// own Streamer when there is one. Otherwise it runs Generate and emits its
// result: a delta for each message and reasoning item, then each item, then
// EventDone. A failure is yielded once, with a zero Event, and ends the
// stream. Only EventDone carries a Response, and it comes last: the deltas
// and items a native stream yielded before a failure are not a response, and
// a caller that showed them shows them as incomplete (0031-MADR D4).
func Stream(ctx context.Context, p Provider, req *Request) iter.Seq2[Event, error] {
	if s, ok := p.(Streamer); ok {
		return s.Stream(ctx, req)
	}
	return func(yield func(Event, error) bool) {
		resp, err := p.Generate(ctx, req)
		if err == nil && resp == nil {
			err = fmt.Errorf("%w: %s returned no response", ErrIncomplete, p.ID())
		}
		if err != nil {
			yield(Event{}, err)
			return
		}
		for _, ev := range responseEvents(resp) {
			if !yield(ev, nil) {
				return
			}
		}
	}
}

// responseEvents is the event sequence for a response generated whole.
func responseEvents(resp *Response) []Event {
	events := make([]Event, 0, 2*len(resp.Output)+1)
	for _, item := range resp.Output {
		switch it := item.(type) {
		case MessageItem:
			events = append(events, Event{Type: EventTextDelta, Text: it.Text})
		case ReasoningItem:
			// Reasoning held only in encrypted form has no text to stream
			// (0020-MADR F24); it still arrives as an item.
			if it.Text != "" {
				events = append(events, Event{Type: EventReasoningDelta, Text: it.Text})
			}
		default:
			// Calls and call outputs arrive only as items.
		}
		events = append(events, Event{Type: EventItem, Item: item})
	}
	return append(events, Event{Type: EventDone, Response: resp})
}
