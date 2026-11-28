package toolschema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// fixture0084 is the part of a recorded refusal the package reads: the types a
// classification holds, and the message.
type fixture0084 struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func readFixture0084(t *testing.T, rel string) fixture0084 {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "wire", "testdata", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	var f fixture0084
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
	return f
}

// TestRead_0084 (MADR D3): each recorded refusal is read as its own form, from
// its code and message; a record in a form not read, and a marker without its
// code, are None.
func TestRead_0084(t *testing.T) {
	for _, tc := range []struct {
		file string
		want Form
	}{
		{"refusals-0062/responses/grok-string-schema-tool.json", Grok},
		{"toolschema-0084/responses/grok-O2-two-bad.json", Grok},
		{"refusals-0062/responses/openai-string-schema-tool.json", OpenAI},
		{"refusals-0062/responses/opencode-zen-responses-string-schema-tool.json", OpenAI},
		{"refusals-0062/messages/claude-string-schema-tool.json", Anthropic},
		{"toolschema-0084/messages/claude-O1-root-union.json", Anthropic},
		{"refusals-0062/generatecontent/opencode-zen-google-string-schema-tool.json", Google},
		// Gemini's single-quoted name, read since 0095-MADR FU11.
		{"toolschema-0084/interactions/gemini-O1-root-union.json", Gemini},
		// Not read: OpenCode Zen's Google route's refusal of
		// additionalProperties (0095-MADR FU12).
		{"toolschema-0084/generatecontent/zen-google-O1-root-union.json", None},
		// Effort refusals of the same services.
		{"refusals-0062/responses/grok-4.5-max.json", None},
		{"refusals-0062/messages/claude-sonnet-5-adaptive-max.json", None},
	} {
		f := readFixture0084(t, tc.file)
		if got := Read([]string{f.Code}, f.Message); got != tc.want {
			t.Errorf("%s: %v; want %v", tc.file, got, tc.want)
		}
	}
	grok := readFixture0084(t, "refusals-0062/responses/grok-string-schema-tool.json")
	if got := Read([]string{"invalid_request_error"}, grok.Message); got != None {
		t.Errorf("Grok's tag under another type: %v; want None", got)
	}
	if got := Read([]string{"invalid-argument"}, "Invalid reasoning effort."); got != None {
		t.Errorf("Grok's code without the tag: %v; want None", got)
	}
	// Anthropic's marker needs its type, a path that begins the message, and
	// the field.
	for _, msg := range []string{"x tools.1.custom.input_schema.type: bad", "tools.1.custom.type: bad", "toolbox.1.input_schema"} {
		if got := Read([]string{"invalid_request_error"}, msg); got != None {
			t.Errorf("Anthropic's marker in %q: %v; want None", msg, got)
		}
	}
}

// TestReferences_0084 (MADR D2, D3): what each form's message names: Grok's
// name between the tag and its colon, OpenAI's inside function '...', the
// Google route's in backticks before its phrase, Anthropic's positions; each
// occurrence is read, a name cut short or missing is an empty name, and a
// position is read once.
func TestReferences_0084(t *testing.T) {
	for _, tc := range []struct {
		name    string
		form    Form
		message string
		want    Refs
	}{
		{"grok", Grok, "Failed to start sampling: [invalid_client_tool_schema] record_city: tool parameter root", Refs{Names: []string{"record_city"}}},
		{"grok, a redacted neighbour", Grok, "[invalid_client_tool_schema] lookup_api_key: [REDACTED] parameter root",
			Refs{Names: []string{"lookup_api_key"}}},
		{"grok, no name", Grok, "[invalid_client_tool_schema] : tool parameter", Refs{Names: []string{""}}},
		{"grok, cut before the colon", Grok, "[invalid_client_tool_schema] record_ci", Refs{Names: []string{""}}},
		{"grok, two", Grok, "[invalid_client_tool_schema] a: x [invalid_client_tool_schema] b: y", Refs{Names: []string{"a", "b"}}},
		{"openai", OpenAI, "Invalid schema for function 'record_city': schema must be", Refs{Names: []string{"record_city"}}},
		{"openai, no name", OpenAI, "Invalid schema for function '': schema", Refs{Names: []string{""}}},
		{"openai, cut", OpenAI, "Invalid schema for function 'record_c", Refs{Names: []string{""}}},
		{"openai, two", OpenAI, "function 'a' and function 'b'", Refs{Names: []string{"a", "b"}}},
		{"google", Google, "Unable to submit request because `record_city` functionDeclaration parameters schema should be",
			Refs{Names: []string{"record_city"}}},
		{"google, cut inside the name", Google, "Unable to submit request because `record_c", Refs{}},
		{"google, no name", Google, "because `` functionDeclaration parameters schema", Refs{Names: []string{""}}},
		{"google, two", Google, "`a` functionDeclaration parameters schema, then `b` functionDeclaration parameters schema",
			Refs{Names: []string{"a", "b"}}},
		{"anthropic", Anthropic, "tools.1.custom.input_schema.type: Input should be 'object'", Refs{Indexes: []int{1}}},
		{"anthropic, two, one repeated", Anthropic, "tools.1.custom.input_schema: x; tools.2.custom.input_schema: y; tools.1.custom.z",
			Refs{Indexes: []int{1, 2}}},
		{"none", None, "tools.1. function 'a'", Refs{}},
	} {
		if got := References(tc.form, tc.message); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %+v; want %+v", tc.name, got, tc.want)
		}
	}
}
