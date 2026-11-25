package providers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// 0088-PLAN Phase 1b (OQ11 "Inside 0088, this tag"; 0088-MADR ST75 to ST81,
// ST93 to ST95): a reply the output limit cut is reported one way on every
// wire, keeping what is usable. Each route is built with a caller's
// WithMaxTokens(8192), so its Generate is a plain request read by the wire's
// decoder and its Stream the wire's stream reader. Every reply is this
// file's own; none is a recorded one.

// cutCall is a call of a cut reply: the arguments the service sent, and
// whether it is the call the limit cut, which each wire writes in its own
// cut form.
type cutCall struct {
	id, name, args string
	cut            bool
}

// cutReply is a reply the output limit cut: its reasoning, text and calls.
type cutReply struct {
	reasoning, text string
	calls           []cutCall
}

// cutRoute is a wire, the route that speaks it, and its cut forms.
type cutRoute struct {
	name  string
	id    llmprovider.ProviderID
	model string
	opts  []llmprovider.Option
	// reason is the Reason of an ErrEmptyAnswer for a cut reply with nothing
	// usable: the service's finish reason (ST79, ST93).
	reason string
	// plainCuts and streamCuts report whether the wire's plain reply, and
	// its stream, can carry a cut call (ST78): generateContent's parts and an
	// Interaction's steps arrive whole.
	plainCuts, streamCuts bool
	plain, stream         func(cutReply) string
}

// cutRoutes are the five wires: Chat Completions (Kilo), Messages (Claude),
// generateContent (OpenCode's google route), Interactions (Gemini) and
// Responses (OpenAI with a key).
func cutRoutes() []cutRoute {
	return []cutRoute{
		{name: "chat completions", id: llmprovider.ProviderKilo, model: "openai/gpt-6-luna", reason: "length",
			plainCuts: true, streamCuts: true, plain: chatCutPlain, stream: chatCutStream},
		{name: "messages", id: llmprovider.ProviderClaude, model: "claude-sonnet-5", reason: "length",
			plainCuts: true, streamCuts: true, plain: messagesCutPlain, stream: messagesCutStream},
		{name: "generatecontent", id: llmprovider.ProviderOpencodeZen, model: "gemini-3.7-flash", reason: "length",
			opts: []llmprovider.Option{opencode.WithRoute(opencode.RouteGoogle)}, plain: gcCutPlain, stream: gcCutStream},
		{name: "interactions", id: llmprovider.ProviderGemini, model: "gemini-3.7-flash", reason: "incomplete",
			streamCuts: true, plain: interactionCutPlain, stream: interactionCutStream},
		{name: "responses", id: llmprovider.ProviderOpenAI, model: "gpt-5", reason: "max_output_tokens",
			plainCuts: true, streamCuts: true, plain: responsesCutPlain, stream: responsesCutStream},
	}
}

// js is v as JSON.
func js(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// rawArgs is a complete call's arguments as a JSON value, "{}" for none. A
// cut call's are not JSON: they are quoted, for a form that cannot carry a
// cut call and is not served with one.
func rawArgs(args string) json.RawMessage {
	switch {
	case args == "":
		return json.RawMessage("{}")
	case !json.Valid([]byte(args)):
		return json.RawMessage(js(args))
	}
	return json.RawMessage(args)
}

func chatCutPlain(r cutReply) string {
	msg := map[string]any{"role": "assistant", "content": nil}
	if r.text != "" {
		msg["content"] = r.text
	}
	if r.reasoning != "" {
		msg["reasoning"] = r.reasoning
	}
	var calls []any
	for _, c := range r.calls {
		calls = append(calls, map[string]any{"id": c.id, "type": "function",
			"function": map[string]any{"name": c.name, "arguments": c.args}})
	}
	if calls != nil {
		msg["tool_calls"] = calls
	}
	return js(map[string]any{"id": "c", "model": "m", "usage": map[string]any{"prompt_tokens": 3, "completion_tokens": 2},
		"choices": []any{map[string]any{"index": 0, "finish_reason": "length", "message": msg}}})
}

func chatCutStream(r cutReply) string {
	var b strings.Builder
	chunk := func(delta map[string]any, finish any) {
		b.WriteString("data: " + js(map[string]any{"id": "c", "model": "m",
			"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": finish}}}) + "\n\n")
	}
	if r.reasoning != "" {
		chunk(map[string]any{"reasoning": r.reasoning}, nil)
	}
	if r.text != "" {
		chunk(map[string]any{"content": r.text}, nil)
	}
	for i, c := range r.calls {
		chunk(map[string]any{"tool_calls": []any{map[string]any{"index": i, "id": c.id, "type": "function",
			"function": map[string]any{"name": c.name, "arguments": c.args}}}}, nil)
	}
	chunk(map[string]any{}, "length")
	b.WriteString("data: " + js(map[string]any{"id": "c", "model": "m", "choices": []any{},
		"usage": map[string]any{"prompt_tokens": 3, "completion_tokens": 2}}) + "\n\ndata: [DONE]\n\n")
	return b.String()
}

