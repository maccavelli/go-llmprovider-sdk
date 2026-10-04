//go:build windows

package ownerperm

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"unsafe"
)

// The read-back calls are test-only, so they stay out of the generated file.
var (
	procGetNamedSecurityInfoW                                = modadvapi32.NewProc("GetNamedSecurityInfoW")
	procConvertSecurityDescriptorToStringSecurityDescriptorW = modadvapi32.NewProc("ConvertSecurityDescriptorToStringSecurityDescriptorW")
)

// sddlOf reads path's owner and DACL back as SDDL.
func sddlOf(t *testing.T, path string) string {
	t.Helper()
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	const info = ownerSecurityInformation | daclSecurityInformation
	var sd uintptr
	r1, _, _ := syscall.SyscallN(procGetNamedSecurityInfoW.Addr(), uintptr(unsafe.Pointer(p)), seFileObject, info,
		0, 0, 0, 0, uintptr(unsafe.Pointer(&sd)))
	if r1 != 0 {
		t.Fatalf("GetNamedSecurityInfo(%s): %v", path, syscall.Errno(r1))
	}
	defer syscall.LocalFree(syscall.Handle(sd))
	var s *uint16
	r1, _, e1 := syscall.SyscallN(procConvertSecurityDescriptorToStringSecurityDescriptorW.Addr(), sd, sddlRevision1, info,
		uintptr(unsafe.Pointer(&s)), 0)
	if r1 == 0 {
		t.Fatalf("ConvertSecurityDescriptorToStringSecurityDescriptor(%s): %v", path, e1)
	}
	defer syscall.LocalFree(syscall.Handle(uintptr(unsafe.Pointer(s))))
	n := 0
	for *(*uint16)(unsafe.Add(unsafe.Pointer(s), 2*n)) != 0 {
		n++
	}
	return syscall.UTF16ToString(unsafe.Slice(s, n))
}

// sidOf is an SDDL trustee as a SID string. SDDL writes a well-known account
// as a two-letter alias, such as LA for the built-in Administrator (RID 500),
// which a CI runner's account is; ConvertStringSidToSidW takes either form
// (0010-PLAN, deviation 2026-10-04).
func sidOf(t *testing.T, trustee string) string {
	t.Helper()
	sid, err := syscall.StringToSid(trustee)
	if err != nil {
		t.Fatalf("SDDL trustee %q: %v", trustee, err)
	}
	s, err := sid.String()
	if err != nil {
		t.Fatalf("SDDL trustee %q: %v", trustee, err)
	}
	return s
}

// assertOnlyUser checks the recipe's own invariants (0010-MADR open question
// 3): the user owns path, and every entry grants the user full access with
// the wanted inheritance flags. A DACL set directly is protected; one only
// inherited ("ID") is auto-inherited instead, as no inherited DACL can be
// protected (0010-PLAN, deviation 2026-10-04).
func assertOnlyUser(t *testing.T, path, wantFlags string) {
	t.Helper()
	sid, err := currentUserSID()
	if err != nil {
		t.Fatal(err)
	}
	got := sddlOf(t, path)
	owner, dacl, ok := strings.Cut(strings.TrimPrefix(got, "O:"), "D:")
	if !ok || sidOf(t, owner) != sid {
		t.Fatalf("%s: SDDL %q, want owner %s", path, got, sid)
	}
	flags, aces, _ := strings.Cut(dacl, "(")
	switch {
	case wantFlags == "ID" && !strings.Contains(flags, "AI"):
		t.Errorf("%s: SDDL %q, want an auto-inherited DACL (AI)", path, got)
	case wantFlags != "ID" && !strings.Contains(flags, "P"):
		t.Errorf("%s: SDDL %q, want a protected DACL (P)", path, got)
	}
	for ace := range strings.SplitSeq(strings.TrimSuffix(aces, ")"), ")(") {
		f := strings.Split(ace, ";")
		if len(f) < 6 || f[0] != "A" || f[1] != wantFlags || (f[2] != "FA" && f[2] != "0x1f01ff") || sidOf(t, f[5]) != sid {
			t.Errorf("%s: entry (%s), want (A;%s;FA;;;%s)", path, ace, wantFlags, sid)
		}
	}
}

// TestMkdirAllAndFile_OnlyTheCurrentUser: the directory is restricted and
// inheritable, a file created in it is restricted from its creation, and File
// gives the file its own entry.
func TestMkdirAllAndFile_OnlyTheCurrentUser(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tokens")
	if err := MkdirAll(dir); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	assertOnlyUser(t, dir, "OICI")
	f, err := os.CreateTemp(dir, ".tok-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	assertOnlyUser(t, f.Name(), "ID")
	if err := File(f); err != nil {
		t.Fatalf("File: %v", err)
	}
	assertOnlyUser(t, f.Name(), "")
}
