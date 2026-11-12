# Changelog

All notable changes to the `authcontract` module are recorded here, in the
form of [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/). The
module is tagged `authcontract/vX.Y.Z`, and a section is named for its tag's
version without the prefix. Within a major version, v0 included, it only
adds: the evolution rules in [README.md](README.md) forbid a rename, a
removal and a new required field.

## [Unreleased]

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
