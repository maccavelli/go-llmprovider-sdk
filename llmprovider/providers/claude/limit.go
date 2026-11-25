package claude

import (
	"cmp"
	"context"
	"slices"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/limitmemo"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// The request's output limit (0089-MADR D6, D7): the Messages wire requires
// one, so a request that names none carries the model's chosen output fact,
// held to its window and to a cap the service stated, else the route's
// fallback; a refusal that states a cap resends once at it, and keeps it.

// memoKey is req's key in the memo of output caps (0089-MADR D5).
func (p *provider) memoKey(req *llmprovider.Request, stream bool) limitmemo.Key {
	base := p.baseURL
	if base == defaultBaseURL {
		base = ""
	}
	return limitmemo.Key{Provider: string(llmprovider.ProviderClaude), BaseURL: base, Scope: p.workspace,
		Model: p.modelFor(req), Stream: stream}
}

// limitRetry is req's output limit across its sends.
func (p *provider) limitRetry(req *llmprovider.Request, stream bool) *wire.LimitRetry {
	r := &wire.LimitRetry{Wire: wire.MessagesLimits, Model: p.modelFor(req),
		Observed: llmprovider.LimitObservation{Model: p.modelFor(req), BaseURL: p.givenBase, Scope: p.workspace}}
	if p.learn {
		memo, key := p.cache.LimitMemo(), p.memoKey(req, stream)
		r.Keep = func(tokens int) bool { return memo.Keep(key, tokens) }
	}
	return r
}

// observe is req's memo's drop of a cap a response shows wrong (DL39).
func (p *provider) observe(req *llmprovider.Request, stream bool) func(int) {
	memo, key := p.cache.LimitMemo(), p.memoKey(req, stream)
	return func(output int) { memo.Observe(key, output) }
}

// limitInput is what req's output limit is chosen from: its own value, the
// option, what the caches state of its model, read with the request's
// token, as its facts are (0041-MADR D1), and the window, else the input
// limit, with req's estimated input, which hold the value chosen (0088-MADR
// ST87). It sets r's resolved id (V6).
func (p *provider) limitInput(ctx context.Context, req *llmprovider.Request, token llmprovider.Token, stream bool,
	r *wire.LimitRetry) wire.LimitInput {
	model := p.modelFor(req)
	limits := catalog.CachedLimits(ctx, llmprovider.ProviderClaude, model,
		slices.Concat(p.listing, []llmprovider.Option{llmprovider.WithAPIKey(token.Value)})...)
	out := limits.Output
	if stream {
		out = limits.StreamedOutput
	}
	memo, key := p.cache.LimitMemo(), p.memoKey(req, false)
	r.Resolved = memo.Resolved(key)
	in := wire.LimitInput{Request: req.MaxOutputTokens, Option: p.maxTokens, Given: p.maxGiven, Sends: true,
		Required: true, Provider: string(llmprovider.ProviderClaude), Model: model,
		Window: cmp.Or(limits.Window.Tokens, limits.Input.Tokens), Estimate: wire.EstimateInput(req)}
	if out.Known {
		in.Output, in.Source = out.Tokens, out.Evidence[0].Source
	}
	// The route's fallback is this request's own read's (0093-MADR D7).
	if f := limits.RouteFallback; f.Known {
		in.Fallback, in.FallbackSource = f.Tokens, f.Evidence[0].Source
	}
	return in
}
