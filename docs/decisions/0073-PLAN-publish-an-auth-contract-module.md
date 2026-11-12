---
status: in-progress
date: 2026-11-11
associated-madr: "0073-MADR-publish-an-auth-contract-module.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement the Nested Auth Contract Module, `authcontract`

Associated MADR: [0073-MADR-publish-an-auth-contract-module.md](0073-MADR-publish-an-auth-contract-module.md)

## Goal

After this PLAN, the repository holds a second Go module,
`github.com/maccavelli/go-llmprovider-sdk/authcontract`, standard library
only, tagged `authcontract/v0.1.0`. It holds the sign-in vocabulary of
0073-MADR D2 and D3, a generated JSON Schema (D4), golden vectors (D5) and
written evolution rules (D6). `make contract-check` (D7) and the extended
gates (D8) hold it to those rules in CI and before every commit.

The module is data only. Nothing in the root module imports it yet: that is
[0074-PLAN-one-sign-in-handle-and-method-facts.md](0074-PLAN-one-sign-in-handle-and-method-facts.md)'s
first import.

The MADR is `proposed`. Nothing below starts before the owner accepts it and
approves this PLAN.

## Scope

### In scope

* **The module** (D1, D2, D3, D6): `authcontract/go.mod`, `authcontract/doc.go`,
  `authcontract/method.go`, `authcontract/step.go`, `authcontract/answer.go`,
  `authcontract/errorkind.go`, `authcontract/README.md`,
  `authcontract/CHANGELOG.md`, and their tests; and
  `authcontract/sessionkind.go` *(added 2026-11-11, the MADR's "Amendment 2026-11-11: a session-kind type and its labels")*.
* **The schema** (D4): `authcontract/internal/schemagen/` (`main.go` and
  `main_test.go`),
  `authcontract/authcontract.schema.json`, and the `//go:generate` line in
  `authcontract/doc.go`.
* **The vectors and their validator** (D5): `authcontract/testdata/vectors/`,
  `authcontract/testdata/vectors/forward/`, `authcontract/internal/schemacheck/`.
* **The gate** (D7): `scripts/check_contract.py`, a `contract-check` target
  in the `Makefile`.
* **The other gates** (D8): `scripts/go-precheck.sh`, the `Makefile`'s
  `test`, `vet`, `lint` and `vuln` targets, `scripts/check_changelog.py`,
  `scripts/test_gates.py`, `.github/workflows/ci.yml`, `.gitignore`, and
  `tools/gate.py` with its count and list in `tools/README.md` (and
  `tools/test_tools.py`, where it pins them).
* **The docs:** `docs/architecture.md` (prose, D8), `docs/guides/api-standards.md`
  R1, `AGENTS.md` "Dependencies", `README.md` (one paragraph naming the
  module), and the amendment to
  `0015-MADR-canonical-sdk-api-and-module-layout.md`.
* **The tag** `authcontract/v0.1.0`, on the owner's word.

### Exported names (R55)

Every name is in package `authcontract`. "x/signin" is 0074-MADR's producer;
"gobble" is gobble-cli's `--json` steps and ACP elicitation mapping (the
owner's Q7, Q21); "Magic" is magic-cli-remote's envelope and its generated
Dart models (Q8, Q9).

| Name | Call site that needs it |
| :--- | :--- |
| `SchemaVersion` | every reader's refusal of a newer object (D6 rule 3): gobble, Magic |
| `type Feature`; `FeatureConsent`, `FeatureForm`, `FeaturePaste` | `Method.Features`, set by x/signin's `Methods`; Magic refuses to start a method whose feature its app cannot render |
| `type Kind`; `KindAPIKey`, `KindTokenPaste`, `KindDeviceCode`, `KindBrowser`, `KindOAuth`, `KindVendorImport`, `KindHostOnly` | x/signin's `Methods`; Magic's thin mappers for codex, grok, kilo and opencode (the report's M18), whose host-only methods are `KindHostOnly` and whose kilo and opencode OAuth entries are `KindOAuth` (the owner's O1) |
| `type Completion`; `CompletionValue`, `CompletionDevice`, `CompletionBrowserPaste`, `CompletionBrowserLoopback`, `CompletionAtStart`, `CompletionLocalFile`, `CompletionHostCommand` | x/signin's `Methods`; Magic's mapper for opencode's `command` and `external` entries (`CompletionHostCommand`) and for kilo's and opencode's OAuth entries (`CompletionAtStart`, O1) |
| `type Paste`; `PasteNone`, `PasteURLOrCode`, `PasteURLOnly`, `PasteCode` | `Method.Paste`, `Method.FirstPaste` and `Step.Accepts`, set by x/signin; Magic's paste box |
| `type Yields`; `YieldsAPIKey`, `YieldsOAuthSession`, `YieldsAccessOnly`, `YieldsDeviceKey`, `YieldsReadThrough`, `YieldsUnknown` | x/signin's `Methods` and `done` step; gobble's policy filter, which replaces `HasLogin`, `HasKeyLogin` and `offers` (the report's G11); `YieldsUnknown` for Magic's kilo and opencode `at_start` methods (MADR D2, O1) |
| `type ClientIdentity`; `ClientNone`, `ClientVendorBorrowed`, `ClientAppRegistered`, `ClientUnknown` | x/signin's `Methods`; gobble's refusal of Codex's client, and the phone's display of it (Q10); `ClientUnknown` for Magic's `at_start` methods (MADR D2, O1) |
| `type StepKind`; `StepOpenURL`, `StepShowCode`, `StepPaste`, `StepConfirm`, `StepForm`, `StepWaiting`, `StepDone`, `StepFailed` | x/signin's `Steps`; gobble's `--json` and elicitation; Magic's `provider.auth_event` |
| `type LinkRel`; `LinkTerms`, `LinkUsage`, `LinkNotice` | 0075-MADR D1's usage link on the plan's consent |
| `type FieldType`; `FieldText`, `FieldSecret`, `FieldSelect` | Magic's mapper for kilo's Copilot inputs and opencode's non-string fields (the report's §5.6.2) |
| `type ErrorKind` and D3's eighteen constants, `FailureCanceled` to `FailureOther` | x/signin's `ErrorKindOf` (0074-MADR D8); gobble's `--json` in place of `FailureKind` (Q15) |
| `type SessionKind`; `SessionChatGPT`, `SessionChatGPTPlan`, `SessionGrok`, `SessionKiloDevice` *(added 2026-11-11, the MADR's "Amendment 2026-11-11: a session-kind type and its labels")* | gobble-cli's `auth status --json` member `sessionKind`, whose values these are (gobble-cli `0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md` D8); magic-cli-remote's auth block, whose daemon holds the kind as this type and whose phone shows its label, generated from the schema's `x-enum-labels` (magic-cli-remote `0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md` D13). 0074-PLAN Phase 4's `TestSessionKindsMatchContract_0074` pins `auth.SessionKind` to it |
| `type Method` and its eighteen fields | x/signin's `Methods`; Magic's `providers.list_result`; `Command` for Magic's opencode mapper and the phone's copy button (the owner's O3) |
| `type Consent` and its four fields | 0075-MADR D1's `chatgpt_plan.background_use` |
| `type Link` and its three fields | `Method.Links`, `Consent.Links`, `Step.Links` |
| `type Field` and its six fields | `Step.Fields` |
| `type Step` and its sixteen fields | x/signin's `Steps`; `Secret` for every reader that redacts an answer: x/signin, gobble's ACP relay (its `_meta.gobble.step`), Magic's daemon and phone (MADR D2) |
| `type Answer` and its seven fields; `Answer.String`, `Answer.GoString`, `Answer.LogValue` | x/signin's `SignIn.Answer`; Magic's `provider.auth_answer`, logged redacted (Q11); `Secret`, set by the reader from the step it holds |
| `type Failure` and its three fields | `Step.Error`, set by x/signin on `failed` |

The constant names follow R21, a short prefix naming the type, with one
stated exception: `ErrorKind`'s eighteen constants are `Failure*`, which the
owner ruled on 2026-11-11 to keep (MADR D3). No other name is exported.
`internal/schemagen` and `internal/schemacheck` are internal. *(2026-11-11:
the sentence claimed R21 for every constant.)*

### Out of scope

* Any producer of these types: x/signin, descriptor facts and the wizard are
  0074's; consent is 0075's.
* The root module's requirement on the module, `scripts/check_deps.py`'s
  `REQUIRES`, and `scripts/check_tables.py`'s second module: 0074-PLAN Phase
  4, with the first import.
* gobble's and Magic's adoption, and Magic's Dart generator and its drift
  gate (their records).
* The module's `v1.0.0` (a later record, after 0074-MADR D9's promotion).
* A GitHub Release for a contract tag (MADR D8: none).

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts only after the owner has committed the last one.
* **Every new gate, assertion or test is seen to fail first,** on a planted
  break, on a scratch clone, never in the tree: `python3 -B tools/plant.py
  SPEC.json` for a planted source change (with `"command"` for a script or a
  `make` target), as `tools/README.md` describes. Each phase below names its
  plants. This record holds each failure line, quoted, in its execution
  record. A plant that does not fail is a deviation: stop and report it; do
  not loosen the check.
* **A plant in the nested module names its command.** `tools/plant.py`'s
  default command is `go test -count=1` with the spec's packages, run from
  the clone root (MADR C39), which cannot test `./authcontract`. Every
  Phase 1b and Phase 3 spec therefore sets `"command": ["go", "-C",
  "authcontract", "test", "-count=1", "-run", "<the test>", "./..."]`.
* **A plant that is not one replacement is a gate self-test.** A
  `tools/plant.py` plant replaces text that occurs once in a file that
  exists, and runs its command in a scratch clone (P5, P6). A plant that adds or deletes a file, edits
  `go.mod`'s requirements, or tags a commit is a test case in
  `scripts/test_gates.py`, on a `copy_tree` copy, as
  `PrecheckTest.test_precheck_deleted_file_checks_its_package` and
  `GateSelfTest` plant theirs (P8). A case that needs a tag first commits
  the copy's tree with `git commit-tree`, as `tools/_common.scratch_clone`
  does, so that no hook runs, then tags that commit in the copy. Phase 1a's
  and Phase 2's tables say which plants these are. *(2026-11-11: added; the
  plan named only `tools/plant.py`, which cannot add a file or a tag.)*
* **The drift check.** Every existing test, gate and golden passes unchanged.
  This PLAN changes no file under `llmprovider/` or `wizard/`.
* **Pre-add.** `make pre-add-check FILES="…"` on each phase's Go files. Phase
  1a teaches the check the nested module before any `authcontract/*.go` file
  exists, so no Go file of this PLAN is staged that the check cannot vet
  (AGENTS.md, "Pre-add checks": a file that fails is not committed).
* **Before staging:** `python3 -B tools/gate.py` (or `--markdown` for a
  records-only phase), `python3 -B tools/claims.py
  docs/decisions/0073-MADR-publish-an-auth-contract-module.md`, `make
  records-check`, `python3 -B tools/recordlint.py --changed`, `python3 -B
  tools/denyscan.py --changed`, `python3 -B tools/linkcheck.py`. Each exit
  status is captured before any filter.
* **Stage, then the owner commits** with `git commit --no-edit`. Pushes and
  tags happen on the owner's word, after the disclosure guard has run over
  the outgoing commits.
* **Stop conditions:**
  * a claim of the MADR no longer holds at the phase's base;
  * an existing test, gate or golden needs a change;
  * the module needs a requirement;
  * a plant is not caught;
  * `-race` reports a race.

  On any of these: stop, record a dated deviation here (with an amendment to
  the MADR when a decision or a fact changes), and wait.

## Implementation Steps

### Phase 0: records

1. Commit this PLAN, the MADR, and their rows in `docs/README.md`
   (`python3 -B tools/records.py add …`), as records only.
2. On the owner's acceptance, `python3 -B tools/records.py status` sets the
   MADR `accepted` and this PLAN `in-progress`, and the MADR's Decision
   Outcome records the acceptance and its date.
3. **Apply the supersession notes** the MADR's "Records this decision
   amends" lists, without rewriting the decisions:
   * `0015-MADR-canonical-sdk-api-and-module-layout.md`: a dated amendment
     section, "Amendment 2026-MM-DD: a second module for a wire contract
     (0073)", quoting D2's bullet and adding the MADR's sentence;
   * `docs/guides/api-standards.md` R1: the second sentence becomes "A
     second module needs its own MADR and a dependency, or a wire contract
     that must version apart from the SDK, that justifies it. (0015 D2, and
     its amendment for 0073)";
   * `AGENTS.md` "Dependencies": "The nested module `authcontract`
     (0073-MADR) requires nothing; the root module may require it at a tag."
4. Run `python3 -B tools/gate.py --markdown`, then stage.

Acceptance: `make records-check` and `make links-check` pass; the three
notes are in place; no source file changed.

### Phase 1a: the module file, and the pre-add check learns it (D1, D8)

This phase lands before any `authcontract/*.go` file, so that every later
phase's Go files pass the pre-add check for their own reasons (MADR D8).

1. **`authcontract/go.mod`:** `module github.com/maccavelli/go-llmprovider-sdk/authcontract`,
   and the `go` line of the root's `go.mod` as it stands when this phase
   runs (`go 1.27.3` at `773b84c`; MADR D1), nothing else. No `go.sum`. No
   Go file yet. *Amended 2026-11-11: the line follows R54's floor rather
   than naming 1.27.3, since GO-2026-6859 is expected to raise that floor
   first.* *Amended again 2026-11-11: the record that raises it is
   [0077-MADR-go-1-27-4-for-go-2026-6859.md](0077-MADR-go-1-27-4-for-go-2026-6859.md),
   accepted that day. Its PLAN runs before this phase ("Rollout and
   Rollback", step 0), so the line is `go 1.27.4`. A root `go.mod` that
   still says `go 1.27.3` when this phase starts is a stop.*
2. **`.gitignore`** gains `go.work` and `go.work.sum` (D1).
3. **`scripts/go-precheck.sh`.** For each given `.go` file under a directory
   that holds a `go.mod` other than the root's, its module's packages are
   vetted and tested with `go -C <module> vet` and `go -C <module> test
   -race`, linted with `golangci-lint run -c ../.golangci.yml` from the
   module's directory, and `go -C <module> mod tidy -diff` is run. The root
   module's run is unchanged. With no arguments, every module is checked; a
   module with no package yet is reported as such and is not a failure (the
   run's output is recorded here; a failure there is a deviation).
4. **`scripts/test_gates.py`'s `PrecheckTest`** gains the two cases below,
   each on a `copy_tree` copy with `GO_PRECHECK_SKIP_VULN=1`, as its existing
   case runs. *(2026-11-11: the plants were named without saying where they
   run; a new file is not a `tools/plant.py` plant.)*

| Check | Planted break, on a scratch copy | Expected failure |
| :--- | :--- | :--- |
| `go-precheck.sh`, nested module (`test_precheck_nested_module_vets_its_file`) | `bash scripts/go-precheck.sh authcontract/plant.go`, the new file `package authcontract` with a `fmt.Printf("%d", "x")` in it | exit 1, and `go vet` names the file; run once by hand on a copy of the tree before step 3, the run fails on the directory instead (recorded verbatim, the MADR's unverified item) |
| `go-precheck.sh`, nested tidy (`test_precheck_nested_module_tidy`) | the same, with a clean `authcontract/plant.go` and `require golang.org/x/sys v0.47.0` added to `authcontract/go.mod` with no import | exit 1, and `go mod tidy -diff` names the requirement |

Acceptance: `make pre-add-check` passes on the tree; each plant failed as
expected; no `authcontract/*.go` file exists.

### Phase 1b: the module's types (D1, D2, D3, D6)

1. *(Phase 1a holds `authcontract/go.mod`.)*
2. *(Phase 1a holds `.gitignore`.)*
3. **`authcontract/doc.go`:** the package doc, carrying D6's six rules
   word for word, and `SchemaVersion`.
4. **`authcontract/method.go`:** `Feature`, `Kind`, `Completion`, `Paste`,
   `Yields`, `ClientIdentity`, `LinkRel`, `FieldType` and their constants;
   `Method`, `Consent`, `Link`, `Field` (D2). Each type's doc says it is a
   value, safe for concurrent use while no goroutine changes it (R51); each
   constant's doc says what it means.
5. **`authcontract/step.go`:** `StepKind` and its constants, `Step`.
6. **`authcontract/answer.go`:** `Answer`, with value-receiver `String`,
   `GoString` and `LogValue` that print `Paste` as `[redacted]` when it is not
   empty, and each `Fields` value as `[redacted]`, keys shown and sorted,
   unless `Secret` is a non-nil false (MADR D2); `Secret` itself is shown.
7. **`authcontract/errorkind.go`:** `ErrorKind`, D3's eighteen constants in
   D3's order, and `Failure`.

   *(Added 2026-11-11, the MADR's "Amendment 2026-11-11: a session-kind type and its labels":)*
   **`authcontract/sessionkind.go`:** `SessionKind` and its four constants,
   `SessionChatGPT`, `SessionChatGPTPlan`, `SessionGrok` and
   `SessionKiloDevice`, in the amendment's order, with no method. The
   type's doc says it is a value, safe for concurrent use while no
   goroutine changes it (R51), that it is open (D6 rule 2), and that its
   labels are in the schema's `x-enum-labels`, not in Go; each constant's
   doc says which session it names. `authcontract/README.md` (step 8) lists
   the four values and their labels.
8. **`authcontract/README.md`:** what the module is, its tags, D6's rules,
   and how a consumer pins it and reads the vectors.
9. **`authcontract/CHANGELOG.md`:** Keep a Changelog form, `## [Unreleased]`
   with the additions.
10. **Tests,** in `authcontract/`:

    | Test | Asserts | Planted break (each must fail it) |
    | :--- | :--- | :--- |
    | `TestJSONNames_0073` | each exported field's JSON name is D2's, by a table of type, field and name, and each list or map field is `omitempty` | `Step.URL`'s tag spelled `uri`; `Method.Links` without `omitempty` |
    | `TestAnswerRedacts_0073` | with `Secret` true and with `Secret` nil, a planted paste `sk-PLANTED-0073` and a planted field value appear in no output of `%v`, `%+v`, `%#v`, `%s`, `%q` on an `Answer` and a `*Answer`, nor in `slog`'s text and JSON handlers; `Seq`, `Confirm`, `Secret` and the field keys do appear | `String` returns `fmt.Sprintf("%#v", struct)` of the raw fields; `LogValue` omitted, so `slog` prints the struct; a nil `Secret` read as false |
    | `TestAnswerShowsNonSecret_0073` | with `Secret` a non-nil false, the paste and field values are printed, so the mask follows the step and not a blanket rule | the mask applied whatever `Secret` says |
    | `TestStepSecretJSON_0073` | a `Step` with `Secret` true encodes `"secret":true`, and one without omits the member; an `Answer`'s nil `Secret` omits it, and false encodes `"secret":false` | `Answer.Secret` declared `bool`, so false is dropped |
    | `TestAnswerJSONCarriesPaste_0073` | `json.Marshal` of an `Answer` holds the paste, and decodes back equal | a `MarshalJSON` added that redacts |
    | `TestUnknownEnumValuesKept_0073` | `Step{Kind: "a_future_kind"}`, `Method{Completion: "a_future_completion"}` and `Failure{Kind: "a_future_kind"}` decode and re-encode with the value kept | an `UnmarshalText` on `StepKind` that refuses an unlisted value |
    | `ExampleAnswer_String` | R50: an answer with a paste prints redacted; `// Output:` shown | the example's output, planted wrong |
    | `TestSessionKindValues_0073` *(added 2026-11-11, the MADR's amendment)* | the `SessionKind` constants that `go/parser` finds in the package's non-test files are exactly the amendment's table, by name and value: `SessionChatGPT` `chatgpt`, `SessionChatGPTPlan` `chatgpt_plan`, `SessionGrok` `grok`, `SessionKiloDevice` `kilo_device` | `SessionKiloDevice` deleted, so a value is missing; `SessionGrok`'s value spelled `xai` |
    | `TestUnknownSessionKindKept_0073` *(added 2026-11-11, the MADR's amendment)* | `"a_future_session_kind"` decodes into a `SessionKind` with no error, and re-encodes with the value kept | an `UnmarshalText` on `SessionKind` that refuses an unlisted value |

Acceptance: `go -C authcontract test -race -count=1 ./...` passes; `go -C
authcontract list -deps ./...` names standard packages and the module's own
only; each plant above failed its test; `make docs-check` passes with the
module in its walk.

### Phase 2: the gates (D7, D8)

1. *(Moved to Phase 1a, step 3: `scripts/go-precheck.sh` learns the module
   before its first Go file.)*
2. **`Makefile`:** `test`, `vet`, `lint` and `vuln` also run in
   `authcontract`; a new `contract-check` target runs `python3 -B
   scripts/check_contract.py`; it is added to `.PHONY`, and its `##` comment
   puts it in `help`'s list, which reads those comments.
3. **`scripts/check_contract.py`** (D7), with its usage doc and exit codes
   as the other gates':
   * the module's dependencies and requirements: a package from `go -C
     authcontract list -deps -test ./...` fails when it is neither standard
     nor under the module's path (MADR D7);
   * `go -C authcontract vet ./...` and `test -race -count=1 ./...`;
   * coverage of each package with statements at 80 % or more;
   * apidiff, at `scripts/check_api.py`'s `APIDIFF` pin (imported, not
     copied), against the latest `authcontract/v*` tag, failing on any
     incompatible change. As `check_api.py` does for the root, it extracts
     `git archive <tag>` into a temporary directory, then writes export data
     with `apidiff -m -w` for the module path from that directory's
     `authcontract/` and from the tree's, and compares them with `apidiff -m
     -incompatible`;
   * the schema comparison against `authcontract/authcontract.schema.json`
     at that tag (D7's five refusals);
   * before any `authcontract/v*` tag, it prints that there is nothing to
     compare, and passes.
4. **`scripts/check_changelog.py`** also checks `authcontract/CHANGELOG.md`
   against `authcontract/v*` tags, by the same rules, with the tag prefix
   stripped for the section's name. Its problems name the full tag
   (`authcontract/v0.1.0 has no section`), so they are told apart from the
   SDK's.
5. **`scripts/test_gates.py`:** `contract-check` joins `CODE_GATES` and
   `GATES`; `WorkflowContractTest` gains the assertion that every gate in
   `CODE_GATES` is in `ci.yml`'s gate line, with the message `ci.yml does not
   run these code gates` (MADR D8; today only the Markdown gates are
   asserted, P9). A test class `ContractCheckTest` runs
   `check_contract.py` on a `copy_tree` copy with each `contract-check` plant
   below, as `GateSelfTest` does; the plants that need a tag tag the copy as
   "Rules for every phase" says. The `changelog-check` plant is a
   `GateSelfTest` case. *(2026-11-11: the code-gate assertion added.)*
6. **`.github/workflows/ci.yml`:**
   * `tags: ['v*', 'authcontract/v*']`;
   * `go -C authcontract test -shuffle=on ./...` on every runner, and `go -C
     authcontract test -race -shuffle=on ./...` on Linux;
   * on Linux, in the step that runs the root's, `go -C authcontract mod
     tidy -diff`, and govulncheck at the root's pin, `go -C authcontract run
     golang.org/x/vuln/cmd/govulncheck@<the pin> ./...` *(2026-11-11: added,
     MADR D8)*;
   * `contract-check` added to the gate list on Linux;
   * the release job is unchanged: its `refs/tags/v` condition already
     excludes `authcontract/` tags.
7. **`tools/gate.py`:** its `CHECKS` list, a flat list of named checks, gains
   `make("contract-check")` after `make("api-check")`. The module's vet and
   tests run inside `contract-check` (step 3), so no other entry is added.
   `tools/test_tools.py`'s `TestGateChecks` (P10) pins the list and gains
   the name in that place, and `tools/README.md`'s "29 checks" becomes 30;
   `make tools-test` passes. *(2026-11-11: the step named a "step-4" and a
   "step-3" of `gate.py`, which has no steps.)*
8. **`docs/architecture.md`:** a section "The auth contract module" in
   prose, under "Tree", naming the module, its tag scheme, its gate, and that
   R2's and the package table's rows are the root module's.

| Check | Planted break, on a scratch clone | Expected failure |
| :--- | :--- | :--- |
| `contract-check`, dependencies | `import _ "golang.org/x/term"` in a new `authcontract/plant.go` | names `golang.org/x/term` |
| `contract-check`, requirements | `require golang.org/x/sys v0.47.0` added to `authcontract/go.mod` | names the requirement |
| `contract-check`, apidiff | the scratch clone tags its tree `authcontract/v0.1.0`, then deletes `StepWaiting` | names `StepWaiting: removed` |
| `contract-check`, apidiff at v0 | the same tag, then `Step.Seq` becomes `int64` | an incompatible change, though the module is at v0 |
| `contract-check`, schema | the same tag, then `Step.URL`'s JSON tag becomes `uri` and the schema is regenerated | names `url` removed from `Step` |
| `contract-check`, schema | the same tag, then `PasteCode` is deleted and the schema regenerated | names the enum value `code` removed |
| `contract-check`, coverage | every test file of `authcontract` but one deleted | names the package and its coverage under 80 % |
| `contract-check`, dependencies admit the module's own | the module-prefix rule removed from the scratch copy of the script (the tree's `go list -deps -test ./...` names the module's own package) | names `github.com/maccavelli/go-llmprovider-sdk/authcontract`; restored, the tree's run passes |
| `changelog-check`, the contract | a scratch tag `authcontract/v0.1.0` with no `## [v0.1.0]` section in `authcontract/CHANGELOG.md` | names `authcontract/v0.1.0 has no section` |
| `gate-selftest`, workflow contract | `contract-check` removed from `ci.yml`'s gate line (a `tools/plant.py` replacement, its command `python3 -B -m unittest scripts/test_gates.py`) | `ci.yml does not run these code gates` names it |

Acceptance: `make contract-check`, `make gate-selftest` and `make
changelog-check` pass on the tree; each plant above failed as expected.

### Phase 3: the schema and the vectors (D4, D5)

1. **`authcontract/internal/schemagen/`:** `main.go` reads the package's
   types by `reflect` (it imports the package) and its constants by
   `go/parser` over the package's `.go` files, grouped by declared type, and
   writes D4's schema with `-output`. Deterministic: `$defs` and enums in
   source order, two-space indent, a trailing newline. It writes `$schema`
   (draft 2020-12) and `$id`, whose last segment is `v` and
   `authcontract.SchemaVersion` (`.../authcontract/schema/v1`); it writes no
   module version (MADR D4). A pointer field gets `"x-nullable": true`; a
   field tagged `omitempty` or `omitzero` is left out of `required`. `main`
   only parses flags and calls `run(args []string, stdout io.Writer) error`,
   which holds the work, so the package's own test can reach its statements.

   *(Added 2026-11-11, the MADR's "Amendment 2026-11-11: a session-kind type and its labels":)* `main.go` also holds one table, value to
   label, for `SessionKind`, and writes it on that type's `$defs` entry as
   `"x-enum-labels"`, an object with one member per `enum` value, in the
   enum's order. No other entry gets one. The entry is written though no
   struct refers to the type, as every enum type found by `go/parser` is.
2. **`authcontract/doc.go`** gains `//go:generate go run ./internal/schemagen -output authcontract.schema.json`.
3. **`authcontract/authcontract.schema.json`:** generated, committed.
4. **`authcontract/internal/schemacheck/`:** a validator for the keywords D4
   lists. It validates by `type`, `properties`, `required`, `enum`, `$ref`
   into `$defs`, `items`, `additionalProperties` and `format: date-time`.
   It accepts and ignores the annotations `$schema`, `$id`, `x-open-enum` and
   `x-nullable` (`$defs` is where `$ref` resolves). Anything else in the
   schema is an error naming the keyword, so the generator cannot emit a
   keyword the validator does not know. *(Added 2026-11-11, the MADR's
   amendment: it accepts and ignores `x-enum-labels` too.)*
5. **`authcontract/testdata/vectors/`:** one file per case. At least:
   `method-` for each `Kind`, among them `method-oauth-at-start.json`
   (`oauth`, `at_start`, no `paste`, `yields` and `client_identity`
   `unknown`, `refreshable` and `revocable` false) and
   `method-host-only-command.json` (`host_only`, `host_command`, a
   `command`); `step-` for each `StepKind`, with `open_url` both with and
   without `accepts`, and `step-paste.json` with `secret` true and no
   `expires_at`; `answer-paste.json` (`secret` true),
   `answer-paste-unknown-secret.json` (no `secret`), `answer-confirm.json`
   (`secret` false, `confirm` true), `answer-confirm-false.json` (`secret`
   false, `confirm` false), `answer-form.json`; `failure-` for `denied`,
   `expired`, `offline`, `other`. `forward/` holds
   `step-unknown-kind.json` and `method-unknown-completion.json`, each with an
   unknown enum value, and `answer-unknown-field.json`, whose only novelty is
   an unknown member; each has a `_meta.example` entry.

   *(Added 2026-11-11, the MADR's "Amendment 2026-11-11: a session-kind type and its labels":)* `session-kind-chatgpt.json`,
   `session-kind-chatgpt-plan.json`, `session-kind-grok.json` and
   `session-kind-kilo-device.json`, each the bare JSON string of its value;
   and `forward/session-kind-unknown.json`, the string
   `"a_future_session_kind"`, which as a bare string has no `_meta.example`
   entry.
6. **Tests:**

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestVectorsRoundTrip_0073` | every vector decodes into the type its name gives and re-encodes to the same compact bytes; `answer-confirm-false.json` keeps `"confirm":false` | a vector's `seq` written as a string; `Step.Code`'s `omitempty` removed; `Answer.Confirm` declared `bool`, so `false` is dropped |
   | `TestVectorsValidate_0073` | every vector outside `forward/` validates against the committed schema, `step-paste.json` with no `expires_at` among them | a vector with `"kind": "open_ur"`; `expires_at` listed in `required` |
   | `TestForwardVectors_0073` | each `forward/` vector decodes without error and drops its unknown member; `step-unknown-kind.json` and `method-unknown-completion.json` keep their unknown enum value and fail strict validation; `answer-unknown-field.json` validates | a forward vector's unknown value replaced by a listed one, so strict validation passes; `additionalProperties` set false in a scratch copy of the schema, so `answer-unknown-field.json` fails |
   | `TestSchemaKeywords_0073` | the schema holds no `oneOf`, `anyOf`, `allOf`, `not`, `if`, `then`, `else`, `description` or `title`, and each enum `$defs` entry has `"x-open-enum": true` | the generator emits `oneOf` for `Step`; it drops `x-open-enum` |
   | `TestSchemaMarksPointers_0073` | exactly the pointer fields (`Step.default`, `Step.error`, `Answer.secret`, `Answer.confirm`) carry `"x-nullable": true`, found by `reflect` | the generator drops `x-nullable` from `Answer.confirm`; it adds it to `Step.secret` |
   | `TestSchemaIDIsSchemaVersion_0073` | the schema's `$id` ends `/v` followed by `SchemaVersion`, and the schema has no `x-module-version` | `SchemaVersion` set to 2 without regenerating |
   | `TestSchemaListsEveryConstant_0073` | each enum `$defs` entry lists exactly the constants of its type found by `go/parser`, `YieldsUnknown` and `ClientUnknown` included | a constant `StepPause` added to `step.go` without regenerating |
   | `TestSchemacheckRefusesUnknownKeywords_0073` | the validator errors on a schema with `patternProperties`, naming it | the validator ignores unknown keywords |
   | `TestSchemacheckAcceptsAnnotations_0073` | a schema carrying `$schema`, `$id`, `x-open-enum` and `x-nullable` validates the vectors, and each annotation is ignored for validation | `x-nullable` dropped from the validator's accepted list |
   | `TestSchemagenMatchesCommitted_0073` (`internal/schemagen/main_test.go`) | `run` writes into a temporary file the bytes of the committed `authcontract.schema.json`; the package's coverage is 80 % or more | `$defs` written in reverse source order; the committed schema's `$defs` reordered by hand |
   | `TestSchemaSessionKindLabels_0073` *(added 2026-11-11, the MADR's amendment)* | the schema has a `$defs` entry `SessionKind`, though no property refers to it, with `"x-open-enum": true` and an `x-enum-labels` object whose members are exactly its `enum` values, each the amendment's label; no other `$defs` entry carries `x-enum-labels` | the generator's table without `grok`; the generator writes the labels as the entry's `description` instead (which `TestSchemaKeywords_0073` also refuses); the generator skips an enum type no struct refers to, so `SessionKind` has no entry |

   And `make generate-check` is seen to fail on a hand edit of
   `authcontract.schema.json` (one enum value removed by hand).

   *(Added 2026-11-11, the MADR's "Amendment 2026-11-11: a session-kind type and its labels": the existing tests take the new type.
   `TestSchemaKeywords_0073` also asserts that `x-enum-labels` appears on
   `SessionKind`'s entry and no other. `TestSchemacheckAcceptsAnnotations_0073`'s
   schema also carries `x-enum-labels`, with a further planted break:
   `x-enum-labels` dropped from the validator's accepted list.
   `TestVectorsRoundTrip_0073`, `TestVectorsValidate_0073` and
   `TestForwardVectors_0073` take step 5's session-kind vectors, a bare
   string decoding into the type its name gives, with a further planted
   break: `forward/session-kind-unknown.json`'s value replaced by `grok`, so
   strict validation passes. `TestSchemaListsEveryConstant_0073` covers
   `SessionKind` as it covers every enum type.)*

Acceptance: the tests pass, each plant failed; `make generate-check` passes
on the tree; `make contract-check` passes.

### Phase 4: docs, changelog and the tag

1. `authcontract/CHANGELOG.md`: the section `## [v0.1.0] - <date>`.
2. `README.md`: one paragraph, "The auth contract module", linking
   `authcontract/README.md`.

   *(Added 2026-11-12, the owner's choice on this phase's deviation, below:)*
   **`scripts/test_gates.py`:** `test_changelog_check_contract_tag_without_section`
   first removes the copy's `## [v0.1.0] - 2026-11-12` heading and its blank
   line with `replace_once`, then tags and asserts as before; and
   `test_changelog_check_contract_wrong_date` changes that heading's date to
   `2001-01-01` instead of inserting a second section. Their assertions are
   unchanged. Step 1's section made the first case pass where it must fail.

   *(Added 2026-11-12, the owner's choice on the CI deviation in this phase's
   entry:)* **`.gitattributes`:** `* text=auto eol=lf`, so a Windows checkout
   holds the committed bytes. The push of this phase's commit failed
   `validate (windows-2025)` on the byte-exact schema test.
3. Run the full gate (`python3 -B tools/gate.py`).
4. On the owner's word: push, then tag `authcontract/v0.1.0` at the commit,
   and push the tag. CI runs on it (Phase 2, step 6).
5. After CI passes: `go list -m
   github.com/maccavelli/go-llmprovider-sdk/authcontract@v0.1.0`, run from a
   scratch directory outside the repository, resolves it through the module
   proxy.

Acceptance: the tag exists and CI passed on it; the module resolves through
the proxy; `make changelog-check` passes with the tag.

## Verification

Each command's exit status is captured before any filter.

* `go -C authcontract test -race -count=1 ./...`
* `go -C authcontract vet ./...`
* `make contract-check` (apidiff for the nested module, at
  `scripts/check_api.py`'s pin, against `authcontract/v*`; the schema
  comparison; dependencies; coverage)
* `make api-check` and `python3 -B tools/apidiff.py v1.29.0`: the root
  module's API is unchanged (no line).
* `make test`, `make lint`, `make vuln`, `go test -race ./...` (the root,
  unchanged)
* `make dep-check tables-check coverage-check generate-check docs-check
  changelog-check records-check links-check disclosure-check gate-selftest`
* `make markdownlint` (the module's `README.md` and `CHANGELOG.md` are not
  records, so markdownlint lints them)
* `python3 -B tools/gate.py`
* `python3 -B tools/claims.py docs/decisions/0073-MADR-publish-an-auth-contract-module.md`
* `python3 -B tools/recordlint.py --changed`, `python3 -B tools/denyscan.py --changed`
* Every plant of Phases 1 to 3, with its failure line quoted in the
  execution record.

## Rollout and Rollback

* **Order across repositories.**
  0. [0077-PLAN-go-1-27-4-for-go-2026-6859.md](0077-PLAN-go-1-27-4-for-go-2026-6859.md)
     raises the root's Go floor to 1.27.4, before any code of this series.
     It has no tag of its own: it ships in 0076's `v1.30.0` (step 4).
     *(Added 2026-11-11, when the owner approved 0077. Phase 1a's `go` line
     follows it.)*
  1. This PLAN's Phase 4 tags `authcontract/v0.1.0`.
  2. 0074-PLAN Phase 4 makes the root module require it, and the SDK minor
     that carries 0074 and 0075 is tagged after.
  3. gobble-cli and magic-cli-remote may pin the contract from step 1 on
     (Magic's codex, kilo and opencode mappers need only the types). gobble
     needs the engine, so it pins the SDK minor of step 2 as well (`v1.31.0`
     if 0076 ships as `v1.30.0`). gobble then tags `acpext/v0.5.0`, whose
     `ErrorKind` holds MADR D3's eighteen words, and its own `v0.6.0`;
     Magic's cut-over waits for both (gobble-cli
     `0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`
     and magic-cli-remote
     `0246-PLAN-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`,
     both proposed). *(2026-11-11: the owner ruled that gobble-cli `v0.6.0`
     is the release that must pin the same acp-go-sdk2 tag as
     magic-cli-remote. Both move from `v1.35.0` to `v1.36.0` together before
     it, each by a record of its own.)* *(2026-11-12: gobble's release on the
     contract, named `v0.6.0` above, is now `v0.7.0`. The owner decided that
     gobble-cli releases `v0.6.0` now, from its main, without the sign-in
     redesign, and that the redesign is `v0.7.0` (gobble-cli
     `docs/decisions/0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md`,
     D1 and D2), so Magic's cut-over waits for `acpext/v0.5.0` and gobble's
     `v0.7.0`. The ruling of 2026-11-11 stays true of `v0.6.0`, which is cut
     on acp-go-sdk2 `v1.36.0` as magic-cli-remote is (0055-MADR F8); `v0.7.0`
     must meet the same lockstep when it is cut (its D2).)*
  4. 0076's SDK release is independent of all three, and comes first: the
     owner decided on 2026-11-10 (DF) that it lands now, ahead of the
     redesign. *(2026-11-11: first after step 0, whose floor it ships.)*
* **Rollout.** Additive: no existing file under `llmprovider/` or `wizard/`
  changes, and no existing API.
* **Rollback before the tag:** revert the phase's commit.
* **Rollback after the tag:** a published tag cannot be withdrawn from the
  module proxy. A wrong type is fixed forward by an additive `v0.1.1` or
  `v0.2.0`, under D6. A tag that must not be used is retracted with a
  `retract v0.1.0 // reason` line in `authcontract/go.mod` and a new tag,
  on the owner's word.

## Code claims

The facts this PLAN's ordering rests on, checked by `tools/claims.py`.

<!-- claims: self=773b84c -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| P1 | The pre-add check vets the given files' packages from the root | `scripts/go-precheck.sh:134` | `if ! vet_out="$(go vet "${pkgs[@]}" 2>&1)"; then` |
| P2 | and tests them from the root | `scripts/go-precheck.sh:139` | `if ! test_out="$(go test -race "${pkgs[@]}" 2>&1)"; then` |
| P3 | A failing file is not committed | `AGENTS.md:142` | `A file that fails is not committed.` |
| P4 | The agent gate denies a commit the script fails | `AGENTS.md:145` | `denies the commit when it fails` |
| P5 | plant.py's default command | `tools/plant.py:45` | `args = ["go", "test", "-count=1"]` |
| P6 | and its packages default to the root's | `tools/plant.py:48` | `return args + list(spec.get("packages", ["./..."]))` |
| P7 | R47 measures each package by its own tests | `docs/guides/api-standards.md:306` | `package is measured by its own tests` |
| P8 | The pre-add check's self-test copies the tree and runs the script on it | `scripts/test_gates.py:111` | `proc = subprocess.run(["bash", "scripts/go-precheck.sh", gone], cwd=tree, capture_output=True, text=True,` |
| P9 | The workflow contract asserts that `ci.yml` runs the Markdown gates, and no such line exists for the code gates | `scripts/test_gates.py:619` | `self.assertEqual([g for g in MARKDOWN_GATES if g not in ci], [], "ci.yml does not run these Markdown gates")` |
| P10 | `tools/gate.py`'s checks are pinned in order | `tools/test_tools.py:421` | `self.assertEqual([c.name for c in gate.CHECKS], want)` |
| P11 | `tools/README.md` counts them | `tools/README.md:16` | `the local gate: 29 checks on a scratch clone of the working tree` |

## Execution record

The owner committed this pair, its index rows and the pairs 0074 to 0077 as
records only (`771b422`), and on 2026-11-11 accepted the MADR and approved
this PLAN.

### Phase 0 (2026-11-11)

* Step 1: done by the owner's commit `771b422`.
* Step 2: `python3 -B tools/records.py status RECORD STATUS --date
  2026-11-11` set the MADR `accepted` and this PLAN `in-progress`, each with
  its row in `docs/README.md`; each run printed "records: written;
  records-check: 0 problem(s)". The MADR's Decision Outcome records the
  acceptance and its date.
* Step 3: the three supersession notes, with no decision rewritten:
  * `0015-MADR-canonical-sdk-api-and-module-layout.md`, "Amendment
    2026-11-11: a second module for a wire contract (0073)", appended after
    its last amendment, quoting D2's bullet and adding the MADR's sentence;
  * `docs/guides/api-standards.md` R1's second sentence, as step 3 words it,
    kept on R1's three lines, so that no later line of the guide moves;
  * `AGENTS.md` "Dependencies", step 3's sentence, after the paragraph's
    last. The file's later lines move down two; this record's claims are
    read at `773b84c`, so none of them moves.
* Step 4: `python3 -B tools/gate.py --markdown` exited 0, started at load
  9.3 after waiting for it to fall below 10: "gate: 11 of 11 passed"
  (parity-check, records-check, tables-check, changelog-check,
  markdownlint, links-check, disclosure-check, gate-selftest, deny-scan,
  recordlint, claims). Each also exited 0: `tools/claims.py` on the MADR,
  "44 of 44 pass", and on this PLAN, "11 of 11 pass"; `make records-check`,
  "records-check: 162 records, 0 problem(s)"; `make links-check`,
  "links-check: 0 problem(s), 1233 relative link(s) in 174 file(s)";
  `tools/recordlint.py --changed`, "recordlint: 8 record(s); no
  rule rose"; `tools/denyscan.py --changed`, "deny-scan: 0 hit(s) in 11
  file(s)"; `tools/linkcheck.py`, "G-links: 0 problem(s)". This entry was
  added after those runs, so the record checks were run again before
  staging, each exiting 0. The changed files were then staged for the
  owner to commit.

### Records, 2026-11-11: the session-kind amendment

* The owner ruled on 2026-11-11, after accepting the MADR, that the contract
  gains a session-kind type with `auth.SessionKind`'s four values and a
  human label for each, before `authcontract/v0.1.0`; the ruling was relayed
  by the cross-repository coordinator. The MADR gains "Amendment
  2026-11-11: a session-kind type and its labels", with dated pointers in
  its Decision Outcome, D2, D4 and D5 and its `consulted` line; it stays
  `accepted`. This PLAN gains, each with a dated note: the file
  `authcontract/sessionkind.go` (Scope, Phase 1b step 7), the R55 row,
  Phase 1b's `TestSessionKindValues_0073` and
  `TestUnknownSessionKindKept_0073`, and Phase 3's label table (step 1),
  validator annotation (step 4), vectors (step 5),
  `TestSchemaSessionKindLabels_0073` and the note on the existing tests.
  It stays `in-progress`. Only Phase 0 has run, so no executed phase is
  reopened; Phase 1a, the next, is unaffected.
* In the same change, 0074-MADR D5 gains a dated note and 0074-PLAN Phase 4
  gains `TestSessionKindsMatchContract_0074`, which pins `auth.SessionKind`
  to the contract's type once the root requires it; `auth.SessionKind`'s
  v1 shape is unchanged.
* The labels' wording is the amendment's; the owner may change any before
  the tag, by a dated note in the MADR.
* Checks, each exit status captured before any filter. `python3 -B
  tools/gate.py --markdown` ran twice and ran nothing either time: "gate:
  the load did not fall below 10 within 1500 s; nothing run", exit 2,
  starting at load 99.1 and then 57.3 on a 10-CPU host. In its place the
  checks it would run that read these records ran directly, each exiting
  0: `tools/claims.py --changed`, "44 of 44 pass" on the MADR, "11 of 11"
  on this PLAN, "55 of 55" on 0074-MADR and "6 of 6" on 0074-PLAN; `make
  records-check`, "records-check: 162 records, 0 problem(s)"; `make
  links-check`, "links-check: 0 problem(s), 1240 relative link(s) in 174
  file(s)"; `tools/recordlint.py --changed`, "recordlint: 4 record(s); no
  rule rose"; `tools/denyscan.py --changed`, "deny-scan: 0 hit(s) in 4
  file(s)"; `make parity-check`, `changelog-check`, `disclosure-check` and
  `tables-check`, each "0 problem(s)" or "0 hit(s)". `markdownlint` (which
  lints no record) and `gate-selftest` were not run; the gate is to be run
  again once the load allows. This entry was added after those runs, so
  the record checks were run again before staging.
* 2026-11-11, later: the owner chose the labels "ChatGPT", "ChatGPT plan",
  "Grok" and "Kilo", written into the MADR's amendment with a dated note.
  On that text `python3 -B tools/gate.py --markdown` ran (it waited from
  load 30.7, started at load 9.9): parity-check, records-check,
  tables-check, changelog-check, markdownlint, links-check,
  disclosure-check, gate-selftest, deny-scan, recordlint and claims each
  rc=0, "gate: 11 of 11 passed", exit 0.

### Phase 1a (2026-11-11)

Base: `main` at `0c65221`, one ahead of `origin/main`, tree clean; the
root's `go.mod` says `go 1.27.4`, so the stop for `go 1.27.3` did not
apply. Go is 1.27.4. No claim of the MADR was found false; no existing
test, gate or golden was changed.

* Step 1: `authcontract/go.mod` holds `module
  github.com/maccavelli/go-llmprovider-sdk/authcontract` and `go 1.27.4`,
  nothing else. No `go.sum`; no Go file.
* Step 2: `.gitignore` gains `go.work` and `go.work.sum`.
* Step 3: `scripts/go-precheck.sh`. It finds each file's module (the nearest
  `go.mod` above it) and, for every nested module a given file belongs to
  (or, with no arguments, every `go.mod` below the root that git lists),
  runs `go -C <module> vet ./...`, `go -C <module> test -race ./...`,
  `golangci-lint run -c ../.golangci.yml ./...` from the module's directory
  (the configuration's path computed from the module's depth), and `go -C
  <module> mod tidy -diff`; its failures are headed "go vet (authcontract):",
  "go test -race (authcontract):", "golangci-lint (authcontract):" and "go
  mod tidy -diff (authcontract):". The root's run is unchanged for a
  root file: same commands, same headings, same final line. Three choices
  the step left open: (1) when every given file is a nested module's, the
  root's lint, vet, test, tidy and govulncheck do not run ("govulncheck:
  skipped (no file of the root module)"), since no file of the root
  changed and `go vet` with no package would vet the root's own directory;
  (2) the nested lint is the host's only, with no `-tags` and no
  `GOOS=windows` pass, as the step words it; (3) a module with no Go file
  still has `go mod tidy -diff` run, so a requirement added to an empty
  module is caught. `bash -n scripts/go-precheck.sh` exit 0; `shellcheck
  scripts/go-precheck.sh` exit 0 with no output, as it was on the previous
  version (the repository does not run shellcheck).
* Step 4: `scripts/test_gates.py`'s `PrecheckTest` gains
  `test_precheck_nested_module_vets_its_file` and
  `test_precheck_nested_module_tidy`, each on a `copy_tree` copy with
  `GO_PRECHECK_SKIP_VULN=1`, through one helper that plants
  `authcontract/plant.go` (and, for the tidy case, appends `require
  golang.org/x/sys v0.47.0` to `authcontract/go.mod`), runs the precheck on
  the new file, and requires exit 1. The vet case also requires the heading
  "go vet (authcontract):", the file name `plant.go` and the vet message
  `arg "x" of wrong type string`; the tidy case, "go mod tidy -diff
  (authcontract):" and the diff line `-require golang.org/x/sys v0.47.0`.
* The MADR's unverified item (D8, `go-precheck.sh` on a nested file), run
  by hand before step 3 on a copy of the tree holding the new `go.mod` and
  the vet plant, with the old script: exit 1, and, verbatim, "go vet:" then
  "main module (github.com/maccavelli/go-llmprovider-sdk) does not contain
  package github.com/maccavelli/go-llmprovider-sdk/authcontract"; "go test
  -race:" then "# ./authcontract" and the same message, "FAIL
  ./authcontract [setup failed]". The run fails on the directory, as the
  table says, and never names `plant.go`.
* New script on the same plant, exit 1: "go vet (authcontract):
  plant.go:5:28: fmt.Printf format %d has arg "x" of wrong type string".
  On the tidy plant, exit 1: "go mod tidy -diff (authcontract):" and, in its
  diff, "-require golang.org/x/sys v0.47.0". The vet, test and lint of that
  clean package passed. Both runs were also made by hand on a scratch copy
  before the cases were written.
* **The two cases seen to fail on the old script.** On a scratch copy of the
  tree with `scripts/go-precheck.sh` put back to its `HEAD` text, `python3
  -B -m unittest` on the two cases: "FF", "FAILED (failures=2)". Both exit 1
  on the old script too, so the exit-code assertion passes, and both fail
  on the text: `AssertionError: 'go vet (authcontract):' not found in
  'go vet:\n  main module (github.com/maccavelli/go-llmprovider-sdk) does
  not contain package github.com/maccavelli/go-llmprovider-sdk/authcontract
  ...'` and `AssertionError: 'go mod tidy -diff (authcontract):' not found
  in 'go vet:\n  main module (...) does not contain package ...'`. On the
  new script, the `PrecheckTest` class: "Ran 3 tests in 13.223s", "OK".
* Checks, each exit status captured before any filter:
  * `make pre-add-check` (no arguments), exit 0: "go-precheck: authcontract:
    no package yet (go vet, go test and golangci-lint skipped).",
    "go-precheck: 917 file(s) clean (gofmt, golangci-lint, go vet, go test,
    govulncheck).", "go-precheck: nested module(s) clean: authcontract."
  * Mixed arguments, a root file and the clean planted nested file, on a
    scratch copy, exit 0: both final lines above. Nested file alone, exit
    0: "govulncheck: skipped (no file of the root module)" and the nested
    line only.
  * `make gate-selftest` (`python3 -B -m unittest scripts/test_gates.py`),
    exit 0: "Ran 51 tests in 193.102s", "OK".
  * `python3 -B tools/gate.py`, exit 0: it waited ("load 50.0; waiting for
    it to fall below 10"), "started at load 9.1", and "gate: 29 of 29
    passed", among them pre-add-check, lint, tidy-diff, dep-check,
    coverage-check, api-check, records-check, tables-check, gate-selftest,
    markdownlint, links-check, disclosure-check, deny-scan and tools-test,
    each rc=0.
  * `ls authcontract`: `go.mod` only.
  * The gate ran before this entry was written, so the checks that read the
    records were run again after it, each exit 0: `tools/claims.py` on the
    MADR, "44 of 44 pass", and on this PLAN, "11 of 11 pass"; `make
    records-check`, "records-check: 162 records, 0 problem(s)"; `make
    links-check`, "links-check: 0 problem(s), 1243 relative link(s) in 174
    file(s)"; `make disclosure-check`, "0 hit(s)"; `tools/recordlint.py
    --changed`, "recordlint: 1 record(s); no rule rose"; `tools/denyscan.py
    --changed`, "deny-scan: 0 hit(s) in 5 file(s)"; `tools/linkcheck.py`,
    "G-links: 0 problem(s)".
* Not done, as the phase says: no `authcontract/*.go` file, no `go.sum`, no
  Makefile target, no `ci.yml` change, no `AGENTS.md` change (its "Pre-add
  checks" paragraph still describes the root's run only; Phase 2 and Phase
  4 are where the PLAN touches docs), no govulncheck of the nested module,
  and no commit. The files are staged for the owner to commit.

### Phase 1b (2026-11-11)

The coordinator committed Phase 1a as `b064c0e`, on the owner's
authorization. Base: `main` at `b064c0e`, two ahead of `origin/main`, tree
clean, `authcontract/` holding only `go.mod` (`go 1.27.4`). No claim of the
MADR was found false; no existing test, gate or golden was changed; no file
under `llmprovider/` or `wizard/` changed.

* Steps 3 to 7: `authcontract/doc.go`, `method.go`, `step.go`, `answer.go`,
  `errorkind.go` and `sessionkind.go`, with D2's types, field names, JSON
  tags and enum spellings, D3's eighteen `Failure*` constants in D3's order,
  and the amendment's `SessionKind` and its four constants, data only, with
  no method. Each exported type's doc says it is a value, safe for
  concurrent use while no goroutine changes it (R51); each enum type's doc
  says it is open; `SessionKind`'s says its labels are in the schema's
  `x-enum-labels`, not in Go.
* Exported names, by `go -C authcontract doc -all .`: `SchemaVersion`; the
  types `Feature`, `Kind`, `Completion`, `Paste`, `Yields`,
  `ClientIdentity`, `StepKind`, `LinkRel`, `FieldType`, `ErrorKind`,
  `SessionKind`, `Method`, `Consent`, `Link`, `Field`, `Step`, `Answer` and
  `Failure`; 67 constants (3, 7, 7, 4, 6, 4, 8, 3, 3, 18 and 4 by type, in
  that order); and `Answer.String`, `Answer.GoString` and
  `Answer.LogValue`. The fields are 18, 4, 3, 6, 16, 7 and 3 for `Method`,
  `Consent`, `Link`, `Field`, `Step`, `Answer` and `Failure`. Each is in the
  R55 table above, and nothing else is exported.
* Choices the steps left open:
  1. The package doc carries D6's six rules in D6's words, with the `**`
     emphasis dropped (a Go doc comment has none), after one sentence that
     names this MADR as the source of the rules' references D2, R34 and
     Q11. `authcontract/README.md` carries them as D6 prints them.
  2. `Answer.String` prints `Answer{SchemaVersion:N Seq:N Secret:S ...}`,
     `Secret` as `true`, `false` or `<nil>`, then `Paste`, `Confirm`,
     `Fields` (keys sorted) and `Meta` when set. A shown value is quoted; a
     masked one is `[redacted]`. `Meta` is printed by its keys only, never
     its values, which the step does not name; this is the conservative
     reading of D6 rule 5. `GoString` prefixes `authcontract.`. `LogValue`
     is a group under the JSON names, `secret` logged as null when nil.
  3. Each constant's meaning follows D2, D3 and the amendment, and, for
     `Kind`, `Completion`, `Yields` and `ClientIdentity`, the method facts
     table of 0074-MADR D1.
* Steps 8 and 9: `authcontract/README.md` (what the module is, its types,
  the four session kinds and their labels, its tags, D6's rules, how a
  consumer pins it, and how it reads the schema and the vectors at a tag)
  and `authcontract/CHANGELOG.md` (Keep a Changelog, `## [Unreleased]`,
  `### Added`).
* Step 10: the tests `TestJSONNames_0073`, `TestStepSecretJSON_0073`,
  `TestAnswerJSONCarriesPaste_0073` and `TestUnknownEnumValuesKept_0073`
  (`json_0073_test.go`); `TestAnswerRedacts_0073` and
  `TestAnswerShowsNonSecret_0073` (`answer_0073_test.go`);
  `TestSessionKindValues_0073` and `TestUnknownSessionKindKept_0073`
  (`sessionkind_0073_test.go`); `ExampleAnswer_String`
  (`example_test.go`). Coverage of the package: 95.9 %.
* **Plants.** Each by `python3 -B tools/plant.py SPEC.json` on a scratch
  clone, each spec's command `["go", "-C", "authcontract", "test",
  "-count=1", "-run", "^<the test>$", "./..."]`; every plant exit 1,
  "plant: all N plant(s) caught" for each spec, run again on the final tree
  with the same result:
  * `TestJSONNames_0073`, `Step.URL`'s tag spelled `uri`:
    `json_0073_test.go:115: Step.URL: tag "uri,omitempty", want
    "url,omitempty"`; `Method.Links` without `omitempty`:
    `json_0073_test.go:115: Method.Links: tag "links", want
    "links,omitempty"` and `json_0073_test.go:118: Method.Links: a slice
    field without omitempty (tag "links")`.
  * `TestAnswerRedacts_0073`, `String` returning `fmt.Sprintf("%#v", …)` of
    the raw fields: `answer_0073_test.go:74: %v of a authcontract.Answer
    shows sk-PLANTED-0073: authcontract.raw{SchemaVersion:1, Seq:7, …
    Paste:"sk-PLANTED-0073", …`; `LogValue` renamed away, so `slog` prints
    the struct: `answer_0073_test.go:74: slog's JSON handler with a
    authcontract.Answer shows sk-PLANTED-0073:
    {…"a":{"schema_version":1,"seq":7,"secret":true,"paste":"sk-PLANTED-0073",…`
    (the text handler prints through `String`, still masked, and fails on
    `a.seq=7` missing); a nil `Secret` read as false:
    `--- FAIL: TestAnswerRedacts_0073/Secret_nil` and
    `answer_0073_test.go:74: %v of a authcontract.Answer shows
    sk-PLANTED-0073: Answer{SchemaVersion:1 Seq:7 Secret:<nil>
    Paste:"sk-PLANTED-0073" …`.
  * `TestAnswerShowsNonSecret_0073`, the mask applied whatever `Secret`
    says: `answer_0073_test.go:98: %v of a authcontract.Answer does not
    show sk-PLANTED-0073: Answer{SchemaVersion:1 Seq:7 Secret:false
    Paste:[redacted] …`.
  * `TestStepSecretJSON_0073`, `Answer.Secret` declared `bool`: the package
    does not build, "FAIL github.com/maccavelli/go-llmprovider-sdk/authcontract
    [build failed]"; on a scratch copy of the module, the build error is
    `./answer.go:33:52: invalid operation: a.Secret == nil (mismatched
    types bool and untyped nil)`. Since that plant fails the build, not the
    assertion, two more plants were run on the same test: `Step.Secret`'s
    tag without `omitempty`, `json_0073_test.go:147: a step that is not
    secret: {"schema_version":1,"seq":2,"kind":"waiting","secret":false},
    want no "secret" member`; and `Answer.Secret`'s tag `json:"-"`,
    `json_0073_test.go:153: an answer whose Secret is false:
    {"schema_version":1,"seq":2,"confirm":false}, want "secret":false`.
  * `TestAnswerJSONCarriesPaste_0073`, a `MarshalJSON` that redacts:
    `json_0073_test.go:170: json.Marshal:
    {"schema_version":1,"seq":4,"secret":true,"paste":"[redacted]",…}, want
    the paste and the field value`.
  * `TestUnknownEnumValuesKept_0073`, an `UnmarshalText` on `StepKind` that
    refuses an unlisted value: `json_0073_test.go:201: decode
    {"schema_version":1,"seq":1,"kind":"a_future_kind"} into
    authcontract.Step: json: unsupported value: unlisted step kind
    a_future_kind`.
  * `ExampleAnswer_String`, the output planted wrong:
    `--- FAIL: ExampleAnswer_String (0.00s)`.
  * `TestSessionKindValues_0073`, `SessionKiloDevice` deleted:
    `sessionkind_0073_test.go:82: SessionKiloDevice: missing; want
    "kilo_device"`; `SessionGrok`'s value spelled `xai`:
    `sessionkind_0073_test.go:84: SessionGrok = "xai", want "grok"`.
  * `TestUnknownSessionKindKept_0073`, an `UnmarshalText` on `SessionKind`
    that refuses an unlisted value: `sessionkind_0073_test.go:100: decode
    "a_future_session_kind": unknown session kind a_future_session_kind`.
* Checks, each exit status captured before any filter:
  * `go -C authcontract test -race -count=1 ./...`, exit 0: "ok
    github.com/maccavelli/go-llmprovider-sdk/authcontract"; no race.
    `go -C authcontract vet ./...`, exit 0.
  * `go -C authcontract list -deps ./...`, exit 0: 79 packages, 78 standard
    and the module's own; packages neither standard nor the module's own:
    0. With `-test`, the only others are the module's test packages. `go -C
    authcontract mod tidy -diff`, exit 0, no output. No `go.sum`; `go.mod`
    unchanged.
  * `gofmt -l authcontract`: no file.
  * `make pre-add-check FILES="<the ten .go files>"`, exit 0:
    "govulncheck: skipped (no file of the root module)", "go-precheck:
    nested module(s) clean: authcontract."
  * `make docs-check`: its first run, exit 2, named two of this phase's
    types, "authcontract/errorkind.go:9 R51 type ErrorKind does not say
    whether it is safe for concurrent use" and the same for `FieldType`:
    their docs broke "concurrent use" across two lines, which the audit
    does not join. Rewrapped; then exit 0, "sdkaudit: 0 violation(s) of
    R50,R51,R52", with the module in its walk.
  * `make markdownlint`, exit 0: 14 files, among them
    `authcontract/README.md` and `authcontract/CHANGELOG.md`, "0 issues".
  * `python3 -B tools/gate.py`: its first run, exit 2, ran nothing, "gate:
    the load did not fall below 10 within 1500 s; nothing run" (it waited
    from load 52.3). Run once more, exit 0: it waited from load 28.9,
    "started at load 9.7", and "gate: 29 of 29 passed", each check rc=0,
    among them pre-add-check ("go-precheck: nested module(s) clean:
    authcontract."), test-race-cover, lint, api-check ("api-check: against
    v1.30.0, 0 incompatible change(s) outside llmprovider/x/"),
    docs-check ("sdkaudit: 0 violation(s) of R50,R51,R52"),
    changelog-check, gate-selftest, markdownlint, links-check,
    disclosure-check, deny-scan ("0 hit(s) in 12 file(s)") and tools-test.
  * The gate ran before this entry was written, so the checks that read
    the records were run again after it, each exit 0: `tools/claims.py` on the
    MADR, "44 of 44 pass", and on this PLAN, "11 of 11 pass"; `make
    records-check`, "records-check: 162 records, 0 problem(s)";
    `tools/recordlint.py --changed`, "recordlint: 1 record(s); no rule
    rose"; `tools/denyscan.py --changed`, "deny-scan: 0 hit(s) in 13
    file(s)"; `tools/linkcheck.py`, "G-links: 0 problem(s), 1247 relative
    link(s) in 176 file(s)"; `make links-check`, the same count; `make
    disclosure-check`, "disclosure-check: 0 hit(s) in 1527 file(s)".
* Not done, as the phase says: no schema, generator, validator or vectors
  (Phase 3); no `//go:generate` line; no Makefile, `ci.yml`,
  `check_changelog.py` or `test_gates.py` change (Phase 2); no root
  requirement on the module (0074); no tag; no commit. The files are staged
  for the coordinator to commit.

### Phase 2 (2026-11-12)

The coordinator committed Phase 1b as `be1ff06`, on the owner's
authorization. Base: `main` at `be1ff06`, four ahead of `origin/main`, tree
clean. No claim of the MADR was found false; no existing test, gate or golden
was changed beyond what Phase 2 names (`TestGateChecks`' list gains the new
name; `check_changelog.py` and `test_gates.py` are step 4's and step 5's); no
file under `llmprovider/` or `wizard/` changed, and no Go file changed.

* Step 3, `scripts/check_contract.py` (D7). It runs, in the module, with
  `GOWORK=off` so that a developer's `go.work` cannot supply what the
  module's own `go.mod` does not, and it collects every problem rather than
  stopping at the first:
  * `go mod edit -json`: any `Require` is a problem ("authcontract/go.mod
    requires golang.org/x/sys v0.47.0; the module requires nothing");
  * `go list -e -deps -test`: a package that is neither `Standard` nor the
    module's path or under it, test decoration removed, is a problem naming
    it. `-e` is what names a missing package (`golang.org/x/term`, with no
    requirement) instead of failing the listing;
  * `go vet ./...`, and `go test -race -count=1 -cover ./...`, whose
    coverage lines are read with `check_coverage.py`'s `COVER` (imported, not
    copied); a package with statements under 80 % is a problem naming it
    ("authcontract is 28.6%, below the 80.0% floor"). There is no floors
    file: no package is exempt;
  * with a latest `authcontract/v*` tag (by version, not by name): apidiff at
    `check_api.py`'s `APIDIFF` (imported), as `check_api.py` runs it, from
    `git archive <tag> authcontract` extracted to a temporary directory and
    from the tree's `authcontract/`, `-m -w` for the module path, then `-m
    -incompatible`; any line is a problem, though the module is at v0;
  * with that tag, the schema comparison: `authcontract/authcontract.schema.json`
    at the tag against the tree's, recursively through `properties`, `items`
    and `additionalProperties`, for D7's five refusals: a `$defs` entry
    removed ("$defs entry Paste removed"), a property removed ("property url
    removed from Step"), an enum value removed ("enum value code removed
    from Paste"), a type or `$ref` changed ("Step.seq: type changed from
    integer to string"), a property newly required ("property url newly
    required in Step"). An addition (an entry, an optional property, an enum
    value) is not a problem. A tag that holds no schema says so and compares
    nothing; a tree that lacks one the tag holds is a problem;
  * with no tag: "no authcontract/v* tag yet, so nothing to compare; the API
    and the schema are held from the first tag on", and the two comparisons
    pass; the first four checks still run.
  Exit codes 0, 1 and 2 as the other gates'. On the tree it exits 0: "authcontract:
  95.9% against 80.0%", the no-tag line, "authcontract, 0 problem(s)".
* Step 2, the `Makefile`: `contract-check` (in `.PHONY`, with a `##` comment,
  so `make help` lists it); `test`, `vet`, `lint` (host and `GOOS=windows`,
  `golangci-lint run -c ../.golangci.yml` from `authcontract`, no build tags,
  as the module has none) and `vuln` (the Makefile's one pin) also run in
  `authcontract`. `make vet`, `make lint` ("0 issues." three times) and `make
  vuln` ("No vulnerabilities found." twice) each exit 0.
* Step 4, `scripts/check_changelog.py`: `check` takes a tag prefix and a file
  label; the nested changelog is checked after the root's by the same rules,
  its sections named for the version and its tags `authcontract/vX.Y.Z`. A
  problem names the full tag and, for the nested file, the file in front
  ("authcontract/CHANGELOG.md: authcontract/v0.1.0 has no section"). The
  root's problems are unchanged. The root's `v*` listing does not match a
  contract tag (`git tag --list v*`), nor does `check_api.py`'s `v1.*`.
* Step 5, `scripts/test_gates.py`: `contract-check` in `GATES` and
  `CODE_GATES`; `WorkflowContractTest`'s test gains the assertion `ci.yml
  does not run these code gates`; `ContractCheckTest` (11 cases, below);
  `tag_tree`, which commits the copy with `git commit-tree`, so that no hook
  runs, moves the copy's `HEAD` to it and tags it there; two `GateSelfTest`
  cases for the nested changelog (a tag with no section, and a section with
  the wrong date).
* Step 6, `.github/workflows/ci.yml`: `tags: ['v*', 'authcontract/v*']`; a
  step `go -C authcontract test -shuffle=on ./...` on every runner; on Linux,
  `go -C authcontract test -race -shuffle=on ./...` in the race step,
  `go -C authcontract run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...` in
  the govulncheck step (the root's pin), `go -C authcontract mod tidy -diff`
  in the vet, gofmt, tidy and lint step (whose `gofmt -l .` and `make lint`
  already reach the module), and `contract-check` after `api-check` in the
  gate line. The release job is unchanged: `refs/tags/v` does not match
  `refs/tags/authcontract/v0.1.0`. `actionlint` v1.7.12, the installed
  binary, on `ci.yml` and `docs.yml`: exit 0, no output. `docs.yml` is
  unchanged: `contract-check` is a code gate, which a Markdown-only change
  does not run.
* Step 7, `tools/gate.py`: `make("contract-check")` after `make("api-check")`;
  `tools/test_tools.py`'s `TestGateChecks` and `tools/README.md` (30
  checks, and the list of the gate's checks) follow. `make tools-test` exit
  0.
* Step 8, `docs/architecture.md`: a section "The auth contract module" under
  "Tree", with a line each in the tree listing for the module and the script,
  and the make-target, CI and self-test lists gaining `contract-check`; R2's
  and the package table's rows stay the root module's, as the section says.
  `make tables-check`: "check-tables: 42 packages, 0 problem(s)".
* **The ten plants.** Plants 1 to 7 and 9 are cases of `scripts/test_gates.py`
  on `copy_tree` copies (a new file, an edit of `go.mod`, a tag), plants 8
  and 10 one replacement in an existing file and so `tools/plant.py` plants as
  well (plant 8 is also a permanent case). Their failure lines, from the
  script's own output on the copies:
  1. dependencies, `import _ "golang.org/x/term"` in a new
     `authcontract/plant.go`: "contract-check: authcontract depends on
     golang.org/x/term, which is neither standard nor under
     github.com/maccavelli/go-llmprovider-sdk/authcontract"; `go vet` and `go
     test` also fail ("no required module provides package").
  2. requirements, `require golang.org/x/sys v0.47.0` in `authcontract/go.mod`:
     "contract-check: authcontract/go.mod requires golang.org/x/sys v0.47.0;
     the module requires nothing".
  3. apidiff, tagged, `StepWaiting` deleted: "contract-check: apidiff, against
     authcontract/v0.1.0: incompatible change: StepWaiting: removed" (and the
     module's own vet and tests fail on the reference to it).
  4. apidiff at v0, `Step.Seq` an `int64`: "contract-check: apidiff, against
     authcontract/v0.1.0: incompatible change: Step.Seq: changed from int to
     int64". Only that line: the module is at v0.
  5. schema, `Step.URL`'s JSON tag `uri` and the schema changed to match:
     "contract-check: schema, against authcontract/v0.1.0: property url
     removed from Step". **apidiff printed nothing for the changed tag** (the
     case asserts it), so with `APIDIFF` at its pin the MADR's unverified item
     is answered: apidiff does not report a changed struct tag, and the schema
     comparison is what catches it. The module's own `TestJSONNames_0073`
     failed too.
  6. schema, `PasteCode` deleted and its value `code` removed from the enum:
     "contract-check: schema, against authcontract/v0.1.0: enum value code
     removed from Paste", with "apidiff … incompatible change: PasteCode:
     removed".
  7. coverage, every test file but `example_test.go` deleted: "contract-check:
     authcontract is 28.6%, below the 80.0% floor".
  8. dependencies admit the module's own, the rule `return path == MODULE or
     path.startswith(MODULE + "/")` replaced by `return False`:
     "contract-check: authcontract depends on
     github.com/maccavelli/go-llmprovider-sdk/authcontract, which is neither
     standard nor under github.com/maccavelli/go-llmprovider-sdk/authcontract";
     by `tools/plant.py` (a wrapper prefixes each output line with the word
     ERROR and a colon, since the tool prints only the lines it recognises):
     "caught (exit 1)", "plant: all 1 plant(s) caught". Restored, the case's second run exits 0, "authcontract, 0
     problem(s)".
  9. changelog-check, a scratch tag `authcontract/v0.1.0` with no section:
     "changelog-check: authcontract/CHANGELOG.md: authcontract/v0.1.0 has no
     section". The extra case, a section dated 2001-01-01: "changelog-check:
     authcontract/CHANGELOG.md: line 12: authcontract/v0.1.0 is dated
     2001-01-01; its tag's commit is 2026-11-12".
  10. gate-selftest, workflow contract, `contract-check` removed from the
     gate line of a scratch `ci.yml`, by `tools/plant.py`, its command
     `python3 -B -m unittest scripts/test_gates.py -k WorkflowContractTest`
     (the class, not the whole file, to keep the plant short): "caught
     (exit 1)"; "AssertionError: Lists differ: ['contract-check'] != []",
     "+ [] : ci.yml does not run these code gates".
  Also seen: a copy whose schema only gains an entry, an optional property
  and an enum value passes (`test_schema_additions_are_allowed`); a clean
  copy with no tag passes and says so (`test_no_tag_passes_and_says_so`);
  `test_clean_copy_passes_every_gate` runs `contract-check` on a clean copy.
* **Deviation noted, for the owner (2026-11-12).** The schema plants of the
  table (5 and 6) say "the schema is regenerated". The generator is Phase 3's:
  at this phase no `authcontract.schema.json` exists. Cases 5 and 6, and the
  three further schema cases, therefore commit a small hand-written schema
  (`PLANT_SCHEMA` in `scripts/test_gates.py`) with the tag and edit that file
  after it, with the Go source edited to match. The comparison reads a JSON
  file whatever wrote it, so the check's behaviour is what the table asks,
  but the generated-schema variants (the generator's output at the tag and
  after the edit) cannot run until Phase 3. They are Phase 3's to add, to
  `ContractCheckTest`, once `go generate` exists; they are not written yet.
  `AGENTS.md`'s sentence listing the gates CI runs does not name
  `contract-check` yet: Phase 2 does not list `AGENTS.md` among its files, so
  it is left for Phase 4's docs, or for the owner's word to add it now.
  *(2026-11-12, the owner's decisions, relayed by the coordinator: the
  hand-written schema cases are accepted for this phase, and Phase 3 adds the
  generator-based variants of plants 5 and 6 to `ContractCheckTest`; the
  `AGENTS.md` sentence is added in Phase 4's docs.)*
* Checks, each exit status captured before any filter:
  * `make contract-check`, exit 0 (above). `make changelog-check`, exit 0,
    "changelog-check: 43 release section(s), 0 problem(s)". `make dep-check`,
    `generate-check`, `docs-check` ("sdkaudit: 0 violation(s) of R50,R51,R52"),
    `links-check` ("0 problem(s), 1247 relative link(s) in 176 file(s)") and
    `markdownlint` ("0 issues") each exit 0.
  * `make gate-selftest`, exit 0: "Ran 64 tests in 274.765s", "OK" (51 before,
    and 13 new: 11 `ContractCheckTest`, 2 `GateSelfTest`).
  * **`python3 -B tools/gate.py` did not run.** Its first run, exit 2: "gate:
    the load did not fall below 10 within 1500 s; nothing run" (it waited from
    load 71.0). Run once more, as the phase allows, with `--timeout 3300`
    (only the wait, not `--max-load`), exit 2: "gate: the load did not fall
    below 10 within 3300 s; nothing run" (it waited from load 67.8; the host's
    one-minute load stayed between 19 and 84 throughout). `python3 -B -c` on
    `gate.CHECKS` counts 30 entries, `contract-check` after `api-check`; `make
    tools-test` (`TestGateChecks` pins the list) exit 0. In the gate's place
    its checks that this phase can affect, and the Python gates that no Go
    change reaches, ran directly, each exit 0: `make contract-check`, `lint`,
    `vet`, `vuln`, `dep-check`, `generate-check`, `docs-check`,
    `tables-check`, `changelog-check`, `parity-check` ("409 identifiers, 409
    rows"), `api-check` ("against v1.30.0, 0 incompatible change(s) outside
    llmprovider/x/"), `fixtures-check` ("0 problem(s)"), `go mod tidy -diff` in
    the root and in `authcontract` (no output), `gate-selftest`, `markdownlint`
    ("0 issues"), `links-check`, `disclosure-check` ("0 hit(s) in 1528
    file(s)") and `tools-test`. Not run, and unaffected because no Go file
    changed: `pre-add-check`, the six `go vet` runs for three platforms, `go
    test` in `test-race-cover`, `test-shuffle` and `test-repeat`, and
    `g-wire-stable`. The full gate is to be run once the load allows; the
    count "30 of 30" is therefore not yet observed.
    *(2026-11-12, later: the coordinator ran `python3 -B tools/gate.py
    --timeout 5400` on the staged tree. It waited from load 30.1, started at
    load 9.1, and exited 0 with "gate: 30 of 30 passed", every check rc=0,
    the six `go vet` runs, `test-race-cover`, `test-shuffle`, `test-repeat`
    and `g-wire-stable` included.)*
  * After this entry was written, each exit 0: `tools/claims.py` on the MADR,
    "44 of 44 pass", and on this PLAN, "11 of 11 pass"; `make records-check`,
    "records-check: 162 records, 0 problem(s)"; `tools/recordlint.py
    --changed`, "recordlint: 1 record(s); no rule rose"; `tools/denyscan.py
    --changed`, "deny-scan: 0 hit(s) in 10 file(s)"; `tools/linkcheck.py`,
    "G-links: 0 problem(s), 1247 relative link(s) in 176 file(s)"; `make
    links-check`, the same count; `make disclosure-check`, "0 hit(s) in 1528
    file(s)".
* Not done, as the phase says: no schema, generator or vectors (Phase 3); no
  `AGENTS.md` change; no root requirement on the module (0074); no tag in the
  tree (the tags of the plants exist only in scratch copies); no commit. The
  files are staged for the coordinator to commit.

### Phase 3 (2026-11-12)

The coordinator committed Phase 2 as `06d1315`, on the owner's authorization, after the full gate passed 30 of 30.
Base: `main` at `06d1315`, four ahead of `origin/main`, tree clean; no
`authcontract/v*` tag. No claim of the MADR was found false (`tools/claims.py`,
"44 of 44 pass"); no file under `llmprovider/` or `wizard/` changed; the
module requires nothing. Two existing self-tests needed a change Phase 3 did
not name: the deviation below. Phase 3 stopped before staging, and resumed
once the owner chose the resolutions the same day.

* Step 1, `authcontract/internal/schemagen/main.go`, standard library only.
  `main` calls `run(args, stdout)`, which parses `-dir` (the package's
  directory, default `.`) and `-output` (standard output when absent). It
  reads the non-test `.go` files of the package in name order with
  `go/parser`: each exported struct type, each exported string type and each
  typed string constant, in source order. Struct types are read by `reflect`
  from a list in the generator; a struct the source declares and the list
  lacks, or the reverse, is an error, as is a constant `go/parser` cannot
  type (an implicit repetition of the line above, or a conversion). Each
  struct is an object entry: properties in field order, `required` the
  fields without `omitempty` or `omitzero`, `"additionalProperties": true`; a
  pointer field's property carries `"x-nullable": true`. `time.Time` is
  `{"type": "string", "format": "date-time"}`, `[]T` an array whose `items`
  is `T`'s schema, `map[string]string` an object whose
  `additionalProperties` is `{"type": "string"}`, and `_meta`'s
  `map[string]json.RawMessage` an object whose `additionalProperties` is `{}`,
  any value. Each string type is `{"type": "string", "enum": [...],
  "x-open-enum": true}`, and `SessionKind`'s carries `x-enum-labels` from the
  generator's one table, in the enum's order. `$schema` is
  `https://json-schema.org/draft/2020-12/schema`; `$id` is
  `https://github.com/maccavelli/go-llmprovider-sdk/authcontract/schema/v1`,
  its last segment `v` and `SchemaVersion`. Two-space indent, trailing
  newline.
* Step 2, `authcontract/doc.go`: `//go:generate go run ./internal/schemagen
  -output authcontract.schema.json`, after the package clause.
* Step 3, `authcontract/authcontract.schema.json`, by `go generate ./...`,
  8492 bytes. `$defs`, in source order: `Answer`, `ErrorKind`, `Failure`,
  `Feature`, `Kind`, `Completion`, `Paste`, `Yields`, `ClientIdentity`,
  `LinkRel`, `FieldType`, `Method`, `Consent`, `Link`, `Field`,
  `SessionKind`, `StepKind`, `Step`: seven objects and eleven enums. Each
  enum carries `x-open-enum`; `x-nullable` is on `Step.default`,
  `Step.error`, `Answer.secret` and `Answer.confirm` and nowhere else;
  `x-enum-labels` is on `SessionKind` alone, `chatgpt` "ChatGPT",
  `chatgpt_plan` "ChatGPT plan", `grok` "Grok", `kilo_device` "Kilo". No
  struct refers to `SessionKind`. The keywords used are D4's list exactly.
* Step 4, `authcontract/internal/schemacheck/schemacheck.go`: `Parse` and
  `(*Schema).Validate(def, instance)`. `$schema`, `$id` and `$defs` are
  accepted at the root only; `x-open-enum`, `x-nullable` and `x-enum-labels`
  anywhere, ignored for validation; any other keyword is an error naming it
  and its place. It validates `type` (the seven JSON types), `properties`,
  `required`, `enum` (by value), `$ref` (`#/$defs/Name` only), `items`,
  `additionalProperties` (a boolean or a schema) and `format` (`date-time`
  only, by `time.Parse` with RFC 3339). An error names the keyword and the
  value's place as a JSON pointer.
* Step 5, `authcontract/testdata/vectors/`, 43 files, written with Go's
  encoder (`json.MarshalIndent`, two spaces) by a scratch program that is not
  committed: `method-` `api-key`, `token-paste`, `device-code`, `browser`
  (with `paste`, `first_paste`, `slot`, `features`, a consent with a link,
  and a link), `oauth-at-start`, `vendor-import` and `host-only-command`;
  `step-` `open-url` (no `accepts`), `open-url-paste` (`accepts`, an
  `expires_at`, and an `&` in its URL, written `&` as Go writes it),
  `show-code`, `paste` (`secret` true, no `expires_at`), `confirm` (a
  `default` of false), `form` (three fields, one secret, so `secret` true),
  `waiting`, `done` and `failed`; `answer-paste`, `answer-paste-unknown-secret`,
  `answer-confirm`, `answer-confirm-false`, `answer-form`; a `failure-` vector
  for each of D3's eighteen kinds (D5 asks one per failure kind; step 5's
  `denied`, `expired`, `offline` and `other` are among them); and the four
  `session-kind-` vectors, bare strings. `forward/`, by hand:
  `step-unknown-kind.json`, `method-unknown-completion.json` and
  `answer-unknown-field.json`, each with a `_meta.example` entry, and
  `session-kind-unknown.json`, `"a_future_session_kind"`, with none.
  `scripts/check_fixtures.py` on the 47 files: "0 hit(s), 0 problem(s)".
* Step 6, the tests: `TestVectorsRoundTrip_0073`, `TestVectorsValidate_0073`
  and `TestForwardVectors_0073` (`authcontract/vectors_0073_test.go`);
  `TestSchemagenMatchesCommitted_0073` (`internal/schemagen/main_test.go`);
  `TestSchemaKeywords_0073`, `TestSchemaMarksPointers_0073`,
  `TestSchemaIDIsSchemaVersion_0073`, `TestSchemaListsEveryConstant_0073`
  and `TestSchemaSessionKindLabels_0073`
  (`internal/schemagen/schema_0073_test.go`);
  `TestSchemacheckRefusesUnknownKeywords_0073` and
  `TestSchemacheckAcceptsAnnotations_0073`
  (`internal/schemacheck/schemacheck_test.go`). Coverage: `authcontract`
  95.9 %, `internal/schemacheck` 99.4 %, `internal/schemagen` 87.9 %.
* Choices the steps left open:
  1. The five schema tests live in `internal/schemagen`, and each runs twice,
     as the subtests `committed` (the file) and `generated` (`run` into a
     temporary file). A plant in the generator then fails its own test under
     the phase's command form, with no regeneration, and a plant "without
     regenerating" fails on the committed file.
  2. Two tests beyond the table, for the error paths that hold the two
     internal packages over 80 %, each planted:
     `TestSchemacheckRules_0073` (each keyword's rule, and each malformed
     schema) and `TestSchemagenRefuses_0073` (each form the generator
     refuses).
  3. "The package's coverage is 80 % or more" is held by `contract-check`'s
     coverage line, not inside `TestSchemagenMatchesCommitted_0073`. A
     `TestMain` that read `testing.Coverage()` after the run printed 59.6 %
     where `go test -cover` printed 87.9 % (its documentation: "not a
     replacement for the reports generated by 'go test -cover'"), so it was
     dropped.
  4. `TestForwardVectors_0073` holds a table of the four forward vectors (an
     unknown enum value, or an unknown member) and requires `forward/` to
     hold exactly them; each re-encodes to the vector less its unknown
     member, and each refused one is refused on `enum`.
* **The plants.** Each by `python3 -B tools/plant.py SPEC.json` on a scratch
  clone, each spec's command `["go", "-C", "authcontract", "test",
  "-count=1", "-run", "^<the test>$", "./..."]`; every spec exit 0, "plant:
  all N plant(s) caught", run again on the final tree with the same result:
  * `TestVectorsRoundTrip_0073`: a vector's `seq` written as a string,
    `vectors_0073_test.go:107: step-waiting.json: decode into
    *authcontract.Step: json: cannot unmarshal string into Go struct field
    Step.seq of type int`; `Step.Code`'s `omitempty` removed,
    `vectors_0073_test.go:116: step-done.json: re-encoded
    {"schema_version":1,"seq":4,"kind":"done","code":"","yields":"oauth_session"}`
    and the same for each step vector without a code; `Answer.Confirm`
    declared `bool`, "FAIL github.com/maccavelli/go-llmprovider-sdk/authcontract
    [build failed]", on a scratch copy of the module `./answer.go:53:18:
    invalid operation: a.Confirm != nil (mismatched types bool and untyped
    nil)`. Since that plant fails the build, one more on the same test:
    `Answer.Confirm`'s tag `json:"-"`, `vectors_0073_test.go:116:
    answer-confirm-false.json: re-encoded
    {"schema_version":1,"seq":1,"secret":false}` and
    `vectors_0073_test.go:124: answer-confirm-false.json: Confirm decoded as
    <nil>, want a non-nil false`.
  * `TestVectorsValidate_0073`: `"kind": "open_ur"`,
    `vectors_0073_test.go:140: step-open-url.json: schemacheck: #/kind: enum:
    "open_ur" is not listed`; `expires_at` added to `Step`'s `required` in
    the schema, `vectors_0073_test.go:140: step-paste.json: schemacheck: #:
    required: expires_at is missing`, and the same for each step vector
    without one.
  * `TestForwardVectors_0073`: `step-unknown-kind.json`'s kind made
    `waiting`, `vectors_0073_test.go:214: step-unknown-kind.json: passed
    strict validation; its unknown enum value must fail it`; `Answer`'s
    `additionalProperties` made false in the schema,
    `vectors_0073_test.go:212: answer-unknown-field.json: schemacheck: #:
    additionalProperties: a_future_member is not allowed; a vector whose only
    novelty is an unknown member validates`; `session-kind-unknown.json`'s
    value made `grok`, `vectors_0073_test.go:214: session-kind-unknown.json:
    passed strict validation; its unknown enum value must fail it`.
  * `TestSchemaKeywords_0073`: the generator emits `oneOf` for `Step`,
    `--- FAIL: TestSchemaKeywords_0073/generated` and
    `schema_0073_test.go:109: #/$defs/Step: oneOf, which D4 refuses`; it
    drops `x-open-enum`, `schema_0073_test.go:120: $defs entry Yields: an
    enum without "x-open-enum": true` and the same for the other ten; it
    writes the labels as `description`, `schema_0073_test.go:109:
    #/$defs/SessionKind: description, which D4 refuses` and
    `schema_0073_test.go:124: x-enum-labels at [], want at
    [#/$defs/SessionKind] only`; and, for the amendment's assertion, it
    writes `x-enum-labels` on every enum, `schema_0073_test.go:124:
    x-enum-labels at [#/$defs/ClientIdentity #/$defs/Completion …], want at
    [#/$defs/SessionKind] only`.
  * `TestSchemaMarksPointers_0073`: `x-nullable` dropped from
    `Answer.confirm`, `schema_0073_test.go:177: x-nullable on [Answer.secret
    Step.default Step.error], want exactly the pointer fields [Answer.confirm
    Answer.secret Step.default Step.error]`; added to `Step.secret`,
    `schema_0073_test.go:177: x-nullable on [Answer.confirm Answer.secret
    Step.default Step.error Step.secret], want exactly the pointer fields
    […]`.
  * `TestSchemaIDIsSchemaVersion_0073`: `SchemaVersion` set to 2 without
    regenerating, `--- FAIL: TestSchemaIDIsSchemaVersion_0073/committed` and
    `schema_0073_test.go:196: $id
    "https://github.com/maccavelli/go-llmprovider-sdk/authcontract/schema/v1":
    last segment "v1", want "v2" (SchemaVersion)`.
  * `TestSchemaListsEveryConstant_0073`: `StepPause` added to `step.go`
    without regenerating, `schema_0073_test.go:279: $defs entry StepKind
    lists ["open_url" … "failed"], want its constants ["open_url" … "failed"
    "pause"]`.
  * `TestSchemacheckRefusesUnknownKeywords_0073`: the validator ignores
    unknown keywords, `schemacheck_test.go:35: Parse({"$defs": {"A":
    {"type": "object", "patternProperties": …}}}) = <nil>, want an error
    naming "patternProperties"`, and the same for `oneOf`, `description` and
    `$id` below the root.
  * `TestSchemacheckAcceptsAnnotations_0073`: `x-nullable` dropped from the
    accepted list, `schemacheck_test.go:71: Parse: schemacheck:
    #/$defs/Thing/properties/next: unknown keyword "x-nullable"`;
    `x-enum-labels` dropped, `schemacheck_test.go:71: Parse: schemacheck:
    #/$defs/Kind: unknown keyword "x-enum-labels"`.
  * `TestSchemacheckRules_0073`: `required` ignored, `schemacheck_test.go:157:
    Obj {}: <nil>, want an error holding "#: required: n is missing"`.
  * `TestSchemagenMatchesCommitted_0073`: `$defs` written in reverse source
    order (`slices.Backward`), and, apart, the committed schema's `Feature`
    and `Kind` entries swapped by hand: each `main_test.go:44: run wrote 8492
    bytes that differ from the committed ../../authcontract.schema.json (8492
    bytes); run go generate`.
  * `TestSchemaSessionKindLabels_0073`: the table without `grok`,
    `schema_0073_test.go:330: SessionKind's x-enum-labels has members
    [chatgpt chatgpt_plan kilo_device], want exactly its enum values [chatgpt
    chatgpt_plan grok kilo_device]`; the labels as `description`,
    `schema_0073_test.go:327: SessionKind: no x-enum-labels object (got
    <nil>)`; the generator skips `SessionKind`, the one enum type no struct
    refers to (a `continue` for that name), `schema_0073_test.go:313: no
    $defs entry SessionKind`; `x-enum-labels` on every enum,
    `schema_0073_test.go:339: $defs entry LinkRel carries x-enum-labels; only
    SessionKind's does`.
  * `TestSchemagenRefuses_0073`: an extra argument accepted,
    `main_test.go:114: run(["extra"]) = main.go:269:2: structType's value is
    not a string literal, want an error holding "unexpected argument
    \"extra\""`.
  * `make generate-check`, its command `python3 -B
    scripts/check_generated.py`, one enum value (`other`) removed from the
    schema by hand: "caught (exit 1)"; on a scratch copy, the line
    "generate-check: authcontract/authcontract.schema.json differs from what
    its //go:generate line produces; run go generate".
* **The two generator-based `ContractCheckTest` cases** (the owner's decision
  of 2026-11-12 on Phase 2's plants 5 and 6), in `scripts/test_gates.py`:
  `tag_generated` requires the copy's schema to be what `go generate` writes,
  then tags the copy; `regenerate` runs `go -C authcontract generate ./...`
  in the copy. `test_schema_property_removed_generated`: `Step.URL`'s tag
  `uri`, regenerated (the schema's `Step` then has `uri` and no `url`):
  "contract-check: schema, against authcontract/v0.1.0: property url removed
  from Step", apidiff silent (asserted), and "go test -race failed" (the
  module's own tests). `test_schema_enum_value_removed_generated`:
  `PasteCode` deleted, regenerated (`Paste`'s enum `["", "url_or_code",
  "url_only"]`): "contract-check: apidiff, against authcontract/v0.1.0:
  incompatible change: PasteCode: removed" and "contract-check: schema,
  against authcontract/v0.1.0: enum value code removed from Paste". Both
  cases pass on the tree ("Ran 2 tests", "OK"). Seen to fail: by
  `tools/plant.py`, `contract-check`'s schema comparison not run
  (`problems += apidiff(tag) + schema(tag)` made `problems +=
  apidiff(tag)`), its command the two cases: "caught (exit 1)",
  `AssertionError: 'schema, against authcontract/v0.1.0: property url removed
  from Step' not found in …` and the same for `enum value code removed from
  Paste`. `PLANT_SCHEMA`'s comment no longer says it stands in until the
  generator; the hand-written cases are unchanged.
* **Deviation (2026-11-12): two existing self-tests fail, and Phase 3 does
  not name their change. Stopped, with nothing staged, until the owner
  chose (below).** `make gate-selftest` on
  this tree, exit 2: "Ran 66 tests in 363.659s", "FAILED (failures=2)", the
  two cases below; the other 64 pass, the two new generator-based cases
  among them. Both passed at `06d1315` (Phase 2's gate-selftest, "Ran 64
  tests", "OK", and the full gate's 30 of 30). Both fail because of what
  MADR D4 and this phase require, not because of a defect in the phase's
  code.
  * Evidence, the first. `scripts/test_gates.py:217-226`,
    `GateSelfTest.test_generate_check_no_output`, strips `-output` from the
    `//go:generate` lines of `ownerperm` and `filelock` ("Every package that
    generates code loses it") and requires `generate-check` to exit 1 with "0
    generated files". With step 2's line in `authcontract/doc.go` the copy
    still holds one directive with `-output`, so `generate-check` passes:
    `AssertionError: 0 != 1 : generate-check exit 0, want 1:` and
    "generate-check: 1 generated file(s), 0 problem(s)" (`python3 -B -m
    unittest scripts.test_gates.GateSelfTest.test_generate_check_no_output`,
    exit 1). Step 2's line causes it.
  * Evidence, the second. `ContractCheckTest.test_schema_additions_are_allowed`
    (`scripts/test_gates.py:684-694` at `06d1315`, `:724-734` in the tree)
    writes Phase 2's hand-written `PLANT_SCHEMA` over
    `authcontract/authcontract.schema.json`, tags, grows it (an entry, an
    optional property, an enum value) and requires `contract-check` to exit
    0. The schema comparison still reports nothing, but the file is now the
    committed schema that Phase 3's own tests read, so the module's tests
    fail on the hand-written one: `AssertionError: 1 != 0 : contract-check
    exit 1, want 0:`, "contract-check: go test -race failed:", among its
    lines `main_test.go:44: run wrote 8492 bytes that differ from the
    committed ../../authcontract.schema.json (657 bytes); run go generate`
    and `schemacheck_test.go:116: session-kind-grok.json: schemacheck: no
    $defs entry SessionKind`. Steps 3 and 6 cause it. Phase 2's other
    hand-written cases (`test_schema_property_removed`,
    `test_schema_enum_value_removed`, `test_schema_other_breaks`) want exit
    1 and their lines, and still pass.
  * Resolutions, each leaving the tree correct:
    1. *Recommended, for the first.* Amend Phase 3 to name the change:
       `test_generate_check_no_output` also strips `-output
       authcontract.schema.json` from `authcontract/doc.go`'s line, and its
       docstring names the module (0073). Cost: a few lines of one existing
       test; its assertion and its meaning ("a directive that lost its
       -output checks nothing, which fails") are unchanged. Alternative: the
       test strips `-output` from every directive `scripts/check_generated.py`
       finds, so a later generator needs no edit; a slightly larger change,
       and the test no longer names the packages it covers.
    2. *Recommended, for the second.* Amend Phase 3 to name the change: the
       case runs on the generated schema, as the two new `_generated` cases
       do (`tag_generated`), adds what the module's code can add while its
       own tests pass, a constant to an enum type and a new string type with
       one constant, regenerates, and requires exit 0; the leg that no Go
       change can add without failing `TestJSONNames_0073` (whose table is
       D2's), a new optional property and a new object entry, is asserted by
       calling `check_contract.schema_breaks` on `PLANT_SCHEMA` and its grown
       copy and requiring no problem. Cost: one existing case rewritten in
       two parts; what it asserts (additions are not breaks) is unchanged,
       and the end-to-end leg is stronger, since it is the generator's
       output. Alternative: the case asserts only `schema_breaks` directly,
       dropping its end-to-end run; smaller, and weaker.
  * Not resolutions: moving the line out of `authcontract/doc.go`, or
    writing it without `-output`, would contradict D4 and step 2 and leave
    the schema unchecked by `generate-check`; letting the additions case
    accept any exit code, or dropping its exit-0 assertion, loosens it.
  * Doing nothing: `make gate-selftest` fails, and with it `tools/gate.py`
    and CI's gate line; Phase 3 cannot be staged.
  * **Resolution (2026-11-12): the owner chose,** relayed by the
    coordinator: (1) `test_generate_check_no_output` strips `-output` from
    every `//go:generate` directive generate-check finds, with the same
    assertion (the first resolution's alternative); (2)
    `test_schema_additions_are_allowed` is split in two, an end-to-end run
    on the generated schema and a direct `check_contract.schema_breaks`
    call (the second resolution as recommended). Both are now Phase 3's, in
    `scripts/test_gates.py`:
    1. `test_generate_check_no_output` lists the copy's Go files as the gate
       does (`git ls-files -co --exclude-standard *.go`), removes `-output`
       and its file, in either form (`OUTPUT_FLAG`), from every
       `//go:generate` line, asserts that `ownerperm`, `filelock` and
       `authcontract` each lost one, and requires exit 1 with "0 generated
       files". Seen to fail, by `tools/plant.py`, generate-check's `if
       checked == 0:` made `if checked == -1:`: "caught (exit 1)",
       `AssertionError: 0 != 1 : generate-check exit 0, want 1:`.
    2. `test_schema_additions_are_allowed` tags the generated schema
       (`tag_generated`), adds `StepPlanted` to `StepKind` and a new string
       type `Planted` with one constant, regenerates, asserts both reached
       the schema, and requires exit 0 and "authcontract, against
       authcontract/v0.1.0, 0 problem(s)".
       `SchemaBreaksTest.test_schema_breaks_ignore_additions` calls
       `schema_breaks` on `PLANT_SCHEMA` and its copy grown by an entry, an
       optional property and an enum value, and requires no problem. Seen
       to fail, by `tools/plant.py` in `scripts/check_contract.py`, its
       command the two cases, each "caught (exit 1)": an added enum value
       reported as removed (the comparison's `was` and `now` swapped), both
       cases, "contract-check: schema, against authcontract/v0.1.0: enum
       value planted removed from StepKind" and "AssertionError: Lists
       differ: ['enum value a_new_value removed from Paste'] != []"; an
       added `$defs` entry reported, both cases, "contract-check: schema,
       against authcontract/v0.1.0: $defs entry Planted added" and
       "AssertionError: Lists differ: ['$defs entry Extra added'] != []"; an
       added property reported, the direct case, "AssertionError: Lists
       differ: ['property more added to Step'] != []".
* Checks, each exit status captured before any filter:
  * `go -C authcontract test -race -count=1 ./...`, exit 0: three packages
    "ok"; no race. `go -C authcontract vet ./...`, exit 0. `gofmt -l
    authcontract`: no file.
  * `make pre-add-check FILES="<the seven .go files>"`, exit 0:
    "govulncheck: skipped (no file of the root module)", "go-precheck:
    nested module(s) clean: authcontract."
  * `make generate-check`, exit 0: "generate-check: 3 generated file(s), 0
    problem(s)". `make contract-check`, exit 0: the three packages' coverage
    lines, "no authcontract/v* tag yet, so nothing to compare; the API and the
    schema are held from the first tag on", "authcontract, 0 problem(s)".
  * `make docs-check`, exit 0, "sdkaudit: 0 violation(s) of R50,R51,R52";
    `make fixtures-check`, exit 0, "0 problem(s)"; `make disclosure-check`,
    exit 0, "0 hit(s) in 1582 file(s)".
  * Before the resolution, `make gate-selftest`, exit 2: the deviation's two
    failures, 64 of 66 passing. After it, exit 0: "Ran 67 tests in
    336.754s", "OK" (66, and `SchemaBreaksTest`'s one).
  * `python3 -B tools/gate.py --timeout 5400`, after the resolution, exit 0:
    it waited ("load 61.1; waiting for it to fall below 10"), "started at
    load 9.7", and "gate: 30 of 30 passed", each check rc=0, among them
    pre-add-check, the six `go vet` runs, test-race-cover, test-shuffle,
    test-repeat, lint, api-check, contract-check, generate-check,
    docs-check, gate-selftest, markdownlint, g-wire-stable, links-check,
    disclosure-check, deny-scan and tools-test.
  * The gate ran before this entry's last edits, so the record checks ran
    again after them.
  * After this entry was written, each exit 0: `tools/claims.py` on the MADR,
    "44 of 44 pass", and on this PLAN, "11 of 11 pass"; `make
    records-check`, "records-check: 162 records, 0 problem(s)";
    `tools/recordlint.py --changed`, "no rule rose"; `tools/denyscan.py
    --changed`, "0 hit(s)"; `tools/linkcheck.py`, "G-links: 0 problem(s)";
    `make disclosure-check`, "0 hit(s)".
* Not done, as the phase says: no `AGENTS.md`, README or CHANGELOG change
  (Phase 4); no root requirement on the module (0074); no tag; no commit.
  The files are staged for the coordinator to commit.

### Phase 4, prepared (2026-11-12)

The coordinator committed Phase 3 as `1ba0840`, and the v0.7.0 notes as `c384136`, on the owner's authorization.
Base: `main` at `c384136`, six ahead of `origin/main`, tree clean; no
`authcontract/v*` tag. No file under `llmprovider/` or `wizard/` changed, no
Go file changed, and no existing test, gate or golden changed but the two
self-test cases the owner's resolution of the deviation below names. This
entry covers steps 1 to 3. Steps 4 and 5, the tag and the proxy check, are the
owner's, and are listed at its end.

* Step 1, `authcontract/CHANGELOG.md`: `## [v0.1.0] - 2026-11-12` under an
  empty `## [Unreleased]`, as the root's changelog keeps one. Its `### Added`
  holds the three bullets Phase 1b wrote and two more for what Phase 3 added
  and the changelog did not yet name: `authcontract.schema.json` (D4) and the
  golden vectors (D5). The date is the day the coordinator is to commit this
  phase: `changelog-check` compares it with the tag's commit's committer date
  (`git log -1 --format=%cs`), so a commit made on another day needs the
  section's date changed to match before the tag.
* Step 2, `README.md`: a section "The auth contract module", one paragraph,
  before "Status": the module's path, that it is standard library only, what
  it holds, its tags from `authcontract/v0.1.0`, that it only adds within a
  major version, that the root does not require it yet, and a link to
  `authcontract/README.md`.
* The docs the earlier phases left for this one:
  * `AGENTS.md`, "Pre-add checks": the sentence on the gates CI runs names
    `contract-check`, after `api-check`, as `ci.yml`'s gate line has it (the
    owner's decision of 2026-11-12, in Phase 2's entry). A new paragraph says
    how `scripts/go-precheck.sh` checks a file of the nested module, which
    Phase 1a's "Not done" left for Phase 4: `go -C authcontract vet ./...`
    and `test -race ./...`, `golangci-lint run -c ../.golangci.yml ./...`
    from its directory, for the host, `go mod tidy -diff` there, `gofmt` on
    every file, every module with no `FILES`, and no root lint, vet, test,
    tidy or govulncheck when every given file is the module's.
  * `docs/architecture.md`: "The auth contract module" gains a paragraph on
    the schema, its generator and `generate-check`, and the vectors and their
    validator, which Phase 3 added and the section did not name (R43, the
    layout); the `scripts/go-precheck.sh` bullet under "Tooling" gains the
    nested module's run, the same fact as `AGENTS.md`'s.
  * `authcontract/README.md` needed no change: it already names the first
    tag, `go get …@v0.1.0` and the schema and vectors at a tag.
* **Deviation (2026-11-12): an existing self-test fails, and Phase 4 does
  not name its change. Stopped, with nothing staged, until the owner
  chose (below).** `python3 -B tools/gate.py --timeout 5400`, exit 1: it waited
  ("load 41.5; waiting for it to fall below 10"), "started at load 9.4", and
  "gate: 29 of 30 passed"; every check rc=0 but `gate-selftest`, rc=2:
  "Ran 67 tests in 336.947s", "FAILED (failures=1)".
  * Evidence. `scripts/test_gates.py:399-403`,
    `GateSelfTest.test_changelog_check_contract_tag_without_section`
    (Phase 2's plant 9), tags the copy `authcontract/v0.1.0` and requires
    `changelog-check` to fail with "authcontract/v0.1.0 has no section".
    Step 1 gives the copy that section, so the check passes: `AssertionError:
    0 != 1 : changelog-check exit 0, want 1:` and "changelog-check: 44
    release section(s), 0 problem(s)". Run alone on the tree (the case works
    on a copy), exit 1, the same lines. Not pre-existing: on a scratch clone
    of `c384136` the case passes ("OK", exit 0), as the Phase 3 gate's 30 of
    30 did. Step 1 causes it; the gate is right to fail.
  * The case beside it, `test_changelog_check_contract_wrong_date`
    (`:405-411`), still passes, for a reason that changed: it inserts a
    second `## [v0.1.0] - 2001-01-01` heading after `## [Unreleased]`, so
    the copy now holds two v0.1.0 sections, and the check reports the date
    and "has a section already". Its assertion (the date line) still holds.
  * Resolutions, each leaving the tree correct:
    1. *Recommended.* Amend Phase 4 to name `scripts/test_gates.py`: the
       tag-without-section case first removes the copy's
       `## [v0.1.0] - 2026-11-12` heading and its blank line
       (`replace_once`), then tags and asserts as now; and the wrong-date
       case changes that heading's date to `2001-01-01` instead of inserting
       a second section, so each case again plants one thing. The
       assertions are unchanged. Cost: two lines in two existing cases and a
       docstring sentence; the Phase 4 commit is then no longer Markdown
       only, so its push to `main` runs `ci.yml` in full rather than
       `docs.yml`. Tried on the scratch clone of `c384136` with this
       phase's `authcontract/CHANGELOG.md` copied in: both cases "OK", exit
       0; with `check_changelog.py`'s "has no section" line planted away
       there, the amended case fails, `AssertionError: 0 != 1 :
       changelog-check exit 0, want 1:`, exit 1. Nothing of it is in the
       tree.
    2. The tag-without-section case tags a version no section names, as the
       root's case tags `v1.99.0` (for example `authcontract/v0.99.0`, with
       its assertion's tag), and the wrong-date case as in 1. Cost: the same
       size; the copy then also reports "authcontract/v0.1.0 has a section
       and no tag", which the case does not assert. It does not depend on
       step 1's date.
  * Not resolutions: leaving step 1's section under `## [Unreleased]` until
    after the tag (then `changelog-check` fails on the tag, in CI's run on
    it); deleting the case, or letting it accept exit 0, which loosens it.
  * Doing nothing: `make gate-selftest` fails, and with it `tools/gate.py`,
    `docs.yml` on the push to `main` (it runs `gate-selftest`) and `ci.yml`
    on the tag; Phase 4 cannot be staged or tagged.
  * **Resolution (2026-11-12): the owner chose (1),** relayed by the
    coordinator, and Phase 4 gains `scripts/test_gates.py`, with a dated
    note under its step 2. The change is the one tried on the scratch
    clone: `test_changelog_check_contract_tag_without_section` removes the
    copy's `## [v0.1.0] - 2026-11-12` heading and its blank line by
    `replace_once` before it tags, its docstring saying so;
    `test_changelog_check_contract_wrong_date` replaces that heading with
    `## [v0.1.0] - 2001-01-01`. The assertions are unchanged.
    * Seen on scratch clones of `c384136` with this phase's
      `authcontract/CHANGELOG.md` and `scripts/test_gates.py` copied in,
      never in the tree, the two cases each run: unplanted, "Ran 2 tests",
      "OK", exit 0. With `check_changelog.py`'s `has no section` problem
      replaced by `pass`: `FAIL:
      test_changelog_check_contract_tag_without_section`, `AssertionError:
      0 != 1 : changelog-check exit 0, want 1:`, "changelog-check: 43
      release section(s), 0 problem(s)", exit 1. With its date comparison
      `if date != want:` made `if False:`: `FAIL:
      test_changelog_check_contract_wrong_date`, `AssertionError: 0 != 1 :
      changelog-check exit 0, want 1:`, "changelog-check: 44 release
      section(s), 0 problem(s)", exit 1. The second shows the wrong-date
      case now fails on the date alone; before the change its copy also
      held a second v0.1.0 section, which failed the check by itself.
    * After the change, each exit status captured before any filter: `make
      gate-selftest`, exit 0, "Ran 67 tests in 354.843s", "OK". `python3
      -B tools/gate.py --timeout 5400`, exit 0: it waited ("load 55.8;
      waiting for it to fall below 10"), "started at load 9.9", and "gate:
      30 of 30 passed", every check rc=0, among them pre-add-check ("933
      file(s) clean", "nested module(s) clean: authcontract."), the six `go
      vet` runs, test-race-cover, test-shuffle, test-repeat, lint,
      api-check, contract-check ("authcontract, 0 problem(s)"),
      generate-check, changelog-check ("44 release section(s), 0
      problem(s)"), docs-check, gate-selftest ("Ran 67 tests", "OK"),
      markdownlint, g-wire-stable, links-check, disclosure-check, deny-scan
      and tools-test. The gate ran on this entry as it stood before these
      results were written in; after them, each exit 0: `make
      contract-check` (the no-tag line, "authcontract, 0 problem(s)"),
      `make changelog-check` ("44 release section(s), 0 problem(s)"),
      `tools/claims.py` on the MADR ("44 of 44 pass") and this PLAN ("11 of
      11 pass"), `make records-check` ("162 records, 0 problem(s)"),
      `tools/recordlint.py --changed` ("no rule rose"), `tools/denyscan.py
      --changed` ("0 hit(s) in 6 file(s)"), `make links-check` and
      `tools/linkcheck.py` ("0 problem(s), 1248 relative link(s) in 176
      file(s)") and `make disclosure-check` ("0 hit(s) in 1582 file(s)").
* Checks, each exit status captured before any filter, on this phase's
  files:
  * In the gate, rc=0: `pre-add-check` ("933 file(s) clean", "nested
    module(s) clean: authcontract."), the six `go vet` runs,
    `test-race-cover`, `test-shuffle`, `test-repeat`, `tidy-diff`, `lint`,
    `parity-check`, `dep-check`, `coverage-check`, `api-check`,
    `contract-check`, `generate-check`, `records-check`, `tables-check`,
    `fixtures-check`, `changelog-check`, `docs-check`, `markdownlint`,
    `g-wire-stable`, `links-check`, `disclosure-check`, `deny-scan` and
    `tools-test`.
  * Directly, before the gate, each exit 0: `make contract-check`, "no
    authcontract/v* tag yet, so nothing to compare; the API and the schema
    are held from the first tag on", "authcontract, 0 problem(s)"; `make
    changelog-check`, "44 release section(s), 0 problem(s)" (43 before, and
    v0.1.0's, which has no tag and is newer than every contract tag, so it
    is the release being prepared); `make markdownlint`, "Linting: 14
    files", "0 issues"; `make links-check`, "0 problem(s), 1248 relative
    link(s) in 176 file(s)"; `make disclosure-check`, "0 hit(s) in 1582
    file(s)".
  * After this entry was written, each exit 0: `tools/claims.py` on the
    MADR, "44 of 44 pass", and on this PLAN, "11 of 11 pass"; `make
    records-check`, "records-check: 162 records, 0 problem(s)";
    `tools/recordlint.py --changed`, "recordlint: 1 record(s); no rule
    rose"; `tools/denyscan.py --changed`, "deny-scan: 0 hit(s) in 5
    file(s)"; `tools/linkcheck.py` and `make links-check`, "0 problem(s),
    1248 relative link(s) in 176 file(s)"; `make disclosure-check`, "0
    hit(s) in 1582 file(s)". They were run once more after this line was
    written.
* Not done: no tag, no push, no commit. Phase 4's files, `AGENTS.md`,
  `README.md`, `authcontract/CHANGELOG.md`, `docs/architecture.md`,
  `scripts/test_gates.py` and this PLAN, are staged for the coordinator to
  commit.

**For the owner, after the commit (steps 4 and 5).** With this phase
committed, each from the repository root, on `main`, in this order:

1. The disclosure guard over the outgoing commits (AGENTS.md,
   "Identifiers"):

   ```bash
   echo "refs/heads/main $(git rev-parse HEAD) refs/heads/main $(git rev-parse origin/main)" |
     python3 ~/.global-git-hooks/github-disclosure.py pre-push origin "$(git remote get-url origin)"
   ```

2. `git push origin main`. A commit of Markdown alone runs
   `.github/workflows/docs.yml`, not `ci.yml`
   (`0052-MADR-skip-full-ci-for-docs-only-pushes.md`); this commit also
   changes `scripts/test_gates.py` (the resolution above), so the push runs
   `ci.yml` in full, which is to pass before the tag.
3. The tag, annotated, on `main`'s head, with its own name as its message,
   the form of every SDK tag (`v1.30.0`'s message is `v1.30.0`) and of
   gobble-cli's `acpext/v0.4.0`, the precedent D1 follows:
   `git tag -a authcontract/v0.1.0 -m authcontract/v0.1.0`, then
   `git cat-file -t authcontract/v0.1.0` prints `tag` and
   `git rev-parse authcontract/v0.1.0^{commit}` equals
   `git rev-parse HEAD`.
4. `git push origin authcontract/v0.1.0`.
5. The CI run on the tag: `ci.yml` runs in full on `authcontract/v*`
   (Phase 2, step 6), and its `release` job is skipped, since its condition
   is `refs/tags/v`. Wait for it:
   `gh run list --workflow ci.yml --branch authcontract/v0.1.0 --limit 1`,
   then `gh run watch <id> --exit-status`; every `validate` job must pass,
   and `release` must be skipped.
6. After CI passes, from a scratch directory outside the repository:
   `cd "$(mktemp -d)" && go list -m github.com/maccavelli/go-llmprovider-sdk/authcontract@v0.1.0`,
   which prints `github.com/maccavelli/go-llmprovider-sdk/authcontract v0.1.0`
   once the module proxy resolves it.
7. Back in the repository: `make changelog-check` passes with the tag, and
   `make contract-check` compares against `authcontract/v0.1.0` and reports
   0 problems. Phase 4's acceptance is then met, and this PLAN's status and
   the record of the tag are the next records change.

### Phase 4, CI deviation (2026-11-12)

The coordinator committed Phase 4 as `219c9b4` on the owner's authorization,
ran the disclosure guard (exit 0) and pushed `main` (`aaa2904..219c9b4`), on
the owner's approval of steps 4 and 5. No tag was made.

* **Deviation: `ci.yml` failed on the push, on Windows only.** Run
  43284169297 on `219c9b4`: `validate (macos-15)` and `validate
  (ubuntu-24.04)` passed; `validate (windows-2025)` failed at the step "the
  nested module authcontract (0073-MADR D8)", so its later steps (race,
  govulncheck, vet, lint, the gates) were skipped there.
  * Evidence. `go -C authcontract test -shuffle=on ./...`: `--- FAIL:
    TestSchemagenMatchesCommitted_0073`, `main_test.go:44: run wrote 8492
    bytes that differ from the committed ../../authcontract.schema.json
    (8920 bytes); run go generate`, `line 1: generated "{", committed
    "{\r"`, then `main_test.go:52: run with no -output wrote other bytes
    than the committed schema`. The schema has 428 lines, and 8492 + 428 =
    8920: the runner's checkout (`core.autocrlf=true`, the Windows runner
    default) wrote every line with CRLF. The repository had no
    `.gitattributes`. Not caught earlier: Phases 2 to 4 reached CI in this
    one push, and the gate runs on this Mac, whose checkout is LF.
    `internal/wiretest/wiretest.go` already strips CRLF from its goldens for
    the same reason.
  * Resolutions put to the owner: (1) *recommended*, a `.gitattributes` of
    `* text=auto eol=lf`, the fleet's line-ending rule (gobble-cli's carries
    it), so every checkout holds the committed bytes; (2) the same for
    `authcontract/authcontract.schema.json` alone, which leaves the golden
    vectors and every other byte-compared file to meet it later; (3) strip
    CRLF in the test, as `wiretest` does, which loosens a byte-exact check
    on a published file and leaves a Windows checkout's bytes differing
    from the ones the proxy serves. Doing nothing: `ci.yml` stays red on
    Windows, and Phase 4, step 3's tag cannot be made on green CI.
  * **Resolution (2026-11-12): the owner chose (1).** Phase 4 gains
    `.gitattributes`, with a dated note under its step 2. Survey before the
    change: `git ls-files --eol` lists 1574 files `i/lf` and 8 `i/none`
    (single-line JSON fixtures with no line end), none `i/crlf` or
    `i/mixed`; no tracked file is binary (933 `.go`, 423 `.json`, 176 `.md`,
    32 `.py`, and the rest text). So the attribute changes no committed
    blob.
  * Seen to fail, then pass, on a scratch clone of `219c9b4` made with
    `git clone -c core.autocrlf=true`, never in the tree: the schema held
    428 CRs; `go -C authcontract test -count=1 ./internal/schemagen/`, exit
    1, `--- FAIL: TestSchemagenMatchesCommitted_0073`, `line 1: generated
    "{", committed "{\r"`. With this `.gitattributes` committed there and
    the tree checked out again: `git status --porcelain` empty, 0 CRs in the
    schema and 0 files with a CR in the tree; `go -C authcontract test
    -count=1 ./...`, exit 0, `ok` for `authcontract`,
    `authcontract/internal/schemacheck` and
    `authcontract/internal/schemagen`.
  * Next: the commit of `.gitattributes` and this PLAN, the guard, the push
    of `main` and its `ci.yml` run, which is to pass on all three runners
    before steps 3 to 7 above.
