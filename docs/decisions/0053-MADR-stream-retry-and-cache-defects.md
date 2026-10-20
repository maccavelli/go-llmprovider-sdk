---
status: proposed
date: 2026-10-20
decision-makers: repository owner
consulted: this module at cadaad5 (v1.19.0 is 87b6c9c); probes on a scratch copy of cadaad5 (method below); a read-only analysis of the catalog, retry, transport, streaming and redaction code on 2026-10-20, whose optimization ideas are recorded in 0054-REPORT-heuristics-and-performance-opportunities.md; Ollama's API reference (docs/api.md in the ollama repository)
informed: anyone who streams from a provider, wraps one in WithRetry or WithStreamRetry, lists models, or lists Ollama's
---

<!-- markdownlint-disable MD013 -->

# Fix Six Defects in Streaming, Retry and the Catalog's Caches

## Context and Problem Statement

A read-only analysis on 2026-10-20 named five defects. Proving the first found
a sixth in the same code. Each is proven below by a probe that ran on a scratch
copy of `cadaad5`, never in the tree. None needs an exported name. This record
decides how each is fixed:

* **F1. Two stream decoders accumulate text quadratically.** Chat
  Completions joins each `reasoning_details` fragment to the text before it
  with `prev + next` (C1). generateContent joins each text part with `+=` (C2).
  A stream of N fragments copies O(N²) bytes. The control path, text deltas,
  uses a `strings.Builder` and is linear.
* **F2. Four stream decoders index a slice with an integer the server
  sends, unchecked.** The four are Chat Completions' tool calls (C4) and
  `reasoning_details` (C3), Messages' content blocks (C5), and Gemini
  Interactions' steps (C6). A negative index panics the caller's goroutine,
  because nothing recovers it (M3). A large one allocates a slot for every
  index up to it. F1's probe found this; the analysis had not.
* **F3. A failure event inside a 200 stream is retried past the once-only
  rule.**
  * `WithRetry`'s doc promises that "a failure while reading a reply the
    service had begun to send … is retried at most once" (C9).
  * The shared stream reader marks its own read failure and early end so
    (C8). It returns the decoders' in-stream failures unmarked (C7), such as
    Responses' `response.failed` and Messages' `error` event.
  * So `WithRetry` sends them up to `MaxAttempts` times (C10). Each
    attempt may have generated, and billed, text.
* **F4. The listing cache's single-flight shares nothing, and its claimant's
  context drives the fetch.**
  * The caller that claims a key fetches on its own context (C12), and stores
    nothing when the fetch fails.
  * Each waiter wakes on a bare channel (C11), finds no fresh listing, and
    claims and fetches again: N concurrent callers make N serial requests.
  * A claimant that is cancelled ends the fetch its waiters are waiting for,
    and one of them starts it again.
  * The metadata document's single-flight already does both properly: it
    detaches its fetch and shares the result (C13).
* **F5. Ollama's `/api/show` replies never expire while listings keep
  coming.**
  * `listOllamaOnce` emits every reply, cached or fetched (C14).
  * It then stores them all as a fresh entry stamped now (C15, C16).
  * So a reply is never older than the last listing, and is never asked for
    again while listings arrive within ten minutes of each other.
  * 0041-MADR A4 caches each reply "as `LookupModel` does" (C19), that is,
    for `listingMetadataTTL`.
* **F6. Concurrent listings each send their own probes.** `ListModels` probes
  up to six models (`catalog.MaxListed`) with a billed one-word generation
  (0038-MADR D4). The probe cache has no single-flight (C17), so N concurrent
  cold listings send up to 6·N probes.

### How the facts were found (2026-10-20)

On a scratch copy of `cadaad5` (`git archive HEAD`), never in the tree, probe
tests printed what they measured. They ran in each package, with Go 1.27.2 on
an Apple M1 Pro:

* **Decoders:** `chatcompletions.Events`, `generatecontent.Events`,
  `messages.Events` and Gemini's `interactionEvents`, fed scripted event
  streams, with time and `runtime.MemStats.TotalAlloc` measured around each
  call and panics recovered for the report.
