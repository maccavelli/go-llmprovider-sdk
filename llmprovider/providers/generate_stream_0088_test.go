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
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// 0088-MADR ST88 and ST88a to ST88d (OQ15 (1), "Over the stream
// (Recommended)"): a plain Generate whose output limit the SDK chose or left
// out is read over the service's stream and returned whole; a Generate that
// names its own limit stays a plain request. Built in 0089-PLAN's merged
// phase, layer D. Every reply is this file's own; none is a recorded one.

// overStreamServer answers each POST with what its reply function returns
// for the POST's index and whether its body asks for a stream, and records
// each POST's body.
type overStreamServer struct {
	*httptest.Server
	mu     sync.Mutex
	bodies []map[string]any
}

func newOverStreamServer(t *testing.T, reply func(n int, stream bool, w http.ResponseWriter)) *overStreamServer {
	t.Helper()
	s := &overStreamServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		reply(n, body["stream"] == true, w)
	}))
	t.Cleanup(s.Close)
	return s
}

// posts is every POST body, in order.
func (s *overStreamServer) posts() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]map[string]any(nil), s.bodies...)
}

// streamed reports whether body asked for a stream.
func streamed(body map[string]any) bool { return body["stream"] == true }

// limitIn is body's output limit, from the JSON number it was decoded as.
func limitIn(body map[string]any) (int, bool) {
	for _, key := range []string{"max_tokens", "max_output_tokens"} {
		if n, ok := body[key].(float64); ok {
			return int(n), true
		}
	}
	return 0, false
}

// plainMessages is a whole Messages reply saying "hello".
const plainMessages = `{"id":"msg_plain","type":"message","role":"assistant","stop_reason":"end_turn",` +
	`"content":[{"type":"text","text":"hello"}],"usage":{"input_tokens":3,"output_tokens":2}}`

