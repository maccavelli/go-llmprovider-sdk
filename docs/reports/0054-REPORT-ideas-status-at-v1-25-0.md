---
status: observation
date: 2026-11-03
subject: which of the 30 ideas in 0054-REPORT-heuristics-and-performance-opportunities.md the module has taken up by v1.25.0, which of that report's facts still hold, and what it missed
examines: "the tree at d65e9d1 (v1.25.0 is the last tag; the seven commits after it change only tools/, docs/ and AGENTS.md), against the report's own tree, cadaad5; the records from 0053 to 0067"
---

<!-- markdownlint-disable MD013 -->

# Status of the 0054 Ideas at v1.25.0, Audited 2026-11-03

This report records findings. It decides nothing. It audits
[0054-REPORT-heuristics-and-performance-opportunities.md](0054-REPORT-heuristics-and-performance-opportunities.md),
which lists 30 ideas (1.1 to 10.3) for the module's heuristics and
performance and says that each idea taken up needs its own MADR and PLAN.
No record from 0053 to 0067 takes one up whole. The decision that follows
from this audit is
[0068-MADR-five-internal-fixes-from-the-0054-audit.md](../decisions/0068-MADR-five-internal-fixes-from-the-0054-audit.md).

## Method

* **Read-only.** Four parallel read-only reviews, one per group of the
  report's sections (1 to 3; 4, 6 and 10; 5 and 7; 8 and 9), read the code
  at `d65e9d1`, the same files at `cadaad5` (`git show`), and the records
  written since. Each finding below that names a line was then re-read by
  hand at `d65e9d1`; the evidence index at the end pins them, and
  `tools/claims.py` checks it.
* **Probes,** in the session's scratch directory, never in the tree:
  * a Go program that sends a request to a TLS test server whose
    certificate the client does not trust, and to one it trusts under a
    wrong name, and prints the error chain;
  * a Go program that lets a pooled HTTP/2 connection's peer go silent and
    times the next request, with and without the transport's HTTP/2 health
    check;
  * a memory profile of `BenchmarkHistory/items=200` in
    `llmprovider/internal/wire/messages`, on a scratch clone of `d65e9d1`;
  * the redaction benchmarks, on a scratch copy of `internal/redact`
    identical to the tree's.
* **Host:** an Apple M1 Pro, Go 1.27.3. The redaction timings ran with a load
  average near 39 on 10 CPUs and are read as ratios on the same input only.

## Status of each idea

None is done whole. Five are partly done; 25 are open.

