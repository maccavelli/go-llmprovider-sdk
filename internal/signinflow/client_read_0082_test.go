package signinflow

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// decodeFailure0082 is the error a ClientStore returns for a record that exists
// but cannot be decoded: encoding/json's, wrapped as FileTokenStore wraps it.
func decodeFailure0082() error {
	var v struct{}
	err := json.Unmarshal([]byte("{{ junk"), &v)
	return fmt.Errorf("store client decode: %w", err)
}

// R-12 (0082-MADR, fourth amendment): what tells a record that exists but cannot
// be decoded from any other read error is encoding/json's typed errors, found
// through the wrapping, and no text. A permission error, an input/output error,
// a file that is too large, a context error and nil are not decode failures.
// Red while the helper says no to every error.
func TestUndecodable_0082(t *testing.T) {
	var syntax, typed error
	var s struct{}
	syntax = json.Unmarshal([]byte("{"), &s)
	var rec struct{ A int }
	typed = json.Unmarshal([]byte(`{"A":"x"}`), &rec)
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"a syntax error", syntax, true},
		{"a type error", typed, true},
		{"a syntax error, wrapped twice", fmt.Errorf("a: %w", fmt.Errorf("b: %w", syntax)), true},
		{"the store's decode failure", decodeFailure0082(), true},
		{"a permission error", &fs.PathError{Op: "open", Path: "openai.client", Err: fs.ErrPermission}, false},
		{"a plain error that says decode", errors.New("FileTokenStore client decode: planted"), false},
		{"a size error", errors.New("openai.client is larger than 65536 bytes"), false},
		{"nil", nil, false},
	}
	for _, c := range cases {
		if got := Undecodable(c.err); got != c.want {
			t.Errorf("%s: Undecodable = %v, want %v", c.name, got, c.want)
		}
	}
}

// signInPlanRead is signInPlan with the caller's own read of the record given
// to the engine.
func signInPlanRead(t *testing.T, store auth.ClientStore, read *ClientRead) (auth.ChatGPTPlanOptions, Result, error) {
	t.Helper()
	var given auth.ChatGPTPlanOptions
	run, err := Start(background(), Spec{Provider: llmprovider.ProviderOpenAI, Method: planNoConsent(), Mode: ModePlan,
		First: true, ClientStore: store, Client: read, NoPaste: true,
		Logins: Logins{Plan: func(_ context.Context, o auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
			given = o
			return &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "access-0123456789",
				ClientID: cmp.Or(o.ClientID, "client-issued"), HostID: cmp.Or(o.HostID, "urn:uuid:minted")}, nil
		}}})
	if err != nil {
		t.Fatalf("Start = %v", err)
	}
	result, err := run.Wait(bounded(t))
	return given, result, err
}

// R-12: a decode failure is none, as R-9 ruled; the login runs, and the new
// client replaces the record. Red on the first read: the helper does not know
// the error.
func TestPlanUndecodableRecordFromTheStoreIsNone_0082(t *testing.T) {
	loadErr := decodeFailure0082()
	store := &recordStore{loadErr: loadErr, record: auth.ClientRecord{ClientID: "stale-client", HostID: "urn:uuid:stale"}}
	given, result, err := signInPlan(t, store, auth.ChatGPTPlanOptions{})
	if err != nil || result.Session == nil || given.OpenURL == nil || given.ClientID != "" || given.HostID != "" {
		t.Fatalf("err %v, session %v, login ran %t with client %q: want the sign-in to go on with none of the record's", err, result.Session, given.OpenURL != nil, given.ClientID)
	}
	if !errors.Is(result.ClientLoadErr, loadErr) || store.saves != 1 || store.record.ClientID != "client-issued" {
		t.Errorf("ClientLoadErr %v, record %+v after %d save(s); want the error reported and the new client saved once", result.ClientLoadErr, store.record, store.saves)
	}
}

// R-12: any other read error fails the sign-in before the login runs, naming
// the error as the store gave it, and the record is untouched. Red while every
// error is none.
func TestPlanReadErrorFailsTheSignIn_0082(t *testing.T) {
	readErr := &fs.PathError{Op: "open", Path: "/x/openai.client", Err: fs.ErrPermission}
	good := auth.ClientRecord{ClientID: "client-mine", HostID: "urn:uuid:mine"}
	store := &recordStore{loadErr: readErr, record: good}
	given, result, err := signInPlan(t, store, auth.ChatGPTPlanOptions{})
	if !errors.Is(err, readErr) || given.OpenURL != nil || result.Session != nil {
		t.Errorf("err %v, login ran %t, session %v; want the read error and no login", err, given.OpenURL != nil, result.Session)
	}
	if store.record != good || store.saves != 0 {
		t.Errorf("the record = %+v after %d save(s); want it untouched", store.record, store.saves)
	}
}

// R-12 (c): given the caller's own read, the engine reads nothing: with a
// record, the login is given its client; with a decode failure, none, and the
// failure is Result.ClientLoadErr. Red while the engine reads again.
func TestPlanGivenTheCallersReadReadsNothing_0082(t *testing.T) {
	t.Run("a record", func(t *testing.T) {
		store := &recordStore{record: auth.ClientRecord{ClientID: "client-other", HostID: "urn:uuid:other"}}
		given, _, err := signInPlanRead(t, store, &ClientRead{Record: auth.ClientRecord{ClientID: "client-mine", HostID: "urn:uuid:mine"}})
		if err != nil || given.ClientID != "client-mine" || given.HostID != "urn:uuid:mine" || store.loads != 0 {
			t.Errorf("err %v, login given client %q host %q after %d load(s); want the caller's record and no read", err, given.ClientID, given.HostID, store.loads)
		}
	})
	t.Run("a decode failure", func(t *testing.T) {
		store := &recordStore{record: auth.ClientRecord{ClientID: "client-other"}}
		failure := decodeFailure0082()
		given, result, err := signInPlanRead(t, store, &ClientRead{Err: failure})
		if err != nil || given.ClientID != "" || store.loads != 0 || !errors.Is(result.ClientLoadErr, failure) {
			t.Errorf("err %v, client %q, %d load(s), ClientLoadErr %v; want none of the record, no read, the failure reported", err, given.ClientID, store.loads, result.ClientLoadErr)
		}
		if store.saves != 1 || store.record.ClientID != "client-issued" {
			t.Errorf("the record = %+v after %d save(s); want the new client saved over it", store.record, store.saves)
		}
	})
}
