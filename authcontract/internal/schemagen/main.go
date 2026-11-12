// Command schemagen writes authcontract.schema.json, the JSON Schema of the
// package authcontract (0073-MADR D4). It reads the struct types by reflect,
// and the constants of each enum type by go/parser over the package's
// source, and writes, in source order:
//
//   - one $defs entry per struct type, an object whose required members are
//     the fields without omitempty or omitzero, and which allows additional
//     properties; a pointer field carries "x-nullable": true;
//   - one $defs entry per named string type, {"type": "string", "enum":
//     [...]} listing every constant of the type, with "x-open-enum": true,
//     and, for a type in labels, "x-enum-labels".
//
// The schema's $id ends with "v" and authcontract.SchemaVersion. The
// package carries the line
//
//	//go:generate go run ./internal/schemagen -output authcontract.schema.json
//
// so make generate-check holds the committed schema to the types.
//
// Usage:
//
//	go run ./internal/schemagen [-dir DIR] [-output FILE]
//
// -dir is the package's directory (default "."); with no -output the schema
// is written to standard output.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
)

const (
	draft  = "https://json-schema.org/draft/2020-12/schema"
	idBase = "https://github.com/maccavelli/go-llmprovider-sdk/authcontract/schema/v"
	// kwType is the keyword type, and jsonString its value for a string.
	kwType     = "type"
	jsonString = "string"
)

// structs are the package's struct types, read by reflect. A struct type the
// source declares and this list lacks, or the reverse, is an error.
var structs = []reflect.Type{
	reflect.TypeFor[authcontract.Method](),
	reflect.TypeFor[authcontract.Consent](),
	reflect.TypeFor[authcontract.Link](),
	reflect.TypeFor[authcontract.Field](),
	reflect.TypeFor[authcontract.Step](),
	reflect.TypeFor[authcontract.Answer](),
	reflect.TypeFor[authcontract.Failure](),
}

// labels holds, by enum type, each value's label, written as the type's
// "x-enum-labels" in the enum's order ("Amendment 2026-11-11: a session-kind
// type and its labels" of 0073-MADR). A label is display text; the module's
// tests hold this table to the amendment's.
var labels = map[string]map[string]string{
	"SessionKind": {
		"chatgpt":      "ChatGPT",
		"chatgpt_plan": "ChatGPT plan",
		"grok":         "Grok",
		"kilo_device":  "Kilo",
	},
}

var (
	pkgPath        = reflect.TypeFor[authcontract.Method]().PkgPath()
	timeType       = reflect.TypeFor[time.Time]()
	rawMessageType = reflect.TypeFor[json.RawMessage]()
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "schemagen:", err)
		os.Exit(1)
	}
}

