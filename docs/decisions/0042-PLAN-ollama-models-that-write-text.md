---
status: proposed
date: 2026-10-14
associated-madr: "0042-MADR-ollama-models-that-write-text.md"
---

<!-- markdownlint-disable MD013 -->

# Implement Only the Ollama Models That Write Text, and Keep the Tool-less Ones Recommended

Associated MADR: [0042-MADR-ollama-models-that-write-text.md](0042-MADR-ollama-models-that-write-text.md)

## Goal

Close [issue 1](https://github.com/maccavelli/go-llmprovider-sdk/issues/1)
and remove the regression 35c2e47 introduced, as 0042-MADR decides:

* **D1.** An Ollama model whose capabilities are known and lack
  `completion` is not usable. It leaves `Usable`, `Menu` and
  `Recommended`. A model whose capabilities are unknown is kept. The fact
  is recorded as `ModelInfo.Output`.
* **D2.** On Ollama, a model whose only caveat is `CaveatNoTools` stays in
  `Recommended`, after the uncaveated ones. Every other provider is
  unchanged.

No exported identifier is added, removed or changed (R48). Only Ollama's
listing results, and Ollama's `ModelInfo.Output`, change.

## Scope

**In scope.**

| File | Change | Phase |
| :--- | :--- | :--- |
| `llmprovider/catalog/listing_metadata.go` | capability constants; `ollamaInfo` sets `Output` and reads an empty list as unknown; `ollamaOutputs`; `ollamaTagInfo` | 1 |
| `llmprovider/catalog/discovery.go` | the five positional `listingResult` literals keyed, no value changed; `fetchOllamaNames` becomes `fetchOllamaTags`; `ollamaTag`; `ollamaCatalog` builds `listingResult.listed`; the Ollama branch of `listingResult.catalog` reads it; `Catalog.Usable` and `Catalog.Recommended` docs | 1, 2 |
| `llmprovider/catalog/menu.go` | `writesNoText`, used by `caveatsOf` and `ollamaMenu`; `ollamaMenu` filters `Usable` (1) and builds `Recommended` with `ollamaRecommends` (2); `menuIDsOf` removed (2); `Caveat` and `CaveatNoTools` docs (2) | 1, 2 |
| `llmprovider/catalog/ollama_0042_test.go` (new) | the listing tests, and `ollamaInfo`'s | 1, 2 |
| `llmprovider/catalog/ollama_facts_0042_test.go` (new) | the tests of `ollamaTagInfo` and `fetchOllamaTags` | 1 |
| `llmprovider/catalog/read_limit_test.go` | the renamed fetch | 1 |
| `llmprovider/catalog/lookup_0033_test.go` | Ollama's listing case gains `FieldOutput` | 1 |
| `llmprovider/catalog/menu_0041_test.go` | `TestMenu_Ollama_0041`: `Recommended` keeps `notools:1` | 2 |
| `llmprovider/providers/ollama/listing_test.go` | `ListModels` leaves out non-text models (1) and keeps tool-less ones (2) | 1, 2 |
| `wizard/model_select_edge_test.go` | the wizard's menu and search for S1 (1) and S2 (2) | 1, 2 |
| `llmprovider/catalog/live_ollama_0042_test.go` (new, `live_gateways`) | the live check | 3 |
| `docs/architecture.md` | the Ollama facts and menu lines (R43) | 1, 2 |
| `docs/decisions/0038-MADR-full-curated-catalog-by-default.md` | dated annotations on its two Ollama lines | 1 |
| `docs/decisions/0041-MADR-decide-per-model-behaviour-from-metadata.md` | an amendment for D2, and annotations on D3 and A4 | 2 |
| `docs/decisions/0041-PLAN-decide-per-model-behaviour-from-metadata.md` | a dated cross-reference under Phase 4's record | 2 |
| this PLAN, `docs/README.md` | execution records, statuses | each |

**Out of scope.**

* Skipping `/api/show` for a model whose `/api/tags` capabilities lack
  `completion` (0042-MADR D1, "What is not done").
* T2, a profile-scoped no-tools rule (0042-MADR, owner decision 3).
* prepare-commit-msg's pin bump and its README workaround. They are that
  repository's work, after a release carries this plan.
* A release or tag. 0041-PLAN Phase 9 releases, after this plan's Phase 2
  (0042-MADR owner decision 2).
* `catalog.ListRaw` (0041-PLAN Phase 8). It must not apply D1's filter, and
  its own plant ("the usability filter applied in `ListRaw`") already
  guards that.

**Interactions with 0041-PLAN.**

* **Phase 5** edits `discovery.go`. This plan starts after Phase 5's
  commit (Phase 0, step 1).
* **Phase 6** gives `listingEntry` the ordered usable ids. Run after this
  plan, it caches Ollama's filtered `Usable`. Its research step should read
  `ollamaCatalog` as this plan leaves it.
* **Phase 8's** research reads Ollama's fetch as P34, `fetchOllamaNames`.
  After Phase 1 that is `fetchOllamaTags`. P34 is pinned to `2c82cf1`, so
  `tools/claims.py` still passes. Phase 8 reads the new name, and keeps
  `name` as the raw `ID`.
* **Phase 9's** release and live check include this plan's work.

## Design

**The data flow, after Phase 2.**

1. `fetchOllamaTags` decodes `/api/tags` into `[]ollamaTag{Name,
   Capabilities}`. `Capabilities` is nil before Ollama v0.30.0.
2. `ollamaShows` reads `/api/show` for each name that is not cached, four
   at a time, as today. A failed reply leaves its name out of the map.
3. For each tag, `ollamaCatalog`:
   * starts from `ollamaTagInfo(tag)`, which sets only `Output`, and only
     when the tag lists capabilities;
   * overlays the `/api/show` reply field by field (`overlay`,
     `lookup.go:197`), so every field the reply sets wins;
   * emits the reply alone (`cfg.emit`).

   So `listOnce` caches what `/api/show` said, and nothing from
   `/api/tags` (`storeListing(..., models, false)`, `discovery.go:171`).
   A failed reply is asked again on the next listing, rather than cached
   as answered for ten minutes by `/api/tags`' partial fact.
4. `listingResult.listed` carries the merged facts to
   `listingResult.catalog`. Its Ollama branch merges the caller's layers
   over them (`factsOf`), as every other provider's facts are merged.
5. `ollamaMenu` drops each name for which `writesNoText` holds. That
   predicate is the one `caveatsOf` uses for `CaveatNoTextOutput`, so a
   caveat and the filter cannot disagree. It then orders the menu as
   today, and recommends each entry for which `ollamaRecommends` holds:
   no caveat, or `CaveatNoTools` only.

**Why this shape.**

* **One fact, one predicate.** "Writes no text" is `ModelInfo.Output`,
  the field models.dev already fills for every other provider. Ollama gets
  no special flag that `LookupModel`, `CachedFacts` and the menu would
  each have to learn.
* **The cache stays honest.** Only `/api/show` replies are cached, exactly
  as today. The `/api/tags` fallback lives for one listing.
* **Caller layers still win** (0041-MADR D1). The filter reads the merged
  facts, so a caller can re-admit a model by setting `Output`.
* **No new request,** no new goroutine, no new option, no exported change.
* **Ollama's absent or empty capability list is "unknown" everywhere.**
  One `len(caps) > 0` test in each reader replaces `!= nil`.

**Proven on a prototype** (2026-10-14). The plan's code was applied, by a
script, to a scratch clone of the working tree (0041-PLAN Phase 5,
uncommitted, over 35c2e47), with a printing test of every case. The tree
was not touched.

* **Every final-state value in the tables below came out as written:**
  S1 to S9 with Phase 2's `Recommended`, M1 to M4, S4b's second listing
  and cache check, and `ollamaInfo`'s seven rows. The prototype held both
  phases, so the Phase 1-only column for S2 and S9 was not measured.
  Phase 1's red and green runs measure it.
  * An empty install's `Usable` stayed nil.
  * S8's came out empty, not nil, so its test checks the length.
* **`go test -count=1 ./...` failed in one test only:**
  `TestMenu_Ollama_0041`, with
  `Recommended [b:1 a:1 notools:1]; want [b:1 a:1]`, which is Phase 2's
  named assertion change. Every other package passed, `TestWireGoldens`
  in `providers/ollama` among them. So the golden is unchanged.
* **`go vet` was clean,** and `golangci-lint run -c .golangci.yml
  --build-tags live_gateways ./llmprovider/catalog/...` reported
  0 issues. `gofmt -l` listed nothing.
* **It found one gap, now step 3's first change.** The other listers
  build `listingResult` positionally (five literals, `discovery.go`), so
  adding `listed` failed the build with "too few values in struct literal
  of type listingResult", five times. Keyed literals fixed it.
* **Each plant's `old` string occurs exactly once** in the prototype's
  files.

## Rules for every phase

* **Red first.** Each new test is run before the change it pins, and its
  failure is recorded in the phase's execution record.
* **Plants.** Each new assertion is seen to fail on a deliberate breach,
  through `python3 -B tools/plant.py SPEC.json`, on a scratch clone, never
  in the tree.
  * Every plant compiles. A plant that only breaks the build does not
    count.
  * The specs below are written against this plan's code. Their `old`
    strings are copied from the code as committed, and the tool refuses
    any that does not occur exactly once.
* **Checks before staging:**
  * `make pre-add-check FILES="<each changed .go file>"`;
  * `python3 -B tools/apidiff.py <phase start>` shows no exported change;
  * `make records-check`;
  * `python3 -B tools/claims.py docs/decisions/0042-PLAN-ollama-models-that-write-text.md`;
  * `python3 -B tools/denyscan.py --changed`;
  * `python3 -B tools/linkcheck.py`;
  * `python3 -B tools/gate.py --out <scratch>`, 23 of 23.

  Each command's output goes to a scratch file, and its exit status is
  read before any filter.
* **Docs in the same change** (R43): `docs/architecture.md`, and the doc
  comment of each symbol whose behaviour changes.
* **Commits.** The agent stages; the owner commits with
  `git commit --no-edit`. Each phase is its own commit. No push or tag
  without the owner's ask in the same turn.
* **Deviations stop work.** The agent prompts the owner with the
  evidence, and the resolutions with the recommended one first. Then it
  records a dated deviation here, and a 0042-MADR amendment where a
  decision or a fact changes.
* **Scripts** are Python, in the scratchpad, or in `tools/` with a test if
  worth keeping.

## Implementation Steps

### Phase 0: preconditions and the baseline

1. **0041-PLAN Phase 5 is committed.**
   * `git log --oneline -3` shows Phase 5's commit above 35c2e47.
   * `git status --short` lists nothing but this pair and
     `docs/README.md`, or nothing once the owner has committed them.
   * If Phase 5 is not committed, stop. This plan does not stage over
     uncommitted work.
2. **The records commit.** The owner commits 0042-MADR, this PLAN and
   `docs/README.md` alone, under the bootstrap exception. It may be before
   or after step 1; it is never in a commit with source.
3. **The baseline.**
   * Record `git rev-parse HEAD` as `<base>`.
   * `go test -count=1 ./llmprovider/catalog/ ./llmprovider/providers/ollama/ ./wizard/`
     passes.
   * `python3 -B tools/gate.py --out <scratch>` reports 23 of 23.
4. **The reproduction** is Phase 1's first red run, at `<base>`. Its S1
   to S7 must give 0042-MADR section 3's 35c2e47 column. Phase 5 does not
   touch Ollama, and the working tree measured equal to 35c2e47 on
   2026-10-14. Any other value is a deviation.
5. **Re-find the lines.**
   * Run `tools/claims.py` on this PLAN. It is pinned to 35c2e47, so it
     passes.
   * Then find each Evidence-index row's text at `<base>` by content.
     `discovery.go`'s lines moved in Phase 5. Edit by content, never by
     line number.

### Phase 1: what Ollama says a model writes (D1)

1. **Research.** Each result goes into the execution record.
   * `grep -rn 'fetchOllamaNames' llmprovider wizard`. Expect only
     `discovery.go` (the definition and `ollamaCatalog`) and
     `read_limit_test.go:35`. Any other caller is a deviation.
   * `grep -rn 'FieldOutput\|\.Output, ' llmprovider/providers llmprovider/internal`.
     Expect no request-side reader of `ModelInfo.Output`, which proves
     that setting it changes no request. A reader is a deviation.
   * Confirm `listOnce` stores exactly what was emitted
     (`discovery.go:171` at 35c2e47).
   * `grep -n 'listingResult{' llmprovider/catalog/discovery.go`. Expect
     five positional literals: the first-party providers, Hugging Face,
     Kilo, Together and the OpenCode gateways. A different count changes
     step 3's first edit, and is recorded.
   * Confirm `slices.Clone(nil)` and `slices.DeleteFunc(nil, …)` return
     nil, so an empty install keeps a nil `Usable`, as
     `TestConfigureLLM_OllamaEmptyListing` relies on.
2. **Red: the tests first,** in two files, so that a compile error cannot
   hide an assertion's failure:
   * **`llmprovider/catalog/ollama_0042_test.go`** holds every test that
     compiles at `<base>`: `TestOllamaInfo_Output_0042` and
     `TestOllamaUsable_0042`. With them go the `LookupModel`, provider and
     wizard changes below. Run at `<base>`, they fail on their
     assertions.
   * **`llmprovider/catalog/ollama_facts_0042_test.go`** holds the two that
     name the new functions: `TestOllamaTagInfo_0042` and
     `TestFetchOllamaTags_0042`. It is added second. Its red is
     `undefined: ollamaTagInfo` and `undefined: fetchOllamaTags`.

   The tests:

   * **The fake server, `ollama0042Server(t, models...)`.** Each model is
     `{name, tags, show string}`:
     * `tags`: the capabilities JSON in its `/api/tags` entry, or `""` to
       leave the field out;
     * `show`: the capabilities JSON in its `/api/show` reply, `""` for a
       reply without the field, or `"500"` for a failure.

     It counts `/api/show` requests per model under a mutex. The MADR's
     scratch reproduction is its model.
   * **`TestOllamaInfo_Output_0042`** (table, through `ollamaInfo`):

     | `capabilities` | `Output` | `FieldOutput` known | `Tools` known / value |
     | :--- | :--- | :--- | :--- |
     | `["completion","tools","insert"]` | `[text]` | yes | yes / true |
     | `["embedding"]` | empty | yes | yes / false |
     | `["image"]` | `[image]` | yes | yes / false |
     | `["completion","image"]` | `[text image]` | yes | yes / false |
     | `["decision"]` | empty | yes | yes / false |
     | field absent | — | no | no |
     | `[]` | — | no | no *(was yes / false)* |

   * **`TestOllamaTagInfo_0042`.**
     * Each capability row above, given as an `ollamaTag`, sets the same
       `Output`, with `SourceOf(FieldOutput) == "listing:ollama"`.
     * `FieldTools`, `FieldInput` and `FieldReasoning` are never set.
     * With no capabilities, nothing is set.
   * **`TestFetchOllamaTags_0042`.** A body with two entries, one with
     `capabilities` and one without, decodes in order. The first has its
     list; the second has nil.
   * **`TestOllamaUsable_0042`** (table). Each case runs `List` for Ollama
     with `WithBaseURL`, `WithoutModelMetadata` and `WithLimit(MaxListed)`,
     after `resetLookupCaches(t)`:

     | Case | Installed: `tags` / `show` | `Usable` | `Recommended` (Phase 1) | Menu caveats | `/api/show` asked |
     | :--- | :--- | :--- | :--- | :--- | :--- |
     | S1 | `granite-embedding:30m` – / `["embedding"]`; `qwen3-vl:8b` – / `["completion","vision","tools","thinking"]`; `llama3.1:8b-instruct-q5_K_M` – / `["completion","tools"]`; `qwen2.5-coder:7b` – / `["completion","tools","insert"]` | the three chat models, in order | the same three | none | 4 |
     | S2 | `gemma3:12b` – / `["completion","vision"]`; `codellama:7b` – / `["completion","insert"]` | both | none *(Phase 2 changes it)* | both `no-tools` | 2 |
     | S3 | `embed-tools:1` – / `["tools","embedding"]`; `chat:1` – / `["completion","tools"]` | `chat:1` | `chat:1` | none | 2 |
     | S4 | `granite-embedding:30m` – / `500`; `chat:1` – / `["completion","tools"]` | both | both | none | 2 |
     | S4b | `granite-embedding:30m` `["embedding"]` / `500`; `chat:1` `["completion","tools"]` / `["completion","tools"]` | `chat:1` | `chat:1` | none | 2; a second `List` asks the embedding model again and `chat:1` not (3 in all) |
     | S5 | `nomic-embed-text:latest` – / no field; `chat:1` – / no field | both | both | none | 2 |
     | S6 | `x/flux2:1` – / `["image"]`; `classifier:1` – / `["decision"]`; `chat:1` – / `["completion","tools"]` | `chat:1` | `chat:1` | none | 3 |
     | S7 | `granite-embedding:30m` `["embedding"]` / `["embedding"]`; `qwen2.5-coder:7b` `["completion","tools","insert"]` / the same | `qwen2.5-coder:7b` | the same | none | 2 |
     | S8 | `granite-embedding:30m` and `nomic-embed-text:latest`, each – / `["embedding"]` | empty, `Live` true, `Err` nil | empty | — | 2 |
     | S9 | S1, with a caller layer (`staticLayer`, `lookup_0033_test.go`) that sets the embedding model's `Output` to `[text]` | all four | the three chat models *(Phase 2: all four, the embedding model last)* | the embedding model `no-tools` | 4 |

     S4b also asserts that `cachedListing` holds no entry for the
     embedding model. That is the proof the `/api/tags` fact is not
     cached.
   * **`LookupModel`.** Ollama's row in `listingCases`
     (`lookup_0033_test.go:183`) adds `FieldOutput` to its fields. Its
     check asserts `Output` is `[text]`.
   * **The provider:** `TestListModels_LeavesOutNonText_0042` in
     `providers/ollama/listing_test.go`, with probes off
     (`WithModelProbes(false)`):
     * S1's server gives the three chat models;
     * S8's gives `nil` and a nil error.
   * **The wizard:** `TestConfigureLLM_OllamaLeavesOutEmbedding_0042` in
     `wizard/model_select_edge_test.go`, with S1's server and the file's
     `fakePrompter`:
     * a blank search shows the three chat models;
     * the search `granite` notifies "no Ollama models match".

     A second case serves S8: it falls through to manual entry with no
     notice, as `TestConfigureLLM_OllamaEmptyListing` does.
   * **Run them before the change,** in that order, and record each
     failure.
     * **At `<base>`,** S1, S3, S4b, S6, S7 and S8 fail on `Usable`, and
       `TestOllamaInfo_Output_0042` fails on `Output` and on the `[]` row.
       The `LookupModel` row fails, and so does the wizard's `granite`
       search.
     * **These pass at `<base>`:**
       * S2, S4, S5 and S9. They pin what must not change, and Phase 1's
         plants are what prove them.
       * The provider's S1 and S8 tests, and the wizard's blank search.
         35c2e47 already keeps the embedding models out of `Recommended`,
         through `CaveatNoTools`. Phase 2's plant "the filter removed,
         under D2" proves them.
     * **The `/api/show` counts** at `<base>` must equal those in
       0042-MADR section 3. That run is the reproduction (Phase 0,
       step 4).
3. **Change.**
   * **First, key the five positional `listingResult` literals** in
     `discovery.go`, for example
     `listingResult{usable: usable, err: err, static: Static(p), curate: metadataCurate(…), meta: meta}`.
     * No value changes.
     * `go test -count=1 ./llmprovider/...` passing, before any other
       edit, is the proof.
     * It is what lets `listingResult` gain a field without breaking every
       lister.
   * **`listing_metadata.go`:**

     ```go
     // Ollama's capability names, as its types/model/capability.go spells
     // them (0042-MADR, section 2).
     const (
         ollamaCompletion = "completion"
         ollamaImage      = "image"
         ollamaThinking   = "thinking"
         ollamaTools      = "tools"
         ollamaVision     = "vision"
     )

     // ollamaInfo reads one /api/show reply. A missing or empty capability
     // list says nothing: Ollama omits an empty one (0042-MADR D1).
     func ollamaInfo(model string, show ollamaShowReply) ModelInfo {
         b := newInfo(llmprovider.ProviderOllama, model)
         // ... the context window, unchanged ...
         if caps := show.Capabilities; len(caps) > 0 {
             b.info.Tools = slices.Contains(caps, ollamaTools)
             b.info.Input = []Modality{ModalityText}
             if slices.Contains(caps, ollamaVision) {
                 b.info.Input = append(b.info.Input, ModalityImage)
             }
             b.info.Output = ollamaOutputs(caps)
             b.info.Reasoning.Supported = slices.Contains(caps, ollamaThinking)
             b.set(FieldTools, FieldInput, FieldOutput, FieldReasoning)
         }
         return b.done()
     }

     // ollamaOutputs is what a model writes, from its capabilities: text
     // when it completes, an image when it makes one. An embedding or a
     // decision model writes neither (0042-MADR D1).
     func ollamaOutputs(capabilities []string) []Modality {
         var out []Modality
         if slices.Contains(capabilities, ollamaCompletion) {
             out = append(out, ModalityText)
         }
         if slices.Contains(capabilities, ollamaImage) {
             out = append(out, ModalityImage)
         }
         return out
     }

     // ollamaTagInfo is what one /api/tags entry says of its model: only
     // what it writes. Its tools, input and reasoning come from /api/show
     // alone, where the two endpoints were measured to differ (0042-MADR D1).
     func ollamaTagInfo(t ollamaTag) ModelInfo {
         b := newInfo(llmprovider.ProviderOllama, t.Name)
         if len(t.Capabilities) > 0 {
             b.info.Output = ollamaOutputs(t.Capabilities)
             b.set(FieldOutput)
         }
         return b.done()
     }
     ```

     `fetchOllamaShow`'s doc keeps naming "tools, vision, thinking", and
     adds "and what it writes".
   * **`discovery.go`:**

     ```go
     // ollamaTag is one /api/tags entry: the model's name, and the
     // capabilities Ollama v0.30.0 and later list beside it.
     type ollamaTag struct {
         Name         string   `json:"name"`
         Capabilities []string `json:"capabilities"`
     }

     // fetchOllamaTags returns every installed model in /api/tags order,
     // with its capabilities where Ollama lists them (0042-MADR D1).
     // Ollama takes no credential, so a token is sent only where it names
     // its own Header (R16; 0016-MADR D2, A6).
     func fetchOllamaTags(ctx context.Context, token llmprovider.Token, cfg config) ([]ollamaTag, error) {
         // ... the request, the status check and decodeLimited, unchanged ...
         var result struct {
             Models []ollamaTag `json:"models"`
         }
         // ...
         return result.Models, nil
     }
     ```

     `ollamaCatalog`:

     ```go
     // ollamaCatalog lists every installed model, and reads /api/show for
     // each one whose reply is not cached, for its caveats (0041-MADR A4).
     // Only the replies are emitted, and so cached. A model's /api/tags
     // capabilities say what it writes where its reply does not, for this
     // listing alone, so a failed reply is asked again the next time
     // (0042-MADR D1). Ollama has no static catalog, so its errors are
     // returned rather than degraded, and an empty install is a successful
     // (Live) listing.
     func ollamaCatalog(ctx context.Context, token llmprovider.Token, cfg config) (listingResult, error) {
         tags, err := fetchOllamaTags(ctx, token, cfg)
         if err != nil {
             return listingResult{}, err
         }
         var names []string
         for _, t := range tags {
             names = append(names, t.Name)
         }
         cached, fresh := cachedListing(listingKeyFor(llmprovider.ProviderOllama, cfg, token))
         if !fresh {
             cached = nil
         }
         shows := ollamaShows(ctx, token, cfg, names, cached)
         listed := make(map[string]ModelInfo, len(tags))
         for _, t := range tags {
             info := ollamaTagInfo(t)
             if show, ok := shows[t.Name]; ok {
                 cfg.emit(show)
                 overlay(&info, show, "")
             }
             listed[t.Name] = info
         }
         // Recommended never shares Usable's array (0020-MADR F34).
         return listingResult{usable: names, listed: listed}, nil
     }
     ```

     `listingResult` gains:

     ```go
     // listed is what an Ollama listing read of each model, its
     // /api/tags entry under its /api/show reply, cached or not; nil for
     // the other listers, whose facts are the ones they emit (0042-MADR D1).
     listed map[string]ModelInfo
     ```

     Its Ollama branch reads it:
     `return ollamaMenu(l.usable, factsOf(id, cfg, l.usable, l.listed, nil), cfg.limit, now)`.

     `Catalog.Usable`'s doc gains: "Ollama's admits each installed model
     that writes text, or whose output Ollama does not report (0042-MADR
     D1)."
   * **`menu.go`:**

     ```go
     // writesNoText reports whether info says the model writes no text:
     // its output is known, and text is not in it (0041-MADR D3; 0042-MADR D1).
     func writesNoText(info ModelInfo) bool {
         return info.Has(FieldOutput) && !slices.Contains(info.Output, ModalityText)
     }
     ```

     * `caveatsOf` calls `writesNoText(info)` for `CaveatNoTextOutput`.
     * `ollamaMenu`'s first statement is
       `usable := slices.DeleteFunc(slices.Clone(names), func(id string) bool { return writesNoText(facts[id]) })`.
       It iterates `usable`, and returns `Usable: usable`.
     * Its doc says a model Ollama says writes no text is not listed.
   * **`read_limit_test.go:35`** calls `fetchOllamaTags`.
4. **Green.** Every test of step 2 passes, and
   `go test -count=1 ./llmprovider/... ./wizard/` passes.
   * The Ollama wire golden
     (`providers/ollama/testdata/wire/ollama/listing.json`) is unchanged,
     with no `-update`. A change is a deviation: 0042-MADR's annotated
     consequence predicts none.
5. **Plants** (`<scratch>/plant42_p1.json`):
   * packages: `./llmprovider/catalog/`, `./llmprovider/providers/ollama/`,
     `./wizard/`;
   * run: `_0042|_0033|_0041|TestListing_PageLimit`.

   | Plant | File | `old` → `new` | Caught by |
   | :--- | :--- | :--- | :--- |
   | the filter removed | `menu.go` | `usable := slices.DeleteFunc(slices.Clone(names), func(id string) bool { return writesNoText(facts[id]) })` → `usable := slices.Clone(names)` | S1, S3, S6, S7, S8; the wizard's `granite` search |
   | unknown read as no text | `menu.go` | `return info.Has(FieldOutput) && !slices.Contains(info.Output, ModalityText)` → `return !info.Has(FieldOutput) \|\| !slices.Contains(info.Output, ModalityText)` | S4, S5 |
   | the `/api/tags` fallback ignored | `discovery.go` | `info := ollamaTagInfo(t)` → `info := newInfo(llmprovider.ProviderOllama, t.Name).done()` | S4b |
   | the `/api/tags` fact cached | `discovery.go` | `listed[t.Name] = info` → `listed[t.Name] = info; cfg.emit(info)` | S4b's second listing and its cache assertion |
   | an empty list read as known | `listing_metadata.go` | `if caps := show.Capabilities; len(caps) > 0 {` → `if caps := show.Capabilities; caps != nil {` | `TestOllamaInfo_Output_0042`'s `[]` row |
   | the image ignored | `listing_metadata.go` | `if slices.Contains(capabilities, ollamaImage) {` → `if false && slices.Contains(capabilities, ollamaImage) {` | the `image` rows |
   | `Output` not marked | `listing_metadata.go` | `b.set(FieldTools, FieldInput, FieldOutput, FieldReasoning)` → `b.set(FieldTools, FieldInput, FieldReasoning)` | S1, S3, S6; `LookupModel`'s Ollama row |
   | the caller's layers skipped | `discovery.go` | `factsOf(id, cfg, l.usable, l.listed, nil)` → `factsOf(id, config{}, l.usable, l.listed, nil)` | S9 |
   | tools read from `/api/tags` | `listing_metadata.go` | `b.set(FieldOutput)` → `b.info.Tools = true; b.set(FieldOutput, FieldTools)` | `TestOllamaTagInfo_0042` |

6. **Docs.**
   * `docs/architecture.md`:
     * the facts list's Ollama line (about line 225) becomes "Ollama:
       window, capabilities and what it writes, from one model's
       `/api/show`; what it writes from `/api/tags` when that reply fails
       (Ollama v0.30.0 on)".
     * The Discovery bullet on Ollama (about line 627) adds: "a model
       Ollama says writes no text, such as an embedding model, is not
       listed".
   * `0038-MADR`, D1's "**Ollama:** the installed models, in listing
     order" and A5's "**Ollama is unchanged:**": each gains
     "*(0042-MADR D1, 2026-10-14: less the models Ollama says write no
     text.)*".
