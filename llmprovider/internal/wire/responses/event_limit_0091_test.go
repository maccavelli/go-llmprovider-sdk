package responses

import (
	"errors"
	"fmt"
	"iter"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// collect0091 reads stream with Events through wire.Collect, as a Generate
// read over the stream is.
func collect0091(stream string) (*llmprovider.Response, error) {
	var seq iter.Seq2[llmprovider.Event, error] = func(yield func(llmprovider.Event, error) bool) {
		stopped := false
		err := Decoder{}.Events("p", strings.NewReader(stream), func(ev llmprovider.Event) bool {
			if !yield(ev, nil) {
				stopped = true
				return false
			}
			return true
		})
		if err != nil && !stopped {
			yield(llmprovider.Event{}, err)
		}
	}
	return wire.Collect("p", seq)
}

// TestCollect_ResponsesTerminalOverEventLimit_0091 (0091-MADR ST72, "Correct
// the record and changelog (Recommended)"): a Responses answer over
// EventLimit, 16 MiB, whose terminal response.completed repeats it in output,
// fails at that event, though Collect's StreamAnswerLimit is 32 MiB. Two
// messages of 9 MiB, each delivered by deltas and an output_item.done under
// 16 MiB, end in a terminal of both, about 18 MiB: ErrIncomplete naming the
// event bound, and no response.
func TestCollect_ResponsesTerminalOverEventLimit_0091(t *testing.T) {
	chunk := strings.Repeat("x", 64<<10)
	text := strings.Repeat(chunk, 144)
	var b strings.Builder
	b.WriteString(tfCreated)
	for i, id := range []string{"msg_1", "msg_2"} {
		b.WriteString(tfAdded(i, tfMsg(id, "")))
		for range 144 {
			b.WriteString(tfEvent("response.output_text.delta",
				fmt.Sprintf(`,"output_index":%d,"content_index":0,"delta":"%s"`, i, chunk)))
		}
		b.WriteString(tfDone(i, tfMsg(id, text)))
	}
	b.WriteString(tfCompleted(tfMsg("msg_1", text), tfMsg("msg_2", text)))
	resp, err := collect0091(b.String())
	if resp != nil || !errors.Is(err, llmprovider.ErrIncomplete) || !strings.Contains(fmt.Sprint(err), "stream event over 16 MiB") {
		n := 0
		if resp != nil {
			n = len(resp.OutputText())
		}
		t.Errorf("two 9 MiB messages and their 18 MiB terminal: %d bytes, %v; want ErrIncomplete naming the 16 MiB event bound", n, err)
	}
}
