package responses

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0028-PLAN Phase 8, step 5: responses.Input for 100 plain messages, against a typed
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

// BenchmarkPlainMessagesMaps: the map encoder, frozen as the reference
// (0028-PLAN Phase 9b), for comparison.
func BenchmarkPlainMessagesMaps(b *testing.B) {
	items := plainItems(100)
	want, _ := json.Marshal(refInput(items))
	got, _ := json.Marshal(typedPrototype(items))
	if string(got) != string(want) {
		b.Fatalf("the prototype's JSON differs:\n%s\n%s", got, want)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := json.Marshal(refInput(items)); err != nil {
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

// historyItems is n items in turns of a user message, an assistant message,
// a call with about 1 KiB of arguments, and its output.
func historyItems(n int) []llmprovider.Item {
	args := `{"query":"` + strings.Repeat("x", 1000) + `"}`
	items := make([]llmprovider.Item, 0, n)
	for i := 0; len(items) < n; i++ {
		id := fmt.Sprintf("call_%d", i)
		items = append(items,
			llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: fmt.Sprintf("question %d", i)},
			llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "Let me look that up."},
			llmprovider.FunctionCallItem{CallID: id, Name: "search", Arguments: args},
			llmprovider.FunctionCallOutputItem{CallID: id, Output: `{"result":"found"}`})
	}
	return items[:n]
}

// BenchmarkPlainMessages: the wire's encoder on 100 plain messages.
func BenchmarkPlainMessages(b *testing.B) {
	items := plainItems(100)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := json.Marshal(Input(items)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHistory: the wire's encoder on a history with calls, at three
// lengths.
func BenchmarkHistory(b *testing.B) {
	for _, n := range []int{40, 100, 200} {
		b.Run(fmt.Sprintf("items=%d", n), func(b *testing.B) {
			items := historyItems(n)
			b.ReportAllocs()
			for b.Loop() {
				if _, err := json.Marshal(Input(items)); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
