package responses

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestImage_Encode_0061 (0061-MADR D5): the user's text then an image by its
// bytes then one by URL go as one user input item, its content parts an
// input_text, then input_image parts whose image_url is a data URL or the
// URL; an image after a function_call_output opens a user item of its own; a
// message with no image is still a string. InputLoading and InputDispatch,
// with a deferred tool and a search tool, send each as Input does
// (0061-MADR, amendment "D5 made exact before Phase 6", item 6).
func TestImage_Encode_0061(t *testing.T) {
	png := llmprovider.ImageItem{MediaType: "image/png", Data: []byte("PNG")} // base64 "UE5H"
	byURL := llmprovider.ImageItem{Role: llmprovider.RoleUser, URL: "https://example.com/cat.png"}
	look := llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "look"}
	deferring := &llmprovider.Request{Tools: []llmprovider.Tool{{Name: "read"},
		{Name: "tool_search", Search: true}, {Name: "lookup", Deferred: true}}}
	for _, c := range []struct {
		name  string
		items []llmprovider.Item
		want  string
	}{
		{"joined", []llmprovider.Item{look, png, byURL},
			`[{"role":"user","content":[{"type":"input_text","text":"look"},` +
				`{"type":"input_image","image_url":"data:image/png;base64,UE5H"},` +
				`{"type":"input_image","image_url":"https://example.com/cat.png"}]}]`},
		{"after a result", []llmprovider.Item{
			llmprovider.FunctionCallItem{CallID: "c1", Name: "read", Arguments: "{}"},
			llmprovider.FunctionCallOutputItem{CallID: "c1", Output: "a picture"}, png},
			`[{"type":"function_call","call_id":"c1","name":"read","arguments":"{}"},` +
				`{"type":"function_call_output","call_id":"c1","output":"a picture"},` +
				`{"role":"user","content":[{"type":"input_image","image_url":"data:image/png;base64,UE5H"}]}]`},
		{"after the assistant", []llmprovider.Item{
			llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "send it"}, byURL},
			`[{"role":"assistant","content":"send it"},` +
				`{"role":"user","content":[{"type":"input_image","image_url":"https://example.com/cat.png"}]}]`},
		{"no image", []llmprovider.Item{look}, `[{"role":"user","content":"look"}]`},
	} {
		var want any
		if err := json.Unmarshal([]byte(c.want), &want); err != nil {
			t.Fatal(err)
		}
		for name, input := range map[string][]InputItem{
			"Input":         Input(c.items),
			"InputLoading":  InputLoading(c.items, deferring),
			"InputDispatch": InputDispatch(c.items, deferring),
		} {
			raw, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			var got any
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s, %s =\n%s\nwant\n%s", c.name, name, raw, c.want)
			}
		}
	}
}

// TestMessageContent_0061: content with neither text nor parts is omitted
// from its item and, marshalled alone, is null.
func TestMessageContent_0061(t *testing.T) {
	raw, err := json.Marshal(InputItem{Type: &typeReasoning})
	if err != nil || string(raw) != `{"type":"reasoning"}` {
		t.Errorf("an item with no content = %s, %v; want the content left out", raw, err)
	}
	if raw, err := (MessageContent{}).MarshalJSON(); err != nil || string(raw) != "null" {
		t.Errorf("empty content = %s, %v; want null", raw, err)
	}
}
