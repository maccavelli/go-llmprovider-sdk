---
status: accepted
date: 2026-10-14
decision-makers: repository owner
consulted: GitHub issue 1 and its reporter's downstream, prepare-commit-msg (a local checkout at v1.9.0 plus two documentation commits); Ollama's source through the GitHub API at the tags v0.6.4, v0.30.0, v0.33.2 and v0.40.2 and at main 8e4dad49f9f5; a local Ollama 0.33.2 with eight installed models; the ollama.com library's embedding search; reproductions on scratch clones of this repository at v1.12.0, v1.13.0, 7f34e31, 35c2e47 and the working tree (below)
informed: every caller of `catalog.List` and of the Ollama provider's `ListModels`; prepare-commit-msg and gobble-cli first
---

<!-- markdownlint-disable MD013 -->

# List Only the Ollama Models That Write Text, and Keep the Tool-less Ones Recommended

## Context and Problem Statement

[Issue 1](https://github.com/maccavelli/go-llmprovider-sdk/issues/1),
"catalog: Ollama listing recommends embedding models", was opened on
2026-10-14 against v1.12.0. `catalog.List` for Ollama recommends every
installed model in `/api/tags` order, embedding models included. A caller
that takes the first recommended model, or the next few as fallbacks, can
therefore get a model that cannot write text. prepare-commit-msg's
`configure --yes --provider ollama` did both. With `granite-embedding:30m`
installed and listed first:

* with `--model`, that model became a fallback;
* without `--model`, it became the model.

The issue asks the SDK to recommend, and count as usable, only the models
whose `capabilities` include `completion`. It leaves three questions to the
record that decides it:

1. What to do with a model whose `/api/show` fails, or reports no
   capabilities.
2. What one `/api/show` per installed model costs.
3. Whether `Usable` keeps the full list.

This record measures each claim the issue makes, against every release and
against the unreleased work since. Every number below comes from a command
listed in More Information, "Evidence index".

### 1. What each version does

| Ref | What it is | Ollama's `Recommended` | `/api/show` from `List` |
| :--- | :--- | :--- | :--- |
| v1.0.0 to v1.10.2 | 18 releases, v1.3.2 among them | the first `MaxListed` names from `/api/tags` | none |
| v1.11.0 to v1.12.0 | 2 releases; v1.12.0 is the issue's version and prepare-commit-msg's pin | `listlimit.Cut(names, cfg.limit)`: every name, or the first n with `WithLimit(n)` | none |
| v1.13.0 | latest release (2026-10-13) | as v1.12.0 | none |
| 7f34e31 | `origin/main` | as v1.12.0 | none |
| 35c2e47 | local `main`, committed, **not pushed** (0041-PLAN Phase 4) | the names without a caveat, in `/api/tags` order | one per uncached model, four at a time |
| working tree | 0041-PLAN Phase 5, uncommitted | as 35c2e47 | as 35c2e47 |

* **Up to 7f34e31.** `ollamaCatalog` returns the names unchanged
  (`llmprovider/catalog/discovery.go:492` at v1.12.0). Its body was read at
  each of the 21 release tags. It takes three forms, and none of them
  filters a name. `catalog.Static`
  returns `nil` for Ollama at every ref, as the issue says
  (`models_catalog.go`, "Installed models are machine-specific"). v1.12.0
  reads `/api/show` only for `LookupModel` (`listing_cache.go:123`), never
  for `List`.
* **At 35c2e47.** 0041-MADR amendment A4 decided that `List` asks
  `/api/show` for each installed model whose reply is not cached
  (`ollamaShows`, `menu.go:204`). An entry that takes no tools then sinks,
  and leaves `Recommended` (`ollamaMenu`, `menu.go:166`).
* **The facts read.** `ollamaInfo` (`listing_metadata.go:375`) reads
  `tools`, `vision` and `thinking` from the reply's capabilities. It never
  reads `completion` or `embedding`, and it sets no `Output` modality.

### 2. What Ollama reports

* **The capability values** at main 8e4dad49f9f5
  (`types/model/capability.go`): `completion`, `tools`, `insert`,
  `vision`, `embedding`, `thinking`, `image`, `audio` and `decision`. The
  same list stands at v0.33.2, without `decision`, which v0.40.2 has.
* **How `completion` is decided** (`server/images.go`,
  `ggufCapabilities`, line 207 at main, line 147 at v0.33.2):
  * main: a GGUF model with `decision.type` metadata gets `decision`;
    with `pooling_type` it gets `embedding`; otherwise it gets
    `completion`.
  * v0.33.2: the same, without the `decision` case.
  * `filterUnsupportedCapabilities` (line 495 at main) removes
    `completion`, `insert`, `tools` and `thinking` from a decision model.
  * A non-GGUF model gets `completion` only from its config
    (`configCapabilities`). `ggufCapabilities` returns early for it.
  * A safetensors model created with none gets `["completion"]`
    (`server/create.go`, `writeSafetensorsManifest`).
  * Ollama's own cloud-model fixture carries `"capabilities": ["completion"]`
    (`server/routes_cloud_test.go:61`).
* **An embedding model can report `tools`.** At both refs,
  `chatTemplateCapabilities` appends `tools` when the chat template
  contains `tools` or `tool_call`. It runs before the pooling check.
  `filterUnsupportedCapabilities` removes only `audio` and `vision` at
  v0.33.2 (line 460), and only a decision model's text capabilities and
  `audio` at main. So nothing removes `tools` from an embedding model.
  A reply of `["tools", "embedding"]` is therefore possible. No real
  model was observed doing so.
* **`/api/show` has carried `capabilities` since v0.6.4** (absent at
  v0.6.3). The field is `omitempty` (`api/types.go:822` at main), so an
  empty list is never sent.
* **`/api/tags` has carried `capabilities` since v0.30.0** (absent at
  v0.24.0; no stable release lies between). At main its entries come from
  the same `publicCapabilities()` as `/api/show`
  (`server/model_list.go:210`; `server/routes.go:1927`). A model that will
  not load gets no capabilities in `/api/tags` (`model_list.go:206`).
* **On a local Ollama 0.33.2, the two endpoints differed** for three of
  eight installed models. They agreed on `completion` for all eight:

  | Model | `/api/tags` | `/api/show` |
  | :--- | :--- | :--- |
  | `llama3.2:latest` | completion, tools | same |
  | `granite3-dense:8b` | completion, tools | same |
  | `granite3-dense:8b-instruct-q5_K_M` | completion, tools | same |
  | `qwen3:8b` | completion, tools, thinking | same |
  | `gpt-oss:20b` | completion, tools, thinking | same |
  | `gemma4:latest` | completion, tools, thinking | completion, vision, audio, tools, thinking |
  | `deepseek-r1:7b` | completion, thinking | tools, thinking, completion |
  | `gemma3:12b` | completion | completion, vision |

  The 0.33.2 listing path was not read, so this record does not explain
  the difference. It only measures it.
* **A probe by request loads the model.** Ollama documents that "If an
  empty prompt is provided, the model will be loaded into memory"
  (`docs/api.md:436`; for chat, `docs/api.md:1115`).
* **A name rule misses some.** The ollama.com library lists 13 embedding
  models. The substring `embed`, the SDK's `denyEmbed` for the other
  providers, catches 9 of them and misses 4: `bge-m3`, `all-minilm`,
  `paraphrase-multilingual` and `bge-large`.

### 3. The reproduction

Each case below runs `catalog.List` for Ollama against a fake server, with
the options prepare-commit-msg passes (`WithLimit(MaxListed)`, metadata
off). Some capability values were seen, and the rest are fixtures:

* **From the issue (Ollama 0.40.2):** `granite-embedding:30m` and
  `qwen2.5-coder:7b`.
* **Observed locally (0.33.2):** `gemma3:12b`.
* **Fixtures:** every other model. S3 and S6 are built from the code paths
  in section 2.

| Case | Installed, with `/api/show` capabilities | v1.12.0 = v1.13.0 = 7f34e31: `Recommended` | 35c2e47 = working tree: `Recommended` | 35c2e47: what else changed |
| :--- | :--- | :--- | :--- | :--- |
| S1, the issue | `granite-embedding:30m` [embedding], `qwen3-vl:8b` [completion vision tools thinking], `llama3.1:8b-instruct-q5_K_M` [completion tools], `qwen2.5-coder:7b` [completion tools insert] | all four, the embedding model first | the three chat models | the embedding model is still in `Usable` and in `Menu`, with caveat `no-tools`; 4 `/api/show` requests |
| S2, chat models without tools | `gemma3:12b` [completion vision], `codellama:7b` [completion insert] | both | **none** | both carry `no-tools`; `Usable` has both |
| S3, embedding model that reports tools | `embed-tools:1` [tools embedding], `chat:1` [completion tools] | both | **both** | no caveat on the embedding model |
| S4, `/api/show` fails (HTTP 500) | `granite-embedding:30m`, `chat:1` [completion tools] | both | both | no caveat on the model whose reply failed |
| S5, no capabilities field (Ollama before v0.6.4) | `nomic-embed-text:latest`, `chat:1` | both | both | no caveats |
| S6, image and decision models | `x/flux2:1` [image], `classifier:1` [decision], `chat:1` [completion tools] | all three | `chat:1` | the two others carry `no-tools` |
| S7, `/api/tags` carries capabilities | `granite-embedding:30m` [embedding], `qwen2.5-coder:7b` [completion tools insert], in both endpoints | both | `qwen2.5-coder:7b` | 2 `/api/show` requests; the listing's capabilities are not read |

`Usable` is every installed name, in `/api/tags` order, at every ref and in
every case. The output of v1.13.0 and of 7f34e31 is identical to v1.12.0's,
line for line, apart from `go test`'s timing lines. The working tree's is
identical to 35c2e47's in the same way.

### 4. Findings

* **F1. The issue holds in every release, and on `origin/main`.**
  * S1 reproduces at v1.12.0, v1.13.0 and 7f34e31.
  * The source of every release from v1.0.0 to v1.11.0 recommends the
    installed names unfiltered (section 1).
  * The issue's "the same in v1.3.2" is one case of this.
* **F2. 35c2e47 drops embedding models from `Recommended`, but for the
  wrong reason.** They leave because they take no tools, so the caveat
  reads `no-tools` and not `no-text-output`. They stay in `Usable` and in
  `Menu`. The wizard searches `Usable` (`wizard/model_select.go:61`), so a
  search still offers them. S6's image and decision models are treated
  the same way.
* **F3. 35c2e47 adds a regression.** It empties `Recommended` when no
  installed chat model takes tools (S2), where v1.13.0 recommends them
  all. Locally, `gemma3:12b` reports `completion` without `tools`. Three
  callers depend on `Recommended`:
  * prepare-commit-msg falls back to `catalog.Static`, which is `nil` for
    Ollama (`internal/ui/setup.go:176`, `:198`). `configure --yes` then
    fails with "no live models listed for ollama; pass --model"
    (`setup.go:447`).
  * The Ollama provider's own `ListModels` returns `cat.Recommended`, and
    nothing when it is empty
    (`llmprovider/providers/ollama/ollama.go:296-301`).
  * The wizard warns "no %s model meets the profile; search the listing
    for one" (`wizard/model_select.go:23`, `:50-56`).

  ProfileUtility, the zero value and prepare-commit-msg's profile, is
  documented as "short, frequent tasks such as commit messages"
  (`model_profile.go:8` at 35c2e47). Such tasks send no tools:
  prepare-commit-msg's Go source sets no tool (`git grep` for
  `llmprovider.Tool`, `Tools:`, `ToolChoice` and `FunctionTool` finds
  none).
