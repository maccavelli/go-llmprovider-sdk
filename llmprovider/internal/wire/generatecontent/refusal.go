package generatecontent

import (
	"errors"
	"net/http"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// MatchRefusal is generateContent's wire.Matcher (0062-MADR D4, amendment
// "Phase 3's google-route levels, and 0013's B9"): HTTP 400 INVALID_ARGUMENT
// whose message names an invalid value at
// 'generation_config.thinking_config.thinking_level', as OpenCode's google
// route refuses xhigh: the level sent, with no list. A refusal of the
// contents or of a tool's schema shares the status and the code, and is told
// apart by the field it names.
func MatchRefusal(err error, sent llmprovider.Effort) (wire.Refusal, bool) {
	var apiErr *llmprovider.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusBadRequest || apiErr.Code != "INVALID_ARGUMENT" || sent == "" ||
		!strings.Contains(apiErr.Message, "Invalid value at 'generation_config.thinking_config.thinking_level'") {
		return wire.Refusal{}, false
	}
	return wire.Refusal{Effort: sent}, true
}
