package wizard

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// terminals returns the done and failed steps a driver recorded, and
// whether the last step it recorded is one of them.
func (d *recordingDriver0074) terminals() (terminal []authcontract.Step, last bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, s := range d.steps {
		if s.Kind == authcontract.StepDone || s.Kind == authcontract.StepFailed {
			terminal = append(terminal, s)
		}
	}
	if n := len(d.steps); n > 0 {
		k := d.steps[n-1].Kind
		last = k == authcontract.StepDone || k == authcontract.StepFailed
	}
	return terminal, last
}

// resolveBounded0074 runs ResolveCredential for Grok's browser sign-in with
// o, failing t if it has not returned in 5 s.
func resolveBounded0074(t *testing.T, ctx context.Context, f *fakePrompter, o Options) (Result, error) {
	t.Helper()
	type outcome struct {
		res Result
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		res, _, err := ResolveCredential(ctx, f, o, llmprovider.ProviderGrok)
		done <- outcome{res, err}
	}()
	select {
	case r := <-done:
		return r.res, r.err
	case <-time.After(5 * time.Second):
		t.Fatal("the sign-in did not return within 5 s")
		return Result{}, nil
	}
}

// TestDriver_TerminalDone_0074 (0074-MADR D6, "The terminal step"): a
// successful sign-in gives the Driver exactly one terminal step, last, a
// done that carries the method's Yields.
func TestDriver_TerminalDone_0074(t *testing.T) {
	stubGrokDevice0057(t, grokCode0057)
	d := &recordingDriver0074{}
	f := newFake(t, fakePrompter{selects: []int{grokDeviceIdx}})
	res, err := resolveBounded0074(t, context.Background(), f, Options{TokenStore: newMemoryTokenStore(),
		NoEndpointPrompt: true, Driver: d})
	if err != nil || res.Kind != CredOAuth {
		t.Fatalf("kind %q, error %v; want the session", res.Kind, err)
	}
	terminal, last := d.terminals()
	if len(terminal) != 1 || !last || terminal[0].Kind != authcontract.StepDone {
		t.Fatalf("terminal steps %+v, last %v; want one done, last", terminal, last)
	}
	if terminal[0].Yields != authcontract.YieldsOAuthSession || terminal[0].Seq != 3 {
		t.Errorf("done step %+v; want Yields %q, Seq 3", terminal[0], authcontract.YieldsOAuthSession)
	}
}

// TestDriver_TerminalFailed_0074: a failed sign-in gives the Driver exactly
// one terminal step, last, a failed that carries the error's kind, for the
// login's failure and for the caller's ctx ending alike; the error is
// returned as the login gave it.
func TestDriver_TerminalFailed_0074(t *testing.T) {
	denied := fmt.Errorf("grok: %w", auth.ErrSignInDenied)
	for _, c := range []struct {
		name string
		want authcontract.ErrorKind
		run  func(cancel context.CancelFunc) error // the login's failure
	}{
		{"denied", authcontract.FailureDenied, func(context.CancelFunc) error { return denied }},
		{"caller canceled", authcontract.FailureCanceled, func(cancel context.CancelFunc) error {
			cancel()
			return context.Canceled
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			stubBrowserLogin(t, func(_ context.Context, _ llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
				_ = opts.OpenURL("https://authorize.test")
				return nil, c.run(cancel)
			})
			d := &recordingDriver0074{}
			f := newFake(t, fakePrompter{selects: []int{grokBrowserIdx}})
			_, err := resolveBounded0074(t, ctx, f, Options{TokenStore: newMemoryTokenStore(), NoEndpointPrompt: true,
				Driver: d})
			if err == nil {
				t.Fatal("the sign-in succeeded; want its failure")
			}
			terminal, last := d.terminals()
			if len(terminal) != 1 || !last || terminal[0].Kind != authcontract.StepFailed {
				t.Fatalf("terminal steps %+v, last %v; want one failed, last", terminal, last)
			}
			if got := terminal[0].Error; got == nil || got.Kind != c.want {
				t.Errorf("failed step's error %+v; want kind %q", got, c.want)
			}
		})
	}
}

