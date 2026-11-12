package authcontract_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
)

// wantSessionKinds is 0073-MADR's "Amendment 2026-11-11: a session-kind type
// and its labels": each constant's name and value. A value added on either
// side fails until a record adds it on the other.
var wantSessionKinds = map[string]string{
	"SessionChatGPT":     "chatgpt",
	"SessionChatGPTPlan": "chatgpt_plan",
	"SessionGrok":        "grok",
	"SessionKiloDevice":  "kilo_device",
}

// constantsOf returns the constants of type typ that the package's non-test
// files declare, by name, read with go/parser.
func constantsOf(t *testing.T, typ string) map[string]string {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			g, ok := decl.(*ast.GenDecl)
			if !ok || g.Tok != token.CONST {
				continue
			}
			for _, spec := range g.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				if id, ok := vs.Type.(*ast.Ident); !ok || id.Name != typ {
					continue
				}
				for i, n := range vs.Names {
					lit, ok := vs.Values[i].(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						t.Fatalf("%s: %s's value is not a string literal", fset.Position(n.Pos()), n.Name)
					}
					v, err := strconv.Unquote(lit.Value)
					if err != nil {
						t.Fatal(err)
					}
					got[n.Name] = v
				}
			}
		}
	}
	return got
}

// TestSessionKindValues_0073: the SessionKind constants are exactly the
// amendment's table, by name and value.
func TestSessionKindValues_0073(t *testing.T) {
	got := constantsOf(t, "SessionKind")
	for _, name := range slices.Sorted(maps.Keys(wantSessionKinds)) {
		if v, ok := got[name]; !ok {
			t.Errorf("%s: missing; want %q", name, wantSessionKinds[name])
		} else if v != wantSessionKinds[name] {
			t.Errorf("%s = %q, want %q", name, v, wantSessionKinds[name])
		}
	}
	for _, name := range slices.Sorted(maps.Keys(got)) {
		if _, ok := wantSessionKinds[name]; !ok {
			t.Errorf("%s = %q: not in the amendment's table", name, got[name])
		}
	}
}

// TestUnknownSessionKindKept_0073: a session kind this version does not list
// decodes with no error, and re-encodes kept (MADR D6 rule 2).
func TestUnknownSessionKindKept_0073(t *testing.T) {
	const data = `"a_future_session_kind"`
	var k authcontract.SessionKind
	if err := json.Unmarshal([]byte(data), &k); err != nil {
		t.Fatalf("decode %s: %v", data, err)
	}
	if k != "a_future_session_kind" {
		t.Errorf("decoded %q, want a_future_session_kind", k)
	}
	b, err := json.Marshal(k)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != data {
		t.Errorf("re-encoded %s, want %s", b, data)
	}
}
