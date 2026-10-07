package wire

import (
	"bytes"
	"encoding/json"
	"reflect"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// JSON keys of the tool lists.
const (
	keyFunction    = "function"
	keyDescription = "description"
	keyParameters  = "parameters"
	keyToolChoice  = "tool_choice"
)

// ToolSchema is a tool's JSON schema as sent. A nil schema is an object with
// no properties: Anthropic refuses a null input_schema (0021-MADR W11). A
// typed nil, such as a nil map or a nil json.RawMessage, and a RawMessage of
// null, are nil too, since each would be sent as null (0026-MADR F27).
func ToolSchema(schema any) any {
	if noSchema(schema) {
		return map[string]any{KeyType: "object", "properties": map[string]any{}}
	}
	return schema
}

// noSchema reports whether schema would be sent as null.
func noSchema(schema any) bool {
	if raw, ok := schema.(json.RawMessage); ok {
		return bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || len(bytes.TrimSpace(raw)) == 0
	}
	if schema == nil {
		return true
	}
	switch v := reflect.ValueOf(schema); v.Kind() {
	case reflect.Map, reflect.Slice, reflect.Pointer, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

// ResponsesTools is tools as Responses function tools.
func ResponsesTools(tools []llmprovider.Tool) []map[string]any {
	list := make([]map[string]any, len(tools))
	for i, tool := range tools {
		list[i] = map[string]any{
			KeyType:        keyFunction,
			KeyName:        tool.Name,
			keyDescription: tool.Description,
			keyParameters:  ToolSchema(tool.Schema),
		}
	}
	return list
}

// AddResponsesTools adds tools and the tool_choice for choice to a Responses
// body, when there are tools. Auto sends no tool_choice.
func AddResponsesTools(body map[string]any, tools []llmprovider.Tool, choice llmprovider.ToolChoice) {
	if len(tools) == 0 {
		return
	}
	body["tools"] = ResponsesTools(tools)
	if name, forced := choice.Tool(); forced {
		body[keyToolChoice] = map[string]any{KeyType: keyFunction, KeyName: name}
	} else if choice == llmprovider.ToolChoiceRequired || choice == llmprovider.ToolChoiceNone {
		body[keyToolChoice] = string(choice)
	}
}

// MessagesTools is tools as Messages tools. Each provider sends its own
// tool_choice.
func MessagesTools(tools []llmprovider.Tool) []map[string]any {
	list := make([]map[string]any, len(tools))
	for i, tool := range tools {
		list[i] = map[string]any{KeyName: tool.Name, keyDescription: tool.Description, "input_schema": ToolSchema(tool.Schema)}
	}
	return list
}
