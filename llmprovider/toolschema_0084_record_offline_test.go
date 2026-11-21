package llmprovider_test

// 0084-PLAN Phase 1a's offline proofs of the recorder: its case table, its
// outcome for each reply, its redaction, its writer, its one printed line,
// the call allowlist that keeps it from printing anything else, and the scan
// for a key's shape. Every reply is replayed in process; no host is reached.
// Made-up keys are built at run time, so no key-shaped text is in this file.
//
// This file never runs in the recording, and is not checked by the call
// allowlist; it holds no init function and no package-level call, which
// TestToolSchemaRecord_CallAllowlist_0084 checks, since either would run
// during the recording.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// toolReplay serves each request from a function of the request, with no
// network.
type toolReplay func(req *http.Request) (status int, body string)

func (f toolReplay) RoundTrip(req *http.Request) (*http.Response, error) {
	status, body := f(req)
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	return &http.Response{StatusCode: status, Status: strconv.Itoa(status) + " " + http.StatusText(status),
		Header: h, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
}

// toolMade is a made-up run of characters: prefix, then unit n times.
func toolMade(prefix, unit string, n int) string { return prefix + strings.Repeat(unit, n) }

// toolFixtureField reads one field of a 0062 fixture under
// internal/wire/testdata/refusals-0062/.
func toolFixtureField(t *testing.T, rel, field string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("internal", "wire", "testdata", "refusals-0062", rel))
	if err != nil {
		t.Fatalf("fixture %s: %v", rel, err)
	}
	var rec map[string]any
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("fixture %s: %v", rel, err)
	}
	s, _ := rec[field].(string)
	if s == "" {
		t.Fatalf("fixture %s: no %s", rel, field)
	}
	return s
}

// toolCaseByID returns the case with the id.
func toolCaseByID(t *testing.T, id string) toolCase {
	t.Helper()
	for _, c := range toolCases() {
		if c.id == id {
			return c
		}
	}
	t.Fatalf("no case %s", id)
	return toolCase{}
}

