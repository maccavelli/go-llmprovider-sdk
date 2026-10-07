package auth

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestFileTokenStore_LockReleasedWhenHolderDies (0026-MADR F13): a process
// that dies holding the refresh lock leaves nothing stale behind. The next
// process takes the lock at once, not after a stale window, and well inside
// a 2 s wait.
func TestFileTokenStore_LockReleasedWhenHolderDies(t *testing.T) {
	if dir := os.Getenv("AUTH_LOCK_HOLD"); dir != "" {
		holder := &FileTokenStore{Dir: dir}
		if _, err := holder.LockRefresh(context.Background(), llmprovider.ProviderOpenAI); err != nil {
			os.Exit(3)
		}
		_, _ = os.Stdout.WriteString("held\n")
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store.wait = 2 * time.Second
	cmd := exec.Command(os.Args[0], "-test.run=^TestFileTokenStore_LockReleasedWhenHolderDies$")
	cmd.Env = append(os.Environ(), "AUTH_LOCK_HOLD="+store.Dir)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if line, err := bufio.NewReader(out).ReadString('\n'); err != nil || line != "held\n" {
		_ = cmd.Process.Kill()
		t.Fatalf("child: %q, %v; want it to hold the lock", line, err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	start := time.Now()
	unlock, err := store.LockRefresh(context.Background(), llmprovider.ProviderOpenAI)
	if err != nil {
		t.Fatalf("LockRefresh after the holder died = %v after %v; want the lock", err, time.Since(start))
	}
	unlock()
}

// TestFileTokenStore_RefreshLockExcludes: many holders, each with its own
// store value on one directory, as separate processes have, never hold the
// lock at once (0016-MADR amendment A2; 0026-MADR F13).
func TestFileTokenStore_RefreshLockExcludes(t *testing.T) {
	dir := t.TempDir()
	var holding, most atomic.Int32
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			store := &FileTokenStore{Dir: dir}
			for range 5 {
				unlock, err := store.LockRefresh(context.Background(), llmprovider.ProviderOpenAI)
				if err != nil {
					t.Error(err)
					return
				}
				n := holding.Add(1)
				for m := most.Load(); n > m && !most.CompareAndSwap(m, n); m = most.Load() {
				}
				time.Sleep(time.Millisecond)
				holding.Add(-1)
				unlock()
			}
		})
	}
	wg.Wait()
	if m := most.Load(); m != 1 {
		t.Errorf("at most %d holders at once, want 1", m)
	}
}
