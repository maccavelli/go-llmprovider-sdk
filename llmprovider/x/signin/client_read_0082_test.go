package signin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/signinflow"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// brokenClients is a ClientStore whose read fails with err, and which counts
// its loads and saves.
type brokenClients struct {
	err    error
	record auth.ClientRecord
	loads  int
	saves  int
}

func (b *brokenClients) LoadClient(context.Context, llmprovider.ProviderID) (auth.ClientRecord, error) {
	b.loads++
	return b.record, b.err
}

func (b *brokenClients) SaveClient(_ context.Context, _ llmprovider.ProviderID, c auth.ClientRecord) error {
	b.saves++
	b.record = c
	return nil
}

func runPlanWith(t *testing.T, clients auth.ClientStore) (logins int, res Credential, err error) {
	t.Helper()
	l := signinflow.Logins{Plan: func(_ context.Context, o auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		logins++
		return &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "access-0123456789", ClientID: "client-new", HostID: "urn:uuid:new"}, nil
	}}
	run, startErr := startWith(background(), llmprovider.ProviderOpenAI, llmprovider.AuthChatGPTPlan,
		Options{AgentName: "My App", NoPaste: true, ClientStore: clients}, l)
	if startErr != nil {
		t.Fatalf("start = %v", startErr)
	}
	answerConsent(t, run)
	res, err = run.Wait(bounded(t))
	return logins, res, err
}

// R-12 (0082-MADR, fourth amendment): a caller of x/signin whose ClientStore
// cannot read its record for a reason that is not a decode failure, a
// permission or an input/output error, gets a Wait that fails with that
// error, before any registration, and the record is untouched; Options says
// so. Red while any read error is none.
func TestStart_PlanClientReadErrorFailsWait_0082(t *testing.T) {
	readErr := &fs.PathError{Op: "open", Path: "/x/openai.client", Err: fs.ErrPermission}
	good := auth.ClientRecord{ClientID: "client-mine", HostID: "urn:uuid:mine"}
	clients := &brokenClients{err: readErr, record: good}
	logins, _, err := runPlanWith(t, clients)
	if !errors.Is(err, readErr) || logins != 0 {
		t.Errorf("Wait = %v after %d login(s); want the read error and no registration", err, logins)
	}
	if clients.record != good || clients.saves != 0 || clients.loads != 1 {
		t.Errorf("the record = %+v after %d save(s), %d load(s); want it untouched and read once", clients.record, clients.saves, clients.loads)
	}
}

// R-12: a decode failure, found as encoding/json's error through any wrapping,
// is still none for such a caller: the sign-in goes on and the new client
// replaces the record, with no notice.
func TestStart_PlanClientDecodeFailureIsNone_0082(t *testing.T) {
	var v struct{}
	decode := fmt.Errorf("custom store: %w", json.Unmarshal([]byte("{{"), &v))
	clients := &brokenClients{err: decode, record: auth.ClientRecord{ClientID: "client-stale"}}
	logins, res, err := runPlanWith(t, clients)
	if err != nil || logins != 1 {
		t.Fatalf("Wait = %v after %d login(s); want the sign-in to go on", err, logins)
	}
	if s, ok := res.(Session); !ok || s.OAuth == nil {
		t.Errorf("result %#v; want a session", res)
	}
	if want := (auth.ClientRecord{ClientID: "client-new", HostID: "urn:uuid:new"}); clients.record != want || clients.saves != 1 || clients.loads != 1 {
		t.Errorf("the record = %+v after %d save(s), %d load(s); want %+v saved once, read once", clients.record, clients.saves, clients.loads, want)
	}
}
