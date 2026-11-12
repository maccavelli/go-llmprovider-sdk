package schemacheck_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract/internal/schemacheck"
)

func parse(t *testing.T, schema string) *schemacheck.Schema {
	t.Helper()
	s, err := schemacheck.Parse([]byte(schema))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return s
}

// TestSchemacheckRefusesUnknownKeywords_0073: a keyword D4 does not list is
// an error naming it, wherever it is (0073-MADR D4, PLAN Phase 3 step 4).
func TestSchemacheckRefusesUnknownKeywords_0073(t *testing.T) {
	for _, c := range []struct{ name, schema, keyword string }{
		{"in an entry", `{"$defs": {"A": {"type": "object", "patternProperties": {"^x": {"type": "string"}}}}}`, "patternProperties"},
		{"in a property", `{"$defs": {"A": {"type": "object", "properties": {"p": {"type": "string", "patternProperties": {}}}}}}`, "patternProperties"},
		{"at the root", `{"$defs": {}, "patternProperties": {}}`, "patternProperties"},
		{"oneOf", `{"$defs": {"A": {"oneOf": [{"type": "string"}]}}}`, "oneOf"},
		{"description", `{"$defs": {"A": {"type": "string", "description": "x"}}}`, "description"},
		{"$id below the root", `{"$defs": {"A": {"$id": "x", "type": "string"}}}`, "$id"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := schemacheck.Parse([]byte(c.schema))
			if err == nil || !strings.Contains(err.Error(), `"`+c.keyword+`"`) {
				t.Errorf("Parse(%s) = %v, want an error naming %q", c.schema, err, c.keyword)
			}
		})
	}
}

