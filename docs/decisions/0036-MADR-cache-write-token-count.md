---
status: accepted
date: 2026-10-12
decision-makers: repository owner
consulted: provider documentation and reference clients, read 2026-10-12 (below)
informed: gobble-cli maintainers, and any caller that prices a response
---

<!-- markdownlint-disable MD013 -->

# Report Cache-Write Tokens in `Usage`, as a Part of the Input Count

## Context and Problem Statement

`llmprovider.Usage` has four counts: `InputTokens`, `OutputTokens`,
`ReasoningTokens` and `CachedTokens` (`llmprovider/contract.go:145`). A
total holds its part, so `CachedTokens` is the part of `InputTokens` read
from the service's cache (0015-MADR-canonical-sdk-api-and-module-layout.md,
amendment "what `Usage` counts"; `docs/guides/api-standards.md` R7).

**No count says how many input tokens were written to the cache.** Two
services now charge those tokens at their own rate:

* **Anthropic** charges a 5-minute write at 1.25 times the input price, and
  a 1-hour write at 2 times.
* **OpenAI,** from GPT-5.6, charges a write at 1.25 times. Earlier models
  have no write charge.

A caller who prices a response from `Usage` therefore prices written tokens
at the plain input rate. On those models that is 20% too low for each
5-minute write, and 50% too low for each 1-hour write.

**The SDK already has the number on one wire and drops it.** The Messages
decoder reads `cache_creation_input_tokens`, adds it to `InputTokens`, and
keeps no separate count (`internal/wire/messages/messages.go:469-475`). Its
stream merge already carries the cumulative write count to the end of the
message (`messages.go:365-366`). The Responses and Chat Completions decoders
do not read the write field that both wires now define.

**gobble-cli has asked for it.** Its `llm.Usage` already has
`CacheWriteTokens` (`llm/event.go:54-61`), which its adapter never sets
(`llm/provider/provider.go:168-171`). Its 0014-REPORT asks for "`Usage`
cache-write, already decoded on Messages, not exported". 0032-PLAN's "not
done" list carries the item as "prompt caching with a cache-write count".

This record decides the count only. How a request asks for caching
(Anthropic's `cache_control`, OpenAI's `prompt_cache_options`) is a separate
gobble-cli item with its own record.

### What each service reports

Read from each vendor's documentation on 2026-10-12, unless marked otherwise.

