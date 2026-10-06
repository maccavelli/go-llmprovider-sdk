---
status: proposed
date: 2026-10-05
associated-madr: "0023-MADR-huggingface-tool-choice-none.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement Hugging Face's `ToolChoiceNone` by Sending No Tools

Associated MADR: [0023-MADR-huggingface-tool-choice-none.md](0023-MADR-huggingface-tool-choice-none.md)

## Goal

A Hugging Face request with `ToolChoiceNone` sends neither `tools` nor
`tool_choice`, and every other tool choice is sent as today. Done when:

* the offline red test fails on `HEAD` and passes on the change;
* the gate is clean at the end of each phase;
* `TestLive_HuggingFaceToolChoices` passes live, `none` included;
* `none` after a tool call, the case the MADR leaves unmeasured, is measured
  live and pinned by a live subtest;
* the execution record below holds the output.

## Scope

* **Changed:**
  * `llmprovider/providers/huggingface/huggingface.go`: `body`, and the
    package comment's Degradations;
  * `llmprovider/providers/huggingface/request_test.go`:
    `TestGenerate_RequestFields`'s tool-choice table;
  * `llmprovider/live_huggingface_test.go`: `TestLive_HuggingFaceToolChoices`
    gains a subtest (phase 2).
* **Records:** this PLAN, its MADR, and `docs/README.md`.
* **Not changed:**
  * `internal/wire/chatcompletions`, and every other provider;
  * the `huggingface` goldens, which pin a named tool;
  * `Capabilities`, any exported API, CI and `scripts/`.

## Implementation Steps

### Phase 1: the change, offline

#### 1.1 Red

In `TestGenerate_RequestFields` (`request_test.go:80`), the tool-choice
table gains a `tools` column: how many tools the body carries.

| Choice | `tool_choice` | `tools` |
| :--- | :--- | :--- |
| `ToolChoiceAuto` | `<nil>` | 2 |
| `ToolChoiceRequired` | `required` | 2 |
| `ToolChoiceNone` | `<nil>` | 0 |
| `ForceTool("get_weather")` | the function object, as today | 2 |

* Only the `none` row's wants change; the others are today's.
* The `tools` check reads the column instead of the fixed 2.
* **Red,** on a scratch copy of `HEAD` with the new test file: the `none`
  row fails, `tool_choice = none, want <nil>` and `2 tools sent, want 0`;
  the other three rows pass.

#### 1.2 The change

In `huggingface.go`'s `body` (`:146-161`):

```go
tools, choice := req.Tools, req.ToolChoice
if choice == llmprovider.ToolChoiceNone {
	// The router's upstreams refuse a call made under "none" with HTTP 400
	// tool_use_failed, so none is kept by offering no tools (0023-MADR).
	tools, choice = nil, llmprovider.ToolChoiceAuto
}
```

and `chatcompletions.Opts{..., Tools: tools, ToolChoice: choice}`.

* **Green:** `go test ./llmprovider/providers/huggingface/` passes, the red
  test included.
* `go test -count=3 ./llmprovider/... -run TestWireGoldens` passes: no
  golden changes.

#### 1.3 Docs

The package comment's Degradations line, `huggingface.go:28-29`, becomes:

```go
//   - ToolChoiceRequired is sent as "required", the Chat Completions value.
//     ToolChoiceNone is kept by sending no tools and no tool_choice: the
//     router's upstreams answer a call made under "none" with HTTP 400
//     tool_use_failed (0023-MADR, measured 2026-10-05).
```

Nothing in `docs/architecture.md` or `docs/guides/` describes tool choice
per provider, so neither changes. This is checked again before staging.

#### 1.4 Gate

G1–G11 as `0021-PLAN-harden-and-tune-after-the-v1-1-review.md` defines
them, with its phase 6 additions: G4 with `-shuffle=on`, and G7 with
`gate-selftest`. `api-check` compares against `v1.2.0`, and must report 0
incompatible changes.

#### 1.5 Phase 1 record

The execution record holds 1.1–1.4's output. The changed files are staged,
and the owner commits. The PLAN stays `in-progress`: phase 2 is live.

### Phase 2: live, when the account's credits allow

The account's Hugging Face credits were used up on 2026-10-05. Until they
reset or are bought, every call answers `HTTP 402`, which `SkipIfTransient`
skips. Phase 2 waits for the owner's word that they are back.

Every run below is on a scratch copy of the tree, with `HF_TOKEN` checked
for presence only. It records outcomes, not replies.

#### 2.1 The credit check

Run `TestLive_HuggingFaceChatCompletions` once. If it skips on the 402, stop
and tell the owner. Nothing else in phase 2 runs.

#### 2.2 The existing test

Run `TestLive_HuggingFaceToolChoices` twice. Both pass, `required` and
`none` included.

#### 2.3 The unmeasured cases

A scratch test, never in the tree, three runs of each:

* **(a)** `openai/gpt-oss-120b`, `ToolChoiceNone`, one tool offered, and a
  history of a user turn, a `FunctionCallItem` for `get_weather` and its
  `FunctionCallOutputItem`;
* **(b)** `openai/gpt-oss-20b`, `ToolChoiceNone`, one tool offered, no
  history: the case the MADR's probe could not finish.

Each must answer with text and no call. **Stop and prompt with the counts**
if any run of (a) or (b) answers with a 400, or with a call.

#### 2.4 Pin (a)

`TestLive_HuggingFaceToolChoices` gains a subtest, `none-after-a-call`: case
(a) once, asserting no error and no call.

* **Seen to fail:** on a scratch copy with step 1.2's change reverted, the
  subtest fails with the 400. If it passes there instead, record that the
  history alone does not trigger the refusal, and keep the subtest as a
  guard.
* It passes twice on a tree copy.

#### 2.5 Gate and close-out

* The gate, as in 1.4.
* The execution record holds 2.1–2.5's output.
* This PLAN is `complete`, and its `docs/README.md` row with it. The MADR's
  status is the owner's.
* Stage; the owner commits.

## Verification

* **V1:** 1.1's FAIL line on `HEAD`, then the PASS on the change.
* **V2:** the gate is clean at the end of each phase.
* **V3:** 2.2's two passes, 2.3's counts, and 2.4's planted failure and
  passes.
* **V4:** outside `docs/`, the diff touches only the three files in Scope:
  `git diff --cached --stat` is recorded each phase.

## Rollout and Rollback

* **Rollout:**
  * one commit per phase on `main`, staged by the agent and committed by
    the owner;
  * a behaviour change in a provider and no API change, so it can ship in a
    patch release, `v1.2.1`, when the owner tags it.
* **Rollback:** revert phase 1's commit to send `"tool_choice":"none"` with
  the tools again. Phase 2's commit only adds a live subtest, and reverts on
  its own.

## Execution Record

Not started.
