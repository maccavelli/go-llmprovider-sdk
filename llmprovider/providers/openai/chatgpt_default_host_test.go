package openai

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestListChatGPTModels_DefaultHostIsCodexNotPlatform (0010-MADR D8): with no
// WithBaseURL, a ChatGPT session lists the Codex catalog at chatgpt.com, never
// the platform API. The transport records the URL and never dials.
func TestListChatGPTModels_DefaultHostIsCodexNotPlatform(t *testing.T) {
	var got []*url.URL
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		got = append(got, r.URL)
		return nil, errors.New("not dialled")
	})}
	p := sessionProvider(t, chatGPTSession(), "gpt-6-astra", llmprovider.WithHTTPClient(client))
	if _, err := list(t, p); err == nil {
		t.Fatal("ListModels succeeded through a transport that never answers")
	}
	if len(got) != 1 {
		t.Fatalf("requests = %d, want one listing request", len(got))
	}
	u := got[0]
	if u.Host != "chatgpt.com" || !strings.HasSuffix(u.Path, "/backend-api/codex/models") {
		t.Errorf("listing URL = %s, want https://chatgpt.com/backend-api/codex/models", u.Redacted())
	}
	if u.Host == "api.openai.com" {
		t.Errorf("a ChatGPT session listed the platform API: %s", u.Redacted())
	}
}
