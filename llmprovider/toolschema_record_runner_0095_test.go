package llmprovider_test

import (
	"net/http"
	"path/filepath"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestToolSchemaRecord_LiveRunner_0095 (0095-MADR FU21): the live test's
// runner, built offline, sends every case through recordToolCases, each in a
// subtest of its own: two refused cases make two sends and write two
// records. A runner that forgot to call run would make none.
func TestToolSchemaRecord_LiveRunner_0095(t *testing.T) {
	t.Setenv(llmprovider.ProviderEnvVars()[llmprovider.ProviderClaude], toolMade("k-", "Rt5v", 8))
	refusal := toolFixtureField(t, "messages/claude-string-schema-tool.json", "body")
	sends := 0
	reply := toolReplay(func(*http.Request) (int, string) {
		sends++
		return http.StatusBadRequest, refusal
	})
	dir := t.TempDir()
	cases := []toolCase{toolCaseByID(t, "claude-root-string"), toolCaseByID(t, "claude-O1-root-union")}
	if failed := recordToolCases(cases, dir, reply, liveCaseRunner(t)); len(failed) != 0 {
		t.Errorf("%d records not written", len(failed))
	}
	written, err := filepath.Glob(filepath.Join(dir, "*", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if sends != len(cases) || len(written) != len(cases) {
		t.Errorf("the live runner made %d sends and wrote %d records; want %d of each", sends, len(written), len(cases))
	}
}
