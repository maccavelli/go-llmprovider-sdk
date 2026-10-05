//go:build unix

package ownerperm

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// ownerOf reads the uid that owns a file; a seam for tests (0021-MADR T15).
var ownerOf = statOwner

// statOwner is the owner in fi's stat data; false when it has none.
func statOwner(fi os.FileInfo) (uid int, ok bool) {
	if st, isStat := fi.Sys().(*syscall.Stat_t); isStat {
		return int(st.Uid), true
	}
	return 0, false
}

// MkdirAll creates dir, and any parent it needs, with mode 0o700. dir itself
// must then be a directory, not a symlink, owned by the current user; one
// that group or others can write loses those bits, so no other user can
// replace a file in it (0021-MADR T15).
func MkdirAll(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	fi, err := os.Lstat(dir)
	if err != nil || !fi.IsDir() {
		return errors.Join(fmt.Errorf("ownerperm: %s is not a directory", dir), err)
	}
	if uid, ok := ownerOf(fi); !ok || uid != os.Getuid() {
		return fmt.Errorf("ownerperm: %s is not owned by the current user", dir)
	}
	if mode := fi.Mode().Perm(); mode&0o022 != 0 {
		return os.Chmod(dir, mode&^0o022)
	}
	return nil
}

// File sets f's mode to 0o600. Call it before writing anything secret.
func File(f *os.File) error {
	return f.Chmod(0o600)
}
