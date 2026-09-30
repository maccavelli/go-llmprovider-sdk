---
status: accepted
date: 2026-09-30
decision-makers: go-llmprovider-sdk maintainers
consulted: the sources of pi, grok-build, codex, kilocode and opencode; the documentation of antigravity-cli and claude-code
informed: every consumer of github.com/maccavelli/go-llmprovider-sdk
---
# Add Together AI, Kilo Device Login and Command-Sourced Keys, from the Reference-Client Survey

## Context and Problem Statement

On 2026-09-30 the owner asked for:

* an assessment of how six coding agents authenticate, by browser/OAuth and
  by API key, and of whether this module's `llmprovider` can be improved from
  those facts;
* Together AI to be added as a provider.

The evidence is recorded in
[0017-REPORT-reference-client-auth-survey.md](../reports/0017-REPORT-reference-client-auth-survey.md).
Its comparison table shows that most of what the reference clients do,
`llmprovider` already does. That is the result of the 0011 audit and the
0012 work. The findings fall into three groups:

* **Settled already.** Two apparent differences are deliberate recorded
  divergences:
  * the Grok session host ([0006-MADR-subscription-auth-for-llm-providers.md](0006-MADR-subscription-auth-for-llm-providers.md),
    decision 2);
  * the Kilo chat path ([0012-PLAN-gateway-conventions.md](0012-PLAN-gateway-conventions.md)).
* **Current work.** Four findings land in phases of
  [0015-PLAN-canonical-sdk-api-and-module-layout.md](0015-PLAN-canonical-sdk-api-and-module-layout.md)
  and
  [0016-PLAN-provider-auth-and-support-baseline.md](0016-PLAN-provider-auth-and-support-baseline.md)
  that have not run yet. They are recorded as amendments to those records,
  not here:
  * re-reading the store after a rejected refresh, and the cross-process
    lock, amend
    [0016-MADR-provider-auth-and-support-baseline.md](0016-MADR-provider-auth-and-support-baseline.md);
  * the `id_token` details also amend 0016-MADR;
  * a context-overflow error kind amends
    [0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md);
  * the usage facts amend 0015-PLAN S9.
* **New.** This record decides these.

The question: **how is Together AI added, and which of the survey's new
capabilities does the SDK take on?**

## Decision Drivers

* **The owner's request.** Together AI is added.
* **Identity.** A request names this module, never a reference client
  ([0012-MADR-conform-providers-to-reference-clients.md](0012-MADR-conform-providers-to-reference-clients.md)
  §1.4).
* **Dependencies.** The standard library only in `llmprovider`
  ([0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md)
  D2).
* **One migration.** Work done now in `llmprovider` is ported by 0015-PLAN
  S7 anyway, so new code must be cheap to port.
* **Measured, not assumed.** A wire shape taken from another client's
  source, or from documentation, is confirmed by a live-tagged test before
  a release depends on it.

## Considered Options

For Together, the one choice with real alternatives:

* A. A dedicated `together` provider now, in `llmprovider`, on the shared
  Chat Completions code, ported by 0015-PLAN S7 like the others.
* B. A dedicated provider after S7, written directly as
  `llmprovider/providers/together` on the new contract.
* C. No dedicated provider: a generic Chat Completions provider configured
  by base URL and key, which would also serve Together.

## Decision Outcome

Chosen option: "A. A dedicated `together` provider now", because the owner
asked for Together now. The code is small: Hugging Face's provider, the
closest template, is 194 lines over the shared primitive. Porting it in S7
costs one more package among nine. B makes Together wait for S7. C loses
what the dedicated providers give: a curated listing, a descriptor for
`wizard`, and typed quirks. C remains a candidate for its own record.

### D1. Together AI

* **Identity.**
  * Provider id `together`.
  * Base URL `https://api.together.ai/v1`.
  * Key from `WithAPIKey`, sent as `Authorization: Bearer`.
  * The environment variable named in the catalog is `TOGETHER_API_KEY`
    (0017-REPORT, Together section; models.dev key `togetherai`).
