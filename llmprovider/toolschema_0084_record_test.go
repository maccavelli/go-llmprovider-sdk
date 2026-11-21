package llmprovider_test

// 0084-PLAN Phase 1a's recorder of how each provider answers a tool whose
// schema it may refuse: its case table, the outcome of a reply, the record,
// the redaction and the one function that writes a record. Untagged, so that
// the offline proofs (toolschema_0084_record_offline_test.go) run the code the
// live recorder (live_0084_toolschema_measure_test.go) runs, as
// live_twins_test.go runs live_checks_test.go's (0046-MADR D3). A record holds
// a status, an error's kind, code, message and redacted body, and a count of
// tokens; never a key, a header or a model's text.
//
// This file, the live file and the scan file (toolschema_0084_scan_test.go)
// run during the recording, with the keys in the process environment, so
// TestToolSchemaRecord_CallAllowlist_0084 type-checks them and allows no call
// that is not named there: nothing here prints, logs or writes an error's
// text, a body or a key (0084-MADR ST71, ST77, ST79, ST80).

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// The outcomes a record holds (0084-PLAN Phase 1a, step 2, the corrected
// table; 0084-MADR ST67).
const (
	toolOutcomeAccepted    = "accepted"
	toolOutcomeRefused     = "refused"
	toolOutcomeUnreachable = "unreachable"
	toolOutcomeOther       = "other"
	toolOutcomeSkipped     = "skipped"
)

// The fixed reasons a record gives, never an error's text.
const (
	toolReasonNoKey     = "key unset or blank"
	toolReasonTransport = "transport"
)

// toolLineFormat is the one line a case prints: the case, its route and
// model, the HTTP status and the outcome, and nothing else (0084-MADR ST79).
const toolLineFormat = "%s %s %s: status %d, %s"

// The recorder's fixed failure and skip texts: each holds at most a case id.
// The scan's allowlist matches them (toolschema_0084_scan_test.go).
const (
	toolTextSkip       = "set LLMPROVIDER_LIVE_0084 to run 0084-PLAN Phase 1's recorder"
	toolTextNoOut      = "LLMPROVIDER_LIVE_0084_OUT unset: name a directory outside the module for the records"
	toolTextInside     = "LLMPROVIDER_LIVE_0084_OUT is inside the module: name a directory outside it"
	toolTextNotWritten = "%s: record not written"
)

// The wires of 0062-MADR D1, as the fixtures' directories name them.
const (
	toolWireMessages        = "messages"
	toolWireResponses       = "responses"
	toolWireInteractions    = "interactions"
	toolWireGenerateContent = "generatecontent"
	toolWireChatCompletions = "chatcompletions"
)

const (
	// toolBodyLimit bounds a record's error body: 4 KiB, as 0062's is.
	toolBodyLimit = 4 << 10
	// toolCaseTimeout is each case's deadline.
	toolCaseTimeout = 90 * time.Second
	// toolMaxOutputTokens bounds each generation (0084-PLAN Phase 1, Billed).
	toolMaxOutputTokens = 64
	// toolPrompt is every case's input text.
	toolPrompt = "Reply with OK."
	// toolDescription describes every tool sent.
	toolDescription = "Records a value."
	// toolHistoryNote says what an O5 case's input holds beside the prompt.
	toolHistoryNote = "a prior call of record_city and its result"
)

// emitFunc is where a case's line goes: t.Logf in the live test, a buffer in
// the offline proofs. The call allowlist lets a call through this type take
// only constants and the named fields (case id, route, model, status,
// outcome).
type emitFunc func(format string, args ...any)

// toolSpec is a tool as a record holds it: its name and the schema sent.
type toolSpec struct {
	Name   string `json:"name"`
	Schema any    `json:"schema"`
}

// toolCase is one row of 0084-PLAN Phase 1a's case table.
type toolCase struct {
	id       string
	route    string
	model    string
	shape    string
	wire     string
	provider llmprovider.ProviderID
	via      opencode.Route
	tools    []toolSpec
	history  bool
}

// toolRoute is a route of the case table: its label, wire, provider, the
// OpenCode route where there is one, and its model.
type toolRoute struct {
	label    string
	wire     string
	provider llmprovider.ProviderID
	via      opencode.Route
	model    string
}

