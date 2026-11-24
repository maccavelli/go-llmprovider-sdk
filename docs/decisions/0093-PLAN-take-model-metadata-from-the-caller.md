---
status: in-progress
date: 2026-11-24
associated-madr: "0093-MADR-take-model-metadata-from-the-caller.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement: The SDK Takes a models.dev Document From Its Caller and Fetches None

Associated MADR: [0093-MADR-take-model-metadata-from-the-caller.md](0093-MADR-take-model-metadata-from-the-caller.md)

*(2026-11-24, second draft: written for the MADR's second draft, after the owner's answers to Q1 to Q4 and the first
draft's review. BASE is no longer "the DL65 commit to come": DL65 is committed at b2917f0.)*

*(2026-11-24, approved. Owner ruling, quoted exactly, given for the MADR and this PLAN together: "Accept; approve; build next (Recommended)". The repository's PLAN statuses (`scripts/check_records.py`) have no `approved`; an approved PLAN awaiting its build carries `in-progress`, as 0089-PLAN does.)*

## Goal

Every SDK reader of models.dev's facts reads the document its caller hands in with `catalog.WithMetadata`, a request reading
only its provider's: no download, no cache entry, no refresh, no environment variable, no embedded extract. Then v1.33.0's
release steps resume.

## Scope

In: the MADR's decided rows, MH1 to MH46 and MH51 to MH57. Out: MH47 to MH50 (gobble's records; no store; no
presentation). A finding outside the rows is recorded as a follow-up, not built.

## Preconditions

* The MADR is `accepted`; this PLAN is approved; the owner says `proceed`, `execute the plan` or `do phase N`.
* BASE is the SDK's HEAD when the build starts (b2917f0 at drafting, or later if 0092 lands first); Phase 0 records it.
* No `git add`, commit, stash, checkout, restore, reset, clean or worktree in the tree: the coordinator stages each phase, the
  owner commits and pushes. Scripts are Python 3, standard library, in the build's scratch directory unless a row puts them
  in `tools/`.

## Exported names (R55), each with its call site

| Name | `apidiff` line | Call site |
| :--- | :--- | :--- |
| `catalog.WithMetadata` | `- ./llmprovider/catalog.WithMetadata: added` | gobble-cli's listings, lookups, limits calls and `provider.New` (its 0072 D17); the wizard's `Options.ProviderOptions`; `tools/offline.py`'s program; `internal/cmd/catalogs` |
| `catalog.Limits.RouteFallback` | none: `Limits` is new since v1.32.0 | `providers/claude/limit.go`'s `limitInput`; `providers/opencode`'s `limitInput` |

The whole expected `apidiff` output is the MADR's "The expected apidiff lines".

## Implementation Steps

Each phase ends with no blocker, every row it names tested, and every plant caught; the execution record numbers each pass,
and a third pass of one phase is a stop.

### Phase 0: re-ground and inventory (read only, and scratch clones)

1. `python3 -B tools/claims.py docs/decisions/0093-MADR-take-model-metadata-from-the-caller.md --rev self=BASE`; a row
   that moved gets a dated note in the MADR before any code.
2. Re-run the drafting step's `inventory.py` at BASE; classify each file by hand: `off`, `served`, `fetch`, `extract`, or
   production. A file that fits no class is a stop.
3. On a scratch clone at BASE, make `bundledExtract` (`limits_reads.go:378`) return an empty extract and run
   `go test ./...` there with the goldens unchanged: every test and golden that fails is an implicit extract dependent
   (MH55), listed with what it asserted. The review's five (the Claude `text.json:15` golden, Kilo's and OpenCode's
   messages-route goldens, `claude-refuses-forced`, `TestStreamTextDeltas_0051`) must be among them, or the run is wrong.
4. On the same clone, with `WithoutModelMetadata` given, record MH4's baseline: the rankings, `ModelInfo` and `Limits` the
   MH4 test compares against, for each provider it covers.
5. List every test the build deletes (MH30), each with the record row it pinned.
6. List every live assertion that needs a document (MH32), with the fixture it will hand in and the fixture's value read
   against the assertion, by hand; no live run unless the owner asks.
7. Pick, from the extract at BASE, a Kilo model the extract states a limit for and Kilo's static list names (MH51), and an
   OpenCode model whose canonical id is in the `anthropic` section (MH52).
