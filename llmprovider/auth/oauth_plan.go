package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// OpenAI's Sign in with ChatGPT for agents (0043-MADR): a plan session's
// constants and its issuer's endpoints.
const (
	// planResource is the resource a plan session's tokens are granted for:
	// the platform's API, where they are used.
	planResource = "https://api.openai.com/v1"
	// planParamResource names it in the authorize, token and refresh requests.
	planParamResource = "resource"
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
