---
status: in-progress
date: 2026-10-11
associated-madr: "0034-MADR-degrade-forced-tool-choice-per-model.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement a Per-Model Degradation of Forced Tool Choice

Associated MADR: [0034-MADR-degrade-forced-tool-choice-per-model.md](0034-MADR-degrade-forced-tool-choice-per-model.md)

## Goal

When it is done:

* **No refusal reaches the caller.** On every model that refuses a forced
  tool choice, a forced request is sent degraded: `auto`, with a turn
  instruction naming the tool.
  * This holds through the Claude provider and OpenCode's `messages` route.
  * The 400 never reaches the caller.
  * A model not yet known is learned after one refusal (D3).
* **Strict on Messages.** A set `Tool.Strict` is sent on the Messages
  wire.
* **The menu.** `staticClaude` lists the refusing models Phase 1 shows
  answering.
* **The live tests.** They no longer depend on `claude-haiku-4-5`.

Every other model sends exactly what it sends today. It ships as `v1.8.0` (amended 2026-10-11 from `v1.7.2`: the amendment adds exported API).

## Scope

### In scope

* **`llmprovider/internal/wire/messages`:**
  * the refusing-model table (D1);
  * `ToolChoice(choice, thinking, model)`, the shared mapping;
  * the degraded request (D2);
  * the learned-refusal memo (D3).
* **`llmprovider/providers/claude`:**
  * use of the shared mapping;
  * the one resend (D3);
  * Strict (D4);
  * the package doc.
* **`llmprovider/providers/opencode`:**
  * the same, on the `messages` route; `messagesToolChoice` is deleted;
  * Strict on that route if Phase 1 allows (D4);
  * the package doc.
* **`llmprovider/internal/wire/tools.go`:** `MessagesTools` sends a set
  `strict` (D4).
* **The catalog:** `staticClaude`, `modelLabels` and `waiting0033` (D5).
* **The live tests:**
  * the moves off `claude-haiku-4-5`;
  * `TestLive_ClaudeForcedChoiceDegrades` (D5);
  * `llmprovider/live_0034_measure_test.go` (Phase 1).
* **Wirecase:** a new Claude case on a refusing model. The goldens of
  existing cases change only as Phase 3 records.
* **Records:**
  * 0032-MADR's amendment (D4);
  * 0033-MADR's note that D6 is answered;
  * the docs, the index, and the release notes.

### Out of scope

