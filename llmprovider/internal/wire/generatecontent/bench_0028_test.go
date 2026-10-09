package generatecontent

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0028-PLAN Phase 8, step 5: generatecontent.Contents for 100 plain messages, against a typed
// prototype of the same JSON. Measured only; the wire is not changed here.

// plainItems is n plain messages, user and assistant in turn.
func plainItems(n int) []llmprovider.Item {
	items := make([]llmprovider.Item, n)
	for i := range items {
		role := llmprovider.RoleUser
		if i%2 == 1 {
			role = llmprovider.RoleAssistant
		}
		items[i] = llmprovider.MessageItem{Role: role, Text: fmt.Sprintf("message %d: a sentence of ordinary length for a chat turn.", i)}
	}
	return items
}

type typedPart struct {
	Text string `json:"text"`
}

type typedContent struct {
	Parts []typedPart `json:"parts"`
	Role  string      `json:"role"`
}

func typedPrototype(items []llmprovider.Item) []typedContent {
	out := make([]typedContent, len(items))
	for i, item := range items {
		m := item.(llmprovider.MessageItem)
		role := string(m.Role)
		if role == "assistant" {
			role = "model"
		}
		out[i] = typedContent{Parts: []typedPart{{Text: m.Text}}, Role: role}
	}
	return out
}

// BenchmarkPlainMessagesMaps: the wire's own encoder.
func BenchmarkPlainMessagesMaps(b *testing.B) {
	items := plainItems(100)
	want, _ := json.Marshal(Contents(items))
	got, _ := json.Marshal(typedPrototype(items))
	if string(got) != string(want) {
		b.Fatalf("the prototype's JSON differs:\n%s\n%s", got, want)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := json.Marshal(Contents(items)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPlainMessagesTyped: the typed prototype.
func BenchmarkPlainMessagesTyped(b *testing.B) {
	items := plainItems(100)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := json.Marshal(typedPrototype(items)); err != nil {
			b.Fatal(err)
		}
	}
}
