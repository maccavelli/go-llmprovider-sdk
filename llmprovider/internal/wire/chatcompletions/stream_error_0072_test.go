package chatcompletions

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestEvents_GatewayErrorHeaders_0072 (0072-MADR D3; 0072-PLAN Deviation 1):
// a gateway's error object, at the top of a chunk and in its choice, reaches
// the classifier, whose headers member gives RetryAfter. It is a plumbing
// fixture in Kilo's stream-error shape (0072-MADR K3) with a headers member
// no gateway is known to send; the message states no delay, so only the
// object carries it.
func TestEvents_GatewayErrorHeaders_0072(t *testing.T) {
	const failure = `{"message":"Rate limited","code":429,"headers":{"retry-after":"7"}}`
	for name, chunk := range map[string]string{
		"at the top":    `{"error":` + failure + `,"choices":[{"index":0,"delta":{"content":""},"finish_reason":"error"}]}`,
		"in the choice": `{"choices":[{"index":0,"delta":{"content":""},"finish_reason":"error","error":` + failure + `}]}`,
	} {
		err := Events("kilo", strings.NewReader("data: "+chunk+"\n\n"), func(llmprovider.Event) bool { return true })
		apiErr, ok := errors.AsType[*llmprovider.APIError](err)
		if !ok || !errors.Is(err, llmprovider.ErrRateLimited) || apiErr.RetryAfter != 7*time.Second {
			t.Errorf("%s: %v; want ErrRateLimited with RetryAfter 7s", name, err)
		}
	}
}
