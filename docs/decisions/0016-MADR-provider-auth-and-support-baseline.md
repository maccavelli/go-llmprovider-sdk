---
status: accepted
date: 2026-09-30
decision-makers: go-llmprovider-sdk maintainers
consulted: owners of mcplib and magic-cli-remote
informed: every consumer of github.com/maccavelli/go-llmprovider-sdk; mcp-server-magicdev, mcp-server-magictools, prepare-commit-msg, pi-go
---
# Build Provider Support and Authentication from mcplib's `llmprovider`, and Adopt magic-cli-remote's Credential Hygiene

## Context and Problem Statement

This repository is to become a standalone SDK for LLM provider access and
provider authentication.
[0002-MADR-migrate-llmprovider-from-mcplib.md](0002-MADR-migrate-llmprovider-from-mcplib.md)
moves `mcplib`'s `llmprovider` and `wizard` here.
[0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md)
(proposed) decides the API shape. Both assume that the `mcplib` code is the
base. Neither compared it with the fleet's other body of provider and
credential code: `magic-cli-remote`.

On 2026-09-29 the owner asked for both codebases to be evaluated for LLM
provider **auth** and **support**, to establish the core the SDK is built
from. The companion `mcplib` record,
`mcplib` `docs/decisions/0015-MADR-transfer-llmprovider-to-go-llmprovider-sdk.md`
(accepted, with an amendment for the canonical API), was read for context.
It decides only `mcplib`'s side and is unaffected by this record.

The question: **which codebase supplies the SDK's provider support and
authentication, and which of the other's practices does the SDK adopt?**

### Method

Evidence is marked **(M)** measured, or **(R)** read in the source.

* **`mcplib` code.** This repository's `llmprovider/`, `wizard/` and
  `internal/redact/` at `4ddcb54`. `diff -rq` against the `mcplib` clone
  exits 0 for both packages: the tree is byte-identical to `mcplib`
  `4e1f9a5` (`v1.6.0`) and not yet re-homed. Paths below are this
  repository's. There is no `go.mod` yet, so the `go` commands ran in the
  `mcplib` clone, whose tree they left unchanged.
* **`magic-cli-remote` code.** Its `internal/provider/**`,
  `internal/providerauth/`, `internal/auth/` and its records, at
  `15b991ce`. Paths below are that repository's and are marked `mcr:`.
* **Counts.** `grep -c '^func Test'` over `*_test.go`. A file counts as live
  when its `//go:build` line names a live tag.

### Findings: `mcplib` `llmprovider` and `wizard`

* **M1 — It is an LLM API client (R).** There are nine provider ids
  (`llmprovider/constants.go:5-21`) across five wire formats:
  * Responses: `openai`, and `grok` (`openai.go:180`, `grok.go:215`);
  * Messages: `claude` (`claude.go:223-230`);
  * Gemini Interactions: `gemini` (`gemini.go:215-222`);
  * Chat Completions: `huggingface`, `kilo`, `ollama`;
  * per-model routing for `opencode-zen` and `opencode-go` over Responses,
    Messages, Chat Completions and Gemini `generateContent`, the fifth
    format (`opencode_route.go:44-56`).

  ChatGPT (Codex subscription) is a mode of `openai`
  (`openai_chatgpt.go:30-34`). All of it is `net/http` against the
  vendors' APIs.
* **M2 — Subscription OAuth is implemented and follows the vendor clients
  (R).**
  * Browser PKCE loopback, `LoginBrowserOAuth` (`oauth_loopback.go:76`),
    with S256 (`oauth_pkce.go:15-25`). OpenAI binds the Codex CLI's fixed
    ports; Grok uses an ephemeral port with a CORS allowance for one origin
    (`oauth_loopback.go:267-305,397-418`).
  * Device code, `LoginDeviceOAuth` (`oauth_device.go:74`): OpenAI's
    proprietary endpoints and RFC 8628 for Grok, including `slow_down` and
    validation of the user code and verification URI
    (`oauth_device.go:85-263`).
  * Refresh: a 5-minute proactive skew, single-flight within the process,
    a reload from the store before refreshing so a rotation by another
    process is adopted, and bounded retries (`oauth_session.go:23,102-133,
    198-213,366-382`). A 401 forces one reactive refresh
    (`openai_chatgpt.go:88-97`, `grok.go:271-280`).
  * Revocation, `RevokeOAuthSession` (`oauth_revoke.go:25`).
  * Read-through reuse of the Codex and Grok CLI logins, which never
    refreshes the vendor's token family (`vendor_session.go:49-156`).
* **M3 — Only two providers accept a `TokenSource` (M).**
  `NewProviderWithSource` accepts `openai` and `grok` and refuses the rest
  (`provider.go:277-286`). `Token.Type` and `Token.Header`
  (`token.go:19-29`) are read by no non-test code: both providers
  hard-code `Authorization: Bearer` (`openai.go:191`, `grok.go:226`).
* **M4 — A rotated refresh token is lost when saving fails (R).**
  `reloadOrRefresh` returns the store's `Save` error after a successful
  refresh, and the caller adopts the new session only when there is no
  error (`oauth_session.go:121-125,376-381`). The issuer has already
  rotated the refresh token, so the old one in memory is spent, and the
  issuer's reuse detection can end the session.
