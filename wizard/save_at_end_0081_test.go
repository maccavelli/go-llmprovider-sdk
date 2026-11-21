package wizard

// 0081-MADR D5 (the owner's Q4 ruling): ConfigureLLM saves a new session as
// the last thing it does, after the model and fallback questions, so a cancel
// or an error at any of them saves nothing; a run that ends well saves once;
// ResolveCredential saves inside its step. Same stand-ins as
// cancel_save_0081_test.go: the login seams, a RoundTripper or an issuer the
// run's HTTPClient serves, a store of the test's own. No outside host, no real
// credential.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// grokStatic0081 is Grok's built-in model count: the model menu offers each,
// then Other.
func grokStatic0081(t *testing.T) int {
	t.Helper()
	d, ok := providers.Default().Descriptor(llmprovider.ProviderGrok)
	if !ok {
		t.Fatal("no Grok descriptor")
	}
	return len(d.StaticModels)
}

// TestCancelAtModelStageSavesNothing_0081 (S14, S16; P-3): ConfigureLLM, a
// Grok device session, a cancel while a model-stage question is on screen,
// answered all the same: no Result, nothing saved, and an error matching the
// context's in the words of the first check that runs after the cancel:
// "wizard: <ctx>" where the question answered was the last (the final check),
// "select model: <ctx>" where the answer leads to a matches' loop,
// "select fallbacks: <ctx>" where a model chosen leads to the fallbacks' loop.
func TestCancelAtModelStageSavesNothing_0081(t *testing.T) {
	for _, tc := range []struct {
		name      string
		cancelAt  string
		inputs    []string
		model     func(other int) []int // the answers from the model menu on
		multi     [][]int
		fallbacks bool
		deadline  bool
		want      string
		target    error
	}{
		{name: "the model search, blank", cancelAt: "Search models", inputs: []string{""},
			model: func(int) []int { return []int{0} }, want: "wizard: context canceled"},
		{name: "the model search, a query with matches", cancelAt: "Search models", inputs: []string{"grok"},
			model: func(int) []int { return nil }, want: "select model: context canceled"},
		{name: "the model menu", cancelAt: "model:", inputs: []string{""},
			model: func(int) []int { return []int{0} }, want: "wizard: context canceled"},
		{name: "the model menu, fallbacks on", cancelAt: "model:", inputs: []string{""},
			model: func(int) []int { return []int{0} }, fallbacks: true, want: "select fallbacks: context canceled"},
		{name: "the model id", cancelAt: "Model id", inputs: []string{"", "my-model"},
			model: func(other int) []int { return []int{other} }, want: "wizard: context canceled"},
		{name: "the fallbacks' menu", cancelAt: "fallback models", inputs: []string{"", ""},
			model: func(int) []int { return []int{0} }, multi: [][]int{{0}}, fallbacks: true, want: "wizard: context canceled"},
		{name: "the run's deadline, the model menu on screen", cancelAt: "", inputs: []string{""},
			model: func(int) []int { return []int{0} }, deadline: true, want: "wizard: context deadline exceeded",
			target: context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			other := grokStatic0081(t)
			stubSessions0081(t, nil)
			ctx, cancel := context.WithCancel(context.Background())
			target := context.Canceled
			if tc.deadline {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), 300*time.Millisecond)
				target = tc.target
			}
			defer cancel()
			store := newCountStore0081()
			o := Options{TokenStore: store, NoEndpointPrompt: true, NeedFallbacks: tc.fallbacks}
			method := methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthDeviceCode, o)
			f := newFake(t, fakePrompter{selects: append([]int{providerIdx(t, llmprovider.ProviderGrok), method}, tc.model(other)...),
				inputs: tc.inputs, multiSelects: tc.multi, allowLeftover: true}) // a cancel may end the run before every answer
			p := newStage0081(f, tc.cancelAt, cancel, store)
			if tc.deadline {
				p.awaitAt, p.awaitCtx = "model:", ctx
			}
			res, err := ConfigureLLM(ctx, p, o)
			if err == nil || !errors.Is(err, target) || err.Error() != tc.want || res.Kind != "" {
				t.Errorf("err %v, kind %q; want %q matching %v and no Result", err, res.Kind, tc.want, target)
			}
			if n := store.count(); n != 0 {
				t.Errorf("a run cancelled at %s saved %d time(s); want none", tc.name, n)
			}
		})
	}
}

