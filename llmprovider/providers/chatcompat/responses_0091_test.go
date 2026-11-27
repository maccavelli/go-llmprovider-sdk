package chatcompat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// The tests of 0091-PLAN Phase 2: chatcompat.WithResponses (0091-MADR ST50
// to ST69), over a fake server answering /responses with the shapes probe
// rounds 3 and 7 recorded, redacted: Ollama's reasoning item with its
// summary and encrypted content and a reasoning count of 0 (PR7 (a)), and
// OpenAI's cut (PR10).

const (
	// respMessage is a message item saying hello.
	respMessage = `{"type":"message","id":"msg_1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"hello"}]}`
	// respReasoning is Ollama's reasoning item (PR7 (a)).
	respReasoning = `{"type":"reasoning","id":"rs_1","summary":[{"type":"summary_text","text":"thinking"}],"encrypted_content":"enc-1"}`
	// respCutCall is a call whose arguments do not parse, as a call cut
	// inside them was returned (PR6).
	respCutCall = `{"type":"function_call","id":"fc_1","call_id":"call_cut","name":"get_weather","status":"completed","arguments":"{\"city\":"}`
)

// respReply is a plain Responses reply: its status, "" for none, its output
// count, and its items.
func respReply(status string, outputTokens int, items ...string) string {
	head := `{"id":"resp_1","model":"m",`
	if status != "" {
		head += fmt.Sprintf(`"status":%q,`, status)
	}
	return head + `"output":[` + strings.Join(items, ",") + `],` +
		fmt.Sprintf(`"usage":{"input_tokens":3,"output_tokens":%d,"output_tokens_details":{"reasoning_tokens":0}}}`, outputTokens)
}

// chatAnswer is the Chat Completions reply respServer gives, its text
// "chat", so that no Responses test passes on a reply from that route.
const chatAnswer = `{"id":"cmpl-acme","model":"m","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"chat"}}]}`

// respServer answers /responses with reply, the listing with one model, and
// /chat/completions with chatAnswer.
func respServer(t *testing.T, reply func(n int, r *http.Request) (int, string)) *server {
	t.Helper()
	return newServer(t, func(n int, r *http.Request) (int, string) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/responses"):
			return reply(n, r)
		case strings.HasSuffix(r.URL.Path, "/models"):
			return http.StatusOK, `{"object":"list","data":[{"id":"m","object":"model"}]}`
		}
		return http.StatusOK, chatAnswer
	})
}

// always answers every /responses request with status and body.
func always(status int, body string) func(int, *http.Request) (int, string) {
	return func(int, *http.Request) (int, string) { return status, body }
}

// buildResponses is build with WithResponses.
func buildResponses(t *testing.T, url string, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	return build(t, url, append([]llmprovider.Option{WithResponses()}, opts...)...)
}

// limited is a text request that names its output limit, so Generate is a
// plain request (0088-MADR ST88).
func limited(n int) *llmprovider.Request {
	req := text("hi")
	req.MaxOutputTokens = n
	return req
}

// streamed is Stream's response, or its error, and the items it yielded.
func streamed(p llmprovider.Provider, req *llmprovider.Request) (*llmprovider.Response, []llmprovider.Item, error) {
	var items []llmprovider.Item
	for ev, err := range p.(llmprovider.Streamer).Stream(context.Background(), req) {
		if err != nil {
			return nil, items, err
		}
		switch ev.Type {
		case llmprovider.EventItem:
			items = append(items, ev.Item)
		case llmprovider.EventDone:
			return ev.Response, items, nil
		}
	}
	return nil, items, errors.New("the stream ended with no done event")
}

// paths is the path of each request s saw.
func (s *server) paths() []string {
	var out []string
	for _, r := range s.seen() {
		out = append(out, r.method+" "+r.path)
	}
	return out
}

// cutWarnings counts resp's WarningToolCallCut.
func cutWarnings(resp *llmprovider.Response) int {
	n := 0
	for _, w := range resp.Warnings {
		if w.Kind == llmprovider.WarningToolCallCut {
			n++
		}
	}
	return n
}

// hasCall reports whether items hold a call.
func hasCall(items []llmprovider.Item) bool {
	return slices.ContainsFunc(items, func(item llmprovider.Item) bool {
		_, ok := item.(llmprovider.FunctionCallItem)
		return ok
	})
}

