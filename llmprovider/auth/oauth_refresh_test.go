package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// refreshServer answers the n-th refresh request (from 1) with reply(n) and
// counts the requests.
func refreshServer(t *testing.T, reply func(n int32, w http.ResponseWriter, r *http.Request)) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reply(calls.Add(1), w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func refreshOK(w http.ResponseWriter, access string) {
	_ = json.NewEncoder(w).Encode(map[string]any{"access_token": access, "refresh_token": "rt-new", "expires_in": 3600})
}

func expiredSession(srv *httptest.Server, issuer string) *OAuthSession {
	return &OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "a-old", Refresh: "rt-old", Expiry: time.Now().Add(-time.Minute),
		Issuer: issuer, ClientID: "client-test", TokenURL: srv.URL, HTTPClient: srv.Client()}
}

// TestOAuthRefresh_ReloadsStoreFirst: a store another process already
// refreshed is adopted, and the stale refresh token is never sent.
func TestOAuthRefresh_ReloadsStoreFirst(t *testing.T) {
	srv, calls := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) { refreshOK(w, "a-refreshed") })
	store, err := NewFileTokenStore(filepath.Join(t.TempDir(), "tokens"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), llmprovider.ProviderOpenAI, &OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "a-stored",
		Refresh: "rt-stored", Expiry: time.Now().Add(time.Hour), ClientID: "client-test", TokenURL: srv.URL}); err != nil {
		t.Fatal(err)
	}
	session := expiredSession(srv, "")
	session.Store = store
	tok, err := session.Token(context.Background())
	if err != nil || tok.Value != "a-stored" || calls.Load() != 0 || session.Refresh != "rt-stored" {
		t.Fatalf("Token = %q, %v after %d refreshes (refresh %q); want the stored session and none",
			tok.Value, err, calls.Load(), session.Refresh)
	}
}

// TestOAuthRefresh_StoredTokenIsTheOneRefreshed: when the stored session is
// due too, the refresh sends the stored refresh token, never the stale one.
func TestOAuthRefresh_StoredTokenIsTheOneRefreshed(t *testing.T) {
	var sent map[string]string
	srv, calls := refreshServer(t, func(_ int32, w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&sent)
		refreshOK(w, "a-refreshed")
	})
	store, err := NewFileTokenStore(filepath.Join(t.TempDir(), "tokens"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), llmprovider.ProviderOpenAI, &OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "a-stored",
		Refresh: "rt-stored", Expiry: time.Now().Add(-time.Minute), Issuer: DefaultOpenAIIssuer, ClientID: "client-test",
		TokenURL: srv.URL}); err != nil {
		t.Fatal(err)
	}
	session := expiredSession(srv, DefaultOpenAIIssuer)
	session.Store = store
	tok, err := session.Token(context.Background())
	if err != nil || tok.Value != "a-refreshed" || calls.Load() != 1 || sent["refresh_token"] != "rt-stored" {
		t.Fatalf("Token = %q, %v after %d requests sending %q; want one refresh of rt-stored",
			tok.Value, err, calls.Load(), sent["refresh_token"])
	}
}

// TestOAuthRefresh_SameStoredTokenStillRefreshes: a store holding the same
// refresh token changes nothing; the session refreshes.
func TestOAuthRefresh_SameStoredTokenStillRefreshes(t *testing.T) {
	srv, calls := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) { refreshOK(w, "a-refreshed") })
	store, err := NewFileTokenStore(filepath.Join(t.TempDir(), "tokens"))
	if err != nil {
		t.Fatal(err)
	}
	session := expiredSession(srv, "")
	if err := store.Save(context.Background(), llmprovider.ProviderOpenAI, session); err != nil {
		t.Fatal(err)
	}
	session.Store = store
	tok, err := session.Token(context.Background())
	if err != nil || tok.Value != "a-refreshed" || calls.Load() != 1 {
		t.Fatalf("Token = %q, %v after %d refreshes; want a-refreshed after 1", tok.Value, err, calls.Load())
	}
}

// TestOAuthRefresh_OpenAISendsJSON: the OpenAI issuer's refresh is a JSON
// body, as Codex sends it.
func TestOAuthRefresh_OpenAISendsJSON(t *testing.T) {
	var contentType string
	var body map[string]string
	srv, _ := refreshServer(t, func(_ int32, w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&body)
		refreshOK(w, "a-new")
	})
	if _, err := expiredSession(srv, DefaultOpenAIIssuer).Token(context.Background()); err != nil {
		t.Fatalf("Token: %v", err)
	}
	if contentType != "application/json" || body["grant_type"] != "refresh_token" ||
		body["refresh_token"] != "rt-old" || body["client_id"] != "client-test" {
		t.Fatalf("Content-Type %q, body %v; want the JSON refresh grant", contentType, body)
	}
}