// annotated and bare are one schema with and without every annotation the
// validator accepts.
const (
	annotated = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://example.com/schema/v1",
  "$defs": {
    "Kind": {"type": "string", "enum": ["a", "b"], "x-open-enum": true, "x-enum-labels": {"a": "A", "b": "B", "c": "C"}},
    "Thing": {"type": "object",
      "properties": {"kind": {"$ref": "#/$defs/Kind"}, "ok": {"type": "boolean", "x-nullable": true},
        "next": {"$ref": "#/$defs/Thing", "x-nullable": true}},
      "required": ["kind"], "additionalProperties": true}
  }
}`
	bare = `{
  "$defs": {
    "Kind": {"type": "string", "enum": ["a", "b"]},
    "Thing": {"type": "object",
      "properties": {"kind": {"$ref": "#/$defs/Kind"}, "ok": {"type": "boolean"}, "next": {"$ref": "#/$defs/Thing"}},
      "required": ["kind"], "additionalProperties": true}
  }
}`
)

// TestSchemacheckAcceptsAnnotations_0073: a schema carrying $schema, $id,
// x-open-enum, x-nullable and x-enum-labels parses, and validates each
// instance as the same schema without them does: an open enum still refuses
// an unlisted value, a nullable member still refuses null, and a label adds
// no value. The committed schema, which carries them, validates the vectors.
func TestSchemacheckAcceptsAnnotations_0073(t *testing.T) {
	with, without := parse(t, annotated), parse(t, bare)
	for _, c := range []struct {
		def, instance string
		valid         bool
	}{
		{"Thing", `{"kind": "a"}`, true},
		{"Thing", `{"kind": "a", "ok": false, "next": {"kind": "b"}}`, true},
		{"Thing", `{"kind": "c"}`, false},
		{"Thing", `{"kind": "a", "ok": null}`, false},
		{"Thing", `{"kind": "a", "next": null}`, false},
		{"Thing", `{"ok": true}`, false},
		{"Kind", `"b"`, true},
		{"Kind", `"c"`, false},
	} {
		errWith, errWithout := with.Validate(c.def, []byte(c.instance)), without.Validate(c.def, []byte(c.instance))
		if (errWith == nil) != c.valid || (errWithout == nil) != c.valid {
			t.Errorf("%s %s: with the annotations %v, without %v; want valid %t", c.def, c.instance, errWith, errWithout, c.valid)
		}
	}

	data, err := os.ReadFile("../../authcontract.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	committed, err := schemacheck.Parse(data)
	if err != nil {
		t.Fatalf("the committed schema: %v", err)
	}
	paths, err := filepath.Glob("../../testdata/vectors/*.json")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no vectors (%v)", err)
	}
	defs := map[string]string{"method": "Method", "step": "Step", "answer": "Answer", "failure": "Failure", "session": "SessionKind"}
	for _, p := range paths {
		prefix, _, _ := strings.Cut(filepath.Base(p), "-")
		def, ok := defs[prefix]
		if !ok {
			t.Errorf("%s: its name gives no type", p)
			continue
		}
		v, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := committed.Validate(def, v); err != nil {
			t.Errorf("%s: %v", filepath.Base(p), err)
		}
	}
}

// rules is a schema with one property per rule the validator applies.
const rules = `{"$defs": {"Obj": {"type": "object",
  "properties": {
    "n": {"type": "integer"}, "x": {"type": "number"}, "s": {"type": "string"},
    "t": {"type": "string", "format": "date-time"}, "l": {"type": "array", "items": {"type": "string"}},
    "m": {"type": "object", "additionalProperties": {"type": "string"}}, "b": {"type": "boolean"}, "z": {"type": "null"}},
  "required": ["n"], "additionalProperties": false}}}`

// TestSchemacheckRules_0073: each keyword is applied as JSON Schema defines
// it, each error names the keyword and the place, and a malformed schema or
// instance is refused.
func TestSchemacheckRules_0073(t *testing.T) {
	s := parse(t, rules)
	for _, c := range []struct{ def, instance, want string }{
		{"Obj", `{"n": 1}`, ""},
		{"Obj", `{"n": 1, "x": 2.5, "s": "a", "t": "2026-11-12T10:15:00.5Z", "l": ["a"], "m": {"k": "v"}, "b": true, "z": null}`, ""},
		{"Obj", `{"n": 1.5}`, "#/n: type: want integer, got number"},
		{"Obj", `{"n": "1"}`, "#/n: type: want integer, got string"},
		{"Obj", `{"n": 1, "x": "2"}`, "#/x: type: want number, got string"},
		{"Obj", `{}`, "#: required: n is missing"},
		{"Obj", `{"n": 1, "extra": 1}`, "#: additionalProperties: extra is not allowed"},
		{"Obj", `{"n": 1, "t": "yesterday"}`, `#/t: format: "yesterday" is not a date-time`},
		{"Obj", `{"n": 1, "l": ["a", 1]}`, "#/l/1: type: want string, got number"},
		{"Obj", `{"n": 1, "m": {"k": 1}}`, "#/m/k: type: want string, got number"},
		{"Obj", `{"n": 1, "b": "true"}`, "#/b: type: want boolean, got string"},
		{"Obj", `{"n": 1, "z": 0}`, "#/z: type: want null, got number"},
		{"Obj", `[]`, "#: type: want object, got array"},
		{"Obj", `{"n": 1} {}`, "data after the JSON value"},
		{"Obj", `{`, "the instance"},
		{"Missing", `{}`, "no $defs entry Missing"},
	} {
		err := s.Validate(c.def, []byte(c.instance))
		switch {
		case c.want == "" && err != nil:
			t.Errorf("%s %s: %v, want valid", c.def, c.instance, err)
		case c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)):
			t.Errorf("%s %s: %v, want an error holding %q", c.def, c.instance, err, c.want)
		}
	}
	for _, c := range []struct{ schema, want string }{
		{`not json`, "the schema"},
		{`[]`, "not an object"},
		{`{"$defs": []}`, "$defs is missing or not an object"},
		{`{"$schema": 1, "$defs": {}}`, "$schema is not a string"},
		{`{"$defs": {"A": 1}}`, "#/$defs/A: not an object"},
		{`{"$defs": {"A": {"type": "strng"}}}`, "strng is not a JSON type"},
		{`{"$defs": {"A": {"$ref": "#/$defs/B"}}}`, "#/$defs/B names no $defs entry"},
		{`{"$defs": {"A": {"$ref": "other.json#/A"}}}`, "names no $defs entry"},
		{`{"$defs": {"A": {"format": "email"}}}`, "email is not date-time"},
		{`{"$defs": {"A": {"required": [1]}}}`, "1 is not a string"},
		{`{"$defs": {"A": {"required": "n"}}}`, "required: not an array"},
		{`{"$defs": {"A": {"enum": "a"}}}`, "enum: not an array"},
		{`{"$defs": {"A": {"properties": []}}}`, "properties: not an object"},
		{`{"$defs": {"A": {"items": true}}}`, "items: not a schema object"},
		{`{"$defs": {"A": {"additionalProperties": {"type": "x"}}}}`, "x is not a JSON type"},
	} {
		_, err := schemacheck.Parse([]byte(c.schema))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("Parse(%s) = %v, want an error holding %q", c.schema, err, c.want)
		}
	}
}
