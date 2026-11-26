//go:build unix

package wizard

import (
	"context"
	"errors"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"golang.org/x/sys/unix"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// The tests of 0087-PLAN Phase 2 that need the Unix settle (select(2) on the
// pipe's descriptor). The helpers are late_lf_0087_test.go's.

// TestSecret_LateLineFeedDropped_0087 (ST3): a CR, then an LF the line-end
// hook writes before Secret returns: the LF is dropped, and the next prompt
// of each kind takes the person's answer.
func TestSecret_LateLineFeedDropped_0087(t *testing.T) {
	t.Run("Input", func(t *testing.T) {
		p, w, hook := latePipe0087(t)
		write0087(t, w, key0087+"\r")
		hook.arm(writeLater0087(t, w, "\n"))
		if got, err := p.Secret("Key"); err != nil || got != key0087 {
			t.Fatalf("Secret = %q, %v; want the key", got, err)
		}
		write0087(t, w, "my-model\n")
		if got, err := p.Input("Model", "saved-model"); err != nil || got != "my-model" {
			t.Errorf("Input after a late LF = %q, %v; want my-model: the late LF answered with the default", got, err)
		}
	})
	t.Run("Select", func(t *testing.T) {
		p, w, hook := latePipe0087(t)
		write0087(t, w, key0087+"\r")
		hook.arm(writeLater0087(t, w, "\n"))
		if got, err := p.Secret("Key"); err != nil || got != key0087 {
			t.Fatalf("Secret = %q, %v; want the key", got, err)
		}
		write0087(t, w, "3\n")
		choices := []Choice{{Label: "a"}, {Label: "b"}, {Label: "c"}}
		if got, err := p.Select("Pick", choices, 0); err != nil || got != 2 {
			t.Errorf("Select after a late LF = %d, %v; want 2", got, err)
		}
	})
	t.Run("Confirm", func(t *testing.T) {
		p, w, hook := latePipe0087(t)
		write0087(t, w, key0087+"\r")
		hook.arm(writeLater0087(t, w, "\n"))
		if got, err := p.Secret("Key"); err != nil || got != key0087 {
			t.Fatalf("Secret = %q, %v; want the key", got, err)
		}
		write0087(t, w, "n\n")
		if got, err := p.Confirm("Go on?", true); err != nil || got {
			t.Errorf("Confirm after a late LF = %v, %v; want false", got, err)
		}
	})
	t.Run("Secret", func(t *testing.T) {
		p, w, hook := latePipe0087(t)
		write0087(t, w, key0087+"\r")
		hook.arm(writeLater0087(t, w, "\n"))
		if got, err := p.Secret("Key"); err != nil || got != key0087 {
			t.Fatalf("Secret = %q, %v; want the key", got, err)
		}
		write0087(t, w, "key2\r")
		if got, err := p.Secret("Key"); err != nil || got != "key2" {
			t.Errorf("second Secret = %q, %v; want key2", got, err)
		}
		if strings.Contains(hook.String(), "empty value") {
			t.Errorf("the late LF was read as an empty entry: %q", hook.String())
		}
	})
}

// TestSecret_LateLineFeedDuringWait_0087 (ST3): an LF that arrives 10 ms into
// the wait is dropped. The wait is set to 2 s, a margin for a stalled -race
// run; W itself is pinned by ST23's claims row.
func TestSecret_LateLineFeedDuringWait_0087(t *testing.T) {
	saved := settleWait
	settleWait = 2 * time.Second
	t.Cleanup(func() { settleWait = saved })
	p, w, hook := latePipe0087(t)
	write0087(t, w, key0087+"\r")
	var wg sync.WaitGroup
	hook.arm(func() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
			if _, err := w.WriteString("\n"); err != nil {
				t.Errorf("late LF: %v", err)
			}
		}()
	})
	if got, err := p.Secret("Key"); err != nil || got != key0087 {
		t.Fatalf("Secret = %q, %v; want the key", got, err)
	}
	wg.Wait()
	write0087(t, w, "my-model\n")
	if got, err := p.Input("Model", "saved-model"); err != nil || got != "my-model" {
		t.Errorf("Input after an LF 10 ms into the wait = %q, %v; want my-model", got, err)
	}
}

