//go:build !windows

package wizard

// 0082-PLAN Phase 1's fifth pass (the fourth amendment of 2026-11-18, R-12): the
// permission case on a real file. The mode 0 that refuses a read here does not on
// Windows, where a file's owner reads it whatever its mode; the same case with
// the error injected, which every platform runs, is
// TestClientRecordPermissionErrorFailsTheSignIn_0082.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// R-12 (0082-MADR, fourth amendment): a client record that exists but cannot be
// read for a reason other than its content, here a permission refused by the
// file's own mode, fails the sign-in with an error that names the file and none
// of its content. The login does not run, so no client is registered; the
// record and the saved session are as they were; no notice says the record is
// replaced. Red while any read error is an unreadable record.
func TestClientRecordPermissionDeniedFailsTheSignIn_0082(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("a permission is not refused to root")
	}
	ctx := context.Background()
	store := newClientStore(t)
	mine := auth.ClientRecord{ClientID: "oaiapp_mine", HostID: "urn:uuid:SECRET-HOST"}
	if err := store.SaveClient(ctx, llmprovider.ProviderOpenAI, mine); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, llmprovider.ProviderOpenAI, planSessionFor("oaiapp_mine")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Dir, "openai.client")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
	calls, _, f, err := planSignIn0082(t, store)
	if err == nil || !strings.Contains(err.Error(), "openai.client") {
		t.Fatalf("err = %v; want the sign-in failed, naming openai.client", err)
	}
	for _, bad := range []string{"SECRET-HOST", "oaiapp_mine", "urn:uuid"} {
		if strings.Contains(err.Error(), bad) {
			t.Errorf("the error %q carries %q from the record", err, bad)
		}
	}
	if *calls != 0 {
		t.Errorf("the login ran %d time(s); want none, so that no client is registered", *calls)
	}
	if n := clientNotices0082(f); n != 0 {
		t.Errorf("notices %q; want none about the record: it is not replaced", f.seenNotify)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(before) {
		t.Errorf("the record after = %q, %v; want it as it was", after, err)
	}
	if rec, err := store.LoadClient(ctx, llmprovider.ProviderOpenAI); err != nil || rec != mine {
		t.Errorf("the record reads %+v, %v; want %+v", rec, err, mine)
	}
	if s, err := store.Load(ctx, llmprovider.ProviderOpenAI); err != nil || s == nil || s.ClientID != "oaiapp_mine" {
		t.Errorf("the saved session = %v, %v; want it untouched", s, err)
	}
}
