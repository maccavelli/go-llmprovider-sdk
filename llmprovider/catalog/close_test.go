package catalog

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// closeFails is a body whose Close fails.
type closeFails struct{ io.Reader }

func (closeFails) Close() error { return errors.New("close broke") }

// TestCloseBody_LogsToTheListingsLogger (0015-PLAN S10): a close failure goes
// to the logger the caller passed, never to the global one; with none it is
// dropped.
func TestCloseBody_LogsToTheListingsLogger(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&out, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := testConfig(t, llmprovider.WithLogger(logger))
	cfg.closeBody(&http.Response{Body: closeFails{strings.NewReader("")}})
	if !strings.Contains(out.String(), "close broke") {
		t.Errorf("log = %q, want the close failure", out.String())
	}
	config{}.closeBody(&http.Response{Body: closeFails{strings.NewReader("")}}) // must not panic
}
