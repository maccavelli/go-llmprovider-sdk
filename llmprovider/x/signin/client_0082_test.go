package signin

import (
	"context"
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/signinflow"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// unreadableClients is a ClientStore whose record cannot be read.
type unreadableClients struct {
	record auth.ClientRecord
	saves  int
}

func (u *unreadableClients) LoadClient(context.Context, llmprovider.ProviderID) (auth.ClientRecord, error) {
	return auth.ClientRecord{}, errors.New("planted decode failure")
}

func (u *unreadableClients) SaveClient(_ context.Context, _ llmprovider.ProviderID, c auth.ClientRecord) error {
	u.saves++
	u.record = c
	return nil
}

// TestStart_PlanUnreadableClientRecord_0082 (0082-MADR, third amendment, R-9):
// a caller that uses the engine without the wizard, with a ClientStore whose
// record cannot be read, gets a sign-in that goes on as with no record, a
// session, no error and no notice (Options documents it, as it does a record
// that cannot be saved), and the new client replaces the record. Red while
// the sign-in fails on the record.
func TestStart_PlanUnreadableClientRecord_0082(t *testing.T) {
	clients := &unreadableClients{}
	var given auth.ChatGPTPlanOptions
	logins := signinflow.Logins{Plan: func(_ context.Context, o auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		given = o
		return &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "access-0123456789", ClientID: "client-new", HostID: "urn:uuid:new"}, nil
	}}
	run, err := startWith(background(), llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan,
		Options{AgentName: "My App", NoPaste: true, ClientStore: clients}, logins)
	if err != nil {
		t.Fatalf("start = %v", err)
	}
	answerConsent(t, run)
	res, err := run.Wait(bounded(t))
	if err != nil {
		t.Fatalf("Wait = %v; want the sign-in to go on", err)
	}
	if s, ok := res.(Session); !ok || s.OAuth == nil {
		t.Errorf("result %#v; want a session", res)
	}
	if given.ClientID != "" || given.HostID != "" {
		t.Errorf("the login was given client %q, host %q; want neither", given.ClientID, given.HostID)
	}
	if want := (auth.ClientRecord{ClientID: "client-new", HostID: "urn:uuid:new"}); clients.record != want || clients.saves != 1 {
		t.Errorf("the record after the sign-in = %+v after %d save(s); want %+v, saved once", clients.record, clients.saves, want)
	}
}
