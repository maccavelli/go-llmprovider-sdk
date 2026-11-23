---
status: accepted
date: 2026-11-23
decision-makers: repository owner
consulted: "0081-MADR-a-cancelled-sign-in-saves-nothing.md (row S78 and its D4 ruling) and 0081-PLAN-a-cancelled-sign-in-saves-nothing.md (the follow-ups it lists after pass 1 and pass 2), read only; this repository at 6ace750, read only: wizard/configure.go, wizard/describe.go, wizard/save_at_end_0081_test.go, scripts/check_tables.py, docs/architecture.md, llmprovider/catalog/discovery.go; the coordinator's brief of 2026-11-23, which assigns this number and names the three follow-ups; the owner's answers of 2026-11-23 to the four questions, relayed by the coordinator (labels quoted below)"
informed: gobble-cli and prepare-commit-msg, which drive wizard.ConfigureLLM and wizard.ResolveCredential; anyone who configures Ollama through the wizard
---

<!-- markdownlint-disable MD013 MD024 -->

# A Cancelled Run Does Not Ask About the Endpoint: `resolveBaseURL` Reads the Run's Context Before It Warns, and Two Test Gaps From 0081 Are Closed or Left

Associated PLAN: [0090-PLAN-a-cancelled-run-does-not-ask-about-the-endpoint.md](0090-PLAN-a-cancelled-run-does-not-ask-about-the-endpoint.md)

## Context and Problem Statement

