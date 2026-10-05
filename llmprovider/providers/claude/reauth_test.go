package claude

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// keyHelperEnv names the file TestHelperClaudeKeyCommand counts its runs in;
// set, it makes this test binary act as a key command.
const keyHelperEnv = "LLMPROVIDER_TEST_CLAUDE_KEY_HELPER"

// TestHelperClaudeKeyCommand is not a test: run as a child with keyHelperEnv
// set, it appends a line to that file and prints key-N for its Nth run.
func TestHelperClaudeKeyCommand(t *testing.T) {
	path := os.Getenv(keyHelperEnv)
	if path == "" {
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

// TestClaude_ConcurrentRefusalsRunCommandOnce (0021-MADR D5, T1): four
// requests hold the CommandToken's first key and each gets a 401. The 401s
// land one at a time, each after the previous request was resent, so every
// late one names a key the source has already replaced. The command runs
// twice in all: once for the first key and once for its replacement.
func TestClaude_ConcurrentRefusalsRunCommandOnce(t *testing.T) {
	const requests = 4
	path := filepath.Join(t.TempDir(), "runs")
	t.Setenv(keyHelperEnv, path)
	src := llmprovider.NewCommandToken(os.Args[0], "-test.run=^TestHelperClaudeKeyCommand$")

	arrived, retried := make(chan struct{}, requests), make(chan struct{}, requests)
	release, done := make(chan struct{}), make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
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
