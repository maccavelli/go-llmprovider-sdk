package catalog

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0066-PLAN Phase 1 (V1; Q3 (a)): a listed id holding a control character,
// which a terminal would act on, is dropped from every listing.

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
		{"newline\n", true},
	} {
		if got := hasControl(c.s); got != c.want {
			t.Errorf("hasControl(%q) = %v; want %v", c.s, got, c.want)
		}
	}
}

// TestList_ControlIDsDropped_0066: a chatcompat listing and an Ollama
// /api/tags reply, each with ids holding a control character among clean
// ones, give catalogs of the clean ones only.
func TestList_ControlIDsDropped_0066(t *testing.T) {
	clean := []string{"clean-a", "clean-b"}
	t.Run("chatcompat", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"object":"list","data":[{"id":"clean-a"},`+
				`{"id":"evil\u001b]0;owned\u0007\u001b[2Jmodel"},{"id":"clean-b"},{"id":"c1\u009bx"}]}`)
		}))
		t.Cleanup(srv.Close)
		cat, err := List(context.Background(), "acme", llmprovider.NewStaticToken(""), withCompatID("acme"),
			llmprovider.WithBaseURL(srv.URL+"/v1"), WithCache(&Cache{}), llmprovider.WithoutModelMetadata())
		if err != nil {
			t.Fatal(err)
		}
		assertClean0066(t, cat, clean)
	})
	t.Run("ollama", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/tags" {
				http.NotFound(w, r)
				return
			}
			_, _ = io.WriteString(w, `{"models":[{"name":"clean-a"},{"name":"evil\u001b[2J"},`+
				`{"name":"clean-b"},{"name":"del\u007f"}]}`)
		}))
		t.Cleanup(srv.Close)
		cat, err := List(context.Background(), llmprovider.ProviderOllama, llmprovider.NewStaticToken(""),
			llmprovider.WithBaseURL(srv.URL), WithCache(&Cache{}), llmprovider.WithoutModelMetadata())
		if err != nil {
			t.Fatal(err)
		}
		assertClean0066(t, cat, clean)
	})
}

// assertClean0066 fails t unless cat's Usable, Recommended and Menu hold
// only want's ids, and no id with a control character.
func assertClean0066(t *testing.T, cat Catalog, want []string) {
	t.Helper()
	var menu []string
	for _, e := range cat.Menu {
		menu = append(menu, e.ID)
	}
	for name, ids := range map[string][]string{"Usable": cat.Usable, "Recommended": cat.Recommended, "Menu": menu} {
		got := slices.Clone(ids)
		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Errorf("%s = %q; want %q", name, ids, want)
		}
		for _, id := range ids {
			if strings.ContainsFunc(id, func(r rune) bool { return r < 0x20 || r == 0x7f || r >= 0x80 && r <= 0x9f }) {
				t.Errorf("%s holds %q", name, id)
			}
		}
	}
}
