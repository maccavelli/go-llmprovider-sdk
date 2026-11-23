//go:build unix

package wizard

import (
	"errors"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// errNoSelect is a descriptor select(2) cannot take: negative, as a closed
// file's is, or at or above FD_SETSIZE.
var errNoSelect = errors.New("wizard: descriptor out of select's range")

// pendingWithin reports whether f becomes readable within wait, by select(2):
// poll(2) does not support devices on macOS (0087-MADR A1). It is false and
// an error for a descriptor select cannot take, which is never given to it,
// and for any select error, EINTR included; false and no error when the wait
// ends (0087-MADR A3).
func pendingWithin(f *os.File, wait time.Duration) (bool, error) {
	fd := int(f.Fd())
	if fd < 0 || fd >= unix.FD_SETSIZE {
		return false, errNoSelect
	}
	var set unix.FdSet
	set.Zero()
	set.Set(fd)
	tv := unix.NsecToTimeval(wait.Nanoseconds())
	n, err := unix.Select(fd+1, &set, nil, nil, &tv)
	if err != nil {
		return false, err
	}
	return n > 0 && set.IsSet(fd), nil
}
