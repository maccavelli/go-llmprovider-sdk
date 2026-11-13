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
	state   string
	accepts authcontract.Paste
}

// newPasteCheck reads the state from authorizeURL.
func newPasteCheck(authorizeURL string, accepts authcontract.Paste) pasteCheck {
	state := ""
	if u, err := url.Parse(authorizeURL); err == nil {
		state = u.Query().Get("state")
	}
	return pasteCheck{state: state, accepts: accepts}
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
// reads it: anything with a scheme.
//
// A URL must carry this sign-in's state and a code, and, where the step
// accepts only a URL, a client_id; a bare code is taken only where the step
// accepts one. A URL with this state and an error is the identity provider's
// answer, not a bad paste: it passes, and auth ends the sign-in with it
// (0074-MADR D4).
func (c pasteCheck) check(paste string) string {
	paste = strings.TrimSpace(paste)
	if paste == "" {
		return "Nothing was pasted. " + promptFor(c.accepts)
	}
	parsed, err := url.Parse(paste)
	if err != nil || parsed.Scheme == "" {
		if c.accepts == authcontract.PasteURLOnly {
			return "That is not a URL. " + promptFor(c.accepts)
		}
		return ""
	}
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
