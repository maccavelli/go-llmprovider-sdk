//go:build windows

package filelock

// The kernel32 calls, which doc.go's go:generate line turns into
// zsyscall_windows.go.

//sys	lockFileEx(file syscall.Handle, flags uint32, reserved uint32, bytesLow uint32, bytesHigh uint32, overlapped *syscall.Overlapped) (err error) = kernel32.LockFileEx
//sys	unlockFileEx(file syscall.Handle, reserved uint32, bytesLow uint32, bytesHigh uint32, overlapped *syscall.Overlapped) (err error) = kernel32.UnlockFileEx

// Values from the Windows SDK: minwinbase.h and winerror.h.
const (
	lockfileFailImmediately = 0x00000001
	lockfileExclusiveLock   = 0x00000002
	errorLockViolation      = 33
)
