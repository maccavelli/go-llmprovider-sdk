// Package deferral decides how a request's deferred tools are sent, once,
// for every encoder (0040-MADR D1 to D6, A1 to A5).
//
// A request with no deferred tool is sent as before: every function here
// returns its tools as they are. With one, Of picks the form the wire and
// model take, Tools says which tools that form sends, Loaded and SearchLoads
// say what the input's ToolLoadItems have loaded.
package deferral

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/messages"
)

// Wire is the wire and route a request goes out on.
type Wire int

const (
	// Messages is Anthropic's Messages API: claude, and OpenCode's messages
	// route.
	Messages Wire = iota + 1
	// Responses is OpenAI's Responses API with an API key, and OpenCode's
	// responses route.
	Responses
	// ChatGPT is a ChatGPT session's Responses backend.
	ChatGPT
	// Grok is xAI's Responses API.
	Grok
	// Other is every other wire: Chat Completions, generateContent and
	// Interactions.
	Other
)

// Form is how a request's deferred tools are sent.
type Form int

const (
	// Append sends the deferred tools a result has loaded after the others,
	// in the order they were loaded (D6).
	Append Form = iota
	// MessagesNative sends every tool, the deferred ones with defer_loading,
	// and a load as a tool_reference beside its result (D2, A1).
	MessagesNative
	// ResponsesNative sends the search as a client tool_search, the deferred
	// tools only in its tool_search_output, and appends a load from any
	// other result (D3).
	ResponsesNative
	// Dispatch sends no deferred tool, and runs a loaded one through a fixed
	// wrapper (D5).
	Dispatch
)

// UseTool is the wrapper Dispatch sends in place of the deferred tools, as
// grok-build names it (0040-MADR D5, E20).
const UseTool = "use_tool"

// keyType is a JSON schema's type key.
const keyType = "type"

// useToolSchema is UseTool's input: the name of a tool the search found, and
// that tool's arguments, an open object (0040-MADR D5, E21).
var useToolSchema = map[string]any{
	keyType: "object",
	"properties": map[string]any{
		"tool_name":  map[string]any{keyType: "string", "description": "The name of a tool the search found."},
		"tool_input": map[string]any{keyType: "object", "description": "Its arguments, by the input schema the search gave."},
	},
	"required": []string{"tool_name", "tool_input"},
}

// DispatchTool is UseTool's definition.
func DispatchTool() llmprovider.Tool {
	return llmprovider.Tool{Name: UseTool, Schema: useToolSchema,
		Description: "Runs a tool that the search found, with the input schema the search gave."}
}

// gptVersionRE reads an OpenAI GPT id's version, such as 5.4 in gpt-5.4-mini
// or 6 in gpt-6-luna.
var gptVersionRE = regexp.MustCompile(`(?i)^gpt-(\d+)(?:\.(\d+))?(?:[-.]|$)`)

// Of is the form for a request on wire to model (D1's table). An unreadable
// model takes Append: a lost saving, never a 400.
func Of(wire Wire, model string) Form {
	switch wire {
	case Messages:
		// Claude 4.5 and later (Anthropic's tool search page lists Haiku 4.5
		// and Sonnet 4.5 first; "Opus 4.1 and earlier models don't support").
		if major, minor, ok := messages.ClaudeVersion(model); ok && (major > 4 || major == 4 && minor >= 5) {
			return MessagesNative
		}
	case Responses:
		// "only gpt-5.4 and later models support tool_search".
		if major, minor, ok := gptVersion(model); ok && (major > 5 || major == 5 && minor >= 4) {
			return ResponsesNative
		}
	case ChatGPT:
		// codex uses the client tool_search for every model the backend
		// serves (0040-MADR E12–E17, A3).
		return ResponsesNative
	case Grok:
		return Dispatch
	}
	return Append
}