// toolReadAll returns the text of every file under dir, by path.
func toolReadAll(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		out[p] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// toolBuffer is an emit function that keeps what it is given.
func toolBuffer(buf *bytes.Buffer) emitFunc {
	return func(format string, args ...any) { fmt.Fprintf(buf, format+"\n", args...) }
}

// toolEach is a caseRunner that gives every send the same buffer.
func toolEach(buf *bytes.Buffer) caseRunner {
	return func(c toolCase, run func(emit emitFunc)) { run(toolBuffer(buf)) }
}

// TestToolSchemaRecord_Runner_0084 (ST67): recordToolCases makes every send,
// the first and the second, through the runner it is given: one call per
// case, and one more for each case that was other, which is what lets the
// live test give each a subtest, and a rerun of one case run its second send.
func TestToolSchemaRecord_Runner_0084(t *testing.T) {
	t.Setenv(llmprovider.ProviderEnvVars()[llmprovider.ProviderClaude], toolMade("k-", "Rt5v", 8))
	refusal := toolFixtureField(t, "messages/claude-string-schema-tool.json", "body")
	const busy = `{"type":"error","error":{"type":"overloaded_error","message":"busy"}}`
	calls := 0
	reply := toolReplay(func(*http.Request) (int, string) {
		calls++
		if calls == 1 {
			return http.StatusBadRequest, refusal
		}
		return http.StatusServiceUnavailable, busy
	})
	var ran []string
	each := func(c toolCase, run func(emitFunc)) {
		ran = append(ran, c.id)
		run(func(string, ...any) {})
	}
	cases := []toolCase{toolCaseByID(t, "claude-root-string"), toolCaseByID(t, "claude-O1-root-union")}
	if failed := recordToolCases(cases, t.TempDir(), reply, each); len(failed) != 0 {
		t.Errorf("%d records not written", len(failed))
	}
	if want := []string{"claude-root-string", "claude-O1-root-union", "claude-O1-root-union"}; !slices.Equal(ran, want) {
		t.Errorf("the runner ran %v; want %v", ran, want)
	}
}

// TestToolSchemaRecord_Cases_0084: the case table is 57 cases, each id is
// <route>-<shape>, every tool list starts with the valid control tool noop,
// so a refused tool sits at index 1, and grok2 runs O6's three alone.
func TestToolSchemaRecord_Cases_0084(t *testing.T) {
	cases := toolCases()
	if len(cases) != 57 {
		t.Errorf("%d cases; want 57 (nine routes of six, and grok2's three)", len(cases))
	}
	seen := map[string]bool{}
	grok2 := 0
	for _, c := range cases {
		if c.id != c.route+"-"+c.shape || seen[c.id] {
			t.Errorf("case id %q: not <route>-<shape> or repeated", c.id)
		}
		seen[c.id] = true
		if len(c.tools) == 0 || c.tools[0].Name != "noop" {
			t.Errorf("%s: tools %v; want noop first", c.id, c.tools)
		}
		if c.route == "grok2" {
			grok2++
		}
		if c.history != (c.shape == "O5-history-dropped") || (c.history && len(c.tools) != 1) {
			t.Errorf("%s: history %v, %d tools", c.id, c.history, len(c.tools))
		}
		if got := len(c.request().Tools); got != len(c.tools) {
			t.Errorf("%s: request carries %d tools; want %d", c.id, got, len(c.tools))
		}
	}
	if grok2 != 3 {
		t.Errorf("grok2 runs %d cases; want 3", grok2)
	}
}

// TestToolSchemaRecord_InsideModule_0084: the output directory must be
// outside the module.
func TestToolSchemaRecord_InsideModule_0084(t *testing.T) {
	if !toolInsideModule(".") || !toolInsideModule(filepath.Join("..", "docs")) {
		t.Error("a directory inside the module was taken for outside")
	}
	if toolInsideModule(t.TempDir()) {
		t.Error("a temporary directory was taken for inside the module")
	}
}

// TestToolSchemaRecord_Redacts_0084 (0084-MADR ST46): a record from F4's
// refusal (Claude's) into which a made-up key of each shape, a bearer value,
// an api_key pair, a request id, a UUID, an e-mail address and an
// organization id are planted (a named edit of F4, in its body, and in the
// error's Message and Code) holds none of them in any field of the written
// file, and holds req_REDACTED.
func TestToolSchemaRecord_Redacts_0084(t *testing.T) {
	xai, proj := toolMade("xai-", "Zq8", 16), toolMade("sk-proj-", "Ab3_x", 8)
	bearerValue := strings.Repeat("Qw9z", 10)
	kvValue := strings.Repeat("Mn4k", 6)
	req := toolMade("req_", "0Ab1", 6)
	uuid, email, org := "3f2b8c1e-9a47-4d65-b0e2-7c91a5d3e8f4", "jane.roe@acme-mail.net", "org-Ab12Cd34Ef56"
	planted := []string{xai, proj, bearerValue, kvValue, req, uuid, email, org}

	echo := strings.Join([]string{xai, proj, "Bearer " + bearerValue, `\"api_key\": \"` + kvValue + `\"`, uuid, email, org}, " ")
	f4 := toolFixtureField(t, "messages/claude-string-schema-tool.json", "body")
	const marker = `"request_id":"req_REDACTED"`
	if strings.Count(f4, marker) != 1 {
		t.Fatalf("F4's body has no %s", marker)
	}
	body := strings.Replace(f4, marker, `"request_id":"`+req+`","echo":"`+echo+`"`, 1)
	apiErr := &llmprovider.APIError{Provider: "claude", Status: http.StatusBadRequest,
		Code:    "invalid_request_error " + org + " " + uuid + " " + email,
		Message: "tools.1.custom.input_schema.type: Input should be 'object' for " + org + " " + uuid + " " + email}

	c := toolCaseByID(t, "claude-root-string")
	r := toolFill(c.record(), toolReply{err: apiErr, status: http.StatusBadRequest, body: []byte(body)}, false)
	dir := t.TempDir()
	if err := writeToolRecord(dir, r); err != nil {
		t.Fatal(err)
	}
	files := toolReadAll(t, dir)
	if len(files) != 1 {
		t.Fatalf("%d files written; want 1", len(files))
	}
	for p, text := range files {
		for _, v := range planted {
			if strings.Contains(text, v) {
				t.Errorf("%s holds a planted value (%d bytes, starting %q)", filepath.Base(p), len(v), v[:4])
			}
		}
		if !strings.Contains(text, "req_REDACTED") {
			t.Errorf("%s holds no req_REDACTED", filepath.Base(p))
		}
	}
	for field, text := range map[string]string{"Message": r.Message, "Code": r.Code, "Body": r.Body} {
		for _, v := range planted {
			if strings.Contains(text, v) {
				t.Errorf("%s holds a planted value (%d bytes, starting %q)", field, len(v), v[:4])
			}
		}
	}
	if !strings.Contains(r.Message, "org-REDACTED") || !strings.Contains(r.Code, "user@example.com") {
		t.Errorf("Message %q and Code %q: want the fixture rules applied", r.Message, r.Code)
	}
}

// claudeRefusal is F4's refusal with the tool at index 1 and a made-up key
// echoed in a field the recorder must never print.
func claudeRefusal(t *testing.T, echo string) toolReplay {
	t.Helper()
	body := toolFixtureField(t, "messages/claude-string-schema-tool.json", "body")
	body = strings.Replace(body, `"request_id":"req_REDACTED"`, `"request_id":"req_REDACTED","echo":"`+echo+`"`, 1)
	return func(*http.Request) (int, string) { return http.StatusBadRequest, body }
}

// TestToolSchemaRecord_LogLine_0084 (ST47, ST79): a case whose reply holds a
// made-up key prints exactly the five-field line and nothing else.
func TestToolSchemaRecord_LogLine_0084(t *testing.T) {
	key := toolMade("sk-ant-api03-", "Ab1x", 6)
	t.Setenv(llmprovider.ProviderEnvVars()[llmprovider.ProviderClaude], toolMade("k-", "Rt5v", 8))
	c := toolCaseByID(t, "claude-root-string")
	var buf bytes.Buffer
	if _, ok := recordToolCase(c, t.TempDir(), claudeRefusal(t, key), false, toolBuffer(&buf)); !ok {
		t.Fatal("the record was not written")
	}
	if want := "claude-root-string claude claude-sonnet-5: status 400, refused\n"; buf.String() != want {
		t.Errorf("printed %q; want %q", buf.String(), want)
	}
}

// toolClassify classifies a reply as the SDK does and asserts it is an error.
func toolClassify(t *testing.T, provider string, status int, body string, hdr map[string]string) error {
	t.Helper()
	h := http.Header{}
	for k, v := range hdr {
		h.Set(k, v)
	}
	err := llmprovider.ClassifyHTTPError(provider, &http.Response{StatusCode: status, Header: h,
		Body: io.NopCloser(strings.NewReader(body))})
	if err == nil {
		t.Fatalf("%s %d: no error", provider, status)
	}
	return err
}

// failingTransport fails the test if any request reaches it.
func failingTransport(t *testing.T) toolReplay {
	return func(*http.Request) (int, string) {
		t.Error("a request reached the transport")
		return http.StatusInternalServerError, "{}"
	}
}

// TestToolSchemaRecord_Outcome_0084 (ST67, ST68, ST70, ST78): each row of
// the corrected outcome table; an unset, empty and blank key; and the
// writer's rule that a refused or accepted record is never replaced and a
// skipped one is written only where there is none.
func TestToolSchemaRecord_Outcome_0084(t *testing.T) {
	const plain = `{"type":"error","error":{"type":"invalid_request_error","message":"m"}}`
	xaiKey := toolClassify(t, "grok", 400, `{"code":"invalid-argument","error":"Incorrect API key provided."}`, nil)
	geminiKey := toolClassify(t, "gemini", 400,
		`{"error":{"code":400,"message":"API key not valid.","status":"INVALID_ARGUMENT","details":[{"reason":"API_KEY_INVALID"}]}}`, nil)
	payment := toolClassify(t, "claude", 402, `{"type":"error","error":{"type":"billing_error","message":"m"}}`, nil)
	retry := toolClassify(t, "claude", 400, plain, map[string]string{"x-should-retry": "true"})
	// The setup is the evidence: the unwrap chain says invalid request for
	// each of these, and only the error's own kind tells them apart.
	for name, c := range map[string]struct {
		err  error
		kind error
	}{"xai key": {xaiKey, llmprovider.ErrAuthFailure}, "gemini key": {geminiKey, llmprovider.ErrAuthFailure},
		"402": {payment, llmprovider.ErrQuotaExhausted}, "should-retry": {retry, llmprovider.ErrProviderUnavailable}} {
		if !errors.Is(c.err, c.kind) {
			t.Fatalf("%s: not %v; the case is not what the test says", name, c.kind)
		}
	}
	for _, c := range []struct {
		name    string
		err     error
		outcome string
		reason  string
	}{
		{"no error", nil, toolOutcomeAccepted, ""},
		{"incomplete 200", &llmprovider.APIError{Kind: llmprovider.ErrIncomplete}, toolOutcomeAccepted, ""},
		{"empty answer 200", &llmprovider.APIError{Kind: llmprovider.ErrEmptyAnswer}, toolOutcomeAccepted, ""},
		{"400 invalid request", toolClassify(t, "claude", 400, plain, nil), toolOutcomeRefused, ""},
		{"422, status-mapped", &llmprovider.APIError{Status: 422}, toolOutcomeRefused, ""},
		{"400 context overflow", &llmprovider.APIError{Status: 400, Kind: llmprovider.ErrContextOverflow}, toolOutcomeRefused, ""},
		{"400 workspace", &llmprovider.APIError{Status: 400, Kind: llmprovider.ErrWorkspaceRequired}, toolOutcomeRefused, ""},
		{"wrapped 400", fmt.Errorf("wrapped: %w", toolClassify(t, "claude", 400, plain, nil)), toolOutcomeRefused, ""},
		{"xai 400 key refusal", xaiKey, toolOutcomeUnreachable, ""},
		{"gemini 400 key refusal", geminiKey, toolOutcomeUnreachable, ""},
		{"402 quota", payment, toolOutcomeUnreachable, ""},
		{"400 x-should-retry", retry, toolOutcomeUnreachable, ""},
		{"302", toolClassify(t, "claude", 302, "x", nil), toolOutcomeUnreachable, ""},
		{"401", &llmprovider.APIError{Status: 401}, toolOutcomeUnreachable, ""},
		{"403", &llmprovider.APIError{Status: 403}, toolOutcomeUnreachable, ""},
		{"404", &llmprovider.APIError{Status: 404}, toolOutcomeUnreachable, ""},
		{"429", &llmprovider.APIError{Status: 429}, toolOutcomeUnreachable, ""},
		{"not permitted 4xx", &llmprovider.APIError{Status: 451, Kind: llmprovider.ErrNotPermitted}, toolOutcomeUnreachable, ""},
		{"408", &llmprovider.APIError{Status: 408}, toolOutcomeOther, ""},
		{"409", &llmprovider.APIError{Status: 409}, toolOutcomeOther, ""},
		{"500", &llmprovider.APIError{Status: 500}, toolOutcomeOther, ""},
		{"503", &llmprovider.APIError{Status: 503}, toolOutcomeOther, ""},
		{"599", &llmprovider.APIError{Status: 599}, toolOutcomeOther, ""},
		{"failed 200", &llmprovider.APIError{Kind: llmprovider.ErrProviderUnavailable}, toolOutcomeOther, ""},
		{"transport", errors.New("boom"), toolOutcomeUnreachable, toolReasonTransport},
	} {
		if outcome, reason := toolOutcome(c.err); outcome != c.outcome || reason != c.reason {
			t.Errorf("%s: %s, %q; want %s, %q", c.name, outcome, reason, c.outcome, c.reason)
		}
	}
	// The second send: a case still other is unreachable.
	if r := toolFill(toolCaseByID(t, "grok-root-string").record(),
		toolReply{err: &llmprovider.APIError{Status: 503}, status: 503}, true); r.Outcome != toolOutcomeUnreachable {
		t.Errorf("a second send still other: %s; want unreachable", r.Outcome)
	}

	// A key unset, empty or all spaces: skipped, no request sent, the case
	// line printed, the record written.
	c := toolCaseByID(t, "openai-root-string")
	envName := llmprovider.ProviderEnvVars()[llmprovider.ProviderOpenAI]
	for name, set := range map[string]func(){
		"unset": func() { t.Setenv(envName, "x"); os.Unsetenv(envName) },
		"empty": func() { t.Setenv(envName, "") },
		"blank": func() { t.Setenv(envName, " \t  ") },
	} {
		set()
		dir := t.TempDir()
		var buf bytes.Buffer
		r, ok := recordToolCase(c, dir, failingTransport(t), false, toolBuffer(&buf))
		if !ok || r.Outcome != toolOutcomeSkipped || r.Reason != toolReasonNoKey {
			t.Errorf("%s key: %v, %s, %q; want skipped, %q", name, ok, r.Outcome, r.Reason, toolReasonNoKey)
		}
		if got, want := buf.String(), "openai-root-string openai gpt-6-luna: status 0, skipped\n"; got != want {
			t.Errorf("%s key printed %q; want %q", name, got, want)
		}
		if files := toolReadAll(t, dir); len(files) != 1 {
			t.Errorf("%s key: %d files written; want 1", name, len(files))
		}
	}

	// The writer's rule (ST70).
	dir := t.TempDir()
	rec := func(id, outcome string) toolRecord {
		r := toolCaseByID(t, id).record()
		r.Outcome = outcome
		return r
	}
	outcomeOnDisk := func(id string) string {
		raw, err := os.ReadFile(filepath.Join(dir, toolCaseByID(t, id).wire, id+".json"))
		if err != nil {
			return "none"
		}
		for _, o := range []string{toolOutcomeAccepted, toolOutcomeRefused, toolOutcomeUnreachable, toolOutcomeOther, toolOutcomeSkipped} {
			if strings.Contains(string(raw), "\n  \"outcome\": \""+o+"\"") {
				return o
			}
		}
		return "unknown"
	}
	write := func(id, outcome string) {
		if err := writeToolRecord(dir, rec(id, outcome)); err != nil {
			t.Fatal(err)
		}
	}
	for _, step := range []struct{ id, write, want string }{
		{"claude-root-string", toolOutcomeSkipped, toolOutcomeSkipped}, // skipped where there is none
		{"claude-root-string", toolOutcomeOther, toolOutcomeOther},     // replaces a skipped one
		{"claude-root-string", toolOutcomeSkipped, toolOutcomeOther},   // skipped never replaces
		{"claude-root-string", toolOutcomeUnreachable, toolOutcomeUnreachable},
		{"claude-root-string", toolOutcomeRefused, toolOutcomeRefused},     // replaces an unreachable one
		{"claude-root-string", toolOutcomeUnreachable, toolOutcomeRefused}, // never replaces refused
		{"claude-root-string", toolOutcomeAccepted, toolOutcomeRefused},
		{"claude-root-string", toolOutcomeSkipped, toolOutcomeRefused},
		{"gemini-root-string", toolOutcomeAccepted, toolOutcomeAccepted},
		{"gemini-root-string", toolOutcomeOther, toolOutcomeAccepted}, // never replaces accepted
		{"gemini-root-string", toolOutcomeRefused, toolOutcomeAccepted},
	} {
		write(step.id, step.write)
		if got := outcomeOnDisk(step.id); got != step.want {
			t.Errorf("%s: wrote %s over the record; it holds %s; want %s", step.id, step.write, got, step.want)
		}
	}
}

// TestToolSchemaRecord_NoLeak_0084 (ST77, ST79, ST80): the recorder's case
// path, against an in-process transport replaying F2 (OpenAI's refusal) with
// a made-up key-shaped string and a made-up org- id planted in its body and
// message, and a made-up key set in the route's variable: no record, output
// or log line holds the key or either planted value, and the line holds the
// five fields only.
func TestToolSchemaRecord_NoLeak_0084(t *testing.T) {
	envKey := toolMade("sk-proj-", "Gh7_q", 8)
	planted := toolMade("xai-", "Lm2", 16)
	org := "org-Zx98Yw76Vu54"
	t.Setenv(llmprovider.ProviderEnvVars()[llmprovider.ProviderOpenAI], envKey)
	reply := toolReplay(func(req *http.Request) (int, string) {
		if req.Header.Get("Authorization") != "Bearer "+envKey {
			t.Error("the key did not reach the provider")
		}
		msg := "Invalid schema for function 'record_city': " + planted + " of " + org
		return http.StatusBadRequest, `{"error":{"message":"` + msg + `","type":"invalid_request_error",` +
			`"param":"tools[1].parameters","code":"invalid_function_parameters","echo":"` + planted + " " + org + `"}}`
	})
	c := toolCaseByID(t, "openai-O1-root-union")
	dir := t.TempDir()
	var buf bytes.Buffer
	r, ok := recordToolCase(c, dir, reply, false, toolBuffer(&buf))
	if !ok || r.Outcome != toolOutcomeRefused || r.Status != http.StatusBadRequest {
		t.Fatalf("record %v, %s, %d; want written, refused, 400", ok, r.Outcome, r.Status)
	}
	texts := toolReadAll(t, dir)
	texts["output"] = buf.String()
	if len(texts) != 2 {
		t.Fatalf("%d texts; want the record and the output", len(texts))
	}
	for name, text := range texts {
		for _, v := range []string{envKey, planted, org} {
			if strings.Contains(text, v) {
				t.Errorf("%s holds a made-up value (starting %q)", filepath.Base(name), v[:4])
			}
		}
	}
	if want := "openai-O1-root-union openai gpt-6-luna: status 400, refused\n"; buf.String() != want {
		t.Errorf("printed %q; want %q", buf.String(), want)
	}
}

// TestToolSchemaRecord_FieldPerLine_0084 (ST83): every line of a written
// record lies under exactly one top-level key, the nearest line at or above
// it at two spaces' indent; the body, Message and Code are each one line, and
// the tools, nested, span several.
func TestToolSchemaRecord_FieldPerLine_0084(t *testing.T) {
	c := toolCaseByID(t, "claude-O2-two-bad")
	apiErr := &llmprovider.APIError{Status: 400, Code: "invalid_request_error", Message: "line one\nline two"}
	body := "{\n  \"error\": {\n    \"message\": \"tools.1\"\n  }\n}"
	r := toolFill(c.record(), toolReply{err: apiErr, status: 400, body: []byte(body)}, false)
	dir := t.TempDir()
	if err := writeToolRecord(dir, r); err != nil {
		t.Fatal(err)
	}
	var text string
	for _, v := range toolReadAll(t, dir) {
		text = v
	}
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if len(lines) < 10 || lines[0] != "{" || lines[len(lines)-1] != "}" {
		t.Fatalf("%d lines, first %q: want an indented object", len(lines), lines[0])
	}
	keyAt := map[int]string{}
	var key string
	for i, line := range lines[1 : len(lines)-1] {
		n := i + 1
		switch {
		case strings.HasPrefix(line, "  \""):
			key, _, _ = strings.Cut(strings.TrimPrefix(line, "  "), ":")
			keyAt[n] = key
		case strings.HasPrefix(line, "   "), strings.HasPrefix(line, "  ]"), strings.HasPrefix(line, "  }"):
			// Nested, or the closer of a nested value at the key's indent.
			if key == "" {
				t.Errorf("line %d lies under no top-level key", n+1)
			}
		default:
			t.Errorf("line %d is neither a top-level key nor nested under one", n+1)
		}
	}
	span := func(k string) int {
		for n, kk := range keyAt {
			if kk == k {
				next := len(lines) - 1
				for m := range keyAt {
					if m > n && m < next {
						next = m
					}
				}
				return next - n
			}
		}
		return 0
	}
	for _, k := range []string{`"body"`, `"message"`, `"code"`} {
		if got := span(k); got != 1 {
			t.Errorf("%s spans %d lines; want 1", k, got)
		}
	}
	if got := span(`"tools"`); got < 5 {
		t.Errorf("tools spans %d lines; want several", got)
	}
}

// TestToolSchemaRecord_SecondSend_0084 (ST67): recordToolCases, the loop the
// live test calls, sends a case that was other once more after the pass: still
// other, it is recorded unreachable; a refusal, refused; a case that was not
// other is sent once; and a record that cannot be written is returned, not
// lost.
func TestToolSchemaRecord_SecondSend_0084(t *testing.T) {
	t.Setenv(llmprovider.ProviderEnvVars()[llmprovider.ProviderClaude], toolMade("k-", "Rt5v", 8))
	refusal := toolFixtureField(t, "messages/claude-string-schema-tool.json", "body")
	const busy = `{"type":"error","error":{"type":"overloaded_error","message":"busy"}}`
	c := toolCaseByID(t, "claude-root-string")
	const line = "claude-root-string claude claude-sonnet-5: status "
	for _, tc := range []struct {
		name   string
		status []int
		body   []string
		calls  int
		want   string
		out    string
	}{
		{"503 then 503", []int{503, 503}, []string{busy, busy}, 2, toolOutcomeUnreachable,
			line + "503, other\n" + line + "503, unreachable\n"},
		{"503 then 400", []int{503, 400}, []string{busy, refusal}, 2, toolOutcomeRefused,
			line + "503, other\n" + line + "400, refused\n"},
		{"400 once", []int{400, 400}, []string{refusal, refusal}, 1, toolOutcomeRefused, line + "400, refused\n"},
	} {
		calls := 0
		reply := toolReplay(func(*http.Request) (int, string) {
			calls++
			return tc.status[min(calls, 2)-1], tc.body[min(calls, 2)-1]
		})
		dir := t.TempDir()
		var buf bytes.Buffer
		if failed := recordToolCases([]toolCase{c}, dir, reply, toolEach(&buf)); len(failed) != 0 {
			t.Errorf("%s: %d records not written", tc.name, len(failed))
		}
		if calls != tc.calls {
			t.Errorf("%s: %d sends; want %d", tc.name, calls, tc.calls)
		}
		var text string
		for _, v := range toolReadAll(t, dir) {
			text = v
		}
		if !strings.Contains(text, "\n  \"outcome\": \""+tc.want+"\"") {
			t.Errorf("%s: the record is not %s", tc.name, tc.want)
		}
		if buf.String() != tc.out {
			t.Errorf("%s: printed %q; want %q", tc.name, buf.String(), tc.out)
		}
	}

	// A directory that is a file: the write fails, and the case is returned.
	file := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	failed := recordToolCases([]toolCase{c}, file, toolReplay(func(*http.Request) (int, string) {
		return 400, refusal
	}), toolEach(&buf))
	if len(failed) != 1 || failed[0].id != c.id {
		t.Errorf("an unwritable directory: %d cases returned; want %s", len(failed), c.id)
	}
}

// TestToolSchemaRecord_NoModelText_0084 (ST46): a record never holds the
// model's text: a 200 that the SDK found cut short, with a body of model
// text, gives a record with no body; an accepted answer gives counts, not its
// text; and neither text is in the written file.
func TestToolSchemaRecord_NoModelText_0084(t *testing.T) {
	text := "the model said " + strings.Repeat("Zebra", 4)
	c := toolCaseByID(t, "claude-root-string")
	body := `{"content":[{"type":"text","text":"` + text + `"}],"stop_reason":"max_tokens"}`
	r := toolFill(c.record(), toolReply{err: &llmprovider.APIError{Kind: llmprovider.ErrIncomplete, Reason: "max_output_tokens"},
		status: http.StatusOK, body: []byte(body)}, false)
	if r.Outcome != toolOutcomeAccepted || r.Status != http.StatusOK || r.Body != "" {
		t.Errorf("a cut-short 200: %s, %d, body of %d bytes; want accepted, 200, no body", r.Outcome, r.Status, len(r.Body))
	}
	dir := t.TempDir()
	if err := writeToolRecord(dir, r); err != nil {
		t.Fatal(err)
	}
	for _, v := range toolReadAll(t, dir) {
		if strings.Contains(v, "Zebra") || strings.Contains(v, "content") {
			t.Error("the written record of a cut-short 200 holds the model's text")
		}
	}

	// An accepted answer, through the case path, against a replayed message.
	t.Setenv(llmprovider.ProviderEnvVars()[llmprovider.ProviderClaude], toolMade("k-", "Rt5v", 8))
	answer := `{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-5",` +
		`"content":[{"type":"text","text":"` + text + `"}],"stop_reason":"end_turn",` +
		`"usage":{"input_tokens":3,"output_tokens":2}}`
	dir = t.TempDir()
	var buf bytes.Buffer
	r, ok := recordToolCase(c, dir, toolReplay(func(*http.Request) (int, string) { return http.StatusOK, answer }), false, toolBuffer(&buf))
	if !ok || r.Outcome != toolOutcomeAccepted || r.InputTokens != 3 || r.OutputTokens != 2 || r.Body != "" {
		t.Fatalf("an accepted answer: %v, %s, tokens %d and %d, body %d bytes; want accepted, 3 and 2, none",
			ok, r.Outcome, r.InputTokens, r.OutputTokens, len(r.Body))
	}
	texts := toolReadAll(t, dir)
	texts["output"] = buf.String()
	for name, v := range texts {
		if strings.Contains(v, "Zebra") {
			t.Errorf("%s holds the model's text", filepath.Base(name))
		}
	}
}

// ---------------------------------------------------------------------------
// The call allowlist (0084-PLAN Phase 1a, step 5, as amended on 2026-11-20
// and 2026-11-21).

const toolModulePath = "github.com/maccavelli/go-llmprovider-sdk"

// toolCheckedFiles are the code that runs during the recording.
var toolCheckedFiles = []string{"toolschema_0084_record_test.go", "live_0084_toolschema_measure_test.go",
	"toolschema_0084_scan_test.go"}

const (
	toolRecordFile  = "toolschema_0084_record_test.go"
	toolScanFile    = "toolschema_0084_scan_test.go"
	toolOfflineFile = "toolschema_0084_record_offline_test.go"
	toolWriteFunc   = "writeToolRecord"
)

// toolAllowedImports, each for a named need (the PLAN's list).
var toolAllowedImports = map[string]bool{
	"context": true, "errors": true, "net/http": true, "io": true, "bytes": true, "os": true,
	"path/filepath": true, "strings": true, "regexp": true, "unicode/utf8": true, "time": true,
	"testing": true, "io/fs": true,
	toolModulePath + "/internal/redact":                true,
	toolModulePath + "/internal/wiretest":              true,
	toolModulePath + "/llmprovider":                    true,
	toolModulePath + "/llmprovider/providers":          true,
	toolModulePath + "/llmprovider/providers/opencode": true,
}

// toolAllowedCalls are the callees, by types.Func.FullName, that a call may
// resolve to outside the checked files. testing declares Logf, Fatal, Fatalf,
// Skip and Helper on its unexported common type (the owner's reading of
// 2026-11-21).
var toolAllowedCalls = map[string]bool{
	"context.Background": true, "context.WithTimeout": true,
	"errors.Is": true, "errors.AsType": true,
	"io.ReadAll": true, "io.LimitReader": true, "io.NopCloser": true,
	"bytes.NewReader": true,
	"os.Getenv":       true, "os.MkdirAll": true, "os.WriteFile": true, "os.ReadFile": true,
	"path/filepath.Join": true, "path/filepath.Dir": true, "path/filepath.Abs": true,
	"path/filepath.Rel": true, "path/filepath.Base": true, "path/filepath.WalkDir": true,
	"strings.TrimSpace": true, "strings.HasPrefix": true, "strings.Contains": true,
	"strings.Cut": true, "strings.Split": true,
	"regexp.MustCompile":                                         true,
	"unicode/utf8.RuneStart":                                     true,
	toolModulePath + "/internal/redact.String":                   true,
	toolModulePath + "/internal/redact.Redact":                   true,
	toolModulePath + "/internal/wiretest.Encode":                 true,
	toolModulePath + "/llmprovider.WithAPIKey":                   true,
	toolModulePath + "/llmprovider.WithModel":                    true,
	toolModulePath + "/llmprovider.WithModelProbes":              true,
	toolModulePath + "/llmprovider.WithoutModelMetadata":         true,
	toolModulePath + "/llmprovider.WithHTTPClient":               true,
	toolModulePath + "/llmprovider.ProviderEnvVars":              true,
	toolModulePath + "/llmprovider/providers.New":                true,
	toolModulePath + "/llmprovider/providers/opencode.WithRoute": true,
	"(*regexp.Regexp).ReplaceAllString":                          true,
	"(*regexp.Regexp).MatchString":                               true,
	"(" + toolModulePath + "/llmprovider.Provider).Generate":     true,
	"(net/http.RoundTripper).RoundTrip":                          true,
	"(io.Closer).Close":                                          true,
	"(*testing.common).Errorf":                                   true,
	"(*testing.common).Logf":                                     true,
	"(*testing.common).Fatal":                                    true,
	"(*testing.common).Fatalf":                                   true,
	"(*testing.common).Skip":                                     true,
	"(*testing.common).Helper":                                   true,
	"(*testing.T).Run":                                           true,
	"(io/fs.DirEntry).IsDir":                                     true,
}

// toolArgRuled are the calls whose arguments must be constants or named
// fields; t.Run is ruled for its name argument alone.
var toolArgRuled = map[string]bool{
	"(*testing.common).Errorf": true,
	"(*testing.common).Logf":   true, "(*testing.common).Fatal": true, "(*testing.common).Fatalf": true,
	"(*testing.common).Skip": true, "(*testing.T).Run": true,
}

// toolFieldsByType are the fields a printing call may take, by the type that
// declares them: the case's id, route and model, the record's status and
// outcome; in the scan file, a hit's file, line and label. A field of another
// type, or of the same name, is refused (the review's B1).
var toolFieldsByType = map[string]map[string]bool{
	"toolCase":   {"id": true, "route": true, "model": true},
	"toolRecord": {"Status": true, "Outcome": true},
	"scanHit":    {"file": true, "line": true, "label": true},
}

var toolAllowedBuiltins = map[string]bool{"len": true, "append": true, "make": true, "copy": true, "min": true, "max": true}

// toolChecker type-checks files and reports every call or import the
// allowlist does not name, as "<file>:<line>: <what>", never an argument's
// value.
type toolChecker struct {
	fset  *token.FileSet
	imp   types.Importer
	goVer string
	calls int // calls inspected, so that the guard is not vacuous
	emits int // calls through emitFunc
}

func newToolChecker(t *testing.T) *toolChecker {
	t.Helper()
	// Loading net/http from source needs no C toolchain with cgo off.
	old := build.Default.CgoEnabled
	build.Default.CgoEnabled = false
	t.Cleanup(func() { build.Default.CgoEnabled = old })
	fset := token.NewFileSet()
	return &toolChecker{fset: fset, imp: importer.ForCompiler(fset, "source", nil), goVer: toolGoVersion(t)}
}

// toolGoVersion is the module's language version, "go1.N", from go.mod.
func toolGoVersion(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if v, ok := strings.CutPrefix(line, "go "); ok {
			if parts := strings.Split(strings.TrimSpace(v), "."); len(parts) >= 2 {
				return "go" + parts[0] + "." + parts[1]
			}
		}
	}
	t.Fatal("no go directive in go.mod")
	return ""
}