* **Through providers:**
  * a ChatGPT-session OpenAI provider's `Generate` under `WithRetry`;
  * Claude's `Stream` under `WithStreamRetry`;
  * Hugging Face's public `Stream`;
  * each against an `httptest` server counting requests.
* **Catalog:** `catalog.List` for OpenAI and for Ollama against counting
  servers, each with its own `catalog.Cache`. The Ollama probe moved the
  cache's clock (`listingNow`).
* **Probes:** OpenAI's `ListModels` against a server that lists seven models
  and answers each probe after 50 ms.

The probe files were scratch files, not kept. What they printed is below.

### What was measured

* **M1. Quadratic accumulation (F1).**

  | Fragments | Chat Completions `reasoning_details` | generateContent text parts | Chat Completions text deltas (control) |
  | ---: | :--- | :--- | :--- |
  | 2,000 | 6.2 ms, 10 MiB | 3.9 ms, 8.6 MiB | 1.5 ms, 0.6 MiB |
  | 4,000 | 14.2 ms, 36 MiB | 10.4 ms, 33 MiB | 2.9 ms, 1.1 MiB |
  | 8,000 | 41.8 ms, 135 MiB | 27.2 ms, 131 MiB | 6.2 ms, 2.1 MiB |
  | 16,000 | 96.4 ms, 540 MiB | 77.8 ms, 531 MiB | 12.0 ms, 4.1 MiB |

  Each doubling quadruples the bytes allocated on the two quadratic paths, and
  doubles them on the control. A fragment here is four bytes of text. A
  reasoning model that streams its reasoning a token at a time sends one per
  token, as the 0031 fixture does (`events_0031b_test.go`, `"text":"thi"`, then
  `"nk"`).
* **M2. Unchecked indexes (F2).** Each decoder, given one event with index
  `-1`, panicked with `runtime error: index out of range [-1]`:
  * Chat Completions' `tool_calls` and `reasoning_details`;
  * Messages' `content_block_delta`;
  * Gemini Interactions' `step.delta`.

  One 126-byte Chat Completions event with tool-call index 1,000,000
  allocated 505 MiB in 173 ms.
* **M3. The panic reaches the caller.** Hugging Face's public `Stream`, given
  the `-1` event, panicked the ranging goroutine. The module's only `recover()`
  outside tests is in `llmtest` (`llmtest.go:278`).
* **M4. In-stream failures retried past the once-only rule (F3).**
  * **`Generate` under `WithRetry`:** a ChatGPT-session provider's
    `Generate`, whose reply is always an event stream, got a delta, then
    `response.failed` with `server_error`, on every request. It sent 3
    requests at `MaxAttempts` 3 and 5 at `MaxAttempts` 5. The final error
    carried no after-reply mark. The once-only rule allows 2.
  * **`Stream` under `WithStreamRetry`:** Claude's `Stream` sent 1 request
    and passed the error through. The failure came after the first event, so
    0051-MADR D2 holds there.
* **M5. The listing single-flight (F4).** OpenAI listings against a server
  answering after 100 ms:

  | Case | Requests | Wall time |
  | :--- | ---: | ---: |
  | 5 concurrent `List`, each reply 503 | 5 | 510 ms |
  | 5 concurrent `List`, each reply 401 | 5 | 510 ms |
  | a claimant cancelled at 30 ms, then a waiter | 2 | waiter 130 ms |

  With the result shared, the first two rows would be 1 request in about
  100 ms.
* **M6. Ollama's show cache (F5).** Five listings, nine minutes apart on the
  cache's clock: `/api/tags` was asked 5 times, `/api/show` once. Under a
  ten-minute life per reply, `/api/show` is asked at 0, 18 and 36 minutes:
  3 times.
* **M7. Probes from concurrent listings (F6).**
  * **Cold:** five concurrent cold `ListModels` sent 30 probe generations,
    where one shared run sends 6. A sixth, warm call sent none.
  * **Every probe failing:** each of three listings sent 6 more. This is
    0028's decision that a result with no model is not cached (C18, C20).
* **M8. Ollama's listing carries a digest.** In Ollama's API reference, each
  `/api/tags` entry carries `name`, `model`, `modified_at`, `size`, `digest`
  and `details`. The reference does not say what `digest` identifies. The
  module decodes only `name` and `capabilities` (C16).
