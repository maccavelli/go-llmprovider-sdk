---
status: accepted
date: 2026-10-04
decision-makers: repository owner
consulted: 0009-MADR-use-case-aware-default-model-ranking.md (§3, §4), 0012-MADR-conform-providers-to-reference-clients.md (§1.1, §1.3, §4.1), 0015-MADR-canonical-sdk-api-and-module-layout.md (D8, R48), 0016-MADR-provider-auth-and-support-baseline.md (D8, A2), 0020-MADR-remediate-v1-debugging-pass-findings.md (Q2, F9, F24, F42, F56, F59)
informed: consumers of go-llmprovider-sdk v1
---

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Harden and Tune the SDK After the v1.1 Review

> **Status.** Accepted 2026-10-04. The owner decided the five design
> questions that day (§1 of the Decision Outcome, D1–D5): "1. retry 1 time.
> 2. what is the chatgpt native stream limit? we should align the timeout
> with the provider native function. 3. amend and change. 4. adopt, change
> the goldens. 5. add it." They accepted the record the same day: "accept
> the madr and write the detailed, actionable, and deterministic plan". The
> amendment at the end corrects facts found while writing
> [0021-PLAN-harden-and-tune-after-the-v1-1-review.md](0021-PLAN-harden-and-tune-after-the-v1-1-review.md).

## Context and Problem Statement

`v1.1.0` shipped on 2026-10-04 with 0020's 59 fixes. On the same day the
owner asked for a pass "across the codebase", looking for "areas we can
improve heuristic and better leverage algorithms" and "opportunities to
harden and optimize".

### Method

* **Four read-only reviews ran in parallel**, one per area:
  * the model catalog: ranking, search, metadata, discovery, and OpenCode's
    route heuristic;
  * transport, retries, errors and credentials;
  * request encoding, response decoding, streaming, the providers and the
    `llmtest` harness;
  * the wizard, secret redaction, and the repository's gates and tests.
* **Each review ran from a scratch copy** outside the repository, and wrote
  scratch tests and benchmarks there. The repository was not changed.
* **Each review read 0020's findings table** and did not re-report a
  finding fixed there, except where the fix is incomplete. That is said
  where it applies (C1, T5).
* **The author re-read the code of 16 findings** before writing this
  record: C1, C4, C5, C13, T4, T6, T7, T11, T12, W1, W2, W4, W10, Z4 and
  Z9, and the code behind W8. Every one matched the review.
* **The reviews' overall verdict:**
  * the code is in good shape;
  * `go test -race -shuffle=on -count=5 ./...` passes, and `go test -race
    ./llmprovider/...` reports no data race;
  * error bodies are bounded;
  * every type that holds a secret redacts it in `String`, `GoString`,
    `LogValue` and `MarshalJSON`;
  * the ranking formula and its comparators are sound.

  What remains is heuristic gaps, a few real bugs, and gates that can pass
  while what they guard is broken.

In the tables, **Evidence** says how each finding is known:

* **re-read:** the author read the cited lines;
* **reproduced:** a scratch test by the review showed the failure;
* **measured:** a scratch benchmark by the review;
* **read:** the review read the code and did not run it.

### A. Bugs with user impact

| ID | Location | Finding | Evidence |
| :--- | :--- | :--- | :--- |
| W1 | `internal/wire/messages/messages.go:128, 164-172`; `generatecontent/generatecontent.go:116-119, 156-163`; `internal/wire/wire.go:39-48`; `chatcompletions` arguments decode | **Tool-call arguments lose precision and can fail a reply.** <ul><li>Claude `tool_use.input` and Gemini `functionCall.args` decode into `map[string]any`, then re-marshal. `{"z":1,"id":12345678901234567890}` comes back as `{"id":12345678901234567000,"z":1}`.</li><li>`wire.ToolArguments` round-trips the same way on replay.</li><li>A Claude call with no `input` gives `Arguments ""`, where generateContent gives `"{}"`.</li><li>Chat Completions fails the whole reply when a gateway sends `function.arguments` as an object.</li><li>Gemini Interactions (`providers/gemini/interactions.go:179-190`) already keeps the raw JSON.</li></ul> | re-read (the map); reproduced (precision, the object form) |
| W2 | `internal/wire/responses/responses.go:170, 203-204, 246-261` | **The ChatGPT stream reader's 16 MiB limit covers the whole stream, not one event.** <ul><li>A stream of 68,826 realistic delta events (17.8 MB, about 69k output tokens) fails with `ErrIncomplete … unexpected end of JSON input`, or with the retryable "stream ended before response.completed" when the cut lands on a line.</li><li>`bufio.ErrTooLong` is wrapped as `ErrProviderUnavailable`, so it would be retried.</li><li>Each event costs about five allocations, and ignored delta events are fully unmarshalled.</li></ul> | re-read; reproduced; measured: 4 MiB stream 17.5 ms, 16.4 MB, 81,014 allocs/op against 0.85 ms, 67 KB, 12 allocs/op for a prototype with the same output on all three `.sse` goldens |
| W3 | `messages.go:32-39, 139-146`; `generatecontent.go:133-141, 172-197`; `chatcompletions.go:226-230, 243-247` | **Finish reasons are mapped in each decoder, and the decoders disagree.** <ul><li>`contract.go:125-127` says an unknown value is kept as sent. Messages and generateContent map it to `""` (Gemini `OTHER`).</li><li>Claude's `stop_reason: model_context_window_exceeded` with a `tool_use` returns the truncated call as valid: F3's hole, through a missing table entry.</li><li>Chat Completions reports `stop` for a reply with `tool_calls`; the other wires report `tool_calls`.</li><li>An empty answer loses its reason: Gemini `SAFETY` with no parts, Claude `refusal` and Chat `content_filter` all become "no content".</li><li>Chat copies the reply's `role` verbatim, so `"Assistant"` fails the next turn's `validate`.</li></ul> | reproduced |
| W4 | `responses.go:86-95, 136-147, 233-242` | **The Responses wire returns an empty success, and drops refusals.** `{"status":"completed","output":[{"type":"reasoning","summary":[]}]}` returns no output, `FinishReason: stop`, no error, from `Decode` and `ReadStream`; the other four decoders return `ErrIncomplete` (0020 Q2 a). A content part `{"type":"refusal","refusal":"…"}` is ignored, so a refusal is that same empty success. Affects OpenAI, ChatGPT, Grok and OpenCode's responses route. | re-read; reproduced |
| W10 | `generatecontent.go:116-119, 164-169` | **Parallel Gemini calls to one function share a CallID.** `CallID` is the function name, so two `weather` calls (ids `fc_1`, `fc_2`) both decode as `CallID="weather"`, and a caller keying results by CallID overwrites one. OpenCode's Google route. | re-read; reproduced |
| T4 | `llmprovider/auth/oauth_session.go:91-95, 365-380` | **A refresh reply that decodes strictly loses the rotated refresh token.** `ExpiresIn int64`: a 200 with `"expires_in":"3600"` or `3600.0` fails to decode, and the session keeps the spent token. The issuer has rotated, so the next refresh replays a spent token, which revokes the family (0016-MADR A2). This is also the only OAuth decode with no size limit. | re-read; reproduced |
| T2 | `oauth_session.go:149-163, 245` | **A failed early refresh discards a token that still works**, and is retried on every call. A token valid for 4 more minutes with the issuer answering 503 gives an empty token, an error, and 3 refresh attempts per call. | reproduced |
| T3 | `oauth_session.go:22-26, 241-253, 495-503` | **A token that lives 5 minutes or less is refreshed on every `Token` call.** With `expires_in=240`, 10 calls made 10 refreshes. A fast local clock does the same. | reproduced |
| T11 | `auth/oauth_device.go:214-220, 239-245, 302-316`; `auth/kilo_device.go:85-87, 113-114` | **One transient error ends a device-code login.** A single 502 while polling Kilo returns `device poll failed`; the OpenAI and Grok loops do the same on a transport error, 429 or 5xx. RFC 8628 §3.5 has the client slow down and keep polling. | re-read; reproduced |
| C1 | `catalog/model_metadata.go:136-146, 241-258` | **A caller's own deadline poisons the metadata cache for a minute.** F5's fix skips caching only for `context.Canceled`; a `DeadlineExceeded` from the caller's short deadline is stored as a host failure. On OpenCode that minute loses the metadata route, `reasoning_effort` and the interleaved replay field. | re-read; reproduced |
| C7 | `catalog/model_ranking.go:230-237`; `discovery.go:525-528` | **The fill re-admits models that fail at request time.** A sparse OpenCode Go set fills with two region-gated DeepSeek models and a `-contributor` model, which 0009 §3 items 7–8 exclude. The metadata-failure fallback never applies item 8. 0009 §4's fill skips only item 9. | reproduced |
| C13 | `catalog/discovery.go:94-102` | **A live listing that curates to nothing is replaced by the static catalog.** A Kilo listing of only `kilo-auto/small` and `kilo-auto/balanced` (utility) returns `Live=false`, `Err=nil`, and the static six in `Usable`, none of which the listing holds; the real tiers drop out of search. Reachable with an organization's restricted catalog. | re-read; reproduced |
| Z3 | `llmprovider/api_error.go:198-203, 236`; `auth/oauth_loopback.go:551-571`; `wizard/configure.go:375-385` | **Terminal escape sequences in a remote error body reach `Error()` and the wizard's terminal.** `{"error":{"type":"x\u001b[31m","message":"\u001b]0;owned\u0007\u001b[2Jhello"}}` keeps every ESC byte. Whoever controls the body (any entered base URL, a proxy, a LAN Ollama) can retitle or clear the terminal, spoof OSC 8 links, or set the clipboard with OSC 52 where allowed. `Code` is not bounded, and the callback's `code` is not stripped. | reproduced |
| Z2 | `internal/redact/redact.go:31-63` | **Redaction misses secrets and redacts diagnostics.** Missed: camelCase keys (`accessToken`, `refreshToken`, `idToken`, `sessionToken`); a PEM key's body (only the header goes); every cookie pair after the first; a quoted multi-word value; URL-encoded forms; `signature=`. Redacted in error: `{"code":"context_length_exceeded"}`, `'code': 'model_not_found'`, `token: 128000`. The package is at 100% coverage, which says nothing about this. | reproduced |

