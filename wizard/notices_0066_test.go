package wizard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0066-PLAN Phase 5 (V8, V9, V10, V17): no notice shows a service's text or
// a URL's key, an offline machine is called offline, and a failed listing
// beside a static catalog is told.

// refusedDial0066 fails every request as a refused connection does.
var refusedDial0066 = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
	return nil, &net.OpError{Op: "dial", Net: "tcp", Err: os.NewSyscallError("connect", syscall.ECONNREFUSED)}
})}

// assertNoneShows0066 fails when a prompt shows secret.
func assertNoneShows0066(t *testing.T, f *fakePrompter, secret string) {
	t.Helper()
	for _, shown := range f.allText {
		if strings.Contains(shown, secret) {
			t.Errorf("a prompt shows %s: %q", secret, shown)
		}
	}
}

// TestEndpointCheck_NoServerText_0066: the MADR's probe E. An Ollama
// endpoint that answers 500 is told by class and status, not its text, and
// one that cannot be reached by its class.
func TestEndpointCheck_NoServerText_0066(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":"SERVER-TEXT-0066 echoed sk-ant-api03-••••abcd"}`)
	}))
	t.Cleanup(srv.Close)
	for _, tc := range []struct {
		name, endpoint, want string
		client               *http.Client
	}{
		{"answered 500", srv.URL, srv.URL + " answered: provider unavailable, HTTP 500", nil},
		{"not reached", "http://127.0.0.1:1", "cannot reach http://127.0.0.1:1 (offline)", refusedDial0066},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake(t, fakePrompter{inputs: []string{tc.endpoint}, confirms: []bool{false}})
			_, _, err := ResolveCredential(context.Background(), f, Options{HTTPClient: tc.client,
				ProviderOptions: []llmprovider.Option{testCache(t), llmprovider.WithoutModelMetadata()}}, llmprovider.ProviderOllama)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(f.seenNotify, []string{tc.want}) {
				t.Errorf("notices %q; want [%q]", f.seenNotify, tc.want)
			}
			assertNoneShows0066(t, f, "SERVER-TEXT-0066")
			assertNoneShows0066(t, f, "sk-ant-")
		})
	}
}

// TestBaseURL_QueryNotShown_0066: the MADR's probe F, and the other refused
// shapes. No error and no notice shows the planted value; a URL is shown by
// its scheme, host and path.
func TestBaseURL_QueryNotShown_0066(t *testing.T) {
	const secret = "SECRET-0066"
	for _, tc := range []struct{ name, raw, want string }{
		{"a query", "https://proxy.example/v1?key=" + secret,
			"wizard: base URL refused: https://proxy.example/v1?… has a query or a fragment"},
		{"a fragment", "https://proxy.example/v1#" + secret,
			"wizard: base URL refused: https://proxy.example/v1#… has a query or a fragment"},
		{"both", "https://proxy.example/v1?k=" + secret + "#" + secret,
			"wizard: base URL refused: https://proxy.example/v1?…#… has a query or a fragment"},
		{"a user and password", "https://user:" + secret + "@proxy.example/v1",
			"wizard: base URL refused: a base URL carries no user name or password"},
		{"no scheme", "proxy.example/v1?key=" + secret,
			"wizard: base URL refused: an endpoint needs http:// or https:// and a host"},
		{"not a URL", "http://[" + secret,
			"wizard: base URL refused: the endpoint is not a URL"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake(t, fakePrompter{})
			_, err := validateBaseURL(f, tc.raw, true)
			if !errors.Is(err, errBaseURLRefused) || err.Error() != tc.want {
				t.Errorf("err = %v; want %q", err, tc.want)
			}
		})
	}
	t.Run("through ResolveCredential", func(t *testing.T) {
		f := newFake(t, fakePrompter{inputs: []string{"https://proxy.example/v1?key=" + secret, "https://proxy.example/v1"},
			secrets: []string{"tgp_v1_made-up-0066"}})
		res, _, err := ResolveCredential(context.Background(), f, Options{
			ProviderOptions: []llmprovider.Option{testCache(t), llmprovider.WithoutModelMetadata()}}, llmprovider.ProviderTogether)
		if err != nil || res.BaseURL != "https://proxy.example/v1" {
			t.Fatalf("BaseURL %q, %v; want the second endpoint", res.BaseURL, err)
		}
		want := "wizard: base URL refused: https://proxy.example/v1?… has a query or a fragment"
		if !slices.Equal(f.seenNotify, []string{want}) {
			t.Errorf("notices %q; want [%q]", f.seenNotify, want)
		}
		assertNoneShows0066(t, f, secret)
	})
}

// TestDescribe_Offline_0066: the MADR's probe D. An offline error is
// "offline", and so is the listing's notice.
func TestDescribe_Offline_0066(t *testing.T) {
	if got := describe(fmt.Errorf("x: %w", llmprovider.ErrOffline)); got != "offline" {
		t.Errorf("describe(ErrOffline) = %q; want offline", got)
	}
	if got := describe(fmt.Errorf("x: %w", llmprovider.ErrProviderUnavailable)); got != "provider unavailable" {
		t.Errorf("describe(ErrProviderUnavailable) = %q; want provider unavailable", got)
	}
	f := newFake(t, fakePrompter{secrets: []string{"sk-ant-api03-made-up-0066"}})
	_, _, err := ResolveCredential(context.Background(), f, Options{Discover: true, DiscoverLimit: 2 * time.Second,
		HTTPClient: refusedDial0066, ProviderOptions: []llmprovider.Option{testCache(t), llmprovider.WithoutModelMetadata()}},
		llmprovider.ProviderClaude)
	if err != nil {
		t.Fatal(err)
	}
	want := "live model listing for Claude (Anthropic) is unavailable (offline); search covers the built-in catalog only"
	if !slices.Equal(f.seenNotify, []string{want}) {
		t.Errorf("notices %q; want [%q]", f.seenNotify, want)
	}
}

// TestListing_StaticBesideNoticed_0066: V17. A Chat Completions server whose
// descriptor has a static catalog, answering 502: the notice once, Err the
// failure, and the static catalog.
func TestListing_StaticBesideNoticed_0066(t *testing.T) {
	f := newFake(t, fakePrompter{inputs: []string{failingServer0066(t, http.StatusBadGateway).URL + "/v1"}})
	o := compatOptions0066(t)
	o.Registry = staticCompatRegistry0066(t)
	_, cat, err := ResolveCredential(context.Background(), f, o, compatID0066)
	if err != nil {
		t.Fatal(err)
	}
	want := "could not list models for LM Studio (provider unavailable, HTTP 502); using the built-in catalog"
	if !slices.Equal(f.seenNotify, []string{want}) {
		t.Errorf("notices %q; want [%q]", f.seenNotify, want)
	}
	if cat.Live || !errors.Is(cat.Err, llmprovider.ErrProviderUnavailable) || !slices.Equal(cat.Recommended, thirdPartyStatic) {
		t.Errorf("Live %v, Err %v, Recommended %q; want false, the failure, %q", cat.Live, cat.Err, cat.Recommended, thirdPartyStatic)
	}
}
