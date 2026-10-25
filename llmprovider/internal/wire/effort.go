package wire

import (
	"fmt"
	"slices"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// belowMax is the efforts below EffortMax, lowest first.
var belowMax = []llmprovider.Effort{llmprovider.EffortLow, llmprovider.EffortMedium, llmprovider.EffortHigh,
	llmprovider.EffortXHigh}

// ResolveMax is the effort a request at EffortMax is sent at, by the efforts
// the model's facts list, known when the facts state them (0061-MADR D3):
// EffortMax when they list it (rule 1), and when they are not known or list
// none of low, medium, high and xhigh (rule 3); otherwise the highest of
// those they list, and degraded (rule 2).
func ResolveMax(efforts []llmprovider.Effort, known bool) (effort llmprovider.Effort, degraded bool) {
	if !known || slices.Contains(efforts, llmprovider.EffortMax) {
		return llmprovider.EffortMax, false
	}
	for _, e := range slices.Backward(belowMax) {
		if slices.Contains(efforts, e) {
			return e, true
		}
	}
	return llmprovider.EffortMax, false
}

// EffortWarning is the warning that a request at EffortMax on model was sent
// at sent (0061-MADR D3).
func EffortWarning(model string, sent llmprovider.Effort) llmprovider.Warning {
	return llmprovider.Warning{Kind: llmprovider.WarningEffortDegraded, Model: model,
		Detail: fmt.Sprintf("reasoning effort max is not offered for %s; sent %s", model, sent)}
}

// MaxEffort is effort as sent to model: effort itself unless it is
// EffortMax, which ResolveMax resolves by the model's efforts, with an
// EffortWarning when it degraded.
func MaxEffort(model string, effort llmprovider.Effort, efforts []llmprovider.Effort, known bool) (llmprovider.Effort,
	[]llmprovider.Warning) {
	if effort != llmprovider.EffortMax {
		return effort, nil
	}
	sent, degraded := ResolveMax(efforts, known)
	if !degraded {
		return sent, nil
	}
	return sent, []llmprovider.Warning{EffortWarning(model, sent)}
}