* Structured outputs (Q2's fourth option).
* The token counting endpoint and Message Batches: the SDK uses neither.
* Kilo, Together and other gateways that may carry Claude models over Chat
  Completions. Their `tool_choice` is translated upstream, unmeasured.
* The metadata plan (`0033-PLAN-public-model-metadata.md`). A forced-choice
  field in `ModelInfo` waits for it, if ever needed.

## Rules for every phase

The rules of
[0031-PLAN-native-streaming-on-the-responses-wire.md](0031-PLAN-native-streaming-on-the-responses-wire.md)
apply unchanged:

1. approval, naming billed requests;
2. red first, and plants only on a scratch copy;
3. the 22-check gate;
4. coverage floors;
5. benchmarks, where a phase names them;
6. live checks: keys for presence only, nothing logged but statuses, shapes
   and redacted errors;
7. deviations stop and prompt;
8. one commit per phase, the owner's or on a same-turn ask;
9. no identifiers.

## Implementation Steps

### Phase 0: records

1. This PLAN and the MADR. The owner answered Q1 to Q4 on 2026-10-11, the
   recommended option of each. The MADR records them, and is accepted.
2. `docs/README.md`: two rows, the count, and an "I want to…" row.
3. Records check, links and markdownlint; stage.

### Phase 1: measurement (live; billed: at most 24 requests of at most 256 output tokens)

The tests are behind `LLMPROVIDER_LIVE_0034`, in
`llmprovider/live_0034_measure_test.go`. Each request's status, its
`stop_reason` and whether it holds a `tool_use` block are logged. Nothing
else is.

| Step | What | Model(s) | Requests |
| :--- | :--- | :--- | :--- |
| M1 | A forced choice (`GenerateToolCall`) | `claude-fable-5-1` | 1 |
| M2 | The refusal's exact error: status, `error.type`, message | `claude-sonnet-5-5` | 1 |
| M3 | `auto` alone, a control | `claude-sonnet-5-5`, `claude-opus-5-5`, `claude-fable-5-1` | 3 |
| M4 | `auto` and D2's mid-conversation system message, three times each | the same three | 9 |
| M5 | `auto` and D2's text in the top-level `system`, the fallback | `claude-sonnet-5-5` | 1 |
| M6 | `strict: true` on a schema with `additionalProperties: false`: forced on `claude-haiku-5-5` and `claude-sonnet-5`, and degraded on `claude-sonnet-5-5` | three | 3 |
| M7 | `strict: true` on a schema without `additionalProperties: false` | `claude-haiku-5-5` | 1 |
| M8 | `strict: true` on Go's `messages` route | `qwen3.8-flash`, `minimax-m3` | 2 |
| M9 | A forced choice, to confirm a refusal outside the table is recognised | `qwen3.8-flash` on Go's `messages` route | 1 |
| | | | **22** |

Each step decides something:

* **M1:** `claude-fable-5-1` stays in the table only if it refuses. If it
  accepts, that is a deviation.
* **M2:** the refusal is recognised as HTTP 400, `invalid_request_error`,
  and a message naming `tool_choice`.
* **M3 to M5:** the degraded request's call rate. Under 2 of 3 calls with
  the instruction (M4), on any model, is a deviation.
* **M6, M7:** D4 is built only if M6 answers 200 with a valid input.
  M7 answering 400 confirms Q2's reason for not forcing `strict`.
* **M8:** whether D4 covers OpenCode's `messages` route.
* **M9:**
  * whether the qwen refusal of 2026-10-02 matches D3's rule;
  * if it does, OpenCode learns it, and the package doc's note changes;
  * if not, the note stays, and qwen stays out of D3's reach.

Record a table per step, before Phase 2 uses it.

### Phase 2: the shared mapping and the table (D1, D2)

1. **Red first** (`internal/wire/messages/toolchoice_0034_test.go`):
   * **The table:** `refusesForcedChoice` is true for the four ids, in any
     case, and false for `claude-haiku-5-5`, `claude-sonnet-5`,
     `claude-opus-5-55` and `claude-sonnet-5-5-x`. Exact ids only.
   * **The mapping:** for a model not in the table, `ToolChoice` returns
     exactly what today's `toolChoice` returns for every choice and
     thinking value. The test is a table copied from today's
     `request_test.go` cases.
   * **A refusing model:**
     * `ForceTool("get_weather")` gives `{"type":"auto"}` and D2's system
       message naming `get_weather`;
     * `ToolChoiceRequired` gives `{"type":"auto"}` and the "one of the
       tools" text;
     * `ToolChoiceNone` and `auto` are unchanged.
   * **Placement:**
     * the system message follows the last `user` turn, and is the last
       entry;
     * with an input ending in an `assistant` turn, the text goes to the
       top-level `system`, and no system message is appended.
2. **Change:**
   * `ToolChoice` and the table in `internal/wire/messages`;
   * `claude.toolChoice` and `opencode.messagesToolChoice` replaced by it;
   * `FromItems` or the body builder appends the system message.
3. **Package docs:** the Claude and OpenCode Degradations entries name the
   four models, the degraded request, and `ErrIncomplete` on a text reply.
4. **Plants**, each failing its test:
   * the table matched by prefix (`claude-sonnet-5-5-x` degraded);
   * the system message dropped;
   * the system message placed before the last `user` turn;
   * the mapping changed for a model outside the table.
5. Gate; stage.

### Phase 3: proof on the wire (D1, D2)

1. **Wirecase:** a new Claude case, `claude-refuses-forced`, built with
   `WithModel("claude-sonnet-5-5")`, recorded for every scenario.
   * Its `tool` golden shows `{"type":"auto"}` and the system message.
   * Its `thinking-tool` golden shows `auto`, as before.
   * Every other case's goldens are unchanged.
2. **OpenCode:** a request-test row for `claude-opus-5-5` on Zen's
   `messages` route, with the same degraded body.
3. **Plant:** the Claude provider bypasses the shared mapping. The new
   golden fails.
4. Gate; stage.

### Phase 4: learning a refusal (D3; Q1 kept it)

1. **Red first** (`providers/claude/forced_learn_0034_test.go`), with a
   fake server:
   * a forced request to an unknown model answered by M2's refusal is sent
     once more, degraded, and its reply returned;
   * the second request to that model is sent degraded at once;
   * a 400 of another message is returned, with one request sent;
   * a refusal of the degraded resend is returned, with two requests sent;
   * the memo is bounded: past its limit, the oldest entry goes.

   OpenCode's `messages` route has the same tests.
2. **Change:**
   * the memo in `internal/wire/messages`, behind a mutex (R20);
   * the resend in each provider, beside `wire.Reauth`.
3. **Plants**, each failing:
   * a resend on any 400;
   * no memo, so every request is refused once;
   * a second resend.
4. Gate; stage.

### Amendment (2026-10-11): prevention first

*Added by the owner on 2026-10-11, after Phase 4 was committed (`0ed27fa`).
It replaces a Phase 4b that moved the resend into the shared openers, which
was never committed; that work is Phase 4f below.*

**The research, read 2026-10-11:**

* Vercel's AI SDK (`packages/anthropic`: `rejectsForcedToolUse`, warnings,
  structured outputs for its forced-JSON path).
* LiteLLM (`supports_forced_tool_use`, `UnsupportedParamsError` unless
  `drop_params`).
* OpenRouter's `supported_parameters`.
* Goose (`aaif-goose/goose`, read in a local checkout at `01eb01a`).
* OpenCode, Kilo Code, pi, Codex and grok-build (local checkouts), and
  Claude Code's changelog.
