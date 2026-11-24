package wiretest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

// A plain Generate whose output limit the SDK chose or left out asks for the
// service's event stream (0088-MADR ST88), so a test server that answers a
// plain JSON reply must answer that reply's stream form when a request asks
// for one (ST88e). StreamReply writes a whole reply as the events of its
// wire's stream that the SDK's stream readers assemble into the same
// response: every block, item, step or part whole, at its start, then the
// stream's end, with the reply's id, model, stop reason and usage.

// The stream events' keys, which every wire's events repeat.
const (
	keyType      = "type"
	keyIndex     = "index"
	keyResponse  = "response"
	keyEventType = "event_type"
)

// WantsStream reports whether r, whose body is body, asks for an event
// stream: a JSON body with "stream": true, or a generateContent stream path.
func WantsStream(r *http.Request, body []byte) bool {
	if strings.HasSuffix(r.URL.Path, ":streamGenerateContent") {
		return true
	}
	var v struct {
		Stream bool `json:"stream"`
	}
	return json.Unmarshal(body, &v) == nil && v.Stream
}

// StreamReply is plain, a whole JSON reply to a request at path, as its
// wire's event stream, and true; or false when path names no wire it knows
// or plain is not a JSON object, such as a reply that is already a stream or
// is garbled on purpose.
func StreamReply(path string, plain string) (string, bool) {
	var reply map[string]any
	dec := json.NewDecoder(strings.NewReader(plain))
	dec.UseNumber()
	if dec.Decode(&reply) != nil || dec.More() {
		return "", false
	}
	var s sse
	switch {
	case strings.HasSuffix(path, "/messages"):
		s.messages(reply)
	case strings.HasSuffix(path, "/chat/completions"):
		s.chat(reply)
	case strings.HasSuffix(path, "/responses"):
		s.responses(reply)
	case strings.HasSuffix(path, "/interactions"):
		s.interactions(reply)
	case strings.HasSuffix(path, ":streamGenerateContent"):
		s.google(reply)
	default:
		return "", false
	}
	if s.err != nil {
		return "", false
	}
	return s.buf.String(), true
}

// StreamBody is plain, a fake transport's 200 JSON reply to r, as its stream
// form when r asks for a stream (StreamReply), and otherwise plain. r's body
// is read and put back.
func StreamBody(r *http.Request, plain string) string {
	if r.Body == nil {
		return plain
	}
	body, err := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil || !WantsStream(r, body) {
		return plain
	}
	if streamed, ok := StreamReply(r.URL.Path, plain); ok {
		return streamed
	}
	return plain
}

// StreamReplies wraps h, a test server's handler, so that a request asking
// for a stream gets the stream form of h's 200 JSON reply (StreamReply). A
// reply whose first byte is not a JSON object's, such as an event stream h
// writes and flushes itself, or a reply with another status, passes through
// as h writes it, flushes and hijacks included; so does any request that
// asks for no stream.
func StreamReplies(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		if !WantsStream(r, body) {
			h.ServeHTTP(w, r)
			return
		}
		s := &sniffer{w: w, status: http.StatusOK}
		h.ServeHTTP(s, r)
		s.finish(r.URL.Path)
	})
}

// sniffer is a ResponseWriter that holds a 200 reply whose first byte opens
// a JSON object, to write its stream form at the end, and passes any other
// reply through as it is written.
type sniffer struct {
	w       http.ResponseWriter
	status  int
	decided bool
	held    bool
	buf     bytes.Buffer
}

func (s *sniffer) Header() http.Header { return s.w.Header() }

func (s *sniffer) WriteHeader(status int) {
	if !s.decided {
		s.status = status
	}
}

// decide holds the reply when it is a 200 whose first byte opens a JSON
// object, and otherwise writes the status and passes the reply through.
func (s *sniffer) decide(first []byte) {
	if s.decided {
		return
	}
	s.decided = true
	trimmed := bytes.TrimLeft(first, " \t\r\n")
	s.held = s.status == http.StatusOK && len(trimmed) > 0 && trimmed[0] == '{'
	if !s.held {
		s.w.WriteHeader(s.status)
	}
}

