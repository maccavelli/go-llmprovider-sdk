package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// vendorAuthFileLimit bounds a vendor CLI auth file.
const vendorAuthFileLimit = 1 << 20

var (
	// vendorReadFile reads a CLI's auth file; tests replace it.
	vendorReadFile = readVendorAuthFile
	// vendorRetryDelay is the wait before the one re-read of a file that did
	// not parse; tests shorten it.
	vendorRetryDelay = 50 * time.Millisecond
)

// VendorCLISession is a TokenSource that borrows a vendor CLI's login
// read-through (MADR 0012 §5.1). Every Token call re-reads the CLI's auth
// file and returns its access token; it never refreshes. The refresh token
// stays the CLI's alone, because both vendors revoke a refresh-token family
// when one is used twice (grok-build xai-grok-login/src/oidc/refresh.rs:29-35;
// codex login/src/auth/manager.rs:1657-1690). An expired token is
// ErrAuthFailure telling the user to run the CLI, which refreshes it.
type VendorCLISession struct {
	// Provider is ProviderOpenAI (the Codex CLI's auth.json) or ProviderGrok
	// (the Grok CLI's auth.json).
	Provider llmprovider.ProviderID
	// Path is the CLI's auth file.
	Path string

	mu        sync.Mutex
	accountID string // the ChatGPT account id last read
	fedramp   bool   // the id token's chatgpt_account_is_fedramp, last read
}

// vendorCredential is what one read of a CLI auth file yields.
type vendorCredential struct {
	access    string
	expiry    time.Time
	accountID string
	fedramp   bool
}

// Token re-reads the auth file and returns its access token, or
// ErrAuthFailure when the file is unreadable, holds no token, or the token
// has expired.
func (s *VendorCLISession) Token(ctx context.Context) (llmprovider.Token, error) {
	if err := ctx.Err(); err != nil {
		return llmprovider.Token{}, err
	}
	cli, hint, parse := vendorCLI(s.Provider)
	if parse == nil {
		return llmprovider.Token{}, fmt.Errorf("%w: no vendor CLI session for provider %q", llmprovider.ErrInvalidRequest, s.Provider)
	}
	raw, err := vendorReadFile(s.Path)
	if errors.Is(err, errVendorBrokenLink) {
		// A login cannot mend a dangling link (0026-MADR F40).
		return llmprovider.Token{}, fmt.Errorf("%w: read the %s login: %w", llmprovider.ErrAuthFailure, cli, err)
	}
	if err != nil {
		return llmprovider.Token{}, fmt.Errorf("%w: read the %s login: %w; %s", llmprovider.ErrAuthFailure, cli, err, hint)
	}
	cred, err := parse(raw)
	if err != nil {
		// The CLI rewrites its file in place, truncate then write, so a read
		// can land between the two: read it once more (0026-MADR F41).
		if sleepErr := sleepWithContext(ctx, vendorRetryDelay); sleepErr != nil {
			return llmprovider.Token{}, sleepErr
		}
		if raw, err = vendorReadFile(s.Path); err == nil {
			cred, err = parse(raw)
		}
	}
	if err != nil {
		return llmprovider.Token{}, fmt.Errorf("%w: the %s login in %s: %w; %s", llmprovider.ErrAuthFailure, cli, s.Path, err, hint)
	}
	if !cred.expiry.IsZero() && !time.Now().Before(cred.expiry) {
		return llmprovider.Token{}, fmt.Errorf("%w: the %s login in %s expired at %s; %s",
			llmprovider.ErrAuthFailure, cli, s.Path, cred.expiry.Format(time.RFC3339), hint)
	}
	s.mu.Lock()
	s.accountID, s.fedramp = cred.accountID, cred.fedramp
	s.mu.Unlock()
	return llmprovider.Token{Value: cred.access, Type: llmprovider.TokenBearer, Expiry: cred.expiry}, nil
}

// vendorCLI names a provider's CLI, the advice for a stale login, and its
// auth-file parser; the parser is nil for a provider without one.
func vendorCLI(provider llmprovider.ProviderID) (cli, hint string, parse func([]byte) (vendorCredential, error)) {
	switch provider {
	case llmprovider.ProviderOpenAI:
		return "Codex CLI", "run codex to refresh it, or codex login", parseCodexAuth
	case llmprovider.ProviderGrok:
		return "Grok CLI", "run grok to refresh it, or grok login", parseGrokAuth
	default:
		return "", "", nil
	}
}

