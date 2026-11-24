package llmprovider

import (
	"fmt"
	"slices"
	"time"
)

// LimitObservation is an output cap a service stated when it refused an
// output limit, validated as 0089-MADR D4 says (D5). A provider that keeps
// one reports it as a WarningLimitLearned's Observed, or, on a call that
// fails, as its *APIError's Observed (DL78); a caller that keeps it passes it
// back with WithLimitObservations on its next start. It is a value, safe for
// concurrent use while no goroutine changes it.
type LimitObservation struct {
	// Model is the model refused, as the request named it.
	Model string
	// BaseURL is the provider's base URL, as WithBaseURL gave it; empty for
	// the provider's default.
	BaseURL string
	// Scope is the Kilo organization or the Claude workspace the request was
	// sent in; empty for none.
	Scope string
	// Tokens is the cap the service stated: the most the output-limit field
	// may carry.
	Tokens int
	// Seen is when the refusal was read.
	Seen time.Time
}

// WithLimitObservations passes back observations that WarningLimitLearned
// or APIError.Observed reported (0089-MADR D5, DL89). catalog.LookupLimits and catalog.CachedLimits read
// each as the evidence "remembered", only where its model (matched in lower
// case), base URL and scope are the lookup's, and only while its Seen is
// within 24 hours; it is ranked below every caller, service and registry
// source, so it is chosen only where none of them states the output limit,
// and otherwise kept as evidence beside the stated source. A provider's
// default base URL is read as empty, in the observation and in the lookup
// alike. A provider given it reads the same caps for each request, through
// catalog.CachedLimits: on a route that sends a limit the SDK chooses
// (Claude, OpenCode's messages route, Kilo) a remembered cap is sent where
// no other source states the model's output limit; on a route where the
// limit is optional it only cuts a given WithMaxTokens (0089-MADR D6). A
// later call adds to an earlier one's. It is a shared option
// (SharedOption): every provider's New takes it. New refuses, with
// ErrInvalidRequest, an observation with no model, Tokens below 1, or a
// zero Seen.
func WithLimitObservations(obs ...LimitObservation) Option {
	return SharedOption("WithLimitObservations", slices.Clone(obs))
}

// checkLimitObservations refuses a passed-back observation no lookup or
// request could use (R23): one with no model, Tokens below 1, or a zero
// Seen, among the shared values.
func checkLimitObservations(values []any) error {
	for _, v := range values {
		obs, ok := v.([]LimitObservation)
		if !ok {
			continue
		}
		for i, o := range obs {
			switch {
			case o.Model == "":
				return fmt.Errorf("%w: WithLimitObservations: observation %d names no model", ErrInvalidRequest, i)
			case o.Tokens < 1:
				return fmt.Errorf("%w: WithLimitObservations: observation %d (%s): Tokens %d; a cap is at least 1",
					ErrInvalidRequest, i, o.Model, o.Tokens)
			case o.Seen.IsZero():
				return fmt.Errorf("%w: WithLimitObservations: observation %d (%s) has no Seen time", ErrInvalidRequest, i, o.Model)
			}
		}
	}
	return nil
}
