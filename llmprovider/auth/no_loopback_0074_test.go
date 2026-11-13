package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// NoLoopback on the two browser sign-ins (0074-MADR D4): no listener, the
// redirect URI names the first registered port, and only a paste completes it.

// holdPorts0074 gives the browser sign-ins two registered ports and binds
// both, on IPv4 and, where there is one, IPv6, so that a sign-in that tried
// to listen could not.
func holdPorts0074(t *testing.T) []int {
	t.Helper()
	ports := []int{nettest.Port(t), nettest.Port(t)}
	original := openaiLoopbackPorts
	openaiLoopbackPorts = ports
	t.Cleanup(func() { openaiLoopbackPorts = original })
	for _, port := range ports {
		for _, host := range []string{"127.0.0.1", "::1"} {
			ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
			if err != nil {
				if host == "::1" {
					continue // no IPv6 loopback here
				}
				t.Fatalf("hold port %d: %v", port, err)
			}
			t.Cleanup(func() { closeListener(t, ln) })
		}
	}
	return ports
}

// pasteBack0074 is an OpenURL and an InputCode: the first records the
// authorize URL, the second waits for it and answers with the callback URL
// the identity provider would have redirected to, which the person pastes.
func pasteBack0074(callback func(authorize string) (string, error)) (func(string) error, func(context.Context) (string, error)) {
	authorize := make(chan string, 1)
	openURL := func(raw string) error {
		authorize <- raw
		return nil
	}
	inputCode := func(ctx context.Context) (string, error) {
		select {
		case raw := <-authorize:
			return callback(raw)
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return openURL, inputCode
}

// TestNoLoopback_OpenAI_0074: with NoLoopback the Codex sign-in binds no
// listener (both registered ports are held and it still starts), names the
// first port in its redirect URI, and a pasted callback URL completes it;
// without an InputCode it is refused with ErrInvalidRequest, before any
// request.
func TestNoLoopback_OpenAI_0074(t *testing.T) {
	ports := holdPorts0074(t)
	var mu sync.Mutex
	var tokenForm url.Values
	var requests atomic.Int32
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	ti := newTestIssuer(mux, srv.URL)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, ti.discovery(t, nil))
	})
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse token form: %v", err)
		}
		mu.Lock()
		tokenForm = r.PostForm
		mu.Unlock()
		writeTestJSON(t, w, ti.tokenResponse(t, "RS256", "test-client", "access", "refresh"))
	})
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		return srv.Client().Transport.RoundTrip(r)
	})}
	opts := OAuthFlowOptions{HTTPClient: client, ClientID: "test-client", Issuer: srv.URL, NoLoopback: true}

	t.Run("a paste completes it", func(t *testing.T) {
		var authorizeURL url.Values
		withPaste := opts
		withPaste.OpenURL, withPaste.InputCode = pasteBack0074(func(raw string) (string, error) {
			u, err := url.Parse(raw)
			if err != nil {
				return "", err
			}
			authorizeURL = u.Query()
			return "http://127.0.0.1/auth/callback?code=pasted-code&state=" + url.QueryEscape(authorizeURL.Get("state")), nil
		})
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		session, err := LoginBrowserOAuth(ctx, llmprovider.ProviderOpenAI, withPaste)
		if err != nil || session == nil || session.Access != "access" {
			t.Fatalf("LoginBrowserOAuth = %v, %v; want a session", session, err)
		}
		want := fmt.Sprintf("http://127.0.0.1:%d/auth/callback", ports[0])
		if got := authorizeURL.Get("redirect_uri"); got != want {
			t.Errorf("authorize redirect_uri = %q, want %q", got, want)
		}
		mu.Lock()
		defer mu.Unlock()
		if got := tokenForm.Get("redirect_uri"); got != want || tokenForm.Get("code") != "pasted-code" {
			t.Errorf("token form redirect_uri %q code %q, want %q and pasted-code", got, tokenForm.Get("code"), want)
		}
	})
	t.Run("without an InputCode", func(t *testing.T) {
		before := requests.Load()
		withoutInput := opts
		withoutInput.OpenURL = func(string) error { return nil }
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		session, err := LoginBrowserOAuth(ctx, llmprovider.ProviderOpenAI, withoutInput)
		if session != nil || !errors.Is(err, llmprovider.ErrInvalidRequest) {
			t.Fatalf("LoginBrowserOAuth = %v, %v; want ErrInvalidRequest", session, err)
		}
		if got := requests.Load(); got != before {
			t.Errorf("%d request(s) before the refusal; want none", got-before)
		}
	})
}

