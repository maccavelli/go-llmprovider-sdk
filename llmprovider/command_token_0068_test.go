package llmprovider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"
)

// 0068-MADR D4: a command token remembers a failed run for 5 s, and starts
// one run early, in the background, once its output is four fifths of its
// TTL old.

// TestHelperKeyCommand_0068 is not a test: run as a child by failArgv, it
// appends a line to the file its argument names and exits 3, a key command
// that fails and counts its runs. A plain test run has no "--", so it skips.
func TestHelperKeyCommand_0068(t *testing.T) {
	at := slices.Index(os.Args, "--")
	if at < 0 || at+1 >= len(os.Args) {
		t.Skip("helper process only")
	}
	f, err := os.OpenFile(os.Args[at+1], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		os.Exit(9)
	}
	_, _ = f.WriteString("run\n")
	_ = f.Close()
	os.Exit(3)
}

// failArgv is a key command that fails, counting its runs in path.
func failArgv(path string) []string {
	return []string{os.Args[0], "-test.run=^TestHelperKeyCommand_0068$", "--", path}
}

// clockAt is a settable clock for a CommandToken.
type clockAt struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clockAt) get() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clockAt) add(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

// TestCommandToken_RemembersFailure_0068: a failing command runs once for two
// calls inside 5 s, again after them, and again after Invalidate.
func TestCommandToken_RemembersFailure_0068(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "runs")
	clock := &clockAt{now: time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)}
	c := &CommandToken{Argv: failArgv(path), now: clock.get}
	ctx := context.Background()
	for i := range 2 {
		if _, err := c.Token(ctx); !errors.Is(err, ErrAuthFailure) {
			t.Fatalf("call %d: Token = %v; want ErrAuthFailure", i+1, err)
		}
	}
	if n := runs(t, path); n != 1 {
		t.Fatalf("%d runs for two calls inside 5 s; want 1", n)
	}
	clock.add(6 * time.Second)
	if _, err := c.Token(ctx); !errors.Is(err, ErrAuthFailure) || runs(t, path) != 2 {
		t.Fatalf("after 6 s: Token = %v after %d runs; want ErrAuthFailure from a second run", err, runs(t, path))
	}
	c.Invalidate()
	if _, err := c.Token(ctx); !errors.Is(err, ErrAuthFailure) || runs(t, path) != 3 {
		t.Fatalf("after Invalidate: Token = %v after %d runs; want ErrAuthFailure from a third run", err, runs(t, path))
	}
}

// TestCommandToken_AbandonedNotRemembered_0068: a run that ended because its
// caller gave up is not remembered: the next caller runs the command.
func TestCommandToken_AbandonedNotRemembered_0068(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "runs")
	c := NewCommandToken(helperArgv(t, "slowcount", path)...)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := c.Token(ctx); err == nil {
		t.Fatal("Token succeeded; want the caller's deadline to end the run")
	}
	if tok, err := c.Token(context.Background()); err != nil || tok.Value == "" {
		t.Fatalf("next Token = %q, %v; want a run, not the abandoned one's error", tok.Value, err)
	}
}

// TestCommandToken_EarlyRun_0068: past four fifths of its TTL, Token returns
// the output at once and starts one run, which outlives its caller's context
// and replaces the output; a second caller meanwhile starts none.
func TestCommandToken_EarlyRun_0068(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "runs")
	clock := &clockAt{now: time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)}
	c := &CommandToken{Argv: helperArgv(t, "slowcount", path), TTL: 10 * time.Second, now: clock.get}
	if tok, err := c.Token(context.Background()); err != nil || tok.Value != "key-1" {
		t.Fatalf("Token = %q, %v; want key-1", tok.Value, err)
	}
	clock.add(9 * time.Second)

	// slowcount counts a run only after half a second, so a Token that waited
	// for the run would return after the count.
	ctx, cancel := context.WithCancel(context.Background())
	tok, err := c.Token(ctx)
	cancel()
	if n := runs(t, path); err != nil || tok.Value != "key-1" || n != 1 {
		t.Fatalf("Token = %q, %v with %d runs counted; want key-1 before the early run ends", tok.Value, err, n)
	}
	inflight := func() *tokenFuture {
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.inflight
	}
	early := inflight()
	if early == nil {
		t.Fatal("no run in flight after Token past four fifths of the TTL")
	}
	if tok, err := c.Token(context.Background()); err != nil || tok.Value != "key-1" {
		t.Fatalf("second caller: Token = %q, %v; want key-1", tok.Value, err)
	}
	if inflight() != early {
		t.Error("the second caller started a run of its own; want it to leave the early run alone")
	}
	waitForInflight(t, func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.inflight == nil && c.value == "key-2"
	})
	if n := runs(t, path); n != 2 {
		t.Errorf("%d runs; want 2: the first, and one early run", n)
	}
	if tok, err := c.Token(context.Background()); err != nil || tok.Value != "key-2" {
		t.Errorf("after the early run: Token = %q, %v; want key-2", tok.Value, err)
	}
}