// ForRequest is Of for req: Append when req defers nothing, which sends it
// as before. A forced choice must name a tool in the list, so it is Append on
// Responses when req forces its Search tool, which is a tool_search entry
// there (0040-MADR D3), and under Dispatch when req forces a deferred tool,
// which the list holds only behind UseTool (D5).
func ForRequest(wire Wire, model string, req *llmprovider.Request) Form {
	if !Deferring(req.Tools) {
		return Append
	}
	form := Of(wire, model)
	if name, forced := req.ToolChoice.Tool(); forced {
		switch {
		case form == ResponsesNative && name == searchTool(req.Tools),
			form == Dispatch && slices.ContainsFunc(req.Tools, func(t llmprovider.Tool) bool { return t.Deferred && t.Name == name }):
			return Append
		}
	}
	return form
}

// CheckDispatch refuses, when form is Dispatch, a request with a tool of its
// own named UseTool, whose name the wrapper would take (0040-MADR D5), or a
// deferred tool whose schema does not encode as JSON. Such a schema is sent
// only inside a result's text, so it would otherwise not fail the request,
// as it does where tools are encoded.
func CheckDispatch(form Form, req *llmprovider.Request) error {
	if form != Dispatch {
		return nil
	}
	for _, tool := range req.Tools {
		if tool.Name == UseTool {
			return fmt.Errorf("%w: tool %q has the name of the deferred-tool wrapper; rename it, or defer no tool",
				llmprovider.ErrInvalidRequest, UseTool)
		}
		if tool.Deferred {
			if _, err := json.Marshal(tool.Schema); err != nil {
				return fmt.Errorf("%w: tool %q: schema: %w", llmprovider.ErrInvalidRequest, tool.Name, err)
			}
		}
	}
	return nil
}

// Dispatched is the deferred tools' names: those a UseTool call may run
// (0040-MADR D5).
func Dispatched(tools []llmprovider.Tool) map[string]bool {
	out := map[string]bool{}
	for _, tool := range tools {
		if tool.Deferred {
			out[tool.Name] = true
		}
	}
	return out
}

// gptVersion reads a GPT id's major and minor version by gptVersionRE; a
// missing minor is 0. It reports false for an id it cannot read.
func gptVersion(model string) (major, minor int, ok bool) {
	m := gptVersionRE.FindStringSubmatch(strings.TrimSpace(model))
	if m == nil {
		return 0, 0, false
	}
	major, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, 0, false
	}
	if m[2] != "" {
		if minor, err = strconv.Atoi(m[2]); err != nil {
			return 0, 0, false
		}
	}
	return major, minor, true
}

// Deferring reports whether any of tools is deferred.
func Deferring(tools []llmprovider.Tool) bool {
	for _, tool := range tools {
		if tool.Deferred {
			return true
		}
	}
	return false
}

// Loaded is every tool the input's ToolLoadItems load, in the order each was
// first loaded, once each.
func Loaded(input []llmprovider.Item) []string {
	var out []string
	seen := map[string]bool{}
	for _, item := range input {
		if load, ok := item.(llmprovider.ToolLoadItem); ok && !seen[load.Name] {
			seen[load.Name] = true
			out = append(out, load.Name)
		}
	}
	return out
}

// SearchName is the Search tool's name among tools, or "": what a Responses
// decoder names a client tool_search_call (0040-MADR D3).
func SearchName(tools []llmprovider.Tool) string { return searchTool(tools) }

// searchTool is the request's Search tool's name, or "".
func searchTool(tools []llmprovider.Tool) string {
	for _, tool := range tools {
		if tool.Search {
			return tool.Name
		}
	}
	return ""
}

// searchCalls is the call ids the input's calls to the search tool carry.
func searchCalls(req *llmprovider.Request) map[string]bool {
	search, out := searchTool(req.Tools), map[string]bool{}
	if search == "" {
		return out
	}
	for _, item := range req.Input {
		if call, ok := item.(llmprovider.FunctionCallItem); ok && call.Name == search {
			out[call.CallID] = true
		}
	}
	return out
}

// SearchLoads is, by call id, the tools each result of the search tool
// loads, in order: what a Responses tool_search_output carries (D3).
func SearchLoads(req *llmprovider.Request) map[string][]string {
	calls, out := searchCalls(req), map[string][]string{}
	for _, item := range req.Input {
		if load, ok := item.(llmprovider.ToolLoadItem); ok && calls[load.CallID] {
			out[load.CallID] = append(out[load.CallID], load.Name)
		}
	}
	return out
}