* models.dev's schema, Anthropic's errors, define-tools and Models API
  pages, and Anthropic's Swift bridge PR #34.
* ACP's initialization, prompt-turn notices and extensibility pages, and
  the local ACP Go SDKs.
* JSON-RPC 2.0, MCP 2026-07-28, RFC 9110, RFC 9457, the WHATWG SSE spec,
  gRPC's A6 retry design, net/http's transport, and the AWS SDK for Go v2's
  retryer.

The findings and the owner's answers (Q5 to Q8) are in 0034-MADR,
"Amendment (2026-10-11): prevention first".

**Phases 4b to 4f** build D7 to D10 and the narrowed D3. Phases 5 to 7
follow as written, except that the release is `v1.8.0`.

### Phase 4b: measurement (live; billed: at most 8 requests of at most 512 output tokens)

Behind `LLMPROVIDER_LIVE_0034`, in `llmprovider/live_0034_measure_test.go`.
The test logs statuses, `stop_reason`, the content blocks' types, and
whether the text parses against the schema.

| Step | What | Model(s) | Requests |
| :--- | :--- | :--- | :--- |
| S1 | `output_config.format` with a strict-ready tool schema; the tools offered with `tool_choice: none` | `claude-sonnet-5-5`, `claude-opus-5-5`, `claude-fable-5-1` | 3 |
| S2 | S1 with a schema lacking `additionalProperties: false` | `claude-sonnet-5-5` | 1 |
| S3 | a follow-up turn replaying S1's answer as a synthesised `tool_use`, with a `tool_result` | `claude-sonnet-5-5` | 1 |
| S4 | S1 with the tools omitted, and a history holding a `tool_use` | `claude-sonnet-5-5` | 1 |
| S5 | S1 with `Reasoning` set (adaptive thinking and an effort) | `claude-opus-5-5` | 1 |
| | | | **7** |

What each step decides:

* **S1:** D8 is built only if each model answers 200 with schema-valid JSON
  text and no `tool_use` block.
* **S2:** confirms the 400 that D8 returns.
* **S3:** a synthesised call must replay. If it does not, that is a
  deviation.
* **S4:** whether the tools must stay offered: if S4 answers 400, D8 keeps
  them, with `tool_choice: none`.
* **S5:** structured outputs alongside thinking.

### Phase 4c: warnings (D7)

1. **Red first** (`llmprovider/warning_0034_test.go`; and in
   `internal/wire/messages`, `providers/claude`, `providers/opencode`):
   * `Warning` and `WarningKind` exist, with the two constants;
   * a degraded request's `Response.Warnings` holds a
     `WarningForcedChoiceDegraded` naming the model;
   * so does 0020-MADR F43's thinking case, whose wire stays the same;
   * a request that is not degraded has none;
   * a stream's `EventDone` response carries the same.
2. **Change:**
   * the type, the constants and the field;
   * the Messages wire reports what `AddToolChoice` did;
   * Claude and OpenCode put it on the decoded `Response`, and on the
     stream's done event.
3. **Plants:**
   * the warning dropped;
   * a warning on a request not degraded;
   * the stream omits it.
4. Gate; stage. apidiff shows the additions.

### Phase 4d: the fact and the policy (D9, D10)

1. **Red first:**
   * **Normalisation:**
     * `anthropic/claude-sonnet-5-5`, `anthropic.claude-opus-5-5-v1:0`,
       `claude-fable-5-1-20260901` and `claude-sonnet-5-5@20260928` match
       the table;
     * `claude-sonnet-5-55` and `claude-sonnet-5` do not.
   * **Layers:**
     * `WithModelForcedChoice(m, Supported)` overrides the table;
     * `(m, Unsupported)` marks an unknown model;
     * the learned memo sits below both;
     * unknown accepts.
   * **The policy:**
     * by default, `ToolChoiceRequired` on a refusing model fails before
       the network, with an error matching `ErrUnsupported`, and no request
       is sent (R11);
     * with `ForcedChoiceDegrade`, it sends D2's degraded request and a
       warning.
   * **llmtest:** every harness still passes; `ForcedToolChoice` stays
     `BestEffort`.
2. **Change:**
   * the options `WithForcedChoiceFallback` and `WithModelForcedChoice`, in
     `llmprovider`, as common options (R17);
   * `ForcedChoiceFallback` and its constants;
   * the resolution in `internal/wire/messages`;
   * both providers pass their settings;
   * the package docs.
3. **Plants:**
   * normalisation off;
   * the override below the table;
   * degrading by default;
   * the strict refusal sent to the network.
4. Gate; stage.

### Phase 4e: a forced named tool as structured outputs (D8)

