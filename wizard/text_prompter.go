package wizard

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/term"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
)

// minRevealRunes is the entry length below which Secret reveals nothing. Below
// it, four revealed runes would be most of a short or partially-pasted value.
const minRevealRunes = 8

// revealTail is how many trailing runes Secret shows while typing.
const revealTail = 4

// maskGlyphs is the fixed-width mask body, so the rendering does not encode the
// credential's length as it grows.
const maskGlyphs = 8

// TextPrompter is the zero-toolkit Prompter: stdlib plus golang.org/x/term,
// which this module already requires. Consumers with
// a UI toolkit implement Prompter over it instead. It is not safe for
// concurrent use: two goroutines reading at once race on its reader. Notify
// may be called while another goroutine waits in a read, as a sign-in does.
// In and Out must not change once it is in use.
//
// It does not implement InputerContext: a read of stdin cannot be given up,
// so after a browser sign-in that the loopback finished, the wizard asks for
// the Enter that ends the paste prompt (0057-MADR D3).
type TextPrompter struct {
	// In is the input source. When it is an *os.File attached to a terminal,
	// Secret uses raw mode for live masking; otherwise it degrades to a plain
	// read. Accepting io.Reader rather than *os.File lets consumers inject a
	// scripted reader in their own tests.
	In  io.Reader
	Out io.Writer

	// reader is retained across prompts. A fresh bufio.Reader per call would
	// discard whatever the previous call buffered, silently losing input.
	reader *bufio.Reader

	// mu guards writeErr and the writes to Out: a sign-in's Notify can run
	// while another goroutine waits in Input (0020-MADR F16).
	mu sync.Mutex

	// writeErr records the first failed write. A prompt whose output never
	// reached the user must not be treated as answered, so every method that
	// can return an error surfaces this rather than continuing blind.
	writeErr error

	// eof records that the input ended. The first read at the end still
	// answers its prompt's default, which scripts rely on; a read after it is
	// errExhausted, so a loop that asks again ends (0020-MADR F6).
	eof bool
}

// errExhausted is a read after the input ended. It wraps io.EOF.
var errExhausted = fmt.Errorf("wizard: input exhausted: %w", io.EOF)

// NewTextPrompter returns a TextPrompter on stdin/stdout.
func NewTextPrompter() *TextPrompter {
	return &TextPrompter{In: os.Stdin, Out: os.Stdout}
}

var _ Prompter = (*TextPrompter)(nil)

func (p *TextPrompter) out() io.Writer {
	if p.Out == nil {
		return os.Stdout
	}
	return p.Out
}

// printf writes to the output, remembering the first failure. Subsequent
// writes are skipped: once the terminal is gone, further output is noise.
func (p *TextPrompter) printf(format string, args ...any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.writeErr != nil {
		return
	}
	if _, err := fmt.Fprintf(p.out(), format, args...); err != nil {
		p.writeErr = err
	}
}

// flushErr returns and clears any recorded write failure.
func (p *TextPrompter) flushErr() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	err := p.writeErr
	p.writeErr = nil
	return err
}

func (p *TextPrompter) in() io.Reader {
	if p.In == nil {
		return os.Stdin
	}
	return p.In
}

// isTerminal, makeRaw and restoreTerm are golang.org/x/term's, as variables
// that a test replaces to run secretOnce's raw mode with no terminal
// (0059-MADR W10).
var (
	isTerminal  = term.IsTerminal
	makeRaw     = term.MakeRaw
	restoreTerm = term.Restore
)

// inTTY returns the input as a terminal file descriptor, if it is one. Raw-mode
// masking is only possible when it is.
func (p *TextPrompter) inTTY() (*os.File, bool) {
	f, ok := p.in().(*os.File)
	if !ok {
		return nil, false
	}
	return f, isTerminal(int(f.Fd()))
}

// bufReader is the reader kept across prompts.
func (p *TextPrompter) bufReader() *bufio.Reader {
	if p.reader == nil {
		p.reader = bufio.NewReader(p.in())
	}
	return p.reader
}

// readLine reads one line. It returns io.EOF alongside any final partial line
// so callers can distinguish "the user pressed enter on an empty prompt" from
// "there is no more input" — without that distinction, a re-prompting loop
// never terminates. Every read after that returns errExhausted.
func (p *TextPrompter) readLine() (string, error) {
	if p.eof {
		return "", errExhausted
	}
	line, err := p.bufReader().ReadString('\n')
	trimmed := strings.TrimSpace(line)
	if errors.Is(err, io.EOF) {
		p.eof = true
		return trimmed, io.EOF
	}
	if err != nil {
		return "", err
	}
	return trimmed, nil
}

