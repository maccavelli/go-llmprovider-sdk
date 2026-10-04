package openai

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestNew_SessionLogsThroughWithLogger (0020-MADR F48): a session's failed
// rotation save is logged through the provider's WithLogger logger
// (0016-MADR D4), so New hands the session that logger.
func TestNew_SessionLogsThroughWithLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	session := &auth.OAuthSession{Issuer: auth.DefaultOpenAIIssuer, Access: "a", Refresh: "r",
		Expiry: time.Now().Add(time.Hour)}
	if _, err := New(llmprovider.WithTokenSource(session), llmprovider.WithModel("gpt-6-astra"),
		llmprovider.WithLogger(logger)); err != nil {
		t.Fatal(err)
	}
	if session.Logger != logger {
		t.Error("the session does not log through the provider's WithLogger logger")
	}
}