// sseOf frames payloads as an event stream.
func sseOf(payloads ...string) string {
	var b strings.Builder
	for _, p := range payloads {
		b.WriteString("data: " + p + "\n\n")
	}
	return b.String()
}

// TestChatcompat_WithResponses_0091 (ST50): with WithResponses, Generate and
// Stream go to /responses and the listing to /models, unchanged; without it,
// to /chat/completions. A 404 from /responses is returned by its status,
// never resent to /chat/completions.
func TestChatcompat_WithResponses_0091(t *testing.T) {
	for _, tc := range []struct {
		name string
		opt  bool
		want string
	}{{"with", true, "POST /responses"}, {"without", false, "POST /chat/completions"}} {
		t.Run(tc.name, func(t *testing.T) {
			srv := respServer(t, always(http.StatusOK, respReply("completed", 2, respMessage)))
			p := build(t, srv.URL)
			if tc.opt {
				p = buildResponses(t, srv.URL)
			}
			if _, err := p.Generate(context.Background(), text("hi")); err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if _, _, err := streamed(p, text("hi")); err != nil {
				t.Fatalf("Stream: %v", err)
			}
			if _, err := p.(llmprovider.ModelLister).ListModels(context.Background()); err != nil {
				t.Fatalf("ListModels: %v", err)
			}
			want := []string{tc.want, tc.want, "GET /models"}
			if got := srv.paths(); !slices.Equal(got, want) {
				t.Errorf("requests %v; want %v", got, want)
			}
		})
	}
	t.Run("404", func(t *testing.T) {
		srv := respServer(t, always(http.StatusNotFound, `{"error":{"message":"no such route"}}`))
		_, err := buildResponses(t, srv.URL).Generate(context.Background(), limited(100))
		var apiErr *llmprovider.APIError
		if !errors.As(err, &apiErr) || apiErr.Status != http.StatusNotFound {
			t.Fatalf("Generate: %v; want the 404 by its status", err)
		}
		if got := srv.paths(); !slices.Equal(got, []string{"POST /responses"}) {
			t.Errorf("requests %v; want one to /responses, no fallback", got)
		}
	})
}

