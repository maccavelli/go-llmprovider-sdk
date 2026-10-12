//go:build live_gateways

package llmprovider_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// 0036-PLAN Phase 1's measurement. TestLive_0036CacheWrites runs only with
// LLMPROVIDER_LIVE_0036 set, and each target only with its key. It sends one
// long prompt twice to each target, and logs each reply's raw usage object
// and the decoded Usage. It logs no prompt, header or key. It asserts only
// what 0036-MADR states as fact, through problems.

// usage0036 is what one reply says of its cache, in its wire's spelling.
type usage0036 struct {
	raw json.RawMessage
	// input is the whole input in the wire's own convention: on Messages,
	// input_tokens plus the cache reads and writes.
	input, read, write int
	// hasWrite says whether the reply carries a write field at all.
	hasWrite bool
	// creation is Messages' cache_creation object, when present.
	creation *struct{ m5, h1 int }
}

// parseUsage0036 reads the usage of a reply to a POST at path.
func parseUsage0036(path string, body []byte) (usage0036, error) {
	var top struct {
		Usage json.RawMessage `json:"usage"`
	}
	if err := json.Unmarshal(body, &top); err != nil {
		return usage0036{}, err
	}
	u := usage0036{raw: top.Usage}
	if len(top.Usage) == 0 {
		return u, nil
	}
	switch {
	case strings.HasSuffix(path, "/messages"):
		var m struct {
			Input         int  `json:"input_tokens"`
			Read          int  `json:"cache_read_input_tokens"`
			Write         *int `json:"cache_creation_input_tokens"`
			CacheCreation *struct {
				M5 int `json:"ephemeral_5m_input_tokens"`
				H1 int `json:"ephemeral_1h_input_tokens"`
			} `json:"cache_creation"`
		}
		if err := json.Unmarshal(top.Usage, &m); err != nil {
			return u, err
		}
		u.read, u.hasWrite = m.Read, m.Write != nil
		if m.Write != nil {
			u.write = *m.Write
		}
		u.input = m.Input + u.read + u.write
		if m.CacheCreation != nil {
			u.creation = &struct{ m5, h1 int }{m.CacheCreation.M5, m.CacheCreation.H1}
		}
	case strings.HasSuffix(path, "/responses"):
		var r struct {
			Input   int `json:"input_tokens"`
			Details struct {
				Read  int  `json:"cached_tokens"`
				Write *int `json:"cache_write_tokens"`
			} `json:"input_tokens_details"`
		}
		if err := json.Unmarshal(top.Usage, &r); err != nil {
			return u, err
		}
		u.input, u.read, u.hasWrite = r.Input, r.Details.Read, r.Details.Write != nil
		if r.Details.Write != nil {
			u.write = *r.Details.Write
		}
	case strings.HasSuffix(path, "/chat/completions"):
		// The write is read from the details, else from Kilo's top-level
		// spelling (0036-MADR amendment 2026-10-12).
		var c struct {
			Prompt  int `json:"prompt_tokens"`
			Details struct {
				Read  int  `json:"cached_tokens"`
				Write *int `json:"cache_write_tokens"`
			} `json:"prompt_tokens_details"`
			TopWrite *int `json:"cache_creation_input_tokens"`
		}
		if err := json.Unmarshal(top.Usage, &c); err != nil {
			return u, err
		}
		u.input, u.read = c.Prompt, c.Details.Read
		for _, w := range []*int{c.Details.Write, c.TopWrite} {
			if w != nil {
				u.write, u.hasWrite = *w, true
				break
			}
		}
	default:
		return u, fmt.Errorf("no usage spelling for %s", path)
	}
	return u, nil
}

// problems lists what breaks 0036-MADR's stated facts: no count is negative,
// reads and writes are parts of the input and do not overlap, and Messages'
// cache_creation sums to cache_creation_input_tokens.
func (u usage0036) problems() []string {
	var out []string
	if u.input < 0 || u.read < 0 || u.write < 0 {
		out = append(out, fmt.Sprintf("a negative count: input %d, read %d, write %d", u.input, u.read, u.write))
	}
	if u.read+u.write > u.input {
		out = append(out, fmt.Sprintf("read %d and write %d sum above the input %d", u.read, u.write, u.input))
	}
	if c := u.creation; c != nil && c.m5+c.h1 != u.write {
		out = append(out, fmt.Sprintf("cache_creation sums to %d (5m %d, 1h %d), not cache_creation_input_tokens %d",
			c.m5+c.h1, c.m5, c.h1, u.write))
	}
	return out
}

