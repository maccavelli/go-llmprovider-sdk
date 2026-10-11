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

Every other model sends exactly what it sends today. It ships as `v1.7.2`.

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
   * `README.md` Status, `v1.7.2`;
   * release notes in this PLAN;
   * 0033-MADR: D6 answered by this record.
2. **The live suites** on the release commit (billed, as before).
3. **The owner** commits, pushes, and tags `v1.7.2`.
4. **After the tag:**
   * CI;
   * the proxy;
   * apidiff `v1.7.1` to `v1.7.2`, which must be empty;
   * a scratch consumer, and a scratch copy of gobble-cli.
5. This PLAN `complete`; the MADR's status recorded; the index updated.

## Verification

* **V1. Red, then green,** with both lines and every plant's FAIL line in
  the Execution Record.
* **V2. The gate** at each phase. Goldens change only as Phases 3 and 5
  record.
* **V3. Measurement:** Phase 1's tables recorded before Phase 2.
* **V4. API:** apidiff `v1.7.1` to `v1.7.2` empty.
* **V5. Live:** `TestLive_ClaudeForcedChoiceDegrades` and the live suites
  on the release commit.
* **V6. Records:**
  * 0032-MADR amended;
  * 0033-MADR noted;
  * each package doc names the degradation.

## Rollout and Rollback

* **Rollout.** Six commits after the records, then `v1.7.2`.
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
