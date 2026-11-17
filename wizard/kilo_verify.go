package wizard

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/signinflow"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// discoverLimit is the bound of one check request: Options.DiscoverLimit, else
// defaultDiscoverLimit.
func (o Options) discoverLimit() time.Duration {
	if o.DiscoverLimit > 0 {
		return o.DiscoverLimit
	}
	return defaultDiscoverLimit
}

// kiloProfileOptions are the options of the one read of Kilo's profile that
// resolveKiloDevice makes itself, for a device session's organization menu:
// Options.Identity's, the endpoint in use, the caller's client and
// ProviderOptions last (0020-MADR F20; 0079-MADR D5; 0080-MADR D3, D7). The
// sign-in check is given listingOptions instead (0080-MADR D4).
func (o Options) kiloProfileOptions(baseURL string) []llmprovider.Option {
	opts := o.identityOptions()
	if baseURL != "" {
		opts = append(opts, llmprovider.WithBaseURL(baseURL))
	}
	if o.HTTPClient != nil {
		opts = append(opts, llmprovider.WithHTTPClient(o.HTTPClient))
	}
	return append(opts, o.ProviderOptions...)
}

// checkEnded is the failure of a check that ctx ended before it could say
// anything: neither a refusal nor a warning, and nothing may be saved after it
// (0080-MADR D6). credentialStep returns it as the context's error.
type checkEnded struct{ err error }

func (e *checkEnded) Error() string { return e.err.Error() }
func (e *checkEnded) Unwrap() error { return e.err }

// checkCredential is the credential step's check of one credential: Kilo's
// profile, for a Kilo credential used with no organization, then the model
// listing. A profile that refuses the credential is returned as the listing's
// refusal is, and the listing is not made (0080-MADR D3).
func checkCredential(
	ctx context.Context,
	p Prompter,
	d llmprovider.Descriptor,
	res Result,
	c resolvedCredential,
	o Options,
) (catalog.Catalog, error) {
	if err := verifyKilo(ctx, p, d, res, c, o); err != nil {
		return catalog.Catalog{}, err
	}
	return discoverModels(ctx, p, d, res, c.source, o)
}

// verifyKilo checks a Kilo credential against Kilo's profile, under
// Options.Discover and for a credential used with no organization; an
// organization's listing authenticates, and is the check there, which
// signinflow.VerifyKilo decides from the explicit organization, a token's
// embedded URL and the base URL (0080-MADR D1, D4). It reads the profile once,
// within Options.DiscoverLimit, with the credential's token: a key's, or a
// session's access token (a typed, environment or saved key, a device key, a
// device session or a kept session). A device session's read was made by
// resolveKiloDevice, for its organization menu, and is not made again (D3); its
// failure that is no refusal is told here, once, in D6's words, after the
// menu's own warning, which resolveKiloDevice has already told.
//
// A 401 is a refusal, returned for credentialStep's loop as the listing's is.
// Any other failure, a 403 included, is a warning by class and status, never
// the service's text, and the sign-in goes on (D5, D6). A ctx that has ended is
// neither: the check returns a *checkEnded.
func verifyKilo(
	ctx context.Context,
	p Prompter,
	d llmprovider.Descriptor,
	res Result,
	c resolvedCredential,
	o Options,
) error {
	if !o.Discover || d.ID != llmprovider.ProviderKilo {
		return nil
	}
	token := kiloToken(c)
	if token == "" {
		// A Prompter of the caller's own that returns an empty or all-space
		// key gets it trimmed to empty by promptAPIKey and returned as a key,
		// for every provider, with no check: found, and not changed here
		// (0080-PLAN, "Found, not changed"). TextPrompter never returns one.
		return nil
	}
	read := kiloProfile
	if c.profileRead {
		// resolveKiloDevice read the profile, for its organization menu: that
		// read is this check, and the rule below sorts its result as it
		// sorts a read made here, with no second request (D3).
		read = func(context.Context, string, ...llmprovider.Option) (auth.KiloAccount, error) {
			return auth.KiloAccount{}, c.profileErr
		}
	}
	vctx, cancel := context.WithTimeout(ctx, o.discoverLimit())
	defer cancel()
	// Whether the credential has an organization, and so is checked by its
	// listing and not here, is decided once, by signinflow.VerifyKilo (D4),
	// from the options the listing itself is given, built by one function
	// (listingOptions), the Result's organization among them in its own place;
	// so it cannot disagree with the listing.
	refused, unverified := signinflow.VerifyKilo(vctx, read, token, "", o.listingOptions(res)...)
	if refused != nil {
		return refused
	}
	if unverified == nil {
		return nil
	}
	// A deadline of the check's own is an outage; ctx's end is not.
	if err := ctx.Err(); err != nil {
		return &checkEnded{err: fmt.Errorf("wizard: %w", err)}
	}
	p.Notify(LevelWarn, "could not verify the Kilo key (%s); it is saved unchecked", describe(unverified))
	return nil
}

// kiloToken is the token a Kilo credential sends: an API key, or a session's
// access token. A credential with neither has none to verify.
func kiloToken(c resolvedCredential) string {
	if c.session != nil {
		return c.session.Access
	}
	return c.apiKey
}

// endedCheck is the context's error when a check ended with ctx, else nil.
func endedCheck(err error) error {
	if ended, ok := errors.AsType[*checkEnded](err); ok {
		return ended
	}
	return nil
}
