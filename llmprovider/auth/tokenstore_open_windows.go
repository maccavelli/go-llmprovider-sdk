//go:build windows

package auth

import (
	"os"
	"path/filepath"
	"syscall"
)

// openShared opens path for reading, sharing read, write and delete access.
// os.Open does not share delete, so while another process reads a token
// file, a save's rename onto it fails; with FILE_SHARE_DELETE the rename
// succeeds and the reader keeps the file it opened (0026-MADR F42).
func openShared(path string) (*os.File, error) {
	path = filepath.Clean(path)
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_READ,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	return os.NewFile(uintptr(h), path), nil
}
