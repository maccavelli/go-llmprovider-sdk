//go:build live_gateways

package llmprovider_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// liveName is the application name TestLive_0079Name signs in, lists,
// generates, refreshes and revokes under (0079-PLAN Phase 5, the owner's
// ruling "gobble in both").
const liveName = "gobble"

// newLiveNameProbe is a probe of liveName with a scratch FileTokenStore, and
// the client that records every request of the half, the sign-in's included,
// through idrec, which also keeps Grok's device form's referrer.
func newLiveNameProbe(t *testing.T) (nameProbe, *identityRecorder) {
	t.Helper()
	store, err := auth.NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rec := &nameRecorder{}
	idrec := &identityRecorder{base: http.DefaultTransport}
	return nameProbe{name: liveName, rec: rec, client: &http.Client{Transport: rec.wrap(idrec)}, store: store}, idrec
}

// TestLive_0079Name is 0079-PLAN Phase 5, the owner's probe of the name on
// the sign-in, the listing, the generation, the refresh and the revocation,
// at both vendors, under "gobble". It REQUIRES LLMPROVIDER_LIVE_0079=1 and a
// person for each half's sign-in, within five minutes. Each half signs in
// under the name, saves the session to a scratch store and loads it back, so
// that the session has no identity and no client until the provider or the
// listing it is given to hands it the name (0079-MADR D4); what the issuer
// and the backend then receive is recorded as originator and User-Agent
// product name only (nameRecorder). A call that fails under the name is run
// once without it, to tell a refusal of the name from another failure.
// Nothing is logged but counts and names: no URL's query, token or id.
//
//   - ChatGPT, also under LLMPROVIDER_LIVE_BROWSER_LOGIN and
//     LLMPROVIDER_LIVE_CHATGPT (it spends one short generation of the owner's
//     ChatGPT plan): a browser sign-in, the model listing, one generation, a
//     forced refresh and the revocation.
//   - Grok, also under LLMPROVIDER_LIVE_DEVICE_LOGIN: a device sign-in, the
//     raw listing, a forced refresh and the revocation.
func TestLive_0079Name(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0079") != "1" {
		t.Skip("LLMPROVIDER_LIVE_0079 unset: this signs in at OpenAI and xAI with a person")
	}
	t.Run("ChatGPT", func(t *testing.T) {
		if os.Getenv("LLMPROVIDER_LIVE_BROWSER_LOGIN") != "1" || os.Getenv("LLMPROVIDER_LIVE_CHATGPT") != "1" {
			t.Skip("LLMPROVIDER_LIVE_BROWSER_LOGIN and LLMPROVIDER_LIVE_CHATGPT unset: a browser sign-in, and a generation on the ChatGPT plan")
		}
		n, _ := newLiveNameProbe(t)
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
		defer cancel()
		signInCtx, stopSignIn := context.WithTimeout(ctx, 5*time.Minute)
		defer stopSignIn()
		signed, err := auth.LoginBrowserOAuth(signInCtx, llmprovider.ProviderOpenAI, auth.OAuthFlowOptions{
			HTTPClient: n.client,
			Identity:   auth.Identity{Name: liveName},
			OpenURL: func(raw string) error {
				if !checkAuthorizeIdentity(t, raw, "originator", liveName) {
					stopSignIn()
					return nil
				}
				t.Logf("open this URL and sign in: %s", raw)
				return nil
			},
		})
		if err != nil {
			t.Fatalf("LoginBrowserOAuth: %v", err)
		}
		t.Logf("FINDING sign-in: the authorize URL carried originator %q and the sign-in completed", liveName)
		runChatGPTName(ctx, t, n, signed)
	})
	t.Run("Grok", func(t *testing.T) {
		if os.Getenv("LLMPROVIDER_LIVE_DEVICE_LOGIN") != "1" {
			t.Skip("LLMPROVIDER_LIVE_DEVICE_LOGIN unset: this needs a person to approve a device code")
		}
		n, idrec := newLiveNameProbe(t)
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
		defer cancel()
		signInCtx, stopSignIn := context.WithTimeout(ctx, 5*time.Minute)
		defer stopSignIn()
		login, err := auth.StartDeviceOAuth(signInCtx, llmprovider.ProviderGrok, auth.OAuthFlowOptions{
			HTTPClient: n.client, Identity: auth.Identity{Name: liveName}})
		if err != nil {
			t.Fatalf("StartDeviceOAuth: %v", err)
		}
		checkDeviceReferrer(t, idrec, liveName)
		t.Logf("open %s and enter the code %s", login.VerificationURI, login.UserCode)
		signed, err := login.Wait(signInCtx)
		if err != nil {
			t.Fatalf("Wait: %v", err)
		}
		t.Logf("FINDING sign-in: the device form's referrer was %q and the sign-in completed", liveName)
		runGrokName(ctx, t, n, signed)
	})
}
