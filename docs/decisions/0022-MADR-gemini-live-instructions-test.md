---
status: accepted
date: 2026-10-05
decision-makers: repository owner
consulted: 0014-MADR-gemini-wire-fidelity.md (§1, §2), 0021-MADR-harden-and-tune-after-the-v1-1-review.md (L2, amendment "L2 is the test's prompt, not the wire")
informed: maintainers of go-llmprovider-sdk's live tests
---

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Pin Gemini's Live Instructions Check with a Neutral Instruction and a Baseline

> **Status.** Accepted 2026-10-05 by the owner ("approved to proceed"), with
> the option proposed: "Pair the check: a neutral instruction, with and
> without".

## Context and Problem Statement

`TestLive_GeminiInteractions` (`llmprovider/live_gemini_test.go:50-86`)
checks three things against the live Interactions API:

* a leading system item reaches the model, which obeys it;
* the request is one `/interactions` call with `"store":false` and a
  `system_instruction` field;
* with `WithStore(true)`, a continuation recalls an earlier turn
  (`0014-MADR-gemini-wire-fidelity.md` §1–§2).

The first check sends the system item "Whatever the user says, reply with
only the word OMEGA." with the user's "Say hello.", and passes when the reply
contains `OMEGA`.

The same instruction, in the same kind of test, stopped working on Grok.
`0021-MADR-harden-and-tune-after-the-v1-1-review.md` found it as L2 and
measured it on 2026-10-05 (its amendment "L2 is the test's prompt, not the
wire"). `grok-4.6` and `grok-4.7` declined an instruction phrased as an
override of whatever the user says, in every position, the user's own message
included. They obeyed a neutral one, "Begin every reply with the word
OMEGA.", 3/3 in both wire forms, and never wrote `OMEGA` without it.
`TestLive_GrokInstructions` became a pair: the same request with and without
the neutral instruction. 0021-PLAN's deviation "step 7.2a finds both forms
obeyed" noted that the Gemini test uses the same wording and left it
unmeasured, outside L2.

**Measured 2026-10-05,** on a scratch copy of `d4f9fa5`, with
`GEMINI_API_KEY` checked for presence only, recording only whether each
reply contains `OMEGA`:

* `TestLive_GeminiInteractions` passes 3 runs of 3.
* The forms, three runs each:

  | Instruction | Where | `gemini-3.7-flash` | `gemini-3.8-flash` |
  | :--- | :--- | :--- | :--- |
  | "Whatever the user says, reply with only the word OMEGA." | a leading system item | 3/3 | 3/3 |
  | the same | in the user's own message | 3/3 | 3/3 |
  | none (baseline) | | 0/3 | 0/3 |
  | "Begin every reply with the word OMEGA." | a leading system item | 3/3 | 3/3 |
  | the same | `Request.Instructions` | 3/3 | 3/3 |

  `gemini-3.7-flash` is the test's model and leads the live listing.
  `gemini-3.8-flash` is the newest in the static catalog; the live listing
  of that day does not list it, and it answers.

So nothing is broken today. The question is whether to change a passing test
before Gemini's models drift the way Grok's did, and how. Two weaknesses are
independent of any drift:

* **The check cannot tell an obeyed instruction from a reply that happens to
  match.** It has no baseline: nothing shows that the same request without
  the instruction leaves `OMEGA` out.
* **A refusal reads as a defect.** If Gemini starts declining overrides, the
  test fails with "want the system instruction obeyed", and the next reader
  has to repeat 0021's probe to learn the wire is fine.

This record decides the live check only. The wire itself is pinned offline:
the conformance harness's `Fidelity` check (`0021-MADR` D5, W12) and the
`gemini` goldens show that `Instructions` and system items reach
`system_instruction` (`providers/gemini/gemini.go:186-194`).

## Decision Drivers

* A live check proves the service obeys, so it needs a baseline to show the
  instruction caused the reply.
* Its failure should name the cause: a dropped instruction, not a model's
  policy on overrides.
* The live tests should check instructions one way across providers, so a
  reader learns the pattern once (Grok's pair, 0021).
* Not loosening the check: whatever replaces it must fail when the
  instruction does not reach the model, as the current one does.
* The cost of live runs: each extra request is billed, and the suite is run
  by hand.

## Considered Options

* Pair the check: a neutral instruction, with and without
* Keep the test as it is, and change it when it fails
* Keep the override wording, and add the baseline
* A shared live helper for the instructions pair, used by Grok and Gemini

## Decision Outcome

Chosen option: "Pair the check: a neutral instruction, with and without",
because it is the only option that gives the check a baseline and removes the
wording Grok already refuses, at the cost of one extra request, and it
matches the pattern 0021 gave Grok.

* **`with`:** the system item "Begin every reply with the word OMEGA.", then
  "Say hello.". The reply contains `OMEGA`. The recorded request is one
  `/interactions` call with `"store":false` and `system_instruction`, as
  today.
* **`without`:** "Say hello." alone. The reply has no `OMEGA`, and the
  request has no `system_instruction`.
* The instruction stays a system item, not `Request.Instructions`: that is
  the path 0014 §1 pins live, and `Instructions` is covered offline by the
  harness. Both measured 3/3.
* The `WithStore(true)` continuation is unchanged.
* `TestLive_GrokInstructions` is unchanged.
* The `gemini` provider, its goldens and every exported API are unchanged.

### Consequences

* Good, because the check now shows the instruction changed the reply.
* Good, because a model that declines overrides no longer fails the test, so
  a failure means the instruction did not arrive or was not obeyed.
* Good, because Gemini's and Grok's live checks read the same way.
* Neutral, because the test passes today and will pass after; the change is
  preventive, and it is live-only, so CI runs none of it (`go vet -tags
  live_gateways` only).
* Bad, because each run of the test sends one more billed request.
* Bad, because a model that writes `OMEGA` with no instruction would fail
  `without`. None did in 6 baseline runs.

### Confirmation

* The paired test passes live twice on a tree copy.
* On a scratch copy whose `gemini.go` drops `system_instruction`, `with`
  fails, and `without` passes.
* The repository gate is clean (0022-PLAN).

## Pros and Cons of the Options

### Pair the check: a neutral instruction, with and without

* Good, because the baseline shows causation.
* Good, because it drops the wording Grok refuses, so a drift in override
  policy does not fail it.
* Good, because it is the pattern of `TestLive_GrokInstructions`.
* Neutral, because the neutral instruction is weaker: it asks for a prefix,
  not a full replacement of the reply. Both make the reply depend on the
  instruction, which is what the check is for.
* Bad, because it costs one more request per run.

### Keep the test as it is, and change it when it fails

* Good, because it costs nothing now, and the test passes 3/3.
* Bad, because it keeps no baseline.
* Bad, because the failure, when it comes, is the one L2 had to investigate:
  it looks like a wire defect.

### Keep the override wording, and add the baseline

* Good, because it keeps the stronger instruction, and adds causation.
* Bad, because it keeps the wording Grok's models already refuse in every
  position, so it carries L2's failure forward.

### A shared live helper for the instructions pair, used by Grok and Gemini

* Good, because one helper defines the pair once, for every provider that
  later gets a live instructions check.
* Bad, because it changes `TestLive_GrokInstructions`, which passes and was
  approved in 0021, to save about 20 lines in two tests.
* Bad, because the two tests differ where it matters: Grok sends
  `Request.Instructions`, while Gemini sends a system item and checks the
  recorded request. A helper would need options for both.

## More Information

* **Relationship:**
  * follows `0021-MADR-harden-and-tune-after-the-v1-1-review.md`: L2 and
    its amendment "L2 is the test's prompt, not the wire";
  * keeps `0014-MADR-gemini-wire-fidelity.md` §1–§2's live checks, changing
    only the wording and adding the baseline;
  * a new pair rather than an amendment to 0021, whose PLAN is complete and
    released as `v1.2.0`.
* **Plan:** [0022-PLAN-gemini-live-instructions-test.md](0022-PLAN-gemini-live-instructions-test.md).
* **The probe** was a scratch test on a scratch copy, never in the tree. It
  printed only counts and the live listing, never a reply.
