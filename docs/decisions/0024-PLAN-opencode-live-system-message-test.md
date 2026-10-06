---
status: proposed
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

Not started.
