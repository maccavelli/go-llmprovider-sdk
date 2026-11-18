package auth

// 0082-PLAN Phase 1's sixth pass (the fifth amendment of 2026-11-18, R-14): a client
// record file over the size bound is corruption and is replaced, where the session
// file over the same bound keeps being refused.

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// undecodableMarker is what signinflow.Undecodable asks of an error besides
// encoding/json's types; the test cannot import signinflow, which imports auth.
type undecodableMarker interface{ Undecodable() bool }

func marked0082(err error) bool {
	var m undecodableMarker
	return errors.As(err, &m) && m.Undecodable()
}

// validRecord0082 is a client record file of exactly size bytes, valid JSON:
// the host id is padded.
func validRecord0082(size int) []byte {
	head, tail := []byte(`{"client_id":"oaiapp_mine","host_id":"OVERSIZE-SECRET`), []byte(`"}`)
	return append(append(head, bytes.Repeat([]byte("x"), size-len(head)-len(tail))...), tail...)
}

// R-14 (0082-MADR, fifth amendment): a client record file over 64 KiB, whatever
// it holds, is reported as a record that cannot be decoded: the error carries
// the marker signinflow.Undecodable recognises, still names the file and the
// bound, and shows nothing of the content. Red while the size check's error is
// a plain read error.
func TestClientRecordOverTheSizeBoundIsUndecodable_0082(t *testing.T) {
	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Dir, "openai.client")
	if err := os.WriteFile(path, validRecord0082(maxTokenFileBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = store.LoadClient(context.Background(), llmprovider.ProviderOpenAI)
	if err == nil || !marked0082(err) {
		t.Fatalf("LoadClient = %v; want an error marked undecodable", err)
	}
	if !strings.Contains(err.Error(), "openai.client is larger than 65536 bytes") || strings.Contains(err.Error(), "OVERSIZE-SECRET") {
		t.Errorf("error %q; want it to name the file and the bound, and none of the content", err)
	}
	// The record is read as it is at the bound, and the file is not touched.
	if err := os.WriteFile(path, validRecord0082(maxTokenFileBytes), 0o600); err != nil {
		t.Fatal(err)
	}
	if rec, err := store.LoadClient(context.Background(), llmprovider.ProviderOpenAI); err != nil || rec.ClientID != "oaiapp_mine" {
		t.Errorf("a record of exactly 64 KiB: %+v, %v; want it read", rec, err)
	}
}

// R-14: only an over-size file is undecodable by this route. A path that cannot
// be read as a file (a directory in its place) stays a plain read error, which
// fails the sign-in. Guards the marker from spreading to every read error.
func TestClientRecordThatCannotBeReadIsNotMarkedUndecodable_0082(t *testing.T) {
	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(store.Dir, "openai.client"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err = store.LoadClient(context.Background(), llmprovider.ProviderOpenAI)
	if err == nil || marked0082(err) {
		t.Errorf("LoadClient = %v, marked %v; want a read error that is not marked undecodable", err, marked0082(err))
	}
}

// R-14, the proof the session file is unchanged: readBounded is shared, and a
// session file over 64 KiB is still refused with the same error text, an error
// that is not marked undecodable (the marker is only LoadClient's), and the file
// is left as it is. A guard of code R-14 does not touch; the plant that gives the
// session path the client path's error fails it.
func TestSessionFileOverTheSizeBoundKeepsBeingRefused_0082(t *testing.T) {
	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Dir, "openai.json")
	big := validRecord0082(maxTokenFileBytes + 1)
	if err := os.WriteFile(path, big, 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := store.Load(context.Background(), llmprovider.ProviderOpenAI)
	if s != nil || err == nil || err.Error() != "FileTokenStore read: openai.json is larger than 65536 bytes" {
		t.Errorf("Load = %v, %v; want no session and the error \"FileTokenStore read: openai.json is larger than 65536 bytes\"", s, err)
	}
	if marked0082(err) {
		t.Errorf("the session file's size error is marked undecodable: %v", err)
	}
	if after, rerr := os.ReadFile(path); rerr != nil || !bytes.Equal(after, big) {
		t.Errorf("the session file after Load = %d bytes, %v; want it as it was", len(after), rerr)
	}
}
