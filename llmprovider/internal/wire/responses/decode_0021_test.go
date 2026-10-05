package responses

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestDecode_EmptyOutputIsIncomplete (0021-MADR W4): a completed answer with
// nothing usable is ErrIncomplete, from Decode and from ReadStream, as the
// other wires have it.
func TestDecode_EmptyOutputIsIncomplete(t *testing.T) {
	body := `{"status":"completed","output":[{"type":"reasoning","summary":[]}]}`
	if res, err := Decode(strings.NewReader(body)); !errors.Is(err, llmprovider.ErrIncomplete) {
		t.Errorf("Decode = %+v, %v; want ErrIncomplete", res, err)
	}
	stream := sse(`{"type":"response.output_item.done","item":{"type":"reasoning","summary":[]}}`,
		`{"type":"response.completed","response":{"id":"r1"}}`)
	if _, err := ReadStream("p", strings.NewReader(stream)); !errors.Is(err, llmprovider.ErrIncomplete) {
		t.Errorf("ReadStream: %v; want ErrIncomplete", err)
	}
}

// TestDecode_RefusalIsKept (0021-MADR W4): a refusal part is the answer's
// text, and the answer finishes content_filter.
func TestDecode_RefusalIsKept(t *testing.T) {
	item := `{"type":"message","content":[{"type":"refusal","refusal":"I can't help with that."}]}`
	res, err := Decode(strings.NewReader(`{"status":"completed","output":[` + item + `]}`))
	if err != nil || res.OutputText() != "I can't help with that." || res.FinishReason != llmprovider.FinishContentFilter {
		t.Errorf("Decode = %+v, %v; want the refusal's text, content_filter", res, err)
	}
	res, err = ReadStream("p", strings.NewReader(sse(`{"type":"response.output_item.done","item":`+item+`}`,
		`{"type":"response.completed","response":{"id":"r1"}}`)))
	if err != nil || res.OutputText() != "I can't help with that." || res.FinishReason != llmprovider.FinishContentFilter {
		t.Errorf("ReadStream = %+v, %v; want the refusal's text, content_filter", res, err)
	}
}

// TestReadStream_FailureReturnsNoResponse (0021-MADR W14): a stream event
// that fails gives its error and no response, the partial one included,
// whether a blank line ends the event or the stream does.
func TestReadStream_FailureReturnsNoResponse(t *testing.T) {
	item := `data: {"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"partial"}]}}` + "\n\n"
	for name, event := range map[string]string{
		"incomplete":  `data: {"type":"response.incomplete","response":{"incomplete_details":{"reason":"max_output_tokens"}}}`,
		"failed":      `data: {"type":"response.failed","response":{"error":{"code":"server_error","message":"boom"}}}`,
		"error":       `data: {"type":"error","code":"server_error","message":"boom"}`,
		"undecodable": `data: {"garbled`,
	} {
		for ending, tail := range map[string]string{"blank line": "\n\n", "end of stream": ""} {
			t.Run(name+"/"+ending, func(t *testing.T) {
				res, err := ReadStream("p", strings.NewReader(item+event+tail))
				if err == nil || res != nil {
					t.Errorf("ReadStream = %+v, %v; want no response and the error", res, err)
				}
			})
		}
	}
}
