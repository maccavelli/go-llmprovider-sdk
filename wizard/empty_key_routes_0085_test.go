package wizard

// 0085-MADR: the routes beside the typed prompt, the optional-key server, the
// listing, the SDK's own TextPrompter, and a key the service refused followed
// by blanks. Credentials are made up; every request is answered in process.

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// TestBlankEnvAndSavedKeyAreAbsent_0085 (ST37, ST38, ST41, ST42, ST64): a
// blank environment value and a blank Existing key, `""` among them, are not
// offered to keep (no Confirm); the typed key is used. A Chat Completions
// server with a blank saved key is listed with no key and asks nothing.
func TestBlankEnvAndSavedKeyAreAbsent_0085(t *testing.T) {
	d, _ := providers.Default().Descriptor(llmprovider.ProviderTogether)
	values := []string{""}
	for _, in := range blankInputs0085 {
		values = append(values, in.value)
	}
	for _, v := range values {
		o := options0085(t)
		o.AllowEnv, o.LookupEnv = true, func(string) string { return v }
		// confirms scripted true: a blank value that is offered is kept.
		f := newFake(t, fakePrompter{confirms: []bool{true}, secrets: []string{"made-up-key-1234"}, allowLeftover: true})
		res, _, err := ResolveCredential(context.Background(), f, o, d.ID)
		if err != nil || res.APIKey != "made-up-key-1234" || len(f.seenConfirm) != 0 {
			t.Errorf("environment %q: err %v, key %q, confirms %d; want the typed key and no question", v, err, res.APIKey, len(f.seenConfirm))
		}
		o = options0085(t)
		o.Existing = Result{Provider: d.ID, Kind: CredAPIKey, APIKey: v}
		f = newFake(t, fakePrompter{confirms: []bool{true}, secrets: []string{"made-up-key-1234"}, allowLeftover: true})
		res, _, err = ResolveCredential(context.Background(), f, o, d.ID)
		if err != nil || res.APIKey != "made-up-key-1234" || len(f.seenConfirm) != 0 {
			t.Errorf("saved %q: err %v, key %q, confirms %d; want the typed key and no question", v, err, res.APIKey, len(f.seenConfirm))
		}
	}

	t.Run("optional key, saved blank", func(t *testing.T) {
		srv := newCompatServer0066(t, 2, "")
		f := newFake(t, fakePrompter{confirms: []bool{true}, allowLeftover: true})
		o := compatOptions0066(t)
		o.NoEndpointPrompt = true
		o.Existing = Result{Provider: compatID0066, BaseURL: srv.URL + "/v1", Kind: CredAPIKey, APIKey: "   "}
		res, _, err := ResolveCredential(context.Background(), f, o, compatID0066)
		if err != nil || res.Kind != CredNone || res.APIKey != "" || len(f.seenConfirm) != 0 || len(f.seenSecret) != 0 ||
			!slices.Equal(srv.seen(), []string{""}) {
			t.Errorf("err %v, kind %q, key %q, confirms %d, secrets %d, headers %q; want kind none, no question, one request with no key",
				err, res.Kind, res.APIKey, len(f.seenConfirm), len(f.seenSecret), srv.seen())
		}
	})
}

// TestNonBlankEnvAndSavedKeyUntrimmed_0085 (ST39, ST43): a value that is not
// blank is offered and, kept, returned as it is, with its line feed or
// spaces (MADR D5).
func TestNonBlankEnvAndSavedKeyUntrimmed_0085(t *testing.T) {
	d, _ := providers.Default().Descriptor(llmprovider.ProviderTogether)
	o := options0085(t)
	o.AllowEnv, o.LookupEnv = true, func(string) string { return "made-up-key\n" }
	f := newFake(t, fakePrompter{confirms: []bool{true}})
	res, _, err := ResolveCredential(context.Background(), f, o, d.ID)
	if err != nil || res.APIKey != "made-up-key\n" || len(f.seenConfirm) != 1 {
		t.Errorf("environment: err %v, key %q, confirms %d; want the value as it is after one question", err, res.APIKey, len(f.seenConfirm))
	}
	o = options0085(t)
	o.Existing = Result{Provider: d.ID, Kind: CredAPIKey, APIKey: " made-up-key "}
	f = newFake(t, fakePrompter{confirms: []bool{true}})
	res, _, err = ResolveCredential(context.Background(), f, o, d.ID)
	if err != nil || res.APIKey != " made-up-key " || len(f.seenConfirm) != 1 {
		t.Errorf("saved: err %v, key %q, confirms %d; want the value as it is after one question", err, res.APIKey, len(f.seenConfirm))
	}
}

