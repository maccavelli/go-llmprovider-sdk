package wizard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// plantedToken is in every secret below; no Result field, and no formatted
// form of a Result, may show it.
const plantedToken = "PLANTED-TOKEN-5c1e"

// resultFieldsHolding returns the names of res's fields whose text contains
// secret, walking every string and []string field.
func resultFieldsHolding(res Result, secret string) []string {
	var found []string
	v := reflect.ValueOf(res)
	for i := range v.NumField() {
		f := v.Field(i)
		switch f.Kind() {
		case reflect.String:
			if strings.Contains(f.String(), secret) {
				found = append(found, v.Type().Field(i).Name)
			}
		case reflect.Slice:
			for j := range f.Len() {
				if s, ok := f.Index(j).Interface().(string); ok && strings.Contains(s, secret) {
					found = append(found, v.Type().Field(i).Name)
				}
			}
		default:
		}
	}
	return found
}

// TestConfigureLLM_StoredSessionLeavesNoTokenInResult (0016-MADR D11, A7): a
// session saved to the TokenStore is its only copy. Result keeps the
// non-secret fields.
func TestConfigureLLM_StoredSessionLeavesNoTokenInResult(t *testing.T) {
	expiry := time.Now().Add(time.Hour).Truncate(time.Second)
	stubBrowserLogin(t, func(_ context.Context, provider llmprovider.ProviderID, _ auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		s := testOAuthSession(provider)
		s.Access, s.Refresh, s.Expiry, s.AccountID = "at-"+plantedToken, "rt-"+plantedToken, expiry, "acct-visible"
		return s, nil
	})
	store := newMemoryTokenStore()
	f := newFake(t, fakePrompter{blankSearches: true, selects: []int{providerIdx(t, llmprovider.ProviderGrok), 1, 0}})
	res, err := ConfigureLLM(context.Background(), f, Options{TokenStore: store})
	if err != nil {
		t.Fatalf("ConfigureLLM() error = %v", err)
	}
	if fields := resultFieldsHolding(res, plantedToken); len(fields) != 0 {
		t.Errorf("Result fields %v hold a token; the store is the only copy", fields)
	}
	saved := store.sessions[llmprovider.ProviderGrok]
	if saved == nil || saved.Access != "at-"+plantedToken || saved.Refresh != "rt-"+plantedToken {
		t.Errorf("stored session %v, want both tokens", saved)
	}
	if res.Kind != CredOAuth || res.Issuer != auth.DefaultGrokOAuthIssuer || res.AccountID != "acct-visible" ||
		!res.TokenExpiry.Equal(expiry) {
		t.Errorf("Result %v, want kind oauth with the issuer, account and expiry", res)
	}
}

// TestConfigureLLM_KeepsTheStoredSession (0016-MADR A10): keeping a session
// reads it from the store; Existing carries no token.
func TestConfigureLLM_KeepsTheStoredSession(t *testing.T) {
	stubBrowserLogin(t, func(context.Context, llmprovider.ProviderID, auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		t.Error("signed in again; want the stored session kept")
		return nil, fmt.Errorf("no sign-in expected")
	})
	store := newMemoryTokenStore()
	stored := testOAuthSession(llmprovider.ProviderGrok)
	stored.Access = "stored-access-wxyz"
	if err := store.Save(context.Background(), llmprovider.ProviderGrok, stored); err != nil {
		t.Fatal(err)
	}
	f := newFake(t, fakePrompter{blankSearches: true, selects: []int{providerIdx(t, llmprovider.ProviderGrok), 0}, confirms: []bool{true}})
	res, err := ConfigureLLM(context.Background(), f, Options{
		Existing:   Result{Provider: llmprovider.ProviderGrok, Kind: CredOAuth},
		TokenStore: store,
	})
	if err != nil {
		t.Fatalf("ConfigureLLM() error = %v", err)
	}
	if len(f.seenConfirm) != 1 || !strings.Contains(f.seenConfirm[0], "Keep the existing session") {
		t.Fatalf("confirms %q, want the keep prompt", f.seenConfirm)
	}
	if res.Kind != CredOAuth || store.saves != 1 {
		t.Errorf("kind %q, saves %d; want oauth and no new save", res.Kind, store.saves)
	}
	assertTextMasksSecret(t, f.allText, "stored-access-wxyz")
}

