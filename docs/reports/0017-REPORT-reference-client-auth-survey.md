---
status: observation
date: 2026-09-30
---
# Reference-Client Survey: How Six Coding Agents Authenticate, Compared with `llmprovider`

## Purpose

On 2026-09-30 the owner asked for the provider authentication of six coding
agents to be assessed: browser/OAuth and API-key paths alike. The agents are
pi, Grok (`grok-build`), Codex, Kilo, OpenCode, Antigravity CLI (`agy`) and
Claude Code. The owner also asked which findings would improve this module's
`llmprovider`, and for Together AI to be added as a provider.

This report records what was observed. It decides nothing.
[0017-MADR-together-provider-and-auth-extensions.md](../decisions/0017-MADR-together-provider-and-auth-extensions.md)
makes the new decisions. Amendments to
[0016-MADR-provider-auth-and-support-baseline.md](../decisions/0016-MADR-provider-auth-and-support-baseline.md)
and
[0015-MADR-canonical-sdk-api-and-module-layout.md](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md)
carry the findings that land in their phases.

## Method

- **Sources.** Each repository's source was read at the commit below. No
  credential file on the machine was opened. Six read-only research passes,
  one per repository, reported with file and line. Every claim this report
  relies on for a decision was then re-read directly in the source; those
  are marked **(V)**.

  | Repository | Commit | Content |
  |---|---|---|
  | `pi` | `312184edb` (2026-09-29) | TypeScript source; LLM layer in `packages/ai/src` |
  | `grok-build` | `f0e3be11`, `SOURCE_REV` `036a5d8` | Rust source of the Grok CLI |
  | `codex` | `25270df261` | Rust source, `codex-rs/` |
  | `kilocode` | `c267794785` | TypeScript source |
  | `opencode` | `696f41bc8e` | TypeScript source, including the Zen/Go server |
  | `antigravity-cli` | `6dadd62` (v1.2.11) | **no source**: README, CHANGELOG, examples |
  | `claude-code` | `7779afb` (v2.1.283) | **no source**: README, CHANGELOG, examples, plugins |

- **External.** Together AI's API reference
  (`docs.together.ai/reference/chat-completions-1` and `/models-1`), and
  `models.dev/api.json`, both read on 2026-09-30.
- **This module.** `llmprovider/` at `5df7851`.
- **Citations.** Paths are relative to each repository's root and are
  prefixed with its name. This module's paths are unprefixed.

## Findings by repository

### pi

- **P1 — Provider model (R).** A provider is `createProvider({id, baseUrl,
  auth, models, api})` over shared wire implementations
  (`pi: packages/ai/src/models.ts:989-1175`). Per-model quirks are data: an
  OpenAI-completions `compat` record with about 25 flags (`maxTokensField`,
  `thinkingFormat`, `supportsStore`, `sendSessionAffinityHeaders`, …)
  (`pi: packages/ai/src/types.ts:789-866`), plus auto-detection by provider
  id or base URL (`api/openai-completions.ts:1585-1726`).
- **P2 — Together AI (V).** `id: "together"`, `baseUrl:
  "https://api.together.ai/v1"`, key `TOGETHER_API_KEY`, the Chat Completions
  wire (`pi: packages/ai/src/providers/together.ts:6-15`,
  `env-api-keys.ts:106`).
  - Its `thinkingFormat: "together"` sends `reasoning: {enabled}`, and
    `reasoning_effort` only where the model supports it
    (`api/openai-completions.ts:948-956`).
  - It detects Together by provider id, or by a base URL containing
    `api.together.ai` or `api.together.xyz` (`:1595`).
  - Its model list is static, generated from models.dev
    (`scripts/generate-models.ts:2182-2213`).
  - Its overflow message is
    `The input (X tokens) is longer than the model's context length (Y tokens).`
    (`utils/overflow.ts:21,48`).
