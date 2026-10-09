package wire

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// unreadableToken is a TokenSource whose token cannot be read.
type unreadableToken struct{}

func (unreadableToken) Token(context.Context) (llmprovider.Token, error) {
	return llmprovider.Token{}, errors.New("the token store is locked")
}

// TestProbeHealth_UnreadableTokenProbesUncached (0028-MADR D-A3): with no
// token to fingerprint, the listing is probed every time, never cached
// under a key without its credential.
func TestProbeHealth_UnreadableTokenProbesUncached(t *testing.T) {
	var probes atomic.Int32
	probe := func(context.Context, string) (string, error) { probes.Add(1); return "hello", nil }
	for range 2 {
		got := ProbeHealth(context.Background(), llmprovider.ProviderClaude, "https://unreadable.test", unreadableToken{},
			[]string{"a", "b"}, 10, probe)
		if len(got) != 2 {
			t.Fatalf("ProbeHealth = %v; want both models", got)
		}
	}
	if n := probes.Load(); n != 4 {
		t.Errorf("probes = %d; want 4, neither listing cached", n)
	}
}
