package gemini

import (
	"errors"
	"net/http"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// matchRefusal is the Interactions wire's wire.Matcher (0062-MADR D4,
// amendment "Phase 1's evidence"), on HTTP 400 invalid_request, of the level
// sent:
//   - "The value '<sent>' is not supported for
//     'generation_config.thinking_level'. Supported values: …", with the list;
//   - "'<sent>' is not a supported thinking level for this model. Allowed
//     values are: …", with the list;
//   - "The thinking budget <n> is invalid. …", with none: the wire sends no
//     budget, so the budget is the level's (Q13 (a)).
//
// A 404 is never read: an empty input is answered 404 "Model … not found".
func matchRefusal(err error, sent llmprovider.Effort) (wire.Refusal, bool) {
	var apiErr *llmprovider.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusBadRequest || apiErr.Code != "invalid_request" || sent == "" {
		return wire.Refusal{}, false
	}
	msg := apiErr.Message
	quoted := "'" + string(sent) + "'"
	if list, ok := strings.CutPrefix(msg, "The value "+quoted+" is not supported for 'generation_config.thinking_level'. Supported values: "); ok {
		accepted, listed := wire.QuotedEfforts(list)
		return wire.Refusal{Effort: sent, Accepted: accepted, Listed: listed}, true
	}
	if list, ok := strings.CutPrefix(msg, quoted+" is not a supported thinking level for this model. Allowed values are: "); ok {
		accepted, listed := wire.PlainEfforts(list)
		return wire.Refusal{Effort: sent, Accepted: accepted, Listed: listed}, true
	}
	if strings.HasPrefix(msg, "The thinking budget ") && strings.Contains(msg, " is invalid") {
		return wire.Refusal{Effort: sent}, true
	}
	return wire.Refusal{}, false
}