| Service | Write field | Is the write inside the input total? | Write price | SDK wire |
| :--- | :--- | :--- | :--- | :--- |
| Anthropic Messages | `cache_creation_input_tokens`, and `cache_creation.ephemeral_5m_input_tokens` and `ephemeral_1h_input_tokens`, which sum to it | no: "`total_input_tokens = cache_read_input_tokens + cache_creation_input_tokens + input_tokens`" | 5 min: 1.25×; 1 h: 2× | Messages (Claude; OpenCode Zen's Anthropic route) |
| OpenAI Responses | `input_tokens_details.cache_write_tokens` | yes: the guide's own example computes `inputTokens - cachedTokens - cacheWriteTokens` | GPT-5.6 and later: 1.25×, "not an additive fee"; earlier: "No additional cache-write charge" | Responses (OpenAI, ChatGPT, Grok, Zen's OpenAI route) |
| OpenAI Chat Completions | `prompt_tokens_details.cache_write_tokens` (openai-go `CompletionUsage`) | yes | as above | Chat Completions |
| OpenRouter, and gateways compatible with it | `prompt_tokens_details.cache_write_tokens`, "only returned for models with explicit caching and cache write pricing" | not stated; pi and the OpenRouter AI SDK provider treat reads and writes as separate parts of `prompt_tokens` | the upstream's | Chat Completions (Kilo) |
| Kilo Gateway | `cache_write_tokens` is listed as tracked; the API reference shows only the three totals. *(Measured 2026-10-12: a top-level `usage.cache_creation_input_tokens`; see the amendment.)* | not documented. *(Measured: inside `prompt_tokens`.)* | the upstream's | Chat Completions |
| Gemini `generateContent` | none; `cachedContentTokenCount` is reads | `promptTokenCount` includes cached content | an explicit cache is billed as storage per hour, not per request | generateContent (Gemini; Zen's Google route) |
| Gemini Interactions | none; `total_cached_tokens` is reads | not documented | explicit caching is not supported there | Interactions |
| xAI | none; `cached_tokens` is reads | yes, by its examples | not documented | Responses (Grok) |
| Together AI | none | not documented | "No charge for writing to the cache" | Chat Completions |
| DeepSeek, Hugging Face router, Ollama | none | — | — | Chat Completions |
| OpenCode Zen | not documented: its docs page answered 404 on 2026-10-12. *(Measured 2026-10-12, 0036-PLAN Phase 1: each route passes its upstream's usage through. Messages sends Anthropic's fields, Responses OpenAI's `cache_write_tokens`.)* | as the upstream | — | all four wires |

Sources: `platform.claude.com/docs/en/build-with-claude/prompt-caching`
and `…/streaming`; `developers.openai.com/api/docs/guides/prompt-caching`;
`ai.google.dev/api/generate-content` and
`ai.google.dev/gemini-api/docs/caching`;
`docs.x.ai/developers/advanced-api-usage/prompt-caching/usage-and-pricing`;
`openrouter.ai/docs/guides/administration/usage-accounting`;
`kilo.ai/docs/gateway/usage-and-billing`;
`docs.together.ai/docs/inference/chat/prompt-caching.md`;
`api-docs.deepseek.com/api/create-chat-completion`.

**Two conventions, one rule.** Anthropic counts the cache outside its input
count, and everyone else counts it inside. The SDK already normalizes this:
the Messages decoder adds reads and writes into `InputTokens`. A write count
fits the same rule. Each service that reports both treats reads and writes
as **disjoint** parts of the input: Anthropic's sum, OpenAI's subtraction,
and the OpenRouter comment in pi (`packages/ai/src/api/openai-completions.ts`,
"Do not subtract writes from cached_tokens").

**Two services can write without being asked.**

* OpenAI caches automatically, so on GPT-5.6 and later a plain request can
  carry billed writes today.
* Anthropic writes only when the request carries `cache_control`, on a
  block or once at the top level ("automatic caching"). The SDK sends
  neither today. So the Claude provider reports a write only once
  request-side caching exists, or where a gateway adds `cache_control`
  itself.

**Measured so far.** The field is present, at zero, in this repository's
recorded live payloads:

* the ChatGPT backend's `response.completed`
  (0012-PLAN-chatgpt-backend.md), with `input_tokens_details.cache_write_tokens`;
* Kilo's Chat Completions (0015-PLAN-canonical-sdk-api-and-module-layout.md,
  Phase S9), with `prompt_tokens_details.cache_write_tokens`.

No recorded payload has a write above zero. Whether OpenCode Zen's Messages
route passes Anthropic's write count through is unmeasured.

### What the consumers do

Each was read in a local checkout:

| Consumer | Write count | Input convention | Cost |
| :--- | :--- | :--- | :--- |
| **Pi** (`312184edb`) | `Usage.cacheWrite`, and `cacheWrite1h`, "Subset of `cacheWrite` written with 1h retention" (`packages/ai/src/types.ts:427-448`); filled from Messages, Responses, Chat Completions and Bedrock; Google sets 0 | `input` excludes reads and writes | `calculateCost` picks the price tier by `input + cacheRead + cacheWrite`, prices 5-minute writes at `cost.cacheWrite` and 1-hour writes at twice the input price (`models.ts:1193-1213`) |
| **OpenCode** (`907b3bc518`) | `tokens.cache.write`; its LLM layer keeps "`nonCachedInputTokens + cacheReadInputTokens + cacheWriteInputTokens = inputTokens`" (`packages/llm/src/schema/events.ts:25-26`) | stored input excludes both | `cache.write` × models.dev `cache_write`. Its Zen gateway bills the 5-minute and 1-hour writes apart |
| **Kilo Code** (`4a7c4e883f`) | as OpenCode, which it forks | as OpenCode | the gateway's reported `cost` first, else as OpenCode |
| **Codex** (`823ea830c0`) | `TokenUsage.cache_write_input_tokens`, from Responses `input_tokens_details.cache_write_tokens` (`codex-rs/protocol/src/protocol.rs:2241-2260`) | `input_tokens` includes the cache | no local cost |
| **grok-build** (`2bdd1d6a`) | `TokenUsage.cache_creation_prompt_tokens`, "Part of `prompt_tokens` but distinct from cache reads; 0 on backends without a cache-write signal" (`conversation.rs:771-788`) | `prompt_tokens` includes reads and writes | the provider's reported cost |
| **gobble-cli** (`ad6ea57`) | `llm.Usage.CacheWriteTokens`, never set | — | none yet |
| **ACP** (`acp-go-sdk2`, `fe6b1cb`) | `Usage.CachedWriteTokens`, **unstable** schema only (`types_gen.go:19164-19190`) | not specified | — |

**What follows:**

1. Every reference client but gobble-cli carries a write count, beside
   the read count and disjoint from it.
2. Codex and grok-build hold it inside their input total, as this SDK holds
   `CachedTokens`. Pi and OpenCode subtract both parts to get uncached
   input. Either way, the parts are disjoint.
3. Only Pi and OpenCode's Zen gateway keep the 1-hour writes apart. They
   need to because the 1-hour price is not in models.dev, whose
   `cache_write` is the 5-minute price.

## Decision Drivers

* **A caller can price a response from `Usage` and the model's
  `ModelRates`** (0033-MADR), on every service that reports writes.
* **R7 stands:** a total holds its parts, so `InputTokens +
  OutputTokens` stays the whole count. No existing count changes meaning.
* **Additive only** (R48, `make api-check`): one minor release, and no
  caller changes.
* **Facts, not policy** (0033-MADR Q12): the SDK reports counts, not
  costs.
* **Zero means "not reported",** as for every count in `Usage` (R7).
* **One spelling per wire,** decoded where the SDK already decodes usage.

## Considered Options

* `Usage.CacheWriteTokens`, a part of `InputTokens` beside `CachedTokens`,
  disjoint from it
* `CacheWriteTokens` counted outside `InputTokens`, as Anthropic does
* A nested cache breakdown, such as `Usage.Cache{Read, Write, Write1h}`
* A count that says whether the service reported it (`*int`, or a `Has`
  method)
* No change: callers read the raw response

## Decision Outcome

Chosen option: "`Usage.CacheWriteTokens`, a part of `InputTokens` beside
`CachedTokens`, disjoint from it", because:

* it is the convention of every service but Anthropic, and of Codex and
  grok-build;
* it keeps R7's rule, so no existing count changes;
* it is one exported field, and Anthropic's 1-hour part is a second, a part
  of the first (Q1).

### D1. The field

```go
type Usage struct {
	InputTokens     int
	OutputTokens    int
	ReasoningTokens int
	CachedTokens    int
	// CacheWriteTokens is the part of InputTokens written to the
	// service's cache. It does not overlap CachedTokens.
	CacheWriteTokens int
	// CacheWrite1hTokens is the part of CacheWriteTokens written with a
	// one-hour lifetime. Only Anthropic's Messages reports it.
	CacheWrite1hTokens int
}
```

* **Invariants:**
  * `CachedTokens + CacheWriteTokens <= InputTokens`. So `InputTokens -
    CachedTokens - CacheWriteTokens` is the uncached input.
  * `CacheWrite1hTokens <= CacheWriteTokens`. The rest of the writes is
    the 5-minute part, as Anthropic's two `cache_creation` counts sum to
    `cache_creation_input_tokens`.
* **Zero** when the service reports no write, or wrote none. The two are
  not told apart (Q2, answered: zero).
* **The names:**
  * `CacheWriteTokens` is OpenAI's and OpenRouter's (`cache_write_tokens`),
    and gobble-cli's own;
  * `CacheWrite1hTokens` is Pi's `cacheWrite1h`;
  * the read count keeps its name, `CachedTokens`.

### D2. Each wire

| Wire | `CacheWriteTokens` from | `InputTokens` |
| :--- | :--- | :--- |
| Messages | `cache_creation_input_tokens`; and `CacheWrite1hTokens` from `cache_creation.ephemeral_1h_input_tokens` | unchanged: `input_tokens` + reads + writes |
| Responses | `input_tokens_details.cache_write_tokens` | unchanged: `input_tokens` |
| Chat Completions | `prompt_tokens_details.cache_write_tokens`; *(amended 2026-10-12)* else a top-level `cache_creation_input_tokens` | unchanged: `prompt_tokens` |
| generateContent | 0: the service reports no write | unchanged |
| Interactions | 0: the service reports no write | unchanged |

* **`CacheWrite1hTokens` is zero on every wire but Messages.** OpenAI's
  lifetimes (`in_memory`, `24h`, `30m`) do not change its write price, and
  its usage has no split.
* **The Messages stream** keeps its cumulative merge: a count `message_delta`
  leaves out keeps `message_start`'s. Anthropic's streaming documentation
  shows the cache counts on both events. Whether `message_delta` carries
  the `cache_creation` object is not stated there, so the merge treats it
  as it treats the other counts, and the PLAN's canned streams cover both
  cases.
* **The value is reported as given.** A decoder does not clamp it. A
  gateway whose `prompt_tokens` left writes out would break the invariant.
  The PLAN measures the gateways, and a breach found there is a deviation.
* ~~**No other spelling is read.** Each service that reports writes uses one
  of the spellings above.~~ *(Amended 2026-10-12: Kilo uses a second
  spelling on Chat Completions, which is read after the first. See the
  amendment.)*

### D3. The rule and its check

* **`api-standards.md` R7** gains the fields: "`InputTokens` includes
  `CachedTokens` and `CacheWriteTokens`, which do not overlap, and
  `CacheWriteTokens` includes `CacheWrite1hTokens`." This record amends R7,
  and 0015-MADR's amendment "what `Usage` counts" stands.
* **`llmtest`'s R7 check** refuses a negative count, a `CachedTokens +
  CacheWriteTokens` larger than `InputTokens`, and a `CacheWrite1hTokens`
  larger than `CacheWriteTokens`.
* **`internal/wirecase`'s canned replies** report a write on each wire that
  has one, so every provider's decoded `Usage` is checked against it.

### D4. Out of scope

* **Asking for caching:** Anthropic's `cache_control`, and OpenAI's
  `prompt_cache_retention` and `prompt_cache_options`. That is gobble-cli's
  prompt-caching item, and its own record.
* **A cost helper** (0033-MADR Q12). A caller multiplies each part by its
  `ModelRates` field.
* **A 1-hour price.** models.dev's `cache_write` is the 5-minute price, and
  `ModelRates` has no 1-hour one. Anthropic prices a 1-hour write at twice
  the input price, and a caller applies that itself, as Pi does.
* **gobble-cli's adapter.** It belongs in that repository. One observation
  for it, inferred from reading and not run: its adapter copies the SDK's
  `InputTokens`, which includes reads, into Pi's `input`, which excludes
  them (`acpserver/history.go:116`).

### Consequences

* Good, because a caller can price writes on Anthropic, on OpenAI from
  GPT-5.6, and on gateways that report them. It does so from `Usage` and
  `ModelRates.CacheWrite`, which `v1.9.0` already publishes.
* Good, because the Messages wire stops dropping a number it already
  decodes.
* Good, because gobble-cli can fill the field it already has, and ACP's
  unstable `cachedWriteTokens`.
* Neutral, because services that report no write leave it at zero: Gemini,
  xAI, Together, DeepSeek, Hugging Face and Ollama. Gemini, Together and
  DeepSeek document no write charge. xAI does not document one either way.
* Neutral, because the Claude provider reports a write only once a request
  can ask for caching. That is a later record.
* Bad, because zero cannot say "not reported". A caller cannot tell a
  service that writes nothing from one that does not say (Q2).
* Good, because a caller can price Anthropic's 1-hour writes apart from
  its 5-minute ones (Q1), as Pi and OpenCode's Zen gateway do.
* Neutral, because through this SDK a 1-hour write happens only where a
  gateway chooses that lifetime itself, until a request can ask for one.
  The count is ready before the request is.
* Neutral, because a later lifetime would need another field.

### Confirmation

* **Red first:** every wire test and `wirecase`'s check fail on a canned
  write before the decoders read it, and every plant's FAIL lines are
  recorded.
* **`llmtest`:** its R7 check fails on a planted `Usage` whose parts exceed
  the total.
* **apidiff** of the release against `v1.9.0`: two additions,
  `Usage.CacheWriteTokens` and `Usage.CacheWrite1hTokens`. `make api-check`
  and `make parity-check` pass.
* **Live:** each service that can write is measured with a repeated long
  prompt. This covers OpenAI on a GPT-5.6-or-later model, Kilo, and OpenCode
  Zen's Messages and Responses routes. It checks that the decoded write
  count is above zero once and that the invariant holds. The PLAN states
  the billed request count before any run.
* **The 1-hour part is checked on canned replies only.** No route the SDK
  can drive asks for a 1-hour lifetime, so a live 1-hour write is not
  expected. A live Messages reply that carries `cache_creation` is checked
  to sum to `cache_creation_input_tokens`.

## Pros and Cons of the Options

### `CacheWriteTokens` inside `InputTokens`, disjoint from `CachedTokens`

* Good, because it matches OpenAI, OpenRouter, Codex, grok-build and the
  SDK's own `CachedTokens`.
* Good, because no existing count changes.
* Good, because the 1-hour part fits beside it as a part of it, as Pi's
  `cacheWrite1h` is.
* Neutral, because Pi-shaped callers subtract two parts, not one.

### Outside `InputTokens`, as Anthropic counts it

* Good, because it is Anthropic's own wire convention.
* Bad, because it breaks R7. `InputTokens` would stop being the whole input
  on the Messages wire, a silent change of meaning for every caller.

### A nested cache breakdown

* Good, because a 1-hour part, and any later lifetime, has room.
* Bad, because it either duplicates `CachedTokens` or moves it, and a move
  breaks the API.
* Bad, because no service but Anthropic reports a part by lifetime.

### A presence-aware count

* Good, because "not reported" and "none written" become different.
* Bad, because no other `Usage` count works that way, and R7 says zero
  means not reported.
* Neutral, because pricing does not need the difference: a service that
  reports no write charges no write, except xAI, which documents neither.

### No change

* Good, because it costs nothing.
* Bad, because a caller cannot reach the raw usage, so nobody can price a
  write. The Messages wire keeps discarding a number it has.

## Amendment 2026-10-12: a second write spelling on Chat Completions

Status: **accepted** 2026-10-12. The owner's decision, asked during
0036-PLAN-cache-write-token-count.md Phase 1, before any decoder changed.

### Fact found

Phase 1's measurement sent the same long prompt twice to Kilo's
`openai/gpt-6-luna`, on the Chat Completions wire. Kilo reported the write
under Anthropic's name, at the top level of `usage`:

```text
request 1: "prompt_tokens":8730, "prompt_tokens_details":{"cached_tokens":0,…}, "cache_creation_input_tokens":8727
request 2: "prompt_tokens":8730, "prompt_tokens_details":{"cached_tokens":8727,…}, "cache_creation_input_tokens":0
```

* `prompt_tokens_details` had no `cache_write_tokens` field.
* The write is inside `prompt_tokens`. Request 2 read the 8,727 tokens that
  request 1 wrote, of the same 8,730.
* Kilo's `anthropic/claude-haiku-5.5` was cached the same way, though the
  SDK sent no `cache_control`: 14,132 of 14,136 prompt tokens written on
  request 1, and read on request 2, under the same top-level spelling. So
  Kilo caches Claude on its own, and a caller there sees writes now. Kilo
  sends no lifetime split.
* On 2026-10-02, Kilo's `kilo-auto/free` sent
  `prompt_tokens_details.cache_write_tokens`
  (0015-PLAN-canonical-sdk-api-and-module-layout.md, Phase S9). So the
  spelling varies with the upstream.

D2 read only `prompt_tokens_details.cache_write_tokens`, so Kilo's writes
would have decoded as zero.

### Decided

The owner chose "read both spellings":

* **Chat Completions reads `prompt_tokens_details.cache_write_tokens`
  first, then a top-level `cache_creation_input_tokens`.** That is the
  order its cache read already follows: the documented field first, then
  the gateways' own spellings.
* **The write is inside `prompt_tokens`,** as measured, so `InputTokens` is
  unchanged.
* **Not chosen:**
  * a Kilo-only spelling, since another gateway of the same kind could send
    it;
  * leaving it unread, which would drop a billed count the service sends.

## More Information

### Owner questions

Answered by the owner on 2026-10-12.

* **Q1 (D1, D2). The 1-hour part.** Anthropic splits writes by lifetime, and a
  1-hour write costs 2× where a 5-minute one costs 1.25×. Pi keeps
  `cacheWrite1h`. Offered:
  * **defer it** to the request-side caching record, where a caller first
    chooses a lifetime (recommended). Through this SDK only a gateway can
    pick 1 hour today, and adding the field later is additive;
  * **add `CacheWrite1hTokens` now,** the part of `CacheWriteTokens`
    Anthropic reports as `cache_creation.ephemeral_1h_input_tokens`, zero
    on every other wire.

  **Answer: add `CacheWrite1hTokens` now.** D1, D2 and D3 carry it.
* **Q2 (D1). Zero or unknown.** Offered:
  * **zero when not reported,** as every `Usage` count is (recommended).
    The services without a write field document no write charge, except
    xAI, which documents neither;
  * **a presence flag** for the write count only, at the cost of a second
    convention in `Usage`.

  **Answer: zero when not reported,** as every `Usage` count is.

### Related

* [0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md),
  amendment "what `Usage` counts": the rule this record extends.
* [0033-MADR-public-model-metadata-and-current-catalogs.md](0033-MADR-public-model-metadata-and-current-catalogs.md):
  `ModelRates.CacheWrite`, the price this count is multiplied by.
* [0032-PLAN-tool-strict-and-result-error-flags.md](0032-PLAN-tool-strict-and-result-error-flags.md):
  the gobble-cli items not yet done.
* gobble-cli, `docs/reports/0014-REPORT-llmprovider-sdk-integration.md`
  (F5, and the ask list) and `docs/decisions/0004-MADR-go-module-architecture.md`
  (amendment 2026-10-02), at `ad6ea57`.
* The plan: [0036-PLAN-cache-write-token-count.md](0036-PLAN-cache-write-token-count.md).
  The owner accepted this record on 2026-10-12, after answering Q1 and Q2.