8. Record each F6 fixture's provenance from its record or commit, for the `NOTICE`.
9. Run `python3 -B tools/apidiff.py v1.32.0` at BASE; its lines must be the 22 compatible lines of the MADR's expected
   output; a difference is a stop.

Exit: the classification, the implicit-dependent list, MH4's baseline, the deletion list, the live table, the chosen models
and the `apidiff` lines are in the execution record.

### Phase 1: red-first tests

Write the rows' tests. A test of existing behaviour goes in a file that builds at BASE, and
`python3 -B tools/redgreen.py SPEC --expect fail --rev BASE` (with `drop` for any function naming a new name) shows it
failing on its assertion; a test naming `catalog.WithMetadata` or `Limits.RouteFallback` takes its red from its plant
(Phase 3). Each failing line is quoted in the execution record.

| Row | Test | Package | Red at BASE because |
| :--- | :--- | :--- | :--- |
| MH1 | `TestWithMetadata_Shared_0093` | `llmprovider/providers` | new name: its plant |
| MH2 | `TestWithMetadata_Scope_0093` | `llmprovider/catalog`, `llmprovider/providers` | new name |
| MH3 | `TestWithMetadata_LastWins_0093` | `llmprovider/catalog` | new name |
| MH4 | `TestWithMetadata_NoneIsOff_0093` | `llmprovider/catalog` | new name; compares with Phase 0's baseline |
| MH5 | `TestWithMetadata_Swap_0093` (`-race`) | `llmprovider/catalog` | new name |
| MH6 | `TestWithMetadata_MergeRank_0093` | `llmprovider/catalog` | new name |
| MH7 | `TestWithMetadata_LimitsRank_0093` | `llmprovider/catalog` | new name |
| MH8 | `TestWithMetadata_EvidenceTime_0093` | `llmprovider/catalog` | new name |
| MH9 | `TestWithMetadata_AlsoLayer_0093` | `llmprovider/catalog` | new name |
| MH10 | `TestListRaw_IgnoresMetadata_0093` | `llmprovider/catalog` | new name |
| MH11 | `TestList_RanksHandedDocument_0093`; `TestList_NoDocumentRequest_0093` | `llmprovider/catalog` | the first new; the second: at BASE a listing fetches the default document through the test's counting client |
| MH12 | `TestMenu_FactsFromHandedDocument_0093` | `llmprovider/catalog` | new name |
| MH13 | `TestTogether_ListedOnlyHandedDocument_0093` | `llmprovider/catalog` | new name |
| MH14 | `TestCompat_CanonicalFromHandedDocument_0093` | `llmprovider/catalog` | new name |
| MH15 | `TestLookupModel_HandedDocument_0093` | `llmprovider/catalog` | new name |
| MH16 | `TestLimits_NoExtract_0093` | `llmprovider/catalog` | at BASE, with the fetch off, `claude-opus-4-6` states 128,000 from the extract |
| MH17 | `TestRouteFallback_NoExtract_0093` | `llmprovider/catalog` | at BASE the Claude route's fallback is the extract's 64,000 |
| MH18 | `TestOutputLimit_ColdMessagesRefused_0093` | `llmprovider/providers` | at BASE a cold Claude and OpenCode messages request sends the extract's fallback |
| MH19 | `TestLimitRule_MessageNamesWithMetadata_0093` | `llmprovider/internal/wire` | at BASE the message does not name `catalog.WithMetadata` |
| MH20 | `TestOpencode_RouteNoWait_0093`; `TestOpencode_RouteFromHandedDocument_0093` | `llmprovider/providers/opencode` | the first: at BASE a model outside the table waits for a document request through the counting client; the second new |
| MH21 | `TestProviders_FactsFromHandedDocument_0093` | `llmprovider/providers` | new name |
| MH22 | `TestWizard_HandedDocument_0093` | `wizard` | new name |
| MH23 | `TestNoMetadataFetch_0093` (a `go/parser` scan of string literals in non-test Go files) | `llmprovider/catalog` | at BASE `model_metadata.go:28` holds the address |
| MH24 | `TestNoEmbeddedExtract_0093` | `llmprovider/catalog` | at BASE `limits_reads.go:335` embeds the extract |
| MH25 | `TestNoticeNamesModelsDevData_0093` | `llmprovider/catalog` | at BASE there is no `NOTICE` |
| MH26 | `TestCatalogs_RequiresDoc_0093` | `llmprovider/internal/cmd/catalogs` | at BASE no `-doc` is accepted |
| MH27 | `TestNoMetadataEnv_0093` | `llmprovider/catalog` | at BASE `model_metadata.go:29-30` name the variables |
| MH34 | `test_no_limitsnapshot_0093` | `tools/test_tools.py` | at BASE the tool and its README row exist |
| MH35 | `test_offline_hands_document_0093` | `tools/test_tools.py` | at BASE the program carries `WithoutModelMetadata` |
| MH36 | `test_cachecensus_no_metadata_0093` | `tools/test_tools.py` | at BASE the anchor is listed |
| MH37 | `test_catalogcheck_requires_doc_0093` | `tools/test_tools.py` | at BASE `--doc` is optional |
| MH39 | api-check's accepted-removals case | `scripts/test_gates.py` | at BASE no removal is accepted |
| MH51 | `TestKilo_ColdSendsNone_0093` | `llmprovider/providers` | at BASE the extract's limit for Phase 0's Kilo model is sent |
| MH52 | `TestWithMetadata_OtherSectionsOnly_0093` | `llmprovider/catalog` | new name |
| MH53 | `TestLimits_RouteFallbackPerCall_0093` (`-race`) | `llmprovider/catalog`, `llmprovider/providers` | new name |
| MH54 | `TestLookupLimits_DocumentNotSeenByRequest_0093` | `llmprovider/providers` | new name |
| MH56 | `test_offline_refuses_before_v1_33_0` | `tools/test_tools.py` | at BASE an older `VERSION` is accepted |

