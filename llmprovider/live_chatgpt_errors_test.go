//go:build live_gateways

package llmprovider_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestLive_ChatGPTErrorDetail: the backend refuses a model outside the Codex
// catalog with 400 {"detail": ...}; the error carries that text.
func TestLive_ChatGPTErrorDetail(t *testing.T) {
	session := liveChatGPTSession(t)
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	_, err := liveOpenAI(t, session, "gpt-4.1-mini").Generate(ctx, userText("hi"))
	var apiErr *llmprovider.APIError
	if !errors.Is(err, llmprovider.ErrInvalidRequest) || !errors.As(err, &apiErr) || !strings.Contains(apiErr.Message, "not supported") {
		t.Fatalf("err = %v, want ErrInvalidRequest carrying the backend's detail", err)
	}
}
