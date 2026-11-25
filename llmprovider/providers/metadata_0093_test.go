package providers

// 0093-MADR's rows on the document a provider is given (MH1, MH21, MH53,
// MH54). Each names catalog.WithMetadata, so each takes its red from its
// plant (0093-PLAN Phase 3).

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/chatcompat"
)

// doc0093 parses body, failing the test.
func doc0093(t *testing.T, body string) catalog.Metadata {
	t.Helper()
	m, err := catalog.ParseMetadata(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// TestWithMetadata_Shared_0093 (MH1; D1): WithMetadata is a shared option:
// every built-in provider and chatcompat take it; a provider outside the
// module ignores it; llmprovider.For scopes it to one provider.
func TestWithMetadata_Shared_0093(t *testing.T) {
	opt := catalog.WithMetadata(doc0093(t, `{"anthropic":{"models":{"claude-x":{"limit":{"output":1000}}}}}`))
	for _, d := range Default().Descriptors() {
		if _, err := New(d.ID, llmprovider.WithAPIKey("k"), llmprovider.WithModel("m"), opt); err != nil {
			t.Errorf("%s: New given WithMetadata: %v", d.ID, err)
		}
	}
	if _, err := chatcompat.New(chatcompat.WithID("acme-0093"), llmprovider.WithBaseURL("http://127.0.0.1:1/v1"),
		llmprovider.WithModel("m"), opt); err != nil {
		t.Errorf("chatcompat: New given WithMetadata: %v", err)
	}
	if _, err := llmprovider.ResolveOptions("outside-0093", []llmprovider.Option{opt}); err != nil {
		t.Errorf("a provider outside the module: %v; want the option ignored", err)
	}
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderOpencodeZen,
		[]llmprovider.Option{llmprovider.For(llmprovider.ProviderClaude, opt)})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range st.Values() {
		if _, ok := v.(catalog.Metadata); ok {
			t.Error("For(claude, WithMetadata) reached opencode-zen")
		}
	}
}

// TestProviders_FactsFromHandedDocument_0093 (MH21; D2, Q4): a provider's
// request reads its facts from the document given to its New: a Claude
// request with nothing listed sends the document's output limit, basis
// fact; without the document it is refused.
func TestProviders_FactsFromHandedDocument_0093(t *testing.T) {
	doc := catalog.WithMetadata(doc0093(t, `{"anthropic":{"models":{"claude-doc-0093":{"limit":{"context":200000,"output":30000}}}}}`))
	srv := newLimitServer(t, "", canned(wirecase.MessagesReply))
	p, err := New(llmprovider.ProviderClaude, llmprovider.WithBaseURL(srv.URL), llmprovider.WithAPIKey("k"),
		llmprovider.WithModel("claude-doc-0093"), llmprovider.WithModelProbes(false), testCache(t), doc)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Generate(context.Background(), hiRequest())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got, ok := sentLimit(lastPost(srv)); !ok || got != 30000 {
		t.Errorf("sent %v (found %t); want the document's 30000", got, ok)
	}
	if resp.OutputLimit.Basis != llmprovider.LimitBasisFact || resp.OutputLimit.Source != "models.dev" {
		t.Errorf("OutputLimit = %+v; want basis fact from models.dev", resp.OutputLimit)
	}
}

// TestLimits_RouteFallbackPerCall_0093 (MH53; D7, Q4): two Claude providers
// on one cache, under -race, one given a document and one not, each send a
// model no source states: the first the document's route fallback, the
// second nothing, refused: a fallback found with one call's document is
// never read by another's request. Both providers are on one server, so one
// base URL, and send one model: the memo key BASE kept the fallback under
// (0089-MADR D5) is the same for both, so a note kept under it would be seen.
func TestLimits_RouteFallbackPerCall_0093(t *testing.T) {
	cache := testCache(t)
	doc := catalog.WithMetadata(doc0093(t, `{"anthropic":{"models":{"claude-a":{"limit":{"output":20000}}}}}`))
	srv := newLimitServer(t, "", canned(wirecase.MessagesReply))
	with, err := New(llmprovider.ProviderClaude, llmprovider.WithBaseURL(srv.URL), llmprovider.WithAPIKey("k"),
		llmprovider.WithModel("claude-unstated-0093"), llmprovider.WithModelProbes(false), cache, doc)
	if err != nil {
		t.Fatal(err)
	}
	without, err := New(llmprovider.ProviderClaude, llmprovider.WithBaseURL(srv.URL), llmprovider.WithAPIKey("k"),
		llmprovider.WithModel("claude-unstated-0093"), llmprovider.WithModelProbes(false), cache)
	if err != nil {
		t.Fatal(err)
	}
	// One call with the document first, so that every call after it has a
	// fallback found with a document to be wrongly read.
	if _, err := with.Generate(context.Background(), hiRequest()); err != nil {
		t.Fatalf("with the document: %v", err)
	}
	var wg sync.WaitGroup
	for n := range 16 {
		wg.Go(func() {
			if n%2 == 0 {
				if _, err := with.Generate(context.Background(), hiRequest()); err != nil {
					t.Errorf("with the document: %v", err)
				}
				return
			}
			if _, err := without.Generate(context.Background(), hiRequest()); !errors.Is(err, llmprovider.ErrInvalidRequest) {
				t.Errorf("without the document: err = %v; want refused", err)
			}
		})
	}
	wg.Wait()
	// Only the provider given the document sent a request: its first
	// call and eight more, each at its fallback; the other's eight were
	// refused before the wire.
	posts := srv.posts()
	if len(posts) != 9 {
		t.Errorf("%d request(s) sent; want the 9 of the provider given the document", len(posts))
	}
	for _, body := range posts {
		if got, ok := sentLimit(body); !ok || got != 20000 {
			t.Errorf("with the document: sent %v (found %t); want its fallback 20000", got, ok)
		}
	}
}

// TestLookupLimits_DocumentNotSeenByRequest_0093 (MH54; D2, Q4): a document
// handed to LookupLimits is not seen by a request of a provider built
// without one on the same cache: the Claude request is refused. The lookup
// and the provider are on one base URL, so they share a memo key.
func TestLookupLimits_DocumentNotSeenByRequest_0093(t *testing.T) {
	cache := testCache(t)
	doc := catalog.WithMetadata(doc0093(t, `{"anthropic":{"models":{"claude-opus-4-6":{"limit":{"context":1000000,"output":64000}}}}}`))
	srv := newLimitServer(t, "", canned(wirecase.MessagesReply))
	l, err := catalog.LookupLimits(context.Background(), llmprovider.ProviderClaude, nil, "claude-opus-4-6",
		llmprovider.WithBaseURL(srv.URL), doc, cache)
	if err != nil || !l.Output.Known {
		t.Fatalf("LookupLimits with the document: %+v, %v; want its output known", l.Output, err)
	}
	p, err := New(llmprovider.ProviderClaude, llmprovider.WithBaseURL(srv.URL), llmprovider.WithAPIKey("k"),
		llmprovider.WithModel("claude-opus-4-6"), llmprovider.WithModelProbes(false), cache)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Generate(context.Background(), hiRequest()); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("a request of a provider given no document: err = %v; want refused", err)
	}
	if n := len(srv.posts()); n != 0 {
		t.Errorf("%d request(s) sent; want none", n)
	}
}
