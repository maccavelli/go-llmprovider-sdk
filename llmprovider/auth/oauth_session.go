package auth

import (
	"bytes"
	"cmp"
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
	// (xai-grok-login/src/oidc/protocol.rs:430-438); a refused certificate
	// is not retried (0068-MADR D2).
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
// (0020-MADR F45; 0026-MADR F44). A custom issuer's session with no token URL
// gives an error matching llmprovider.ErrAuthFailure; the other refusals are
// plain errors naming the reason.
func ValidateOAuthSession(session *OAuthSession) error {
	switch {
	case session == nil || strings.TrimSpace(session.Access) == "":
		return errors.New("oauth: session has no access token")
	case session.Access == chatGPTAccessFixture:
		return errors.New("oauth: session holds the chatgpt-access test fixture, not a real token")
	case session.Kind == SessionChatGPTPlan && session.Resource == "":
		return errors.New("oauth: session kind chatgpt_plan contradicts its fields: it has no resource")
	case session.Kind == SessionKiloDevice && session.Refresh != "":
		return errors.New("oauth: session kind kilo_device contradicts its fields: it has a refresh token")
	case session.Refresh == "":
		// An access-only session is a ChatGPT one, from OpenAI's own issuer.
		if !openAIKind(sessionKindOf(session.Kind, session.Provider, session.Issuer, session.Resource)) ||
			strings.TrimRight(session.Issuer, "/") != DefaultOpenAIIssuer ||
			session.ClientID != DefaultOpenAIClientID || !session.Expiry.IsZero() {
			return errors.New("oauth: no refresh token")
		}
	case session.ClientID == "":
		return errors.New("oauth: refreshable session has no client id")
	// A plan session refreshes only at the token endpoint it signed in at
	// (0043-MADR D2).
	case session.Resource != "" && session.TokenURL == "":
		return errors.New("oauth: plan session has no token URL")
	default:
		if _, err := refreshTokenURL(oauthSessionState{provider: session.Provider, issuer: session.Issuer,
			tokenURL: session.TokenURL, resource: session.Resource, kind: session.Kind}); err != nil {
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
	// Scope and IDToken are kept on a plan session when a refresh returns
	// them (0043-MADR D2).
	Scope   string `json:"scope"`
	IDToken string `json:"id_token"`
}

// Token returns the current bearer token, refreshing and persisting it when
// it is expired or due: within the refresh skew, or within half its lifetime
// when that is shorter (0021-MADR T3). A refreshed session is adopted before
// it is saved: if the save fails, the fresh token is still returned, the
// failure goes to Logger, and the save is retried at most every 30 s without
// waiting for the refresh lock, so a rotated refresh token is never discarded
// (0016-MADR D4; 0021-MADR T10).
//
// Before it refreshes, the session re-reads its store and adopts what a
// sibling process rotated, but only a rotation of the same sign-in: the same
// provider, kind, issuer and client, and the same account when both name one;
// an empty issuer is the kind's built-in issuer. A stored record that is
// another sign-in, another application's login, or another kind, client,
// issuer, account or provider, with a refresh token or without one, is never
// adopted, used or refreshed: the token request fails with an error matching
// llmprovider.ErrAuthFailure, which names the field that differs and no value
// the store holds, and nothing is sent or saved. The session keeps its own
// fields, and each later token request reads the store again, so it resumes
// once the store holds this sign-in again, or nothing. A record that holds the
// session's own refresh token or its own access token is its own whatever its
// other fields say: nothing is adopted from it, and the session refreshes its
// own token and saves over it (0082-MADR, second amendment, R-5).
//
// The refresh's save is guarded the same way: under the store's refresh lock,
// the store is read again before the save, and the rotation is written only if
// the store still holds this sign-in's token or nothing. If another sign-in
// was written during the refresh's request, the rotation is kept in memory,
// unsaved: the token it holds is returned, the unsaved save is retried as
// above and fails the same way while the store holds the other sign-in, and
// the next due refresh is refused. A store that is not a RefreshLocker gets the
// same check, which is then not atomic with the save. Of the tokens the
// session spent, only the last is known: a store that returns to it takes the
// rotation, and it is never adopted back (0082-MADR D1 to D3 and its
// amendment of 2026-11-17).
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

// Wait returns once the session has no refresh in flight, and no retried
// save of a rotated session (0071-MADR D3). It returns nil, or the context's
// error, ctx.Err(), if ctx ends first; it returns no other error. It cancels
// nothing: a refresh runs detached from its
// callers so that a rotation is saved (0026-MADR F3), and Wait lets a test,
// or a process that is ending, wait for it. When Wait returns nil, the
// refresh has released the store's refresh lock and attempted its save. A
// Token call after that may start another refresh.
//
// A caller that also owns a catalog.Cache through which the session is
// asked for tokens calls the cache's Close first. Close ends a background
// fetch's wait for its token, but not the refresh behind it. Wait is safe
// for concurrent use.
func (s *OAuthSession) Wait(ctx context.Context) error {
	for {
		s.mu.Lock()
		future := s.inflight
		s.mu.Unlock()
		if future == nil {
			return nil
		}
		select {
		case <-future.done:
		case <-ctx.Done():
			s.mu.Lock()
			idle := s.inflight == nil
			s.mu.Unlock()
			if idle {
				return nil
			}
			return ctx.Err()
		}
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
			s.Consents = pending.Consents
		case errors.Is(saveErr, errStoreMovedOn):
			s.storedRefresh = "" // another process saved a newer session
		}
		// errStoreOtherSignIn leaves storedRefresh as it is: the token the
		// session last knew the store to hold stays known as spent, so it is
		// never adopted back if the store returns to it (0020-MADR F1).
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
		TokenURL: s.TokenURL, Resource: s.Resource, Scope: s.Scope, IDToken: s.IDToken, HostID: s.HostID,
		Kind: s.Kind, Consents: s.Consents, Store: s.Store, HTTPClient: s.HTTPClient,
	}
}

// snapshot is persistable taken under the session's lock: the copy a store
// reads when a refresh may be adopting into the session (0074-PLAN Phase 7
// step 6). No caller holds s.mu while it calls a store's Save.
func (s *OAuthSession) snapshot() *OAuthSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.persistable()
}

// errStoreMovedOn reports that the store no longer holds the refresh token an
// unsaved rotation spent: another process saved a newer session of this
// sign-in, which must not be overwritten.
var errStoreMovedOn = errors.New("oauth: the store holds a newer session; the unsaved rotation is not written")

// errStoreOtherSignIn reports that the store holds another sign-in: nothing is
// written over it, and the rotation stays in memory (0082-MADR D1, and its
// amendment of 2026-11-17, R-1 and R-2).
var errStoreOtherSignIn = errors.New("oauth: the store holds another sign-in; the unsaved rotation is not written over it")

// ownTokens are the tokens a session knows to be its own: the refresh tokens
// it holds, spent or noted as the store's, and its access tokens. A stored
// record that carries one of them is the session's own record, whatever its
// other fields say, and is not judged by sameSignIn: a store that drops the
// refresh token, the client or another field still serves its one process, as
// it did before 0082. An empty token is nobody's. Another sign-in's record
// carries another sign-in's tokens: refresh and access tokens are the
// issuer's, each belongs to one login (0082-MADR amendment of 2026-11-17, R-5).
type ownTokens struct{ refresh, access []string }

// holds reports whether rec carries one of the session's own tokens.
func (o ownTokens) holds(rec *OAuthSession) bool {
	return isKnownToken(rec.Refresh, o.refresh...) || isKnownToken(rec.Access, o.access...)
}

// isKnownToken reports whether token is a non-empty token among known.
func isKnownToken(token string, known ...string) bool {
	return token != "" && slices.Contains(known, token)
}

// storeMovedOn is the one test of whether the store, which holds loaded, may
// be written over by a rotation of held's sign-in; nil means it may (E27). It
// may when the record carries a token the session knows (own), or is of this
// sign-in with no refresh token of its own, such as a pasted access token. A
// record of another sign-in is errStoreOtherSignIn; one of this sign-in with a
// refresh token the session does not know is errStoreMovedOn. The comparison
// is sameSignIn's, so the save and the read agree (0082-MADR amendment of
// 2026-11-17, R-1, R-5). loaded is a record the store holds: the caller has
// none to judge when it holds nothing.
func storeMovedOn(held oauthSessionState, loaded *OAuthSession, own ownTokens) error {
	switch {
	case own.holds(loaded):
		return nil
	case !sameSignIn(held, loaded):
		return errStoreOtherSignIn
	case loaded.Refresh == "":
		return nil
	}
	return errStoreMovedOn
}

// saveRotation saves next, a rotation of held's sign-in, unless the store has
// moved on (storeMovedOn); it then writes nothing and returns that error, so
// the caller keeps the rotation in memory. The caller holds the store's
// refresh lock when the store offers one: the load and the save are then not
// interleaved with another process's refresh or sign-in save that takes it.
// A store that is not a RefreshLocker gets the same check, not atomic with the
// save. A failed load saves, as persistRotation always did (a known limit: a
// persistent read error would otherwise lose every rotation).
func saveRotation(ctx context.Context, held oauthSessionState, next *OAuthSession, own ownTokens) error {
	loaded, err := held.store.Load(ctx, held.provider)
	if err == nil && loaded != nil {
		if moved := storeMovedOn(held, loaded, own); moved != nil {
			return moved
		}
		// The record holds this session's token, or the one its unsaved
		// rotation spent, or its access token, or this sign-in's pasted one:
		// its answers are the user's latest (0075-MADR D3).
		next.Consents = loaded.Consents
	}
	return held.store.Save(ctx, held.provider, next)
}

// persistRotation retries saving a rotated session, under the store's
// refresh lock, and only while the store still holds the refresh token it is
// known to hold (stored), the rotation itself, or this sign-in's access-only
// record: storeMovedOn is the test. It tries the lock once, with a context
// already done, so a call that holds a valid token never waits for another
// process's refresh (0021-MADR T10; 0026-MADR F1).
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
	return saveRotation(ctx, state, next, ownTokens{
		refresh: []string{stored, next.Refresh}, access: []string{state.access, next.Access}})
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
	// The issuer is a guard: a session from another issuer names no provider
	// by its kind, whatever its Kind.
	issuer := strings.TrimRight(s.Issuer, "/")
	switch kind := s.kindLocked(); {
	case openAIKind(kind) && issuer == DefaultOpenAIIssuer:
		return llmprovider.ProviderOpenAI
	case kind == SessionGrok && issuer == DefaultGrokOAuthIssuer:
		return llmprovider.ProviderGrok
	}
	return ""
}

