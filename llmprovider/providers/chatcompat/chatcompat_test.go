package chatcompat

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// refusedMax is the Chat Completions wire's refusal of max, in Hugging Face's
// recorded form (0062-REPORT-thinking-refusals.md E2).
const refusedMax = `{"error":{"code":"400","message":"Invalid request: [\"reasoning_effort (max): Input should be ` +
	`'minimal', 'low', 'medium', 'high' or 'none'\"]","type":"Bad Request"}}`

// TestNew_Refusals_0061 (0061-MADR D4; the amendment "Phase 4's options"):
// New refuses no WithID, an empty id and a built-in's id in any case with
// ErrInvalidProvider; no base URL with ErrInvalidRequest; an OAuth or vendor
// CLI session with ErrUnsupported; and a catalog option scoped to the
// built-ins with ErrInvalidRequest. It takes catalog.WithCache, and the last
// WithID.
func TestNew_Refusals_0061(t *testing.T) {
	base := llmprovider.WithBaseURL("http://127.0.0.1:1")
	for _, tc := range []struct {
		name string
		opts []llmprovider.Option
		want error
	}{
		{"no WithID", []llmprovider.Option{base}, llmprovider.ErrInvalidProvider},
		{"an empty id", []llmprovider.Option{WithID(" "), base}, llmprovider.ErrInvalidProvider},
		{"a built-in id", []llmprovider.Option{WithID("OpenAI"), base}, llmprovider.ErrInvalidProvider},
		{"no base URL", []llmprovider.Option{WithID(testID)}, llmprovider.ErrInvalidRequest},
		{"an OAuth session", []llmprovider.Option{WithID(testID), base,
			llmprovider.WithTokenSource(&auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "a"})}, llmprovider.ErrUnsupported},
		{"a vendor CLI session", []llmprovider.Option{WithID(testID), base,
			llmprovider.WithTokenSource(&auth.VendorCLISession{Provider: llmprovider.ProviderOpenAI, Path: "unused"})},
			llmprovider.ErrUnsupported},
		{"a built-ins' catalog option", []llmprovider.Option{WithID(testID), base, catalog.WithLimit(2)},
			llmprovider.ErrInvalidRequest},
	} {
		if _, err := New(tc.opts...); !errors.Is(err, tc.want) {
			t.Errorf("%s: %v; want %v", tc.name, err, tc.want)
		}
	}
	p, err := New(WithID("first"), WithID(testID), base, catalog.WithCache(new(catalog.Cache)))
	if err != nil || p.ID() != testID {
		t.Errorf("the last WithID, with WithCache: %v, %v; want %q", p, err, testID)
	}
}