* **M9. Every wire's stream goes through one reader.** `wire.ReadEvents` is
  called by Responses, Chat Completions, Messages, generateContent and Gemini
  Interactions, and by nothing else. A mark applied there reaches every
  decoder.

### Code claims

<!-- claims: self=cadaad5 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | Chat Completions joins a `reasoning_details` string to the one before | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:727` | `next = prev + next` |
| C2 | generateContent joins text parts with `+=` | `llmprovider/internal/wire/generatecontent/generatecontent.go:339` | `parts[n-1].Text += part.Text` |
| C3 | A `reasoning_details` index, taken from a float, unchecked | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:706` | `index = int(f)` |
| C4 | A tool-call index grows the slice, unchecked | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:656` | `for len(s.calls) <= tc.Index {` |
| C5 | A Messages content-block index, likewise | `llmprovider/internal/wire/messages/messages.go:410` | `for len(m.Content) <= i {` |
| C6 | A Gemini Interactions step index, likewise | `llmprovider/providers/gemini/interactions.go:308` | `for len(s.raw.Steps) <= index {` |
| C7 | The reader returns a decoder's failure as it is | `llmprovider/internal/wire/sse.go:58` | `if stop, err := next(); stop` |
| C8 | It marks its own read failure | `llmprovider/internal/wire/sse.go:74` | `transport.AfterReply(err))` |
| C9 | `WithRetry` promises the once-only rule | `llmprovider/retry.go:69` | `A failure while reading a reply the service had begun to send` |
| C10 | The rule keys on the mark | `llmprovider/retry.go:163` | `if transport.IsAfterReply(err) {` |
| C11 | A waiter is handed a bare channel | `llmprovider/catalog/discovery.go:364` | `done, wait := cfg.store.claimListing(key)` |
| C12 | The claimant fetches on its own context | `llmprovider/catalog/discovery.go:383` | `f, failure := fetchListing(ctx, id, token, cfg)` |
| C13 | The metadata fetch is detached, the precedent | `llmprovider/catalog/model_metadata.go:490` | `fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), metadataFetchTimeout)` |
| C14 | Ollama emits cached replies again | `llmprovider/catalog/discovery.go:843` | `cfg.emit(show)` |
| C15 | Then stores them all, not merged | `llmprovider/catalog/discovery.go:347` | `cfg.store.storeListing(listingKeyFor(llmprovider.ProviderOllama, cfg, token), models, false)` |
| C16 | `/api/tags` is decoded for name and capabilities only | `llmprovider/catalog/discovery.go:856` | `Capabilities []string` |
| C17 | The probe runs with no flight shared | `llmprovider/internal/transport/probe.go:170` | `healthy := probeHead(ctx, preferred, limit, generate)` |
| C18 | Only a result with a model is cached | `llmprovider/internal/transport/probe.go:171` | `if len(healthy) > 0 {` |
| C19 | 0041 A4's life for a show reply | `docs/decisions/0041-MADR-decide-per-model-behaviour-from-metadata.md:837` | `` and caches each reply as `LookupModel` does `` |
| C20 | 0028's test of the empty result | `docs/decisions/0028-PLAN-heuristics-and-performance-from-the-research-pass.md:1807` | `TestProbeCache_EmptyResultNotCached` |
| C21 | The store's one-at-a-time claims, today | `llmprovider/catalog/listing_cache.go:149` | `ch := make(chan struct{})` |
| C22 | The probe's output limit, recorded | `llmprovider/providers/claude/claude.go:126` | `probeMaxTokens = 8192` |

## Decision Drivers

* **No billed request is sent twice by a heuristic** (0028-MADR's driver for
  D-H1). F3 and F6 both send billed requests that a rule exists to prevent.
* **A reply from the network never panics the caller.** The module made
  errors of its panics in 0049 and 0050.
* **One pattern per problem, already in the module:**
  * `strings.Builder` for accumulation, as the decoders' text deltas use;
  * the after-reply mark at the one shared reader, as its read failure uses;
  * the metadata document's detached, shared single-flight for every
    single-flight;
  * the standard library only, as `AGENTS.md` requires. That rules out
    `golang.org/x/sync/singleflight`, whose shape this follows.
* **The wire does not change** (R45), **and no exported name does** (R48).
* **A recorded decision is amended in the open, not worked around.** F3
  touches 0021-MADR D1, F5 0041-MADR A4, and F6 0028-MADR D-A3.

## Considered Options

* **A. Fix each defect at its root with the module's own patterns:** builders
  (F1), a bound on stream indexes (F2), the mark at the shared reader (F3),
  the metadata cache's single-flight shape for listings and probes (F4, F6),
  and a life per Ollama reply (F5).
* **B. Recover panics at the stream boundary,** and fix only F1, F3 and F4.
* **C. Fix the correctness defects (F2, F3, F5) now, and defer the
  performance ones (F1, F4, F6)** to the optimization work in the 0054
  report.
* **D. Add `golang.org/x/sync`** for `singleflight`, under a dependency
  MADR.

## Decision Outcome

Chosen option: **"A. Fix each defect at its root with the module's own
patterns"**, because each fix applies a pattern the module already uses where
the defect sits. Together they turn a panic, a quadratic copy, two billed
re-sends and a stale cache into tested behaviour, without a new dependency, an
exported name or a change on the wire.

### D1. Accumulate streamed text in builders (F1)

* **Chat Completions' `reasoning_details`:** each entry, by index, keeps a
  `strings.Builder` for each string a stream splits (`text`, `summary`,
  `data`, `signature`), beside its other fields, which keep their first value
  as today. At `[DONE]` the builders' strings take their keys and the entry
  is marshalled as today. Map keys marshal sorted, so the bytes replayed are
  unchanged.
* **generateContent:** the trailing text or thought part accumulates in a
  builder, written to the part when another part arrives or the reply is
  built. The merge rule of `appendPart`'s doc (text joins text, thought
  joins thought, a signature is kept wherever it arrives) is unchanged.
* **Gemini Interactions' `thought_signature`** (`interactions.go:326`,
  `step.Signature += …`) is the same shape and takes the same builder. It
  was not measured: a signature usually comes in one delta.
* **Confirmation:** M1's workloads grow linearly, by a benchmark per wire
  (`BenchmarkEvents_*`, which 0054-REPORT also asks for).

### D2. Bound every index a stream sends (F2)

* **The rule:** a streamed index outside `0 ≤ i < maxStreamIndex` ends the
  stream with an error matching `ErrIncomplete`, naming the field and the
  index, as an event over `EventLimit` does (`sse.go:33`). It is sent whole,
  and asking again buys the same.
* **The bound:** `maxStreamIndex` is 1024 (Q2), one unexported constant in
  `internal/wire`, with one helper used at C3 to C6. Gaps below the bound
  are allowed, as today.
* **Why a bound and not a `recover`:** a recover at the stream boundary
  (option B) would hide every other panic, as a bug report's stack trace,
  behind an error. It would also do nothing for a large index, which
  allocates without panicking (M2).

### D3. A failure event inside a 200 stream is retried at most once (F3)

* **The mark at the shared reader:** `wire.ReadEvents` marks a dispatch
  error with `transport.AfterReply`, as it already marks its own read failure
  and early end (C8). The mark reaches every wire (M9).
* **Its effect:**
  * `WithRetry` retries such a failure at most once per `Generate` call;
  * `WithStreamRetry` retries one before the first event at most once per
    `Stream` call (0051-MADR D2), and passes one after it through, as
    today;
  * classification is unchanged (0028-MADR D-H8), and so is every error's
    text and kind; `errors.Is` and `errors.As` see through the mark.
* **Amends 0021-MADR D1:** its "What counts" gains "a failure event the
  service sends inside a 200 stream". The doc of `WithRetry` (C9) already
  says so; the code now does it. `ClassifyStreamFailure`, exported, is
  unchanged: the mark is the reader's, not the classifier's.
* **Not chosen: marking only failures after a yielded event.** That would
  keep `MaxAttempts` for an in-200 error that arrives before any content,
  such as a gateway's rate limit. Two reasons not to:
  * by then the service has accepted the request and begun its reply, as
    Messages' `message_start` shows (M4), and the rule is about what may have
    been billed;
  * one place, the reader, cannot see which events a decoder yielded.

### D4. One listing fetch per key, detached, its result shared (F4)

* **The shape:** `claimListing` hands out a call, not a bare channel. The
  call carries its own done channel and the fetch's result: the listing, its
  models, when it was read, and its failure.
* **The fetch** runs on the store's `listingFetching` group, on
  `context.WithTimeout(context.WithoutCancel(ctx), modelListingTimeout)`, as
  the metadata fetch does (C13). A claimant's cancellation ends only that
  caller's wait.
* **Every caller,** the claimant included, waits on the call's done channel
  or its own context. Each receives the same result, a failure included. A
  success is stored as today. A failure is not, so the next listing after the
  flight fetches again; caching failures is 0054-REPORT's idea, not this
  record's.
* **Unchanged:**
  * the key (0041-MADR D6, A6);
  * the last good listing on a transient failure;
  * the renewal of a refused credential, which lists again under a new key.

### D5. Each Ollama reply keeps its own age, and its model's digest (F5)

* **The life:** a cached `/api/show` reply is used while it is younger than
  `listingMetadataTTL` (0041-MADR A4). It records the time it was fetched,
  and storing a listing no longer resets that time for replies it re-emits.
* **The digest:** `/api/tags`' `digest` is decoded (M8), and a reply records
  the digest it was fetched for. A listed model whose digest differs from its
  reply's is asked again at once, so a model pulled again within the ten
  minutes is not described by its old reply.
  * The digest is compared as an opaque string, because Ollama does not
    define it.
  * When either digest is absent, the age alone decides.
* **Unchanged:** a failed reply is not cached, and is asked again on the next
  listing (0042-MADR D1). The concurrency of four (`ollamaShowConcurrency`)
  is unchanged too.

### D6. One probe run per key at a time, its result shared (F6)

* **The single-flight:** `transport.ProbeGenerateHealthCached` runs one
  `probeHead` per `ProbeKey` at a time. Concurrent callers wait for it and
  receive its healthy list, empty included, with their own tails appended as
  today.
* **Detached:** the run's probes are detached from the claimant's context,
  each still bounded by `probeTimeout`, so a cancelled claimant does not turn
  every waiter's result into "nothing answered". The probes of a cancelled
  caller then finish. At most six one-word generations, in the background.
* **The empty result** follows the owner's Q1:
  * kept as 0028-MADR D-A3 has it, not cached past the run (C18, C20);
  * or remembered for one minute, as a failed metadata fetch is
    (`modelMetadataRetryAfter`). The listing shown is the same either way
    (no probe answered, so the listing is not filtered). That would amend
    D-A3, and `TestProbeCache_EmptyResultNotCached` with it.
* **The output limit stays.** `probeMaxTokens` (C22) caps the output; the
  answer, not the cap, is billed. 0015-MADR D1 kept it so the probe request
  would be unchanged, and a reasoning model may spend tokens before it says
  "Hello". It is not a defect.

### D7. Tests, docs and the release

* **Tests:** each finding gets a test that fails on today's code. Each is
  then seen to fail on a planted return of the defect, on a scratch clone.
  * F1: a linear-growth benchmark per wire, plus a byte-for-byte replay test
    of `reasoning_details`.
  * F2: each decoder, given index `-1` and index `maxStreamIndex`, returns
    `ErrIncomplete` and does not panic. So does one provider's public
    `Stream`.
  * F3: M4's `Generate` sends 2 requests at `MaxAttempts` 3 and 5, and its
    error matches `ErrProviderUnavailable` as before.
  * F4: M5's rows give 1, 1 and 1 requests, and the cancelled claimant's
    waiter succeeds on the first fetch.
  * F5: M6's clock gives 3 `/api/show` requests, and a changed digest is
    asked at once.
  * F6: M7's cold row gives 6 probe generations.
* **Docs:** `ReadEvents`' doc names the mark on a dispatch error. Each
  function whose errors change names them (R52); `make docs-check` stays at
  0. `docs/architecture.md`'s listing-cache and streaming sections say what
  D3 to D6 change.
* **Release:** `v1.19.1`, a patch release, since no exported name changes
  (Q3). `CHANGELOG.md`'s Fixed lists each finding.

### Consequences

* Good, because a malformed stream index is an error, not a crash of the
  caller (F2), and a long reasoning stream costs linear time and memory (F1).
* Good, because a failure inside a stream is retried once, as the doc already
  says, and not up to `MaxAttempts` billed attempts (F3).
* Good, because N concurrent listings or probes for one key cost one fetch or
  one run (F4, F6), and a cancelled caller no longer restarts the work for the
  others.
* Good, because an Ollama model's caveats follow its reply's age and its
  digest (F5).
* Neutral, because an in-200 error that arrives before any content, such as a
  gateway's rate limit, is now retried once rather than up to `MaxAttempts`.
  That is the once-only rule applied as written.
* Bad, because a detached listing fetch or probe run keeps running for a
  caller that has gone, up to its own bound (10 s, or 5 s a probe). The
  metadata fetch already does.
* Bad, because the stream-index bound is a new limit a pathological but
  valid reply could exceed. 1024 content blocks or tool calls in one reply is
  far beyond any measured, and the error names the index.

### Confirmation

* D7's tests pass, and each failed on its planted break.
* M1 to M7, run again unchanged on the result:
  * no panic;
  * linear growth;
  * 2 requests at `MaxAttempts` 3 and 5;
  * 1 listing request per key;
  * 3 `/api/show` requests;
  * 6 cold probes.
* `tools/apidiff.py v1.19.0` reports no change; G-wire's goldens do not
  change; `make docs-check` reports 0.

## Pros and Cons of the Options

### A. Fix each defect at its root

* Good, because every fix uses a pattern already in the module, where the
  defect is.
* Good, because each is small, separately testable and separately
  revertible.
* Neutral, because it amends three recorded decisions (0021 D1, 0041 A4, and
  0028 D-A3 if Q1 says so), in the open.

### B. Recover panics at the stream boundary

* Good, because one deferred `recover` covers every decoder.
* Bad, because it turns every future bug in a decoder into an error with no
  stack, and does nothing for M2's 505 MiB allocation.
* Bad, because Go's practice is to return errors for bad input and keep
  panics for bugs. A panic on network input is the bug to fix.

### C. Correctness now, performance later

* Good, because the smallest change ships first.
* Bad, because F1 is also a memory hazard (540 MiB at 16,000 fragments, M1),
  and F4 and F6 send requests, some billed, that the cache exists to prevent.

### D. Add `golang.org/x/sync`

* Good, because `singleflight` is the community's standard shape.
* Bad, because `AGENTS.md` allows the standard library and `golang.org/x/term`
  only. Any other module needs its own MADR, and the module's own metadata
  single-flight (C13) already has the shape, detached as `singleflight` is
  not.

## Owner questions

* **Q1 (D6). The empty probe result.** Offered: keep 0028-MADR D-A3, so a
  run in which no probe answered is not cached past the run (recommended for
  this record, which fixes defects and decides no new behaviour); or remember
  it for one minute, as a failed metadata fetch is, saving six billed probes
  per listing while every probe fails, with the same listing shown.
* **Q2 (D2). The stream-index bound.** Offered: 1024 (recommended); or
  4096; or reject only a negative index, which leaves M2's allocation.
* **Q3 (D7). The release.** Offered: `v1.19.1` (recommended), a patch; or
  `v1.20.0`, if Q1's one-minute memory is taken as a behaviour change.

## More Information

* **Not in this record:** the analysis's sixth finding, that a 429's wait
  takes the longest reset header rather than the reset of the exhausted
  dimension, is a heuristic, not a defect. It is idea 6.1 in
  0054-REPORT-heuristics-and-performance-opportunities.md, with the other
  optimization ideas.
* **Records amended once this is accepted:** 0021-MADR-harden-and-tune-after-the-v1-1-review.md
  D1 (D3); 0041-MADR-decide-per-model-behaviour-from-metadata.md A4 (D5);
  0028-MADR-heuristics-and-performance-from-the-research-pass.md D-A3, if Q1
  takes the one-minute memory (D6).
* The PLAN, `0053-PLAN-stream-retry-and-cache-defects.md`, follows once
  this record is reviewed.