* **Wire.** `POST {base}/chat/completions` through the shared Chat
  Completions code, with `max_tokens`.
  * A forced tool uses the object form of `tool_choice`, as the other
    Chat Completions providers do.
  * **The thinking path** sends `reasoning: {"enabled": true}`, and
    `reasoning_effort` only when the caller set an effort.
  * **The plain path** sends neither, so the model's default applies.

  pi sends `reasoning.enabled` to every toggleable model, and
  `reasoning_effort` only to models known to accept it (0017-REPORT P2). A
  live-tagged test confirms both shapes on one toggleable model and on
  `openai/gpt-oss-120b` before release.
* **Responses.** The existing decoder already reads `reasoning` and
  `reasoning_content`. `finish_reason: "eos"` is a normal stop; only
  `length` is truncation.
* **Listing.** `GET {base}/models` returns a bare array. Entries with
  `type: "chat"` are kept, then curated by the open-catalog ranking with
  the models.dev key `togetherai`.
  * There is **no generation probe**: every call is metered
    ([0016-MADR-provider-auth-and-support-baseline.md](0016-MADR-provider-auth-and-support-baseline.md) D9).
  * The static fallback is a short list of tool-capable chat models, taken
    from models.dev when the provider is written.
* **Capabilities.** Tools, forced tool choice and reasoning are
  `BestEffort`, because support is per model on Together. There is no
  continuation: Chat Completions is stateless.
* **Descriptor.** An API-key method only. Together has no OAuth in any
  surveyed client.
* **Sequence.** It lands in `llmprovider` now, with its own G-wire case (new
  golden files, not a regeneration of existing ones). 0015-PLAN S7 ports it
  after `ollama`.

### D2. Kilo device login

* `auth` gains a Kilo device login:
  * `POST {base}/api/device-auth/codes` starts it;
  * `GET {base}/api/device-auth/codes/{code}` polls every 3 s until 200
    (approved, with `token`), 403 (denied) or 410 (expired)
    (0017-REPORT K1).
* It returns the handle of 0016-MADR D6.
* **The token has no refresh and no known expiry.** It is stored in the
  `TokenStore` and applied like an API key. It is never refreshed. A 401 is
  `ErrAuthFailure`, telling the caller to log in again.
* After login, `/api/profile` offers the organizations, for Kilo's
  organization option.
* The requests name this module; nothing presents as Kilo's own client.

### D3. Keys from a command

* `auth` gains a `TokenSource` that runs a caller-supplied command and uses
  its standard output as the token. This is the pattern of Claude Code's
  `apiKeyHelper`, pi's `!command` keys and Grok's
  `GROK_AUTH_PROVIDER_COMMAND` (0017-REPORT A2, P6, G section).
* **How it runs:**
  * The command is an argv. It never runs through a shell.
  * Each run has a timeout (default 10 s), and its output is capped
    (8 KiB, as 0016-MADR R3).
  * The result is cached for a TTL (default 5 minutes, as `apiKeyHelper`).
  * A 401 from the service runs the command once more.
* **Safety:**
  * The value redacts itself (0016-MADR D5).
  * A failure's message names the command, and never its output.
* **Ambient state.** Library code never finds a command on its own: the
  caller names it. So 0015-MADR D9 holds.

### D4. OpenAI sign-in without Codex's client id: probe, then decide

* pi's source shows a public-client flow with **dynamic client
  registration**, whose token is used on `api.openai.com/v1`: scope
  `chatgpt.tokens.use.direct`, resource `https://api.openai.com/v1`
  (0017-REPORT P3).
* If it works for any third-party agent, it removes 0016-MADR's recorded
  risk of borrowing Codex's client id.
* **The evidence is one day old and comes from one client.** No OpenAI
  document was read.
* **Decided:** a live-tagged probe, run only on the owner's request, records:
  * whether registration succeeds under this module's name;
  * which scopes are granted;
  * which Responses fields the token is refused.

  A later amendment to this record decides whether it becomes an OpenAI
  auth method. Nothing ships before that.

### D5. Session-affinity headers on Kilo: not now

* Kilo's client adds `X-Session-Id` and `x-session-affinity` beside the
  task id (0017-REPORT K2).
* Sending them would change every Kilo G-wire golden, and no benefit is
  measured.
* Revisit when a measurement shows cache or routing gains.

### D6. Not adopted, with the reason recorded