// countStore0085 is a memory TokenStore that counts its deletes; the memory
// store counts its saves.
type countStore0085 struct {
	*memoryTokenStore
	deletes int
}

func (s *countStore0085) Delete(ctx context.Context, p llmprovider.ProviderID) error {
	s.deletes++
	return s.memoryTokenStore.Delete(ctx, p)
}

// TestBlankAfterDecliningSavedKey_0085 (ST44 to ST46, ST59): a stored key or
// session that the person declines is never a fallback for a blank answer.
// (a) a saved key, declined, three blanks: the no-key error, no Result, three
// asks. (b) declined, a blank, a new key: the new key. (c) a stored session,
// declined, the API key method, three blanks: the error, and the store was
// neither saved to nor deleted from. (d) the environment value, declined,
// three blanks.
func TestBlankAfterDecliningSavedKey_0085(t *testing.T) {
	together, _ := providers.Default().Descriptor(llmprovider.ProviderTogether)
	openai, _ := providers.Default().Descriptor(llmprovider.ProviderOpenAI)
	ends := func(t *testing.T, d llmprovider.Descriptor, o Options, f *fakePrompter) (*foreverBlank0085, Result, error) {
		t.Helper()
		p := &foreverBlank0085{fakePrompter: f}
		res, _, err := ResolveCredential(context.Background(), p, o, d.ID)
		return p, res, err
	}
	check := func(t *testing.T, d llmprovider.Descriptor, p *foreverBlank0085, res Result, err error) {
		t.Helper()
		if err == nil || err.Error() != noKeyError0085(d) || !zero0085(res) || p.asks != maxTypedKeys {
			t.Errorf("err %v, result %+v, asks %d; want %q, no result, %d asks", err, res, p.asks, noKeyError0085(d), maxTypedKeys)
		}
	}
	t.Run("a: saved key", func(t *testing.T) {
		o := options0085(t)
		o.Existing = Result{Provider: together.ID, Kind: CredAPIKey, APIKey: "saved-key-1234"}
		p, res, err := ends(t, together, o, newFake(t, fakePrompter{confirms: []bool{false}}))
		check(t, together, p, res, err)
	})
	t.Run("b: then a new key", func(t *testing.T) {
		o := options0085(t)
		o.Existing = Result{Provider: together.ID, Kind: CredAPIKey, APIKey: "saved-key-1234"}
		f := newFake(t, fakePrompter{confirms: []bool{false}, secrets: []string{"  ", "new-key-1234"}})
		res, _, err := ResolveCredential(context.Background(), f, o, together.ID)
		if err != nil || res.APIKey != "new-key-1234" || len(f.seenSecret) != 2 {
			t.Errorf("err %v, key %q, asks %d; want the new key after two asks", err, res.APIKey, len(f.seenSecret))
		}
	})
	t.Run("c: stored session", func(t *testing.T) {
		store := &countStore0085{memoryTokenStore: newMemoryTokenStore()}
		o := options0085(t)
		o.TokenStore = store
		o.Existing = storedExisting(t, store.memoryTokenStore, Result{Provider: openai.ID, Kind: CredOAuth,
			TokenExpiry: time.Now().Add(time.Hour), Issuer: auth.DefaultOpenAIIssuer, ClientID: auth.DefaultOpenAIClientID,
			AccountID: "acct_test"}, "existing-access-abcd", "existing-refresh")
		saves := store.saves
		p, res, err := ends(t, openai, o, newFake(t, fakePrompter{confirms: []bool{false}, selects: apiKeyMenu0085(t, openai, o)}))
		check(t, openai, p, res, err)
		if kept := store.sessions[openai.ID]; store.saves != saves || store.deletes != 0 || kept == nil || kept.Access != "existing-access-abcd" {
			t.Errorf("saves %d (was %d), deletes %d, kept session %+v; want the store untouched", store.saves, saves, store.deletes, kept)
		}
	})
	t.Run("d: environment value", func(t *testing.T) {
		o := options0085(t)
		o.AllowEnv, o.LookupEnv = true, func(string) string { return "env-key-1234" }
		p, res, err := ends(t, together, o, newFake(t, fakePrompter{confirms: []bool{false}}))
		check(t, together, p, res, err)
	})
}

// TestCompatBlankKeyIsNotListed_0085 (ST28, ST30): an optional-key server that
// wants "lm-key" is asked for it after its listing refuses none; a blank
// answer is never listed with. The server sees no key, then the key.
func TestCompatBlankKeyIsNotListed_0085(t *testing.T) {
	srv := newCompatServer0066(t, 2, "lm-key")
	f := newFake(t, fakePrompter{secrets: []string{"   ", "lm-key"}})
	o := compatOptions0066(t)
	o.NoEndpointPrompt = true
	o.Existing.Provider, o.Existing.BaseURL = compatID0066, srv.URL+"/v1"
	res, _, err := ResolveCredential(context.Background(), f, o, compatID0066)
	if err != nil || res.APIKey != "lm-key" || !slices.Equal(srv.seen(), []string{"", "Bearer lm-key"}) {
		t.Errorf("err %v, key set %v, headers %q; want the key, after a request with none", err, res.APIKey == "lm-key", srv.seen())
	}
}

