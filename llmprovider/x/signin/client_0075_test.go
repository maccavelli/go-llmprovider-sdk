package signin

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/signinflow"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// memoryClients is a ClientStore in memory.
type memoryClients struct{ record auth.ClientRecord }

func (m *memoryClients) LoadClient(context.Context, llmprovider.ProviderID) (auth.ClientRecord, error) {
	return m.record, nil
}

func (m *memoryClients) SaveClient(_ context.Context, _ llmprovider.ProviderID, c auth.ClientRecord) error {
	m.record = c
	return nil
}

// TestStart_PlanClientStore_0075 (0075-MADR D6, D7): Options.HostID and
// Options.ClientStore reach the plan login: the stored client is signed in
// to, the option's host id comes before the record's, and the client and host
// id used are saved once the sign-in returns.
func TestStart_PlanClientStore_0075(t *testing.T) {
	clients := &memoryClients{record: auth.ClientRecord{ClientID: "client-rec", HostID: "urn:uuid:rec"}}
	var given auth.ChatGPTPlanOptions
	logins := signinflow.Logins{Plan: func(_ context.Context, o auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		given = o
		return &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "access-0123456789", ClientID: o.ClientID, HostID: o.HostID}, nil
	}}
	run, err := startWith(background(), llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan,
		Options{AgentName: "My App", NoPaste: true, HostID: "urn:uuid:opt", ClientStore: clients}, logins)
	if err != nil {
		t.Fatalf("start = %v", err)
	}
	answerConsent(t, run)
	if _, err := run.Wait(bounded(t)); err != nil {
		t.Fatalf("Wait = %v", err)
	}
	if given.ClientID != "client-rec" || given.HostID != "urn:uuid:opt" {
		t.Errorf("the login was given client %q, host %q; want the record's client and the option's host id", given.ClientID, given.HostID)
	}
	if want := (auth.ClientRecord{ClientID: "client-rec", HostID: "urn:uuid:opt"}); clients.record != want {
		t.Errorf("the record after the sign-in = %+v; want %+v", clients.record, want)
	}
}
