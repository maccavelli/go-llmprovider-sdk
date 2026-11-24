package wire

import (
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// The refusal texts below are the forum copies 0089-MADR W7 names, and the
// window forms of 0088-MADR C39; none is a recorded reply.

// limitRefusal is a 400 invalid_request_error from provider carrying msg.
func limitRefusal(provider, msg string) error {
	return &llmprovider.APIError{Provider: provider, Status: http.StatusBadRequest, Code: "invalid_request_error", Message: msg}
}

// anthropicCapText is Anthropic's cap form refusing n with cap m for model.
func anthropicCapText(n, m, model string) string {
	return "max_tokens: " + n + " > " + m + ", which is the maximum allowed number of output tokens for " + model
}

// openRouterWindowText is OpenRouter's window form as Kilo passes it: a
// window of 200,000 and a request of about 150,000 tokens.
const openRouterWindowText = "This endpoint's maximum context length is 200000 tokens. However, you requested about 150000 tokens."

// TestLimitRefusal_Validator_0089 (DL31 to DL34, DL67): the validator reads
// a cap form's stated maximum only when V1 to V4 hold, and compares N with
// the value it is given, the wire's after AddThinkingFor.
func TestLimitRefusal_Validator_0089(t *testing.T) {
	ask := LimitAsk{Sent: 128000, Model: "claude-opus-4-5-20251101"}
	// (1) Claude's cap form, N the wire's value: a bound of 64,000.
	r, ok := ReadLimitRefusal(MessagesLimits, limitRefusal("claude", anthropicCapText("128000", "64000", "claude-opus-4-5-20251101")), ask)
	if !ok || r.Form != FormAnthropicCap || r.Cap != 64000 || r.Window != 0 || !r.Keep {
		t.Errorf("case 1: %+v, %v; want a kept cap of 64000", r, ok)
	}
	// (2) The cap form and OpenRouter's window form in one text: rejected,
	// read neither as a cap nor as a window, on a route that reads both;
	// the window form alone is read there, as a window.
	both := anthropicCapText("128000", "64000", "claude-opus-4-5-20251101") + ". " + openRouterWindowText
	if r, ok := ReadLimitRefusal(KiloLimits, limitRefusal("kilo", both), ask); ok {
		t.Errorf("case 2: %+v; want a text matching a cap form and a window form rejected (V1)", r)
	}
	if r, ok := ReadLimitRefusal(KiloLimits, limitRefusal("kilo", openRouterWindowText), ask); !ok ||
		r.Form != FormOpenRouterWindow || r.Cap != 0 || r.Window != 200000 || r.Input != 150000-128000 {
		t.Errorf("case 2, the window form alone: %+v, %v; want window 200000, input 22000, no cap", r, ok)
	}
	// (3) N other than the wire's value: rejected.
	if r, ok := ReadLimitRefusal(MessagesLimits, limitRefusal("claude", anthropicCapText("100000", "64000", "claude-opus-4-5-20251101")), ask); ok {
		t.Errorf("case 3: %+v; want N 100000 against a wire value of 128000 rejected (V2)", r)
	}
	// (4) M at N: rejected.
	if r, ok := ReadLimitRefusal(MessagesLimits, limitRefusal("claude", anthropicCapText("64000", "64000", "claude-opus-4-5-20251101")),
		LimitAsk{Sent: 64000, Model: ask.Model}); ok {
		t.Errorf("case 4: %+v; want M at N rejected (V3)", r)
	}
	// (5) Kilo, OpenAI's form stating at most 8, on a test wire whose field
	// takes at least 16: rejected below the minimum.
	sixteen := LimitWire{Forms: KiloLimits.Forms, Minimum: 16}
	openAI := limitRefusal("kilo", "This model supports at most 8 completion tokens, whereas you provided 100.")
	if r, ok := ReadLimitRefusal(sixteen, openAI, LimitAsk{Sent: 100, Model: "openai/gpt-4o"}); ok {
		t.Errorf("case 5: %+v; want a cap of 8 below the minimum 16 rejected (V4)", r)
	}
	if r, ok := ReadLimitRefusal(KiloLimits, openAI, LimitAsk{Sent: 100, Model: "openai/gpt-4o"}); !ok || r.Cap != 8 || !r.Keep {
		t.Errorf("case 5, Kilo's own minimum: %+v, %v; want a kept cap of 8", r, ok)
	}
	// (6) OpenCode google, Gemini's range: its exclusive bound less one.
	gemini := limitRefusal("opencode-zen/google",
		"Unable to submit request because it has a maxOutputTokens value of 100000 but the supported range is from 1 (inclusive) to 65537 (exclusive). Update the value and try again.")
	if r, ok := ReadLimitRefusal(GenerateContentLimits, gemini, LimitAsk{Sent: 100000, Model: "gemini-3-pro"}); !ok ||
		r.Form != FormGeminiRange || r.Cap != 65536 {
		t.Errorf("case 6: %+v, %v; want a cap of 65536", r, ok)
	}
	// (7) The budget shape raised the wire's value from a chosen 32,000 to
	// 44,096: a refusal stating 44096 is read; one stating the pre-raise
	// value is rejected.
	raised := LimitAsk{Sent: 44096, Model: "claude-opus-4-5-20251101"}
	if r, ok := ReadLimitRefusal(MessagesLimits, limitRefusal("claude", anthropicCapText("44096", "32000", "claude-opus-4-5-20251101")), raised); !ok || r.Cap != 32000 {
		t.Errorf("case 7: %+v, %v; want 44096 > 32000 read against the raised value", r, ok)
	}
	if r, ok := ReadLimitRefusal(MessagesLimits, limitRefusal("claude", anthropicCapText("32000", "30000", "claude-opus-4-5-20251101")), raised); ok {
		t.Errorf("case 7, the pre-raise value: %+v; want rejected (V2, DL67)", r)
	}
	// Not an invalid request: not read.
	unavailable := &llmprovider.APIError{Provider: "claude", Status: http.StatusServiceUnavailable,
		Message: anthropicCapText("128000", "64000", "claude-opus-4-5-20251101")}
	if r, ok := ReadLimitRefusal(MessagesLimits, unavailable, ask); ok {
		t.Errorf("a 503: %+v; want not read", r)
	}
}

// TestLimitRefusal_FormsByWire_0089 (DL68): each route reads its own cap
// forms only: OpenAI's text not on Responses, but on Kilo; Gemini's range
// not on Interactions, but on generateContent.
func TestLimitRefusal_FormsByWire_0089(t *testing.T) {
	openAI := "This model supports at most 4096 completion tokens, whereas you provided 8192."
	ask := LimitAsk{Sent: 8192, Model: "gpt-4o"}
	if r, ok := ReadLimitRefusal(ResponsesLimits, limitRefusal("openai", openAI), ask); ok {
		t.Errorf("OpenAI's form on Responses: %+v; want not read", r)
	}
	if r, ok := ReadLimitRefusal(KiloLimits, limitRefusal("kilo", openAI), ask); !ok || r.Cap != 4096 {
		t.Errorf("OpenAI's form on Kilo: %+v, %v; want 4096", r, ok)
	}
	if r, ok := ReadLimitRefusal(ChatCompletionsLimits, limitRefusal("together", openAI), ask); !ok || r.Cap != 4096 {
		t.Errorf("OpenAI's form on Chat Completions: %+v, %v; want 4096", r, ok)
	}
	gemini := "it has a maxOutputTokens value of 100000 but the supported range is from 1 (inclusive) to 65537 (exclusive)."
	gask := LimitAsk{Sent: 100000, Model: "gemini-3-pro"}
	if r, ok := ReadLimitRefusal(InteractionsLimits, limitRefusal("gemini", gemini), gask); ok {
		t.Errorf("Gemini's range on Interactions: %+v; want not read", r)
	}
	if r, ok := ReadLimitRefusal(GenerateContentLimits, limitRefusal("opencode-zen/google", gemini), gask); !ok || r.Cap != 65536 {
		t.Errorf("Gemini's range on generateContent: %+v, %v; want 65536", r, ok)
	}
	anthropic := anthropicCapText("128000", "64000", "claude-opus-4-5")
	cask := LimitAsk{Sent: 128000, Model: "claude-opus-4-5"}
	for name, w := range map[string]LimitWire{"Responses": ResponsesLimits, "Chat Completions": ChatCompletionsLimits,
		"generateContent": GenerateContentLimits, "Interactions": InteractionsLimits} {
		if r, ok := ReadLimitRefusal(w, limitRefusal("x", anthropic), cask); ok {
			t.Errorf("Anthropic's form on %s: %+v; want not read", name, r)
		}
	}
	if r, ok := ReadLimitRefusal(KiloLimits, limitRefusal("kilo", anthropic), cask); !ok || r.Cap != 64000 {
		t.Errorf("Anthropic's form on Kilo: %+v, %v; want 64000", r, ok)
	}
}

// TestLimitRefusal_NamedModel_0089 (DL36, V6): a cap naming the request's
// resolved id, or its canonical tokens with the provider prefix dropped, is
// kept; a cap naming another model serves this request only.
func TestLimitRefusal_NamedModel_0089(t *testing.T) {
	for _, c := range []struct {
		name string
		w    LimitWire
		ask  LimitAsk
		text string
		keep bool
	}{
		{"Claude's alias, refused naming its resolved id", MessagesLimits,
			LimitAsk{Sent: 128000, Model: "claude-sonnet-4-5", Resolved: "claude-sonnet-4-5-20250929"},
			anthropicCapText("128000", "64000", "claude-sonnet-4-5-20250929"), true},
		{"Kilo's routed id, refused naming the upstream's canonical id", KiloLimits,
			LimitAsk{Sent: 128000, Model: "anthropic/claude-sonnet-4.5"},
			anthropicCapText("128000", "64000", "claude-sonnet-4-5"), true},
		{"Kilo's auto route, refused naming a Claude id", KiloLimits,
			LimitAsk{Sent: 128000, Model: "kilo-auto/balanced"},
			anthropicCapText("128000", "64000", "claude-sonnet-4-5-20250929"), false},
	} {
		r, ok := ReadLimitRefusal(c.w, limitRefusal("x", c.text), c.ask)
		if !ok || r.Cap != 64000 {
			t.Errorf("%s: %+v, %v; want a cap of 64000 for this request", c.name, r, ok)
			continue
		}
		if r.Keep != c.keep {
			t.Errorf("%s: Keep = %v; want %v", c.name, r.Keep, c.keep)
		}
	}
}
