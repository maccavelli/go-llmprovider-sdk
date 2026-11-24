package providers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/cacheref"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/limitmemo"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/claude"
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

// 0089-PLAN Phase 6b's tests (0089-MADR, second amendment of 2026-11-24,
// DL78 to DL91): a cap the service stated when it refused a call's output
// limit, which the provider kept, reaches the caller on the error that ends
// the call, APIError.Observed; WithRetry and WithStreamRetry carry a retried
// attempt's to the final result. Every server is the test's own.

// callerModel0089 is a Claude model no source states.
const callerModel0089 = "claude-caller-0089"

// callerRefusal is Claude's refusal of sent, stating 128,000 for named.
func callerRefusal(sent int, named string) string {
	return messagesError(fmt.Sprintf("max_tokens: %d > 128000, which is the maximum allowed number of output "+
		"tokens for %s", sent, named))
}

// refuseAt answers a POST whose limit is sent with a 400 carrying refusal,
// and any other with ok.
func refuseAt(sent int, refusal, ok string) func(int, map[string]any) (int, string) {
	return func(_ int, body map[string]any) (int, string) {
		if limit, _ := sentLimit(body); int(limit) == sent {
			return http.StatusBadRequest, refusal
		}
		return http.StatusOK, ok
	}
}

// limitRequest is hiRequest naming limit as its own MaxOutputTokens.
func limitRequest(limit int) *llmprovider.Request {
	req := hiRequest()
	req.MaxOutputTokens = limit
	return req
}

// wantObserved checks that err wraps an *APIError whose Observed is want,
// its Seen between from and to, or zero when want is.
func wantObserved(t *testing.T, label string, err error, want llmprovider.LimitObservation, from, to time.Time) {
	t.Helper()
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	if !ok {
		t.Errorf("%s: err %v (%T); want an *APIError", label, err, err)
		return
	}
	got := apiErr.Observed
	seen := got.Seen
	got.Seen = time.Time{}
	if got != want {
		t.Errorf("%s: Observed = %+v; want %+v", label, got, want)
	}
	switch {
	case want.Tokens > 0 && (seen.Before(from) || seen.After(to)):
		t.Errorf("%s: Observed.Seen %v; want within the call, %v to %v (DL90)", label, seen, from, to)
	case want.Tokens == 0 && !seen.IsZero():
		t.Errorf("%s: Observed.Seen %v; want zero", label, seen)
	}
}

// memoCap is the memo's cap for model on the Claude route at url.
func memoCap(t *testing.T, cache *catalog.Cache, url, model string, stream bool) int {
	t.Helper()
	c, _ := memoIn(t, cache).Cap(limitmemo.Key{Provider: string(llmprovider.ProviderClaude), BaseURL: url, Model: model,
		Stream: stream})
	return c
}

// callerRefusedText0089 is the refusal's text at 33e6f09, which DL91 keeps.
const callerRefusedText0089 = "llmprovider: invalid request: claude HTTP 400 invalid_request_error: max_tokens: 200000 > " +
	"128000, which is the maximum allowed number of output tokens for claude-caller-0089"

// callerServer is a Claude server refusing 200,000 for callerModel0089.
func callerServer(t *testing.T) *limitServer {
	t.Helper()
	return newLimitServer(t, "", refuseAt(200000, callerRefusal(200000, callerModel0089), wirecase.MessagesReply))
}

