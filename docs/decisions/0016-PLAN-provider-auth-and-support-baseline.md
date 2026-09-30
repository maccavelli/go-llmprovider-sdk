---
status: in-progress
date: 2026-09-29
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
   connection limits. A provider builds one client in `New` and passes it
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
| V8 | No billed probe by default | T3.2 test | T3 |
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

None.

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
