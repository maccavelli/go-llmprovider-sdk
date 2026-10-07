package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

const (
	// oauthRefreshSkew refreshes an access token this long before it
	// expires: five minutes, as Codex (login/src/auth/manager.rs:203-216) and
	// the Grok CLI (xai-grok-login/src/model.rs:9) do.
	oauthRefreshSkew         = 5 * time.Minute
	oauthAuthorizationHeader = "Authorization"
	// oauthErrorBodyLimit caps how much of a failed token response an error
	// carries (MADR 0008 D8).
	oauthErrorBodyLimit = 2048
	// oauthRefreshAttempts and oauthRefreshBackoff retry a refresh that
	// failed in transport, with 429 or with 5xx, as the Grok CLI does
	// (xai-grok-login/src/oidc/protocol.rs:430-438).
	oauthRefreshAttempts = 3
	oauthRefreshBackoff  = 200 * time.Millisecond
	// oauthRefreshToken is the refresh grant type, token field and hint.
	oauthRefreshToken = "refresh_token"
	// oauthEarlyRefreshFloor is how much life a token must have left to be
	// used after its early refresh failed; the next try waits
	// oauthEarlyRefreshRetry plus up to oauthEarlyRefreshJitter (0021-MADR
	// T2).
	oauthEarlyRefreshFloor  = 30 * time.Second
	oauthEarlyRefreshRetry  = 10 * time.Second
	oauthEarlyRefreshJitter = 20 * time.Second
	// oauthResaveInterval spaces the retries of an unsaved rotation's save
	// (0021-MADR T10).
	oauthResaveInterval = 30 * time.Second
)

// oauthRefreshAttemptTimeout bounds one refresh attempt, from send until its
// body is read, so a stalled issuer cannot hold the refresh lock for long
// (0021-MADR T8). It is a variable so that tests can shorten it.
var oauthRefreshAttemptTimeout = 15 * time.Second

// oauthTerminalRefreshCodes mean the refresh token is dead: Codex's permanent
// failures (login/src/auth/manager.rs:1657-1690) and the Grok CLI's
// (xai-grok-login/src/oidc/refresh.rs:21-27).
var oauthTerminalRefreshCodes = []string{
	"invalid_grant", "invalid_client",
	"refresh_token_expired", "refresh_token_reused", "refresh_token_invalidated",
}

// chatGPTAccessFixture is the stub access token a consumer test once wrote into
// a live token store (MADR 0008 F3, F8).
const chatGPTAccessFixture = "chatgpt-access"

// ValidateOAuthSession reports whether a session can be used for generation
// (MADR 0008 D7). It must be refreshable, or the explicit access-only ChatGPT
// token that token_stdin produces, and never a stub. A
// refreshable session needs a client id, and a token URL, which may be empty
// only for OpenAI's and xAI's own issuers, from which the refresh derives it
// (0020-MADR F45; 0026-MADR F44).
func ValidateOAuthSession(session *OAuthSession) error {
	switch {
	case session == nil || strings.TrimSpace(session.Access) == "":
		return errors.New("oauth: session has no access token")
	case session.Access == chatGPTAccessFixture:
		return errors.New("oauth: session holds the chatgpt-access test fixture, not a real token")
	case session.Refresh == "":
		if strings.TrimRight(session.Issuer, "/") != DefaultOpenAIIssuer ||
			session.ClientID != DefaultOpenAIClientID || !session.Expiry.IsZero() {
			return errors.New("oauth: no refresh token")
		}
	case session.ClientID == "":
		return errors.New("oauth: refreshable session has no client id")
	default:
		if _, err := refreshTokenURL(oauthSessionState{provider: session.Provider, issuer: session.Issuer,
			tokenURL: session.TokenURL}); err != nil {
			return err
		}
	}
	return nil
}

// oauthHTTPStatusError reports a failed token-endpoint response by its status
// and the first oauthErrorBodyLimit bytes of its body, stripped of control
// characters and redacted (0026-MADR F18). It closes the body; the raw body is
// never logged.
func oauthHTTPStatusError(op string, resp *http.Response) error {
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, oauthErrorBodyLimit))
	closeErr := resp.Body.Close()
	err := fmt.Errorf("oauth: %s failed: %s: %s", op, redact.StripControl(resp.Status),
		redact.String(redact.StripControl(strings.TrimSpace(string(body)))))
	if readErr != nil {
		err = errors.Join(err, fmt.Errorf("oauth: read %s response: %w", op, readErr))
	}
	if closeErr != nil {
		err = errors.Join(err, fmt.Errorf("oauth: close %s response: %w", op, closeErr))
	}
	return err
}

type oauthRefreshResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    oauthSeconds `json:"expires_in"`
}

// Token returns the current bearer token, refreshing and persisting it when
// it is expired or due: within the refresh skew, or within half its lifetime
// when that is shorter (0021-MADR T3). A refreshed session is adopted before
// it is saved: if the save fails, the fresh token is still returned, the
// failure goes to Logger, and the save is retried at most every 30 s without
// waiting for the refresh lock, so a rotated refresh token is never discarded
// (0016-MADR D4; 0021-MADR T10).
//
// An early refresh that fails with anything but ErrAuthFailure, while the
// token has more than 30 s left, returns the token: the failure goes to
// Logger, and the next refresh waits 10 to 30 s (0021-MADR T2).
//
// The refresh runs detached from ctx, within its own per-attempt bound: the
// issuer rotates the refresh token before it answers, so a refresh cut off by
// a caller's deadline would lose the rotation. Each caller stops waiting when
// its own ctx ends, and a refresh that finishes after every caller has gone
// is still adopted and saved (0026-MADR F3).
func (s *OAuthSession) Token(ctx context.Context) (llmprovider.Token, error) {
	s.mu.Lock()
	now := s.clock()
	token, current := s.currentToken()
	if !current && now.Before(s.nextRefresh) {
		token, current = s.heldToken()
	}
	unsaved := s.storedRefresh != "" && s.storedRefresh != s.Refresh
	retrySave := current && unsaved && s.Store != nil && !now.Before(s.nextResave)
	if current && !retrySave {
		s.mu.Unlock()
		return token, nil
	}
	future := s.inflight
	if future != nil {
		onJoin := s.onJoin
		s.mu.Unlock()
		if onJoin != nil {
			onJoin.joined()
		}
		return waitToken(ctx, future)
	}
	if !current && s.Refresh == "" {
		s.mu.Unlock()
		// Nothing can renew it: the user must sign in again.
		return llmprovider.Token{}, fmt.Errorf("oauth: no refresh token: %w", llmprovider.ErrAuthFailure)
	}
	if err := ctx.Err(); err != nil {
		s.mu.Unlock()
		return llmprovider.Token{}, err
	}

	future = &tokenFuture{done: make(chan struct{})}
	s.inflight = future
	state := s.refreshState()
	state.stored = s.storedRefresh
	var pending *OAuthSession
	if retrySave {
		pending = s.persistable()
		s.nextResave = now.Add(oauthResaveInterval)
	}
	s.mu.Unlock()

	go s.complete(context.WithoutCancel(ctx), future, state, token, pending)
	return waitToken(ctx, future)
}

// waitToken waits for the refresh in flight, or for ctx.
func waitToken(ctx context.Context, future *tokenFuture) (llmprovider.Token, error) {
	select {
	case <-future.done:
		return future.tok, future.err
	case <-ctx.Done():
		return llmprovider.Token{}, ctx.Err()
	}
}

// complete runs the refresh, or the save of an unsaved rotation (pending,
// with token the current token), records the outcome on the session, and
// releases future. It runs detached from every caller (0026-MADR F3).
func (s *OAuthSession) complete(ctx context.Context, future *tokenFuture, state oauthSessionState, token llmprovider.Token, pending *OAuthSession) {
	s.mu.Lock()
	logger := s.Logger
	s.mu.Unlock()
	var err error
	if pending != nil {
		saveErr := persistRotation(ctx, state, pending, state.stored)
		s.mu.Lock()
		switch {
		case saveErr == nil:
			s.storedRefresh = pending.Refresh
		case errors.Is(saveErr, errStoreMovedOn):
			s.storedRefresh = "" // another process saved a newer session
		}
		if saveErr != nil {
			logUnsavedRotation(logger, state.provider, saveErr)
		}
	} else {
		var out refreshed
		out, err = reloadOrRefresh(ctx, state)
		s.mu.Lock()
		if err == nil {
			s.adopt(out.next)
			token = out.token
			s.nextRefresh = time.Time{}
			switch {
			case out.saveErr == nil:
				// Saved, or adopted from the store: the store holds it.
				s.storedRefresh = out.next.Refresh
			case s.storedRefresh == "":
				// The store still holds the token this refresh spent. After
				// an earlier failed save, it still holds that one's.
				s.storedRefresh = out.spent
			}
			if out.saveErr != nil {
				logUnsavedRotation(logger, state.provider, out.saveErr)
			}
		} else if held, ok := s.heldToken(); ok && !errors.Is(err, llmprovider.ErrAuthFailure) {
			//nolint:gosec // G404: non-crypto jitter between refresh attempts
			s.nextRefresh = s.clock().Add(oauthEarlyRefreshRetry + rand.N(oauthEarlyRefreshJitter))
			logEarlyRefreshFailure(logger, state.provider, err)
			token, err = held, nil
		} else {
			token = llmprovider.Token{}
		}
	}
	future.tok = token
	future.err = err
	s.inflight = nil
	close(future.done)
	s.mu.Unlock()
}

