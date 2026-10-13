package messages

import "github.com/maccavelli/go-llmprovider-sdk/llmprovider"

// keyCacheControl is Anthropic's cache marker, on the request or a block.
const keyCacheControl = "cache_control"

// CacheMarker is the cache_control a cached request carries: an ephemeral
// entry with retention's TTL, "5m" for CacheShort and "1h" for CacheLong
// (0039-MADR D2). Every marker of one request has the same TTL, since
// Anthropic refuses a longer one after a shorter.
func CacheMarker(retention llmprovider.CacheRetention) map[string]any {
	ttl := "5m"
	if retention == llmprovider.CacheLong {
		ttl = "1h"
	}
	return map[string]any{"type": "ephemeral", "ttl": ttl}
}

// AddCache marks a Messages body for caching, as 0039-MADR D2 decides. It
// runs once the body's system and tools are set. With a retention, the body
// gets two markers at most, of Anthropic's four:
//
//   - the top-level cache_control, Anthropic's automatic caching, whose
//     breakpoint moves forward with the conversation;
//   - one breakpoint at the end of the fixed prefix: on the system prompt,
//     sent then as one text block, or with none, on the last tool.
//
// With no retention the body is left as it was, its system a string.
func AddCache(body map[string]any, retention llmprovider.CacheRetention) {
	if retention == "" {
		return
	}
	body[keyCacheControl] = CacheMarker(retention)
	if system, ok := body["system"].(string); ok && system != "" {
		body["system"] = []map[string]any{{"type": "text", "text": system, keyCacheControl: CacheMarker(retention)}}
		return
	}
	if tools, ok := body["tools"].([]map[string]any); ok && len(tools) > 0 {
		tools[len(tools)-1][keyCacheControl] = CacheMarker(retention)
	}
}
