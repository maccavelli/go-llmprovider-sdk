---
status: proposed
date: 2026-10-20
decision-makers: repository owner
consulted: GitHub issue 4 ("WithRetry buffers a stream", 2026-10-20); this module at b5583cf (v1.18.0 is d19f7a3); gobble-cli at 6ab7928; a prototype and probes on a scratch copy of b5583cf (method below); 0031-MADR-native-streaming-on-the-responses-wire.md (D4, D6, Q1); 0044-MADR-observe-retries.md
informed: anyone who wraps a provider in WithRetry and streams from it
---

<!-- markdownlint-disable MD013 -->

# Stream Through a Retry, Opt-In: Retry Before the First Event, Pass Through After

## Context and Problem Statement

`WithRetry` and `WithRetryNotify` return a provider that does not stream
natively, even when the provider they wrap does (C1, C2). `llmprovider.Stream`
over the wrapper finds no `Streamer` (C6) and falls back to the wrapper's
`Generate`, which retries the whole generation and emits the result as events
once it has succeeded. That is
[0031-MADR-native-streaming-on-the-responses-wire.md](0031-MADR-native-streaming-on-the-responses-wire.md)
D6, which followed from the owner's answer to its Q1 ("buffer and retry"):
"Live deltas and `WithRetry` are therefore exclusive" (C8). A caller that
wants text as it arrives streams from the unwrapped provider and handles a
failure itself; the docs say so (C1, C9).

