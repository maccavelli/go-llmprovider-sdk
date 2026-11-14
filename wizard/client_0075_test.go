package wizard

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// planClientSignIn runs ResolveCredential's plan sign-in, with the consent
// allowed, over a FileTokenStore, and returns the options its login was given.
func planClientSignIn(t *testing.T, store auth.TokenStore, hostID string) auth.ChatGPTPlanOptions {
	t.Helper()
	var given auth.ChatGPTPlanOptions
	stubPlanLogin(t, func(_ context.Context, opts auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		given = opts
		return planSessionFor("oaiapp_0075"), nil
	})
	f := newFake(t, fakePrompter{selects: []int{planMethodIdx(t), 0}})
	if _, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: store, AgentName: planTestApp,
		NoPastePrompt: true, HostID: hostID}, llmprovider.ProviderOpenAI); err != nil {
		t.Fatal(err)
	}
	return given
}

// orderedStore is a FileTokenStore that records the order of its saves.
type orderedStore struct {
	*auth.FileTokenStore
	events *[]string
}

func (s orderedStore) Save(ctx context.Context, provider llmprovider.ProviderID, session *auth.OAuthSession) error {
	*s.events = append(*s.events, "Save")
	return s.FileTokenStore.Save(ctx, provider, session)
}

func (s orderedStore) SaveClient(ctx context.Context, provider llmprovider.ProviderID, c auth.ClientRecord) error {
	*s.events = append(*s.events, "SaveClient")
	return s.FileTokenStore.SaveClient(ctx, provider, c)
}

func newClientStore(t *testing.T) *auth.FileTokenStore {
	t.Helper()
	store, err := auth.NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return store
}

// TestWizardPlanReusesClient_0075 (0075-MADR D7): after a first sign-in the
// issued client and the host id are saved at once, before the session is, and
// so before any listing; after the session is deleted, as Logout does at its
// last step, a plan sign-in passes the stored client and host id; an
// Options.HostID comes first.
func TestWizardPlanReusesClient_0075(t *testing.T) {
	ctx := context.Background()
	store := newClientStore(t)
	var events []string
	first := planClientSignIn(t, orderedStore{store, &events}, "")
	if len(events) == 0 || events[0] != "SaveClient" {
		t.Errorf("saves in the order %q; want the client record first, as the sign-in returns, before the session", events)
	}
	if first.ClientID != "" || first.Previous != nil {
		t.Errorf("a first sign-in was given client %q, Previous %v; want neither", first.ClientID, first.Previous)
	}
	record, err := store.LoadClient(ctx, llmprovider.ProviderOpenAI)
	if err != nil || record.ClientID != "oaiapp_0075" || record.HostID != "urn:uuid:oaiapp_0075" {
		t.Fatalf("the record after the sign-in = %+v, %v; want the issued client and host id", record, err)
	}

	// The session is signed out: Logout's last step deletes it.
	if err := store.Delete(ctx, llmprovider.ProviderOpenAI); err != nil {
		t.Fatal(err)
	}
	again := planClientSignIn(t, store, "")
	if again.ClientID != "oaiapp_0075" || again.HostID != "urn:uuid:oaiapp_0075" || again.Previous != nil {
		t.Errorf("a sign-in after the sign-out was given client %q, host %q, Previous %v; want the stored client and host id",
			again.ClientID, again.HostID, again.Previous)
	}
	if err := store.Delete(ctx, llmprovider.ProviderOpenAI); err != nil {
		t.Fatal(err)
	}
	if own := planClientSignIn(t, store, "urn:uuid:host-0075"); own.HostID != "urn:uuid:host-0075" || own.ClientID != "oaiapp_0075" {
		t.Errorf("a sign-in with Options.HostID was given host %q, client %q; want the option's host id, the stored client", own.HostID, own.ClientID)
	}
}

// TestWizardPlanSwitchKeepsClient_0075 (0075-MADR D3, D7): a plan session
// replaced by a Codex-id one, then a plan sign-in, reuses the client.
func TestWizardPlanSwitchKeepsClient_0075(t *testing.T) {
	ctx := context.Background()
	store := newClientStore(t)
	planClientSignIn(t, store, "")
	if err := store.Save(ctx, llmprovider.ProviderOpenAI, planSessionFor("oaiapp_0075")); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, llmprovider.ProviderOpenAI, testOAuthSession(llmprovider.ProviderOpenAI)); err != nil {
		t.Fatal(err)
	}
	given := planClientSignIn(t, store, "")
	if given.ClientID != "oaiapp_0075" || given.Previous != nil {
		t.Errorf("a plan sign-in after a Codex-id session was given client %q, Previous %v; want the stored client and no Previous",
			given.ClientID, given.Previous)
	}
}