// TestChatcompat_ResponsesDecode_0091 (ST52; 0092-MADR ST15, ST20): the
// reply is read by the shared Responses decoder, a failure labelled with the
// provider's id; plain and stream on Ollama's shape, and OpenAI's cut
// finishing length; and, streamed, M1's message and M2's call that only the
// terminal output carries, the take logged once through WithLogger.
func TestChatcompat_ResponsesDecode_0091(t *testing.T) {
	ollama := respReply("completed", 9, respReasoning, respMessage)
	wantReasoning := llmprovider.ReasoningItem{Text: "thinking", Encrypted: "enc-1", Format: "responses"}
	check := func(t *testing.T, resp *llmprovider.Response) {
		t.Helper()
		want := []llmprovider.Item{wantReasoning, llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "hello"}}
		if resp.FinishReason != llmprovider.FinishStop || resp.ID != "resp_1" || !slices.Equal(resp.Output, want) {
			t.Errorf("response %+v; want resp_1, stop, %+v", resp, want)
		}
	}
	t.Run("plain", func(t *testing.T) {
		srv := respServer(t, always(http.StatusOK, ollama))
		resp, err := buildResponses(t, srv.URL).Generate(context.Background(), limited(100))
		if err != nil {
			t.Fatal(err)
		}
		check(t, resp)
	})
	t.Run("stream", func(t *testing.T) {
		srv := respServer(t, always(http.StatusOK, ollama))
		resp, _, err := streamed(buildResponses(t, srv.URL), text("hi"))
		if err != nil {
			t.Fatal(err)
		}
		check(t, resp)
	})
	t.Run("openai cut", func(t *testing.T) {
		cut := `{"id":"resp_1","model":"m","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[` +
			`{"type":"message","id":"msg_1","status":"incomplete","content":[{"type":"output_text","text":"hel"}]}],"usage":{"output_tokens":16}}`
		srv := respServer(t, always(http.StatusOK, cut))
		resp, err := buildResponses(t, srv.URL).Generate(context.Background(), limited(16))
		if err != nil || resp.FinishReason != llmprovider.FinishLength || resp.OutputText() != "hel" {
			t.Fatalf("Generate: %+v, %v; want hel, finishing length", resp, err)
		}
	})
	t.Run("label", func(t *testing.T) {
		failed := `{"id":"resp_1","status":"failed","error":{"code":"server_error","message":"boom"}}`
		srv := respServer(t, always(http.StatusOK, failed))
		p := buildResponses(t, srv.URL)
		_, err := p.Generate(context.Background(), limited(100))
		if err == nil || !strings.Contains(err.Error(), string(testID)) || strings.Contains(err.Error(), "responses:") {
			t.Errorf("Generate: %v; want the failure labelled %q", err, testID)
		}
		_, _, err = streamed(p, limited(100))
		if err == nil || !strings.Contains(err.Error(), string(testID)) || strings.Contains(err.Error(), "responses:") {
			t.Errorf("Stream: %v; want the failure labelled %q", err, testID)
		}
	})
	m1 := sseOf(`{"type":"response.created","response":{"id":"resp_1","model":"m"}}`,
		`{"type":"response.output_text.delta","delta":"Hello"}`,
		`{"type":"response.completed","response":{"id":"resp_1","model":"m","output":[{"type":"message","id":"msg_1","content":[{"type":"output_text","text":"Hello"}]}],"usage":{"output_tokens":1}}}`)
	m2 := sseOf(`{"type":"response.created","response":{"id":"resp_1","model":"m"}}`,
		`{"type":"response.output_item.done","item":{"type":"message","id":"msg_1","content":[{"type":"output_text","text":"Hello"}]}}`,
		`{"type":"response.completed","response":{"id":"resp_1","model":"m","output":[{"type":"message","id":"msg_1","content":[{"type":"output_text","text":"Hello"}]},`+
			`{"type":"function_call","id":"fc_1","call_id":"call_1","name":"get_weather","arguments":"{\"city\":\"Paris\"}"}],"usage":{"output_tokens":2}}}`)
	t.Run("M1", func(t *testing.T) {
		var log bytes.Buffer
		srv := respServer(t, always(http.StatusOK, m1))
		resp, _, err := streamed(buildResponses(t, srv.URL,
			llmprovider.WithLogger(slog.New(slog.NewTextHandler(&log, &slog.HandlerOptions{Level: slog.LevelDebug})))), text("hi"))
		if err != nil || resp.OutputText() != "Hello" || resp.FinishReason != llmprovider.FinishStop {
			t.Fatalf("Stream: %+v, %v; want Hello, finishing stop", resp, err)
		}
		if n := strings.Count(log.String(), "items taken from the terminal response"); n != 1 || strings.Contains(log.String(), "Hello") {
			t.Errorf("log %q: %d lines; want one, with no text", log.String(), n)
		}
	})
	t.Run("M2", func(t *testing.T) {
		srv := respServer(t, always(http.StatusOK, m2))
		resp, items, err := streamed(buildResponses(t, srv.URL), text("hi"))
		if err != nil || resp.FinishReason != llmprovider.FinishToolCalls || !hasCall(resp.Output) || !hasCall(items) {
			t.Fatalf("Stream: %+v, %v; want the terminal's call, finishing tool_calls", resp, err)
		}
	})
}

// TestChatcompat_ResponsesErrors_0091 (ST53): an HTTP 401 is ErrAuthFailure;
// a 200 failed whose code is server_error is ErrProviderUnavailable, plain
// and streamed, never an empty success.
func TestChatcompat_ResponsesErrors_0091(t *testing.T) {
	srv := respServer(t, always(http.StatusUnauthorized, `{"error":{"message":"bad key","code":"invalid_api_key"}}`))
	if _, err := buildResponses(t, srv.URL).Generate(context.Background(), limited(100)); !errors.Is(err, llmprovider.ErrAuthFailure) {
		t.Errorf("401: %v; want ErrAuthFailure", err)
	}
	failed := `{"id":"resp_1","status":"failed","error":{"code":"server_error","message":"boom"}}`
	srv = respServer(t, always(http.StatusOK, failed))
	p := buildResponses(t, srv.URL)
	if resp, err := p.Generate(context.Background(), limited(100)); !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Errorf("200 failed: %+v, %v; want ErrProviderUnavailable", resp, err)
	}
	if resp, _, err := streamed(p, limited(100)); !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Errorf("200 failed, streamed: %+v, %v; want ErrProviderUnavailable", resp, err)
	}
}