// SessionKind returns the session's kind: Kind when it is set, else the kind
// the session's fields name by the rules every session had before Kind
// existed (a ChatGPT login, a plan session, a Grok login), else "" when they
// name none. It reads the fields under the session's lock, since a refresh
// rewrites them, and returns no error (0074-MADR D5; R51, R52).
func (s *OAuthSession) SessionKind() SessionKind {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.kindLocked()
}

// kindLocked is SessionKind for a caller that holds the session's lock.
func (s *OAuthSession) kindLocked() SessionKind {
	return sessionKindOf(s.Kind, s.Provider, s.Issuer, s.Resource)
}

// sessionKindOf is kind when it is set, else inferKind's.
func sessionKindOf(kind SessionKind, provider llmprovider.ProviderID, issuer, resource string) SessionKind {
	if kind != "" {
		return kind
	}
	return inferKind(provider, issuer, resource)
}

// inferKind is the kind of a session whose Kind is empty: today's rules, which
// the seven dispatch sites once each held. A plan session is an OpenAI one
// with a resource; a ChatGPT session is issued by OpenAI's own issuer, or has
// none and is OpenAI's; a Grok session is xAI's, or has none and is Grok's.
// A Kilo session, and any other, names no kind.
func inferKind(provider llmprovider.ProviderID, issuer, resource string) SessionKind {
	issuer = strings.TrimRight(issuer, "/")
	switch {
	case resource != "":
		if provider == llmprovider.ProviderOpenAI || provider == "" && issuer == DefaultOpenAIIssuer {
			return SessionChatGPTPlan
		}
	case issuer == DefaultOpenAIIssuer || issuer == "" && provider == llmprovider.ProviderOpenAI:
		return SessionChatGPT
	}
	if issuer == DefaultGrokOAuthIssuer || issuer == "" && provider == llmprovider.ProviderGrok {
		return SessionGrok
	}
	return ""
}

