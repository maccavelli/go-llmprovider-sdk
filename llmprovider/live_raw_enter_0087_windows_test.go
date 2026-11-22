//go:build live_gateways

package llmprovider_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// TestLive_RawEnterBytes_0087 is 0087-PLAN Phase 1's probe (0087-MADR,
// "Windows: the probe and what each observation decides", OR3, OR7, OR9): it
// records which bytes a Windows console in raw mode hands a bufio.Reader for
// an Enter, for a pasted CR+LF and after a line-mode answer, and whether an
// Enter's CR and LF arrive in one read or in two. It REQUIRES
// LLMPROVIDER_LIVE_0087:
//
//   - pty: the agent's run. For each of P1 (crlf-one-write), P2 (lf-after-cr)
//     and P3 (cooked-then-raw) it creates a pseudo console, starts this test
//     binary as a child attached to it, and writes the case's bytes into the
//     pseudo console's input. Writes tmp/live-0087/pty.txt.
//   - console: the owner's run, in a classic console window or a Windows
//     Terminal tab, with no redirection. For each of R1 (enter), R2 (paste)
//     and R3 (cooked-then-raw) it starts this test binary as a child on the
//     same console, which shows its request there and reads the person's
//     keys. Writes tmp/live-0087/conhost.txt, or terminal.txt when WT_SESSION
//     is set.
//
// The child is this test binary with the variable set to
// child:<console kind>:<case>. It opens CONIN$ and CONOUT$ itself (go test
// gives a test binary no standard input), puts CONIN$ in raw mode with the
// bits golang.org/x/term's makeRaw sets, reads one case through a
// bufio.Reader (rawEnterRun0087) and writes one line (rawEnterLine0087), then
// restores the console's mode and exits, which ends any read still waiting.
//
// Only allowlisted lines are written, to the file and the test's log: the
// mode, the console kind, the case name, a result word and the line-ending
// bytes of each read as hexadecimal; never any other byte read
// (TestRawEnterBytesFormat_0087 proves it on planted letters). The standard
// library only, kernel32 through syscall.NewLazyDLL; no key, sign-in, browser
// or network.
func TestLive_RawEnterBytes_0087(t *testing.T) {
	v := os.Getenv("LLMPROVIDER_LIVE_0087")
	switch {
	case v == "":
		t.Skip("set LLMPROVIDER_LIVE_0087=pty or console to run 0087-PLAN Phase 1's probe")
	case v == "pty":
		rawEnterPty0087(t)
	case v == "console":
		rawEnterConsole0087(t)
	case strings.HasPrefix(v, "child:"):
		parts := strings.Split(v, ":")
		if len(parts) != 3 {
			t.Fatalf("LLMPROVIDER_LIVE_0087: want child:<console kind>:<case>")
		}
		rawEnterChild0087(parts[1], parts[2])
	default:
		t.Fatalf("LLMPROVIDER_LIVE_0087=%q: want pty or console", v)
	}
}

var (
	kernel32DLL0087                       = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode0087                = kernel32DLL0087.NewProc("GetConsoleMode")
	procSetConsoleMode0087                = kernel32DLL0087.NewProc("SetConsoleMode")
	procFlushConsoleInputBuffer0087       = kernel32DLL0087.NewProc("FlushConsoleInputBuffer")
	procCreatePseudoConsole0087           = kernel32DLL0087.NewProc("CreatePseudoConsole")
	procClosePseudoConsole0087            = kernel32DLL0087.NewProc("ClosePseudoConsole")
	procInitializeProcThreadAttrList0087  = kernel32DLL0087.NewProc("InitializeProcThreadAttributeList")
	procUpdateProcThreadAttribute0087     = kernel32DLL0087.NewProc("UpdateProcThreadAttribute")
	procDeleteProcThreadAttributeList0087 = kernel32DLL0087.NewProc("DeleteProcThreadAttributeList")
	procCreateProcessW0087                = kernel32DLL0087.NewProc("CreateProcessW")
)

// The console mode bits golang.org/x/term's makeRaw clears and sets.
const (
	enableProcessedInput0087       = 0x0001
	enableLineInput0087            = 0x0002
	enableEchoInput0087            = 0x0004
	enableVirtualTerminalInput0087 = 0x0200

	extendedStartupInfoPresent0087  = 0x00080000
	startfUseStdHandles0087         = 0x00000100
	procThreadAttrPseudoConsole0087 = 0x00020016
)

