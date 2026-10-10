---
status: in-progress
date: 2026-10-10
associated-madr: "0031-MADR-native-streaming-on-the-responses-wire.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Native Streaming on the OpenAI Responses Wire

Associated MADR: [0031-MADR-native-streaming-on-the-responses-wire.md](0031-MADR-native-streaming-on-the-responses-wire.md)

## Goal

When it is done, every decision of the MADR holds, each proven red first,
and `v1.5.0` is released:

* **D1:** `openai` (API key and ChatGPT session) and `grok` implement
  `Streamer` and declare `NativeStreaming: Supported`; `opencode-zen` and
  `opencode-go` implement it and declare `BestEffort`, streaming natively
  on the `responses` route and falling back on the other three. Any of
  these that Phase 1 shows does not stream keeps `Unsupported`, by a
  recorded deviation.
* **D2:** one incremental decoder in `internal/wire/responses` yields
  `Event`s; `ReadStream` collects it, with unchanged results.
* **D3:** a two-step streaming seam in `internal/wire`: open (which
  `wire.Reauth` wraps), then read.
* **D4:** only `EventDone` carries a `Response`; a failure ends the stream
  with no `EventDone`.
* **D5:** `Generate` sends and decodes exactly as at `v1.4.0`; the wire
  goldens are unchanged.
* **D6:** `WithRetry` is unchanged in code: not a `Streamer`,
  `NativeStreaming: Unsupported`; its doc says live deltas need the
  provider unwrapped.
* **D7:** `llmtest.Run` checks any `Streamer` whose harness supplies a
  streamed reply, and every built-in streaming provider's harness does.

## Scope

### In scope

| Phase | Decisions | Files (production) |
| :--- | :--- | :--- |
| 0 | records | this PLAN, 0031-MADR, `docs/README.md` |
| 1 | measurement for D1 and D2 | `llmprovider/live_0031_measure_test.go` (new, `live_gateways`) |
| 2 | D2 | `llmprovider/internal/wire/responses/responses.go` |
| 3 | D3, D4 | `llmprovider/internal/wire/post.go` (or a new `stream.go` beside it), `llmprovider/stream.go` (doc) |
| 4 | D1, D5 | `llmprovider/providers/openai/openai.go`, `llmprovider/providers/grok/grok.go`, `llmprovider/providers/opencode/opencode.go`, their package docs, new live tests |
| 5 | D6 | `llmprovider/retry.go` (doc only) |
| 6 | D7; D2's refusal row (D4) | `llmprovider/llmtest/llmtest.go`, the three providers' harnesses; `llmprovider/internal/wire/responses/responses.go` (D4) |
| 7 | docs, release `v1.5.0` | `docs/architecture.md`, `README.md`, `docs/guides/api-standards.md` if a rule's text no longer holds |

Each phase adds the tests named in its steps, beside the code they test.

### Out of scope

* The other four wires: Chat Completions, Messages, Gemini Interactions,
  generateContent (MADR D8; each a later amendment and phase).
* Streaming inside `Generate` (MADR Q3), function-call argument deltas
  (Q4), and live deltas through `WithRetry` (Q1).
* `0030-REPORT`: a report records what was observed on its date and is
  not rewritten.
* Consumers: prepare-commit-msg and gobble-cli move under their own
  records.
* Push and tags, which are the owner's.

## Rules for every phase

1. **Approval.** A phase starts on the owner's "proceed" (or "do phase
   N"). A phase that sends a billed request says so, and how many, when it
   asks.
2. **Red first.** Every new test is run on the tree before the change and
   its FAIL line recorded (a compile failure of a not-yet-written function
   counts, quoted); then it passes. A test that cannot fail on the tree (a
   guard of existing behaviour) is seen failing on a scratch copy with the
   behaviour planted out. Plants are never made in the tree.
3. **The gate** at the end of every phase that changes Go code:
   * `make pre-add-check`;
   * `CGO_ENABLED=0 go vet ./...` for linux, darwin and windows, with and
     without `-tags live_gateways`;
   * `go test -race -cover ./...` and `go test -shuffle=on ./...`;
   * `go mod tidy -diff`;
   * `make lint parity-check dep-check coverage-check api-check
     generate-check records-check gate-selftest`;
   * markdownlint on the lint scope, relative links, the identifier scan;
   * `go test -count=3 ./llmprovider/... -run TestWireGoldens` without
     `-update`: the goldens are unchanged (D5).
4. **Coverage floors** (`scripts/coverage-floors.txt`) hold. A floor
   changes only by a record.
5. **Benchmarks.** `BenchmarkReadStream_Deltas4MiB` and
   `BenchmarkReadStream_4MiB` (`responses/stream_limit_test.go`) are
   recorded with `-benchmem -count=5`, before and after Phase 2, on the
   same host, as medians.
6. **Live checks** read keys from the environment, checked for presence
   only, never printed. They log event types and statuses, never a
   payload's text beyond its `type`, and never a key.
7. **Deviations stop and prompt,** with evidence and resolutions, and are
   recorded here as a dated entry, and in the MADR when a decision or
   asserted fact changes, before work continues.
8. **Commits.** One commit per phase, `git commit --no-edit`: the owner's,
   or the agent's on a same-turn "commit to main".
9. **Identifiers.** Nothing committed carries a hostname, an account name or
   a machine path.

## Implementation Steps

### Phase 0: records

1. 0031-MADR is `accepted`, with the owner's answers of 2026-10-10; this
   PLAN is `proposed`.
2. `docs/README.md` indexes both, 66 records.
3. `make records-check` clean; the owner commits.

### Phase 1: measurement (live; billed: at most 8 requests of 16 output tokens, plus 2 on a ChatGPT session if it is switched on)

This phase is the only source of D1's gate and D2's reasoning event names.

1. **T1, does `/responses` stream.** `TestLive_0031StreamReplies`, in
   `llmprovider/live_0031_measure_test.go` (`live_gateways`; runs only when
   `LLMPROVIDER_LIVE_0031` is set):
   * targets: `openai` (`OPENAI_API_KEY`), `grok` (`XAI_API_KEY`),
     `opencode-zen` and `opencode-go` (`OPENCODE_API_KEY`, a model whose
     `routes_snapshot.json` entry is `@ai-sdk/openai`, chosen at run time);
     a ChatGPT session when `LLMPROVIDER_LIVE_CHATGPT` is also set. A
     target whose key is absent is skipped and recorded as not measured;
   * for each, two `POST {base}/responses` with `"stream": true`,
     `Accept: text/event-stream`, `max_output_tokens` 16 and the input
     `"hi"`, carrying the headers the provider package sends (OpenCode Go's
     session header included): one plain, one with reasoning effort `low`
     on a model the static catalog marks as reasoning;
   * it logs per request: status, `Content-Type`, and the ordered,
     de-duplicated list of event `type` values. It fails only on a
     transport error.

   *Amended by D1 (2026-10-10):* `opencode-zen`'s chosen model was
   refused to the key; Go's measurement stands for Zen.
