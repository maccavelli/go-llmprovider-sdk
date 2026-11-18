package signin

// 0082-PLAN Phase 1's sixth pass (the fifth amendment of 2026-11-18, R-14).

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// R-14: a caller of x/signin whose ClientStore is a FileTokenStore with a client
// record file over 64 KiB gets a sign-in that goes on, with no notice from this
// package, and the new client replaces the file; `Options.ClientStore` says so.
// Red while an over-size file is a read error that fails Wait.
func TestStart_PlanOverSizeClientRecordIsReplaced_0082(t *testing.T) {
	store, err := auth.NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	big := append(append([]byte(`{"client_id":"oaiapp_big","host_id":"`), bytes.Repeat([]byte("x"), 70<<10)...), `"}`...)
	if err := os.WriteFile(filepath.Join(store.Dir, "openai.client"), big, 0o600); err != nil {
		t.Fatal(err)
	}
	logins, res, err := runPlanWith(t, store)
	if err != nil || logins != 1 {
		t.Fatalf("Wait = %v after %d login(s); want the sign-in to go on", err, logins)
	}
	if s, ok := res.(Session); !ok || s.OAuth == nil {
		t.Errorf("result %#v; want a session", res)
	}
	rec, err := store.LoadClient(context.Background(), llmprovider.ProviderOpenAI)
	if err != nil || rec != (auth.ClientRecord{ClientID: "client-new", HostID: "urn:uuid:new"}) {
		t.Errorf("the record after = %+v, %v; want the new client in place of the file", rec, err)
	}
}