// toolRoutes are the case table's routes, each with the model its 0062
// fixture used (0084-PLAN Phase 1a, step 1). grok and grok2 are Grok's two
// models, and zen-messages is OpenCode Zen's Messages route (the ruling on
// N3), read under the opencode label (0084-MADR ST66).
var toolRoutes = []toolRoute{
	{"grok", toolWireResponses, llmprovider.ProviderGrok, "", "grok-4.6"},
	{"grok2", toolWireResponses, llmprovider.ProviderGrok, "", "grok-4.5"},
	{"openai", toolWireResponses, llmprovider.ProviderOpenAI, "", "gpt-6-luna"},
	{"zen-responses", toolWireResponses, llmprovider.ProviderOpencodeZen, opencode.RouteResponses, "gpt-5.5"},
	{"claude", toolWireMessages, llmprovider.ProviderClaude, "", "claude-sonnet-5"},
	{"zen-google", toolWireGenerateContent, llmprovider.ProviderOpencodeZen, opencode.RouteGoogle, "gemini-3.8-flash"},
	{"gemini", toolWireInteractions, llmprovider.ProviderGemini, "", "gemini-3.8-flash"},
	{"zen-chat", toolWireChatCompletions, llmprovider.ProviderOpencodeZen, opencode.RouteChatCompletions, "glm-5.3"},
	{"go-messages", toolWireMessages, llmprovider.ProviderOpencodeGo, opencode.RouteMessages, "qwen3.8-flash"},
	{"zen-messages", toolWireMessages, llmprovider.ProviderOpencodeZen, opencode.RouteMessages, "claude-sonnet-5"},
}

// The shapes each route runs (0084-PLAN Phase 1a, step 1): root-string and
// O1 to O5 on every route, and O6's three on grok2 alone.
var (
	toolShapes = []string{"root-string", "O1-root-union", "O2-two-bad", "O3-typed-branches",
		"O4-nested-union", "O5-history-dropped"}
	toolShapesO6 = []string{"root-string", "O1-root-union", "O2-two-bad"}
)

// toolObject is an empty-object schema, a valid tool's.
func toolObject() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

// toolUnion is the owner's shape (O1): an object with city and country,
// additionalProperties false, and a root anyOf of two branches that each
// require one property; typed gives each branch "type": "object" and
// "properties": {} (O3).
func toolUnion(typed bool) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"city":    map[string]any{"type": "string"},
			"country": map[string]any{"type": "string"},
		},
		"additionalProperties": false,
		"anyOf":                []any{toolBranch("city", typed), toolBranch("country", typed)},
	}
}

// toolBranch is one branch of the root anyOf: it requires one property, and
// when typed is an object with no properties of its own.
func toolBranch(name string, typed bool) map[string]any {
	b := map[string]any{"required": []any{name}}
	if typed {
		b["type"] = "object"
		b["properties"] = map[string]any{}
	}
	return b
}

// toolCity is the tool record_city with a schema.
func toolCity(schema any) toolSpec { return toolSpec{Name: "record_city", Schema: schema} }

// toolNested is O4's: an object whose property target is an anyOf of a string
// and an object.
func toolNested() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"target": map[string]any{"anyOf": []any{
				map[string]any{"type": "string"},
				map[string]any{"type": "object", "properties": map[string]any{}},
			}},
		},
	}
}

// toolShapeTools returns a shape's tools, the valid control tool noop first,
// so that a refused tool sits at index 1, never 0.
func toolShapeTools(shape string) []toolSpec {
	noop := toolSpec{Name: "noop", Schema: toolObject()}
	switch shape {
	case "root-string":
		return []toolSpec{noop, toolCity(map[string]any{"type": "string"})}
	case "O1-root-union":
		return []toolSpec{noop, toolCity(toolUnion(false))}
	case "O2-two-bad":
		return []toolSpec{noop, toolCity(toolUnion(false)),
			{Name: "record_country", Schema: map[string]any{"type": "string"}}}
	case "O3-typed-branches":
		return []toolSpec{noop, toolCity(toolUnion(true))}
	case "O4-nested-union":
		return []toolSpec{noop, toolCity(toolNested())}
	default:
		// O5-history-dropped: noop only; the input holds the call.
		return []toolSpec{noop}
	}
}

// toolCases returns the case table, route by route: 57 cases with every key
// set (nine routes of six, and grok2's three).
func toolCases() []toolCase {
	var cases []toolCase
	for _, r := range toolRoutes {
		shapes := toolShapes
		if r.label == "grok2" {
			shapes = toolShapesO6
		}
		for _, shape := range shapes {
			cases = append(cases, toolCase{id: r.label + "-" + shape, route: r.label, model: r.model,
				shape: shape, wire: r.wire, provider: r.provider, via: r.via,
				tools: toolShapeTools(shape), history: shape == "O5-history-dropped"})
		}
	}
	return cases
}