* **F4. A `tools` rule is the wrong test for "writes text".** It misses an
  embedding model that reports `tools` (S3). It also removes chat models
  that write text but take no tools (F3).
* **F5. Unknown capabilities keep the model, at 35c2e47 and before**
  (S4, S5). That is the issue's own answer to its first question.
* **F6. What one `/api/show` per model costs.** At 35c2e47, a listing
  sends one local, unbilled `/api/show` per installed model that is not
  cached, four at a time. It caches each reply
  (`TestMenu_Ollama_0041`, `menu_0041_test.go:314`, asserts 3 requests,
  then none more on a second listing). Since v0.30.0, `/api/tags` already
  says whether each model has `completion`, at no extra request (S7).
  Since 35c2e47 sends `/api/show` anyway for the tools fact, a
  `completion` test adds no request, whichever endpoint it reads.
* **F7. `Usable` holding embedding models contradicts the SDK's own
  documentation, and serves no SDK call.**
  * `List`'s doc says it curates "so configure UIs never show huge
    unusable lists (embeddings, …)" (`discovery.go:91-93` at 35c2e47).
  * `Catalog.Usable` is "every id the provider's usability filters admit"
    (`discovery.go:68`).
  * Every other provider's filter drops embeddings by name (`denyEmbed`,
    `models_catalog.go:173`).
  * The SDK has no embeddings API (no `Embed` function, no `/api/embed`
    call in `llmprovider/`). An embedding model in `Usable` cannot be used
    through the SDK.

