---
status: accepted
date: 2026-10-13
decision-makers: repository owner
consulted: vendor documentation read 2026-10-13 (below); the reference clients opencode, kilocode, codex, pi and grok-build, each a local checkout (below); gobble-cli's records
informed: every caller of `Generate`, gobble-cli first
---

<!-- markdownlint-disable MD013 -->

# Let a Request Ask for Prompt Caching

## Context and Problem Statement

The SDK reads every cache count a service reports (0036-MADR-cache-write-token-count.md,
0037-MADR-reasoning-and-cache-read-spellings.md), but no request can ask a
service to cache. 0036-MADR D4 left that to "its own record". This is that
record.

### 1. What the SDK sends today

* **`Request` has no cache field.** Its fields are `Model`, `Instructions`,
  `Input`, `Tools`, `ToolChoice`, `Reasoning`, `MaxOutputTokens` and
  `PreviousResponseID` (`llmprovider/contract.go:169-186`; R6).
* **The Claude body** (`llmprovider/providers/claude/claude.go:273-298`):
  * `system` is one joined string, so no block can carry a marker;
  * `tools` come from `wire.MessagesTools` (`internal/wire/tools.go:81-92`),
    and no tool carries a cache field;
  * the Messages wire's `Block` (`internal/wire/messages/messages.go`) has
    no `cache_control` field.
* **The only cache field the SDK sends** is `prompt_cache_key`, and only on
  a ChatGPT session (`providers/openai/openai.go:246`). Its value is the
  provider's session id.
* **The session id.** `WithSessionID` (`llmprovider/identification.go:22-29`)
  sets "the conversation id sent as x-opencode-session, Kilo's task id, and
  the ChatGPT backend's session-id". Grok, Together and an OpenAI API key
  send none of it.

**What that costs, measured** (0036-PLAN-cache-write-token-count.md,
Phase 1):

* A direct Claude call writes nothing to the cache. Anthropic writes "only
  when the request carries `cache_control`".
* Kilo cached `anthropic/claude-haiku-5.5` on its own: 14,132 of 14,136
  prompt tokens written on the first request, and read on the second.

### 2. What the services offer

All of this was read on 2026-10-13. The quotes are short.

**Anthropic Messages** (platform.claude.com, "Prompt caching", and the
Messages API reference):

* **Two ways to turn caching on.** "Automatic caching: Add a single
  `cache_control` field at the top level of your request. The system
  automatically applies the cache breakpoint to the last cacheable block and
  moves it forward as conversations grow." And "Explicit cache breakpoints:
  Place `cache_control` directly on individual content blocks".
* **Limits.**
  * "You can define up to 4 cache breakpoints". "When used together, the
    automatic cache breakpoint uses one of the 4 available breakpoint
    slots."
  * "If the last block has an explicit `cache_control` with a different
    TTL, the API returns a 400 error."
  * The lookback window is 20 blocks per breakpoint.
* **Prefix order.** "Cache prefixes are created in the following order:
  `tools`, `system`, then `messages`." "Changes at each level invalidate
  that level and all subsequent levels."
* **Lifetime.** `ttl` is `"5m"` (the default) or `"1h"`, with no beta
  header: "Prompt caching no longer requires the beta prefix." "Cache
  entries with longer TTL must appear before shorter TTLs".
* **Price.** A 5-minute write is 1.25 times the input price, a 1-hour write
  2 times, and a read 0.1 times (less on some models).
* **Short prompts.** A prompt below the model's minimum, 512 to 4,096
  tokens, "will be processed without caching, and no error is returned."
* **What invalidates the cache.** Changing the thinking configuration, the
  effort or `tool_choice` invalidates message blocks. "Modifying tool
  definitions … invalidates the entire cache".
* **One platform refuses the top-level field.** "Automatic caching is
  available on every platform except the legacy Amazon Bedrock (Opus 4.6
  and earlier) integration", which returns 400. The SDK has no Bedrock
  provider.

**OpenAI** (developers.openai.com, the prompt-caching guide and the
Responses and Chat Completions references):

