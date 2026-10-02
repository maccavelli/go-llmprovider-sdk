package wizard

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestConfigureLLM_AnthropicKeyOnly is 0016-PLAN T5 (0016-MADR D12, the
// owner's decision of 2026-09-29): Claude's key is read from
// ANTHROPIC_API_KEY only. With only CLAUDE_API_KEY set no key is found, and
// the user is asked for one; with ANTHROPIC_API_KEY set it is offered.
func TestConfigureLLM_AnthropicKeyOnly(t *testing.T) {
	if got := llmprovider.ProviderEnvVars()[llmprovider.ProviderClaude]; got != "ANTHROPIC_API_KEY" {
		t.Errorf("ProviderEnvVars()[claude] = %q, want ANTHROPIC_API_KEY", got)
	}
	env := func(vals map[string]string) func(string) string { return func(k string) string { return vals[k] } }

	f := &fakePrompter{t: t, selects: []int{providerIdx(t, llmprovider.ProviderClaude), 0}, secrets: []string{"typed-key"}}
	res, err := ConfigureLLM(context.Background(), f, Options{AllowEnv: true,
		LookupEnv: env(map[string]string{"CLAUDE_API_KEY": testKey})})
	if err != nil || res.APIKey != "typed-key" || len(f.seenConfirm) != 0 {
		t.Errorf("CLAUDE_API_KEY only: key %q, confirms %q, err %v; want no key offered and the typed one used",
			res.APIKey, f.seenConfirm, err)
	}

	f = &fakePrompter{t: t, selects: []int{providerIdx(t, llmprovider.ProviderClaude), 0}, confirms: []bool{true}}
	res, err = ConfigureLLM(context.Background(), f, Options{AllowEnv: true,
		LookupEnv: env(map[string]string{"ANTHROPIC_API_KEY": testKey})})
	if err != nil || res.APIKey != testKey || len(f.seenSecret) != 0 {
		t.Errorf("ANTHROPIC_API_KEY: key %q, secrets %q, err %v; want the environment's key", res.APIKey, f.seenSecret, err)
	}
}
