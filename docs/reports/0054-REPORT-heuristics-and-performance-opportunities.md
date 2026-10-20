---
status: observation
date: 2026-10-20
subject: where the module's heuristics and algorithms could be improved for efficiency, performance, resilience and result quality, with prior art from similar projects
examines: "the tree at cadaad5 (v1.19.0 is 87b6c9c): llmprovider/catalog, llmprovider/retry.go, llmprovider/internal/transport, llmprovider/api_error.go, llmprovider/auth, llmprovider/internal/wire, internal/redact; the existing benchmarks; public documentation of similar projects"
---

<!-- markdownlint-disable MD013 -->

# Heuristics and Performance Opportunities, 2026-10-20

This report records findings and ideas. It decides nothing. Each idea that is
taken up needs its own MADR and PLAN, and several would amend a recorded
decision, which each names.

The analysis behind it, on 2026-10-20, was read-only:

* **Surveys:** four, in parallel:
  * the catalog's ranking, matching and caches;
  * retry, transport, error classification and tokens;
  * streaming, encoding and redaction;
  * a web survey of similar projects.
* **Measurements:** the existing benchmarks, run on an Apple M1 Pro with Go
  1.27.2.
* **Spot checks:** of the code claims the ideas rest on.

It also found defects. Five, and a sixth found while proving them, are
decided in
[0053-MADR-stream-retry-and-cache-defects.md](../decisions/0053-MADR-stream-retry-and-cache-defects.md),
not here.

Every idea can be built with the standard library, as `AGENTS.md` requires.
Ideas that would need data from outside the module, such as a quality index,
take it through an interface the caller fills.

## The baseline

The existing benchmarks, `-benchtime=200ms`. Where a benchmark was run more
than once, its range is shown.

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| redact `Redact_16KiBTokens` | 1,334,000–1,474,000 | ~101,900 | 13 |
| redact `Redact_Clean4KiB` | 26,700–27,100 | 0 | 0 |
| llmprovider `Classify_2KiBOverflow` | ~174,700 | ~19,100 | 37 |
| llmprovider `Classify_16KiBTokens` | 308,900–315,900 | ~84,000 | 48–49 |
| llmprovider `ClassifyHTTPError_64KiBHTML` | ~118,000 | ~213,000 | 49 |
| responses `ReadStream_Deltas4MiB` | 1,374,000–1,480,000 (3.5–3.8 GB/s) | ~67,000 | 13–14 |
| responses `ReadStream_4MiB` (ChatGPT fixture) | 7,354,000–7,510,000 (~560 MB/s) | ~925,900 | ~7,120 |
| chatcompletions `BodyMarshal` 40 / 100 / 200 | 17,400 / 41,300 / 80,800 | 20 K / 49 K / 99 K | 38 / 68 / 118 |
| messages `History` 40 / 100 / 200 | 48,200 / 119,900 / 234,000 | 105 K / 266 K / 504 K | 167 / 387 / 744 |
| responses `History` 40 / 100 / 200 | 49,600 / 121,500 / 240,000 | 103 K / 247 K / 474 K | 90 / 199 / 378 |
| generatecontent `History` 40 / 100 / 200 | 41,100 / 100,800 / 201,000 | 93 K / 240 K / 451 K | 65 / 131 / 236 |
| catalog `DecodeModelMetadata` (5.8 MB) | 8,260,000–8,460,000 | ~6,000,000 | 37–40 |
| catalog `DecodeModelMetadataAllSections` | 28,600,000–29,800,000 | ~13,430,000 | ~159,000 |

**Not measured at all:**

* `catalog.List`, cold or warm;
* the ranking, `menuOf`, `Search`;
* `CachedFacts`, which runs on every generation request;
* every wire's `Events`, the path `Stream` takes. `ReadStream`, which the
  benchmarks measure, skips text deltas; `Events` decodes every one;
* deferral;
* `RemapCallIDs`.

## 1. Model ranking and curation (`llmprovider/catalog`)

**Today:**

* `List` builds the menu with `menuOf` (`menu.go:125`). Uncaveated ids go to
  a curator, which picks a band of `MaxListed` = 6. Everything after the band
  is ordered newest first (`menu.go:78`).
