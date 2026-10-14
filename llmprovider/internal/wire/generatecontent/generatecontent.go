// Package generatecontent is Gemini's generateContent wire, which OpenCode's
// google route speaks (0015-MADR D2), with its thinking request shape. The
// gemini provider speaks the Interactions API instead, in providers/gemini
// (MADR 0014).
package generatecontent

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

const (
	// roleModel is Gemini's assistant role.
	roleModel = "model"
	// skipThoughtSignature is Gemini's documented placeholder for a replayed
	// call it did not issue (accepted live, 2026-09-27).
	skipThoughtSignature = "skip_thought_signature_validator"

	keyThinkingBudget = "thinkingBudget"
)

// dynamicThinkingBudget (-1) lets the model size its own thinking budget.
const dynamicThinkingBudget = -1

// Content is one generateContent content, and System a systemInstruction.
// Part is every part in one struct: text, a function call with its thought
// signature, and a function response; a field a part does not have is nil,
// and omitted. The fields are in the order json.Marshal wrote the maps they
// replace, so the JSON is the same byte for byte (0028-MADR D-A6, amendments
// 2026-10-09).
type Content struct {
	Parts []Part `json:"parts"`
	Role  string `json:"role"`
}

// System is a systemInstruction.
type System struct {
	Parts []Part `json:"parts"`
}

// Part is one part of a content.
type Part struct {
	FunctionCall     *FunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
	Text             *string           `json:"text,omitempty"`
	ThoughtSignature *string           `json:"thoughtSignature,omitempty"`
}

// FunctionCall is a part's functionCall.
type FunctionCall struct {
	Args any    `json:"args"`
	Name string `json:"name"`
}

// FunctionResponse is a part's functionResponse.
type FunctionResponse struct {
	Name     string         `json:"name"`
	Response FunctionOutput `json:"response"`
}

// FunctionOutput is a functionResponse's response: the output, or, for a
// failed call, the error, as Gemini's convention has it (0032-MADR D3).
// Exactly one is set.
type FunctionOutput struct {
	Error  *string `json:"error,omitempty"`
	Output *string `json:"output,omitempty"`
}

// SystemInstruction is generateContent's systemInstruction for the system
// items, or nil when there are none (MADR 0014 §3).
func SystemInstruction(items []llmprovider.Item) *System {
	system := wire.SystemPrompt(items)
	if system == "" {
		return nil
	}
	return &System{Parts: []Part{{Text: &system}}}
}

// Contents is generateContent's contents, for OpenCode's google route. The
// strings the parts carry are kept in one arena, the calls and responses in
// one array each, and every content's parts in one backing array: only the
// last content ever takes a part, so each content's parts are contiguous
// (0028-PLAN D16).
func Contents(items []llmprovider.Item) []Content {
	// Gemini pairs a functionResponse with its functionCall by name, so a
	// result takes the name of the call it answers (MADR 0012 §2).
	names := map[string]string{}
	for _, item := range items {
		if call, ok := item.(llmprovider.FunctionCallItem); ok {
			names[call.CallID] = call.Name
		}
	}
	var (
		contents  []Content
		arena     = wire.NewArena(len(items))
		parts     = make([]Part, 0, len(items))
		calls     = make([]FunctionCall, 0, len(items))
		responses = make([]FunctionResponse, 0, len(items))
		start     int // where the last content's parts begin in parts
	)
	lastIsResponses := false
	appendPart := func(role string, part Part, merge bool) {
		if n := len(contents); merge && n > 0 && contents[n-1].Role == role {
			parts = append(parts, part)
			contents[n-1].Parts = parts[start:len(parts):len(parts)]
			return
		}
		start = len(parts)
		parts = append(parts, part)
		contents = append(contents, Content{Role: role, Parts: parts[start:len(parts):len(parts)]})
	}
	for _, item := range items {
		switch v := item.(type) {
		case llmprovider.MessageItem:
			if v.Role == wire.RoleSystem {
				continue // systemInstruction; see SystemInstruction
			}
			role := string(v.Role)
			if role == "" || role == wire.RoleUser {
				role = wire.RoleUser
			} else {
				role = roleModel
			}
			appendPart(role, Part{Text: arena.Keep(v.Text)}, false)
			lastIsResponses = false
		case llmprovider.FunctionCallItem:
			// Gemini refuses a replayed call without its thoughtSignature. A call
			// Gemini did not issue (synthetic, or from another provider) carries
			// the documented skip value instead.
			signature := v.Signature
			if signature == "" {
				signature = skipThoughtSignature
			}
			calls = append(calls, FunctionCall{Name: v.Name, Args: wire.ToolArguments(v.Arguments)})
			appendPart(roleModel, Part{FunctionCall: &calls[len(calls)-1], ThoughtSignature: arena.Keep(signature)}, true)
			lastIsResponses = false
		case llmprovider.FunctionCallOutputItem:
			name := names[v.CallID]
			if name == "" {
				name = callName(v.CallID)
			}
			// A turn's results share one user turn, as its calls share one model turn.
			output := FunctionOutput{Output: arena.Keep(v.Output)}
			if v.IsError {
				output = FunctionOutput{Error: arena.Keep(v.Output)}
			}
			responses = append(responses, FunctionResponse{Name: name, Response: output})
			appendPart(wire.RoleUser, Part{FunctionResponse: &responses[len(responses)-1]}, lastIsResponses)
			lastIsResponses = true
		}
	}
	return contents
}

