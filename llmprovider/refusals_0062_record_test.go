package llmprovider_test

// 0062-PLAN Phase 1's recorder of each wire's thinking refusals: its case
// table, the transport that rewrites each request to carry a case's fields,
// and the writer of one JSON record per case. Untagged, so that the offline
// test (refusals_0062_test.go) runs the code the live recorder
// (live_0062_refusals_measure_test.go) runs, as live_twins_test.go runs
// live_checks_test.go's (0046-MADR D3). A record holds a status, an error's
// code, message and redacted body, and a count of tokens; never a key or a
// model's text.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// The wires of 0062-MADR D1, as the fixtures' directories name them.
const (
	wireMessages        = "messages"
	wireResponses       = "responses"
	wireInteractions    = "interactions"
	wireGenerateContent = "generatecontent"
	wireChatCompletions = "chatcompletions"
)

// The causes a case sends: a thinking field, or a failure of another kind
// (the case table's last row), which no matcher may match (0062-MADR D4).
const (
	causeThinking = "thinking"
	causeOther    = "other"
)

// The outcomes a record holds (0062-PLAN Phase 1, steps 1 and 2; Q8 (b)).
const (
	outcomeAccepted    = "accepted"
	outcomeRefused     = "refused"
	outcomeUnreachable = "unreachable"
	outcomeOther       = "other"
	outcomeSkipped     = "skipped"
)

// refusalBodyLimit bounds a record's error body: 4 KiB (0062-PLAN Phase 1,
// step 1).
const refusalBodyLimit = 4 << 10

// refusalPrompt is every case's input.
const refusalPrompt = "Reply with OK."

// stringSchema is the other-cause case's tool schema: not an object.
const stringSchema = `{"type":"string"}`

// stringSchemaTool is the other-cause case's tool.
var stringSchemaTool = llmprovider.Tool{Name: "record_city", Description: "Records a city name.",
	Schema: json.RawMessage(stringSchema)}

// refusalCase is one row of 0062-PLAN Phase 1's case table, one field set
// each.
type refusalCase struct {
	id       string
	cause    string
	wire     string
	provider llmprovider.ProviderID
	route    opencode.Route
	model    string
	// set holds the body fields the rewriter sets, by dotted path; drop the
	// paths it removes, so that the field a case measures is not sent beside
	// another of the same kind.
	set  map[string]any
	drop []string
	// tool sends stringSchemaTool.
	tool bool
}

// refusalRoute is a provider and route of the case table, with the model its
// other-cause cases use: its first row's.
type refusalRoute struct {
	key      string
	wire     string
	provider llmprovider.ProviderID
	route    opencode.Route
	model    string
}

// inputPath is where each wire's body holds the conversation.
var inputPath = map[string]string{
	wireMessages:        "messages",
	wireResponses:       "input",
	wireInteractions:    "input",
	wireGenerateContent: "contents",
	wireChatCompletions: "messages",
}

// refusalRoutes are the case table's providers and routes, in its order.
// Ollama is recorded in Phase 5 (0062-MADR Q9 (c)).
var refusalRoutes = []refusalRoute{
	{"claude", wireMessages, llmprovider.ProviderClaude, "", "claude-sonnet-5"},
	{"opencode-go-messages", wireMessages, llmprovider.ProviderOpencodeGo, opencode.RouteMessages, "qwen3.8-flash"},
	{"openai", wireResponses, llmprovider.ProviderOpenAI, "", "gpt-6-luna"},
	{"grok", wireResponses, llmprovider.ProviderGrok, "", "grok-4.5"},
	{"opencode-zen-responses", wireResponses, llmprovider.ProviderOpencodeZen, opencode.RouteResponses, "gpt-5.5"},
	{"gemini", wireInteractions, llmprovider.ProviderGemini, "", "gemini-3.8-flash"},
	{"opencode-zen-google", wireGenerateContent, llmprovider.ProviderOpencodeZen, opencode.RouteGoogle, "gemini-3.8-flash"},
	{"huggingface", wireChatCompletions, llmprovider.ProviderHuggingFace, "", "deepseek-ai/DeepSeek-V4.1-Flash"},
	{"together", wireChatCompletions, llmprovider.ProviderTogether, "", "openai/gpt-oss-120b"},
	{"kilo", wireChatCompletions, llmprovider.ProviderKilo, "", "openai/gpt-6-luna"},
	{"opencode-zen-chat", wireChatCompletions, llmprovider.ProviderOpencodeZen, opencode.RouteChatCompletions, "glm-5.3"},
}

