package llmprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	deviceAuthorizationGrantType = "urn:ietf:params:oauth:grant-type:device_code"
	openAIDeviceTimeout          = 15 * time.Minute
	defaultDeviceExpiry          = 300 * time.Second
	defaultDevicePollInterval    = 5 * time.Second
	deviceSlowDownIncrement      = 5 * time.Second
)

type oauthSeconds float64

func (seconds *oauthSeconds) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "null" || text == "" {
		*seconds = 0
		return nil
	}
	if strings.HasPrefix(text, `"`) {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		text = value
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return err
	}
	*seconds = oauthSeconds(value)
	return nil
}

type openAIDeviceCode struct {
	DeviceAuthID string       `json:"device_auth_id"`
	UserCode     string       `json:"user_code"`
	Interval     oauthSeconds `json:"interval"`
}

type openAIDeviceToken struct {
	AuthorizationCode string `json:"authorization_code"`
	CodeVerifier      string `json:"code_verifier"`
}

type grokDeviceCode struct {
	DeviceCode              string       `json:"device_code"`
	UserCode                string       `json:"user_code"`
	VerificationURI         string       `json:"verification_uri"`
	VerificationURIComplete string       `json:"verification_uri_complete"`
	ExpiresIn               oauthSeconds `json:"expires_in"`
	Interval                oauthSeconds `json:"interval"`
}

type oauthDeviceError struct {
	Code string `json:"error"`
}

// DeviceLogin is a device-code login in progress (0016-MADR D6). Show the
// user UserCode and VerificationURI, then call Wait for the session. The
// code is valid until Expiry. Cancel stops the login from another goroutine,
// so one surface can show the code while another waits.
type DeviceLogin struct {
	// UserCode is the code the user enters.
	UserCode string
	// VerificationURI is where the user enters it; for Grok it may already
	// carry the code.
	VerificationURI string
	// Expiry is when the code stops being accepted.
	Expiry time.Time

	poll func(context.Context) (*OAuthSession, error)

	mu       sync.Mutex
	canceled bool
	stop     context.CancelFunc
	done     chan struct{}
	session  *OAuthSession
	err      error
}

// errDeviceLoginCanceled is what Wait returns after Cancel; it matches
// context.Canceled.
var errDeviceLoginCanceled = fmt.Errorf("oauth: device login canceled: %w", context.Canceled)

// StartDeviceOAuth starts the provider's device-code login: it requests the
// code, and returns a handle to show it and wait for approval.
func StartDeviceOAuth(ctx context.Context, provider string, opts OAuthFlowOptions) (*DeviceLogin, error) {
	if provider == ProviderKilo {
		return startKiloDevice(ctx, opts)
	}
	config, err := resolveOAuthFlowConfig(provider, opts)
	if err != nil {
		return nil, err
	}
	if provider == ProviderOpenAI {
		return startOpenAIDevice(ctx, config)
	}
	return startGrokDevice(ctx, config)
}