- **P3 — "Sign in with ChatGPT" for the API (V).** Added
  `02eed88fd`, 2026-09-29.
  - The flow is a public client with **dynamic registration**: `client_id
    dynamic_agent_client`, an agent name hint, and the issued client id
    returned in the callback.
  - Endpoints: `https://auth.openai.com/api/accounts/authorize` and
    `/api/accounts/oauth/token`.
  - `resource https://api.openai.com/v1`, scope `openid profile email
    offline_access resource.invoke chatgpt.tokens.use.direct`.
  - The grant must include the direct scope
    (`pi: packages/ai/src/auth/oauth/openai-chatgpt.ts:15-27`, and its
    refresh at about `:162-223`).
  - The access token is sent to `api.openai.com/v1` as a bearer. pi treats a
    non-`sk-` key there as such a token, and omits fields it rejects, such as
    `prompt_cache_retention` (`api/openai-responses.ts:36-47,329`).
- **P4 — Other flows (R).**
  - **Anthropic Pro/Max:** PKCE at `claude.ai/oauth/authorize`. Requests
    then **present as Claude Code**: `user-agent: claude-cli/2.1.280`,
    `x-app: cli`, and a required "You are Claude Code…" system prefix
    (`auth/oauth/anthropic.ts:14-22`, `api/anthropic-messages.ts:914-968`).
  - **GitHub Copilot:** device code. It sends
    `User-Agent: GitHubCopilotChat/…`, `Editor-Version` and
    `Copilot-Integration-Id: vscode-chat` (`auth/oauth/github-copilot.ts:11-19`).
  - **OpenAI Codex and xAI:** as this module does.
  - **Others:** Kimi, Meta, OpenRouter (PKCE that yields a permanent key) and
    Radius.
- **P5 — Credential storage and refresh (V).**
  - `auth.json` is `0600` and every write happens under a **cross-process
    lock**: `proper-lockfile`, stale after 30 s, with a jittered retry
    (`pi: packages/coding-agent/src/core/auth-storage.ts:116-200`).
  - Refresh runs when less than 5 minutes remain. The rotated credential is
    persisted before the lock is released (`auth/resolve.ts:102-155`).
- **P6 — Keys from commands (R).** A stored key beginning with `!` runs as a
  command, with a 10 s timeout, its output cached per process
  (`pi: packages/coding-agent/src/core/resolve-config-value.ts:80-160`).
- **P7 — Error handling (R).**
  - About 24 vendor context-overflow patterns (`utils/overflow.ts:37-170`).
  - Retry honours `x-should-retry`, `retry-after-ms` and `retry-after`, and
    fails fast past a 60 s server delay (`utils/provider-retry.ts:22-124`).
  - Chat Completions usage reads cached tokens from three spellings:
    `prompt_tokens_details.cached_tokens`, `prompt_cache_hit_tokens` and
    `cached_tokens` (`api/openai-completions.ts:1511-1552`).

### Grok (`grok-build`)

- **G1 — Browser login (R).**
  - OIDC discovery on `https://auth.x.ai`, cached for 1 h, with 2 retries
    (`grok-build: crates/codegen/xai-grok-login/src/oidc/protocol.rs:247-280`).
  - PKCE S256, with a uuid-v7 `state` and `nonce`.
  - Redirect to `http://127.0.0.1:{ephemeral}/callback`, with a CORS
    allowance for `https://accounts.x.ai` (`oidc/login.rs:114,373-387`).
  - `referrer` defaults to `grok-build`.
  - The scopes now add `conversations:*` and `workspaces:*` to the set this
    module requests (`config.rs:14-27`).
- **G2 — `id_token` validation (R).**
  - A `kid` is required, and the JWKS is fetched from `jwks_uri` without
    caching.
  - Allowed algorithms are RS/PS/ES/EdDSA, never HS\*. The algorithm must
    also be in discovery's `id_token_signing_alg_values_supported` when that
    list is present.
  - `iss`, `aud ∋ client_id`, `exp` and `nonce` are checked
    (`oidc/protocol.rs:76-86,588-687`).
  - **Team logins** carry no `id_token` and skip the check (`:698-719`).
  - The device flow decodes its `id_token` without verification
    (`device_code.rs:373-385`).
