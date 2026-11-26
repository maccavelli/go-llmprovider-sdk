package wire

import (
	"fmt"
	"iter"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// A plain Generate whose output limit the SDK chose or left out is read over
// the service's stream and returned whole (0088-MADR ST88, OQ15 (1)): a long
// answer then sends its first byte at once, where a plain request sends
// nothing until the whole answer is generated and can outlast the reply's
// wait. A Generate that names its own limit stays a plain request.

// OverStream reports whether a plain Generate of req is read over the
// stream: neither the request's MaxOutputTokens nor a given option (given:
// Settings.MaxTokensGiven, or a listing probe's 8192) names its limit.
func OverStream(req *llmprovider.Request, given bool) bool {
	return req != nil && req.MaxOutputTokens <= 0 && !given
}

// StreamAnswerLimit bounds an answer Collect reads, on every wire: its text,
// reasoning and call-argument bytes, reasoning counted so that a runaway
// stream stops. It is twice ReplyLimit, which a plain reply's body keeps
// (0091-MADR ST71, "Raise the bound", "Every provider, one bound
// (Recommended)"): ReplyLimit's 16 MiB allowed 32 bytes for each of the
// 524,288 output tokens the largest recorded output limit states, and a
// server that streams both a summary and the raw text of its reasoning has
// that reasoning counted twice.
const StreamAnswerLimit = 2 * ReplyLimit

// Collect is Generate's response read from stream, a provider's own Stream of
// the request (0088-MADR ST88a to ST88d):
//
//   - A failure before the first event, such as a refused opening or an
//     error inside a 200 as the stream's first payload, is returned as
//     the stream classified it, so WithRetry retries it as it retries a
//     plain Generate (ST88a, ST88d); the stream resent it already when a
//     retry check said to (StreamEventsRetry).
//   - A failure after an event is marked transport.AfterEvent: nothing is
//     resent after an event (ST88a).
//   - The answer's text and reasoning deltas and its calls' arguments are
//     bounded by StreamAnswerLimit, on top of each event's EventLimit: past
//     it, ErrIncomplete and no response (ST88b).
//   - The response is EventDone's, its usage as the stream reported it, zero
//     when the server sent none (ST88c).
func Collect(provider string, stream iter.Seq2[llmprovider.Event, error]) (*llmprovider.Response, error) {
	started, size := false, 0
	for ev, err := range stream {
		if err != nil {
			if started {
				return nil, transport.AfterEvent(err)
			}
			return nil, err
		}
		switch ev.Type {
		case llmprovider.EventDone:
			if ev.Response == nil {
				return nil, fmt.Errorf("%w: %s: the stream's last event held no response", llmprovider.ErrIncomplete, provider)
			}
			return ev.Response, nil
		case llmprovider.EventTextDelta, llmprovider.EventReasoningDelta:
			size += len(ev.Text)
		case llmprovider.EventItem:
			if call, ok := ev.Item.(llmprovider.FunctionCallItem); ok {
				size += len(call.Arguments)
			}
		default:
			// An event type this reader does not know carries no answer.
		}
		started = true
		if size > StreamAnswerLimit {
			return nil, fmt.Errorf("%w: %s: the answer read over the stream is over %d MiB", llmprovider.ErrIncomplete,
				provider, StreamAnswerLimit>>20)
		}
	}
	return nil, fmt.Errorf("%w: %s: the stream ended with no response", llmprovider.ErrIncomplete, provider)
}
