package llmprovider

import "testing"

// TestProviderInterfaceSatisfaction verifies the full interface satisfaction
// matrix across every concrete provider implementation in llmprovider.
func TestProviderInterfaceSatisfaction(t *testing.T) {
	// Base Provider interface (all 4 satisfy)
	var _ LegacyProvider = (*OllamaProvider)(nil)

	// ToolProvider interface (all 4 satisfy)
	var _ ToolProvider = (*OllamaProvider)(nil)

	// ThinkingProvider interface (all 4 satisfy)
	var _ ThinkingProvider = (*OllamaProvider)(nil)

	// ThinkingToolProvider interface (all 4 satisfy)
	var _ ThinkingToolProvider = (*OllamaProvider)(nil)

	// ItemProvider interface (all 4 satisfy)
	var _ ItemProvider = (*OllamaProvider)(nil)

	// ItemToolProvider interface (all 4 satisfy)
	var _ ItemToolProvider = (*OllamaProvider)(nil)

	// ItemThinkingProvider interface (all 4 satisfy)
	var _ ItemThinkingProvider = (*OllamaProvider)(nil)

	// ItemThinkingToolProvider interface (all 4 satisfy)
	var _ ItemThinkingToolProvider = (*OllamaProvider)(nil)

	// Continuer: none of the providers still here satisfies it. OpenAI,
	// Gemini, Grok and OpenCode moved to the new API (0015-PLAN S7).

	// ModelDiscoverer interface (all 4 satisfy)
	var _ ModelDiscoverer = (*OllamaProvider)(nil)
}
