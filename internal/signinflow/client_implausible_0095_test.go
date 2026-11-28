package signinflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestPlanImplausibleRecordIsNone_0095 (0095-MADR FU10; 0082-MADR N-a): the
// engine, given FileTokenStore and a client record that decodes but names a
// client id that is not printable text, takes it as none, as an undecodable
// record: the login runs with none of its client, Result.ClientLoadErr
// carries the error, and the new client replaces the file.
func TestPlanImplausibleRecordIsNone_0095(t *testing.T) {
	store, err := auth.NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Dir, "openai.client")
	if err := os.WriteFile(path, []byte("{\"client_id\":\"oaiapp_\xff\xfe\",\"host_id\":\"h1\"}"), 0o600); err != nil {
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
		t.Errorf("ClientLoadErr = nil; want the record's error reported")
	}
	rec, err := store.LoadClient(context.Background(), llmprovider.ProviderOpenAI)
	if err != nil || rec.ClientID != "client-issued" {
		t.Errorf("the record after = %+v, %v; want the new client in place of the file", rec, err)
	}
}
