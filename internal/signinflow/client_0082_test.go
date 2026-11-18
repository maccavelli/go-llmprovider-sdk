package signinflow

import (
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// R-9 (0082-MADR, third amendment): a client record that cannot be read is no
// record. The sign-in runs, the login is given no client and no host id of
// the record's, the error is in Result.ClientLoadErr, and the client the
// sign-in registered replaces the unreadable record, saved once. Red while the
// engine fails the sign-in on the record.
func TestPlanUnreadableClientRecordIsNone_0082(t *testing.T) {
	loadErr := errors.New("planted decode failure: invalid character")
	// A store may return a partial record with its error: none of it is used.
	store := &recordStore{loadErr: loadErr, record: auth.ClientRecord{ClientID: "stale-client", HostID: "urn:uuid:stale"}}
	given, result, err := signInPlan(t, store, auth.ChatGPTPlanOptions{})
	if err != nil || result.Session == nil {
		t.Fatalf("sign-in with an unreadable record: err %v, session %v; want it to go on", err, result.Session)
	}
	if given.OpenURL == nil || given.ClientID != "" || given.HostID != "" {
		t.Errorf("login ran %t with client %q, host %q; want it run with neither of the record's", given.OpenURL != nil, given.ClientID, given.HostID)
	}
	if !errors.Is(result.ClientLoadErr, loadErr) {
		t.Errorf("ClientLoadErr = %v; want the read error", result.ClientLoadErr)
	}
	if store.saves != 1 || store.record.ClientID != "client-issued" || store.record.HostID != "urn:uuid:minted" {
		t.Errorf("the record = %+v after %d save(s); want the new client saved once, replacing the unreadable one", store.record, store.saves)
	}
	if result.ClientSaveErr != nil {
		t.Errorf("ClientSaveErr = %v, want none", result.ClientSaveErr)
	}
}
