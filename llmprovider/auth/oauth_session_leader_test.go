package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// joinSignal is a joinHook that reports each join on the channel.
type joinSignal chan struct{}

func (j joinSignal) joined() { j <- struct{}{} }

// TestOAuthSession_WaiterOutlivesLeaderCancel (0020-MADR F14): the caller
// that started the shared refresh cancels; a caller waiting on it, whose own
// context is live, gets a token instead of the leader's cancellation. Since
// 0026-MADR F3 the refresh runs detached, so the leader's cancel no longer
// stops it: the waiter gets that refresh's own result, from one issuer call.
func TestOAuthSession_WaiterOutlivesLeaderCancel(t *testing.T) {
	var calls atomic.Int32
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		<-release // the refresh is held until the leader has gone
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "fresh", "refresh_token": "rotated", "expires_in": 1800,
		})
	}))
	t.Cleanup(srv.Close)
	var releaseOnce sync.Once
	let := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(let) // runs first: frees a held handler
	joined := make(joinSignal, 1)
	session := &OAuthSession{Provider: "openai", Access: "old", Refresh: "refresh",
		Expiry: time.Now().Add(-time.Minute), ClientID: "client-test", TokenURL: srv.URL, HTTPClient: srv.Client(),
		onJoin: joined}

	leaderCtx, cancel := context.WithCancel(context.Background())
	leaderDone := make(chan error, 1)
	go func() {
		_, err := session.Token(leaderCtx)
		leaderDone <- err
	}()
	for deadline := time.Now().Add(5 * time.Second); calls.Load() == 0; time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("the leader's refresh never reached the issuer")
		}
	}
	type result struct {
		tok llmprovider.Token
		err error
	}
	waiterDone := make(chan result, 1)
	go func() {
		tok, err := session.Token(context.Background())
		waiterDone <- result{tok, err}
	}()
	select { // the waiter is now waiting on the leader's refresh
	case <-joined:
	case <-time.After(5 * time.Second):
		t.Fatal("the waiter never joined the leader's refresh")
	}
	cancel()
	if err := <-leaderDone; !errors.Is(err, context.Canceled) {
		t.Errorf("leader Token = %v, want context.Canceled", err)
	}
	let()
	got := <-waiterDone
	if got.err != nil || got.tok.Value != "fresh" {
		t.Errorf("waiter Token = %q, %v; want the fresh token: only the leader was cancelled", got.tok.Value, got.err)
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("issuer calls = %d, want 1: the waiter shares the leader's refresh", n)
	}
}
