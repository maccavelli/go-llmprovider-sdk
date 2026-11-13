//go:build live_gateways

package llmprovider_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestLive_0074GrokPaste is 0074-PLAN Phase 3a's owner-run probe (Phase 7
// step 4): whether Grok's browser sign-in completes by a paste, when the
// browser that signs in cannot reach the host's loopback. It REQUIRES
// LLMPROVIDER_LIVE_0074=1, LLMPROVIDER_LIVE_BROWSER_LOGIN=1 and
// LLMPROVIDER_LIVE_0074_GROK_PASTE_FILE, a path that must not exist yet, and
// a person. It uses shipped code only: LoginBrowserOAuth with an InputCode
// that reads the paste from that file (pasteFromFile), because go test gives
// the binary no standard input. The person opens the logged authorize URL in
// a browser that cannot reach this host's 127.0.0.1, signs in, and writes
// what the page shows into the file. The test logs one FINDING sign-in line
// (pasteFinding); on a pass it lists models with the session, with no
// generation, and revokes it. A session a failed check leaves behind is
// revoked when the test ends.
func TestLive_0074GrokPaste(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0074") != "1" {
		t.Skip("LLMPROVIDER_LIVE_0074 unset: this signs in at xAI with a person")
	}
	if os.Getenv("LLMPROVIDER_LIVE_BROWSER_LOGIN") != "1" {
		t.Skip("LLMPROVIDER_LIVE_BROWSER_LOGIN unset: this needs a person to sign in in a browser")
	}
	path := os.Getenv("LLMPROVIDER_LIVE_0074_GROK_PASTE_FILE")
	if path == "" {
		t.Skip("LLMPROVIDER_LIVE_0074_GROK_PASTE_FILE unset: the paste reaches the test through this file")
	}
	if _, err := os.Lstat(path); err == nil {
		t.Fatalf("LLMPROVIDER_LIVE_0074_GROK_PASTE_FILE %s exists; it must not exist when the test starts", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("LLMPROVIDER_LIVE_0074_GROK_PASTE_FILE %s: %v", path, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 11*time.Minute)
	defer cancel()

	t.Logf("after consent, write the page's text into %s: cat > that path, the paste, Enter, Ctrl-D", path)
	rec := &pasteRecord{}
	session, err := auth.LoginBrowserOAuth(ctx, llmprovider.ProviderGrok, auth.OAuthFlowOptions{
		OpenURL:   openAuthorizeURL(t, "referrer", cancel),
		InputCode: pasteFromFile(path, rec),
	})
	line, passed := pasteFinding(rec, err)
	t.Log(line)
	if err == nil {
		revoked := false
		t.Cleanup(func() {
			if revoked {
				return
			}
			if err := auth.RevokeOAuthSession(context.Background(), session); err != nil {
				t.Errorf("RevokeOAuthSession when the test ends: %v", err)
			}
		})
		if passed {
			checkIdentitySession(ctx, t, session)
			checkIdentityRevoked(ctx, t, session)
			revoked = true
		}
	}
	if !passed {
		t.FailNow()
	}
}
