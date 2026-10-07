package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// failingStore holds one session; its first failures Saves fail, the rest
// succeed, as a disk that fills and is then cleared does.
type failingStore struct {
	mu       sync.Mutex
	held     *OAuthSession
	failures int
}

func (s *failingStore) Load(context.Context, llmprovider.ProviderID) (*OAuthSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.held == nil {
		return nil, nil
	}
	return s.held.persistable(), nil
}

func (s *failingStore) Save(_ context.Context, _ llmprovider.ProviderID, session *OAuthSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failures > 0 {
		s.failures--
		return errors.New("save failed: disk full")
	}
	s.held = session.persistable()
	return nil
}

func (s *failingStore) Delete(context.Context, llmprovider.ProviderID) error { return nil }

func (s *failingStore) refresh() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.held.Refresh
}

// rotatingIssuer rotates on every refresh and, like OpenAI's and xAI's,
// rejects a refresh token sent twice. It records each token sent, and can
// hold a request until release is closed, after it has committed the
// rotation.
type rotatingIssuer struct {
	mu       sync.Mutex
	sent     []string
	used     map[string]bool
	received chan struct{}
	release  chan struct{}
}

func newRotatingIssuer(t *testing.T, hold bool) (*httptest.Server, *rotatingIssuer) {
	t.Helper()
	ri := &rotatingIssuer{used: map[string]bool{}, received: make(chan struct{}, 8)}
	if hold {
		ri.release = make(chan struct{})
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		refresh := r.Form.Get("refresh_token")
		ri.mu.Lock()
		ri.sent = append(ri.sent, refresh)
		reused := ri.used[refresh]
		ri.used[refresh] = true
		n := len(ri.sent)
		release := ri.release
		ri.mu.Unlock()
		select { // tests that wait for a request read it; others never block the issuer
		case ri.received <- struct{}{}:
		default:
		}
		if reused {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"refresh_token_reused"}`))
			return
		}
		if release != nil {
			<-release // the rotation is committed before the reply is read
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": fmt.Sprintf("access-%d", n), "refresh_token": fmt.Sprintf("refresh-%d", n), "expires_in": 1800,
		})
	}))
	t.Cleanup(srv.Close)
	if hold {
		t.Cleanup(func() { // runs before srv.Close: frees a held handler
			ri.mu.Lock()
			defer ri.mu.Unlock()
			select {
			case <-ri.release:
			default:
				close(ri.release)
			}
		})
	}
	return srv, ri
}

func (ri *rotatingIssuer) tokens() []string {
	ri.mu.Lock()
	defer ri.mu.Unlock()
	return slices.Clone(ri.sent)
}

func (ri *rotatingIssuer) let() {
	ri.mu.Lock()
	defer ri.mu.Unlock()
	close(ri.release)
}

func expiredStoredSession(srv *httptest.Server, store TokenStore) *OAuthSession {
	return &OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "old-access", Refresh: "old-refresh",
		Expiry: time.Now().Add(-time.Minute), ClientID: "client-test", TokenURL: srv.URL, HTTPClient: srv.Client(), Store: store}
}

// TestOAuthSession_RepeatedFailedSavesKeepRotation (0026-MADR F1): however
// many saves in a row fail, the store's token is never adopted as a newer
// one, no refresh token is sent twice, and once saving works the store ends
// on the newest session. The resave case lets a current token trigger the
// pending save before the next refresh, as the audit's reproduction did.
func TestOAuthSession_RepeatedFailedSavesKeepRotation(t *testing.T) {
	for _, c := range []struct {
		name     string
		failures int
		resave   bool
	}{
		{"one failed save", 1, false},
		{"two failed saves", 2, false},
		{"three failed saves", 3, false},
		{"two failed saves, then a resave", 2, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			srv, issuer := newRotatingIssuer(t, false)
			store := &failingStore{failures: c.failures}
			session := expiredStoredSession(srv, store)
			store.held = session.persistable()

			refreshes := c.failures + 1
			for i := 1; i <= refreshes; i++ {
				if c.resave && i == refreshes {
					// The token is current, and a rotation is unsaved:
					// this Token retries the save, and refreshes nothing.
					if tok, err := session.Token(context.Background()); err != nil || tok.Value != fmt.Sprintf("access-%d", i-1) {
						t.Fatalf("resave Token = %q, %v; want the current access-%d", tok.Value, err, i-1)
					}
					if got, want := store.refresh(), fmt.Sprintf("refresh-%d", i-1); got != want {
						t.Fatalf("after the resave the store holds %q, want %q", got, want)
					}
				}
				if i > 1 {
					session.Invalidate()
				}
				if tok, err := session.Token(context.Background()); err != nil || tok.Value != fmt.Sprintf("access-%d", i) {
					t.Fatalf("refresh %d: Token = %q, %v; want access-%d", i, tok.Value, err, i)
				}
			}
			sent := issuer.tokens()
			want := []string{"old-refresh"}
			for i := 1; i < refreshes; i++ {
				want = append(want, fmt.Sprintf("refresh-%d", i))
			}
			if !slices.Equal(sent, want) {
				t.Errorf("refresh tokens sent = %v, want %v: a spent token was sent again", sent, want)
			}
			if got, want := store.refresh(), fmt.Sprintf("refresh-%d", refreshes); got != want {
				t.Errorf("store holds %q, want the newest, %q", got, want)
			}
		})
	}
}

// TestOAuthSession_CallerCancelKeepsRotation (0026-MADR F3): the caller's
// context ends after the issuer rotated and before the reply is read. The
// refresh still completes and is adopted and saved, so the next refresh
// sends the rotated token, never the spent one.
func TestOAuthSession_CallerCancelKeepsRotation(t *testing.T) {
	srv, issuer := newRotatingIssuer(t, true)
	store := &failingStore{}
	session := expiredStoredSession(srv, store)
	store.held = session.persistable()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := session.Token(ctx)
		done <- err
	}()
	<-issuer.received
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Token = %v, want context.Canceled", err)
	}
	issuer.let()
	if tok, err := session.Token(context.Background()); err != nil || tok.Value != "access-1" {
		t.Fatalf("Token after the cancelled one = %q, %v; want access-1 from the completed refresh", tok.Value, err)
	}
	session.Invalidate()
	if _, err := session.Token(context.Background()); err != nil {
		t.Fatalf("next refresh: %v", err)
	}
	if sent, want := issuer.tokens(), []string{"old-refresh", "refresh-1"}; !slices.Equal(sent, want) {
		t.Errorf("refresh tokens sent = %v, want %v", sent, want)
	}
	if got := store.refresh(); got != "refresh-2" {
		t.Errorf("store holds %q, want refresh-2", got)
	}
}

// TestOAuthSession_AbandonedWaiterDoesNotResendSpent (0026-MADR F3): the
// caller that started the refresh is cancelled while another waits on it.
// The waiter gets the refresh's own result, and no second refresh sends the
// spent token.
func TestOAuthSession_AbandonedWaiterDoesNotResendSpent(t *testing.T) {
	srv, issuer := newRotatingIssuer(t, true)
	joined := make(joinSignal, 1)
	session := expiredStoredSession(srv, nil)
	session.onJoin = joined

	leaderCtx, cancel := context.WithCancel(context.Background())
	leaderDone := make(chan error, 1)
	go func() {
		_, err := session.Token(leaderCtx)
		leaderDone <- err
	}()
	<-issuer.received
	type result struct {
		tok llmprovider.Token
		err error
	}
	waiterDone := make(chan result, 1)
	go func() {
		tok, err := session.Token(context.Background())
		waiterDone <- result{tok, err}
	}()
	select {
	case <-joined:
	case <-time.After(5 * time.Second):
		t.Fatal("the waiter never joined the refresh")
	}
	cancel()
	<-leaderDone
	issuer.let()
	got := <-waiterDone
	if got.err != nil || got.tok.Value != "access-1" {
		t.Errorf("waiter Token = %q, %v; want access-1", got.tok.Value, got.err)
	}
	if sent, want := issuer.tokens(), []string{"old-refresh"}; !slices.Equal(sent, want) {
		t.Errorf("refresh tokens sent = %v, want %v: the spent token was sent again", sent, want)
	}
}