// TestChatcompat_ResponsesRefusal_0091 (ST54): OpenAI's invalid_value
// refusal of the effort sent is learned through the Responses matcher, and
// the request sent again lower; a refusal in another form is returned.
func TestChatcompat_ResponsesRefusal_0091(t *testing.T) {
	refusal := `{"error":{"message":"Invalid value: 'max'. Supported values are: 'low', 'medium', and 'high'.",` +
		`"type":"invalid_request_error","param":"reasoning.effort","code":"invalid_value"}}`
	srv := respServer(t, func(n int, _ *http.Request) (int, string) {
		if n == 1 {
			return http.StatusBadRequest, refusal
		}
		return http.StatusOK, respReply("completed", 2, respMessage)
	})
	req := limited(100)
	req.Reasoning = &llmprovider.Reasoning{Effort: llmprovider.EffortMax}
	resp, err := buildResponses(t, srv.URL).Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate: %v; want the refusal learned and the request sent again", err)
	}
	var efforts []any
	for _, r := range srv.seen() {
		efforts = append(efforts, r.body["reasoning"])
	}
	learned := slices.ContainsFunc(resp.Warnings, func(w llmprovider.Warning) bool { return w.Kind == llmprovider.WarningReasoningLearned })
	if len(efforts) != 2 || fmt.Sprint(efforts[1]) != "map[effort:high]" || !learned {
		t.Errorf("efforts sent %v, warnings %+v; want max then high, and WarningReasoningLearned", efforts, resp.Warnings)
	}

	other := respServer(t, always(http.StatusBadRequest, `{"error":{"message":"no","type":"invalid_request_error","code":"other"}}`))
	if _, err := buildResponses(t, other.URL).Generate(context.Background(), req); err == nil || len(other.seen()) != 1 {
		t.Errorf("another form: %v after %d requests; want it returned after one", err, len(other.seen()))
	}
}

// outputLayer is a caller layer stating a model's output limit.
type outputLayer int

func (outputLayer) Name() string { return "test" }

func (l outputLayer) Model(_ llmprovider.ProviderID, id string) (catalog.ModelInfo, bool) {
	return catalog.ModelInfo{ID: id, MaxOutputTokens: int(l)}.Known(catalog.FieldMaxOutputTokens), true
}

