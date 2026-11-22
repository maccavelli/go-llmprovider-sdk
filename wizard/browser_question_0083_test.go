package wizard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// The words the tests assert are written out, not read from the wizard's
// constants, so that a change of a word fails them (0083-MADR D3, D5).
const (
	question0083       = "Where is the browser you will sign in with?"
	here0083           = "On this machine"
	elsewhere0083      = "On another device: I will paste the address back"
	codeToday0083      = "If the browser does not return here, paste the redirected URL or the authorization code"
	urlToday0083       = "If the browser does not return here, paste the redirected URL"
	codeNoListener0083 = "This sign-in does not wait for the browser to come back. After you sign in, paste here the code the page shows or, " +
		"if the browser is sent to a page it cannot open, the whole address from its address bar"
	urlNoListener0083 = "This sign-in does not wait for the browser to come back. After you sign in, the browser cannot open the page it is sent to: " +
		"copy the whole address from its address bar and paste it here"
	cancelledNotice0083 = "Sign-in cancelled; press Enter to continue"
	timedOutNotice0083  = "Sign-in timed out; press Enter to continue"
	finishedNotice0083  = "Browser sign-in finished; press Enter to continue"
	authorizeURL0083    = "https://authorize.test"
)

// logins0083 records what each browser or plan login was given.
type logins0083 struct {
	mu         sync.Mutex
	noLoopback []bool
}

func (l *logins0083) add(noLoopback bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.noLoopback = append(l.noLoopback, noLoopback)
}

func (l *logins0083) got() []bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.noLoopback)
}

// stubLogins0083 replaces the browser and plan logins with ones that record
// their NoLoopback, show the authorize URL and return a session.
func stubLogins0083(t *testing.T) *logins0083 {
	t.Helper()
	l := &logins0083{}
	stubBrowserLogin(t, func(_ context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		l.add(opts.NoLoopback)
		_ = opts.OpenURL(authorizeURL0083)
		return testOAuthSession(provider), nil
	})
	stubPlanLogin(t, func(_ context.Context, opts auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		l.add(opts.NoLoopback)
		_ = opts.OpenURL(authorizeURL0083)
		return planSessionFor("oaiapp_0083"), nil
	})
	return l
}

// scenario0083 is one browser sign-in the wizard runs.
type scenario0083 struct {
	name     string
	provider llmprovider.ProviderID
	plan     bool // the ChatGPT plan sign-in
	again    bool // ... with a plan session in the store: a sign-in again
}

var scenarios0083 = []scenario0083{
	{name: "Grok browser", provider: llmprovider.ProviderGrok},
	{name: "OpenAI browser", provider: llmprovider.ProviderOpenAI},
	{name: "plan first", provider: llmprovider.ProviderOpenAI, plan: true},
	{name: "plan again", provider: llmprovider.ProviderOpenAI, plan: true, again: true},
}

// urlOnly is the first plan sign-in, whose paste must be the whole address.
func (s scenario0083) urlOnly() bool { return s.plan && !s.again }

// today is the instruction with a listener, and noListener without one.
func (s scenario0083) today() string {
	if s.urlOnly() {
		return urlToday0083
	}
	return codeToday0083
}

func (s scenario0083) noListener() string {
	if s.urlOnly() {
		return urlNoListener0083
	}
	return codeNoListener0083
}

// options is o with the store, the app name and the endpoint the scenario needs.
func (s scenario0083) options(o Options) Options {
	store := newMemoryTokenStore()
	if s.again {
		store.sessions[llmprovider.ProviderOpenAI] = planSessionFor("oaiapp_prev")
	}
	o.TokenStore, o.AgentName, o.NoEndpointPrompt = store, planTestApp, true
	return o
}

func (s scenario0083) method(t *testing.T, o Options) int {
	t.Helper()
	switch {
	case s.plan:
		return planMethodIdx(t)
	case s.provider == llmprovider.ProviderGrok:
		return grokBrowserIdx
	}
	return methodIdx0059(t, o, s.provider, llmprovider.AuthBrowserOAuth)
}

type entry0083 string

const (
	viaResolve0083   entry0083 = "ResolveCredential"
	viaConfigure0083 entry0083 = "ConfigureLLM"
)

var entries0083 = []entry0083{viaResolve0083, viaConfigure0083}

// run0083 runs the scenario through entry, the person answering the question
// with ask (none when it is not asked) and the plan's consent "Allow".
func (s scenario0083) run(t *testing.T, entry entry0083, o Options, ask []int) (*fakePrompter, error) {
	t.Helper()
	o = s.options(o)
	var selects []int
	if entry == viaConfigure0083 {
		selects = append(selects, providerIdx(t, s.provider))
	}
	selects = append(selects, s.method(t, o))
	selects = append(selects, ask...)
	if s.plan {
		selects = append(selects, 0)
	}
	var inputs []string
	if entry == viaConfigure0083 {
		if s.provider == llmprovider.ProviderGrok {
			selects = append(selects, 0)
		} else {
			inputs = []string{"my-model"}
		}
	}
	f := newFake(t, fakePrompter{blankSearches: true, selects: selects, inputs: inputs})
	var err error
	if entry == viaConfigure0083 {
		_, err = ConfigureLLM(context.Background(), f, o)
	} else {
		_, _, err = ResolveCredential(context.Background(), f, o, s.provider)
	}
	return f, err
}

