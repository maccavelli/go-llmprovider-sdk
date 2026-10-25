package responses

import (
	"errors"
	"net/http"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// MatchRefusal is the Responses wire's wire.Matcher (0062-MADR D4, amendment
// "Phase 1's evidence"), on HTTP 400:
//   - invalid_value, "Invalid value: '<sent>'. Supported values are: …", every
//     listed word an effort word: the effort sent, with the list;
//   - unsupported_value, "Unsupported value: '<sent>' is not supported with
//     the '<model>' model. Supported values are: …": the same;
//   - unsupported_parameter naming 'reasoning.effort': the model takes no
//     effort;
//   - xAI's invalid-argument "Invalid reasoning effort.": the effort sent,
//     with no list;
//   - xAI's invalid-argument "Model <model> does not support parameter
//     reasoningEffort.": the model takes no effort (0062-PLAN Phase 2, the
//     Grok cases of 2026-10-25).
//
// The value sent and a list made only of effort words stand in for the
// body's param, which the *APIError does not keep (F6).
func MatchRefusal(err error, sent llmprovider.Effort) (wire.Refusal, bool) {
	var apiErr *llmprovider.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusBadRequest || sent == "" {
		return wire.Refusal{}, false
	}
	msg := apiErr.Message
	switch apiErr.Code {
	case "invalid_value":
		if list, ok := strings.CutPrefix(msg, "Invalid value: '"+string(sent)+"'. Supported values are: "); ok {
			return listed(sent, list)
		}
	case "unsupported_value":
		if rest, ok := strings.CutPrefix(msg, "Unsupported value: '"+string(sent)+"' is not supported with the '"); ok {
			if _, list, ok := strings.Cut(rest, "' model. Supported values are: "); ok {
				return listed(sent, list)
			}
		}
	case "unsupported_parameter":
		if strings.Contains(msg, "'reasoning.effort' is not supported") {
			return wire.Refusal{Field: true}, true
		}
	case "invalid-argument":
		switch {
		case msg == "Invalid reasoning effort.":
			return wire.Refusal{Effort: sent}, true
		case strings.HasPrefix(msg, "Model ") && strings.HasSuffix(msg, " does not support parameter reasoningEffort."):
			return wire.Refusal{Field: true}, true
		}
	}
	return wire.Refusal{}, false
}

// listed is a refusal of sent whose list must be made of effort words only.
func listed(sent llmprovider.Effort, list string) (wire.Refusal, bool) {
	accepted, ok := wire.QuotedEfforts(list)
	if !ok {
		return wire.Refusal{}, false
	}
	return wire.Refusal{Effort: sent, Accepted: accepted, Listed: true}, true
}
