package claude

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// keyHelperEnv names the file TestHelperClaudeKeyCommand counts its runs in;
// set, it makes this test binary act as a key command.
const keyHelperEnv = "LLMPROVIDER_TEST_CLAUDE_KEY_HELPER"

// keyHelperMark is what a parent writes to the marker beside keyHelperEnv's
// file, which proves the parent launched the helper (0090-MADR T13).
const keyHelperMark = "launched by TestClaude_ConcurrentRefusalsRunCommandOnce"

// keyHelperMarker is the marker file beside path, keyHelperEnv's file.
func keyHelperMarker(path string) string {
	return filepath.Join(filepath.Dir(filepath.Clean(path)), "parent-0090")
}

// keyHelperLaunched reports whether a parent marked path for the helper: a
// value left set in a shell names no marked file, so it is ignored.
func keyHelperLaunched(path string) bool {
	raw, err := os.ReadFile(keyHelperMarker(path))
	return err == nil && string(raw) == keyHelperMark
}

// TestHelperClaudeKeyCommand is not a test: run as a child with keyHelperEnv
// set by a parent that marked it, it appends a line to that file and prints
// key-N for its Nth run.
func TestHelperClaudeKeyCommand(t *testing.T) {
	path := os.Getenv(keyHelperEnv)
	if path == "" || !keyHelperLaunched(path) {
		t.Skip("helper process only")
	}
	f, err := os.OpenFile(filepath.Clean(path), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		os.Exit(9)
	}
	_, _ = f.WriteString("run\n")
	_ = f.Close()
	raw, _ := os.ReadFile(filepath.Clean(path))
	fmt.Printf("key-%d\n", strings.Count(string(raw), "run"))
	os.Exit(0)
}

// TestHelperClaudeKeyCommand_StrayValue_0090 (0090-MADR T13): keyHelperEnv
// left set in a shell, to a file no parent marked, does not make the test
// binary act as a key command: TestHelperClaudeKeyCommand skips, and writes
// nothing to that file.
func TestHelperClaudeKeyCommand_StrayValue_0090(t *testing.T) {
	stray := filepath.Join(t.TempDir(), "runs")
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperClaudeKeyCommand$", "-test.count=1", "-test.v")
	cmd.Env = append(os.Environ(), keyHelperEnv+"="+stray)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the test binary: %v\n%s", err, out)
	}
	if _, statErr := os.Stat(stray); statErr == nil {
		t.Errorf("the stray value ran the key command: it wrote %s; output:\n%s", stray, out)
	}
	if !strings.Contains(string(out), "--- SKIP: TestHelperClaudeKeyCommand") {
		t.Errorf("TestHelperClaudeKeyCommand did not skip under a stray value; output:\n%s", out)
	}
}

// TestClaude_ConcurrentRefusalsRunCommandOnce (0021-MADR D5, T1): four
// requests hold the CommandToken's first key and each gets a 401. The 401s
// land one at a time, each after the previous request was resent, so every
// late one names a key the source has already replaced. The command runs
// twice in all: once for the first key and once for its replacement.
func TestClaude_ConcurrentRefusalsRunCommandOnce(t *testing.T) {
	const requests = 4
	path := filepath.Join(t.TempDir(), "runs")
	if err := os.WriteFile(keyHelperMarker(path), []byte(keyHelperMark), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(keyHelperEnv, path)
	src := llmprovider.NewCommandToken(os.Args[0], "-test.run=^TestHelperClaudeKeyCommand$")

	arrived, retried := make(chan struct{}, requests), make(chan struct{}, requests)
	release, done := make(chan struct{}), make(chan struct{})
	srv := httptest.NewServer(wiretest.StreamReplies(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "key-1" {
			retried <- struct{}{}
			_, _ = io.WriteString(w, okReply)
			return
		}
		arrived <- struct{}{}
		select {
		case <-release:
		case <-done:
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`)
	})))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(done) })
	p := build(t, llmprovider.WithTokenSource(src), llmprovider.WithBaseURL(srv.URL))

	errs := make([]error, requests)
	var wg sync.WaitGroup
	for i := range requests {
		wg.Go(func() { _, errs[i] = p.Generate(context.Background(), text("hi")) })
	}
	wait := func(ch <-chan struct{}, what string) {
		t.Helper()
		select {
		case <-ch:
		case <-time.After(10 * time.Second):
			t.Fatalf("timed out waiting for %s", what)
		}
	}
	for range requests {
		wait(arrived, "the first requests")
	}
	for range requests {
		release <- struct{}{}
		wait(retried, "a resend")
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("request %d: %v", i, err)
		}
	}
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	if runs := strings.Count(string(raw), "run"); runs != 2 {
		t.Errorf("the key command ran %d times, want 2: a late 401 discarded a fresh key", runs)
	}
}
