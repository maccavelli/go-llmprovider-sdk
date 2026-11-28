package toolschema

import (
	"reflect"
	"testing"
)

// TestRead_Gemini_0095 (0095-MADR FU11): Gemini's Interactions refusal, as
// 0084-PLAN Phase 1 recorded it, is a form, and it names the tool in single
// quotes; the same text under another code, or a message without the
// marker, is none.
func TestRead_Gemini_0095(t *testing.T) {
	const recorded = "Invalid schema in function tool 'record_city' parameters: specifies required field 'city' which is not defined in properties."
	form := Read([]string{"invalid_request"}, recorded)
	if form == None {
		t.Fatalf("Read(invalid_request, %q) = None; want Gemini's form", recorded)
	}
	if got := References(form, recorded); !reflect.DeepEqual(got.Names, []string{"record_city"}) || got.Indexes != nil {
		t.Errorf("References = %+v; want the one name record_city", got)
	}
	if got := References(form, "Invalid schema in function tool 'cut short"); !reflect.DeepEqual(got.Names, []string{""}) {
		t.Errorf("a name with no closing quote: %+v; want one empty name, which names no tool", got)
	}
	for _, c := range []struct {
		name, code, message string
	}{
		{"another code", "invalid_argument", recorded},
		{"no marker", "invalid_request", "Invalid value for field 'record_city'."},
	} {
		if got := Read([]string{c.code}, c.message); got != None {
			t.Errorf("%s: Read = %v; want None", c.name, got)
		}
	}
	// The OpenAI form's marker, function '<name>', is not Gemini's: its
	// message holds "function tool '", so the OpenAI form reads no name there.
	if got := References(OpenAI, recorded); len(got.Names) != 0 {
		t.Errorf("References(OpenAI, Gemini's message) = %+v; want none", got)
	}
}
