// Package authcontract is the shared vocabulary of signing in without an API
// key: method facts, steps, answers and failure kinds, as data. No flow, no
// HTTP, no provider. Standard library only.
//
// A [Method] is one way to sign in, as facts a client reads before it starts.
// A [Step] is one thing a sign-in shows or asks, and an [Answer] returns what
// a paste, confirm or form step asked for. A [Failure] ends a sign-in with an
// [ErrorKind]. A [SessionKind] names the kind of a stored session. Every type
// is a value, and its JSON is the wire form.
//
// # Evolution rules
//
// The module changes by these rules, which are D6 of this repository's
// docs/decisions/0073-MADR-publish-an-auth-contract-module.md; D2, R34 and
// Q11 are that record's references.
//
//  1. Additive only within a major version, v0 included. No rename, no
//     removal, no change of a field's type or JSON name, and no new required
//     field. A new field is optional, and its absence means the old
//     behaviour.
//  2. Enums are open. A reader keeps an unknown value and renders it
//     generically: an unknown step kind shows its `text` and offers cancel;
//     an unknown method kind or completion is listed and not driven. A
//     producer emits only the values of the schema of its version. `oauth`
//     and `at_start` are known values, not unknown ones: a method that
//     carries them is driven, and its first steps say its mechanism (D2).
//  3. `schema_version` and features. Every `Method`, `Step` and `Answer`
//     carries `schema_version`. A reader refuses an object whose
//     `schema_version` is higher than its own `SchemaVersion`. A `Method`
//     names in `features` the additive capabilities its sign-in uses
//     (`consent`, `form`, `paste`), so a client that cannot render one says
//     so before it starts.
//  4. `_meta` is the extension namespace, keyed by owner (`gobble`, `magic`,
//     and so on). Nothing in `_meta` is required to complete a sign-in. This
//     SDK writes none.
//  5. Secrets. A `Method` and a `Step` carry no key material (R34); a
//     `Method`'s `command` is display text, never run by a relay or a
//     client. A step marks the answer it asks for as a secret with
//     `secret`. An answer to such a step, and an answer whose step a reader
//     does not hold, is a secret: never logged, never stored by a relay
//     (Q11). A reader sets `Answer.secret` from the step it holds, never
//     from the sender's copy.
//  6. A break needs a record, a new major version and a new `SchemaVersion`.
package authcontract

//go:generate go run ./internal/schemagen -output authcontract.schema.json

// SchemaVersion counts the incompatible changes of this module's JSON. The
// evolution rules forbid one within a major version, so it is 1 for v0 and v1.
const SchemaVersion = 1
