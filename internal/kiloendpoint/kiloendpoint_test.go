package kiloendpoint

import (
	"net/url"
	"strings"
	"testing"
)

// TestResolve was llmprovider's TestKiloGatewayFor (0015-PLAN S8, commit 2),
// with the listing URL and the organization's: the base, a URL-prefixed
// token's origin and organization, and an explicit organization. The URL rules
// are pinned end to end in providers/kilo's endpoints_test.go.
func TestResolve(t *testing.T) {
	for _, tc := range []struct{ base, token, org, wantGateway, wantModels, wantOrg string }{
		{"", "sk-plain", "", BaseURL, BaseURL + "/models", ""},
		{"http://127.0.0.1:9/", "sk-plain", "org-1", "http://127.0.0.1:9", "http://127.0.0.1:9/api/organizations/org-1/models", "org-1"},
		{"", "https://kilo.example.test/api/organizations/org-9:secret", "", "https://kilo.example.test/api/gateway",
			"https://kilo.example.test/api/organizations/org-9/models", "org-9"},
		{"", "https://kilo.example.test/prefix/api:secret", "", "https://kilo.example.test/prefix/api/gateway",
			"https://kilo.example.test/prefix/api/gateway/models", ""},
		{"", "https://kilo.example.test/api/organizations/org-9:secret", "org-1", "https://kilo.example.test/api/gateway",
			"https://kilo.example.test/api/organizations/org-1/models", "org-1"},
	} {
		e := Resolve(tc.base, tc.token, tc.org)
		if e.Gateway != tc.wantGateway || e.Models != tc.wantModels || e.Org != tc.wantOrg {
			t.Errorf("Resolve(%q, %q, %q) = %+v; want %q, %q, %q", tc.base, tc.token, tc.org, e,
				tc.wantGateway, tc.wantModels, tc.wantOrg)
		}
	}
}

// TestRoute: the path before the last "api" segment, then /api/{name}; with no
// "api" segment, the whole path.
func TestRoute(t *testing.T) {
	for raw, want := range map[string]string{
		"https://kilo.example.test/x/api/gateway?q=1": "https://kilo.example.test/x/api/profile",
		"https://kilo.example.test/x/y":               "https://kilo.example.test/x/y/api/profile",
	} {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		if got := Route(u, "profile"); got != want {
			t.Errorf("Route(%q) = %q, want %q", raw, got, want)
		}
	}
}

// TestOrganization_0080 (0080-MADR D4): the organization a listing is for is the
// explicit one, a token's embedded one, or the base URL's own path's, and a
// listing for one is exactly a listing whose URL is an organization's.
func TestOrganization_0080(t *testing.T) {
	for _, tc := range []struct{ base, token, org, want string }{
		{"", "sk-plain", "", ""},
		{"http://127.0.0.1:9/", "sk-plain", "org-1", "org-1"},
		{"", "https://kilo.example.test/api/organizations/org-9:secret", "", "org-9"},
		{"", "https://kilo.example.test/api/organizations/org-9:secret", "org-1", "org-1"},
		{"", "https://kilo.example.test/prefix/api:secret", "", ""},
		{"https://kilo.example.test/api/organizations/org-5", "sk-plain", "", "org-5"},
		{"https://kilo.example.test/api/organizations/org-5/", "sk-plain", "", "org-5"},
		{"https://kilo.example.test/api/gateway", "sk-plain", "", ""},
	} {
		got := Organization(tc.base, tc.token, tc.org)
		if got != tc.want {
			t.Errorf("Organization(%q, %q, %q) = %q; want %q", tc.base, tc.token, tc.org, got, tc.want)
		}
		if models := Resolve(tc.base, tc.token, tc.org).Models; (got != "") != strings.Contains(models, "/organizations/") {
			t.Errorf("Organization(%q, %q, %q) = %q, but the listing is at %q", tc.base, tc.token, tc.org, got, models)
		}
	}
}

// TestOptionOrganization_0080 (0080-MADR D4): the last OrganizationOption among
// the values replaces any earlier one, an empty one included; other values are
// not organizations.
func TestOptionOrganization_0080(t *testing.T) {
	other := struct{ x int }{1}
	for _, tc := range []struct {
		name   string
		values []any
		want   string
	}{
		{"no values", nil, ""},
		{"an option", []any{OrganizationOption("org-b")}, "org-b"},
		{"the last option wins", []any{OrganizationOption("org-a"), other, OrganizationOption("org-b")}, "org-b"},
		{"an empty option replaces", []any{OrganizationOption("org-a"), OrganizationOption("")}, ""},
		{"a later option replaces an empty one", []any{OrganizationOption(""), OrganizationOption("org-a")}, "org-a"},
		{"another value is no organization", []any{other, "org-z"}, ""},
	} {
		if got := OptionOrganization(tc.values); got != tc.want {
			t.Errorf("%s: OptionOrganization(%v) = %q; want %q", tc.name, tc.values, got, tc.want)
		}
	}
}