// openAIKind reports a kind issued by OpenAI.
func openAIKind(kind SessionKind) bool {
	return kind == SessionChatGPT || kind == SessionChatGPTPlan
}

// ChatGPT reports whether the session is a ChatGPT login for the Codex
// backend: issued by the default OpenAI issuer, and not a plan session, whose
// requests go to the platform (0043-MADR D1).
func (s *OAuthSession) ChatGPT() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	// An OpenAI session with no issuer is an older ChatGPT sign-in, as its
	// refresh treats it (refreshTokenURL), never a platform key (0026-MADR F2).
	// The issuer stays a guard, as at refreshTokenURL: a ChatGPT session from
	// another issuer is not one for the Codex backend, whatever its Kind.
	issuer := strings.TrimRight(s.Issuer, "/")
	return s.kindLocked() == SessionChatGPT && (issuer == DefaultOpenAIIssuer || issuer == "")
}

// PlanUsage reports whether the session is a Sign in with ChatGPT plan
// session, from LoginChatGPTPlan: an OpenAI session with a Resource. Its
// requests use the user's ChatGPT plan on the platform's Responses API
// (0043-MADR D1).
func (s *OAuthSession) PlanUsage() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.kindLocked() == SessionChatGPTPlan
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
	// access is the session's access token when the state was taken: with
	// its refresh tokens, what a stored record must not carry for it to be
	// another sign-in's (ownTokens; 0082-MADR amendment of 2026-11-17, R-5).
	access string
	// stored is the refresh token the store is known to hold. While an
	// unsaved rotation is pending it is a token this session spent: a store
	// that still holds it has not moved on, and it must never be adopted and
	// sent again (0020-MADR F1; 0026-MADR F1).
	stored    string
	issuer    string
	clientID  string
	accountID string
	fedramp   bool
	tokenURL  string
	// resource, scope, idToken and hostID are a plan session's (0043-MADR
	// D1); resource set selects its form refresh and revocation (D2).
	resource string
	scope    string
	idToken  string
	hostID   string
	// consents is the session's answers, or the stored record's when a
	// refresh keeps the store's (0075-MADR D3).
	consents string
	// kind is the session's kind: its Kind when set, else inferKind's.
	// declared is its Kind alone, which a refreshed session keeps, so that
	// a refresh never writes an inferred kind into a record.
	kind       SessionKind
	declared   SessionKind
	store      TokenStore
	httpClient *http.Client
	now        sessionClock
	// identity names the application on the refresh's and the revocation's
	// requests; its zero value names this module (0079-MADR D3).
	identity Identity
}