// messagesCutPlain writes a cut call with no input: a plain reply carries
// whole JSON, so its cut call is one cut before its first argument (ST94).
func messagesCutPlain(r cutReply) string {
	var content []any
	if r.reasoning != "" {
		content = append(content, map[string]any{"type": "thinking", "thinking": r.reasoning, "signature": "sig"})
	}
	if r.text != "" {
		content = append(content, map[string]any{"type": "text", "text": r.text})
	}
	for _, c := range r.calls {
		block := map[string]any{"type": "tool_use", "id": c.id, "name": c.name}
		if !c.cut {
			block["input"] = rawArgs(c.args)
		}
		content = append(content, block)
	}
	return js(map[string]any{"id": "msg_cut", "type": "message", "role": "assistant", "model": "claude-sonnet-5",
		"stop_reason": "max_tokens", "content": content, "usage": map[string]any{"input_tokens": 3, "output_tokens": 2}})
}

func messagesCutStream(r cutReply) string {
	var b strings.Builder
	ev := func(v map[string]any) { b.WriteString("event: " + v["type"].(string) + "\ndata: " + js(v) + "\n\n") }
	ev(map[string]any{"type": "message_start", "message": map[string]any{"model": "claude-sonnet-5",
		"usage": map[string]any{"input_tokens": 3}}})
	i := 0
	block := func(start, delta map[string]any) {
		ev(map[string]any{"type": "content_block_start", "index": i, "content_block": start})
		if delta != nil {
			ev(map[string]any{"type": "content_block_delta", "index": i, "delta": delta})
		}
		ev(map[string]any{"type": "content_block_stop", "index": i})
		i++
	}
	if r.reasoning != "" {
		block(map[string]any{"type": "thinking", "thinking": ""}, map[string]any{"type": "thinking_delta", "thinking": r.reasoning})
	}
	if r.text != "" {
		block(map[string]any{"type": "text", "text": ""}, map[string]any{"type": "text_delta", "text": r.text})
	}
	for _, c := range r.calls {
		var delta map[string]any
		if c.args != "" {
			delta = map[string]any{"type": "input_json_delta", "partial_json": c.args}
		}
		block(map[string]any{"type": "tool_use", "id": c.id, "name": c.name, "input": map[string]any{}}, delta)
	}
	ev(map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": "max_tokens"},
		"usage": map[string]any{"output_tokens": 2}})
	ev(map[string]any{"type": "message_stop"})
	return b.String()
}

// gcParts is a generateContent reply's parts; its calls are whole.
func gcParts(r cutReply) []any {
	var parts []any
	if r.reasoning != "" {
		parts = append(parts, map[string]any{"text": r.reasoning, "thought": true})
	}
	if r.text != "" {
		parts = append(parts, map[string]any{"text": r.text})
	}
	for _, c := range r.calls {
		parts = append(parts, map[string]any{"functionCall": map[string]any{"id": c.id, "name": c.name, "args": rawArgs(c.args)}})
	}
	return parts
}

func gcCutPlain(r cutReply) string {
	return js(map[string]any{"modelVersion": "gemini-3.7-flash",
		"usageMetadata": map[string]any{"promptTokenCount": 3, "candidatesTokenCount": 2},
		"candidates":    []any{map[string]any{"finishReason": "MAX_TOKENS", "content": map[string]any{"role": "model", "parts": gcParts(r)}}}})
}

func gcCutStream(r cutReply) string {
	var b strings.Builder
	for _, part := range gcParts(r) {
		b.WriteString("data: " + js(map[string]any{"candidates": []any{map[string]any{
			"content": map[string]any{"role": "model", "parts": []any{part}}}}}) + "\n\n")
	}
	b.WriteString("data: " + js(map[string]any{"modelVersion": "gemini-3.7-flash",
		"usageMetadata": map[string]any{"promptTokenCount": 3, "candidatesTokenCount": 2},
		"candidates":    []any{map[string]any{"finishReason": "MAX_TOKENS", "content": map[string]any{"role": "model", "parts": []any{}}}}}) + "\n\n")
	return b.String()
}

