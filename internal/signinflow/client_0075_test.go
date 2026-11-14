package signinflow

import (
	"cmp"
	"context"
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// recordStore is a ClientStore in memory that counts its loads.
type recordStore struct {
	record  auth.ClientRecord
	loads   int
	saves   int
	loadErr error
	saveErr error
}

func (s *recordStore) LoadClient(context.Context, llmprovider.ProviderID) (auth.ClientRecord, error) {
	s.loads++
	return s.record, s.loadErr
}

func (s *recordStore) SaveClient(_ context.Context, _ llmprovider.ProviderID, c auth.ClientRecord) error {
	s.saves++
	if s.saveErr != nil {
		return s.saveErr
	}
	s.record = c
	return nil
}

// planNoConsent is the plan method with no consent, so that no confirm step
// stands before the login.
func planNoConsent() authcontract.Method {
	m := planConsentMethod()
	m.Consents = nil
	return m
}

// signInPlan runs a plan sign-in whose login records the options it was given
// and returns a session with the issued client and the host id it names.
func signInPlan(t *testing.T, store auth.ClientStore, plan auth.ChatGPTPlanOptions) (auth.ChatGPTPlanOptions, Result, error) {
	t.Helper()
	var given auth.ChatGPTPlanOptions
	run, err := Start(background(), Spec{Provider: llmprovider.ProviderOpenAI, Method: planNoConsent(), Mode: ModePlan,
		First: plan.Previous == nil, Plan: plan, ClientStore: store, NoPaste: true,
		Logins: Logins{Plan: func(_ context.Context, o auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
			given = o
			hostID, clientID := o.HostID, o.ClientID
			if o.Previous != nil {
				hostID, clientID = cmp.Or(hostID, o.Previous.HostID), o.Previous.ClientID
			}
			hostID = cmp.Or(hostID, "urn:uuid:minted")
			clientID = cmp.Or(clientID, "client-issued")
			return &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "access-0123456789", ClientID: clientID, HostID: hostID}, nil
		}}})
	if err != nil {
		t.Fatalf("Start = %v", err)
	}
	result, err := run.Wait(bounded(t))
	return given, result, err
}

// TestHostIDOrder_0075 (0075-MADR D6, D7): Options.HostID, then Previous's,
// then the record's when there is no Previous, then a new one, which is
// saved; the record is loaded only when there is no Previous, and its client
// is the one signed in to.
func TestHostIDOrder_0075(t *testing.T) {
	record := auth.ClientRecord{ClientID: "client-rec", HostID: "urn:uuid:rec"}

	t.Run("the option beats the record", func(t *testing.T) {
		store := &recordStore{record: record}
		given, _, err := signInPlan(t, store, auth.ChatGPTPlanOptions{HostID: "urn:uuid:opt"})
		if err != nil || given.HostID != "urn:uuid:opt" || given.ClientID != "client-rec" {
			t.Errorf("login given host %q, client %q, err %v; want the option's host id and the record's client", given.HostID, given.ClientID, err)
		}
		if store.record.HostID != "urn:uuid:opt" || store.record.ClientID != "client-rec" {
			t.Errorf("the record = %+v after the sign-in; want the host id used saved", store.record)
		}
	})
	t.Run("the record with no Previous", func(t *testing.T) {
		store := &recordStore{record: record}
		given, _, err := signInPlan(t, store, auth.ChatGPTPlanOptions{})
		if err != nil || given.HostID != "urn:uuid:rec" || given.ClientID != "client-rec" {
			t.Errorf("login given host %q, client %q, err %v; want the record's", given.HostID, given.ClientID, err)
		}
	})
	t.Run("Previous before the record", func(t *testing.T) {
		store := &recordStore{record: record}
		previous := &auth.OAuthSession{ClientID: "client-prev", HostID: "urn:uuid:prev"}
		given, _, err := signInPlan(t, store, auth.ChatGPTPlanOptions{Previous: previous})
		if err != nil || given.HostID != "" || given.ClientID != "" || given.Previous != previous {
			t.Errorf("login given host %q, client %q, Previous %v, err %v; want Previous alone, for auth to use its host id and client", given.HostID, given.ClientID, given.Previous, err)
		}
		if store.loads != 0 {
			t.Errorf("the record was loaded %d time(s) with a Previous; want none", store.loads)
		}
		if store.record.ClientID != "client-prev" || store.record.HostID != "urn:uuid:prev" {
			t.Errorf("the record = %+v; want the session's client and host id saved", store.record)
		}
	})
	t.Run("a new one, saved", func(t *testing.T) {
		store := &recordStore{}
		given, _, err := signInPlan(t, store, auth.ChatGPTPlanOptions{})
		if err != nil || given.HostID != "" || given.ClientID != "" {
			t.Errorf("login given host %q, client %q, err %v; want neither: auth mints", given.HostID, given.ClientID, err)
		}
		if store.record.HostID != "urn:uuid:minted" || store.record.ClientID != "client-issued" || store.saves != 1 {
			t.Errorf("the record = %+v after %d save(s); want the issued client and the minted host id saved once", store.record, store.saves)
		}
	})
}

// TestPlanClientRecordErrors_0075 (0075-MADR D7): a record that cannot be
// loaded fails the sign-in before the login runs; one that cannot be saved is
// reported and does not fail it.
func TestPlanClientRecordErrors_0075(t *testing.T) {
	loadErr := errors.New("planted load failure")
	store := &recordStore{loadErr: loadErr}
	given, _, err := signInPlan(t, store, auth.ChatGPTPlanOptions{})
	if !errors.Is(err, loadErr) || given.OpenURL != nil {
		t.Errorf("sign-in with a failing LoadClient: err %v, login ran %t; want the load error and no login", err, given.OpenURL != nil)
	}
	saveErr := errors.New("planted save failure")
	store = &recordStore{saveErr: saveErr}
	_, result, err := signInPlan(t, store, auth.ChatGPTPlanOptions{})
	if err != nil || result.Session == nil || !errors.Is(result.ClientSaveErr, saveErr) {
		t.Errorf("sign-in with a failing SaveClient: err %v, session %v, ClientSaveErr %v; want the session and the error reported", err, result.Session, result.ClientSaveErr)
	}
}
