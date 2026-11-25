package wizard

// 0081-MADR: a run whose context ends before the wizard saves a new session
// saves nothing and returns the context's error, for every provider, session
// kind and key, through ConfigureLLM and ResolveCredential (D2 to D5). Every
// request is answered in process: the sign-ins are the wizard's own seams
// (loginBrowserOAuth, deviceLogin, loginChatGPTPlan, kiloProfile) or an issuer
// the run's HTTPClient serves, and a listing is a RoundTripper that ends the
// run's context when its request arrives. No outside host, no real credential.
//
// A test that asserts an error's words compares the whole text, "wizard:
// <ctx>" (the check right after the step, or commitCredential's) or "the new
// <label> session was not saved: <ctx>" (save and write), since write's error
// also begins with "wizard:". Without the words, a plant on one check is
// hidden by the next.

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// countStore0081 is a TokenStore that counts its saves, under a lock: the
// listing runs on other goroutines. It holds the session it was given as it
// was at the save.
type countStore0081 struct {
	mu        sync.Mutex
	held      map[llmprovider.ProviderID]*auth.OAuthSession
	saves     int
	refreshes []string // each save's refresh token, as it was saved
}

func newCountStore0081() *countStore0081 {
	return &countStore0081{held: map[llmprovider.ProviderID]*auth.OAuthSession{}}
}

func (s *countStore0081) Load(_ context.Context, p llmprovider.ProviderID) (*auth.OAuthSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.held[p], nil
}

func (s *countStore0081) Save(_ context.Context, p llmprovider.ProviderID, session *auth.OAuthSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.held[p] = session
	s.saves++
	s.refreshes = append(s.refreshes, session.Refresh)
	return nil
}

func (s *countStore0081) Delete(_ context.Context, p llmprovider.ProviderID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.held, p)
	return nil
}

func (s *countStore0081) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saves
}

func (s *countStore0081) heldSession(p llmprovider.ProviderID) *auth.OAuthSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.held[p]
}

// wantEnded0081 fails t unless err is the check's error for a context that
// ended with target: "wizard: " and the context's own words, matching target.
func wantEnded0081(t *testing.T, err, target error) {
	t.Helper()
	want := "wizard: " + target.Error()
	if err == nil || !errors.Is(err, target) || err.Error() != want {
		t.Errorf("err %v; want %q matching %v", err, want, target)
	}
}

// wantNotSaved0081 fails t unless err is save's (or write's) error for a
// context that ended with target.
func wantNotSaved0081(t *testing.T, err, target error, label string) {
	t.Helper()
	want := "wizard: the new " + label + " session was not saved: " + target.Error()
	if err == nil || !errors.Is(err, target) || err.Error() != want {
		t.Errorf("err %v; want %q matching %v", err, want, target)
	}
}

// cancellingClient0081 ends the run (cancel) when a request arrives, then
// fails it with the request's context error, as a listing cut off by Ctrl-C
// does.
func cancellingClient0081(cancel context.CancelFunc) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		cancel()
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}
}

// waitingClient0081 fails a request with its context's error once that ends,
// as a listing that is cut off by the run's deadline does.
func waitingClient0081() *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}
}

// methodIdx0081 is method's index in id's sign-in menu under o.
func methodIdx0081(t *testing.T, id llmprovider.ProviderID, m llmprovider.AuthMethodID, o Options) int {
	t.Helper()
	d, ok := providers.Default().Descriptor(id)
	if !ok {
		t.Fatalf("no descriptor %q", id)
	}
	for i, offered := range offeredAuthMethods(d, o) {
		if offered.ID == m {
			return i
		}
	}
	t.Fatalf("%q does not offer %q", id, m)
	return -1
}

// stubSessions0081 replaces the browser, device and plan sign-ins and Kilo's
// profile with in-process stand-ins that issue a current session. cancelIn,
// when set, ends the run inside the login, after the session exists and
// before the wizard checks it.
func stubSessions0081(t *testing.T, cancelIn context.CancelFunc) {
	t.Helper()
	b, dv, kp, pl := loginBrowserOAuth, deviceLogin, kiloProfile, loginChatGPTPlan
	t.Cleanup(func() { loginBrowserOAuth, deviceLogin, kiloProfile, loginChatGPTPlan = b, dv, kp, pl })
	end := func() {
		if cancelIn != nil {
			cancelIn()
		}
	}
	loginBrowserOAuth = func(_ context.Context, p llmprovider.ProviderID, _ auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		end()
		return testOAuthSession(p), nil
	}
	deviceLogin = func(_ context.Context, p llmprovider.ProviderID, _ auth.OAuthFlowOptions, _ func(DeviceCode) error) (*auth.OAuthSession, error) {
		end()
		if p == llmprovider.ProviderKilo {
			return &auth.OAuthSession{Provider: p, Access: kiloWizardToken}, nil
		}
		return testOAuthSession(p), nil
	}
	loginChatGPTPlan = func(context.Context, auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		end()
		return planSessionFor("oaiapp_0081"), nil
	}
	kiloProfile = func(context.Context, string, ...llmprovider.Option) (auth.KiloAccount, error) {
		return auth.KiloAccount{HasPersonalAccount: true}, nil
	}
}

// Both entry points, by name.
const (
	entryResolve0081   = "ResolveCredential"
	entryConfigure0081 = "ConfigureLLM"
)

var entries0081 = []string{entryResolve0081, entryConfigure0081}

// run0081 runs entry for id, scripting the provider menu's answer first for
// ConfigureLLM, and returns the Result, the Prompter it ran with and the
// error. A ConfigureLLM run may end before every scripted answer is used, as
// a cancel does, so leftovers are allowed for it; wrap, when set, wraps the
// scripted Prompter.
func run0081(t *testing.T, ctx context.Context, entry string, id llmprovider.ProviderID, o Options, f fakePrompter,
	wrap func(*fakePrompter) Prompter) (Result, *fakePrompter, error) {
	t.Helper()
	var p Prompter
	var fp *fakePrompter
	if entry == entryConfigure0081 {
		idx := 0 // a Registry of the test's own holds the one provider
		if o.Registry == nil {
			idx = providerIdx(t, id)
		}
		f.selects = append([]int{idx}, f.selects...)
		f.allowLeftover = true
		fp = newFake(t, f)
	} else {
		fp = newFake(t, f)
	}
	p = fp
	if wrap != nil {
		p = wrap(fp)
	}
	if entry == entryConfigure0081 {
		res, err := ConfigureLLM(ctx, p, o)
		return res, fp, err
	}
	res, _, err := ResolveCredential(ctx, p, o, id)
	return res, fp, err
}

// opts0081 are the options of a run that lists through client, with the
// metadata off and a cache of its own.
func opts0081(t *testing.T, store auth.TokenStore, client *http.Client) Options {
	t.Helper()
	return Options{TokenStore: store, Discover: true, NoEndpointPrompt: true, HTTPClient: client,
		ProviderOptions: []llmprovider.Option{testCache(t)}}
}

// issuerKey0081 signs the in-process issuer's ID tokens: one key for the whole
// test binary, since auth keeps an issuer's keys for a while by URL.
var issuerKey0081 = sync.OnceValue(func() *ecdsa.PrivateKey {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	return key
})

