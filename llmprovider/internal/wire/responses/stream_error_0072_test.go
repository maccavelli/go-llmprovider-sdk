package responses

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestStream_FailedHeaders_0072 (0072-MADR D3, X11): a failed response's
// error.headers give RetryAfter, as Codex reads them, in a stream and in a
// 200 JSON reply. The message states no delay, so only the error object
// carries it. The fixture is synthetic, from Codex's parser.
func TestStream_FailedHeaders_0072(t *testing.T) {
	const failure = `{"code":"rate_limit_exceeded","message":"Slow down","headers":{"retry-after":"7"}}`
	_, streamed := ReadStream("openai", strings.NewReader(sse(`{"type":"response.failed","response":{"error":`+failure+`}}`)))
	_, replied := DecodeFor("openai")(strings.NewReader(`{"status":"failed","error":` + failure + `}`))
	for name, err := range map[string]error{"streamed": streamed, "replied": replied} {
		apiErr, ok := errors.AsType[*llmprovider.APIError](err)
		if !ok || !errors.Is(err, llmprovider.ErrRateLimited) || apiErr.RetryAfter != 7*time.Second {
			t.Errorf("%s: %v; want ErrRateLimited with RetryAfter 7s", name, err)
		}
	}
}