// persistable copies the session's stored fields, without its lock.
func (s *OAuthSession) persistable() *OAuthSession {
	return &OAuthSession{
		Provider: s.Provider, Access: s.Access, Refresh: s.Refresh, Expiry: s.Expiry,
		Issuer: s.Issuer, ClientID: s.ClientID, AccountID: s.AccountID, FedRAMP: s.FedRAMP,
		TokenURL: s.TokenURL, Store: s.Store, HTTPClient: s.HTTPClient,
	}
}

// errStoreMovedOn reports that the store no longer holds the refresh token an
// unsaved rotation spent: another process saved a newer session, which must
// not be overwritten.
var errStoreMovedOn = errors.New("oauth: the store holds a newer session; the unsaved rotation is not written")

// persistRotation retries saving a rotated session, under the store's
// refresh lock, and only while the store still holds stored, the refresh
// token it is known to hold (or nothing). It tries the lock once, with a
// context already done, so a call that holds a valid token never waits for
// another process's refresh (0021-MADR T10; 0026-MADR F1).
func persistRotation(ctx context.Context, state oauthSessionState, next *OAuthSession, stored string) error {
	if locker, ok := state.store.(RefreshLocker); ok {
		once, cancel := context.WithCancel(ctx)
		cancel()
		unlock, err := locker.LockRefresh(once, state.provider)
		if err != nil {
			return err
		}
		defer unlock()
	}
	loaded, err := state.store.Load(ctx, state.provider)
	if err == nil && loaded != nil && loaded.Refresh != stored && loaded.Refresh != next.Refresh {
		return errStoreMovedOn
	}
	return state.store.Save(ctx, state.provider, next)
}

// logUnsavedRotation reports a rotated session that could not be saved.
func logUnsavedRotation(logger *slog.Logger, provider llmprovider.ProviderID, err error) {
	if logger == nil {
		return
	}
	logger.Warn("oauth: rotated session not saved; it is kept in memory and the save is retried",
		"provider", provider, "error", err)
}

// logEarlyRefreshFailure reports an early refresh that failed while the
// current token still works.
func logEarlyRefreshFailure(logger *slog.Logger, provider llmprovider.ProviderID, err error) {
	if logger == nil {
		return
	}
	logger.Warn("oauth: early refresh failed; the current token is used and the refresh is retried",
		"provider", provider, "error", err)
}

// Owner returns the provider whose credential the session is: Provider when
// it is set, else the provider whose sign-in issuer the session names
// (OpenAI's is openai's, xAI's is grok's), else "". A provider refuses a
// session another provider owns (R16; 0026-MADR F2 and its amendment of
// 2026-10-06). It reads the fields under the session's lock, since a refresh
// rewrites them.
func (s *OAuthSession) Owner() llmprovider.ProviderID {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Provider != "" {
		return s.Provider
	}
	switch strings.TrimRight(s.Issuer, "/") {
	case DefaultOpenAIIssuer:
		return llmprovider.ProviderOpenAI
	case DefaultGrokOAuthIssuer:
		return llmprovider.ProviderGrok
	}
	return ""
}

// ChatGPT reports whether the session was issued by the default OpenAI issuer.
func (s *OAuthSession) ChatGPT() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	// An OpenAI session with no issuer is an older ChatGPT sign-in, as its
	// refresh treats it (refreshTokenURL), never a platform key (0026-MADR F2).
	issuer := strings.TrimRight(s.Issuer, "/")
	return issuer == DefaultOpenAIIssuer || issuer == "" && s.Provider == llmprovider.ProviderOpenAI
}

