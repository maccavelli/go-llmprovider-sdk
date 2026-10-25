//go:build live_gateways

package llmprovider_test

import (
	"os"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestLive_Refusals_0062 is 0062-PLAN Phase 1's recorder. It runs only with
// LLMPROVIDER_LIVE_0062 set, and writes one record per case to the directory
// LLMPROVIDER_LIVE_0062_OUT names, outside the module, as
// <wire>/<case>.json. A provider's cases are sent when its key variable is
// set, and recorded as skipped when it is not (0062-MADR Q8 (b)). Each case
// is a subtest, so one reruns with -run 'Live_Refusals_0062/<case>'. Billed:
// one generation per case at the provider's default output limit (Q11 (a));
// a refused one carries no usage unless its record says so. It logs each
// case's outcome, status and sends; never a message, a body or a key.
func TestLive_Refusals_0062(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0062") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0062 to run 0062-PLAN Phase 1's recorder")
	}
	out := os.Getenv("LLMPROVIDER_LIVE_0062_OUT")
	if out == "" {
		t.Fatal("LLMPROVIDER_LIVE_0062_OUT unset: name a directory outside the module for the records")
	}
	if err := outsideModule(out); err != nil {
		t.Fatal(err)
	}
	keys := llmprovider.ProviderEnvVars()
	for _, c := range refusalCases() {
		t.Run(c.id, func(t *testing.T) {
			r := c.skipped()
			if key := os.Getenv(keys[c.provider]); key != "" {
				ctx, cancel := llmprovider.LiveCtx(t)
				defer cancel()
				var err error
				if r, err = recordCase(ctx, c, key); err != nil {
					t.Fatal(err)
				}
			}
			if err := writeRefusalRecord(out, r); err != nil {
				t.Fatal(err)
			}
			t.Logf("%s: %s, status %d, %d send(s)", c.id, r.Outcome, r.Status, r.Sends)
		})
	}
}
