package messages

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0028-PLAN Phase 8, step 5: messages.FromItems for 100 plain messages, against a typed
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

type typedMessage struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

func typedPrototype(items []llmprovider.Item) []typedMessage {
	out := make([]typedMessage, len(items))
	for i, item := range items {
		m := item.(llmprovider.MessageItem)
		out[i] = typedMessage{Content: m.Text, Role: string(m.Role)}
	}
	return out
}

// BenchmarkPlainMessagesMaps: the wire's own encoder.
func BenchmarkPlainMessagesMaps(b *testing.B) {
	items := plainItems(100)
	want, _ := json.Marshal(FromItems(items))
	got, _ := json.Marshal(typedPrototype(items))
	if string(got) != string(want) {
		b.Fatalf("the prototype's JSON differs:\n%s\n%s", got, want)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := json.Marshal(FromItems(items)); err != nil {
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
