package signin

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// deviceCase is one device sign-in against the fake issuer.
type deviceCase struct {
	provider llmprovider.ProviderID
	issuer   string
	url      string
	code     string
	yields   authcontract.Yields
}

var deviceCases = []deviceCase{
	{llmprovider.ProviderOpenAI, fakeBase, fakeBase + "/codex/device", "OPEN-1234", authcontract.YieldsOAuthSession},
	{llmprovider.ProviderGrok, fakeBase, fakeBase + "/device", "GROK-1234", authcontract.YieldsOAuthSession},
	{llmprovider.ProviderKilo, fakeKilo, fakeKilo + "/device", "KILO-1", authcontract.YieldsDeviceKey},
}

func (c deviceCase) options(fake *fakeIssuer) Options {
	return Options{HTTPClient: fake.client(), ClientID: "test-client", Issuer: c.issuer}
}

// assertDeviceRun fails t unless the run's steps are show_code with the
// URL, code and expiry, waiting, and done with Yields, numbered 1 to 3, the
// channel is closed, and the credential is of the right type.
func assertDeviceRun(t *testing.T, c deviceCase, run *SignIn) {
	t.Helper()
	credential, err := run.Wait(background())
	if err != nil {
		t.Fatalf("%s: Wait = %v", c.provider, err)
	}
	steps := collect(t, run)
	want := []authcontract.StepKind{authcontract.StepShowCode, authcontract.StepWaiting, authcontract.StepDone}
	if got := kinds(steps); !slices.Equal(got, want) {
		t.Fatalf("%s: steps = %v; want %v", c.provider, got, want)
	}
	assertSeq(t, steps)
	if show := steps[0]; show.URL != c.url || show.Code != c.code || show.ExpiresAt.IsZero() {
		t.Errorf("%s: show_code = URL %q, code %q, expiry %v; want %q, %q and an expiry", c.provider, show.URL, show.Code, show.ExpiresAt, c.url, c.code)
	}
	if done := steps[2]; done.Yields != c.yields || done.Error != nil {
		t.Errorf("%s: done = yields %q, error %v; want %q", c.provider, done.Yields, done.Error, c.yields)
	}
	if credential.Yields() != c.yields {
		t.Errorf("%s: credential yields %q; want %q", c.provider, credential.Yields(), c.yields)
	}
	switch got := credential.(type) {
	case Session:
		if got.OAuth == nil || got.OAuth.Access == "" {
			t.Errorf("%s: the session holds no access token", c.provider)
		}
	case DeviceKey:
		if got.Key == "" {
			t.Errorf("%s: the device key is empty", c.provider)
		}
	default:
		t.Errorf("%s: credential is a %T", c.provider, credential)
	}
}

// TestStart_DeviceSteps_0074: for OpenAI, Grok and Kilo against a fake
// issuer, show_code with the URL, code and expiry, waiting, and done with
// Yields, Seq 1, 2 and 3, the channel closed (0074-MADR D3).
func TestStart_DeviceSteps_0074(t *testing.T) {
	for _, c := range deviceCases {
		t.Run(string(c.provider), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				run, err := Start(background(), c.provider, llmprovider.AuthDeviceCode, c.options(&fakeIssuer{}))
				if err != nil {
					t.Fatalf("Start = %v", err)
				}
				if got := run.Method().ID; got != string(llmprovider.AuthDeviceCode) {
					t.Errorf("Method().ID = %q; want device_code", got)
				}
				assertDeviceRun(t, c, run)
			})
		})
	}
}

