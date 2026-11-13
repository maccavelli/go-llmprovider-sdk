package signin

import (
	"errors"
	"net"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// browserProviders are the providers whose browser_oauth method takes a paste
// (Grok's, since the live probe of 2026-11-13).
var browserProviders = []llmprovider.ProviderID{llmprovider.ProviderOpenAI, llmprovider.ProviderGrok}

// TestStart_BrowserSteps_0074: a browser sign-in's open_url carries accepts,
// then comes the paste step, then done: by the loopback, or by a paste
// answer. NoPaste emits no paste step and no accepts. A paste step is secret
// (0074-MADR D3).
func TestStart_BrowserSteps_0074(t *testing.T) {
	for _, p := range browserProviders {
		t.Run(string(p)+"/loopback", func(t *testing.T) {
			probe := newLoginProbe()
			run := startBrowser(t, probe, p, Options{})
			open, ask := nextStep(t, run), nextStep(t, run)
			wantStep(t, open, authcontract.StepOpenURL, 1)
			wantStep(t, ask, authcontract.StepPaste, 2)
			if open.URL != probe.authorizeURL() || open.Accepts != authcontract.PasteURLOrCode || open.Secret {
				t.Errorf("open_url = URL %q, accepts %q, secret %v", open.URL, open.Accepts, open.Secret)
			}
			if ask.Accepts != authcontract.PasteURLOrCode || !ask.Secret || ask.Text == "" {
				t.Errorf("paste = accepts %q, secret %v, text %q; want url_or_code, secret, a prompt", ask.Accepts, ask.Secret, ask.Text)
			}
			close(probe.release)
			credential, err := run.Wait(bounded(t))
			if err != nil || credential.Yields() != authcontract.YieldsOAuthSession {
				t.Fatalf("Wait = %v, %v; want a session", credential, err)
			}
			rest := collect(t, run)
			if len(rest) != 1 || rest[0].Kind != authcontract.StepDone || rest[0].Seq != 3 || rest[0].Yields != authcontract.YieldsOAuthSession {
				t.Errorf("after the loopback: %+v; want done #3 with oauth_session", rest)
			}
		})
		t.Run(string(p)+"/paste", func(t *testing.T) {
			probe := newLoginProbe()
			run := startBrowser(t, probe, p, Options{})
			wantStep(t, nextStep(t, run), authcontract.StepOpenURL, 1)
			ask := nextStep(t, run)
			wantStep(t, ask, authcontract.StepPaste, 2)
			pasted := callback("code-0074")
			if err := run.Answer(background(), authcontract.Answer{Seq: ask.Seq, Paste: pasted}); err != nil {
				t.Fatalf("Answer = %v", err)
			}
			if _, err := run.Wait(bounded(t)); err != nil {
				t.Fatalf("Wait = %v", err)
			}
			if got := <-probe.pasted; got != pasted {
				t.Errorf("the login was given %q; want the pasted callback", got)
			}
			rest := collect(t, run)
			if len(rest) != 1 || rest[0].Kind != authcontract.StepDone || rest[0].Seq != 3 {
				t.Errorf("after the paste: %+v; want done #3", rest)
			}
		})
		t.Run(string(p)+"/NoPaste", func(t *testing.T) {
			probe := newLoginProbe()
			run := startBrowser(t, probe, p, Options{NoPaste: true})
			open := nextStep(t, run)
			wantStep(t, open, authcontract.StepOpenURL, 1)
			if open.Accepts != "" {
				t.Errorf("open_url accepts %q under NoPaste; want none", open.Accepts)
			}
			if given := probe.given(t); given.hasInput {
				t.Error("the login was given an InputCode under NoPaste")
			}
			close(probe.release)
			if _, err := run.Wait(bounded(t)); err != nil {
				t.Fatalf("Wait = %v", err)
			}
			rest := collect(t, run)
			if len(rest) != 1 || rest[0].Kind != authcontract.StepDone || rest[0].Seq != 2 {
				t.Errorf("after the loopback: %+v; want done #2 and no paste step", rest)
			}
		})
	}
}

// TestStart_PlanSteps_0074: the plan's first paste is url_only, its
// FirstPaste; a sign-in again with Previous takes the method's Paste; the
// plan needs its AgentName before any request (0074-MADR D3).
func TestStart_PlanSteps_0074(t *testing.T) {
	for _, c := range []struct {
		name     string
		previous *auth.OAuthSession
		want     authcontract.Paste
	}{
		{"first", nil, authcontract.PasteURLOnly},
		{"again", &auth.OAuthSession{ClientID: "client-0074"}, authcontract.PasteURLOrCode},
	} {
		t.Run(c.name, func(t *testing.T) {
			probe := newLoginProbe()
			run, err := startWith(background(), llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan,
				Options{AgentName: "My App", Previous: c.previous}, probe.logins())
			if err != nil {
				t.Fatalf("start = %v", err)
			}
			open, ask := nextStep(t, run), nextStep(t, run)
			wantStep(t, open, authcontract.StepOpenURL, 1)
			wantStep(t, ask, authcontract.StepPaste, 2)
			if open.Accepts != c.want || ask.Accepts != c.want {
				t.Errorf("accepts = %q and %q; want %q", open.Accepts, ask.Accepts, c.want)
			}
			if given := probe.given(t); given.agentName != "My App" {
				t.Errorf("the plan login was given AgentName %q", given.agentName)
			}
			close(probe.release)
			if _, err := run.Wait(bounded(t)); err != nil {
				t.Fatalf("Wait = %v", err)
			}
		})
	}
	_, err := Start(background(), llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan, Options{})
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("Start without an AgentName = %v; want ErrInvalidRequest", err)
	}
}