// TestNoLoopback_Plan_0074: the same for LoginChatGPTPlan.
func TestNoLoopback_Plan_0074(t *testing.T) {
	pl := newPlanLogin(t)
	ports := holdPorts0074(t)
	opts := pl.options()
	opts.NoLoopback = true

	t.Run("a paste completes it", func(t *testing.T) {
		withPaste := opts
		withPaste.OpenURL, withPaste.InputCode = pasteBack0074(pl.callbackURL)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		session, err := LoginChatGPTPlan(ctx, withPaste)
		if err != nil || session == nil || session.ClientID != "oaiapp_issued0043" {
			t.Fatalf("LoginChatGPTPlan = %v, %v; want a plan session", session, err)
		}
		want := fmt.Sprintf("http://127.0.0.1:%d/auth/callback", ports[0])
		pl.mu.Lock()
		defer pl.mu.Unlock()
		if got := pl.authorize.Get("redirect_uri"); got != want {
			t.Errorf("authorize redirect_uri = %q, want %q", got, want)
		}
		if got := pl.tokenForm.Get("redirect_uri"); got != want {
			t.Errorf("token redirect_uri = %q, want %q", got, want)
		}
	})
	t.Run("without an InputCode", func(t *testing.T) {
		tokens := pl.tokens.Load()
		opened := pl.opened.Load()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		session, err := LoginChatGPTPlan(ctx, opts)
		if session != nil || !errors.Is(err, llmprovider.ErrInvalidRequest) {
			t.Fatalf("LoginChatGPTPlan = %v, %v; want ErrInvalidRequest", session, err)
		}
		if pl.tokens.Load() != tokens || pl.opened.Load() != opened {
			t.Errorf("the refused sign-in opened a URL or exchanged a code")
		}
	})
}

// TestNoLoopback_GrokUnsupported_0074: Grok's browser sign-in with NoLoopback
// is ErrUnsupported before any request, with or without an InputCode.
func TestNoLoopback_GrokUnsupported_0074(t *testing.T) {
	var requests atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests.Add(1)
		return nil, errors.New("no request expected")
	})}
	for _, withInput := range []bool{true, false} {
		opts := OAuthFlowOptions{HTTPClient: client, NoLoopback: true}
		if withInput {
			opts.InputCode = func(context.Context) (string, error) { return "code", nil }
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		session, err := LoginBrowserOAuth(ctx, llmprovider.ProviderGrok, opts)
		cancel()
		if session != nil || !errors.Is(err, llmprovider.ErrUnsupported) {
			t.Errorf("InputCode %v: LoginBrowserOAuth = %v, %v; want ErrUnsupported", withInput, session, err)
		}
	}
	if got := requests.Load(); got != 0 {
		t.Errorf("%d request(s) sent; want none", got)
	}
}

// TestNoLoopback_DeviceIgnored_0074: a device sign-in does not read the
// option (0074-MADR D4): it runs as it does without it.
func TestNoLoopback_DeviceIgnored_0074(t *testing.T) {
	f := newFake0048(t, llmprovider.ProviderGrok, reply0048(t, "/token", http.StatusBadRequest, `{"error":"access_denied"}`))
	opts := f.options(newFakeOAuthClock())
	opts.NoLoopback = true
	login, err := StartDeviceOAuth(context.Background(), llmprovider.ProviderGrok, opts)
	if err != nil {
		t.Fatalf("StartDeviceOAuth with NoLoopback = %v; want it to start as without", err)
	}
	if _, err = login.Wait(context.Background()); !errors.Is(err, ErrSignInDenied) {
		t.Errorf("Wait = %v; want the same denial as without the option", err)
	}
}