// TestChatcompat_ResponsesCut_0091 (ST55, ST33, ST34, ST65, ST69): a reply
// finished as completed whose output_tokens reached the limit sent is a cut:
// it finishes length, a call whose arguments do not parse dropped and named,
// and on a stream never yielded; under the limit it finishes as it came. The
// limit is the one the body carried. With no limit sent nothing is read as
// cut. A reply with no status, or an unknown one, at the limit is a cut too.
// On a stream at the limit, an announced call never delivered is the cut
// call; under it, the stream fails naming item_not_finished.
func TestChatcompat_ResponsesCut_0091(t *testing.T) {
	plain := func(t *testing.T, reply string, req *llmprovider.Request, opts ...llmprovider.Option) *llmprovider.Response {
		t.Helper()
		srv := respServer(t, always(http.StatusOK, reply))
		resp, err := buildResponses(t, srv.URL, opts...).Generate(context.Background(), req)
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		return resp
	}
	for _, tc := range []struct {
		name   string
		reply  string
		finish llmprovider.FinishReason
	}{
		{"text at the limit", respReply("completed", 16, respMessage), llmprovider.FinishLength},
		{"text under the limit", respReply("completed", 15, respMessage), llmprovider.FinishStop},
		{"no status at the limit", respReply("", 16, respMessage), llmprovider.FinishLength},
		{"unknown status at the limit", respReply("in_progress", 16, respMessage), llmprovider.FinishLength},
	} {
		t.Run("plain/"+tc.name, func(t *testing.T) {
			if resp := plain(t, tc.reply, limited(16)); resp.FinishReason != tc.finish || resp.OutputText() != "hello" {
				t.Errorf("%+v; want hello, finishing %s", resp, tc.finish)
			}
		})
	}
	t.Run("plain/a call cut at the limit", func(t *testing.T) {
		resp := plain(t, respReply("completed", 16, respMessage, respCutCall), limited(16))
		if resp.FinishReason != llmprovider.FinishLength || hasCall(resp.Output) || cutWarnings(resp) != 1 {
			t.Errorf("%+v; want length, the call dropped and named once", resp)
		}
	})
	t.Run("plain/the limit the body carried", func(t *testing.T) {
		// WithMaxTokens(1000) is cut to the model's stated 16, which is sent.
		resp := plain(t, respReply("completed", 16, respMessage), text("hi"), llmprovider.WithMaxTokens(1000),
			catalog.WithModelLayer(outputLayer(16)))
		if resp.FinishReason != llmprovider.FinishLength {
			t.Errorf("%+v; want length at the 16 sent", resp)
		}
	})
	t.Run("no limit sent", func(t *testing.T) {
		resp := plain(t, respReply("completed", 1000000, respMessage), text("hi"))
		if resp.FinishReason != llmprovider.FinishStop {
			t.Errorf("%+v; want stop: no limit, no check", resp)
		}
	})
	stream := func(t *testing.T, reply string) (*llmprovider.Response, []llmprovider.Item, error) {
		t.Helper()
		srv := respServer(t, always(http.StatusOK, reply))
		return streamed(buildResponses(t, srv.URL), limited(16))
	}
	t.Run("stream/text at the limit", func(t *testing.T) {
		resp, _, err := stream(t, respReply("completed", 16, respMessage))
		if err != nil || resp.FinishReason != llmprovider.FinishLength {
			t.Errorf("%+v, %v; want length", resp, err)
		}
	})
	t.Run("stream/text under the limit", func(t *testing.T) {
		resp, _, err := stream(t, respReply("completed", 15, respMessage))
		if err != nil || resp.FinishReason != llmprovider.FinishStop {
			t.Errorf("%+v, %v; want stop", resp, err)
		}
	})
	t.Run("stream/a call cut at the limit", func(t *testing.T) {
		resp, items, err := stream(t, sseOf(
			`{"type":"response.created","response":{"id":"resp_1","model":"m"}}`,
			`{"type":"response.output_item.done","item":`+respMessage+`}`,
			`{"type":"response.output_item.added","item":{"type":"function_call","id":"fc_1","call_id":"call_cut","name":"get_weather"}}`,
			`{"type":"response.output_item.done","item":`+respCutCall+`}`,
			`{"type":"response.completed","response":{"id":"resp_1","model":"m","usage":{"output_tokens":16}}}`))
		if err != nil || resp.FinishReason != llmprovider.FinishLength || hasCall(items) || hasCall(resp.Output) || cutWarnings(resp) != 1 {
			t.Errorf("%+v, items %+v, %v; want length, the call never yielded, named once", resp, items, err)
		}
	})
	announced := func(outputTokens int) string {
		return sseOf(`{"type":"response.created","response":{"id":"resp_1","model":"m"}}`,
			`{"type":"response.output_item.done","item":`+respMessage+`}`,
			`{"type":"response.output_item.added","item":{"type":"function_call","id":"fc_2","call_id":"call_lost","name":"get_weather"}}`,
			fmt.Sprintf(`{"type":"response.completed","response":{"id":"resp_1","model":"m","output":[%s],"usage":{"output_tokens":%d}}}`,
				respMessage, outputTokens))
	}
	t.Run("stream/an announced call lost at the limit", func(t *testing.T) {
		resp, _, err := stream(t, announced(16))
		if err != nil || resp.FinishReason != llmprovider.FinishLength || cutWarnings(resp) != 1 ||
			!strings.Contains(resp.Warnings[0].Detail, "call_lost") {
			t.Errorf("%+v, %v; want length and one WarningToolCallCut naming call_lost", resp, err)
		}
	})
	t.Run("stream/an announced call lost under the limit", func(t *testing.T) {
		_, _, err := stream(t, announced(15))
		if !errors.Is(err, llmprovider.ErrProviderUnavailable) || !strings.Contains(fmt.Sprint(err), "item_not_finished") {
			t.Errorf("%v; want ErrProviderUnavailable naming item_not_finished", err)
		}
	})
}