Exit: every base-red test red on its assertion, quoted; every other test written.

### Phase 2: build

In layers, each `go build ./...` and `go vet ./...` clean before the next:

1. **`catalog`.** `WithMetadata` beside `WithModelLayer`, read in `configFor`, the last one winning. The call's document
   replaces `loadModelMetadata`, `cachedMetadata`, `peekMetadata` and `startModelMetadata` in `discovery.go`, `lookup.go`,
   `limits.go`, `menu.go` and `canonical.go`. The fetch, its constants and types, `metadataNow`, `metadataDecodes`, the store's
   metadata maps and their `Close` handling, and `metadataFetchedAt` go; so do the extract, `bundledExtract`, `rankBundled`
   and its source names. The document's evidence has a zero `Seen`, `Stale` false. `Limits.RouteFallback` is set by
   `assembleLimits`; the memo's `SetFallback`, `Fallback` and note fields go (MH53). `LookupLimits`' doc comment names what it
   warms (MH54). The doc comments of `Metadata` and its readers say the document is the caller's (MH43).
2. **`llmprovider`.** Delete `WithModelMetadataURL`, `WithoutModelMetadata`, the two `Settings` methods and their
   `providerConfig` fields (Q1). `internal/wire/limit_rule.go`'s message names `WithMaxTokens` or `catalog.WithMetadata`.
3. **Providers, wizard, commands.** Claude and OpenCode read the fallback from the `Limits` they read with their own options.
   OpenCode's route reads the document given to `New`; no lookup, no wait; `errMetadataOff` and `metadata` go. Every
   provider keeps `WithMetadata` in the options it passes to `catalog` (MH21). `internal/cmd/catalogs` requires `-doc`.
   `llmtest` and `wirecase` drop `WithoutModelMetadata`.
4. **The fixture.** `llmprovider/catalog/testdata/extract-2026-10-12/api.json`: the extract at BASE re-expressed as a
   models.dev document (each section, each model, its `limit` fields only), written by a scratch script that then reads both
   back and fails on any model or value that differs, seen failing on a planted difference first.
5. **Tests.** `off` files lose the option; `served` files parse the bytes they served and hand them in; the `fetch` tests on
   Phase 0's list go; the `extract` tests and the implicit dependents of Phase 0 hand in the fixture (each provider package's
   test helper parses it; G-wire's cases get it through each package's build function, so `wirecase` keeps its import rule);
   their expected values change only as the MADR's D12 (a) to (d) say; `limitmemo`'s fallback cases go; the live tests on
   Phase 0's list change as it says.
