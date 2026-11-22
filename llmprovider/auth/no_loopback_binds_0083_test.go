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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// listenRecorder0083 is the listen hook of a browser sign-in: it records every
// bind the sign-in attempts and binds nothing, failing each with fail's error
// when it has one (0083-MADR ST63).
type listenRecorder0083 struct {
	mu    sync.Mutex
	binds []string
	fail  func(network, address string) error
}

func (r *listenRecorder0083) listen(network, address string) (net.Listener, error) {
	r.mu.Lock()
	r.binds = append(r.binds, network+" "+address)
	r.mu.Unlock()
	if r.fail != nil {
		if err := r.fail(network, address); err != nil {
			return nil, err
		}
	}
	return nil, errors.New("listenRecorder0083: no listener is bound in this test")
}

func (r *listenRecorder0083) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.binds)
}

// swapPorts0083 gives the OpenAI sign-ins two registered ports that no other
// package can be given while they are free, and binds neither (0045-MADR D2).
func swapPorts0083(t *testing.T) []int {
	t.Helper()
	ports := []int{nettest.Port(t), nettest.Port(t)}
	original := openaiLoopbackPorts
	openaiLoopbackPorts = ports
	t.Cleanup(func() { openaiLoopbackPorts = original })
	return ports
}

// signIn0083 is one browser sign-in, run against an in-process issuer.
type signIn0083 struct {
	name string
	// login runs it with noLoopback and listen, completing it by a paste when
	// it can start.
	login func(t *testing.T, noLoopback bool, listen listenFunc) (*OAuthSession, error)
	// grok is Grok's sign-in, whose bind is an ephemeral one.
	grok bool
	plan bool
}

func ctx0083(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func state0083(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	return url.QueryEscape(u.Query().Get("state")), nil
}

// signIns0083 is the three sign-ins of the browser: OpenAI's, Grok's and the
// plan's, each with an issuer in process.
func signIns0083(t *testing.T) []signIn0083 {
	t.Helper()
	openai := func() signIn0083 {
		mux := http.NewServeMux()
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		ti := newTestIssuer(mux, srv.URL)
		mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
			writeTestJSON(t, w, ti.discovery(t, nil))
		})
		mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, _ *http.Request) {
			writeTestJSON(t, w, ti.tokenResponse(t, "RS256", "test-client", "access", "refresh"))
		})
		return signIn0083{name: "OpenAI", login: func(t *testing.T, noLoopback bool, listen listenFunc) (*OAuthSession, error) {
			opts := OAuthFlowOptions{HTTPClient: srv.Client(), ClientID: "test-client", Issuer: srv.URL, NoLoopback: noLoopback, listen: listen}
			opts.OpenURL, opts.InputCode = pasteBack0074(func(raw string) (string, error) {
				state, err := state0083(raw)
				return "http://127.0.0.1/auth/callback?code=pasted-code&state=" + state, err
			})
			return LoginBrowserOAuth(ctx0083(t), llmprovider.ProviderOpenAI, opts)
		}}
	}
	grok := func() signIn0083 {
		mux := http.NewServeMux()
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		ti := newTestIssuer(mux, srv.URL)
		mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
			writeTestJSON(t, w, ti.discovery(t, map[string]string{"authorization_endpoint": srv.URL + "/authorize", "token_endpoint": srv.URL + "/token"}))
		})
		mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
			writeTestJSON(t, w, ti.tokenResponse(t, "ES256", "test-client", "access", "refresh"))
		})
		return signIn0083{name: "Grok", grok: true, login: func(t *testing.T, noLoopback bool, listen listenFunc) (*OAuthSession, error) {
			opts := OAuthFlowOptions{HTTPClient: srv.Client(), ClientID: "test-client", Issuer: srv.URL, NoLoopback: noLoopback, listen: listen}
			openURL, inputCode := pasteBack0074(func(raw string) (string, error) {
				state, err := state0083(raw)
				return "http://127.0.0.1/callback?code=pasted-code&state=" + state, err
			})
			opts.OpenURL = func(raw string) error {
				ti.captureNonce(raw)
				return openURL(raw)
			}
			opts.InputCode = inputCode
			return LoginBrowserOAuth(ctx0083(t), llmprovider.ProviderGrok, opts)
		}}
	}
	plan := func() signIn0083 {
		pl := newPlanLogin(t)
		return signIn0083{name: "plan", plan: true, login: func(t *testing.T, noLoopback bool, listen listenFunc) (*OAuthSession, error) {
			opts := pl.options()
			opts.NoLoopback, opts.listen = noLoopback, listen
			opts.OpenURL, opts.InputCode = pasteBack0074(pl.callbackURL)
			return LoginChatGPTPlan(ctx0083(t), opts)
		}}
	}
	swapPorts0083(t)
	return []signIn0083{openai(), grok(), plan()}
}

