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
