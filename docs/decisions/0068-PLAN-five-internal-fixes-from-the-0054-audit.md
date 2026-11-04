---
status: in-progress
date: 2026-11-03
associated-madr: "0068-MADR-five-internal-fixes-from-the-0054-audit.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Five Internal Fixes from the 0054 Audit

Associated MADR: [0068-MADR-five-internal-fixes-from-the-0054-audit.md](0068-MADR-five-internal-fixes-from-the-0054-audit.md)

## Goal

The MADR's D1 to D6, as the owner answered its questions on 2026-11-03 (Q1
30 s, Q2 5 s, Q3 four fifths, Q4 60 s, Q5 `v1.26.0`):

* the shared transport pings a silent HTTP/2 connection, and a dead one
  fails its request within the ping timeouts, not the 300 s header timeout
  (F1);
* a refused certificate, HTTP 501 and HTTP 505 are not retried, by
  `WithRetry` or by an OAuth refresh (F2);
* `lockWait` exceeds the longest refresh hold, and a test holds the order
  (F3);
* `CommandToken` keeps a failed run's error for the window, and starts one
  early run near the end of its TTL (F4);
* `reAuth`'s pass finds its keyword first, with output identical to the
  expression's own search and a faster benchmark (F5);
* no exported name added or changed, no request changed, released as the
  owner answers Q5.

## Scope

### In scope

* **Phase 1, the probe:** a scratch program outside the tree. No file in
  the tree changes.
* **Phase 2, the transport (D1):** `llmprovider/internal/transport/transport.go`,
  a new test file beside it, and `llmprovider/options.go` (`NewHTTPClient`'s
  doc).
* **Phase 3, retries (D2):** `llmprovider/retry.go`,
  `llmprovider/internal/transport` (one function that reports a refused
  certificate), `llmprovider/auth/oauth_session.go`,
  `llmprovider/api_error.go`, and their tests.
* **Phase 4, auth and keys (D3, D4):** `llmprovider/auth/tokenstore_file.go`,
  `llmprovider/command_token.go`, and their tests.
* **Phase 5, redaction (D5):** `internal/redact/redact.go` and its tests.
* **Phase 6, docs and the release:** `docs/architecture.md`,
  `CHANGELOG.md`, the README's Status if it names a version, and the tag,
  which is the owner's.
* **Records:** this PLAN's execution record; the MADR's amendments if a fact
  changes; the dated amendments to the records the MADR's "More
  Information" lists.

### Out of scope

* Any exported name added, changed or removed (R48, R55):
  `tools/apidiff.py v1.25.0` reports nothing.
