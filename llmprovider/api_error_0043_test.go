package llmprovider

import (
	"errors"
	"testing"
)

// TestPlanCodes_0043: a ChatGPT plan session's subscription_sharing_* codes
// mean the same in a reply and inside a stream (0043-MADR D6, A4), and none
// is retried.
func TestPlanCodes_0043(t *testing.T) {
	for _, c := range []struct {
		code   string
		status int
		want   error
	}{
		{"subscription_sharing_usage_limit_exceeded", 429, ErrQuotaExhausted},
		{"subscription_sharing_user_not_eligible", 403, ErrNotPermitted},
		{"subscription_sharing_route_not_supported", 403, ErrNotPermitted},
		{"subscription_sharing_unsupported_capability", 400, ErrInvalidRequest},
	} {
		body := `{"error":{"code":"` + c.code + `","message":"refused","param":"tools"}}`
		reply := classifyFixture(string(ProviderOpenAI), c.status, body, nil)
		stream := ClassifyStreamFailure(string(ProviderOpenAI), c.code, "", "refused")
		for where, err := range map[string]error{"reply": reply, "stream": stream} {
			if !errors.Is(err, c.want) {
				t.Errorf("%s in a %s: %v; want %v", c.code, where, err, c.want)
			}
			if apiErr, ok := errors.AsType[*APIError](err); !ok || apiErr.Retryable() {
				t.Errorf("%s in a %s: %#v; want a terminal *APIError", c.code, where, err)
			}
		}
	}
}

// TestPlanCodes_OnlyOpenAI_0043: another service's stream with the same
// code is classified as before, retryable.
func TestPlanCodes_OnlyOpenAI_0043(t *testing.T) {
	err := ClassifyStreamFailure("together", "subscription_sharing_user_not_eligible", "", "refused")
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Errorf("together: %v; want ErrProviderUnavailable, as any unknown code", err)
	}
}
