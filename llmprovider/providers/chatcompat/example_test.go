package chatcompat_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/chatcompat"
)

// ExampleNew builds a chatcompat provider for a local server and asks it for
// text. The httptest server stands in for the server's Chat Completions, such
// as LM Studio's at http://localhost:1234/v1, so that the example runs
// offline. A local server takes no key, so none is given.
func ExampleNew() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// A Generate that names no output limit reads the service's event
		// stream (0088-MADR ST88).
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"Hello!\"},\"finish_reason\":\"stop\"}]}\n\n"+
			"data: [DONE]\n\n")
	}))
	defer srv.Close()

	p, err := chatcompat.New(
		chatcompat.WithID("lmstudio"),
		llmprovider.WithBaseURL(srv.URL),
		llmprovider.WithModel("qwen3-8b"),
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

// ExampleWithResponses builds a chatcompat provider for a server the caller
// knows speaks OpenAI's Responses API, such as a local Ollama's
// http://localhost:11434/v1, and asks it for text: the request goes to
// {base}/responses. The httptest server stands in for it, so that the
// example runs offline.
func ExampleWithResponses() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			http.NotFound(w, r)
			return
		}
		// A Generate that names no output limit reads the service's event
		// stream (0088-MADR ST88).
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"Hello!\"}\n\n"+
			"data: {\"type\":\"response.output_item.done\",\"item\":{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"Hello!\"}]}}\n\n"+
			"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\"}}\n\n")
	}))
	defer srv.Close()

	p, err := chatcompat.New(
		chatcompat.WithID("ollama-responses"),
		chatcompat.WithResponses(),
		llmprovider.WithBaseURL(srv.URL),
		llmprovider.WithModel("qwen3:8b"),
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
