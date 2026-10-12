//go:build live_gateways

package llmprovider_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// 0037-PLAN Phase 2's measurement. TestLive_0037ReasoningCount runs only with
// LLMPROVIDER_LIVE_0037 set, and each target only with its key. It asks
// Claude to reason, once for a full response and once for a stream, and
// checks that the decoded ReasoningTokens is the reply's
// output_tokens_details.thinking_tokens. It logs only usage objects and the
// decoded Usage: no prompt, header or key.

// messagesUsage0037 is the part of a Messages usage object the check reads.
type messagesUsage0037 struct {
	Output  int `json:"output_tokens"`
	Details *struct {
		Thinking int `json:"thinking_tokens"`
	} `json:"output_tokens_details"`
}

// reasoning0037 is what one Messages reply says of its output and reasoning.
type reasoning0037 struct {
	raw              []string // each usage object, as sent
	output, thinking int
	stream           bool
	onStart          bool // a stream's message_start carried thinking_tokens
	notLast          bool // a message_delta before the last carried it
}

// parseReasoning0037 reads a Messages reply: a JSON body, or an event stream,
// whose counts are its last message_delta's, as Anthropic's guide says.
func parseReasoning0037(reply []byte) (reasoning0037, error) {
	var r reasoning0037
	trimmed := bytes.TrimSpace(reply)
	if bytes.HasPrefix(trimmed, []byte("{")) {
		var body struct {
			Usage json.RawMessage `json:"usage"`
		}
		if err := json.Unmarshal(trimmed, &body); err != nil {
			return r, err
		}
		var u messagesUsage0037
		if err := json.Unmarshal(body.Usage, &u); err != nil {
			return r, err
		}
		r.raw = append(r.raw, string(body.Usage))
		r.output = u.Output
		if u.Details != nil {
			r.thinking = u.Details.Thinking
		}
		return r, nil
	}
	r.stream = true
	var deltas []messagesUsage0037
	sc := bufio.NewScanner(bytes.NewReader(reply))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		data, ok := strings.CutPrefix(sc.Text(), "data: ")
		if !ok {
			continue
		}
		var ev struct {
			Type    string          `json:"type"`
			Usage   json.RawMessage `json:"usage"`
			Message struct {
				Usage json.RawMessage `json:"usage"`
			} `json:"message"`
		}
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			continue
		}
		var raw json.RawMessage
		switch ev.Type {
		case "message_start":
			raw = ev.Message.Usage
		case "message_delta":
			raw = ev.Usage
		default:
			continue
		}
		var u messagesUsage0037
		if err := json.Unmarshal(raw, &u); err != nil {
			return r, err
		}
		r.raw = append(r.raw, ev.Type+" "+string(raw))
		if ev.Type == "message_start" {
			r.onStart = u.Details != nil && u.Details.Thinking > 0
			continue
		}
		deltas = append(deltas, u)
	}
	if err := sc.Err(); err != nil {
		return r, err
	}
	for i, d := range deltas {
		if i < len(deltas)-1 && d.Details != nil && d.Details.Thinking > 0 {
			r.notLast = true
		}
	}
	if len(deltas) > 0 {
		last := deltas[len(deltas)-1]
		r.output = last.Output
		if last.Details != nil {
			r.thinking = last.Details.Thinking
		}
	}
	return r, nil
}

// problems lists what breaks 0037-MADR D1's facts: thinking_tokens is a part
// of output_tokens, and a stream sends it on its last message_delta only.
func (r reasoning0037) problems() []string {
	var out []string
	if r.thinking < 0 || r.thinking > r.output {
		out = append(out, fmt.Sprintf("thinking_tokens %d is above output_tokens %d, or negative", r.thinking, r.output))
	}
	if r.onStart {
		out = append(out, "thinking_tokens was sent on message_start")
	}
	if r.notLast {
		out = append(out, "thinking_tokens was sent on a message_delta that is not the last")
	}
	return out
}