* **Anthropic Pro/Max and GitHub Copilot subscription logins.** Both
  surveyed implementations present as the vendor's own client:
  * pi sends `claude-cli` as its user agent and a required Claude Code
    system prompt;
  * pi sends Copilot's editor headers.

  That breaks 0012-MADR §1.4. OpenCode dropped its Anthropic flow.
  0016-MADR D10 stands.
* **Grok's chat proxy and `X-XAI-Token-Auth`.** 0006-MADR decision 2 stands.
  The marker identifies the official CLI, and the proxy refused a
  third-party session with 426.
* **Kilo's `/api/openrouter` path.** The 0011/0012 choice of `/api/gateway`
  stands.
* **OS keyring storage** (Codex, `agy`, Claude Code). It needs a module
  outside the dependency rule, or running platform tools. Not now.
* **OpenRouter, Kimi and Meta logins.** They belong to providers this
  module does not have. Each is a new provider, with its own record.

### Consequences

* Good, because Together is available without waiting for the
  restructure, on code that S7 already knows how to port.
* Good, because D2 gives Kilo users a login that needs no copied key, and
  D3 covers short-lived and vaulted keys in any provider without a new
  dependency.
* Good, because D4 tests the one finding that could remove a standing
  risk, before any code depends on it.
* Neutral, because D5 and D6 change nothing, but now carry their evidence.
* Bad, because S7 has one more provider to port. The G-wire suite grows by
  one case, and the parity map by no rows, since Together was never in
  `mcplib`.
* Bad, because D1's reasoning shapes rest on documentation and on pi's
  source until the live test runs. A model that rejects `reasoning` would
  fail on the thinking path until corrected.
* Bad, because D3 runs a caller's command inside the library's process.
  The caller owns what it runs; the library bounds how long it runs, and
  what it reads.

### Confirmation

* `together` passes G-wire (its own case) and the ported tests. S7 then
  moves it into `llmprovider/providers/together` with its `llmtest`
  harness.
* A live-tagged Together test (`LLMPROVIDER_LIVE_TOGETHER`) confirms:
  * text, a forced tool, and both thinking shapes of D1;
  * the listing's bare array.
* Kilo device login is tested against a fake server for approved, denied,
  expired and slow cases. A live test on the owner's request confirms it
  against the service.
* The command source is tested for a timeout, oversized output, a failing
  command, the TTL cache, the 401 re-run and redaction. Each test is first
  seen to fail.

## Pros and Cons of the Options

### A. A dedicated `together` provider now

* Good, because it is available now, and it is typed, curated and offered
  by `wizard`.
* Bad, because it is ported once more in S7.

### B. A dedicated provider after S7

* Good, because it is written once, on the final contract.
* Bad, because the owner's request waits for the restructure.

### C. A generic Chat Completions provider

* Good, because one provider serves every compatible service.
* Bad, because it has no curated listing, no descriptor and no per-service
  quirks. Each caller would rediscover Together's reasoning fields.

## More Information

* Evidence:
  [0017-REPORT-reference-client-auth-survey.md](../reports/0017-REPORT-reference-client-auth-survey.md).
* Implementation:
  [0017-PLAN-together-provider-and-auth-extensions.md](0017-PLAN-together-provider-and-auth-extensions.md).
* **Findings that amend current work,** each proposed in its own record:
  * 0016-MADR, "Amendment 2026-09-30": re-read the store after a rejected
    refresh; reopen D3's cross-process lock; the details of `id_token`
    checking.
  * 0015-MADR, "Amendment 2026-09-30": `ErrContextOverflow`.
  * 0015-PLAN: the S9 usage facts, and S7's order with `together`.

## Owner's decisions (2026-09-30)

The owner answered on 2026-09-30: "1. accept d1, proceed with d1-d5. 2. both as recommended. 3. add it."

* **D1–D5 are accepted as written.**
  * D1: Together lands now, in place (0017-PLAN U1).
  * D2 and D3 run after 0015-PLAN S4.
  * D4 is a probe that runs only when the owner asks.
  * D5 stays deferred.
* **The survey's amendments to current work:**
  * 0016-MADR A1 and A3 are accepted, and A2 as option (b);
  * 0015-MADR's `ErrContextOverflow` is accepted.
