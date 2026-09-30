package llmprovider

import "testing"

// TestProviderInterfaceSatisfaction verifies the full interface satisfaction
// matrix across every concrete provider implementation in llmprovider.
func TestProviderInterfaceSatisfaction(t *testing.T) {
	// Base Provider interface (all 4 satisfy)
	var _ LegacyProvider = (*ClaudeProvider)(nil)
	var _ LegacyProvider = (*GeminiProvider)(nil)
	var _ LegacyProvider = (*GrokProvider)(nil)
	var _ LegacyProvider = (*OpencodeProvider)(nil)
	var _ LegacyProvider = (*HuggingFaceProvider)(nil)
	var _ LegacyProvider = (*KiloProvider)(nil)
	var _ LegacyProvider = (*OllamaProvider)(nil)

	// ToolProvider interface (all 4 satisfy)
	var _ ToolProvider = (*ClaudeProvider)(nil)
	var _ ToolProvider = (*GeminiProvider)(nil)
	var _ ToolProvider = (*GrokProvider)(nil)
	var _ ToolProvider = (*OpencodeProvider)(nil)
	var _ ToolProvider = (*HuggingFaceProvider)(nil)
	var _ ToolProvider = (*KiloProvider)(nil)
	var _ ToolProvider = (*OllamaProvider)(nil)

	// ThinkingProvider interface (all 4 satisfy)
	var _ ThinkingProvider = (*ClaudeProvider)(nil)
	var _ ThinkingProvider = (*GeminiProvider)(nil)
	var _ ThinkingProvider = (*GrokProvider)(nil)
	var _ ThinkingProvider = (*OpencodeProvider)(nil)
	var _ ThinkingProvider = (*HuggingFaceProvider)(nil)
	var _ ThinkingProvider = (*KiloProvider)(nil)
	var _ ThinkingProvider = (*OllamaProvider)(nil)

	// ThinkingToolProvider interface (all 4 satisfy)
	var _ ThinkingToolProvider = (*ClaudeProvider)(nil)
	var _ ThinkingToolProvider = (*GeminiProvider)(nil)
	var _ ThinkingToolProvider = (*GrokProvider)(nil)
	var _ ThinkingToolProvider = (*OpencodeProvider)(nil)
	var _ ThinkingToolProvider = (*HuggingFaceProvider)(nil)
	var _ ThinkingToolProvider = (*KiloProvider)(nil)
	var _ ThinkingToolProvider = (*OllamaProvider)(nil)

	// ItemProvider interface (all 4 satisfy)
	var _ ItemProvider = (*ClaudeProvider)(nil)
	var _ ItemProvider = (*GeminiProvider)(nil)
	var _ ItemProvider = (*GrokProvider)(nil)
	var _ ItemProvider = (*OpencodeProvider)(nil)
	var _ ItemProvider = (*HuggingFaceProvider)(nil)
	var _ ItemProvider = (*KiloProvider)(nil)
	var _ ItemProvider = (*OllamaProvider)(nil)

	// ItemToolProvider interface (all 4 satisfy)
	var _ ItemToolProvider = (*ClaudeProvider)(nil)
	var _ ItemToolProvider = (*GeminiProvider)(nil)
	var _ ItemToolProvider = (*GrokProvider)(nil)
	var _ ItemToolProvider = (*OpencodeProvider)(nil)
	var _ ItemToolProvider = (*HuggingFaceProvider)(nil)
	var _ ItemToolProvider = (*KiloProvider)(nil)
	var _ ItemToolProvider = (*OllamaProvider)(nil)

	// ItemThinkingProvider interface (all 4 satisfy)
	var _ ItemThinkingProvider = (*ClaudeProvider)(nil)
	var _ ItemThinkingProvider = (*GeminiProvider)(nil)
	var _ ItemThinkingProvider = (*GrokProvider)(nil)
	var _ ItemThinkingProvider = (*OpencodeProvider)(nil)
	var _ ItemThinkingProvider = (*HuggingFaceProvider)(nil)
	var _ ItemThinkingProvider = (*KiloProvider)(nil)
	var _ ItemThinkingProvider = (*OllamaProvider)(nil)

	// ItemThinkingToolProvider interface (all 4 satisfy)
	var _ ItemThinkingToolProvider = (*ClaudeProvider)(nil)
	var _ ItemThinkingToolProvider = (*GeminiProvider)(nil)
	var _ ItemThinkingToolProvider = (*GrokProvider)(nil)
	var _ ItemThinkingToolProvider = (*OpencodeProvider)(nil)
	var _ ItemThinkingToolProvider = (*HuggingFaceProvider)(nil)
	var _ ItemThinkingToolProvider = (*KiloProvider)(nil)
	var _ ItemThinkingToolProvider = (*OllamaProvider)(nil)

	// Continuer optional interface (OpenAI, Gemini, Grok satisfy; Claude is
	// stateless, and the OpenCode gateway rejects previous_response_id with
	// HTTP 400, so OpencodeProvider deliberately does not implement it)
	var _ Continuer = (*GeminiProvider)(nil)
	var _ Continuer = (*GrokProvider)(nil)

	// ModelDiscoverer interface (all 4 satisfy)
	var _ ModelDiscoverer = (*ClaudeProvider)(nil)
	var _ ModelDiscoverer = (*GeminiProvider)(nil)
	var _ ModelDiscoverer = (*GrokProvider)(nil)
	var _ ModelDiscoverer = (*OpencodeProvider)(nil)
	var _ ModelDiscoverer = (*HuggingFaceProvider)(nil)
	var _ ModelDiscoverer = (*KiloProvider)(nil)
	var _ ModelDiscoverer = (*OllamaProvider)(nil)
}