// refusalCases returns the case table: each thinking field of 0062-PLAN
// Phase 1 step 2, then the two other causes on each provider and route.
func refusalCases() []refusalCase {
	route := map[string]refusalRoute{}
	for _, r := range refusalRoutes {
		route[r.key] = r
	}
	thinking := func(id, key, model string, set map[string]any, drop ...string) refusalCase {
		r := route[key]
		return refusalCase{id: id, cause: causeThinking, wire: r.wire, provider: r.provider, route: r.route,
			model: model, set: set, drop: drop}
	}
	adaptive := map[string]any{"type": "adaptive"}
	effort := func(path, value string) map[string]any { return map[string]any{path: value} }
	cases := []refusalCase{
		// Messages: R1 and R2, and the efforts.
		thinking("claude-sonnet-5-budget", "claude", "claude-sonnet-5",
			map[string]any{"thinking": map[string]any{"type": "enabled", "budget_tokens": 4096}}, "output_config"),
		thinking("claude-sonnet-5-adaptive-max", "claude", "claude-sonnet-5",
			map[string]any{"thinking": adaptive, "output_config.effort": "max"}),
		thinking("claude-sonnet-5-adaptive-bogus", "claude", "claude-sonnet-5",
			map[string]any{"thinking": adaptive, "output_config.effort": "bogus"}),
		thinking("claude-sonnet-4-5-adaptive-high", "claude", "claude-sonnet-4-5-20250929",
			map[string]any{"thinking": adaptive, "output_config.effort": "high"}),
		thinking("opencode-go-messages-qwen3.8-flash-adaptive", "opencode-go-messages", "qwen3.8-flash",
			map[string]any{"thinking": adaptive}, "output_config"),
		thinking("opencode-go-messages-minimax-m3-adaptive-high", "opencode-go-messages", "minimax-m3",
			map[string]any{"thinking": adaptive, "output_config.effort": "high"}),
		// Responses: the efforts, and R6.
		thinking("openai-gpt-6-luna-max", "openai", "gpt-6-luna", effort("reasoning.effort", "max")),
		thinking("openai-gpt-6-luna-bogus", "openai", "gpt-6-luna", effort("reasoning.effort", "bogus")),
		thinking("openai-gpt-4.1-mini-low", "openai", "gpt-4.1-mini", effort("reasoning.effort", "low")),
		thinking("grok-4.5-xhigh", "grok", "grok-4.5", effort("reasoning.effort", "xhigh")),
		thinking("grok-4.5-max", "grok", "grok-4.5", effort("reasoning.effort", "max")),
		thinking("grok-4.3-max", "grok", "grok-4.3", effort("reasoning.effort", "max")),
		// Models outside R6's menu, which it sends no effort (0062-PLAN Phase 2,
		// the stop of 2026-10-25).
		thinking("grok-build-0.1-high", "grok", "grok-build-0.1", effort("reasoning.effort", "high")),
		thinking("grok-4.20-0309-non-reasoning-high", "grok", "grok-4.20-0309-non-reasoning",
			effort("reasoning.effort", "high")),
		thinking("opencode-zen-responses-gpt-5.5-max", "opencode-zen-responses", "gpt-5.5",
			effort("reasoning.effort", "max")),
		thinking("opencode-zen-responses-gpt-5.5-bogus", "opencode-zen-responses", "gpt-5.5",
			effort("reasoning.effort", "bogus")),
		// Interactions: Gemini's xhigh, and R5.
		thinking("gemini-3.8-flash-max", "gemini", "gemini-3.8-flash", effort("generation_config.thinking_level", "max")),
		thinking("gemini-3.8-flash-xhigh", "gemini", "gemini-3.8-flash", effort("generation_config.thinking_level", "xhigh")),
		thinking("gemini-3.8-flash-bogus", "gemini", "gemini-3.8-flash", effort("generation_config.thinking_level", "bogus")),
		thinking("gemini-2.5-flash-lite-medium", "gemini", "gemini-2.5-flash-lite",
			effort("generation_config.thinking_level", "medium")),
		thinking("gemini-2.5-flash-lite-low", "gemini", "gemini-2.5-flash-lite",
			effort("generation_config.thinking_level", "low")),
		// generateContent: a level, a budget, and R4.
		thinking("opencode-zen-google-gemini-3.8-flash-level-max", "opencode-zen-google", "gemini-3.8-flash",
			effort("generationConfig.thinkingConfig.thinkingLevel", "max"), "generationConfig.thinkingConfig.thinkingBudget"),
		thinking("opencode-zen-google-gemini-3.8-flash-budget", "opencode-zen-google", "gemini-3.8-flash",
			map[string]any{"generationConfig.thinkingConfig.thinkingBudget": 4096},
			"generationConfig.thinkingConfig.thinkingLevel"),
		thinking("opencode-zen-google-gemini-2.5-flash-level-low", "opencode-zen-google", "gemini-2.5-flash",
			effort("generationConfig.thinkingConfig.thinkingLevel", "low"), "generationConfig.thinkingConfig.thinkingBudget"),
		// Each level, for D2's level against 0013-MADR B9's dynamic budget
		// (0062-PLAN Phase 3, the stop of 2026-10-25).
		thinking("opencode-zen-google-gemini-3.8-flash-level-low", "opencode-zen-google", "gemini-3.8-flash",
			effort("generationConfig.thinkingConfig.thinkingLevel", "low"), "generationConfig.thinkingConfig.thinkingBudget"),
		thinking("opencode-zen-google-gemini-3.8-flash-level-medium", "opencode-zen-google", "gemini-3.8-flash",
			effort("generationConfig.thinkingConfig.thinkingLevel", "medium"), "generationConfig.thinkingConfig.thinkingBudget"),
		thinking("opencode-zen-google-gemini-3.8-flash-level-high", "opencode-zen-google", "gemini-3.8-flash",
			effort("generationConfig.thinkingConfig.thinkingLevel", "high"), "generationConfig.thinkingConfig.thinkingBudget"),
		thinking("opencode-zen-google-gemini-3.8-flash-level-xhigh", "opencode-zen-google", "gemini-3.8-flash",
			effort("generationConfig.thinkingConfig.thinkingLevel", "xhigh"), "generationConfig.thinkingConfig.thinkingBudget"),
		// Chat Completions: the efforts, and the chat route without metadata.
		thinking("huggingface-deepseek-v4.1-flash-max", "huggingface", "deepseek-ai/DeepSeek-V4.1-Flash",
			effort("reasoning_effort", "max")),
		thinking("huggingface-deepseek-v4.1-flash-bogus", "huggingface", "deepseek-ai/DeepSeek-V4.1-Flash",
			effort("reasoning_effort", "bogus")),
		thinking("together-gpt-oss-120b-max", "together", "openai/gpt-oss-120b", effort("reasoning_effort", "max")),
		thinking("together-gpt-oss-120b-bogus", "together", "openai/gpt-oss-120b", effort("reasoning_effort", "bogus")),
		thinking("kilo-gpt-6-luna-max", "kilo", "openai/gpt-6-luna", effort("reasoning.effort", "max")),
		thinking("kilo-gpt-6-luna-bogus", "kilo", "openai/gpt-6-luna", effort("reasoning.effort", "bogus")),
		thinking("opencode-zen-chat-glm-5.3-max", "opencode-zen-chat", "glm-5.3", effort("reasoning_effort", "max")),
	}
	for _, r := range refusalRoutes {
		other := refusalCase{cause: causeOther, wire: r.wire, provider: r.provider, route: r.route, model: r.model}
		empty, tool := other, other
		empty.id, empty.set = r.key+"-empty-input", map[string]any{inputPath[r.wire]: []any{}}
		tool.id, tool.tool = r.key+"-string-schema-tool", true
		cases = append(cases, empty, tool)
	}
	return cases
}