0081-MADR-a-cancelled-sign-in-saves-nothing.md made a run whose context ends
save nothing, and its D4 ruling ("Suppress every listing warning once the
context has ended (Recommended)") keeps the wizard from telling a person about
a built-in catalog nobody will use (`notifyListing`, `wizard/configure.go`
at 6ace750). That ruling names listing warnings only. Its row S78 found one
more place where a run that has ended still talks, and left it as a follow-up
for the owner. Two more follow-ups came from 0081's reviews; they are test and
gate gaps, not behaviour. This record holds all three so that one review
settles them. (0081-PLAN's list had a fourth, the stale comment at
`wizard/auth_test.go:585`; at 6ace750 that comment reads correctly
(`wizard/auth_test.go:586` to `591`), so it is closed and is row T16.)

### S78: the endpoint question after a cancel

`resolveBaseURL` (`wizard/configure.go:806`) asks for an endpoint when a
provider supports one. A provider whose descriptor sets `IsLocal` (Ollama is
the only one) is probed: `catalog.ValidateOllamaURLWith(ctx, url, ...)`
(line 855) does one `GET <url>/api/version` under a 5-second bound derived
from the run's context. When the probe fails the wizard shows "cannot reach
<url> (<class>)", or "<url> answered: <class>, HTTP <status>" for a status
other than 200 (lines 861 to 866), and asks "Try a different endpoint?"
(line 867).

If the run's context has ended, by a cancel or a deadline, while the probe is
in flight, or before it starts because the endpoint question's answer arrived
as the cancel did, the probe fails at once with the context's error.
`describe` turns that into "cancelled" or "timed out" (`wizard/describe.go:38`
to `41`), so the person who pressed Ctrl-C reads "cannot reach
`http://localhost:11434` (cancelled)" and is asked whether to try another
endpoint. Answering yes ends the run at the top of the loop (line 836) with
"enter base URL: context canceled"; answering no returns the endpoint and the
next step ends the run. Either way the run ends as 0081 requires; the
question is a wrong thing to put in front of someone who already stopped. The
same ended context under `Options.NoEndpointPrompt` (line 830) has nothing to
ask: it returns "wizard: cannot reach <url>: context canceled", which still
matches `context.Canceled` but says the endpoint is unreachable.

0081's PLAN lists this in its execution record as found and not changed. Its
state table row S78 is `out of scope` because the owner's ruling covers
listing warnings only; "whether it should change is the owner's to decide in
a later record". This is that record, and S78 is its question.

### P-20: the "kept" assertion

`TestSaveNotesInTheArchitecture_0081` (`wizard/save_at_end_0081_test.go:679`)
pins what the architecture note's "When a session is saved" bullet says. One
of its five checks is `{"the premise was measured kept", "kept"}`: it passes
if the four letters appear anywhere in the bullet. The sentence it means to
pin, "The earlier grant was measured kept on 2026-11-19, that is, a second
sign-in did not end the first session", could be deleted and the check would
pass on any other use of the word. The neighbouring checks (the date, the
sentence about the device sign-ins) are specific; this one is not.

### P-16: test-only environment variables

`TestKilledAtTheModelQuestionKeepsTheStore_0081` re-runs the test binary as a
child and hands it two variables, `SDK_TEST_0081_STORE` and
`SDK_TEST_0081_SIGNAL` (`wizard/save_at_end_0081_test.go:531`). No gate sees
them: `scripts/check_tables.py` checks only `LLMPROVIDER_LIVE_*` variables
that live tests read against the names `AGENTS.md` gives, so a test-only
variable can be added or dropped with no document or gate noticing. There are
six such `*_TEST_*` variables in the test files at 6ace750, each a string
constant read through its name (the parent and the child of a test share one
constant, so a rename moves both): `SDK_TEST_0081_STORE`,
`SDK_TEST_0081_SIGNAL`, `WIZARD_TEST_PTY_CHILD_0087` and
`WIZARD_TEST_PTY_DIR_0087` (`wizard/late_lf_0087_windows_test.go:27`),
`LLMPROVIDER_TEST_PROXY_CHILD` (`llmprovider/providers/proxy_test.go:20`) and
`LLMPROVIDER_TEST_CLAUDE_KEY_HELPER`
(`llmprovider/providers/claude/reauth_test.go:21`). No document names any of
them, and no 0087 record names 0087's two. Because they are read through
constants, a gate must scan string literals in test files, not `os.Getenv`
call sites. The real hazard is not a rename but a variable left set in a
developer's shell: it makes the parent test run the child branch
(`wizard/save_at_end_0081_test.go:558`), so the test runs the wrong code and
may pass or hang. Naming the variables documents them but does not stop
that; only a test that refuses to run when its variable is already set in
the parent does.

### The release

v1.32.0 is the newest tag. 0088 and 0089 are the next release (the owner's
answer to 0088-MADR OQ18, "after" 0089). At 6ace750 `CHANGELOG.md`'s
`[Unreleased]` carries 0089's entries, which add exported names, so that tag
is a minor, v1.33.0, by the changelog's rule that a release that is not
major only adds to the exported API (`CHANGELOG.md:5` to `7`); 0088's PLAN is
in progress and its entries are not there yet. S78's change is wizard-only
and adds no exported name, so by itself it is patch-level (the repository
has patch tags, such as v1.7.1); the gaps are test and gate changes.

## Decision Drivers

* A run the person has ended must not ask them anything (0081-MADR's reading
  of "cancelled saves nothing", applied to the one question that remained).
* The fix reads the run's context, not the error's class: the probe's own
  5-second bound is an outage with the run alive, and is told as today, as
  0081-MADR D4 did for the listing's own bound.
* No exported name changes (the standards guide's R48; `make api-check`).
* A new check is unused until seen to fail on a planted break (the owner's
  global rule), so each decided row names its plant.
* The release (v1.33.0, 0088 and 0089) waits on 0089's phases; the owner
  chose to let this record lengthen that wait (question 2).

## Considered Options

Per question; the owner's rulings are under "Owner rulings".

* **S78 A: stop at once, on both paths.** After a failed probe, if the run's
  context has ended, return its error and show nothing, in the interactive
  loop and under `NoEndpointPrompt`.
* **S78 B: stop at once, interactive loop only.** The same, but
  `NoEndpointPrompt` keeps "wizard: cannot reach <url>: <err>".
* **S78 C: leave it, and say so.** No code change; the architecture note
  states that a run that ends during the endpoint probe may show the
  unreachable notice.

## Decision Outcome

Chosen option: "S78 A: stop at once, on both paths", the owner's ruling
"Stop at once, both modes (Recommended)" (2026-11-23), because a run that has
ended must not ask the person anything and the guard reads the run's context,
not the probe's own bound. The other three questions are decided in "Owner
rulings" below: the fix rides in v1.33.0; the "kept" check matches its whole
sentence; the names of the six test variables are gated and each child
branch refuses to run unless its parent launched it. No row is open.

### Consequences

* A: two small edits in `resolveBaseURL` (one guard after each failed probe);
  one existing behaviour changes in text only: under `NoEndpointPrompt`, an
  ended run's error no longer begins "wizard: cannot reach"; it is
  "wizard: endpoint check ended: <context error>" (T7), and still matches
  `context.Canceled` or `context.DeadlineExceeded`. No known caller is
  affected: prepare-commit-msg's `--yes` path does not use the wizard
  (`internal/ui/setup.go:155` to `160`) and its `ConfigureLLM` call sets no
  `NoEndpointPrompt` (`:274` to `287`); gobble-cli sets it
  (`llm/provider/signin.go:357`) only for key providers, never Ollama; neither
  repository matches the text "wizard: cannot reach" (those two reads are the
  coordinator's, relayed; gobble's `NoEndpointPrompt` line and the absence of
  the text in prepare-commit-msg were re-checked here).
* B: one edit; the no-prompt mode keeps "cannot reach" for an ended run, so
  the two modes word one cancel two ways. No known caller is affected either.
* C: no risk, and the person still sees a question after Ctrl-C.

### Confirmation

Each decided row names its test and a plant (one production change that turns
it red, run on a scratch clone). The PLAN's exit criterion: no blocker, every
row tested, plants caught.

## Pros and Cons of the Options

### S78 A

* Good, because no run that has ended speaks, on either path, by one rule the
  listing already follows (`notifyListing`).
* Good, because the error a caller reads is the context's, which is what a
  caller of a cancelled run tests for.
* Bad, because `NoEndpointPrompt`'s error text changes where a caller might
  match on it; neither known caller does, and the `errors.Is` match is
  unchanged. Unknown callers are not searched.

### S78 B

* Good, because it changes only what a person sees.
* Bad, because two paths then answer one ended context in two ways, and the
  `NoEndpointPrompt` error stays misleading ("cannot reach" for a cancel).

### S78 C

* Good, because nothing changes and nothing can regress.
* Bad, because the wrong question stays, and the architecture note must
  document a quirk the SDK could remove in two lines.

## State table

Status words: `decided`, `out of scope`, `open`. "Question" names the owner's
question that decides an open row. Tests marked "new" are written in the
PLAN's Phase 1; the line numbers are at 6ace750.

| Row | Input or state | Status | Outcome | Decided by | Test, and its plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| T1 | Interactive, `IsLocal`, the run alive, the endpoint unreachable | decided | as today: "cannot reach <url> (<class>)", then "Try a different endpoint?" | 0081-MADR D4's reading (the run is alive; given) | new `TestResolveBaseURL_UnreachableAlive_0090` (green at the base). Plant: the new guard made unconditional, which asks nothing |
| T2 | Interactive, `IsLocal`, the run cancelled while the probe is in flight, or by the time the answer arrives | decided | nothing shown; the context's error returned as "enter base URL: <ctx err>" | owner, question 1: "Stop at once, both modes (Recommended)" | new `TestResolveBaseURL_CancelledProbe_0090`, two subtests (the cancel while the probe is in flight; `Input` cancelling as it returns): the error matches `context.Canceled`, no `Notify`, no `Confirm`. The test's transport fails the request whether or not the context is done: a custom `RoundTripper` is still called with a done context, and one that answered 200 would make the case T10. Plant: the guard removed |
| T3 | The same, the run's deadline passing | decided | as T2, with `context.DeadlineExceeded` | owner, question 1: "Stop at once, both modes (Recommended)" | new `TestResolveBaseURL_DeadlineProbe_0090` (`context.DeadlineExceeded`). Plant: as T2 |
| T4 | The run already ended when the loop starts (before `Input` is asked) | decided | as today: "enter base URL: <ctx err>", the probe never runs | `wizard/configure.go:836` (given) | new pin `TestResolveBaseURL_EndedBeforeAsk_0090` (green at the base). Plant: the loop-top check removed |
| T5 | Interactive, the run alive, the server answers a status other than 200 | decided | as today: "<url> answered: <class>, HTTP <status>", then the question | `wizard/configure.go:861` (given) | new `TestResolveBaseURL_AnsweredAlive_0090` (green at the base). Plant: as T1 |
| T6 | `NoEndpointPrompt`, `IsLocal`, the run alive, the endpoint unreachable | decided | as today: error "wizard: cannot reach <url>: <err>", nothing asked | `wizard/configure.go:830` (given) | new `TestResolveBaseURL_NoPromptUnreachableAlive_0090` (green at the base). Plant: the guard made unconditional |
| T7 | `NoEndpointPrompt`, `IsLocal`, the run ended during the probe, whether the probe failed to connect or the server answered a status other than 200 | decided | nothing shown; "wizard: endpoint check ended: <context error>" (nothing was entered, so not "enter base URL"); it matches the context's error | owner, question 1: "Stop at once, both modes (Recommended)" | new `TestResolveBaseURL_NoPromptEnded_0090`, subtests for a connect failure and for a non-200 answer: matches the context's error, the text above, never "cannot reach". Plant: the guard removed |
| T8 | The probe's own 5-second bound passes, the run alive | decided | told as an outage (`describe` says "timed out"), then the question, as today: the guard reads the run's context, never the probe's bound | 0081-MADR D4 (the listing's bound, given) | new `TestResolveBaseURL_ProbeBoundAlive_0090` (a client whose transport returns `context.DeadlineExceeded` with the run alive). Plant: the guard reads the error's class instead of the run's context |
| T9 | A remote provider's endpoint (`validateBaseURL`, no probe, no network) | out of scope | no request is made, so there is nothing for a cancel to interrupt | the remote branch is claim E3 and calls `validateBaseURL`, which makes no request (given) | none |
| T10 | The probe answered 200 and the run ended as it returned | decided | the endpoint is returned; the next step's context check ends the run, as 0081 requires | 0081-MADR D4, S1 (given) | new `TestResolveBaseURL_EndedAfterAnswer_0090` (a transport that answers 200 and cancels). Plant: the guard placed before the probe's success check |
| T11 | Which release carries this | decided | rides in v1.33.0 with 0088 and 0089: the PLAN is built after the SDK's merged output-limit phase in the same tree, serially, and the v1.33.0 tag waits for it; a Fixed entry joins `[Unreleased]` | owner, question 2: "Inside v1.33.0" | none (a release step) |
| T12 | P-20: the "kept" assertion matches the word anywhere in the bullet | decided | the check wants the whole sentence "was measured kept on 2026-11-19, that is, a second sign-in did not end the first session" | owner, question 3: "Match the whole sentence (Recommended)" | the changed check in `TestSaveNotesInTheArchitecture_0081`; plant: the sentence removed and another "kept" added to the bullet |
| T13 | P-16: the six test-only `*_TEST_*` variables (see above), named by no document and not covered by `scripts/check_tables.py`; and a variable left set in a developer's shell sending the parent test down the child branch | decided | `scripts/check_tables.py` requires each `*_TEST_*` string literal in a test file to be named, as a whole word, by `AGENTS.md` or by a record numbered in its name; and each of the four test files' child branch refuses to run unless its parent launched it (a value only the parent writes), so a variable left set in a shell is ignored | owner, question 4: "Gate names + close the hazard" | new `scripts/test_gates.py` cases and four parent-run tests (a stray value in the parent runs the parent branch), red at the base; plants: a variable declared in a test and named nowhere; the parent check removed in each file. |
| T14 | The architecture note says what a run that ends during the endpoint probe does | decided | one sentence added to the endpoint passage of `docs/architecture.md` | owner, question 1: "Stop at once, both modes (Recommended)" | `make docs-check`; a `tools/claims.py` row on the sentence, planted false |
| T15 | Interactive, `IsLocal`, the server answers a status other than 200 and the run ends (cancel or deadline) before or as the answer is read | decided | nothing shown and the context's error returned, as T2 (a guard only in the "cannot reach" branch would miss this one, since the "answered" branch is separate) | owner, question 1: "Stop at once, both modes (Recommended)" | new `TestResolveBaseURL_AnsweredEnded_0090`. Plant: the guard placed only in the connect-failure branch |
| T16 | 0081-PLAN's fourth follow-up, the stale comment at `wizard/auth_test.go:585` | out of scope | closed: at 6ace750 the comment reads correctly (`wizard/auth_test.go:586` to `591`) | read at 6ace750 | none |

Counts: 16 rows; **14 decided** (T1 to T8, T10 to T15; T2, T3, T7, T11 to T15
decided by the owner's rulings of 2026-11-23), **2 out of scope** (T9, T16),
**0 open**.

## Evidence index

<!-- claims: self=6ace750 -->

| Id | What it shows | Location | Text |
| :--- | :--- | :--- | :--- |
| E1 | The no-prompt probe | `wizard/configure.go:830` | `catalog.ValidateOllamaURLWith(ctx, def, check...)` |
| E2 | The loop-top context check | `wizard/configure.go:836` | `if err := ctx.Err(); err != nil {` |
| E3 | The remote-provider branch, which makes no probe | `wizard/configure.go:843` | `if !d.IsLocal {` |
| E4 | The interactive probe | `wizard/configure.go:855` | `catalog.ValidateOllamaURLWith(ctx, url, check...)` |
| E5 | The "answered" branch | `wizard/configure.go:861` | `var answered *llmprovider.APIError` |
| E6 | The question asked after a failed probe | `wizard/configure.go:867` | `p.Confirm("Try a different endpoint?", true)` |
| E7 | A cancelled run's wording | `wizard/describe.go:40` | `case errors.Is(err, context.Canceled):` |

## Owner rulings (2026-11-23)

Relayed by the coordinator; labels verbatim.

1. **"Stop at once, both modes (Recommended)"** (T2, T3, T7, T14, T15). A run that has ended during the Ollama endpoint check shows nothing and returns the context's error, in the prompting loop and under `NoEndpointPrompt`.
2. **"Inside v1.33.0"** (T11; not the recommended option). The fix rides in v1.33.0 with 0088 and 0089. Its build follows the SDK's merged output-limit phase in the same tree, serially, and the v1.33.0 tag waits for it. Cost accepted: the tag waits for this record's build, docs and review, and `[Unreleased]` gains a Fixed entry.
3. **"Match the whole sentence (Recommended)"** (T12). The "kept" check matches "was measured kept on 2026-11-19, that is, a second sign-in did not end the first session".
4. **"Gate names + close the hazard"** (T13; not the recommended option). The gate on the six variables' names, and each of the four test files' child branch refuses to run unless its parent launched it. Cost accepted: the widest scope, edits to four existing test files, a test and a plant each.

## More Information

* The rule for T2, T3 and T7 mirrors `notifyListing` (`wizard/configure.go`,
  "Suppress every listing warning once the context has ended"): the run's
  context, never the bound of the thing that failed.
* A cancel while `p.Input` itself is waiting is not this record's: the
  prompter returns its own error (`enter base URL: ...`), as today.
* No exported name, dependency or provider behaviour changes in any option.
