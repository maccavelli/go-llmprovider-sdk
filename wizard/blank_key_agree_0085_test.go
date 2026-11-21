package wizard

// 0085-MADR ST77: the wizard and the constructors agree on a blank value. The
// test lives here, not in llmprovider/providers, because importing wizard from
// that package's tests would make it depend on golang.org/x/term and
// golang.org/x/sys, which `make dep-check` forbids.

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// TestWizardAndConstructorAgreeOnBlank_0085 (ST77): an environment value of
// spaces is not offered by ResolveCredential under AllowEnv, and the same value
// given to WithAPIKey gives the outcome of "".
func TestWizardAndConstructorAgreeOnBlank_0085(t *testing.T) {
	f := newFake(t, fakePrompter{secrets: []string{"made-up-key-1234"}})
	o := options0085(t)
	o.AllowEnv, o.LookupEnv = true, func(string) string { return "   " }
	res, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderTogether)
	if err != nil || len(f.seenConfirm) != 0 || res.APIKey != "made-up-key-1234" {
		t.Errorf("wizard: err %v, confirms %d, key %q; want the typed key and no question", err, len(f.seenConfirm), res.APIKey)
	}
	_, want := providers.New(llmprovider.ProviderTogether, llmprovider.WithAPIKey(""))
	_, got := providers.New(llmprovider.ProviderTogether, llmprovider.WithAPIKey("   "))
	if want == nil || got == nil || got.Error() != want.Error() {
		t.Errorf("constructor: err %v; want the outcome of an empty key, %v", got, want)
	}
}
