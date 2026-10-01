package llmprovider

import "testing"

// TestProviderInterfaceSatisfaction verifies the full interface satisfaction
// matrix across every concrete provider implementation in llmprovider.
func TestProviderInterfaceSatisfaction(t *testing.T) {
	// Base Provider interface (all 4 satisfy)
	var _ LegacyProvider = (*HuggingFaceProvider)(nil)
	var _ LegacyProvider = (*KiloProvider)(nil)
	var _ LegacyProvider = (*OllamaProvider)(nil)

	// ToolProvider interface (all 4 satisfy)
	var _ ToolProvider = (*HuggingFaceProvider)(nil)
	var _ ToolProvider = (*KiloProvider)(nil)
	var _ ToolProvider = (*OllamaProvider)(nil)

	// ThinkingProvider interface (all 4 satisfy)
	var _ ThinkingProvider = (*HuggingFaceProvider)(nil)
	var _ ThinkingProvider = (*KiloProvider)(nil)
	var _ ThinkingProvider = (*OllamaProvider)(nil)

	// ThinkingToolProvider interface (all 4 satisfy)
	var _ ThinkingToolProvider = (*HuggingFaceProvider)(nil)
	var _ ThinkingToolProvider = (*KiloProvider)(nil)
	var _ ThinkingToolProvider = (*OllamaProvider)(nil)

	// ItemProvider interface (all 4 satisfy)
	var _ ItemProvider = (*HuggingFaceProvider)(nil)
	var _ ItemProvider = (*KiloProvider)(nil)
	var _ ItemProvider = (*OllamaProvider)(nil)

	// ItemToolProvider interface (all 4 satisfy)
	var _ ItemToolProvider = (*HuggingFaceProvider)(nil)
	var _ ItemToolProvider = (*KiloProvider)(nil)
	var _ ItemToolProvider = (*OllamaProvider)(nil)

	// ItemThinkingProvider interface (all 4 satisfy)
	var _ ItemThinkingProvider = (*HuggingFaceProvider)(nil)
	var _ ItemThinkingProvider = (*KiloProvider)(nil)
	var _ ItemThinkingProvider = (*OllamaProvider)(nil)

	// ItemThinkingToolProvider interface (all 4 satisfy)
	var _ ItemThinkingToolProvider = (*HuggingFaceProvider)(nil)
	var _ ItemThinkingToolProvider = (*KiloProvider)(nil)
	var _ ItemThinkingToolProvider = (*OllamaProvider)(nil)

	// Continuer: none of the providers still here satisfies it. OpenAI,
	// Gemini, Grok and OpenCode moved to the new API (0015-PLAN S7).

	// ModelDiscoverer interface (all 4 satisfy)
	var _ ModelDiscoverer = (*HuggingFaceProvider)(nil)
	var _ ModelDiscoverer = (*KiloProvider)(nil)
	var _ ModelDiscoverer = (*OllamaProvider)(nil)
}
