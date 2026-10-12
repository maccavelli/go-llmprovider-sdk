---
status: in-progress
date: 2026-10-12
associated-madr: "0033-MADR-public-model-metadata-and-current-catalogs.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Publish Model Metadata

*Rewritten 2026-10-12, before any phase ran, to follow 0033-MADR's
"Amendment (2026-10-12): the research, and the API surface" (Q9 to Q12).
The 2026-10-11 draft is this file at `0d5f668`. It is released as `v1.9.0`,
since `v1.8.0` was 0034-MADR's.*

Associated MADR: [0033-MADR-public-model-metadata-and-current-catalogs.md](0033-MADR-public-model-metadata-and-current-catalogs.md)

This is the second of the MADR's two plans: D1 to D3, as amended, and D4's
maintainer tool. It follows
[0033-PLAN-refresh-the-curated-catalogs.md](0033-PLAN-refresh-the-curated-catalogs.md),
released as `v1.7.1`.

## Goal

When it is done:

* **What a caller can read,** for any model a layer knows: its context
  window and its input and output limits; its input and output
  modalities; whether it takes tools, a temperature and structured
  output; its reasoning descriptor; its prices in USD per million tokens,
  with cache read and write, reasoning, and a list of tiers; its release
  date, status and retirement date.
* **For each value,** whether it is known (`Has`) and which layer set it
  (`SourceOf`). A known zero, such as a free price, is told apart from an
  unknown one.
* **Where from:** `catalog.LookupModel`, or `CachedModel` without waiting.
  These merge the caller's layers (`WithModelLayer`), the provider's
  listing and models.dev, field by field.
* **Offline:** from a document the caller persists with `Metadata.Encode`
  and reads back with `ParseMetadata`, passed as a layer, losslessly.
* **The tool:** a maintainer command proposes the next curated lists.

Every exported change is an addition. Nothing on the generation wire
changes.

## Scope

### In scope

* **`llmprovider/catalog`, new exported API** (0033-MADR D1 to D3, as
  amended):
  * the types `ModelInfo`, `ModelReasoning`, `ModelRates`, `ModelCost`,
    `ModelCostTier`, `Field`, `Modality`, `ModelStatus` and `Layer`, with
    their constants;
  * the methods `ModelInfo.Has`, `SourceOf` and `Known`, and
    `ModelCost.RatesFor`;
  * the functions `LookupModel`, `CachedModel`, `WithModelLayer` and
    `ParseMetadata`;
  * the methods `Metadata.Encode`, `Name`, `Model` and `Models`.
* **`llmprovider/catalog`, internal:**
  * the models.dev decoder's added fields, and the five added sections;
  * each read section's bytes, kept for `Encode`;
  * the listing decoders' added fields;
  * Ollama's `/api/show`;
  * the listing-metadata cache;
  * the merge.
* **The package comment** of `catalog`, which R41 requires and the package
  lacks.
* **The maintainer tool:**
  * `llmprovider/internal/retired`, a leaf package holding the table of
    retired ids, moved from `catalog/refresh_0033_test.go`;
  * `llmprovider/internal/catalogtool`, tested;
  * `llmprovider/internal/cmd/catalogs`, its `main`.
* **Tests:**
  * a fixture under `catalog/testdata/metadata-0033/`;
  * the decoder benchmark;
  * a measurement test behind `LLMPROVIDER_LIVE_0033`;
  * a live test of `LookupModel`.
* **Docs:**
  * `docs/architecture.md`;
  * `docs/guides/api-standards.md` R2's table, for the tool's package;
  * `docs/guides/adding-a-provider.md`;
  * `README.md`;
  * this PLAN's release notes.

### Out of scope

* **A cache-write count in `Usage`,** and with it a cost helper
  (`ModelCost.Of(Usage)`). Both are their own record (0033-MADR Q12).
* **Using `MaxOutputTokens` as a request's default output cap.** It changes
  the request on the wire (R45), so it needs its own record.