// TestLimitRefusal_CallerReport_0089 (DL78 to DL84, DL88's caller case,
// DL89 to DL91): a caller's limit refused in a cap form, validated and kept,
// is sent once and returned, and the returned *APIError's Observed is the
// kept cap, as a WarningLimitLearned carries it; nothing kept, nothing
// reported; the error's text is 33e6f09's.
func TestLimitRefusal_CallerReport_0089(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprintf("the request's own limit, stream %t", stream), func(t *testing.T) { callerOwnLimit(t, stream) })
		t.Run(fmt.Sprintf("WithMaxTokens, stream %t", stream), func(t *testing.T) { callerOption(t, stream) })
		t.Run(fmt.Sprintf("under a retry wrapper, stream %t", stream), func(t *testing.T) { callerUnderRetry(t, stream) })
	}
	t.Run("kilo, the first payload inside a 200", callerKiloFirstPayload)
	t.Run("the same value refused again", callerRefusedAgain)
	t.Run("WithoutLimitLearning", callerLearningOff)
	t.Run("a refusal naming another model (V6)", callerOtherModel)
	t.Run("a window form", callerWindowForm)
	t.Run("passed back, with a workspace", callerPassedBack)
}

// callerOwnLimit is DL78 (Generate) and DL79 (Stream, refused before any
// event), with DL91's text.
func callerOwnLimit(t *testing.T, stream bool) {
	srv := callerServer(t)
	cache := new(catalog.Cache)
	p := newOn(t, llmprovider.ProviderClaude, limitOpts(srv.URL, callerModel0089, cache))
	from := time.Now()
	_, failures, err := callFor(context.Background(), p, limitRequest(200000), stream)
	to := time.Now()
	wantObserved(t, "refused", err, llmprovider.LimitObservation{Model: callerModel0089, BaseURL: srv.URL,
		Tokens: 128000}, from, to)
	if n := len(srv.posts()); n != 1 || failures != 1 || !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("%d send(s), %d error(s), err %v; want one send, one error, ErrInvalidRequest", n, failures, err)
	}
	if err == nil || err.Error() != callerRefusedText0089 {
		t.Errorf("text %q; want %q, 33e6f09's (DL91)", fmt.Sprint(err), callerRefusedText0089)
	}
	if c := memoCap(t, cache, srv.URL, callerModel0089, stream); c != 128000 {
		t.Errorf("the memo holds %d under the call's own key; want 128000", c)
	}
}

// callerOption is DL81: WithMaxTokens' value, refused.
func callerOption(t *testing.T, stream bool) {
	srv := newLimitServer(t, "", refuseAt(150000, callerRefusal(150000, callerModel0089), wirecase.MessagesReply))
	p := newOn(t, llmprovider.ProviderClaude, limitOpts(srv.URL, callerModel0089, new(catalog.Cache),
		llmprovider.WithMaxTokens(150000)))
	from := time.Now()
	_, _, err := callFor(context.Background(), p, hiRequest(), stream)
	wantObserved(t, "option refused", err, llmprovider.LimitObservation{Model: callerModel0089, BaseURL: srv.URL,
		Tokens: 128000}, from, time.Now())
	if n := len(srv.posts()); n != 1 {
		t.Errorf("%d send(s); want one: a caller's value is not resent", n)
	}
}

// callerUnderRetry is DL88's caller case: a refused caller value is not
// retried, and is returned once with Observed.
func callerUnderRetry(t *testing.T, stream bool) {
	srv := callerServer(t)
	p := newOn(t, llmprovider.ProviderClaude, limitOpts(srv.URL, callerModel0089, new(catalog.Cache)))
	policy := llmprovider.RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond}
	if stream {
		p = llmprovider.WithStreamRetry(p, policy, nil)
	} else {
		p = llmprovider.WithRetry(p, policy)
	}
	from := time.Now()
	_, failures, err := callFor(context.Background(), p, limitRequest(200000), stream)
	wantObserved(t, "refused under retry", err, llmprovider.LimitObservation{Model: callerModel0089,
		BaseURL: srv.URL, Tokens: 128000}, from, time.Now())
	if n := len(srv.posts()); n != 1 || failures != 1 {
		t.Errorf("%d send(s), %d error(s); want one of each: ErrInvalidRequest is not retried", n, failures)
	}
}