[Issue 4](https://github.com/maccavelli/go-llmprovider-sdk/issues/4) asks for
an **opt-in** way to have both, leaving D6's default unchanged. Its consumer,
gobble-cli, wraps every provider in `WithRetry` (G1) and already shows each
text delta as its agent loop receives it, so:

* a user sees nothing until the whole reply is generated, on every provider
  that streams natively;
* gobble reports `NativeStreaming` false while the wrapper buffers (G2);
* a first-byte deadline cannot be built over a buffered stream: it would
  bound the whole reply;
* streaming from the unwrapped provider, as D6 suggests, gives up the
  policy's classification, waits, cap, deadline rule, notices and the
  once-only resend of a written request (0021-MADR D1, 0028-MADR D-H1),
  which the consumer would have to write again.

The issue asks for a result that implements `Streamer` and reports the inner
provider's `NativeStreaming`; that retries a failure **before the first
event** under the same policy as `Generate`; and that, **after the first
event**, passes every event through and ends the stream on a failure with
that error and no `EventDone`, as 0031-MADR D4 has it, with no retry. That is
Q1's first answer, "yield it unretried", for this mode only. A caller that
chooses it accepts that a failure after the first delta ends the turn.

The question is the shape of that opt-in, and what exactly it promises.

### How the facts were found (2026-10-20)

On a scratch copy of `b5583cf` (`git archive HEAD`), never in the tree:

* **A prototype** of the issue's sketch, `WithStreamRetry(p, policy,
  notify)`: a wrapper whose `Stream` ranges over the inner `Streamer`,
  retries when the inner stream's first pair is an error, under the same
  decisions as `retrying.Generate` (C3 to C5), and passes every pair through
  once one event has been yielded.
* **`llmtest`, made to wrap** every provider its `build` makes (and the one
  its reauth check makes), as an environment variable chose: none,
  `WithRetry`, or the prototype, each with `BaseDelay` 1 ms and `MaxDelay`
  10 ms.
* **A probe** added to `llmtest`'s checks: for each provider whose harness
  sets `StreamText` and `StreamFailure`, five scripted servers, each
  answering the first request one way and later ones another, through the
  prototype with a notice counter.

The prototype and the probe were scratch files, not kept; what they printed
is below.

### What was measured

* **M1. The conformance suites.** `go test -run '^TestConformance'
  ./llmprovider/providers/...` passed in all three modes: unwrapped, under
  `WithRetry` (whose result is not a `Streamer`, so D7's check returns
  early), and under the prototype (a `Streamer`, so D7's check ran over it).
  To show the wrapping took effect, the prototype was planted to drop text
  deltas: 34 subtests failed, among them "D7 (native streaming, 0031-MADR
  D2): the text deltas joined are \"\"; EventDone's message text is
  \"hello\"".
* **M2. A failure before the first event is the stream's first pair, and
  classified.** For every built-in provider, in each of the fifteen
  conformance runs over the ten ids (some harnesses run twice, as for a
  stored and an unstored request, or two OpenCode routes), identically:

  | Case (first request, then later ones) | Requests | Notices | Result |
  | :--- | :--- | :--- | :--- |
  | 503, then `StreamText` | 2 | 1 | the stream succeeds; the deltas joined equal `EventDone`'s text |
  | `StreamFailure` (fails after one delta) | 1 | 0 | one delta, then the error (`ErrProviderUnavailable`), no `EventDone` |
  | 503 every time | 3 | 2 | the error, after `MaxAttempts`; no event |
  | 200 with no body (a reply that ends early), then `StreamText` | 2 | 1 | the stream succeeds |
  | 200 with no body every time | 2 | 1 | the error: the once-only resend of a failure after the reply began holds |

* **M3. The scripted replies carry one text delta** for every provider but
  one OpenCode Zen route, which carries two; `llmtest`'s contract is "its
  text in at least one delta" (C10).
* **M4. `llmtest`'s capabilities check** holds a provider to implementing
  `Streamer` exactly when its `NativeStreaming` is not `Unsupported` (C7).
  A wrapper that implements `Streamer` over a provider that does not would
  fail it.
* **M5. Liveness can be shown with one delta, by holding the reply back.**
  A second probe served each harness's `StreamText` script in parts, a
  part being what ends in a blank line, as an event stream's events do.
  * **Holding back only the last part** caught a wrapper that collects the
    whole inner stream before yielding on every wire but Gemini's, whose
    reader ends at `interaction.completed`, the part before the last
    (`[DONE]`): the inner stream ended without it.
  * **Refined:** the unwrapped provider is first served one part at a
    time, 50 ms apart, to find the part `k` after which its first text
    delta arrives; then a wrapped provider is served parts 0 to `k` at once,
    and the rest only once the caller has a delta, or after 1.5 s. Run
    three times over the fifteen conformance runs, every time: the
    unwrapped provider and the prototype had their first delta within 1 ms,
    with the same events in the same order; the collecting wrapper waited
    the full 1.5 s every time, Gemini's included, though its events were
    the same, so comparing events alone does not catch it. `k` was 0 for
    two- and three-part scripts and 2 for the six-part ones (Claude's,
    Gemini's, and one of OpenCode Go's routes).

### Code claims

<!-- claims: self=b5583cf gobble-cli=6ab7928 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | `WithRetry`'s doc: no native stream | `llmprovider/retry.go:81` | `// The result does not stream natively, even when p does; Stream falls back to` |
| C2 | The wrapper drops `NativeStreaming` | `llmprovider/retry.go:135` | `caps.NativeStreaming = Unsupported` |
| C3 | It retries the inner `Generate` | `llmprovider/retry.go:142` | `resp, err := r.inner.Generate(ctx, req)` |
| C4 | Once-only after a reply began | `llmprovider/retry.go:149` | `if transport.IsAfterReply(err) {` |
| C5 | A notice only once a wait is certain | `llmprovider/retry.go:167` | `if r.notify != nil {` |
| C6 | `Stream` uses a `Streamer` when there is one | `llmprovider/stream.go:50` | `if s, ok := p.(Streamer); ok {` |
| C7 | `llmtest`: `Streamer` exactly when it streams | `llmprovider/llmtest/llmtest.go:301` | `if _, streams := p.(llmprovider.Streamer); streams != (caps.NativeStreaming != llmprovider.Unsupported) {` |
| C8 | 0031's D6: exclusive | `docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md:195` | `` Live deltas and `WithRetry` are therefore exclusive `` |
| C9 | The architecture's "not yet" list | `docs/architecture.md:928` | `` Live text through `WithRetry`. `` |
| C10 | The stream harness's contract | `llmprovider/llmtest/llmtest.go:97` | `// StreamText and StreamFailure, when both are set and the provider` |
| C11 | `WithRetryNotify`'s doc: notices before the first event | `llmprovider/retry.go:101` | `// Through llmprovider.Stream the notices come before the first event, since` |
| C12 | R28 names the retry wrappers | `docs/guides/api-standards.md:210` | `` error kind; `WithRetryNotify` retries the same `` |
| G1 | gobble wraps every provider in `WithRetry` | `gobble-cli:llm/provider/provider.go:197` | `w = llmprovider.WithRetry(p, policyOf(r))` |
| G2 | gobble's `NativeStreaming` waits on it | `gobble-cli:llm/llm.go:46` | `// NativeStreaming stays false while the SDK's retry wrapper buffers the stream (0005-PLAN F4-1b).` |

## Decision Drivers

* **D6's default stands.** A caller of `WithRetry` today gets a buffered
  stream that is retried whole and never repeats text; that does not change
  under it.
* **One policy.** The streaming mode retries what `Generate` retries, waits
  as long, stops where it stops, and reports the same notices: one decision,
  not a second copy that drifts.
* **Nothing repeated, nothing lost.** A caller never receives an event from
  an attempt that is later retried, and never loses a failure.
* **R10 holds.** A provider is a `Streamer` exactly when its capabilities
  say it streams (C7).
* **R55: a new export is justified,** by its call site: gobble-cli's G1.
* **R48: compatible.** `apidiff` against `v1.18.0` reports an addition only.

## Considered Options

* **A. A new wrapper, `WithStreamRetry(p, policy, notify)`:** a `Streamer`
  over a `Streamer`, retrying before the first event and passing through
  after it; `WithRetry` and `WithRetryNotify` unchanged.
* **B. A field on `RetryPolicy`,** a `bool` that `WithRetry` and
  `WithRetryNotify` read to choose the streaming mode.
* **C. A pair of wrappers,** `WithStreamRetry(p, policy)` and
  `WithStreamRetryNotify(p, policy, notify)`, mirroring the existing pair.
* **D. Stream live in `WithRetry` itself:** Q1's first answer for every
  caller.
* **E. Restart on a failure after the first event,** with a new "reset"
  event: Q1's third answer.
* **F. Leave it:** a caller streams from the unwrapped provider, as D6 says.

## Decision Outcome

Chosen option: **"A. A new wrapper, `WithStreamRetry(p, policy, notify)`"**
(the owner's Q1), because it is the one shape that leaves every
existing call and doc true, adds one exported name, and keeps the policy one
decision shared by `Generate` and `Stream`.

### D1. The function

```go
func WithStreamRetry(p Provider, policy RetryPolicy, notify func(context.Context, RetryNotice)) Provider
```

`notify` may be nil, for no notices; it is called exactly when
`WithRetryNotify` would call it, with the same `RetryNotice`. `RetryPolicy`
and `RetryNotice` are unchanged. The result is safe for concurrent use, as
the provider it wraps is (R20).

### D2. Its `Stream`: the first event is the line

The **first event** is the first pair the inner stream yields with a nil
error: a text delta, a reasoning delta or an item. The caller has seen
nothing before it.

* **Before it,** a failure, the inner stream's first pair, is decided
  exactly as `Generate` decides one, by one unexported function the two
  share: the `retryable` test; a failure after the reply began, or after
  the request was written, retried at most once per `Stream` call
  (0021-MADR D1, 0028-MADR D-H1); `MaxAttempts`, with the "failed after N
  attempts" error at the last; a wait the service asks for beyond
  `MaxDelay`, or past `ctx`'s deadline, not begun, the error returned at
  once; `notify` before each wait, on the goroutine ranging the stream;
  `ctx` ending during a wait gives an error matching both `ctx.Err()` and
  the failure. The failure that ends the stream is yielded once, with a
  zero `Event`.
* **After it,** every pair passes through as the inner stream yields it. A
  failure ends the stream with that error, unchanged and unwrapped, and no
  `EventDone`, as 0031-MADR D4 has it: no retry, no wait, no notice. The
  events already yielded are not a response; the caller discards them or
  shows them as incomplete.
* **A caller that stops ranging** stops the inner stream at once, which
  closes its reply (0031-MADR D3, D7).

### D3. Its other methods, and a provider that does not stream

* `Generate` is `WithRetryNotify`'s: the whole generation, retried whole.
* `Capabilities` is the inner provider's, `NativeStreaming` included.
* `ListModels`, when the inner provider lists, goes through it once
  (0020-MADR F32).
* Over a provider that is not a `Streamer`, there is no live stream to pass
  through: `WithStreamRetry` returns `WithRetryNotify(p, policy, notify)`,
  whose result is not a `Streamer` and reports `NativeStreaming:
  Unsupported`, as R10 requires (M4).

### D4. `WithRetry`, `WithRetryNotify` and 0031-MADR

* Both are unchanged: buffered, retried whole, nothing repeated. Their docs
  name `WithStreamRetry` where they say a caller who wants live text
  streams from the provider itself (C1, C11).
* 0031-MADR D6 stands for them. Its bullet "Live deltas and `WithRetry` are
  therefore exclusive" (C8) is narrowed by this record: 0031-MADR gains a
  dated amendment that says so and links here, without rewriting D6.
* R28 (C12) names `WithStreamRetry` beside the other two, and the
  architecture's "not yet" entry (C9) goes, its list entries gaining the
  new wrapper.

### D5. The tests

* **In `llmprovider`,** over a scripted inner `Streamer`, under `-race`:
  * **live:** the caller receives the first text delta before the inner
    stream produces the second (the script waits for the caller's receipt),
    and two or more deltas arrive before `EventDone`, joined equal to its
    text;
  * a 503 before the first event retried, then success: two inner
    streams, one notice;
  * a failure after the first delta: the error, no `EventDone`, one inner
    stream, no notice;
  * the policy's edges, each against `Generate`'s: `MaxAttempts`; the
    once-only resend of an `AfterReply` failure; a service wait over
    `MaxDelay`; a wait past the deadline; `ctx` ending during a wait; a
    failure that is not retryable;
  * a caller that stops after the first event: the inner stream stopped;
  * over a provider that is not a `Streamer`: no `Streamer`,
    `NativeStreaming` `Unsupported`; a `ModelLister` kept.
* **In `llmtest.Run`,** for a provider that is a `Streamer` and whose
  harness sets `StreamText` and `StreamFailure`, a check over the provider
  wrapped in `WithStreamRetry`, as M2 measured: a 503 (the harness's
  `Error`) then `StreamText` gives two requests, one notice and D7's
  success properties; `StreamFailure` gives one request, the error and no
  `EventDone`. It needs no new `Harness` field, so `llmtest`'s exported API
  is unchanged; every built-in provider's suite runs it, and so does any
  third party's.
* **Live on every wire, at least one delta, two preferred** (the owner's
  Q3, as revised). The same check shows the wrapper live, by M5's
  heuristic:
  * it finds, on the unwrapped provider, the part of `StreamText` after
    which the first text delta arrives, then serves the wrapped provider
    up to that part and holds the rest back until the caller has a delta;
    a wrapper that has none by the limit fails, as one that collects the
    stream would;
  * the wrapped stream's events equal the unwrapped stream's, in order,
    so a wrapper that merges or drops a delta fails where the script
    carries two;
  * a script with no part after its first delta (one written in one
    piece, or not as an event stream) cannot show liveness: the check
    says so in the test log and passes.

  `StreamText`'s contract (C10) stays "its text in at least one delta",
  and gains that two or more are preferred, since one cannot show that
  deltas are not merged; a one-delta script passes, with a line in the test
  log. A third party's harness that passes today passes still.
* **The built-in scripts carry two,** and are held to it. Every built-in
  harness's `StreamText` script, twelve in all (one in each of eight
  provider packages, and OpenCode's four, one per route), carries its text
  in two deltas, joined equal to the text `Text` writes, so that D7's
  equality with `Generate` still holds. Each of the nine packages has a
  test that its own scripts give at least two text deltas through its
  provider's `Stream`, so a script that slips back to one fails in this
  module, not only in a log.
* Each test is seen to fail on a planted break first.

### D6. The docs and the release

* `WithStreamRetry`'s doc says what D2 and D3 say, the issue's trade-off
  included: a failure after the first event ends the stream.
* An example, `ExampleWithStreamRetry`, runs offline (R50).
* `CHANGELOG.md`: `WithStreamRetry` under Added; under Changed,
  `llmtest.Run`'s new check of a streaming provider through it, which a
  provider whose stream does not yield a failure before its first event as
  a classified first pair fails. The release is
  `v1.19.0`, a minor release (the owner's Q2). Issue 4 is answered with the
  release, on the owner's word.

### Consequences

* Good, because a consumer gets live text and the policy together, by
  changing one call: gobble's G1, then G2's `NativeStreaming` from the
  inner provider.
* Good, because a failure before the first event, which M2 shows is how a
  rate limit, an unavailable service and a reply that ends before its first
  event all arrive on every wire, is retried as `Generate` retries it.
* Good, because every existing caller, doc and test of `WithRetry` and
  `WithRetryNotify` is unchanged.
* Good, because `llmtest` checks the pre-first-event contract of every
  provider, built-in or not, that the new wrapper relies on.
* Bad, because a caller of `WithStreamRetry` loses a turn to a failure
  after the first event that `WithRetry` would have retried. It is the
  caller's choice, and the doc says so.
* Bad, because the module has three retry constructors where it had two.
* Good, because liveness through the wrapper is shown on every wire,
  whatever its script's delta count, and two or more deltas on every
  built-in wire, as the issue's acceptance asks (Q3).
* Good, because `StreamText`'s contract is not tightened: a one-delta
  harness passes, and is told that two show more.
* Bad, because `llmtest.Run` gains a check a third-party streaming
  provider can fail, as 0031-MADR D7 did; it is named under Changed. No
  exported identifier changes.
* Bad, because the check waits: about 50 ms a part until the first delta
  on the unwrapped provider, and the limit only when it fails.

### Confirmation

* D5's tests pass, and each failed on its planted break.
* Every built-in provider's `TestConformance` passes with the new check.
* `tools/apidiff.py v1.18.0` lists `WithStreamRetry` as the only change, an
  addition; `make docs-check` and `sdkaudit -check` report 0.
* The issue's acceptance, each item by the test that shows it: two or more
  deltas before `EventDone` (the unit test, live; on every built-in wire,
  `llmtest`'s event comparison over the two-delta scripts, which the nine
  packages' tests hold to two); a 503 before the first event retried, two
  requests, one notice (the unit test and the `llmtest` check); a failure
  after the first delta, one request (both); D7's checks over the wrapped
  provider (`llmtest`).
* The liveness check fails a collecting wrapper on every wire, Gemini's
  included, as M5 measured.

## Pros and Cons of the Options

### A. A new wrapper, `WithStreamRetry(p, policy, notify)`

* Good, because `WithRetry`'s and `WithRetryNotify`'s docs and behaviour
  stay exactly true; the new mode is documented in one place.
* Good, because the call site says which mode it is.
* Good, because one exported name serves callers with and without notices.
* Neutral, because its `notify` argument differs from the `WithRetry` /
  `WithRetryNotify` split, which exists because notices came later
  (0044-MADR).

### B. A field on `RetryPolicy`

* Good, because it adds a field, not a function, and a consumer that reads
  its policy from configuration can set it there.
* Good, because a `bool` keeps `RetryPolicy` comparable and plain data,
  which 0044-MADR's options B and C did not.
* Bad, because a policy field would change the result's method set:
  whether it is a `Streamer` would depend on a value, and `WithRetry`'s doc
  ("does not stream natively") and `WithRetryNotify`'s (C11) would each
  become conditional.
* Bad, because a policy built for one wrapper and passed to another changes
  its behaviour silently.

### C. A pair of wrappers

* Good, because it mirrors `WithRetry` and `WithRetryNotify`.
* Bad, because it adds two exported names where one serves; R55 asks each
  to be justified, and the second has no call site that the first, with a
  nil `notify`, does not serve.

### D. Stream live in `WithRetry` itself

* Good, because no name is added, and every consumer streams live.
* Bad, because it reverses the owner's Q1 answer for every caller: a
  failure after the first event, retried today, would end the stream. That
  is a behaviour change under a minor release.

### E. Restart with a "reset" event

* Good, because a failure after the first event could still be retried.
* Bad, because it adds an `EventType` every consumer must handle, and a
  caller that does not shows text twice; 0031-MADR Q1 declined it.

### F. Leave it

* Good, because nothing changes.
* Bad, because each consumer writes the policy again, and the issue's
  consumer has waited on this since 2026-10-11.

## Owner questions

Answered by the owner on 2026-10-20.

* **Q1 (D1). The shape.** Offered: A, a new `WithStreamRetry(p, policy,
  notify)` (recommended); B, a `bool` field on `RetryPolicy`; C, a pair of
  wrappers. **Answer: A.**
* **Q2 (D6). The release.** Offered: `v1.19.0` (recommended), a minor
  release for an addition. **Answer: `v1.19.0`.**
* **Q3 (D5). Two deltas per wire.** The scripted replies carry one text
  delta for every provider but one route (M3). Offered: the unit test shows
  the live property once, and the `llmtest` check runs on the scripts as
  they are (recommended); or every provider's harness gains a two-delta
  script, in nine packages, and `llmtest` asserts two deltas, which is a
  change to its `StreamText` contract (C10). **Answer: two deltas.**
  **Revised the same day,** on the owner's question whether the contract
  could stay flexible by a better heuristic: at least one delta, two
  preferred; liveness shown by holding the reply back (M5); and, of the
  two ways offered to keep the built-in scripts at two (a line in the
  test log, or a test in each provider package), **a test in each
  package**, the robust one. D5's last two bullets.

## More Information

* Narrows 0031-MADR D6's last bullet (C8) for an opt-in caller; D6, D4 and
  Q1's answer stand for `WithRetry` and `WithRetryNotify`.
* The retry policy is 0015-MADR D8's, with 0021-MADR D1 and T6/T7,
  0026-MADR F22, 0028-MADR D-H1 and 0044-MADR D1/D2.
* The PLAN, `0051-PLAN-opt-in-streaming-retry.md`, follows once this
  record is reviewed.