func b64url0081(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// idToken0081 is an ID token the real auth flow accepts for Grok's default
// issuer and client: ES256, signed with issuerKey0081.
func idToken0081(t *testing.T) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "ES256", "typ": "JWT", "kid": "k-0081"})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := json.Marshal(map[string]any{"iss": auth.DefaultGrokOAuthIssuer, "aud": auth.DefaultGrokOAuthClientID,
		"sub": "user-0081", "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix()})
	if err != nil {
		t.Fatal(err)
	}
	input := b64url0081(header) + "." + b64url0081(claims)
	digest := sha256.Sum256([]byte(input))
	r, s, err := ecdsa.Sign(rand.Reader, issuerKey0081(), digest[:])
	if err != nil {
		t.Fatal(err)
	}
	sig := append(r.FillBytes(make([]byte, 32)), s.FillBytes(make([]byte, 32))...)
	return input + "." + b64url0081(sig)
}

// grokIssuer0081 is Grok's issuer, answered in process by the run's
// HTTPClient: discovery, its keys, the device-code and token endpoints. A
// request to any other URL is the listing's, answered by other.
type grokIssuer0081 struct {
	t       *testing.T
	token   func(*http.Request) (*http.Response, error) // nil: approve at once
	other   func(*http.Request) (*http.Response, error)
	devices atomic.Int32
	tokens  atomic.Int32
}

func (g *grokIssuer0081) client() *http.Client { return &http.Client{Transport: g} }

func (g *grokIssuer0081) json(r *http.Request, body any) (*http.Response, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		g.t.Fatal(err)
	}
	return answer0059(r, http.StatusOK, string(raw)), nil
}

func (g *grokIssuer0081) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host != "auth.x.ai" {
		return g.other(r)
	}
	switch r.URL.Path {
	case "/.well-known/openid-configuration":
		return g.json(r, map[string]any{"issuer": auth.DefaultGrokOAuthIssuer,
			"jwks_uri":                              "https://auth.x.ai/.well-known/jwks.json",
			"device_authorization_endpoint":         "https://auth.x.ai/oauth2/device/code",
			"token_endpoint":                        "https://auth.x.ai/oauth2/token",
			"id_token_signing_alg_values_supported": []string{"ES256"}})
	case "/.well-known/jwks.json":
		key := issuerKey0081()
		return g.json(r, map[string]any{"keys": []map[string]string{{"kty": "EC", "crv": "P-256", "kid": "k-0081", "alg": "ES256", "use": "sig",
			"x": b64url0081(key.X.FillBytes(make([]byte, 32))), "y": b64url0081(key.Y.FillBytes(make([]byte, 32)))}}})
	case "/oauth2/device/code":
		g.devices.Add(1)
		return g.json(r, map[string]any{"device_code": "device-secret-0081", "user_code": "ABCD-EFGH",
			"verification_uri": "https://auth.x.ai/activate", "expires_in": 60, "interval": 1})
	case "/oauth2/token":
		g.tokens.Add(1)
		if g.token != nil {
			return g.token(r)
		}
		return g.json(r, map[string]any{"access_token": "grok-access-0081", "refresh_token": "grok-refresh-0081",
			"expires_in": 3600, "id_token": idToken0081(g.t)})
	}
	return answer0059(r, http.StatusNotFound, `{}`), nil
}

// sessionCase0081 is one session kind of the listing cases.
type sessionCase0081 struct {
	name     string
	provider llmprovider.ProviderID
	method   llmprovider.AuthMethodID
	secrets  []string // a pasted credential
	plan     bool     // the plan's consent is scripted, allowed
}

var sessionCases0081 = []sessionCase0081{
	{name: "kilo device session", provider: llmprovider.ProviderKilo, method: llmprovider.AuthDeviceCode},
	{name: "grok device session", provider: llmprovider.ProviderGrok, method: llmprovider.AuthDeviceCode},
	{name: "grok browser session", provider: llmprovider.ProviderGrok, method: llmprovider.AuthBrowserOAuth},
	{name: "openai browser session", provider: llmprovider.ProviderOpenAI, method: llmprovider.AuthBrowserOAuth},
	{name: "openai device session", provider: llmprovider.ProviderOpenAI, method: llmprovider.AuthDeviceCode},
	{name: "openai pasted access token", provider: llmprovider.ProviderOpenAI, method: llmprovider.AuthTokenStdin,
		secrets: []string{jwtShapedAccess}},
	{name: "chatgpt plan session", provider: llmprovider.ProviderOpenAI, method: llmprovider.AuthChatGPTPlan, plan: true},
}

// TestCancelDuringListingSavesNothing_0081 (S11, S28): the run's context ends
// while the listing checks a new session, of each kind; nothing is saved, and
// the run returns "wizard: context canceled" through both entry points, asking
// no model question.
func TestCancelDuringListingSavesNothing_0081(t *testing.T) {
	for _, tc := range sessionCases0081 {
		for _, entry := range entries0081 {
			t.Run(tc.name+"/"+entry, func(t *testing.T) {
				stubSessions0081(t, nil)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				store := newCountStore0081()
				o := opts0081(t, store, cancellingClient0081(cancel))
				o.AgentName = planTestApp
				o.NoBrowserQuestion = true
				selects := []int{methodIdx0081(t, tc.provider, tc.method, o)}
				if tc.plan {
					selects = append(selects, 0)
				}
				_, f, err := run0081(t, ctx, entry, tc.provider, o, fakePrompter{selects: selects, secrets: slices.Clone(tc.secrets)}, nil)
				wantEnded0081(t, err, context.Canceled)
				if n := store.count(); n != 0 {
					t.Errorf("a sign-in cancelled during its listing saved %d time(s); want none", n)
				}
				if len(f.seenInput) != 0 {
					t.Errorf("model questions %q asked; want none", f.seenInput)
				}
			})
		}
	}
}

// TestCancelDuringDeviceFlowSavesNothing_0081 (S11): Grok's device sign-in
// through the real auth device flow, not the wizard's seam: its issuer is
// answered in process by the run's HTTPClient (discovery, keys, the device
// code, one poll, an ID token the flow verifies), and the context ends during
// the listing. Nothing is saved; the run returns "wizard: context canceled".
// The flow's poll interval is one second, so each run waits for it.
func TestCancelDuringDeviceFlowSavesNothing_0081(t *testing.T) {
	for _, entry := range entries0081 {
		t.Run(entry, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			issuer := &grokIssuer0081{t: t, other: func(r *http.Request) (*http.Response, error) {
				cancel()
				<-r.Context().Done()
				return nil, r.Context().Err()
			}}
			store := newCountStore0081()
			o := opts0081(t, store, issuer.client())
			method := methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthDeviceCode, o)
			_, f, err := run0081(t, ctx, entry, llmprovider.ProviderGrok, o, fakePrompter{selects: []int{method}}, nil)
			if issuer.devices.Load() != 1 || issuer.tokens.Load() != 1 {
				t.Fatalf("device requests %d, token requests %d; want the real flow's one each (err %v)", issuer.devices.Load(), issuer.tokens.Load(), err)
			}
			wantEnded0081(t, err, context.Canceled)
			if n := store.count(); n != 0 {
				t.Errorf("a device sign-in cancelled during its listing saved %d time(s); want none", n)
			}
			if len(f.seenInput) != 0 {
				t.Errorf("model questions %q asked; want none", f.seenInput)
			}
		})
	}
}