* **Caching is automatic.** "Prompt caching is enabled by default for
  supported OpenAI models." On GPT-5.6 and later a write costs 1.25 times
  the input price, and earlier models have "No additional cache-write
  charge".
* **`prompt_cache_options`** `{ mode, ttl }`, plus `prewarm` on Responses,
  is "Supported for `gpt-5.6` and later models".
  * The `ttl` "defaults to `30m`, which is currently the only supported
    value".
  * `mode: "explicit"` with no marked block means "the request does not use
    prompt caching".
* **`prompt_cache_retention`** is "Deprecated. Use
  `prompt_cache_options.ttl` instead."
* **`prompt_cache_key`** steers routing before GPT-5.6. On 5.6 and later
  "OpenAI handles cache routing automatically; the key is not needed to
  optimize caching."

**Other services:**

* **xAI** caches "automatically". The `x-grok-conv-id` header, or
  `prompt_cache_key` in the body, routes a conversation to one server "to
  maximize your cache hit rate".
* **Together** caches automatically, with "no parameter, header, or account
  setting to turn it on". `prompt_cache_key` routes "those requests as a
  group". A write is free.
* **Gemini** caches implicitly on 2.5 and later models: "There is nothing
  you need to do". Explicit caching is a separate `cachedContents`
  resource, in beta.
* **OpenRouter,** which Kilo's gateway is shaped on, takes a top-level
  `cache_control` in a Chat Completions body for Anthropic upstreams, or
  one inside a content part.
* **Kilo's gateway** documents `X-KiloCode-TaskId` "for prompt cache
  keying". It does not document whether it passes `cache_control` on.
* **OpenCode Go** asks for "a stable session ID in `x-opencode-session`".
  **OpenCode Zen** serves Claude on its own `/v1/messages`.
* **Hugging Face's router** documents no caching parameter.

### 3. What the reference clients do

| Client (local checkout) | Anthropic markers | Lifetime | OpenAI | Setting |
| :--- | :--- | :--- | :--- | :--- |
| opencode (`907b3bc518`) | the first two system and the last two other messages; with `cacheControl` set, the top-level automatic form instead (`provider/transform.ts:358-407`, `:465-484`) | 5 minutes | `prompt_cache_key` = the session id (`:1323-1336`) | on for Claude; `setCacheKey` |
| kilocode (`4a7c4e883f`) | as opencode, skipping a trailing `<environment_details>` part; adds OpenAI's explicit breakpoint for GPT-5.6 and later (`transform.ts:410-487`) | 5 minutes | the session id; Kilo's task header (`session/llm/request.ts:262`) | as opencode |
| codex (`823ea830c0`) | none (Responses only) | none | `prompt_cache_key` = the root session id, always (`core/src/client.rs:584-596`, `:984`) | always on |
| pi (`312184edb`) | the system block, the last tool, and the last block of the last user message (`api/anthropic-messages.ts:1101-1109`, `:1122`, `:1407-1433`) | `"1h"` when retention is long (`:70-84`) | the session id; `prompt_cache_options` on GPT-5.6 and later (`openai-responses.ts:97-114`, `:327-336`) | `cacheRetention`: none, short or long, default short |
| grok-build (`2bdd1d6a`) | the last system block, the tip, and the end of the previous request; "The fourth slot stays free: a gateway that turns on automatic caching takes it, and five is rejected outright" (`conversation/messages.rs:36-73`) | 5 minutes | `prompt_cache_key` = the conversation id, and `x-grok-conv-id` on every request (`xai-grok-sampler/src/client.rs:69-74`) | always on |

**Every client caches Claude, and keys OpenAI by session.** They differ in
two ways:

* **Where the markers go.** No two clients agree.
* **Whether caching can be turned off.** Only pi lets a caller turn it off,
  or choose the hour.

### 4. What the consumers need

**gobble-cli** (a local checkout at `ad6ea57`) asks for "cache control:
Anthropic `cache_control`". It is one of three requests to this SDK in its
0012-MADR D13, and not a blocker.

* **What it builds on caching.** Its prompt-cache warming is "blocked" until
  "the SDK supports Claude `cache_control`" (its 0005-MADR).