// TestChatcompat_ResponsesRefusalAtLimit_0091 (ST70, "The refusal wins
// (Recommended)"): a refusal finishes content_filter, at the limit sent as
// under it, plain and streamed: the refusal is read before the limit check.
func TestChatcompat_ResponsesRefusalAtLimit_0091(t *testing.T) {
	refusal := `{"type":"message","id":"msg_1","role":"assistant","status":"completed","content":[{"type":"refusal","refusal":"I cannot help with that."}]}`
	for _, tc := range []struct {
		name   string
		tokens int
	}{{"at the limit", 16}, {"under the limit", 15}} {
		for _, stream := range []bool{false, true} {
			name := tc.name
			if stream {
				name += "/stream"
			}
			t.Run(name, func(t *testing.T) {
				srv := respServer(t, always(http.StatusOK, respReply("completed", tc.tokens, refusal)))
				p := buildResponses(t, srv.URL)
				var resp *llmprovider.Response
				var err error
				if stream {
					resp, _, err = streamed(p, limited(16))
				} else {
					resp, err = p.Generate(context.Background(), limited(16))
				}
				if err != nil || resp.FinishReason != llmprovider.FinishContentFilter {
					t.Errorf("%+v, %v; want content_filter", resp, err)
				}
			})
		}
	}
	// ST74, "Refusal, plus a 'call cut' warning (Recommended)": with a call
	// announced and never delivered, the refusal at the limit still finishes
	// content_filter, the call named in a WarningToolCallCut, with no error.
	t.Run("at the limit, a call lost/stream", func(t *testing.T) {
		srv := respServer(t, always(http.StatusOK, sseOf(
			`{"type":"response.created","response":{"id":"resp_1","model":"m"}}`,
			`{"type":"response.output_item.added","item":{"type":"function_call","id":"fc_2","call_id":"call_lost","name":"get_weather"}}`,
			`{"type":"response.output_item.done","item":`+refusal+`}`,
			`{"type":"response.completed","response":{"id":"resp_1","model":"m","output":[`+refusal+`],"usage":{"output_tokens":16}}}`)))
		resp, _, err := streamed(buildResponses(t, srv.URL), limited(16))
		if err != nil || resp.FinishReason != llmprovider.FinishContentFilter || cutWarnings(resp) != 1 ||
			!strings.Contains(resp.Warnings[0].Detail, "call_lost") {
			t.Errorf("%+v, %v; want content_filter and one WarningToolCallCut naming call_lost", resp, err)
		}
	})
}

// TestChatcompat_ResponsesReasoningText_0091 (ST63): reasoning a server sends
// only as reasoning_text content is shown, and not sent back on the next
// turn, carrying no encrypted content.
func TestChatcompat_ResponsesReasoningText_0091(t *testing.T) {
	reasoning := `{"type":"reasoning","id":"rs_1","summary":[],"content":[{"type":"reasoning_text","text":"thinking aloud"}]}`
	srv := respServer(t, always(http.StatusOK, respReply("completed", 9, reasoning, respMessage)))
	p := buildResponses(t, srv.URL)
	resp, err := p.Generate(context.Background(), limited(100))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(resp.Output, llmprovider.Item(llmprovider.ReasoningItem{Text: "thinking aloud", Format: "responses"})) {
		t.Fatalf("output %+v; want the reasoning text shown", resp.Output)
	}
	next := limited(100)
	next.Input = append(append(next.Input, resp.Output...), llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "and?"})
	if _, err := p.Generate(context.Background(), next); err != nil {
		t.Fatal(err)
	}
	seen := srv.seen()
	if body := fmt.Sprint(seen[len(seen)-1].body["input"]); strings.Contains(body, "reasoning") || strings.Contains(body, "thinking aloud") {
		t.Errorf("the next request's input %s; want no reasoning item", body)
	}
}

// factsLayer is a caller layer stating a model's input and efforts.
type factsLayer struct{}

func (factsLayer) Name() string { return "test" }

func (factsLayer) Model(_ llmprovider.ProviderID, id string) (catalog.ModelInfo, bool) {
	info := catalog.ModelInfo{ID: id, Input: []catalog.Modality{catalog.ModalityText}}
	info.Reasoning.Efforts = []llmprovider.Effort{llmprovider.EffortLow, llmprovider.EffortMedium}
	return info.Known(catalog.FieldInput, catalog.FieldReasoningEfforts), true
}

