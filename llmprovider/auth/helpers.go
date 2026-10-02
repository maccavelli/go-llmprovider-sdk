package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// auth's copies of llmprovider's unexported helpers. Each package closes its
// own response bodies, because bodyclose sees only a close in the same
// package.

// redactedSecret stands in for a secret in a formatted value.
const redactedSecret = "[redacted]"

// secretText is "" for an empty secret and redactedSecret for any other.
func secretText(secret string) string {
	if secret == "" {
		return ""
	}
	return redactedSecret
}

// expiryText renders an expiry, or "none" for the zero time.
func expiryText(t time.Time) string {
	if t.IsZero() {
		return "none"
	}
	return t.UTC().Format(time.RFC3339)
}

// closeResponseBody closes resp's body, logging a failure at debug level.
func closeResponseBody(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	if err := resp.Body.Close(); err != nil {
		slog.Debug("auth: close response body", "error", err)
	}
}

// jsonString is raw as a JSON string, or "" when it is not one.
func jsonString(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return ""
	}
	return s
}

// tokenFuture is one in-flight refresh that concurrent callers share.
type tokenFuture struct {
	done chan struct{}
	tok  llmprovider.Token
	err  error
}
