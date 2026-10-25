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
