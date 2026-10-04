package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestFileTokenStore_RejectsReservedNames (0010-MADR D12): a provider id that
// Windows would turn into a device, or would silently change, is refused on
// every OS, so a store copied onto Windows cannot hold one.
func TestFileTokenStore_RejectsReservedNames(t *testing.T) {
	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	session := &OAuthSession{Access: "at", Refresh: "rt"}
	for _, id := range []llmprovider.ProviderID{
		"CON", "con", "NUL", "COM1", "LPT9", "CON.json", "COM1.txt", "aux", "foo.", "foo ",
	} {
		if err := store.Save(context.Background(), id, session); !errors.Is(err, llmprovider.ErrInvalidProvider) {
			t.Errorf("Save(%q) = %v, want ErrInvalidProvider", id, err)
		}
	}
	for _, id := range []llmprovider.ProviderID{"console", "com10", "nullable", "openai"} {
		if err := store.Save(context.Background(), id, session); err != nil {
			t.Errorf("Save(%q) = %v, want nil: not a reserved name", id, err)
		}
	}
}

// TestFileTokenStore_OverwriteExisting (0010-MADR D13): a second Save
// replaces the first. It pins behaviour the rename already gives.
func TestFileTokenStore_OverwriteExisting(t *testing.T) {
	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, access := range []string{"at-A", "at-B"} {
		if err := store.Save(ctx, "openai", &OAuthSession{Access: access, Refresh: "rt"}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.Load(ctx, "openai")
	if err != nil || got == nil || got.Access != "at-B" {
		t.Fatalf("Load after two saves = %+v, %v; want the second session (at-B)", got, err)
	}
}