1. **Red first:**
   * **`internal/wire/messages`:** a `ForceTool(name)` body on a refusing
     model carries:
     * `output_config.format` with that tool's schema;
     * the tools;
     * `tool_choice: none`, or the tools omitted, as S4 decides;
     * no turn instruction.
   * **The decoder:** a text reply becomes a `FunctionCallItem` for the tool,
     with `Arguments` the compacted JSON, a synthesised call id and
     `FinishToolCalls`. A stream yields no text deltas for it, and yields
     the item and the done event.
   * **`GenerateToolCall`** on a refusing model returns the call, under
     either policy.
   * **Wirecase:** `claude-refuses-forced`'s `tool` and `thinking-tool`
     goldens change from Phase 3's degraded body to the structured-outputs
     body. The record lists both, with the reason. No other golden changes.
2. **Change:**
   * the body;
   * the reply's conversion in both providers' decode and stream paths;
   * the package docs.
3. **Plants:**
   * the turn instruction sent instead;
   * the JSON returned as text;
   * text deltas streamed;
   * the wrong tool named.
4. Gate; stage.

### Phase 4f: the narrowed resend, in the shared openers (D3)

1. **Red first:**
   * **`internal/wire/retry_0034_test.go`:** the send opener and the stream
     opener take a builder and an optional retry check.
     * A refused send with the check true is rebuilt, and sent once more.
     * The check false returns the refusal, after one request.
     * A refused resend is returned, with two requests.
     * Success never consults the check.
     * A 401 is still renewed inside each send.
     * A stream never resends after an event.
   * **`internal/wire/messages`:** a refusal from a model no layer knows is
     learned.
     * A named tool is resent as D8.
     * `ToolChoiceRequired` under refuse returns an error matching both
       `ErrUnsupported` and `*APIError`, with no resend.
     * Under degrade, it is resent degraded.
     * Each adds a `WarningForcedChoiceLearned` naming the model.
   * **The provider tests of Phase 4,** updated for the policy: by
     default, a learned `ToolChoiceRequired` no longer resends.
2. **Change:**
   * the openers;
   * `SendLearning` and `StreamLearning` replaced by the check;
   * both providers use the openers.
3. **Plants:**
   * a resend after an event;
   * a second resend;
   * the first request reused, not rebuilt;
   * the check consulted on success;
   * a resend under refuse;
   * no learned warning.
4. Gate; stage. The Phase 4 record's `SendLearning` and `StreamLearning`
   are noted as superseded.

### Phase 5: Strict on Messages (D4)

1. **Red first:**
   * `MessagesTools` sends `"strict": true` and `false` when set, and
     nothing when nil;
   * the Claude harness sets `SendsStrict: true`, so its Fidelity check
     fails until the change.
2. **Change:**
   * `wire.MessagesTools`;
   * the Claude doc;
   * OpenCode's doc and harness, if M8 allows;
   * 0032-MADR's amendment.
3. **Goldens:** the `flags` goldens of `claude`, `opencode-zen-messages`
   and `opencode-go-messages` gain `strict`, `true` then `false`, as
   0032's scenario sets. They are regenerated only for that, and every
   other golden is unchanged.
4. **Plants:**
   * `strict` dropped on Messages;
   * `strict` sent when nil.

   Each fails.
5. Gate; stage.

### Phase 6: the catalog and the live tests (D5)

1. **Red first:**
   * `TestStatic_NoRetiredIDs_0033`'s `waiting0033` loses the models
     added, as Q4 decides;
   * `TestStaticIDs_Labelled` fails until each has a label.
2. **Change:** `staticClaude` (within `MaxListed`), `modelLabels`, and the
   three live-test moves.
3. **`TestLive_ClaudeForcedChoiceDegrades`:** `GenerateToolCall` on each
   refusing model in the menu returns the call, or `ErrIncomplete`, and
   never `ErrInvalidRequest`. It is billed: one request per model, in the
   live suites.
4. **Plant:** on a scratch copy, with the table emptied, the live test
   fails with the 400. That is one billed request.
5. Gate; stage.

### Phase 7: documentation and release

1. **Docs:**
   * `docs/architecture.md`: the Messages paragraph;
   * `README.md` Status, `v1.8.0`;
   * release notes in this PLAN;
   * 0033-MADR: D6 answered by this record.
2. **The live suites** on the release commit (billed, as before).
3. **The owner** commits, pushes, and tags `v1.8.0`.
4. **After the tag:**
   * CI;
   * the proxy;
   * apidiff `v1.7.1` to `v1.8.0`: D7, D9 and D10's additions only;
   * a scratch consumer, and a scratch copy of gobble-cli.
5. This PLAN `complete`; the MADR's status recorded; the index updated.

## Verification

* **V1. Red, then green,** with both lines and every plant's FAIL line in
  the Execution Record.
* **V2. The gate** at each phase. Goldens change only as Phases 3, 4e and 5
  record.
* **V3. Measurement:** Phase 1's tables recorded before Phase 2.
* **V4. API:** apidiff `v1.7.1` to `v1.8.0`: D7, D9 and D10's additions only.
* **V5. Live:** `TestLive_ClaudeForcedChoiceDegrades` and the live suites
  on the release commit.
* **V6. Records:**
  * 0032-MADR amended;
  * 0033-MADR noted;
  * each package doc names the degradation.

## Rollout and Rollback

