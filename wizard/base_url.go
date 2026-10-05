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
// used only after the user confirms. A blank URL keeps the provider's
// default. A refused URL is an error matching errBaseURLRefused.
func validateBaseURL(p Prompter, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	u, err := url.Parse(raw)
	switch {
	case err != nil:
		return "", fmt.Errorf("%w: %s is not a URL", errBaseURLRefused, raw)
	case u.Scheme != "http" && u.Scheme != "https" || u.Host == "":
		return "", fmt.Errorf("%w: %s needs http:// or https:// and a host", errBaseURLRefused, raw)
	case u.User != nil:
		return "", fmt.Errorf("%w: a base URL carries no user name or password", errBaseURLRefused)
	case u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(raw, "#"):
		return "", fmt.Errorf("%w: %s has a query or a fragment", errBaseURLRefused, raw)
	}
	if u.Scheme == "http" && !loopbackHost(u.Hostname()) {
		ok, err := p.Confirm(fmt.Sprintf("%s is plain http to a remote host, so the key travels unencrypted. Use it?", raw), false)
		if err != nil {
			return "", fmt.Errorf("confirm base URL: %w", err)
		}
		if !ok {
			return "", fmt.Errorf("%w: plain http not confirmed", errBaseURLRefused)
		}
	}
	return raw, nil
}

// loopbackHost reports localhost or a loopback IP address.
func loopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
