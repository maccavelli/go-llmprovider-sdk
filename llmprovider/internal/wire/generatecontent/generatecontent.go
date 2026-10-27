// Package generatecontent is Gemini's generateContent wire, which OpenCode's
// google route speaks (0015-MADR D2), with its thinking request shape. The
// gemini provider speaks the Interactions API instead, in providers/gemini
// (MADR 0014).
package generatecontent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
	InlineData       *InlineData       `json:"inline_data,omitempty"`
	Text             *string           `json:"text,omitempty"`
	ThoughtSignature *string           `json:"thoughtSignature,omitempty"`
}

// InlineData is an image part's bytes, base64-encoded, and their media type,
// under the keys 0061-MADR D5 writes (its amendment "D5 made exact before
// Phase 6", item 5).
type InlineData struct {
	Data     string `json:"data"`
	MimeType string `json:"mime_type"`
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
		case llmprovider.ImageItem:
			// An image's bytes join the user content just before it, as a
			// part (0061-MADR D5). An image by URL is not sent: no form for it
			// is confirmed here, and the provider drops it first, warned (E7).
			if v.URL != "" {
				continue
			}
			appendPart(wire.RoleUser, Part{InlineData: &InlineData{MimeType: v.MediaType,
				Data: base64.StdEncoding.EncodeToString(v.Data)}}, true)
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
	var parts streamParts
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
			parts.add(part)
			if part.Text != "" && !part.Thought && part.FunctionCall == nil &&
				!yield(llmprovider.Event{Type: llmprovider.EventTextDelta, Text: part.Text}) {
				return true, nil
			}
		}
		if next.FinishReason == "" {
			return false, nil
		}
		raw.Candidates[0].FinishReason = next.FinishReason
		raw.Candidates[0].Content.Parts = parts.all()
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

// streamParts is a reply's parts as they stream: a text part joins a text
// part before it, and a thought part a thought part, as a whole reply sends
// one of each; a signature is kept wherever it arrives. The text joining the
// last part is held in a builder, so a stream of many parts costs time linear
// in its length (0053-MADR D1).
type streamParts struct {
	parts []replyPart
	tail  strings.Builder // text joining parts[len(parts)-1], not yet in it
}

// add adds one streamed part.
func (p *streamParts) add(part replyPart) {
	if n := len(p.parts); n > 0 && part.FunctionCall == nil && p.parts[n-1].FunctionCall == nil &&
		p.parts[n-1].Thought == part.Thought && p.parts[n-1].Text != "" {
		p.tail.WriteString(part.Text)
		if part.ThoughtSignature != "" {
			p.parts[n-1].ThoughtSignature = part.ThoughtSignature
		}
		return
	}
	p.flush()
	p.parts = append(p.parts, part)
}

// flush writes the held text into the last part.
func (p *streamParts) flush() {
	if p.tail.Len() > 0 {
		p.parts[len(p.parts)-1].Text += p.tail.String()
		p.tail.Reset()
	}
}

// all is every part, joined.
func (p *streamParts) all() []replyPart {
	p.flush()
	return p.parts
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

// ThinkingConfig returns a Gemini thinkingConfig in D2's shape for a request
// with no facts: ThinkingConfigFor with no facts and PreferredShape's shape
// (0062-MADR D2).
func ThinkingConfig(effort string, budget int) map[string]any {
	return ThinkingConfigFor(effort, budget, PreferredShape(Thinking{}, budget))
}

// Thinking is what a model's facts say of its Gemini thinking (0041-MADR D2),
// resolved by the provider so that this package reads no facts.
type Thinking struct {
	// Known reports that the facts answer. Without it the request decides
	// the shape (0062-MADR D2).
	Known bool
	// Budget is a thinking budget without efforts: thinkingBudget.
	Budget bool
	// Levels are the model's efforts: thinkingLevel.
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

// PreferredShape is D2's thinking shape on generateContent: the facts', when
// they answer, a level where they list levels; else a budget's, when the
// request sets one; else the level (0062-MADR D2, and its amendment "Phase
// 3's google-route levels, and 0013's B9"). D3's ladder may then send the
// other, when the model refused this one.
func PreferredShape(facts Thinking, budget int) llmprovider.ThinkingShape {
	switch {
	case facts.Known && len(facts.Levels) > 0, !facts.Known && budget <= 0:
		return llmprovider.ThinkingAdaptive
	default:
		return llmprovider.ThinkingBudget
	}
}

// ThinkingConfigFor returns a Gemini thinkingConfig of shape (0062-MADR D3),
// asking for thought summaries, as OpenCode's client asks
// (transform.ts:1280-1288, MADR 0014 §3). The level is thinkingLevel, the
// effort, with none leaving the model's default; the budget is
// thinkingBudget, the request's, else wire.LowEffortThinkingBudget for "low",
// else wire.DefaultThinkingBudget (R3). It replaces 0013-MADR B9's dynamic
// budget for an effort.
func ThinkingConfigFor(effort string, budget int, shape llmprovider.ThinkingShape) map[string]any {
	cfg := map[string]any{"includeThoughts": true}
	if shape == llmprovider.ThinkingAdaptive {
		if effort != "" {
			cfg["thinkingLevel"] = effort
		}
		return cfg
	}
	if budget <= 0 {
		budget = wire.DefaultThinkingBudget
		if effort == string(llmprovider.EffortLow) {
			budget = wire.LowEffortThinkingBudget
		}
	}
	cfg[keyThinkingBudget] = budget
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