// TestNoKey_NoAuthorization_0061 (0061-MADR D4): with no key, or an empty one,
// neither a generation nor the listing carries an Authorization header.
func TestNoKey_NoAuthorization_0061(t *testing.T) {
	srv := newServer(t, func(_ int, r *http.Request) (int, string) {
		if r.Method == http.MethodGet {
			return http.StatusOK, `{"data":[{"id":"m"}]}`
		}
		return http.StatusOK, fxChat
	})
	for _, opts := range [][]llmprovider.Option{nil, {llmprovider.WithAPIKey("")}} {
		p := build(t, srv.URL, opts...)
		if _, err := p.Generate(context.Background(), text("hi")); err != nil {
			t.Fatal(err)
		}
		if _, err := p.(llmprovider.ModelLister).ListModels(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range srv.seen() {
		if r.authorization != "" {
			t.Errorf("%s %s: Authorization %q; want none", r.method, r.path, r.authorization)
		}
	}
}

// TestKey_Bearer_0061 (0061-MADR D4; R16): a key goes in Authorization as a
// bearer token, on a generation and on the listing.
func TestKey_Bearer_0061(t *testing.T) {
	srv := newServer(t, func(_ int, r *http.Request) (int, string) {
		if r.Method == http.MethodGet {
			return http.StatusOK, `{"data":[{"id":"m"}]}`
		}
		return http.StatusOK, fxChat
	})
	p := build(t, srv.URL, llmprovider.WithAPIKey("sk-test"))
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatal(err)
	}
	if _, err := p.(llmprovider.ModelLister).ListModels(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, r := range srv.seen() {
		if r.authorization != "Bearer sk-test" {
			t.Errorf("%s %s: Authorization %q; want the bearer token", r.method, r.path, r.authorization)
		}
	}
}

// TestListModels_0061 (0061-MADR D4): ListModels is GET {base}/models, its
// ids in order, each once, with no probe by default; under
// WithModelProbes(true) the models are probed; a failed listing is its error.
func TestListModels_0061(t *testing.T) {
	listing := `{"object":"list","data":[{"id":"b"},{"id":"a"},{"id":"b"},{"id":""}]}`
	srv := newServer(t, func(_ int, r *http.Request) (int, string) {
		if r.Method == http.MethodGet {
			return http.StatusOK, listing
		}
		return http.StatusOK, fxChat
	})
	got, err := build(t, srv.URL).(llmprovider.ModelLister).ListModels(context.Background())
	if err != nil || !slices.Equal(got, []string{"b", "a"}) {
		t.Errorf("ListModels = %v, %v; want [b a]", got, err)
	}
	if reqs := srv.seen(); len(reqs) != 1 || reqs[0].method != http.MethodGet || reqs[0].path != "/models" {
		t.Errorf("requests %+v; want one GET /models, no probe", reqs)
	}
	if _, err := build(t, srv.URL, llmprovider.WithModelProbes(true)).(llmprovider.ModelLister).ListModels(
		context.Background()); err != nil {
		t.Fatal(err)
	}
	probes := 0
	for _, r := range srv.seen() {
		if r.method == http.MethodPost && r.path == "/chat/completions" {
			probes++
		}
	}
	if probes != 2 {
		t.Errorf("%d probe(s) under WithModelProbes(true); want one per listed model, 2", probes)
	}

	failing := newServer(t, func(int, *http.Request) (int, string) { return http.StatusServiceUnavailable, `{"error":"down"}` })
	if _, err := build(t, failing.URL).(llmprovider.ModelLister).ListModels(context.Background()); !errors.Is(err,
		llmprovider.ErrProviderUnavailable) {
		t.Errorf("a 503 listing: %v; want ErrProviderUnavailable", err)
	}
	garbled := newServer(t, func(int, *http.Request) (int, string) { return http.StatusOK, `{"data":[` })
	if _, err := build(t, garbled.URL).(llmprovider.ModelLister).ListModels(context.Background()); !errors.Is(err,
		llmprovider.ErrIncomplete) {
		t.Errorf("a garbled listing: %v; want ErrIncomplete", err)
	}
}

// TestNew_ForID_0061 (the amendment "Phase 4's options"): For(id, …) reaches
// the provider of that id, and For(another id, …) does not.
func TestNew_ForID_0061(t *testing.T) {
	srv := newServer(t, nil)
	p, err := New(WithID(testID), llmprovider.For(testID, llmprovider.WithBaseURL(srv.URL), llmprovider.WithModel("m")),
		testCache(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Generate(context.Background(), text("hi")); err != nil || len(srv.seen()) != 1 {
		t.Errorf("For(%q, WithBaseURL(…)): %v after %d request(s); want one", testID, err, len(srv.seen()))
	}
	if _, err := New(WithID(testID), llmprovider.For("other", llmprovider.WithBaseURL(srv.URL))); !errors.Is(err,
		llmprovider.ErrInvalidRequest) {
		t.Errorf("For(another id, WithBaseURL(…)): %v; want no base URL, ErrInvalidRequest", err)
	}
}

// efforts is reasoning_effort of each POST so far.
func efforts(s *server) []string {
	var out []string
	for _, r := range s.seen() {
		if r.method == http.MethodPost {
			e, _ := r.body["reasoning_effort"].(string)
			out = append(out, e)
		}
	}
	return out
}

// atMax is a request at EffortMax.
func atMax() *llmprovider.Request {
	r := text("hi")
	r.Reasoning = &llmprovider.Reasoning{Effort: llmprovider.EffortMax}
	return r
}

// TestLearn_Effort_0061 (0062-MADR D2 to D6; this PLAN's deviation of
// 2026-10-25): max refused in a recorded Chat Completions form is learned and
// sent again at the next effort, warned; the next request sends it first; a
// refusal passed back with WithReasoningRefusals is never sent.
func TestLearn_Effort_0061(t *testing.T) {
	srv := newServer(t, func(n int, _ *http.Request) (int, string) {
		if n == 1 {
			return http.StatusBadRequest, refusedMax
		}
		return http.StatusOK, fxChat
	})
	p := build(t, srv.URL)
	resp, err := p.Generate(context.Background(), atMax())
	if err != nil {
		t.Fatal(err)
	}
	refused := llmprovider.ReasoningRefusal{Model: "m", Effort: llmprovider.EffortMax}
	if !slices.ContainsFunc(resp.Warnings, func(w llmprovider.Warning) bool {
		return w.Kind == llmprovider.WarningReasoningLearned && w.Refused == refused
	}) {
		t.Errorf("warnings %v; want a WarningReasoningLearned of %+v", resp.Warnings, refused)
	}
	if _, err := p.Generate(context.Background(), atMax()); err != nil {
		t.Fatal(err)
	}
	if got := efforts(srv); !slices.Equal(got, []string{"max", "high", "high"}) {
		t.Errorf("efforts sent %v; want max, then high, then high first", got)
	}
	passed := newServer(t, nil)
	if _, err := build(t, passed.URL, llmprovider.WithReasoningRefusals(refused)).Generate(context.Background(),
		atMax()); err != nil || !slices.Equal(efforts(passed), []string{"xhigh"}) {
		t.Errorf("max passed back: %v after %v; want xhigh first", err, efforts(passed))
	}
}

// TestCache_Shared_0061 (the amendment "Phase 4's options"; R49): two
// providers given one catalog.Cache share a learned refusal; a provider with
// a cache of its own does not see it.
func TestCache_Shared_0061(t *testing.T) {
	srv := newServer(t, func(n int, _ *http.Request) (int, string) {
		if n == 1 {
			return http.StatusBadRequest, refusedMax
		}
		return http.StatusOK, fxChat
	})
	shared := catalog.WithCache(new(catalog.Cache))
	if _, err := build(t, srv.URL, shared).Generate(context.Background(), atMax()); err != nil {
		t.Fatal(err)
	}
	if _, err := build(t, srv.URL, shared).Generate(context.Background(), atMax()); err != nil {
		t.Fatal(err)
	}
	if _, err := build(t, srv.URL, catalog.WithCache(new(catalog.Cache))).Generate(context.Background(), atMax()); err != nil {
		t.Fatal(err)
	}
	if got := efforts(srv); !slices.Equal(got, []string{"max", "high", "high", "max"}) {
		t.Errorf("efforts sent %v; want max then high (learned), high (the shared cache), max (another cache)", got)
	}
}

// TestDescriptor_0061 (0061-MADR D4): the descriptor takes the caller's id and
// label, a base URL the caller sets, and no key required.
func TestDescriptor_0061(t *testing.T) {
	d := Descriptor(testID, "Acme")
	if d.ID != testID || d.Label != "Acme" || !d.SupportsBaseURL || d.RequiresAPIKey || d.DefaultBaseURL != "" ||
		d.EnvVar != "" || len(d.AuthMethods) != 0 || len(d.StaticModels) != 0 {
		t.Errorf("Descriptor = %+v", d)
	}
	if !strings.EqualFold(string(scope), "chatcompat") {
		t.Errorf("scope = %q", scope)
	}
}