// TestStart_BadPaste_0074: a paste with another state, one with no code, and
// a bare code where url_only is accepted each bring a new paste step whose
// text does not hold the paste; of three bad pastes the first two are
// asked again and the third ends the sign-in, failed invalid_request
// (0074-MADR D4).
func TestStart_BadPaste_0074(t *testing.T) {
	const planted = "PLANTED-PASTE-VALUE"
	wrongState := "http://127.0.0.1:1455/auth/callback?code=" + planted + "&state=another-sign-in"
	noCode := "http://127.0.0.1:1455/auth/callback?state=" + testState + "&note=" + planted
	t.Run("each kind is asked again", func(t *testing.T) {
		probe := newLoginProbe()
		run := startBrowser(t, probe, llmprovider.ProviderOpenAI, Options{})
		wantStep(t, nextStep(t, run), authcontract.StepOpenURL, 1)
		ask := nextStep(t, run)
		for i, bad := range []string{wrongState, noCode} {
			if err := run.Answer(background(), authcontract.Answer{Seq: ask.Seq, Paste: bad}); err != nil {
				t.Fatalf("answer %d = %v", i+1, err)
			}
			again := nextStep(t, run)
			wantStep(t, again, authcontract.StepPaste, ask.Seq+1)
			if !again.Secret || again.Text == "" || strings.Contains(again.Text, planted) || again.Text == ask.Text {
				t.Errorf("re-ask %d = secret %v, text %q; want a secret paste step whose text says why and holds no paste", i+1, again.Secret, again.Text)
			}
			ask = again
		}
		if err := run.Answer(background(), authcontract.Answer{Seq: ask.Seq, Paste: callback("good")}); err != nil {
			t.Fatalf("the good answer = %v", err)
		}
		if _, err := run.Wait(bounded(t)); err != nil {
			t.Fatalf("Wait = %v", err)
		}
	})
	t.Run("a bare code where url_only is accepted", func(t *testing.T) {
		probe := newLoginProbe()
		run, err := startWith(background(), llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan,
			Options{AgentName: "My App"}, probe.logins())
		if err != nil {
			t.Fatalf("start = %v", err)
		}
		wantStep(t, nextStep(t, run), authcontract.StepOpenURL, 1)
		ask := nextStep(t, run)
		if err := run.Answer(background(), authcontract.Answer{Seq: ask.Seq, Paste: planted}); err != nil {
			t.Fatalf("Answer = %v", err)
		}
		again := nextStep(t, run)
		wantStep(t, again, authcontract.StepPaste, 3)
		if strings.Contains(again.Text, planted) || again.Text == "" {
			t.Errorf("re-ask text = %q; want a reason with no paste", again.Text)
		}
		run.Cancel()
		_, _ = run.Wait(bounded(t))
	})
	t.Run("the third bad paste ends it", func(t *testing.T) {
		probe := newLoginProbe()
		run := startBrowser(t, probe, llmprovider.ProviderOpenAI, Options{})
		wantStep(t, nextStep(t, run), authcontract.StepOpenURL, 1)
		ask := nextStep(t, run)
		for i := range 2 {
			if err := run.Answer(background(), authcontract.Answer{Seq: ask.Seq, Paste: wrongState}); err != nil {
				t.Fatalf("answer %d = %v", i+1, err)
			}
			ask = nextStep(t, run)
			wantStep(t, ask, authcontract.StepPaste, 3+i)
		}
		if err := run.Answer(background(), authcontract.Answer{Seq: ask.Seq, Paste: wrongState}); err != nil {
			t.Fatalf("the third answer = %v", err)
		}
		_, err := run.Wait(bounded(t))
		if !errors.Is(err, llmprovider.ErrInvalidRequest) || ErrorKindOf(err) != authcontract.FailureInvalidRequest {
			t.Fatalf("Wait = %v; want invalid_request after three bad pastes", err)
		}
		rest := collect(t, run)
		if len(rest) != 1 || rest[0].Kind != authcontract.StepFailed || rest[0].Seq != 5 ||
			rest[0].Error == nil || rest[0].Error.Kind != authcontract.FailureInvalidRequest {
			t.Fatalf("after the third paste: %+v; want one failed #5 with invalid_request", rest)
		}
		if strings.Contains(rest[0].Error.Message, planted) {
			t.Errorf("the failure message holds the paste: %q", rest[0].Error.Message)
		}
		if err := run.Answer(background(), authcontract.Answer{Seq: ask.Seq, Paste: callback("late")}); !errors.Is(err, llmprovider.ErrInvalidRequest) {
			t.Errorf("a fourth answer = %v; want ErrInvalidRequest", err)
		}
	})
}

