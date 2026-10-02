package llmprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Command-sourced keys (docs/decisions/0017-MADR-together-provider-and-auth-extensions.md
// D3): the pattern of Claude Code's apiKeyHelper, pi's "!command" keys and the
// Grok CLI's GROK_AUTH_PROVIDER_COMMAND.
const (
	defaultCommandTokenTTL     = 5 * time.Minute
	defaultCommandTokenTimeout = 10 * time.Second
	// commandTokenLimit caps a command's output, as 0016-MADR R3 caps a secret.
	commandTokenLimit = 8 << 10
)

// InvalidatingSource is a TokenSource that can be told its token was refused
// (HTTP 401), so its next Token fetches a fresh one. A provider that receives
// a 401 invalidates such a source and retries once. CommandToken is one.
type InvalidatingSource interface {
	TokenSource
	Invalidate()
}

// tokenFuture is one in-flight fetch that concurrent callers share.
type tokenFuture struct {
	done chan struct{}
	tok  Token
	err  error
}

// CommandToken is a TokenSource whose token is the standard output of a
// command the caller names. The command runs directly, never through a shell,
// with the process's environment; its output is trimmed, capped at 8 KiB and
// cached for TTL. Concurrent callers share one run. A failure names the
// command, never its output. Its String, GoString, LogValue and MarshalJSON
// show the command's name only, never its arguments or the token.
type CommandToken struct {
	// Argv is the command and its arguments.
	Argv []string
	// TTL is how long one output is reused; zero is 5 minutes.
	TTL time.Duration
	// Timeout bounds one run; zero is 10 seconds.
	Timeout time.Duration
	// Header is the request header the token goes in; empty is the service's
	// own (see Token).
	Header string

	mu       sync.Mutex
	value    string
	fetched  time.Time
	inflight *tokenFuture
	now      func() time.Time // tests
}

// NewCommandToken returns a CommandToken for argv, with the default TTL and
// timeout.
func NewCommandToken(argv ...string) *CommandToken {
	return &CommandToken{Argv: argv}
}

// Token returns the cached output, or runs the command when there is none or
// it is older than TTL.
func (c *CommandToken) Token(ctx context.Context) (Token, error) {
	c.mu.Lock()
	now := c.clock()
	if c.value != "" && now.Sub(c.fetched) < c.ttl() {
		token := c.cachedToken()
		c.mu.Unlock()
		return token, nil
	}
	if future := c.inflight; future != nil {
		c.mu.Unlock()
		select {
		case <-future.done:
			return future.tok, future.err
		case <-ctx.Done():
			return Token{}, ctx.Err()
		}
	}
	future := &tokenFuture{done: make(chan struct{})}
	c.inflight = future
	argv := append([]string(nil), c.Argv...)
	timeout := c.Timeout
	c.mu.Unlock()

	value, err := runTokenCommand(ctx, argv, timeout)

	c.mu.Lock()
	if err == nil {
		c.value, c.fetched = value, c.clock()
		future.tok = c.cachedToken()
	}
	future.err = err
	c.inflight = nil
	close(future.done)
	c.mu.Unlock()
	return future.tok, err
}

// Invalidate drops the cached output, so the next Token runs the command.
func (c *CommandToken) Invalidate() {
	c.mu.Lock()
	c.value, c.fetched = "", time.Time{}
	c.mu.Unlock()
}

func (c *CommandToken) clock() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

func (c *CommandToken) ttl() time.Duration {
	if c.TTL > 0 {
		return c.TTL
	}
	return defaultCommandTokenTTL
}

// cachedToken is the cached output as a Token; the lock is held.
func (c *CommandToken) cachedToken() Token {
	return Token{Value: c.value, Type: TokenAPIKey, Header: c.Header, Expiry: c.fetched.Add(c.ttl())}
}

// commandName is the command's base name: what formatted forms and errors show.
func commandName(argv []string) string {
	if len(argv) == 0 {
		return "(no command)"
	}
	return filepath.Base(argv[0])
}

// runTokenCommand runs argv and returns its trimmed output.
func runTokenCommand(ctx context.Context, argv []string, timeout time.Duration) (string, error) {
	name := commandName(argv)
	if len(argv) == 0 || argv[0] == "" {
		return "", fmt.Errorf("%w: key command: no command", ErrInvalidRequest)
	}
	if timeout <= 0 {
		timeout = defaultCommandTokenTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	//nolint:gosec // G204: running the caller's named command is this type's purpose (0017-MADR D3); no shell is involved.
	cmd := exec.CommandContext(runCtx, argv[0], argv[1:]...)
	out := &cappedBuffer{limit: commandTokenLimit}
	cmd.Stdout = out
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	switch {
	case errors.Is(runCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil:
		return "", fmt.Errorf("%w: key command %q timed out after %s", ErrAuthFailure, name, timeout)
	case ctx.Err() != nil:
		return "", ctx.Err()
	case out.over:
		return "", fmt.Errorf("%w: key command %q wrote more than %d bytes", ErrAuthFailure, name, commandTokenLimit)
	case err != nil:
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("%w: key command %q exited with status %d", ErrAuthFailure, name, exitErr.ExitCode())
		}
		return "", fmt.Errorf("%w: key command %q: %w", ErrAuthFailure, name, err)
	}
	value := strings.TrimSpace(out.String())
	if value == "" {
		return "", fmt.Errorf("%w: key command %q printed nothing", ErrAuthFailure, name)
	}
	return value, nil
}

// cappedBuffer keeps at most limit bytes and records that more were written.
// The buffer is a field, not embedded: an embedded bytes.Buffer's ReadFrom
// would let io.Copy bypass Write, and the cap with it.
type cappedBuffer struct {
	buf   bytes.Buffer
	limit int
	over  bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if room := b.limit - b.buf.Len(); len(p) > room {
		b.over = true
		if room > 0 {
			b.buf.Write(p[:room])
		}
		return len(p), nil
	}
	return b.buf.Write(p)
}

func (b *cappedBuffer) String() string { return b.buf.String() }

// String renders the source by its command's name only.
func (c *CommandToken) String() string {
	return fmt.Sprintf("CommandToken{Command:%s Header:%s}", commandName(c.Argv), c.Header)
}

// GoString renders the source for %#v, by its command's name only.
func (c *CommandToken) GoString() string { return "&llmprovider." + c.String() }

// LogValue renders the source for slog, by its command's name only.
func (c *CommandToken) LogValue() slog.Value {
	return slog.GroupValue(slog.String("command", commandName(c.Argv)), slog.String("header", c.Header))
}

// MarshalJSON encodes the source by its command's name only.
func (c *CommandToken) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Command string `json:"command"`
		Header  string `json:"header"`
	}{commandName(c.Argv), c.Header})
}
