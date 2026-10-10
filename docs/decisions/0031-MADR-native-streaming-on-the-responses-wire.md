---
status: accepted
date: 2026-10-10
decision-makers: repository owner
consulted: 0030-REPORT-v1-api-and-protocol-surfaces.md (F4, F8), 0015-MADR-canonical-sdk-api-and-module-layout.md (D4), 0020-MADR-remediate-v1-debugging-pass-findings.md (F32), 0021-MADR-harden-and-tune-after-the-v1-1-review.md (D1, W2, W14)
informed: consumers of go-llmprovider-sdk v1
---

<!-- markdownlint-disable MD013 -->

# Stream Natively on the OpenAI Responses Wire

## Context and Problem Statement

[0015-MADR](0015-MADR-canonical-sdk-api-and-module-layout.md) D4 put
streaming in the contract from `v1.0.0`: `llmprovider.Stream(ctx, p, req)`
returns `iter.Seq2[Event, error]` for every provider, uses a provider's
`Streamer` when it has one, and otherwise runs `Generate` and emits the
result as events. Native streaming was to be "added provider by provider
later, with no API change". At `v1.4.0` no provider has added it.
[0030-REPORT](../reports/0030-REPORT-v1-api-and-protocol-surfaces.md) F8
names this the largest gap between the contract and the implementation.

What the tree has today (read at `73ac09a`):

* **Every built-in provider declares `NativeStreaming: Unsupported`** and
  says so in its package comment (`openai.go:13`, `grok.go:14`,
  `opencode.go:27`). A caller of `Stream` waits for the whole generation,
  then receives it at once.
* **The ChatGPT path already consumes a Responses event stream.** It sends
  `"stream": true` (`openai.go:221`) and `Accept: text/event-stream`
  (`openai.go:293`), and `responses.ReadStream` (`responses.go:288`) reads
  the stream into one `Response`. The reader is bounded per event, not per
  stream (16 MiB, `eventLimit`;
  [0021-MADR](0021-MADR-harden-and-tune-after-the-v1-1-review.md) W2), and
  a stream that ends before `response.completed` is retryable once
  (0021-MADR D1).
* **`ReadStream` skips the deltas.** `ignored` drops every `response.*`
  event outside `handledEvents` (`responses.go:409-412`) without decoding
  it, because the `output_item.done` events carry the whole items. The three
  ChatGPT goldens (`providers/openai/testdata/*.sse`) contain
  `response.output_text.delta` (2) and
  `response.function_call_arguments.delta` (7) among the events skipped.
* **Three providers speak the Responses wire:** `openai` (an API key on
  `/v1/responses`, decoded whole by `responses.DecodeFor`, and a ChatGPT
  session on the Codex backend, streamed), `grok` (`DecodeFor`,
  `grok.go:242`), and the OpenCode gateways' `responses` route (`DecodeFor`,
  `opencode.go:239`).
* **`wire.Post` decodes only after the reply is in hand.** It is generic
  over a `decode func(io.Reader) (T, error)` that returns one value
  (`post.go`); `Call.Stream` lifts the whole-body limit but the caller still
  receives nothing until `decode` returns.
* **`WithRetry` hides streaming.** Its wrapper has no `Stream` method, so it
  reports `NativeStreaming: Unsupported` (`retry.go:88-93`), as
  [0020-MADR](0020-MADR-remediate-v1-debugging-pass-findings.md) F32's
  remediation decided: claiming the capability without the method was the
  defect.
* **The conformance suite pins the declaration to the method.**
  `llmtest.Run` fails a provider whose `NativeStreaming` is not
  `Unsupported` exactly when it implements `Streamer`
  (`llmtest.go:277-278`). It has no check of a `Streamer`'s events.

The question: how should native streaming enter the tree — which wires
first, through what internal seam, with what failure and retry semantics —
without changing the public generation API?

## Decision Drivers

* **No public API change for providers** (0015-MADR D4;
  [api-standards.md](../guides/api-standards.md) R5, R13). `Event`,
  `EventType` and `Streamer` already exist (`stream.go`).
* **Reuse what is proven.** The Responses stream reader has goldens, a
  per-event bound, an idle limit and a once-only retry for an early end;
  the other four wires have no stream reader at all.
* **A streamed failure must not be mistaken for a result.** A failed
  response never hands back its partial output (R7; 0021-MADR W14). Text
  already yielded cannot be taken back.
