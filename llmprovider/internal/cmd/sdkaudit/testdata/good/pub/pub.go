// Package pub is a planted package that meets R50 to R52.
//
//	err := pub.Do()
package pub

import "errors"

// ErrBad is the error Do returns.
var ErrBad = errors.New("bad")

// Thing is a value. It is safe for concurrent use.
type Thing struct{}

// Do returns ErrBad.
func Do() error { return ErrBad }

// MarshalJSON writes the empty object. It is a method of a standard
// interface, so only the exemption lets this doc pass R52.
func (Thing) MarshalJSON() ([]byte, error) { return []byte("{}"), nil }
