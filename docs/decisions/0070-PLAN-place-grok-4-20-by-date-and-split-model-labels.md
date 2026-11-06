---
status: in-progress
date: 2026-11-06
associated-madr: "0070-MADR-place-grok-4-20-by-date-and-split-model-labels.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Reading Each Model Family's Versions from Its Dates and Siblings, and Splitting a Label's Name and Note

Associated MADR: [0070-MADR-place-grok-4-20-by-date-and-split-model-labels.md](0070-MADR-place-grok-4-20-by-date-and-split-model-labels.md)

## Goal

The MADR's D1 to D4, with its three amendments of 2026-11-06 and the
owner's answers to Q1 to Q6 (each as recommended):

* **D1.** Each model family's two-digit minor is read as an integer or as a
  decimal, chosen from the family's dates, else its gaps, else the integer
  rule. No version is named by hand. `Rank` keeps its signature and reads
  `Static(provider)` with the id. The new `SortByRank` reads the whole list
  and the caller's dates. The module's curation reads the listing's ids and
  the static list (amendment "the module's curation reads siblings, not
  dates").
* **D2.** `LabelName` and `LabelNote` return a label's parts. `Label` is
  composed from them, with the star on `Static(provider)`'s first id, and
  keeps its string for the 12 entries whose note does not change. Eight
  notes change (D2, and the amendment "Q5's answer adds an eighth note").
* **D3.** Every new test is seen red at `b5ce51d`, or fails on a planted
  break. The guard runs over every dated source the module records.
* **D4.** The docs, the two amended records, `CHANGELOG.md`, and the
  release `v1.27.0` (Q1).

Done means every Confirmation item of the MADR holds, and this record's
execution record shows each one.

## Scope

### In scope

| Phase | Files |
| :--- | :--- |
| 0, records | this PLAN; the MADR; `docs/README.md`; the amendments to `0012-MADR-conform-providers-to-reference-clients.md` §6 and `0021-MADR-harden-and-tune-after-the-v1-1-review.md` C6 |
| 1, the version reading (D1) | `llmprovider/catalog/version_reading.go` (new); `llmprovider/catalog/models_catalog.go`; `llmprovider/catalog/discovery.go`; `llmprovider/catalog/testdata/versions-2026-11-06/families.json` (new); `llmprovider/catalog/rank_dates_0070_test.go` (new); `llmprovider/catalog/version_reading_0070_test.go` (new); `llmprovider/catalog/models_catalog_test.go`; `llmprovider/catalog/example_test.go` |
| 2, the labels (D2) | `llmprovider/catalog/models_catalog.go`; `llmprovider/catalog/search.go`; `llmprovider/catalog/model_matcher.go`; `llmprovider/catalog/label_notes_0070_test.go` (new); `llmprovider/catalog/labels_0070_test.go` (new); `llmprovider/catalog/model_matcher_test.go` (the `withLabel` helper only); `llmprovider/catalog/example_test.go` |
| 3, docs (D4) | `llmprovider/catalog/doc.go`; `docs/architecture.md`; `CHANGELOG.md` |
| 4, the live check | none; one listing request to xAI |
| 5, the release (Q1) | `CHANGELOG.md`; `README.md`; the tag, which is the owner's |

### Exported names (R55)

| Name | Phase | Call sites that need it |
| :--- | :--- | :--- |
| `catalog.SortByRank(provider llmprovider.ProviderID, ids []string, released map[string]time.Time) []string` | 1 | Magic's picker, sorting Grok's live or saved list, which its proposed 0235-MADR refuses to do today (MADR Context §1) |
| `catalog.LabelName(provider llmprovider.ProviderID, model string) string` | 2 | the search index (`search.go:141`, MADR C27); Magic's picker, for the SDK floor's labels |
| `catalog.LabelNote(provider llmprovider.ProviderID, model string) string` | 2 | Magic's picker, or any caller with a detail line, such as `wizard.Choice.Detail` (MADR C32) |

No other exported name is added, changed or removed. `tools/apidiff.py
v1.26.1` lists exactly these three.

### Out of scope

* The other rankers (`rankOpenAIModel`, `rankOpencodeModel`,
  `rankHuggingFaceModel`, `rankKiloModel`): they do not call
  `generationScore` (MADR C4, C59, C61).
* The anchors, the hints and the cap rule of `generationScore`: unchanged,
  so `grok-4`, `grok-4.1` and `grok-4.20` tie at 40 (Q6).
* `staticGrok`'s order (Q2), `grok-build-0.1`'s score (Q3), and the wizard,
  which keeps passing `Label` (Q4).
* Passing the menu's dates to the curators (the MADR's amendment).
* `docs/guides/migrating-from-mcplib.md`: no new name replaces an `mcplib`
  identifier. `llmprovider.ModelLabel` maps to `catalog.Label`, which
  stays (R43).
* Magic's own records, which are that repository's.
* Any request's content (R45): no golden changes.

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts after the owner has committed the last.
* **Red first.** A new or changed test that calls only names `b5ce51d` has
  is seen red there, by `python3 -B tools/redgreen.py SPEC.json --expect
  fail --rev b5ce51d`. A test that calls a new name instead fails on each
  of its planted breaks, by `python3 -B tools/plant.py SPEC.json`. Both run
  on scratch clones. The specs live in the session's scratch directory, and
  this record holds each failure line.
* **The drift check.** Every test outside this PLAN's list passes
  unchanged. The one exception is the `withLabel` helper's body
  (`model_matcher_test.go:165`), whose map type changes. Its signature, its
  callers and what they assert do not change (R46).
* **Signatures kept.** `rankGeminiModel`, `rankClaudeModel` and
  `rankGrokModel` stay `func(string) int`, the single-id path `Rank` uses.
  `curateFromCatalog` and the `curator` type are unchanged. `withLabel`
  keeps its signature. Fifteen test calls depend on these (the MADR's
  amendment).
* **Pre-add.** `make pre-add-check FILES="…"` on the phase's Go files: gofmt,
  golangci-lint on the host and `GOOS=windows`, vet, the packages' tests
  and govulncheck.
