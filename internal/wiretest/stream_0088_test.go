package wiretest_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/claude"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/gemini"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/together"
)

// 0088-MADR ST88e: a test server's plain reply, written as its wire's stream
// by StreamReply, is read by the provider's stream into the response the
// plain reply gives a plain Generate.

// Whole replies, each with reasoning, text and a call where its wire has them.
// A stream's reasoning_details entry is merged by its index and written again
// with its keys sorted, so the entries here are written that way.
const (
	messagesReply = `{"id":"msg_1","type":"message","stop_reason":"tool_use","content":[` +
		`{"type":"thinking","thinking":"think","signature":"sig"},{"type":"text","text":"hello"},` +
		`{"type":"tool_use","id":"toolu_1","name":"get_weather","input":{"city":"Paris"}}],` +
		`"usage":{"input_tokens":15,"cache_read_input_tokens":100,"output_tokens":40}}`
	responsesReply = `{"id":"resp_1","status":"completed","output":[` +
		`{"type":"reasoning","summary":[{"type":"summary_text","text":"think"}]},` +
		`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]},` +
		`{"type":"function_call","call_id":"call_1","name":"get_weather","arguments":"{\"city\":\"Paris\"}"}],` +
		`"usage":{"input_tokens":120,"output_tokens":40,"output_tokens_details":{"reasoning_tokens":30}}}`
	interactionsReply = `{"id":"v1_int_1","status":"completed","steps":[` +
		`{"type":"thought","signature":"sig","summary":[{"type":"text","text":"think"}]},` +
		`{"type":"model_output","content":[{"type":"text","text":"hello"}]},` +
		`{"type":"function_call","id":"call_1","name":"get_weather","arguments":{"city":"Paris"}}],` +
		`"usage":{"total_input_tokens":120,"total_output_tokens":10,"total_thought_tokens":30}}`
	googleReply = `{"candidates":[{"content":{"role":"model","parts":[` +
		`{"text":"think","thought":true},{"text":"hello"},` +
		`{"functionCall":{"name":"get_weather","args":{"city":"Paris"}},"thoughtSignature":"sig"}]}}],` +
		`"usageMetadata":{"promptTokenCount":120,"candidatesTokenCount":10,"thoughtsTokenCount":30}}`
	chatReply = `{"id":"chat_1","choices":[{"index":0,"finish_reason":"tool_calls","message":{` +
		`"role":"assistant","content":[{"type":"text","text":"hel"},{"type":"text","text":"lo"}],"reasoning_content":"think",` +
		`"reasoning_details":[{"index":0,"text":"a","type":"reasoning.text"},{"index":1,"text":"b","type":"reasoning.text"}],` +
		`"tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":{"city":"Paris"}}}]}}],` +
		`"usage":{"prompt_tokens":120,"completion_tokens":40}}`
)

// wireCase is one wire: a provider built for a server's URL, and its reply.
type wireCase struct {
	name  string
	reply string
	build func(url string, opts ...llmprovider.Option) (llmprovider.Provider, error)
}

func base(url, model string, opts ...llmprovider.Option) []llmprovider.Option {
	return append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModel(model), llmprovider.WithBaseURL(url),
		llmprovider.WithModelProbes(false), catalog.WithCache(new(catalog.Cache))}, opts...)
}

var wires = []wireCase{
	{"messages", messagesReply, func(url string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
		return claude.New(base(url, "claude-sonnet-5", append([]llmprovider.Option{extractDoc()}, opts...)...)...)
	}},
	{"responses", responsesReply, func(url string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
		return openai.New(base(url, "gpt-6-luna", opts...)...)
	}},
	{"interactions", interactionsReply, func(url string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
		return gemini.New(base(url, "gemini-3.7-flash", opts...)...)
	}},
	{"generateContent", googleReply, func(url string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
		return opencode.NewZen(base(url, "gemini-3.7-flash", append(opts, opencode.WithRoute(opencode.RouteGoogle))...)...)
	}},
	{"chat completions", chatReply, func(url string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
		return together.New(base(url, "openai/gpt-oss-120b", opts...)...)
	}},
}

func hi() *llmprovider.Request {
	return &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}
}

// generate is p's response to hi, built for url with opts.
func generate(t *testing.T, w wireCase, url string, opts ...llmprovider.Option) *llmprovider.Response {
	t.Helper()
	p, err := w.build(url, opts...)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Generate(context.Background(), hi())
	if err != nil {
		t.Fatalf("%s: %v", w.name, err)
	}
	return resp
}

// same reports what differs between a plain Generate's response and one read
// over the stream: the output, the id, the finish reason and the usage.
func same(t *testing.T, name string, plain, streamed *llmprovider.Response) {
	t.Helper()
	if !reflect.DeepEqual(plain.Output, streamed.Output) || plain.ID != streamed.ID ||
		plain.FinishReason != streamed.FinishReason || plain.Usage != streamed.Usage {
		t.Errorf("%s: over the stream %+v; want the plain response %+v", name, streamed, plain)
	}
}

// TestNewServer_StreamForm_0088: NewServer answers a plain Generate, which
// names its limit, with the reply, and one read over the stream with the
// reply's stream form; both give the same response on every wire.
func TestNewServer_StreamForm_0088(t *testing.T) {
	for _, w := range wires {
		srv := wiretest.NewServer(t, func(*http.Request) wiretest.Reply { return wiretest.Reply{Body: w.reply} })
		plain := generate(t, w, srv.URL, llmprovider.WithMaxTokens(8192))
		streamed := generate(t, w, srv.URL)
		reqs := srv.Requests()
		if len(reqs) != 2 {
			t.Fatalf("%s: %d request(s); want 2", w.name, len(reqs))
		}
		same(t, w.name, plain, streamed)
	}
}

