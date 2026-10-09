package responses

import "testing"

// TestIgnored_EventTypes holds ignored to its contract: only a "response."
// event that ReadStream does not act on is skipped. A wrong "skip" loses an
// event, and a wrong "decode" passes every stream test, only slower, so both
// directions are asserted here (0019-PLAN Phase 2).
func TestIgnored_EventTypes(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		want    bool
	}{
		{"a delta", `{"type":"response.output_text.delta","delta":"word"}`, true},
		{"in progress", `{"type":"response.in_progress","response":{}}`, true},
		{"a reasoning delta", `{"type":"response.reasoning_summary_text.delta","delta":"x"}`, true},
		{"not a response event", `{"type":"error","message":"x"}`, false},
		{"no type", `{"delta":"word"}`, false},
		{"a type not closed", `{"type":"response.output_text.delta`, false},
		{"empty", ``, false},
		{"the first type is an item's", `{"item":{"type":"message"},"type":"response.output_text.delta"}`, false},
	}
	for event := range handledEvents {
		cases = append(cases, struct {
			name    string
			payload string
			want    bool
		}{"handled " + event, `{"type":"` + event + `","sequence_number":1}`, false})
	}
	for _, c := range cases {
		if got := ignored([]byte(c.payload)); got != c.want {
			t.Errorf("%s: ignored(%s) = %v, want %v", c.name, c.payload, got, c.want)
		}
	}
}
