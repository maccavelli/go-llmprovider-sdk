---
status: complete
date: 2026-10-02
associated-madr: "0016-MADR-provider-auth-and-support-baseline.md"
decision-makers: go-llmprovider-sdk maintainers
---
# Implement the Provider Auth and Support Baseline

Associated MADR: [0016-MADR-provider-auth-and-support-baseline.md](0016-MADR-provider-auth-and-support-baseline.md)

## Goal

At the end of this plan, before `v1.0.0`:

1. Every built-in provider authenticates through a `TokenSource` that it
   applies as `Token` describes (MADR D2), with no request change on the
   wire.
2. `FileTokenStore` writes durably (D3), and a failed save never discards a
   rotated token (D4).
3. No secret-bearing type prints its secret (D5).
4. Device login returns a handle (D6); the OAuth checks of D7 hold.
5. The default transport honours proxies and is shared per provider (D8).
6. Listing sends no billed probe unless asked (D9).
7. `wizard` keeps one copy of a refresh token and offers logout (D11).
8. The environment helper reads `ANTHROPIC_API_KEY` first (D12).

## Scope

### In scope

* The packages 0015-MADR D2 creates: `llmprovider`, `llmprovider/auth`,
  `llmprovider/catalog`, `llmprovider/providers/*`,
  `llmprovider/internal/transport`, `wizard`, and their tests.
* `docs/guides/api-standards.md` and `docs/guides/adding-a-provider.md`
  (rules for D2 and D5), which 0015-PLAN creates.
* This pair; the 0015 pair and the 0002 MADR, for the cross-references in
  T0 only.

### Out of scope

* Everything the MADR's D10 excludes: agent-CLI drivers, writes into vendor
  CLI stores, Anthropic or Gemini subscription OAuth.
* The Windows DACL and reserved names for `FileTokenStore`: the transferred
  [0010-PLAN-windows-stdio-oauth-tokenstore-ci.md](0010-PLAN-windows-stdio-oauth-tokenstore-ci.md).
* A cross-process lock (MADR D3). ~~and `id_token` signature verification
  (MADR D3, D7)~~ *(verification moved into scope 2026-09-29, T2 step 5)*
* Any file in `magic-cli-remote` or `mcplib`.

## 0. Preconditions and conventions

* **Start.** T0 needs the owner's decision on this MADR **and** on
  0015-MADR, because D2, D6 and D8 are written in 0015's API. If 0015-MADR
  is rejected, this plan stops for an amendment; it is not executed against
  the `mcplib` API.
* **No phase of its own.** Each step below lands inside the named
  0015-PLAN phase and in that phase's commits, so the tree is restructured
  once. The 0015-PLAN gate applies unchanged, including G-wire.
* **Red first.** Every new test is seen to fail against the pre-change code
  in a scratch copy of the tree (`SCRATCH/0016-<step>`), never by dirtying
  this tree. The failing output is quoted in the execution record.
* **Stop conditions.** Those of 0015-PLAN, and also:
  * a G-wire request difference caused by D2 or D8;
  * a consumer found reading a field D11 removes, other than
    `prepare-commit-msg`'s (MADR M9).

## Implementation Steps

### T0: accept (docs only)

1. On the owner's decision, set this MADR `accepted` and this PLAN
   `in-progress`.
2. Amend 0015-MADR with a dated section: D9's ambient check allows
   `http.ProxyFromEnvironment` and nothing else (0016-MADR D8).
3. Amend 0015-PLAN with a dated note listing, per phase, the T-steps below
   that land in it.
4. Amend 0002-MADR §13 with a dated note: `prepare-commit-msg`'s companion
   must read the refresh token from its `TokenStore`, not from
   `wizard.Result` (0016-MADR D11).
5. Update `docs/README.md`. Commit these records only (bootstrap
   exception).

### T1: transport (inside 0015-PLAN S3)

1. **D8.** The default client in `internal/transport` sets
   `Proxy: http.ProxyFromEnvironment` and keeps today's timeouts and
   connection limits. *Amended 2026-09-30:* it lands in `llmprovider`
   (`options.go`) in 0015-PLAN S3, and moves to `internal/transport` with the
   rest of the transport code in 0015-PLAN S7b. A provider builds one client in `New` and passes it
   to listing and to its OAuth session when that has none.
2. **Test.** A provider whose base URL is an `httptest` server, with the
   proxy variables pointing at a second `httptest` server acting as the
   proxy, reaches the proxy. Red first against `options.go:14-24`.
3. **Test.** Listing and a refresh through a provider built without
   `WithHTTPClient` use the same `*http.Client` as its requests.

### T2: auth (inside 0015-PLAN S4)

1. **D3, durable writes.** `FileTokenStore.Save`: `os.CreateTemp` in the
   store directory, `Chmod(0600)` before the first write, write, `Sync`,
   `Close`, `Rename`, then `Sync` of the directory on non-Windows. `Load`
   refuses a file larger than 64 KiB.
   * **Test:** a planted rename failure leaves the previous file intact and
     no temp file behind.
   * **Test (unix):** the temp file's mode is `0600` before any byte is
     written, observed through a write hook. Red first against
     `tokenstore_file.go:88-117`.
   * **Test:** a 64 KiB + 1 file is refused.
2. **D4, rotation kept.** The session adopts the refreshed tokens before
   `Save`. On a `Save` error it returns the fresh token, logs the error
   through the injected logger, and retries `Save` on the next `Token`.
   * **Test:** with a store whose first `Save` fails, the next forced
     refresh sends the rotated refresh token. Red first against
     `oauth_session.go:121-125,376-381`, where it sends the spent one.
3. **D5, redaction.** `Token`, `StaticToken` and the OAuth session
   implement `String`, `GoString` and `LogValue`, masking with
   `internal/redact`'s `MaskSecret`.
   * **Test:** a table over every secret-bearing type, formatted with `%v`,
     `%+v`, `%#v`, `%s` and through `slog` JSON and text handlers, fails if
     a planted secret appears. Red first on the current types.
4. **D6, device handle.** Device login returns a handle with the user code,
   verification URI, expiry, `Wait(ctx)` and `Cancel()`. `Cancel` stops
   polling and makes `Wait` return a cancellation error.
   * **Tests:** `Wait` returns the session after an `httptest` device server
     approves; `Cancel` during polling returns within one poll interval;
     the existing `slow_down` and validation tests pass unchanged in
     meaning.