### B. Hardening and availability

| ID | Location | Finding | Evidence |
| :--- | :--- | :--- | :--- |
| T5 | `internal/wire/decode_error.go:37-40`; `retry.go:140-142`; `internal/transport/transport.go:32` | **A read failure after a 200 is retried and billed again.** 0020 Q2 (a) retries a kindless failure "only when it comes from the transport". The code accepts any `net.Error`, and `DecodeError` passes body-read failures through. A 200 whose body stalls past `Client.Timeout` (`*http.timeoutError`) was sent 3 times: up to 3 × 330 s of billed generation. Decided by D1. | reproduced |
| W6 | `decode_error.go:37-41`; each provider's `io.LimitReader(resp.Body, wire.ReplyLimit)` | **A connection dropped mid-body is never retried.** A Content-Length body cut at 40 bytes gives `io.ErrUnexpectedEOF`, not a `net.Error`, so it becomes terminal `ErrIncomplete`, and is indistinguishable from the `ReplyLimit` cut. The SSE path retries the same loss. Decided by D1. | reproduced |
| T13 | `transport.go:32`; `providers/openai/openai.go:296-298` | **Long ChatGPT streams are cut at 330 s in total.** `Client.Timeout` covers the body; the event stream is read under it, so a reasoning reply that runs past 330 s fails while events still arrive. The references use an idle limit (More Information, "Research"). Decided by D2. | read; no live capture over 330 s |
| T12 | `transport.go:30-41` | **The default client has no connect timeout.** No `DialContext`, so a blackholed host waits for the OS connect timeout (about 75 s on macOS, 127 s on Linux), and `WithRetry` repeats it. Setting a dialer also requires `ForceAttemptHTTP2: true`, or HTTP/2 is silently lost. | re-read |
| T6 | `retry.go:86-102` | **`WithRetry` sleeps until the caller's deadline and loses the error.** A 429 with `Retry-After: 5` under a 1 s deadline waits 1 s and returns bare `context deadline exceeded`; the caller loses the kind and `RetryAfter` it needs to reschedule. | re-read; reproduced |
| T7 | `retry.go:108-121` | **Retries arrive together.** A server-directed wait is used exactly: 16 callers rate-limited together with `Retry-After: 1` retried within 1 ms. Backoff jitter adds up to 25% and is then clipped by `min(delay, MaxDelay)`, so at the cap every caller waits exactly `MaxDelay`. | re-read; reproduced |
| T14 | `api_error.go:213-216, 294-306` | **409 and `x-should-retry: true` are not retried.** Only `x-should-retry: false` is honoured; 409 is terminal `ErrInvalidRequest`. The official OpenAI and Anthropic SDKs obey the header both ways and retry 408, 409, 429 and 5xx. | read; SDK source verified (More Information) |
| T1 | `internal/wire/reauth.go:17-28`; `auth/oauth_session.go:235-239`; `llmprovider/command_token.go:120-124` | **Concurrent 401s renew the token several times.** `Invalidate()` does not know which token was refused, so a late 401 discards a fresh token. 8 in-flight requests on one token gave 4 refreshes (1 needed); through `claude.New` with a `CommandToken`, 4 command runs (2 needed). Decided by D5. | reproduced |
| T8 | `oauth_session.go:328-343, 352-358`; `auth/tokenstore_file.go:29-31` | **A refresh can hang for about 16 minutes while it holds the cross-process lock.** Each attempt is bounded only by the client (330 s), retried 3 times, with the heartbeat keeping the lock fresh. | reproduced |
| T9 | `tokenstore_file.go:235-246, 338-343, 241-243` | **Lock staleness depends on clocks and on heartbeat reads.** Staleness is the waiter's clock minus the file's mtime, so a writer 40 s behind loses a live lock. `lockWait` (25 s) is under `lockStaleAfter` (30 s). One failed read stops the heartbeat for good. A Windows takeover rename failure is not retried. | reproduced (skew case unlikely in practice) |
| T10 | `oauth_session.go:105, 140-148, 192-199` | **A pending re-save blocks requests that hold a valid token.** While `spentRefresh` is set, every `Token` call takes the lock; with another process holding it, each call waits up to `lockWait` (25 s). | reproduced |
| T15 | `internal/ownerperm/ownerperm_unix.go:9-11`; `tokenstore_file.go:48-56` | **An existing token directory keeps whatever mode it has on Unix.** A group- or world-writable directory lets another user rename a file over `<provider>.json` or `.lock`. | read |
| C2 | `model_metadata.go:223-262` | **The metadata cache has no single-flight and no stale-while-revalidate.** 20 concurrent cold lookups made 20 full downloads; after the TTL, 10 lookups made 10, each blocked for the fetch. OpenCode makes up to three lookups per `Generate`. | measured |
| C3 | `model_metadata.go:288`; `discovery.go:244, 338, 392, 492, 607, 785` | **Metadata and listing bodies are read with no size limit**, against Together's 8 MiB cap (`discovery.go:665, 696`). The metadata URL can come from the caller or the environment and is fetched on the generation path. | read |
| C9 | `model_metadata.go:345-353`; `model_ranking.go:126-157`; `discovery.go:750-756` | **Negative, future-dated and non-finite metadata are not guarded.** A negative cost ranks first; a future `release_date` gives a negative age; costs of 1e308 overflow to +Inf, so the blend is NaN and the comparator is no longer a strict weak order. `kiloPriceRank("NaN")` returns NaN. | reproduced |
| W5 | `chatcompletions.go:196-222` | **A 200 reply carrying an error envelope loses its message.** `{"error":{"message":"Upstream overloaded","code":502}}` gives terminal "the answer has no choices"; a choice with `finish_reason: "error"` (OpenRouter's convention, which Kilo inherits) gives "no usable content". | reproduced (gateway behaviour documented by OpenRouter, not captured from Kilo) |
| W7 | `responses.go:42-51`; `messages.go:83-94` | **Opaque reasoning is replayed onto another wire.** Any `ReasoningItem.Encrypted` goes out as `encrypted_content` on Responses and as `redacted_thinking.data` on Messages. OpenCode's per-request `Model` can switch wires on the same `Input`. The ChatGPT backend refuses altered content with a terminal 400 (0020 amendment of 2026-10-04). Decided by D5. | read |
| W8 | `providers/openai/openai.go:197-198, 206-207`; `providers/grok/grok.go:183`; `providers/opencode/opencode.go:358` | **Stateless Responses requests never ask for encrypted reasoning, outside the ChatGPT session.** `include: ["reasoning.encrypted_content"]` is sent only for a ChatGPT session. OpenAI with `WithStore(false)`, Grok with `WithStore(false)`, and every OpenCode responses-route model send `store: false` without it, so a reasoning tool loop carries no reasoning between turns; OpenCode also sends no `summary`. Codex sends the `include` on every request. Decided by D4. | re-read; Codex source verified |
| W9 | `chatcompletions.go:183-193` | **Kilo drops `reasoning_details`.** OpenRouter-style gateways carry Anthropic thinking signatures and Gemini 3 thought signatures there; Gemini 3 is documented to refuse a replayed call without one. The Kilo-path counterpart of F7. | read; not run live |
| Z1 | `api_error.go:190-203`; `redact.go:65-90` | **Redaction runs on up to 64 KiB of an error body, then keeps 512 bytes.** A 64 KiB HTML 502 page costs 27.5 ms and 340 KB per `ClassifyHTTPError`. Redaction is about 400 ns per byte; `(?i)` and `\b` leave the regexes no literal prefix. A prefilter on anchor strings cut clean 4 KiB from 1.69 ms to 42 µs. | measured |
| Z7 | `wizard/configure.go:255-260`; `llmprovider/options.go:53-57`; `wizard/auth.go:239-247` | **Remote base URLs are accepted unchecked, and credentials follow them.** `http://` to a remote host, `user:pass@`, a query, or a scheme-less typo are taken as entered, so a key or Kilo token may be sent in cleartext before the typo shows. | read |
| Z10 | `wizard/auth.go:347-357`; `auth/oauth_session.go:57-72` | **Any pasted OpenAI credential that is not `sk-` is saved as a ChatGPT session**: an `xai-` key, a typo, or an expired JWT, with no expiry; the failure shows later as a 401. | read |

### C. Heuristic quality

| ID | Location | Finding | Evidence |
| :--- | :--- | :--- | :--- |
| C4 | `catalog/models_catalog.go:162-173, 329-350` | **OpenAI's filter admits non-chat models.** The deny list is prefix-shaped (`"tts-"`), so `gpt-4o-mini-tts`, `gpt-4o-transcribe`, `gpt-4o-search-preview`, `o3-deep-research` and `gpt-5-search-api` are usable, and `mini` gives the first three +180 in backfill. | re-read; reproduced |
| C5 | `models_catalog.go:272-299, 650-665` | **Raw substring heuristics: "gemini" and "minimax" contain "mini".** `rankOpencodeModel`'s switch stops at `mini` (+170), so `gemini-3.1-pro` never gets the −300 `-pro` penalty; `minimax-m3` scores 170. This order is the metadata-failure fallback and the start of the fill list. | re-read; reproduced |
| C6 | `models_catalog.go:459-475, 517-526, 558-568` | **Hard-coded version tables invert newer generations.** `gemini-3.8-flash` 180 < `gemini-2.5-flash` 210; `gemini-3.10-flash` matches `"3.1"`; `grok-4.7` 40 < `grok-4.6` 60; `grok-5` 0; `claude-opus-5` 100 < `claude-opus-4-8` 135. | reproduced |
| C8 | `model_ranking.go:243-252` | **The diversity group is too fine for gateway ids.** The group is models.dev `family`, which is per model line: Zen has 10 gpt families, Go 6 qwen and 5 mimo, so the two-per-group cap barely applies. 0009's Consequences call it coarse. Grouping by the family's leading letters changed only Zen capable (`gpt-6-astra` gives way to `claude-fable-5-1`) in the 2026-09-26 snapshot. Decided by D3. | measured |
| C11, Z11 | `catalog/model_matcher.go:40-66, 85-135`; `wizard/model_select.go:45-49` | **Search ranking is weak.** Annotation text matches at high tiers (`"fast"` hits 4 of 6 OpenAI models). `o3` returns 66 matches, about 62 of them subsequence noise. Ties break on full-id length, so the vendor prefix sways them. Listing order is discarded. Glob matches are unranked. No way to use a query with no match as the id. Search costs about 0.45 ms over 400 ids, which is fine. | reproduced |
| C12 | `models_catalog.go:183-201, 676-725` | **Exported helpers disagree with `List`, and labels drift.** `Rank("Gemini", …)` is 0 while `List` folds case (F35); `Rank` has no Together case; 29 static ids have no label; 6 labels name no static id; `Label` ignores the provider. | reproduced |
| C10, W13 | `providers/opencode/route.go:76-251`; `metadata_route_test.go:149-180` | **The route table and heuristic are kept in step by hand.** The table equals `routeForNPM(testdata/opencode-routes.json)`: Zen 77/77, Go 33/33. Leave-one-out finds 0 disagreements today, but only spot tests guard it. | measured |
| Z6 | `wizard/text_prompter.go:379-459` | **Masked entry redraws on every character and relies on an ANSI code.** A 2,000-character paste writes 2,001 times, 118 KB to the terminal. `\033[K` is used, and `x/term`'s `MakeRaw` enables VT input only on Windows (`term_windows.go:25-36`), so legacy consoles may print it. Esc swallows the next key; Ctrl-U is ignored; Ctrl-C does not match `context.Canceled`. | reproduced (Windows not run) |

### D. Tooling, tests and structure

| ID | Location | Finding | Evidence |
| :--- | :--- | :--- | :--- |
| Z4 | `scripts/check_deps.py:45-61` | **dep-check passes a module only a test imports.** It reads `.Deps` of the non-test build. A planted `example.com/evil` imported from a `_test.go` passed dep-check and `go mod tidy -diff`. | re-read; reproduced |
| Z9 | `scripts/check_generated.py:39-43`; `scripts/check_parity_map.py` | **Two gates can pass while broken.** generate-check checks 0 files when the directive says `-output=zsyscall_windows.go`. parity-check accepts a cell with no Go name at all, such as `TBD` (a gap in 0020 F56). | re-read (parity); reproduced |
| Z5 | `.github/workflows/ci.yml`; `scripts/go-precheck.sh:119, 132-149`; `scripts/coverage-floors.txt` | **CI and precheck gaps.** govulncheck never runs in CI, and nothing scheduled sees a new advisory. No `-shuffle=on`, `timeout-minutes` or `concurrency`. The precheck runs no `-race` or `tidy -diff`. Floors sit far below actual (llmprovider 98.5% against 89.2%). | read; measured |
| Z8 | `wizard/fake_prompter_test.go` | **The fake prompter never fails on unused scripted answers.** A Cleanup check fails 9 tests: their scripts no longer describe their flows since F19 (a gap in 0020 F59). | reproduced |
| Z12 | `llmprovider/command_token_leader_test.go:37`; `auth/oauth_session_leader_test.go:58` | **Two F14 tests depend on timing**: a 50 ms sleep is assumed to let the waiter join. 12 `time.Sleep` calls in tests; no `testing/synctest`. | read |
| W11 | `generateOnce` in 9 providers; Responses tools ×3; Messages tools ×2; effort merge in 7 variants | **Request plumbing is copied, and the copies drift.** A stale "1 MiB" comment in 6 files (ReplyLimit is 16 MiB); doubled error prefixes; a nil `Tool.Schema` sent as `null`, which Anthropic refuses. | read |
| W12 | `llmtest/llmtest.go:27-50, 418-460` | **The conformance harness has gaps.** A provider passes `Run` while dropping `FunctionCallOutputItem` or `Instructions`, ignoring `Request.Model`, skipping `wire.DecodeError`, reporting a tool reply without `tool_calls`, returning `Arguments ""`, or mishandling a cut answer. Decided by D5. | read |

## Decision Drivers

* **Correctness first:** a tool argument altered, a refusal reported as
  success, or a rotated token lost is worse than any slowness.
* **Billing:** a generation that succeeded upstream must not be paid for
  several times by an automatic retry (0020 Q2).
* **Align with the reference clients and the official SDKs** where they
  agree, and record the evidence (0012).
* **Additive only** (R48 in `docs/guides/api-standards.md`); a wire change
  updates its goldens (R45).
* **Standard library and `golang.org/x/term` only** (`AGENTS.md`).
* **Every check is seen to fail first**, and a gate is not trusted until it
  fails on a planted breach.

## Considered Options

* Remediate in a 0021 PLAN, ordered by severity, with the owner's five
  decisions (D1–D5)
* Fold each finding into the record that owns its area (0009, 0012, 0016,
  0020)
* Fix the bugs of Table A only, and record the rest as accepted
* Defer everything to a v1.2 planning cycle

## Decision Outcome

Chosen option: "Remediate in a 0021 PLAN, ordered by severity, with the
owner's five decisions", because several findings cross records (D1 touches
0012 and 0020; D3 touches 0009), and one record can say what changes in each
and why, while the PLAN executes and proves each fix.