- **G3 — Refresh (V).**
  - Refresh starts 300 s before expiry.
  - Protection has two layers:
    - an in-process mutex, then a **cross-process `flock`** on
      `auth.json.lock`;
    - a re-read of the file after the lock is taken
      (`manager/refresh_chain.rs:59-162`, `manager/lock.rs:25-101`).
  - **On a rejected refresh token it re-reads the file.** If a sibling
    process has already rotated the token, the failure is demoted to
    transient and the sibling's token is used
    (`manager.rs:1344-1373`, re-read directly).
- **G4 — Session inference host (V).**
  - Subscription traffic goes to `https://cli-chat-proxy.grok.com/v1`, with
    `X-XAI-Token-Auth: xai-grok-cli`.
  - Only API-key traffic goes to `https://api.x.ai/v1`: the source says the
    proxy "NEVER falls back to `xai_api_base_url`"
    (`xai-grok-shell/src/agent/config.rs:48-49,272-283`;
    `xai-grok-login/src/grok_auth_credentials.rs:96-111`).
- **G5 — Storage and logout (R).**
  - `~/.grok/auth.json`, keyed `"{issuer}::{client_id}"`, mode `0600`,
    written through a temp file, `fsync` and rename (`storage.rs:46-55,247-298`).
  - Logout deletes the entry. **No revocation call exists** (`flow.rs:916-950`).

### Codex

- **C1 — Login (R).**
  - Browser PKCE on `127.0.0.1:1455`, falling back to `1457`, with
    `originator`, `id_token_add_organizations` and `codex_cli_simplified_flow`
    on the authorize URL (`codex: codex-rs/login/src/server.rs:77-79,594-608`).
  - Device code at `/api/accounts/deviceauth/{usercode,token}`
    (`device_code_auth.rs:68-147`).
  - The `id_token` claims are **decoded without signature verification**
    (`token_data.rs:129-140`).
- **C2 — Refresh (V).**
  - The request is JSON to `/oauth/token`.
  - These count as permanent: 401, `invalid_grant`, `refresh_token_expired`,
    `refresh_token_reused` and `refresh_token_invalidated`
    (`login/src/auth/manager.rs:1654-1707`).
  - Locking is in-process only, with a re-read from storage inside the lock
    (`:2848-2882`).
  - **No cross-process lock.**
- **C3 — Storage (R).**
  - `auth.json` is rewritten in place with `0600`: no temp file, no `fsync`
    (`storage.rs:206-223`).
  - Optional keyring mode (`storage.rs:235-310`).
- **C4 — Requests (R).**
  - `ChatGPT-Account-ID`, `X-OpenAI-Fedramp`, `originator`, `session-id`, and
    `prompt_cache_key` equal to the session.
  - The usage-limit error types are `usage_limit_reached` (with `resets_at`),
    `usage_not_included` and `insufficient_quota`
    (`codex-api/src/api_bridge.rs:99-238`).

### Kilo (`kilocode`)

- **K1 — Device login (V).**
  - `POST {base}/api/device-auth/codes`, no body, returning `{code,
    verificationUrl, expiresIn}`.
  - Polling is `GET {base}/api/device-auth/codes/{code}`, unauthenticated.
    202 means pending, 403 denied, 410 expired, and 200 returns `{token,
    userEmail}`. It polls every 3 s
    (`kilocode: packages/kilo-gateway/src/auth/device.ts:5-40`,
    `api/constants.ts:28`).
  - The token is opaque and has **no refresh**. The client invents a
    one-year expiry (`api/constants.ts:37`).
  - `/api/profile` then lists organizations (`api/profile.ts:23-60`).
- **K2 — Requests (R).**
  - Chat goes to `https://api.kilo.ai/api/openrouter/`.
  - Per request it also sends `X-Session-Id` and `x-session-affinity` equal
    to the session, besides `X-KILOCODE-TASKID`
    (`packages/opencode/src/session/llm/request.ts:243-258`).
  - The body always carries `usage: {include: true}`
    (`provider/transform.ts:1508-1518`).
  - Non-retryable codes: `PAID_MODEL_AUTH_REQUIRED` and
    `PROMOTION_MODEL_LIMIT_REACHED` (`kilocode/kilo-errors.ts:4-7`).