// TestStart_PastedDenial_0074: a pasted URL with this sign-in's state and
// error=access_denied, as the first paste, ends failed denied at once, with
// no re-ask: auth's parsing ends it, on the real code against a fake issuer
// (0074-MADR D4).
func TestStart_PastedDenial_0074(t *testing.T) {
	fake := &fakeIssuer{}
	run, err := Start(background(), llmprovider.ProviderOpenAI, llmprovider.AuthBrowserOAuth,
		Options{HTTPClient: fake.client(), Issuer: fakeBase, ClientID: "test-client", NoLoopback: true})
	if err != nil {
		t.Fatalf("Start = %v", err)
	}
	open := nextStep(t, run)
	ask := nextStep(t, run)
	wantStep(t, open, authcontract.StepOpenURL, 1)
	wantStep(t, ask, authcontract.StepPaste, 2)
	authorize, err := url.Parse(open.URL)
	if err != nil {
		t.Fatalf("open_url's URL = %q: %v", open.URL, err)
	}
	denial := "http://127.0.0.1:1455/auth/callback?error=access_denied&state=" + url.QueryEscape(authorize.Query().Get("state"))
	if err := run.Answer(background(), authcontract.Answer{Seq: ask.Seq, Paste: denial}); err != nil {
		t.Fatalf("Answer = %v", err)
	}
	_, err = run.Wait(bounded(t))
	if !errors.Is(err, auth.ErrSignInDenied) || ErrorKindOf(err) != authcontract.FailureDenied {
		t.Fatalf("Wait = %v; want denied", err)
	}
	rest := collect(t, run)
	if len(rest) != 1 || rest[0].Kind != authcontract.StepFailed || rest[0].Seq != 3 ||
		rest[0].Error == nil || rest[0].Error.Kind != authcontract.FailureDenied {
		t.Errorf("after the denial: %+v; want one failed #3 with denied and no re-ask", rest)
	}
}

// TestStart_NoLoopback_0074: with NoLoopback, a browser sign-in binds no
// port and completes by a paste, for OpenAI and for Grok; NoLoopback with
// NoPaste is ErrInvalidRequest, before any request. The real auth code runs
// against a fake issuer (0074-MADR D4).
func TestStart_NoLoopback_0074(t *testing.T) {
	for _, p := range browserProviders {
		t.Run(string(p), func(t *testing.T) {
			fake := &fakeIssuer{}
			run, err := Start(background(), p, llmprovider.AuthBrowserOAuth,
				Options{HTTPClient: fake.client(), Issuer: fakeBase, ClientID: "test-client", NoLoopback: true})
			if err != nil {
				t.Fatalf("Start = %v", err)
			}
			open := nextStep(t, run)
			ask := nextStep(t, run)
			wantStep(t, open, authcontract.StepOpenURL, 1)
			wantStep(t, ask, authcontract.StepPaste, 2)
			authorize, err := url.Parse(open.URL)
			if err != nil {
				t.Fatalf("open_url's URL = %q: %v", open.URL, err)
			}
			query := authorize.Query()
			assertNothingListens(t, p, query.Get("redirect_uri"))
			fake.setNonce(open.URL)
			pasted := "http://127.0.0.1:1455/callback?code=code-0074&state=" + url.QueryEscape(query.Get("state"))
			if err := run.Answer(background(), authcontract.Answer{Seq: ask.Seq, Paste: pasted}); err != nil {
				t.Fatalf("Answer = %v", err)
			}
			credential, err := run.Wait(bounded(t))
			if err != nil {
				t.Fatalf("Wait = %v", err)
			}
			if _, ok := credential.(Session); !ok {
				t.Errorf("credential is a %T; want a Session", credential)
			}
			rest := collect(t, run)
			if len(rest) != 1 || rest[0].Kind != authcontract.StepDone || rest[0].Seq != 3 {
				t.Errorf("after the paste: %+v; want done #3", rest)
			}
		})
	}
	t.Run("NoPaste", func(t *testing.T) {
		for _, p := range browserProviders {
			fake := &fakeIssuer{}
			_, err := Start(background(), p, llmprovider.AuthBrowserOAuth,
				Options{HTTPClient: fake.client(), NoLoopback: true, NoPaste: true})
			if !errors.Is(err, llmprovider.ErrInvalidRequest) {
				t.Errorf("%s: Start with NoLoopback and NoPaste = %v; want ErrInvalidRequest", p, err)
			}
			if got := fake.seen(); len(got) != 0 {
				t.Errorf("%s: requests were sent before the refusal: %v", p, got)
			}
		}
	})
}