// TestCancelAfterIssueSavesNothing_0081 (S7, S13, S31, S32): the context ends
// after the provider issued the session and before the wizard checks it, with
// and without a listing, and with Options.Driver set, which is given the
// sign-in's done step first: nothing is saved, the run returns "wizard:
// context canceled", and ConfigureLLM asks no model question.
func TestCancelAfterIssueSavesNothing_0081(t *testing.T) {
	for _, discover := range []bool{true, false} {
		for _, driver := range []bool{false, true} {
			for _, entry := range entries0081 {
				name := fmt.Sprintf("discover %t, driver %t/%s", discover, driver, entry)
				t.Run(name, func(t *testing.T) {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					stubSessions0081(t, cancel)
					store := newCountStore0081()
					o := opts0081(t, store, cancellingClient0081(cancel))
					o.Discover = discover
					o.NoBrowserQuestion = true
					var d *recordingDriver0074
					if driver {
						d = &recordingDriver0074{}
						o.Driver = d
					}
					f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthBrowserOAuth, o)}}
					_, fp, err := run0081(t, ctx, entry, llmprovider.ProviderGrok, o, f, nil)
					wantEnded0081(t, err, context.Canceled)
					if n := store.count(); n != 0 {
						t.Errorf("a sign-in cancelled after the issue saved %d time(s); want none", n)
					}
					if len(fp.seenInput) != 0 {
						t.Errorf("model questions %q asked; want none", fp.seenInput)
					}
					if driver && !slices.Contains(d.kinds(), authcontract.StepDone) {
						t.Errorf("the Driver was given %v; want the done step among them", d.kinds())
					}
				})
			}
		}
	}
}

// stage0081 wraps the scripted Prompter. It records, in order, each question
// asked and each notice shown; it ends the run once it has answered the
// prompt whose text holds cancelAt (the person pressed Ctrl-C while it was on
// screen, and the Prompter still returned its answer); and it records how many
// saves the store held when the first model question came.
type stage0081 struct {
	*fakePrompter
	cancelAt     string
	cancel       context.CancelFunc
	store        *countStore0081
	savesAtModel int // -1 until the model question is asked
	onModel      func()
	onAsk        func(kind, text string) // before the answer
	// awaitAt, when it holds in an answered prompt's text, makes the answer
	// wait for awaitCtx to end first: a deadline that passes while the
	// question is on screen.
	awaitAt  string
	awaitCtx context.Context

	mu     sync.Mutex
	events []string
}

func newStage0081(f *fakePrompter, cancelAt string, cancel context.CancelFunc, store *countStore0081) *stage0081 {
	return &stage0081{fakePrompter: f, cancelAt: cancelAt, cancel: cancel, store: store, savesAtModel: -1}
}

func (p *stage0081) log() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.events)
}

func (p *stage0081) ask(kind, text string) {
	p.mu.Lock()
	p.events = append(p.events, kind+": "+text)
	model := p.savesAtModel < 0 && kind == "input" && (text == searchModelsPrompt || strings.HasPrefix(text, "No models found"))
	if model {
		p.savesAtModel = 0
		if p.store != nil {
			p.savesAtModel = p.store.count()
		}
	}
	hook, onModel := p.onAsk, p.onModel
	p.mu.Unlock()
	if model && onModel != nil {
		onModel()
	}
	if hook != nil {
		hook(kind, text)
	}
}

func (p *stage0081) answered(text string) {
	if p.awaitAt != "" && p.awaitCtx != nil && strings.Contains(text, p.awaitAt) {
		<-p.awaitCtx.Done()
	}
	if p.cancelAt != "" && p.cancel != nil && strings.Contains(text, p.cancelAt) {
		p.cancel()
	}
}

func (p *stage0081) Input(prompt, def string) (string, error) {
	p.ask("input", prompt)
	v, err := p.fakePrompter.Input(prompt, def)
	p.answered(prompt)
	return v, err
}

func (p *stage0081) Secret(prompt string) (string, error) {
	p.ask("secret", prompt)
	v, err := p.fakePrompter.Secret(prompt)
	p.answered(prompt)
	return v, err
}

func (p *stage0081) Confirm(question string, def bool) (bool, error) {
	p.ask("confirm", question)
	v, err := p.fakePrompter.Confirm(question, def)
	p.answered(question)
	return v, err
}

func (p *stage0081) Select(title string, choices []Choice, def int) (int, error) {
	p.ask("select", title)
	v, err := p.fakePrompter.Select(title, choices, def)
	p.answered(title)
	return v, err
}

func (p *stage0081) MultiSelect(title string, choices []Choice, pre []int) ([]int, error) {
	p.ask("multiselect", title)
	v, err := p.fakePrompter.MultiSelect(title, choices, pre)
	p.answered(title)
	return v, err
}

func (p *stage0081) Notify(level Level, format string, args ...any) {
	p.mu.Lock()
	p.events = append(p.events, "notice: "+fmt.Sprintf(format, args...))
	p.mu.Unlock()
	p.fakePrompter.Notify(level, format, args...)
}

// endingAt0081 is a wrap for run0081: a stage Prompter that ends the run after
// answering the prompt holding cancelAt.
func endingAt0081(cancelAt string, cancel context.CancelFunc) func(*fakePrompter) Prompter {
	return func(f *fakePrompter) Prompter { return newStage0081(f, cancelAt, cancel, nil) }
}

// noZeroResult0081 fails t unless res is the zero Result.
func noZeroResult0081(t *testing.T, res Result) {
	t.Helper()
	if !reflect.DeepEqual(res, Result{}) {
		t.Errorf("Result %+v; want none", res)
	}
}

// noNotice0081 fails t if a notice holds any of the words.
func noNotice0081(t *testing.T, f *fakePrompter, words ...string) {
	t.Helper()
	for _, n := range f.seenNotify {
		for _, w := range words {
			if strings.Contains(n, w) {
				t.Errorf("notice %q shown; want no notice with %q once the run's context has ended", n, w)
			}
		}
	}
}

// TestCancelDuringKiloProfileSavesNothing_0081 (S8, S10, S73): Kilo's device
// session, without Discover, its profile read for the organization menu: the
// run's context ends during the read (cancelled; its deadline passing), or
// while the organization menu is on screen, answered all the same. Nothing is
// saved, the run returns "wizard: <ctx>", and no organizations warning is
// shown for an ended run.
func TestCancelDuringKiloProfileSavesNothing_0081(t *testing.T) {
	const orgWarning = "could not read the Kilo account's organizations"
	for _, entry := range entries0081 {
		for _, tc := range []struct {
			name   string
			target error
			start  func() (context.Context, context.CancelFunc)
			read   func(cancel context.CancelFunc) func(context.Context, string, ...llmprovider.Option) (auth.KiloAccount, error)
			menu   bool
		}{
			{name: "cancelled during the read", target: context.Canceled,
				start: func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) },
				read: func(cancel context.CancelFunc) func(context.Context, string, ...llmprovider.Option) (auth.KiloAccount, error) {
					return func(c context.Context, _ string, _ ...llmprovider.Option) (auth.KiloAccount, error) {
						cancel()
						return auth.KiloAccount{}, c.Err()
					}
				}},
			{name: "deadline during the read", target: context.DeadlineExceeded,
				start: func() (context.Context, context.CancelFunc) {
					return context.WithTimeout(context.Background(), 300*time.Millisecond)
				},
				read: func(context.CancelFunc) func(context.Context, string, ...llmprovider.Option) (auth.KiloAccount, error) {
					return func(c context.Context, _ string, _ ...llmprovider.Option) (auth.KiloAccount, error) {
						<-c.Done()
						return auth.KiloAccount{}, c.Err()
					}
				}},
			{name: "cancelled with the organization menu on screen", target: context.Canceled, menu: true,
				start: func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) },
				read: func(context.CancelFunc) func(context.Context, string, ...llmprovider.Option) (auth.KiloAccount, error) {
					return func(context.Context, string, ...llmprovider.Option) (auth.KiloAccount, error) {
						return auth.KiloAccount{HasPersonalAccount: true,
							Organizations: []auth.KiloOrganization{{ID: "org-0081", Name: "Acme", Role: "member"}}}, nil
					}
				}},
		} {
			t.Run(tc.name+"/"+entry, func(t *testing.T) {
				stubSessions0081(t, nil)
				ctx, cancel := tc.start()
				defer cancel()
				kiloProfile = tc.read(cancel)
				store := newCountStore0081()
				o := Options{TokenStore: store, NoEndpointPrompt: true}
				f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderKilo, llmprovider.AuthDeviceCode, o)}}
				var wrap func(*fakePrompter) Prompter
				if tc.menu {
					f.selects = append(f.selects, 1) // the organization
					wrap = endingAt0081("Use Kilo as:", cancel)
				}
				res, fp, err := run0081(t, ctx, entry, llmprovider.ProviderKilo, o, f, wrap)
				wantEnded0081(t, err, tc.target)
				noZeroResult0081(t, res)
				if n := store.count(); n != 0 {
					t.Errorf("a sign-in ended during the profile read saved %d time(s); want none", n)
				}
				noNotice0081(t, fp, orgWarning)
				if len(fp.seenInput) != 0 {
					t.Errorf("model questions %q asked; want none", fp.seenInput)
				}
			})
		}
	}
}