// fakeSelect0087 replaces selectFn for the test. step is given each call's
// number, from 1, and its wait; it returns the call's result, or real to make
// the call through unix.Select. The waits are returned as they were asked for.
func fakeSelect0087(t *testing.T, step func(call int, wait time.Duration) (n int, real bool, err error)) *[]time.Duration {
	t.Helper()
	saved := selectFn
	t.Cleanup(func() { selectFn = saved })
	waits := &[]time.Duration{}
	selectFn = func(nfd int, r, w, e *unix.FdSet, tv *unix.Timeval) (int, error) {
		wait := time.Duration(tv.Nano())
		*waits = append(*waits, wait)
		n, real, err := step(len(*waits), wait)
		if real {
			return saved(nfd, r, w, e, tv)
		}
		return n, err
	}
	return waits
}

// pipeReader0087 is the reading end of a new os.Pipe, both ends closed when the
// test ends.
func pipeReader0087(t *testing.T) *os.File {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = w.Close()
		_ = r.Close()
	})
	return r
}

// TestPendingWithin_Interrupted_0087 (ST43; 0087-MADR A-8, OR17): a select
// interrupted by a signal (EINTR) is made again with the time left to a
// deadline fixed before the first call. Each subtest is one of A-8's cases.
func TestPendingWithin_Interrupted_0087(t *testing.T) {
	t.Run("EINTR, then the LF", func(t *testing.T) { // K1, K2
		saved := settleWait
		settleWait = 2 * time.Second
		t.Cleanup(func() { settleWait = saved })
		p, w, _ := latePipe0087(t)
		write0087(t, w, key0087+"\r")
		const spent = 2 * time.Millisecond
		waits := fakeSelect0087(t, func(call int, _ time.Duration) (int, bool, error) {
			if call > 1 {
				return 0, true, nil
			}
			// The signal lands in the wait; the LF arrives after it.
			write0087(t, w, "\n")
			time.Sleep(spent)
			return 0, false, unix.EINTR
		})
		if got, err := p.Secret("Key"); err != nil || got != key0087 {
			t.Fatalf("Secret = %q, %v; want the key", got, err)
		}
		write0087(t, w, "my-model\n")
		if got, err := p.Input("Model", "saved-model"); err != nil || got != "my-model" {
			t.Errorf("Input after an interrupted wait = %q, %v; want my-model: the LF leaked after EINTR", got, err)
		}
		if len(*waits) != 2 {
			t.Fatalf("select called %d time(s); want 2: once, and again after EINTR", len(*waits))
		}
		if first, second := (*waits)[0], (*waits)[1]; first != settleWait || second > first-spent {
			t.Errorf("select waits %v then %v; want %v, then at most %v: the time left, not a fresh wait", first, second, settleWait, settleWait-spent)
		}
	})
	t.Run("EINTR until the deadline", func(t *testing.T) { // K3
		// In a synctest bubble the clock moves only when every goroutine
		// in it is blocked, so each interrupted call takes exactly spent
		// and the calls and their waits are fixed, however loaded the host
		// (0087-MADR A-9).
		r := pipeReader0087(t)
		synctest.Test(t, func(t *testing.T) {
			const wait, spent = 30 * time.Millisecond, 5 * time.Millisecond
			const most = 2 * int(wait/spent) // a retry that never ends stops here
			waits := fakeSelect0087(t, func(call int, _ time.Duration) (int, bool, error) {
				if call > most {
					return 0, false, unix.EBADF
				}
				time.Sleep(spent)
				return 0, false, unix.EINTR
			})
			start := time.Now()
			ready, err := pendingWithin(r, wait)
			if ready || err != nil {
				t.Errorf("pendingWithin = %v, %v; want false and no error once the deadline passes", ready, err)
			}
			want := []time.Duration{30 * time.Millisecond, 25 * time.Millisecond, 20 * time.Millisecond,
				15 * time.Millisecond, 10 * time.Millisecond, 5 * time.Millisecond}
			if !slices.Equal(*waits, want) {
				t.Errorf("select waits %v; want %v: each call the time left, and none at or after the deadline", *waits, want)
			}
			if d := time.Since(start); d != wait {
				t.Errorf("pendingWithin took %v of the bubble's clock; want exactly %v, its deadline", d, wait)
			}
		})
	})
	t.Run("EINTR with no time left", func(t *testing.T) { // K4
		r := pipeReader0087(t)
		const wait = 20 * time.Millisecond
		waits := fakeSelect0087(t, func(int, time.Duration) (int, bool, error) {
			time.Sleep(2 * wait)
			return 0, false, unix.EINTR
		})
		ready, err := pendingWithin(r, wait)
		if ready || err != nil {
			t.Errorf("pendingWithin = %v, %v; want false and no error", ready, err)
		}
		if len(*waits) != 1 {
			t.Errorf("select called %d time(s); want 1: no time was left after EINTR", len(*waits))
		}
	})
	t.Run("another error", func(t *testing.T) { // K5
		r := pipeReader0087(t)
		waits := fakeSelect0087(t, func(int, time.Duration) (int, bool, error) {
			return 0, false, unix.EBADF
		})
		ready, err := pendingWithin(r, 20*time.Millisecond)
		if ready || !errors.Is(err, unix.EBADF) {
			t.Errorf("pendingWithin = %v, %v; want false and EBADF, as A3 decides", ready, err)
		}
		if len(*waits) != 1 {
			t.Errorf("select called %d time(s); want 1: only EINTR is waited again", len(*waits))
		}
	})
}

