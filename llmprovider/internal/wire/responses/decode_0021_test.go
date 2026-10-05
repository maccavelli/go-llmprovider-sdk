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
