package wizard

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// recordingDriver0074 is a StepDriver that records each step and answers
// through answer, which may be nil.
type recordingDriver0074 struct {
	mu     sync.Mutex
	steps  []authcontract.Step
	answer func(ctx context.Context, step authcontract.Step) (authcontract.Answer, error)
}

func (d *recordingDriver0074) SignInStep(ctx context.Context, _ llmprovider.ProviderID, step authcontract.Step) (authcontract.Answer, error) {
	d.mu.Lock()
	d.steps = append(d.steps, step)
	answer := d.answer
	d.mu.Unlock()
	if answer == nil {
		return authcontract.Answer{}, nil
	}
	return answer(ctx, step)
}

func (d *recordingDriver0074) kinds() []authcontract.StepKind {
	d.mu.Lock()
	defer d.mu.Unlock()
	var kinds []authcontract.StepKind
	for _, s := range d.steps {
		kinds = append(kinds, s.Kind)
	}
	return kinds
}

// TestDriver_ReceivesSteps_0074 (0074-MADR D6): with a Driver, a device
// sign-in's show_code and waiting, and a browser sign-in's open_url and
// paste, reach it in order; the Prompter gets no sign-in notice and asks no
// paste; ShowDeviceCode and OpenURL are not called.
func TestDriver_ReceivesSteps_0074(t *testing.T) {
	t.Run("device", func(t *testing.T) {
		stubGrokDevice0057(t, grokCode0057)
		d := &recordingDriver0074{}
		shown := 0
		f := newFake(t, fakePrompter{selects: []int{grokDeviceIdx}})
		res, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), NoEndpointPrompt: true,
			Driver: d, ShowDeviceCode: func(DeviceCode) error { shown++; return nil }}, llmprovider.ProviderGrok)
		if err != nil || res.Kind != CredOAuth {
			t.Fatalf("kind %q, error %v; want the session", res.Kind, err)
		}
		if got, want := d.kinds(), []authcontract.StepKind{authcontract.StepShowCode, authcontract.StepWaiting, authcontract.StepDone}; !slices.Equal(got, want) {
			t.Errorf("the Driver was given %v; want %v", got, want)
		}
		if s := d.steps[0]; s.URL != grokCode0057.URL || s.Code != grokCode0057.Code || s.Seq != 1 {
			t.Errorf("show_code step %+v; want the code's URL and code, Seq 1", s)
		}
		if shown != 0 || countContaining(f.seenNotify, grokCode0057.Code) != 0 {
			t.Errorf("ShowDeviceCode called %d times, notices %q; want neither", shown, f.seenNotify)
		}
	})
	t.Run("browser", func(t *testing.T) {
		stubBrowserLogin(t, func(ctx context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
			if err := opts.OpenURL("https://authorize.test"); err != nil {
				t.Errorf("OpenURL() error = %v", err)
			}
			if _, err := opts.InputCode(ctx); err != nil {
				t.Errorf("InputCode() error = %v", err)
			}
			return testOAuthSession(provider), nil
		})
		d := &recordingDriver0074{answer: func(_ context.Context, s authcontract.Step) (authcontract.Answer, error) {
			return authcontract.Answer{Paste: "code-0074"}, nil
		}}
		opened := 0
		f := newFake(t, fakePrompter{selects: []int{grokBrowserIdx}})
		res, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), NoEndpointPrompt: true,
			Driver: d, OpenURL: func(string) error { opened++; return nil }}, llmprovider.ProviderGrok)
		if err != nil || res.Kind != CredOAuth {
			t.Fatalf("kind %q, error %v; want the session", res.Kind, err)
		}
		if got, want := d.kinds(), []authcontract.StepKind{authcontract.StepOpenURL, authcontract.StepPaste, authcontract.StepDone}; !slices.Equal(got, want) {
			t.Errorf("the Driver was given %v; want %v", got, want)
		}
		if opened != 0 || len(f.seenNotify) != 0 || len(f.seenInput) != 0 {
			t.Errorf("OpenURL called %d times, notices %q, inputs %q; want none", opened, f.seenNotify, f.seenInput)
		}
	})
}

