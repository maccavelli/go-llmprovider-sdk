---
status: in-progress
date: 2026-10-10
associated-madr: "0031-MADR-native-streaming-on-the-responses-wire.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Native Streaming on the Messages, Chat Completions, generateContent and Interactions Wires

Associated MADR: [0031-MADR-native-streaming-on-the-responses-wire.md](0031-MADR-native-streaming-on-the-responses-wire.md),
D8 and its amendment of 2026-10-10 (E1 to E7, Q5 to Q8). The Responses wire
was implemented by
[0031-PLAN-native-streaming-on-the-responses-wire.md](0031-PLAN-native-streaming-on-the-responses-wire.md),
whose rules, decoder and conformance check this plan reuses.

## Goal

When it is done, every decision of the amendment holds, each proven red
first, and `v1.6.0` is released:

* **E1:** each provider and route Phase 1 shows streaming implements native
  streaming on its wire, and declares it:
  * `claude`, `gemini`, `huggingface`, `kilo`, `together`, `ollama`;
  * OpenCode's `messages`, `chat_completions` and `google` routes.

  One that does not stream keeps its declaration, by a recorded deviation.
* **E2:** one SSE reader in `internal/wire` serves all five wires, and the
  Responses benchmarks hold.
* **E3:** each wire's `Events` assembles the items its `Decode` returns.
  `EventDone`'s `Response` equals `Decode`'s for the same answer.
* **E4:** `Generate` sends and decodes exactly as at `v1.5.0`, and the wire
  goldens are unchanged.
* **E5:** usage in a stream is as Q5 decides.
* **E6:** every new streaming provider passes D7's check.

## Scope

### In scope

| Phase | Decisions | Files (production) |
| :--- | :--- | :--- |
| 0 | records | the amendment, this PLAN, `docs/README.md` |
| 1 | measurement for E1, E3, E5 | `llmprovider/live_0031b_measure_test.go` (new, `live_gateways`) |
| 2 | E2 | `llmprovider/internal/wire/sse.go` (new), `stream.go` beside `post.go`, `llmprovider/internal/wire/responses/responses.go` |
| 3 | E3 to E6, Messages | `llmprovider/internal/wire/messages/messages.go`, `llmprovider/providers/claude/claude.go`, `llmprovider/providers/opencode/opencode.go` |
| 4 | E3 to E6, Chat Completions | `llmprovider/internal/wire/chatcompletions/chatcompletions.go`; the `huggingface`, `kilo`, `together`, `ollama` and `opencode` providers |
| 5 | E3 to E6, generateContent | `llmprovider/internal/wire/generatecontent/generatecontent.go`, `llmprovider/providers/opencode/opencode.go` |
| 6 | E3 to E6, Interactions | `llmprovider/providers/gemini/interactions.go`, `gemini.go` |
| 7 | docs, release `v1.6.0` | `docs/architecture.md`, `README.md`, the package docs |

Each phase adds the tests named in its steps, beside the code they test, and
each wire phase adds its providers' live stream tests.

### Out of scope

* `WithRetry` stays buffered (0031-MADR D6, Q1), and call-argument deltas
  stay unyielded (Q4).
* `Generate` reading a stream (Q3).
* The other gobble-cli items, under their own records:
  * a per-tool strict flag and a tool result's error flag;
  * model metadata;
  * image input;
  * prompt caching;
  * tool deferral;
  * timeouts.
* gobble-cli's own changes: the `v1.5.0` upgrade, and streaming from the
  unwrapped provider.
* Push and tags, which are the owner's.

## Rules for every phase

The first plan's rules 1 to 9 apply unchanged:

1. approval, and naming billed requests;
2. red first, and plants only on a scratch copy;
3. the 22-check gate;
4. coverage floors;
5. benchmarks;
6. live checks: keys checked for presence only, and nothing logged but
   types and statuses;
7. deviations stop and prompt;
8. one commit per phase, by the owner or on a same-turn ask;
9. no identifiers.

Two more apply here:

10. **A wire phase builds only what Phase 1 showed streaming.** A provider
    or route left out is named in the phase's record.
11. **EventDone equals Decode.** Every wire phase pins, for each scripted
    stream, that `EventDone`'s `Response` equals that wire's `Decode` on the
    same answer, written as one JSON reply.

## Implementation Steps

### Phase 0: records