func (s *sniffer) Write(p []byte) (int, error) {
	s.decide(p)
	if s.held {
		return s.buf.Write(p)
	}
	return s.w.Write(p)
}

// Flush flushes a reply passed through; a held one is written at the end.
func (s *sniffer) Flush() {
	s.decide(nil)
	if f, ok := s.w.(http.Flusher); !s.held && ok {
		f.Flush()
	}
}

// Hijack hands over the connection, after writing anything held as it was.
func (s *sniffer) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if s.held {
		s.held = false
		s.w.WriteHeader(s.status)
		if _, err := s.w.Write(s.buf.Bytes()); err != nil {
			return nil, nil, fmt.Errorf("wiretest: write the held reply: %w", err)
		}
	}
	s.decided = true
	hj, ok := s.w.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("wiretest: the response writer cannot be hijacked")
	}
	return hj.Hijack()
}

// finish writes a held reply's stream form, or the reply as it came when it
// has none, or the status of a reply that wrote nothing. A write that fails
// aborts the handler (http.ErrAbortHandler), as for a client gone.
func (s *sniffer) finish(path string) {
	if !s.decided {
		s.w.WriteHeader(s.status)
		return
	}
	if !s.held {
		return
	}
	out := s.buf.String()
	if streamed, ok := StreamReply(path, out); ok {
		out = streamed
		s.w.Header().Del("Content-Length")
		s.w.Header().Set("Content-Type", "text/event-stream")
	}
	s.w.WriteHeader(s.status)
	if _, err := io.WriteString(s.w, out); err != nil {
		panic(http.ErrAbortHandler)
	}
}

// StreamRepliesFunc is StreamReplies for a handler function.
func StreamRepliesFunc(f func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return StreamReplies(http.HandlerFunc(f)).ServeHTTP
}

// sse is a stream being written.
type sse struct {
	buf bytes.Buffer
	err error
}

// data writes one event: its event line when name is set, then v as data.
func (s *sse) data(name string, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		s.err = fmt.Errorf("wiretest: stream event: %w", err)
		return
	}
	if name != "" {
		s.buf.WriteString("event: " + name + "\n")
	}
	s.buf.WriteString("data: ")
	s.buf.Write(b)
	s.buf.WriteString("\n\n")
}

// asObj is v as an object, or nil.
func asObj(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

// obj is m's field k as an object, or nil.
func obj(m map[string]any, k string) map[string]any { return asObj(m[k]) }

// list is m's field k as a list, or nil.
func list(m map[string]any, k string) []any {
	if v, ok := m[k].([]any); ok {
		return v
	}
	return nil
}

// pick is the fields of m named in keys that m has.
func pick(m map[string]any, keys ...string) map[string]any {
	out := map[string]any{}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			out[k] = v
		}
	}
	return out
}

// messages writes a Messages reply as its stream: message_start with its
// id, model and usage, each content block whole at its start, then the stop
// reason and message_stop. An error reply is the stream's error event.
func (s *sse) messages(m map[string]any) {
	if m[keyType] == "error" {
		s.data("error", m)
		return
	}
	s.data("message_start", map[string]any{keyType: "message_start", "message": pick(m, "id", "model", "usage")})
	for i, b := range list(m, "content") {
		s.data("content_block_start", map[string]any{keyType: "content_block_start", keyIndex: i, "content_block": b})
		s.data("content_block_stop", map[string]any{keyType: "content_block_stop", keyIndex: i})
	}
	delta := map[string]any{}
	if r, ok := m["stop_reason"].(string); ok {
		delta["stop_reason"] = r
	}
	s.data("message_delta", map[string]any{keyType: "message_delta", "delta": delta})
	s.data("message_stop", map[string]any{keyType: "message_stop"})
}