// TestOAuthRefresh_GrokStaysForm: the Grok issuer's refresh stays
// form-encoded, as the Grok CLI sends it.
func TestOAuthRefresh_GrokStaysForm(t *testing.T) {
	var contentType, grant string
	srv, _ := refreshServer(t, func(_ int32, w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		if err := r.ParseForm(); err == nil {
			grant = r.PostForm.Get("grant_type")
		}
		refreshOK(w, "a-new")
	})
	if _, err := expiredSession(srv, DefaultGrokOAuthIssuer).Token(context.Background()); err != nil {
		t.Fatalf("Token: %v", err)
	}
	if contentType != "application/x-www-form-urlencoded" || grant != "refresh_token" {
		t.Fatalf("Content-Type %q, grant %q; want the form grant", contentType, grant)
	}
}

// TestOAuthRefresh_TerminalFailuresAreAuthFailure: a dead refresh token is
// ErrAuthFailure after one request, so no retry loop resends it.
func TestOAuthRefresh_TerminalFailuresAreAuthFailure(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
	}{
		"invalid_grant":             {http.StatusBadRequest, `{"error":"invalid_grant"}`},
		"invalid_client":            {http.StatusBadRequest, `{"error":"invalid_client"}`},
		"refresh_token_expired":     {http.StatusBadRequest, `{"error":{"code":"refresh_token_expired"}}`},
		"refresh_token_reused":      {http.StatusBadRequest, `{"error":{"code":"refresh_token_reused"}}`},
		"refresh_token_invalidated": {http.StatusBadRequest, `{"error":"refresh_token_invalidated"}`},
		"401":                       {http.StatusUnauthorized, `{"error":"unauthorized"}`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv, calls := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			})
			_, err := expiredSession(srv, DefaultOpenAIIssuer).Token(context.Background())
			if !errors.Is(err, llmprovider.ErrAuthFailure) || calls.Load() != 1 {
				t.Fatalf("err = %v after %d requests, want ErrAuthFailure after 1", err, calls.Load())
			}
		})
	}
}

// TestOAuthRefresh_RetriesTransientFailures: 503 twice, then success.
func TestOAuthRefresh_RetriesTransientFailures(t *testing.T) {
	srv, calls := refreshServer(t, func(n int32, w http.ResponseWriter, _ *http.Request) {
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		refreshOK(w, "a-third")
	})
	tok, err := expiredSession(srv, DefaultOpenAIIssuer).Token(context.Background())
	if err != nil || tok.Value != "a-third" || calls.Load() != 3 {
		t.Fatalf("Token = %q, %v after %d requests; want a-third after 3", tok.Value, err, calls.Load())
	}
}

