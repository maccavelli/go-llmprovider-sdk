package messages

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// asAny is v as JSON decodes it, for comparing JSON.
func asAny(t *testing.T, v any) any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// TestImage_Encode_0061 (0061-MADR D5): the user's text then an image by its
// bytes then one by URL go as one user message, the text a text block, each
// image an image block with its source; an image after the assistant opens a
// user message of its own; one after a turn's results joins them.
func TestImage_Encode_0061(t *testing.T) {
	png := llmprovider.ImageItem{MediaType: "image/png", Data: []byte("PNG")} // base64 "UE5H"
	byURL := llmprovider.ImageItem{Role: llmprovider.RoleUser, URL: "https://example.com/cat.png"}
	for _, c := range []struct {
		name  string
		items []llmprovider.Item
		want  string
	}{
		{"joined to the user's text", []llmprovider.Item{
			llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "look"}, png, byURL},
			`[{"role":"user","content":[{"type":"text","text":"look"},` +
				`{"type":"image","source":{"type":"base64","media_type":"image/png","data":"UE5H"}},` +
				`{"type":"image","source":{"type":"url","url":"https://example.com/cat.png"}}]}]`},
		{"after the assistant", []llmprovider.Item{
			llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "send it"}, png},
			`[{"role":"assistant","content":"send it"},` +
				`{"role":"user","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"UE5H"}}]}]`},
		{"after a turn's results", []llmprovider.Item{
			llmprovider.FunctionCallItem{CallID: "toolu_1", Name: "read", Arguments: "{}"},
			llmprovider.FunctionCallOutputItem{CallID: "toolu_1", Output: "a picture"}, byURL},
			`[{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"read","input":{}}]},` +
				`{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"a picture"},` +
				`{"type":"image","source":{"type":"url","url":"https://example.com/cat.png"}}]}]`},
		{"alone", []llmprovider.Item{png},
			`[{"role":"user","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"UE5H"}}]}]`},
	} {
		t.Run(c.name, func(t *testing.T) {
			var want any
			if err := json.Unmarshal([]byte(c.want), &want); err != nil {
				t.Fatal(err)
			}
			if got := asAny(t, FromItems(c.items)); !reflect.DeepEqual(got, want) {
				t.Errorf("FromItems =\n%v\nwant\n%v", got, want)
			}
			if got := asAny(t, FromItemsLoading(c.items)); !reflect.DeepEqual(got, want) {
				t.Errorf("FromItemsLoading =\n%v\nwant\n%v", got, want)
			}
		})
	}
}
