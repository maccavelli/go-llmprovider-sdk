package llmprovider

import (
	"errors"
	"strings"
	"testing"
)

// 0039-MADR D1: Request.Cache is CacheShort, CacheLong or zero; any other
// value is refused before the network.

// TestRequest_CacheValues: the zero value and both constants pass, and any
// other value is ErrInvalidRequest, naming it.
func TestRequest_CacheValues(t *testing.T) {
	var c Capabilities
	for _, ok := range []CacheRetention{"", CacheShort, CacheLong} {
		if err := c.Check(&Request{Cache: ok}); err != nil {
			t.Errorf("Cache %q: %v; want it accepted", ok, err)
		}
	}
	err := c.Check(&Request{Cache: "forever"})
	if !errors.Is(err, ErrInvalidRequest) || err == nil || !strings.Contains(err.Error(), `"forever"`) {
		t.Errorf("Cache \"forever\": %v; want ErrInvalidRequest naming the value", err)
	}
}
