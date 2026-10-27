package generatecontent

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestImage_Encode_0061 (0061-MADR D5): an image's bytes join the user
// content just before it as an inline_data part, under D5's keys; one after
// the model's turn opens a user content of its own; an image by URL is not
// sent here, since the google route drops it first, warned (the amendment
// "D5 made exact before Phase 6", items 4 and 5).
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
			`[{"role":"user","parts":[{"text":"look"},{"inline_data":{"mime_type":"image/png","data":"UE5H"}}]}]`},
		{"after the model", []llmprovider.Item{
			llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "send it"}, png},
			`[{"role":"model","parts":[{"text":"send it"}]},` +
				`{"role":"user","parts":[{"inline_data":{"mime_type":"image/png","data":"UE5H"}}]}]`},
		{"by URL only", []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "look"}, byURL},
			`[{"role":"user","parts":[{"text":"look"}]}]`},
	} {
		t.Run(c.name, func(t *testing.T) {
			raw, err := json.Marshal(Contents(c.items))
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
				t.Errorf("Contents =\n%s\nwant\n%s", raw, c.want)
			}
		})
	}
}
