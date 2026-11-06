---
status: accepted
date: 2026-11-06
decision-makers: repository owner
consulted: this module at b5ce51d (v1.26.1 is the last tag); grok-build at 2bdd1d6a, its remote's head on 2026-11-06; the installed Grok CLI 1.0.70 and its model cache; magic-cli-remote at 0f595dc9 and its proposed 0235-MADR; GitHub issue 11 of this repository, opened 2026-11-06; the live models.dev document, fetched 2026-11-06; prior art on version comparison (semver, PEP 440, dpkg, sort -V, Perl's version.pm, OpenRouter's newest sort); scratch probes on 2026-11-06, listed under More Information
informed: anyone who sorts a model listing by catalog.Rank, or shows catalog.Label outside a terminal menu
---

<!-- markdownlint-disable MD013 -->

# Read Each Model Family's Version Numbers from Its Dates and Siblings, and Give a Label's Name and Note Apart

## Context and Problem Statement

Two defects were found while magic-cli-remote ("Magic") planned to take
Grok's fallback list from this module. Its proposed
`docs/decisions/0235-MADR-save-the-last-live-model-catalog-and-take-groks-fallback-from-the-sdk.md`
(staged in that repository, not committed, on 2026-11-06) works around both.
It uses `Rank` only on the five ids of `Static(grok)`, never on Grok's live
list, and it shows ids instead of `Label`.

### 1. Rank reads grok-4.20 as a newer generation than grok-4.7

`catalog.Rank` (C1) sends a Grok id to `rankGrokModel` (C2). That function
names the generations 4.3 to 4.7 as hints (C3) and places any other id with
`generationScore` (C4) among the anchors (C5). `parseGeneration` reads a
minor of one or two digits (C6), so `grok-4.20-0309-non-reasoning` is
generation 4, minor 20, order key 420. That is above the newest anchor,
4.7 (key 407). `generationScore` then scores it 70 + 10 + min(40, 20 − 7) =
93 (C7).

A scratch program ran `catalog.Rank` and `catalog.Label` at `b5ce51d` on
every chat id in xAI's recorded listing. The dates are that listing's
`created` values (C17 to C19):

| id | `Rank` | xAI `created` |
| :--- | ---: | :--- |
| `grok-4.20-0309-non-reasoning`, `grok-4.20-0309-reasoning`, `grok-4.20-multi-agent-0309` | **93** | 2026-03-09 |
| `grok-4.3` | 45 | 2026-04-17 |
| `grok-4.5` | 50 | 2026-06-29 |
| `grok-4.6` | 60 | 2026-08-06 |
| `grok-4.7` | 70 | 2026-09-02 |
| `grok-build-0.1` | 0 | 2026-04-16 |

Sorted by `Rank`, the 4.20 ids come first, though xAI created them before
every other `grok-4.x` it lists.

The rule is deliberate. 0021-MADR's C6 finding made generations numeric
(C15), and Gemini's `gemini-3.10-flash` must rank above `gemini-3.9-flash`
(C11, C60). 0021-PLAN's live check saw this exact result on 2026-10-05: "4.20
scores 84, above 4.7's 71, though its date suggests it came before 4.3"
(C16). It accepted the result as C6 decided. Since then 0033-MADR D4 made 4.7
a hint and the newest anchor (C47), so the 4.20 ids rose from 84 to 93.

**Where it shows.**

* **A caller that sorts a Grok listing by `Rank`:** the 4.20 ids rank first.
  Outside its tests, the module never calls `Rank`; its callers are other
  programs.
* **Inside the module:** Grok's list keeps `staticGrok`'s five ids in their
  order. The rest of the listing follows, sorted by the ranker, stably (C8
  to C10). On the recorded listing the rest is the three 4.20 ids alone. All
  three score 93, so they keep the listing's order, and
  `grok-4.20-multi-agent-0309` is last of eight (C14). No list the module
  builds from that listing changes order under this record. The order would
  change if none of the five static ids were listed, or if xAI listed a
  newer id beside the 4.20 ones.

### 2. A version string does not say how to read it

`4.20` and `3.10` have the same shape. Read as an integer minor, 3.10
follows 3.9, which Gemini needs. Read as a decimal, 4.20 is 4.2, between 4.1
and 4.3, which is where xAI's dates put it. No rule that reads one string
alone can give both answers. Any rule of that kind is either today's
integer rule, or one that names the exceptions by hand.

**Prior art.** No project found infers the reading.

* semver, PEP 440, Debian's version comparison and `sort -V` declare each
  component an integer, so 1.10 follows 1.9.
* Perl's `version.pm` has both readings. A decimal version `1.10` is
  `v1.100.0`, below `1.9` (`v1.900.0`), while the dotted `v1.10` is above
  it. The author declares which one is meant, with a leading `v`.
* OpenRouter's models API sorts "newest" by creation date. models.dev
  publishes a `release_date` for each model.

**What the module already holds.** The two kinds of evidence a string
lacks:

* **Dates:** models.dev's `release_date` (C48) is parsed into
  `ModelInfo.Released` (C49, C50), and so is a listing's own date (C51). The
  module already treats a listing's dates as evidence that can be wrong: a
  date that every row shares is dropped as the listing's build date (C52).
  An unranked listing is ordered by them (C53). `CachedFacts` reads them
  with no fetch, but it takes a context and options (C54).
* **Siblings:** the other versions of the same family, in the static list
  and in the listing.

**The experiment.** A scratch script grouped every versioned id by provider,
family (the tokens before the version) and major. It read each group both
ways:

* the **integer** reading: a two-digit minor M sits at M, so 3.10 sits at 10;
* the **decimal** reading: it sits at M/10, so 4.20 sits at 2, and 4.25
  at 2.5.

A one-digit minor sits at the same place under both readings. Two rules
chose between them:

* **The date test:** among pairs of dated ids that the readings order
  differently, the reading that puts fewer against the dates. A pair that
  both readings order alike is not counted.
* **The gap test** (parsimony): the reading whose versions fill more of
  their span. Grok's `4.3, 4.5, 4.6, 4.7, 4.20` fill 5 of 18 slots as
  integers, and 5 of 6 as decimals.

The results:

| Data | Families | With a two-digit minor | Result |
| :--- | ---: | ---: | :--- |
| live models.dev, 2026-11-06 | 791 | 22 | dates decide 20, all decimal; the gap test agrees on all 20; 2 have too few dated pairs to decide |
| recorded models.dev (`ranking-2026-10-12`), Google, Anthropic, xAI | 12 | 1 (xAI's Grok) | 12 reading-sensitive pairs, all 12 against the integer reading, none against the decimal |

Of the 22 families, 21 are Grok, on xAI and on resellers and gateways. One
is Llama `3.05`, on a gateway. The two undecided families are Grok on two
resellers, each with too few dated versions. No Gemini family holds a
two-digit minor yet.

**Dates are noisy.**

* xAI dates 4.20 2026-03-09, while Kilo, OpenRouter and nano-gpt date it
  2026-03-31.
* Even under the better reading, the live document has 34 pairs whose date
  order disagrees with the version order.
* In the recorded Google data, 11 pairs disagree. These are later
  snapshots, previews and image variants of older generations, such as
  `gemini-3-pro-image` dated after the `gemini-3.1` previews. They disagree
  under both readings alike, so the date test does not count them: the
  reading-sensitive count there is 0.
* OpenAI's recorded listing has 8 such pairs: `gpt-3.5-turbo` snapshots
  dated after `gpt-4`. Its `gpt` family has no two-digit minor, so they too
  count alike under both readings.

So dates are sound evidence for choosing a reading. Used as the order
itself, they are not sound.

**Twelve constructed cases** were run through the layered rule: dates,
then gaps, then integer. They included Gemini 3.5 to 3.10, Gemini's static
ids plus 3.10, 3.9 and 3.10 alone, Python-style 3.8 to 3.14, Grok's static
ids plus 4.20, Grok 4.1 and 4.20 alone, and Llama 3.05. The rule gave the
expected answer in 11. The one it missed was built to fool the gap test: an
undated family that leaps from 3.5 to 3.12, with nothing between. The gap
test reads it as decimal, and it is meant as integer. With a date on each,
the date test reads it correctly. No rule that reads only strings can read
that family. That is why Perl has the author declare the reading.

### 3. A label is a padded terminal string, and two of its notes are wrong

`modelLabels` (C21) maps 20 ids, across five providers, to strings such as
`"Grok 4.7               [current flagship]"` (C24). A scratch check of the
table found every one of the 20 in a single form: the name, padded with
spaces to 23 columns, then the note in brackets. Every name is ASCII.
`Label` (C25) returns the whole string, or the bare id when the table has no
entry (C26). So a caller that does not draw a fixed-width menu has to parse
`Label` to get the name. The module parses it too: the search index cuts
each label at `[` with `displayName` to get the name it matches (C27, C28).
The wizard passes `Label` to `Prompter`, the rendering seam a consumer
implements over its own UI (C30, C31). `Choice` already has a `Detail`
field (C32).

Two notes are wrong today:

* **`grok-4.6`: "★ Recommended: the Grok CLI's default" (C22).** Issue 11
  reports this. The id is grok-build's compiled-in default (C42, C44), the
  last of five sources the CLI resolves a model from: "CLI flag, ENV var,
  config.toml, remote settings, these defaults" (C43). On 2026-11-06 the
  installed CLI's binary, 1.0.70, held `"default": "grok-4.6"` twice, and
  no other default. Magic's recorded 1.0.70 session with an xAI key reports
  `currentModelId` `grok-4.20-0309-non-reasoning` (C46). Issue 11 also
  reports `grok-4.6` for a signed-out home, which it measured. It reports
  `grok-4.7` for a signed-in account, which a user reported and it did not
  re-measure. This record re-measured neither. So the note presents the
  fallback as the CLI's default. The same claim is in `staticGrok`'s comment
  (C34) and in 0012-MADR §6 (C35, C36).
* **`grok-4.5`: "previous flagship" (C23)** is false by xAI's dates.
  `grok-4.6` (2026-08-06) comes between `grok-4.5` (2026-06-29) and
  `grok-4.7` (2026-09-02). `git log -S` shows how: the note was written on
  2026-10-05 (`1ba08e2`), when `grok-4.6` was labelled "current flagship".
  The refresh of 2026-10-11 (`3558c91`) moved "current flagship" to
  `grok-4.7` and left `grok-4.5`'s note as it was.

Five other notes are true on the recorded listings, but each one ranks its
model against others in time. Each will become false at a release, with no
edit to the model's entry:

| id | Note | Checked against |
| :--- | :--- | :--- |
| `grok-4.7` | current flagship | xAI's listing, newest `grok-4.x` |
| `gemini-3.8-flash` | newest Flash, frontier coding | Gemini's listing, highest `gemini-N.M-flash`, no dates |
| `gpt-5.6-luna` | previous efficient tier | OpenAI's listing, `gpt-6-luna` created later |
| `claude-sonnet-5-5`, `claude-opus-5-5` | latest Sonnet, latest Opus | Anthropic's listing, newest of each |

The star is a third case. Four labels open their note with "★
Recommended:". On `b5ce51d` each of the four is the first id of its
provider's `Static` list, and the first id of every other list with a label
has no entry (Together's first id is unlabelled). The star is written by
hand, so it can drift from the list order in the same way.

### Code claims

<!-- claims: self=b5ce51d grok-build=2bdd1d6a magic-cli-remote=0f595dc9 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | `Rank` is the exported ranking | `llmprovider/catalog/models_catalog.go:232` | `func Rank(provider llmprovider.ProviderID, model string) int {` |
| C2 | A Grok id goes to `rankGrokModel` | `llmprovider/catalog/models_catalog.go:241` | `return rankGrokModel(model)` |
| C3 | Grok's hints; 4.7 is the newest | `llmprovider/catalog/models_catalog.go:665` | `case hasRun(sm, "4.7"):` |
| C4 | Any other Grok id is placed among the anchors | `llmprovider/catalog/models_catalog.go:674` | `return generationScore(sm, grokAnchors, 0)` |
| C5 | Grok's anchors | `llmprovider/catalog/models_catalog.go:679` | `var grokAnchors = []generationAnchor{{300, 30}, {400, 40}, {403, 45}, {405, 50}, {406, 60}, {407, 70}}` |
| C6 | A minor is one or two digits | `llmprovider/catalog/models_catalog.go:690` | `N.M or N-M, where N is one digit and M one or two` |
| C7 | The score above every anchor | `llmprovider/catalog/models_catalog.go:740` | `return newest.score + 10 + min(40, steps)` |
| C8 | Grok's list is curated with the ranker | `llmprovider/catalog/discovery.go:1080` | `return curateFromCatalog(llmprovider.ProviderGrok, staticGrok, usable, isUsableGrokModel, rankGrokModel, limit)` |
| C9 | The backfill is sorted by rank | `llmprovider/catalog/models_catalog.go:479` | `sortByRankDesc(remaining, rankFn)` |
| C10 | Stably | `llmprovider/catalog/models_catalog.go:499` | `slices.SortStableFunc(ids, func(a, b string) int { return cmp.Compare(rankFn(b), rankFn(a)) })` |
| C11 | Gemini's two-digit minor is tested | `llmprovider/catalog/models_catalog_test.go:114` | `{"gemini", rankGeminiModel, "gemini-3.10-flash", "gemini-3.9-flash"},` |
| C12 | Grok's generation order is tested | `llmprovider/catalog/models_catalog_test.go:272` | `order := []string{"grok-4.7", "grok-4.6", "grok-4.5", "grok-4.3", "grok-4", "grok-3-mini-fast"}` |
| C13 | `grok-4.1` scores 40 | `llmprovider/catalog/models_catalog_test.go:139` | `{rankGrokModel, "grok-4.1", 40},` |
| C14 | The full Grok list ends with a 4.20 id | `llmprovider/catalog/evidence_0041_test.go:152` | `grok[n-1] != "grok-4.20-multi-agent-0309"` |
| C15 | 0021's finding | `docs/decisions/0021-MADR-harden-and-tune-after-the-v1-1-review.md:117` | `**Hard-coded version tables invert newer generations.**` |
| C16 | 0021-PLAN saw 4.20 rank first, and kept it | `docs/decisions/0021-PLAN-harden-and-tune-after-the-v1-1-review.md:2492` | `Read numerically, 4.20 scores 84, above 4.7's 71, though its date suggests` |
| C17 | xAI's date for the 4.20 ids (2026-03-09) | `llmprovider/catalog/testdata/firstparty-2026-10-14/grok.json:1` | `"created":1773014400,"id":"grok-4.20-0309-non-reasoning"` |
| C18 | For `grok-4.3` (2026-04-17) | `llmprovider/catalog/testdata/firstparty-2026-10-14/grok.json:1` | `"created":1776384000,"id":"grok-4.3"` |
| C19 | For `grok-4.7` (2026-09-02) | `llmprovider/catalog/testdata/firstparty-2026-10-14/grok.json:1` | `"created":1788307200,"id":"grok-4.7"` |
| C20 | A test helper reads that listing | `llmprovider/catalog/listingdates_0041_test.go:92` | `func firstPartyPage(` |
| C21 | The label table | `llmprovider/catalog/models_catalog.go:855` | `var modelLabels = map[llmprovider.ProviderID]map[string]string{` |
| C22 | `grok-4.6`'s note | `llmprovider/catalog/models_catalog.go:877` | `[★ Recommended: the Grok CLI's default]` |
| C23 | `grok-4.5`'s note | `llmprovider/catalog/models_catalog.go:878` | `"Grok 4.5               [previous flagship]"` |
| C24 | `grok-4.7`'s note | `llmprovider/catalog/models_catalog.go:879` | `"Grok 4.7               [current flagship]"` |
| C25 | `Label` | `llmprovider/catalog/models_catalog.go:893` | `func Label(provider llmprovider.ProviderID, model string) string {` |
| C26 | Its fallback is the bare id | `llmprovider/catalog/models_catalog.go:853` | `Labels are advisory. Label falls back to the bare id` |
| C27 | The search index parses the label | `llmprovider/catalog/search.go:141` | `name := displayName(strings.ToLower(label))` |
| C28 | By cutting at the bracket | `llmprovider/catalog/model_matcher.go:139` | `func displayName(label string) string {` |
| C29 | A search result carries `Label` | `llmprovider/catalog/model_matcher.go:39` | `Label string // Label(provider, ID)` |
| C30 | The wizard's model choices | `wizard/configure.go:1001` | `label := catalog.Label(provider, m)` |
| C31 | `Prompter` is a consumer's UI | `wizard/prompter.go:61` | `// Prompter is the rendering seam.` |
| C32 | `Choice` has a `Detail` | `wizard/prompter.go:58` | `Detail string` |
| C33 | A test pins each label's prefix | `llmprovider/catalog/refresh_0033_test.go:78` | `if got := Label(c.provider, c.id); !strings.HasPrefix(got, c.want) {` |
| C34 | `staticGrok`'s comment | `llmprovider/catalog/models_catalog.go:151` | `staticGrok leads with the Grok CLI's defaults, grok-4.6 then grok-4.5` |
| C35 | 0012's fact | `docs/decisions/0012-MADR-conform-providers-to-reference-clients.md:194` | `* **Catalog filter.** The CLI's default models are` |
| C36 | 0012's decision | `docs/decisions/0012-MADR-conform-providers-to-reference-clients.md:556` | `leads with the CLI defaults` |
| C37 | 0033 made the labels follow the lists | `docs/decisions/0033-MADR-public-model-metadata-and-current-catalogs.md:314` | `* **The menu labels**` |
| C38 | R55 | `docs/guides/api-standards.md:357` | `- **R55. A new export is justified.**` |
| C39 | R48 | `docs/guides/api-standards.md:310` | `every non-internal package outside` |
| C40 | The package doc lists `Label` | `llmprovider/catalog/doc.go:4` | `It holds the curated static catalogs (Static, Label, Rank)` |
| C41 | The architecture note on labels | `docs/architecture.md:853` | `is per provider, so an id two providers offer can be` |
| C42 | grok-build's default | `grok-build:crates/codegen/xai-grok-models/default_models.json:2` | `"default": "grok-4.6",` |
| C43 | It is the last of five sources | `grok-build:crates/codegen/xai-grok-models/src/lib.rs:4` | `At runtime each model is resolved from the first of these that is set: CLI flag, ENV var, config.toml, remote settings, these defaults.` |
| C44 | It is compiled in | `grok-build:crates/codegen/xai-grok-models/src/lib.rs:12` | `include_str!("../default_models.json")` |
| C45 | `grok-4.5` is its second model | `grok-build:crates/codegen/xai-grok-models/default_models.json:47` | `"id": "grok-4.5",` |
| C46 | A keyed 1.0.70 session's model | `magic-cli-remote:internal/provider/grok/testdata/wire/1.0.70/frames.jsonl:1` | `"currentModelId":"grok-4.20-0309-non-reasoning"` |
| C47 | 4.7 has been a hint and the newest anchor since 0033-MADR D4 | `llmprovider/catalog/models_catalog_test.go:135` | `grok-4.7 is a hint since 0033-MADR D4, and the newest anchor, so` |
| C48 | models.dev's release date is read | `llmprovider/catalog/model_metadata.go:111` | `json:"release_date"` |
| C49 | Into `Released`, when it is not in the future | `llmprovider/catalog/model_metadata.go:698` | `if t, ok := parseRankDate(m.ReleaseDate); ok && !t.After(now) {` |
| C50 | `ModelInfo.Released` | `llmprovider/catalog/model_info.go:40` | `// Released is the model's release date, in UTC.` |
| C51 | A listing's own date sets it too | `llmprovider/catalog/listing_metadata.go:116` | `b.set(FieldReleased)` |
| C52 | A date every row shares is dropped | `llmprovider/catalog/discovery.go:443` | `func undatedConstant(models map[string]ModelInfo) {` |
| C53 | An unranked listing is ordered by date | `llmprovider/catalog/menu.go:252` | `func defaultRank(ids []string, facts map[string]ModelInfo, created map[string]time.Time,` |
| C54 | The cached facts take a context and options | `llmprovider/catalog/lookup.go:159` | `func CachedFacts(ctx context.Context, provider llmprovider.ProviderID, model string, opts ...llmprovider.Option) (ModelInfo,` |
| C55 | A generation above the anchors extrapolates | `llmprovider/catalog/models_catalog_test.go:138` | `{rankGrokModel, "grok-5", 100},` |
| C56 | Gemini's anchors | `llmprovider/catalog/models_catalog.go:553` | `var geminiAnchors = []generationAnchor{{205, 30}, {300, 40}, {305, 80}, {306, 90}, {307, 100}}` |
| C57 | Gemini's hints match whole tokens, so "3.1" is not 3.10 | `llmprovider/catalog/models_catalog.go:534` | `case hasRun(sm, "3.1") \|\| hasRun(sm, "3.0"):` |
| C58 | models.dev describes `grok-4.7` | `llmprovider/catalog/testdata/ranking-2026-10-12/metadata.json:1` | `"id":"grok-4.7","name":"Grok 4.7","description":"xAI's frontier model` |
| C59 | The Claude and Gemini rankers share `generationScore` | `llmprovider/catalog/models_catalog.go:631` | `score += generationScore(sm, claudeAnchors, 0)` |
| C60 | `gemini-3.10-flash` scores 180 + 113 | `llmprovider/catalog/models_catalog_test.go:133` | `{rankGeminiModel, "gemini-3.10-flash", 180 + 113},` |
| C61 | Gemini's ranker calls `generationScore` | `llmprovider/catalog/models_catalog.go:541` | `score += generationScore(sm, geminiAnchors, 0)` |

## Decision Drivers

* **No hand-written version.** A table of names to fix by hand goes stale.
  The reading must come from evidence the module reads.
* **Evidence over a guess.** A provider's dates and a family's own siblings
  say how its versions are read. One string does not (Context §2).
* **Robust to noisy dates.** A gateway's date, or a later snapshot of an
  old generation, must not reorder a family.
* **Right offline.** Magic's floor, and any caller with no metadata, must
  get the right order from the static list alone.
* **Gemini's 3.10 is not changed** (C11, C60).
* **A label's parts are data.** The module already parses its own labels
  to get names (C27). A caller should not have to.
* **A note states what stays true.** A note that ranks its model against
  others in time becomes false at the next release.
* **Compatibility (R48, C39).** `Rank` and `Label` keep their signatures,
  and `Label` keeps its form. New names are additions, each justified
  (R55, C38).

## Considered Options

### For Rank

* **R1. A hand-written Grok hint for 4.20, and a date test.** This record's
  first draft; set aside by the owner on 2026-11-06.
* **R2. The gap test alone.**
* **R3. Order generations by release date.**
* **R4. Layered inference: dates choose each family's reading, the gap test
  decides when there are no dates, the integer reading on a tie, and a
  guard over the recorded data.**
* **R5. Read every two-digit minor as a decimal.**
* **R6. Document that Rank orders the curated ids only.**

### For labels

* **L1. Export a label's name and note apart, and keep notes free of
  relative time.**
* **L1b. Export one function that returns both parts as a struct.**
* **L2. Correct the two wrong notes only.**

## Decision Outcome

Chosen options: **"R4"** and **"L1"**, chosen by the owner on 2026-11-06,
to be accepted on review of this record. R4 was chosen after the research
and the experiment in Context §2. The owner set aside the first draft's R1,
because it named a version by hand.

* **R4** reads each family's versions from evidence the module already
  holds. It needs no table of exceptions, it holds against noisy dates, and
  it is right offline for every case the experiment found. The guard turns
  the next naming the rule cannot read into a failing test at the next
  refresh of the recorded data, not a stale ranking.
* **L1** gives callers, and the module's own search, the name and the note
  as values. It keeps `Label`'s output for every entry whose note does not
  change, and it corrects the two wrong notes and the five that will go
  stale.

### D1. Each family's versions are read from its evidence

* **The readings.** For a version N.M, as `parseGeneration` finds it (C6),
  the order key is 100·N plus:
  * M, under the **integer** reading;
  * M, for a one-digit M, or M/10, for a two-digit M, under the
    **decimal** reading. So 4.20 has key 402, between 4.1 and 4.3, and 4.25
    has key 402.5.

  A one-digit minor has the same key under both readings, so only a family
  that holds a two-digit minor has a choice to make.
* **A family** is one provider's ids with the same tokens before the
  version and the same major: `grok` 4, `gemini` 3, `claude-opus` 4.
* **The choice, per family,** over the family's ids in view:
  1. **Dates.** For each pair of dated ids that the two readings order
     differently, count the pairs each reading puts against the dates. The
     reading with fewer wins. A pair both readings order alike is not
     counted, so a later snapshot of an older generation is not evidence
     (Context §2: 11 such pairs in the recorded Google data, 0 counted).
  2. **Otherwise, gaps.** Under each reading, the share of slots from the
     lowest version to the highest that a version fills, counting slots in
     units of one for the integer reading and of one tenth for the decimal
     reading. The fuller reading wins.
  3. **Otherwise, the integer reading,** C6's rule.
* **The ids in view, and the dates:**

  | Call | Ids in view | Dates |
  | :--- | :--- | :--- |
  | `Rank(provider, model)` | `Static(provider)` and `model` | none: `Rank` stays a pure function, reading no cache. `CachedFacts` would need a context and options (C54), and a cache's state would change `Rank`'s answer |
  | `SortByRank(provider, ids, released)`, new | `ids` and `Static(provider)` | `released`, which may be nil |
  | the module's curation (C8 to C10) | the listing and the static list | the listing's `Released`, where it gives one (C51, C52) |

* **`catalog.SortByRank`** is the one export D1 adds:

  | Name | Returns | Call sites that need it (R55) |
  | :--- | :--- | :--- |
  | `catalog.SortByRank(provider llmprovider.ProviderID, ids []string, released map[string]time.Time) []string` | a sorted copy of `ids`, by `Rank`'s score with each family's reading taken from the whole list, highest first, stably | Magic's picker, which would sort Grok's live or saved list. Its proposed 0235-MADR refuses to today, because of Context §1. The module's own curation, where the listing's dates reach the rule |

  `Rank`'s doc points to it for a whole list.
* **Scores.** `generationScore` (C7) takes the chosen reading's key. Its
  anchors, its hints and its rule above the newest anchor are unchanged.
  * A new generation still scores above the anchors with no edit (C55).
  * Gemini's 3.10 stays 293 (C60): its static siblings 3.5, 3.6 and 3.8
    with 3.10 fill 4 of 6 slots as integers, and 4 of 8 as decimals.
  * Grok's 4.20 takes key 402. With its static siblings it fills 5 of 6
    slots as decimals, and 5 of 18 as integers. The anchors give it 40: the
    anchor at 4.0, below the cap of 44 under 4.3. That is the score
    `grok-4` and `grok-4.1` take (C13). A tie is not an inversion, and a
    stable sort keeps the listing's order among ties (C10). Q6 asks about
    ties.
  * Hints match whole tokens, so Gemini's "3.1" hint does not match 3.10
    under either reading (C57).
  * The rule applies to the three rankers that call `generationScore`:
    Gemini, Claude and Grok (C61, C59, C4).
* **0021-MADR gains an amendment to C6:** generations stay numeric. A
  family's two-digit minor is read as D1 says, by this record.
* **Not in scope:**
  * `grok-build-0.1` scores 0 (Q3).
  * The other rankers do not read versions with `generationScore`.
  * An undated family that leaps from N.5 to N.12 is read as decimal. Only
    a date, or the author, can say otherwise (Context §2).

### D2. A label's name and note are exported apart

* **The table holds the parts.** `modelLabels` becomes
  `map[llmprovider.ProviderID]map[string]modelLabel`, where `modelLabel` is
  an unexported struct of `name` and `note`. The padding and the star leave
  the table.
* **Two exported functions**, with `Label`'s signature, provider
  comparison and fallback:

  | Name | Returns | Call sites that need it (R55) |
  | :--- | :--- | :--- |
  | `catalog.LabelName(provider, model) string` | the curated display name, such as `Grok 4.7`, or the bare id when the table has no entry | the search index, in place of `displayName(Label(...))` (C27); Magic's picker, for the SDK floor's option labels (its proposed 0235-MADR) |
  | `catalog.LabelNote(provider, model) string` | the note, such as `frontier model`, without the star, or `""` when there is none | Magic's picker, or any caller whose UI shows a detail line, such as `wizard.Choice.Detail` (C32); otherwise it parses `Label` for it |

  The names sort beside `Label` in the package's documentation.
* **`Label` is composed from the parts:** `fmt.Sprintf("%-23s[%s]", name,
  note)`. The note gains "★ Recommended: " in front when `model` is the
  first id of `Static(provider)`. The 23 columns become a named constant.
  For each of the 13 entries whose note does not change, the result is the
  string `Label` returns at `b5ce51d`. The scratch check proved the form of
  all 20, and the star rule holds on all four starred entries (Context §3).
  The star now follows the list's order by construction.
* **Seven notes change:**

  | id | Note at `b5ce51d` | Note | Why |
  | :--- | :--- | :--- | :--- |
  | `grok-4.6` | ★ Recommended: the Grok CLI's default | the Grok CLI's built-in default | issue 11; C42 to C44 |
  | `grok-4.5` | previous flagship | second in the Grok CLI's built-in list | false by xAI's dates; C45 |
  | `grok-4.7` | current flagship | frontier model | relative time; models.dev's description, "xAI's frontier model" (C58) |
  | `gemini-3.8-flash` | newest Flash, frontier coding | frontier coding | relative time |
  | `gpt-5.6-luna` | previous efficient tier | efficient tier | relative time |
  | `claude-sonnet-5-5` | latest Sonnet, high precision | high precision | relative time |
  | `claude-opus-5-5` | latest Opus, maximum capability | maximum capability | relative time |

  `grok-4.6` stays starred, as the first id of `staticGrok`, so its `Label`
  reads `Grok 4.6               [★ Recommended: the Grok CLI's built-in
  default]`.
* **The search index** reads `strings.ToLower(LabelName(provider, id))`,
  and the unused `displayName` is removed. For an entry, `displayName(Label)`
  is the trimmed name, which is `LabelName`. Without one, both are the id,
  for an id with no `[` and no surrounding space. None of the 375 recorded
  fixture files holds an `id`, `name` or `model` value with a `[`; a planted
  one was found. An id that held one would be indexed whole, where it was
  cut. That fixes the index rather than changing a match that worked.
  Search's ranking otherwise does not change.
* **The wizard is not changed.** It still passes `Label` to `Prompter`, so
  a consumer's prompter sees the same strings, apart from the seven notes.
  Q4 asks whether it should move the note to `Choice.Detail`.

### D3. Tests

A test that calls only names `b5ce51d` has is seen red there, or seen green
where it pins behaviour that must not change. A test that calls a new name
fails on a planted break instead. All of it runs on a scratch clone:

* **`TestRankGrok_FollowsListingDates_0070`.**
  * It reads the recorded xAI listing (`firstPartyPage`, C20) and keeps the
    `grok-N.M` ids `isUsableGrokModel` admits.
  * For every pair, the id with the earlier `created` must not rank higher.
  * Red at `b5ce51d`: the 4.20 ids, created first, score 93 against 4.3's
    45.
* **`TestRankGrokModel` (C12)** gains a check that `grok-4.20-0309-reasoning`
  ranks below `grok-4.3`, and not below `grok-4`. Red at `b5ce51d` (93).
* **`TestRankGenerations` (C11, C13, C55, C60)** passes unchanged at both
  revisions: Gemini's 3.10 is still 293, and `grok-5` is still 100.
* **`TestVersionReading_0070`,** a table run through the rule:
  * the cases of Context §2;
  * the dated leap, read as integer;
  * a family with no two-digit minor, which has no choice;
  * the 11 recorded Google pairs that both readings order alike, which do
    not count.

  Planted breaks:
  * the date test counting every pair;
  * the gap test inverted;
  * the tie going to decimal.

  The undated leap is not a case: it would pin the answer the rule cannot
  give.
* **`TestVersionReading_Guard_0070`** is the guard. It runs over the recorded
  dated sources: the first-party listings, the recorded models.dev
  document, and a new fixture of the 22 families from the live document of
  2026-11-06. For every family with a two-digit minor and enough dated pairs
  to decide:
  * the reading the gap test gives offline (the family's ids, with the
    static list where the provider has one) must equal the reading its
    dates choose;
  * `Rank`, for the providers that rank, must order each pair the readings
    order differently as the dates do.

  A future refresh of the recorded data that holds a naming the rule cannot
  read fails here. Planted break: the decimal reading turned off.
* **`TestSortByRank_0070`:**
  * Magic's 14 Grok ids (C46's list) sort as 4.7, 4.6, 4.5, 4.3, the 4.20
    ids, with `grok-build-0.1` and the media models after them.
  * Given dates, the leap family sorts by them.
  * Planted break: `SortByRank` ignores `released`.
* **`TestLabelNotes_NoRelativeTime_0070`.**
  * For every entry of `modelLabels`, the bracketed part of `Label` holds
    none of the words `current`, `previous`, `newest`, `newer`, `latest`,
    `older`, `oldest`, `last`, matched as words.
  * It is written over `Label`, so that it compiles at both revisions. Red
    at `b5ce51d` on the six entries of Context §3.
  * It also requires that `Label(grok, "grok-4.6")` does not hold "the Grok
    CLI's default]", issue 11's text (C22). Red at `b5ce51d`.
* **`TestLabel_Parts_0070`:** for every entry, `Label` equals the
  composition, and a table of all 20 `Label` strings, whole, is the golden.
  * Planted breaks: a width of 22; the star on every entry.
  * The 13 unchanged strings are compared with `b5ce51d`'s on the scratch
    clone.
* **`TestLabelName_Fallback_0070`:** an unlabelled id, and a gateway's
  id, give the id and `""` (0033-MADR D4, C37). Planted break: `LabelName`
  returns `""`.
* **`TestLabel_StarIsFirstStatic_0070`:** the star is on exactly the first
  `Static` id of each provider with an entry. Planted break: the star on
  `grok-4.5`.
* **`TestLabel_ByProvider_0033` (C33)** passes unchanged: it checks
  prefixes, which do not change. The search tests pass unchanged, and a
  planted `LabelName` that returns the id must fail one of them.

### D4. Docs, records and the release

* **Code comments:**
  * `generationScore` and `parseGeneration` describe the two readings and
    the choice, citing this record.
  * `staticGrok`'s comment (C34) says grok-4.6 and grok-4.5 are
    grok-build's compiled-in defaults, which a CLI with a key or a
    signed-in account does not use.
  * The `modelLabels` comment describes the parts and the composed `Label`.
  * `doc.go` (C40) names `SortByRank`, `LabelName` and `LabelNote`.
* **`docs/architecture.md`:**
  * the `llmprovider/catalog` row names the three functions (R43);
  * the label note (C41) says that the parts are exported;
  * a sentence on how versions are read.
* **0012-MADR gains an amendment** to §6's fact (C35) and decision (C36).
  The two ids are grok-build's last-resort defaults, compiled in (C42 to
  C44). A CLI with an xAI key reported `grok-4.20-0309-non-reasoning` (C46).
  This record leaves the list's order unchanged (Q2).
* **0021-MADR gains the amendment to C6** in D1.
* **The fixture** of the 22 live families holds each id's provider, id and
  `release_date`, and nothing else. It goes under
  `llmprovider/catalog/testdata/`, with its date in its directory's name.
  `make fixtures-check` passes on it.
* **`CHANGELOG.md`:**
  * Added: `SortByRank`, `LabelName` and `LabelNote`.
  * Fixed: `Rank`'s reading of a two-digit minor, which placed
    `grok-4.20-*` above `grok-4.7`.
  * Changed: the seven notes.
* **The release is minor (R48):** `v1.27.0` (Q1). `tools/apidiff.py` must
  list exactly the three additions, and no incompatible change.
* **Issue 11** is answered by the release. The owner closes it; the agent
  posts nothing to GitHub unasked.

### Consequences

* Good, because no version is named by hand. A family's versions are read
  from its own dates and siblings, and a new release needs no edit.
* Good, because a caller can sort any Grok listing, live or static, and get
  xAI's order.
* Good, because noisy dates do not reorder a family. They vote only on the
  reading, and only where the readings differ.
* Good, because the guard checks the rule against every dated family the
  module records, so a naming it cannot read fails a test.
* Good, because a caller with any UI gets the name and the note without
  parsing, and the module stops parsing its own labels.
* Good, because the star can no longer disagree with the list, and the
  test of notes fails when relative time returns to a note.
* Good, because issue 11's misleading claim leaves the label, the comment
  and 0012-MADR.
* Neutral, because no list the module builds from the recorded listing
  changes order (C14).
* Neutral, because `Label`'s form is kept: a terminal menu sees the same
  layout.
* Bad, because the rule is more code than a hint. It needs a test of its
  own, and the guard.
* Bad, because `Rank`, which sees one id, takes its siblings from the
  static list only. A family that the static list lacks is read from the id
  alone, which is a tie, and so the integer reading. `SortByRank` sees the
  whole list.
* Bad, because an undated family with a leap is read wrong, and only a
  date can correct it. No such family exists in the data measured.
* Bad, because between two anchors the anchors' scores collapse. `grok-4`,
  `grok-4.1` and `grok-4.20` all score 40 (Q6).
* Bad, because three exports are added, which the module keeps for v1.
  `LabelNote`'s only named call site is outside this module.

### Confirmation

* Every test in D3 passes on the tree. Each one marked red is seen red at
  `b5ce51d` (`tools/redgreen.py --expect fail --rev b5ce51d`). Each planted
  break is caught (`tools/plant.py`).
* `tools/claims.py` passes this record's rows, and its plants of false
  claims are caught.
* `tools/apidiff.py v1.26.1` lists exactly `SortByRank`, `LabelName` and
  `LabelNote` as additions.
* `make fixtures-check` passes on the new fixture.
* `tools/gate.py` passes before staging.
* After the release, `tools/consumer.py` and `tools/offline.py` pass on the
  new tag.

## Pros and Cons of the Options

### R1. A hand-written Grok hint for 4.20, and a date test

* Good, because it is one case in the ranker's switch (C3).
* Bad, because it names a version by hand. The next naming of the kind
  needs another, which is the staleness the owner set out to remove.

### R2. The gap test alone

* Good, because it needs no data, and it is right for all 20 decided live
  families and for every constructed case but one.
* Bad, because it reads an undated leap wrong, and it cannot use the dates
  the module already holds to correct it.

### R3. Order generations by release date

* Good, because it follows the providers' own record directly.
* Bad, because dates are noisy. A gateway dates 4.20 three weeks late, the
  live document has 34 discordant pairs under the better reading, and a
  later snapshot of an older generation would rise above newer ones.
* Bad, because with no metadata, as on Magic's floor, it falls back to the
  name, and the defect returns where it was found.
* Bad, because `Rank`'s answer would depend on what is cached.

### R4. Layered inference, with a guard

* Good, because it uses each kind of evidence where it is sound: dates to
  choose a reading, gaps when there are no dates, and C6's rule when
  neither decides.
* Good, because it is right on the recorded data and the live data, and on
  11 of 12 constructed cases. The twelfth, an undated leap, is read
  correctly once it is dated; no string-only rule can read it undated.
* Good, because the guard keeps it honest as the recorded data is
  refreshed.
* Bad, because it is the most code of the options, and adds `SortByRank`.

### R5. Read every two-digit minor as a decimal

* Good, because it is one rule, and right for every two-digit minor the
  live document holds today.
* Bad, because `gemini-3.10-flash` would fall below `gemini-3.9-flash`,
  which C6 exists to prevent (C11).

### R6. Document that Rank orders the curated ids only

* Good, because it changes no code.
* Bad, because `Rank` stays wrong for the ids xAI lists, and a caller who
  misses the documentation repeats Magic's finding.

### L1. Export the name and the note, notes free of relative time

* Good, because each part is a plain string, which any UI can use.
* Good, because `Label` keeps its output for the 13 unchanged entries, and
  the module's search reads the name directly.
* Bad, because two functions are added for one table.

### L1b. One function returning a struct

* Good, because a caller makes one call.
* Bad, because a new exported type needs R51's concurrency sentence and a
  name of its own. The package already has `ModelInfo.Name`, from
  models.dev, so a type with a `Name` field invites confusion with it.
* Bad, because a caller that wants only the name still unpacks a struct.

### L2. Correct the two wrong notes only

* Good, because it is a patch.
* Bad, because a caller still parses padded strings, and the module still
  parses its own.
* Bad, because the five relative notes go stale at the next releases.

## Owner questions

*Answered by the owner on 2026-11-06, with the record's acceptance: each
question as recommended. Q5's answer adds an eighth note; see the amendment
"Q5's answer adds an eighth note".*

* **Q1. The release.** Recommended: `v1.27.0`, since three exports are
  added (R48).
* **Q2. Issue 11's second question: should `staticGrok` lead with
  `grok-4.7`?** Recommended: not in this record. `Static(grok)`'s first id
  carries the star (D2). A caller that takes it as a default, as issue 11
  describes, would change model. That is a decision of its own, against
  0012-MADR §6 (C36), with its own measurement.
* **Q3. `grok-build-0.1` ranks 0,** below every generation, though xAI
  created it on 2026-04-16. Recommended: leave it. It is a coding model,
  not a generation, and it is fifth in `staticGrok`, which sets its place
  in the module's list.
* **Q4. Should the wizard show the note in `Choice.Detail`?**
  Recommended: no. A consumer's `Prompter` (C31) would see a different
  `Label` for every labelled model. `LabelNote` lets a consumer do it.
* **Q5. `claude-opus-4-8` and `claude-opus-5-5` both read "maximum
  capability".** The word is a superlative, not a time, so the test in D3
  does not catch it. Recommended: `claude-opus-4-8`'s note becomes
  "maximum capability of Opus 4", which names its generation.
* **Q6. Ties between anchors.** `grok-4`, `grok-4.1` and `grok-4.20` all
  score 40, though their dates differ. Recommended: leave the anchors' rule
  as 0021-MADR's amendment "C6's anchors are capped" set it. A tie is not an
  inversion. Ordering within the gap would need scores between anchors,
  which is a change to that amendment, for its own record.

## More Information

**Probes on 2026-11-06.** All were run outside the tree, which stayed
clean:

* **The Rank probe:** a scratch module, with a `replace` directive to this
  tree at `b5ce51d`. It printed `Rank` and `Label` for nine Grok ids and
  their order by `Rank`, and `Static` and `Label` for the 20 labelled ids.
* **The reading experiment:** a Python script over the live models.dev
  document, which was fetched once, with no credential, and read with
  `python3 -I`. A second script ran the recorded document and the
  first-party listings. A third ran the twelve constructed cases. It exits
  1 on the leap case, as Context §2 states.
* **The label check:** a Python script parsed `modelLabels` and required
  the 23-column form of every entry. It found 20 entries, all in the form,
  and listed the relative notes. A copy of the source with one label padded
  to 22 columns made it fail (exit 1, naming `grok-4.3`).
* **The listing dates:** read from
  `llmprovider/catalog/testdata/firstparty-2026-10-14/`, in `grok.json`,
  `gemini.json`, `openai.json` and `claude.json`.
* **grok-build:** the checkout's head, `2bdd1d6a` (2026-09-29), equals the
  remote's (`git ls-remote`). `default_models.json` last changed in
  `9684fa3c` (2026-08-27).
* **The installed CLI:** its binary, 1.0.70, was searched for the
  compiled-in `"default": "grok-` text, which occurs twice, both
  `grok-4.6`. Its model cache, fetched by CLI 1.0.67 from xAI's CLI proxy,
  lists `grok-4.7`, `grok-4.7-build-fast`, `grok-4.6`, `grok-4.5`, with no
  default field. The script printed ids, names and descriptions only; the
  cache's per-model key fields were not read.
* **Magic's capture:** read with `git show 0f595dc9:…`. The keyed 1.0.70
  session lists 14 ids, the same 14 as xAI's recorded listing.

**Corrections to the analysis given to the owner before this record.**

* `grok-4.7-build-fast` is on the CLI proxy's list, not in xAI's API
  listing. So the module's own Grok list puts no newer id behind the 4.20
  ones.
* The first guess at the stale note, that the CLI's compiled-in default
  had moved, was wrong. The compiled-in default is still `grok-4.6`. The
  note misleads because a CLI with an xAI key, or, as reported, a signed-in
  one, resolves another model first (issue 11, C43, C46).

**Sources.**

* [Perl version.pm](https://metacpan.org/pod/version) and
  [version::Internals](https://perldoc.perl.org/version::Internals): the
  decimal and dotted readings, and the leading `v`;
* [`sort -V` ordering 1.9 before 1.10](https://gitbucket.inist.fr/tdm/web-dumps/pull/3);
* [OpenRouter's models sort, `newest` by creation date](https://openrouter.ai/docs/client-sdks/go/models/operations/getmodelssort.md);
* [models.dev](https://github.com/sst/models.dev): `release_date`, the
  first public release.

**Related records.**

* `docs/decisions/0012-MADR-conform-providers-to-reference-clients.md`
  (§6, amended by D4);
* `docs/decisions/0021-MADR-harden-and-tune-after-the-v1-1-review.md`
  (C6 and its amendment "C6's anchors are capped"; C6 amended by D1);
* `docs/decisions/0033-MADR-public-model-metadata-and-current-catalogs.md`
  (D4, the labels follow the lists);
* `docs/decisions/0041-MADR-decide-per-model-behaviour-from-metadata.md`
  (the listing's dates, `Released`);
* `docs/decisions/0048-MADR-adopt-go-sdk-best-practices.md` (D7, R55).

The PLAN is
[0070-PLAN-place-grok-4-20-by-date-and-split-model-labels.md](0070-PLAN-place-grok-4-20-by-date-and-split-model-labels.md).

## Amendment 2026-11-06: Q5's answer adds an eighth note

Decided by the owner on 2026-11-06, with the answers to the owner questions.

* D2's table of changed notes gains a row: `claude-opus-4-8`, "maximum
  capability" becomes "maximum capability of Opus 4", which names its
  generation (Q5).
* So eight notes change, and 12 entries keep `Label`'s string at `b5ce51d`.
  Where D2 and D3 say seven notes and 13 entries, they mean eight and 12.

## Amendment 2026-11-06: the module's curation reads siblings, not dates

Found while writing the PLAN, before any code changed.

* **New fact.** The curators that rank a live listing see only ids: the
  `curator` type is a function of the usable ids and a limit (C62). Those
  ids are all that the Grok listing's fetch decodes (C67). The dates reach
  `menuOf` in its facts, after the curator has run. `menuOf` takes the
  curator's band of six (C63) and orders every other id newest first, by
  those dates (C64). For Gemini, Claude and Grok, the band is
  `metadataCurate`'s: it ranks by models.dev's facts, ages included (C65).
  The name ranker only fills behind them (C66).
* **D1's table is refined.** "The module's curation" reads the listing's
  ids and the static list, with no dates: the gap test, then the integer
  reading. Passing the menu's dates to the curators would change the
  `curator` type and 15 test calls in five files, for the one case the gap
  test misreads, an undated leap. Where the menu has dates, it already
  orders by them outside the band.
* **Dates still choose the reading** wherever a caller gives them, through
  `SortByRank`'s `released`. A caller holding a `Catalog` has them in each
  `MenuEntry`'s `Info.Released`.

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C62 | A curator sees ids and a limit | `llmprovider/catalog/discovery.go:491` | `type curator func(usable []string, limit int) []string` |
| C63 | The band is the curator's | `llmprovider/catalog/menu.go:138` | `band := curate(slices.Clone(uncaveated), MaxListed)` |
| C64 | The rest is newest first | `llmprovider/catalog/menu.go:140` | `newestFirst(rest, facts)` |
| C65 | `metadataCurate` reads models.dev's age | `llmprovider/catalog/model_metadata.go:699` | `c.ageDays, c.ageKnown = floorDays(t, now), true` |
| C66 | The name ranker fills behind it | `llmprovider/catalog/model_metadata.go:754` | `return rankRecommended(profile, provider, cands, append(fallback(usable, limit), usable...), limit)` |
| C67 | Grok's fetch decodes ids | `llmprovider/catalog/discovery.go:1085` | `func fetchDataIDs(ctx context.Context, endpoint, header, value string, cfg config, provider llmprovider.ProviderID) ([]string, error) {` |

## Amendment 2026-11-06: the date test's pair rule is an identity, not a behaviour

Found while writing the PLAN, before any code changed.

* **New fact.** A pair that both readings order alike is either against
  the dates under both readings or under neither. It adds one to both
  counts, or nothing. So it never changes which count is lower. Counting
  every pair, or only the pairs the readings order differently, makes the
  same choice. A scratch script compared the two over the live document of
  2026-11-06: 2,478 dated families (provider, family and major), and the
  choice differed in none.
* **D1's rule stands as written.** It counts only the pairs the readings
  order differently, because that says why a later snapshot of an older
  generation is not evidence. It is not a separate behaviour.
* **D3 is corrected:**
  * the planted break "the date test counting every pair" is removed, since
    it changes nothing a test could see;
  * the row "the 11 recorded Google pairs that both readings order alike"
    becomes a constructed row: a family with `4.20` and `4.3`, and a
    `4.1` dated after both, which reads as decimal.