// TestOAuthRefresh_BadRequestIsNotRetried: a 400 that is not a known
// terminal code fails after one request and is not an ErrAuthFailure.
func TestOAuthRefresh_BadRequestIsNotRetried(t *testing.T) {
	srv, calls := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"invalid_request"}`)
	})
	_, err := expiredSession(srv, DefaultOpenAIIssuer).Token(context.Background())
	if err == nil || errors.Is(err, llmprovider.ErrAuthFailure) || calls.Load() != 1 || !strings.Contains(err.Error(), "invalid_request") {
		t.Fatalf("err = %v after %d requests, want one plain failure", err, calls.Load())
	}
}

// TestOAuthRefresh_ExpiryFromJWT: the access token's exp wins over
// expires_in, on refresh and on login.
func TestOAuthRefresh_ExpiryFromJWT(t *testing.T) {
	exp := time.Now().Add(10 * time.Minute).Truncate(time.Second)
	access := openAITestJWT(t, map[string]any{"exp": exp.Unix()})
	srv, _ := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) { refreshOK(w, access) })
	session := expiredSession(srv, DefaultOpenAIIssuer)
	tok, err := session.Token(context.Background())
	if err != nil || !tok.Expiry.Equal(exp) {
		t.Fatalf("refresh Expiry = %v, %v; want %v", tok.Expiry, err, exp)
	}
	login, err := oauthSessionFromResponse(oauthFlowConfig{provider: llmprovider.ProviderOpenAI, issuer: DefaultOpenAIIssuer, now: time.Now},
		srv.URL, oauthTokenResponse{AccessToken: access, ExpiresIn: 3600})
	if err != nil || !login.Expiry.Equal(exp) {
		t.Fatalf("login Expiry = %v, %v; want %v", login.Expiry, err, exp)
	}
}

// TestOAuthRefresh_FiveMinuteWindow: a token with four minutes left is
// refreshed; one with six is not.
func TestOAuthRefresh_FiveMinuteWindow(t *testing.T) {
	srv, calls := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) { refreshOK(w, "a-new") })
	session := expiredSession(srv, "")
	session.Expiry = time.Now().Add(6 * time.Minute)
	if tok, err := session.Token(context.Background()); err != nil || tok.Value != "a-old" {
		t.Fatalf("six minutes left: Token = %q, %v; want a-old", tok.Value, err)
	}
	session.Expiry = time.Now().Add(4 * time.Minute)
	if tok, err := session.Token(context.Background()); err != nil || tok.Value != "a-new" || calls.Load() != 1 {
		t.Fatalf("four minutes left: Token = %q, %v after %d; want a-new after 1", tok.Value, err, calls.Load())
	}
}

// TestOAuthSession_FailedEarlyRefreshKeepsToken (0021-MADR T2): a token with
// four minutes left whose early refresh gets a 503 is still returned, the
// failure is logged, and a second call within 10 s sends nothing.
func TestOAuthSession_FailedEarlyRefreshKeepsToken(t *testing.T) {
	srv, calls := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	session := expiredSession(srv, "")
	session.Expiry = time.Now().Add(4 * time.Minute)
	var logged bytes.Buffer
	session.Logger = slog.New(slog.NewTextHandler(&logged, nil))
	for call := 1; call <= 2; call++ {
		if tok, err := session.Token(context.Background()); err != nil || tok.Value != "a-old" {
			t.Fatalf("call %d: Token = %q, %v; want the current token", call, tok.Value, err)
		}
	}
	if n := calls.Load(); n != 3 {
		t.Errorf("refresh requests = %d, want 3: the first call's attempts, and none from the second", n)
	}
	if !strings.Contains(logged.String(), "level=WARN") {
		t.Errorf("log = %q, want the failed early refresh at Warn", logged.String())
	}
}

// TestOAuthSession_RefreshesAgainAfterBackoff (0021-MADR T2): once the 10-30 s
// wait after a failed early refresh has passed, the next call refreshes
// again.
func TestOAuthSession_RefreshesAgainAfterBackoff(t *testing.T) {
	srv, calls := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	clock := &stepClock{now: time.Now()}
	session := expiredSession(srv, "")
	session.now = clock
	session.Expiry = clock.now.Add(4 * time.Minute)
	if tok, err := session.Token(context.Background()); err != nil || tok.Value != "a-old" || calls.Load() != 3 {
		t.Fatalf("Token = %q, %v after %d requests; want the current token after 3", tok.Value, err, calls.Load())
	}
	clock.now = clock.now.Add(31 * time.Second)
	if tok, err := session.Token(context.Background()); err != nil || tok.Value != "a-old" || calls.Load() != 6 {
		t.Fatalf("31 s later: Token = %q, %v after %d requests; want the current token after 6", tok.Value, err, calls.Load())
	}
}

// stepClock is a session clock that a test moves by hand.
type stepClock struct{ now time.Time }

func (c *stepClock) Now() time.Time { return c.now }

// TestOAuthSession_AuthFailureStillFails (0021-MADR T2): an early refresh the
// issuer refuses with a 401 is still an ErrAuthFailure, with no token, though
// the current token has minutes left.
func TestOAuthSession_AuthFailureStillFails(t *testing.T) {
	srv, _ := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	})
	session := expiredSession(srv, "")
	session.Expiry = time.Now().Add(4 * time.Minute)
	if tok, err := session.Token(context.Background()); !errors.Is(err, llmprovider.ErrAuthFailure) || tok.Value != "" {
		t.Fatalf("Token = %q, %v; want no token and ErrAuthFailure", tok.Value, err)
	}
}

// TestOAuthSession_ShortLivedTokenNotRefreshedEachCall (0021-MADR T3): a token
// that lives 240 s is due at half its life, not five minutes before expiry:
// ten calls on a fresh one refresh nothing.
func TestOAuthSession_ShortLivedTokenNotRefreshedEachCall(t *testing.T) {
	now := time.Now()
	access := openAITestJWT(t, map[string]any{"iat": now.Unix(), "exp": now.Add(240 * time.Second).Unix()})
	srv, calls := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) { refreshOK(w, access) })
	session := expiredSession(srv, "")
	session.Access, session.Expiry = access, now.Add(240*time.Second).Truncate(time.Second)
	for range 10 {
		if tok, err := session.Token(context.Background()); err != nil || tok.Value != access {
			t.Fatalf("Token = %q, %v; want the current token", tok.Value, err)
		}
	}
	if n := calls.Load(); n != 0 {
		t.Errorf("10 calls made %d refreshes, want 0", n)
	}
}

// TestOAuthSession_RefreshAttemptBounded (0021-MADR T8): an issuer that
// stalls its reply body ends each attempt at oauthRefreshAttemptTimeout, after
// at most 3 attempts, and the refresh lock is released.
func TestOAuthSession_RefreshAttemptBounded(t *testing.T) {
	old := oauthRefreshAttemptTimeout
	oauthRefreshAttemptTimeout = 50 * time.Millisecond
	t.Cleanup(func() { oauthRefreshAttemptTimeout = old })
	srv, calls := refreshServer(t, func(_ int32, w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"access_token":`)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	})
	store, session, _ := sharedStoreSessions(t, srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	_, err := session.Token(ctx)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("Token succeeded on a stalled reply")
	}
	if elapsed > 2*time.Second || calls.Load() > 3 {
		t.Errorf("Token = %v after %v and %d attempts; want each attempt ended at 50 ms, at most 3", err, elapsed, calls.Load())
	}
	done, cancelDone := context.WithCancel(context.Background())
	cancelDone() // a context already done tries the lock once
	unlock, lockErr := (&FileTokenStore{Dir: store.Dir}).LockRefresh(done, llmprovider.ProviderOpenAI)
	if lockErr != nil {
		t.Errorf("the refresh lock is still held: %v", lockErr)
	} else {
		unlock()
	}
}
