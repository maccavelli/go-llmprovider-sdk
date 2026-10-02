package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

const (
	oauthBrowserTimeout         = 10 * time.Minute
	oauthResponseLimit          = 1 << 20
	oauthParamClientID          = "client_id"
	oauthParamGrantType         = "grant_type"
	oauthGrantAuthorizationCode = "authorization_code"
	openAIOAuthScopes           = "openid profile email offline_access api.connectors.read api.connectors.invoke"
	grokOAuthScopes             = "openid profile email offline_access grok-cli:access api:access"
	// oauthParamOriginator names this module to OpenAI's authorize endpoint,
	// as the ChatGPT backend's originator header does (0002-MADR §5).
	oauthParamOriginator = "originator"
	originatorValue      = "go-llmprovider-sdk"
)

var openaiLoopbackPorts = []int{1455, 1457}

// OAuthFlowOptions supplies transport and user-interaction hooks for OAuth login.
type OAuthFlowOptions struct {
	HTTPClient   *http.Client
	OpenURL      func(string) error
	InputCode    func(context.Context) (string, error)
	NotifyDevice func(verificationURL, userCode string)
	ClientID     string
	Issuer       string

	now   func() time.Time
	sleep func(context.Context, time.Duration) error
}

type oauthFlowConfig struct {
	provider   llmprovider.ProviderID
	clientID   string
	issuer     string
	httpClient *http.Client
	openURL    func(string) error
	inputCode  func(context.Context) (string, error)
	notify     func(string, string)
	now        func() time.Time
	sleep      func(context.Context, time.Duration) error
}

type oauthEndpoints struct {
	Authorization string `json:"authorization_endpoint"`
	Token         string `json:"token_endpoint"`
	Device        string `json:"device_authorization_endpoint"`
	Revocation    string `json:"revocation_endpoint"`
	JWKS          string `json:"jwks_uri"`
	// SigningAlgs is id_token_signing_alg_values_supported (0016-MADR A3).
	SigningAlgs []string `json:"id_token_signing_alg_values_supported"`
}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type oauthCallbackResult struct {
	code string
	err  error
}