// kindOf is the state's kind: the one refreshState resolved, else the kind its
// fields name, for a state built from fields alone.
func (st oauthSessionState) kindOf() SessionKind {
	return sessionKindOf(st.kind, st.provider, st.issuer, st.resource)
}

func (s *OAuthSession) refreshState() oauthSessionState {
	return oauthSessionState{
		provider:   s.Provider,
		refresh:    s.Refresh,
		access:     s.Access,
		issuer:     s.Issuer,
		clientID:   s.ClientID,
		accountID:  s.AccountID,
		fedramp:    s.FedRAMP,
		tokenURL:   s.TokenURL,
		resource:   s.Resource,
		scope:      s.Scope,
		idToken:    s.IDToken,
		hostID:     s.HostID,
		consents:   s.Consents,
		kind:       s.kindLocked(),
		declared:   s.Kind,
		store:      s.Store,
		httpClient: s.HTTPClient,
		now:        s.now,
		identity:   s.identity,
	}
}

// UseHTTPClient gives the session a caller's client, so its refreshes use the
// same transport as the provider's requests and listing (0016-MADR D8). It
// replaces a provider's default (UseDefaultHTTPClient), but never another
// caller's client, nor one set on the HTTPClient field; a nil client changes
// nothing. The built-in providers call it with a WithHTTPClient client, so a
// session shared by several providers takes the first caller's client,
// whichever provider was built first (0026-MADR F37).
func (s *OAuthSession) UseHTTPClient(client *http.Client) {
	if client == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.HTTPClient == nil || s.clientDefault {
		s.HTTPClient, s.clientDefault = client, false
	}
}

