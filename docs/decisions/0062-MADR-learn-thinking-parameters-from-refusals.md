---
status: proposed
date: 2026-10-24
decision-makers: repository owner
consulted: "this repository at 1b46940 (the Code claims, run by tools/claims.py); the records 0013, 0014, 0026, 0034, 0041, 0049 and 0061 as cited; the owner's answers of 2026-10-24 to Q1 to Q4 of the thinking design's re-assessment and to this record's Q5 to Q7, and of 2026-10-25 to its Q8 to Q13"
informed: every consumer of llmprovider and its providers; gobble-cli and prepare-commit-msg
---

<!-- markdownlint-disable MD013 -->

# Decide Thinking Parameters From Facts and Learned Refusals, Not Model Names

## Context and Problem Statement

A request that reasons carries up to three decisions about its model: the
thinking shape (adaptive or a budget on Messages; a level or a budget on
generateContent), the effort in the wire's vocabulary, and the budget.
0041-MADR-decide-per-model-behaviour-from-metadata.md D1 orders where they
come from: the caller's layers, the provider's listing, models.dev, and last a
name rule. Where no fact answers, the name rule decides: for a model released
after the SDK, a gateway model models.dev omits, or any provider with metadata
off. Where a fact or a rule is wrong, every request fails the same way until a
release changes the code. The SDK has no feedback loop.

Each name rule began as a measured HTTP 400 that was then written into code.
0061-MADR-consumer-gaps-from-gobble-issue-9.md D3 adds more rows of the same
kind for `EffortMax`. Its PLAN's Phase 3 was committed as `dd41faf` on
2026-10-24, after the owner had chosen, by Q2 (b) below, to hold it (D10 and
the amendment "0061's Phase 3 committed").

On 2026-10-24 the owner asked to stop hard-coding around the differences
between providers and models, and to decide the parameters from what the
services say. This record decides how.

Every fact below was read in this repository at `1b46940` (the Code claims,
run by `tools/claims.py`) or in the records it cites. What no record measured
is marked UNCONFIRMED, and the PLAN's Phase 1 measures it before any code
depends on it.

### Findings, verified

* **F1. The name rules that decide thinking.**

  | Rule | Decides | Claim |
  | :--- | :--- | :--- |
  | R1 `claudeAdaptiveOnly` | adaptive or a budget on Messages, from the version in a `claude-` id: 4.7 and later adaptive | C1, C3 |
  | R2 `minimaxAdaptiveThinking` | adaptive with no effort for a `minimax-m3` id | C4 |
  | (R1 and R2 applied) | both decide only when the facts do not | C5 |
  | R4 `geminiLegacyRE` | `thinkingBudget` or `thinkingLevel` on generateContent | C8, C9 |
  | R5 `thinkingLevel` | Gemini's levels; `gemini-2.5-flash-lite` gets `high` or nothing | C10, C11 |
  | Gemini's `xhigh` | sent as `high` | C12 |
  | R6 `effortMenu` | Grok's efforts, for five ids | C13, C14 |
  | Ollama's `xhigh` | sent as `"max"` | C15, C16 |
  | OpenCode's chat route | `reasoning_effort` only when the metadata lists it; none without metadata | C17 |

  The R numbers are 0041-MADR §2's. R3, the fixed budgets of 4096 and, for
  `low`, 1024 (C6, C7), is a default for a request with no budget, not a
  model rule, and the facts already bound it (0041-MADR D2).