// TestDeadlineDuringListingSavesNothing_0081 (S37, S72): the run's deadline,
// not a cancel, passes during the listing; nothing is saved and the run
// returns "wizard: context deadline exceeded". The listing's own limit is left
// at its default, far beyond the run's.
func TestDeadlineDuringListingSavesNothing_0081(t *testing.T) {
	for _, entry := range entries0081 {
		t.Run(entry, func(t *testing.T) {
			stubSessions0081(t, nil)
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			store := newCountStore0081()
			o := opts0081(t, store, waitingClient0081())
			f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthDeviceCode, o)}}
			_, fp, err := run0081(t, ctx, entry, llmprovider.ProviderGrok, o, f, nil)
			wantEnded0081(t, err, context.DeadlineExceeded)
			if n := store.count(); n != 0 {
				t.Errorf("a sign-in whose deadline passed during its listing saved %d time(s); want none", n)
			}
			if len(fp.seenInput) != 0 {
				t.Errorf("model questions %q asked; want none", fp.seenInput)
			}
		})
	}
}

// TestDeadlineReturnsNoKey_0081 (S37, S72; P-14): the run's deadline passes
// during the listing of a typed key: an error matching
// context.DeadlineExceeded, in the words "wizard: context deadline exceeded",
// and no key; ConfigureLLM asks no model.
func TestDeadlineReturnsNoKey_0081(t *testing.T) {
	for _, entry := range entries0081 {
		t.Run(entry, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			o := opts0081(t, nil, waitingClient0081())
			f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthAPIKey, o)}, secrets: []string{"xai-typed-0081"}}
			res, fp, err := run0081(t, ctx, entry, llmprovider.ProviderGrok, o, f, nil)
			wantEnded0081(t, err, context.DeadlineExceeded)
			if res.APIKey != "" {
				t.Errorf("a key was returned after the run's deadline; want none")
			}
			if len(fp.seenInput) != 0 {
				t.Errorf("model questions %q asked; want none", fp.seenInput)
			}
		})
	}
}

// TestCancelDuringListingReturnsNoKey_0081 (S24): a typed key whose listing is
// cancelled is not returned, through both entry points, in the words "wizard:
// context canceled". The wizard writes no key itself.
func TestCancelDuringListingReturnsNoKey_0081(t *testing.T) {
	for _, entry := range entries0081 {
		t.Run(entry, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			o := opts0081(t, nil, cancellingClient0081(cancel))
			f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthAPIKey, o)}, secrets: []string{"xai-typed-0081"}}
			res, fp, err := run0081(t, ctx, entry, llmprovider.ProviderGrok, o, f, nil)
			wantEnded0081(t, err, context.Canceled)
			if res.APIKey != "" {
				t.Errorf("a key was returned after the run's context ended; want none")
			}
			if len(fp.seenInput) != 0 {
				t.Errorf("model questions %q asked; want none", fp.seenInput)
			}
		})
	}
}

// TestCancelDuringRefreshSavesNothing_0081 (S12; K5): a session whose refresh
// the listing left in flight when its own limit ran out. The save waits for
// that refresh, and the run's context ends while it waits: nothing is saved,
// the error is save's ("the new Grok (xAI) session was not saved: context
// canceled"), and it comes within a second of the cancel, not when the issuer
// finally answers, five seconds on.
func TestCancelDuringRefreshSavesNothing_0081(t *testing.T) {
	for _, entry := range entries0081 {
		t.Run(entry, func(t *testing.T) {
			stubSessions0081(t, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			arrived := make(chan struct{}, 1)
			release := make(chan struct{})
			issuer := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if !strings.HasSuffix(r.URL.Path, "/oauth/token") {
					return answer0059(r, http.StatusNotFound, `{}`), nil
				}
				select {
				case arrived <- struct{}{}:
				default:
				}
				select {
				case <-release:
				case <-time.After(5 * time.Second):
				}
				return answer0059(r, http.StatusServiceUnavailable, `{"error":"unavailable"}`), nil
			})}
			var session *auth.OAuthSession
			loginBrowserOAuth = func(_ context.Context, p llmprovider.ProviderID, _ auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
				session = testOAuthSession(p)
				session.Expiry = time.Now().Add(-time.Minute)
				session.TokenURL, session.HTTPClient = "https://issuer.test/oauth/token", issuer
				return session, nil
			}
			store := newCountStore0081()
			o := opts0081(t, store, issuer)
			o.DiscoverLimit = 50 * time.Millisecond
			o.NoBrowserQuestion = true
			f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthBrowserOAuth, o)}}
			if entry == entryConfigure0081 {
				f.selects, f.inputs = append(f.selects, 0), []string{""}
			}
			var cancelledAt atomic.Int64
			go func() {
				<-arrived
				time.Sleep(400 * time.Millisecond) // the save is waiting by then, on a loaded host too
				cancelledAt.Store(time.Now().UnixNano())
				cancel()
			}()
			_, _, err := run0081(t, ctx, entry, llmprovider.ProviderGrok, o, f, nil)
			took := time.Since(time.Unix(0, cancelledAt.Load()))
			close(release)
			if session != nil {
				_ = session.Wait(context.Background())
			}
			wantNotSaved0081(t, err, context.Canceled, "Grok (xAI)")
			if took > time.Second {
				t.Errorf("the run returned %v after the cancel; want within one second, not when the issuer answers", took)
			}
			if n := store.count(); n != 0 {
				t.Errorf("saved %d time(s); want none", n)
			}
		})
	}
}

// cancelOnSecret0081 ends the run once it has answered the key prompt: the
// user types the key, then cancels.
type cancelOnSecret0081 struct {
	*fakePrompter
	cancel context.CancelFunc
}

func (p cancelOnSecret0081) Secret(prompt string) (string, error) {
	v, err := p.fakePrompter.Secret(prompt)
	p.cancel()
	return v, err
}

// TestCancelledConfigureReturnsNoKey_0081 (S13, S24; N3): a typed key for a
// provider with no static catalog (a third party's descriptor), the run
// cancelled after the key was typed: ConfigureLLM returns "wizard: context
// canceled" and no key, and asks no model id.
func TestCancelledConfigureReturnsNoKey_0081(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	unavailable := &llmprovider.APIError{Provider: "acme", Status: http.StatusInternalServerError, Message: "down"}
	f := newFake(t, fakePrompter{selects: []int{0}, secrets: []string{"acme-key"},
		inputs: []string{"some-model"}, allowLeftover: true}) // the model id is asked only without N3's check
	res, err := ConfigureLLM(ctx, cancelOnSecret0081{f, cancel},
		Options{Registry: registry0056(t, nil, unavailable), Discover: true})
	wantEnded0081(t, err, context.Canceled)
	if res.APIKey != "" || len(f.seenInput) != 0 {
		t.Errorf("key returned %t, model ids asked %d; want no key and none asked", res.APIKey != "", len(f.seenInput))
	}
}

