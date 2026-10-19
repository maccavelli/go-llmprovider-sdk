package llmprovider

import (
	"fmt"
	"maps"
	"strings"
)

// ForcedChoiceFallback is what a provider does with ToolChoiceRequired for a
// model that refuses a forced tool choice (0034-MADR D10). A forced named tool
// (ForceTool) is not affected: it is sent in a form the model accepts. It is
// an integer, safe for concurrent use.
type ForcedChoiceFallback int

const (
	// ForcedChoiceRefuse fails the request before any network call, with an
	// error matching ErrUnsupported (R11). It is the default: nothing is sent
	// other than as asked.
	ForcedChoiceRefuse ForcedChoiceFallback = iota
	// ForcedChoiceDegrade sends the choice as "auto", with an instruction to
	// call a tool, and reports it as a WarningForcedChoiceDegraded. The model
	// may answer without the call.
	ForcedChoiceDegrade
)

// String returns the constant's name.
func (f ForcedChoiceFallback) String() string {
	switch f {
	case ForcedChoiceRefuse:
		return "ForcedChoiceRefuse"
	case ForcedChoiceDegrade:
		return "ForcedChoiceDegrade"
	default:
		return "ForcedChoiceFallback(invalid)"
	}
}

// WithForcedChoiceFallback sets what a provider does with ToolChoiceRequired
// for a model that refuses a forced tool choice. Without it, the request fails
// before the network (ForcedChoiceRefuse).
func WithForcedChoiceFallback(f ForcedChoiceFallback) Option {
	return commonOption("WithForcedChoiceFallback", func(cfg *providerConfig) {
		cfg.ForcedChoiceFallback = f
	})
}

// WithModelForcedChoice says whether model accepts a forced tool choice:
// Supported if it does, Unsupported if it refuses one. It overrides the
// provider's own knowledge, and a refusal it learned, either way (0034-MADR
// D9). The model is matched in lower case, after the gateway or platform
// prefixes and snapshot dates the provider strips. Use it to seed a refusal a
// WarningForcedChoiceLearned reported, or to correct the table before a
// release. New refuses an empty model or another Support, with
// ErrInvalidRequest.
func WithModelForcedChoice(model string, s Support) Option {
	return commonOption("WithModelForcedChoice", func(cfg *providerConfig) {
		if cfg.ModelForcedChoice == nil {
			cfg.ModelForcedChoice = map[string]Support{}
		}
		cfg.ModelForcedChoice[strings.ToLower(model)] = s
	})
}

// ForcedChoiceFallback is the fallback from WithForcedChoiceFallback, or
// ForcedChoiceRefuse.
func (st *Settings) ForcedChoiceFallback() ForcedChoiceFallback { return st.s.cfg.ForcedChoiceFallback }

// ModelForcedChoices is a copy of the per-model facts WithModelForcedChoice
// set, keyed by lower-case model, or nil.
func (st *Settings) ModelForcedChoices() map[string]Support {
	return maps.Clone(st.s.cfg.ModelForcedChoice)
}

// checkForcedChoice refuses a forced-choice setting no request could use
// (R23).
func (cfg providerConfig) checkForcedChoice() error {
	if f := cfg.ForcedChoiceFallback; f != ForcedChoiceRefuse && f != ForcedChoiceDegrade {
		return fmt.Errorf("%w: WithForcedChoiceFallback(%d): not a ForcedChoiceFallback", ErrInvalidRequest, int(f))
	}
	for model, s := range cfg.ModelForcedChoice {
		switch {
		case model == "":
			return fmt.Errorf("%w: WithModelForcedChoice needs a model", ErrInvalidRequest)
		case hasControl(model):
			return fmt.Errorf("%w: WithModelForcedChoice's model holds a control character", ErrInvalidRequest)
		case s != Supported && s != Unsupported:
			return fmt.Errorf("%w: WithModelForcedChoice(%q, %v): a model accepts a forced choice (Supported) or refuses one (Unsupported)",
				ErrInvalidRequest, model, s)
		}
	}
	return nil
}
