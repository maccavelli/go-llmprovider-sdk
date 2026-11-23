package wizard

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// The tests of 0087-PLAN Phase 2 that run on every platform. Each reads a
// hidden entry through TextPrompter's raw path (rawSeam0059) on an os.Pipe,
// whose writer stays open unless a test closes it, so that a read waits for
// the next byte as a terminal's does. "The line-end hook" (lineEnd0087) is an
// Out that writes given bytes into the pipe once, when the prompter prints
// "\r\n": they arrive after the CR is read and before Secret returns, as a
// terminal's CR+LF does (0087-MADR, "What option A is").

// lineEnd0087 is the line-end hook, and the prompter's output.
type lineEnd0087 struct {
	mu    sync.Mutex
	out   bytes.Buffer
	armed func()
}

func (h *lineEnd0087) Write(b []byte) (int, error) {
	h.mu.Lock()
	h.out.Write(b)
	fire := h.armed
	if fire != nil && bytes.Contains(b, []byte("\r\n")) {
		h.armed = nil
	} else {
		fire = nil
	}
	h.mu.Unlock()
	if fire != nil {
		fire()
	}
	return len(b), nil
}

// arm makes the next "\r\n" the prompter prints run fn, once.
func (h *lineEnd0087) arm(fn func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.armed = fn
}

func (h *lineEnd0087) String() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.out.String()
}

// latePipe0087 is a TextPrompter on an os.Pipe under rawSeam0059, its writer
// open, and its line-end hook unarmed.
func latePipe0087(t *testing.T) (p *TextPrompter, w *os.File, hook *lineEnd0087) {
	t.Helper()
	rawSeam0059(t, nil, nil)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = w.Close()
		_ = r.Close()
	})
	hook = &lineEnd0087{}
	return &TextPrompter{In: r, Out: hook}, w, hook
}

// write0087 writes s to w, failing the test on an error.
func write0087(t *testing.T, w *os.File, s string) {
	t.Helper()
	if _, err := w.WriteString(s); err != nil {
		t.Fatalf("write to the pipe: %v", err)
	}
}

// writeLater0087 is a hook action that writes s to w.
func writeLater0087(t *testing.T, w *os.File, s string) func() {
	return func() {
		if _, err := w.WriteString(s); err != nil {
			t.Errorf("line-end hook: %v", err)
		}
	}
}

const key0087 = "sk-late-lf-0087-keyAA"

// TestSecret_NextEnterKept_0087 (ST4): an LF that arrives after Secret has
// returned is the person's next Enter at the next prompt, and answers it with
// its default (0020-MADR F17).
func TestSecret_NextEnterKept_0087(t *testing.T) {
	p, w, _ := latePipe0087(t)
	write0087(t, w, key0087+"\r")
	if got, err := p.Secret("Key"); err != nil || got != key0087 {
		t.Fatalf("Secret = %q, %v; want the key", got, err)
	}
	write0087(t, w, "\nans\n")
	if got, err := p.Input("Model", "saved-model"); err != nil || got != "saved-model" {
		t.Errorf("Input after a later Enter = %q, %v; want its default", got, err)
	}
}

// TestSecret_TypeAheadKept_0087 (ST5): a byte other than an LF that follows
// the CR within the wait stays for the next prompt.
func TestSecret_TypeAheadKept_0087(t *testing.T) {
	p, w, hook := latePipe0087(t)
	write0087(t, w, key0087+"\r")
	hook.arm(writeLater0087(t, w, "3\n"))
	if got, err := p.Secret("Key"); err != nil || got != key0087 {
		t.Fatalf("Secret = %q, %v; want the key", got, err)
	}
	choices := []Choice{{Label: "a"}, {Label: "b"}, {Label: "c"}}
	if got, err := p.Select("Pick", choices, 0); err != nil || got != 2 {
		t.Errorf("Select after typed-ahead 3 = %d, %v; want 2", got, err)
	}
}

