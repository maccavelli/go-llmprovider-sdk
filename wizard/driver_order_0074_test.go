package wizard

import (
	"context"
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// showErrorSignIns0074 is how many device sign-ins
// TestDriver_ShowErrorRecordedFirst_0074 runs. The order it checks is lost on
// a race, which one sign-in in a few hundred loses when the order is wrong.
const showErrorSignIns0074 = 1000

// TestDriver_ShowErrorRecordedFirst_0074 (0074-MADR D6, "No final step after
// its own error"; the PLAN's "Deviation (2026-11-13): two flaky tests
// root-caused"): an error the Driver returns on show_code is recorded, and
// the sign-in cancelled, before the device login hears it, so that the
// login's own end cannot end the sign-in first and hand the Driver a failed
// step. The login reports whether its ctx was still live when its show
// returned.
func TestDriver_ShowErrorRecordedFirst_0074(t *testing.T) {
	errStep := errors.New("the event could not be written")
	live := make(chan bool, 1)
	original := deviceLogin
	t.Cleanup(func() { deviceLogin = original })
	deviceLogin = func(ctx context.Context, provider llmprovider.ProviderID, _ auth.OAuthFlowOptions, show func(DeviceCode) error) (*auth.OAuthSession, error) {
		err := show(grokCode0057)
		live <- ctx.Err() == nil
		if err != nil {
			return nil, err
		}
		return testOAuthSession(provider), nil
	}
	for i := range showErrorSignIns0074 {
		d := &recordingDriver0074{answer: func(_ context.Context, s authcontract.Step) (authcontract.Answer, error) {
			if s.Kind == authcontract.StepShowCode {
				return authcontract.Answer{}, errStep
			}
			return authcontract.Answer{}, nil
		}}
		f := newFake(t, fakePrompter{selects: []int{grokDeviceIdx}})
		_, err := resolveBounded0074(t, context.Background(), f, Options{TokenStore: newMemoryTokenStore(),
			NoEndpointPrompt: true, Driver: d})
		if !errors.Is(err, errStep) {
			t.Fatalf("sign-in %d: error %v; want the Driver's", i, err)
		}
		if <-live {
			t.Fatalf("sign-in %d: the login heard the Driver's error while the sign-in was still live", i)
		}
		if terminal, _ := d.terminals(); len(terminal) != 0 {
			t.Fatalf("sign-in %d: the Driver was handed %+v after its own error; want no terminal step", i, terminal)
		}
	}
}
