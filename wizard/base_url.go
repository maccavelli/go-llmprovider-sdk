package wizard

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// errBaseURLRefused marks a base URL the wizard asks for again.
var errBaseURLRefused = errors.New("wizard: base URL refused")

// validateBaseURL checks a remote provider's base URL (0021-MADR Z7). It
// must be http or https with a host, and carry no user name or password, no
// query and no fragment, since the provider appends its own paths. Plain
// http to a host that is not loopback sends the key unencrypted, so it is
// used only after the user confirms; when ask is false, as for a URL the
// caller holds under Options.NoEndpointPrompt, it is used with a warning
// notice instead (0059-MADR W9). A blank URL keeps the provider's default. A
// refused URL is an error matching errBaseURLRefused. Neither the error nor
// a notice shows the URL's user information, query or fragment, where a key
// can be, and a value that is not an http or https URL with a host is not
// shown at all (0066-MADR V9).
func validateBaseURL(p Prompter, raw string, ask bool) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	u, err := url.Parse(raw)
	switch {
	case err != nil:
		return "", fmt.Errorf("%w: the endpoint is not a URL", errBaseURLRefused)
	case u.Scheme != "http" && u.Scheme != "https" || u.Host == "":
		return "", fmt.Errorf("%w: an endpoint needs http:// or https:// and a host", errBaseURLRefused)
	case u.User != nil:
		return "", fmt.Errorf("%w: a base URL carries no user name or password", errBaseURLRefused)
	case u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(raw, "#"):
		return "", fmt.Errorf("%w: %s has a query or a fragment", errBaseURLRefused, shownURL(u, raw))
	}
	if u.Scheme == "http" && !loopbackHost(u.Hostname()) {
		if !ask {
			p.Notify(LevelWarn, "%s is plain http to a remote host, so the key travels unencrypted", shownURL(u, raw))
			return raw, nil
		}
		ok, err := p.Confirm(fmt.Sprintf("%s is plain http to a remote host, so the key travels unencrypted. Use it?",
			shownURL(u, raw)), false)
		if err != nil {
			return "", fmt.Errorf("confirm base URL: %w", err)
		}
		if !ok {
			return "", fmt.Errorf("%w: plain http not confirmed", errBaseURLRefused)
		}
	}
	return raw, nil
}

// shownURL is u, parsed from raw, as a notice shows it: its scheme, host and
// path, then "?…" for a query and "#…" for a fragment, and never its user
// information (0066-MADR V9).
func shownURL(u *url.URL, raw string) string {
	shown := u.Scheme + "://" + u.Host + u.EscapedPath()
	if u.RawQuery != "" || u.ForceQuery {
		shown += "?…"
	}
	if u.Fragment != "" || strings.Contains(raw, "#") {
		shown += "#…"
	}
	return shown
}

// loopbackHost reports localhost or a loopback IP address.
func loopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
