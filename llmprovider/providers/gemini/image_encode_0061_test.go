package gemini

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestImage_Encode_0061 (0061-MADR D5, E14): the user's text then an image by
// its bytes then one by URL go as one user_input step, its content a text
// block then image blocks, with data and mime_type or with uri; an image
// after the model's output opens a user_input step of its own.
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
			`[{"type":"user_input","content":[{"type":"text","text":"look"},` +
				`{"type":"image","data":"UE5H","mime_type":"image/png"},` +
				`{"type":"image","uri":"https://example.com/cat.png"}]}]`},
		{"after the model", []llmprovider.Item{
			llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "send it"}, byURL},
			`[{"type":"model_output","content":[{"type":"text","text":"send it"}]},` +
				`{"type":"user_input","content":[{"type":"image","uri":"https://example.com/cat.png"}]}]`},
	} {
		t.Run(c.name, func(t *testing.T) {
			raw, err := json.Marshal(interactionsInput(c.items))
			if err != nil {
				t.Fatal(err)
			}
			var got, want any
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(c.want), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("interactionsInput =\n%s\nwant\n%s", raw, c.want)
			}
		})
	}
}
