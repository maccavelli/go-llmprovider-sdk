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
   *(Deviation, 2026-10-12: the eager decode was 35.7% slower; resolved by
   decoding each section lazily. See Phase 2's record.)*
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

### Phase 2: the type, the models.dev layer and the document (2026-10-12)

**Change:**

* **`catalog/model_info.go`** (new):
  * `ModelInfo`, `ModelReasoning`, `ModelRates`, `ModelCost` and
    `ModelCostTier`;
  * `Modality`, `ModelStatus` and `Field`, with their constants;
  * presence as a bitset and a fixed array of source names, copied with the
    value, so `Known` can never reach its receiver;
  * `Has`, `SourceOf`, `Known` and `RatesFor`.
* **`catalog/model_metadata.go`:**
  * the entry gains `limit.input` and `limit.output`, `modalities`,
    `tool_call`, `temperature`, `structured_output`, the cache and
    reasoning prices, `tiers`, `context_over_200k`, and the budget range of
    `reasoning_options`;
  * the five sections are added to `modelMetadataKey`;
  * the document keeps each section's bytes, and decodes it when first read
    (the deviation below).
* **`catalog/metadata_document.go`** (new): `ParseMetadata`, `Encode`,
  `Name`, `Model` and `Models`, and the conversion of an entry to a
  `ModelInfo`.
* **`catalog/doc.go`** (new): the package comment, which R41 requires and
  the package lacked, naming models.dev's known errors.
* **`docs/architecture.md`:** the catalog row lists the new exports, in the
  same change, as R43 requires.

**Red first:**

```text
llmprovider/catalog/model_info_0033_test.go:32:12: undefined: ParseMetadata
llmprovider/catalog/model_info_0033_test.go:40:79: undefined: ModelInfo
llmprovider/catalog/model_info_0033_test.go:42:16: m.Model undefined (type Metadata has no field or method Model)
```

**Tests:**

* The 11 the step lists, and `TestLazySection_DecodesOnce_0033`.
* `TestDecodeModelMetadata_MatchesUnmarshal` covers the nine sections, as
  the step says. It still finds json/v2 and `json.Unmarshal` equal,
  repeated keys included.
* Test plumbing only: the document is no longer a map, so the tests that
  indexed it read it through the test helper `all()`. Their expectations
  are unchanged.

**One additive effect:** `Metadata.ReasoningEfforts`, `InterleavedField`
and `NPM` now also answer for the five added providers, where they gave
nothing. Only OpenCode calls them, with its own ids.

**Green:**

* `go test -race -count=1 ./...` passes.
* `golangci-lint` first reported `func ModelInfo.clone is unused`. It was
  removed until Phase 3's merge needs it, and lint then reported 0 issues.

**Deviation (2026-10-12): the decode's cost.** The eager decode was 35.7%
slower than before on `BenchmarkDecodeModelMetadata`, past the 25% bound.
Both were run ten times, alternated, under the same load. A split, on
scratch clones, showed where the cost came from:

| Decoder | Generated 5 MB document | The real models.dev document |
| :--- | :--- | :--- |
| before | 21.27 ms, 4.71 MB | 8.80 ms, 0.12 MB |
| eager, with the bytes kept | 28.86 ms (+35.7%), 13.27 MB | 10.58 ms (+20.2%), 1.62 MB |
| eager, without the bytes | 24.52 ms (+15.3%), 7.27 MB | 10.08 ms (+14.5%), 0.56 MB |

About 15 points were D1's added fields. About 20 were the bytes kept for a
lossless `Encode`, which read each section twice.

The owner chose to decode each section lazily:

* the fetch reads each section once and keeps its bytes;
* a section is decoded the first time it is read, once, under a
  `sync.Once`;
* `ParseMetadata` decodes every section at once, so a malformed one is
  reported.

0033-MADR's D3 is annotated. `BenchmarkDecodeModelMetadataAllSections` is
added, so the deferred cost stays measured.

**Benchmark,** before (HEAD) and after (lazy), ten runs each, alternated,
at a load average near 6:

| Measure | Before | After | Change |
| :--- | :--- | :--- | :--- |
| generated document, the fetch (`BenchmarkDecodeModelMetadata`) | 21.54 ms, 4.71 MB | 8.30 ms, 6.00 MB | -61.5% |
| generated document, every section decoded | 21.54 ms, 4.71 MB | 29.16 ms, 13.27 MB | +35.4% |
| the real document, the fetch | 8.94 ms, 0.12 MB | 8.39 ms, 1.06 MB | -6.2% |
| the real document, every section decoded | 8.94 ms, 0.12 MB | 10.72 ms, 1.62 MB | +19.9% |

The step's benchmark is 61.5% faster. Decoding every section costs what the
eager decode did. Only `ParseMetadata` and `Encode` do that; a fetch, a
ranking and a `Model` call decode only the sections they read.

**Plants,** through `tools/plant.py`, on `./llmprovider/catalog/` with `-run
_0033$`:

| Plant | FAIL lines |
| :--- | :--- |
| the decoder drops `limit.output` | `TestModelInfo_Decodes_0033`: `claude-x:` with `MaxOutputTokens:0` |
| a zero price marked unknown | `a price of 0: Has false, Input 0; want known, 0`; `SourceOf(FieldCostInput) = "", want models.dev` |
| `Models` returns the cached slice | `a change to Models' result reached the next call: {… Input:[changed image pdf] …}` |
| `Encode` drops a section | `claude: 2 models before, 0 after, or they differ`; `section anthropic: the encoded bytes differ from those read`; `"future_field" was lost` |
| `Encode` re-encodes from the struct | `section <each of the nine>: the encoded bytes differ from those read`; `"future_field" was lost` |
| `RatesFor` applies a tier at exactly `Above` | `RatesFor(200000) = {Input:2 …}, want {Input:1 …}`; `RatesFor(1000000) = {Input:3 …}, want {Input:2 …}` |
| `Known` changes its receiver | `Known: copy true, receiver true; want true, false` |
| the fetch decodes every section at once | `the section was decoded before it was read` |
| `ParseMetadata` skips the up-front decode | `ParseMetadata accepted a section that does not decode` |

**API:** `tools/apidiff.py v1.8.0` lists 46 additions and nothing else. They
are the types, constants, functions and methods of amended D1 and D3 that
this phase builds.

**Gate:** `tools/gate.py`, 23 of 23, started at a load average of 5.1.

### Phase 3: the listings, the cache and the resolver (2026-10-12)

**Change:**

* **`catalog/discovery.go`:**
  * the listing page structs gain amended D2's fields: Gemini's, Claude's,
    the OpenAI-shaped `dataModel` (OpenAI and xAI), Together's and Kilo's;
  * each fetcher hands every entry it decodes to an optional sink, so the
    same replies are decoded further, with no request added;
  * `List` is split into `listAndRecord` (the listing, with a refused
    credential renewed once) and `listOnce`, which records a successful
    listing's metadata in the listing cache. `List` and `LookupModel` share
    them.
* **`catalog/listing_metadata.go`** (new):
  * each listing's fields as a `ModelInfo`, sourced `listing:<provider>`;
  * xAI's prices are multiplied by 10⁻⁴, from cents per 100 million, and
    its `long_context_threshold` *t* becomes the tier `Above` *t* − 1;
  * Kilo's per-token strings are multiplied by 10⁶, and `-1` or anything
    unparseable is left unset;
  * Ollama's `POST /api/show` is read for one model.

  Anthropic's `lifecycle` and `retires_at` are not read.
* **`catalog/listing_cache.go`** (new):
  * a cache of 10 minutes, keyed by provider, base URL and
    `transport.Fingerprint` of the credential;
  * one background fetch per key, detached and bounded as a listing is,
    whose failure goes only to the caller's logger (R31).
* **`catalog/lookup.go`** (new):
  * `Layer`, `WithModelLayer` (scoped to `builtinIDs`), `LookupModel` and
    `CachedModel`;
  * the merge, by `Field`: models.dev, then the listing, then each caller
    layer in order, with slices copied.
* **`catalog/config.go`:** the layers, and the listing sink.
* **`catalog/model_metadata.go`:** `CachedMetadataWith`'s body becomes
  `cachedMetadata`, which `CachedModel` shares.
* **`docs/architecture.md`:** the catalog row lists `Layer`, `LookupModel`,
  `CachedModel` and `WithModelLayer` (R43).

**Two facts measured before the tests were written** (unbilled):

* OpenAI's non-null `shutdown_date` is `YYYY-MM-DD` (54 of 54).
* Kilo's non-null `expiration_date` is `YYYY-MM-DD` (16 of 16), and its
  `top_provider.max_completion_tokens` is an integer (382) or `null` (8).

**Red first:**

```text
llmprovider/catalog/lookup_0033_test.go:100:2: undefined: resetListingCache
llmprovider/catalog/lookup_0033_test.go:198:9: undefined: LookupModel
llmprovider/catalog/lookup_0033_test.go:296:42: undefined: WithModelLayer
```

Two of the test's own names first collided with other test files
(`kiloListing`, `countingTransport`). They were renamed with a `0033`
suffix before the test ran.

**Tests:**

* `catalog/lookup_0033_test.go`: the step's eleven tests but the last.
* `catalog/lookup_scope_0033_test.go` (`package catalog_test`):
  `TestWithModelLayer_Scoped_0033`.
  * It needs `providers.New`, which imports `catalog`, so it is an
    external test.
  * It checks the refusal with `ResolveOptions` on an id outside
    `builtinIDs`, which is what `New` calls.

**Green:**

* `go test -race -count=1 ./...` passes.
* No golden file changed (step 3).
* `golangci-lint` first reported `errcheck` on the background fetch's
  discarded error. The error is now logged to the caller's logger, as R31
  allows, and lint reported 0 issues.

**Plants,** through `tools/plant.py`, on `./llmprovider/catalog/` with `-run
_0033$`:

| Plant | FAIL lines |
| :--- | :--- |
| models.dev wins over a listing | `field 4: source "models.dev", want listing:claude`, and the same for Gemini and xAI |
| caller layers in the wrong order | `ContextWindow 1 from "first", want 2 from the later layer` |
| xAI's unit not converted | `xAI: <nil>, cost {ModelRates:{Input:20000 Output:60000 CacheRead:5000 …` |
| a `-1` price kept | `Kilo: <nil>, Input 2, Output -1e+06 from "listing:kilo"; want 2.0, and the -1 output left to models.dev's 0.6` |
| `CachedModel` waits | `cold: ok false after 10.001600625s; want false within 50 ms` |
| the cache ignores the credential | `another credential: 1 listing requests, want 2`; `after ten minutes: 2 listing requests, want 3` |
| tiers merged element by element | `tiers [{Above:199999 …} {Above:100 …}], want the caller's list whole` |
| Anthropic's lifecycle sets `Status` | `Status "deprecated" from "listing:claude", …; want active from models.dev, no Retires` |

**API:** `tools/apidiff.py v1.8.0` lists 50 additions and nothing else:
Phase 2's 46, and `Layer`, `LookupModel`, `CachedModel` and
`WithModelLayer`.

**Gate:** `tools/gate.py`, 23 of 23, started at a load average of 6.7.