// Decode decodes a generateContent response.
func Decode(body io.Reader) (*llmprovider.Response, error) {
	var raw reply
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return nil, err
	}
	return raw.response()
}

// reply is a GenerateContentResponse, whole, or assembled from its stream.
type reply struct {
	ModelVersion string      `json:"modelVersion"`
	Candidates   []candidate `json:"candidates"`
	Usage        *usage      `json:"usageMetadata"`
	// Error is a failure inside a stream (0031-MADR E3).
	Error          *replyError `json:"error"`
	PromptFeedback struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
}

// candidate is one of a reply's candidates.
type candidate struct {
	FinishReason string `json:"finishReason"`
	Content      struct {
		Parts []replyPart `json:"parts"`
	} `json:"content"`
}

// replyPart is one part of a candidate's content.
type replyPart struct {
	Text         string `json:"text"`
	Thought      bool   `json:"thought"`
	FunctionCall *struct {
		ID   string          `json:"id"`
		Name string          `json:"name"`
		Args json.RawMessage `json:"args"`
	} `json:"functionCall"`
	ThoughtSignature string `json:"thoughtSignature"`
}

// replyError is Google's error body.
type replyError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// response is raw as a Response, or the error its candidate makes it.
func (raw reply) response() (*llmprovider.Response, error) {
	// A blocked prompt has no candidate; its block reason says why
	// (0026-MADR F25).
	if len(raw.Candidates) == 0 {
		return nil, wire.EmptyAnswer("generatecontent", llmprovider.FinishReason(raw.PromptFeedback.BlockReason))
	}
	hasCall := false
	for _, part := range raw.Candidates[0].Content.Parts {
		hasCall = hasCall || part.FunctionCall != nil
	}
	finish := wire.Finish(raw.Candidates[0].FinishReason, finishReasons, hasCall)
	if len(raw.Candidates[0].Content.Parts) == 0 {
		return nil, wire.EmptyAnswer("generatecontent", finish)
	}
	if finish == llmprovider.FinishLength && hasCall {
		// A call cut by the token limit has unusable arguments (0020-MADR F3).
		return nil, &llmprovider.APIError{Kind: llmprovider.ErrIncomplete, Reason: string(llmprovider.FinishLength)}
	}

	result := &llmprovider.Response{Model: raw.ModelVersion, FinishReason: finish}
	if raw.Usage != nil {
		result.Usage = raw.Usage.counts()
	}
	calls := 0
	for _, part := range raw.Candidates[0].Content.Parts {
		// A thought summary is flagged thought: true, with its text in text.
		if part.Thought {
			if part.Text != "" {
				result.Output = append(result.Output, llmprovider.ReasoningItem{Text: part.Text, Format: wire.FormatGenerateContent})
			}
			continue
		}
		if part.Text != "" {
			result.Output = append(result.Output, llmprovider.MessageItem{Role: wire.RoleAssistant, Text: part.Text})
		}
		if part.FunctionCall != nil {
			// A tool without parameters sends no args: "{}".
			args, err := wire.CompactArguments(part.FunctionCall.Args)
			if err != nil {
				return nil, fmt.Errorf("generatecontent: functionCall args: %w", err)
			}
			result.Output = append(result.Output, llmprovider.FunctionCallItem{
				CallID:    callID(part.FunctionCall.ID, part.FunctionCall.Name, calls),
				Name:      part.FunctionCall.Name,
				Arguments: args,
				Signature: part.ThoughtSignature,
			})
			calls++
		}
	}
	// An answer cut while it was still thinking has nothing usable
	// (0026-MADR F7); a cut text answer keeps its text.
	if len(result.Output) == 0 || finish == llmprovider.FinishLength && result.OutputText() == "" {
		return nil, wire.EmptyAnswer("generatecontent", finish)
	}
	return result, nil
}