// TestCompatNoKeyNeeded_0085 (ST31): a server that takes no key is never asked
// for one: no Secret prompt, kind none.
func TestCompatNoKeyNeeded_0085(t *testing.T) {
	srv := newCompatServer0066(t, 2, "")
	f := newFake(t, fakePrompter{})
	o := compatOptions0066(t)
	o.NoEndpointPrompt = true
	o.Existing.Provider, o.Existing.BaseURL = compatID0066, srv.URL+"/v1"
	res, _, err := ResolveCredential(context.Background(), f, o, compatID0066)
	if err != nil || res.Kind != CredNone || len(f.seenSecret) != 0 {
		t.Errorf("err %v, kind %q, secrets asked %d; want none, kind none", err, res.Kind, len(f.seenSecret))
	}
}

// TestTextPrompterRefusesBlank_0085 (ST20, ST21): the package's own prompter
// asks again for a blank entry itself, on a line reader and in raw mode, three
// warnings of its own and none of the wizard's, and returns the key. End of
// input after a blank line is an error matching io.EOF after one warning.
func TestTextPrompterRefusesBlank_0085(t *testing.T) {
	d, _ := providers.Default().Descriptor(llmprovider.ProviderTogether)
	for _, raw := range []bool{false, true} {
		t.Run(map[bool]string{false: "line", true: "raw"}[raw], func(t *testing.T) {
			end := "\n"
			if raw {
				rawSeam0059(t, nil, nil)
				end = "\r"
			}
			typed := "   " + end + "\t" + end + "\u00a0" + end + "made-up-key-1234" + end
			p, out := pipe0059(t, typed)
			res, _, err := ResolveCredential(context.Background(), p, options0085(t), d.ID)
			if err != nil || res.APIKey != "made-up-key-1234" || strings.Count(out.String(), "empty value") != 3 ||
				strings.Contains(out.String(), "no key was entered") {
				t.Errorf("err %v, key %q, warnings %d, output %q; want the key after three TextPrompter warnings and no wizard notice",
					err, res.APIKey, strings.Count(out.String(), "empty value"), out.String())
			}
		})
	}
	t.Run("end of input", func(t *testing.T) {
		p, out := pipe0059(t, "   \n")
		_, _, err := ResolveCredential(context.Background(), p, options0085(t), d.ID)
		if !errors.Is(err, io.EOF) || strings.Count(out.String(), "empty value") != 1 || strings.Contains(out.String(), "no key was entered") {
			t.Errorf("err %v, warnings %d; want an error matching io.EOF after one TextPrompter warning", err, strings.Count(out.String(), "empty value"))
		}
	})
}

// TestBlankKeyNeverReachesTheListing_0085 (ST51): under Discover a blank answer,
// then a key, make one listing request, not two, and no "refused this key"
// notice, for Claude, whose listing refuses a blank key.
func TestBlankKeyNeverReachesTheListing_0085(t *testing.T) {
	var requests atomic.Int32
	o := options0085(t)
	o.Discover, o.DiscoverLimit, o.HTTPClient = true, 2*time.Second, claudeRefusals(0, &requests)
	f := newFake(t, fakePrompter{secrets: []string{"   ", "made-up-key-1234"}})
	res, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderClaude)
	if err != nil || res.APIKey != "made-up-key-1234" || requests.Load() != 1 || countContaining(f.seenNotify, "refused this key") != 0 {
		t.Errorf("err %v, key %q, listing requests %d, notices %q; want the key, one request and no refusal",
			err, res.APIKey, requests.Load(), f.seenNotify)
	}
}

// TestBlankKeyNeverReachesKiloProfile_0085 (ST51, Kilo case): Kilo's listing is
// public and answers a blank key, so a blank answer, then a key, must leave one
// profile read, with the key, and no "refused this key" notice. Its stand-in is
// 0080's, which does not exist before 0080.
func TestBlankKeyNeverReachesKiloProfile_0085(t *testing.T) {
	k := newKilo0080(kiloGood0080)
	o := k.options(t)
	o.NoEndpointPrompt = true
	f := newFake(t, fakePrompter{secrets: []string{"   ", kiloGood0080}})
	res, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
	if err != nil || res.APIKey != kiloGood0080 || !slices.Equal(k.profileTokens(), []string{kiloGood0080}) ||
		countContaining(f.seenNotify, "refused this key") != 0 {
		t.Errorf("err %v, key %q, profile read with %q, notices %q; want the key, one profile read with it and no refusal",
			err, res.APIKey, k.profileTokens(), f.seenNotify)
	}
}