var interactionUsageJSON = map[string]any{"total_input_tokens": 3, "total_output_tokens": 2}

// interactionCutPlain writes an Interaction with status incomplete: its
// steps are whole.
func interactionCutPlain(r cutReply) string {
	var steps []any
	if r.reasoning != "" {
		steps = append(steps, map[string]any{"type": "thought", "signature": "sig",
			"summary": []any{map[string]any{"type": "text", "text": r.reasoning}}})
	}
	if r.text != "" {
		steps = append(steps, map[string]any{"type": "model_output", "content": []any{map[string]any{"type": "text", "text": r.text}}})
	}
	for _, c := range r.calls {
		steps = append(steps, map[string]any{"type": "function_call", "id": c.id, "name": c.name, "arguments": rawArgs(c.args)})
	}
	return js(map[string]any{"id": "int_1", "model": "gemini-3.7-flash", "status": "incomplete", "steps": steps,
		"usage": interactionUsageJSON})
}

// interactionCutStream sends a call's arguments as one text fragment, so a
// cut call's are not whole.
func interactionCutStream(r cutReply) string {
	var b strings.Builder
	ev := func(kind string, fields map[string]any) {
		fields["event_type"] = kind
		b.WriteString("event: " + kind + "\ndata: " + js(fields) + "\n\n")
	}
	ev("interaction.created", map[string]any{"interaction": map[string]any{"id": "int_1", "model": "gemini-3.7-flash"}})
	i := 0
	if r.reasoning != "" {
		ev("step.start", map[string]any{"index": i, "step": map[string]any{"type": "thought",
			"summary": []any{map[string]any{"type": "text", "text": r.reasoning}}}})
		ev("step.delta", map[string]any{"index": i, "delta": map[string]any{"type": "thought_signature", "signature": "sig"}})
		i++
	}
	if r.text != "" {
		ev("step.start", map[string]any{"index": i, "step": map[string]any{"type": "model_output"}})
		ev("step.delta", map[string]any{"index": i, "delta": map[string]any{"type": "text", "text": r.text}})
		i++
	}
	for _, c := range r.calls {
		ev("step.start", map[string]any{"index": i, "step": map[string]any{"type": "function_call", "id": c.id, "name": c.name}})
		if c.args != "" {
			ev("step.delta", map[string]any{"index": i, "delta": map[string]any{"type": "arguments_delta", "arguments": c.args}})
		}
		i++
	}
	ev("interaction.completed", map[string]any{"interaction": map[string]any{"id": "int_1", "model": "gemini-3.7-flash",
		"status": "incomplete", "usage": interactionUsageJSON}})
	return b.String()
}

// responsesItems is a cut Responses reply's items; a cut call's status is
// incomplete.
func responsesItems(r cutReply) []map[string]any {
	var items []map[string]any
	if r.reasoning != "" {
		items = append(items, map[string]any{"type": "reasoning", "summary": []any{map[string]any{"type": "summary_text", "text": r.reasoning}}})
	}
	if r.text != "" {
		items = append(items, map[string]any{"type": "message", "role": "assistant", "status": "incomplete",
			"content": []any{map[string]any{"type": "output_text", "text": r.text}}})
	}
	for _, c := range r.calls {
		status := "completed"
		if c.cut {
			status = "incomplete"
		}
		items = append(items, map[string]any{"type": "function_call", "status": status, "call_id": c.id, "name": c.name,
			"arguments": c.args})
	}
	return items
}

var responsesUsageJSON = map[string]any{"input_tokens": 3, "output_tokens": 2}

func responsesCutPlain(r cutReply) string {
	return js(map[string]any{"id": "resp_1", "model": "gpt-5", "status": "incomplete",
		"incomplete_details": map[string]any{"reason": "max_output_tokens"}, "output": responsesItems(r), "usage": responsesUsageJSON})
}

