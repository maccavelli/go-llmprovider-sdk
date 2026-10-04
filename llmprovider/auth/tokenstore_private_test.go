package auth

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestFileTokenStore_RestrictsDirAndTempBeforeWrite (0010-PLAN P6c): on every
// OS, NewFileTokenStore makes its directory private, and Save makes each temp
// file private before its first byte. What "private" means per OS is
// ownerperm's, and its tests show it.
func TestFileTokenStore_RestrictsDirAndTempBeforeWrite(t *testing.T) {
	mkdir, restrict, before := tokenStoreMkdir, tokenStoreRestrict, tokenStoreBeforeWrite
	t.Cleanup(func() { tokenStoreMkdir, tokenStoreRestrict, tokenStoreBeforeWrite = mkdir, restrict, before })
	var events []string
	tokenStoreMkdir = func(dir string) error {
		events = append(events, "private dir "+filepath.Base(dir))
		return mkdir(dir)
	}
	tokenStoreRestrict = func(f *os.File) error {
		events = append(events, "private "+tempKind(f.Name()))
		return restrict(f)
	}
	tokenStoreBeforeWrite = func(name string) {
		events = append(events, "first byte "+tempKind(name))
	}

	store, err := NewFileTokenStore(filepath.Join(t.TempDir(), "tokens"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), "openai", &OAuthSession{Access: "at", Refresh: "rt"}); err != nil {
		t.Fatal(err)
	}
	want := []string{"private dir tokens", "private temp file", "first byte temp file"}
	if !slices.Equal(events, want) {
		t.Errorf("events = %q, want %q", events, want)
	}
}

// tempKind names a Save temp file without its random part.
func tempKind(name string) string {
	if strings.HasPrefix(filepath.Base(name), ".tok-") {
		return "temp file"
	}
	return filepath.Base(name)
}