// The waits: how long a child waits for its first read and for quiet after
// the last, by mode, and how long the parent waits for a child.
const (
	ptyFirstWait0087      = 15 * time.Second
	ptyQuiet0087          = time.Second
	consoleFirstWait0087  = 120 * time.Second
	consoleQuiet0087      = 1500 * time.Millisecond
	ptyChildLimit0087     = 60 * time.Second
	consoleChildLimit0087 = 270 * time.Second
	ptyBeforeWrite0087    = 200 * time.Millisecond
	ptyLateLineFeed0087   = 10 * time.Millisecond
)

// startupInfoEx0087 is STARTUPINFOEXW: STARTUPINFOW and the attribute list.
type startupInfoEx0087 struct {
	syscall.StartupInfo
	attributeList uintptr
}

// rawEnterDir0087 is tmp/live-0087 under the module root, created.
func rawEnterDir0087() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no go.mod above the working directory")
		}
		dir = parent
	}
	out := filepath.Join(dir, "tmp", "live-0087")
	return out, os.MkdirAll(out, 0o755)
}

func consoleMode0087(h syscall.Handle) (uint32, error) {
	var m uint32
	if r, _, err := procGetConsoleMode0087.Call(uintptr(h), uintptr(unsafe.Pointer(&m))); r == 0 {
		return 0, err
	}
	return m, nil
}

func setConsoleMode0087(h syscall.Handle, m uint32) error {
	if r, _, err := procSetConsoleMode0087.Call(uintptr(h), uintptr(m)); r == 0 {
		return err
	}
	return nil
}

// rawEnterRequests0087 is what the console child shows the person at each
// stage of each case. A pseudo console's child shows the same; no one reads it.
var rawEnterRequests0087 = map[string]string{
	rawEnterCaseEnter0087 + "/raw": "\r\n0087 probe, R1 of 3: press Enter once, then wait.\r\n",
	rawEnterCasePaste0087 + "/raw": "\r\n0087 probe, R2 of 3: paste, once, the two lines you copied " +
		"(none of the text is shown or saved). Do not press Enter.\r\n",
	rawEnterCaseCooked0087 + "/line": "\r\n0087 probe, R3 of 3, first half: type y and press Enter.\r\n",
	rawEnterCaseCooked0087 + "/raw":  "\r\n0087 probe, R3 of 3, second half: type one letter and press Enter.\r\n",
	rawEnterCaseCRLF0087 + "/raw":    "\r\n0087 probe, P1.\r\n",
	rawEnterCaseLate0087 + "/raw":    "\r\n0087 probe, P2.\r\n",
}

// rawEnterChild0087 runs one case in this process and exits: it is the child
// both modes start. It writes .child-<case>.<stage> when a stage is ready and
// .child-<case>.line with its line, in tmp/live-0087.
func rawEnterChild0087(console, name string) {
	dir, err := rawEnterDir0087()
	if err != nil {
		os.Exit(2)
	}
	mode := "console"
	firstWait, quiet := consoleFirstWait0087, consoleQuiet0087
	if console == "pty" {
		mode = "pty"
		firstWait, quiet = ptyFirstWait0087, ptyQuiet0087
	}
	finish := func(line string) {
		_ = os.WriteFile(filepath.Join(dir, ".child-"+name+".line"), []byte(line+"\n"), 0o600)
		os.Exit(0)
	}
	in, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		finish(rawEnterLine0087(mode, console, name, rawEnterNoConsole0087, nil, nil))
	}
	out, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		finish(rawEnterLine0087(mode, console, name, rawEnterNoConsole0087, nil, nil))
	}
	h := syscall.Handle(in.Fd())
	orig, err := consoleMode0087(h)
	if err != nil {
		finish(rawEnterLine0087(mode, console, name, rawEnterNoConsole0087, nil, nil))
	}
	if mode == "console" {
		// Keys pressed between two requests are not this case's.
		_, _, _ = procFlushConsoleInputBuffer0087.Call(uintptr(h))
	}
	raw := orig&^(enableEchoInput0087|enableProcessedInput0087|enableLineInput0087) | enableVirtualTerminalInput0087
	line := (orig | enableEchoInput0087 | enableProcessedInput0087 | enableLineInput0087) &^ enableVirtualTerminalInput0087
	ready := func(stage string) {
		_, _ = out.WriteString(rawEnterRequests0087[name+"/"+stage])
		_ = os.WriteFile(filepath.Join(dir, ".child-"+name+"."+stage), nil, 0o600)
	}
	br := bufio.NewReaderSize(in, 4096)
	cooked, reads, fault := rawEnterRun0087(br, name,
		func() error { return setConsoleMode0087(h, raw) },
		func() error { return setConsoleMode0087(h, line) },
		ready, firstWait, quiet)
	_ = setConsoleMode0087(h, orig)
	if mode == "console" {
		_, _ = out.WriteString("\r\n0087 probe: recorded.\r\n")
	}
	finish(rawEnterLine0087(mode, console, name, fault, cooked, reads))
}

