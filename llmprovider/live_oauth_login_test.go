//go:build live_gateways

package llmprovider_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestLive_ChatGPTBrowserLogin is the owner-run gate for the 127.0.0.1
// redirect (MADR 0012 §5.3). It REQUIRES LLMPROVIDER_LIVE_BROWSER_LOGIN=1 and a
// person at a browser: it prints the authorize URL, waits up to five minutes
// for the ChatGPT login to redirect back, generates once with the new
// session, and then revokes it, so no session of this module outlives the test.
func TestLive_ChatGPTBrowserLogin(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_BROWSER_LOGIN") != "1" {
		t.Skip("LLMPROVIDER_LIVE_BROWSER_LOGIN unset: this needs a person to sign in in a browser")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	session, err := llmprovider.LoginBrowserOAuth(ctx, llmprovider.ProviderOpenAI, llmprovider.OAuthFlowOptions{
		OpenURL: func(u string) error {
			t.Logf("open this URL and sign in: %s", u)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("LoginBrowserOAuth: %v", err)
	}
	defer func() {
		if err := llmprovider.RevokeOAuthSession(context.Background(), session); err != nil {
			t.Errorf("RevokeOAuthSession: %v", err)
		}
	}()
	out, err := llmprovider.GenerateText(ctx, liveOpenAI(t, session, "gpt-6-astra"), userText("Reply with only the word ALPHA"))
	if err != nil || !strings.Contains(strings.ToUpper(out), "ALPHA") {
		t.Fatalf("Generate = %q, %v", out, err)
	}
}
