package catalog

// Test hooks for the external tests in this directory (package catalog_test),
// which build providers through llmprovider/providers: an in-package test
// cannot, as the providers import catalog.
var (
	PinRankingNow           = pinRankingNow
	RefNow                  = refNow
	EnableModelMetadata     = enableModelMetadata
	ResetModelMetadataCache = resetModelMetadataCache
	MetadataServer          = metadataServer
	ServeBody               = serveBody
	KiloRankEntry           = kiloRankEntry
	ZenStyleListing         = zenStyleListing
	MDModel                 = mdModel
	MDSection               = mdSection
	HFRankListing           = hfRankListing
	HFRankMetadata          = hfRankMetadata
)