func eachScenario0083(t *testing.T, body func(t *testing.T, s scenario0083, entry entry0083)) {
	t.Helper()
	for _, s := range scenarios0083 {
		for _, entry := range entries0083 {
			t.Run(s.name+"/"+string(entry), func(t *testing.T) { body(t, s, entry) })
		}
	}
}

func opener0083(n *int) func(string) error {
	return func(string) error { *n++; return nil }
}

// TestNoLoopbackReachesTheLogin_0083 (ST6, ST19, ST21, ST22): Options.NoLoopback
// reaches the login of OpenAI's and Grok's browser sign-in and of the plan,
// first and again, through both entry points, with no question; D5's words are
// shown, the URL-only ones for a first plan sign-in, and Options.OpenURL is
// called once.
func TestNoLoopbackReachesTheLogin_0083(t *testing.T) {
	eachScenario0083(t, func(t *testing.T, s scenario0083, entry entry0083) {
		logins := stubLogins0083(t)
		opened := 0
		f, err := s.run(t, entry, Options{NoLoopback: true, OpenURL: opener0083(&opened)}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := logins.got(); !slices.Equal(got, []bool{true}) {
			t.Errorf("the login's NoLoopback = %v; want one login with it set", got)
		}
		if slices.Contains(f.seenSelect, question0083) {
			t.Errorf("selects %q; want no question under NoLoopback", f.seenSelect)
		}
		if !slices.Contains(f.seenNotify, s.noListener()) || slices.Contains(f.seenNotify, s.today()) {
			t.Errorf("notices %q; want the no-listener instruction %q, not today's", f.seenNotify, s.noListener())
		}
		if opened != 1 {
			t.Errorf("OpenURL called %d times; want once", opened)
		}
	})
}

// TestNoLoopbackOffByDefault_0083 (ST1, ST2, ST25, ST56): the person answering
// "On this machine", or giving an empty line, gets no NoLoopback at the login
// and today's instruction, and Options.OpenURL is called.
func TestNoLoopbackOffByDefault_0083(t *testing.T) {
	eachScenario0083(t, func(t *testing.T, s scenario0083, entry entry0083) {
		logins := stubLogins0083(t)
		opened := 0
		f, err := s.run(t, entry, Options{OpenURL: opener0083(&opened)}, []int{0})
		if err != nil {
			t.Fatal(err)
		}
		if got := logins.got(); !slices.Equal(got, []bool{false}) {
			t.Errorf("the login's NoLoopback = %v; want one login without it", got)
		}
		if !slices.Contains(f.seenNotify, s.today()) || slices.Contains(f.seenNotify, s.noListener()) {
			t.Errorf("notices %q; want today's instruction %q", f.seenNotify, s.today())
		}
		if opened != 1 {
			t.Errorf("OpenURL called %d times; want once", opened)
		}
	})
	t.Run("an empty line is the default", func(t *testing.T) {
		logins := stubLogins0083(t)
		var out strings.Builder
		tp := &TextPrompter{In: strings.NewReader("2\n\n"), Out: &out}
		_, _, err := ResolveCredential(context.Background(), tp, Options{TokenStore: newMemoryTokenStore(), NoEndpointPrompt: true},
			llmprovider.ProviderGrok)
		if err != nil {
			t.Fatal(err)
		}
		if got := logins.got(); !slices.Equal(got, []bool{false}) {
			t.Errorf("the login's NoLoopback = %v after an empty line; want one login without it", got)
		}
		if n := strings.Count(out.String(), question0083); n != 1 {
			t.Errorf("the question was shown %d times; want once", n)
		}
	})
}

// TestQuestionAnswerElsewhere_0083 (ST5, ST17, ST21, ST55): "On another device"
// gives NoLoopback at the login and D5's words, and Options.OpenURL is not
// called, through both entry points, for each sign-in.
func TestQuestionAnswerElsewhere_0083(t *testing.T) {
	eachScenario0083(t, func(t *testing.T, s scenario0083, entry entry0083) {
		logins := stubLogins0083(t)
		opened := 0
		f, err := s.run(t, entry, Options{OpenURL: opener0083(&opened)}, []int{1})
		if err != nil {
			t.Fatal(err)
		}
		if got := logins.got(); !slices.Equal(got, []bool{true}) {
			t.Errorf("the login's NoLoopback = %v; want one login with it set", got)
		}
		if !slices.Contains(f.seenNotify, s.noListener()) || slices.Contains(f.seenNotify, s.today()) {
			t.Errorf("notices %q; want the no-listener instruction %q", f.seenNotify, s.noListener())
		}
		if countContaining(f.seenNotify, authorizeURL0083) != 1 {
			t.Errorf("notices %q; want the URL shown once", f.seenNotify)
		}
		if opened != 0 {
			t.Errorf("OpenURL called %d times after the answer; want none", opened)
		}
	})
}

// TestQuestionNoLoopbackStillOpens_0083 (ST6, D6): under the caller's
// NoLoopback, which reaches the login, Options.OpenURL is called once.
func TestQuestionNoLoopbackStillOpens_0083(t *testing.T) {
	logins := stubLogins0083(t)
	opened := 0
	s := scenarios0083[0]
	if _, err := s.run(t, viaResolve0083, Options{NoLoopback: true, OpenURL: opener0083(&opened)}, nil); err != nil {
		t.Fatal(err)
	}
	if got := logins.got(); !slices.Equal(got, []bool{true}) {
		t.Errorf("the login's NoLoopback = %v; want one login with it set", got)
	}
	if opened != 1 {
		t.Errorf("OpenURL called %d times under NoLoopback; want once", opened)
	}
}

// TestQuestionForm_0083 (ST2, ST13): one Select, with the title and the two
// choices word for word, the first the default.
func TestQuestionForm_0083(t *testing.T) {
	stubLogins0083(t)
	f, err := scenarios0083[0].run(t, viaResolve0083, Options{}, []int{0})
	if err != nil {
		t.Fatal(err)
	}
	var at []int
	for i, title := range f.seenSelect {
		if title == question0083 {
			at = append(at, i)
		}
	}
	if len(at) != 1 {
		t.Fatalf("selects %q; want the question once", f.seenSelect)
	}
	i := at[0]
	if got := labels(f.seenSelectItems[i]); !slices.Equal(got, []string{here0083, elsewhere0083}) {
		t.Errorf("the question's choices = %q; want %q and %q", got, here0083, elsewhere0083)
	}
	if f.seenSelectDefault[i] != 0 {
		t.Errorf("the question's default = %d; want 0, %q", f.seenSelectDefault[i], here0083)
	}
}

// TestQuestionBeforeConsent_0083 (ST14): the question comes after the method
// menu and before the plan's consent, and for Grok's browser sign-in right
// after the menu.
func TestQuestionBeforeConsent_0083(t *testing.T) {
	stubLogins0083(t)
	f, err := scenarios0083[2].run(t, viaResolve0083, Options{}, []int{0})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.seenSelect) != 3 || !strings.HasPrefix(f.seenSelect[0], "Choose how to authenticate with OpenAI") ||
		f.seenSelect[1] != question0083 || !strings.Contains(f.seenSelect[2], "You're using your ChatGPT plan") {
		t.Errorf("selects %q; want the method menu, the question, the consent", f.seenSelect)
	}
	f, err = scenarios0083[0].run(t, viaResolve0083, Options{}, []int{0})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.seenSelect) != 2 || !strings.HasPrefix(f.seenSelect[0], "Choose how to authenticate with") || f.seenSelect[1] != question0083 {
		t.Errorf("selects %q; want the method menu, then the question", f.seenSelect)
	}
}