* **Its prompt.** It keeps the system prompt fixed for a session and only
  appends tools, so that the cached prefix holds (its 0005-MADR, choice 8).
* **Its requests.** Its request builder joins the system messages into
  `Instructions` and sends the history as `Input`
  (`llm/provider/provider.go:191-235`). It sets no cache field and no
  session id.

**prepare-commit-msg** sends one request per commit, each with a new diff:

* **On Anthropic** a write it never reads costs a quarter more.
* **On OpenAI** at GPT-5.6 and later the same write premium applies, and it
  is automatic.

**The problem:** let a caller ask for prompt caching once, in the neutral
`Request`, and have each provider ask its service in the service's own way.
A caller that does not ask must see no change.

## Decision Drivers

* **The contract stays neutral.** A new feature is a field on `Request`
  (R5). A field's zero value means "the provider's default" or "not
  requested" (R6).
* **No change unless asked.** The request on the wire does not change for
  a caller that sets nothing (R45). No caller pays a write it did not ask
  for.
* **A real saving where caching pays:** an agent's growing conversation,
  over a fixed system prompt and tools.
* **The services' limits are the SDK's to keep, not the caller's:** four
  Anthropic breakpoints, one TTL ordering, and room for a gateway's own
  breakpoint.
* **Caching is a hint.** No service fails a request for asking. A short
  prompt is silently not cached. So no request is refused for it.
* **The lifetime is a choice.** On Anthropic, a 1-hour write costs 2 times
  the input price and a 5-minute one 1.25 times, and an agent idle between
  turns needs the hour.
* **The reference clients' practice:** automatic caching, keyed by session,
  and pi's `none`, `short` and `long`.

## Considered Options

* **A.** A `Request.Cache` field, `CacheShort` or `CacheLong`, and each
  provider marks the request in its service's way.
* **B.** Markers in the contract: a cache flag on `MessageItem` and `Tool`,
  placed by the caller.
* **C.** A construction option only, such as `WithPromptCache(retention)`.
* **D.** Always on, as opencode, codex and grok-build do.

## Decision Outcome

Chosen option: "A", because it keeps the contract neutral and leaves every
request unchanged unless asked. It puts the services' breakpoint rules
inside the providers, where they differ, and gives gobble-cli caching with
one field.

### D1. `Request.Cache`

* **The type:** `type CacheRetention string`, with:
  * `CacheShort` (`"short"`): the service's short lifetime, five minutes on
    Anthropic;
  * `CacheLong` (`"long"`): the longest the service offers, one hour on
    Anthropic.
* **The zero value, `""`,** asks for nothing. The request is sent as today,
  and a service that caches on its own (OpenAI, xAI, Together, Gemini, Kilo
  for Claude) still does.
* **Any other value** is refused before the network with
  `ErrInvalidRequest`, as other invalid values are (R23).
* **The names are pi's,** since pi is the one client that offers the choice.
  `none` is not offered (D6, Q3).

### D2. Anthropic Messages: automatic caching, plus one fixed breakpoint

