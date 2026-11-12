package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
)

// Each test below holds the schema to 0073-MADR D4 twice: as committed, so
// that a change to the types not regenerated fails, and as the generator
// writes it now, so that a change to the generator fails before it is run.
func eachSchema(t *testing.T, check func(t *testing.T, schema map[string]any)) {
	t.Helper()
	read := map[string]func(t *testing.T) []byte{
		"committed": func(t *testing.T) []byte {
			data, err := os.ReadFile(committed)
			if err != nil {
				t.Fatal(err)
			}
			return data
		},
		"generated": generateTo,
	}
	for _, name := range []string{"committed", "generated"} {
		t.Run(name, func(t *testing.T) {
			var schema map[string]any
			if err := json.Unmarshal(read[name](t), &schema); err != nil {
				t.Fatalf("not a JSON object: %v", err)
			}
			check(t, schema)
		})
	}
}

// entries returns the schema's $defs entries.
func entries(t *testing.T, schema map[string]any) map[string]map[string]any {
	t.Helper()
	defs, ok := schema["$defs"].(map[string]any)
	if !ok {
		t.Fatal("no $defs object")
	}
	out := map[string]map[string]any{}
	for name, v := range defs {
		entry, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("$defs entry %s is not an object", name)
		}
		out[name] = entry
	}
	return out
}

// keywords calls visit with each keyword of the schema and its place: the
// root's, each $defs entry's, and, within an entry, those of each property,
// items and additionalProperties schema. Property names, $defs names and the
// members of x-enum-labels are not keywords.
func keywords(schema map[string]any, visit func(at, keyword string, value any)) {
	var walk func(at string, s map[string]any)
	walk = func(at string, s map[string]any) {
		for _, k := range slices.Sorted(maps.Keys(s)) {
			visit(at, k, s[k])
			switch k {
			case "$defs", "properties":
				sub, _ := s[k].(map[string]any)
				for _, name := range slices.Sorted(maps.Keys(sub)) {
					if m, ok := sub[name].(map[string]any); ok {
						walk(at+"/"+k+"/"+name, m)
					}
				}
			case "items", "additionalProperties":
				if m, ok := s[k].(map[string]any); ok {
					walk(at+"/"+k, m)
				}
			}
		}
	}
	walk("#", schema)
}

// d4Keywords are the keywords D4 lists, with the amendment's x-enum-labels.
var d4Keywords = []string{
	"$schema", "$id", "$defs", "$ref", "type", "properties", "required", "enum", "items",
	"additionalProperties", "format", "x-open-enum", "x-nullable", "x-enum-labels",
}

// TestSchemaKeywords_0073: the schema holds no oneOf, anyOf, allOf, not, if,
// then, else, description or title, and no keyword outside D4's list; its
// one format is date-time; each enum $defs entry has "x-open-enum": true; and
// x-enum-labels is on SessionKind's entry and no other.
func TestSchemaKeywords_0073(t *testing.T) {
	refused := []string{"oneOf", "anyOf", "allOf", "not", "if", "then", "else", "description", "title"}
	eachSchema(t, func(t *testing.T, schema map[string]any) {
		var labelled []string
		keywords(schema, func(at, k string, v any) {
			switch {
			case slices.Contains(refused, k):
				t.Errorf("%s: %s, which D4 refuses", at, k)
			case !slices.Contains(d4Keywords, k):
				t.Errorf("%s: %s, a keyword outside D4's list", at, k)
			case k == "format" && v != "date-time":
				t.Errorf("%s: format %v; D4's one format is date-time", at, v)
			case k == "x-enum-labels":
				labelled = append(labelled, at)
			}
		})
		for name, entry := range entries(t, schema) {
			if _, ok := entry["enum"]; ok && entry["x-open-enum"] != true {
				t.Errorf("$defs entry %s: an enum without \"x-open-enum\": true", name)
			}
		}
		if want := []string{"#/$defs/SessionKind"}; !slices.Equal(labelled, want) {
			t.Errorf("x-enum-labels at %v, want at %v only", labelled, want)
		}
	})
}

