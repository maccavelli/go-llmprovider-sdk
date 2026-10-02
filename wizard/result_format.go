package wizard

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// redactedSecret stands in for a secret in a formatted Result, as it does in
// llmprovider's formatted tokens.
const redactedSecret = "[redacted]"

// secretText is "" for an empty secret and redactedSecret for any other.
func secretText(secret string) string {
	if secret == "" {
		return ""
	}
	return redactedSecret
}

// String renders the Result without its API key (0016-MADR D5). Fields that
// are unset are left out.
func (r Result) String() string {
	var b strings.Builder
	b.WriteString("Result{")
	sep := ""
	for _, a := range r.attrs() {
		fmt.Fprintf(&b, "%s%s:%s", sep, a.Key, a.Value.String())
		sep = " "
	}
	b.WriteString("}")
	return b.String()
}

// GoString renders the Result for %#v, without its API key.
func (r Result) GoString() string { return "wizard." + r.String() }

// LogValue renders the Result for slog, without its API key.
func (r Result) LogValue() slog.Value { return slog.GroupValue(r.attrs()...) }

// attrs lists the Result's set fields, with the API key redacted. JSON is
// not redacted: the consumer persists a Result (0016-MADR A9).
func (r Result) attrs() []slog.Attr {
	var out []slog.Attr
	text := func(key, value string) {
		if value != "" {
			out = append(out, slog.String(key, value))
		}
	}
	text("Provider", string(r.Provider))
	text("Kind", string(r.Kind))
	text("APIKey", secretText(r.APIKey))
	if !r.TokenExpiry.IsZero() {
		text("TokenExpiry", r.TokenExpiry.UTC().Format(time.RFC3339))
	}
	text("Issuer", r.Issuer)
	text("ClientID", r.ClientID)
	text("AccountID", r.AccountID)
	if r.FedRAMP {
		out = append(out, slog.Bool("FedRAMP", true))
	}
	text("Model", r.Model)
	text("BaseURL", r.BaseURL)
	if len(r.Fallbacks) > 0 {
		text("Fallbacks", "["+strings.Join(r.Fallbacks, " ")+"]")
	}
	text("VendorAuthPath", r.VendorAuthPath)
	text("Organization", r.Organization)
	return out
}
