package llmprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestValidateOAuthSession_RejectsFixture(t *testing.T) {
	t.Parallel()

	accessOnly := func(expiry time.Time) *OAuthSession {
		return &OAuthSession{Access: "eyJhbGciOiJub25lIn0.e30.x", Issuer: DefaultOpenAIIssuer,
			ClientID: DefaultOpenAIClientID, Expiry: expiry}
	}
	for _, test := range []struct {
		name    string
		session *OAuthSession
		valid   bool
	}{
		{name: "nil", session: nil},
		{name: "empty access", session: &OAuthSession{Refresh: "refresh", ClientID: "client"}},
		{name: "chatgpt-access fixture", session: &OAuthSession{Access: "chatgpt-access", Issuer: DefaultOpenAIIssuer}},
		{name: "chatgpt-access fixture with refresh", session: &OAuthSession{Access: "chatgpt-access", Refresh: "refresh",
			Issuer: DefaultOpenAIIssuer, ClientID: DefaultOpenAIClientID}},
		{name: "no refresh outside ChatGPT", session: &OAuthSession{Access: "access", Issuer: DefaultGrokOAuthIssuer,
			ClientID: DefaultOpenAIClientID}},
		{name: "refreshable with token URL", session: &OAuthSession{Access: "access", Refresh: "refresh",
			ClientID: "client", TokenURL: "https://issuer.test/token"}, valid: true},
		{name: "refreshable with derived token URL", session: &OAuthSession{Access: "access", Refresh: "refresh",
			ClientID: DefaultGrokOAuthClientID, Issuer: DefaultGrokOAuthIssuer}, valid: true},
		{name: "refreshable without client id", session: &OAuthSession{Access: "access", Refresh: "refresh",
			TokenURL: "https://issuer.test/token"}},
		{name: "ChatGPT access-only", session: accessOnly(time.Time{}), valid: true},
		{name: "ChatGPT access-only with expiry", session: accessOnly(time.Now().Add(time.Hour))},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateOAuthSession(test.session)
			if (err == nil) != test.valid {
				t.Fatalf("ValidateOAuthSession() = %v, want valid=%t", err, test.valid)
			}
		})
	}
}

// jwtShapedDescription gives the redactor a token to remove from an error body.
const jwtShapedDescription = "eyJhbGciOiJub25lIn0.aaa.bbb"

func TestExchangeOAuthCode_IncludesRedactedBody(t *testing.T) {
	t.Parallel()
	srv := tokenErrorServer(t, `{"error":"invalid_grant","error_description":"`+jwtShapedDescription+`"}`)
	_, err := exchangeOAuthCode(context.Background(),
		oauthFlowConfig{provider: ProviderGrok, clientID: "test-client", httpClient: srv.Client()},
		oauthEndpoints{Token: srv.URL}, "code", "http://127.0.0.1/callback", "verifier", "")
	assertRedactedTokenError(t, err)
}

func TestRefreshOAuthSession_IncludesRedactedBody(t *testing.T) {
	t.Parallel()
	srv := tokenErrorServer(t, `{"error":"invalid_grant","error_description":"`+jwtShapedDescription+`"}`)
	_, _, err := refreshOAuthSession(context.Background(), oauthSessionState{
		provider: ProviderGrok, refresh: "refresh", clientID: "test-client", tokenURL: srv.URL, httpClient: srv.Client(),
	})
	assertRedactedTokenError(t, err)
}

func TestOAuthTokenError_CapsBody(t *testing.T) {
	t.Parallel()
	srv := tokenErrorServer(t, strings.Repeat("A", 4096))
	_, err := exchangeOAuthCode(context.Background(),
		oauthFlowConfig{provider: ProviderGrok, clientID: "test-client", httpClient: srv.Client()},
		oauthEndpoints{Token: srv.URL}, "code", "http://127.0.0.1/callback", "verifier", "")
	if err == nil {
		t.Fatal("exchangeOAuthCode() error = nil, want the 400")
	}
	if msg := err.Error(); !strings.Contains(msg, strings.Repeat("A", 2048)) || strings.Contains(msg, strings.Repeat("A", 2049)) {
		t.Fatalf("error carries %d body bytes, want exactly the first 2048", strings.Count(msg, "A"))
	}
}

func tokenErrorServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func assertRedactedTokenError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("error = nil, want the token endpoint's 400")
	}
	msg := err.Error()
	if !strings.Contains(msg, "400") || !strings.Contains(msg, "invalid_grant") {
		t.Errorf("error = %q, want the status and the body's invalid_grant", msg)
	}
	if strings.Contains(msg, "eyJ") {
		t.Errorf("error = %q leaks the JWT-shaped value", msg)
	}
}

type failingTokenStore struct {
	err          error
	saves        int
	savedAccess  string
	savedRefresh string
}

func (s *failingTokenStore) Load(context.Context, ProviderID) (*OAuthSession, error) {
	return nil, nil
}

func (s *failingTokenStore) Save(_ context.Context, _ ProviderID, session *OAuthSession) error {
	s.saves++
	s.savedAccess = session.Access
	s.savedRefresh = session.Refresh
	return s.err
}

func (s *failingTokenStore) Delete(context.Context, ProviderID) error {
	return nil
}

// TestOAuthSession_SaveFailureKeepsRotation (0016-MADR D4, replacing the
// test that pinned finding M4): when saving a rotated session fails, the
// session still adopts it and returns the fresh token, logs the failure,
// sends the rotated refresh token on the next refresh, and retries the save
// on the next call.
func TestOAuthSession_SaveFailureKeepsRotation(t *testing.T) {
	var sent []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		if got := r.Form.Get("grant_type"); got != "refresh_token" {
			t.Errorf("grant_type = %q, want refresh_token", got)
		}
		if got := r.Form.Get("client_id"); got != "client-test" {
			t.Errorf("client_id = %q, want client-test", got)
		}
		sent = append(sent, r.Form.Get("refresh_token"))
		n := len(sent)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  fmt.Sprintf("access-%d", n),
			"refresh_token": fmt.Sprintf("refresh-%d", n),
			"expires_in":    1800,
		})
	}))
	defer srv.Close()

	saveErr := errors.New("save failed")
	store := &failingTokenStore{err: saveErr}
	var logged bytes.Buffer
	session := &OAuthSession{
		Provider:   "openai",
		Access:     "old-access",
		Refresh:    "old-refresh",
		Expiry:     time.Now().Add(-time.Minute),
		ClientID:   "client-test",
		TokenURL:   srv.URL,
		Store:      store,
		HTTPClient: srv.Client(),
		Logger:     slog.New(slog.NewTextHandler(&logged, nil)),
	}

	tok, err := session.Token(context.Background())
	if err != nil || tok.Value != "access-1" {
		t.Fatalf("Token = %q, %v; want the fresh access-1 despite the failed save", tok.Value, err)
	}
	if session.Refresh != "refresh-1" || store.savedRefresh != "refresh-1" {
		t.Errorf("session refresh %q, save attempted with %q; want refresh-1 adopted and offered to the store",
			session.Refresh, store.savedRefresh)
	}
	if !strings.Contains(logged.String(), "rotated session not saved") || !strings.Contains(logged.String(), "save failed") {
		t.Errorf("log = %q, want the unsaved rotation reported", logged.String())
	}

	session.Expiry = time.Now().Add(-time.Minute) // force the next refresh
	if _, err := session.Token(context.Background()); err != nil {
		t.Fatalf("second Token: %v", err)
	}
	if want := []string{"old-refresh", "refresh-1"}; !slices.Equal(sent, want) {
		t.Errorf("refresh tokens sent = %v, want %v: the rotated token, never the spent one", sent, want)
	}

	store.err = nil
	saves := store.saves
	if _, err := session.Token(context.Background()); err != nil {
		t.Fatalf("third Token: %v", err)
	}
	if store.saves != saves+1 || store.savedRefresh != "refresh-2" {
		t.Errorf("retry saved %q after %d saves; want refresh-2 saved once more", store.savedRefresh, store.saves-saves)
	}
	if _, err := session.Token(context.Background()); err != nil || store.saves != saves+1 {
		t.Errorf("a saved session was saved again (%d saves), err %v", store.saves-saves, err)
	}
}

