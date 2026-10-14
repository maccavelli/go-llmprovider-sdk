---
status: accepted
date: 2026-10-13
decision-makers: repository owner
consulted: the models.dev document (live, 2026-10-13, and the committed 2026-10-12 snapshot); the SDK's own curators, replayed on scratch clones; magic-cli-remote's model menus; opencode's model selection; published usage figures from OpenRouter, Requesty and Artificial Analysis (below)
informed: every caller of `Generate` and `catalog`, gobble-cli and prepare-commit-msg first
---

<!-- markdownlint-disable MD013 -->

# Order a Live Model Menu, and Decide Per-Model Behaviour, From Model Facts

## Context and Problem Statement

The SDK lists each provider's models live, and fetches the models.dev
document, 8,811 models across 228 providers. Yet the order of its menus, and
most per-model decisions, come from the model's **id**: a regex, a prefix
or a hand-written table. The facts the SDK has already fetched go unused.

This record decides how the facts drive both:
- the live menu every provider returns, in what order, and what it shows
  when a listing fails;
- the per-model decisions a request makes.

The menu stays live, as it is today. The owner rejected a menu that changes
only when the SDK does (2026-10-13).

Every claim about code below has an id, E*n*, in the evidence index under
More Information, with its file, line and the text found there. Every
measured number names the data it was measured on, and Phase 0 of the PLAN
commits the measurement as a test.

### 1. What the SDK reads today

* **The live listing is the menu's membership.** `catalog.List` reads the
  provider's listing with the caller's credential (E6, E27). Its result,
  `Catalog`, holds `Recommended`, `Usable`, `Live` and `Err` (E6, E7).
* **The facts are fetched and cached.** The models.dev document (E1) and
  each listing's metadata (E3) are cached in-process for ten minutes (E2,
  E3). `LookupModel` and `CachedModel` merge them field by field (E21,
  E28).
* **What decides from them.** Two places only:
  * the ranking of Zen, Go, Hugging Face, Together and Kilo, which reads
    `reasoning`, the sum of the input and output prices, `release_date`,
    `limit.context`, `status`, `family` and `name` (`metadataCandidate`,
    E29);
  * OpenCode's request route, efforts and reasoning-replay field (E18, E19,
    E20).
* **What decides nothing.**
  * `tool_call`, `structured_output`, `modalities`, `limit.output`, the
    budget bounds of `reasoning_options`, and the cache prices.
  * The whole `openai`, `anthropic`, `google` and `xai` sections. Those
    four providers curate by name scores (E17).
  * The first-party listings' own facts: Anthropic's thinking and efforts,
    xAI's efforts (E30, E31).
  * `CachedModel`, which no code in the SDK calls.

### 2. The per-model decisions made from the id

| # | Rule | Decides | Fact that answers it |
| :--- | :--- | :--- | :--- |
| R1 | `claudeAdaptiveOnly` (E12) | adaptive thinking, or an enabled budget | `reasoning_options`: a `budget_tokens` entry or not; Anthropic's `capabilities.thinking` |
| R2 | `minimaxAdaptiveThinking` (E13) | adaptive thinking with no effort | `reasoning_options`: `toggle` only |
| R3 | the fixed thinking budgets (E32) | the budget | `reasoning_options` `min` and `max`; `limit.output` |
| R4 | `geminiLegacyRE` (E14) | `thinkingBudget` or `thinkingLevel` | `reasoning_options`: a budget or efforts |
| R5 | `thinkingLevel` (E33) | the levels Gemini takes | `reasoning_options` effort values |
| R6 | `effortMenu` (E15) | Grok's efforts | xAI's listing `reasoning_effort`; `reasoning_options` |
| R7 | `MaxTokens: 8192` (E16) | `max_tokens` for every model | `limit.output` |
| R8 | the usability filters (`catalog/models_catalog.go`) | which listed ids are text models | `modalities`, `tool_call`, `status` |
| R9 | the name scorers for OpenAI, Claude, Gemini and Grok (E17) | the order of those four menus | `cost`, `release_date`, `status`, `reasoning` |
| R10 | `unstableModelRE` (E34) | preview and alpha ids left out of the ranking | `status` |

No fact answers the rest, which stay as they are:
- the forced-choice table (0034-MADR);
- the region and serving gates (`excludedAtRequest`);
- the dense-size rule;
- Kilo's `anthropic/` cache rule;
- 0040-MADR's tool-search model rules.

### 3. How well the facts agree with the rules

Measured on the committed snapshot,
`llmprovider/catalog/testdata/ranking-2026-10-12/metadata.json`. The live
document of 2026-10-13 gives the same results to within one model.

| Rule | Agreement with the facts |
| :--- | :--- |
| R1 | **34 of 34** Claude models in the `anthropic`, `opencode` and `opencode-go` sections |
| R2 | `minimax-m3` on Go has `toggle` only: agrees |
| R4 | **27 of 29**. The two others, an image model and a computer-use model, are dropped by the usability filter |
| R6 | 5 of 7 reasoning models. **`grok-4.3`:** xAI's listing gives `xhigh`, and a request accepted it (0033-MADR, "D2's xAI row"), but models.dev omits it. `grok-4.20-multi-agent-0309`: models.dev has efforts the menu omits |
| R7 | **52 models** in the SDK's sections allow less output than 8192 tokens. Every Claude model allows at least 64,000 |

**The facts can be wrong.** `catalog/doc.go` lists models.dev errors that
0033-MADR found, and `grok-4.3` is another. A first-party listing, and a
measured response, outrank models.dev.

### 4. The menu today

* **Its order** comes from the name scorers for the four first-party
  providers (R9), and from the 0009-MADR ranking for the rest.
* **Its full list** is the ranked six, then the rest of the curation
  (0038-MADR A1).