* **F2. Each rule records a refusal someone measured.** R1: HTTP 400 on
  `claude-sonnet-5` and `claude-opus-4-8` for `thinking.type` `"enabled"`
  (C2; 0013-PLAN's live check, C35). R4: `thinkingLevel` is HTTP 400 on
  Gemini 2.x (C36). R5: `gemini-2.5-flash-lite` "refuses medium and fails on
  low" (C10).
* **F3. The records keep a refusal's status, not its body.** 0013-PLAN
  records "claude HTTP 400" (C35). No record holds an error's code or message
  for a thinking refusal. **UNCONFIRMED:** whether each service's refusal
  names the refused parameter or value, whether it lists the accepted
  values, and whether a refused request is billed.
* **F4. A wrong value may be accepted and ignored.** Hugging Face documents
  `reasoning_effort` as "provider and model-dependent", so an ignored effort
  is a normal outcome (C18). **UNCONFIRMED** for the other services.
  `Usage.ReasoningTokens` (C32) cannot tell an ignored effort from a low one,
  so learning cannot see an accepted-and-ignored value.
* **F5. The learning machinery exists, for forced tool choice**
  (0034-MADR-degrade-forced-tool-choice-per-model.md D3):
  * a memo of refusals per `catalog.Cache`, or the process's without one
    (0049-MADR D3), bounded at 256 and keyed by provider and model
    (C19 to C21);
  * a matcher that reads the `*APIError` (C22);
  * one resend through `wire.Send`, and on a stream before its first event
    (C23, C24, C27);
  * a warning that names what a caller keeps and passes back with
    `WithModelForcedChoice` (C25, C26).

  Only Claude and OpenCode send through `wire.Send` (C27, C28). Every other
  provider calls `wire.Post` inside `wire.Reauth` (C29).
* **F6. An `*APIError` keeps the service's code and a message redacted and
  bounded to 512 bytes** (C30, C31). No field holds an error's `param`.
* **F7. Probes are billed.** The health probes are on by default and billed
  (C33). A thinking probe that is accepted reasons, and the reasoning is
  billed.
* **F8. Facts never wait.** A request reads facts only from the caches
  (C34), and a caller can add facts with a layer (C37).
* **F9. `Effort` is a closed set** checked by `valid()` (C38). 0061-MADR D3
  adds `EffortMax` above `EffortXHigh`.

## Decision Drivers

* **No name rule** decides a model's thinking shape or vocabulary.
* **Right the first time where the facts know; corrected within the same
  request where they do not.**
* **No billed request the caller did not ask for.**
* **Bounded and deterministic.** The same facts and memo give the same body,
  and a request sends at most a fixed number of times.
* **Evidence first.** A refusal is recognised only by a matcher built from a
  recorded refusal, and tested against recorded failures of other causes.
* **Every provider and route,** on `Generate` and on `Stream`.
* **Additive API** (R48); each export with its call site (R55).
* **Learned state outlives the process only through the caller** (Q3 (a)).

## Considered Options

* **A. Facts, then learned refusals, plus an explicit probe a caller can
  run.**
* **B. Probe every model before first use,** and cache the results.
* **C. Facts only:** delete the name rules; send what the facts allow, or
  nothing.
* **D. Keep 0041's order,** with the name rules as the last source.

## Decision Outcome

Chosen option: "A. Facts, then learned refusals, plus an explicit probe a
caller can run", because:

* it removes a name rule wherever the service's refusal can be read;
* it costs nothing on a request the service accepts;
* it corrects a wrong fact or rule within the request that met it;
* it reuses 0034's memo, resend and warning instead of building a second
  mechanism.

The owner chose it on 2026-10-24 (Q1 (a)), with Q2 (b), Q3 (a) and Q4 (a).

### D1. The parameters (Q4 (a))

| Wire | Providers and routes | Shape | Effort | Budget |
| :--- | :--- | :--- | :--- | :--- |
| Messages | Claude; OpenCode's messages route | `thinking.type`: `adaptive` or `enabled` | `output_config.effort` | `thinking.budget_tokens` |
| Responses | OpenAI (Platform, ChatGPT, plan); Grok; OpenCode's responses route | none | `reasoning.effort` | none |
| Interactions | Gemini | none | `generation_config.thinking_level` | none |
| generateContent | OpenCode's google route | `thinkingLevel` or `thinkingBudget` | `thinkingConfig.thinkingLevel` | `thinkingConfig.thinkingBudget` |
| Chat Completions | Hugging Face, Together, Ollama, Kilo, OpenCode's chat route, and `chatcompat` (0061-MADR D4) | none | `reasoning_effort`; Kilo's `reasoning.effort` | none |

Outside this decision:

* forced tool choice (0034-MADR is unchanged);
* the output limit and the cache;
* Together's `reasoning.enabled` and Kilo's `WithCapabilities` gate, which
  are set per wire or declared by the caller, not decided per model.

### D2. Who decides each parameter, per request

1. **What the model refused.** The memo (D5) and the caller's passed-back
   refusals (D6). A refused value is never sent while it is remembered.
2. **The facts,** in 0041-MADR D1's order:
   * where they list efforts, only listed efforts are candidates;
   * where they name a shape (0041-MADR D2's reading of R1 to R4), only
     that shape is.
3. **Otherwise the request:**
   * the effort it asks, in the wire's vocabulary;
   * the shape it implies: a `Budget` means the budget shape; an effort, or
     neither, means the adaptive shape (Messages) or the level
     (generateContent).

No name rule decides. A wire's field set stays: a wire with no budget field
sends no budget, as today. That is a fact about the wire, not a model rule.

### D3. The ladders

* **Effort.** The order is `low` < `medium` < `high` < `xhigh` < `max`. For a
  requested effort the candidates are, in order:
  1. the effort itself;
  2. each lower effort, highest first;
  3. each higher effort, lowest first;
  4. none: the effort field is left out and the model's default applies.

  D2's step 2 removes efforts the facts do not list, and step 1 removes
  refused ones. The first candidate left is sent.
  * Rung 3 applies only when the facts leave nothing below the request. A
    request for `low` on a model that lists only `medium` and `high` gets
    `medium`. Grok's clamp sends the menu's highest there today, `high`
    (C14's function).
  * An effort other than the one asked carries 0061-MADR D3's
    `WarningEffortDegraded`, whatever moved it.
* **Shape** (Messages and generateContent). The candidates are D2's shape,
  then the other one.
  * The budget shape carries the request's budget, else 1024 for `low` and
    4096 otherwise (R3), bounded by the facts as in 0041-MADR D2.
  * The adaptive shape and the level carry the effort ladder's choice.
  * When both shapes are refused, the refusal is returned. The request asked
    for reasoning that the model will not do.

### D4. Recognising a refusal

* **Each wire package has one matcher.** It recognises an `*APIError` whose
  status, code and message, as the error keeps them, match a refusal that
  Phase 1 recorded for that wire. It returns:
  * the refused parameter (shape or effort) and its value;
  * the accepted values, where the error lists them.
* **Accepted values are learned as the model's efforts,** so the next send
  takes the ladder's first listed value instead of stepping down one rung at
  a time.
* **The matcher is tested** against every recorded refusal of its wire, and
  against every recorded 400 of another cause, which must not match.
* **A 400 the matcher does not recognise is returned as today.**
* **A wire whose refusals Phase 1 cannot tell apart from other failures
  learns nothing for that parameter.** D8 says what it does instead.
* **If a matcher needs a field the `*APIError` does not keep,** such as an
  error's `param` (F6), this record is amended before anything is exported.

### D5. Learning and resending

* **On a recognised refusal, the provider:**
  1. records the refusal in the memo;
  2. builds the request again from D2 and D3;
  3. sends it once more, through `wire.Reauth`, before any stream event, as
     `wire.Send` and `StreamEventsRetry` do (F5).

  Every provider sends this way: Gemini, Grok, Hugging Face, Kilo, Ollama,
  OpenAI and Together move from `wire.Post` to that path. Ollama moves only
  once its refusals are recorded (D8, Q9 (c)).
* **The bound on sends per request is 7 (Q5 (a)):** until the ladders run
  out, one send per effort rung, `max` down to none, and one for the other
  shape. A forced-choice refusal (0034-MADR D3) counts
  against the same bound.
* **The memo** follows 0034's: one per `catalog.Cache`, or the process's
  without one; never written to disk; at most 256 models, oldest dropped.
  * It is keyed by provider, base URL and normalised model. The base URL is
    added to 0034's key because one provider id at two base URLs (two Ollama
    hosts, OpenCode's two gateways) serves different models under the same
    id.
  * It holds refused efforts, refused shapes, and any accepted efforts an
    error listed.
* **A request that succeeds after a refusal** carries D6's warning. A
  request that still fails returns its error, and the memo keeps what it
  learned.

### D6. The warning and the pass-back (Q3 (a))

* `WarningReasoningLearned WarningKind = "reasoning_learned"`: the model
  refused a thinking setting the provider sent, and the provider learned it
  for the rest of the process.
* `type ReasoningRefusal struct { Model string; Effort Effort; Shape ThinkingShape }`:
  what a model refused. Exactly one of `Effort` and `Shape` is set.
* `type ThinkingShape string`, with `ThinkingAdaptive ThinkingShape =
  "adaptive"` and `ThinkingBudget ThinkingShape = "budget"`.
* **`Warning` gains `Refused ReasoningRefusal`.** It is set on a
  `WarningReasoningLearned` and zero otherwise, and `Warning` stays
  comparable. It is the value a caller keeps.
* `WithReasoningRefusals(refusals ...ReasoningRefusal) Option`: the caller
  passes the kept refusals back on its next start. A provider reads them
  through `func (st *Settings) ReasoningRefusals() []ReasoningRefusal`, as it
  reads `WithModelForcedChoice`'s through `ModelForcedChoices` (C41), and
  D2's step 1 reads them beside the memo. `New` refuses, with
  `ErrInvalidRequest`:
  * a refusal with no model;
  * a refusal with both or neither of `Effort` and `Shape`;
  * an unknown effort or shape.

### D7. The explicit probe (Q1 (a))

* `func ProbeReasoning(ctx context.Context, p Provider, model string) ([]ReasoningRefusal, error)`.
  * It sends one generation at `EffortMax`, with a fixed one-line prompt, and
    returns the refusals that the `WarningReasoningLearned` warnings carry.
  * Through D2 and D3, that one request also finds a shape refusal: an
    effort request on Messages starts adaptive, and steps to a budget if the
    model refuses adaptive.
  * Refused sends carry no usage (whether they are billed is UNCONFIRMED,
    F3). The accepted send is billed; on a budget model its budget is 4096
    (R3).
  * It runs whatever `WithModelProbes` says, because the caller asked for it
    explicitly.
* **No caller in this repository (Q6 (a)).** Consumers call it from their
  configure step; `wizard` does not.

### D8. The rules after Phase 1

A rule is removed when Phase 1 records a refusal that its matcher (D4)
recognises. Otherwise it stays as a wire mapping, its Phase 1 record cited in
the code, and the provider's package doc lists it under Degradations (R12).

* **Every rule is decided from a recording (Q8 (b)).** No code depends on
  Phase 1 until every provider and route of its case table is recorded. A
  case left unrun for want of a key is not a recording, and blocks the
  next phase. A model that cannot be reached is a recording: `unreachable`.
* **Ollama is recorded later (Q9 (c)).** Its cases run in the PLAN's Phase
  5, against a host and a thinking model the owner names. Until then Ollama
  sends as `dd41faf` does: no matcher, no resend, and its `xhigh` and `max`
  mappings kept. Phase 5 decides its row of the table below, as every other
  row is decided.

| Rule | Removed when Phase 1 records a recognisable refusal of |
| :--- | :--- |
| R1 `claudeAdaptiveOnly` | `thinking.type` `"enabled"` on `claude-sonnet-5`; and `"adaptive"` on a budget model: `claude-sonnet-4-5-20250929`, which retires on 2026-11-30 (C39), and `qwen3.8-flash` on OpenCode Go's messages route |
| R2 `minimaxAdaptiveThinking` | `output_config.effort` on OpenCode's messages route for `minimax-m3` |
| R4 `geminiLegacyRE` | `thinkingLevel` on `gemini-2.5-flash`, open only to past users (C40); if the owner's key cannot reach it, the rule is removed as dead, every model it serves being retired |
| R5 the flash-lite exception | `medium` on `gemini-2.5-flash-lite`, open only to past users (C40); if unreachable, removed as dead |
| Gemini's `xhigh` to `high` | `"xhigh"` as `thinking_level` |
| R6 `effortMenu` | an effort a Grok model does not take |
| Ollama's `xhigh` to `"max"` | `"xhigh"`, recorded in the PLAN's Phase 5 (Q9 (c)). If Ollama accepts and ignores it (F4), the mapping stays |
| 0061-MADR D3's rows for Grok, Gemini and OpenCode's google route | `"max"` on each, by the same criterion |
| OpenCode's chat-route gate (C17) | `reasoning_effort` on the chat route without metadata, as on every other wire (Q7 (a)) |

R3 stays (F1).

### D9. The exports and their call sites (R55)

| Export | Package | Call site |
| :--- | :--- | :--- |
| `WarningReasoningLearned` | `llmprovider` | each provider's resend path (D5) |
| `ReasoningRefusal` | `llmprovider` | `Warning.Refused`, `WithReasoningRefusals`, `ProbeReasoning`; a caller's saved configuration, as `WithModelForcedChoice`'s callers keep theirs |
| `ThinkingShape`, `ThinkingAdaptive`, `ThinkingBudget` | `llmprovider` | `ReasoningRefusal.Shape`; the Messages and generateContent shape ladders |
| `Warning.Refused` | `llmprovider` | each provider's resend path; the caller that keeps it |
| `WithReasoningRefusals` | `llmprovider` | a caller's next start |
| `Settings.ReasoningRefusals` | `llmprovider` | each provider's D2 step 1 |
| `ProbeReasoning` | `llmprovider` | a caller's configure step; none in this repository (Q6 (a)) |

No other name is exported. The memo, the ladders and the matchers are
internal.

### D10. 0061's Phase 3 (Q2 (b))

* 0061-MADR D3's rules 1 and 2, which read the facts, become D2's step 2
  and D3's effort ladder.
* Its rule 3's per-wire rows (Grok's clamp; Gemini's and the google route's
  `high`) are replaced by D2 to D4, and kept only as D8 says.
* `EffortMax` and `WarningEffortDegraded` stay as 0061-MADR D3 and D10
  define them.
* ~~0061-PLAN's Phase 3 stays staged and uncommitted until this PLAN's
  Phase 2 reworks it in place. The owner commits the two together.~~
  Overtaken: the phase was committed as `dd41faf` (the amendment "0061's
  Phase 3 committed"). This PLAN's Phase 2 reworks that code in place, in
  its own commit.
* 0061-PLAN's Phases 4 to 8 follow this PLAN, and 0061's release carries
  this decision.

### D11. Docs and the standard

* **`docs/guides/api-standards.md` gains R56:** "No per-model name rule
  decides a request parameter. The model's facts or its learned refusals do,
  and a rule a service's refusal cannot replace stays as a wire mapping,
  cited and listed under Degradations. (0062 D2, D8)"
* **Each provider's package doc** states its ladder and what it learns.
* **`docs/architecture.md`:** the order of authority, the memo, and the
  resend path.
* **`CHANGELOG.md`:** the exports and the behaviour change.

### Consequences

* Good, because a new model works without a release. A model the facts do
  not know gets what was asked, and a refusal corrects it within the same
  request.
* Good, because a wrong fact or rule is corrected the same way, not by a
  release.
* Good, because each new value, such as `EffortMax`, needs no table per
  provider.
* Good, because one mechanism, 0034's, serves tool choice and thinking.
* Bad, because the first request to a model that refuses makes extra round
  trips, up to 7 sends (Q5 (a)), and adds their latency.
* Bad, because whether a refused send is billed is UNCONFIRMED (F3).
* Bad, because an accepted-and-ignored value is invisible to learning (F4).
  Where Phase 1 finds one, a rule stays.
* Bad, because a matcher reads a vendor's message, which can change. An
  unrecognised refusal is returned as an error, as today, and the live test
  of PLAN Phase 5 finds the change.
* Bad, because learned state is lost on restart unless the caller keeps
  `Warning.Refused` and passes it back.
* Bad, because it adds nine exported names (D9).
* Neutral, because the G-wire goldens of the thinking scenarios change
  wherever a removed rule decided. Each change is listed in the phase's
  record.

### Confirmation

* `docs/reports/0062-REPORT-thinking-refusals.md` holds each wire's recorded
  refusals and failures of other causes, and D8's outcome for each rule.
* Each matcher's test passes every recorded refusal of its wire and no other
  failure; planted breaks of the matchers are caught.
* For each provider and route, a fake server refuses as the recorded
  refusal does. The test checks that:
  * the request succeeds within the bound;
  * the response carries `WarningReasoningLearned` with its `Refused`;
  * a second request sends the learned value first;
  * `WithReasoningRefusals` makes the first request send it first.
* `LLMPROVIDER_LIVE_0062` confirms each removed rule's case against the live
  service.
* No removed rule's function remains (`grep` in the phase's record).

## Pros and Cons of the Options

### A. Facts, then learned refusals, plus an explicit probe

* Good, because it costs nothing on an accepted request, and corrects
  itself.
* Good, because it reuses 0034's machinery.
* Neutral, because the probe is opt-in, so its cost is the caller's choice.
* Bad, because it depends on readable refusals (F3), which Phase 1 must
  establish.
* Bad, because it cannot see an ignored value (F4).

### B. Probe every model before first use

* Good, because a request never meets a refusal.
* Bad, because each probe that the service accepts reasons and is billed
  (F7). Each model needs one probe per shape and effort it might send.
* Bad, because it adds latency before the first request, or a warm-up step.
* Bad, because it cannot see an ignored value either (F4).

### C. Facts only

* Good, because it is the simplest.
* Bad, because a model without facts gets no thinking parameter, or one that
  fails, and nothing corrects it. 0041-MADR rejected it as its option B.

### D. Keep 0041's order

* Good, because it changes nothing.
* Bad, because a model newer than the SDK or missing from the facts is
  decided by a name rule, and a wrong one fails until a release (the problem
  this record states).

## Owner questions

* **Q1 to Q4.** Answered on 2026-10-24: Q1 (a), facts, then learned
  refusals, plus an explicit probe; Q2 (b), hold 0061-PLAN Phase 3
  (overtaken; the amendment "0061's Phase 3 committed"); Q3 (a),
  process memory plus a pass-back; Q4 (a), every thinking parameter on every
  provider.
* **Q5 (D5). How many sends a request may make.** Answered on 2026-10-24:
  (a).
  * **(a)** Until the ladders run out: at most 7 (recommended). A refused
    send costs one round trip and carries no usage.
  * **(b)** One resend, as 0034 allows. A second refusal is returned, and the
    next request starts from what was learned.
  * **(c)** At most 3 sends.
* **Q6 (D7). Who calls `ProbeReasoning` in this repository.** Answered on
  2026-10-24: (a).
  * **(a)** No one in this repository. Consumers call it from their
    configure step (recommended: `wizard` is outside this decision, and the
    probe is billed).
  * **(b)** `wizard` calls it after a model is chosen, when probes are on,
    and returns the refusals in its result.
  * **(c)** No probe API now; learning on use only.
* **Q7 (D8). OpenCode's chat route without metadata.** Answered on
  2026-10-24: (a).
  * **(a)** Send the effort and learn, as every other wire does
    (recommended).
  * **(b)** Send no effort without metadata, as today (MADR 0009 §6).
  * **(c)** Send the effort without metadata, but never learn on this
    route.
* **Q8 (D8). A provider of Phase 1's case table with no key set.**
  Answered on 2026-10-25: (b).
  * **(a)** Record it as `skipped`; its wire learns nothing and its rules
    stay as wire mappings; Phase 2 goes ahead.
  * **(b)** Block Phase 2 until every provider and route of the case table
    is recorded.
  * **(c)** Build its matcher from the service's published error
    documentation, amending D4.
* **Q9 (D8). Ollama's host and thinking model.** Answered on 2026-10-25:
  (c).
  * **(a)** A local host and an installed thinking model, in Phase 1.
  * **(b)** Skip Ollama; its mappings stay under D8.
  * **(c)** Defer Ollama to the PLAN's Phase 5, keeping its mappings until
    then; Phase 5 may reopen D8 for it.
* **Q10 (D4). How widely the other causes run** (an empty input; a tool
  whose schema is `{"type":"string"}`). Answered on 2026-10-25: (a).
  * **(a)** On one model of every provider and route, as the PLAN's case
    table says.
  * **(b)** On one provider per wire.
  * **(c)** Canned bodies only, no live sends.
* **Q11 (F3). The output limit of the recorder's sends.** Answered on
  2026-10-25: (a).
  * **(a)** The SDK's default: the recorder sets no `MaxTokens`, so each
    provider sends what it sends today, and the 4096 budget stays valid.
  * **(b)** `MaxTokens` 8192 on every case.
  * **(c)** `MaxTokens` 2048, with Claude's budget case lowered below it.
* **Q12 (D8). A rule whose avoided value Phase 1 records as accepted.**
  Answered on 2026-10-25: (a). D8 named two outcomes, a recognisable
  refusal or none; for R1's non-Claude side, R2 and 0061's google-route row
  the value each avoids was accepted.
  * **(a)** Remove the rule: nothing is refused, so nothing is learned, and
    D2 and D3 send what the request asks.
  * **(b)** Keep each as a wire mapping, citing its fixture.
  * **(c)** Decide in Phase 5, comparing reasoning tokens between the shapes.
* **Q13 (D4). `gemini-2.5-flash-lite`'s refusal of `low`,** "The thinking
  budget 256 is invalid", which names a budget the request did not send.
  Answered on 2026-10-25: (a).
  * **(a)** Read it as a refusal of the level sent: Interactions sends no
    budget (D1), so the budget is the level's. R5 is removed.
  * **(b)** Keep R5 for `low` only.
  * **(c)** Keep R5 whole.

## More Information

### Related

* 0034-MADR-degrade-forced-tool-choice-per-model.md: the memo, matcher,
  resend and pass-back this record reuses.
* 0041-MADR-decide-per-model-behaviour-from-metadata.md: the facts and the
  name rules (R1 to R7).
* 0049-MADR's D3: the memo held by a `catalog.Cache`.
* 0061-MADR-consumer-gaps-from-gobble-issue-9.md: D3 and its amendment
  "Phase 3's corrections"; its amendment "D3 reworked by 0062".
* 0013-PLAN-remediate-debugging-pass-findings.md,
  0014-PLAN-gemini-wire-fidelity.md,
  0026-MADR-remediate-v1-2-debugging-pass-findings.md: where R1, R4 and R5
  were measured.

### Code claims

<!-- claims: self=1b46940 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | R1 | `llmprovider/internal/wire/messages/messages.go:536` | `func claudeAdaptiveOnly(model string) bool {` |
| C2 | R1's measurement | `llmprovider/internal/wire/messages/messages.go:533` | `claude-sonnet-5 and claude-opus-4-8, measured 2026-09-27` |
| C3 | R1's version cut | `llmprovider/internal/wire/messages/messages.go:555` | `return major > 4 \|\| (major == 4 && minor >= 7)` |
| C4 | R2 | `llmprovider/internal/wire/messages/messages.go:561` | `func minimaxAdaptiveThinking(model string) bool {` |
| C5 | R1 and R2 apply without facts | `llmprovider/internal/wire/messages/messages.go:615` | `adaptive = minimaxAdaptiveThinking(model) \|\| claudeAdaptiveOnly(model)` |
| C6 | R3's default budget | `llmprovider/internal/wire/messages/messages.go:47` | `const defaultThinkingBudget = 4096` |
| C7 | R3's low budget | `llmprovider/internal/wire/wire.go:49` | `const LowEffortThinkingBudget = 1024` |
| C8 | R4 | `llmprovider/internal/wire/generatecontent/generatecontent.go:413` | `var geminiLegacyRE = regexp.MustCompile(` |
| C9 | R4 applied | `llmprovider/internal/wire/generatecontent/generatecontent.go:429` | `case geminiLegacyRE.MatchString(model):` |
| C10 | R5's measurement | `llmprovider/providers/gemini/interactions.go:105` | `gemini-2.5-flash-lite refuses medium and fails on low (measured` |
| C11 | R5 | `llmprovider/providers/gemini/interactions.go:132` | `if strings.HasPrefix(strings.ToLower(model), "gemini-2.5-flash-lite") && level != llmprovider.EffortHigh {` |
| C12 | Gemini's xhigh | `llmprovider/providers/gemini/interactions.go:117` | `if level == llmprovider.EffortXHigh {` |
| C13 | R6 | `llmprovider/providers/grok/reasoning.go:47` | `func effortMenu(model string) []string {` |
| C14 | R6's ids | `llmprovider/providers/grok/reasoning.go:50` | `case sm == model46, sm == model47, sm == model43:` |
| C15 | Ollama's max | `llmprovider/providers/ollama/ollama.go:87` | `maxEffort = "max"` |
| C16 | Ollama's xhigh | `llmprovider/providers/ollama/ollama.go:297` | `if effort == llmprovider.EffortXHigh {` |
| C17 | OpenCode's chat gate | `llmprovider/providers/opencode/opencode.go:698` | `Metadata that is unavailable, disabled or silent on the model` |
| C18 | Hugging Face ignores | `llmprovider/providers/huggingface/huggingface.go:19` | `Hugging Face documents reasoning_effort as "provider and` |
| C19 | The memo per cache | `llmprovider/internal/wire/messages/learn.go:19` | `catalog.Cache holds one; a nil *Refusals is the process's (0049-MADR D3).` |
| C20 | The memo's bound | `llmprovider/internal/wire/messages/learn.go:14` | `const learnedRefusalsLimit = 256` |
| C21 | The memo's key | `llmprovider/internal/wire/messages/learn.go:43` | `func refusalKey(provider, model string) string {` |
| C22 | The matcher | `llmprovider/internal/wire/messages/learn.go:77` | `func IsForcedChoiceRefusal(err error) bool {` |
| C23 | One resend | `llmprovider/internal/wire/retry.go:13` | `a stream's first event, nor on a resend (0034-MADR D3).` |
| C24 | Send's resend | `llmprovider/internal/wire/retry.go:28` | `resend, err := retry(err)` |
| C25 | The pass-back | `llmprovider/warning.go:16` | `keep, and pass back with WithModelForcedChoice on its next start.` |
| C26 | Its option | `llmprovider/forcing.go:55` | `func WithModelForcedChoice(model string, s Support) Option {` |
| C27 | The stream's resend | `llmprovider/internal/wire/sse.go:135` | `func StreamEventsRetry(ctx context.Context, provider string, src llmprovider.TokenSource, build func(llmprovider.Token) Call,` |
| C28 | Claude sends through Send | `llmprovider/providers/claude/claude.go:223` | `resp, err := wire.Send(ctx, string(llmprovider.ProviderClaude), p.src,` |
| C28b | OpenCode sends through Send | `llmprovider/providers/opencode/opencode.go:295` | `resp, err := wire.Send(ctx, "opencode", p.src,` |
| C29 | Kilo, for one, posts | `llmprovider/providers/kilo/kilo.go:276` | `return wire.Post(ctx, p.call(ctx, req, token, false), token, chatcompletions.DecodeFor(string(llmprovider.ProviderKilo)))` |
| C30 | APIError's code | `llmprovider/api_error.go:99` | `// Code is the service's error type or code, e.g. "FreeUsageLimitError".` |
| C31 | APIError's message | `llmprovider/api_error.go:101` | `// Message is the service's message, redacted and bounded to 512 bytes.` |
| C32 | Reasoning tokens | `llmprovider/contract.go:188` | `ReasoningTokens int` |
| C33 | Probes are billed | `llmprovider/providers/claude/claude.go:421` | `// Probes are billed; they are on by default, and callers can turn them off (0016-MADR A5).` |
| C34 | Facts from the caches | `llmprovider/catalog/lookup.go:146` | `func CachedFacts(ctx context.Context, provider llmprovider.ProviderID, model string, opts ...llmprovider.Option) (ModelInfo,` |
| C35 | Claude's 400, live | `docs/decisions/0013-PLAN-remediate-debugging-pass-findings.md:342` | `` `llm: invalid request: claude HTTP 400` at both efforts `` |
| C36 | Gemini 2.x's 400 | `docs/decisions/0014-PLAN-gemini-wire-fidelity.md:1898` | `a 1024 budget on 2.x (thinkingLevel is HTTP 400` |
| C37 | A caller's layer | `llmprovider/catalog/lookup.go:32` | `func WithModelLayer(l Layer) llmprovider.Option {` |
| C38 | The closed set | `llmprovider/contract.go:81` | `case EffortLow, EffortMedium, EffortHigh, EffortXHigh:` |
| C39 | The last budget Claude retires | `llmprovider/internal/retired/retired.go:29` | `"claude-sonnet-4-5-20250929": "retires 2026-11-30",` |
| C40 | Gemini 2.5 is closed | `llmprovider/internal/retired/retired.go:23` | `"gemini-2.5-flash":      "open only to past users",` |
| C41 | The forced-choice accessor | `llmprovider/forcing.go:70` | `func (st *Settings) ModelForcedChoices() map[string]Support {` |

## Amendment 2026-10-24: 0061's Phase 3 committed

While this record was proposed, the owner committed 0061-PLAN's Phase 3 as
`dd41faf feat(llmprovider): add EffortMax reasoning effort`: its 44 staged
paths, 0061's records at that time included, and nothing else. Asked to
choose between keeping the commit and amending the records, reverting it and
holding the phase again, or resetting it, the owner kept it (2026-10-24).

* Q2's answer (b) stands as given; the hold it asked for no longer applies.
* D10 holds otherwise: 0061 D3's rules 1 and 2 become D3's effort ladder, and
  its rule-3 rows are replaced, in this PLAN's Phase 2, which reworks the
  committed code in place in its own commit.
* Until that phase, `main` sends `max` as 0061 D3's rule-3 rows say: Grok
  clamped to its menu, and `high` on Gemini and OpenCode's google route.
  None of it is pushed or released; the release is `v1.23.0`, after this
  PLAN.
* The Code claims stay pinned to `1b46940`, where they were read, and pass
  there, 41 of 41. At `dd41faf` (`tools/claims.py --rev self=dd41faf`), 31
  of 41 pass:
  * C13 to C17, C28, C29, C32 and C33 hold the same text, moved by Phase 3's
    lines;
  * C38 changed as F9 says: `valid()`'s closed set is
    `llmprovider/contract.go:87` there,
    `case EffortLow, EffortMedium, EffortHigh, EffortXHigh, EffortMax:`.

## Amendment 2026-10-25: Phase 1's evidence

The PLAN's Phase 1 recorded 51 cases on 2026-10-25:
[0062-REPORT-thinking-refusals.md](../reports/0062-REPORT-thinking-refusals.md)
(its findings E1 to E9) and the fixtures in
`llmprovider/internal/wire/testdata/refusals-0062/`. No case is `skipped`
or 401 (Q8 (b)). What follows restates F3, F4 and F6 as measured, gives D4's
matcher per wire, and D8's outcome per rule, under the owner's Q12 (a) and
Q13 (a).

### F3, F4 and F6, as measured

* **F3. What a refusal says** (report E1 to E3, E6, E8):
  * The parameter is named in the message by Claude (both shapes and the
    effort), OpenAI's `unsupported_parameter`, Gemini's Interactions and
    Hugging Face. The Responses wire's `invalid_value` (OpenAI and OpenCode's
    responses route) and `unsupported_value` (OpenCode's responses route, for
    `gpt-5.5`) name the value sent; Kilo's "Invalid option" names only the accepted values;
    Grok's "Invalid reasoning effort." names neither; `gemini-2.5-flash-lite`
    names a budget for a `low` level.
  * The accepted values are listed by Claude's effort refusal, the
    Responses wire's `invalid_value` and `unsupported_value`, Gemini's
    Interactions, Hugging Face and Kilo. Not
    by Claude's shape refusals (which name the other shape), Grok,
    flash-lite's `low`, or `unsupported_parameter`.
  * **No refusal counts usage:** none of the 32 replies that are not 2xx
    carries a usage field. Whether a refused request is billed stays
    **UNCONFIRMED**: no reply says.
* **F4. Accepted and ignored.** Together accepts a bogus `reasoning_effort`
  and reasons (E5): confirmed for Together. Every other service sent a bogus
  effort refused it. OpenCode Zen's google route accepts `thinkingLevel`
  `"max"`, which Gemini's own Interactions refuses; whether the gateway uses,
  maps or drops it cannot be seen, as F4 says of any accepted value.
* **F6. No matcher needs `param`.** Where only the body's `param` names the
  parameter (E8), the message holds the value the provider sent or a list
  made only of effort words, and a matcher has the request it sent. D4's
  clause for a missing field is not met: `*APIError` and D9 are unchanged.

### D4: the matcher of each wire

Each row is a refusal that Phase 1 recorded; every recorded 400 of another
cause on the wire must not match (report E9). The effort words are `none`,
`minimal`, `low`, `medium`, `high`, `xhigh` and `max`; a learned list keeps
only those that are `Effort` values.

| Wire | Status, code | Message | Refused | Accepted values |
| :--- | :--- | :--- | :--- | :--- |
| Messages | 400 `invalid_request_error` | contains `"thinking.type.enabled" is not supported` | the budget shape | none |
| Messages | 400 `invalid_request_error` | contains `adaptive thinking is not supported` | the adaptive shape | none |
| Messages | 400 `invalid_request_error` | starts `output_config.effort: Input should be`, then the list | the effort sent | the quoted words |
| Responses | 400 `invalid_value` | `Invalid value: '<sent>'. Supported values are: …`, every listed word an effort word | the effort sent | the quoted words |
| Responses | 400 `unsupported_value` | `Unsupported value: '<sent>' is not supported with the '<model>' model. Supported values are: …` | the effort sent | the quoted words |
| Responses | 400 `unsupported_parameter` | contains `'reasoning.effort' is not supported` | the effort field: D3's rung 4, none | none |
| Responses (Grok) | 400 `invalid-argument` | is `Invalid reasoning effort.` | the effort sent | none |
| Interactions | 400 `invalid_request` | `The value '<sent>' is not supported for 'generation_config.thinking_level'. Supported values: …` | the level sent | the quoted words |
| Interactions | 400 `invalid_request` | `'<sent>' is not a supported thinking level for this model. Allowed values are: …` | the level sent | the listed words |
| Interactions | 400 `invalid_request` | `The thinking budget <n> is invalid. …` (Q13 (a)) | the level sent | none |
| Chat Completions (Hugging Face) | 400 `400` | `reasoning_effort (<sent>): Input should be …` | the effort sent | the quoted words |
| Chat Completions (Kilo) | 400 `invalid_request_error` | starts `Invalid option: expected one of`, then the list, every listed word an effort word | the effort sent | the quoted words |

* **The overlaps a matcher must keep apart:** Claude's tool-schema refusal
  shares the code and "Input should be" (anchored on `output_config.effort:`);
  Grok's shares `invalid-argument` (anchored on the whole message); Kilo's
  empty input shares the code (anchored on "Invalid option"). A 404 is never
  read: Gemini's Interactions answers an empty input 404 "Model … not found".
* **No refusal was recorded** on generateContent (OpenCode's google route
  accepted a level of `max` and a budget), Together (it accepts any value),
  OpenCode's chat route or its messages route: no matcher is built for them,
  and they learn nothing, having nothing recorded to learn.

### D8: the outcome of each rule

| Rule | Outcome | Evidence |
| :--- | :--- | :--- |
| R1 `claudeAdaptiveOnly` | removed | `claude-sonnet-5` refuses the budget shape, and `claude-sonnet-4-5-20250929` the adaptive one, each recognisably (E1); `qwen3.8-flash` accepts adaptive (E4; Q12 (a)) |
| R2 `minimaxAdaptiveThinking` | removed (Q12 (a)) | `minimax-m3` accepts adaptive with an effort (E4) |
| R4 `geminiLegacyRE` | removed as dead | it matches only `gemini-2.5-flash` and `gemini-2.5-flash-lite`, both open only to past users (C40), and the first is "Model is unavailable" on OpenCode's google route (E7) |
| R5 the flash-lite exception | removed | `medium` and `low` are refused recognisably (E3; Q13 (a)) |
| Gemini's `xhigh` to `high` | removed | refused with the accepted levels (E3) |
| R6 `effortMenu` | removed | Grok refuses an effort it does not take, recognisably (E3) |
| Ollama's `xhigh` to `"max"` | decided in Phase 5 (Q9 (c)) | not recorded |
| 0061-MADR D3's Grok row | removed | as R6 |
| 0061-MADR D3's Gemini row | removed | `max` refused with the accepted levels (E3) |
| 0061-MADR D3's google-route row | removed (Q12 (a)) | `max` accepted (E4) |
| OpenCode's chat-route gate | removed (Q7 (a)) | `max` accepted on `glm-5.3` without metadata (E4) |
| R3, the default budgets | stays (F1) | not a model rule |

* **The first request that learns,** on the evidence: `claude-sonnet-4-5`'s
  adaptive is refused, then its budget sent (2 sends); `max` on Gemini, on
  OpenCode's responses route for `gpt-5.5`, and on Grok steps to the first
  accepted value (2 sends where the list is given; on Grok, one per rung);
  flash-lite's `low` is refused, `medium` refused with the list, `high` sent
  (3 sends). Each is within Q5's 7, and the next request is right the first
  time.

## Amendment 2026-10-25: Phase 2's Grok cases

R6 sends no effort to a Grok model outside its menu, and the amendment
"Phase 1's evidence" removed R6 on Grok's refusal of `max` on models that
take efforts; no case had measured a model outside the menu. The PLAN's
Phase 2 stopped there, and the owner chose to record first (its record of
2026-10-25). Two cases were added and sent:
[0062-REPORT-thinking-refusals.md](../reports/0062-REPORT-thinking-refusals.md)'s
finding E10, and the fixtures `grok-build-0.1-high.json` and
`grok-4.20-0309-non-reasoning-high.json`.

* **D4 gains a row.** Responses (Grok), 400 `invalid-argument`, a message
  that starts `Model` and ends `does not support parameter
  reasoningEffort.`: the model takes no effort, D3's last rung, none.
* **D8: R6 is removed whole.** A model outside the menu refuses the effort
  field recognisably: its first request is sent with the effort, refused,
  learned, and sent again without it (2 sends); its next is right the first
  time. The finding that justified the branch, "other models reason
  automatically and may reject the parameter", holds, and is now learned
  rather than listed.

## Amendment 2026-10-25: R2's row for internal/cacheref

D5 holds the memo of refused thinking settings in each `catalog.Cache`,
beside 0034-MADR's memo of forced-choice refusals, and the PLAN's Phase 2
puts it in `internal/cacheref`. The memo's type, `wire.Learned`, is in
`internal/wire`, which `docs/guides/api-standards.md` R2's row for
`internal/cacheref` did not allow it to import; it already reached
`internal/wire` through `internal/wire/messages`. The owner chose, on
2026-10-25, to amend the row: `internal/cacheref` may import
`internal/wire`, by this record's D5. Offered and not chosen: a new leaf
package for the memo; holding it untyped, as `any`.