* **Rollout.** Ten commits after the records (Phases 4b to 4f added on 2026-10-11), then `v1.8.0`.
* **Rollback, before the tag:** each phase reverts alone, newest first.
  Phase 6's catalog change reverts with Phase 2.
* **Rollback, after the tag:** fix forward in `v1.7.3`. A caller on a model
  outside the table is unaffected.

## Execution Record

### Phase 0: records (2026-10-11)

* The MADR and this PLAN were written `proposed`.
* The owner answered Q1 to Q4 with the recommended option of each, and the
  MADR was accepted with them.
* The owner committed the records as `9ad49eb`.
* The index has two rows, 74 records, and an "I want to…" row.

### Phase 1: measurement (2026-10-11)

`llmprovider/live_0034_measure_test.go`, behind `LLMPROVIDER_LIVE_0034`,
run at 15:43 UTC.

* A recording transport rewrote each request as its step needs.
* The test logs only each reply's status, `stop_reason`, whether it holds a
  `tool_use` block, the shape of that block's input, and an error's type and
  message.
* Each key was read from its variable, and sent only in its header.
* Every request asked for at most 256 output tokens, on the prompt "Record
  the city Paris."

| Step | Model | Result |
| :--- | :--- | :--- |
| M1, forced | `claude-fable-5-1` | 400 `invalid_request_error`: `tool_choice: type "tool" and "any" are not supported for this model.` |
| M2, forced | `claude-sonnet-5-5` | 400 `invalid_request_error`, the same message |
| M3, `auto` alone | `claude-sonnet-5-5`, `claude-opus-5-5`, `claude-fable-5-1` | 200 each, `stop_reason tool_use`, input valid JSON with `city` a string: 3 of 3 calls |
| M4, `auto` and the turn instruction, three times each | the same three | 200 each, `stop_reason tool_use`, valid input: 9 of 9 calls |
| M5, `auto` and the instruction in the top-level `system` | `claude-sonnet-5-5` | 200, `stop_reason tool_use`, valid input |
| M6, `strict: true` on a strict-ready schema | forced on `claude-haiku-5-5` and `claude-sonnet-5`; degraded on `claude-sonnet-5-5` | 200 each, `stop_reason tool_use`, valid input |
| M7, `strict: true` on a schema without `additionalProperties: false` | `claude-haiku-5-5` | 400 `invalid_request_error`: `tools.0.custom: For 'object' type, 'additionalProperties' must be explicitly set to false` |
| M8, `auto` and `strict: true`, on Go's `messages` route | `qwen3.8-flash`, `minimax-m3` | 200 each, `stop_reason tool_use`, valid input |
| M9, forced, on Go's `messages` route | `qwen3.8-flash` | 400, body `{"model":"qwen3.8-flash"}` |

**M9 was sent twice.**

* Its first reply was not in Anthropic's error shape, and the log, which
  read only `error.type` and `error.message`, showed neither.
* The test was changed to log a reply's body, bounded to 400 bytes, when
  neither field is present, and M9 alone ran again.
* That makes 23 billed requests, within the phase's 24.

**What each step decides:**

* **M1:** `claude-fable-5-1` refuses, so it stays in D1's table. The table
  is the four ids Anthropic names. `claude-mythos-5-1` is not offered to
  this account, and is not measured.
* **M2:** D3's rule is the reply seen here: HTTP 400,
  `invalid_request_error`, and a message naming `tool_choice`.
* **M3 to M5:**
  * the degraded request called the tool in 13 of 13 tries, well above
    the deviation line of 2 of 3;
  * `auto` alone also called it in 3 of 3, on this one-tool prompt. The
    instruction stays, as Anthropic advises, for prompts where the call is
    less obvious;
  * the top-level fallback works.
* **M6:** strict tool use is accepted on Messages, forced and degraded, so
  D4 is built.
* **M7:** a schema that strict mode cannot compile answers 400, as Q2's
  answer assumed. The SDK sends `strict` only where the caller set it.
