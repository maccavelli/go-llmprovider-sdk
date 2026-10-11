---
status: accepted
date: 2026-10-11
decision-makers: repository owner
consulted: 0033-MADR-public-model-metadata-and-current-catalogs.md (D6, Q6), 0032-MADR-tool-strict-and-result-error-flags.md (D2), 0020-MADR-remediate-v1-debugging-pass-findings.md (F43)
informed: consumers of go-llmprovider-sdk v1
---

<!-- markdownlint-disable MD013 -->

# Degrade a Forced Tool Choice Per Model, Where Anthropic Refuses It

## Context and Problem Statement

[0033-MADR-public-model-metadata-and-current-catalogs.md](0033-MADR-public-model-metadata-and-current-catalogs.md)
D6 asks for this record:

* whether a model accepts a forced tool choice becomes a per-model fact,
  "never a name pattern" (its Q6);
* where the choice is refused, the provider sends `auto` with strict tool
  use, as a `BestEffort` degradation.

**What Anthropic says** (official pages, read 2026-10-11):

* **Define tools, "Forcing tool use".** For "Claude Opus 5.5, Claude Sonnet
  5.5, Claude Fable 5.1, and Claude Mythos 5.1": "`any` and `tool` return a
  400 error". It advises: "`auto` with strict tool use to guarantee
  schema-valid tool inputs, or structured outputs when you need a response
  in a fixed JSON shape. Prompting still influences which tool `auto`
  picks."
  * Haiku 5.5, Opus 5, Sonnet 5, Fable 5, Opus 4.8 and Sonnet 4.6 accept a
    forced choice.
  * Its migration guide says: "Claude Haiku 5.5 accepts a forced
    `tool_choice`".
* **API errors, "Forced tool use not supported".** The 400 is
  `invalid_request_error`: `tool_choice: type "tool" and "any" are not
  supported for this model.`
  * The check also applies to the Message Batches and token counting
    endpoints.
  * The page advises callers to "Catch the SDK's typed classes rather than
    string-matching error messages".
* **Models API.** Its `capabilities` object has no key for forced tool
  choice, and "Keys are always present for all known capabilities". The
  fact cannot be discovered from metadata.
  0033-MADR's measured note found the same.
* **Strict tool use.**
  * `"strict": true` is a top-level property of a tool definition, out of
    beta since the 2026-01-29 release note.
  * It guarantees that a call's `input` follows the schema. It does not
    guarantee a call is made: "The model can then answer without calling the
    tool, so say in the prompt when to use it" (Sonnet 5.5 migration guide).
  * It requires `additionalProperties: false` on every object, among other
    schema limits. An unsupported schema answers 400.
* **Making `auto` call the tool.** The Fable 5.1 migration guide advises
  appending a mid-conversation system message after the latest `user` turn.
  * It names the tool, says the call is required for this turn, and asks
    the response to open with it.
  * Its shape is `{"role": "system", "content": "…"}`. "Mid-conversation
    system messages need no beta header."
  * They are supported on Fable 5.1, Mythos 5.1, Opus 5.5, Sonnet 5.5,
    Haiku 5.5, Fable 5, Opus 5 and Opus 4.8, but not on Sonnet 5.
  * One must immediately follow a `user` turn, and be the last entry or be
    followed by an `assistant` turn.

**What was measured.** `0033-PLAN-refresh-the-curated-catalogs.md` Phase 1,
on 2026-10-11, sent `GenerateToolCall`:

* `claude-sonnet-5-5` and `claude-opus-5-5` answered 400 with that message;
* `claude-haiku-5-5`, `claude-sonnet-5`, `claude-opus-4-8` and
  `claude-sonnet-4-6` accepted it;
* `claude-fable-5-1` was not measured. `claude-mythos-5-1` is not in this
  account's listing.

**What the SDK does today** (`main` at `1d42479`):

* **`GenerateToolCall`** always forces its one tool (`convenience.go:30-33`).
  A reply without the call is `ErrIncomplete` (`:42`).
  * Its only non-test caller is the wirecase test package.
  * gobble-cli never forces a tool, and never sets `ToolChoice`.
* **The Claude provider** maps the choice in `toolChoice`
  (`providers/claude/claude.go:242-259`). A forced choice is sent as
  `tool` or `any`, unless thinking is on, when it is sent as `auto`
  (0020-MADR F43). `ForcedToolChoice` is `BestEffort` (`:113`).