// readFailed reports whether a readLine error ends the prompt: every error
// but the first end of input, which answers with what was read, or with the
// default.
func readFailed(err error) bool {
	return errors.Is(err, errExhausted) || err != nil && !errors.Is(err, io.EOF)
}

// Notify implements Prompter.
func (p *TextPrompter) Notify(level Level, format string, args ...any) {
	prefix := ""
	switch level {
	case LevelWarn:
		prefix = "warning: "
	case LevelError:
		prefix = "error: "
	case LevelInfo:
	}
	// A notice can carry a server's text: no escape sequence reaches the
	// terminal (0021-MADR Z3).
	p.printf("%s%s\n", prefix, redact.StripControl(fmt.Sprintf(format, args...)))
}

// maxLabelPad is the widest label, in runes, a menu pads to align its
// details; a longer label is not padded.
const maxLabelPad = 40

// renderChoices prints the menu, marking the selected rows. The details are
// a column: each number right-aligned to the widest, and each label of a
// row with a detail padded to the longest such label, at most maxLabelPad
// runes (0061-MADR, amendment "Model search, rebuilt", S4).
func (p *TextPrompter) renderChoices(title string, choices []Choice, selected []int) {
	p.printf("\n%s\n", title)
	numWidth := len(strconv.Itoa(len(choices)))
	pad := 0
	for _, c := range choices {
		if n := utf8.RuneCountInString(c.Label); c.Detail != "" && n <= maxLabelPad {
			pad = max(pad, n)
		}
	}
	for i, c := range choices {
		mark := ""
		if slices.Contains(selected, i) {
			mark = " (selected)"
		}
		if c.Detail != "" {
			fill := strings.Repeat(" ", max(0, pad-utf8.RuneCountInString(c.Label)))
			p.printf("  %*d) %s%s — %s%s\n", numWidth, i+1, c.Label, fill, c.Detail, mark)
		} else {
			p.printf("  %*d) %s%s\n", numWidth, i+1, c.Label, mark)
		}
	}
}

// Select implements Prompter. No choices is a plain error. A failed read of
// In or write to Out is returned, matching the reader's or writer's error; a
// read after the input has ended matches io.EOF; an answer that names no
// choice on the input's last line is a plain error.
func (p *TextPrompter) Select(title string, choices []Choice, defaultIdx int) (int, error) {
	if len(choices) == 0 {
		return 0, fmt.Errorf("wizard: Select called with no choices")
	}
	if defaultIdx < 0 || defaultIdx >= len(choices) {
		defaultIdx = 0
	}
	for {
		p.renderChoices(title, choices, nil)
		p.printf("Select [1-%d] (default %d): ", len(choices), defaultIdx+1)
		line, err := p.readLine()
		if readFailed(err) {
			return 0, err
		}
		exhausted := errors.Is(err, io.EOF)
		if line == "" {
			return defaultIdx, p.flushErr()
		}
		n, convErr := strconv.Atoi(line)
		if convErr == nil && n >= 1 && n <= len(choices) {
			return n - 1, p.flushErr()
		}
		if exhausted {
			return 0, fmt.Errorf("wizard: input exhausted while selecting")
		}
		p.Notify(LevelWarn, "enter a number between 1 and %d", len(choices))
	}
}

// MultiSelect implements Prompter. Input is a comma-separated list of indices,
// a repeated index counting once (MADR 0013 C1); an empty line accepts the
// preselection. A preselection is marked, and 0 clears it (0020-MADR F50). No
// choices returns nil and no error. A failed read of In or write to Out is
// returned, matching the reader's or writer's error; a read after the input
// has ended matches io.EOF; an answer that names no choice on the input's
// last line is a plain error.
func (p *TextPrompter) MultiSelect(title string, choices []Choice, preselected []int) ([]int, error) {
	if len(choices) == 0 {
		return nil, nil
	}
	hint := "blank for none"
	if len(preselected) > 0 {
		numbers := make([]string, 0, len(preselected))
		for _, i := range preselected {
			numbers = append(numbers, strconv.Itoa(i+1))
		}
		hint = "blank keeps " + strings.Join(numbers, ",") + "; 0 for none"
	}
	for {
		p.renderChoices(title, choices, preselected)
		p.printf("Select any (comma-separated, e.g. 1,3; %s): ", hint)
		line, err := p.readLine()
		if readFailed(err) {
			return nil, err
		}
		exhausted := errors.Is(err, io.EOF)
		if line == "" {
			return preselected, p.flushErr()
		}
		if line == "0" && len(preselected) > 0 {
			return []int{}, p.flushErr()
		}
		var out []int
		ok := true
		for part := range strings.SplitSeq(line, ",") {
			n, convErr := strconv.Atoi(strings.TrimSpace(part))
			if convErr != nil || n < 1 || n > len(choices) {
				ok = false
				break
			}
			if !slices.Contains(out, n-1) {
				out = append(out, n-1)
			}
		}
		if ok {
			return out, p.flushErr()
		}
		if exhausted {
			return nil, fmt.Errorf("wizard: input exhausted while selecting")
		}
		p.Notify(LevelWarn, "enter comma-separated numbers between 1 and %d", len(choices))
	}
}