// errVendorBrokenLink marks an auth file that is a symlink to nothing.
var errVendorBrokenLink = errors.New("the auth file is a symlink whose target cannot be read")

// readVendorAuthFile reads at most vendorAuthFileLimit bytes of path. A
// symlinked file, as the Grok CLI writes through dotfile and GROK_HOME
// overlays, is resolved first, and the target is opened within its own
// directory (0026-MADR F40).
func readVendorAuthFile(path string) ([]byte, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		if info, lerr := os.Lstat(path); lerr == nil && info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("%s: %w: %w", path, errVendorBrokenLink, err)
		}
		return nil, err
	}
	root, err := os.OpenRoot(filepath.Dir(resolved))
	if err != nil {
		return nil, err
	}
	defer func() { ignoreOAuthError(root.Close()) }()
	f, err := root.Open(filepath.Base(resolved))
	if err != nil {
		return nil, err
	}
	defer func() { ignoreOAuthError(f.Close()) }()
	return io.ReadAll(io.LimitReader(f, vendorAuthFileLimit))
}

// parseCodexAuth reads ~/.codex/auth.json's ChatGPT tokens. Expiry is the
// access token's JWT exp, as Codex reads it (login/src/auth/manager.rs:3004-3026).
func parseCodexAuth(raw []byte) (vendorCredential, error) {
	var file struct {
		Tokens struct {
			AccessToken string `json:"access_token"`
			AccountID   string `json:"account_id"`
			IDToken     string `json:"id_token"`
		} `json:"tokens"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return vendorCredential{}, fmt.Errorf("decode: %w", err)
	}
	if file.Tokens.AccessToken == "" {
		return vendorCredential{}, fmt.Errorf("no ChatGPT access token")
	}
	return vendorCredential{
		access:    file.Tokens.AccessToken,
		expiry:    jwtExpiry(file.Tokens.AccessToken),
		accountID: file.Tokens.AccountID,
		fedramp:   chatGPTFedRAMP(file.Tokens.IDToken),
	}, nil
}

// grokAuthScope is the Grok CLI's key for its default login,
// "{issuer}::{client_id}" (xai-grok-login/src/config.rs:188-196, :241-256).
var grokAuthScope = strings.TrimRight(DefaultGrokOAuthIssuer, "/") + "::" + DefaultGrokOAuthClientID

// parseGrokAuth reads the Grok CLI's auth.json: exactly the default login's
// entry, never another scope. Expiry is the entry's expires_at, else the
// access token's JWT exp.
func parseGrokAuth(raw []byte) (vendorCredential, error) {
	var file map[string]struct {
		Key       string `json:"key"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return vendorCredential{}, fmt.Errorf("decode: %w", err)
	}
	entry, ok := file[grokAuthScope]
	if !ok || entry.Key == "" {
		return vendorCredential{}, fmt.Errorf("no %s login", grokAuthScope)
	}
	expiry := jwtExpiry(entry.Key)
	if entry.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, entry.ExpiresAt)
		if err != nil {
			return vendorCredential{}, fmt.Errorf("expires_at: %w", err)
		}
		expiry = parsed
	}
	return vendorCredential{access: entry.Key, expiry: expiry}, nil
}

// jwtExpiry reads a JWT's numeric exp claim, or returns the zero time for a
// token that is not a JWT or has none.
func jwtExpiry(token string) time.Time {
	exp, _ := jwtTimes(token)
	return exp
}

// jwtTimes reads a JWT's exp and iat claims; each is zero when absent or
// unreadable.
func jwtTimes(token string) (exp, iat time.Time) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, time.Time{}
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, time.Time{}
	}
	var claims struct {
		Exp *float64 `json:"exp"`
		Iat *float64 `json:"iat"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, time.Time{}
	}
	if claims.Exp != nil {
		exp = time.Unix(int64(*claims.Exp), 0).UTC()
	}
	if claims.Iat != nil {
		iat = time.Unix(int64(*claims.Iat), 0).UTC()
	}
	return exp, iat
}

// Account returns the ChatGPT account id and FedRAMP flag a Codex CLI
// session last read.
func (s *VendorCLISession) Account() (id string, fedRAMP bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.accountID, s.fedramp
}