// responsesCutStream sends output_item.done for each item but the cut call,
// which is only added and given an arguments delta, then response.incomplete.
func responsesCutStream(r cutReply) string {
	var b strings.Builder
	ev := func(v map[string]any) { b.WriteString("event: " + v["type"].(string) + "\ndata: " + js(v) + "\n\n") }
	ev(map[string]any{"type": "response.created", "response": map[string]any{"id": "resp_1", "model": "gpt-5"}})
	for _, item := range responsesItems(r) {
		switch {
		case item["status"] == "incomplete" && item["type"] == "function_call":
			ev(map[string]any{"type": "response.output_item.added", "item": item})
			ev(map[string]any{"type": "response.function_call_arguments.delta", "delta": item["arguments"]})
			continue
		case item["type"] == "message":
			ev(map[string]any{"type": "response.output_text.delta", "delta": r.text})
		}
		ev(map[string]any{"type": "response.output_item.done", "item": item})
	}
	ev(map[string]any{"type": "response.incomplete", "response": map[string]any{"id": "resp_1", "model": "gpt-5",
		"status": "incomplete", "incomplete_details": map[string]any{"reason": "max_output_tokens"}, "usage": responsesUsageJSON}})
	return b.String()
}

// cutServer answers every POST with plain or stream, as the request asks,
// and counts the POSTs; every other request is a 404.
type cutServer struct {
	*httptest.Server
	mu    sync.Mutex
	posts int
}

func newCutServer(t *testing.T, plain, stream string) *cutServer {
	t.Helper()
	s := &cutServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		s.mu.Lock()
		s.posts++
		s.mu.Unlock()
		if body["stream"] == true || strings.Contains(r.URL.Path, "streamGenerateContent") {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, stream)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, plain)
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *cutServer) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.posts
}

// cutOutcome is what one call of a route returned: the response and error,
// and for a stream the calls it yielded as items and whether it ended with
// EventDone.
type cutOutcome struct {
	resp   *llmprovider.Response
	err    error
	items  []string // the yielded calls' names
	done   bool
	posts  int
	stream bool
}

// cutRun sends one request on route, by Generate (a plain request) or by
// Stream, to a server answering with reply in the wire's form.
func cutRun(t *testing.T, route cutRoute, stream bool, reply cutReply) cutOutcome {
	t.Helper()
	srv := newCutServer(t, route.plain(reply), route.stream(reply))
	p, err := New(route.id, append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModel(route.model),
		llmprovider.WithBaseURL(srv.URL), llmprovider.WithModelProbes(false),
		llmprovider.WithMaxTokens(8192), testCache(t)}, route.opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	out := cutOutcome{stream: stream}
	if !stream {
		out.resp, out.err = p.Generate(context.Background(), overStreamHi())
		out.posts = srv.count()
		return out
	}
	for ev, e := range llmprovider.Stream(context.Background(), p, overStreamHi()) {
		if e != nil {
			out.err = e
			break
		}
		switch ev.Type {
		case llmprovider.EventItem:
			if call, ok := ev.Item.(llmprovider.FunctionCallItem); ok {
				out.items = append(out.items, call.Name)
			}
		case llmprovider.EventDone:
			out.done, out.resp = true, ev.Response
		default:
		}
	}
	out.posts = srv.count()
	return out
}

// mode names a call's mode for a failure message.
func (o cutOutcome) mode() string {
	if o.stream {
		return "Stream"
	}
	return "Generate"
}

// calls is resp's calls, as name and arguments.
func cutCalls(resp *llmprovider.Response) []string {
	if resp == nil {
		return nil
	}
	var out []string
	for _, item := range resp.Output {
		if call, ok := item.(llmprovider.FunctionCallItem); ok {
			out = append(out, call.Name+" "+call.Arguments)
		}
	}
	return out
}

// TestCutAnswer_Text_0088 (ST75, ST93, ST95): on every wire, by Generate and
// Stream, a reply the limit cut with text is a response finishing length
// with the text, after one request: Gemini's Interaction incomplete is such a
// cut (ST93), and an answer stopped at a per-call cap is not continued
// (ST95).
func TestCutAnswer_Text_0088(t *testing.T) {
	for _, route := range cutRoutes() {
		for _, stream := range []bool{false, true} {
			o := cutRun(t, route, stream, cutReply{text: "hello"})
			if o.err != nil || o.resp == nil || o.resp.FinishReason != llmprovider.FinishLength ||
				o.resp.OutputText() != "hello" || o.posts != 1 {
				t.Errorf("%s, %s: %d request(s), resp %+v, err %v; want one request, a response finishing length with hello",
					route.name, o.mode(), o.posts, o.resp, o.err)
			}
		}
	}
}

