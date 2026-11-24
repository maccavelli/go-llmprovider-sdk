package wire

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestEstimateInput_0088 (0088-MADR ST87 as amended): the estimate counts a
// request's content as pi does, at 3.5 characters a token, rounded up: the
// instructions, each message's text, a reasoning item's text and not its
// signature or encrypted form, a call's name and arguments, a call output, a
// loaded tool's name, a tool sent (name, description, JSON schema) and not a
// deferred one, and an image at 4,800 characters whatever its size; a
// character is a code point, not a byte.
func TestEstimateInput_0088(t *testing.T) {
	schema := map[string]any{"type": "object"} // {"type":"object"}: 17 characters
	for _, tc := range []struct {
		name string
		req  *llmprovider.Request
		want int
	}{
		{"nothing", &llmprovider.Request{}, 0},
		{"350,000 characters of a message", &llmprovider.Request{Input: []llmprovider.Item{
			llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: strings.Repeat("a", 350000)}}}, 100000},
		{"one character rounds up", &llmprovider.Request{Instructions: "a"}, 1},
		{"code points, not bytes", &llmprovider.Request{Instructions: strings.Repeat("é", 7)}, 2},
		{"a 1 MiB image at 4,800 characters", &llmprovider.Request{Input: []llmprovider.Item{
			llmprovider.ImageItem{MediaType: "image/png", Data: make([]byte, 1<<20)}}}, 1372},
		{"reasoning text, not its signature or encrypted form", &llmprovider.Request{Input: []llmprovider.Item{
			llmprovider.ReasoningItem{Text: strings.Repeat("r", 7), Signature: strings.Repeat("s", 7000),
				Encrypted: strings.Repeat("e", 7000)}}}, 2},
		{"a call, its output, a loaded tool", &llmprovider.Request{Input: []llmprovider.Item{
			llmprovider.FunctionCallItem{CallID: "call_1", Name: "abc", Arguments: "{}"},
			llmprovider.FunctionCallOutputItem{CallID: "call_1", Output: "xy"},
			llmprovider.ToolLoadItem{CallID: "call_1", Name: "z"}}}, 3},
		{"a tool sent, not a deferred one", &llmprovider.Request{Tools: []llmprovider.Tool{
			{Name: "ab", Description: "cd", Schema: schema},
			{Name: "deferred", Description: strings.Repeat("d", 7000), Schema: schema, Deferred: true}}}, 6},
	} {
		if got := EstimateInput(tc.req); got != tc.want {
			t.Errorf("%s: EstimateInput = %d; want %d", tc.name, got, tc.want)
		}
	}
}

// TestFit_0088 (0088-MADR ST87, ST105): a value is held to the window less
// the estimate less 4,096, never below the wire's minimum, nor at least 1;
// with no window it is as it was.
func TestFit_0088(t *testing.T) {
	for _, tc := range []struct {
		name   string
		tokens int
		in     LimitInput
		want   int
	}{
		{"no window", 60000, LimitInput{Estimate: 100000}, 60000},
		{"the room", 60000, LimitInput{Window: 131072, Estimate: 100000}, 26976},
		{"room to spare", 20000, LimitInput{Window: 131072, Estimate: 100000}, 20000},
		{"no room: the minimum", 60000, LimitInput{Window: 100000, Estimate: 100000, Minimum: 16}, 16},
		{"no room, no minimum: 1", 60000, LimitInput{Window: 100000, Estimate: 100000}, 1},
	} {
		if got := fit(tc.tokens, tc.in); got != tc.want {
			t.Errorf("%s: fit = %d; want %d", tc.name, got, tc.want)
		}
	}
}
