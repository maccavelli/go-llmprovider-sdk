package signinflow

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

const testState = "state-0074"

func background() context.Context { return context.Background() }

// bounded is a context that ends after 5 s of real time, so that a sign-in
// that should have ended fails its test instead of hanging it.
func bounded(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// loopbackOnly is a method whose only completion is the loopback: no paste.
// No built-in method is one since Grok's paste was verified on 2026-11-13,
// but the engine's rule for a method with an empty Paste fact is held here
// (0074-MADR D3).
func loopbackOnly() authcontract.Method {
	return authcontract.Method{ID: "browser_oauth", Kind: authcontract.KindBrowser,
		Completion: authcontract.CompletionBrowserLoopback, Yields: authcontract.YieldsOAuthSession}
}

func pastable() authcontract.Method {
	return authcontract.Method{ID: "browser_oauth", Kind: authcontract.KindBrowser,
		Completion: authcontract.CompletionBrowserPaste, Paste: authcontract.PasteURLOrCode, Yields: authcontract.YieldsOAuthSession}
}

// browserProbe is a browser login that shows an authorize URL and ends when
// released, or by what InputCode returns.
type browserProbe struct {
	release  chan struct{}
	hadInput chan bool
}

func newBrowserProbe() *browserProbe {
	return &browserProbe{release: make(chan struct{}), hadInput: make(chan bool, 1)}
}

func (p *browserProbe) logins() Logins {
	return Logins{Browser: func(ctx context.Context, provider llmprovider.ProviderID, o auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		p.hadInput <- o.InputCode != nil
		_ = o.OpenURL("https://issuer.test/oauth/authorize?state=" + testState)
		inputs := make(chan error, 1)
		if o.InputCode != nil {
			inputCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			go func() { _, err := o.InputCode(inputCtx); inputs <- err }()
		}
		select {
		case <-p.release:
			return &auth.OAuthSession{Provider: provider, Access: "access-0123456789"}, nil
		case err := <-inputs:
			if err != nil {
				return nil, err
			}
			return &auth.OAuthSession{Provider: provider, Access: "access-0123456789"}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}}
}

func next(t *testing.T, run *Run) authcontract.Step {
	t.Helper()
	select {
	case step, ok := <-run.Steps():
		if !ok {
			t.Fatal("the steps channel closed")
		}
		return step
	case <-time.After(10 * time.Second):
		t.Fatal("no step in 10 s")
	}
	return authcontract.Step{}
}

// TestEngine_LoopbackOnlyMethod_0074: a browser method with no Paste fact
// gets an open_url with no accepts and no paste step, with NoPaste unset,
// and an InputCode the login is never given; done follows the loopback; an
// answer naming the open_url's Seq is ErrInvalidRequest (0074-MADR D3).
func TestEngine_LoopbackOnlyMethod_0074(t *testing.T) {
	probe := newBrowserProbe()
	run, err := Start(background(), Spec{Provider: llmprovider.ProviderGrok, Method: loopbackOnly(),
		Mode: ModeBrowser, Logins: probe.logins()})
	if err != nil {
		t.Fatalf("Start = %v", err)
	}
	open := next(t, run)
	if open.Kind != authcontract.StepOpenURL || open.Seq != 1 || open.Accepts != "" {
		t.Fatalf("first step = %+v; want open_url #1 with no accepts", open)
	}
	if <-probe.hadInput {
		t.Error("the login was given an InputCode for a method that takes no paste")
	}
	if err := run.Answer(background(), authcontract.Answer{Seq: open.Seq, Paste: "x"}); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("Answer to the open_url = %v; want ErrInvalidRequest", err)
	}
	close(probe.release)
	if _, err := run.Wait(bounded(t)); err != nil {
		t.Fatalf("Wait = %v", err)
	}
	if done := next(t, run); done.Kind != authcontract.StepDone || done.Seq != 2 {
		t.Errorf("last step = %+v; want done #2", done)
	}
	if _, ok := <-run.Steps(); ok {
		t.Error("a step after done")
	}
}

// TestEngine_NoLoopbackRefusals_0074: with NoLoopback, a method that takes
// no paste is ErrUnsupported, and NoPaste as well is ErrInvalidRequest; a
// device sign-in is not refused; each refusal is before any login ran.
func TestEngine_NoLoopbackRefusals_0074(t *testing.T) {
	ran := false
	logins := Logins{
		Browser: func(context.Context, llmprovider.ProviderID, auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
			ran = true
			return nil, errors.New("must not run")
		},
		Device: func(context.Context, llmprovider.ProviderID, auth.OAuthFlowOptions) (DeviceFlow, error) {
			return DeviceFlow{}, errors.New("device ran")
		},
	}
	for _, c := range []struct {
		name string
		spec Spec
		want error
	}{
		{"no paste fact", Spec{Method: loopbackOnly(), Mode: ModeBrowser, NoLoopback: true}, llmprovider.ErrUnsupported},
		{"no paste fact, plan", Spec{Method: loopbackOnly(), Mode: ModePlan, NoLoopback: true}, llmprovider.ErrUnsupported},
		{"NoPaste", Spec{Method: pastable(), Mode: ModeBrowser, NoLoopback: true, NoPaste: true}, llmprovider.ErrInvalidRequest},
		{"an unknown mode", Spec{Method: pastable(), Mode: 99}, llmprovider.ErrInvalidRequest},
		{"an import with no Import", Spec{Method: pastable(), Mode: ModeImport}, llmprovider.ErrInvalidRequest},
	} {
		c.spec.Logins = logins
		run, err := Start(background(), c.spec)
		if !errors.Is(err, c.want) || run != nil {
			t.Errorf("%s: Start = %v, %v; want %v", c.name, run, err, c.want)
		}
	}
	if ran {
		t.Error("a login ran before a refusal")
	}
	run, err := Start(background(), Spec{Method: loopbackOnly(), Mode: ModeDevice, NoLoopback: true, Logins: logins})
	if err != nil {
		t.Fatalf("a device sign-in with NoLoopback = %v; want no refusal", err)
	}
	if _, err := run.Wait(bounded(t)); err == nil || err.Error() != "device ran" {
		t.Errorf("Wait = %v; want the device login's own error", err)
	}
}

// TestStart_PasteIsSecret_0074: every paste step has Secret true; an answer
// sent with Secret a non-nil false to a paste step is masked in the engine's
// log, because the engine sets Secret from its own step (0073-MADR D6 rule
// 5).
func TestStart_PasteIsSecret_0074(t *testing.T) {
	const planted = "PLANTED-SECRET-PASTE"
	var logged bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logged, &slog.HandlerOptions{Level: slog.LevelDebug}))
	probe := newBrowserProbe()
	run, err := Start(background(), Spec{Provider: llmprovider.ProviderOpenAI, Method: pastable(),
		Mode: ModeBrowser, Logins: probe.logins(), Logger: logger})
	if err != nil {
		t.Fatalf("Start = %v", err)
	}
	next(t, run)
	ask := next(t, run)
	if ask.Kind != authcontract.StepPaste || !ask.Secret {
		t.Fatalf("paste step = %+v; want secret", ask)
	}
	open := false
	if err := run.Answer(background(), authcontract.Answer{Seq: ask.Seq, Secret: &open, Paste: "http://127.0.0.1/?state=" + testState + "&code=" + planted}); err != nil {
		t.Fatalf("Answer = %v", err)
	}
	if _, err := run.Wait(bounded(t)); err != nil {
		t.Fatalf("Wait = %v", err)
	}
	if strings.Contains(logged.String(), planted) {
		t.Errorf("the engine's log holds the paste: %s", logged.String())
	}
	if !strings.Contains(logged.String(), "answer") {
		t.Errorf("the engine logged nothing of the answer: %q", logged.String())
	}
}

