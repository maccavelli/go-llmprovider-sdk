package gemini

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestInteractionEvents_PerDayQuota_0072 (0072-MADR D3; 0026-MADR F12): a
// per-day quota in an Interactions stream is ErrQuotaExhausted, with
// RetryInfo's delay. Only the error's details carry either. The error object
// is Google's documented shape: synthetic, not captured.
func TestInteractionEvents_PerDayQuota_0072(t *testing.T) {
	const perDay = `{"code":429,"message":"You exceeded your current quota.","status":"RESOURCE_EXHAUSTED","details":[` +
		`{"@type":"type.googleapis.com/google.rpc.QuotaFailure","violations":[{"quotaId":"GenerateRequestsPerDayPerProjectPerModel-FreeTier","quotaValue":"5"}]},` +
		`{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"52s"}]}`
	stream := "data: {\"event_type\":\"error\",\"error\":" + perDay + "}\n\n"
	err := interactionEvents("gemini", strings.NewReader(stream), func(llmprovider.Event) bool { return true })
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	if !ok || !errors.Is(err, llmprovider.ErrQuotaExhausted) || apiErr.RetryAfter != 52*time.Second {
		t.Errorf("%v; want ErrQuotaExhausted with RetryAfter 52s", err)
	}
}
