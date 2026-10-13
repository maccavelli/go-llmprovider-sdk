package wire

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestAddPromptCacheKey: the key is the session for either retention, and
// absent for the zero value or an empty session (0039-MADR D3).
func TestAddPromptCacheKey(t *testing.T) {
	for _, tc := range []struct {
		retention llmprovider.CacheRetention
		session   string
		want      any
	}{
		{llmprovider.CacheShort, "s-1", "s-1"},
		{llmprovider.CacheLong, "s-1", "s-1"},
		{"", "s-1", nil},
		{llmprovider.CacheShort, "", nil},
	} {
		body := map[string]any{}
		AddPromptCacheKey(body, tc.retention, tc.session)
		if got := body[keyPromptCacheKey]; got != tc.want {
			t.Errorf("retention %q, session %q: prompt_cache_key = %v; want %v", tc.retention, tc.session, got, tc.want)
		}
	}
}
