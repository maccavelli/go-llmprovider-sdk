---
status: proposed
date: 2026-10-11
associated-madr: "0033-MADR-public-model-metadata-and-current-catalogs.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Publish Model Metadata

Associated MADR: [0033-MADR-public-model-metadata-and-current-catalogs.md](0033-MADR-public-model-metadata-and-current-catalogs.md)

This is the second of the MADR's two plans: D1 to D3 and D4's maintainer
tool, released as `v1.8.0`. It follows
[0033-PLAN-refresh-the-curated-catalogs.md](0033-PLAN-refresh-the-curated-catalogs.md).

## Goal

When it is done:

* **What a caller can read,** for any model the SDK knows:
  * its context window, input and output limits;
  * its input modalities, tools and reasoning efforts;
  * its prices, with cache read and write and a long-context tier;
  * its release date and lifecycle.

  For each value, it can tell whether it is known and which layer set it.
* **Where from:** `catalog.Lookup`, or `CachedLookup` without waiting.
  These merge the caller's layers, the provider's listing and models.dev.
* **Offline:** from a document the caller persists through
  `Metadata.Encode` and `ParseMetadata`.
* **The tool:** a maintainer command proposes the next curated lists.

Every exported change is an addition.

## Scope

### In scope

* **`llmprovider/catalog`, new:**
  * the types `ModelInfo`, `ModelCost`, `ModelCostTier`, `Field`,
    `Modality`, `ModelStatus` and `Layer`;
  * `ModelInfo.Has` and `SourceOf`;
  * `WithModelLayer`;
  * `Lookup` and `CachedLookup`;
  * `ParseMetadata`;
  * `Metadata.Encode`, `Model` and `Models`, with `Metadata` satisfying
    `Layer`.
* **`model_metadata.go`:** the decoder's added fields, and
  `modelMetadataKey`'s five added sections.
* **`discovery.go`:**
  * the listing fields of D2's table, for Anthropic, Gemini, xAI, Kilo,
    Together and OpenAI;
  * Ollama's `/api/show` for one model.
* **The maintainer tool:** `internal/cmd/catalogs`, run with `go run`.
* **Tests:** a decoder fixture, the benchmark, and a coverage-floor line if
  the tool's package needs one, by this record.
