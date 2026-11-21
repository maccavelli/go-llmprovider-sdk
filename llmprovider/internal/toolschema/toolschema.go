// Package toolschema reads a service's refusal of a tool's schema from the
// error it replied with (0084-MADR D3). It is the one place the four recorded
// forms' markers are written: llmprovider's classification asks it whether a
// reply is a refusal, and the wire's naming helper asks it which tools the
// reply names, so both read the same markers. They do not read the same
// input: the classification reads every type of the reply and its message as
// the service sent it, while the helper reads only the error's Code, which is
// the first type, and its Message, which is redacted and bounded to 512 bytes.
// A marker the helper cannot see therefore costs a name, never a wrong one:
// the kind is set, and Tool stays empty, which is the safe direction.
//
// A form is read from the reply's types, its codes and error types, and its
// message, never from the service it came from: the caller decides, per
// service, which forms it accepts (D3's table, and the owner's ruling on N3).
// Each form was recorded from one service and is documented by none, so a
// service that rewords its refusal is read as no form, and the reply keeps
// the kind it had (D3, "When a provider rewords a refusal").
package toolschema

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Form is one service's way of refusing a tool's schema.
type Form int

const (
	// None is a reply that is no refusal of a tool's schema in a form this
	// package reads.
	None Form = iota
	// Grok is xAI's: code invalid-argument, and the message holds the tag
	// [invalid_client_tool_schema], then the tool's name, then a colon.
	Grok
	// OpenAI is OpenAI's, and OpenCode Zen's Responses route's: code
	// invalid_function_parameters, and the message holds function '<name>'.
	OpenAI
	// Anthropic is Anthropic's: error type invalid_request_error, and a message
	// that begins tools.N. and holds input_schema, N the tool's index in the
	// list the request sent.
	Anthropic
	// Google is OpenCode Zen's Google route's: status INVALID_ARGUMENT, and the
	// message holds the tool's name in backticks before the phrase
	// functionDeclaration parameters schema.
	Google
)

// The markers each form is read by, as recorded in 0062-PLAN Phase 1 and
// 0084-PLAN Phase 1.
const (
	grokCode      = "invalid-argument"
	grokTag       = "[invalid_client_tool_schema]"
	openAICode    = "invalid_function_parameters"
	openAIFunc    = "function '"
	anthropicType = "invalid_request_error"
	anthropicKey  = "input_schema"
	googleStatus  = "INVALID_ARGUMENT"
	googlePhrase  = "functionDeclaration parameters schema"
)

// anthropicPath is the start of Anthropic's path to a refused field, such as
// "tools.1.custom.input_schema", and index its N.
var (
	anthropicStart = regexp.MustCompile(`^tools\.\d+\.`)
	anthropicPath  = regexp.MustCompile(`\btools\.(\d+)\.`)
)

// Read is the form of refusal a reply is, or None. types are the reply's
// codes and error types, as the classification holds them; message is its
// text, as the service sent it or as the error keeps it.
func Read(types []string, message string) Form {
	has := func(t string) bool { return slices.Contains(types, t) }
	switch {
	case has(openAICode):
		return OpenAI
	case has(grokCode) && strings.Contains(message, grokTag):
		return Grok
	case has(googleStatus) && strings.Contains(message, googlePhrase):
		return Google
	case has(anthropicType) && anthropicStart.MatchString(message) && strings.Contains(message, anthropicKey):
		return Anthropic
	}
	return None
}

// Refs are the tools a refusal's message names.
type Refs struct {
	// Names are the names given, in the order given. An empty name stands for
	// a place where the form puts a name and the message has none, or one cut
	// short, so it names no tool.
	Names []string
	// Indexes are the positions given, for the form that names by position.
	Indexes []int
}

// References are the tools message names, in form's way, each occurrence
// once. It reads text only: nothing in the result is known to be a tool the
// request sent.
func References(form Form, message string) Refs {
	var refs Refs
	switch form {
	case Grok:
		for rest := message; ; {
			_, after, found := strings.Cut(rest, grokTag)
			if !found {
				break
			}
			name, _, colon := strings.Cut(strings.TrimLeft(after, " "), ":")
			if !colon {
				name = ""
			}
			refs.Names = append(refs.Names, strings.TrimSpace(name))
			rest = after
		}
	case OpenAI:
		for rest := message; ; {
			_, after, found := strings.Cut(rest, openAIFunc)
			if !found {
				break
			}
			name, _, quote := strings.Cut(after, "'")
			if !quote {
				name = ""
			}
			refs.Names = append(refs.Names, name)
			rest = after
		}
	case Google:
		const closing = "` " + googlePhrase
		for offset := 0; ; {
			at := strings.Index(message[offset:], closing)
			if at < 0 {
				break
			}
			at += offset
			name := ""
			if open := strings.LastIndex(message[:at], "`"); open >= 0 {
				name = message[open+1 : at]
			}
			refs.Names = append(refs.Names, name)
			offset = at + len(closing)
		}
	case Anthropic:
		for _, m := range anthropicPath.FindAllStringSubmatch(message, -1) {
			if n, err := strconv.Atoi(m[1]); err == nil && !slices.Contains(refs.Indexes, n) {
				refs.Indexes = append(refs.Indexes, n)
			}
		}
	}
	return refs
}
