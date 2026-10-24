package wizard

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
)

// 0056-MADR D1: ResolveCredential runs ConfigureLLM's credential step alone.

// TestResolveCredential_AsksNoProviderNoModel_0056: with the id given, the
// only question is the key; the Result has the key and no model, and the
// catalog is the listing that checked it.
func TestResolveCredential_AsksNoProviderNoModel_0056(t *testing.T) {
	var requests atomic.Int32
	f := newFake(t, fakePrompter{secrets: []string{"good-1"}})
	res, cat, err := ResolveCredential(context.Background(), f, Options{
		Discover: true, DiscoverLimit: 2 * time.Second, HTTPClient: claudeRefusals(0, &requests),
		ProviderOptions: []llmprovider.Option{testCache(t)},
	}, llmprovider.ProviderClaude)
	if err != nil {
		t.Fatalf("ResolveCredential: %v", err)
	}
	if len(f.seenSelect) != 0 || len(f.seenInput) != 0 || len(f.seenConfirm) != 0 || len(f.seenSecret) != 1 {
		t.Errorf("selects %q, inputs %q, confirms %q, secrets %q; want the key's prompt only",
			f.seenSelect, f.seenInput, f.seenConfirm, f.seenSecret)
	}
	if res.Provider != llmprovider.ProviderClaude || res.Kind != CredAPIKey || res.APIKey != "good-1" ||
		res.Model != "" || res.Fallbacks != nil {
		t.Errorf("Result %+v; want Claude's typed key, no model and no fallbacks", res)
	}
	model := catalog.Static(llmprovider.ProviderClaude)[0]
	if !cat.Live || !slices.Contains(cat.Usable, model) || requests.Load() != 1 {
		t.Errorf("catalog live %v, usable %q, after %d listings; want the one live listing, with %q",
			cat.Live, cat.Usable, requests.Load(), model)
	}
}

// TestResolveCredential_RefusedThreeKeys_0056: a refused typed key is asked
// for again; after three, the refusal is the error, and nothing is stored.
func TestResolveCredential_RefusedThreeKeys_0056(t *testing.T) {
	var requests atomic.Int32
	store := newMemoryTokenStore()
	f := newFake(t, fakePrompter{secrets: []string{"bad-1", "bad-2", "bad-3"}})
	_, _, err := ResolveCredential(context.Background(), f, Options{
		Discover: true, DiscoverLimit: 2 * time.Second, HTTPClient: claudeRefusals(3, &requests), TokenStore: store,
		ProviderOptions: []llmprovider.Option{testCache(t)},
	}, llmprovider.ProviderClaude)
	if !errors.Is(err, llmprovider.ErrAuthFailure) || err == nil || !strings.Contains(err.Error(), "refused 3 keys") {
		t.Fatalf("error %v; want ErrAuthFailure naming 3 keys", err)
	}
	if len(f.seenSecret) != 3 || store.saves != 0 {
		t.Errorf("%d key prompts, %d saves; want 3 and none", len(f.seenSecret), store.saves)
	}
}

// TestResolveCredential_ChatGPTPlan_0056: OpenAI's plan sign-in shows its
// notice and saves the session, as ConfigureLLM's does, and no model id is
// asked, where ConfigureLLM asks one.
func TestResolveCredential_ChatGPTPlan_0056(t *testing.T) {
	calls := stubPlanLogin(t, func(context.Context, auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		return planSessionFor("oaiapp_0056"), nil
	})
	store := newMemoryTokenStore()
	f := newFake(t, fakePrompter{selects: []int{planMethodIdx(t)}, confirms: []bool{true}})
	res, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: store, AgentName: planTestApp},
		llmprovider.ProviderOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	if *calls != 1 || res.Kind != CredOAuth || res.ClientID != "oaiapp_0056" || res.Model != "" {
		t.Errorf("calls %d, Result %+v; want one plan sign-in, oauth, no model", *calls, res)
	}
	if saved := store.sessions[llmprovider.ProviderOpenAI]; saved == nil || !saved.PlanUsage() {
		t.Errorf("saved %v; want the plan session", saved)
	}
	if len(f.seenConfirm) != 1 || !strings.Contains(f.seenConfirm[0], "You're using your ChatGPT plan") ||
		!strings.Contains(f.seenConfirm[0], openai.ChatGPTUsageURL) {
		t.Errorf("confirms %q; want OpenAI's notice with the usage page", f.seenConfirm)
	}
	if len(f.seenInput) != 0 {
		t.Errorf("inputs %q; want none, no model id asked", f.seenInput)
	}
}

