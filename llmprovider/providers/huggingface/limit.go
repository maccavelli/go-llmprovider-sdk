package huggingface

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
// window (the suffixed provider's, else the smallest live provider's,
// 0089-MADR DL10) and to a kept cap, and to the room the window leaves the
// request's estimated input (0096-MADR HF1, HF4); with no fact, or before the
// cached listing states a live provider's window, none is sent, and the
// service's own default applies (HF2, HF10). A refusal of the chosen value in
// a recorded form is sent once more at the stated cap (HF5); a refused
// caller's value is not resent. A cap a refusal states, validated, is kept for
// the process (DL37, HF5), and a response showing more output than a kept cap
// drops it (DL39). The Response's OutputLimit, and a done event's, is the limit sent.

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
	return limitmemo.Key{Provider: string(llmprovider.ProviderHuggingFace), BaseURL: p.givenBase, Model: cmp.Or(req.Model, p.model),
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
// Hugging Face sends the chosen fact, as 0096-MADR HF1 decided, and
// requires none; it sends it only once the cached listing states a live
// provider's window, which holds it (HF4), and none before (HF10).
func (p *provider) limitInput(ctx context.Context, req *llmprovider.Request, token llmprovider.Token,
	stream bool) wire.LimitInput {
	model := cmp.Or(req.Model, p.model)
	limits := catalog.CachedLimits(ctx, llmprovider.ProviderHuggingFace, model,
		slices.Concat(p.listing, []llmprovider.Option{llmprovider.WithAPIKey(token.Value)})...)
	out := limits.Output
	if stream {
		out = limits.StreamedOutput
	}
	in := wire.LimitInput{Request: req.MaxOutputTokens, Option: p.maxTokens, Given: p.maxGiven, Sends: liveWindow(limits.Window),
		Provider: string(llmprovider.ProviderHuggingFace), Model: model,
		Window: cmp.Or(limits.Window.Tokens, limits.Input.Tokens), Estimate: wire.EstimateInput(req)}
	if out.Known {
		in.Output, in.Source = out.Tokens, out.Evidence[0].Source
	}
	return in
}

// listingSource is the evidence name of what the cached listing states
// (catalog.Evidence.Source).
const listingSource = "listing:" + string(llmprovider.ProviderHuggingFace)

// liveWindow reports whether the cached listing states window: a live
// provider's, the hold 0096-MADR HF4 decides. A window only the document or a
// caller's layer states is not one (HF10).
func liveWindow(window catalog.Fact) bool {
	return slices.ContainsFunc(window.Evidence, func(e catalog.Evidence) bool { return e.Source == listingSource })
}