// TestStart_NoLoopbackDevice_0074: NoLoopback has no effect on a device
// sign-in: Start gives no error and the same steps, which complete by the
// poll, and auth.StartDeviceOAuth called directly with the option set starts
// and completes against the same fake (0074-MADR D4).
func TestStart_NoLoopbackDevice_0074(t *testing.T) {
	for _, c := range deviceCases {
		t.Run(string(c.provider)+"/Start", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				opts := c.options(&fakeIssuer{})
				opts.NoLoopback = true
				run, err := Start(background(), c.provider, llmprovider.AuthDeviceCode, opts)
				if err != nil {
					t.Fatalf("Start with NoLoopback = %v; want no error for a device sign-in", err)
				}
				assertDeviceRun(t, c, run)
			})
		})
		t.Run(string(c.provider)+"/StartDeviceOAuth", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				login, err := auth.StartDeviceOAuth(background(), c.provider, auth.OAuthFlowOptions{
					HTTPClient: (&fakeIssuer{}).client(), ClientID: "test-client", Issuer: c.issuer, NoLoopback: true})
				if err != nil {
					t.Fatalf("StartDeviceOAuth with NoLoopback = %v; want it to start", err)
				}
				session, err := login.Wait(background())
				if err != nil || session == nil || session.Access == "" {
					t.Fatalf("Wait = %v, %v; want a session", session, err)
				}
			})
		})
	}
}

// TestStart_Cancel_0074: Cancel during waiting ends with failed canceled,
// Wait's error matches context.Canceled, and no goroutine is left.
func TestStart_Cancel_0074(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fake := &fakeIssuer{kiloPoll: []int{202}}
		before := runtime.NumGoroutine()
		run, err := Start(background(), llmprovider.ProviderKilo, llmprovider.AuthDeviceCode,
			Options{HTTPClient: fake.client(), Issuer: fakeKilo})
		if err != nil {
			t.Fatalf("Start = %v", err)
		}
		for _, kind := range []authcontract.StepKind{authcontract.StepShowCode, authcontract.StepWaiting} {
			if step := <-run.Steps(); step.Kind != kind {
				t.Fatalf("step = %s; want %s", step.Kind, kind)
			}
		}
		canceledAt := time.Now()
		run.Cancel()
		credential, err := run.Wait(background())
		if !errors.Is(err, context.Canceled) || credential != nil {
			t.Fatalf("Wait = %v, %v; want an error matching context.Canceled", credential, err)
		}
		// The bubble's clock moves only while every goroutine sleeps: a poll
		// that Cancel did not stop would sleep to the code's expiry first.
		if waited := time.Since(canceledAt); waited > time.Second {
			t.Errorf("the poll ran %v after Cancel; want it stopped at once", waited)
		}
		steps := collect(t, run)
		if len(steps) != 1 || steps[0].Kind != authcontract.StepFailed || steps[0].Error == nil ||
			steps[0].Error.Kind != authcontract.FailureCanceled || steps[0].Seq != 3 {
			t.Errorf("steps after Cancel = %+v; want one failed #3 with kind canceled", steps)
		}
		run.Cancel()
		synctest.Wait()
		if after := runtime.NumGoroutine(); after != before {
			t.Errorf("%d goroutines after the sign-in, %d before", after, before)
		}
	})
}

