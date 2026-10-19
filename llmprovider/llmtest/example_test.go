package llmtest_test

import (
	"context"
	"fmt"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"
)

// ExampleFake scripts a provider's answers for a test of code that uses one,
// and reads back the requests it was sent.
func ExampleFake() {
	fake := llmtest.NewFake("example", llmprovider.Capabilities{}).ReplyText("first").ReplyText("second")
	for range 2 {
		text, err := llmprovider.GenerateText(context.Background(), fake, &llmprovider.Request{
			Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Next?"}},
		})
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(text)
	}
	fmt.Println(len(fake.Requests()), "requests")
	// Output:
	// first
	// second
	// 2 requests
}
