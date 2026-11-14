//go:build live_gateways

package llmprovider_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/x/signin"
)

// pasteProbeAgentName is the name the plan probe registers its client under,
// which is no real product's.
const pasteProbeAgentName = probeIdentityName

// requirePasteFile skips the test unless LLMPROVIDER_LIVE_0074 and
// LLMPROVIDER_LIVE_0074_PASTE_FILE are set, and fails it when that path
// exists: the paste reaches the test through a file that is not there yet
// (pasteFromFile), because go test gives the binary no standard input.
func requirePasteFile(t *testing.T) string {
	t.Helper()
	if os.Getenv("LLMPROVIDER_LIVE_0074") != "1" {
		t.Skip("LLMPROVIDER_LIVE_0074 unset: this signs in at OpenAI with a person")
	}
	path := os.Getenv("LLMPROVIDER_LIVE_0074_PASTE_FILE")
	if path == "" {
		t.Skip("LLMPROVIDER_LIVE_0074_PASTE_FILE unset: the paste reaches the test through this file")
	}
	if _, err := os.Lstat(path); err == nil {
		t.Fatalf("LLMPROVIDER_LIVE_0074_PASTE_FILE %s exists; it must not exist when the test starts", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("LLMPROVIDER_LIVE_0074_PASTE_FILE %s: %v", path, err)
	}
	return path
}

// revokeUnlessDone revokes session when the test ends unless the returned
// function was called first, which says the test revoked it itself.
func revokeUnlessDone(t *testing.T, session *auth.OAuthSession) (done func()) {
	t.Helper()
	revoked := false
	t.Cleanup(func() {
		if revoked {
			return
		}
		if err := auth.RevokeOAuthSession(context.Background(), session); err != nil {
			t.Errorf("RevokeOAuthSession when the test ends: %v", err)
		}
	})
	return func() { revoked = true }
}

// sessionOf is the OAuth session a finished sign-in yields, or fails the test.
func sessionOf(ctx context.Context, t *testing.T, run *signin.SignIn) *auth.OAuthSession {
	t.Helper()
	cred, err := run.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	s, ok := cred.(signin.Session)
	if !ok || s.OAuth == nil {
		t.Fatalf("the sign-in yielded %T; want a session", cred)
	}
	return s.OAuth
}

// pasteSignIn drives run with a person's paste from path, logging each URL
// and confirm text it shows, and returns what signInFinding needs: the paste
// record, the steps seen and the sign-in's error. The authorize URL is checked
// against want, what that method's authorize URL really carries, before
// anyone signs in (0074-PLAN Phase 7, deviation 2026-11-14).
func pasteSignIn(ctx context.Context, t *testing.T, run *signin.SignIn, path string, want authorizeWant) (*pasteRecord, []authcontract.StepKind, error) {
	t.Helper()
	rec := &pasteRecord{}
	kinds, err := driveSteps(ctx, run, stepHooks{
		confirm: func(step authcontract.Step) bool {
			t.Logf("consent %s, answered yes by the owner's run: %s", step.ConfirmID, step.Text)
			return true
		},
		openURL: func(step authcontract.Step) {
			if err := authorizeError(step.URL, want); err != nil {
				t.Errorf("authorize URL: %v", err)
				run.Cancel()
				return
			}
			if port, perr := redirectPort(step.URL); perr == nil {
				t.Logf("FINDING redirect: the authorize URL's redirect_uri names port %d, where nothing listens", port)
			}
			t.Logf("open this URL on another device and sign in: %s", step.URL)
			t.Logf("then write what the page shows into %s (the run sheet's pbpaste line)", path)
		},
		paste: pasteFromFile(path, rec),
	})
	if err == nil {
		_, err = run.Wait(ctx)
	}
	return rec, kinds, err
}

// TestLive_0074PasteBrowser is 0074-PLAN Phase 7 step 2 (the report's U2):
// OpenAI's browser sign-in completed by a paste from another device, through
// x/signin with NoLoopback, so that no listener is bound. It REQUIRES
// LLMPROVIDER_LIVE_0074=1, LLMPROVIDER_LIVE_BROWSER_LOGIN=1 and
// LLMPROVIDER_LIVE_0074_PASTE_FILE, a path that must not exist yet, and a
// person. It checks the session refreshes once into a scratch store, then
// revokes it. No generation.
func TestLive_0074PasteBrowser(t *testing.T) {
	path := requirePasteFile(t)
	if os.Getenv("LLMPROVIDER_LIVE_BROWSER_LOGIN") != "1" {
		t.Skip("LLMPROVIDER_LIVE_BROWSER_LOGIN unset: this needs a person to sign in in a browser")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 11*time.Minute)
	defer cancel()
	run, err := signin.Start(ctx, llmprovider.ProviderOpenAI, llmprovider.AuthBrowserOAuth, signin.Options{NoLoopback: true})
	if err != nil {
		t.Fatalf("signin.Start: %v", err)
	}
	defer run.Cancel()
	rec, kinds, err := pasteSignIn(ctx, t, run, path, browserAuthorizeWant("originator", identityValue))
	line, passed := signInFinding("OpenAI browser", rec, kinds, err)
	t.Log(line)
	if !passed {
		t.FailNow()
	}
	session := sessionOf(ctx, t, run)
	done := revokeUnlessDone(t, session)
	checkRefreshOnce(ctx, t, session)
	checkIdentityRevoked(ctx, t, session)
	done()
}

// TestLive_0074PastePlan is 0074-PLAN Phase 7 step 3 (the report's U3): the
// plan sign-in completed by a pasted callback URL from another device, and
// then the port fallback. It REQUIRES LLMPROVIDER_LIVE_0074=1,
// LLMPROVIDER_LIVE_CHATGPT_PLAN=1 and LLMPROVIDER_LIVE_0074_PASTE_FILE, and a
// person. Part one is x/signin's chatgpt_plan with NoLoopback; it registers
// one client under the owner's ChatGPT account. Part two signs in again to
// that client (Previous, so no second registration) with the test holding
// port 1455, so that the loopback falls back to 1457, and the owner signs in
// in a browser on this host. Both sessions are revoked. No generation.
func TestLive_0074PastePlan(t *testing.T) {
	path := requirePasteFile(t)
	if os.Getenv("LLMPROVIDER_LIVE_CHATGPT_PLAN") != "1" {
		t.Skip("LLMPROVIDER_LIVE_CHATGPT_PLAN unset: the plan sign-in registers a client under the owner's account")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 22*time.Minute)
	defer cancel()

	run, err := signin.Start(ctx, llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan,
		signin.Options{AgentName: pasteProbeAgentName, NoLoopback: true})
	if err != nil {
		t.Fatalf("signin.Start: %v", err)
	}
	defer run.Cancel()
	rec, kinds, err := pasteSignIn(ctx, t, run, path, planFirstAuthorizeWant(pasteProbeAgentName))
	line, passed := signInFinding("OpenAI plan", rec, kinds, err)
	t.Log(line)
	if !passed {
		t.FailNow()
	}
	first := sessionOf(ctx, t, run)
	firstDone := revokeUnlessDone(t, first)
	checkPlanKind(t, first)
	_, store := planStoreRoundTrip(ctx, t, first)
	first.Store = store
	planRefresh(ctx, t, first, store)

	holdLoopbackPort(t, 1455)
	again, err := signin.Start(ctx, llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan,
		signin.Options{AgentName: pasteProbeAgentName, Previous: first, NoPaste: true})
	if err != nil {
		t.Fatalf("signin.Start again: %v", err)
	}
	defer again.Cancel()
	var wrongPort bool
	kinds, err = driveSteps(ctx, again, stepHooks{
		confirm: func(step authcontract.Step) bool {
			t.Logf("consent %s, answered yes by the owner's run: %s", step.ConfirmID, step.Text)
			return true
		},
		openURL: func(step authcontract.Step) {
			if err := authorizeError(step.URL, planAgainAuthorizeWant(first)); err != nil {
				t.Errorf("authorize URL: %v", err)
				wrongPort = true
				again.Cancel()
				return
			}
			port, perr := redirectPort(step.URL)
			if perr != nil || port != 1457 {
				t.Errorf("redirect_uri port = %d, %v; want 1457, with 1455 held by this test", port, perr)
				wrongPort = true
				again.Cancel()
				return
			}
			t.Logf("FINDING port: with 1455 held, the authorize URL redirects to port %d", port)
			t.Logf("open this URL in a browser on this host and sign in: %s", step.URL)
		},
	})
	if err == nil {
		_, err = again.Wait(ctx)
	}
	line, passed = loopbackFinding("OpenAI plan again", 1457, kinds, err)
	t.Log(line)
	if !passed || wrongPort {
		t.FailNow()
	}
	second := sessionOf(ctx, t, again)
	secondDone := revokeUnlessDone(t, second)
	checkPlanKind(t, second)
	if err := sameClient(first, second); err != nil {
		t.Errorf("FINDING reuse: %v", err)
	} else {
		t.Log("FINDING reuse: the sign-in again is under the first sign-in's client and host id")
	}
	planRevoke(ctx, t, first)
	firstDone()
	planRevoke(ctx, t, second)
	secondDone()
}