// Wait polls until the user approves, the code expires, ctx ends or Cancel is
// called. A second Wait, concurrent or later, returns the first one's result.
func (d *DeviceLogin) Wait(ctx context.Context) (*OAuthSession, error) {
	d.mu.Lock()
	if done := d.done; done != nil {
		d.mu.Unlock()
		select {
		case <-done:
			return d.session, d.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	d.done = make(chan struct{})
	if d.canceled {
		d.err = errDeviceLoginCanceled
		close(d.done)
		d.mu.Unlock()
		return nil, d.err
	}
	pollCtx, stop := context.WithCancel(ctx)
	d.stop = stop
	d.mu.Unlock()

	session, err := d.poll(pollCtx)
	stop()

	d.mu.Lock()
	if err != nil && d.canceled {
		err = errDeviceLoginCanceled
	}
	d.session, d.err = session, err
	close(d.done)
	d.mu.Unlock()
	return session, err
}

// Cancel stops the login: a running Wait returns promptly with an error
// matching context.Canceled, and a later Wait returns it at once.
func (d *DeviceLogin) Cancel() {
	d.mu.Lock()
	d.canceled = true
	stop := d.stop
	d.mu.Unlock()
	if stop != nil {
		stop()
	}
}

// LoginDeviceOAuth completes the provider's headless device-code flow in one
// call: it starts the login, reports the code through opts.NotifyDevice, and
// waits for approval. StartDeviceOAuth gives the same flow as a handle.
func LoginDeviceOAuth(ctx context.Context, provider string, opts OAuthFlowOptions) (*OAuthSession, error) {
	login, err := StartDeviceOAuth(ctx, provider, opts)
	if err != nil {
		return nil, err
	}
	if opts.NotifyDevice != nil {
		opts.NotifyDevice(login.VerificationURI, login.UserCode)
	}
	return login.Wait(ctx)
}

func startOpenAIDevice(ctx context.Context, config oauthFlowConfig) (*DeviceLogin, error) {
	endpoints, err := oauthEndpointsFor(ctx, config)
	if err != nil {
		return nil, err
	}
	userCodeURL := config.issuer + "/api/accounts/deviceauth/usercode"
	resp, err := postOAuthJSON(ctx, config.httpClient, userCodeURL, map[string]string{oauthParamClientID: config.clientID})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, closeOAuthStatusError(resp, "OpenAI device-code request")
	}
	var device openAIDeviceCode
	if err := decodeOAuthResponse(resp, &device); err != nil {
		return nil, fmt.Errorf("oauth: decode OpenAI device-code response: %w", err)
	}
	if device.DeviceAuthID == "" || device.UserCode == "" {
		return nil, errors.New("oauth: OpenAI device-code response is incomplete")
	}
	deadline := config.now().Add(openAIDeviceTimeout)
	interval := durationFromSeconds(float64(device.Interval), defaultDevicePollInterval, false)
	poll := func(ctx context.Context) (*OAuthSession, error) {
		pollURL := config.issuer + "/api/accounts/deviceauth/token"
		for {
			if !config.now().Before(deadline) {
				return nil, errors.New("oauth: OpenAI device-code login timed out after 15 minutes")
			}
			pollResponse, pollErr := postOAuthJSON(ctx, config.httpClient, pollURL, map[string]string{
				"device_auth_id": device.DeviceAuthID,
				"user_code":      device.UserCode,
			})
			if pollErr != nil {
				return nil, pollErr
			}
			if pollResponse.StatusCode >= http.StatusOK && pollResponse.StatusCode < http.StatusMultipleChoices {
				var code openAIDeviceToken
				if decodeErr := decodeOAuthResponse(pollResponse, &code); decodeErr != nil {
					return nil, fmt.Errorf("oauth: decode OpenAI device token: %w", decodeErr)
				}
				if code.AuthorizationCode == "" || code.CodeVerifier == "" {
					return nil, errors.New("oauth: OpenAI device token response is incomplete")
				}
				return exchangeOAuthCode(
					ctx,
					config,
					endpoints,
					code.AuthorizationCode,
					config.issuer+"/deviceauth/callback",
					code.CodeVerifier,
					"",
				)
			}
			status := pollResponse.StatusCode
			if closeErr := pollResponse.Body.Close(); closeErr != nil {
				return nil, fmt.Errorf("oauth: close OpenAI device poll response: %w", closeErr)
			}
			if status != http.StatusForbidden && status != http.StatusNotFound {
				return nil, fmt.Errorf("oauth: OpenAI device poll failed: %s", http.StatusText(status))
			}
			if err := sleepBeforeDeadline(ctx, config, interval, deadline); err != nil {
				return nil, err
			}
		}
	}
	return &DeviceLogin{UserCode: device.UserCode, VerificationURI: config.issuer + "/codex/device",
		Expiry: deadline, poll: poll}, nil
}

func startGrokDevice(ctx context.Context, config oauthFlowConfig) (*DeviceLogin, error) {
	endpoints, err := oauthEndpointsFor(ctx, config)
	if err != nil {
		return nil, err
	}
	if err := requireEndpoints(config.issuer, endpoints.Device, endpoints.Token); err != nil {
		return nil, err
	}
	form := url.Values{
		oauthParamClientID: {config.clientID},
		"scope":            {grokOAuthScopes},
		"referrer":         {"go-llmprovider-sdk"},
	}
	resp, err := postOAuthForm(ctx, config.httpClient, endpoints.Device, form)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, closeOAuthStatusError(resp, "Grok device-code request")
	}
	var device grokDeviceCode
	if err := decodeOAuthResponse(resp, &device); err != nil {
		return nil, fmt.Errorf("oauth: decode Grok device-code response: %w", err)
	}
	if device.DeviceCode == "" || device.UserCode == "" || device.VerificationURI == "" {
		return nil, errors.New("oauth: Grok device-code response is incomplete")
	}
	if err := validateGrokDeviceCode(device); err != nil {
		return nil, err
	}
	verificationURL := device.VerificationURI
	if device.VerificationURIComplete != "" {
		verificationURL = device.VerificationURIComplete
	}
	expires := durationFromSeconds(float64(device.ExpiresIn), defaultDeviceExpiry, false)
	interval := durationFromSeconds(float64(device.Interval), defaultDevicePollInterval, true)
	deadline := config.now().Add(expires)
	poll := func(ctx context.Context) (*OAuthSession, error) {
		for {
			if err := sleepBeforeDeadline(ctx, config, interval, deadline); err != nil {
				return nil, err
			}
			tokenForm := url.Values{
				oauthParamGrantType: {deviceAuthorizationGrantType},
				"device_code":       {device.DeviceCode},
				oauthParamClientID:  {config.clientID},
			}
			tokenResponse, tokenErr := postOAuthForm(ctx, config.httpClient, endpoints.Token, tokenForm)
			if tokenErr != nil {
				return nil, tokenErr
			}
			if tokenResponse.StatusCode >= http.StatusOK && tokenResponse.StatusCode < http.StatusMultipleChoices {
				var payload oauthTokenResponse
				if decodeErr := decodeOAuthResponse(tokenResponse, &payload); decodeErr != nil {
					return nil, fmt.Errorf("oauth: decode Grok device token: %w", decodeErr)
				}
				return verifiedSession(ctx, config, endpoints, payload, "")
			}
			var deviceErr oauthDeviceError
			if decodeErr := decodeOAuthResponse(tokenResponse, &deviceErr); decodeErr != nil {
				return nil, fmt.Errorf("oauth: decode Grok device error: %w", decodeErr)
			}
			switch deviceErr.Code {
			case "authorization_pending":
				continue
			case "slow_down":
				interval += deviceSlowDownIncrement
				continue
			case "access_denied":
				return nil, errors.New("oauth: Grok device authorization denied")
			case "expired_token":
				return nil, errors.New("oauth: Grok device code expired")
			default:
				return nil, fmt.Errorf("oauth: Grok device token failed: %s", deviceErr.Code)
			}
		}
	}
	return &DeviceLogin{UserCode: device.UserCode, VerificationURI: verificationURL, Expiry: deadline, poll: poll}, nil
}