// TestCutAnswer_CompleteCalls_0088 (ST76, ST77, ST78, ST93, ST94): a reply
// the limit cut with text, one complete call and one cut call is a response
// finishing length with the text and the complete call only, and one
// WarningToolCallCut naming the cut call by name and id; where the wire's
// form cannot carry a cut call (generateContent; an Interaction's plain
// steps, ST93), the complete call is returned with no warning. On Chat
// Completions, a call whose arguments are {"path":"a" is the cut one; an
// earlier call with empty arguments followed by another call is a complete
// call, {}; the last call with empty arguments is the cut one (ST94). On
// Messages, a tool_use that is the last block is the cut one, its input
// closed and parsed or not (ST103).
func TestCutAnswer_CompleteCalls_0088(t *testing.T) {
	complete := cutCall{id: "call_a", name: "lookup", args: `{"x":1}`}
	cut := cutCall{id: "call_b", name: "fetch", args: `{"path":"a"`, cut: true}
	for _, route := range cutRoutes() {
		for _, stream := range []bool{false, true} {
			canCut := route.plainCuts && !stream || route.streamCuts && stream
			reply := cutReply{text: "partial", calls: []cutCall{complete}}
			if canCut {
				reply.calls = append(reply.calls, cut)
			}
			o := cutRun(t, route, stream, reply)
			checkCutCalls(t, route.name+", "+o.mode(), o, []string{`lookup {"x":1}`}, canCut, cut, llmprovider.WarningToolCallCut)
		}
	}
	// ST103: on Messages, the last block, a tool_use whose input the
	// service closed and parsed, is the call the limit cut.
	messages := cutRoutes()[1]
	parsed := cutCall{id: "call_b", name: "commit", args: `{"subject":"fix: tru"}`}
	for _, stream := range []bool{false, true} {
		o := cutRun(t, messages, stream, cutReply{text: "partial", calls: []cutCall{complete, parsed}})
		checkCutCalls(t, "messages, the last block parsed (ST103), "+o.mode(), o, []string{`lookup {"x":1}`}, true,
			parsed, llmprovider.WarningToolCallCut)
	}
	chat := cutRoutes()[0]
	emptyLast := cutCall{id: "call_c", name: "list", args: "", cut: true}
	for _, stream := range []bool{false, true} {
		o := cutRun(t, chat, stream, cutReply{calls: []cutCall{complete, {id: "call_e", name: "now", args: ""}, emptyLast}})
		checkCutCalls(t, "chat completions, the empty last call, "+o.mode(), o, []string{`lookup {"x":1}`, "now {}"}, true,
			emptyLast, llmprovider.WarningToolCallCut)
	}
}

// checkCutCalls checks a cut answer returned with its text, if any, the
// calls want, and one warning of kind, WarningToolCallCut, naming cut when
// warned.
func checkCutCalls(t *testing.T, name string, o cutOutcome, want []string, warned bool, cut cutCall,
	kind llmprovider.WarningKind) {
	t.Helper()
	if o.err != nil || o.resp == nil || o.resp.FinishReason != llmprovider.FinishLength {
		t.Errorf("%s: resp %+v, err %v; want a response finishing length", name, o.resp, o.err)
		return
	}
	if got := cutCalls(o.resp); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("%s: calls %q; want %q, the cut call dropped", name, got, want)
	}
	var cuts []llmprovider.Warning
	for _, w := range o.resp.Warnings {
		if w.Kind == kind {
			cuts = append(cuts, w)
		}
	}
	switch {
	case !warned && len(cuts) != 0:
		t.Errorf("%s: warnings %+v; want no WarningToolCallCut: no call was cut", name, cuts)
	case warned && (len(cuts) != 1 || !strings.Contains(cuts[0].Detail, cut.name) || !strings.Contains(cuts[0].Detail, cut.id)):
		t.Errorf("%s: warnings %+v; want one WarningToolCallCut naming %s and %s", name, o.resp.Warnings, cut.name, cut.id)
	}
}

// TestCutAnswer_NothingUsable_0088 (ST79, ST93): a reply the limit cut with
// only reasoning, and one with only a cut call, is ErrEmptyAnswer with the
// service's finish reason (length; incomplete on an Interaction;
// max_output_tokens on Responses), and no response.
func TestCutAnswer_NothingUsable_0088(t *testing.T) {
	onlyCut := cutCall{id: "call_b", name: "fetch", args: `{"path":"a"`, cut: true}
	for _, route := range cutRoutes() {
		for _, stream := range []bool{false, true} {
			replies := map[string]cutReply{"only reasoning": {reasoning: "thinking"}}
			if route.plainCuts && !stream || route.streamCuts && stream {
				replies["only a cut call"] = cutReply{calls: []cutCall{onlyCut}}
			}
			for what, reply := range replies {
				o := cutRun(t, route, stream, reply)
				apiErr, ok := errors.AsType[*llmprovider.APIError](o.err)
				if !errors.Is(o.err, llmprovider.ErrEmptyAnswer) || !ok || apiErr.Reason != route.reason || o.resp != nil {
					t.Errorf("%s, %s, %s: resp %+v, err %v; want ErrEmptyAnswer with reason %q and no response",
						route.name, o.mode(), what, o.resp, o.err, route.reason)
				}
			}
		}
	}
}

