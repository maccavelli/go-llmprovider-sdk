package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// 0088-PLAN Phase 1's remaining tests, as 0089-PLAN's merged phase builds
// them (its 0088 table, with 0088-MADR ST99's values), in layer E. Each reads
// only the wire, so that it runs at a base that sends 8192; every reply is
// this file's own, and none is a recorded one.

// e88Server records each POST's body and answers it with reply(n, body, w), n
// counting the POSTs from 0. A request get answers (true) is not recorded;
// every other GET is a 404.
type e88Server struct {
	*httptest.Server
	mu     sync.Mutex
	bodies []map[string]any
}

func newE88Server(t *testing.T, get func(http.ResponseWriter, *http.Request) bool,
	reply func(n int, body map[string]any, w http.ResponseWriter)) *e88Server {
	t.Helper()
	s := &e88Server{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if get != nil && get(w, r) {
			return
		}
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		s.mu.Lock()
		n := len(s.bodies)
		s.bodies = append(s.bodies, body)
		s.mu.Unlock()
		reply(n, body, w)
	}))
	t.Cleanup(s.Close)
	return s
}

// posts is every POST body, in order.
func (s *e88Server) posts() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]map[string]any(nil), s.bodies...)
}

// limits is the output limit each POST carried, in order; -1 for none.
func (s *e88Server) limits() []int {
	var out []int
	for _, b := range s.posts() {
		n, ok := limitIn(b)
		if !ok {
			n = -1
		}
		out = append(out, n)
	}
	return out
}

// windowLayer is a caller layer stating one model's output limit and window;
// a zero states none.
type windowLayer struct {
	model          string
	output, window int
}

func (l windowLayer) Name() string { return "test" }

func (l windowLayer) Model(_ llmprovider.ProviderID, id string) (catalog.ModelInfo, bool) {
	if id != l.model {
		return catalog.ModelInfo{}, false
	}
	info := catalog.ModelInfo{ID: id, MaxOutputTokens: l.output, ContextWindow: l.window}
	var known []catalog.Field
	if l.output > 0 {
		known = append(known, catalog.FieldMaxOutputTokens)
	}
	if l.window > 0 {
		known = append(known, catalog.FieldContextWindow)
	}
	return info.Known(known...), true
}