// assertNothingListens fails t when the redirect URI of Grok's no-listener
// sign-in has a listener behind it. OpenAI's names a registered port that
// another process may hold, so only its shape is checked.
func assertNothingListens(t *testing.T, p llmprovider.ProviderID, redirect string) {
	t.Helper()
	target, err := url.Parse(redirect)
	if err != nil || target.Host == "" {
		t.Fatalf("redirect_uri = %q", redirect)
	}
	if p != llmprovider.ProviderGrok {
		return
	}
	port, err := strconv.Atoi(target.Port())
	if err != nil || port < 49152 || port > 65535 || target.Path != "/callback" {
		t.Fatalf("Grok's redirect_uri = %q; want a port in 49152 to 65535 and /callback", redirect)
	}
	conn, err := net.DialTimeout("tcp", target.Host, time.Second)
	if err == nil {
		_ = conn.Close()
		t.Errorf("something listens at %s; want no listener under NoLoopback", target.Host)
	}
}

// TestStart_NoLoopbackPassedOn_0074: the plan and the browser logins are
// given NoLoopback, and an InputCode to complete it by.
func TestStart_NoLoopbackPassedOn_0074(t *testing.T) {
	probe := newLoginProbe()
	run := startBrowser(t, probe, llmprovider.ProviderOpenAI, Options{NoLoopback: true})
	if given := probe.given(t); !given.noLoopback || !given.hasInput {
		t.Errorf("browser login was given %+v; want NoLoopback and an InputCode", given)
	}
	run.Cancel()
	_, _ = run.Wait(bounded(t))

	probe = newLoginProbe()
	run, err := startWith(background(), llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan,
		Options{AgentName: "My App", NoLoopback: true}, probe.logins())
	if err != nil {
		t.Fatalf("start = %v", err)
	}
	if given := probe.given(t); !given.noLoopback || !given.hasInput {
		t.Errorf("plan login was given %+v; want NoLoopback and an InputCode", given)
	}
	run.Cancel()
	_, _ = run.Wait(bounded(t))
}