* **Docs:**
  * `docs/architecture.md`;
  * `docs/guides/adding-a-provider.md` (a new provider names its
    models.dev section, and its listing's fields);
  * `README.md` and release notes.

### Out of scope

* Image input limits, which no source publishes. gobble-cli sets them in
  its own layer.
* A cache-write count in `Usage`: its own record.
* Forced tool choice as a per-model fact: 0033-MADR D6's record. This plan
  adds no forced-choice field to `ModelInfo`; that record does, if it needs
  one.
* Correcting models.dev. The known errors are named in the package doc.
* Hugging Face's per-provider router prices.
* `ListModels`' signature, and `Catalog`.

## Rules for every phase

As in
[0033-PLAN-refresh-the-curated-catalogs.md](0033-PLAN-refresh-the-curated-catalogs.md).

## Implementation Steps

### Phase 1: measurement (live; no billed request)

Behind `LLMPROVIDER_LIVE_0033`, with each key checked for presence only.
Listings and `/api/show` are not billed.

1. **models.dev:**
   * Fetch the document. For every static id of every provider but Ollama,
     record which D1 fields its section sets. An id absent from its
     section is listed.
   * Record the size of the nine sections, re-encoded, for `Encode`.
2. **Listings:** for each listing in D2's table, record the shape of the
   fields D2 maps, for two models each.
3. **Ollama:** `/api/show` on one local model: the `model_info` context
   length and `capabilities`.
4. **Record** a coverage table, provider by field. A field D2 maps that no
   listing returns is a deviation.

### Phase 2: the type, the models.dev layer and the document (D1, D3)

1. **Red first** (`catalog/modelinfo_0033_test.go`), with a fixture of one
   entry per section and every field:
   * **Decoding:**
     * each D1 field decodes;
     * a field present at zero, such as a free price, has `Has` true;
     * a field absent has `Has` false;
     * `SourceOf` says `models.dev`.
   * **Copies:** `Metadata.Models` returns copies.
   * **The document:**
     * `Encode` then `ParseMetadata` gives an equal `Metadata`;
     * `ParseMetadata` makes no network call;
     * it refuses a body over the limit, and one that is not a document.
2. **Change:**
   * the types;
   * the presence set;
   * the decoder and the key map;
   * `Model`, `Models`, `Encode` and `ParseMetadata`.
3. **The benchmark:** `BenchmarkDecodeModelMetadata`, before and after. A
   slowdown over 25% is a deviation.
4. **Plants**, each failing its test:
   * a decoder that drops `limit.output`;
   * a presence set that marks a zero price unknown;
   * a `Models` that returns the cached slice;
   * an `Encode` that drops a section.
5. Gate; stage.

### Phase 3: the layers and the resolver (D2, D3)

1. **Red first,** with canned replies:
   * **Each listing:** `Lookup` takes the listing's value for each field D2
     gives that listing, and models.dev's for the others. `SourceOf` names
     each.
   * **Caller layers:** a caller layer beats both. Of two caller layers,
     the later wins. A field no layer sets stays unset.
   * **Offline:** with `WithoutModelMetadata` and a parsed document as a
     layer, `Lookup` makes no request to models.dev.
   * **`CachedLookup`:** answers from the cache without waiting, starts a
     refresh, and reports false when nothing is known.
   * **Ollama:** `Lookup` for `ollama` reads a canned `/api/show`.
   * **Bounds:** `Lookup` honours its context's cancellation (R40), and its
     errors are bounded (R35).
2. **Change:**
   * decode the listing fields;
   * `Layer` and `WithModelLayer`;
   * the merge;
   * `Lookup` and `CachedLookup`.
3. **Goldens:** the listing and generation requests do not change.
4. **Plants**, each failing:
   * a merge where models.dev wins;
   * caller layers in the wrong order;
   * a listing field not decoded;
   * a `CachedLookup` that waits.
5. Gate; stage.

### Phase 4: the maintainer tool (D4)

1. **`internal/cmd/catalogs`:**
   * it reads models.dev and, for each provider whose key variable is set,
     its listing;
   * it prints, per provider, the ids it proposes to add or remove, with
     each id's status, release date and price;
   * it edits nothing, and prints no key.
2. **Red first:** its proposal function, given a canned document and
   listing:
   * proposes removing a retired id;
   * proposes adding a current one;
   * proposes nothing for a list already current.
3. **Plant:** a proposal that ignores `Status` fails.
4. Gate; stage.

### Phase 5: documentation and release

1. **Docs:**
   * `docs/architecture.md`: the catalog section, the layers, and the tool;
   * `docs/guides/adding-a-provider.md`;
   * `README.md` Status, `v1.8.0`;
   * release notes in this PLAN.
2. **The live suites** on the release commit (billed, as before).
3. **The owner** commits, pushes, and tags `v1.8.0`.
4. **After the tag:**
   * CI;
   * the proxy;
   * apidiff `v1.7.1` to `v1.8.0`: additions only;
   * a scratch consumer that reads `ModelInfo` offline, through
     `ParseMetadata` and a layer;
   * a scratch copy of gobble-cli.
5. This PLAN `complete`; the index updated.

## Verification

* **V1. Red, then green,** with every plant's FAIL line.
* **V2. The gate** at each phase. The generation and listing goldens are
  unchanged.
* **V3. Measurement:** Phase 1's coverage table recorded before Phase 2.
* **V4. Benchmark:** within 25%.
* **V5. API:** additions only.
* **V6. Live:** the live suites on the release commit.

## Rollout and Rollback

* **Rollout.** Four commits after Phase 1, then `v1.8.0`.
* **Rollback, before the tag:** each phase reverts alone, newest first.
* **Rollback, after the tag:** fix forward in `v1.8.x`. A caller that does
  not use the new API is unaffected.

## Execution Record

Not started.