## Decision Drivers

* **Correct for the issue's case.** No model that cannot write text is
  recommended, or offered to a configure UI's search.
* **No regression for tool-less users.** An install whose chat models
  take no tools still gets them recommended, as v1.13.0 does (F3).
* **Facts, not names** (0041-MADR). Ollama states which models complete
  text. A name rule misses 4 of the library's 13 embedding models.
* **Never lose a usable model.** A model whose capabilities are unknown
  is kept (the issue; F5).
* **One meaning of `Usable` across providers.** "Admitted by the
  usability filters" means "usable for generation" everywhere (F7).
* **No new cost.** No extra request, no model loaded into memory.
* **Additive** (R48). No exported identifier changes. Only Ollama
  listings change.

## Considered Options

Which Ollama models are usable (D1):

* **U1. A capability filter.** A model whose reported capabilities lack
  `completion` is not usable. It leaves `Usable`, `Menu` and
  `Recommended`. A model whose capabilities are unknown is kept.
* **U2. A capability caveat.** The same test sets the model's `Output`
  facts, so the existing `CaveatNoTextOutput` fires. The model stays in
  `Usable` and `Menu`, sunk.
* **U3. A name rule.** Deny `embed` in Ollama names, as the other
  providers do.
* **U4. A probe by request.** Send an empty generate or chat request to
  each model, and keep those that answer.
