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

	"github.com/maccavelli/go-llmprovider-sdk/internal/recorderr"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// methodErr0082 is what a custom ClientStore outside the module could return: an
// error type of its own with a method named Undecodable.
type methodErr0082 struct {
	msg string
	yes bool
}

func (e *methodErr0082) Error() string     { return e.msg }
func (e *methodErr0082) Undecodable() bool { return e.yes }

// R-14, as the sixth amendment narrows it (the sixth review's F-1, the
// coordinator's decision): besides encoding/json's typed errors, the one thing
// Undecodable takes is the module's own marker type, found through any wrapping.
// An error with a method named Undecodable, whatever it says, is not a second
// route for a store of another module; neither is text. This replaces the fifth
// amendment's test of the method route, whose assertion the decision reverses.
// Red while the helper matches a method and not the type.
func TestUndecodableTakesTheModulesMarkerTypeAndNoMethod_0082(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"the marker type", &recorderr.Oversize{Err: errors.New("too large")}, true},
		{"the marker type, wrapped twice", fmt.Errorf("a: %w", fmt.Errorf("b: %w", &recorderr.Oversize{Err: errors.New("too large")})), true},
		{"a custom store's error with the method saying yes", &methodErr0082{"my store: busy", true}, false},
		{"the same, wrapped", fmt.Errorf("load: %w", &methodErr0082{"my store: busy", true}), false},
		{"a custom store's error with the method saying no", &methodErr0082{"read", false}, false},
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
