package llmprovider

import "fmt"

// What the gateway listing and the descriptors need of OpenCode Zen and Go.
// The provider is providers/opencode (0015-PLAN S7), which keeps its own
// copies of these values.

// Default gateway base URLs. Both gateways share one credential; they differ
// only in base URL, catalog, and per-model routing.
const (
	opencodeZenBaseURL = "https://opencode.ai/zen/v1"
	opencodeGoBaseURL  = "https://opencode.ai/zen/go/v1"
)

// opencodeSessionHeader carries a stable conversation id. OpenCode Go rejects
// requests without it (400 MissingSessionID, measured 2026-09-26), and both
// gateways use it for routing and prompt caching.
const opencodeSessionHeader = "x-opencode-session"

// opencodeBaseURL returns the default base URL for a gateway.
func opencodeBaseURL(gateway string) (string, error) {
	switch gateway {
	case ProviderOpencodeZen:
		return opencodeZenBaseURL, nil
	case ProviderOpencodeGo:
		return opencodeGoBaseURL, nil
	default:
		return "", fmt.Errorf("unsupported opencode gateway: %s", gateway)
	}
}