// TestDriver_PasteAnswerCompletes_0074: the Driver's paste answer completes
// a browser sign-in: it is what the login's InputCode returns.
func TestDriver_PasteAnswerCompletes_0074(t *testing.T) {
	var got string
	stubBrowserLogin(t, func(ctx context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		if err := opts.OpenURL("https://authorize.test"); err != nil {
			t.Errorf("OpenURL() error = %v", err)
		}
		code, err := opts.InputCode(ctx)
		if err != nil {
			return nil, err
		}
		got = code
		return testOAuthSession(provider), nil
	})
	d := &recordingDriver0074{answer: func(context.Context, authcontract.Step) (authcontract.Answer, error) {
		return authcontract.Answer{Paste: "the-driver-code"}, nil
	}}
	f := newFake(t, fakePrompter{selects: []int{grokBrowserIdx}})
	if _, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), NoEndpointPrompt: true,
		Driver: d}, llmprovider.ProviderGrok); err != nil {
		t.Fatal(err)
	}
	if got != "the-driver-code" {
		t.Errorf("InputCode returned %q; want the Driver's answer", got)
	}
}

// TestDriver_ContextEndsPaste_0074: when the loopback wins, the Driver's
// pending SignInStep sees ctx end, and the sign-in returns without waiting
// for the Driver to give an answer.
func TestDriver_ContextEndsPaste_0074(t *testing.T) {
	started := make(chan struct{})
	seen := make(chan error, 1)
	stubBrowserLogin(t, func(ctx context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		if err := opts.OpenURL("https://authorize.test"); err != nil {
			t.Errorf("OpenURL() error = %v", err)
		}
		go func() { _, _ = opts.InputCode(context.Background()) }()
		<-started
		return testOAuthSession(provider), nil // the loopback won
	})
	d := &recordingDriver0074{answer: func(ctx context.Context, s authcontract.Step) (authcontract.Answer, error) {
		if s.Kind != authcontract.StepPaste {
			return authcontract.Answer{}, nil
		}
		close(started)
		<-ctx.Done()
		seen <- ctx.Err()
		return authcontract.Answer{}, ctx.Err()
	}}
	type outcome struct {
		kind CredentialKind
		err  error
	}
	done := make(chan outcome, 1)
	go func() {
		f := newFake(t, fakePrompter{selects: []int{grokBrowserIdx}})
		res, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), NoEndpointPrompt: true,
			Driver: d}, llmprovider.ProviderGrok)
		done <- outcome{res.Kind, err}
	}()
	select {
	case o := <-done:
		if o.err != nil || o.kind != CredOAuth {
			t.Errorf("kind %q, error %v; want the loopback's session", o.kind, o.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the sign-in did not return within 5 s of the loopback's win")
	}
	select {
	case err := <-seen:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("the Driver's ctx ended with %v; want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("the Driver's pending SignInStep never saw ctx end")
	}
}

// TestDriver_ErrorEnds_0074: an error from SignInStep on open_url ends the
// sign-in, returned as it is, whatever the login then reports.
func TestDriver_ErrorEnds_0074(t *testing.T) {
	errStep := errors.New("the event could not be written")
	stubBrowserLogin(t, func(ctx context.Context, _ llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		if err := opts.OpenURL("https://authorize.test"); err != nil {
			t.Errorf("OpenURL() error = %v", err)
		}
		<-ctx.Done() // the sign-in goes on until the wizard ends it
		return nil, ctx.Err()
	})
	d := &recordingDriver0074{answer: func(context.Context, authcontract.Step) (authcontract.Answer, error) {
		return authcontract.Answer{}, errStep
	}}
	store := newMemoryTokenStore()
	done := make(chan error, 1)
	go func() {
		f := newFake(t, fakePrompter{selects: []int{grokBrowserIdx}})
		_, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: store, NoEndpointPrompt: true,
			Driver: d}, llmprovider.ProviderGrok)
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, errStep) || err.Error() != errStep.Error() || store.saves != 0 {
			t.Errorf("error %v, %d saves; want the Driver's error itself and nothing saved", err, store.saves)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the sign-in did not end within 5 s of the Driver's error")
	}
}
