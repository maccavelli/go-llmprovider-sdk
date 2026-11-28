package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/chatcompat"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/claude"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// 0089-PLAN Phase 3's tests, built in the merged phase's layer B+C: the
// output limit a request carries (0089-MADR D6) and its report (D7), across
// the built-in providers. Every server is the test's own; no request leaves
// the machine.

// limitServer records each POST body in the order it came, and answers it
// with reply(n, body), n counting the POSTs from 0; a GET is answered with
// listing. Its bodies keep numbers exact, as json.Number.
type limitServer struct {
	URL    string
	mu     sync.Mutex
	bodies []map[string]any
}

// newLimitServer starts a limitServer, closed when t ends.
func newLimitServer(t *testing.T, listing string, reply func(n int, body map[string]any) (int, string)) *limitServer {
	t.Helper()
	s := &limitServer{}
	srv := httptest.NewServer(wiretest.StreamReplies(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			if listing == "" {
				http.NotFound(w, r)
				return
			}
			_, _ = io.WriteString(w, listing)
			return
		}
		body := map[string]any{}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		_ = dec.Decode(&body)
		s.mu.Lock()
		n := len(s.bodies)
		s.bodies = append(s.bodies, body)
		s.mu.Unlock()
		status, answer := reply(n, body)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, answer)
	})))
	t.Cleanup(srv.Close)
	s.URL = srv.URL
	return s
}

// posts is every POST body so far, in order.
func (s *limitServer) posts() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]map[string]any(nil), s.bodies...)
}

// canned answers every POST with reply, the canned reply for its wire.
func canned(reply string) func(int, map[string]any) (int, string) {
	return func(int, map[string]any) (int, string) { return http.StatusOK, reply }
}

// hiRequest is a request with one user message.
func hiRequest() *llmprovider.Request {
	return &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}
}

