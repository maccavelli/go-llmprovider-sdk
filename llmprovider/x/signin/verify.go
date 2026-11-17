package signin

import (
	"context"
	"fmt"

	"github.com/maccavelli/go-llmprovider-sdk/internal/signinflow"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// verifyLimit bounds VerifyKilo's read of the profile: the wizard's default
// discovery limit, one constant (0080-MADR Q5, D10). A variable only so that a
// test can shorten it.
var verifyLimit = signinflow.DefaultCheckLimit

// VerifyKilo checks a Kilo key, or a Kilo device token, against Kilo's
// profile. Kilo's public model listing answers every credential with 200, so
// a caller that lists models with a key and no organization cannot tell a
// wrong key from a right one; the profile can (0080-MADR D1). It is the check
// the wizard makes under Options.Discover for a credential used with no
// organization, for a caller that drives its own sign-in, such as one that
// checks a typed key before it saves it. The sign-in saves nothing and so
// does this: the caller decides what to do with the answer.
//
// It reads the profile once, at the origin auth.KiloProfile reads it: the
// origin of the base URL in opts, or of a token that embeds one. opts are
// kilo's, as auth.KiloProfile takes them: llmprovider.WithBaseURL,
// llmprovider.WithHTTPClient and llmprovider.WithClientInfo, and
// catalog.WithKiloOrganization, which gives the organization as the model
// listing reads it. The read is bound by 10 seconds, the wizard's default discovery limit, and by ctx; a
// profile that has not answered in 10 seconds is one that cannot say.
//
// The result is one of:
//
//   - nil, which means that the key was verified or that the profile could
//     not say, and nothing more: the profile accepted the key; or the
//     credential has an organization, whose listing authenticates and is the
//     check there, so no request is made. That is decided as the model listing
//     decides it, from the options it is given: the last
//     catalog.WithKiloOrganization applied, an empty one included, where
//     llmprovider.ResolveOptions applies the options outside a For first, in
//     order, then those inside llmprovider.For(llmprovider.ProviderKilo, ...),
//     in order; else the organization a token's embedded URL names; else the
//     one the base URL's own path names. A non-empty organization argument is
//     one more such option, where the wizard puts the Result's organization:
//     as catalog.WithKiloOrganization(organization) inside a For for Kilo,
//     before every For in opts and after every option of opts outside a For,
//     so a caller who builds a provider or lists with opts and that same For
//     put first among its Fors gets the listing this decision assumed; an
//     empty one adds nothing, and an organization option in opts stands. Or
//     the profile could not say, which is a 403 (every Kilo 403 is
//     llmprovider.ErrNotPermitted, not a refusal), a 5xx, an unexpected
//     status, a body that does not decode, a machine that is offline, or no
//     answer in 10 seconds. A caller that must tell an answer from a profile
//     that could not say reads the profile itself with auth.KiloProfile;
//   - an error matching llmprovider.ErrAuthFailure and not
//     llmprovider.ErrNotPermitted, the profile's own *llmprovider.APIError
//     with status 401, when Kilo refuses the key; ErrorKindOf gives "auth".
//     The SDK removes from an error's text a key of the shapes its redaction
//     knows, as it does from every listing's refusal, by pattern and not by
//     the key's value: an opaque key a service echoes in its answer may remain
//     in the text, so a caller shows the error's class and status, never its
//     text (0056-MADR D3);
//   - an error matching ctx's error, wrapped, when ctx ended, a deadline of
//     the caller's included, before the profile answered; the 10 seconds
//     above do not end ctx and never give this error;
//   - an error matching llmprovider.ErrInvalidRequest, and no request, when
//     opts hold a mistake that makes the profile unreadable whatever the
//     key: an option for another provider, or a base URL that does not parse.
//
// This function is experimental, like the package (R4 of the API standards):
// it is outside the v1 promise and may change or go in any release.
func VerifyKilo(ctx context.Context, key, organization string, opts ...llmprovider.Option) error {
	if err := signinflow.CheckKiloOptions(opts...); err != nil {
		return fmt.Errorf("signin: verify the Kilo key: %w", err)
	}
	// The bound is this function's own: its end is "cannot say", and only ctx's
	// end, the caller's, is returned.
	bounded, cancel := context.WithTimeout(ctx, verifyLimit)
	defer cancel()
	refused, unverified := signinflow.VerifyKilo(bounded, auth.KiloProfile, key, organization, opts...)
	if refused != nil {
		return refused
	}
	if unverified != nil {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("signin: verify the Kilo key: %w", err)
		}
	}
	return nil
}