### 1. The owner's decisions

#### D1. A failure while reading a 200 reply is retried once

The owner: "retry 1 time." It amends 0020 Q2 (a) for this case.

* **What counts:**
  * a transport failure while reading the body of a 200: a connection
    reset, `io.ErrUnexpectedEOF`, the idle timeout of D2, or the client's
    own timeout;
  * an event stream that ends before its completion event (0012 §4.1).
* **The rule:** `WithRetry` retries such a failure **at most once per
  `Generate` call**, whatever `MaxAttempts` says, then returns it. The
  usual `MaxAttempts` budget still governs failures before the reply
  (connect errors, 429, 5xx).
* **What does not count**, and is never retried:
  * a reply over `ReplyLimit`. A shared body reader reads `limit+1` and
    reports it as its own terminal `ErrIncomplete`, so it is no longer
    mistaken for a cut connection (W6);
  * a body that arrived whole but does not decode (`ErrIncomplete`, as
    0020 F9 decided).
* **How it is told apart:** the shared request path (W11's `wire.Post`)
  records the first non-EOF read error. `DecodeError` turns a read failure
  into `ErrProviderUnavailable` carrying an unexported "after the reply"
  mark, and `retryable` grants that mark one attempt.
* **Without `WithRetry`** nothing is retried, as 0015-MADR D8 decides.
* **Why once, where Codex allows 5** (`stream_max_retries`): a lost reply
  may have been generated and billed in full. One retry recovers the common
  case, a dropped connection, while bounding the cost to two generations.

#### D2. Generation timeouts follow the ChatGPT client: an idle limit, not a total

The owner: "what is the chatgpt native stream limit? we should align the
timeout with the provider native function." The answer, from Codex's source
(More Information, "Research"):

* Codex waits up to **300 s for each stream event**
  (`DEFAULT_STREAM_IDLE_TIMEOUT_MS: u64 = 300_000`), applied as
  `timeout(idle_timeout, stream.next())` around every event. Its client
  sets **no total timeout**.
* The official OpenAI SDK is also idle-based: `httpx.Timeout(timeout=600,
  connect=5.0)`, where httpx's read timeout is "the maximum duration to
  wait for a chunk of data".

Decided, amending 0012 §1.3:

* **Body reads have a 300 s idle limit:** no data for 300 s cancels the
  request and is a read failure under D1. This applies to every generation
  body, event stream or JSON. It is enforced in the shared request path,
  so it also covers a caller's `WithHTTPClient` client.
* **`DefaultClient` drops its 330 s total `Timeout`.** A generation is
  then bounded by the idle limit and by the caller's context.
* **`ResponseHeaderTimeout` stays 300 s**: the wait for the first byte,
  as 0012 §1.3 has it.
* **A connect timeout of 30 s** (`net.Dialer{Timeout: 30 s, KeepAlive:
  30 s}`, `net/http`'s own default), with `ForceAttemptHTTP2: true` so
  HTTP/2 is kept (T12).
* **Listings keep their own 10 s bound** (0013-MADR Q4); the catalog's
  shared client is unchanged.

#### D3. 0009 is amended: the fill keeps out models that fail, and groups by vendor

The owner: "amend and change." `0009-MADR-use-case-aware-default-model-ranking.md`
gets an amendment, and the code follows it:

* **§4's fill** skips what §3 items 7, 8 and 9 exclude, not only item 9.
  These are models that fail at request time (`-contributor` on Go, the
  region-gated DeepSeek ids on Go) or that the owner ruled out (Kilo's
  `kilo-auto/*` under the utility profile). The list may be shorter than
  six, as item 9 already allows. The same rule applies to Go's
  metadata-failure fallback curation (C7).
* **§4's diversity group** for an unprefixed id becomes the leading
  letters of its models.dev `family` (`^[a-z]+` of the lower-cased
  family), so `gpt-sol`, `gpt-luna` and `gpt-astra` are one group (C8). The
  vendor prefix rule for prefixed ids is unchanged. Measured on the
  2026-09-26 snapshot, it changes only the Zen capable six. The §7 golden
  test is updated.
* **§3 item 9 stays**: Kilo's `kilo-auto/*` tiers are not recommended under
  the utility profile.
  * **The owner reviewed it on 2026-10-04.** They noted that Kilo's own CLI
    prefers these tiers:
    * its default model is `kilo-auto/free`
      (`kilocode:packages/kilo-gateway/src/api/constants.ts:31`);
    * its small-task model is `kilo-auto/small`
      (`packages/opencode/src/kilocode/provider/provider.ts:298`).
  * **They kept the exclusion of 2026-09-26** ("option 1"), on one
    condition: "as long as the user can choose the auto models they can be
    excluded".
  * **The condition holds:**
    * the exclusion applies only inside the recommended-list ranking, at
      eligibility and in the fill (`catalog/model_ranking.go:208, 234`);
    * the tiers stay in `Usable` (`isUsableKiloModel`,
      `catalog/kilo_catalog_test.go:45`), and search covers `Usable`,
      including the glob `kilo-auto/*`
      (`catalog/model_matcher_test.go:45`);
    * they stay eligible under `ProfileCapable`;
    * the wizard's search and its Other entry reach them.
  * **One case broke it, and is fixed here:** a listing holding only
    `kilo-auto` tiers was replaced by the static catalog, so they dropped
    out of search (C13). A test in Phase 4 keeps the condition true
    (Confirmation).
* README's Ranking bullet says both.

#### D4. Encrypted reasoning is asked for whenever a Responses request is stateless

The owner: "adopt, change the goldens." Wherever a Responses request is
sent with `store: false`, it also sends `include:
["reasoning.encrypted_content"]`, as Codex does on every request
(`codex-rs/core/src/client.rs:965`):

* OpenAI with `WithStore(false)`;
* Grok with `WithStore(false)`;
* every OpenCode responses-route model, which also gets `reasoning.summary:
  "auto"` so its reasoning items carry text (as 0020 F24 did for OpenAI).

The ChatGPT session already does this. The `openai`, `grok` and
`opencode-*-responses` goldens change, each listed with its reason (R45).

#### D5. Three additive APIs are added

The owner: "add it." Each is additive under R48 and gets its rows in
`docs/architecture.md`:

* **`ReasoningItem.Format`** (W7): a string naming the wire that produced
  the item (`"responses"`, `"messages"`, `"chatcompletions"`,
  `"generatecontent"`).
  * Each decoder sets it. An encoder replays `Signature` and `Encrypted`
    only when it matches its own wire, or when it is empty (items built
    before this field).
  * Kilo's `reasoning_details` (W9) uses the same field, after its live
    check.
* **Token-aware invalidation** (T1): an optional interface,
  `llmprovider.TokenInvalidator` with `InvalidateToken(llmprovider.Token)`.
  * It is implemented by `*auth.OAuthSession` and `*CommandToken`: a no-op
    unless the refused token is still the current one.
  * `wire.Reauth` keeps the token the send used, and calls it when the
    source implements it. Otherwise it falls back to `Invalidate()`, so
    third-party sources keep working.
* **`llmtest.Harness` fields** (W12): optional hooks, such as serving a cut
  answer, that switch on the new checks. They cover tool outputs and
  instructions reaching the request, `Request.Model`, `DecodeError`
  classification, `tool_calls` on a tool reply, and non-empty arguments. A
  harness that sets none keeps passing.

### 2. The remediation, by phase for the PLAN

Every fix lands red-first, with the FAIL line and then the PASS line in the
PLAN's execution record, as in 0020.

* **Phase 1, replies and retries:**
  * **The shared request path (W11, D1, D2).** `wire.Post` sends the
    request, applies the idle limit, reads through a bounded reader, and
    classifies errors. The 9 `generateOnce` copies use it. Responses and
    Messages tool encoders are shared, and a nil schema becomes `{"type":
    "object", "properties": {}}`.
  * **Streams (W2).** The ChatGPT stream reader limits each event, not the
    stream, skips ignored delta events, and reports over-limit as
    `ErrIncomplete`.
  * **Retries (T5, W6, T6, T7, T14):**
    * D1's retry-once rule;
    * a wait that would pass the caller's deadline returns the last error
      at once, and a wait cut short returns `errors.Join(ctx.Err(),
      lastErr)`;
    * server-directed waits get +0–10% jitter (at least +0–250 ms);
      backoff uses equal jitter, applied after the cap;
    * `x-should-retry: true` is retryable;
    * 409 is retryable on the `openai` and `claude` services, as their
      SDKs do.
  * **Timeouts (T12, T13):** D2.
* **Phase 2, answers:**
  * **Tool arguments (W1):** decoded as `json.RawMessage` and compacted;
    empty or `null` becomes `"{}"`; `ToolArguments` passes valid JSON
    through; Chat Completions accepts a string or an object.
  * **Finish reasons (W3):** one `wire.Finish` helper that keeps unknown
    values verbatim, maps a reply with a call to `tool_calls`, treats
    `model_context_window_exceeded` as `length`, puts the reason of an
    empty answer in `APIError.Reason`, and always emits `RoleAssistant`.
  * **Responses (W4):** an empty output is `ErrIncomplete`; a `refusal`
    part is `FinishContentFilter` with its text.
  * **Gateway errors (W5):** a top-level or choice-level `error` in a 200
    is classified with `ClassifyStreamFailure`.
  * **Gemini call ids (W10):** `functionCall.id` is decoded, else
    `name#index`.
  * **Reasoning:** D4 (W8) and D5's `ReasoningItem.Format` (W7).
  * **Kilo (W9):** `reasoning_details` is kept and replayed, but only after
    a live check shows Kilo returns it; measured first, like 0020 F24.
* **Phase 3, credentials:**
  * **Token-aware invalidation (T1):** D5.
  * **Early refresh (T2):** the early-refresh mark is soft. A failure other
    than `ErrAuthFailure`, with more than 30 s left, returns the current
    token. A failed refresh waits 10–30 s before the next.
  * **Short-lived tokens (T3):** the margin is min(5 min, lifetime/2).
  * **The refresh decode (T4):** tolerant (`oauthSeconds`) and bounded
    (`oauthResponseLimit`). A 200 that carries a refresh token is always
    kept.
  * **The lock (T8, T9, T10):**
    * each refresh attempt is bounded at 15 s;
    * staleness is "mtime unchanged for `staleAfter` on the waiter's
      monotonic clock";
    * `lockWait` is greater than `staleAfter` plus the heartbeat;
    * the heartbeat stops only when the owner token no longer matches;
    * a failed takeover rename is retried;
    * a pending re-save tries the lock once, without waiting, at most every
      30 s.
  * **Device logins (T11):** transport errors, 429 and 5xx keep polling,
    with backoff up to the code's expiry (RFC 8628 §3.5).
  * **The token directory (T15):** after `MkdirAll`, the directory must be
    the user's, with no group or other write bit. It is tightened when the
    user owns it, and refused otherwise. This adds to 0010-MADR D10.
* **Phase 4, catalog:**
  * **The metadata cache (C1, C2):** one fetch per URL at a time, run
    under `context.WithoutCancel` with its own timeout; each waiter honours
    its own context; a stale document is served while one background
    refresh runs; a failure is cached only when the library's own timeout
    caused it.
  * **Read limits (C3):** 32 MiB for the metadata document, 8 MiB per
    listing page; truncation is a clear decode error.
  * **Ranking (C4, C5, C6, C7, C8, C9):**
    * OpenAI's filter denies the tokens `tts`, `transcribe`, `search`,
      `deep-research` and `realtime`;
    * name rules match tokens, not substrings;
    * generations are parsed numerically, with the shut-down list kept;
    * D3's fill and grouping;
    * a negative or non-finite cost and a future date are unknown, and
      `kiloPriceRank` reuses `kiloPrice`.
  * **Search (C11, Z11):**
    * the label tier matches the display name only;
    * a separator-free substring tier is added;
    * the subsequence tier runs only when no stronger tier matched, and
      for queries of 3 characters or more;
    * ties break by listing order;
    * glob results are ranked;
    * the wizard offers a query with no match as the model id.

    `Search`'s documented order changes with its doc.
  * **Helpers and labels (C12):** `Rank` folds the provider's case and
    knows Together; every static id has a label, or is on an explicit
    no-label list; orphan labels go.
  * **Empty curation (C13):** a live listing that curates to nothing keeps
    `Live=true` and its `Usable`, with an empty `Recommended`; callers
    already fall back to `Static`.
  * **The route table (C10, W13):** built at init from an embedded copy of
    the snapshot, with a leave-one-out gate that the heuristic agrees with
    every row.
* **Phase 5, wizard and redaction:**
  * **Redaction cost (Z1):** the message is cut to 16 KiB before
    redaction, then bounded at 512 bytes; each regex is gated on its anchor
    strings.
  * **Redaction coverage (Z2):**
    * camelCase token keys, `private_key` and `signature` are covered;
    * a PEM block goes from BEGIN to END;
    * each cookie pair, and quoted values to the closing quote, are
      redacted;
    * the bare `code` and `token` keys skip snake_case words and plain
      numbers;
    * a property test plants each secret class in random contexts.
  * **Control characters (Z3):** C0 and C1 control characters, except
    `\t`, are stripped from `Message`, `Code` and the callback `code`, with
    `Code` bounded; `TextPrompter.Notify` sanitises as well.
  * **Masked entry (Z6):**
    * redraw only when no input is buffered;
    * pad to a fixed width instead of `\033[K`;
    * a lone ESC is nothing;
    * Ctrl-U clears the entry;
    * Ctrl-C wraps `context.Canceled`.

    It is checked on Windows on the owner's machine.
  * **Base URLs (Z7):** the URL is parsed and needs a scheme and a host;
    userinfo, a query and a fragment are refused; `http` to a non-loopback
    host asks to confirm.
  * **Pasted credentials (Z10):** a pasted ChatGPT token must be JWT-shaped
    and not expired. Another vendor's key prefix is named in the error.
* **Phase 6, tooling and tests:**
  * **Dependencies (Z4):** dep-check also checks `go.mod`'s requirements
    against the allowlist, and test builds with `go list -test -deps`.
  * **Gates (Z9):**
    * generate-check reads `-output=` and fails when it checks no file;
    * parity-check needs a resolving Go name or an explicit "no
      equivalent" marker in each cell;
    * each gate script gets a planted-failure self-test.
  * **CI and floors (Z5):**
    * a govulncheck step, and a weekly scheduled run;
    * `-shuffle=on`, `timeout-minutes: 30`, and a `concurrency` group;
    * the precheck adds `-race` and `go mod tidy -diff`;
    * each coverage floor is raised to its measured value minus 2 points,
      which this record authorises (`AGENTS.md`).
  * **The test fake (Z8):** a Cleanup check fails on unused scripted
    answers, with an explicit opt-out for tests that end early; the stale
    scripts are fixed.
  * **Timing tests (Z12):** the F14 tests observe the join through a hook
    or `testing/synctest`; the `CommandToken` tests run in parallel.
  * **The harness (W12):** D5.

### 3. Not adopted

* **A pid and host liveness check on the lock** (T9, optional): the
  monotonic staleness rule suffices; reconsider if a crash recovery is slow
  in practice.
* **Typed request structs instead of `map[string]any`**: a 100-turn,
  350 KB body encodes in 0.63–1.0 ms. Only W1's piece is worth doing.
* **Faster search**: about 0.45 ms over 400 ids on an interactive path.
* **The remaining 30 linters tried**: noise (`exhaustive` 18 hits, all
  deliberate partial switches). Only `nolintlint` is adopted, in Phase 6,
  for its 2 stale directives (`catalog/models_catalog.go:675`,
  `providers/opencode/route.go:75`).

### Consequences

* Good, because the bugs of Table A are fixed: altered tool arguments,
  silent empty successes and refusals, shared call ids, lost rotated
  tokens, terminal injection.
* Good, because long ChatGPT answers complete under the same limit as the
  ChatGPT client itself, and a dropped reply costs at most one more
  generation.
* Good, because four gates that could pass while broken are fixed, and
  each gate script proves itself against a planted breach.
* Good, because stateless reasoning tool loops keep their reasoning on
  OpenAI, Grok and OpenCode (D4).
* Bad, because with the 330 s total gone, a server that keeps sending data
  forever is bounded only by the caller's context. The idle limit covers
  the realistic failure, a stalled connection, as the references do.
* Bad, because D1 still pays for two generations when a reply is lost
  twice, and pays twice where the first reply had already completed
  upstream.
* Bad, because `Rank` and `Search` return different values and orders.
  Their signatures do not change, but a caller that pinned their output
  sees a change.
* Neutral, because D4 and W11's nil-schema rule change goldens, each
  listed in the PLAN with its reason.

### Confirmation

* **Red first:** each finding has a test that fails on a scratch copy
  before its fix, with its FAIL line and then its PASS line in the PLAN's
  execution record. A gate change is seen to fail on a planted breach.
* **The gate** passes at the end of each phase, with `api-check` reporting
  only additive changes against `v1.1.0`.
* **D3's condition:** under the utility profile, a Kilo listing's
  `kilo-auto/*` tiers are absent from `Recommended`, present in `Usable`,
  and found by `Search` and the glob `kilo-auto/*`. This holds both in a
  full listing and in one that holds only those tiers (C13).
* **Live checks**, with the owner's keys and sessions:
  * a ChatGPT-session stream checked against the idle limit, with a long
    reasoning prompt;
  * D4 on OpenAI with `WithStore(false)` and on an OpenCode
    responses-route model: encrypted reasoning returned and replayed;
  * W9's Kilo `reasoning_details`, before it is wired;
  * Z6 on Windows, on the owner's machine.
* **Not on demand:** Claude's `model_context_window_exceeded` with a tool
  call, and a real 409 or `x-should-retry: true`. These are tested from
  the shapes the sources give and recorded as unmeasured live.

## Pros and Cons of the Options

### Remediate in a 0021 PLAN, ordered by severity, with the owner's five decisions

* Good, because one record traces each finding to its fix and to the
  decisions it amends.
* Good, because the owner's decisions are recorded with their evidence.
* Neutral, because it is a large PLAN, mitigated by six phases that each
  ship on their own.
* Bad, because D1 and D2 amend two accepted records (0012, 0020), which
  this record must say plainly; it does, in §1.

### Fold each finding into the record that owns its area

* Good, because each record stays complete for its area.
* Bad, because cross-cutting changes (the shared request path, D1, D2)
  would be split across four records, with no single PLAN to prove them.

### Fix the bugs of Table A only, and record the rest as accepted

* Good, because it is quick.
* Bad, because it leaves the double billing (T5), the 330 s cut (T13), the
  missing connect timeout (T12) and the broken gates (Z4, Z9) in place.

### Defer everything to a v1.2 planning cycle

* Good, because it costs nothing now.
* Bad, because W1, W3 and T4 corrupt data or lose tokens today.

## More Information

### Relationship to other records

* `0009-MADR-use-case-aware-default-model-ranking.md`: §4's fill and
  diversity group are amended by D3.
* `0012-MADR-conform-providers-to-reference-clients.md`:
  * §1.3's 330 s total timeout gives way to D2's 300 s idle limit; the
    300 s header timeout stays;
  * §4.1's retryable "stream ended before completion" is bounded by D1.
* `0015-MADR-canonical-sdk-api-and-module-layout.md`: D8 (retries only
  through `WithRetry`) stands; R48 (additive) governs D5.
* `0016-MADR-provider-auth-and-support-baseline.md`: D8 (one default client
  per provider instance) stands; A2 is why T4 matters.
* `0020-MADR-remediate-v1-debugging-pass-findings.md`:
  * Q2 (a) is amended by D1 for failures after a 200;
  * F5 is completed by C1;
  * F56 and F59 are completed by Z9 and Z8.
* `0010-MADR-windows-stdio-oauth-tokenstore-ci.md`: D10 is extended by T15.

### Research

**Reference clients,** from local checkouts. They are cited by repository,
commit and path, since a relative link cannot reach them.

* **Codex** (`openai/codex` at `33aea33b39`, 2026-10-01):
  * defaults, in `codex-rs/model-provider-info/src/lib.rs:63-65`:
    `DEFAULT_STREAM_IDLE_TIMEOUT_MS: u64 = 300_000`,
    `DEFAULT_STREAM_MAX_RETRIES: u64 = 5`,
    `DEFAULT_REQUEST_MAX_RETRIES: u64 = 4`;
  * the field doc, at `:178-180`: "Idle timeout (in milliseconds) to wait
    for activity on a streaming response before treating the connection as
    lost";
  * the idle limit is applied per event, in
    `codex-rs/codex-api/src/sse/responses.rs:540`:
    `timeout(idle_timeout, stream.next())`;
  * the HTTP client sets no total timeout, only an optional connect
    timeout (`codex-rs/http-client/src/client_builder.rs:143-146,
    357-380`);
  * a dropped stream re-sends the request, up to `stream_max_retries`
    (`codex-rs/core/src/session/turn.rs:1648-1745`);
  * every Responses request includes `reasoning.encrypted_content`
    (`codex-rs/core/src/client.rs:965`).
* **Grok CLI** (`grok-build` at `2bdd1d6a`, 2026-09-29):
  * Messages' `model_context_window_exceeded` is mapped to the `Length`
    stop class
    (`crates/codegen/xai-grok-sampler/src/stream/messages.rs:433-440`);
  * `x-should-retry` is parsed as `true` or `false`
    (`crates/codegen/xai-grok-sampler/src/client.rs:196-206`), and `false`
    stops retrying (`actor/request_task.rs:473`);
  * the sampling retry policy retries 429 and any 5xx
    (`crates/common/xai-circuit-breaker/src/retry_policy.rs:57-65`).
* **OpenCode** (`opencode` at `82ea3a3a63`, 2026-09-30):
  * each Gemini function call gets a distinct id (`tool_N`), not the
    function name (`packages/llm/src/protocols/gemini.ts:441-454`);
  * Gemini finish reasons: `STOP` with tool calls is tool-calls, the safety
    reasons are content-filter, `MALFORMED_FUNCTION_CALL` is an error
    (`:363-377`);
  * a Responses reasoning item is replayed with `encrypted_content` and no
    `id` (`packages/llm/src/protocols/openai-responses.ts:401-405`), which
    matches 0020's live measurement.
* **Kilo** (`kilocode` at `e0c27aa71f`, 2026-09-30): it handles
  `reasoning_details` in `packages/opencode/src/provider/transform.ts`
  (W9, to be measured live).

**Official SDKs,** read on 2026-10-04:

* `openai/openai-python`:
  * `src/openai/_base_client.py`, `_should_retry`: obeys `x-should-retry`
    `true` and `false`, then retries 408, 409 ("lock timeouts"), 429 and
    5xx;
  * `_calculate_retry_timeout`: `min(INITIAL_RETRY_DELAY * 2^n,
    MAX_RETRY_DELAY)` with `jitter = 1 - 0.25 * random()`;
  * `src/openai/_constants.py`: `DEFAULT_MAX_RETRIES = 2`, `DEFAULT_TIMEOUT
    = httpx.Timeout(timeout=600, connect=5.0)`, `INITIAL_RETRY_DELAY =
    0.5`, `MAX_RETRY_DELAY = 8.0`;
  * `src/openai/types/responses/response_output_refusal.py`: `{"type":
    "refusal", "refusal": <str>}` (W4).
* `anthropics/anthropic-sdk-python`, `src/anthropic/_base_client.py`,
  `_should_retry`: the same order and codes, with `x-should-retry` obeyed
  both ways.
* httpx, "Timeouts": the read timeout is "the maximum duration to wait for
  a chunk of data to be received", an idle limit, not a total.

**What remains unmeasured:**

* Claude's `model_context_window_exceeded` with a tool call;
* OpenAI's refusal part in a live reply;
* Kilo's error envelope in a 200;
* `reasoning_details`;
* a ChatGPT stream past 330 s.

Each is tested from the documented shapes, and live where a check can be
run (Confirmation).

### Review provenance

The four reviews' scratch tests and benchmarks live outside the
repository and are not committed. Each finding's red test is written in the
PLAN from the evidence above, so nothing here depends on the scratch files.

## Amendment 2026-10-04: facts corrected while writing the PLAN

Five read-only reviews checked every finding against the code at `6be2d80`
while 0021-PLAN was written. No decision changes. These facts are
corrected, and the PLAN follows them:

1. **Paths.** In the tables, `auth/`, `catalog/`, `providers/`, `llmtest/`,
   `internal/wire/` and `internal/transport/` are under `llmprovider/`.
   `internal/redact/`, `wizard/` and `scripts/` are at the repository
   root.
2. **Line drift:**
   * T1's `reauth.go:17-28` is 17-29;
   * W1's Messages re-marshal is `messages.go:164-178`;
   * C5's `650-665` is `rankKiloModel`, which has the same `mini` defect,
     so C5 covers both rankers;
   * C7's `discovery.go:525-528` is Go's fallback closure (519-529), chosen
     at `model_metadata.go:364-366`;
   * C10's table is `route.go:76-203`;
   * Z1's regexes are `redact.go:27-68` and `Redact` is 70-98. The 512-byte
     bound is `api_error.go:59` (`boundMessage`), not the redact package;
   * Z4's check is `check_deps.py:47-61`.
3. **D4's goldens.** The existing `openai` and `grok` golden cases are
   built without `WithStore`, so they do not change. New `openai-stateless`
   and `grok-stateless` cases record D4. The `opencode-*-responses`
   goldens change.
4. **D5's token-aware invalidation.** `wire.Reauth` does not see the token
   today; each send fetches its own. Under the PLAN, the internal `Reauth`
   fetches the token and passes it to the send, so it knows which token
   was refused. This is internal and not API.
5. **C13.** `Catalog` has no `Static` field; `Static` is a function. The
   wizard replaces a catalog whose `Recommended` is empty with the static
   one (`wizard/configure.go:377-378`), so "callers already fall back"
   would undo C13 in the wizard. The PLAN changes the wizard instead: a
   live catalog with an empty `Recommended` and a non-empty `Usable` is
   kept, and the wizard opens search over `Usable`. That keeps §3 item 9,
   since nothing is recommended, and the owner's condition, since the
   tiers can be chosen.
6. **D2 and the auth requests.** `transport.DefaultClient` also serves the
   OAuth refresh, device polls, the token exchange, revocation and Kilo's
   device login. Without the 330 s total these would be bounded only by
   the 300 s header timeout and the caller's context. The PLAN bounds each
   auth request at 30 s, and each refresh attempt at 15 s (T8).
7. **C1's caching rule.** With the fetch detached from every caller's
   context, no caller can make it fail. So "a failure is cached only when
   the library's own timeout caused it" means: every failure of the
   detached fetch (its timeout, the transport, the status, the decode) is
   cached as today, and a waiter's own context ending never is.
8. **Z10.** `ValidateOAuthSession` requires an access-only session to have
   no `Expiry` (`oauth_session.go:64-67`). The paste check reads the JWT's
   `exp` only to refuse an expired token, and stores no expiry, so that
   rule is unchanged.
9. **W11's nil schema.** Chat Completions' `toolList`, `gemini.go:225` and
   OpenCode's Google route (`opencode.go:447`) also send a nil schema as
   `null`, so the shared rule covers every tool encoder. No golden case
   has a nil schema, so the Consequences' "W11's nil-schema rule change
   goldens" does not happen. The stale "1 MiB" comment is also in one
   test, `together/reply_test.go:39`.
10. **`nolintlint`.** A third `//nolint:goconst`, at
    `catalog/models_catalog.go:33`, is checked as well.
11. **D3's fill.** Items 7 and 8 are inline in `eligible`
    (`model_ranking.go:115, 118`), with no named predicate. The PLAN names
    one, `excludedAtRequest`, for the fill and for Go's fallback.
12. **Parity markers.** The guide already marks a cell with no equivalent
    as `none`, `removed` or `Here`. These are the explicit markers
    parity-check accepts (Z9).
13. **D1's budget.** The one retry after a reply counts against
    `MaxAttempts`, and happens at most once whatever the budget. With
    `MaxAttempts: 1` nothing is retried, as before.

## Amendment 2026-10-04: two live failures added as phase 7

Found by 0021-PLAN's phase 1 live suite (V3.3). Both tests fail the same way
on `6be2d80`, so neither comes from this record's changes. The owner added
them to this record's scope on 2026-10-04 ("added as extra phase, proceed").
0021-PLAN carries them as phase 7.

| ID | Location | Finding | Evidence |
| :--- | :--- | :--- | :--- |
| L1 | `llmprovider/live_opencode_test.go:175` | **The metadata route test pins stale data.** It assumes OpenCode Go's `qwen3.8-max` has no `provider.npm`, so it routes to chat. The live models.opencode.ai document of 2026-10-04 gives it `@ai-sdk/anthropic`, and the provider correctly follows it to `/messages`. The provider is right; the test is stale. | reproduced live, on `HEAD` and the tree |
| L2 | `llmprovider/live_grok_test.go:136`; `providers/grok/grok.go` `body` | **Grok ignores the instructions.** `grok-4.6` answered "Hello!" to a request whose leading system message said to reply only "OMEGA". Grok sends `Instructions` as a system message in `input`. Whether xAI now wants the top-level `instructions` field, or the model changed, is measured first. | reproduced live, twice, on `HEAD` |

## Amendment 2026-10-05: F15 and A6 under the shared fetch

Found while building 0021-PLAN step 4.1; resolved by the owner on
2026-10-05.

* **0020-MADR F15** guarded against a fetch failing after another fetch of
  the same URL had cached a newer document. With C2's one fetch per URL
  running at a time, that race cannot occur. F15's test now checks the
  shared fetch: a second lookup joins the one in flight, and a document
  already cached is kept when the fetch fails.
* **0013-MADR A6** bounds the lookup a request makes at 5 s. That stands:
  each lookup waits at most `metadataLookupTimeout`. The fetch it waits
  for is detached and has its own `metadataFetchTimeout` of 10 s, so a slow
  host still fills the cache for the next caller. A6's test checks the
  fetch's bound.

## Amendment 2026-10-05: the gpt-6 family on OpenAI's listing

Found while building 0021-PLAN step 4.3; decided by the owner on 2026-10-05.

* **New fact.** `isUsableOpenAIChatModel` allows only the prefixes in
  `openaiAllowPrefixes` (`catalog/models_catalog.go:170-173`), which has no
  `gpt-6`. OpenAI's live listing therefore drops the whole `gpt-6` family,
  and `rankOpenAIModel` has no case for it.
* **C4 is widened.** Besides the token denies, `gpt-6` joins the allow
  list, and `rankOpenAIModel` scores it as it scores `gpt-5`.

## Amendment 2026-10-05: C6's anchors are capped

Found while building 0021-PLAN step 4.5; decided by the owner on 2026-10-05.

* **New fact.** Claude's generation hints mix family bonuses, so as anchors
  they do not rise with the generation: 40 at 4.0 (`haiku-4`), 35 at 4.6
  and 4.8, 40 at 5. Taken as written, an old id such as `claude-opus-4-1`
  would outrank `claude-opus-4-8`.
* **C6 is refined.** An id that no hint matches takes the highest anchor at
  or below its generation, capped at one less than the lowest anchor above
  it, so it never outranks a newer hinted generation. The rules above every
  anchor and below every anchor are unchanged.

## Amendment 2026-10-05: Z2 also covers the Google key boundary

Found by 0021-PLAN step 5.2's property test; decided by the owner on
2026-10-05.

* **New fact.** The Google API key rule, `\bAIza[0-9A-Za-z_-]{35}\b`,
  misses a key whose last character is `-` or `_`: the trailing `\b` needs
  a word character on one side. Such a key leaks whole.
* **Z2 is widened.** The rule drops the trailing `\b`.

## Amendment 2026-10-05: L3 added to phase 7

Found by 0021-PLAN's phase 2 live suite (V3.3); investigated and decided by
the owner on 2026-10-05 ("Fix the classification").

| ID | Location | Finding | Evidence |
| :--- | :--- | :--- | :--- |
| L3 | `llmprovider/api_error.go:301-306`, `statusSentinel` (`:176`) | **An OpenCode 403 that names no known type is reported as `ErrAuthFailure`.** On 2026-10-04 OpenCode Go answered `HTTP 403 "Your organization does not have access to this model"` for `gpt-6-luna`, and five live tests failed instead of skipping. OpenCode answers a bad key with `401 "Invalid API key."` (`0004-PLAN-add-gateway-llm-providers.md`, deviation D3), so a 403 there is a valid key that is refused: an entitlement, which `ErrNotPermitted` is documented for. `ErrAuthFailure` tells a caller to sign in again. | live 2026-10-04, failing on `HEAD` and the tree alike; cleared by 2026-10-05, when a probe and two runs of the five tests passed |

* **Decided:** on OpenCode, a 403 is `ErrNotPermitted`, terminal, whatever
  its body, as Kilo's 403 already is. A 401 stays `ErrAuthFailure`.
* `0012-MADR-conform-providers-to-reference-clients.md` §1.1's table gains
  the row, by an amendment there.
* **Considered, not chosen:**
  * an entitlement-aware live model picker, which probes each candidate and
    falls through on `ErrNotPermitted`: it would keep the responses route
    tested whatever the account may use, at a billed request per candidate
    per run;
  * recording L3 as resolved with no change, which leaves the
    misclassification in place.
* The live picker, `liveModel`, reads the public metadata document, which
  cannot show a per-account entitlement. With the new row, such a refusal
  skips through `SkipIfTransient`.

## Amendment 2026-10-05: the empty finish reason and the cut answer's reason

Found by 0021-PLAN step 6.6's conformance checks; decided by the owner on
2026-10-05.

* **W3 is widened.** A reply that carries a call and has no finish reason
  finishes `tool_calls`, as one finishing `stop` already did. Six providers'
  call replies with no finish field returned `FinishReason ""`.
* **`APIError.Reason` stays the service's own reason,** as its doc says:
  `max_output_tokens` on the Responses wire, `incomplete` on Gemini's
  Interactions, `length` where the wire already reports that. W12's
  `Truncated` check pins each wire's value through the harness field
  `TruncatedReason`, rather than requiring `length` everywhere. Normalising
  `Reason` was considered, and not chosen: a caller matching
  `max_output_tokens` today would stop matching.

## Amendment 2026-10-05: W14 found by the conformance checks

Found by 0021-PLAN step 6.6's `Truncated` check on the ChatGPT stream;
decided by the owner on 2026-10-05 ("Fix it in phase 6").

| ID | Location | Finding | Evidence |
| :--- | :--- | :--- | :--- |
| W14 | `internal/wire/responses/responses.go:287-289, 308-310` (`ReadStream`) | **A failed stream event returns the partial response with its error.** On `response.incomplete`, `response.failed`, an `error` event, or one that does not decode, `ReadStream` returns `result, err`, so the ChatGPT path's `Generate` gives a response and an error at once, which R7 forbids. A caller that reads the response first takes a cut or failed answer for a success. Present since `940fee0`, before `v1.1.0`. | reproduced by the W12 check |

* **Decided:** a failed dispatch returns no response, as `Decode` does.

## Amendment 2026-10-05: L2 is the test's prompt, not the wire

Measured by 0021-PLAN step 7.2a; decided by the owner on 2026-10-05 ("Keep
wire; paired test").

* **The probe,** on a scratch copy, three runs of each, recording only
  whether the reply contains `OMEGA`:

  | Instruction | Form | `grok-4.6` | `grok-4.7` |
  | :--- | :--- | :--- | :--- |
  | "Whatever the user says, reply with only the word OMEGA." | (a) a leading system message in `input` | 0/3 | 0/3 |
  | the same | (b) the top-level `instructions` field | 0/3 | 0/3 |
  | the same | (c) in the user's own message | 1/3 | 0/3 |
  | "Begin every reply with the word OMEGA." | none (baseline) | 0/3 | 0/3 |
  | the same | (a) | 3/3 | 3/3 |
  | the same | (b) | 3/3 | 3/3 |

  `grok-4.7` is the newest model in the live listing of 2026-10-05.
* **So L2's finding is corrected:** Grok does not ignore the instructions.
  Both forms reach the model and are obeyed. The models now decline an
  instruction phrased as an override of whatever the user says, in every
  position, the user's own message included.
* **Decided:** the wire is unchanged: `Instructions` stay a leading system
  message, and no golden changes. `TestLive_GrokInstructions` sends the same
  request with and without the neutral instruction: the reply must contain
  `OMEGA` with it, and must not contain it without it, as the probe
  measured. The pair
  shows the instruction changed the reply, which the old single request
  could not.
* **Not chosen:**
  * the top-level `instructions` field: both forms measured 3/3, so nothing
    favours a wire change;
  * keeping the override prompt and recording L2: the test would stay red.

## Amendment 2026-10-08: Z1's bound is 2 KiB, and Z2 keeps a code's diagnostic words

Made by 0028-PLAN-heuristics-and-performance-from-the-research-pass.md,
Phase 4 (0028-MADR D-A10; deviation D2, A10b, chosen by the owner, "Phase 4,
after equivalence").

* **Z1 is tightened.** An error message is cut to 2 KiB before redaction,
  not 16 KiB, and the cut goes back to the last space, tab, newline, comma,
  semicolon or quote before the bound, then to a rune start, so a secret
  that straddles the bound is dropped whole rather than leaving a fragment
  that no rule recognises. The 512-byte bound on what is kept is unchanged.
  A JSON 400 with a 16 KiB message took 12.7 ms to classify, almost all of
  it redaction and the overflow check; the bound and a literal prefilter on
  the overflow forms bring it under 1 ms.
* **Z2 is widened, for `code` only.** A `code` value made of two or more
  letter-only words joined by `-` or `_`, in any case and at most 40 bytes,
  is a diagnostic and is kept, such as xAI's `invalid-argument` and Kilo's
  `INVALID_TOKEN`. A bare `key` or `token` keeps only Z2's rule, a number or
  lower-case snake case: a random value with no digit can take the
  letter-words shape, and a key's value is a secret far more often than a
  code's. Any value holding a digit is still masked.
