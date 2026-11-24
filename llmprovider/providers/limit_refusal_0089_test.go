package providers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/cacheref"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/limitmemo"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// 0089-PLAN Phase 2's tests that assert the resend at a stated limit or a
// provider's request, built in the merged phase's layer B+C (deviation 7's
// "Tests by layer"). The refusal texts are the forum copies 0089-MADR W7
// names and 0088-MADR C38, C39 and C97's forms; none is a recorded reply.

// memoIn is the memo of output caps a provider given cache reads.
func memoIn(t *testing.T, cache *catalog.Cache) *limitmemo.Memo {
	t.Helper()
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderClaude, []llmprovider.Option{catalog.WithCache(cache)})
	if err != nil {
		t.Fatal(err)
	}
	return cacheref.From(st.Values()).LimitMemo()
}

// chatError is a Chat Completions refusal carrying msg, as a 400.
func chatError(msg string) string {
	return fmt.Sprintf(`{"error":{"message":%q,"type":"invalid_request_error","code":400}}`, msg)
}

// messagesError is a Messages refusal carrying msg, as a 400.
func messagesError(msg string) string {
	return fmt.Sprintf(`{"type":"error","error":{"type":"invalid_request_error","message":%q}}`, msg)
}

// refuseFirst answers the first POST with a 400 carrying refusal, and every
// other with ok.
func refuseFirst(refusal, ok string) func(int, map[string]any) (int, string) {
	return func(n int, _ map[string]any) (int, string) {
		if n == 0 {
			return http.StatusBadRequest, refusal
		}
		return http.StatusOK, ok
	}
}

// TestLimitRefusal_WindowForms_0089 (DL35, V5; 0088-MADR ST92): on Kilo,
// which sends its discovered 16,384, a window form stating a window of
// 32,768 and a request of about 40,000 tokens is resent once at the room it
// leaves, 9,152, in OpenRouter's form and in Kilo's documented one; on
// Together, a caller's WithMaxTokens(16384) refused in the split form is
// sent once and the refusal returned. A window form whose stated total is
// below the limit sent states a negative input, and is not read (V5): sent
// once. After each, no window evidence is shown and the memo is empty.
func TestLimitRefusal_WindowForms_0089(t *testing.T) {
	ctx := context.Background()
	const kiloModel = "openai/gpt-6-luna"
	for _, tc := range []struct {
		name    string
		id      llmprovider.ProviderID
		model   string
		opts    []llmprovider.Option
		refusal string
		resent  int // 0: one send, the refusal returned
		source  string
	}{
		{"kilo, OpenRouter's form", llmprovider.ProviderKilo, kiloModel,
			[]llmprovider.Option{catalog.WithModelLayer(limitLayer{kiloModel, 16384})},
			"This endpoint's maximum context length is 32768 tokens. However, you requested about 40000 tokens " +
				"(23616 of text input, 16384 in the output). Please reduce the length of either one.",
			9152, "refusal: openrouter context length"},
		{"kilo, Kilo's documented form", llmprovider.ProviderKilo, kiloModel,
			[]llmprovider.Option{catalog.WithModelLayer(limitLayer{kiloModel, 16384})},
			"Your request exceeds the model's context window of 32768 tokens. Your request contains approximately 40000 tokens.",
			9152, "refusal: kilo context window"},
		{"together, a caller's limit refused in the split form", llmprovider.ProviderTogether, "openai/gpt-oss-120b",
			[]llmprovider.Option{llmprovider.WithMaxTokens(16384)},
			"This model's maximum context length is 32768 tokens. However, you requested 40000 tokens " +
				"(23616 in the messages, 16384 in the completion). Please reduce the length of the messages or completion.",
			0, ""},
		{"kilo, a stated total below the limit sent (V5)", llmprovider.ProviderKilo, kiloModel,
			[]llmprovider.Option{catalog.WithModelLayer(limitLayer{kiloModel, 16384})},
			"This endpoint's maximum context length is 32768 tokens. However, you requested about 10000 tokens.",
			0, ""},
	} {
		srv := newLimitServer(t, "", refuseFirst(chatError(tc.refusal), wirecase.ChatReply))
		cache := new(catalog.Cache)
		opts := append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithoutModelMetadata(),
			llmprovider.WithModelProbes(false), catalog.WithCache(cache), llmprovider.WithBaseURL(srv.URL),
			llmprovider.WithModel(tc.model)}, tc.opts...)
		p, err := New(tc.id, opts...)
		if err != nil {
			t.Fatal(err)
		}
		before := catalog.CachedLimits(ctx, tc.id, tc.model, opts...)
		resp, err := p.Generate(ctx, hiRequest())
		posts := srv.posts()
		if got, _ := sentLimit(posts[0]); got != 16384 {
			t.Errorf("%s: the first send carried %v; want 16384", tc.name, got)
		}
		if tc.resent == 0 {
			if err == nil || !errors.Is(err, llmprovider.ErrInvalidRequest) || len(posts) != 1 {
				t.Errorf("%s: %d send(s), err %v; want one send and the refusal returned", tc.name, len(posts), err)
			}
		} else {
			if err != nil || len(posts) != 2 {
				t.Fatalf("%s: %d send(s), err %v; want a resend", tc.name, len(posts), err)
			}
			if got, _ := sentLimit(posts[1]); int(got) != tc.resent {
				t.Errorf("%s: resent at %v; want %d, the room the window leaves", tc.name, got, tc.resent)
			}
			want := llmprovider.SentLimit{Tokens: tc.resent, Sent: true, Basis: llmprovider.LimitBasisRefusal, Source: tc.source}
			if resp.OutputLimit != want {
				t.Errorf("%s: OutputLimit = %+v; want %+v", tc.name, resp.OutputLimit, want)
			}
		}
		l := catalog.CachedLimits(ctx, tc.id, tc.model, opts...)
		if !reflect.DeepEqual(l.Window, before.Window) || !reflect.DeepEqual(l.Input, before.Input) {
			t.Errorf("%s: window %+v, input %+v; want them as before the refusal, %+v and %+v: no evidence from a window form",
				tc.name, l.Window, l.Input, before.Window, before.Input)
		}
		for _, e := range l.Output.Evidence {
			if e.Source == "observed" {
				t.Errorf("%s: output evidence %+v; want nothing kept from a window form", tc.name, e)
			}
		}
		for _, stream := range []bool{false, true} {
			k := limitmemo.Key{Provider: string(tc.id), BaseURL: srv.URL, Model: tc.model, Stream: stream}
			if c, ok := memoIn(t, cache).Cap(k); ok {
				t.Errorf("%s: the memo holds %d; want it empty after a window form", tc.name, c)
			}
		}
	}
}

