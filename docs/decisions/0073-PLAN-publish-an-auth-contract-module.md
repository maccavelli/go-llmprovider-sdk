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
     it, each by a record of its own.)*
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