// TestErrorKindOf_0074: every row of 0073-MADR D3, by its sentinel, in order
// (0074-MADR D8).
func TestErrorKindOf_0074(t *testing.T) {
	for _, c := range []struct {
		name string
		err  error
		want authcontract.ErrorKind
	}{
		{"canceled", context.Canceled, authcontract.FailureCanceled},
		{"a wrapped cancel", fmt.Errorf("sign-in: %w", context.Canceled), authcontract.FailureCanceled},
		{"timeout", context.DeadlineExceeded, authcontract.FailureTimeout},
		{"denied is not auth", auth.ErrSignInDenied, authcontract.FailureDenied},
		{"expired is not auth", auth.ErrSignInExpired, authcontract.FailureExpired},
		{"auth", llmprovider.ErrAuthFailure, authcontract.FailureAuth},
		{"quota is not rate_limited", llmprovider.ErrQuotaExhausted, authcontract.FailureQuota},
		{"rate_limited", llmprovider.ErrRateLimited, authcontract.FailureRateLimited},
		{"not_permitted", llmprovider.ErrNotPermitted, authcontract.FailureNotPermitted},
		{"workspace required", llmprovider.ErrWorkspaceRequired, authcontract.FailureWorkspace},
		{"workspace not found", llmprovider.ErrWorkspaceNotFound, authcontract.FailureWorkspace},
		{"context_overflow is not invalid_request", llmprovider.ErrContextOverflow, authcontract.FailureContextOverflow},
		{"empty_answer is not incomplete", llmprovider.ErrEmptyAnswer, authcontract.FailureEmptyAnswer},
		{"incomplete is not invalid_request", llmprovider.ErrIncomplete, authcontract.FailureIncomplete},
		{"invalid_request", llmprovider.ErrInvalidRequest, authcontract.FailureInvalidRequest},
		{"invalid_provider", llmprovider.ErrInvalidProvider, authcontract.FailureInvalidProvider},
		{"offline is not unavailable", llmprovider.ErrOffline, authcontract.FailureOffline},
		{"unavailable", llmprovider.ErrProviderUnavailable, authcontract.FailureUnavailable},
		{"unsupported", llmprovider.ErrUnsupported, authcontract.FailureUnsupported},
		{"the standard unsupported", errors.ErrUnsupported, authcontract.FailureUnsupported},
		{"other", errors.New("no kind"), authcontract.FailureOther},
		{"nothing", nil, ""},
	} {
		if got := ErrorKindOf(c.err); got != c.want {
			t.Errorf("%s: ErrorKindOf(%v) = %q; want %q", c.name, c.err, got, c.want)
		}
	}
}

// TestErrorKindOf_DeviceFailures_0074: the failures of a device sign-in, by
// the real auth code against the fake issuer, carry their kinds in Wait's
// error and in the failed step: Grok's denial and expiry, Kilo's, and the
// OpenAI poll's own 15-minute deadline, reached on the bubble's clock
// (0074-MADR D8; 0076-MADR D2).
func TestErrorKindOf_DeviceFailures_0074(t *testing.T) {
	for _, c := range []struct {
		name     string
		provider llmprovider.ProviderID
		issuer   string
		fake     *fakeIssuer
		want     authcontract.ErrorKind
	}{
		{"OpenAI deadline", llmprovider.ProviderOpenAI, fakeBase, &fakeIssuer{openAIPoll: []int{404}}, authcontract.FailureExpired},
		{"Grok expired", llmprovider.ProviderGrok, fakeBase, &fakeIssuer{grokPoll: []string{"expired_token"}}, authcontract.FailureExpired},
		{"Grok denied", llmprovider.ProviderGrok, fakeBase, &fakeIssuer{grokPoll: []string{"access_denied"}}, authcontract.FailureDenied},
		{"Kilo expired", llmprovider.ProviderKilo, fakeKilo, &fakeIssuer{kiloPoll: []int{410}}, authcontract.FailureExpired},
		{"Kilo denied", llmprovider.ProviderKilo, fakeKilo, &fakeIssuer{kiloPoll: []int{403}}, authcontract.FailureDenied},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				fake := c.fake
				run, err := Start(background(), c.provider, llmprovider.AuthDeviceCode,
					Options{HTTPClient: fake.client(), ClientID: "test-client", Issuer: c.issuer})
				if err != nil {
					t.Fatalf("Start = %v", err)
				}
				_, err = run.Wait(background())
				if got := ErrorKindOf(err); got != c.want {
					t.Errorf("ErrorKindOf(%v) = %q; want %q", err, got, c.want)
				}
				steps := collect(t, run)
				last := steps[len(steps)-1]
				if last.Kind != authcontract.StepFailed || last.Error == nil || last.Error.Kind != c.want {
					t.Errorf("last step = %+v; want failed with kind %q", last, c.want)
				}
				if last.Error != nil && strings.TrimSpace(last.Error.Message) == "" {
					t.Error("the failure has no message")
				}
			})
		})
	}
}
