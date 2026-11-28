package together

import (
	"cmp"
	"context"
	"slices"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/cacheref"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/limitmemo"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// The request's output limit (0089-MADR D6, D7; 0096-MADR): the field is
// optional on Chat Completions. A request's own MaxOutputTokens is sent as
// given, and WithMaxTokens' value cut to the model's stated limit and never
// raised. Otherwise the model's chosen output fact is sent, held to its
// window and to a kept cap, and to the room the window leaves the request's
// estimated input (0096-MADR TG1, TG2, TG5); with no fact, none is sent, and
// the service's own default applies (TG3). A refusal of the chosen value in
// a recorded form is sent once more at the stated cap (TG6); a refused
// caller's value is not resent. A cap a refusal states, validated, is kept
// for the process (DL37, TG7), and a response showing more output than a
// kept cap drops it (DL39). The Response, and a stream's done event, carry
// the limit sent in OutputLimit.

// limitState is what a request's output limit reads of the provider's
// options.
type limitState struct {
	// maxGiven reports that WithMaxTokens gave maxTokens, or that a listing
	// probe set it (0088-MADR ST15, ST26).
	maxGiven bool
	// learn is false under llmprovider.WithoutLimitLearning (0089-MADR DL73).
	learn bool
	// limits is catalog.WithCache's memo of output caps, nil for the
	// process's; givenBase is WithBaseURL's, empty for the default (D5).
	limits    *limitmemo.Memo
	givenBase string
}

// newLimitState is the limitState st's options give.
func newLimitState(st *llmprovider.Settings) limitState {
	return limitState{maxGiven: st.MaxTokensGiven(), learn: !limitmemo.LearningOff(st.Values()),
		limits: cacheref.From(st.Values()).LimitMemo(), givenBase: st.BaseURL()}
}

// memoKey is req's key in the memo of output caps (0089-MADR D5).
func (p *provider) memoKey(req *llmprovider.Request, stream bool) limitmemo.Key {
	return limitmemo.Key{Provider: string(llmprovider.ProviderTogether), BaseURL: p.givenBase, Model: cmp.Or(req.Model, p.model),
		Stream: stream}
}

// limitRetry is req's output limit across its sends.
func (p *provider) limitRetry(req *llmprovider.Request, stream bool) *wire.LimitRetry {
	model := cmp.Or(req.Model, p.model)
	r := &wire.LimitRetry{Wire: wire.ChatCompletionsLimits, Model: model,
		Observed: llmprovider.LimitObservation{Model: model, BaseURL: p.givenBase}}
	if p.learn {
		memo, key := p.limits, p.memoKey(req, stream)
		r.Keep = func(tokens int) bool { return memo.Keep(key, tokens) }
	}
	return r
}

// observe is req's memo's drop of a cap a response shows wrong (DL39).
func (p *provider) observe(req *llmprovider.Request, stream bool) func(int) {
	memo, key := p.limits, p.memoKey(req, stream)
	return func(output int) { memo.Observe(key, output) }
}

// limitInput is what req's output limit is chosen from: its own value, the
// option, and what the caches state of its model, read with the request's
// token, as its facts are (0041-MADR D1), and the window, else the input
// limit, with req's estimated input, which hold the fact (0088-MADR ST87).
// Together sends the chosen fact, as 0096-MADR TG1 and TG2 decided, and
// requires none.
func (p *provider) limitInput(ctx context.Context, req *llmprovider.Request, token llmprovider.Token,
	stream bool) wire.LimitInput {
	model := cmp.Or(req.Model, p.model)
	limits := catalog.CachedLimits(ctx, llmprovider.ProviderTogether, model,
		slices.Concat(p.listing, []llmprovider.Option{llmprovider.WithAPIKey(token.Value)})...)
	out := limits.Output
	if stream {
		out = limits.StreamedOutput
	}
	in := wire.LimitInput{Request: req.MaxOutputTokens, Option: p.maxTokens, Given: p.maxGiven, Sends: true,
		Provider: string(llmprovider.ProviderTogether), Model: model,
		Window: cmp.Or(limits.Window.Tokens, limits.Input.Tokens), Estimate: wire.EstimateInput(req)}
	if out.Known {
		in.Output, in.Source = out.Tokens, out.Evidence[0].Source
	}
	return in
}
