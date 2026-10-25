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
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"Hello!"}}]}`)
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
