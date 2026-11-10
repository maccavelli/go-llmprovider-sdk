package llmprovider_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// ExampleAPIError_usageLimit reads which limit a refused request hit, and
// when it lifts, from an *APIError (0072-MADR D1, D2). The reply is a
// ChatGPT usage limit, built in memory.
func ExampleAPIError_usageLimit() {
	body := `{"error":{"type":"usage_limit_reached","message":"The usage limit has been reached",` +
		`"plan_type":"plus","resets_at":1893456000,"limit_window_minutes":300}}`
	err := llmprovider.ClassifyHTTPError("openai", &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(body)),
	})
	var apiErr *llmprovider.APIError
	if !errors.As(err, &apiErr) || apiErr.Limit == nil {
		fmt.Println("no limit stated")
		return
	}
	fmt.Println("quota exhausted:", errors.Is(err, llmprovider.ErrQuotaExhausted))
	fmt.Println("plan:", apiErr.Limit.Plan)
	fmt.Println("window:", apiErr.Limit.Window)
	fmt.Println("lifts at:", apiErr.ResetAt.Format(time.RFC3339))
	fmt.Println("retryable:", apiErr.Retryable())
	// Output:
	// quota exhausted: true
	// plan: plus
	// window: 5h0m0s
	// lifts at: 2030-01-01T00:00:00Z
	// retryable: false
}