// TestCancelledRunReturnsNoCredential_0081 (S25 to S27, S31; P-6): a run
// whose context ends before the commit returns "wizard: context canceled", a
// zero Result and, for ConfigureLLM, asks no model question, whatever the
// credential: an environment key, a saved key, Kilo's device key, a vendor
// CLI read-through, no credential, a Claude key at the workspace question, a
// typed key without Discover.
func TestCancelledRunReturnsNoCredential_0081(t *testing.T) {
	grok, ok := providers.Default().Descriptor(llmprovider.ProviderGrok)
	if !ok {
		t.Fatal("no Grok descriptor")
	}
	for _, entry := range entries0081 {
		check := func(t *testing.T, res Result, fp *fakePrompter, err error) {
			t.Helper()
			wantEnded0081(t, err, context.Canceled)
			noZeroResult0081(t, res)
			if len(fp.seenInput) != 0 && entry == entryConfigure0081 && !slices.Contains(fp.seenInput, workspacePrompt) {
				t.Errorf("model questions %q asked; want none", fp.seenInput)
			}
		}
		t.Run("an environment key/"+entry, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			o := Options{NoEndpointPrompt: true, AllowEnv: true, LookupEnv: envOf(map[string]string{grok.EnvVar: "xai-env-0081"})}
			f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthAPIKey, o)}, confirms: []bool{true}}
			res, fp, err := run0081(t, ctx, entry, llmprovider.ProviderGrok, o, f, endingAt0081("from the environment", cancel))
			check(t, res, fp, err)
		})
		t.Run("a saved key/"+entry, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			o := Options{NoEndpointPrompt: true,
				Existing: Result{Provider: llmprovider.ProviderGrok, Kind: CredAPIKey, APIKey: "xai-saved-0081"}}
			f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthAPIKey, o)}, confirms: []bool{true}}
			res, fp, err := run0081(t, ctx, entry, llmprovider.ProviderGrok, o, f, endingAt0081("Keep the existing key", cancel))
			check(t, res, fp, err)
		})
		t.Run("Kilo's device key/"+entry, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			stubSessions0081(t, cancel)
			o := Options{NoEndpointPrompt: true, KiloDeviceKey: true}
			f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderKilo, llmprovider.AuthDeviceCode, o)}}
			res, fp, err := run0081(t, ctx, entry, llmprovider.ProviderKilo, o, f, nil)
			check(t, res, fp, err)
		})
		t.Run("a vendor CLI read-through/"+entry, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			dir := t.TempDir()
			writeGrokCLILogin(t, dir)
			o := Options{NoEndpointPrompt: true, LookupEnv: envOf(map[string]string{"GROK_HOME": dir})}
			f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthImportVendorCLI, o)}, confirms: []bool{true}}
			res, fp, err := run0081(t, ctx, entry, llmprovider.ProviderGrok, o, f, endingAt0081("CLI login", cancel))
			check(t, res, fp, err)
		})
		t.Run("no credential/"+entry, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ender := ender0081{cancel: cancel}
			reg := registry0081(t, false, nil, func() llmprovider.Provider {
				return lister0081{&thirdParty{}, func(ctx context.Context) ([]string, error) { return nil, ender.end(ctx) }}
			})
			o := Options{Registry: reg, Discover: true}
			res, fp, err := run0081(t, ctx, entry, thirdPartyID, o, fakePrompter{}, nil)
			check(t, res, fp, err)
		})
		t.Run("a Claude key at the workspace question/"+entry, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			a := &anthropic0064{usable: []string{"wrkspc_ok"}}
			o := options0064(t, a)
			o.NoEndpointPrompt = true
			f := fakePrompter{secrets: []string{unscopedKey0064}, inputs: []string{"wrkspc_ok"}}
			res, fp, err := run0081(t, ctx, entry, llmprovider.ProviderClaude, o, f, endingAt0081(workspacePrompt, cancel))
			check(t, res, fp, err)
		})
		t.Run("a typed key without Discover/"+entry, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			o := Options{NoEndpointPrompt: true}
			f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthAPIKey, o)}, secrets: []string{"xai-typed-0081"}}
			res, fp, err := run0081(t, ctx, entry, llmprovider.ProviderGrok, o, f, endingAt0081("API key", cancel))
			check(t, res, fp, err)
		})
	}
}

// ender0081 ends a run from inside a stand-in provider's listing, and then
// waits, as a request does, for the run's context to say so: cancel is nil
// for a run whose deadline does it.
type ender0081 struct{ cancel context.CancelFunc }

func (e ender0081) end(ctx context.Context) error {
	if e.cancel != nil {
		e.cancel()
	}
	<-ctx.Done()
	return ctx.Err()
}

// lister0081 is a third party's provider that is a ModelLister.
type lister0081 struct {
	*thirdParty
	list func(context.Context) ([]string, error)
}

func (p lister0081) ListModels(ctx context.Context) ([]string, error) { return p.list(ctx) }

// catalogLister0081 is a third party's provider that is a catalog.Lister.
type catalogLister0081 struct {
	*thirdParty
	list func(context.Context) (catalog.Catalog, error)
}

func (p catalogLister0081) ListCatalog(ctx context.Context, _ ...llmprovider.Option) (catalog.Catalog, error) {
	return p.list(ctx)
}

// registry0081 holds the one third-party provider build makes, whose
// descriptor takes a key (or none), with the static catalog static.
func registry0081(t *testing.T, takesKey bool, static []string, build func() llmprovider.Provider) *llmprovider.Registry {
	t.Helper()
	r := llmprovider.NewRegistry()
	d := llmprovider.Descriptor{ID: thirdPartyID, Label: "Acme", RequiresAPIKey: takesKey, StaticModels: static}
	if err := r.Register(d, func(...llmprovider.Option) (llmprovider.Provider, error) { return build(), nil }); err != nil {
		t.Fatal(err)
	}
	return r
}

