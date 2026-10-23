package wizard

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// grokBrowserIdx is the browser sign-in's place in Grok's menu with a store.
const grokBrowserIdx = 1

// TestNoPastePrompt_LoopbackOnly_0057 (0057-MADR D2, issue 7): with
// NoPastePrompt, a browser sign-in gives the login no InputCode, reads
// nothing, shows no paste instruction, and still shows and opens the URL.
func TestNoPastePrompt_LoopbackOnly_0057(t *testing.T) {
	stubBrowserLogin(t, func(_ context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		if opts.InputCode != nil {
			t.Error("the login was given an InputCode under NoPastePrompt")
		}
		if err := opts.OpenURL("https://authorize.test"); err != nil {
			t.Errorf("OpenURL() error = %v", err)
		}
		return testOAuthSession(provider), nil // the loopback returned
	})
	opened := 0
	f := newFake(t, fakePrompter{selects: []int{grokBrowserIdx}})
	res, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), NoEndpointPrompt: true,
		NoPastePrompt: true, OpenURL: func(string) error { opened++; return nil }}, llmprovider.ProviderGrok)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != CredOAuth || opened != 1 || countContaining(f.seenNotify, "https://authorize.test") != 1 {
		t.Errorf("kind %q, opened %d, notices %q; want the session, the URL opened and shown once", res.Kind, opened, f.seenNotify)
	}
	for _, n := range f.seenNotify {
		if strings.Contains(strings.ToLower(n), "paste") {
			t.Errorf("notice %q under NoPastePrompt; want no paste instruction", n)
		}
	}
	if len(f.seenInput) != 0 {
		t.Errorf("inputs %q; want none", f.seenInput)
	}
}

// TestNoPastePrompt_PlanSignIn_0057 (0057-MADR D2, Q3 (a)): the ChatGPT plan
// sign-in stays offered under NoPastePrompt, and its login is given no
// InputCode: the loopback carries the issued client (F5).
func TestNoPastePrompt_PlanSignIn_0057(t *testing.T) {
	calls := stubPlanLogin(t, func(_ context.Context, opts auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		if opts.InputCode != nil {
			t.Error("the plan login was given an InputCode under NoPastePrompt")
		}
		if opts.OpenURL == nil {
			t.Error("the plan login was given no OpenURL")
		}
		return planSessionFor("oaiapp_0057"), nil
	})
	f := newFake(t, fakePrompter{selects: []int{planMethodIdx(t)}, confirms: []bool{true}})
	res, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), AgentName: planTestApp,
		NoPastePrompt: true}, llmprovider.ProviderOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	label := methodLabel(t, llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan)
	menu := menuLabels(f, "Choose how to authenticate with "+descriptor0056(t, llmprovider.ProviderOpenAI).Label+":")
	if !slices.Contains(menu, label) || *calls != 1 || res.ClientID != "oaiapp_0057" {
		t.Errorf("menu %q, %d plan logins, client %q; want %q offered and signed in", menu, *calls, res.ClientID, label)
	}
}

// TestNoPastePrompt_OtherMethodsUnchanged_0057: NoPastePrompt changes no
// provider's menu.
func TestNoPastePrompt_OtherMethodsUnchanged_0057(t *testing.T) {
	o := Options{TokenStore: newMemoryTokenStore(), AgentName: planTestApp}
	noPaste := o
	noPaste.NoPastePrompt = true
	for _, d := range providers.Default().Descriptors() {
		if got, want := offeredAuthMethods(d, noPaste), offeredAuthMethods(d, o); !slices.Equal(got, want) {
			t.Errorf("%s: methods %v under NoPastePrompt; want %v", d.ID, got, want)
		}
	}
}

// TestPaste_EmptyAskedAgain_0057 (0057-MADR D4): an empty paste is asked
// again, and the login gets the code that follows.
func TestPaste_EmptyAskedAgain_0057(t *testing.T) {
	stubBrowserLogin(t, func(ctx context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		if err := opts.OpenURL("https://authorize.test"); err != nil {
			t.Errorf("OpenURL() error = %v", err)
		}
		code, err := opts.InputCode(ctx)
		if err != nil || code != "code-0057" {
			t.Errorf("InputCode = %q, %v; want the code after the empty answer", code, err)
		}
		return testOAuthSession(provider), nil
	})
	f := newFake(t, fakePrompter{selects: []int{grokBrowserIdx}, inputs: []string{"", "code-0057"}})
	if _, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), NoEndpointPrompt: true},
		llmprovider.ProviderGrok); err != nil {
		t.Fatal(err)
	}
	if n := countContaining(f.seenInput, pasteCodePrompt); n != 2 {
		t.Errorf("inputs %q; want the paste prompt twice", f.seenInput)
	}
}

// TestPaste_EndOfInputEnds_0057: at the end of a TextPrompter's input, the
// paste prompt's second read is an error matching io.EOF, which ends the
// sign-in, as the empty answer did before 0057.
func TestPaste_EndOfInputEnds_0057(t *testing.T) {
	tp := &TextPrompter{In: strings.NewReader(""), Out: io.Discard}
	flow, drain := browserFlowOptions(tp, Options{})
	if err := flow.OpenURL("https://authorize.test"); err != nil {
		t.Fatal(err)
	}
	code, err := flow.InputCode(context.Background())
	drain()
	if !errors.Is(err, io.EOF) || code != "" {
		t.Errorf("InputCode = %q, %v; want an error matching io.EOF", code, err)
	}
}