// TestEngine_PasteChecks_0074: what pasteCheck accepts, asks again, and
// passes on, by accepts value.
func TestEngine_PasteChecks_0074(t *testing.T) {
	authorize := "https://issuer.test/authorize?state=" + testState
	url := func(query string) string { return "http://127.0.0.1:1455/auth/callback?" + query }
	for _, c := range []struct {
		name    string
		accepts authcontract.Paste
		paste   string
		bad     bool
	}{
		{"url with state and code", authcontract.PasteURLOrCode, url("code=c&state=" + testState), false},
		{"bare code", authcontract.PasteURLOrCode, "code-only", false},
		{"padded bare code", authcontract.PasteURLOrCode, "  code-only\n", false},
		{"empty", authcontract.PasteURLOrCode, "   ", true},
		{"another state", authcontract.PasteURLOrCode, url("code=c&state=other"), true},
		{"no state", authcontract.PasteURLOrCode, url("code=c"), true},
		{"no code", authcontract.PasteURLOrCode, url("state=" + testState), true},
		{"a denial is passed on", authcontract.PasteURLOrCode, url("error=access_denied&state=" + testState), false},
		{"a denial of another sign-in", authcontract.PasteURLOrCode, url("error=access_denied&state=other"), true},
		{"url_only wants a client", authcontract.PasteURLOnly, url("code=c&state=" + testState), true},
		{"url_only with a client", authcontract.PasteURLOnly, url("code=c&client_id=app&state=" + testState), false},
		{"url_only, a bare code", authcontract.PasteURLOnly, "code-only", true},
		{"code, a bare code", authcontract.PasteCode, "code-only", false},
		{"code, a url", authcontract.PasteCode, url("code=c&state=" + testState), true},
	} {
		reason := newPasteCheck(authorize, c.accepts).check(c.paste)
		if (reason != "") != c.bad {
			t.Errorf("%s: check = %q; bad = %v want %v", c.name, reason, reason != "", c.bad)
		}
		if strings.Contains(reason, "code-only") || strings.Contains(reason, "access_denied") {
			t.Errorf("%s: the reason holds the paste: %q", c.name, reason)
		}
	}
	for _, accepts := range []authcontract.Paste{authcontract.PasteURLOrCode, authcontract.PasteURLOnly, authcontract.PasteCode} {
		if promptFor(accepts) == "" {
			t.Errorf("no prompt for %q", accepts)
		}
	}
	// An authorize URL with no state checks nothing of the state.
	if reason := newPasteCheck("::", authcontract.PasteURLOrCode).check(url("code=c&state=any")); reason != "" {
		t.Errorf("an unreadable authorize URL: check = %q; want it to take the paste", reason)
	}
}