// LoginBrowserOAuth completes a browser authorization-code flow with PKCE.
func LoginBrowserOAuth(ctx context.Context, provider llmprovider.ProviderID, opts OAuthFlowOptions) (*OAuthSession, error) {
	config, err := resolveOAuthFlowConfig(provider, opts)
	if err != nil {
		return nil, err
	}

	endpoints, err := oauthEndpointsFor(ctx, config)
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
	state, err := randomBase64URL(32)
	if err != nil {
		return nil, fmt.Errorf("oauth: generate state: %w", err)
	}

	listeners, redirectURI, callbackPath, err := browserListener(config.provider)
	if err != nil {
		return nil, err
	}
	flowCtx, cancel := context.WithTimeout(ctx, oauthBrowserTimeout)
	defer cancel()
	result := make(chan oauthCallbackResult, 1)
	handler := oauthCallbackHandler(callbackPath, state, result, callbackCORSOrigin(config.provider))
	serveErrors := make(chan error, 1)
	shutdown := serveCallbackListeners(handler, listeners, serveErrors)
	defer shutdown()

	nonce := ""
	if config.provider == llmprovider.ProviderGrok {
		if nonce, err = randomBase64URL(32); err != nil {
			return nil, fmt.Errorf("oauth: generate nonce: %w", err)
		}
	}
	authorizeURL, err := buildAuthorizeURL(config, endpoints.Authorization, redirectURI, pkce.challenge, state, nonce)
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
			input, inputErr := config.inputCode(inputCtx)
			if inputErr == nil {
				input, inputErr = parseOAuthInput(input, state)
			}
			select {
			case result <- oauthCallbackResult{code: input, err: inputErr}:
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
	return exchangeOAuthCode(flowCtx, config, endpoints, callback.code, redirectURI, pkce.verifier, nonce)
}

func resolveOAuthFlowConfig(provider llmprovider.ProviderID, opts OAuthFlowOptions) (oauthFlowConfig, error) {
	issuer := strings.TrimRight(opts.Issuer, "/")
	clientID := opts.ClientID
	switch provider {
	case llmprovider.ProviderOpenAI:
		if issuer == "" {
			issuer = DefaultOpenAIIssuer
		}
		if clientID == "" {
			clientID = DefaultOpenAIClientID
		}
	case llmprovider.ProviderGrok:
		if issuer == "" {
			issuer = strings.TrimRight(os.Getenv("GROK_OAUTH2_ISSUER"), "/")
			if issuer == "" {
				issuer = DefaultGrokOAuthIssuer
			}
		}
		if clientID == "" {
			clientID = os.Getenv("GROK_OAUTH2_CLIENT_ID")
			if clientID == "" {
				clientID = DefaultGrokOAuthClientID
			}
		}
	default:
		return oauthFlowConfig{}, fmt.Errorf("oauth: provider %q is not supported", provider)
	}

	client := opts.HTTPClient
	if client == nil {
		client = transport.DefaultClient()
	}
	now := opts.now
	if now == nil {
		now = time.Now
	}
	sleep := opts.sleep
	if sleep == nil {
		sleep = sleepWithContext
	}
	return oauthFlowConfig{
		provider:   provider,
		clientID:   clientID,
		issuer:     issuer,
		httpClient: client,
		openURL:    opts.OpenURL,
		inputCode:  opts.InputCode,
		notify:     opts.NotifyDevice,
		now:        now,
		sleep:      sleep,
	}, nil
}

// oauthEndpointsFor returns the flow's endpoints and the issuer's keys, from
// OIDC discovery. OpenAI's authorize and token endpoints stay the fixed ones
// Codex uses; only its keys come from discovery. When discovery fails, or
// leaves a field out, the built-in values fill in for the built-in issuer
// only: a caller's issuer whose discovery fails is an error (0016-MADR D7).
func oauthEndpointsFor(ctx context.Context, config oauthFlowConfig) (oauthEndpoints, error) {
	builtin := builtinOAuthEndpoints(config)
	discovered, err := discoverOAuthEndpoints(ctx, config)
	if err != nil {
		if builtin == nil {
			return oauthEndpoints{}, err
		}
		return *builtin, nil
	}
	if config.provider == llmprovider.ProviderOpenAI {
		discovered.Authorization = config.issuer + "/oauth/authorize"
		discovered.Token = config.issuer + "/oauth/token"
	}
	if builtin != nil {
		fillOAuthEndpoints(&discovered, *builtin)
	}
	return discovered, nil
}

// requireEndpoints fails when a flow's endpoint is missing from discovery.
// A missing jwks_uri fails later, in verifyIDToken.
func requireEndpoints(issuer string, endpoints ...string) error {
	for _, e := range endpoints {
		if e == "" {
			return fmt.Errorf("oauth: discovery for %s lacks an endpoint this login needs", issuer)
		}
	}
	return nil
}

// builtinOAuthEndpoints returns the built-in issuer's endpoints and keys, or
// nil for any other issuer.
func builtinOAuthEndpoints(config oauthFlowConfig) *oauthEndpoints {
	issuer := strings.TrimRight(config.issuer, "/")
	switch {
	case config.provider == llmprovider.ProviderOpenAI && issuer == DefaultOpenAIIssuer:
		return &oauthEndpoints{
			Authorization: issuer + "/oauth/authorize",
			Token:         issuer + "/oauth/token",
			JWKS:          defaultOpenAIJWKSURL,
			SigningAlgs:   []string{algRS256},
		}
	case config.provider == llmprovider.ProviderGrok && issuer == DefaultGrokOAuthIssuer:
		return &oauthEndpoints{
			Authorization: issuer + "/oauth2/authorize",
			Token:         defaultGrokOAuthRefreshURL,
			Device:        defaultGrokOAuthDeviceURL,
			JWKS:          defaultGrokJWKSURL,
			SigningAlgs:   []string{algES256},
		}
	}
	return nil
}

func fillOAuthEndpoints(e *oauthEndpoints, from oauthEndpoints) {
	for _, pair := range []struct {
		dst *string
		src string
	}{
		{&e.Authorization, from.Authorization}, {&e.Token, from.Token}, {&e.Device, from.Device}, {&e.JWKS, from.JWKS},
	} {
		if *pair.dst == "" {
			*pair.dst = pair.src
		}
	}
	if len(e.SigningAlgs) == 0 {
		e.SigningAlgs = from.SigningAlgs
	}
}

func discoverOAuthEndpoints(ctx context.Context, config oauthFlowConfig) (oauthEndpoints, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.issuer+"/.well-known/openid-configuration", http.NoBody)
	if err != nil {
		return oauthEndpoints{}, fmt.Errorf("oauth: create discovery request: %w", err)
	}
	transport.NewIdentity("", "", "").SetUserAgent(req)
	resp, err := config.httpClient.Do(req)
	if err != nil {
		return oauthEndpoints{}, fmt.Errorf("oauth: discovery for %s: %w", config.issuer, err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		ignoreOAuthError(resp.Body.Close())
		return oauthEndpoints{}, fmt.Errorf("oauth: discovery for %s: %s", config.issuer, resp.Status)
	}
	var discovered oauthEndpoints
	if err := decodeOAuthResponse(resp, &discovered); err != nil {
		return oauthEndpoints{}, fmt.Errorf("oauth: decode discovery for %s: %w", config.issuer, err)
	}
	return discovered, nil
}

func browserListener(provider llmprovider.ProviderID) ([]net.Listener, string, string, error) {
	if provider == llmprovider.ProviderOpenAI {
		listeners, port, err := listenOpenAILoopback(openaiLoopbackPorts)
		if err != nil {
			return nil, "", "", err
		}
		// 127.0.0.1, as Codex redirects with the same client id since
		// 4b97832cfb (login/src/server.rs:193); MADR 0012 §5.3.
		return listeners, fmt.Sprintf("http://127.0.0.1:%d/auth/callback", port), "/auth/callback", nil
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", "", fmt.Errorf("oauth: listen for Grok callback: %w", err)
	}
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		ignoreOAuthError(listener.Close())
		return nil, "", "", errors.New("oauth: Grok callback listener has an unexpected address type")
	}
	port := address.Port
	return []net.Listener{listener}, fmt.Sprintf("http://127.0.0.1:%d/callback", port), "/callback", nil
}