On the Messages wire (Claude, and OpenCode's Claude routes), a request with
`Cache` set carries two markers. Both use the same TTL: `"5m"` for short,
`"1h"` for long.

1. **The top-level `cache_control`,** Anthropic's automatic caching. Its
   breakpoint moves to the last cacheable block as the conversation grows,
   so the growing history is cached with no marker of the caller's.
2. **One explicit breakpoint at the end of the fixed prefix.**
   * It goes on the system prompt's last block, which is sent as a block
     list only when caching is asked.
   * With no system prompt, it goes on the last tool. With neither, there is
     no second marker.
   * It keeps the tools and system cached when a change to the thinking
     configuration, the effort or `tool_choice` invalidates only the
     message blocks.

Two of the four slots are used. Two stay free for a gateway's own markers,
as grok-build leaves one ("five is rejected outright").

### D3. OpenAI, xAI and Together: route by session

These services cache on their own, so `Cache` asks for routing only:

* **OpenAI with an API key, and xAI** (both Responses): `prompt_cache_key`
  is the provider's session id.
* **Together** (Chat Completions): `prompt_cache_key` is the provider's
  session id.
* **The ChatGPT backend** already sends `prompt_cache_key`. It is
  unchanged.

**No other OpenAI field is sent:**

* `prompt_cache_retention` is deprecated.
* `prompt_cache_options.ttl` has one value, the default.
* The explicit breakpoints are for GPT-5.6 and later only.

So `CacheLong` sends the same as `CacheShort` there. Each package doc says
so (R12).

### D4. The gateways

* **OpenCode Zen and Go** send each model on its own wire, so D2 or D3
  applies by route. Their `x-opencode-session` header is unchanged.
* **Kilo** (Chat Completions, shaped on OpenRouter):
  * For a model whose id starts `anthropic/`, a request with `Cache` set
    carries OpenRouter's top-level `cache_control`, with D2's TTL.
  * Its `X-KiloCode-TaskId` header is unchanged.
  * Kilo already caches Claude on its own (§1). Whether it passes the field
    on, and honors the hour, is measured before the PLAN relies on it.
* **Hugging Face, Ollama and Gemini** send nothing for `Cache`.
  * Gemini caches implicitly. Its explicit `cachedContents` is a separate,
    beta resource, out of scope.
  * Each package doc says this.

### D5. No capability, and no warning

* **No capability gate.** A provider that sends nothing for `Cache` still
  serves the request. So no `Capabilities` field is added, and no request is
  refused for asking (R10, R11 do not apply).
* **No warning.** `Response.Warnings` is not used either: a service that
  caches on its own did what was asked.
* **The record is the package docs.** Each says what it sends (R12).

### D6. Out of scope

* **Turning caching off** (`none`, Q3). Only GPT-5.6 and later can turn its
  automatic caching off (`prompt_cache_options.mode: "explicit"`). How the
  older models treat that field is not documented.
* **Explicit breakpoints placed by the caller** (option B).
* **Cache warming:** a caller sends the request again within the lifetime.
* **Gemini's `cachedContents`,** and OpenAI's `prompt_cache_breakpoint` and
  `prewarm`.
* **A per-request session id.** `WithSessionID` stays per provider.
  gobble-cli's own records plan one provider per session.

### Consequences

* Good, because one field turns on caching for every provider that can
  ask, and the services' breakpoint rules stay inside the providers.
* Good, because a caller that sets nothing sends what it sends today, byte
  for byte (R45).
* Good, because gobble-cli's cache warming is unblocked. Its fixed system
  prompt and tools sit under D2's fixed breakpoint, and its history under
  the automatic one.
* Good, because a write is reported in `CacheWriteTokens` and its hour in
  `CacheWrite1hTokens` (0036), so a caller can see what caching cost.
* Neutral, because OpenAI, xAI, Together and Gemini already cache.
  `CacheLong` there sends the same as `CacheShort`.
* Bad, because the SDK chooses the breakpoints, so a caller cannot move
  them. The automatic breakpoint's 20-block lookback can miss a turn that
  appends more than 20 blocks. grok-build adds a third breakpoint for this,
  which this record does not (Q2).
* Bad, because the Kilo field rests on OpenRouter's documented behavior,
  not Kilo's, until it is measured.
* Bad, because R6 must be amended: `Request` gains a field.

### Confirmation

* **Goldens.** The G-wire goldens do not change for a request without
  `Cache` (R45). New goldens show each wire's body with `CacheShort` and
  `CacheLong`:
  * Messages: the top-level field and the system or tool breakpoint, each
    with the TTL;
  * Responses and Chat Completions: `prompt_cache_key`;
  * Kilo: the top-level field for an `anthropic/` model only.
* **Tests:**
  * an unknown `CacheRetention` is refused before the network;
  * no request ever carries more than two Anthropic markers, or two TTLs;
  * the system is sent as a block list only when caching is asked.
* **A billed live run,** with the count approved first, sends each of these
  twice:
  * Claude direct, short and long;
  * Claude through OpenCode Zen;
  * Claude through Kilo;
  * OpenAI with an API key.

  The second request must read what the first wrote, and the first's
  `CacheWrite1hTokens` must be non-zero on a long request to Claude
  directly.
* **Every new assertion** is seen failing on a plant.
* **`tools/apidiff.py`** reports only the additions: the field, the type and
  its two constants.

## Pros and Cons of the Options

### A. `Request.Cache`, placed by the provider

* Good, because the contract stays neutral, and one field serves every
  service.
* Good, because the zero value changes nothing on the wire.
* Bad, because the caller cannot place its own breakpoints.

### B. Markers in the contract

* Good, because it gives the most control, as Anthropic's explicit form
  does.
* Bad, because it puts one vendor's model in the neutral contract. On
  OpenAI and xAI it means nothing.
* Bad, because the four-slot limit, the TTL ordering and a gateway's slot
  become the caller's problem, and a fifth marker is a 400.

### C. A construction option only

* Good, because it is set once, which suits a one-provider-per-session
  harness.
* Bad, because R5 makes a new feature a `Request` field. A caller that
  mixes one-shot and conversational requests on one provider could not
  choose per request.

### D. Always on

* Good, because no caller has to ask, as in opencode, codex and grok-build.
* Bad, because a one-shot caller pays a write premium it never recovers:
  1.25 times on Anthropic. prepare-commit-msg is one.
* Bad, because every request's wire changes, against R45.

## More Information

### Owner questions

Each has a recommendation. On 2026-10-13 the owner answered four:

* **Q1: ask for nothing** by default (D1).
* **Q2: the top-level field plus one fixed breakpoint** (D2).
* **Q4: send `prompt_cache_key` only when `Cache` is set** (D3).
* **Q5: send Kilo's Claude models the top-level field,** measured first
  (D4).

Q3 was not put to the owner. Its recommendation, defer `CacheNone`, stands,
as D6 records.

* **Q1. The default.** Should `Cache`'s zero value ask for nothing (D1,
  recommended), or ask for `CacheShort`? Asking by default changes every
  request's wire, and charges one-shot callers.