// TestQuestionOncePerRun_0083 (ST67): a saved session the listing refuses
// leads to the sign-in menu, and the run's one browser sign-in asks the
// question once, after that menu and never when the kept session is offered.
func TestQuestionOncePerRun_0083(t *testing.T) {
	client := accepting0066("oauth-access")
	store, o := keptGrok0066(t, client)
	browserGrok0059(t, client, false)
	f := newFake(t, fakePrompter{blankSearches: true, confirms: []bool{true},
		selects: []int{0, methodIdx0059(t, o, llmprovider.ProviderGrok, llmprovider.AuthBrowserOAuth), 0, 0}})
	if _, err := ConfigureLLM(context.Background(), f, o); err != nil {
		t.Fatal(err)
	}
	if n := countContaining(f.seenSelect, question0083); n != 1 {
		t.Errorf("selects %q; want the question once", f.seenSelect)
	}
	if len(f.seenSelect) < 3 || f.seenSelect[2] != question0083 || len(f.seenConfirm) != 1 {
		t.Errorf("selects %q, confirms %q; want the keep question, then the provider and method menus, then the question",
			f.seenSelect, f.seenConfirm)
	}
	if saves := store.recorded(); len(saves) != 1 {
		t.Errorf("saves %+v; want the new session, once", saves)
	}
}

// questionBlocks0083 is a Prompter whose Select fails the test, and blocks
// until the test ends, when it is asked the question: a plant that asks it
// where it must not is seen as a failure, not a hang.
type questionBlocks0083 struct {
	*fakePrompter
	release chan struct{}
}

func newQuestionBlocks0083(t *testing.T, f fakePrompter) *questionBlocks0083 {
	t.Helper()
	q := &questionBlocks0083{fakePrompter: newFake(t, f), release: make(chan struct{})}
	t.Cleanup(func() { close(q.release) })
	return q
}

func (q *questionBlocks0083) Select(title string, choices []Choice, defaultIdx int) (int, error) {
	if title == question0083 {
		q.t.Errorf("the question was asked where it must not be: %q", title)
		<-q.release
		return 0, errors.New("the question was asked")
	}
	return q.fakePrompter.Select(title, choices, defaultIdx)
}

// within0083 runs body on a goroutine and fails t when it has not returned in 5 s.
func within0083(t *testing.T, what string, body func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { defer close(done); body() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("%s did not finish within 5 s", what)
	}
}

// TestQuestionNotAskedWhereItMustNot_0083 (ST9 to ST11): under each of
// NoLoopback, NoBrowserQuestion, both, NoPastePrompt and a Driver, the
// question is not asked and the sign-in completes; the login gets NoLoopback
// only under NoLoopback.
func TestQuestionNotAskedWhereItMustNot_0083(t *testing.T) {
	for _, c := range []struct {
		name       string
		set        func(*Options)
		noLoopback bool
	}{
		{"NoLoopback", func(o *Options) { o.NoLoopback = true }, true},
		{"NoBrowserQuestion", func(o *Options) { o.NoBrowserQuestion = true }, false},
		{"NoLoopback with NoBrowserQuestion", func(o *Options) { o.NoLoopback, o.NoBrowserQuestion = true, true }, true},
		{"NoPastePrompt", func(o *Options) { o.NoPastePrompt = true }, false},
		{"a Driver", func(o *Options) { o.Driver = &recordingDriver0074{} }, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			logins := stubLogins0083(t)
			s := scenarios0083[0]
			o := Options{}
			c.set(&o)
			o = s.options(o)
			p := newQuestionBlocks0083(t, fakePrompter{selects: []int{grokBrowserIdx}})
			var err error
			within0083(t, "the sign-in", func() { _, _, err = ResolveCredential(context.Background(), p, o, s.provider) })
			if err != nil {
				t.Fatal(err)
			}
			if got := logins.got(); !slices.Equal(got, []bool{c.noLoopback}) {
				t.Errorf("the login's NoLoopback = %v; want %v", got, c.noLoopback)
			}
		})
	}
}