// TestCancelWithKeptSessionReturnsNoResult_0081 (S29, S30; P-7): a kept
// session, the run's context ending during the listing: "wizard: context
// canceled", no Result, nothing saved by the wizard; the same with the kept
// session's refresh completing during the listing (its one save, through its
// own store, of the rotated pair; none by the wizard); a kept session the
// listing refuses, a new sign-in and a cancel at the last model question: the
// store keeps the kept session.
func TestCancelWithKeptSessionReturnsNoResult_0081(t *testing.T) {
	kept := func(expired bool) *auth.OAuthSession {
		s := testOAuthSession(llmprovider.ProviderGrok)
		s.Access, s.Refresh = "kept-access-0081", "rt-kept-0081"
		s.TokenURL = "https://issuer.test/oauth/token"
		if expired {
			s.Expiry = time.Now().Add(-time.Hour)
		}
		return s
	}
	existing := Result{Provider: llmprovider.ProviderGrok, Kind: CredOAuth}
	for _, entry := range entries0081 {
		t.Run("the listing cancelled/"+entry, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			store := newCountStore0081()
			store.held[llmprovider.ProviderGrok] = kept(false)
			o := opts0081(t, store, cancellingClient0081(cancel))
			o.Existing = existing
			res, fp, err := run0081(t, ctx, entry, llmprovider.ProviderGrok, o, fakePrompter{confirms: []bool{true}}, nil)
			wantEnded0081(t, err, context.Canceled)
			noZeroResult0081(t, res)
			if n := store.count(); n != 0 {
				t.Errorf("the wizard saved %d time(s); want none", n)
			}
			if len(fp.seenInput) != 0 {
				t.Errorf("model questions %q asked; want none", fp.seenInput)
			}
		})
		t.Run("its refresh completing during the listing/"+entry, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			store := newCountStore0081()
			store.held[llmprovider.ProviderGrok] = kept(true)
			client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if strings.HasSuffix(r.URL.Path, "/oauth/token") {
					return answer0059(r, http.StatusOK, `{"access_token":"at-new","refresh_token":"rt-new","expires_in":3600}`), nil
				}
				cancel()
				<-r.Context().Done()
				return nil, r.Context().Err()
			})}
			o := opts0081(t, store, client)
			o.Existing = existing
			res, fp, err := run0081(t, ctx, entry, llmprovider.ProviderGrok, o, fakePrompter{confirms: []bool{true}}, nil)
			wantEnded0081(t, err, context.Canceled)
			noZeroResult0081(t, res)
			store.mu.Lock()
			saves, refreshes := store.saves, slices.Clone(store.refreshes)
			store.mu.Unlock()
			if saves != 1 || !slices.Equal(refreshes, []string{"rt-new"}) {
				t.Errorf("saves %d of %q; want the kept session's own one, of the rotated pair, and none by the wizard", saves, refreshes)
			}
			if len(fp.seenInput) != 0 {
				t.Errorf("model questions %q asked; want none", fp.seenInput)
			}
		})
	}
	t.Run("refused, a new sign-in, a cancel at the last question/ConfigureLLM", func(t *testing.T) {
		stubSessions0081(t, nil)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		store := newCountStore0081()
		keptSession := kept(false)
		store.held[llmprovider.ProviderGrok] = keptSession
		client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if strings.HasSuffix(r.URL.Path, "/oauth/token") { // the refresh a 401 starts: the issuer ends the grant
				return answer0059(r, http.StatusBadRequest, `{"error":"invalid_grant"}`), nil
			}
			if strings.Contains(r.Header.Get("Authorization"), "kept-access-0081") {
				return answer0059(r, http.StatusUnauthorized, `{"error":{"message":"invalid credentials"}}`), nil
			}
			return answer0059(r, http.StatusOK, `{"data":[{"id":"grok-4.5"}]}`), nil
		})}
		o := opts0081(t, store, client)
		o.Existing = existing
		o.NoBrowserQuestion = true
		f := newFake(t, fakePrompter{confirms: []bool{true}, inputs: []string{""},
			selects: []int{providerIdx(t, llmprovider.ProviderGrok), methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthBrowserOAuth, o), 0}})
		res, err := ConfigureLLM(ctx, newStage0081(f, "model:", cancel, store), o)
		wantEnded0081(t, err, context.Canceled)
		noZeroResult0081(t, res)
		if got := store.heldSession(llmprovider.ProviderGrok); got != keptSession || store.count() != 0 {
			t.Errorf("the store holds %v after %d saves; want the kept session, unchanged", got, store.count())
		}
	})
}

// TestCancelAfterSaveReturnsTheResult_0081 (S19, S20): a cancel that lands
// after the write leaves the session saved, and ResolveCredential returns its
// Result, so that its error means nothing was saved (but for S53's store).
func TestCancelAfterSaveReturnsTheResult_0081(t *testing.T) {
	stubSessions0081(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	counts := newCountStore0081()
	store := cancelOnSave0081{counts, cancel}
	ok := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return answer0059(r, http.StatusOK, `{"data":[{"id":"grok-4.5"}]}`), nil
	})}
	o := opts0081(t, store, ok)
	f := newFake(t, fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthDeviceCode, o)}})
	res, _, err := ResolveCredential(ctx, f, o, llmprovider.ProviderGrok)
	if err != nil || res.Kind != CredOAuth || counts.count() != 1 {
		t.Errorf("err %v, kind %q, saves %d; want the Result of the saved session", err, res.Kind, counts.count())
	}
}

// cancelOnSave0081 is a store whose Save ends the run once it has written: a
// cancel that lands during the write (K7).
type cancelOnSave0081 struct {
	*countStore0081
	cancel context.CancelFunc
}

func (s cancelOnSave0081) Save(ctx context.Context, p llmprovider.ProviderID, session *auth.OAuthSession) error {
	err := s.countStore0081.Save(ctx, p, session)
	s.cancel()
	return err
}

// cancelBeforeWrite0081 is a store whose Save ends the run first and writes
// after: the instant between the last check and the write (K6). Nothing reads
// the context between write's check and Save.
type cancelBeforeWrite0081 struct {
	*countStore0081
	cancel context.CancelFunc
}

func (s cancelBeforeWrite0081) Save(ctx context.Context, p llmprovider.ProviderID, session *auth.OAuthSession) error {
	s.cancel()
	return s.countStore0081.Save(ctx, p, session)
}

// TestCancelBetweenTheCheckAndTheWriteSaves_0081 (S18; P-18): a store whose
// Save cancels the run, then writes. The checks have passed, so the session
// is saved and the run returns its Result, through both entry points.
func TestCancelBetweenTheCheckAndTheWriteSaves_0081(t *testing.T) {
	for _, entry := range entries0081 {
		t.Run(entry, func(t *testing.T) {
			stubSessions0081(t, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			counts := newCountStore0081()
			o := Options{TokenStore: cancelBeforeWrite0081{counts, cancel}, NoEndpointPrompt: true}
			f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthDeviceCode, o)}}
			if entry == entryConfigure0081 {
				f.selects, f.inputs = append(f.selects, 0), []string{""}
			}
			res, _, err := run0081(t, ctx, entry, llmprovider.ProviderGrok, o, f, nil)
			if err != nil || res.Kind != CredOAuth || counts.count() != 1 {
				t.Errorf("err %v, kind %q, saves %d; want the Result of the saved session", err, res.Kind, counts.count())
			}
		})
	}
}

// TestConfigureReturnsTheResultWhenCancelledInTheWrite_0081 (S19, S20; P-11):
// ConfigureLLM with a store whose Save writes, then cancels the run: the
// Result and one save, through the end of the run and through the model-id
// branch.
func TestConfigureReturnsTheResultWhenCancelledInTheWrite_0081(t *testing.T) {
	t.Run("through the end", func(t *testing.T) {
		stubSessions0081(t, nil)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		counts := newCountStore0081()
		o := Options{TokenStore: cancelOnSave0081{counts, cancel}, NoEndpointPrompt: true}
		f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderGrok),
			methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthDeviceCode, o), 0}, inputs: []string{""}})
		res, err := ConfigureLLM(ctx, f, o)
		if err != nil || res.Kind != CredOAuth || res.Model == "" || counts.count() != 1 {
			t.Errorf("err %v, kind %q, model %q, saves %d; want the Result and one save", err, res.Kind, res.Model, counts.count())
		}
	})
	t.Run("through the model-id branch", func(t *testing.T) {
		stubSessions0081(t, nil)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		counts := newCountStore0081()
		down := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return answer0059(r, http.StatusServiceUnavailable, `{"error":{"message":"down"}}`), nil
		})}
		o := opts0081(t, cancelOnSave0081{counts, cancel}, down)
		o.NoBrowserQuestion = true
		f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderOpenAI),
			methodIdx0081(t, llmprovider.ProviderOpenAI, llmprovider.AuthBrowserOAuth, o)}, inputs: []string{"my-model"}})
		res, err := ConfigureLLM(ctx, f, o)
		if err != nil || res.Kind != CredOAuth || res.Model != "my-model" || counts.count() != 1 {
			t.Errorf("err %v, kind %q, model %q, saves %d; want the Result and one save", err, res.Kind, res.Model, counts.count())
		}
	})
}