* **Q2. Anthropic's markers.**
  * **The top-level field plus one fixed breakpoint** (D2, recommended):
    two slots.
  * **The top-level field alone:** one slot, but a change to `tool_choice`
    or the effort costs the tools and system too.
  * **Also grok-build's "end of the previous request" breakpoint:** three
    slots, and it covers a turn longer than 20 blocks.
* **Q3. A `CacheNone` value**, to turn off GPT-5.6's automatic caching for
  a one-shot caller. The recommendation is to defer it until the older
  models' handling of `prompt_cache_options` is measured. Turning it off is
  additive later.
* **Q4. The routing key.** Should `prompt_cache_key` be sent only when
  `Cache` is set (D3, recommended), or whenever a session id is? The second
  changes the wire for every caller with a session.
* **Q5. Kilo.** Should Claude models get OpenRouter's top-level field (D4,
  recommended, measured first), or nothing, leaving Kilo's own caching?

### Sources

* **Anthropic:**
  * `platform.claude.com/docs/en/build-with-claude/prompt-caching`
  * `platform.claude.com/docs/en/api/messages/create`
* **OpenAI:**
  * `developers.openai.com/api/docs/guides/prompt-caching`
  * the Responses and Chat Completions references
* **xAI:** `docs.x.ai/developers/advanced-api-usage/prompt-caching`
* **Together:** `docs.together.ai/docs/inference/chat/prompt-caching`
* **Gemini:** `ai.google.dev/gemini-api/docs/generate-content/caching`
* **OpenRouter:** `openrouter.ai/docs/guides/best-practices/prompt-caching`
* **Kilo:** `kilo.ai/docs/gateway/authentication`
* **OpenCode:** `opencode.ai/docs/go`, `opencode.ai/docs/zen`

All were read on 2026-10-13.

### Related records

* **0036-MADR-cache-write-token-count.md:** the write counts this record
  makes visible. Its D4 defers this record.
* **0037-MADR-reasoning-and-cache-read-spellings.md:** the read counts.
* **0012-MADR-conform-providers-to-reference-clients.md:** the session id
  and `prompt_cache_key` on the ChatGPT backend.
* **0015-MADR-canonical-sdk-api-and-module-layout.md:** R5 and R6, which D1
  amends. `docs/guides/api-standards.md` changes with the code.