// TestRefusedKeyThenBlanks_0085 (ST15, ST67): a key the service refused,
// followed by three blanks, ends with the no-key error, which does not match
// llmprovider.ErrAuthFailure; the blank count restarts at each call of the
// typed prompt.
func TestRefusedKeyThenBlanks_0085(t *testing.T) {
	notRefusal := func(t *testing.T, err error, want string) {
		t.Helper()
		if err == nil || err.Error() != want || errors.Is(err, llmprovider.ErrAuthFailure) {
			t.Errorf("err %v; want exactly %q, matching no ErrAuthFailure", err, want)
		}
	}
	t.Run("typed key refused", func(t *testing.T) {
		var requests atomic.Int32
		d, _ := providers.Default().Descriptor(llmprovider.ProviderClaude)
		o := options0085(t)
		o.Discover, o.DiscoverLimit, o.HTTPClient = true, 2*time.Second, claudeRefusals(1, &requests)
		f := newFake(t, fakePrompter{secrets: []string{"bad-key-1", "  ", "\t", "  "}})
		_, _, err := ResolveCredential(context.Background(), f, o, d.ID)
		notRefusal(t, err, noKeyError0085(d))
		if requests.Load() != 1 || len(f.seenSecret) != 4 {
			t.Errorf("%d listing requests, %d asks; want 1 and 4", requests.Load(), len(f.seenSecret))
		}
	})
	t.Run("optional key wanted", func(t *testing.T) {
		srv := newCompatServer0066(t, 2, "lm-key")
		f := newFake(t, fakePrompter{secrets: []string{"  ", "  ", "  "}})
		o := compatOptions0066(t)
		o.NoEndpointPrompt = true
		o.Existing.Provider, o.Existing.BaseURL = compatID0066, srv.URL+"/v1"
		_, _, err := ResolveCredential(context.Background(), f, o, compatID0066)
		notRefusal(t, err, "wizard: no LM Studio key entered after 3 tries")
		if !slices.Equal(srv.seen(), []string{""}) {
			t.Errorf("listing headers %q; want one request, with no key", srv.seen())
		}
	})
	t.Run("kept session refused", func(t *testing.T) {
		d, _ := providers.Default().Descriptor(llmprovider.ProviderOpenAI)
		store := newStore0059()
		kept := testOAuthSession(llmprovider.ProviderOpenAI)
		kept.Access, kept.Refresh, kept.Expiry = keptAccess0066, "kept-refresh-0066", time.Now().Add(2*time.Hour)
		store.held[d.ID] = kept
		o := Options{TokenStore: store, Discover: true, HTTPClient: accepting0066(""),
			Providers:       []llmprovider.ProviderID{d.ID},
			Existing:        Result{Provider: d.ID, Kind: CredOAuth},
			ProviderOptions: []llmprovider.Option{testCache(t)}}
		f := newFake(t, fakePrompter{confirms: []bool{true},
			selects: []int{methodIdx0059(t, o, d.ID, llmprovider.AuthAPIKey)}, secrets: []string{"  ", "  ", "  "}})
		_, _, err := ResolveCredential(context.Background(), f, o, d.ID)
		notRefusal(t, err, noKeyError0085(d))
		if countContaining(f.seenNotify, "session was refused") != 1 {
			t.Errorf("notices %q; want one that the saved session was refused", f.seenNotify)
		}
	})
	t.Run("count restarts at each prompt", func(t *testing.T) {
		var requests atomic.Int32
		d, _ := providers.Default().Descriptor(llmprovider.ProviderClaude)
		o := options0085(t)
		o.Discover, o.DiscoverLimit, o.HTTPClient = true, 2*time.Second, claudeRefusals(3, &requests)
		f := newFake(t, fakePrompter{secrets: []string{"bad-key-1", "  ", "  ", "bad-key-2", "  ", "  ", "bad-key-3"}})
		_, _, err := ResolveCredential(context.Background(), f, o, d.ID)
		if !errors.Is(err, llmprovider.ErrAuthFailure) || err == nil || !strings.Contains(err.Error(), "refused 3 keys") ||
			len(f.seenSecret) != 7 || requests.Load() != 3 {
			t.Errorf("err %v, %d asks, %d listing requests; want a refusal of 3 keys after 7 asks and 3 requests",
				err, len(f.seenSecret), requests.Load())
		}
	})
}
