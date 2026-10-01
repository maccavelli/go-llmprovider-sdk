//go:build live_gateways

package llmprovider

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

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
	LiveTogetherKey        = togetherLiveKey
)

// withSDKVersion makes the build report v as this module's version for one
// (non-parallel) test. It was in chatgpt_listing_version_test.go, whose tests
// moved to catalog (0015-PLAN S8, commit 2); the external live tests still use
// it, as WithSDKVersion.
func withSDKVersion(t *testing.T, v string) {
	t.Helper()
	saved := transport.BuildVersions
	transport.BuildVersions = func() (string, string) { return v, v }
	t.Cleanup(func() { transport.BuildVersions = saved })
}
