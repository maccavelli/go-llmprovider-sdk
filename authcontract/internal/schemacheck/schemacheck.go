// Package schemacheck validates JSON against a JSON Schema written with the
// keywords of 0073-MADR D4 alone, and refuses a schema that uses any other.
//
// It validates by type, properties, required, enum, $ref into $defs, items,
// additionalProperties and format (date-time only). It accepts the
// annotations $schema and $id at the root, and x-open-enum, x-nullable and
// x-enum-labels anywhere, and ignores them for validation: an x-open-enum
// entry still refuses an unlisted value, which is what strict validation
// means, and an x-nullable member still refuses null. Any other keyword is
// an error naming it, so a generator cannot emit a keyword this validator
// does not know. Standard library only.
package schemacheck

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"slices"
	"strings"
	"time"
)

// refPrefix is the one form of $ref the schema uses: an entry of the root's
// $defs.
const refPrefix = "#/$defs/"

// rootKeywords are the keywords of the schema's root object.
var rootKeywords = map[string]bool{
	"$schema": true,
	"$id":     true,
	"$defs":   true,
}

// annotations are accepted anywhere and ignored for validation.
var annotations = map[string]bool{
	"x-open-enum":   true,
	"x-nullable":    true,
	"x-enum-labels": true,
}

// The JSON types, the values of the keyword type.
const (
	typeObject  = "object"
	typeArray   = "array"
	typeString  = "string"
	typeInteger = "integer"
	typeNumber  = "number"
	typeBoolean = "boolean"
	typeNull    = "null"
)

var types = []string{typeObject, typeArray, typeString, typeInteger, typeNumber, typeBoolean, typeNull}

// Schema is a parsed schema whose every keyword is one D4 lists. It is not
// changed after Parse, so it is safe for concurrent use.
type Schema struct {
	defs map[string]map[string]any
}