### OpenCode

- **O1 — Storage (R).** `~/.local/share/opencode/auth.json` holds `oauth`,
  `api` and `wellknown` entries. It is written whole and then chmodded
  `0600`, not atomically (`opencode: packages/opencode/src/auth/index.ts:14-89`,
  `packages/core/src/fs-util.ts:110-114`).
- **O2 — Zen/Go server (V).** The server is in the repository.
  - `"public"` is the anonymous key
    (`packages/console/app/src/routes/zen/util/handler.ts:103-104`).
  - Go serves `chat`, `messages`, `models` and `responses`, and **no Google
    route** (`routes/zen/go/v1/`).
  - `x-opencode-session` is the sticky-routing key (`handler.ts:125-128`).
- **O3 — OAuth (R).**
  - Codex browser flow with `originator=opencode`, and the Codex device flow.
  - xAI device flow with `referrer=opencode`.
  - GitHub Copilot device flow.
  - **Anthropic Pro/Max is absent**; only i18n strings remain.
- **O4 — "wellknown" (R).** `<url>/.well-known/opencode` names a command
  whose output becomes a config variable. That is an enterprise
  remote-config mechanism, not a provider credential
  (`cli/cmd/providers.ts:325-351`).
- **O5 — Together (R).** Only the generic `@ai-sdk/togetherai` package, with
  no Together-specific auth.

### Antigravity CLI and Claude Code (documentation only)

- **A1 — `agy` (R, CHANGELOG).**
  - Google sign-in: a browser flow, or a pasted code over SSH.
  - The OS keyring with a file fallback.
  - Refresh 5 minutes before expiry (`antigravity-cli: CHANGELOG.md:240`).
  - `GEMINI_API_KEY` with `GOOGLE_GEMINI_BASE_URL` (`:372`).
  - ADC or Workforce Identity Federation for GCP projects (`:447-448`).
  - **No client id, endpoint or backend host is established.**