// TestNoLoopback_BindsNothing_0083 (ST19, ST20, ST63): under NoLoopback each of
// OpenAI's and Grok's browser sign-ins and the plan's completes by a paste with
// no bind attempted, which is shown through the hook every bind passes
// through; without NoLoopback each sign-in's bind reaches the hook, which
// shows the hook is on the path of every bind. Nothing outside the process can
// pass or fail it, as a dial of a redirect's port could.
func TestNoLoopback_BindsNothing_0083(t *testing.T) {
	for _, s := range signIns0083(t) {
		t.Run(s.name, func(t *testing.T) {
			rec := &listenRecorder0083{}
			session, err := s.login(t, true, rec.listen)
			if err != nil || session == nil {
				t.Fatalf("%s sign-in with NoLoopback = %v, %v; want a session by the paste", s.name, session, err)
			}
			if n := rec.count(); n != 0 {
				t.Errorf("%s sign-in with NoLoopback attempted %d bind(s) %q; want none", s.name, n, rec.binds)
			}
			control := &listenRecorder0083{}
			if _, err := s.login(t, false, control.listen); err == nil || !strings.Contains(err.Error(), "listenRecorder0083") {
				t.Errorf("%s sign-in without NoLoopback = %v; want the hook's refusal of its bind", s.name, err)
			}
			if control.count() == 0 {
				t.Errorf("%s sign-in without NoLoopback bound through no hook; want its bind to reach it", s.name)
			}
		})
	}
}

// TestPortsAdviceOtherFailures_0083 (ST30, ST32, ST70): a bind that fails for a
// reason other than a port in use keeps the prefix and points to the sign-in
// with no listener, in the ruled words, and says nothing of freeing ports;
// Grok's keeps its own prefix and gives the same advice.
func TestPortsAdviceOtherFailures_0083(t *testing.T) {
	const advice = "this host does not allow the callback listener: sign in with no listener and paste the address back " +
		"(in the wizard, answer that the browser is on another device; in code, set NoLoopback)"
	denied := func(network, address string) error {
		return fmt.Errorf("listen %s %s: bind: permission denied", network, address)
	}
	for _, s := range signIns0083(t) {
		t.Run(s.name, func(t *testing.T) {
			rec := &listenRecorder0083{fail: denied}
			_, err := s.login(t, false, rec.listen)
			if err == nil {
				t.Fatalf("%s sign-in = nil error; want the bind's failure", s.name)
			}
			prefix := "oauth: registered callback ports unavailable; "
			if s.grok {
				prefix = "oauth: listen for Grok callback: "
			}
			if !strings.HasPrefix(err.Error(), prefix+advice) || strings.Contains(err.Error(), "free port") ||
				!strings.Contains(err.Error(), "permission denied") {
				t.Errorf("error = %q; want %q, the advice, no \"free ports\", and the bind's reason", err, prefix)
			}
			if rec.count() == 0 {
				t.Errorf("no bind reached the hook")
			}
		})
	}
}

// TestPortsAdviceMixedFailure_0083 (ST68): when one registered port fails in use
// and the other for another reason, the error gives the "free ports" advice and
// still shows both reasons.
func TestPortsAdviceMixedFailure_0083(t *testing.T) {
	for _, s := range signIns0083(t) {
		if s.grok {
			continue
		}
		t.Run(s.name, func(t *testing.T) {
			ports := openaiLoopbackPorts
			rec := &listenRecorder0083{fail: func(network, address string) error {
				if strings.HasSuffix(address, ":"+strconv.Itoa(ports[0])) {
					return fmt.Errorf("listen %s %s: bind: address already in use", network, address)
				}
				return fmt.Errorf("listen %s %s: bind: permission denied", network, address)
			}}
			_, err := s.login(t, false, rec.listen)
			if err == nil {
				t.Fatalf("%s sign-in = nil error; want the bind's failure", s.name)
			}
			want := fmt.Sprintf("free ports %d and %d", ports[0], ports[1])
			if !strings.HasPrefix(err.Error(), "oauth: registered callback ports unavailable; "+want) ||
				!strings.Contains(err.Error(), "address already in use") || !strings.Contains(err.Error(), "permission denied") {
				t.Errorf("error = %q; want the %q advice and both reasons", err, want)
			}
		})
	}
}