// rawEnterTake0087 reads and removes a child's line. A missing line is
// no-line, and a line the allowlist refuses is bad-line: nothing else is
// passed on.
func rawEnterTake0087(dir, mode, console, name string) string {
	path := filepath.Join(dir, ".child-"+name+".line")
	b, err := os.ReadFile(path)
	_ = os.Remove(path)
	if err != nil {
		return rawEnterLine0087(mode, console, name, rawEnterNoLine0087, nil, nil)
	}
	line := strings.TrimRight(string(b), "\r\n")
	if !rawEnterAllowed0087(line) {
		return rawEnterLine0087(mode, console, name, rawEnterBadLine0087, nil, nil)
	}
	return line
}

// rawEnterClean0087 removes a case's marker and line files before it runs.
func rawEnterClean0087(dir, name string) {
	for _, s := range []string{"line", "raw"} {
		_ = os.Remove(filepath.Join(dir, ".child-"+name+"."+s))
	}
	_ = os.Remove(filepath.Join(dir, ".child-"+name+".line"))
}

// rawEnterWrite0087 writes the lines to tmp/live-0087/<console kind>.txt and
// the test's log, then reads the file back into the log.
func rawEnterWrite0087(t *testing.T, dir, console string, lines []string) {
	t.Helper()
	path := filepath.Join(dir, console+".txt")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("write %s.txt: %v", console, err)
	}
	for _, l := range lines {
		t.Log(l)
	}
	back, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s.txt back: %v", console, err)
	}
	for _, l := range strings.Split(strings.TrimRight(string(back), "\n"), "\n") {
		t.Logf("tmp/live-0087/%s.txt: %s", console, l)
	}
	for _, l := range lines {
		if strings.Contains(l, "result="+rawEnterNoLine0087) || strings.Contains(l, "result="+rawEnterBadLine0087) {
			t.Errorf("a case yielded no line it may write: %s", l)
		}
	}
}

// rawEnterConsole0087 is the owner's run: each case is a child on this
// console.
func rawEnterConsole0087(t *testing.T) {
	dir, err := rawEnterDir0087()
	if err != nil {
		t.Fatalf("tmp/live-0087: %v", err)
	}
	console := "conhost"
	if os.Getenv("WT_SESSION") != "" {
		console = "terminal"
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("this test binary: %v", err)
	}
	var lines []string
	for _, name := range []string{rawEnterCaseEnter0087, rawEnterCasePaste0087, rawEnterCaseCooked0087} {
		rawEnterClean0087(dir, name)
		ctx, cancel := context.WithTimeout(context.Background(), consoleChildLimit0087)
		cmd := exec.CommandContext(ctx, exe, "-test.run=^TestLive_RawEnterBytes_0087$", "-test.count=1", "-test.timeout=5m")
		cmd.Env = append(os.Environ(), "LLMPROVIDER_LIVE_0087=child:"+console+":"+name)
		runErr := cmd.Run()
		cancel()
		if runErr != nil {
			t.Logf("case %s: the child ended with an error", name)
		}
		lines = append(lines, rawEnterTake0087(dir, "console", console, name))
	}
	rawEnterWrite0087(t, dir, console, lines)
}

