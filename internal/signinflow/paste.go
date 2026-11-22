package signinflow

import (
	"net/url"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
)

// pasteCheck holds what a pasted callback must match: the state of the
// authorize URL the open_url step showed, and what the paste step accepts
// (0074-MADR D4).
type pasteCheck struct {
	state string
	// callback is the host and port of the authorize URL's redirect_uri, "" when
	// it names none: a paste that begins with it, with no scheme, is read as the
	// address it is (0083-MADR D11).
	callback string
	accepts  authcontract.Paste
}

// newPasteCheck reads the state, and the callback's host and port, from
// authorizeURL.
func newPasteCheck(authorizeURL string, accepts authcontract.Paste) pasteCheck {
	state, callback := "", ""
	if u, err := url.Parse(authorizeURL); err == nil {
		query := u.Query()
		state = query.Get("state")
		if redirect, redirectErr := url.Parse(query.Get("redirect_uri")); redirectErr == nil {
			callback = redirect.Host
		}
	}
	return pasteCheck{state: state, callback: callback, accepts: accepts}
}

// promptFor is the paste step's first text.
func promptFor(accepts authcontract.Paste) string {
	switch accepts {
	case authcontract.PasteURLOnly:
		return "Paste the whole URL the browser was sent to after you signed in."
	case authcontract.PasteCode:
		return "Paste the code the sign-in page shows."
	default:
		return "Paste the URL the browser was sent to after you signed in, or the code the page shows."
	}
}

// check returns why a pasted value is not usable, or "" when it is, or is a
// pasted denial. The reason never holds the paste. A URL is read as auth
// reads it: anything with a scheme, and a paste that begins with the
// callback's host and port, with no scheme, is read as that address
// (schemeless; 0083-MADR D11).
//
// A URL must carry this sign-in's state and a code, and, where the step
// accepts only a URL, a client_id; a bare code is taken only where the step
// accepts one. A URL with this state and an error is the identity provider's
// answer, not a bad paste: it passes, and auth ends the sign-in with it
// (0074-MADR D4).
func (c pasteCheck) check(paste string) string {
	reason, _ := c.read(paste)
	return reason
}

// schemeless reports whether paste, trimmed, begins with the callback's exact
// host and port and then ends or goes on with a path, a query or a fragment:
// the address bar shows the host and port without the scheme until it is
// tapped, so a paste can lack http:// (0083-MADR D11, ST83). A longer port
// that begins with the callback's, another port, another host and a bare code
// do not qualify.
func (c pasteCheck) schemeless(paste string) bool {
	if c.callback == "" || !strings.HasPrefix(paste, c.callback) {
		return false
	}
	rest := paste[len(c.callback):]
	return rest == "" || strings.ContainsRune("/?#", rune(rest[0]))
}

// read is check, and with it the address to hand on: "" unless the paste
// began with the callback's host and port and no scheme, when it is the paste,
// trimmed, with http:// in front, which auth then reads as the address it is.
// The retry is of the parse only: the state, the code and a client_id are
// checked as for any address, a step that takes a code only refuses it, and a
// retry that does not parse is refused (0083-MADR D11, ST76 to ST85).
func (c pasteCheck) read(paste string) (reason, address string) {
	paste = strings.TrimSpace(paste)
	if paste == "" {
		return "Nothing was pasted. " + promptFor(c.accepts), ""
	}
	parsed, err := url.Parse(paste)
	if err != nil || parsed.Scheme == "" {
		if c.schemeless(paste) {
			retried, retryErr := url.Parse("http://" + paste)
			switch {
			case retryErr != nil:
				return "That is not a usable address. " + promptFor(c.accepts), ""
			case retried.Host == c.callback:
				address, parsed = "http://"+paste, retried
				return c.checkURL(parsed), address
			}
		}
		if c.accepts == authcontract.PasteURLOnly {
			return "That is not a URL. " + promptFor(c.accepts), ""
		}
		return "", ""
	}
	return c.checkURL(parsed), ""
}

// checkURL is the check of an address that parsed with a scheme.
func (c pasteCheck) checkURL(parsed *url.URL) string {
	if c.accepts == authcontract.PasteCode {
		return "That is a URL, not a code. " + promptFor(c.accepts)
	}
	query := parsed.Query()
	switch {
	case c.state != "" && query.Get("state") != c.state:
		return "That URL is not from this sign-in (its state does not match). " + promptFor(c.accepts)
	case query.Get("error") != "":
		return ""
	case query.Get("code") == "":
		return "That URL carries no code. " + promptFor(c.accepts)
	case c.accepts == authcontract.PasteURLOnly && query.Get("client_id") == "":
		return "That URL carries no client_id. " + promptFor(c.accepts)
	}
	return ""
}