func toolUnwrap(e ast.Expr) ast.Expr {
	for {
		switch x := e.(type) {
		case *ast.ParenExpr:
			e = x.X
		case *ast.IndexExpr:
			e = x.X
		case *ast.IndexListExpr:
			e = x.X
		default:
			return e
		}
	}
}

// check parses and type-checks files (name to source) as one package, in this
// package's directory so that its imports resolve.
func (ck *toolChecker) check(t *testing.T, files map[string]string) []string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	var parsed []*ast.File
	for _, name := range toolCheckedFiles {
		f, err := parser.ParseFile(ck.fset, filepath.Join(dir, name), files[name], 0)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		parsed = append(parsed, f)
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Uses: map[*ast.Ident]types.Object{}, Defs: map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{}, Instances: map[*ast.Ident]types.Instance{}}
	conf := types.Config{Importer: ck.imp, GoVersion: ck.goVer, Error: func(err error) {
		if te, ok := errors.AsType[types.Error](err); ok {
			pos := te.Fset.Position(te.Pos)
			out = append(out, fmt.Sprintf("%s:%d: type error: %s", filepath.Base(pos.Filename), pos.Line, te.Msg))
			return
		}
		out = append(out, "type error: "+err.Error())
	}}
	pkg, _ := conf.Check("llmprovider_test", ck.fset, parsed, info)
	for _, f := range parsed {
		name := filepath.Base(ck.fset.Position(f.Package).Filename)
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			if !toolAllowedImports[path] {
				out = append(out, fmt.Sprintf("%s:%d: import %q is not allowed", name, ck.fset.Position(imp.Pos()).Line, path))
			}
		}
		for _, d := range f.Decls {
			encl, pathObj := "", types.Object(nil)
			if fd, ok := d.(*ast.FuncDecl); ok {
				// Only the record file's function with no receiver is the
				// write function; a method or a function of the same name
				// elsewhere is any other function (the review's B2).
				encl = "other:" + fd.Name.Name
				if fd.Recv == nil && fd.Name.Name == toolWriteFunc && name == toolRecordFile && fd.Body != nil {
					encl = toolWriteFunc
					var dirObj types.Object
					dirObj, pathObj = ck.writeObjects(info, fd)
					out = append(out, ck.writeAssigns(info, fd, name, dirObj, pathObj)...)
				}
			}
			ast.Inspect(d, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					out = append(out, ck.call(info, pkg, call, name, encl, pathObj)...)
				}
				return true
			})
		}
	}
	return out
}