func listenOpenAILoopback(ports []int) ([]net.Listener, int, error) {
	var failures []error
	for _, port := range ports {
		listeners, err := listenLoopbackBothFamilies(port)
		if err == nil {
			return listeners, port, nil
		}
		failures = append(failures, err)
	}
	return nil, 0, fmt.Errorf("oauth: registered callback ports unavailable; use device-code login: %w", errors.Join(failures...))
}

func listenLoopbackBothFamilies(port int) ([]net.Listener, error) {
	portText := fmt.Sprintf("%d", port)
	v4, err4 := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", portText))
	v6, err6 := net.Listen("tcp6", net.JoinHostPort("::1", portText))
	if err4 != nil && err6 != nil {
		return nil, errors.Join(err4, err6)
	}
	if err4 != nil && isAddrInUse(err4) {
		closeCallbackListeners(v6)
		return nil, err4
	}
	if err6 != nil && isAddrInUse(err6) {
		closeCallbackListeners(v4)
		return nil, err6
	}
	listeners := make([]net.Listener, 0, 2)
	if err4 == nil {
		listeners = append(listeners, v4)
	}
	if err6 == nil {
		listeners = append(listeners, v6)
	}
	if len(listeners) == 0 {
		return nil, errors.Join(err4, err6)
	}
	return listeners, nil
}

func listenFirstAvailable(host string, ports []int) (net.Listener, int, error) {
	var failures []error
	for _, port := range ports {
		listener, err := net.Listen("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)))
		if err == nil {
			return listener, port, nil
		}
		failures = append(failures, err)
	}
	return nil, 0, fmt.Errorf("oauth: registered callback ports unavailable; use device-code login: %w", errors.Join(failures...))
}

func serveCallbackListeners(handler http.Handler, listeners []net.Listener, serveErrors chan<- error) func() {
	servers := make([]*http.Server, 0, len(listeners))
	for _, listener := range listeners {
		server := &http.Server{
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		}
		servers = append(servers, server)
		go func(srv *http.Server, ln net.Listener) {
			serveErr := srv.Serve(ln)
			if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
				select {
				case serveErrors <- fmt.Errorf("oauth: callback server: %w", serveErr):
				default:
				}
			}
		}(server, listener)
	}
	return func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
		defer shutdownCancel()
		for _, server := range servers {
			ignoreOAuthError(server.Shutdown(shutdownCtx))
		}
	}
}

func closeCallbackListeners(listeners ...net.Listener) {
	for _, listener := range listeners {
		if listener != nil {
			ignoreOAuthError(listener.Close())
		}
	}
}

func isAddrInUse(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "address already in use") ||
		strings.Contains(msg, "only one usage of each socket address")
}

