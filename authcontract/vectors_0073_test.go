package authcontract_test

import (
	"bytes"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/authcontract/internal/schemacheck"
)

// The golden vectors of 0073-MADR D5, and the committed schema of D4.
const (
	vectorDir  = "testdata/vectors"
	forwardDir = "testdata/vectors/forward"
	schemaFile = "authcontract.schema.json"
)

// vectorType returns a new value of the type a vector's name gives, and that
// type's $defs entry.
func vectorType(t *testing.T, name string) (any, string) {
	t.Helper()
	switch {
	case strings.HasPrefix(name, "method-"):
		return new(authcontract.Method), "Method"
	case strings.HasPrefix(name, "step-"):
		return new(authcontract.Step), "Step"
	case strings.HasPrefix(name, "answer-"):
		return new(authcontract.Answer), "Answer"
	case strings.HasPrefix(name, "failure-"):
		return new(authcontract.Failure), "Failure"
	case strings.HasPrefix(name, "session-kind-"):
		return new(authcontract.SessionKind), "SessionKind"
	}
	t.Fatalf("%s: its name gives no type", name)
	return nil, ""
}

// vectors returns the names of the .json files in dir, sorted.
func vectors(t *testing.T, dir string) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatalf("%s: no vector", dir)
	}
	names := make([]string, len(paths))
	for i, p := range paths {
		names[i] = filepath.Base(p)
	}
	slices.Sort(names)
	return names
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// committedSchema parses the committed schema with the strict validator.
func committedSchema(t *testing.T) *schemacheck.Schema {
	t.Helper()
	s, err := schemacheck.Parse(readFile(t, schemaFile))
	if err != nil {
		t.Fatalf("%s: %v", schemaFile, err)
	}
	return s
}

// decodeAny decodes data into a generic value, its numbers as json.Number.
func decodeAny(t *testing.T, data []byte) any {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("decode %s: %v", data, err)
	}
	return v
}

// TestVectorsRoundTrip_0073: every vector outside forward/ decodes into the
// type its name gives and re-encodes to the same compact bytes (MADR D5);
// answer-confirm-false.json keeps its "confirm":false (D4).
func TestVectorsRoundTrip_0073(t *testing.T) {
	for _, name := range vectors(t, vectorDir) {
		data := readFile(t, filepath.Join(vectorDir, name))
		var want bytes.Buffer
		if err := json.Compact(&want, data); err != nil {
			t.Errorf("%s: not JSON: %v", name, err)
			continue
		}
		v, _ := vectorType(t, name)
		if err := json.Unmarshal(data, v); err != nil {
			t.Errorf("%s: decode into %T: %v", name, v, err)
			continue
		}
		got, err := json.Marshal(v)
		if err != nil {
			t.Errorf("%s: re-encode %T: %v", name, v, err)
			continue
		}
		if !bytes.Equal(got, want.Bytes()) {
			t.Errorf("%s: re-encoded %s\nwant         %s", name, got, want.Bytes())
		}
	}
	var a authcontract.Answer
	if err := json.Unmarshal(readFile(t, filepath.Join(vectorDir, "answer-confirm-false.json")), &a); err != nil {
		t.Fatal(err)
	}
	if a.Confirm == nil || *a.Confirm {
		t.Errorf("answer-confirm-false.json: Confirm decoded as %v, want a non-nil false", a.Confirm)
	}
	if b, err := json.Marshal(a); err != nil || !bytes.Contains(b, []byte(`"confirm":false`)) {
		t.Errorf("answer-confirm-false.json: re-encoded %s (%v), want \"confirm\":false kept", b, err)
	}
}

// TestVectorsValidate_0073: every vector outside forward/ validates against
// the committed schema, step-paste.json with no expires_at among them (MADR
// D5).
func TestVectorsValidate_0073(t *testing.T) {
	schema := committedSchema(t)
	names := vectors(t, vectorDir)
	for _, name := range names {
		_, def := vectorType(t, name)
		if err := schema.Validate(def, readFile(t, filepath.Join(vectorDir, name))); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	if !slices.Contains(names, "step-paste.json") {
		t.Fatal("step-paste.json: missing")
	}
	paste, ok := decodeAny(t, readFile(t, filepath.Join(vectorDir, "step-paste.json"))).(map[string]any)
	if !ok {
		t.Fatal("step-paste.json: not an object")
	}
	if _, ok := paste["expires_at"]; ok {
		t.Error("step-paste.json has an expires_at; the vector is a paste step without one")
	}
	if paste["secret"] != true {
		t.Errorf("step-paste.json: secret %v, want true", paste["secret"])
	}
}

// forwardVectors is what each forward/ vector holds that this version does
// not know: an unknown enum value, which strict validation refuses, or an
// unknown member, which a reader drops and validation allows (MADR D5).
var forwardVectors = map[string]struct {
	valid   bool
	unknown []string
}{
	"step-unknown-kind.json":         {valid: false},
	"method-unknown-completion.json": {valid: false},
	"answer-unknown-field.json":      {valid: true, unknown: []string{"a_future_member"}},
	"session-kind-unknown.json":      {valid: false},
}

// TestForwardVectors_0073: each forward/ vector, a newer producer's output,
// decodes without error and drops its unknown member, keeping everything
// else; one with an unknown enum value keeps it and fails strict validation
// on that enum; one whose only novelty is an unknown member validates.
func TestForwardVectors_0073(t *testing.T) {
	schema := committedSchema(t)
	names := vectors(t, forwardDir)
	if want := slices.Sorted(maps.Keys(forwardVectors)); !slices.Equal(names, want) {
		t.Fatalf("forward/ holds %v, want %v", names, want)
	}
	for _, name := range names {
		fv := forwardVectors[name]
		data := readFile(t, filepath.Join(forwardDir, name))
		v, def := vectorType(t, name)
		if err := json.Unmarshal(data, v); err != nil {
			t.Errorf("%s: decode into %T: %v", name, v, err)
			continue
		}
		got, err := json.Marshal(v)
		if err != nil {
			t.Errorf("%s: re-encode: %v", name, err)
			continue
		}
		want := decodeAny(t, data)
		if obj, ok := want.(map[string]any); ok {
			if meta, _ := obj["_meta"].(map[string]any); meta["example"] == nil {
				t.Errorf("%s: no _meta.example entry", name)
			}
			for _, member := range fv.unknown {
				if _, ok := obj[member]; !ok {
					t.Errorf("%s: no member %s; the vector is to carry it", name, member)
				}
				delete(obj, member)
			}
		}
		if g := decodeAny(t, got); !reflect.DeepEqual(g, want) {
			t.Errorf("%s: re-encoded %s\nwant everything but %v kept", name, got, fv.unknown)
		}
		err = schema.Validate(def, data)
		switch {
		case fv.valid && err != nil:
			t.Errorf("%s: %v; a vector whose only novelty is an unknown member validates", name, err)
		case !fv.valid && err == nil:
			t.Errorf("%s: passed strict validation; its unknown enum value must fail it", name)
		case !fv.valid && !strings.Contains(err.Error(), "enum:"):
			t.Errorf("%s: refused for %v; want its unknown enum value refused", name, err)
		}
	}
}