// TestLimitRefusal_CallerLimit_0089 (DL37; 0088-MADR ST17, ST18): a
// caller's limit refused with a stated cap is sent once, the refusal
// returned as ErrInvalidRequest, and the cap kept, so CachedLimits shows the
// bound: the request's own MaxOutputTokens, and WithMaxTokens' value alike.
func TestLimitRefusal_CallerLimit_0089(t *testing.T) {
	ctx := context.Background()
	const model = "claude-caller-0089" // no source states it
	for _, tc := range []struct {
		name    string
		request int
		opts    []llmprovider.Option
		sent    string
	}{
		{"the request's MaxOutputTokens", 200000, nil, "200000"},
		{"WithMaxTokens", 0, []llmprovider.Option{llmprovider.WithMaxTokens(150000)}, "150000"},
	} {
		srv := newLimitServer(t, "", refuseFirst(messagesError("max_tokens: "+tc.sent+
			" > 128000, which is the maximum allowed number of output tokens for "+model), wirecase.MessagesReply))
		cache := new(catalog.Cache)
		opts := append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithoutModelMetadata(),
			llmprovider.WithModelProbes(false), catalog.WithCache(cache), llmprovider.WithBaseURL(srv.URL),
			llmprovider.WithModel(model)}, tc.opts...)
		p, err := New(llmprovider.ProviderClaude, opts...)
		if err != nil {
			t.Fatal(err)
		}
		req := hiRequest()
		req.MaxOutputTokens = tc.request
		_, err = p.Generate(ctx, req)
		if n := len(srv.posts()); n != 1 || !errors.Is(err, llmprovider.ErrInvalidRequest) {
			t.Errorf("%s: %d send(s), err %v; want one send and ErrInvalidRequest", tc.name, n, err)
		}
		l := catalog.CachedLimits(ctx, llmprovider.ProviderClaude, model, opts...)
		if !l.Output.Known || l.Output.Tokens != 128000 || l.Output.Evidence[0].Source != "observed" {
			t.Errorf("%s: output %+v; want the stated 128000 kept, observed", tc.name, l.Output)
		}
	}
}