1. The amendment is answered (Q5 to Q8: each recommended option, the
   owner's of 2026-10-10) and marked accepted; this PLAN is `proposed`.
2. `docs/README.md` indexes this PLAN, 67 records.
3. `make records-check` clean; the owner commits.

### Phase 1: measurement (live; billed: at most 26 requests)

Plain requests are capped at 16 output tokens. Reasoning requests use the
smallest output the wire allows with reasoning on: for Claude, a 1,024-token
thinking budget and 1,040 output tokens.

1. **T2, does each wire stream.** `TestLive_0031bStreamReplies`, in
   `llmprovider/live_0031b_measure_test.go` (`live_gateways`; runs only when
   `LLMPROVIDER_LIVE_0031B` is set).
   * **How it measures.** As the first plan's T1 did: a recording transport
     turns each provider's own request into a streamed one, so every other
     header is the provider's. It logs per request:
     * the status;
     * the `Content-Type`;
     * the ordered, de-duplicated event names;
     * for Chat Completions, the distinct `delta` keys.

     It never logs text or a key.
   * **Targets:**
     * `claude`, and OpenCode Go's `messages` route (Messages);
     * `huggingface`, `kilo`, `together`, OpenCode Go's `chat_completions`
       route, and `ollama` when a local server answers (Chat Completions);
     * Google's own generateContent endpoint with `GEMINI_API_KEY`, and
       OpenCode Zen's `google` route (generateContent, Q7);
     * `gemini` (Interactions).

     A target whose key is absent is skipped and recorded as not measured.
   * **Requests:** a plain one and a reasoning one per target. For Chat
     Completions the plain request carries `stream_options: {"include_usage":
     true}`; if it is refused, it is sent once more without the field (E5).
     For one target per wire (`claude`, `together`, Google's endpoint,
     `gemini`) there is also a forced-tool request, which records the call's
     fragment shape.
2. **Rule (E1).** A target streams when a request answers 2xx with an event
   stream that ends in its wire's completion event:
   * `message_stop` for Messages;
   * a `finish_reason`, and `[DONE]`, for Chat Completions;
   * a `finishReason` for generateContent;
   * Interactions' completion, named from T2.

   A stream that ends at the output cap counts, as the first plan's D2
   decided. Any other outcome is a deviation.
3. **Rule (E3).** The names that become deltas are exactly the ones T2
   shows. A name in the MADR's mapping that T2 does not show is not mapped.
4. **Rule (E5).** A gateway whose stream carries no usage takes Q5's answer.
5. Gate; record T2; stage the measurement test.

### Phase 2: the shared reader (E2)

1. **Benchmarks before:** `BenchmarkReadStream_Deltas4MiB` and
   `BenchmarkReadStream_4MiB`.
2. **Tests, red first** (`internal/wire/sse_0031b_test.go`):
   * events joined from several `data:` lines;
   * a comment line and an `event:` line ignored;
   * an event over 16 MiB is `ErrIncomplete`;
   * a read failure is `ErrProviderUnavailable` marked `AfterReply`;
   * an end before the caller says done is `ErrProviderUnavailable` marked
     `AfterReply`;
   * a callback that stops reads no further.
3. **Change.** The SSE framing moves from `responses.readEvents` into
   `internal/wire`. `responses` keeps its event mapping and fast skip on top
   of it. `responses.Stream`'s loop moves to `internal/wire` as a generic
   stream over a decoder function, and the Responses providers call it with
   `responses.Events`.
4. Every existing `responses` and `wire` test, the first plan's stream and
   conformance tests, and the goldens pass unchanged.
5. **Benchmarks after:** within 10% of step 1, or a deviation.
6. Gate; stage.

### Phases 3 to 6: one wire each

Each wire phase runs the same steps, for Messages (3), Chat Completions (4),
generateContent (5) and Interactions (6):

1. **Decoder tests, red first,** in the wire's package
   (`events_0031b_test.go`), over scripted streams in T2's shapes:
   * the event sequence for a text answer, a reasoning answer and a call;
   * `EventDone`'s `Response` equals `Decode`'s on the same answer as one
     JSON reply (rule 11);
   * the joined text deltas equal the message text;
   * a refusal, a cut answer and an empty answer give `Decode`'s result;
   * a failure after a delta yields the delta, then the classified error,
     and no `EventDone`;
   * stopping early reads no further.
2. **Change:** the wire's `Events`, on Phase 2's reader.
3. **Provider tests, red first.** For each provider and route on the wire
   that Phase 1 showed streaming:
   * `TestStream`: the request asks for a stream, and the events are the
     decoder's;
   * `TestGenerate_Unchanged`;
   * the capability test's expected value moves to its new declaration.
4. **Change:**
   * each provider gains `Stream`, as the first plan's providers did:
     `Capabilities.Check`, then the generic stream with its wire's
     `Events`;
   * OpenCode's `Stream` sends the wire's route natively, in place of its
     fallback;
   * package docs and declarations change (E1, Q5, Q6, Q7).
5. **Conformance:** each such provider's harness sets `StreamText` and
   `StreamFailure` in its wire's format, and passes D7's check.
6. **Plants,** on a scratch clone, for the guards a red run does not
   exercise:
   * the provider's `Generate` sending a stream flag;
   * the decoder yielding an item out of `Decode`'s order.
7. **Live:** a `TestLive_<Provider>Streams` per streaming provider, as the
   first plan's Phase 4: at least one text delta before `EventDone`, and a
   refused or unfunded model recorded as not run. The billed count is named
   when the phase is asked.
8. Gate; stage.

When the last OpenCode route streams, both gateways' declarations move from
`BestEffort` to `Supported` (D8), in that phase.

### Phase 7: documentation and release

1. **`docs/architecture.md`:**
   * the streaming paragraphs: which providers stream, and how each wire
     maps;
   * "What is not here": no wire left on the fallback, or the ones left
     named.
2. **Package docs and `docs/guides/api-standards.md`:** only where a text no
   longer holds.
3. **Release notes** in this PLAN's Execution Record, as `v1.6.0`;
   `README.md`'s Status names `v1.6.0`.
4. **The live suites** on the release commit; a failure is a deviation, a
   transient one rerun once alone.
5. **The owner** commits, pushes, and tags `v1.6.0`.
6. **After the tag:**
   * CI on the tag;
   * the proxy;
   * apidiff `v1.5.0` to `v1.6.0`, which must show nothing incompatible;
   * a scratch consumer, and a scratch copy of gobble-cli moved to
     `v1.6.0`, built and tested.
7. This PLAN `complete`; the amendment's status recorded; the index updated.

## Verification

* **V1. Red, then green,** for every named test, with both lines in the
  Execution Record, and every plant's FAIL line.
* **V2. The gate** clean at the end of each phase; the goldens unchanged.
* **V3. Measurement:** T2 recorded before Phases 3 to 6 use it.
* **V4. Benchmarks:** Phase 2's before and after, within 10%.
* **V5. API:** apidiff against `v1.5.0` shows nothing incompatible.
* **V6. Live:** each wire phase's live tests, and the live suites on the
  release commit.
* **V7. Records:** the amendment amended for anything Phase 1 left
  unstreamed; `docs/architecture.md` and the package docs match the code.

## Rollout and Rollback

* **Rollout.** Seven commits after the records, one per phase, then
  `v1.6.0`. gobble-cli moves under its own record.
* **Rollback, before the tag:** a wire phase reverts alone. Phase 2 reverts
  only after every wire phase.
* **Rollback, after the tag:** fix forward in `v1.6.x`. A consumer that
  meets a defect pins `v1.5.0`, or calls `Generate`, which is unchanged.

## Execution Record

### Phase 0: records (2026-10-10)

The amendment was answered (Q5 to Q8, each recommended option) and marked
accepted, and this PLAN was indexed (67 records). The owner approved
execution; this PLAN is `in-progress`.

### Phase 1: measurement (2026-10-10)

`TestLive_0031bStreamReplies` (T2) ran at 23:02 UTC and
`TestLive_0031bGeminiShapes` (T2b, D2) at 23:07 UTC.

* **Setup:** both in `llmprovider/live_0031b_measure_test.go`, with
  `LLMPROVIDER_LIVE_0031B=1` and every key variable set (checked for
  presence only).
* **How they measure:** a recording transport turns each provider's own
  request into a streamed one:
  * `"stream": true`, and `stream_options` on Chat Completions' plain
    request;
  * `:streamGenerateContent?alt=sse` for generateContent;
  * `Accept: text/event-stream`.

  It logs each reply's status, its `Content-Type`, whether it ended with
  `[DONE]`, and its event shape: key paths, and the values of keys that
  name an event or a stop. It never logs text or a key.
* **Result:** exit 0 for both.
* **Billed:** T2, 20 requests, of which the 2 Ollama requests were local
  and the 2 to Zen were refused before generation. T2b, 4 more.

**T2:**

| Target | Request | Model | Status | Shape (abridged) |
| :--- | :--- | :--- | :--- | :--- |
| `claude` | plain, reasoning (1,024 budget), tool | `claude-haiku-4-5` | 200 ×3, `text/event-stream` | `message_start`, `content_block_start`/`_delta`/`_stop`, `message_delta`, `message_stop`, `ping`; deltas `text_delta`, `thinking_delta`, `signature_delta`, `input_json_delta`; stop reasons `end_turn`, `tool_use` |
| OpenCode Go `messages` | plain, reasoning | `qwen3.8-flash` | 200 ×2 | the same events; `thinking_delta`, `signature_delta`, `text_delta`; `max_tokens`, `end_turn` |
| `huggingface` | plain (with `include_usage`), reasoning | `deepseek-ai/DeepSeek-V4-Flash-0731` | 200 ×2 | `delta.content`, `delta.reasoning_content`; `finish_reason=length`; usage; `[DONE]` |
| `kilo` | plain (with `include_usage`), reasoning | `deepseek/deepseek-v4.1-flash` | 200 ×2 | `delta.reasoning`, `delta.reasoning_details[]` (`reasoning.text`), `delta.provider_metadata`; `length`; usage; `[DONE]` |
| `together` | plain (with `include_usage`), reasoning, tool | `deepseek-ai/DeepSeek-V4.1-Flash` | 200 ×3 | `delta.reasoning_content`; `delta.tool_calls[]` with `index`, `id`, `type`, `function.name`, `function.arguments`; `length`, `tool_calls`; usage on the plain request only; `[DONE]` |
| OpenCode Go `chat_completions` | plain (with `include_usage`), reasoning | `glm-5.3-flash` | 200 ×2 | `delta.content`, `delta.reasoning_content`, `delta.refusal`, `delta.tool_calls`; `length`; usage; `[DONE]` |
| `ollama` (local) | plain (with `include_usage`), reasoning | `llama3.2:latest` | 200, then 400 | plain: `delta.content`; `length`; usage; `[DONE]`. Reasoning: "\"llama3.2:latest\" does not support thinking", a property of the model |
| Google's endpoint | plain, reasoning, tool | `gemini-3.7-flash` | 200 ×3 | `candidates[].content.parts[].text`, `.thoughtSignature`; `finishReason` `MAX_TOKENS`, `STOP`; `usageMetadata`; no `[DONE]` |
| OpenCode Zen `google` | plain, reasoning | `gemini-3.5-flash-lite` | 402 ×2, `application/json` | "quota exhausted: opencode-zen/google HTTP 402 UNKNOWN: Upstream request failed: Insufficient account funds" |
| `gemini` | plain, reasoning, tool | `gemini-3.7-flash` | 200 ×3 | `interaction.created`, `interaction.status_update`, `step.start`, `step.delta`, `step.stop`, `interaction.completed`, `done`; deltas `text`, `thought_signature`; step types `thought`, `model_output`; statuses `incomplete`, `completed` |

The `include_usage` retry never ran: no gateway refused the field. On the
Gemini wires, the tool requests' 64-token caps went wholly to thinking: no
call was returned (D2).

**T2b** (D2): effort `low`, a 1,024-token cap, `gemini-3.7-flash`.

| Target | Request | Status | Shape (abridged) |
| :--- | :--- | :--- | :--- |
| Google's endpoint | reasoning | 200 | text parts with `thoughtSignature`; `STOP`; no `thought` part |
| Google's endpoint | tool | 200 | `parts[].functionCall` with `id`, `name`, `args`, sent whole; `STOP` |
| `gemini` | reasoning | 200 | steps `thought` (deltas `thought_signature` only) and `model_output` (`text`); `completed` |
| `gemini` | tool | 200 | step `function_call` with `id`, `name`, `signature`, `arguments`, and `arguments_delta` deltas; `requires_action` |

**Outcome, by rules 2 to 4:**

* Every target except Zen's `google` route streams. Zen's route stands on
  Google's endpoint (D1).
* The delta names are as 0031-MADR's measured note lists. Neither Gemini
  wire maps a reasoning delta: none was seen.
* Every Chat Completions gateway streams usage, so none takes Q5's
  `BestEffort`.

### Deviation D1 (2026-10-10): Zen's `google` route answered 402

* **Found** in step 1. Both requests to OpenCode Zen's `google` route
  answered 402 "Insufficient account funds", as the route had in the first
  plan's Phase 7. The route is OpenCode's only use of generateContent. Rule
  10 and Q7's answer left Phase 5 nothing measured to build.
* **Options put to the owner:**
  * fund the Zen account and re-measure the route (recommended);
  * build the route on Google's endpoint's evidence, OpenCode staying
    `BestEffort`;
  * defer Phase 5.
* **Decision.** The owner chose the second. Phase 5 builds the `google`
  route's native stream on Google's endpoint's evidence. Both OpenCode
  gateways stay `BestEffort`, and the package doc names the unmeasured
  route. 0031-MADR's amendment records this as a third exception to its
  "Measured, not assumed" driver, and revises Q7's answer.

### Deviation D2 (2026-10-10): the Gemini wires' caps went to thinking

* **Found** in step 1. On Google's endpoint and on `gemini`, the 16- and
  64-token caps were spent on thinking. No function call and no
  thought-summary delta was returned, so the call shape the decoders must
  assemble was unmeasured on both Gemini wires.
* **Options put to the owner:**
  * re-run a reasoning and a forced-tool request on each, at effort `low`
    and a 1,024-token cap (4 billed requests; recommended);
  * build from the documented shapes alone.
* **Decision.** The owner chose the re-run. It is T2b, above, added to the
  measurement file as `TestLive_0031bGeminiShapes`.

### Phase 2: the shared reader (2026-10-10)

**Before it.** The owner committed Phases 0 and 1 together as `67c7165`.

**Change.**

* `llmprovider/internal/wire/sse.go` (new):
  * `ReadEvents(provider, body, done, dispatch)`. The SSE framing moves
    here from `responses.readEvents`, with the same rules: data lines
    joined with "\n", other lines not data, empty events not dispatched,
    `EventLimit` (16 MiB) per event as `ErrIncomplete`, a read failure as
    `ErrProviderUnavailable` marked `AfterReply`. An early end is
    `ErrProviderUnavailable` marked `AfterReply`, and now names `done`, the
    event the stream lacked.
  * `readLine`, moved unchanged.
  * `StreamEvents(ctx, c, src, decode)`: the open-read-yield loop that was
    `responses.Stream`, generic over a decoder.
* `internal/wire/responses`:
  * `readEvents` keeps its event mapping and fast skip, and reads through
    `wire.ReadEvents(…, "response.completed", …)`, so its early-end text is
    unchanged.
  * `responses.Stream`, `eventLimit`, `readLine` and their errors are
    removed.
* `openai`, `grok` and `opencode` call `wire.StreamEvents(…,
  responses.Events)`.

**Red** (step 2), `internal/wire/sse_0031b_test.go` on the tree before the
change:

```text
llmprovider/internal/wire/sse_0031b_test.go:18:9: undefined: ReadEvents
llmprovider/internal/wire/sse_0031b_test.go:121:9: undefined: ReadEvents
FAIL	github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire [build failed]
```

**Green, under `-race`:**

* `TestReadEvents_Framing`
* `TestReadEvents_LastEventWithoutBlankLine`
* `TestReadEvents_EventTooLarge` (one line, and joined data)
* `TestReadEvents_ReadFailure`
* `TestReadEvents_EndsEarly`
* `TestReadEvents_StopReadsNoFurther`
* `TestReadEvents_DispatchError`

`go test -count=1 ./...` passes, which covers step 4: every `responses` and
`wire` test, the first plan's stream and conformance tests, and the
goldens.

**Plants** (rule 2), on a scratch clone:

| Plant | Test | Result |
| :--- | :--- | :--- |
| an event with no data is dispatched | `TestReadEvents_Framing` | `sse_0031b_test.go:37: events ["{\"a\":\n1}" "" "no space" "end"] (<nil>), want [...]`; FAIL |
| the early end does not name the event it lacked | `TestReadEvents_EndsEarly` | `sse_0031b_test.go:91: an early end: … stream ended early; want ErrProviderUnavailable after the reply, naming the end event`; FAIL |

**Benchmarks** (rule 5; `-count=5`, same host, medians):

| Benchmark | Before | After | Change |
| :--- | :--- | :--- | :--- |
| `ReadStream_Deltas4MiB` ns/op | 1,347,644 | 1,372,694 | +1.9% |
| `ReadStream_Deltas4MiB` B/op | 66,858 | 66,863 | 0.0% |
| `ReadStream_Deltas4MiB` allocs/op | 13 | 13 | 0 |
| `ReadStream_4MiB` ns/op | 7,485,438 | 7,526,427 | +0.5% |
| `ReadStream_4MiB` B/op | 860,498 | 860,468 | 0.0% |
| `ReadStream_4MiB` allocs/op | 7,120 | 7,120 | 0 |

Every figure is within step 5's 10%.

**Gate:** all 22 checks pass on a scratch copy with the phase's files staged.

* `internal/wire/responses` coverage is 99.2%.
* `internal/wire` coverage is 89.3%, down from 93.9%: `StreamEvents` is
  exercised by the provider packages' stream tests, not by `wire`'s own.
  The floors hold.

### Phase 3: Messages (2026-10-10)

Phase 1 showed `claude` and OpenCode Go's `messages` route streaming. OpenCode
Zen's `messages` route was not measured in Phase 1, and its live test below
was refused for want of funds; it is built with Go's route (rule 10), as D1
of the first plan stood Go for Zen.

**Change.**

* `internal/wire/messages`:
  * `Decode` now reads a `message` (model, stop reason, `[]block`, usage)
    and returns `message.response()`. That function holds Decode's rules
    unchanged: finish, empty answer, a call cut by the limit, an answer cut
    while thinking.
  * `block.item()` builds one block's item.
  * The new `Events` assembles a `message` from the stream and reads it
    through `wire.ReadEvents(…, "message_stop", …)`:
    * `message_start` gives the model and usage;
    * `content_block_start` gives the block;
    * `text_delta` and `thinking_delta` are yielded;
    * `signature_delta` and `input_json_delta` are accumulated;
    * `content_block_stop` finishes the block and yields its item, but a
      call whose input does not parse is not yielded: its error is
      `response()`'s at `message_stop`;
    * `message_delta` gives the stop reason and the cumulative usage;
    * `message_stop` yields `EventDone` with `response()`;
    * an `error` event is classified by its type.
* `claude`: `call(req, stream)` is shared by `generateOnce` and the new
  `Stream`, which adds `"stream": true` and `Accept: text/event-stream`.
  `NativeStreaming` is `Supported`, and the package doc says so.
* `opencode`: `Stream` streams the `messages` route natively with
  `messages.Events`, beside the `responses` route. The package doc names
  both routes. The gateways stay `BestEffort` (D1).

**Red** (steps 1 and 3), on the tree before each change:

```text
llmprovider/internal/wire/messages/events_0031b_test.go:29:9: undefined: Events
llmprovider/internal/wire/messages/events_0031b_test.go:181:9: undefined: Events
--- FAIL: TestClaude_Capabilities: claude_test.go:108: Capabilities() = {… NativeStreaming:Unsupported}, want {… NativeStreaming:Supported}
--- FAIL: TestStream (claude): stream_0031b_test.go:59: the provider does not stream natively
--- FAIL: TestStream_MessagesRouteNative: stream_0031_test.go:169: llmprovider: incomplete response: opencode-zen/messages: invalid character 'd' looking for beginning of value (and opencode-go)
```

**Green, under `-race`:**

* `TestEvents_EqualsDecode` has eight cases: text, reasoning, redacted
  thinking, a call, a call with no arguments, a refusal, an answer cut
  while thinking, and a call cut by the limit. Each case passes on three
  counts:
  * its events are pinned;
  * its `EventDone` equals `Decode` on the same answer as one JSON reply,
    or both fail with the same error (rule 11);
  * the text deltas, joined, equal the message text.
* `TestEvents_FailureAfterDelta` (`overloaded_error` →
  `ErrProviderUnavailable`), `TestEvents_StopEarly` and
  `TestEvents_EndsEarly` (naming `message_stop`).
* `claude`: `TestStream`, `TestGenerate_Unchanged`, and
  `TestClaude_Capabilities` at `Supported`.
* `opencode`: `TestStream_MessagesRouteNative` on both gateways. Its
  `TestStream_OtherRoutesFallBack` now covers the `chat_completions` and
  `google` routes.
* Conformance: `claude`'s and OpenCode's `go-messages` harnesses set
  `StreamText` and `StreamFailure` (a Messages stream, and one failing with
  `overloaded_error` after a delta). Both `D7-native-stream` subtests pass.
* `go test -count=1 ./...` passes.

**Plants** (step 6), on a scratch clone:

| Plant | Test | Result |
| :--- | :--- | :--- |
| `claude`'s `Generate` sends `"stream": false` | `TestGenerate_Unchanged` | `stream_0031b_test.go:92: Generate sent stream false, Accept ""; want neither`; FAIL |
| the Messages stream's `EventDone` items reversed | `TestEvents_EqualsDecode` | `/reasoning` and `/redacted_thinking`: `events_0031b_test.go:133: EventDone's Response … Output:[{Role:assistant Text:ok} {Text:think …}]; Decode's … Output:[{Text:think …} {Role:assistant Text:ok}]`; FAIL |

**Live** (step 7), `llmprovider/live_stream_0031b_test.go`, at 23:25 UTC.
Billed: 2 requests; Zen's was refused before generation.

| Test | Model | Result |
| :--- | :--- | :--- |
| `TestLive_ClaudeStreams` | `claude-haiku-4-5`, 16 tokens | PASS: 7 text deltas, then `EventDone` |
| `TestLive_OpencodeMessagesStreams/opencode-go` | `qwen3.8-flash`, a 1,024-token budget | PASS: 3 text deltas, then `EventDone` |
| `TestLive_OpencodeMessagesStreams/opencode-zen` | the same | SKIP, by 0027-MADR's transient rule: "quota exhausted: opencode-zen/messages HTTP 402 … Insufficient account funds"; not run |

**Gate:** all 22 checks pass on a scratch copy with the phase's files staged.
Coverage: `internal/wire/messages` 97.3%, `claude` 95.6%, `opencode` 97.4%.

### Phase 4: Chat Completions (2026-10-10)

Phase 1 showed the following streaming, each with usage under
`include_usage`, so each declares `Supported` (E5, Q6):

* `huggingface`
* `kilo`
* `together`
* `ollama` (local)
* OpenCode Go's `chat_completions` route

OpenCode Zen's `chat_completions` route is built with Go's, as in Phase 3.

**Change.**

* `internal/wire/chatcompletions`:
  * `decodeAs` reads a named `chatReply` and returns
    `chatReply.response(provider)`, which keeps its rules unchanged.
  * The new `Events` keeps a `chatStream`. Its chunks add up:
    * `id`, `model` and `usage` (the usage chunk);
    * the finish reason;
    * `delta.content` and `delta.refusal` as text deltas;
    * `delta.reasoning_content`, else `delta.reasoning`, as reasoning
      deltas;
    * call fragments assembled by `index`;
    * `reasoning_details` fragments merged by `index` (`text`, `summary`,
      `data` and `signature` joined; other fields keep their first value).
  * At `[DONE]`, `response()` gives the items, in `Decode`'s order, then
    `EventDone`. An error inside the 200, at the top level or in the
    choice, is classified as `Decode` classifies it. It reads through
    `wire.ReadEvents(…, "[DONE]", …)`.
* `internal/wire`: `StreamEventsFor(ctx, src, build, decode)` builds the
  request from the token it is sent with. Kilo's endpoint comes from its
  key. `StreamEvents` is now `StreamEventsFor` over a fixed call.
* `huggingface`, `together`, `ollama`:
  * `call(req, stream)` is shared by `generateOnce` and the new `Stream`;
  * a stream adds `"stream": true`, `stream_options: {"include_usage":
    true}` and `Accept: text/event-stream`.
* `kilo`: `call(req, token, stream)`, through `StreamEventsFor`.
* Each of the four declares `NativeStreaming: Supported`, and its package
  doc says so.
* `opencode`: `Stream` streams the `chat_completions` route natively, with
  `include_usage`. Only the `google` route still falls back.

**Red** (steps 1 and 3), on the tree before each change:

```text
llmprovider/internal/wire/chatcompletions/events_0031b_test.go:40:9: undefined: Events
--- FAIL: TestHuggingFace_Capabilities: huggingface_test.go:218: Capabilities = {… NativeStreaming:Unsupported}; huggingface_test.go:222: the provider does not implement Streamer
--- FAIL: TestKilo_Capabilities, TestTogether_Capabilities, TestOllama_Capabilities: likewise
--- FAIL: TestStream (huggingface, kilo, together, ollama): stream_0031b_test.go:57: the provider does not stream natively
--- FAIL: TestStream_ChatRouteNative: stream_0031_test.go:210: llmprovider: incomplete response: opencode-zen/chat_completions: invalid character 'd' looking for beginning of value
```

**A correction to a new test:** the first green run failed
`TestEvents_EqualsDecode/reasoning_content` and `/reasoning_with_details_(Kilo)`.
The expected lists I had written left out the `text_delta:ok` that their
scripted streams send. The code was right, and the expected lists were
corrected to include it.

**Green, under `-race`:**

* `TestEvents_EqualsDecode` has eight cases: text, `reasoning_content`,
  Kilo's `reasoning` with `reasoning_details`, two calls in fragments, a
  refusal, an answer cut while reasoning, a call cut by the limit, and no
  usage. In each:
  * the events are pinned;
  * `EventDone` equals `Decode` on the same answer, or both fail with the
    same error (rule 11);
  * the text deltas, joined, equal the message text.
* `TestEvents_ErrorInsideTheStream` (top level, and in the choice),
  `TestEvents_StopEarly`, and `TestEvents_EndsEarly` (naming `[DONE]`).
* For each of the four providers: `TestStream` (a stream with its usage),
  `TestGenerate_Unchanged` (no `stream`, no `stream_options`), and the
  capability test at `Supported`.
* `opencode`: `TestStream_ChatRouteNative` on both gateways.
  `TestStream_OtherRoutesFallBack` now covers the `google` route alone.
* Conformance: the four harnesses and OpenCode's `zen-chat` case set
  `StreamText` and `StreamFailure` (a Chat Completions stream, and one
  failing with an error inside the 200 after a delta). Every
  `D7-native-stream` subtest passes.
* `go test -count=1 ./...` passes.

**Fixed at the gate, without a deviation.** The first gate run failed
`make lint` on the new decoder:

* `errcheck` on three type assertions that discarded their ok value;
* `gocognit` complexity 79 for `Events`, above the limit of 50.

The code was changed:

* `Events` is split into `chatStream.done`, `.chunk`, `.delta` and
  `.detail`;
* the assertions check their ok value.

`golangci-lint` then reported `0 issues.`, and the tests, plants and gate
were rerun.

**Plants** (step 6), on a scratch clone, against the final code:

| Plant | Test | Result |
| :--- | :--- | :--- |
| `huggingface`'s `Generate` sends `"stream": false` | `TestGenerate_Unchanged` | `stream_0031b_test.go:95: Generate sent stream false, stream_options <nil>, Accept ""; want none`; FAIL |
| `huggingface`'s stream asks for no usage | `TestStream` | `stream_0031b_test.go:78: … stream true, stream_options <nil>; want a stream with its usage`; FAIL |
| the Chat Completions stream's items reversed | `TestEvents_EqualsDecode` | `/reasoning_content`, `/reasoning_with_details_(Kilo)`: items in the wrong order; `/calls`: `EventDone's Response … Output:[{CallID:call_2 …} {CallID:call_1 …}]`; FAIL |

The third plant's anchor was updated after the lint refactor moved the loop
it plants into.

**Live** (step 7), `llmprovider/live_stream_0031b_test.go`, at 23:46 UTC:

* a 512-token cap, since the gateways' default models reason first;
* billed: 4 requests, plus Zen's, which was refused before generation;
* Ollama's request was local.

| Test | Model | Result |
| :--- | :--- | :--- |
| `TestLive_HuggingFaceStreams` | `deepseek-ai/DeepSeek-V4-Flash-0731` | PASS: 3 text deltas, then `EventDone` |
| `TestLive_KiloStreams` | `deepseek/deepseek-v4.1-flash` | PASS: 8 text deltas, then `EventDone` |
| `TestLive_TogetherStreams` | `deepseek-ai/DeepSeek-V4.1-Flash` | PASS: 2 text deltas, then `EventDone` |
| `TestLive_OpencodeChatStreams/opencode-go` | `glm-5.3-flash` | PASS: 3 text deltas, then `EventDone` |
| `TestLive_OpencodeChatStreams/opencode-zen` | the same | SKIP, by 0027-MADR's transient rule: "quota exhausted: opencode-zen/chat_completions HTTP 402 … Insufficient account funds"; not run |
| `TestLive_OllamaStreams` | `llama3.2:latest`, 16 tokens | PASS: 7 text deltas, then `EventDone` |

The live-test meta checks (`TestLiveTests_TransientSkipsUseTheHelper`,
`TestLiveTransient_SkipsOnlyTransientClasses`) pass.

**Gate:** all 22 checks pass on a scratch copy with the phase's files staged.

| Package | Coverage |
| :--- | :--- |
| `internal/wire/chatcompletions` | 93.5% |
| `internal/wire` | 88.1% |
| `huggingface` | 93.5% |
| `kilo` | 95.7% |
| `together` | 95.1% |
| `ollama` | 95.8% |
| `opencode` | 97.4% |

### Phase 5: generateContent (2026-10-10)

The `google` route is built on Google's own endpoint's evidence (D1). Both
OpenCode gateways stay `BestEffort`.

**Change.**

* `internal/wire/generatecontent`:
  * `Decode` reads a named `reply` and returns `reply.response()`, which
    keeps its rules unchanged: the blocked prompt, the finish, a call cut by
    the limit, an answer cut while thinking. `reply.Usage` is now a pointer,
    so a chunk without usage leaves the last one in place.
  * The new `Events` reads `:streamGenerateContent?alt=sse` through
    `wire.ReadEvents(…, "a finishReason", …)`:
    * each text part is yielded as `EventTextDelta`;
    * parts are kept with `appendPart`, which joins a text part to a text
      part before it, and a thought part to a thought part, as a whole
      reply sends one of each;
    * the chunk with the `finishReason` gives the items, in `Decode`'s
      order, then `EventDone`;
    * Google's error body in the stream is classified by the status its
      code names (`ClassifyStreamFailure` with the code as a string);
    * a blocked prompt is `Decode`'s error.
  * Thought parts are not yielded as deltas: none was seen streaming
    (Phase 1, rule 3).
* `opencode`:
  * the `google` route streams natively at
    `{base}/models/{model}:streamGenerateContent?alt=sse`, with no `stream`
    field in the body;
  * every route now streams;
  * the package doc says why the gateways stay `BestEffort`: the `google`
    route's stream was measured on Google's endpoint, and Zen's routes
    stand on Go's measurements.

**Red** (steps 1 and 3), on the tree before each change:

```text
llmprovider/internal/wire/generatecontent/events_0031b_test.go:41:9: undefined: Events
--- FAIL: TestStream_GoogleRouteNative: stream_0031_test.go:114: llmprovider: incomplete response: opencode-zen/google: invalid character 'd' looking for beginning of value (and opencode-go)
```

**A test replaced.** `TestStream_OtherRoutesFallBack` asserted that the
`google` route, the last on the fallback, sends one request without
`"stream"`. That behaviour is changed on purpose by this phase. The test is
replaced by `TestStream_GoogleRouteNative`, which asserts the native stream
in its place. No route is left on the fallback to test.

**Green, under `-race`:**

* `TestEvents_EqualsDecode` has seven cases: text in two parts, thought
  then text, two calls, a filtered answer, an answer cut while thinking, a
  call cut by the limit, and a blocked prompt. In each:
  * the events are pinned;
  * `EventDone` equals `Decode` on the same answer as one reply, or both
    fail with the same error (rule 11);
  * the text deltas, joined, equal the message text.
* `TestEvents_ErrorInsideTheStream` (`503 UNAVAILABLE` →
  `ErrProviderUnavailable`), `TestEvents_StopEarly`, and
  `TestEvents_EndsEarly` (naming `finishReason`).
* `opencode`: `TestStream_GoogleRouteNative` on both gateways, with the
  path, `alt=sse`, and no `stream` field.
* Conformance: OpenCode's `zen-google` case sets `StreamText` and
  `StreamFailure` (a stream in two parts, and one failing with Google's
  error body). All four OpenCode `D7-native-stream` subtests pass.
* `go test -count=1 ./...` passes.
* A lint finding (`whitespace`: a blank line opening `reply.response`) was
  fixed before the gate.

**Plants** (step 6), on a scratch clone:

| Plant | Test | Result |
| :--- | :--- | :--- |
| the `google` route's stream sends `"stream": true` | `TestStream_GoogleRouteNative` | `stream_0031_test.go:124: sent /models/gemini-3.8-flash:streamGenerateContent?alt=sse, stream true; want … no stream field`; FAIL |
| streamed text parts are not joined | `TestEvents_EqualsDecode` | `events_0031b_test.go:100: events [text_delta:hel text_delta:lo item:MessageItem item:MessageItem done], want [… item:MessageItem done]` (and the thought case); FAIL |

**Live** (step 7), `TestLive_OpencodeGoogleStreams` in
`llmprovider/live_stream_0031b_test.go`, at 00:30 UTC on 2026-10-11:

* effort `low` and a 1,024-token cap (D2);
* billed: 1 request, plus Zen's, which was refused before generation.

`liveStreamKey`, added beside `liveStreamWith`, reads the key from a named
variable.

| Subtest | Model | Result |
| :--- | :--- | :--- |
| `google-endpoint`: the route's request, at Google's endpoint, with `GEMINI_API_KEY` | `gemini-3.7-flash` | PASS: 1 text delta, then `EventDone` |
| `opencode-zen` | `gemini-3.5-flash-lite` | SKIP, by 0027-MADR's transient rule: "quota exhausted: opencode-zen/google HTTP 402 … Insufficient account funds"; not run |

The live-test meta checks pass.

**Gate:** all 22 checks pass on a scratch copy with the phase's files staged.
Coverage: `internal/wire/generatecontent` 96.9%, `opencode` 96.7%.

### Phase 6: Interactions (2026-10-11)

Phase 1 showed `gemini` streaming on the Interactions API, so it declares
`Supported`.

**Change** (`llmprovider/providers/gemini`, since the Interactions wire is
this provider's alone):

* `interactions.go`:
  * `decodeInteraction` reads a named `interaction` (steps of
    `interactionStep`) and returns `interaction.response()`, which keeps its
    rules unchanged.
  * The new `interactionEvents` keeps an `interactionStream` and reads
    through `wire.ReadEvents(…, "interaction.completed", …)`:
    * `interaction.created` gives the id and the model;
    * `step.start` gives each step;
    * a `model_output` step's `text` delta is yielded;
    * `thought_signature` deltas are joined into the step's signature;
    * `arguments_delta` is accumulated as text fragments, or replaced by
      an object when one arrives whole, since Phase 1 showed the delta's
      key but not its form;
    * `interaction.completed` gives the status, usage, id and model, then
      the items, in `decodeInteraction`'s order, then `EventDone`;
    * an error event is classified by the status its code names.
  * Each step's text and argument builders are held by pointer. A
    `strings.Builder` must not be copied once written, and the slices grow.
* `gemini.go`: `call(req, stream)` is shared by `generateOnce` and the new
  `Stream`, which adds `"stream": true` and `Accept: text/event-stream`.
  `NativeStreaming` is `Supported`. The package doc says a thought summary
  is not streamed and arrives only in its item.

**Red** (steps 1 and 3), on the tree before each change:

```text
llmprovider/providers/gemini/events_0031b_test.go:34:9: undefined: interactionEvents
--- FAIL: TestGemini_Capabilities: items_test.go:134: Capabilities = {… NativeStreaming:Unsupported}; items_test.go:138: the provider does not implement Streamer
--- FAIL: TestStream (gemini): stream_0031b_test.go:54: the provider does not stream natively
```

A first build of the provider tests failed: the test's stream constant was
named `interactionStream`, which is also the decoder's type. The constant
was renamed `scriptedInteraction` before the red run above.

**Green, under `-race`:**

* `TestInteractionEvents_EqualsDecode` has four cases: text after a
  thought, a call with its arguments in deltas, a call with its arguments
  whole, and an incomplete interaction. Each `EventDone` equals
  `decodeInteraction` on the same interaction, or both fail with the same
  error (rule 11).
* `TestInteractionEvents_ErrorInsideTheStream` (`503 UNAVAILABLE` →
  `ErrProviderUnavailable`), `TestInteractionEvents_StopEarly`, and
  `TestInteractionEvents_EndsEarly` (naming `interaction.completed`).
* `TestStream`: a stream, with the interaction's id on `EventDone`'s
  `Response`. Also `TestGenerate_Unchanged`, and `TestGemini_Capabilities`
  at `Supported`.
* Conformance: the harness, unstored and stored, sets `StreamText` and
  `StreamFailure` (an Interactions stream, and one failing with an error
  event after a delta). Both `D7-native-stream` subtests pass.
* `go test -count=1 ./...` passes; `golangci-lint` reports `0 issues.`

**Plants** (step 6), on a scratch clone:

| Plant | Test | Result |
| :--- | :--- | :--- |
| `gemini`'s `Generate` sends `"stream": false` | `TestGenerate_Unchanged` | `stream_0031b_test.go:91: Generate sent stream false, Accept ""; want neither`; FAIL |
| the stream drops the interaction's id | `TestInteractionEvents_EqualsDecode` | `events_0031b_test.go:109: EventDone's Response &{ID: …}; decodeInteraction's &{ID:int_1 …}` (all three answer cases); FAIL |

**Live** (step 7), `TestLive_GeminiStreams` in
`llmprovider/live_stream_0031b_test.go`, at 01:16 UTC.

* Setup: `gemini-3.7-flash`, effort `low`, a 1,024-token cap (D2), with
  `gemini.WithStore(true)`, so that it also checks that a streamed answer
  carries the interaction's id that continuation needs. Billed: 1 request.
* Result: PASS. "gemini gemini-3.7-flash: 1 text deltas, then EventDone with
  an interaction id".

The live-test meta checks pass.

**Gate:** all 22 checks pass on a scratch copy with the phase's files staged.
`gemini` coverage is 96.3%.

With this phase, every built-in provider streams natively. `opencode-zen`
and `opencode-go` stay `BestEffort` (D1); every other provider is
`Supported`.

### Phase 7: documentation and release (2026-10-11)

**Steps 1 to 3**, on `727a2ad` (Phase 6):

* **`docs/architecture.md`:**
  * every provider's row names `Stream`, and OpenCode's says "native on
    every route";
  * the `internal/wire` row names `ReadEvents`, `EventLimit`,
    `StreamEvents` and `StreamEventsFor`;
  * the four wire rows name `Events`;
  * the request flow and the contract section say every provider streams,
    which declare what, how each wire maps its deltas, and that a Chat
    Completions stream asks for its usage;
  * the `llmtest` paragraph says every harness sets the stream fields;
  * "What is not here" names what remains: streamed reasoning on the
    Gemini wires, and live text through `WithRetry`.
* **Package docs:** changed in their phases. **`api-standards.md`:**
  unchanged; R13's text holds.
* **`README.md`:** Status names `v1.6.0`, with a `v1.6` bullet.

#### Release notes: `v1.6.0`

`v1.6.0` streams natively on every wire (0031-MADR, amendment of
2026-10-10). The exported API does not change.

**Native streaming.** `llmprovider.Stream` now yields text as it is
generated from every built-in provider:

| Provider | Wire | `NativeStreaming` | Streamed deltas |
| :--- | :--- | :--- | :--- |
| `claude` | Messages | `Supported` | text, thinking |
| `huggingface`, `kilo`, `together`, `ollama` | Chat Completions | `Supported` | text, refusal, reasoning (`reasoning_content` or `reasoning`) |
| `gemini` | Interactions | `Supported` | text |
| `openai`, `grok` | Responses (`v1.5`) | `Supported` | text, refusal, reasoning summary |
| `opencode-zen`, `opencode-go` | all four, by route | `BestEffort` | as each wire |

* Every stream yields each finished item as `EventItem`, and then
  `EventDone`, carrying the `Response` that wire's `Generate` would return
  for the same answer.
* Calls arrive as items; argument deltas are not yielded.
* A Chat Completions stream asks for its usage
  (`stream_options.include_usage`), which every gateway measured accepts.
* The OpenCode gateways stay `BestEffort`. Their `google` route's stream
  was measured on Google's own endpoint, because the Zen account was
  refused; Zen's other routes stand on Go's measurements.
* Neither Gemini wire streams a thought summary, so Gemini reasoning
  arrives only as an item.
* A stored Gemini interaction's id is on the streamed `Response`, so
  continuation works from a streamed answer.

**Failures, as in `v1.5`:**

* a failure is yielded once and ends the stream, with no `EventDone`;
* what was yielded before it is not a response;
* a 401 before the first event renews a refreshable credential;
* stopping the range closes the reply;
* an error a service sends inside its stream is classified as its
  non-streamed errors are.

**Unchanged:** `Generate` sends and decodes as in `v1.5.0` on every wire.
`WithRetry` still buffers.

**`llmtest`:** every built-in provider's harness now runs the native stream
check (`StreamText`, `StreamFailure`). The fields are as in `v1.5`.

**Upgrading:**

* No code change is needed.
* Code that reads `NativeStreaming` sees `Supported` for `claude`,
  `gemini`, `huggingface`, `kilo`, `together` and `ollama`.
* Code that calls `Stream` on them now receives text while it is
  generated, and should treat events before a failure as partial.
* gobble-cli receives live deltas once it streams from the provider rather
  than through `WithRetry` (its own record).
