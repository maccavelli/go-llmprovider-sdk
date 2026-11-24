package wizard

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The Windows tests of 0087-PLAN Phase 2, on a pseudo console, as Phase 1's
// probe ran its cases (0087-MADR, "Amendments of 2026-11-22", A-3 and A-4):
// the test starts this test binary as a child attached to a pseudo console
// of its own, and writes the person's keys into the pseudo console's input.
// The child, switched by ptyChildEnv0087, runs a TextPrompter on CONIN$ and
// CONOUT$, the real raw mode and the real console reads, and reports through
// files in the directory ptyDirEnv0087 names: a marker when each prompt is
// shown, and its answers.

const (
	ptyChildEnv0087 = "WIZARD_TEST_PTY_CHILD_0087"
	ptyDirEnv0087   = "WIZARD_TEST_PTY_DIR_0087"
)

// ptyMark0087 is what the parent writes to the marker in ptyDirEnv0087's
// directory, which proves the parent launched the child (0090-MADR T13).
const ptyMark0087 = "launched by runPty0087"

// ptyMarker0087 is the marker file in dir.
func ptyMarker0087(dir string) string { return filepath.Join(dir, "parent-0090") }

// ptyLaunched0087 reports whether the parent marked ptyDirEnv0087's
// directory: a value left set in a shell names no marked directory, so it is
// ignored and the test runs as the parent.
func ptyLaunched0087() bool {
	dir := os.Getenv(ptyDirEnv0087)
	if dir == "" {
		return false
	}
	raw, err := os.ReadFile(ptyMarker0087(filepath.Clean(dir)))
	return err == nil && string(raw) == ptyMark0087
}

var (
	kernel32Test0087           = windows.NewLazySystemDLL("kernel32.dll")
	procUpdateProcThreadAttr87 = kernel32Test0087.NewProc("UpdateProcThreadAttribute")
)

// markOut0087 is the child's Out: CONOUT$, a copy of what was printed, and a
// marker file the first time each watched prompt is printed.
type markOut0087 struct {
	out     io.Writer
	dir     string
	marks   map[string]string // printed text -> marker name
	printed strings.Builder
}

func (m *markOut0087) Write(b []byte) (int, error) {
	m.printed.Write(b)
	for text, name := range m.marks {
		if strings.Contains(string(b), text) {
			delete(m.marks, text)
			_ = os.WriteFile(filepath.Join(m.dir, name), nil, 0o600)
		}
	}
	return m.out.Write(b)
}

// ptyChild0087 is the child's part. Every role asks for a hidden entry
// ("Key") first. Role "dropped" then asks for a second one ("Key2") and
// writes both entries and how many "empty value" warnings were printed; role
// "line" asks a line-mode question with a default and writes the answer;
// role "bounded" only marks when Secret returns.
func ptyChild0087(role string) {
	dir := os.Getenv(ptyDirEnv0087)
	result := func(s string) { _ = os.WriteFile(filepath.Join(dir, "result"), []byte(s), 0o600) }
	in, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		result("no CONIN$")
		return
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		result("no CONOUT$")
		return
	}
	defer func() { _ = out.Close() }()
	mo := &markOut0087{out: out, dir: dir, marks: map[string]string{
		"Key: ":                         "secret.ready",
		"Key2: ":                        "secret2.ready",
		"Model (default saved-model): ": "input.ready",
	}}
	p := &TextPrompter{In: in, Out: mo}
	key, err := p.Secret("Key")
	if err != nil {
		result("Secret: " + err.Error())
		return
	}
	_ = os.WriteFile(filepath.Join(dir, "secret.done"), nil, 0o600)
	switch role {
	case "bounded":
		result("key=" + key)
	case "dropped":
		key2, err := p.Secret("Key2")
		if err != nil {
			result("Secret2: " + err.Error())
			return
		}
		result("key=" + key + " key2=" + key2 + " warnings=" +
			strconv.Itoa(strings.Count(mo.printed.String(), "empty value")))
	default:
		ans, err := p.Input("Model", "saved-model")
		if err != nil {
			result("Input: " + err.Error())
			return
		}
		result("key=" + key + " input=" + ans)
	}
}