* **OpenCode's `messages` route.** `messagesToolChoice`
  (`providers/opencode/opencode.go:461-479`) is a verbatim copy of it.
  * Zen routes `claude-fable-5-1` and `claude-opus-5-5` to that route
    (`routes_snapshot.json`), and an unlisted `claude-` id by heuristic
    (`route.go:130-133`). Zen fails on them the same way.
  * Its package doc already records qwen models that refuse a forced choice
    on Go's `messages` route (measured 2026-10-02). Those requests are
    still sent as asked.
* **Strict.** `Tool.Strict` is never sent on Messages (`wire/tools.go:81-86`).
  * The Claude package doc says "the Messages API has no such field
    (0032-MADR D2)". Anthropic's strict tool use shows that this is no
    longer true.
  * 0032-MADR D2 anticipates the change: "If a measurement shows Anthropic
    accepting a documented `strict`, adding it is an amendment here."
* **No resend.** No code resends a request after a 400, and no cache holds
  a fact learned from an error. Only `wire.Reauth` resends, once, on a
  refused credential (`internal/wire/reauth.go:27-53`).
* **The catalog.** `staticClaude` leaves the three models out until this
  record (`catalog/models_catalog.go:64-74`), and `waiting0033` enforces
  it (`catalog/refresh_0033_test.go:40-44`). The list holds four of
  `MaxListed`'s six.
* **The live tests.** Four files name `claude-haiku-4-5`, whose retirement
  floor is 2026-10-15. `0033-PLAN-refresh-the-curated-catalogs.md` records
  them as found.

## Decision Drivers

* `GenerateToolCall` and `ToolChoiceRequired` must not fail with a 400 on a
  current Claude model.
* The fact must be per model and exact (0033-MADR Q6). It must also hold
  for the next model that refuses, which no metadata reports.
* The degraded request should make the call as likely as Anthropic's own
  guidance does, without rewriting the caller's input or schema.
* R45: the request on the wire changes only as a record says. Every other
  model sends exactly what it sends today.
* R12: a degradation is named in the package doc. R24/R25: errors keep
  their kinds.
* One implementation for the two providers that speak Messages.
* No new dependency, and no exported API unless needed.

## Considered Options

* **An exact table of refusing models, learned additions at runtime, and a
  degraded request of `auto` with a turn instruction, shared by both
  Messages providers**
* An exact table only, updated by release
* Learning only: send forced, and on the refusal resend degraded
* Structured outputs in place of a tool call, for `GenerateToolCall`
* Leave it: document the 400

## Decision Outcome

Chosen option: "An exact table of refusing models, learned additions at
runtime, and a degraded request of `auto` with a turn instruction, shared by
both Messages providers", because:

* the table sends no wasted request for the four models Anthropic names;
* learning covers the next one without a release;
* `auto` with an instruction is Anthropic's own advice;
* one shared function removes OpenCode's duplicate.

### D1. The fact: an exact table

`internal/wire/messages` holds the models that refuse a forced choice: exact
ids, compared case-insensitively.

* `claude-fable-5-1`, `claude-mythos-5-1`, `claude-opus-5-5` and
  `claude-sonnet-5-5`, as Anthropic's define-tools and errors pages name
  them.
* Phase 1 measures `claude-fable-5-1`. `claude-mythos-5-1` is not offered
  to this account; Anthropic names it, and it is the same model as Fable 5.1
  for verified organisations.
* No version pattern: a new model is added by a record, or learned (D3).

### D2. The degraded request

When a request forces a choice (`ForceTool`, or `ToolChoiceRequired`) and
the model refuses forcing:

* **`tool_choice`** is `{"type": "auto"}`.
* **A mid-conversation system message** is appended after the last `user`
  turn, in Anthropic's documented form, naming the tool, or "one of the
  tools" for `ToolChoiceRequired`:

  > Tool-use requirement for the current turn: the application requires a
  > call to the get_weather tool in your response to the user's latest
  > message. Begin your response with the get_weather tool call. Do not
  > reply with text only.

  If the last entry is not a `user` turn, the system message is not
  appended. The text is added to the top-level `system` instead, which every
  model accepts.
* **`strict`** follows the caller's `Tool.Strict` (D4). The SDK does not set
  it itself: a schema that strict mode cannot compile answers 400 (Q2).
* **The rest:**
  * it is a `BestEffort` degradation, named in each package doc (R12);
  * a reply without the call is `ErrIncomplete`, as today;
  * `ToolChoiceNone` and `auto` are unchanged;
  * the thinking rule of 0020-MADR F43 stays.

