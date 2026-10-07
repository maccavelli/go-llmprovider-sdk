package auth

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestOAuthHTTPStatusError_StripsControl (0026-MADR F18): the token
// endpoint's error body is the IdP's text, so it is stripped of control
// characters as well as redacted before it reaches an error.
func TestOAuthHTTPStatusError_StripsControl(t *testing.T) {
	resp := &http.Response{Status: "400 Bad Request", StatusCode: http.StatusBadRequest,
		Body: io.NopCloser(strings.NewReader(`{"error":"invalid_grant"}` + "\x1b]0;owned\x07\x1b[2J"))}
	msg := oauthHTTPStatusError("refresh", resp).Error()
	if strings.ContainsAny(msg, "\x1b\x07") {
		t.Fatalf("error %q holds a control character", msg)
	}
	if !strings.Contains(msg, "invalid_grant") {
		t.Fatalf("error %q lost the IdP's code", msg)
	}
}
