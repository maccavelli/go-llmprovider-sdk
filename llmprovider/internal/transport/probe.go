package transport

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

// probeTimeout bounds one probe; tests shorten it.
var probeTimeout = 5 * time.Second

// ProbeGenerateHealth runs a tiny generate against each candidate and returns
// those that did not fail, preserving preferred order. At most limit
// candidates are probed; the providers pass llmprovider.MaxListedModels. A
// probe that failed, or answered wrongly, drops its model. One that timed out,
// such as a model a local server is still loading, keeps it (0020-MADR F21,
// Q3 a).
func ProbeGenerateHealth(ctx context.Context, preferred []string, limit int, generate func(ctx context.Context, modelID string) (string, error)) []string {
	if len(preferred) == 0 {
		return nil
	}
	candidates := preferred
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	type result struct {
		id    string
		ok    bool
		index int
	}
	ch := make(chan result, len(candidates))
	var wg sync.WaitGroup

	for i, id := range candidates {
		wg.Add(1)
		go func(idx int, modelID string) {
			defer wg.Done()
			tCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()
			out, err := generate(tCtx, modelID)
			ok := err == nil && strings.Contains(strings.ToLower(out), "hello")
			// The probe's own timeout, not the caller's: the model is slow,
			// not broken.
			timedOut := err != nil && errors.Is(tCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil
			ok = ok || timedOut
			ch <- result{id: modelID, ok: ok, index: idx}
		}(i, id)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	healthy := make(map[string]struct{})
	for r := range ch {
		if r.ok {
			healthy[strings.ToLower(r.id)] = struct{}{}
		}
	}

	var out []string
	for _, id := range candidates {
		if _, ok := healthy[strings.ToLower(id)]; ok {
			out = append(out, id)
		}
	}
	return out
}
