package messages

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

var (
	// bedrockVersion is Bedrock's model version suffix, such as -v1:0.
	bedrockVersion = regexp.MustCompile(`-v\d+(?::\d+)?$`)
	// snapshotDate is a snapshot's date suffix: -20260928 or @20260928.
	snapshotDate = regexp.MustCompile(`[-@]\d{8}$`)
)

// normalizeModel is the id a model's forced-choice fact is kept under: lower
// case, without a gateway's anthropic/ prefix, Bedrock's anthropic. prefix and
// version suffix, or a snapshot date (0034-MADR D9). Matching stays exact
// after it: no version pattern.
func normalizeModel(id string) string {
	s := strings.ToLower(strings.TrimSpace(id))
	s = strings.TrimPrefix(s, "anthropic/")
	s = strings.TrimPrefix(s, "anthropic.")
	s = bedrockVersion.ReplaceAllString(s, "")
	return snapshotDate.ReplaceAllString(s, "")
}

// Policy is a provider's forced-choice settings: its fallback, the caller's
// per-model facts, keyed by normalised id, and the memo it learns refusals
// into (0034-MADR D9, D10; 0049-MADR D3).
type Policy struct {
	fallback  llmprovider.ForcedChoiceFallback
	overrides map[string]llmprovider.Support
	refusals  *Refusals
}

// NewPolicy is the policy for WithForcedChoiceFallback's fallback and
// WithModelForcedChoice's facts, learning refusals into refusals, a nil one
// being the process's.
func NewPolicy(fallback llmprovider.ForcedChoiceFallback, overrides map[string]llmprovider.Support,
	refusals *Refusals) Policy {
	p := Policy{fallback: fallback, refusals: refusals}
	for model, s := range overrides {
		if p.overrides == nil {
			p.overrides = make(map[string]llmprovider.Support, len(overrides))
		}
		p.overrides[normalizeModel(model)] = s
	}
	return p
}

// For is the policy applied to model, through provider.
func (p Policy) For(provider, model string) Forcing {
	return Forcing{Provider: provider, Model: model, Policy: p}
}

// Forcing is one request's forced-choice context: the provider, the model,
// and the policy (0034-MADR D9, D10). Its zero Policy refuses, with no
// overrides.
type Forcing struct {
	Provider, Model string
	Policy          Policy
}

// override is the caller's fact for the model, and whether there is one.
func (f Forcing) override() (llmprovider.Support, bool) {
	s, ok := f.Policy.overrides[normalizeModel(f.Model)]
	return s, ok
}

// refuses reports whether the model refuses a forced choice, by layer: the
// caller's override, then the table, then a refusal learned this process;
// unknown accepts (0034-MADR D9).
func (f Forcing) refuses() bool {
	if s, ok := f.override(); ok {
		return s == llmprovider.Unsupported
	}
	return refusesForcedChoice(f.Policy.refusals, f.Provider, f.Model)
}

// CheckForcing refuses, before any network call, ToolChoiceRequired with tools
// for a model that refuses forcing, unless the policy degrades it: an error
// matching ErrUnsupported (0034-MADR D10; R11). A named tool passes: it is
// sent in a form the model accepts.
func CheckForcing(req *llmprovider.Request, f Forcing) error {
	if len(req.Tools) == 0 || req.ToolChoice != llmprovider.ToolChoiceRequired ||
		f.Policy.fallback == llmprovider.ForcedChoiceDegrade || !f.refuses() {
		return nil
	}
	return fmt.Errorf("%w: %s refuses a forced tool choice (ToolChoiceRequired); "+
		"WithForcedChoiceFallback(ForcedChoiceDegrade) sends it as auto, with an instruction to call a tool (0034-MADR D10)",
		llmprovider.ErrUnsupported, f.Model)
}
