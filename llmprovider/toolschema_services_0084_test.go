package llmprovider

import (
	"errors"
	"strings"
	"testing"
)

// TestToolSchemaRefusal_OtherServices_0084 (MADR N3, ST07; the review of
// build pass 1, 2026-11-21): the kind is read by service and marker, so a
// form served under a service that was not recorded sending it keeps the kind
// that reply has always had, ErrInvalidRequest. The bodies are the 0062
// refusals F1, F2, F4 and F7, unedited; only the label differs. Every form
// under the services that read none (Gemini, Kilo, Hugging Face, Ollama,
// Together, and a chatcompat id), and each form under the other read
// services.
func TestToolSchemaRefusal_OtherServices_0084(t *testing.T) {
	forms := map[string]recorded0084{
		"grok":      load0084(t, "refusals-0062/responses/grok-string-schema-tool.json"),
		"openai":    load0084(t, "refusals-0062/responses/openai-string-schema-tool.json"),
		"anthropic": load0084(t, "refusals-0062/messages/claude-string-schema-tool.json"),
		"google":    load0084(t, "refusals-0062/generatecontent/opencode-zen-google-string-schema-tool.json"),
	}
	every := []string{"grok", "openai", "anthropic", "google"}
	for _, tc := range []struct {
		labels []string
		forms  []string
	}{
		// A service that reads none. "my-compat-server" is the id a caller
		// gives a chatcompat provider.
		{[]string{"gemini", "kilo", "huggingface", "ollama", "together", "my-compat-server"}, every},
		{[]string{"openai", "claude", "opencode"}, []string{"grok"}},
		{[]string{"grok", "claude"}, []string{"openai"}},
		{[]string{"openai", "grok", "claude", "gemini"}, []string{"google"}},
		{[]string{"grok", "openai", "gemini", "kilo"}, []string{"anthropic"}},
	} {
		for _, label := range tc.labels {
			for _, form := range tc.forms {
				f := forms[form]
				apiErr, err := toolSchemaRefused(t, label, f.Status, f.Body, nil)
				if errors.Is(err, ErrToolSchemaRefused) || apiErr.Kind != ErrInvalidRequest || apiErr.Tool != "" { //nolint:errorlint // the kind itself
					t.Errorf("%s's form under %s: %v, Kind %v; want ErrInvalidRequest as before, not the kind", form, label, err,
						apiErr.Kind)
				}
			}
		}
	}
}

// padded0084 is body, a JSON object, with a "pad" member added after its last
// member so that the whole is exactly total bytes: a named edit of a recorded
// body (padding). It fails the test when body is longer than total allows.
func padded0084(t *testing.T, body string, total int) string {
	t.Helper()
	end := strings.LastIndex(body, "}")
	const open, closing = `,"pad":"`, `"}`
	n := total - end - len(open) - len(closing)
	if end < 0 || n < 0 {
		t.Fatalf("cannot pad a body of %d bytes to %d", len(body), total)
	}
	return body[:end] + open + strings.Repeat("x", n) + closing
}

// TestToolSchemaRefusal_BodyLimit_0084 (MADR ST43, ST86; the owner's ruling
// "change this to the recommended 1 MiB"): an error body is read to 1 MiB, so
// a refusal whose body is up to that long parses and gets the kind, on every
// form: at 70 KiB, which the old 64 KiB read cut, and at exactly 1 MiB, the
// cap. A body whose JSON the read cuts keeps the kind it has always had: here
// one byte over the cap, which drops the object's closing brace. A body over
// 1 MiB whose error object closes inside the read, followed only by
// whitespace, parses and gets the kind: the case is a cut object, not a size.
// Each body is a recorded one with a named edit: padding in a member after the
// error, or whitespace after the object.
func TestToolSchemaRefusal_BodyLimit_0084(t *testing.T) {
	const limit = 1 << 20
	for _, tc := range refusals0062 {
		f := load0084(t, tc.file)
		for _, label := range tc.labels {
			for _, c := range []struct {
				name string
				body string
				kind bool // whether the body gets the kind
			}{
				{"70 KiB", padded0084(t, f.Body, 70<<10), true},
				{"exactly 1 MiB, the cap", padded0084(t, f.Body, limit), true},
				{"one byte over the cap, the object cut", padded0084(t, f.Body, limit+1), false},
				{"2 MiB whose object closes early", f.Body + strings.Repeat(" ", 2*limit), true},
			} {
				apiErr, err := toolSchemaRefused(t, label, f.Status, c.body, nil)
				switch {
				case c.kind && !errors.Is(err, ErrToolSchemaRefused):
					t.Errorf("%s (%s), %s: Kind %v; want ErrToolSchemaRefused", f.Case, label, c.name, apiErr.Kind)
				case !c.kind && (errors.Is(err, ErrToolSchemaRefused) || apiErr.Kind != ErrInvalidRequest): //nolint:errorlint // the kind itself
					t.Errorf("%s (%s), %s: Kind %v; want the ErrInvalidRequest it had before, not the kind", f.Case, label, c.name,
						apiErr.Kind)
				}
				if apiErr.Tool != "" || len(apiErr.Message) > apiErrorMessageLimit+len("…") {
					t.Errorf("%s (%s), %s: Tool %q, Message of %d bytes; want none and a bounded Message", f.Case, label, c.name,
						apiErr.Tool, len(apiErr.Message))
				}
			}
		}
	}
}

// TestClassify_ErrorBodyLimit_0084 (MADR ST86): the limit is the module's one
// error-body limit, not the refusal's: an OpenAI quota body, a 429 with its
// code, is ErrQuotaExhausted at 200 KiB, which the old 64 KiB read cut, and
// at exactly 1 MiB; one byte over, its JSON cut, it is the plain rate limit
// its status gives. Message stays bounded to 512 bytes and Code to 128.
func TestClassify_ErrorBodyLimit_0084(t *testing.T) {
	const limit = 1 << 20
	quota := `{"error":{"type":"insufficient_quota","code":"insufficient_quota","message":"You exceeded your current quota."}}`
	for _, tc := range []struct {
		name  string
		body  string
		quota bool
	}{
		{"200 KiB", padded0084(t, quota, 200<<10), true},
		{"exactly 1 MiB", padded0084(t, quota, limit), true},
		{"one byte over, cut", padded0084(t, quota, limit+1), false},
	} {
		apiErr, err := toolSchemaRefused(t, "openai", 429, tc.body, nil)
		if got := errors.Is(err, ErrQuotaExhausted); got != tc.quota || !errors.Is(err, ErrRateLimited) {
			t.Errorf("%s: %v; want quota exhausted: %t, and a rate limit", tc.name, err, tc.quota)
		}
		if len(apiErr.Message) > apiErrorMessageLimit+len("…") || len(apiErr.Code) > 128 {
			t.Errorf("%s: Message of %d bytes, Code of %d; want them bounded", tc.name, len(apiErr.Message), len(apiErr.Code))
		}
	}
}
