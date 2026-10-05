package llmprovider

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// tokenResult is one Token call's outcome.
type tokenResult struct {
	tok Token
	err error
}

// TestCommandToken_WaiterOutlivesLeaderCancel (0020-MADR F14): the caller
// that started the shared run cancels it; a caller waiting on that run, whose
// own context is live, gets a token instead of the leader's cancellation.
func TestCommandToken_WaiterOutlivesLeaderCancel(t *testing.T) {
	t.Parallel()
	c := NewCommandToken(helperArgv(t, "slowcount", filepath.Join(t.TempDir(), "runs"))...)
	joined := make(chan struct{}, 1)
	c.onJoin = func() { joined <- struct{}{} }
	leaderCtx, cancel := context.WithCancel(context.Background())
	leaderDone := make(chan error, 1)
	go func() {
		_, err := c.Token(leaderCtx)
		leaderDone <- err
	}()
	waitForInflight(t, func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.inflight != nil
	})
	waiterDone := make(chan tokenResult, 1)
	go func() {
		tok, err := c.Token(context.Background())
		waiterDone <- tokenResult{tok, err}
	}()
	select { // the waiter is now waiting on the leader's run
	case <-joined:
	case <-time.After(5 * time.Second):
		t.Fatal("the waiter never joined the leader's run")
	}
	cancel()
	if err := <-leaderDone; err == nil {
		t.Log("the leader's run finished before its cancellation; nothing to show")
	}
	got := <-waiterDone
	if got.err != nil || got.tok.Value == "" {
		t.Errorf("waiter Token = %q, %v; want a token: only the leader was cancelled", got.tok.Value, got.err)
	}
}

// waitForInflight polls until a shared fetch has started.
func waitForInflight(t *testing.T, started func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if started() {
			return
		}
	}
	t.Fatal("the shared fetch never started")
}