* **A retry must never duplicate what the caller already received,** and
  must not bill a generation more than 0021-MADR D1 and
  [0028-MADR](0028-MADR-heuristics-and-performance-from-the-research-pass.md)
  D-H1 allow.
* **Capabilities stay data** (R10–R12): `NativeStreaming` is declared only
  where `Streamer` is implemented, and every `BestEffort` degradation is in
  the package doc.
* **Measured, not assumed.** Whether xAI and the OpenCode gateway honour
  `"stream": true` on `/responses` has not been measured in this
  repository.

## Considered Options

* **Responses wire first:** one incremental decoder behind `openai`,
  `grok` and OpenCode's `responses` route; `WithRetry` buffers, so that it
  can retry the whole generation; the other wires are later phases.
* **All five wires at once:** Responses, Chat Completions, Messages, Gemini
  Interactions and generateContent in one change.
* **ChatGPT sessions only:** expose the stream the ChatGPT path already
  reads, and leave the API-key, Grok and OpenCode paths unstreamed.
* **Keep the fallback:** no native streaming; document `Stream` as a
  convenience over `Generate`.

## Decision Outcome

Chosen option: "Responses wire first", because it closes the gap for the
three providers that share the one wire with a tested stream reader, costs
no public API change, and fixes the contract (decoder, transport seam,
failure and retry rules, conformance check) once, so that each later wire
is a reader and a phase rather than a new decision.

### D1. Which providers stream

* `openai` implements `Streamer` for an API key and for a ChatGPT session,
  and declares `NativeStreaming: Supported`.
* `grok` implements `Streamer` and declares `Supported`.
* `opencode-zen` and `opencode-go` implement `Streamer` and declare
  `BestEffort`: a request routed to `responses` streams natively; one
  routed to `messages`, `chat_completions` or `google` falls back to
  `Generate`'s result as events until that wire's phase lands. The package
  doc lists it as a degradation (R12).
* **Gated on measurement.** The PLAN first sends a streamed request to each
  of xAI and both OpenCode gateways' `responses` route and records the
  status, `Content-Type` and event types. A service that does not stream on
  `/responses` keeps `Unsupported`, and this MADR is amended with the
  evidence.

### D2. One incremental decoder, and `ReadStream` collects it

* `internal/wire/responses` gains an internal event decoder that reads a
  Responses event stream and yields `llmprovider.Event`s as they arrive.
  `ReadStream` becomes a collector over it, with identical results on the
  three goldens and every existing `ReadStream` test.
* Mapping:

  | Responses event | `Event` |
  | :--- | :--- |
  | `response.output_text.delta` | `EventTextDelta`, the delta's text |
  | the reasoning text and reasoning summary text deltas | `EventReasoningDelta` |
  | `response.output_item.done` | `EventItem`, the item as `appendOutput` builds it |
  | `response.completed` | `EventDone`, with the `Response` `ReadStream` would return |
  | `response.failed`, `response.incomplete`, `error` | the error, classified as today |
  | `response.function_call_arguments.delta`, and every other event | none |

* A function call arrives only as an `EventItem`, as it does in `Stream`'s
  fallback (`stream.go`). A call-argument delta would need a new
  `EventType`; that is out of scope (Q4).
* The skip of unused events stays cheap: the decoder decodes only the event
  types it maps, as `ignored` does now (0021-MADR W2's measurement is the
  baseline).
* The reasoning delta event names are taken from the measured streams of
  D1, not assumed; a name not seen there is not mapped.

### D3. A streaming seam in `internal/wire`

* `internal/wire` gains a streaming counterpart of `Post` that keeps the
  reply open while the decoder yields. It shares `Post`'s request building,
  `Prepare`, `ClassifyHTTPError` of a non-2xx, the idle limit, the
  `AfterReply` marking of 0028-MADR D-H1, and the per-event bound in place
  of `ReplyLimit`.
* A 401 renews the credential and resends through `wire.Reauth` as today:
  a 401 arrives before any event, so nothing has been yielded.
* The body is closed when the stream completes, fails, the caller stops
  ranging (`yield` returns false), or `ctx` ends.

### D4. Failure semantics of a native stream

* Only `EventDone` carries a `Response`, and it is the last event of a
  stream that succeeds.
