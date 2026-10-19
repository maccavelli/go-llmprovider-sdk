package llmprovider_test

import (
	"context"
	"fmt"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"
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
