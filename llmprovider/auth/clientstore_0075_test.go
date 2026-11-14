package auth

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestFileTokenStore_Client_0075 (0075-MADR D7): save, load, and a zero
// record removes the file; the temp file passes tokenStoreRestrict before its
// first byte, as Save's does, and on Unix the file is mode 0600; Delete of the
// session leaves the record; a bad provider id is ErrInvalidProvider.
func TestFileTokenStore_Client_0075(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := NewFileTokenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := store.LoadClient(ctx, llmprovider.ProviderOpenAI); err != nil || got != (ClientRecord{}) {
		t.Fatalf("LoadClient with no file = %+v, %v; want the zero record", got, err)
	}

	restrict, before := tokenStoreRestrict, tokenStoreBeforeWrite
	t.Cleanup(func() { tokenStoreRestrict, tokenStoreBeforeWrite = restrict, before })
	var events []string
	tokenStoreRestrict = func(f *os.File) error {
		events = append(events, "private "+tempKind(f.Name()))
		return restrict(f)
	}
	tokenStoreBeforeWrite = func(name string) { events = append(events, "first byte "+tempKind(name)) }
	want := ClientRecord{ClientID: "oaiapp_0075", HostID: "urn:uuid:host-0075"}
	if err := store.SaveClient(ctx, llmprovider.ProviderOpenAI, want); err != nil {
		t.Fatal(err)
	}
	tokenStoreRestrict, tokenStoreBeforeWrite = restrict, before
	if wantEvents := []string{"private temp file", "first byte temp file"}; !slices.Equal(events, wantEvents) {
		t.Errorf("events = %q, want %q: the temp file is restricted before its first byte", events, wantEvents)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(dir, "openai.client"))
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Errorf("the client file = %v, %v; want mode 0600", info, err)
		}
	}
	if got, err := store.LoadClient(ctx, llmprovider.ProviderOpenAI); err != nil || got != want {
		t.Errorf("LoadClient = %+v, %v; want %+v", got, err, want)
	}

	if err := store.Save(ctx, llmprovider.ProviderOpenAI, testSession("a-1", "rt-1", time.Now().Add(time.Hour))); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, llmprovider.ProviderOpenAI); err != nil {
		t.Fatal(err)
	}
	if got, err := store.LoadClient(ctx, llmprovider.ProviderOpenAI); err != nil || got != want {
		t.Errorf("LoadClient after Delete of the session = %+v, %v; want the record kept", got, err)
	}

	if err := store.SaveClient(ctx, llmprovider.ProviderOpenAI, ClientRecord{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "openai.client")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("after saving the zero record, Stat = %v; want the file removed", err)
	}
	if err := store.SaveClient(ctx, llmprovider.ProviderOpenAI, ClientRecord{}); err != nil {
		t.Errorf("removing an absent record = %v; want nil", err)
	}

	if _, err := store.LoadClient(ctx, "../bad"); !errors.Is(err, llmprovider.ErrInvalidProvider) {
		t.Errorf("LoadClient with a bad id = %v; want ErrInvalidProvider", err)
	}
	if err := store.SaveClient(ctx, "../bad", want); !errors.Is(err, llmprovider.ErrInvalidProvider) {
		t.Errorf("SaveClient with a bad id = %v; want ErrInvalidProvider", err)
	}
}

// TestPlanClientID_0075 (0075-MADR D7): with ClientID and no Previous, the
// authorize request names the client, has no agent_name_hint and no
// id_token_hint; a callback naming another client is refused; a Previous's
// client wins over ClientID; a first sign-in still names the app.
func TestPlanClientID_0075(t *testing.T) {
	query := func(opts ChatGPTPlanOptions) url.Values {
		t.Helper()
		raw, err := planAuthorizeURL("https://issuer.test/authorize", opts, "urn:uuid:host-0075",
			"http://127.0.0.1:1455/auth/callback", "challenge", "state", "nonce")
		if err != nil {
			t.Fatal(err)
		}
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		return u.Query()
	}
	q := query(ChatGPTPlanOptions{AgentName: "My App", ClientID: "client-0075"})
	if q.Get("client_id") != "client-0075" || q.Has("agent_name_hint") || q.Has("id_token_hint") || q.Get("ext_agent_host_id") != "urn:uuid:host-0075" {
		t.Errorf("the request with a ClientID = %v; want that client and host id, no agent_name_hint, no id_token_hint", q)
	}
	if q := query(ChatGPTPlanOptions{AgentName: "My App"}); q.Get("agent_name_hint") != "My App" || q.Get("client_id") != planDynamicClientID {
		t.Errorf("a first sign-in's request = %v; want the app's name and the dynamic client", q)
	}
	previous := &OAuthSession{ClientID: "client-prev", IDToken: "id-token-prev"}
	if q := query(ChatGPTPlanOptions{AgentName: "My App", ClientID: "client-0075", Previous: previous}); q.Get("client_id") != "client-prev" ||
		q.Get("id_token_hint") != "id-token-prev" || q.Has("agent_name_hint") {
		t.Errorf("a re-sign-in's request = %v; want Previous's client and ID token hint", q)
	}

	known := knownPlanClient(ChatGPTPlanOptions{ClientID: " client-0075 "})
	if known != "client-0075" {
		t.Fatalf("knownPlanClient = %q; want the trimmed ClientID", known)
	}
	if _, err := planIssuedClient(known, "client-other"); err == nil {
		t.Error("a callback naming another client was accepted")
	}
	for _, fromCallback := range []string{"", "client-0075"} {
		if got, err := planIssuedClient(known, fromCallback); err != nil || got != "client-0075" {
			t.Errorf("planIssuedClient(%q) = %q, %v; want the known client", fromCallback, got, err)
		}
	}
}
