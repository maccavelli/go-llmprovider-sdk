// Package ownerperm keeps a directory and the files written in it private to
// the user running the process (0010-MADR D10).
//
// On Unix the directory is created with mode 0o700 and each file is set to
// 0o600. On Windows the directory gets a protected DACL whose one entry, full
// access for the current user, is inherited by everything created in it, and
// each file gets the same entry, not inherited; the user also becomes the
// owner.
//
// It uses only the standard library (0015-MADR D2). The Windows calls are
// declared in syscall_windows.go and generated into zsyscall_windows.go by
// mkwinsyscall, the generator the standard library and golang.org/x/sys use.
// -systemdll=false keeps the output on syscall alone: advapi32.dll is a DLL
// Go itself uses, so syscall loads it from System32 only (syscall.LoadDLL).
// The generator runs at development time, on any host, and is not a module
// requirement.
package ownerperm

//go:generate go run golang.org/x/sys/windows/mkwinsyscall@v0.47.0 -systemdll=false -output zsyscall_windows.go syscall_windows.go
