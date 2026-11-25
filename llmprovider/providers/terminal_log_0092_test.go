package providers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// 0092-PLAN Phase 1 (0092-MADR ST11, ST20): the Responses stream reader as
// the providers that use it build it. Every reply is this file's own, in the
// shapes the MADR measured; none is a recorded one.

// t92Event is one server-sent event of kind with its other fields.
func t92Event(kind, fields string) string {
	return "event: " + kind + "\ndata: {\"type\":\"" + kind + "\"" + fields + "}\n\n"
}

// t92Stream is a Responses stream: created, the events between, and a
// response.completed whose output is the items given.
func t92Stream(between string, output ...string) string {
	return t92Event("response.created", `,"response":{"id":"resp_t","model":"gpt-5","status":"in_progress","output":[]}`) +
		between +
		t92Event("response.completed", `,"response":{"id":"resp_t","model":"gpt-5","status":"completed","output":[`+
			strings.Join(output, ",")+`],"usage":{"input_tokens":3,"output_tokens":5}}`)
}

const (
	t92Msg       = `{"id":"msg_1","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Hello"}]}`
	t92MsgAdded  = `{"id":"msg_1","type":"message","role":"assistant","status":"in_progress","content":[]}`
	t92CallAdded = `{"id":"fc_1","type":"function_call","status":"in_progress","call_id":"call_1","name":"lookup","arguments":""}`
)

// t92Hello is a message announced, its delta, and its .done.
var t92Hello = t92Event("response.output_item.added", `,"output_index":0,"item":`+t92MsgAdded) +
	t92Event("response.output_text.delta", `,"output_index":0,"content_index":0,"delta":"Hello"`) +
	t92Event("response.output_item.done", `,"output_index":0,"item":`+t92Msg)

// t92M1 is M1's shape: the message's delta, no .done, the message only in
// the terminal output.
var t92M1 = t92Stream(t92Event("response.output_item.added", `,"output_index":0,"item":`+t92MsgAdded)+
	t92Event("response.output_text.delta", `,"output_index":0,"content_index":0,"delta":"Hello"`), t92Msg)

