package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// OpenAI's Sign in with ChatGPT for agents (0043-MADR): a plan session's
// constants, its issuer's endpoints and its login.
const (
	// planResource is the resource a plan session's tokens are granted for:
	// the platform's API, where they are used.
	planResource = "https://api.openai.com/v1"
	// planParamResource names it in the authorize, token and refresh requests.
	planParamResource = "resource"
	// planDynamicClientID is the first sign-in's client: registration's entry
	// point, never the client a session keeps.
	planDynamicClientID = "dynamic_agent_client"
	// planDirectScope grants ChatGPT plan usage; a valid ID token alone does
	// not.
	planDirectScope = "chatgpt.tokens.use.direct"
	planScopes      = "openid profile email offline_access resource.invoke " + planDirectScope
	// planCallbackPath is the loopback redirect's path; only its port varies.
	planCallbackPath = "/auth/callback"
)

// planBuiltinEndpoints are OpenAI's, read from its discovery document on
// 2026-10-18 (0043-MADR M9). They fill in for the built-in issuer only, when
// its discovery fails or leaves one out, as the built-in values do for the
// other logins (0016-MADR D7).
var planBuiltinEndpoints = oauthEndpoints{
	Authorization: DefaultOpenAIIssuer + "/api/accounts/authorize",
	Token:         DefaultOpenAIIssuer + "/api/accounts/oauth/token",
	Revocation:    DefaultOpenAIIssuer + "/api/accounts/oauth/revoke",
	JWKS:          defaultOpenAIJWKSURL,
	SigningAlgs:   []string{algRS256},
}

// ChatGPTPlanOptions configures LoginChatGPTPlan (0043-MADR D4).
type ChatGPTPlanOptions struct {
	// AgentName is the app's own name, shown to the user at sign-in and kept
	// the same across installations. It is required: OpenAI's terms ask an
	// app to sign in under its own name, never another app's.
	AgentName string
	// HostID identifies this host to OpenAI, as ext_agent_host_id: a
	// urn:uuid, urn:ietf:params:oauth:jwk-thumbprint or did:key value kept
	// for the host's life, never an email or a user id. Empty takes
	// Previous's, else a new random urn:uuid, which the session keeps.
	HostID string
	// Previous is a plan session to sign in to again: its issued client id,
	// host id and ID token are reused, and no client is registered.
	Previous *OAuthSession
	// HTTPClient, OpenURL and InputCode are as OAuthFlowOptions's. InputCode
	// takes the callback URL pasted when the browser cannot reach the
	// loopback listener; a first sign-in needs the whole URL, which names the
	// issued client.
	HTTPClient *http.Client
	OpenURL    func(string) error
	InputCode  func(context.Context) (string, error)

	// issuer is a test's fake issuer; empty is DefaultOpenAIIssuer.
	issuer string
	now    func() time.Time
}