2. **Rule (D1).** ~~A target whose plain request answers 2xx with an event
   stream ending in `response.completed` streams.~~ *Amended by D2
   (2026-10-10):* a target streams when either of its requests answers 2xx
   with an event stream ending in `response.completed` or
   `response.incomplete`. Any other outcome keeps that provider
   `Unsupported`, as a deviation and a MADR amendment.
3. **Rule (D2).** The reasoning delta events mapped to
   `EventReasoningDelta` are exactly the reasoning text and summary-text
   `*.delta` types T1 shows. None seen is a valid outcome: then no
   reasoning delta is mapped, and reasoning arrives only as `EventItem`.
4. Gate; record T1; stage the measurement test.

### Phase 2: the decoder (D2)

1. **Benchmarks before:** rule 5's two benchmarks on the tree.
2. **Tests, red first** (`responses/events_0031_test.go`):
   * `TestEvents_Goldens`: for each of the three `providers/openai/testdata/*.sse`
     goldens (copied or referenced as the existing tests do), the event
     sequence is pinned: the text deltas, each `EventItem`, then one
     `EventDone`, last; the concatenated text deltas equal the
     `MessageItem` texts of `EventDone`'s `Response`, and that `Response`
     equals `ReadStream`'s on the same golden.
   * `TestEvents_ReasoningDelta`: a scripted stream with each name Phase 1
     recorded yields `EventReasoningDelta` with its text. Omitted when
     Phase 1 recorded none.
   * `TestEvents_CallArgumentDeltasSilent`: `chatgpt-tool.sse`'s seven
     `response.function_call_arguments.delta` events yield nothing; the
     call arrives as one `EventItem`.
   * `TestEvents_StopEarly`: the decoder returns as soon as `yield`
     returns false, reading no further.
   * `TestEvents_FailureAfterDelta`: a delta, then `response.failed`:
     one delta, then the classified error, no `EventDone`.
