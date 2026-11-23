package wizard

import (
	"testing"

	"golang.org/x/sys/unix"
)

// fionread0087 is macOS's FIONREAD, which golang.org/x/sys does not name: the
// SDK's sys/filio.h defines it as _IOR('f', 127, int), which sys/ioccom.h's
// _IOC makes IOC_OUT (0x40000000) | (sizeof(int) & IOCPARM_MASK) << 16 |
// 'f' << 8 | 127 (0087-PLAN, ST14).
const fionread0087 = 0x4004667f

// pendingCount0087 is the number of bytes a descriptor holds unread.
func pendingCount0087(fd int) (int, error) { return unix.IoctlGetInt(fd, fionread0087) }

// TestSecret_NoReadLeftBehind_0087 (ST14): see noReadLeftBehind0087.
func TestSecret_NoReadLeftBehind_0087(t *testing.T) { noReadLeftBehind0087(t, pendingCount0087) }