// TestCutAnswer_Stream_0088 (ST80): each wire's Stream of a reply the limit
// cut yields the complete call as an item, never the cut one, and ends with
// EventDone carrying a response finishing length, not an error.
func TestCutAnswer_Stream_0088(t *testing.T) {
	complete := cutCall{id: "call_a", name: "lookup", args: `{"x":1}`}
	cut := cutCall{id: "call_b", name: "fetch", args: `{"path":"a"`, cut: true}
	for _, route := range cutRoutes() {
		reply := cutReply{text: "partial", calls: []cutCall{complete}}
		if route.streamCuts {
			reply.calls = append(reply.calls, cut)
		}
		o := cutRun(t, route, true, reply)
		if o.err != nil || !o.done || o.resp == nil || o.resp.FinishReason != llmprovider.FinishLength ||
			strings.Join(o.items, ",") != "lookup" {
			t.Errorf("%s: items %q, EventDone %t, resp %+v, err %v; want the item lookup alone, then EventDone finishing length",
				route.name, o.items, o.done, o.resp, o.err)
		}
	}
}

// TestCutAnswer_OtherIncomplete_0088 (ST81): a Responses answer incomplete
// for another reason than the output limit, a content filter, is as before:
// ErrIncomplete with that reason, no response, by Generate and Stream.
func TestCutAnswer_OtherIncomplete_0088(t *testing.T) {
	route := cutRoutes()[4]
	plain, stream := route.plain, route.stream
	route.plain = func(r cutReply) string {
		return strings.ReplaceAll(plain(r), `"max_output_tokens"`, `"content_filter"`)
	}
	route.stream = func(r cutReply) string {
		return strings.ReplaceAll(stream(r), `"max_output_tokens"`, `"content_filter"`)
	}
	for _, stream := range []bool{false, true} {
		o := cutRun(t, route, stream, cutReply{text: "hello"})
		apiErr, ok := errors.AsType[*llmprovider.APIError](o.err)
		if !errors.Is(o.err, llmprovider.ErrIncomplete) || errors.Is(o.err, llmprovider.ErrEmptyAnswer) || !ok ||
			apiErr.Reason != "content_filter" || o.resp != nil || o.done {
			t.Errorf("responses, %s, a content filter: resp %+v, err %v; want ErrIncomplete with reason content_filter, "+
				"no response", o.mode(), o.resp, o.err)
		}
	}
}

