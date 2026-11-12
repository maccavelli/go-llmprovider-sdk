package authcontract_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
)

const (
	plantedPaste = "sk-PLANTED-0073"
	plantedField = "fv-PLANTED-0073"
	fieldKey     = "api_token"
)

// output is one rendering of an answer, and the texts it must show.
type output struct {
	how  string
	text string
	show []string
}

// renderings formats a and &a with each fmt verb, and logs each through
// slog's text and JSON handlers. secret is how each rendering prints Secret.
func renderings(a authcontract.Answer, secret map[string]string) []output {
	var out []output
	for _, v := range []any{a, &a} {
		for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
			out = append(out, output{
				how:  fmt.Sprintf("%s of a %T", verb, v),
				text: fmt.Sprintf(verb, v),
				show: []string{"Seq:7", "Confirm:true", "Secret:" + secret["fmt"], fieldKey},
			})
		}
		var text, js bytes.Buffer
		slog.New(slog.NewTextHandler(&text, nil)).Info("answer", "a", v)
		slog.New(slog.NewJSONHandler(&js, nil)).Info("answer", "a", v)
		out = append(out,
			output{how: fmt.Sprintf("slog's text handler with a %T", v), text: text.String(),
				show: []string{"a.seq=7", "a.confirm=true", "a.secret=" + secret["text"], fieldKey}},
			output{how: fmt.Sprintf("slog's JSON handler with a %T", v), text: js.String(),
				show: []string{`"seq":7`, `"confirm":true`, `"secret":` + secret["json"], fieldKey}})
	}
	return out
}

func planted(secret *bool) authcontract.Answer {
	return authcontract.Answer{
		SchemaVersion: authcontract.SchemaVersion, Seq: 7, Secret: secret, Paste: plantedPaste,
		Confirm: ptr(true), Fields: map[string]string{fieldKey: plantedField, "region": plantedField},
	}
}

// TestAnswerRedacts_0073: with Secret true and with Secret nil, no
// rendering of an Answer or a *Answer shows the paste or a field value, and
// each shows Seq, Confirm, Secret and the field keys (MADR D2, Q11).
func TestAnswerRedacts_0073(t *testing.T) {
	cases := []struct {
		name   string
		secret *bool
		shows  map[string]string
	}{
		{"Secret true", ptr(true), map[string]string{"fmt": "true", "text": "true", "json": "true"}},
		{"Secret nil", nil, map[string]string{"fmt": "<nil>", "text": "<nil>", "json": "null"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, o := range renderings(planted(c.secret), c.shows) {
				for _, secret := range []string{plantedPaste, plantedField} {
					if strings.Contains(o.text, secret) {
						t.Errorf("%s shows %s: %s", o.how, secret, o.text)
					}
				}
				if !strings.Contains(o.text, "[redacted]") {
					t.Errorf("%s has no mask: %s", o.how, o.text)
				}
				for _, s := range o.show {
					if !strings.Contains(o.text, s) {
						t.Errorf("%s does not show %s: %s", o.how, s, o.text)
					}
				}
			}
		})
	}
}

// TestAnswerShowsNonSecret_0073: with Secret a non-nil false, the paste and
// the field values are shown, so the mask follows the step, not a blanket
// rule (MADR D2).
func TestAnswerShowsNonSecret_0073(t *testing.T) {
	shows := map[string]string{"fmt": "false", "text": "false", "json": "false"}
	for _, o := range renderings(planted(ptr(false)), shows) {
		for _, s := range append([]string{plantedPaste, plantedField}, o.show...) {
			if !strings.Contains(o.text, s) {
				t.Errorf("%s does not show %s: %s", o.how, s, o.text)
			}
		}
		if strings.Contains(o.text, "[redacted]") {
			t.Errorf("%s masks an answer whose step is not secret: %s", o.how, o.text)
		}
	}
}
