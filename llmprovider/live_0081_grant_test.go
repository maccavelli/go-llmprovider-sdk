//go:build live_gateways

package llmprovider_test

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestLive_EarlierGrant_0081 is 0081-PLAN Phase 1a, the owner's probe of
// whether a provider ends the earlier grant when a person signs in to it a
// second time (0081-MADR D12, N1). It REQUIRES LLMPROVIDER_LIVE_0081=1 and
// LLMPROVIDER_LIVE_0081_PROVIDER, one of chatgpt, plan, grok or kilo, and a
// person for each of its two sign-ins, within four minutes each. One run is
// one provider's: sign-in-1, baseline (a use of the first session),
// sign-in-2, first-after (the first session used again, the measurement),
// control (the second session used) and last-after (the first session used a
// third time, when first-after succeeded), then one verdict: kept, ended or
// indeterminate (earlierGrant.run, grantVerdict, whose table is the verdict's
// definition). After the verdict is printed, each grant the run obtained is
// revoked, where the provider has a revocation and the owner did not rule
// otherwise (chatgpt's two are left, grantRevokes), on a bound of its own, and
// one line per grant says what the provider answered, or by_hand; that never
// changes the verdict. The commands give the test -timeout 15m: the sign-ins and uses are
// bound at nine minutes, the revocations at twenty seconds each.
//
// Both sessions live in the test's memory: no store is opened, passed to a
// sign-in or written, and the test reads no environment variable but its two
// (the HTTP transport reads the proxy variables, and the browser opener
// inherits the environment). Each authorize URL, and Kilo's verification page,
// is handed to the system's browser opener and shown nowhere else; Kilo's user
// code is written to the terminal device (/dev/tty) and nowhere else. The
// test's output holds the step lines and the verdict only: statuses, an error's
// kind, an allow-listed OAuth code and a count of requests, never a token, a
// code, a URL, an id or an error's text (TestEarlierGrantReport_0081 proves it
// on planted values). Only a "kept" verdict passes (grantPasses).
//
// The body is grantRunTest, which TestEarlierGrantRunTest_0081 runs against
// stand-ins, and grantApply, which TestGrantApply_0081 runs on a stand-in
// testing.T; what is left here is what only a live run can do: the real
// sign-ins, the real opener and terminal, and the real environment.
//
// If a provider does end earlier grants, the probe's own sign-ins end the
// person's other sessions with that provider, in every app (0081-PLAN Phase
// 1b says so before the run).
func TestLive_EarlierGrant_0081(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), grantRunLimit)
	defer cancel()
	out := grantRunTest(ctx, os.Getenv, func(provider string) (*earlierGrant, error) {
		hostID := ""
		if provider == grantPlan {
			var err error
			if hostID, err = liveGrantHostID(); err != nil {
				return nil, err
			}
		}
		rec := &grantRecorder{next: http.DefaultTransport}
		g := &earlierGrant{
			provider:    provider,
			logf:        t.Logf,
			rec:         rec,
			io:          liveGrantIO(),
			needsCode:   provider == grantKilo,
			wait:        grantSignInWait,
			signIn:      liveGrantSignIn(provider, rec.client(), hostID),
			use:         grantUseRefresh,
			revoke:      auth.RevokeOAuthSession,
			revokeLimit: grantRevokeLimit,
		}
		if provider == grantKilo {
			g.use = grantUseKiloProfile(llmprovider.WithHTTPClient(rec.client()))
		}
		return g, nil
	})
	grantApply(t, out)
}

// liveGrantSignIn is provider's sign-in, the shipped one: the browser
// sign-ins through auth.LoginBrowserOAuth, the plan's through
// auth.LoginChatGPTPlan (the second with the first session as Previous, as
// the wizard does, and one host id for both, as one host would have), and
// Kilo's through the device flow. No flow is given an InputCode, so nothing
// is pasted, and none a store.
func liveGrantSignIn(provider string, client *http.Client, hostID string) func(context.Context, *grantShow, *auth.OAuthSession) (*auth.OAuthSession, error) {
	return func(ctx context.Context, show *grantShow, previous *auth.OAuthSession) (*auth.OAuthSession, error) {
		switch provider {
		case grantChatGPT:
			return auth.LoginBrowserOAuth(ctx, llmprovider.ProviderOpenAI, auth.OAuthFlowOptions{HTTPClient: client, OpenURL: show.URL})
		case grantGrok:
			return auth.LoginBrowserOAuth(ctx, llmprovider.ProviderGrok, auth.OAuthFlowOptions{HTTPClient: client, OpenURL: show.URL})
		case grantPlan:
			opts := auth.ChatGPTPlanOptions{AgentName: planAgentName, Previous: previous, HTTPClient: client, OpenURL: show.URL}
			if previous == nil {
				opts.HostID = hostID
			}
			return auth.LoginChatGPTPlan(ctx, opts)
		}
		login, err := auth.StartDeviceOAuth(ctx, llmprovider.ProviderKilo, auth.OAuthFlowOptions{HTTPClient: client})
		if err != nil {
			return nil, err
		}
		if err := show.URL(login.VerificationURI); err != nil {
			return nil, err
		}
		show.Code(login.UserCode)
		return login.Wait(ctx)
	}
}

// liveGrantIO is the real browser opener and terminal device. The opener is
// the system's: open on macOS, xdg-open on Linux, none elsewhere, where the
// run stops with its plain line. It is given a web URL only (grantWebURL),
// as a process argument: the URL is visible to this machine's process list
// for the moment the opener runs, and the browser keeps it in its history.
func liveGrantIO() grantIO {
	opener := ""
	switch runtime.GOOS {
	case "darwin":
		opener = "open"
	case "linux":
		opener = "xdg-open"
	}
	return grantIO{
		canOpen: func() bool {
			if opener == "" {
				return false
			}
			_, err := exec.LookPath(opener)
			return err == nil
		},
		open: func(raw string) error {
			if !grantWebURL(raw) {
				return errors.New("not a web URL")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			return exec.CommandContext(ctx, opener, raw).Run()
		},
		tty: func() (io.WriteCloser, error) { return os.OpenFile("/dev/tty", os.O_WRONLY, 0) },
	}
}

// liveGrantHostID is a random urn:uuid (version 4) for the plan's two
// sign-ins to share.
func liveGrantHostID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("urn:uuid:%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
