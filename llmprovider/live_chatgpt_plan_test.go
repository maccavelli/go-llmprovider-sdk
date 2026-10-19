//go:build live_gateways

package llmprovider_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// planAgentName is the app name the live test signs in under: this module's
// own, since the test is this module's (0043-MADR D4).
const planAgentName = "go-llmprovider-sdk"

// TestLive_ChatGPTPlan is 0043-MADR D9's live test of the shipped plan
// sign-in, which replaced 0017's probe. It REQUIRES
// LLMPROVIDER_LIVE_CHATGPT_PLAN=1 and a person to sign in at the URL it logs
// within ten minutes. Each run registers a new client at OpenAI under this
// module's name, and revokes the session at the end.
//
// It signs in, saves the session to a file store and uses what it loads back,
// lists the account's models with no probe, generates, streams, calls a
// forced tool, refreshes, revokes, and checks that a refresh after the
// revocation is refused. It logs no token and no client id.
func TestLive_ChatGPTPlan(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_CHATGPT_PLAN") != "1" {
		t.Skip("LLMPROVIDER_LIVE_CHATGPT_PLAN unset: this registers a client at OpenAI and needs a person to sign in")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()

	signedIn, err := auth.LoginChatGPTPlan(ctx, auth.ChatGPTPlanOptions{
		AgentName: planAgentName,
		OpenURL: func(raw string) error {
			t.Logf("open this URL and sign in: %s", raw)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("LoginChatGPTPlan: %v", err)
	}
	checkPlanKind(t, signedIn)
	session, store := planStoreRoundTrip(ctx, t, signedIn)
	revoked := false
	t.Cleanup(func() {
		if !revoked {
			if err := auth.RevokeOAuthSession(context.Background(), session); err != nil {
				t.Errorf("RevokeOAuthSession at cleanup: %v", err)
			}
		}
	})
	planRequests(ctx, t, session, llmprovider.SkipIfTransient)
	planRefresh(ctx, t, session, store)
	planRevoke(ctx, t, session)
	revoked = true
}