* The curators rank with `rankRecommended` (`model_ranking.go:336`) in three
  orders:
  * **Utility** (`:227`): a quality signal, then small models first, then
    `0.8·cost/maxCost + 0.2·age/maxAge` (`:222`), normalised by the maximum
    over the eligible set (`:193`). 0009-MADR §4 chose one scale for both
    tiers.
  * **Capable** (`:239`): signal, bench, preferred index, not-small, newer,
    costlier, id.
  * **Agent** (`agentOrder`, `:255`): Capable's six, by release quarter, then
    the cached-turn price (`model_profile.go:74`, 0041-MADR D5).
* Constants (`:24-34`): `maxRankAgeDays` 540, `minRankContext` 32768,
  `minRankExpiryDays` 30, `maxPerRankGroup` 2.

**What was found:**

* **Cost barely counts.** On the Kilo snapshot (205 eligible models, of
  them 106 small), one model at $210/M input+output sets `maxCost`. Among
  the small models, `0.8·cost/max` then spans 0.0004–0.011 while
  `0.2·age/max` spans 0.0016–0.20. Age outweighs cost about 18 times, so
  utility is in practice "the newest small model".
* **Three cost definitions.** "Small" uses a 3:1 input:output blend
  (`:126`), utility uses input+output (`model_metadata.go:689`), and agent
  uses the cached turn.
* **Capable is "newest first".** Age has day resolution, so the cost key is
  rarely reached. 0041-MADR §5 counted the pairs that reach it: 13 of 276 for
  OpenAI, 3 of 136 for Claude, 0 of 78 for Together.
* **No hysteresis.** 0041-MADR D8 put it out of scope, so a small change in
  the catalog can reorder the band.

**Ideas:**

