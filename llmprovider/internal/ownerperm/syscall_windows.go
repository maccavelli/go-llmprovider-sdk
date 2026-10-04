//go:build windows

package ownerperm

// The advapi32 calls, which doc.go's go:generate line turns into
// zsyscall_windows.go.

//sys	convertStringSecurityDescriptorToSecurityDescriptor(sddl string, revision uint32, sd *uintptr, size *uint32) (err error) = advapi32.ConvertStringSecurityDescriptorToSecurityDescriptorW
//sys	getSecurityDescriptorOwner(sd uintptr, owner *uintptr, defaulted *int32) (err error) = advapi32.GetSecurityDescriptorOwner
//sys	getSecurityDescriptorDacl(sd uintptr, present *int32, dacl *uintptr, defaulted *int32) (err error) = advapi32.GetSecurityDescriptorDacl
//sys	setNamedSecurityInfo(name string, objectType uint32, info uint32, owner uintptr, group uintptr, dacl uintptr, sacl uintptr) (ret error) = advapi32.SetNamedSecurityInfoW

// Values from the Windows SDK: sddl.h, accctrl.h and winnt.h.
const (
	sddlRevision1                    = 1
	seFileObject                     = 1
	ownerSecurityInformation         = 0x00000001
	daclSecurityInformation          = 0x00000004
	protectedDACLSecurityInformation = 0x80000000
)
