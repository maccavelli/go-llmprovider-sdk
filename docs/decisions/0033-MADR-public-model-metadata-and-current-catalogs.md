---
status: accepted
date: 2026-10-12
decision-makers: repository owner
consulted: gobble-cli 0005-MADR-v1-feature-scope.md (Model catalog row, choice 11), gobble-cli 0004-MADR-go-module-architecture.md (decision 6), gobble-cli 0014-REPORT-llmprovider-sdk-integration.md (F4), 0009-MADR-use-case-aware-default-model-ranking.md
informed: consumers of go-llmprovider-sdk v1
---

<!-- markdownlint-disable MD013 -->

# Publish Model Metadata, and Bring the Curated Catalogs Up to Date

## Context and Problem Statement

gobble-cli needs per-model facts the SDK does not publish. Its records ask
for a catalog in Pi's `Model` shape:

* context window and max output tokens;
* cost tiers;
* input modalities;
* reasoning, and a per-model thinking map.

gobble-cli 0005-MADR's "Model catalog" row names them. Its choice 11 says
"The catalog's data stays go-llmprovider-sdk's; gobble overlays limits or
costs only where the SDK lacks them". gobble-cli 0004-MADR decision 6 wants
"`MaxOutputTokens` per model from `llm/catalog`".

Until they exist, gobble-cli shows `Cost: $0.000`, a context bar of an
"unknown window", and `usage_update.size` 0. It sends no output cap, so the
SDK's default of 8192 applies. It reads only `Descriptor.StaticModels`, and
it turns the metadata fetch off (`WithoutModelMetadata`).

gobble-cli must start offline: no network call before `session/new`, and
completion reads only its on-disk catalog cache (its 0005-MADR and
0008-MADR).

**What the SDK holds today** (`main` at `692d8cd`):

* **models.dev document.** `catalog` fetches it from
  `models.opencode.ai/api.json`, caching it for 10 minutes with ETag
  revalidation (`catalog/model_metadata.go`).
  * It decodes only four sections: `opencode`, `opencode-go`,
    `huggingface` and `togetherai` (`modelMetadataKey`, :218-230).
  * Of each entry it keeps `name`, `family`, `reasoning`, `cost.input` and
    `cost.output`, `limit.context`, `release_date`, `status`,
    `reasoning_options`, `interleaved` and `provider.npm` (:56-76).
  * Only `ReasoningEfforts`, `InterleavedField` and `NPM` are public. The
    rest feeds ranking only.
* **Kilo's listing** is the only vendor listing whose context length and
  prices are decoded (`discovery.go:800-818`). They are used for ranking
  only.
* **`ListModels`** returns `[]string` (`contract.go:28-30`), and
  `catalog.Catalog` holds ids only.
* **No table** of context windows, output caps or prices exists. Every
  provider defaults `MaxTokens` to 8192 (`settings.go:82`).
* **The curated catalogs** (`catalog/models_catalog.go:36-136`) are
  `Descriptor.StaticModels`. They are the first entries of every menu, and
  what ranking curates against. Their comments date from 2026-07 (Gemini)
  to 2026-09-30 (Together).

**What the providers' official pages say today** (2026-10-11):

