// Package deferral decides how a request's deferred tools are sent, once,
// for every encoder (0040-MADR D1 to D6, A1 to A5).
//
// A request with no deferred tool is sent as before: every function here
// returns its tools as they are. With one, Of picks the form the wire and
// model take, Tools says which tools that form sends, Loaded and SearchLoads
// say what the input's ToolLoadItems have loaded.
package deferral

import (
	"regexp"
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
// as before; and Append on Responses when req forces its Search tool, since
// tool_choice can name a function but not a tool_search entry (0040-MADR D3).
func ForRequest(wire Wire, model string, req *llmprovider.Request) Form {
	if !Deferring(req.Tools) {
		return Append
	}
	form := Of(wire, model)
	if name, forced := req.ToolChoice.Tool(); forced && form == ResponsesNative && name == searchTool(req.Tools) {
		return Append
	}
	return form
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
		return withLoaded(req.Tools, nil)
	}
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
