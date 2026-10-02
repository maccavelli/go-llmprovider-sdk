package catalog

import (
	"fmt"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// What the listing needs of the gateways. The providers are in providers/,
// and keep their own copies of these values (0015-PLAN S7).

// Default gateway base URLs. Both OpenCode gateways share one credential; they
// differ only in base URL, catalog, and per-model routing.
const (
	opencodeZenBaseURL = "https://opencode.ai/zen/v1"
	opencodeGoBaseURL  = "https://opencode.ai/zen/go/v1"
	// huggingFaceBaseURL is the router's OpenAI-compatible base.
	huggingFaceBaseURL = "https://router.huggingface.co/v1"
	togetherBaseURL    = "https://api.together.ai/v1"
)

// opencodeSessionHeader carries a stable conversation id. OpenCode Go rejects
// requests without it (400 MissingSessionID, measured 2026-09-26), and both
// gateways use it for routing and prompt caching.
const opencodeSessionHeader = "x-opencode-session"

// opencodeBaseURL returns the default base URL for a gateway.
func opencodeBaseURL(gateway llmprovider.ProviderID) (string, error) {
	switch gateway {
	case llmprovider.ProviderOpencodeZen:
		return opencodeZenBaseURL, nil
	case llmprovider.ProviderOpencodeGo:
		return opencodeGoBaseURL, nil
	default:
		return "", fmt.Errorf("unsupported opencode gateway: %s", gateway)
	}
}