// validateGrokDeviceCode refuses a device-code response a malicious issuer
// could use against the user, as the Grok CLI does
// (xai-grok-login/src/device_code.rs:148-155, :474-487): the user_code must
// be letters, digits and '-', and each verification URI https, or http to a
// loopback host, with no control characters.
func validateGrokDeviceCode(device grokDeviceCode) error {
	for _, r := range device.UserCode {
		if r > unicode.MaxASCII || !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' {
			return errors.New("oauth: Grok device-code response has an invalid user_code")
		}
	}
	for _, uri := range []string{device.VerificationURI, device.VerificationURIComplete} {
		if uri != "" && !safeVerificationURI(uri) {
			return errors.New("oauth: Grok device-code response has an invalid verification URI")
		}
	}
	return nil
}

func safeVerificationURI(uri string) bool {
	if strings.IndexFunc(uri, unicode.IsControl) >= 0 {
		return false
	}
	parsed, err := url.Parse(uri)
	if err != nil {
		return false
	}
	switch parsed.Scheme {
	case "https":
		return parsed.Host != ""
	case "http":
		host := parsed.Hostname()
		return host == "localhost" || host == "127.0.0.1"
	default:
		return false
	}
}

func durationFromSeconds(seconds float64, fallback time.Duration, floor bool) time.Duration {
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds <= 0 {
		return fallback
	}
	duration := time.Duration(seconds * float64(time.Second))
	if floor && duration < time.Second {
		return time.Second
	}
	return duration
}

func sleepBeforeDeadline(ctx context.Context, config oauthFlowConfig, interval time.Duration, deadline time.Time) error {
	remaining := deadline.Sub(config.now())
	if remaining <= 0 {
		return errors.New("oauth: device code expired")
	}
	if interval > remaining {
		interval = remaining
	}
	if err := config.sleep(ctx, interval); err != nil {
		return fmt.Errorf("oauth: device-code wait: %w", err)
	}
	if !config.now().Before(deadline) {
		return errors.New("oauth: device code expired")
	}
	return nil
}

func postOAuthJSON(ctx context.Context, client *http.Client, endpoint string, payload any) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("oauth: encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("oauth: create request: %w", err)
	}
	identityOf(providerConfig{}).SetUserAgent(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth: request: %w", err)
	}
	return resp, nil
}

func postOAuthForm(ctx context.Context, client *http.Client, endpoint string, form url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("oauth: create request: %w", err)
	}
	identityOf(providerConfig{}).SetUserAgent(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth: request: %w", err)
	}
	return resp, nil
}

func closeOAuthStatusError(resp *http.Response, action string) error {
	statusErr := fmt.Errorf("oauth: %s failed: %s", action, resp.Status)
	if closeErr := resp.Body.Close(); closeErr != nil {
		return errors.Join(statusErr, fmt.Errorf("oauth: close response: %w", closeErr))
	}
	return statusErr
}
