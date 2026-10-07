package auth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

const grokDefaultLogin = `{"https://auth.x.ai::b1a00492-073a-47ea-816f-4c329264a828": {"key": "grok-default", "auth_mode": "oidc", "expires_at": "2099-01-01T00:00:00Z"}}`

// TestVendorCLISession_SymlinkedAuthFile: the Grok CLI writes auth.json
// through a symlink into another directory (dotfile and GROK_HOME overlays),
// and the login is read through it (0026-MADR F40).
func TestVendorCLISession_SymlinkedAuthFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need a privilege on Windows")
	}
	real := filepath.Join(t.TempDir(), "auth.json")
	writeVendorFile(t, real, grokDefaultLogin)
	link := filepath.Join(t.TempDir(), "auth.json")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	tok, err := (&VendorCLISession{Provider: llmprovider.ProviderGrok, Path: link}).Token(context.Background())
	if err != nil || tok.Value != "grok-default" {
		t.Fatalf("Token = %q, %v; want the linked login", tok.Value, err)
	}
}

// TestVendorCLISession_BrokenSymlink: a link whose target is gone says so,
// rather than advising a login that cannot fix it (0026-MADR F40).
func TestVendorCLISession_BrokenSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need a privilege on Windows")
	}
	link := filepath.Join(t.TempDir(), "auth.json")
	if err := os.Symlink(filepath.Join(t.TempDir(), "gone.json"), link); err != nil {
		t.Fatal(err)
	}
	_, err := (&VendorCLISession{Provider: llmprovider.ProviderGrok, Path: link}).Token(context.Background())
	if !errors.Is(err, llmprovider.ErrAuthFailure) || !strings.Contains(err.Error(), "symlink") || strings.Contains(err.Error(), "grok login") {
		t.Fatalf("Token err = %v; want ErrAuthFailure naming the broken symlink, without login advice", err)
	}
}

// TestVendorCLISession_TornFileRetriedOnce: a read that lands while the CLI
// rewrites its file in place (truncate, then write) is read again once, and
// the whole file is used (0026-MADR F41). A file that stays torn keeps
// today's error.
func TestVendorCLISession_TornFileRetriedOnce(t *testing.T) {
	prevRead, prevDelay := vendorReadFile, vendorRetryDelay
	t.Cleanup(func() { vendorReadFile, vendorRetryDelay = prevRead, prevDelay })
	vendorRetryDelay = time.Millisecond
	reads := 0
	vendorReadFile = func(string) ([]byte, error) {
		reads++
		if reads == 1 {
			return []byte(grokDefaultLogin[:20]), nil
		}
		return []byte(grokDefaultLogin), nil
	}
	s := &VendorCLISession{Provider: llmprovider.ProviderGrok, Path: "unused"}
	if tok, err := s.Token(context.Background()); err != nil || tok.Value != "grok-default" || reads != 2 {
		t.Fatalf("Token = %q, %v after %d reads; want the whole file on the second read", tok.Value, err, reads)
	}

	reads = 0
	vendorReadFile = func(string) ([]byte, error) { reads++; return []byte(grokDefaultLogin[:20]), nil }
	if _, err := s.Token(context.Background()); !errors.Is(err, llmprovider.ErrAuthFailure) || reads != 2 {
		t.Fatalf("Token err = %v after %d reads; want ErrAuthFailure after 2", err, reads)
	}
}