* **U5. Keep 35c2e47's `CaveatNoTools` as the fix.**
* **U6. Nothing:** v1.13.0's behaviour.

Ollama's chat models that take no tools (D2):

* **T1. Keep 0041-MADR A4 as built.** They leave `Recommended`, and each
  consumer falls back to `Usable` itself.
* **T2. A no-tools model leaves `Recommended` only under `ProfileCapable`
  and `ProfileAgent`** (0041-MADR D5, uncommitted), for every provider.
* **T3. On Ollama, they stay in `Recommended`,** after the models that
  take tools. `Menu` still carries `CaveatNoTools`.
* **T4. Ollama sets no tools fact.** The no-tools caveat never fires on
  Ollama.

## Decision Outcome

Chosen option: **"U1. A capability filter"** with **"T3. On Ollama, they
stay in `Recommended`"**. It was proposed for the owner's review and
accepted on 2026-10-14 ("follow recommendations"; see "Owner decisions"
under More Information).

* **U1, because:**
  * it is the only option that removes every model Ollama says cannot
    complete text (S1, S3, S6, S7) from `Usable` and the menu, without
    dropping a model it cannot judge (S4, S5);
  * it costs no request (F6);
  * it gives `Usable` the meaning it has for every other provider (F7).
* **T3, because:**
  * it is the only option that removes F3's regression without changing
    any other provider's `Recommended`;
  * it keeps the tool fact visible, and keeps the tool-capable models
    first, for callers that send tools.

### D1. An Ollama model is usable when it writes text, or when that is unknown

* **The test.** An installed model is **not usable** when its reported
  capabilities are known and do not contain `completion`.
  * Such a model is left out of `Usable`, and so out of `Menu` and
    `Recommended`, in every case S1 to S7.
  * Ollama assigns `completion` to every GGUF model that is not an
    embedding or decision model (section 2). So the test removes
    embedding, image-generation and decision models, and no chat model.
* **Where the capabilities come from.**
  1. The model's `/api/show` reply, which 35c2e47 already fetches.
  2. When that reply fails or carries none, the model's `capabilities`
     in `/api/tags` (Ollama v0.30.0 on).
  3. Otherwise they are unknown.
  * An empty list counts as unknown. Ollama never sends one: the field
    is `omitempty`. *(0042-PLAN research, 2026-10-14: this covers the
    tools fact too. Today `ollamaInfo` reads `[]` as "takes no tools"
    (`show.Capabilities != nil`, `listing_metadata.go:385`), and after
    D1 it reads `[]` as nothing at all.)*
  * *(0042-PLAN research, 2026-10-14.)* The test reads the model's facts
    as merged for the menu (0041-MADR D1), so a caller's layer that sets
    `Output` decides over Ollama's reply, as it does for every other
    field.
  * `tools`, `vision` and `thinking` keep coming from `/api/show` only.
    The two endpoints differed on those for three of eight local models,
    and agreed on `completion` for all eight (section 2).