// t92Provider is route's provider over a server answering every stream with
// stream, with opts.
func t92Provider(t *testing.T, id llmprovider.ProviderID, model, stream string, opts ...llmprovider.Option) (llmprovider.Provider, *cutServer) {
	t.Helper()
	srv := newCutServer(t, `{}`, stream)
	p, err := New(id, append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModel(model),
		llmprovider.WithBaseURL(srv.URL), llmprovider.WithModelProbes(false), testCache(t)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return p, srv
}

// t92Drain is a Stream of p: the response EventDone carried, and the error
// that ended it.
func t92Drain(p llmprovider.Provider) (*llmprovider.Response, error) {
	var resp *llmprovider.Response
	for ev, err := range llmprovider.Stream(context.Background(), p, overStreamHi()) {
		if err != nil {
			return nil, err
		}
		if ev.Type == llmprovider.EventDone {
			resp = ev.Response
		}
	}
	return resp, nil
}

// TestTerminalFill_ProviderLogger_0092 (ST20): openai, grok and OpenCode's
// responses route, each given WithLogger, log the one line for M1's shape.
func TestTerminalFill_ProviderLogger_0092(t *testing.T) {
	for _, tc := range []struct {
		name  string
		id    llmprovider.ProviderID
		model string
		opts  []llmprovider.Option
	}{
		{"openai", llmprovider.ProviderOpenAI, "gpt-5", nil},
		{"grok", llmprovider.ProviderGrok, "grok-4.6", nil},
		{"opencode responses", llmprovider.ProviderOpencodeZen, "gpt-5.5", []llmprovider.Option{opencode.WithRoute(opencode.RouteResponses)}},
	} {
		log := newRecordedLog()
		p, _ := t92Provider(t, tc.id, tc.model, t92M1, append(tc.opts, llmprovider.WithLogger(slog.New(log)))...)
		resp, err := t92Drain(p)
		if err != nil || resp == nil || resp.OutputText() != "Hello" {
			t.Errorf("%s, M1: %+v (%v); want the answer", tc.name, resp, err)
		}
		log.mu.Lock()
		lines := slices.Clone(*log.lines)
		log.mu.Unlock()
		found := 0
		for _, line := range lines {
			if strings.Contains(line, "items taken from the terminal response") && strings.Contains(line, "message=1") {
				found++
			}
		}
		if found != 1 {
			t.Errorf("%s, M1: log lines %q; want one line of the take", tc.name, lines)
		}
	}
}

// TestTerminalFill_Unfinished_0092 (ST11): on a completed stream, a call or
// a message announced and delivered neither by .done nor by the terminal
// fails the reply as a broken stream, ErrProviderUnavailable after the
// reply, naming item_not_finished and the item's id; a reasoning item so
// announced is a success, as before. A Generate read over the stream of a
// reply whose only item is the lost call is sent twice under WithRetry; M3
// under WithStreamRetry, after its text, is sent once.
func TestTerminalFill_Unfinished_0092(t *testing.T) {
	added := func(index int, item string) string {
		return t92Event("response.output_item.added", fmt.Sprintf(`,"output_index":%d,"item":%s`, index, item))
	}
	m3 := t92Stream(t92Hello + added(1, t92CallAdded))
	broken := func(name string, err error, id, kind string) {
		t.Helper()
		// IsAfterReply pins ReadEvents' mark, which every error a decoder
		// returns carries (0053-MADR D3); it cannot tell whether unfinished
		// marks its error too (0092-MADR, amendment of 2026-11-25, sixth).
		if !errors.Is(err, llmprovider.ErrProviderUnavailable) || errors.Is(err, llmprovider.ErrIncomplete) ||
			!transport.IsAfterReply(err) || err == nil || !strings.Contains(err.Error(), "item_not_finished") ||
			!strings.Contains(err.Error(), id) || !strings.Contains(err.Error(), kind) {
			t.Errorf("%s: %v; want ErrProviderUnavailable after the reply, naming item_not_finished, %s and %s", name, err, kind, id)
		}
	}

	p, _ := t92Provider(t, llmprovider.ProviderOpenAI, "gpt-5", m3)
	resp, err := t92Drain(p)
	broken("M3, a call announced, never delivered", err, "fc_1", "function_call")
	if resp != nil {
		t.Errorf("M3: a response %+v; want none", resp)
	}

	noID := `{"type":"function_call","status":"in_progress","call_id":"call_9","name":"lookup","arguments":""}`
	p, _ = t92Provider(t, llmprovider.ProviderOpenAI, "gpt-5", t92Stream(t92Hello+added(1, noID)))
	_, err = t92Drain(p)
	broken("a call announced with no id, never delivered", err, "call_9", "function_call")

	msg2 := `{"id":"msg_2","type":"message","role":"assistant","status":"in_progress","content":[]}`
	p, _ = t92Provider(t, llmprovider.ProviderOpenAI, "gpt-5", t92Stream(t92Hello+added(1, msg2)))
	_, err = t92Drain(p)
	broken("a message announced, never delivered", err, "msg_2", "message")

	rs := `{"id":"rs_1","type":"reasoning","summary":[]}`
	p, _ = t92Provider(t, llmprovider.ProviderOpenAI, "gpt-5", t92Stream(t92Hello+added(1, rs)))
	if resp, err := t92Drain(p); err != nil || resp == nil || resp.OutputText() != "Hello" {
		t.Errorf("a reasoning item announced, never delivered: %+v (%v); want the answer, as before", resp, err)
	}

	policy := llmprovider.RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}
	p, srv := t92Provider(t, llmprovider.ProviderOpenAI, "gpt-5", t92Stream(added(0, t92CallAdded)))
	_, err = llmprovider.WithRetry(p, policy).Generate(context.Background(), overStreamHi())
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) || srv.count() != 2 {
		t.Errorf("Generate over the stream, the only item lost, WithRetry: %v after %d request(s); want ErrProviderUnavailable after 2",
			err, srv.count())
	}

	p, srv = t92Provider(t, llmprovider.ProviderOpenAI, "gpt-5", m3)
	_, err = t92Drain(llmprovider.WithStreamRetry(p, policy, nil))
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) || srv.count() != 1 {
		t.Errorf("M3, WithStreamRetry: %v after %d request(s); want ErrProviderUnavailable after 1", err, srv.count())
	}
}