// TestEngine_DeviceStartFails_0074: a device login that cannot start ends
// the run with failed #1, kind by the error, and Wait's error.
func TestEngine_DeviceStartFails_0074(t *testing.T) {
	boom := fmt.Errorf("%w: too many pending", llmprovider.ErrRateLimited)
	run, err := Start(background(), Spec{Method: authcontract.Method{Yields: authcontract.YieldsDeviceKey}, Mode: ModeDevice,
		Logins: Logins{Device: func(context.Context, llmprovider.ProviderID, auth.OAuthFlowOptions) (DeviceFlow, error) {
			return DeviceFlow{}, boom
		}}})
	if err != nil {
		t.Fatalf("Start = %v", err)
	}
	if _, err := run.Wait(bounded(t)); !errors.Is(err, boom) {
		t.Fatalf("Wait = %v; want the login's error", err)
	}
	step := next(t, run)
	if step.Kind != authcontract.StepFailed || step.Seq != 1 || step.Error == nil ||
		step.Error.Kind != authcontract.FailureRateLimited || !step.Error.Retryable {
		t.Errorf("step = %+v; want failed #1, rate_limited, retryable", step)
	}
}

// TestEngine_CancelAndContexts_0074: a Cancel before the device login has
// its flow still ends canceled and cancels the flow; Wait and Answer return
// their own ctx's error; a canceled parent ctx ends the run, kind canceled.
func TestEngine_CancelAndContexts_0074(t *testing.T) {
	canceled := make(chan struct{})
	started := make(chan struct{})
	logins := Logins{Device: func(context.Context, llmprovider.ProviderID, auth.OAuthFlowOptions) (DeviceFlow, error) {
		close(started)
		return DeviceFlow{URL: "https://issuer.test/device", Code: "C-1", Expiry: time.Now().Add(time.Minute),
			Wait:   func(ctx context.Context) (*auth.OAuthSession, error) { <-ctx.Done(); return nil, ctx.Err() },
			Cancel: func() { close(canceled) }}, nil
	}}
	run, err := Start(background(), Spec{Method: authcontract.Method{}, Mode: ModeDevice, Logins: logins})
	if err != nil {
		t.Fatalf("Start = %v", err)
	}
	<-started
	waiting := next(t, run)
	if waiting.Kind != authcontract.StepShowCode {
		t.Fatalf("first step = %+v", waiting)
	}
	ended, cancel := context.WithCancel(background())
	cancel()
	if _, err := run.Wait(ended); !errors.Is(err, context.Canceled) {
		t.Errorf("Wait with an ended ctx = %v; want context.Canceled", err)
	}
	if err := run.Answer(ended, authcontract.Answer{Seq: 1}); !errors.Is(err, context.Canceled) {
		t.Errorf("Answer with an ended ctx = %v; want context.Canceled", err)
	}
	next(t, run)
	run.Cancel()
	if _, err := run.Wait(bounded(t)); !errors.Is(err, context.Canceled) {
		t.Errorf("Wait after Cancel = %v; want context.Canceled", err)
	}
	select {
	case <-canceled:
	case <-time.After(10 * time.Second):
		t.Error("the flow's Cancel was not called")
	}

	parent, stop := context.WithCancel(background())
	run, err = Start(parent, Spec{Method: authcontract.Method{}, Mode: ModeImport,
		Import: func(ctx context.Context) (*auth.VendorCLISession, error) { <-ctx.Done(); return nil, ctx.Err() }})
	if err != nil {
		t.Fatalf("Start(import) = %v", err)
	}
	stop()
	_, err = run.Wait(bounded(t))
	if ErrorKindOf(err) != authcontract.FailureCanceled {
		t.Errorf("Wait after the parent ended = %v; want kind canceled", err)
	}
}

