package wire

import (
	"errors"
	"fmt"
	"iter"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// The rule a request's output limit follows (0089-MADR D6), and its resend
// at a validated stated cap (D4, D5), in plain values: the providers read
// the facts (catalog.Limits) and the memo, and this decides.

// LimitInput is what one request's output limit is chosen from.
type LimitInput struct {
	// Request is Request.MaxOutputTokens; 0 for none.
	Request int
	// Option is WithMaxTokens' value, or a listing probe's, and Given
	// whether it was given (Settings.MaxTokensGiven, or a probe); a value
	// not given is the default, which decides nothing.
	Option int
	Given  bool
	// Sends reports that the route sends a limit the SDK chooses: the
	// Messages wire, and Kilo until 0089-MADR DL65's verdict (OQ1 (1)).
	Sends bool
	// Required reports that the wire requires a limit (Messages): with no
	// fact, the fallback is sent, and with none the request is refused.
	Required bool
	// Output is the chosen output fact, held to the window and to a kept
	// cap (catalog.Limits); 0 when unknown. Source is its evidence's name.
	Output int
	Source string
	// Fallback is the smallest limit any current model on the route is
	// stated to take, 0 for none, and FallbackSource its evidence's name
	// (0089-MADR DL51).
	Fallback       int
	FallbackSource string
	// Window is the model's stated window, else its stated input limit where
	// only that is stated (0088-MADR ST87; 0089-MADR DL56), 0 for neither;
	// Estimate is the request's estimated input (EstimateInput); Minimum is
	// the wire's least limit, at least 1.
	Window, Estimate, Minimum int
	// Provider and Model name the request, for the refusal's message.
	Provider, Model string
}

// ChooseLimit is the output limit a request carries (0089-MADR D6): the
// request's own value as given; a given option cut to the stated output and
// never raised; on a route that sends one, the output fact, else where the
// wire requires one the fallback; otherwise none. The fact and the fallback,
// the values the SDK chose, are held to the room a stated window leaves the
// estimated input (0088-MADR ST87, ST105); a caller's value is not. On a
// wire that requires a limit with no fact and no fallback, it returns an
// error matching llmprovider.ErrInvalidRequest that says how to state the
// limit, and no request is sent (DL51).
func ChooseLimit(in LimitInput) (llmprovider.SentLimit, error) {
	switch {
	case in.Request > 0:
		return llmprovider.SentLimit{Tokens: in.Request, Sent: true, Basis: llmprovider.LimitBasisRequest}, nil
	case in.Given && in.Option > 0:
		if in.Output > 0 && in.Output < in.Option {
			return llmprovider.SentLimit{Tokens: in.Output, Sent: true, Basis: llmprovider.LimitBasisOption, Source: in.Source}, nil
		}
		return llmprovider.SentLimit{Tokens: in.Option, Sent: true, Basis: llmprovider.LimitBasisOption}, nil
	case !in.Sends:
		return llmprovider.SentLimit{Basis: llmprovider.LimitBasisNone}, nil
	case in.Output > 0:
		return llmprovider.SentLimit{Tokens: fit(in.Output, in), Sent: true, Basis: llmprovider.LimitBasisFact,
			Source: in.Source}, nil
	case !in.Required:
		return llmprovider.SentLimit{Basis: llmprovider.LimitBasisNone}, nil
	case in.Fallback > 0:
		return llmprovider.SentLimit{Tokens: fit(in.Fallback, in), Sent: true, Basis: llmprovider.LimitBasisFallback,
			Source: in.FallbackSource}, nil
	}
	return llmprovider.SentLimit{}, fmt.Errorf("%w: %s: no source states an output limit for model %q, and the wire requires "+
		"one: state the model's limit, for example as maxTokens in models.json, or in a layer given with "+
		"catalog.WithModelLayer, or pass WithMaxTokens or Request.MaxOutputTokens", llmprovider.ErrInvalidRequest,
		in.Provider, in.Model)
}

// formNames are the refusal forms' names, a SentLimit's Source on a resend.
var formNames = map[LimitForms]string{
	FormAnthropicCap:     "refusal: anthropic max_tokens",
	FormOpenAICap:        "refusal: openai at most",
	FormGeminiRange:      "refusal: gemini range",
	FormKiloRange:        "refusal: kilo max_tokens range",
	FormOpenRouterWindow: "refusal: openrouter context length",
	FormKiloWindow:       "refusal: kilo context window",
	FormSplitWindow:      "refusal: context split",
}

// String is f's name, as a SentLimit's Source names it.
func (f LimitForms) String() string {
	if name, ok := formNames[f]; ok {
		return name
	}
	return fmt.Sprintf("LimitForms(%d)", uint8(f))
}

// errUncounted marks a resend that does not count toward MaxSends: the one
// resend at a validated stated limit, which 0089-MADR D6 puts outside
// 0062's bound.
var errUncounted = errors.New("wire: a resend outside the bound")

// uncounted reports whether a retry's resend was the limit's.
func uncounted(replaced error) bool { return errors.Is(replaced, errUncounted) }

// LimitRetry is one request's output limit across its sends (0089-MADR D4
// to D6): it chooses the limit for each build, holds the resend's value once
// a refusal is validated, keeps the stated cap through Keep, and reports the
// warning. A request uses it from one goroutine; it is not safe for
// concurrent use, and a provider makes one per request.
type LimitRetry struct {
	// Wire is what the route reads of a refusal.
	Wire LimitWire
	// Model is the request's model, and Resolved the id a one-model read
	// resolved it to, or empty (V6).
	Model, Resolved string
	// Keep keeps a validated, kept cap, and reports whether that changed the
	// memo; nil keeps nothing. Learning off (WithoutLimitLearning) passes nil.
	Keep func(tokens int) bool
	// Observed is the warning's observation, its Tokens and Seen set when a
	// cap is kept.
	Observed llmprovider.LimitObservation

	sent    llmprovider.SentLimit // the last build's
	resend  llmprovider.SentLimit // the resend's, once a refusal set it
	resent  bool
	warning *llmprovider.Warning
	// kept is the observation of a cap this request kept, whether or not it
	// changed the memo: the error that ends the call carries it (0089-MADR
	// DL78, DL82).
	kept llmprovider.LimitObservation
}

// Choose is the limit the next send carries: the resend's, once a refusal
// set it (sticky), else ChooseLimit's, held no lower than r's wire's minimum.
func (r *LimitRetry) Choose(in LimitInput) (llmprovider.SentLimit, error) {
	if r.resent {
		r.sent = r.resend
		return r.sent, nil
	}
	in.Minimum = max(in.Minimum, r.Wire.Minimum)
	l, err := ChooseLimit(in)
	r.sent = l
	return l, err
}

// Optional is the limit the next send carries on a wire where the output
// limit is optional (0089-MADR D6, DL47, DL48): the request's own value as
// given; a given option cut to the stated output, which stated reads and is
// called only then; otherwise none, so the service's own default applies.
// stated reports 0 when no source states the output.
func (r *LimitRetry) Optional(request, option int, given bool, stated func() (int, string)) llmprovider.SentLimit {
	in := LimitInput{Request: request, Option: option, Given: given}
	if request <= 0 && given && option > 0 && stated != nil {
		in.Output, in.Source = stated()
	}
	l, err := r.Choose(in)
	if err != nil {
		// Unreached: Required is false, so ChooseLimit refuses nothing.
		return llmprovider.SentLimit{Basis: llmprovider.LimitBasisNone}
	}
	return l
}

// DropLimit deletes a body's output-limit field when l was not sent
// (0089-MADR D6, DL47): path names the field, through the nested maps that
// hold it.
func DropLimit(body map[string]any, l llmprovider.SentLimit, path ...string) {
	if l.Sent || len(path) == 0 {
		return
	}
	for _, key := range path[:len(path)-1] {
		inner, ok := body[key].(map[string]any)
		if !ok {
			return
		}
		body = inner
	}
	delete(body, path[len(path)-1])
}

// Sent records the value the built body carries, after a thinking budget
// raised it (DL55, DL67): what the report names and V2 compares.
func (r *LimitRetry) Sent(tokens int) {
	if r.sent.Sent {
		r.sent.Tokens = tokens
	}
}

// Report is the last send's limit.
func (r *LimitRetry) Report() llmprovider.SentLimit { return r.sent }

// Check is a Retry: a refusal of the limit sent, validated by
// ReadLimitRefusal, keeps a kept cap (Keep), and resends once at the stated
// cap, or the room a window form leaves, when the SDK chose the value
// (basis fact or fallback); a caller's value is not resent (0088-MADR ST17,
// ST18; 0089-MADR DL37). A kept cap is reported on the error that ends the
// call, every time (Finish; DL78 to DL82), and in a WarningLimitLearned on a
// response only when it changed the memo, once per key (D5). Anything else
// is left to the next check.
func (r *LimitRetry) Check(err error) (bool, error) {
	if r.resent || !r.sent.Sent {
		return false, err
	}
	reading, ok := ReadLimitRefusal(r.Wire, err, LimitAsk{Sent: r.sent.Tokens, Model: r.Model, Resolved: r.Resolved})
	if !ok {
		return false, err
	}
	if reading.Cap > 0 && reading.Keep && r.Keep != nil {
		obs := r.Observed
		obs.Tokens, obs.Seen = reading.Cap, time.Now()
		r.kept = obs
		if r.Keep(reading.Cap) {
			r.warning = &llmprovider.Warning{Kind: llmprovider.WarningLimitLearned, Model: r.Model,
				Detail: fmt.Sprintf("the service refused max output %d and stated %d for %s; kept for this process",
					r.sent.Tokens, reading.Cap, r.Model),
				Observed: obs}
		}
	}
	if r.sent.Basis != llmprovider.LimitBasisFact && r.sent.Basis != llmprovider.LimitBasisFallback {
		return false, err
	}
	tokens := reading.Cap
	if tokens == 0 {
		tokens = reading.Window - reading.Input
	}
	if tokens < max(r.Wire.Minimum, 1) {
		return false, err
	}
	r.resend = llmprovider.SentLimit{Tokens: tokens, Sent: true, Basis: llmprovider.LimitBasisRefusal,
		Source: reading.Form.String()}
	r.resent = true
	return true, errUncounted
}

// Warning is the WarningLimitLearned a kept cap earns, once per key: only
// when Keep reported a change.
func (r *LimitRetry) Warning() (llmprovider.Warning, bool) {
	if r.warning == nil {
		return llmprovider.Warning{}, false
	}
	return *r.warning, true
}

// Finish ends a call: given its error, it returns that error, with the cap
// the request kept, if any, set as the Observed of the *APIError it wraps
// (errors.As), whose type and text stay; an error that is no *APIError is
// returned as it came, and the cap stays in the memo (0089-MADR DL78, DL80,
// DL86, DL87). Given none, it reports the limit on resp (D7), adds the
// warning, and hands its output tokens to observe, the memo's drop of a cap
// the response shows wrong (D5, DL39); a nil resp or observe is skipped.
func (r *LimitRetry) Finish(resp *llmprovider.Response, err error, observe func(output int)) error {
	if err != nil {
		r.markObserved(err)
		return err
	}
	r.finishResponse(resp, observe)
	return nil
}

// finishResponse is Finish on a response.
func (r *LimitRetry) finishResponse(resp *llmprovider.Response, observe func(output int)) {
	if resp == nil {
		return
	}
	resp.OutputLimit = r.Report()
	if w, ok := r.Warning(); ok {
		resp.Warnings = append(resp.Warnings, w)
	}
	if observe != nil && resp.Usage.OutputTokens > 0 {
		observe(resp.Usage.OutputTokens)
	}
}

// markObserved sets the kept cap on the *APIError err wraps, if any (DL78,
// DL80).
func (r *LimitRetry) markObserved(err error) {
	if r.kept.Tokens == 0 {
		return
	}
	if apiErr, ok := errors.AsType[*llmprovider.APIError](err); ok {
		apiErr.Observed = r.kept
	}
}

// FinishStream is seq with Finish applied to its done event's Response, the
// stream's final one (DL54), and to an error it yields (DL79).
func (r *LimitRetry) FinishStream(seq iter.Seq2[llmprovider.Event, error],
	observe func(output int)) iter.Seq2[llmprovider.Event, error] {
	return func(yield func(llmprovider.Event, error) bool) {
		for ev, err := range seq {
			switch {
			case err != nil:
				err = r.Finish(nil, err, observe)
			case ev.Type == llmprovider.EventDone:
				r.finishResponse(ev.Response, observe)
			}
			if !yield(ev, err) {
				return
			}
		}
	}
}
