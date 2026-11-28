// Package recorderr holds the error types that tell a client record file's
// failure from an ordinary read error, for the two packages that must agree on
// them: llmprovider/auth, which returns them, and internal/signinflow, which
// asks for them (0082-MADR, fifth amendment of 2026-11-18, R-14; sixth, F-1;
// 0095-MADR FU10).
//
// The types are in an internal package so that no code outside this module can
// name them, and so no store of another module can return one: a custom
// ClientStore has exactly one route to "undecodable", wrapping encoding/json's
// error, and a method on its own error type is not a second one.
package recorderr

// Oversize marks the error of a record file over its size bound: the file
// exists and is corruption, not a permission or an input or output failure. Err
// is the error it marks, whose text it shows.
type Oversize struct{ Err error }

// Error is the marked error's text.
func (e *Oversize) Error() string { return e.Err.Error() }

// Unwrap returns the marked error.
func (e *Oversize) Unwrap() error { return e.Err }

// Implausible marks the error of a record file that decodes but names a
// client id that is not printable text, such as bytes that were not UTF-8,
// which JSON decoding turns into replacement characters: no issuer gave it,
// so the file is corruption, as an oversize one is (0095-MADR FU10). Err is
// the error it marks, whose text it shows.
type Implausible struct{ Err error }

// Error is the marked error's text.
func (e *Implausible) Error() string { return e.Err.Error() }

// Unwrap returns the marked error.
func (e *Implausible) Unwrap() error { return e.Err }
