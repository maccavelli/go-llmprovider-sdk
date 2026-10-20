package messages

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestStreamIndex_0053 (0053-MADR D2): a content-block index outside 0 to
// 4095, on a start, a delta or a stop, ends the stream with ErrIncomplete
// naming the field, without a panic; 4095 is accepted.
func TestStreamIndex_0053(t *testing.T) {
	stream := func(event, index string) string {
		ev := map[string]string{
			"content_block_start": `{"type":"content_block_start","index":` + index + `,"content_block":{"type":"text","text":""}}`,
			"content_block_delta": `{"type":"content_block_delta","index":` + index + `,"delta":{"type":"text_delta","text":"x"}}`,
			"content_block_stop":  `{"type":"content_block_stop","index":` + index + `}`,
		}[event]
		return "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"model\":\"m\"}}\n\n" +
			"event: " + event + "\ndata: " + ev + "\n\n" +
			"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
	}
	run := func(s string) (panicked any, err error) {
		defer func() { panicked = recover() }()
		return nil, Events("p", strings.NewReader(s), func(llmprovider.Event) bool { return true })
	}
	for _, event := range []string{"content_block_start", "content_block_delta", "content_block_stop"} {
		for _, index := range []string{"-1", "4096", "100000"} {
			panicked, err := run(stream(event, index))
			if panicked != nil || !errors.Is(err, llmprovider.ErrIncomplete) || !strings.Contains(fmt.Sprint(err), "content_block") {
				t.Errorf("%s index %s: err %v, panic %v; want ErrIncomplete naming content_block, no panic", event, index, err, panicked)
			}
		}
		// 4095 passes the bound; this script's answer has no content, so any
		// error is that, not the bound's.
		if panicked, err := run(stream(event, "4095")); panicked != nil || strings.Contains(fmt.Sprint(err), "out of range") {
			t.Errorf("%s index 4095: err %v, panic %v; want it past the bound", event, err, panicked)
		}
	}
}