1. **1.1 Normalise by rank, and use one cost.**
   * Normalise cost and age by percentile rank, or by log-cost clipped at the
     90th percentile, instead of by the maximum, so that one outlier cannot
     flatten a criterion.
   * Use one cost everywhere: the 3:1 input:output blend, which Artificial
     Analysis also uses for its blended price
     ([artificialanalysis.ai/models](https://artificialanalysis.ai/models)).
   * Amends 0009-MADR §4.
2. **1.2 Pareto filter first, then a lexicographic order with tolerance.**
   * Drop models dominated on (quality proxy, cost, recency, context). In two
     dimensions, Kung's skyline sort is O(n log n); in more, an O(n²)
     dominance check is fine at a few hundred models.
   * Order what is left by workload, treating values within a tolerance as
     ties: age by quarter, as `agentOrder` already does; cost within 10 %.
   * This avoids TOPSIS's known rank reversal when a model joins the catalog,
     and keeps the band explainable.
   * Prior art: the Pareto analysis in FutureSearch's Deep Research Bench
     ([futuresearch.ai](https://futuresearch.ai/docs/case-studies/deep-research-bench-pareto-analysis)).
   * Amends 0041-MADR §5's orders.
3. **1.3 A sticky band, and a quality signal the caller supplies.**
   * A model keeps its place in the band unless a challenger beats it by a
     margin. This is the anti-flap idea of Envoy's outlier ejection
     ([Envoy](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/upstream/outlier)).
   * Take a quality index through the existing `catalog.Layer` interface.
     OpenRouter's `/api/v1/models` already carries
     `benchmarks.artificial_analysis.intelligence_index`, on 287 of 468 models
     when surveyed.
   * Treat score gaps inside the index's confidence interval as ties.
     LMArena reports 95 % intervals more than 30 Elo wide for models with few
     votes ([news.lmarena.ai/arena-rank](https://news.lmarena.ai/arena-rank)),
     and an ICML 2026 study shows that perturbations under 1 % can flip its
     top rank ([icml.cc](https://icml.cc/virtual/2026/68325)).
   * Amends 0041-MADR D8.

## 2. Model search and identity

**Today:**

* **Search** (`model_matcher.go:52`) works in score tiers: exact, id
  prefix, substring, separator-free substring, display name, all tokens. It
  falls back to a greedy leftmost subsequence on the id (`:207`). Ties keep
  the listing's order (0021-MADR C11).
* **Label** (`:892`) is an exact lookup in a table of 25 static ids.
* **Version parsing** in the name-rule fallbacks hard-codes versions
  (`models_catalog.go:504-633`). `parseGeneration` (`:691`) needs a one-digit
  major in its own token, so `qwen3.8` and `gpt-10` do not parse.
* **Ids are not normalised across providers.** `canonical_model_id` fills
  only a missing cache-read price (`metadata_document.go:89-134`).
* 0007-MADR §3 rejected edit distance on cost. 0021-MADR C11 measured
  `Search` at about 0.45 ms over 400 ids.

**Ideas:**

1. **2.1 A canonical-id resolver.**
   * Try, in order: the exact id; an alias table; a canonical form; a key of
     the id's tokens, sorted.
   * The canonical form lowercases the id, then strips the provider prefix,
     a `:variant`, a `-YYYYMMDD` or `@YYYYMMDD` date, and `-latest`.
   * OpenRouter's own endpoint data names one model both
     `anthropic/claude-4.5-sonnet-20250929` and `anthropic/claude-sonnet-4.5`:
     words reordered, separators differing, a date suffix.
   * LiteLLM resolves aliases to one record at load time
     ([docs.litellm.ai](https://docs.litellm.ai/docs/provider_registration/add_model_pricing)).
   * The resolver would also retire the hard-coded version tables, with a
     parser that takes multi-digit majors and dotted minors.
2. **2.2 Search the metadata's display names, and score closeness within a
   tier.**
   * Search models.dev's `Name` and `Family`, which `Menu` entries already
     carry, so a gateway's id finds its model.
   * Within a tier, score token-boundary hits, the length ratio and recency,
     so `gpt-5` ranks `gpt-5` above `gpt-5.1-codex-mini` rather than by
     listing order.
3. **2.3 Best-alignment fuzzy matching.**
   * Replace the greedy subsequence with fzf's v2 scoring: dynamic
     programming with boundary and camel-case bonuses, O(n·m) for a short
     query.
   * Build the query's transforms once, rather than per candidate as today.
   * Tolerate one transposition for queries of four runes or more.
   * The 0.45 ms budget over 400 ids leaves room. Revisits 0007-MADR §3.

## 3. The catalog's caches and listing fetch

**Today:**

* **The listing cache** (`listing_cache.go`) is keyed by provider, base URL,
  Kilo organization and credential fingerprint. It is fresh for 10 minutes,
  and serves a last good listing up to 24 hours old on a transient failure.
  It is never swept.
* **`CachedFacts`** reads a listing of any age (`lookup.go:163`).
* **Entry size:** one cached Kilo key holds about 391 `ModelInfo` values of
  about 760 bytes each, of which 400 are `sources [25]string`. That is
  0.5–0.7 MB per credential.
* **Shared listings stored per credential.** Kilo, Hugging Face and OpenCode
  publish the same listing to every key, but each key keeps its own copy.
* **Decoding:** `decodeLimited` (`discovery.go:97`) reads a page of up to 8 MiB
  whole, then decodes it with v1 `json.Unmarshal`. The metadata document
  streams through json/v2 instead.
* **Kilo:** `KiloModelCapabilities` (`discovery.go:1379`) downloads the
  Kilo catalog, about 487 KB, on every call.
* **Defects decided elsewhere:** the single-flight that shares nothing, and
  Ollama's ageless show replies, are 0053-MADR's F4 and F5.

**Ideas:**

1. **3.1 Stale-while-revalidate listings, and a short negative cache.**
   * After the 10 minutes, serve the cached listing while one background
     fetch revalidates it. Remember a failure for 30–60 s, so a 401 is not
     asked again by every caller.
   * This is RFC 5861's `stale-while-revalidate` and `stale-if-error`
     ([rfc-editor.org/rfc/rfc5861](https://www.rfc-editor.org/rfc/rfc5861)).
     OpenRouter serves its own models API with `max-age=120,
     stale-while-revalidate=3600, stale-if-error=3600`.
   * Builds on 0053-MADR D4's shared flight. Amends 0041-MADR D6.
2. **3.2 A bounded store, with one copy of each public listing.**
   * Sweep entries older than the last-good window, and add a size cap.
   * Store a public listing once by its content hash, rather than once per
     credential.
   * Give `CachedFacts` a maximum age.
   * Shrink `ModelInfo.sources` to byte indexes into a layer-name table,
     saving about 375 of its 760 bytes.
   * Read the facts on the request path from an `atomic.Pointer` snapshot,
     not under the store's mutexes.
3. **3.3 Stream every listing decode, and serve Kilo's capabilities from the
   cache.**
   * Decode listing pages with the json/v2 streaming decoder the metadata
     path uses (0028-MADR D-A1), instead of a whole-page read and a v1
     unmarshal.
   * Answer `KiloModelCapabilities` from the cached listing.
   * Add `BenchmarkList` for a warm and a cold listing first.

## 4. Retry and resilience (`llmprovider/retry.go`)

**Today:**

* **Each call retries alone.** `WithRetry` and `WithStreamRetry` keep no
  shared budget, circuit breaker or hedging, and no state across calls.
* **No fallback.** The wizard collects `Fallbacks` (`wizard/configure.go:46`)
  and hands them back; nothing in the module uses them.
* **Outages multiply load.** N concurrent callers during an outage each send
  up to `MaxAttempts`.
* **A 429 is learned alone.** A `RetryAfter` one goroutine learns is not
  shared, so its siblings each pay for the same 429.
* **The precedent for shared state:** the only learned state across calls is
  the forced-choice memo (`internal/wire/messages/learn.go`), scoped to
  `catalog.Cache` by 0049-MADR D3.

**Ideas:**

1. **4.1 A shared retry budget per provider, base URL and credential.** A
   token bucket that never blocks a first attempt. Two published designs give
   its constants:
   * **AWS's 2026 retry behaviour:** capacity 500; a transient retry costs 14
     and a throttling retry 5; a successful retry refunds its cost; a
     first-try success adds 1
     ([docs.aws.amazon.com](https://docs.aws.amazon.com/sdkref/latest/guide/feature-retry-behavior.html)).
   * **gRPC's retry throttling:** each failure subtracts 1, each success adds
     a ratio, and retries stop below half the bucket
     ([gRFC A6](https://github.com/grpc/proposal/blob/master/A6-client-retries.md)).

   Finagle and Envoy default to about 20 % of requests
   ([Finagle clients](https://twitter.github.io/finagle/guide/Clients.html)).
   The budget would live in `catalog.Cache`, as the forced-choice memo does.
2. **4.2 A shared cooldown, and pacing from the rate-limit headers.**
   * **Cooldown:** a 429's `RetryAfter` becomes "blocked until" for every
     call with the same key. Each waits, or fails at once when the wait is
     over `MaxDelay` or past its deadline.
   * **Pacing:** parse the `*-remaining-*` headers that successful replies
     already carry, into one rate-limit state, so callers can pace before a
     429. No code reads them today.
     * OpenAI's resets are Go durations
       ([developers.openai.com](https://developers.openai.com/api/docs/guides/rate-limits)).
     * Anthropic's are RFC 3339 times, and its rate limiting is a continuously
       refilled token bucket
       ([platform.claude.com](https://platform.claude.com/docs/en/api/rate-limits)).
   * **Concurrency:** AIMD on 429s. Netflix's concurrency-limits recommends
     AIMD for clients
     ([github.com/Netflix/concurrency-limits](https://github.com/Netflix/concurrency-limits)).
     Latency-based Vegas fits poorly, because an LLM call's latency is mostly
     its output length.
3. **4.3 An explicit fallback wrapper, and an opt-in way to continue a
   stream.**
   * **`WithFallback(p, alternates...)`**, triggered by error kind:
     * exhausted quota;
     * not permitted;
     * still unavailable after retries;
     * a wait over `MaxDelay`;
     * context overflow, sent to a model with a larger context.

     An alternate that keeps failing is ejected for longer each time:
     Envoy's `base_ejection_time × times ejected`.
   * **Prior art:** LiteLLM's cooldowns (3 failures a minute, 5 s), context-
     window fallbacks and per-error-class retry counts
     ([docs.litellm.ai/docs/routing](https://docs.litellm.ai/docs/routing)),
     and Portkey's nested fallback and load-balance strategies
     ([portkey.ai](https://portkey.ai/docs/product/ai-gateway/fallbacks)).
   * **Continuing a stream:** an opt-in `WithStreamRetry` mode that, after a
     failure mid-stream, sends the request again with the text already
     delivered as a trailing assistant message. LiteLLM calls this
     mid-stream fallback, best effort.
   * **Constraint:** the order must be the caller's. 0004-PLAN §11 says
     "no auto-fallback (silently changing wire format turns a loud failure
     into a wrong answer)". Continuing a stream would amend 0051-MADR D2.

## 5. Transport and connections (`llmprovider/internal/transport`)

**Today:**

* **Configuration** (`transport.go:70-103`):

  | Setting | Value |
  | :--- | :--- |
  | Dial timeout | 30 s |
  | TLS handshake timeout | 10 s |
  | `ResponseHeaderTimeout` | 300 s |
  | `IdleConnTimeout` | 90 s |
  | `MaxIdleConnsPerHost` | 16 |
  | `ForceAttemptHTTP2` | true |

* **Not set:** `HTTP2Config`, so no HTTP/2 health check, since
  `SendPingTimeout`'s zero value means none. Nor `TLSClientConfig` (outside
  tests), so there is no TLS session cache.
* **Why the header timeout is 300 s:** every non-ChatGPT `Generate` is
  unstreamed, and its provider sends no headers until the generation is done
  (0028-MADR H1).
* **Closing bodies:** a body closed before its end loses its HTTP/1.1
  connection; 0028 A8 measured this and did not change it.

**Ideas:**

1. **5.1 HTTP/2 health pings and TLS session resumption.**
   * Set `HTTP2: &http.HTTP2Config{SendPingTimeout: ~30 s, PingTimeout: ~15 s}`
     and `tls.NewLRUClientSessionCache`.
   * A pooled HTTP/2 connection that died silently, after a laptop sleep or a
     NAT change, is written to "successfully" today. It then hangs up to
     300 s, marked after-reply, and gets one retry.
   * 0028 measured 45 ms for a warm handshake and 230 ms for a cold one, and
     the 90 s idle timeout is shorter than a typical agent's thinking time.
2. **5.2 Stream every `Generate` internally, then shorten the header
   timeout.**
   * Ask every provider for an event stream, and collect it as the ChatGPT
     path does. The decoders exist (0031-MADR).
   * Headers then arrive in seconds, the idle timer proves the server is
     alive, and the first event gives a commit point as precise as gRPC's.
   * Revisits 0031-MADR Q3 ("unchanged"); it needs a measurement of the
     billed path first.
3. **5.3 Prewarm, a shorter connect timeout, and keep-alive for HTTP/1.1.**
   * **Prewarm:** an optional `Prewarm(ctx)` that dials and completes the
     handshake with the base URL before the first turn, as LiveKit's plugins
     offer
     ([docs.livekit.io](https://docs.livekit.io/reference/agents-js/classes/plugins_agents_plugin_openai.responses.LLM.html)).
   * **Connect timeout:** 5 s, as in the OpenAI SDK, which 0021-MADR D2
     quotes; today a blackholed host costs about 3 × 30 s.
   * **Keep-alive:** on success only, a bounded drain of the body, so that
     HTTP/1.1 gateways such as Ollama and proxies keep their connections.

## 6. Error classification (`llmprovider/api_error.go`)

**Today:**

* **`RetryAfter`** comes from, in order: `retry-after-ms`; `Retry-After`;
  the body's `resets_at`; Google's `RetryInfo`; and, for a 429 only,
  `transport.RateLimitReset`. That function returns the **longest** of the
  reset headers (`transport.go:250-265`).
* **Exhausted quota** is detected from vendor tables and HTTP 402.
* **A kindless error is retried** when it is a `*url.Error` that is not
  "unsendable". That includes x509 verification errors and DNS NXDOMAIN.
  0028 shortlisted the TLS case as H6 and did not change it.
* **Backoff** uses equal jitter.
* **A 429 with no hint,** as OpenCode Zen and Kilo send, is retried after
  0.5–1 s and 1–2 s. 0028-MADR H5 measured those gaps (0.90 s and 1.42 s)
  as futile. The owner deferred a longer base in its Q3.

**Ideas:**

1. **6.1 Wait for the reset of the exhausted dimension.**
   * Pair each reset header with its `remaining` header, and take the reset
     of the dimension whose `remaining` is 0.
   * When requests are exhausted but tokens are not, the tokens' reset can be
     minutes away. Today that exceeds `MaxDelay`, so the 429 is returned at
     once instead of after the short wait that would succeed.
   * Also read the reset headers on 503 and 529, not only on 429.
   * This is the analysis's sixth finding. It is a heuristic, not a defect,
     and so is not in 0053-MADR.
2. **6.2 Stop retrying what cannot mend.**
   * Treat x509 verification failures, DNS NXDOMAIN, 501 and 505 as terminal.
   * Never retry Anthropic's spend-cap 429, `enforced_spend_limit_reached`.
     It carries no `retry-after`
     ([platform.claude.com](https://platform.claude.com/docs/en/api/rate-limits)).
   * Revisits 0028's H6.
3. **6.3 Backoff by error class, with full jitter.**
   * Give throttling its own base: AWS uses 50 ms for transient errors and
     1,000 ms for throttling, with full jitter
     ([AWS](https://docs.aws.amazon.com/sdkref/latest/guide/feature-retry-behavior.html)).
   * Offer full jitter. AWS's measurements found equal jitter does about the
     same work but takes longer to finish
     ([aws.amazon.com/blogs](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/)).
   * Spread the jitter below the cap, so that server-directed waits near
     `MaxDelay` stop landing at the same moment (`retry.go:301`).
   * Revisits 0028 H5 and its Q3.

## 7. Streaming decoders (`llmprovider/internal/wire`)

**Today:**

* **The reader:** `ReadEvents` reads each stream through a new 64 KiB
  `bufio` buffer. A line longer than the buffer is copied twice.
* **Responses** skips unhandled events by a byte search for their type
  (`responses.go:680`).
* **The other decoders** unmarshal every event into a full struct:
  * Messages decodes each `text_delta` into `streamEvent`, at three string
    allocations or more per delta;
  * Chat Completions re-allocates `id` and `model` on every chunk.
* **Tool-call arguments** arrive whole, by 0031-MADR Q4.
* **Conformance:** the reader handles LF, and CR LF through `readLine`.
  WHATWG's event-stream rules also allow a lone CR as a line end, and a
  leading BOM
  ([html.spec.whatwg.org](https://html.spec.whatwg.org/multipage/server-sent-events.html#event-stream-interpretation)).
  The official openai-go and anthropic-sdk-go use `bufio.Scanner` with its
  default `ScanLines`, which misses both.
* **Defects decided elsewhere:** quadratic accumulation and unchecked
  indexes are 0053-MADR's F1 and F2.

**Ideas:**

1. **7.1 Benchmark the real path, then decode by type.**
   * Add `BenchmarkEvents_*` for the five wires, over 0021's W2 workload.
   * Then read each event's `type` first, as Responses does, and decode each
     delta into a minimal struct with a pooled `jsontext.Decoder`. Its
     `PeekKind`, `ReadToken` and `Reset` suit an event that is read once
     ([pkg.go.dev/encoding/json/jsontext](https://pkg.go.dev/encoding/json/jsontext)).
   * Skip `id` and `model` after the first chunk.
2. **7.2 Pool the reader's buffer, and stop copying long lines twice.**
   * Use a `sync.Pool` for the 64 KiB reader, which is almost all of the
     67 KB per stream in `ReadStream_Deltas4MiB`.
   * Append a long line's `ReadSlice` chunks straight into the event's data.
   * Shrink the data buffer after an oversized event, which today keeps its
     capacity, up to 16 MiB, for the rest of the stream.
3. **7.3 Partial tool-call arguments, and a conformance table.**
   * **Partial arguments:** port the Vercel AI SDK's `fixJson`, an O(n)
     state machine that closes a truncated JSON buffer, to emit partial
     argument objects as an opt-in event. It is about 150 lines of Go
     ([github.com/vercel/ai](https://github.com/vercel/ai/blob/main/packages/ai/src/util/fix-json.ts)).
     Anthropic's `eager_input_streaming` sends unvalidated partial JSON for
     the same purpose
     ([platform.claude.com](https://platform.claude.com/docs/en/agents-and-tools/tool-use/fine-grained-tool-streaming)).
     Amends 0031-MADR Q4.
   * **Conformance table:** add a WHATWG table (lone CR, BOM, an incomplete
     event at end of stream) run against all five decoders.

## 8. Request encoding and prompt caching

**Today:**

* **Bodies:** a `map[string]any` of typed union structs, marshalled with v1
  `encoding/json`, which runs on the v2 engine (0028-MADR D-A6).
* **Schemas are marshalled more than once:**
  * `Capabilities.Check` marshals every tool schema to validate it, and
    throws the bytes away (`contract.go:262`). 0028 measured this as A4: 0.29
    ms and 86 KB per `Generate` with twenty 2 KiB schemas;
  * the wire marshals the schemas again;
  * Grok's dispatch form marshals deferred ones twice more (`deferral.go:148`,
    `responses.go:247`).
* **The whole history is re-encoded every turn,** so over a conversation the
  cost grows with the square of its length (0028-MADR D-A6). The baseline's
  `History` rows show the per-turn cost.
* **Each retry and reauth attempt** re-marshals the body (`post.go:118`).
* **No prompt-caching option.** The module exposes none.

**Ideas:**

1. **8.1 Encode each schema once per request.**
   * Have `validate` produce `json.RawMessage` once and pass it to the
     encoders.
   * Document a `json.RawMessage` schema as the fast path.
   * A memo per provider instance would need an immutability rule for map
     schemas.
2. **8.2 Encode history incrementally.**
   * Memoise each item's encoding across turns. Callers append to the same
     slice, so a turn would encode only its new items.
   * Reuse the encoded body across retry and reauth attempts.
   * Marshal into a pooled buffer; bytes allocated run at about twice the
     output size in the `History` rows.
   * Byte-loop the common case of `RemapCallIDs`, which runs a regular
     expression per id today (`callids.go:82`). That accounts for Messages'
     744 allocations at 200 items, against 378 on Responses.
3. **8.3 Automate prompt caching, and report its tokens.**
   * **Anthropic:** set the top-level automatic `cache_control`. A cache read
     costs 0.1× the input price or less, and the minimum cached prompt is 512
     tokens on the 5.5 models
     ([platform.claude.com](https://platform.claude.com/docs/en/build-with-claude/prompt-caching)).
   * **OpenAI:** derive `prompt_cache_key` from a stable conversation id; a
     read costs 0.1× on GPT-5.6 and later
     ([developers.openai.com](https://developers.openai.com/api/docs/guides/prompt-caching)).
   * **One usage struct:** report `CacheReadTokens` and `CacheWriteTokens`
     across providers, as the Vercel AI SDK normalises them
     ([ai-sdk.dev](https://ai-sdk.dev/providers/ai-sdk-providers/anthropic)).
   * **Prefix order:** keep it stable (tools, then system, then messages),
     so that tool deferral does not throw the cache away.
   * This changes requests on the wire, so it needs its own record (R45).
     For agent loops it is likely the largest real-world gain in this report.

## 9. Redaction (`internal/redact`)

**Today:**

* **The passes:** eleven compiled RE2 expressions, run as nine ordered
  passes behind a prefilter of about 50 literal anchors. Each anchor is
  looked up with its own `IndexByte` scan of an ASCII-lowered copy of the
  text.
* **Where it runs:** on error paths only, capped at 2 KiB of an error body
  (0028-MADR D-A10).
* **The cost:**
  * `reAuth`, which has no literal prefix, accounts for almost all of the
    1.4 ms on 16 KiB, and of the 175 µs per classified overflow error;
  * the anchor scans alone take 26.7 µs on 4 KiB of clean text.
* **The safety net:** `FuzzRedact_NeverPanics`, `FuzzAnchorIn_MatchesContains`,
  and a frozen pre-0028 reference with about 200 equivalence cases.

**Ideas:**

1. **9.1 An exact-value pass first.** A `strings.NewReplacer` over the
   secrets the module actually holds for the call, such as the request's
   token and a static key. It is the cheapest and most precise layer, and
   runs before the patterns. Go practice for redacting logs does the same
   ([arcjet.com](https://arcjet.com/learn/redact-sensitive-data-logs)).
2. **9.2 One multi-pattern prefilter pass, and `reAuth` keyword-first.**
   * Replace the 50 or so anchor scans with one Aho–Corasick or first-byte-
     table pass. TruffleHog measured about a 2× speedup from moving to
     Aho–Corasick
     ([trufflesecurity.com](https://trufflesecurity.com/blog/making-trufflehog-faster-with-aho-corasick)).
   * Apply 0028-MADR D-A10's keyword-first anchoring to `reAuth`: find
     `bearer`, `basic` or `token` followed by whitespace, then match anchored
     there.
   * The fuzz and equivalence tests keep a rewrite safe.
3. **9.3 An entropy fallback for token shapes no rule knows.**
   * Flag base64 above 4.5 bits per character and hex above 3.0, on runs of 20
     characters or more, in error bodies only, with an allowlist for ids.
   * This is the approach of gitleaks
     ([github.com/gitleaks/gitleaks](https://github.com/gitleaks/gitleaks))
     and of the original truffleHog. Nosey Parker's choice of narrow,
     anchored rules without entropy is the counterpoint
     ([github.com/praetorian-inc/noseyparker](https://github.com/praetorian-inc/noseyparker)).
     Its false positives on ids and hashes are why it stays a fallback.

## 10. Tokens and refresh (`llmprovider/auth`, `llmprovider/command_token.go`)

**Today:**

* **`OAuthSession.Token`** refreshes inside a margin of `min(5 min,
  lifetime/2)` (0021-MADR T3). Within it, every caller blocks on the network
  refresh, plus a wait of up to 40 s for another process's lock.
* **The refresh retry:** 3 attempts of 15 s each, backed off from 200 ms
  without jitter, ignoring `Retry-After`. Its worst case, about 45.6 s,
  outlasts the 40 s lock wait, so a waiter can give up during a refresh that
  is still working. 0028 found this (H9) and did not change it.
* **The file lock** polls every 50–150 ms (0028 A7, not changed).
* **`CommandToken`:**
  * runs its command on the first caller's goroutine and context;
  * blocks every caller when its TTL ends;
  * caches no failure, so a failing command runs again on every call.
* **`VendorCLISession`** re-reads and re-parses its file on every call.

**Ideas:**

1. **10.1 Refresh in the background before blocking.**
   * Within a "stale" margin, serve the current token and refresh in the
     background; block only below a floor. The module's 30 s early-refresh
     floor already exists.
   * Google's Java auth library refreshes at 4 minutes and blocks at 3
     ([github.com/googleapis](https://github.com/googleapis/google-auth-library-java/pull/749)).
     MSAL refreshes at half the lifetime and ignores a failure until the next
     call
     ([learn.microsoft.com](https://learn.microsoft.com/en-us/entra/msal/dotnet/advanced/high-availability)).
   * Add jitter: aws-sdk-go-v2's `ExpiryWindowJitterFrac`, or WorkOS's
     refresh at 75 % of the lifetime ± 5 %
     ([workos.com](https://workos.com/blog/oauth-refresh-token-race-condition)).
   * Amends 0021-MADR T2 and T3.
2. **10.2 A refresh retry that honours `Retry-After`, and a lock that
   blocks.**
   * Add jitter and honour `Retry-After` in the refresh retry.
   * Resolve H9 by making the lock wait longer than the worst-case refresh.
   * Replace the polling with a blocking `flock` in a goroutine, behind an
     in-process mutex per path.
3. **10.3 Refresh a `CommandToken` ahead of its TTL, and cache failures.**
   * Run the command again in the background at about 80 % of its TTL, on a
     detached context.
   * Remember a failure, with backoff, so a failing command is not run on
     every call.
   * Cache a `VendorCLISession` by its file's modification time and size.

## Not ideas

* **The probe's 8192-token output limit** (`probeMaxTokens`) costs nothing.
  A probe pays only for what it generates, and a reasoning model may spend
  tokens before it answers. 0015-MADR D1 kept the limit so the probe request
  would be unchanged. See 0053-MADR D6.
* **Semantic response caching** (GPTCache) belongs to the caller. It needs an
  embedding model and a vector store, and its own README warns of false hits
  ([github.com/zilliztech/GPTCache](https://github.com/zilliztech/GPTCache)).
  Provider prompt caching (8.3) is the better lever.
* **Learned routers** (RouteLLM, FrugalGPT, Not Diamond, Martian) need model
  weights or a hosted service
  ([arxiv.org/abs/2406.18665](https://arxiv.org/abs/2406.18665),
  [arxiv.org/abs/2305.05176](https://arxiv.org/abs/2305.05176)).
  FrugalGPT's cascade *mechanism*, with an acceptance function the caller
  supplies, would fit under 4.3.

## Caveats

* **Measurements:** the benchmark figures come from one run of each on one
  machine, an Apple M1 Pro. The ranking figures in section 1 replay the Kilo
  snapshot in `testdata/ranking-2026-10-12`.
* **Formulas reconstructed:** the published pages show some formulas only as
  images, so these were reconstructed: the AWS jitter variants, Google SRE's
  adaptive throttling, and resilience4j's default failure threshold.
* **Secondary or unconfirmed sources:** LiteLLM's latency routing by time to
  first token is known only from mirrored source. The truffleHog entropy
  thresholds come from a fork.
* **Vendor claims, not checked:** the performance figures of the commercial
  routers.
