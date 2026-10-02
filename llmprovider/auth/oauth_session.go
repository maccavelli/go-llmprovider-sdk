package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
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
)

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
// refreshable session needs a client id. Its token URL may be empty, because
// the refresh derives it from the issuer.
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
	}
	return nil
}

// oauthHTTPStatusError reports a failed token-endpoint response by its status
// and the first oauthErrorBodyLimit bytes of its body, redacted. It closes the
// body; the raw body is never logged.
func oauthHTTPStatusError(op string, resp *http.Response) error {
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, oauthErrorBodyLimit))
	closeErr := resp.Body.Close()
	err := fmt.Errorf("oauth: %s failed: %s: %s", op, resp.Status,
		redact.String(strings.TrimSpace(string(body))))
	if readErr != nil {
		err = errors.Join(err, fmt.Errorf("oauth: read %s response: %w", op, readErr))
	}
	if closeErr != nil {
		err = errors.Join(err, fmt.Errorf("oauth: close %s response: %w", op, closeErr))
	}
	return err
}

type oauthRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// Token returns the current bearer token, refreshing and persisting it when
// it is expired or within the refresh skew. A refreshed session is adopted
// before it is saved: if the save fails, the fresh token is still returned,
// the failure goes to Logger, and the save is retried on the next call, so a
// rotated refresh token is never discarded (0016-MADR D4).
func (s *OAuthSession) Token(ctx context.Context) (llmprovider.Token, error) {
	s.mu.Lock()
	token, current := s.currentToken()
	retrySave := current && s.spentRefresh != "" && s.Store != nil
	if current && !retrySave {
		s.mu.Unlock()
		return token, nil
	}
	if future := s.inflight; future != nil {
		s.mu.Unlock()
		select {
		case <-future.done:
			return future.tok, future.err
		case <-ctx.Done():
			return llmprovider.Token{}, ctx.Err()
		}
	}
	if !current && s.Refresh == "" {
		s.mu.Unlock()
		// Nothing can renew it: the user must sign in again.
		return llmprovider.Token{}, fmt.Errorf("oauth: no refresh token: %w", llmprovider.ErrAuthFailure)
	}

	future := &tokenFuture{done: make(chan struct{})}
	s.inflight = future
	state := s.refreshState()
	logger, spent := s.Logger, s.spentRefresh
	var unsaved *OAuthSession
	if retrySave {
		unsaved = s.persistable()
	}
	s.mu.Unlock()

	var err error
	if retrySave {
		saveErr := persistRotation(ctx, state, unsaved, spent)
		s.mu.Lock()
		if saveErr == nil || errors.Is(saveErr, errStoreMovedOn) {
			s.spentRefresh = ""
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
			s.spentRefresh = ""
			if out.saveErr != nil {
				s.spentRefresh = out.spent
				logUnsavedRotation(logger, state.provider, out.saveErr)
			}
		} else {
			token = llmprovider.Token{}
		}
	}
	future.tok = token
	future.err = err
	s.inflight = nil
	close(future.done)
	s.mu.Unlock()

	return token, err
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
// refresh lock, and only while the store still holds the refresh token that
// rotation spent (or nothing).
func persistRotation(ctx context.Context, state oauthSessionState, next *OAuthSession, spent string) error {
	if locker, ok := state.store.(RefreshLocker); ok {
		unlock, err := locker.LockRefresh(ctx, state.provider)
		if err != nil {
			return err
		}
		defer unlock()
	}
	stored, err := state.store.Load(ctx, state.provider)
	if err == nil && stored != nil && stored.Refresh != spent && stored.Refresh != next.Refresh {
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

// ChatGPT reports whether the session was issued by the default OpenAI issuer.
func (s *OAuthSession) ChatGPT() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.TrimRight(s.Issuer, "/") == DefaultOpenAIIssuer
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
	s.Expiry = time.Now().Add(-time.Second)
	s.mu.Unlock()
}

func (s *OAuthSession) currentToken() (llmprovider.Token, bool) {
	if s.Access == "" {
		return llmprovider.Token{}, false
	}
	if !s.Expiry.IsZero() && time.Until(s.Expiry) <= oauthRefreshSkew {
		return llmprovider.Token{}, false
	}
	return llmprovider.Token{
		Value:  s.Access,
		Type:   llmprovider.TokenBearer,
		Expiry: s.Expiry,
	}, true
}

type oauthSessionState struct {
	provider   llmprovider.ProviderID
	refresh    string
	issuer     string
	clientID   string
	accountID  string
	fedramp    bool
	tokenURL   string
	store      TokenStore
	httpClient *http.Client
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

// refreshOAuthSessionOnce makes one refresh request. retry reports a
// transport error, 429 or 5xx.
func refreshOAuthSessionOnce(ctx context.Context, state oauthSessionState) (next *OAuthSession, token llmprovider.Token, retry bool, err error) {
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
		return nil, llmprovider.Token{}, true, fmt.Errorf("oauth: refresh request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		retry = resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError
		return nil, llmprovider.Token{}, retry, refreshFailure(resp)
	}

	var payload oauthRefreshResponse
	decodeErr := json.NewDecoder(resp.Body).Decode(&payload)
	closeErr := resp.Body.Close()
	if decodeErr != nil {
		err := fmt.Errorf("oauth: decode refresh response: %w", decodeErr)
		if closeErr != nil {
			err = errors.Join(err, fmt.Errorf("oauth: close refresh response: %w", closeErr))
		}
		return nil, llmprovider.Token{}, false, err
	}
	if closeErr != nil {
		return nil, llmprovider.Token{}, false, fmt.Errorf("oauth: close refresh response: %w", closeErr)
	}
	if payload.AccessToken == "" {
		return nil, llmprovider.Token{}, false, errors.New("oauth: refresh response missing access token")
	}

	refresh := payload.RefreshToken
	if refresh == "" {
		refresh = state.refresh
	}
	next = &OAuthSession{
		Provider:   state.provider,
		Access:     payload.AccessToken,
		Refresh:    refresh,
		Expiry:     tokenExpiry(payload.AccessToken, payload.ExpiresIn, time.Now()),
		Issuer:     state.issuer,
		ClientID:   state.clientID,
		AccountID:  state.accountID,
		FedRAMP:    state.fedramp,
		TokenURL:   state.tokenURL,
		Store:      state.store,
		HTTPClient: state.httpClient,
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, refreshTokenURL(state), body)
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

// tokenExpiry is the access token's JWT exp when it has one, else now plus
// expires_in (3600 s when absent), as Codex reads it (MADR 0012 §5.2).
func tokenExpiry(access string, expiresIn int64, now time.Time) time.Time {
	if exp := jwtExpiry(access); !exp.IsZero() {
		return exp
	}
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	return now.Add(time.Duration(expiresIn) * time.Second)
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
// other than state's, else nil.
func loadRotated(ctx context.Context, state oauthSessionState) *OAuthSession {
	if state.store == nil {
		return nil
	}
	stored, err := state.store.Load(ctx, state.provider)
	if err != nil || stored == nil || stored.Refresh == "" || stored.Refresh == state.refresh {
		return nil
	}
	stored.Store, stored.HTTPClient = state.store, state.httpClient
	return stored
}

func refreshTokenURL(state oauthSessionState) string {
	if state.tokenURL != "" {
		return state.tokenURL
	}
	issuer := strings.TrimRight(state.issuer, "/")
	if issuer == DefaultOpenAIIssuer {
		return issuer + "/oauth/token"
	}
	return defaultGrokOAuthRefreshURL
}
