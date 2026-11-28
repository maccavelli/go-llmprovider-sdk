package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/limitmemo"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// 0096-PLAN Phase 2's tests: on Together, Hugging Face and OpenCode's chat
// route, a request that names no limit carries the model's chosen output
// fact, as Kilo's does (0096-MADR TG1 to TG14, HF1 to HF9, OC1 to OC9).
// Every server is the test's own; no request leaves the machine.

// fitMargin0096 is the margin the room for the output keeps below the
// window (0088-MADR ST87's 4,096, wire's fitMargin).
const fitMargin0096 = 4096

// docLayer is a "models.dev" layer stating output and, when context is
// above 0, the window, for model in section: registry evidence (0089-MADR
// DL15).
func docLayer(t *testing.T, section, model string, output, context int) llmprovider.Option {
	t.Helper()
	return catalog.WithModelLayer(docMeta(t, section, model, output, context))
}

// docMeta is a models.dev document stating output and, when context is above
// 0, the window, for model in section.
func docMeta(t *testing.T, section, model string, output, context int) catalog.Metadata {
	t.Helper()
	limit := `"output":` + itoa(output)
	if context > 0 {
		limit += `,"context":` + itoa(context)
	}
	meta, err := catalog.ParseMetadata(strings.NewReader(`{"` + section + `":{"models":{"` + model +
		`":{"limit":{` + limit + `}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	return meta
}

// hfListing0096 is a Hugging Face listing naming model with one live
// provider whose window is 1,000,000: once cached, the hold HF4 decides is
// known, so the stated output is sent (0096-MADR HF10, HF8a).
func hfListing0096(model string) string {
	return `{"data":[{"id":"` + model + `","architecture":{"input_modalities":["text"],"output_modalities":["text"]},` +
		`"providers":[{"provider":"a","status":"live","context_length":1000000}]}]}`
}

// room is what the SDK sends for an output above the room window leaves
// req's estimated input (0088-MADR ST87).
func room(window int, req *llmprovider.Request) int {
	return window - wire.EstimateInput(req) - fitMargin0096
}

// wire0096 is one of the three wires this record changes.
type wire0096 struct {
	name    string
	id      llmprovider.ProviderID // the memo's provider
	section string                 // the models.dev section
	model   string
	build   func(base string, opts ...llmprovider.Option) (llmprovider.Provider, error)
	// listing is what the wire's server answers a GET with, read once before
	// the request; empty for none. Hugging Face's states a live window, so
	// that its stated output is sent (HF10, HF8a).
	listing string
}

// wires0096 are Together, Hugging Face, and OpenCode's chat route on Zen and
// on Go.
func wires0096() []wire0096 {
	return []wire0096{
		{"together", llmprovider.ProviderTogether, "togetherai", "Qwen/Qwen4-Max", byID(llmprovider.ProviderTogether), ""},
		{"huggingface", llmprovider.ProviderHuggingFace, "huggingface", "org/hf-0096", byID(llmprovider.ProviderHuggingFace),
			hfListing0096("org/hf-0096")},
		{"opencode-zen chat", llmprovider.ProviderOpencodeZen, "opencode", "glm-0096", zenOn(opencode.RouteChatCompletions), ""},
		{"opencode-go chat", llmprovider.ProviderOpencodeGo, "opencode-go", "qwen4-max",
			func(base string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
				return opencode.NewGo(append(opts, llmprovider.WithBaseURL(base), opencode.WithRoute(opencode.RouteChatCompletions))...)
			}, ""},
	}
}

// warm0096 caches w's listing through p, when w has one.
func warm0096(t *testing.T, w wire0096, p llmprovider.Provider) {
	t.Helper()
	if w.listing == "" {
		return
	}
	if _, err := p.(llmprovider.ModelLister).ListModels(context.Background()); err != nil {
		t.Fatalf("%s ListModels: %v", w.name, err)
	}
}

// generate0096 builds w against srv with opts, caches its listing, and sends
// req, failing t on an error; it returns the response.
func generate0096(t *testing.T, w wire0096, srv *limitServer, req *llmprovider.Request,
	opts ...llmprovider.Option) *llmprovider.Response {
	t.Helper()
	p, err := w.build(srv.URL, append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModelProbes(false),
		llmprovider.WithModel(w.model)}, opts...)...)
	if err != nil {
		t.Fatalf("%s: %v", w.name, err)
	}
	warm0096(t, w, p)
	resp, err := p.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("%s: %v", w.name, err)
	}
	return resp
}

// TestOutputLimit_Sends_0096 (TG2, TG2a, TG3, HF1, HF2, OC1, OC2): with no
// caller value, a limit a "models.dev" layer states is sent, basis fact, its
// source "models.dev"; on Together, so is one the models.dev document handed
// in with catalog.WithMetadata states (TG2a); with nothing stated, none is
// sent, basis none, and no fallback on OpenCode's chat route. Hugging Face's
// listing is cached first (HF8a).
func TestOutputLimit_Sends_0096(t *testing.T) {
	for _, w := range wires0096() {
		srv := newLimitServer(t, w.listing, canned(wirecase.ChatReply))
		resp := generate0096(t, w, srv, hiRequest(), testCache(t), docLayer(t, w.section, w.model, 262144, 1000000))
		want := llmprovider.SentLimit{Tokens: 262144, Sent: true, Basis: llmprovider.LimitBasisFact, Source: "models.dev"}
		if got, ok := sentLimit(lastPost(srv)); !ok || int(got) != 262144 || resp.OutputLimit != want {
			t.Errorf("%s, a layer stating 262,144: sent %v (found %t), OutputLimit %+v; want 262144, %+v (TG2, HF1, OC1)",
				w.name, got, ok, resp.OutputLimit, want)
		}

		if w.id == llmprovider.ProviderTogether {
			doc := newLimitServer(t, w.listing, canned(wirecase.ChatReply))
			resp = generate0096(t, w, doc, hiRequest(), testCache(t),
				catalog.WithMetadata(docMeta(t, w.section, w.model, 262144, 1000000)))
			if got, ok := sentLimit(lastPost(doc)); !ok || int(got) != 262144 || resp.OutputLimit != want {
				t.Errorf("%s, the handed-in document stating 262,144: sent %v (found %t), OutputLimit %+v; want 262144, %+v (TG2a)",
					w.name, got, ok, resp.OutputLimit, want)
			}
		}

		none := newLimitServer(t, w.listing, canned(wirecase.ChatReply))
		resp = generate0096(t, w, none, hiRequest(), testCache(t))
		if got, ok := sentLimit(lastPost(none)); ok || resp.OutputLimit != (llmprovider.SentLimit{Basis: llmprovider.LimitBasisNone}) {
			t.Errorf("%s, nothing stated: sent %v (found %t), OutputLimit %+v; want none (TG3, HF2, OC2)", w.name, got, ok,
				resp.OutputLimit)
		}
	}
}

// TestOutputLimit_TogetherListing_0096 (TG1): Together's listing stating
// config.max_output_length 40,000 is sent, basis fact, source the listing.
func TestOutputLimit_TogetherListing_0096(t *testing.T) {
	const model = "org/listed-0096"
	srv := newLimitServer(t, `[{"id":"`+model+`","type":"chat","context_length":1000000,"config":{"max_output_length":40000}}]`,
		canned(wirecase.ChatReply))
	p, err := New(llmprovider.ProviderTogether, llmprovider.WithAPIKey("k"), llmprovider.WithModelProbes(false), testCache(t),
		llmprovider.WithBaseURL(srv.URL), llmprovider.WithModel(model))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := p.(llmprovider.ModelLister).ListModels(ctx); err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	resp, err := p.Generate(ctx, hiRequest())
	if err != nil {
		t.Fatal(err)
	}
	want := llmprovider.SentLimit{Tokens: 40000, Sent: true, Basis: llmprovider.LimitBasisFact, Source: "listing:together"}
	if got, ok := sentLimit(lastPost(srv)); !ok || got != 40000 || resp.OutputLimit != want {
		t.Errorf("a listing stating 40,000: sent %v (found %t), OutputLimit %+v; want 40000, %+v", got, ok, resp.OutputLimit, want)
	}
}

// TestOutputLimit_Caller_0096 (TG4, HF3, OC3): a caller's value is handled as
// before: the request's own sent as given, basis request; WithMaxTokens above
// the stated 40,000 cut to it, basis option.
func TestOutputLimit_Caller_0096(t *testing.T) {
	for _, w := range wires0096() {
		srv := newLimitServer(t, w.listing, canned(wirecase.ChatReply))
		req := hiRequest()
		req.MaxOutputTokens = 77
		resp := generate0096(t, w, srv, req, testCache(t), docLayer(t, w.section, w.model, 40000, 1000000))
		if got, _ := sentLimit(lastPost(srv)); got != 77 || resp.OutputLimit.Basis != llmprovider.LimitBasisRequest {
			t.Errorf("%s, MaxOutputTokens 77: sent %v, basis %q; want 77, request", w.name, got, resp.OutputLimit.Basis)
		}

		opt := newLimitServer(t, w.listing, canned(wirecase.ChatReply))
		resp = generate0096(t, w, opt, hiRequest(), testCache(t), docLayer(t, w.section, w.model, 40000, 1000000),
			llmprovider.WithMaxTokens(100000))
		if got, _ := sentLimit(lastPost(opt)); got != 40000 || resp.OutputLimit.Basis != llmprovider.LimitBasisOption {
			t.Errorf("%s, WithMaxTokens(100000), 40,000 stated: sent %v, basis %q; want 40000, option", w.name, got,
				resp.OutputLimit.Basis)
		}
	}
}

// TestOutputLimit_Window_0096 (TG5, OC4): a stated 40,000 with a stated
// window of 32,768 is held to the window, then to the room it leaves the
// request's estimated input.
func TestOutputLimit_Window_0096(t *testing.T) {
	for _, w := range wires0096() {
		if w.id == llmprovider.ProviderHuggingFace {
			continue // its window is its live providers' (TestOutputLimit_HuggingFaceWindow_0096)
		}
		srv := newLimitServer(t, "", canned(wirecase.ChatReply))
		generate0096(t, w, srv, hiRequest(), testCache(t), docLayer(t, w.section, w.model, 40000, 32768))
		if got, _ := sentLimit(lastPost(srv)); int(got) != room(32768, hiRequest()) {
			t.Errorf("%s, 40,000 stated, window 32,768: sent %v; want %d", w.name, got, room(32768, hiRequest()))
		}
	}
}

// TestOutputLimit_HuggingFaceWindow_0096 (HF4): a document stating 384,000,
// the listing's live providers stating windows of 1,048,576 and 262,144: held
// to the smaller (0089-MADR DL10), then to the room.
func TestOutputLimit_HuggingFaceWindow_0096(t *testing.T) {
	const model = "org/hf-window-0096"
	srv := newLimitServer(t, `{"data":[{"id":"`+model+`","architecture":{"input_modalities":["text"],"output_modalities":["text"]},`+
		`"providers":[{"provider":"a","status":"live","context_length":1048576},{"provider":"b","status":"live","context_length":262144}]}]}`,
		canned(wirecase.ChatReply))
	p, err := New(llmprovider.ProviderHuggingFace, llmprovider.WithAPIKey("k"), llmprovider.WithModelProbes(false), testCache(t),
		llmprovider.WithBaseURL(srv.URL), llmprovider.WithModel(model), docLayer(t, "huggingface", model, 384000, 0))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := p.(llmprovider.ModelLister).ListModels(ctx); err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if _, err := p.Generate(ctx, hiRequest()); err != nil {
		t.Fatal(err)
	}
	if got, _ := sentLimit(lastPost(srv)); int(got) != room(262144, hiRequest()) {
		t.Errorf("384,000 stated, windows 1,048,576 and 262,144: sent %v; want %d", got, room(262144, hiRequest()))
	}
}

// TestOutputLimit_HuggingFaceCold_0096 (HF10): in a cold process, with the
// listing not cached, a document stating 384,000 and a context of 1,000,000
// sends nothing, basis none, and adds no request: no live provider's window
// is known to hold it to.
func TestOutputLimit_HuggingFaceCold_0096(t *testing.T) {
	const model = "org/hf-cold-0096"
	srv := newLimitServer(t, hfListing0096(model), canned(wirecase.ChatReply))
	p, err := New(llmprovider.ProviderHuggingFace, llmprovider.WithAPIKey("k"), llmprovider.WithModelProbes(false), testCache(t),
		llmprovider.WithBaseURL(srv.URL), llmprovider.WithModel(model),
		catalog.WithMetadata(docMeta(t, "huggingface", model, 384000, 1000000)))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Generate(context.Background(), hiRequest())
	if err != nil {
		t.Fatal(err)
	}
	got, ok := sentLimit(lastPost(srv))
	if ok || resp.OutputLimit != (llmprovider.SentLimit{Basis: llmprovider.LimitBasisNone}) || len(srv.posts()) != 1 {
		t.Errorf("a cold process, 384,000 stated: sent %v (found %t), OutputLimit %+v, %d POST(s); want none, basis none, 1 POST",
			got, ok, resp.OutputLimit, len(srv.posts()))
	}
}

// TestOutputLimit_Streamed_0096 (TG9, HF7, OC7): a stream reads the streamed
// output, held to the cap the memo keeps for a streamed call (30,000), not
// the plain call's (20,000).
func TestOutputLimit_Streamed_0096(t *testing.T) {
	for _, w := range wires0096() {
		srv := newLimitServer(t, w.listing, canned(wirecase.ChatReply))
		cache := new(catalog.Cache)
		memo := memoIn(t, cache)
		memo.Keep(limitmemo.Key{Provider: string(w.id), BaseURL: srv.URL, Model: w.model, Stream: true}, 30000)
		memo.Keep(limitmemo.Key{Provider: string(w.id), BaseURL: srv.URL, Model: w.model}, 20000)
		p, err := w.build(srv.URL, llmprovider.WithAPIKey("k"), llmprovider.WithModelProbes(false), catalog.WithCache(cache),
			llmprovider.WithModel(w.model), docLayer(t, w.section, w.model, 40000, 1000000))
		if err != nil {
			t.Fatal(err)
		}
		warm0096(t, w, p)
		for _, err := range llmprovider.Stream(context.Background(), p, hiRequest()) {
			if err != nil {
				t.Fatalf("%s: %v", w.name, err)
			}
		}
		if got, _ := sentLimit(lastPost(srv)); got != 30000 {
			t.Errorf("%s, a stream, caps 30,000 streamed and 20,000 plain: sent %v; want 30000", w.name, got)
		}
	}
}

// TestOutputLimit_NoRequestAdded_0096 (TG10, HF7, HF10, OC7): a request
// reads the caches only: one POST and no GET. Hugging Face's is counted twice:
// after its listing is cached (HF8a), when it sends its stated limit, and in
// a cold process, when it sends none and does not fetch the listing to learn
// the window (HF10).
func TestOutputLimit_NoRequestAdded_0096(t *testing.T) {
	for _, w := range wires0096() {
		if w.listing != "" {
			cold := w
			cold.name += ", cold"
			countRequests0096(t, cold, false)
		}
		countRequests0096(t, w, true)
	}
}

// countRequests0096 sends one request on w, after caching its listing when
// warm, and reports on t unless it made one POST and no GET.
func countRequests0096(t *testing.T, w wire0096, warm bool) {
	t.Helper()
	var posts, gets atomic.Int32
	srv := httptest.NewServer(wiretest.StreamReplies(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		if r.Method != http.MethodPost {
			gets.Add(1)
			if w.listing != "" {
				rw.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(rw, w.listing)
				return
			}
			http.NotFound(rw, r)
			return
		}
		posts.Add(1)
		rw.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(rw, wirecase.ChatReply)
	})))
	t.Cleanup(srv.Close)
	p, err := w.build(srv.URL, llmprovider.WithAPIKey("k"), llmprovider.WithModelProbes(false), testCache(t),
		llmprovider.WithModel(w.model), docLayer(t, w.section, w.model, 40000, 1000000))
	if err != nil {
		t.Fatal(err)
	}
	if warm {
		warm0096(t, w, p)
	}
	posts.Store(0)
	gets.Store(0)
	if _, err := p.Generate(context.Background(), hiRequest()); err != nil {
		t.Fatalf("%s: %v", w.name, err)
	}
	if posts.Load() != 1 || gets.Load() != 0 {
		t.Errorf("%s: %d POST(s), %d GET(s); want 1 and 0", w.name, posts.Load(), gets.Load())
	}
}

// TestOutputLimit_OverStream_0096 (TG14): a plain Generate whose limit the
// SDK chose, 262,144, is sent as a stream; one naming its own limit stays a
// plain request (Together's streaming_required, measured on a plain
// request, is not reached by the SDK's own choice).
func TestOutputLimit_OverStream_0096(t *testing.T) {
	w := wires0096()[0]
	srv := newLimitServer(t, "", canned(wirecase.ChatReply))
	generate0096(t, w, srv, hiRequest(), testCache(t), docLayer(t, w.section, w.model, 262144, 1000000))
	body := lastPost(srv)
	if got, _ := sentLimit(body); got != 262144 || body["stream"] != true {
		t.Errorf("no caller value: sent %v, stream %v; want 262144 over a stream", got, body["stream"])
	}

	plain := newLimitServer(t, "", canned(wirecase.ChatReply))
	generate0096(t, w, plain, limitRequest(1000), testCache(t), docLayer(t, w.section, w.model, 262144, 1000000))
	body = lastPost(plain)
	if got, _ := sentLimit(body); got != 1000 || body["stream"] != nil {
		t.Errorf("MaxOutputTokens 1000: sent %v, stream %v; want 1000 on a plain request", got, body["stream"])
	}
}

// capRefusal0096 is OpenAI's Chat Completions cap form, refusing 40,000 and
// stating 32,000.
var capRefusal0096 = chatError("This model supports at most 32000 completion tokens, whereas you provided 40000.")

// TestLimitRefusal_Resend_0096 (TG6, HF5, OC5): the sent 40,000, refused in
// OpenAI's cap form, is resent once at 32,000, basis refusal, and the cap is
// reported in a WarningLimitLearned.
func TestLimitRefusal_Resend_0096(t *testing.T) {
	for _, w := range wires0096() {
		srv := newLimitServer(t, w.listing, refuseFirst(capRefusal0096, wirecase.ChatReply))
		resp := generate0096(t, w, srv, hiRequest(), catalog.WithCache(new(catalog.Cache)),
			docLayer(t, w.section, w.model, 40000, 1000000))
		posts := srv.posts()
		var warned bool
		for _, wn := range resp.Warnings {
			warned = warned || (wn.Kind == llmprovider.WarningLimitLearned && wn.Observed.Tokens == 32000)
		}
		first, _ := sentLimit(posts[0])
		resent, _ := sentLimit(posts[len(posts)-1])
		if len(posts) != 2 || first != 40000 || resent != 32000 || resp.OutputLimit.Basis != llmprovider.LimitBasisRefusal || !warned {
			t.Errorf("%s: %d send(s), %v then %v, basis %q, warned %t; want 40000 then 32000, refusal, warned", w.name,
				len(posts), first, resent, resp.OutputLimit.Basis, warned)
		}
	}
}

// TestLimitRefusal_Kept_0096 (TG7, HF5, OC5): after that refusal, the next
// request on the same cache sends the kept 32,000 at once.
func TestLimitRefusal_Kept_0096(t *testing.T) {
	for _, w := range wires0096() {
		srv := newLimitServer(t, w.listing, refuseFirst(capRefusal0096, wirecase.ChatReply))
		cache := catalog.WithCache(new(catalog.Cache))
		p, err := w.build(srv.URL, llmprovider.WithAPIKey("k"), llmprovider.WithModelProbes(false), cache,
			llmprovider.WithModel(w.model), docLayer(t, w.section, w.model, 40000, 1000000))
		if err != nil {
			t.Fatal(err)
		}
		warm0096(t, w, p)
		for i := range 2 {
			if _, err := p.Generate(context.Background(), hiRequest()); err != nil {
				t.Fatalf("%s, request %d: %v", w.name, i+1, err)
			}
		}
		posts := srv.posts()
		sent := make([]string, len(posts))
		for i, b := range posts {
			v, _ := sentLimit(b)
			sent[i] = fmt.Sprint(v)
		}
		if strings.Join(sent, ",") != "40000,32000,32000" {
			t.Errorf("%s: sent %s; want 40000,32000,32000 (the second request at the kept cap)", w.name, strings.Join(sent, ","))
		}
	}
}
