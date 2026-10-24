package wizard

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

// 0059-MADR W4 and W5: an escape sequence split across reads, and a paste
// prompt whose prompter answers at once with nothing.

// TestReadMasked_SplitEscape_0059 (P3): a bracketed paste's marker or a
// cursor key split between reads adds nothing to the entry.
func TestReadMasked_SplitEscape_0059(t *testing.T) {
	for name, chunks := range map[string][]string{
		"the end marker after its ESC":   {"sk-ant-keyAA\x1b", "[201~\r"},
		"the start marker after its ESC": {"\x1b", "[200~sk-ant-keyAA\r"},
		"an arrow after its [":           {"sk-ant-key", "\x1b[", "D", "AA\r"},
		"an SS3 key after its ESC":       {"sk-ant-key\x1b", "OA", "AA\r"},
	} {
		p := &TextPrompter{In: &chunkReader{chunks: chunks}, Out: &bytes.Buffer{}}
		if got, err := p.readMasked("Key"); err != nil || got != "sk-ant-keyAA" {
			t.Errorf("%s: readMasked = %q, %v; want sk-ant-keyAA", name, got, err)
		}
	}
}

// emptyInput0059 answers every Input with nothing, after delay, counting the
// calls; the rest is fakePrompter's.
type emptyInput0059 struct {
	*fakePrompter
	delay    time.Duration
	answer   string // returned once calls reach answerAt, when answerAt > 0
	calls    int
	answerAt int
}

func (e *emptyInput0059) Input(string, string) (string, error) {
	e.calls++
	time.Sleep(e.delay)
	if e.answerAt > 0 && e.calls == e.answerAt {
		return e.answer, nil
	}
	return "", nil
}

// paste0059 is a paste prompt over p, its URL shown.
func paste0059(p Prompter) *pastePrompt {
	s := &pastePrompt{p: p, shown: make(chan struct{}), prompt: pasteCodePrompt}
	close(s.shown)
	return s
}

// TestPaste_QuickEmptiesStop_0059 (P4): a prompter that answers at once with
// nothing is asked three times, says so once, and the input ends with ctx.
func TestPaste_QuickEmptiesStop_0059(t *testing.T) {
	e := &emptyInput0059{fakePrompter: newFake(t, fakePrompter{})}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := paste0059(e).input(ctx)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) < 100*time.Millisecond {
		t.Errorf("input: %v after %v; want ctx's deadline, once it ended", err, time.Since(start))
	}
	if e.calls != pasteQuickEmpties || countContaining(e.seenNotify, "answers at once") != 1 {
		t.Errorf("%d Input calls, notices %q; want %d and one notice", e.calls, e.seenNotify, pasteQuickEmpties)
	}
}

// TestPaste_SlowEmptyAskedAgain_0059: an empty answer slower than pasteQuick
// is a person's Enter, asked again past the count, so the paste that follows
// is read.
func TestPaste_SlowEmptyAskedAgain_0059(t *testing.T) {
	original := pasteQuick
	t.Cleanup(func() { pasteQuick = original })
	pasteQuick = 5 * time.Millisecond
	e := &emptyInput0059{fakePrompter: newFake(t, fakePrompter{}), delay: 10 * time.Millisecond,
		answer: "the-code", answerAt: pasteQuickEmpties + 1}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	got, err := paste0059(e).input(ctx)
	if err != nil || got != "the-code" || e.calls != pasteQuickEmpties+1 || len(e.seenNotify) != 0 {
		t.Errorf("input = %q, %v after %d calls, notices %q; want the code on call %d, no notice",
			got, err, e.calls, e.seenNotify, pasteQuickEmpties+1)
	}
}
