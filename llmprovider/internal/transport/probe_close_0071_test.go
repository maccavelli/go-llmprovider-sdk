package transport

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestProbeStoreClose_0071: Close cancels a run in flight and Wait returns
// once it has ended; a closed store starts no run, and answers as a run with
// no answer does (0071-MADR D1, D2).
func TestProbeStoreClose_0071(t *testing.T) {
	key := ProbeKey{Provider: "p", BaseURL: "http://127.0.0.1:1", Credential: "c"}

	t.Run("a run in flight is waited for", func(t *testing.T) {
		s := NewProbeStore()
		entered := make(chan struct{})
		var ended atomic.Bool
		go ProbeGenerateHealthCached(context.Background(), s, key, []string{"m"}, 1,
			func(ctx context.Context, _ string) (string, error) {
				close(entered)
				<-ctx.Done()
				time.Sleep(50 * time.Millisecond)
				ended.Store(true)
				return "", ctx.Err()
			})
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatal("the probe never started")
		}
		s.Close()
		waited := make(chan struct{})
		go func() {
			s.Wait()
			close(waited)
		}()
		select {
		case <-waited:
		case <-time.After(2 * time.Second):
			t.Fatal("Wait did not return after Close")
		}
		if !ended.Load() {
			t.Error("Wait returned before the run's probe had ended")
		}
	})

	t.Run("a closed store starts no run", func(t *testing.T) {
		s := NewProbeStore()
		s.Close()
		var calls atomic.Int32
		got := ProbeGenerateHealthCached(context.Background(), s, key, []string{"m"}, 1,
			func(context.Context, string) (string, error) {
				calls.Add(1)
				return "hello", nil
			})
		if got != nil || calls.Load() != 0 {
			t.Errorf("closed store: %v, %d probe(s); want nil and none", got, calls.Load())
		}
		s.Wait()
	})
}
