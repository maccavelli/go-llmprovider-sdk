package llmprovider_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/claude"
)

// The pass-back of 0089-MADR D5: what CachedLimits reads of
// WithLimitObservations, and what New refuses. Every lookup reads only what
// the options give: no document, no listing, no request.

// passBackModel is a model no source states, in no extract section.
const passBackModel = "claude-pass-back-0089"

// observed is a passed-back cap of tokens for model, seen ago.
func observed(model string, tokens int, ago time.Duration) llmprovider.LimitObservation {
	return llmprovider.LimitObservation{Model: model, Tokens: tokens, Seen: time.Now().Add(-ago)}
}

// TestLimitObservations_PassBack_0089 (DL40, DL41): a passed-back cap is
// used, as "remembered", only where its base URL and scope match, only
// within 24 hours of its Seen, and only where no other source states the
// output; otherwise it is kept as evidence and the stated source chosen.
func TestLimitObservations_PassBack_0089(t *testing.T) {
	ctx := context.Background()
	off := llmprovider.WithoutModelMetadata()
	zen := func(opts ...llmprovider.Option) catalog.Limits {
		return catalog.CachedLimits(ctx, llmprovider.ProviderOpencodeZen, passBackModel,
			append([]llmprovider.Option{off, catalog.WithCache(new(catalog.Cache))}, opts...)...)
	}

	// (1) OpenCode's messages route, a model no source states.
	l := zen(llmprovider.WithLimitObservations(observed(passBackModel, 30000, time.Hour)))
	if f := l.Output; !f.Known || f.Tokens != 30000 || len(f.Evidence) != 1 || f.Evidence[0].Source != "remembered" {
		t.Errorf("seen an hour ago: output %+v; want 30000 from remembered", f)
	}
	if f := zen(llmprovider.WithLimitObservations(observed(strings.ToUpper(passBackModel), 30000, time.Hour))).Output; f.Tokens != 30000 {
		t.Errorf("the model in upper case: output %+v; want 30000, matched in lower case", f)
	}
	if f := zen(llmprovider.WithLimitObservations(observed(passBackModel, 30000, 25*time.Hour))).Output; f.Known || len(f.Evidence) != 0 {
		t.Errorf("seen 25 hours ago: output %+v; want unknown, the cap unused", f)
	}
	elsewhere := observed(passBackModel, 30000, time.Hour)
	elsewhere.BaseURL = "https://gateway.example.test"
	if f := zen(llmprovider.WithLimitObservations(elsewhere)).Output; f.Known || len(f.Evidence) != 0 {
		t.Errorf("another base URL: output %+v; want unknown, the cap unused", f)
	}
	// The gateway's default base URL is read as empty on both sides (DL40,
	// as annotated 2026-11-23): a cap passed back with no base URL matches a
	// lookup that names the default, as a provider's own request does, and a
	// cap naming the default matches a lookup that names none.
	const zenDefault = "https://opencode.ai/zen/v1"
	if f := zen(llmprovider.WithBaseURL(zenDefault),
		llmprovider.WithLimitObservations(observed(passBackModel, 30000, time.Hour))).Output; f.Tokens != 30000 {
		t.Errorf("no base URL passed back, the default looked up: output %+v; want the remembered 30000", f)
	}
	atDefault := observed(passBackModel, 30000, time.Hour)
	atDefault.BaseURL = zenDefault + "/"
	if f := zen(llmprovider.WithLimitObservations(atDefault)).Output; f.Tokens != 30000 {
		t.Errorf("the default passed back, none looked up: output %+v; want the remembered 30000", f)
	}
	inOrg := observed("anthropic/claude-sonnet-4.5", 30000, time.Hour)
	inOrg.Scope = "org-a"
	kilo := func(org string) catalog.Fact {
		return catalog.CachedLimits(ctx, llmprovider.ProviderKilo, "anthropic/claude-sonnet-4.5", off,
			catalog.WithCache(new(catalog.Cache)), catalog.WithKiloOrganization(org), llmprovider.WithLimitObservations(inOrg)).Output
	}
	if f := kilo("org-b"); hasSource(f, "remembered") {
		t.Errorf("another Kilo organization: output %+v; want the cap unused", f)
	}
	if f := kilo("org-a"); !hasSource(f, "remembered") {
		t.Errorf("the cap's own Kilo organization: output %+v; want the remembered 30000", f)
	}

	// (2) A registry fact of 64,000 beside a passed-back 30,000: the
	// registry's chosen, the passed-back kept as evidence.
	meta, err := catalog.ParseMetadata(strings.NewReader(`{"opencode":{"models":{"` + passBackModel + `":{"limit":{"output":64000}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	l = zen(catalog.WithModelLayer(meta), llmprovider.WithLimitObservations(observed(passBackModel, 30000, time.Hour)))
	if f := l.Output; !f.Known || f.Tokens != 64000 || f.Evidence[0].Source != "models.dev" || !hasSource(f, "remembered") {
		t.Errorf("a registry fact beside a passed-back cap: output %+v; want 64000 from models.dev, the remembered 30000 kept", f)
	}
}

// hasSource reports whether f's evidence holds one from src.
func hasSource(f catalog.Fact, src string) bool {
	for _, e := range f.Evidence {
		if e.Source == src {
			return true
		}
	}
	return false
}

// TestLimitObservations_Invalid_0089 (DL42): New refuses, with
// ErrInvalidRequest, an observation with no model, Tokens below 1, or a zero
// Seen; it takes a valid one.
func TestLimitObservations_Invalid_0089(t *testing.T) {
	valid := observed("claude-opus-4-6", 64000, time.Minute)
	noModel, noTokens, noSeen := valid, valid, valid
	noModel.Model, noTokens.Tokens, noSeen.Seen = "", 0, time.Time{}
	for name, o := range map[string]llmprovider.LimitObservation{"no model": noModel, "Tokens 0": noTokens, "zero Seen": noSeen} {
		opts := []llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithLimitObservations(valid, o)}
		if _, err := llmprovider.ResolveOptions(llmprovider.ProviderClaude, opts); !errors.Is(err, llmprovider.ErrInvalidRequest) {
			t.Errorf("%s: ResolveOptions err = %v; want ErrInvalidRequest", name, err)
		}
		if _, err := claude.New(opts...); !errors.Is(err, llmprovider.ErrInvalidRequest) {
			t.Errorf("%s: claude.New err = %v; want ErrInvalidRequest", name, err)
		}
	}
	if _, err := claude.New(llmprovider.WithAPIKey("k"), llmprovider.WithLimitObservations(valid)); err != nil {
		t.Errorf("a valid observation: claude.New err = %v; want nil", err)
	}
}
