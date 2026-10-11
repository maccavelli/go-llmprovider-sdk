package messages

import (
	"errors"
	"iter"
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

// SendLearning is send, which builds and sends a request with choice to f's
// model. When it is refused for forcing a tool, the model is learned
// and send is called once more, building the degraded request (0034-MADR D3).
func SendLearning[T any](f Forcing, choice llmprovider.ToolChoice, send func() (T, error)) (T, error) {
	v, err := send()
	if !shouldLearn(f, choice, err) {
		return v, err
	}
	learnRefusal(f.Provider, f.Model)
	return send()
}

// StreamLearning is SendLearning for a stream: a stream refused for forcing a
// tool before its first event is opened once more, degraded.
func StreamLearning(f Forcing, choice llmprovider.ToolChoice,
	open func() iter.Seq2[llmprovider.Event, error]) iter.Seq2[llmprovider.Event, error] {
	return func(yield func(llmprovider.Event, error) bool) {
		first := true
		for ev, err := range open() {
			if first && shouldLearn(f, choice, err) {
				learnRefusal(f.Provider, f.Model)
				for ev, err := range open() {
					if !yield(ev, err) {
						return
					}
				}
				return
			}
			first = false
			if !yield(ev, err) {
				return
			}
		}
	}
}
