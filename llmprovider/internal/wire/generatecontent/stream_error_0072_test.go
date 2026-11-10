package generatecontent

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// perDayError is Google's 429 error object for a per-day quota, in its
// documented shape (0026-MADR F12): synthetic, not captured.
const perDayError = `{"code":429,"message":"You exceeded your current quota.","status":"RESOURCE_EXHAUSTED","details":[` +
	`{"@type":"type.googleapis.com/google.rpc.QuotaFailure","violations":[{"quotaId":"GenerateRequestsPerDayPerProjectPerModel-FreeTier","quotaValue":"5"}]},` +
	`{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"52s"}]}`

// TestEvents_PerDayQuota_0072 (0072-MADR D3; 0026-MADR F12): a per-day
// quota inside a stream is ErrQuotaExhausted, with RetryInfo's delay, as in
// an HTTP reply. Only the error's details carry either.
func TestEvents_PerDayQuota_0072(t *testing.T) {
	err := Events("gemini", strings.NewReader(sse(`{"error":`+perDayError+`}`)), func(llmprovider.Event) bool { return true })
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	if !ok || !errors.Is(err, llmprovider.ErrQuotaExhausted) || apiErr.RetryAfter != 52*time.Second {
		t.Errorf("%v; want ErrQuotaExhausted with RetryAfter 52s", err)
	}
}
