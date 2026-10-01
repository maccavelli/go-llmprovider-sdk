package llmprovider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// withSDKVersion makes transport.BuildVersions report v as this module's version for one
// (non-parallel) test.
func withSDKVersion(t *testing.T, v string) {
	t.Helper()
	saved := transport.BuildVersions
	transport.BuildVersions = func() (string, string) { return v, v }
	t.Cleanup(func() { transport.BuildVersions = saved })
}

// chatGPTListingVersion returns the client_version one ChatGPT listing sent.
func chatGPTListingVersion(t *testing.T) string {
	t.Helper()
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query().Get("client_version")
		_, _ = w.Write([]byte(`{"models":[{"slug":"gpt-6-astra","visibility":"list","priority":1}]}`))
	}))
	t.Cleanup(srv.Close)
	session := &OAuthSession{Issuer: DefaultOpenAIIssuer, Access: "a", Expiry: time.Now().Add(time.Hour)}
	if _, err := ListAvailableModelsWithSource(context.Background(), ProviderOpenAI, session,
		WithHTTPClient(srv.Client()), WithBaseURL(srv.URL)); err != nil {
		t.Fatalf("listing: %v", err)
	}
	return got
}

// TestChatGPTListing_SendsSDKVersion: a release, a pseudo-version and an
// +incompatible build send X.Y.Z (the backend answers 400 to "v1.5.0",
// "1.5", "(devel)" and anything over 32 characters; measured 2026-09-27).
func TestChatGPTListing_SendsSDKVersion(t *testing.T) {
	for _, tc := range []struct{ build, want string }{
		{"v1.5.0", "1.5.0"},
		{"v1.5.1-0.20260927120000-7ea0ad4abcde", "1.5.1"},
		{"v2.0.0+incompatible", "2.0.0"},
	} {
		t.Run(tc.build, func(t *testing.T) {
			withSDKVersion(t, tc.build)
			if got := chatGPTListingVersion(t); got != tc.want {
				t.Errorf("client_version = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestChatGPTListing_DevelBuildSendsFallback: a build with no release version
// sends 0.0.0, which the backend accepts.
func TestChatGPTListing_DevelBuildSendsFallback(t *testing.T) {
	for _, build := range []string{"(devel)", "", "1.5"} {
		t.Run(build, func(t *testing.T) {
			withSDKVersion(t, build)
			if got := chatGPTListingVersion(t); got != "0.0.0" {
				t.Errorf("client_version = %q, want 0.0.0", got)
			}
		})
	}
}
