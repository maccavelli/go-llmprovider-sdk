package messages

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"iter"
	"slices"
	"strconv"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// offered reports whether tools holds one named name.
func offered(tools []llmprovider.Tool, name string) bool {
	return slices.ContainsFunc(tools, func(t llmprovider.Tool) bool { return t.Name == name })
}

// closedSchema is a copy of schema in which every object that does not say
// otherwise is closed with additionalProperties false, as structured outputs
// requires (0034-MADR Q9; 0034-PLAN S2). An object is a schema whose type is
// or holds "object", or that has properties. schema itself is not changed,
// and is returned as given when it is not JSON.
func closedSchema(schema any) any {
	raw, err := json.Marshal(schema)
	if err != nil {
		return schema
	}
	var copied any
	if json.Unmarshal(raw, &copied) != nil {
		return schema
	}
	closeObjects(copied)
	return copied
}

// Keywords whose value is a subschema, a list of them, or a map of them.
var (
	subschemaKeys     = []string{"items", "additionalProperties", "not", "contains", "if", "then", "else", "propertyNames"}
	subschemaListKeys = []string{"anyOf", "oneOf", "allOf", "prefixItems"}
	subschemaMapKeys  = []string{"properties", "patternProperties", "$defs", "definitions", "dependentSchemas"}
)

// closeObjects closes, in place, schema and each subschema it holds that is
// an open object.
func closeObjects(schema any) {
	m, ok := schema.(map[string]any)
	if !ok {
		return
	}
	if _, set := m["additionalProperties"]; !set && isObject(m) {
		m["additionalProperties"] = false
	}
	for _, k := range subschemaKeys {
		closeObjects(m[k])
	}
	for _, k := range subschemaListKeys {
		if list, ok := m[k].([]any); ok {
			for _, s := range list {
				closeObjects(s)
			}
		}
	}
	for _, k := range subschemaMapKeys {
		if defs, ok := m[k].(map[string]any); ok {
			for _, s := range defs {
				closeObjects(s)
			}
		}
	}
}

// isObject reports whether schema describes an object.
func isObject(schema map[string]any) bool {
	if _, ok := schema["properties"]; ok {
		return true
	}
	switch t := schema["type"].(type) {
	case string:
		return t == "object"
	case []any:
		return slices.Contains(t, any("object"))
	}
	return false
}

// structuredName is the tool req's forced choice names, when that choice was
// sent as structured outputs: a named tool req offers, to a model that refuses
// forcing (0034-MADR D8). Asked after the request is sent, it sees a refusal
// learned by its resend.
func structuredName(req *llmprovider.Request, f Forcing) (string, bool) {
	name, named := req.ToolChoice.Tool()
	return name, named && f.refuses() && offered(req.Tools, name)
}

// structuredItem is the call text answers for the tool name: text compacted,
// when it is one JSON object, with a call id derived from the call and the
// request, so that a stream's item and its done Response agree, and a later
// turn's call differs.
func structuredItem(req *llmprovider.Request, name, text string) (llmprovider.FunctionCallItem, bool) {
	trimmed := strings.TrimSpace(text)
	var compact bytes.Buffer
	if !strings.HasPrefix(trimmed, "{") || json.Compact(&compact, []byte(trimmed)) != nil {
		return llmprovider.FunctionCallItem{}, false
	}
	sum := sha256.Sum256([]byte(name + "\x00" + compact.String() + "\x00" + strconv.Itoa(len(req.Input))))
	return llmprovider.FunctionCallItem{CallID: "toolu_" + hex.EncodeToString(sum[:12]), Name: name, Arguments: compact.String()}, true
}

// StructuredCall turns resp, the answer to req sent as structured outputs,
// into the call the forced choice asked for: its assistant text, one JSON
// object, becomes a FunctionCallItem for the tool, in place of the text, and
// the answer finishes tool_calls (0034-MADR D8). Any other answer, and an
// answer to a request not sent so, is left as it came.
func StructuredCall(req *llmprovider.Request, f Forcing, resp *llmprovider.Response) {
	name, ok := structuredName(req, f)
	if !ok || resp == nil {
		return
	}
	first, text := -1, ""
	for i, item := range resp.Output {
		if m, isMessage := item.(llmprovider.MessageItem); isMessage && m.Role == llmprovider.RoleAssistant {
			if first < 0 {
				first = i
			}
			text += m.Text
		}
	}
	call, ok := structuredItem(req, name, text)
	if first < 0 || !ok {
		return
	}
	out := make([]llmprovider.Item, 0, len(resp.Output))
	for i, item := range resp.Output {
		if m, isMessage := item.(llmprovider.MessageItem); isMessage && m.Role == llmprovider.RoleAssistant {
			if i == first {
				out = append(out, call)
			}
			continue
		}
		out = append(out, item)
	}
	resp.Output, resp.FinishReason = out, llmprovider.FinishToolCalls
}

// StructuredStream is seq, the stream of req, with a structured-outputs answer
// turned into the call: its text deltas are not yielded, its text item is
// yielded as the FunctionCallItem, and its done Response holds the call
// (0034-MADR D8). Whether the request was sent so is asked at the first event,
// after any resend.
func StructuredStream(req *llmprovider.Request, f Forcing, seq iter.Seq2[llmprovider.Event, error]) iter.Seq2[llmprovider.Event, error] {
	return func(yield func(llmprovider.Event, error) bool) {
		decided, structured, name := false, false, ""
		for ev, err := range seq {
			if !decided {
				decided = true
				name, structured = structuredName(req, f)
			}
			if structured && err == nil {
				switch ev.Type {
				case llmprovider.EventTextDelta:
					continue
				case llmprovider.EventItem:
					if m, ok := ev.Item.(llmprovider.MessageItem); ok && m.Role == llmprovider.RoleAssistant {
						if call, ok := structuredItem(req, name, m.Text); ok {
							ev.Item = call
						}
					}
				case llmprovider.EventDone:
					StructuredCall(req, f, ev.Response)
				}
			}
			if !yield(ev, err) {
				return
			}
		}
	}
}
