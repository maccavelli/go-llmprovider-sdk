package llmprovider

import (
	"reflect"
	"testing"
)

func assertThinkingFields(t *testing.T, body map[string]any, want map[string]any) {
	t.Helper()
	for k, v := range want {
		got, present := body[k]
		switch {
		case v == nil && present:
			t.Errorf("%s = %v, want it absent", k, got)
		case v != nil && !reflect.DeepEqual(got, v):
			t.Errorf("%s = %v, want %v", k, got, v)
		}
	}
}