* **M5 — `FileTokenStore` has no durability or Windows protection (R).**
  Writes are temp-then-rename with a `chmod 0600` **after** the rename,
  with no `fsync` and no inter-process lock (`tokenstore_file.go:88-117`).
  On Windows the chmod is a no-op and no DACL is set
  (`tokenstore_file_windows.go:5-7`). The imported
  [0010-MADR-windows-stdio-oauth-tokenstore-ci.md](0010-MADR-windows-stdio-oauth-tokenstore-ci.md)
  (proposed, not executed) already covers the DACL, reserved names and
  `fsync`. 0002-MADR §11 transfers it here.
* **M6 — OAuth claims are unverified and one check is missing (R).**
  `id_token` claims (account id, FedRAMP) are decoded without verifying the
  signature (`oauth_loopback.go:586-628`). The Grok `nonce` is generated and
  sent but never compared (`oauth_loopback.go:381-386`, its only use). Grok
  OIDC discovery falls back to hard-coded endpoints on any failure
  (`oauth_loopback.go:201-242`), even when the issuer was overridden by
  `GROK_OAUTH2_ISSUER` (`:161`).
* **M7 — The default HTTP client ignores proxies (R).** `defaultHTTPClient`
  builds an `http.Transport` with no `Proxy` field (`options.go:14-24`),
  so `HTTPS_PROXY` has no effect. A new one is built per provider
  (`options.go:205`) and another per refresh when none is set
  (`oauth_session.go:222-225`).
* **M8 — Listing spends metered calls (R).** For `openai`, `claude`,
  `gemini`, `grok` and `ollama`, listing sends a "Hello" generation to each
  of up to `MaxListedModels` candidates (`probe.go:13-60`). For the four
  API-key services that is billed usage, on a call a caller reads as a
  catalog lookup.
* **M9 — The wizard keeps two copies of a refresh token and has no logout
  (R).** `Result` carries the raw access and refresh tokens
  (`wizard/configure.go:23-40`) while the same session is saved to
  `Options.TokenStore` (`wizard/auth.go:323-341`). After the store's copy
  rotates, the consumer's copy is spent. `prepare-commit-msg` persists
  that second copy (`prepare-commit-msg` `internal/ui/setup.go:200-201`).
  Neither `RevokeOAuthSession` nor `TokenStore.Delete` has a caller.
* **M10 — Secret-bearing values do not redact themselves (R).** Redaction
  is applied at call sites: `RedactString` on OAuth error bodies
  (`oauth_session.go:77-78`) and `MaskSecret` in the wizard
  (`wizard/auth.go:142,173`). `Token`, `StaticToken`, `OAuthSession` and
  `wizard.Result` print their secrets under `%v` and `slog`.
* **M11 — Anthropic's key variable (R).** `ProviderEnvVars` maps `claude`
  to `CLAUDE_API_KEY` (`provider.go:165`). Anthropic's clients use
  `ANTHROPIC_API_KEY`, and so do this code's own live tests
  (`live_static_test.go:15`).
* **M12 — Tested and dependency-light (M).** There are 473 test functions
  in `llmprovider` (42 of them live-tagged, in 23 files), 70 in `wizard`,
  and 12 in the redaction copy. Coverage at `F`: `llmprovider` 89.2 %,
  `wizard` 82.5 %. The `llmprovider` package imports nothing outside the
  standard library except `mcplib/logging`, which 0002 replaces with
  `internal/redact`.

### Findings: `magic-cli-remote`

* **R1 — Its providers are coding-agent CLI drivers, not LLM clients (R).**
  `Provider` is `ID`, `Ready` and `Start(ctx, StartOptions) (Session,
  error)` (`mcr: internal/provider/provider.go:761-766`). A `Session`
  prompts, cancels and emits the app's own `event.Event`. Every
  implementation drives a vendor process:
  * `codex` spawns `codex app-server` (JSON-RPC over stdio or WebSocket);
  * `grok` is an ACP-over-stdio client, through `acpagent`;
  * `opencode` and `kilo` run the vendor's local HTTP+SSE server, through
    `httpagent`.

  No Go file under `mcr: internal/provider` references a vendor API host
  or path: the only matches for `api.openai.com`, `api.x.ai`,
  `api.anthropic.com`, `/v1/responses` and `chat/completions` are recorded
  Grok wire frames under `grok/testdata/`. No file in the repository
  mentions `llmprovider`.
* **R2 — Its LLM auth writes the vendor CLIs' own stores, and never
  refreshes (R).**
  * The principle is "write only into each vendor CLI's native store"
    (`mcr: docs/spec/0074-MADR-remote-provider-auth-from-phone.md:95-96`).
  * Codex keys go in through `codex login --with-api-key` with the secret
    on standard input (`mcr: internal/provider/codex/auth.go:22-54`);
    Grok keys are written into its `config.toml`
    (`mcr: internal/provider/grok/auth.go:23-58`); OpenCode and Kilo keys
    go through the engine's `PUT /auth/{id}`, else into its `auth.json`
    (`mcr: internal/provider/credstore/write.go:85-165`).
  * Device login runs the vendor CLI (`codex login --device-auth`,
    `grok login --device-auth`) or the engine's OAuth endpoint. Browser
    OAuth is refused (`mcr: internal/provider/httpagent/deviceauth.go:80-83`,
    `grok/auth.go:167-173`).
  * It rejects the Codex app-server's token-refresh request: "token refresh
    not supported — manage credentials outside mcremote"
    (`mcr: internal/provider/codex/session.go:1851`).
