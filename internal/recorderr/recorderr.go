// Package recorderr holds the one error type that tells a client record file's
// failure from an ordinary read error, for the two packages that must agree on
// it: llmprovider/auth, which returns it, and internal/signinflow, which asks for
// it (0082-MADR, fifth amendment of 2026-11-18, R-14; sixth, F-1).
//
// The type is in an internal package so that no code outside this module can
// name it, and so no store of another module can return it: a custom ClientStore
// has exactly one route to "undecodable", wrapping encoding/json's error, and a
// method on its own error type is not a second one.
package recorderr

// Oversize marks the error of a record file over its size bound: the file
// exists and is corruption, not a permission or an input or output failure. Err
// is the error it marks, whose text it shows.
type Oversize struct{ Err error }

// Error is the marked error's text.
func (e *Oversize) Error() string { return e.Err.Error() }

// Unwrap returns the marked error.
func (e *Oversize) Unwrap() error { return e.Err }
