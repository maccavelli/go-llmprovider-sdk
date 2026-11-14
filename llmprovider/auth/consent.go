package auth

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// The two answers a Consents entry holds.
const (
	consentGranted  = "granted"
	consentDeclined = "declined"
)

// Consent reports the session's own answer to the consent id: granted, and
// whether the user answered at all. An id with no entry, or an entry this
// version does not understand, is unanswered. It reads the session as it is
// in memory, which a SetConsent made through another handle on the store does
// not change until a refresh adopts it; the openai provider reads the store's
// answer through StoredConsent. Consent is safe for concurrent use.
func (s *OAuthSession) Consent(id string) (granted, answered bool) {
	if s == nil {
		return false, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return consentOf(s.Consents, id)
}

// consentOf reads id's answer from a Consents string.
func consentOf(consents, id string) (granted, answered bool) {
	for entry := range strings.FieldsSeq(consents) {
		name, value, ok := strings.Cut(entry, "=")
		if !ok || name != id {
			continue
		}
		switch value {
		case consentGranted:
			return true, true
		case consentDeclined:
			return false, true
		}
	}
	return false, false
}

// withConsent returns consents with id's answer set, entries sorted by id.
// An entry this version does not understand is dropped.
func withConsent(consents, id string, granted bool) string {
	answers := map[string]string{}
	for entry := range strings.FieldsSeq(consents) {
		name, value, ok := strings.Cut(entry, "=")
		if ok && name != "" && (value == consentGranted || value == consentDeclined) {
			answers[name] = value
		}
	}
	answers[id] = consentDeclined
	if granted {
		answers[id] = consentGranted
	}
	names := make([]string, 0, len(answers))
	for name := range answers {
		names = append(names, name)
	}
	slices.Sort(names)
	entries := make([]string, len(names))
	for i, name := range names {
		entries[i] = name + "=" + answers[name]
	}
	return strings.Join(entries, " ")
}

// SetConsent changes the stored session's answer to the consent id: granted
// or declined. It loads the provider's session from store, sets the entry
// and saves it, and when store is a RefreshLocker it holds the provider's
// refresh lock throughout, so it never races a refresh; a store that is not
// one is loaded and saved unlocked, and a refresh in another process can then
// still race the save. A session that refreshes afterwards keeps the answer
// (0075-MADR D3).
//
// SetConsent returns an error matching llmprovider.ErrInvalidRequest for an
// id that is empty or holds white space or "=", and for a provider with no
// stored session. The store's own errors come back as they are:
// llmprovider.ErrInvalidProvider for a bad provider id and a plain error
// otherwise, and a lock not taken within the store's wait or a ctx that ends
// first as the store's LockRefresh reports them. SetConsent is safe for
// concurrent use when store is.
func SetConsent(ctx context.Context, store TokenStore, provider llmprovider.ProviderID, id string, granted bool) error {
	if id == "" || strings.ContainsAny(id, "= \t\r\n") {
		return fmt.Errorf("%w: auth: SetConsent: %q is not a consent id", llmprovider.ErrInvalidRequest, id)
	}
	if store == nil {
		return fmt.Errorf("%w: auth: SetConsent: no store", llmprovider.ErrInvalidRequest)
	}
	if locker, ok := store.(RefreshLocker); ok {
		unlock, err := locker.LockRefresh(ctx, provider)
		if err != nil {
			return err
		}
		defer unlock()
	}
	session, err := store.Load(ctx, provider)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("%w: auth: SetConsent: no stored %s session", llmprovider.ErrInvalidRequest, provider)
	}
	session.Consents = withConsent(session.Consents, id, granted)
	return store.Save(ctx, provider, session)
}
