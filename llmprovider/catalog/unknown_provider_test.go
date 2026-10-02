package catalog

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestList_UnknownProviderIsUnsupported: an id List does not list gives an
// error matching ErrUnsupported, which is how wizard knows to ask the
// provider's own ListModels (0015-PLAN S12b; 0015-MADR, amendment "the wizard
// lists a provider through its own ListModels").
func TestList_UnknownProviderIsUnsupported(t *testing.T) {
	_, err := listT(context.Background(), "acme-test", llmprovider.NewStaticToken("key"))
	if !errors.Is(err, llmprovider.ErrUnsupported) {
		t.Fatalf("err = %v, want one matching ErrUnsupported", err)
	}
	if want := `model listing: unsupported provider "acme-test"`; !strings.HasPrefix(err.Error(), want) {
		t.Errorf("err = %q, want it to start %q (R27)", err, want)
	}
}
