package wizard

import (
	"context"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/internal/signinflow"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// DeviceCode is a device sign-in's code, given to Options.ShowDeviceCode for
// a caller that shows it itself, such as one reporting it as data
// (0057-MADR D1). It is a value, safe for concurrent use while no goroutine
// changes it.
type DeviceCode struct {
	// Provider is the provider being signed in to.
	Provider llmprovider.ProviderID
	// URL is where the user enters the code; it may already carry it.
	URL string
	// Code is the code the user enters.
	Code string
	// Expiry is when the code stops being accepted. It is never zero: a
	// provider that gives none has a default, and OpenAI's is this module's
	// own wait.
	Expiry time.Time
}

// deviceLogin is the device sign-in the wizard runs, replaced by tests.
var deviceLogin = runDeviceLogin

// runDeviceLogin starts a device sign-in, gives its code to show, and waits
// for approval. A DeviceLogin polls only in Wait, so an error from show ends
// the sign-in before any poll, and is returned as it is (0057-MADR D1 and its
// amendment).
func runDeviceLogin(
	ctx context.Context,
	provider llmprovider.ProviderID,
	flow auth.OAuthFlowOptions,
	show func(DeviceCode) error,
) (*auth.OAuthSession, error) {
	login, err := auth.StartDeviceOAuth(ctx, provider, flow)
	if err != nil {
		return nil, err
	}
	code := DeviceCode{Provider: provider, URL: login.VerificationURI, Code: login.UserCode, Expiry: login.Expiry}
	if err := show(code); err != nil {
		return nil, err
	}
	return login.Wait(ctx)
}

// deviceSignIn runs a device sign-in on the step engine, given the wizard's
// own seam (deviceLogin), and renders its steps: through Options.Driver, or
// through showDeviceCode. An error from either ends the sign-in before any
// poll and is returned as it is (0057-MADR D1, 0074-MADR D6).
func deviceSignIn(ctx context.Context, p Prompter, provider llmprovider.ProviderID, o Options) (*auth.OAuthSession, error) {
	verdict := make(chan error, 1)
	yields := authcontract.YieldsOAuthSession
	if deviceKey(provider, llmprovider.AuthDeviceCode, o) {
		yields = authcontract.YieldsDeviceKey
	}
	r := &stepRenderer{p: p, o: o, provider: provider, verdict: verdict}
	result, err := r.run(ctx, signinflow.Spec{
		Provider: provider,
		Method:   authcontract.Method{ID: string(llmprovider.AuthDeviceCode), Yields: yields},
		Mode:     signinflow.ModeDevice,
		Flow:     oauthFlowOptions(o),
		Logins:   signinflow.Logins{Device: deviceFlowFor(verdict)},
	})
	if err != nil {
		return nil, err
	}
	return result.Session, nil
}

// deviceFlowFor is deviceLogin as the engine's device login. deviceLogin
// shows the code and waits in one call, so it runs on a goroutine of its
// own: the code it shows is the flow's, and its show waits for verdict, the
// renderer's result of showing the code, before the wait for approval
// starts. A login that shows no code, as a test's may, gives a flow of no
// code, which the renderer does not show.
func deviceFlowFor(verdict <-chan error) func(context.Context, llmprovider.ProviderID, auth.OAuthFlowOptions) (signinflow.DeviceFlow, error) {
	type outcome struct {
		session *auth.OAuthSession
		err     error
	}
	return func(ctx context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (signinflow.DeviceFlow, error) {
		codes := make(chan DeviceCode, 1)
		done := make(chan outcome, 1)
		go func() {
			session, err := deviceLogin(ctx, provider, opts, func(c DeviceCode) error {
				codes <- c
				select {
				case err := <-verdict:
					return err
				case <-ctx.Done():
					return ctx.Err()
				}
			})
			done <- outcome{session, err}
		}()
		select {
		case c := <-codes:
			return signinflow.DeviceFlow{URL: c.URL, Code: c.Code, Expiry: c.Expiry,
				Wait: func(ctx context.Context) (*auth.OAuthSession, error) {
					select {
					case o := <-done:
						return o.session, o.err
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}}, nil
		case o := <-done:
			if o.err != nil {
				return signinflow.DeviceFlow{}, o.err
			}
			return signinflow.DeviceFlow{Wait: func(context.Context) (*auth.OAuthSession, error) { return o.session, nil }}, nil
		}
	}
}

// showDeviceCode is how a device sign-in's code is shown: through
// Options.ShowDeviceCode when set, else as a notice through p.
func showDeviceCode(p Prompter, o Options) func(DeviceCode) error {
	if o.ShowDeviceCode != nil {
		return o.ShowDeviceCode
	}
	return func(c DeviceCode) error {
		p.Notify(LevelInfo, "Open %s and enter code %s", c.URL, c.Code)
		return nil
	}
}
