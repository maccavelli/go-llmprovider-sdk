package chatcompletions

import (
	"path/filepath"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestRefusal_OllamaThinking_0061 (0061-MADR, amendment "Ollama's refusal of
// thinking, learned"): Ollama's recorded refusal of an effort by a model that
// does not think is read as a refusal of the field, whatever effort was
// sent; the same words under another code or status, with no effort sent, or
// about something else, are not read.
func TestRefusal_OllamaThinking_0061(t *testing.T) {
	fixtures := refusalFixtures(t, filepath.Join("..", "testdata", "refusals-0061", "chatcompletions"))
	if len(fixtures) != 1 {
		t.Fatalf("%d fixtures; want Ollama's one", len(fixtures))
	}
	r := fixtures[0]
	for _, sent := range []llmprovider.Effort{r.sent(), llmprovider.EffortLow, llmprovider.EffortHigh} {
		if got, ok := MatchRefusal(r.err(), sent); !ok || !got.Field || got.Effort != "" || got.Shape != "" ||
			got.Listed || got.Accepted != nil {
			t.Errorf("%s, sent %s: %+v, %t; want the field refused", r.Case, sent, got, ok)
		}
	}
	refusal := func(status int, code, message string) error {
		return &llmprovider.APIError{Provider: "ollama", Status: status, Code: code, Message: message}
	}
	for name, err := range map[string]error{
		"another code":     refusal(400, "api_error", r.Message),
		"another status":   refusal(500, r.Code, r.Message),
		"another refusal":  refusal(400, r.Code, `"gemma3:12b" does not support tools`),
		"no quoted model":  refusal(400, r.Code, "the model does not support thinking"),
		"a longer message": refusal(400, r.Code, r.Message+", try another model"),
	} {
		if got, ok := MatchRefusal(err, llmprovider.EffortMax); ok {
			t.Errorf("%s: matched %+v; want no refusal read", name, got)
		}
	}
	if got, ok := MatchRefusal(r.err(), ""); ok {
		t.Errorf("no effort sent: matched %+v; want no refusal read", got)
	}
}
