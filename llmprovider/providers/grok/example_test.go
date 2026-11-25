package grok_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/grok"
)

// ExampleNew builds the Grok provider and asks it for text. The httptest
// server stands in for the xAI Responses API, so that the example runs offline; without
// WithBaseURL the provider sends to the service itself.
func ExampleNew() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// A Generate that names no output limit reads the service's event
		// stream (0088-MADR ST88).
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"Hello!\"}\n\n"+
			"data: {\"type\":\"response.output_item.done\",\"item\":{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"Hello!\"}]}}\n\n"+
			"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_example\"}}\n\n")
	}))
	defer srv.Close()

	p, err := grok.New(
		llmprovider.WithAPIKey("your-api-key"),
		llmprovider.WithBaseURL(srv.URL),
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