7. **Checks** per "Rules for every phase". `tools/apidiff.py <base>` shows
   no exported change.
8. **Record and stage.** Write the Phase 1 execution record. Stage the
   phase's files only. The owner commits.

### Phase 2: tool-less chat models stay recommended (D2)

1. **Research.**
   * `grep -rn 'menuIDsOf' llmprovider`. Expect only `ollamaMenu`, which
     makes it removable.
   * Read `TestMenu_Caveats_0041` (`menu_0041_test.go:118`). It is the
     guard that `menuOf`, every other provider's menu, still keeps a
     `no-tools` entry out of `Recommended`. It must pass unchanged.
2. **Red: the tests first,** in `ollama_0042_test.go` unless named:
   * `TestOllamaUsable_0042`'s S2 expects `Recommended`
     `[gemma3:12b codellama:7b]`. S9 expects all four, the embedding model
     last.
   * **`TestOllamaRecommended_0042`** (table):

     | Case | Installed, in `/api/tags` order (`show`) | `Menu` | `Recommended` |
     | :--- | :--- | :--- | :--- |
     | M1 | `na:1` `["completion"]`, `tb:1` `["completion","tools"]`, `nc:1` `["completion"]`, `td:1` `["completion","tools"]` | `tb td na nc` | `tb td na nc` |
     | M2 | M1, with caller layers marking `td:1` and `nc:1` `StatusDeprecated` | `tb na nc td`; `nc` carries `deprecated, no-tools`, `td` `deprecated` | `tb na` |
     | M3 | M1 with `WithLimit(2)` | `tb td` | `tb td` |
     | M4 | M1 with `WithLimit(3)` | `tb td na` | `tb td na` |

   * **`TestMenu_Ollama_0041`'s `Recommended`** (`menu_0041_test.go:329`)
     becomes `[b:1 a:1 notools:1]`. Its doc says the no-tools model is
     after the others, and in `Recommended`. This assertion change is
     D2's, and the execution record names it.
   * **The provider:** `TestListModels_KeepsToolless_0042`, with probes
     off. S2's server gives both models. It gives nothing at Phase 1's
     commit, as at 35c2e47: that is its red.
   * **The wizard:** `TestConfigureLLM_OllamaToolless_0042`. S2's server:
     * a blank search shows both models;
     * no "meets the profile" notice is given.