* **A forced-tool-choice field.** 0034-MADR's table remains the source.
* **Other `Field`s:** image size limits, audio prices and knowledge cutoff.
* **Correcting models.dev.** The package doc names its known errors.
* **Hugging Face's per-provider router prices.**
* **Unchanged:** `ListModels`' signature, `Catalog`, `LookupMetadata`,
  `LookupMetadataWith` and `CachedMetadataWith`.
* **ACP types.** The SDK publishes facts; gobble-cli maps them (R5).

## Rules for every phase

* **Red first.** Each new test is seen failing before the change, and its
  FAIL lines are recorded.
* **Plants** run on a scratch clone, never in the tree. Each must fail a
  test, and its FAIL lines are recorded.
* **A phase is staged only after:**
  * the gate, 22 of 22, started when the load average is below the host's
    CPU count;
  * `make pre-add-check FILES=…` on every changed Go file;
  * `python3 -B scripts/check_records.py`;
  * markdownlint;
  * the identifier scan;
  * apidiff against `v1.8.0`, additions only.
* **Billed requests** are counted and approved before they are sent.
  Phases 1 to 4 send none.
* **A deviation stops the work:** evidence, options with the recommended
  one first, and a dated entry here once the owner chooses. Where a fact
  or a decision changes, the MADR is amended too.
* **The owner commits** each phase.

## Implementation Steps

### Phase 1: measurement (live; no billed request)

Extend `llmprovider/live_0033_measure_test.go` with
`TestLive_0033MetadataSources`, behind `LLMPROVIDER_LIVE_0033`. Keys are
checked for presence only, and their values are never logged. Listings,
the models.dev document and a local Ollama are not billed.