// rawEnterPty0087 is the agent's run: each case is a child on a pseudo
// console of its own.
func rawEnterPty0087(t *testing.T) {
	dir, err := rawEnterDir0087()
	if err != nil {
		t.Fatalf("tmp/live-0087: %v", err)
	}
	feeds := []struct {
		name string
		feed func(w *os.File, wait func(stage string) bool) error
	}{
		{rawEnterCaseCRLF0087, func(w *os.File, wait func(string) bool) error {
			if !wait("raw") {
				return errors.New("the child never readied")
			}
			time.Sleep(ptyBeforeWrite0087)
			_, err := w.Write([]byte("\r\n"))
			return err
		}},
		{rawEnterCaseLate0087, func(w *os.File, wait func(string) bool) error {
			if !wait("raw") {
				return errors.New("the child never readied")
			}
			time.Sleep(ptyBeforeWrite0087)
			if _, err := w.Write([]byte("\r")); err != nil {
				return err
			}
			time.Sleep(ptyLateLineFeed0087)
			_, err := w.Write([]byte("\n"))
			return err
		}},
		{rawEnterCaseCooked0087, func(w *os.File, wait func(string) bool) error {
			if !wait("line") {
				return errors.New("the child never readied for its line")
			}
			time.Sleep(ptyBeforeWrite0087)
			if _, err := w.Write([]byte("y\r\n")); err != nil {
				return err
			}
			if !wait("raw") {
				return errors.New("the child never readied for its raw read")
			}
			time.Sleep(ptyBeforeWrite0087)
			_, err := w.Write([]byte("abc\r"))
			return err
		}},
	}
	if err := procCreatePseudoConsole0087.Find(); err != nil {
		var lines []string
		for _, f := range feeds {
			lines = append(lines, rawEnterLine0087("pty", "pty", f.name, rawEnterNoPty0087, nil, nil))
		}
		rawEnterWrite0087(t, dir, "pty", lines)
		return
	}
	var lines []string
	for _, f := range feeds {
		rawEnterClean0087(dir, f.name)
		if err := rawEnterPtyCase0087(t, dir, f.name, f.feed); err != nil {
			if errors.Is(err, errNoPty0087) {
				lines = append(lines, rawEnterLine0087("pty", "pty", f.name, rawEnterNoPty0087, nil, nil))
				continue
			}
			t.Logf("case %s: %v", f.name, err)
		}
		lines = append(lines, rawEnterTake0087(dir, "pty", "pty", f.name))
	}
	rawEnterWrite0087(t, dir, "pty", lines)
}

var errNoPty0087 = errors.New("the pseudo console cannot be created")

