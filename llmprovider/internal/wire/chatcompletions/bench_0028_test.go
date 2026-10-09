package chatcompletions

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0028-MADR A6: the cost of a Chat Completions body, built and marshalled
// as the transport does.

// plainHistory is n plain messages, user and assistant in turn.
func plainHistory(n int) []llmprovider.Item {
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

// toolHistory is n items in turns of a user message, an assistant message, a
// call with about 1 KiB of arguments, and its output.
func toolHistory(n int) []llmprovider.Item {
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

// BenchmarkPlainMessages: 100 plain messages, built and marshalled.
func BenchmarkPlainMessages(b *testing.B) {
	items := plainHistory(100)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := json.Marshal(Body("m", 1024, items, Opts{})); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBodyMarshal: a history with calls, at three lengths.
func BenchmarkBodyMarshal(b *testing.B) {
	for _, n := range []int{40, 100, 200} {
		b.Run(fmt.Sprintf("items=%d", n), func(b *testing.B) {
			items := toolHistory(n)
			b.ReportAllocs()
			for b.Loop() {
				if _, err := json.Marshal(Body("m", 1024, items, Opts{})); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
