# Changelog

All notable changes to the `authcontract` module are recorded here, in the
form of [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/). The
module is tagged `authcontract/vX.Y.Z`, and a section is named for its tag's
version without the prefix. Within a major version, v0 included, it only
adds: the evolution rules in [README.md](README.md) forbid a rename, a
removal and a new required field.

## [Unreleased]

## [v0.1.0] - 2026-11-12

### Added

- The module `github.com/maccavelli/go-llmprovider-sdk/authcontract`,
  standard library only, with the sign-in vocabulary of
  [0073-MADR](../docs/decisions/0073-MADR-publish-an-auth-contract-module.md)
  D2 and D3: `Method`, with `Consent`, `Link`, `Feature`, `Kind`,
  `Completion`, `Paste`, `Yields`, `ClientIdentity` and `LinkRel`; `Step`,
  with `StepKind`, `Field` and `FieldType`; `Answer`, whose `String`,
  `GoString` and `LogValue` mask its paste and field values unless its
  `Secret` is a non-nil `false`; `Failure` and `ErrorKind`'s eighteen
  `Failure*` constants; and `SchemaVersion`, 1.
- `SessionKind`, with `SessionChatGPT`, `SessionChatGPTPlan`, `SessionGrok`
  and `SessionKiloDevice`, the strings of the SDK's `auth.SessionKind`
  (0073-MADR, "Amendment 2026-11-11: a session-kind type and its labels").
- The evolution rules, in the package doc and in [README.md](README.md).
- `authcontract.schema.json`, the JSON Schema (draft 2020-12) generated from
  the types by `go generate` (0073-MADR D4): each struct a `$defs` entry,
  each enum a string `enum` with `"x-open-enum": true`, each pointer member
  `"x-nullable": true`, and `SessionKind`'s labels in its `x-enum-labels`.
  Its `$id` ends with `/v1`, for `SchemaVersion` 1.
- Golden vectors in `testdata/vectors/`, one JSON file per case, and in
  `testdata/vectors/forward/`, a newer producer's output with unknown enum
  values and members (0073-MADR D5).