3. **Change.** The decoder (a function of `internal/wire/responses`, so
   outside the module's API, taking the provider, the body and a `yield`)
   maps per MADR D2's table and keeps `ignored`'s fast skip for every type
   it does not map. `ReadStream` collects it, keeping its doc's
   guarantees.
4. Every existing `responses` test and the G-wire goldens pass unchanged.
5. **Benchmarks after:** no median of `ns/op`, `B/op` or `allocs/op` worse
   than 10% over its Phase 2 step 1 figure; worse is a deviation.
6. Gate; stage.

### Phase 3: the streaming seam and its failure rules (D3, D4)

1. **Tests, red first** (`internal/wire/stream_0031_test.go`), each with an
   `httptest.Server`:
   * `TestOpenStream_NonSuccessClassified`: a 429 with `Retry-After`, a
     403, a 500: the open step returns the same `*APIError` `Post` would.
   * `TestOpenStream_ReauthBeforeEvents`: a 401, then a stream: through
     `wire.Reauth`, the source is invalidated once and the second request
     streams.
   * `TestStream_IdleStall`, `TestStream_EventTooLarge`,
     `TestStream_EndsEarly`: each ends with the kind `ReadStream` gives for
     the same input today (`ErrProviderUnavailable` marked `AfterReply`,
     `ErrIncomplete`, `ErrProviderUnavailable` marked `AfterReply`).
   * `TestStream_StopClosesBody` and `TestStream_ContextCancel`: stopping
     the range, or cancelling `ctx` mid-stream, closes the body (the
     server's handler observes it) and leaves no goroutine (`-race`, and a
     goroutine count before and after).
2. **Change.** The open step builds and sends the request as `Post` does
   (`Prepare`, `Content-Type`, the D-H1 write trace, `Unsendable`),
   classifies a non-2xx, and returns a handle owning the body under the
   idle limit with no whole-body limit. The read step drives a decoder
   over it, yielding events, and closes the body on every exit. `Post` is
   unchanged.
3. **`stream.go` doc (D4):** a native stream's events before a failure
   are not a response; only `EventDone` carries one.
4. Gate; stage.

### Phase 4: the providers (D1, D5)

Only the providers Phase 1 showed streaming.

1. **Tests, red first**, in each provider's package, against scripted
   SSE replies:
   * `openai`: `TestStream_APIKey` (the request carries `"stream": true`
     and `Accept: text/event-stream` on `/responses`; the events are D2's),
     `TestStream_ChatGPT` (the ChatGPT headers, as `Generate` sends them);
     the capability test's `want` has `NativeStreaming: Supported`.
   * `grok`: `TestStream` and the capability `want`, likewise.
   * `opencode`: `TestStream_ResponsesRouteNative` (a delta arrives before
     the reply ends: the server holds the stream open after the first
     delta until the test reads it), `TestStream_OtherRoutesFallBack`
     (`messages`, `chat_completions`, `google`: one request without
     `"stream"`, events as `Stream`'s fallback emits them), and both ids'
     capability `want` at `BestEffort`.
   * `TestGenerate_Unchanged` in each: `Generate`'s request body has no
     `"stream"` key for an API key, and the G-wire goldens pass unchanged
     (D5).
2. **Change.** Each provider gains `Stream(ctx, req)`: `Capabilities.Check`
   first, then `wire.Reauth` over the open step, then the read step with
   D2's decoder. OpenCode's non-`responses` routes run `Generate` and emit
   its events through `llmprovider.Stream` over a value exposing only
   `Generate`, so the fallback's events are the shared ones. Capabilities
   and package docs change per D1; OpenCode's doc lists the route
   degradation (R12).
3. **Live tests** (`live_gateways`, under 0027-MADR's transient rule):
   `TestLive_OpenAIStreams`, `TestLive_GrokStreams`,
   `TestLive_OpencodeStreams` (a `responses`-route model on each gateway):
   one `Stream` of `"Say hi"` with `WithMaxTokens(16)`, wanting at least
   one `EventTextDelta` before `EventDone`. Run once here: at most 4
   billed requests, named when the phase is asked.

   *Per D1 (2026-10-10):* on `opencode-zen`, a 403 refusing the model is
   recorded as not run, not as a failure.
4. `llmtest.Run` still passes for every built-in provider (its R10 check
   now sees `Streamer` and the new values together).
5. Gate; stage.

### Phase 5: `WithRetry` (D6)

1. **Test, a guard of existing behaviour** (`retry_0031_test.go`):
   `TestWithRetry_StreamIsBuffered`: `WithRetry` over the `openai`
   provider (API key, scripted server: a 503, then a streamed success):
   the wrapper is not a `Streamer`, reports `NativeStreaming:
   Unsupported`, and `llmprovider.Stream` over it yields events from the
   second attempt only, with one `EventDone`. Seen failing on a scratch
   copy whose wrapper is planted with a pass-through `Stream`.
2. **Doc.** `WithRetry`'s comment says live deltas are not available
   through it, and why (MADR D6).
3. Gate; stage.

### Phase 6: the conformance check (D7)

*Added by D4 (2026-10-10), before step 1:* **the refusal delta.**
`deltaEvents` gains `response.refusal.delta` → `EventTextDelta`.
`TestEvents_RefusalDelta`, in `responses/events_0031_test.go` and red first:
a refusal delta, then a message item with a refusal part, then
`response.completed`. The events must be `text_delta:<text>`,
`item:MessageItem`, `done`. The joined deltas must equal the message text,
and `EventDone`'s `FinishReason` must be `content_filter`. Phase 2's two
benchmarks are rerun, within rule 5's 10% of Phase 2's figures.

1. **Harness fields** in `llmtest`: a streamed successful reply, and a
   streamed reply that fails after one delta. The check runs only when
   both are set and the provider implements `Streamer` (0021-MADR D5's
   "a Harness that sets none passes as before").
2. **The check:** MADR D7's five properties, against the scripted replies.
3. **Plants, on a scratch copy:** a `Streamer` emitting an item after
   `EventDone`; one with no `EventDone`; one whose text deltas differ from
   the final text; one that ignores `yield`'s false and keeps the body
   open. Each fails the check; each FAIL line is recorded.
4. The `openai`, `grok` and both `opencode` harnesses set the fields;
   `llmtest.Run` passes for each.
5. Gate (apidiff: the new `Harness` fields are additions); stage.

### Phase 7: documentation and release

1. **`docs/architecture.md`:** the `Stream` text (no longer "no built-in
   provider streams natively yet"), the capabilities text, the
   `internal/wire/responses` row (the decoder), the `internal/wire` row
   (the streaming seam).
2. **`docs/guides/api-standards.md`:** only if a rule's text no longer
   holds (none is expected; R13 already allows native streaming).
3. **Release notes** in this PLAN's Execution Record, as `v1.5.0`;
   `README.md`'s Status names `v1.5.0`.
4. **The live suites** on the release commit:
   `LLMPROVIDER_LIVE_TOGETHER=1 go test -tags live_gateways -count=1 -v
   ./llmprovider/... -run Live`; a failure is a deviation, a transient one
   rerun once alone.
5. **The owner** commits, pushes, and tags `v1.5.0`.
6. **After the tag:** CI on the tag; `go list -m …@v1.5.0` through the
   proxy; apidiff between `v1.4.0` and `v1.5.0`: additions only; a scratch
   consumer using every public package, streaming from one provider, and a
   scratch copy of prepare-commit-msg, built and tested against `v1.5.0`.
7. This PLAN `complete`; 0031-MADR's D8 left open for the later wires; the
   index updated.

## Verification

* **V1. Red, then green,** for every test the phases name, with both lines
  in the Execution Record; every plant's FAIL line.
* **V2. The gate** clean at the end of each phase; the goldens unchanged.
* **V3. Measurement:** T1 recorded before Phases 2 and 4 use it.
* **V4. Benchmarks:** Phase 2's before and after medians, within 10%.
* **V5. API:** `api-check` against `v1.4.0` shows additions only: the
  `llmtest.Harness` fields, and nothing else unless recorded.
* **V6. Live:** Phase 4's live tests, and the live suites on the release
  commit.
* **V7. Records:** 0031-MADR amended for any provider Phase 1 left
  `Unsupported`; `docs/architecture.md` and the package docs match the
  code.

## Rollout and Rollback

* **Rollout.** Seven commits after the records, one per phase; then
  `v1.5.0`. Consumers move under their own records.
* **Rollback, before the tag:** each phase reverts alone, newest first.
  Phase 4 depends on Phases 2 and 3; Phase 6 on Phase 4; Phase 5 on none.
* **Rollback, after the tag:** fix forward in `v1.5.x`. A consumer that
  meets a defect pins `v1.4.0`, or calls `Generate` or wraps the provider
  in `WithRetry`, both of which behave as at `v1.4.0`.

## Execution Record

### Phase 0: records (2026-10-10)

The owner committed 0031-MADR (`accepted`), this PLAN (`proposed`) and the
index (66 records) as `0a466f9`, and approved execution; this PLAN is
`in-progress`.

### Phase 1: measurement (2026-10-10)

`TestLive_0031StreamReplies` (`llmprovider/live_0031_measure_test.go`),
run at 20:14 UTC with `LLMPROVIDER_LIVE_0031=1`, `OPENAI_API_KEY`,
`XAI_API_KEY` and `OPENCODE_API_KEY` set (checked for presence only), and
`LLMPROVIDER_LIVE_CHATGPT` unset. A recording transport turns each
provider's own `POST {base}/responses` into a streamed one (`"stream":
true`, `Accept: text/event-stream`), so every other header is the
provider's. Exit 0. Billed: 8 requests of at most 16 output tokens; then
the two `opencode-zen` requests once more, with the redacted error logged
(both refused before generation).

| Target | Request | Model | Status | `Content-Type` | Event types, in order of first appearance |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `openai` | plain | `gpt-4.1-mini` | 200 | `text/event-stream; charset=utf-8` | `created`, `in_progress`, `output_item.added`, `content_part.added`, `output_text.delta`, `output_text.done`, `content_part.done`, `output_item.done`, `completed` |
| `openai` | effort `low` | `o4-mini` | 200 | `text/event-stream; charset=utf-8` | `created`, `in_progress`, `output_item.added`, `output_item.done`, `incomplete` |
| `grok` | plain | `grok-4.6` | 200 | `text/event-stream` | `created`, `in_progress`, `output_item.added`, `reasoning_summary_part.added`, `reasoning_summary_text.delta`, `reasoning_summary_text.done`, `reasoning_summary_part.done`, `output_item.done`, `content_part.added`, `output_text.delta`, `output_text.done`, `content_part.done`, `completed` |
| `grok` | effort `low` | `grok-3-mini` | 200 | `text/event-stream` | the same as `grok`'s plain request |
| `opencode-zen` | both | `gpt-5-nano` | 403 | `application/json` | none: "not permitted: opencode-zen/responses HTTP 403 server_error: Upstream request failed: Model access is disabled" |
| `opencode-go` | plain | `gpt-6-luna` | 200 | `text/event-stream` | `created`, `in_progress`, `output_item.added`, `output_item.done`, `incomplete` |
| `opencode-go` | effort `low` | `gpt-6-luna` | 200 | `text/event-stream` | `created`, `in_progress`, `output_item.added`, `content_part.added`, `output_text.delta`, `output_text.done`, `content_part.done`, `output_item.done`, `completed` |
| ChatGPT | — | — | — | — | skipped: `LLMPROVIDER_LIVE_CHATGPT` unset; not measured |

Every type is prefixed `response.` on the wire; the table drops the prefix.

**Outcome, by rule 2 as amended by D2:** `openai`, `grok` and
`opencode-go` stream on `/responses`; `opencode-zen` streams by D1. All
four take MADR D1's declarations.

**Outcome, by rule 3:** the one reasoning delta seen is
`response.reasoning_summary_text.delta` (Grok). It alone maps to
`EventReasoningDelta`. `response.reasoning_text.delta` was not seen and is
not mapped; `o4-mini`'s 16 tokens went to reasoning with no delta.

### Deviation D1 (2026-10-10): Zen's key is refused the chosen model

* **Found** in step 1. Step 1's rule chose `gpt-5-nano` for
  `opencode-zen` (no model in Zen's static catalog routes to
  `responses`, and it is the snapshot's first "nano" model). Both
  requests answered 403 "Model access is disabled": an entitlement of the
  key, which says nothing of streaming. By step 2 as written, Zen would
  have stayed `Unsupported`.
* **Options put to the owner:** try Zen's other `responses`-route models
  until one is permitted (recommended); record Zen as not measured and
  keep it `Unsupported`; or let Go's measurement stand for Zen, the same
  gateway and route.
* **Decision.** The owner chose the third. Zen takes `BestEffort` on Go's
  evidence; Phase 4's live test records a refused Zen model as not run.
  0031-MADR is amended (its "Measured, not assumed" driver has this one
  exception).

### Deviation D2 (2026-10-10): a plain request ended at the output cap

* **Found** in step 2. `opencode-go`'s plain request spent its 16 tokens
  reasoning at the default effort and ended `response.incomplete`; its
  effort-`low` request on the same model streamed text and ended
  `response.completed`. Step 2 as written required the plain request to
  complete, and would have kept Go `Unsupported`.
* **Options put to the owner:** count a 2xx event stream ending in
  `response.completed` or `response.incomplete`, from either request
  (recommended); or keep the rule and rerun the plain request with a
  256-token cap (one billed request).
* **Decision.** The owner chose the first; step 2 is amended in place. No
  decision of 0031-MADR changes.

### Phase 2: the decoder (2026-10-10)

**Change** (`llmprovider/internal/wire/responses/responses.go`):

* `Events(provider, body, yield) error` yields the events of MADR D2's
  table, with Phase 1's one reasoning delta.
* `ReadStream` collects it.
* Both run on one reader, `readEvents`, which decodes only the event types
  in the set it is given. `ReadStream` passes `handledEvents`, so it still
  skips the deltas undecoded (0021-MADR W2); `Events` passes
  `streamedEvents`, which is `handledEvents` plus `deltaEvents`.
* A delta is decoded into a one-field struct.
* `ignored` takes the set. The new `eventType` returns a slice of the
  payload, so a map lookup by it does not allocate.

**Red** (step 2), `events_0031_test.go` on the tree before the change:

```text
llmprovider/internal/wire/responses/events_0031_test.go:19:9: undefined: Events
llmprovider/internal/wire/responses/events_0031_test.go:147:9: undefined: Events
FAIL	github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/responses [build failed]
```

**Green:** the five tests below pass after the change, and so does
`go test -count=1 ./llmprovider/...`: every existing `responses` test, and
the G-wire goldens (step 4).

* `TestEvents_Goldens`: `chatgpt-text.sse` gives `text_delta:ok`,
  `item:MessageItem`, `done`; `chatgpt-reasoning.sse` gives
  `text_delta:391`, `item:MessageItem`, `done`; `chatgpt-tool.sse` gives
  `item:FunctionCallItem`, `done`. In each, `EventDone`'s `Response` equals
  `ReadStream`'s, the `EventItem`s equal its `Output`, and the text deltas
  joined equal its message text.
* `TestEvents_ReasoningDelta`
* `TestEvents_CallArgumentDeltasSilent`
* `TestEvents_StopEarly`
* `TestEvents_FailureAfterDelta`

`TestIgnored_EventTypes` gains a loop: no type in `deltaEvents` is ignored
under `streamedEvents`.

**Plants** (rule 2), on a scratch clone, for the two guards a compile
failure does not exercise:

| Plant | Test | Result |
| :--- | :--- | :--- |
| `streamedEvents` without the deltas | `TestIgnored_EventTypes` | `ignored_test.go:39: Events ignores response.output_text.delta` (and `…reasoning_summary_text.delta`); FAIL |
| a delta that ignores `yield`'s false | `TestEvents_StopEarly` | `events_0031_test.go:152: stopped at once: 10001 calls, <nil>; want 1 call and nil`; FAIL |

The first plant was first run against a loop over `streamedEvents`. It
passed, because removing the deltas from that map also removed the cases
that should have caught it. The loop was changed to range over
`deltaEvents`, and the plant then failed as shown.

**Benchmarks** (rule 5; `-benchmem -count=5`, same host, medians):

| Benchmark | Before | After | Change |
| :--- | :--- | :--- | :--- |
| `ReadStream_Deltas4MiB` ns/op | 1,416,623 | 1,374,522 | −3.0% |
| `ReadStream_Deltas4MiB` B/op | 66,874 | 66,879 | 0.0% |
| `ReadStream_Deltas4MiB` allocs/op | 13 | 13 | 0 |
| `ReadStream_4MiB` ns/op | 7,713,890 | 7,715,058 | 0.0% |
| `ReadStream_4MiB` B/op | 860,482 | 860,554 | 0.0% |
| `ReadStream_4MiB` allocs/op | 7,120 | 7,120 | 0 |

Every figure is within step 5's 10%.

**Gate:** all 22 checks pass on a scratch copy with the phase's files staged.
`internal/wire/responses` coverage is 98.1%.

**Found, for Phase 6 (not a deviation of this phase):** MADR D7 asks that
the text deltas, joined, equal the `MessageItem` texts. `appendOutput`
keeps a refusal's text as the message text (0021-MADR W4), but D2's table
maps no refusal delta. OpenAI documents a `response.refusal.delta` event;
it was not measured here. On a refusal, D7's check would therefore fail.
The owner settles this before Phase 6.

### Deviation D3 (2026-10-10): `bodyclose` cannot follow the open reply

* **Found** at Phase 3's gate. `make lint` reported
  `llmprovider/internal/wire/post.go:136:26: response body must be closed
  (bodyclose)` on `resp, err := c.Client.Do(req)`. The finding is new: at
  `v1.4.0`, `Post` closed the body with a `defer` in the same function. In
  D3's two-step design, `OpenStream` returns the body open, owned by
  `*Stream`, and `Read` and `Close` close it.
* **Two code changes were tried; neither satisfied the analyzer:**
  * keeping the `*http.Response` in the `Stream` struct;
  * building the `Stream` in a constructor that takes the response.
* **Evidence that the body is closed:**
  * `TestStream_StopClosesBody` passes;
  * a plant that drops `Read`'s close fails it (below);
  * a non-2xx reply is closed inside `open`.
* **Options put to the owner:**
  * a line `//nolint:bodyclose` with its reason, in the repository's
    existing nolint-with-reason style (recommended);
  * a callback API that closes with `defer`, under which `Reauth` would wrap
    the whole stream and need a guard against resending after events (a
    change to MADR D3);
  * excluding `bodyclose` for the file in `.golangci.yml`.
* **Decision.** The owner chose the first. The constructor from the
  experiment was removed, so the open step reads as `Post`'s did. No
  decision of 0031-MADR changes.

### Phase 3: the streaming seam and its failure rules (2026-10-10)

**Change** (`llmprovider/internal/wire/post.go`):

* `Post` and the new `OpenStream` share one open step, `open(ctx, c,
  token, limit)`. It holds what `Post` did:
  * marshalling;
  * the D-H1 write trace;
  * `Prepare`;
  * `Unsendable`;
  * `AfterReply`;
  * the reply under the idle limit;
  * `ClassifyHTTPError`, after which a non-2xx is closed.
* It returns a `*Stream`.
* `Post` reads that reply with `decode` and `DecodeError`, as before, under
  `ReplyLimit` unless `c.Stream`.
* `OpenStream` returns it open, with no whole-body limit.
* `(*Stream).Read(decode)` runs the decoder, gives a read failure its kind
  through `DecodeError`, and closes the reply on every exit.
* `(*Stream).Close()` stops the idle limit, closes the body and cancels the
  request. It is safe to call twice.
* The `Do` line carries D3's `//nolint:bodyclose`.

`llmprovider/stream.go`: `Stream`'s doc now says that only `EventDone`
carries a `Response`, and that what a native stream yielded before a failure
is not a response (MADR D4).

**Red** (step 1), `stream_0031_test.go` (package `wire_test`, which also
imports `responses`) on the tree before the change:

```text
llmprovider/internal/wire/stream_0031_test.go:49:17: undefined: wire.OpenStream
llmprovider/internal/wire/stream_0031_test.go:80:22: undefined: wire.OpenStream
llmprovider/internal/wire/stream_0031_test.go:120:93: undefined: wire.Stream
llmprovider/internal/wire/stream_0031_test.go:121:15: undefined: wire.OpenStream
llmprovider/internal/wire/stream_0031_test.go:223:17: undefined: wire.OpenStream
llmprovider/internal/wire/stream_0031_test.go:255:17: undefined: wire.OpenStream
FAIL	github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire [build failed]
```

**Green, under `-race`:**

* `TestOpenStream_NonSuccessClassified`: for a 429 with `Retry-After`, a
  403 and a 500, `OpenStream`'s `*APIError` has the same status, kind and
  `RetryAfter` as `Post`'s.
* `TestOpenStream_ReauthBeforeEvents`: a 401, then a stream; one
  invalidation, two requests, then three events.
* `TestStream_Failures`:
  * an idle stall is `ErrProviderUnavailable`, after the reply;
  * an event over 16 MiB is `ErrIncomplete`;
  * an early end is `ErrProviderUnavailable`, after the reply.

  In each case the error matches `Post` with `ReadStream` on the same
  reply, and no `EventDone` is yielded.
* `TestStream_StopClosesBody` and `TestStream_ContextCancel`: the server
  sees the client go, and the goroutine count falls back. The cancelled
  stream's error matches `context.Canceled`.

The new tests passed 10 runs in a row under `-race`. `go test
./llmprovider/...` passes.

**Corrections to the new tests while they were being made green** (neither
loosens an assertion):

* The held and stalling handlers now drain the request body first. A Go
  server notices a client hang up only once it has read the request body.
  Without the drain, `StopClosesBody` and `ContextCancel` failed ("the
  server never saw the client go") whatever the client did.
* The 50 ms idle limit in `TestStream_Failures` applies only to the
  idle-stall case. Set for every case, it made the 16 MiB case flaky under
  `pre-add-check`'s load: a write pause over 50 ms tripped the idle limit,
  so `Post`'s error was "read reply: no data …" rather than `ErrIncomplete`.
* The kind comparison carries the repository's `//nolint:errorlint // the
  kind itself, not what it wraps`.

**Plants** (rule 2), on a scratch clone, run against the final code:

| Plant | Test | Result |
| :--- | :--- | :--- |
| `Read` leaves the reply open (no `defer s.Close()`) | `TestStream_StopClosesBody` | `stream_0031_test.go:252: the server never saw the client go: the body was left open`; FAIL |
| `open` classifies a non-2xx only under a size limit | `TestOpenStream_NonSuccessClassified` | `stream_0031_test.go:87: HTTP 429: OpenStream <nil>, …; want the same *APIError` (also 403 and 500); FAIL |

**Gate:** all 22 checks pass on a scratch copy with the phase's files staged.
`internal/wire` coverage is 93.9%, and `llmprovider` 98.2%.

### Phase 4: the providers (2026-10-10)

Phase 1 showed all four providers streaming, so all four take MADR D1's
declarations.

**Change.**

* `internal/wire/responses`: `Stream(ctx, c, src)` is the shared native
  stream. It opens `c` through `wire.Reauth` and `wire.OpenStream`, reads
  it with `Events`, and yields each event. A failure, before or during the
  stream, is yielded once with a zero `Event`; a caller that stops ranging
  closes the reply.
* `openai` and `grok`:
  * The request is built by `call(req, stream)`, which `generateOnce` and
    the new `Stream` share.
  * `stream` adds `"stream": true` and `Accept: text/event-stream`. A
    ChatGPT session's request carries both already, as before.
  * `Stream` runs `Capabilities.Check`, then `responses.Stream`.
  * `NativeStreaming` is `Supported`, and the package docs say so.
* `opencode`:
  * `generateOnce` is split into `resolve` (model, input, limits,
    reasoning, route) and `wireCall` (the `wire.Call` for a route).
  * `Stream` resolves the route. On `responses` it sends the responses body
    with `"stream": true` through `responses.Stream`. On the other three
    routes it returns `llmprovider.Stream` over `generateOnly`, which is the
    provider without its `Stream`, so the fallback's events are the shared
    ones.
  * `NativeStreaming` is `BestEffort`, and the package doc lists the route
    degradation (R12).
* `Generate`'s request and decoding are unchanged on all three (D5).

**Red** (step 1), on the tree before the change. The updated capability
expectations and the new tests:

```text
--- FAIL: TestOpenAI_Capabilities: openai_test.go:142: Capabilities() = {… NativeStreaming:Unsupported}, want {… NativeStreaming:Supported}
--- FAIL: TestStream_APIKey: stream_0031_test.go:71: the provider does not stream natively
--- FAIL: TestStream_ChatGPT: stream_0031_test.go:93: the provider does not stream natively
--- FAIL: TestGrok_IDAndCapabilities: grok_test.go:365: Capabilities = {… NativeStreaming:Unsupported}; grok_test.go:368: the provider does not implement Streamer (0031-MADR D1)
--- FAIL: TestStream (grok): stream_0031_test.go:63: the provider does not stream natively
--- FAIL: TestOpencode_Capabilities: identification_test.go:168: opencode-zen: Capabilities = {… NativeStreaming:Unsupported}; identification_test.go:173: … does not implement Streamer (and opencode-go)
--- FAIL: TestStream_ResponsesRouteNative, TestStream_OtherRoutesFallBack: the provider does not implement Streamer
```

Each package's `TestGenerate_Unchanged` passed on the tree, as a guard of
existing behaviour should, and is proven by a plant (below).

**Green, under `-race`:**

* `openai`:
  * `TestStream_APIKey`: `/responses` with `"stream": true` and `Accept:
    text/event-stream`; events `text_delta:ok`, `item:MessageItem`,
    `done`.
  * `TestStream_ChatGPT`: the same events, with the headers and body
    `Generate` sends on the same session.
  * `TestGenerate_Unchanged`.
* `grok`:
  * `TestStream`: Phase 1's event shape gives `reasoning_delta:think`,
    `item`, `text_delta:ok`, `item`, `done`.
  * `TestGenerate_Unchanged`.
* `opencode`, on both gateways:
  * `TestStream_ResponsesRouteNative`: the server holds the stream after
    its first delta until the test has read it, so the delta reached the
    caller while the reply was open.
  * `TestStream_OtherRoutesFallBack`: on each of `messages`,
    `chat_completions` and `google`, one request without `"stream"`, and
    the fallback's `text_delta:ok`, `item`, `done`.
  * `TestGenerate_Unchanged`.
* `go test -count=1 ./...` passes. Every built-in provider's `llmtest.Run`
  passes (step 4), its R10 check now seeing `Streamer` and the new values
  together.

**Plants** (rule 2), on a scratch clone:

| Plant | Test | Result |
| :--- | :--- | :--- |
| `openai`'s `Generate` sends `"stream": false` | `TestGenerate_Unchanged` | `stream_0031_test.go:115: Generate sent stream false, Accept ""; want neither`; FAIL |
| `grok`'s `Generate` sends `"stream": false` | `TestGenerate_Unchanged` | `stream_0031_test.go:96: Generate sent stream false, Accept ""; want neither`; FAIL |
| `opencode`'s responses-route `Generate` sends `"stream": false` | `TestGenerate_Unchanged` | `stream_0031_test.go:138: Generate sent stream false; want none`; FAIL |
| `opencode`'s responses route falls back | `TestStream_ResponsesRouteNative` | `stream_0031_test.go:69: after []: llmprovider: incomplete response: opencode-zen/responses: invalid character 'd' looking for beginning of value`, after the held server's 5 s; FAIL |

The fourth plant first failed to build, because a quote in the plant
script's replacement text was unescaped. It was fixed and rerun, with the
result above.

**Live** (step 3), `llmprovider/live_stream_0031_test.go`, at 21:04 UTC with
the three keys set (checked for presence only). Billed: 3 requests of at
most 16 output tokens. Zen's request was refused before generation.

| Test | Model | Result |
| :--- | :--- | :--- |
| `TestLive_OpenAIStreams` | `gpt-4.1-mini` | PASS: 9 text deltas, then `EventDone` |
| `TestLive_GrokStreams` | `grok-4.6` | PASS: 9 text deltas, then `EventDone` |
| `TestLive_OpencodeStreams/opencode-go` | `gpt-6-luna`, effort `low` | PASS: 2 text deltas, then `EventDone` |
| `TestLive_OpencodeStreams/opencode-zen` | `gpt-5-nano`, effort `low` | SKIP, by D1 and 0027-MADR's transient rule: "not permitted: opencode-zen/responses HTTP 403 server_error: Upstream request failed: Model access is disabled"; not run |

The OpenCode test asks for effort `low`, because at the default effort a
16-token answer can go wholly to reasoning (D2's evidence).

**Gate:** all 22 checks pass on a scratch copy with the phase's files staged,
the G-wire goldens unchanged among them. Coverage:

| Package | Coverage |
| :--- | :--- |
| `openai` | 93.3% |
| `grok` | 95.7% |
| `opencode` | 97.4% |
| `internal/wire/responses` | 90.7% |

`internal/wire/responses` was 98.1% after Phase 2. `responses.Stream` is
covered by the provider tests, not by a test in its own package. The floors
hold.

### Deviation D4 (2026-10-10): a refusal's text has no delta

* **Found** in Phase 2 and recorded there. `appendOutput` keeps a refusal's
  text as the message text (0021-MADR W4), but MADR D2's table maps no
  refusal delta. On a refusal, D7's check (text deltas joined equal the
  message text) would fail, and native streams would differ from `Stream`'s
  fallback, which emits the refusal text as `EventTextDelta`.
* **Options put to the owner:**
  * map OpenAI's documented `response.refusal.delta` to `EventTextDelta`
    (recommended);
  * narrow D7's check to answers whose `FinishReason` is not
    `content_filter`;
  * try to elicit a refusal live first (billed, and uncertain to trigger).
* **Decision.** The owner chose the first. It is done at the start of
  Phase 6 (above), which gains `responses.go` in its scope. 0031-MADR is
  amended: a row added to D2's table, with an unmeasured event name as the
  second exception to its "Measured, not assumed" driver.

### Phase 5: `WithRetry` (2026-10-10)

**Test**, a guard of existing behaviour: `TestWithRetry_StreamIsBuffered`
(`llmprovider/retry_0031_test.go`, package `llmprovider_test`).

* It runs `WithRetry` over the `openai` provider with an API key, under
  `RetryPolicy{MaxAttempts: 3, BaseDelay: 1 ms}`. It first checks that the
  inner provider is a `Streamer`; otherwise the test would prove nothing.
* The server answers in whatever format the request asks for (an event
  stream when `Accept` asks for one, else JSON). It sends the first request
  the start of an answer ("par") with a longer `Content-Length`, then cuts
  the connection. Every later request gets the whole answer, "ok".
* The wrapper must not be a `Streamer`, and must report `NativeStreaming:
  Unsupported`.
* `llmprovider.Stream` over it must yield `text_delta:ok`, `item`, `done`,
  with no error and over 2 requests: the cut reply, then its one retry
  (0021-MADR D1).

PLAN step 1 named "a 503, then a streamed success". A reply cut part way
was used instead: it is the case that tells a buffered stream from a live
one, since a 503 yields nothing either way.

It passes on the tree, and under `-race`.

**Plant** (rule 2), on a scratch clone: a `retry_plant.go` giving the wrapper
a pass-through `Stream(ctx, req)` that returns `Stream(ctx, r.inner, req)`.

```text
--- FAIL: TestWithRetry_StreamIsBuffered (0.00s)
    retry_0031_test.go:84: WithRetry's result is a Streamer
    retry_0031_test.go:93: after [text_delta:par]: llmprovider: provider unavailable: openai: read reply: unexpected EOF
```

**Doc** (step 2), `llmprovider/retry.go`: `WithRetry`'s comment says the
result does not stream natively even when `p` does. Its stream is buffered
so it can be retried whole, and the caller receives events only from the
attempt that succeeds. A caller that wants text as it is generated streams
from `p` itself (MADR D6).

**Gate:** all 22 checks pass on a scratch copy with the phase's files staged.

### Deviation D5 (2026-10-10): `llmtest` checks the bodies, not the goroutines

* **Found** while building step 2. MADR D7's fifth property names "no
  goroutine left behind under `-race`". `llmtest.Run` runs inside other
  modules' test binaries, possibly beside parallel tests. A process-wide
  goroutine count there can rise or fall for reasons that have nothing to
  do with the provider.
* **Options put to the owner:**
  * the check asserts that every reply body is closed, and the goroutine
    count stays with `internal/wire`'s tests (recommended);
  * add a `runtime.NumGoroutine` comparison with a 2 s settle as well,
    which may fail or pass at random.
* **Decision.** The owner chose the first. 0031-MADR is amended. No code
  changes.

### Phase 6: the conformance check (2026-10-10)

**D4, the refusal delta** (the step D4 added).

* **Red:**

  ```text
  --- FAIL: TestEvents_RefusalDelta (0.00s)
      events_0031_test.go:185: events [item:MessageItem done] (<nil>), want [text_delta:I can't help with that. item:MessageItem done]
  ```

* **Change:** `deltaEvents` gains `response.refusal.delta` →
  `EventTextDelta`, with its reason and source in the comment.
* **Green:** the test passes, and so does the `responses` package.
* **Benchmarks** (rule 5; medians, `-count=5`), against Phase 2's
  after-figures:

  | Benchmark | Phase 2 after | Now | Change |
  | :--- | :--- | :--- | :--- |
  | `ReadStream_Deltas4MiB` ns/op | 1,374,522 | 1,349,359 | −1.8% |
  | `ReadStream_Deltas4MiB` B/op | 66,879 | 66,867 | 0.0% |
  | `ReadStream_Deltas4MiB` allocs/op | 13 | 13 | 0 |
  | `ReadStream_4MiB` ns/op | 7,715,058 | 7,380,447 | −4.3% |
  | `ReadStream_4MiB` B/op | 860,554 | 860,501 | 0.0% |
  | `ReadStream_4MiB` allocs/op | 7,120 | 7,120 | 0 |

**Steps 1, 2 and 4, the check** (`llmprovider/llmtest/llmtest.go`).

* `Harness` gains `StreamText` and `StreamFailure`.
* `Run` adds `D7-native-stream` when both are set. The check returns
  without effect for a provider that is not a `Streamer`.
* It runs `Generate` on the `Text` reply, then streams through an HTTP
  client that counts the reply bodies opened and closed (its own
  `http.Transport`; see the note on R30 below). It asserts:
  * the `StreamText` stream yields no error, exactly one `EventDone` with a
    `Response`, and nothing after it;
  * the text deltas are non-empty, and joined equal `EventDone`'s message
    text;
  * `EventDone`'s `Response` equals `Generate`'s;
  * every body is closed after the completed stream, and after a caller
    that stopped after its first event;
  * the `StreamFailure` stream yields exactly one error, last, no
    `EventDone`, and its body is closed (D5 above).
* `Run`'s doc lists the check.

**Harnesses:**

* `openai` (the API key and the ChatGPT session), `grok` (the API key and
  the session) and `opencode`'s `go-responses` case set `llmtestStream`
  (a text delta, the item, completed) and `llmtestStreamFailure` (a delta,
  then `response.failed`).
* `opencode`'s other three routes set neither: they do not stream natively.

**Red** (rule 2): with the harness fields set before the check existed:

```text
llmprovider/providers/openai/llmtest_test.go:55:3: unknown field StreamText in struct literal of type llmtest.Harness
llmprovider/providers/openai/llmtest_test.go:56:3: unknown field StreamFailure in struct literal of type llmtest.Harness
llmprovider/providers/grok/llmtest_test.go:51:5: unknown field StreamText in struct literal of type llmtest.Harness
llmprovider/providers/grok/llmtest_test.go:52:5: unknown field StreamFailure in struct literal of type llmtest.Harness
FAIL	…/providers/openai [build failed]; FAIL	…/providers/grok [build failed]; FAIL	…/providers/opencode [build failed]
```

**Green, under `-race`:**

* `TestConformance/api-key/D7-native-stream`
* `TestConformance/chatgpt/D7-native-stream` (`openai`)
* `TestConformance/api_key/D7-native-stream`
* `TestConformance/session/D7-native-stream` (`grok`)
* `TestConformance/go-responses/D7-native-stream` (`opencode`)

**Plants** (step 3), on a scratch clone, each run against
`TestConformance/api-key/D7-native-stream` in `openai`:

| Plant | The check's report |
| :--- | :--- |
| an item yielded after `EventDone` (in `Events`) | `D7 (native streaming, 0031-MADR D4): event 3 (item) after EventDone` |
| no `EventDone` | `D7 (…D4): the stream ended without EventDone; events [text_delta item]` |
| text deltas that differ from the final text (`+ "x"`) | `D7 (…D2): the text deltas joined are "hellox"; EventDone's message text is "hello"` |
| a reply left open (`Read` without its `Close`) | `D7 (…D3): a completed stream left its reply open`; `…a caller that stopped after the first event got []; reply left open: true`; `…a failed stream left its reply open` |

Each FAILed.

**Fixed at the gate, without a deviation:**

* The first gate run failed `TestNoAmbientState` (R30-R31):
  `llmtest.go:1021: http.DefaultTransport is shared by the process`. The new
  tracker had sent through `http.DefaultTransport`. It now owns its
  `http.Transport` and closes its idle connections. That is the new code
  meeting an existing rule; no rule was changed.
* The plants were rerun after the fix, with the same four results.

**API:** apidiff from `v1.4.0` (both trees, without `-incompatible`) reports
only additions:

```text
Compatible changes:
- ./llmprovider/llmtest.Harness.StreamFailure: added
- ./llmprovider/llmtest.Harness.StreamText: added
```

**Gate:** all 22 checks pass on a scratch copy with the phase's files staged.

| Package | Coverage | Note |
| :--- | :--- | :--- |
| `llmtest` | 81.3% | `checkNativeStream` is exercised by the provider packages' conformance runs, not by `llmtest`'s own tests; there is no floor for it |
| `internal/wire/responses` | 90.7% | |

### Phase 7: documentation and release (2026-10-10)

**Steps 1 to 3**, on `3e0882e` (Phase 6):

* **`docs/architecture.md`:**
  * the `openai`, `grok` and `opencode` rows name `Stream`;
  * the `internal/wire` row names `OpenStream` and `Stream`;
  * the `internal/wire/responses` row names `Events` and `Stream`;
  * "A generation" says which providers stream natively, and that
    `WithRetry` does not;
  * the contract section gives the declarations, the event mapping and
    the failure rule;
  * the `WithRetry` and `llmtest` paragraphs cover D6 and D7;
  * "What is not here" names the four wires still on the fallback (D8).
* **`docs/guides/api-standards.md`:** unchanged. R13 already says native
  streaming is added without an API change, and no rule's text stopped
  holding.
* **`README.md`:** Status names `v1.5.0`, with a `v1.5` bullet.

#### Release notes: `v1.5.0`

`v1.5.0` streams natively on the OpenAI Responses wire
([0031-MADR](0031-MADR-native-streaming-on-the-responses-wire.md)). The only
API change is an addition, in `llmtest`.

**Native streaming.**

* `llmprovider.Stream` on `openai` (an API key or a ChatGPT session) and on
  `grok` asks `/responses` for an event stream and yields each event as it
  arrives.
* Both OpenCode gateways do the same on the `responses` route. On
  `messages`, `chat_completions` and `google` they still yield `Generate`'s
  result as events.
* `NativeStreaming` is `Supported` for `openai` and `grok`, and
  `BestEffort` for `opencode-zen` and `opencode-go`. Every other provider
  stays `Unsupported`, and `Stream` there is unchanged.
* `opencode-zen`'s streaming was not measured on its own: the key used was
  refused the model tried. It rests on `opencode-go`'s measurement, the
  same gateway and route (D1).

**What a native stream yields.**

* Text deltas, and a refusal's deltas, as `EventTextDelta`.
* The reasoning summary delta as `EventReasoningDelta`, the one reasoning
  delta measured (from xAI; OpenAI's 16-token measurement sent none).
* Each finished output item as `EventItem`. A function call arrives only as
  its item; argument deltas are not yielded.
* `response.completed` as `EventDone`, carrying the same `Response`
  `Generate` would return.

**Failures.**

* A failure is yielded once and ends the stream, with no `EventDone`.
* Deltas and items already yielded are not a response: show them as
  incomplete, or discard them.
* A 401 before the first event renews a refreshable credential and sends
  once more, as `Generate` does.
* Stopping the range closes the reply.

**`WithRetry`** is unchanged: it does not stream natively. `Stream` over it
runs its `Generate`, so a failure at any point is retried and the caller
receives events only from the attempt that succeeds. For text as it arrives,
stream from the provider itself and handle a failure there.

**`Generate`** sends and decodes exactly as in `v1.4.0`.

**`llmtest`:**

* `Harness.StreamText` and `Harness.StreamFailure` switch on a native
  stream check for any `Streamer`: one `EventDone`, last, equal to
  `Generate`'s `Response`, with the deltas holding its text; a failure with
  no `EventDone`; every reply body closed.
* A harness that sets neither passes as before.

**Upgrading.**

* No code change is needed.
* Code that reads `NativeStreaming` sees new values for four ids.
* Code that calls `Stream` on those ids now receives text while it is
  generated, and should treat events before a failure as partial.
* `Stream` on those ids sends a streamed request where it used to send
  `Generate`'s. The output and the billing are the same.

**Not in this release:** native streaming for Chat Completions, Messages,
Gemini Interactions and generateContent (MADR D8).
