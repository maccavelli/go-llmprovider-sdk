package llmprovider_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"
)

// 0048-MADR D3: what the docs of Factory, Settings and Option say of
// concurrent use, run under -race.

// TestFactory_RunsConcurrently_0048: Registry.New does not serialise a
// factory: two New calls for one id run it at once.
func TestFactory_RunsConcurrently_0048(t *testing.T) {
	const id llmprovider.ProviderID = "concurrent-0048"
	var in atomic.Int32
	both := make(chan struct{})
	var once sync.Once
	var r llmprovider.Registry
	err := r.Register(llmprovider.Descriptor{ID: id, Label: "Concurrent"}, func(...llmprovider.Option) (llmprovider.Provider, error) {
		defer in.Add(-1)
		if in.Add(1) == 2 {
			once.Do(func() { close(both) })
		}
		select {
		case <-both:
		case <-time.After(2 * time.Second):
		}
		return llmtest.NewFake(id, llmprovider.Capabilities{}), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			if _, err := r.New(id); err != nil {
				t.Errorf("New: %v", err)
			}
		})
	}
	wg.Wait()
	select {
	case <-both:
	default:
		t.Error("the two New calls never ran the factory at once")
	}
}

// sharedOptions0048 is one list of options, a For and a scoped option among
// them, given to every concurrent build.
func sharedOptions0048() []llmprovider.Option {
	return []llmprovider.Option{
		llmprovider.WithModel("m-0048"),
		llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: llmprovider.EffortLow}),
		llmprovider.WithBaseURL("https://example.test/v1"),
		llmprovider.ScopedOptionFor([]llmprovider.ProviderID{llmprovider.ProviderClaude, llmprovider.ProviderKilo}, "test.scoped0048", 7),
		llmprovider.For(llmprovider.ProviderClaude, llmprovider.WithMaxTokens(123)),
	}
}

// TestOption_SharedByConcurrentBuilds_0048: one list of options resolves at
// once for 16 builds, each getting the same settings: applying an option does
// not change it.
func TestOption_SharedByConcurrentBuilds_0048(t *testing.T) {
	opts := sharedOptions0048()
	var wg sync.WaitGroup
	for i := range 16 {
		id := llmprovider.ProviderClaude
		if i%2 == 1 {
			id = llmprovider.ProviderKilo
		}
		wg.Go(func() {
			st, err := llmprovider.ResolveOptions(id, opts)
			if err != nil {
				t.Errorf("%s: ResolveOptions: %v", id, err)
				return
			}
			want := 8192
			if id == llmprovider.ProviderClaude {
				want = 123
			}
			if st.Model() != "m-0048" || st.MaxTokens() != want || len(st.Values()) != 1 || st.Values()[0] != 7 {
				t.Errorf("%s: model %q, max tokens %d, values %v; want m-0048, %d, [7]", id, st.Model(), st.MaxTokens(), st.Values(), want)
			}
		})
	}
	wg.Wait()
}

// TestSettings_ConcurrentReads_0048: one Settings read by 16 goroutines at
// once gives each the same values.
func TestSettings_ConcurrentReads_0048(t *testing.T) {
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderClaude, sharedOptions0048())
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			r := st.Reasoning()
			if st.Model() != "m-0048" || st.BaseURL() != "https://example.test/v1" || st.MaxTokens() != 123 ||
				r == nil || r.Effort != llmprovider.EffortLow || st.SessionID() == "" || st.UserAgent() == "" ||
				len(st.Values()) != 1 || st.HTTPClient() == nil || !st.ModelProbes() {
				t.Error("a concurrent read of Settings gave another value")
			}
			r.Effort = llmprovider.EffortHigh // a copy: no other reader sees it
		})
	}
	wg.Wait()
	if st.Reasoning().Effort != llmprovider.EffortLow {
		t.Error("changing Reasoning's copy changed the Settings")
	}
}
