package providers

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// routeGemini is the gemini provider, on the Interactions API, as 0084-PLAN
// Phase 1 recorded it.
var routeGemini = toolRoute{"gemini", llmprovider.ProviderGemini, nil, "gemini-3.8-flash"}

// TestGeminiNamesRefusedTool_0095 (0095-MADR FU11): Gemini's recorded
// refusals, which name the tool in single quotes, are the kind, with Tool
// the tool the request sent at index 1, through Generate read over the
// stream, through a plain Generate (a limit given), and through Stream; a
// name the request did not send is the kind with Tool empty.
func TestGeminiNamesRefusedTool_0095(t *testing.T) {
	for _, file := range []string{
		"toolschema-0084/interactions/gemini-O1-root-union.json",
		"toolschema-0084/interactions/gemini-O2-two-bad.json",
	} {
		f := readRecorded0084(t, file)
		gen, stream := refusalErrors(t, routeGemini, asked(f.tools()...), f.Body)
		wantTool(t, f.Case+" through Generate", gen, "record_city")
		wantTool(t, f.Case+" through Stream", stream, "record_city")
		plain, _ := refusalErrors(t, routeGemini, asked(f.tools()...), f.Body, llmprovider.WithMaxTokens(8192))
		wantTool(t, f.Case+" through a plain Generate", plain, "record_city")
	}
	f := readRecorded0084(t, "toolschema-0084/interactions/gemini-O1-root-union.json")
	ghost := renamed(t, f.Body, "record_city", "ghost_tool")
	gen, stream := refusalErrors(t, routeGemini, asked(f.tools()...), ghost)
	wantTool(t, "a name the request did not send, through Generate", gen, "")
	wantTool(t, "a name the request did not send, through Stream", stream, "")
}
