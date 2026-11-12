package authcontract_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
)

// wantTags is each exported field's JSON tag, as 0073-MADR D2 writes it.
var wantTags = map[string]map[string]string{
	"Method": {
		"SchemaVersion":  "schema_version",
		"ID":             "id",
		"Kind":           "kind",
		"Label":          "label",
		"Detail":         "detail,omitempty",
		"Completion":     "completion",
		"Paste":          "paste,omitempty",
		"FirstPaste":     "first_paste,omitempty",
		"Yields":         "yields",
		"ClientIdentity": "client_identity",
		"Refreshable":    "refreshable",
		"Revocable":      "revocable",
		"Slot":           "slot,omitempty",
		"Command":        "command,omitempty",
		"Features":       "features,omitempty",
		"Consents":       "consents,omitempty",
		"Links":          "links,omitempty",
		"Meta":           "_meta,omitempty",
	},
	"Consent": {
		"ID":       "id",
		"Required": "required",
		"Text":     "text,omitempty",
		"Links":    "links,omitempty",
	},
	"Link": {
		"Rel":   "rel",
		"URL":   "url",
		"Title": "title,omitempty",
	},
	"Field": {
		"Key":      "key",
		"Label":    "label",
		"Type":     "type",
		"Options":  "options,omitempty",
		"Required": "required,omitempty",
		"Default":  "default,omitempty",
	},
	"Step": {
		"SchemaVersion": "schema_version",
		"Seq":           "seq",
		"Kind":          "kind",
		"URL":           "url,omitempty",
		"Code":          "code,omitempty",
		"ExpiresAt":     "expires_at,omitzero",
		"Accepts":       "accepts,omitempty",
		"Secret":        "secret,omitempty",
		"ConfirmID":     "confirm_id,omitempty",
		"Text":          "text,omitempty",
		"Default":       "default,omitempty",
		"Links":         "links,omitempty",
		"Fields":        "fields,omitempty",
		"Yields":        "yields,omitempty",
		"Error":         "error,omitempty",
		"Meta":          "_meta,omitempty",
	},
	"Answer": {
		"SchemaVersion": "schema_version",
		"Seq":           "seq",
		"Secret":        "secret,omitempty",
		"Paste":         "paste,omitempty",
		"Confirm":       "confirm,omitempty",
		"Fields":        "fields,omitempty",
		"Meta":          "_meta,omitempty",
	},
	"Failure": {
		"Kind":      "kind",
		"Message":   "message",
		"Retryable": "retryable",
	},
}

// TestJSONNames_0073: each exported field's JSON name is D2's, and each list
// or map field is omitempty (0073-PLAN Phase 1b).
func TestJSONNames_0073(t *testing.T) {
	types := []any{
		authcontract.Method{}, authcontract.Consent{}, authcontract.Link{}, authcontract.Field{},
		authcontract.Step{}, authcontract.Answer{}, authcontract.Failure{},
	}
	if len(types) != len(wantTags) {
		t.Fatalf("the test lists %d types, the table %d", len(types), len(wantTags))
	}
	for _, v := range types {
		typ := reflect.TypeOf(v)
		want, ok := wantTags[typ.Name()]
		if !ok {
			t.Errorf("%s: not in the table", typ.Name())
			continue
		}
		seen := map[string]bool{}
		for f := range typ.Fields() {
			if !f.IsExported() {
				continue
			}
			seen[f.Name] = true
			tag := f.Tag.Get("json")
			if w, ok := want[f.Name]; !ok {
				t.Errorf("%s.%s: tag %q, not in D2's table", typ.Name(), f.Name, tag)
			} else if tag != w {
				t.Errorf("%s.%s: tag %q, want %q", typ.Name(), f.Name, tag, w)
			}
			if k := f.Type.Kind(); (k == reflect.Slice || k == reflect.Map) && !strings.HasSuffix(tag, ",omitempty") {
				t.Errorf("%s.%s: a %s field without omitempty (tag %q)", typ.Name(), f.Name, k, tag)
			}
		}
		for name := range want {
			if !seen[name] {
				t.Errorf("%s.%s: in D2's table, not in the type", typ.Name(), name)
			}
		}
	}
}

