package together

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestTogether_TypedErrorInside200SentOnce (0028-MADR D-H8): a 200 carrying
// an invalid_request_error with no code is classified by its type: an
// overflow is ErrContextOverflow, another invalid request ErrInvalidRequest,
// and neither is sent again.
func TestTogether_TypedErrorInside200SentOnce(t *testing.T) {
	req := &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}
	for _, c := range []struct {
		name, body string
		kind       error
	}{
		{"overflow", `{"error":{"message":"This model's maximum context length is 8192 tokens. However, you requested 9000 tokens (8000 in the messages, 1000 in the completion). Please reduce the length of the messages or completion.","type":"invalid_request_error","code":null}}`,
			llmprovider.ErrContextOverflow},
		{"invalid", `{"error":{"message":"'messages' must contain at least one message","type":"invalid_request_error","code":null}}`,
			llmprovider.ErrInvalidRequest},
	} {
		t.Run(c.name, func(t *testing.T) {
			srv, hits := replyServer(t, func(w http.ResponseWriter) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, c.body)
			})
			_, err := replyProvider(t, srv.URL).Generate(context.Background(), req)
			if hits.Load() != 1 || !errors.Is(err, c.kind) {
				t.Fatalf("%d request(s), %v; want 1 and %v", hits.Load(), err, c.kind)
			}
		})
	}
}
