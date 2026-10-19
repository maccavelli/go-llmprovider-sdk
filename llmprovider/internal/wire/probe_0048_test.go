package wire

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// TestProbeHealth_ProbesIntoItsStore_0048 (0048-MADR D1): a listing probes
// through the store it is given, so a second store probes again and the
// first answers from its own.
func TestProbeHealth_ProbesIntoItsStore_0048(t *testing.T) {
	var probes atomic.Int32
	probe := func(context.Context, string) (string, error) { probes.Add(1); return "hello", nil }
	a, b := transport.NewProbeStore(), transport.NewProbeStore()
	for i, step := range []struct {
		store *transport.ProbeStore
		want  int32
	}{{a, 2}, {a, 2}, {b, 4}} {
		got := ProbeHealth(context.Background(), step.store, llmprovider.ProviderClaude, "https://store-0048.test",
			llmprovider.NewStaticToken("k-0048"), []string{"a", "b"}, 10, probe)
		if len(got) != 2 {
			t.Fatalf("step %d: ProbeHealth = %v; want both models", i, got)
		}
		if n := probes.Load(); n != step.want {
			t.Fatalf("step %d: %d probes in all; want %d", i, n, step.want)
		}
	}
}
