package wizard

import (
	"context"
	"time"

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
