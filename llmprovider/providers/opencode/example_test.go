package opencode_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// ExampleNewZen builds the OpenCode Zen provider and asks it for text. The httptest
// server stands in for the OpenCode Zen gateway's Chat Completions route, so that the example runs offline; without
// WithBaseURL the provider sends to the service itself.
func ExampleNewZen() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"Hello!"}}]}`)
	}))
	defer srv.Close()

	p, err := opencode.NewZen(
		llmprovider.WithAPIKey("your-api-key"),
		llmprovider.WithBaseURL(srv.URL),
		llmprovider.WithoutModelMetadata(),
		opencode.WithRoute(opencode.RouteChatCompletions),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	text, err := llmprovider.GenerateText(context.Background(), p, &llmprovider.Request{
		Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Say hello."}},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(text)
	// Output: Hello!
}