// Events reads a generateContent event stream (:streamGenerateContent?alt=sse)
// and yields its events as they arrive (0031-MADR E3): each text part as
// EventTextDelta and, at the chunk carrying the finishReason, the items Decode
// returns for the same answer as one reply, in its order, then EventDone,
// carrying that Response. A thought part is assembled into its item, not
// yielded as a delta: none was seen streaming (0031-PLAN-native-streaming-on-
// the-other-wires, Phase 1). A function call arrives whole, as an item.
// Consecutive text parts, and consecutive thought parts, are one part, as a
// whole reply sends them. Google's error body in the stream is classified by
// the status its code names, and a blocked prompt is Decode's error. It
// returns nil once EventDone is yielded or yield returns false, and otherwise
// the error that ended the stream; what it yielded before an error is not a
// response (0031-MADR D4).
func Events(provider string, body io.Reader, yield func(llmprovider.Event) bool) error {
	raw := reply{Candidates: []candidate{{}}}
	return wire.ReadEvents(provider, body, "a finishReason", func(payload []byte) (bool, error) {
		var chunk reply
		if err := json.Unmarshal(payload, &chunk); err != nil {
			return false, fmt.Errorf("%s: decode stream chunk: %w", provider, err)
		}
		if e := chunk.Error; e != nil {
			return false, llmprovider.ClassifyStreamFailure(provider, strconv.Itoa(e.Code), e.Status, e.Message)
		}
		if chunk.ModelVersion != "" {
			raw.ModelVersion = chunk.ModelVersion
		}
		if chunk.Usage != nil {
			raw.Usage = chunk.Usage
		}
		if len(chunk.Candidates) == 0 {
			if chunk.PromptFeedback.BlockReason == "" {
				return false, nil
			}
			chunk.Candidates = nil
			_, err := chunk.response()
			return false, err
		}
		next := chunk.Candidates[0]
		for _, part := range next.Content.Parts {
			raw.Candidates[0].Content.Parts = appendPart(raw.Candidates[0].Content.Parts, part)
			if part.Text != "" && !part.Thought && part.FunctionCall == nil &&
				!yield(llmprovider.Event{Type: llmprovider.EventTextDelta, Text: part.Text}) {
				return true, nil
			}
		}
		if next.FinishReason == "" {
			return false, nil
		}
		raw.Candidates[0].FinishReason = next.FinishReason
		resp, err := raw.response()
		if err != nil {
			return false, err
		}
		for _, item := range resp.Output {
			if !yield(llmprovider.Event{Type: llmprovider.EventItem, Item: item}) {
				return true, nil
			}
		}
		yield(llmprovider.Event{Type: llmprovider.EventDone, Response: resp})
		return true, nil
	})
}

// appendPart adds one streamed part to parts: a text part joins a text part
// before it, and a thought part a thought part, as a whole reply sends one of
// each; a signature is kept wherever it arrives.
func appendPart(parts []replyPart, part replyPart) []replyPart {
	if n := len(parts); n > 0 && part.FunctionCall == nil && parts[n-1].FunctionCall == nil &&
		parts[n-1].Thought == part.Thought && parts[n-1].Text != "" {
		parts[n-1].Text += part.Text
		if part.ThoughtSignature != "" {
			parts[n-1].ThoughtSignature = part.ThoughtSignature
		}
		return parts
	}
	return append(parts, part)
}

// callID is a call's own id, or "name#index" for a service that sends none,
// so that two calls to one function in a reply have distinct ids, as OpenCode
// gives them (0021-MADR W10).
func callID(id, name string, index int) string {
	if id != "" {
		return id
	}
	return name + "#" + strconv.Itoa(index)
}

// callName is the function a call id names: the id with any "#index" suffix
// that callID added removed.
func callName(id string) string {
	if i := strings.LastIndexByte(id, '#'); i > 0 {
		if _, err := strconv.Atoi(id[i+1:]); err == nil {
			return id[:i]
		}
	}
	return id
}

