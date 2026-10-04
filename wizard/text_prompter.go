package wizard

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/term"
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
// a UI toolkit implement Prompter over it instead.
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

// inTTY returns the input as a terminal file descriptor, if it is one. Raw-mode
// masking is only possible when it is.
func (p *TextPrompter) inTTY() (*os.File, bool) {
	f, ok := p.in().(*os.File)
	if !ok {
		return nil, false
	}
	return f, term.IsTerminal(int(f.Fd()))
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
	p.printf("%s%s\n", prefix, fmt.Sprintf(format, args...))
}

// renderChoices prints the menu, marking the selected rows.
func (p *TextPrompter) renderChoices(title string, choices []Choice, selected []int) {
	p.printf("\n%s\n", title)
	for i, c := range choices {
		mark := ""
		if slices.Contains(selected, i) {
			mark = " (selected)"
		}
		if c.Detail != "" {
			p.printf("  %d) %s — %s%s\n", i+1, c.Label, c.Detail, mark)
		} else {
			p.printf("  %d) %s%s\n", i+1, c.Label, mark)
		}
	}
}

// Select implements Prompter.
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
// preselection. A preselection is marked, and 0 clears it (0020-MADR F50).
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

// Confirm implements Prompter.
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

// Input implements Prompter.
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
		if value != "" {
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

	state, err := term.MakeRaw(fd)
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
		if rErr := term.Restore(fd, state); rErr != nil {
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
func (p *TextPrompter) readMasked(prompt string) (string, error) {
	if p.eof {
		return "", errExhausted
	}
	r := p.bufReader()
	var entered []rune
	for {
		c, _, readErr := r.ReadRune()
		if readErr != nil {
			return p.endMasked(entered, readErr)
		}
		switch {
		case c == '\r' || c == '\n':
			// Enter sends \r. A paste may follow it with a \n, read with
			// it: drop that, or it answers the next prompt. A \n not yet
			// read is not waited for: it is the user's next Enter.
			if c == '\r' && r.Buffered() > 0 {
				if next, err := r.Peek(1); err == nil && next[0] == '\n' {
					if _, err := r.Discard(1); err != nil {
						return p.endMasked(entered, err)
					}
				}
			}
			p.printf("\r\n")
			return strings.TrimSpace(string(entered)), nil
		case c == 3: // Ctrl-C
			p.printf("\r\n")
			return "", fmt.Errorf("wizard: cancelled")
		case c == 127 || c == 8: // DEL, Backspace — remove one rune
			if len(entered) > 0 {
				entered = entered[:len(entered)-1]
			}
		case c == 0x1b: // a cursor or function key
			if err := skipEscape(r); err != nil {
				return p.endMasked(entered, err)
			}
			continue
		case c < 32: // other control characters
			continue
		default:
			entered = append(entered, c)
		}
		// Redraw the whole line so the revealed tail updates as it moves.
		p.printf("\r\033[K%s: %s", prompt, renderSecret(entered))
	}
}

// endMasked ends a masked entry on a read error, remembering the end of
// input.
func (p *TextPrompter) endMasked(entered []rune, err error) (string, error) {
	if errors.Is(err, io.EOF) {
		p.eof = true
	}
	p.printf("\r\n")
	return strings.TrimSpace(string(entered)), err
}

// skipEscape consumes the rest of an escape sequence after ESC, so a cursor
// or function key adds nothing to the entry: a CSI (ESC [, parameters, a
// final byte in 0x40-0x7E) or an SS3 (ESC O and one byte). Any other
// character after ESC, such as Alt with a key, is dropped with it.
func skipEscape(r *bufio.Reader) error {
	c, _, err := r.ReadRune()
	if err != nil {
		return err
	}
	switch c {
	case '[':
		for {
			b, err := r.ReadByte()
			if err != nil {
				return err
			}
			if b >= 0x40 && b <= 0x7e {
				return nil
			}
		}
	case 'O':
		_, err = r.ReadByte()
	}
	return err
}