// callerKiloFirstPayload is DL80: Kilo's refusal as its stream's first
// payload inside a 200: the *APIError is found through the transport's
// wrapper, whose type and text stay those of the same call with learning off.
func callerKiloFirstPayload(t *testing.T) {
	const model = "anthropic/" + callerModel0089
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `data: {"error":{"code":400,"message":"max_tokens: 200000 > 128000, which is `+
			`the maximum allowed number of output tokens for `+callerModel0089+`"}}`+"\n\n")
	}))
	t.Cleanup(srv.Close)
	p := newOn(t, llmprovider.ProviderKilo, limitOpts(srv.URL, model, new(catalog.Cache)))
	from := time.Now()
	_, _, errOn := callFor(context.Background(), p, limitRequest(200000), true)
	to := time.Now()
	off := newOn(t, llmprovider.ProviderKilo, limitOpts(srv.URL, model, new(catalog.Cache),
		llmprovider.WithoutLimitLearning()))
	_, _, errOff := callFor(context.Background(), off, limitRequest(200000), true)
	if errOn == nil || fmt.Sprintf("%T", errOn) == fmt.Sprintf("%T", &llmprovider.APIError{}) {
		t.Fatalf("err %v (%T); want the transport's wrapper of an *APIError (0088-MADR A5)", errOn, errOn)
	}
	wantObserved(t, "kilo", errOn, llmprovider.LimitObservation{Model: model, BaseURL: srv.URL, Tokens: 128000},
		from, to)
	wantObserved(t, "kilo, learning off", errOff, llmprovider.LimitObservation{}, from, to)
	if fmt.Sprintf("%T %v", errOn, errOn) != fmt.Sprintf("%T %v", errOff, errOff) {
		t.Errorf("err %T %q; want the type and text learning off gives, %T %q", errOn, errOn, errOff, errOff)
	}
}

// callerRefusedAgain is DL82: the same value refused again, the memo
// unchanged: each refused call carries Observed.
func callerRefusedAgain(t *testing.T) {
	srv := callerServer(t)
	p := newOn(t, llmprovider.ProviderClaude, limitOpts(srv.URL, callerModel0089, new(catalog.Cache)))
	for i := range 2 {
		from := time.Now()
		_, err := p.Generate(context.Background(), limitRequest(200000))
		wantObserved(t, fmt.Sprintf("call %d", i+1), err, llmprovider.LimitObservation{Model: callerModel0089,
			BaseURL: srv.URL, Tokens: 128000}, from, time.Now())
	}
	if n := len(srv.posts()); n != 2 {
		t.Errorf("%d send(s); want one per call", n)
	}
}

// callerLearningOff is DL83: WithoutLimitLearning keeps nothing and reports
// nothing; the refusal is returned as at 33e6f09.
func callerLearningOff(t *testing.T) {
	srv := callerServer(t)
	cache := new(catalog.Cache)
	p := newOn(t, llmprovider.ProviderClaude, limitOpts(srv.URL, callerModel0089, cache,
		llmprovider.WithoutLimitLearning()))
	from := time.Now()
	_, err := p.Generate(context.Background(), limitRequest(200000))
	wantObserved(t, "learning off", err, llmprovider.LimitObservation{}, from, time.Now())
	if err == nil || err.Error() != callerRefusedText0089 {
		t.Errorf("text %q; want %q", fmt.Sprint(err), callerRefusedText0089)
	}
	if c := memoCap(t, cache, srv.URL, callerModel0089, false); c != 0 {
		t.Errorf("the memo holds %d; want nothing kept", c)
	}
}

// callerOtherModel is DL84: a refusal whose named model fails V6 is not
// kept, and reports nothing.
func callerOtherModel(t *testing.T) {
	srv := newLimitServer(t, "", refuseAt(200000, callerRefusal(200000, "claude-other-0089"), wirecase.MessagesReply))
	cache := new(catalog.Cache)
	p := newOn(t, llmprovider.ProviderClaude, limitOpts(srv.URL, callerModel0089, cache))
	from := time.Now()
	_, err := p.Generate(context.Background(), limitRequest(200000))
	wantObserved(t, "V6", err, llmprovider.LimitObservation{}, from, time.Now())
	if c := memoCap(t, cache, srv.URL, callerModel0089, false); c != 0 {
		t.Errorf("the memo holds %d; want nothing kept", c)
	}
}

