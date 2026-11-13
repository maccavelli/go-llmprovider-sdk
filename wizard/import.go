package wizard

import (
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// vendorAuthPath resolves a vendor CLI's auth file as the CLI does (MADR 0012
// §5.1), through auth.VendorCLIPath (0074-MADR D10) and the caller's
// environment reader. The wizard reads no environment variable unless the
// caller passes one (0015-MADR D9).
func vendorAuthPath(provider llmprovider.ProviderID, o Options) (string, error) {
	return auth.VendorCLIPath(provider, o.LookupEnv)
}
