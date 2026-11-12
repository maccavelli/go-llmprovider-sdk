//go:build live_gateways

package llmprovider_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestLive_0076Identity is 0076-PLAN Phase 4's owner-run probe: whether
// OpenAI's browser sign-in and xAI's device sign-in complete under a custom
// name, auth.Identity{Name: "llmprovider-probe"} (0076-MADR D3). It REQUIRES
// LLMPROVIDER_LIVE_0076=1 and a person. Its OpenAI half also REQUIRES
// LLMPROVIDER_LIVE_BROWSER_LOGIN=1, and its Grok half
// LLMPROVIDER_LIVE_DEVICE_LOGIN=1, as AGENTS.md "Live tests" assigns them.
// Each half checks the name it sent, logs whether the sign-in completed,
// binds the session to a scratch store, lists models with it (no generation)
// and revokes it; a session a failed check leaves unrevoked is revoked when
// the half ends. A sign-in refused under the name is a deviation (0076-PLAN
// Phase 4 step 2).
func TestLive_0076Identity(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0076") != "1" {
		t.Skip("LLMPROVIDER_LIVE_0076 unset: this signs in at OpenAI and xAI under a custom name, with a person")
	}
	identity := auth.Identity{Name: probeIdentityName}

	t.Run("OpenAIBrowser", func(t *testing.T) {
		if os.Getenv("LLMPROVIDER_LIVE_BROWSER_LOGIN") != "1" {
			t.Skip("LLMPROVIDER_LIVE_BROWSER_LOGIN unset: this needs a person to sign in in a browser")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		session, err := auth.LoginBrowserOAuth(ctx, llmprovider.ProviderOpenAI, auth.OAuthFlowOptions{
			Identity: identity,
			// LoginBrowserOAuth ignores an error OpenURL returns, so a wrong
			// URL ends the sign-in through cancel, before anyone signs in.
			OpenURL: func(raw string) error {
				if !checkAuthorizeIdentity(t, raw, "originator", identity.Name) {
					cancel()
					return errors.New("the authorize URL does not carry the probe's originator")
				}
				t.Logf("open this URL and sign in: %s", raw)
				return nil
			},
		})
		revoked := probeSignIn(t, "OpenAI's browser sign-in", session, err)
		checkIdentitySession(ctx, t, session)
		checkIdentityRevoked(ctx, t, session)
		*revoked = true
	})

	t.Run("GrokDevice", func(t *testing.T) {
		if os.Getenv("LLMPROVIDER_LIVE_DEVICE_LOGIN") != "1" {
			t.Skip("LLMPROVIDER_LIVE_DEVICE_LOGIN unset: this needs a person to approve a device code")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		rec := &identityRecorder{base: http.DefaultTransport.(*http.Transport).Clone()}
		session, err := auth.LoginDeviceOAuth(ctx, llmprovider.ProviderGrok, auth.OAuthFlowOptions{
			Identity:   identity,
			HTTPClient: &http.Client{Transport: rec},
			NotifyDevice: func(verificationURL, userCode string) {
				t.Logf("open %s and enter the code %s", verificationURL, userCode)
			},
		})
		checkDeviceReferrer(t, rec, identity.Name)
		revoked := probeSignIn(t, "xAI's device sign-in", session, err)
		checkIdentitySession(ctx, t, session)
		checkIdentityRevoked(ctx, t, session)
		*revoked = true
	})
}

// probeSignIn logs whether a sign-in under the probe's name completed, and
// ends the half when it did not. When it did, the session is revoked when the
// half ends unless the returned flag is set first, so that a check that
// stops early leaves no session of the probe behind.
func probeSignIn(t *testing.T, signIn string, session *auth.OAuthSession, err error) *bool {
	t.Helper()
	if err != nil {
		t.Fatalf("FINDING sign-in: %s under %q did not complete: %v", signIn, probeIdentityName, err)
	}
	t.Logf("FINDING sign-in: %s under %q completed", signIn, probeIdentityName)
	revoked := false
	t.Cleanup(func() {
		if revoked {
			return
		}
		if err := auth.RevokeOAuthSession(context.Background(), session); err != nil {
			t.Errorf("RevokeOAuthSession when the half ends: %v", err)
		}
	})
	return &revoked
}
