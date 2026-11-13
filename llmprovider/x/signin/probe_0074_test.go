package signin

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/internal/signinflow"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// bounded is a context that ends after 5 s of real time, so that a sign-in
// that should have ended fails its test instead of hanging it.
func bounded(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// testState is the state of the authorize URL the probe's login shows.
const testState = "state-0074"

// loginProbe stands in for auth's browser and plan logins: it shows an
// authorize URL through OpenURL, asks InputCode as auth's input goroutine
// does, and ends when the test says the loopback delivered (release), when
// InputCode returns a value, or when it returns an error, whose error it
// returns, as auth's does.
type loginProbe struct {
	release chan struct{}
	pasted  chan string
	options chan loginOptions
}

// loginOptions is what the engine gave a login.
type loginOptions struct {
	hasInput   bool
	noLoopback bool
	agentName  string
}

func newLoginProbe() *loginProbe {
	return &loginProbe{release: make(chan struct{}), pasted: make(chan string, 8), options: make(chan loginOptions, 1)}
}

// authorizeURL is the URL the probe shows.
func (p *loginProbe) authorizeURL() string {
	return "https://issuer.test/oauth/authorize?client_id=client-0074&state=" + url.QueryEscape(testState) +
		"&redirect_uri=" + url.QueryEscape("http://127.0.0.1:1455/auth/callback")
}

// run is the login: it opens the URL, asks for a paste, and waits.
func (p *loginProbe) run(ctx context.Context, provider llmprovider.ProviderID, opts loginOptions,
	openURL func(string) error, inputCode func(context.Context) (string, error)) (*auth.OAuthSession, error) {
	opts.hasInput = inputCode != nil
	p.options <- opts
	inputCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := openURL(p.authorizeURL()); err != nil {
		return nil, err
	}
	type result struct {
		value string
		err   error
	}
	got := make(chan result, 1)
	if inputCode != nil {
		go func() {
			value, err := inputCode(inputCtx)
			got <- result{value, err}
		}()
	}
	session := &auth.OAuthSession{Provider: provider, Access: "access-0123456789"}
	select {
	case <-p.release:
		return session, nil
	case r := <-got:
		if r.err != nil {
			return nil, r.err
		}
		p.pasted <- r.value
		return session, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// logins are the login functions that run the probe.
func (p *loginProbe) logins() signinflow.Logins {
	return signinflow.Logins{
		Browser: func(ctx context.Context, provider llmprovider.ProviderID, o auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
			return p.run(ctx, provider, loginOptions{noLoopback: o.NoLoopback}, o.OpenURL, o.InputCode)
		},
		Plan: func(ctx context.Context, o auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
			return p.run(ctx, llmprovider.ProviderOpenAI, loginOptions{noLoopback: o.NoLoopback, agentName: o.AgentName}, o.OpenURL, o.InputCode)
		},
	}
}

// deviceLogins is a device login that shows a code and waits until release.
func deviceLogins(release <-chan struct{}) signinflow.Logins {
	return signinflow.Logins{Device: func(_ context.Context, p llmprovider.ProviderID, _ auth.OAuthFlowOptions) (signinflow.DeviceFlow, error) {
		return signinflow.DeviceFlow{URL: "https://kilo.test/device", Code: "CODE-1", Expiry: time.Now().Add(time.Minute),
			Wait: func(ctx context.Context) (*auth.OAuthSession, error) {
				select {
				case <-release:
					return &auth.OAuthSession{Provider: p, Access: "access-0123456789"}, nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}}, nil
	}}
}

// given returns the options the engine gave the login, once it started.
func (p *loginProbe) given(t *testing.T) loginOptions {
	t.Helper()
	select {
	case o := <-p.options:
		return o
	case <-time.After(10 * time.Second):
		t.Fatal("the login was never started")
	}
	return loginOptions{}
}

// callback is a callback URL for the probe's state.
func callback(code string) string {
	return "http://127.0.0.1:1455/auth/callback?code=" + url.QueryEscape(code) + "&state=" + url.QueryEscape(testState)
}

// startBrowser starts the probe's browser sign-in for p.
func startBrowser(t *testing.T, probe *loginProbe, p llmprovider.ProviderID, o Options) *SignIn {
	t.Helper()
	run, err := startWith(background(), p, llmprovider.AuthBrowserOAuth, o, probe.logins())
	if err != nil {
		t.Fatalf("startWith(%s, browser_oauth) = %v", p, err)
	}
	return run
}

// wantStep fails t unless step is of kind with Seq seq.
func wantStep(t *testing.T, step authcontract.Step, kind authcontract.StepKind, seq int) {
	t.Helper()
	if step.Kind != kind || step.Seq != seq {
		t.Fatalf("step = %s #%d; want %s #%d", step.Kind, step.Seq, kind, seq)
	}
}