// TestCacheWriteChecks_0036 proves the measurement's checks on canned bodies,
// with no key and no network (0036-PLAN Phase 1, step 4).
func TestCacheWriteChecks_0036(t *testing.T) {
	for _, c := range []struct {
		name, wire, body, want string
	}{
		{"messages, clean", "/messages", `{"usage":{"input_tokens":15,"cache_read_input_tokens":100,"cache_creation_input_tokens":5,` +
			`"cache_creation":{"ephemeral_5m_input_tokens":3,"ephemeral_1h_input_tokens":2},"output_tokens":40}}`, ""},
		{"responses, clean", "/responses", `{"usage":{"input_tokens":120,"input_tokens_details":{"cached_tokens":100,"cache_write_tokens":5},"output_tokens":40}}`, ""},
		{"chat, clean", "/chat/completions", `{"usage":{"prompt_tokens":120,"prompt_tokens_details":{"cached_tokens":100,"cache_write_tokens":5},"completion_tokens":40}}`, ""},
		{"chat, no cache fields", "/chat/completions", `{"usage":{"prompt_tokens":7,"completion_tokens":3}}`, ""},
		{"chat, Kilo's top-level write, clean", "/chat/completions",
			`{"usage":{"prompt_tokens":8730,"prompt_tokens_details":{"cached_tokens":0},"cache_creation_input_tokens":8727,"completion_tokens":5}}`, ""},
		{"chat, Kilo's top-level write above the input", "/chat/completions",
			`{"usage":{"prompt_tokens":100,"prompt_tokens_details":{"cached_tokens":90},"cache_creation_input_tokens":20,"completion_tokens":4}}`, "above the input"},
		{"chat, the details before the top level", "/chat/completions",
			`{"usage":{"prompt_tokens":100,"prompt_tokens_details":{"cached_tokens":90,"cache_write_tokens":5},"cache_creation_input_tokens":20,"completion_tokens":4}}`, ""},
		{"responses, read and write above the input", "/responses",
			`{"usage":{"input_tokens":100,"input_tokens_details":{"cached_tokens":90,"cache_write_tokens":20},"output_tokens":4}}`, "above the input"},
		{"chat, read and write above the input", "/chat/completions",
			`{"usage":{"prompt_tokens":100,"prompt_tokens_details":{"cached_tokens":90,"cache_write_tokens":20},"completion_tokens":4}}`, "above the input"},
		{"messages, cache_creation not summing", "/messages", `{"usage":{"input_tokens":15,"cache_creation_input_tokens":5,` +
			`"cache_creation":{"ephemeral_5m_input_tokens":3,"ephemeral_1h_input_tokens":3},"output_tokens":40}}`, "cache_creation sums"},
		{"responses, a negative write", "/responses",
			`{"usage":{"input_tokens":100,"input_tokens_details":{"cache_write_tokens":-1},"output_tokens":4}}`, "negative"},
	} {
		t.Run(c.name, func(t *testing.T) {
			u, err := parseUsage0036(c.wire, []byte(c.body))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			problems := strings.Join(u.problems(), "; ")
			switch {
			case c.want == "" && problems != "":
				t.Errorf("problems %q, want none", problems)
			case c.want != "" && !strings.Contains(problems, c.want):
				t.Errorf("problems %q, want one containing %q", problems, c.want)
			}
		})
	}
}

// postRecorder keeps the path and body of each reply to a POST, passing the
// reply on.
type postRecorder struct {
	mu      sync.Mutex
	replies []struct {
		path string
		body []byte
	}
}

func (r *postRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
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
	r.replies = append(r.replies, struct {
		path string
		body []byte
	}{req.URL.Path, b})
	r.mu.Unlock()
	resp.Body = io.NopCloser(bytes.NewReader(b))
	return resp, nil
}

// last is the path and body of the last reply to a POST.
func (r *postRecorder) last() (string, []byte, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.replies) == 0 {
		return "", nil, false
	}
	l := r.replies[len(r.replies)-1]
	return l.path, l.body, true
}

// prompt0036 is the same long prompt on every request: about 6,000 tokens of
// numbered plain text, past every service's caching minimum.
func prompt0036() string {
	var b strings.Builder
	b.WriteString("You are a terse assistant. The reference list below is context only.\n")
	for i := 1; i <= 300; i++ {
		fmt.Fprintf(&b, "Entry %03d: the catalogue record for item %d lists alpha, bravo, charlie, delta, echo, foxtrot and golf.\n", i, i)
	}
	return b.String()
}

