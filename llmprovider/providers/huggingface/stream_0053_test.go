package huggingface

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestStream_IndexOutOfRange_0053 (0053-MADR D2, M3): a tool-call index of -1
// in a Chat Completions stream ends the public Stream with ErrIncomplete; it
// panicked the ranging goroutine.
func TestStream_IndexOutOfRange_0053(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, `data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":-1,"id":"c","function":{"name":"f","arguments":"{}"}}]}}]}`+
			"\n\ndata: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	p, err := New(append([]llmprovider.Option{llmprovider.WithAPIKey("hf_0053"), llmprovider.WithModel("openai/gpt-oss-120b"),
		llmprovider.WithBaseURL(srv.URL)}, metadataOff()...)...)
	if err != nil {
		t.Fatal(err)
	}
	var last error
	var panicked any
	func() {
		defer func() { panicked = recover() }()
		for _, err := range llmprovider.Stream(context.Background(), p, &llmprovider.Request{
			Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}) {
			if err != nil {
				last = err
			}
		}
	}()
	if panicked != nil || !errors.Is(last, llmprovider.ErrIncomplete) {
		t.Errorf("panic %v, error %v; want ErrIncomplete and no panic", panicked, last)
	}
}
