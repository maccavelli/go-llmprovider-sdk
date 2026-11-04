---
status: accepted
date: 2026-11-04
decision-makers: repository owner
consulted: this module at 8203326 (v1.26.0 is the last tag); the WHATWG HTML Living Standard's server-sent events section; anthropic-sdk-go v1.57.0 and openai-go v3.42.0, read in the module cache; a capture of 17 live and local event streams on 2026-11-04
informed: anyone who streams from a provider, or calls Generate on a provider whose Generate reads a stream, through a server that does not end its last event with a blank line
---

<!-- markdownlint-disable MD013 -->

# Discard a Server-Sent Event the Stream Ends Inside, as the WHATWG Specification Requires

## Context and Problem Statement

`wire.ReadEvents` is the one server-sent-events reader every wire's stream
decoder uses (0031-MADR E2, C12; C13 to C17). It dispatches an event when a
blank line ends it (C2). When the body ends, it also dispatches whatever data
is still pending, with no blank line after it (C1). The specification says
the opposite, in its section 9.2.6, "Interpreting an event stream":

> Once the end of the file is reached, any pending data must be discarded.
> (If the file ends in the middle of an event, before the final empty line,
> the incomplete event is not dispatched.)

The format's grammar, in section 9.2.5, agrees: `event = *( comment / field )
end-of-line`, so every event ends with an empty line
([html.spec.whatwg.org, server-sent events](https://html.spec.whatwg.org/multipage/server-sent-events.html#event-stream-interpretation)).

No record chose the module's behaviour. Three tests require it (C4, C5, C6).
The Responses case came with the wire packages' extraction on 2026-10-01
(`940fee0`), and the shared reader kept it.
[0054-REPORT-ideas-status-at-v1-25-0.md](../reports/0054-REPORT-ideas-status-at-v1-25-0.md)
found it (C18).

The two behaviours differ in one place only: a body that ends *cleanly* with
an event pending. A body cut off where its framing shows it, chunked or with
a `Content-Length`, reads as `unexpected EOF`. That is already the reader's
read failure, the same either way. A probe on 2026-11-04 showed this:

| How the server's body ends | What net/http's client returns |
| :--- | :--- |
| chunked, cut inside a chunk | `unexpected EOF` |
| `Content-Length`, cut short | `unexpected EOF` |
| delimited by the connection's close, cut | a clean end |
| chunked and complete, the last event with no blank line | a clean end |

A clean end with an event pending happens in three cases:

* **Case 1: a complete answer whose server leaves off the final blank
  line.** Today it succeeds.
* **Case 2: a body delimited by the connection's close, cut mid-line.**
  Today the cut JSON goes to the decoder, and its error has no kind: no
  sentinel, no `*APIError`, no `*url.Error`. So `retryable` does not retry it
  (C8, C10). A scratch test of the Responses decoder showed this, as `p:
  decode stream event: unexpected end of JSON input`.
* **Case 3: a service's failure event, last, with no blank line.** Today it
  is the service's own classified failure.

### The captures

On 2026-11-04 a scratch program streamed "Say hi" through each route the
live streaming tests cover, configured as those tests configure it, and
through a local Ollama, a local llama.cpp server and OpenRouter. It wrapped
`NewHTTPClient`'s transport and kept each `text/event-stream` body. When the
reader stopped at its done event, it drained the rest of the body, so each
ending is the server's. It recorded no header, query string or key. Its
self-test served bodies ending in a blank line, in one line ending and in
none, and two planted breaks of it (no drain; a classifier that always
answers "conforming") each failed the self-test.

| Route | Wire | Transport | Last bytes |
| :--- | :--- | :--- | :--- |
| OpenAI, `gpt-4.1-mini` | Responses | HTTP/2 | `}\n\n` |
| xAI, `grok-4.6` | Responses | HTTP/2 | `}\n\n` |
| OpenCode Zen and Go | Responses | HTTP/1.1, chunked | `}\n\n` |
| Anthropic, `claude-haiku-5-5` | Messages | HTTP/2 | `message_stop" }\n\n` |
| OpenCode Zen and Go, `qwen3.8-flash` | Messages | HTTP/1.1, chunked | `message_stop"}\n\n` |
| Hugging Face's router | Chat Completions | HTTP/2 | `[DONE]\n\n` |
| Together | Chat Completions | HTTP/2 | `[DONE]\n\n` |
| OpenCode Zen and Go, `glm-5.3-flash` | Chat Completions | HTTP/1.1, chunked | `[DONE]\n\n` |
| OpenRouter, through `chatcompat` | Chat Completions | HTTP/2 | `[DONE]\n\n` |
| Ollama, local, `llama3.2` | Chat Completions | HTTP/1.1, chunked | `[DONE]\n\n` |
| llama.cpp's `llama-server`, local | Chat Completions | HTTP/1.1, chunked | `[DONE]\n\n` |
| Google, `streamGenerateContent` | generateContent | HTTP/2 | `}\r\n\r\n` |
| OpenCode Zen, `gemini-3.5-flash-lite` | generateContent | HTTP/1.1, chunked | `}\n\n` |
| Gemini's Interactions | Interactions | HTTP/2 | `event: done\ndata: [DONE]\n\n` |

