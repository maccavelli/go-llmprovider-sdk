package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestOAuthSession_WaiterOutlivesLeaderCancel (0020-MADR F14): the caller
// that started the shared refresh cancels it; a caller waiting on it, whose
// own context is live, gets a token instead of the leader's cancellation.
func TestOAuthSession_WaiterOutlivesLeaderCancel(t *testing.T) {
	var calls atomic.Int32
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			select { // the leader's request hangs until it is cancelled
			case <-r.Context().Done():
			case <-release:
			}
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "fresh", "refresh_token": "rotated", "expires_in": 1800,
		})
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) }) // runs first: frees a hung handler
	session := &OAuthSession{Provider: "openai", Access: "old", Refresh: "refresh",
		Expiry: time.Now().Add(-time.Minute), ClientID: "client-test", TokenURL: srv.URL, HTTPClient: srv.Client()}

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
	time.Sleep(50 * time.Millisecond) // the waiter is now waiting on the leader's refresh
	cancel()
	<-leaderDone
	got := <-waiterDone
	if got.err != nil || got.tok.Value != "fresh" {
		t.Errorf("waiter Token = %q, %v; want the fresh token: only the leader was cancelled", got.tok.Value, got.err)
	}
}