// Confirm implements Prompter. An empty answer is def, and an answer other
// than y or yes is false, with no error. A failed read of In or write to Out
// is returned, matching the
// reader's or writer's error; a read after the input has ended matches
// io.EOF.
func (p *TextPrompter) Confirm(question string, def bool) (bool, error) {
	hint := "[y/N]"
	if def {
		hint = "[Y/n]"
	}
	p.printf("%s %s: ", question, hint)
	line, err := p.readLine()
	if readFailed(err) {
		return false, err
	}
	if wErr := p.flushErr(); wErr != nil {
		return false, wErr
	}
	switch strings.ToLower(line) {
	case "":
		return def, nil
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

// Input implements Prompter. A failed read of In or write to Out is returned,
// matching the reader's or writer's error; a read after the input has ended
// matches io.EOF.
func (p *TextPrompter) Input(prompt, def string) (string, error) {
	if def != "" {
		p.printf("%s (default %s): ", prompt, def)
	} else {
		p.printf("%s: ", prompt)
	}
	line, err := p.readLine()
	if readFailed(err) {
		return "", err
	}
	if wErr := p.flushErr(); wErr != nil {
		return "", wErr
	}
	if line == "" {
		return def, nil
	}
	return line, nil
}

// renderSecret produces the masked view of an in-progress credential entry.
//
// The last four runes are revealed LIVE, so a paste is confirmed the instant it
// lands and the key is identifiable without pressing Enter. Below
// minRevealRunes nothing is revealed: on a short or partially-pasted value,
// four runes would be most of the secret.
//
// This is split from the terminal I/O so it is testable without a TTY.
func renderSecret(entered []rune) string {
	if len(entered) < minRevealRunes {
		return strings.Repeat("•", len(entered))
	}
	return strings.Repeat("•", maskGlyphs) + string(entered[len(entered)-revealTail:])
}

// Secret implements Prompter with live masking.
//
// Raw mode is REQUIRED for live masking and it fails on Git Bash / mintty. That
// fallback is not optional: without it those users cannot configure at all. On
// failure this prints the documented notice and falls back to a plain visible
// read — the behaviour prepare-commit-msg has today, preserved here as the
// shared baseline.
//
// The live tail is a deliberate exposure: four characters are on screen for the
// whole entry, visible in screen shares and recordings. See MADR 0005
// revision 2, "Accepted trade-off".
func (p *TextPrompter) Secret(prompt string) (string, error) {
	for {
		value, err := p.secretOnce(prompt)
		// A failed write is returned now, as Input returns it, not at the
		// next prompt; a value read with a failing reader is not kept. The
		// first end of input answers with what was read (0026-MADR F49).
		if wErr := p.flushErr(); wErr != nil {
			return "", wErr
		}
		if value != "" && !readFailed(err) {
			return value, nil
		}
		if err != nil {
			// Includes io.EOF: input is exhausted, so re-prompting would spin
			// forever. Report rather than loop.
			return "", fmt.Errorf("wizard: no value entered: %w", err)
		}
		p.Notify(LevelWarn, "empty value; please try again")
	}
}

func (p *TextPrompter) secretOnce(prompt string) (string, error) {
	f, isTTY := p.inTTY()
	if !isTTY {
		// Not a terminal (piped input, CI, an injected reader): read plainly.
		// Nothing is echoed by us, so there is no masking to do.
		p.printf("%s: ", prompt)
		return p.readLine()
	}
	fd := int(f.Fd())

	state, err := makeRaw(fd)
	if err != nil {
		// Git Bash / mintty and similar: raw mode is unavailable. Falling
		// back keeps configuration possible; failing here would not.
		p.printf("(hidden input unavailable: %v — typing will be visible)\n", err)
		p.printf("%s: ", prompt)
		return p.readLine()
	}
	defer func() {
		// A terminal left in raw mode is unusable, so surface the failure
		// rather than discarding it.
		if rErr := restoreTerm(fd, state); rErr != nil {
			p.Notify(LevelError, "failed to restore terminal mode: %v", rErr)
		}
	}()

	p.printf("%s: ", prompt)
	return p.readMasked(prompt)
}

// readMasked reads one masked entry in raw mode, rune by rune, redrawing the
// masked view after each change. It is split from the terminal so it is
// testable without one. Raw mode does not translate output, so a line ends
// with \r\n. The entry is trimmed, so a paste's spaces are not part of it
// (0020-MADR F17, F18).
//
// The view is redrawn only when the reader has nothing buffered, so a paste
// is drawn once, not once per character, and a shorter view is padded with
// spaces to the previous width, not erased with \033[K, which some consoles
// lack. Ctrl-U clears the entry, and Ctrl-C cancels it with an error that
// matches context.Canceled (0021-MADR Z6). A cursor or function key adds
// nothing, however many reads its sequence is split across (escapeState;
// 0059-MADR W4).
func (p *TextPrompter) readMasked(prompt string) (string, error) {
	if p.eof {
		return "", errExhausted
	}
	r := p.bufReader()
	var entered []rune
	width := 0 // the masked view's width on screen, in runes
	esc := escNone
	redraw := func() {
		view := renderSecret(entered)
		n := utf8.RuneCountInString(view)
		pad := max(0, width-n)
		p.printf("\r%s: %s%s%s", prompt, view, strings.Repeat(" ", pad), strings.Repeat("\b", pad))
		width = n
	}
	for {
		c, _, readErr := r.ReadRune()
		if readErr != nil {
			return p.endMasked(readErr)
		}
		if esc != escNone {
			var consumed bool
			if esc, consumed = esc.next(c); consumed {
				continue
			}
		}
		switch {
		case c == '\r' || c == '\n':
			// Enter sends \r. A paste may follow it with a \n, read with
			// it: drop that, or it answers the next prompt. A \n not yet
			// read is not waited for: it is the user's next Enter.
			if c == '\r' && r.Buffered() > 0 {
				if next, err := r.Peek(1); err == nil && next[0] == '\n' {
					if _, err := r.Discard(1); err != nil {
						return p.endMasked(err)
					}
				}
			}
			p.printf("\r\n")
			return strings.TrimSpace(string(entered)), nil
		case c == 3: // Ctrl-C
			p.printf("\r\n")
			return "", fmt.Errorf("wizard: cancelled: %w", context.Canceled)
		case c == 0x15: // Ctrl-U — clear the entry
			entered = entered[:0]
		case c == 127 || c == 8: // DEL, Backspace — remove one rune
			if len(entered) > 0 {
				entered = entered[:len(entered)-1]
			}
		case c == 0x1b: // a cursor or function key, Alt and a key, or Escape
			esc = escLone
			if r.Buffered() > 0 {
				esc = escAlt
			}
			continue
		case c < 32: // other control characters
			continue
		default:
			entered = append(entered, c)
		}
		// Redraw the whole line so the revealed tail updates as it moves,
		// once the reader has caught up.
		if r.Buffered() == 0 {
			redraw()
		}
	}
}

// endMasked ends a masked entry on a read error, remembering the end of
// input. The entry was not finished with Enter, so it is not returned: a
// hang-up mid-paste must not save a truncated key (0026-MADR F50).
func (p *TextPrompter) endMasked(err error) (string, error) {
	if errors.Is(err, io.EOF) {
		p.eof = true
	}
	p.printf("\r\n")
	return "", err
}

// escapeState is where readMasked is in an escape sequence. It is kept
// across reads, so that a sequence split between them, as a bracketed paste's
// markers can be, adds nothing to the entry (0059-MADR W4).
type escapeState int

const (
	escNone escapeState = iota
	escLone             // ESC with nothing buffered behind it: the Escape key, or a split sequence
	escAlt              // ESC with bytes buffered behind it
	escCSI              // ESC [ and the parameters so far
	escSS3              // ESC O
)

// next takes c in state s: the state after it, and whether c was part of the
// sequence. After ESC, a [ begins a CSI, ended by a final byte in 0x40-0x7E,
// and an O an SS3, ended by one more character. Any other character after an
// ESC with bytes buffered behind it, such as Alt with a key, is dropped with
// it; one after a lone ESC is a key typed after the Escape key, and is kept
// (0021-MADR Z6).
func (s escapeState) next(c rune) (escapeState, bool) {
	switch s {
	case escLone, escAlt:
		switch c {
		case '[':
			return escCSI, true
		case 'O':
			return escSS3, true
		}
		return escNone, s == escAlt
	case escCSI:
		if c >= 0x40 && c <= 0x7e {
			return escNone, true
		}
		return escCSI, true
	case escSS3:
		return escNone, true
	}
	return escNone, false
}
