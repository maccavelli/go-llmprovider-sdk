package kiloendpoint

import (
	"net/url"
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