func ptr[T any](v T) *T { return &v }

func marshal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal(%T): %v", v, err)
	}
	return string(b)
}

// TestStepSecretJSON_0073: Step.secret is absent when false, and
// Answer.secret is absent when nil and present when false (MADR D4).
func TestStepSecretJSON_0073(t *testing.T) {
	if got := marshal(t, authcontract.Step{SchemaVersion: 1, Seq: 2, Kind: authcontract.StepPaste, Secret: true}); !strings.Contains(got, `"secret":true`) {
		t.Errorf("a secret step: %s, want \"secret\":true", got)
	}
	if got := marshal(t, authcontract.Step{SchemaVersion: 1, Seq: 2, Kind: authcontract.StepWaiting}); strings.Contains(got, `"secret"`) {
		t.Errorf("a step that is not secret: %s, want no \"secret\" member", got)
	}
	if got := marshal(t, authcontract.Answer{SchemaVersion: 1, Seq: 2, Paste: "x"}); strings.Contains(got, `"secret"`) {
		t.Errorf("an answer with a nil Secret: %s, want no \"secret\" member", got)
	}
	if got := marshal(t, authcontract.Answer{SchemaVersion: 1, Seq: 2, Secret: ptr(false), Confirm: ptr(false)}); !strings.Contains(got, `"secret":false`) {
		t.Errorf("an answer whose Secret is false: %s, want \"secret\":false", got)
	}
}

// TestAnswerJSONCarriesPaste_0073: the JSON is not masked, since the answer
// must reach the sign-in, and it decodes back equal (MADR D2).
func TestAnswerJSONCarriesPaste_0073(t *testing.T) {
	const paste = "sk-PLANTED-0073"
	in := authcontract.Answer{
		SchemaVersion: authcontract.SchemaVersion, Seq: 4, Secret: ptr(true), Paste: paste,
		Fields: map[string]string{"api_token": "fv-PLANTED-0073"},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(paste)) || !bytes.Contains(b, []byte("fv-PLANTED-0073")) {
		t.Fatalf("json.Marshal: %s, want the paste and the field value", b)
	}
	var out authcontract.Answer
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Errorf("decoded %#v\nwant %#v", out, in)
	}
	if out.Paste != paste {
		t.Errorf("decoded Paste %q, want %q", out.Paste, paste)
	}
}

// roundTrip decodes data into a new T and re-encodes it, failing unless the
// bytes come back the same; it returns the decoded value.
func roundTrip[T any](t *testing.T, data string) T {
	t.Helper()
	var v T
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		t.Fatalf("decode %s into %T: %v", data, v, err)
	}
	if got := marshal(t, v); got != data {
		t.Errorf("re-encoded %T: %s\nwant %s", v, got, data)
	}
	return v
}

// TestUnknownEnumValuesKept_0073: an enum value this version does not list
// decodes, and re-encodes kept (MADR D6 rule 2).
func TestUnknownEnumValuesKept_0073(t *testing.T) {
	step := roundTrip[authcontract.Step](t, `{"schema_version":1,"seq":1,"kind":"a_future_kind"}`)
	if step.Kind != "a_future_kind" {
		t.Errorf("Step.Kind %q, want a_future_kind", step.Kind)
	}
	method := roundTrip[authcontract.Method](t, `{"schema_version":1,"id":"x","kind":"oauth","label":"X",`+
		`"completion":"a_future_completion","yields":"unknown","client_identity":"unknown","refreshable":false,"revocable":false}`)
	if method.Completion != "a_future_completion" {
		t.Errorf("Method.Completion %q, want a_future_completion", method.Completion)
	}
	failure := roundTrip[authcontract.Failure](t, `{"kind":"a_future_kind","message":"m","retryable":false}`)
	if failure.Kind != "a_future_kind" {
		t.Errorf("Failure.Kind %q, want a_future_kind", failure.Kind)
	}
}