6. **Tools, gates and the notice.** Delete `tools/limitsnapshot.py`, its two tests and its README row; change
   `tools/offline.py` (program, plants, the `VERSION` check), `tools/cachecensus.py` and `tools/catalogcheck.py`; give
   `scripts/check_api.py` the accepted set keyed by `v1.32.0`, each line citing this record, with its case in
   `scripts/test_gates.py`. Add `NOTICE` and `LICENSE.models.dev` at the root.
7. `go mod tidy -diff` is clean.

Checks of the migration, each a scratch script, each seen failing on a planted input first (a copy, never the tree):

* MH28 and MH29: a diff of each `off` and `served` file against BASE in which every removed line holds the option and every
  added line is the same line without it, or the parse-and-hand-in of the bytes the file served; planted: one other changed
  line.
* MH30: `go test -list . ./...` at BASE and after; every name missing after is on Phase 0's list, and every listed name is
  missing; planted: one more test deleted.

Exit: `go test ./...` through `run-heavy` passes; `python3 -B tools/goldendiff.py --rev BASE` reports 0 findings.

### Phase 3: plants

One production change each, on a scratch clone, by `python3 -B tools/plant.py SPEC.json`, each caught only by its row's
test, the failing line quoted.

| Row | Plant | Caught by |
| :--- | :--- | :--- |
| MH1 | `WithMetadata` a scoped option for the built-in ids only | `TestWithMetadata_Shared_0093` (chatcompat refuses it) |
| MH2 | `configFor` falls back to a package variable holding the last document any call handed in | `TestWithMetadata_Scope_0093` |
| MH3 | the first `WithMetadata` kept | `TestWithMetadata_LastWins_0093` |
| MH4 | `LookupModel` returns an error naming a missing document | `TestWithMetadata_NoneIsOff_0093` |
| MH5 | the menu reads the document from a variable the latest call set | `TestWithMetadata_Swap_0093` |
| MH6 | the document merged above the listing | `TestWithMetadata_MergeRank_0093` |
| MH7 | the document's evidence ranked above the service | `TestWithMetadata_LimitsRank_0093` |
| MH8 | the document's evidence marked `Stale` | `TestWithMetadata_EvidenceTime_0093` |
| MH9 | a "models.dev" layer skipped when a document is handed in | `TestWithMetadata_AlsoLayer_0093` |
| MH10 | `ListRaw` refuses the option | `TestListRaw_IgnoresMetadata_0093` |
| MH11 | (a) the curation given no document; (b) a GET of `/api.json` through the client when none is handed in | (a) `TestList_RanksHandedDocument_0093`; (b) `TestList_NoDocumentRequest_0093` |
| MH12 | the menu's facts built without the document | `TestMenu_FactsFromHandedDocument_0093` |
| MH13 | `listedOnly` false for Together | `TestTogether_ListedOnlyHandedDocument_0093` |
| MH14 | the canonical-id read skipped for chatcompat | `TestCompat_CanonicalFromHandedDocument_0093` |
| MH15 | `LookupModel` drops the document layer | `TestLookupModel_HandedDocument_0093` |
| MH16 | a one-entry extract (`claude-opus-4-6`, 128,000) added back | `TestLimits_NoExtract_0093` |
| MH17 | the route fallback 64,000 for Claude when nothing states one | `TestRouteFallback_NoExtract_0093` |
| MH18 | a cold Messages request sends 64,000 | `TestOutputLimit_ColdMessagesRefused_0093` |
| MH19 | the message without `catalog.WithMetadata` | `TestLimitRule_MessageNamesWithMetadata_0093` |
| MH20 | (a) a GET of `/api.json` for a model outside the table; (b) `provider.npm` not read | (a) `TestOpencode_RouteNoWait_0093`; (b) `TestOpencode_RouteFromHandedDocument_0093` |
| MH21 | Claude strips `WithMetadata` from the options it passes to `catalog` | `TestProviders_FactsFromHandedDocument_0093` |
| MH22 | the wizard drops `WithMetadata` from its listing options | `TestWizard_HandedDocument_0093` |
| MH23 | `const defaultModelMetadataURL = "https://models.dev/api.json"` in `model_metadata.go` | `TestNoMetadataFetch_0093` |
| MH24 | a `//go:embed` of a JSON file in `catalog` | `TestNoEmbeddedExtract_0093` |
| MH25 | one data file's name taken out of `NOTICE` | `TestNoticeNamesModelsDevData_0093` |
| MH26 | with no `-doc`, the command proceeds with an empty `Metadata` | `TestCatalogs_RequiresDoc_0093` |
| MH27 | `"LLMPROVIDER_MODELS_METADATA_URL"` in a non-test file | `TestNoMetadataEnv_0093` |
| MH31 | 0089's existing plants of the `extract` files' rows, re-grounded | their own tests |
| MH33 | the `catalog` floor raised past its measure | `make coverage-check` |
| MH34 | `tools/README.md`'s `limitsnapshot.py` row kept | `test_no_limitsnapshot_0093` |
| MH35 | the "a listing asked for" plant made to match nothing | `test_offline_hands_document_0093` |
| MH36 | the metadata anchor kept | `test_cachecensus_no_metadata_0093` |
| MH37 | `--doc` optional again | `test_catalogcheck_requires_doc_0093` |
| MH39 | a ninth removal not in the accepted set | `scripts/test_gates.py`'s api-check case |
| MH42 | one more exported name | `tools/apidiff.py v1.32.0` reports it |
| MH43, MH44 | 0033's existing `Encode` and `ParseMetadata` plants, re-run | their own tests |
| MH51 | Kilo sends the fallback of its route's models when nothing states one | `TestKilo_ColdSendsNone_0093` |
| MH52 | the canonical stage skipped when the provider's own section is absent | `TestWithMetadata_OtherSectionsOnly_0093` |
| MH53 | `assembleLimits` also writes the fallback to the memo, and Claude reads it from there | `TestLimits_RouteFallbackPerCall_0093` |
| MH54 | `CachedLimits` reads a document the last `LookupLimits` on the same `Cache` was given | `TestLookupLimits_DocumentNotSeenByRequest_0093` |
| MH55 | one fixture value changed from the extract's | `tools/goldendiff.py --rev BASE` reports it |
| MH56 | the `VERSION` check removed | `test_offline_refuses_before_v1_33_0` |

