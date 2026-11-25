package wizard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0090-PLAN Phase 1 (S78): a run whose context ends during the Ollama
// endpoint check shows no notice and asks nothing, in the prompting loop and
// under NoEndpointPrompt; a run that is alive is told and asked as before.
// Each test calls resolveBaseURL with a client whose transport answers or
// fails as its row needs, and sends nothing over the network.

// ollama0090 is the Ollama descriptor's endpoint fields, the ones
// resolveBaseURL reads.
var ollama0090 = llmprovider.Descriptor{ID: llmprovider.ProviderOllama, Label: "Ollama",
	SupportsBaseURL: true, IsLocal: true, DefaultBaseURL: "http://localhost:11434"}

// endpoint0090 is the endpoint each interactive test answers with.
const endpoint0090 = "http://127.0.0.1:11434"

// refused0090 is a refused connection, the error a stopped Ollama gives.
func refused0090() error {
	return &net.OpError{Op: "dial", Net: "tcp", Err: os.NewSyscallError("connect", syscall.ECONNREFUSED)}
}

// answered0090 is a reply with status and an empty JSON body.
func answered0090(r *http.Request, status int) *http.Response {
	return &http.Response{StatusCode: status, Status: http.StatusText(status), Request: r,
		Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{}`))}
}

// options0090 are the run's options with a client whose transport is rt.
func options0090(t *testing.T, rt roundTripFunc) Options {
	return Options{HTTPClient: &http.Client{Transport: rt},
		ProviderOptions: []llmprovider.Option{testCache(t)}}
}

// assertEnded0090 fails t unless err matches want and nothing was shown or
// asked after the check.
func assertEnded0090(t *testing.T, f *fakePrompter, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Errorf("err = %v; want one matching %v", err, want)
	}
	if len(f.seenNotify) != 0 {
		t.Errorf("notices %q; want none after the run ended", f.seenNotify)
	}
	if len(f.seenConfirm) != 0 {
		t.Errorf("asked %q; want nothing after the run ended", f.seenConfirm)
	}
}

// inputCancels0090 is fakePrompter, but its Input ends the run's context as
// it returns its answer: the answer and the cancel arrive together.
type inputCancels0090 struct {
	*fakePrompter
	cancel context.CancelFunc
}

func (p inputCancels0090) Input(prompt, def string) (string, error) {
	defer p.cancel()
	return p.fakePrompter.Input(prompt, def)
}

// TestResolveBaseURL_UnreachableAlive_0090: T1. The run is alive and the
// endpoint refuses: "cannot reach", then the question, as before.
func TestResolveBaseURL_UnreachableAlive_0090(t *testing.T) {
	f := newFake(t, fakePrompter{inputs: []string{endpoint0090}, confirms: []bool{false}})
	got, err := resolveBaseURL(context.Background(), f, ollama0090, options0090(t, func(*http.Request) (*http.Response, error) {
		return nil, refused0090()
	}))
	if err != nil || got != endpoint0090 {
		t.Fatalf("resolveBaseURL = %q, %v; want %q, nil", got, err, endpoint0090)
	}
	wantNotice := "cannot reach " + endpoint0090 + " (offline)"
	if !slices.Equal(f.seenNotify, []string{wantNotice}) {
		t.Errorf("notices %q; want [%q]", f.seenNotify, wantNotice)
	}
	if !slices.Equal(f.seenConfirm, []string{"Try a different endpoint?"}) {
		t.Errorf("asked %q; want the endpoint question", f.seenConfirm)
	}
}

// TestResolveBaseURL_AnsweredAlive_0090: T5. The run is alive and the server
// answers 500: "answered", then the question, as before.
func TestResolveBaseURL_AnsweredAlive_0090(t *testing.T) {
	f := newFake(t, fakePrompter{inputs: []string{endpoint0090}, confirms: []bool{false}})
	got, err := resolveBaseURL(context.Background(), f, ollama0090, options0090(t, func(r *http.Request) (*http.Response, error) {
		return answered0090(r, http.StatusInternalServerError), nil
	}))
	if err != nil || got != endpoint0090 {
		t.Fatalf("resolveBaseURL = %q, %v; want %q, nil", got, err, endpoint0090)
	}
	wantNotice := endpoint0090 + " answered: provider unavailable, HTTP 500"
	if !slices.Equal(f.seenNotify, []string{wantNotice}) {
		t.Errorf("notices %q; want [%q]", f.seenNotify, wantNotice)
	}
	if !slices.Equal(f.seenConfirm, []string{"Try a different endpoint?"}) {
		t.Errorf("asked %q; want the endpoint question", f.seenConfirm)
	}
}

// TestResolveBaseURL_NoPromptUnreachableAlive_0090: T6. Under
// NoEndpointPrompt, the run alive and the endpoint refusing: "wizard: cannot
// reach", and nothing shown or asked, as before.
func TestResolveBaseURL_NoPromptUnreachableAlive_0090(t *testing.T) {
	f := newFake(t, fakePrompter{})
	o := options0090(t, func(*http.Request) (*http.Response, error) { return nil, refused0090() })
	o.NoEndpointPrompt = true
	_, err := resolveBaseURL(context.Background(), f, ollama0090, o)
	want := "wizard: cannot reach " + ollama0090.DefaultBaseURL + ": "
	if err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Errorf("err = %v; want one beginning %q", err, want)
	}
	if !errors.Is(err, llmprovider.ErrOffline) {
		t.Errorf("err = %v; want one matching llmprovider.ErrOffline", err)
	}
	if len(f.allText) != 0 {
		t.Errorf("shown %q; want nothing under NoEndpointPrompt", f.allText)
	}
}

// TestResolveBaseURL_EndedBeforeAsk_0090: T4. A run already ended is not
// asked for an endpoint and makes no check: "enter base URL: <ctx err>".
func TestResolveBaseURL_EndedBeforeAsk_0090(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := newFake(t, fakePrompter{})
	var probes int
	_, err := resolveBaseURL(ctx, f, ollama0090, options0090(t, func(*http.Request) (*http.Response, error) {
		probes++
		return nil, refused0090()
	}))
	if err != nil && !strings.HasPrefix(err.Error(), "enter base URL: ") {
		t.Errorf("err = %v; want it to begin \"enter base URL: \"", err)
	}
	if len(f.seenInput) != 0 || probes != 0 {
		t.Errorf("asked %q and checked %d time(s); want neither", f.seenInput, probes)
	}
	assertEnded0090(t, f, err, context.Canceled)
}

// TestResolveBaseURL_ProbeBoundAlive_0090: T8. The check's own bound passes
// with the run alive: an outage, told as "timed out", then the question. The
// guard reads the run's context, never the error's class.
func TestResolveBaseURL_ProbeBoundAlive_0090(t *testing.T) {
	f := newFake(t, fakePrompter{inputs: []string{endpoint0090}, confirms: []bool{false}})
	got, err := resolveBaseURL(context.Background(), f, ollama0090, options0090(t, func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("the check's own bound: %w", context.DeadlineExceeded)
	}))
	if err != nil || got != endpoint0090 {
		t.Fatalf("resolveBaseURL = %q, %v; want %q, nil", got, err, endpoint0090)
	}
	wantNotice := "cannot reach " + endpoint0090 + " (timed out)"
	if !slices.Equal(f.seenNotify, []string{wantNotice}) {
		t.Errorf("notices %q; want [%q]", f.seenNotify, wantNotice)
	}
	if !slices.Equal(f.seenConfirm, []string{"Try a different endpoint?"}) {
		t.Errorf("asked %q; want the endpoint question", f.seenConfirm)
	}
}

// TestResolveBaseURL_EndedAfterAnswer_0090: T10. The server answers 200 and
// the run ends as the answer returns: the endpoint is returned, and the next
// step's context check ends the run.
func TestResolveBaseURL_EndedAfterAnswer_0090(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := newFake(t, fakePrompter{inputs: []string{endpoint0090}})
	got, err := resolveBaseURL(ctx, f, ollama0090, options0090(t, func(r *http.Request) (*http.Response, error) {
		cancel()
		return answered0090(r, http.StatusOK), nil
	}))
	if err != nil || got != endpoint0090 {
		t.Fatalf("resolveBaseURL = %q, %v; want %q, nil: the check succeeded", got, err, endpoint0090)
	}
	if len(f.seenNotify) != 0 || len(f.seenConfirm) != 0 {
		t.Errorf("notices %q, asked %q; want neither", f.seenNotify, f.seenConfirm)
	}
}

// TestResolveBaseURL_CancelledProbe_0090: T2. The run is cancelled while the
// check is in flight, or by the Input that returns the endpoint: nothing
// shown, nothing asked, the context's error. The transport fails the request
// whether or not the context is done, since a client still calls it.
func TestResolveBaseURL_CancelledProbe_0090(t *testing.T) {
	t.Run("while the check is in flight", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		f := newFake(t, fakePrompter{inputs: []string{endpoint0090}})
		_, err := resolveBaseURL(ctx, f, ollama0090, options0090(t, func(*http.Request) (*http.Response, error) {
			cancel()
			return nil, refused0090()
		}))
		assertEnded0090(t, f, err, context.Canceled)
		if err != nil && !strings.HasPrefix(err.Error(), "enter base URL: ") {
			t.Errorf("err = %v; want it to begin \"enter base URL: \"", err)
		}
	})
	t.Run("as the endpoint is entered", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		f := newFake(t, fakePrompter{inputs: []string{endpoint0090}})
		p := inputCancels0090{fakePrompter: f, cancel: cancel}
		var probes int
		_, err := resolveBaseURL(ctx, p, ollama0090, options0090(t, func(*http.Request) (*http.Response, error) {
			probes++
			return nil, refused0090()
		}))
		assertEnded0090(t, f, err, context.Canceled)
		if err != nil && !strings.HasPrefix(err.Error(), "enter base URL: ") {
			t.Errorf("err = %v; want it to begin \"enter base URL: \"", err)
		}
		t.Logf("the transport was called %d time(s) with a done context", probes)
	})
}

// assertProbed0090 fails t unless the check was made: a deadline that passed
// before the loop would end the run at its top, T4's path, without reaching
// the guard after a failed check (0090-PLAN Phase 1, pass 2).
func assertProbed0090(t *testing.T, probes *atomic.Int32) {
	t.Helper()
	if n := probes.Load(); n < 1 {
		t.Errorf("the check was made %d time(s); want at least 1, so the guard after a failed check is what ended the run", n)
	}
}

// TestResolveBaseURL_DeadlineProbe_0090: T3. The run's deadline passes while
// the check is in flight: as T2, with context.DeadlineExceeded.
func TestResolveBaseURL_DeadlineProbe_0090(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	f := newFake(t, fakePrompter{inputs: []string{endpoint0090}})
	var probes atomic.Int32
	_, err := resolveBaseURL(ctx, f, ollama0090, options0090(t, func(*http.Request) (*http.Response, error) {
		probes.Add(1)
		<-ctx.Done()
		return nil, refused0090()
	}))
	assertProbed0090(t, &probes)
	assertEnded0090(t, f, err, context.DeadlineExceeded)
	if err != nil && !strings.HasPrefix(err.Error(), "enter base URL: ") {
		t.Errorf("err = %v; want it to begin \"enter base URL: \"", err)
	}
}

// TestResolveBaseURL_NoPromptEnded_0090: T7. Under NoEndpointPrompt, the run
// ends during the check, which fails to connect or is answered 500: nothing
// shown, "wizard: endpoint check ended: <ctx err>", never "cannot reach".
func TestResolveBaseURL_NoPromptEnded_0090(t *testing.T) {
	for _, tc := range []struct {
		name  string
		reply func(*http.Request) (*http.Response, error)
	}{
		{"a connect failure", func(*http.Request) (*http.Response, error) { return nil, refused0090() }},
		{"a non-200 answer", func(r *http.Request) (*http.Response, error) {
			return answered0090(r, http.StatusInternalServerError), nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			f := newFake(t, fakePrompter{})
			o := options0090(t, func(r *http.Request) (*http.Response, error) {
				cancel()
				return tc.reply(r)
			})
			o.NoEndpointPrompt = true
			_, err := resolveBaseURL(ctx, f, ollama0090, o)
			assertEnded0090(t, f, err, context.Canceled)
			if want := "wizard: endpoint check ended: context canceled"; err == nil || err.Error() != want {
				t.Errorf("err = %v; want %q", err, want)
			}
			if err != nil && strings.Contains(err.Error(), "cannot reach") {
				t.Errorf("err = %v; want no \"cannot reach\" for an ended run", err)
			}
			if len(f.allText) != 0 {
				t.Errorf("shown %q; want nothing under NoEndpointPrompt", f.allText)
			}
		})
	}
}

// TestResolveBaseURL_AnsweredEnded_0090: T15. The server answers 500 and the
// run ends, by a cancel or a deadline, as the answer is read: nothing shown,
// nothing asked, the context's error. A guard only in the "cannot reach"
// branch would miss it.
func TestResolveBaseURL_AnsweredEnded_0090(t *testing.T) {
	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		f := newFake(t, fakePrompter{inputs: []string{endpoint0090}})
		_, err := resolveBaseURL(ctx, f, ollama0090, options0090(t, func(r *http.Request) (*http.Response, error) {
			cancel()
			return answered0090(r, http.StatusInternalServerError), nil
		}))
		assertEnded0090(t, f, err, context.Canceled)
	})
	t.Run("deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		f := newFake(t, fakePrompter{inputs: []string{endpoint0090}})
		var probes atomic.Int32
		_, err := resolveBaseURL(ctx, f, ollama0090, options0090(t, func(r *http.Request) (*http.Response, error) {
			probes.Add(1)
			<-ctx.Done()
			return answered0090(r, http.StatusInternalServerError), nil
		}))
		assertProbed0090(t, &probes)
		assertEnded0090(t, f, err, context.DeadlineExceeded)
	})
}
