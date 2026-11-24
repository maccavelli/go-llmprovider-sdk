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

// selectFn is select(2): a variable, so that a test can interrupt it
// (0087-MADR A-8).
var selectFn = unix.Select

// pendingWithin reports whether f becomes readable within wait, by select(2):
// poll(2) does not support devices on macOS (0087-MADR A1). A select that a
// signal interrupts (EINTR), the Go runtime's own preemption signal among
// them, is made again with the time left to a deadline fixed before the first
// call, so the whole check still ends within wait; with no time left it is
// false and no error, as when the wait ends (0087-MADR A-8, OR17). It is false
// and an error for a descriptor select cannot take, which is never given to
// it, and for any other select error (0087-MADR A3).
func pendingWithin(f *os.File, wait time.Duration) (bool, error) {
	fd := int(f.Fd())
	if fd < 0 || fd >= unix.FD_SETSIZE {
		return false, errNoSelect
	}
	deadline := time.Now().Add(wait)
	left := wait
	for {
		var set unix.FdSet
		set.Zero()
		set.Set(fd)
		tv := unix.NsecToTimeval(left.Nanoseconds())
		n, err := selectFn(fd+1, &set, nil, nil, &tv)
		if errors.Is(err, unix.EINTR) {
			if left = time.Until(deadline); left > 0 {
				continue
			}
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return n > 0 && set.IsSet(fd), nil
	}
}