* **Deprecated and tool-less models are in it.** The OpenAI curator, run
  over its models.dev section of 53 models, gives a full list of 42 that
  holds **11 deprecated models**. Among them is `gpt-3.5-turbo` (#32),
  which cannot call tools. Grok's full list of 8 ends with
  `grok-4.20-multi-agent-0309`, which cannot call tools either.
* **A failed listing flaps the menu.** Any listing error drops the result
  to the static list (E4, E5). A live menu of 285 Kilo models becomes the
  static six for that call, though the listing read a few minutes before is
  still cached (E3).
* **Release dates come from models.dev alone,** except Kilo's (E24). The
  recorded listings' own `created` field, compared with models.dev's
  `release_date`:

  | Listing (2026-10-12) | Rows | Distinct `created` | Within 31 days of models.dev |
  | :--- | :--- | :--- | :--- |
  | Kilo | 391 | 368 | 365 of 382 matched |
  | Hugging Face | 125 | 125 | 66 of 69 matched |
  | Together | 265 | 231, 29 of them `0` | 11 of 16 matched |
  | OpenCode Zen | 83 | **1** | 13 of 80 |
  | OpenCode Go | 38 | **1** | 8 of 34 |

  **Zen's and Go's `created` is not a release date:** every row carries the
  same timestamp. A listing date is a fact only where it is measured to be
  one. The first-party listings (OpenAI, Anthropic, xAI) were not recorded,
  and are measured in the PLAN.

### 5. Three flaws in the ranking, independent of the data

* **"Small" is a name regex** (E8), `nano|flash|lite|mini|haiku|small|fast`.
  OpenAI's 2026 tiers are named `luna`, `terra`, `sol` and `astra`. So
  `gpt-6-luna`, at $0.10 input and $0.50 output, is not "small".
* **The diversity group is the family's leading letters** (E9, E10), at
  most two per group (E35). On a one-vendor catalog every OpenAI family,
  from `gpt-luna` to `gpt-sol`, is group `gpt`. The cap admits two GPT
  models, and the ranking reaches for `o3-pro`, at $20 and $80, to fill
  prepare-commit-msg's utility six.
* **`ProfileCapable` almost never reaches its cost key.** Its order is a
  quality signal, then bench, preferred index, not small, newer, costlier
  (E11). Age is a whole number of days, so two models tie on it only when
  released the same day.

  | Section | Eligible pairs | Pairs that reach the cost key |
  | :--- | :--- | :--- |
  | OpenAI | 276 | 13 |
  | Claude | 136 | 3 |
  | Together | 78 | 0 |
  | Kilo | 23,220 | 144 |

  Any profile that only changes that key would rank like `ProfileCapable`.

### 6. What magic-cli-remote does

magic-cli-remote (a local checkout at `9b75e5f3`) builds live model menus
for codex, Grok, opencode and Kilo. Its decisions:

* **The live list is the menu.** It shows every model the engine reports,
  with no top-N. Rows are dropped only at a 500-row cap, and that sets
  `Truncated`: "Silent truncation is not acceptable" (its 0043-MADR D4).
* **The order comes from facts:** the current model first, then the engine's
  recommendation, then "Newest first, where 'newest' is a fact", then the
  engine's order. A deprecated model "sinks to the bottom regardless"
  (0043-MADR D3; `OrderModels`, E36). "The daemon must not invent an
  ordering it cannot justify."
* **Name heuristics are rejected:** "Wrong the first time a vendor renames"
  (0043-MADR's alternatives table).
* **Deprecated models are kept and marked:** "Rank last and badge (D3)
  rather than deny" (the same table).
* **Facts are passed through,** never invented: thinking levels, inputs and
  the tool flag come from the provider (its 0052-MADR D2; `Option`, E37).
* **The cache** (E38):
  * a five-minute time to live;
  * one fetch per key at a time;
  * a generation number that drops entries when the engine restarts;
  * a failure that is never cached.

  The static list is "a floor, never a supplement".
* **The order is deterministic.** Options are sorted by id before the
  stable sort, so Go's map order never reaches the menu (its 0096-MADR D4).

**Its own gaps,** which this record avoids:
- opencode's menu skips the id sort;
- opencode caps its list at 200 without `Truncated`;
- the Grok menu does no ordering at all.

### 7. What the wider research shows

* **opencode** picks a small model by family priority (`gemini-flash`,
  `gpt-nano`, `claude-haiku`), from the newest release down. Its catalog
  keeps only `status === "active"` models with text in and out, and ranks
  cost and age 0.8 to 0.2 with the same small regex (E39, E40).
* **Artificial Analysis** prices a model as a blend of input and output at
  **3:1**, and groups prices in fixed bands: under $0.15, $0.15 to $1, over
  $1 per million tokens.
* **The cache share of agent traffic is published.**
  * OpenRouter, 8 October 2026: "Cached prompt tokens hovered just under 90%
    of all tokens used by agents in September". "Output plus reasoning was a
    sliver of around 1%". The median provider cached 85%.
  * Requesty, April 2026: cache hit rates of 92% for Claude Code, 89% for
    OpenCode and 46% for Kilo Code.
* **A cross-check of the cached-turn price** (D5) against OpenRouter's own
  example for Sonnet 5 ($2 input, $10 output, $0.20 cache read, $2.50 cache
  write), with 0.9 of the input cached and 0.01 of the tokens output:
  * **Uncached:** `0.99 × 2 + 0.01 × 10 = 2.08`. OpenRouter states "about
    $2.08". It matches.
  * **Cached:** 0.48 if the new input costs the input price, 0.53 if it
    costs the write price. OpenRouter states "about $0.51", between the two.
* **Routers** (LiteLLM, OpenRouter's `:floor`, Not Diamond, RouteLLM) route
  each request by cost, latency or a learned quality score. A menu is not a
  router, but they show the norm: deterministic sort keys, and Pareto
  trade-offs where a quality score exists.

### 8. What `List` does to a listing

A consumer that keeps its own model logic needs the provider's listing as
the provider sent it. `List` changes it in seven ways before a caller sees
it:

* **Fields are dropped.** Each listing is decoded into a struct of the
  fields the SDK reads, and the rest are lost. The shared OpenAI-shaped
  fetch keeps only ids (E50); Gemini's keeps its methods among a few others
  (E51).
* **Models are filtered** by the usability filters (E52), and on Together
  by models.dev's section (`listedOnly`).
* **The order is replaced** by the ranking (D4).
* **A failure becomes the static list,** with a nil error (E4, E5).
* **Facts are merged in** from models.dev, which `List` fetches alongside.
* **The listing is cached** for `CachedModel` (E53).
* **Ids are made unique,** and Gemini's `models/` prefix is cut.

Only the transport is the provider's own:
- the credential header;
- the paging (Anthropic's `after_id`, Gemini's `pageToken`, at most ten
  pages, E54);
- the size bound of 8 MiB a page (E55);
- the ChatGPT session's Codex backend listing (E56).

The SDK already reads raw listing bodies in one place: the snapshot
command records each body as received, through a recording transport
(E49).

### 9. What the consumers need

* **gobble-cli** lists the whole menu with `ProfileCapable` (E41) and runs
  every model with tools. It treats `!cat.Live` as a failure to retry
  (E42, E43).
* **prepare-commit-msg** shows `Recommended` (E44). It reports a refused
  credential from `cat.Err` (E45). It forces one tool per request.
* **A consumer with its own mechanics** wants none of this. It wants the
  provider's model data and nothing else, so it can filter, rank, cache and
  store it itself. The owner asked for this on 2026-10-13: "a 'raw' mode,
  simply providing the provider lists without any additional functionality
  from the sdk."

**The problem:**
- Make every provider's menu live, ordered by facts, honest about what each
  model can do and where the list came from, and steady when a listing
  fails.
- Let each request decide per-model behaviour from the same facts.
- Neither may wait on a fetch, and both must work with no metadata at all.
- Let a consumer that wants none of this have each provider's listing as
  the provider sent it.

## Decision Drivers

* **Live.** Membership and facts come from the providers and models.dev
  now, not from a release.
* **Facts, not names.** Order and decisions follow facts. A name rule
  decides only where no fact exists.
* **Never invent.** A fact the source does not give is unknown, and a
  listing field that is not a fact (Zen's `created`) is not read as one.
* **Steady.** A failed listing does not empty or swap the menu, and the
  same facts always give the same order.
* **No request waits** on a fetch, and every path works with metadata off.
* **Every provider,** Ollama included.
* **Additive.** The exported API only gains (R48).
* **Opt out of all of it.** A consumer can have the providers' own data and
  nothing the SDK adds.

## Considered Options

* **A. A live menu ordered by facts.** Facts are merged by precedence;
  every listed model appears, flagged where unusable; a profile ranking
  marks a recommended band; a failure serves the last good listing.
  Per-model request decisions use the same facts, with the name rules as
  the fallback. A separate function returns the raw listings.
* **B. Facts only:** delete the name rules.
* **C. Name rules only,** checked against the facts offline.
* **D. Rank from a snapshot shipped with each release,** with the live data
  only removing models.
* **E. The engine's order only,** as magic-cli-remote's Grok menu: no
  recommendation.
* **F. Nothing.**

## Decision Outcome

Chosen option: **"A. A live menu ordered by facts"**, because:
- it keeps the menu live;
- it orders it by facts, as magic-cli-remote does;
- it keeps the short recommended list prepare-commit-msg needs;
- it removes the flap on failure;
- the facts agree with the rules wherever both answer (§3);
- a consumer that wants none of it calls `ListRaw` instead (D9).

### D1. Facts, and which source wins

* **The order of sources,** field by field:
  1. the caller's layers;
  2. the provider's listing;
  3. models.dev;
  4. the name rule.

  This is 0033-MADR's order. `grok-4.3`'s `xhigh` shows the listing beating
  models.dev (§3). Measured tables stay above all of them: the forced-choice
  table and its learned memo (0034-MADR), and every explicit caller option.
* **A listing field is a fact only where measured to be one.**
  * Release dates from a listing are read for Kilo and Hugging Face (§4).
    For OpenAI, Anthropic and xAI they are read once the PLAN measures them.
  * A listing whose `created` is the same on every row, or is zero, gives
    no date.
* **No request waits, and no request lists.** A provider reads the facts
  once per request, from what is already cached:
  * the caller's layers;
  * a listing that a `catalog.List` or `ListModels` call has already read;
  * the models.dev document.

  A missing document starts its fetch for the next request, as OpenCode's
  route already does (E18). A request never starts a listing fetch, unlike
  `CachedModel`, which "starts any fetch that is missing or stale" (E47). A
  generation then sends no request it did not send before, and the wire
  goldens, recorded with metadata off (E48), keep the rule path.
* **The wire packages stay fact-free.** A provider resolves the facts and
  passes values in. Every wire function keeps today's rule as its default.
* **`canonical_model_id`** (5,946 of 8,811 models carry one) is decoded. It
  fills a gateway model's missing price from the same model in another
  section.

### D2. The per-model request decisions

* **R1–R3, Messages thinking.** From `Reasoning`:
  * a budget bound means an enabled budget, clamped to the bounds;
  * efforts without a budget mean adaptive thinking with the effort;
  * a toggle alone means adaptive thinking with no effort.
* **R4–R5, Gemini thinking.** A budget without efforts means
  `thinkingBudget`. Efforts mean `thinkingLevel`, sent only when the level
  is in the model's list.
* **R6, Grok efforts.** `Reasoning.Efforts`, with the listing above
  models.dev. An effort not in the list clamps as today.
* **R7, the output limit.** A request without `MaxOutputTokens` gets the
  configured default (8192), cut to the model's `MaxOutputTokens` when that
  is smaller. A thinking budget's raise of `max_tokens` never passes it. The
  default is never raised (Q3).
* **Reasoning replay on Chat Completions.** `interleaved.field` names the
  field reasoning is replayed under, for every adapter on that wire, as
  OpenCode does now (E26). Kilo keeps `reasoning_details` (E25).

### D3. The live menu

**Membership.** Every model the provider's live listing serves and the
usability filters admit, which is `Usable` today. Nothing is dropped except
by an explicit `WithLimit`.

**Each entry** (`Catalog.Menu`, a new field) carries:

* its id;
* its facts, as `ModelInfo`, with the source of each (`SourceOf`);
* its rank in the recommended band, 1 to `MaxListed`, or 0;
* its caveats, each a reason it should not be the default:
  * `deprecated`, from `status`;
  * no tools, from `tool_call`;
  * no text output, from `modalities`;
  * expiring within 30 days, from a listing's expiry or shutdown date;
  * failing at request, from `excludedAtRequest`.

  A caveat does not remove the entry. It sinks it, and the caller sees why.

**The order,** the same for every provider:

1. the recommended band, by rank (D4);
2. the other entries without a caveat, newest first by release date (D1),
   then in listing order, then by id;
3. the entries with a caveat, in the same order.

The id is the last key, so the same facts always give the same order.

**`Recommended`** keeps its type and its role (0038-MADR):

* the band, then step 2 of the order;
* entries with a caveat never appear in it;
* `WithLimit(n)` cuts it to its first `n`. `WithLimit(MaxListed)` is the
  band.

0038-MADR's amendment A4 records this change to its full list.

**`Usable`** is unchanged: every admitted id, in the listing's order.

### D4. The recommended band

The band is the profile ranking (0009-MADR), with three changes:

* **"Small" from cost** (V5). A model is small when its 3:1 blended price,
  `(3 × input + output) / 4`, is $1 or less per million tokens, Artificial
  Analysis's middle band. The name regex decides only when the price is
  unknown. The threshold is absolute, so a new expensive model does not move
  it.
* **The group on a one-vendor catalog is the full family.** On OpenAI,
  Claude, Gemini and Grok, the group is models.dev's `family` (`gpt-luna`,
  `claude-opus`). The gateways keep the vendor prefix, where the cap spreads
  the band across vendors.
* **Every provider ranks by facts.** OpenAI, Claude, Gemini and Grok join
  `metadataCurate` (E29), for the band and the rest alike (Q2). Their name
  scorers rank only when no document is available.

A caveated entry is never eligible.

**Replayed on the committed 2026-10-12 snapshot,** with today's rules and
with D4's. The gateways use their recorded listings. The first-party
providers use their models.dev section, since no first-party listing was
recorded. "Kept" is how many of today's six remain.

| Provider | Utility, kept | Capable, kept | What changes |
| :--- | :--- | :--- | :--- |
| OpenCode Zen | 5 of 6 | 6 of 6 | `gpt-6-luna` joins the utility six; `gemini-3.5-flash-lite` leaves it |
| OpenCode Go | 6 of 6 | 3 of 6 | in the capable six, `space-bunny`, `mimo-v2.6-pro` and `gpt-6-luna` are small by price, and give way to `qwen3.8-max`, `kimi-k3` and `glm-5.2` |
| Kilo | 4 of 6 | 6 of 6 | `minimax/minimax-m3` and `nvidia/nemotron-3-ultra-550b-a55b` replace `meta/muse-spark-1.2` and `openai/gpt-6.1-sol` |
| Hugging Face | 3 of 6 | 6 of 6 | `tencent/Hy3`, `google/gemma-4-26B-A4B-it` and `google/gemma-4-31B-it`, now small by price, outrank `thinkingmachines/Inkling-Small` and the two Step models |
| Together | 6 of 6 | 5 of 6 | `nvidia/nemotron-3-ultra-550b-a55b` replaces `MiniMaxAI/MiniMax-M3` |
| OpenAI | 3 of 6 static | 3 of 6 static | utility: `gpt-6-luna`, `gpt-5.6-luna`, `gpt-5.4-nano`, `gpt-5-nano`, `gpt-5-mini`, `gpt-6.1-sol`. Capable: `gpt-6.1-sol`, `gpt-6-sol`, `gpt-6-astra`, `gpt-5.6-terra`, `gpt-5.5-pro`, `gpt-5.5`. No `o3-pro` |
| Claude | 4 of 5 static | 3 of 5 static | utility adds `claude-haiku-4-5` and `claude-opus-5`; capable adds `claude-fable-5-1`, `claude-opus-5` and `claude-fable-5` |
| Gemini, Grok | 3 of 3 / 5 of 5 static | 3 of 3 / 5 of 5 static | the sixes fill out from the section |

The "before" replay equals `TestListModelCatalog_Snapshot20261012`'s pinned
sixes exactly (E22), so the harness reproduces today's ranking.

### D5. `ProfileAgent`, for cached agent sessions

* **A new `Profile` constant,** for a caller that runs long sessions with
  prompt caching on (0039-MADR), as gobble does. Its `ReasoningEffort` is
  `""`, as `ProfileCapable`'s is. Its eligibility is `ProfileCapable`'s.
* **Its band has `ProfileCapable`'s members,** the strongest six. The
  cached-turn price orders them, and never chooses them.
* **Its order within the band:**
  1. a quality signal;
  2. a higher terminalBench;
  3. a lower preferred index;
  4. **the release quarter** (age in 90-day buckets);
  5. **the cheaper cached turn;**
  6. the id.

  The bucket gives the cost key models to decide between (§5).
* **Why the members are `ProfileCapable`'s.** The price was also tried as
  the key that chooses the members, on the committed snapshot with D4's
  rules. The SDK has no quality signal outside Kilo, so the cheaper model
  of a quarter then takes a place from a stronger one:
  * Zen's band kept 1 of capable's 6: `claude-sonnet-5-5` and
    `claude-opus-5-5` gave way to Gemini flash models of the same quarter;
  * the Kilo section's kept 1 of 6;
  * OpenAI, Claude, Hugging Face and Together kept 6 of 6;
  * Go kept 5 of 6, and Gemini 4 of 5.

  An agent's menu must not trade strength for price without a signal of
  strength. Choosing by price waits for one (D8).
* **The cached turn,** per million tokens:

  ```text
  turn = (1 − o) × (h × cache_read + (1 − h) × write) + o × output
  ```

  * `write` is `cost.cache_write` where the model has one, else
    `cost.input`.
  * A missing `cache_read` comes from the same model's other section, by
    `canonical_model_id` (D1). Failing that it is `cost.input`: no discount
    is assumed.
  * An unknown price sorts last.
* **The constants are published figures,** not one person's sessions:
  * `h = 0.90`, the share of an agent turn's input read from the cache
    (OpenRouter's September agent figure; Requesty's 89% for OpenCode);
  * `o = 0.01`, the share of an agent turn's tokens that are output
    (OpenRouter's "around 1%").

  §7's cross-check reproduces OpenRouter's own prices with them.
* **The caller can set its own.**
  * `catalog.WithWorkload(cacheShare, outputShare float64)` is an option,
    scoped as `WithProfile` is. Each value must be in (0, 1).
  * `catalog.ObservedWorkload(usages ...llmprovider.Usage)` returns the two
    shares measured from the caller's own responses, for `WithWorkload`.
    The SDK keeps no hidden state.
* **How much the constants matter.** They order the band, and never change
  its members. In the price-chooses-members trial, across `h` from 0.60 to
  0.96 and `o` from 0.005 to 0.12 (16 combinations):
  * OpenAI, Claude, Gemini, Grok and Hugging Face each gave one band;
  * Go and Together gave 3, Zen 4, and the Kilo section 12.
* `ProfileCapable` and `ProfileUtility` are unchanged by D5. gobble switches
  to `ProfileAgent` by its own change.

### D6. Steady when a listing fails

* **The listing is cached** per provider, base URL and credential
  fingerprint, as its metadata already is (E3). There is one fetch per key
  at a time, and the cache lives for `listingMetadataTTL`.
* **A transient failure serves the last good listing.** A timeout, a 5xx, a
  rate limit or a network error, with a listing for the same key read within
  24 hours, serves that listing. For that response:
  * `Source` is `ListStale`;
  * `FetchedAt` is when it was read;
  * `Err` is the failure;
  * `Live` stays true.

  `Live` stays true because gobble treats `!Live` as "discard and retry"
  (E42). The listing did come from the provider; `Source` says when.
* **An authentication failure never serves the last good listing.** It
  returns the static list with `Err`, as today. prepare-commit-msg reports
  that error (E45), and a revoked key must not keep a menu.
* **The static list is a floor,** used only when no good listing exists. It
  is never merged with a live one.
* **`Catalog.Source`**, of a new type `ListSource` (`ListLive`, `ListStale` or `ListStatic`), and
  **`Catalog.FetchedAt`** are new fields.
* **`catalog.WithLastGood(ids []string, fetchedAt time.Time)`** is a new
  option. It lets a caller hand in a listing it kept from an earlier run,
  used as the last good listing when none is cached. A process that restarts
  while its provider is down keeps its menu. The SDK ships no snapshot.

### D7. Checking the rules and the facts agree

* **Offline, in CI.** A test per D2 rule checks that the fact and the name
  rule agree on every model in the committed snapshot that both answer,
  except a listed set. A cold request (rule) and a warm one (fact) then
  send the same body.
* **At each refresh.** A tool (0035-MADR) records a snapshot with the
  existing command, `internal/cmd/catalogs -snapshot` (E46), and reports:
  * each rule that differs from the facts;
  * how many of each provider's six changed since the last snapshot.

  It reports and does not gate: in a live menu, a change caused by new
  facts is the point.

### D9. Raw listings, on request

**What it is.** `catalog.ListRaw(ctx, id, src, opts...) (RawListing, error)`
is a separate function. A caller asks for raw data by calling it. `List` and
every other path are unchanged.

**It does only the transport** that `List` does:
- the credential, in the provider's header;
- `WithBaseURL`, `WithHTTPClient` and the user agent;
- OpenCode's session header, and Kilo's organisation endpoint;
- every page, up to `maxListingPages`;
- the 8 MiB bound on each page;
- the 10-second bound on the whole listing;
- one renewal of a refused credential, as `List` does.

**It does none of the curation:**
- no usability filter, ranking, band, caveat or static fallback;
- no models.dev fetch and no merged facts;
- no cache read or write, so `CachedFacts` and `CachedModel` never see what
  it fetched;
- no de-duplication, sorting, id rewriting or decoding into `ModelInfo`.

**What it returns:**

```go
type RawListing struct {
	Provider  llmprovider.ProviderID
	Endpoint  string            // the URL listed, without its query string
	FetchedAt time.Time
	Pages     []json.RawMessage // each page's body, byte for byte, in order
	Models    []RawModel        // each entry of the pages' model arrays, in order
}

type RawModel struct {
	ID  string          // the provider's own identifier field, unchanged
	Raw json.RawMessage // the entry's bytes, unchanged
}
```

* **`Pages`** keep each body's envelope: `has_more`, `nextPageToken`,
  `object`, and anything the provider adds later.
* **`Models`** only splits the model array of each page, so a caller need
  not parse each provider's envelope. Each entry's bytes are the provider's.
* **`ID`** is the provider's own identifier field, unchanged:
  * `id` for most;
  * `name` for Gemini, prefix included;
  * `name` for Ollama;
  * `slug` for a ChatGPT session (E57).

  The ChatGPT listing is returned unsorted and unfiltered. `ListModels`
  today sorts it by `priority` and drops entries by `visibility` and
  `supported_in_api` (E57); raw mode does neither.

**Errors** are returned, never degraded:
- a failed page gives a zero `RawListing` and the classified error;
- a page that is not JSON, or has no model array where the provider has
  one, is an error naming the page.

**Every provider:**

| Provider | Endpoint | Paging | Model array |
| :--- | :--- | :--- | :--- |
| Claude | `GET /v1/models` | `has_more`, `after_id` | `data` |
| OpenAI (key) | `GET /models` | none | `data` |
| OpenAI (ChatGPT session) | the Codex backend's `GET /models?client_version=…` (E56) | none | `models` |
| Gemini | `GET /models` | `nextPageToken` | `models` |
| Grok | `GET /models` | none | `data` |
| OpenCode Zen and Go | `GET /models` | none | `data` |
| Kilo | `GET /models`, or the organisation's | none | `data` |
| Hugging Face | `GET /models` | none | `data` |
| Together | `GET /models` | none | the body is the array |
| Ollama | `GET /api/tags` | none | `models` |

The PLAN checks each envelope against a recorded body before it codes it.

**What it does not cover.**
- The models.dev document is not a provider listing. A caller that wants it
  raw already has `LookupMetadata` and `Metadata.Encode`, which keep the
  SDK's sections byte for byte. The whole document is at its URL (E1).
- Per-model detail calls, such as Ollama's `/api/show`, are not listings.

**Its contents carry no promise.** The SDK promises `RawListing`'s shape
(R48). The bytes inside are each provider's, and change when the provider
changes them.

### D8. Out of scope

* Hysteresis on the band, or a margin before a challenger takes a place:
  no measured flapping justifies it.
* External quality signals: OpenRouter's usage rankings, Artificial
  Analysis's index. With one, `ProfileAgent` could let the price choose
  its members (D5).
* `cost.tiers`, `context_over_200k` and `experimental.modes` (fast, flex and
  priority tiers).
* `knowledge`, `open_weights` and `attachment`.
* Removing a name rule entirely. Each stays as the fallback (D1).

### Consequences

* Good, because every menu stays live, and its order is explained by facts
  the caller can see on each entry.
* Good, because a failed listing no longer swaps a live menu for six static
  ids. A revoked key still fails as today.
* Good, because a model the regexes have never seen gets its thinking
  shape, efforts and output limit from the facts as soon as they list it.
* Good, because the 52 models with an output limit under 8192 are no longer
  asked for more.
* Good, because OpenAI's utility six leads with its cheapest current model,
  and holds no `o3-pro`.
* Good, because an agent caller sees the strongest models ordered by what
  their cached turns cost.
* Neutral, because `Usable`, the measured tables and the static lists'
  role as the floor are unchanged.
* Bad, because the sixes change (D4's table). prepare-commit-msg's menu
  changes with them, and `TestStatic_EqualsLatestSnapshot` makes the static
  lists follow (E23).
* Bad, because a cold request uses the rule. Where a rule and a fact
  disagree, warm and cold requests differ. D7's tests keep that set empty
  for the snapshot, and its tool reports new cases.
* Bad, because a models.dev error reaches the wire where the rule was
  right, until D7 finds it. The listing layer limits this for the
  first-party providers.
* Bad, because `Catalog` gains fields, and `Live` now also covers a stale
  listing. A caller that read `Live` as "fetched just now" must read
  `Source`.
* Bad, because the cached-turn constants are fleet averages. A caller whose
  sessions differ should pass its own, through `ObservedWorkload`.
* Good, because a consumer with its own model logic gets every provider's
  listing, byte for byte, through one function, with the SDK's transport and
  none of its curation (D9).
* Bad, because `ListRaw` is one more exported function to keep. Its contents
  change whenever a provider changes its listing, and the SDK promises only
  their shape.

### Confirmation

* **The evidence.**
  * Every E*n* below is checked by a script, which Phase 0 commits as
    `tools/claims.py`.
  * Every number measured on the committed snapshot, in §3–§5, D4 and D5,
    is asserted by an offline test the PLAN commits.
  * The live document's totals (§1, D1) and the published figures (§7) are
    recorded with their source and date under Sources.
* **Offline tests:**
  * D1's precedence;
  * D2's decisions, from the fact, from the rule, and with metadata off;
  * D3's membership, caveats, order and determinism;
  * D4's band, against a new pinned snapshot table;
  * D5's order, prices and options;
  * D6's stale, authentication and floor paths;
  * D9's raw listings: byte-for-byte pages and entries, every page, models
    the filters would drop still present, no fallback, no cache or metadata
    side effect, and each provider's envelope.
* **The goldens.** No recorded wire scenario changes, unless a phase
  records why.
* **Seen failing.** Every new assertion fails on a planted break, on a
  scratch clone.
* **API.** `make api-check` shows only additions.
* **Live.** No billed request is needed:
  * recording the first-party listings in Phase 2 calls listing endpoints,
    which are free, except Together's, which bills (0038-PLAN);
  * the owner may approve one generation per changed thinking shape.

## Pros and Cons of the Options

### A. A live menu ordered by facts

* Good, because it is live, ordered by facts, and steady on failure.
* Good, because it keeps every name rule as a fallback, so offline and cold
  paths work as today.
* Bad, because `Catalog` grows, and the sixes change.

### B. Facts only

* Good, because the name rules go.
* Bad, because a cold cache, metadata off or a failed fetch leaves no
  answer, and every models.dev error reaches the wire.

### C. Name rules only, checked offline

* Good, because nothing changes at runtime.
* Bad, because the menus keep deprecated and tool-less models, and unseen
  models are still guessed.

### D. Rank from a snapshot shipped with each release

* Good, because the menus are reproducible from the release alone.
* Bad, because a menu changes only when the SDK does. The owner rejected
  this on 2026-10-13: "we have to have a dynamic live provider model menu".

### E. The engine's order only

* Good, because there is nothing to rank.
* Bad, because prepare-commit-msg needs a short recommended list, and most
  listings' order means nothing (Kilo's is by price, Hugging Face's by
  speed).

### F. Nothing

* Bad, because the gaps in §3 and §4 stay.

## More Information

### Owner questions

All answered by the owner on 2026-10-13, who then accepted this record:

* **Q1. The order of sources:** caller, listing, models.dev, then the rule
  (D1).
* **Q2. The first-party six:** ranked by facts (D4). The owner also chose
  the two fixes, "small" from cost and the full family as the group, when the
  first ranking put `o3-pro` in OpenAI's utility six.
* **Q3. The default output limit:** cut only (D2).
* **Q4. Cache-aware cost:** a new profile (D5). Its members are
  `ProfileCapable`'s six, and the cached-turn price only orders them. The
  owner confirmed this: "yes, keep the 6 models reorder them".
* **The menu stays live** (D3, D6): "we have to have a dynamic live provider
  model menu not a static menu that only changes when i update the sdk".
* **Raw mode** (D9), added to the scope by the owner: "an opt-in that
  returns raw model schema data from providers without any of the curated
  features or functionality provided by the sdk by default for consumers who
  only want model data and write their own mechanics to manage that data
  outside of the sdk".

### Revision history

* **2026-10-13, first draft.** Facts first for request decisions and
  curation; "small" as the cheapest third of the provider's eligible models;
  `ProfileAgent` with `ProfileCapable`'s order and a cheaper cost last, its
  constants to be measured from one person's sessions.
* **2026-10-13, this revision.** It follows the owner's request for research
  into the first draft's downsides, and the owner's rejection of a pinned
  menu:
  * **"Small"** became V5, the absolute 3:1 band. The cheapest third kept 2
    of 6 of Hugging Face's utility six on the live document; V5 keeps 3.
  * **`ProfileAgent`** gained the release quarter, once §5 showed its cost
    key unreachable. Its constants now come from published figures (§7).
    Its members are `ProfileCapable`'s, once a trial showed the price
    choosing weaker members on Zen (D5).
  * **The live menu** (D3), the steadiness rules (D6), the measured listing
    dates (D1) and the agreement tool (D7) were added.
  * **A pinned ranking** (option D) was considered and rejected.
* **2026-10-13, raw mode.** The owner added raw listings to the scope. §8
  records what `List` does to a listing; D9 decides `ListRaw`. The consumer
  section became §9.

### Sources

These web pages cannot be checked by a script, so the quotes here are the
evidence. Each was read on 2026-10-13.

* OpenRouter, "Cache is Crucial", 8 October 2026,
  `openrouter.ai/data/newsletter/cache-is-crucial`: "Cached prompt tokens
  hovered just under 90% of all tokens used by agents in September"; "output
  plus reasoning was a sliver of around 1%"; the Sonnet 5 example, "about
  $0.51" with caching against "about $2.08" without.
* Requesty, "The Coding Agent Economy",
  `requesty.ai/coding-agent-economy`: cache hit rates for April 2026, Claude
  Code 92%, OpenCode 89%, Aider 84%, Kilo Code 46%; "cached_tokens /
  input_tokens".
* Artificial Analysis, model pages, `artificialanalysis.ai/models/...`: "a
  blend of Input & Output token prices (3:1 ratio)", and the bands "<$0.15,
  $0.15–$1, and >$1".
* LiteLLM, `docs.litellm.ai/docs/routing`: cost-, latency- and usage-based
  routing over a price map with a bundled fallback.
* OpenRouter, `openrouter.ai/docs/features/provider-routing`: `:floor`
  sorts by price, `:nitro` by throughput.
* models.dev, `https://models.dev/api.json`, fetched 2026-10-13, 5,619,333
  bytes, byte-identical to the SDK's default URL (E1).

### Evidence index

<!-- claims: self=3e81e3d magic-cli-remote=9b75e5f3 opencode=907b3bc518 gobble-cli=007b63f prepare-commit-msg=7a2b7b4 -->

Checked on 2026-10-13 against go-llmprovider-sdk `3e81e3d`, gobble-cli
`origin/main` `007b63f`, prepare-commit-msg `7a2b7b4`, magic-cli-remote
`9b75e5f3` and opencode `907b3bc518`, each at that commit, not its working
tree. "Matched" is text found on that line. The first run found nine rows
wrong, by a line or by a revision; they were corrected, and the second run
passed every row.

| Id | Claim | Where | Matched |
| :--- | :--- | :--- | :--- |
| E1 | the default metadata document | `llmprovider/catalog/model_metadata.go:27` | `defaultModelMetadataURL = "https://models.opencode.ai/api.json"` |
| E2 | the document's cache lifetime | `llmprovider/catalog/model_metadata.go:30` | `modelMetadataTTL        = 10 * time.Minute` |
| E3 | the listing metadata's cache lifetime | `llmprovider/catalog/listing_cache.go:19` | `const listingMetadataTTL = 10 * time.Minute` |
| E4 | a failed listing takes the static path | `llmprovider/catalog/discovery.go:194` | `if fetchErr != nil \|\| len(usable) == 0 {` |
| E5 | the static path | `llmprovider/catalog/discovery.go:195` | `cat := staticCatalog(static, limit)` |
| E6 | `Catalog` | `llmprovider/catalog/discovery.go:60` | `type Catalog struct {` |
| E7 | `Catalog.Live` | `llmprovider/catalog/discovery.go:73` | `Live bool` |
| E8 | the small-model regex | `llmprovider/catalog/model_ranking.go:37` | `smallModelRE = regexp.MustCompile` |
| E9 | the diversity group | `llmprovider/catalog/model_ranking.go:292` | `func rankGroup(id, family string) string {` |
| E10 | the group is the family's leading letters | `llmprovider/catalog/model_ranking.go:297` | `familyLetters.FindString(strings.ToLower(family))` |
| E11 | `ProfileCapable`'s order | `llmprovider/catalog/model_ranking.go:195` | `func (n rankNorm) capableOrder(a, b rankCandidate) int {` |
| E12 | R1 | `llmprovider/internal/wire/messages/messages.go:403` | `func claudeAdaptiveOnly(model string) bool {` |
| E13 | R2 | `llmprovider/internal/wire/messages/messages.go:428` | `func minimaxAdaptiveThinking(model string) bool {` |
| E14 | R4 | `llmprovider/internal/wire/generatecontent/generatecontent.go:387` | `var geminiLegacyRE` |
| E15 | R6 | `llmprovider/providers/grok/reasoning.go:37` | `func effortMenu(model string) []string {` |
| E16 | R7 | `llmprovider/settings.go:82` | `MaxTokens: 8192` |
| E17 | OpenAI curates by name | `llmprovider/catalog/discovery.go:258` | `curateOpenAI, cfg.limit` |
| E18 | OpenCode's route from `provider.npm` | `llmprovider/providers/opencode/opencode.go:475` | `meta.NPM(p.gateway, model)` |
| E19 | OpenCode's efforts | `llmprovider/providers/opencode/opencode.go:599` | `meta.ReasoningEfforts(p.gateway, c.model)` |
| E20 | OpenCode's replay field | `llmprovider/providers/opencode/opencode.go:616` | `meta.InterleavedField(p.gateway, c.model)` |
| E21 | `CachedModel` | `llmprovider/catalog/lookup.go:94` | `func CachedModel(` |
| E22 | the pinned sixes | `llmprovider/catalog/snapshot_20261012_test.go:76` | `func TestListModelCatalog_Snapshot20261012(t *testing.T) {` |
| E23 | static lists equal the latest snapshot | `llmprovider/catalog/snapshot_20261012_test.go:124` | `func TestStatic_EqualsLatestSnapshot(t *testing.T) {` |
| E24 | Kilo's listing date | `llmprovider/catalog/discovery.go:872` | `` Created      int64  `json:"created"` `` |
| E25 | Kilo's `reasoning_details` replay | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:58` | `ReplayReasoningDetails replays` |
| E26 | OpenCode's replay field in use | `llmprovider/providers/opencode/opencode.go:583` | `ReplayReasoningField: p.chatReplayField(ctx, c),` |
| E27 | `catalog.List` | `llmprovider/catalog/discovery.go:94` | `func List(ctx context.Context, id llmprovider.ProviderID` |
| E28 | `LookupModel` | `llmprovider/catalog/lookup.go:58` | `func LookupModel(` |
| E29 | ranking from the document | `llmprovider/catalog/model_metadata.go:653` | `func metadataCandidate(id string, m modelMetadata` |
| E30 | Anthropic's capabilities decoded | `llmprovider/catalog/listing_metadata.go:31` | `type claudeCapabilities struct {` |
| E31 | xAI's efforts decoded | `llmprovider/catalog/listing_metadata.go:59` | `ReasoningEffort []string` |
| E32 | the fixed thinking budget | `llmprovider/internal/wire/messages/messages.go:47` | `const defaultThinkingBudget = 4096` |
| E33 | R5 | `llmprovider/providers/gemini/interactions.go:106` | `func thinkingLevel(` |
| E34 | R10 | `llmprovider/catalog/model_ranking.go:44` | `unstableModelRE = regexp.MustCompile` |
| E35 | two per group | `llmprovider/catalog/model_ranking.go:27` | `maxPerRankGroup   = 2` |
| E36 | magic-cli-remote's order | `magic-cli-remote:internal/picker/order.go:58` | `func OrderModels(opts []Option, currentID string) []Option {` |
| E37 | magic-cli-remote passes facts through | `magic-cli-remote:internal/picker/picker.go:58` | `ThinkingLevels []ThinkingLevel` |
| E38 | magic-cli-remote's cache lifetime | `magic-cli-remote:internal/picker/cache.go:14` | `const DefaultCatalogTTL = 5 * time.Minute` |
| E39 | opencode's small-model families | `opencode:packages/opencode/src/provider/provider.ts:2070` | `const smallModelFamilyPriority = ["gemini-flash", "gpt-nano", "claude-haiku"]` |
| E40 | opencode's active, text-only filter | `opencode:packages/core/src/catalog.ts:255` | `model.status === "active"` |
| E41 | gobble lists with `ProfileCapable` | `gobble-cli:llm/provider/list.go:37` | `catalog.WithProfile(catalog.ProfileCapable)` |
| E42 | gobble treats `!Live` as a failure | `gobble-cli:llm/provider/list.go:50` | `if !cat.Live \|\| len(cat.Recommended) == 0 {` |
| E43 | gobble retries when not live | `gobble-cli:internal/cli/models.go:61` | `if !l.result.Live && !l.retrying` |
| E44 | prepare-commit-msg shows `Recommended` | `prepare-commit-msg:internal/ui/setup.go:173` | `return cat.Recommended, nil` |
| E45 | prepare-commit-msg reports a refused credential | `prepare-commit-msg:internal/ui/setup.go:164` | `if refusedCredential(cat.Err) {` |
| E46 | the snapshot recorder | `llmprovider/internal/cmd/catalogs/main.go:15` | `go run ./llmprovider/internal/cmd/catalogs [-doc FILE] [-snapshot DIR]` |
| E47 | `CachedModel` starts missing fetches, the listing's included | `llmprovider/catalog/lookup.go:92` | `any fetch that is missing or stale, so a later call finds it` |
| E48 | the wire goldens run with metadata off | `llmprovider/internal/wirecase/wirecase.go:330` | `extra := []llmprovider.Option{llmprovider.WithoutModelMetadata()}` |
| E49 | the snapshot command records raw bodies | `llmprovider/internal/cmd/catalogs/main.go:155` | `func (r *listingRecorder) RoundTrip(req *http.Request) (*http.Response, error) {` |
| E50 | the OpenAI-shaped fetch keeps ids | `llmprovider/catalog/discovery.go:611` | `func fetchDataIDs(ctx context.Context, endpoint, header, value string` |
| E51 | Gemini's decoded fields | `llmprovider/catalog/discovery.go:316` | `SupportedGenerationMethods []string` |
| E52 | the usability filter | `llmprovider/catalog/discovery.go:651` | `func filterIDs(ids []string, usable func(string) bool) []string {` |
| E53 | a listing is cached | `llmprovider/catalog/discovery.go:157` | `storeListing(listingKeyFor(id, cfg, token), models, false)` |
| E54 | at most ten pages | `llmprovider/catalog/discovery.go:26` | `maxListingPages     = 10` |
| E55 | 8 MiB a page | `llmprovider/catalog/discovery.go:34` | `const listingPageLimit = 8 << 20` |
| E56 | the ChatGPT session's listing | `llmprovider/providers/openai/chatgpt_listing.go:55` | `func (p *provider) listChatGPT(ctx context.Context, token llmprovider.Token) ([]string, error) {` |
| E57 | the ChatGPT entry's fields | `llmprovider/providers/openai/chatgpt_listing.go:46` | `` Slug       string `json:"slug"` `` |

### Related records

* [0038-MADR-full-curated-catalog-by-default.md](0038-MADR-full-curated-catalog-by-default.md):
  the full list, whose order its amendment A4 changes to D3's.
* [0033-MADR-public-model-metadata-and-current-catalogs.md](0033-MADR-public-model-metadata-and-current-catalogs.md):
  `ModelInfo`, its layers, the snapshot command, and `grok-4.3`.
* [0009-MADR-use-case-aware-default-model-ranking.md](0009-MADR-use-case-aware-default-model-ranking.md):
  the ranking D4 changes and D5 extends.
* [0034-MADR-degrade-forced-tool-choice-per-model.md](0034-MADR-degrade-forced-tool-choice-per-model.md):
  the measured table D1 keeps above the facts.
* [0039-MADR-request-side-prompt-caching.md](0039-MADR-request-side-prompt-caching.md):
  the caching `ProfileAgent` assumes.
* [0035-MADR-maintainer-tools-in-python.md](0035-MADR-maintainer-tools-in-python.md):
  where D7's tool and the claims checker live.
* [0040-MADR-native-tool-deferral.md](0040-MADR-native-tool-deferral.md):
  its model rules have no fact, and stay hand-written.
* magic-cli-remote `docs/decisions/0043-MADR-model-selection.md`,
  `0096-MADR-kilo-model-catalog-scope-and-order.md` and
  `0052-MADR-thinking-levels-and-settings.md`: the menu design §6 follows.