* **R3 — Its credential writes are more durable (R).** `writeFileAtomic`
  creates the temp file in the target directory, sets the mode **before**
  writing, writes, `fsync`s, closes, then renames
  (`mcr: internal/provider/credstore/write.go:42-75`). Secrets are capped
  at 8 KiB (`write.go:13-33`).
* **R4 — It never hands back or logs key material (R).** Status responses
  must not return key material (`mcr: internal/provider/auth.go:176-182`).
  The credential payload type redacts itself in `String` and `LogValue`
  (`mcr: internal/protocol/messages.go:859-875`).
* **R5 — Its device flow is a handle, not a callback (R).**
  `DeviceAuthHandle{Flow, Wait, Cancel}`
  (`mcr: internal/provider/auth.go:282-316`) lets a caller show the code on
  one surface and wait on another. The `mcplib` flow reports the code
  through a `NotifyDevice` callback and blocks
  (`llmprovider/oauth_loopback.go:32-42`).
* **R6 — Its transactional coordinator exists because it writes vendor
  stores (R).** `providerauth` isolates each device login in an empty
  home, publishes atomically under two locks, keeps current and previous
  generations, and recovers from a crash through a state machine
  (`mcr: internal/providerauth/transaction.go:64-70,282-514`,
  `recovery.go:1-17`). It serves Codex and Grok only
  (`mcr: internal/daemon/daemon.go:189-197`).
* **R7 — It is not extractable as a standard-library package (M, R).**
  The providers depend on `coder/acp-go-sdk` (forked), `coder/websocket`,
  `google/uuid` and, in `providerauth`, `fsnotify`. Their transitive
  production code is 23,000–31,000 lines each, much of it phone-UI shaped
  (`event.Event`, `picker.Catalog`, 52 optional interfaces in the root
  package). ~~The module requires Go 1.27.1.~~ *(Struck 2026-09-29: 1.27.1
  is the fleet's standard, not a cost; see 0002-MADR's fifth amendment.)*
  `credstore` (562 lines) and
  `providerauth/classify.go` (143 lines) import only the standard library.
* **R8 — Tested heavily, against pinned CLI versions (M).** 1,568 test
  functions, 121 of them live-tagged, with recorded wire fixtures per CLI
  version (Codex 0.155.1, Grok 1.0.5, Kilo 7.4.23).

### What the comparison shows

The two codebases solve different problems. `mcplib` calls LLM APIs and
owns the OAuth tokens it obtains. `magic-cli-remote` supervises agent CLIs
and leaves every token to the CLI that owns it. Only `mcplib` has what the
SDK's purpose names: provider requests, subscription OAuth, refresh and a
token store. `magic-cli-remote` handles credential **material** more
carefully: durable writes, secrets that cannot leak through formatting,
and a device flow that fits non-terminal callers. Those practices transfer;
its provider model does not.

## Decision Drivers

* **Purpose.** The SDK gives callers direct access to LLM providers and
  authenticates them, including subscription OAuth.
* **Dependencies.** The standard library, plus `golang.org/x/term` in
  `wizard` (AGENTS.md). Any other module needs a MADR.
* **Credential safety.** No rotated token is ever discarded; no secret
  reaches a log or a formatted string; stored secrets survive a crash.
* **One credential path.** Every provider authenticates the same way, so
  the 0015 contract's `WithTokenSource` means the same thing everywhere.
* **Functional parity** with `mcplib` `v1.6.0` (0015-MADR D1). A behaviour
  may change only under a record.
* **Keep measured knowledge.** Wire and OAuth behaviour that live tests
  established is kept, not rewritten.

## Considered Options

* A. `mcplib` `llmprovider` is the base; adopt `magic-cli-remote`'s
  credential-hygiene practices.
* B. `magic-cli-remote`'s agent-CLI provider model is the base.
* C. Both, as two provider families in this module: API providers from
  `mcplib` and agent-CLI drivers from `magic-cli-remote`.
* D. Neither; write providers and auth from the vendors' documentation.

## Decision Outcome

Chosen option: "A. `mcplib` `llmprovider` is the base; adopt
`magic-cli-remote`'s credential-hygiene practices", because it is the only
option that meets the SDK's purpose inside the dependency rule, and it
fixes the credential defects M3–M10 with practices that `magic-cli-remote`
has already proven.

B has no LLM request path, no OAuth of its own and no refresh (R1, R2),
and needs four non-standard modules (R7). C brings those modules and a
second, unrelated contract into this module. D discards M2's measured OAuth
behaviour and M12's tests.

### D1. The base

* Provider support and authentication are built from the imported
  `llmprovider` and `wizard`, restructured by 0015-MADR.
* From `magic-cli-remote`, the SDK adopts D3, D5 and D6 below as
  practices, and re-implements them. It copies no file and adds no
  dependency on that module.