// TestCancelAtModelStageReturnsNoKey_0081 (S24): a typed key, a cancel while
// the model menu is on screen, answered all the same: ConfigureLLM returns
// "wizard: context canceled" and no key.
func TestCancelAtModelStageReturnsNoKey_0081(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := newCountStore0081()
	o := Options{NoEndpointPrompt: true}
	method := methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthAPIKey, o)
	f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderGrok), method, 0},
		secrets: []string{"xai-typed-0081"}, inputs: []string{""}})
	res, err := ConfigureLLM(ctx, newStage0081(f, "model:", cancel, store), o)
	wantEnded0081(t, err, context.Canceled)
	if res.APIKey != "" {
		t.Errorf("a key was returned after the cancel; want none")
	}
}

// TestCancelAtManualModelSavesNothing_0081 (S15): a ChatGPT session whose
// listing fails (not a refusal) leaves no catalog, so ConfigureLLM asks for a
// model id; a cancel there saves nothing, and the run returns "wizard:
// context canceled".
func TestCancelAtManualModelSavesNothing_0081(t *testing.T) {
	stubSessions0081(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := newCountStore0081()
	down := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return answer0059(r, http.StatusServiceUnavailable, `{"error":{"message":"down"}}`), nil
	})}
	o := opts0081(t, store, down)
	method := methodIdx0081(t, llmprovider.ProviderOpenAI, llmprovider.AuthBrowserOAuth, o)
	f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderOpenAI), method},
		inputs: []string{"my-model"}})
	res, err := ConfigureLLM(ctx, newStage0081(f, "No models found", cancel, store), o)
	wantEnded0081(t, err, context.Canceled)
	if res.Kind != "" || store.count() != 0 {
		t.Errorf("kind %q, saves %d; want no Result and none saved", res.Kind, store.count())
	}
}

// TestConfigureSavesOnceAfterTheModel_0081 (S21): a ConfigureLLM run that ends
// well saves the new session exactly once, and the store is unwritten while
// the model question is asked.
func TestConfigureSavesOnceAfterTheModel_0081(t *testing.T) {
	stubSessions0081(t, nil)
	store := newCountStore0081()
	o := Options{TokenStore: store, NoEndpointPrompt: true}
	method := methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthDeviceCode, o)
	f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderGrok), method, 0}, inputs: []string{""}})
	p := newStage0081(f, "", nil, store)
	res, err := ConfigureLLM(context.Background(), p, o)
	if err != nil || res.Kind != CredOAuth || res.Model == "" {
		t.Fatalf("err %v, kind %q, model %q; want a configured session", err, res.Kind, res.Model)
	}
	if n := store.count(); n != 1 || p.savesAtModel != 0 {
		t.Errorf("saves %d, saves when the model was asked %d; want one, after the model", n, p.savesAtModel)
	}
}