// TestStreamReplies_0088: a handler's 200 JSON reply reaches a request read
// over the stream as its stream form; an event stream the handler writes and
// flushes itself, a reply with another status, a request that asks for no
// stream, and a hijacked connection pass through.
func TestStreamReplies_0088(t *testing.T) {
	for _, w := range wires {
		srv := httptest.NewServer(wiretest.StreamReplies(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(rw, w.reply)
		})))
		same(t, w.name, generate(t, w, srv.URL, llmprovider.WithMaxTokens(8192)), generate(t, w, srv.URL))
		srv.Close()
	}

	// An event stream the handler writes and flushes passes through as written.
	const stream = "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hello\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"
	srv := httptest.NewServer(wiretest.StreamRepliesFunc(func(rw http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(rw, stream[:20])
		rw.(http.Flusher).Flush()
		_, _ = io.WriteString(rw, stream[20:])
	}))
	defer srv.Close()
	if got := generate(t, wires[4], srv.URL).OutputText(); got != "hello" {
		t.Errorf("a flushed event stream: %q; want hello", got)
	}

	// A refusal keeps its status and body, and nothing is converted.
	refused := httptest.NewServer(wiretest.StreamRepliesFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(rw, `{"error":{"message":"bad","type":"invalid_request_error"}}`)
	}))
	defer refused.Close()
	p, err := wires[4].build(refused.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Generate(context.Background(), hi()); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("a 400: %v; want ErrInvalidRequest", err)
	}

	// A hijacked connection, cut after the head, is a stream cut before its
	// first event.
	cut := httptest.NewServer(wiretest.StreamRepliesFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("Content-Length", "100")
		_, _ = io.WriteString(rw, `{"choices":`)
		conn, buf, err := rw.(http.Hijacker).Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err)
			return
		}
		_, _ = buf.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\n{\"choices\":")
		_ = buf.Flush()
		_ = conn.Close()
	}))
	defer cut.Close()
	if p, err = wires[4].build(cut.URL); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Generate(context.Background(), hi()); !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Errorf("a cut connection: %v; want ErrProviderUnavailable", err)
	}
}

// TestStreamBody_0088: a fake transport's reply reaches a request read over
// the stream as its stream form, and a plain request as it is.
func TestStreamBody_0088(t *testing.T) {
	w := wires[4]
	client := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{},
			Body: io.NopCloser(strings.NewReader(wiretest.StreamBody(r, w.reply))), Request: r}, nil
	})}
	same(t, w.name, generate(t, w, "http://fake.invalid", llmprovider.WithHTTPClient(client), llmprovider.WithMaxTokens(8192)),
		generate(t, w, "http://fake.invalid", llmprovider.WithHTTPClient(client)))
	if got := wiretest.StreamBody(&http.Request{}, "x"); got != "x" {
		t.Errorf("a request with no body: %q; want the reply as it is", got)
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestStreamReply_Errors_0088: a reply that fails is the stream's failure on
// each wire; a reply on no known path, or not a JSON object, has no stream
// form.
func TestStreamReply_Errors_0088(t *testing.T) {
	for _, tc := range []struct {
		name, path, reply string
		wire              int // into wires
		want              error
	}{
		{"messages error", "/v1/messages", `{"type":"error","error":{"type":"overloaded_error","message":"busy"}}`, 0,
			llmprovider.ErrProviderUnavailable},
		{"chat error in a 200", "/chat/completions", `{"error":{"message":"bad","type":"invalid_request_error","code":400}}`, 4,
			llmprovider.ErrInvalidRequest},
		{"responses failed", "/responses", `{"id":"r","status":"failed","error":{"code":"server_error","message":"x"}}`, 1,
			llmprovider.ErrProviderUnavailable},
		{"responses incomplete", "/responses", `{"id":"r","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"}}`, 1,
			llmprovider.ErrIncomplete},
		{"interactions error", "/interactions", `{"error":{"code":503,"message":"busy","status":"UNAVAILABLE"}}`, 2,
			llmprovider.ErrProviderUnavailable},
	} {
		sse, ok := wiretest.StreamReply(tc.path, tc.reply)
		if !ok || !strings.Contains(sse, "data: ") {
			t.Fatalf("%s: %q, %t; want a stream", tc.name, sse, ok)
		}
		srv := httptest.NewServer(wiretest.StreamRepliesFunc(func(rw http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(rw, tc.reply)
		}))
		p, err := wires[tc.wire].build(srv.URL)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.Generate(context.Background(), hi()); !errors.Is(err, tc.want) {
			t.Errorf("%s: %v; want %v", tc.name, err, tc.want)
		}
		srv.Close()
	}
	for _, tc := range []struct{ path, reply string }{
		{"/v1/other", messagesReply}, {"/v1/messages", `{"garbled`}, {"/v1/messages", "data: x\n\n"},
	} {
		if sse, ok := wiretest.StreamReply(tc.path, tc.reply); ok {
			t.Errorf("StreamReply(%q, %q) = %q; want no stream form", tc.path, tc.reply, sse)
		}
	}
}