* `magic-cli-remote`'s provider model, `providerauth` coordinator and
  writes into vendor CLI stores are **not** adopted (D10).

### D2. Every provider authenticates through a `TokenSource`

* Every provider accepts a `TokenSource`, through 0015-MADR D5's
  `WithTokenSource`. `WithAPIKey(k)` is shorthand for a `StaticToken`.
* **The provider applies the token as `Token` describes it.**
  * An empty `Header` means the header the service requires: for example,
    `Authorization` with `Bearer` for OpenAI, `x-api-key` for Claude, and,
    for OpenCode, the header of the request's route
    (`llmprovider/opencode.go:390-399`). That is exactly what each provider
    sends today, so the wire is unchanged (0015-MADR D12, G-wire).
  * A non-empty `Header` overrides it, with `Type` as the scheme prefix
    (none when empty), for callers behind a gateway that expects another
    header.
* An OAuth or vendor-CLI source given to a provider whose service does not
  accept it is refused by `New`, never sent.

### D3. Token-store writes are durable

`FileTokenStore` writes the way `magic-cli-remote`'s `credstore` does (R3):

* a temp file in the target directory, created with mode `0600` before any
  byte is written;
* write, `fsync`, close, rename; then `fsync` of the directory where the
  platform supports it;
* a size cap on read.

The Windows DACL and reserved-name checks (M5) are executed under the
transferred `0010` record, not here. ~~A cross-process lock is not added:
the reload-before-refresh (M2) and D4 cover the known failure. Revisit if a
lost rotation is observed across processes.~~ *Replaced 2026-09-30 by the
amendment's A2 (accepted): `FileTokenStore` locks a refresh across
processes.*

### D4. A rotated token is never discarded

* The session adopts the refreshed tokens in memory **before** persisting
  them.
* If `Save` fails, `Token` still returns the fresh token. The failure is
  logged through the logger passed with `WithLogger` (0015-MADR D9), and
  `Save` is retried on the next `Token` call until it succeeds.
* A test asserts that, after a failed `Save`, the next refresh sends the
  rotated refresh token and not the spent one.

### D5. Secret-bearing values redact themselves

* `Token`, `StaticToken`, the OAuth session and `wizard.Result` implement
  `String`, `GoString` and `slog.LogValuer`, and return a masked form.
  Code that needs the secret reads the field; formatting never shows it.
* Descriptor and status values never carry key material (R4).
* A test formats each type with `%v`, `%+v`, `%#v` and `slog`, and fails if
  a planted secret appears.
* *Amended 2026-10-01 (A9):* `wizard.Result`'s JSON is not redacted.

### D6. Device login returns a handle

* `auth` starts a device login and returns a handle with the user code,
  the verification URI, the expiry, `Wait(ctx)` for the session, and
  `Cancel` (R5).
* The browser flow keeps its `OpenURL` and `InputCode` callbacks.
* `wizard` uses the handle; its prompts are unchanged.

### D7. OAuth checks

* The Grok `nonce` is compared with the `id_token`'s `nonce` claim, and a
  mismatch fails the login.
* ~~`id_token` claims stay **unverified hints**: they are used only to
  address the issuer that issued them. The `auth` package doc says so.
  Signature verification is not added (revisit if a claim ever gates a
  local decision).~~
  *Replaced by the owner's decision of 2026-09-29 ("Owner's decisions" below):* the `id_token` signature is verified against the issuer's
  published keys before any claim is used, and any failure fails the login.
* OIDC discovery falls back to the built-in endpoints only for the
  built-in issuer. An issuer passed by the caller whose discovery fails is
  an error.
* The client ids borrowed from the Codex and Grok CLIs remain the default,
  because subscription OAuth works only with them. They stay overridable,
  and the risk is recorded in Consequences.

### D8. Transport

* The default HTTP client uses `http.ProxyFromEnvironment` and the
  connection limits of today's `defaultHTTPClient`.
* One default client per provider instance serves requests, listing and
  refresh.
* **Interaction with 0015-MADR D9.** `net/http` reading the proxy variables
  is the standard library's behaviour, not a library read. 0015's ambient
  check must allow exactly this call. 0015-MADR takes that as an amendment
  when both records are accepted.

### D9. Listing does not spend usage by default

*Replaced 2026-09-30 by amendment A5: probes are on by default again.*

* ~~Listing returns the service's model list without generation probes.~~
* ~~Probing is an explicit option on the listing call, documented as
  sending one billed request per model.~~
* ~~This is a recorded behaviour change under 0015-MADR D1: the capability
  stays, its default changes.~~

### D10. What the SDK does not do

* **It does not drive agent CLIs.** Supervising `codex app-server`, ACP
  agents or local agent servers belongs in a separate module with its own
  MADR, because it needs modules this one excludes (R7).
* **It does not write a vendor CLI's credential store.** Vendor-CLI
  credentials stay read-through (M2), so no coordinator like R6 is needed.
* **It adds no Anthropic or Gemini subscription OAuth.** That stays out of
  scope, as the imported
  [0006-MADR-subscription-auth-for-llm-providers.md](0006-MADR-subscription-auth-for-llm-providers.md)
  decided.

