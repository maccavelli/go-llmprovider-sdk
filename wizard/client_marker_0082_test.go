package wizard

// 0082-PLAN Phase 1's seventh pass (the sixth amendment of 2026-11-18, the sixth
// review's F-1): a custom ClientStore has exactly one route to "undecodable".

import (
	"context"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// customMarkerErr0082 is what a store outside the module could return: its own
// error type with a method named Undecodable that says yes.
type customMarkerErr0082 struct{}

func (customMarkerErr0082) Error() string     { return "my store: busy, try later" }
func (customMarkerErr0082) Undecodable() bool { return true }

// A custom store whose read error merely has an Undecodable() bool method is
// reporting a read error: the sign-in fails with the store's own error, nothing
// is registered, no notice says the record is replaced, and the record is as it
// was. Wrapping encoding/json's error is the one route (auth.ClientStore says so).
// Red while any error with such a method counts as a decode failure.
func TestCustomStoreErrorWithAMarkerMethodIsAReadError_0082(t *testing.T) {
	ctx := context.Background()
	fs := newClientStore(t)
	store := &countingClients{FileTokenStore: fs, fail: []error{customMarkerErr0082{}}}
	mine := auth.ClientRecord{ClientID: "oaiapp_mine", HostID: "urn:uuid:mine"}
	if err := store.SaveClient(ctx, llmprovider.ProviderOpenAI, mine); err != nil {
		t.Fatal(err)
	}
	calls, _, f, err := planSignIn0082(t, store)
	if err == nil || !strings.Contains(err.Error(), "my store: busy, try later") {
		t.Fatalf("err = %v; want the sign-in failed with the store's own error", err)
	}
	if *calls != 0 || clientNotices0082(f) != 0 {
		t.Errorf("the login ran %d time(s) and notices were %q; want neither", *calls, f.seenNotify)
	}
	if rec, rerr := fs.LoadClient(ctx, llmprovider.ProviderOpenAI); rerr != nil || rec != mine {
		t.Errorf("the record reads %+v, %v; want it untouched", rec, rerr)
	}
}
