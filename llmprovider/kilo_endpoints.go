package llmprovider

import (
	"net/url"
	"regexp"
	"strings"
)

// Kilo's endpoints, which the listing (discovery.go) and the kilo provider
// (providers/kilo, 0015-PLAN S7) both derive from a credential.

// wireShapesProbedOnKilo is the date Kilo's wire shapes were measured against
// the live gateway: the message.reasoning spelling (not reasoning_content),
// supported_parameters as a per-model capability list, pricing.completion as
// a string with "-1" for variable-priced tiers, and /chat/completions
// answering free models with no credential. The live probes in
// live_gateways_test.go check it; the provider is providers/kilo.
// Re-validate with: go test -tags live_gateways ./llmprovider/ -run Live
const wireShapesProbedOnKilo = "2026-08-29"

// kiloBaseURL is Kilo Gateway's generation base.
const kiloBaseURL = "https://api.kilo.ai/api/gateway"

// kiloTokenURLRE matches Kilo's URL-prefixed token, "{backend URL}:{secret}"
// (kilocode auth/token.ts:9). The whole token is still the bearer.
var kiloTokenURLRE = regexp.MustCompile(`^(https?://[^:]+(?::\d+)?(?:/[^:]*)?):`)

// kiloOrganizationHeader scopes a request to a Kilo organization.
const kiloOrganizationHeader = "X-KILOCODE-ORGANIZATIONID"

// kiloEndpoints are the URLs and organization one Kilo credential uses
// (MADR 0012 §3.3).
type kiloEndpoints struct {
	gateway string // generation base, {origin}{prefix}/api/gateway by default
	models  string // listing URL
	org     string // organization id, or ""
}

// resolveKiloEndpoints derives the endpoints from the configured base (else
// kiloBaseURL), the token and an explicit organization. A URL-prefixed token
// replaces the base with its origin and path prefix, as Kilo's client does
// (api/url.ts:9-30), and an /api/organizations/{id} path in it names the
// organization when none is given. An organization lists
// {origin}{prefix}/api/organizations/{id}/models (api/models.ts:219).
func resolveKiloEndpoints(baseURL, token, org string) kiloEndpoints {
	base := kiloBaseURL
	if baseURL != "" {
		base = strings.TrimRight(baseURL, "/")
	}
	if m := kiloTokenURLRE.FindStringSubmatch(token); m != nil {
		if u, err := url.Parse(m[1]); err == nil && u.Host != "" {
			base = kiloRoute(u, "gateway")
			if org == "" {
				org = kiloPathOrganization(u)
			}
		}
	}
	e := kiloEndpoints{gateway: base, models: base + "/models", org: org}
	if u, err := url.Parse(base); err == nil && org != "" {
		e.models = kiloRoute(u, "organizations/"+url.PathEscape(org)) + "/models"
	}
	return e
}

// kiloPathSegments splits a URL path into its non-empty segments and returns
// them with the index of the last "api" segment, or -1.
func kiloPathSegments(u *url.URL) (parts []string, api int) {
	parts = strings.FieldsFunc(u.Path, func(r rune) bool { return r == '/' })
	for api = len(parts) - 1; api >= 0; api-- {
		if parts[api] == "api" {
			break
		}
	}
	return parts, api
}

// kiloRoute is Kilo's route() (api/url.ts:9-18): the path before the last
// "api" segment, then /api/{name}, without query, fragment or trailing slash.
func kiloRoute(u *url.URL, name string) string {
	parts, api := kiloPathSegments(u)
	if api >= 0 {
		parts = parts[:api]
	}
	return u.Scheme + "://" + u.Host + "/" + strings.Join(append(parts, "api", name), "/")
}

// kiloPathOrganization returns {id}
// kiloPathOrganization returns {id} from a path ending .../api/organizations/{id},
// or "".
func kiloPathOrganization(u *url.URL) string {
	parts, api := kiloPathSegments(u)
	if api >= 0 && len(parts) == api+3 && parts[api+1] == "organizations" {
		return parts[api+2]
	}
	return ""
}

// KiloGatewayFor returns the generation base and the organization one Kilo
// credential uses: baseURL, else Kilo Gateway; a URL-prefixed token ("https://
// host/prefix:secret") replaces it and may name the organization when org is
// empty (MADR 0012 §3.3).
//
// Temporary export for the provider packages (0015-PLAN S7); S8b moves it to catalog.
func KiloGatewayFor(baseURL, token, org string) (gateway, organization string) {
	e := resolveKiloEndpoints(baseURL, token, org)
	return e.gateway, e.org
}