// TestSecret_NoWaitWhenNotNeeded_0087 (ST2, ST7): with the wait set to 5 s, an
// entry whose CR+LF arrive together, and one ended by an LF, return within
// 2 s: neither waits.
func TestSecret_NoWaitWhenNotNeeded_0087(t *testing.T) {
	saved := settleWait
	settleWait = 5 * time.Second
	t.Cleanup(func() { settleWait = saved })
	for _, c := range []struct{ name, typed string }{
		{"CR+LF together", key0087 + "\r\n"},
		{"LF-ended", key0087 + "\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			p, w, _ := latePipe0087(t)
			write0087(t, w, c.typed)
			start := time.Now()
			if got, err := p.Secret("Key"); err != nil || got != key0087 {
				t.Fatalf("Secret = %q, %v; want the key", got, err)
			}
			if d := time.Since(start); d > 2*time.Second {
				t.Errorf("Secret took %v; want under 2 s, no wait", d)
			}
		})
	}
}

// noReadLeftBehind0087 is TestSecret_NoReadLeftBehind_0087 (ST14), which the
// Linux and macOS files run with their request for the pipe's pending count:
// after Secret returns, one byte written to the pipe is still pending 200 ms
// later, so nothing reads the input after Secret returns.
func noReadLeftBehind0087(t *testing.T, pending func(fd int) (int, error)) {
	t.Helper()
	p, w, _ := latePipe0087(t)
	write0087(t, w, key0087+"\r")
	if got, err := p.Secret("Key"); err != nil || got != key0087 {
		t.Fatalf("Secret = %q, %v; want the key", got, err)
	}
	write0087(t, w, "x")
	time.Sleep(200 * time.Millisecond)
	r, ok := p.In.(*os.File)
	if !ok {
		t.Fatal("In is not the pipe")
	}
	n, err := pending(int(r.Fd()))
	if err != nil {
		t.Fatalf("the pipe's pending count: %v", err)
	}
	if n != 1 {
		t.Errorf("the pipe holds %d byte(s) 200 ms after a write; want 1: something read after Secret returned", n)
	}
}

// TestSecret_RawFieldCleared_0087 (ST36): once Secret has returned, readMasked
// called directly does not settle: no raw read is running.
func TestSecret_RawFieldCleared_0087(t *testing.T) {
	p, w, hook := latePipe0087(t)
	write0087(t, w, key0087+"\r")
	if got, err := p.Secret("Key"); err != nil || got != key0087 {
		t.Fatalf("Secret = %q, %v; want the key", got, err)
	}
	write0087(t, w, "key2\r")
	hook.arm(writeLater0087(t, w, "\n"))
	if got, err := p.readMasked("Key"); err != nil || got != "key2" {
		t.Fatalf("readMasked = %q, %v; want key2", got, err)
	}
	write0087(t, w, "ans\n")
	if got, err := p.Input("Model", "saved-model"); err != nil || got != "saved-model" {
		t.Errorf("Input = %q, %v; want its default: the LF is kept with no raw read running", got, err)
	}
}