// toolDirParam is the identifier of fd's first parameter, or nil when it has
// none or it is unnamed.
func toolDirParam(fd *ast.FuncDecl) *ast.Ident {
	if ps := fd.Type.Params.List; len(ps) > 0 && len(ps[0].Names) > 0 {
		return ps[0].Names[0]
	}
	return nil
}

// isJoinOf reports whether e is filepath.Join(dir, ...), dir being the
// object dirObj.
func (ck *toolChecker) isJoinOf(info *types.Info, e ast.Expr, dirObj types.Object) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok || len(call.Args) == 0 || ck.callee(info, call) != "path/filepath.Join" {
		return false
	}
	id, ok := call.Args[0].(*ast.Ident)
	return ok && dirObj != nil && info.Uses[id] == dirObj
}

// writeObjects returns the objects of the write function's dir parameter and
// of the variable that `path := filepath.Join(dir, ...)` defines. Everything
// that follows is judged by object, not by name, so that a variable or a
// parameter of the same name elsewhere is another object (the re-review's
// N3 and N4).
func (ck *toolChecker) writeObjects(info *types.Info, fd *ast.FuncDecl) (dirObj, pathObj types.Object) {
	id := toolDirParam(fd)
	if id == nil {
		return nil, nil
	}
	dirObj = info.Defs[id]
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok || pathObj != nil || as.Tok != token.DEFINE || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			return true
		}
		if lhs, ok := as.Lhs[0].(*ast.Ident); ok && ck.isJoinOf(info, as.Rhs[0], dirObj) {
			pathObj = info.Defs[lhs]
		}
		return true
	})
	return dirObj, pathObj
}

