package wizard

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestSelectRecommended_ExistingModelOnlyForItsProvider (0026-MADR F46): the
// saved model is the menu's default only for its own provider (MADR 0007
// §4.3), even when another provider lists the same id.
func TestSelectRecommended_ExistingModelOnlyForItsProvider(t *testing.T) {
	models := []string{"a/one", "b/two", "meta-llama/shared"}
	for _, c := range []struct {
		name     string
		existing llmprovider.ProviderID
		want     int
	}{
		{"another provider's model", llmprovider.ProviderTogether, 0},
		{"its own provider's model", llmprovider.ProviderHuggingFace, 2},
	} {
		f := newFake(t, fakePrompter{selects: []int{0}})
		o := Options{Existing: Result{Provider: c.existing, Model: "meta-llama/shared"}}
		d := llmprovider.Descriptor{ID: llmprovider.ProviderHuggingFace, Label: "Hugging Face"}
		if _, err := selectRecommended(f, d, models, o); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := f.seenSelectDefault[0]; got != c.want {
			t.Errorf("%s: default index = %d; want %d", c.name, got, c.want)
		}
	}
}
