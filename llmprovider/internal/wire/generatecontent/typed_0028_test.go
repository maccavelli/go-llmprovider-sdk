package generatecontent

import (
	"encoding/json"
	"math/rand/v2"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// generatedItems is the wire case's conversation and 300 generated sequences
// (PCG seed 28, 0) of every item type, in each variant the encoders read.
func generatedItems() [][]llmprovider.Item {
	corpus := [][]llmprovider.Item{wirecase.Items(), nil, {}}
	r := rand.New(rand.NewPCG(28, 0))
	texts := []string{"", "hello", "It is <b>sunny</b> & warm", "quote \" and \\ slash", "日本語 ✓", "line\nbreak"}
	args := []string{`{"city":"Paris"}`, `{}`, ``, `not json`, `[1,2]`, ` {"n": 12345678901234567890} `, `null`}
	ids := []string{"call_1", "call_2", "toolu_01AbC", "", "call with spaces", "fc_9"}
	formats := []string{"", wire.FormatMessages, wire.FormatResponses, wire.FormatChatCompletions}
	for range 300 {
		var items []llmprovider.Item
		for range 1 + r.IntN(12) {
			text := texts[r.IntN(len(texts))]
			switch r.IntN(10) {
			case 0:
				items = append(items, llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: text})
			case 1:
				items = append(items, llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: text})
			case 2:
				items = append(items, llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: text})
			case 3:
				items = append(items, llmprovider.MessageItem{Role: []llmprovider.Role{"", "developer"}[r.IntN(2)], Text: text})
			case 4, 5:
				items = append(items, llmprovider.FunctionCallItem{CallID: ids[r.IntN(len(ids))],
					Name: []string{"get_weather", "search"}[r.IntN(2)], Arguments: args[r.IntN(len(args))],
					Signature: []string{"", "sig-1"}[r.IntN(2)]})
			case 6:
				items = append(items, llmprovider.FunctionCallOutputItem{CallID: ids[r.IntN(len(ids))], Output: text})
			case 7, 8:
				items = append(items, llmprovider.ReasoningItem{Text: text,
					Signature: []string{"", "sig-r"}[r.IntN(2)], Encrypted: []string{"", "enc-blob"}[r.IntN(2)],
					Format: formats[r.IntN(len(formats))]})
			case 9:
				items = append(items, llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: text},
					llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: text})
			}
		}
		corpus = append(corpus, items)
	}
	return corpus
}

// sameJSON fails t when got and want marshal to different bytes.
func sameJSON(t *testing.T, name string, i int, got, want any) {
	t.Helper()
	g, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("%s case %d: %v", name, i, err)
	}
	w, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("%s case %d: reference: %v", name, i, err)
	}
	if string(g) != string(w) {
		t.Errorf("%s case %d:\n  got  %s\n  want %s", name, i, g, w)
	}
}

// TestContents_SameJSONAsReference (0028-MADR D-A6, amendment 2026-10-09):
// the typed encoders send the map encoders' JSON, byte for byte.
func TestContents_SameJSONAsReference(t *testing.T) {
	for i, items := range generatedItems() {
		sameJSON(t, "Contents", i, Contents(items), refContents(items))
		sameJSON(t, "SystemInstruction", i, SystemInstruction(items), refSystemInstruction(items))
	}
}