// callerWindowForm is DL84: a window form keeps nothing, and reports
// nothing.
func callerWindowForm(t *testing.T) {
	srv := newLimitServer(t, "", refuseFirst(chatError("This model's maximum context length is 32768 tokens. "+
		"However, you requested 40000 tokens (23616 in the messages, 16384 in the completion). Please reduce the "+
		"length of the messages or completion."), wirecase.ChatReply))
	p := newOn(t, llmprovider.ProviderTogether, limitOpts(srv.URL, "openai/gpt-oss-120b", new(catalog.Cache),
		llmprovider.WithMaxTokens(16384)))
	from := time.Now()
	_, err := p.Generate(context.Background(), hiRequest())
	wantObserved(t, "window form", err, llmprovider.LimitObservation{}, from, time.Now())
}

// callerPassedBack is DL89 and DL90: Claude with a workspace: the error's
// Observed names it, and passed back to a new provider with
// WithLimitObservations it is taken (New refuses a zero Seen, DL42) and read
// as "remembered" for the workspace's requests.
func callerPassedBack(t *testing.T) {
	const workspace = "wrkspc_0089"
	ctx := context.Background()
	srv := callerServer(t)
	p := newOn(t, llmprovider.ProviderClaude, limitOpts(srv.URL, callerModel0089, new(catalog.Cache),
		claude.WithWorkspace(workspace)))
	from := time.Now()
	_, err := p.Generate(ctx, limitRequest(200000))
	wantObserved(t, "refused in a workspace", err, llmprovider.LimitObservation{Model: callerModel0089,
		BaseURL: srv.URL, Scope: workspace, Tokens: 128000}, from, time.Now())
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	if !ok {
		t.FailNow()
	}
	again, err := New(llmprovider.ProviderClaude, limitOpts(srv.URL, callerModel0089, new(catalog.Cache),
		claude.WithWorkspace(workspace), llmprovider.WithLimitObservations(apiErr.Observed))...)
	if err != nil {
		t.Fatalf("New refused the error's Observed passed back: %v", err)
	}
	resp, err := again.Generate(ctx, hiRequest())
	if err != nil {
		t.Fatal(err)
	}
	want := llmprovider.SentLimit{Tokens: 128000, Sent: true, Basis: llmprovider.LimitBasisFact, Source: "remembered"}
	if resp.OutputLimit != want {
		t.Errorf("OutputLimit = %+v; want %+v, the passed-back cap", resp.OutputLimit, want)
	}
}