// TestDriverWithNoLoopback_0083 (ST12): a Driver with NoLoopback gets no
// question and NoLoopback at the login, and its paste completes the sign-in.
func TestDriverWithNoLoopback_0083(t *testing.T) {
	logins := &logins0083{}
	stubBrowserLogin(t, func(ctx context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		logins.add(opts.NoLoopback)
		_ = opts.OpenURL(authorizeURL0083)
		code, err := opts.InputCode(ctx)
		if err != nil || code != "driver-code-0083" {
			t.Errorf("InputCode = %q, %v; want the Driver's paste", code, err)
		}
		return testOAuthSession(provider), nil
	})
	d := &recordingDriver0074{answer: func(_ context.Context, s authcontract.Step) (authcontract.Answer, error) {
		if s.Kind == authcontract.StepPaste {
			return authcontract.Answer{Paste: "driver-code-0083"}, nil
		}
		return authcontract.Answer{}, nil
	}}
	s := scenarios0083[0]
	o := s.options(Options{NoLoopback: true, Driver: d})
	p := newQuestionBlocks0083(t, fakePrompter{selects: []int{grokBrowserIdx}})
	var err error
	within0083(t, "the sign-in", func() { _, _, err = ResolveCredential(context.Background(), p, o, s.provider) })
	if err != nil {
		t.Fatal(err)
	}
	if got := logins.got(); !slices.Equal(got, []bool{true}) {
		t.Errorf("the login's NoLoopback = %v; want it set", got)
	}
	if !slices.Contains(d.kinds(), authcontract.StepPaste) {
		t.Errorf("the Driver was given %v; want a paste step", d.kinds())
	}
}