// TestResolveCredential_EmptyIDAsksProvider_0056: with no id, the provider
// is asked, as ConfigureLLM asks it.
func TestResolveCredential_EmptyIDAsksProvider_0056(t *testing.T) {
	f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderClaude)}, secrets: []string{"key-1"}})
	res, _, err := ResolveCredential(context.Background(), f, Options{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.seenSelect) != 1 || f.seenSelect[0] != "Choose an LLM provider:" || res.Provider != llmprovider.ProviderClaude {
		t.Errorf("selects %q, provider %q; want the provider menu, then Claude", f.seenSelect, res.Provider)
	}
}

// TestResolveCredential_UnknownID_0056: an id the Registry lacks, or one
// Options.Providers leaves out, matches ErrInvalidProvider, before any prompt.
func TestResolveCredential_UnknownID_0056(t *testing.T) {
	for _, tc := range []struct {
		name string
		o    Options
		id   llmprovider.ProviderID
	}{
		{"not in the registry", Options{}, "no-such-provider"},
		{"not in Options.Providers", Options{Providers: []llmprovider.ProviderID{llmprovider.ProviderGemini}}, llmprovider.ProviderClaude},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake(t, fakePrompter{})
			_, _, err := ResolveCredential(context.Background(), f, tc.o, tc.id)
			if !errors.Is(err, llmprovider.ErrInvalidProvider) {
				t.Errorf("error %v; want ErrInvalidProvider", err)
			}
			if len(f.allText) != 0 {
				t.Errorf("shown %q; want no prompt", f.allText)
			}
		})
	}
}

// TestResolveCredential_EmptyCatalog_0056: Ollama with nothing to list
// returns an empty catalog and no error; the "enter a model id" step is
// ConfigureLLM's.
func TestResolveCredential_EmptyCatalog_0056(t *testing.T) {
	f := newFake(t, fakePrompter{inputs: []string{nettest.RefusedURL}, confirms: []bool{false}})
	res, cat, err := ResolveCredential(context.Background(), f, Options{HTTPClient: nettest.Refusing()},
		llmprovider.ProviderOllama)
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Recommended) != 0 || len(cat.Usable) != 0 || res.Kind != CredNone || res.Model != "" {
		t.Errorf("catalog %+v, Result %+v; want an empty catalog, no credential, no model", cat, res)
	}
	if len(f.seenInput) != 1 {
		t.Errorf("inputs %q; want the endpoint only", f.seenInput)
	}
}

// TestResolveCredential_ErrorKinds_0056 (R52): a Prompter's error matches as
// it is, an ended ctx matches ctx's, and a choice out of range is a plain
// error.
func TestResolveCredential_ErrorKinds_0056(t *testing.T) {
	t.Run("prompter", func(t *testing.T) {
		p := &failOnce0048{fakePrompter: newFake(t, fakePrompter{}), fail: "Secret"}
		if _, _, err := ResolveCredential(context.Background(), p, Options{}, llmprovider.ProviderClaude); !errors.Is(err, errPrompt0048) {
			t.Errorf("error %v; want the Prompter's", err)
		}
	})
	t.Run("ctx", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		f := newFake(t, fakePrompter{})
		if _, _, err := ResolveCredential(ctx, f, Options{}, llmprovider.ProviderClaude); !errors.Is(err, context.Canceled) {
			t.Errorf("error %v; want context.Canceled", err)
		}
	})
	t.Run("ctx ended during the listing", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		f := newFake(t, fakePrompter{secrets: []string{"key-1"}})
		var requests atomic.Int32
		client := claudeRefusals(0, &requests)
		inner := client.Transport
		client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			cancel()
			return inner.RoundTrip(r)
		})
		_, _, err := ResolveCredential(ctx, f, Options{Discover: true, HTTPClient: client,
			ProviderOptions: []llmprovider.Option{testCache(t)}}, llmprovider.ProviderClaude)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("error %v; want context.Canceled, not the static catalog as a checked key", err)
		}
	})
	t.Run("choice out of range", func(t *testing.T) {
		f := newFake(t, fakePrompter{selects: []int{99}})
		_, _, err := ResolveCredential(context.Background(), f, Options{}, "")
		if err == nil || errors.Is(err, llmprovider.ErrInvalidProvider) || errors.Is(err, context.Canceled) {
			t.Errorf("error %v; want a plain error", err)
		}
	})
}