// TestStart_Answers_0074: an answer to another Seq, to each step that does
// not ask (open_url with and without accepts, show_code, waiting), a second
// answer to an answered step, or one after the end is ErrInvalidRequest; the
// paste step's own Seq completes a browser sign-in (0074-MADR D3).
func TestStart_Answers_0074(t *testing.T) {
	wantInvalid := func(t *testing.T, what string, err error) {
		t.Helper()
		if !errors.Is(err, llmprovider.ErrInvalidRequest) {
			t.Errorf("%s: Answer = %v; want ErrInvalidRequest", what, err)
		}
	}
	for _, p := range browserProviders {
		t.Run(string(p)+"/browser", func(t *testing.T) {
			probe := newLoginProbe()
			run := startBrowser(t, probe, p, Options{})
			open, ask := nextStep(t, run), nextStep(t, run)
			wantInvalid(t, "another Seq", run.Answer(background(), authcontract.Answer{Seq: 99, Paste: callback("x")}))
			wantInvalid(t, "open_url with accepts", run.Answer(background(), authcontract.Answer{Seq: open.Seq, Paste: callback("x")}))
			// A bad paste is answered, and its own paste step is asked again.
			if err := run.Answer(background(), authcontract.Answer{Seq: ask.Seq, Paste: "http://127.0.0.1/?state=other"}); err != nil {
				t.Fatalf("first answer = %v", err)
			}
			wantInvalid(t, "a second answer", run.Answer(background(), authcontract.Answer{Seq: ask.Seq, Paste: callback("y")}))
			again := nextStep(t, run)
			wantStep(t, again, authcontract.StepPaste, 3)
			if err := run.Answer(background(), authcontract.Answer{Seq: again.Seq, Paste: callback("good")}); err != nil {
				t.Fatalf("the paste step's own Seq = %v; want it to complete the sign-in", err)
			}
			if _, err := run.Wait(bounded(t)); err != nil {
				t.Fatalf("Wait = %v", err)
			}
			wantInvalid(t, "after the end", run.Answer(background(), authcontract.Answer{Seq: again.Seq, Paste: callback("z")}))
		})
	}
	t.Run("open_url without accepts", func(t *testing.T) {
		probe := newLoginProbe()
		run := startBrowser(t, probe, llmprovider.ProviderOpenAI, Options{NoPaste: true})
		open := nextStep(t, run)
		wantInvalid(t, "open_url without accepts", run.Answer(background(), authcontract.Answer{Seq: open.Seq, Paste: callback("x")}))
		close(probe.release)
		_, _ = run.Wait(bounded(t))
	})
	t.Run("show_code and waiting", func(t *testing.T) {
		release := make(chan struct{})
		logins := deviceLogins(release)
		run, err := startWith(background(), llmprovider.ProviderKilo, llmprovider.AuthDeviceCode, Options{}, logins)
		if err != nil {
			t.Fatalf("start = %v", err)
		}
		show, waiting := nextStep(t, run), nextStep(t, run)
		wantStep(t, show, authcontract.StepShowCode, 1)
		wantStep(t, waiting, authcontract.StepWaiting, 2)
		wantInvalid(t, "show_code", run.Answer(background(), authcontract.Answer{Seq: show.Seq, Paste: "x"}))
		wantInvalid(t, "waiting", run.Answer(background(), authcontract.Answer{Seq: waiting.Seq, Paste: "x"}))
		close(release)
		if _, err := run.Wait(bounded(t)); err != nil {
			t.Fatalf("Wait = %v", err)
		}
	})
}

// TestStart_Unsupported_0074: api_key, token_stdin and an unknown pair are
// ErrUnsupported (0074-MADR D3).
func TestStart_Unsupported_0074(t *testing.T) {
	for _, c := range []struct {
		provider llmprovider.ProviderID
		method   llmprovider.AuthMethodID
	}{
		{llmprovider.ProviderOpenAI, llmprovider.AuthAPIKey},
		{llmprovider.ProviderOpenAI, llmprovider.AuthTokenStdin},
		{llmprovider.ProviderGrok, llmprovider.AuthTokenStdin},
		{llmprovider.ProviderKilo, llmprovider.AuthBrowserOAuth},
		{llmprovider.ProviderClaude, llmprovider.AuthDeviceCode},
		{llmprovider.ProviderOpenAI, "no_such_method"},
	} {
		run, err := Start(background(), c.provider, c.method, Options{})
		if !errors.Is(err, llmprovider.ErrUnsupported) || !errors.Is(err, errors.ErrUnsupported) || run != nil {
			t.Errorf("Start(%s, %s) = %v, %v; want ErrUnsupported", c.provider, c.method, run, err)
		}
	}
}

// TestStart_Methods_0074: every pair Start drives reports the facts the
// descriptor's method has, and the step kinds follow them.
func TestStart_Methods_0074(t *testing.T) {
	var pairs []pair
	for p := range driven {
		pairs = append(pairs, p)
	}
	slices.SortFunc(pairs, func(a, b pair) int {
		return strings.Compare(string(a.provider)+"/"+string(a.method), string(b.provider)+"/"+string(b.method))
	})
	if len(pairs) != 8 {
		t.Fatalf("%d pairs driven; want 8: device_code for three, browser_oauth for two, the plan, and two imports", len(pairs))
	}
	for _, p := range pairs {
		method, ok := methodFor(p.provider, p.method)
		if !ok || method.ID != string(p.method) {
			t.Errorf("methodFor(%s, %s) = %+v, %v", p.provider, p.method, method, ok)
		}
	}
	if _, ok := methodFor(llmprovider.ProviderOpenAI, llmprovider.AuthAPIKey); ok {
		t.Error("methodFor reports a value method as driven")
	}
}
