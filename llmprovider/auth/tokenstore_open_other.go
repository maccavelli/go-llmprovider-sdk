//go:build !windows

package auth

import (
	"os"
	"path/filepath"
)

// openShared opens path for reading. On Unix a reader never blocks a rename
// over the file, so it is os.Open.
func openShared(path string) (*os.File, error) {
	return os.Open(filepath.Clean(path))
}