func buildAuthorizeURL(config oauthFlowConfig, endpoint, redirectURI, challenge, state, nonce string) (string, error) {
	authorizeURL, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("oauth: parse authorization endpoint: %w", err)
	}
	query := authorizeURL.Query()
	query.Set("response_type", "code")
	query.Set("client_id", config.clientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("code_challenge", challenge)
	query.Set("code_challenge_method", "S256")
	query.Set("state", state)
	if config.provider == llmprovider.ProviderOpenAI {
		query.Set("scope", openAIOAuthScopes)
		query.Set("id_token_add_organizations", "true")
		query.Set("codex_cli_simplified_flow", "true")
		query.Set(oauthParamOriginator, originatorValue)
	} else {
		query.Set("scope", grokOAuthScopes)
		query.Set("nonce", nonce)
		query.Set("referrer", "go-llmprovider-sdk")
	}
	authorizeURL.RawQuery = query.Encode()
	return authorizeURL.String(), nil
}

// grokAccountsAppOrigin is the only browser origin allowed to call the Grok
// loopback callback. accounts.x.ai delivers the code with a cross-origin,
// private-network request, as the official CLI's callback router allows
// (MADR 0008 D2).
const grokAccountsAppOrigin = "https://accounts.x.ai"

// callbackCORSOrigin names the origin a provider's callback allows; "" allows
// none.
func callbackCORSOrigin(provider llmprovider.ProviderID) string {
	if provider == llmprovider.ProviderGrok {
		return grokAccountsAppOrigin
	}
	return ""
}

// oauthCallbackHandler serves the loopback redirect. A preflight never
// completes the waiter; CORS headers go only to corsOrigin, never to "*".
func oauthCallbackHandler(path, state string, result chan<- oauthCallbackResult, corsOrigin string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, request *http.Request) {
		if corsOrigin != "" && request.Header.Get("Origin") == corsOrigin {
			header := w.Header()
			header.Set("Access-Control-Allow-Origin", corsOrigin)
			header.Set("Access-Control-Allow-Methods", http.MethodGet)
			header.Set("Access-Control-Allow-Private-Network", "true")
		}
		if request.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		query := request.URL.Query()
		if !callbackStateMatches(query.Get("state"), state) {
			http.Error(w, "State mismatch", http.StatusBadRequest)
			select {
			case result <- oauthCallbackResult{err: errors.New("oauth: callback state mismatch")}:
			default:
			}
			return
		}
		if authErr := callbackError(query); authErr != nil {
			http.Error(w, "Authorization failed", http.StatusBadRequest)
			select {
			case result <- oauthCallbackResult{err: authErr}:
			default:
			}
			return
		}
		code := query.Get("code")
		if code == "" {
			http.Error(w, "Missing authorization code", http.StatusBadRequest)
			select {
			case result <- oauthCallbackResult{err: errors.New("oauth: callback URL missing authorization code")}:
			default:
			}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if _, err := io.WriteString(w, "<!doctype html><html><body>You can close this window.</body></html>"); err != nil {
			select {
			case result <- oauthCallbackResult{err: fmt.Errorf("oauth: write callback response: %w", err)}:
			default:
			}
			return
		}
		select {
		case result <- oauthCallbackResult{code: code}:
		default:
		}
	})
	return mux
}

// lifeSciencesStateSuffix is the onboarding marker ChatGPT may append to the
// callback state (codex login/src/callback_params.rs:1).
const lifeSciencesStateSuffix = ".onboarding_entrypoint=life_sciences"

// callbackStateMatches accepts the expected state, or it followed by exactly
// lifeSciencesStateSuffix (codex login/src/server.rs:366-374).
func callbackStateMatches(received, expected string) bool {
	return received == expected || received == expected+lifeSciencesStateSuffix
}

// callbackErrorMessageLimit bounds the IdP's error_description.
const callbackErrorMessageLimit = 300

// callbackError reports an authorize error from the callback query, or nil.
// It carries error_description, and Codex's explanation for a workspace
// without Codex (codex login/src/server.rs:934-956).
func callbackError(query url.Values) error {
	code := query.Get("error")
	if code == "" {
		return nil
	}
	description := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, query.Get("error_description"))
	if len(description) > callbackErrorMessageLimit {
		description = description[:callbackErrorMessageLimit]
	}
	switch {
	case code == "access_denied" && strings.Contains(strings.ToLower(description), "missing_codex_entitlement"):
		return fmt.Errorf("oauth: authorization failed (%s): Codex is not enabled for your workspace; "+
			"ask your workspace administrator for access to Codex", code)
	case strings.TrimSpace(description) != "":
		return fmt.Errorf("oauth: authorization failed (%s): %s", code, description)
	default:
		return fmt.Errorf("oauth: authorization failed: %s", code)
	}
}

