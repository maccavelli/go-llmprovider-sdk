package messages

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// learnedRefusalsLimit bounds the memo of learned refusals (0034-MADR D3).
const learnedRefusalsLimit = 256

// learned holds the models this process learned refuse a forced tool choice,
// by provider and model, oldest first. It is never written to disk, and a
// restart clears it (0034-MADR D3).
var learned = struct {
	mu    sync.Mutex
	order []string
	set   map[string]struct{}
}{set: map[string]struct{}{}}

// refusalKey is the memo's key for provider and model, normalised.
func refusalKey(provider, model string) string { return provider + "\x00" + normalizeModel(model) }

// learnRefusal remembers that model refuses a forced choice on provider,
// dropping the oldest entry past learnedRefusalsLimit.
func learnRefusal(provider, model string) {
	key := refusalKey(provider, model)
	learned.mu.Lock()
	defer learned.mu.Unlock()
	if _, ok := learned.set[key]; ok {
		return
	}
	if len(learned.order) >= learnedRefusalsLimit {
		delete(learned.set, learned.order[0])
		learned.order = learned.order[1:]
	}
	learned.order = append(learned.order, key)
	learned.set[key] = struct{}{}
}

// learnedRefusal reports whether this process learned that model refuses a
// forced choice on provider.
func learnedRefusal(provider, model string) bool {
	learned.mu.Lock()
	defer learned.mu.Unlock()
	_, ok := learned.set[refusalKey(provider, model)]
	return ok
}

// IsForcedChoiceRefusal reports whether err is a refusal of a forced tool
// choice: HTTP 400, invalid_request_error, and a message naming tool_choice,
// as Anthropic answers one (0034-PLAN Phase 1, M2). The API gives no other
// signal; the match only decides whether to resend (0034-MADR D3).
func IsForcedChoiceRefusal(err error) bool {
	var apiErr *llmprovider.APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusBadRequest &&
		apiErr.Code == "invalid_request_error" && strings.Contains(apiErr.Message, "tool_choice")
}

// forcedChoice reports whether choice forces a tool.
func forcedChoice(choice llmprovider.ToolChoice) bool {
	_, named := choice.Tool()
	return named || choice == llmprovider.ToolChoiceRequired
}

// shouldLearn reports whether err, the reply to a request with choice, teaches
// a refusal of f's model: the request forced a tool, no layer said the model
// refuses, the caller said nothing of it, and err is the refusal. A caller's
// override is never second-guessed (0034-MADR D9).
func shouldLearn(f Forcing, choice llmprovider.ToolChoice, err error) bool {
	_, overridden := f.override()
	return forcedChoice(choice) && !overridden && !f.refuses() && IsForcedChoiceRefusal(err)
}

// RefusalRetry is the retry check for one request with a forced choice to
// f's model (0034-MADR D3, narrowed). A refusal from a model no layer knows
// is learned, then:
//   - a named tool is resent, which the rebuilt request sends as structured
//     outputs (D8);
//   - ToolChoiceRequired is resent, degraded, under ForcedChoiceDegrade, and
//     otherwise returned as an error matching ErrUnsupported and wrapping the
//     *APIError (D10).
//
// Anything else is returned as it came. A RefusalRetry serves one request.
type RefusalRetry struct {
	f       Forcing
	choice  llmprovider.ToolChoice
	learned bool
}

// NewRefusalRetry is the retry check for a request with choice to f's model.
func NewRefusalRetry(f Forcing, choice llmprovider.ToolChoice) *RefusalRetry {
	return &RefusalRetry{f: f, choice: choice}
}

// Check is a wire.Retry.
func (r *RefusalRetry) Check(err error) (bool, error) {
	if !shouldLearn(r.f, r.choice, err) {
		return false, err
	}
	learnRefusal(r.f.Provider, r.f.Model)
	r.learned = true
	if _, named := r.choice.Tool(); named || r.f.Policy.fallback == llmprovider.ForcedChoiceDegrade {
		return true, nil
	}
	return false, fmt.Errorf("%w: %s refused a forced tool choice (ToolChoiceRequired), and is remembered as refusing it; "+
		"WithForcedChoiceFallback(ForcedChoiceDegrade) sends it as auto, with an instruction to call a tool (0034-MADR D3, D10): %w",
		llmprovider.ErrUnsupported, r.f.Model, err)
}

// Warning is the WarningForcedChoiceLearned, when Check learned a refusal.
func (r *RefusalRetry) Warning() (llmprovider.Warning, bool) {
	if !r.learned {
		return llmprovider.Warning{}, false
	}
	return llmprovider.Warning{Kind: llmprovider.WarningForcedChoiceLearned, Model: r.f.Model,
		Detail: r.f.Model + " refused a forced tool choice that no table or option said it would; " +
			"it is remembered for the rest of the process, and WithModelForcedChoice can say so on the next start."}, true
}
