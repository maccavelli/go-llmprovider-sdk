package messages

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestEvents_ErrorHeaders_0072 (0072-MADR D3): the stream's error object
// reaches the classifier, whose headers member gives RetryAfter. It is a
// plumbing fixture: Anthropic is not known to send headers in a stream
// error, and the message states no delay, so only the object carries it.
func TestEvents_ErrorHeaders_0072(t *testing.T) {
	stream := sse(`{"type":"error","error":{"type":"rate_limit_error","message":"slow down","headers":{"retry-after":"7"}}}`)
	err := Events("claude", strings.NewReader(stream), func(llmprovider.Event) bool { return true })
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	if !ok || !errors.Is(err, llmprovider.ErrRateLimited) || apiErr.RetryAfter != 7*time.Second {
		t.Errorf("%v; want ErrRateLimited with RetryAfter 7s", err)
	}
}
