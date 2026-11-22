package llmprovider_test

import (
	"bufio"
	"encoding/hex"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"
)

// This file holds the part of 0087-PLAN's Phase 1 probe,
// TestLive_RawEnterBytes_0087 (live_raw_enter_0087_windows_test.go), that a
// test can run anywhere: how one case reads its input through a
// bufio.Reader (rawEnterRun0087) and how what it read becomes the one
// allowlisted line the probe writes (rawEnterLine0087). Its offline twin,
// TestRawEnterBytesFormat_0087, runs both over an injected reader that holds
// letters planted as secrets, and checks that only the mode, the console
// kind, the case name, the result word and line-ending bytes are printed
// (0087-MADR, "Windows: the probe and what each observation decides").

// The case names: P1 and R1 to R3 of 0087-MADR. P3 and R3 share one name.
const (
	rawEnterCaseCRLF0087   = "crlf-one-write"
	rawEnterCaseLate0087   = "lf-after-cr"
	rawEnterCaseCooked0087 = "cooked-then-raw"
	rawEnterCaseEnter0087  = "enter"
	rawEnterCasePaste0087  = "paste"
)

// The faults a case reports in place of a result word.
const (
	rawEnterNoConsole0087     = "no-console"
	rawEnterModeRefused0087   = "mode-refused"
	rawEnterCookedTimeout0087 = "cooked-timeout"
	rawEnterNoPty0087         = "no-pty"
	rawEnterNoLine0087        = "no-line"
	rawEnterBadLine0087       = "bad-line"
)

// rawEnterLineRE is the allowlist: a line the probe writes matches it, and
// nothing else is written.
var rawEnterLineRE = regexp.MustCompile(`^mode=(pty|console) console=(pty|conhost|terminal) ` +
	`case=(crlf-one-write|lf-after-cr|cooked-then-raw|enter|paste) ` +
	`result=(split|together|cr-only|lf-only|other|none|letters-first|lf-first|cr-first|nul-first|other-first|` +
	`no-console|mode-refused|cooked-timeout|no-pty|no-line|bad-line)` +
	`( cooked=(none|-|(0d|0a)+))? reads=(none|(-|(0d|0a)+)(,(-|(0d|0a)+))*)$`)

// rawEnterAllowed0087 reports whether line is one the probe may write.
func rawEnterAllowed0087(line string) bool { return rawEnterLineRE.MatchString(line) }

// rawEnterEndings0087 is the hexadecimal of the CR and LF bytes of b, in
// order, and "-" when it holds neither: no other byte of b is printed.
func rawEnterEndings0087(b []byte) string {
	var kept []byte
	for _, c := range b {
		if c == '\r' || c == '\n' {
			kept = append(kept, c)
		}
	}
	if len(kept) == 0 {
		return "-"
	}
	return hex.EncodeToString(kept)
}

// rawEnterResult0087 is the result word for what a case read. For the
// line-ending cases: "split" when a read ends with a CR and the next read
// begins with an LF (0087-MADR's mapping); else "together" when a read holds
// a CR then an LF; else "cr-only", "lf-only", "other" or "none". For
// cooked-then-raw: what the first raw read begins with.
func rawEnterResult0087(name string, reads [][]byte) string {
	if len(reads) == 0 {
		return "none"
	}
	if name == rawEnterCaseCooked0087 {
		switch c := reads[0][0]; {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			return "letters-first"
		case c == '\n':
			return "lf-first"
		case c == '\r':
			return "cr-first"
		case c == 0:
			return "nul-first"
		default:
			return "other-first"
		}
	}
	for i := 0; i+1 < len(reads); i++ {
		if a, b := reads[i], reads[i+1]; a[len(a)-1] == '\r' && b[0] == '\n' {
			return "split"
		}
	}
	var cr, lf, together bool
	for _, r := range reads {
		cr = cr || strings.IndexByte(string(r), '\r') >= 0
		lf = lf || strings.IndexByte(string(r), '\n') >= 0
		together = together || strings.Contains(string(r), "\r\n")
	}
	switch {
	case together:
		return "together"
	case cr && !lf:
		return "cr-only"
	case lf && !cr:
		return "lf-only"
	case cr && lf:
		return "other"
	default:
		return "none"
	}
}