| Idea | Status at `d65e9d1` | Evidence and what remains |
| :--- | :--- | :--- |
| 1.1 Normalise by rank, one cost | open | Cost and age are still divided by their maximum (E25). Remains: percentile or clipped log-cost scaling, and one cost definition. |
| 1.2 Pareto filter, then lexicographic order | open | No dominance filter in `llmprovider/catalog`. Outside Kilo there is no quality axis (0041-MADR D5), which weakens it. |
| 1.3 Sticky band, caller's quality signal | open | 0041-MADR D8 still puts both out of scope. |
| 2.1 Canonical-id resolver | partly done | 0061-MADR built canonical ids for Chat Completions servers' models.dev lookups only. Remains: aliases, date and `-latest` suffixes, built-in providers, and a version parser to replace the anchors. |
| 2.2 Search names and families | partly done | 0061-MADR's amendment "Model search, rebuilt" indexes names and families and orders a tier by BM25F. Remains: length or recency inside a tier. |
| 2.3 Best-alignment fuzzy matching | partly done | Same amendment: bounded transposition edits, query transforms built once. Remains: fzf v2's alignment, and per-candidate rebuilds of the needle. |
| 3.1 Stale-while-revalidate listings, failure memory | open | A failed listing is still not stored (E24). 0053-MADR D4 built the shared fetch it would extend; the models.dev document cache is the in-module precedent. |
| 3.2 Bounded store, one copy per listing | open | Nothing in the store is swept; `CachedFacts` reads a listing of any age. |
| 3.3 Stream listing decodes, Kilo from the cache | open | `KiloModelCapabilities` fetches the catalog on every call (E26). |
| 4.1 Shared retry budget | open | Each call retries alone. `WithRetry` takes a provider and a policy, and cannot reach a cache today. |
| 4.2 Shared cooldown and pacing | open | `Retry-After` is read per error only. No 429's headers have been measured beyond Gemini's. |
| 4.3 `WithFallback` | open | Nothing in `llmprovider/` reads the wizard's `Fallbacks` (E9). |
| 5.1 HTTP/2 health check, TLS session cache | open | The transport sets neither (E18). The health-check probe below measures the hazard. |
| 5.2 Stream every `Generate` | open | 300 s header timeout unchanged (E17), and now in `NewHTTPClient`'s exported doc (E28). |
| 5.3 Prewarm, shorter connect timeout, drain | open | Dialer unchanged at 30 s. |
| 6.1 Wait for the exhausted dimension's reset | open | The longest reset still wins. 0053-MADR left it to 0054 as a heuristic (E29). |
| 6.2 Stop retrying what cannot mend | open | A refused certificate, 501 and 505 are retried (E10, E11, E12, P1). DNS failures are now `ErrOffline`, retried by decision (E13, E30). |
| 6.3 Backoff by error class, full jitter | open | Equal jitter unchanged (E31). |
| 7.1 Benchmark the real path, decode by type | partly done | 0053-MADR D1 added linear-growth benchmarks for two of five wires. No type-first decoding. |
| 7.2 Pool the reader's buffer | open | A 64 KiB reader per stream (E19). |
| 7.3 Partial tool-call arguments, conformance table | open | 0031-MADR Q4 stands. |
| 8.1 Encode each schema once | open | Also missed: `CheckDispatch` marshals a deferred schema that `validate` already proved encodes (E22). |
| 8.2 Encode history incrementally | open | See "Corrections", item 8. |
| 8.3 Automate prompt caching, report its tokens | partly done | Every mechanism predates the report (see "Corrections", item 1). Only default-on caching remains, which 0039-MADR Q1 rejected. |
| 9.1 Exact-value pass | open | No value-based pass outside test fixtures. |
| 9.2 Prefilter, `reAuth` keyword-first | open | `reAuth` runs unanchored over the text (E20, E21); only `reKV` is keyword-first (E32). |
| 9.3 Entropy fallback | open | Nothing names entropy in code or records. |
| 10.1 Background OAuth refresh | open | Callers inside the margin wait. |
| 10.2 Refresh retry, blocking lock | open | `lockWait` (E14) is shorter than the longest refresh hold (E15, E16, P3). |
| 10.3 Refresh a `CommandToken` early, remember failures | open | Only a success is stored (E23). |

## Corrections to 0054-REPORT

Each was wrong at `cadaad5`, unless it says it went stale since.

1. **"No prompt-caching option. The module exposes none."** (E1). False when
   written: 0039-MADR, dated 2026-10-13 (E2), added
   `llmprovider.CacheRetention` (E3) before the report's 2026-10-20. Claude,
   Kilo and OpenCode send `cache_control`; OpenAI, Grok, Together and
   OpenCode send `prompt_cache_key`; `Usage` carries `CachedTokens` and
   `CacheWriteTokens` (0036-MADR).
2. **The label table "of 25 static ids"** (E4) has 20, across five providers
   (E5), at `cadaad5` and at `d65e9d1`. `Label` is in `models_catalog.go`
   (E6), not `model_matcher.go`, which had 239 lines at `cadaad5`.
3. **"`qwen3.8` and `gpt-10` do not parse"** (E7). `qwen3.8` parses, wrongly:
   the tokens are `qwen3` and `8`, and the first one-digit token is taken as
   the major (E8), so it reads as version 8. `gemini-10.1-pro` reads as 1.0
   the same way. `gpt-10` is moot: OpenAI's ranker does not parse versions.
4. **2.2's `gpt-5` example** (E33): an exact id already scored first at
   `cadaad5` (`scoreExact`).
5. **1.3's "through the existing `catalog.Layer` interface"** (E34): a layer
   returns `ModelInfo`, which has no quality field, and no ranker reads
   layers.
