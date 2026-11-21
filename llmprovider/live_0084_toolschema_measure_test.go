//go:build live_gateways

package llmprovider_test

import (
	"net/http"
	"os"
	"testing"
)

// TestLive_ToolSchema_0084 is 0084-PLAN Phase 1's recorder. It runs only with
// LLMPROVIDER_LIVE_0084 set, and writes one record per case to the directory
// LLMPROVIDER_LIVE_0084_OUT names, outside the module, as
// <wire>/<case>.json. A route's cases are sent when its key variable is set
// to something other than blanks, and recorded skipped when it is not: the
// key reaches the run only through the process environment, never an
// argument or a file (0084-MADR ST77, ST78). Every case runs in one call
// (recordToolCases, untagged, so that a test runs the same loop), each as a
// subtest named by its id, so one case reruns with -run
// '^TestLive_ToolSchema_0084$/^<case>(#01)?$', the second send being the
// subtest Go names <case>#01 (an anchored ^<case>$ would skip it). After the
// pass over every case, each case that was other is sent once more and
// recorded unreachable if it is still other, so one command records
// everything (ST67); the writer never replaces a refused or accepted record
// (ST70), so a rerun is harmless. Billed:
// one generation per case, at most 64 output tokens; not established whether
// a refused request is billed.
//
// It prints, per case, one line: "<case> <route> <model>: status <n>,
// <outcome>", and nothing else. Every failure and skip text is fixed and holds
// at most a case id, and no call here takes an error, a body or a key
// (TestToolSchemaRecord_CallAllowlist_0084; 0084-MADR ST71, ST79, ST80).
func TestLive_ToolSchema_0084(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0084") == "" {
		t.Skip(toolTextSkip)
	}
	out := os.Getenv("LLMPROVIDER_LIVE_0084_OUT")
	if out == "" {
		t.Fatal(toolTextNoOut)
	}
	if toolInsideModule(out) {
		t.Fatal(toolTextInside)
	}
	// One call runs every case, each as a subtest named by its id, and the
	// second send of each that was other as a second subtest of the same name,
	// which Go names <case>#01; a case whose record could not be written fails
	// with a fixed text, and the others still run.
	each := func(c toolCase, run func(emitFunc)) {
		t.Run(c.id, func(t *testing.T) { run(t.Logf) })
	}
	for _, c := range recordToolCases(toolCases(), out, http.DefaultTransport, each) {
		t.Errorf(toolTextNotWritten, c.id)
	}
}