// rawEnterLine0087 is the one line a case writes. A fault replaces the
// result word. cooked is printed for cooked-then-raw only.
func rawEnterLine0087(mode, console, name, fault string, cooked []byte, reads [][]byte) string {
	result := fault
	if result == "" {
		result = rawEnterResult0087(name, reads)
	}
	line := "mode=" + mode + " console=" + console + " case=" + name + " result=" + result
	if name == rawEnterCaseCooked0087 {
		c := "none"
		if cooked != nil {
			c = rawEnterEndings0087(cooked)
		}
		line += " cooked=" + c
	}
	groups := make([]string, 0, len(reads))
	for _, r := range reads {
		groups = append(groups, rawEnterEndings0087(r))
	}
	if len(groups) == 0 {
		return line + " reads=none"
	}
	return line + " reads=" + strings.Join(groups, ",")
}

// rawEnterRun0087 runs one case on br, which reads the console (or, in the
// twin, an injected reader). For cooked-then-raw it first puts the input in
// line mode (setLine), calls ready("line") and reads one line with
// ReadString, as TextPrompter's line-mode prompts do; then, for every case,
// it puts the input in raw mode (setRaw), starts one goroutine that reads br
// in 4096-byte reads, calls ready("raw"), and keeps each read as one group
// until quiet has passed with no read, firstWait has passed with none at
// all, or the input ends. The goroutine is not stopped: the probe's child
// process exits after one case.
func rawEnterRun0087(br *bufio.Reader, name string, setRaw, setLine func() error, ready func(stage string),
	firstWait, quiet time.Duration) (cooked []byte, reads [][]byte, fault string) {
	if name == rawEnterCaseCooked0087 {
		if setLine() != nil {
			return nil, nil, rawEnterModeRefused0087
		}
		ready("line")
		lines := make(chan string, 1)
		go func() {
			s, err := br.ReadString('\n')
			if err != nil && s == "" {
				close(lines)
				return
			}
			lines <- s
		}()
		select {
		case s, ok := <-lines:
			if ok {
				cooked = []byte(s)
			}
		case <-time.After(firstWait):
			return nil, nil, rawEnterCookedTimeout0087
		}
	}
	if setRaw() != nil {
		return cooked, nil, rawEnterModeRefused0087
	}
	groups := make(chan []byte, 64)
	go func() {
		defer close(groups)
		for {
			buf := make([]byte, 4096)
			n, err := br.Read(buf)
			if n > 0 {
				groups <- buf[:n]
			}
			if err != nil {
				return
			}
		}
	}()
	ready("raw")
	timer := time.NewTimer(firstWait)
	defer timer.Stop()
	for {
		select {
		case g, ok := <-groups:
			if !ok {
				return cooked, reads, ""
			}
			reads = append(reads, g)
			timer.Reset(quiet)
		case <-timer.C:
			return cooked, reads, ""
		}
	}
}

// chunks0087 hands over one chunk per Read, then io.EOF: each chunk is one
// read of the console.
type chunks0087 struct{ chunks []string }

func (c *chunks0087) Read(p []byte) (int, error) {
	if len(c.chunks) == 0 {
		return 0, io.EOF
	}
	n := copy(p, c.chunks[0])
	c.chunks[0] = c.chunks[0][n:]
	if c.chunks[0] == "" {
		c.chunks = c.chunks[1:]
	}
	return n, nil
}