// writeAssigns reports, in the write function, any assignment to the dir
// parameter, any assignment to the path variable but its one definition from
// filepath.Join(dir, ...), and the address of either being taken (the
// review's B3 and the re-review's N5).
func (ck *toolChecker) writeAssigns(info *types.Info, fd *ast.FuncDecl, file string, dirObj, pathObj types.Object) []string {
	var out []string
	report := func(n ast.Node, msg string) {
		out = append(out, fmt.Sprintf("%s:%d: %s", file, ck.fset.Position(n.Pos()).Line, msg))
	}
	objOf := func(e ast.Expr) types.Object {
		id, ok := e.(*ast.Ident)
		if !ok {
			return nil
		}
		if o := info.Defs[id]; o != nil {
			return o
		}
		return info.Uses[id]
	}
	watched := func(o types.Object) bool { return o != nil && (o == dirObj || o == pathObj) }
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.AssignStmt:
			defines := func(i int) bool {
				return s.Tok == token.DEFINE && len(s.Rhs) == len(s.Lhs) && ck.isJoinOf(info, s.Rhs[i], dirObj)
			}
			for i, lhs := range s.Lhs {
				o := objOf(lhs)
				switch {
				case o == nil:
				case o == dirObj:
					report(s, "assignment to dir")
				case o == pathObj && !defines(i):
					report(s, "path reassigned from something but filepath.Join(dir, ...)")
				}
			}
		case *ast.IncDecStmt:
			if watched(objOf(s.X)) {
				report(s, "path or dir reassigned")
			}
		case *ast.RangeStmt:
			for _, e := range []ast.Expr{s.Key, s.Value} {
				if e != nil && watched(objOf(e)) {
					report(s, "path or dir reassigned")
				}
			}
		case *ast.UnaryExpr:
			if s.Op == token.AND && watched(objOf(s.X)) {
				report(s, "address of dir or path taken")
			}
		}
		return true
	})
	return out
}

// callee is the FullName of the function a call names, or "".
func (ck *toolChecker) callee(info *types.Info, call *ast.CallExpr) string {
	var obj types.Object
	switch x := toolUnwrap(call.Fun).(type) {
	case *ast.Ident:
		obj = info.Uses[x]
	case *ast.SelectorExpr:
		if sel := info.Selections[x]; sel != nil {
			obj = sel.Obj()
		} else {
			obj = info.Uses[x.Sel]
		}
	}
	if fn, ok := obj.(*types.Func); ok {
		return fn.Origin().FullName()
	}
	return ""
}

// argsOK reports whether each argument (the first alone for t.Run) is a
// constant or a named field of its declaring type, read from a variable.
func (ck *toolChecker) argsOK(info *types.Info, pkg *types.Package, file string, call *ast.CallExpr,
	firstOnly bool) bool {
	args := call.Args
	if firstOnly && len(args) > 1 {
		args = args[:1]
	}
	for _, a := range args {
		if tv, ok := info.Types[a]; ok && tv.Value != nil {
			continue
		}
		if !ck.namedField(info, pkg, file, a) {
			return false
		}
	}
	return true
}