// registryLayer is a "models.dev" layer stating output for model in section.
func registryLayer(t *testing.T, section, model string, output int) llmprovider.Option {
	t.Helper()
	meta, err := catalog.ParseMetadata(strings.NewReader(`{"` + section + `":{"models":{"` + model +
		`":{"limit":{"output":` + itoa(output) + `}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	return catalog.WithModelLayer(meta)
}

// itoa is n in decimal.
func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// omitModel is a Claude-shaped model no extract section names, so that the
// test's own source is the only one stating its limit.
const omitModel = "claude-omit-0089"

// TestOutputLimit_Omitted_0089 (OQ1 (1): DL46 to DL48, DL65's verdict,
// DL69; 0088-MADR ST33, ST74, ST85, ST86, ST90, ST98, ST99): a request that
// names no limit carries the model's stated limit on a Messages route
// (Claude, read from its listing; OpenCode's messages route, from a
// registry) with no fixed ceiling, and Kilo its discovered limit, as
// DL65 decided, as do Together and OpenCode's chat route (0096-MADR TG11,
// OC8); every other optional wire carries none, Hugging Face with no listing
// cached included (0096-MADR HF10, HF8a),
// whatever a registry or a caller's layer states; a caller's own
// WithMaxTokens is sent on an optional wire. It reads only the wire, so it runs at a base that sends 8192.
func TestOutputLimit_Omitted_0089(t *testing.T) {
	ctx := context.Background()
	off := []llmprovider.Option{llmprovider.WithAPIKey("k"),
		llmprovider.WithModelProbes(false), testCache(t)}

	// Claude, a listing stating 128,000.
	claudeSrv := newLimitServer(t, `{"data":[{"id":"`+omitModel+`","max_tokens":128000,"max_input_tokens":1000000}],"has_more":false}`,
		canned(wirecase.MessagesReply))
	cl, err := New(llmprovider.ProviderClaude, append(off, llmprovider.WithBaseURL(claudeSrv.URL), llmprovider.WithModel(omitModel))...)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cl.(llmprovider.ModelLister).ListModels(ctx); err != nil {
		t.Fatalf("claude ListModels: %v", err)
	}
	if _, err := cl.Generate(ctx, hiRequest()); err != nil {
		t.Fatalf("claude: %v", err)
	}
	if got, ok := sentLimit(lastPost(claudeSrv)); !ok || got != 128000 {
		t.Errorf("claude, a listing stating 128,000: sent %v (found %t); want 128000", got, ok)
	}

	// OpenCode's messages route, a registry stating 64,000.
	zenSrv := newLimitServer(t, "", canned(wirecase.MessagesReply))
	zen, err := opencode.NewZen(append(off, llmprovider.WithBaseURL(zenSrv.URL), llmprovider.WithModel(omitModel),
		opencode.WithRoute(opencode.RouteMessages), registryLayer(t, "opencode", omitModel, 64000))...)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := zen.Generate(ctx, hiRequest()); err != nil {
		t.Fatalf("opencode-zen messages: %v", err)
	}
	if got, ok := sentLimit(lastPost(zenSrv)); !ok || got != 64000 {
		t.Errorf("opencode-zen messages, a registry stating 64,000: sent %v (found %t); want 64000", got, ok)
	}

	// Kilo: its discovered limit, as DL65 decided.
	kiloSrv := newLimitServer(t, "", canned(wirecase.ChatReply))
	kilo, err := New(llmprovider.ProviderKilo, append(off, llmprovider.WithBaseURL(kiloSrv.URL),
		llmprovider.WithModel("openai/gpt-6-luna"), catalog.WithModelLayer(limitLayer{"openai/gpt-6-luna", 128000}))...)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := kilo.Generate(ctx, hiRequest()); err != nil {
		t.Fatalf("kilo: %v", err)
	}
	if got, ok := sentLimit(lastPost(kiloSrv)); !ok || got != 128000 {
		t.Errorf("kilo, a layer stating 128,000: sent %v (found %t); want 128000 (DL65's verdict)", got, ok)
	}

	// Every optional wire: none, a caller's layer stating 128,000 included;
	// but Together and OpenCode's chat route send it (0096-MADR TG11, OC8).
	// Hugging Face, with no listing cached, knows no live window to hold it
	// to, so sends none (HF10, HF8a).
	for _, tc := range []struct {
		name  string
		reply string
		build func(base string, opts ...llmprovider.Option) (llmprovider.Provider, error)
		model string
		sends bool
	}{
		{"gemini", wirecase.InteractionsReply, byID(llmprovider.ProviderGemini), "gemini-3.8-flash", false},
		{"grok", wirecase.ResponsesReply, byID(llmprovider.ProviderGrok), "grok-4.6", false},
		{"openai", wirecase.ResponsesReply, byID(llmprovider.ProviderOpenAI), "gpt-6-luna", false},
		{"together", wirecase.ChatReply, byID(llmprovider.ProviderTogether), "openai/gpt-oss-120b", true},
		{"huggingface", wirecase.ChatReply, byID(llmprovider.ProviderHuggingFace), "deepseek-ai/DeepSeek-V4.1-Flash", false},
		{"ollama", wirecase.ChatReply, byID(llmprovider.ProviderOllama), "llama3.3:latest", false},
		{"chatcompat", wirecase.ChatReply, func(base string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
			return chatcompat.New(append(opts, llmprovider.WithBaseURL(base), chatcompat.WithID("compat-0089"))...)
		}, "org/compat-model", false},
		{"opencode-zen chat", wirecase.ChatReply, zenOn(opencode.RouteChatCompletions), "glm-5.3", true},
		{"opencode-zen responses", wirecase.ResponsesReply, zenOn(opencode.RouteResponses), "gpt-5.5", false},
		{"opencode-zen google", wirecase.GoogleReply, zenOn(opencode.RouteGoogle), "gemini-3.8-flash", false},
	} {
		srv := newLimitServer(t, "", canned(tc.reply))
		p, err := tc.build(srv.URL, append(off, llmprovider.WithModel(tc.model),
			catalog.WithModelLayer(limitLayer{tc.model, 128000}))...)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if _, err := p.Generate(ctx, hiRequest()); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		got, ok := sentLimit(lastPost(srv))
		switch {
		case tc.sends && (!ok || got != 128000):
			t.Errorf("%s, a layer stating 128,000 and no caller value: sent %v (found %t); want 128000 (0096-MADR)", tc.name, got, ok)
		case !tc.sends && ok:
			t.Errorf("%s, a layer stating 128,000 and no caller value: sent %v; want none (DL47, DL48, DL69)", tc.name, got)
		}
	}

	// A caller's own WithMaxTokens on an optional wire: sent.
	gemSrv := newLimitServer(t, "", canned(wirecase.InteractionsReply))
	gem, err := New(llmprovider.ProviderGemini, append(off, llmprovider.WithBaseURL(gemSrv.URL),
		llmprovider.WithModel("gemini-3.8-flash"), llmprovider.WithMaxTokens(5000))...)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gem.Generate(ctx, hiRequest()); err != nil {
		t.Fatalf("gemini: %v", err)
	}
	if got, ok := sentLimit(lastPost(gemSrv)); !ok || got != 5000 {
		t.Errorf("gemini, WithMaxTokens(5000): sent %v (found %t); want 5000", got, ok)
	}
}

// byID builds the built-in provider id against base.
func byID(id llmprovider.ProviderID) func(string, ...llmprovider.Option) (llmprovider.Provider, error) {
	return func(base string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
		return New(id, append(opts, llmprovider.WithBaseURL(base))...)
	}
}

// zenOn builds OpenCode Zen against base on route.
func zenOn(route opencode.Route) func(string, ...llmprovider.Option) (llmprovider.Provider, error) {
	return func(base string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
		return opencode.NewZen(append(opts, llmprovider.WithBaseURL(base), opencode.WithRoute(route))...)
	}
}

// lastPost is s's last POST body, or nil.
func lastPost(s *limitServer) map[string]any {
	posts := s.posts()
	if len(posts) == 0 {
		return nil
	}
	return posts[len(posts)-1]
}

// TestOutputLimit_Reported_0089 (DL44, DL46, DL47, DL53, DL54; 0088-MADR
// ST15 to ST17, ST67): Response.OutputLimit says what the request carried
// and why: a request's own value, basis request; WithMaxTokens', basis
// option, cut to the stated limit; no caller value on Claude, the stated
// fact, basis fact; on an optional wire, none; a listing probe's 8192 sent
// as given whatever the listing states; a stream's final response carries
// it.
func TestOutputLimit_Reported_0089(t *testing.T) {
	ctx := context.Background()
	const model = "claude-opus-4-6" // the extract fixture states 128,000
	claude := func(t *testing.T, srv *limitServer, opts ...llmprovider.Option) llmprovider.Provider {
		t.Helper()
		p, err := New(llmprovider.ProviderClaude, append([]llmprovider.Option{llmprovider.WithAPIKey("k"),
			llmprovider.WithModelProbes(false), testCache(t), extractDoc(),
			llmprovider.WithBaseURL(srv.URL), llmprovider.WithModel(model)}, opts...)...)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	for _, tc := range []struct {
		name    string
		request int
		opts    []llmprovider.Option
		want    llmprovider.SentLimit
	}{
		{"the request's own value", 5000, nil, llmprovider.SentLimit{Tokens: 5000, Sent: true, Basis: llmprovider.LimitBasisRequest}},
		{"WithMaxTokens(4000)", 0, []llmprovider.Option{llmprovider.WithMaxTokens(4000)},
			llmprovider.SentLimit{Tokens: 4000, Sent: true, Basis: llmprovider.LimitBasisOption}},
		{"WithMaxTokens(200000), a stated 128,000", 0, []llmprovider.Option{llmprovider.WithMaxTokens(200000)},
			llmprovider.SentLimit{Tokens: 128000, Sent: true, Basis: llmprovider.LimitBasisOption, Source: "models.dev"}},
		{"no caller value", 0, nil,
			llmprovider.SentLimit{Tokens: 128000, Sent: true, Basis: llmprovider.LimitBasisFact, Source: "models.dev"}},
	} {
		srv := newLimitServer(t, "", canned(wirecase.MessagesReply))
		req := hiRequest()
		req.MaxOutputTokens = tc.request
		resp, err := claude(t, srv, tc.opts...).Generate(ctx, req)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if resp.OutputLimit != tc.want {
			t.Errorf("%s: OutputLimit = %+v; want %+v", tc.name, resp.OutputLimit, tc.want)
		}
		if got, _ := sentLimit(lastPost(srv)); int(got) != tc.want.Tokens {
			t.Errorf("%s: the wire carried %v; want %d", tc.name, got, tc.want.Tokens)
		}
	}

	// A stream's final response carries it (DL54).
	streamSrv := newLimitServer(t, "", canned(messagesStream))
	req := hiRequest()
	req.MaxOutputTokens = 5000
	var final *llmprovider.Response
	for ev, err := range llmprovider.Stream(ctx, claude(t, streamSrv), req) {
		if err != nil {
			t.Fatalf("stream: %v", err)
		}
		if ev.Type == llmprovider.EventDone {
			final = ev.Response
		}
	}
	if want := (llmprovider.SentLimit{Tokens: 5000, Sent: true, Basis: llmprovider.LimitBasisRequest}); final == nil || final.OutputLimit != want {
		t.Errorf("the stream's final response: %+v; want OutputLimit %+v", final, want)
	}

	// An optional wire: a caller's value, basis option; none, basis none.
	for _, opts := range [][]llmprovider.Option{{llmprovider.WithMaxTokens(4000)}, nil} {
		srv := newLimitServer(t, "", canned(wirecase.InteractionsReply))
		p, err := New(llmprovider.ProviderGemini, append([]llmprovider.Option{llmprovider.WithAPIKey("k"),
			llmprovider.WithModelProbes(false), testCache(t),
			llmprovider.WithBaseURL(srv.URL), llmprovider.WithModel("gemini-3.8-flash")}, opts...)...)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := p.Generate(ctx, hiRequest())
		if err != nil {
			t.Fatal(err)
		}
		want := llmprovider.SentLimit{Basis: llmprovider.LimitBasisNone}
		if opts != nil {
			want = llmprovider.SentLimit{Tokens: 4000, Sent: true, Basis: llmprovider.LimitBasisOption}
		}
		if resp.OutputLimit != want {
			t.Errorf("gemini, options %d: OutputLimit = %+v; want %+v", len(opts), resp.OutputLimit, want)
		}
	}

	// A listing probe: its 8192 is given, sent whatever the listing states
	// (DL53; 0088-MADR ST26).
	probeSrv := newLimitServer(t, `{"data":[{"id":"`+model+`","max_tokens":128000}],"has_more":false}`,
		canned(wirecase.MessagesReply))
	p := claude(t, probeSrv, llmprovider.WithModelProbes(true))
	if _, err := p.(llmprovider.ModelLister).ListModels(ctx); err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	posts := probeSrv.posts()
	if len(posts) == 0 {
		t.Fatal("the listing sent no probe")
	}
	for _, body := range posts {
		if got, ok := sentLimit(body); !ok || got != 8192 {
			t.Errorf("a probe carried %v (found %t); want 8192, given", got, ok)
		}
	}
}

// TestOutputLimit_BudgetReported_0089 (DL55, DL67's plant (f)): on both
// Messages routes, with the budget thinking shape, budget 40,000 over a
// given 32,000, the wire carries 44,096 and OutputLimit.Tokens says so; a
// refusal stating "44096 > 32000" is read against that raised value, so its
// cap is kept (a caller's value is not resent: DL37).
func TestOutputLimit_BudgetReported_0089(t *testing.T) {
	ctx := context.Background()
	const model = "claude-opus-4-6"
	refusal := `{"type":"error","error":{"type":"invalid_request_error","message":"max_tokens: 44096 > 32000, which is the maximum allowed number of output tokens for ` + model + `"}}`
	for _, route := range []struct {
		name  string
		id    llmprovider.ProviderID
		build func(string, ...llmprovider.Option) (llmprovider.Provider, error)
	}{
		{"claude", llmprovider.ProviderClaude, byID(llmprovider.ProviderClaude)},
		{"opencode-zen messages", llmprovider.ProviderOpencodeZen, zenOn(opencode.RouteMessages)},
	} {
		for _, refused := range []bool{false, true} {
			reply := canned(wirecase.MessagesReply)
			if refused {
				reply = func(int, map[string]any) (int, string) { return http.StatusBadRequest, refusal }
			}
			srv := newLimitServer(t, "", reply)
			cache := catalog.WithCache(new(catalog.Cache))
			opts := []llmprovider.Option{llmprovider.WithAPIKey("k"),
				llmprovider.WithModelProbes(false), cache, llmprovider.WithModel(model)}
			p, err := route.build(srv.URL, append(opts, llmprovider.WithMaxTokens(32000))...)
			if err != nil {
				t.Fatal(err)
			}
			req := hiRequest()
			req.Reasoning = &llmprovider.Reasoning{Budget: 40000}
			resp, err := p.Generate(ctx, req)
			if got, _ := sentLimit(srv.posts()[0]); got != 44096 {
				t.Errorf("%s: the wire carried %v; want 44096, raised above the budget", route.name, got)
			}
			if !refused {
				if err != nil || resp.OutputLimit.Tokens != 44096 {
					t.Errorf("%s: OutputLimit = %+v, %v; want Tokens 44096, the wire's", route.name, resp, err)
				}
				continue
			}
			if err == nil || len(srv.posts()) != 1 {
				t.Errorf("%s, refused: %d send(s), err %v; want one send and the refusal, a caller's value", route.name,
					len(srv.posts()), err)
			}
			l := catalog.CachedLimits(ctx, route.id, model, append(opts, llmprovider.WithBaseURL(srv.URL))...)
			if l.Output.Tokens != 32000 || len(l.Output.Evidence) == 0 || l.Output.Evidence[0].Source != "observed" {
				t.Errorf("%s, refused stating 44096 > 32000: output %+v; want 32000 observed, read against the raised value",
					route.name, l.Output)
			}
		}
	}
}

// TestOutputLimit_EstimateReadsLimits_0089 (DL56; 0088-MADR ST87 as amended):
// Claude, a model in no other source whose layer states an input limit of
// 1,000,000 and an output of 1,000,000, and no window; a request whose
// content holds 350,000 characters (100,000 tokens at 3.5), no caller limit:
// the estimate reads Limits.Input where no window is stated, so the request
// carries 1,000,000 − 100,000 − 4,096 = 895,904, by Generate and by Stream.
func TestOutputLimit_EstimateReadsLimits_0089(t *testing.T) {
	ctx := context.Background()
	const model = "claude-estimate-0089"
	for _, streamCall := range []bool{false, true} {
		srv := newE88Server(t, nil, func(_ int, body map[string]any, w http.ResponseWriter) {
			answer(w, streamed(body), messagesStream, plainMessages)
		})
		p := e88Provider(t, llmprovider.ProviderClaude, model, srv,
			catalog.WithModelLayer(estimateLayer{model: model, output: 1000000, input: 1000000}))
		var err error
		if streamCall {
			_, _, _, err = drain(ctx, p, estimateRequest(350000))
		} else {
			_, err = p.Generate(ctx, estimateRequest(350000))
		}
		if err != nil {
			t.Fatalf("stream %t: %v", streamCall, err)
		}
		if got := srv.limits(); len(got) != 1 || got[0] != 895904 {
			t.Errorf("stream %t: sent %v; want one send at 895904, held by the input limit", streamCall, got)
		}
	}
}

// TestLimitObservations_Request_0089 (DL40, DL41, at the request): on the
// routes that require a limit, Claude and OpenCode's messages route, a model
// no source states and a passed-back cap of 30,000 seen an hour ago, for the
// provider's base URL (and, on Claude, its workspace): the request carries
// 30,000, basis fact, source remembered. Seen 25 hours ago, or for another
// base URL, or (Claude) another workspace, it is not used, and the request
// carries its route's fallback, 64,000 (DL51, DL77). A registry stating
// 64,000 beside the passed-back cap is chosen.
func TestLimitObservations_Request_0089(t *testing.T) {
	ctx := context.Background()
	const model = "claude-pass-back-0089" // in no extract section
	const elsewhere = "https://gateway.example.test"
	remembered := llmprovider.SentLimit{Tokens: 30000, Sent: true, Basis: llmprovider.LimitBasisFact, Source: "remembered"}
	fallback := llmprovider.SentLimit{Tokens: 64000, Sent: true, Basis: llmprovider.LimitBasisFallback,
		Source: "models.dev"}
	registry := llmprovider.SentLimit{Tokens: 64000, Sent: true, Basis: llmprovider.LimitBasisFact, Source: "models.dev"}
	for _, tc := range []struct {
		name   string
		claude bool
		ago    time.Duration
		base   string // the observation's base URL; "" for the test server's
		scope  string
		opts   []llmprovider.Option
		want   llmprovider.SentLimit
	}{
		{"claude, seen an hour ago", true, time.Hour, "", "", nil, remembered},
		{"claude, seen 25 hours ago", true, 25 * time.Hour, "", "", nil, fallback},
		{"claude, another base URL", true, time.Hour, elsewhere, "", nil, fallback},
		{"claude, another workspace", true, time.Hour, "", "wrkspc_other",
			[]llmprovider.Option{claude.WithWorkspace("wrkspc_mine")}, fallback},
		{"claude, the cap's own workspace", true, time.Hour, "", "wrkspc_mine",
			[]llmprovider.Option{claude.WithWorkspace("wrkspc_mine")}, remembered},
		{"opencode-zen messages, seen an hour ago", false, time.Hour, "", "", nil, remembered},
		{"opencode-zen messages, seen 25 hours ago", false, 25 * time.Hour, "", "", nil, fallback},
		{"opencode-zen messages, another base URL", false, time.Hour, elsewhere, "", nil, fallback},
		{"opencode-zen messages, a registry beside it", false, time.Hour, "", "",
			[]llmprovider.Option{registryLayer(t, "opencode", model, 64000)}, registry},
	} {
		srv := newLimitServer(t, "", canned(wirecase.MessagesReply))
		base := tc.base
		if base == "" {
			base = srv.URL
		}
		obs := llmprovider.LimitObservation{Model: model, BaseURL: base, Scope: tc.scope, Tokens: 30000,
			Seen: time.Now().Add(-tc.ago)}
		opts := append([]llmprovider.Option{llmprovider.WithAPIKey("k"),
			llmprovider.WithModelProbes(false), testCache(t), extractDoc(), llmprovider.WithModel(model),
			llmprovider.WithLimitObservations(obs)}, tc.opts...)
		build := zenOn(opencode.RouteMessages)
		if tc.claude {
			build = byID(llmprovider.ProviderClaude)
		}
		p, err := build(srv.URL, opts...)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		resp, err := p.Generate(ctx, hiRequest())
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if resp.OutputLimit != tc.want {
			t.Errorf("%s: OutputLimit = %+v; want %+v", tc.name, resp.OutputLimit, tc.want)
		}
		if got, _ := sentLimit(lastPost(srv)); int(got) != tc.want.Tokens {
			t.Errorf("%s: the wire carried %v; want %d", tc.name, got, tc.want.Tokens)
		}
	}
}

// TestOutputLimit_OutsideProvider_0089 (DL57): a provider outside this
// module, a chatcompat id named with chatcompat.WithID, reads CachedLimits
// for its model, a caller's layer's fact, and reads SentLimit on a
// response.
func TestOutputLimit_OutsideProvider_0089(t *testing.T) {
	ctx := context.Background()
	opts := []llmprovider.Option{chatcompat.WithID("outside-0089"),
		catalog.WithModelLayer(limitLayer{"org/outside", 20000}), testCache(t)}
	l := catalog.CachedLimits(ctx, "outside-0089", "org/outside", opts...)
	if !l.Output.Known || l.Output.Tokens != 20000 || l.Output.Evidence[0].Source != "test" {
		t.Errorf("CachedLimits for a chatcompat id: output %+v; want 20000 from the layer", l.Output)
	}
	srv := newLimitServer(t, "", canned(wirecase.ChatReply))
	p, err := chatcompat.New(append(opts, llmprovider.WithBaseURL(srv.URL), llmprovider.WithModel("org/outside"),
		llmprovider.WithAPIKey("k"), llmprovider.WithMaxTokens(30000))...)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Generate(ctx, hiRequest())
	if err != nil {
		t.Fatal(err)
	}
	if want := (llmprovider.SentLimit{Tokens: 20000, Sent: true, Basis: llmprovider.LimitBasisOption, Source: "test"}); resp.OutputLimit != want {
		t.Errorf("OutputLimit = %+v; want %+v, WithMaxTokens cut to the layer's", resp.OutputLimit, want)
	}
}
