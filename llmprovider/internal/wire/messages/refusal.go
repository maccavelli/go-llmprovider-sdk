package messages

import (
	"errors"
	"net/http"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// MatchRefusal is the Messages wire's wire.Matcher (0062-MADR D4, amendment
// "Phase 1's evidence"): HTTP 400 invalid_request_error, whose message
// refuses the budget shape ("thinking.type.enabled" is not supported), the
// adaptive shape (adaptive thinking is not supported), or the effort sent
// (output_config.effort: Input should be, then the accepted efforts). A
// refusal of a tool's schema shares the code and "Input should be", and is
// told apart by the field it names.
func MatchRefusal(err error, sent llmprovider.Effort) (wire.Refusal, bool) {
	var apiErr *llmprovider.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusBadRequest || apiErr.Code != "invalid_request_error" {
		return wire.Refusal{}, false
	}
	msg := apiErr.Message
	switch {
	case strings.Contains(msg, `"thinking.type.enabled" is not supported`):
		return wire.Refusal{Shape: llmprovider.ThinkingBudget}, true
	case strings.Contains(msg, "adaptive thinking is not supported"):
		return wire.Refusal{Shape: llmprovider.ThinkingAdaptive}, true
	}
	list, ok := strings.CutPrefix(msg, "output_config.effort: Input should be ")
	if !ok || sent == "" {
		return wire.Refusal{}, false
	}
	accepted, listed := wire.QuotedEfforts(list)
	return wire.Refusal{Effort: sent, Accepted: accepted, Listed: listed}, true
}
