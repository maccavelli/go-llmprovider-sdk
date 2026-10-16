package chatgpt

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// TestIsSession: an OpenAI OAuth session with ChatGPT's issuer, or none, and
// the Codex CLI's login are ChatGPT sessions; a platform session, another
// vendor's login and a key are not.
func TestIsSession(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  llmprovider.TokenSource
		want bool
	}{
		{"ChatGPT's issuer", &auth.OAuthSession{Issuer: auth.DefaultOpenAIIssuer}, true},
		{"an older OpenAI sign-in", &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI}, true},
		{"another issuer", &auth.OAuthSession{Issuer: "https://auth.x.ai", Provider: llmprovider.ProviderOpenAI}, false},
		{"the Codex CLI", &auth.VendorCLISession{Provider: llmprovider.ProviderOpenAI}, true},
		{"the Grok CLI", &auth.VendorCLISession{Provider: llmprovider.ProviderGrok}, false},
		{"a key", llmprovider.NewStaticToken("k"), false},
		{"nil", nil, false},
	} {
		if got := IsSession(tc.src); got != tc.want {
			t.Errorf("%s: IsSession = %v; want %v", tc.name, got, tc.want)
		}
	}
}

// TestSetHeaders: the originator always; the account id and the FedRAMP flag
// only when the session knows them.
func TestSetHeaders(t *testing.T) {
	for _, tc := range []struct {
		name             string
		src              llmprovider.TokenSource
		account, fedRAMP string
	}{
		{"an account on FedRAMP", &auth.OAuthSession{AccountID: "acct-1", FedRAMP: true}, "acct-1", "true"},
		{"an account", &auth.OAuthSession{AccountID: "acct-1"}, "acct-1", ""},
		{"no account", &auth.OAuthSession{}, "", ""},
		{"a key", llmprovider.NewStaticToken("k"), "", ""},
	} {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		SetHeaders(req, tc.src)
		if got := req.Header.Get(HeaderOriginator); got != OriginatorValue {
			t.Errorf("%s: originator %q; want %q", tc.name, got, OriginatorValue)
		}
		if got := req.Header.Get(HeaderAccount); got != tc.account {
			t.Errorf("%s: account %q; want %q", tc.name, got, tc.account)
		}
		if got := req.Header.Get(HeaderFedRAMP); got != tc.fedRAMP {
			t.Errorf("%s: FedRAMP %q; want %q", tc.name, got, tc.fedRAMP)
		}
	}
}

// TestClientVersion: X.Y.Z from a release or a pseudo-version, and 0.0.0 for
// anything the backend refuses.
func TestClientVersion(t *testing.T) {
	for v, want := range map[string]string{
		"v1.5.0":                               "1.5.0",
		"1.5.0":                                "1.5.0",
		"v1.6.0-0.20261001120000-0123456789ab": "1.6.0",
		"v1.2.3+dirty":                         "1.2.3",
		"1.5":                                  ModelsClientVersion,
		"(devel)":                              ModelsClientVersion,
		"":                                     ModelsClientVersion,
		"v1234567890.0.0":                      ModelsClientVersion,
	} {
		if got := ClientVersion(v); got != want {
			t.Errorf("ClientVersion(%q) = %q; want %q", v, got, want)
		}
	}
}

// TestModelsRequest: GET base/models with this build's client_version, the
// user agent, the bearer token and the session's headers; a base URL that does
// not parse is an error.
func TestModelsRequest(t *testing.T) {
	src := &auth.OAuthSession{Issuer: auth.DefaultOpenAIIssuer, AccountID: "acct-1"}
	req, err := ModelsRequest(context.Background(), "http://example.test/codex/", "agent/1", llmprovider.Token{Value: "t"}, src)
	if err != nil {
		t.Fatal(err)
	}
	sdk, _ := transport.BuildVersions()
	if want := "http://example.test/codex/models?client_version=" + ClientVersion(sdk); req.Method != http.MethodGet ||
		req.URL.String() != want {
		t.Errorf("%s %s; want GET %s", req.Method, req.URL, want)
	}
	for header, want := range map[string]string{
		"User-Agent": "agent/1", "Authorization": "Bearer t", HeaderOriginator: OriginatorValue, HeaderAccount: "acct-1",
	} {
		if got := req.Header.Get(header); got != want {
			t.Errorf("%s: %q; want %q", header, got, want)
		}
	}
	if _, err := ModelsRequest(context.Background(), "http://[::1", "", llmprovider.Token{}, nil); err == nil ||
		!strings.Contains(err.Error(), "parse chatgpt models URL") {
		t.Errorf("a base URL that does not parse: %v; want a parse error", err)
	}
}
