package responses

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// longStream is a realistic answer of n text deltas, each about 250 bytes on
// the wire, then the done and completed events.
func longStream(n int) string {
	var b strings.Builder
	delta := `{"type":"response.output_text.delta","item_id":"msg_1","output_index":0,"content_index":0,` +
		`"delta":"word","sequence_number":12345,"obfuscation":"` + strings.Repeat("x", 120) + `"}`
	for range n {
		b.WriteString("event: response.output_text.delta\ndata: " + delta + "\n\n")
	}
	b.WriteString(sse(
		`{"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"done"}]}}`,
		`{"type":"response.completed","response":{"id":"r1"}}`,
	))
	return b.String()
}

// TestReadStream_LongStreamCompletes (0021-MADR W2): a stream longer than
// 16 MiB, made of ordinary events, completes.
func TestReadStream_LongStreamCompletes(t *testing.T) {
	stream := longStream(68826)
	if len(stream) <= 16<<20 {
		t.Fatalf("the stream is %d bytes, want more than 16 MiB", len(stream))
	}
	res, err := ReadStream("p", strings.NewReader(stream))
	if err != nil || res.OutputText() != "done" {
		t.Fatalf("a %d-byte stream: %+v, %v; want it read whole", len(stream), res, err)
	}
}

// TestReadStream_OverlongEventIsIncomplete (0021-MADR W2): one event over the
// event limit is ErrIncomplete, never a retryable failure.
func TestReadStream_OverlongEventIsIncomplete(t *testing.T) {
	event := `{"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"` +
		strings.Repeat("x", 16<<20) + `"}]}}`
	_, err := ReadStream("p", strings.NewReader(sse(event)))
	if !errors.Is(err, llmprovider.ErrIncomplete) || errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Fatalf("an event over 16 MiB: %v; want ErrIncomplete", err)
	}
}

// TestReadStream_EndWithoutCompletedIsAfterReply (0021-MADR D1): a stream that
// ends early is retryable, and marked after the reply, so it is retried once.
func TestReadStream_EndWithoutCompletedIsAfterReply(t *testing.T) {
	_, err := ReadStream("p", strings.NewReader(sse(`{"type":"response.created","response":{"id":"r0"}}`)))
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) || !transport.IsAfterReply(err) {
		t.Fatalf("an early end: %v; want ErrProviderUnavailable after the reply", err)
	}
	if _, err := ReadStream("p", errReader{}); !transport.IsAfterReply(err) {
		t.Fatalf("a read error: %v; want it after the reply", err)
	}
}

// BenchmarkReadStream_Deltas4MiB (0021-MADR W2): a 4 MiB long answer, mostly
// text deltas, the shape W2 measured. It holds the step's target of a tenth
// of the old allocations (0021-PLAN deviation of 2026-10-04).
func BenchmarkReadStream_Deltas4MiB(b *testing.B) {
	stream := longStream(4 << 20 / 250)
	b.SetBytes(int64(len(stream)))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := ReadStream("p", strings.NewReader(stream)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReadStream_4MiB (0021-MADR W2): the ChatGPT text fixture's events
// repeated to 4 MiB. It repeats response.created, which a real stream sends
// once, so it is recorded as information.
func BenchmarkReadStream_4MiB(b *testing.B) {
	fixture, err := os.ReadFile("../../../providers/openai/testdata/chatgpt-text.sse")
	if err != nil {
		b.Fatal(err)
	}
	events := string(fixture)
	cut := strings.Index(events, "event: response.completed")
	if cut < 0 {
		b.Fatal("no response.completed in the fixture")
	}
	head, tail := events[:cut], events[cut:]
	var s strings.Builder
	for s.Len() < 4<<20 {
		s.WriteString(head)
	}
	s.WriteString(tail)
	stream := s.String()
	b.SetBytes(int64(len(stream)))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := ReadStream("p", strings.NewReader(stream)); err != nil {
			b.Fatal(err)
		}
	}
}