### D11. The wizard keeps one copy of a refresh token, and can log out

* When a session is saved to `Options.TokenStore`, `Result` carries the
  provider, the method and the non-secret fields, but no access or refresh
  token. The store is the single owner.
* `wizard` offers "log out" for a stored session: `RevokeOAuthSession`,
  then `TokenStore.Delete`. A revocation failure is reported and the local
  copy is still deleted.
* *Amended 2026-10-01 (A7, A8, A10):* how `wizard` meets this.

### D12. Anthropic's key variable

~~The opt-in environment helper of 0015-MADR D9 reads `ANTHROPIC_API_KEY`
for `claude`, then `CLAUDE_API_KEY`. The first is the vendor's name; the
second keeps today's behaviour.~~
*Replaced by the owner's decision of 2026-09-29 ("Owner's decisions" below):* `ANTHROPIC_API_KEY` only. `CLAUDE_API_KEY` is no longer read.

### Consequences

* Good, because the SDK starts from the only code that calls LLM APIs and
  performs subscription OAuth, with its tests and history.
* Good, because every provider takes the same credential type, so a caller
  can supply keys from a vault or a rotating source to any provider.
* Good, because a failed save, a crash mid-write or a stray `%v` no longer
  loses or leaks a token.
* Good, because device login fits non-terminal callers such as a phone UI
  or an ACP client.
* Good, because listing no longer bills a caller who did not ask for it.
* Neutral, because `magic-cli-remote` is unaffected. It does not import
  `llmprovider`, and nothing here requires it to.
* Bad, because D9 and D11 change behaviour a consumer can see.
  `prepare-commit-msg` persists the refresh token from `Result`
  (M9) and must read it from its `TokenStore` instead. Its companion
  record under 0002-MADR §13 must list this.
* Bad, because the borrowed client ids (D7) remain a policy and breakage
  risk: a vendor can revoke or restrict them without notice.
* Bad, because there is still no cross-process lock. ~~and no signature
  verification of `id_token`~~ *(struck 2026-09-29: verified, D7)*
* Bad, because agent-CLI support, which a consumer such as `pi-go` may
  want, needs a further module and record.

### Confirmation

* Each of D2–D9, D11 and D12 has a test that is first seen to fail on the
  pre-change code in a scratch copy, as the PLAN lists.
* G-wire (0015-MADR D12) shows no request difference from D2 or D8. D9's
  listing difference is the only one, and it is listed.
* `make dep-check` (0015-PLAN S12) shows no new module.

*Annotated 2026-10-02, at 0016-PLAN T6: all three are met.* D9's listing
difference no longer exists: A5 restored the probes, so the goldens match
their content before 0015-PLAN S5. The PLAN's T6 record holds the evidence.

## Pros and Cons of the Options

### A. `mcplib` base, `magic-cli-remote` practices

* Good, because it meets the purpose within the dependency rule.
* Good, because it keeps M2's measured OAuth behaviour and M12's tests.
* Good, because the adopted practices are small and already proven in
  `magic-cli-remote`.
* Bad, because the adopted practices are re-implemented, not shared, so a
  later fix in one repository does not reach the other.

### B. `magic-cli-remote`'s agent-CLI model

* Good, because it is the more heavily tested code (R8), and its credential
  handling is the more careful (R3–R6).
* Bad, because it makes no LLM API request and performs no OAuth or
  refresh itself (R1, R2), so the SDK's purpose would be unmet.
* Bad, because it needs `acp-go-sdk`, `websocket`, `uuid` and `fsnotify`
  (R7). ~~and Go 1.27.1~~ *(struck 2026-09-29, as in R7)*
* Bad, because its types are shaped by a phone UI, not by a library
  caller.

### C. Both, as two provider families

* Good, because callers get API providers and agent CLIs from one module.
* Bad, because it pulls R7's modules into every importer.
* Bad, because the two contracts share nothing (a request/response call
  against a supervised interactive session), so "one consistent API"
  (0015-MADR) would be false.
* Bad, because it ties the SDK's releases to the CLIs' version churn (R8).

### D. Neither; rewrite from the vendors' documentation

* Good, because it gives the cleanest start.
* Bad, because much of the working OAuth behaviour (ports, CORS, state
  suffixes, device-code validation, refresh reuse rules) comes from the
  vendors' reference clients and live tests, not from their documentation.
* Bad, because parity with `mcplib` could not be proven against the code
  that has it.

## More Information

* **Implementation:**
  [0016-PLAN-provider-auth-and-support-baseline.md](0016-PLAN-provider-auth-and-support-baseline.md).
  It runs inside 0015-PLAN's phases and cannot start before 0015-MADR is
  accepted.
* **Related records here:**
  * [0002-MADR-migrate-llmprovider-from-mcplib.md](0002-MADR-migrate-llmprovider-from-mcplib.md):
    the migration, §6 environment names, §11 transferred work, §13
    consumer companions.
  * [0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md):
    the API that D2, D6 and D8 are expressed in.
  * [0010-MADR-windows-stdio-oauth-tokenstore-ci.md](0010-MADR-windows-stdio-oauth-tokenstore-ci.md):
    the Windows token-store work D3 leaves to it.
