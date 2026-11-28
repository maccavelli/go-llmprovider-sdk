package llmprovider

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestPerDayQuota_PastTheCap_0095 (0095-MADR FU13): a Gemini 429 whose
// QuotaFailure holds 33 violations, only the 33rd per-day, is
// ErrQuotaExhausted with PerDay set: every violation of a kept detail is
// read; only the list kept stays capped at 32.
func TestPerDayQuota_PastTheCap_0095(t *testing.T) {
	violations := make([]string, 0, 33)
	for i := range 32 {
		violations = append(violations, fmt.Sprintf(`{"quotaMetric":"m","quotaId":"GenerateRequestsPerMinute-%d","quotaValue":"5"}`, i))
	}
	violations = append(violations, `{"quotaMetric":"m","quotaId":"GenerateRequestsPerDayPerProjectPerModel-FreeTier","quotaValue":"5"}`)
	body := `{"error":{"code":429,"message":"You exceeded your current quota.","status":"RESOURCE_EXHAUSTED","details":[` +
		`{"@type":"type.googleapis.com/google.rpc.QuotaFailure","violations":[` + strings.Join(violations, ",") + `]}]}}`
	err := classifyFixture("gemini", 429, body, nil)
	if !errors.Is(err, ErrQuotaExhausted) {
		t.Errorf("33 violations, the 33rd per-day: %v; want ErrQuotaExhausted", err)
	}
	l := limitOfErr(t, err)
	if !l.PerDay || len(l.Quotas) != 32 {
		t.Errorf("Limit PerDay %t, %d quotas kept; want PerDay and the 32 the list is capped at", l.PerDay, len(l.Quotas))
	}
}
