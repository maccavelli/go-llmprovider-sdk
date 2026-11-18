package signinflow

// 0082-PLAN Phase 1's sixth pass (the fifth amendment of 2026-11-18, R-14).

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// markedErr0082 is an error that says by a method that the record it came from
// exists and cannot be decoded, as FileTokenStore's over-size error does.
type markedErr0082 struct {
	msg string
	yes bool
}

func (e *markedErr0082) Error() string     { return e.msg }
func (e *markedErr0082) Undecodable() bool { return e.yes }

// R-14: besides encoding/json's typed errors, an error found through any
// wrapping whose Undecodable method says yes is a decode failure; one whose
// method says no, and one with no such method, is not. Red while the helper
// knows only encoding/json's.
func TestUndecodableKnowsTheStoresOwnMarker_0082(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"a marked error", &markedErr0082{"too large", true}, true},
		{"a marked error, wrapped twice", fmt.Errorf("a: %w", fmt.Errorf("b: %w", &markedErr0082{"too large", true})), true},
		{"a marker that says no", &markedErr0082{"read", false}, false},
		{"an error that merely says larger than", errors.New("openai.client is larger than 65536 bytes"), false},
	}
	for _, c := range cases {
		if got := Undecodable(c.err); got != c.want {
			t.Errorf("%s: Undecodable = %v, want %v", c.name, got, c.want)
		}
	}
}

// R-14: the engine, given FileTokenStore and a client record file over 64 KiB,
// takes it as none: the login runs with none of its client, Result.ClientLoadErr
// carries the error, and the new client replaces the file. Red while an over-size
// file is a read error that fails the sign-in.
func TestPlanOverSizeRecordFromFileTokenStoreIsNone_0082(t *testing.T) {
	store, err := auth.NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Dir, "openai.client")
	big := append(append([]byte(`{"client_id":"oaiapp_big","host_id":"`), bytes.Repeat([]byte("x"), 70<<10)...), `"}`...)
	if err := os.WriteFile(path, big, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadClient(context.Background(), llmprovider.ProviderOpenAI); !Undecodable(err) {
		t.Fatalf("LoadClient = %v; want an error Undecodable recognises", err)
	}
	given, result, err := signInPlan(t, store, auth.ChatGPTPlanOptions{})
	if err != nil || result.Session == nil || given.ClientID != "" || given.HostID != "" {
		t.Fatalf("err %v, session %v, login given client %q: want the sign-in to go on with none of the file's", err, result.Session, given.ClientID)
	}
	if result.ClientLoadErr == nil {
		t.Errorf("ClientLoadErr = nil; want the size error reported")
	}
	rec, err := store.LoadClient(context.Background(), llmprovider.ProviderOpenAI)
	if err != nil || rec.ClientID != "client-issued" {
		t.Errorf("the record after = %+v, %v; want the new client in place of the file", rec, err)
	}
}