* **Cross-repository, cited by name:**
  * `mcplib` `docs/decisions/0015-MADR-transfer-llmprovider-to-go-llmprovider-sdk.md`;
  * `magic-cli-remote` `docs/spec/0074-MADR-remote-provider-auth-from-phone.md`
    (the native-store principle and the device-flow handle);
  * `magic-cli-remote` `docs/ops-credential-recovery.md` (the coordinator
    D10 declines).
* **Revisit** if a lost refresh-token rotation is seen across processes
  (D3), if an `id_token` claim comes to gate a local decision (D7), or when
  a consumer asks for agent-CLI support (D10).

## Owner's decisions (2026-09-29)

Status: **accepted**. The owner was asked about each open choice and
answered:

| Decision | Answer |
|---|---|
| D9, listing probes | Off by default, as proposed |
| D11, tokens in `Result` | The store only, plus logout, as proposed |
| D7, `id_token` | **Verify signatures** (changed), and fail the login on any failure, including keys that cannot be fetched |
| D12, the Claude key variable | **`ANTHROPIC_API_KEY` only** (changed) |
| D4, a failed save after refresh | Keep the rotation, log, retry, as proposed |
| D3, cross-process lock | None for now, as proposed |
| D10, agent CLIs | Out of scope, revisit on demand, as proposed |

The other decisions (D1, D2, D5, D6, D8) stand as proposed.

### D7 as decided

* Before any claim is used, `auth` verifies the `id_token`:
  * the keys come from the `jwks_uri` of the issuer's discovery document,
    and are cached;
  * the algorithm must be one the SDK supports, and `none` and HMAC are
    refused;
  * `iss`, `aud` (the client id) and `exp` are checked, and for Grok the
    `nonce`.
* A bad signature, a key that cannot be found, or keys that cannot be
  fetched fails the login, and nothing is saved.
* The nonce check and fail-closed discovery for a caller's issuer stand, as
  D7 first decided.
* **The two built-in issuers, probed read-only on 2026-09-30T03:58Z (UTC).**
  Both are verifiable with the standard library alone (`crypto/rsa`,
  `crypto/ecdsa`), so the dependency rule is unchanged.

  | Issuer | `jwks_uri` | Advertised | Keys published |
  |---|---|---|---|
  | `https://auth.openai.com` | `https://auth.openai.com/.well-known/jwks.json` | RS256 | 4 RSA, RS256 |
  | `https://auth.x.ai` | `https://auth.x.ai/.well-known/jwks.json` | ES256 | 2 EC P-256, ES256 |

  A live-tagged test pins this, so a change of algorithm on either side
  fails loudly.

### D12 as decided

`ANTHROPIC_API_KEY` is the only variable read for `claude`: by the opt-in
environment helper (0015-MADR D9), and as the `claude` descriptor's variable
that the wizard's `AllowEnv` reads.

### Consequences of the decisions

* Good, because no claim from an `id_token` is trusted unverified.
* Bad, because a subscription login now depends on fetching the issuer's
  keys. An outage of the key endpoint blocks new logins, though refresh of
  an existing session does not need it.
* Bad, because a setup that sets only `CLAUDE_API_KEY` stops finding its
  key, and must rename it. `prepare-commit-msg`'s companion record must say
  so.

## Amendment 2026-09-30: findings of the reference-client survey (proposed)

Status: **accepted** 2026-09-30. The owner answered on 2026-09-30: "1. accept d1, proceed with d1-d5. 2. both as recommended. 3. add it." A1 and A3 are accepted as
written, and A2 as option (b) with A1. Evidence:
[0017-REPORT-reference-client-auth-survey.md](../reports/0017-REPORT-reference-client-auth-survey.md).

### A1. Re-read the store after a rejected refresh (extends D4)

* **Gap.** `reloadOrRefresh` re-reads the store *before* refreshing, and
  never after a refresh fails (`llmprovider/oauth_session.go`).
* **The race.** Two processes that load the same refresh token both
  refresh. The second is told `invalid_grant` or `refresh_token_reused`,
  and returns a permanent error, although the first has already saved a
  rotated session in the store.
* **What Grok does** (0017-REPORT G3). It re-reads its file on a rejected
  refresh token. When a sibling has rotated the token, it demotes the
  failure to transient and uses the sibling's token
  (`grok-build: crates/codegen/xai-grok-login/src/manager.rs:1344-1373`).
* **Proposed.** On a permanent refresh failure, re-read the store once:
  * if it holds a different refresh token, adopt that session: use its
    access token if current, else refresh with it once;
  * otherwise return the permanent error.
* **Test:** two sessions share one store, and the second's refresh is
  rejected after the first has saved. It must adopt, not fail.

### A2. Reopen D3: a cross-process refresh lock