// TestModelStageErrorSavesNothing_0081 (S17; P-4): an error at the model stage
// that is not the context's ends ConfigureLLM with that error and saves
// nothing: a Prompter's error at the model search, no model entered in the
// empty-catalog branch, and a Prompter's error at the fallbacks.
func TestModelStageErrorSavesNothing_0081(t *testing.T) {
	t.Run("the model search", func(t *testing.T) {
		stubSessions0081(t, nil)
		store := newCountStore0081()
		o := Options{TokenStore: store, NoEndpointPrompt: true}
		method := methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthDeviceCode, o)
		// No scripted Input: the model search returns the fake's error.
		f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderGrok), method}})
		_, err := ConfigureLLM(context.Background(), f, o)
		if err == nil || !strings.Contains(err.Error(), "search models") {
			t.Fatalf("err %v; want the model search's error", err)
		}
		if n := store.count(); n != 0 {
			t.Errorf("a run that failed at the model question saved %d time(s); want none", n)
		}
	})
	t.Run("no model entered", func(t *testing.T) {
		stubSessions0081(t, nil)
		store := newCountStore0081()
		down := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return answer0059(r, http.StatusServiceUnavailable, `{"error":{"message":"down"}}`), nil
		})}
		o := opts0081(t, store, down)
		method := methodIdx0081(t, llmprovider.ProviderOpenAI, llmprovider.AuthBrowserOAuth, o)
		f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderOpenAI), method}, inputs: []string{"  "}})
		_, err := ConfigureLLM(context.Background(), f, o)
		if err == nil || !strings.Contains(err.Error(), "none entered") {
			t.Fatalf("err %v; want the blank model id's error", err)
		}
		if n := store.count(); n != 0 {
			t.Errorf("a run that ended on a blank model id saved %d time(s); want none", n)
		}
	})
	t.Run("the fallbacks", func(t *testing.T) {
		stubSessions0081(t, nil)
		store := newCountStore0081()
		o := Options{TokenStore: store, NoEndpointPrompt: true, NeedFallbacks: true}
		method := methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthDeviceCode, o)
		// The model is chosen; the fallbacks' search has no scripted Input.
		f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderGrok), method, 0}, inputs: []string{""}})
		_, err := ConfigureLLM(context.Background(), f, o)
		if err == nil || !strings.Contains(err.Error(), "search fallback models") {
			t.Fatalf("err %v; want the fallbacks' search error", err)
		}
		if n := store.count(); n != 0 {
			t.Errorf("a run that failed at the fallbacks saved %d time(s); want none", n)
		}
	})
}

// TestResolveSavesInTheStep_0081 (S21): ResolveCredential saves once, in its
// step, and asks no model question.
func TestResolveSavesInTheStep_0081(t *testing.T) {
	stubSessions0081(t, nil)
	store := newCountStore0081()
	o := Options{TokenStore: store, NoEndpointPrompt: true}
	f := newFake(t, fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthDeviceCode, o)}})
	res, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderGrok)
	if err != nil || res.Kind != CredOAuth || store.count() != 1 || len(f.seenInput) != 0 {
		t.Errorf("err %v, kind %q, saves %d, inputs %q; want one save and no model question",
			err, res.Kind, store.count(), f.seenInput)
	}
}

// TestRotationDuringQuestionsSavedAtEnd_0081 (S23): a refresh the listing
// started is still running when the model questions start (the listing's own
// limit ran out first); the store is unwritten during the questions, and the
// save at the end waits for the refresh and holds the rotated pair, not the
// spent one.
func TestRotationDuringQuestionsSavedAtEnd_0081(t *testing.T) {
	release := make(chan struct{})
	after := time.AfterFunc(400*time.Millisecond, func() { close(release) })
	t.Cleanup(func() { after.Stop() })
	store := newCountStore0081()
	client := client0059(http.StatusOK, http.StatusOK, release)
	session := browserGrok0059(t, client, true)
	o := Options{TokenStore: store, Discover: true, DiscoverLimit: 100 * time.Millisecond, HTTPClient: client,
		NoEndpointPrompt: true, ProviderOptions: []llmprovider.Option{llmprovider.WithoutModelMetadata(), testCache(t)}}
	method := methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthBrowserOAuth, o)
	f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderGrok), method, 0}, inputs: []string{""}})
	p := newStage0081(f, "", nil, store)
	_, err := ConfigureLLM(context.Background(), p, o)
	if *session != nil {
		_ = (*session).Wait(context.Background())
	}
	if err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	refreshes := append([]string(nil), store.refreshes...)
	store.mu.Unlock()
	if len(refreshes) != 1 || refreshes[0] != "rt-new" || p.savesAtModel != 0 {
		t.Errorf("saved refresh tokens %q, saves when the model was asked %d; want one save, of rt-new, after the model",
			refreshes, p.savesAtModel)
	}
}

