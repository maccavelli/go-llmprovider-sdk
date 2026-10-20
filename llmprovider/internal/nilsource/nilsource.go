// Package nilsource tells a typed nil from a value: a nil pointer, map,
// slice, func, chan or interface held in a non-nil interface, such as a
// TokenSource whose dynamic value is nil (0050-MADR D1, D2). llmprovider and
// catalog refuse such a source where it enters. Nothing here is exported
// outside the module.
package nilsource

import "reflect"

// Is reports whether v is a typed nil: not a nil interface, but a value of a
// nilable kind that is nil. A nil interface is not, nor is a value of any
// other kind. No method of v is called.
func Is(v any) bool {
	if v == nil {
		return false
	}
	switch rv := reflect.ValueOf(v); rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.Interface:
		return rv.IsNil()
	default:
		return false
	}
}