// TestSecret_SettleIsBounded_0087 (ST1; on Windows also ST31): with nothing
// after the CR, Secret returns within 1 s, waited for at most 5 s.
func TestSecret_SettleIsBounded_0087(t *testing.T) {
	p, w, _ := latePipe0087(t)
	write0087(t, w, key0087+"\r")
	type result struct {
		key string
		err error
	}
	done := make(chan result, 1)
	start := time.Now()
	go func() {
		k, err := p.Secret("Key")
		done <- result{k, err}
	}()
	select {
	case r := <-done:
		if r.err != nil || r.key != key0087 {
			t.Errorf("Secret = %q, %v; want the key", r.key, r.err)
		}
		if d := time.Since(start); d > time.Second {
			t.Errorf("Secret took %v with nothing after the CR; want under 1 s", d)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Secret did not return within 5 s with nothing after the CR")
	}
}

// TestSecret_LineFeedEndedEntry_0087 (ST7): an entry ended by an LF is not
// settled; a CR that follows stays: a line-mode prompt trims it, and a raw
// prompt reads an empty entry and warns, as before.
func TestSecret_LineFeedEndedEntry_0087(t *testing.T) {
	t.Run("line mode", func(t *testing.T) {
		p, w, hook := latePipe0087(t)
		write0087(t, w, key0087+"\n")
		hook.arm(writeLater0087(t, w, "\r"))
		if got, err := p.Secret("Key"); err != nil || got != key0087 {
			t.Fatalf("Secret = %q, %v; want the key", got, err)
		}
		write0087(t, w, "ans\n")
		if got, err := p.Input("Model", "saved-model"); err != nil || got != "ans" {
			t.Errorf("Input = %q, %v; want ans, its CR trimmed", got, err)
		}
	})
	t.Run("raw mode", func(t *testing.T) {
		p, w, hook := latePipe0087(t)
		write0087(t, w, key0087+"\n")
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
}

// TestSecret_PipeCRLF_0087 (ST11): a pipe is not a terminal: no raw mode, no
// settle, and each line is trimmed, as before.
func TestSecret_PipeCRLF_0087(t *testing.T) {
	p, _ := pipe0059(t, key0087+"\r\nmy-model\r\n")
	if got, err := p.Secret("Key"); err != nil || got != key0087 {
		t.Fatalf("Secret = %q, %v; want the key", got, err)
	}
	if got, err := p.Input("Model", "saved-model"); err != nil || got != "my-model" {
		t.Errorf("Input = %q, %v; want my-model", got, err)
	}
}

// TestSecret_AfterLineAnswer_0087 (ST13): a line-mode answer takes its LF, so
// the hidden entry after it gets only what follows.
func TestSecret_AfterLineAnswer_0087(t *testing.T) {
	p, w, hook := latePipe0087(t)
	write0087(t, w, "y\n"+key0087+"\r")
	if got, err := p.Confirm("Go on?", false); err != nil || !got {
		t.Fatalf("Confirm = %v, %v; want true", got, err)
	}
	if got, err := p.Secret("Key"); err != nil || got != key0087 {
		t.Fatalf("Secret = %q, %v; want the key", got, err)
	}
	if strings.Contains(hook.String(), "empty value") {
		t.Errorf("an empty-value warning after a line answer: %q", hook.String())
	}
}

// TestSecret_LateNulKept_0087 (ST24): a late NUL after the CR is not dropped:
// it reaches the next line-mode prompt untrimmed.
func TestSecret_LateNulKept_0087(t *testing.T) {
	p, w, hook := latePipe0087(t)
	write0087(t, w, key0087+"\r")
	hook.arm(writeLater0087(t, w, "\x00ans\n"))
	if got, err := p.Secret("Key"); err != nil || got != key0087 {
		t.Fatalf("Secret = %q, %v; want the key", got, err)
	}
	if got, err := p.Input("Model", "saved-model"); err != nil || got != "\x00ans" {
		t.Errorf("Input = %q, %v; want the NUL and ans", got, err)
	}
}