// finishReasons maps generateContent's finishReason to FinishReason
// (0020-MADR F3). A reason not listed is kept as sent (0021-MADR W3), as
// are MALFORMED_FUNCTION_CALL and UNEXPECTED_TOOL_CALL, which say a tool
// call failed, not that the answer is complete (0026-MADR F26).
var finishReasons = map[string]llmprovider.FinishReason{
	"STOP":                      llmprovider.FinishStop,
	"MAX_TOKENS":                llmprovider.FinishLength,
	"SAFETY":                    llmprovider.FinishContentFilter,
	"RECITATION":                llmprovider.FinishContentFilter,
	"BLOCKLIST":                 llmprovider.FinishContentFilter,
	"PROHIBITED_CONTENT":        llmprovider.FinishContentFilter,
	"SPII":                      llmprovider.FinishContentFilter,
	"IMAGE_SAFETY":              llmprovider.FinishContentFilter,
	"FINISH_REASON_UNSPECIFIED": "",
}

// geminiLegacyRE matches Gemini 1.x and 2.x ids, which take thinkingBudget but
// not thinkingLevel (HTTP 400 on gemini-2.5-flash, measured 2026-09-27). It is
// OpenCode's GEMINI_LEGACY_RE.
var geminiLegacyRE = regexp.MustCompile(`(?i)gemini-(?:(?:flash|pro)-)?[12](?:[.-]|$)`)

// ThinkingConfig returns a Gemini thinkingConfig. A configured budget wins.
// Otherwise "low" is thinkingLevel "low" on Gemini 3 and later and a
// wire.LowEffortThinkingBudget budget on 1.x and 2.x, and any other effort
// keeps the dynamic budget (MADR 0013 Q1, B9).
func ThinkingConfig(model, effort string, budget int) map[string]any {
	// Thought summaries come back only when asked for, as OpenCode's client
	// asks (transform.ts:1280-1288, MADR 0014 §3).
	low := string(llmprovider.EffortLow)
	cfg := map[string]any{"includeThoughts": true}
	switch {
	case budget > 0:
		cfg[keyThinkingBudget] = budget
	case effort != low:
		cfg[keyThinkingBudget] = dynamicThinkingBudget
	case geminiLegacyRE.MatchString(model):
		cfg[keyThinkingBudget] = wire.LowEffortThinkingBudget
	default:
		cfg["thinkingLevel"] = low
	}
	return cfg
}

// Thinking is what a model's facts say of its Gemini thinking (0041-MADR D2,
// R4–R5), resolved by the provider so that this package reads no facts.
type Thinking struct {
	// Known reports that the facts answer. Without it geminiLegacyRE decides.
	Known bool
	// Budget is a thinking budget without efforts: thinkingBudget.
	Budget bool
	// Levels are the model's efforts: thinkingLevel, sent only when the
	// effort is one of them.
	Levels []string
}

// FactsThinking is the Thinking a model's reasoning facts give: efforts are
// levels, and a budget without efforts is a budget. With neither the facts do
// not answer.
func FactsThinking(efforts []string, budget bool) Thinking {
	switch {
	case len(efforts) > 0:
		return Thinking{Known: true, Levels: efforts}
	case budget:
		return Thinking{Known: true, Budget: true}
	}
	return Thinking{}
}

// ThinkingConfigFor is ThinkingConfig with the model's facts (0041-MADR D2): a
// configured budget still wins; a model with levels gets thinkingLevel, only
// for an effort among them; one with a budget gets ThinkingConfig's budgets.
func ThinkingConfigFor(model, effort string, budget int, facts Thinking) map[string]any {
	if !facts.Known || budget > 0 {
		return ThinkingConfig(model, effort, budget)
	}
	cfg := map[string]any{"includeThoughts": true}
	switch {
	case len(facts.Levels) > 0:
		if effort != "" && slices.Contains(facts.Levels, effort) {
			cfg["thinkingLevel"] = effort
		}
	case effort == string(llmprovider.EffortLow):
		cfg[keyThinkingBudget] = wire.LowEffortThinkingBudget
	default:
		cfg[keyThinkingBudget] = dynamicThinkingBudget
	}
	return cfg
}

// usage is generateContent's usageMetadata. promptTokenCount holds the cached
// tokens; candidatesTokenCount leaves out the thoughts, so the output count
// adds them.
type usage struct {
	Prompt     int `json:"promptTokenCount"`
	Candidates int `json:"candidatesTokenCount"`
	Thoughts   int `json:"thoughtsTokenCount"`
	Cached     int `json:"cachedContentTokenCount"`
}

func (u usage) counts() llmprovider.Usage {
	return llmprovider.Usage{InputTokens: u.Prompt, OutputTokens: u.Candidates + u.Thoughts,
		ReasoningTokens: u.Thoughts, CachedTokens: u.Cached}
}
