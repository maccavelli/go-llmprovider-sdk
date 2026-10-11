---
status: accepted
date: 2026-10-11
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

### Confirmation

* Each new test is seen failing first.
* The refresh's lists, and every field D2 maps, are measured live before
  they are built.
* The G-wire listing goldens change only as the refresh PLAN records.
* apidiff: nothing for `v1.7.1`; additions only for `v1.8.0`. *(Amended
  2026-10-11: the metadata's release is `v1.9.0`.)*
* The live catalog test passes on each release commit.

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
