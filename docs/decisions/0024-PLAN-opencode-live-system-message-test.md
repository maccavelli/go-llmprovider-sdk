---
status: complete
date: 2026-10-06
associated-madr: "0024-MADR-opencode-live-system-message-test.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement the Paired OpenCode Live System-Message Check

Associated MADR: [0024-MADR-opencode-live-system-message-test.md](0024-MADR-opencode-live-system-message-test.md)

## Goal

`TestLive_OpencodeSystemMessage` checks the system item as a pair: with the
system item "Begin every reply with the word OMEGA.", the reply contains
`OMEGA` and the request carries `system`; without it, neither. Done when:

* the pair passes live twice, on a tree copy;
* a planted break on a scratch copy, a `system` never sent, fails `with`
  and not `without`;
* the repository gate is clean;
* the execution record below holds the output.

## Scope

* **Changed:** `llmprovider/live_opencode_test.go`, in
  `TestLive_OpencodeSystemMessage` only (`:342-360`).
* **Records:** this PLAN, its MADR, and `docs/README.md`.
* **Not changed:** `llmprovider/providers/opencode/` and its goldens, every
  other live test, any exported API, CI and `scripts/`.

## Implementation Steps

### Phase 1: the paired check

#### 1.1 The test

`TestLive_OpencodeSystemMessage` picks the model once, `goModel(t,
"qwen3.8-flash", "minimax-m3")`, then runs two subtests from a table:

| Subtest | `Input` | Reply contains `OMEGA` | Recorded request |
| :--- | :--- | :--- | :--- |
| `with` | a system item "Begin every reply with the word OMEGA.", then the user's "Say hello." | yes | one POST, path ending `/messages`, body has a top-level `"system"` key |
| `without` | the user's "Say hello." only | no | one POST, path ending `/messages`, body has no `"system"` key |

* Each subtest has its own `opencodeRecorder`, passed with
  `llmprovider.WithHTTPClient`, and its own two-minute context, as today.
* Each calls `llmprovider.SkipIfTransient(t, err)` and fails on any other
  error.
* It checks the reply, then the request. It reads the body's top-level keys
  by decoding the JSON, not by a substring match, so a `system` inside a
  message does not count. Its failure message quotes the reply, as today.
* The doc comment says the check is a pair with a neutral instruction, and
  cites `0024-MADR-opencode-live-system-message-test.md`.

#### 1.2 Seen to fail

On a scratch copy of the tree, with `OPENCODE_API_KEY` checked for presence
only:

* plant, in `llmprovider/providers/opencode/opencode.go`, `if system :=
  wire.SystemPrompt(c.input); system != "" {` → `if system :=
  wire.SystemPrompt(c.input); false && system != "" {`;
* run `go test -tags live_gateways -count=1 -v -run
  '^TestLive_OpencodeSystemMessage$' ./llmprovider`;
* **expected:** exit 1; `with` fails on the reply or the request, and
  `without` passes. The record keeps the FAIL line, without the reply.

#### 1.3 Live

On a tree copy, the same command twice: both pass, each subtest included.

**Stop and prompt with the counts,** rather than change the wording, if:

* `without` finds `OMEGA`;
* `with` fails without the plant;
* the picker skips both times.

#### 1.4 Gate

G1–G11 as `0021-PLAN-harden-and-tune-after-the-v1-1-review.md` defines
them, with its phase 6 additions: G4 with `-shuffle=on`, and G7 with
`gate-selftest`.

#### 1.5 Close-out

* The execution record below: each step's output.
* This PLAN's status `complete`, and its `docs/README.md` row with it.
* Stage; the owner commits.

## Verification

* **V1:** 1.2's planted FAIL line, then 1.3's two PASS runs.
* **V2:** the gate is clean.
* **V3:** outside `docs/`, the diff touches only
  `llmprovider/live_opencode_test.go`: `git diff --cached --stat` is
  recorded.

## Rollout and Rollback

* **Rollout:** one commit on `main`, staged by the agent and committed by
  the owner. No release is needed: the change is a live-tagged test.
* **Rollback:** revert that commit.

## Execution Record

### Phase 1: the paired check (2026-10-06)

Approved by the owner on 2026-10-06 ("proceed to 0024"), after the records
were committed as `759592c`. No deviation.

**1.1, the test.**
* **Built:** `TestLive_OpencodeSystemMessage` picks the model once, then
  runs `with` and `without` from a table, as step 1.1's table says.
* Each subtest has its own `opencodeRecorder`, read through its `last()`
  accessor as the file's other tests do, and its own two-minute context.
* It checks the reply, then the request. It reads the top-level keys by
  decoding the body into `map[string]json.RawMessage`.
* The doc comment cites this PLAN's MADR.
* `gofmt` and `go vet -tags live_gateways ./llmprovider` are clean.

**1.2, seen to fail,** on a scratch copy with `opencode.go`'s `if system :=
wire.SystemPrompt(c.input); system != "" {` planted as `...; false && system
!= "" {`, `OPENCODE_API_KEY` checked for presence only:

```text
live_opencode_test.go:349: picked qwen3.8-flash from [qwen3.8-flash minimax-m3]
live_opencode_test.go:373: reply <elided>: contains OMEGA = false, want true
--- FAIL: TestLive_OpencodeSystemMessage/with (1.98s)
--- PASS: TestLive_OpencodeSystemMessage/without (1.65s)
```

Exit 1. `with` failed on the reply, before its request check, and `without`
passed.

**1.3, live,** on a tree copy, twice: exit 0 both times, with `with` and
`without` passing on `qwen3.8-flash`. No stop condition applied.

**1.4, the gate,** all exit 0:

* G1: `make pre-add-check`, "396 file(s) clean (gofmt, golangci-lint, go
  vet, go test, govulncheck)";
* G2: `go vet` for darwin, linux and windows; G3: `go vet -tags
  live_gateways ./...`;
* G4: 26 packages ok with `-race -cover`, and `go test -race -shuffle=on
  ./...` exit 0; `llmprovider` 97.9% against its 95.9% floor;
* G5: `go mod tidy -diff` clean;
* G6: `make lint`, 0 issues on both builds;
* G7:
  * parity: 409 identifiers, 0 problems;
  * dep-check: go.mod and 27 packages on both builds, 0 problems;
  * coverage-check: 27 packages, 0 problems;
  * api-check: against `v1.2.1`, now the newest tag, 0 incompatible
    changes;
  * generate-check: 1 generated file, 0 problems;
  * gate-selftest: 7 tests OK in 49 s;
* G8: 0 issues on the repository's markdownlint scope;
* G9: stable;
* G10: 0 problems, 593 relative links in 65 files;
* G11: 0 hits in 4 files.

**V3, the diff** against `759592c`, counted before this record was written:

```text
docs/README.md                                     |  4 +-
.../0024-MADR-opencode-live-system-message-test.md |  5 +-
.../0024-PLAN-opencode-live-system-message-test.md |  2 +-
llmprovider/live_opencode_test.go                  | 55 ++++++++++++++++------
4 files changed, 47 insertions(+), 19 deletions(-)
```

The `docs/` lines are the approval's statuses.

**1.5, close-out.** The Goal holds. This PLAN is `complete`, and
`docs/README.md` says so. Staged for the owner's commit.