// TestQuestionNotAskedForOtherMethods_0083 (ST23, ST24): a device sign-in, an
// API key, a pasted token, a vendor import and a kept session ask no question,
// and NoLoopback leaves a device sign-in as it is.
func TestQuestionNotAskedForOtherMethods_0083(t *testing.T) {
	deviceStub := func(t *testing.T) {
		t.Helper()
		original := deviceLogin
		t.Cleanup(func() { deviceLogin = original })
		deviceLogin = func(_ context.Context, provider llmprovider.ProviderID, _ auth.OAuthFlowOptions, show func(DeviceCode) error) (*auth.OAuthSession, error) {
			if err := show(DeviceCode{Provider: provider, URL: "https://verify.test", Code: "ABCD-EFGH"}); err != nil {
				return nil, err
			}
			return testOAuthSession(provider), nil
		}
	}
	for _, c := range []struct {
		name     string
		provider llmprovider.ProviderID
		method   llmprovider.AuthMethodID
		setup    func(t *testing.T, o *Options)
		secrets  []string
		confirms []bool
	}{
		{"Grok device", llmprovider.ProviderGrok, llmprovider.AuthDeviceCode, func(t *testing.T, _ *Options) { deviceStub(t) }, nil, nil},
		{"Grok device with NoLoopback", llmprovider.ProviderGrok, llmprovider.AuthDeviceCode,
			func(t *testing.T, o *Options) { deviceStub(t); o.NoLoopback = true }, nil, nil},
		{"OpenAI device", llmprovider.ProviderOpenAI, llmprovider.AuthDeviceCode, func(t *testing.T, _ *Options) { deviceStub(t) }, nil, nil},
		{"Grok API key", llmprovider.ProviderGrok, llmprovider.AuthAPIKey, nil, []string{"xai-0083"}, nil},
		{"OpenAI pasted token", llmprovider.ProviderOpenAI, llmprovider.AuthTokenStdin, nil, []string{"sk-0083-pasted-key"}, nil},
		{"Grok vendor import", llmprovider.ProviderGrok, llmprovider.AuthImportVendorCLI, func(t *testing.T, o *Options) {
			dir := t.TempDir()
			writeGrokCLILogin(t, dir)
			o.LookupEnv = func(name string) string {
				if name == "GROK_HOME" {
					return dir
				}
				return ""
			}
		}, nil, []bool{true}},
		{"Kilo device", llmprovider.ProviderKilo, llmprovider.AuthDeviceCode, func(t *testing.T, _ *Options) {
			stubKilo(t, auth.KiloAccount{HasPersonalAccount: true}, nil)
		}, nil, nil},
		{"Kilo device as a key", llmprovider.ProviderKilo, llmprovider.AuthDeviceCode, func(t *testing.T, o *Options) {
			stubKilo(t, auth.KiloAccount{HasPersonalAccount: true}, nil)
			o.KiloDeviceKey = true
		}, nil, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			stubLogins0083(t)
			o := Options{TokenStore: newMemoryTokenStore(), NoEndpointPrompt: true}
			if c.setup != nil {
				c.setup(t, &o)
			}
			p := newQuestionBlocks0083(t, fakePrompter{selects: []int{methodIdx0059(t, o, c.provider, c.method)},
				secrets: c.secrets, confirms: c.confirms})
			var err error
			within0083(t, "the sign-in", func() { _, _, err = ResolveCredential(context.Background(), p, o, c.provider) })
			if err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Run("a kept session", func(t *testing.T) {
		client := accepting0066(keptAccess0066)
		_, o := keptGrok0066(t, client)
		p := newQuestionBlocks0083(t, fakePrompter{confirms: []bool{true}})
		var err error
		within0083(t, "the run", func() { _, _, err = ResolveCredential(context.Background(), p, o, llmprovider.ProviderGrok) })
		if err != nil {
			t.Fatal(err)
		}
	})
}

// TestQuestionEndedInputIsThisMachine_0083 (ST3, ST57): a TextPrompter whose
// input ends after the method's answer gives the default at the question, and
// the listener binds.
func TestQuestionEndedInputIsThisMachine_0083(t *testing.T) {
	logins := stubLogins0083(t)
	var out strings.Builder
	tp := &TextPrompter{In: strings.NewReader("2\n"), Out: &out}
	var err error
	within0083(t, "the run", func() {
		_, _, err = ResolveCredential(context.Background(), tp, Options{TokenStore: newMemoryTokenStore(), NoEndpointPrompt: true},
			llmprovider.ProviderGrok)
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := logins.got(); !slices.Equal(got, []bool{false}) {
		t.Errorf("the login's NoLoopback = %v; want one login without it", got)
	}
	if n := strings.Count(out.String(), question0083); n != 1 {
		t.Errorf("the question was shown %d times; want once", n)
	}
}

// TestQuestionExhaustedInputEnds_0083 (ST4, ST57): a TextPrompter whose input
// is exhausted at the question gives one error that ends the run, with the
// question shown once and no login.
func TestQuestionExhaustedInputEnds_0083(t *testing.T) {
	logins := stubLogins0083(t)
	var out strings.Builder
	tp := &TextPrompter{In: strings.NewReader("2"), Out: &out}
	var err error
	within0083(t, "the run", func() {
		_, _, err = ResolveCredential(context.Background(), tp, Options{TokenStore: newMemoryTokenStore(), NoEndpointPrompt: true},
			llmprovider.ProviderGrok)
	})
	if !errors.Is(err, io.EOF) {
		t.Errorf("err = %v; want one matching io.EOF", err)
	}
	if n := strings.Count(out.String(), question0083); n != 1 {
		t.Errorf("the question was shown %d times; want once", n)
	}
	if got := logins.got(); len(got) != 0 {
		t.Errorf("the login ran %d times; want none", len(got))
	}
}

// TestNoLoopbackWithNoPastePromptRefused_0083 (ST7, ST17): the two options
// together are an error matching ErrInvalidRequest from ConfigureLLM and from
// ResolveCredential, with no Prompter call.
func TestNoLoopbackWithNoPastePromptRefused_0083(t *testing.T) {
	o := Options{NoLoopback: true, NoPastePrompt: true, TokenStore: newMemoryTokenStore()}
	f := newFake(t, fakePrompter{})
	if _, err := ConfigureLLM(context.Background(), f, o); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("ConfigureLLM error = %v; want one matching ErrInvalidRequest", err)
	}
	if _, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderGrok); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("ResolveCredential error = %v; want one matching ErrInvalidRequest", err)
	}
	if n := len(f.seenSelect) + len(f.seenConfirm) + len(f.seenInput) + len(f.seenSecret) + len(f.seenNotify); n != 0 {
		t.Errorf("the Prompter was used %d times; want none", n)
	}
}

// TestNoBrowserQuestionKeepsToday_0083 (ST8): NoBrowserQuestion asks nothing and
// leaves today's flow: no NoLoopback at the login, today's instruction, and
// OpenURL called.
func TestNoBrowserQuestionKeepsToday_0083(t *testing.T) {
	eachScenario0083(t, func(t *testing.T, s scenario0083, entry entry0083) {
		logins := stubLogins0083(t)
		opened := 0
		f, err := s.run(t, entry, Options{NoBrowserQuestion: true, OpenURL: opener0083(&opened)}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := logins.got(); !slices.Equal(got, []bool{false}) {
			t.Errorf("the login's NoLoopback = %v; want one login without it", got)
		}
		if slices.Contains(f.seenSelect, question0083) || !slices.Contains(f.seenNotify, s.today()) || opened != 1 {
			t.Errorf("selects %q, notices %q, %d opens; want no question, today's instruction, one open", f.seenSelect, f.seenNotify, opened)
		}
	})
}

// TestQuestionAskedAgainNextRun_0083 (ST16): two runs with one Options value
// each ask the question once and each give NoLoopback to their login.
func TestQuestionAskedAgainNextRun_0083(t *testing.T) {
	logins := stubLogins0083(t)
	s := scenarios0083[0]
	o := s.options(Options{})
	for run := 1; run <= 2; run++ {
		f := newFake(t, fakePrompter{selects: []int{grokBrowserIdx, 1}})
		if _, _, err := ResolveCredential(context.Background(), f, o, s.provider); err != nil {
			t.Fatal(err)
		}
		if n := countContaining(f.seenSelect, question0083); n != 1 {
			t.Errorf("run %d asked the question %d times; want once", run, n)
		}
	}
	if got := logins.got(); !slices.Equal(got, []bool{true, true}) {
		t.Errorf("the logins' NoLoopback = %v; want both set", got)
	}
}

// TestPortsBusyEndsTheRun_0083 (ST29): after "On this machine", the login's
// busy-ports error ends the run as it is: the question asked once, no further
// prompt, no notice of the other answer.
func TestPortsBusyEndsTheRun_0083(t *testing.T) {
	const busy = "oauth: registered callback ports unavailable; free ports 1455 and 1457, or use device-code login: bind"
	stubBrowserLogin(t, func(context.Context, llmprovider.ProviderID, auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		return nil, errors.New(busy)
	})
	s := scenarios0083[0]
	f := newFake(t, fakePrompter{selects: []int{grokBrowserIdx, 0}})
	_, _, err := ResolveCredential(context.Background(), f, s.options(Options{}), s.provider)
	if err == nil || !strings.Contains(err.Error(), busy) {
		t.Fatalf("err = %v; want the login's error", err)
	}
	if n := countContaining(f.seenSelect, question0083); n != 1 || len(f.seenSelect) != 2 || len(f.seenConfirm) != 0 || len(f.seenInput) != 0 {
		t.Errorf("selects %q, confirms %q, inputs %q; want the menu and the question, nothing more", f.seenSelect, f.seenConfirm, f.seenInput)
	}
	if countContaining(f.seenNotify, "another device") != 0 {
		t.Errorf("notices %q; want none that names the other answer", f.seenNotify)
	}
}

// cancelAtQuestion0083 answers the question "On another device" after it ends
// the run's context.
type cancelAtQuestion0083 struct {
	*fakePrompter
	cancel context.CancelFunc
}

func (c *cancelAtQuestion0083) Select(title string, choices []Choice, defaultIdx int) (int, error) {
	if title == question0083 {
		c.cancel()
		return 1, nil
	}
	return c.fakePrompter.Select(title, choices, defaultIdx)
}

// TestQuestionThenCancelledRunSavesNothing_0083 (ST48): a run whose context
// ends while the question is on screen returns an error matching
// context.Canceled and saves nothing; the login, if it is called, is given the
// ended context.
func TestQuestionThenCancelledRunSavesNothing_0083(t *testing.T) {
	stubBrowserLogin(t, func(ctx context.Context, provider llmprovider.ProviderID, _ auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		if ctx.Err() == nil {
			t.Error("the login was given a live context after the run was cancelled")
		}
		return testOAuthSession(provider), nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := scenarios0083[0]
	o := s.options(Options{})
	store, _ := o.TokenStore.(*memoryTokenStore)
	p := &cancelAtQuestion0083{fakePrompter: newFake(t, fakePrompter{selects: []int{grokBrowserIdx}}), cancel: cancel}
	var err error
	within0083(t, "the run", func() { _, _, err = ResolveCredential(ctx, p, o, s.provider) })
	if !errors.Is(err, context.Canceled) || store.saves != 0 {
		t.Errorf("err = %v with %d saves; want one matching context.Canceled and no save", err, store.saves)
	}
}

// pasteReader0083 is a Prompter for a sign-in with no listener whose paste
// read waits. As an InputerContext it ends when its context does; otherwise
// its read waits for the test's release, which also ends it at cleanup.
type pasteReader0083 struct {
	*fakePrompter
	started chan struct{}
	release chan struct{}
	once    sync.Once
	mu      sync.Mutex
	notices []string
}

func newPasteReader0083(t *testing.T, selects ...int) *pasteReader0083 {
	t.Helper()
	p := &pasteReader0083{fakePrompter: newFake(t, fakePrompter{selects: selects}),
		started: make(chan struct{}), release: make(chan struct{})}
	t.Cleanup(func() { p.releaseRead() })
	return p
}

func (p *pasteReader0083) releaseRead() {
	p.once.Do(func() { close(p.release) })
}

func (p *pasteReader0083) Input(string, string) (string, error) {
	select {
	case <-p.started:
	default:
		close(p.started)
	}
	<-p.release
	return "", nil
}

func (p *pasteReader0083) Notify(_ Level, format string, args ...any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.notices = append(p.notices, fmt.Sprintf(format, args...))
}

func (p *pasteReader0083) sawNotice(want string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Contains(p.notices, want)
}

// contextReader0083 is pasteReader0083 as an InputerContext.
type contextReader0083 struct{ *pasteReader0083 }

func (p contextReader0083) InputContext(ctx context.Context, _, _ string) (string, error) {
	select {
	case <-p.started:
	default:
		close(p.started)
	}
	<-ctx.Done()
	return "", ctx.Err()
}

// waitFor0083 waits up to 5 s for cond, checking it every 10 ms.
func waitFor0083(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if cond() {
			return
		}
	}
	t.Fatalf("%s did not happen within 5 s", what)
}

// waitingLogin0083 is a browser login that shows the URL, asks for the paste on
// its own goroutine, as the real login does, and waits for its context or end.
func waitingLogin0083(t *testing.T, end func(ctx context.Context) error) *logins0083 {
	t.Helper()
	l := &logins0083{}
	stubBrowserLogin(t, func(ctx context.Context, _ llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		l.add(opts.NoLoopback)
		_ = opts.OpenURL(authorizeURL0083)
		go func() { _, _ = opts.InputCode(ctx) }()
		return nil, end(ctx)
	})
	return l
}

// TestNoLoopbackCancelDuringPasteSavesNothing_0083 (ST50): with NoLoopback and
// an InputerContext Prompter, a cancel and the run's deadline each end the
// paste read by its context, with an error matching the context's, within 5 s,
// and save nothing.
func TestNoLoopbackCancelDuringPasteSavesNothing_0083(t *testing.T) {
	for _, c := range []struct {
		name        string
		make        func() (context.Context, context.CancelFunc)
		want, other error
	}{
		{"cancelled", func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) },
			context.Canceled, context.DeadlineExceeded},
		{"deadline", func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), 300*time.Millisecond)
		}, context.DeadlineExceeded, context.Canceled},
	} {
		t.Run(c.name, func(t *testing.T) {
			logins := waitingLogin0083(t, func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() })
			ctx, cancel := c.make()
			defer cancel()
			p := newPasteReader0083(t, grokBrowserIdx)
			s := scenarios0083[0]
			o := s.options(Options{NoLoopback: true})
			store, _ := o.TokenStore.(*memoryTokenStore)
			var err error
			done := make(chan struct{})
			go func() {
				defer close(done)
				_, _, err = ResolveCredential(ctx, contextReader0083{p}, o, s.provider)
			}()
			waitFor0083(t, "the paste read", func() bool {
				select {
				case <-p.started:
					return true
				default:
					return false
				}
			})
			if errors.Is(c.want, context.Canceled) {
				cancel()
			}
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("the run did not end within 5 s of the context's end")
			}
			// ST50: a cancel is context.Canceled and the run's deadline is
			// context.DeadlineExceeded, every time, never the other.
			if !errors.Is(err, c.want) || errors.Is(err, c.other) || store.saves != 0 {
				t.Errorf("err = %v with %d saves; want one matching %v only, and no save", err, store.saves, c.want)
			}
			if got := logins.got(); !slices.Equal(got, []bool{true}) {
				t.Errorf("the login's NoLoopback = %v; want one login with it set", got)
			}
		})
	}
}