// LoginChatGPTPlan signs a user in with their ChatGPT plan, under the app's
// own name, by OpenAI's Sign in with ChatGPT for agents (0043-MADR D4). The
// first sign-in registers a client for the app under the user's account;
// with opts.Previous it signs in to that client again. The session it returns
// is a plan session (PlanUsage): the openai provider sends its requests to
// the platform's Responses API, billed to the plan.
//
// The flow is a browser authorization-code flow with PKCE and a nonce, on a
// loopback redirect at 127.0.0.1. Every endpoint comes from the issuer's
// discovery document and must be on its origin. The ID token is verified,
// and a grant without plan usage (chatgpt.tokens.use.direct) is refused,
// matching llmprovider.ErrNotPermitted. There is no device flow.
//
// What OpenAI's terms ask of the app itself, the SDK cannot do for it:
// requests from the user's own runtime, consent before any background use,
// the plan used only for the app the user connected, no pooling or sharing,
// and no charge.
func LoginChatGPTPlan(ctx context.Context, opts ChatGPTPlanOptions) (*OAuthSession, error) {
	config, err := resolvePlanConfig(opts)
	if err != nil {
		return nil, err
	}
	hostID, err := planHostID(opts)
	if err != nil {
		return nil, err
	}
	endpoints, err := planOAuthEndpoints(ctx, config.httpClient, config.issuer)
	if err != nil {
		return nil, err
	}
	if err := requireEndpoints(config.issuer, endpoints.Authorization, endpoints.Token); err != nil {
		return nil, err
	}
	pkce, err := newPKCE()
	if err != nil {
		return nil, err
	}
	state, err := randomBase64URL()
	if err != nil {
		return nil, fmt.Errorf("oauth: generate state: %w", err)
	}
	nonce, err := randomBase64URL()
	if err != nil {
		return nil, fmt.Errorf("oauth: generate nonce: %w", err)
	}

	listeners, redirectURI, _, err := browserListener(llmprovider.ProviderOpenAI)
	if err != nil {
		return nil, err
	}
	flowCtx, cancel := context.WithTimeout(ctx, oauthBrowserTimeout)
	defer cancel()
	result := make(chan oauthCallbackResult, 1)
	serveErrors := make(chan error, 1)
	shutdown := serveCallbackListeners(oauthCallbackHandler(planCallbackPath, state, result, ""), listeners, serveErrors)
	defer shutdown()

	authorizeURL, err := planAuthorizeURL(endpoints.Authorization, opts, hostID, redirectURI, pkce.challenge, state, nonce)
	if err != nil {
		return nil, err
	}
	if config.openURL != nil {
		go func() {
			ignoreOAuthError(config.openURL(authorizeURL))
		}()
	}
	inputCancel := func() {}
	if config.inputCode != nil {
		var inputCtx context.Context
		inputCtx, inputCancel = context.WithCancel(flowCtx)
		go func() {
			callback := oauthCallbackResult{}
			input, inputErr := config.inputCode(inputCtx)
			if inputErr != nil {
				callback.err = inputErr
			} else {
				callback = parseOAuthCallback(input, state)
			}
			select {
			case result <- callback:
			case <-inputCtx.Done():
			}
		}()
	}
	defer inputCancel()

	var callback oauthCallbackResult
	select {
	case callback = <-result:
	case serveErr := <-serveErrors:
		return nil, serveErr
	case <-flowCtx.Done():
		return nil, fmt.Errorf("oauth: browser login timed out: %w", flowCtx.Err())
	}
	if callback.err != nil {
		return nil, callback.err
	}
	if config.clientID, err = planIssuedClient(opts.Previous, callback.clientID); err != nil {
		return nil, err
	}
	payload, err := planExchange(flowCtx, config, endpoints.Token, callback.code, redirectURI, pkce.verifier)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(strings.Fields(payload.Scope), planDirectScope) {
		return nil, fmt.Errorf("oauth: %w: the grant does not include ChatGPT plan usage (%s)", llmprovider.ErrNotPermitted, planDirectScope)
	}
	session, err := verifiedSession(flowCtx, config, endpoints, payload, nonce)
	if err != nil {
		return nil, err
	}
	session.Resource, session.Scope, session.IDToken, session.HostID = planResource, payload.Scope, payload.IDToken, hostID
	return session, nil
}

// resolvePlanConfig checks opts and fills in the defaults.
func resolvePlanConfig(opts ChatGPTPlanOptions) (oauthFlowConfig, error) {
	if strings.TrimSpace(opts.AgentName) == "" {
		return oauthFlowConfig{}, fmt.Errorf("oauth: %w: LoginChatGPTPlan needs the app's own name (AgentName)", llmprovider.ErrInvalidRequest)
	}
	if opts.Previous != nil && (!opts.Previous.PlanUsage() || opts.Previous.ClientID == "") {
		return oauthFlowConfig{}, fmt.Errorf("oauth: %w: Previous is not a plan session with an issued client", llmprovider.ErrInvalidRequest)
	}
	client := opts.HTTPClient
	if client == nil {
		client = transport.DefaultClient()
	}
	now := opts.now
	if now == nil {
		now = time.Now
	}
	issuer := strings.TrimRight(opts.issuer, "/")
	if issuer == "" {
		issuer = DefaultOpenAIIssuer
	}
	return oauthFlowConfig{
		provider:     llmprovider.ProviderOpenAI,
		issuer:       issuer,
		httpClient:   client,
		callerClient: opts.HTTPClient,
		openURL:      opts.OpenURL,
		inputCode:    opts.InputCode,
		now:          now,
	}, nil
}

