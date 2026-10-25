package wire

import (
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// learnedLimit bounds the memo of thinking refusals, as 0034-MADR D3 bounds
// the forced-choice memo (0062-MADR D5).
const learnedLimit = 256

// Learned holds what models refused of the thinking settings a provider sent,
// by provider, base URL and model, oldest model first, at most learnedLimit
// models (0062-MADR D5). It is never written to disk, and a restart clears
// it. A catalog.Cache holds one; a nil *Learned is the process's. It is safe
// for concurrent use.
type Learned struct {
	mu     sync.Mutex
	order  []string
	models map[string]*learnedModel
}

// learnedModel is what one model refused, and the efforts a refusal listed.
type learnedModel struct {
	refused  []llmprovider.Effort
	accepted []llmprovider.Effort
	listed   bool
}

// NewLearned returns an empty memo.
func NewLearned() *Learned { return &Learned{models: map[string]*learnedModel{}} }

// processLearned is the process's memo: every provider's without
// catalog.WithCache.
var processLearned = NewLearned()

// or is l, or the process's memo when l is nil.
func (l *Learned) or() *Learned {
	if l == nil {
		return processLearned
	}
	return l
}

// LearnedKey is the memo's key for a model on a provider at a base URL. The
// base URL is in it because one provider id at two base URLs, two Ollama
// hosts or OpenCode's two gateways, may serve two models under one id
// (0062-MADR D5).
func LearnedKey(provider, baseURL, model string) string {
	return provider + "\x00" + strings.TrimRight(baseURL, "/") + "\x00" + strings.ToLower(strings.TrimSpace(model))
}

// efforts is what l learned of key's model: the efforts it refused, and the
// efforts a refusal listed, with listed reporting that one did.
func (l *Learned) efforts(key string) (refused, accepted []llmprovider.Effort, listed bool) {
	l = l.or()
	l.mu.Lock()
	defer l.mu.Unlock()
	m := l.models[key]
	if m == nil {
		return nil, nil, false
	}
	return slices.Clone(m.refused), slices.Clone(m.accepted), m.listed
}

// learnEffort remembers that key's model refused effort, and the efforts the
// refusal listed when listed, dropping the oldest model past learnedLimit. It
// reports whether anything was new.
func (l *Learned) learnEffort(key string, effort llmprovider.Effort, accepted []llmprovider.Effort, listed bool) bool {
	l = l.or()
	l.mu.Lock()
	defer l.mu.Unlock()
	m := l.models[key]
	if m == nil {
		if len(l.order) >= learnedLimit {
			delete(l.models, l.order[0])
			l.order = l.order[1:]
		}
		m = &learnedModel{}
		l.models[key] = m
		l.order = append(l.order, key)
	}
	changed := false
	if !slices.Contains(m.refused, effort) {
		m.refused = append(m.refused, effort)
		changed = true
	}
	if listed && (!m.listed || !slices.Equal(m.accepted, accepted)) {
		m.accepted, m.listed = slices.Clone(accepted), true
		changed = true
	}
	return changed
}

// Refusal is what a wire's matcher reads from a refused thinking setting
// (0062-MADR D4).
type Refusal struct {
	// Effort is the refused effort: the one the request sent.
	Effort llmprovider.Effort
	// Field reports that the model takes no effort at all: the field itself
	// was refused, so no effort is sent (D3's last rung).
	Field bool
	// Shape is a refused thinking shape, for the shape's ladder (D3).
	Shape llmprovider.ThinkingShape
	// Accepted are the efforts the refusal lists, Effort values only, and
	// Listed reports that it lists any.
	Accepted []llmprovider.Effort
	Listed   bool
}

// Matcher reads a refusal of a thinking setting from err, an effort's when
// the request sent effort sent ("" for none), each wire's from the refusals
// 0062-PLAN Phase 1 recorded for it. It reports false for anything else,
// which is returned as it came.
type Matcher func(err error, sent llmprovider.Effort) (Refusal, bool)

// effortWords are the words a service's list of efforts holds: the Effort
// values, and the services' none and minimal (0062-MADR, amendment "Phase
// 1's evidence").
var effortWords = []string{"none", "minimal", "low", "medium", "high", "xhigh", "max"}

// quotedWord is a word in single or double quotes.
var quotedWord = regexp.MustCompile(`['"]([A-Za-z]+)['"]`)

// QuotedEfforts reads a list of quoted effort words, such as "'low',
// 'medium' or 'max'": the Effort values among them, and whether there is at
// least one word and every word is an effort word, so that a list of other
// values is not taken for one.
func QuotedEfforts(list string) ([]llmprovider.Effort, bool) {
	var words []string
	for _, m := range quotedWord.FindAllStringSubmatch(list, -1) {
		words = append(words, m[1])
	}
	return effortsOf(words)
}

// PlainEfforts reads a list of bare effort words separated by commas, such as
// "low, high.": as QuotedEfforts.
func PlainEfforts(list string) ([]llmprovider.Effort, bool) {
	var words []string
	for word := range strings.SplitSeq(strings.TrimSuffix(strings.TrimSpace(list), "."), ",") {
		words = append(words, strings.TrimSpace(word))
	}
	return effortsOf(words)
}

// effortsOf is the Effort values among words, and whether there is one word
// at least and every word is an effort word.
func effortsOf(words []string) ([]llmprovider.Effort, bool) {
	if len(words) == 0 {
		return nil, false
	}
	var out []llmprovider.Effort
	for _, w := range words {
		w = strings.ToLower(w)
		if !slices.Contains(effortWords, w) {
			return nil, false
		}
		if e := llmprovider.Effort(w); slices.Contains(effortOrder, e) && !slices.Contains(out, e) {
			out = append(out, e)
		}
	}
	return out, true
}

// Learner is one request's effort (0062-MADR D2, D3, D5, D6). Its Effort
// chooses the effort to send from what the model refused, by the memo and by
// the caller, then the facts, then the request; its Check, a Retry, learns a
// refusal its wire's matcher reads and has the request built again. A
// Learner serves one request, built and checked in turn, and is not safe for
// concurrent use.
type Learner struct {
	memo   *Learned
	key    string
	model  string
	passed []llmprovider.Effort
	match  Matcher
	// sent is the effort the last build sent, "" for none.
	sent llmprovider.Effort
	// learned are the WarningReasoningLearned of this request.
	learned []llmprovider.Warning
}

// NewLearner is the Learner of one request to model, on provider at
// baseURL, whose refusals memo holds and whose wire's refusals match reads (a
// nil match learns nothing). passed are the caller's refusals, from
// WithReasoningRefusals: those of model are never sent.
func NewLearner(memo *Learned, provider, baseURL, model string, passed []llmprovider.ReasoningRefusal,
	match Matcher) *Learner {
	l := &Learner{memo: memo, key: LearnedKey(provider, baseURL, model), model: model, match: match}
	for _, r := range passed {
		if r.Effort != "" && strings.EqualFold(strings.TrimSpace(r.Model), strings.TrimSpace(model)) {
			l.passed = append(l.passed, r.Effort)
		}
	}
	return l
}

// Effort is the effort to send for want, "" for none, by Ladder over what the
// model refused and the efforts a refusal listed, else the facts' (listed,
// known when they state them). Its warnings are the response's: each refusal
// this request learned, and a WarningEffortDegraded when the effort sent is
// not want. No want sends none, unwarned.
func (l *Learner) Effort(want llmprovider.Effort, listed []llmprovider.Effort, known bool) (llmprovider.Effort,
	[]llmprovider.Warning) {
	warnings := slices.Clone(l.learned)
	l.sent = ""
	if want == "" {
		return "", warnings
	}
	refused, accepted, learnedList := l.memo.efforts(l.key)
	if learnedList {
		listed, known = accepted, true
	}
	if rungs := Ladder(want, listed, known, slices.Concat(refused, l.passed)); len(rungs) > 0 {
		l.sent = rungs[0]
	}
	if l.sent != want {
		warnings = append(warnings, DegradedWarning(l.model, want, l.sent))
	}
	return l.sent, warnings
}

// Check is a Retry: a refusal of the effort the last build sent, read by the
// matcher, is learned and the request built again, with a
// WarningReasoningLearned. Anything else, a shape's refusal included, which
// 0062-PLAN's Phase 3 learns, is returned as it came.
func (l *Learner) Check(err error) (bool, error) {
	if l.match == nil || l.sent == "" {
		return false, err
	}
	r, ok := l.match(err, l.sent)
	if !ok || r.Shape != "" || (!r.Field && r.Effort != l.sent) {
		return false, err
	}
	accepted, listed := r.Accepted, r.Listed
	if r.Field {
		// The model takes no effort: none is listed.
		accepted, listed = nil, true
	}
	if !l.memo.learnEffort(l.key, l.sent, accepted, listed) {
		return false, err
	}
	refused := llmprovider.ReasoningRefusal{Model: l.model, Effort: l.sent}
	l.learned = append(l.learned, llmprovider.Warning{Kind: llmprovider.WarningReasoningLearned, Model: l.model,
		Detail: l.model + " refused reasoning effort " + string(l.sent) + "; it is remembered for the rest of the process, " +
			"and WithReasoningRefusals can say so on the next start",
		Refused: refused})
	return true, nil
}

// Retries is checks as one Retry, in order: the first that resends, or
// replaces the error, decides; a nil check is skipped.
func Retries(checks ...Retry) Retry {
	return func(err error) (bool, error) {
		for _, check := range checks {
			if check == nil {
				continue
			}
			resend, replaced := check(err)
			if resend {
				return true, nil
			}
			if replaced != err { //nolint:errorlint // the same error, not one it wraps: the check left it alone
				return false, replaced
			}
		}
		return false, err
	}
}
