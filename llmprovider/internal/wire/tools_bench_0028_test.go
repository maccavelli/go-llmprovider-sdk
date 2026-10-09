package wire

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0028-PLAN Phase 9b, step 6: MessagesTools and ResponsesTools against typed
// prototypes of the same JSON. Measured only; they move only on the bar the
// wires met (3 times faster, a tenth of the allocations).

// benchTools is ten tools with a small object schema each.
func benchTools() []llmprovider.Tool {
	tools := make([]llmprovider.Tool, 10)
	for i := range tools {
		tools[i] = llmprovider.Tool{Name: fmt.Sprintf("tool_%d", i), Description: "Does a thing with a city.",
			Schema: map[string]any{"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}}}
	}
	return tools
}

type typedMessagesTool struct {
	Description string `json:"description"`
	InputSchema any    `json:"input_schema"`
	Name        string `json:"name"`
}

type typedResponsesTool struct {
	Description string `json:"description"`
	Name        string `json:"name"`
	Parameters  any    `json:"parameters"`
	Type        string `json:"type"`
}

func typedMessagesTools(tools []llmprovider.Tool) []typedMessagesTool {
	list := make([]typedMessagesTool, len(tools))
	for i, tool := range tools {
		list[i] = typedMessagesTool{Name: tool.Name, Description: tool.Description, InputSchema: ToolSchema(tool.Schema)}
	}
	return list
}

func typedResponsesTools(tools []llmprovider.Tool) []typedResponsesTool {
	list := make([]typedResponsesTool, len(tools))
	for i, tool := range tools {
		list[i] = typedResponsesTool{Type: keyFunction, Name: tool.Name, Description: tool.Description, Parameters: ToolSchema(tool.Schema)}
	}
	return list
}

// benchEncode benchmarks json.Marshal of encode(tools), after checking it
// writes want's JSON.
func benchEncode[T any](b *testing.B, encode func([]llmprovider.Tool) T, want []byte) {
	tools := benchTools()
	if got, _ := json.Marshal(encode(tools)); string(got) != string(want) {
		b.Fatalf("JSON differs:\n%s\n%s", got, want)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := json.Marshal(encode(tools)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMessagesToolsMaps(b *testing.B) {
	want, _ := json.Marshal(MessagesTools(benchTools()))
	benchEncode(b, MessagesTools, want)
}

func BenchmarkMessagesToolsTyped(b *testing.B) {
	want, _ := json.Marshal(MessagesTools(benchTools()))
	benchEncode(b, typedMessagesTools, want)
}

func BenchmarkResponsesToolsMaps(b *testing.B) {
	want, _ := json.Marshal(ResponsesTools(benchTools()))
	benchEncode(b, ResponsesTools, want)
}

func BenchmarkResponsesToolsTyped(b *testing.B) {
	want, _ := json.Marshal(ResponsesTools(benchTools()))
	benchEncode(b, typedResponsesTools, want)
}