// TestCutAnswer_WireGeneral_0088 (ST107, ST110; R1 as the owner ruled it): a
// call of a reply the limit cut is the cut one when its arguments are not a
// JSON object, on every wire, and when it is the last item and they are {}
// only where its arguments arrive in pieces. A trailing call with no
// arguments is dropped and named on Chat Completions, Messages and
// Responses, by Generate and Stream; on generateContent, by Generate and
// Stream, and an Interaction's plain reply, whose calls arrive whole, it is a
// whole call that takes none, returned, yielded on a stream, with no warning
// (ST107). On an Interaction's stream what arrived decides (ST110): with no
// arguments_delta it is dropped and named; with one delta delivering {} it is
// kept and yielded. On Chat Completions, a call whose arguments are [1] is
// dropped and named, the whole call after it kept.
func TestCutAnswer_WireGeneral_0088(t *testing.T) {
	complete := cutCall{id: "call_a", name: "lookup", args: `{"x":1}`}
	trailing := cutCall{id: "call_z", name: "now"}
	for _, route := range cutRoutes() {
		for _, stream := range []bool{false, true} {
			// interactionCutStream sends no arguments_delta for a call with
			// no arguments (ST110 (a)).
			whole := route.name == "generatecontent" || route.name == "interactions" && !stream
			o := cutRun(t, route, stream, cutReply{text: "partial", calls: []cutCall{complete, trailing}})
			checkTrailing(t, route.name+", a trailing call with no arguments", o, whole, trailing)
		}
	}
	// ST110 (b): the trailing call's one arguments_delta delivers {}.
	delivered := cutCall{id: "call_z", name: "now", args: "{}"}
	o := cutRun(t, cutRoutes()[3], true, cutReply{text: "partial", calls: []cutCall{complete, delivered}})
	checkTrailing(t, "interactions, a trailing call whose delta is {}", o, true, delivered)

	// ST110 as OQ21 amends it: a delta whose piece is empty, null or white
	// space, or carries no arguments, has not arrived, so the trailing call
	// is the cut one.
	for what, field := range map[string]string{`""`: `,"arguments":""`, "null": `,"arguments":null`,
		"white space": `,"arguments":"  "`, "absent": "", `"null"`: `,"arguments":"null"`} {
		route := cutRoutes()[3]
		route.stream = func(r cutReply) string {
			delta := "event: step.delta\ndata: {\"event_type\":\"step.delta\",\"index\":2," +
				"\"delta\":{\"type\":\"arguments_delta\"" + field + "}}\n\n"
			s := interactionCutStream(r)
			at := strings.Index(s, "event: interaction.completed")
			return s[:at] + delta + s[at:]
		}
		o := cutRun(t, route, true, cutReply{text: "partial", calls: []cutCall{complete, trailing}})
		checkTrailing(t, "interactions, a trailing call whose delta is "+what, o, false, trailing)
	}

	notObject := cutCall{id: "call_n", name: "list", args: "[1]"}
	o = cutRun(t, cutRoutes()[0], false, cutReply{calls: []cutCall{notObject, complete}})
	checkCutCalls(t, "chat completions, arguments not an object, "+o.mode(), o, []string{`lookup {"x":1}`}, true,
		notObject, llmprovider.WarningToolCallCut)
}

// checkTrailing checks a cut reply of text, lookup and a trailing call that
// takes no arguments: kept, both calls returned and on a stream yielded, with
// no warning; or not, lookup alone returned and yielded, and the trailing
// call named in one WarningToolCallCut.
func checkTrailing(t *testing.T, name string, o cutOutcome, kept bool, trailing cutCall) {
	t.Helper()
	name += ", " + o.mode()
	want, items := []string{`lookup {"x":1}`}, "lookup"
	if kept {
		want, items = append(want, trailing.name+" {}"), "lookup,"+trailing.name
	}
	checkCutCalls(t, name, o, want, !kept, trailing, llmprovider.WarningToolCallCut)
	if o.stream && strings.Join(o.items, ",") != items {
		t.Errorf("%s: items %q; want %s", name, o.items, items)
	}
}

// TestCutAnswer_ResponsesCutItem_0088 (the MADR's tenth amendment, R2, open
// for the review): on Responses, a cut reply's call whose item is incomplete
// is the cut one, its arguments whole or not, by Generate; and a call a
// stream added and never finished is named, by Stream.
func TestCutAnswer_ResponsesCutItem_0088(t *testing.T) {
	route := cutRoutes()[4]
	complete := cutCall{id: "call_a", name: "lookup", args: `{"x":1}`}
	incomplete := cutCall{id: "call_b", name: "fetch", args: `{"y":2}`, cut: true}
	for _, stream := range []bool{false, true} {
		o := cutRun(t, route, stream, cutReply{text: "partial", calls: []cutCall{complete, incomplete}})
		checkCutCalls(t, "responses, an incomplete call item, "+o.mode(), o, []string{`lookup {"x":1}`}, true,
			incomplete, llmprovider.WarningToolCallCut)
	}
}

// TestCutAnswer_HeldCalls_0088 (the MADR's tenth amendment, R3, open for the
// review): on a Responses stream, a finished call with {} is held until a
// later item shows it was not the last, then yielded; the last one, {}, is
// never yielded, and the cut reply drops and names it.
func TestCutAnswer_HeldCalls_0088(t *testing.T) {
	first := cutCall{id: "call_f", name: "first", args: "{}"}
	complete := cutCall{id: "call_a", name: "lookup", args: `{"x":1}`}
	last := cutCall{id: "call_z", name: "now", args: "{}"}
	o := cutRun(t, cutRoutes()[4], true, cutReply{calls: []cutCall{first, complete, last}})
	if strings.Join(o.items, ",") != "first,lookup" {
		t.Errorf("responses, Stream: items %q; want first, then lookup, and never the last {} call", o.items)
	}
	checkCutCalls(t, "responses, held calls, Stream", o, []string{"first {}", `lookup {"x":1}`}, true, last,
		llmprovider.WarningToolCallCut)
}

