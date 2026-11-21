package deferral

import (
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestListed_0084 (0084-MADR D4, ST10, ST72): the names of the list a form
// sends, in order. In the Dispatch form the SDK's own use_tool wrapper is
// listed under no name, so a refusal is never taken to name it; in any other
// form a caller's tool of that name is an ordinary one, and a Search tool is
// listed under its own name.
func TestListed_0084(t *testing.T) {
	req := &llmprovider.Request{Tools: []llmprovider.Tool{{Name: "read_file"}, {Name: "tool_search", Search: true},
		{Name: "get_weather", Deferred: true}}}
	if got, want := Listed(Tools(req, Dispatch), Dispatch), []string{"read_file", "tool_search", ""}; !slices.Equal(got, want) {
		t.Errorf("Dispatch: %v; want %v, the wrapper nameless", got, want)
	}
	if got, want := Listed(Tools(req, ResponsesNative), ResponsesNative), []string{"read_file", "tool_search"}; !slices.Equal(got, want) {
		t.Errorf("ResponsesNative: %v; want %v, the Search tool under its own name", got, want)
	}
	if got, want := Listed(Tools(req, MessagesNative), MessagesNative), []string{"read_file", "tool_search", "get_weather"}; !slices.Equal(got, want) {
		t.Errorf("MessagesNative: %v; want %v", got, want)
	}
	// No deferral: a caller's use_tool is an ordinary tool (ST72).
	plain := &llmprovider.Request{Tools: []llmprovider.Tool{{Name: "use_tool"}, {Name: "x"}}}
	if got, want := Listed(Tools(plain, Append), Append), []string{"use_tool", "x"}; !slices.Equal(got, want) {
		t.Errorf("Append: %v; want %v", got, want)
	}
}
