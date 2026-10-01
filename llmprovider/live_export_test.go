//go:build live_gateways

package llmprovider

// Live helpers and the build-version hook, exported to the external live
// tests in this directory, which build OpenAI through its own package
// (0015-PLAN S7). They are test-only: this file is not in the library.
var (
	LiveChatGPTSession     = liveChatGPTSession
	LiveVendorSession      = liveVendorSession
	LiveCtx                = liveCtx
	SkipIfTransient        = skipIfTransient
	WithSDKVersion         = withSDKVersion
	LiveEnvKey             = liveEnvKey
	LiveModel              = liveModel
	LiveOpencodeKey        = opencodeKey
	LiveKiloKey            = kiloKey
	LiveKiloFreeCollecting = kiloFreeCollecting
	LiveKiloNonTraining    = kiloNonTraining
)