### D3. Learning a refusal (Q1)

When a forced request to a model not in the table is refused, the request
is sent once more, degraded, and the model is remembered.

* **What counts as the refusal:** HTTP 400, `invalid_request_error`, and a
  message naming `tool_choice`.
* **The memo:**
  * in-process, keyed by provider and model;
  * bounded in size;
  * never written to disk;
  * cleared only by a restart.
* **Bounds:**
  * one resend per request, as `wire.Reauth` resends once;
  * a 400 that is not this refusal is returned as it is.
* The match is narrower than a typed class, because the API gives no other
  signal. Anthropic's advice against string-matching addresses callers
  branching on errors; this match only chooses whether to resend.

### D4. Strict on the Messages wire (amends 0032-MADR D2; Q3)

`Tool.Strict`, when set, is sent as the tool's top-level `strict`:

* on the Claude provider;
* on OpenCode's `messages` route, if Phase 1 shows Go's models accepting
  it.

Nil still sends nothing, so a caller that never sets it sends the same
bytes. 0032-MADR gains an amendment.

### D5. The catalog and the live tests

* **The menu (Q4).** `staticClaude` becomes, fast and cheap first:
  * `claude-haiku-5-5`;
  * `claude-sonnet-5-5`;
  * `claude-sonnet-5`;
  * `claude-opus-5-5`;
  * `claude-opus-4-8`.

  `claude-sonnet-4-6` leaves the menu. `claude-fable-5-1` stays out of it,
  but is usable by id, degraded the same way.
  * `waiting0033` loses the two added; `claude-fable-5-1` stays in it.
  * Each new id has a label.
  * Phase 1 must show each one answering.
* **The live tests.** Three move from `claude-haiku-4-5` to current models:
  * `TestLive_ClaudeThinkingShapes`'s budget case moves to
    `claude-sonnet-4-6`, which still takes budget thinking;
  * `TestLive_ToolRoundTripClaude`, `TestLive_SystemMessageClaude` and
    `TestLive_ClaudeStreams` move to `claude-haiku-5-5`.
* **Two stay:**
  * `TestLive_OpencodeKeyHeaderPerRoute` sends a bogus key, so the model is
    never served;
  * `TestLive_0032ResultError` is 0032's measurement, kept as it ran.
* **A new live test.** `GenerateToolCall` on each refusing model in the menu
  returns the call, or `ErrIncomplete`, and never a 400.

### D6. Release

`v1.7.2`. No exported API changes. The wire changes for the refusing
models, and for a set `Tool.Strict` on Messages.

### Owner questions

* **Q1 (D3). Where the fact comes from.** Offered:
  * the exact table, and learning at runtime (recommended): known models
    cost nothing, and a future model degrades after one refused request,
    with no release;
  * the exact table only: no resend logic, but each new model fails until a
    release;
  * learning only: no table, but every process pays one refused request per
    model.
* **Q2 (D2). The degraded request.** Offered:
  * `auto`, the turn instruction, and `strict` only where the caller set it
    (recommended): Anthropic's advice, without rewriting the caller's
    schema;
  * the same, but always `strict: true` on the tool: input guaranteed
    schema-valid, but a schema strict mode cannot compile, such as one
    without `additionalProperties: false`, answers 400;
  * `auto` alone: simplest, but the model may answer in text;
  * structured outputs: a guaranteed JSON object, turned into a call by the
    SDK. Reliable, but `GenerateToolCall` would no longer send a tool, and
    it is incompatible with citations and prefill.
* **Q3 (D4). Strict on Messages.** Offered:
  * send a set `Tool.Strict` on Claude, and on OpenCode's `messages` route
    if measured (recommended);
  * Claude only;
  * keep it unsent, as 0032 decided.
* **Q4 (D5). The menu.** Offered:
  * add `claude-sonnet-5-5` and `claude-opus-5-5`, drop
    `claude-sonnet-4-6`, and leave `claude-fable-5-1` out of the menu, at
    $10/$50 per million tokens (recommended), keeping six;
  * add all three, and drop `claude-sonnet-4-6` and `claude-opus-4-8`;
  * add none.

**Answered by the owner on 2026-10-11:** the recommended option of each.

* **Q1:** the exact table, and learning at runtime.
* **Q2:** `auto`, the turn instruction, and `strict` only where the caller
  set it.
* **Q3:** a set `Tool.Strict` on Claude, and on OpenCode's `messages` route
  if measured.