// TestDriver_OwnErrorGetsNoTerminal_0074 (the owner's ruling "No final step
// after its own error"): an error the Driver returns on open_url, on paste or
// on show_code ends the sign-in, is returned as it is, and the Driver is
// handed no terminal step.
func TestDriver_OwnErrorGetsNoTerminal_0074(t *testing.T) {
	errStep := errors.New("the event could not be written")
	for _, kind := range []authcontract.StepKind{authcontract.StepOpenURL, authcontract.StepPaste, authcontract.StepShowCode} {
		t.Run(string(kind), func(t *testing.T) {
			stubGrokDevice0057(t, grokCode0057)
			stubBrowserLogin(t, func(ctx context.Context, _ llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
				_ = opts.OpenURL("https://authorize.test")
				if _, err := opts.InputCode(ctx); err != nil {
					return nil, err
				}
				<-ctx.Done() // the sign-in goes on until the wizard ends it
				return nil, ctx.Err()
			})
			d := &recordingDriver0074{answer: func(_ context.Context, s authcontract.Step) (authcontract.Answer, error) {
				if s.Kind == kind {
					return authcontract.Answer{}, errStep
				}
				return authcontract.Answer{Paste: "code-0074"}, nil
			}}
			idx := grokBrowserIdx
			if kind == authcontract.StepShowCode {
				idx = grokDeviceIdx
			}
			f := newFake(t, fakePrompter{selects: []int{idx}})
			_, err := resolveBounded0074(t, context.Background(), f, Options{TokenStore: newMemoryTokenStore(),
				NoEndpointPrompt: true, Driver: d})
			if !errors.Is(err, errStep) || err.Error() != errStep.Error() {
				t.Errorf("error %v; want the Driver's error as it is", err)
			}
			if terminal, _ := d.terminals(); len(terminal) != 0 {
				t.Errorf("the Driver was handed %+v after its own error; want no terminal step", terminal)
			}
		})
	}
}

// TestDriver_TerminalErrorWarns_0074 (the owner's ruling "Keep the
// credential, warn", and the coordinator's reading for failed): an error the
// Driver returns when handed done leaves the credential; one returned when
// handed failed leaves the sign-in's own error. Each is one warning notice.
func TestDriver_TerminalErrorWarns_0074(t *testing.T) {
	errStep := errors.New("the last line could not be written")
	failTerminal := func(_ context.Context, s authcontract.Step) (authcontract.Answer, error) {
		if s.Kind == authcontract.StepDone || s.Kind == authcontract.StepFailed {
			return authcontract.Answer{}, errStep
		}
		return authcontract.Answer{}, nil
	}
	warnings := func(f *fakePrompter) int { return countContaining(f.seenNotify, "returned an error on the") }
	t.Run("done keeps the credential", func(t *testing.T) {
		stubGrokDevice0057(t, grokCode0057)
		f := newFake(t, fakePrompter{selects: []int{grokDeviceIdx}})
		res, err := resolveBounded0074(t, context.Background(), f, Options{TokenStore: newMemoryTokenStore(),
			NoEndpointPrompt: true, Driver: &recordingDriver0074{answer: failTerminal}})
		if err != nil || res.Kind != CredOAuth {
			t.Errorf("kind %q, error %v; want the session kept", res.Kind, err)
		}
		if n := warnings(f); n != 1 || countContaining(f.seenNotify, "the last line could not be written") != 1 {
			t.Errorf("notices %q; want one warning naming the Driver's error", f.seenNotify)
		}
	})
	t.Run("failed keeps the sign-in's error", func(t *testing.T) {
		stubBrowserLogin(t, func(_ context.Context, _ llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
			_ = opts.OpenURL("https://authorize.test")
			return nil, auth.ErrSignInDenied
		})
		f := newFake(t, fakePrompter{selects: []int{grokBrowserIdx}})
		_, err := resolveBounded0074(t, context.Background(), f, Options{TokenStore: newMemoryTokenStore(),
			NoEndpointPrompt: true, Driver: &recordingDriver0074{answer: failTerminal}})
		if !errors.Is(err, auth.ErrSignInDenied) || errors.Is(err, errStep) {
			t.Errorf("error %v; want the sign-in's own", err)
		}
		if n := warnings(f); n != 1 {
			t.Errorf("notices %q; want one warning", f.seenNotify)
		}
	})
}
