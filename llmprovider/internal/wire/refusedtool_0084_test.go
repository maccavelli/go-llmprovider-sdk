package wire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0084-MADR D2 and D4 (ST33, ST34): NameRefusedTool sets Tool on the error of
// kind ErrToolSchemaRefused it finds, and on none other; the error it is given
// comes back. The refusals are the 0062 recordings, classified as a reply is.

// refusal0084 classifies the recorded body at rel (under testdata) as label
// sends it, and returns the *APIError.
func refusal0084(t *testing.T, rel, label string) *llmprovider.APIError {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	var rec struct {
		Status int    `json:"status"`
		Body   string `json:"body"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
	err = llmprovider.ClassifyHTTPError(label, &http.Response{StatusCode: rec.Status, Header: http.Header{},
		Body: io.NopCloser(strings.NewReader(rec.Body))})
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	if !ok {
		t.Fatalf("%s: %v; want an *APIError", rel, err)
	}
	return apiErr
}

// held0084 is the list of the recorded run: noop, then the refused tool.
var held0084 = Held{Listed: []string{"noop", "record_city"}}

// TestNameRefusedTool_PassThrough_0084 (MADR ST34): any other error comes
// back unchanged with no Tool: nil, a plain error, a cancelled or timed-out
// context, and an *APIError of every other kind, whatever its message
// names. The *APIErrors are F1's, classified, with the kind replaced.
func TestNameRefusedTool_PassThrough_0084(t *testing.T) {
	if got := NameRefusedTool(nil, held0084); got != nil {
		t.Errorf("nil became %v", got)
	}
	for _, in := range []error{errors.New("plain"), context.Canceled, context.DeadlineExceeded,
		fmt.Errorf("wrapped: %w", context.Canceled)} {
		if got := NameRefusedTool(in, held0084); got != in { //nolint:errorlint // the same error, not one it wraps
			t.Errorf("%v became %v", in, got)
		}
	}
	for _, kind := range []error{llmprovider.ErrInvalidRequest, llmprovider.ErrContextOverflow, llmprovider.ErrRateLimited,
		llmprovider.ErrAuthFailure, llmprovider.ErrQuotaExhausted, llmprovider.ErrProviderUnavailable, llmprovider.ErrNotPermitted,
		llmprovider.ErrIncomplete, nil} {
		in := refusal0084(t, "refusals-0062/responses/grok-string-schema-tool.json", "grok")
		in.Kind = kind
		in.Tool = ""
		if got := NameRefusedTool(in, held0084); got != error(in) { //nolint:errorlint // the same error, not one it wraps
			t.Errorf("kind %v: %v became %v", kind, in, got)
		}
		if in.Tool != "" {
			t.Errorf("kind %v: Tool set to %q on an error that is not the kind", kind, in.Tool)
		}
	}
}

// TestNameRefusedTool_Wrapped_0084 (MADR ST33; N6, "find the error through
// wrapping"): an error that wraps the kind, by %w or errors.Join, gets its
// Tool set on the *APIError found, and the error returned is the one given.
func TestNameRefusedTool_Wrapped_0084(t *testing.T) {
	for name, wrap := range map[string]func(error) error{
		"bare":      func(e error) error { return e },
		"%w":        func(e error) error { return fmt.Errorf("renewing: %w", e) },
		"deeper":    func(e error) error { return fmt.Errorf("a: %w", fmt.Errorf("b: %w", e)) },
		"Join":      func(e error) error { return errors.Join(errors.New("renewal failed"), e) },
		"Join last": func(e error) error { return errors.Join(e, errors.New("renewal failed")) },
	} {
		apiErr := refusal0084(t, "refusals-0062/responses/grok-string-schema-tool.json", "grok")
		in := wrap(apiErr)
		got := NameRefusedTool(in, held0084)
		if got != in { //nolint:errorlint // the same error, not one it wraps
			t.Errorf("%s: returned %v; want the error given, %v", name, got, in)
		}
		if apiErr.Tool != "record_city" {
			t.Errorf("%s: Tool %q; want record_city, set on the *APIError found", name, apiErr.Tool)
		}
		if !errors.Is(got, llmprovider.ErrToolSchemaRefused) {
			t.Errorf("%s: %v lost its kind", name, got)
		}
	}
}

// TestNameRefusedTool_Forms_0084 (MADR D2, D3; ST09, ST10, ST12, ST13): each
// recorded form is read from the *APIError's own fields, with no more than
// the names the body held: Grok's tag, OpenAI's function, Anthropic's index
// into the list as sent, OpenCode's backticks. A name or index the list does
// not give, a nameless entry, several names, an unsent name, and a name found
// beside the list (Held.Inside) each decide as the decision says.
func TestNameRefusedTool_Forms_0084(t *testing.T) {
	for _, tc := range []struct {
		name, file, label string
		held              Held
		message           func(string) string
		want              string
	}{
		{"grok", "refusals-0062/responses/grok-string-schema-tool.json", "grok", held0084, nil, "record_city"},
		{"openai", "refusals-0062/responses/openai-string-schema-tool.json", "openai", held0084, nil, "record_city"},
		{"google", "refusals-0062/generatecontent/opencode-zen-google-string-schema-tool.json", "opencode", held0084, nil, "record_city"},
		{"anthropic, index 1", "toolschema-0084/messages/claude-root-string.json", "claude", held0084, nil, "record_city"},
		{"anthropic, index 0", "refusals-0062/messages/claude-string-schema-tool.json", "claude", held0084, nil, "noop"},
		{"anthropic, past the list", "toolschema-0084/messages/claude-root-string.json", "claude",
			Held{Listed: []string{"noop"}}, nil, ""},
		{"anthropic, a nameless entry", "toolschema-0084/messages/claude-root-string.json", "claude",
			Held{Listed: []string{"noop", ""}}, nil, ""},
		{"grok, not in the list", "refusals-0062/responses/grok-string-schema-tool.json", "grok",
			Held{Listed: []string{"noop"}}, nil, ""},
		// A message that gives no name at all, no slot where one would sit (ST11,
		// ST42, ST43; the review of Phase 4), with two tools held and with one:
		// never the last tool, never the only one.
		{"openai, no name slot, two tools", "refusals-0062/responses/openai-string-schema-tool.json", "openai", held0084,
			func(m string) string { return strings.ReplaceAll(m, "function 'record_city'", "a function") }, ""},
		{"openai, no name slot, one tool", "refusals-0062/responses/openai-string-schema-tool.json", "openai",
			Held{Listed: []string{"record_city"}},
			func(m string) string { return strings.ReplaceAll(m, "function 'record_city'", "a function") }, ""},
		{"google, no name slot, two tools", "refusals-0062/generatecontent/opencode-zen-google-string-schema-tool.json", "opencode",
			held0084, func(m string) string { return strings.ReplaceAll(m, "`record_city` ", "") }, ""},
		{"google, no name slot, one tool", "refusals-0062/generatecontent/opencode-zen-google-string-schema-tool.json", "opencode",
			Held{Listed: []string{"record_city"}}, func(m string) string { return strings.ReplaceAll(m, "`record_city` ", "") }, ""},
		{"grok, in Inside only", "refusals-0062/responses/grok-string-schema-tool.json", "grok",
			Held{Listed: []string{"noop"}, Inside: []string{"record_city"}}, nil, "record_city"},
		{"grok, an entry of no name", "refusals-0062/responses/grok-string-schema-tool.json", "grok",
			Held{Listed: []string{"noop", "", "record_city"}}, nil, "record_city"},
		{"grok, one name twice listed", "refusals-0062/responses/grok-string-schema-tool.json", "grok",
			Held{Listed: []string{"record_city", "record_city"}}, nil, "record_city"},
		{"grok, two names", "refusals-0062/responses/grok-string-schema-tool.json", "grok",
			Held{Listed: []string{"record_city", "record_country"}},
			func(m string) string { return m + " [invalid_client_tool_schema] record_country: tool parameter" }, ""},
		{"grok, a second name unsent", "refusals-0062/responses/grok-string-schema-tool.json", "grok", held0084,
			func(m string) string { return m + " [invalid_client_tool_schema] ghost: tool parameter" }, ""},
		{"grok, the same name twice in the message", "refusals-0062/responses/grok-string-schema-tool.json", "grok", held0084,
			func(m string) string { return m + " [invalid_client_tool_schema] record_city: tool parameter" }, "record_city"},
	} {
		apiErr := refusal0084(t, tc.file, tc.label)
		if tc.message != nil {
			apiErr.Message = tc.message(apiErr.Message)
		}
		got := NameRefusedTool(apiErr, tc.held)
		if got != error(apiErr) { //nolint:errorlint // the same error, not one it wraps
			t.Errorf("%s: returned another error", tc.name)
		}
		if apiErr.Tool != tc.want {
			t.Errorf("%s: Tool %q; want %q", tc.name, apiErr.Tool, tc.want)
		}
	}
}

// TestNameRefusedStream_0084 (MADR D4): a stream's events pass through, and
// the error it yields is named; one that is not the kind is yielded as it was.
func TestNameRefusedStream_0084(t *testing.T) {
	refused := refusal0084(t, "refusals-0062/responses/grok-string-schema-tool.json", "grok")
	other := errors.New("not a refusal")
	seq := func(yield func(llmprovider.Event, error) bool) {
		if !yield(llmprovider.Event{Type: llmprovider.EventDone}, nil) {
			return
		}
		if !yield(llmprovider.Event{}, refused) {
			return
		}
		yield(llmprovider.Event{}, other)
	}
	var errs []error
	events := 0
	for ev, err := range NameRefusedStream(seq, held0084) {
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if ev.Type == llmprovider.EventDone {
			events++
		}
	}
	if events != 1 || len(errs) != 2 || errs[0] != error(refused) || errs[1] != other { //nolint:errorlint // the same errors
		t.Fatalf("events %d, errors %v; want the done event and the two errors as given", events, errs)
	}
	if refused.Tool != "record_city" {
		t.Errorf("Tool %q; want record_city", refused.Tool)
	}
}