Every body ended with a blank line, and in each the SDK yielded `EventDone`
with no error. None was delimited by the connection's close. Three were not
captured:

* **Kilo:** its default model answered HTTP 402 `usage_limit_exceeded`, and
  a free model HTTP 400 `data_collection_required`, a setting of the
  account. No stream was sent.
* **The ChatGPT subscription:** its sign-in switch was not set. The three
  streams recorded from it in `llmprovider/providers/openai/testdata/` end
  with a blank line.
* **Case 3:** no service sent a failure event; none can be made to on
  demand.

The official Go SDKs read the same way the specification does.
`anthropic-sdk-go` v1.57.0 and `openai-go` v3.42.0, in
`packages/ssestream/ssestream.go`, dispatch only on an empty line, and drop
pending data when the scanner ends. A service whose events lacked the blank
line would lose them in its own SDK. `openai-go` does not require `[DONE]`:
a stream that never sends it ends with no error. This module requires each
wire's done event (C3), so it is stricter there.

### Code claims

<!-- claims: self=8203326 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | The reader dispatches pending data at the end | `llmprovider/internal/wire/sse.go:82` | `if stop, err := next(); stop \|\| err != nil {` |
| C2 | It dispatches on a blank line | `llmprovider/internal/wire/sse.go:60` | `case len(line) == 0 && err == nil:` |
| C3 | The early-end error | `llmprovider/internal/wire/sse.go:86` | `transport.AfterReply(fmt.Errorf("stream ended before %s", done)))` |
| C4 | The test that pins the dispatch | `llmprovider/internal/wire/sse_0031b_test.go:43` | `func TestReadEvents_LastEventWithoutBlankLine(t *testing.T) {` |
| C5 | 0053's case at the stream end | `llmprovider/internal/wire/sse_0053_test.go:21` | `"at the stream end": "data: x",` |
| C6 | The Responses case | `llmprovider/internal/wire/responses/wire_test.go:100` | `t.Run("data split over lines, and a final event with no blank line", func(t *testing.T) {` |
| C7 | `ErrProviderUnavailable` is retried | `llmprovider/retry.go:340` | `case errors.Is(err, ErrRateLimited), errors.Is(err, ErrProviderUnavailable):` |
| C8 | A cut event's decode error has no kind | `llmprovider/internal/wire/responses/responses.go:657` | `return false, fmt.Errorf("%s: decode stream event: %w", provider, err)` |
| C9 | 0053's mark at the reader | `docs/decisions/0053-MADR-stream-retry-and-cache-defects.md:250` | `### D3. A failure event inside a 200 stream is retried at most once (F3)` |
| C10 | A kindless error is retried only as a send failure | `llmprovider/retry.go:350` | `return errors.As(err, &urlErr) && !transport.Unsendable(err)` |
| C11 | The live streaming tests require EventDone | `llmprovider/live_stream_0031b_test.go:58` | `if deltas == 0 \|\| !done {` |
| C12 | 0031's shared reader | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:353` | `**E2. One server-sent-events reader, shared.**` |
| C13 | Chat Completions' done event | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:666` | `return wire.ReadEvents(provider, body, "[DONE]", func(payload []byte) (bool, error) {` |
| C14 | generateContent's | `llmprovider/internal/wire/generatecontent/generatecontent.go:301` | `return wire.ReadEvents(provider, body, "a finishReason", func(payload []byte) (bool, error) {` |
| C15 | Messages' | `llmprovider/internal/wire/messages/messages.go:528` | `return wire.ReadEvents(provider, body, "message_stop", dispatch)` |
| C16 | Responses' | `llmprovider/internal/wire/responses/responses.go:694` | `return wire.ReadEvents(provider, body, "response.completed", dispatch)` |
| C17 | Interactions' | `llmprovider/providers/gemini/interactions.go:263` | `return wire.ReadEvents(provider, body, "interaction.completed", s.event)` |
| C18 | The audit's finding | `docs/reports/0054-REPORT-ideas-status-at-v1-25-0.md:128` | `* **The stream reader dispatches an event cut off by the end of the stream**` |