3. **Change, in `menu.go`:**

   ```go
   // ollamaMenu is an Ollama listing's catalog: no band; the installed
   // models that may write text, in /api/tags order, the caveated after
   // the others (0041-MADR A4; 0042-MADR D1). Recommended is the menu
   // less each entry with a caveat other than CaveatNoTools: the install
   // is all there is, so a chat model that takes no tools stays
   // recommended, after those that do (0042-MADR D2).
   func ollamaMenu(names []string, facts map[string]ModelInfo, limit int, now time.Time) Catalog {
       usable := slices.DeleteFunc(slices.Clone(names), func(id string) bool { return writesNoText(facts[id]) })
       var uncaveated, sunk []MenuEntry
       for _, id := range usable {
           e := entry(llmprovider.ProviderOllama, id, facts, 0, now)
           if len(e.Caveats) == 0 {
               uncaveated = append(uncaveated, e)
           } else {
               sunk = append(sunk, e)
           }
       }
       menu := slices.Concat(uncaveated, sunk)
       var recommended []string
       for _, e := range menu {
           if ollamaRecommends(e) {
               recommended = append(recommended, e.ID)
           }
       }
       return Catalog{Recommended: listlimit.Cut(recommended, limit), Usable: usable, Live: true,
           Menu: cutMenu(menu, limit)}
   }

   // ollamaRecommends reports whether an Ollama menu entry is recommended:
   // it has no caveat but CaveatNoTools (0042-MADR D2).
   func ollamaRecommends(e MenuEntry) bool {
       return !slices.ContainsFunc(e.Caveats, func(c Caveat) bool { return c != CaveatNoTools })
   }
   ```

   * `menuIDsOf` is deleted.
   * The `Caveat` type's doc ends "…and keeps it out of Recommended
     (0041-MADR D3), except CaveatNoTools on Ollama (0042-MADR D2)."
   * `CaveatNoTools`'s doc adds "On Ollama it stays in Recommended, after
     the models that take tools (0042-MADR D2)."
   * `Catalog.Recommended`'s doc: "It is Menu's ids less the caveated
     ones, except that Ollama's keeps those whose only caveat is
     CaveatNoTools (0042-MADR D2)."
