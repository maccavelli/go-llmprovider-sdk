package wire

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// readAll runs ReadEvents over stream, collecting each event's data, and
// stopping after stopAt events when it is above 0.
func readAll(stream io.Reader, stopAt int) ([]string, error) {
	var got []string
	err := ReadEvents("p", stream, "the end event", func(data []byte) (bool, error) {
		got = append(got, string(data))
		return data == nil || string(data) == "end" || (stopAt > 0 && len(got) >= stopAt), nil
	})
	return got, err
}

// TestReadEvents_Framing (0031-MADR E2): data lines join into one event with
// "\n"; comment, event: and id: lines are not data; an event with no data is
// not dispatched.
func TestReadEvents_Framing(t *testing.T) {
	stream := ": a comment\nevent: first\ndata: {\"a\":\ndata: 1}\n\n" +
		"id: 7\nevent: no-data\n\n" +
		"data:no space\r\n\r\n" +
		"data: end\n\n" +
		"data: after the end\n\n"
	got, err := readAll(strings.NewReader(stream), 0)
	want := []string{"{\"a\":\n1}", "no space", "end"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("events %q (%v), want %q", got, err, want)
	}
}

// TestReadEvents_LastEventWithoutBlankLine: an event the stream ends inside is
// still dispatched.
func TestReadEvents_LastEventWithoutBlankLine(t *testing.T) {
	got, err := readAll(strings.NewReader("data: one\n\ndata: end"), 0)
	if err != nil || !reflect.DeepEqual(got, []string{"one", "end"}) {
		t.Fatalf("events %q (%v), want one and end", got, err)
	}
}

// TestReadEvents_EventTooLarge (0021-MADR W2): an event over 16 MiB, in one
// line or across several, is ErrIncomplete, never retryable.
func TestReadEvents_EventTooLarge(t *testing.T) {
	big := strings.Repeat("x", 16<<20)
	for name, stream := range map[string]string{
		"one line":    "data: " + big + "x\n\n",
		"joined data": "data: " + big[:8<<20] + "\ndata: " + big[:8<<20] + "\n\n",
	} {
		_, err := readAll(strings.NewReader(stream), 0)
		if !errors.Is(err, llmprovider.ErrIncomplete) || errors.Is(err, llmprovider.ErrProviderUnavailable) {
			t.Errorf("%s: %v; want ErrIncomplete", name, err)
		}
	}
}

// failingReader yields data, then a read failure.
type failingReader struct{ data *strings.Reader }

func (f failingReader) Read(p []byte) (int, error) {
	if f.data.Len() > 0 {
		return f.data.Read(p)
	}
	return 0, errors.New("connection reset")
}

// TestReadEvents_ReadFailure (0021-MADR D1): a read failure is
// ErrProviderUnavailable, marked after the reply.
func TestReadEvents_ReadFailure(t *testing.T) {
	_, err := readAll(failingReader{strings.NewReader("data: one\n\n")}, 0)
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) || !transport.IsAfterReply(err) {
		t.Fatalf("a read failure: %v; want ErrProviderUnavailable after the reply", err)
	}
}

// TestReadEvents_EndsEarly (0021-MADR D1): a stream that ends before the
// caller stops is ErrProviderUnavailable, marked after the reply, naming the
// event it lacked.
func TestReadEvents_EndsEarly(t *testing.T) {
	_, err := readAll(strings.NewReader("data: one\n\n"), 0)
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) || !transport.IsAfterReply(err) ||
		!strings.Contains(err.Error(), "stream ended before the end event") {
		t.Fatalf("an early end: %v; want ErrProviderUnavailable after the reply, naming the end event", err)
	}
}

// countingReader counts the bytes read through it.
type countingReader struct {
	r io.Reader
	n int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	return n, err
}

// TestReadEvents_StopReadsNoFurther: a dispatch that stops ends the read at
// once, with no error.
func TestReadEvents_StopReadsNoFurther(t *testing.T) {
	stream := strings.Repeat("data: "+strings.Repeat("x", 200)+"\n\n", 10000)
	body := &countingReader{r: strings.NewReader(stream)}
	got, err := readAll(body, 1)
	if err != nil || len(got) != 1 || body.n >= len(stream)/2 {
		t.Fatalf("stopped after %d events, %v, having read %d of %d bytes", len(got), err, body.n, len(stream))
	}
}

// TestReadEvents_DispatchError: a dispatch error ends the read with it.
func TestReadEvents_DispatchError(t *testing.T) {
	want := errors.New("bad event")
	err := ReadEvents("p", strings.NewReader("data: one\n\ndata: two\n\n"), "the end event", func([]byte) (bool, error) {
		return false, want
	})
	if !errors.Is(err, want) {
		t.Fatalf("a dispatch error: %v; want it returned", err)
	}
}
