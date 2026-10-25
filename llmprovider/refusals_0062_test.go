package llmprovider_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// 0062-PLAN Phase 1 step 3: the recorder (refusals_0062_record_test.go),
// offline, against a wiretest server.

// TestRefusalRecorder_Fields_0062: the case's fields arrive as set and
// dropped, and a 400 is recorded with its status, code, message and body,
// with a secret in the body redacted.
func TestRefusalRecorder_Fields_0062(t *testing.T) {
	secret := "sk-ant-api03-" + strings.Repeat("Ab1x", 6)
	refused := `{"type":"error","error":{"type":"invalid_request_error",` +
		`"message":"output_config.effort: bogus is not one of low, medium, high"},"echo":"` + secret + `"}`
	srv := wiretest.NewServer(t, func(*http.Request) wiretest.Reply {
		return wiretest.Reply{Status: http.StatusBadRequest, Body: refused}
	})
	c := refusalCase{id: "offline-fields", cause: causeThinking, wire: wireMessages,
		provider: llmprovider.ProviderClaude, model: "claude-sonnet-5",
		set:  map[string]any{"thinking": map[string]any{"type": "adaptive"}, "output_config.effort": "bogus"},
		drop: []string{"max_tokens", "no.such.field"}}
	r, err := recordCase(context.Background(), c, "k", llmprovider.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	reqs := srv.Requests()
	if len(reqs) == 0 {
		t.Fatal("no request reached the server")
	}
	body, _ := reqs[len(reqs)-1].Body.(map[string]any)
	thinking, _ := body["thinking"].(map[string]any)
	config, _ := body["output_config"].(map[string]any)
	if thinking["type"] != "adaptive" || config["effort"] != "bogus" || body["model"] != "claude-sonnet-5" {
		t.Errorf("sent thinking %v, output_config %v, model %v; want adaptive, effort bogus, claude-sonnet-5",
			body["thinking"], body["output_config"], body["model"])
	}
	if _, ok := body["max_tokens"]; ok {
		t.Errorf("sent max_tokens %v; want it dropped", body["max_tokens"])
	}
	if r.Outcome != outcomeRefused || r.Status != http.StatusBadRequest || r.Code != "invalid_request_error" ||
		!strings.Contains(r.Message, "bogus is not one of") || r.Sends != len(reqs) || r.Usage {
		t.Errorf("record %+v; want refused, 400, invalid_request_error, its message, %d send(s), no usage", r, len(reqs))
	}
	if !strings.Contains(r.Body, "bogus is not one of") {
		t.Errorf("body %q; want the error body", r.Body)
	}
	for field, text := range map[string]string{"body": r.Body, "message": r.Message} {
		if strings.Contains(text, secret) {
			t.Errorf("%s holds the planted secret unredacted", field)
		}
	}
}

// TestRefusalRecorder_Accepted_0062: an accepted reply is recorded with its
// usage and no body, and the other-cause tool is sent as its schema.
func TestRefusalRecorder_Accepted_0062(t *testing.T) {
	srv := wiretest.NewServer(t, wirecase.Case{}.Reply)
	c := refusalCase{id: "offline-tool", cause: causeOther, wire: wireMessages,
		provider: llmprovider.ProviderClaude, model: "claude-sonnet-5", tool: true}
	r, err := recordCase(context.Background(), c, "k", llmprovider.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if r.Outcome != outcomeAccepted || r.Status != http.StatusOK || !r.Usage || r.Body != "" || r.Tool != stringSchema {
		t.Errorf("record %+v; want accepted, 200, usage, no body, the tool's schema", r)
	}
	reqs := srv.Requests()
	body, _ := reqs[len(reqs)-1].Body.(map[string]any)
	tools, _ := body["tools"].([]any)
	var schema any
	if len(tools) == 1 {
		tool, _ := tools[0].(map[string]any)
		schema = tool["input_schema"]
	}
	if got, _ := json.Marshal(schema); string(got) != stringSchema {
		t.Errorf("sent tools %v; want one whose input_schema is %s", body["tools"], stringSchema)
	}
}

// TestRefusalOutcome_0062: the outcome is the status's, wrapped or not.
func TestRefusalOutcome_0062(t *testing.T) {
	api := func(status int) error { return &llmprovider.APIError{Provider: "p", Status: status} }
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"none", nil, outcomeAccepted},
		{"404", api(http.StatusNotFound), outcomeUnreachable},
		{"400", api(http.StatusBadRequest), outcomeRefused},
		{"422", api(http.StatusUnprocessableEntity), outcomeRefused},
		{"400 wrapped", fmt.Errorf("generate: %w", api(http.StatusBadRequest)), outcomeRefused},
		{"401", api(http.StatusUnauthorized), outcomeOther},
		{"403", api(http.StatusForbidden), outcomeOther},
		{"408", api(http.StatusRequestTimeout), outcomeOther},
		{"409", api(http.StatusConflict), outcomeOther},
		{"429", api(http.StatusTooManyRequests), outcomeOther},
		{"500", api(http.StatusInternalServerError), outcomeOther},
		{"stream", api(0), outcomeOther},
		{"transport", fmt.Errorf("dial: %w", io.ErrUnexpectedEOF), outcomeOther},
	} {
		if got := refusalOutcome(tc.err); got != tc.want {
			t.Errorf("%s: %q; want %q", tc.name, got, tc.want)
		}
	}
}

