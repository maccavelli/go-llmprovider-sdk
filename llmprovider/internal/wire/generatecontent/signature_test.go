package generatecontent

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// TestItemFidelity_GeminiReplaysSignature: the thoughtSignature Gemini issues
// with a call is kept on the item and sent back with it.
func TestItemFidelity_GeminiReplaysSignature(t *testing.T) {
	res, err := Decode(strings.NewReader(
		`{"candidates":[{"content":{"parts":[{"functionCall":{"name":"get_weather","args":{"city":"Paris"}},"thoughtSignature":"sig-abc"}]}}]}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	call, ok := res.Output[0].(llmprovider.FunctionCallItem)
	if !ok || call.Signature != "sig-abc" {
		t.Fatalf("item = %#v, want Signature sig-abc", res.Output[0])
	}
	contents := Contents([]llmprovider.Item{llmprovider.MessageItem{Role: wire.RoleUser, Text: "weather?"}, call,
		llmprovider.FunctionCallOutputItem{CallID: call.CallID, Output: "sunny"}})
	mustEqualJSON(t, contents[1], `{"role":"model","parts":[{"functionCall":{"name":"get_weather","args":{"city":"Paris"}},"thoughtSignature":"sig-abc"}]}`)
}

// mustEqualJSON compares got, through JSON, with want.
func mustEqualJSON(t *testing.T, got any, want string) {
	t.Helper()
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var g, w any
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("bad want: %v", err)
	}
	if !reflect.DeepEqual(g, w) {
		t.Fatalf("got  %s\nwant %s", raw, want)
	}
}