// TestCancelBeforeIssueSavesNothing_0081 (S3 to S6, S39; P-1): the context
// ends, or the login ends on its own, before the provider issues anything:
// while the browser waits, while the device poll waits, at the plan's consent
// question (answered after the cancel), during the token exchange (the shipped
// auth.LoginBrowserOAuth, a paste completing it, its token endpoint served in
// process and held until the context ends), and the login's own bound with
// the run alive. The run returns the login's own error, matching the
// context's where the context ended; no Result, no save, no client record.
func TestCancelBeforeIssueSavesNothing_0081(t *testing.T) {
	for _, entry := range entries0081 {
		// check is the assertion every case shares.
		check := func(t *testing.T, res Result, fp *fakePrompter, err error, store *countStore0081, wants ...error) {
			t.Helper()
			for _, want := range wants {
				if !errors.Is(err, want) {
					t.Errorf("err %v; want one matching %v", err, want)
				}
			}
			noZeroResult0081(t, res)
			if n := store.count(); n != 0 {
				t.Errorf("a sign-in that issued nothing saved %d time(s); want none", n)
			}
			if len(fp.seenInput) != 0 {
				t.Errorf("model questions %q asked; want none", fp.seenInput)
			}
		}
		t.Run("the browser waits/"+entry, func(t *testing.T) {
			stubSessions0081(t, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			loginBrowserOAuth = func(c context.Context, _ llmprovider.ProviderID, _ auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
				cancel()
				<-c.Done()
				return nil, fmt.Errorf("oauth: browser login stopped: %w", c.Err())
			}
			store := newCountStore0081()
			o := opts0081(t, store, nil)
			o.NoBrowserQuestion = true
			f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthBrowserOAuth, o)}}
			res, fp, err := run0081(t, ctx, entry, llmprovider.ProviderGrok, o, f, nil)
			check(t, res, fp, err, store, context.Canceled)
			if err == nil || !strings.Contains(err.Error(), "oauth: browser login stopped") {
				t.Errorf("err %v; want the login's own error", err)
			}
		})
		t.Run("the device poll waits/"+entry, func(t *testing.T) {
			stubSessions0081(t, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			deviceLogin = func(c context.Context, _ llmprovider.ProviderID, _ auth.OAuthFlowOptions, _ func(DeviceCode) error) (*auth.OAuthSession, error) {
				cancel()
				<-c.Done()
				return nil, fmt.Errorf("oauth: device login stopped: %w", c.Err())
			}
			store := newCountStore0081()
			o := opts0081(t, store, nil)
			f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthDeviceCode, o)}}
			res, fp, err := run0081(t, ctx, entry, llmprovider.ProviderGrok, o, f, nil)
			check(t, res, fp, err, store, context.Canceled)
			if err == nil || !strings.Contains(err.Error(), "oauth: device login stopped") {
				t.Errorf("err %v; want the login's own error", err)
			}
		})
		t.Run("the plan's consent question, answered after the cancel/"+entry, func(t *testing.T) {
			stubSessions0081(t, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			logins := stubPlanLogin(t, func(context.Context, auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
				return planSessionFor("oaiapp_0081"), nil
			})
			files := newClientStore(t)
			o := Options{TokenStore: files, AgentName: planTestApp, NoEndpointPrompt: true, NoPastePrompt: true}
			f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan, o), 0}}
			res, fp, err := run0081(t, ctx, entry, llmprovider.ProviderOpenAI, o, f, endingAt0081("You're using your ChatGPT plan", cancel))
			noZeroResult0081(t, res)
			if !errors.Is(err, context.Canceled) {
				t.Errorf("err %v; want one matching context.Canceled", err)
			}
			if *logins != 0 {
				t.Errorf("%d plan login(s) made after the cancel; want none", *logins)
			}
			session, loadErr := files.Load(context.Background(), llmprovider.ProviderOpenAI)
			record, recErr := files.LoadClient(context.Background(), llmprovider.ProviderOpenAI)
			if session != nil || loadErr != nil || record.ClientID != "" || recErr != nil {
				t.Errorf("the store holds session %v (%v) and client record %+v (%v); want neither", session, loadErr, record, recErr)
			}
			if len(fp.seenInput) != 0 {
				t.Errorf("model questions %q asked; want none", fp.seenInput)
			}
		})
		t.Run("the token exchange cut off/"+entry, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			issuer := &grokIssuer0081{t: t, token: func(r *http.Request) (*http.Response, error) {
				cancel()
				<-r.Context().Done()
				return nil, r.Context().Err()
			}}
			// The shipped login, with no listener: a paste completes it.
			loginBrowserOAuth = func(c context.Context, p llmprovider.ProviderID, flow auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
				flow.NoLoopback = true
				flow.OpenURL = func(string) error { return nil }
				flow.InputCode = func(context.Context) (string, error) { return "code-0081", nil }
				return auth.LoginBrowserOAuth(c, p, flow)
			}
			t.Cleanup(func() { loginBrowserOAuth = auth.LoginBrowserOAuth })
			store := newCountStore0081()
			o := opts0081(t, store, issuer.client())
			o.NoBrowserQuestion = true
			f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthBrowserOAuth, o)}}
			res, fp, err := run0081(t, ctx, entry, llmprovider.ProviderGrok, o, f, nil)
			check(t, res, fp, err, store, context.Canceled)
			if issuer.tokens.Load() != 1 || err == nil || !strings.HasPrefix(err.Error(), "oauth: ") {
				t.Errorf("token requests %d, err %v; want the exchange's one request and the login's own error", issuer.tokens.Load(), err)
			}
		})
		t.Run("the login's own bound, the run alive/"+entry, func(t *testing.T) {
			stubSessions0081(t, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			loginBrowserOAuth = func(context.Context, llmprovider.ProviderID, auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
				return nil, fmt.Errorf("oauth: browser login timed out: %w", context.DeadlineExceeded)
			}
			store := newCountStore0081()
			o := opts0081(t, store, nil)
			o.NoBrowserQuestion = true
			f := fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthBrowserOAuth, o)}}
			res, fp, err := run0081(t, ctx, entry, llmprovider.ProviderGrok, o, f, nil)
			check(t, res, fp, err, store, context.DeadlineExceeded)
			if ctx.Err() != nil || err == nil || !strings.Contains(err.Error(), "oauth: browser login timed out") {
				t.Errorf("run's context %v, err %v; want the run alive and the login's own error", ctx.Err(), err)
			}
		})
	}
}

// listingSite0081 drives one of the listing's six warnings: ended says whether
// the run's context is to end inside the listing, by ender.
type listingSite0081 struct {
	name  string
	setup func(t *testing.T, ender ender0081, ended bool, store *countStore0081) (Options, llmprovider.ProviderID, fakePrompter)
}

var errListingDown0081 = errors.New("listing down")

// thirdSite0081 is the options of a typed key at a third party's provider.
func thirdSite0081(t *testing.T, takesKey bool, static []string, build func() llmprovider.Provider) (Options, llmprovider.ProviderID, fakePrompter) {
	o := Options{Registry: registry0081(t, takesKey, static, build), Discover: true,
		ProviderOptions: []llmprovider.Option{testCache(t)}}
	return o, thirdPartyID, fakePrompter{secrets: []string{"acme-key"}}
}

// transportSite0081 is a listing's transport that ends the run, or fails.
func transportSite0081(ender ender0081, ended bool) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if ended {
			return nil, ender.end(r.Context())
		}
		return nil, errListingDown0081
	})}
}

