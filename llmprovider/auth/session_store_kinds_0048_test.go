package auth

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestValidateOAuthSession_ErrorKinds_0048 shows that a custom issuer's
// session with no token URL is ErrAuthFailure, and every other refusal is a
// plain error naming its reason (0048-MADR D4).
func TestValidateOAuthSession_ErrorKinds_0048(t *testing.T) {
	err := ValidateOAuthSession(&OAuthSession{Access: "access", Refresh: "refresh", ClientID: "client",
		Issuer: "https://login.example"})
	if !errors.Is(err, llmprovider.ErrAuthFailure) || !strings.Contains(err.Error(), "has no token URL") {
		t.Errorf("custom issuer without a token URL: %v, want ErrAuthFailure naming the token URL", err)
	}
	for _, tc := range []struct {
		name    string
		session *OAuthSession
		step    string
	}{
		{"nil", nil, "no access token"},
		{"empty access", &OAuthSession{Refresh: "refresh", ClientID: "client"}, "no access token"},
		{"the fixture", &OAuthSession{Access: chatGPTAccessFixture, Issuer: DefaultOpenAIIssuer}, "test fixture"},
		{"no refresh token", &OAuthSession{Access: "access", Issuer: DefaultGrokOAuthIssuer, ClientID: DefaultGrokOAuthClientID}, "no refresh token"},
		{"no client id", &OAuthSession{Access: "access", Refresh: "refresh", TokenURL: "https://issuer.test/token"}, "has no client id"},
		{"plan session with no token URL", planTestSession(""), "plan session has no token URL"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertPlain0048(t, ValidateOAuthSession(tc.session), tc.step)
		})
	}
}

// TestNewFileTokenStore_ErrorKinds_0048 shows that an empty directory is a
// plain error, and a directory that cannot be made wraps the filesystem's
// *fs.PathError (0048-MADR D4).
func TestNewFileTokenStore_ErrorKinds_0048(t *testing.T) {
	_, err := NewFileTokenStore("")
	assertPlain0048(t, err, "empty dir")

	file := filepath.Join(t.TempDir(), "a-file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewFileTokenStore(filepath.Join(file, "tokens"))
	var pathErr *fs.PathError
	if store != nil || !errors.As(err, &pathErr) {
		t.Fatalf("a directory under a regular file: %v, %v; want a *fs.PathError", store, err)
	}
	assertPlain0048(t, err, "FileTokenStore mkdir")
}

// TestFileTokenStoreDelete_ErrorKinds_0048 shows that Delete refuses an
// invalid provider id with ErrInvalidProvider, and returns a failure to
// remove as the *fs.PathError os.Remove gives (0048-MADR D4).
func TestFileTokenStoreDelete_ErrorKinds_0048(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := NewFileTokenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []llmprovider.ProviderID{"", "../escape", "a/b"} {
		if err := store.Delete(ctx, id); !errors.Is(err, llmprovider.ErrInvalidProvider) {
			t.Errorf("Delete(%q) = %v, want ErrInvalidProvider", id, err)
		}
	}
	// A directory at the session's path, with something in it, cannot be
	// removed.
	if err := os.MkdirAll(filepath.Join(dir, string(llmprovider.ProviderGrok)+".json", "child"), 0o700); err != nil {
		t.Fatal(err)
	}
	err = store.Delete(ctx, llmprovider.ProviderGrok)
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) || pathErr.Op != "remove" {
		t.Fatalf("Delete that cannot remove = %v, want os.Remove's *fs.PathError", err)
	}
	assertPlain0048(t, err, "remove")
}