// toolRecord is one case's JSON record (0084-PLAN Phase 1a, step 2).
type toolRecord struct {
	Case          string     `json:"case"`
	Route         string     `json:"route"`
	Wire          string     `json:"wire"`
	Provider      string     `json:"provider"`
	OpenCodeRoute string     `json:"opencode_route,omitempty"`
	Model         string     `json:"model"`
	Shape         string     `json:"shape"`
	Tools         []toolSpec `json:"tools"`
	Input         string     `json:"input"`
	History       string     `json:"history,omitempty"`
	Status        int        `json:"status"`
	// Kind names the *APIError's kind, as the base classifies it.
	Kind    string `json:"kind,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	// Body is the error body, redacted and cut to 4 KiB.
	Body    string `json:"body,omitempty"`
	Outcome string `json:"outcome"`
	Reason  string `json:"reason,omitempty"`
	// The counts of an accepted case.
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	ToolCalls    int `json:"tool_calls,omitempty"`
}

// record is c's record before anything is sent.
func (c toolCase) record() toolRecord {
	r := toolRecord{Case: c.id, Route: c.route, Wire: c.wire, Provider: string(c.provider),
		OpenCodeRoute: string(c.via), Model: c.model, Shape: c.shape, Tools: c.tools, Input: toolPrompt}
	if c.history {
		r.History = toolHistoryNote
	}
	return r
}

// request is c's request: the prompt, and the case's tools in order; an O5
// case's input also holds a prior call of record_city and its result.
func (c toolCase) request() *llmprovider.Request {
	req := &llmprovider.Request{MaxOutputTokens: toolMaxOutputTokens,
		Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: toolPrompt}}}
	for _, s := range c.tools {
		req.Tools = append(req.Tools, llmprovider.Tool{Name: s.Name, Description: toolDescription, Schema: s.Schema})
	}
	if c.history {
		req.Input = append(req.Input,
			llmprovider.FunctionCallItem{CallID: "call_0084", Name: "record_city", Arguments: `{"city":"Paris"}`},
			llmprovider.FunctionCallOutputItem{CallID: "call_0084", Output: `"ok"`})
	}
	return req
}

// toolKinds are the kind sentinels, each child before its parent, so that a
// record names the most specific kind an error matches.
var toolKinds = []struct {
	err  error
	name string
}{
	{llmprovider.ErrQuotaExhausted, "quota_exhausted"},
	{llmprovider.ErrRateLimited, "rate_limited"},
	{llmprovider.ErrNotPermitted, "not_permitted"},
	{llmprovider.ErrAuthFailure, "auth_failure"},
	{llmprovider.ErrContextOverflow, "context_overflow"},
	{llmprovider.ErrEmptyAnswer, "empty_answer"},
	{llmprovider.ErrIncomplete, "incomplete"},
	{llmprovider.ErrWorkspaceRequired, "workspace_required"},
	{llmprovider.ErrWorkspaceNotFound, "workspace_not_found"},
	{llmprovider.ErrInvalidRequest, "invalid_request"},
	{llmprovider.ErrOffline, "offline"},
	{llmprovider.ErrProviderUnavailable, "provider_unavailable"},
	{llmprovider.ErrUnsupported, "unsupported"},
}

// toolKindName is the name of the most specific kind err matches, or "".
func toolKindName(err error) string {
	for _, k := range toolKinds {
		if errors.Is(err, k.err) {
			return k.name
		}
	}
	return ""
}

// toolOutcome is the outcome of what Generate returned, and the fixed reason
// where there is one: the first row of 0084-PLAN Phase 1a's corrected table
// that applies, short of the unset key, which the recorder decides before it
// sends (0084-MADR ST67).
func toolOutcome(err error) (outcome, reason string) {
	if err == nil {
		return toolOutcomeAccepted, ""
	}
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	if !ok {
		return toolOutcomeUnreachable, toolReasonTransport
	}
	switch s := apiErr.Status; {
	case s == 0 && errors.Is(err, llmprovider.ErrIncomplete):
		// A 200 the SDK found cut short or empty: the schema was not refused
		// (the review's a1).
		return toolOutcomeAccepted, ""
	case s == 0, s == http.StatusRequestTimeout, s == http.StatusConflict, s >= 500 && s < 600:
		return toolOutcomeOther, ""
	case s >= 400 && s < 500 && s != http.StatusUnauthorized && s != http.StatusForbidden &&
		s != http.StatusNotFound && s != http.StatusTooManyRequests &&
		(apiErr.Kind == nil || errors.Is(apiErr.Kind, llmprovider.ErrInvalidRequest)):
		// The error's own kind, not the unwrap chain: Unwrap also yields the
		// status's legacy ErrInvalidRequest for a refused key and a 402
		// (0084-MADR, fifth amendment, ST67).
		return toolOutcomeRefused, ""
	default:
		return toolOutcomeUnreachable, ""
	}
}

// toolCapture sends each request on base and keeps the last reply's status
// and body for the record. It never reads the request.
type toolCapture struct {
	base   http.RoundTripper
	status int
	body   []byte
}

func (cp *toolCapture) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := cp.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(b))
	cp.status, cp.body = resp.StatusCode, b
	return resp, nil
}

// toolReply is what one send returned.
type toolReply struct {
	resp   *llmprovider.Response
	err    error
	status int
	body   []byte
}

// send sends c's request through its provider, with key, on base, once and
// with no retry. built is false when the provider could not be built, which
// is a fault of the recorder, not a reply.
func (c toolCase) send(key string, base http.RoundTripper) (rep toolReply, built bool) {
	ctx, cancel := context.WithTimeout(context.Background(), toolCaseTimeout)
	defer cancel()
	cp := &toolCapture{base: base}
	opts := []llmprovider.Option{llmprovider.WithAPIKey(key), llmprovider.WithModel(c.model),
		llmprovider.WithModelProbes(false), llmprovider.WithoutModelMetadata(),
		llmprovider.WithHTTPClient(&http.Client{Transport: cp})}
	if c.via != "" {
		opts = append(opts, opencode.WithRoute(c.via))
	}
	p, err := providers.New(c.provider, opts...)
	if err != nil {
		return toolReply{}, false
	}
	rep.resp, rep.err = p.Generate(ctx, c.request())
	rep.status, rep.body = cp.status, cp.body
	return rep, true
}

// toolFill sets r's outcome and what the reply says: the status, the kind,
// the code, the message and the body of a refusal, each redacted; the counts
// of an accepted case. A case still other after its second send is
// unreachable (retry).
func toolFill(r toolRecord, rep toolReply, retry bool) toolRecord {
	r.Outcome, r.Reason = toolOutcome(rep.err)
	if r.Outcome == toolOutcomeOther && retry {
		r.Outcome = toolOutcomeUnreachable
	}
	if rep.err == nil {
		r.Status = rep.status
		r.InputTokens, r.OutputTokens = rep.resp.Usage.InputTokens, rep.resp.Usage.OutputTokens
		for _, it := range rep.resp.Output {
			if _, ok := it.(llmprovider.FunctionCallItem); ok {
				r.ToolCalls++
			}
		}
		return r
	}
	apiErr, ok := errors.AsType[*llmprovider.APIError](rep.err)
	if !ok {
		return r
	}
	r.Status = apiErr.Status
	if r.Outcome == toolOutcomeAccepted {
		r.Status = rep.status
	}
	r.Kind = toolKindName(rep.err)
	r.Code = redactToolText(apiErr.Code)
	r.Message = redactToolText(apiErr.Message)
	if rep.status/100 != 2 {
		r.Body = boundToolBody(redactToolText(string(rep.body)))
	}
	return r
}

// recordToolCase runs one case and writes its record to dir. The route's key
// is read from the environment by its ProviderEnvVars name and handed to the
// provider; a key unset, empty or all spaces makes the case skipped
// (0084-MADR ST77, ST78). A case line goes to emit and nothing else is
// printed. retry says this is the second send of a case that was other. The
// result is false when the record could not be written or the provider could
// not be built; the caller fails with a fixed text.
func recordToolCase(c toolCase, dir string, base http.RoundTripper, retry bool, emit emitFunc) (toolRecord, bool) {
	r := c.record()
	if key := os.Getenv(llmprovider.ProviderEnvVars()[c.provider]); strings.TrimSpace(key) == "" {
		r.Outcome, r.Reason = toolOutcomeSkipped, toolReasonNoKey
	} else {
		rep, built := c.send(key, base)
		if !built {
			return r, false
		}
		r = toolFill(r, rep, retry)
	}
	if writeToolRecord(dir, r) != nil {
		return r, false
	}
	emit(toolLineFormat, c.id, c.route, c.model, r.Status, r.Outcome)
	return r, true
}

// caseRunner runs one case's sends: the live test gives each case a subtest
// named by its id and passes t.Logf to run, so that one case, and its second
// send, reruns alone; the offline proofs pass a buffer. The allowlist lets a
// call through it take a toolCase variable and a function literal alone.
type caseRunner func(c toolCase, run func(emit emitFunc))

// recordToolCases runs every case through each, then sends each case that was
// other once more, also through each, so that one command records
// everything: a case still other is recorded unreachable, and one whose second
// send is a refusal is recorded refused (0084-MADR ST67). It returns the cases
// whose record could not be written, for the caller to fail on with a fixed
// text. It is here, not in the live file, so that an untagged test runs it
// (0084-MADR, sixth amendment).
func recordToolCases(cases []toolCase, dir string, base http.RoundTripper, each caseRunner) []toolCase {
	var failed, again []toolCase
	for _, c := range cases {
		each(c, func(emit emitFunc) {
			r, ok := recordToolCase(c, dir, base, false, emit)
			switch {
			case !ok:
				failed = append(failed, c)
			case r.Outcome == toolOutcomeOther:
				again = append(again, c)
			}
		})
	}
	for _, c := range again {
		each(c, func(emit emitFunc) {
			if _, ok := recordToolCase(c, dir, base, true, emit); !ok {
				failed = append(failed, c)
			}
		})
	}
	return failed
}

// toolFixtureRules are the repository's fixture rules (AGENTS.md, "Recorded
// fixtures"; scripts/check_fixtures.py), applied after internal/redact: a
// request id, an account, user or organization id, a workspace id, a UUID, an
// e-mail address and a home path. 0062 redacted request ids by hand after its
// run; here the writer does it.
var toolFixtureRules = []struct {
	re   *regexp.Regexp
	with string
}{
	{regexp.MustCompile(`(?i)(request_id\\*"\s*:\s*\\*")[^"\\]*`), "${1}req_REDACTED"},
	{regexp.MustCompile(`\breq_[A-Za-z0-9]+`), "req_REDACTED"},
	{regexp.MustCompile(`\b((?i:user|org|acct|account)[-_])[A-Za-z0-9]{8,}`), "${1}REDACTED"},
	{regexp.MustCompile(`\bwrkspc_[A-Za-z0-9]+`), "wrkspc_REDACTED"},
	{regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`),
		"00000000-0000-0000-0000-000000000000"},
	{regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)+`), "user@example.com"},
	{regexp.MustCompile(`/(?:Users|home)/[A-Za-z0-9._-]+`), "/home/<user>"},
}

