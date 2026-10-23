package wizard

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// ctxPrompter is a fakePrompter that implements InputerContext for the
// paste prompt. Its read answers at once when answer is set; otherwise it
// blocks until its context ends, or, with ignoreCtx, until the wizard asks
// for the Enter. It fails a second read that starts while one runs, an
// InputContext for any other prompt, and a Select that starts while the
// read runs.
type ctxPrompter struct {
	*fakePrompter
	answer    string
	ignoreCtx bool

	started     chan struct{}
	startedOnce sync.Once
	release     chan struct{}
	releaseOnce sync.Once
	reading     atomic.Bool
	calls       atomic.Int32

	mu     sync.Mutex
	ctxErr error // the read's context's error, once it ended
}

func newCtxPrompter(f *fakePrompter) *ctxPrompter {
	return &ctxPrompter{fakePrompter: f, started: make(chan struct{}), release: make(chan struct{})}
}

var _ InputerContext = (*ctxPrompter)(nil)

func (c *ctxPrompter) InputContext(ctx context.Context, prompt, _ string) (string, error) {
	c.calls.Add(1)
	if prompt != pasteCodePrompt {
		c.t.Errorf("InputContext(%q); want the paste prompt only", prompt)
	}
	if c.reading.Swap(true) {
		c.t.Error("a second paste read started while one ran")
	}
	defer c.reading.Store(false)
	if c.answer != "" {
		return c.answer, nil
	}
	c.startedOnce.Do(func() { close(c.started) })
	if c.ignoreCtx {
		<-c.release
		return "", nil
	}
	<-ctx.Done()
	c.mu.Lock()
	c.ctxErr = ctx.Err()
	c.mu.Unlock()
	return "", ctx.Err()
}

func (c *ctxPrompter) Notify(level Level, format string, args ...any) {
	c.fakePrompter.Notify(level, format, args...)
	if strings.Contains(fmt.Sprintf(format, args...), "press Enter") {
		c.releaseOnce.Do(func() { close(c.release) })
	}
}

func (c *ctxPrompter) Select(title string, choices []Choice, defaultIdx int) (int, error) {
	if c.reading.Load() {
		c.t.Errorf("Select(%q) started while the paste read ran", title)
	}
	return c.fakePrompter.Select(title, choices, defaultIdx)
}

// stubLoopbackWin0057 is a browser login whose loopback wins once the paste
// read has started. It gives InputCode a context it never ends itself, so
// that only the wizard's own cancel can end the read.
func stubLoopbackWin0057(t *testing.T, c *ctxPrompter) {
	stubBrowserLogin(t, func(_ context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		if err := opts.OpenURL("https://authorize.test"); err != nil {
			t.Errorf("OpenURL() error = %v", err)
		}
		go func() { _, _ = opts.InputCode(context.Background()) }()
		<-c.started
		return testOAuthSession(provider), nil
	})
}

// runBounded0057 runs ConfigureLLM, failing t if it has not returned in 5 s,
// so that a plant which hangs the drain fails here.
func runBounded0057(t *testing.T, p Prompter, o Options) (Result, error) {
	t.Helper()
	type result struct {
		res Result
		err error
	}
	done := make(chan result, 1)
	go func() {
		res, err := ConfigureLLM(context.Background(), p, o)
		done <- result{res, err}
	}()
	select {
	case r := <-done:
		return r.res, r.err
	case <-time.After(5 * time.Second):
		t.Fatal("ConfigureLLM did not return within 5 s of a finished sign-in")
		return Result{}, nil
	}
}

// grokBrowserScript0057 chooses Grok, its browser sign-in, and the first
// recommended model.
func grokBrowserScript0057(t *testing.T) fakePrompter {
	return fakePrompter{blankSearches: true, selects: []int{providerIdx(t, llmprovider.ProviderGrok), grokBrowserIdx, 0}}
}