// xAI's measured shapes (0088-MADR's eleventh amendment, a live probe of
// grok-4.20-0309-non-reasoning at max_output_tokens 16): a streamed cut ends
// with response.incomplete and no .done event, the text only in the
// terminal response's incomplete message item; a plain cut is incomplete at
// the top level with its message item completed. The text is the probe's
// shape, not its words.
const (
	xaiModel = "grok-4.20-0309-non-reasoning"
	xaiText  = "Counting: 1, 2, 3,"
	xaiUsage = `"usage":{"input_tokens":10,"output_tokens":16}`
)

func xaiEvent(kind, fields string) string {
	return "event: " + kind + "\ndata: {\"type\":\"" + kind + "\"" + fields + "}\n\n"
}

// xaiStream is the measured stream; terminal is the terminal response's
// output items beyond the message.
func xaiStream(added, terminal string) string {
	response := `"response":{"id":"resp_x","model":"` + xaiModel + `","status":"in_progress","output":[]}`
	return xaiEvent("response.created", ","+response) + xaiEvent("response.in_progress", ","+response) +
		xaiEvent("response.output_item.added", `,"output_index":0,"item":{"id":"msg_x","type":"message","role":"assistant","status":"in_progress","content":[]}`) +
		added +
		xaiEvent("response.content_part.added", `,"item_id":"msg_x","output_index":0,"content_index":0,"part":{"type":"output_text","text":""}`) +
		xaiEvent("response.output_text.delta", `,"item_id":"msg_x","output_index":0,"content_index":0,"delta":"`+xaiText+`"`) +
		xaiEvent("response.incomplete", `,"response":{"id":"resp_x","model":"`+xaiModel+`","status":"incomplete",`+
			`"incomplete_details":{"reason":"max_output_tokens"},"output":[{"id":"msg_x","type":"message","role":"assistant",`+
			`"status":"incomplete","content":[{"type":"output_text","text":"`+xaiText+`"}]}`+terminal+`],`+xaiUsage+`}`)
}

// TestCutAnswer_XAIShapes_0088 (ST104): on Grok's Responses wire, xAI's
// measured streamed cut, with no .done event, is a response finishing length
// with its text, ending with EventDone; its plain
// cut, the message item completed, is a cut too. A stream whose terminal
// response also carries calls no .done delivered (not measured: xAI did not
// cut a call) takes them under the cut rules: a whole call is returned and
// yielded, an incomplete one named once.
func TestCutAnswer_XAIShapes_0088(t *testing.T) {
	route := cutRoute{name: "grok", id: llmprovider.ProviderGrok, model: xaiModel}
	route.plain = func(cutReply) string {
		return `{"id":"resp_x","model":"` + xaiModel + `","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},` +
			`"output":[{"id":"msg_x","type":"message","role":"assistant","status":"completed",` +
			`"content":[{"type":"output_text","text":"` + xaiText + `"}]}],` + xaiUsage + `}`
	}
	route.stream = func(cutReply) string { return xaiStream("", "") }
	for _, stream := range []bool{false, true} {
		o := cutRun(t, route, stream, cutReply{})
		if o.err != nil || o.resp == nil || o.resp.FinishReason != llmprovider.FinishLength || o.resp.OutputText() != xaiText ||
			o.stream && !o.done {
			t.Errorf("grok, %s, xAI's cut: resp %+v, EventDone %t, err %v; want a response finishing length with its text",
				o.mode(), o.resp, o.done, o.err)
		}
	}

	added := xaiEvent("response.output_item.added", `,"output_index":1,"item":{"id":"fc_b","type":"function_call","status":"in_progress","call_id":"call_b","name":"fetch","arguments":""}`)
	terminal := `,{"id":"fc_a","type":"function_call","status":"completed","call_id":"call_a","name":"lookup","arguments":"{\"x\":1}"}` +
		`,{"id":"fc_b","type":"function_call","status":"incomplete","call_id":"call_b","name":"fetch","arguments":"{\"path\":\"a\""}`
	route.stream = func(cutReply) string { return xaiStream(added, terminal) }
	o := cutRun(t, route, true, cutReply{})
	if strings.Join(o.items, ",") != "lookup" {
		t.Errorf("grok, Stream, terminal calls: items %q; want lookup alone, yielded before EventDone", o.items)
	}
	checkCutCalls(t, "grok, Stream, terminal calls", o, []string{`lookup {"x":1}`}, true,
		cutCall{id: "call_b", name: "fetch"}, llmprovider.WarningToolCallCut)
}
