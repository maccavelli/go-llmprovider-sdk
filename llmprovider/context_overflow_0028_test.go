package llmprovider

import (
	"strings"
	"testing"
)

// TestContextOverflow_PrefilterAdmitsEveryPattern (0028-MADR D-A10): each
// message form's literal, which the form cannot match without, is in the
// form's own sample, and the sample still matches.
func TestContextOverflow_PrefilterAdmitsEveryPattern(t *testing.T) {
	for _, form := range contextOverflowMessages {
		sample := overflowSamples[form.seen]
		if form.anchor == "" || !strings.Contains(strings.ToLower(sample), form.anchor) {
			t.Errorf("%s: literal %q is not in the sample %q", form.seen, form.anchor, sample)
		}
		if !form.pattern.MatchString(sample) {
			t.Errorf("%s: the pattern no longer matches its sample %q", form.seen, sample)
		}
		if !contextOverflow(apiErrorEnvelope{msg: sample}) {
			t.Errorf("%s: contextOverflow(%q) = false", form.seen, sample)
		}
	}
}

// TestContextOverflow_MatchesReference: contextOverflow decides as it did
// before the prefilter, on every sample, on rate-limit and ordinary
// messages, and on each type, for messages within the 2 KiB it reads.
func TestContextOverflow_MatchesReference(t *testing.T) {
	envs := []apiErrorEnvelope{
		{msg: "Rate limit reached: too many tokens per minute"},
		{msg: "Too many requests"},
		{msg: "The model is overloaded"},
		{msg: ""},
		{msg: "invalid x-api-key"},
		{msg: strings.Repeat("detail ", 280) + "prompt is too long"},
		{types: []string{"context_length_exceeded"}},
		{types: []string{"request_too_large"}, msg: "Request exceeds the maximum size"},
		{types: []string{"model_context_window_exceeded"}},
	}
	for _, sample := range overflowSamples {
		envs = append(envs, apiErrorEnvelope{msg: sample}, apiErrorEnvelope{msg: strings.ToUpper(sample)})
	}
	for _, env := range envs {
		if got, want := contextOverflow(env), contextOverflowReference(env); got != want {
			t.Errorf("contextOverflow(%+v) = %t; the reference says %t", env, got, want)
		}
	}
}