* **M8:** OpenCode Go's `messages` route accepts `strict`, so D4 covers it
  (Q3's answer).
* **M9:** the qwen refusal on Go's `messages` route has no error type, and
  no message naming `tool_choice`.
  * D3's rule does not match it, so OpenCode does not learn it.
  * The package doc's note of 2026-10-02 stays: the request is sent as
    asked, and fails with an error matching `ErrInvalidRequest`.
  * The plan foresaw this outcome, so it is not a deviation.

**Pre-add:** the test file is clean.

### Phase 2: the shared mapping and the table (2026-10-11)

**Change.**

* **`llmprovider/internal/wire/messages/toolchoice.go`** (new):
  * `forcedChoiceRefused`, the four ids, and `refusesForcedChoice`, an
    exact, case-insensitive match;
  * `ToolChoice(choice, thinking, model)`: today's mapping, with a forced
    choice sent as `auto` on a refusing model as well as with thinking;
  * `AddToolChoice(body, choice, thinking, model)`, which sets
    `tool_choice`. On a refusing model, a forced choice also appends
    Anthropic's turn instruction as a `role: system` message after the last
    `user` turn. When the messages do not end with one, it adds the text to
    the top-level `system`, after any system text there.
* **The Claude provider and OpenCode's `messages` route** call
  `messages.AddToolChoice`. Their copies are deleted: `claude.toolChoice`
  and `opencode.messagesToolChoice`.
* **The Claude constants** `jsonKeyType` and `jsonKeyName` became unused
  (`unused`) and were removed.
* **Package docs:**
  * Claude's capability line and Degradations entry name the four models,
    the degraded request, and `ErrIncomplete` on a text reply;
  * OpenCode's `messages` route entry names them too.

**Red first** (`internal/wire/messages/toolchoice_0034_test.go`).

* The tests alone did not build:
  `undefined: refusesForcedChoice`, `undefined: ToolChoice`,
  `undefined: AddToolChoice`.
* With the three functions holding today's behaviour (no table, the old
  mapping, no instruction), exactly the new assertions failed:

  ```text
  toolchoice_0034_test.go:19: refusesForcedChoice("claude-fable-5-1") = false, want true
  toolchoice_0034_test.go:68: ToolChoice("tool:get_weather", thinking false) on claude-opus-5-5 = map[name:get_weather type:tool], want map[type:auto]
  toolchoice_0034_test.go:68: ToolChoice("required", thinking false) on claude-opus-5-5 = map[type:any], want map[type:auto]
  toolchoice_0034_test.go:105: last message = {"content":"hi","role":"user"}, want {"content":"Tool-use requirement for the current turn: …","role":"system"}
  toolchoice_0034_test.go:108: tool_choice = map[name:get_weather type:tool], want auto
  toolchoice_0034_test.go:131: system = <nil>, want the instruction
  toolchoice_0034_test.go:138: system = "Be brief.", want the caller's system, then the instruction
  ```

  The abridged list also had the other three refusing ids, the
  required-choice instruction and the message count. The two tests that
  pin unchanged behaviour passed on today's mapping, as they must:
  * `TestToolChoice_UnchangedOutsideTheTable_0034`;
  * `TestAddToolChoice_NoInstructionOtherwise_0034`.

**A defect found by the test, before green.**

* The first implementation named the tool from `choice.Tool()`. For
  `ToolChoiceRequired` that returns the choice's own string, so the
  instruction asked for "the required tool":

  ```text
  toolchoice_0034_test.go:117: ToolChoiceRequired's last message = {"content":"Tool-use requirement … a call to the required tool …"}, want the one-of-the-tools instruction
  ```

* The name is now used only when a tool is named.

**An assertion added after green.** On a refusing model, the instruction
is also added with thinking, as D2 applies to every forced request there.
It passed at once, so it was shown failing on a plant (below).

**Green:**

* `go test -race -count=1 ./llmprovider/...` passes;
* every G-wire golden is unchanged: the golden models, `claude-sonnet-5`
  and `qwen3.8-flash`, are outside the table.

**Lint.** The first pre-add found four issues, fixed in code:

* `errcheck` on two type assertions in `AddToolChoice`, now checked;
* `unused` on the two Claude constants.

**Plants** on a scratch clone, each against the `_0034` tests:

| Plant | FAIL line |
| :--- | :--- |
| the table matched by prefix | `refusesForcedChoice("claude-sonnet-5-5-x") = true, want false: exact ids only` |
| the system message dropped | `1 messages, want the user turn and the system message` |
| the system message before the last `user` turn | `last message = {"content":"hi","role":"user"}, want {"content":"Tool-use requirement …","role":"system"}` |
| the mapping changed outside the table | `ToolChoice("required", thinking false) = map[type:auto], want map[type:any]` |
| no instruction with thinking | `with thinking, last message = {"content":"hi","role":"user"}, want …` |

The plants ran on the final code. One plant's text was rewritten after the
lint fix, which had removed the variable it used, so that it fails its
test rather than the build.

**Pre-add:** 4 files clean.

### Phase 3: proof on the wire (2026-10-11)

**Wirecase.** `providers/claude/wire_test.go` gains a case,
`claude-refuses-forced`, built with `WithModel("claude-sonnet-5-5")`, with
the same canned replies.

* **Red:** its seven scenarios failed with `wiretest: no golden file
  testdata/wire/claude-refuses-forced/<scenario>.json; record one with
  -update`, and no other subtest failed. Continuation is not recorded: the
  P7 type had none.
* **Green:**
  * its goldens were recorded once, alone
    (`-run 'TestWireGoldens/claude-refuses-forced/' -update`);
  * `TestWireGoldens` passes at `-count=3` across `./llmprovider/...`;
  * no existing golden changed.
* **What they pin**, read from each:

| Scenario | `tool_choice` | `thinking` | Last message |
| :--- | :--- | :--- | :--- |
| `tool` | `{"type":"auto"}` | none | `role: system`, the instruction naming `get_weather` |
| `thinking-tool` | `{"type":"auto"}` | `{"type":"adaptive"}` | `role: system`, the instruction naming `get_weather` |
| `text` | none | none | the user's prompt |
| `flags` | none (no forced choice) | none | the user's tool results |