* **Unknown keeps the model**, with no caveat. That covers a failed reply
  (S4), an Ollama before v0.6.4 (S5), and a model that will not load. This
  is the issue's own answer, and the "never lose a usable model" driver.
* **The fact is recorded.** `ollamaInfo` sets `Output` and marks
  `FieldOutput` when it has capabilities:
  * `[text]` when they contain `completion`;
  * `[image]` when they contain `image` and not `completion`;
  * empty otherwise.

  *(0042-PLAN research, 2026-10-14: generalised to text for `completion`
  and an image for `image`, so a model with both writes `[text, image]`
  rather than dropping the image. Each of the three cases above gives
  the same result as before.)*

  `LookupModel` and `CachedFacts` then say whether a model writes text.
  `CaveatNoTextOutput` fires for a model a caller looks up by name, though
  the listing never shows one.
* **The answers to the issue's questions:**
  1. Unknown keeps the model.
  2. The test sends no request beyond 35c2e47's. On v0.30.0 and later,
     `/api/tags` answers when `/api/show` fails.
  3. `Usable` is filtered as well as `Recommended`, as every other
     provider's is.
* **What is not done:** skipping `/api/show` for a model whose
  `/api/tags` capabilities lack `completion`.
  * It would save one local, unbilled request per non-generation model.
  * It would make D1's answer depend on which endpoint was read first.
  * It is left out to keep one rule.

### D2. On Ollama, a chat model that takes no tools stays recommended

* **What changes.** Ollama's `Recommended` is the usable models in
  `/api/tags` order:
  * first those without a caveat;
  * then those whose only caveat is `CaveatNoTools`.
* **What does not change.**
  * `Menu` keeps 35c2e47's order and caveats, less the models D1
    removes.
  * A model with any other caveat stays out of `Recommended`, as
    0041-MADR D3 has it. On Ollama, only a caller's layer
    (0041-MADR D1), or an id containing `-contributor`
    (`excludedAtRequest`), can raise one.
  * Every other provider is unchanged.
* **Why Ollama, and not every provider.**
  * Ollama's listing is the user's own install, with no band and no other
    model to offer. Dropping the tool-less models leaves nothing (F3).
  * Elsewhere, dropping them leaves other models. On the committed
    2026-10-12 snapshot, the no-tools caveat removes one model from all
    five open catalogs' `ProfileUtility` lists: Hugging Face's
    `zai-org/AutoGLM-Phone-9B-Multilingual`.
* **What it amends.** 0041-MADR D3's rule "entries with a caveat never
  appear in it", for Ollama's `CaveatNoTools` only. It also amends the
  `Catalog.Recommended` doc ("It is Menu's ids less the caveated ones").
  If this record is accepted, 0041-MADR gets an amendment that points
  here.

### Consequences

* Good, because S1 recommends `qwen3-vl:8b`, `llama3.1:8b-instruct-q5_K_M`
  and `qwen2.5-coder:7b`, and `Usable` holds the same three. Neither
  prepare-commit-msg nor the wizard can take the embedding model.
* Good, because S2 recommends `gemma3:12b` and `codellama:7b`, as v1.13.0
  does. prepare-commit-msg's `configure --yes` and `ListModels` work for
  tool-less installs.
* Good, because S3's embedding model, which reports `tools`, is removed.
  So are S6's image and decision models.
* Good, because no request is added, and S4 and S5 keep every model.
* Neutral, because these outcomes are predicted by the rules, not
  measured. The PLAN's tests pin them.
* Bad, because a user who wants an embedding model in the wizard can no
  longer find it by search. The wizard still offers "use the query
  itself" (0021-MADR Z11), and the SDK cannot run such a model anyway
  (F7).
* Bad, because Ollama's `Recommended` is no longer `Menu` less every
  caveated entry. A caller that sends tools must check `Caveats` itself,
  or rely on the tool-capable models coming first.
* Bad, because ~~the Ollama listing golden
  (`llmprovider/providers/ollama/testdata/wire/ollama/listing.json`) and~~
  `TestMenu_Ollama_0041`'s expectation changes. *(0042-PLAN research,
  2026-10-14: the golden does not change. Its fake server sends
  `/api/tags` with no capabilities (`providers/ollama/wire_test.go:19`)
  and answers `/api/show` 404 (`internal/wirecase/wirecase.go:238`), so
  both of its models stay unknown and kept.)*

### Confirmation

