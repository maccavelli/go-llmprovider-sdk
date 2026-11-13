package auth

import (
	"net/http"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// Identity names the consuming application to the identity providers on a
// sign-in, as llmprovider.WithClientInfo names it on a provider's requests
// (0076-MADR D3). Name is OpenAI's originator on the authorize URL and Grok's
// referrer on the authorize URL and the device request, and it leads the
// User-Agent of every request of the sign-in, which names this module after
// it. Its zero value names this module, as every sign-in did before Identity.
// The session a sign-in returns keeps the Identity in memory, and its refresh
// and its revocation, with the discovery a revocation reads first, send it as
// the sign-in did; a session loaded from a store has none, and
// (*OAuthSession).UseIdentity gives it one, when it has none. Nothing stores
// it (0079-MADR D2, D3). Whether OpenAI and xAI accept every name is not
// verified (0076-MADR, "Not verified").
//
// Identity is a comparable value. It is safe for concurrent use while no
// goroutine changes it.
type Identity struct {
	// Name is the application's name. Empty names this module.
	Name string
	// Version is the application's version, read only with Name. Empty is
	// the main module's version, as llmprovider.WithClientInfo's is.
	Version string
}

// clientName is the name OpenAI's originator and Grok's referrer carry:
// Name, or this module's.
func (id Identity) clientName() string {
	if id.Name == "" {
		return originatorValue
	}
	return id.Name
}

// setUserAgent names id's application on req, or this module when Name is
// empty, Version then unread.
func (id Identity) setUserAgent(req *http.Request) {
	if id.Name == "" {
		transport.NewIdentity("", "", "").SetUserAgent(req)
		return
	}
	transport.NewIdentity(id.Name, id.Version, "").SetUserAgent(req)
}
