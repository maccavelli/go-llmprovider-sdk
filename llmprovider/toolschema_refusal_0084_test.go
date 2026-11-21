package llmprovider

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// 0084-MADR D1 to D3 and the state table's ST14 to ST37: a service's refusal
// of a tool's schema is the kind ErrToolSchemaRefused where, and only where,
// its route's marker is in the reply, as classified from the replies this
// repository recorded: 0062-PLAN Phase 1's and 0084-PLAN Phase 1's. The tests
// here never reach a provider; the ones that do are the providers package's.

// recorded0084 is the part of a recorded refusal a classification reads.
type recorded0084 struct {
	Case    string `json:"case"`
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Body    string `json:"body"`
	Outcome string `json:"outcome"`
	// Provider and Route are the label a recording's provider sent under.
	Provider string `json:"provider"`
	Route    string `json:"route"`
}

// load0084 reads the record at rel under internal/wire/testdata.
func load0084(t *testing.T, rel string) recorded0084 {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("internal", "wire", "testdata", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	var r recorded0084
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
	return r
}

// The labels a route's errors carry: Generate sends OpenCode's routes under
// "opencode", Stream under "<gateway>/<route>" (MADR H3).
var (
	labelsGrok   = []string{"grok"}
	labelsOpenAI = []string{"openai"}
	labelsClaude = []string{"claude"}
	labelsZenRes = []string{"opencode", "opencode-zen/responses"}
	labelsZenGoo = []string{"opencode", "opencode-zen/google"}
)

// recognised0084 are the records a D3 matcher reads, by 0084-PLAN Phase 1c's
// sort (MADR ST53 to ST55, ST59, ST69): Phase 1's refusals in a recognised
// form, and the 0062 refusal of the Google route, which has none of its own.
// Each is a refusal whose tool sits at index 1 of the list its run sent.
var recognised0084 = []struct {
	file   string
	labels []string
}{
	{"toolschema-0084/responses/grok-root-string.json", labelsGrok},
	{"toolschema-0084/responses/grok-O1-root-union.json", labelsGrok},
	{"toolschema-0084/responses/grok-O2-two-bad.json", labelsGrok},
	{"toolschema-0084/responses/grok2-root-string.json", labelsGrok},
	{"toolschema-0084/responses/grok2-O1-root-union.json", labelsGrok},
	{"toolschema-0084/responses/grok2-O2-two-bad.json", labelsGrok},
	{"toolschema-0084/responses/openai-root-string.json", labelsOpenAI},
	{"toolschema-0084/responses/openai-O2-two-bad.json", labelsOpenAI},
	{"toolschema-0084/responses/zen-responses-root-string.json", labelsZenRes},
	{"toolschema-0084/responses/zen-responses-O2-two-bad.json", labelsZenRes},
	{"toolschema-0084/messages/claude-root-string.json", labelsClaude},
	{"toolschema-0084/messages/claude-O1-root-union.json", labelsClaude},
	{"toolschema-0084/messages/claude-O2-two-bad.json", labelsClaude},
	{"refusals-0062/generatecontent/opencode-zen-google-string-schema-tool.json", labelsZenGoo},
}

// unrecognised0084 are the refusals of Phase 1 that no matcher reads (the
// plan's Phase 1c: ST54 (3), ST55 (4)): Gemini's, which names the tool in
// single quotes, and OpenCode Zen's Google route's, which refuses
// additionalProperties.
var unrecognised0084 = []struct {
	file   string
	labels []string
}{
	{"toolschema-0084/interactions/gemini-O1-root-union.json", []string{"gemini"}},
	{"toolschema-0084/interactions/gemini-O2-two-bad.json", []string{"gemini"}},
	{"toolschema-0084/generatecontent/zen-google-O1-root-union.json", labelsZenGoo},
	{"toolschema-0084/generatecontent/zen-google-O2-two-bad.json", labelsZenGoo},
}

// evidenceOnly0084 are the refusals of Phase 1 that feed no test (ST56, the
// typed branches): Anthropic's is in a recognised form, and the other two are
// not. They are listed so that a new refused record cannot be forgotten.
var evidenceOnly0084 = []string{
	"toolschema-0084/messages/claude-O3-typed-branches.json",
	"toolschema-0084/interactions/gemini-O3-typed-branches.json",
	"toolschema-0084/generatecontent/zen-google-O3-typed-branches.json",
}

// the 0062 refusals of the four read forms, which ST20, ST23 and ST50 serve
// altered.
var refusals0062 = []struct {
	file   string
	labels []string
}{
	{"refusals-0062/responses/grok-string-schema-tool.json", labelsGrok},
	{"refusals-0062/responses/openai-string-schema-tool.json", labelsOpenAI},
	{"refusals-0062/messages/claude-string-schema-tool.json", labelsClaude},
	{"refusals-0062/generatecontent/opencode-zen-google-string-schema-tool.json", labelsZenGoo},
}

// toolSchemaRefused classifies body at status as label sends it, and returns
// the *APIError with the error it came in.
func toolSchemaRefused(t *testing.T, label string, status int, body string, header http.Header) (*APIError, error) {
	t.Helper()
	err := classifyFixture(label, status, body, header)
	apiErr, ok := errors.AsType[*APIError](err)
	if !ok {
		t.Fatalf("%s %d: %v; want an *APIError", label, status, err)
	}
	return apiErr, err
}

// TestToolSchemaRefusal_Classify_0084 (MADR D1, D3, D6; ST01, ST02, ST04 to
// ST06, ST15, ST18, ST29 to ST31, ST37): each recorded refusal a matcher
// reads is an *APIError of kind ErrToolSchemaRefused, which is also an
// ErrInvalidRequest and nothing else, is never retried, has no Tool (the
// provider names it, D4), and reads "llmprovider: tool schema refused: ..."
// with the Tool, were it set, not printed.
func TestToolSchemaRefusal_Classify_0084(t *testing.T) {
	for _, tc := range recognised0084 {
		f := load0084(t, tc.file)
		for _, label := range tc.labels {
			name := f.Case + "/" + label
			apiErr, err := toolSchemaRefused(t, label, f.Status, f.Body, nil)
			if !errors.Is(err, ErrToolSchemaRefused) || !errors.Is(err, ErrInvalidRequest) {
				t.Errorf("%s: %v; want ErrToolSchemaRefused, and so ErrInvalidRequest", name, err)
			}
			// ST30: errors.As finds the error, whose own kind is the new one.
			if !errors.Is(apiErr.Kind, ErrToolSchemaRefused) || apiErr.Status != http.StatusBadRequest {
				t.Errorf("%s: Kind %v, Status %d; want ErrToolSchemaRefused, 400", name, apiErr.Kind, apiErr.Status)
			}
			// ST37: terminal, and not a rate limit by any parent.
			if apiErr.Retryable() || errors.Is(err, ErrRateLimited) || errors.Is(err, ErrProviderUnavailable) {
				t.Errorf("%s: retryable %t, or a rate limit or unavailability; want a terminal invalid request", name,
					apiErr.Retryable())
			}
			// ST29: Unwrap yields the kind alone: the legacy sentinel of a 400
			// is already beneath it.
			if u := apiErr.Unwrap(); len(u) != 1 || !errors.Is(u[0], ErrToolSchemaRefused) {
				t.Errorf("%s: Unwrap %v; want the kind alone", name, u)
			}
			if apiErr.Tool != "" {
				t.Errorf("%s: Tool %q; ClassifyHTTPError never sees the request, so it names none (D4)", name, apiErr.Tool)
			}
			// ST31: the kind leads, and the Tool, were it set, is not printed.
			if !strings.HasPrefix(err.Error(), "llmprovider: tool schema refused: ") {
				t.Errorf("%s: Error() %q; want it to begin %q", name, err.Error(), "llmprovider: tool schema refused: ")
			}
			before := err.Error()
			apiErr.Tool = "tool_named_for_the_probe"
			if after := err.Error(); after != before {
				t.Errorf("%s: Error() changed from %q to %q when Tool was set; D6 says it is not printed", name, before, after)
			}
			apiErr.Tool = ""
		}
	}
}

// TestToolSchemaRefusal_Reworded_0084 (MADR D3, "When a provider rewords a
// refusal"; ST14, ST24): the staleness guard. Each recorded refusal with its
// route's marker changed is an ordinary invalid request, not the kind, with
// no Tool. Each edit is a named edit of a recorded body (the plan's Rules, on
// the owner's ruling on N7): the marker's text is replaced, nothing else.
func TestToolSchemaRefusal_Reworded_0084(t *testing.T) {
	for _, tc := range []struct {
		name, file, from, to string
		labels               []string
	}{
		{"grok: the tag renamed", "refusals-0062/responses/grok-string-schema-tool.json",
			"[invalid_client_tool_schema]", "[invalid_client_tool_shape]", labelsGrok},
		// ST24: a content-policy code of the same service.
		{"openai: the code changed to invalid_prompt", "refusals-0062/responses/openai-string-schema-tool.json",
			"invalid_function_parameters", "invalid_prompt", labelsOpenAI},
		{"zen responses: the code changed to invalid_prompt",
			"toolschema-0084/responses/zen-responses-root-string.json", "invalid_function_parameters", "invalid_prompt",
			labelsZenRes},
		{"claude: the tools.N. path renamed", "toolschema-0084/messages/claude-root-string.json",
			"tools.1.", "toolbox.1.", labelsClaude},
		{"claude: input_schema renamed", "toolschema-0084/messages/claude-root-string.json",
			"input_schema", "input_shape", labelsClaude},
		{"zen google: the phrase changed", "refusals-0062/generatecontent/opencode-zen-google-string-schema-tool.json",
			"functionDeclaration parameters schema", "functionDeclaration parameter schema", labelsZenGoo},
	} {
		f := load0084(t, tc.file)
		if !strings.Contains(f.Body, tc.from) {
			t.Fatalf("%s: the record holds no %q to change", tc.name, tc.from)
		}
		body := strings.ReplaceAll(f.Body, tc.from, tc.to)
		for _, label := range tc.labels {
			apiErr, err := toolSchemaRefused(t, label, f.Status, body, nil)
			if errors.Is(err, ErrToolSchemaRefused) || !errors.Is(err, ErrInvalidRequest) || apiErr.Tool != "" {
				t.Errorf("%s (%s): %v, Tool %q; want an ErrInvalidRequest that is not ErrToolSchemaRefused, with no Tool",
					tc.name, label, err, apiErr.Tool)
			}
		}
	}
}

// TestToolSchemaRefusal_RecordedFallback_0084 (MADR ST54 (3), ST55 (4),
// ST60): each Phase 1 refusal that no matcher reads is classified to the kind
// ClassifyHTTPError gave it before this record, ErrInvalidRequest, with no
// Tool; and every refused record of Phase 1 is in one list of this file, so
// that a new refusal cannot go untested.
func TestToolSchemaRefusal_RecordedFallback_0084(t *testing.T) {
	for _, tc := range unrecognised0084 {
		f := load0084(t, tc.file)
		for _, label := range tc.labels {
			apiErr, err := toolSchemaRefused(t, label, f.Status, f.Body, nil)
			if errors.Is(err, ErrToolSchemaRefused) || apiErr.Kind != ErrInvalidRequest || apiErr.Tool != "" { //nolint:errorlint // the kind itself
				t.Errorf("%s (%s): %v, Kind %v, Tool %q; want ErrInvalidRequest as before, not the kind, no Tool",
					f.Case, label, err, apiErr.Kind, apiErr.Tool)
			}
		}
	}

	listed := map[string]bool{}
	for _, tc := range recognised0084 {
		listed[tc.file] = true
	}
	for _, tc := range unrecognised0084 {
		listed[tc.file] = true
	}
	for _, file := range evidenceOnly0084 {
		listed[file] = true
	}
	matches, err := filepath.Glob(filepath.Join("internal", "wire", "testdata", "toolschema-0084", "*", "*.json"))
	if err != nil || len(matches) != 57 {
		t.Fatalf("%d records of Phase 1 (%v); want the 57 the plan's execution record names", len(matches), err)
	}
	for _, path := range matches {
		rel, _ := filepath.Rel(filepath.Join("internal", "wire", "testdata"), path)
		rel = filepath.ToSlash(rel)
		if f := load0084(t, rel); f.Outcome == "refused" && !listed[rel] {
			t.Errorf("%s is a refused record in no list of this file", rel)
		}
	}
}

// TestToolSchemaRefusal_OthersUnchanged_0084 (MADR ST07, ST21, ST22, ST25,
// ST27; D3 "Not read, so unchanged"): every other recorded 400 under
// refusals-0062 keeps the kind it has always had, ErrInvalidRequest: the
// refused efforts and shapes, the empty inputs; xAI's refused key stays
// ErrAuthFailure; Anthropic's marker is read for Claude only, so OpenCode's
// Messages route keeps ErrInvalidRequest for it (ST66 (5)); and an overflow
// stays one.
func TestToolSchemaRefusal_OthersUnchanged_0084(t *testing.T) {
	// The 0062 refusals of the read forms are the kind's, not another 400:
	// F1, F2, F4 and F7, and F3, OpenCode Zen's Responses route's, which
	// Phase 1's own record of that route repeats.
	skip := map[string]bool{"refusals-0062/responses/opencode-zen-responses-string-schema-tool.json": true}
	for _, tc := range refusals0062 {
		skip[tc.file] = true
	}
	matches, err := filepath.Glob(filepath.Join("internal", "wire", "testdata", "refusals-0062", "*", "*.json"))
	if err != nil || len(matches) == 0 {
		t.Fatalf("no 0062 records (%v)", err)
	}
	checked := 0
	for _, path := range matches {
		rel, _ := filepath.Rel(filepath.Join("internal", "wire", "testdata"), path)
		rel = filepath.ToSlash(rel)
		f := load0084(t, rel)
		if skip[rel] || f.Status != http.StatusBadRequest {
			continue
		}
		label := f.Provider
		if f.Route != "" {
			label += "/" + f.Route
		}
		apiErr, err := toolSchemaRefused(t, label, f.Status, f.Body, nil)
		if errors.Is(err, ErrToolSchemaRefused) || apiErr.Kind != ErrInvalidRequest || apiErr.Tool != "" { //nolint:errorlint // the kind itself
			t.Errorf("%s (%s): %v, Kind %v; want ErrInvalidRequest as before", f.Case, label, err, apiErr.Kind)
		}
		checked++
	}
	if checked < 20 {
		t.Errorf("%d other 400 records checked; want every one recorded (20 or more)", checked)
	}

	// ST21, ST22: a refused key and a refused effort of the same service and
	// status.
	if apiErr, err := toolSchemaRefused(t, "grok", 400, grokRefusedKey, nil); apiErr.Kind != ErrAuthFailure { //nolint:errorlint // the kind itself
		t.Errorf("xAI's refused key: %v; want ErrAuthFailure", err)
	}
	if apiErr, err := toolSchemaRefused(t, "grok", 400, `{"code":"invalid-argument","error":"Invalid reasoning effort."}`,
		nil); apiErr.Kind != ErrInvalidRequest { //nolint:errorlint // the kind itself
		t.Errorf("xAI's refused effort: %v; want ErrInvalidRequest", err)
	}

	// ST66 (5): Anthropic's marker is not read for OpenCode, whose Messages
	// route is recorded in 0084-PLAN Phase 1 with no refusal.
	claude := load0084(t, "refusals-0062/messages/claude-string-schema-tool.json")
	for _, label := range []string{"opencode", "opencode-zen/messages", "opencode-go/messages"} {
		if apiErr, err := toolSchemaRefused(t, label, claude.Status, claude.Body, nil); errors.Is(err, ErrToolSchemaRefused) ||
			apiErr.Kind != ErrInvalidRequest { //nolint:errorlint // the kind itself
			t.Errorf("Anthropic's refusal under %s: %v; want ErrInvalidRequest", label, err)
		}
	}

	// ST25: an overflow alone.
	overflow := `{"type":"error","error":{"type":"invalid_request_error","message":"prompt is too long: 250000 tokens > 200000 maximum"}}`
	if apiErr, err := toolSchemaRefused(t, "claude", 400, overflow, nil); apiErr.Kind != ErrContextOverflow { //nolint:errorlint // the kind itself
		t.Errorf("an overflow: %v; want ErrContextOverflow", err)
	}
}

// TestToolSchemaRefusal_KeyRowFirst_0084 (MADR ST21): a Grok 400 that carries
// the tag and names an API key is a refused credential, as xAI sends one; the
// kind is checked after the key row. The refusal is F1 with its tool renamed
// "api key", a named edit of the recorded body.
func TestToolSchemaRefusal_KeyRowFirst_0084(t *testing.T) {
	f := load0084(t, "refusals-0062/responses/grok-string-schema-tool.json")
	body := strings.Replace(f.Body, "record_city", "api key", 1)
	if body == f.Body {
		t.Fatal("the record holds no tool name to rename")
	}
	apiErr, err := toolSchemaRefused(t, "grok", f.Status, body, nil)
	if apiErr.Kind != ErrAuthFailure || errors.Is(err, ErrToolSchemaRefused) { //nolint:errorlint // the kind itself
		t.Errorf("the tag with an API key named: %v, Kind %v; want ErrAuthFailure, not the kind", err, apiErr.Kind)
	}
}

// TestToolSchemaRefusal_ProxyAltered_0084 (MADR ST20, ST23, ST50; the plan's
// Rules, N7): a proxy that changes the status or replaces the body gives no
// D3 row its marker's status or its body, so each reply keeps the kind its
// status gave before: 401, 403, 404, 422, 429, 502, and the body as an HTML
// page at 400; a 200 is no error. The bodies are the 0062 refusals of the four
// forms; each edit is named: the status served, or the body wrapped in a page.
// 404 and 422 pin D3's 400 against widening to any status that maps to an
// invalid request (the review of build pass 1, 2026-11-21).
func TestToolSchemaRefusal_ProxyAltered_0084(t *testing.T) {
	for _, tc := range refusals0062 {
		f := load0084(t, tc.file)
		page := "<html><body><h1>Bad Request</h1><pre>" + f.Body + "</pre></body></html>"
		for _, label := range tc.labels {
			for _, alt := range []struct {
				name   string
				status int
				body   string
				want   error
			}{
				{"401", 401, f.Body, ErrAuthFailure},
				{"403", 403, f.Body, ErrNotPermitted},
				{"404", 404, f.Body, ErrInvalidRequest},
				{"422", 422, f.Body, ErrInvalidRequest},
				{"429", 429, f.Body, ErrRateLimited},
				{"502", 502, f.Body, ErrProviderUnavailable},
				{"an HTML page at 400", 400, page, ErrInvalidRequest},
			} {
				apiErr, err := toolSchemaRefused(t, label, alt.status, alt.body, nil)
				if errors.Is(err, ErrToolSchemaRefused) || !errors.Is(err, alt.want) || apiErr.Tool != "" {
					t.Errorf("%s (%s) served as %s: %v; want %v, not the kind", f.Case, label, alt.name, err, alt.want)
				}
			}
			if err := classifyFixture(label, 200, f.Body, nil); err != nil {
				t.Errorf("%s (%s) served as 200: %v; want no error", f.Case, label, err)
			}
		}
	}
}

// errorObject is the JSON object under "error" in a recorded body, as a
// stream's failure event carries it, or nil when there is none.
func errorObject(body string) []byte {
	var top struct {
		Error json.RawMessage `json:"error"`
	}
	if json.Unmarshal([]byte(body), &top) != nil || len(top.Error) == 0 || top.Error[0] != '{' {
		return nil
	}
	return top.Error
}

// TestToolSchemaRefusal_StreamUnchanged_0084 (MADR ST17; the owner's ruling
// on N4): a failure inside a 200 stream is never the kind, whatever it
// carries: the D3 rows apply to an HTTP error reply only, though a code that
// is an HTTP status is classified as one (H2). The guard, a failure event
// holding each recorded refusal as its code 400 and error object, and the
// OpenAI form with its own code, each keeps the kind it had.
func TestToolSchemaRefusal_StreamUnchanged_0084(t *testing.T) {
	for _, tc := range []struct {
		file, label, code, errType string
		want                       error
	}{
		// F7's error object, as OpenCode Zen's Google route's stream carries
		// it with the status as the code.
		{"refusals-0062/generatecontent/opencode-zen-google-string-schema-tool.json", "opencode-zen/google", "400",
			"INVALID_ARGUMENT", ErrInvalidRequest},
		{"refusals-0062/responses/grok-string-schema-tool.json", "grok", "400", "", ErrInvalidRequest},
		{"refusals-0062/responses/openai-string-schema-tool.json", "openai", "400", "invalid_request_error", ErrInvalidRequest},
		{"refusals-0062/messages/claude-string-schema-tool.json", "claude", "400", "invalid_request_error", ErrInvalidRequest},
		{"toolschema-0084/responses/zen-responses-root-string.json", "opencode-zen/responses", "400",
			"invalid_request_error", ErrInvalidRequest},
		// F2 with its own code, as a Responses stream's response.failed
		// carries it.
		{"refusals-0062/responses/openai-string-schema-tool.json", "openai", "invalid_function_parameters",
			"invalid_request_error", ErrInvalidRequest},
	} {
		f := load0084(t, tc.file)
		err := ClassifyStreamFailureBody(tc.label, tc.code, tc.errType, f.Message, errorObject(f.Body))
		apiErr, ok := errors.AsType[*APIError](err)
		if !ok || errors.Is(err, ErrToolSchemaRefused) || apiErr.Kind != tc.want || apiErr.Tool != "" { //nolint:errorlint // the kind itself
			t.Errorf("%s as a stream failure with code %s (%s): %v; want %v, not the kind", f.Case, tc.code, tc.label, err,
				tc.want)
		}
	}
}

// TestToolSchemaRefusal_Precedence_0084 (MADR ST26, ST32; the owner's ruling
// on N5, "Existing rules win"): a reply the service marks x-should-retry: true
// is ErrProviderUnavailable, retryable, with no Tool, as any invalid request
// is; and a refusal whose message also says the prompt is too long is
// ErrContextOverflow. The edits are named: the header added, a phrase from the
// overflow table added to F4's message.
func TestToolSchemaRefusal_Precedence_0084(t *testing.T) {
	for _, tc := range refusals0062 {
		f := load0084(t, tc.file)
		for _, label := range tc.labels {
			apiErr, err := toolSchemaRefused(t, label, f.Status, f.Body, http.Header{"X-Should-Retry": {"true"}})
			if apiErr.Kind != ErrProviderUnavailable || errors.Is(err, ErrToolSchemaRefused) || !apiErr.Retryable() || //nolint:errorlint // the kind itself
				apiErr.Tool != "" {
				t.Errorf("%s (%s) with x-should-retry: true: %v; want a retryable ErrProviderUnavailable, not the kind", f.Case,
					label, err)
			}
		}
	}
	f := load0084(t, "toolschema-0084/messages/claude-root-string.json")
	body := strings.Replace(f.Body, "Input should be 'object'", "Input should be 'object'. prompt is too long", 1)
	if body == f.Body {
		t.Fatal("the record holds no phrase to extend")
	}
	apiErr, err := toolSchemaRefused(t, "claude", f.Status, body, nil)
	if apiErr.Kind != ErrContextOverflow || errors.Is(err, ErrToolSchemaRefused) { //nolint:errorlint // the kind itself
		t.Errorf("a refusal that also says the prompt is too long: %v; want ErrContextOverflow, not the kind", err)
	}
}

// TestToolSchemaRefusal_Kinds_0084 (MADR D1): the sentinel is a child of
// ErrInvalidRequest and of nothing else, terminal as every one of them is;
// the kinds it must not be confused with are not it.
func TestToolSchemaRefusal_Kinds_0084(t *testing.T) {
	if !errors.Is(ErrToolSchemaRefused, ErrInvalidRequest) {
		t.Error("ErrToolSchemaRefused does not match ErrInvalidRequest")
	}
	for _, other := range []error{ErrRateLimited, ErrProviderUnavailable, ErrAuthFailure, ErrContextOverflow, ErrQuotaExhausted} {
		if errors.Is(ErrToolSchemaRefused, other) || errors.Is(other, ErrToolSchemaRefused) {
			t.Errorf("ErrToolSchemaRefused and %v match each other", other)
		}
	}
	if (&APIError{Kind: ErrToolSchemaRefused}).Retryable() {
		t.Error("an error of kind ErrToolSchemaRefused is retryable")
	}
	if got := ErrToolSchemaRefused.Error(); got != "llmprovider: tool schema refused" {
		t.Errorf("the kind's text is %q", got)
	}
	// The Tool field's text: set, it is held and not printed.
	e := &APIError{Provider: "p", Status: 400, Kind: ErrToolSchemaRefused, Code: "c", Message: "m", Tool: "named"}
	if slices.Contains(strings.Fields(e.Error()), "named") {
		t.Errorf("Error() %q prints the Tool", e.Error())
	}
}
