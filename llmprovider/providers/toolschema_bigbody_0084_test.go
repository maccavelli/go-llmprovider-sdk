package providers

import (
	"strings"
	"testing"
)

// paddedBody is body, a JSON object, with a "pad" member added after its last
// member so that the whole is total bytes: a named edit of a recorded body
// (padding).
func paddedBody(t *testing.T, body string, total int) string {
	t.Helper()
	end := strings.LastIndex(body, "}")
	const open, closing = `,"pad":"`, `"}`
	n := total - end - len(open) - len(closing)
	if end < 0 || n < 0 {
		t.Fatalf("cannot pad a body of %d bytes to %d", len(body), total)
	}
	return body[:end] + open + strings.Repeat("x", n) + closing
}

// TestToolSchemaRefusal_BigBody_0084 (MADR ST43, ST86; the owner's ruling
// "change this to the recommended 1 MiB"): through Grok's Generate and
// Stream, a refusal whose body is 70 KiB, which the old 64 KiB read cut, and
// one of exactly 1 MiB, the cap, is the kind with its tool named. D2's name
// bound still holds: a name that lies past the 512 bytes Message keeps, here
// after 700 bytes of text in a 70 KiB body, is the kind with no Tool, never a
// part of the name. Each body is F1 with a named edit: padding after the error,
// or 700 bytes of text before the tag.
func TestToolSchemaRefusal_BigBody_0084(t *testing.T) {
	grok := readRecorded0084(t, "refusals-0062/responses/grok-string-schema-tool.json")
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"70 KiB", paddedBody(t, grok.Body, 70<<10), "record_city"},
		{"exactly 1 MiB", paddedBody(t, grok.Body, 1<<20), "record_city"},
		{"the name past the 512 bytes Message keeps", paddedBody(t, editMessage(t, grok.Body, func(msg string) string {
			return strings.Replace(msg, "[invalid_client_tool_schema]", strings.Repeat("y", 700)+" [invalid_client_tool_schema]", 1)
		}), 70<<10), ""},
	} {
		gen, stream := refusalErrors(t, routeGrok, asked(grok.tools()...), tc.body)
		wantTool(t, tc.name+" through Generate", gen, tc.want)
		wantTool(t, tc.name+" through Stream", stream, tc.want)
	}
}
