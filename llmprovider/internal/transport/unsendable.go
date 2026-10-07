package transport

import (
	"errors"
	"net/url"
	"strings"
)

// unsendableCauses are net/http's and net/url's messages for a request that
// cannot be built or sent as given. They have no exported error values to
// match, so the text is matched; a retry cannot mend any of them.
var unsendableCauses = []string{
	"invalid header field value",
	"invalid header field name",
	"unsupported protocol scheme",
	"invalid control character in URL",
	"no Host in request URL",
}

// Unsendable reports whether err is a failure to build or send a request that
// no retry can mend: a URL that does not parse, a scheme no transport serves,
// or a header value net/http refuses, such as a key with a newline. A
// failure to reach the service is not one (0026-MADR F11).
func Unsendable(err error) bool {
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		return false
	}
	if urlErr.Op == "parse" {
		return true
	}
	msg := urlErr.Err.Error()
	for _, cause := range unsendableCauses {
		if strings.Contains(msg, cause) {
			return true
		}
	}
	return false
}
