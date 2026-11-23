package wizard

import (
	"testing"

	"golang.org/x/sys/unix"
)

// pendingCount0087 is the number of bytes a descriptor holds unread: Linux's
// TIOCINQ (0x541b), which golang.org/x/sys names (0087-PLAN, ST14).
func pendingCount0087(fd int) (int, error) { return unix.IoctlGetInt(fd, unix.TIOCINQ) }

// TestSecret_NoReadLeftBehind_0087 (ST14): see noReadLeftBehind0087.
func TestSecret_NoReadLeftBehind_0087(t *testing.T) { noReadLeftBehind0087(t, pendingCount0087) }