// TestRawEnterBytesFormat_0087 is the probe's offline twin (0087-PLAN Phase
// 1 step 2): every case's input carries planted letters, and every line must
// be exactly the expected one, pass the allowlist, and hold none of them.
func TestRawEnterBytesFormat_0087(t *testing.T) {
	const secret = "SECRETzq"
	cases := []struct {
		name, mode, console, caseName string
		chunks                        []string
		stages                        []string
		want                          string
	}{
		{"split", "pty", "pty", rawEnterCaseLate0087,
			[]string{"sk-0087-" + secret + "\r", "\n"}, []string{"raw"},
			"mode=pty console=pty case=lf-after-cr result=split reads=0d,0a"},
		{"together", "console", "terminal", rawEnterCasePaste0087,
			[]string{secret + "\r\n" + secret + "\r\n"}, []string{"raw"},
			"mode=console console=terminal case=paste result=together reads=0d0a0d0a"},
		{"cr only", "console", "conhost", rawEnterCaseEnter0087,
			[]string{"\r"}, []string{"raw"},
			"mode=console console=conhost case=enter result=cr-only reads=0d"},
		{"letters then CR in two reads", "pty", "pty", rawEnterCaseCRLF0087,
			[]string{secret, "\r"}, []string{"raw"},
			"mode=pty console=pty case=crlf-one-write result=cr-only reads=-,0d"},
		{"cooked then a late LF", "pty", "pty", rawEnterCaseCooked0087,
			[]string{"y\r\n", "\n" + secret + "\r"}, []string{"line", "raw"},
			"mode=pty console=pty case=cooked-then-raw result=lf-first cooked=0d0a reads=0a0d"},
		{"cooked then letters buffered", "console", "conhost", rawEnterCaseCooked0087,
			[]string{"y\r\n" + secret + "\r"}, []string{"line", "raw"},
			"mode=console console=conhost case=cooked-then-raw result=letters-first cooked=0d0a reads=0d"},
		{"nothing", "pty", "pty", rawEnterCaseCRLF0087,
			nil, []string{"raw"},
			"mode=pty console=pty case=crlf-one-write result=none reads=none"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stages []string
			var modes []string
			br := bufio.NewReaderSize(&chunks0087{chunks: tc.chunks}, 4096)
			cooked, reads, fault := rawEnterRun0087(br, tc.caseName,
				func() error { modes = append(modes, "raw"); return nil },
				func() error { modes = append(modes, "line"); return nil },
				func(s string) { stages = append(stages, s) },
				2*time.Second, 2*time.Second)
			got := rawEnterLine0087(tc.mode, tc.console, tc.caseName, fault, cooked, reads)
			if got != tc.want {
				t.Errorf("line = %q, want %q", got, tc.want)
			}
			if !rawEnterAllowed0087(got) {
				t.Errorf("line %q is not allowlisted", got)
			}
			if strings.Contains(got, "SECRET") || strings.Contains(got, "zq") || strings.Contains(got, "sk-") {
				t.Errorf("line %q prints a letter it read", got)
			}
			if strings.Join(stages, ",") != strings.Join(tc.stages, ",") || strings.Join(modes, ",") != strings.Join(tc.stages, ",") {
				t.Errorf("stages %v, modes %v; want %v for both", stages, modes, tc.stages)
			}
		})
	}

	t.Run("quiet ends the case on an open input", func(t *testing.T) {
		pr, pw := io.Pipe()
		t.Cleanup(func() { _ = pw.Close() })
		go func() { _, _ = pw.Write([]byte(secret + "\r")) }()
		_, reads, fault := rawEnterRun0087(bufio.NewReaderSize(pr, 4096), rawEnterCaseEnter0087,
			func() error { return nil }, func() error { return nil }, func(string) {},
			2*time.Second, 50*time.Millisecond)
		got := rawEnterLine0087("console", "conhost", rawEnterCaseEnter0087, fault, nil, reads)
		if want := "mode=console console=conhost case=enter result=cr-only reads=0d"; got != want {
			t.Errorf("line = %q, want %q", got, want)
		}
	})

	t.Run("first wait ends the case with nothing read", func(t *testing.T) {
		pr, pw := io.Pipe()
		t.Cleanup(func() { _ = pw.Close() })
		_, reads, fault := rawEnterRun0087(bufio.NewReaderSize(pr, 4096), rawEnterCaseCRLF0087,
			func() error { return nil }, func() error { return nil }, func(string) {},
			50*time.Millisecond, 50*time.Millisecond)
		got := rawEnterLine0087("pty", "pty", rawEnterCaseCRLF0087, fault, nil, reads)
		if want := "mode=pty console=pty case=crlf-one-write result=none reads=none"; got != want {
			t.Errorf("line = %q, want %q", got, want)
		}
	})

	t.Run("faults", func(t *testing.T) {
		refused := errors.New("refused")
		_, _, fault := rawEnterRun0087(bufio.NewReader(strings.NewReader(secret)), rawEnterCaseEnter0087,
			func() error { return refused }, func() error { return nil }, func(string) {}, time.Second, time.Second)
		if fault != rawEnterModeRefused0087 {
			t.Errorf("raw mode refused: fault = %q, want %q", fault, rawEnterModeRefused0087)
		}
		for _, f := range []string{rawEnterNoConsole0087, rawEnterModeRefused0087, rawEnterCookedTimeout0087,
			rawEnterNoPty0087, rawEnterNoLine0087, rawEnterBadLine0087} {
			got := rawEnterLine0087("console", "conhost", rawEnterCaseCooked0087, f, nil, nil)
			want := "mode=console console=conhost case=cooked-then-raw result=" + f + " cooked=none reads=none"
			if got != want || !rawEnterAllowed0087(got) {
				t.Errorf("fault %s: line = %q (allowlisted %v), want %q", f, got, rawEnterAllowed0087(got), want)
			}
		}
		for _, bad := range []string{
			"mode=pty console=pty case=enter result=split reads=0d,0a " + secret,
			"mode=pty console=pty case=enter result=split reads=0d,73",
			"mode=pty console=pty case=enter result=" + secret + " reads=none",
		} {
			if rawEnterAllowed0087(bad) {
				t.Errorf("allowlist admits %q", bad)
			}
		}
	})
}