// target0036 is one row of 0036-PLAN Phase 1's target table.
type target0036 struct {
	name  string
	id    llmprovider.ProviderID
	key   func(t *testing.T) string
	model func(t *testing.T) string
}

func envKey0036(id llmprovider.ProviderID) func(t *testing.T) string {
	return func(t *testing.T) string { return llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[id]) }
}

func fixed0036(model string) func(t *testing.T) string {
	return func(*testing.T) string { return model }
}

func picked0036(id llmprovider.ProviderID, candidates ...string) func(t *testing.T) string {
	return func(t *testing.T) string { return llmprovider.LiveModel(t, id, candidates...) }
}

// TestLive_0036CacheWrites is 0036-PLAN Phase 1's measurement. Billed: two
// requests per target, eight targets, about 6,000 input and at most 16 output
// tokens each.
func TestLive_0036CacheWrites(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0036") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0036 to run 0036-PLAN Phase 1's measurement")
	}
	zen, kilo := llmprovider.ProviderOpencodeZen, llmprovider.ProviderKilo
	targets := []target0036{
		{"T1 openai", llmprovider.ProviderOpenAI, envKey0036(llmprovider.ProviderOpenAI), fixed0036("gpt-6-luna")},
		{"T2 claude", llmprovider.ProviderClaude, envKey0036(llmprovider.ProviderClaude), fixed0036("claude-haiku-5-5")},
		{"T3 grok", llmprovider.ProviderGrok, envKey0036(llmprovider.ProviderGrok), fixed0036("grok-4.6")},
		{"T4 kilo openai", kilo, llmprovider.LiveKiloKey, picked0036(kilo, "openai/gpt-6-luna", "openai/gpt-6.1-sol", "openai/gpt-5.6-luna")},
		{"T5 kilo claude", kilo, llmprovider.LiveKiloKey, picked0036(kilo, "anthropic/claude-haiku-5.5", "anthropic/claude-sonnet-5.5")},
		{"T6 zen messages", zen, llmprovider.LiveOpencodeKey, picked0036(zen, "claude-haiku-5-5", "claude-sonnet-5-5")},
		{"T7 zen responses", zen, llmprovider.LiveOpencodeKey, picked0036(zen, "gpt-6-luna", "gpt-6.1-sol")},
		{"T8 zen chat", zen, llmprovider.LiveOpencodeKey, picked0036(zen, "deepseek-v4.1-flash")},
	}
	prompt := prompt0036()
	for _, tg := range targets {
		t.Run(tg.name, func(t *testing.T) {
			key := tg.key(t)
			model := tg.model(t)
			rec := &postRecorder{}
			p, err := providers.New(tg.id, llmprovider.WithAPIKey(key), llmprovider.WithModel(model),
				llmprovider.WithModelProbes(false), llmprovider.WithMaxTokens(16),
				llmprovider.WithHTTPClient(&http.Client{Transport: rec}))
			if err != nil {
				t.Fatal(err)
			}
			for n := 1; n <= 2; n++ {
				if n == 2 {
					time.Sleep(3 * time.Second)
				}
				req := &llmprovider.Request{Instructions: prompt,
					Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Reply with the word ok."}}}
				ctx, cancel := llmprovider.LiveCtx(t)
				resp, genErr := p.Generate(ctx, req)
				cancel()
				path, body, ok := rec.last()
				if !ok {
					llmprovider.SkipIfTransient(t, genErr)
					t.Fatalf("%s request %d (%s): no reply: %v", tg.name, n, model, genErr)
				}
				u, err := parseUsage0036(path, body)
				if err != nil {
					llmprovider.SkipIfTransient(t, genErr)
					t.Fatalf("%s request %d (%s): reply at %s not read: %v (generate: %v)", tg.name, n, model, path, err, genErr)
				}
				if len(u.raw) == 0 || string(u.raw) == "null" {
					// No usage to measure: a quota or rate-limit error skips the
					// target, and any other error fails it.
					llmprovider.SkipIfTransient(t, genErr)
					t.Fatalf("%s request %d (%s): no usage at %s (generate: %v)", tg.name, n, model, path, genErr)
				}
				decoded := "n/a"
				if resp != nil {
					decoded = fmt.Sprintf("%+v", resp.Usage)
				}
				t.Logf("%s request %d: model %s, path %s, write field present %t, raw usage %s, decoded %s, generate error %v",
					tg.name, n, model, path, u.hasWrite, u.raw, decoded, genErr)
				for _, problem := range u.problems() {
					t.Errorf("%s request %d: %s", tg.name, n, problem)
				}
			}
		})
	}
}