// Tools is the tools form sends for req, in order. A request with no deferred
// tool gets its own slice back, unchanged.
func Tools(req *llmprovider.Request, form Form) []llmprovider.Tool {
	if !Deferring(req.Tools) {
		return req.Tools
	}
	switch form {
	case MessagesNative:
		return req.Tools
	case Append:
		if req.PreviousResponseID != "" {
			// The earlier loads are on the server, not in Input: every
			// deferred tool is sent, so none goes missing (D6).
			return req.Tools
		}
		return withLoaded(req.Tools, Loaded(req.Input))
	case ResponsesNative:
		// A search's loads ride in its tool_search_output; any other
		// result's are appended (D3, Q4).
		calls := searchCalls(req)
		var other []string
		seen := map[string]bool{}
		for _, item := range req.Input {
			if load, ok := item.(llmprovider.ToolLoadItem); ok && !calls[load.CallID] && !seen[load.Name] {
				seen[load.Name] = true
				other = append(other, load.Name)
			}
		}
		return withLoaded(req.Tools, other)
	default: // Dispatch
		// The list never changes, so the cache holds across loads (D5).
		return append(withLoaded(req.Tools, nil), DispatchTool())
	}
}

// Loads is, by call id, the tools each result's ToolLoadItems load, in
// order, once each: what a Dispatch result carries (0040-MADR D5, A5).
func Loads(input []llmprovider.Item) map[string][]string {
	out := map[string][]string{}
	for _, item := range input {
		if load, ok := item.(llmprovider.ToolLoadItem); ok && !slices.Contains(out[load.CallID], load.Name) {
			out[load.CallID] = append(out[load.CallID], load.Name)
		}
	}
	return out
}

// Warning is the WarningDeferralEmulated req earns sent in form to model, if
// any (0040-MADR D7): when it defers a tool and form appends or dispatches
// them, or form is ResponsesNative and a result other than the search's
// loaded a tool, which is appended.
func Warning(form Form, req *llmprovider.Request, model string) (llmprovider.Warning, bool) {
	if !Deferring(req.Tools) {
		return llmprovider.Warning{}, false
	}
	warn := func(detail string) (llmprovider.Warning, bool) {
		return llmprovider.Warning{Kind: llmprovider.WarningDeferralEmulated, Model: model, Detail: detail}, true
	}
	switch form {
	case Append:
		if req.PreviousResponseID != "" {
			return warn("Deferred tools have no native form here, and the earlier loads are held by the service, " +
				"so every deferred tool was sent as an ordinary tool.")
		}
		return warn("Deferred tools have no native form here, so each loaded tool was appended to the tools; " +
			"the prompt cache breaks once at each load.")
	case Dispatch:
		return warn("Deferred tools were sent through the use_tool wrapper, with each loaded tool's schema in its result.")
	case ResponsesNative:
		calls := searchCalls(req)
		var other []string
		for _, item := range req.Input {
			if load, ok := item.(llmprovider.ToolLoadItem); ok && !calls[load.CallID] {
				other = append(other, load.Name)
			}
		}
		if len(other) > 0 {
			return warn("A tool loaded by a result other than the search's (" + strings.Join(other, ", ") +
				") has no native form on this wire, so it was appended to the tools.")
		}
	}
	return llmprovider.Warning{}, false
}

// Warnings is Warning as what a wire.Call carries: the warning, or nil.
func Warnings(form Form, req *llmprovider.Request, model string) []llmprovider.Warning {
	if w, ok := Warning(form, req, model); ok {
		return []llmprovider.Warning{w}
	}
	return nil
}

// withLoaded is the tools that are not deferred, in order, then the deferred
// ones named in loaded, in loaded's order.
func withLoaded(tools []llmprovider.Tool, loaded []string) []llmprovider.Tool {
	byName := map[string]llmprovider.Tool{}
	out := make([]llmprovider.Tool, 0, len(tools))
	for _, tool := range tools {
		if tool.Deferred {
			byName[tool.Name] = tool
			continue
		}
		out = append(out, tool)
	}
	for _, name := range loaded {
		if tool, ok := byName[name]; ok {
			out = append(out, tool)
		}
	}
	return out
}