// run parses args, generates the schema of the package in -dir, and writes it
// to -output, or to stdout when -output is not given.
func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("schemagen", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("dir", ".", "the package's directory")
	output := fs.String("output", "", "the file to write (default: standard output)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	data, err := generate(*dir)
	if err != nil {
		return err
	}
	if *output == "" {
		_, err = stdout.Write(data)
		return err
	}
	return os.WriteFile(*output, data, 0o644) //nolint:gosec // G306: the schema is a committed, public file
}

// generate returns the schema of the package in dir: two-space indent, a
// trailing newline.
func generate(dir string) ([]byte, error) {
	decls, err := parsePackage(dir)
	if err != nil {
		return nil, err
	}
	byName := map[string]reflect.Type{}
	for _, t := range structs {
		byName[t.Name()] = t
	}
	var defs object
	seen := map[string]bool{}
	for _, d := range decls.types {
		var entry object
		switch d.kind {
		case structType:
			t, ok := byName[d.name]
			if !ok {
				return nil, fmt.Errorf("struct type %s is not in schemagen's list", d.name)
			}
			seen[d.name] = true
			if entry, err = objectEntry(t); err != nil {
				return nil, err
			}
		case stringType:
			entry = enumEntry(d.name, decls.consts[d.name])
		}
		defs = append(defs, member{d.name, entry})
	}
	for _, t := range structs {
		if !seen[t.Name()] {
			return nil, fmt.Errorf("struct type %s is in schemagen's list, not in the source", t.Name())
		}
	}
	schema := object{
		{"$schema", draft},
		{"$id", idBase + strconv.Itoa(authcontract.SchemaVersion)},
		{"$defs", defs},
	}
	out, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// enumEntry is a named string type's entry: its values, open, with labels
// when the table has them.
func enumEntry(name string, values []string) object {
	if values == nil {
		values = []string{}
	}
	entry := object{{kwType, jsonString}, {"enum", values}, {"x-open-enum", true}}
	if table, ok := labels[name]; ok {
		var named object
		for _, v := range values {
			if label, ok := table[v]; ok {
				named = append(named, member{v, label})
			}
		}
		entry = append(entry, member{"x-enum-labels", named})
	}
	return entry
}

// objectEntry is a struct type's entry, its properties in field order.
func objectEntry(t reflect.Type) (object, error) {
	var props object
	required := []string{}
	for f := range t.Fields() {
		if !f.IsExported() {
			continue
		}
		if f.Anonymous {
			return nil, fmt.Errorf("%s.%s: an embedded field is not supported", t.Name(), f.Name)
		}
		name, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" && opts == "" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		ft := f.Type
		nullable := ft.Kind() == reflect.Pointer
		if nullable {
			ft = ft.Elem()
		}
		prop, err := valueSchema(ft)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", t.Name(), f.Name, err)
		}
		if nullable {
			prop = append(prop, member{"x-nullable", true})
		}
		props = append(props, member{name, prop})
		optional := slices.ContainsFunc(strings.Split(opts, ","), func(o string) bool { return o == "omitempty" || o == "omitzero" })
		if !optional {
			required = append(required, name)
		}
	}
	entry := object{{kwType, "object"}, {"properties", props}}
	if len(required) > 0 {
		entry = append(entry, member{"required", required})
	}
	return append(entry, member{"additionalProperties", true}), nil
}

// valueSchema is the schema of a value of type t: a $ref for the package's
// own named types, a JSON type otherwise.
func valueSchema(t reflect.Type) (object, error) {
	switch {
	case t == timeType:
		return object{{kwType, jsonString}, {"format", "date-time"}}, nil
	case t == rawMessageType:
		return object{}, nil
	case t.PkgPath() == pkgPath && t.Name() != "":
		return object{{"$ref", "#/$defs/" + t.Name()}}, nil
	}
	switch t.Kind() {
	case reflect.String:
		return object{{kwType, jsonString}}, nil
	case reflect.Bool:
		return object{{kwType, "boolean"}}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return object{{kwType, "integer"}}, nil
	case reflect.Slice:
		items, err := valueSchema(t.Elem())
		if err != nil {
			return nil, err
		}
		return object{{kwType, "array"}, {"items", items}}, nil
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("a map keyed by %s is not supported", t.Key())
		}
		values, err := valueSchema(t.Elem())
		if err != nil {
			return nil, err
		}
		return object{{kwType, "object"}, {"additionalProperties", values}}, nil
	default:
		return nil, fmt.Errorf("a value of type %s is not supported", t)
	}
}

type declKind int

const (
	structType declKind = iota
	stringType
)

// typeDecl is a type the source declares.
type typeDecl struct {
	name string
	kind declKind
}

// declarations are a package's exported struct and string types in source
// order, and each string type's constants' values in source order.
type declarations struct {
	types  []typeDecl
	consts map[string][]string
}