* **Why D3 is worth revisiting.** D3 decided no lock, "revisit if a lost
  rotation is observed across processes". Two facts bear on it:
  * **The cost of a race is not one lost token.** This module's own source
    records that both vendors revoke the **whole refresh-token family** when
    one refresh token is used twice (`llmprovider/vendor_session.go:21-24`,
    citing `grok-build` `xai-grok-login/src/oidc/refresh.rs` and `codex`
    `login/src/auth/manager.rs:1657-1690`). Grok's source, at `036a5d8`, says
    that re-sending a rotated refresh token "trips the IdP's reuse detection
    and revokes a successor a sibling may hold". Its only allowance is a
    60-second client-side grace (`ROTATION_GRACE_MS`). A1 cannot help once
    the issuer has revoked the family.
  * **Most of the surveyed clients lock** (0017-REPORT G3, P5, A2):
    * Grok takes a `flock` on `auth.json.lock`, re-reads after acquiring,
      and re-checks the inode;
    * pi uses `proper-lockfile`, stale after 30 s;
    * Claude Code's changelog describes a cross-process refresh lock.

    Codex alone does not (C2).
* **Options:**
  * (a) keep D3, with A1: the window stays open;
  * (b) `FileTokenStore` takes a lock file beside the store for a refresh.
* **How (b) would work:**
  * exclusive create (`O_CREATE|O_EXCL`), holding the pid and the time;
  * stale after 30 s;
  * a jittered retry to a deadline;
  * a re-read after acquiring;
  * never refreshing unlocked, so a timeout is a retryable error, as Grok
    and Claude Code do.

  It is standard-library only. Other `TokenStore` implementations may offer
  their own lock through an optional interface.
* **Recommended: (b) with A1.**

### A3. `id_token` checking: implementation details for D7 as decided

These refine D7 as the owner decided it, from Grok's implementation
(0017-REPORT G2):

* **A `kid` is required.**
* **Algorithms:** the algorithm must be one the standard library verifies
  (RS256/384/512, PS256/384/512, ES256/384, EdDSA). It must also be in the
  discovery document's `id_token_signing_alg_values_supported` when that is
  present.
* **An unknown `kid`** refetches the keys once (already T2 step 5).
* **Scope of the check:** it applies to every login that returns an
  `id_token`, the device flows included. Grok's own device flow does not
  verify; this module is deliberately stricter.
* **A missing `id_token`** fails the login when `openid` was requested.
  Grok's team logins legitimately return none, but this module never
  requests team scopes. If it ever does, that takes an amendment.
* Codex verifies no `id_token` signature (0017-REPORT C1). D7 is therefore
  stricter than the vendor's own client; that is the owner's choice, and it
  stands.

### Notes, no change proposed

* **D10.** The survey found the Anthropic and Copilot subscription logins
  only in forms that present as the vendor's own client (0017-REPORT P4).
  That supports D10 as decided.
* **D12.** Claude Code and pi also read `ANTHROPIC_AUTH_TOKEN`, a bearer for
  gateways. D12 stands. A caller with such a token passes it through
  `WithTokenSource`, with D2's `Header` override.

### A4. `MarshalJSON` redaction (extends D5; the owner's decision, 2026-09-30)

* **Found in T2 step 3.** slog's JSON handler encodes a struct *holding* a
  `Token` with `encoding/json`, which never calls the nested `LogValue`, so
  the secret showed. `json.Marshal` of any of the types did the same.
* **Decided.** The owner answered "add the MarshalJSON redaction". `Token`,
  `StaticToken` and `*OAuthSession` also implement `MarshalJSON`, with the
  same redacted fields as their other forms.
* **Cost.** The encoding does not round-trip: code that needs a secret reads
  the field. Nothing in this module JSON-encodes these types to store or
  send them. `FileTokenStore` writes its own record type.

### A5. Listing probes are on by default, with a caller option and an opt-in variable (the owner's decision, 2026-09-30)

The owner wrote on 2026-09-30: "Actually re-enable probes by default. Add an env var LLM_PROVIDER_PROBES=true/false defaulting to true. Add args to the api that can be configured by callers to enable or disable." Asked about two conflicts, the owner chose "Opt-in helper only" and "LLMPROVIDER_PROBES".

* **Default.** Listing probes, as `mcplib` `v1.6.0` did: for OpenAI (API
  key), Claude, Gemini, Grok and Ollama, `DiscoverModels` sends one short
  generation to each candidate, up to `MaxListedModels`, and keeps those
  that answer. Every probe is billed; the default accepts that cost, which
  finding M8 described.
* **Caller option.** `WithModelProbes(bool)` enables or disables the probe
  for one provider. A ChatGPT session, Kilo, OpenCode, Hugging Face and
  Together never probe (0012-MADR §1.6), whatever the option says.
* **Variable.** `LLMPROVIDER_PROBES` (`true` or `false`, as
  `strconv.ParseBool` reads them) is read only by the opt-in helper
  `ModelProbesFromEnv()`. It returns the matching `WithModelProbes`, or an
  option that does nothing when the variable is unset or not a boolean. So
  0015-MADR D9 ("library code reads no environment variable") holds.
  * The name follows the module's `LLMPROVIDER_` prefix (0002-MADR §6). The
    owner first wrote `LLM_PROVIDER_PROBES`.
  * Options apply in order: a caller wanting an explicit option to win
    passes the helper first.
* **Effect on the record.** D9 as first decided is struck. The G-wire
  listing goldens return to their `P7` form, with the probe requests.

### A6. A token names only the header its caller set (clarifies D2; the owner's decision, 2026-09-30)