// listingSites0081 is the six sites of the listing's warnings.
func listingSites0081() []listingSite0081 {
	down := errListingDown0081
	third, transport := thirdSite0081, transportSite0081
	return []listingSite0081{
		{"no static catalog, a listing that failed (:948)", func(t *testing.T, e ender0081, ended bool, store *countStore0081) (Options, llmprovider.ProviderID, fakePrompter) {
			stubSessions0081(t, nil)
			o := opts0081(t, store, transport(e, ended))
			o.NoBrowserQuestion = true
			return o, llmprovider.ProviderOpenAI, fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderOpenAI, llmprovider.AuthBrowserOAuth, o)}}
		}},
		{"a static catalog, a listing that failed (:951)", func(t *testing.T, e ender0081, ended bool, _ *countStore0081) (Options, llmprovider.ProviderID, fakePrompter) {
			return third(t, true, thirdPartyStatic, func() llmprovider.Provider {
				return lister0081{&thirdParty{}, func(ctx context.Context) ([]string, error) {
					if ended {
						return nil, e.end(ctx)
					}
					return nil, down
				}}
			})
		}},
		{"no static catalog, a catalog not live carrying the failure (:958)", func(t *testing.T, e ender0081, ended bool, _ *countStore0081) (Options, llmprovider.ProviderID, fakePrompter) {
			return third(t, true, nil, func() llmprovider.Provider {
				return catalogLister0081{&thirdParty{}, func(ctx context.Context) (catalog.Catalog, error) {
					if ended {
						return catalog.Catalog{Err: e.end(ctx)}, nil
					}
					return catalog.Catalog{Err: down}, nil
				}}
			})
		}},
		{"a static catalog, a catalog not live, nothing recommended (:967)", func(t *testing.T, e ender0081, ended bool, _ *countStore0081) (Options, llmprovider.ProviderID, fakePrompter) {
			return third(t, true, thirdPartyStatic, func() llmprovider.Provider {
				return catalogLister0081{&thirdParty{}, func(ctx context.Context) (catalog.Catalog, error) {
					if ended {
						return catalog.Catalog{Err: e.end(ctx)}, nil
					}
					return catalog.Catalog{Err: down}, nil
				}}
			})
		}},
		{"a built-in catalog beside the failure (:974)", func(t *testing.T, e ender0081, ended bool, _ *countStore0081) (Options, llmprovider.ProviderID, fakePrompter) {
			o := opts0081(t, nil, transport(e, ended))
			return o, llmprovider.ProviderGrok, fakePrompter{selects: []int{methodIdx0081(t, llmprovider.ProviderGrok, llmprovider.AuthAPIKey, o)},
				secrets: []string{"xai-typed-0081"}}
		}},
		{"a catalog not live and no failure (:977)", func(t *testing.T, e ender0081, ended bool, _ *countStore0081) (Options, llmprovider.ProviderID, fakePrompter) {
			return third(t, true, thirdPartyStatic, func() llmprovider.Provider {
				return catalogLister0081{&thirdParty{}, func(ctx context.Context) (catalog.Catalog, error) {
					if ended {
						_ = e.end(ctx)
					}
					return catalog.Catalog{Recommended: []string{"m-1"}, Usable: []string{"m-1"}}, nil
				}}
			})
		}},
	}
}

// TestNoListingWarningOnAnEndedRun_0081 (S61, S71, S72, S77; P-17): the
// listing's warning is not shown once the run's context has ended, by cancel
// or by deadline, at each of the six sites that tell one, through both entry
// points: the run returns "wizard: <ctx>" and nothing is saved. Each site has
// a control, with the run alive and the listing failing as the stand-in does,
// that shows the warning, so the site is known to be reached.
func TestNoListingWarningOnAnEndedRun_0081(t *testing.T) {
	listing := []string{"could not list models", "live model listing"}
	sites := listingSites0081()
	for _, s := range sites {
		t.Run(s.name+"/the run alive", func(t *testing.T) {
			store := newCountStore0081()
			o, id, f := s.setup(t, ender0081{}, false, store)
			_, fp, err := run0081(t, context.Background(), entryResolve0081, id, o, f, nil)
			if err != nil {
				t.Fatalf("err %v; want the run to go on after its listing failed", err)
			}
			if !slices.ContainsFunc(fp.seenNotify, func(n string) bool {
				return strings.Contains(n, listing[0]) || strings.Contains(n, listing[1])
			}) {
				t.Errorf("notices %q; want the listing's warning shown with the run alive", fp.seenNotify)
			}
		})
		for _, entry := range entries0081 {
			for _, deadline := range []bool{false, true} {
				name, target := "cancelled", context.Canceled
				if deadline {
					name, target = "deadline", context.DeadlineExceeded
				}
				t.Run(s.name+"/"+name+"/"+entry, func(t *testing.T) {
					ctx, cancel := context.WithCancel(context.Background())
					ender := ender0081{cancel: cancel}
					if deadline {
						cancel()
						ctx, cancel = context.WithTimeout(context.Background(), 300*time.Millisecond)
						ender = ender0081{}
					}
					defer cancel()
					store := newCountStore0081()
					o, id, f := s.setup(t, ender, true, store)
					res, fp, err := run0081(t, ctx, entry, id, o, f, nil)
					wantEnded0081(t, err, target)
					noZeroResult0081(t, res)
					noNotice0081(t, fp, listing...)
					if n := store.count(); n != 0 {
						t.Errorf("saved %d time(s); want none", n)
					}
					if len(fp.seenInput) != 0 {
						t.Errorf("model questions %q asked; want none", fp.seenInput)
					}
				})
			}
		}
	}
}

// TestKiloUnverifiedNoticeClaimsNoSave_0081 (S67, S68, S76; P-21): Kilo under
// Discover with no organization, the profile answering 503, the run alive:
// the notice reads "could not verify the Kilo key (...); going on without
// verifying it", claims no save, and for ConfigureLLM comes before the model
// question. A typed key and a device session, through both entry points.
func TestKiloUnverifiedNoticeClaimsNoSave_0081(t *testing.T) {
	const words = "could not verify the Kilo key (provider unavailable, HTTP 503); going on without verifying it"
	for _, entry := range entries0081 {
		for _, kind := range []string{"a typed key", "a device session"} {
			t.Run(kind+"/"+entry, func(t *testing.T) {
				k := newKilo0080(kiloGood0080)
				k.outage(http.StatusServiceUnavailable)
				o := k.options(t)
				f := fakePrompter{inputs: []string{acceptDefault}}
				if kind == "a typed key" {
					f.secrets = []string{kiloGood0080}
				} else {
					stubKiloDeviceLogin0080(t)
					o.TokenStore = newCountStore0081()
					f.selects = []int{methodIdx0081(t, llmprovider.ProviderKilo, llmprovider.AuthDeviceCode, o)}
				}
				if entry == entryConfigure0081 {
					f.inputs = append(f.inputs, "")
					f.selects = append(f.selects, 0)
				}
				var stage *stage0081
				_, fp, err := run0081(t, context.Background(), entry, llmprovider.ProviderKilo, o, f, func(fp *fakePrompter) Prompter {
					stage = newStage0081(fp, "", nil, nil)
					return stage
				})
				if err != nil {
					t.Fatalf("err %v; want the run to go on", err)
				}
				if n := countContaining(fp.seenNotify, words); n != 1 {
					t.Errorf("notices %q; want %q once", fp.seenNotify, words)
				}
				for _, n := range fp.seenNotify {
					if strings.Contains(n, "saved") {
						t.Errorf("notice %q claims a save; want none", n)
					}
				}
				if entry == entryConfigure0081 {
					events := stage.log()
					notice := slices.Index(events, "notice: "+words)
					model := slices.IndexFunc(events, func(e string) bool { return strings.HasPrefix(e, "input: "+searchModelsPrompt) })
					if notice < 0 || model < 0 || notice > model {
						t.Errorf("events %q; want the notice before the model question", events)
					}
				}
			})
		}
	}
}
