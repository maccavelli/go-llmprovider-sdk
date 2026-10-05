//go:build unix

package ownerperm

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

// TestMkdirAll_TightensExisting (0021-MADR T15): an existing directory that
// group and others can write loses those bits.
func TestMkdirAll_TightensExisting(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tokens")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := MkdirAll(dir); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if fi, err := os.Stat(dir); err != nil || fi.Mode().Perm() != 0o755 {
		t.Fatalf("dir mode = %v (%v), want 0755", fi.Mode().Perm(), err)
	}
}

// TestMkdirAll_RefusesSymlink (0021-MADR T15): a symlink at the path, even to
// a directory, is refused.
func TestMkdirAll_RefusesSymlink(t *testing.T) {
	target := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "tokens")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := MkdirAll(link); err == nil {
		t.Fatal("MkdirAll accepted a symlink")
	}
}

// TestMkdirAll_UnderAFile: a path under a regular file cannot be created.
func TestMkdirAll_UnderAFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := MkdirAll(filepath.Join(file, "tokens")); err == nil {
		t.Fatal("MkdirAll created a directory under a regular file")
	}
}

// TestMkdirAll_RefusesForeignOwner (0021-MADR T15): a directory another user
// owns, or whose owner cannot be read, is refused and left as it is.
func TestMkdirAll_RefusesForeignOwner(t *testing.T) {
	t.Cleanup(func() { ownerOf = statOwner })
	for name, owner := range map[string]func(os.FileInfo) (int, bool){
		"another user": func(os.FileInfo) (int, bool) { return os.Getuid() + 1, true },
		"unreadable":   func(os.FileInfo) (int, bool) { return 0, false },
	} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "tokens")
			if err := os.Mkdir(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(dir, 0o777); err != nil {
				t.Fatal(err)
			}
			ownerOf = owner
			if err := MkdirAll(dir); err == nil {
				t.Fatal("MkdirAll accepted a directory the user does not own")
			}
			if fi, err := os.Stat(dir); err != nil || fi.Mode().Perm() != 0o777 {
				t.Errorf("dir mode = %v (%v); a refused directory must not be changed", fi.Mode().Perm(), err)
			}
		})
	}
}

// noSys is a FileInfo with no system data.
type noSys struct{}

func (noSys) Name() string       { return "x" }
func (noSys) Size() int64        { return 0 }
func (noSys) Mode() os.FileMode  { return os.ModeDir }
func (noSys) ModTime() time.Time { return time.Time{} }
func (noSys) IsDir() bool        { return true }
func (noSys) Sys() any           { return nil }

// TestStatOwner: the owner of a directory the test made is the current user;
// a FileInfo with no system data has no owner.
func TestStatOwner(t *testing.T) {
	fi, err := os.Lstat(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if uid, ok := statOwner(fi); !ok || uid != os.Getuid() {
		t.Errorf("statOwner = %d, %t; want %d, true", uid, ok, os.Getuid())
	}
	if _, ok := statOwner(noSys{}); ok {
		t.Error("statOwner read an owner from no system data")
	}
}