// rawEnterPtyCase0087 runs one case: a pseudo console, this test binary as a
// child on it, the feed writing into its input, and the child waited for.
func rawEnterPtyCase0087(t *testing.T, dir, name string,
	feed func(w *os.File, wait func(stage string) bool) error) error {
	var inRead, inWrite, outRead, outWrite syscall.Handle
	if err := syscall.CreatePipe(&inRead, &inWrite, nil, 0); err != nil {
		return fmt.Errorf("input pipe: %w", err)
	}
	if err := syscall.CreatePipe(&outRead, &outWrite, nil, 0); err != nil {
		_ = syscall.CloseHandle(inRead)
		_ = syscall.CloseHandle(inWrite)
		return fmt.Errorf("output pipe: %w", err)
	}
	inW := os.NewFile(uintptr(inWrite), "pty-in")
	outR := os.NewFile(uintptr(outRead), "pty-out")
	defer func() { _ = inW.Close() }()

	var hpc syscall.Handle
	size := uintptr(80) | uintptr(25)<<16 // COORD{X: 80, Y: 25}, passed by value
	hr, _, _ := procCreatePseudoConsole0087.Call(size, uintptr(inRead), uintptr(outWrite), 0, uintptr(unsafe.Pointer(&hpc)))
	// The pseudo console holds its own copies of its ends.
	_ = syscall.CloseHandle(inRead)
	_ = syscall.CloseHandle(outWrite)
	if hr != 0 {
		_ = outR.Close()
		return errNoPty0087
	}
	drained := make(chan struct{})
	go func() {
		// The child's console output is drained, never read: a pseudo
		// console stalls when no one reads it.
		_, _ = io.Copy(io.Discard, outR)
		close(drained)
	}()
	defer func() {
		closed := make(chan struct{})
		go func() {
			_, _, _ = procClosePseudoConsole0087.Call(uintptr(hpc))
			close(closed)
		}()
		select {
		case <-closed:
		case <-time.After(10 * time.Second):
			t.Logf("case %s: closing the pseudo console took over 10 s", name)
		}
		_ = outR.Close()
		select {
		case <-drained:
		case <-time.After(10 * time.Second):
			t.Logf("case %s: the pseudo console's output did not end within 10 s", name)
		}
	}()

	var listSize uintptr
	_, _, _ = procInitializeProcThreadAttrList0087.Call(0, 1, 0, uintptr(unsafe.Pointer(&listSize)))
	if listSize == 0 {
		return errors.New("InitializeProcThreadAttributeList gave no size")
	}
	list := make([]byte, listSize)
	listPtr := uintptr(unsafe.Pointer(&list[0]))
	if r, _, err := procInitializeProcThreadAttrList0087.Call(listPtr, 1, 0, uintptr(unsafe.Pointer(&listSize))); r == 0 {
		return fmt.Errorf("InitializeProcThreadAttributeList: %w", err)
	}
	defer func() {
		_, _, _ = procDeleteProcThreadAttributeList0087.Call(listPtr)
		runtime.KeepAlive(list)
	}()
	if r, _, err := procUpdateProcThreadAttribute0087.Call(listPtr, 0, procThreadAttrPseudoConsole0087,
		uintptr(hpc), unsafe.Sizeof(hpc), 0, 0); r == 0 {
		return fmt.Errorf("UpdateProcThreadAttribute: %w", err)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("this test binary: %w", err)
	}
	args := []string{exe, "-test.run=^TestLive_RawEnterBytes_0087$", "-test.count=1", "-test.timeout=2m"}
	for i, a := range args {
		args[i] = syscall.EscapeArg(a)
	}
	cmdLine, err := syscall.UTF16PtrFromString(strings.Join(args, " "))
	if err != nil {
		return fmt.Errorf("command line: %w", err)
	}
	t.Setenv("LLMPROVIDER_LIVE_0087", "child:pty:"+name)
	var si startupInfoEx0087
	si.Cb = uint32(unsafe.Sizeof(si))
	// No standard handles: the child opens CONIN$ and CONOUT$, the pseudo
	// console's, and inherits none of this process's redirected ones.
	si.Flags = startfUseStdHandles0087
	si.attributeList = listPtr
	var pi syscall.ProcessInformation
	r, _, cpErr := procCreateProcessW0087.Call(0, uintptr(unsafe.Pointer(cmdLine)), 0, 0, 0,
		extendedStartupInfoPresent0087, 0, 0, uintptr(unsafe.Pointer(&si)), uintptr(unsafe.Pointer(&pi)))
	t.Setenv("LLMPROVIDER_LIVE_0087", "pty")
	if r == 0 {
		return fmt.Errorf("CreateProcessW: %w", cpErr)
	}
	defer func() {
		_ = syscall.CloseHandle(pi.Thread)
		_ = syscall.CloseHandle(pi.Process)
	}()

	wait := func(stage string) bool {
		marker := filepath.Join(dir, ".child-"+name+"."+stage)
		deadline := time.Now().Add(ptyChildLimit0087)
		for time.Now().Before(deadline) {
			if _, statErr := os.Stat(marker); statErr == nil {
				return true
			}
			if ev, _ := syscall.WaitForSingleObject(pi.Process, 0); ev == syscall.WAIT_OBJECT_0 {
				return false
			}
			time.Sleep(20 * time.Millisecond)
		}
		return false
	}
	feedErr := feed(inW, wait)
	ev, _ := syscall.WaitForSingleObject(pi.Process, uint32(ptyChildLimit0087/time.Millisecond))
	if ev == syscall.WAIT_TIMEOUT {
		_ = syscall.TerminateProcess(pi.Process, 1)
		_, _ = syscall.WaitForSingleObject(pi.Process, 5000)
		return errors.New("the child did not end within its limit and was stopped")
	}
	return feedErr
}
