//go:build live_gateways

package llmprovider_test

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
)

// TestLive_ResponsesStoreFalseOpenAI: OpenAI (API key) accepts store:false
// from openai.WithStore(false). It was the OpenAI row of
// TestLive_ResponsesStoreFalse.
func TestLive_ResponsesStoreFalseOpenAI(t *testing.T) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY unset")
	}
	var sent []byte
	client := &http.Client{Transport: liveRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodPost && r.Body != nil {
			b, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			sent = b
			r.Body = io.NopCloser(bytes.NewReader(b))
		}
		return http.DefaultTransport.RoundTrip(r)
	})}
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	p, err := openai.New(llmprovider.WithAPIKey(key), llmprovider.WithModel("gpt-6-luna"), openai.WithStore(false),
		llmprovider.WithHTTPClient(client))
	if err != nil {
		t.Fatal(err)
	}
	out, err := llmprovider.GenerateText(ctx, p, userText("Reply with only the word ALPHA"))
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !bytes.Contains(sent, []byte(`"store":false`)) || !strings.Contains(strings.ToUpper(out), "ALPHA") {
		t.Fatalf("out = %q, sent %s", out, sent)
	}
}
