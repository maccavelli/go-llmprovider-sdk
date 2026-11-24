package llmprovider

import "github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/limitmemo"

// SentLimit is what a request carried as its output limit, and why
// (0089-MADR D7): a Response's OutputLimit, and a stream's final Response's.
// It is a value, safe for concurrent use. A provider that does not report
// one leaves it zero.
type SentLimit struct {
	// Tokens is the value the output-limit field carried on the wire, after
	// a thinking budget raised it (0089-MADR DL55); 0 when Sent is false.
	Tokens int
	// Sent reports whether the request carried an output limit at all.
	Sent bool
	// Basis is why it carried that value, or none.
	Basis LimitBasis
	// Source names the evidence the value came from: a catalog.Evidence
	// source name for LimitBasisFact and LimitBasisFallback, or the refusal
	// form's name for LimitBasisRefusal; empty for a caller's value and for
	// none.
	Source string
}

// LimitBasis is why a request carried its output limit (0089-MADR D7). It is
// a closed set (R21): the six constants below. It is a value, safe for
// concurrent use.
type LimitBasis string

const (
	// LimitBasisRequest is Request.MaxOutputTokens, sent as given.
	LimitBasisRequest LimitBasis = "request"
	// LimitBasisOption is WithMaxTokens' value, cut to the model's stated
	// limit and never raised, or a listing probe's (0088-MADR ST15, ST26).
	LimitBasisOption LimitBasis = "option"
	// LimitBasisFact is the model's chosen output fact (catalog.Limits),
	// held to its window and to a cap the service stated.
	LimitBasisFact LimitBasis = "fact"
	// LimitBasisRefusal is a resend at the cap, or the room, a validated
	// refusal stated (0089-MADR D4, D6).
	LimitBasisRefusal LimitBasis = "refusal"
	// LimitBasisFallback is the smallest limit any current model on the
	// route is stated to take, sent where the wire requires a limit and no
	// source states the model's (0089-MADR DL51).
	LimitBasisFallback LimitBasis = "fallback"
	// LimitBasisNone is nothing sent: the service's own default applies.
	LimitBasisNone LimitBasis = "none"
)

// WithoutLimitLearning turns off the memo and the report of output caps
// (0089-MADR D5, DL73): a provider given it keeps no cap a refusal stated
// and reports no WarningLimitLearned, and still resends once at a validated
// stated cap (DL49). Without it, both run. It is a shared option
// (SharedOption): every provider's New takes it, and a provider that keeps
// no caps ignores it.
func WithoutLimitLearning() Option {
	return SharedOption("WithoutLimitLearning", limitmemo.Off{})
}