- **A2 — Claude Code (R, CHANGELOG and plugins).**
  - Subscription login, with a pasted code when the callback can't reach
    localhost (`claude-code: CHANGELOG.md:4236`).
  - A **cross-process refresh lock** ("another Claude Code process is
    refreshing it", `:116`, `:1673`, re-read directly).
  - Keys: `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN` (bearer) and
    `ANTHROPIC_BASE_URL`.
  - **`apiKeyHelper`**: a command whose key is cached for 5 minutes
    (`:7525`).
  - Keychain storage on macOS, `~/.claude/.credentials.json` elsewhere.
  - A bundled plugin states that subscription tokens require an exact Claude
    Code system prompt
    (`plugins/security-guidance/hooks/llm.py:66-70`). That is a plugin
    comment, not an API document.
  - **No client id or endpoint is established.**

## Comparison with `llmprovider`

| Mechanism | Reference clients | `llmprovider` at `5df7851` | Status |
|---|---|---|---|
| OpenAI browser PKCE, ports 1455/1457, `originator` on authorize | Codex C1, OpenCode O3, pi P4 | same (`oauth_loopback.go:29,369-379`) | conforms |
| OpenAI device code | Codex C1 | same (`oauth_device.go:85-147`) | conforms |
| Refresh reuse codes are permanent | Codex C2 | same (`oauth_session.go:41-42`) | conforms |
| Usage-limit and quota types | Codex C4, Kilo K2, OpenCode | same (`api_error.go:180-202`) | conforms |
| Revocation, JSON body, `client_id` with a refresh token | Codex | same (`oauth_revoke.go:70-92`) | conforms; Grok has none (G5) |
| Grok session host | `cli-chat-proxy` + `X-XAI-Token-Auth` (G4) | `api.x.ai`, no marker | **recorded divergence**: [0006-MADR](../decisions/0006-MADR-subscription-auth-for-llm-providers.md) decision 2 and its live probe of 2026-09-12 (the proxy answered a third-party session with 426) |
| Kilo chat path | `/api/openrouter` (K2) | `/api/gateway` (`kilo.go:15`) | **recorded divergence**: [0011-REPORT](0011-REPORT-provider-source-compatibility-audit.md) and [0012-PLAN-gateway-conventions.md](../decisions/0012-PLAN-gateway-conventions.md) |
| Zen anonymous key, Go has no Google route | O2 | same (`opencode.go:50-52`, `opencode_route.go:181-220`) | conforms |
| Proxy honoured | all | since `5df7851` (0016 T1) | conforms |
| Re-read the store after a rejected refresh | Grok G3 | re-read **before** refreshing only (`oauth_session.go`, `reloadOrRefresh`) | **gap** |
| Cross-process refresh lock | Grok G3, pi P5, Claude Code A2; Codex has none (C2) | none; 0016-MADR D3 decided none | **evidence against D3** |
| `id_token` signature | Grok verifies at browser login (G2); Codex never (C1) | not yet; 0016-MADR D7 decides to verify | in plan (0016 T2); G2 supplies details |
| Context overflow recognised | pi P7 (about 24 patterns) | OpenAI's `context_length_exceeded` only (`api_error.go:157`) | **gap** |
| Together AI | pi P2, OpenCode O5 | absent | requested by the owner |
| Kilo login | device code (K1) | API key only | **gap** |
| OpenAI sign-in without Codex's client id | pi P3 | borrows Codex's (`oauth_constants.go:7`) | **candidate** |
| Key from a command | Claude Code A2, pi P6, Grok (`GROK_AUTH_PROVIDER_COMMAND`) | none | **candidate** |
| Session-affinity headers on Kilo | K2 | `X-KiloCode-TaskId` only (`kilo.go:292`) | candidate, low value |
| Chat Completions usage and cached tokens | pi P7, Kilo `usage.include` | usage not decoded yet (0015-PLAN S9) | in plan (0015 S9) |
| Anthropic and Copilot subscription | pi P4 (they present as the vendor's own client) | absent; 0016-MADR D10 | excluded; the identity rule of [0012-MADR](../decisions/0012-MADR-conform-providers-to-reference-clients.md) §1.4 forbids presenting as a reference client |
| Keyring storage | Codex C3, `agy` A1, Claude Code A2 | file store | out of reach without a module outside the dependency rule |

## Together AI, from its reference (2026-09-30)

- `POST https://api.together.ai/v1/chat/completions`, with
  `Authorization: Bearer $TOGETHER_API_KEY`.
- **Request fields:** `max_tokens`, `tools`, `tool_choice` (a string, or an
  object naming a function), `reasoning_effort` (`low`, `medium`, `high`),
  `reasoning: {enabled}`, `response_format`, `stream`.
- **Response:** `choices[].message.{content, tool_calls}`, with reasoning in
  `reasoning` or `reasoning_content`.
  - `finish_reason` is one of `stop`, `eos`, `length`, `tool_calls` or
    `function_call`.
  - `usage.{prompt_tokens, completion_tokens, total_tokens}`.
- **Listing:** `GET https://api.together.ai/v1/models` returns a **bare
  array** of `{id, object, created, type, display_name, organization,
  context_length, pricing{input, output, …}}`.
- **models.dev:** the key is `togetherai`, with 36 models, `env
  ["TOGETHER_API_KEY"]` and `npm @ai-sdk/togetherai` (measured 2026-09-30).
- **This module's Chat Completions decoder** already reads both `reasoning`
  and `reasoning_content` (`chatcompletions.go:143-162`). It treats only
  `length` as truncation (`truncation.go:23`), so `eos` decodes as a normal
  stop. The listing helper `fetchDataIDs` expects `{"data": [...]}`
  (`discovery.go:589-627`), so Together needs its own listing decoder.
