package providers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/limitmemo"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// 0089-PLAN Phase 6b's scope addition (0089-MADR DL95, OQ6 (1)): Kilo's
// range form, from the body measured on 2026-11-24. Its red at the phase's
// base was taken before its APIError.Observed assertion was added, which
// the base cannot build (0089-PLAN, Phase 6b's entries).

// kiloRangeModel0089 is the model the form was measured on; Kilo's listing
// states 131,072 for it.
const kiloRangeModel0089 = "z-ai/glm-5.3-flashx"

// kiloRangeBody is Kilo's refusal in its range form, as measured: a body of
// error only, with message, type and param. param's values were not
// recorded, so they are null here.
func kiloRangeBody(lower, upper int) string {
	return fmt.Sprintf(`{"error":{"message":"The max_tokens parameter is illegal.：限制数值范围[%d,%d]",`+
		`"type":"AI_APICallError","param":{"error":null,"statusCode":null,"name":null,"message":null,`+
		`"isRetryable":null,"type":null}}}`, lower, upper)
}

// TestLimitRefusal_KiloRangeForm_0089 (DL95): Kilo's range form, validated
// as OQ6 (1) says: a limit the SDK chose is resent once at the stated upper
// bound, which is kept and reported; a caller's limit is returned, its cap
// kept; a lower bound other than 1, or an upper bound not below the value
// sent (V3), is not read.
func TestLimitRefusal_KiloRangeForm_0089(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name         string
		stated       int // a layer's output for the model; 0 for none
		request      int // the request's own limit; 0 for none
		lower, upper int // the range the refusal states
		limits       []int
		kept         int  // the memo's cap after the call; 0 for none
		stream       bool // the key the cap is kept under
	}{
		{"a limit the SDK chose", 200000, 0, 1, 131072, []int{200000, 131072}, 131072, true},
		{"a caller's limit", 0, 200000, 1, 131072, []int{200000}, 131072, false},
		{"a lower bound of 2", 200000, 0, 2, 131072, []int{200000}, 0, true},
		{"an upper bound not below the value sent (V3)", 100000, 0, 1, 131072, []int{100000}, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := newLimitServer(t, "", func(_ int, body map[string]any) (int, string) {
				if limit, _ := sentLimit(body); int(limit) == tc.limits[0] {
					return http.StatusBadRequest, kiloRangeBody(tc.lower, tc.upper)
				}
				return http.StatusOK, wirecase.ChatReply
			})
			var extra []llmprovider.Option
			if tc.stated > 0 {
				extra = append(extra, catalog.WithModelLayer(limitLayer{kiloRangeModel0089, tc.stated}))
			}
			cache := new(catalog.Cache)
			p := newOn(t, llmprovider.ProviderKilo, limitOpts(srv.URL, kiloRangeModel0089, cache, extra...))
			req := hiRequest()
			req.MaxOutputTokens = tc.request
			from := time.Now()
			resp, err := p.Generate(ctx, req)
			to := time.Now()
			var limits []int
			for _, body := range srv.posts() {
				v, _ := sentLimit(body)
				limits = append(limits, int(v))
			}
			if !reflect.DeepEqual(limits, tc.limits) {
				t.Fatalf("sent %v, err %v; want %v", limits, err, tc.limits)
			}
			if len(tc.limits) == 1 && !errors.Is(err, llmprovider.ErrInvalidRequest) {
				t.Errorf("err %v; want the refusal returned, ErrInvalidRequest", err)
			}
			if len(tc.limits) == 2 {
				checkKiloRangeResend(t, resp, err)
			}
			k := limitmemo.Key{Provider: string(llmprovider.ProviderKilo), BaseURL: srv.URL, Model: kiloRangeModel0089,
				Stream: tc.stream}
			if c, _ := memoIn(t, cache).Cap(k); c != tc.kept {
				t.Errorf("the memo holds %d; want %d", c, tc.kept)
			}
			if len(tc.limits) == 1 {
				// A refused call reports the cap it kept, or nothing (DL95, DL78).
				want := llmprovider.LimitObservation{}
				if tc.kept > 0 {
					want = llmprovider.LimitObservation{Model: kiloRangeModel0089, BaseURL: srv.URL, Tokens: tc.kept}
				}
				wantObserved(t, tc.name, err, want, from, to)
			}
		})
	}
}

// checkKiloRangeResend checks a resend at the range's upper bound: answered,
// reported as basis refusal, and the cap reported once.
func checkKiloRangeResend(t *testing.T, resp *llmprovider.Response, err error) {
	t.Helper()
	if err != nil || resp == nil {
		t.Fatalf("err %v; want the resend answered", err)
	}
	want := llmprovider.SentLimit{Tokens: 131072, Sent: true, Basis: llmprovider.LimitBasisRefusal,
		Source: "refusal: kilo max_tokens range"}
	if resp.OutputLimit != want {
		t.Errorf("OutputLimit = %+v; want %+v", resp.OutputLimit, want)
	}
	learned := 0
	for _, w := range resp.Warnings {
		if w.Kind == llmprovider.WarningLimitLearned && w.Observed.Tokens == 131072 && w.Observed.Model == kiloRangeModel0089 {
			learned++
		}
	}
	if learned != 1 {
		t.Errorf("warnings %+v; want one WarningLimitLearned with 131072", resp.Warnings)
	}
}
