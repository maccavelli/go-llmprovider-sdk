package wizard

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// defaultsPrompter answers its scripted selects, inputs and secrets in turn,
// then accepts every default, for ever, as a --yes or headless consumer does.
// search, when set, is typed at every model search instead of a blank line.
type defaultsPrompter struct {
	selects []int
	inputs  []string
	secrets []string
	search  string
	calls   atomic.Int64
}

func (d *defaultsPrompter) Select(_ string, _ []Choice, def int) (int, error) {
	d.calls.Add(1)
	if len(d.selects) > 0 {
		v := d.selects[0]
		d.selects = d.selects[1:]
		return v, nil
	}
	return def, nil
}

func (d *defaultsPrompter) MultiSelect(_ string, _ []Choice, pre []int) ([]int, error) {
	d.calls.Add(1)
	return pre, nil
}

func (d *defaultsPrompter) Confirm(_ string, def bool) (bool, error) {
	d.calls.Add(1)
	return def, nil
}

func (d *defaultsPrompter) Input(prompt, def string) (string, error) {
	d.calls.Add(1)
	if len(d.inputs) > 0 {
		v := d.inputs[0]
		d.inputs = d.inputs[1:]
		return v, nil
	}
	if prompt == searchModelsPrompt && d.search != "" {
		return d.search, nil
	}
	return def, nil
}

func (d *defaultsPrompter) Secret(string) (string, error) {
	d.calls.Add(1)
	if len(d.secrets) > 0 {
		v := d.secrets[0]
		d.secrets = d.secrets[1:]
		return v, nil
	}
	return "", errors.New("defaultsPrompter: no secret scripted")
}

func (d *defaultsPrompter) Notify(Level, string, ...any) { d.calls.Add(1) }

// TestConfigure_EmptyRecommendedHonoursCtx (0026-MADR F9): a prompter that
// accepts defaults never makes ConfigureLLM loop. With a listing that
// recommends nothing, the menu offers the current model, if any, and Other,
// so the run ends at once. A search that never matches, answered by its
// default, Search again, stops when ctx ends (R40).
func TestConfigure_EmptyRecommendedHonoursCtx(t *testing.T) {
	listing := `{"data":[` + kiloProfileEntry("kilo-auto/small", "-1", "-1", 0, 0) + "," +
		kiloProfileEntry("kilo-auto/balanced", "-1", "-1", 0, 0) + `]}`
	for _, c := range []struct {
		name, search string
		wantCtx      bool
	}{
		{"defaults on an empty recommendation", "", false},
		{"a search that never matches", "no-such-model-zz", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			srv := zenServer(t, http.StatusOK, listing)
			p := &defaultsPrompter{selects: []int{providerIdx(t, llmprovider.ProviderKilo)},
				inputs: []string{srv.URL}, secrets: []string{testKey}, search: c.search}
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			done := make(chan error, 1)
			go func() {
				_, err := ConfigureLLM(ctx, p, zenOptions())
				done <- err
			}()
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("ConfigureLLM chose a model no one gave")
				}
				if c.wantCtx && !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("ConfigureLLM = %v; want context.DeadlineExceeded", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatalf("ctx expired 1.7s ago; ConfigureLLM still running after %d prompter calls", p.calls.Load())
			}
		})
	}
}

// TestConfigure_EmptyRecommendedSearchStillWorks (0026-MADR F9, 0021-MADR
// C13): with nothing recommended, a query typed at the search prompt still
// searches the listing.
func TestConfigure_EmptyRecommendedSearchStillWorks(t *testing.T) {
	listing := `{"data":[` + kiloProfileEntry("kilo-auto/small", "-1", "-1", 0, 0) + "," +
		kiloProfileEntry("kilo-auto/balanced", "-1", "-1", 0, 0) + `]}`
	srv := zenServer(t, http.StatusOK, listing)
	f := newFake(t, fakePrompter{
		selects: []int{providerIdx(t, llmprovider.ProviderKilo), 0},
		inputs:  []string{srv.URL, "small"}, secrets: []string{testKey},
	})
	res, err := ConfigureLLM(context.Background(), f, zenOptions())
	if err != nil || res.Model != "kilo-auto/small" {
		t.Fatalf("ConfigureLLM = %+v, %v; want kilo-auto/small, found by search", res, err)
	}
}
