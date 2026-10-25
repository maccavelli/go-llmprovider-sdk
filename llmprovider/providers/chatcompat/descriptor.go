package chatcompat

import "github.com/maccavelli/go-llmprovider-sdk/llmprovider"

// Descriptor is a chatcompat provider's descriptor, for a caller's Registry
// (0015-MADR D10; 0061-MADR D4): id and label as given, a base URL the caller
// sets, and no key required. It has no default base URL, no environment
// variable and no static models: the server is the caller's.
func Descriptor(id llmprovider.ProviderID, label string) llmprovider.Descriptor {
	return llmprovider.Descriptor{
		ID:              id,
		Label:           label,
		SupportsBaseURL: true,
	}
}