// TestLimitMemo_Concurrent_0089 (DL43): under -race, 32 concurrent requests
// on one Claude provider, half of them refused at the stated 128,000 with
// the cap form stating 64,000: each refused request is resent once, at
// 64,000; the memo holds one entry for the model, 64,000.
func TestLimitMemo_Concurrent_0089(t *testing.T) {
	ctx := context.Background()
	const model = "claude-opus-4-6" // the extract states 128,000
	refusal := messagesError("max_tokens: 128000 > 64000, which is the maximum allowed number of output tokens for " + model)
	srv := newLimitServer(t, "", func(_ int, body map[string]any) (int, string) {
		limit, _ := sentLimit(body)
		msgs, _ := body["messages"].([]any)
		first, _ := msgs[0].(map[string]any)
		if limit == 128000 && fmt.Sprint(first["content"]) == "refuse" {
			return http.StatusBadRequest, refusal
		}
		return http.StatusOK, wirecase.MessagesReply
	})
	cache := new(catalog.Cache)
	opts := []llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithoutModelMetadata(),
		llmprovider.WithModelProbes(false), catalog.WithCache(cache), llmprovider.WithBaseURL(srv.URL),
		llmprovider.WithModel(model)}
	p, err := New(llmprovider.ProviderClaude, opts...)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make([]error, 32)
	limits := make([]llmprovider.SentLimit, 32)
	for i := range 32 {
		wg.Go(func() {
			text := "plain"
			if i%2 == 0 {
				text = "refuse"
			}
			resp, err := p.Generate(ctx, &llmprovider.Request{Input: []llmprovider.Item{
				llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: text}}})
			errs[i] = err
			if resp != nil {
				limits[i] = resp.OutputLimit
			}
		})
	}
	wg.Wait()
	resent := 0
	for i, err := range errs {
		if err != nil {
			t.Errorf("request %d: %v", i, err)
		}
		if limits[i].Basis == llmprovider.LimitBasisRefusal {
			resent++
			if limits[i].Tokens != 64000 || i%2 != 0 {
				t.Errorf("request %d resent: %+v; want only a refused request resent, at 64000", i, limits[i])
			}
		}
	}
	refused := 0
	for _, body := range srv.posts() {
		msgs, _ := body["messages"].([]any)
		first, _ := msgs[0].(map[string]any)
		if limit, _ := sentLimit(body); limit == 128000 && fmt.Sprint(first["content"]) == "refuse" {
			refused++
		}
	}
	if refused == 0 || resent != refused || len(srv.posts()) != 32+refused {
		t.Errorf("%d sends, %d refused, %d resent; want each refused request resent once (32 + refused sends)",
			len(srv.posts()), refused, resent)
	}
	// A Generate whose limit the SDK chose is read over the stream, so the cap
	// is kept under the streamed key (0088-MADR ST88, ST88e; DL38).
	if c, ok := memoIn(t, cache).Cap(limitmemo.Key{Provider: "claude", BaseURL: srv.URL, Model: model, Stream: true}); !ok || c != 64000 {
		t.Errorf("the memo: %d, %v; want one entry, 64000", c, ok)
	}
}

// TestLimitLearning_Default_0089 (OQ3 (1): DL49, DL73): by default a
// validated cap refusal of a limit the SDK chose is resent once at the cap,
// kept, and reported in a WarningLimitLearned; with WithoutLimitLearning()
// it is still resent, and neither kept nor reported.
func TestLimitLearning_Default_0089(t *testing.T) {
	ctx := context.Background()
	const model = "claude-opus-4-6" // the extract states 128,000
	refusal := messagesError("max_tokens: 128000 > 64000, which is the maximum allowed number of output tokens for " + model)
	for _, off := range []bool{false, true} {
		srv := newLimitServer(t, "", refuseFirst(refusal, wirecase.MessagesReply))
		opts := []llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithoutModelMetadata(),
			llmprovider.WithModelProbes(false), catalog.WithCache(new(catalog.Cache)), llmprovider.WithBaseURL(srv.URL),
			llmprovider.WithModel(model)}
		if off {
			opts = append(opts, llmprovider.WithoutLimitLearning())
		}
		p, err := New(llmprovider.ProviderClaude, opts...)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := p.Generate(ctx, hiRequest())
		posts := srv.posts()
		if err != nil || len(posts) != 2 {
			t.Fatalf("learning off %t: %d send(s), err %v; want a resend", off, len(posts), err)
		}
		if got, _ := sentLimit(posts[1]); got != 64000 {
			t.Errorf("learning off %t: resent at %v; want 64000, the stated cap", off, got)
		}
		var warned *llmprovider.Warning
		for i, w := range resp.Warnings {
			if w.Kind == llmprovider.WarningLimitLearned {
				warned = &resp.Warnings[i]
			}
		}
		// The Generate, whose limit the SDK chose, was read over the stream, so
		// a kept cap bounds the streamed output (0088-MADR ST88, ST88e; DL76).
		l := catalog.CachedLimits(ctx, llmprovider.ProviderClaude, model, opts...)
		kept := l.StreamedOutput.Tokens == 64000 && l.StreamedOutput.Evidence[0].Source == "observed"
		switch {
		case !off && (warned == nil || warned.Observed.Tokens != 64000 || warned.Observed.Model != model || !kept):
			t.Errorf("by default: warning %+v, streamed output %+v; want 64000 kept and reported", warned, l.StreamedOutput)
		case off && (warned != nil || kept || l.StreamedOutput.Tokens != 128000):
			t.Errorf("WithoutLimitLearning: warning %+v, streamed output %+v; want nothing kept or reported", warned,
				l.StreamedOutput)
		}
	}
}