// TestSessionDueAtTheCommit_0081 (S22; P-5): a Grok device session that falls
// due while the model question is on screen is refreshed by the save: the
// refresh answered gives one save of the rotated pair and the Result; refused
// by the issuer, an error matching llmprovider.ErrAuthFailure and no save;
// failed on each of the SDK's three tries with the run alive (a 503, a 429, a
// transport error), the session saved as it was, its refresh token unspent,
// and the Result.
func TestSessionDueAtTheCommit_0081(t *testing.T) {
	for _, tc := range []struct {
		name        string
		status      int
		body        string
		wantSaves   int
		wantRefresh string
		wantAuth    bool
		transport   bool // the request itself fails, as a reset connection does
	}{
		{name: "the refresh answered", status: http.StatusOK,
			body: `{"access_token":"at-new","refresh_token":"rt-new","expires_in":3600}`, wantSaves: 1, wantRefresh: "rt-new"},
		{name: "the refresh refused", status: http.StatusBadRequest, body: `{"error":"invalid_grant"}`, wantAuth: true},
		{name: "the refresh failing, three tries", status: http.StatusServiceUnavailable, body: `{"error":"unavailable"}`,
			wantSaves: 1, wantRefresh: "oauth-refresh"},
		{name: "the refresh rate limited, three tries", status: http.StatusTooManyRequests, body: `{"error":"rate_limited"}`,
			wantSaves: 1, wantRefresh: "oauth-refresh"},
		{name: "the refresh failing in transport, three tries", transport: true,
			wantSaves: 1, wantRefresh: "oauth-refresh"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stubSessions0081(t, nil)
			var tries atomic.Int32
			issuer := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if strings.HasSuffix(r.URL.Path, "/oauth/token") {
					tries.Add(1)
					if tc.transport {
						return nil, errors.New("connection reset")
					}
					return answer0059(r, tc.status, tc.body), nil
				}
				return answer0059(r, http.StatusNotFound, `{}`), nil
			})}
			var session *auth.OAuthSession
			deviceLogin = func(_ context.Context, p llmprovider.ProviderID, _ auth.OAuthFlowOptions, _ func(DeviceCode) error) (*auth.OAuthSession, error) {
				session = testOAuthSession(p)
				session.TokenURL, session.HTTPClient = "https://issuer.test/oauth/token", issuer
				return session, nil
			}
			store := newCountStore0081()
			o := Options{TokenStore: store, NoEndpointPrompt: true}
			method := methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthDeviceCode, o)
			f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderGrok), method, 0}, inputs: []string{""}})
			p := newStage0081(f, "", nil, store)
			p.onModel = func() { session.Invalidate() } // the access token falls due while the question is on screen
			res, err := ConfigureLLM(context.Background(), p, o)
			if tc.wantAuth {
				if !errors.Is(err, llmprovider.ErrAuthFailure) || res.Kind != "" {
					t.Errorf("err %v, kind %q; want an error matching ErrAuthFailure and no Result", err, res.Kind)
				}
			} else if err != nil || res.Kind != CredOAuth {
				t.Errorf("err %v, kind %q; want the Result", err, res.Kind)
			}
			store.mu.Lock()
			saves, refreshes := store.saves, slices.Clone(store.refreshes)
			store.mu.Unlock()
			if saves != tc.wantSaves || (tc.wantSaves == 1 && refreshes[0] != tc.wantRefresh) {
				t.Errorf("saves %d of %q; want %d, of %q", saves, refreshes, tc.wantSaves, tc.wantRefresh)
			}
			if tries.Load() == 0 {
				t.Errorf("the issuer saw no refresh; want the save to have asked for a token")
			}
		})
	}
}

