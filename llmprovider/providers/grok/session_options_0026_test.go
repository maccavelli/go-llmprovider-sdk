package grok

import (
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestSession_LaterProviderOptionsApply (0026-MADR F37): as for OpenAI, a
// provider's default client is the session's only until a caller's arrives,
// and a caller's client and logger are never replaced.
func TestSession_LaterProviderOptionsApply(t *testing.T) {
	session := &auth.OAuthSession{Provider: llmprovider.ProviderGrok, Access: "a", Refresh: "r",
		Expiry: time.Now().Add(time.Hour)}
	build(t, llmprovider.WithTokenSource(session), llmprovider.WithModel("grok-4.5"))
	if session.HTTPClient == nil {
		t.Fatal("a provider without WithHTTPClient gave the session no client (0016-MADR D8)")
	}
	client := &http.Client{}
	logger := slog.New(slog.DiscardHandler)
	build(t, llmprovider.WithTokenSource(session), llmprovider.WithModel("grok-4.5"),
		llmprovider.WithHTTPClient(client), llmprovider.WithLogger(logger))
	if session.HTTPClient != client || session.Logger != logger {
		t.Fatalf("after a second New with WithHTTPClient and WithLogger: client is the caller's %t, logger %t; want both",
			session.HTTPClient == client, session.Logger == logger)
	}
	build(t, llmprovider.WithTokenSource(session), llmprovider.WithModel("grok-4.5"))
	if session.HTTPClient != client || session.Logger != logger {
		t.Fatal("a third New with no options replaced the caller's client or logger")
	}
}
