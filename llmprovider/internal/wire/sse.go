package wire

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// EventLimit bounds one server-sent event. The stream as a whole has no limit:
// a long answer is many events, and the idle limit bounds a stalled one
// (0021-MADR W2, D2).
const EventLimit = 16 << 20

var (
	dataPrefix = []byte("data:")
	// errLineTooLong is a line over the event limit.
	errLineTooLong = errors.New("stream line too long")
)

// ReadEvents reads a server-sent event stream from body, every wire's stream
// (0031-MADR E2). An event's data lines are joined with "\n" and passed to
// dispatch, which may keep them only for the call; comment, event: and id:
// lines are not data, and an event with no data is not dispatched. It returns
// nil once dispatch says stop, and dispatch's error when it fails.
//
// An event over EventLimit is ErrIncomplete: it was sent whole, and asking
// again would buy the same. A failure to read the body is
// ErrProviderUnavailable, and so is a stream that ends before dispatch says
// stop, naming done, the event it lacked: both are marked
// transport.AfterReply, since the service may have generated the answer, so
// WithRetry asks again at most once (0021-MADR D1). So is dispatch's error,
// such as a failure event the service sent inside the stream: the service
// had begun its reply, so WithRetry and WithStreamRetry ask again at most
// once (0053-MADR D3). The mark changes neither its text nor its kind.
func ReadEvents(provider string, body io.Reader, done string, dispatch func(data []byte) (stop bool, err error)) error {
	reader := bufio.NewReaderSize(body, 64<<10)
	var data []byte
	next := func() (bool, error) {
		payload := data
		data = data[:0]
		if len(payload) == 0 {
			return false, nil
		}
		return dispatch(payload)
	}
	tooLarge := fmt.Errorf("%w: %s: stream event over %d MiB", llmprovider.ErrIncomplete, provider, EventLimit>>20)
	for {
		line, err := readLine(reader, EventLimit)
		if errors.Is(err, errLineTooLong) {
			return tooLarge
		}
		switch {
		case len(line) == 0 && err == nil:
			if stop, err := next(); stop || err != nil {
				return transport.AfterReply(err)
			}
		case bytes.HasPrefix(line, dataPrefix):
			if len(data) > 0 {
				data = append(data, '\n')
			}
			payload := bytes.TrimPrefix(bytes.TrimPrefix(line, dataPrefix), []byte(" "))
			if len(data)+len(payload) > EventLimit {
				return tooLarge
			}
			data = append(data, payload...)
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return fmt.Errorf("%w: %s: read stream: %w", llmprovider.ErrProviderUnavailable, provider,
					transport.AfterReply(err))
			}
			break
		}
	}
	if stop, err := next(); stop || err != nil {
		return transport.AfterReply(err)
	}
	return fmt.Errorf("%w: %s: %w", llmprovider.ErrProviderUnavailable, provider,
		transport.AfterReply(fmt.Errorf("stream ended before %s", done)))
}

// readLine reads one line without its line ending, up to limit bytes. At the
// end of the stream it returns the last line, which may be empty, with io.EOF.
// A line that fits the reader's buffer is the reader's own slice, valid until
// the next read; only a longer one is copied.
func readLine(r *bufio.Reader, limit int) ([]byte, error) {
	chunk, err := r.ReadSlice('\n')
	line := chunk
	if errors.Is(err, bufio.ErrBufferFull) {
		line = append([]byte(nil), chunk...)
		for errors.Is(err, bufio.ErrBufferFull) {
			if chunk, err = r.ReadSlice('\n'); len(line)+len(chunk) > limit+2 { // +2 for the line ending
				return nil, errLineTooLong
			}
			line = append(line, chunk...)
		}
	}
	line = bytes.TrimSuffix(bytes.TrimSuffix(line, []byte("\n")), []byte("\r"))
	if len(line) > limit {
		return nil, errLineTooLong
	}
	return line, err
}

// StreamEvents sends c as an event stream through Reauth, and yields the
// events decode reads from its reply as they arrive (0031-MADR D1-D4, E2). A
// refused credential arrives before any event, so src's token is renewed and
// c sent once more. A failure, before or during the stream, is yielded once
// with a zero Event and ends it; a caller that stops ranging closes the reply.
func StreamEvents(ctx context.Context, c Call, src llmprovider.TokenSource,
	decode func(provider string, body io.Reader, yield func(llmprovider.Event) bool) error) iter.Seq2[llmprovider.Event, error] {
	return StreamEventsFor(ctx, src, func(llmprovider.Token) Call { return c }, decode)
}

// StreamEventsFor is StreamEvents for a request built from the token it is
// sent with, as Kilo's endpoint is from its key.
func StreamEventsFor(ctx context.Context, src llmprovider.TokenSource, build func(llmprovider.Token) Call,
	decode func(provider string, body io.Reader, yield func(llmprovider.Event) bool) error) iter.Seq2[llmprovider.Event, error] {
	// The label names the provider before a token is fetched, for a failure
	// to fetch one.
	return StreamEventsRetry(ctx, build(llmprovider.Token{}).Provider, src, build, decode, nil)
}

// StreamEventsRetry is StreamEventsFor with a retry check: a stream whose
// opening fails, which is before any event, is built again and opened once
// more when retry says to. A failure after the stream opened is never resent
// (0034-MADR D3). provider labels a failure to fetch a token.
func StreamEventsRetry(ctx context.Context, provider string, src llmprovider.TokenSource, build func(llmprovider.Token) Call,
	decode func(provider string, body io.Reader, yield func(llmprovider.Event) bool) error, retry Retry) iter.Seq2[llmprovider.Event, error] {
	return func(yield func(llmprovider.Event, error) bool) {
		var warnings []llmprovider.Warning // the call's, added to the done event's Response
		open := func(token llmprovider.Token) (*Stream, error) {
			c := build(token)
			warnings = c.Warnings
			return OpenStream(ctx, c, token)
		}
		reply, err := Reauth(ctx, provider, src, open)
		if err != nil && retry != nil {
			var resend bool
			if resend, err = retry(err); resend {
				reply, err = Reauth(ctx, provider, src, open)
			}
		}
		if err != nil {
			yield(llmprovider.Event{}, err)
			return
		}
		stopped := false
		err = reply.Read(func(body io.Reader) error {
			return decode(provider, body, func(ev llmprovider.Event) bool {
				if ev.Type == llmprovider.EventDone && ev.Response != nil {
					ev.Response.Warnings = append(ev.Response.Warnings, warnings...)
				}
				stopped = !yield(ev, nil)
				return !stopped
			})
		})
		if err != nil && !stopped {
			yield(llmprovider.Event{}, err)
		}
	}
}
