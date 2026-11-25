package gemini_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/gemini"
)

// ExampleNew builds the Gemini provider and asks it for text. The httptest
// server stands in for the Gemini Interactions API, so that the example runs offline; without
// WithBaseURL the provider sends to the service itself.
func ExampleNew() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// A Generate that names no output limit reads the service's event
		// stream (0088-MADR ST88).
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: interaction.created\ndata: {\"event_type\":\"interaction.created\",\"interaction\":{\"id\":\"v1_example\",\"model\":\"gemini-x\"}}\n\n"+
			"event: step.start\ndata: {\"event_type\":\"step.start\",\"index\":0,\"step\":{\"type\":\"model_output\"}}\n\n"+
			"event: step.delta\ndata: {\"event_type\":\"step.delta\",\"index\":0,\"delta\":{\"type\":\"text\",\"text\":\"Hello!\"}}\n\n"+
			"event: interaction.completed\ndata: {\"event_type\":\"interaction.completed\",\"interaction\":{\"status\":\"completed\"}}\n\n")
	}))
	defer srv.Close()

	p, err := gemini.New(
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
