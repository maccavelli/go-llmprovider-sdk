package chatcompletions

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// detailsStream0053 is n reasoning_details fragments of four bytes each,
// then a text delta and [DONE].
func detailsStream0053(n int) string {
	var b strings.Builder
	for range n {
		b.WriteString("data: " + d(`{"reasoning_details":[{"type":"reasoning.text","index":0,"text":"abcd"}]}`, "") + "\n\n")
	}
	return b.String() + chunks(d(`{"content":"ok"}`, "stop"))
}

// allocated is the bytes f allocates, the least of three runs.
func allocated(f func()) uint64 {
	least := uint64(0)
	for i := range 3 {
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		f()
		runtime.ReadMemStats(&after)
		if n := after.TotalAlloc - before.TotalAlloc; i == 0 || n < least {
			least = n
		}
	}
	return least
}

func drain0053(stream string) error {
	return Events("p", strings.NewReader(stream), func(llmprovider.Event) bool { return true })
}

// TestEvents_ReasoningDetailsLinear_0053 (0053-MADR D1): eight times the
// fragments allocate less than twelve times the bytes. Joining each fragment
// to the text before it, as before, allocated 55 times (M1).
func TestEvents_ReasoningDetailsLinear_0053(t *testing.T) {
	small, large := detailsStream0053(2000), detailsStream0053(16000)
	a := allocated(func() { _ = drain0053(small) })
	b := allocated(func() { _ = drain0053(large) })
	if b >= 12*a {
		t.Errorf("16,000 fragments allocated %d bytes, 2,000 allocated %d: %.1f times; want under 12", b, a, float64(b)/float64(a))
	}
}

func BenchmarkEvents_ReasoningDetails_0053(b *testing.B) {
	for _, n := range []int{2000, 16000} {
		stream := detailsStream0053(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if err := drain0053(stream); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// TestReasoningDetailsReplay_0053 (0053-MADR D1): an entry whose text,
// summary and signature are split across fragments replays as the entry sent
// whole: EventDone's Response is Decode's on the same answer.
func TestReasoningDetailsReplay_0053(t *testing.T) {
	stream := chunks(
		d(`{"role":"assistant","reasoning_details":[{"type":"reasoning.text","index":0,"text":"thi","signature":"s1","format":"unknown"}]}`, ""),
		d(`{"reasoning_details":[{"index":0,"text":"nk","summary":"su","signature":"s2"}]}`, ""),
		d(`{"reasoning_details":[{"index":0,"summary":"m","format":"ignored"}]}`, ""),
		d(`{"content":"ok"}`, "stop"))
	reply := `{"id":"c1","model":"m","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"ok",` +
		`"reasoning_details":[{"format":"unknown","index":0,"signature":"s1s2","summary":"sum","text":"think","type":"reasoning.text"}]}}]}`
	events, _, err := collect(t, stream)
	if err != nil {
		t.Fatal(err)
	}
	want, err := Decode(strings.NewReader(reply))
	if err != nil {
		t.Fatal(err)
	}
	if done := events[len(events)-1].Response; !reflect.DeepEqual(done, want) {
		t.Errorf("EventDone's Response %+v; Decode's on the whole entry %+v", done, want)
	}
}

// TestStreamIndex_0053 (0053-MADR D2): an index outside 0 to 4095, of a tool
// call or a reasoning_details entry, ends the stream with ErrIncomplete
// naming the field, without a panic or a large allocation; 4095 is accepted.
func TestStreamIndex_0053(t *testing.T) {
	call := func(index string) string {
		return chunks(d(`{"tool_calls":[{"index":`+index+`,"id":"c","function":{"name":"f","arguments":"{}"}}]}`, "tool_calls"))
	}
	detail := func(index string) string {
		return chunks(d(`{"reasoning_details":[{"type":"reasoning.text","index":`+index+`,"text":"x"}]}`, ""), d(`{"content":"ok"}`, "stop"))
	}
	for _, tc := range []struct {
		name, stream, field string
	}{
		{"tool_calls -1", call("-1"), "tool_calls"},
		{"tool_calls 4096", call("4096"), "tool_calls"},
		{"tool_calls 100000", call("100000"), "tool_calls"},
		{"reasoning_details -1", detail("-1"), "reasoning_details"},
		{"reasoning_details 4096", detail("4096"), "reasoning_details"},
		{"reasoning_details 1e5", detail("1e5"), "reasoning_details"},
		{"reasoning_details 1.5", detail("1.5"), "reasoning_details"},
	} {
		var err error
		var panicked any
		n := allocated(func() {
			defer func() { panicked = recover() }()
			err = drain0053(tc.stream)
		})
		if panicked != nil || !errors.Is(err, llmprovider.ErrIncomplete) || !strings.Contains(fmt.Sprint(err), tc.field) || n > 1<<20 {
			t.Errorf("%s: err %v, panic %v, %d bytes; want ErrIncomplete naming %s, no panic, under 1 MiB", tc.name, err, panicked, n, tc.field)
		}
	}
	for _, stream := range []string{call("4095"), detail("4095")} {
		if err := drain0053(stream); err != nil {
			t.Errorf("index 4095: %v; want it accepted", err)
		}
	}
}