// Account returns the session's ChatGPT account id and FedRAMP flag, under
// its lock: what the openai provider sends with a ChatGPT session (0015-MADR
// amendment "the ChatGPT helpers leave llmprovider").
func (s *OAuthSession) Account() (id string, fedRAMP bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.AccountID, s.FedRAMP
}

// Invalidate moves the session's expiry into the past, so its next Token
// refreshes. With it, an *OAuthSession is an InvalidatingSource: a provider
// that gets a 401 invalidates the session and retries once.
func (s *OAuthSession) Invalidate() {
	s.mu.Lock()
	s.Expiry = s.clock().Add(-time.Second)
	s.mu.Unlock()
}

// InvalidateToken expires the session when t is its current access token, so
// its next Token refreshes; a token the session has already replaced is left
// alone. With it, an *OAuthSession is a TokenInvalidator (0021-MADR D5).
func (s *OAuthSession) InvalidateToken(t llmprovider.Token) {
	s.mu.Lock()
	if s.Access != "" && s.Access == t.Value {
		s.Expiry = s.clock().Add(-time.Second)
	}
	s.mu.Unlock()
}

func (s *OAuthSession) clock() time.Time {
	if s.now != nil {
		return s.now.Now()
	}
	return time.Now()
}

// currentToken is the access token when it is not yet due for refresh.
func (s *OAuthSession) currentToken() (llmprovider.Token, bool) {
	if s.Access == "" {
		return llmprovider.Token{}, false
	}
	if !s.Expiry.IsZero() && s.Expiry.Sub(s.clock()) <= s.refreshMargin() {
		return llmprovider.Token{}, false
	}
	return s.bearer(), true
}

// heldToken is the access token while it has more than
// oauthEarlyRefreshFloor left: what Token returns after an early refresh
// failed (0021-MADR T2).
func (s *OAuthSession) heldToken() (llmprovider.Token, bool) {
	if s.Access == "" || s.Expiry.IsZero() || s.Expiry.Sub(s.clock()) <= oauthEarlyRefreshFloor {
		return llmprovider.Token{}, false
	}
	return s.bearer(), true
}

func (s *OAuthSession) bearer() llmprovider.Token {
	return llmprovider.Token{Value: s.Access, Type: llmprovider.TokenBearer, Expiry: s.Expiry}
}

// refreshMargin is how long before expiry the token is due: the five-minute
// skew, or half the token's lifetime when that is shorter, so a token that
// lives five minutes or less is not refreshed on every call (0021-MADR T3).
// The lifetime is exp − iat from the access JWT, else Expiry − issued.
func (s *OAuthSession) refreshMargin() time.Duration {
	lifetime := time.Duration(0)
	if exp, iat := jwtTimes(s.Access); !exp.IsZero() && !iat.IsZero() {
		lifetime = exp.Sub(iat)
	} else if !s.issued.IsZero() && !s.Expiry.IsZero() {
		lifetime = s.Expiry.Sub(s.issued)
	}
	if lifetime <= 0 {
		return oauthRefreshSkew
	}
	return min(oauthRefreshSkew, lifetime/2)
}

type oauthSessionState struct {
	provider llmprovider.ProviderID
	refresh  string
	// stored is the refresh token the store is known to hold. While an
	// unsaved rotation is pending it is a token this session spent: a store
	// that still holds it has not moved on, and it must never be adopted and
	// sent again (0020-MADR F1; 0026-MADR F1).
	stored     string
	issuer     string
	clientID   string
	accountID  string
	fedramp    bool
	tokenURL   string
	store      TokenStore
	httpClient *http.Client
	now        sessionClock
}

func (s *OAuthSession) refreshState() oauthSessionState {
	return oauthSessionState{
		provider:   s.Provider,
		refresh:    s.Refresh,
		issuer:     s.Issuer,
		clientID:   s.ClientID,
		accountID:  s.AccountID,
		fedramp:    s.FedRAMP,
		tokenURL:   s.TokenURL,
		store:      s.Store,
		httpClient: s.HTTPClient,
		now:        s.now,
	}
}

