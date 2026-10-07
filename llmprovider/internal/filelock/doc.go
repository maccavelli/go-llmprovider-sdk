// Package filelock takes an exclusive, advisory lock on an open file. The
// operating system releases it when the file is closed or its process ends,
// so a holder that dies leaves nothing stale behind (0026-MADR F13, Q1 a).
//
// On Unix it is flock(2), on Windows LockFileEx. Both lock per open file, not
// per process, so two handles in one process exclude each other as two
// processes do.
//
// It uses only the standard library (0015-MADR D2). The Windows calls are
// declared in syscall_windows.go and generated into zsyscall_windows.go by
// mkwinsyscall, as in internal/ownerperm. -systemdll=false keeps the output on
// syscall alone: kernel32.dll is a DLL Go itself uses, so syscall loads it
// from System32 only. The generator runs at development time, on any host,
// and is not a module requirement.
package filelock

import "errors"

//go:generate go run golang.org/x/sys/windows/mkwinsyscall@v0.47.0 -systemdll=false -output zsyscall_windows.go syscall_windows.go

// ErrLocked reports that another holder has the lock.
var ErrLocked = errors.New("filelock: the file is locked by another holder")