// TestQuestionAfterThePlanReads_0083 (ST18): in a first plan sign-in the
// question comes after the reads of the saved session and the client record: a
// read the store refuses gives the run's error and no question, and the warning
// for a record that cannot be decoded comes before the question.
func TestQuestionAfterThePlanReads_0083(t *testing.T) {
	t.Run("a refused read", func(t *testing.T) {
		stubLogins0083(t)
		store := &clientStore0083{memoryTokenStore: newMemoryTokenStore(), loadErr: errors.New("permission denied")}
		p := newQuestionBlocks0083(t, fakePrompter{selects: []int{planMethodIdx(t)}})
		var err error
		within0083(t, "the run", func() {
			_, _, err = ResolveCredential(context.Background(), p, Options{TokenStore: store, AgentName: planTestApp, NoEndpointPrompt: true},
				llmprovider.ProviderOpenAI)
		})
		if err == nil || !strings.Contains(err.Error(), "permission denied") {
			t.Errorf("err = %v; want the read's error", err)
		}
	})
	t.Run("an undecodable record", func(t *testing.T) {
		stubLogins0083(t)
		var syntax *json.SyntaxError
		_ = json.Unmarshal([]byte("{"), new(any))
		store := &clientStore0083{memoryTokenStore: newMemoryTokenStore(), loadErr: fmt.Errorf("record: %w", func() error {
			err := json.Unmarshal([]byte("{"), new(any))
			if !errors.As(err, &syntax) {
				t.Fatalf("test setup: %v is no syntax error", err)
			}
			return err
		}())}
		var events []string
		p := &orderPrompter0083{fakePrompter: newFake(t, fakePrompter{selects: []int{planMethodIdx(t), 0, 0}}), events: &events}
		if _, _, err := ResolveCredential(context.Background(), p, Options{TokenStore: store, AgentName: planTestApp, NoEndpointPrompt: true},
			llmprovider.ProviderOpenAI); err != nil {
			t.Fatal(err)
		}
		warn := slices.IndexFunc(events, func(e string) bool { return strings.HasPrefix(e, "notice: the saved") })
		ask := slices.Index(events, "select: "+question0083)
		if warn < 0 || ask < 0 || warn > ask {
			t.Errorf("events %q; want the warning before the question", events)
		}
	})
}