// parsePackage reads the non-test .go files of dir, in name order.
func parsePackage(dir string) (declarations, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return declarations{}, err
	}
	files = slices.DeleteFunc(files, func(name string) bool { return strings.HasSuffix(name, "_test.go") })
	if len(files) == 0 {
		return declarations{}, fmt.Errorf("%s: no Go file", dir)
	}
	slices.Sort(files)
	d := declarations{consts: map[string][]string{}}
	fset := token.NewFileSet()
	for _, name := range files {
		f, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			return declarations{}, err
		}
		for _, decl := range f.Decls {
			g, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			if err := d.add(fset, g); err != nil {
				return declarations{}, err
			}
		}
	}
	for typ := range d.consts {
		if !slices.ContainsFunc(d.types, func(t typeDecl) bool { return t.name == typ && t.kind == stringType }) {
			return declarations{}, fmt.Errorf("constants of %s, which is not a string type of the package", typ)
		}
	}
	return d, nil
}

// add records g's exported types and its typed constants. A constant whose
// type the source does not spell, by an implicit repetition of the line above
// or by a conversion, is an error, since go/parser alone cannot type it.
func (d *declarations) add(fset *token.FileSet, g *ast.GenDecl) error {
	for _, spec := range g.Specs {
		if s, ok := spec.(*ast.ValueSpec); ok && g.Tok == token.CONST && s.Type == nil {
			if err := untyped(fset, s); err != nil {
				return err
			}
			continue
		}
		switch s := spec.(type) {
		case *ast.TypeSpec:
			if !s.Name.IsExported() {
				continue
			}
			switch t := s.Type.(type) {
			case *ast.StructType:
				d.types = append(d.types, typeDecl{s.Name.Name, structType})
			case *ast.Ident:
				if t.Name != "string" {
					return fmt.Errorf("%s: type %s: only string types are enums", fset.Position(s.Pos()), s.Name.Name)
				}
				d.types = append(d.types, typeDecl{s.Name.Name, stringType})
			default:
				return fmt.Errorf("%s: type %s: not a struct or a string type", fset.Position(s.Pos()), s.Name.Name)
			}
		case *ast.ValueSpec:
			if g.Tok != token.CONST {
				continue
			}
			typ, ok := s.Type.(*ast.Ident)
			if !ok {
				continue
			}
			for i, n := range s.Names {
				if i >= len(s.Values) {
					return fmt.Errorf("%s: %s has no value of its own", fset.Position(n.Pos()), n.Name)
				}
				lit, ok := s.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return fmt.Errorf("%s: %s's value is not a string literal", fset.Position(n.Pos()), n.Name)
				}
				v, err := strconv.Unquote(lit.Value)
				if err != nil {
					return fmt.Errorf("%s: %s: %w", fset.Position(n.Pos()), n.Name, err)
				}
				d.consts[typ.Name] = append(d.consts[typ.Name], v)
			}
		}
	}
	return nil
}

// untyped refuses a constant spec with no type that may yet be typed: one
// with no value, which repeats the spec above it, or one whose value is a
// conversion to an exported type.
func untyped(fset *token.FileSet, s *ast.ValueSpec) error {
	if len(s.Values) == 0 {
		return fmt.Errorf("%s: %s has no value of its own", fset.Position(s.Pos()), s.Names[0].Name)
	}
	for i, v := range s.Values {
		if call, ok := v.(*ast.CallExpr); ok {
			if fn, ok := call.Fun.(*ast.Ident); ok && fn.IsExported() {
				return fmt.Errorf("%s: %s is a conversion to %s; write it as %s %s = \"...\"",
					fset.Position(s.Pos()), s.Names[i].Name, fn.Name, s.Names[i].Name, fn.Name)
			}
		}
	}
	return nil
}

// member is one member of an object.
type member struct {
	key   string
	value any
}

// object is a JSON object whose members keep their order.
type object []member

// MarshalJSON writes the members in order.
func (o object) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, m := range o {
		if i > 0 {
			b.WriteByte(',')
		}
		k, err := json.Marshal(m.key)
		if err != nil {
			return nil, err
		}
		v, err := json.Marshal(m.value)
		if err != nil {
			return nil, fmt.Errorf("member %s: %w", m.key, err)
		}
		b.Write(k)
		b.WriteByte(':')
		b.Write(v)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}
