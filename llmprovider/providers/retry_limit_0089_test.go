package providers

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// 0089-PLAN Phase 6b's retry test (0089-MADR DL88), and the helpers Phase
// 6b's tests share that read no APIError.Observed, so that this file builds
// at the phase's base, where the test is red on its assertion.

// opus0089 is a Claude model the extract states 128,000 for.
const opus0089 = "claude-opus-4-6"

// overloaded0089 is a Messages 503 that names no wait.
const overloaded0089 = `{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`

// opusRefusal is Claude's refusal of sent, stating capped for opus0089.
func opusRefusal(sent, capped int) string {
	return messagesError(fmt.Sprintf("max_tokens: %d > %d, which is the maximum allowed number of output tokens "+
		"for %s", sent, capped, opus0089))
}

// limitOpts are a provider's options against url for model, with its own
// cache and the extract fixture (0093-MADR D12, MH55).
func limitOpts(url, model string, cache *catalog.Cache, extra ...llmprovider.Option) []llmprovider.Option {
	return append([]llmprovider.Option{llmprovider.WithAPIKey("k"), extractDoc(),
		llmprovider.WithModelProbes(false), catalog.WithCache(cache), llmprovider.WithBaseURL(url),
		llmprovider.WithModel(model)}, extra...)
}

// newOn is New(id, opts...), and fails t on an error.
func newOn(t *testing.T, id llmprovider.ProviderID, opts []llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := New(id, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// callFor runs req on p once: Generate, or llmprovider.Stream ranged whole.
// It returns the final response, the count of error events and the error.
func callFor(ctx context.Context, p llmprovider.Provider, req *llmprovider.Request, stream bool) (*llmprovider.Response,
	int, error) {
	if !stream {
		resp, err := p.Generate(ctx, req)
		if err != nil {
			return resp, 1, err
		}
		return resp, 0, nil
	}
	var resp *llmprovider.Response
	var err error
	failures := 0
	for ev, e := range llmprovider.Stream(ctx, p, req) {
		if e != nil {
			err = e
			failures++
			continue
		}
		if ev.Type == llmprovider.EventDone {
			resp = ev.Response
		}
	}
	return resp, failures, err
}

// TestRetry_LimitObservedCarried_0089 (DL88): the SDK's own limit refused,
// the cap kept, the resend answered 503, and the retried attempt, which sends
// the kept cap, answered: the final response carries a WarningLimitLearned
// with the kept cap, under WithRetry and WithStreamRetry alike, unless one
// for the same model, base URL and scope is there.
func TestRetry_LimitObservedCarried_0089(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		stream bool
		// replies are the answers to the POSTs in order, the last repeated.
		replies []func() (int, string)
		limits  []int
		learned int // the one WarningLimitLearned's tokens
		sent    llmprovider.SentLimit
	}{
		{"generate under WithRetry", false,
			[]func() (int, string){
				func() (int, string) { return http.StatusBadRequest, opusRefusal(128000, 64000) },
				func() (int, string) { return http.StatusServiceUnavailable, overloaded0089 },
				func() (int, string) { return http.StatusOK, wirecase.MessagesReply }},
			[]int{128000, 64000, 64000}, 64000,
			llmprovider.SentLimit{Tokens: 64000, Sent: true, Basis: llmprovider.LimitBasisFact, Source: "observed"}},
		{"stream under WithStreamRetry", true,
			[]func() (int, string){
				func() (int, string) { return http.StatusBadRequest, opusRefusal(128000, 64000) },
				func() (int, string) { return http.StatusServiceUnavailable, overloaded0089 },
				func() (int, string) { return http.StatusOK, wirecase.MessagesReply }},
			[]int{128000, 64000, 64000}, 64000,
			llmprovider.SentLimit{Tokens: 64000, Sent: true, Basis: llmprovider.LimitBasisFact, Source: "observed"}},
		{"a warning for the same key already there", false,
			[]func() (int, string){
				func() (int, string) { return http.StatusBadRequest, opusRefusal(128000, 64000) },
				func() (int, string) { return http.StatusServiceUnavailable, overloaded0089 },
				func() (int, string) { return http.StatusBadRequest, opusRefusal(64000, 32000) },
				func() (int, string) { return http.StatusOK, wirecase.MessagesReply }},
			[]int{128000, 64000, 64000, 32000}, 32000,
			llmprovider.SentLimit{Tokens: 32000, Sent: true, Basis: llmprovider.LimitBasisRefusal,
				Source: "refusal: anthropic max_tokens"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := newLimitServer(t, "", func(n int, _ map[string]any) (int, string) {
				return tc.replies[min(n, len(tc.replies)-1)]()
			})
			p := newOn(t, llmprovider.ProviderClaude, limitOpts(srv.URL, opus0089, new(catalog.Cache)))
			policy := llmprovider.RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond}
			if tc.stream {
				p = llmprovider.WithStreamRetry(p, policy, nil)
			} else {
				p = llmprovider.WithRetry(p, policy)
			}
			resp, _, err := callFor(ctx, p, hiRequest(), tc.stream)
			if err != nil || resp == nil {
				t.Fatalf("err %v; want the retried attempt's response", err)
			}
			var limits []int
			for _, body := range srv.posts() {
				v, _ := sentLimit(body)
				limits = append(limits, int(v))
			}
			if !reflect.DeepEqual(limits, tc.limits) {
				t.Errorf("sent %v; want %v", limits, tc.limits)
			}
			if resp.OutputLimit != tc.sent {
				t.Errorf("OutputLimit = %+v; want %+v", resp.OutputLimit, tc.sent)
			}
			var learned []llmprovider.Warning
			for _, w := range resp.Warnings {
				if w.Kind == llmprovider.WarningLimitLearned {
					learned = append(learned, w)
				}
			}
			want := llmprovider.LimitObservation{Model: opus0089, BaseURL: srv.URL, Tokens: tc.learned}
			if len(learned) != 1 {
				t.Fatalf("%d WarningLimitLearned (%+v); want one, with %+v", len(learned), learned, want)
			}
			got := learned[0].Observed
			got.Seen = time.Time{}
			if got != want || learned[0].Model != opus0089 || learned[0].Observed.Seen.IsZero() {
				t.Errorf("WarningLimitLearned %+v; want Model %s, Observed %+v with a Seen", learned[0], opus0089, want)
			}
		})
	}
}
