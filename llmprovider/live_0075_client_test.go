//go:build live_gateways

package llmprovider_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/x/signin"
)

// browserPlanSignIn signs in to the ChatGPT plan through x/signin by the
// loopback, in a browser on this host, with store as its ClientStore, and
// returns the session and the authorize URL it showed. It answers the
// usage-terms confirm yes and logs its text: the owner's run is the consent.
// The one FINDING sign-in line says what happened. It fails the test unless
// the sign-in completed.
func browserPlanSignIn(ctx context.Context, t *testing.T, store auth.ClientStore, label, onFail string) (*auth.OAuthSession, string) {
	t.Helper()
	run, err := signin.Start(ctx, llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan,
		signin.Options{AgentName: pasteProbeAgentName, ClientStore: store, NoPaste: true})
	if err != nil {
		t.Fatalf("signin.Start (%s): %v", label, err)
	}
	t.Cleanup(run.Cancel)
	var shown string
	kinds, err := driveSteps(ctx, run, stepHooks{
		confirm: func(step authcontract.Step) bool {
			t.Logf("consent %s, answered yes by the owner's run: %s", step.ConfirmID, step.Text)
			return true
		},
		openURL: func(step authcontract.Step) {
			shown = step.URL
			t.Logf("open this URL in a browser on this host and sign in (%s sign-in): %s", label, step.URL)
		},
	})
	if err == nil {
		_, err = run.Wait(ctx)
	}
	port, perr := redirectPort(shown)
	if perr != nil {
		port = 0
	}
	line, passed := loopbackFinding("OpenAI plan "+label, port, kinds, err)
	t.Log(line)
	if !passed {
		t.Log(onFail)
		t.FailNow()
	}
	return sessionOf(ctx, t, run), shown
}

// TestLive_0075ClientReuse is 0075-PLAN Phase 6 steps 1 and 2: whether OpenAI
// accepts a plan sign-in that reuses the issued client with no
// id_token_hint, and the consent end to end. It REQUIRES
// LLMPROVIDER_LIVE_0075=1 and LLMPROVIDER_LIVE_CHATGPT_PLAN=1, and a person at
// a browser on this host for two sign-ins. A first sign-in registers one
// client under the owner's ChatGPT account and its client is saved; the
// session is revoked and deleted, as wizard.Logout does; a second sign-in
// names the kept client and host id with no session, so no hint. It records
// whether OpenAI accepted it and whether the client is unchanged. It then
// declines the consent (a marked request is refused, no request sent), grants
// it, and generates once, marked. Everything is kept in a scratch store; no
// token and no id is logged.
func TestLive_0075ClientReuse(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0075") != "1" {
		t.Skip("LLMPROVIDER_LIVE_0075 unset: this registers a client at OpenAI and needs a person to sign in")
	}
	if os.Getenv("LLMPROVIDER_LIVE_CHATGPT_PLAN") != "1" {
		t.Skip("LLMPROVIDER_LIVE_CHATGPT_PLAN unset: the plan sign-in registers a client under the owner's account")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()
	store, err := auth.NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	first, _ := browserPlanSignIn(ctx, t, store, "first", "the first sign-in did not complete; run again")
	firstDone := revokeUnlessDone(t, first)
	checkPlanKind(t, first)
	kept, err := store.LoadClient(ctx, llmprovider.ProviderOpenAI)
	if err != nil {
		t.Fatalf("LoadClient: %v", err)
	}
	if err := reusedClientError(first, kept); err != nil {
		t.Fatalf("FINDING client: the first sign-in did not save its client: %v", err)
	}
	t.Log("FINDING client: the first sign-in saved its issued client and host id")
	planRevoke(ctx, t, first)
	firstDone()
	if err := logoutKeepsClientError(ctx, store, first); err != nil {
		t.Fatalf("FINDING client: %v", err)
	}
	t.Log("FINDING client: the sign-out deleted the session and kept the client record")

	signInCtx, stopSignIn := context.WithTimeout(ctx, 8*time.Minute)
	defer stopSignIn()
	second, shown := browserPlanSignIn(signInCtx, t, store, "second",
		"FINDING reuse: if the page showed a refusal of the client, OpenAI refuses it with no id_token_hint: "+
			"that is a deviation of 0075-PLAN Phase 6 step 1; report what the page said, quoting no id")
	if err := reuseAuthorizeError(shown, kept); err != nil {
		t.Errorf("FINDING reuse: %v", err)
	}
	secondDone := revokeUnlessDone(t, second)
	if err := reusedClientError(second, kept); err != nil {
		t.Fatalf("FINDING reuse: OpenAI accepted the sign-in, but %v", err)
	}
	after, err := store.LoadClient(ctx, llmprovider.ProviderOpenAI)
	if err != nil || after != kept {
		t.Fatalf("FINDING reuse: the client record after the second sign-in: err %v, unchanged %t", err, after == kept)
	}
	t.Log("FINDING reuse: OpenAI accepted the client with no id_token_hint; the client and host id are unchanged")

	models, err := liveOpenAI(t, second, "").(llmprovider.ModelLister).ListModels(ctx)
	if err != nil || len(models) == 0 {
		t.Fatalf("ListModels = %v, %v", models, err)
	}
	checkPlanConsent(ctx, t, second, store, models[0], &http.Client{})
	planRevoke(ctx, t, second)
	secondDone()
}
