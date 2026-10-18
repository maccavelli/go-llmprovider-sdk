//go:build live_gateways

package llmprovider_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
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

	session, store := planSignIn(ctx, t)
	revoked := false
	t.Cleanup(func() {
		if !revoked {
			if err := auth.RevokeOAuthSession(context.Background(), session); err != nil {
				t.Errorf("RevokeOAuthSession at cleanup: %v", err)
			}
		}
	})
	planRequests(ctx, t, session)
	planRefresh(ctx, t, session, store)

	if err := auth.RevokeOAuthSession(ctx, session); err != nil {
		t.Fatalf("RevokeOAuthSession: %v", err)
	}
	revoked = true
	session.Invalidate()
	if _, err := session.Token(ctx); err == nil {
		t.Fatal("a refresh after revocation succeeded")
	} else if !errors.Is(err, llmprovider.ErrAuthFailure) {
		t.Logf("FINDING refresh after revocation: refused, but not as ErrAuthFailure: %v", err)
	} else {
		t.Log("FINDING refresh after revocation: refused, ErrAuthFailure")
	}
}

// planSignIn signs in, checks the session's kind, and returns what a
// FileTokenStore gives back after saving it, bound to that store.
func planSignIn(ctx context.Context, t *testing.T) (*auth.OAuthSession, auth.TokenStore) {
	t.Helper()
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
	if !signedIn.PlanUsage() || signedIn.ChatGPT() || !strings.HasPrefix(signedIn.ClientID, "oaia") ||
		!strings.Contains(" "+signedIn.Scope+" ", " chatgpt.tokens.use.direct ") || signedIn.HostID == "" {
		t.Fatalf("session: plan %v, ChatGPT %v, issued client %v (%d chars), scope has plan usage %v, host id %v",
			signedIn.PlanUsage(), signedIn.ChatGPT(), strings.HasPrefix(signedIn.ClientID, "oaia"), len(signedIn.ClientID),
			strings.Contains(signedIn.Scope, "chatgpt.tokens.use.direct"), signedIn.HostID != "")
	}
	t.Logf("FINDING sign-in: issued client of %d chars; token URL %s", len(signedIn.ClientID), signedIn.TokenURL)

	store, err := auth.NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, llmprovider.ProviderOpenAI, signedIn); err != nil {
		t.Fatalf("Save: %v", err)
	}
	session, err := store.Load(ctx, llmprovider.ProviderOpenAI)
	if err != nil || session == nil || !session.PlanUsage() || session.ClientID != signedIn.ClientID ||
		session.Scope != signedIn.Scope || session.HostID != signedIn.HostID || session.IDToken == "" {
		t.Fatalf("the stored session does not round-trip: err %v", err)
	}
	session.Store = store
	return session, store
}

// planRequests lists the account's models with probes asked for, then
// generates, streams and calls a forced tool on the first.
func planRequests(ctx context.Context, t *testing.T, session *auth.OAuthSession) {
	t.Helper()
	lister := liveOpenAI(t, session, "", llmprovider.WithModelProbes(true)).(llmprovider.ModelLister)
	models, err := lister.ListModels(ctx)
	llmprovider.SkipIfTransient(t, err)
	if err != nil || len(models) == 0 {
		t.Fatalf("ListModels = %v, %v", models, err)
	}
	t.Logf("FINDING listing: %d listed models, the first %q", len(models), models[0])
	raw, err := catalog.ListRaw(ctx, llmprovider.ProviderOpenAI, session)
	if err != nil || len(raw.Models) < len(models) || raw.Models[0].ID == "" {
		t.Fatalf("ListRaw: %d entries, %v", len(raw.Models), err)
	}
	p := liveOpenAI(t, session, models[0])

	out, err := llmprovider.GenerateText(ctx, p, userText("Reply with only the word ALPHA"))
	llmprovider.SkipIfTransient(t, err)
	if err != nil || !strings.Contains(strings.ToUpper(out), "ALPHA") {
		t.Fatalf("Generate = %q, %v", out, err)
	}

	var streamed strings.Builder
	done := false
	for ev, err := range llmprovider.Stream(ctx, p, userText("Reply with only the word BRAVO")) {
		llmprovider.SkipIfTransient(t, err)
		if err != nil {
			t.Fatalf("Stream: %v", err)
		}
		if ev.Type == llmprovider.EventTextDelta {
			streamed.WriteString(ev.Text)
		}
		done = done || ev.Type == llmprovider.EventDone
	}
	if !done || !strings.Contains(strings.ToUpper(streamed.String()), "BRAVO") {
		t.Fatalf("Stream = %q, done %v", streamed.String(), done)
	}

	tool := llmprovider.Tool{Name: "get_weather", Description: "Get the weather for a city", Schema: map[string]any{
		"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}, "required": []string{"city"}}}
	call, err := llmprovider.GenerateToolCall(ctx, p, &llmprovider.Request{
		Input: userText("What is the weather in Paris?").Input, Tools: []llmprovider.Tool{tool},
		ToolChoice: llmprovider.ForceTool(tool.Name)})
	llmprovider.SkipIfTransient(t, err)
	if err != nil || call.Name != tool.Name || !strings.Contains(call.Arguments, "Paris") {
		t.Fatalf("forced tool = %+v, %v", call, err)
	}
}

// planRefresh forces a refresh, and checks that the token rotated and that
// the store holds the rotation with its plan fields.
func planRefresh(ctx context.Context, t *testing.T, session *auth.OAuthSession, store auth.TokenStore) {
	t.Helper()
	firstAccess, err := session.Token(ctx)
	if err != nil {
		t.Fatal(err)
	}
	session.Invalidate()
	refreshed, err := session.Token(ctx)
	if err != nil || refreshed.Value == firstAccess.Value || !refreshed.Expiry.After(time.Now()) {
		t.Fatalf("forced refresh: err %v, new token %v", err, refreshed.Value != firstAccess.Value)
	}
	if again, err := store.Load(ctx, llmprovider.ProviderOpenAI); err != nil || again == nil || again.Scope == "" || again.HostID != session.HostID {
		t.Fatalf("the refreshed session was not saved with its plan fields: %v", err)
	}
	t.Log("FINDING refresh: the token rotated and the store holds it")
}