* A failure is yielded once, with a zero `Event`, and ends the stream —
  `Stream`'s fallback rule, unchanged. Deltas and items yielded before it
  are not a response: the caller discards them or shows them as
  incomplete. The `Stream` doc says so. This extends 0021-MADR W14 to a
  stream: the partial output is never handed back as a result.
* A stream that ends before `response.completed` is
  `ErrProviderUnavailable` marked `AfterReply`, as in `ReadStream`.

### D5. `Generate` is unchanged

* An API key's `Generate` stays a whole JSON reply through `DecodeFor`; a
  ChatGPT session's stays a stream collected by `ReadStream`. `Stream` is a
  separate path that shares the decoder. Switching `Generate` itself to a
  stream is a separate decision (Q3).

### D6. `WithRetry` buffers, and retries the whole generation

* The wrapper stays as it is: it does not implement `Streamer`, and it
  reports `NativeStreaming: Unsupported`. `Stream` over it falls back to
  the wrapper's own `Generate` and emits the result as events. That is the
  buffered stream Q1 chose: a failure at any point is retried under the
  policy (the same `retryable` test, the same waits, the once-only resend
  of an `AfterReply` failure; 0021-MADR D1, 0028-MADR D-H1), and the
  caller receives events only from the attempt that succeeds, so nothing
  is ever repeated. 0020-MADR F32's remediation stands.
* Buffering the inner provider's `Stream`, rather than calling its
  `Generate`, was considered and not taken: it yields the same events at
  the same billing, with more code.
* Live deltas and `WithRetry` are therefore exclusive. A caller that wants
  text as it arrives streams from the provider unwrapped and handles a
  failure itself; the `WithRetry` doc says so.

### D7. A conformance check for every `Streamer`

`llmtest.Run` gains a check that runs for any provider implementing
`Streamer`, over scripted replies from the harness:

* the event order: deltas and items, then exactly one `EventDone`, last;
* the concatenated text deltas equal the `MessageItem` texts of
  `EventDone`'s `Response`;
* that `Response` equals `Generate`'s for the same scripted reply;
* a failure mid-stream ends it with the error and no `EventDone`;
* stopping the range early closes the reply body, with no goroutine left
  behind under `-race`.

A harness field for a scripted stream reply is an addition to
`llmprovider/llmtest`'s exported API, which apidiff reports as compatible.

### D8. The other wires are later phases of this decision

Chat Completions (`huggingface`, `kilo`, `together`, `ollama`, OpenCode
`chat_completions`), Messages (`claude`, OpenCode `messages`), Gemini
Interactions (`gemini`) and generateContent (OpenCode `google`) each gain a
stream reader under D2–D7's contract, as an amendment of this MADR and a
phase of its PLAN, approved one at a time. When the last OpenCode route
streams, both gateways move from `BestEffort` to `Supported`.

### Owner questions

Answered by the owner on 2026-10-10.

* **Q1 (D6). A failure after the first event.** Offered: yield it
  unretried (recommended); buffer the whole stream so it can be retried,
  which gives up streaming under `WithRetry`; or restart and emit a new
  "reset" event, which adds an `EventType` every consumer must handle.
  **Answer: buffer and retry.** D6 records what follows from it.
* **Q2 (D1). OpenCode's declaration.** Offered: `BestEffort` with the
  route degradation listed (recommended); or `Unsupported` until every
  route streams. **Answer: `BestEffort`.**
* **Q3 (D5). An API key's `Generate`.** Offered: unchanged (recommended);
  or read every reply as a stream, which bounds a long generation by the
  idle limit rather than the 300 s header wait, but changes the billed
  path of every call and needs its own measurement. **Answer: unchanged.**
* **Q4 (D2). Function-call argument deltas.** Offered: none, calls arrive
  as items (recommended); or a new `EventToolCallDelta`, an `EventType`
  addition under `Event`'s "has a default case" rule. **Answer: none.**

### Consequences

* Good, because `Stream` delivers text as it is generated for `openai`,
  `grok` and OpenCode's `responses` route, with no change to any exported
  signature.
* Good, because one decoder serves `Generate` (ChatGPT) and `Stream`, so
  the goldens and the `ReadStream` tests guard both.
* Good, because `WithRetry` never hands a caller output it later repeats:
  events come only from the attempt that succeeds.
