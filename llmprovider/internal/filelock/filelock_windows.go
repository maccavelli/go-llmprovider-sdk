//go:build windows

package filelock

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// lockAll locks or unlocks the whole file: the largest range, from byte 0.
const lockAll = ^uint32(0)

// TryLock takes an exclusive lock on f without waiting. It returns ErrLocked
// when another open file holds it.
func TryLock(f *os.File) error {
	var overlapped syscall.Overlapped
	err := lockFileEx(syscall.Handle(f.Fd()), lockfileExclusiveLock|lockfileFailImmediately, 0, lockAll, lockAll, &overlapped)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, syscall.Errno(errorLockViolation)):
		return ErrLocked
	default:
		return fmt.Errorf("filelock: lock %s: %w", f.Name(), err)
	}
}

// Unlock releases f's lock. Closing f releases it too.
func Unlock(f *os.File) error {
	var overlapped syscall.Overlapped
	if err := unlockFileEx(syscall.Handle(f.Fd()), 0, lockAll, lockAll, &overlapped); err != nil {
		return fmt.Errorf("filelock: unlock %s: %w", f.Name(), err)
	}
	return nil
}