// TestInputerContext_LoopbackWinNoEnter_0057 (0057-MADR D3, issue 8): when
// the loopback overtakes an InputerContext's paste read, the wizard ends the
// read's context itself, asks for no Enter, and starts its next prompt only
// after the read returned.
func TestInputerContext_LoopbackWinNoEnter_0057(t *testing.T) {
	c := newCtxPrompter(newFake(t, grokBrowserScript0057(t)))
	stubLoopbackWin0057(t, c)
	res, err := runBounded0057(t, c, Options{TokenStore: newMemoryTokenStore()})
	if err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	ctxErr := c.ctxErr
	c.mu.Unlock()
	if res.Kind != CredOAuth || !errors.Is(ctxErr, context.Canceled) {
		t.Errorf("kind %q, the read's context ended with %v; want oauth, and context.Canceled", res.Kind, ctxErr)
	}
	if n := countContaining(c.seenNotify, "press Enter"); n != 0 {
		t.Errorf("notices %q; want no press-Enter notice", c.seenNotify)
	}
}

// TestInputerContext_PasteWins_0057: an InputContext that answers before the
// loopback gives the login its code.
func TestInputerContext_PasteWins_0057(t *testing.T) {
	c := newCtxPrompter(newFake(t, grokBrowserScript0057(t)))
	c.answer = "code-0057"
	stubBrowserLogin(t, func(ctx context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		if err := opts.OpenURL("https://authorize.test"); err != nil {
			t.Errorf("OpenURL() error = %v", err)
		}
		if code, err := opts.InputCode(ctx); err != nil || code != "code-0057" {
			t.Errorf("InputCode = %q, %v; want the pasted code", code, err)
		}
		return testOAuthSession(provider), nil
	})
	if _, err := runBounded0057(t, c, Options{TokenStore: newMemoryTokenStore()}); err != nil {
		t.Fatal(err)
	}
	if n := c.calls.Load(); n != 1 {
		t.Errorf("%d InputContext calls; want 1", n)
	}
}

// TestInputerContext_IgnoresContext_0057: an InputContext that does not give
// up when its context ends gets the press-Enter notice after drainGrace, as
// any other Prompter does, and the run finishes.
func TestInputerContext_IgnoresContext_0057(t *testing.T) {
	original := drainGrace
	t.Cleanup(func() { drainGrace = original })
	drainGrace = 10 * time.Millisecond
	c := newCtxPrompter(newFake(t, grokBrowserScript0057(t)))
	c.ignoreCtx = true
	stubLoopbackWin0057(t, c)
	if _, err := runBounded0057(t, c, Options{TokenStore: newMemoryTokenStore()}); err != nil {
		t.Fatal(err)
	}
	if n := countContaining(c.seenNotify, "press Enter to continue"); n != 1 {
		t.Errorf("notices %q; want one press-Enter notice", c.seenNotify)
	}
}

// TestInputerContext_PasteOnly_0057: only the paste prompt goes to
// InputContext; the model search's Input does not.
func TestInputerContext_PasteOnly_0057(t *testing.T) {
	c := newCtxPrompter(newFake(t, grokBrowserScript0057(t)))
	c.answer = "code-0057"
	stubBrowserLogin(t, func(ctx context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		if err := opts.OpenURL("https://authorize.test"); err != nil {
			t.Errorf("OpenURL() error = %v", err)
		}
		_, _ = opts.InputCode(ctx)
		return testOAuthSession(provider), nil
	})
	if _, err := runBounded0057(t, c, Options{TokenStore: newMemoryTokenStore()}); err != nil {
		t.Fatal(err)
	}
	if n := c.calls.Load(); n != 1 || countContaining(c.seenInput, searchModelsPrompt) != 1 {
		t.Errorf("%d InputContext calls, inputs %q; want the paste alone there, and the search through Input", n, c.seenInput)
	}
}

// TestTextPrompter_NotInputerContext_0057 (0057-MADR Q2): TextPrompter keeps
// the Enter, so it must not implement InputerContext.
func TestTextPrompter_NotInputerContext_0057(t *testing.T) {
	var p any = &TextPrompter{}
	if _, ok := p.(InputerContext); ok {
		t.Error("*TextPrompter implements InputerContext; a read of stdin cannot be given up")
	}
}
