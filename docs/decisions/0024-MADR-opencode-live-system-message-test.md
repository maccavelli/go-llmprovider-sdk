---
status: accepted
date: 2026-10-06
decision-makers: repository owner
consulted: 0022-MADR-gemini-live-instructions-test.md, 0012-MADR-conform-providers-to-reference-clients.md (§2), 0021-MADR-harden-and-tune-after-the-v1-1-review.md (L2)
informed: maintainers of go-llmprovider-sdk's live tests
---

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Pair OpenCode's Live System-Message Check with a Neutral Instruction and a Baseline

> **Status.** Accepted 2026-10-06 by the owner ("proceed to 0024"), with the
> option proposed: "Pair the check: a neutral instruction, with and without".

## Context and Problem Statement

`TestLive_OpencodeSystemMessage` (`llmprovider/live_opencode_test.go:342-360`)
checks that a system item reaches the model on OpenCode Go's messages route,
and that the model follows it. It sends the system item "You only ever reply
in French." with the user's "Say hello in one word, nothing else.". It passes
when the reply contains `bonjour`, `salut` or `coucou`. The picker chooses
`qwen3.8-flash`, which OpenCode Go routes to `/messages`. There the provider
sends the system text as the top-level `system` field
(`providers/opencode/opencode.go:352-353`).

The instruction conflicts with the user's request, which is in English. It
is the same shape as the override wordings that 0021's L2 and
`0022-MADR-gemini-live-instructions-test.md` replaced, and the check has no
baseline.

**Measured,** with `OPENCODE_API_KEY` checked for presence only, on scratch
copies of `HEAD`:

* **0021-PLAN's phase 2 live run (2026-10-04):** the test failed once with
  the reply "Hello", then passed 2 of 2. It was recorded as flaky and not
  reproduced.
* **2026-10-06, `64b82d5`:** the test passed 2 runs of 3. In the probe,
  three runs each unless stated, the forms were:

  | Instruction | Where | Followed |
  | :--- | :--- | :--- |
  | "You only ever reply in French." | a system item (today's) | 5/6, then 10/10 |
  | the same | `Request.Instructions` | 3/3 |
  | none (baseline) | | 0/3 |
  | "Begin every reply with the word OMEGA." | a system item | 3/3 |
  | none (baseline) | | 0/3 |

  Today's form followed the instruction 17 times in 19, counting the test's
  own runs. In the 10-run batch, every reply contained one of the three
  listed words, so the misses are replies not in French at all, not French
  greetings the test does not recognise.

So the check fails about one run in ten, because the model sometimes takes
the user's English over the system's French. A failure then reads as "the
system instruction did not arrive", which the wire does not cause: the
instruction reached the model in every run the probe counted. As in 0022,
the check also cannot tell an obeyed instruction from a reply that happens
to match.

## Decision Drivers

* A live check fails only when the thing it checks is broken: here, the
  system item not reaching, or not steering, the model.
* A baseline shows that the instruction caused the reply.
* One pattern across providers' live instruction checks: Grok's (0021) and
  Gemini's (0022) are pairs with "Begin every reply with the word OMEGA.".
* The route matters: this is the live check of the messages route's
  top-level `system` field.
* Not loosening: whatever replaces it must fail when the system text does
  not reach the model.

## Considered Options

* Pair the check: a neutral instruction, with and without
* Keep French, and add the baseline
* Keep French, and accept more replies, or retry once
* Leave the test as it is

## Decision Outcome

Chosen option: "Pair the check: a neutral instruction, with and without",
because it measured 3/3 with and 0/3 without, it does not pit the instruction
against the user, and it is the pattern 0021 and 0022 settled on.

* **`with`:** the system item "Begin every reply with the word OMEGA.", then
  the user's "Say hello.". The reply contains `OMEGA`. The recorded request
  is one POST to a path ending `/messages`, with a top-level `"system"` key.
* **`without`:** "Say hello." alone. The reply has no `OMEGA`, and the
  request has no `"system"` key.
* The model is picked as today, `goModel(t, "qwen3.8-flash", "minimax-m3")`,
  and both subtests use the same pick.
* Unchanged: the provider, its goldens, every other live test, and every
  exported API.

### Consequences

* Good, because the check stops failing on the model's choice of language.
* Good, because the baseline and the recorded `system` key show the route
  carries the instruction and the instruction steers the reply.
* Good, because the three providers' live instruction checks read the same
  way.
* Neutral, because the test is live-only; CI vets it and never runs it.
* Bad, because each run sends one more request on the OpenCode Go
  subscription.
* Bad, because a model that writes `OMEGA` unprompted would fail `without`.
  None did in 6 baseline runs, here and in 0022.

### Confirmation

* The pair passes live twice on a tree copy.
* On a scratch copy whose `opencode.go` never sets `system`, `with` fails
  and `without` passes.
* The repository gate is clean (0024-PLAN).

## Pros and Cons of the Options

### Pair the check: a neutral instruction, with and without

* Good, because it measured 3/3 and 0/3, and checks the request too.
* Good, because the instruction does not contradict the user.
* Neutral, because asking for a prefix is a weaker instruction than a
  language. Both make the reply depend on the system text, which is what
  the check is for.
* Bad, because it costs one more request per run.

### Keep French, and add the baseline

* Good, because a language is a strong, visible instruction, and the
  baseline adds causation.
* Bad, because it keeps the conflict with the user's English, which failed
  2 runs in 19.

### Keep French, and accept more replies, or retry once

* Good, because it is the smallest edit.
* Bad, because the misses are not unrecognised French; accepting more words
  would not catch them.
* Bad, because a retry hides a real failure behind a second chance: it
  loosens the check.

### Leave the test as it is

* Good, because it costs nothing.
* Bad, because it stays flaky, about one run in ten, and keeps no baseline.

## More Information

* **Relationship:**
  * applies `0022-MADR-gemini-live-instructions-test.md`'s pattern to
    OpenCode Go's messages route;
  * keeps `0012-MADR-conform-providers-to-reference-clients.md` §2's live
    check of the `system` field;
  * the flake was first recorded in 0021-PLAN's phase 2 live run.
* **Plan:** [0024-PLAN-opencode-live-system-message-test.md](0024-PLAN-opencode-live-system-message-test.md).
* **The probe** was a scratch test on a scratch copy, never in the tree. It
  logged counts and reply categories (French and listed, English, other),
  never a reply.