// UseDefaultHTTPClient gives the session a provider's default client, when it
// has none, so that a provider built without WithHTTPClient refreshes the
// session through its own client (0016-MADR D8). The client is marked a
// default: a caller's, given later through UseHTTPClient, replaces it
// (0026-MADR F37). A nil client changes nothing.
func (s *OAuthSession) UseDefaultHTTPClient(client *http.Client) {
	if client == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.HTTPClient == nil {
		s.HTTPClient, s.clientDefault = client, true
	}
}

// UseIdentity gives the session the identity of the application using it, so
// its refreshes and its revocation, and the discovery a revocation reads
// first, lead their User-Agent with the application's name, as a sign-in's
// requests do (0079-MADR D2, D3). A session that has an identity, from its
// sign-in or an earlier call, keeps it: the first name wins. A zero id
// changes nothing. The identity lives in memory only: no store keeps it, so a
// session loaded from one has none until it is given one. The built-in
// providers and catalog call it with the WithClientInfo they hold. It returns
// no error.
func (s *OAuthSession) UseIdentity(id Identity) {
	if id == (Identity{}) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.identity == (Identity{}) {
		s.identity = id
	}
}

// UseLogger gives the session logger, when it has none, so a rotated session
// that could not be saved is reported through the provider's WithLogger
// logger (0016-MADR D4; 0020-MADR F48). A session that has a logger keeps it;
// a nil logger changes nothing. The built-in providers call it only with a
// caller's WithLogger logger, never their discarding default (0026-MADR F37).
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

