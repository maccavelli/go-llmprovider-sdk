package wizard

import (
	"context"
	"errors"
	"fmt"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Logout signs the user out of the session Options.TokenStore holds for id
// (0016-MADR D11, A8). After the user confirms, it revokes the session with
// the service, then deletes it from the store. A failed revocation is reported
// through p, and the local copy is deleted anyway. A session of a provider
// with no revocation (only OpenAI and Grok offer one) is deleted, and p is
// told so. With no stored session, p is told and nothing changes.
//
// It returns an error only when there is no store, or reading or deleting the
// session fails.
func Logout(ctx context.Context, p Prompter, o Options, id llmprovider.ProviderID) error {
	if o.TokenStore == nil {
		return errors.New("wizard: logout needs Options.TokenStore")
	}
	label := providerLabel(o, id)
	session, err := o.TokenStore.Load(ctx, id)
	if err != nil {
		return fmt.Errorf("wizard: load the saved %s session: %w", label, err)
	}
	if session == nil {
		p.Notify(LevelInfo, "No saved %s session", label)
		return nil
	}
	sure, err := p.Confirm(fmt.Sprintf("Log out of %s (%s)?", label, redact.MaskSecret(session.Access)), true)
	if err != nil {
		return err
	}
	if !sure {
		return nil
	}
	switch id {
	case llmprovider.ProviderOpenAI, llmprovider.ProviderGrok:
		if session.HTTPClient == nil {
			session.HTTPClient = o.HTTPClient
		}
		if err := llmprovider.RevokeOAuthSession(ctx, session); err != nil {
			p.Notify(LevelWarn, "could not revoke the %s session (%v); deleting the local copy", label, err)
		}
	default:
		p.Notify(LevelInfo, "%s offers no revocation; deleting the local copy", label)
	}
	if err := o.TokenStore.Delete(ctx, id); err != nil {
		return fmt.Errorf("wizard: delete the saved %s session: %w", label, err)
	}
	p.Notify(LevelInfo, "Logged out of %s", label)
	return nil
}

// providerLabel is id's menu label in the run's registry, or id itself.
func providerLabel(o Options, id llmprovider.ProviderID) string {
	if d, ok := o.registry().Descriptor(id); ok {
		return d.Label
	}
	return string(id)
}