// overStreamProvider builds id for srv with opts, no metadata fetch, no
// probe and a cache of its own.
func overStreamProvider(t *testing.T, id llmprovider.ProviderID, model string, srv *overStreamServer,
	opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := New(id, append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModel(model),
		llmprovider.WithBaseURL(srv.URL), llmprovider.WithModelProbes(false),
		testCache(t)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// overStreamHi is a request with one user message and no limit of its own.
func overStreamHi() *llmprovider.Request {
	return &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}
}

// TestGenerateOverStream_0088 (ST88, ST88e): Claude with a stated limit of
// 128,000 and no caller limit sends a request asking for the stream, with
// that limit, and returns the whole response; OpenAI with a key, which sends
// no limit, asks for the stream too; a caller's WithMaxTokens(4000), or the
// request's own MaxOutputTokens, keeps a plain request.
func TestGenerateOverStream_0088(t *testing.T) {
	ctx := context.Background()
	const claudeModel = "claude-sonnet-5"
	for _, tc := range []struct {
		name   string
		id     llmprovider.ProviderID
		model  string
		opts   []llmprovider.Option
		req    *llmprovider.Request
		stream bool
		limit  int // 0: none sent
	}{
		{"claude, a stated 128,000", llmprovider.ProviderClaude, claudeModel,
			[]llmprovider.Option{catalog.WithModelLayer(limitLayer{claudeModel, 128000})}, overStreamHi(), true, 128000},
		{"openai with a key, nothing sent", llmprovider.ProviderOpenAI, "gpt-6-luna", nil, overStreamHi(), true, 0},
		{"claude, WithMaxTokens(4000)", llmprovider.ProviderClaude, claudeModel,
			[]llmprovider.Option{catalog.WithModelLayer(limitLayer{claudeModel, 128000}), llmprovider.WithMaxTokens(4000)},
			overStreamHi(), false, 4000},
		{"claude, the request's MaxOutputTokens 3000", llmprovider.ProviderClaude, claudeModel,
			[]llmprovider.Option{catalog.WithModelLayer(limitLayer{claudeModel, 128000})},
			&llmprovider.Request{MaxOutputTokens: 3000, Input: overStreamHi().Input}, false, 3000},
	} {
		srv := newOverStreamServer(t, func(_ int, stream bool, w http.ResponseWriter) {
			switch {
			case tc.id == llmprovider.ProviderOpenAI && stream:
				_, _ = io.WriteString(w, wirecase.SSEReply)
			case tc.id == llmprovider.ProviderOpenAI:
				_, _ = io.WriteString(w, wirecase.ResponsesReply)
			case stream:
				_, _ = io.WriteString(w, messagesStream)
			default:
				_, _ = io.WriteString(w, plainMessages)
			}
		})
		resp, err := overStreamProvider(t, tc.id, tc.model, srv, tc.opts...).Generate(ctx, tc.req)
		if err != nil || resp.OutputText() != "hello" {
			t.Fatalf("%s: Generate = %v, %v; want the whole response, hello", tc.name, resp, err)
		}
		posts := srv.posts()
		if len(posts) != 1 {
			t.Fatalf("%s: %d request(s); want 1", tc.name, len(posts))
		}
		got, sent := limitIn(posts[0])
		if streamed(posts[0]) != tc.stream || got != tc.limit || sent != (tc.limit > 0) {
			t.Errorf("%s: stream %t, limit %d (sent %t); want stream %t, limit %d", tc.name, streamed(posts[0]), got, sent,
				tc.stream, tc.limit)
		}
	}
}

// TestGenerateOverStream_Retry_0088 (ST88a; 0028-MADR D-H1): under WithRetry,
// a Generate read over the stream whose opening fails with 503 is retried, as
// a plain one is; one whose stream fails after a text event is not resent.
func TestGenerateOverStream_Retry_0088(t *testing.T) {
	ctx := context.Background()
	const model = "claude-sonnet-5"
	policy := llmprovider.RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}
	busy := `{"type":"error","error":{"type":"overloaded_error","message":"busy"}}`
	cut := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"model\":\"m\"}}\n\n" +
		"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hel\"}}\n\n"

	srv := newOverStreamServer(t, func(n int, stream bool, w http.ResponseWriter) {
		switch {
		case n == 0:
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, busy)
		case stream:
			_, _ = io.WriteString(w, messagesStream)
		default:
			_, _ = io.WriteString(w, plainMessages)
		}
	})
	p := llmprovider.WithRetry(overStreamProvider(t, llmprovider.ProviderClaude, model, srv,
		catalog.WithModelLayer(limitLayer{model, 128000})), policy)
	resp, err := p.Generate(ctx, overStreamHi())
	posts := srv.posts()
	if err != nil || resp.OutputText() != "hello" || len(posts) != 2 || !streamed(posts[0]) || !streamed(posts[1]) {
		t.Errorf("a 503 at the opening: %d request(s), err %v; want it retried once over the stream, then hello",
			len(posts), err)
	}

	srv = newOverStreamServer(t, func(_ int, stream bool, w http.ResponseWriter) {
		if stream {
			_, _ = io.WriteString(w, cut)
			return
		}
		_, _ = io.WriteString(w, plainMessages[:40]) // a plain reply cut short
	})
	p = llmprovider.WithRetry(overStreamProvider(t, llmprovider.ProviderClaude, model, srv,
		catalog.WithModelLayer(limitLayer{model, 128000})), policy)
	resp, err = p.Generate(ctx, overStreamHi())
	posts = srv.posts()
	if err == nil || resp != nil || len(posts) != 1 || !streamed(posts[0]) {
		t.Errorf("a stream failing after a text event: %d request(s), resp %v, err %v; want one streamed send and "+
			"the failure returned, not resent", len(posts), resp, err)
	}
}

// TestGenerateOverStream_ReplyLimit_0088 (ST88b): a Generate read over the
// stream whose text events add up to more than the streamed answer's bound,
// each event within the per-event bound, is ErrIncomplete. The bound is 32
// MiB on every wire since 0091-MADR ST71 ("Every provider, one bound
// (Recommended)"), no longer a plain reply's 16 MiB: 33 MiB is
// ErrIncomplete, and 17 MiB, over the old bound, is returned whole.
func TestGenerateOverStream_ReplyLimit_0088(t *testing.T) {
	const model = "claude-sonnet-5"
	for _, mib := range []int{17, 33} {
		srv := newOverStreamServer(t, func(_ int, stream bool, w http.ResponseWriter) { writeMiBStream(w, stream, mib) })
		resp, err := overStreamProvider(t, llmprovider.ProviderClaude, model, srv,
			catalog.WithModelLayer(limitLayer{model, 128000})).Generate(context.Background(), overStreamHi())
		n := 0
		if resp != nil {
			n = len(resp.OutputText())
		}
		switch over := mib > 32; {
		case over && (!errors.Is(err, llmprovider.ErrIncomplete) || resp != nil):
			t.Errorf("%d MiB of text events: a response of %d bytes, err %v; want ErrIncomplete and no response", mib, n, err)
		case !over && (err != nil || n != mib<<20):
			t.Errorf("%d MiB of text events: a response of %d bytes, err %v; want it whole", mib, n, err)
		}
	}
}