// TestConfigureWaitsForTheLockAfterTheModel_0081 (S35; P-10): ConfigureLLM's
// commit meets a lock another process holds: the model and fallback
// questions come first, then the waiting notice and, at the store's bound, the
// retry question; declined, the store's error, no save and no Result.
func TestConfigureWaitsForTheLockAfterTheModel_0081(t *testing.T) {
	stubSessions0081(t, nil)
	store := &scriptedLock0082{held: true, bounds: []time.Duration{20 * time.Millisecond}}
	o := Options{TokenStore: store, NoEndpointPrompt: true, NeedFallbacks: true}
	method := methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthDeviceCode, o)
	f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderGrok), method, 0},
		inputs: []string{"", ""}, multiSelects: [][]int{{0}}, confirms: []bool{false}})
	p := newStage0081(f, "", nil, nil)
	res, err := ConfigureLLM(context.Background(), p, o)
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) || res.Kind != "" {
		t.Errorf("err %v, kind %q; want the store's error and no Result", err, res.Kind)
	}
	if got := store.got(); slices.Contains(got, "save") {
		t.Errorf("store events %q; want no save", got)
	}
	events := p.log()
	at := func(prefix string) int {
		return slices.IndexFunc(events, func(e string) bool { return strings.HasPrefix(e, prefix) })
	}
	fallbacks, waiting, retry := at("multiselect: Choose fallback"), at("notice: waiting for another process"), at("confirm: Another process is still refreshing")
	if fallbacks < 0 || waiting < fallbacks || retry < waiting {
		t.Errorf("events %q; want the questions, then the waiting notice, then the retry question", events)
	}
}

// failStore0081 is a store whose Save fails: before it writes, or after.
type failStore0081 struct {
	*countStore0081
	after bool // the failure follows the write
	err   error
}

func (s failStore0081) Save(ctx context.Context, p llmprovider.ProviderID, session *auth.OAuthSession) error {
	if s.after {
		if err := s.countStore0081.Save(ctx, p, session); err != nil {
			return err
		}
	}
	return s.err
}

// TestStoreErrorAtTheCommit_0081 (S52, S53; P-12): a store whose Save fails
// before it writes gives the store's error, no Result, and the store as it was;
// one that writes and then fails gives its error and has the new session. For
// ConfigureLLM after the model questions, and for ResolveCredential.
func TestStoreErrorAtTheCommit_0081(t *testing.T) {
	for _, entry := range entries0081 {
		for _, after := range []bool{false, true} {
			name := fmt.Sprintf("write fails after the write %t/%s", after, entry)
			t.Run(name, func(t *testing.T) {
				stubSessions0081(t, nil)
				boom := errors.New("the disk is full")
				counts := newCountStore0081()
				o := Options{TokenStore: failStore0081{counts, after, boom}, NoEndpointPrompt: true}
				f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthDeviceCode, o)}}
				if entry == entryConfigure0081 {
					f.selects, f.inputs = append(f.selects, 0), []string{""}
				}
				res, _, err := run0081(t, context.Background(), entry, llmprovider.ProviderGrok, o, f, nil)
				if !errors.Is(err, boom) || res.Kind != "" {
					t.Errorf("err %v, kind %q; want the store's error and no Result", err, res.Kind)
				}
				if got, want := counts.count(), map[bool]int{false: 0, true: 1}[after]; got != want {
					t.Errorf("saves %d; want %d", got, want)
				}
				if held := counts.heldSession(llmprovider.ProviderGrok); (held != nil) != after {
					t.Errorf("the store holds %v; want a session only when the write completed", held)
				}
			})
		}
	}
}