// clientStore0083 is a memory store that keeps the issued client apart, and
// whose read fails with loadErr.
type clientStore0083 struct {
	*memoryTokenStore
	loadErr error
}

func (c *clientStore0083) LoadClient(context.Context, llmprovider.ProviderID) (auth.ClientRecord, error) {
	return auth.ClientRecord{}, c.loadErr
}

func (c *clientStore0083) SaveClient(context.Context, llmprovider.ProviderID, auth.ClientRecord) error {
	return nil
}

// orderPrompter0083 records the order of the notices and selects.
type orderPrompter0083 struct {
	*fakePrompter
	events *[]string
}

func (p *orderPrompter0083) Select(title string, choices []Choice, defaultIdx int) (int, error) {
	*p.events = append(*p.events, "select: "+title)
	return p.fakePrompter.Select(title, choices, defaultIdx)
}

func (p *orderPrompter0083) Notify(level Level, format string, args ...any) {
	*p.events = append(*p.events, "notice: "+fmt.Sprintf(format, args...))
	p.fakePrompter.Notify(level, format, args...)
}

// TestNoLoopbackQuickEmptiesEndAtOnce_0083 (ST38, ST71): with NoLoopback, a
// Prompter that answers the paste with nothing at once, three times, ends the
// sign-in at once with an error matching ErrInvalidRequest, in the ruled
// words, within 5 s.
func TestNoLoopbackQuickEmptiesEndAtOnce_0083(t *testing.T) {
	waitingLogin0083(t, func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() })
	s := scenarios0083[0]
	f := newFake(t, fakePrompter{selects: []int{grokBrowserIdx}, inputs: []string{"", "", ""}})
	var err error
	within0083(t, "the sign-in", func() {
		_, _, err = ResolveCredential(context.Background(), f, s.options(Options{NoLoopback: true}), s.provider)
	})
	const want = "wizard: the prompter answered the paste with nothing, three times at once; with no callback listener only a paste can finish the sign-in"
	if err == nil || err.Error() != want || !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("err = %v; want %q matching ErrInvalidRequest", err, want)
	}
}

