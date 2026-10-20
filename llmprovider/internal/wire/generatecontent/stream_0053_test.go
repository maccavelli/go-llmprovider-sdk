package generatecontent

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// partsStream0053 is n text parts of four bytes each, then a finishing part.
func partsStream0053(n int) string {
	var b strings.Builder
	for range n {
		b.WriteString(`data: {"candidates":[{"content":{"role":"model","parts":[{"text":"abcd"}]},"index":0}]}` + "\n\n")
	}
	b.WriteString(`data: {"candidates":[{"content":{"role":"model","parts":[{"text":"."}]},"finishReason":"STOP","index":0}]}` + "\n\n")
	return b.String()
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

func drain0053(stream string) (string, error) {
	var text string
	err := Events("p", strings.NewReader(stream), func(ev llmprovider.Event) bool {
		if ev.Type == llmprovider.EventDone {
			text = ev.Response.OutputText()
		}
		return true
	})
	return text, err
}

// TestEvents_TextPartsLinear_0053 (0053-MADR D1): eight times the parts
// allocate less than twelve times the bytes. Joining each part with += , as
// before, allocated 62 times (M1). The joined text is unchanged.
func TestEvents_TextPartsLinear_0053(t *testing.T) {
	small, large := partsStream0053(2000), partsStream0053(16000)
	a := allocated(func() { _, _ = drain0053(small) })
	b := allocated(func() { _, _ = drain0053(large) })
	if b >= 12*a {
		t.Errorf("16,000 parts allocated %d bytes, 2,000 allocated %d: %.1f times; want under 12", b, a, float64(b)/float64(a))
	}
	text, err := drain0053(partsStream0053(3))
	if err != nil || text != "abcdabcdabcd." {
		t.Errorf("three parts and a finishing one: %q, %v; want %q", text, err, "abcdabcdabcd.")
	}
}

func BenchmarkEvents_TextParts_0053(b *testing.B) {
	for _, n := range []int{2000, 16000} {
		stream := partsStream0053(n)
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := drain0053(stream); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
