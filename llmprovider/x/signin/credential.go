package signin

import (
	"log/slog"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// Credential is what a sign-in yields: an APIKey, Session, AccessOnly,
// DeviceKey or ReadThrough. The interface is closed: only this package's
// types implement it, so a type can be added after promotion without
// breaking an implementer. A Credential saves nothing; the caller stores
// what Yields says needs storing. Each of the five types is safe for
// concurrent use while no goroutine changes it.
type Credential interface {
	// Yields is the contract's name for what this credential is.
	Yields() authcontract.Yields
	// TokenSource returns the source that signs requests with it.
	TokenSource() llmprovider.TokenSource
	credential()
}

// redactedKey is what an APIKey or DeviceKey prints in place of its value.
const redactedKey = "[redacted]"

// APIKey is an API key. Its String, GoString and LogValue never show the
// key. It is safe for concurrent use while no goroutine changes it.
type APIKey struct {
	// Key is the key itself.
	Key string
}

// Yields reports authcontract.YieldsAPIKey.
func (APIKey) Yields() authcontract.Yields { return authcontract.YieldsAPIKey }

// TokenSource returns a StaticToken holding the key.
func (k APIKey) TokenSource() llmprovider.TokenSource { return llmprovider.NewStaticToken(k.Key) }

func (APIKey) credential() {}

// String shows no key.
func (APIKey) String() string { return "signin.APIKey(" + redactedKey + ")" }

// GoString shows no key.
func (k APIKey) GoString() string { return k.String() }

// LogValue shows no key.
func (APIKey) LogValue() slog.Value { return slog.StringValue(redactedKey) }

// Session is an OAuth session that refreshes itself and a logout revokes. The
// caller saves it, through a TokenStore. It is safe for concurrent use while
// no goroutine changes it.
type Session struct {
	// OAuth is the session; it renders without its tokens.
	OAuth *auth.OAuthSession
}

// Yields reports authcontract.YieldsOAuthSession.
func (Session) Yields() authcontract.Yields { return authcontract.YieldsOAuthSession }

// TokenSource returns the session, which refreshes itself.
func (s Session) TokenSource() llmprovider.TokenSource { return s.OAuth }

func (Session) credential() {}

// AccessOnly is an access token the sign-in cannot refresh, held in a session
// that a store keeps and a logout revokes. It is safe for concurrent use
// while no goroutine changes it.
type AccessOnly struct {
	// OAuth is the session holding the access token.
	OAuth *auth.OAuthSession
}

// Yields reports authcontract.YieldsAccessOnly.
func (AccessOnly) Yields() authcontract.Yields { return authcontract.YieldsAccessOnly }

// TokenSource returns the session.
func (a AccessOnly) TokenSource() llmprovider.TokenSource { return a.OAuth }

func (AccessOnly) credential() {}

// DeviceKey is the key a device sign-in issues, used as an API key and
// refreshed never, as Kilo's is. Its String, GoString and LogValue never show
// the key. It is safe for concurrent use while no goroutine changes it.
type DeviceKey struct {
	// Key is the key itself.
	Key string
}

// Yields reports authcontract.YieldsDeviceKey.
func (DeviceKey) Yields() authcontract.Yields { return authcontract.YieldsDeviceKey }

// TokenSource returns a StaticToken holding the key.
func (k DeviceKey) TokenSource() llmprovider.TokenSource { return llmprovider.NewStaticToken(k.Key) }

func (DeviceKey) credential() {}

// String shows no key.
func (DeviceKey) String() string { return "signin.DeviceKey(" + redactedKey + ")" }

// GoString shows no key.
func (k DeviceKey) GoString() string { return k.String() }

// LogValue shows no key.
func (DeviceKey) LogValue() slog.Value { return slog.StringValue(redactedKey) }

// ReadThrough is a vendor CLI's login, read through on every call: nothing
// is copied, so nothing is stored but the path. It is safe for concurrent use
// while no goroutine changes it.
type ReadThrough struct {
	// Source reads the CLI's auth file; Source.Path is what a caller keeps.
	Source *auth.VendorCLISession
}

// Yields reports authcontract.YieldsReadThrough.
func (ReadThrough) Yields() authcontract.Yields { return authcontract.YieldsReadThrough }

// TokenSource returns the session that reads the CLI's file.
func (r ReadThrough) TokenSource() llmprovider.TokenSource { return r.Source }

func (ReadThrough) credential() {}