func parseOAuthInput(input, expectedState string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", errors.New("oauth: empty authorization code")
	}
	parsed, err := url.Parse(input)
	if err == nil && parsed.Scheme != "" {
		query := parsed.Query()
		if receivedState := query.Get("state"); receivedState != "" && !callbackStateMatches(receivedState, expectedState) {
			return "", errors.New("oauth: callback state mismatch")
		}
		if authErr := callbackError(query); authErr != nil {
			return "", authErr
		}
		if code := query.Get("code"); code != "" {
			return code, nil
		}
		return "", errors.New("oauth: callback URL missing authorization code")
	}
	if input != "" {
		return input, nil
	}
	return "", errors.New("oauth: empty authorization code")
}

func exchangeOAuthCode(
	ctx context.Context,
	config oauthFlowConfig,
	endpoints oauthEndpoints,
	code, redirectURI, verifier, nonce string,
) (*OAuthSession, error) {
	tokenURL := endpoints.Token
	form := url.Values{
		oauthParamGrantType: {oauthGrantAuthorizationCode},
		"code":              {code},
		"redirect_uri":      {redirectURI},
		oauthParamClientID:  {config.clientID},
		"code_verifier":     {verifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("oauth: create token request: %w", err)
	}
	transport.NewIdentity("", "", "").SetUserAgent(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := config.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth: token request: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, oauthHTTPStatusError("token exchange", resp)
	}
	var payload oauthTokenResponse
	if err := decodeOAuthResponse(resp, &payload); err != nil {
		return nil, fmt.Errorf("oauth: decode token response: %w", err)
	}
	return verifiedSession(ctx, config, endpoints, payload, nonce)
}

// verifiedSession verifies the token response's id_token, then builds the
// session from its claims (0016-MADR D7, A3). Every login requests openid, so
// a response without an id_token fails, and so does any verification failure:
// nothing is returned to save.
func verifiedSession(ctx context.Context, config oauthFlowConfig, endpoints oauthEndpoints, payload oauthTokenResponse, nonce string) (*OAuthSession, error) {
	if payload.IDToken == "" {
		return nil, fmt.Errorf("%w: the token response has none, although openid was requested", errIDToken)
	}
	check := idTokenCheck{issuer: config.issuer, clientID: config.clientID, nonce: nonce,
		jwksURL: endpoints.JWKS, advertised: endpoints.SigningAlgs}
	if err := verifyIDToken(ctx, config.httpClient, payload.IDToken, check, config.now()); err != nil {
		return nil, err
	}
	return oauthSessionFromResponse(config, endpoints.Token, payload)
}

func oauthSessionFromResponse(config oauthFlowConfig, tokenURL string, payload oauthTokenResponse) (*OAuthSession, error) {
	if payload.AccessToken == "" {
		return nil, errors.New("oauth: token response missing access token")
	}
	expiresIn := payload.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	return &OAuthSession{
		Provider:   config.provider,
		Access:     payload.AccessToken,
		Refresh:    payload.RefreshToken,
		Expiry:     tokenExpiry(payload.AccessToken, expiresIn, config.now()),
		Issuer:     config.issuer,
		ClientID:   config.clientID,
		AccountID:  chatGPTAccountID(payload.IDToken),
		FedRAMP:    chatGPTFedRAMP(payload.IDToken),
		TokenURL:   tokenURL,
		HTTPClient: config.httpClient,
	}, nil
}

func chatGPTAccountID(idToken string) string {
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		AccountID string `json:"chatgpt_account_id"`
		Auth      struct {
			AccountID string `json:"chatgpt_account_id"`
		} `json:"https://api.openai.com/auth"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	if claims.AccountID != "" {
		return claims.AccountID
	}
	return claims.Auth.AccountID
}

// chatGPTFedRAMP reports the id token's chatgpt_account_is_fedramp claim, as
// Codex reads it (login/src/token_data.rs AuthClaims). A missing or
// unreadable claim is false.
func chatGPTFedRAMP(idToken string) bool {
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	var claims struct {
		Auth struct {
			FedRAMP bool `json:"chatgpt_account_is_fedramp"`
		} `json:"https://api.openai.com/auth"`
	}
	return json.Unmarshal(payload, &claims) == nil && claims.Auth.FedRAMP
}

func decodeOAuthResponse(resp *http.Response, target any) error {
	decodeErr := json.NewDecoder(io.LimitReader(resp.Body, oauthResponseLimit)).Decode(target)
	closeErr := resp.Body.Close()
	if decodeErr != nil {
		if closeErr != nil {
			return errors.Join(decodeErr, closeErr)
		}
		return decodeErr
	}
	return closeErr
}

func sleepWithContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func ignoreOAuthError(_ error) {}
