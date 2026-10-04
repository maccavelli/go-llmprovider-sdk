package auth

import (
	"io"
	"log/slog"
	"testing"
)

// TestOAuthSession_UseLogger (0020-MADR F48): a session without a logger
// takes the provider's; one with its own keeps it; nil changes nothing.
func TestOAuthSession_UseLogger(t *testing.T) {
	provider := slog.New(slog.NewTextHandler(io.Discard, nil))
	own := slog.New(slog.NewTextHandler(io.Discard, nil))

	bare := &OAuthSession{}
	bare.UseLogger(nil)
	if bare.Logger != nil {
		t.Error("UseLogger(nil) set a logger")
	}
	bare.UseLogger(provider)
	if bare.Logger != provider {
		t.Error("a session without a logger did not take the provider's")
	}
	withOwn := &OAuthSession{Logger: own}
	withOwn.UseLogger(provider)
	if withOwn.Logger != own {
		t.Error("a session with its own logger lost it")
	}
}