// namedField reports whether e selects an allowed field of toolCase,
// toolRecord (the record and live files) or scanHit (the scan file), declared
// in the package, from a variable and not from a literal or a call.
func (ck *toolChecker) namedField(info *types.Info, pkg *types.Package, file string, e ast.Expr) bool {
	se, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	sel := info.Selections[se]
	if sel == nil || sel.Kind() != types.FieldVal {
		return false
	}
	recv := sel.Recv()
	if ptr, ok := recv.(*types.Pointer); ok {
		recv = ptr.Elem()
	}
	named, ok := recv.(*types.Named)
	if !ok || named.Obj().Pkg() != pkg || !toolFieldsByType[named.Obj().Name()][se.Sel.Name] {
		return false
	}
	if (named.Obj().Name() == "scanHit") != (file == toolScanFile) {
		return false
	}
	return ck.fromVariable(info, se.X)
}

// fromVariable reports whether e is a variable, or a field read from one.
func (ck *toolChecker) fromVariable(info *types.Info, e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.ParenExpr:
		return ck.fromVariable(info, x.X)
	case *ast.StarExpr:
		return ck.fromVariable(info, x.X)
	case *ast.Ident:
		_, ok := info.Uses[x].(*types.Var)
		return ok
	case *ast.SelectorExpr:
		sel := info.Selections[x]
		return sel != nil && sel.Kind() == types.FieldVal && ck.fromVariable(info, x.X)
	}
	return false
}

func (ck *toolChecker) call(info *types.Info, pkg *types.Package, call *ast.CallExpr, file, encl string,
	pathObj types.Object) []string {
	at := func(msg string) []string {
		return []string{fmt.Sprintf("%s:%d: %s", file, ck.fset.Position(call.Pos()).Line, msg)}
	}
	if tv, ok := info.Types[call.Fun]; ok && tv.IsType() {
		return nil // a conversion is not a call
	}
	ck.calls++
	fun := toolUnwrap(call.Fun)
	if _, ok := fun.(*ast.FuncLit); ok {
		return nil
	}
	var obj types.Object
	switch x := fun.(type) {
	case *ast.Ident:
		obj = info.Uses[x]
	case *ast.SelectorExpr:
		if sel := info.Selections[x]; sel != nil {
			obj = sel.Obj()
		} else {
			obj = info.Uses[x.Sel]
		}
	}
	switch o := obj.(type) {
	case *types.Builtin:
		if !toolAllowedBuiltins[o.Name()] {
			return at(o.Name() + ": builtin is not allowed")
		}
		return nil
	case *types.Var:
		if msg := ck.varCall(info, pkg, o, call, file); msg != "" {
			return at(msg)
		}
		return nil
	case *types.Func:
		fn := o.Origin()
		name := fn.FullName()
		if fn.Pkg() != pkg && !toolAllowedCalls[name] {
			return at(name + ": call is not on the allowlist")
		}
		var out []string
		switch name {
		case "os.WriteFile", "os.MkdirAll":
			if encl != toolWriteFunc {
				out = append(out, at(name+": outside the one write function")...)
			}
		case "os.ReadFile":
			if encl != toolWriteFunc && file != toolScanFile {
				out = append(out, at(name+": outside the write function and the scan file")...)
			}
		}
		if encl == toolWriteFunc && (name == "os.WriteFile" || name == "os.MkdirAll" || name == "os.ReadFile") &&
			!ck.pathArgOK(info, call.Args[0], pathObj) {
			out = append(out, at(name+": path argument is not built from filepath.Join(dir, ...)")...)
		}
		if toolArgRuled[name] && !ck.argsOK(info, pkg, file, call, name == "(*testing.T).Run") {
			out = append(out, at(name+": an argument is not a constant or an allowlisted field")...)
		}
		return out
	}
	return at("call that does not resolve")
}

// varCall judges a call through a variable of function type, and returns the
// reason it is refused, or "": only emitFunc, with constants and named fields;
// caseRunner, with a toolCase variable and a function literal; a run
// function, with t.Logf; and a context.CancelFunc.
func (ck *toolChecker) varCall(info *types.Info, pkg *types.Package, o *types.Var, call *ast.CallExpr,
	file string) string {
	named, _ := o.Type().(*types.Named)
	switch {
	case named != nil && named.Obj().Name() == "emitFunc" && named.Obj().Pkg() == pkg:
		ck.emits++
		if !ck.argsOK(info, pkg, file, call, false) {
			return "emitFunc: an argument is not a constant or an allowlisted field"
		}
		return ""
	case named != nil && named.Obj().Name() == "caseRunner" && named.Obj().Pkg() == pkg:
		ok := len(call.Args) == 2 && ck.isToolVar(info, pkg, call.Args[0], "toolCase")
		if ok {
			_, ok = call.Args[1].(*ast.FuncLit)
		}
		if !ok {
			return "caseRunner: the arguments are not a toolCase variable and a function literal"
		}
		return ""
	case ck.isRunFunc(pkg, o.Type()):
		if !ck.logfArg(info, call) {
			return o.Name() + ": the argument is not t.Logf"
		}
		return ""
	case named != nil && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "context" &&
		named.Obj().Name() == "CancelFunc":
		return ""
	}
	return o.Name() + ": call through a function value"
}

// pathArgOK: the variable that filepath.Join(dir, ...) defined, by object, or
// filepath.Dir of it.
func (ck *toolChecker) pathArgOK(info *types.Info, e ast.Expr, pathObj types.Object) bool {
	if pathObj == nil {
		return false
	}
	if id, ok := e.(*ast.Ident); ok {
		return info.Uses[id] == pathObj
	}
	if call, ok := e.(*ast.CallExpr); ok && ck.callee(info, call) == "path/filepath.Dir" && len(call.Args) == 1 {
		id, ok := call.Args[0].(*ast.Ident)
		return ok && info.Uses[id] == pathObj
	}
	return false
}

// isToolVar reports whether e is a variable of the package's type typeName.
func (ck *toolChecker) isToolVar(info *types.Info, pkg *types.Package, e ast.Expr, typeName string) bool {
	id, ok := e.(*ast.Ident)
	if !ok {
		return false
	}
	v, ok := info.Uses[id].(*types.Var)
	if !ok {
		return false
	}
	named, ok := v.Type().(*types.Named)
	return ok && named.Obj().Pkg() == pkg && named.Obj().Name() == typeName
}

// isRunFunc reports whether t is func(emitFunc), the type of a caseRunner's
// run parameter.
func (ck *toolChecker) isRunFunc(pkg *types.Package, t types.Type) bool {
	sig, ok := t.(*types.Signature)
	if !ok || sig.Params().Len() != 1 || sig.Results().Len() != 0 || sig.Variadic() {
		return false
	}
	named, ok := sig.Params().At(0).Type().(*types.Named)
	return ok && named.Obj().Pkg() == pkg && named.Obj().Name() == "emitFunc"
}

// logfArg reports whether a call's one argument is the method value t.Logf.
func (ck *toolChecker) logfArg(info *types.Info, call *ast.CallExpr) bool {
	if len(call.Args) != 1 {
		return false
	}
	se, ok := call.Args[0].(*ast.SelectorExpr)
	if !ok {
		return false
	}
	sel := info.Selections[se]
	if sel == nil || sel.Kind() != types.MethodVal {
		return false
	}
	fn, ok := sel.Obj().(*types.Func)
	return ok && fn.FullName() == "(*testing.common).Logf"
}

// toolOfflineShape reports an init function or a package-level call in src,
// the offline proofs file: either would run during the recording.
func toolOfflineShape(t *testing.T, src string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, toolOfflineFile, src, 0)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil && d.Name.Name == "init" {
				out = append(out, fmt.Sprintf("%s:%d: an init function", toolOfflineFile, fset.Position(d.Pos()).Line))
			}
		case *ast.GenDecl:
			ast.Inspect(d, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.FuncLit:
					return false
				case *ast.CallExpr:
					out = append(out, fmt.Sprintf("%s:%d: a package-level call", toolOfflineFile, fset.Position(n.Pos()).Line))
				}
				return true
			})
		}
	}
	return out
}

// toolEdit replaces old, which must occur once in file, with new.
type toolEdit struct{ file, old, new string }

// toolPlant is a break of the checked files, and the text its report must
// hold, so that it is caught for the reason it plants.
type toolPlant struct {
	name  string
	edits []toolEdit
	want  string
}

const (
	toolAnchorFn   = "func boundToolBody(s string) string {\n"
	toolAnchorLive = "func TestLive_ToolSchema_0084(t *testing.T) {\n"
	toolAnchorImp  = "import (\n"
)