// decodedProblems lists where got, the decoded Usage, differs from the reply.
func (r reasoning0037) decodedProblems(got llmprovider.Usage) []string {
	var out []string
	if got.ReasoningTokens != r.thinking {
		out = append(out, fmt.Sprintf("decoded ReasoningTokens %d, raw thinking_tokens %d", got.ReasoningTokens, r.thinking))
	}
	if got.OutputTokens != r.output {
		out = append(out, fmt.Sprintf("decoded OutputTokens %d, raw output_tokens %d", got.OutputTokens, r.output))
	}
	if got.ReasoningTokens > got.OutputTokens {
		out = append(out, fmt.Sprintf("decoded ReasoningTokens %d above OutputTokens %d", got.ReasoningTokens, got.OutputTokens))
	}
	return out
}

// TestReasoningChecks_0037 proves the measurement's checks on canned replies,
// with no key and no network (0037-PLAN Phase 2, step 1).
func TestReasoningChecks_0037(t *testing.T) {
	sse := func(events ...string) string {
		var b strings.Builder
		for _, e := range events {
			b.WriteString("event: x\ndata: " + e + "\n\n")
		}
		return b.String()
	}
	start := `{"type":"message_start","message":{"model":"m","usage":{"input_tokens":15,"output_tokens":1}}}`
	startThinking := `{"type":"message_start","message":{"model":"m","usage":{"input_tokens":15,"output_tokens":1,"output_tokens_details":{"thinking_tokens":30}}}}`
	delta := func(out, thinking string) string {
		details := ""
		if thinking != "" {
			details = `,"output_tokens_details":{"thinking_tokens":` + thinking + `}`
		}
		return `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":` + out + details + `}}`
	}
	stop := `{"type":"message_stop"}`
	good := llmprovider.Usage{InputTokens: 15, OutputTokens: 40, ReasoningTokens: 30}
	for _, c := range []struct {
		name, reply string
		got         llmprovider.Usage
		want        string
	}{
		{"a body, clean", `{"usage":{"input_tokens":15,"output_tokens":40,"output_tokens_details":{"thinking_tokens":30}}}`, good, ""},
		{"a stream, clean", sse(start, delta("40", "30"), stop), good, ""},
		{"a body, thinking above the output", `{"usage":{"input_tokens":15,"output_tokens":40,"output_tokens_details":{"thinking_tokens":50}}}`,
			llmprovider.Usage{InputTokens: 15, OutputTokens: 40, ReasoningTokens: 50}, "above output_tokens"},
		{"a stream, thinking on message_start", sse(startThinking, delta("40", "30"), stop), good, "message_start"},
		{"a stream, thinking on a message_delta that is not the last", sse(start, delta("20", "30"), delta("40", ""), stop),
			llmprovider.Usage{InputTokens: 15, OutputTokens: 40}, "not the last"},
		{"a body, decoded reasoning missing", `{"usage":{"input_tokens":15,"output_tokens":40,"output_tokens_details":{"thinking_tokens":30}}}`,
			llmprovider.Usage{InputTokens: 15, OutputTokens: 40}, "decoded ReasoningTokens 0, raw thinking_tokens 30"},
		{"a stream, decoded output wrong", sse(start, delta("40", "30"), stop),
			llmprovider.Usage{InputTokens: 15, OutputTokens: 1, ReasoningTokens: 30}, "decoded OutputTokens 1, raw output_tokens 40"},
	} {
		t.Run(c.name, func(t *testing.T) {
			r, err := parseReasoning0037([]byte(c.reply))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			problems := strings.Join(append(r.problems(), r.decodedProblems(c.got)...), "; ")
			switch {
			case c.want == "" && problems != "":
				t.Errorf("problems %q, want none", problems)
			case c.want != "" && !strings.Contains(problems, c.want):
				t.Errorf("problems %q, want one containing %q", problems, c.want)
			}
		})
	}
}

