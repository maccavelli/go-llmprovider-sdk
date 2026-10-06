---
status: complete
date: 2026-10-05
associated-madr: "0022-MADR-gemini-live-instructions-test.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement the Paired Gemini Live Instructions Check

Associated MADR: [0022-MADR-gemini-live-instructions-test.md](0022-MADR-gemini-live-instructions-test.md)

## Goal

`TestLive_GeminiInteractions` checks the instructions as a pair: with the
neutral system item "Begin every reply with the word OMEGA." the reply
contains `OMEGA`; without it, the reply does not. Done when:

* the paired test passes live twice, on a tree copy;
* a planted break on a scratch copy, a `system_instruction` that is never
  sent, fails `with` and not `without`;
* the repository gate is clean;
* the execution record below holds the output.

## Scope

* **Changed:** `llmprovider/live_gemini_test.go`, in
  `TestLive_GeminiInteractions` only (`:50-86`).
* **Records:** this PLAN, its MADR, and `docs/README.md`.
* **Not changed:**
  * `llmprovider/providers/gemini/` and its goldens;
  * `TestLive_GrokInstructions` and every other live test;
  * any exported API, CI, or `scripts/`.

## Implementation Steps

### Phase 1: the paired check

#### 1.1 The test

Rewrite the first part of `TestLive_GeminiInteractions` as two subtests from
a table:

| Subtest | `Input` | Reply contains `OMEGA` | Recorded request |
| :--- | :--- | :--- | :--- |
| `with` | a system item "Begin every reply with the word OMEGA.", then the user's "Say hello." | yes | one POST, path ending `/interactions`, body has `"store":false` and `"system_instruction"` |
| `without` | the user's "Say hello." only | no | one POST, path ending `/interactions`, body has `"store":false` and no `"system_instruction"` |

* Each subtest has its own `geminiRecorder`, its own `LiveCtx`, and model
  `gemini-3.7-flash`, as today.
* Each calls `llmprovider.SkipIfTransient(t, err)` and fails on any other
  error before it checks the reply.
* It checks the reply first, then the request. Its failure message names
  the subtest's want, and quotes the reply as the current test does.
* The `WithStore(true)` continuation that follows stays as it is, with its
  own recorder and client.
* The doc comment says the check is a pair with a neutral instruction, and
  cites `0022-MADR-gemini-live-instructions-test.md` and
  `0014-MADR-gemini-wire-fidelity.md` §1–§2.

#### 1.2 Seen to fail

On a scratch copy of the tree, never in it, with `GEMINI_API_KEY` checked
for presence only:

* plant, in `llmprovider/providers/gemini/gemini.go`, `if len(system) > 0 {`
  → `if false && len(system) > 0 {`, so `system_instruction` is never sent;
* run `go test -tags live_gateways -count=1 -v -run
  '^TestLive_GeminiInteractions$' ./llmprovider`;
* **expected:** exit 1; `with` fails on the reply or the request, and
  `without` passes. The record keeps the FAIL line, without the reply's
  text.

#### 1.3 Live

On a tree copy, the same command twice. Both pass, every subtest included,
and the continuation too.

**Stop and prompt with the counts,** rather than change the wording, if:

* `without` finds `OMEGA` in either run;
* `with` fails without the plant;
* the run skips (`SkipIfTransient`) both times.

#### 1.4 Gate

G1–G11 as `0021-PLAN-harden-and-tune-after-the-v1-1-review.md` defines
them, with phase 6's additions: G4 with `-shuffle=on`, and G7 with
`gate-selftest`. G3, `go vet -tags live_gateways ./...`, and G6, `make
lint`, are the ones that read the live-tagged file.

#### 1.5 Close-out

* The execution record below: each step's output.
* This PLAN's status `complete`, and its `docs/README.md` row with it.
* Stage the changed files. The owner commits and pushes.

## Verification

* **V1:** step 1.2's planted FAIL line, then step 1.3's two PASS runs.
* **V2:** the gate, G1–G11, is clean.
* **V3:** the diff touches only `llmprovider/live_gemini_test.go` outside
  `docs/`: `git diff --cached --stat` is recorded.

## Rollout and Rollback

* **Rollout:** one commit on `main`, staged by the agent and committed by
  the owner. No release is needed: the change is a live-tagged test, which
  no build of the module includes.
* **Rollback:** revert that commit. Nothing depends on it.

## Execution Record

### Phase 1: the paired check (2026-10-05)

Approved by the owner on 2026-10-05 ("approved to proceed"). No deviation.

**1.1, the test.**
* **Built:** `TestLive_GeminiInteractions` (`llmprovider/live_gemini_test.go`)
  runs `with` and `without` from a table, as step 1.1's table says. Each
  subtest has its own `geminiRecorder` and `LiveCtx`, skips through
  `SkipIfTransient`, checks the reply, then the request.
* The `WithStore(true)` continuation follows, with its own client and
  context, otherwise unchanged.
* The doc comment cites `0022-MADR-gemini-live-instructions-test.md` and
  0014 §1–§2.
* `gofmt` and `go vet -tags live_gateways ./llmprovider` are clean.

**1.2, seen to fail,** on a scratch copy with `gemini.go`'s `if len(system)
> 0 {` planted as `if false && len(system) > 0 {`, `GEMINI_API_KEY` checked
for presence only:

```text
live_gemini_test.go:81: Generate = <reply elided>; contains OMEGA = false, want true
--- FAIL: TestLive_GeminiInteractions (21.92s)
--- FAIL: TestLive_GeminiInteractions/with (9.09s)
--- PASS: TestLive_GeminiInteractions/without (2.86s)
FAIL
```

Exit 1. `with` fails on the reply, before its request check, and `without`
passes. The continuation ran and passed.

**1.3, live,** on a tree copy, twice: exit 0 both times. `with` and
`without` passed each time, and the continuation too. No stop condition
applied: `without` found no `OMEGA`, and nothing skipped.

**1.4, the gate,** all exit 0:

* G1: `make pre-add-check`, "396 file(s) clean (gofmt, golangci-lint, go
  vet, go test, govulncheck)";
* G2: `go vet` for darwin, linux and windows; G3: `go vet -tags
  live_gateways ./...`;
* G4: 26 packages ok with `-race -cover`, and `go test -race -shuffle=on
  ./...` exit 0; `llmprovider` 97.9%, `wizard` 88.5%;
* G5: `go mod tidy -diff` clean;
* G6: `make lint`, 0 issues on both builds;
* G7:
  * parity: 409 identifiers, 0 problems;
  * dep-check: go.mod and 27 packages on both builds, 0 problems;
  * coverage-check: 27 packages, 0 problems;
  * api-check: against `v1.2.0`, now the newest tag, 0 incompatible
    changes;
  * generate-check: 1 generated file, 0 problems;
  * gate-selftest: 7 tests OK in 58 s;
* G8: 0 issues on the repository's markdownlint scope, which
  `.markdownlint-cli2.jsonc` defines without the records;
* G9: stable; G10: 0 problems; G11: 0 hits in 4 files.

**V3, the diff** against `d4f9fa5`, outside `docs/`, is the test file only:

```text
docs/README.md                                     |   4 +-
.../0022-MADR-gemini-live-instructions-test.md     | 186 +++++++++++++++++++++
.../0022-PLAN-gemini-live-instructions-test.md     | 115 +++++++++++++
llmprovider/live_gemini_test.go                    |  56 +++++--
4 files changed, 342 insertions(+), 19 deletions(-)
```

Counted before this record was written; the PLAN's line count grows with
it.

**1.5, close-out.** The Goal holds. This PLAN is `complete`, and
`docs/README.md` says so.