// structTypes are the package's struct types, this test's own list.
var structTypes = []reflect.Type{
	reflect.TypeFor[authcontract.Method](),
	reflect.TypeFor[authcontract.Consent](),
	reflect.TypeFor[authcontract.Link](),
	reflect.TypeFor[authcontract.Field](),
	reflect.TypeFor[authcontract.Step](),
	reflect.TypeFor[authcontract.Answer](),
	reflect.TypeFor[authcontract.Failure](),
}

// TestSchemaMarksPointers_0073: exactly the pointer fields, found by
// reflect, carry "x-nullable": true, and they are D4's four: Step.default,
// Step.error, Answer.secret and Answer.confirm.
func TestSchemaMarksPointers_0073(t *testing.T) {
	var pointers []string
	for _, typ := range structTypes {
		for f := range typ.Fields() {
			if f.Type.Kind() == reflect.Pointer {
				name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
				pointers = append(pointers, typ.Name()+"."+name)
			}
		}
	}
	slices.Sort(pointers)
	if d4 := []string{"Answer.confirm", "Answer.secret", "Step.default", "Step.error"}; !slices.Equal(pointers, d4) {
		t.Fatalf("the pointer fields are %v; D4 names %v", pointers, d4)
	}
	eachSchema(t, func(t *testing.T, schema map[string]any) {
		var marked, objects []string
		keywords(schema, func(at, k string, v any) {
			if k == "x-nullable" {
				parts := strings.Split(at, "/")
				if len(parts) != 5 || parts[1] != "$defs" || parts[3] != "properties" || v != true {
					t.Errorf("%s: x-nullable %v, want true and on a property only", at, v)
					return
				}
				marked = append(marked, parts[2]+"."+parts[4])
			}
		})
		for name, entry := range entries(t, schema) {
			if entry["type"] == "object" {
				objects = append(objects, name)
			}
		}
		slices.Sort(marked)
		slices.Sort(objects)
		if !slices.Equal(marked, pointers) {
			t.Errorf("x-nullable on %v, want exactly the pointer fields %v", marked, pointers)
		}
		var names []string
		for _, typ := range structTypes {
			names = append(names, typ.Name())
		}
		slices.Sort(names)
		if !slices.Equal(objects, names) {
			t.Errorf("object entries %v, want the struct types %v", objects, names)
		}
	})
}

// TestSchemaIDIsSchemaVersion_0073: the $id's last segment is "v" and
// SchemaVersion, and the schema states no module version (MADR D4).
func TestSchemaIDIsSchemaVersion_0073(t *testing.T) {
	eachSchema(t, func(t *testing.T, schema map[string]any) {
		id, _ := schema["$id"].(string)
		if want := "v" + strconv.Itoa(authcontract.SchemaVersion); path.Base(id) != want {
			t.Errorf("$id %q: last segment %q, want %q (SchemaVersion)", id, path.Base(id), want)
		}
		keywords(schema, func(at, k string, _ any) {
			if k == "x-module-version" {
				t.Errorf("%s: x-module-version; the schema states no module version", at)
			}
		})
	})
}

// sourceEnums reads the package's non-test files with go/parser, this test's
// own reading: each string type, and each typed constant's value by type,
// in source order.
func sourceEnums(t *testing.T) (types []string, consts map[string][]string) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(pkgDir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(files)
	consts = map[string][]string{}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch s := n.(type) {
			case *ast.TypeSpec:
				if id, ok := s.Type.(*ast.Ident); ok && id.Name == "string" {
					types = append(types, s.Name.Name)
				}
			case *ast.ValueSpec:
				id, ok := s.Type.(*ast.Ident)
				if !ok {
					return true
				}
				for _, v := range s.Values {
					lit, ok := v.(*ast.BasicLit)
					if !ok {
						t.Fatalf("%s: a constant of %s that is not a literal", fset.Position(v.Pos()), id.Name)
					}
					value, err := strconv.Unquote(lit.Value)
					if err != nil {
						t.Fatal(err)
					}
					consts[id.Name] = append(consts[id.Name], value)
				}
			}
			return true
		})
	}
	return types, consts
}

