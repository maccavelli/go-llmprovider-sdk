package wire

import "github.com/maccavelli/go-llmprovider-sdk/llmprovider"

// keyPromptCacheKey is the routing key OpenAI, xAI and Together cache by.
const keyPromptCacheKey = "prompt_cache_key"

// AddPromptCacheKey sends session as prompt_cache_key when the request asks
// for caching, so that a conversation's requests reach the same cache
// (0039-MADR D3). With no retention, or no session, the body is unchanged.
func AddPromptCacheKey(body map[string]any, retention llmprovider.CacheRetention, session string) {
	if retention != "" && session != "" {
		body[keyPromptCacheKey] = session
	}
}
