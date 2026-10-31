package llmprovider_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
)

// ExampleGenerateText asks a provider for text. A Fake stands in for the
// provider that providers.New, or a provider package's New, builds.
func ExampleGenerateText() {
	p := llmtest.NewFake("example", llmprovider.Capabilities{}).ReplyText("Hello!")
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

// ExampleWithRetry retries a failure the service may recover from: here the
// first try finds the provider unavailable, and the second answers.
func ExampleWithRetry() {
	fake := llmtest.NewFake("example", llmprovider.Capabilities{}).
		Fail(llmprovider.ErrProviderUnavailable).
		ReplyText("Hello, at the second try.")
	p := llmprovider.WithRetry(fake, llmprovider.RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond})
	text, err := llmprovider.GenerateText(context.Background(), p, &llmprovider.Request{
		Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Say hello."}},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(text)
	// Output: Hello, at the second try.
}

// ExampleWithStreamRetry streams text as it is generated, and retries a
// failure before the first delta: here a local server is unavailable once,
// then streams "Hello" in two deltas, as the OpenAI Responses API does.
func ExampleWithStreamRetry() {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, `{"error":{"message":"busy"}}`, http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"Hel\"}\n\n"+
			"data: {\"type\":\"response.output_text.delta\",\"delta\":\"lo\"}\n\n"+
			"data: {\"type\":\"response.output_item.done\",\"item\":{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"Hello\"}]}}\n\n"+
			"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp-example\"}}\n\n")
	}))
	defer srv.Close()
	inner, err := openai.New(llmprovider.WithAPIKey("example-key"), llmprovider.WithModel("gpt-example"),
		llmprovider.WithBaseURL(srv.URL))
	if err != nil {
		fmt.Println(err)
		return
	}
	p := llmprovider.WithStreamRetry(inner, llmprovider.RetryPolicy{BaseDelay: time.Millisecond},
		func(_ context.Context, n llmprovider.RetryNotice) {
			fmt.Printf("retrying after attempt %d\n", n.Attempt)
		})
	for ev, err := range llmprovider.Stream(context.Background(), p, &llmprovider.Request{
		Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Say hello."}},
	}) {
		if err != nil {
			fmt.Println(err)
			return
		}
		if ev.Type == llmprovider.EventTextDelta {
			fmt.Printf("delta %q\n", ev.Text)
		}
	}
	// Output:
	// retrying after attempt 1
	// delta "Hel"
	// delta "lo"
}

// exampleHeaders adds X-Example to each request, then sends it on base, the
// bounded transport NewHTTPClient gave.
type exampleHeaders struct{ base http.RoundTripper }

func (h exampleHeaders) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("X-Example", "from-the-wrapper")
	return h.base.RoundTrip(r)
}

// ExampleNewHTTPClient adds a request header by wrapping the default
// client's transport, so its bounds stay, and gives the client to a
// provider with WithHTTPClient.
func ExampleNewHTTPClient() {
	var seen atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.Store(r.Header.Get("X-Example"))
		fmt.Fprint(w, `{"id":"resp-example","output":[{"type":"message","content":[{"type":"output_text","text":"Hello"}]}]}`)
	}))
	defer srv.Close()
	c := llmprovider.NewHTTPClient()
	c.Transport = exampleHeaders{base: c.Transport}
	p, err := openai.New(llmprovider.WithAPIKey("example-key"), llmprovider.WithModel("gpt-example"),
		llmprovider.WithBaseURL(srv.URL), llmprovider.WithHTTPClient(c))
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
	fmt.Println("the server saw X-Example:", seen.Load())
	// Output:
	// Hello
	// the server saw X-Example: from-the-wrapper
}
