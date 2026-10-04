//go:build unix

package ownerperm

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMkdirAllAndFile_Modes: the directory is created 0o700 and File sets
// 0o600, whatever the umask allowed at creation.
func TestMkdirAllAndFile_Modes(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tokens")
	if err := MkdirAll(dir); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if fi, err := os.Stat(dir); err != nil || fi.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode = %v (%v), want 0700", fi.Mode().Perm(), err)
	}
	f, err := os.OpenFile(filepath.Join(dir, "s.json"), os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := File(f); err != nil {
		t.Fatalf("File: %v", err)
	}
	if fi, err := f.Stat(); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("file mode = %v (%v), want 0600", fi.Mode().Perm(), err)
	}
}