* Any request's content (R45): G-wire's goldens do not change.
* A TLS session cache (the report's 5.1, second half): TLS 1.3's resumption
  saves no round trip, and nothing measured what it would save.
* NXDOMAIN and Anthropic's spend cap (MADR D2), refresh jitter,
  `Retry-After` on a refresh and a blocking lock (D3), and
  `VendorCLISession`'s file (D4).
* Every other idea in
  `0054-REPORT-heuristics-and-performance-opportunities.md`.

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase starts after the owner has committed the
  last.
* **The gate, before staging:**
  * `python3 -B tools/gate.py`, or `--markdown` for a records-only phase;
  * `make pre-add-check FILES="…"` on the phase's Go files;
  * `tools/claims.py` on the MADR and on the report;
  * `make records-check`;
  * `tools/recordlint.py --changed`;
  * `tools/denyscan.py --changed`;
  * `tools/linkcheck.py`.

  Each exit status is captured before any filter, and read before anything
  is written from it.
* **Red first.** Every new test is seen to fail at the code before its fix,
  by `tools/redgreen.py SPEC.json --expect fail --rev HEAD`, which copies
  the new test files onto a clone of `HEAD`. Every test is then seen to
  fail on a planted return of its defect, by `tools/plant.py` on a scratch
  clone, never in the tree. The phase's record holds each failure line.
* **The docs.** A function whose behaviour changes has its doc changed in
  the same phase, naming only what a test shows (R52). `make docs-check`
  stays at 0.
* **The API and the wire.** `tools/apidiff.py v1.25.0` reports no change;
  G-wire's goldens do not change; `make coverage-check` holds every floor
  (R47).
* **The existing tests are the drift check.** Every test at `d65e9d1`
  passes unchanged, except one a step names as changing, with the reason.
* **Other platforms.** Phases 2 to 4 change networking, TLS errors and a
  file lock, which can differ by operating system. Before a push, the
  phase's packages' tests run on the Linux and Windows dev hosts, in a
  checkout of the pushed commit, and the record says what ran and where, by
  platform, never by host name.
* **Stage, then the owner commits** with `git commit --no-edit`. Push and
  tags happen on the owner's word, after the disclosure guard has run over
  the outgoing commits.
* **Stop conditions:**
  * a fact of the MADR, or one of its claims, no longer holds;
  * Phase 1 finds a host that refuses pings at the chosen interval;
  * a step would add or change an exported name, or change a golden;
  * an existing test would need to change for a reason no step names;
  * D5's benchmark is not faster, or its fuzz test finds a difference.

  On any of these, stop, record a dated deviation here (with an amendment in
  the MADR when a decision or a fact changes), and wait.

## Implementation Steps

### Phase 0: records

1. This PLAN, the MADR, `0054-REPORT-ideas-status-at-v1-25-0.md` and their
   rows in `docs/README.md`, committed as records only.
2. The owner's answers to Q1 to Q5, given 2026-11-03, are in the MADR's
   questions. Once the owner approves this PLAN, `tools/records.py status`
   sets the MADR `accepted` and this PLAN `in-progress`.
3. A dated amendment, without rewriting the decision, in each record the
   MADR's "More Information" lists:
   * 0021-MADR D2: the health check (MADR D1);
   * 0026-MADR F11: a refused certificate is not a failure to reach the
     service (D2);
   * 0012-MADR §1.1: 501 and 505 are terminal (D2);
   * 0021-PLAN's `lockWait`: 60 s, past the longest refresh hold (D3);
   * 0017-MADR D3: the failure memory and the early run (D4);
   * 0028-MADR D-A10: `reAuth` keyword-first (D5); and its shortlist, that
     H6 and H9 are taken up by 0068-MADR.

### Phase 1: probe each API host's tolerance of pings (D1)

No file in the tree changes.

1. **The hosts:** every `https://` host the module's non-test Go code names
   as a default base URL, an OAuth issuer, or a listing or metadata source,
   found by a search of `llmprovider/` and `wizard/`, and listed in the
   record.
2. **The program,** in the session's scratch directory: for each host, a
   client with `ForceAttemptHTTP2`, `HTTP2Config{SendPingTimeout: Q1's
   interval, PingTimeout: 15 s}` and an idle timeout of 10 minutes, sends
   one unauthenticated `GET` to the host's root, reads the reply, keeps the
   connection idle for 6 minutes, and sends a second `GET`. It runs with
   `GODEBUG=http2debug=2`, its log in the scratch directory. No key is sent
   or read.
3. **What is recorded,** per host: the protocol; the PINGs sent and
   acknowledged; any GOAWAY and its error code; when the server closed the
   connection, if it did; whether the second `GET` reused the connection.
4. **The rule:** a GOAWAY with `ENHANCE_YOUR_CALM`, or a connection closed
   after a PING and before its idle time, is a refusal, and stops the work.
   A GOAWAY with `NO_ERROR` after the server's own idle time is recorded,
   not a refusal.
5. **The limit,** stated in the record: the probe tests an idle connection.
   A connection with a stream open, a long generation's wait, is not
   measured, because a generation that long is billed. Under gRPC's default
   keepalive policy, the one known to send `ENHANCE_YOUR_CALM`, a connection
   with no call open accepts no pings at all
   ([grpc/grpc doc/keepalive.md](https://github.com/grpc/grpc/blob/master/doc/keepalive.md)),
   so the idle case is the stricter of the two there.

### Phase 2: the transport's health check (D1)

1. `transport.Config` gains `SendPingTimeout` and `PingTimeout`;
   `defaultConfig` takes Q1's interval and 15 s; `manager.transport` sets
   `HTTP2` when `SendPingTimeout` is above zero. The doc of `DefaultClient`
   names the health check with the other bounds.
2. **Tests,** in a new `transport_0068_test.go`:
   * `TestDefaultClient_HTTP2HealthCheck_0068`: `DefaultClient`'s transport
     (`Base()`) carries `HTTP2.SendPingTimeout` and `PingTimeout` as set;
   * `TestDeadHTTP2Connection_FailsWithinPing_0068`: the report's P2 as a
     test, through `manager.transport` with a test `Config` (1 s and 1 s, a
     header timeout of 30 s): the second request fails in under 10 s, and its
     error is not the header timeout's.
3. `llmprovider/options.go`: `NewHTTPClient`'s list of bounds adds "a
   silent HTTP/2 connection pinged after Q1's interval and closed when the
   ping is not answered within 15 s".
4. **Red and plants:** both tests red at `HEAD`; plants: no `HTTP2` field
   set; `SendPingTimeout` zero in `defaultConfig`.

### Phase 3: what is not retried (D2)

1. `transport.CertificateRefused(err error) bool`: whether err's chain holds
   a `*tls.CertificateVerificationError`.
2. `retryable` (`llmprovider/retry.go`): returns false when
   `CertificateRefused`, before the kind switch. `WithRetry`'s and
   `RetryPolicy`'s docs name a refused certificate among the failures not
   retried.
3. `refreshOAuthSessionOnce`: `retry` is false when `CertificateRefused`.
4. `classifyAPIError`: 501 and 505 return terminal, with
   `ErrProviderUnavailable`, beside 525 and 526.
5. **Tests:**
   * `TestWithRetry_RefusedCertificateOnce_0068`: a provider over an
     untrusted `httptest` TLS server, under `WithRetry` with `MaxAttempts`
     3: one handshake (counted by the server's `ConnState`), and the error
     holds the verification error;
   * `TestOAuthRefresh_RefusedCertificateOnce_0068`: likewise, one
     handshake;
   * `TestClassify_501And505Terminal_0068`: an `APIError` from each is not
     `Retryable`, and matches `ErrProviderUnavailable`;
   * the existing classification table gains the two rows, if it has a
     5xx row.
6. **Red and plants:** each test red at `HEAD`; plants: the check after the
   kind switch (the refresh's wrapped error retried); the refresh loop's
   check removed; 501 dropped from the terminal row.

### Phase 4: the lock wait and the command token (D3, D4)

1. `lockWait`: Q4's value. `TestLockWait_OutlastsLongestRefresh_0068`
   computes `oauthRefreshAttempts × oauthRefreshAttemptTimeout` plus each
   backoff, not in parallel with a test that shortens them, and requires
   `lockWait` to exceed it.
2. `CommandToken`:
   * a failed run whose error matches `ErrAuthFailure`, and which was not
     abandoned, is kept with its error and its time; `Token` within Q2's
     window returns it without running; `Invalidate` clears it;
   * `Token` finding the output older than Q3's fraction of its TTL, with no
     run in flight, starts one run on `context.WithoutCancel(ctx)` bounded
     by `Timeout`, and returns the output; the run's success replaces it, its
     failure is dropped;
   * the type's doc says both.
3. **Tests,** with the type's clock seam and a command the test builds:
   * `TestCommandToken_RemembersFailure_0068`: a failing command runs once
     for two `Token` calls inside the window, again after it, and again after
     `Invalidate`;
   * `TestCommandToken_AbandonedNotRemembered_0068`: a run whose caller's
     context ended is run again by the next caller;
   * `TestCommandToken_EarlyRun_0068`: past the fraction, `Token` returns the
     old output at once while one run starts, and the next `Token` after it
     returns the new output; two callers start one run.
4. **Red and plants:** each test red at `HEAD`; plants: `lockWait` back to
   40 s; the failure kept with no window; the early run on the caller's
   context; two early runs allowed at once.

### Phase 5: `reAuth` keyword-first (D5)

1. **Before:** `BenchmarkRedact_16KiBTokens` and `BenchmarkRedact_Clean4KiB`,
   `-count=10`, once `tools/gate.py`'s load wait passes, saved in the scratch
   directory.
2. `redactAuthPass`, the pass's `apply`, as MADR D5 describes, beside
   `redactKVPass`, with a `reAuthAnchored` built from `reAuth` as
   `reKVAnchored` is.
3. **Tests:**
   * `FuzzRedactAuthPass_MatchesExpression_0068`: on any input, the pass
     equals `replaceSubmatches(reAuth, …, redactAuth)`, seeded with
     `equivCorpus()` and with label, case and spacing cases;
   * every existing redaction test passes unchanged;
   * the fuzz test runs for 5 minutes on a scratch clone, and the record
     says how many inputs it tried.
4. **After:** the same benchmarks, with `benchstat`-style medians in the
   record. Not faster is a stop.
5. **Plants:** the anchored try skipped for `authorization` (a label not
   kept); the non-ASCII fallback removed (the Kelvin-sign case); the skip
   inside the last match removed.

### Phase 6: docs and the release (D6)

1. `docs/architecture.md`: the transport's health check; what `WithRetry`
   does not retry; `CommandToken`'s memory and early run.
2. `CHANGELOG.md`, under Unreleased: Changed (D1, D2's 501 and 505, D3, D4);
   Fixed (F2's certificate, F3).
3. The report's P1 and P2, run again against the result, their lines in the
   record.
4. The Linux and Windows dev hosts: `go test ./llmprovider/...
   ./internal/...` at the phase's commit.
5. The release: the owner tags Q5's version; this PLAN goes `complete` once
   the tag's CI and its GitHub Release are seen green.

## Verification

* Each phase: its tests red at `HEAD` and green after; each plant caught;
  `tools/gate.py` passed; `tools/apidiff.py v1.25.0` reports nothing.
* Phase 1: every host's line recorded, none a refusal.
* Phase 5: the fuzz test clean for 5 minutes; the benchmark faster.
* Phase 6: P1 makes one attempt; P2 fails within the ping timeouts; the
  tag's CI is green on all three operating systems.

## Rollout and Rollback

* **Rollout:** each phase lands on `main` as it passes; only the tag waits
  for Phase 6.
* **Rollback:** each phase is one commit and reverts alone. D1 also has a
  value to turn off: `SendPingTimeout` zero in `defaultConfig` restores
  today's transport, should a host start refusing pings after the release.

## Execution record

### Phase 0: records (2026-11-03)

* **Step 1:** the report, the MADR, this PLAN and their rows, committed by
  the owner as `ccc6486`; the owner's answers to Q1 to Q5, as `925c1cb`.
* **Step 2:** the owner accepted the MADR and approved this PLAN on
  2026-11-03 ("accept the madr, approved to proceed"). `tools/records.py
  status` set the MADR `accepted` and this PLAN `in-progress`, each with its
  row in `docs/README.md`.
* **Step 3:** a dated amendment of 2026-11-03 at the end of each record the
  MADR lists, none rewriting the decision it amends:
  * 0021-MADR: D2 gains the health check; T9's `lockWait` is 60 s;
  * 0026-MADR: F11 does not cover a refused certificate;
  * 0012-MADR: §1.1's table gains 501 and 505, terminal;
  * 0017-MADR: D3's `CommandToken` remembers a failure and runs early;
  * 0028-MADR: D-A10 extends to `reAuth`; H6 and H9 are taken up;
  * 0021-PLAN: a note that `lockWait` is 60 s.
* **Not done here:** Phases 1 to 6. Phase 1 starts after the owner has
  committed this phase.

### Phase 1: probe each API host's tolerance of pings (2026-11-03)

The owner committed Phase 0 and said "proceed". No file in the tree changed
in this phase but this record.

* **The hosts,** from a search of the non-test Go code in `llmprovider/` and
  `wizard/` for `https://` hosts, leaving out the doc examples `host` and
  `example.com`: `api.openai.com`, `auth.openai.com`, `chatgpt.com`,
  `api.anthropic.com`, `generativelanguage.googleapis.com`, `api.x.ai`,
  `auth.x.ai`, `accounts.x.ai`, `api.together.ai`, `router.huggingface.co`,
  `api.kilo.ai`, `opencode.ai`, `models.opencode.ai` (the default
  `ModelMetadataURL`).
* **The program,** as step 2 describes, in the session's scratch directory:
  a Go probe and a Python runner that starts one probe per host and reads
  each host's `GODEBUG=http2debug=2` log. No key was sent or read.
* **The refusal rule, seen to fail first.** The log's formats were read from
  a 2 s probe of `api.openai.com` (`wrote PING len=8`, `read PING
  flags=ACK`) and from Go's source for GOAWAY (`net/http/internal/http2/frame.go:1899`,
  `LastStreamID=%v ErrCode=%v Debug=%q`). The runner's self-test passes on
  four planted logs: clean, not refused; `ENHANCE_YOUR_CALM`, refused;
  `NO_ERROR`, not refused; three PINGs unanswered, refused. With the GOAWAY
  check removed it failed: `BAD ENHANCE_YOUR_CALM: refused=False, want
  True`.
* **The run,** 30 s pings and 6 minutes idle, all 13 hosts at once
  (2026-11-04 01:15 to 01:21 UTC):
  * no `ENHANCE_YOUR_CALM` from any host;
  * 9 hosts acknowledged every PING and kept the connection for the second
    request; `opencode.ai` and `models.opencode.ai` answered over HTTP/1.1,
    so sent no PING;
  * 4 hosts closed with GOAWAY `NO_ERROR`, having acknowledged every PING
    before it: `api.openai.com` 125 s after the first request,
    `generativelanguage.googleapis.com` 240 s (`Debug="session_timed_out"`),
    `api.kilo.ai` 300 s, `api.anthropic.com` 323 s. Each log's one
    unanswered PING is the client's health check, written 30 s after the
    GOAWAY on a connection already closing.
* **The control,** the same four hosts with no PINGs (01:22 to 01:28 UTC):
  `generativelanguage.googleapis.com` closed at 240 s and `api.kilo.ai` at
  300 s, as before: their own idle timeouts. `api.openai.com` and
  `api.anthropic.com` sent no GOAWAY and kept the connection for 6 minutes.

### Deviation 1 (2026-11-03): two hosts closed a pinged connection only

* **Found:** `api.openai.com` and `api.anthropic.com` closed an idle
  connection with GOAWAY `NO_ERROR` when it was pinged every 30 s, and kept
  it when it was not, one sample each. Step 4's rule counts "a connection
  closed after a PING and before its idle time" as a refusal. Each close was
  graceful, and came at least 125 s after the request, past the module's
  own 90 s idle close (`IdleConnTimeout`); a connection with a stream open
  was not measured.
* **Offered:** repeat and settle it (recommended); re-probe at 60 s; drop
  D1.
* **Decision (owner):** repeat and settle it. Three probes with PINGs and
  three without on the two hosts, and one probe ending at the module's 90 s
  idle limit. If the PINGs reliably cause the closes, always `NO_ERROR` and
  always after 90 s, the MADR's D1 is amended with that and Phase 2 starts;
  a close inside 90 s, or with another code, stops the work again.
* **Files added to the phase:** none.
* **The repeat** (2026-11-04 01:41 to 01:47 UTC), seven runs at once on the
  two hosts, each run's stop rule first seen to fail on the first run's logs
  with its limit planted at 200 s (`<-- STOP` on `api.openai.com`'s close at
  125 s, exit 1):

  | Runs | PINGs written / acknowledged, each host | GOAWAY | Second request |
  | :--- | :--- | :--- | :--- |
  | 3, 30 s PINGs, 6 min idle | 11 / 11 | none | the same connection |
  | 3, no PINGs, 6 min idle | 0 / 0 | none | the same connection |
  | 1, 30 s PINGs, 90 s idle | 2 / 2 | none | the same connection |

  Runner's verdict: "no close inside 90 s, none but NO_ERROR", exit 0.
* **Outcome:** the PINGs do not cause the closes. The first run's two did not
  recur in three pinged runs; each host kept a pinged connection for 6
  minutes. No close came inside 90 s or with a code but `NO_ERROR`, so the
  owner's condition to stop does not hold. The MADR's amendment "Phase 1's
  ping probe" records the measurement under D1. Phase 2 starts after the
  owner commits this phase.

### Phase 2: the transport's health check (2026-11-03)

The owner committed Phase 1 and said "proceed".

* **Built, as steps 1 and 3 say:**
  * `transport.Config` has `SendPingTimeout` and `PingTimeout`;
    `defaultConfig` sets 30 s and 15 s; `manager.transport` sets
    `HTTP2: &http.HTTP2Config{…}` when `SendPingTimeout` is above zero;
    `DefaultClient`'s doc lists the health check with its other bounds;
  * `NewHTTPClient`'s doc lists it with the transport's bounds.
* **Tests,** in `llmprovider/internal/transport/transport_0068_test.go`:
  * `TestDefaultClient_HTTP2HealthCheck_0068`: `DefaultClient`'s transport
    carries `SendPingTimeout` 30 s and `PingTimeout` 15 s;
  * `TestDeadHTTP2Connection_FailsWithinPing_0068`: the report's P2 as a
    test. A listener whose connections can be silenced stands in for a dead
    peer. The transport is `defaultConfig`'s, from the test's own manager,
    with the test server's CA and a 5 s header timeout; the health check, if
    the manager set one, is shortened to 500 ms and 500 ms. The second
    request must fail within 3 s. It passed in 1.02 s.
* **Deviation 2 (2026-11-03): the dead-connection test's setup.** Step 2
  says "a test `Config` (1 s and 1 s, a header timeout of 30 s): the second
  request fails in under 10 s". A test that sets the new `Config` fields
  cannot compile at `HEAD`, and `tools/redgreen.py` counts a build failure
  as a finding, not a failure, so the rule "red first" could not be met.
  As built, the test takes `defaultConfig` with a 5 s header timeout,
  shortens the health check in the built transport to 500 ms and 500 ms
  when the manager set one, and requires the failure within 3 s; it
  compiles at `HEAD` and fails there on the hang itself. No decision
  changes. Offered: accept as built (recommended); build it as step 2 says,
  red by plants only; keep both. **Decision (owner): accept as built.**
  Files added to the phase: none.
* **Red at `HEAD`** (`tools/redgreen.py --expect fail --rev HEAD`, the test
  file copied): "2 of 2 fail as expected":
  * `transport_0068_test.go:23: HTTP2 is nil; want a health check`;
  * `transport_0068_test.go:136: second request failed after 5s (Get
    "https://127.0.0.1:63750": http2: timeout awaiting response headers);
    want within the 500 ms + 500 ms health check, not at the 5 s header
    timeout`.
* **Plants** (`tools/plant.py`, on a scratch clone): "all 3 plant(s)
  caught":
  * no `HTTP2` field set: both tests fail, the second after 5 s with the
    header timeout;
  * `SendPingTimeout` left out of `defaultConfig`: the same;
  * `PingTimeout` not passed on: `SendPingTimeout, PingTimeout = 30s, 0s;
    want 30s, 15s`.
* **Checks:**
  * `make pre-add-check` on the 3 Go files: "3 file(s) clean (gofmt,
    golangci-lint, go vet, go test, govulncheck)";
  * `tools/apidiff.py v1.25.0`: no change printed, exit 0; `make
    api-check`: "0 incompatible change(s)"; `make docs-check`: "0
    violation(s) of R50,R51,R52";
  * `tools/gate.py`: "29 of 29 passed", `g-wire-stable` among them (no
    golden changed).
* **Other platforms:** the transport package's tests on the Linux and
  Windows dev hosts run at the committed phase, before a push.
* **Staged:** this record, `llmprovider/internal/transport/transport.go`,
  `llmprovider/internal/transport/transport_0068_test.go`,
  `llmprovider/options.go`.

### Phase 3: what is not retried (2026-11-03)

The owner committed Phase 2 as `ca47d46` and said "proceed".

* **Phase 2 on other platforms,** at `ca47d46`, copied as a git bundle and
  cloned on each host, Go 1.27.3: `go test -count=1
  ./llmprovider/internal/transport/ ./llmprovider/` passed on Linux
  (amd64; 1.420 s and 1.725 s) and on Windows (amd64; 1.402 s and
  1.834 s).
* **Built, as steps 1 to 4 say:**
  * `transport.CertificateRefused`, in `unsendable.go`: whether the chain
    holds a `*tls.CertificateVerificationError`;
  * `retryable` returns false for it before anything else, so a refresh's
    failure wrapped as `ErrProviderUnavailable` is not retried either;
    `WithRetry`'s doc gains a bullet, and `RetryPolicy`'s a sentence;
  * `refreshOAuthSessionOnce` reports `retry` false for it; its doc and the
    attempt constants' say so;
  * `classifyAPIError` returns 501 and 505 terminal, with
    `ErrProviderUnavailable`, beside 525 and 526.
* **Made exact, no decision changed:** the four terminal 5xx statuses are a
  list, `terminalUnavailable`, matched by one `slices.Contains`. Written as
  four `||` conditions, `classifyAPIError`'s cyclomatic complexity rose to
  42, past `.golangci.yml`'s `gocyclo` limit of 40; as a list, the lint
  passes.
* **Tests:**
  * `TestWithRetry_RefusedCertificateOnce_0068` (`llmprovider/retry_0068_test.go`):
    a provider sends through `transport.DefaultClient()` to a TLS test
    server the client does not trust; under `WithRetry` with `MaxAttempts`
    3, one attempt and one connection, as sent and wrapped as
    `ErrProviderUnavailable`; the error holds the verification error;
  * `TestOAuthRefresh_RefusedCertificateOnce_0068`
    (`llmprovider/auth/oauth_refresh_0068_test.go`): a refresh against such
    a server makes one connection;
  * `TestClassify_501And505Terminal_0068`: 501 and 505 are
    `ErrProviderUnavailable` and not `Retryable`; 500 still is;
  * `TestClassifyHTTPStatus` gains rows for 501 and 505. They check the
    kind, which this phase does not change, so they pass at `HEAD` too and
    are not counted as red tests.
* **Red at `HEAD`:** "3 of 3 fail as expected":
  * `retry_0068_test.go:81: 3 attempts, 3 connections; want 1 and 1: a
    refused certificate is not retried`, in both subtests;
  * `oauth_refresh_0068_test.go:44: 3 connections; want 1: a refused
    certificate is not retried`;
  * `retry_0068_test.go:105: 501: Retryable = true; want false`, and the
    same for 505.
* **Plants:** "all 5 plant(s) caught":
  * the check moved after the kind switch, into the kindless branch: only
    the wrapped subtest fails, `3 attempts, 3 connections`. A first plant
    that removed the check outright failed both subtests; it is not the
    plant step 6 names, so the move was planted as written, the whole of
    `retryable` replaced through `old_file` and `new_file`;
  * the refresh loop's check removed: `3 connections; want 1`;
  * 501, then 505, dropped from the list: `Retryable = true; want false`;
  * `CertificateRefused` never true: all three certificate tests fail.
* **Checks:**
  * `make pre-add-check` on the 7 Go files: "7 file(s) clean". Its first run
    failed: the `gocyclo` limit above, and two `//nolint:errcheck`
    directives in the tests that `nolintlint` found unused, now removed;
  * `tools/apidiff.py v1.25.0`: no change printed; `make api-check`: "0
    incompatible change(s)"; `make docs-check`: "0 violation(s)";
  * `tools/gate.py`: "29 of 29 passed", `g-wire-stable` among them.
* **Other platforms:** the three packages' tests on the Linux and Windows
  dev hosts run at the committed phase, before a push.
* **Staged:** this record, `llmprovider/api_error.go`,
  `llmprovider/api_error_test.go`, `llmprovider/auth/oauth_session.go`,
  `llmprovider/auth/oauth_refresh_0068_test.go`,
  `llmprovider/internal/transport/unsendable.go`, `llmprovider/retry.go`,
  `llmprovider/retry_0068_test.go`.

### Phase 4: the lock wait and the command token (2026-11-03)

The owner committed Phase 3 as `a5152cb` and said "proceed".

* **Phase 3 on other platforms,** at `a5152cb`, as Phase 2's were: `go test
  -count=1 ./llmprovider/ ./llmprovider/auth/
  ./llmprovider/internal/transport/` passed on Linux (1.770 s, 7.845 s,
  1.403 s) and on Windows (1.807 s, 8.254 s, 1.388 s).
* **Built, as steps 1 and 2 say:**
  * `lockWait` is 60 s, its doc naming the 45.6 s hold it outlasts;
  * `CommandToken` keeps a run's error matching `ErrAuthFailure` for
    `commandTokenFailureMemory`, 5 s; `Token` in that window returns it
    without a run; a success or `Invalidate` clears it;
  * `Token` finding the output at least four fifths of its TTL old, with no
    run in flight, returns it and starts one run (`runEarly`) on
    `context.WithoutCancel(ctx)`, bounded by `Timeout`; a success replaces
    the output; a failure is not remembered. A caller whose output expires
    while that run goes joins it, as any caller joins a run in flight, and
    gets its result, an error included;
  * the type's doc and `Token`'s say both; `Invalidate`'s says it clears a
    remembered failure.
* **Made exact, no decision changed:** step 2 keeps a failure "which was
  not abandoned". An abandoned run returns its caller's context's error,
  never `ErrAuthFailure` (`runTokenCommand`), so matching `ErrAuthFailure`
  is the whole rule; a separate "not abandoned" test was dead code that no
  plant could catch, and is not written. A comment says why.
* **Tests:**
  * `TestLockWait_OutlastsLongestRefresh_0068` (`llmprovider/auth/lockwait_0068_test.go`),
    not parallel, as step 1 says;
  * in `llmprovider/command_token_0068_test.go`, with its own helper
    process, `TestHelperKeyCommand_0068`, a command that fails and counts
    its runs, and a settable clock:
    * `TestCommandToken_RemembersFailure_0068`: one run for two calls inside
      5 s, a second after 6 s, a third after `Invalidate`;
    * `TestCommandToken_AbandonedNotRemembered_0068`: a run ended by its
      caller's 100 ms deadline is not remembered; the next caller runs the
      command;
    * `TestCommandToken_EarlyRun_0068`: with a 10 s TTL, 9 s on, `Token`
      returns `key-1` before the early run has counted (the command counts
      after half a second), a run is in flight, a second caller leaves it the
      same run, the run outlives the first caller's cancelled context, and
      the next `Token` returns `key-2` after exactly two runs.
  * The first version of the early-run test required `Token` to return
    within 250 ms, which a loaded host could miss; the count before the run
    ends is the same check without a clock.
* **Red at `HEAD`:** "3 of 3 fail as expected":
  * `lockwait_0068_test.go:18: lockWait = 40s; want more than the longest
    refresh hold, 45.6s`;
  * `command_token_0068_test.go:72: 2 runs for two calls inside 5 s; want
    1`;
  * `command_token_0068_test.go:128: no run in flight after Token past four
    fifths of the TTL`.
* **Deviation 3 (2026-11-03): one test cannot be red at `HEAD`.**
  `TestCommandToken_AbandonedNotRemembered_0068` pins a rule `HEAD` already
  keeps, that an abandoned run is not remembered, so it passes there and is
  left out of the red run; it is seen red only by its plant below.
  Offered: keep it, red by plant (recommended); drop it; amend the "Red
  first" rule for later phases. **Decision (owner): keep it, red by
  plant.** Files added to the phase: none.
* **Plants:** "all 7 plant(s) caught":
  * `lockWait` back to 40 s: `lockWait = 40s; want more than … 45.6s`;
  * the failure kept with no window: `after 6 s: … after 1 runs; want
    ErrAuthFailure from a second run`;
  * `Invalidate` keeping the failure: `after Invalidate: … after 2 runs`;
  * any failure remembered, an abandoned run's included: `next Token = "",
    context deadline exceeded; want a run, not the abandoned one's error`;
  * the early run on the caller's context: cancelling the caller ends the
    run at once, `no run in flight after Token past four fifths of the TTL`;
  * two early runs allowed at once: `the second caller started a run of its
    own`;
  * no early run: `no run in flight …`.
* **Checks:**
  * `go test -count=2 -race -run 'CommandToken|_0068$'` on both packages:
    ok;
  * `make pre-add-check` on the 4 Go files: "4 file(s) clean";
  * `tools/apidiff.py v1.25.0`: no change printed; `make api-check`: "0
    incompatible change(s)"; `make docs-check`: "0 violation(s)";
  * `tools/gate.py`: "29 of 29 passed".
* **Other platforms:** the two packages' tests on the Linux and Windows dev
  hosts run at the committed phase, before a push.
* **Staged:** this record, `llmprovider/command_token.go`,
  `llmprovider/command_token_0068_test.go`,
  `llmprovider/auth/tokenstore_file.go`,
  `llmprovider/auth/lockwait_0068_test.go`.

### Phase 5: `reAuth` keyword-first (2026-11-03)

The owner committed Phase 4 as `2d8fd44` and said "proceed".

* **Phase 4 on other platforms,** at `2d8fd44`: `go test -count=1
  ./llmprovider/ ./llmprovider/auth/` passed on Linux (2.357 s, 7.894 s)
  and on Windows (2.469 s, 8.245 s).
* **Before** (step 1), on the tree at `2d8fd44`, load 6.6 to 8.6 on 10
  CPUs, `-count=10 -benchmem`:
  * `BenchmarkRedact_16KiBTokens`: median 1,415,558 ns/op (1,382,777 to
    1,831,852), 101,747 B/op, 12 allocs/op;
  * `BenchmarkRedact_Clean4KiB`: median 28,031 ns/op, 0 B/op, 0 allocs/op.
* **Built, as step 2 says:** `redactAuthPass`, the `reAuth` pass's `apply`,
  beside `redactKVPass`, with `reAuthAnchored` and `authKeywords`
  (`authorization`, `bearer`, `basic`, `token`). `keywordStarts` takes its
  keywords as an argument, so both passes share it.
  `kwAuthorization` names the literal, which `goconst` asked for at its
  third use.
* **Tests** (`internal/redact/redact_auth_0068_test.go`):
  `FuzzRedactAuthPass_MatchesExpression_0068`, the pass against
  `replaceSubmatches(reAuth, …, redactAuth)`, seeded with `equivCorpus()`
  and 19 shapes of label, case, spacing, overlap and folding. Every existing
  redaction test passes unchanged.
* **The fuzz run** (step 3), on a scratch copy of the tree: "fuzz: elapsed:
  5m1s, execs: 637770, new interesting: 119 (total: 356)", PASS, no input
  written under `testdata`. The copy was taken before `kwAuthorization`
  was named; the rename changes no byte the pass reads.
* **After** (step 4), load 8.1 to 8.3:
  * `BenchmarkRedact_16KiBTokens`: median 138,239 ns/op (136,454 to
    151,311), 152,838 B/op, 20 allocs/op: 10.2 times faster, with 50 % more
    bytes and 8 more allocations. They were not profiled. One source is
    plain in the code: the pass lower-cases the text into a 4 KiB stack
    buffer, which 16 KiB outgrows. An error body is cut to 2 KiB before
    redaction (0028-MADR D-A10), so in use that copy stays on the stack;
  * `BenchmarkRedact_Clean4KiB`: median 28,042 ns/op, unchanged: clean text
    holds no anchor, so the pass does not run.
* **Plants** (step 5), against the fuzz test alone: "all 3 plant(s)
  caught":
  * `authorization` left out of the keywords:
    `redactAuthPass("Authorization: Bearer invalid")` differs;
  * the non-ASCII fallback removed: `redactAuthPass("a Kelvin sign: toKen
    abcd1234efgh, …")` differs;
  * the skip inside the last match turned off: a slice out of range at the
    test's call of the pass.
* **What the plants found first:** the Kelvin-sign seed's credential was
  first `abcdefghij`, a word that `redactAuth` keeps with no label, so the
  fallback's removal changed nothing it read and was not caught; with
  `abcd1234efgh` it is. The first skip plant deleted the check, which left
  `pos` unread and the package unbuilt, a catch that proves nothing; the
  plant as recorded keeps it compiling (`s < pos && false`).
* **Deviation 4 (2026-11-03): the fuzz test cannot be red at `HEAD`.** It
  calls `redactAuthPass`, which `HEAD` lacks, and it asserts output
  identical to `HEAD`'s; like Deviation 3's guard, it is seen red only by
  its plants. Offered: accept, red by plants (recommended); amend the "Red
  first" rule; add a timing test that fails at `HEAD`. **Decision (owner):
  accept, red by plants.** Files added to the phase: none.
* **Checks:**
  * `make pre-add-check` on the 2 Go files: "2 file(s) clean". Its first run
    failed on the `goconst` finding above;
  * `tools/apidiff.py v1.25.0`: no change printed; `make docs-check`: "0
    violation(s)";
  * `tools/gate.py`: "29 of 29 passed".
* **Other platforms:** the redaction package's tests on the Linux and
  Windows dev hosts run at the committed phase, before a push.
* **Staged:** this record, `internal/redact/redact.go`,
  `internal/redact/redact_auth_0068_test.go`.
