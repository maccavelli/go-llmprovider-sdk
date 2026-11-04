---
status: proposed
date: 2026-11-03
decision-makers: repository owner
consulted: this module at d65e9d1 (v1.25.0 is the last tag); 0054-REPORT-ideas-status-at-v1-25-0.md, the audit of 0054-REPORT-heuristics-and-performance-opportunities.md, and its probes; Go 1.27.3's net/http and crypto/tls documentation (go doc)
informed: anyone who wraps a provider in WithRetry, signs in with an OAuth session saved by FileTokenStore, keys a provider with CommandToken, or reads an error the module redacted
---

<!-- markdownlint-disable MD013 -->

# Five Internal Fixes from the 0054 Audit: an HTTP/2 Health Check, No Retry of a Refused Certificate or of 501 and 505, a Lock Wait Past the Longest Refresh, a Command Token That Remembers Failure and Runs Early, and a Keyword-First Auth Redaction

## Context and Problem Statement

[0054-REPORT-ideas-status-at-v1-25-0.md](../reports/0054-REPORT-ideas-status-at-v1-25-0.md)
audited the 30 ideas of
[0054-REPORT-heuristics-and-performance-opportunities.md](../reports/0054-REPORT-heuristics-and-performance-opportunities.md)
against `d65e9d1`. Five of the open ones are small, internal, and change
neither an exported name nor what a request carries. This record decides
them:

* **F1. A dead HTTP/2 connection holds a request for the whole header
  timeout** (the report's idea 5.1). The shared transport sets no HTTP/2
  health check (C1). A pooled connection whose peer has gone silent, after a
  laptop sleeps or a NAT forgets the flow, takes the next request and holds
  it until `ResponseHeaderTimeout`, 300 s (C2). The report's probe P2
  measured it: 8.0 s at a shortened timeout of 8 s with no health check,
  2.0 s with a 1 s ping and 1 s ping timeout.
* **F2. A failure no retry can mend is retried** (idea 6.2; 0028-MADR's
  shortlist H6, not chosen then, C3).
  * A certificate the client refuses comes back as a `*url.Error` wrapping
    a `*tls.CertificateVerificationError`, with no `*net.OpError`. It is
    neither offline nor unsendable, so `WithRetry` sends it again
    `MaxAttempts` times (C4; the report's P1).
  * A refresh that fails so is also retried, three times, and is wrapped as
    `ErrProviderUnavailable` (C5), which `WithRetry` retries too (C6).
  * HTTP 501 (Not Implemented) and 505 (HTTP Version Not Supported) fall
    into the 5xx row and are retried (C7); only 525 and 526 are terminal
    there (C8).
* **F3. A waiter gives up on the refresh lock before a legitimate holder
  can finish** (idea 10.2; 0028-MADR's shortlist H9, C9). `lockWait` is
  40 s (C10). The holder keeps the lock across up to three attempts of
  15 s each, with 200 ms and 400 ms between them (C11, C12, C13): 45.6 s.
  A second process waiting for it fails with "another process is
  refreshing" while the first is still within its bounds.
* **F4. A failing key command runs on every request, and a good one blocks
  its caller once per TTL** (idea 10.3). `CommandToken` stores only a
  successful output (C14); a failed run is forgotten, so each `Token` runs
  the command again, up to its 10 s timeout. When the output is older than
  its TTL, five minutes by default (C15), the caller that finds it runs the
  command and waits for it (C16).
* **F5. One expression is most of the redaction's cost** (idea 9.2).
  `reAuth` (C17) runs its own unanchored search over the text whenever the
  text holds `bearer`, `basic` or `token` (C18). Its optional
  `authorization:` label begins with a repetition that the search tries
  from every position. On the same 2 KiB text, `reAuth.Match` alone took
  about as long as the whole of `Redact` (the report's P5). 0028-MADR D-A10
  made `reKV` look up its keyword first for the same reason (C19), and kept
  the result identical, checked by a differential fuzz test (C20).

### Code claims

<!-- claims: self=d65e9d1 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | The shared transport, with no HTTP2 field | `llmprovider/internal/transport/transport.go:95` | `t := &http.Transport{` |
| C2 | The header timeout | `llmprovider/internal/transport/transport.go:72` | `ResponseHeaderTimeout: 300 * time.Second,` |
| C3 | 0028 shortlisted the certificate retry | `docs/decisions/0028-MADR-heuristics-and-performance-from-the-research-pass.md:696` | `H6 (a` |
| C4 | A kindless send failure is retried | `llmprovider/retry.go:339` | `return errors.As(err, &urlErr) && !transport.Unsendable(err)` |
| C5 | A refresh's send failure is retried and wrapped | `llmprovider/auth/oauth_session.go:589` | `return nil, llmprovider.Token{}, true, fmt.Errorf("%w: oauth: refresh request: %w", llmprovider.ErrProviderUnavailable,` |
| C6 | `ErrProviderUnavailable` is retried | `llmprovider/retry.go:329` | `case errors.Is(err, ErrRateLimited), errors.Is(err, ErrProviderUnavailable):` |
| C7 | Every other 5xx is retryable | `llmprovider/api_error.go:457` | `case status >= http.StatusInternalServerError:` |
| C8 | 525 and 526 are terminal | `llmprovider/api_error.go:455` | `case status == 525 \|\| status == 526:` |
| C9 | 0028 shortlisted the lock wait | `docs/decisions/0028-MADR-heuristics-and-performance-from-the-research-pass.md:698` | `` H9 (`lockWait`, 40 s, is `` |
| C10 | The lock wait | `llmprovider/auth/tokenstore_file.go:25` | `const lockWait = 40 * time.Second` |
| C11 | Refresh attempts | `llmprovider/auth/oauth_session.go:36` | `oauthRefreshAttempts = 3` |
| C12 | The backoff between them | `llmprovider/auth/oauth_session.go:37` | `oauthRefreshBackoff  = 200 * time.Millisecond` |
| C13 | Each attempt's bound | `llmprovider/auth/oauth_session.go:55` | `var oauthRefreshAttemptTimeout = 15 * time.Second` |
| C14 | Only a successful run is stored | `llmprovider/command_token.go:132` | `if err == nil {` |
| C15 | The default TTL | `llmprovider/command_token.go:21` | `defaultCommandTokenTTL     = 5 * time.Minute` |
| C16 | Past its TTL the caller runs the command | `llmprovider/command_token.go:102` | `if c.value != "" && now.Sub(c.fetched) < c.ttl() {` |
| C17 | `reAuth` | `internal/redact/redact.go:37` | `reAuth = regexp.MustCompile(` |
| C18 | Its pass runs the expression's own search | `internal/redact/redact.go:197` | `{re: reAuth, replaceSub: redactAuth, anchors: []string{"bearer", "basic", kwToken}},` |
| C19 | `reKV`'s keyword-first pass | `internal/redact/redact.go:294` | `func redactKVPass(p []byte) []byte {` |
| C20 | Its differential fuzz test | `internal/redact/redact_equiv_0028_test.go:167` | `func FuzzRedact_MatchesReference(f *testing.F) {` |
| C21 | A request written before a failure is marked | `llmprovider/internal/wire/post.go:168` | `return nil, transport.AfterReply(err)` |
| C22 | An abandoned run is not kept | `llmprovider/command_token.go:137` | `future.abandoned = err != nil && ctx.Err() != nil` |
| C23 | 0021's lock wait | `docs/decisions/0021-PLAN-harden-and-tune-after-the-v1-1-review.md:718` | `` `lockWait` becomes 40 s `` |
| C24 | 0059's wizard wait | `docs/decisions/0059-MADR-wizard-debugging-pass-findings.md:514` | `at most about 46 s` |
| C25 | 0017's command keys | `docs/decisions/0017-MADR-together-provider-and-auth-extensions.md:137` | `### D3. Keys from a command` |

## Decision Drivers

* **No exported name and no request changes** (R48, R45). PING frames are
  connection frames, not request content.
* **A failure is reported when it is known**, not after a bound that exists
  for a slow but healthy reply (F1), and not after retries that cannot
  succeed (F2).
* **Two bounds that must hold in order do** (F3), and a test holds them.
* **One pattern per problem, already in the module:** the after-reply mark
  for a failure after the request was written (C21); `reKV`'s keyword-first
  pass and its differential fuzz test (C19, C20); the abandoned-run rule
  (C22).
* **The standard library only,** as `AGENTS.md` requires.

## Considered Options

* **A. Take up the five, each at its root,** as decided below.
* **B. Fix only the two that are defects** (F2's certificate, F3's lock
  wait), and leave F1, F4 and F5 in the 0054 backlog.
* **C. Fold them into the larger retry and transport ideas** (0054's 4.1,
  4.2 and 5.2), under one record later.

## Decision Outcome

Chosen option: **"A. Take up the five, each at its root"**, because each is
small and separately testable and revertible, each uses a mechanism the
standard library or the module already has, and together they turn a
300 s hang, futile retries, a spurious lock failure, a command run on every
request and most of the redaction's cost into tested behaviour, with no
exported name and no change to a request.

### D1. An HTTP/2 health check on the shared transport (F1)

* **The setting:** `transport.Config` gains `SendPingTimeout` and
  `PingTimeout` (`time.Duration`, so it stays a comparable map key).
  `manager.transport` sets `HTTP2: &http.HTTP2Config{SendPingTimeout,
  PingTimeout}` when `SendPingTimeout` is above zero. `defaultConfig` takes
  30 s and 15 s (Q1).
* **What it does,** by net/http's documentation: a connection that receives
  no frame for `SendPingTimeout` is sent a PING; one that does not answer
  within `PingTimeout` is closed, and its requests fail with `http2: client
  connection lost`. A dead connection is found within 45 s, not 300 s.
* **What follows a failure:** a request already written is marked after the
  reply (C21), so `WithRetry` resends it at most once (0028-MADR D-H1). One
  not yet written is net/http's to send on a new connection.
* **Scope:** HTTP/2 only. HTTP/1.1 connections, such as a local Ollama's,
  are unchanged, and keep the dialer's TCP keep-alive.
* **The hazard:** a server that limits how often it accepts pings closes
  the connection with GOAWAY `ENHANCE_YOUR_CALM` and "too_many_pings".
  gRPC's keepalive guide gives its defaults: pings no closer than five
  minutes apart while no data flows, and none on a connection with no call
  open
  ([grpc/grpc doc/keepalive.md](https://github.com/grpc/grpc/blob/master/doc/keepalive.md)).
  A host with that policy would refuse a 30 s or a 60 s interval alike
  during a long silent wait.
  Whether any provider's API host has it is not established. The PLAN's
  first phase probes each before any code changes, and a refusal at the
  chosen interval stops the work for the owner.
* **The doc:** `NewHTTPClient`'s doc, which lists the transport's bounds
  (0066-MADR D2), adds the health check.

### D2. A refused certificate, 501 and 505 are not retried (F2)

* **A refused certificate:** `retryable` returns false for an error whose
  chain holds a `*tls.CertificateVerificationError`, before it looks at the
  error's kind, so a refresh's failure wrapped as `ErrProviderUnavailable`
  (C5) is not retried either. The OAuth refresh loop does not retry it. The
  error keeps its text and gains no kind. `WithRetry`'s doc names it among
  the failures it does not retry.
* **501 and 505:** `classifyAPIError` returns them terminal, with
  `ErrProviderUnavailable`, as it returns 525 and 526 (C8). MADR 0012 §1.1's
  table gains the two rows.
* **Not chosen here:**
  * a DNS failure that names no host, NXDOMAIN, stays retried: 0061-MADR D2
    decided that an offline failure is retried, and a resolver that has
    just woken is a real case;
  * Anthropic's spend cap (`enforced_spend_limit_reached`): its reply was
    never measured, and a rule needs the text.

### D3. The lock wait outlasts the longest refresh (F3)

* **The value:** `lockWait` goes from 40 s to 60 s (Q4), past the longest
  hold, 45.6 s, and the save after it.
* **The test:** one test computes the longest hold from
  `oauthRefreshAttempts`, `oauthRefreshAttemptTimeout` and
  `oauthRefreshBackoff`, and fails when `lockWait` does not exceed it. A
  later change to either side that breaks the order fails it.
* **Unchanged:** the longest hold, and so 0059-MADR's wizard wait, "at most
  about 46 s" (C24).
* **Not chosen here:**
  * jitter on the refresh's backoff: the file lock and each process's
    single flight already serialise refreshes, so 200 ms and 400 ms of
    jitter would spread nothing;
  * honouring `Retry-After` on a refresh: no issuer was measured sending
    one;
  * a blocking lock in place of the 50 to 150 ms poll (0028-MADR A7):
    platform code for a small gain.

### D4. A command token remembers a failure, and runs early (F4)

* **Failure memory:** a run that ends in an error matching `ErrAuthFailure`
  (the command failed, timed out, wrote too much or printed nothing) is kept
  with its error for 5 s (Q2). A `Token` in that window returns the same
  error without running the command. `Invalidate` clears it. A run that
  ended because its caller gave up is not kept, as today (C22).
* **Early run:** a `Token` that finds the output older than four fifths of
  its TTL (Q3), with no run in flight, returns the output and starts one
  run. The run is detached from the caller's context and bounded by
  `Timeout`. Its success replaces the output; its failure is dropped, and
  the output serves to its TTL, after which `Token` runs the command as
  today.
* **The doc:** `CommandToken`'s doc says both.
* **Not chosen here:** caching `auth.VendorCLISession`'s file (0028-MADR
  A5): it reads a small file per `Token`, unmeasured.

### D5. `reAuth` finds its keyword first (F5)

* **The pass:** `reAuth`'s pass takes an `apply` function, as `reKV`'s does
  (C19). In ASCII text it finds each occurrence of `authorization`,
  `bearer`, `basic` and `token` in the lower-cased copy, in order, skipping
  any inside the last match, and tries `reAuth` anchored there. Every match
  starts at one of them: at the label when it has one, else at its scheme.
  So the first anchored match at the leftmost such place is the match the
  expression's own search finds, and `redactAuth` replaces it as before.
* **Outside ASCII:** the expression's own search, as `reKV`'s pass does,
  because `(?i)` folds letters such as the Kelvin sign that a byte search
  would miss.
* **The proof:** a differential fuzz test against the expression's own
  search, seeded with the redaction tests' corpus, as C20 is.
* **The measure:** `BenchmarkRedact_16KiBTokens` before and after, on a host
  whose load is below its CPU count. A result that is not faster stops the
  work for the owner.

### D6. Tests, docs and the release

* **Tests:** each fix gets a test that fails on today's code, and each test
  is seen to fail on a planted return of the defect, on a scratch clone:
  * D1: a test server whose connection goes silent after one request; the
    next fails within the ping timeouts the test sets, not the header
    timeout; and `DefaultClient`'s transport carries the health check;
  * D2: `WithRetry` over an untrusted TLS server makes one attempt; an OAuth
    refresh against one makes one; 501 and 505 are not `Retryable`;
  * D3: the ordering test;
  * D4: a failing command runs once in the window and again after it, and
    after `Invalidate`; an output past four fifths of its TTL is returned at
    once while one run goes;
  * D5: the fuzz test, and the benchmark.
* **Docs:** the doc of each function whose behaviour changes (`WithRetry`,
  `RetryPolicy`, `NewHTTPClient`, `CommandToken`, `FileTokenStore`), with
  `make docs-check` at 0; `docs/architecture.md`'s transport and auth
  paragraphs; `CHANGELOG.md`'s Changed (D1 to D4) and Fixed (F2, F3).
* **Release:** `v1.26.0` (Q5).

### Consequences

* Good, because a dead HTTP/2 connection fails a request in about 45 s, not
  300 s (D1).
* Good, because a refused certificate, 501 and 505 fail at once, where they
  cost `MaxAttempts` attempts and the waits between them (D2).
* Good, because a second process waits out a legitimate refresh rather than
  failing beside it (D3).
* Good, because a failing key command runs at most once per 5 s, and a
  working one stops blocking a request once per TTL (D4).
* Good, because redaction, which runs on every error body, stops paying for
  an unanchored search (D5).
* Neutral, because a caller whose certificate failure was transient, a
  clock briefly wrong, now sees it at once; a retry within seconds would
  rarely have mended it.
* Bad, because PINGs go to every provider's API host on silent connections,
  one 8-byte frame per 30 s of silence, and a host that limits them would
  close connections. Phase 1's probe is the guard.
* Bad, because the early run of a command is a goroutine that outlives the
  call that started it, up to `Timeout`. The listing fetch and metadata
  fetch already detach the same way (0053-MADR D4).
* Bad, because a mended key command is not run again for up to 5 s, unless
  the caller calls `Invalidate`.

### Confirmation

* D6's tests pass, and each failed on its planted break.
* The report's P1, run again on the result: one attempt. Its P2, through
  the module's transport with test timeouts: the second request fails
  within the ping timeouts.
* `BenchmarkRedact_16KiBTokens` faster, and the fuzz test clean.
* `tools/apidiff.py v1.25.0` reports no change; G-wire's goldens do not
  change; `make docs-check` reports 0.

## Pros and Cons of the Options

### A. Take up the five, each at its root

* Good, because each is a few lines where the problem is, with a test that
  pins it.
* Good, because none needs an exported name or changes a request.
* Neutral, because three amend recorded decisions in the open (More
  Information).
* Bad, because D1 sends a frame the module never sent, whose reception by
  each host is to be measured.

### B. Only the two defects

* Good, because it is the smallest change.
* Bad, because F1's 300 s hang is the largest cost a caller sees of the
  five, and F4 and F5 cost on every request or every error.

### C. Fold them into the larger ideas

* Good, because a retry budget (4.1) or streaming every `Generate` (5.2)
  would change the same code.
* Bad, because those need exported API or change every request, and wait
  on measurements and owner decisions; these five do not.

## Owner questions

Answered by the owner on 2026-11-03, each as recommended.

* **Q1 (D1). The ping interval.** Offered: 30 s (recommended: a dead
  connection is found within 45 s, and a long generation's silent wait sends
  at most ten pings); or 60 s (within 75 s, half the pings); or 120 s
  (within 135 s, at most two pings in a 300 s wait). Any holds only if
  Phase 1's probe finds no host that refuses it. **Answer: 30 s.**
* **Q2 (D4). The failure memory.** Offered: 5 s (recommended: a caller
  looping on errors spawns at most one process per 5 s, and a mended
  command is used within 5 s); or 30 s; or none, keeping only the early run.
  **Answer: 5 s.**
* **Q3 (D4). When the early run starts.** Offered: at four fifths of the TTL
  (recommended: the last minute of the default five); or at half; or no
  early run, keeping only the failure memory. **Answer: at four fifths.**
* **Q4 (D3). The lock wait.** Offered: 60 s (recommended: past 45.6 s with
  room for the save); or the longest hold plus 5 s, computed from the
  refresh's constants; or 90 s. **Answer: 60 s.**
* **Q5 (D6). The release.** Offered: `v1.26.0` (recommended: a caller can
  see four behaviours change, as 0053-MADR's Q3 answer took for its
  release); or `v1.25.1`, a patch, since no exported name changes; or no
  tag yet, holding the phases for the next feature release. **Answer:
  `v1.26.0`.**

## More Information

* **Records amended once this is accepted:**
  * 0021-MADR-harden-and-tune-after-the-v1-1-review.md D2, the transport's
    bounds: a health check is added, none changed (D1);
  * 0026-MADR-remediate-v1-2-debugging-pass-findings.md F11, which retries
    a failure to reach the service: a refused certificate is not one (D2);
  * MADR 0012 §1.1's status table, 0012-MADR-conform-providers-to-reference-clients.md:
    501 and 505 (D2);
  * the 40 s that 0021-PLAN-harden-and-tune-after-the-v1-1-review.md set
    for `lockWait` (C23; D3);
  * 0017-MADR-together-provider-and-auth-extensions.md D3, keys from a
    command (C25; D4);
  * 0028-MADR-heuristics-and-performance-from-the-research-pass.md D-A10,
    extended to `reAuth` (D5); its shortlist's H6 and H9 are taken up here.
* **Stands:** 0061-MADR-consumer-gaps-from-gobble-issue-9.md D2 (offline
  failures retried); 0059-MADR-wizard-debugging-pass-findings.md's wait.
* **Exported names:** none added (R55 has nothing to list).
* The PLAN is
  [0068-PLAN-five-internal-fixes-from-the-0054-audit.md](0068-PLAN-five-internal-fixes-from-the-0054-audit.md).