6. **"Every non-ChatGPT `Generate` is" unstreamed** (E35): an OpenAI plan
   session streamed too, at `cadaad5` and now (E27).
7. **"0028 A8 measured this"** (E36): 0028-MADR lists A8 as shortlisted and
   not chosen, with no measurement.
8. **Messages' 744 allocations at 200 items** (E37) are not mostly
   `RemapCallIDs`. The profile (P4) puts `RemapCallIDs` at 16.6 % of the
   allocated objects, cumulative, and `reflect.unsafe_New` at 50.6 %, every
   one reached from `encoding/json/v2` making an addressable copy of an
   interface value inside `Content.MarshalJSONTo` (E38). Passing the blocks
   by pointer, which one review proposed, would not reach those copies; how
   to remove them is not established.
9. **"Eleven compiled RE2 expressions"** (E39): `redact.go` compiles 14; it is
   11 without the three that validate a match rather than find one.
10. **Stale since:** "the only learned state across calls is" the
    forced-choice memo (E40). 0062-MADR D5 added the memo of refused
    thinking settings (E41).
11. **Stale since:** "x509 verification errors and DNS NXDOMAIN" are
    kindless and retried (E42). x509 still is (P1). A DNS failure is now
    `ErrOffline` (E13), and still retried, by 0061-MADR D2's decision (E30).

## Found by this audit, not in 0054-REPORT

* **The wizard collects fallbacks nothing uses.** `Options.NeedFallbacks`
  fills `Result.Fallbacks` (E9); nothing in `llmprovider/` reads it.
* **`lockWait` is shorter than a refresh may hold the lock.** 0028-MADR
  shortlisted this as H9 and did not choose it (E43). 0059-MADR's wizard wait
  rests on the same bound, "at most about 46 s" (E44).
* **The stream reader dispatches an event cut off by the end of the stream**
  (E45), against the SSE specification's rule, pinned by
  `TestReadEvents_LastEventWithoutBlankLine` (E46). No record names it.
* **`NewHTTPClient`'s doc now states the transport's bounds** (E28), so a
  change to one of section 5's values is a change to exported documentation
  (0066-MADR D2).

## Probe results

* **P1. A refused certificate is retried.** Against an untrusted test
  server, `http.Client.Do` returned a `*url.Error` wrapping a
  `*tls.CertificateVerificationError`, which wraps
  `x509.UnknownAuthorityError`; under a wrong server name, it wraps
  `x509.HostnameError`. Neither chain holds a `*net.OpError`, so
  `transport.Offline` and `transport.Unsendable` both report false, and
  `retryable` returns true (E10). The message read `tls: failed to verify
  certificate: x509: certificate signed by unknown authority`.
* **P2. A silent HTTP/2 connection holds the next request.** A request over
  HTTP/2.0 succeeded; the peer then stopped reading and its writes were
  dropped. With no health check, the next request failed after 8.0 s, the
  probe's `ResponseHeaderTimeout` (the module's is 300 s), with `http2:
  timeout awaiting response headers`. With `HTTP2Config{SendPingTimeout:
  1s, PingTimeout: 1s}` it failed after 2.0 s with `http2: client connection
  lost`.
* **P3. The longest refresh hold.** Three attempts of up to 15 s (E15, E16)
  and backoffs of 200 ms and 400 ms hold the lock up to 45.6 s; `lockWait` is
  40 s (E14).
* **P4. Messages' history allocations.** `BenchmarkHistory/items=200`, 2000
  iterations, `-memprofilerate=1`: 743 allocations and 510,863 bytes per
  operation. By allocated objects: `reflect.unsafe_New` 50.57 % (through
  `reflect.New` 57 % and `reflect.packEfaceData` 43 %),
  `Content.MarshalJSONTo` 14.41 % flat and 66.73 % cumulative,
  `regexp.(*Regexp).replaceAll` 14.21 %, `RemapCallIDs` 16.59 % cumulative.
