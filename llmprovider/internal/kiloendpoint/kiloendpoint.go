// Package kiloendpoint derives Kilo's endpoints from a credential (MADR 0012
// §3.3). The listing (catalog), the kilo provider and the Kilo device login
// (llmprovider) all need it, and it uses only the standard library, so it is
// internal to this module rather than any one of them (0015-MADR, amendment
// "`catalog` before the old API's removal").
package kiloendpoint

import (
	"net/url"
	"regexp"
	"strings"
)

// BaseURL is Kilo Gateway's generation base.
const BaseURL = "https://api.kilo.ai/api/gateway"

// OrganizationHeader scopes a request to a Kilo organization.
const OrganizationHeader = "X-KILOCODE-ORGANIZATIONID"

// tokenURLRE matches Kilo's URL-prefixed token, "{backend URL}:{secret}"
// (kilocode auth/token.ts:9). The whole token is still the bearer.
var tokenURLRE = regexp.MustCompile(`^(https?://[^:]+(?::\d+)?(?:/[^:]*)?):`)

// Endpoints are the URLs and organization one Kilo credential uses.
type Endpoints struct {
	Gateway string // generation base, {origin}{prefix}/api/gateway by default
	Models  string // listing URL
	Org     string // organization id, or ""
}

// Resolve derives the endpoints from the configured base (else BaseURL), the
// token and an explicit organization. A URL-prefixed token replaces the base
// with its origin and path prefix, as Kilo's client does (api/url.ts:9-30),
// and an /api/organizations/{id} path in it names the organization when none
// is given. An organization lists {origin}{prefix}/api/organizations/{id}/models
// (api/models.ts:219).
func Resolve(baseURL, token, org string) Endpoints {
	base := BaseURL
	if baseURL != "" {
		base = strings.TrimRight(baseURL, "/")
	}
	if m := tokenURLRE.FindStringSubmatch(token); m != nil {
		if u, err := url.Parse(m[1]); err == nil && u.Host != "" {
			base = Route(u, "gateway")
			if org == "" {
				org = pathOrganization(u)
			}
		}
	}
	e := Endpoints{Gateway: base, Models: base + "/models", Org: org}
	if u, err := url.Parse(base); err == nil && org != "" {
		e.Models = Route(u, "organizations/"+url.PathEscape(org)) + "/models"
	}
	return e
}

// pathSegments splits a URL path into its non-empty segments and returns them
// with the index of the last "api" segment, or -1.
func pathSegments(u *url.URL) (parts []string, api int) {
	parts = strings.FieldsFunc(u.Path, func(r rune) bool { return r == '/' })
	// A forward scan that keeps the last match: go fix's slices.Backward
	// rewrite of a backward loop drops the index this returns (0019-MADR).
	api = -1
	for i, part := range parts {
		if part == "api" {
			api = i
		}
	}
	return parts, api
}

// Route is Kilo's route() (api/url.ts:9-18): the path before the last "api"
// segment, then /api/{name}, without query, fragment or trailing slash.
func Route(u *url.URL, name string) string {
	parts, api := pathSegments(u)
	if api >= 0 {
		parts = parts[:api]
	}
	return u.Scheme + "://" + u.Host + "/" + strings.Join(append(parts, "api", name), "/")
}

// pathOrganization returns {id} from a path ending .../api/organizations/{id},
// or "".
func pathOrganization(u *url.URL) string {
	parts, api := pathSegments(u)
	if api >= 0 && len(parts) == api+3 && parts[api+1] == "organizations" {
		return parts[api+2]
	}
	return ""
}
