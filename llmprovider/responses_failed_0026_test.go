package llmprovider_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/grok"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// TestResponsesProviders_FailedStatusUsesTheirVocabulary (0026-MADR F28, the
// PLAN's deviation D7): a non-streamed "status":"failed" reply is classified
// by the provider's own error vocabulary, and names the provider, as the
// stream's response.failed is.
func TestResponsesProviders_FailedStatusUsesTheirVocabulary(t *testing.T) {
	reply := func(body string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, body)
		}))
	}
	quota := reply(`{"status":"failed","error":{"code":"insufficient_quota","message":"quota"}}`)
	defer quota.Close()
	failed := reply(`{"status":"failed","error":{"code":"server_error","message":"boom"}}`)
	defer failed.Close()

	for _, c := range []struct {
		name     string
		build    func(base string) (llmprovider.Provider, error)
		srv      *httptest.Server
		kind     error
		provider string
	}{
		{"openai quota", func(base string) (llmprovider.Provider, error) {
			return openai.New(llmprovider.WithAPIKey("sk-test"), llmprovider.WithModel("gpt-5.5"), llmprovider.WithBaseURL(base))
		}, quota, llmprovider.ErrQuotaExhausted, "openai"},
		{"grok", func(base string) (llmprovider.Provider, error) {
			return grok.New(llmprovider.WithAPIKey("xai-test"), llmprovider.WithModel("grok-4.5"), llmprovider.WithBaseURL(base))
		}, failed, llmprovider.ErrProviderUnavailable, "grok"},
		{"opencode responses", func(base string) (llmprovider.Provider, error) {
			return opencode.NewGo(llmprovider.WithAPIKey("oc-test"), llmprovider.WithModel("gpt-6-luna"),
				llmprovider.WithBaseURL(base), opencode.WithRoute(opencode.RouteResponses), llmprovider.WithoutModelMetadata())
		}, failed, llmprovider.ErrProviderUnavailable, "opencode-go/responses"},
	} {
		p, err := c.build(c.srv.URL)
		if err != nil {
			t.Fatalf("%s: New: %v", c.name, err)
		}
		_, err = p.Generate(context.Background(), &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Text: "hi"}}})
		apiErr, ok := errors.AsType[*llmprovider.APIError](err)
		if !ok || !errors.Is(err, c.kind) || apiErr.Provider != c.provider {
			t.Errorf("%s: %v; want %v from provider %q", c.name, err, c.kind, c.provider)
		}
	}
}
