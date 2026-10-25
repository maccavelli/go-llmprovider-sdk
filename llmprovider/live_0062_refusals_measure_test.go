//go:build live_gateways

package llmprovider_test

import (
	"net/http"
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
//
// Ollama's cases (0062-PLAN Phase 5, Q9 (c)) run only when
// LLMPROVIDER_LIVE_0062_OLLAMA names the host's base URL and
// LLMPROVIDER_LIVE_0062_OLLAMA_MODEL a thinking model installed there; run
// them alone with -run 'Live_Refusals_0062/ollama-'. Their records hold the
// model, never the host.
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
	host, model := os.Getenv("LLMPROVIDER_LIVE_0062_OLLAMA"), os.Getenv("LLMPROVIDER_LIVE_0062_OLLAMA_MODEL")
	if host == "" || model == "" {
		return
	}
	for _, c := range ollamaRefusalCases(model) {
		t.Run(c.id, func(t *testing.T) {
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			r, err := recordCase(ctx, c, "", llmprovider.WithBaseURL(host))
			if err != nil {
				t.Fatal(err)
			}
			if err := writeRefusalRecord(out, r); err != nil {
				t.Fatal(err)
			}
			t.Logf("%s: %s, status %d, %d send(s)", c.id, r.Outcome, r.Status, r.Sends)
		})
	}
}

// TestLive_Learn_0062 is 0062-PLAN Phase 5's live confirmation of each rule
// 0062-MADR D8 removed. It runs only with LLMPROVIDER_LIVE_0062_CONFIRM set.
// Each case is a subtest, run when its provider's key variable is set and
// skipped, by name, when it is not. Each sends its request twice through one
// provider, with no rewriter, no metadata and no MaxTokens (Q11 (a)), and
// checks that the first succeeds having learned what Phase 1's fixtures say,
// and the second after one send, learning nothing (confirmLearn). Billed: one
// accepted generation per request, and a refused send per refusal learned.
// Ollama's case runs under the variables of its recording. It logs each case's
// rule and what it learned; never a message, a body or a key.
func TestLive_Learn_0062(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0062_CONFIRM") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0062_CONFIRM to run 0062-PLAN Phase 5's live confirmation")
	}
	keys := llmprovider.ProviderEnvVars()
	for _, c := range learnConfirms() {
		t.Run(c.id, func(t *testing.T) {
			key := os.Getenv(keys[c.provider])
			if key == "" {
				t.Skipf("%s unset", keys[c.provider])
			}
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			if err := confirmLearn(ctx, c, key, http.DefaultTransport); err != nil {
				t.Fatal(err)
			}
			t.Logf("%s (%s): learned %v, then right the first time", c.id, c.rule, c.learns)
		})
	}
	host, model := os.Getenv("LLMPROVIDER_LIVE_0062_OLLAMA"), os.Getenv("LLMPROVIDER_LIVE_0062_OLLAMA_MODEL")
	c := ollamaConfirm(model)
	t.Run(c.id, func(t *testing.T) {
		if host == "" || model == "" {
			t.Skip("LLMPROVIDER_LIVE_0062_OLLAMA or LLMPROVIDER_LIVE_0062_OLLAMA_MODEL unset")
		}
		ctx, cancel := llmprovider.LiveCtx(t)
		defer cancel()
		if err := confirmLearn(ctx, c, "", http.DefaultTransport, llmprovider.WithBaseURL(host)); err != nil {
			t.Fatal(err)
		}
		t.Logf("%s (%s): learned %v, then right the first time", c.id, c.rule, c.learns)
	})
}
