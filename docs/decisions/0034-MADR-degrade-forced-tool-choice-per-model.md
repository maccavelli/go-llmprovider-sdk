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

*(Amended 2026-10-11: the table is one layer of D9, matched on a normalised id; see "Amendment (2026-10-11): prevention first".)*

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

*(Amended 2026-10-11: degrading is now the caller's opt-in, a forced named tool uses structured outputs, and every degradation is reported; see "Amendment (2026-10-11): prevention first".)*

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

*(Amended 2026-10-11: narrowed to models no layer knows, and bounded by the caller's policy; see "Amendment (2026-10-11): prevention first".)*

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
  * one resend per request, as `wire.Reauth` resends once. *(Amended
    2026-10-11: the resend sits in the shared send and stream openers,
    beside `wire.Reauth`, for `Generate` and `Stream` alike; 0034-PLAN
    Phase 4f.)*
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

*(Amended 2026-10-11: `v1.8.0`, since the amendment adds exported API.)*

`v1.7.2`. No exported API changes. The wire changes for the refusing
models, and for a set `Tool.Strict` on Messages.

### Amendment (2026-10-11): prevention first

After Phase 4 was committed, the owner asked for the handling to be
re-assessed. The aims were:

* make preventing degraded requests a priority;
* follow the protocols' standards, idiomatic Go, and how other agent
  projects handle it.

**What the research found** (2026-10-11; recorded in 0034-PLAN, "Amendment
(2026-10-11): prevention first"):

* **Every library that handles this refusal prevents it before sending,
  from a per-model flag:**
  * Vercel's AI SDK has `rejectsForcedToolUse`, and degrades to `auto` with
    a structured warning on the result. Its forced-JSON path uses native
    structured outputs instead.
  * LiteLLM has `supports_forced_tool_use`. It refuses before sending with
    a typed error unless the caller opts in to degrading (`drop_params`).
  * Anthropic's own Swift bridge proposed refusing: "A required call is a
    contract, so it's refused rather than silently downgraded."
* **No project learns from the 400 and resends.**
* **Some never force a tool at all:**
  * Goose, pi and Codex always send `auto`;
  * pi's neutral API cannot express a forced choice.
* **Others have no handling,** and fail on these models: OpenCode, Kilo
  Code and grok-build.
* **No public metadata carries the fact.** Neither models.dev nor
  Anthropic's Models API has a field for it, and OpenRouter's
  `supported_parameters` cannot tell `auto` from forced.
* **The standards:**
  * ACP reports a degradation as a non-fatal session `notice`, separate
    from errors;
  * gRPC's retry design and the SSE spec allow a resend only before
    anything is committed to the caller;
  * RFC 9110 allows a 400 to be resent only once the request is changed;
  * Go favours typed errors and switchable retries;
  * this SDK's R11 refuses an unsupported request before the network, with
    `ErrUnsupported`.

**The owner's answers**, the recommended option of each:

* **Q5. The default for a model that refuses forcing:** strict.
  * A forced choice that would be degraded fails before the network, with
    an error matching `ErrUnsupported` (R11).
  * The caller opts in to degrading.
* **Q6. `GenerateToolCall`, and any forced named tool, on such a model:**
  native structured outputs, so the guarantee is kept and nothing is
  degraded.
* **Q7. Visibility:** a typed `Response.Warnings`.
* **Q8. The fact:** layered, matched on a normalised id, and seeded by the
  caller.

**What changes:**

* **D7. Warnings.** `llmprovider` gains, as additions only:

  ```go
  type WarningKind string
  const (
      WarningForcedChoiceDegraded WarningKind = "forced_tool_choice_degraded"
      WarningForcedChoiceLearned  WarningKind = "forced_tool_choice_learned"
  )
  type Warning struct {
      Kind   WarningKind
      Model  string // the model the warning is about
      Detail string // a sentence for a person
  }
  // Response gains: Warnings []Warning
  ```

  * A stream carries them on its `EventDone` response.
  * Every degradation of a forced choice adds one, including 0020-MADR
    F43's `auto` with thinking, whose request does not change.
  * A refusal learned at runtime adds one naming the model, which a
    caller can persist and pass back as an override (D9).
  * The SDK knows nothing of ACP. gobble-cli maps a warning to a session
    `notice` when the client advertises notices, or to an agent message
    otherwise, under its own record.
* **D8. A forced named tool is sent as structured outputs.** On a model
  that refuses forcing, `ForceTool(name)` (and so `GenerateToolCall`) is
  sent as follows:
  * `output_config.format` carries that tool's schema (`json_schema`);
  * the tools stay offered, with `tool_choice: none`, so the conversation's
    tool history stays valid;
  * the reply's JSON is returned as a `FunctionCallItem` for that tool,
    with a synthesised call id, finishing `tool_calls`.

  It is not a degradation: the arguments are schema-valid, except on a
  refusal or a cut. If the schema cannot be compiled (it lacks
  `additionalProperties: false`, as 0034-PLAN M7 found), the API's 400 is
  returned. Phase 4b measures the shape before it is built.
* **D9. The fact, by layer, highest first:**
  1. **the caller's override,** `WithModelForcedChoice(model, Supported)`
     or `(model, Unsupported)`, in either direction;
  2. **D1's dated table;**
  3. **a refusal learned this process** (D3);
  4. **unknown, which is treated as accepting,** as every library does.

  * **Matching:** ids are normalised before matching:
    * lower case;
    * a gateway prefix (`anthropic/`), and Bedrock's `anthropic.` prefix
      and `-v1:0` suffix, removed;
    * a trailing `-YYYYMMDD` or `@YYYYMMDD` snapshot date removed.

    Matching is then exact. That is still a per-model fact, not a version
    pattern (0033-MADR Q6).
  * **0033's `ModelInfo`** gains the fact as a field when that plan lands,
    and this table becomes one of its layers.
* **D10. The caller's policy.**
  * `WithForcedChoiceFallback(ForcedChoiceRefuse)` is the default.
  * `WithForcedChoiceFallback(ForcedChoiceDegrade)` opts in to D2's `auto`
    and turn instruction, with a `WarningForcedChoiceDegraded`.
  * The policy decides only `ToolChoiceRequired`: a named tool takes D8
    under either policy.
* **D3, narrowed.** A forced request to a model no layer knows, refused in
  Anthropic's form, is learned, then handled by policy:
  * **a named tool** is resent once as D8;
  * **`ToolChoiceRequired` under the refuse policy** returns an error
    matching `ErrUnsupported`, wrapping the `*APIError`;
  * **`ToolChoiceRequired` under the degrade policy** is resent once
    degraded.

  Each adds a `WarningForcedChoiceLearned`. The resend sits in the shared
  openers, before anything reaches the caller (0034-PLAN Phase 4f).
* **D6. Release:** `v1.8.0`, for the exported additions. 0033's metadata
  plan moves to `v1.9.0`.

The options declined:

* degrading by default, silently or with a warning;
* following the policy for a named tool too;
* logging only, or the package docs only;
* keeping the table and memo as they were, or relying only on the caller.

### Measured (2026-10-11), structured outputs

0034-PLAN Phase 4b sent 7 requests:

* **Schema-valid JSON.** `output_config.format` with a strict-ready schema
  returned schema-valid JSON as text, beside a thinking block and with no
  tool call, on `claude-sonnet-5-5`, `claude-opus-5-5` and
  `claude-fable-5-1`. It did so with the tools offered under
  `tool_choice: none` (S1), and beside an effort (S5).
* **An incompatible schema.** A schema without `additionalProperties: false`
  answers 400 (S2).
* **Replaying the call.** A synthesised `tool_use` and its `tool_result`
  replay (S3).
* **The tools.** They may be omitted (S4). D8 keeps them, so that the tool
  definitions stay the same from turn to turn.

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
