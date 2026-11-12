package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestGoStringPrefix_0076: %#v names the session's own package, auth, where
// it read "&llmprovider." from before the type moved (0076-MADR D5, C25).
func TestGoStringPrefix_0076(t *testing.T) {
	session := &OAuthSession{Provider: llmprovider.ProviderGrok, Issuer: DefaultGrokOAuthIssuer, ClientID: "client-visible"}
	if got := fmt.Sprintf("%#v", session); !strings.HasPrefix(got, "&auth.OAuthSession{") {
		t.Errorf("%%#v = %s; want it to begin &auth.OAuthSession{", got)
	}
}

// pauseEndCtx is a context that ends when it is first asked for Done.
// LockRefresh reads Err between attempts and asks for Done only in the pause
// after that check, so this context is live when the check reads it and
// ends during the pause, on every run, with no clock and no seam.
type pauseEndCtx struct {
	context.Context
	mu        sync.Mutex
	done      chan struct{}
	err       error
	liveReads int // Err calls that found it live
}

func newPauseEndCtx() *pauseEndCtx {
	return &pauseEndCtx{Context: context.Background(), done: make(chan struct{})}
}

func (c *pauseEndCtx) Done() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err == nil {
		c.err = context.Canceled
		close(c.done)
	}
	return c.done
}

func (c *pauseEndCtx) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err == nil {
		c.liveReads++
	}
	return c.err
}

// TestLockRefreshContextKinds_0076: with the lock held by another handle, the
// two kinds LockRefresh's doc names for a ctx that ends while it waits
// (0076-MADR D5): seen ended between attempts, an error matching both
// ErrProviderUnavailable and ctx's (C46); ended during the pause, ctx's error
// alone (C47).
func TestLockRefreshContextKinds_0076(t *testing.T) {
	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	holder := &FileTokenStore{Dir: store.Dir}
	unlock, err := holder.LockRefresh(context.Background(), llmprovider.ProviderOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	t.Run("seen between attempts", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		release, err := store.LockRefresh(ctx, llmprovider.ProviderOpenAI)
		if err == nil {
			release()
			t.Fatal("LockRefresh took a held lock")
		}
		if !errors.Is(err, llmprovider.ErrProviderUnavailable) || !errors.Is(err, context.Canceled) {
			t.Errorf("LockRefresh = %v; want an error matching both llmprovider.ErrProviderUnavailable and context.Canceled", err)
		}
	})

	t.Run("during the pause", func(t *testing.T) {
		ctx := newPauseEndCtx()
		release, err := store.LockRefresh(ctx, llmprovider.ProviderOpenAI)
		if err == nil {
			release()
			t.Fatal("LockRefresh took a held lock")
		}
		ctx.mu.Lock()
		liveReads, ended := ctx.liveReads, ctx.err != nil
		ctx.mu.Unlock()
		if liveReads == 0 || !ended {
			t.Fatalf("the context was read live %d time(s) and ended %t; want it live at the check between attempts and ended in the pause", liveReads, ended)
		}
		if err != context.Canceled || errors.Is(err, llmprovider.ErrProviderUnavailable) { //nolint:errorlint // ctx's error alone, unwrapped
			t.Errorf("LockRefresh = %v; want context.Canceled alone, not matching llmprovider.ErrProviderUnavailable", err)
		}
	})
}