// TestListingLimitIsAnOutageAndSaves_0081 (S38, S71; P-13): the listing's own
// limit (Options.DiscoverLimit) passes with the run's context alive: an outage.
// The warning is shown, the session saved once, and the Result returned, in the
// step for ResolveCredential and at the end for ConfigureLLM.
func TestListingLimitIsAnOutageAndSaves_0081(t *testing.T) {
	for _, entry := range entries0081 {
		t.Run(entry, func(t *testing.T) {
			stubSessions0081(t, nil)
			store := newCountStore0081()
			o := opts0081(t, store, waitingClient0081())
			o.DiscoverLimit = 50 * time.Millisecond
			f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthDeviceCode, o)}}
			if entry == entryConfigure0081 {
				f.selects, f.inputs = append(f.selects, 0), []string{""}
			}
			res, fp, err := run0081(t, context.Background(), entry, llmprovider.ProviderGrok, o, f, nil)
			if err != nil || res.Kind != CredOAuth {
				t.Errorf("err %v, kind %q; want the Result", err, res.Kind)
			}
			if store.count() != 1 {
				t.Errorf("saves %d; want one", store.count())
			}
			if countContaining(fp.seenNotify, "(timed out)") != 1 {
				t.Errorf("notices %q; want the listing's warning once", fp.seenNotify)
			}
		})
	}
}

// TestAbandonedPlanRunKeepsTheClientRecord_0081 (S48; P-15): over a
// FileTokenStore, a first plan sign-in abandoned at the model question saves no
// session and leaves the record naming the issued client; with an undecodable
// record and a stored plan session of client X, one notice, a new client Y and
// the run abandoned leave the store holding X and the record Y, and a second
// run's plan sign-in is made with no Previous and the consent asked again.
func TestAbandonedPlanRunKeepsTheClientRecord_0081(t *testing.T) {
	ctx := context.Background()
	plan := func(t *testing.T, store auth.TokenStore, client string, previous **auth.OAuthSession) *int {
		t.Helper()
		return stubPlanLogin(t, func(_ context.Context, opts auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
			if previous != nil {
				*previous = opts.Previous
			}
			return planSessionFor(client), nil
		})
	}
	// abandon runs ConfigureLLM to the model question, which has no answer.
	abandon := func(t *testing.T, store auth.TokenStore) (*fakePrompter, error) {
		t.Helper()
		o := Options{TokenStore: store, AgentName: planTestApp, NoEndpointPrompt: true, NoPastePrompt: true}
		f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderOpenAI),
			methodIdx0081(t, llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan, o), 0}})
		_, err := ConfigureLLM(ctx, f, o)
		return f, err
	}
	t.Run("a first sign-in", func(t *testing.T) {
		store := newClientStore(t)
		plan(t, store, "oaiapp_first_0081", nil)
		_, err := abandon(t, store)
		if err == nil || !strings.Contains(err.Error(), "enter model") {
			t.Fatalf("err %v; want the run abandoned at the model question", err)
		}
		session, loadErr := store.Load(ctx, llmprovider.ProviderOpenAI)
		record, recErr := store.LoadClient(ctx, llmprovider.ProviderOpenAI)
		if session != nil || loadErr != nil || record.ClientID != "oaiapp_first_0081" || recErr != nil {
			t.Errorf("session %v (%v), record %+v (%v); want no session and the record naming the issued client", session, loadErr, record, recErr)
		}
	})
	t.Run("an undecodable record and a stored plan session", func(t *testing.T) {
		store := newClientStore(t)
		if err := store.Save(ctx, llmprovider.ProviderOpenAI, planSessionFor("oaiapp_x_0081")); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(store.Dir, "openai.client"), []byte("{{ junk"), 0o600); err != nil {
			t.Fatal(err)
		}
		var previous *auth.OAuthSession
		plan(t, store, "oaiapp_y_0081", &previous)
		f, err := abandon(t, store)
		if err == nil || !strings.Contains(err.Error(), "enter model") {
			t.Fatalf("err %v; want the run abandoned at the model question", err)
		}
		if n := clientNotices0082(f); n != 1 || previous != nil {
			t.Errorf("client notices %d, Previous %v; want one notice and no Previous", n, previous)
		}
		session, loadErr := store.Load(ctx, llmprovider.ProviderOpenAI)
		record, recErr := store.LoadClient(ctx, llmprovider.ProviderOpenAI)
		if loadErr != nil || session == nil || session.ClientID != "oaiapp_x_0081" || recErr != nil || record.ClientID != "oaiapp_y_0081" {
			t.Errorf("session %v (%v), record %+v (%v); want the stored session of X and the record naming Y", session, loadErr, record, recErr)
		}
		// A second run: the stored session is another client's, so no Previous
		// and the consent is asked again.
		given := &auth.OAuthSession{}
		plan(t, store, "oaiapp_y_0081", &given)
		o := Options{TokenStore: store, AgentName: planTestApp, NoEndpointPrompt: true, NoPastePrompt: true}
		second := newFake(t, fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan, o), 0}})
		if _, _, err := ResolveCredential(ctx, second, o, llmprovider.ProviderOpenAI); err != nil {
			t.Fatalf("the second run: %v", err)
		}
		if given != nil || !slices.ContainsFunc(second.seenSelect, func(s string) bool { return strings.Contains(s, "You're using your ChatGPT plan") }) {
			t.Errorf("Previous %v, selects %q; want no Previous and the consent asked", given, second.seenSelect)
		}
	})
}