4. **Green.**
   * Every test passes, `TestMenu_Caveats_0041` and
     `TestMenu_Snapshot_0041` unchanged among them.
   * The Ollama wire golden is unchanged.
5. **Plants** (`<scratch>/plant42_p2.json`): the same packages, run
   `_0042|_0041`.

   | Plant | File | `old` → `new` | Caught by |
   | :--- | :--- | :--- | :--- |
   | D2 removed | `menu.go` | `return !slices.ContainsFunc(e.Caveats, func(c Caveat) bool { return c != CaveatNoTools })` → `return len(e.Caveats) == 0` | S2, S9, M1; `TestMenu_Ollama_0041`; the provider and wizard S2 tests |
   | every caveat recommended | `menu.go` | the same `old` → `return true` | M2 |
   | the tool-less first | `menu.go` | `for _, e := range menu {` → `for _, e := range slices.Concat(sunk, uncaveated) {` | M1 |
   | D2 applied to every provider | `menu.go` | `if len(caveatsOf(provider, id, facts[id], now)) == 0 {` → `if !slices.ContainsFunc(caveatsOf(provider, id, facts[id], now), func(c Caveat) bool { return c != CaveatNoTools }) {` | `TestMenu_Caveats_0041`; `TestMenu_Snapshot_0041` (Hugging Face's 118) |
   | the limit ignored | `menu.go` | `Recommended: listlimit.Cut(recommended, limit)` → `Recommended: recommended` | M3, M4 |
| the filter removed, under D2 | `menu.go` | Phase 1's first plant, again | S1 and S8 through `List`, through the provider's `ListModels`, and through the wizard's blank search: the embedding models carry only `no-tools`, so D2 would recommend them |

6. **Docs and records.**
   * `docs/architecture.md`:
     * "A listing in which every model has a caveat stays live, with an
       empty `Recommended`" (about line 172) adds "(on Ollama, every model
       with a caveat other than `CaveatNoTools`)".
     * The Discovery bullet's "`Recommended` is the menu less the
       caveated" adds "(Ollama keeps its tool-less models: 0042-MADR D2)".
     * The Ollama bullet's "follows the others, out of `Recommended`"
       becomes "follows the others, and stays in `Recommended`".
   * `0041-MADR`:
     * a new amendment, "Amendment 2026-10-NN: Ollama's tool-less models
       stay recommended", with the next free `A` number after Phase 5's.
       It records the finding (0042-MADR F3), the decision (0042-MADR D2)
       and what it changes, and links 0042-MADR.
     * Annotations, not rewrites:
       * D3's "entries with a caveat never appear in it" gains "*(except
         Ollama's `CaveatNoTools`: A<n>, 0042-MADR D2)*".
       * D3's "Membership" gains "*(Ollama's usability filter: 0042-MADR
         D1)*".
       * A4's Ollama bullet, "with the entries `/api/show` marks as taking
         no tools after the others", gains "*(and in `Recommended`: A<n>)*".
   * `0041-PLAN`, under Phase 4's execution record: a dated note that
     0042-PLAN Phase 2 changed `TestMenu_Ollama_0041`'s `Recommended`
     assertion, citing 0042-MADR D2.
