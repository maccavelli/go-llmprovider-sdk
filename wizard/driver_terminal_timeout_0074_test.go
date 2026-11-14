package wizard

import (
	"context"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
)

// TestDriver_BlockedTerminalStepTimesOut_0074 (the owner's ruling "Fixed 5 s
// limit", 2026-11-14): a Driver that blocks on the terminal step until its
// ctx ends is cut off after 5 s. The wizard returns, with the credential
// kept and exactly one warning, and not before the limit. The wait is real
// time, bounded at 7 s, so a limit that is gone fails here and does not hang.
func TestDriver_BlockedTerminalStepTimesOut_0074(t *testing.T) {
	stubGrokDevice0057(t, grokCode0057)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	d := &recordingDriver0074{answer: func(ctx context.Context, s authcontract.Step) (authcontract.Answer, error) {
		if s.Kind == authcontract.StepDone {
			select {
			case <-ctx.Done():
				return authcontract.Answer{}, ctx.Err()
			case <-release:
			}
		}
		return authcontract.Answer{}, nil
	}}
	f := newFake(t, fakePrompter{selects: []int{grokDeviceIdx}})
	type outcome struct {
		res Result
		err error
	}
	done := make(chan outcome, 1)
	start := time.Now()
	go func() {
		res, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: newMemoryTokenStore(),
			NoEndpointPrompt: true, Driver: d}, "grok")
		done <- outcome{res, err}
	}()
	var got outcome
	select {
	case got = <-done:
	case <-time.After(7 * time.Second):
		t.Fatal("the wizard did not return within 7 s of a Driver that blocks on the terminal step")
	}
	if elapsed := time.Since(start); elapsed < 4*time.Second {
		t.Errorf("returned after %v; want the Driver held for its 5 s", elapsed)
	}
	if got.err != nil || got.res.Kind != CredOAuth {
		t.Errorf("kind %q, error %v; want the credential kept", got.res.Kind, got.err)
	}
	if n := countContaining(f.seenNotify, "returned an error on the done step"); n != 1 {
		t.Errorf("notices %q; want exactly one warning", f.seenNotify)
	}
}
