package llmprovider_test

// The checks the person-gated live tests run after their sign-in, untagged so
// that an offline twin (live_twins_test.go) runs each against a fake and a
// planted break of a check is caught with no sign-in (0046-MADR D3). The live
// tests call them with the real session; the twins with a fake's.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/grok"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/kilo"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
)

// liveOpenAI builds OpenAI from a session, or fails the test.
func liveOpenAI(t *testing.T, src llmprovider.TokenSource, model string, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := openai.New(append([]llmprovider.Option{llmprovider.WithTokenSource(src), llmprovider.WithModel(model)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func liveGrok(t *testing.T, src llmprovider.TokenSource, model string, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := grok.New(append([]llmprovider.Option{llmprovider.WithTokenSource(src), llmprovider.WithModel(model)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func userText(text string) *llmprovider.Request {
	return &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: "user", Text: text}}}
}

// transientSkip is how a check skips a transient failure: the live suite's
// llmprovider.SkipIfTransient where a live test already skipped, noSkip where
// it did not, and noSkip in every twin.
type transientSkip func(*testing.T, error)

func noSkip(*testing.T, error) {}

// checkWord asks p for word alone, and fails unless the reply holds it.
func checkWord(ctx context.Context, t *testing.T, p llmprovider.Provider, word string, skip transientSkip) {
	t.Helper()
	out, err := llmprovider.GenerateText(ctx, p, userText("Reply with only the word "+word))
	skip(t, err)
	if err != nil || !strings.Contains(strings.ToUpper(out), word) {
		t.Fatalf("Generate = %q, %v", out, err)
	}
}

// checkKiloSession checks a Kilo device login's session: an access token with
// no refresh and no expiry, the account's profile (counts only), and one
// generation on model.
func checkKiloSession(ctx context.Context, t *testing.T, session *auth.OAuthSession, model string, skip transientSkip,
	opts ...llmprovider.Option) {
	t.Helper()
	if session.Access == "" || session.Refresh != "" || !session.Expiry.IsZero() {
		t.Fatalf("session: access set %t, refresh set %t, expiry %v; want an access token only, with no refresh and no expiry",
			session.Access != "", session.Refresh != "", session.Expiry)
	}
	account, err := auth.KiloProfile(ctx, session.Access, opts...)
	if err != nil {
		t.Fatalf("KiloProfile: %v", err)
	}
	t.Logf("profile: %d organizations, personal account %t", len(account.Organizations), account.HasPersonalAccount)
	p, err := kilo.New(append([]llmprovider.Option{llmprovider.WithTokenSource(session), kilo.WithDataCollection(true),
		llmprovider.WithModel(model)}, opts...)...)
	if err != nil {
		t.Fatalf("kilo.New: %v", err)
	}
	out, err := llmprovider.GenerateText(ctx, p, userText("Reply with only the word ALPHA"))
	skip(t, err)
	if err != nil || strings.TrimSpace(out) == "" {
		t.Fatalf("Generate = %q, %v", out, err)
	}
}

// checkPlanKind checks a session from LoginChatGPTPlan: a plan session, not a
// ChatGPT one, under an issued client, with plan usage granted and a host id.
// It logs no token and no client id.
func checkPlanKind(t *testing.T, s *auth.OAuthSession) {
	t.Helper()
	if !s.PlanUsage() || s.ChatGPT() || !strings.HasPrefix(s.ClientID, "oaia") ||
		!strings.Contains(" "+s.Scope+" ", " chatgpt.tokens.use.direct ") || s.HostID == "" {
		t.Fatalf("session: plan %v, ChatGPT %v, issued client %v (%d chars), scope has plan usage %v, host id %v",
			s.PlanUsage(), s.ChatGPT(), strings.HasPrefix(s.ClientID, "oaia"), len(s.ClientID),
			strings.Contains(s.Scope, "chatgpt.tokens.use.direct"), s.HostID != "")
	}
	t.Logf("FINDING sign-in: issued client of %d chars; token URL %s", len(s.ClientID), s.TokenURL)
}

// planStoreRoundTrip saves signedIn to a FileTokenStore, and returns what the
// store gives back, bound to it.
func planStoreRoundTrip(ctx context.Context, t *testing.T, signedIn *auth.OAuthSession) (*auth.OAuthSession, auth.TokenStore) {
	t.Helper()
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
// generates, streams and calls a forced tool on the first. opts reach every
// provider and listing it builds.
func planRequests(ctx context.Context, t *testing.T, session *auth.OAuthSession, skip transientSkip, opts ...llmprovider.Option) {
	t.Helper()
	lister := liveOpenAI(t, session, "", append([]llmprovider.Option{llmprovider.WithModelProbes(true)}, opts...)...).(llmprovider.ModelLister)
	models, err := lister.ListModels(ctx)
	skip(t, err)
	if err != nil || len(models) == 0 {
		t.Fatalf("ListModels = %v, %v", models, err)
	}
	t.Logf("FINDING listing: %d listed models, the first %q", len(models), models[0])
	raw, err := catalog.ListRaw(ctx, llmprovider.ProviderOpenAI, session, opts...)
	if err != nil || len(raw.Models) < len(models) || raw.Models[0].ID == "" {
		t.Fatalf("ListRaw: %d entries, %v", len(raw.Models), err)
	}
	p := liveOpenAI(t, session, models[0], opts...)

	out, err := llmprovider.GenerateText(ctx, p, userText("Reply with only the word ALPHA"))
	skip(t, err)
	if err != nil || !strings.Contains(strings.ToUpper(out), "ALPHA") {
		t.Fatalf("Generate = %q, %v", out, err)
	}

	var streamed strings.Builder
	done := false
	for ev, err := range llmprovider.Stream(ctx, p, userText("Reply with only the word BRAVO")) {
		skip(t, err)
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
	skip(t, err)
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
	if again, err := store.Load(ctx, llmprovider.ProviderOpenAI); err != nil || again == nil || again.Scope == "" ||
		again.HostID != session.HostID {
		t.Fatalf("the refreshed session was not saved with its plan fields: %v", err)
	}
	t.Log("FINDING refresh: the token rotated and the store holds it")
}

// planRevoke revokes the session, and checks that a refresh afterwards is
// refused.
func planRevoke(ctx context.Context, t *testing.T, session *auth.OAuthSession) {
	t.Helper()
	if err := auth.RevokeOAuthSession(ctx, session); err != nil {
		t.Fatalf("RevokeOAuthSession: %v", err)
	}
	session.Invalidate()
	if _, err := session.Token(ctx); err == nil {
		t.Fatal("a refresh after revocation succeeded")
	} else if !errors.Is(err, llmprovider.ErrAuthFailure) {
		t.Logf("FINDING refresh after revocation: refused, but not as ErrAuthFailure: %v", err)
	} else {
		t.Log("FINDING refresh after revocation: refused, ErrAuthFailure")
	}
}