// helperStore0081 and helperSignal0081 tell the re-run test binary of
// TestKilledAtTheModelQuestionKeepsTheStore_0081 where its store is and where
// to write its signal. They are this test's own: nothing else reads them.
const (
	helperStore0081  = "SDK_TEST_0081_STORE"
	helperSignal0081 = "SDK_TEST_0081_SIGNAL"
)

// blocksAtModel0081 answers like fakePrompter, but at the model question it
// writes the signal file and never returns.
type blocksAtModel0081 struct {
	*fakePrompter
	signal string
}

func (p blocksAtModel0081) Input(prompt, def string) (string, error) {
	if prompt == searchModelsPrompt {
		if err := os.WriteFile(p.signal, []byte("asked"), 0o600); err != nil {
			return "", err
		}
		time.Sleep(time.Hour)
	}
	return p.fakePrompter.Input(prompt, def)
}

// TestKilledAtTheModelQuestionKeepsTheStore_0081 (S41; P-16): a process killed
// while ConfigureLLM's model question is on screen has saved nothing: its
// FileTokenStore still holds the earlier session. The test re-runs its own
// binary as the process and kills it with Process.Kill, which every platform
// has.
func TestKilledAtTheModelQuestionKeepsTheStore_0081(t *testing.T) {
	if dir := os.Getenv(helperStore0081); dir != "" {
		// The re-run process: sign in through the stand-in and block at the
		// model question, to be killed.
		stubSessions0081(t, nil)
		store, err := auth.NewFileTokenStore(dir)
		if err != nil {
			t.Fatal(err)
		}
		o := Options{TokenStore: store, NoEndpointPrompt: true}
		f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderGrok),
			methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthDeviceCode, o)}, allowLeftover: true})
		_, _ = ConfigureLLM(context.Background(), blocksAtModel0081{f, os.Getenv(helperSignal0081)}, o)
		return
	}
	ctx := context.Background()
	store := newClientStore(t)
	earlier := testOAuthSession(llmprovider.ProviderGrok)
	earlier.Access, earlier.Refresh = "earlier-access-0081", "earlier-refresh-0081"
	if err := store.Save(ctx, llmprovider.ProviderGrok, earlier); err != nil {
		t.Fatal(err)
	}
	signal := filepath.Join(t.TempDir(), "asked")
	cmd := exec.Command(os.Args[0], "-test.run=^TestKilledAtTheModelQuestionKeepsTheStore_0081$", "-test.count=1")
	cmd.Env = append(os.Environ(), helperStore0081+"="+store.Dir, helperSignal0081+"="+signal)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	killed := false
	t.Cleanup(func() {
		if !killed {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	asked := false
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if _, err := os.Stat(signal); err == nil {
			asked = true
			break
		}
	}
	if !asked {
		t.Fatalf("the process never reached the model question; its output: %s", out.String())
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	killed = true
	_ = cmd.Wait()
	got, err := store.Load(ctx, llmprovider.ProviderGrok)
	if err != nil || got == nil || got.Access != "earlier-access-0081" || got.Refresh != "earlier-refresh-0081" {
		t.Errorf("the store holds %+v (%v) after the kill; want the earlier session", got, err)
	}
}

// TestLastCommitWins_0081 (S50): two ConfigureLLM runs on one store; run A
// signs in and waits at its model question while run B completes and saves; A
// then answers and commits. The store holds A's session, saved twice in all:
// the last commit wins.
func TestLastCommitWins_0081(t *testing.T) {
	stubSessions0081(t, nil)
	var logins atomic.Int32
	deviceLogin = func(_ context.Context, p llmprovider.ProviderID, _ auth.OAuthFlowOptions, _ func(DeviceCode) error) (*auth.OAuthSession, error) {
		session := testOAuthSession(p)
		if logins.Add(1) == 1 {
			session.Access, session.Refresh = "access-A-0081", "rt-A-0081"
		} else {
			session.Access, session.Refresh = "access-B-0081", "rt-B-0081"
		}
		return session, nil
	}
	store := newCountStore0081()
	o := Options{TokenStore: store, NoEndpointPrompt: true}
	script := func(t *testing.T) *fakePrompter {
		return newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderGrok),
			methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthDeviceCode, o), 0}, inputs: []string{""}})
	}
	atModel, release := make(chan struct{}), make(chan struct{})
	a := newStage0081(script(t), "", nil, store)
	a.onModel = func() {
		close(atModel)
		<-release
	}
	var wg sync.WaitGroup
	var errA error
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, errA = ConfigureLLM(context.Background(), a, o)
	}()
	select {
	case <-atModel:
	case <-time.After(10 * time.Second):
		t.Fatal("run A never reached its model question")
	}
	if _, err := ConfigureLLM(context.Background(), script(t), o); err != nil {
		t.Fatalf("run B: %v", err)
	}
	close(release)
	wg.Wait()
	if errA != nil {
		t.Fatalf("run A: %v", errA)
	}
	store.mu.Lock()
	refreshes := slices.Clone(store.refreshes)
	store.mu.Unlock()
	if !slices.Equal(refreshes, []string{"rt-B-0081", "rt-A-0081"}) {
		t.Errorf("saved refresh tokens %q; want B's, then A's, A being the last to commit", refreshes)
	}
	if held := store.heldSession(llmprovider.ProviderGrok); held == nil || held.Refresh != "rt-A-0081" {
		t.Errorf("the store holds %v; want A's session", held)
	}
}