// TestChatcompat_ResponsesFacts_0091 (ST51, restated): the Responses body is
// built from the model's cached facts, as the Chat body is: an image to a
// model that takes none is refused before the network, and an effort outside
// the facts' list is stepped down, with WarningEffortDegraded.
func TestChatcompat_ResponsesFacts_0091(t *testing.T) {
	srv := respServer(t, always(http.StatusOK, respReply("completed", 2, respMessage)))
	p := buildResponses(t, srv.URL, catalog.WithModelLayer(factsLayer{}))
	img := limited(100)
	img.Input = append(img.Input, llmprovider.ImageItem{MediaType: "image/png", Data: []byte{0x89, 'P', 'N', 'G'}})
	if _, err := p.Generate(context.Background(), img); !errors.Is(err, llmprovider.ErrUnsupported) || len(srv.seen()) != 0 {
		t.Errorf("an image: %v after %d requests; want ErrUnsupported before the network", err, len(srv.seen()))
	}
	req := limited(100)
	req.Reasoning = &llmprovider.Reasoning{Effort: llmprovider.EffortHigh}
	resp, err := p.Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	degraded := slices.ContainsFunc(resp.Warnings, func(w llmprovider.Warning) bool { return w.Kind == llmprovider.WarningEffortDegraded })
	if sent := fmt.Sprint(srv.seen()[0].body["reasoning"]); sent != "map[effort:medium]" || !degraded {
		t.Errorf("reasoning sent %s, warnings %+v; want medium and WarningEffortDegraded", sent, resp.Warnings)
	}
}

// TestChatcompat_ResponsesLimitRefusal_0091 (ST56): a refusal of a given
// limit in the Chat Completions "supports at most" form is not read on the
// Responses path: it is returned after one request, and no cap is kept.
func TestChatcompat_ResponsesLimitRefusal_0091(t *testing.T) {
	refusal := `{"error":{"message":"This model supports at most 4096 completion tokens, whereas you provided 20000.",` +
		`"type":"invalid_request_error","param":"max_output_tokens","code":"invalid_value"}}`
	srv := respServer(t, always(http.StatusBadRequest, refusal))
	_, err := buildResponses(t, srv.URL, llmprovider.WithMaxTokens(20000)).Generate(context.Background(), text("hi"))
	var apiErr *llmprovider.APIError
	if !errors.As(err, &apiErr) || apiErr.Observed.Tokens != 0 || len(srv.seen()) != 1 {
		t.Errorf("%v (observed %+v) after %d requests; want the refusal returned, nothing learned", err, apiErr, len(srv.seen()))
	}
}

// TestChatcompat_ResponsesNoContinuation_0091 (ST58, ST23): Continuation stays
// Unsupported, so a PreviousResponseID is refused before the network; a
// Generate that names no limit is read over the stream.
func TestChatcompat_ResponsesNoContinuation_0091(t *testing.T) {
	srv := respServer(t, always(http.StatusOK, respReply("completed", 2, respMessage)))
	p := buildResponses(t, srv.URL)
	req := text("hi")
	req.PreviousResponseID = "resp_0"
	if _, err := p.Generate(context.Background(), req); !errors.Is(err, llmprovider.ErrUnsupported) || len(srv.seen()) != 0 {
		t.Errorf("continuation: %v after %d requests; want ErrUnsupported before the network", err, len(srv.seen()))
	}
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatal(err)
	}
	if first := srv.seen()[0]; first.path != "/responses" || first.body["stream"] != true {
		t.Errorf("a Generate with no limit sent %s %v; want a stream request to /responses", first.path, first.body)
	}
}

// TestChatcompat_ResponsesUsage_0091 (ST13): a reasoning count of 0 with a
// reasoning item, as round 3 recorded, is passed through as 0.
func TestChatcompat_ResponsesUsage_0091(t *testing.T) {
	srv := respServer(t, always(http.StatusOK, respReply("completed", 9, respReasoning, respMessage)))
	resp, err := buildResponses(t, srv.URL).Generate(context.Background(), limited(100))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Usage.ReasoningTokens != 0 || resp.Usage.OutputTokens != 9 {
		t.Errorf("usage %+v; want 9 output, 0 reasoning, as reported", resp.Usage)
	}
}