// chat writes a Chat Completions reply as its stream: one chunk carrying the
// id, model, usage and the first choice's message as its delta, its calls
// indexed and their arguments as text, then [DONE]. An error at the top
// level, or in the choice, goes in the chunk as it came.
func (s *sse) chat(m map[string]any) {
	chunk := pick(m, "id", "model", "usage", "error")
	if choices := list(m, "choices"); len(choices) > 0 {
		choice := asObj(choices[0])
		out := pick(choice, "finish_reason", "error")
		out[keyIndex] = 0
		msg := obj(choice, "message")
		delta := pick(msg, "role", "refusal", "reasoning_content", "reasoning")
		delta["content"] = chatText(msg["content"])
		details := list(msg, "reasoning_details")
		for i, d := range details {
			if entry, ok := d.(map[string]any); ok && len(details) > 1 {
				if _, has := entry[keyIndex]; !has {
					entry[keyIndex] = i
				}
			}
		}
		if len(details) > 0 {
			delta["reasoning_details"] = details
		}
		var calls []any
		for i, c := range list(msg, "tool_calls") {
			call := asObj(c)
			fn := obj(call, "function")
			args, ok := fn["arguments"].(string)
			if !ok && fn["arguments"] != nil {
				b, err := json.Marshal(fn["arguments"])
				if err != nil {
					s.err = fmt.Errorf("wiretest: a call's arguments: %w", err)
					return
				}
				args = string(b)
			}
			calls = append(calls, map[string]any{keyIndex: i, "id": call["id"], keyType: "function",
				"function": map[string]any{"name": fn["name"], "arguments": args}})
		}
		if len(calls) > 0 {
			delta["tool_calls"] = calls
		}
		out["delta"] = delta
		chunk["choices"] = []any{out}
	}
	s.data("", chunk)
	s.buf.WriteString("data: [DONE]\n\n")
}

// chatText is a message's content as one text: a string, or an array whose
// text parts are joined.
func chatText(v any) string {
	switch c := v.(type) {
	case string:
		return c
	case []any:
		var sb strings.Builder
		for _, p := range c {
			if part, ok := p.(map[string]any); ok {
				if t, ok := part["text"].(string); ok {
					sb.WriteString(t)
				}
			}
		}
		return sb.String()
	}
	return ""
}

// responses writes a Responses reply as its stream: response.created with
// its id and model, each output item done, then the end its status names:
// failed, incomplete, or completed with its usage.
func (s *sse) responses(m map[string]any) {
	s.data("response.created", map[string]any{keyType: "response.created", keyResponse: pick(m, "id", "model")})
	for _, item := range list(m, "output") {
		s.data("response.output_item.done", map[string]any{keyType: "response.output_item.done", "item": item})
	}
	switch m["status"] {
	case "failed":
		s.data("response.failed", map[string]any{keyType: "response.failed", keyResponse: pick(m, "id", "error")})
	case "incomplete":
		s.data("response.incomplete", map[string]any{keyType: "response.incomplete",
			keyResponse: pick(m, "id", "incomplete_details")})
	default:
		s.data("response.completed", map[string]any{keyType: "response.completed",
			keyResponse: pick(m, "id", "model", "usage")})
	}
}

// google writes a generateContent reply as its stream: the reply as one
// chunk. A stream ends at the chunk with a finishReason, which a whole reply
// may leave out; the chunk then carries STOP, as a service's last chunk does.
func (s *sse) google(m map[string]any) {
	if candidates := list(m, "candidates"); len(candidates) > 0 {
		if c, ok := candidates[0].(map[string]any); ok && c["finishReason"] == nil {
			c["finishReason"] = "STOP"
		}
	}
	s.data("", m)
}

// interactions writes an Interaction as its stream: interaction.created, each
// step whole at its start, then interaction.completed with its status and
// usage. An error reply is the stream's error event.
func (s *sse) interactions(m map[string]any) {
	if _, ok := m["error"]; ok {
		s.data("error", m)
		return
	}
	s.data("interaction.created", map[string]any{keyEventType: "interaction.created", "interaction": pick(m, "id", "model")})
	for i, step := range list(m, "steps") {
		s.data("step.start", map[string]any{keyEventType: "step.start", keyIndex: i, "step": step})
		s.data("step.stop", map[string]any{keyEventType: "step.stop", keyIndex: i})
	}
	s.data("interaction.completed", map[string]any{keyEventType: "interaction.completed",
		"interaction": pick(m, "id", "model", "status", "usage")})
	s.buf.WriteString("event: done\ndata: [DONE]\n\n")
}