// Parse reads a schema and checks every keyword in it. It returns an error
// when data is not one JSON object, when a keyword is not one D4 lists
// (naming the keyword and where it is), when a keyword's value has the wrong
// form, or when a $ref names no $defs entry.
func Parse(data []byte) (*Schema, error) {
	v, err := decode(data)
	if err != nil {
		return nil, fmt.Errorf("schemacheck: the schema: %w", err)
	}
	root, ok := v.(map[string]any)
	if !ok {
		return nil, errors.New("schemacheck: the schema is not an object")
	}
	s := &Schema{defs: map[string]map[string]any{}}
	for _, key := range sortedKeys(root) {
		if !rootKeywords[key] {
			return nil, fmt.Errorf("schemacheck: #: unknown keyword %q", key)
		}
	}
	for _, key := range []string{"$schema", "$id"} {
		if v, ok := root[key]; ok {
			if _, ok := v.(string); !ok {
				return nil, fmt.Errorf("schemacheck: #: %s is not a string", key)
			}
		}
	}
	defs, ok := root["$defs"].(map[string]any)
	if !ok {
		return nil, errors.New("schemacheck: #: $defs is missing or not an object")
	}
	for name, entry := range defs {
		obj, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("schemacheck: #/$defs/%s: not an object", name)
		}
		s.defs[name] = obj
	}
	for _, name := range sortedKeys(defs) {
		if err := s.check("#/$defs/"+name, s.defs[name]); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// check checks the keywords of the schema object at path.
func (s *Schema) check(path string, schema map[string]any) error {
	for _, key := range sortedKeys(schema) {
		v := schema[key]
		at := path + "/" + key
		var err error
		switch key {
		case "type":
			if t, ok := v.(string); !ok || !slices.Contains(types, t) {
				err = fmt.Errorf("schemacheck: %s: %v is not a JSON type", at, v)
			}
		case "properties":
			err = s.checkProperties(at, v)
		case "required":
			err = checkStrings(at, v)
		case "enum":
			if _, ok := v.([]any); !ok {
				err = fmt.Errorf("schemacheck: %s: not an array", at)
			}
		case "$ref":
			if _, ok := s.resolve(v); !ok {
				err = fmt.Errorf("schemacheck: %s: %v names no $defs entry", at, v)
			}
		case "items":
			err = s.checkSubschema(at, v)
		case "additionalProperties":
			if _, ok := v.(bool); !ok {
				err = s.checkSubschema(at, v)
			}
		case "format":
			if v != "date-time" {
				err = fmt.Errorf("schemacheck: %s: %v is not date-time, the one format the schema uses", at, v)
			}
		default:
			if !annotations[key] {
				err = fmt.Errorf("schemacheck: %s: unknown keyword %q", path, key)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Schema) checkProperties(path string, v any) error {
	props, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("schemacheck: %s: not an object", path)
	}
	for _, name := range sortedKeys(props) {
		if err := s.checkSubschema(path+"/"+name, props[name]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Schema) checkSubschema(path string, v any) error {
	obj, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("schemacheck: %s: not a schema object", path)
	}
	return s.check(path, obj)
}

func checkStrings(path string, v any) error {
	list, ok := v.([]any)
	if !ok {
		return fmt.Errorf("schemacheck: %s: not an array", path)
	}
	for _, item := range list {
		if _, ok := item.(string); !ok {
			return fmt.Errorf("schemacheck: %s: %v is not a string", path, item)
		}
	}
	return nil
}

// resolve returns the $defs entry ref names.
func (s *Schema) resolve(ref any) (map[string]any, bool) {
	r, ok := ref.(string)
	if !ok || !strings.HasPrefix(r, refPrefix) {
		return nil, false
	}
	entry, ok := s.defs[strings.TrimPrefix(r, refPrefix)]
	return entry, ok
}

// Validate checks instance against the $defs entry def. It returns an error
// when def names no entry, when instance is not one JSON value, or when the
// value breaks a keyword: the error names the keyword and the value's place,
// as a JSON pointer.
func (s *Schema) Validate(def string, instance []byte) error {
	schema, ok := s.defs[def]
	if !ok {
		return fmt.Errorf("schemacheck: no $defs entry %s", def)
	}
	v, err := decode(instance)
	if err != nil {
		return fmt.Errorf("schemacheck: the instance: %w", err)
	}
	return s.validateValue("#", schema, v)
}

// validateValue checks v, at path, against schema.
func (s *Schema) validateValue(path string, schema map[string]any, v any) error {
	if ref, ok := schema["$ref"]; ok {
		target, _ := s.resolve(ref)
		if err := s.validateValue(path, target, v); err != nil {
			return err
		}
	}
	if t, ok := schema["type"].(string); ok && !hasType(t, v) {
		return fmt.Errorf("schemacheck: %s: type: want %s, got %s", path, t, typeOf(v))
	}
	if enum, ok := schema["enum"].([]any); ok && !slices.ContainsFunc(enum, func(e any) bool { return reflect.DeepEqual(e, v) }) {
		return fmt.Errorf("schemacheck: %s: enum: %s is not listed", path, show(v))
	}
	if schema["format"] == "date-time" {
		if str, ok := v.(string); ok {
			if _, err := time.Parse(time.RFC3339, str); err != nil {
				return fmt.Errorf("schemacheck: %s: format: %q is not a date-time", path, str)
			}
		}
	}
	switch val := v.(type) {
	case map[string]any:
		return s.validateObject(path, schema, val)
	case []any:
		if items, ok := schema["items"].(map[string]any); ok {
			for i, item := range val {
				if err := s.validateValue(fmt.Sprintf("%s/%d", path, i), items, item); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// validateObject checks an object's members: required, properties and
// additionalProperties.
func (s *Schema) validateObject(path string, schema map[string]any, obj map[string]any) error {
	var required []any
	if r, ok := schema["required"].([]any); ok {
		required = r
	}
	for _, r := range required {
		if name, ok := r.(string); ok && !hasMember(obj, name) {
			return fmt.Errorf("schemacheck: %s: required: %s is missing", path, name)
		}
	}
	var props map[string]any
	if p, ok := schema["properties"].(map[string]any); ok {
		props = p
	}
	for _, name := range sortedKeys(obj) {
		at := path + "/" + name
		if prop, ok := props[name].(map[string]any); ok {
			if err := s.validateValue(at, prop, obj[name]); err != nil {
				return err
			}
			continue
		}
		switch extra := schema["additionalProperties"].(type) {
		case bool:
			if !extra {
				return fmt.Errorf("schemacheck: %s: additionalProperties: %s is not allowed", path, name)
			}
		case map[string]any:
			if err := s.validateValue(at, extra, obj[name]); err != nil {
				return err
			}
		}
	}
	return nil
}

func hasMember(obj map[string]any, name string) bool {
	_, ok := obj[name]
	return ok
}

// hasType reports whether v is of the JSON type t.
func hasType(t string, v any) bool {
	switch t {
	case typeInteger:
		n, ok := v.(json.Number)
		if !ok {
			return false
		}
		f, err := n.Float64()
		return err == nil && f == math.Trunc(f)
	case typeNumber:
		_, ok := v.(json.Number)
		return ok
	default:
		return typeOf(v) == t
	}
}

// typeOf names v's JSON type, a number as number.
func typeOf(v any) string {
	switch v.(type) {
	case map[string]any:
		return typeObject
	case []any:
		return typeArray
	case string:
		return typeString
	case json.Number:
		return typeNumber
	case bool:
		return typeBoolean
	default:
		return typeNull
	}
}

func show(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

// decode reads exactly one JSON value, its numbers as json.Number.
func decode(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("data after the JSON value")
	}
	return v, nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