// TestSchemaListsEveryConstant_0073: each enum $defs entry lists exactly the
// constants of its type that go/parser finds, in source order, YieldsUnknown
// and ClientUnknown included, and each string type has an entry.
func TestSchemaListsEveryConstant_0073(t *testing.T) {
	types, consts := sourceEnums(t)
	if !slices.Contains(consts["Yields"], string(authcontract.YieldsUnknown)) ||
		!slices.Contains(consts["ClientIdentity"], string(authcontract.ClientUnknown)) {
		t.Fatalf("the source's Yields %v or ClientIdentity %v lacks unknown", consts["Yields"], consts["ClientIdentity"])
	}
	eachSchema(t, func(t *testing.T, schema map[string]any) {
		defs := entries(t, schema)
		for _, typ := range types {
			entry, ok := defs[typ]
			if !ok {
				t.Errorf("%s: no $defs entry", typ)
				continue
			}
			raw, _ := entry["enum"].([]any)
			var got []string
			for _, v := range raw {
				s, _ := v.(string)
				got = append(got, s)
			}
			if !slices.Equal(got, consts[typ]) {
				t.Errorf("$defs entry %s lists %q, want its constants %q", typ, got, consts[typ])
			}
		}
		for name, entry := range defs {
			if _, ok := entry["enum"]; ok && !slices.Contains(types, name) {
				t.Errorf("$defs entry %s: an enum of no string type of the package", name)
			}
		}
	})
}

// sessionKindLabels is 0073-MADR's "Amendment 2026-11-11: a session-kind type
// and its labels": each value and its label, in the type's order.
var sessionKindLabels = []struct{ value, label string }{
	{"chatgpt", "ChatGPT"},
	{"chatgpt_plan", "ChatGPT plan"},
	{"grok", "Grok"},
	{"kilo_device", "Kilo"},
}

// TestSchemaSessionKindLabels_0073: the schema has a $defs entry SessionKind,
// though no property refers to it, open, with an x-enum-labels object whose
// members are exactly its enum values, each the amendment's label; no other
// entry carries x-enum-labels.
func TestSchemaSessionKindLabels_0073(t *testing.T) {
	eachSchema(t, func(t *testing.T, schema map[string]any) {
		defs := entries(t, schema)
		keywords(schema, func(at, k string, v any) {
			if k == "$ref" && v == "#/$defs/SessionKind" {
				t.Errorf("%s refers to SessionKind; the amendment adds no member of it", at)
			}
		})
		entry, ok := defs["SessionKind"]
		if !ok {
			t.Fatal("no $defs entry SessionKind")
		}
		if entry["x-open-enum"] != true {
			t.Error("SessionKind: no \"x-open-enum\": true")
		}
		var values []string
		for _, l := range sessionKindLabels {
			values = append(values, l.value)
		}
		if enum, _ := entry["enum"].([]any); !reflect.DeepEqual(enum, toAny(values)) {
			t.Errorf("SessionKind's enum %v, want %v", enum, values)
		}
		labels, ok := entry["x-enum-labels"].(map[string]any)
		if !ok {
			t.Fatalf("SessionKind: no x-enum-labels object (got %v)", entry["x-enum-labels"])
		}
		if got := slices.Sorted(maps.Keys(labels)); !slices.Equal(got, slices.Sorted(slices.Values(values))) {
			t.Errorf("SessionKind's x-enum-labels has members %v, want exactly its enum values %v", got, values)
		}
		for _, l := range sessionKindLabels {
			if got, _ := labels[l.value].(string); got != l.label {
				t.Errorf("SessionKind's label for %s is %q, want %q", l.value, got, l.label)
			}
		}
		for name, e := range defs {
			if _, ok := e["x-enum-labels"]; ok && name != "SessionKind" {
				t.Errorf("$defs entry %s carries x-enum-labels; only SessionKind's does", name)
			}
		}
	})
}

func toAny(s []string) []any {
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}
