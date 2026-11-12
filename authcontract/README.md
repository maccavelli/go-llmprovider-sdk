# authcontract

`github.com/maccavelli/go-llmprovider-sdk/authcontract` is the shared
vocabulary of signing in without an API key: method facts, steps, answers,
failure kinds and session kinds, as data. It has no flow, no HTTP and no
provider. It imports the standard library only, and its `go.mod` requires
nothing.

It is a module of its own, nested in this repository, so that a consumer can
use the types without the SDK, and so that the contract versions apart from
the SDK.
[0073-MADR](../docs/decisions/0073-MADR-publish-an-auth-contract-module.md)
decides it.

## The types

| Type | What it is |
| :--- | :--- |
| `Method` | one way to sign in, as facts a client reads before it starts, with its `Consent`s and `Link`s |
| `Step` | one thing a sign-in shows or asks; `Seq` rises by one per step, and the last is `done` or `failed` |
| `Answer` | what a `paste`, `confirm` or `form` step asked for |
| `Field` | one named input of a `form` step |
| `Failure` | why a sign-in failed, with its `ErrorKind` |
| `SessionKind` | the kind of a stored sign-in session |

Every enum is a named string with a constant per value, and every JSON name
is `snake_case`. An `Answer` formats itself with its paste and its field
values masked unless its `Secret` is a non-nil `false`; its JSON is not
masked, since the answer must reach the sign-in.

### Session kinds

A client shows a session kind by its label. The labels are in the JSON
Schema, as the `SessionKind` entry's `x-enum-labels`, not in Go.

| Constant | Value | Label |
| :--- | :--- | :--- |
| `SessionChatGPT` | `chatgpt` | ChatGPT |
| `SessionChatGPTPlan` | `chatgpt_plan` | ChatGPT plan |
| `SessionGrok` | `grok` | Grok |
| `SessionKiloDevice` | `kilo_device` | Kilo |

A label is display text: a reader never matches on it, and a later tag may
reword it. A value is never renamed.

## Tags

The module is tagged `authcontract/vX.Y.Z`, apart from the SDK's `vX.Y.Z`
tags. The first tag is `authcontract/v0.1.0`. The module stays at v0 until
its consumers have each shipped a release that uses it; a record then moves
it to `v1.0.0`. The evolution rules below hold at v0 too.

## Evolution rules

1. **Additive only within a major version, v0 included.** No rename, no
   removal, no change of a field's type or JSON name, and no new required
   field. A new field is optional, and its absence means the old behaviour.
2. **Enums are open.** A reader keeps an unknown value and renders it
   generically: an unknown step kind shows its `text` and offers cancel; an
   unknown method kind or completion is listed and not driven. A producer
   emits only the values of the schema of its version. `oauth` and
   `at_start` are known values, not unknown ones: a method that carries them
   is driven, and its first steps say its mechanism (D2).
3. **`schema_version` and features.** Every `Method`, `Step` and `Answer`
   carries `schema_version`. A reader refuses an object whose
   `schema_version` is higher than its own `SchemaVersion`. A `Method` names
   in `features` the additive capabilities its sign-in uses (`consent`,
   `form`, `paste`), so a client that cannot render one says so before it
   starts.
4. **`_meta` is the extension namespace,** keyed by owner (`gobble`, `magic`,
   and so on). Nothing in `_meta` is required to complete a sign-in. This SDK
   writes none.
5. **Secrets.** A `Method` and a `Step` carry no key material (R34); a
   `Method`'s `command` is display text, never run by a relay or a client. A
   step marks the answer it asks for as a secret with `secret`. An answer to
   such a step, and an answer whose step a reader does not hold, is a secret:
   never logged, never stored by a relay (Q11). A reader sets
   `Answer.secret` from the step it holds, never from the sender's copy.
6. **A break** needs a record, a new major version and a new
   `SchemaVersion`.

The rules are D6 of 0073-MADR; D2, R34 and Q11 are that record's
references.

## Using it

Require the module at a tag:

```bash
go get github.com/maccavelli/go-llmprovider-sdk/authcontract@v0.1.0
```

Pin a published tag. Do not `replace` the module with a local path in a
committed `go.mod`, and do not commit a `go.work`; this repository ignores
`go.work` and `go.work.sum` for the same reason.

## The schema and the vectors

At each tag, the module's source holds:

- `authcontract.schema.json`, the JSON Schema (draft 2020-12) generated from
  the types. Each struct is a `$defs` entry, each enum a string `enum` with
  `"x-open-enum": true`, and each pointer member `"x-nullable": true`. Its
  `$id` ends with `/v` and `SchemaVersion`.
- `testdata/vectors/`, golden vectors: one JSON file per case, named for its
  type and case, such as `step-paste.json` or `answer-confirm-false.json`.
  `testdata/vectors/forward/` holds a newer producer's output: unknown enum
  values and unknown members, which a reader must accept.

A consumer reads them from the tagged source in its module cache:

```bash
go mod download -json github.com/maccavelli/go-llmprovider-sdk/authcontract@v0.1.0
```

prints the `Dir` that holds them. A Go consumer decodes each vector into
the type its name gives, re-encodes it, and compares the bytes after
`json.Compact`; the vectors are in Go's encoding, which escapes `<`, `>`
and `&` in strings. A consumer in another language compares JSON values,
every member present or absent as in the vector.