// lastPost0037 keeps the body of the last reply to a POST, read whole, and
// passes it on.
type lastPost0037 struct {
	mu   sync.Mutex
	body []byte
}

func (r *lastPost0037) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil || req.Method != http.MethodPost || resp.Body == nil {
		return resp, err
	}
	b, err := io.ReadAll(resp.Body)
	if cerr := resp.Body.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.body = b
	r.mu.Unlock()
	resp.Body = io.NopCloser(bytes.NewReader(b))
	return resp, nil
}

func (r *lastPost0037) last() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.body
}

// TestLive_0037ReasoningCount is 0037-PLAN Phase 2's measurement. Billed: two
// requests per target, two targets, a short prompt and at most 2,048 output
// tokens each.
func TestLive_0037ReasoningCount(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0037") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0037 to run 0037-PLAN Phase 2's measurement")
	}
	zen := llmprovider.ProviderOpencodeZen
	targets := []struct {
		name  string
		id    llmprovider.ProviderID
		key   func(t *testing.T) string
		model func(t *testing.T) string
	}{
		{"R1 claude", llmprovider.ProviderClaude,
			func(t *testing.T) string {
				return llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[llmprovider.ProviderClaude])
			},
			func(*testing.T) string { return "claude-haiku-5-5" }},
		{"R2 zen messages", zen, llmprovider.LiveOpencodeKey,
			func(t *testing.T) string {
				return llmprovider.LiveModel(t, zen, "claude-haiku-5-5", "claude-sonnet-5-5")
			}},
	}
	req := func() *llmprovider.Request {
		return &llmprovider.Request{Reasoning: &llmprovider.Reasoning{Effort: llmprovider.EffortHigh},
			Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser,
				Text: "A train leaves at 9:47 and the trip takes 2 hours 38 minutes, with a 19-minute stop added. " +
					"Work it out step by step, then give the arrival time only."}}}
	}
	for _, tg := range targets {
		t.Run(tg.name, func(t *testing.T) {
			key, model := tg.key(t), tg.model(t)
			rec := &lastPost0037{}
			p, err := providers.New(tg.id, llmprovider.WithAPIKey(key), llmprovider.WithModel(model),
				llmprovider.WithModelProbes(false), llmprovider.WithMaxTokens(2048),
				llmprovider.WithHTTPClient(&http.Client{Transport: rec}))
			if err != nil {
				t.Fatal(err)
			}
			reasoned := false
			for _, call := range []string{"generate", "stream"} {
				ctx, cancel := llmprovider.LiveCtx(t)
				var resp *llmprovider.Response
				var genErr error
				if call == "generate" {
					resp, genErr = p.Generate(ctx, req())
				} else {
					for ev, err := range llmprovider.Stream(ctx, p, req()) {
						if err != nil {
							genErr = err
							break
						}
						if ev.Type == llmprovider.EventDone {
							resp = ev.Response
						}
					}
				}
				cancel()
				if resp == nil {
					llmprovider.SkipIfTransient(t, genErr)
					t.Fatalf("%s %s (%s): no response: %v", tg.name, call, model, genErr)
				}
				r, err := parseReasoning0037(rec.last())
				if err != nil {
					t.Fatalf("%s %s (%s): reply not read: %v", tg.name, call, model, err)
				}
				t.Logf("%s %s: model %s, stream %t, raw usage %s, decoded %+v", tg.name, call, model, r.stream,
					strings.Join(r.raw, " | "), resp.Usage)
				for _, problem := range append(r.problems(), r.decodedProblems(resp.Usage)...) {
					t.Errorf("%s %s: %s", tg.name, call, problem)
				}
				reasoned = reasoned || r.thinking > 0
			}
			if !reasoned {
				t.Errorf("%s (%s): neither call reported thinking_tokens above zero", tg.name, model)
			}
		})
	}
}