// TestLimitRefusal_ResendFails_0089 (DL86, DL87, and DL88's final error): the
// SDK's own limit refused in a cap form, the cap kept, and the resend at it
// failing: an *APIError carries Observed; any other error is returned as it
// came, and the cap stays in the memo and bounds the next request.
func TestLimitRefusal_ResendFails_0089(t *testing.T) {
	ctx := context.Background()
	refused := opusRefusal(128000, 64000)
	t.Run("the resend answered 503", func(t *testing.T) {
		srv := newLimitServer(t, "", func(n int, _ map[string]any) (int, string) {
			if n == 0 {
				return http.StatusBadRequest, refused
			}
			return http.StatusServiceUnavailable, overloaded0089
		})
		p := newOn(t, llmprovider.ProviderClaude, limitOpts(srv.URL, opus0089, new(catalog.Cache)))
		from := time.Now()
		_, err := p.Generate(ctx, hiRequest())
		wantObserved(t, "503", err, llmprovider.LimitObservation{Model: opus0089, BaseURL: srv.URL, Tokens: 64000},
			from, time.Now())
		if apiErr, ok := errors.AsType[*llmprovider.APIError](err); !ok || apiErr.Status != http.StatusServiceUnavailable ||
			len(srv.posts()) != 2 {
			t.Errorf("%d send(s), err %v; want the resend's 503", len(srv.posts()), err)
		}
	})
	t.Run("the resend's connection closed", func(t *testing.T) {
		srv := newLimitServer(t, "", func(_ int, body map[string]any) (int, string) {
			if limit, _ := sentLimit(body); limit == 128000 {
				return http.StatusBadRequest, refused
			}
			msgs, _ := body["messages"].([]any)
			first, _ := msgs[0].(map[string]any)
			if fmt.Sprint(first["content"]) == "cut" {
				panic(http.ErrAbortHandler)
			}
			return http.StatusOK, wirecase.MessagesReply
		})
		cut := &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "cut"}}}
		cache := new(catalog.Cache)
		opts := limitOpts(srv.URL, opus0089, cache)
		p := newOn(t, llmprovider.ProviderClaude, opts)
		_, errOn := p.Generate(ctx, cut)
		off := newOn(t, llmprovider.ProviderClaude, limitOpts(srv.URL, opus0089, new(catalog.Cache),
			llmprovider.WithoutLimitLearning()))
		_, errOff := off.Generate(ctx, cut)
		if _, ok := errors.AsType[*llmprovider.APIError](errOn); ok || errOn == nil {
			t.Fatalf("err %v (%T); want a failure that is no *APIError", errOn, errOn)
		}
		if fmt.Sprintf("%T %v", errOn, errOn) != fmt.Sprintf("%T %v", errOff, errOff) {
			t.Errorf("err %T %q; want the type and text learning off gives, %T %q (R24)", errOn, errOn, errOff, errOff)
		}
		if c := memoCap(t, cache, srv.URL, opus0089, true); c != 64000 {
			t.Errorf("the memo holds %d; want 64000 kept", c)
		}
		l := catalog.CachedLimits(ctx, llmprovider.ProviderClaude, opus0089, opts...)
		if l.StreamedOutput.Tokens != 64000 || l.StreamedOutput.Evidence[0].Source != "observed" {
			t.Errorf("streamed output %+v; want 64000, observed", l.StreamedOutput)
		}
		resp, err := p.Generate(ctx, hiRequest())
		want := llmprovider.SentLimit{Tokens: 64000, Sent: true, Basis: llmprovider.LimitBasisFact, Source: "observed"}
		if err != nil {
			t.Fatalf("the next request: %v", err)
		}
		if resp.OutputLimit != want {
			t.Errorf("the next request's OutputLimit = %+v; want %+v, the kept cap", resp.OutputLimit, want)
		}
	})
	// DL88: every attempt failing, the final *APIError's Observed, zero on
	// the last attempt's own error, is the one a retried attempt carried.
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprintf("every attempt failing under a retry wrapper, stream %t", stream), func(t *testing.T) {
			srv := newLimitServer(t, "", func(n int, _ map[string]any) (int, string) {
				if n == 0 {
					return http.StatusBadRequest, refused
				}
				return http.StatusServiceUnavailable, overloaded0089
			})
			p := newOn(t, llmprovider.ProviderClaude, limitOpts(srv.URL, opus0089, new(catalog.Cache)))
			policy := llmprovider.RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond}
			if stream {
				p = llmprovider.WithStreamRetry(p, policy, nil)
			} else {
				p = llmprovider.WithRetry(p, policy)
			}
			from := time.Now()
			_, _, err := callFor(ctx, p, hiRequest(), stream)
			wantObserved(t, "the final error", err, llmprovider.LimitObservation{Model: opus0089, BaseURL: srv.URL,
				Tokens: 64000}, from, time.Now())
			if n := len(srv.posts()); n != 4 {
				t.Errorf("%d send(s); want 4: the refusal, its resend, two retries", n)
			}
		})
	}
}