The `thinking-tool` row differs from the plan's "shows `auto`, as before".
On a refusing model the instruction is added with thinking too, which
Phase 2 decided and asserted.

**OpenCode.** `providers/opencode/toolchoice_0034_test.go`,
`TestGenerate_RefusingModelDegrades_0034`. Zen routes `claude-opus-5-5` to
the `messages` route, by `routes_snapshot.json`:

* a named choice sends `{"type":"auto"}`, with a system message naming
  `get_weather`;
* `ToolChoiceRequired` sends `auto`, with the one-of-the-tools instruction;
* `ToolChoiceNone` and `auto` are unchanged, with no message appended.

It passed on Phase 2's code, so it was shown failing on a plant.

**Plants** on a scratch clone:

| Plant | Test | FAIL lines |
| :--- | :--- | :--- |
| the Claude provider bypasses the table (`AddToolChoice(…, "")`) | `TestWireGoldens` | `claude-refuses-forced/tool`: `tool_choice.type: want "auto", got "tool"`, `tool_choice.name: want (absent), got "get_weather"`, `messages[1]: want {…"Tool-use requirement…"}`; and `claude-refuses-forced/thinking-tool` |
| OpenCode's `messages` route bypasses the table | `TestGenerate_RefusingModelDegrades_0034` | `"tool:get_weather": tool_choice = map[name:get_weather type:tool], want map[type:auto]`; `last message = map[content:hi role:user], want a system message holding "a call to the get_weather tool"` |

**Pre-add:** 2 files clean.

### Phase 4: learning a refusal (2026-10-11)

**Change.**

* **`internal/wire/messages/learn.go`** (new):
  * **the memo:** `learned`, keyed by provider and lowercased model, oldest
    first, at most `learnedRefusalsLimit` (256) entries, behind a mutex,
    in memory only;
  * **`IsForcedChoiceRefusal(err)`:** HTTP 400, `invalid_request_error`,
    and a message naming `tool_choice`, matched through `errors.As`;
  * **`SendLearning(provider, model, choice, send)`:** when a forced request
    to a model not already known to refuse is refused that way, it learns
    the model and calls `send` once more, which builds the degraded request;
  * **`StreamLearning`:** the same for a stream refused before its first
    event.
* **`refusesForcedChoice(provider, model)`** is the table or the memo.
  `ToolChoice` and `AddToolChoice` take the provider, and Phase 2's tests
  pass it.
* **The Claude provider:**
  * `Generate` wraps its `wire.Reauth` call in `SendLearning`, and `Stream`
    its `wire.StreamEvents` in `StreamLearning`, with the key `claude`;
  * `modelFor(req)` names the model once, for both and for `body`.
* **OpenCode:**
  * `Generate` resolves the call once. On the `messages` route it wraps
    the send in `SendLearning`, with the gateway's id as the key; the other
    routes send as before.
  * `generateOnce` now takes the resolved call, so a reauth resend reuses
    the same route rather than resolving again. That is a change of
    behaviour only if a route could change between a 401 and its resend.
  * `Stream` on the `messages` route builds its body inside
    `StreamLearning`, so a resend is degraded.
* **Package docs:** Claude and OpenCode name the learning. OpenCode's
  notes that its qwen refusals have another form, and are not learned
  (Phase 1, M9).

**Red first.**

* **The provider tests** use only the public API, and built on the old
  code. They failed as the learning was missing:

  ```text
  forced_learn_0034_test.go:81: GenerateToolCall = {…}, llmprovider: invalid request: claude HTTP 400 invalid_request_error: tool_choice: type "tool" and "any" are not supported for this model.; want the call
  forced_learn_0034_test.go:118: tool_choice [tool], want tool then auto, and no third request
  forced_learn_0034_test.go:132: Stream: llmprovider: invalid request: claude HTTP 400 invalid_request_error: tool_choice: …
  learn_0034_test.go:45: GenerateToolCall: llmprovider: invalid request: opencode-zen/messages HTTP 400 invalid_request_error: tool_choice: …
  ```

  `TestGenerate_OtherBadRequestNotResent_0034` passed on the old code, as
  it must: nothing resent then.
* **The messages tests** did not build: `undefined: IsForcedChoiceRefusal`,
  `undefined: learnRefusal`, and `too many arguments in call to
  refusesForcedChoice`.

**Corrections while making them build**, none loosening an assertion:

* A Claude test read `ev.Kind`. The event's field is `Type`.
* `TestSendLearning_0034` indexed its replies past their end when a plant
  sent too many. It now returns an error for a send the case does not
  expect, so the plant fails on the assertion, not a panic.
* `errorlint` flagged two `!=` comparisons of errors, now `errors.Is`.

**OpenCode's tests added before green**, so that its route has what
Claude's has:

* `TestGenerate_MessagesRouteOtherBadRequest_0034`;
* `TestStream_MessagesRouteLearnsARefusal_0034`.

**A slip in my own script.** Its assert miscounted the Phase 2 test calls
it rewrites: 12 expected, 10 present. It had already written
`toolchoice.go`, and stopped before the tests and the providers. The
rest was applied by a second script, holding only the unapplied parts. The
tree was checked by building it.

**Green:** `go test -race -count=1 ./llmprovider/...` passes. Every G-wire
golden is unchanged.

**Plants** on a scratch clone, all against the final code:

| Plant | FAIL lines |
| :--- | :--- |
| a resend on any 400 | `IsForcedChoiceRefusal(… max_tokens: 999999 > 128000) = true, want false`; `another 400: 2 sends`; Claude `2 requests, want 1` |
| no memo | `a learned refusal is not remembered, in any case`; Claude `tool_choice [tool tool], want tool then auto`; OpenCode `GenerateToolCall: … HTTP 400` |
| a second resend | `refused twice: 3 sends …; want 2`; Claude `tool_choice [tool auto auto], want tool then auto, and no third request` |
| a Claude stream that does not learn | `Stream: llmprovider: invalid request: claude HTTP 400 …` |
| an unbounded memo | `the oldest refusal is kept past 256 entries` |
| OpenCode's `Generate` does not learn | `GenerateToolCall: llmprovider: invalid request: opencode-zen/messages HTTP 400 …` |
| OpenCode's stream opened without `StreamLearning` | `Stream: llmprovider: invalid request: opencode-zen/messages HTTP 400 …` |

A first OpenCode stream plant disabled the route check. That also took
the stream off the `messages` route, so it failed on a decode error, a
weaker proof. It was replaced by the plant in the table, which keeps the
stream and drops only the learning.

**Pre-add:** 8 files clean.

### Amendment (2026-10-11): prevention first, recorded

* **The decision.** The owner asked for the refusal handling to be
  re-assessed, with preventing degraded requests the priority. The
  research and the answers Q5 to Q8 are in 0034-MADR, "Amendment
  (2026-10-11): prevention first".
* **The records.**
  * Phases 4b to 4f replaced a Phase 4b that moved the resend into the
    shared openers, which was never committed.
  * The release became `v1.8.0`.
  * 0033's metadata plan moved to `v1.9.0`.
  * The owner committed the amendment as `0d5f668`.
* **What Phase 4 built that this changes.** Phase 4's `SendLearning` and
  `StreamLearning` stay in the tree until Phase 4f replaces them.

### Phase 4b: measurement (2026-10-11)

`TestLive_0034StructuredOutputs`, in `llmprovider/live_0034_measure_test.go`,
behind `LLMPROVIDER_LIVE_0034`, run at 17:35 UTC.

* The provider built each request, and a recording transport added
  `output_config.format` with a JSON schema, keeping any `output_config`
  effort.
* The test logs the status, `stop_reason`, the content blocks' types, and
  whether the text parses as `{"city": string}`; never the text.
* Each request allowed at most 512 output tokens.

| Step | Model | Result |
| :--- | :--- | :--- |
| S1, `output_config.format` with a strict-ready schema; the tool offered with `tool_choice: none` | `claude-sonnet-5-5`, `claude-opus-5-5`, `claude-fable-5-1` | 200 each, `stop_reason end_turn`, blocks `[thinking text]`, the text a JSON object with `city` a string and 1 key; no `tool_use` |
| S2, a schema without `additionalProperties: false` | `claude-sonnet-5-5` | 400 `invalid_request_error`: `output_config.format.schema: For 'object' type, 'additionalProperties' must be explicitly set to false` |
| S3, a synthesised `tool_use` (`toolu_so_1`) and its `tool_result` replayed, then a question | `claude-sonnet-5-5` | 200, `end_turn`, a text answer |
| S4, no tools, the same history holding a `tool_use`, `output_config.format` | `claude-sonnet-5-5` | 200, `end_turn`, `[thinking text]`, valid JSON |
| S5, S1 with `Reasoning` at `EffortLow` (adaptive thinking and an effort) | `claude-opus-5-5` | 200, `end_turn`, `[thinking text]`, valid JSON |

7 billed requests, within the phase's 8.

**What each step decides:**

* **S1:** D8 is built. On each refusing model, structured outputs return
  schema-valid JSON as text, with no tool call, beside the thinking block.
* **S2:** a schema strict mode cannot compile answers 400. D8 returns that
  error as the API sent it.
* **S3:** a synthesised call id replays, so a conversation can go on from
  D8's call.
* **S4:** the tools may be omitted, but D8 keeps them, with
  `tool_choice: none`, which S1 measured on all three models. The tool
  definitions then stay the same from turn to turn, which a cached prompt
  prefix needs.
* **S5:** structured outputs work beside adaptive thinking and an effort.

**A fix to the new helper before the run:** `soShape` first read `city`
from the map before decoding into it. That was harmless, but misleading, and
the line was removed.

**Pre-add:** the test file is clean.