5. **D7, OAuth checks.**
   * The Grok login compares the `id_token` `nonce` claim with the one it
     sent. **Test:** a mismatched nonce fails the login. Red first against
     `oauth_loopback.go:381-386`.
   * Discovery falls back to built-in endpoints only for the built-in
     issuer. **Test:** a caller-supplied issuer whose discovery returns 500
     fails the login. Red first against `oauth_loopback.go:201-242`.
   * **Signature verification (the owner's decision of 2026-09-29).** `auth`
     verifies each `id_token` before using any claim:
     * keys from the `jwks_uri` of the issuer's discovery document, cached,
       and refetched once for an unknown `kid`;
     * RS256 (`crypto/rsa`, OpenAI) and ES256 on P-256 (`crypto/ecdsa`,
       xAI), the algorithms both issuers advertise (MADR, "Owner's
       decisions"). Any other `alg`, including `none` and HS256, is refused;
     * `iss` equals the issuer, `aud` contains the client id, `exp` is in the
       future, and for Grok the `nonce` matches;
     * any failure (bad signature, a key still unknown after the refetch, or
       keys that cannot be fetched) fails the login and saves nothing.
   * **Tests** against an `httptest` issuer serving discovery and a JWKS, with
     keys generated in the test:
     * valid RS256 and ES256 tokens pass;
     * a tampered payload, a wrong `aud`, an expired token, `alg: none` and
       an HS256 token each fail;
     * an unknown `kid` triggers exactly one refetch, then fails;
     * an unreachable JWKS fails the login.

     Red first: today's code accepts the tampered token.
   * **Live pin.** A `live_gateways` test fetches both built-in issuers'
     discovery documents and asserts their advertised algorithms are among
     RS256 and ES256.
   * The `auth` package doc states that the default client ids are the
     vendor CLIs'.

#### T2 additions from the survey (2026-09-30, accepted; see the MADR's amendment)

The owner accepted the MADR's A1–A3 on 2026-09-30, A2 as option (b). They
run in T2 (0015-PLAN S4).

* **A1, step 2b.** On a permanent refresh failure, re-read the store once
  and adopt a sibling-rotated session. Red first: two sessions share a
  `FileTokenStore`, and the second's refresh is rejected after the first
  saved. Today it fails; it must adopt.
* **A2, step 1b (if option (b) is chosen).** `FileTokenStore` locks a
  refresh:
  * exclusive-create lock file, pid and time, stale after 30 s;
  * a jittered retry to a deadline, and a re-read after acquiring;
  * a timeout is a retryable error.

  Tests:
  * two goroutines standing in for processes, on one store, refresh once in
    total;
  * a stale lock is taken over;
  * a held lock times out as retryable.
* **A3, step 5.** Require `kid`, and apply the algorithm allowlist
  intersected with discovery's list. Verify on the device flows too, and
  fail a login that lacks an `id_token` when `openid` was requested. Each
  case gets a red-first test.

### T3: providers and catalog (inside 0015-PLAN S5 and S7)

1. **D2, per provider in S7.** Each provider takes its credential only as a
   `TokenSource`. With an empty `Token.Header` it sends the header it sends
   today; a non-empty `Header` overrides it, prefixed by `Type` when set.
   Today's headers, read at `4ddcb54`:

   | Provider | Header today | Source |
   |---|---|---|
   | openai, grok | `Authorization: Bearer` | `openai.go:191`, `grok.go:226` |
   | huggingface, kilo | `Authorization: Bearer` | `huggingface.go:161`, `kilo.go:293` |
   | claude | `x-api-key` | `claude.go:229` |
   | gemini | `x-goog-api-key` | `gemini.go:222` |
   | opencode-zen, opencode-go | per route: `x-api-key` (Messages), `x-goog-api-key` (Gemini), else `Authorization: Bearer` | `opencode.go:390-399` |
   | ollama | none | `ollama.go:176` |

   Listing sends the same headers from `discovery.go`, and follows the same
   rule.

   The table is re-read from the source at S7, per provider, before the
   change; any row that differs stops the step.
   * **Test:** G-wire for every provider shows no request difference.
   * **Test:** a `TokenSource` returning `Header: "X-Custom"` and a
     non-empty `Type` reaches the server as that header, for every
     provider that accepts a caller token. Red first on `claude` (which
     today refuses a `TokenSource`).
   * **Test:** `New` refuses an OAuth or vendor-CLI source for a provider
     whose descriptor does not list that method.
2. **D9, probes in S5.** Listing returns the service's list. A listing
   option turns on the generation probe, with a doc comment stating it
   sends one billed request per model.
   * **Test:** listing against an `httptest` server receives no generation
     request by default, and one per candidate with the option. Red first
     against `probe.go:13-60`.
   * Record this difference in the G-wire execution notes as caused by
     0016-MADR D9.

### T4: wizard (inside 0015-PLAN S8)

1. **D11.** When the session is saved to `Options.TokenStore`, `Result`
   carries no access or refresh token.
   * **Test:** after an OAuth configuration with a store, `Result`'s token
     fields are empty and the store holds the session. Red first against
     `wizard/auth.go:323-341`.
2. **D11, logout.** For a provider with a stored session, the wizard offers
   "log out": revoke, then `Delete`, reporting a revocation failure and
   deleting anyway.
   * **Tests:** a revoke server that succeeds, and one that returns 500;
     both end with the store empty.
3. **D5.** `wizard.Result` redacts itself; it joins T2 step 3's table.

### T5: environment helper (inside 0015-PLAN S10)

1. **D12 (the owner's decision of 2026-09-29).** `ANTHROPIC_API_KEY` only:
   for the opt-in helper, and as the `claude` descriptor's variable that the
   wizard's `AllowEnv` reads. `CLAUDE_API_KEY` is not read.
   * **Test:** with only `CLAUDE_API_KEY` set, no key is found; with
     `ANTHROPIC_API_KEY` set, it is used. Red first against
     `provider.go:165`.

### T6: close-out

1. Fill the execution record with each red-first output, the G-wire result
   and coverage.
2. Add the D2 and D5 rules to `docs/guides/api-standards.md` and the
   `TokenSource` rule to `docs/guides/adding-a-provider.md`, each citing
   this MADR.
3. Set this PLAN `complete` and mark the MADR's Confirmation items met.

## Verification

| # | Criterion | Check | Step |
|---|---|---|---|
| V1 | Every provider accepts a `TokenSource`; no request changes | T3.1 tests; G-wire | T3 |
| V2 | Writes are durable and `0600` from creation | T2.1 tests | T2 |
| V3 | A failed save keeps the rotation | T2.2 test | T2 |
| V4 | No secret in any formatted value | T2.3 table, with T4.3 | T2, T4 |
| V5 | Device handle waits and cancels | T2.4 tests | T2 |
| V6 | Nonce checked; caller issuer fails closed; `id_token` signatures verified, failures fail the login | T2.5 tests; live pin | T2 |
| V7 | Proxy honoured; one client per provider | T1 tests | T1 |
| V8 | ~~No billed probe by default~~ *Superseded 2026-09-30 by A5:* probes on by default, off with `WithModelProbes(false)` | T3.2 test, as reversed | T3 |
| V9 | One refresh-token copy; logout works | T4 tests | T4 |
| V10 | `ANTHROPIC_API_KEY` only | T5 test | T5 |
| V11 | No new module | `make dep-check` (0015-PLAN S12) | T6 |

## Rollout and Rollback

* No consumer imports the module before `v1.0.0-rc.1`, so each step is
  reverted with its 0015-PLAN phase commit if needed.
* The consumer-visible changes (D9, D11) reach consumers through their
  0002-MADR §13 companion records, which must list them.
* Abandoning this plan leaves 0015-PLAN executable as written. The M3–M11
  defects then remain, and the MADR is marked `rejected` with the reason.

## Deviation Log

* **2026-09-30, T1's location.** 0015-PLAN S3 could not extract
  `internal/transport` before the providers leave `llmprovider` (an import
  cycle). The owner chose to move the extraction to a new 0015-PLAN S7b. T1
  therefore lands in `llmprovider/options.go` in S3 and moves in S7b. What T1
  does, and its tests, are unchanged. Recorded in full in 0015-PLAN,
  "Deviation 2026-09-30".
* **2026-09-30, T2's and T3's locations.** The same cycle holds for
  0015-PLAN S4 (`auth`) and S5 (`catalog`), and the owner extended the same
  resolution to them:
  * T2 lands in S4 and T3 step 2 in S5, inside `llmprovider`;
  * both move with their files in 0015-PLAN S7b;
  * where T2 step 5 names the `auth` package doc (the note on the default
    client ids), the text goes in the doc comment of the file that moves, and
    becomes the package doc in S7b.

  What the steps do, and their tests, are unchanged.
* **2026-09-30, T2 step 1's red-first premise.** The step expected the
  "temp file is `0600` before any byte is written" test to fail against the
  old code. It cannot: `os.CreateTemp` creates its file `0600` (measured in a
  clone of `fe625b8`: `os.CreateTemp mode: 600`). The old code's `chmod`
  after the rename was redundant, not late, and MADR finding M5's "chmod
  0600 after the rename" was literally true but was never a window of
  exposure. The test stays as a regression guard, proven by a deliberate
  break instead. D3's other defects were real: no `fsync`, and no size cap
  on `Load`.
* **2026-09-30, T2 step 3's masking.** The step says to mask with
  `internal/redact`'s `MaskSecret`.
  * `MaskSecret` reveals the last four runes on purpose, and its own
    documentation says never to use it on a value written to a log
    (`internal/redact/mask.go:22-26`).
  * `String`, `GoString` and `LogValue` exist for logs.
  * **The owner chose to hide secrets fully** (asked 2026-09-30, answered
    "Hide fully"): every secret field prints `[redacted]`, or nothing when
    empty. `MaskSecret` stays for the wizard's on-screen prompts.

  D5's "a masked form" is met, and no MADR text changes.
* **2026-09-30, T2 step 5's discovery test.**
  `TestOAuthEndpointsFor_GrokFallsBackAfterDiscoveryFailure` asserted MADR
  finding M6's behaviour: a *caller's* issuer (`https://issuer.example`)
  whose discovery fails silently gets xAI's built-in endpoints. D7 reverses
  that. The test is replaced by two:
  * `TestOAuthEndpointsFor_BuiltinIssuerFallsBack`, for both built-in
    issuers;
  * `TestOAuthEndpointsFor_CallerIssuerDiscoveryFailureFails`.

  This is the same kind of replacement as step 2's M4 test. Separately, ten
  login tests gained a signing issuer (discovery with `jwks_uri`, `/jwks`,
  signed `id_token`s, the Grok nonce echoed). Their fixtures grew; no
  assertion changed.
* **2026-09-30, the survey.** [0017-REPORT-reference-client-auth-survey.md](../reports/0017-REPORT-reference-client-auth-survey.md) found three points for T2.
  They are proposed as the MADR's A1–A3, and as "T2 additions from the
  survey". Nothing changes until the owner decides.

* **2026-09-30, T3 step 1's header override.** 0015-PLAN maps T3 step 1 to S7, per
  provider. The openai and claude commits (`00fe139`, `da4425a`) did not
  build the override or its `X-Custom` test: no code read `Token.Header`.
  Found before the gemini commit. The owner chose a commit of its own
  first, for openai and claude, generation and listing. Each later S7
  commit builds it for its own provider. Building it showed that the
  sources contradicted D2; the owner's decision is 0016-MADR A6. Recorded
  in full in 0015-PLAN, "Deviation 2026-09-30: T3 step 1's header override
  was not built".

* **2026-10-01, T4's open points.** Four choices that D11 and D5 left open
  were put to the owner during 0015-PLAN S8, commit 6, before any code:
  * Kilo's device token is stored only, with `Kind` `CredOAuth`;
  * logout is a function, `wizard.Logout`;
  * `Result`'s JSON is not redacted (T4.3 joins T2 step 3's table for fmt
    and slog only);
  * keeping a session reads the store only.

  Recorded as 0016-MADR A7-A10. Also, per A7, `Result.AccessToken` and
  `RefreshToken` are removed, because no path sets them. T4's steps are
  otherwise unchanged.

## Execution Record

### The owner's decisions (2026-09-29)

* The MADR is `accepted`, with the owner's answers to each open choice
  (MADR, "Owner's decisions"). 0015-MADR was accepted earlier the same day.
* **Changed here:** T2 step 5 (signature verification, with its tests and a
  live pin), T5 (`ANTHROPIC_API_KEY` only), V6 and V10, and the out-of-scope
  list. The transferred token-store record's path is corrected after
  0002-PLAN Phase 6 moved it.
* **Status.** This PLAN stays `proposed` until the owner approves
  execution. Its steps land inside 0015-PLAN's phases, so T0 runs when
  0015-PLAN S0 is approved.

### T0: cross-references (2026-09-29)

* The owner answered "Proceed" after the MADR's decisions were recorded, so
  this PLAN is `in-progress`.
* 0015-MADR gains the proxy exemption (T0 step 2), and 0015-PLAN gains its S10
  annotation and a table of which 0016 step lands in which phase (T0 step 3).
* 0002-MADR §13 names `prepare-commit-msg`'s new obligations (T0 step 4).
* `docs/README.md` shows this PLAN `in-progress` (T0 step 5).
* The commit holds records only (bootstrap exception).

### T1: transport (2026-09-30, in 0015-PLAN S3)

* **Where.** In `llmprovider`, as the Deviation Log's first entry says:
  `options.go` and `oauth_session.go`, called from `openai_chatgpt.go` and
  `grok.go`.
* **Step 1.** `defaultHTTPClient` sets `Proxy: http.ProxyFromEnvironment`.
  Each provider already built one client in its constructor
  (`ApplyOptions`) and used it for listing and probes. Its OAuth session now
  receives it too, when it has none (`shareHTTPClient`).
* **Step 2, test `TestDefaultClient_HonoursProxy`**
  (`llmprovider/transport_defaults_test.go`).
  * Each of the 15 G-wire cases is built without `WithHTTPClient`, aimed at
    `http://<case>.invalid`, and must generate "hello" through
    `HTTP_PROXY`, an `httptest` server that answers with that case's canned
    reply.
  * The parent checks that every case's host reached the proxy.
  * **A child process.** `net/http` reads the proxy variables once per
    process, so the providers run in a child of the test binary that starts
    with only the test's proxy set. The host's own proxy variables are
    removed.
  * **Why `.invalid`.** Loopback is never proxied, so the base URL is a
    non-loopback host that cannot resolve: only a proxied request can
    succeed.
* **Step 3, test `TestProviderClient_SharedWithListingAndRefresh`,** for
  `chatgpt` and `grok`. An expired session with no client is given to a
  provider built without `WithHTTPClient`. The provider's client is wrapped
  in a path recorder. `Generate` and `DiscoverModels` must send
  `POST /oauth/token`, `POST /responses` and `GET /models` through it.
  `TestShareHTTPClient_KeepsTheSessionsOwn` covers a session's own client, a
  nil client and a non-session source.
* **Red first,** against a clean clone of `51ebc85` in scratch, with the two
  tests copied in (the helper's own test names the new function and cannot
  compile there). Exit 1:
  * `openai: Generate = "", Post "http://openai.invalid/responses": dial tcp:
    lookup openai.invalid: no such host; want "hello" through the proxy`, and
    the same for all 15 cases;
  * `chatgpt`: `POST /oauth/token did not go through the provider's client
    (it carried [POST /responses GET /models])`, and the same for `grok`.
* **Seen to fail on deliberate breaks,** in scratch copies, for the two
  assertions the pre-change code already met:
  * the ChatGPT listing given its own `defaultHTTPClient()`:
    `GET /models did not go through the provider's client (it carried
    [POST /oauth/token POST /responses])`;
  * the guard in `shareHTTPClient` removed:
    `a session's own client was replaced`.
* **V7** (proxy honoured; one client per provider) is met.
  *Annotated 2026-10-01:* from `00fe139` on, the proxy test lost each
  provider that 0015-PLAN S7 moved, with no replacement. At `20b6b9e` it
  covered only `together`. 0015-PLAN, "Phase S7, commit 13", moved the test
  over `providers.Default()`, which covers every built-in provider again.

### T2 step 1 and A2: durable writes and the refresh lock (2026-09-30, in 0015-PLAN S4)

* **Durable writes (D3).** `FileTokenStore.Save` now:
  * creates a temp file in the store directory and sets `0600` before any
    byte is written;
  * writes, `Sync`s, closes and renames;
  * then `fsync`s the directory (`syncDir`, a no-op on Windows).

  On any failure the temp file is removed and the previous file is intact.
  `Load` refuses a file over 64 KiB (`readBounded`). The old post-rename
  `chmod0600` helper is gone; `tmp.Chmod(0o600)` replaces it on every
  platform.
* **The refresh lock (A2, option (b)).**
  * `RefreshLocker` is an optional `TokenStore` interface, and
    `FileTokenStore.LockRefresh` implements it:
    * `{provider}.lock` is created exclusively, holding the pid and the time;
    * it is touched every 5 s while held;
    * it is taken over when untouched for 30 s;
    * a waiter gives up after 25 s with an error matching
      `ErrProviderUnavailable`, and never refreshes unlocked.
  * `reloadOrRefresh` takes the lock before its store re-read, so a session
    adopts what another process saved while it waited.
* **Tests** (`tokenstore_durable_test.go`, and `tokenstore_file_unix_test.go`
  for the mode):
  * a rename failure keeps the previous session and leaves no temp file;
  * the temp file is `0600` and empty before its first write;
  * an oversized file is refused;
  * two sessions on one store refresh once in total, and both get the
    refreshed token;
  * a stale lock is taken over;
  * a held lock times out as retryable, and the heartbeat keeps it from
    being taken over.

  All pass under `-race`.
* **Red first,** against a clean clone of `fe625b8`, with tests using the
  old API only. Exit 1:
  * `Load of a 65569-byte file succeeded, want a size refusal`;
  * `2 refresh requests, want 1: the same refresh token was spent twice`.
    That is the A2 race, observed.
  * The premise check passed: `os.CreateTemp mode: 600` (see the Deviation
    Log).
* **Seen to fail on deliberate breaks,** each in its own scratch copy:

  | Break | Failure |
  |---|---|
  | temp chmod `0644` | `before the first write: mode 644, size 0; want 0600 and empty` |
  | temp file not removed | `temp file .tok-3449366731.json left behind` |
  | no size cap | `Load = FileTokenStore decode: unexpected end of JSON input, want a size refusal` |
  | stale window ×1000 | `Token = "", llm: provider unavailable: oauth: another process is refreshing the openai session after 0 refreshes; want the stale lock taken over` |
  | refresh unlocked after the wait | `Token = <nil>, want a retryable ErrProviderUnavailable while the lock is held` |
  | no heartbeat | `Token = <nil>, want a retryable ErrProviderUnavailable while the lock is held` |
  | no lock around the refresh | `2 refresh requests, want 1: the same refresh token was spent twice` |

  A first attempt at the stale-window break did not compile, and so proved
  nothing. It was rewritten, and the break runner now reports a break that
  does not compile as invalid.
* **Gate,** every step exit 0:
  * `make pre-add-check`;
  * `go vet` for three GOOS and `live_gateways`;
  * `go test -race -count=1 -cover ./...` (`llmprovider` 90.2 %);
  * `go mod tidy -diff`;
  * `make lint` (`0 issues.`);
  * `make parity-check`;
  * markdownlint;
  * G-wire three times over, unchanged;
  * G-links;
  * the deny list.

### T2 step 2 and A1: the rotation is kept, and a sibling's is adopted (2026-09-30, in 0015-PLAN S4)

* **D4.** `OAuthSession.Token` adopts a refreshed session before it is
  saved.
  * If `Save` fails, `Token` still returns the fresh token, and reports the
    failure to the session's new `Logger` field (nil logs nothing; never the
    global logger).
  * It remembers the refresh token the refresh spent, and retries the save
    on the next call.
  * The retry runs under the store's refresh lock, and writes only while
    the store still holds that spent token, or nothing. When another process
    has saved since, the retry is dropped (`errStoreMovedOn`), logged, and
    not repeated. So a stale rotation never overwrites a newer one.
* **`Logger` is a new field, ahead of 0015's `WithLogger`.** D4 names "the
  logger passed with `WithLogger`", which does not exist until 0015-PLAN
  S6. An `OAuthSession` field is its place until then. S6 wires the option
  to it.
* **A1.** `reloadOrRefresh` returns a `refreshed` value, carrying the save
  error separately. After a permanent refresh failure (one matching
  `ErrAuthFailure`) it re-reads the store once. If the store holds a
  different refresh token, it adopts that session: its token when current,
  otherwise one refresh with the stored token.
* **An existing test pinned the defect D4 fixes.**
  `TestOAuthSession_RefreshPersistsBeforeReturn` asserted MADR finding M4's
  behaviour:
  * the save error returned;
  * the session keeping its old tokens.

  D4, which the owner accepted, reverses exactly that. T2 step 2 requires
  the replacement test. It is replaced by
  `TestOAuthSession_SaveFailureKeepsRotation`, which keeps the old test's
  checks on the request's encoding. No other assertion changed meaning.
* **Tests:**
  * `TestOAuthSession_SaveFailureKeepsRotation`: the fresh token despite the
    failed save; the rotation adopted and logged; the next forced refresh
    sending the rotated token; the save retried once on the next call, and
    not again after it succeeds;
  * `TestOAuthRefresh_RejectedAdoptsSiblingRotation`;
  * `TestOAuthRefresh_RejectedRefreshesWithSiblingToken`;
  * `TestOAuthRefresh_UnsavedRotationNeverOverwritesNewer`.
* **A test that reached a real service, found and fixed before commit.**
  The first version of the sibling-refresh test saved a sibling session with
  no `TokenURL`, so its second refresh went to
  `https://auth.openai.com/oauth/token`. That endpoint answered 401
  `invalid_client`: it was sent only the test's dummy refresh token and
  client id. The test now gives the sibling the test server's URL.
* **Red first,** against a clean clone of `0cd659d` (step 1 committed, this
  step not), with tests using that API. Exit 1:
  * `Token = "access-1", save failed; want the fresh access-1 despite the
    failed save`;
  * `refresh tokens sent = [old-refresh old-refresh], want [old-refresh
    refresh-1]`. That is M4's spent-token reuse, observed.
  * `Token = "", llm: authentication failed: oauth: refresh failed: 400 Bad
    Request: {"error":"refresh_token_reused"}; want the sibling's session
    adopted`.
* **Seen to fail on deliberate breaks,** each in its own scratch copy:

  | Break | Failure |
  |---|---|
  | save never retried | `retry saved "refresh-2" after 0 saves; want refresh-2 saved once more` |
  | retry overwrites a newer session | `store holds &{… Refresh:rt-new …}, <nil>; the newer session must not be overwritten` |
  | rotation not logged | `log = "", want the unsaved rotation reported` |
  | no re-read after a rejection | both A1 tests: `refresh failed: 400 Bad Request: {"error":"refresh_token_reused"}` |
  | save error returned | `Token = "access-1", save failed; want the fresh access-1 despite the failed save` |

  The overwrite failure printed a whole session, secrets included, with
  `%+v`: the leak step 3 (D5) closes.
* **Gate,** every step exit 0:
  * `go test -race -count=1 -cover ./...`: `llmprovider` 90.3 %;
  * `make lint` (`0 issues.`);
  * G-wire, unchanged;
  * the rest of the 0015-PLAN §0 gate.

### T2 step 3: secret-bearing values redact themselves (2026-09-30, in 0015-PLAN S4)

* **D5.** These types implement `String`, `GoString` and `slog.LogValuer`:
  * `Token` and `StaticToken`, on value receivers;
  * `*OAuthSession`, which takes the session lock to read consistently.

  Each keeps its non-secret fields (type, header, expiry, provider, issuer,
  client id, account id) and prints `[redacted]` for a secret, per the
  Deviation Log's entry for this step. `wizard.Result` is T4's.
* **Tests** (`redaction_test.go`): a table over `Token`, `*Token`,
  `StaticToken`, `*StaticToken`, `*OAuthSession` and a struct holding a
  `Token`.
  * Each is formatted with `%v`, `%+v`, `%#v` and `%s`, and through slog's
    JSON and text handlers.
  * The test fails if a planted secret appears, or a non-secret field is
    lost.
  * Also covered: an empty secret, a nil session, and a zero expiry.
* **Open item, for the owner (not in T2 as planned).** slog's JSON handler
  encodes a struct that *holds* a `Token` with `encoding/json`, which never
  calls the nested `LogValue`, so the secret shows.
  * Only a redacting `MarshalJSON` on the three types would close it. That
    would also change any caller's deliberate JSON encoding of a token.
  * The test names this one form as a known gap instead of passing silently.
  * Every form the step lists is covered, and so is the nested struct
    through `fmt` and the text handler.
* **Red first,** against a clean clone of `3a627d6`. Exit 1: every type
  leaked in every form. For example:
  * `Token via %v shows the secret: {tok-PLANTED-SECRET-7f3a bearer … Authorization}`;
  * `*OAuthSession via %+v shows the secret: &{Provider:grok Access:at-PLANTED-SECRET-7f3a Refresh:rt-PLANTED-SECRET-7f3a …}`.
* **Seen to fail on deliberate breaks:**
  * `Token.String` printing its value: `Token via %v shows the secret: Token{… Value:tok-PLANTED-SECRET-7f3a}`;
  * `OAuthSession.LogValue` printing the access token: `*OAuthSession via slog JSON shows the secret`;
  * `StaticToken.String` dropping its header: `StaticToken via %v lost the non-secret "x-api-key"`.
* **Gate,** every step exit 0: `llmprovider` 90.3 %, `make lint`
  `0 issues.`, and G-wire unchanged.

### T2 step 4: device login returns a handle (2026-09-30, in 0015-PLAN S4)

* **D6.** `StartDeviceOAuth(ctx, provider, opts)` requests the code and
  returns a `*DeviceLogin` with `UserCode`, `VerificationURI` and `Expiry`,
  before any polling.
  * **`Wait(ctx)`** polls. It returns the session, the flow's error, or,
    after `Cancel`, an error matching `context.Canceled`.
  * **Concurrent or later `Wait`s** share the first one's result.
  * **`Cancel()`** stops a running `Wait` through its context, and makes a
    later `Wait` return at once without polling.
  * The OpenAI and Grok flows are split into a start, which requests the
    code, and a poll closure. The wire requests are unchanged.
* **`LoginDeviceOAuth` keeps its signature and callback,** now as start,
  `NotifyDevice`, then `Wait`. So `wizard`, and every existing device test,
  run unchanged through the handle. Moving `wizard` onto the handle itself
  is T4's (D6: "`wizard` uses the handle; its prompts are unchanged").
* **Tests** (`device_login_test.go`):
  * the Grok handle's fields before polling, then `Wait`'s session;
  * `Cancel` during real polling (1 s interval) returning within one
    interval, with `context.Canceled` and no approval;
  * `Cancel` before `Wait` returning at once with no poll;
  * three concurrent `Wait`s sharing one approval;
  * the OpenAI handle's device page and 15-minute window.

  The existing `slow_down`, expiry and validation tests pass unchanged in
  meaning.
* **Red first** is by absence: the old code has no handle. Seen to fail on
  deliberate breaks instead:

  | Break | Failure |
  |---|---|
  | `Cancel` does not stop a running `Wait` | `Wait did not return after Cancel` |
  | `Cancel` before `Wait` ignored | `Wait after Cancel = oauth: device login canceled: context canceled after 3.002177625s and 2 polls; want context.Canceled at once, without polling` |
  | a later waiter polls again | `3 approvals, sessions … ; want one poll shared` |

  **Two first attempts proved too little, and were corrected:**
  * The first `Cancel`-before-`Wait` test passed on its break. The broken
    `Wait` polled to expiry and still ended in `context.Canceled`. The test
    now also requires a prompt return and zero polls.
  * The first shared-waiter break panicked (`close of closed channel`)
    instead of reaching the assertion. It was replaced by one that re-polls
    for a later waiter.
* **Gate,** every step exit 0: `llmprovider` 90.5 %, `wizard` 83.4 %,
  `make lint` `0 issues.`, and G-wire unchanged.

### T2 step 5 and A3: `id_token` verification and fail-closed discovery (2026-09-30, in 0015-PLAN S4)

* **D7 as decided, with A3.** `oauth_idtoken.go` verifies every login's
  `id_token` before any claim is used.
  * **Keys:**
    * a `kid` is required;
    * keys come from discovery's `jwks_uri`, cached for 1 h, and refetched
      exactly once for an unknown `kid`;
    * keys that cannot be fetched fail.
  * **Algorithms:** RS256/384/512, PS256/384/512, ES256/384 and EdDSA, all
    through the standard library. They are intersected with the issuer's
    `id_token_signing_alg_values_supported` when it lists them. `none` and
    HMAC are refused.
  * **Claims:** `iss` equals the issuer, `aud` names the client, `exp` is in
    the future (60 s leeway), and for a Grok browser login the `nonce`
    matches the one sent.
  * **Where it applies:** the browser login and both device flows.
    `verifiedSession` fails a response without an `id_token`, since every
    login requests `openid`. RSA keys under 2048 bits are refused, and EC
    points are validated on their curve.
* **Nonce plumbing.** `LoginBrowserOAuth` generates the Grok nonce and passes
  it both to `buildAuthorizeURL` and to verification. Before, the nonce was
  generated inside `buildAuthorizeURL` and discarded.
* **Discovery fails closed.** `oauthEndpointsFor` falls back to built-in
  endpoints and keys only for `https://auth.openai.com` and
  `https://auth.x.ai`. A caller's issuer whose discovery fails is an error.
  * The OpenAI flows now fetch discovery for their keys. Their authorize and
    token endpoints stay Codex's fixed ones, so no request changes.
  * The built-in fallback keys are `defaultOpenAIJWKSURL` and
    `defaultGrokJWKSURL`.
  * Each login checks the endpoints it needs (`requireEndpoints`).
    Revocation, which shares discovery, needs no keys, so the check lives
    in the flows, not in discovery.
* **Client ids.** `oauth_constants.go`'s doc says the default client ids are
  the vendor CLIs' own, borrowed, and overridable. It becomes `auth`'s
  package doc in S7b.
* **Tests:**
  * `oauth_idtoken_test.go`: every accepted algorithm; 16 rejections
    (tampered, audience, issuer, expiry, no `exp`, `none`, HS256, no `kid`,
    an algorithm the issuer does not advertise, a key of the wrong type, a
    nonce mismatch or absence, no `jwks_uri`, unreachable keys, not a JWT);
    the unknown-`kid` refetch counted exactly; a missing `id_token`; and
    the built-in-only fallback.
  * `oauth_login_verify_test.go`: a Grok browser login that fails on a nonce
    mismatch, a tampered payload, `alg: none` and a missing `id_token`, and
    passes with a valid token; and a caller-issuer device login failing on
    discovery.
  * The shared signing issuer is `oauth_testissuer_test.go`.
* **No test reaches the network.** The discovery-failure test uses a
  transport that answers everything itself, including the requests a
  regression would send to `https://auth.x.ai`.
* **Red first,** against a clean clone of `fbe8bcf`, with the login-level
  tests, which use only APIs that exist there. Exit 1:
  * `login error = <nil>, want error true`, four times (nonce mismatch,
    tampered, `alg: none`, no `id_token`). The old code accepted each.
  * `StartDeviceOAuth succeeded against an issuer whose discovery fails
    (requests: [https://issuer.example/.well-known/openid-configuration
    https://auth.x.ai/oauth2/device/code])`: the silent fallback of M6,
    intercepted by the test's transport.
* **Seen to fail on deliberate breaks,** each check alone:

  | Break | Failure |
  |---|---|
  | signature not checked | `verifyIDToken = <nil>, want … "bad signature"` |
  | nonce not compared | `… naming "nonce"`, and the login test's `login error = <nil>` |
  | audience not checked | `… naming "audience"` |
  | no `kid` accepted | `… naming "no key id"` |
  | advertised algorithms ignored | `… naming "\"RS256\" is not allowed"` |
  | unknown `kid` not refetched | `no key "ec-1" after 1 fetches; want a rejection after exactly one refetch` |
  | expiry not checked | `… naming "expired"` |
  | any Grok issuer gets the fallback | `grok: oauthEndpointsFor = <nil>, want the discovery failure` |
  | a missing `id_token` not refused | `verifiedSession = … not a signed JWT, want a missing-id_token rejection` |

* **Live pin,** `live_oauth_issuers_test.go` (`live_gateways`). It reads
  public discovery only, so it has no variable. Run on 2026-09-30, it
  passed:
  * `https://auth.openai.com`: `jwks_uri https://auth.openai.com/.well-known/jwks.json`, algorithms `[RS256]`;
  * `https://auth.x.ai`: `jwks_uri https://auth.x.ai/.well-known/jwks.json`, algorithms `[ES256]`.

  With the Grok fallback URL changed in a scratch copy, it failed:
  `issuer "https://auth.x.ai" jwks_uri "https://auth.x.ai/.well-known/jwks.json"; want … "https://auth.x.ai/keys"`.
* **Lint.** Four findings in the new code were fixed:
  * three `goconst`, for the algorithm names;
  * a `staticcheck` SA1019, on building an `ecdsa.PublicKey` from
    coordinates. It now uses `ecdsa.ParseUncompressedPublicKey`, which also
    validates the point.
* **Gate,** every step exit 0:
  * `go test -race -count=1 -cover ./...`: `llmprovider` 90.1 %,
    `wizard` 83.4 %;
  * `make lint` `0 issues.`;
  * G-wire, unchanged;
  * the rest of 0015-PLAN §0.
* **V2–V6 are met for T2:**
  * V2, durable writes;
  * V3, rotation kept;
  * V4, in part: no secret formatted, with `wizard.Result` in T4;
  * V5, the device handle;
  * V6, the OAuth checks.

  V4's nested-JSON case is an open item, in step 3's record.

### T2 step 3, addition: `MarshalJSON` redaction (2026-09-30, the owner's decision)

* **Closes step 3's open item.** The owner answered "add the MarshalJSON
  redaction" (0016-MADR, amendment A4).
* **The methods.** `Token` and `StaticToken` (value receivers) and
  `*OAuthSession` (under its lock; `null` for nil) implement `MarshalJSON`,
  with the same fields as `LogValue` and `[redacted]` for each secret.
* **Checked first:** no non-test code in the module JSON-encodes these
  types. The encoders are request bodies, and `FileTokenStore`'s
  `fileRecord`.
* **The test.** `TestSecretBearingTypesRedact` gains a `json.Marshal` form
  for every row. It loses its known-gap skip, and its exemption of the
  nested struct from the non-secret check. `TestSecretText` covers a nil
  session's JSON.
* **Red first,** against a clean clone of `9a264db`. Exit 1, with seven
  leaks:
  * `json.Marshal` of `Token`, `*Token`, `StaticToken`, `*StaticToken`,
    `*OAuthSession` and the struct;
  * the struct through slog's JSON handler.

  For example: `*OAuthSession via json.Marshal shows the secret: {"Provider":"grok","Access":"at-PLANTED-SECRE…`.
* **Seen to fail on deliberate breaks:**
  * `Token.MarshalJSON` encoding its value: `Token via json.Marshal shows the secret`, and `struct holding a Token via slog JSON shows the secret`;
  * `OAuthSession.MarshalJSON` encoding the refresh token: `*OAuthSession via json.Marshal shows the secret`.
* **V4 is met for T2's types.** `wizard.Result` remains T4's.

### T3 step 2: listing sends no billed probe by default (2026-09-30, in 0015-PLAN S5)

* **D9.** `WithModelProbes(true)` turns the listing probe on. Without it,
  `DiscoverModels` returns the listing and sends no generation. The five
  G-wire listing goldens lost 9 probe `POST`s and nothing else. The test is
  `TestDiscoverModels_ProbesOnlyWhenEnabled`. Details, and the breaks that
  prove it, are in 0015-PLAN's S5 record. **V8 is met.**

### Amendment 2026-09-30: T3 step 2 reversed (0016-MADR A5)

* T3 step 2 as executed turned probes off by default. The owner reversed
  that the same day (0016-MADR A5):
  * probes are on by default;
  * `WithModelProbes(bool)` enables or disables them;
  * `ModelProbesFromEnv()` reads `LLMPROVIDER_PROBES` for callers who opt
    in.
* V8 ("no billed probe by default") is withdrawn. Its successor: probes
  follow the default, the option and the helper, as A5 states.
* Executed in 0015-PLAN S5's amendment of the same date.

### Execution 2026-09-30: T3 step 2 reversed (0016-MADR A5)

* Probes are on by default. `WithModelProbes(bool)` sets them, and
  `ModelProbesFromEnv()` reads `LLMPROVIDER_PROBES` for a caller who opts
  in. The five listing goldens are back to their content before S5. The
  test is `TestDiscoverModels_ProbesFollowDefaultOptionAndEnv`. Details,
  and the breaks that prove it, are in 0015-PLAN's record of the same date.
  **A5 is met.**

### T3 step 1: the header override, openai and claude (2026-09-30, in 0015-PLAN S7)

* **D2, A6.** A token names a `Header` only when its caller set one. An
  overriding `Header` carries a `TokenBearer` as `Bearer <value>`, and
  anything else bare.
  * openai and claude apply the rule in generation and in their listings.
  * The table's headers for them are unchanged, as re-read from the moved
    sources: `Authorization: Bearer` and `x-api-key`.
  * G-wire shows no difference.
* **Tests:** `TestOpenAI_TokenHeaderOverride` and
  `TestClaude_TokenHeaderOverride` are the step's `X-Custom` test, each red
  first. The refusal of OAuth sources for claude landed in its S7 commit.
* **Still to do.** Each later S7 commit applies the rule to the provider it
  moves. Details are in 0015-PLAN, "Phase S7, commit 5".

### T3 step 1: the header override, gemini (2026-09-30, in 0015-PLAN S7)

* gemini applies the rule in generation and in its listing. Its header is
  unchanged: `x-goog-api-key`, as the table says, re-read from the moved
  source. G-wire shows no difference.
* **Test:** `TestGemini_TokenHeaderOverride`. Breaking generation or the
  listing fails it. Details are in 0015-PLAN, "Phase S7, commit 6".

### T3 step 1: the header override, grok (2026-09-30, in 0015-PLAN S7)

* grok applies the rule in generation and in its listing. Its header is
  unchanged: `Authorization: Bearer`, as the table says, re-read from the
  moved source. G-wire shows no difference.
* **Test:** `TestGrok_TokenHeaderOverride`. Breaking generation or the
  listing fails it. Details are in 0015-PLAN, "Phase S7, commit 7".

### T3 step 1: the header override, opencode (2026-09-30, in 0015-PLAN S7)

* Both gateways apply the rule in generation and in their listing.
* The table's per-route headers are unchanged: `x-api-key` on messages,
  `x-goog-api-key` on google, else `Authorization: Bearer`, re-read from
  the moved source. G-wire shows no difference.
* **Test:** `TestOpencode_TokenHeaderOverride`, on all four routes.
  Breaking generation or the listing fails it.
* An OAuth or CLI source is refused, as the step asks. Details are in
  0015-PLAN, "Phase S7, commit 9".

### T3 step 1: the header override, kilo (2026-10-01, in 0015-PLAN S7)

* kilo applies the rule in generation and in its listing. Its header is
  unchanged: `Authorization: Bearer`, re-read from the moved source. G-wire
  shows no difference.
* **Test:** `TestKilo_TokenHeaderOverride`. Breaking generation or the
  listing fails it.
* An OAuth or CLI source is refused. Details are in 0015-PLAN, "Phase S7,
  commit 10".

### T3 step 1: the header override, huggingface (2026-10-01, in 0015-PLAN S7)

* huggingface applies the rule in generation and in its listing. Its header
  is unchanged: `Authorization: Bearer`. G-wire shows no difference.
* **Test:** `TestHuggingFace_TokenHeaderOverride`. Breaking generation or
  the listing fails it.
* An OAuth or CLI source is refused. Details are in 0015-PLAN, "Phase S7,
  commit 11".

### T3 step 1: the header override, ollama (2026-10-01, in 0015-PLAN S7)

* Re-read: Ollama sends no credential header (`ollama.go:176` at `4ddcb54`,
  `providers/ollama/ollama.go` now). The row is unchanged.
* With no `Header`, nothing is sent, so G-wire shows no difference. A token
  naming a `Header` is sent there, in generation and in the listing.
* **Test:** `TestOllama_TokenHeaderOverride`. Breaking generation or the
  listing fails it.
* An OAuth or CLI source is refused. Details are in 0015-PLAN, "Phase S7,
  commit 12".

### T3 step 1: the header override, together (2026-10-01, in 0015-PLAN S7)

* Together was added after `4ddcb54` (0017-PLAN U1), so the table has no
  row for it. Read from the source before the change: `Authorization:
  Bearer`, in generation (`together.go:140` at `98a3f9d`) and in the listing
  (`discovery.go`).
* Both now apply the rule. G-wire shows no difference.
* **Test:** `TestTogether_TokenHeaderOverride`. Breaking generation or the
  listing fails it.
* An OAuth or CLI source is refused. Details are in 0015-PLAN, "Phase S7,
  commit 14".
* Every provider now applies the rule; V1's T3.1 tests are complete.

### T4: wizard (2026-10-01, in 0015-PLAN S8, commit 6)

Under 0016-MADR A7-A10, the owner's decisions of the same day, recorded
before the code. Details are in 0015-PLAN, "Phase S8, commit 6".

* **Step 1, D11 (A7).** A stored session leaves no token in `Result`: every
  sign-in returns `CredOAuth` with the non-secret fields only, and Kilo's
  device login is one. `Result.AccessToken` and `RefreshToken` are removed.
  * **Red first,** against `wizard/configure.go:137-145` and
    `wizard/auth.go:176-180`:
    * `TestConfigureLLM_StoredSessionLeavesNoTokenInResult`: `Result fields
      [AccessToken RefreshToken] hold a token; the store is the only copy`;
    * `TestConfigureLLM_KiloDeviceLogin`: `Result kind "api_key" …; want
      oauth` and `Result fields [APIKey] hold the token`.
* **A10.** Keeping a session reads the store.
  * **Red first:** `TestConfigureLLM_KeepsTheStoredSession`, `signed in
    again; want the stored session kept`.
  * `TestConfigureLLM_NoStoredSessionSignsIn` guards the empty store; it
    passed before the change too, as it should.
* **Step 2, D11 (A8).** `wizard.Logout` confirms, revokes (OpenAI and
  Grok), reports a failed revocation, and deletes.
  * **Red first:** the tests did not compile, `undefined: Logout`.
  * **Tests:** `TestLogout_RevokesThenDeletes`, a revoke server that
    answers 200 and one that answers 500, both ending with the store empty;
    `TestLogout_NoRevocationForKilo`; `TestLogout_LeavesTheStoreAlone`; and
    `TestLogout_Errors`.
* **Step 3, D5 (A9).** `Result` has `String`, `GoString` and `LogValue`,
  which print `[redacted]` for `APIKey`.
  * **Red first:** `TestResult_Redacts`, with `Result via %v shows the key`
    and the same for every verb and both slog handlers.
  * `json.Marshal` keeps the key, and the test pins it.
  * One form can still show the key: slog's JSON handler, given a struct
    *holding* a `Result`, encodes it with `encoding/json`. The test exempts
    that form, citing A9.
* **V9 is met.** **V4 is met,** with A9's exception for JSON.

### T5: environment helper (2026-10-02, in 0015-PLAN S10)

* `ANTHROPIC_API_KEY` only (D12, the owner's decision of 2026-09-29).
  `ProviderEnvVars()` and the Claude descriptor's `EnvVar` name it, which is
  the variable `wizard`'s `AllowEnv` reads, through `Options.LookupEnv`.
  `CLAUDE_API_KEY` is not read.
* **Red first,** against `llmprovider/provider.go:60`:
  `TestConfigureLLM_AnthropicKeyOnly` failed with `ProviderEnvVars()[claude] =
  "CLAUDE_API_KEY"`; a `CLAUDE_API_KEY` key was offered; an
  `ANTHROPIC_API_KEY` key was not found.
* The step's "opt-in helper" for the key has no function of its own. The
  key is read only where a caller passes a reader: `wizard`'s
  `Options.LookupEnv`. Details are in 0015-PLAN, "Phase S10".
* **V10 is met.**

### T6: close-out (2026-10-02)

The owner chose to close this PLAN out ("0016 T6 close-out"). T1–T5 ran
inside 0015-PLAN's phases, whose records hold the detail cited here.

* **Step 1, the execution record.** Each decision's first-seen failure, by
  the step that holds it:

  | Decision | Test first seen to fail | Where |
  |---|---|---|
  | D2 | `TestTokenHeader` and `TestTokenSources_ReportOnlyWhatIsSet`, red first on `da4425a`, and breaks | 0015-PLAN S7 commit 5; per provider, T3 step 1 above |
  | D3 | durable writes and the refresh lock, red first on `fe625b8` | T2 step 1 |
  | D4 | the rotation kept and a sibling's adopted, red first on `0cd659d` | T2 step 2 |
  | D5 | every secret-bearing type formatted, red first on `3a627d6`; `MarshalJSON`, red first on `9a264db` | T2 step 3, and its addition |
  | D6 | the device handle: red first by absence, then breaks | T2 step 4 |
  | D7 | four login refusals and the discovery fallback, red first on `fbe8bcf`; nine breaks; the live pin | T2 step 5 |
  | D8 | the shared client and the proxy, red first on `51ebc85` | T1 |
  | D9, as A5 | `TestListModels_ProbesFollowDefaultOptionAndEnv` (`openai`, `claude`, `gemini`, `grok`), `TestListModels_NeverProbes` (`kilo`, `together`) | T3 step 2 and its reversal |
  | D11 | `TestConfigureLLM_KeepsTheStoredSession` and the `Logout` tests, red first | T4 |
  | D12 | `TestConfigureLLM_AnthropicKeyOnly`, red first | T5 |

  * D1 chooses the base, and D10 says what the SDK does not do: neither
    has a behaviour to test.
  * **G-wire.** D2 and D8 show no request difference, in every step's
    record. D9's listing difference, the probe `POST`s, was removed in S5
    and restored by A5. So the goldens now match their content before S5,
    and no D9 difference remains.
  * **Coverage,** today, from `make coverage-check`, which passes:
    `llmprovider` 98.2 % and `wizard` 84.7 %, against `P7`'s 89.2 % and
    83.4 %; `auth` 85.3 %.
  * **Verification,** each criterion:
    * V1–V7, V9 and V10 are met, by the steps above;
    * V8 is met as A5 reversed it: the table is annotated;
    * V11 is met. `make dep-check` passes in CI (run `37046592893`, and on
      the `v1.0.0` tag), and `go.mod` requires only `golang.org/x/term`.
* **Step 2, the guides.**
  * `api-standards.md` already carries D2, as R15 and R16, and D5, as R33
    and R34, each citing 0016. 0015-PLAN S1 and S11 wrote them, so nothing
    is added there.
  * `adding-a-provider.md` stated the `TokenSource` rule but cited only R16.
    It now cites this MADR's D2, and A6 where the token is applied.
* **Step 3.** This PLAN is `complete`. The MADR's Confirmation items are
  annotated as met.
