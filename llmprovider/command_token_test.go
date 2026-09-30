package llmprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// keyHelperEnv makes this test binary act as a key command, for
// TestHelperKeyCommand.
const keyHelperEnv = "LLMPROVIDER_TEST_KEY_HELPER"

// TestHelperKeyCommand is not a test: run as a child with keyHelperEnv set,
// it behaves as the key command its arguments after "--" name, then exits.
func TestHelperKeyCommand(t *testing.T) {
	if os.Getenv(keyHelperEnv) != "1" {
		t.Skip("helper process only")
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	switch args[0] {
	case "print":
		fmt.Println("  " + args[1] + "  ")
	case "count": // count PATH: append a line, print key-N for the Nth run
		f, err := os.OpenFile(args[1], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			os.Exit(9)
		}
		_, _ = f.WriteString("run\n")
		_ = f.Close()
		raw, _ := os.ReadFile(args[1])
		fmt.Printf("key-%d\n", strings.Count(string(raw), "run"))
	case "sleep":
		time.Sleep(30 * time.Second)
	case "big":
		fmt.Print(strings.Repeat("x", commandTokenLimit+1))
	case "fail":
		fmt.Print("OUTPUT-" + plantedSecret)
		os.Exit(3)
	case "empty":
	}
	os.Exit(0)
}

// helperArgv is a command running this binary as the key helper in mode.
func helperArgv(t *testing.T, mode ...string) []string {
	t.Helper()
	t.Setenv(keyHelperEnv, "1")
	return append([]string{os.Args[0], "-test.run=^TestHelperKeyCommand$", "--"}, mode...)
}

func runs(t *testing.T, path string) int {
	t.Helper()
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(raw), "run")
}

// TestCommandToken_TrimsAndCaches (0017-MADR D3): the trimmed output is the
// token, reused for TTL, and rerun after it.
func TestCommandToken_TrimsAndCaches(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs")
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	c := &CommandToken{Argv: helperArgv(t, "count", path), TTL: time.Minute, now: func() time.Time { return now }}
	ctx := context.Background()
	for range 3 {
		tok, err := c.Token(ctx)
		if err != nil || tok.Value != "key-1" {
			t.Fatalf("Token = %q, %v; want the cached key-1", tok.Value, err)
		}
	}
	now = now.Add(2 * time.Minute)
	if tok, err := c.Token(ctx); err != nil || tok.Value != "key-2" || runs(t, path) != 2 {
		t.Fatalf("after TTL: Token = %q, %v after %d runs; want key-2 from a second run", tok.Value, err, runs(t, path))
	}
	if tok, _ := NewCommandToken(helperArgv(t, "print", "padded")...).Token(ctx); tok.Value != "padded" {
		t.Errorf("output %q was not trimmed", tok.Value)
	}
}

// TestCommandToken_Failures: a timeout, oversized output, a failing command,
// empty output and no command each fail as an auth failure that names the
// command and never its output.
func TestCommandToken_Failures(t *testing.T) {
	for _, tc := range []struct {
		name string
		c    *CommandToken
		want string
	}{
		{"timeout", &CommandToken{Argv: helperArgv(t, "sleep"), Timeout: 300 * time.Millisecond}, "timed out"},
		{"oversized", NewCommandToken(helperArgv(t, "big")...), "more than 8192 bytes"},
		{"non-zero exit", NewCommandToken(helperArgv(t, "fail")...), "exited with status 3"},
		{"empty", NewCommandToken(helperArgv(t, "empty")...), "printed nothing"},
		{"missing", NewCommandToken(filepath.Join(t.TempDir(), "no-such-command")), `"no-such-command"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			_, err := tc.c.Token(context.Background())
			if !errors.Is(err, ErrAuthFailure) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Token = %v, want an auth failure naming %q", err, tc.want)
			}
			if strings.Contains(err.Error(), plantedSecret) || strings.Contains(err.Error(), "xxxx") {
				t.Errorf("the error carries the command's output: %v", err)
			}
			if time.Since(start) > 5*time.Second {
				t.Errorf("took %v", time.Since(start))
			}
		})
	}
	if _, err := NewCommandToken().Token(context.Background()); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("no argv: %v, want ErrInvalidRequest", err)
	}
}

// TestCommandToken_ConcurrentCallersRunOnce: callers arriving together share
// one run.
func TestCommandToken_ConcurrentCallersRunOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs")
	c := NewCommandToken(helperArgv(t, "count", path)...)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if tok, err := c.Token(context.Background()); err != nil || tok.Value != "key-1" {
				t.Errorf("Token = %q, %v", tok.Value, err)
			}
		})
	}
	wg.Wait()
	if n := runs(t, path); n != 1 {
		t.Errorf("%d runs for 8 concurrent callers, want 1", n)
	}
}

// TestCommandToken_Redacts: formatted forms name the command only, never its
// arguments or the token.
func TestCommandToken_Redacts(t *testing.T) {
	c := &CommandToken{Argv: []string{"/usr/local/bin/vault-key", "--secret=" + plantedSecret}, Header: "x-api-key"}
	c.value = "tok-" + plantedSecret
	var logged bytes.Buffer
	slog.New(slog.NewJSONHandler(&logged, nil)).Info("m", "src", c)
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	for form, out := range map[string]string{"%v": fmt.Sprintf("%v", c), "%#v": fmt.Sprintf("%#v", c),
		"slog": logged.String(), "json": string(raw)} {
		if strings.Contains(out, plantedSecret) || !strings.Contains(out, "vault-key") {
			t.Errorf("%s = %s; want the command's name and no secret", form, out)
		}
	}
	if got := strconv.Quote(commandName(nil)); got != `"(no command)"` {
		t.Errorf("commandName(nil) = %s", got)
	}
}

// TestCommandToken_RerunsAfterInvalidate: Invalidate makes the next Token run
// the command again, for its fresh output. It is the source half of the old
// TestCommandToken_RerunAfter401; the openai package keeps the provider half,
// TestOpenAI_RetriesOnceAfterInvalidate (0015-PLAN S7).
func TestCommandToken_RerunsAfterInvalidate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs")
	src := NewCommandToken(helperArgv(t, "count", path)...)
	first, err := src.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	src.Invalidate()
	second, err := src.Token(context.Background())
	if err != nil || first.Value != "key-1" || second.Value != "key-2" || runs(t, path) != 2 {
		t.Fatalf("tokens %q then %q (%v) after %d runs; want key-1, then key-2 after one rerun", first.Value, second.Value, err, runs(t, path))
	}
}
