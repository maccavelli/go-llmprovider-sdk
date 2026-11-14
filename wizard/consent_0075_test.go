package wizard

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
)

const consent0075 = "chatgpt_plan.background_use"

// TestWizardPlanConsentSelect_0075 (0075-MADR D4): the consent is a Select of
// three choices, the second the default, its title the step's text and the
// usage page; "Cancel the sign-in" gives "declined", with no login and no
// save.
func TestWizardPlanConsentSelect_0075(t *testing.T) {
	for _, c := range []struct {
		name   string
		choice int
		want   string
	}{
		{"allow", 0, consent0075 + "=granted"},
		{"do not allow", 1, consent0075 + "=declined"},
		{"cancel", 2, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			calls := stubPlanLogin(t, func(context.Context, auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
				return planSessionFor("oaiapp_0075"), nil
			})
			store := newMemoryTokenStore()
			f := newFake(t, fakePrompter{
				selects: []int{providerIdx(t, llmprovider.ProviderOpenAI), planMethodIdx(t), c.choice},
				inputs:  []string{"gpt-6.1-sol"},
			})
			f.allowLeftover = c.choice == 2
			_, err := ConfigureLLM(context.Background(), f, Options{TokenStore: store, AgentName: planTestApp})
			if len(f.seenSelect) < 3 {
				t.Fatalf("Select titles = %q; want the consent third", f.seenSelect)
			}
			title := f.seenSelect[2]
			if !strings.Contains(title, "You're using your ChatGPT plan") || !strings.Contains(title, openai.ChatGPTUsageURL) {
				t.Errorf("the consent's title = %q; want the step's text and the usage page", title)
			}
			if got := labels(f.seenSelectItems[2]); !slices.Equal(got, []string{"Allow background use", "Do not allow background use", "Cancel the sign-in"}) {
				t.Errorf("the consent's choices = %q", got)
			}
			if f.seenSelectDefault[2] != 1 {
				t.Errorf("the consent's default = %d; want the second choice", f.seenSelectDefault[2])
			}
			if c.choice == 2 {
				if err == nil || !strings.Contains(err.Error(), "declined") || *calls != 0 || store.saves != 0 {
					t.Errorf("err %v, %d logins, %d saves; want declined, none, none", err, *calls, store.saves)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if saved := store.sessions[llmprovider.ProviderOpenAI]; saved == nil || saved.Consents != c.want {
				t.Errorf("saved session = %v; want Consents %q", saved, c.want)
			}
		})
	}
	t.Run("the app's wording", func(t *testing.T) {
		stubPlanLogin(t, func(context.Context, auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
			return planSessionFor("oaiapp_0075"), nil
		})
		f := newFake(t, fakePrompter{selects: []int{planMethodIdx(t), 0}})
		_, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), AgentName: planTestApp,
			ConsentText: map[string]string{consent0075: "Let My App work on its own?"}}, llmprovider.ProviderOpenAI)
		if err != nil {
			t.Fatal(err)
		}
		if title := f.seenSelect[1]; !strings.Contains(title, "Let My App work on its own?") || strings.Contains(title, "You're using") ||
			!strings.Contains(title, openai.ChatGPTUsageURL) {
			t.Errorf("the consent's title = %q; want the app's wording and the usage page", title)
		}
	})
}

// TestWizardReSignInWithoutAnswerAsks_0075 (0075-MADR D4): a stored plan
// session with no answer shows the consent Select before the login, passes
// the session as Previous, and records the choice on the saved session.
func TestWizardReSignInWithoutAnswerAsks_0075(t *testing.T) {
	calls := stubPlanLogin(t, func(_ context.Context, opts auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		if opts.Previous == nil || opts.Previous.ClientID != "oaiapp_prev" {
			t.Errorf("Previous %v; want the stored plan session", opts.Previous)
		}
		return planSessionFor("oaiapp_prev"), nil
	})
	store := newMemoryTokenStore()
	store.sessions[llmprovider.ProviderOpenAI] = planSessionFor("oaiapp_prev")
	f := newFake(t, fakePrompter{
		selects: []int{providerIdx(t, llmprovider.ProviderOpenAI), planMethodIdx(t), 0},
		inputs:  []string{"gpt-6.1-sol"},
	})
	if _, err := ConfigureLLM(context.Background(), f, Options{TokenStore: store, AgentName: planTestApp}); err != nil {
		t.Fatal(err)
	}
	if *calls != 1 || len(f.seenSelect) != 3 || !strings.Contains(f.seenSelect[2], "You're using your ChatGPT plan") {
		t.Errorf("%d logins, selects %q; want one login after the consent Select", *calls, f.seenSelect)
	}
	if saved := store.sessions[llmprovider.ProviderOpenAI]; saved == nil || saved.Consents != consent0075+"=granted" {
		t.Errorf("saved %v; want the choice recorded", saved)
	}
}
