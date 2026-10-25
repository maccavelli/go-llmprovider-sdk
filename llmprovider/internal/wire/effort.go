package wire

import (
	"fmt"
	"slices"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// effortOrder is 0062-MADR D3's order of the efforts, lowest first.
var effortOrder = []llmprovider.Effort{llmprovider.EffortLow, llmprovider.EffortMedium, llmprovider.EffortHigh,
	llmprovider.EffortXHigh, llmprovider.EffortMax}

// Ladder is the efforts a request for want may be sent at, in the order they
// are tried (0062-MADR D3): want itself, then each lower effort, highest
// first, then each higher one, lowest first. An effort the facts do not list,
// when they are known, and an effort refused are left out. The first is
// sent; none left means the effort is left out of the request. An unknown
// want has none.
func Ladder(want llmprovider.Effort, listed []llmprovider.Effort, known bool, refused []llmprovider.Effort) []llmprovider.Effort {
	at := slices.Index(effortOrder, want)
	if at < 0 {
		return nil
	}
	rungs := []llmprovider.Effort{want}
	for i := at - 1; i >= 0; i-- {
		rungs = append(rungs, effortOrder[i])
	}
	rungs = append(rungs, effortOrder[at+1:]...)
	return slices.DeleteFunc(rungs, func(e llmprovider.Effort) bool {
		return (known && !slices.Contains(listed, e)) || slices.Contains(refused, e)
	})
}

// DegradedWarning is the warning that a request for want on model was sent
// at sent, "" being none (0061-MADR D3; 0062-MADR D3).
func DegradedWarning(model string, want, sent llmprovider.Effort) llmprovider.Warning {
	shown := string(sent)
	if shown == "" {
		shown = "none"
	}
	return llmprovider.Warning{Kind: llmprovider.WarningEffortDegraded, Model: model,
		Detail: fmt.Sprintf("reasoning effort %s is not offered for %s; sent %s", want, model, shown)}
}

// belowMax is the efforts below EffortMax, lowest first.
var belowMax = effortOrder[:len(effortOrder)-1]

// ResolveMax is the effort a request at EffortMax is sent at, by the efforts
// the model's facts list, known when the facts state them (0061-MADR D3):
// EffortMax when they list it (rule 1), and when they are not known or list
// none of low, medium, high and xhigh (rule 3); otherwise the highest of
// those they list, and degraded (rule 2). Only Ollama decides by it, until
// 0062-PLAN's Phase 5 records Ollama's refusals (0062-MADR Q9 (c)); every
// other provider decides by Ladder.
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
	return DegradedWarning(model, llmprovider.EffortMax, sent)
}

// MaxEffort is effort as sent to model: effort itself unless it is
// EffortMax, which ResolveMax resolves by the model's efforts, with an
// EffortWarning when it degraded. Only Ollama decides by it (ResolveMax).
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