* **P5. `reAuth` carries most of the redaction cost.** On the same 2 KiB
  text, `Redact` took 202 to 220 µs and `reAuth.Match` alone 169 to 273 µs,
  three runs each, under the load noted above. One review's figure of 92 % at
  16 KiB compared `Redact` on one text with `reAuth` on another, and is not
  used.

## Ranking of what remains

By value against cost. The first group is internal, small, and changes
neither the exported API nor what a request carries:

1. **Small and internal:** 5.1's health check; 6.2's refused certificates,
   501 and 505; 10.2's `lockWait`; 10.3's failure memory and early run;
   9.2's keyword-first `reAuth`. 0068-MADR takes these up.
2. **Small, wanting a measurement first:** 6.1 (a live 429's headers); 3.1
   and 3.3's Kilo part (a precedent exists); 8.2's `RemapCallIDs` byte loop.
3. **New exported API:** 4.3 `WithFallback` over the wizard's list; 4.2's
   cooldown.
4. **Against an accepted record, or large:** 5.2 (every provider's request,
   0031-MADR Q3); 8.3's default-on caching (0039-MADR Q1); 1.3; 4.1; 7.3's
   partial arguments; 9.3.

## Evidence index

<!-- claims: self=d65e9d1 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| E1 | 0054 says the module has no caching option | `docs/reports/0054-REPORT-heuristics-and-performance-opportunities.md:471` | `No prompt-caching option.` |
| E2 | 0039-MADR's date | `docs/decisions/0039-MADR-request-side-prompt-caching.md:3` | `date: 2026-10-13` |
| E3 | The caching option | `llmprovider/contract.go:111` | `type CacheRetention string` |
| E4 | 0054's label count | `docs/reports/0054-REPORT-heuristics-and-performance-opportunities.md:147` | `table of 25 static ids` |
| E5 | The label table | `llmprovider/catalog/models_catalog.go:855` | `var modelLabels = map[llmprovider.ProviderID]map[string]string{` |
| E6 | `Label` | `llmprovider/catalog/models_catalog.go:893` | `func Label(provider llmprovider.ProviderID, model string) string {` |
| E7 | 0054's parse claim | `docs/reports/0054-REPORT-heuristics-and-performance-opportunities.md:150` | `` so `qwen3.8` and `gpt-10` do not parse `` |
| E8 | A major is a one-digit token | `llmprovider/catalog/models_catalog.go:696` | `if !isVersion \|\| len(tokens[i]) > 1 {` |
| E9 | The wizard's fallbacks | `wizard/configure.go:47` | `Fallbacks   []string` |
| E10 | A kindless send failure is retried | `llmprovider/retry.go:339` | `return errors.As(err, &urlErr) && !transport.Unsendable(err)` |
| E11 | Only 525 and 526 are terminal among 5xx | `llmprovider/api_error.go:455` | `case status == 525 \|\| status == 526:` |
| E12 | Every other 5xx is retryable | `llmprovider/api_error.go:457` | `case status >= http.StatusInternalServerError:` |
| E13 | A DNS failure is offline | `llmprovider/internal/transport/offline.go:24` | `errors.AsType[*net.DNSError](err)` |
| E14 | The lock wait | `llmprovider/auth/tokenstore_file.go:25` | `const lockWait = 40 * time.Second` |
| E15 | Refresh attempts | `llmprovider/auth/oauth_session.go:36` | `oauthRefreshAttempts = 3` |
| E16 | Each attempt's bound | `llmprovider/auth/oauth_session.go:55` | `var oauthRefreshAttemptTimeout = 15 * time.Second` |
| E17 | The header timeout | `llmprovider/internal/transport/transport.go:72` | `ResponseHeaderTimeout: 300 * time.Second,` |
| E18 | The shared transport, no HTTP2 field | `llmprovider/internal/transport/transport.go:95` | `t := &http.Transport{` |
| E19 | A reader per stream | `llmprovider/internal/wire/sse.go:43` | `reader := bufio.NewReaderSize(body, 64<<10)` |
| E20 | `reAuth` | `internal/redact/redact.go:37` | `reAuth = regexp.MustCompile(` |
| E21 | Its pass is the regex's own search | `internal/redact/redact.go:197` | `{re: reAuth, replaceSub: redactAuth, anchors: []string{"bearer", "basic", kwToken}},` |
| E22 | `CheckDispatch` marshals a deferred schema again | `llmprovider/internal/wire/deferral/deferral.go:148` | `if _, err := json.Marshal(tool.Schema); err != nil {` |
| E23 | A command's failure is not stored | `llmprovider/command_token.go:132` | `if err == nil {` |
| E24 | A failed listing is not stored | `llmprovider/catalog/discovery.go:423` | `return fetchedListing{}, models, time.Time{}, failure` |
| E25 | Ranking divides by the maximum | `llmprovider/catalog/model_ranking.go:223` | `rankCostWeight*n.cost(c)/n.maxCost` |
| E26 | Kilo's capabilities fetch each call | `llmprovider/catalog/discovery.go:1500` | `entries, err := fetchKiloCatalog(ctx, llmprovider.Token{Value: apiKey}, cfg)` |
| E27 | OpenAI streams plan sessions too | `llmprovider/providers/openai/openai.go:519` | `streamed := stream \|\| p.chatGPT \|\| p.plan` |
| E28 | `NewHTTPClient`'s doc names the bound | `llmprovider/options.go:63` | `reply within 300 s` |
| E29 | 0053 leaves 6.1 to 0054 | `docs/decisions/0053-MADR-stream-retry-and-cache-defects.md:445` | `It is idea 6.1 in` |
| E30 | 0061 retries an offline failure | `docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md:175` | `It is retried, as` |
| E31 | Equal jitter | `llmprovider/retry.go:308` | `if half := delay / 2; half > 0 {` |
| E32 | `reKV`'s keyword-first pass | `internal/redact/redact.go:294` | `func redactKVPass(p []byte) []byte {` |
| E33 | 0054's `gpt-5` example | `docs/reports/0054-REPORT-heuristics-and-performance-opportunities.md:175` | `` so `gpt-5` ranks `gpt-5` above `` |
| E34 | 0054's layer claim | `docs/reports/0054-REPORT-heuristics-and-performance-opportunities.md:128` | `` the existing `catalog.Layer` interface `` |
| E35 | 0054's streaming claim | `docs/reports/0054-REPORT-heuristics-and-performance-opportunities.md:326` | `every non-ChatGPT` |
| E36 | 0054's A8 claim | `docs/reports/0054-REPORT-heuristics-and-performance-opportunities.md:330` | `0028 A8 measured this` |
| E37 | 0054's allocation claim | `docs/reports/0054-REPORT-heuristics-and-performance-opportunities.md:489` | `744 allocations at 200 items` |
| E38 | The marshal the copies come through | `llmprovider/internal/wire/messages/messages.go:71` | `return jsonv2.MarshalEncode(enc, *c.blocks)` |
| E39 | 0054's expression count | `docs/reports/0054-REPORT-heuristics-and-performance-opportunities.md:510` | `eleven compiled RE2 expressions` |
| E40 | 0054's learned-state claim | `docs/reports/0054-REPORT-heuristics-and-performance-opportunities.md:247` | `the only learned state across calls is` |
| E41 | The second memo | `llmprovider/internal/cacheref/cacheref.go:25` | `Thinking *wire.Learned` |
| E42 | 0054's NXDOMAIN claim | `docs/reports/0054-REPORT-heuristics-and-performance-opportunities.md:370` | `x509 verification errors and DNS NXDOMAIN` |
| E43 | 0028 shortlisted H9 | `docs/decisions/0028-MADR-heuristics-and-performance-from-the-research-pass.md:698` | `` H9 (`lockWait`, 40 s, is `` |
| E44 | 0059's wizard wait | `docs/decisions/0059-MADR-wizard-debugging-pass-findings.md:514` | `at most about 46 s` |
| E45 | The reader dispatches at the end | `llmprovider/internal/wire/sse.go:82` | `if stop, err := next(); stop \|\| err != nil {` |
| E46 | The test that pins it | `llmprovider/internal/wire/sse_0031b_test.go:43` | `func TestReadEvents_LastEventWithoutBlankLine(t *testing.T) {` |