// UseStore binds the session to store, so a refresh that rotates its tokens
// saves them there, as the Store field does when it is set before the session
// is shared. Unlike the field, it is safe to call while another goroutine
// uses the session, such as a listing that is still refreshing it
// (0074-PLAN Phase 7 step 6). The last call wins, as an assignment to the
// field does: the caller names the store that is due the session, whatever
// the sign-in carried. A nil store changes nothing. It returns no error.
func (s *OAuthSession) UseStore(store TokenStore) {
	if store == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Store = store
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
	s.Resource = next.Resource
	s.Scope = next.Scope
	s.IDToken = next.IDToken
	s.HostID = next.HostID
	s.Kind = next.Kind
	s.Consents = next.Consents
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
// oauthRefreshAttemptTimeout. retry reports a transport error other than a
// refused certificate (0068-MADR D2), 429 or 5xx.
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
		// A refused certificate is not retried: the next attempt meets the
		// same one (0068-MADR D2).
		return nil, llmprovider.Token{}, !transport.CertificateRefused(err), fmt.Errorf("%w: oauth: refresh request: %w", llmprovider.ErrProviderUnavailable,
			transport.WrapOffline(ctx, err, llmprovider.ErrOffline))
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
		Resource:   state.resource,
		Scope:      cmp.Or(payload.Scope, state.scope),
		IDToken:    cmp.Or(payload.IDToken, state.idToken),
		HostID:     state.hostID,
		Consents:   state.consents,
		Kind:       state.declared,
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
// A plan session's is a form with its resource and no scope, which keeps the
// grant (0043-MADR D2; OpenAI's profiles-and-sessions page).
func newRefreshRequest(ctx context.Context, state oauthSessionState) (*http.Request, error) {
	params := map[string]string{
		"grant_type":      oauthRefreshToken,
		oauthRefreshToken: state.refresh,
		"client_id":       state.clientID,
	}
	if state.resource != "" {
		params[planParamResource] = state.resource
	}
	contentType := "application/x-www-form-urlencoded"
	var body io.Reader
	if state.kindOf() == SessionChatGPT && strings.TrimRight(state.issuer, "/") == DefaultOpenAIIssuer {
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
	state.identity.setUserAgent(req)
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
// A stored session with a different refresh token is adopted when it is the
// same sign-in, and refreshed only if its own access token is due (MADR 0012
// §5.2; 0082-MADR D2).
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
//
// A stored session is adopted, at either read, only when it is the same
// sign-in as the held session's: sameSignIn says what that is. A record of
// another sign-in, another application's login, another kind, client, issuer,
// account or provider, with a refresh token or without, is refused with an
// error matching llmprovider.ErrAuthFailure before its token is used, its
// refresh token sent or anything saved, so the caller adopts nothing and the
// session keeps its own fields (0082-MADR D1, D3, and its amendment of
// 2026-11-17). Both reads compare with held, the state this call was given,
// the held session's, even after the first read took up a due rotation. The
// refresh's own save is guarded too: under the lock, saveRotation re-reads the
// store and writes only if it still holds this sign-in's token or nothing, and
// otherwise keeps the rotation in memory unsaved (R-1). The guards read only
// the states, copies the caller took under the session's lock, and the
// sessions the store returned, which this call alone holds; they take no lock
// of the session's.
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
	held := state
	stored, kept, keep, err := loadRotated(ctx, held, state)
	if err != nil {
		return refreshed{}, err
	}
	if stored != nil {
		if token, ok := stored.currentToken(); ok {
			return refreshed{next: stored, token: token}, nil
		}
		state = stored.refreshState()
	} else if keep {
		state.consents = kept
	}
	next, token, err := refreshOAuthSession(ctx, state)
	if errors.Is(err, llmprovider.ErrAuthFailure) {
		stored, _, _, reErr := loadRotated(ctx, held, state)
		if reErr != nil {
			return refreshed{}, reErr
		}
		if stored != nil {
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
		out.saveErr = saveRotation(ctx, held, next, ownTokens{
			refresh: []string{state.refresh, state.stored, next.Refresh}, access: []string{held.access}})
	}
	return out, nil
}

// loadRotated reads the store for state and returns the stored session to
// adopt: the store holds a refresh token other than state's, else nil. The
// token the store is known to hold is not another: while a rotation is unsaved
// it is one this session spent, however many saves failed (0020-MADR F1;
// 0026-MADR F1). A record it passes over because it holds the session's own
// token, or the one an unsaved rotation spent, still holds the user's latest
// answers: keep reports it, with its consents, which the session about to be
// saved takes (0075-MADR D3).
//
// A record is not adopted for lack of a refresh token, but it is still judged:
// a record that holds neither of the session's tokens and is not held's
// sign-in (sameSignIn, held being the held session's state) is another
// sign-in, whatever it holds, and the error is errOtherSignIn, matching
// llmprovider.ErrAuthFailure (0082-MADR D1 and its amendment of 2026-11-17,
// R-2). A record that holds the session's own refresh token, or its own access
// token as the session held it before this refresh (held.access), is the
// session's whatever its other fields say, so a store that drops a field,
// even the refresh token, still refreshes as before (R-5). A same-sign-in
// record with no refresh token is passed over: the held session refreshes its
// own and saves over it.
func loadRotated(ctx context.Context, held, state oauthSessionState) (rotated *OAuthSession, consents string, keep bool, err error) {
	if state.store == nil {
		return nil, "", false, nil
	}
	stored, loadErr := state.store.Load(ctx, state.provider)
	if loadErr != nil {
		stored = nil // a failed read adopts nothing (0016-MADR D4)
	}
	if stored == nil {
		return nil, "", false, nil
	}
	if (ownTokens{refresh: []string{state.refresh, state.stored}, access: []string{held.access}}).holds(stored) {
		return nil, stored.Consents, true, nil
	}
	if !sameSignIn(held, stored) {
		return nil, "", false, errOtherSignIn(held, stored)
	}
	if stored.Refresh == "" {
		return nil, "", false, nil
	}
	stored.Store, stored.HTTPClient, stored.now = state.store, state.httpClient, state.now
	return stored, "", false, nil
}

// sameSignIn reports whether stored, a session the store holds, is the sign-in
// held describes: signInDiff names no field that differs.
func sameSignIn(held oauthSessionState, stored *OAuthSession) bool {
	return signInDiff(held, stored) == ""
}

// signInDiff names the first field by which stored, a session the store holds,
// is not the sign-in held describes, else "". The fields are the provider (a
// stored session that names none is taken to name held's), the kind (a stored
// session with no Kind has the one SessionKind infers, from the provider
// above), the issuer (a trailing slash is ignored; an empty one is the kind's
// built-in issuer, as refreshTokenURL, ChatGPT and inferKind read it; the case
// of the host is not folded, as nowhere else in the module), the client (an
// empty one is never the same as a named one) and, only when both name one,
// the account. Tokens, expiry, scope, ID token and consents change in a
// legitimate rotation and are not compared, nor are the token URL, resource
// and host id, which follow from the kind and the client (0082-MADR D2). A
// refresh keeps every field compared, and FileTokenStore keeps them, so a
// sibling's rotation of this sign-in always passes. The fields are the store's
// labels, which whatever writes the store sets: the check stops another
// application's login written honestly, not a record forged to look like this
// one. The names it returns are the words a refusal uses, never a value.
func signInDiff(held oauthSessionState, stored *OAuthSession) string {
	if held.provider != "" && stored.Provider != "" && stored.Provider != held.provider {
		return "provider"
	}
	kind := held.kindOf()
	if sessionKindOf(stored.Kind, cmp.Or(stored.Provider, held.provider), stored.Issuer, stored.Resource) != kind {
		return "kind"
	}
	if signInIssuer(kind, stored.Issuer) != signInIssuer(kind, held.issuer) {
		return "issuer"
	}
	if stored.ClientID != held.clientID {
		return "client"
	}
	if stored.AccountID != "" && held.accountID != "" && stored.AccountID != held.accountID {
		return "account"
	}
	return ""
}

// signInIssuer is issuer without a trailing slash, or, when it is empty, the
// built-in issuer of kind (builtinIssuer).
func signInIssuer(kind SessionKind, issuer string) string {
	if issuer = strings.TrimRight(issuer, "/"); issuer != "" {
		return issuer
	}
	return builtinIssuer(kind)
}

// builtinIssuer is the issuer a session of kind has when it names none: an
// older session, or one a caller built, of OpenAI's or xAI's own kinds, as
// refreshTokenURL reads it. Any other kind has none.
func builtinIssuer(kind SessionKind) string {
	switch {
	case openAIKind(kind):
		return DefaultOpenAIIssuer
	case kind == SessionGrok:
		return DefaultGrokOAuthIssuer
	}
	return ""
}

// errOtherSignIn is the refusal to use a stored session that is not held's
// sign-in: it matches llmprovider.ErrAuthFailure and names the held session's
// provider and the one field by which the stored session differs, by its name,
// never a value, so that nothing the store holds reaches the text: no token,
// account, client, issuer or kind (0082-MADR D3 and its amendment of
// 2026-11-17, N1).
func errOtherSignIn(held oauthSessionState, stored *OAuthSession) error {
	return fmt.Errorf("oauth: the store holds another sign-in than this %s session's (its %s differs); sign in again: %w",
		cmp.Or(string(held.provider), "OAuth"), signInDiff(held, stored), llmprovider.ErrAuthFailure)
}

func refreshTokenURL(state oauthSessionState) (string, error) {
	if state.tokenURL != "" {
		return state.tokenURL, nil
	}
	// Without a token URL, only OpenAI's and xAI's own issuers have a known
	// endpoint. A session with no issuer is an older one of those two
	// providers. Any other issuer's refresh token is never sent elsewhere
	// (0020-MADR F45).
	// The kind chooses the endpoint; the issuer stays a guard, so that no kind
	// sends a refresh token to a built-in endpoint from another issuer
	// (0074-MADR D5).
	kind := state.kindOf()
	switch issuer := signInIssuer(kind, state.issuer); {
	case openAIKind(kind) && issuer == DefaultOpenAIIssuer:
		return DefaultOpenAIIssuer + "/oauth/token", nil
	case kind == SessionGrok && issuer == DefaultGrokOAuthIssuer:
		return defaultGrokOAuthRefreshURL, nil
	default:
		return "", fmt.Errorf("oauth: the %s session from %q has no token URL; sign in again: %w",
			state.provider, state.issuer, llmprovider.ErrAuthFailure)
	}
}