func TestOAuthSession_SingleFlight(t *testing.T) {
	var calls atomic.Int32
	arrived := make(chan struct{}, 2)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		arrived <- struct{}{}
		<-release
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "fresh-access",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	session := &OAuthSession{
		Provider:   "openai",
		Refresh:    "refresh",
		ClientID:   "client-test",
		TokenURL:   srv.URL,
		HTTPClient: srv.Client(),
	}
	type result struct {
		token Token
		err   error
	}
	results := make(chan result, 2)
	requestToken := func() {
		tok, err := session.Token(context.Background())
		results <- result{token: tok, err: err}
	}

	go requestToken()
	select {
	case <-arrived:
	case <-time.After(time.Second):
		t.Fatal("first refresh request did not arrive")
	}

	secondStarted := make(chan struct{})
	go func() {
		close(secondStarted)
		requestToken()
	}()
	<-secondStarted

	select {
	case <-arrived:
		// A second request is the failure asserted below after both calls return.
	case <-time.After(250 * time.Millisecond):
	}
	close(release)

	for range 2 {
		res := <-results
		if res.err != nil {
			t.Errorf("Token: %v", res.err)
		}
		if res.token.Value != "fresh-access" {
			t.Errorf("token value = %q, want fresh-access", res.token.Value)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("refresh request count = %d, want 1", got)
	}
}

func TestOAuthSession_SkewsFiveMinutes(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "refreshed-access",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	session := &OAuthSession{
		Provider:   "openai",
		Access:     "current-access",
		Refresh:    "refresh",
		ClientID:   "client-test",
		TokenURL:   srv.URL,
		HTTPClient: srv.Client(),
	}

	session.Expiry = time.Time{}
	tok, err := session.Token(context.Background())
	if err != nil {
		t.Fatalf("Token with zero expiry: %v", err)
	}
	if tok.Value != "current-access" || calls.Load() != 0 {
		t.Errorf("zero expiry token = %q, calls = %d; want current-access, 0", tok.Value, calls.Load())
	}

	session.Expiry = time.Now().Add(6 * time.Minute)
	tok, err = session.Token(context.Background())
	if err != nil {
		t.Fatalf("Token outside skew: %v", err)
	}
	if tok.Value != "current-access" || calls.Load() != 0 {
		t.Errorf("outside-skew token = %q, calls = %d; want current-access, 0", tok.Value, calls.Load())
	}

	session.Expiry = time.Now().Add(90 * time.Second)
	tok, err = session.Token(context.Background())
	if err != nil {
		t.Fatalf("Token inside skew: %v", err)
	}
	if tok.Value != "refreshed-access" || calls.Load() != 1 {
		t.Errorf("inside-skew token = %q, calls = %d; want refreshed-access, 1", tok.Value, calls.Load())
	}
}

func TestOAuthSession_ChatGPTDetectsIssuer(t *testing.T) {
	for _, tc := range []struct {
		name   string
		issuer string
		want   bool
	}{
		{name: "exact", issuer: DefaultOpenAIIssuer, want: true},
		{name: "trailing slash", issuer: DefaultOpenAIIssuer + "/", want: true},
		{name: "different issuer", issuer: "https://auth.x.ai", want: false},
		{name: "empty", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			session := &OAuthSession{Issuer: tc.issuer}
			if got := session.ChatGPT(); got != tc.want {
				t.Errorf("ChatGPT() = %t, want %t", got, tc.want)
			}
		})
	}
}
