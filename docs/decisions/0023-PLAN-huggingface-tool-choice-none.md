---
status: complete
date: 2026-10-06
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
* *Amended 2026-10-06:* both `none` subtests are pinned to Groq, and
  `none-after-a-call` ends with a follow-up turn. See the deviation "step
  2.4's check does not fail without the fix".
* *Amended 2026-10-06:* `none-after-a-call` is dropped. See the deviation
  "`none` after a call is refused with the fix".

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

### Phase 1: the change, offline (2026-10-05)

Approved by the owner on 2026-10-05 ("approved, proceed"), after the
records were committed as `f00f677`. No deviation.

**1.1, red,** on a scratch copy of `HEAD` (`f00f677`) with the new
`request_test.go`:

```text
request_test.go:117: "none": tool_choice = none, want <nil>
request_test.go:120: "none": 2 tools sent, want 0
--- FAIL: TestGenerate_RequestFields (0.00s)
```

Exit 1. Only the `none` row failed; the auto, `required` and named-tool rows
passed.

**1.2, the change.**
* `body` passes no `Tools` and the auto choice under `ToolChoiceNone`, as
  the step's code says, with the comment citing this PLAN's MADR.
* **Green:** `go test ./llmprovider/providers/huggingface/` ok, the red test
  included.
* `go test -count=3 ./llmprovider/... -run TestWireGoldens`: ok, no golden
  changed.

**1.3, docs.** The package comment's Degradations line reads as the step
gives it. A search of `docs/architecture.md`, `docs/guides/adding-a-provider.md`
and `docs/guides/api-standards.md` finds no per-provider tool-choice text, so
none changed.

**1.4, the gate,** all exit 0:

* G1: `make pre-add-check`, "396 file(s) clean (gofmt, golangci-lint, go
  vet, go test, govulncheck)";
* G2: `go vet` for darwin, linux and windows; G3: `go vet -tags
  live_gateways ./...`;
* G4: 26 packages ok with `-race -cover`, and `go test -race -shuffle=on
  ./...` exit 0; `llmprovider` 97.9%, `providers/huggingface` 96.2%,
  `wizard` 88.5%;
* G5: `go mod tidy -diff` clean;
* G6: `make lint`, 0 issues on both builds;
* G7:
  * parity: 409 identifiers, 0 problems;
  * dep-check: go.mod and 27 packages on both builds, 0 problems;
  * coverage-check: 27 packages, 0 problems;
  * api-check: against `v1.2.0`, 0 incompatible changes;
  * generate-check: 1 generated file, 0 problems;
  * gate-selftest: 7 tests OK in 54 s;
* G8: 0 issues on the repository's markdownlint scope, which
  `.markdownlint-cli2.jsonc` defines without the records;
* G9: stable; G10: 0 problems; G11: 0 hits in 5 files.

**V4, the diff** against `f00f677`, counted before this record was written:

```text
docs/README.md                                           |  4 ++--
docs/decisions/0023-MADR-huggingface-tool-choice-none.md |  6 +++++-
docs/decisions/0023-PLAN-huggingface-tool-choice-none.md |  2 +-
llmprovider/providers/huggingface/huggingface.go         | 14 +++++++++++---
llmprovider/providers/huggingface/request_test.go        | 15 +++++++++------
5 files changed, 28 insertions(+), 13 deletions(-)
```

The `docs/` lines are the approval's statuses: the MADR `accepted`, with its
status note, and this PLAN `in-progress`.

**1.5.** Staged for the owner's commit. The PLAN stays `in-progress`: phase
2 waits for the account's Hugging Face credits.

### Deviation 2026-10-06: step 2.4's check does not fail without the fix

* **Found,** running step 2.4 on a scratch copy with step 1.2's change taken
  out: `TestLive_HuggingFaceToolChoices` passed, `required`, `none` and
  `none-after-a-call` alike. Then, with the change still out, three runs
  each:

  | Request: tools and `"tool_choice":"none"` | Served by | Outcome |
  | :--- | :--- | :--- |
  | `openai/gpt-oss-120b`, the router's choice | `cerebras` | text, 3/3 |
  | `openai/gpt-oss-120b:groq`, pinned | `groq` | `400 tool_use_failed`, 3/3 |

  * Groq still refuses. The router sent the model to Groq on 2026-10-05 and
    to Cerebras on 2026-10-06, and Cerebras honours `none`. So the unpinned
    `none` subtest passes with or without the fix.
  * `none-after-a-call` passed without the fix even on Groq. With the tool's
    result in the history, the model answers rather than calling again, so
    there is no call to refuse. Step 2.3's case (a) on Groq gave text 3/3
    with the fix.