// TestSecret_SecondLineEndKept_0087 (ST6, ST37): one LF is dropped per CR; a
// second line ending is kept.
func TestSecret_SecondLineEndKept_0087(t *testing.T) {
	t.Run("CR+LF then CR+LF", func(t *testing.T) {
		p, w, hook := latePipe0087(t)
		write0087(t, w, key0087+"\r")
		hook.arm(writeLater0087(t, w, "\n\r\nans\n"))
		if got, err := p.Secret("Key"); err != nil || got != key0087 {
			t.Fatalf("Secret = %q, %v; want the key", got, err)
		}
		// Nothing more comes: a prompt that would wait for another line
		// meets the end instead.
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		if got, err := p.Input("Model", "saved-model"); err != nil || got != "saved-model" {
			t.Errorf("first Input = %q, %v; want its default, for the second line ending", got, err)
		}
		if got, err := p.Input("Model", "saved-model"); err != nil || got != "ans" {
			t.Errorf("second Input = %q, %v; want ans", got, err)
		}
	})
	t.Run("CR then CR, raw", func(t *testing.T) {
		p, w, hook := latePipe0087(t)
		write0087(t, w, key0087+"\r")
		hook.arm(writeLater0087(t, w, "\r"))
		if got, err := p.Secret("Key"); err != nil || got != key0087 {
			t.Fatalf("Secret = %q, %v; want the key", got, err)
		}
		write0087(t, w, "key2\r")
		if got, err := p.Secret("Key"); err != nil || got != "key2" {
			t.Errorf("second Secret = %q, %v; want key2", got, err)
		}
		if n := strings.Count(hook.String(), "empty value"); n != 1 {
			t.Errorf("%d empty-value warnings; want 1, for the kept CR", n)
		}
	})
	t.Run("CR then CR, line", func(t *testing.T) {
		p, w, hook := latePipe0087(t)
		write0087(t, w, key0087+"\r")
		hook.arm(writeLater0087(t, w, "\r"))
		if got, err := p.Secret("Key"); err != nil || got != key0087 {
			t.Fatalf("Secret = %q, %v; want the key", got, err)
		}
		write0087(t, w, "ans\n")
		if got, err := p.Input("Model", "saved-model"); err != nil || got != "ans" {
			t.Errorf("Input = %q, %v; want ans: the kept CR does not end a line", got, err)
		}
	})
}

// TestSecret_EndOfInputWhileSettling_0087 (ST8): the input ends while
// settling: Secret returns the key, the next prompt answers its default, and
// the read after that ends the input.
func TestSecret_EndOfInputWhileSettling_0087(t *testing.T) {
	p, w, hook := latePipe0087(t)
	write0087(t, w, key0087+"\r")
	hook.arm(func() {
		if err := w.Close(); err != nil {
			t.Errorf("close the writer: %v", err)
		}
	})
	if got, err := p.Secret("Key"); err != nil || got != key0087 {
		t.Fatalf("Secret = %q, %v; want the key", got, err)
	}
	if got, err := p.Input("Model", "saved-model"); err != nil || got != "saved-model" {
		t.Errorf("Input at the end = %q, %v; want its default and no error", got, err)
	}
	if _, err := p.Input("Model", "saved-model"); !errors.Is(err, io.EOF) {
		t.Errorf("Input after the end: %v; want io.EOF", err)
	}
}