// TestRefusalCases_Table_0062: the case table is 0062-PLAN Phase 1 step 2's,
// amended by Q9 (c) and Q10 (a): 29 thinking cases, and the two other causes
// on each of 11 providers and routes; no Ollama; each id unique and fit for
// a subtest and a file name; each provider's key named.
func TestRefusalCases_Table_0062(t *testing.T) {
	cases := refusalCases()
	keys := llmprovider.ProviderEnvVars()
	seen := map[string]bool{}
	counts := map[string]int{}
	others := map[string]int{}
	for _, c := range cases {
		if c.id == "" || strings.ContainsAny(c.id, `/\ `) || seen[c.id] {
			t.Errorf("case id %q: empty, unfit for a subtest or a file, or repeated", c.id)
		}
		seen[c.id] = true
		counts[c.cause]++
		if _, ok := inputPath[c.wire]; !ok {
			t.Errorf("%s: unknown wire %q", c.id, c.wire)
		}
		if c.provider == llmprovider.ProviderOllama || keys[c.provider] == "" {
			t.Errorf("%s: provider %q is Ollama or has no key variable", c.id, c.provider)
		}
		if c.cause == causeThinking && len(c.set) == 0 {
			t.Errorf("%s: a thinking case that sets no field", c.id)
		}
		if c.cause == causeOther {
			others[string(c.provider)+"/"+string(c.route)]++
		}
	}
	if counts[causeThinking] != 29 || counts[causeOther] != 22 || len(cases) != 51 {
		t.Errorf("%d thinking and %d other-cause cases of %d; want 29, 22 and 51", counts[causeThinking],
			counts[causeOther], len(cases))
	}
	if len(others) != len(refusalRoutes) || len(refusalRoutes) != 11 {
		t.Errorf("other causes on %d providers and routes, of %d; want 11", len(others), len(refusalRoutes))
	}
	for route, n := range others {
		if n != 2 {
			t.Errorf("%s: %d other-cause case(s); want 2", route, n)
		}
	}
}

// TestRefusalPaths_0062: a set makes the objects on its way; a drop of a
// field that is not there changes nothing.
func TestRefusalPaths_0062(t *testing.T) {
	body := map[string]any{"a": map[string]any{"keep": 1}}
	setPath(body, "a.b.c", "v")
	dropPath(body, "a.x.y")
	dropPath(body, "a.keep")
	got, _ := json.Marshal(body)
	if want := `{"a":{"b":{"c":"v"}}}`; string(got) != want {
		t.Errorf("body %s; want %s", got, want)
	}
}

// TestRefusalRecord_Write_0062: a skipped case's record is written to
// <wire>/<case>.json with its outcome, and no send.
func TestRefusalRecord_Write_0062(t *testing.T) {
	dir := t.TempDir()
	c := refusalCases()[0]
	if err := writeRefusalRecord(dir, c.skipped()); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, c.wire, c.id+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["case"] != c.id || got["outcome"] != outcomeSkipped || got["sends"] != float64(0) || got["status"] != nil {
		t.Errorf("record %s; want case %s, skipped, 0 sends, no status", raw, c.id)
	}
}

// TestRefusalBody_Bound_0062: a body over 4 KiB is cut to it, whole runes
// only. The leading byte puts each two-byte rune's start at an odd offset,
// so the bound falls inside one.
func TestRefusalBody_Bound_0062(t *testing.T) {
	got := boundRefusalBody([]byte("a" + strings.Repeat("é", refusalBodyLimit)))
	if len(got) > refusalBodyLimit || len(got) < refusalBodyLimit-1 || !utf8.ValidString(got) {
		t.Errorf("bound body: %d bytes, valid UTF-8 %v; want at most %d, whole runes", len(got), utf8.ValidString(got),
			refusalBodyLimit)
	}
	if short := boundRefusalBody([]byte("short")); short != "short" {
		t.Errorf("short body %q; want it whole", short)
	}
}

// TestRefusalRecorder_OutsideModule_0062: the output directory must be
// outside the module.
func TestRefusalRecorder_OutsideModule_0062(t *testing.T) {
	for _, dir := range []string{".", "..", "testdata", filepath.Join("..", "docs")} {
		if err := outsideModule(dir); err == nil {
			t.Errorf("outsideModule(%q) = nil; want an error, it is inside the module", dir)
		}
	}
	if err := outsideModule(t.TempDir()); err != nil {
		t.Errorf("outsideModule(a temporary directory) = %v; want nil", err)
	}
}
