package chatcompletions

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestImage_Encode_0061 (0061-MADR D5): the user's text then an image by its
// bytes then one by URL go as one user message, its content parts: a text
// part, then image_url parts, an object, or with ImageURLString the URL
// alone (Ollama, E9); an image after the assistant or a tool result opens a
// user message of its own; a message with no image is still a string.
func TestImage_Encode_0061(t *testing.T) {
	png := llmprovider.ImageItem{MediaType: "image/png", Data: []byte("PNG")} // base64 "UE5H"
	byURL := llmprovider.ImageItem{Role: llmprovider.RoleUser, URL: "https://example.com/cat.png"}
	look := llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "look"}
	for _, c := range []struct {
		name  string
		items []llmprovider.Item
		opts  Opts
		want  string
	}{
		{"joined, as objects", []llmprovider.Item{look, png, byURL}, Opts{},
			`[{"role":"user","content":[{"type":"text","text":"look"},` +
				`{"type":"image_url","image_url":{"url":"data:image/png;base64,UE5H"}},` +
				`{"type":"image_url","image_url":{"url":"https://example.com/cat.png"}}]}]`},
		{"joined, as strings", []llmprovider.Item{look, png}, Opts{ImageURLString: true},
			`[{"role":"user","content":[{"type":"text","text":"look"},` +
				`{"type":"image_url","image_url":"data:image/png;base64,UE5H"}]}]`},
		{"after the assistant", []llmprovider.Item{
			llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "send it"}, png}, Opts{},
			`[{"role":"assistant","content":"send it"},` +
				`{"role":"user","content":[{"type":"image_url","image_url":{"url":"data:image/png;base64,UE5H"}}]}]`},
		{"after a tool result", []llmprovider.Item{
			llmprovider.FunctionCallItem{CallID: "c1", Name: "read", Arguments: "{}"},
			llmprovider.FunctionCallOutputItem{CallID: "c1", Output: "a picture"}, byURL}, Opts{},
			`[{"role":"assistant","content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"read","arguments":"{}"}}]},` +
				`{"role":"tool","tool_call_id":"c1","content":"a picture"},` +
				`{"role":"user","content":[{"type":"image_url","image_url":{"url":"https://example.com/cat.png"}}]}]`},
		{"no image", []llmprovider.Item{look}, Opts{}, `[{"role":"user","content":"look"}]`},
	} {
		t.Run(c.name, func(t *testing.T) {
			raw, err := json.Marshal(Body("m", 16, c.items, c.opts)[keyMessages])
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
				t.Errorf("messages =\n%s\nwant\n%s", raw, c.want)
			}
		})
	}
}

// TestImage_ReplayLeavesTheUserParts_0061: a replayed reasoning field goes on
// the assistant message, and an image's user message keeps its parts.
func TestImage_ReplayLeavesTheUserParts_0061(t *testing.T) {
	items := []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "look"},
		llmprovider.ImageItem{MediaType: "image/png", Data: []byte("PNG")},
		llmprovider.ReasoningItem{Text: "a cat"},
		llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "A cat."},
	}
	raw, err := json.Marshal(Body("m", 16, items, Opts{ReplayReasoningField: "reasoning_content"})[keyMessages])
	if err != nil {
		t.Fatal(err)
	}
	const want = `[{"content":[{"text":"look","type":"text"},{"image_url":{"url":"data:image/png;base64,UE5H"},"type":"image_url"}],"role":"user"},` +
		`{"content":"A cat.","reasoning_content":"a cat","role":"assistant"}]`
	if string(raw) != want {
		t.Errorf("messages =\n%s\nwant\n%s", raw, want)
	}
}