MH40, MH41, MH45, MH46 and MH57 take their plants in Phase 5.

Exit: every plant caught, each by its own row's test only.

### Phase 4: docs

Each returned as `path:line — fixed` or `path:line — open: <reason>`, then `docs: searched <paths>`. Lines are b2917f0's,
re-grounded in Phase 0:

* `README.md:190`, `:269-282`: the ranking reads the document a caller hands in; a paragraph on getting one from
  `https://models.dev/api.json`, `ParseMetadata`, `WithMetadata`, and what happens without one (MH4, MH18, MH51).
* `CHANGELOG.md`: a dated note of D10's exception after the preamble's promise (`:5-7`, MH57); `[Unreleased]` **Added**
  `catalog.WithMetadata` and `Limits.RouteFallback`; **Removed** the eight names, the two variables, the download and the
  extract; **Changed** the listing, the lookups and the limits read only a handed-in document, a request only its
  provider's, OpenCode's route never waits, a cold required-wire request is refused and a cold Kilo request sends none, the
  refusal's message; the **Added** entry's extract sentence (`:28-29`) and the **Changed** entry's 64,000 (`:66-67`)
  corrected.
* `docs/architecture.md:180`, `:263-267`, `:297-306` (deleted), `:316-327`, `:1136-1139`, `:1163`, and a dated note of the
  exception at `:1572-1576` (MH57).
* `docs/guides/api-standards.md`: R6 (`:110-115`, `:121-123`), R17 (`:212-223`), R30 (`:284-289`), R42 (`:332-333`) and R48
  (`:357-359`) with the exception and this record, R49 (`:367-379`).
* `docs/guides/adding-a-provider.md:95`; `docs/guides/migrating-from-mcplib.md:34`, `:293`, `:395`.
* `AGENTS.md:237-246`: a test serves its own replies, and hands a parsed document in when it needs one.
* `tools/README.md:29` (deleted), `:32`, `:33`.
* Doc comments: `catalog/doc.go:37-55`; `lookup.go:14-36`, `:71-78`, `:155-166`; `limits.go:79-96` (`Evidence`),
  `:139-163`, `:193-197`, and `RouteFallback`'s; `model_metadata.go:24-26`, `:251-258`, `:349-365`;
  `metadata_document.go:18-22`; `cache.go:50-58`, `:185-195`; `internal/limitmemo/limitmemo.go:7-12`; the providers'
  metadata sentences (`claude.go:110-111`, `gemini.go:74`, `grok.go:89-92`, `openai.go:107`, `ollama.go:84`, `kilo.go:86`,
  `together.go:57`, `:75`, `huggingface.go:61`, `:78`, `opencode.go:16`, `:111`, `:711-715`, `chatcompat.go:84`, `:95`);
  `wizard/configure.go:123-127`; `internal/cmd/catalogs/main.go:15-17`; `internal/localonly/localonly.go:115`.