// UseHTTPClient gives the session client, when it has none, so its refreshes
// use the same transport as the provider's requests and listing (0016-MADR
// D8). A session that has a client keeps it; a nil client changes nothing.
func (s *OAuthSession) UseHTTPClient(client *http.Client) {
	if client == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.HTTPClient == nil {
		s.HTTPClient = client
	}
}

// UseLogger gives the session logger, when it has none, so a rotated session
// that could not be saved is reported through the provider's WithLogger
// logger (0016-MADR D4; 0020-MADR F48). A session that has a logger keeps it;
// a nil logger changes nothing.
func (s *OAuthSession) UseLogger(logger *slog.Logger) {
	if logger == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Logger == nil {
		s.Logger = logger
	}
}

func (s *OAuthSession) adopt(next *OAuthSession) {
	s.Provider = next.Provider
	s.Access = next.Access
	s.Refresh = next.Refresh
	s.Expiry = next.Expiry
	s.Issuer = next.Issuer
	s.ClientID = next.ClientID
	s.AccountID = next.AccountID
	s.FedRAMP = next.FedRAMP
	s.TokenURL = next.TokenURL
	s.Store = next.Store
	s.HTTPClient = next.HTTPClient
	s.issued = next.issued
}

func refreshOAuthSession(ctx context.Context, state oauthSessionState) (*OAuthSession, llmprovider.Token, error) {
	var lastErr error
	for attempt := range oauthRefreshAttempts {
		if attempt > 0 {
			if err := sleepWithContext(ctx, oauthRefreshBackoff<<(attempt-1)); err != nil {
				return nil, llmprovider.Token{}, err
			}
		}
		next, token, retry, err := refreshOAuthSessionOnce(ctx, state)
		if !retry {
			return next, token, err
		}
		lastErr = err
	}
	return nil, llmprovider.Token{}, lastErr
}

// refreshOAuthSessionOnce makes one refresh request, bounded by
// oauthRefreshAttemptTimeout. retry reports a transport error, 429 or 5xx.
func refreshOAuthSessionOnce(ctx context.Context, state oauthSessionState) (next *OAuthSession, token llmprovider.Token, retry bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, oauthRefreshAttemptTimeout)
	defer cancel()
	req, err := newRefreshRequest(ctx, state)
	if err != nil {
		return nil, llmprovider.Token{}, false, err
	}
	client := state.httpClient
	if client == nil {
		client = transport.DefaultClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, llmprovider.Token{}, true, fmt.Errorf("%w: oauth: refresh request: %w", llmprovider.ErrProviderUnavailable, err)
	}
	if resp.StatusCode != http.StatusOK {
		retry = resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError
		return nil, llmprovider.Token{}, retry, refreshFailure(resp)
	}

	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, oauthResponseLimit+1))
	if closeErr := resp.Body.Close(); closeErr != nil {
		readErr = errors.Join(readErr, closeErr)
	}
	switch {
	case readErr != nil:
		return nil, llmprovider.Token{}, false, fmt.Errorf("%w: oauth: read refresh response: %w", llmprovider.ErrProviderUnavailable, readErr)
	case len(raw) > oauthResponseLimit:
		return nil, llmprovider.Token{}, false, fmt.Errorf("%w: oauth: refresh response is larger than %d bytes", llmprovider.ErrIncomplete, oauthResponseLimit)
	}
	payload, err := decodeRefreshResponse(raw)
	if err != nil {
		return nil, llmprovider.Token{}, false, fmt.Errorf("%w: %w", llmprovider.ErrIncomplete, err)
	}
	if payload.AccessToken == "" {
		return nil, llmprovider.Token{}, false, fmt.Errorf("%w: oauth: refresh response missing access token", llmprovider.ErrIncomplete)
	}

	refresh := payload.RefreshToken
	if refresh == "" {
		refresh = state.refresh
	}
	now := time.Now()
	if state.now != nil {
		now = state.now.Now()
	}
	next = &OAuthSession{
		Provider:   state.provider,
		Access:     payload.AccessToken,
		Refresh:    refresh,
		Expiry:     tokenExpiry(payload.AccessToken, payload.ExpiresIn, now),
		Issuer:     state.issuer,
		ClientID:   state.clientID,
		AccountID:  state.accountID,
		FedRAMP:    state.fedramp,
		TokenURL:   state.tokenURL,
		Store:      state.store,
		HTTPClient: state.httpClient,
		issued:     now,
		now:        state.now,
	}
	return next, llmprovider.Token{
		Value:  next.Access,
		Type:   llmprovider.TokenBearer,
		Expiry: next.Expiry,
	}, false, nil
}

