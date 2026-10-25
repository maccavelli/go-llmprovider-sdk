package chatcompletions

import (
	"errors"
	"net/http"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// MatchRefusal is the Chat Completions wire's wire.Matcher (0062-MADR D4,
// amendment "Phase 1's evidence"), on HTTP 400:
//   - Hugging Face's "reasoning_effort (<sent>): Input should be …": the
//     effort sent, with the list;
//   - Kilo's invalid_request_error "Invalid option: expected one of …",
//     every listed word an effort word: the same. Kilo refuses an empty
//     input with the same code ("At least one message is required"), which
//     the message tells apart.
func MatchRefusal(err error, sent llmprovider.Effort) (wire.Refusal, bool) {
	var apiErr *llmprovider.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusBadRequest || sent == "" {
		return wire.Refusal{}, false
	}
	msg := apiErr.Message
	if _, list, ok := strings.Cut(msg, "reasoning_effort ("+string(sent)+"): Input should be "); ok {
		accepted, listed := wire.QuotedEfforts(list)
		return wire.Refusal{Effort: sent, Accepted: accepted, Listed: listed}, true
	}
	if list, ok := strings.CutPrefix(msg, "Invalid option: expected one of "); ok && apiErr.Code == "invalid_request_error" {
		if accepted, ok := wire.QuotedEfforts(list); ok {
			return wire.Refusal{Effort: sent, Accepted: accepted, Listed: true}, true
		}
	}
	return wire.Refusal{}, false
}