* **Q4:** add `claude-sonnet-5-5` and `claude-opus-5-5`, drop
  `claude-sonnet-4-6`, and leave `claude-fable-5-1` out of the menu.

### Measured (2026-10-11)

0034-PLAN Phase 1 sent 23 requests:

* **The table.** `claude-fable-5-1` refuses a forced choice with the same
  400 as `claude-sonnet-5-5` (M1, M2). D1's table stands.
* **The degraded request.** `auto` with the turn instruction called the
  tool in 9 of 9 tries on the three refusing models, with valid input (M4).
  The top-level fallback (M5), and `auto` alone (M3, 3 of 3), also called
  it.
* **Strict.** Messages accepts a tool's `strict: true`: forced on
  `claude-haiku-5-5` and `claude-sonnet-5`, degraded on
  `claude-sonnet-5-5` (M6), and on OpenCode Go's `messages` route (M8). A
  schema without `additionalProperties: false` answers 400 (M7). D4 is
  built on both providers, with `strict` only where the caller set it.
* **D3's rule.** It matches Anthropic's refusal (M2). OpenCode Go's qwen
  refusal answers 400 with the body `{"model":"qwen3.8-flash"}`, which it
  does not match (M9), so that refusal is not learned.

### Consequences

* Good, because `GenerateToolCall` works on every current Claude model, and
  through Zen's `messages` route.
* Good, because a model Anthropic adds to the list later degrades after one
  refused request, not after a release.
* Good, because Strict means on Messages what it means on Responses.
* Neutral, because a degraded request may still answer in text, which is
  `ErrIncomplete`, as an unforced request can.
* Bad, because a learned refusal costs one refused request per model and
  process.
* Bad, because the refusal is recognised by its message, which Anthropic
  could reword. The table still covers the known models, and a reworded
  refusal is returned as a 400, as today.

### Confirmation

* Each new test is seen failing first.
* Phase 1 measures, before anything is built:
  * the refusal on `claude-fable-5-1`;
  * the degraded request's call rate;
  * `strict` on Messages;
  * the refusal's exact error.
* The goldens of every model not in the table are unchanged. A new
  wirecase case pins a refusing model.
* The live suites pass on the release commit, with the new live test.

## Pros and Cons of the Options

### An exact table, learned additions, and `auto` with a turn instruction

* Good, because no request is wasted for a known model.
* Good, because a new model is covered without a release.
* Good, because it follows Anthropic's documented advice.
* Bad, because it adds a resend path, and a memo.

### An exact table only

* Good, because it is the smallest change.
* Bad, because each new refusing model fails until a release adds it.

### Learning only

* Good, because there is no table to keep.
* Bad, because every process pays one refused request per model, including
  the four already known.

### Structured outputs

* Good, because the JSON is guaranteed, apart from a refusal or a cut.
* Bad, because the request carries no tool, so the reply has no
  `tool_use`, and the SDK must invent the call.
* Bad, because it changes what `GenerateToolCall` sends on the wire for
  these models far more than a tool choice does.

### Leave it

* Good, because nothing changes.
* Bad, because the newest Claude models stay out of the menu, and
  `GenerateToolCall` fails on them.

## More Information

* **Anthropic's pages read on 2026-10-11:**
  * `agents-and-tools/tool-use/define-tools#forcing-tool-use`;
  * `api/errors#forced-tool-use-not-supported`;
  * `agents-and-tools/tool-use/strict-tool-use`;
  * `build-with-claude/structured-outputs`;
  * `build-with-claude/mid-conversation-system-messages`;
  * `api/models/list`;
  * the migration guides of Fable 5.1, Opus 5.5, Sonnet 5.5 and Haiku 5.5;
  * the release notes of 2026-01-29, 2026-09-01, 2026-09-22, 2026-09-28
    and 2026-10-07.
* **Related:**
  * [0033-MADR-public-model-metadata-and-current-catalogs.md](0033-MADR-public-model-metadata-and-current-catalogs.md)
    (D6, which this record answers);
  * [0032-MADR-tool-strict-and-result-error-flags.md](0032-MADR-tool-strict-and-result-error-flags.md)
    (D2, which D4 amends);
  * [0020-MADR-remediate-v1-debugging-pass-findings.md](0020-MADR-remediate-v1-debugging-pass-findings.md)
    (F43, the thinking rule).
* **The plan** is
  [0034-PLAN-degrade-forced-tool-choice-per-model.md](0034-PLAN-degrade-forced-tool-choice-per-model.md).