// e88Provider builds id for srv with opts, no metadata fetch, no probe and a
// cache of its own.
func e88Provider(t *testing.T, id llmprovider.ProviderID, model string, srv *e88Server,
	opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := New(id, append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModel(model),
		llmprovider.WithBaseURL(srv.URL), llmprovider.WithoutModelMetadata(), llmprovider.WithModelProbes(false),
		testCache(t)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// drain ranges over p's stream of req, and returns its text, its final
// response, and the error it yielded, if any, with the events before it.
func drain(ctx context.Context, p llmprovider.Provider, req *llmprovider.Request) (text string,
	final *llmprovider.Response, before int, err error) {
	for ev, e := range llmprovider.Stream(ctx, p, req) {
		if e != nil {
			return text, final, before, e
		}
		before++
		if ev.Type == llmprovider.EventTextDelta {
			text += ev.Text
		}
		if ev.Type == llmprovider.EventDone {
			final = ev.Response
		}
	}
	return text, final, before, nil
}

// Replies of this file's own.
const (
	// plainChat is a whole Chat Completions reply saying "hello".
	plainChat = `{"id":"c","model":"m","choices":[{"index":0,"finish_reason":"stop",` +
		`"message":{"role":"assistant","content":"hello"}}],"usage":{"prompt_tokens":3,"completion_tokens":2}}`
	// kiloWindow is OpenRouter's window form as Kilo passes it (0088-MADR
	// C39): a window of 32,768 and about 40,000 tokens asked.
	kiloWindow = "This endpoint's maximum context length is 32768 tokens. However, you requested about 40000 tokens."
)

// e88ChatError is a Chat Completions refusal carrying msg, as a 400's body.
func e88ChatError(msg string) string {
	return fmt.Sprintf(`{"error":{"message":%q,"type":"invalid_request_error","code":400}}`, msg)
}

// e88MessagesError is a Messages refusal carrying msg, as a 400's body.
func e88MessagesError(msg string) string {
	return fmt.Sprintf(`{"type":"error","error":{"type":"invalid_request_error","message":%q}}`, msg)
}

// chatErrorEvent is a Chat Completions stream's error payload carrying msg.
func chatErrorEvent(msg string) string {
	return fmt.Sprintf("data: {\"error\":{\"message\":%q,\"type\":\"invalid_request_error\",\"code\":400}}\n\n", msg)
}

// claudeCap is Anthropic's refusal of a max_tokens of sent above the cap,
// for model (0089-MADR W7), as a 400's body.
func claudeCap(sent, limit int, model string) string {
	return e88MessagesError(fmt.Sprintf("max_tokens: %d > %d, which is the maximum allowed number of output tokens for %s",
		sent, limit, model))
}

// answer writes the reply for a stream, or the plain one.
func answer(w http.ResponseWriter, stream bool, streamed, plain string) {
	if stream {
		_, _ = io.WriteString(w, streamed)
		return
	}
	_, _ = io.WriteString(w, plain)
}

// refuse writes status with body.
func refuse(w http.ResponseWriter, status int, body string) {
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

// TestDefaultLimit_GrokSilent_0088 (ST63): Grok, a layer stating 128,000 and
// no caller limit: Generate and Stream send no max_output_tokens, so xAI's
// own default applies; a caller's WithMaxTokens(5000) is sent, on both.
func TestDefaultLimit_GrokSilent_0088(t *testing.T) {
	ctx := context.Background()
	const model = "grok-4.6"
	for _, opts := range [][]llmprovider.Option{nil, {llmprovider.WithMaxTokens(5000)}} {
		want := -1
		if opts != nil {
			want = 5000
		}
		for _, streamCall := range []bool{false, true} {
			srv := newE88Server(t, nil, func(_ int, body map[string]any, w http.ResponseWriter) {
				answer(w, streamed(body), wirecase.SSEReply, wirecase.ResponsesReply)
			})
			p := e88Provider(t, llmprovider.ProviderGrok, model, srv,
				append([]llmprovider.Option{catalog.WithModelLayer(limitLayer{model, 128000})}, opts...)...)
			var err error
			if streamCall {
				_, _, _, err = drain(ctx, p, overStreamHi())
			} else {
				_, err = p.Generate(ctx, overStreamHi())
			}
			if err != nil {
				t.Fatalf("stream %t, options %d: %v", streamCall, len(opts), err)
			}
			if got := srv.limits(); len(got) != 1 || got[0] != want {
				t.Errorf("grok, stream %t, options %d: sent %v; want %d (-1: no max_output_tokens)", streamCall, len(opts),
					got, want)
			}
		}
	}
}

// TestDefaultLimit_WindowCap_0088 (ST91), on Kilo, which sends the limit it
// discovers until 0089-MADR DL65's verdict: a layer stating L 16,384 and W
// 8,192 sends 8,192; L 20,000 and W 1,000,000 sends 20,000; a caller's
// WithMaxTokens(20000) with a stated W of 8,192, and no output limit stated,
// is sent as given. The model is in no extract section, so the layer is the
// only source.
func TestDefaultLimit_WindowCap_0088(t *testing.T) {
	ctx := context.Background()
	const model = "org/window-cap-0088"
	for _, tc := range []struct {
		name  string
		layer windowLayer
		opts  []llmprovider.Option
		want  int
	}{
		{"L 16,384, W 8,192", windowLayer{model, 16384, 8192}, nil, 8192},
		{"L 20,000, W 1,000,000", windowLayer{model, 20000, 1000000}, nil, 20000},
		{"WithMaxTokens(20000), W 8,192", windowLayer{model, 0, 8192}, []llmprovider.Option{llmprovider.WithMaxTokens(20000)}, 20000},
	} {
		srv := newE88Server(t, nil, func(_ int, body map[string]any, w http.ResponseWriter) {
			answer(w, streamed(body), chatStream, plainChat)
		})
		p := e88Provider(t, llmprovider.ProviderKilo, model, srv,
			append([]llmprovider.Option{catalog.WithModelLayer(tc.layer)}, tc.opts...)...)
		if _, err := p.Generate(ctx, overStreamHi()); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got := srv.limits(); len(got) != 1 || got[0] != tc.want {
			t.Errorf("kilo, %s: sent %v; want %d", tc.name, got, tc.want)
		}
	}
}

// TestStreamEventsRetry_FirstPayloadError_0088 (ST55; 0088-MADR A5), on Kilo
// only (Together sends no limit the SDK chose, 0089-MADR DL47): a Stream
// whose limit is the discovered 16,384, answered by a 200 whose first payload
// is C39's window form, is resent once at the room it states, 9,152, and the
// answer streamed with no event before it; the same error after a text delta
// is yielded after the delta, after one send.
func TestStreamEventsRetry_FirstPayloadError_0088(t *testing.T) {
	ctx := context.Background()
	const model = "openai/gpt-6-luna"
	delta := "data: {\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"hel\"},\"finish_reason\":null}]}\n\n"
	for _, tc := range []struct {
		name  string
		first string
	}{
		{"the error as the first payload", chatErrorEvent(kiloWindow)},
		{"the error after a text delta", delta + chatErrorEvent(kiloWindow)},
	} {
		srv := newE88Server(t, nil, func(n int, body map[string]any, w http.ResponseWriter) {
			if n == 0 {
				_, _ = io.WriteString(w, tc.first)
				return
			}
			answer(w, streamed(body), chatStream, plainChat)
		})
		p := e88Provider(t, llmprovider.ProviderKilo, model, srv, catalog.WithModelLayer(limitLayer{model, 16384}))
		text, _, before, err := drain(ctx, p, overStreamHi())
		got := srv.limits()
		if tc.name == "the error as the first payload" {
			if err != nil || text != "hello" || len(got) != 2 || got[0] != 16384 || got[1] != 9152 {
				t.Errorf("%s: sent %v, text %q, err %v; want 16384 then 9152, and hello", tc.name, got, text, err)
			}
			continue
		}
		if err == nil || before == 0 || text != "hel" || len(got) != 1 {
			t.Errorf("%s: sent %v, %d event(s) then err %v; want one send, the delta, then the error", tc.name, got,
				before, err)
		}
	}
}

// Claude's refusals of an effort (0062-PLAN Phase 1's recorded form), each
// listing the efforts left, and of the adaptive shape.
func refusedEffort(accepted string) string {
	return e88MessagesError("output_config.effort: Input should be " + accepted)
}

// TestDefaultLimit_ResendLadder_0088 (ST56, ST57), on Claude, L 128,000,
// effort max, by Generate and Stream: (1) a limit refusal stating 64,000,
// then an effort refusal: the third send at 64,000 with the lower effort,
// and so when the cap names another model, which no memo keeps; (2) an effort refusal first: the second send keeps 128,000 with the lower
// effort; (3) a limit refusal, then refusals down the whole ladder, the last
// rung accepted: eight sends, the answer returned, the limit's resend not
// counted against the bound of seven.
func TestDefaultLimit_ResendLadder_0088(t *testing.T) {
	ctx := context.Background()
	const model = "claude-sonnet-5"
	limit := claudeCap(128000, 64000, model)
	adaptive := e88MessagesError("adaptive thinking is not supported on this model")
	for _, tc := range []struct {
		name     string
		refusals []string
		limits   []int
		efforts  []string
	}{
		{"(1) limit, then effort", []string{limit, refusedEffort("'low', 'medium', 'high' or 'xhigh'")},
			[]int{128000, 64000, 64000}, []string{"max", "max", "xhigh"}},
		// A cap naming another model serves this request only (0089-MADR
		// DL36): no memo holds it, so only the resend's stickiness keeps it.
		{"(1) limit naming another model, then effort", []string{claudeCap(128000, 64000, "claude-elsewhere-0088"),
			refusedEffort("'low', 'medium', 'high' or 'xhigh'")}, []int{128000, 64000, 64000}, []string{"max", "max", "xhigh"}},
		{"(2) effort first", []string{refusedEffort("'low', 'medium', 'high' or 'xhigh'")},
			[]int{128000, 128000}, []string{"max", "xhigh"}},
		{"(3) limit, then the whole ladder", []string{limit, refusedEffort("'low', 'medium', 'high' or 'xhigh'"),
			refusedEffort("'low', 'medium' or 'high'"), refusedEffort("'low' or 'medium'"), refusedEffort("'low'"),
			refusedEffort("'none'"), adaptive},
			[]int{128000, 64000, 64000, 64000, 64000, 64000, 64000, 64000},
			[]string{"max", "max", "xhigh", "high", "medium", "low", "", ""}},
	} {
		for _, streamCall := range []bool{false, true} {
			srv := newE88Server(t, nil, func(n int, body map[string]any, w http.ResponseWriter) {
				if n < len(tc.refusals) {
					refuse(w, http.StatusBadRequest, tc.refusals[n])
					return
				}
				answer(w, streamed(body), messagesStream, plainMessages)
			})
			p := e88Provider(t, llmprovider.ProviderClaude, model, srv, catalog.WithModelLayer(limitLayer{model, 128000}))
			req := overStreamHi()
			req.Reasoning = &llmprovider.Reasoning{Effort: llmprovider.EffortMax}
			var text string
			var err error
			if streamCall {
				text, _, _, err = drain(ctx, p, req)
			} else {
				var resp *llmprovider.Response
				if resp, err = p.Generate(ctx, req); resp != nil {
					text = resp.OutputText()
				}
			}
			var efforts []string
			for _, b := range srv.posts() {
				efforts = append(efforts, sentEffort(b))
			}
			if err != nil || text != "hello" || fmt.Sprint(srv.limits()) != fmt.Sprint(tc.limits) ||
				fmt.Sprint(efforts) != fmt.Sprint(tc.efforts) {
				t.Errorf("%s, stream %t: sent %v at efforts %q, text %q, err %v; want %v at %q, and hello", tc.name,
					streamCall, srv.limits(), efforts, text, err, tc.limits, tc.efforts)
			}
		}
	}
}

// TestDefaultLimit_Concurrent_0088 (ST46), under -race, on Kilo: 32
// concurrent requests on one provider, its discovered limit 16,384, the
// server refusing half of them (by a marker in the request) in C39's window
// form: each refused request sends 16,384, then the room the refusal states,
// 9,152; each other sends 16,384 once. Every value lives in the request's own
// build.
func TestDefaultLimit_Concurrent_0088(t *testing.T) {
	ctx := context.Background()
	const model = "openai/gpt-6-luna"
	srv := newE88Server(t, nil, func(_ int, body map[string]any, w http.ResponseWriter) {
		limit, _ := limitIn(body)
		if limit == 16384 && strings.Contains(fmt.Sprint(body["messages"]), "refuse") {
			refuse(w, http.StatusBadRequest, e88ChatError(kiloWindow))
			return
		}
		answer(w, streamed(body), chatStream, plainChat)
	})
	p := e88Provider(t, llmprovider.ProviderKilo, model, srv, catalog.WithModelLayer(limitLayer{model, 16384}))
	var wg sync.WaitGroup
	errs := make([]error, 32)
	for i := range 32 {
		wg.Go(func() {
			text := fmt.Sprintf("plain %d", i)
			if i%2 == 0 {
				text = fmt.Sprintf("refuse %d", i)
			}
			_, errs[i] = p.Generate(ctx, &llmprovider.Request{Input: []llmprovider.Item{
				llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: text}}})
		})
	}
	wg.Wait()
	sent := map[string][]int{}
	for _, b := range srv.posts() {
		limit, _ := limitIn(b)
		key := fmt.Sprint(b["messages"])
		sent[key] = append(sent[key], limit)
	}
	for i, err := range errs {
		if err != nil {
			t.Errorf("request %d: %v", i, err)
		}
	}
	if len(sent) != 32 {
		t.Fatalf("%d distinct requests sent; want 32", len(sent))
	}
	for key, limits := range sent {
		want := "[16384]"
		if strings.Contains(key, "refuse") {
			want = "[16384 9152]"
		}
		if fmt.Sprint(limits) != want {
			t.Errorf("%s: sent %v; want %s", key, limits, want)
		}
	}
}

// TestDefaultLimit_ListingProbe_0088 (ST26; 0089-MADR DL53): Claude's and
// Ollama's listing probes, a caller's layer stating 128,000 for the probed
// model: each probe sends 8192, given, as the probe always did.
func TestDefaultLimit_ListingProbe_0088(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		id      llmprovider.ProviderID
		model   string
		listing string
		plain   string
	}{
		{"claude", llmprovider.ProviderClaude, "claude-sonnet-5",
			`{"data":[{"id":"claude-sonnet-5","type":"model"}],"has_more":false}`, plainMessages},
		{"ollama", llmprovider.ProviderOllama, "llama3.3:latest",
			`{"models":[{"name":"llama3.3:latest","model":"llama3.3:latest"}]}`, plainChat},
	} {
		srv := newE88Server(t, func(w http.ResponseWriter, r *http.Request) bool {
			switch {
			case r.Method == http.MethodGet:
				_, _ = io.WriteString(w, tc.listing)
			case strings.HasPrefix(r.URL.Path, "/api/"):
				http.NotFound(w, r)
			default:
				return false
			}
			return true
		}, func(_ int, body map[string]any, w http.ResponseWriter) {
			answer(w, streamed(body), chatStream, tc.plain)
		})
		p := e88Provider(t, tc.id, tc.model, srv, llmprovider.WithModelProbes(true),
			catalog.WithModelLayer(limitLayer{tc.model, 128000}))
		if _, err := p.(llmprovider.ModelLister).ListModels(ctx); err != nil {
			t.Fatalf("%s: ListModels: %v", tc.name, err)
		}
		got := srv.limits()
		if len(got) == 0 {
			t.Fatalf("%s: the listing sent no probe", tc.name)
		}
		for _, n := range got {
			if n != 8192 {
				t.Errorf("%s: a probe sent %v; want 8192, given", tc.name, got)
				break
			}
		}
	}
}

// TestDefaultLimit_ClaudeWindowStop_0088 (ST23), on Claude, L 128,000, by
// Generate and Stream: a reply that stops at the window
// (model_context_window_exceeded) with text is FinishLength, the answer
// returned after one send; a 400 saying the prompt is too long names the
// input alone, and is returned after one send.
func TestDefaultLimit_ClaudeWindowStop_0088(t *testing.T) {
	ctx := context.Background()
	const model = "claude-sonnet-5"
	stop := func(s string) string { return strings.Replace(s, `"end_turn"`, `"model_context_window_exceeded"`, 1) }
	for _, streamCall := range []bool{false, true} {
		srv := newE88Server(t, nil, func(_ int, body map[string]any, w http.ResponseWriter) {
			answer(w, streamed(body), stop(messagesStream), stop(plainMessages))
		})
		p := e88Provider(t, llmprovider.ProviderClaude, model, srv, catalog.WithModelLayer(limitLayer{model, 128000}))
		var resp *llmprovider.Response
		var err error
		if streamCall {
			_, resp, _, err = drain(ctx, p, overStreamHi())
		} else {
			resp, err = p.Generate(ctx, overStreamHi())
		}
		if err != nil || resp == nil || resp.FinishReason != llmprovider.FinishLength || resp.OutputText() != "hello" ||
			len(srv.posts()) != 1 {
			t.Errorf("stream %t, stopped at the window: %d send(s), resp %+v, err %v; want one send, FinishLength and hello",
				streamCall, len(srv.posts()), resp, err)
		}

		long := newE88Server(t, nil, func(_ int, _ map[string]any, w http.ResponseWriter) {
			refuse(w, http.StatusBadRequest, e88MessagesError("prompt is too long: 213462 tokens > 200000 maximum"))
		})
		p = e88Provider(t, llmprovider.ProviderClaude, model, long, catalog.WithModelLayer(limitLayer{model, 128000}))
		if streamCall {
			_, _, _, err = drain(ctx, p, overStreamHi())
		} else {
			_, err = p.Generate(ctx, overStreamHi())
		}
		if !errors.Is(err, llmprovider.ErrContextOverflow) || len(long.posts()) != 1 {
			t.Errorf("stream %t, the prompt too long: %d send(s), err %v; want one send and ErrContextOverflow", streamCall,
				len(long.posts()), err)
		}
	}
}

// overflowRoute is a route that sends a limit the SDK chose: Claude,
// OpenCode's messages route and Kilo, with its wire's replies.
type overflowRoute struct {
	name           string
	id             llmprovider.ProviderID
	model          string
	opts           []llmprovider.Option
	stream, plain  string
	errorFor       func(msg string) string
	capFor         func(sent int) string // a cap refusal of sent, in the route's form
	inputAlone     string                // an overflow naming the input alone
	windowAtOrOver []string              // window forms whose input is at or over the window
}

// overflowRoutes are the three routes that send a limit the SDK chose.
func overflowRoutes() []overflowRoute {
	const claudeModel, kiloModel = "claude-sonnet-5", "openai/gpt-6-luna"
	return []overflowRoute{
		{name: "claude", id: llmprovider.ProviderClaude, model: claudeModel, stream: messagesStream, plain: plainMessages,
			errorFor: e88MessagesError, capFor: func(sent int) string { return claudeCap(sent, 16000, claudeModel) },
			inputAlone: e88MessagesError("prompt is too long: 213462 tokens > 200000 maximum")},
		{name: "opencode-zen messages", id: llmprovider.ProviderOpencodeZen, model: claudeModel,
			opts: []llmprovider.Option{opencode.WithRoute(opencode.RouteMessages)}, stream: messagesStream, plain: plainMessages,
			errorFor: e88MessagesError, capFor: func(sent int) string { return claudeCap(sent, 16000, claudeModel) },
			inputAlone: e88MessagesError("prompt is too long: 213462 tokens > 200000 maximum")},
		{name: "kilo", id: llmprovider.ProviderKilo, model: kiloModel, stream: chatStream, plain: plainChat,
			errorFor: e88ChatError, capFor: func(int) string { return e88ChatError(kiloWindow) },
			inputAlone: e88ChatError("The input (140000 tokens) is longer than the model's context length (131072 tokens)."),
			windowAtOrOver: []string{
				e88ChatError("This request exceeds the model's context window of 200000 tokens. Your request contains approximately 330000 tokens."),
				e88ChatError("This endpoint's maximum context length is 200000 tokens. However, you requested about 328000 tokens."),
			}},
	}
}

// TestDefaultLimit_OverflowResend_0088 (ST48, ST59, ST61; the merged phase's
// cases 3, 5, 6 and 7), on the routes that send a limit the SDK chose
// (Claude, OpenCode's messages route, Kilo), L 128,000: (3) a caller's
// WithMaxTokens(20000) refused in a form that states a number: one send; (5)
// an overflow naming the input alone (0088-MADR C99 to C101): one send; (6)
// a window form whose input is at or over the window (Kilo's two): one send;
// (7) a 429, a 500, a tool-schema refusal (0084) and another invalid
// request: one send each, as before. Each first send not the caller's
// carries 128,000.
func TestDefaultLimit_OverflowResend_0088(t *testing.T) {
	ctx := context.Background()
	toolSchema := `Invalid schema for function 'lookup': In context=('properties', 'q'), 'additionalProperties' is required to be supplied and to be false.`
	for _, r := range overflowRoutes() {
		type refusalCase struct {
			name   string
			status int
			body   string
			opts   []llmprovider.Option
		}
		cases := []refusalCase{
			{"(3) WithMaxTokens(20000) refused", http.StatusBadRequest, r.capFor(20000),
				[]llmprovider.Option{llmprovider.WithMaxTokens(20000)}},
			{"(5) the input alone", http.StatusBadRequest, r.inputAlone, nil},
			{"(7) 429", http.StatusTooManyRequests, r.errorFor("rate limit reached"), nil},
			{"(7) 500", http.StatusInternalServerError, r.errorFor("internal error"), nil},
			{"(7) a tool-schema refusal", http.StatusBadRequest, r.errorFor(toolSchema), nil},
			{"(7) another invalid request", http.StatusBadRequest, r.errorFor("Unknown parameter: temperature_x"), nil},
		}
		for i, body := range r.windowAtOrOver {
			cases = append(cases, refusalCase{fmt.Sprintf("(6) a window form, the input at or over it (%d)", i+1),
				http.StatusBadRequest, body, nil})
		}
		for _, tc := range cases {
			srv := newE88Server(t, nil, func(_ int, _ map[string]any, w http.ResponseWriter) {
				refuse(w, tc.status, tc.body)
			})
			opts := append([]llmprovider.Option{catalog.WithModelLayer(limitLayer{r.model, 128000})}, r.opts...)
			p := e88Provider(t, r.id, r.model, srv, append(opts, tc.opts...)...)
			_, err := p.Generate(ctx, overStreamHi())
			got := srv.limits()
			want := 128000
			if tc.opts != nil {
				want = 20000
			}
			if err == nil || len(got) != 1 || got[0] != want {
				t.Errorf("%s, %s: sent %v, err %v; want one send at %d, the error returned", r.name, tc.name, got, err, want)
			}
		}
	}
}

// TestDefaultLimit_LimitRefusal_0088 (ST50, its Kilo case): Kilo, L 65,536,
// a reply finishing length with text is returned after one send, never
// resent.
func TestDefaultLimit_LimitRefusal_0088(t *testing.T) {
	ctx := context.Background()
	const model = "openai/gpt-6-luna"
	length := func(s string) string {
		return strings.Replace(s, `"finish_reason":"stop"`, `"finish_reason":"length"`, 1)
	}
	srv := newE88Server(t, nil, func(_ int, body map[string]any, w http.ResponseWriter) {
		answer(w, streamed(body), length(chatStream), length(plainChat))
	})
	p := e88Provider(t, llmprovider.ProviderKilo, model, srv, catalog.WithModelLayer(limitLayer{model, 65536}))
	resp, err := p.Generate(ctx, overStreamHi())
	if err != nil || resp.FinishReason != llmprovider.FinishLength || resp.OutputText() != "hello" ||
		fmt.Sprint(srv.limits()) != "[65536]" {
		t.Errorf("kilo, a length finish: sent %v, resp %+v, err %v; want one send at 65536, FinishLength and hello",
			srv.limits(), resp, err)
	}
}

// TestDefaultLimit_ResendBudgetShape_0088 (ST52, with ST99's value), on
// Claude, L 128,000, the budget shape, budget 10,000, by Stream and Generate:
// the first send's max_tokens 128,000 refused stating 64,000; the second
// sends max_tokens 64,000 and budget_tokens 10,000.
func TestDefaultLimit_ResendBudgetShape_0088(t *testing.T) {
	ctx := context.Background()
	const model = "claude-sonnet-5"
	for _, streamCall := range []bool{false, true} {
		srv := newE88Server(t, nil, func(_ int, body map[string]any, w http.ResponseWriter) {
			if limit, _ := limitIn(body); limit > 64000 {
				refuse(w, http.StatusBadRequest, claudeCap(limit, 64000, model))
				return
			}
			answer(w, streamed(body), messagesStream, plainMessages)
		})
		p := e88Provider(t, llmprovider.ProviderClaude, model, srv, catalog.WithModelLayer(limitLayer{model, 128000}))
		req := overStreamHi()
		req.Reasoning = &llmprovider.Reasoning{Budget: 10000}
		var err error
		if streamCall {
			_, _, _, err = drain(ctx, p, req)
		} else {
			_, err = p.Generate(ctx, req)
		}
		posts := srv.posts()
		var budget any
		if len(posts) == 2 {
			th, _ := posts[1]["thinking"].(map[string]any)
			budget = th["budget_tokens"]
		}
		if err != nil || fmt.Sprint(srv.limits()) != "[128000 64000]" || budget != float64(10000) {
			t.Errorf("stream %t: sent %v, the second's budget_tokens %v, err %v; want 128000 then 64000, budget 10000",
				streamCall, srv.limits(), budget, err)
		}
	}
}
