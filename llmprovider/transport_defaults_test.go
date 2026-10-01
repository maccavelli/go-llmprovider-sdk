package llmprovider

import (
	"net/http"
	"testing"
)

// TestShareHTTPClient_KeepsTheSessionsOwn: a session that already has a
// client keeps it; anything but an OAuth session is left alone.
func TestShareHTTPClient_KeepsTheSessionsOwn(t *testing.T) {
	own, provider := &http.Client{}, &http.Client{}
	session := &OAuthSession{HTTPClient: own}
	ShareHTTPClient(session, provider)
	if session.HTTPClient != own {
		t.Error("a session's own client was replaced")
	}
	bare := &OAuthSession{}
	ShareHTTPClient(bare, nil)
	if bare.HTTPClient != nil {
		t.Error("a nil client was shared")
	}
	ShareHTTPClient(NewStaticToken("k"), provider) // must not panic
}
