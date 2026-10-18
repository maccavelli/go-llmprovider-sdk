package auth

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// RevokeOAuthSession revokes an SDK-owned session at its issuer, so a
// logout ends the refresh-token family rather than only forgetting it (MADR
// 0012 §5.3). It revokes the refresh token when there is one, else the access
// token. It does not touch any TokenStore: the caller deletes its copy
// whether or not revocation succeeds, as Codex does
// (codex login/src/auth/revoke.rs:1-5). Never pass a vendor CLI's session:
// revoking it would log the CLI out.
//
// OpenAI: a JSON request to {issuer}/oauth/revoke (revoke.rs:31-53, 104-114).
// Grok: an RFC 7009 form request to the discovery document's
// revocation_endpoint; without one the result wraps errors.ErrUnsupported.
func RevokeOAuthSession(ctx context.Context, session *OAuthSession) error {
	if session == nil {
		return errors.New("oauth: revoke: nil session")
	}
	session.mu.Lock()
	state := session.refreshState()
	access := session.Access
	session.mu.Unlock()

	token, hint := state.refresh, oauthRefreshToken
	if token == "" {
		token, hint = access, "access_token"
	}
	if token == "" {
		return errors.New("oauth: revoke: session holds no token")
	}
	client := state.httpClient
	if client == nil {
		client = transport.DefaultClient()
	}

	// Revocation goes to the session's own issuer only; a provider without
	// it sends nothing (0020-MADR F45).
	var req *http.Request
	var err error
	switch {
	case state.resource != "":
		req, err = planRevokeRequest(ctx, state, client, token, hint)
	case state.provider == llmprovider.ProviderOpenAI || strings.TrimRight(state.issuer, "/") == DefaultOpenAIIssuer:
		req, err = openAIRevokeRequest(ctx, state, token, hint)
	case state.provider == llmprovider.ProviderGrok:
		req, err = grokRevokeRequest(ctx, state, client, token, hint)
	default:
		return fmt.Errorf("oauth: revoke: %s sessions have no revocation: %w", state.provider, errors.ErrUnsupported)
	}
	if err != nil {
		return err
	}
	transport.NewIdentity("", "", "").SetUserAgent(req)
	resp, err := doBounded(client, req)
	if err != nil {
		return fmt.Errorf("oauth: revoke request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return oauthHTTPStatusError("revoke", resp)
	}
	return resp.Body.Close()
}

// openAIRevokeRequest posts Codex's JSON body. The endpoint is the token
// URL's origin with path /oauth/revoke, as Codex derives it from its refresh
// URL (revoke.rs:134-150), else the issuer's.
func openAIRevokeRequest(ctx context.Context, state oauthSessionState, token, hint string) (*http.Request, error) {
	endpoint := DefaultOpenAIIssuer + "/oauth/revoke"
	if issuer := strings.TrimRight(state.issuer, "/"); issuer != "" {
		endpoint = issuer + "/oauth/revoke"
	}
	if state.tokenURL != "" {
		u, err := url.Parse(state.tokenURL)
		if err != nil {
			return nil, fmt.Errorf("oauth: revoke: parse token URL: %w", err)
		}
		u.Path, u.RawQuery, u.Fragment = "/oauth/revoke", "", ""
		endpoint = u.String()
	}
	body := map[string]string{"token": token, "token_type_hint": hint}
	if hint == oauthRefreshToken {
		body[oauthParamClientID] = state.clientID
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("oauth: revoke: encode: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("oauth: revoke: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// grokRevokeRequest posts an RFC 7009 form to the issuer's discovered
// revocation_endpoint (https://auth.x.ai/oauth2/revoke on 2026-09-27).
func grokRevokeRequest(ctx context.Context, state oauthSessionState, client *http.Client, token, hint string) (*http.Request, error) {
	// An older Grok session with no issuer refreshes against xAI's
	// (refreshTokenURL), and is revoked there (0026-MADR F45).
	issuer := strings.TrimRight(state.issuer, "/")
	if issuer == "" {
		issuer = DefaultGrokOAuthIssuer
	}
	endpoints, err := oauthEndpointsFor(ctx, oauthFlowConfig{provider: llmprovider.ProviderGrok, issuer: issuer, httpClient: client})
	if err != nil {
		return nil, err
	}
	if endpoints.Revocation == "" {
		return nil, fmt.Errorf("oauth: revoke: %s publishes no revocation_endpoint: %w", issuer, errors.ErrUnsupported)
	}
	return formRevokeRequest(ctx, endpoints.Revocation, state.clientID, token, hint)
}

// planRevokeRequest posts an RFC 7009 form to the revocation_endpoint of the
// plan session's issuer, with its issued client id; an endpoint off the
// issuer's origin is refused (0043-MADR D2, A2). OpenAI answers an empty 200,
// also for a token already invalid.
func planRevokeRequest(ctx context.Context, state oauthSessionState, client *http.Client, token, hint string) (*http.Request, error) {
	issuer := cmp.Or(strings.TrimRight(state.issuer, "/"), DefaultOpenAIIssuer)
	endpoints, err := planOAuthEndpoints(ctx, client, issuer)
	if err != nil {
		return nil, err
	}
	if endpoints.Revocation == "" {
		return nil, fmt.Errorf("oauth: revoke: %s publishes no revocation_endpoint: %w", issuer, errors.ErrUnsupported)
	}
	return formRevokeRequest(ctx, endpoints.Revocation, state.clientID, token, hint)
}

// formRevokeRequest is an RFC 7009 revocation form for token, with its hint
// and the client id.
func formRevokeRequest(ctx context.Context, endpoint, clientID, token, hint string) (*http.Request, error) {
	form := url.Values{"token": {token}, "token_type_hint": {hint}, oauthParamClientID: {clientID}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("oauth: revoke: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}
