package catalog

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/cacheref"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/limitmemo"
)

// memoOf is the memo of output caps a provider given opts reads, as it finds
// the cache: WithCache's, else the process's.
func memoOf(t *testing.T, opts ...llmprovider.Option) *limitmemo.Memo {
	t.Helper()
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderClaude, opts)
	if err != nil {
		t.Fatal(err)
	}
	return cacheref.From(st.Values()).LimitMemo()
}

// TestLimitMemo_Scope_0089 (DL38): the memo is per catalog.Cache, and keyed
// by provider, base URL, scope, model and stream or plain call: a cap kept
// under one is not read under another.
func TestLimitMemo_Scope_0089(t *testing.T) {
	one, two := memoOf(t, WithCache(new(Cache))), memoOf(t, WithCache(new(Cache)))
	process := memoOf(t)
	k := limitmemo.Key{Provider: "claude", Model: "claude-scope-0089"}
	one.Keep(k, 4096)
	if c, ok := one.Cap(k); !ok || c != 4096 {
		t.Fatalf("the first cache's memo: %d, %v; want 4096", c, ok)
	}
	if c, ok := two.Cap(k); ok {
		t.Errorf("the second cache's memo holds the first's cap %d; want one memo per cache", c)
	}
	if c, ok := process.Cap(k); ok {
		t.Errorf("the process's memo holds a cache's cap %d", c)
	}

	for _, c := range []struct {
		name      string
		kept, not limitmemo.Key
	}{
		{"two base URLs", limitmemo.Key{Provider: "ollama", BaseURL: "http://a.test:11434", Model: "qwen3"},
			limitmemo.Key{Provider: "ollama", BaseURL: "http://b.test:11434", Model: "qwen3"}},
		{"two Kilo organizations", limitmemo.Key{Provider: "kilo", Scope: "org-a", Model: "anthropic/claude-sonnet-4.5"},
			limitmemo.Key{Provider: "kilo", Scope: "org-b", Model: "anthropic/claude-sonnet-4.5"}},
		{"a streamed and a plain Gemini request", limitmemo.Key{Provider: "gemini", Model: "gemini-3-pro", Stream: true},
			limitmemo.Key{Provider: "gemini", Model: "gemini-3-pro"}},
	} {
		m := memoOf(t, WithCache(new(Cache)))
		m.Keep(c.kept, 65536)
		if got, ok := m.Cap(c.not); ok {
			t.Errorf("%s: %+v reads %+v's cap %d; want each apart", c.name, c.not, c.kept, got)
		}
		if got, ok := m.Cap(c.kept); !ok || got != 65536 {
			t.Errorf("%s: %+v: %d, %v; want 65536", c.name, c.kept, got, ok)
		}
	}
}

// TestLimitMemo_StreamedOutput_0089 (DL76): Limits.StreamedOutput is the
// output fact held to the cap the memo keeps for a streamed call, and Output
// to the cap it keeps for a plain one, each named "observed"; with no cap
// kept for either, the two are equal.
func TestLimitMemo_StreamedOutput_0089(t *testing.T) {
	ctx := context.Background()
	const model = "gemini-stream-0089"
	cache := new(Cache)
	opts := []llmprovider.Option{WithCache(cache),
		WithModelLayer(outputLayer("test", 65536))}
	read := func() Limits { return CachedLimits(ctx, llmprovider.ProviderGemini, model, opts...) }
	if l := read(); l.Output.Tokens != 65536 || l.StreamedOutput.Tokens != 65536 {
		t.Fatalf("no cap kept: output %+v, streamed %+v; want both 65536", l.Output, l.StreamedOutput)
	}
	memo := memoOf(t, WithCache(cache))
	memo.Keep(limitmemo.Key{Provider: "gemini", Model: model, Stream: true}, 32768)
	l := read()
	if l.StreamedOutput.Tokens != 32768 || l.StreamedOutput.Evidence[0].Source != "observed" {
		t.Errorf("a streamed cap of 32768: streamed %+v; want 32768 observed", l.StreamedOutput)
	}
	if l.Output.Tokens != 65536 || l.Output.Evidence[0].Source != "test" {
		t.Errorf("a streamed cap of 32768: output %+v; want the layer's 65536, the plain call unbounded", l.Output)
	}
	memo.Keep(limitmemo.Key{Provider: "gemini", Model: model}, 16384)
	l = read()
	if l.Output.Tokens != 16384 || l.Output.Evidence[0].Source != "observed" || l.StreamedOutput.Tokens != 32768 {
		t.Errorf("a plain cap of 16384 too: output %+v, streamed %+v; want 16384 and 32768, each its own", l.Output,
			l.StreamedOutput)
	}
}