// newRefreshRequest builds the refresh request: JSON for the OpenAI issuer,
// as Codex sends it (login/src/oauth/client.rs:80-110), form-encoded
// otherwise, as the Grok CLI sends it (xai-grok-login/src/oidc/protocol.rs:492-507).
func newRefreshRequest(ctx context.Context, state oauthSessionState) (*http.Request, error) {
	params := map[string]string{
		"grant_type":      oauthRefreshToken,
		oauthRefreshToken: state.refresh,
		"client_id":       state.clientID,
	}
	contentType := "application/x-www-form-urlencoded"
	var body io.Reader
	if strings.TrimRight(state.issuer, "/") == DefaultOpenAIIssuer {
		raw, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("oauth: encode refresh request: %w", err)
		}
		contentType, body = "application/json", bytes.NewReader(raw)
	} else {
		form := url.Values{}
		for k, v := range params {
			form.Set(k, v)
		}
		body = strings.NewReader(form.Encode())
	}
	tokenURL, err := refreshTokenURL(state)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, body)
	if err != nil {
		return nil, fmt.Errorf("oauth: create refresh request: %w", err)
	}
	transport.NewIdentity("", "", "").SetUserAgent(req)
	req.Header.Set("Content-Type", contentType)
	return req, nil
}

// refreshFailure reports a failed refresh response, closing its body, with
// one kind (0015-MADR D7). A 401 or a terminal error code wraps
// ErrAuthFailure: the refresh token is dead and the user must sign in again.
// A 429 is ErrRateLimited and a 5xx ErrProviderUnavailable. Any other 4xx is
// ErrInvalidRequest: the request was malformed, which signing in again does
// not fix.
func refreshFailure(resp *http.Response) error {
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, oauthErrorBodyLimit))
	ignoreOAuthError(resp.Body.Close())
	resp.Body = io.NopCloser(bytes.NewReader(body))
	err := oauthHTTPStatusError("refresh", resp)
	if readErr != nil {
		err = errors.Join(err, fmt.Errorf("oauth: read refresh response: %w", readErr))
	}
	switch status := resp.StatusCode; {
	case status == http.StatusUnauthorized || slices.Contains(oauthTerminalRefreshCodes, refreshErrorCode(body)):
		return fmt.Errorf("%w: %w", llmprovider.ErrAuthFailure, err)
	case status == http.StatusTooManyRequests:
		return fmt.Errorf("%w: %w", llmprovider.ErrRateLimited, err)
	case status >= http.StatusInternalServerError:
		return fmt.Errorf("%w: %w", llmprovider.ErrProviderUnavailable, err)
	case status >= http.StatusBadRequest:
		return fmt.Errorf("%w: %w", llmprovider.ErrInvalidRequest, err)
	default:
		return err
	}
}

// refreshErrorCode reads an OAuth error code: {"error": "code"},
// {"error": {"code": "code"}} or {"code": "code"}, lower-cased.
func refreshErrorCode(body []byte) string {
	var top struct {
		Error json.RawMessage `json:"error"`
		Code  string          `json:"code"`
	}
	if json.Unmarshal(body, &top) != nil {
		return ""
	}
	if code := jsonString(top.Error); code != "" {
		return strings.ToLower(code)
	}
	var inner struct {
		Code string `json:"code"`
	}
	if json.Unmarshal(top.Error, &inner) == nil && inner.Code != "" {
		return strings.ToLower(inner.Code)
	}
	return strings.ToLower(top.Code)
}

// decodeRefreshResponse decodes a refresh reply. When the full decode fails,
// as on an unreadable expires_in, the two tokens alone are kept if both are
// present: the issuer has already rotated the refresh token, and losing it
// would replay a spent one (0021-MADR T4).
func decodeRefreshResponse(raw []byte) (oauthRefreshResponse, error) {
	var payload oauthRefreshResponse
	err := json.Unmarshal(raw, &payload)
	if err == nil {
		return payload, nil
	}
	var tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if json.Unmarshal(raw, &tokens) == nil && tokens.AccessToken != "" && tokens.RefreshToken != "" {
		return oauthRefreshResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken}, nil
	}
	return oauthRefreshResponse{}, fmt.Errorf("oauth: decode refresh response: %w", err)
}

