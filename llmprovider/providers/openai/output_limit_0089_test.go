package openai

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestOutputLimit_Default_0089 (0089-MADR DL45, DL8's plan case; 0088-MADR
// ST4): the ChatGPT backend and a ChatGPT plan session carry no output limit,
// whatever the caller gives, and report none, basis none. It is here, not in
// llmprovider/providers, since the sessions it signs in with are this
// package's test helpers.
func TestOutputLimit_Default_0089(t *testing.T) {
	for _, tc := range []struct {
		name    string
		session func() *auth.OAuthSession
		model   string
	}{
		{"the ChatGPT backend", chatGPTSession, "gpt-5.4-mini"},
		{"a ChatGPT plan session", planSession, "gpt-6.1-sol"},
	} {
		for _, given := range []bool{false, true} {
			client, c := chatGPTClient(t, fixture(t, "chatgpt-text.sse"))
			var opts []llmprovider.Option
			req := text("hi")
			if given {
				opts = append(opts, llmprovider.WithMaxTokens(2048))
				req.MaxOutputTokens = 512
			}
			p := sessionProvider(t, tc.session(), tc.model, append(opts, llmprovider.WithHTTPClient(client))...)
			resp, err := p.Generate(context.Background(), req)
			if err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if v, sent := c.body["max_output_tokens"]; sent {
				t.Errorf("%s, a caller's value %t: max_output_tokens %v; want none", tc.name, given, v)
			}
			if want := (llmprovider.SentLimit{Basis: llmprovider.LimitBasisNone}); resp.OutputLimit != want {
				t.Errorf("%s, a caller's value %t: OutputLimit = %+v; want %+v", tc.name, given, resp.OutputLimit, want)
			}
		}
	}
}