// TestConfigureLLM_NoStoredSessionSignsIn: with nothing in the store, an
// Existing oauth Result does not offer to keep anything.
func TestConfigureLLM_NoStoredSessionSignsIn(t *testing.T) {
	signedIn := false
	stubBrowserLogin(t, func(_ context.Context, provider llmprovider.ProviderID, _ auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		signedIn = true
		return testOAuthSession(provider), nil
	})
	f := newFake(t, fakePrompter{blankSearches: true, selects: []int{providerIdx(t, llmprovider.ProviderGrok), 1, 0}})
	if _, err := ConfigureLLM(context.Background(), f, Options{
		Existing:   Result{Provider: llmprovider.ProviderGrok, Kind: CredOAuth},
		TokenStore: newMemoryTokenStore(),
	}); err != nil {
		t.Fatalf("ConfigureLLM() error = %v", err)
	}
	if !signedIn || len(f.seenConfirm) != 0 {
		t.Errorf("signed in %v, confirms %q; want a sign-in and no keep prompt", signedIn, f.seenConfirm)
	}
}

// TestResult_Redacts (0016-MADR D5, A9): fmt and slog never show the API key;
// json.Marshal keeps it, because the consumer persists Result.
func TestResult_Redacts(t *testing.T) {
	res := Result{Provider: llmprovider.ProviderClaude, Kind: CredAPIKey, APIKey: "key-" + plantedToken,
		Model: "model-visible", Fallbacks: []string{"fallback-visible"}}
	for _, tc := range []struct {
		name  string
		value any
	}{
		{"Result", res},
		{"*Result", &res},
		{"struct holding a Result", struct{ Res Result }{res}},
	} {
		forms := map[string]string{}
		for _, verb := range []string{"%v", "%+v", "%#v", "%s"} {
			forms[verb] = fmt.Sprintf(verb, tc.value)
		}
		var jsonOut, textOut bytes.Buffer
		slog.New(slog.NewJSONHandler(&jsonOut, nil)).Info("m", "v", tc.value)
		slog.New(slog.NewTextHandler(&textOut, nil)).Info("m", "v", tc.value)
		forms["slog JSON"], forms["slog text"] = jsonOut.String(), textOut.String()
		for form, out := range forms {
			if tc.name == "struct holding a Result" && form == "slog JSON" {
				// slog's JSON handler encodes a struct with encoding/json,
				// which keeps the key by design (0016-MADR A9).
				continue
			}
			if strings.Contains(out, plantedToken) {
				t.Errorf("%s via %s shows the key: %s", tc.name, form, out)
			}
			for _, shown := range []string{"model-visible", "fallback-visible", string(llmprovider.ProviderClaude)} {
				if !strings.Contains(out, shown) {
					t.Errorf("%s via %s lost the non-secret %q: %s", tc.name, form, shown, out)
				}
			}
		}
	}
	raw, err := json.Marshal(res)
	if err != nil || !strings.Contains(string(raw), "key-"+plantedToken) {
		t.Errorf("json.Marshal = %s, %v; want the key kept for the consumer to persist", raw, err)
	}
}

// storedExisting saves existing's session, with its tokens, to store, and
// returns existing: what a consumer holds after a stored sign-in. The tokens
// are only in the store (0016-MADR A10).
func storedExisting(t *testing.T, store *memoryTokenStore, existing Result, access, refresh string) Result {
	t.Helper()
	s := &auth.OAuthSession{Provider: existing.Provider, Access: access, Refresh: refresh,
		Expiry: existing.TokenExpiry, Issuer: existing.Issuer, ClientID: existing.ClientID,
		AccountID: existing.AccountID, FedRAMP: existing.FedRAMP}
	if err := store.Save(context.Background(), existing.Provider, s); err != nil {
		t.Fatal(err)
	}
	return existing
}