* Good, because the contract each later wire must meet is written once,
  with a conformance check to enforce it.
* Neutral, because `Generate`'s behaviour and billing are unchanged (D5).
* Neutral, because the release is a minor one (`v1.5.0`): the `llmtest`
  harness addition and the changed capability values are additive.
* Bad, because a consumer that reads `NativeStreaming` to choose a code
  path sees new values for four ids.
* Bad, because a caller cannot have both retries and live deltas from
  `WithRetry`: it streams the provider unwrapped and retries itself.
* Bad, because the OpenCode gateways stream for some models and not others
  until D8's phases land.
* Bad, because a stream that fails after deltas leaves the caller holding
  partial text it must discard; the fallback never did.
* Bad, because the measurement of D1 and the live tests are billed
  requests.

### Confirmation

* **D2:** `ReadStream`'s results on the three goldens and every existing
  test are unchanged; the decoder's events on each golden are pinned by a
  new test; a benchmark over the 0021-MADR W2 workload shows no regression
  in `ReadStream`.
* **D3, D4:** unit tests with a scripted server: a 401 before the stream
  renews once; a non-2xx is classified as today; an idle stall, an
  oversize event, an early end and a `response.failed` after deltas each
  end the stream with the expected kind; stopping early closes the body.
* **D6:** `WithRetry` over a streaming provider is not a `Streamer` and
  reports `NativeStreaming: Unsupported`; `Stream` over it, with a
  retryable failure on the first attempt, yields events from the second
  attempt only.
* **D7:** every built-in provider passes `llmtest.Run`; the new check is
  first seen failing on a deliberately broken scratch `Streamer` (events
  out of order, a missing `EventDone`, a leaked body).
* **D1:** the measurement's evidence is in the PLAN; live-tagged tests
  stream from each provider that declares the capability.
* **The API:** apidiff from `v1.4.0` shows additions only.
* **The records:** `docs/architecture.md` (the `Stream` and capabilities
  text, the `internal/wire/responses` row), each changed provider's package
  doc, and `api-standards.md` where R13's wording names the fallback.

## Pros and Cons of the Options

### Responses wire first

* Good, because the stream reader exists, is bounded and tested, and the
  ChatGPT path already exercises it against the live service.
* Good, because three providers, and one route of two more, stream after
  one decoder change.
* Good, because the failure, retry and conformance rules are settled on
  one wire before four more depend on them.
* Bad, because ten ids stream unevenly until D8's phases land.

### All five wires at once

* Good, because every provider streams in one release.
* Bad, because four stream readers are new code with no goldens, each
  needing its own live measurement, in one change.
* Bad, because the contract (D3–D7) would be settled while five readers
  depend on it, so a correction lands five times.

### ChatGPT sessions only

* Good, because the smallest change: the stream is already requested and
  read.
* Bad, because an API key on the same wire, Grok and OpenCode stay
  unstreamed with no technical reason.
* Bad, because `WithRetry` and the conformance check would still need
  deciding, for one path.

### Keep the fallback

* Good, because nothing changes and nothing is billed.
* Bad, because 0015-MADR D4's stated plan stays undone, and the largest
  gap 0030-REPORT found stays open.
* Bad, because a caller showing progress waits for the whole generation.

## More Information

* **Keeps:** [0020-MADR-remediate-v1-debugging-pass-findings.md](0020-MADR-remediate-v1-debugging-pass-findings.md)
  F32's remediation: `WithRetry` reports `NativeStreaming: Unsupported`
  (D6).
* **Implements:** [0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md)
  D4's "native streaming is added provider by provider later".
* **Extends:** [0021-MADR-harden-and-tune-after-the-v1-1-review.md](0021-MADR-harden-and-tune-after-the-v1-1-review.md)
  W14 to a stream (D4); keeps D1 and W2 as they are.
* **Keeps:** [0028-MADR-heuristics-and-performance-from-the-research-pass.md](0028-MADR-heuristics-and-performance-from-the-research-pass.md)
  D-H1 (D3, D6).
* **Prompted by:** [0030-REPORT-v1-api-and-protocol-surfaces.md](../reports/0030-REPORT-v1-api-and-protocol-surfaces.md)
  F8.
* The PLAN is
  [0031-PLAN-native-streaming-on-the-responses-wire.md](0031-PLAN-native-streaming-on-the-responses-wire.md).