// planHostID is opts.HostID, else Previous's, else a new random urn:uuid.
func planHostID(opts ChatGPTPlanOptions) (string, error) {
	switch {
	case strings.TrimSpace(opts.HostID) != "":
		return strings.TrimSpace(opts.HostID), nil
	case opts.Previous != nil && opts.Previous.HostID != "":
		return opts.Previous.HostID, nil
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("oauth: generate host id: %w", err)
	}
	b[6] = b[6]&0x0f | 0x40 // version 4
	b[8] = b[8]&0x3f | 0x80 // RFC 4122 variant
	return fmt.Sprintf("urn:uuid:%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// planAuthorizeURL is the authorize request: a first sign-in names the app
// and registers through dynamic_agent_client; a re-sign-in names its issued
// client and hints its last ID token, without the app's name.
func planAuthorizeURL(endpoint string, opts ChatGPTPlanOptions, hostID, redirectURI, challenge, state, nonce string) (string, error) {
	authorizeURL, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("oauth: parse authorization endpoint: %w", err)
	}
	query := authorizeURL.Query()
	query.Set("response_type", "code")
	query.Set("redirect_uri", redirectURI)
	query.Set("scope", planScopes)
	query.Set(planParamResource, planResource)
	query.Set("code_challenge", challenge)
	query.Set("code_challenge_method", "S256")
	query.Set("state", state)
	query.Set("nonce", nonce)
	query.Set("ext_agent_host_id", hostID)
	if previous := opts.Previous; previous != nil {
		query.Set(oauthParamClientID, previous.ClientID)
		if previous.IDToken != "" {
			query.Set("id_token_hint", previous.IDToken)
		}
	} else {
		query.Set(oauthParamClientID, planDynamicClientID)
		query.Set("agent_name_hint", strings.TrimSpace(opts.AgentName))
	}
	authorizeURL.RawQuery = query.Encode()
	return authorizeURL.String(), nil
}

// planIssuedClient is the client the session keeps: the callback's issued
// client on a first sign-in, which it must name; Previous's on a re-sign-in,
// which a callback naming another client contradicts.
func planIssuedClient(previous *OAuthSession, fromCallback string) (string, error) {
	if previous != nil {
		if fromCallback != "" && fromCallback != previous.ClientID {
			return "", errors.New("oauth: the callback names another client than the session being signed in to again")
		}
		return previous.ClientID, nil
	}
	if fromCallback == "" || fromCallback == planDynamicClientID {
		return "", errors.New("oauth: the callback carries no issued client_id: registration did not complete")
	}
	return fromCallback, nil
}

// planExchange trades the code for tokens with the issued client and the
// resource.
func planExchange(ctx context.Context, config oauthFlowConfig, tokenURL, code, redirectURI, verifier string) (oauthTokenResponse, error) {
	form := url.Values{
		oauthParamGrantType: {oauthGrantAuthorizationCode},
		"code":              {code},
		"redirect_uri":      {redirectURI},
		oauthParamClientID:  {config.clientID},
		"code_verifier":     {verifier},
		planParamResource:   {planResource},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return oauthTokenResponse{}, fmt.Errorf("oauth: create token request: %w", err)
	}
	transport.NewIdentity("", "", "").SetUserAgent(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := doBounded(config.httpClient, req)
	if err != nil {
		return oauthTokenResponse{}, fmt.Errorf("oauth: token request: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return oauthTokenResponse{}, oauthHTTPStatusError("token exchange", resp)
	}
	var payload oauthTokenResponse
	if err := decodeOAuthResponse(resp, &payload); err != nil {
		return oauthTokenResponse{}, fmt.Errorf("oauth: decode token response: %w", err)
	}
	return payload, nil
}

// planOAuthEndpoints returns a plan session's endpoints from issuer's
// discovery document, as published: unlike the Codex login's, its authorize
// and token endpoints are not replaced. Every endpoint must be on the
// issuer's origin (0043-MADR A2).
func planOAuthEndpoints(ctx context.Context, client *http.Client, issuer string) (oauthEndpoints, error) {
	discovered, err := discoverOAuthEndpoints(ctx, oauthFlowConfig{provider: llmprovider.ProviderOpenAI, issuer: issuer, httpClient: client})
	if issuer == DefaultOpenAIIssuer {
		if err != nil {
			discovered, err = planBuiltinEndpoints, nil
		}
		fillOAuthEndpoints(&discovered, planBuiltinEndpoints)
		if discovered.Revocation == "" {
			discovered.Revocation = planBuiltinEndpoints.Revocation
		}
	}
	if err != nil {
		return oauthEndpoints{}, err
	}
	if err := requireSameOrigin(issuer, discovered.Authorization, discovered.Token, discovered.Revocation, discovered.JWKS); err != nil {
		return oauthEndpoints{}, err
	}
	return discovered, nil
}

// requireSameOrigin fails when an endpoint is not on issuer's scheme and
// host, so a code or a refresh token is never posted elsewhere (0043-MADR
// A2). An empty endpoint is left to the caller's presence check.
func requireSameOrigin(issuer string, endpoints ...string) error {
	want, err := url.Parse(issuer)
	if err != nil || want.Scheme == "" || want.Host == "" {
		return fmt.Errorf("oauth: issuer %q is not an absolute URL", issuer)
	}
	for _, endpoint := range endpoints {
		if endpoint == "" {
			continue
		}
		u, err := url.Parse(endpoint)
		if err != nil || !strings.EqualFold(u.Scheme, want.Scheme) || !strings.EqualFold(u.Host, want.Host) {
			return fmt.Errorf("oauth: discovery for %s names an endpoint on another origin: %q", issuer, endpoint)
		}
	}
	return nil
}