* `ExampleWithMetadata` in `catalog` (R50): a document parsed from a string, handed in, a model's window printed, with
  `// Output:`.

Exit: every row fixed or open with its reason; `make docs-check markdownlint links-check parity-check` pass.

### Phase 5: records

* This MADR's execution amendment: claims rows on every doc line Phase 4 wrote, the two exception notes (MH57) and the
  `NOTICE`, checked at a scratch clone's commit and each planted false with `tools/claims.py --plants` (MH40, MH45, MH57).
* A dated note in each record of the MADR's "Relation to other records", additions only (MH41); every "now" claims row of
  0085, 0087, 0088 and 0089 on a file Phase 4 edited re-run with a plain id against a scratch commit, each that moved
  annotated with its new line.
* 0089-PLAN: a dated entry for v1.33.0's steps (MH46).
* `docs/README.md`: this pair's rows, the count, and an "I want to…" row, "hand the SDK a models.dev document".

Exit: `make records-check`, `python3 -B tools/recordlint.py --changed` and `python3 -B tools/claims.py --changed` pass.

### Phase 6: gates

| Command | Host | Expect |
| :--- | :--- | :--- |
| `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- make pre-add-check` | this Mac | 0 |
| `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test ./...` | this Mac | 0 |
| `~/.agents/skills/available-dev-hosts/scripts/run-on-host --need linux --race -- go test -race ./...` | the Linux machine (not WSL; a fallback to this Mac does not count) | 0 |
| `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- make parity-check dep-check coverage-check api-check contract-check generate-check records-check tables-check fixtures-check changelog-check docs-check markdownlint links-check disclosure-check gate-selftest tools-test` | this Mac | 0 |
| `make lint vuln`; `go vet -tags live_gateways ./...` | this Mac | 0 |
| `python3 -B tools/apidiff.py v1.32.0` | this Mac | exit 1 with exactly the MADR's expected lines |
| `python3 -B tools/goldendiff.py --rev BASE` | this Mac | 0, 0 findings |
| `python3 -B tools/claims.py --changed`; `--plants` on this record's spec | this Mac | 0; every plant caught |
| `python3 -B tools/recordlint.py --changed`; `python3 -B tools/denyscan.py --changed` | this Mac | 0; 0 |
| `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B tools/gate.py` | this Mac | 0 |

Each run's output goes to a scratch file, its exit code captured before any filter. Then an independent review of the build.

### Then: v1.33.0

Hand back to 0089-PLAN's steps, as its entry "2026-11-24: v1.33.0 re-scoped" orders them, with DL70's fetch dropped
(Phase 5's entry): the joint gates again (0089 Phase 4's `apidiff` check with the MADR's lines, DL59's and DL60's claims rows
re-run, the plants of 0088 and 0089 re-grounded); the short release review, blockers only, covering this change; the date
step; the tag and the GitHub Release on the owner's word in the same turn; after the tag, `tools/offline.py v1.33.0` and
`tools/consumer.py v1.33.0 --dependent <gitrepos>/gobble-cli`, the latter expected to fail gobble's build until gobble's own
change lands (recorded, not a blocker of the tag).

## Verification

* Every decided row has its test, seen red and green, and its plant caught.
* `apidiff` prints exactly the MADR's lines; `make api-check` passes with the accepted set; `goldendiff --rev BASE` reports 0.
* No non-test Go string literal names a models.dev or models.opencode.ai address or the two variables, and `catalog` embeds
  nothing (MH23, MH24, MH27).
* The docs rows are all fixed, and their claims rows pass and fail when planted.

## Rollout and Rollback

Rollout is v1.33.0. gobble moves to it in its own records (0071 Phase 2; 0072 D17): it hands the document to every listing,
lookup and limits call and to `provider.New`, and drops its six uses of the removed names. Rollback before the tag is
reverting this build's commits; after the tag, the removed names cannot return without a record, so the review before the tag
is the last point to stop.

## Execution record

Not started.