* **Resolution, chosen by the owner** ("Pin to Groq; follow-up turn"):
  * the `none` and `none-after-a-call` subtests use
    `openai/gpt-oss-120b:groq`; `required` stays on the router's choice;
  * `none-after-a-call`'s history ends with a follow-up user turn, "And
    what about Lyon?", so the model has a reason to call again;
  * measured before it is kept: with step 1.2 taken out, both `none`
    subtests must fail with the 400; with it, both pass. If the follow-up
    does not fail without the fix, stop and prompt again.
* **MADR:** amended with "the router's choice of upstream".
* **Files added to the phase:** none.

### Deviation 2026-10-06: `none` after a call is refused with the fix

* **Found,** with the first deviation's resolution applied: the pinned
  `none` subtest fails without step 1.2 and passes with it, as wanted. But
  `none-after-a-call`, with its follow-up "And what about Lyon?", fails on
  the tree too, in both runs:

  ```text
  live_huggingface_test.go:78: Generate: llmprovider: invalid request: huggingface HTTP 400 tool_use_failed: Tool choice is none, but model called a tool
  --- FAIL: TestLive_HuggingFaceToolChoices/none-after-a-call
  ```

  * With no tools sent, `gpt-oss-120b` on Groq still writes a call when the
    history holds one and the user asks for more. Groq refuses that call.
  * Without the follow-up, step 2.3's case (a) gave text 3/3: the model had
    no reason to call again.
  * Not a regression: before `6687ff8` the same request was refused with
    the tools sent. But the MADR's "the model cannot call one" does not hold
    for this case.
* **Resolution, chosen by the owner** ("Measure flattening first"): a
  scratch probe, before any decision changes. Under `none`, the history's
  calls and their results are sent as plain assistant and user text, with
  no `tool_calls` or `tool` messages. If that answers with text and no
  call, 3 runs of 3, the MADR is amended and this PLAN gains the steps.
  Otherwise, stop and prompt again.
* **State, then:** the subtest edit and these entries were uncommitted;
  nothing was staged.
* **The probe,** on a scratch copy of the tree, with the follow-up history
  and every `tool_calls` and `tool` message rewritten as plain assistant and
  user text (2 messages rewritten on every run), three runs each:

  | Model, pinned to Groq | Outcome |
  | :--- | :--- |
  | `openai/gpt-oss-120b:groq` | text 2/3; `400 tool_use_failed` 1/3 |
  | `openai/gpt-oss-20b:groq` | `400 tool_use_failed` 2/3; `400 output_parse_failed` 1/3 |

  So flattening does not hold: asked again for what a tool answered before,
  `gpt-oss` on Groq writes a call with no tools defined, whether the earlier
  call is structured or prose, and Groq refuses it.
* **Resolution, chosen by the owner** ("Record the Groq limit"):
  * `6687ff8`'s fix stays: it keeps `none` with no call history, and the
    Groq-pinned `none` subtest fails without it and passes with it;
  * `none-after-a-call` is dropped. The Goal's item "`none` after a tool
    call ... pinned by a live subtest" is not done: no request form makes
    it pass on Groq, so a subtest would assert a refusal the provider
    cannot prevent;
  * the package comment names the limit;
  * the MADR is amended with "`none` after a tool turn on Groq".
* **Files added to the phase:** none.

### Phase 2: live (2026-10-06)

Run after the owner's "proceed with phase 2", on scratch copies, with
`HF_TOKEN` checked for presence only. Two deviations, above, each decided by
the owner.

**2.1, the credit check:** `TestLive_HuggingFaceChatCompletions` passed, a
real answer, not the 402 skip.

**2.2, the existing test,** on a tree copy, twice, before the deviations:
both passed, `required` and `none` included.

**2.3, the unmeasured cases,** with the fix, three runs each:

