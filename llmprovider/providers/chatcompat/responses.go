package chatcompat

import (
	"cmp"
	"io"
	"slices"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/chatcompletions"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/deferral"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/responses"
)

// The Responses path, for a server the caller knows speaks OpenAI's
// Responses API (0091-MADR ST5, ST50 to ST65, OQ5 "Opt-in WithResponses()").

// responsesValue is WithResponses' value among the options.
type responsesValue struct{}

// WithResponses sends Generate and Stream to POST {base}/responses, OpenAI's
// Responses API, in place of /chat/completions, for a server the caller
// knows speaks it (0091-MADR ST50). The listing and everything else are
// unchanged, and there is no fallback: a server that refuses the route
// answers with its error, classified by its status. It is a shared option
// (llmprovider.SharedOption), as WithID is, so another provider's New takes
// it and ignores it. The package doc names what a caller gives up on this
// path against Chat Completions.
func WithResponses() llmprovider.Option {
	return llmprovider.SharedOption("chatcompat.WithResponses", responsesValue{})
}

// responsesOn reports whether values carry WithResponses.
func responsesOn(values []any) bool {
	return slices.ContainsFunc(values, func(v any) bool {
		_, ok := v.(responsesValue)
		return ok
	})
}

// decode is a plain reply's decoder: the Chat Completions wire's, or the
// Responses wire's with WithResponses, labelled with the provider's id
// (0091-MADR ST52).
func (p *provider) decode(limit *wire.LimitRetry) func(io.Reader) (*llmprovider.Response, error) {
	if !p.responses {
		return chatcompletions.DecodeFor(string(p.id))
	}
	return func(body io.Reader) (*llmprovider.Response, error) {
		return p.responsesDecoder(limit).DecodeFor(string(p.id))(body)
	}
}

// events is a stream's reader: the Chat Completions wire's, or the Responses
// wire's with WithResponses.
func (p *provider) events(limit *wire.LimitRetry) func(string, io.Reader, func(llmprovider.Event) bool) error {
	if !p.responses {
		return chatcompletions.Events
	}
	return func(provider string, body io.Reader, yield func(llmprovider.Event) bool) error {
		return p.responsesDecoder(limit).Events(provider, body, yield)
	}
}

// responsesDecoder is the Responses decoder for the attempt limit last
// built: no Search and no Dispatch, deferred tools being appended
// (0091-MADR ST52, ST57); the provider's logger (0092-MADR ST20); and the
// limit that attempt's body carries, so that a reply at it is read as cut
// (0091-MADR ST33, ST55). It is read when the reply is, after the build, so
// a resend's decoder has the resend's limit.
func (p *provider) responsesDecoder(limit *wire.LimitRetry) responses.Decoder {
	d := responses.Decoder{Logger: p.logger}
	if sent := limit.Report(); sent.Sent {
		d.Limit = sent.Tokens
	}
	return d
}

// responsesBody is the Responses request for req, by its model's facts, as
// the Chat Completions body is (0091-MADR ST51): the input, instructions
// first as a system item; store false, and encrypted reasoning asked for
// (ST18); the output limit only when one is sent (0089-MADR DL48); tools,
// with strict as the caller sets it, and the Responses tool choice; deferred
// tools appended as loaded (ST57); and the effort by 0062's ladder over the
// facts, with its warnings. Prior reasoning goes back only when it carries
// encrypted content (responses.Input), never under the facts' replay field.
func (p *provider) responsesBody(req *llmprovider.Request, facts catalog.ModelInfo, learner *wire.Learner,
	outLimit llmprovider.SentLimit) (map[string]any, []llmprovider.Warning) {
	model := cmp.Or(req.Model, p.model)
	input := req.Input
	if req.Instructions != "" {
		input = append([]llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: req.Instructions}}, input...)
	}
	body := map[string]any{
		"model":   model,
		"input":   responses.Input(input),
		"store":   false,
		"include": []string{"reasoning.encrypted_content"},
	}
	if outLimit.Sent {
		body["max_output_tokens"] = outLimit.Tokens
	}
	wire.AddResponsesTools(body, deferral.Tools(req, deferral.ForRequest(deferral.Other, model, req)), req.ToolChoice)
	effort, warnings := learner.Effort(llmprovider.Effort(p.effort(req)), facts.Reasoning.Efforts,
		facts.Has(catalog.FieldReasoningEfforts))
	if effort != "" {
		body["reasoning"] = map[string]any{"effort": string(effort)}
	}
	return body, warnings
}