* **Unit tests.**
  * S1 to S7 as table cases against a fake Ollama in
    `llmprovider/catalog`, asserting `Recommended`, `Usable`, `Menu` order
    and caveats, and the request count.
  * The same for the Ollama provider's `ListModels`, with probes off.
  * `ollamaInfo` sets `Output` for `completion`, `image` and `embedding`
    replies, and leaves it unknown without capabilities.
* **Plants** on a scratch clone, each caught by an assertion:
  * the `completion` test removed;
  * unknown treated as unusable;
  * the `/api/tags` fallback ignored;
  * D2's tool-less models left out of `Recommended`;
  * D2 applied to a non-Ollama provider.
* **A live-tagged test**, run when a local Ollama answers, as
  `TestLive_OllamaStreams` is. It asserts:
  * every model in `Usable` reports `completion`, or has no capabilities;
  * every installed model left out reports capabilities without
    `completion`;
  * `/api/tags` and `/api/show` agree on `completion`, wherever both
    report it.
* **Records:** `make records-check`, and the 0041-MADR amendment named
  in D2.

## Pros and Cons of the Options

### U1. A capability filter

* Good, because it removes the non-text models of S1, S3, S6 and S7, and
  only those.
* Good, because it keeps every model whose capabilities are unknown.
* Good, because it needs no request beyond 35c2e47's.
* Good, because `Usable` then means for Ollama what it means for every
  other provider.
* Bad, because embedding models leave the wizard's search.

### U2. A capability caveat

* Good, because it follows 0041-MADR D3's "a caveat does not remove the
  entry", and the right caveat fires.
* Bad, because the embedding models stay in `Usable`, so the wizard's
  search still offers them (F2), against `List`'s own documentation (F7).
* Bad, because a model the SDK cannot run is kept listed for no SDK use.

### U3. A name rule

* Good, because it needs no capability data, so it also works before
  Ollama v0.6.4.
* Bad, because it misses 4 of the library's 13 embedding models, and
  every image or decision model.
* Bad, because it guesses where Ollama states the fact.

### U4. A probe by request

* Good, because it measures the model directly.
* Bad, because each probe loads a model into memory (Ollama's
  `docs/api.md:436`). For a whole install, that costs memory and time
  inside `List`'s 10 s bound.
* Bad, because it answers no better than `completion` does.

### U5. Keep 35c2e47's `CaveatNoTools` as the fix

* Good, because it is already built.
* Bad, because S3's embedding model, which reports `tools`, is still
  recommended.
* Bad, because the caveat states the wrong reason, and the models stay
  searchable (F2).
* Bad, because it empties `Recommended` for tool-less installs (F3).

### U6. Nothing

* Bad, because the issue stands in cases S1, S3, S6 and S7.

### T1. Keep 0041-MADR A4 as built

* Good, because 0041-MADR D3 stays uniform across providers.
* Bad, because every consumer has to change, prepare-commit-msg and the
  Ollama provider's own `ListModels` among them. Until each does,
  v1.13.0's working case S2 fails.

### T2. No-tools leaves `Recommended` only under the tool profiles

* Good, because the profile's stated use decides. ProfileUtility names
  commit messages.
* Neutral, because on the 2026-10-12 snapshot it changes no open
  catalog's six, and adds one model to Hugging Face's full list under
  ProfileUtility (118 to 119). The four first-party providers were not
  measured.
* Bad, because a ProfileUtility caller that sends tools could get a
  tool-less model as its default, from a future band.
* Bad, because it re-opens 0041-MADR D3 for every provider, for a defect
  seen only on Ollama.

### T3. Ollama keeps them in `Recommended`, after the tool-capable ones

* Good, because S2 works as in v1.13.0, and a tool-capable model is first
  wherever one is installed.
* Good, because it changes no other provider.
* Bad, because Ollama becomes an exception to "a caveated entry is never
  recommended".

### T4. Ollama sets no tools fact

* Good, because it is simple.
* Bad, because it discards a fact Ollama states, along with the order
  and the caveat that 0041-MADR A4 chose to show.

## More Information

### Owner questions

1. **Where this lands.** Is this record 0042, or an amendment to
   0041-MADR? It is drafted as 0042 because the issue is its own decision,
   what Ollama counts as usable, and is open against the releases. D2
   amends 0041-MADR A4, an unreleased owner decision of 2026-10-14. Either
   way, the work touches `discovery.go`, which 0041-PLAN Phase 5's
   uncommitted changes also edit. The implementation is best ordered
   after Phase 5 is committed.