* **The gate, before staging:**
  * `python3 -B tools/gate.py`, or `--markdown` for a records-only phase;
  * `python3 -B tools/claims.py` on the MADR;
  * `make records-check`, `python3 -B tools/recordlint.py --changed`,
    `python3 -B tools/denyscan.py --changed`, `python3 -B tools/linkcheck.py`.

  Each exit status is captured before any filter, and read before anything
  is written from it. Files are staged only when the gate passes (`&&`).
* **R50 to R52.** Each new export has a doc comment and an `Example` with
  an `// Output:` comment. None returns an error. `make docs-check` stays at
  0.
* **Other platforms.** The change is string and slice logic, with no build
  tag, path, process or network code. CI's three platforms run it before the
  tag, as for 0069.
* **Stage, then the owner commits** with `git commit --no-edit`. Push and
  tags happen on the owner's word, after the disclosure guard has run over
  the outgoing commits.
* **Stop conditions:**
  * a fact of the MADR, or one of its claims, no longer holds;
  * a test outside this PLAN needs a changed assertion;
  * `TestRankGenerations` (C11, C13, C55, C60) or the full-list check (C14)
    fails;
  * a golden changes, or `apidiff` lists anything beyond the three names;
  * `make fixtures-check` flags the new fixture;
  * the guard finds a recorded family whose offline reading differs from its
    dates' reading;
  * `llmprovider/catalog` falls below its 80 % floor (R47).

  On any of these, stop, record a dated deviation here (with an amendment in
  the MADR when a decision or a fact changes), and wait.

## Implementation Steps

### Phase 0: records

1. This PLAN, the MADR (accepted, with its answers and amendments) and
   their rows in `docs/README.md`, committed as records only.
2. Once the owner approves this PLAN, `tools/records.py status` sets it
   `in-progress`.
3. Dated amendments, without rewriting the decisions:
   * **`0021-MADR-harden-and-tune-after-the-v1-1-review.md`, C6:**
     generations stay numeric. A family's two-digit minor is read as an
     integer or a decimal, chosen from its dates, else its gaps, else the
     integer rule (0070-MADR D1). `gemini-3.10` still follows `gemini-3.9`.
     `grok-4.20` reads as 4.2, which 0021-PLAN's live check had accepted
     reading as 4.20 (0070-MADR C16).
   * **`0012-MADR-conform-providers-to-reference-clients.md`, §6:** the fact
     "The CLI's default models are `grok-4.6` then `grok-4.5`" (C35) is
     refined. They are grok-build's compiled-in defaults, the last of five
     sources (C42 to C44). A CLI with an xAI key reported
     `grok-4.20-0309-non-reasoning` (C46). The decision that `StaticGrok`
     leads with them (C36) is unchanged (0070-MADR Q2).
4. `python3 -B tools/gate.py --markdown`, then stage.

### Phase 1: the version reading (D1)

1. **The fixture,** `llmprovider/catalog/testdata/versions-2026-11-06/families.json`.
   * Its source is the models.dev document fetched on 2026-11-06 into the
     session's scratch directory: 5,960,317 bytes, SHA-256
     `f1881e9a51a2fb6d995d41c0767a21eeb7d92f56ed5a701a0d9331b200520bf4`. If
     that file is gone, stop. A new fetch is a new measurement, and the
     MADR's counts would need re-measuring.
   * A scratch script, run with `python3 -I`, reads it. It groups every id
     whose name after the last `/` has a version, by `parseGeneration`'s
     rule (C6), by provider, family (the tokens before the version) and
     major. It keeps each group that holds a two-digit minor and two or
     more dated versions.
   * It writes `{"source", "fetched", "sha256", "families": [{"provider",
     "family", "major", "models": [{"id", "release_date"}]}]}`, sorted, with
     no other field.
   * It asserts what the MADR states: 22 families. Twenty-one are Grok
     families (`grok`, or `xai-grok` on `oci`), and one is `nano-gpt`'s
     `llama`. The date test decides 20 of them, all decimal, and the gap
     test agrees on all 20. The two undecided are `aimlapi`'s and `poe`'s
     Grok. The families hold 152 dated ids. A scratch count with this
     grouping gave each of these numbers on 2026-11-06.
   * `make fixtures-check` must pass on it.
2. **`version_reading.go`** (new), unexported except `SortByRank`:
   * `parseVersion(id) (version, bool)`, where `version` holds the family
     (the tokens before it, joined by `-`), the major, the minor and the
     minor's digit count. `parseGeneration`'s rule is unchanged: the first
     one-digit token after the family name, then an optional one- or
     two-digit minor (C6). `generationScore` is its one caller, so
     `parseGeneration` is replaced.
   * `key(v, decimal)`, the order key in tenths:
     * 1000·N + 10·M under the integer reading;
     * 1000·N + M for a two-digit M under the decimal reading, and 1000·N +
       10·M for a one-digit M.

     So 4.20 decimal is 4020, between 4.1 (4010) and 4.3 (4030).
   * `slot(v, decimal)` for the gap test: M; or ⌊M/10⌋ for a two-digit M
     read as decimal.
   * `chooseReadings(ids []string, released map[string]time.Time)
     map[familyKey]bool` (true is decimal). For each family holding a
     two-digit minor:
     1. **The date test:** over pairs of dated ids whose keys order
        differently under the two readings, count each reading's pairs
        against the dates. Fewer wins.
     2. **The gap test,** when the counts are equal: distinct slots ÷ (max −
        min + 1). Higher wins.
     3. **The integer reading,** when both are equal.

     A family with no two-digit minor is not in the map, and reads as
     integer.
   * `rankerFor(provider, ids, released) func(string) int`: the provider's
     ranker, with the readings chosen over `ids` ∪ `Static(provider)`.
   * **`SortByRank(provider, ids, released)`:** a new slice with `ids` in
     `rankerFor(provider, ids, released)`'s order, highest first, stable.
     A nil or empty `ids` gives an empty slice. Its doc says:
     * that it reads each family's versions from the list and the dates;
     * that `released` may be nil;
     * that a `Catalog`'s `MenuEntry.Info.Released` holds a listing's
       dates;
     * that it is safe for concurrent use, holding no state;
     * and it cites 0070-MADR D1.
