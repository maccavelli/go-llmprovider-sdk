package messages

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0039-MADR D2: a cached Messages body carries the top-level cache_control
// and one breakpoint, on the system prompt or else the last tool, with one
// TTL; an uncached body is untouched.

// cacheBody is a Messages body with a system prompt and two tools, each
// optional.
func cacheBody(system bool, tools int) map[string]any {
	body := map[string]any{"model": "m", "messages": []any{}}
	if system {
		body["system"] = "Be brief."
	}
	if tools > 0 {
		list := make([]map[string]any, tools)
		for i := range list {
			list[i] = map[string]any{"name": "t"}
		}
		body["tools"] = list
	}
	return body
}

// markers finds every cache_control in body's JSON, by path.
func markers(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	found := map[string]any{}
	var walk func(path string, v any)
	walk = func(path string, v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, child := range x {
				if k == "cache_control" {
					found[path] = child
					continue
				}
				walk(path+"."+k, child)
			}
		case []any:
			for i, child := range x {
				walk(path+"["+string(rune('0'+i))+"]", child)
			}
		}
	}
	walk("", v)
	return found
}

// TestAddCache: each retention's TTL, the top-level marker and one
// breakpoint where D2 puts it, never more than two markers.
func TestAddCache(t *testing.T) {
	for _, tc := range []struct {
		retention llmprovider.CacheRetention
		ttl       string
	}{{llmprovider.CacheShort, "5m"}, {llmprovider.CacheLong, "1h"}} {
		want := map[string]any{"type": "ephemeral", "ttl": tc.ttl}
		for _, shape := range []struct {
			name   string
			system bool
			tools  int
			at     []string
		}{
			{"system and tools", true, 2, []string{"", ".system[0]"}},
			{"tools alone", false, 2, []string{"", ".tools[1]"}},
			{"neither", false, 0, []string{""}},
		} {
			body := cacheBody(shape.system, shape.tools)
			AddCache(body, tc.retention)
			got := markers(t, body)
			if len(got) != len(shape.at) {
				t.Errorf("%s, %s: markers at %v; want them at %q", tc.retention, shape.name, keys(got), shape.at)
				continue
			}
			for _, path := range shape.at {
				if !reflect.DeepEqual(got[path], want) {
					t.Errorf("%s, %s: the marker at %q is %v; want %v", tc.retention, shape.name, path, got[path], want)
				}
			}
		}
	}
}

// TestAddCache_SystemFormOnlyWhenAsked: with no retention the body is left
// as it was, its system a string.
func TestAddCache_SystemFormOnlyWhenAsked(t *testing.T) {
	body, before := cacheBody(true, 2), cacheBody(true, 2)
	AddCache(body, "")
	if !reflect.DeepEqual(body, before) {
		t.Errorf("AddCache with no retention changed the body:\n got  %v\n want %v", body, before)
	}
	cached := cacheBody(true, 0)
	AddCache(cached, llmprovider.CacheShort)
	if _, isString := cached["system"].(string); isString {
		t.Errorf("a cached body's system is still a string: %v", cached["system"])
	}
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
