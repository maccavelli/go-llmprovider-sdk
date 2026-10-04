//go:build unix

package ownerperm

import "os"

// MkdirAll creates dir, and any parent it needs, with mode 0o700. An existing
// directory keeps its mode.
func MkdirAll(dir string) error {
	return os.MkdirAll(dir, 0o700)
}

// File sets f's mode to 0o600. Call it before writing anything secret.
func File(f *os.File) error {
	return f.Chmod(0o600)
}