| Provider | SDK list entry | Official status | Source |
| :--- | :--- | :--- | :--- |
| OpenAI | `gpt-4.1-nano` | shutdown 2026-10-23, replaced by `gpt-5.6-luna` | developers.openai.com/api/docs/deprecations |
| OpenAI | `o4-mini` | shutdown 2026-10-23, replaced by `gpt-5.6-terra` | same |
| OpenAI | `gpt-4.1-mini`, `gpt-4.1`, `gpt-4o-mini`, `gpt-4o` | not scheduled; current models are `gpt-6-luna`, `gpt-6.1-sol`, `gpt-6-astra` (and `gpt-5.6-*`) | the deprecations page; developers.openai.com/api/docs/models |
| Gemini | `gemini-3.7-flash` (the list's "★ Recommended") | requests "automatically routed to gemini-3.8-flash"; no shutdown date | ai.google.dev/gemini-api/docs/deprecations |
| Gemini | `gemini-3.5-flash` | routed to `gemini-3.6-flash` | same |
| Gemini | `gemini-2.5-flash`, `gemini-2.5-flash-lite` | "not deprecated", but access is limited "to users who have actively used them in the past" | same |
| Gemini | `gemini-3.6-flash`, `gemini-3.5-flash-lite` | current; `gemini-3.8-flash` (2026-09-02) is missing from the list | same |
| Anthropic | `claude-haiku-4-5` (the list's "★ Recommended") | Active, "Not sooner than October 15, 2026" | platform.claude.com/docs/en/about-claude/model-deprecations |
| Anthropic | `claude-sonnet-5`, `claude-sonnet-4-6`, `claude-opus-4-8` | Active; the current generation is `claude-haiku-5-5`, `claude-sonnet-5-5`, `claude-opus-5-5`, `claude-fable-5-1` | same |
| xAI | `grok-4-fast-reasoning` | retired 2026-05-15, redirected to `grok-4.3` at low effort | docs.x.ai/developers/migration/may-15-retirement |
| xAI | `grok-4`, `grok-3-mini`, `grok-3-mini-fast` (the list's "★ Recommended") | absent from the models page; status not found | docs.x.ai/developers/models |
| xAI | `grok-4.6`, `grok-4.5` | listed; the flagship is `grok-4.7` | same |
| Gateways | all 30 ids for Zen, Go, Hugging Face, Kilo and Together | offered and active | each gateway's live listing or docs table |

On 2026-10-23, two of `staticOpenAI`'s six ids stop answering. Three of
the four "★ Recommended" first entries are deprecated, unlisted, or about
to reach their retirement floor.

One more fact bears on which models a catalog may recommend:

* On `claude-fable-5-1`, `claude-opus-5-5` and `claude-sonnet-5-5`,
  `tool_choice` `any` and `tool` return 400 (release notes of 2026-09-01,
  2026-09-22 and 2026-09-28).
* `GenerateToolCall` always forces its tool (`convenience.go:32`).
* The Claude provider sends a forced choice as `any` or `tool` unless
  thinking is on (`providers/claude/claude.go:244-253`).

So `GenerateToolCall` fails on those three models today.

**What the sources offer.** models.dev is the one source covering most
providers:

* its document has `openai` (53 models), `anthropic` (17), `google` (39),
  `xai` (13) and `kilo` (386) sections, besides the four the SDK reads;
* it has no `ollama` section, only `ollama-cloud`.

Its per-model fields include:

* `limit.context`, `limit.output` and `limit.input`;
* `modalities.input`;
* `tool_call`, `attachment` and `reasoning`;
* `reasoning_options` (`effort`, `toggle`, `budget_tokens`);
* `cost.input`, `cost.output`, `cost.cache_read` and `cost.cache_write`;
* `cost.tiers` and `context_over_200k`;
* `release_date`, `knowledge` and `status` (`deprecated`, `beta`,
  `alpha`).

It has errors, though, all observed on 2026-10-11:

* Together's `Kimi-K3` is priced 3.00/15.00, where Together's docs say
  2.70/13.50.
* Hugging Face prices are one provider's, not the router's.
* Kilo's `gemini-3.8-flash` carries a 50%-off promotion.
* Zen deprecations are not marked.
* Hugging Face's `Inkling-Small` has `limit.output` above its context.

Several vendors' own model APIs carry fields models.dev may lack or get
wrong:

* **Anthropic** `GET /v1/models`: `max_input_tokens`, `max_tokens`,
  `capabilities` and `lifecycle`, with no prices.
* **Gemini** `models.list`: `inputTokenLimit`, `outputTokenLimit` and
  `thinking`.
* **xAI** `/v1/models`: `context_length`, reasoning efforts, and prices
  including cached and long-context ones.
* **Kilo:** `context_length`, `top_provider.max_completion_tokens` and
  prices.
* **Together** `/v1/models`: `context_length` and `pricing`.
* **Ollama** `/api/show`: `model_info.<arch>.context_length` and
  `capabilities`.

OpenAI's `/v1/models` returns only an id, `created`, and an optional
`shutdown_date`.

## Decision Drivers

* gobble-cli's catalog row: window, output cap, prices (with cache read and
  write), image input, and reasoning efforts, for every SDK provider.
* gobble-cli must start offline, so the facts must be usable from a
  document the caller already holds.
* 0009-MADR rejected an embedded snapshot as "stale between releases"; the
  library already fetches models.dev.
* R5 and R3: no vendor JSON in the contract. R21: closed sets are named
  types. R29: nothing mutable at package level, and callers get copies.
* R30: no environment reads. R40: a network call takes a context.
* The catalogs must name models the services answer for, and a stale one
  must be caught before a release, not by a user.
* No new dependency.

## Considered Options

* **Layered metadata: models.dev, the provider's own listing, and layers
  the caller adds, resolved through one entry point**
* models.dev and the listings, with a fixed rule and no caller layer
* models.dev only, exposed as it is decoded
* Each vendor's own model API only
* An embedded snapshot in the SDK
* Leave metadata to gobble-cli

## Decision Outcome

Chosen option: "Layered metadata: models.dev, the provider's own listing,
and layers the caller adds, resolved through one entry point", because:

* one source covers nine of the ten providers;
* the vendors' listings correct it where they are authoritative;
* a caller's own overrides, such as gobble-cli's `models.json`, are one
  more layer, not a second merge;
* a new source is a new layer behind the same entry point, so no caller
  changes;
* offline callers get the same type from a document they persist
  themselves, so 0009-MADR's rejection of an embedded snapshot stands.

The curated catalogs are refreshed in the same record and released first,
because two OpenAI ids stop answering on 2026-10-23.

### D1. `catalog.ModelInfo`, with a presence set

*(Amended 2026-10-12: the fields, names and tiers are as "Amendment
(2026-10-12): the research, and the API surface" sets out.)*

A public, copyable value per model, with plain fields:

```go
type ModelInfo struct {
    ID, Name, Family string
    // Context is the context window, in tokens; MaxInput and MaxOutput its
    // input and output limits.
    Context, MaxInput, MaxOutput int
    Input    []Modality // text, image, audio, video, pdf
    Tools    bool
    Reasoning bool
    Efforts  []string   // reasoning_options' effort values
    Cost     ModelCost
    Released time.Time
    Status   ModelStatus // ModelActive, ModelBeta, ModelAlpha, ModelDeprecated, ModelRetired
    Retires  time.Time   // a published shutdown or retirement date
    // unexported: which fields are set, and by which layer
}

type ModelCost struct { // USD per million tokens
    Input, Output, CacheRead, CacheWrite float64
    LongContext ModelCostTier // the price above LongContext.Above input tokens
}

func (m ModelInfo) Has(f Field) bool        // a layer set f
func (m ModelInfo) SourceOf(f Field) string // "models.dev", "listing:anthropic", or a caller layer's name
```

* `Field` is a named type with one constant per field (`FieldContext`,
  `FieldMaxOutput`, `FieldCostCacheWrite`, …).
* A real zero, such as a free model's price, is set. An unknown value is
  not. A field added later is one more constant.
* `Field`, `Modality` and `ModelStatus` are named types (R21). `Input` and
  `Efforts` are copies (R29).

### D2. Layers, and the order they resolve in

*(Amended 2026-10-12: `Layer` takes no context, and the listing table
gains the fields the research found; see the same amendment.)*

A `Layer` is anything that can describe a model:

```go
type Layer interface {
    Name() string
    Model(ctx context.Context, provider llmprovider.ProviderID, id string) (ModelInfo, bool)
}
```

`Lookup` merges layers field by field. A field set by a higher layer wins,
and a field no layer sets stays unset. From highest to lowest:

1. **The caller's layers,** given with `catalog.WithModelLayer(l)`. A later
   one ranks above an earlier one. gobble-cli's `models.json` and
   `modelOverrides` are such a layer.
2. **The provider's own listing,** where `Lookup` was given a credential,
   for the fields that listing is authoritative for:

   | Listing | Fields it sets |
   | :--- | :--- |
   | Anthropic | `MaxInput`, `MaxOutput`, image input, effort levels (amended 2026-10-11: not `Status` or `Retires`; see "Amendment (2026-10-11): sources that lag") |
   | Gemini | `MaxInput`, `MaxOutput` |
   | xAI | `Context`, `Efforts`, `Cost` |
   | Kilo | `Context`, `MaxOutput`, `Cost` |
   | Together | `Context`, `Cost` |
   | OpenAI | `Retires`, from `shutdown_date` |
   | Ollama | `Context`, image input, `Reasoning`, from one model's `/api/show` |

   OpenCode Zen and Go, and Hugging Face, add nothing. Their listings carry
   no metadata, or a per-provider price that is not the router's.
3. **models.dev.** `modelMetadataKey` gains:
   * `openai` → `openai`;
   * `claude` → `anthropic`;
   * `gemini` → `google`;
   * `grok` → `xai`;
   * `kilo` → `kilo`.

   The decoder keeps every field D1 needs. Ollama has no section.

`Metadata`, the models.dev document, is itself a `Layer`. So is a document
the caller parsed (D3), which is how a caller works offline.

The SDK corrects no source. `SourceOf` says which layer set a value, and
the package doc names the known errors.

### D3. One entry point, and a document the caller persists

*(Amended 2026-10-12: `LookupModel` and `CachedModel`, a listing cache,
and a lossless `Encode`; see the same amendment.)*

```go
func Lookup(ctx context.Context, provider llmprovider.ProviderID, src llmprovider.TokenSource,
    model string, opts ...llmprovider.Option) (ModelInfo, error)
func CachedLookup(ctx context.Context, provider llmprovider.ProviderID, src llmprovider.TokenSource,
    model string, opts ...llmprovider.Option) (ModelInfo, bool)

func ParseMetadata(r io.Reader) (Metadata, error)
func (m Metadata) Encode(w io.Writer) error
func (m Metadata) Model(provider llmprovider.ProviderID, id string) (ModelInfo, bool)
func (m Metadata) Models(provider llmprovider.ProviderID) []ModelInfo
```

* **`Lookup`** resolves through D2's layers.
  * It fetches what it needs: the models.dev document, as today, and the
    provider's listing, as `catalog.List` does, when `src` is not nil.
  * It takes a context and bounds its waits (R40). Its errors are bounded
    and redacted (R35).
* **`CachedLookup` never waits.** It answers from what is cached, starts any
  refresh, and reports false when nothing is known.
* **The caller persists, the SDK refreshes.**
  * `Encode` writes the document's sections the SDK reads, in models.dev's
    format, so `ParseMetadata` reads them back unchanged.
  * gobble-cli seeds its cache with its own embedded copy, under its own
    `NOTICE`.
  * It passes the parsed copy as a layer, with `WithoutModelMetadata` to
    stay offline, and writes a fresh copy back after the SDK fetches one.
  * Any models.dev-format document works: a mirror, or a private registry.
* **Existing functions keep their signatures.** `ListModels` stays
  `[]string`. `Catalog` is unchanged.

### D4. The curated catalogs follow the official pages

* **Each static list is rebuilt** from the provider's official models and
  deprecations pages.
  * It holds no id that is retired, scheduled for shutdown, routed to
    another model, or restricted to past users.
  * Every candidate is measured to answer before it is listed.
  * Each list's comment carries the date it was checked.
* **The menu labels** (`modelLabels`) follow. `Label`'s provider argument
  starts to count.
* **The rankers** learn the current naming:
  * OpenAI's `luna`, `terra`, `sol` and `astra`;
  * Gemini 3.8;
  * Claude 5.5;
  * Grok 4.7 and 4.3.
* **The Grok effort menus** gain `grok-4.7` and `grok-4.3`.
* **The OpenCode Go comment** on region-gated DeepSeek models is re-checked
  against the Go docs.
* **A live test fails** when a static id is missing from its provider's
  listing, or is marked deprecated or scheduled for shutdown there. It runs
  with each release's live suites. *(Amended 2026-10-11: a unit test's
  table of retired ids, taken from the deprecations pages, guards what no
  listing reports; see "Amendment (2026-10-11): sources that lag".)*
* **A maintainer tool proposes the next lists.**
  * It reads models.dev and the official listings, and prints a proposed
    diff for a person to review.
  * It is run by hand (`go run` of an internal command), never at runtime,
    and never edits the lists itself.
  * It reads each key from its provider's variable, for presence and for
    the request only, and never prints a key.

### D5. Releases

* **`v1.7.1`:** the catalog refresh and its live test (D4, apart from the
  tool). It changes no exported API.
* **`v1.8.0`:** *(amended 2026-10-11: `v1.9.0`, since 0034-MADR's
  amendment "prevention first" takes `v1.8.0`)* the metadata (D1 to D3)
  and the maintainer tool. It only
  adds API.

### D6. Forced tool choice on the newest Claude models: a record of its own

`claude-fable-5-1`, `claude-opus-5-5` and `claude-sonnet-5-5` answer a
forced `tool_choice` with 400. The fix is its own record, drafted after
this one:

* whether a model accepts forced choice becomes a per-model fact. It is
  read from metadata, such as Anthropic's models API `capabilities`, with
  a measured table as the fallback. It is never a name pattern;
* where forced choice is refused, the provider sends `auto` with strict
  tool use, a `BestEffort` degradation listed in its package doc, as it
  already does with thinking.

Until that record lands, `staticClaude` holds only models measured to
accept forced choice.

*(Answered 2026-10-11 by
[0034-MADR-degrade-forced-tool-choice-per-model.md](0034-MADR-degrade-forced-tool-choice-per-model.md),
released in `v1.8.0`. The per-model fact is a dated table, the caller's
`WithModelForcedChoice`, and a refusal learned in process (its D1, D3,
D9); the capabilities have no entry for it. Where forced choice is refused,
a named tool is sent as structured outputs, not `auto` (its D8), and
`ToolChoiceRequired` fails unless the caller opts in to degrading it (its
D10). `claude-sonnet-5-5` and `claude-opus-5-5` joined `staticClaude`; its
D5.)*

### Measured (2026-10-11)

0033-PLAN-refresh-the-curated-catalogs.md Phase 1 measured each candidate:

* **Every candidate answered.** Each was listed by its provider and
  answered a generation, and the four lists keep three ids or more.
* **Forced choice.** `claude-sonnet-5-5` and `claude-opus-5-5` refused it
  with 400, as Anthropic's release notes say, so they wait for D6's record.
  `claude-haiku-5-5` accepted it.
* **D6's source.** Anthropic's models API `capabilities` has no entry for
  forced tool choice. D6's per-model fact therefore comes from the measured
  table it names as the fallback, unless Anthropic adds one.
* **D2's xAI row.** xAI's listing reports `reasoning_effort` and
  `context_length` per model, as D2's xAI row assumes. Its efforts can
  differ from the docs: it gives `grok-4.3` `xhigh`, which a request
  accepted.

### Amendment (2026-10-11): sources that lag

0033-PLAN-refresh-the-curated-catalogs.md's deviation D1 found that the
machine-readable sources lag the providers' deprecations pages:

* Anthropic's `GET /v1/models` lists `claude-sonnet-4-5-20250929` with
  `lifecycle "active"`, and `deprecated_at` and `retires_at` null. Its
  deprecations page says the model was deprecated on 2026-09-30, and
  retires on 2026-11-30.
* models.dev's `anthropic` section does not mark it deprecated either, nor
  does its `google` section mark the rerouted `gemini-3.7-flash` and
  `gemini-3.5-flash`.
* models.dev does mark OpenAI's `gpt-4.1-nano` and `o4-mini` `deprecated`,
  and OpenAI's listing gives their `shutdown_date`.

The owner chose two guards, and D2 amended:

* **The live guard** stays as built. It fails on a static id missing from
  its listing, or one a listing marks stale (OpenAI's `shutdown_date`,
  Kilo's `expiration_date`). It reads Anthropic's `lifecycle`, but cannot
  rely on it.
* **The unit test's table of retired ids,** taken from the deprecations
  pages, guards what no listing reports: Anthropic's deprecations, and
  Gemini's reroutes. The maintainer tool (D4) reports both, from one table.
* **D2:** Anthropic's listing sets `MaxInput`, `MaxOutput`, image input and
  effort levels, but not `Status` or `Retires`. Those come from models.dev,
  or from a caller's layer.

The other options were declined:

* the guard parsing each provider's deprecations page. It is complete, but
  fails on a redesign of the page's markup rather than on a stale menu;
* keeping Anthropic's listing authoritative for `Status`, which would
  report deprecated models as active.

### Amendment (2026-10-12): the research, and the API surface

0033-PLAN-public-model-metadata.md had not started. Before it did, the
owner asked for the official schemas, the standards, and how the agents
that will read this metadata use it. What follows was read on 2026-10-12.

#### The official schemas

| Concept | models.dev | Anthropic | Gemini | xAI | OpenAI | OpenRouter | LiteLLM |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Context window | `limit.context` | — | — | `context_length` | — | `context_length` | — |
| Max input | `limit.input` | `max_input_tokens` | `inputTokenLimit` | — | — | `per_request_limits.prompt_tokens` | `max_input_tokens` |
| Max output | `limit.output` | `max_tokens` | `outputTokenLimit` | — | — | `top_provider.max_completion_tokens` | `max_output_tokens` |
| Input modalities | `modalities.input` | `capabilities.image_input`, `pdf_input` | — | `input_modalities` (`/v1/language-models`) | — | `architecture.input_modalities` | `supports_vision`, `supports_pdf_input` |
| Tools | `tool_call` | — | — | — | — | `supported_parameters` holds `tools` | `supports_function_calling` |
| Reasoning | `reasoning`, `reasoning_options` | `capabilities.thinking`, `capabilities.effort` | `thinking` | `capabilities.reasoning_effort` | — | `reasoning.supported_efforts` | `supports_reasoning` |
| Prices | `cost.*` | — | — | `*_token_price` | — | `pricing.*` | `*_cost_per_token` |
| Long-context price | `cost.tiers[]`, `context_over_200k` | — | — | `*_long_context`, `long_context_threshold` | — | `pricing.overrides[].min_prompt_tokens` | `*_above_200k_tokens` |
| Release | `release_date` | `created_at` | — | `created` | `created` | `created` | — |
| Lifecycle | `status` | `lifecycle`, `deprecated_at` | — | — | — | — | — |
| Retirement date | — | `retires_at` | — | — | `shutdown_date` | `expiration_date` | `deprecation_date` |

* **models.dev**, `packages/core/src/schema.ts` on its `dev` branch:
  * Prices are numbers, in USD per million tokens: `input` and `output`;
    optionally `reasoning`, `cache_read`, `cache_write`, `input_audio` and
    `output_audio`.
  * `tiers[]` adds a price per tier, each with `tier: {type: "context",
    size}` and a unique `size`. `context_over_200k` is the older form.
  * `reasoning_options[]` is a union on `type`:
    * `toggle`;
    * `effort`, whose `values` are drawn from `null`, `none`, `minimal`,
      `low`, `medium`, `high`, `xhigh`, `max` and `default`;
    * `budget_tokens`, with optional `min` and `max`.

    It is required when `reasoning` is true, and forbidden otherwise.
  * `status` is `alpha`, `beta` or `deprecated`, and no retirement date.
  * Dates are `YYYY-MM` or `YYYY-MM-DD`.
  * Modalities are `text`, `audio`, `image`, `video` and `pdf`.
  * `temperature` and `structured_output` are flags.
  * Its data is MIT-licensed.
* **Anthropic** `GET /v1/models`:
  * `capabilities` holds `{supported}` leaves: `image_input`, `pdf_input`,
    `structured_outputs`, `thinking.types` (`adaptive`, `enabled`,
    `disabled`), and `effort` (`low` to `max`, and `xhigh`).
  * Its docs say `retires_at` can be revised, and that `lifecycle` is the
    signal. "Sources that lag" stands.
* **Gemini** `models`: `inputTokenLimit`, `outputTokenLimit` and `thinking`.
  There are no prices, modalities or lifecycle.
* **xAI** `/v1/models`:
  * `context_length`, and `capabilities.reasoning_effort` with a default;
  * token prices as integers in **USD cents per 100 million tokens**,
    which is 10⁻⁴ USD per million;
  * `long_context_threshold`, the input size "at or above which" the
    `*_long_context` prices apply.
* **OpenAI** `/v1/models`: `shutdown_date`, and no limits or prices.
* **OpenRouter** `/api/v1/models` is the de-facto gateway shape, with
  prices as strings in USD per token.
  * Kilo's `/api/gateway/models` follows it, with `-1` as the price of its
    auto-routing models.
  * Vercel's AI Gateway follows it too, and copies models.dev's
    `reasoning_options`.
* **Together** `/v1/models`: `context_length`, and `pricing.{input, output,
  cached_input}` as numbers whose unit its docs do not state.
* **Ollama** `POST /api/show`:
  * the context window is `model_info["<general.architecture>.context_length"]`;
  * `capabilities` includes `tools`, `vision` and `thinking`.
* **No standard exists.** No W3C, IETF or model-card format describes model
  limits or prices. The de-facto shapes are OpenRouter's and models.dev's.
* **ACP** (`agentclientprotocol.com`, and the Go SDK at schema 1.25.0)
  carries no per-model metadata. Models are `SessionConfigOption`s of
  category `model`.
  * `usage_update` is stable: `size` (the context window, required),
    `used`, and `cost{amount, currency}`, cumulative.
  * The turn's `Usage` is unstable. It counts `cachedReadTokens` and
    `cachedWriteTokens` apart.
* **Go:**
  * `anthropic-sdk-go`'s `ModelInfo` keeps presence apart from the value,
    asked with `.Valid()`.
  * `google.golang.org/genai`'s `Model` lets zero mean unknown
    (`InputTokenLimit`, `OutputTokenLimit`, all `omitempty`).
  * Go 1.27 ships `encoding/json/v2`, which this SDK already decodes with.

#### The consumers

Each was read in a local checkout:

* **OpenCode** (`907b3bc518`) and **Kilo Code** (`4a7c4e883f`):
  * They read models.dev, cache it on disk, embed a snapshot at build
    time, and merge their config over a provider listing over models.dev,
    field by field.
  * A missing price becomes 0 and shows as "Free".
  * A missing context becomes 0, which turns compaction off.
  * A config override rebuilds `cost` and drops the tiers.
  * They take reasoning variants from `reasoning_options`, filter out
    `deprecated` models, and sort by `release_date`.
  * Kilo sets an OpenRouter model's `release_date` to the day it fetched
    it.
* **Codex** (`823ea830c0`):
  * Its `ModelInfo` has `context_window: Option<i64>`, `input_modalities`,
    `supported_reasoning_levels` with a default, and `upgrade.retirement_at`.
  * It has no prices.
  * It reads a bundled `models.json` and a remote `/models`, cached on
    disk for 300 s with an ETag.
  * Config overrides individual fields.
  * It compacts at 90% of the window.
* **grok-build** (`2bdd1d6a`):
  * Its window cannot be empty, so an unknown one is a 256,000-token
    sentinel value.
  * It has `max_completion_tokens`, an effort menu, a severity-tagged
    notice for lifecycle, and no prices.
  * It caches like Codex, and compacts at 85%.
* **Pi** (`312184edb`), whose shape gobble-cli asks for (D1's context):
  * Its `Model` has `contextWindow`, `maxTokens`, `input`, a `cost` in USD
    per million with `tiers[].inputTokensAbove`, and a `thinkingLevelMap`
    built from models.dev's `reasoning_options`.
  * Missing values become 4096 and a price of 0.
* **gobble-cli** (`ad6ea57`):
  * It still shows `Cost: $0.000` and "an unknown window". It sends ACP
    `usage_update` without `size`, and no output cap.
  * Completion reads only its on-disk cache, within 150 ms.

#### What follows

1. **Unknown is not zero.** Every consumer without presence pays for it:
   "Free" for an unpriced model, compaction off for an unknown window, a
   256k sentinel. D1's presence set (Q3) is confirmed.
2. **Facts, not policy.** Each consumer compacts by its own rule (90%, 85%,
   window minus output, input limit minus a reserve), and computes cost
   its own way. The SDK publishes the facts, and no usable-window or cost
   helper (Q12).
3. **Prices are a list of tiers** (Q9). A model can carry more than one,
   and models.dev, xAI, OpenRouter and LiteLLM each say it differently.
4. **Reasoning is a descriptor** (Q10). Pi's thinking map and OpenCode's
   variants are built from the effort values, the toggle and the budget,
   not from a flag.
5. **The names are the SDK's own** (Q11). `Request.MaxOutputTokens`
   already exists, and Pi and Codex say `contextWindow` and
   `context_window`.
6. **A caller states presence.** D1's presence set is unexported, so a
   caller's layer could not mark a field known, and a known zero (a free
   price) could not be said at all. `Known` lets every layer, the SDK's
   own included, say which fields it sets (Q11).
7. **Offline, as D3 decided.** Every agent read keeps a bundled snapshot,
   a disk cache and an ETag. D3's "the caller persists, the SDK refreshes"
   fits. `Encode` writes each section's bytes as they were read, so a
   field the SDK does not decode yet survives the round trip.
8. **`CachedModel` needs a listing cache.** It must not wait, and no
   listing is cached today. D3 gains an in-process cache of the listings'
   metadata. It lives 10 minutes, as the models.dev cache does, and is
   keyed by provider, base URL and a credential fingerprint, as the probe
   cache is.
9. **`Layer` takes no context.** A caller's layer is data it already
   holds, and the SDK's listings are not caller layers. A layer must not
   block, so `Metadata` satisfies `Layer` as it is.
10. **Prices are USD per million tokens,** the unit of models.dev, Pi,
    OpenCode, Kilo's catalog and ACP's `cost.amount`. xAI's and the
    per-token strings are converted. Together's unit is measured before it
    is trusted.

#### D1, amended

```go
type ModelInfo struct {
    ID, Name, Family string
    ContextWindow, MaxInputTokens, MaxOutputTokens int
    Input, Output    []Modality
    Tools, Temperature, StructuredOutput bool
    Reasoning        ModelReasoning
    Cost             ModelCost
    Released         time.Time
    Status           ModelStatus
    Retires          time.Time
    // unexported: which fields are set, and by which layer
}

type ModelReasoning struct {
    Supported, Toggle    bool
    Efforts              []llmprovider.Effort // as the source names them
    BudgetMin, BudgetMax int
}

type ModelRates struct { // USD per million tokens
    Input, Output, CacheRead, CacheWrite, Reasoning float64
}

type ModelCost struct {
    ModelRates
    Tiers []ModelCostTier // ascending by Above
}

type ModelCostTier struct {
    Above int // the tier applies to more input tokens than this
    ModelRates
}

func (c ModelCost) RatesFor(inputTokens int) ModelRates
func (m ModelInfo) Has(f Field) bool
func (m ModelInfo) SourceOf(f Field) string
func (m ModelInfo) Known(fields ...Field) ModelInfo
```

* **`Field`** has one constant per field: `FieldName`, `FieldFamily`,
  `FieldContextWindow`, `FieldMaxInputTokens`, `FieldMaxOutputTokens`,
  `FieldInput`, `FieldOutput`, `FieldTools`, `FieldTemperature`,
  `FieldStructuredOutput`, `FieldReasoning`, `FieldReasoningToggle`,
  `FieldReasoningEfforts`, `FieldReasoningBudget`, `FieldCostInput`,
  `FieldCostOutput`, `FieldCostCacheRead`, `FieldCostCacheWrite`,
  `FieldCostReasoning`, `FieldCostTiers`, `FieldReleased`, `FieldStatus`
  and `FieldRetires`. A field added later is one more constant.
* **The other named types** (R21):
  * `Modality`: `ModalityText`, `ModalityImage`, `ModalityAudio`,
    `ModalityVideo`, `ModalityPDF`;
  * `ModelStatus`: `StatusActive`, `StatusBeta`, `StatusAlpha`,
    `StatusDeprecated`, `StatusRetired`.
* **`Known`** returns a copy with those fields marked as set, by the layer
  that returned it. A layer's presence is exactly what it marks, and no
  value is inferred from being non-zero.
* **Merging is by `Field`.** `FieldCostTiers` replaces the whole list, as
  Codex replaces an effort menu, so a caller's override never drops tiers
  by accident, as OpenCode's does.
* **`RatesFor`** returns the rates of the highest tier whose `Above` the
  input exceeds, or the base rates.
  * This is the rule of Pi and OpenCode.
  * xAI's "at or above" threshold *t* is the tier `Above` *t* − 1.
  * A tier's rate its source omits is the base rate.
* **Efforts keep the source's values.** They include `none`, `minimal` and
  `max`, which `llmprovider.Effort` holds without constants, as Codex
  keeps `Custom`. models.dev's `null` is read as `none`, as OpenCode does.
* **Out of the type:** image size limits, audio prices and knowledge
  cutoff. Each would be one more `Field` later.

#### D2, amended

`Layer` is `{Name() string; Model(provider llmprovider.ProviderID, id
string) (ModelInfo, bool)}`. It must not block or reach the network.
The listings set:

| Listing | Fields |
| :--- | :--- |
| Anthropic | `MaxInputTokens`, `MaxOutputTokens`; `Input` image and pdf; `StructuredOutput`; `Reasoning` (supported and toggle, from `thinking`), efforts (from `effort`). Not `Status` or `Retires`. |
| Gemini | `MaxInputTokens`, `MaxOutputTokens`, `Reasoning` (from `thinking`) |
| xAI | `ContextWindow`, efforts, `Cost` (input, output, cache read, and a tier at `long_context_threshold`), converted from cents per 100 million |
| Kilo | `ContextWindow`, `MaxOutputTokens`, `Cost` (input, output; a `-1` price is unset), `Retires` (from `expiration_date`). *(Amended 2026-10-12, Phase 1: also cache read, cache write and reasoning, from `input_cache_read`, `input_cache_write` and `internal_reasoning`, when present. `discount` is not applied, so the rates are list prices.)* |
| Together | `ContextWindow`, `Cost` (input, output, cache read), once Phase 1 confirms the unit. *(Phase 1, 2026-10-12: the unit is USD per million.)* |
| OpenAI | `Retires`, from `shutdown_date` |
| Ollama | `ContextWindow`, `Tools`, `Input` image, `Reasoning`, from one model's `/api/show` |

OpenCode Zen and Go, and Hugging Face, still add nothing.

#### D3, amended

```go
func LookupModel(ctx context.Context, provider llmprovider.ProviderID, src llmprovider.TokenSource,
    model string, opts ...llmprovider.Option) (ModelInfo, error)
func CachedModel(ctx context.Context, provider llmprovider.ProviderID, src llmprovider.TokenSource,
    model string, opts ...llmprovider.Option) (ModelInfo, bool)
func WithModelLayer(l Layer) llmprovider.Option

func ParseMetadata(r io.Reader) (Metadata, error)
func (m Metadata) Encode(w io.Writer) error
func (m Metadata) Name() string
func (m Metadata) Model(provider llmprovider.ProviderID, id string) (ModelInfo, bool)
func (m Metadata) Models(provider llmprovider.ProviderID) []ModelInfo
```

* **A model no layer knows** is not an error. Its fields are all unset, as
  `Has` reports.
* **A source that fails** is joined into the error, beside what the other
  layers set, as `catalog.List` reports a failed listing in `Catalog.Err`.
* **`WithModelLayer`** is a `catalog` option, scoped as `WithProfile` is
  (R17).
* **`Models`** returns copies, sorted by id (R29). A slice, not an
  iterator, because callers sort and filter it.
* **Unchanged:** `LookupMetadata`, `LookupMetadataWith`,
  `CachedMetadataWith` and `ListModels`.
* *(Added 2026-10-12, 0033-PLAN-public-model-metadata.md Phase 2's
  deviation.)* A fetched document keeps each section's bytes, and decodes a
  section the first time it is read, once. A caller pays only for the
  sections it reads. `ParseMetadata` decodes every section at once, so a
  malformed one is reported to the caller. On the fetch path, a section that
  does not decode is treated as absent, as a section without models already
  was.

### Owner questions

Answered by the owner on 2026-10-11: the recommended option of each.

* **Q1 (D2). Sources.** Offered:
  * layered sources the caller can add to (recommended);
  * models.dev and the listings with a fixed rule;
  * models.dev only;
  * vendor APIs only.

  **Answer: layered, and the caller can add layers.**
* **Q2 (D3). Offline.** Offered:
  * the caller persists the document and the SDK refreshes it
    (recommended);
  * `ParseMetadata` only;
  * an opt-in snapshot package;
  * a snapshot embedded in `catalog`.

  **Answer: the caller persists, the SDK refreshes.**
* **Q3 (D1). Unknown values.** Offered:
  * plain values with a presence set (recommended);
  * zero meaning unknown, with `Cost` a pointer;
  * a pointer per field.

  **Answer: values and a presence set.**
* **Q4 (D3). API shape.** Offered:
  * one resolver entry point (recommended);
  * a method per source;
  * extending `ListModels`.

  **Answer: one resolver.**
* **Q5 (D5). Order.** Offered:
  * the refresh first as `v1.7.1` (recommended);
  * both in `v1.8.0`.

  **Answer: the refresh first.**
* **Q6 (D6). Forced tool choice.** Offered:
  * its own record, as a per-model fact (recommended);
  * its own record, by name pattern;
  * folded into the refresh.

  **Answer: its own record, as a per-model fact.**
* **Q7 (D4). Keeping the menus current.** Offered:
  * curated lists, the live test, and a maintainer tool (recommended);
  * metadata-driven ranking for the vendors too;
  * the refresh and the live test only.

  **Answer: curated lists, the live test, and the tool.**
* **Q8 (D4). Candidates.** Offered:
  * the refresh PLAN's lists (recommended);
  * the current generation only;
  * the flagship first.

  **Answer: the refresh PLAN's lists.**

Answered by the owner on 2026-10-12, after the research above: the
recommended option of each.

* **Q9 (D1). Price tiers.** Offered:
  * an ordered list of tiers, with `RatesFor` (recommended);
  * D1's one long-context tier.

  **Answer: an ordered list.**
* **Q10 (D1). Reasoning and capability fields.** Offered:
  * the full descriptor (toggle, efforts, budget), with `Temperature`,
    `StructuredOutput`, `Output` and a reasoning price (recommended);
  * D1's flag and effort list.

  **Answer: the full descriptor, and the fields.**
* **Q11 (D1, D3). Names, and presence from a caller.** Offered:
  * `ContextWindow`, `MaxInputTokens`, `MaxOutputTokens`; `LookupModel`
    and `CachedModel`; and `Known` (recommended);
  * D1's and D3's names, with presence inferred from non-zero values.

  **Answer: the SDK's names, and `Known`.**
* **Q12 (D3). A cost helper.** Offered:
  * facts only now, with the helper in the cache-write record
    (recommended);
  * `ModelCost.Of(Usage)` now, short of cache writes.

  **Answer: facts only.**

### Consequences

* Good, because gobble-cli can show a window, a cost and an output cap for
  every provider, offline from its own cache, with its overrides as one
  more layer.
* Good, because a new source or a new field changes no caller.
* Good, because the menus stop recommending retired and rerouted models
  before two of them fail.
* Good, because a stale catalog fails a live test, and the tool proposes
  its replacement.
* Neutral, because the values are only as good as their layers. The SDK
  reports them and says which layer set each; it does not vouch for them.
* Bad, because `Lookup` merges up to three kinds of layer. Its rule per
  field must be tested per listing.
* Bad, because the catalogs still need a person to accept each refresh.
* Bad, because `CachedModel` needs a cache of the listings' metadata,
  which the SDK did not keep before. *(Added 2026-10-12.)*
* Good, because the names, the tiers and the reasoning descriptor match
  what Pi, Codex and OpenCode already read, so a consumer maps them one to
  one. *(Added 2026-10-12.)*

### Confirmation

* Each new test is seen failing first.
* The refresh's lists, and every field D2 maps, are measured live before
  they are built.
* The G-wire listing goldens change only as the refresh PLAN records.
* apidiff: nothing for `v1.7.1`; additions only for `v1.8.0`. *(Amended
  2026-10-11: the metadata's release is `v1.9.0`.)*
* The live catalog test passes on each release commit.

### Amendment (2026-10-12): the second refresh, and what the command reads

Status: **proposed**. The owner answered Q13 to Q15 on 2026-10-12, with
the recommended options; it is accepted when the owner says so.

0033-PLAN-public-model-metadata.md's Phase 4 ran D4's maintainer command
for the first time. It proposed 49 changes, and recorded one as found
outside that plan: "`opencode-zen remove deepseek-v4-flash not in the
listing`. Zen's listing no longer offers a model its static list names."
0036-PLAN-cache-write-token-count.md repeated it. This amendment rests on
the research for a second refresh, on 2026-10-12.

#### F1. The removal is a false positive, from one workspace

* **The live guard passes.** `TestLive_StaticCatalogsCurrent`, run on
  2026-10-12, passes for all nine providers. It reads Zen's public listing,
  with no key, and that listing offers `deepseek-v4-flash`.
* **The command reads a different listing.** It calls `catalog.List` with
  the provider's key, and compares against `Usable`.
* **Zen's listing depends on the key's workspace.** The gateway's own route
  (OpenCode `907b3bc518`, `packages/console/app/src/routes/zen/v1/models.ts`)
  drops, for a keyed request, every model the key's workspace has disabled
  (its `ModelTable` rows).
* **Measured on 2026-10-12:**
  * the public listing had 83 models, and the owner's keyed listing 45;
  * every keyed model was in the public listing;
  * the 38 public-only models included `deepseek-v4-flash`, the Claude Opus
    4.5 to 4.8 and Sonnet 4.5 and 4.6 models, the GPT-5 to 5.6 models,
    and Gemini 3 to 3.7 Flash;
  * models.dev's `opencode` section marks all 38 `active`.

So `deepseek-v4-flash` is not retired. That workspace disables it. The
runtime menu is right to read the keyed listing, since a caller cannot use
a model its workspace disables. A curated list serves every workspace, so
its check reads the public one, as the guard does.

#### F2. The command's additions are not how the lists are built

* **The command adds** any model newer than a list's newest that is not
  known to lack tools. That gave 50 proposals on 2026-10-12: 36 for Kilo,
  11 for Zen, 2 for OpenCode Go, and the removal above.
* **The lists are built by a rule:**
  * Zen, Go, Kilo and Hugging Face hold the utility six that the shipped
    ranker chose, under 0009-MADR's §3 and §4, on 2026-09-26 (§7). Their
    comments say so.
  * Together holds 0017-MADR D1's rule over models.dev's `togetherai`
    section: tool-calling chat models, not deprecated, text out, one per
    vendor, newest first.
* **Each rule, run on 2026-10-12,** from a scratch program and a scratch
  script, on the public listings the guard reads (Kilo and Together with
  their keys):

  | List | Today's rule output, against the list |
  | :--- | :--- |
  | OpenCode Zen | adds `claude-haiku-5-5` (released 2026-10-07); `gemini-3.8-flash` falls to seventh. `deepseek-v4-flash` stays |
  | OpenCode Go | adds `claude-haiku-5-5` and `space-bunny`; drops `mimo-v2.6-pro` and `hy3` |
  | Kilo | `openai/gpt-6.1-sol` replaces `thinkingmachines/inkling` |
  | Hugging Face | unchanged |
  | Together | `zai-org/GLM-5.3-Flash`, `thinkingmachines/Inkling` and `nvidia/nemotron-3-ultra-550b-a55b` in; `zai-org/GLM-5.3`, `Qwen/Qwen3.6-Plus` and `openai/gpt-oss-120b` out. This is a scratch reading of D1's rule, which the PLAN reproduces before use |

* **Two of these need a look before they are listed:**
  * `openai/gpt-6.1-sol` is $2.00 per million input tokens and $10.00
    output, against `thinkingmachines/inkling`'s $0.95 and $4.05
    (models.dev's `kilo` section, 2026-10-12). The ranker's age and cost
    blend still chose it for the utility profile;
  * `space-bunny` is unknown to the records here. D4 requires it to be
    measured, and 0009 §3's data-policy rule applies.
* **What tests pin:** `TestStaticOpenCatalogs_UtilityCriteria` checks 0009
  §3's criteria on each six (six ids, none free, no dense model of 14B or
  less, no `-contributor`, nothing region-gated on Go), not the ids
  themselves.

#### Decided

* **D4a. The command reads OpenCode's public listing.** For OpenCode Zen and
  Go it reads the listing with no key, as the live guard does. A keyed
  listing describes one workspace (Q13).
* **D4b. The command proposes each list by its own rule.** For a ranked list
  it prints the ranker's utility six from today's listing, and the
  difference. For Together it prints 0017-MADR D1's rule. The broad
  candidate list goes (Q14).
* **D4c. This refresh applies each rule's output.** Every new id is
  measured to answer before it is listed (D4), and the 0009 §3 criteria
  test passes. `openai/gpt-6.1-sol` and `space-bunny` are put to the owner
  with their measurements (Q15).
* **D4d. The records are corrected.**
  0033-PLAN-public-model-metadata.md's Phase 4 finding, and
  0036-PLAN-cache-write-token-count.md's "not done" entry, each gain a
  dated note that the removal was a workspace's setting.
* **Unchanged:** the runtime catalog keeps reading the keyed listing, and
  the live guard keeps reading the public one.

#### Owner questions (this amendment)

* **Q13 (D4a). The command's OpenCode listing.** Offered:
  * **the public listing** (recommended). The curated list serves every
    workspace, and the guard already reads it;
  * **the keyed listing, labelled.** An absence is reported as "not in this
    workspace's listing", and never proposed as a removal.

  **Answer: the public listing.**
* **Q14 (D4b). What the command proposes.** Offered:
  * **each list by its own rule** (recommended). The output is the diff a
    refresh would apply, about a dozen lines, not 50;
  * **both:** the rule's output first, then the broad candidates, for
    review.

  **Answer: each list by its own rule.**
* **Q15 (D4c). This refresh.** Offered:
  * **apply each rule's output, measured** (recommended). The two
    surprising picks are shown to the owner with their measurements before
    they are listed;
  * **correct the records only.** The lists change at the next refresh.

  **Answer: apply each rule's output, measured,** with the two surprising
  picks shown to the owner before they are listed.

#### Consequences of this amendment

* Good, because a curated list no longer loses a live model to one
  workspace's settings.
* Good, because the command's output is the change a refresh would make,
  and the guard and the command read the same listing.
* Neutral, because a refresh is a patch release: no exported API changes.
* Bad, because a ranked six can pick a model a person would not, such as
  `openai/gpt-6.1-sol`. Q15 keeps a person on each change.

## Pros and Cons of the Options

### Layered metadata, resolved through one entry point

* Good, because it reuses the fetch, cache and decoder the SDK has.
* Good, because the vendors' listings correct models.dev where they know
  better, and a caller's own overrides are one more layer.
* Good, because the same type works offline, from a document the caller
  persists.
* Good, because a new source is a new layer behind the same entry point.
* Bad, because layers must be merged, and which wins is a rule per field.

### models.dev and the listings, with a fixed rule

* Good, because it is smaller now.
* Bad, because gobble-cli must merge its overrides itself, and each new
  source is a new rule inside the SDK.

### models.dev only

* Good, because it is one decoder and one rule.
* Bad, because its known errors pass through, and Anthropic's lifecycle and
  OpenAI's shutdown dates are lost.

### Each vendor's own model API only

* Good, because each value is the vendor's own.
* Bad, because OpenAI publishes no limits or prices in its API, and OpenCode
  none at all.
* Bad, because it needs a key and a request per provider before any value
  is known, so it cannot work offline.

### An embedded snapshot

* Good, because it works offline with no help from the caller.
* Bad, because it is stale from the day of each release, which 0009-MADR
  rejected.
* Bad, because it grows the module by megabytes.

### Leave metadata to gobble-cli

* Good, because the SDK does nothing.
* Bad, because it contradicts gobble-cli's choice 11, and duplicates the
  fetch and decoder the SDK already has.

## More Information

* **Asked for by:**
  * gobble-cli `docs/decisions/0005-MADR-v1-feature-scope.md`: the "Model
    catalog" row, and choice 11;
  * gobble-cli `docs/decisions/0004-MADR-go-module-architecture.md`:
    decision 6, and its facts as of `940fee0` ("context window, cost and
    modalities are not public");
  * gobble-cli `docs/reports/0014-REPORT-llmprovider-sdk-integration.md`:
    F4.

  All read in that repository on 2026-10-11.
* **Read on 2026-10-12, for the amendment:**
  * models.dev `schema.ts` (`dev` branch) and `api.json`;
  * platform.claude.com/docs/en/api/models/list and `…/retrieve`;
  * ai.google.dev/api/models;
  * docs.x.ai/developers/rest-api-reference/inference/models;
  * developers.openai.com/api/reference/resources/models;
  * openrouter.ai/docs/api/api-reference/models/get-models;
  * kilo.ai/docs/gateway/api-reference; docs.together.ai/reference/models-1;
    docs.ollama.com/api-reference/show-model-details;
  * LiteLLM's `model_prices_and_context_window.json`;
  * vercel.com/docs/ai-gateway/sdks-and-apis/rest-api;
  * agentclientprotocol.com/protocol/schema;
  * go.dev/blog/range-functions, pkg.go.dev/iter, go.dev/doc/go1.27;
  * local checkouts of OpenCode, Kilo Code, Codex, grok-build, Pi,
    gobble-cli and the ACP Go SDK, at the commits named above.
* **Official pages read on 2026-10-11:**
  * developers.openai.com/api/docs/deprecations;
  * ai.google.dev/gemini-api/docs/deprecations;
  * platform.claude.com/docs/en/about-claude/model-deprecations, and its
    release notes;
  * docs.x.ai/developers/migration/may-15-retirement;
  * each gateway's docs and live listing;
  * models.dev's `api.json` and its `schema.ts`.

  The refresh PLAN's Phase 1 re-reads each one through the providers'
  APIs before anything is built.
* **Related:**
  * [0009-MADR-use-case-aware-default-model-ranking.md](0009-MADR-use-case-aware-default-model-ranking.md)
    (models.dev, and the rejected snapshot);
  * [0007-MADR-live-catalog-model-search.md](0007-MADR-live-catalog-model-search.md)
    (static catalogs and live curation);
  * [0032-MADR-tool-strict-and-result-error-flags.md](0032-MADR-tool-strict-and-result-error-flags.md),
    the previous gobble-cli item.
* **The plans:**
  * [0033-PLAN-refresh-the-curated-catalogs.md](0033-PLAN-refresh-the-curated-catalogs.md)
    (D4, `v1.7.1`). Released as `v1.7.1` on 2026-10-11; the plan is
    complete;
  * [0033-PLAN-public-model-metadata.md](0033-PLAN-public-model-metadata.md)
    (D1 to D3 and the maintainer tool, `v1.8.0`; *amended 2026-10-11:
    `v1.9.0`*).
* **Next:** the record D6 names, for forced tool choice. *(Done 2026-10-11:
  0034-MADR, `v1.8.0`.)*