// tokenExpiry is the access token's JWT exp when it has one, else now plus
// expires_in (3600 s when absent or unusable), as Codex reads it (MADR 0012
// §5.2).
func tokenExpiry(access string, expiresIn oauthSeconds, now time.Time) time.Time {
	if exp := jwtExpiry(access); !exp.IsZero() {
		return exp
	}
	return now.Add(durationFromSeconds(float64(expiresIn), time.Hour, false))
}

// reloadOrRefresh re-reads the session from its store before refreshing:
// another process sharing the store may already have rotated the refresh
// token, and sending the old one again trips the issuer's reuse detection.
// A stored session with a different refresh token is adopted, and refreshed
// only if its own access token is due (MADR 0012 §5.2).
// refreshed is what reloadOrRefresh produced.
type refreshed struct {
	next  *OAuthSession
	token llmprovider.Token
	// saveErr is a failure to save next; next is still the session to use
	// (0016-MADR D4).
	saveErr error
	// spent is the refresh token the refresh consumed; "" when next came
	// from the store.
	spent string
}

// reloadOrRefresh adopts a session another process rotated, else refreshes
// and saves. It holds the store's refresh lock throughout (0016-MADR
// amendment A2). After a permanent refresh failure it re-reads the store
// once, and adopts a rotation a sibling saved meanwhile (amendment A1).
func reloadOrRefresh(ctx context.Context, state oauthSessionState) (refreshed, error) {
	// Lock first, so the reload below sees any rotation another process
	// saved while this one waited.
	if locker, ok := state.store.(RefreshLocker); ok {
		unlock, err := locker.LockRefresh(ctx, state.provider)
		if err != nil {
			return refreshed{}, err
		}
		defer unlock()
	}
	if stored := loadRotated(ctx, state); stored != nil {
		if token, ok := stored.currentToken(); ok {
			return refreshed{next: stored, token: token}, nil
		}
		state = stored.refreshState()
	}
	next, token, err := refreshOAuthSession(ctx, state)
	if errors.Is(err, llmprovider.ErrAuthFailure) {
		if stored := loadRotated(ctx, state); stored != nil {
			if token, ok := stored.currentToken(); ok {
				return refreshed{next: stored, token: token}, nil
			}
			state = stored.refreshState()
			next, token, err = refreshOAuthSession(ctx, state)
		}
	}
	if err != nil {
		return refreshed{}, err
	}
	out := refreshed{next: next, token: token, spent: state.refresh}
	if state.store != nil {
		out.saveErr = state.store.Save(ctx, state.provider, next)
	}
	return out, nil
}

// loadRotated returns the stored session when the store holds a refresh token
// other than state's, else nil. The token the store is known to hold is not
// another: while a rotation is unsaved it is one this session spent, however
// many saves failed (0020-MADR F1; 0026-MADR F1).
func loadRotated(ctx context.Context, state oauthSessionState) *OAuthSession {
	if state.store == nil {
		return nil
	}
	stored, err := state.store.Load(ctx, state.provider)
	if err != nil || stored == nil || stored.Refresh == "" || stored.Refresh == state.refresh ||
		(state.stored != "" && stored.Refresh == state.stored) {
		return nil
	}
	stored.Store, stored.HTTPClient, stored.now = state.store, state.httpClient, state.now
	return stored
}

func refreshTokenURL(state oauthSessionState) (string, error) {
	if state.tokenURL != "" {
		return state.tokenURL, nil
	}
	// Without a token URL, only OpenAI's and xAI's own issuers have a known
	// endpoint. A session with no issuer is an older one of those two
	// providers. Any other issuer's refresh token is never sent elsewhere
	// (0020-MADR F45).
	switch issuer := strings.TrimRight(state.issuer, "/"); {
	case issuer == DefaultOpenAIIssuer, issuer == "" && state.provider == llmprovider.ProviderOpenAI:
		return DefaultOpenAIIssuer + "/oauth/token", nil
	case issuer == DefaultGrokOAuthIssuer, issuer == "" && state.provider == llmprovider.ProviderGrok:
		return defaultGrokOAuthRefreshURL, nil
	default:
		return "", fmt.Errorf("oauth: the %s session from %q has no token URL; sign in again: %w",
			state.provider, state.issuer, llmprovider.ErrAuthFailure)
	}
}