// TestNoLoopbackCancelWithPlainPrompter_0083 (ST51, ST69, ST72): with NoLoopback
// and a Prompter that is not an InputerContext, a cancel while the paste is
// awaited shows "Sign-in cancelled; press Enter to continue" and waits for the
// read; the run ends when the Enter comes, with an error matching
// context.Canceled and nothing saved.
func TestNoLoopbackCancelWithPlainPrompter_0083(t *testing.T) {
	plainEnd0083(t, Options{NoLoopback: true}, endCancel0083, cancelledNotice0083, context.Canceled)
}

// TestDrainAfterTheLoginsLimit_0083 (ST51, ST72): the login's own limit, with
// the paste read pending, shows "Sign-in timed out; press Enter to continue".
func TestDrainAfterTheLoginsLimit_0083(t *testing.T) {
	plainEnd0083(t, Options{NoLoopback: true}, endLimit0083, timedOutNotice0083, context.DeadlineExceeded)
}

// TestDrainNoticeWithAListener_0083 (ST72): "On this machine", the same three
// notices: the loopback's win, a cancel and the login's limit.
func TestDrainNoticeWithAListener_0083(t *testing.T) {
	for _, c := range []struct {
		name   string
		end    endKind0083
		notice string
		want   error
	}{
		{"the loopback won", endWin0083, finishedNotice0083, nil},
		{"cancelled", endCancel0083, cancelledNotice0083, context.Canceled},
		{"the login's limit", endLimit0083, timedOutNotice0083, context.DeadlineExceeded},
	} {
		t.Run(c.name, func(t *testing.T) {
			plainEnd0083(t, Options{}, c.end, c.notice, c.want)
		})
	}
}

type endKind0083 int

const (
	endWin0083 endKind0083 = iota
	endCancel0083
	endLimit0083
)

// plainEnd0083 runs Grok's browser sign-in over a plain Prompter whose paste
// read waits for a line, ends the sign-in as kind says once the read is
// pending, waits up to 5 s for notice, checks the run has not returned, feeds
// the line, and checks the run's error matches want (nil: no error) and that
// nothing was saved by a failed run.
func plainEnd0083(t *testing.T, base Options, kind endKind0083, notice string, want error) {
	t.Helper()
	selects := []int{grokBrowserIdx}
	if !base.NoLoopback {
		selects = append(selects, 0) // "On this machine"
	}
	p := newPasteReader0083(t, selects...)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stubBrowserLogin(t, func(c context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		_ = opts.OpenURL(authorizeURL0083)
		go func() { _, _ = opts.InputCode(c) }()
		select {
		case <-p.started:
		case <-time.After(5 * time.Second):
			t.Error("the paste read did not start")
		}
		switch kind {
		case endCancel0083:
			cancel()
			<-c.Done()
			return nil, c.Err()
		case endLimit0083:
			return nil, fmt.Errorf("oauth: browser login timed out: %w", context.DeadlineExceeded)
		}
		return testOAuthSession(provider), nil
	})
	s := scenarios0083[0]
	o := s.options(base)
	store, _ := o.TokenStore.(*memoryTokenStore)
	var err error
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _, err = ResolveCredential(ctx, p, o, s.provider)
	}()
	waitFor0083(t, "the notice "+notice, func() bool { return p.sawNotice(notice) })
	select {
	case <-done:
		t.Fatal("the run returned before the Enter the notice asks for")
	default:
	}
	p.releaseRead()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the run did not return within 5 s of the Enter")
	}
	if want == nil {
		if err != nil {
			t.Errorf("err = %v; want none", err)
		}
		return
	}
	if !errors.Is(err, want) || store.saves != 0 {
		t.Errorf("err = %v with %d saves; want one matching %v and no save", err, store.saves, want)
	}
}

// TestQuestionOutOfRangeIsAnError_0083 (ST73): an answer outside the two
// choices is a plain error naming the choices, and no login runs.
func TestQuestionOutOfRangeIsAnError_0083(t *testing.T) {
	logins := stubLogins0083(t)
	s := scenarios0083[0]
	p := &outOfRange0083{fakePrompter: newFake(t, fakePrompter{selects: []int{grokBrowserIdx}})}
	_, _, err := ResolveCredential(context.Background(), p, s.options(Options{}), s.provider)
	if err == nil || !strings.Contains(err.Error(), "chose 5 of 2 choices") {
		t.Errorf("err = %v; want the choice out of range named", err)
	}
	if got := logins.got(); len(got) != 0 {
		t.Errorf("the login ran %d times; want none", len(got))
	}
}

// outOfRange0083 answers the question with index 5.
type outOfRange0083 struct{ *fakePrompter }

func (p *outOfRange0083) Select(title string, choices []Choice, defaultIdx int) (int, error) {
	if title == question0083 {
		return 5, nil
	}
	return p.fakePrompter.Select(title, choices, defaultIdx)
}
