//go:build windows

package ownerperm

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// MkdirAll creates dir, and any parent it needs, then restricts dir to the
// current user with an entry that what is created in it inherits. Windows
// also applies the entry to what dir already holds.
func MkdirAll(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return restrict(dir, "OICI")
}

// File restricts f to the current user. Call it before writing anything
// secret.
func File(f *os.File) error {
	return restrict(f.Name(), "")
}

// restrict makes the current user path's owner and the only entry of its
// protected DACL. aceFlags are the entry's SDDL inheritance flags.
func restrict(path, aceFlags string) (err error) {
	sid, err := currentUserSID()
	if err != nil {
		return fmt.Errorf("ownerperm: current user: %w", err)
	}
	var sd uintptr
	if err := convertStringSecurityDescriptorToSecurityDescriptor(
		"O:"+sid+"D:P(A;"+aceFlags+";FA;;;"+sid+")", sddlRevision1, &sd, nil); err != nil {
		return fmt.Errorf("ownerperm: security descriptor: %w", err)
	}
	defer func() {
		if _, ferr := syscall.LocalFree(syscall.Handle(sd)); ferr != nil {
			err = errors.Join(err, fmt.Errorf("ownerperm: free: %w", ferr))
		}
	}()
	var owner, dacl uintptr
	var present, defaulted int32
	if err := getSecurityDescriptorOwner(sd, &owner, &defaulted); err != nil {
		return fmt.Errorf("ownerperm: owner: %w", err)
	}
	if err := getSecurityDescriptorDacl(sd, &present, &dacl, &defaulted); err != nil {
		return fmt.Errorf("ownerperm: DACL: %w", err)
	}
	if err := setNamedSecurityInfo(path, seFileObject,
		ownerSecurityInformation|daclSecurityInformation|protectedDACLSecurityInformation,
		owner, 0, dacl, 0); err != nil {
		return fmt.Errorf("ownerperm: %s: %w", path, err)
	}
	return nil
}

// currentUserSID is the process token's user, as a SID string.
func currentUserSID() (sid string, err error) {
	token, err := syscall.OpenCurrentProcessToken()
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, token.Close()) }()
	user, err := token.GetTokenUser()
	if err != nil {
		return "", err
	}
	return user.User.Sid.String()
}
