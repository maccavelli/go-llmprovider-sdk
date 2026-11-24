package wire

import (
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// A refusal of an output limit as evidence (0089-MADR D4, DL68): the forms
// that state a number, which route reads which, and the validator that
// decides whether a refusal's number may be used. The cap forms' texts are
// the forum copies 0089-MADR W7 names, none a recorded reply, but Kilo's
// range form, measured by a live probe (DL95).

// LimitForms is a set of refusal forms that state a limit.
type LimitForms uint8

const (
	// FormAnthropicCap is Anthropic's "max_tokens: N > M, which is the
	// maximum allowed number of output tokens for <model>" (W7).
	FormAnthropicCap LimitForms = 1 << iota
	// FormOpenAICap is OpenAI's Chat Completions "This model supports at
	// most M completion tokens, whereas you provided N" (W7).
	FormOpenAICap
	// FormGeminiRange is generateContent's "value of N but the supported
	// range is from 1 (inclusive) to M (exclusive)", read as M - 1 (W7).
	FormGeminiRange
	// FormOpenRouterWindow is OpenRouter's window form as Kilo passes it:
	// "maximum context length is W tokens. However, you requested about T
	// tokens" (0088-MADR C39).
	FormOpenRouterWindow
	// FormKiloWindow is Kilo's documented window form: "exceeds the model's
	// context window of W tokens. Your request contains approximately T
	// tokens" (0088-MADR C97).
	FormKiloWindow
	// FormSplitWindow is OpenAI's split, which Together sends too: "maximum
	// context length is W tokens ... (I in the messages, C in the
	// completion)" (0089-MADR C27; 0088-MADR C38).
	FormSplitWindow
	// FormKiloRange is Kilo's range form, measured on 2026-11-24 (0089-MADR
	// DL95): "The max_tokens parameter is illegal.：限制数值范围[L,M]", M
	// inclusive. It states no value sent and no model, so its lower bound
	// must be 1 in V2's place, and V3 and V4 hold as for every cap form (OQ6
	// (1)).
	FormKiloRange
)

// LimitWire is what one route reads of a refusal of its output limit: the
// forms it reads, and the least value its field takes (V4).
type LimitWire struct {
	Forms   LimitForms
	Minimum int
}

// The routes' LimitWire, by 0089-MADR DL68: Anthropic's cap form on the
// Messages wire (Claude, OpenCode's messages route) and on Kilo, as an
// upstream's text that V6 checks; OpenAI's on the Chat Completions wires,
// not on Responses, whose text is not established; Gemini's range on
// generateContent (OpenCode's google route), not on Interactions; Kilo's
// two window forms on Kilo (0088-MADR ST51, C39), and its range form
// (DL95); the split on the Chat Completions wires (C38). Responses' field
// takes at least 16 (W8).
var (
	MessagesLimits = LimitWire{Forms: FormAnthropicCap, Minimum: 1}
	KiloLimits     = LimitWire{Forms: FormAnthropicCap | FormOpenAICap | FormKiloRange | FormOpenRouterWindow |
		FormKiloWindow, Minimum: 1}
	ChatCompletionsLimits = LimitWire{Forms: FormOpenAICap | FormSplitWindow, Minimum: 1}
	ResponsesLimits       = LimitWire{Minimum: 16}
	GenerateContentLimits = LimitWire{Forms: FormGeminiRange, Minimum: 1}
	InteractionsLimits    = LimitWire{Minimum: 1}
)

// LimitAsk is the request a refusal answered: Sent, the value its
// output-limit field carried, after AddThinkingFor raised it (V2, DL67), 0
// for none; Model, as the request named it; and Resolved, the id a listing or
// one-model read resolved Model to, empty for none (V6).
type LimitAsk struct {
	Sent     int
	Model    string
	Resolved string
}

// LimitReading is a validated refusal of an output limit: a cap form's Cap,
// or a window form's Window and Input.
type LimitReading struct {
	// Form is the one form read.
	Form LimitForms
	// Cap is a cap form's stated maximum: the most the field may carry. 0
	// for a window form.
	Cap int
	// Window is a window form's stated window, and Input the input it
	// states, or, where it states the request's total, that total less Sent
	// (0088-MADR ST92). Both 0 for a cap form.
	Window, Input int
	// Keep reports, for a cap form, that the refusal names no model or names
	// the request's (V6): the cap may be kept and reported. A cap that is not
	// kept serves this request's resend only; a window is never kept (D5).
	Keep bool
}

// limitForm is one form: its pattern, and the numbers its groups hold.
type limitForm struct {
	form    LimitForms
	pattern *regexp.Regexp
}

// The forms, cap forms first. Each pattern's groups are its numbers in the
// order the text states them; the Anthropic form's last group is the model.
var (
	capLimitForms = []limitForm{
		{FormAnthropicCap, regexp.MustCompile(`max_tokens: (\d+) > (\d+), which is the maximum allowed number of output tokens(?: for ([^\s,;]+))?`)},
		{FormOpenAICap, regexp.MustCompile(`supports at most (\d+) completion tokens, whereas you provided (\d+)`)},
		{FormGeminiRange, regexp.MustCompile(`value of (\d+) but the supported range is from 1 \(inclusive\) to (\d+) \(exclusive\)`)},
		{FormKiloRange, regexp.MustCompile(`max_tokens parameter is illegal\.\s*[：:]\s*限制数值范围\[(\d+),\s*(\d+)\]`)},
	}
	windowLimitForms = []limitForm{
		{FormOpenRouterWindow, regexp.MustCompile(`maximum context length is (\d+) tokens\. However, you requested about (\d+) tokens`)},
		{FormKiloWindow, regexp.MustCompile(`exceeds the model's context window of (\d+) tokens\. Your request contains approximately (\d+) tokens`)},
		{FormSplitWindow, regexp.MustCompile(`maximum context length is (\d+) tokens.*?\((\d+) in the messages, (\d+) in the completion\)`)},
	}
)

// ReadLimitRefusal validates err as a refusal of the output limit ask sent,
// on a route that reads w (0089-MADR D4). err must be an *APIError of kind
// ErrInvalidRequest. Then: (V1) a text matching a cap form and a window form
// is rejected; a cap form never gives a window, nor a window form a cap;
// (V2) a stated N must equal ask.Sent, and where a form states none (Kilo's
// range form) its lower bound must be 1, N being ask.Sent; (V3) a cap below N; (V4) a cap at
// least w.Minimum; (V5) a window above the input it states; (V6) a named
// model must be ask.Model, ask.Resolved, or share its canonical tokens with
// either, the provider prefix dropped, for the cap to be kept. A form w does
// not read is not read. It reports false for anything not read or rejected,
// which is returned as it came.
func ReadLimitRefusal(w LimitWire, err error, ask LimitAsk) (LimitReading, bool) {
	var apiErr *llmprovider.APIError
	if !errors.As(err, &apiErr) || !errors.Is(err, llmprovider.ErrInvalidRequest) {
		return LimitReading{}, false
	}
	msg := apiErr.Message
	capForm, capGroups := matchLimitForm(capLimitForms, msg)
	windowForm, windowGroups := matchLimitForm(windowLimitForms, msg)
	switch {
	case capForm != 0 && windowForm != 0:
		return LimitReading{}, false // V1
	case capForm != 0 && w.Forms&capForm != 0:
		return readCap(w, capForm, capGroups, ask)
	case windowForm != 0 && w.Forms&windowForm != 0:
		return readWindow(windowForm, windowGroups, ask)
	}
	return LimitReading{}, false
}

// matchLimitForm is the first of forms msg matches, with its groups.
func matchLimitForm(forms []limitForm, msg string) (LimitForms, []string) {
	for _, f := range forms {
		if m := f.pattern.FindStringSubmatch(msg); m != nil {
			return f.form, m[1:]
		}
	}
	return 0, nil
}

// numbers is groups' first n as numbers, and whether each is one.
func numbers(groups []string, n int) ([]int, bool) {
	out := make([]int, n)
	for i := range n {
		v, err := strconv.Atoi(groups[i])
		if err != nil {
			return nil, false
		}
		out[i] = v
	}
	return out, true
}

// readCap reads a cap form's N and M, and checks V2 to V4 and V6.
func readCap(w LimitWire, form LimitForms, groups []string, ask LimitAsk) (LimitReading, bool) {
	nums, ok := numbers(groups, 2)
	if !ok {
		return LimitReading{}, false
	}
	var n, m int
	named := ""
	switch form {
	case FormAnthropicCap:
		n, m = nums[0], nums[1]
		if len(groups) > 2 {
			named = strings.TrimRight(groups[2], ".")
		}
	case FormOpenAICap:
		m, n = nums[0], nums[1]
	case FormKiloRange:
		// No value sent is stated: the lower bound must be 1, in V2's place
		// (0089-MADR DL95, OQ6 (1)); M is inclusive.
		if nums[0] != 1 {
			return LimitReading{}, false
		}
		n, m = ask.Sent, nums[1]
	default: // FormGeminiRange: M is exclusive.
		n, m = nums[0], nums[1]-1
	}
	if n != ask.Sent || m >= n || m < max(w.Minimum, 1) {
		return LimitReading{}, false // V2, V3, V4
	}
	return LimitReading{Form: form, Cap: m, Keep: sameModel(named, ask)}, true
}

// readWindow reads a window form's window and the input it states or
// implies, and checks V2 and V5.
func readWindow(form LimitForms, groups []string, ask LimitAsk) (LimitReading, bool) {
	r := LimitReading{Form: form}
	if form == FormSplitWindow {
		nums, ok := numbers(groups, 3)
		if !ok || (ask.Sent > 0 && nums[2] != ask.Sent) {
			return LimitReading{}, false // V2
		}
		r.Window, r.Input = nums[0], nums[1]
	} else {
		nums, ok := numbers(groups, 2)
		if !ok {
			return LimitReading{}, false
		}
		r.Window, r.Input = nums[0], nums[1]-ask.Sent
	}
	if r.Input < 0 || r.Window <= r.Input {
		return LimitReading{}, false // V5
	}
	return r, true
}

// sameModel reports whether named, a refusal's model, is ask's (V6): none
// named; the request's model or the id it resolved to, in any case; or the
// same canonical tokens as either, the provider prefix dropped.
func sameModel(named string, ask LimitAsk) bool {
	if named == "" {
		return true
	}
	for _, id := range []string{ask.Model, ask.Resolved} {
		if id == "" {
			continue
		}
		if strings.EqualFold(named, id) || slices.Equal(canonicalTokens(named), canonicalTokens(id)) {
			return true
		}
	}
	return false
}

// quantToken is a quantisation or format token, dropped from a canonical id.
var quantToken = regexp.MustCompile(`^(?:i?q\d[a-z0-9_]*|awq|gptq|fp8|fp16|bf16|int4|int8|gguf|mlx|\d+bit)$`)

// canonicalTokens is id's canonical tokens, by catalog's canonicalTokens
// rule (0061-MADR, amendment "E2 and E4's menu", X1), which this package
// cannot import: lower case; up to the last "/" dropped; an Ollama tag's
// ":latest" dropped; split on "-", ".", ":" and spaces; a quantisation or
// format token dropped; a "_" within a token becoming "-".
func canonicalTokens(id string) []string {
	s := strings.ToLower(strings.TrimSpace(id))
	if i := strings.LastIndex(s, "/"); i >= 0 {
		s = s[i+1:]
	}
	if i := strings.LastIndex(s, ":"); i >= 0 && s[i+1:] == "latest" {
		s = s[:i]
	}
	var out []string
	for _, t := range strings.FieldsFunc(s, func(r rune) bool { return strings.ContainsRune("-.: ", r) }) {
		if !quantToken.MatchString(t) {
			out = append(out, strings.ReplaceAll(t, "_", "-"))
		}
	}
	return out
}
