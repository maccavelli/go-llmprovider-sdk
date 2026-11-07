package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"
)

// 0071-PLAN-give-a-cache-a-lifetime-and-wait-for-a-refresh.md Phase 2:
// OAuthSession.Wait (0071-MADR D3).

// startAbandonedRefresh starts session's refresh under a caller that is
// cancelled once the issuer holds the request, so the refresh runs on
// detached, as 0026-MADR F3 has it, with no caller left.
func startAbandonedRefresh(t *testing.T, session *OAuthSession, issuer *rotatingIssuer) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := session.Token(ctx)
		done <- err
	}()
	select {
	case <-issuer.received:
	case <-time.After(5 * time.Second):
		t.Fatal("the refresh never reached the issuer")
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Token = %v, want context.Canceled", err)
	}
}

// TestOAuthSessionWait_AfterSave_0071: Wait returns nil only once the
// detached refresh has saved the rotated session.
func TestOAuthSessionWait_AfterSave_0071(t *testing.T) {
	srv, issuer := newRotatingIssuer(t, true)
	store := &failingStore{}
	session := expiredStoredSession(srv, store)
	store.held = session.persistable()
	startAbandonedRefresh(t, session, issuer)

	type result struct {
		err    error
		stored string
	}
	waited := make(chan result, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := session.Wait(ctx)
		waited <- result{err, store.refresh()}
	}()
	issuer.let()
	r := <-waited
	if r.err != nil || r.stored != "refresh-1" {
		t.Errorf("Wait = %v, with the store holding %q; want nil once refresh-1 is saved", r.err, r.stored)
	}
}

// TestOAuthSessionWait_Deadline_0071: while the issuer still holds the
// refresh, Wait returns ctx's error at its deadline.
func TestOAuthSessionWait_Deadline_0071(t *testing.T) {
	srv, issuer := newRotatingIssuer(t, true)
	session := expiredStoredSession(srv, nil)
	startAbandonedRefresh(t, session, issuer)

	waited := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		waited <- session.Wait(ctx)
	}()
	select {
	case err := <-waited:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Wait with the refresh held = %v, want context.DeadlineExceeded", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not return at its 100ms deadline")
	}
}

// TestOAuthSessionWait_Idle_0071: a session with nothing in flight returns
// nil at once, even under a context already done.
func TestOAuthSessionWait_Idle_0071(t *testing.T) {
	session := &OAuthSession{Access: "a", Refresh: "r", Expiry: time.Now().Add(time.Hour)}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := session.Wait(ctx); err != nil {
		t.Errorf("Wait on an idle session = %v, want nil", err)
	}
}

// TestOAuthSessionWait_TempDir_0071: a refresh into a FileTokenStore in a
// test's temporary directory, still running when the test body ends, is
// waited for before the directory is removed. The removal then finds no
// file open, which on Windows it otherwise could (issue 12), and the store
// holds the rotated session.
func TestOAuthSessionWait_TempDir_0071(t *testing.T) {
	srv, issuer := newRotatingIssuer(t, true)
	dir := t.TempDir()
	store, err := NewFileTokenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	session := expiredStoredSession(srv, store)
	if err := store.Save(context.Background(), session.Provider, session.persistable()); err != nil {
		t.Fatal(err)
	}
	// Registered after t.TempDir, so it runs before the directory's removal.
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := session.Wait(ctx); err != nil {
			t.Errorf("Wait = %v, want nil", err)
		}
		saved, err := store.Load(context.Background(), session.Provider)
		if err != nil || saved == nil || saved.Refresh != "refresh-1" {
			t.Errorf("after Wait the store holds %v, %v; want the rotated refresh-1", saved, err)
		}
	})
	startAbandonedRefresh(t, session, issuer)
	issuer.let()
}

// TestOAuthSessionWait_WindowsRemoval_0071: on Windows, a refresh that
// holds a FileTokenStore's refresh lock after its test ends blocks the
// removal of the test's temporary directory, as issue 12 found. Waiting for
// the refresh in a cleanup that runs before the removal lets the removal
// succeed. The issuer answers after 3 s, past the 2 s for which testing
// retries a Windows removal (0071-MADR, amendment "a Windows test of the
// removal").
func TestOAuthSessionWait_WindowsRemoval_0071(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("only Windows refuses to remove a file held open; elsewhere this test cannot fail")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(3 * time.Second)
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access-1", "refresh_token": "refresh-1",
			"expires_in": 1800})
	}))
	t.Cleanup(srv.Close)
	dir := t.TempDir()
	store, err := NewFileTokenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	session := expiredStoredSession(srv, store)
	if err := store.Save(context.Background(), session.Provider, session.persistable()); err != nil {
		t.Fatal(err)
	}
	// Registered after t.TempDir, so it runs before the directory's removal.
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := session.Wait(ctx); err != nil {
			t.Errorf("Wait = %v, want nil", err)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if _, err := session.Token(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Token under a 200ms deadline = %v, want context.DeadlineExceeded", err)
	}
}
