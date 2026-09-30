package llmprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
)

// Kilo device login (docs/decisions/0017-MADR-together-provider-and-auth-extensions.md
// D2), as Kilo's own client runs it (kilocode packages/kilo-gateway/src/auth/device.ts):
// POST {origin}/api/device-auth/codes, then GET .../codes/{code} every 3 s until
// 200 (approved, with the token), 403 (denied) or 410 (expired). The token is
// opaque and long-lived: there is no refresh.
const (
	// kiloAPIOrigin is Kilo's API origin, KILO_API_BASE's default.
	kiloAPIOrigin         = "https://api.kilo.ai"
	kiloDevicePoll        = 3 * time.Second
	kiloDeviceDefaultLife = 10 * time.Minute
	kiloResponseLimit     = 1 << 16
)

// startKiloDevice requests a Kilo device code. opts.Issuer overrides the
// origin; nothing else in opts but HTTPClient and the test clock applies.
func startKiloDevice(ctx context.Context, opts OAuthFlowOptions) (*DeviceLogin, error) {
	config := oauthFlowConfig{provider: ProviderKilo, issuer: strings.TrimRight(opts.Issuer, "/"),
		httpClient: opts.HTTPClient, now: opts.now, sleep: opts.sleep}
	if config.issuer == "" {
		config.issuer = kiloAPIOrigin
	}
	if config.httpClient == nil {
		config.httpClient = defaultHTTPClient()
	}
	if config.now == nil {
		config.now = time.Now
	}
	if config.sleep == nil {
		config.sleep = sleepWithContext
	}

	resp, err := kiloDeviceRequest(ctx, config, http.MethodPost, config.issuer+"/api/device-auth/codes")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		closeResponseBody(resp)
		return nil, fmt.Errorf("%w: kilo: too many pending device authorizations; try again later", ErrRateLimited)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, closeOAuthStatusError(resp, "Kilo device-code request")
	}
	var device struct {
		Code            string       `json:"code"`
		VerificationURL string       `json:"verificationUrl"`
		ExpiresIn       oauthSeconds `json:"expiresIn"`
	}
	if err := decodeKiloResponse(resp, &device); err != nil {
		return nil, fmt.Errorf("oauth: decode Kilo device-code response: %w", err)
	}
	if device.Code == "" || device.VerificationURL == "" {
		return nil, errors.New("oauth: Kilo device-code response is incomplete")
	}
	// The code is shown to the user and put in the poll URL's path.
	if strings.IndexFunc(device.Code, func(r rune) bool { return unicode.IsControl(r) || r == '/' }) >= 0 ||
		!safeVerificationURI(device.VerificationURL) {
		return nil, errors.New("oauth: Kilo device-code response has an invalid code or verification URL")
	}
	deadline := config.now().Add(durationFromSeconds(float64(device.ExpiresIn), kiloDeviceDefaultLife, false))
	pollURL := config.issuer + "/api/device-auth/codes/" + url.PathEscape(device.Code)
	poll := func(ctx context.Context) (*OAuthSession, error) {
		for {
			if err := sleepBeforeDeadline(ctx, config, kiloDevicePoll, deadline); err != nil {
				return nil, err
			}
			resp, err := kiloDeviceRequest(ctx, config, http.MethodGet, pollURL)
			if err != nil {
				return nil, err
			}
			switch resp.StatusCode {
			case http.StatusAccepted:
				closeResponseBody(resp)
				continue
			case http.StatusForbidden:
				closeResponseBody(resp)
				return nil, errors.New("oauth: Kilo device authorization denied")
			case http.StatusGone:
				closeResponseBody(resp)
				return nil, errors.New("oauth: Kilo device code expired")
			case http.StatusOK:
				var approved struct {
					Token string `json:"token"`
				}
				if err := decodeKiloResponse(resp, &approved); err != nil {
					return nil, fmt.Errorf("oauth: decode Kilo device approval: %w", err)
				}
				if len(approved.Token) <= 10 {
					return nil, errors.New("oauth: Kilo device approval carries no usable token")
				}
				// No refresh token and no expiry: the session never refreshes,
				// and the token is applied as Kilo's API key (0017-MADR D2).
				return &OAuthSession{Provider: ProviderKilo, Access: approved.Token, Issuer: config.issuer,
					HTTPClient: config.httpClient}, nil
			default:
				return nil, closeOAuthStatusError(resp, "Kilo device poll")
			}
		}
	}
	return &DeviceLogin{UserCode: device.Code, VerificationURI: device.VerificationURL, Expiry: deadline, poll: poll}, nil
}

func kiloDeviceRequest(ctx context.Context, config oauthFlowConfig, method, target string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, target, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("oauth: create Kilo request: %w", err)
	}
	identityOf(ProviderConfig{}).setUserAgent(req)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := config.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth: Kilo request: %w", err)
	}
	return resp, nil
}

func decodeKiloResponse(resp *http.Response, into any) error {
	defer closeResponseBody(resp)
	return json.NewDecoder(io.LimitReader(resp.Body, kiloResponseLimit)).Decode(into)
}

// KiloOrganization is one organization a Kilo account belongs to.
type KiloOrganization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

// KiloAccount is what GET /api/profile says about a Kilo token's account
// (kilocode packages/kilo-gateway/src/api/profile.ts:23-39).
type KiloAccount struct {
	Email                  string
	Organizations          []KiloOrganization
	SelectedOrganizationID string
	HasPersonalAccount     bool
}

// KiloProfile reads the account behind a Kilo token, so a caller can offer
// its organizations; pass the chosen id with WithKiloOrganization. The
// profile lives at {origin}{prefix}/api/profile, derived from the configured
// base URL or a URL-prefixed token as the gateway's endpoints are.
func KiloProfile(ctx context.Context, token string, opts ...ProviderOption) (KiloAccount, error) {
	cfg := ApplyOptions(opts)
	base, err := url.Parse(resolveKiloEndpoints(cfg.BaseURL, token, "").gateway)
	if err != nil {
		return KiloAccount{}, fmt.Errorf("kilo: profile URL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, kiloRoute(base, "profile"), http.NoBody)
	if err != nil {
		return KiloAccount{}, err
	}
	identityOf(cfg).setUserAgent(req)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return KiloAccount{}, fmt.Errorf("kilo: profile: %w", err)
	}
	if err := ClassifyHTTPError(ProviderKilo, resp); err != nil {
		closeResponseBody(resp)
		return KiloAccount{}, err
	}
	var raw struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
		Organizations          []KiloOrganization `json:"organizations"`
		SelectedOrganizationID string             `json:"selectedOrganizationId"`
		HasPersonalAccount     bool               `json:"hasPersonalAccount"`
	}
	if err := decodeKiloResponse(resp, &raw); err != nil {
		return KiloAccount{}, fmt.Errorf("kilo: decode profile: %w", err)
	}
	return KiloAccount{Email: raw.User.Email, Organizations: raw.Organizations,
		SelectedOrganizationID: raw.SelectedOrganizationID, HasPersonalAccount: raw.HasPersonalAccount}, nil
}