7. **Checks, record and stage,** as in Phase 1. The owner commits.

### Phase 3: the live check, and closing the records

1. **The live test,** `llmprovider/catalog/live_ollama_0042_test.go`
   (`//go:build live_gateways`): `TestLive_OllamaWritesText_0042`.
   * It skips unless `GET http://localhost:11434/api/tags` answers within
     1 s, as `ollamaModel` does (`llmprovider/live_0031b_measure_test.go`).
     Ollama takes no key, and nothing it sends is billed or generates.
   * It reads `fetchOllamaTags`, then `fetchOllamaShow` for each model.
     A failed reply is logged and counted, not fatal. It then calls `List`
     with `WithoutModelMetadata`.
   * It asserts:
     1. every id in `Usable` has merged facts for which `writesNoText` is
        false;
     2. every installed name left out of `Usable` has a `/api/show` or
        `/api/tags` reply that says it writes no text;
     3. wherever both endpoints report capabilities,
        `writesNoText(ollamaTagInfo(tag))` equals
        `writesNoText(showInfo)`;
     4. every `Usable` entry whose only caveat is `CaveatNoTools` is in
        `Recommended`.

     It logs the counts: installed, left out, failed replies, tool-less
     recommended.
2. **Red for the live test** (`<scratch>/plant42_p3.json`, with
   `"command": ["go", "test", "-count=1", "-tags", "live_gateways", "-run", "TestLive_OllamaWritesText_0042", "./llmprovider/catalog/"]`):
   * **`/api/tags` says no model writes text:** in `ollamaTagInfo`,
     `b.info.Output = ollamaOutputs(t.Capabilities)` →
     `b.info.Output = nil`. Assertion 3 catches it on any install of
     Ollama v0.30.0 or later: the local 0.33.2 lists capabilities for all
     eight models (0042-MADR E5).
   * **The filter removed** (Phase 1's first plant). Assertion 1 catches
     it only when a non-text model is installed. The local install has
     none (0042-MADR E5). With the owner's approval in that turn:
     * read the size of the smallest embedding model on its ollama.com
       library page;
     * `ollama pull` it, run the plant, then `ollama rm` it.

     Pulling changes a local service, so it waits for that approval. If
     declined, the record says assertion 1 ran with nothing to exclude,
     and that this plant was not seen to fail live.
   * Plants that skip because no Ollama answers count as not caught. The
     tool reports them, and the run is repeated with Ollama up.
3. **A live run, green:**
   `go test -count=1 -tags live_gateways -run TestLive_OllamaWritesText_0042 -v ./llmprovider/catalog/`.
   Its log goes into the execution record. No host or account name is
   kept (`tools/denyscan.py --changed`).
4. **`go vet -tags live_gateways ./...`** is clean. CI vets the live
   files.
5. **Records.**
   * This PLAN: `status: complete`, and each acceptance criterion below
     with its evidence.
   * `docs/README.md`: this PLAN's status.
   * 0042-MADR: its Confirmation is satisfied. No status change.
6. **The issue.** Draft a comment for issue 1. It names the commits and
   the behaviour of S1, S2 and S8, and says the fix ships in the next
   release. The agent posts it, and closes the issue on that release, only
   on the owner's ask in the same turn.
7. **Checks and stage,** as before. The owner commits.

## Verification

| # | Acceptance criterion | Evidence |
| :--- | :--- | :--- |
| AC1 | No model whose known capabilities lack `completion` is in Ollama's `Usable`, `Menu` or `Recommended` | `TestOllamaUsable_0042` S1, S3, S6, S7, S8; live assertion 1 |
| AC2 | A model with unknown capabilities is kept | S4, S5 |
| AC3 | `/api/tags` decides where `/api/show` failed, and is never cached | S4b, with its cache assertion |
| AC4 | A caller layer decides over Ollama's reply | S9 |
| AC5 | `LookupModel` and `CachedFacts` report Ollama's `Output` | `TestLookupModel_Listings_0033`'s Ollama row |
| AC6 | Tool-less chat models are recommended after the tool-capable ones, on Ollama only | S2, M1 to M4; `TestMenu_Caveats_0041` and `TestMenu_Snapshot_0041` unchanged |
| AC7 | `ListModels` and the wizard follow | the provider and wizard tests of Phases 1 and 2 |
| AC8 | No request added | `/api/show` is asked once per installed model not cached, as at 35c2e47, in S1 to S9 (0042-MADR section 3 for S1 to S7) |
| AC9 | No exported change | `tools/apidiff.py <base>` |
| AC10 | Every new assertion seen to fail | the plant tables' runs, in the execution record |
| AC11 | The wire golden unchanged | `TestWireGoldens` with no `-update` |
| AC12 | Endpoints agree on a real install | live assertion 3, logged |
| AC13 | The gate | 23 of 23 at each phase |

## Rollout and Rollback

* **Rollout.**
  * Three commits, one per phase, plus the records commit.
  * No release before Phase 2 (0042-MADR owner decision 2). 0041-PLAN
    Phase 9 cuts the release that carries both plans.
  * prepare-commit-msg then moves its pin and drops its README workaround,
    in its own repository.
* **Rollback.**
  * The owner reverts a phase's commit.
  * **Phase 2 alone:** Ollama returns to 0041-MADR A4. The tool-less
    regression (0042-MADR F3) returns with it.
  * **Phase 1:** the issue returns.
  * Nothing persists. The listing cache is in-process, and no file or
    setting format changes.

## Evidence index

<!-- claims: self=35c2e47 -->

Each row is checked with
`python3 -B tools/claims.py docs/decisions/0042-PLAN-ollama-models-that-write-text.md`,
against 35c2e47.

| # | What | Where | Text |
| :--- | :--- | :--- | :--- |
| P1 | Ollama's lister | `llmprovider/catalog/discovery.go:567` | `func ollamaCatalog(ctx context.Context, token llmprovider.Token, cfg config) (listingResult, error) {` |
| P2 | the tags fetch | `llmprovider/catalog/discovery.go:589` | `func fetchOllamaNames(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {` |
| P3 | the Ollama branch of the catalog | `llmprovider/catalog/discovery.go:253` | `return ollamaMenu(l.usable, factsOf(id, cfg, l.usable, models, nil), cfg.limit, now)` |
| P4 | the lister's result | `llmprovider/catalog/discovery.go:233` | `type listingResult struct {` |
| P5 | what a listing caches | `llmprovider/catalog/discovery.go:171` | `storeListing(listingKeyFor(id, cfg, token), models, false)` |
| P6 | `Recommended`'s rule | `llmprovider/catalog/discovery.go:64` | `// ids less the caveated ones. It may be empty on a live listing, when` |
| P7 | `Usable`'s rule | `llmprovider/catalog/discovery.go:68` | `// Usable is every id the provider's usability filters admit, once each,` |
| P8 | `/api/show`'s reader | `llmprovider/catalog/listing_metadata.go:375` | `func ollamaInfo(model string, show ollamaShowReply) ModelInfo {` |
| P9 | an empty list read as known | `llmprovider/catalog/listing_metadata.go:385` | `if show.Capabilities != nil {` |
| P10 | the fields it marks | `llmprovider/catalog/listing_metadata.go:392` | `b.set(FieldTools, FieldInput, FieldReasoning)` |
| P11 | the no-text caveat | `llmprovider/catalog/menu.go:61` | `if info.Has(FieldOutput) && !slices.Contains(info.Output, ModalityText) {` |
| P12 | Ollama's menu | `llmprovider/catalog/menu.go:166` | `func ollamaMenu(names []string, facts map[string]ModelInfo, limit int, now time.Time) Catalog {` |
| P13 | its `Recommended` and `Usable` | `llmprovider/catalog/menu.go:176` | `return Catalog{Recommended: listlimit.Cut(menuIDsOf(uncaveated), limit), Usable: names, Live: true,` |
| P14 | the helper Phase 2 removes | `llmprovider/catalog/menu.go:181` | `func menuIDsOf(entries []MenuEntry) []string {` |
| P15 | a failed reply's effect | `llmprovider/catalog/menu.go:203` | `// reply leaves its model out, with no caveat (0041-MADR A4).` |
| P16 | the caveat's rule | `llmprovider/catalog/menu.go:15` | `// does not drop the model: it sinks it to the end of Catalog.Menu, and keeps` |
| P17 | `CaveatNoTools`'s doc | `llmprovider/catalog/menu.go:23` | `// CaveatNoTools is a model that takes no tools.` |
| P18 | the field-by-field merge | `llmprovider/catalog/lookup.go:197` | `func overlay(dst *ModelInfo, src ModelInfo, name string) {` |
| P19 | the fetch's other caller | `llmprovider/catalog/read_limit_test.go:35` | `fetchOllamaNames(ctx, token, cfg)` |
| P20 | `LookupModel`'s Ollama fields | `llmprovider/catalog/lookup_0033_test.go:183` | `[]Field{FieldContextWindow, FieldTools, FieldInput, FieldReasoning},` |
| P21 | the Phase 4 assertion D2 changes | `llmprovider/catalog/menu_0041_test.go:329` | `if want := []string{"b:1", "a:1"}; !slices.Equal(cat.Recommended, want) {` |
| P22 | the guard for other providers | `llmprovider/catalog/menu_0041_test.go:118` | `func TestMenu_Caveats_0041(t *testing.T) {` |
| P23 | `ListModels` returns `Recommended` | `llmprovider/providers/ollama/ollama.go:298` | `listed := cat.Recommended` |
| P24 | the golden's listing has no capabilities | `llmprovider/providers/ollama/wire_test.go:19` | `{"models":[{"name":"llama3.3:latest"},{"name":"qwen3:8b"}]}` |
| P25 | the golden's server routes `/api/tags` | `llmprovider/internal/wirecase/wirecase.go:221` | `path == "/api/tags"):` |
| P26 | and answers `/api/show` 404 | `llmprovider/internal/wirecase/wirecase.go:238` | `no canned reply for this path` |
| P27 | the wizard's empty-listing rule | `wizard/configure.go:459` | `if len(cat.Recommended) == 0 && (!cat.Live` |
| P28 | the wizard searches `Usable` | `wizard/model_select.go:61` | `matches := catalog.Search(d.ID, cat.Usable, q)` |

## Execution Record

No phase has run. Each phase appends its record here: the research
results, the red and green output, the plant runs, the checks' output,
and a dated entry for every deviation.
