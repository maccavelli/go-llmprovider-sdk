package wizard

import (
	"context"
	"errors"
	"sync"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/internal/signinflow"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// StepDriver is given each step of a sign-in the wizard runs, by a caller
// with no terminal, such as one that prints each step as a line of JSON
// (Options.Driver, 0074-MADR D6). It speaks the contract's types only.
//
// SignInStep is called once per step, in order, on the sign-in's goroutine:
// open_url, show_code, waiting and paste, then the sign-in's last step, done
// or failed. For paste it returns the answer, whose Seq the wizard sets to
// the step's; for the others its Answer is ignored.
//
// An error from SignInStep on any step before the last ends the sign-in and
// is returned as it is; the Driver is then given no last step. On every other
// path it is given exactly one, last: done carries the method's Yields, and
// failed the error's kind, whether the login failed, the caller's ctx ended
// or the engine failed. An error from SignInStep on the last step changes
// nothing: the sign-in's result, or its own error, is returned, and the
// error is shown as one warning notice. The last step's ctx is not ended by
// the sign-in's end, which has happened.
//
// SignInStep must return soon after ctx ends, as InputerContext must
// (0057-MADR D3): a loopback that wins ends a pending paste by ending ctx.
// A sign-in calls it from one goroutine at a time; an implementation must be
// safe for concurrent use if its caller runs sign-ins at once.
type StepDriver interface {
	SignInStep(ctx context.Context, provider llmprovider.ProviderID, step authcontract.Step) (authcontract.Answer, error)
}

// stepRenderer renders the steps of one sign-in: through Options.Driver when
// it is set, else as the wizard always has, through the Prompter,
// Options.OpenURL and Options.ShowDeviceCode (0074-MADR D6).
type stepRenderer struct {
	p        Prompter
	o        Options
	provider llmprovider.ProviderID
	// open shows the authorize URL, with the paste instruction when a paste
	// is asked, and gives it to Options.OpenURL; its error is the caller's.
	open func(authorizeURL string) error
	// input reads the paste, the paste prompt of browserFlowOptionsFor.
	input func(ctx context.Context) (string, error)
	// drain finishes a paste read the loopback overtook.
	drain func()
	// verdict, for a device sign-in, carries the result of showing the code
	// to the login that waits to hear it, so that an error from it ends the
	// sign-in before any poll (0057-MADR D1).
	verdict chan error
}

// run starts spec and renders its steps until the sign-in ends. An error
// from Options.Driver, or from the paste read, ends the sign-in and is
// returned as it is, whatever the login then reports.
func (r *stepRenderer) run(ctx context.Context, spec signinflow.Spec) (signinflow.Result, error) {
	run, err := signinflow.Start(ctx, spec)
	if err != nil {
		return signinflow.Result{}, err
	}
	// stepCtx ends with the sign-in, so that a pending paste, the Driver's
	// or the Prompter's, sees it end when the loopback wins.
	stepCtx, stop := context.WithCancel(ctx)
	defer stop()
	type ended struct {
		result signinflow.Result
		err    error
	}
	end := make(chan ended, 1)
	go func() {
		// Not ctx itself: when ctx ends, the sign-in's own error, which
		// the login gives, is the one to return.
		result, err := run.Wait(context.WithoutCancel(ctx))
		stop()
		end <- ended{result, err}
	}()
	var (
		mu     sync.Mutex
		failed error
		wg     sync.WaitGroup
	)
	// fail ends the sign-in with err, the first one given, unless the
	// sign-in has ended already: an error then is ctx's, not a failure.
	fail := func(err error) {
		mu.Lock()
		if failed == nil && stepCtx.Err() == nil {
			failed = err
		}
		mu.Unlock()
		run.Cancel()
	}
	pastes := 0
	for step := range run.Steps() {
		switch step.Kind {
		case authcontract.StepOpenURL:
			if err := r.openURL(stepCtx, step); err != nil {
				fail(err)
			}
		case authcontract.StepShowCode:
			if step.URL == "" && step.Code == "" {
				continue // a login that showed no code
			}
			err := r.showCode(stepCtx, step)
			if r.verdict != nil {
				r.verdict <- err
			}
			if err != nil {
				fail(err)
			}
		case authcontract.StepPaste:
			pastes++
			r.paste(stepCtx, run, step, pastes > 1, fail, &wg)
		case authcontract.StepWaiting:
			if r.o.Driver != nil {
				if _, err := r.o.Driver.SignInStep(stepCtx, r.provider, step); err != nil {
					fail(err)
				}
			}
		case authcontract.StepDone, authcontract.StepFailed:
			mu.Lock()
			own := failed != nil // the Driver's own error ended it: no last step
			mu.Unlock()
			if r.o.Driver == nil || own {
				continue
			}
			// The sign-in has ended, so the Driver's error cannot end it:
			// it is a warning, and the result or error stands.
			if err := r.lastStep(ctx, step); err != nil {
				r.p.Notify(LevelWarn, "the sign-in driver returned an error on the %s step (%s); the sign-in's result stands",
					step.Kind, redact.StripControl(redact.String(err.Error())))
			}
		}
	}
	stop()
	if r.drain != nil {
		r.drain()
	}
	wg.Wait()
	finished := <-end
	mu.Lock()
	defer mu.Unlock()
	if failed != nil {
		return signinflow.Result{}, failed
	}
	return finished.result, finished.err
}

// lastStep hands the Driver the sign-in's last step, under a context the
// sign-in's end has not ended, and returns the Driver's error.
func (r *stepRenderer) lastStep(ctx context.Context, step authcontract.Step) error {
	_, err := r.o.Driver.SignInStep(context.WithoutCancel(ctx), r.provider, step)
	return err
}

// openURL renders an open_url step. Options.OpenURL's error is a warning:
// the URL is on screen already (0074-MADR D4); a Driver's error ends the
// sign-in.
func (r *stepRenderer) openURL(ctx context.Context, step authcontract.Step) error {
	if r.o.Driver != nil {
		_, err := r.o.Driver.SignInStep(ctx, r.provider, step)
		return err
	}
	if err := r.open(step.URL); err != nil {
		r.p.Notify(LevelWarn, "could not open the browser (%s); open the URL above",
			redact.StripControl(redact.String(err.Error())))
	}
	return nil
}

// showCode renders a show_code step.
func (r *stepRenderer) showCode(ctx context.Context, step authcontract.Step) error {
	if r.o.Driver != nil {
		_, err := r.o.Driver.SignInStep(ctx, r.provider, step)
		return err
	}
	return showDeviceCode(r.p, r.o)(DeviceCode{Provider: r.provider, URL: step.URL, Code: step.Code, Expiry: step.ExpiresAt})
}

// paste renders a paste step and sends its answer. The Driver is called
// here, where the sign-in's goroutine waits for it, and the Prompter is read
// on a goroutine of its own, so that the loopback's win is seen while a read
// is pending, as it always was. A step after the first is a re-ask, whose
// text says why the last paste was not used.
func (r *stepRenderer) paste(
	ctx context.Context,
	run *signinflow.Run,
	step authcontract.Step,
	reask bool,
	fail func(error),
	wg *sync.WaitGroup,
) {
	answer := func(text string) {
		err := run.Answer(ctx, authcontract.Answer{SchemaVersion: authcontract.SchemaVersion, Seq: step.Seq, Paste: text})
		// An answer the sign-in refuses came after its end, which the
		// loopback's win may have caused; the sign-in's own result is
		// returned then.
		if err != nil && !errors.Is(err, llmprovider.ErrInvalidRequest) {
			fail(err)
		}
	}
	if r.o.Driver != nil {
		a, err := r.o.Driver.SignInStep(ctx, r.provider, step)
		if err != nil {
			fail(err)
			return
		}
		answer(a.Paste)
		return
	}
	if reask {
		r.p.Notify(LevelWarn, "%s", step.Text)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		text, err := r.input(ctx)
		if err != nil {
			fail(err)
			return
		}
		answer(text)
	}()
}