// TestSecret_SettleCheckFails_0087 (ST9): a check that fails or cannot be made
// is nothing pending: Secret returns the key, and any read error is the next
// read's.
func TestSecret_SettleCheckFails_0087(t *testing.T) {
	t.Run("closed", func(t *testing.T) {
		p, w, hook := latePipe0087(t)
		write0087(t, w, key0087+"\r")
		r, ok := p.In.(*os.File)
		if !ok {
			t.Fatal("In is not the pipe")
		}
		hook.arm(func() {
			if err := r.Close(); err != nil {
				t.Errorf("close the reader: %v", err)
			}
		})
		if got, err := p.Secret("Key"); err != nil || got != key0087 {
			t.Fatalf("Secret = %q, %v; want the key", got, err)
		}
		if _, err := p.Input("Model", "saved-model"); !errors.Is(err, os.ErrClosed) {
			t.Errorf("Input on the closed reader: %v; want os.ErrClosed", err)
		}
	})
	t.Run("descriptor 1100", func(t *testing.T) {
		p, w, _ := latePipe0087(t)
		r, ok := p.In.(*os.File)
		if !ok {
			t.Fatal("In is not the pipe")
		}
		const high = 1100
		if err := unix.Dup2(int(r.Fd()), high); err != nil {
			t.Fatalf("dup2 the reader to descriptor %d (the process's descriptor limit may be below it): %v", high, err)
		}
		f := os.NewFile(uintptr(high), "pipe-"+strconv.Itoa(high))
		t.Cleanup(func() { _ = f.Close() })
		p.In = f
		write0087(t, w, key0087+"\r")
		if got, err := p.Secret("Key"); err != nil || got != key0087 {
			t.Errorf("Secret on descriptor %d = %q, %v; want the key", high, got, err)
		}
	})
}

// TestSecret_LongPasteSplitAtCR_0087 (ST33): a paste whose CR is the last byte
// of the reader's first fill and whose LF is still in the pipe: the LF is
// dropped.
func TestSecret_LongPasteSplitAtCR_0087(t *testing.T) {
	p, w, _ := latePipe0087(t)
	letters := strings.Repeat("k", 4095)
	write0087(t, w, letters+"\r\n")
	if got, err := p.Secret("Key"); err != nil || got != letters {
		t.Fatalf("Secret = %d letters, %v; want the 4095 letters", len(got), err)
	}
	write0087(t, w, "my-model\n")
	if got, err := p.Input("Model", "saved-model"); err != nil || got != "my-model" {
		t.Errorf("Input after a paste split at its CR = %q, %v; want my-model", got, err)
	}
}

// script0087 is an Out that answers the wizard's questions as they are
// printed: the line-end hook, once, at the hidden entry's "\r\n"; the search
// query at the search question; and "1" at the first menu after it.
type script0087 struct {
	t        *testing.T
	w        *os.File
	query    string
	mu       sync.Mutex
	out      strings.Builder
	lf       bool
	searched bool
	picked   bool
}

func (s *script0087) Write(b []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.out.Write(b)
	text := string(b)
	var send string
	switch {
	case !s.lf && strings.Contains(text, "\r\n"):
		s.lf, send = true, "\n"
	case !s.searched && strings.Contains(text, searchModelsPrompt):
		s.searched, send = true, s.query+"\n"
	case s.searched && !s.picked && strings.Contains(text, "Select [1-"):
		s.picked, send = true, "1\n"
	}
	if send != "" {
		if _, err := s.w.WriteString(send); err != nil {
			s.t.Errorf("script: %v", err)
		}
	}
	return len(b), nil
}

// TestConfigure_SearchAfterLateLineFeed_0087 (ST15): ConfigureLLM over a
// TextPrompter, with Claude (a key, a static catalog, Discover off): after a
// typed key whose late LF the line-end hook writes, the search question takes
// the person's query, not a blank, and the model searched for is chosen.
func TestConfigure_SearchAfterLateLineFeed_0087(t *testing.T) {
	static := catalog.Static(llmprovider.ProviderClaude)
	model := static[len(static)-1]
	rawSeam0059(t, nil, nil)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = w.Close()
		_ = r.Close()
	})
	out := &script0087{t: t, w: w, query: model}
	p := &TextPrompter{In: r, Out: out}
	write0087(t, w, strconv.Itoa(providerIdx(t, llmprovider.ProviderClaude)+1)+"\n"+key0087+"\r")
	res, err := ConfigureLLM(context.Background(), p, Options{})
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if res.Model != model {
		t.Errorf("Model = %q; want %q, the model searched for: the late LF answered the search blank", res.Model, model)
	}
}
