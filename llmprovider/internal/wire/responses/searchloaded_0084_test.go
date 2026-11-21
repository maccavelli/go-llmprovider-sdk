package responses

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestSearchLoaded_0084 (0084-MADR D4): the names SearchLoaded gives are the
// definitions InputLoading sent inside tool_search_output items, once each:
// the tools a result loaded, and none that was only declared, loaded by a
// result other than the search's, or loaded before its search call.
func TestSearchLoaded_0084(t *testing.T) {
	tools := []llmprovider.Tool{{Name: "read_file"}, {Name: "tool_search", Search: true},
		{Name: "a", Deferred: true}, {Name: "b", Deferred: true}, {Name: "c", Deferred: true}}
	user := llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "go"}
	search := llmprovider.FunctionCallItem{CallID: "s1", Name: "tool_search", Arguments: `{}`}
	result := llmprovider.FunctionCallOutputItem{CallID: "s1", Output: "loaded"}
	for _, tc := range []struct {
		name  string
		tools []llmprovider.Tool
		input []llmprovider.Item
		want  []string
	}{
		{"one load", tools, []llmprovider.Item{user, search, result, llmprovider.ToolLoadItem{CallID: "s1", Name: "b"}}, []string{"b"}},
		{"two loads, one twice", tools, []llmprovider.Item{user, search, result, llmprovider.ToolLoadItem{CallID: "s1", Name: "b"},
			llmprovider.ToolLoadItem{CallID: "s1", Name: "a"}, llmprovider.ToolLoadItem{CallID: "s1", Name: "b"}}, []string{"b", "a"}},
		{"a load by another result", tools, []llmprovider.Item{user,
			llmprovider.FunctionCallItem{CallID: "x1", Name: "read_file"}, llmprovider.FunctionCallOutputItem{CallID: "x1", Output: "t"},
			llmprovider.ToolLoadItem{CallID: "x1", Name: "c"}}, nil},
		{"nothing loaded", tools, []llmprovider.Item{user, search, result}, nil},
		{"no Search tool", tools[:1], []llmprovider.Item{user}, nil},
		{"a load of a tool not declared", tools, []llmprovider.Item{user, search, result,
			llmprovider.ToolLoadItem{CallID: "s1", Name: "ghost"}}, nil},
	} {
		req := &llmprovider.Request{Tools: tc.tools, Input: tc.input}
		got := SearchLoaded(tc.input, req)
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: %v; want %v", tc.name, got, tc.want)
		}
		// The same names, read from the encoded input.
		raw, err := json.Marshal(InputLoading(tc.input, req))
		if err != nil {
			t.Fatal(err)
		}
		var items []struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		}
		if err := json.Unmarshal(raw, &items); err != nil {
			t.Fatal(err)
		}
		var sent []string
		for _, item := range items {
			for _, tool := range item.Tools {
				if tool.Name != "" && !slices.Contains(sent, tool.Name) {
					sent = append(sent, tool.Name)
				}
			}
		}
		if !slices.Equal(sent, tc.want) {
			t.Errorf("%s: the encoded input holds %v; want %v", tc.name, sent, tc.want)
		}
	}
}
