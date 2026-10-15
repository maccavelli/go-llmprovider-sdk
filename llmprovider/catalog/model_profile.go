package catalog

import (
	"math"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Profile selects how a listing ranks its recommended models (MADR 0009
// §1; 0041-MADR D4, D5).
type Profile int

const (
	// ProfileUtility (the zero value) ranks for short, frequent tasks such as
	// commit messages: reasoning-capable, recent, paid, cheap.
	ProfileUtility Profile = iota
	// ProfileCapable ranks for reasoning-heavy tasks: strongest first.
	ProfileCapable
	// ProfileAgent ranks for long agent sessions with prompt caching on: its
	// band holds ProfileCapable's members, the strongest six, ordered within
	// a release quarter by the cheaper cached turn (0041-MADR D5). Its
	// eligibility is ProfileCapable's. WithWorkload sets the turn's shares.
	ProfileAgent
)

// ReasoningEffort is the recommended request effort for the profile: "low"
// for ProfileUtility, and "" for ProfileCapable and ProfileAgent, meaning
// each provider's documented default (see config.ReasoningEffort; MADR 0013
// Q2). A value outside the three profiles is treated as ProfileUtility.
func (p Profile) ReasoningEffort() string {
	if p.strongest() {
		return ""
	}
	return effortLow
}

// strongest reports a profile that ranks the strongest models: ProfileCapable
// and ProfileAgent, which share eligibility and members.
func (p Profile) strongest() bool { return p == ProfileCapable || p == ProfileAgent }

// workload is an agent turn's shape: the share of its input read from the
// cache, and the share of its tokens that are output (0041-MADR D5).
type workload struct{ cacheShare, outputShare float64 }

const (
	// agentCacheShare is the share of an agent turn's input read from the
	// cache: OpenRouter's September 2026 agent figure, "just under 90%", and
	// Requesty's 89% for OpenCode (0041-MADR §7).
	agentCacheShare = 0.90
	// agentOutputShare is the share of an agent turn's tokens that are
	// output: OpenRouter's "around 1%" (0041-MADR §7).
	agentOutputShare = 0.01
)

// defaultWorkload is the published agent workload, with which §7's
// cross-check reproduces OpenRouter's own Sonnet 5 prices.
var defaultWorkload = workload{cacheShare: agentCacheShare, outputShare: agentOutputShare}

// turnPrices are the prices one cached turn reads, per million tokens. known
// is a known input and output price; read and write are known apart.
type turnPrices struct {
	input, output, read, write float64
	known, readKnown           bool
	writeKnown                 bool
}

// turn is one cached turn's price, per million tokens (0041-MADR D5):
//
//	(1 − o) × (h × read + (1 − h) × write) + o × output
//
// read is the cache read price, else the input price: no discount is
// assumed. write is the cache write price, else the input price. It is false
// when the input or output price is unknown.
func (w workload) turn(p turnPrices) (float64, bool) {
	if !p.known {
		return 0, false
	}
	read, write := p.input, p.input
	if p.readKnown {
		read = p.read
	}
	if p.writeKnown {
		write = p.write
	}
	h, o := w.cacheShare, w.outputShare
	return (1-o)*(h*read+(1-h)*write) + o*p.output, true
}

// share reports whether v is a share WithWorkload takes: within (0, 1).
func share(v float64) bool { return v > 0 && v < 1 && !math.IsNaN(v) }

// ObservedWorkload measures a caller's own workload from its responses'
// usage, for WithWorkload: the cache share is the cached input tokens over
// all input tokens, and the output share the output tokens over all tokens.
// ok is false when there is nothing to measure, or when a share is 0 or 1,
// which WithWorkload refuses. The SDK keeps no state of its own.
func ObservedWorkload(usages ...llmprovider.Usage) (cacheShare, outputShare float64, ok bool) {
	var input, output, cached int
	for _, u := range usages {
		input += u.InputTokens
		output += u.OutputTokens
		cached += u.CachedTokens
	}
	if input == 0 {
		return 0, 0, false
	}
	cacheShare = float64(cached) / float64(input)
	outputShare = float64(output) / float64(input+output)
	if !share(cacheShare) || !share(outputShare) {
		return 0, 0, false
	}
	return cacheShare, outputShare, true
}
