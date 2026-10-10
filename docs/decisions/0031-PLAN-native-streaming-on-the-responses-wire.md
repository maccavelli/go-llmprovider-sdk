---
status: proposed
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
| 6 | D7 | `llmprovider/llmtest/llmtest.go`, the three providers' harnesses |
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
2. **Rule (D1).** A target whose plain request answers 2xx with an event
   stream ending in `response.completed` streams. Any other outcome keeps
   that provider `Unsupported`, as a deviation and a MADR amendment.
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

Nothing executed yet.