// refusalRecord is one case's JSON record (0062-PLAN Phase 1, step 1).
type refusalRecord struct {
	Case     string         `json:"case"`
	Cause    string         `json:"cause"`
	Wire     string         `json:"wire"`
	Provider string         `json:"provider"`
	Route    string         `json:"route,omitempty"`
	Model    string         `json:"model"`
	Set      map[string]any `json:"set,omitempty"`
	Dropped  []string       `json:"dropped,omitempty"`
	Tool     string         `json:"tool,omitempty"`
	// Sends counts the requests sent, a retry's included.
	Sends   int    `json:"sends"`
	Status  int    `json:"status,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	// Body is the last reply's error body, redacted and cut to 4 KiB.
	Body string `json:"body,omitempty"`
	// Error is a failure that is not the service's, redacted.
	Error string `json:"error,omitempty"`
	// Usage reports whether the reply counts tokens: an accepted reply's
	// Usage, or an error body's usage field.
	Usage           bool   `json:"usage"`
	ReasoningTokens int    `json:"reasoning_tokens,omitempty"`
	Outcome         string `json:"outcome"`
}

// record is c's record before anything is sent.
func (c refusalCase) record() refusalRecord {
	r := refusalRecord{Case: c.id, Cause: c.cause, Wire: c.wire, Provider: string(c.provider), Route: string(c.route),
		Model: c.model, Set: c.set, Dropped: c.drop}
	if c.tool {
		r.Tool = stringSchema
	}
	return r
}

// skipped is c's record when its provider's key is not set (0062-MADR Q8).
func (c refusalCase) skipped() refusalRecord {
	r := c.record()
	r.Outcome = outcomeSkipped
	return r
}

// refusalRewriter sets and drops a case's fields in each JSON request body,
// sends it on base, and keeps the last reply's status and body.
type refusalRewriter struct {
	set  map[string]any
	drop []string
	base http.RoundTripper

	mu     sync.Mutex
	sends  int
	status int
	body   []byte
}

func (rw *refusalRewriter) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodPost && req.Body != nil {
		raw, err := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, fmt.Errorf("refusal recorder: a request body that is not a JSON object: %w", err)
		}
		for path, value := range rw.set {
			setPath(body, path, value)
		}
		for _, path := range rw.drop {
			dropPath(body, path)
		}
		if raw, err = json.Marshal(body); err != nil {
			return nil, err
		}
		req = req.Clone(req.Context())
		req.Body = io.NopCloser(bytes.NewReader(raw))
		req.ContentLength = int64(len(raw))
		req.Header.Set("Content-Length", strconv.Itoa(len(raw)))
	}
	resp, err := rw.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(b))
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.sends++
	rw.status, rw.body = resp.StatusCode, b
	return resp, nil
}

func (rw *refusalRewriter) result() (sends, status int, body []byte) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	return rw.sends, rw.status, rw.body
}

// setPath sets body's field at the dotted path to value, making the objects
// on the way.
func setPath(body map[string]any, path string, value any) {
	keys := strings.Split(path, ".")
	for _, k := range keys[:len(keys)-1] {
		next, ok := body[k].(map[string]any)
		if !ok {
			next = map[string]any{}
			body[k] = next
		}
		body = next
	}
	body[keys[len(keys)-1]] = value
}

// dropPath removes body's field at the dotted path, if it is there.
func dropPath(body map[string]any, path string) {
	keys := strings.Split(path, ".")
	for _, k := range keys[:len(keys)-1] {
		next, ok := body[k].(map[string]any)
		if !ok {
			return
		}
		body = next
	}
	delete(body, keys[len(keys)-1])
}

// refusalOutcome classifies a reply by its status alone, so that a record
// says what was sent back, not what it is taken to mean: a 404 is the model
// unreachable; any other 4xx but 401, 403, 408, 409 and 429 is a refusal;
// the rest, a transport failure included, is other. The report reads each
// refusal's message for its cause.
func refusalOutcome(err error) string {
	if err == nil {
		return outcomeAccepted
	}
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	if !ok {
		return outcomeOther
	}
	switch s := apiErr.Status; {
	case s == http.StatusNotFound:
		return outcomeUnreachable
	case s >= 400 && s < 500 && s != http.StatusUnauthorized && s != http.StatusForbidden &&
		s != http.StatusRequestTimeout && s != http.StatusConflict && s != http.StatusTooManyRequests:
		return outcomeRefused
	default:
		return outcomeOther
	}
}

// recordCase sends c's request through its provider, with key and opts, and
// returns its record.
func recordCase(ctx context.Context, c refusalCase, key string, opts ...llmprovider.Option) (refusalRecord, error) {
	rw := &refusalRewriter{set: c.set, drop: c.drop, base: http.DefaultTransport}
	all := []llmprovider.Option{llmprovider.WithAPIKey(key), llmprovider.WithModel(c.model),
		llmprovider.WithModelProbes(false), llmprovider.WithoutModelMetadata(),
		llmprovider.WithHTTPClient(&http.Client{Transport: rw})}
	if c.route != "" {
		all = append(all, opencode.WithRoute(c.route))
	}
	p, err := providers.New(c.provider, append(all, opts...)...)
	if err != nil {
		return refusalRecord{}, fmt.Errorf("%s: %w", c.id, err)
	}
	req := &llmprovider.Request{
		Input:     []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: refusalPrompt}},
		Reasoning: &llmprovider.Reasoning{Effort: llmprovider.EffortHigh}}
	if c.tool {
		req.Tools = []llmprovider.Tool{stringSchemaTool}
	}
	resp, genErr := p.Generate(ctx, req)
	r := c.record()
	var status int
	var body []byte
	r.Sends, status, body = rw.result()
	r.Outcome = refusalOutcome(genErr)
	switch apiErr, ok := errors.AsType[*llmprovider.APIError](genErr); {
	case genErr == nil:
		r.Status = status
		r.Usage = resp.Usage.InputTokens+resp.Usage.OutputTokens > 0
		r.ReasoningTokens = resp.Usage.ReasoningTokens
	case ok:
		r.Status, r.Code, r.Message = apiErr.Status, apiErr.Code, apiErr.Message
		if status/100 != 2 {
			r.Body = boundRefusalBody(redact.Redact(body))
			r.Usage = bytes.Contains(body, []byte(`"usage"`))
		}
	default:
		r.Error = redact.String(genErr.Error())
	}
	return r, nil
}

// boundRefusalBody is b cut to refusalBodyLimit bytes, on a rune boundary.
func boundRefusalBody(b []byte) string {
	if len(b) <= refusalBodyLimit {
		return string(b)
	}
	cut := refusalBodyLimit
	for cut > 0 && !utf8.RuneStart(b[cut]) {
		cut--
	}
	return string(b[:cut])
}

// writeRefusalRecord writes r to dir/<wire>/<case>.json, as golden text.
func writeRefusalRecord(dir string, r refusalRecord) error {
	enc, err := wiretest.Encode(r)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, r.Wire, r.Case+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, enc, 0o600)
}

// outsideModule refuses an output directory inside the module, whose root
// is the parent of this package's directory: records are copied into the
// tree only after they are redacted and checked (0062-PLAN Phase 1, step 5).
func outsideModule(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	root, err := filepath.Abs("..")
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		// Another volume: outside the module.
		return nil //nolint:nilerr // filepath.Rel fails only across volumes
	}
	if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return fmt.Errorf("refusal recorder: %s is inside the module; name a directory outside it", dir)
	}
	return nil
}
