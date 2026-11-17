package signinflow

import (
	"context"
	"errors"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/kiloendpoint"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// DefaultCheckLimit is how long a check of a credential waits for the
// service by default: the wizard's Options.DiscoverLimit when it is zero, and
// signin.VerifyKilo's own bound. It equals the model lister's own bound
// (0013-MADR Q4); one constant, so the wizard and a caller of VerifyKilo wait
// the same (0080-MADR Q5, D6, D10).
const DefaultCheckLimit = 10 * time.Second

// KiloProfileReader reads the account behind a Kilo token: auth.KiloProfile,
// or the wizard's seam over it.
type KiloProfileReader func(ctx context.Context, token string, opts ...llmprovider.Option) (auth.KiloAccount, error)

// Refused reports whether err is a service refusing a credential: an
// authentication failure that is not a refusal to permit something
// (0028-MADR D-H2). It is the one rule by which the wizard's listing check,
// its Kilo profile check and VerifyKilo tell a refused credential from a
// failure that says nothing of it (0080-MADR D1, D5, D9).
func Refused(err error) bool {
	return errors.Is(err, llmprovider.ErrAuthFailure) && !errors.Is(err, llmprovider.ErrNotPermitted)
}

// KiloVerdict sorts the failure of a profile read: a refusal, which is err
// itself, or an unverified result, which is err as the cause. A nil err gives
// neither (0080-MADR D1, D5, D6).
func KiloVerdict(err error) (refused, unverified error) {
	switch {
	case err == nil:
		return nil, nil
	case Refused(err):
		return err, nil
	}
	return nil, err
}

// kiloInputs are the base URL and the explicit organization the listing is
// given by opts, which are the options the listing is given, resolved as it
// resolves them: the options outside a For first, in order, then each For's,
// in order, the last catalog.WithKiloOrganization winning. A non-empty
// organization is one more such option, put where the wizard puts the Result's
// organization: inside a For for Kilo, before every For of opts, so after
// every option outside a For and before every For the caller passed. An empty
// one adds nothing, so that an organization option in opts is not undone by an
// argument that gives none. An option set that cannot be resolved gives no base
// URL and the argument as it is: CheckKiloOptions reports it.
func kiloInputs(organization string, opts []llmprovider.Option) (base, explicit string) {
	all := opts
	if organization != "" {
		all = append([]llmprovider.Option{llmprovider.For(llmprovider.ProviderKilo,
			llmprovider.ScopedOption(llmprovider.ProviderKilo, "catalog.WithKiloOrganization",
				kiloendpoint.OrganizationOption(organization)))}, opts...)
	}
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderKilo, all)
	if err != nil {
		return "", organization
	}
	return st.BaseURL(), kiloendpoint.OptionOrganization(st.Values())
}

// CheckKiloOptions reports a mistake in the options a Kilo check is given, one
// that would make the profile unreadable whatever the key: an option for
// another provider, or a base URL that does not parse. It matches
// llmprovider.ErrInvalidRequest. It sends nothing. Without it, a caller's
// mistake would read as the profile being unable to say (0080-MADR D10).
func CheckKiloOptions(opts ...llmprovider.Option) error {
	_, err := llmprovider.ResolveOptions(llmprovider.ProviderKilo, opts)
	return err
}

// VerifyKilo checks a Kilo credential against Kilo's profile, the check
// Kilo's public listing cannot make since it answers every credential
// (0080-MADR D1). It reads the profile once, with read, under ctx, and
// returns one of:
//
//   - neither, when the profile answers, and when the credential has an
//     organization: the explicit one (the last catalog.WithKiloOrganization
//     applied, where the options outside a For apply first, then each For's,
//     and a non-empty organization is a For option before those of opts), else
//     the one a token's embedded URL names, else the one the base URL's own
//     path names (kiloendpoint.Organization, the rule the listing's URL is
//     built by). An organization's listing authenticates, so no request is
//     made (D4);
//   - refused, the profile's own *llmprovider.APIError, when Kilo refuses the
//     token (401, ErrAuthFailure and not ErrNotPermitted);
//   - unverified, the cause, when the profile cannot say: a 403 (every Kilo
//     403 is ErrNotPermitted), a 5xx, an unexpected status, a body that does
//     not decode, a machine offline, or ctx ending, which is checked before
//     the request, so none is sent after it.
//
// It sets no bound of its own: the caller bounds the read by ctx, as the
// wizard does by Options.DiscoverLimit and signin.VerifyKilo by
// DefaultCheckLimit. opts are kilo's, as auth.KiloProfile takes them, so the
// profile is read at the origin they name, or the origin a token embeds. The
// error it returns shows no token by value: the SDK's redaction removes the
// shapes it knows, as it does from every listing's refusal (0056-MADR D3;
// 0080-MADR D9).
func VerifyKilo(
	ctx context.Context,
	read KiloProfileReader,
	token, organization string,
	opts ...llmprovider.Option,
) (refused, unverified error) {
	if base, explicit := kiloInputs(organization, opts); kiloendpoint.Organization(base, token, explicit) != "" {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_, err := read(ctx, token, opts...)
	return KiloVerdict(err)
}