var toolPlants = []toolPlant{
	{"os.Stderr.WriteString(err.Error())", []toolEdit{{toolRecordFile, toolAnchorFn,
		toolAnchorFn + "\tvar err error\n\tos.Stderr.WriteString(err.Error())\n"}}, "(*os.File).WriteString"},
	{"io.WriteString(os.Stdout, s)", []toolEdit{{toolRecordFile, toolAnchorFn,
		toolAnchorFn + "\tio.WriteString(os.Stdout, s)\n"}}, "io.WriteString"},
	{"println(err)", []toolEdit{{toolRecordFile, toolAnchorFn,
		toolAnchorFn + "\tvar err error\n\tprintln(err)\n"}}, "println"},
	{"panic(err)", []toolEdit{{toolRecordFile, toolAnchorFn,
		toolAnchorFn + "\tvar err error\n\tpanic(err)\n"}}, "panic"},
	{"slog.Info", []toolEdit{{toolRecordFile, toolAnchorImp, toolAnchorImp + "\t\"log/slog\"\n"},
		{toolRecordFile, toolAnchorFn, toolAnchorFn + "\tslog.Info(\"x\", \"err\", s)\n"}}, "log/slog"},
	{"tb := t; tb.Log(err)", []toolEdit{{"live_0084_toolschema_measure_test.go", toolAnchorLive,
		toolAnchorLive + "\tvar err error\n\ttb := t\n\ttb.Log(err)\n"}}, "(*testing.common).Log"},
	{"a call to getListing", []toolEdit{{toolRecordFile, toolAnchorFn,
		toolAnchorFn + "\tgetListing(nil, \"\")\n"}}, "getListing"},
	{"an import of fmt", []toolEdit{{toolRecordFile, toolAnchorImp, toolAnchorImp + "\t\"fmt\"\n"},
		{toolRecordFile, toolAnchorFn, toolAnchorFn + "\t_ = fmt.Sprint\n"}}, `"fmt"`},
	{"t.Fatal(err)", []toolEdit{{"live_0084_toolschema_measure_test.go", toolAnchorLive,
		toolAnchorLive + "\tvar err error\n\tt.Fatal(err)\n"}}, "an argument is not a constant"},
	{"t.Run(err.Error(), f)", []toolEdit{{"live_0084_toolschema_measure_test.go", toolAnchorLive,
		toolAnchorLive + "\tvar err error\n\tt.Run(err.Error(), func(*testing.T) {})\n"}}, "(*testing.T).Run"},
	{"os.WriteFile outside the write function", []toolEdit{{toolRecordFile, toolAnchorFn,
		toolAnchorFn + "\t_ = os.WriteFile(\"x\", nil, 0)\n"}}, "os.WriteFile: outside"},
	{"os.ReadFile outside the write function", []toolEdit{{toolRecordFile, toolAnchorFn,
		toolAnchorFn + "\t_, _ = os.ReadFile(\"x\")\n"}}, "os.ReadFile: outside"},
	{"a write path that is not built from dir", []toolEdit{{toolRecordFile, "return os.WriteFile(path, enc, 0o600)",
		"return os.WriteFile(\"x\", enc, 0o600)"}}, "path argument"},
	{"a call through a function value", []toolEdit{{toolRecordFile, toolAnchorFn,
		toolAnchorFn + "\tf := func() {}\n\tf()\n"}}, "call through a function value"},
	{"P1: a fresh toolCase's id field carrying Message", []toolEdit{{toolRecordFile,
		"emit(toolLineFormat, c.id, c.route, c.model, r.Status, r.Outcome)",
		"emit(toolLineFormat, toolCase{id: r.Message}.id, c.route, c.model, r.Status, r.Outcome)"}},
		"emitFunc: an argument"},
	{"P1c: a fresh scanHit's label carrying a line's text, in the scan file", []toolEdit{{toolScanFile,
		"\treportToolScan(t.Logf, hits)\n", "\treportToolScan(t.Logf, hits)\n\tt.Logf(\"%s\", scanHit{label: root}.label)\n"}},
		"(*testing.common).Logf: an argument"},
	{"P1d: http.Response.Status through an emitFunc", []toolEdit{{toolRecordFile, toolAnchorFn,
		toolAnchorFn + "\tvar resp http.Response\n\tvar e emitFunc\n\te(\"%s\", resp.Status)\n"}},
		"emitFunc: an argument"},
	{"P2: the write path reassigned after the Join", []toolEdit{{toolRecordFile,
		"\tpath := filepath.Join(dir, r.Wire, r.Case+\".json\")\n",
		"\tpath := filepath.Join(dir, r.Wire, r.Case+\".json\")\n\tif r.Case == \"\" {\n\t\tpath = r.Message\n\t}\n"}},
		"path reassigned"},
	{"the dir parameter assigned", []toolEdit{{toolRecordFile,
		"func writeToolRecord(dir string, r toolRecord) error {\n",
		"func writeToolRecord(dir string, r toolRecord) error {\n\tdir = r.Message\n"}},
		"assignment to dir"},
	{"P3: a method named writeToolRecord in the live file writes outside dir", []toolEdit{
		{"live_0084_toolschema_measure_test.go", "\t\"os\"\n", "\t\"os\"\n\t\"path/filepath\"\n"},
		{"live_0084_toolschema_measure_test.go", toolAnchorLive,
			"type toolW struct{}\n\nfunc (toolW) writeToolRecord(dir string) {\n\tp := filepath.Join(dir, \"..\", \"x\")\n\t_ = os.WriteFile(p, nil, 0o600)\n}\n\n" + toolAnchorLive}},
		"os.WriteFile: outside the one write function"},
	{"P10 (a guard): a range over a function value that prints", []toolEdit{{toolRecordFile, toolAnchorFn,
		"func toolLeak(yield func() bool) {\n\tos.Stderr.WriteString(\"x\")\n}\n\n" + toolAnchorFn + "\tfor range toolLeak {\n\t}\n"}},
		"(*os.File).WriteString"},
	{"t.Errorf(err)", []toolEdit{{"live_0084_toolschema_measure_test.go", toolAnchorLive,
		toolAnchorLive + "\tvar err error\n\tt.Errorf(\"%v\", err)\n"}}, "(*testing.common).Errorf: an argument"},
	{"N3: a shadowing var path in an inner block", []toolEdit{{toolRecordFile, "\treturn os.WriteFile(path, enc, 0o600)\n",
		"\t{\n\t\tvar path = r.Message\n\t\t_ = os.WriteFile(path, enc, 0o600)\n\t}\n\treturn os.WriteFile(path, enc, 0o600)\n"}},
		"path argument"},
	{"N4: a function literal's parameter named path", []toolEdit{{toolRecordFile, "\treturn os.WriteFile(path, enc, 0o600)\n",
		"\tfunc(path string) { _ = os.WriteFile(path, enc, 0o600) }(r.Message)\n\treturn os.WriteFile(path, enc, 0o600)\n"}},
		"path argument"},
	{"N5: path changed through a pointer", []toolEdit{{toolRecordFile,
		"\tpath := filepath.Join(dir, r.Wire, r.Case+\".json\")\n",
		"\tpath := filepath.Join(dir, r.Wire, r.Case+\".json\")\n\tif r.Case == \"\" {\n\t\tp := &path\n\t\t*p = r.Message\n\t}\n"}},
		"address of dir or path taken"},
	{"the address of dir taken", []toolEdit{{toolRecordFile,
		"func writeToolRecord(dir string, r toolRecord) error {\n",
		"func writeToolRecord(dir string, r toolRecord) error {\n\tpd := &dir\n\t_ = pd\n"}},
		"address of dir or path taken"},
	{"a caseRunner call given a call as its case", []toolEdit{{toolRecordFile, "\t\teach(c, func(emit emitFunc) {\n\t\t\tr, ok",
		"\t\teach(toolCases()[0], func(emit emitFunc) {\n\t\t\tr, ok"}},
		"caseRunner: the arguments"},
	{"a caseRunner call given a function value, not a literal", []toolEdit{{toolRecordFile, "\tvar failed, again []toolCase\n",
		"\tvar failed, again []toolCase\n\tvar f func(emitFunc)\n\tfor _, c := range cases {\n\t\teach(c, f)\n\t}\n"}},
		"caseRunner: the arguments"},
	{"a run given something but t.Logf", []toolEdit{{"live_0084_toolschema_measure_test.go", "run(t.Logf)",
		"run(func(string, ...any) {})"}},
		"run: the argument is not t.Logf"},
	{"a case line that prints Message", []toolEdit{{toolRecordFile,
		"emit(toolLineFormat, c.id, c.route, c.model, r.Status, r.Outcome)",
		"emit(toolLineFormat, c.id, c.route, c.model, r.Status, r.Message)"}}, "emitFunc: an argument"},
}