2. **Release order.** 35c2e47 is not pushed. Releasing it before D2 ships
   F3's regression to prepare-commit-msg, which says it "will move its pin
   once a release filters the listing".
3. **T2 instead of T3?** If the profile-scoped rule is preferred, the
   four first-party providers need the measurement T2 lacks, before it is
   accepted.

### Owner decisions (2026-10-14)

The owner answered "follow recommendations":

1. **This record stays 0042.** D2 is recorded in 0041-MADR as an
   amendment that points here.
2. **35c2e47 is not released without D2.** No tag is cut between
   0041-PLAN Phase 4 and 0042-PLAN Phase 2.
3. **T3, not T2.**
4. **The implementation follows 0041-PLAN Phase 5's commit.** It is
   [0042-PLAN-ollama-models-that-write-text.md](0042-PLAN-ollama-models-that-write-text.md).

### Evidence index

Each scratch script is Python, and ran on a scratch clone or read-only
against a service. None of them is committed. The PLAN commits the
reproduction as tests.

| # | What | How | Result |
| :--- | :--- | :--- | :--- |
| E1 | The issue's text | `gh issue view 1 --json …` | quoted in the Context |
| E2 | `ollamaCatalog` at all 21 release tags and at 7f34e31 | `git show <ref>:<file>` for each non-test file in `llmprovider/catalog/`, grouping identical bodies | section 1: three forms (v1.0.0; v1.1.0 to v1.10.2; v1.11.0 to v1.13.0), none filtering |
| E3 | Ollama's capability source | `gh api repos/ollama/ollama/contents/<path>?ref=<ref>`, for `types/model/capability.go`, `server/images.go`, `server/model_list.go`, `server/routes.go`, `server/create.go`, `api/types.go`, `docs/api.md` and `server/routes_cloud_test.go`. Each file read at main was compared byte for byte with the same file at 8e4dad49f9f5; the cloud test was re-read at that commit. | section 2 |
| E4 | The first release with each field | binary search over the 261 stable `v0.x.y` tags, testing `api/types.go`'s `ListModelResponse` and `ShowResponse` for `Capabilities` | `/api/show`: v0.6.4 (absent at v0.6.3); `/api/tags`: v0.30.0 (absent at v0.24.0, the stable release before it) |
| E5 | A local Ollama 0.33.2 | `GET /api/tags`, then `POST /api/show` for each of its 8 models; read-only, nothing pulled | the table in section 2: 5 of 8 identical, `completion` agreed in 8 of 8 |
| E6 | The library's embedding models | `https://ollama.com/search?c=embedding`, names parsed from its links | 13 models; `embed` catches 9 |
| E7 | The reproduction S1 to S7 | an external test in `package catalog_test`, copied into a scratch clone of each ref and of the working tree; `go test -run TestIssue1Repro -v ./llmprovider/catalog/` | section 3; apart from timing lines, v1.13.0 and 7f34e31 identical to v1.12.0, and the working tree identical to 35c2e47 |
| E8 | The no-tools caveat's reach | on a scratch clone of the working tree, the 2026-10-12 snapshot replayed under `ProfileUtility`, with and without the `CaveatNoTools` lines of `caveatsOf` | six unchanged on all five open catalogs; only Hugging Face's full list changes, 118 to 119 |
| E9 | prepare-commit-msg's use | read at a04b617: `go.mod` pins v1.12.0; `internal/ui/setup.go:153-178, 198, 447`; README lines 578-581 document the workaround | F3 |

### Related records

* [0041-MADR-decide-per-model-behaviour-from-metadata.md](0041-MADR-decide-per-model-behaviour-from-metadata.md):
  D3, the menu and its caveats, and amendment A4, Ollama's `/api/show`.
  D2 here amends both, for Ollama.
* [0038-MADR-full-curated-catalog-by-default.md](0038-MADR-full-curated-catalog-by-default.md):
  D1, Ollama's installed models in listing order.
* [0021-MADR-harden-and-tune-after-the-v1-1-review.md](0021-MADR-harden-and-tune-after-the-v1-1-review.md):
  C13, an empty `Recommended` on a live listing; Z11, "use the query
  itself".
* [0009-MADR-use-case-aware-default-model-ranking.md](0009-MADR-use-case-aware-default-model-ranking.md):
  the profiles. ProfileUtility is for commit messages.