// waitFile0087 waits up to limit for dir/name, or for the child to end.
func waitFile0087(dir, name string, limit time.Duration, child windows.Handle) bool {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
		if ev, _ := windows.WaitForSingleObject(child, 0); ev == windows.WAIT_OBJECT_0 {
			_, err := os.Stat(filepath.Join(dir, name))
			return err == nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// runPty0087 runs this test's own function as a child on a pseudo console,
// with role, and calls feed with the pseudo console's input and the child's
// handle; it returns the child's result file once the child has ended.
func runPty0087(t *testing.T, role string, feed func(in *os.File, dir string, child windows.Handle)) string {
	t.Helper()
	dir := t.TempDir()
	var inRead, inWrite, outRead, outWrite windows.Handle
	if err := windows.CreatePipe(&inRead, &inWrite, nil, 0); err != nil {
		t.Fatalf("input pipe: %v", err)
	}
	if err := windows.CreatePipe(&outRead, &outWrite, nil, 0); err != nil {
		t.Fatalf("output pipe: %v", err)
	}
	inW := os.NewFile(uintptr(inWrite), "pty-in")
	outR := os.NewFile(uintptr(outRead), "pty-out")
	defer func() { _ = inW.Close() }()
	var hpc windows.Handle
	err := windows.CreatePseudoConsole(windows.Coord{X: 80, Y: 25}, inRead, outWrite, 0, &hpc)
	_ = windows.CloseHandle(inRead)
	_ = windows.CloseHandle(outWrite)
	if err != nil {
		_ = outR.Close()
		t.Fatalf("CreatePseudoConsole: %v", err)
	}
	drained := make(chan struct{})
	go func() {
		// The child's console output is drained: a pseudo console stalls
		// when no one reads it.
		_, _ = io.Copy(io.Discard, outR)
		close(drained)
	}()
	defer func() {
		windows.ClosePseudoConsole(hpc)
		_ = outR.Close()
		select {
		case <-drained:
		case <-time.After(10 * time.Second):
			t.Log("the pseudo console's output did not end within 10 s")
		}
	}()

	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		t.Fatalf("NewProcThreadAttributeList: %v", err)
	}
	defer attrs.Delete()
	if r, _, callErr := procUpdateProcThreadAttr87.Call(uintptr(unsafe.Pointer(attrs.List())), 0,
		windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, uintptr(hpc), unsafe.Sizeof(hpc), 0, 0); r == 0 {
		t.Fatalf("UpdateProcThreadAttribute: %v", callErr)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	args := []string{exe, "-test.run=^" + t.Name() + "$", "-test.count=1", "-test.timeout=2m"}
	for i, a := range args {
		args[i] = syscall.EscapeArg(a)
	}
	cmdLine, err := windows.UTF16PtrFromString(strings.Join(args, " "))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ptyMarker0087(dir), []byte(ptyMark0087), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(ptyChildEnv0087, role)
	t.Setenv(ptyDirEnv0087, dir)
	si := &windows.StartupInfoEx{ProcThreadAttributeList: attrs.List()}
	si.Cb = uint32(unsafe.Sizeof(*si))
	// No standard handles: the child opens CONIN$ and CONOUT$, the pseudo
	// console's, and inherits none of this process's redirected ones.
	si.Flags = windows.STARTF_USESTDHANDLES
	var pi windows.ProcessInformation
	err = windows.CreateProcess(nil, cmdLine, nil, nil, false, windows.EXTENDED_STARTUPINFO_PRESENT,
		nil, nil, &si.StartupInfo, &pi)
	if err != nil {
		t.Fatalf("CreateProcess: %v", err)
	}
	defer func() {
		_ = windows.CloseHandle(pi.Thread)
		_ = windows.CloseHandle(pi.Process)
	}()
	feed(inW, dir, pi.Process)
	if ev, _ := windows.WaitForSingleObject(pi.Process, 30000); ev != windows.WAIT_OBJECT_0 {
		_ = windows.TerminateProcess(pi.Process, 1)
		_, _ = windows.WaitForSingleObject(pi.Process, 5000)
		t.Error("the child did not end within 30 s and was stopped")
	}
	b, err := os.ReadFile(filepath.Join(dir, "result"))
	if err != nil {
		return ""
	}
	return string(b)
}

func ptyWrite0087(t *testing.T, w *os.File, s string) {
	t.Helper()
	if _, err := w.WriteString(s); err != nil {
		t.Fatalf("write to the pseudo console: %v", err)
	}
}

// TestSecret_LateLineFeedDropped_Windows_0087 (ST22, "seen"; retargeted
// 2026-11-22, OR14): on a pseudo console, a key and a CR, then an LF 10 ms
// later, then a second hidden entry, key2 and a CR: the second Secret returns
// key2 and no "empty value" warning is printed. On a pseudo console a
// line-mode read takes a leading LF without ending a line on it, so a second
// raw read is where the late LF shows (0087-MADR, "Amendments of 2026-11-22",
// A-6).
func TestSecret_LateLineFeedDropped_Windows_0087(t *testing.T) {
	if role := os.Getenv(ptyChildEnv0087); role != "" && ptyLaunched0087() {
		ptyChild0087(role)
		return
	}
	got := runPty0087(t, "dropped", func(w *os.File, dir string, child windows.Handle) {
		if !waitFile0087(dir, "secret.ready", 60*time.Second, child) {
			t.Fatal("the child never asked for the hidden entry")
		}
		time.Sleep(200 * time.Millisecond)
		ptyWrite0087(t, w, key0087+"\r")
		time.Sleep(10 * time.Millisecond)
		ptyWrite0087(t, w, "\n")
		if !waitFile0087(dir, "secret2.ready", 10*time.Second, child) {
			t.Error("the child never asked for the second hidden entry")
			return
		}
		time.Sleep(300 * time.Millisecond)
		ptyWrite0087(t, w, "key2\r")
	})
	if want := "key=" + key0087 + " key2=key2 warnings=0"; got != want {
		t.Errorf("child: %q; want %q: the late LF was read as an empty second entry", got, want)
	}
}

// TestSecret_LineModeUnaffected_Windows_0087 (ST22, a guard; 2026-11-22,
// OR14): on a pseudo console, a key and a CR, then an LF 10 ms later, then
// my-model and a CR at a line-mode question with a default: Input returns
// my-model, as it does at the base, since a line-mode read takes a leading LF
// without ending a line on it.
func TestSecret_LineModeUnaffected_Windows_0087(t *testing.T) {
	if role := os.Getenv(ptyChildEnv0087); role != "" && ptyLaunched0087() {
		ptyChild0087(role)
		return
	}
	got := runPty0087(t, "line", func(w *os.File, dir string, child windows.Handle) {
		if !waitFile0087(dir, "secret.ready", 60*time.Second, child) {
			t.Fatal("the child never asked for the hidden entry")
		}
		time.Sleep(200 * time.Millisecond)
		ptyWrite0087(t, w, key0087+"\r")
		time.Sleep(10 * time.Millisecond)
		ptyWrite0087(t, w, "\n")
		if !waitFile0087(dir, "input.ready", 10*time.Second, child) {
			t.Error("the child never asked the line-mode question")
			return
		}
		time.Sleep(200 * time.Millisecond)
		ptyWrite0087(t, w, "my-model\r")
	})
	if want := "key=" + key0087 + " input=my-model"; got != want {
		t.Errorf("child: %q; want %q", got, want)
	}
}

// TestSecret_SettleIsBounded_Windows_0087 (ST32): on a pseudo console, a key
// and a CR only, so only events without a character (the CR's key release)
// may be pending: Secret returns within 1 s, waited for at most 5 s.
func TestSecret_SettleIsBounded_Windows_0087(t *testing.T) {
	if role := os.Getenv(ptyChildEnv0087); role != "" && ptyLaunched0087() {
		ptyChild0087(role)
		return
	}
	var elapsed time.Duration
	var returned bool
	got := runPty0087(t, "bounded", func(w *os.File, dir string, child windows.Handle) {
		if !waitFile0087(dir, "secret.ready", 60*time.Second, child) {
			t.Fatal("the child never asked for the hidden entry")
		}
		time.Sleep(200 * time.Millisecond)
		start := time.Now()
		ptyWrite0087(t, w, key0087+"\r")
		returned = waitFile0087(dir, "secret.done", 5*time.Second, child)
		elapsed = time.Since(start)
		if !returned {
			// The child is stopped by runPty0087 after its wait.
			_ = windows.TerminateProcess(child, 1)
		}
	})
	if !returned {
		t.Fatalf("Secret did not return within 5 s of the CR (child: %q)", got)
	}
	if elapsed > time.Second {
		t.Errorf("Secret returned %v after the CR; want under 1 s", elapsed)
	}
	if want := "key=" + key0087; got != want {
		t.Errorf("child: %q; want %q", got, want)
	}
}

// TestPtyChild_StrayValue_Windows_0090 (0090-MADR T13): ptyChildEnv0087 and
// ptyDirEnv0087 left set in a shell, naming a role and a directory no parent
// marked, do not send TestSecret_SettleIsBounded_Windows_0087 down its child
// part: it runs as the parent, on a pseudo console of its own, and passes,
// and nothing is written to the stray directory. A child part run by mistake
// may wait on the console, so it is stopped as soon as it writes there.
func TestPtyChild_StrayValue_Windows_0090(t *testing.T) {
	stray := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestSecret_SettleIsBounded_Windows_0087$", "-test.count=1",
		"-test.timeout=2m")
	cmd.Env = append(os.Environ(), ptyChildEnv0087+"=bounded", ptyDirEnv0087+"="+stray)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var err error
	ended := false
	var written []string
	for deadline := time.Now().Add(90 * time.Second); !ended && time.Now().Before(deadline); {
		select {
		case err = <-done:
			ended = true
		case <-time.After(20 * time.Millisecond):
			if entries, _ := os.ReadDir(stray); len(entries) != 0 {
				for _, e := range entries {
					written = append(written, e.Name())
				}
				_ = cmd.Process.Kill()
				err = <-done
				ended = true
			}
		}
	}
	if !ended {
		_ = cmd.Process.Kill()
		<-done
		t.Fatalf("the test binary did not end within 90 s; its output: %s", out.String())
	}
	if len(written) != 0 {
		t.Fatalf("the stray value ran the child part: it wrote %q in the stray directory; its output: %s", written, out.String())
	}
	if err != nil {
		t.Errorf("TestSecret_SettleIsBounded_Windows_0087 under a stray value: %v; want it to pass as the parent; its output: %s",
			err, out.String())
	}
}

// TestSettle_CharPending_Windows_0087 (ST32; 2026-11-22, OR15): the settle's
// event filter over built console records. Only a key-down that carries a
// character counts; a key release with its character, a key with none, and
// focus, mouse, menu and resize events do not, so the settle waits on them
// to its bound instead of peeking a read that would block. A pseudo console
// leaves no such event pending, so the pseudo-console test cannot hold one
// (0087-MADR, "Amendments of 2026-11-22", A-7).
func TestSettle_CharPending_Windows_0087(t *testing.T) {
	key := func(down int32, char uint16) inputRecord {
		return inputRecord{eventType: windows.KEY_EVENT, keyDown: down, virtualKey: 0x0d, char: char}
	}
	other := func(kind uint16) inputRecord { return inputRecord{eventType: kind, keyDown: 1, char: 'x'} }
	for _, c := range []struct {
		name    string
		records []inputRecord
		want    bool
	}{
		{"none", nil, false},
		{"key-up with a character", []inputRecord{key(0, '\r')}, false},
		{"key-down with no character", []inputRecord{key(1, 0)}, false},
		{"focus", []inputRecord{other(windows.FOCUS_EVENT)}, false},
		{"mouse", []inputRecord{other(windows.MOUSE_EVENT)}, false},
		{"menu", []inputRecord{other(windows.MENU_EVENT)}, false},
		{"resize", []inputRecord{other(windows.WINDOW_BUFFER_SIZE_EVENT)}, false},
		{"key-down with an LF", []inputRecord{key(1, '\n')}, true},
		{"key-down with a letter, after a release", []inputRecord{key(0, '\r'), key(1, 'a')}, true},
	} {
		if got := charPending(c.records); got != c.want {
			t.Errorf("%s: charPending = %v; want %v", c.name, got, c.want)
		}
	}
}