## Decision Drivers

* **The specification, and the reference readers.** The format's own rule,
  and the two official Go SDKs, discard the event.
* **An error says what happened.** A stream cut on a close-delimited body is
  a service that stopped, not a malformed answer.
* **Measured, not assumed.** Whether a supported server leaves off the blank
  line decides the cost of following the rule.
* **No exported name and no request changes** (R48, R45).

## Considered Options

* **A. Discard the pending event at the end of the body,** as the
  specification requires, and say so in the early-end error.
* **B. Keep dispatching it,** and record the departure and its reason.
* **C. Discard it, except when it is the wire's done event.**

## Decision Outcome

Chosen option: **"A. Discard the pending event at the end of the body"**,
chosen by the owner on 2026-11-04. It is the format's rule and the official
SDKs' behaviour. It classifies a stream cut on a close-delimited body as the
service's failure to finish. On 2026-11-04, every server measured ended its
last event with a blank line, so no stream that works today changes.

### D1. The reader discards an event the stream ends inside

* **The rule:** at a clean end of the body, pending data is not dispatched.
  Data that a blank line ended was dispatched already, as today.
* **The error:** with data pending, the early-end error names the cause:
  "stream ended inside an event, before" the done event. Its kind and mark
  are the early end's, `ErrProviderUnavailable` marked after the reply (C3),
  so `WithRetry` and `WithStreamRetry` ask again at most once (C7). With
  none pending, the error is today's, "stream ended before" the done event.
* **What is pending:** data lines only, as today. An `event:` or `id:` line,
  or a comment, with no data, was never an event to dispatch.
* **The doc:** `ReadEvents`' doc states the rule and cites this record.

### D2. Tests

* **Changed, each for this record:**
  * `TestReadEvents_LastEventWithoutBlankLine` (C4) requires the opposite of
    today: the pending event is not dispatched, and the error is the early
    end's, naming "inside an event". A table covers no line ending, one LF,
    one CR LF, and a comment after the data. An `event:` line with no data
    gets today's text;
  * `TestReadEvents_DispatchErrorMarked_0053`'s "at the stream end" case
    (C5) goes: no dispatch happens at the end, so no dispatch error can be
    marked there. Its "at an event" case stands;
  * the Responses case (C6) ends its split event with a blank line, and
    keeps checking the split. A new case beside it requires the early-end
    error when the final event has none.
* **Red and plants:** each changed test fails at `8203326`. A planted
  return of the dispatch at the end fails each. A plant of today's text for
  the pending case fails the reader's test.