// TestEngine_Failure_0074: Failure carries the kind, the text without
// control characters, and Retryable from the APIError, else from the kind.
func TestEngine_Failure_0074(t *testing.T) {
	retry := &llmprovider.APIError{Kind: llmprovider.ErrRateLimited, Status: 429}
	final := &llmprovider.APIError{Kind: llmprovider.ErrQuotaExhausted, Status: 429}
	for _, c := range []struct {
		name      string
		err       error
		kind      authcontract.ErrorKind
		retryable bool
	}{
		{"an APIError that retries", retry, authcontract.FailureRateLimited, true},
		{"an APIError that does not", final, authcontract.FailureQuota, false},
		{"offline", llmprovider.ErrOffline, authcontract.FailureOffline, true},
		{"unavailable", llmprovider.ErrProviderUnavailable, authcontract.FailureUnavailable, true},
		{"denied", auth.ErrSignInDenied, authcontract.FailureDenied, false},
		{"other", errors.New("bad\x1b[2Jtext"), authcontract.FailureOther, false},
	} {
		got := Failure(c.err)
		if got.Kind != c.kind || got.Retryable != c.retryable || strings.ContainsRune(got.Message, 0x1b) || got.Message == "" {
			t.Errorf("%s: Failure = %+v; want kind %q, retryable %v, a message with no control character", c.name, got, c.kind, c.retryable)
		}
	}
}
