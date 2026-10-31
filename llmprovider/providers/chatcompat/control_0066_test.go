package chatcompat

import (
	"context"
	"net/http"
	"slices"
	"testing"
)

// 0066-PLAN Phase 1 (V1; Q3 (a)): a listed id holding a control character
// is dropped from what Discover returns, as catalog drops it from a listing.

// TestHasControl_0066: the rule of internal/redact's isControl, at each end
// of each range.
func TestHasControl_0066(t *testing.T) {
	for _, c := range []struct {
		s    string
		want bool
	}{
		{"plain-model", false},
		{"tab\tok", false},
		{"nbsp ok", false},
		{"nul\x00", true},
		{"esc\x1b[2J", true},
		{"us\x1f", true},
		{"del\x7f", true},
		{"c1\u0080", true},
		{"csi\u009b", true},
		{"c1\u009f", true},
	} {
		if got := hasControl(c.s); got != c.want {
			t.Errorf("hasControl(%q) = %v; want %v", c.s, got, c.want)
		}
	}
}

// TestDiscover_ControlIDsDropped_0066: a server listing ids with control
// characters among clean ones is found with the clean ones only.
func TestDiscover_ControlIDsDropped_0066(t *testing.T) {
	s := newStub(t, http.StatusOK, `{"object":"list","data":[{"id":"clean-a"},`+
		`{"id":"evil\u001b]0;owned\u0007\u001b[2Jmodel"},{"id":"clean-b"},{"id":"c1\u009bx"}]}`)
	withPorts(t, s.port(t))
	found, err := Discover(context.Background())
	if err != nil || len(found) != 1 {
		t.Fatalf("Discover = %+v, %v; want one server", found, err)
	}
	if !slices.Equal(found[0].Models, []string{"clean-a", "clean-b"}) {
		t.Errorf("Models = %q; want the clean ids only", found[0].Models)
	}
}
