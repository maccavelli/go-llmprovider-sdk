//go:build live_gateways

package llmprovider_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// identityValue is what the sign-ins send as the ChatGPT originator and the
// Grok referrer (0002-MADR §5).
const identityValue = "go-llmprovider-sdk"

// openAuthorizeURL is an OpenURL that checks the authorize URL names this
// module in param before printing it for a person to open. A wrong or
// missing value fails the test and ends the login through cancel, before
// anyone signs in (0002-PLAN Phase 8 step 0): LoginBrowserOAuth ignores an
// error OpenURL returns, so the error alone would leave it waiting.
func openAuthorizeURL(t *testing.T, param string, cancel context.CancelFunc) func(string) error {
	return func(raw string) error {
		u, err := url.Parse(raw)
		if err != nil {
			t.Errorf("authorize URL: %v", err)
			cancel()
			return err
		}
		if got := u.Query()[param]; len(got) != 1 || got[0] != identityValue {
			t.Errorf("authorize URL %s = %q, want [%q]", param, got, identityValue)
			cancel()
			return fmt.Errorf("authorize URL %s = %q", param, got)
		}
		t.Logf("open this URL and sign in: %s", raw)
		return nil
	}
}

// revokeAfter revokes session when the test ends, so no session of this
// module outlives it.
func revokeAfter(t *testing.T, session *auth.OAuthSession) {
	t.Cleanup(func() {
		if err := auth.RevokeOAuthSession(context.Background(), session); err != nil {
			t.Errorf("RevokeOAuthSession: %v", err)
		}
	})
}

// TestLive_ChatGPTBrowserLogin is the owner-run gate for the 127.0.0.1
// redirect (MADR 0012 §5.3) and the originator (0002-MADR §5). It REQUIRES
// LLMPROVIDER_LIVE_BROWSER_LOGIN=1 and a person at a browser: it checks and
// prints the authorize URL, waits up to five minutes for the ChatGPT login to
// redirect back, generates once with the new session, and then revokes it.
func TestLive_ChatGPTBrowserLogin(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_BROWSER_LOGIN") != "1" {
		t.Skip("LLMPROVIDER_LIVE_BROWSER_LOGIN unset: this needs a person to sign in in a browser")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	session, err := auth.LoginBrowserOAuth(ctx, llmprovider.ProviderOpenAI, auth.OAuthFlowOptions{
		OpenURL: openAuthorizeURL(t, "originator", cancel),
	})
	if err != nil {
		t.Fatalf("LoginBrowserOAuth: %v", err)
	}
	revokeAfter(t, session)
	checkWord(ctx, t, liveOpenAI(t, session, "gpt-6-astra"), "ALPHA", noSkip)
}

// TestLive_GrokBrowserLogin is the owner-run gate for Grok's loopback login
// and its referrer (0002-MADR §5; 0002-PLAN Phase 8). It REQUIRES
// LLMPROVIDER_LIVE_BROWSER_LOGIN=1 and a person at a browser, as the ChatGPT
// one does.
func TestLive_GrokBrowserLogin(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_BROWSER_LOGIN") != "1" {
		t.Skip("LLMPROVIDER_LIVE_BROWSER_LOGIN unset: this needs a person to sign in in a browser")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	session, err := auth.LoginBrowserOAuth(ctx, llmprovider.ProviderGrok, auth.OAuthFlowOptions{
		OpenURL: openAuthorizeURL(t, "referrer", cancel),
	})
	if err != nil {
		t.Fatalf("LoginBrowserOAuth: %v", err)
	}
	revokeAfter(t, session)
	checkWord(ctx, t, liveGrok(t, session, "grok-4.6"), "ALPHA", noSkip)
}

// TestLive_GrokDeviceLogin is the owner-run gate for Grok's device-code login
// (0002-PLAN Phase 8). It REQUIRES LLMPROVIDER_LIVE_DEVICE_LOGIN=1 and a
// person to approve the code it logs, within five minutes. The referrer is a
// form field here; the service accepting the login is the live check, and
// TestGrokDeviceRequest_CarriesTheReferrer pins the form offline.
func TestLive_GrokDeviceLogin(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_DEVICE_LOGIN") != "1" {
		t.Skip("LLMPROVIDER_LIVE_DEVICE_LOGIN unset: this needs a person to approve a device code")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	session, err := auth.LoginDeviceOAuth(ctx, llmprovider.ProviderGrok, auth.OAuthFlowOptions{
		NotifyDevice: func(verificationURL, userCode string) {
			t.Logf("open %s and enter the code %s", verificationURL, userCode)
		},
	})
	if err != nil {
		t.Fatalf("LoginDeviceOAuth: %v", err)
	}
	revokeAfter(t, session)
	checkWord(ctx, t, liveGrok(t, session, "grok-4.6"), "ALPHA", noSkip)
}

// TestLive_KiloDeviceLogin is the owner-run gate for Kilo's device login
// (0017-MADR D2; 0017-PLAN U2). It REQUIRES LLMPROVIDER_LIVE_DEVICE_LOGIN=1
// and a person to approve the code it logs, within five minutes. The approved
// token never refreshes, so the session holds no refresh token and no expiry.
// It reads the account's profile, logging counts only, and generates once on
// a free model. Kilo has no revocation, so nothing is revoked.
func TestLive_KiloDeviceLogin(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_DEVICE_LOGIN") != "1" {
		t.Skip("LLMPROVIDER_LIVE_DEVICE_LOGIN unset: this needs a person to approve a device code")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	session, err := auth.LoginDeviceOAuth(ctx, llmprovider.ProviderKilo, auth.OAuthFlowOptions{
		NotifyDevice: func(verificationURL, userCode string) {
			t.Logf("open %s and enter the code %s", verificationURL, userCode)
		},
	})
	if err != nil {
		t.Fatalf("LoginDeviceOAuth: %v", err)
	}
	checkKiloSession(ctx, t, session, llmprovider.LiveModel(t, llmprovider.ProviderKilo, llmprovider.LiveKiloFreeCollecting...),
		llmprovider.SkipIfTransient)
}
