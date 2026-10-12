//go:build live_gateways

package llmprovider_test

import (
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// TestLive_LookupModel_0033 (0033-PLAN-public-model-metadata.md Phase 5): for
// the first static id of each provider whose key is set, LookupModel returns a
// known context window or input limit above zero. It reads the models.dev
// document and each provider's listing; no generation is billed.
func TestLive_LookupModel_0033(t *testing.T) {
	vars := llmprovider.ProviderEnvVars()
	providers := make([]llmprovider.ProviderID, 0, len(vars))
	for p := range vars {
		providers = append(providers, p)
	}
	slices.Sort(providers)
	for _, p := range providers {
		static := catalog.Static(p)
		if len(static) == 0 {
			continue
		}
		t.Run(string(p), func(t *testing.T) {
			key := llmprovider.LiveEnvKey(t, vars[p])
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			info, err := catalog.LookupModel(ctx, p, llmprovider.NewStaticToken(key), static[0])
			llmprovider.SkipIfTransient(t, err)
			window := info.Has(catalog.FieldContextWindow) && info.ContextWindow > 0
			input := info.Has(catalog.FieldMaxInputTokens) && info.MaxInputTokens > 0
			if !window && !input {
				t.Errorf("%s %s: no window or input limit known (err %v)", p, static[0], err)
			}
			t.Logf("%s %s: window %d from %q, input %d from %q, output %d from %q, input price %.4g from %q; err %v", p, static[0],
				info.ContextWindow, info.SourceOf(catalog.FieldContextWindow), info.MaxInputTokens, info.SourceOf(catalog.FieldMaxInputTokens),
				info.MaxOutputTokens, info.SourceOf(catalog.FieldMaxOutputTokens), info.Cost.Input, info.SourceOf(catalog.FieldCostInput), err)
		})
	}
}