// redactToolText redacts s as a record's text fields are: internal/redact's
// keys, tokens, bearer and basic credentials, key-value secrets and signed
// URLs, then the fixture rules above. It applies to every text field of a
// record, Message and Code included (the review's a3).
func redactToolText(s string) string {
	s = redact.String(s)
	for _, rule := range toolFixtureRules {
		s = rule.re.ReplaceAllString(s, rule.with)
	}
	return s
}

// boundToolBody is s cut to toolBodyLimit bytes, on a rune boundary.
func boundToolBody(s string) string {
	if len(s) <= toolBodyLimit {
		return s
	}
	cut := toolBodyLimit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

// toolRecordKept reports whether a written record is one the writer never
// replaces: refused or accepted, found by its top-level outcome line.
func toolRecordKept(old []byte) bool {
	text := string(old)
	return strings.Contains(text, "\n  \"outcome\": \"refused\"") ||
		strings.Contains(text, "\n  \"outcome\": \"accepted\"")
}

// writeToolRecord writes r to dir/<wire>/<case>.json as golden text, and is
// the only function that writes a file. It never replaces a refused or
// accepted record, and writes skipped only where the case has no record, so
// a full rerun keeps every good record (0084-MADR ST70).
func writeToolRecord(dir string, r toolRecord) error {
	enc, err := wiretest.Encode(r)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, r.Wire, r.Case+".json")
	old, err := os.ReadFile(path)
	switch {
	case err == nil:
		if toolRecordKept(old) || r.Outcome == toolOutcomeSkipped {
			return nil
		}
	case !errors.Is(err, os.ErrNotExist):
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, enc, 0o600)
}

// toolInsideModule reports whether dir is inside the module, whose root is
// the parent of this package's directory, or cannot be told: records are
// copied into the tree only after they are redacted and checked, as 0062's
// are (0084-PLAN Phase 1a, step 2).
func toolInsideModule(dir string) bool {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return true
	}
	root, err := filepath.Abs("..")
	if err != nil {
		return true
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		// Another volume: outside the module.
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
