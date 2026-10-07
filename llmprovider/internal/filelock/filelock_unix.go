//go:build unix

package filelock

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// TryLock takes an exclusive lock on f without waiting. It returns ErrLocked
// when another open file holds it.
func TryLock(f *os.File) error {
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		switch {
		case err == nil:
			return nil
		case errors.Is(err, syscall.EINTR):
			continue
		case errors.Is(err, syscall.EWOULDBLOCK):
			return ErrLocked
		default:
			return fmt.Errorf("filelock: lock %s: %w", f.Name(), err)
		}
	}
}

// Unlock releases f's lock. Closing f releases it too.
func Unlock(f *os.File) error {
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN); err != nil {
		return fmt.Errorf("filelock: unlock %s: %w", f.Name(), err)
	}
	return nil
}
