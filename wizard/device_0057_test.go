package wizard

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// grokDeviceIdx is the device sign-in's place in Grok's menu with a store:
// the API key, the browser, then the device code.
const grokDeviceIdx = 2

// stubGrokDevice0057 replaces the device sign-in with one that shows code
// and, when show accepts it, returns a Grok session.
func stubGrokDevice0057(t *testing.T, code DeviceCode) {
	t.Helper()
	original := deviceLogin
	t.Cleanup(func() { deviceLogin = original })
	deviceLogin = func(_ context.Context, provider llmprovider.ProviderID, _ auth.OAuthFlowOptions, show func(DeviceCode) error) (*auth.OAuthSession, error) {
		if err := show(code); err != nil {
			return nil, err
		}
		return testOAuthSession(provider), nil
	}
}

// grokCode0057 is the code the Grok stub shows.
var grokCode0057 = DeviceCode{Provider: llmprovider.ProviderGrok, URL: "https://verify.test", Code: "ABCD-EFGH",
	Expiry: time.Date(2026, 10, 22, 18, 0, 0, 0, time.UTC)}

// TestShowDeviceCode_Data_0057 (0057-MADR D1, issue 6): the hook is given the
// device sign-in's code, once, and no notice shows it.
func TestShowDeviceCode_Data_0057(t *testing.T) {
	stubGrokDevice0057(t, grokCode0057)
	var got []DeviceCode
	f := newFake(t, fakePrompter{selects: []int{grokDeviceIdx}})
	res, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), NoEndpointPrompt: true,
		ShowDeviceCode: func(c DeviceCode) error { got = append(got, c); return nil }}, llmprovider.ProviderGrok)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != grokCode0057 || res.Kind != CredOAuth {
		t.Fatalf("hook given %v, kind %q; want %v once, and the session", got, res.Kind, grokCode0057)
	}
	if n := countContaining(f.seenNotify, grokCode0057.Code); n != 0 {
		t.Errorf("notices %q show the code; the hook replaces the notice", f.seenNotify)
	}
}

// TestShowDeviceCode_Kilo_0057: Kilo's device sign-in, and Kilo's as a key,
// each give the hook the code, and no notice shows it.
func TestShowDeviceCode_Kilo_0057(t *testing.T) {
	for _, keyMode := range []bool{false, true} {
		stubKilo(t, auth.KiloAccount{HasPersonalAccount: true}, nil)
		var got []DeviceCode
		f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, selects: []int{1}})
		_, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), KiloDeviceKey: keyMode,
			ShowDeviceCode: func(c DeviceCode) error { got = append(got, c); return nil }}, llmprovider.ProviderKilo)
		if err != nil {
			t.Fatalf("key mode %v: %v", keyMode, err)
		}
		if len(got) != 1 || got[0].Provider != llmprovider.ProviderKilo || got[0].Code != "KILO-1234" {
			t.Errorf("key mode %v: hook given %v; want Kilo's code once", keyMode, got)
		}
		if n := countContaining(f.seenNotify, "KILO-1234"); n != 0 {
			t.Errorf("key mode %v: notices %q show the code", keyMode, f.seenNotify)
		}
	}
}

// TestShowDeviceCode_ErrorEndsSignIn_0057: an error from the hook ends the
// sign-in, returned as it is, and nothing is saved.
func TestShowDeviceCode_ErrorEndsSignIn_0057(t *testing.T) {
	stubGrokDevice0057(t, grokCode0057)
	errShow := errors.New("the event could not be written")
	store := newMemoryTokenStore()
	f := newFake(t, fakePrompter{selects: []int{grokDeviceIdx}})
	_, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: store, NoEndpointPrompt: true,
		ShowDeviceCode: func(DeviceCode) error { return errShow }}, llmprovider.ProviderGrok)
	if !errors.Is(err, errShow) || store.saves != 0 {
		t.Fatalf("error %v, %d saves; want the hook's error and nothing saved", err, store.saves)
	}
}

// TestShowDeviceCode_NilKeepsNotice_0057: with no hook, the notice is as
// before 0057, word for word.
func TestShowDeviceCode_NilKeepsNotice_0057(t *testing.T) {
	stubGrokDevice0057(t, grokCode0057)
	f := newFake(t, fakePrompter{selects: []int{grokDeviceIdx}})
	if _, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), NoEndpointPrompt: true},
		llmprovider.ProviderGrok); err != nil {
		t.Fatal(err)
	}
	const want = "Open https://verify.test and enter code ABCD-EFGH"
	found := false
	for _, n := range f.seenNotify {
		found = found || n == want
	}
	if !found {
		t.Errorf("notices %q; want %q", f.seenNotify, want)
	}
}

// grokDeviceTransport answers Grok's device-code request with expires_in 600,
// refuses discovery, so that the built-in endpoints are used, and counts
// every other request, which a sign-in ended before its wait never makes.
type grokDeviceTransport struct{ others atomic.Int32 }

func (g *grokDeviceTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	reply := func(status int, body string) *http.Response {
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(strings.NewReader(body)), Request: r}
	}
	switch {
	case strings.HasSuffix(r.URL.Path, "/.well-known/openid-configuration"):
		return reply(http.StatusNotFound, `{}`), nil
	case r.Method == http.MethodPost && r.URL.Path == "/oauth2/device/code":
		return reply(http.StatusOK, `{"device_code":"dev-1","user_code":"WXYZ-1234",`+
			`"verification_uri":"https://accounts.x.ai/device","expires_in":600,"interval":1}`), nil
	}
	g.others.Add(1)
	return reply(http.StatusInternalServerError, `{}`), nil
}

// TestRunDeviceLogin_Expiry_0057 (0057-MADR D1 and its amendment): the real
// device sign-in gives show the provider, URL, code and the expiry the device
// authorization returned; show's error is returned as it is, before any poll.
func TestRunDeviceLogin_Expiry_0057(t *testing.T) {
	transport := &grokDeviceTransport{}
	errStop := errors.New("stop after the code")
	var got DeviceCode
	// A wait that polls is bounded, so that it fails here and does not hang.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	_, err := runDeviceLogin(ctx, llmprovider.ProviderGrok,
		auth.OAuthFlowOptions{HTTPClient: &http.Client{Transport: transport}},
		func(c DeviceCode) error { got = c; return errStop })
	if !errors.Is(err, errStop) {
		t.Fatalf("error %v; want show's error as it is", err)
	}
	if got.Provider != llmprovider.ProviderGrok || got.URL != "https://accounts.x.ai/device" || got.Code != "WXYZ-1234" {
		t.Errorf("show given %+v; want Grok's provider, URL and code", got)
	}
	want := start.Add(600 * time.Second)
	if d := got.Expiry.Sub(want); d < -5*time.Second || d > 5*time.Second {
		t.Errorf("expiry %v; want within 5 s of %v, the device authorization's 600 s", got.Expiry, want)
	}
	if n := transport.others.Load(); n != 0 {
		t.Errorf("%d requests beyond discovery and the device code; want none, no poll", n)
	}
}
