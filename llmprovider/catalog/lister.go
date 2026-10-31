package catalog

import (
	"context"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Lister is a provider that lists its models as a Catalog, its menu, facts
// and Source with them, where llmprovider.ModelLister gives ids only
// (0066-MADR Q1). A caller type-asserts a Provider to it, as to
// llmprovider.ModelLister. An implementation must be safe for
// concurrent use, as the Provider that has it is.
type Lister interface {
	// ListCatalog lists the provider's models as List does, with opts after
	// the provider's own options: those List takes for the provider's id.
	// It returns List's errors, among them one matching
	// llmprovider.ErrUnsupported for an option List refuses for the id, such
	// as WithLimit for a provider that is not built in (0066-MADR, amendment
	// "the Lister's options"). A listing that fails otherwise is told by the
	// Catalog's Live and Err, as List's is.
	ListCatalog(ctx context.Context, opts ...llmprovider.Option) (Catalog, error)
}
