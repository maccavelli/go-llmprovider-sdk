---
status: complete
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
   *(Deviation, 2026-10-12: `go test -cover` reports an untested package as
   0.0%, so `main` calls a tested `run`; see Phase 4's record.)*
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
   *(Deviation, 2026-10-12: one live test drifted with OpenCode Zen's
   wording; see Phase 5's record.)*
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

### Phase 4: the maintainer tool (2026-10-12)

**Change:**

* **`llmprovider/internal/retired`** (new): `IDs`, the table moved from
  `catalog/refresh_0033_test.go`, which now reads it. It imports only
  `llmprovider`.
* **`llmprovider/internal/catalogtool`** (new): `Propose(static, Sources)
  []Proposal`, sorted by provider, then id, and `Proposal.String`, one
  tab-separated line.
  * **Removed:** an id in `retired.IDs`; one models.dev marks deprecated
    or retired; or, where the listing was read, one it lacks.
  * **Added:** a model of the listing, or of models.dev where none was
    read, that is not retired or deprecated, not known to lack tools, and
    released no earlier than the list's newest model. Its reason gives its
    status, release date and input price.
* **`llmprovider/internal/cmd/catalogs`** (new): `main` calls
  `run(args, stdout, stderr, getenv, fetcher)`.
  * It lists each provider whose `ProviderEnvVars` variable is set, and
    says on stderr which listings it skipped.
  * `-doc FILE` reads a document in place of a fetch.
  * It prints the proposals and a summary, edits nothing, and prints no
    key.
* **`docs/guides/api-standards.md`:** R2's table has rows for the three
  packages.

**Deviation (2026-10-12): the command's coverage.** Step 4 said
`internal/cmd/catalogs` "holds only `main`". That assumed a package without
tests needs no coverage. In fact `go test -cover` reports such a package as
0.0%, and `coverage-check` holds it to the default 80%, which is why
`internal/wirecase` has its own measured-by line. A floor of 0 would loosen
a check, so it was not offered. The owner chose a one-line `main` calling a
tested `run`, with a fake fetcher, no network and no key.

**Red first:**

```text
llmprovider/internal/catalogtool/propose_0033_test.go:37:9: undefined: Propose
llmprovider/internal/cmd/catalogs/main_0033_test.go:54:8: undefined: run
```

Pinning the "added" rule changed one of the test's own inputs before green.
`TestPropose_Deterministic_0033`'s Claude list holds `claude-cur` beside
`claude-haiku-4-5`. With no dated model in the list there is nothing to
compare against, and every current model would be proposed. A real menu
always has dated models.

**Green:**

| Package | Coverage | Floor |
| :--- | :--- | :--- |
| `internal/catalogtool` | 96.3% | 80% |
| `internal/cmd/catalogs` | 89.8% | 80% |
| `internal/retired` | no statements | none |

The catalog tests still pass, reading the moved table.

`golangci-lint` first reported seven findings in the command:

* `errcheck` on its five unchecked writes, and on the document's `Close`;
* `gosec` G304 on opening the `-doc` path, which is the maintainer's own
  argument.

The writes now go through a writer that keeps the first error, and a
failed write exits 2. The document is read with
`os.ReadFile(filepath.Clean(path))`. Lint then reported 0 issues.

**The gate's first run failed** 5 of 23 checks (`pre-add-check`,
`test-race-cover`, `test-shuffle`, `coverage-check`, `gate-selftest`), all
on one cause, `TestNoAmbientState`:

```text
llmprovider/internal/cmd/catalogs/main.go:30: os.Getenv outside an exported ...FromEnv function
```

R30 allows the environment to be read only in an exported `…FromEnv`
function. The per-package runs before it had not included
`internal/ambientcheck`, which only the whole suite runs.

* **The fix:** `KeysFromEnv` reads each provider's key variable, and `run`
  takes the map in place of a `getenv`.
* **Coverage:** it fell to 79.7%, below the floor. Two tests were added:
  * `TestKeysFromEnv_0033` blanks every provider variable with
    `t.Setenv` first, so it reads no key the host holds;
  * `TestRun_WriteFails_0033` covers the failed-write path.

  Coverage then reached 89.8%.

**The tool, run once on 2026-10-12** (`go run ./llmprovider/internal/cmd/catalogs`,
listings only, not billed):

* Every key was set, so no listing was skipped. The identifier scan of
  stdout and stderr found nothing.
* It made 49 proposals: 35 additions on Kilo, 11 on OpenCode Zen and 2 on
  OpenCode Go, and one removal. The vendors' lists drew none.
* **Found, outside this plan's scope:** `opencode-zen remove
  deepseek-v4-flash not in the listing`. Zen's listing no longer offers a
  model its static list names. A catalog refresh record decides it.

**Plants,** through `tools/plant.py`:

| Plant | FAIL lines |
| :--- | :--- |
| a proposal that ignores `Status` | every `catalogtool` test; for one, `proposals [claude add claude-dep status deprecated, …], want none` |
| the tool reading an empty retired table | `TestPropose_RemovesARetiredID_0033`: `proposals [claude add claude-new …], want first claude remove claude-haiku-4-5 retired: retirement floor 2026-10-15` |
| a retired id in a static list | `TestStatic_NoRetiredIDs_0033`: `claude: static catalog lists "claude-haiku-4-5", which is retirement floor 2026-10-15`, the moved test's FAIL text unchanged |

The step's second plant reads "the unit test reading an empty table". It is
covered twice. The tool reading an empty table fails its own test. The unit
test, reading the moved table, still fails on a retired id planted in a
static list.

**Gate:** `tools/gate.py`, rerun after the R30 fix: 23 of 23. It waited at a
load average of 56.2, and started at 9.6. **API:** unchanged from Phase 3;
the three packages are internal.

### Phase 5: documentation and release (2026-10-12)

**Step 1,** on `d341cc8` (Phase 4):

* **`docs/architecture.md`:**
  * the tree and the package table name `internal/retired`,
    `internal/catalogtool` and `internal/cmd/catalogs`;
  * a "Model metadata" section gives the three kinds of layer, the merge by
    `Field`, lazy decoding, the listing cache and the units.

  The catalog row's exports were added in Phases 2 and 3, with their API
  (R43).
* **`docs/guides/adding-a-provider.md`:** a new provider names its
  models.dev section, decodes its listing's fields into a converter marked
  with `Known`, and joins `listingLayers`, with a row in
  `TestLookupModel_Listings_0033`.
* **`catalog`'s package comment** has named models.dev's known errors since
  Phase 2.
* **`README.md`:** Status names `v1.9.0`, with a `v1.9.0` bullet.
* **`TestLive_LookupModel_0033`** (new, `llmprovider/live_lookup_0033_test.go`,
  in the live suites). It needs no switch, since it is not billed. It was run
  once before the release commit, and all nine keyed providers passed:

  | Provider | Model | Window | Input limit | Output cap | Input price |
  | :--- | :--- | :--- | :--- | :--- | :--- |
  | claude | `claude-haiku-5-5` | 1000000, models.dev | 1000000, listing | 128000, listing | 0.1, models.dev |
  | gemini | `gemini-3.5-flash-lite` | 1048576, models.dev | 1048576, listing | 65536, listing | 0.3, models.dev |
  | grok | `grok-4.6` | 500000, listing | — | 500000, models.dev | 2, listing |
  | huggingface | `deepseek-ai/DeepSeek-V4-Flash-0731` | 1048576, models.dev | — | 384000, models.dev | 0.14, models.dev |
  | kilo | `deepseek/deepseek-v4.1-flash` | 1048576, listing | — | 393216, listing | 0.3, listing |
  | openai | `gpt-6-luna` | 1050000, models.dev | 922000, models.dev | 128000, models.dev | 0.1, models.dev |
  | opencode-go | `mimo-v2.6-flash` | 1048576, models.dev | — | 131072, models.dev | 0.14, models.dev |
  | opencode-zen | `deepseek-v4.1-flash` | 1000000, models.dev | — | 384000, models.dev | 0.3, models.dev |
  | together | `deepseek-ai/DeepSeek-V4.1-Flash` | 1048576, listing | — | 384000, models.dev | 0.3, listing |

  Each field came from the layer amended D2 gives it.
* **Gate:** `tools/gate.py`, 23 of 23, started at a load average of 4.3.

#### Release notes: `v1.9.0`

`v1.9.0` publishes model metadata (0033-MADR D1 to D3, as amended on
2026-10-12). The exported API only grows.

**`catalog.ModelInfo`.** One model's facts:

* its context window, input and output limits;
* its input and output modalities;
* whether it takes tools, a temperature and structured output;
* its reasoning controls (`ModelReasoning`: supported, toggle, effort
  levels, budget range);
* its prices in USD per million tokens (`ModelRates`: input, output, cache
  read and write, reasoning), with `Tiers` for longer inputs;
* its release date, status and retirement date.

**Known, and from where.**

* `Has(Field)` says whether a layer set a field, so a known zero, such as
  a free price, is told apart from an unknown value.
* `SourceOf(Field)` says which layer set it: `models.dev`,
  `listing:<provider>`, or the caller's layer name.
* `Known(fields...)` is how any layer marks the fields it sets.
* `ModelCost.RatesFor(inputTokens)` picks the highest tier the input
  exceeds.

**Where from.** `catalog.LookupModel(ctx, provider, src, model, opts...)`
merges, field by field:

1. models.dev, unless `WithoutModelMetadata` is set;
2. the provider's listing, when `src` is given;
3. each layer the caller adds with `WithModelLayer`, a later one winning.

A model no layer knows is not an error. A failed source is joined into the
error, beside what the other layers set. `catalog.CachedModel` answers from
the caches without waiting.

**Offline.**

* `catalog.ParseMetadata` reads a models.dev-format document a caller
  persisted.
* `Metadata.Encode` writes the sections the SDK reads, losslessly.
* A `Metadata` is a `Layer`.
* `Metadata.Model` and `Models` read one model, or all of a provider's.

**The listings' metadata:**

| Provider | Fields |
| :--- | :--- |
| Anthropic | limits and capabilities |
| Gemini | limits and thinking |
| xAI | window, efforts and prices |
| Kilo | window, output cap, prices and expiration |
| Together | window and prices |
| OpenAI | shutdown date |
| Ollama | window and capabilities, from `/api/show` |

A listing's metadata is cached for ten minutes, keyed by a fingerprint of the
credential.

**Also:**

* models.dev's document is read for nine sections, each decoded when first
  read. A fetch no longer decodes sections nobody reads.
* `Metadata.ReasoningEfforts`, `InterleavedField` and `NPM` now answer for
  OpenAI, Claude, Gemini, Grok and Kilo too.
* A maintainer command, `go run ./llmprovider/internal/cmd/catalogs`,
  proposes the next curated lists.

**Upgrading:** no code change is needed.

**Step 2, the live suites,** on the release commit `3ed1c25`.

* **Command:** `LLMPROVIDER_LIVE_TOGETHER=1 go test -tags live_gateways
  -count=1 -v ./llmprovider/... -run Live`.
  * Every provider's key variable was set, checked for presence only.
  * A local Ollama answered.
* **Result:** exit 1, sorted with `tools/liveskips.py`.
  * Top level: 77 passed, 31 skipped, 1 failed.
  * Subtests: 95 passed, 5 skipped, 2 failed.
* **This release's test,** `TestLive_LookupModel_0033`, passed for all nine
  keyed providers.
* **The 31 top-level skips:**
  * **Off by their own switch, 29:**
    * the 0028, 0031, 0031B, 0032, 0033 and 0034 measurements, 18 (0033's
      Phase 1 `TestLive_0033MetadataSources` is new);
    * the ChatGPT and Codex CLI subscription, 4;
    * Gemini's rate-limit drain, 1;
    * the Grok CLI, 1;
    * the browser and device sign-ins and the OpenAI sign-in probe, 5.
  * **No active model:** `TestLive_OpencodeInterleavedReasoningReplay`,
    as before.
  * **Transient, 1:** `TestLive_KiloChatCompletions`, `kilo HTTP 429:
    Provider returned error`. Its rerun alone, at 17:31 UTC, passed on
    `kilo-auto/free`.
* **The 5 subtest skips** are the same OpenCode Zen ones, for the same
  account reasons, as in `v1.8.0`.

**Deviation (2026-10-12): OpenCode Zen's wording.**
`TestLive_OpencodeKeyHeaderPerRoute` failed on both its routes. It pins
0007-MADR §1c, which header each Zen route reads, with a bogus key, at no
cost. It passed in `v1.8.0`'s run the day before.

```text
DRIFT: messages with the key in Authorization: want "Missing API key.", got {…"Upstream request failed: Rate-limited models require a workspace"}
DRIFT: messages with the key in x-api-key: want "Invalid API key.", got {…"Upstream request failed: Invalid credential"}
DRIFT: google with the key in Authorization: want "Missing API key.", got {…"Upstream request failed: Rate-limited models require a workspace"}
DRIFT: google with the key in x-goog-api-key: want "Invalid API key.", got {…"Upstream request failed: Invalid credential"}
```

Zen's wording changed. The distinction §1c rests on did not:

* the key in `Authorization` is not read, and the request is answered as
  an anonymous one;
* the key in the route's own header is read, and refused.

The owner chose to update the test's two expected messages, keeping its
meaning, and to rerun that test alone. No code changed, so the run's other
results stand.

* **The test** expects the new wording, and its comment gives both dates.
* **0007-MADR's** measurement table carries a dated note.
* **Rerun,** live, with the bogus key: `--- PASS:
  TestLive_OpencodeKeyHeaderPerRoute`, both routes.
* **Plants,** through `tools/plant.py`, live and not billed:

  | Plant | FAIL lines |
  | :--- | :--- |
  | `Authorization` expected to be read as the key | `DRIFT: messages with the key in Authorization: want "Invalid credential", got {…"Rate-limited models require a workspace"}`, and the same for google |
  | the route's own header expected to be ignored | `DRIFT: messages with the key in x-api-key: want "Rate-limited models require a workspace", got {…"Invalid credential"}`, and the same for google |

  The updated test still tells the two headers apart.

This fix is a test-only commit. It becomes the release commit, and `v1.9.0`
is tagged on it. **Gate:** `tools/gate.py`, 23 of 23.

**Steps 3 and 4, the release.**

* The live suites' fix, `3c006a9`, is the release commit.
* The disclosure guard passed over the outgoing commits before the push.
* The owner pushed `main` and tagged `v1.9.0`, annotated (`59c5c7b`), on
  `3c006a9`.

**Step 5, after the tag:**

| Check | Result |
| :--- | :--- |
| CI on `main` | run `38286544006`, push, success |
| CI on the tag | run `38288513887`, push, success |
| The proxy | `"Version": "v1.9.0"`, `"GoVersion": "1.27.2"`, `Ref refs/tags/v1.9.0`, hash `3c006a9…` |
| apidiff `v1.8.0` to `v1.9.0` | `tools/apidiff.py`, exit 0. 50 additions, all in `catalog`, and nothing incompatible, removed or changed |
| A scratch consumer of every public package | `tools/consumer.py v1.9.0`: 15 packages linked, built for six platforms |
| The offline consumer | passed, built for six platforms; see below |
| A scratch copy of gobble-cli | passed on `v1.9.0`; see below |

The 50 additions are amended D1 to D3 and nothing else:

* **D1, the type:** `ModelInfo`, `ModelReasoning`, `ModelRates`,
  `ModelCost`, `ModelCostTier`, `Modality` and its five values,
  `ModelStatus` and its five values, `Field` and its 23 values.
* **D2, the layers:** `Layer`, `WithModelLayer`, `ParseMetadata`, and
  `Metadata`'s `Name`, `Model`, `Models` and `Encode`.
* **D3, the lookup:** `LookupModel` and `CachedModel`.

**The offline consumer** is a scratch program, outside the repository, that
requires `v1.9.0` from the proxy. It:

* parses the Phase 2 fixture with `ParseMetadata`, and adds it with
  `WithModelLayer`;
* sets `WithoutModelMetadata` and passes a nil token source;
* gives `WithHTTPClient` a transport that counts each request and refuses
  it;
* reads `claude-x` through `LookupModel` and `CachedModel`, and requires
  the window, the output cap, the input and output prices and the tiers to
  be set, each from the layer's `Name()`.

```text
window=1000000 output=128000 input=3 output=15 tiers=1 above-200k input=6 source=models.dev
requests=0
offline: ok
```

`go vet` was clean. The program built with `CGO_ENABLED=0` for
linux, windows and darwin, each on amd64 and arm64. Its two plants, each on
a fresh copy, were caught:

| Plant | FAIL lines |
| :--- | :--- |
| `WithoutModelMetadata` removed | `FAIL: LookupModel: model metadata: Get "https://models.opencode.ai/api.json": offline: request refused`; `FAIL: the counting transport saw 1 request(s), want 0` |
| `WithModelLayer` removed | `FAIL: CachedModel: nothing known`; `FAIL: LookupModel: field 2 unset`, and the same for fields 4, 14, 15 and 19, for both calls |

**The gobble-cli copy.** `tools/consumer.py v1.9.0 --dependent` copied a
local checkout at `ad6ea57` into a scratch directory, without its `.git`,
`*.devops.*.json`, `secrets.json` or `.env*` files. The checkout itself was
not changed.

* **On its own requirement,** `v1.2.1`: `go test ./...` passed.
* **Moved to `v1.9.0`:** `go get` and `go mod tidy`; then `go build`; `go
  vet` for darwin, linux and windows; `go test`; `go test -race`; and
  `golangci-lint run -c .golangci.yml` with 0 issues. Each passed.
* **Pi's `Model`.** gobble-cli has no model type yet: its `llm/catalog` is
  a stub. So a scratch test, `TestPiModelFromModelInfo_0033`, in that
  package of the copy, declares Pi's shape (0033-MADR's consumers). It
  fills `contextWindow`, `maxTokens` and `cost`, with
  `tiers[].inputTokensAbove`, from a `ModelInfo` read offline as above. A
  field `Has` does not report takes Pi's default, 4096 or a price of 0.
  * `claude-x`: window 1000000; `maxTokens` 128000; cost 3, 15, 0.3 and
    3.75; one tier above 200000, at 6, 22.5, 0.6 and 7.5.
  * An unknown model: Pi's defaults.
  * The transport saw no request.
  * `gofmt` clean; `--- PASS`; `golangci-lint` on the package, 0 issues.
* **Its plants** were caught:

  | Plant | FAIL line |
  | :--- | :--- |
  | the window copied without `Has` | `unknown model {ContextWindow:0 MaxTokens:4096 …}; want Pi's defaults, 4096 and a price of 0` |
  | cache write taken from cache read | `claude-x cost {Input:3 Output:15 CacheRead:0.3 CacheWrite:0.3 …}; want {… CacheWrite:3.75 …}` |

  The first is the reason for `Has`: without it an unknown window reads as
  zero.

**Step 6:** this PLAN is `complete`, and `docs/README.md` says so.

#### Verification, confirmed

| | Result |
| :--- | :--- |
| V1. Red, then green | each phase's record gives its red run and every plant's FAIL line |
| V2. The gate | 23 of 23 at every phase; the generation and listing goldens unchanged |
| V3. Measurement | Phase 1's coverage table, recorded before Phase 2, every unit confirmed |
| V4. Benchmark | within 25% once decoding was lazy (Phase 2's deviation): fetch −61.5%, the real document +19.9% |
| V5. API | apidiff `v1.8.0` to `v1.9.0`: 50 additions, those of amended D1 to D3 only |
| V6. Live | `TestLive_LookupModel_0033` passed for nine providers; the live suites on the release commit, with the Zen drift fixed and rerun |
| V7. Offline | the scratch consumer's counting transport saw no request, and saw one when models.dev was left on |

#### Deviations, in all

All dated 2026-10-12, each in its phase's record:

* **Phase 1:** Kilo's listing has cache and reasoning prices; amended D2
  sets them.
* **Phase 2:** decoding every section cost +35.7%, over V4's bound; each
  section is now decoded when first read.
* **Phase 4:** `go test -cover` counts an untested `main` package as 0.0%,
  so `main` calls a tested `run`.
* **Phase 5:** OpenCode Zen changed its error wording; the live test's two
  expected messages were updated, and §1c of 0007-MADR stands.

#### Not done, and why

* **A usable-window or cost helper:** none, by Q12. Each consumer applies
  its own rule.
* **gobble-cli's own adoption:** out of this repository. The scratch test
  shows the shape it needs fills from `ModelInfo`; it lives only in the
  scratch copy.
* **Found, and outside this PLAN:**
  * OpenCode Zen's static `deepseek-v4-flash` is no longer in Zen's
    listing; the Phase 4 command proposes its removal. It waits for the
    next catalog refresh.
  * The `TestCommandToken_*` tests still time out under host load, as
    0034-PLAN-degrade-forced-tool-choice-per-model.md records;
    `tools/gate.py` waits for the load to fall first.