// TestChatcompat_ResponsesDone_0091 (ST16): a literal [DONE] after the
// terminal event, as OpenRouter sends (PR2), ends a completed stream and a
// cut one as their terminal events do: the whole stream is read, and nothing
// follows its done event, no error either. Two guards protect it, the reader
// stopping at the terminal event and the [DONE] skip (0091-MADR ST16, D5).
func TestChatcompat_ResponsesDone_0091(t *testing.T) {
	for _, tc := range []struct {
		name, last string
		finish     llmprovider.FinishReason
	}{
		{"completed", `{"type":"response.completed","response":{"id":"resp_1","model":"m","usage":{"output_tokens":2}}}`, llmprovider.FinishStop},
		{"incomplete", `{"type":"response.incomplete","response":{"id":"resp_1","model":"m","incomplete_details":{"reason":"max_output_tokens"}}}`,
			llmprovider.FinishLength},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := respServer(t, always(http.StatusOK, sseOf(`{"type":"response.created","response":{"id":"resp_1","model":"m"}}`,
				`{"type":"response.output_item.done","item":`+respMessage+`}`, tc.last, `[DONE]`)))
			var resp *llmprovider.Response
			var after []string
			p := buildResponses(t, srv.URL).(llmprovider.Streamer)
			for ev, err := range p.Stream(context.Background(), limited(100)) {
				switch {
				case resp != nil:
					after = append(after, fmt.Sprintf("%s %v", ev.Type, err))
				case err != nil:
					t.Fatalf("Stream: %v", err)
				case ev.Type == llmprovider.EventDone:
					resp = ev.Response
				}
			}
			if resp == nil || resp.FinishReason != tc.finish || resp.OutputText() != "hello" || len(after) > 0 {
				t.Errorf("%+v, then %v; want hello, finishing %s, and nothing after it", resp, after, tc.finish)
			}
		})
	}
}

// TestChatcompat_ResponsesErrorShapes_0091 (ST19): the probes' error shapes,
// codes only: Kilo's 401 with a top-level error_type, and a 400 whose error
// is a string, each classified by its status.
func TestChatcompat_ResponsesErrorShapes_0091(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		kind   error
	}{
		{"401 with error_type", http.StatusUnauthorized, `{"error":{"code":"INVALID_TOKEN","message":"x"},"error_type":"auth"}`, llmprovider.ErrAuthFailure},
		{"400 with a string", http.StatusBadRequest, `{"error":"x"}`, llmprovider.ErrInvalidRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := respServer(t, always(tc.status, tc.body))
			_, err := buildResponses(t, srv.URL).Generate(context.Background(), limited(100))
			if !errors.Is(err, tc.kind) || errors.Is(err, llmprovider.ErrProviderUnavailable) {
				t.Errorf("%v; want %v", err, tc.kind)
			}
		})
	}
}

// TestChatcompat_ResponsesListingProbe_0091 (ST61): a listing probe under
// WithModelProbes(true) goes through Generate, so to /responses with the
// option and to /chat/completions without.
func TestChatcompat_ResponsesListingProbe_0091(t *testing.T) {
	hello := `{"id":"resp_1","model":"m","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"Hello"}]}]}`
	for _, tc := range []struct {
		name string
		opt  bool
		want string
	}{{"with", true, "/responses"}, {"without", false, "/chat/completions"}} {
		t.Run(tc.name, func(t *testing.T) {
			srv := respServer(t, always(http.StatusOK, hello))
			opts := []llmprovider.Option{llmprovider.WithModelProbes(true)}
			if tc.opt {
				opts = append(opts, WithResponses())
			}
			if _, err := build(t, srv.URL, opts...).(llmprovider.ModelLister).ListModels(context.Background()); err != nil {
				t.Fatal(err)
			}
			var probes []string
			for _, r := range srv.seen() {
				if r.method == http.MethodPost {
					probes = append(probes, r.path)
				}
			}
			if len(probes) != 1 || probes[0] != tc.want {
				t.Errorf("probes %v; want one to %s", probes, tc.want)
			}
		})
	}
}
