package providers

// 0093-MADR's rows on a request with no document, tested where they build at
// the record's base as well as after it, so each is seen red there first:
// MH17 (the route fallback with no extract), MH18 (a cold Messages request
// refused) and MH51 (a cold Kilo request sends no limit).

import (
	"context"
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// TestOutputLimit_ColdMessagesRefused_0093 (MH18; Q2): a Claude request, and
// one on OpenCode's messages route, with nothing listed, no document, no
// layer and no limit given, is refused before it is sent, with an error
// matching ErrInvalidRequest.
func TestOutputLimit_ColdMessagesRefused_0093(t *testing.T) {
	for name, build := range map[string]func(string, ...llmprovider.Option) (llmprovider.Provider, error){
		"claude":                byID(llmprovider.ProviderClaude),
		"opencode-zen/messages": zenOn(opencode.RouteMessages),
	} {
		t.Run(name, func(t *testing.T) {
			srv := newLimitServer(t, "", canned(wirecase.MessagesReply))
			p, err := build(srv.URL, llmprovider.WithAPIKey("k"), llmprovider.WithModel("claude-sonnet-5"),
				llmprovider.WithModelProbes(false), testCache(t))
			if err != nil {
				t.Fatal(err)
			}
			_, err = p.Generate(context.Background(), hiRequest())
			if !errors.Is(err, llmprovider.ErrInvalidRequest) {
				t.Errorf("%s: Generate error = %v; want one matching ErrInvalidRequest", name, err)
			}
			if n := len(srv.posts()); n != 0 {
				t.Errorf("%s: %d request(s) sent; want none", name, n)
			}
		})
	}
}

// TestRouteFallback_NoExtract_0093 (MH17; D7): with no document, a Claude
// model no source states takes the route's fallback from the cached
// listing's models only: with nothing listed there is none, and the request
// is refused; with a listing that states a model's limit, the fallback is
// that limit.
func TestRouteFallback_NoExtract_0093(t *testing.T) {
	t.Run("nothing listed", func(t *testing.T) {
		srv := newLimitServer(t, "", canned(wirecase.MessagesReply))
		p, err := New(llmprovider.ProviderClaude, llmprovider.WithBaseURL(srv.URL), llmprovider.WithAPIKey("k"),
			llmprovider.WithModel(omitModel), llmprovider.WithModelProbes(false), testCache(t))
		if err != nil {
			t.Fatal(err)
		}
		_, err = p.Generate(context.Background(), hiRequest())
		if got, ok := sentLimit(lastPost(srv)); ok || !errors.Is(err, llmprovider.ErrInvalidRequest) {
			t.Errorf("no listing, no document: sent %v (found %t), error %v; want none sent, refused", got, ok, err)
		}
	})
	t.Run("a listing states one", func(t *testing.T) {
		listing := `{"data":[{"id":"claude-sonnet-5","type":"model","max_tokens":32000,"max_input_tokens":200000}],` +
			`"has_more":false}`
		srv := newLimitServer(t, listing, canned(wirecase.MessagesReply))
		cache := testCache(t)
		p, err := New(llmprovider.ProviderClaude, llmprovider.WithBaseURL(srv.URL), llmprovider.WithAPIKey("k"),
			llmprovider.WithModel(omitModel), llmprovider.WithModelProbes(false), cache)
		if err != nil {
			t.Fatal(err)
		}
		lister, ok := p.(llmprovider.ModelLister)
		if !ok {
			t.Fatal("claude lists no models")
		}
		if _, err := lister.ListModels(context.Background()); err != nil {
			t.Fatalf("ListModels: %v", err)
		}
		if _, err := p.Generate(context.Background(), hiRequest()); err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if got, ok := sentLimit(lastPost(srv)); !ok || got != 32000 {
			t.Errorf("the listing states 32000 for its one model: sent %v (found %t); want 32000", got, ok)
		}
	})
}

// TestKilo_ColdSendsNone_0093 (MH51; Q2): a Kilo request with nothing listed,
// no document and no limit given sends no output limit, and reports none,
// where the base sent the extract's 393,216 for the model.
func TestKilo_ColdSendsNone_0093(t *testing.T) {
	srv := newLimitServer(t, "", canned(wirecase.ChatReply))
	p, err := New(llmprovider.ProviderKilo, llmprovider.WithBaseURL(srv.URL), llmprovider.WithAPIKey("k"),
		llmprovider.WithModel("deepseek/deepseek-v4.1-flash"), llmprovider.WithModelProbes(false), testCache(t))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Generate(context.Background(), hiRequest())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got, ok := sentLimit(lastPost(srv)); ok {
		t.Errorf("cold Kilo request sent an output limit of %v; want none", got)
	}
	if resp.OutputLimit.Sent || resp.OutputLimit.Basis != llmprovider.LimitBasisNone {
		t.Errorf("OutputLimit = %+v; want none sent, basis %q", resp.OutputLimit, llmprovider.LimitBasisNone)
	}
}