* **The live pin:** the live streaming tests require `EventDone` (C11).
  Under D1, a service that left off the final blank line would fail them
  with D1's error. So they pin the captures without a new test.

### D3. Docs, records and the release

* **Docs:** `ReadEvents`' doc; `docs/architecture.md`'s row for
  `internal/wire`; `CHANGELOG.md`'s Fixed.
* **Records amended once this is accepted:**
  0031-MADR-native-streaming-on-the-responses-wire.md E2, whose framing
  gains the rule; 0053-MADR-stream-retry-and-cache-defects.md D3, whose mark
  no longer has a dispatch at the stream end to cover.
* **Release:** as the owner answers Q1.

### Consequences

* Good, because the reader follows the format's rule and reads as the
  official SDKs do.
* Good, because a stream cut on a close-delimited body is
  `ErrProviderUnavailable`, retried once, not a kindless decode error that
  is never retried.
* Good, because the early-end error tells a caller with a non-conforming
  server what its server did.
* Neutral, because no measured server changes: all 17 captured routes end
  with a blank line.
* Bad, because a server that leaves off the final blank line, none measured,
  turns a complete answer into `ErrProviderUnavailable`. Under `WithRetry`
  or `WithStreamRetry`, that costs one more billed request, which fails the
  same way.
* Bad, because a service's failure event sent last with no blank line loses
  its classification, such as `ErrQuotaExhausted`, which is not retried, and
  becomes the early end's. That is unmeasured, and no service was seen to
  send one.

### Confirmation

* D2's tests pass, each failed at `8203326`, and each failed on its plant.
* Every other test at `8203326` passes unchanged: with D1 planted on a
  scratch clone, `go test ./...` failed in exactly C4, C5 and C6.
* The live streaming tests pass after D1, Kilo aside while its account
  refuses.
* `tools/apidiff.py v1.26.0` reports no change; G-wire's goldens do not
  change; `make docs-check` reports 0.

## Pros and Cons of the Options

### A. Discard the pending event

* Good, because it is the specification's rule and the official SDKs'
  behaviour.
* Good, because case 2 gets the right kind.
* Neutral, because the specification pairs the rule with `EventSource`
  reconnecting and resending from `Last-Event-ID` (sections 9.2.2 and
  9.2.4). This module does not resume, so a discarded event is recovered
  only by asking again.
* Bad, because cases 1 and 3 change for a non-conforming server.

### B. Keep dispatching it

* Good, because it accepts what any server sends, and nothing changes.
* Good, because case 3 keeps the service's classification.
* Bad, because it departs from the format's rule and the official SDKs, for
  servers none of which was measured to need it.
* Bad, because case 2 stays a kindless error, not retried.

### C. Discard it, except the done event

* Good, because case 1 would still succeed.
* Bad, because it tells one wire's terminal event from another in a reader
  that knows only data; each wire would have to say which payload is its
  done event.
* Bad, because it accepts a malformed stream for one event and not others.
  That is a special case for servers none of which was measured to need it.

## Owner questions

Answered by the owner on 2026-11-04, as recommended.

* **Q1 (D3). The release.** Offered:
  * `v1.26.1` (recommended): it fixes a departure from the specification.
    It adds or changes no exported name, and changes no measured server's
    result;
  * `v1.27.0`, since a caller can see a result change, as 0068-MADR's Q5
    answer reasoned;
  * no tag yet, holding the change for the next release.

  **Answer: `v1.26.1`,** tagged by the owner.

## More Information

* **Stands:** 0021-MADR-harden-and-tune-after-the-v1-1-review.md D1, the
  early-end error and its mark, whose kind D1 keeps;
  0051-MADR-opt-in-streaming-retry.md D2.
* **Exported names:** none added (R55 has nothing to list).
* The PLAN is
  [0069-PLAN-discard-an-sse-event-the-stream-ends-inside.md](0069-PLAN-discard-an-sse-event-the-stream-ends-inside.md).
