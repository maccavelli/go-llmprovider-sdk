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

// The request's output limit (0089-MADR D6, D7): the field is optional on
// Chat Completions, so a request carries one only as its caller gives it: its own
// MaxOutputTokens, sent as given, or WithMaxTokens' value, cut to the
// model's stated limit and never raised. Otherwise none is sent, and the
// service's own default applies (DL47, DL48). A refused caller's
// value is not resent; a cap a refusal states, validated, is kept for the
// process (DL37), and a response showing more output than a kept cap drops
// it (DL39). The Response, and a stream's done event, carry the limit sent in
// OutputLimit.

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

// chooseLimit is the limit req's next send carries, chosen by r. The stated
// output, which only cuts a given option, is read from the caches with the
// request's token, as its facts are (0041-MADR D1).
func (p *provider) chooseLimit(ctx context.Context, req *llmprovider.Request, token llmprovider.Token, stream bool,
	r *wire.LimitRetry) llmprovider.SentLimit {
	return r.Optional(req.MaxOutputTokens, p.maxTokens, p.maxGiven, func() (int, string) {
		limits := catalog.CachedLimits(ctx, llmprovider.ProviderTogether, cmp.Or(req.Model, p.model),
			slices.Concat(p.listing, []llmprovider.Option{llmprovider.WithAPIKey(token.Value)})...)
		out := limits.Output
		if stream {
			out = limits.StreamedOutput
		}
		if !out.Known {
			return 0, ""
		}
		return out.Tokens, out.Evidence[0].Source
	})
}