3. **`models_catalog.go`:**
   * `generationScore(id, anchors, below, decimal)` takes the key. Each
     anchor's ord is scaled by ten where it is compared, and each anchor's
     score is unchanged.
     * Above the newest anchor, its steps count whole minors as today: (key
       − the newest anchor's key) ÷ 10 within a major, and 20 a major.
     * Under the integer reading, every score equals today's. A scratch
       model of this rule gave today's values on 2026-11-06:
       `gemini-3.10` 113, `grok-4.20` 93, `grok-4.1` 40 and `grok-5` 100,
       the values `TestRankGenerations` pins (C13, C55, C60). Under the
       decimal reading, `grok-4.20` gives 40. The phase's tests show both
       (step 5).
   * Each of `rankGeminiModel`, `rankClaudeModel` and `rankGrokModel` keeps
     its signature, and reads its family's reading from `Static(provider)`
     with the id, with no dates.
     * Each delegates to an internal form that takes the reading.
     * `rankerFor` uses that form for a list.
   * `Rank` is unchanged in signature and in its switch. Its doc gains:
     "For a whole list, SortByRank reads each family's versions from the
     list, and from dates when the caller has them (0070-MADR D1)."
   * The comments of `rankGrokModel`, `generationScore` and the anchors
     describe the two readings and cite 0070-MADR D1.
   * `staticGrok`'s comment (C34) becomes: "staticGrok leads with
     grok-build's compiled-in defaults, grok-4.6 then grok-4.5, which a
     Grok CLI with an xAI key or a signed-in account does not use
     (0012-MADR §6; 0070-MADR D4)". The rest of the comment is kept.
4. **`discovery.go`:** `curateGemini`, `curateClaude` and `curateGrok`
   (`:798`, `:907`, `:1079`) pass `rankerFor(provider, usable, nil)` as
   `curateFromCatalog`'s `rankFn`, in place of the bare ranker. The rankers
   then read the listing's ids with the static list (the amendment). The
   signatures are unchanged.
5. **Tests:**
   * **`rank_dates_0070_test.go`:** `TestRankGrok_FollowsListingDates_0070`.
     * It reads `firstPartyPage(t, "grok.json")` (C20) and decodes each
       row's `id` and `created`.
     * It keeps the ids that `isUsableGrokModel` admits and the regular
       expression `^grok-\d\.\d+`, written in the test, matches. That is the
       seven `grok-4.x` ids (C17 to C19), and not `grok-build-0.1`.
     * For every pair, the one created earlier must not rank higher by
       `Rank(llmprovider.ProviderGrok, id)`.

     It calls only names `b5ce51d` has, so that it builds there and is
     seen red.
   * **`models_catalog_test.go`, `TestRankGrokModel` (C12):** add that
     `rankGrokModel("grok-4.20-0309-reasoning")` is below
     `rankGrokModel("grok-4.3")` and not below `rankGrokModel("grok-4")`.
     No other line of the file changes.
   * **`version_reading_0070_test.go`:**
     * `TestVersionReading_0070`: a table through `chooseReadings`, each
       row stating the reading wanted:
       * the MADR's constructed cases (Context §2), less the undated leap:
         Gemini 3.5 to 3.10, Gemini's static ids with 3.10, 3.5 to 3.8 with
         3.10, 3.9 with 3.10 and Python-style 3.8 to 3.14, each integer; and
         Grok's static ids with 4.20, 4.1 with 4.20, Grok with xAI's dates
         and Llama 3.05, each decimal;
       * the dated leap (`3.5` in 2026-01, `3.12` in 2026-06): integer;
       * a lone `4.10`, and a lone `4.20`, with no siblings: integer, by the
         tie rule;
       * a family with no two-digit minor: absent from the map;
       * a family with `4.20` (2026-03-09) and `4.3` (2026-04-17), and a
         `4.1` dated 2026-12-01, later than both: decimal. The late `4.1`
         is against the dates under both readings, so it does not decide
         (the MADR's amendment "the date test's pair rule is an identity").

       The undated leap is not a row (the MADR's D3).
     * `TestVersionReading_Guard_0070`, over three sources:
       * the new fixture's 22 families;
       * the recorded models.dev document's `google`, `anthropic` and
         `xai` providers;
       * the first-party listings `grok.json` and `claude.json`.

       It checks each family that holds a two-digit minor and whose date
       test decides:
       * the reading the gap test gives alone, over the family's ids with
         `Static(provider)`'s where the provider is the module's (`xai` is
         Grok, `google` Gemini, `anthropic` Claude), equals the reading its
         dates choose;
       * for the module's providers, `Rank` orders each pair that the
         readings order differently as the dates do.

       It also checks that the fixture holds 22 families, and that the date
       test decides 20.
     * `TestSortByRank_0070`:
       * Magic's 14 Grok ids (MADR C46's list, in the capture's order) sort
         as `grok-4.7`, `grok-4.6`, `grok-4.5`, `grok-4.3`, then the three
         4.20 ids in input order, then `grok-build-0.1` and the six
         `grok-imagine-*` ids in input order. All seven tail ids score 0.
       * Given dates, `gemini-4.5-flash` (2026-01-10) and `gemini-4.12-flash`
         (2026-06-10) sort `4.12` first. That is a family the static list
         lacks, so the dates decide it. A scratch model of the scores gave
         generations of 142 against 135 under the integer reading, and 131
         against 135 under the decimal. The undated order, the known limit,
         is not asserted. Gemini's 3.x family is not used here: with the
         static ids, its gap test ties, and reads as integer with or without
         dates.
       * `SortByRank` returns a new slice, leaving its argument's order
         unchanged. A nil `ids` gives an empty slice.
   * **`example_test.go`:** `ExampleSortByRank`. It sorts `grok-4.20-0309-reasoning`,
     `grok-4.7` and `grok-4.3`, with nil dates, and its `// Output:` lists
     4.7, 4.3, 4.20.
6. **Red:** a spec naming `TestRankGrok_FollowsListingDates_0070` and
   `TestRankGrokModel`, with the files `rank_dates_0070_test.go` and
   `models_catalog_test.go`, `--expect fail --rev b5ce51d`. Each must fail on
   its assertion: 93 against 45.
7. **Plants** (`tools/plant.py`, package `./llmprovider/catalog/`), each
   caught:

   | Plant | Fails |
   | :--- | :--- |
   | `chooseReadings` returns an empty map (the decimal reading off) | `TestRankGrok_FollowsListingDates_0070`, `TestRankGrokModel`, `TestVersionReading_Guard_0070`, `TestSortByRank_0070`, `ExampleSortByRank` |
   | the date test skipped, so the gap test always decides | `TestVersionReading_0070` (the dated leap), `TestSortByRank_0070` (the dated Gemini 4.x pair) |
   | the date test's comparison inverted | `TestVersionReading_0070` (Grok with xAI's dates, the dated leap), `TestVersionReading_Guard_0070` *(2026-11-06, in execution: the guard does not fail. It reads the dates' choice with `againstDates` itself, not through the reading rule, so it never runs the comparison. The plant fails `TestVersionReading_0070` and `TestSortByRank_0070`.)* |
   | the gap test's comparison inverted | `TestVersionReading_0070`, `TestVersionReading_Guard_0070` |
   | a tie read as decimal | `TestVersionReading_0070` (the lone `4.10` and `4.20`) |
   | `SortByRank` passes nil in place of `released` | `TestSortByRank_0070` (the dated Gemini 4.x pair) |
   | the single-id rankers read the id alone, without `Static(provider)` | `TestRankGrok_FollowsListingDates_0070`, `TestRankGrokModel` |
   | `curateGrok` passes the bare `rankGrokModel` again | none expected, since the recorded listing's backfill holds only the 4.20 ids. This plant is run to record that, not as a check. If it is caught, the result is recorded |

8. **The drift check:**
   * `go test ./...` passes.
   * `TestRankGenerations` passes unchanged: Gemini's 3.10 is 293, `grok-5`
     100, `grok-4.1` 40.
   * The full-list check (C14) passes, with `grok-4.20-multi-agent-0309`
     last of eight.
9. `make pre-add-check FILES="…"` on the phase's Go files, `make
   fixtures-check`, `make docs-check`, `python3 -B tools/apidiff.py v1.26.1`
   (listing `SortByRank` alone), and the gate.

### Phase 2: the labels (D2)

1. **`models_catalog.go`:**
   * `type modelLabel struct{ name, note string }`, and `modelLabels`
     becomes `map[llmprovider.ProviderID]map[string]modelLabel`. Each entry
     holds its name and its note, with no padding and no star.
   * `const labelColumn = 23`.
   * `LabelName(provider, model)`: the entry's name, or `model` when there is
     none. Provider compared as `Label` compares it (C25).
   * `LabelNote(provider, model)`: the entry's note, or `""`.
   * `Label(provider, model)`: `model` with no entry; the name alone when
     the note is empty; otherwise `fmt.Sprintf("%-*s[%s]", labelColumn,
     name, note)`. The note gains `★ Recommended:` and a space in front when `model`
     equals `Static(provider)[0]`.
     * No shipped entry has an empty note: the label check found a note on
       all 20.
     * `TestSearch_GlobIDBeforeLabel` injects a label with no bracket,
       through `withLabel`, and its `Label` stays that string.
   * **The eight notes,** as the MADR's D2 table and its amendment give
     them:

     | id | Note |
     | :--- | :--- |
     | `grok-4.6` | the Grok CLI's built-in default |
     | `grok-4.5` | second in the Grok CLI's built-in list |
     | `grok-4.7` | frontier model |
     | `gemini-3.8-flash` | frontier coding |
     | `gpt-5.6-luna` | efficient tier |
     | `claude-sonnet-5-5` | high precision |
     | `claude-opus-5-5` | maximum capability |
     | `claude-opus-4-8` | maximum capability of Opus 4 |

     The four starred entries' notes lose their `★ Recommended:` prefix,
     which `Label` adds: `ultra-low latency`, `fast & cost-effective`,
     `high speed, low latency`, and `grok-4.6`'s new note.
   * The table's comment describes the parts, the composition and the star,
     citing 0070-MADR D2. The docs of `Label`, `LabelName` and `LabelNote`
     name each other.
2. **`search.go:141`:** `name := strings.ToLower(LabelName(provider, id))`.
   `displayName` (`model_matcher.go:138-142`) is removed, since it has no
   other caller.
3. **`model_matcher_test.go`, `withLabel`:**
   * the map literal becomes `map[string]modelLabel{}`;
   * the label is split at its first `[`: the name trimmed, and the note
     less its closing `]`.

   Its signature and its three callers are unchanged.
4. **Tests:**
   * **`label_notes_0070_test.go`:** `TestLabelNotes_NoRelativeTime_0070`.
     * For every key of `modelLabels`, the bracketed part of `Label` holds
       none of `current`, `previous`, `newest`, `newer`, `latest`,
       `older`, `oldest` or `last`, matched as whole words, ignoring case.
     * `Label(grok, "grok-4.6")` does not hold "the Grok CLI's default]".
     * It ranges the map's keys only, so it builds at both revisions.
   * **`labels_0070_test.go`:**
     * `TestLabel_Parts_0070`: a table of all 20 `Label` strings, whole, as
       the golden. For each entry, `Label` equals the composition of
       `LabelName` and `LabelNote` with the star rule.
     * `TestLabelName_Fallback_0070`: `LabelName` and `LabelNote` of an
       unlabelled Grok id give the id and `""`. So do those of
       `gemini-3.8-flash` on `opencode-zen` and `gpt-6-luna` on
       `opencode-go` (0033-MADR D4, C37).
     * `TestLabel_StarIsFirstStatic_0070`: for each provider with an entry,
       exactly one `Label` holds `★`, and it is `Static(provider)[0]`'s.
       (2026-11-06, in execution: Together has an entry, but its first static
       id has none, as the MADR's Context §3 records, so none of its labels
       holds `★`. The test takes the MADR D3's wording: the starred labels
       are exactly the first static id's, when it has an entry; and the first
       ids of Gemini, OpenAI, Claude and Grok each have one.)
   * **`example_test.go`:** `ExampleLabelName`, printing `LabelName` and
     `LabelNote` of `grok-4.7` on Grok, `Grok 4.7` and `frontier model`.
5. **The 12 unchanged strings:** the golden's 12 rows for entries whose note
   does not change must equal `b5ce51d`'s strings. Those were printed on
   2026-11-06 by the scratch Rank probe (`rankprobe/static.txt` in the
   session's scratch directory) and are compared by a scratch script. The
   eight changed rows must differ from `b5ce51d`'s only inside the brackets.
6. **Red:** `TestLabelNotes_NoRelativeTime_0070`, with the file
   `label_notes_0070_test.go`, `--expect fail --rev b5ce51d`. It must fail on
   six relative words and on `grok-4.6`'s text.
7. **Plants,** each caught:

   | Plant | Fails |
   | :--- | :--- |
   | `labelColumn` 22 | `TestLabel_Parts_0070` |
   | the star on every entry | `TestLabel_Parts_0070`, `TestLabel_StarIsFirstStatic_0070` |
   | the star on `Static(provider)[1]` | the same two |
   | `LabelName` returns `""` with no entry | `TestLabelName_Fallback_0070` |
   | `LabelName` returns the id always | `TestSearchModels_LabelSubstringTier`, `TestSearch_ExistingTiers_0061`, `TestLabel_Parts_0070`. On 2026-11-06 the first two were seen to catch the same break at `b5ce51d`, made there by indexing the id: "roadrunner = [], want acme/m1 at scoreLabelSubstring" |
   | `grok-4.5`'s old note restored | `TestLabelNotes_NoRelativeTime_0070`, `TestLabel_Parts_0070` |

8. **The drift check:**
   * `go test ./...` passes.
   * Every search test passes unchanged, among them
     `TestSearch_AnnotationNotSearched` and `TestSearch_GlobIDBeforeLabel`.
   * So do `TestLabel_ByProvider_0033` (C33), `TestStaticIDs_Labelled`
     and `TestLabels_NoOrphans`.
   * Every wizard test passes unchanged.
9. `make pre-add-check FILES="…"`, `make docs-check`, `python3 -B
   tools/apidiff.py v1.26.1` (listing the three names), and the gate.

### Phase 3: docs (D4)

1. **`llmprovider/catalog/doc.go`** (C40): "It holds the curated static
   catalogs (Static, Label, LabelName, LabelNote, Rank, SortByRank)", with a
   sentence that a family's two-digit minor is read from its dates and
   siblings (0070-MADR D1).
2. **`docs/architecture.md`:**
   * the `llmprovider/catalog` row (`:130`) names `SortByRank`, `LabelName`
     and `LabelNote` beside `Label`;
   * the label note (`:853`, C41) adds that `LabelName` and `LabelNote` give
     the parts;
   * one sentence there on reading versions, citing 0070-MADR D1.
3. **`CHANGELOG.md`, `[Unreleased]`:**
   * **Added:** "`catalog.SortByRank` sorts a list of model ids by `Rank`,
     reading each family's version numbers from the list, and from release
     dates the caller gives. `catalog.LabelName` and `catalog.LabelNote`
     return a curated label's name and note, unpadded (0070-MADR)."
   * **Fixed:** "`catalog.Rank` read `grok-4.20-*` as generation 4.20,
     above `grok-4.7`, though xAI released it before `grok-4.3`. A family's
     two-digit minor is now read as an integer or a decimal, from the
     family's dates and siblings, so `grok-4.20` is 4.2 and `gemini-3.10`
     still follows `gemini-3.9` (0070-MADR)."
   * **Changed:** "Eight label notes:
     * `grok-4.6`'s no longer calls it the Grok CLI's default, which is
       grok-build's compiled-in fallback;
     * `grok-4.5`'s 'previous flagship' was false;
     * the notes that ranked a model in time no longer do;
     * `claude-opus-4-8`'s names its generation (0070-MADR)."
4. `make docs-check`, `make changelog-check`, `make links-check`, and the
   gate.

### Phase 4: the live check

No file in the tree changes. One listing request goes to xAI, under the
key in `XAI_API_KEY`. Its presence is checked; its value is never printed.

1. `go test -tags live_gateways ./llmprovider/catalog/ -run '^TestLive_GrokListingTextOnly$' -count=1 -v`,
   at the commit of Phases 1 to 3. It logs the live usable ids.
2. A scratch program in the session's scratch directory, built against the
   tree with a `replace` directive, lists Grok through `catalog.List`. It
   prints:
   * `Recommended`;
   * each `Menu` entry's id and `Released`;
   * `SortByRank` of `Usable`, with nil dates and then with the menu's
     dates.
3. **The rule:**
   * no `grok-4.20-*` id ranks above `grok-4.3` in either `SortByRank`
     order, or in `Recommended` past the static five;
   * the two orders agree.

   A finding otherwise stops the work. A skip, such as an unfunded or
   unreachable account, is recorded with its reason.

### Phase 5: the release (Q1: `v1.27.0`)

1. `CHANGELOG.md`'s `[Unreleased]` becomes `## [v1.27.0] - DATE`, dated by
   the release commit, under an empty `[Unreleased]`. The compare links are
   updated: `[Unreleased]` is `v1.27.0...HEAD`, and `[v1.27.0]` is
   `v1.26.1...v1.27.0`.
2. `README.md:356`: "The current release is `v1.27.0`."
3. Before the tag:
   * the gate;
   * `python3 -B scripts/check_changelog.py --section v1.27.0`;
   * the disclosure guard over the outgoing commits;
   * CI on `main` passes after the owner's push.
4. The tag and its push are the owner's. After it:
   * CI on the tag, including its `release` job;
   * the GitHub Release's notes equal the section, after the job rewrites
     its relative links;
   * the five newest tags carry Releases (R53);
   * `python3 -B tools/consumer.py v1.27.0` and `python3 -B tools/offline.py
     v1.27.0`, once sum.golang.org lists the tag. That took about 29 minutes
     for `v1.26.1`. Its listing is waited for, never bypassed.
5. Issue 11 is answered by the release. The owner closes it.

## Verification

* Phase 1's and Phase 2's red runs and plants, each failure line, in this
  record.
* `go test ./...` and `python3 -B tools/gate.py` passing at each code phase.
* `python3 -B tools/apidiff.py v1.26.1` listing exactly `SortByRank`,
  `LabelName` and `LabelNote`; `make docs-check` at 0.
* `make fixtures-check` passing on the new fixture, and the guard passing
  over all three sources.
* The 12 unchanged `Label` strings equal to `b5ce51d`'s.
* Phase 4's live output meeting its rule, or a recorded skip.
* The MADR's claims passing at their revisions, with their plants caught.
* After the tag: the release checks of Phase 5.

## Rollout and Rollback

* **Rollout:** a minor release, `v1.27.0`. Callers take it by upgrading.
  Nothing is configured. A caller that sorted a Grok listing by `Rank` sees
  the 4.20 ids move below `grok-4.3`. A caller that shows `Label` sees eight
  notes change.
* **Rollback:** revert Phase 2's commit, then Phase 1's. Each is
  self-contained. `Rank` reads every minor as an integer again, and `Label`
  returns its `b5ce51d` strings. `SortByRank`, `LabelName` and `LabelNote`
  go only in a major version (R42, R48): once released, a rollback keeps
  them, delegating to the old behaviour, and deprecates them.
* **If the guard fails at a later refresh of the recorded data:** a family
  is named whose evidence the rule misreads. The remedy is a record that
  weighs it, not a hand-written exception (0070-MADR, R1 rejected).
* **If a caller reports an undated leap read as decimal:** `SortByRank`
  with that family's dates reads it correctly (0070-MADR Context §2).

## Execution record

### Phase 0 (2026-11-06)

* **Step 1:** the MADR (accepted), this PLAN and their rows, committed as
  `4d0c757`. Before that commit:
  * the MADR's claims passed, 67 of 67;
  * eight planted false claims were each caught by `tools/claims.py
    --plants`: a wrong line (C5), a wrong date (C17), grok-build's default
    (C42), Magic's model (C46), `grok-4.5`'s note (C23), Gemini's 3.10 score
    (C60), the future-date check (C49) and models.dev's description (C58);
  * the record lint, the identifier scan and the link check were clean, and
    the Markdown gate passed, 11 of 11.
* **Step 2:** the owner approved this PLAN on 2026-11-06 ("committed,
  proceed"). `tools/records.py status` set it `in-progress`, with its row.
* **Step 3:** the amendments were written, each without rewriting the
  decision it refines:
  * `0021-MADR-harden-and-tune-after-the-v1-1-review.md`, "Amendment
    2026-11-06: a two-digit minor is read from its family's evidence";
  * `0012-MADR-conform-providers-to-reference-clients.md`, "Amendment
    2026-11-06: grok-4.6 and grok-4.5 are grok-build's compiled-in defaults
    (0070 D4)".
* **Step 4:** the Markdown gate passed, 11 of 11. The records were
  committed as `71a7e33`.

### Phase 1 (2026-11-06)

* **Step 1, the fixture:**
  `llmprovider/catalog/testdata/versions-2026-11-06/families.json`, 14,633
  bytes.
  * The source was checked first: 5,960,317 bytes, with the SHA-256 the
    step names.
  * The scratch script grouped by provider, family and major, on the name
    after the last `/`, and asserted each number the step states: 22
    families (20 `grok`, one `xai-grok`, one `llama`); 20 decided by their
    dates, all decimal; the gap test agreeing on all 20; `aimlapi`'s and
    `poe`'s Grok undecided; 152 dated ids.
  * The fixture holds only provider, family, major, id and release date.
  * **Found and recorded:** the same grouping on the whole id, as the
    module's tokenizer splits it, keeps 22 families too, but 15 of the 152
    ids fall out. An org prefix joins the family name, as in
    `meta-llama/llama-3.1-8b-instruct`, so `nano-gpt`'s Llama ids split
    into small families. The step's grouping is the one the MADR counted.
    So the guard reads each fixture id by its name after the last `/`.
  * Inside the module, no id with a `/` reaches the version reader. Gemini,
    Claude and Grok, whose rankers read versions, list ids without one, and
    the gateways' rankers do not read versions (MADR C4, C59, C61). No
    decision changes.
  * `make fixtures-check`: 381 files, 134 hits, 0 problems. A scratch clone
    of `71a7e33` gave 380 files and the same 134 hits, so the fixture adds
    none.
* **Steps 2 to 4:**
  * `version_reading.go` holds `parseVersion`, `versionKey`, `versionSlot`,
    `chooseReadings` (with `readsDecimal`, `againstDates` and `gapShare`),
    `rankerFor` and `SortByRank`.
  * In `models_catalog.go`, `generationScore` takes the reading, and
    `parseGeneration` is gone. `rankGeminiModel`, `rankClaudeModel` and
    `rankGrokModel` keep their signatures and delegate to
    `rankGeminiWith`, `rankClaudeWith` and `rankGrokWith`, with the reading
    over `Static(provider)` and the id.
  * `Rank`'s doc, `staticGrok`'s comment and the rankers' comments are as
    the steps give them.
  * `curateGemini`, `curateClaude` and `curateGrok` pass `rankerFor(…,
    usable, nil)`.
* **Step 5, the tests:** `rank_dates_0070_test.go`;
  `version_reading_0070_test.go` (`TestVersionReading_0070` with 14 rows,
  `TestVersionReading_Guard_0070`, `TestSortByRank_0070`);
  `TestRankGrokModel`'s added check; `ExampleSortByRank`. All pass. The
  guard's counts held: the dates decide 22 families, 20 of them the
  fixture's.
* **Step 6, red at `b5ce51d`:** `tools/redgreen.py --expect fail --rev
  b5ce51d`, "2 of 2 fail as expected":

  ```text
  models_catalog_test.go:284: grok-4.20-0309-reasoning = 93, want below grok-4.3 = 45 and not below grok-4 = 40
  rank_dates_0070_test.go:47: grok-4.20-0309-non-reasoning, created first, ranks 93, above grok-4.3 at 45
  ```

  The second test printed the same failure for each 4.20 id against each of
  4.3 to 4.7.
* **Step 7, plants:** `tools/plant.py`, "all 7 plant(s) caught". A second
  run, whose command printed every failing test's name, gave:

  | Plant | Failed |
  | :--- | :--- |
  | the decimal reading off | `ExampleSortByRank`, `TestRankGrokModel`, `TestRankGrok_FollowsListingDates_0070`, `TestSortByRank_0070`, `TestVersionReading_0070`, `TestVersionReading_Guard_0070` |
  | the date test skipped | `TestSortByRank_0070`, `TestVersionReading_0070` ("the dated leap: x reads as decimal, want integer") |
  | the date test's comparison inverted | `TestSortByRank_0070`, `TestVersionReading_0070` (Grok with xAI's dates, the dated leap, the late 4.1) |
  | the gap test's comparison inverted | `ExampleSortByRank`, `TestRankGenerations` ("gemini-3.10-flash = 220, not above gemini-3.9-flash = 292"), `TestRankGrokModel`, `TestRankGrok_FollowsListingDates_0070`, `TestSortByRank_0070`, `TestVersionReading_0070`, `TestVersionReading_Guard_0070` |
  | a tie read as decimal | `TestVersionReading_0070` (the lone 4.10 and 4.20) |
  | `SortByRank` passes nil dates | `TestSortByRank_0070` ("[gemini-4.5-flash gemini-4.12-flash], want gemini-4.12-flash first") |
  | the single-id rankers without `Static(provider)` | `TestRankGrokModel`, `TestRankGrok_FollowsListingDates_0070`, `TestVersionReading_Guard_0070` |

  Each plant fails at least the tests the step names, except the date
  comparison inverted, which the step's table now annotates: the guard does
  not run it.
  * The eighth plant, `curateGrok` with the bare `rankGrokModel`, was not
    caught, as the step expected. The recorded listing's backfill holds only
    the 4.20 ids, which tie under either ranker.
* **Step 8, the drift check:** every existing test in
  `llmprovider/catalog` passed before the new tests were added, with
  `TestRankGenerations` and the full-list check (C14) unchanged.
* **Step 9:**
  * `make pre-add-check` on the seven Go files: "7 file(s) clean (gofmt,
    golangci-lint, go vet, go test, govulncheck)".
  * `tools/apidiff.py v1.26.1`: "Compatible changes: ./llmprovider/catalog.SortByRank: added", and nothing else.
  * `make docs-check`: "sdkaudit: 0 violation(s) of R50,R51,R52".
  * The gate: `python3 -B tools/gate.py`, "gate: 29 of 29 passed". The
    files were then staged, and the owner committed them as `ee3db9e`.

### Phase 2 (2026-11-06)

* **Steps 1 to 3:**
  * `models_catalog.go`: `modelLabel{name, note}`, `labelColumn = 23`, the
    table of names and notes with no padding and no star, `labelFor`,
    `Label` composed from the parts, `LabelName` and `LabelNote`. The
    table's comment and the three docs are as step 1 gives them.
  * The eight notes are step 1's. The four starred notes lost their prefix,
    which `Label` adds.
  * `search.go` indexes `strings.ToLower(LabelName(provider, id))`;
    `displayName` is removed from `model_matcher.go`.
  * `withLabel`'s body splits the label at its first `[`. Its signature and
    callers are unchanged, and its diff is the body and its comment only.
* **Step 4, the tests:** `label_notes_0070_test.go`
  (`TestLabelNotes_NoRelativeTime_0070`); `labels_0070_test.go`
  (`TestLabel_Parts_0070`, with the 20-row golden, `TestLabelName_Fallback_0070`,
  `TestLabel_StarIsFirstStatic_0070`); `ExampleLabelName`. All pass.
  * **Found and annotated in step 4:** the PLAN's "exactly one `Label` holds
    `★`" does not hold for Together, whose first static id is unlabelled
    (the MADR's Context §3). The test follows the MADR D3's wording, "the
    star is on exactly the first `Static` id of each provider with an
    entry": Together's starred set is empty, and the other four providers'
    is their first id. No decision or asserted fact changes.
* **Step 5, the 12 unchanged strings:** a scratch script read the golden
  from `labels_0070_test.go` and the `b5ce51d` strings from the Rank probe's
  output: "12 unchanged equal b5ce51d; 8 changed only inside the brackets".
  The eight:

  | id | Note at `b5ce51d` | Note |
  | :--- | :--- | :--- |
  | `claude-opus-4-8` | maximum capability | maximum capability of Opus 4 |
  | `claude-opus-5-5` | latest Opus, maximum capability | maximum capability |
  | `claude-sonnet-5-5` | latest Sonnet, high precision | high precision |
  | `gemini-3.8-flash` | newest Flash, frontier coding | frontier coding |
  | `gpt-5.6-luna` | previous efficient tier | efficient tier |
  | `grok-4.5` | previous flagship | second in the Grok CLI's built-in list |
  | `grok-4.6` | ★ Recommended: the Grok CLI's default | ★ Recommended: the Grok CLI's built-in default |
  | `grok-4.7` | current flagship | frontier model |

* **Step 6, red at `b5ce51d`:** `tools/redgreen.py --expect fail --rev
  b5ce51d`, "1 of 1 fail as expected":

  ```text
  label_notes_0070_test.go:31: claude claude-opus-5-5: note "latest Opus, maximum capability" places the model in time with ["latest"]
  label_notes_0070_test.go:31: claude claude-sonnet-5-5: note "latest Sonnet, high precision" places the model in time with ["latest"]
  label_notes_0070_test.go:31: gemini gemini-3.8-flash: note "newest Flash, frontier coding" places the model in time with ["newest"]
  label_notes_0070_test.go:31: grok grok-4.5: note "previous flagship" places the model in time with ["previous"]
  label_notes_0070_test.go:31: grok grok-4.7: note "current flagship" places the model in time with ["current"]
  label_notes_0070_test.go:31: openai gpt-5.6-luna: note "previous efficient tier" places the model in time with ["previous"]
  label_notes_0070_test.go:36: Label(grok, grok-4.6) = "Grok 4.6               [★ Recommended: the Grok CLI's default]", want no claim that it is the Grok CLI's default
  ```

* **Step 7, plants:** `tools/plant.py`, with a command that prints every
  failing test's name, "all 6 plant(s) caught":

  | Plant | Failed |
  | :--- | :--- |
  | `labelColumn` 22 | `TestLabel_Parts_0070` |
  | the star on every entry | `TestLabel_Parts_0070`, `TestLabel_StarIsFirstStatic_0070` |
  | the star on `Static(provider)[1]` | `TestLabel_Parts_0070`, `TestLabel_StarIsFirstStatic_0070` |
  | `LabelName` returns `""` with no entry | `TestLabelName_Fallback_0070` |
  | `LabelName` returns the id always | `ExampleLabelName`, `TestLabelName_Fallback_0070`, `TestLabel_Parts_0070`, `TestSearchModels_LabelSubstringTier`, `TestSearch_ExistingTiers_0061` |
  | `grok-4.5`'s old note restored | `TestLabelNotes_NoRelativeTime_0070`, `TestLabel_Parts_0070` |

  Each plant fails at least the tests the step names.
* **Step 8, the drift check:** `go test ./...` passed. Run by name, these
  passed unchanged: `TestSearch_AnnotationNotSearched`,
  `TestSearch_GlobIDBeforeLabel`, `TestSearchModels_LabelFromModelLabel`,
  `TestSearchModels_NoSubsequenceOverLabels`, `TestLabel_ByProvider_0033`,
  `TestStaticIDs_Labelled` and `TestLabels_NoOrphans`. Every wizard test
  passed.
* **Step 9:**
  * `make pre-add-check` on the seven Go files: "7 file(s) clean (gofmt,
    golangci-lint, go vet, go test, govulncheck)".
  * `tools/apidiff.py v1.26.1`: "Compatible changes:
    ./llmprovider/catalog.LabelName: added, ./llmprovider/catalog.LabelNote:
    added, ./llmprovider/catalog.SortByRank: added", and nothing else.
  * `make docs-check`: "sdkaudit: 0 violation(s) of R50,R51,R52".
  * The gate: `python3 -B tools/gate.py`, "gate: 29 of 29 passed". The
    files were then staged, and the owner committed them as `ef3e86d`.

### Phase 3 (2026-11-06)

* **Step 1, `llmprovider/catalog/doc.go`:** the static catalogs are
  "(Static, Label, LabelName, LabelNote, Rank, SortByRank)", and a new
  paragraph says how `Rank` reads a family's two-digit minor and what
  `SortByRank` reads, citing 0070-MADR D1. Its two examples are tests:
  `grok-4.20` below `grok-4.3` (`TestRankGrokModel`), and `gemini-3.10`
  after `gemini-3.9` (`TestRankGenerations`).
* **Step 2, `docs/architecture.md`:** the `llmprovider/catalog` row names
  `SortByRank` (D1), and `LabelName` and `LabelNote` (D2), beside `Label`.
  The label note adds the parts, and one sentence on reading versions, with
  a link to the MADR.
* **Step 3, `CHANGELOG.md`'s `[Unreleased]`:** Added, Changed and Fixed, as
  the step gives them, each linking the MADR, in the file's section order.
* **Step 4:**
  * `make docs-check`: "sdkaudit: 0 violation(s) of R50,R51,R52".
  * `make changelog-check`: "39 release section(s), 0 problem(s)".
  * `make links-check`: "0 problem(s), 1127 relative link(s) in 160
    file(s)".
  * `make markdownlint`: "0 issues in 0 files".
  * `make pre-add-check FILES=llmprovider/catalog/doc.go`: "1 file(s)
    clean".
  * `tools/denyscan.py --changed`: "0 hit(s) in 3 file(s)".
  * The gate: `python3 -B tools/gate.py`, "gate: 29 of 29 passed". The
    files were then staged, and the owner committed them as `d73a9e9`.

### Phase 4 (2026-11-06)

At `d73a9e9`, the commit of Phases 1 to 3, with a clean tree.
`XAI_API_KEY`'s presence was checked; its value was never printed.

* **Step 1:** `go test -tags live_gateways ./llmprovider/catalog/ -run
  '^TestLive_GrokListingTextOnly$' -count=1 -v` passed:

  ```text
  live_listing_test.go:55: usable: [grok-4.20-0309-non-reasoning grok-4.20-0309-reasoning grok-4.20-multi-agent-0309 grok-4.3 grok-4.5 grok-4.6 grok-4.7 grok-build-0.1]
  --- PASS: TestLive_GrokListingTextOnly (0.34s)
  ```

* **Step 2:** the scratch program, built against the tree with a `replace`
  directive, listed Grok through `catalog.List`:

  ```text
  source: live
  Recommended: [grok-build-0.1 grok-4.3 grok-4.20-0309-reasoning grok-4.6 grok-4.5 grok-4.7 grok-4.20-0309-non-reasoning]
  Menu:
    grok-build-0.1                   2026-04-16
    grok-4.3                         2026-04-17
    grok-4.20-0309-reasoning         2026-03-09
    grok-4.6                         2026-08-06
    grok-4.5                         2026-06-29
    grok-4.7                         2026-09-02
    grok-4.20-0309-non-reasoning     2026-03-09
    grok-4.20-multi-agent-0309       2026-03-09
  SortByRank, nil dates:   [grok-4.7 grok-4.6 grok-4.5 grok-4.3 grok-4.20-0309-non-reasoning grok-4.20-0309-reasoning grok-4.20-multi-agent-0309 grok-build-0.1]
  SortByRank, menu dates:  [grok-4.7 grok-4.6 grok-4.5 grok-4.3 grok-4.20-0309-non-reasoning grok-4.20-0309-reasoning grok-4.20-multi-agent-0309 grok-build-0.1]
  ```

* **Step 3, the rule, held:** in both `SortByRank` orders the three
  `grok-4.20` ids follow `grok-4.3`, and the orders are equal. In
  `Recommended`, `grok-4.3` is second and both `grok-4.20` ids follow it.
  * The program checked the rule over the whole of `Recommended`, which is
    stricter than "past the static five".
  * In the same run, the rule was seen to fail on a planted order, the live
    order with its first `grok-4.20` id moved to the top: "FAIL: PLANTED
    order puts grok-4.20-0309-non-reasoning (0) above grok-4.3 (4)".
* **Found, with no decision changed:**
  * A live `Recommended` does not lead with the static five, as the rule's
    wording supposed. With models.dev's facts the band is
    `metadataCurate`'s, ranked by the profile, and the name ranker fills
    behind it. The MADR records this in its amendment "the module's
    curation reads siblings, not dates" (C63 to C66).
  * Two listing requests went to xAI, not one: steps 1 and 2 are separate
    processes, and the listing cache is in-process. Each request was a
    listing; neither was a generation.
