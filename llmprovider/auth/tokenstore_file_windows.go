//go:build windows

package auth

// syncDir does nothing on Windows, where a directory cannot be opened for
// fsync; the rename itself is atomic.
func syncDir(string) error {
	return nil
}