// TestSaveNotesInTheArchitecture_0081 (S60, S62; P-20): the architecture note's
// "When a session is saved" says, for Kilo, that an abandoned device sign-in
// leaves a token that does not expire, which the person removes in the Kilo
// account's settings; and that the earlier grant was measured kept, naming the
// two device sign-ins as not probed.
func TestSaveNotesInTheArchitecture_0081(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "docs", "architecture.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, rest, found := strings.Cut(string(raw), "- **When a session is saved.**")
	if !found {
		t.Fatal(`docs/architecture.md has no "When a session is saved" bullet`)
	}
	// The bullet runs to the next bullet or the next blank-line-separated block.
	bullet, _, _ := strings.Cut(rest, "\n- **")
	bullet, _, _ = strings.Cut(bullet, "\n\n")
	bullet = strings.Join(strings.Fields(bullet), " ")
	for _, want := range []struct{ name, text string }{
		{"the Kilo token does not expire", "Kilo device sign-in leaves a token that does not expire"},
		{"the person removes it in the account's settings", "Kilo account's settings"},
		{"the premise was measured kept", "kept"},
		{"the measurement's date", "2026-11-19"},
		{"the device sign-ins were not probed", "OpenAI and Grok device sign-ins were not probed"},
	} {
		if !strings.Contains(bullet, want.text) {
			t.Errorf(`"When a session is saved" does not say %s (%q); it reads: %s`, want.name, want.text, bullet)
		}
	}
}