* **Found** in 0015-PLAN S7, before the gemini commit. The sources
  contradicted D2:
  * `StaticToken` filled in `Header: "Authorization"` when its caller set
    none (`token.go:103-111`). `OAuthSession` and `VendorCLISession`
    returned `Authorization` too (`oauth_session.go:225-227`, `:360-362`;
    `vendor_session.go:72`).
  * Every source reported `Type: bearer`, a kind of token rather than a
    scheme.

  Applied as written, D2 would have sent `Authorization: bearer <key>` to
  Claude and Gemini in place of their own headers. An override could never
  carry a bare key: `CommandToken{Header: "x-api-key"}` would have sent
  `x-api-key: bearer <key>`.
* **Decided.** Asked on 2026-09-30, the owner chose "sources say only
  what's set":
  * **Header.** A source returns only the `Header` its caller set.
    `StaticToken` and `CommandToken` return their `Header` field, empty by
    default. `OAuthSession` and `VendorCLISession` return none.
  * **Type.** `StaticToken` and `CommandToken` report `TokenAPIKey`.
    `OAuthSession` and `VendorCLISession` report `TokenBearer`.
  * **Prefix.** With an override, `TokenBearer` is sent as `Bearer <value>`;
    `TokenAPIKey`, or no `Type`, is sent as the bare value. With no
    `Header`, the provider sends its own header and scheme, and `Type` is
    not read.
* **Cost:**
  * A gateway that wants `Authorization: Bearer <key>` in place of Claude's
    `x-api-key` needs a `TokenSource` that reports `TokenBearer`.
    `StaticToken{Header: "Authorization"}` sends the bare key.
  * The `Header` and `Type` a source reports change. A caller that reads
    them sees the change; nothing in this module read either.
* **Effect.** D2's wire is unchanged: G-wire shows no difference. R16 in
  `docs/guides/api-standards.md` states the prefix rule.

## Amendment 2026-10-01: D11 and D5 in `wizard` (the owner's decisions)

Status: **accepted** 2026-10-01. Asked during 0015-PLAN S8, commit 6, which
carries 0016-PLAN T4. Each point is one that D11 or D5 left open.

### A7. A stored session leaves no token in `Result` (D11)

* **Fact found.**
  * Every sign-in that saves to `Options.TokenStore` (browser, device,
    pasted ChatGPT token) set `Result.AccessToken` and `RefreshToken`
    (`wizard/configure.go:137-145`).
  * Kilo's device login saved its token to the store and also returned it
    as `Result.APIKey` (`wizard/auth.go:176-180`).
* **Decided.** The owner chose "store only, Kind oauth" for Kilo.
  * Every stored session gives `Kind` `CredOAuth` and no token, Kilo's
    included. The consumer loads the session from the store and passes it
    as the provider's `TokenSource`. Kilo's requests are unchanged: an
    `OAuthSession` with no `Header` and a static key both send
    `Authorization: Bearer <token>`.
  * The OAuth methods are offered only with a store, so no path sets
    `Result.AccessToken` or `RefreshToken` any more. Both fields are
    removed, rather than left as fields that are always empty.
  * The non-secret fields stay: `TokenExpiry`, `Issuer`, `ClientID`,
    `AccountID`, `FedRAMP`.
* **Rejected.** Keeping Kilo's token as `APIKey`: two copies, which D11
  exists to prevent.

### A8. Logout is a function (D11)

* **Decided.** The owner chose "a `Logout` function".
  `wizard.Logout(ctx, p, o, id)`:
  * confirms through the `Prompter`;
  * loads the stored session;
  * revokes it with `RevokeOAuthSession`, reporting a failure as a warning;
  * then deletes it from `Options.TokenStore`.
  * `ConfigureLLM`'s prompts are unchanged; a consumer wires `Logout` to
    its own command.
* **Following from `RevokeOAuthSession`'s scope.** That function revokes
  OpenAI and Grok sessions only. For another provider's session (Kilo's),
  `Logout` says the service offers no revocation, and deletes.
* **Rejected.**
  * A third choice inside `ConfigureLLM`: it changes the flow's prompts.
  * Both: the cost of each.

### A9. `Result`'s JSON is the consumer's (D5)

* **Fact found.** 0016-PLAN T4.3 says `Result` "joins T2 step 3's table",
  which also checks `json.Marshal`. But `Result` is the consumer's data to
  persist (its doc says so), and it holds `APIKey`.
* **Decided.** The owner chose "fmt and slog only".
  * `String`, `GoString` and `LogValue` mask `APIKey`.
  * `json.Marshal` keeps it, and a test pins that it does.
* **Rejected.** Redacting the JSON as well, as A4 does for `OAuthSession`.
  A consumer persisting `Result` with `encoding/json` would lose its key.

### A10. Keeping a session reads the store (D11)

* **Decided.** The owner chose "store only". When `Options.Existing` names
  the provider with `Kind` `CredOAuth`, the wizard loads that provider's
  session from `Options.TokenStore` and offers to keep it. If the store has
  none, the user signs in again.
* **Rejected.** Also rebuilding the session from an old `Result`'s tokens
  and saving it, to migrate one. That path would have to stay until a
  later removal.

### Effect

D11 and D5 stand. These points say how `wizard` meets them.