// TestToolSchemaRecord_CallAllowlist_0084 (ST71, ST79; the sixth records
// pass and the last edit): the recorder's record, live and scan files make no
// call that the PLAN does not name, import nothing it does not name, and give
// a printing call nothing but constants and the allowlisted fields. The guard:
// the files as built pass. Each plant, on an in-memory copy, turns it red for
// the reason it plants. The offline proofs file holds no init function and no
// package-level call.
func TestToolSchemaRecord_CallAllowlist_0084(t *testing.T) {
	files := map[string]string{}
	for _, name := range toolCheckedFiles {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		files[name] = string(b)
	}
	ck := newToolChecker(t)
	if got := ck.check(t, files); len(got) > 0 {
		t.Fatalf("the checked files break the allowlist:\n%s", strings.Join(got, "\n"))
	}
	if ck.calls < 60 || ck.emits < 2 {
		t.Fatalf("inspected %d calls, %d through emitFunc; the check is vacuous", ck.calls, ck.emits)
	}
	for _, p := range toolPlants {
		planted := maps.Clone(files)
		for _, e := range p.edits {
			if strings.Count(planted[e.file], e.old) != 1 {
				t.Fatalf("plant %q: %q does not occur once in %s", p.name, e.old, e.file)
			}
			planted[e.file] = strings.Replace(planted[e.file], e.old, e.new, 1)
		}
		got := ck.check(t, planted)
		if !slices.ContainsFunc(got, func(s string) bool { return strings.Contains(s, p.want) }) {
			t.Errorf("plant %q: not caught for %q; report:\n%s", p.name, p.want, strings.Join(got, "\n"))
			continue
		}
		t.Logf("plant %q: %s", p.name, firstContaining(got, p.want))
	}

	off, err := os.ReadFile(toolOfflineFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := toolOfflineShape(t, string(off)); len(got) > 0 {
		t.Errorf("the offline proofs file holds what runs in the recording:\n%s", strings.Join(got, "\n"))
	}
	for _, p := range []struct{ name, add, want string }{
		{"an init function", "\nfunc init() {}\n", "an init function"},
		{"a package-level call", "\nvar toolPlantVar = len(\"a\")\n", "a package-level call"},
	} {
		if got := toolOfflineShape(t, string(off)+p.add); !slices.ContainsFunc(got, func(s string) bool {
			return strings.Contains(s, p.want)
		}) {
			t.Errorf("plant %q: not caught", p.name)
		}
	}
}

func firstContaining(lines []string, want string) string {
	for _, l := range lines {
		if strings.Contains(l, want) {
			return l
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// The scan's proof.

// TestToolSchemaScan_Plant_0084 (ST85, ST84, ST79): (1) records the writer
// makes from F1, F2, F4 and F7, and an output file of allowed lines, give no
// report; (2) copies holding, each on its own line, a made-up key of each of
// the five providers' shapes, a bearer value, an api_key pair, and an output
// line of provider text, report exactly those lines, and no planted string
// appears in what the report prints.
func TestToolSchemaScan_Plant_0084(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []struct{ rel, id string }{
		{"responses/grok-string-schema-tool.json", "grok-root-string"},
		{"responses/openai-string-schema-tool.json", "openai-root-string"},
		{"messages/claude-string-schema-tool.json", "claude-root-string"},
		{"generatecontent/opencode-zen-google-string-schema-tool.json", "zen-google-root-string"},
	} {
		status := 400
		apiErr := &llmprovider.APIError{Status: status, Code: toolFixtureField(t, f.rel, "code"),
			Message: toolFixtureField(t, f.rel, "message")}
		r := toolFill(toolCaseByID(t, f.id).record(),
			toolReply{err: apiErr, status: status, body: []byte(toolFixtureField(t, f.rel, "body"))}, false)
		if err := writeToolRecord(dir, r); err != nil {
			t.Fatal(err)
		}
	}
	if hits, ok := scanToolPath(dir); !ok || len(hits) != 0 {
		t.Errorf("records the writer makes: %d hits, ok %v; want none", len(hits), ok)
	}
	allowed := []string{
		"=== RUN   TestLive_ToolSchema_0084",
		"=== RUN   TestLive_ToolSchema_0084/grok-root-string",
		"=== NAME  TestLive_ToolSchema_0084",
		"--- PASS: TestLive_ToolSchema_0084 (1.20s)",
		"    --- SKIP: TestLive_ToolSchema_0084/grok-root-string (0.00s)",
		"    --- FAIL: TestLive_ToolSchema_0084/grok-root-string (0.00s)",
		"        toolschema_0084_record_test.go:312: grok-root-string grok grok-4.6: status 400, refused",
		"        toolschema_0084_record_test.go:312: openai-root-string openai gpt-6-luna: status 0, skipped",
		"        live_0084_toolschema_measure_test.go:30: " + toolTextSkip,
		"        live_0084_toolschema_measure_test.go:30: " + toolTextNoOut,
		"        live_0084_toolschema_measure_test.go:30: " + toolTextInside,
		"        live_0084_toolschema_measure_test.go:30: " + strings.Replace(toolTextNotWritten, "%s", "grok-root-string", 1),
		"PASS", "FAIL",
		"ok  \tgithub.com/maccavelli/go-llmprovider-sdk/llmprovider\t3.456s",
		"FAIL\tgithub.com/maccavelli/go-llmprovider-sdk/llmprovider\t3.456s",
	}
	out := filepath.Join(t.TempDir(), "run.txt")
	if err := os.WriteFile(out, []byte(strings.Join(allowed, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if hits, ok := scanToolPath(out); !ok || len(hits) != 0 {
		t.Errorf("an output file of allowed lines: %d hits, ok %v; want none", len(hits), ok)
	}

	shapes := []string{
		toolMade("sk-proj-", "Ab3_x", 8),                      // OpenAI
		toolMade("sk-ant-api03-", "Ab1x", 6),                  // Anthropic
		toolMade("xai-", "Zq8k", 20),                          // xAI, 80 characters
		toolMade("AIza", "Ab3_", 9),                           // Gemini
		toolMade("sk-", "Ab3x", 16),                           // OpenCode, 64 characters
		"Authorization: Bearer " + strings.Repeat("Qw9z", 10), // a bearer value
		`"api_key": "` + strings.Repeat("Mn4k", 6) + `"`,      // an api_key pair
	}
	lines := []string{"{", "  \"case\": \"x\","}
	for _, s := range shapes {
		lines = append(lines, "  \"body\": \"note "+s+"\",")
	}
	lines = append(lines, "}")
	copyDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(copyDir, "r.json"), []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{}
	for i := range shapes {
		want[strconv.Itoa(i+3)+" "+scanLabelKey] = true
	}
	hits, ok := scanToolPath(copyDir)
	got := map[string]bool{}
	for _, h := range hits {
		got[strconv.Itoa(h.line)+" "+h.label] = true
	}
	if !ok || !maps.Equal(got, want) {
		t.Errorf("records copy: reported %v (ok %v); want %v", got, ok, want)
	}

	outLines := append([]string{"PASS"}, "    toolschema_0084_record_test.go:1: "+"a provider said invalid schema")
	outLines = append(outLines, "ok  \tgithub.com/maccavelli/go-llmprovider-sdk/llmprovider\t3.456s")
	outLines = append(outLines, shapes[0])
	outFile := filepath.Join(t.TempDir(), "run.txt")
	if err := os.WriteFile(outFile, []byte(strings.Join(outLines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	hits, ok = scanToolPath(outFile)
	got = map[string]bool{}
	for _, h := range hits {
		got[strconv.Itoa(h.line)+" "+h.label] = true
	}
	wantOut := map[string]bool{"2 " + scanLabelLine: true, "4 " + scanLabelLine: true, "4 " + scanLabelKey: true}
	if !ok || !maps.Equal(got, wantOut) {
		t.Errorf("output copy: reported %v (ok %v); want %v", got, ok, wantOut)
	}
	var buf bytes.Buffer
	reportToolScan(toolBuffer(&buf), hits)
	for _, s := range append(shapes, "a provider said invalid schema") {
		if strings.Contains(buf.String(), s) {
			t.Errorf("the report prints a planted string (starting %q)", s[:4])
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if !strings.HasPrefix(line, "run.txt:") {
			t.Errorf("report line %q: want <file>:<line>: <label>", line)
		}
	}
}