1. **models.dev** (`models.opencode.ai/api.json`, the SDK's default URL):
   * The nine sections exist: `opencode`, `opencode-go`, `huggingface`,
     `togetherai`, `openai`, `anthropic`, `google`, `xai` and `kilo`.
     Record each one's model count.
   * For every static id of every provider but Ollama, which D1 fields its
     entry sets. An id absent from its section is listed.
   * Across the nine sections: the `reasoning_options` types seen, the
     `status` values, the `release_date` formats, how many entries carry
     `cost.tiers` or `context_over_200k`, and any field D1 does not name.
   * The nine sections' size as fetched, and the time to decode them.
2. **Listings,** two models each, recording only the shape of the fields
   in amended D2's table:
   * **Anthropic:** `max_input_tokens`, `max_tokens`, and the
     `capabilities` leaves `image_input`, `pdf_input`,
     `structured_outputs`, `thinking.types` and `effort`.
   * **Gemini:** `inputTokenLimit`, `outputTokenLimit`, `thinking`.
   * **xAI** `/v1/models`: `context_length`,
     `capabilities.reasoning_effort`, each `*_token_price`, and
     `long_context_threshold`. Compare one model's price with xAI's
     pricing page to confirm "cents per 100 million tokens".
   * **OpenAI:** `shutdown_date`.
   * **Kilo:** `context_length`, `top_provider.max_completion_tokens`,
     each `pricing` key, and `expiration_date`.
   * **Together:** `context_length` and `pricing.{input, output,
     cached_input}`. Compare one model's price with Together's pricing
     page to establish the unit.
3. **Ollama:** `/api/show` on one local model:
   `general.architecture`, `model_info["<arch>.context_length"]`, and
   `capabilities`.
4. **Record** a coverage table, provider by field.
   * A field amended D2 maps that no listing returns is a deviation.
   * A unit that does not match the docs is a deviation, and that listing
     sets no price until the owner decides.

### Phase 2: the type, the models.dev layer and the document (D1, D3)

**Files:**

* `catalog/model_info.go` (new): the types, the constants, the presence
  set, `Has`, `SourceOf`, `Known` and `RatesFor`.
* `catalog/model_metadata.go`: the decoder's added fields; the five added
  sections in `modelMetadataKey`; each read section's bytes kept as a
  `jsontext.Value`.
* `catalog/metadata_document.go` (new): `ParseMetadata`, `Encode`, `Name`,
  `Model` and `Models`.
* `catalog/doc.go` (new): the package comment (R41).
* `catalog/testdata/metadata-0033/fixture.json` (new): one entry per
  section, with every field D1 reads, a zero price, an absent cache write,
  each `reasoning_options` type, two tiers, a `context_over_200k`, each
  date form, each status, and a field D1 does not name
  (`"future_field"`).

1. **Red first** (`catalog/model_info_0033_test.go`):

   | Test | Asserts |
   | :--- | :--- |
   | `TestModelInfo_Decodes_0033` | for each fixture entry, every D1 field equals the fixture's value; `Cost` in USD per million as given |
   | `TestModelInfo_Presence_0033` | a price of `0` has `Has(FieldCostInput)` true; an absent `cache_write` has `Has(FieldCostCacheWrite)` false; an entry without `status` is `StatusActive`, set; `SourceOf` is `"models.dev"` for each set field and `""` for an unset one |
   | `TestModelInfo_Dates_0033` | `"2026-09"` is 2026-09-01 UTC; `"2026-09-28"` is that day; `"soon"` leaves `FieldReleased` unset |
   | `TestModelInfo_Reasoning_0033` | `toggle` sets `Toggle`; `effort` values keep their order, with `null` as `none`; `budget_tokens` sets `BudgetMin` and `BudgetMax`; `reasoning: false` leaves the descriptor unset apart from `Supported` false, set |
   | `TestModelCost_Tiers_0033` | `tiers` decode ascending by `Above`; a lone `context_over_200k` is the tier `Above` 200000; a tier's omitted cache rate is the base rate |
   | `TestModelCost_RatesFor_0033` | with tiers at 200000 and 1000000: 0, 200000 → base; 200001 → the first tier; 1000001 → the second |
   | `TestModelInfo_Known_0033` | `Known(FieldContextWindow)` sets `Has` on the copy, not the receiver; `SourceOf` names the layer that returned it |
   | `TestMetadata_ModelsCopies_0033` | `Models` is sorted by id; changing a returned `Input`, `Efforts` or `Tiers` does not change the next call's (R29) |
   | `TestMetadata_EncodeLossless_0033` | `Encode`, then `ParseMetadata`, gives equal `Model` results for every entry; the encoded bytes of each section equal those read, `"future_field"` included |
   | `TestParseMetadata_Bounds_0033` | it reads only its reader (a test transport fails any request); a body over the limit and one that is not a document are refused, with an error at most 512 bytes long (R35) |
   | `TestModelMetadataKey_0033` | `openai`, `claude`, `gemini`, `grok` and `kilo` map to `openai`, `anthropic`, `google`, `xai` and `kilo`; `ollama` maps to nothing |

   `TestDecodeModelMetadata_MatchesUnmarshal` is updated to the nine
   sections, and recorded as an expectation the MADR changes.
2. **Change:** the files above.
3. **Benchmark:** `BenchmarkDecodeModelMetadata`, before and after, with
   `-count=10`. A slowdown over 25% is a deviation.
4. **Plants**, each failing a test above:
   * the decoder drops `limit.output`;
   * a zero price is marked unknown;
   * `Models` returns the cached slice;
   * `Encode` drops a section;
   * `Encode` re-encodes from the struct, losing `"future_field"`;
   * `RatesFor` applies a tier at exactly `Above`;
   * `Known` changes its receiver.
5. Gate; stage.

### Phase 3: the listings, the cache and the resolver (D2, D3)

**Files:**

* `catalog/listing_metadata.go` (new): each listing's fields as a
  `ModelInfo`, marked with `Known`, sourced `listing:<provider>`; Ollama's
  `/api/show`.
* `catalog/discovery.go`: the listing page structs gain amended D2's
  fields. No request changes: the same replies are decoded further.
* `catalog/listing_cache.go` (new): the listing-metadata cache.
  * It lives 10 minutes.
  * It is keyed by provider, base URL and a SHA-256 credential
    fingerprint, as `transport.ProbeKey` is.
  * It is filled by `List` and by `LookupModel`.
* `catalog/lookup.go` (new): `Layer`, `WithModelLayer`, the merge,
  `LookupModel` and `CachedModel`.

1. **Red first** (`catalog/lookup_0033_test.go`), with canned replies from
   an `httptest` server and a fixture document:

   | Test | Asserts |
   | :--- | :--- |
   | `TestLookupModel_Listings_0033` | for each of Anthropic, Gemini, xAI, Kilo, Together, OpenAI and Ollama: every field amended D2 gives it is the listing's value, sourced `listing:<provider>`; every other field is models.dev's |
   | `TestLookupModel_AnthropicLifecycle_0033` | a `lifecycle` of `deprecated` beside a models.dev entry without `status` leaves `StatusActive` from models.dev ("sources that lag") |
   | `TestLookupModel_Units_0033` | xAI's `prompt_text_token_price` 20000 is 2.0 USD per million; its `long_context_threshold` 200000 is the tier `Above` 199999; Kilo's `"0.000002"` is 2.0; Kilo's `"-1"` leaves the price unset; Together's `input` 0.3 is 0.3 (Phase 1). *(Deviation, 2026-10-12: Kilo's `input_cache_read`, `input_cache_write` and `internal_reasoning` set `CacheRead`, `CacheWrite` and `Reasoning`; `discount` changes nothing.)* |
   | `TestLookupModel_Retires_0033` | OpenAI's `shutdown_date` and Kilo's `expiration_date` set `Retires` |
   | `TestLookupModel_CallerLayers_0033` | a caller layer beats the listing and models.dev; of two `WithModelLayer`s, the later wins; a field no layer sets stays unset; a caller's `FieldCostTiers` replaces the list whole; a caller's known zero price beats a listing's non-zero |
   | `TestLookupModel_Unknown_0033` | a model no layer knows returns no error and no field set; a listing that fails returns models.dev's fields and an error naming the listing |
   | `TestLookupModel_Offline_0033` | with `WithoutModelMetadata`, a parsed document as a layer, and a nil source, a counting transport sees zero requests, and the fields are the document's |
   | `TestCachedModel_NeverWaits_0033` | cold: it returns false within 50 ms while the listing server blocks, and a later `CachedModel` returns the value once the fetch ends; warm: it returns the value without a request |
   | `TestListingCache_Keys_0033` | a second `LookupModel` within 10 minutes sends no listing request; another credential or base URL sends one; after 10 minutes (a fake clock) one is sent |
   | `TestLookupModel_Bounds_0033` | a cancelled context returns `context.Canceled` (R40); a listing error body of 100 KiB yields an error of at most 512 bytes, with no credential (R35); an unknown provider is `ErrInvalidProvider` |
   | `TestWithModelLayer_Scoped_0033` | `WithModelLayer` given to `providers.New` for a built-in provider is accepted; for a provider outside `builtinIDs` it is refused (R18) |

2. **Change:** the files above.
3. **Goldens:** the G-wire listing and generation goldens are unchanged.
   Any change is a deviation.
4. **Plants**, each failing a test above:
   * models.dev wins over a listing;
   * caller layers in the wrong order;
   * xAI's unit not converted;
   * a `-1` price kept;
   * `CachedModel` waits;
   * the cache ignores the credential;
   * tiers merged element by element;
   * Anthropic's `lifecycle` setting `Status`.
5. Gate; stage.

### Phase 4: the maintainer tool (D4)

**Files:**

* `llmprovider/internal/retired` (new): `IDs`, the table of retired ids by
  provider, each with why, moved from `catalog/refresh_0033_test.go`.
  * It imports only `llmprovider`.
  * `catalog`'s in-package test can therefore read it without an import
    cycle, which a table inside `catalogtool` would cause, since
    `catalogtool` imports `catalog`.
* `llmprovider/internal/catalogtool` (new): `Propose(static
  map[llmprovider.ProviderID][]string, sources …) []Proposal`. It reads
  `retired.IDs`.
* `llmprovider/internal/cmd/catalogs/main.go` (new):
  * It reads models.dev, and each listing whose key variable is set.
  * It prints the proposals.
  * It edits nothing, and never prints a key.
* `docs/guides/api-standards.md`, R2's table: rows for
  `internal/retired` (`llmprovider` only), `internal/catalogtool`
  (`llmprovider`, `catalog`, `internal/retired`) and
  `internal/cmd/catalogs`.

1. **Red first** (`internal/catalogtool/propose_0033_test.go`), with a
   canned document and listing:
   * a retired id in a static list is proposed for removal, with its
     reason;
   * a current id missing from it is proposed for addition, with its
     status, release date and input price;
   * a list already current gets no proposal;
   * a `deprecated` model is never proposed for addition;
   * the output is the same for the same input, sorted by provider, then
     by id.
2. **Change:** the files above. `TestStatic_NoRetiredIDs_0033` reads
   `retired.IDs`. Its FAIL text is unchanged.
3. **Plants:**
   * a proposal that ignores `Status`;
   * the unit test reading an empty table, which must fail on a planted
     retired id in a static list.
4. **Coverage:** `internal/catalogtool` keeps the default 80% floor.
   `internal/retired` is data, read by both tests. `internal/cmd/catalogs`
   holds only `main`.
5. Gate; stage.

### Phase 5: documentation and release

1. **Docs:**
   * `docs/architecture.md`:
     * the catalog row's exports;
     * a paragraph on the layers, the merge by `Field`, the listing cache,
       and the units;
     * the tool's package.
   * `docs/guides/adding-a-provider.md`: a new provider names its
     models.dev section, and the fields its listing sets.
   * `catalog`'s package comment: the known errors of models.dev from
     0033-MADR's context.
   * `README.md` Status, `v1.9.0`; release notes in this PLAN.
2. **`TestLive_LookupModel_0033`** (new, in the live suites): for one
   static id of each keyed provider, `LookupModel` returns `ContextWindow`
   or `MaxInputTokens`, set and above zero. It sends listing requests
   only.
3. **The live suites** on the release commit. They are billed: the count
   is given before they run, as before.
4. **The owner** commits, pushes, and tags `v1.9.0`, after the disclosure
   guard.
5. **After the tag:**
   * CI on `main` and on the tag;
   * the proxy;
   * apidiff `v1.8.0` to `v1.9.0`: the additions of D1 to D3 only;
   * a scratch consumer that reads a `ModelInfo` offline, through
     `ParseMetadata` and `WithModelLayer`, with a counting transport
     seeing no request, built for six platforms;
   * a scratch copy of gobble-cli, moved to `v1.9.0`, that fills a Pi
     `Model`'s `contextWindow`, `maxTokens` and `cost` from `ModelInfo`
     and passes its tests and lint.
6. This PLAN `complete`; the index updated.

## Verification

* **V1. Red, then green,** with every plant's FAIL line, in each phase's
  record.
* **V2. The gate** at each phase. The generation and listing goldens are
  unchanged.
* **V3. Measurement:** Phase 1's coverage table, recorded before Phase 2,
  with every unit confirmed.
* **V4. Benchmark:** within 25%.
* **V5. API:** apidiff `v1.8.0` to `v1.9.0`, additions only, and only
  those of amended D1 to D3.
* **V6. Live:** `TestLive_LookupModel_0033` and the live suites on the
  release commit.
* **V7. Offline:** the scratch consumer's counting transport sees no
  request.

## Rollout and Rollback

* **Rollout:**
  * this rewrite and the MADR's amendment;
  * one commit per phase, five in all;
  * then `v1.9.0`.
* **Rollback, before the tag:** each phase reverts alone, newest first.
* **Rollback, after the tag:** fix forward in `v1.9.1`. A caller that does
  not use the new API is unaffected.

## Execution Record

### Phase 0: records (2026-10-12)

* 0033-MADR gained "Amendment (2026-10-12): the research, and the API
  surface", and this PLAN was rewritten to follow it.
* The owner answered Q9 to Q12 with the recommended option of each.
* The owner committed both as `10c1be3`, accepted the amendment, and
  approved this PLAN.

### Phase 1: measurement (2026-10-12)

`TestLive_0033MetadataSources`, in the new
`llmprovider/live_0033_metadata_measure_test.go`, behind
`LLMPROVIDER_LIVE_0033`.

* Each key was checked for presence only. Only field names, shapes and
  public prices were logged.
* No request was billed.
* Result: exit 0, all 8 subtests passing, in 4.3 s.

**Step 1, models.dev** (`models.opencode.ai/api.json`): HTTP 200, 5,388,868
bytes, 226 sections.

* **The nine sections are 508,550 bytes,** decoded generically in 4 ms.

  | Section | Models | `reasoning_options` | `status` | `release_date` | `tiers` | `context_over_200k` |
  | :--- | ---: | :--- | :--- | :--- | ---: | ---: |
  | `anthropic` | 17 | budget 8, effort 13, toggle 2 | none | `YYYY-MM-DD` 17 | 1 | 0 |
  | `google` | 39 | budget 3, effort 17, toggle 4 | beta 1, deprecated 1 | `YYYY-MM-DD` 39 | 6 | 6 |
  | `huggingface` | 81 | effort 20, toggle 3 | none | `YYYY-MM` 3, `YYYY-MM-DD` 78 | 0 | 0 |
  | `kilo` | 386 | effort 264 | beta 1 | `YYYY-MM` 5, `YYYY-MM-DD` 381 | 2 | 2 |
  | `openai` | 53 | effort 33 | deprecated 12 | `YYYY-MM-DD` 53 | 13 | 13 |
  | `opencode` | 120 | budget 11, effort 71, toggle 24 | deprecated 36 | `YYYY-MM-DD` 120 | 17 | 16 |
  | `opencode-go` | 36 | budget 5, effort 23, toggle 10 | deprecated 5 | `YYYY-MM-DD` 36 | 9 | 8 |
  | `togetherai` | 29 | effort 9, toggle 10 | deprecated 11 | `YYYY-MM-DD` 29 | 0 | 0 |
  | `xai` | 13 | effort 5 | none | `YYYY-MM-DD` 13 | 8 | 8 |

  * No `alpha` status was seen, and no section marks a model `retired`.
  * `YYYY-MM` dates occur, so D1's parse of both forms is needed.
  * Most priced tiers carry both `tiers` and `context_over_200k`. Where
    both are present, `tiers` is read.
  * Keys D1 does not read: `canonical_model_id`, `description` and
    `experimental`. `Encode` keeps them.
* **Every static id is in its section.** The D1 fields each one lacks:
  * `limit.input` is set only on the OpenAI models but `gpt-4.1-mini`, and
    on OpenCode Go's `gpt-6-luna` and `hy3`;
  * `cost.reasoning` is set on none;
  * `status` is absent on all, which reads as active;
  * `cost.cache_write` is set on:
    * the Claude models;
    * the OpenAI models but `gpt-4.1-mini`;
    * Kilo's `google/gemini-3.8-flash` and `google/gemini-3.6-flash`;
    * `qwen3.8-flash` on both OpenCode gateways;
    * OpenCode Go's `gpt-6-luna`;
  * `cost.tiers` is set on:
    * `claude-haiku-5-5`;
    * the Grok models;
    * the OpenAI models but `gpt-4.1-mini`;
    * OpenCode Go's `gpt-6-luna`;
  * `structured_output` is absent on 8 gateway models: three on Hugging
    Face, three on OpenCode Go, two on Together;
  * `family` is absent on two Hugging Face models.

**Step 2, the listings:**

| Listing | Fields seen (two models each) |
| :--- | :--- |
| Anthropic | `max_input_tokens` 1000000, `max_tokens` 128000; `capabilities`: `effort` `low` to `max` with `xhigh`; `image_input`, `pdf_input`, `structured_outputs` supported; `thinking.types` `adaptive` and `disabled` supported, `enabled` not |
| Gemini | `inputTokenLimit` 1048576, `outputTokenLimit` 65536, `thinking` true |
| xAI | `context_length` 500000; `capabilities.reasoning_effort` `low`, `medium`, `high`, `xhigh`, default `high`; `long_context_threshold` 200000; the `prompt`, `cached_prompt` and `completion` text prices and their `_long_context` forms; `prompt_image_token_price` |
| OpenAI | 133 models, 54 with a `shutdown_date`; the static ids' is `null` |
| Kilo | 390 models; `context_length`, `top_provider.max_completion_tokens`, `expiration_date` (`null`); `pricing` keys `prompt` and `completion` (390), `discount` (272), `input_cache_read` (245), `web_search` (130), `input_cache_write` (76), `internal_reasoning` (32), `image` (31), `request` (12); 20 prices of `"-1"` |
| Together | `context_length`; `pricing` `input`, `output`, `cached_input`, and zeros for `hourly`, `base`, `finetune`, `image`, `video` and others |

**The units,** compared with models.dev's USD per million:

| Model | Listing | As USD per million | models.dev |
| :--- | :--- | :--- | :--- |
| xAI `grok-4.6` | 20000, 60000, 5000 | 2, 6, 0.5 | 2, 6, 0.5 |
| xAI `grok-4.6`, long context | 40000, 120000, 10000 | 4, 12, 1 | the tier at 200000: 4, 12, 1 |
| Kilo `deepseek/deepseek-v4.1-flash` | `"0.000000300000"`, `"0.000001200000"`, `"0.000000006000"` | 0.3, 1.2, 0.006 | 0.3, 1.2, 0.006 |
| Together `deepseek-ai/DeepSeek-V4.1-Flash` | 0.3, 1.2, 0.006 | the same | 0.3, 1.2, 0.006 |

* xAI's unit, cents per 100 million tokens, is confirmed.
* Kilo's per-token strings are confirmed.
* Together's unit is USD per million.
* Together gives `zai-org/GLM-5.3` a `context_length` of 1048575, where
  models.dev has 1048576. D2 takes the listing's.

**Step 3, Ollama** (`llama3.2:latest`, local):

* `general.architecture` is `llama`;
* `llama.context_length` is 131072;
* `capabilities` is `[completion tools]`;
* `details.context_length` is absent, so the window comes from
  `model_info` only.

**Step 4, coverage:**

* Every field amended D2 maps was returned by its listing, so there is no
  missing-field deviation.
* Every unit matches.

**Deviation (2026-10-12): Kilo's other prices.** Kilo's listing carries
`input_cache_read`, `input_cache_write` and `internal_reasoning`, which
amended D2's Kilo row did not map. The owner chose to set them when
present, without applying `discount`:

* 0033-MADR's D2 table is annotated;
* Phase 3's `TestLookupModel_Units_0033` row gains them.

**The gate.** The scratchpad's gate script had been pruned. It was rebuilt
from its one write and two edits in the session transcript, and proved on
a scratch clone holding two plants before it was trusted:

* a `vet` error in `catalog`:
  `plant_vet.go:5:46: fmt.Sprintf format %d reads arg #1, but call has 0 args`;
* this PLAN's index row marked `complete`:
  `docs/README.md: decisions/0033-PLAN-public-model-metadata.md indexed as 'complete'; the file says 'in-progress'`.

Thirteen checks failed on the planted clone:

* the six `vet` variants;
* `pre-add-check`;
* `test-race-cover` and `test-shuffle`;
* `lint`;
* `coverage-check` and `gate-selftest`;
* `g-wire-stable`;
* `records-check`.

Then, on this phase's tree, all 22 passed, started at a load average of
9.4. The owner then directed that every script be Python, and reusable ones
kept in `tools/`. 0035-MADR records that, so the load wait was rewritten
in Python for this run.

**Pre-add:** 1 file clean. **API:** unchanged; only a live-tagged test file
was added.