// writeMiBStream answers a stream request with mib MiB of Messages text
// events, and any other with plainMessages.
func writeMiBStream(w http.ResponseWriter, stream bool, mib int) {
	if !stream {
		_, _ = io.WriteString(w, plainMessages)
		return
	}
	chunk := strings.Repeat("a", 1<<20)
	_, _ = io.WriteString(w, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"model\":\"m\"}}\n\n"+
		"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
	for range mib {
		_, _ = fmt.Fprintf(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,"+
			"\"delta\":{\"type\":\"text_delta\",\"text\":%q}}\n\n", chunk)
	}
	_, _ = io.WriteString(w, "event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n"+
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\n"+
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
}

// TestGenerateOverStream_NoUsage_0088 (ST88c): a server that sends no usage
// chunk in the stream, though the request asks for one, leaves the usage
// zero, as a Stream reports it.
func TestGenerateOverStream_NoUsage_0088(t *testing.T) {
	srv := newOverStreamServer(t, func(_ int, stream bool, w http.ResponseWriter) {
		if stream {
			_, _ = io.WriteString(w, chatStream)
			return
		}
		_, _ = io.WriteString(w, wirecase.ChatReply)
	})
	resp, err := overStreamProvider(t, llmprovider.ProviderTogether, "openai/gpt-oss-120b", srv).
		Generate(context.Background(), overStreamHi())
	if err != nil || resp.OutputText() != "hello" {
		t.Fatalf("Generate = %v, %v; want hello", resp, err)
	}
	if resp.Usage != (llmprovider.Usage{}) {
		t.Errorf("Usage = %+v; want zero: the stream sent none", resp.Usage)
	}
}

// TestGenerateOverStream_ErrorIn200_0088 (ST88d; 0088-MADR A5, C39, C89; 0089
// DL35): a 200 whose first payload is an error is classified as Stream
// classifies it. On Kilo, which sends its discovered 16,384, C39's window form
// there is resent once over the stream at the room it states, 9,152, and the
// whole response returned; on Together, with nothing stated and so nothing
// sent, it is returned, classified, after one send: never an empty answer.
func TestGenerateOverStream_ErrorIn200_0088(t *testing.T) {
	ctx := context.Background()
	const (
		kiloModel = "openai/gpt-6-luna"
		window    = "This endpoint's maximum context length is 32768 tokens. However, you requested about 40000 tokens."
	)
	inStream := fmt.Sprintf("data: {\"error\":{\"message\":%q,\"type\":\"invalid_request_error\",\"code\":400}}\n\n", window)
	inPlain := fmt.Sprintf(`{"error":{"message":%q,"type":"invalid_request_error","code":400}}`, window)
	for _, tc := range []struct {
		name   string
		id     llmprovider.ProviderID
		model  string
		opts   []llmprovider.Option
		resent int // 0: one send, the error returned
	}{
		{"kilo, its discovered 16,384", llmprovider.ProviderKilo, kiloModel,
			[]llmprovider.Option{catalog.WithModelLayer(limitLayer{kiloModel, 16384})}, 9152},
		{"together, nothing sent", llmprovider.ProviderTogether, "openai/gpt-oss-120b", nil, 0},
	} {
		srv := newOverStreamServer(t, func(n int, stream bool, w http.ResponseWriter) {
			switch {
			case n == 0 && stream:
				_, _ = io.WriteString(w, inStream)
			case n == 0:
				_, _ = io.WriteString(w, inPlain)
			case stream:
				_, _ = io.WriteString(w, chatStream)
			default:
				_, _ = io.WriteString(w, wirecase.ChatReply)
			}
		})
		resp, err := overStreamProvider(t, tc.id, tc.model, srv, tc.opts...).Generate(ctx, overStreamHi())
		posts := srv.posts()
		if len(posts) == 0 || !streamed(posts[0]) {
			t.Fatalf("%s: %d request(s); want the first asking for the stream", tc.name, len(posts))
		}
		if tc.resent == 0 {
			if !errors.Is(err, llmprovider.ErrInvalidRequest) || resp != nil || len(posts) != 1 {
				t.Errorf("%s: %d send(s), resp %v, err %v; want one send and the error, classified", tc.name, len(posts),
					resp, err)
			}
			continue
		}
		if err != nil || resp.OutputText() != "hello" || len(posts) != 2 {
			t.Fatalf("%s: %d send(s), err %v; want a resend, then hello", tc.name, len(posts), err)
		}
		if got, _ := limitIn(posts[1]); !streamed(posts[1]) || got != tc.resent {
			t.Errorf("%s: resent with stream %t at %d; want the stream at %d, the room the window leaves", tc.name,
				streamed(posts[1]), got, tc.resent)
		}
	}
}