| Case | Served by | Outcome |
| :--- | :--- | :--- |
| (a) `openai/gpt-oss-120b`, `none` after a call | `cerebras` | text 3/3 |
| (a) pinned, `openai/gpt-oss-120b:groq` | `groq` | text 3/3 |
| (b) `openai/gpt-oss-20b`, `none` | `groq` | text 3/3 |

(a) was rerun pinned because the router had not sent it to Groq, the
upstream that refuses. Neither case met a stop condition.

**2.4, the live subtest.** Its as-written form did not fail without the fix
(the first deviation), and its follow-up form failed with it (the second).
As built, by the owner's choice "Record the Groq limit":

* `TestLive_HuggingFaceToolChoices` has two subtests: `required` on
  `openai/gpt-oss-120b`, and `none` pinned to `openai/gpt-oss-120b:groq`.
  The doc comment cites the MADR and its amendment "the router's choice of
  upstream".
* **Seen to fail,** on a scratch copy with step 1.2 taken out:

  ```text
  live_huggingface_test.go:68: Generate: llmprovider: invalid request: huggingface HTTP 400 tool_use_failed: Tool choice is none, but model called a tool
  --- PASS: TestLive_HuggingFaceToolChoices/required
  --- FAIL: TestLive_HuggingFaceToolChoices/none
  ```

* **On a tree copy, twice:** exit 0; `required` and `none` passed both
  times.
* **Not done:** `none-after-a-call`. On Groq, `none` after a tool turn is
  refused whatever the request holds (the amendment "`none` after a tool
  turn on Groq"), so no subtest asserts it. The package comment names the
  limit.

**2.5, the gate,** all exit 0:

* G1: `make pre-add-check`, "396 file(s) clean (gofmt, golangci-lint, go
  vet, go test, govulncheck)";
* G2: `go vet` for darwin, linux and windows; G3: `go vet -tags
  live_gateways ./...`;
* G4: 26 packages ok with `-race -cover`, and `go test -race -shuffle=on
  ./...` exit 0; `llmprovider` 97.9% against its 95.9% floor,
  `providers/huggingface` 96.2%;
* G5: `go mod tidy -diff` clean;
* G6: `make lint`, 0 issues on both builds;
* G7:
  * parity: 409 identifiers, 0 problems;
  * dep-check: go.mod and 27 packages on both builds, 0 problems;
  * coverage-check: 27 packages, 0 problems;
  * api-check: against `v1.2.0`, 0 incompatible changes;
  * generate-check: 1 generated file, 0 problems;
  * gate-selftest: 7 tests OK in 49 s;
* G8: 0 issues on the repository's markdownlint scope;
* G9: stable;
* G10: 0 problems, 585 relative links in 61 files;
* G11: 0 hits in 4 files, for the local account name, the hostname and its
  domain, and machine home paths.

The scratch helpers behind G4's floor, G10 and G11 were lost when the
session's scratch directory was pruned. They were rebuilt, and each was seen
to fail on planted input before this run:
* the floor: 90.0% against 95.9% gives exit 1;
* the links: a dead link gives exit 1;
* the scan: a planted account name, hostname and home path give 3 hits and
  exit 1;
* each passes its clean input.

**V4, the diff** against `6687ff8`, counted before this record was written:

```text
.../0023-MADR-huggingface-tool-choice-none.md      | 50 ++++++++++++-
.../0023-PLAN-huggingface-tool-choice-none.md      | 87 +++++++++++++++++++++-
llmprovider/live_huggingface_test.go               | 13 +++-
llmprovider/providers/huggingface/huggingface.go   |  5 +-
4 files changed, 149 insertions(+), 6 deletions(-)
```

`huggingface.go`'s change is its package comment only.

### Close-out (2026-10-06)

* **The Goal,** item by item:
  * the red test failed on `HEAD` and passes (phase 1);
  * the gate was clean at the end of each phase;
  * `TestLive_HuggingFaceToolChoices` passes live, `none` included, pinned
    to Groq;
  * `none` after a tool call is measured live, and **not pinned**: Groq
    refuses it whatever the request holds. The owner chose to record the
    limit rather than assert it (the second deviation);
  * the execution record holds the output.
* **Status:** `complete`, with that item recorded as not done and why;
  `docs/README.md` says so. The MADR's two amendments carry the facts.
* **Release:** the fix is in `6687ff8`; this phase adds a comment and a
  live test. A patch release, `v1.2.1`, is the owner's to tag.
