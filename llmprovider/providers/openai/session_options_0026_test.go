package openai

import (
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestSession_LaterProviderOptionsApply (0026-MADR F37): a provider built on
// a session without WithHTTPClient or WithLogger does not fix the session's
// client and logger, so a later provider's options reach it. A caller's
// client stays the session's: a later provider's default does not replace it.
func TestSession_LaterProviderOptionsApply(t *testing.T) {
	session := &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: auth.DefaultOpenAIIssuer,
		Access: "a", Refresh: "r", Expiry: time.Now().Add(time.Hour)}
	sessionProvider(t, session, "gpt-5.5")
	client := &http.Client{}
	logger := slog.New(slog.DiscardHandler)
	sessionProvider(t, session, "gpt-5.5", llmprovider.WithHTTPClient(client), llmprovider.WithLogger(logger))
	if session.HTTPClient != client || session.Logger != logger {
		t.Fatalf("after a second New with WithHTTPClient and WithLogger: client is the caller's %t, logger %t; want both",
			session.HTTPClient == client, session.Logger == logger)
	}
	sessionProvider(t, session, "gpt-5.5")
	if session.HTTPClient != client || session.Logger != logger {
		t.Fatal("a third New with no options replaced the caller's client or logger")
	}
}
