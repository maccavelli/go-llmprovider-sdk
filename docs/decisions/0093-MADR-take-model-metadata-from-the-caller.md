---
status: accepted
date: 2026-11-24
decision-makers: repository owner
consulted: "the owner's rulings of 2026-11-24, relayed by the coordinator and quoted below, among them the answers to Q1 to Q4; the coordinator's assessment, which the owner then acted on; the first draft's review (nine findings and three stale claims), relayed by the coordinator; this repository at b2917f0 (first draft at 9e86748 with 0089's DL65 change uncommitted), read only, and a scratch clone of it; gobble-cli at ba8f623, read only, and its 0072 record as drafted outside its tree, read only; magic-cli-remote and mcp-server-magictools, read only, for uses of the names this record removes"
informed: "gobble-cli (0071's Phase 2 and its 0072 draft, D17: it hands the document to every listing, lookup and limits call and to provider.New, and drops its six uses of the removed names); magic-cli-remote and mcp-server-magictools (they call none of the names, read at their current trees)"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->
<!-- claims: self=b2917f0 gobble-cli=ba8f623 -->

# The SDK Takes a models.dev Document From Its Caller and Fetches None: `catalog.WithMetadata` In; the Download, Its Cache, Its Options and the Embedded Extract Out

Associated PLAN: [0093-PLAN-take-model-metadata-from-the-caller.md](0093-PLAN-take-model-metadata-from-the-caller.md)

*(2026-11-24, first draft. `python3 -B scripts/check_records.py --next` printed `0092` at 9e86748. 0092 is reserved by
drafts outside the tree (a completed Responses stream filled from its terminal output), so this record takes `0093`. Three
questions were open (Q1 to Q3).)*

*(2026-11-24, second draft. The SDK's HEAD is b2917f0, which commits 0089's DL65 change; `--next` still prints `0092`, and
this record keeps `0093`. Re-grounded at b2917f0: every line the first draft cited as "the index" or "staged"
(`AGENTS.md:237-246`, `CHANGELOG.md:5-7`, `:28-29`, `:66-67`, `llmprovider/internal/wire/limit_rule.go:78-81`,
`docs/guides/api-standards.md:110-135`, `:214`, `:332`, `:358`) holds the same text on the same line, and the evidence index
is checked at b2917f0; `tools/apidiff.py v1.32.0` at b2917f0 prints 22 compatible lines, not the 21 the first draft took
from 0089-PLAN (F11). The owner answered Q1 to Q3 and a new Q4; the review's findings are rows MH51 to MH57 and corrections
of D3, D7, D12, MH4, MH11, MH23, MH32 and MH43. No row is open; the record is `proposed` until the owner accepts it.)*

*(2026-11-24, accepted. Owner ruling, quoted exactly, given for this MADR and its PLAN together: "Accept; approve; build next (Recommended)".)*

*(2026-11-24, amendment A1, after the build's Phase 0 stopped on two findings (0093-PLAN's execution record, "Phase 0,
pass 1"). The owner's rulings, quoted exactly, relayed by the coordinator: on the OpenCode route, "Limits plus OpenCode's
route field (Recommended)"; on the listing goldens, "Regenerate the 5 goldens (Recommended)", chosen over keeping them so
that the goldens show listings ranked by the document, as a real caller (gobble) sees them. D12 and MH55 are annotated, D12
gains (e), and MH58 is added; the text they had stands above each annotation.)*

*(2026-11-24, amendment A2, after the build's Phase 2 stopped on three cases the record did not decide and put four of its
applications up for review (0093-PLAN's execution record, "pass 1 continuing, second run"). The owner's answers, quoted
exactly, relayed by the coordinator, are in "Owner questions at the build's second stop" below. D12 gains (f) and (g),
MH30 is annotated, and MH59 to MH61 are added; the text they had stands above each annotation.)*

## Context and Problem Statement

The SDK downloads a models.dev-format document itself. By default it reads `https://models.opencode.ai/api.json` (C1), once
per URL at a time, keeps it in the process's or the caller's `catalog.Cache` for ten minutes, refreshes it in the background
with an `ETag`, and remembers a failure for a minute. Listings rank by it, lookups merge it, requests read it, and OpenCode
routes by it. Since 0089 the SDK also embeds a limits-only extract of models.dev (`llmprovider/catalog/limits_snapshot.json`,
86,356 bytes, dated 2026-10-12, C11), the registry's offline tier, which ships with no licence notice.

gobble-cli, the harness, keeps its own copy of the same data, dated 2026-10-14, and its 0072 draft keeps a refreshed copy on
disk. So two programs fetch, cache and age the same document, from two addresses, on two schedules.

### Owner rulings

Each is the owner's, relayed by the coordinator, its label quoted as given.

| Ruling, as given | On | Rows |
| :--- | :--- | :--- |
| "make models.dev the source-of-truth for both the SDK and gobble." | which address model data comes from | MH45; MH23 |
| "scope the models.dev fetch/data/presentation out of the SDK. this is better suited to the agent harness CLI than the LLM provider. thoughts?" | where the fetch, the data and their presentation live | every MH row |
| The coordinator's assessment, which the owner then acted on: the SDK should take model facts, not fetch them. It keeps the parser (`catalog.ParseMetadata`, the `Metadata` type, `Encode`) and gains a way for the caller to hand in a metadata document. It drops its own fetch (`catalog.LookupMetadata`, `LookupMetadataWith`, `CachedMetadataWith`, `llmprovider.WithModelMetadataURL`, the default URL `https://models.opencode.ai/api.json` at `llmprovider/catalog/model_metadata.go:28`, its in-process cache and background refresh) and its built-in data (`llmprovider/catalog/limits_snapshot.json`, 86,356 bytes, and whatever generates it). The SDK then never reaches the network or disk for model metadata, and ships no models.dev data (the extract today has no licence notice). | the shape of the change | MH1 to MH46 |
| "Move it before v1.33.0" | timing: v1.33.0 is re-scoped again to include this; the tag waits for it | MH46 |

### Owner questions, answered (2026-11-24)

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| Q1. How the eight names of D10, in v1.32.0, go | "Remove all eight in v1.33.0 (Recommended)" | (1): removed in v1.33.0, a recorded exception to R42 and R48; the changelog's preamble and `docs/architecture.md`'s api-check paragraph carry a dated exception note; `make api-check` accepts exactly the eight against `v1.32.0` | MH38, MH39, MH42, MH57; MH3 (`WithoutModelMetadata` goes) |
| Q2. A cold request on a wire that requires an output limit, now the extract is gone | "Refuse required, Kilo sends none (Recommended)" | (1): a cold Claude or OpenCode messages-route request is refused before sending, its message naming `WithMaxTokens` or `catalog.WithMetadata` as the fix; a cold Kilo request sends no limit | MH18, MH19, MH51 |
| Q3. models.dev-derived data that stays in the module (F6) | "Keep both, add the MIT notice (Recommended)" | (1): the route table and the test fixtures stay; a `NOTICE` and `LICENSE.models.dev` are added | MH25, MH55 (its fixture) |
| Q4 (new, after review). Whether a provider's requests read a document handed only to a lookup | "Requests read only New's document (Recommended)"; and, of gobble's provider, "for both 0093 and here, i want to hand to provider.New" | a request reads only the document given to its provider's `New`; a lookup never warms a request; `LookupLimits`' doc comment (`limits.go:139-141`) is corrected | MH21, MH53, MH54; MH47's cross-reference |

### Owner questions at the build's second stop, answered (2026-11-24, A2)

Put by the coordinator in one prompt, each with three resolutions as 0093-PLAN's entry "pass 1 continuing, second run"
states them; each answer quoted exactly.

| Question | The owner's answer | Option taken | Rows it decides |
| :--- | :--- | :--- | :--- |
| A. Two route tests that serve their own document (no limit in it) and read the extract for their request's limit: `TestOpencode_RoutesFromMetadata/anthropic` and `TestOpencode_RouteWithoutMetadataUsesTable/absent_from_document` | "Give them WithMaxTokens (Recommended)" | (1): each gives an output limit with `WithMaxTokens`; its served document and its path assertion are unchanged; D12 gains (f) naming them | MH59; D12 (f) |
| B. Served tests whose assertion counted the fetch, met by `servedDoc`'s own read: `wizard`'s `TestConfigureLLM_ProviderOptionsReachTheListing`, and the `metaHits` assertion of `TestListModels_HonoursTheMetadataURLOption` in `opencode`, `huggingface` and `together` | "Delete as fetch tests (Recommended)" | (1): the count is an observation of the fetch and goes with it: the wizard test is deleted (0015-PLAN S10; MH22 holds `ProviderOptions` reaching the listing) and the three assertions (0013-MADR A4); the three provider tests keep their ListModels-against-catalog comparison under a name for the handed document | MH60; MH30 |
| C. `TestCachedModel_NeverWaits_0033`'s cold call, "ok false", which MH15 contradicts once the document is handed in | "Cold call hands in no document (Recommended)" | (1): the cold call hands in no document, keeping 0033's "a cold lookup never waits" (false within 50 ms); the later calls hand it in; MH15 keeps its own test; D12 gains (g) | MH61; D12 (g) |
| D. Four applications of the record put up for review: Claude's `TestListModels_DefaultBaseShape_0041`, `claude.ExampleNew`, `TestOpencode_RouteWithoutMetadataUsesTable/disabled` and `TestListRaw_NoSideEffects_0041` | "Accept, noted in the PLAN (Recommended)" | accepted as applications of MH55, D3, MH3 and MH29, as 0093-PLAN's list states them; the list is annotated there; no row changes | none new (MH3, MH29, MH55 as written) |

### Facts read at b2917f0

* **F1. The fetch.** `model_metadata.go:27-37`: the default URL, the variables `LLMPROVIDER_MODELS_METADATA_URL` and
  `LLMPROVIDER_DISABLE_MODELS_METADATA`, a 10-minute TTL, a 1-minute memory of a failure, a 5 s bound on a lookup.
  `LookupMetadata` (`:270`, C2), `LookupMetadataWith` (`:288`, C3), `CachedMetadataWith` (`:308`, C4) and `OptionsFromEnv`
  (`:437`, C5) are exported; `loadModelMetadata` (`:459-491`), `startMetadataFetch` (`:497-526`), `fetchModelMetadata`
  (`:541-564`) and `startModelMetadata` (`:674-685`) are the machinery. `llmprovider.WithModelMetadataURL` (`options.go:136`,
  C7), `llmprovider.WithoutModelMetadata` (`options.go:147`, C8), `(*Settings).ModelMetadataURL` (`settings.go:379`, C9) and
  `(*Settings).ModelMetadataDisabled` (`settings.go:383`, C10) configure it; `configFor` reads both (`config.go:216`, `:221`,
  C26).
* **F2. The parser and the type.** `ParseMetadata` (`metadata_document.go:18-32`), `Encode` (`:34-67`), `Name` (`:70`, C23),
  `Model` (`:74-80`), `Models` (`:138-150`), and the readers `ReasoningEfforts`, `InterleavedField` and `NPM`
  (`model_metadata.go:351-365`, C44 to C46); `Metadata` is safe for concurrent use (`:251-258`). After this record
  `ParseMetadata` is the only constructor of a non-zero `Metadata`, so every handed-in section has decoded.
* **F3. Who reads the document.**
  * a listing's ranking, started beside every listing but Ollama's, which returns before it (`discovery.go:334-335`, C47),
    and Kilo's, which ranks from its own listing (`:339`, C36), ranked by `metadataCurate` (`model_metadata.go:738-768`,
    C37), Together's ids limited to its section (`:781-783`);
  * the menu's facts and the static path (`discovery.go:562-575`; `menu.go:100-101`), which `Catalog.Search` reads;
  * a chatcompat server's facts by canonical id, built over every section (`canonical.go:79-96`, `:106`, C48);
  * `LookupModel` (`lookup.go:101-108`, C14), `CachedModel` (`:137`, C15) and `CachedFacts` (`:175`, C16), merged lowest
    (`:206-224`), with a gateway model's missing cache-read price filled from its maker's first-party section
    (`metadata_document.go:89-133`, C49);
  * `LookupLimits` (`limits.go:175-179`, C17; `:189`) and `CachedLimits` (`:211`): the registry tier by exact, canonical
    and family stage, the canonical stage across every section for any provider with a models.dev key (`:586`, C50), its
    `Seen` the fetch time and `Stale` past the TTL (`:347-351`, C38);
  * OpenCode's request route (`providers/opencode/opencode.go:716-739`, C18; `:995-1004`, C19);
  * every provider's request-time facts, through `CachedFacts` and `CachedLimits` with its own options (for example
    `providers/claude/limit.go:54-55`; `providers/kilo/kilo.go:545`, C51).
* **F4. The embedded extract.** `limits_reads.go:331-378` embeds `limits_snapshot.json` (C11) and parses it once
  (`bundledExtract`, `:378`, C52); `assembleLimits` adds it below the document, always `Stale` (`limits.go:353-354`, C12), and
  `routeFallback` reads its section last (`:439-442`, C13). `tools/limitsnapshot.py` writes it (`:35`, C28). With no document
  and nothing listed, the extract gives a Claude request 64,000 (`CHANGELOG.md:66-67`), an OpenCode messages-route request
  its route's smallest limit (0089 DL77), and a Kilo request its model's extract limit.
* **F5. A refusal when nothing is stated.** `llmprovider/internal/wire/limit_rule.go:78-81` (C34): on a wire that requires
  a limit, with no source and no route fallback, `ErrInvalidRequest`, naming `maxTokens` in models.json,
  `catalog.WithModelLayer`, `WithMaxTokens` and `Request.MaxOutputTokens` (0089 DL51). OpenCode's listing carries ids only
  (0089 DL11).
* **F6. Other models.dev-derived data.** OpenCode's route snapshot, 3,543 bytes, from models.opencode.ai's document of
  2026-09-27, embedded (`llmprovider/internal/opencoderoute/opencoderoute.go:26-31`, C20, C21), on which 0089 DL74 and DL77
  rely. Test fixtures every module download includes: `llmprovider/catalog/testdata/ranking-2026-10-12/metadata.json`
  (509,488 bytes), `ranking-2026-09-26/metadata.json` (58,217), `versions-2026-11-06/families.json` (14,633) and
  `metadata-0033/fixture.json` (5,643). The module has no `NOTICE` and no licence notice for any of them.
* **F7. Tests.** 148 Go files name the fetch, its options or the extract (the drafting step's `inventory.py`, at 9e86748):
  18 non-test; 86 test files whose only use is `WithoutModelMetadata` (193 call sites in all files); 22 that serve a
  document by `WithModelMetadataURL`; 16 that name the fetch's machinery; 6 that name the extract. Others depend on the
  extract without naming it: the Claude G-wire golden's `"max_tokens": 128000` (`providers/claude/testdata/wire/claude/text.json:15`,
  C53), and, as the review found, Kilo's and OpenCode's messages-route goldens, `claude-refuses-forced` and
  `TestStreamTextDeltas_0051`; G-wire builds its providers with the fetch off (`internal/wirecase/wirecase.go:398`, C54).
  `AGENTS.md:237-246` (C55) and `internal/localonly/localonly.go:115` (C35) tell a test to serve its own document or turn it
  off.
* **F8. Tools and gates.** `tools/offline.py`'s program adds a parsed document as a layer and turns the fetch off (`:91-95`,
  C29); `tools/cachecensus.py` instruments the fetch (`:49-52`, C30); `tools/catalogcheck.py` passes `--doc` on (`:130`) to
  `internal/cmd/catalogs`, which otherwise fetches (`main.go:98-106`, C22). `make api-check` fails on any incompatible change
  against the latest `v1.X.Y` tag (`scripts/check_api.py:105-110`, C31), as `docs/architecture.md:1572-1576` says (C56), and
  `CHANGELOG.md:5-7` says a release that is not major only adds to the API (C57).
* **F9. The rules.** R42, "The identifier stays until a major version" (`docs/guides/api-standards.md:332-333`, C39); R48
  (`:357-359`, C40); R17's common options (`:214-215`, C41); R30's example `catalog.OptionsFromEnv()` (`:285`); R49's "the
  models.dev document" in `catalog.Cache` (`:367-370`).
* **F10. Who calls the names.** gobble-cli at ba8f623: `WithoutModelMetadata` at `llm/provider/provider.go:215` (C32),
  `signin.go:344`, `model.go:176`, `image_test.go:209`; `WithModelMetadataURL` at `list.go:130` (C33) and `:160`.
  magic-cli-remote and mcp-server-magictools call none (two magic-cli-remote records mention them).
* **F11. v1.33.0.** `python3 -B tools/apidiff.py v1.32.0` at b2917f0 exits 0 with 22 compatible lines, `WarningToolCallCut`
  and `APIError.Observed` among them (the drafting step's run, saved beside this draft). 0089-PLAN's entry "2026-11-24:
  v1.33.0 re-scoped" orders the remaining steps; DL65's measurement is committed (b2917f0).
* **F12. The route fallback is a shared note.** `assembleLimits` writes the route's fallback into the cache's limit memo
  (`limits.go:365-366`, C58), keyed by provider, base URL, scope, model and stream (`internal/limitmemo/limitmemo.go:31-37`,
  C59; `SetFallback`, `:170`, C60), and a request reads it back from there (`providers/claude/limit.go:68`, C61;
  `providers/opencode/opencode.go:1073`, C62). Nothing in the key names a document, so a fallback found with one call's
  document is read by another call's request.
* **F13. `LookupLimits`' doc** says it "warms the caches a request reads" (`limits.go:139-141`, C63).
* **F14. gobble's 0072 draft** (read at 2026-11-24, before its revision the coordinator relays) assumed the SDK's fetch (its
  D4), the SDK's extract offline (its F9) and `provider.New` handed no document (its D17); the coordinator relays that its D17
  now hands the document to `provider.New`.
* **F15. The apidiff spelling.** On two scratch modules, apidiff at `scripts/check_api.py`'s pin prints a removed function as
  `- ./llmprovider.Gone: removed` and a removed method as `- ./llmprovider.(*Settings).GoneMethod: removed`, under
  "Incompatible changes:", and exits 0 (the drafting step's probe, saved beside this draft).

## Decision Drivers

* The owner's rulings: the fetch, the data and their presentation belong to the harness; models.dev is the source.
* One meaning for "models.dev's facts" wherever the SDK reads them: ranking, menus, lookups, limits, routes.
* No call waits for, or sends, a request for model metadata; no file is read for it.
* A request reads only its provider's document, so what a lookup was given never leaks into a request.
* The API standards, changed only by a record.
* Every case decided up front, each with a test seen red and a plant caught.

## Considered Options

* A. `catalog.WithMetadata(m Metadata)`: a shared option carrying the parsed document, read per call and fixed at a
  provider's `New`
* B. `catalog.WithMetadataFunc(f func() Metadata)`: a shared option carrying a source, read once per call and per request
* C. `(*catalog.Cache).SetMetadata(m Metadata)`: the document held by a caller-owned cache and swapped in place
* D. Keep the fetch, point its default at models.dev, and drop only the extract

## Decision Outcome

Chosen option: "A", because it is the smallest change that meets the rulings: one option, the same shape as
`catalog.WithModelLayer` and `WithLastGood` (R17), a value the caller owns and swaps between calls, and no state in the SDK;
and Q4's answer fixes a provider's requests to the document given to its `New`, which A gives directly. B puts a caller
function on every request's path for a document that changes once a day. C needs a `Cache` for any document at all and keeps
the caller's data in the SDK's state, which D2 takes out of R49. D keeps the SDK fetching, which the rulings move out.

### The decisions

* **D1. The hand-in.** `catalog.WithMetadata(m Metadata) llmprovider.Option`, a shared option (R17, R18) named
  "catalog.WithMetadata", whose value is the `Metadata` itself. A built-in provider and chatcompat read it; a provider outside
  the module ignores it; `llmprovider.For(id, …)` scopes it.
* **D2. Scope.** A call's document is the one its options hand in: `List`, `LookupModel`, `CachedModel`, `CachedFacts`,
  `LookupLimits`, `CachedLimits` and every listing the wizard makes. A provider's is the one given to `New`, fixed for its
  life (R20), and its requests read only that one (Q4): a document handed to a lookup is never seen by a request. There is no
  process-wide document and none in `catalog.Cache` (R29, R49). Given twice, the last wins. `WithoutModelMetadata` goes
  (Q1), so `WithMetadata(catalog.Metadata{})` given last is how a caller drops a document an earlier option in a shared list
  carried.
* **D3. None.** No document, or a zero `Metadata`, or a parsed document with no section at all: every consumer behaves as
  at BASE with `WithoutModelMetadata` and the extract emptied (`limits_reads.go:378` returning an empty extract). It is not
  an error and starts no request. A document that has some sections but none for the provider is a document: its canonical
  stage in `Limits` (`limits.go:586`), a chatcompat server's canonical facts (`canonical.go:106`) and the first-party price
  fill (`metadata_document.go:89-133`) read it; the provider's own ranking and exact and family stages find nothing.
* **D4. Concurrency.** A call reads its document once and uses it for its whole run, so a listing's curation and its menu
  read the same one (0041-MADR D3). A swap between calls changes the next call only. `Metadata` is immutable (R51).
* **D5. Rank.** The document takes the fetched one's place, with nothing else changed: in `ModelInfo` lowest, below the
  listing and the caller's layers, source "models.dev" (or "models.dev (canonical id)"); in `Limits` the registry, below a
  caller layer named "models.dev" (0089 DL15) and above a remembered cap, by exact, canonical and family stage (DL71). Its
  evidence has a zero `Seen` and `Stale` false: its age is the caller's. Given both as `WithMetadata` and as
  `WithModelLayer`, it is read in both places.
* **D6. No fetch.** The default URL, the download, its cache entries, refresh, `ETag` and failure memory, the two variables
  and their helper go. OpenCode's route is the pinned route, else the document's `provider.npm`, else the route table, else
  the name rule, at once. `internal/cmd/catalogs` reads its document only from `-doc`, now required.
* **D7. No extract, and the route fallback per call.** `limits_snapshot.json`, its embed, its parser, its source names and
  `tools/limitsnapshot.py` go. The route's fallback is carried in the `Limits` value a call returns, as
  `Limits.RouteFallback`, a `Fact` (the cached listing's route models, then the document's, then unknown), not in the cache's
  memo; the memo keeps the caps and the resolved id. A request reads the fallback from the `Limits` it read with its own
  options.
* **D8. A cold request on a required wire** (Claude, OpenCode's messages route: no listing cached, no document, no layer,
  no limit given) is refused before sending, `ErrInvalidRequest` (Q2). A cold Kilo request sends no limit (Q2): Kilo's wire
  does not require one, and with no fact none is sent.
* **D9. The refusal's message** names `WithMaxTokens` or `catalog.WithMetadata` as the fix, beside the fixes it names now
  (F5). 0089 DL51's ruling stands.
* **D10. The removed names** (Q1): `catalog.LookupMetadata`, `catalog.LookupMetadataWith`, `catalog.CachedMetadataWith`,
  `catalog.OptionsFromEnv`, `llmprovider.WithModelMetadataURL`, `llmprovider.WithoutModelMetadata`,
  `(*llmprovider.Settings).ModelMetadataURL` and `(*llmprovider.Settings).ModelMetadataDisabled`, removed in v1.33.0 as a
  recorded exception to R42 and R48. `make api-check` accepts exactly these eight against `v1.32.0`; the changelog's preamble
  and `docs/architecture.md`'s api-check paragraph carry a dated note of the exception. The two variables have no effect.
* **D11. The licence** (Q3). The route table and the fixtures stay; a `NOTICE` at the root names each file of models.dev
  data in the module (the route snapshot, the fixtures of F6, and MH55's fixture) and credits models.dev under the MIT
  licence, whose text is `LICENSE.models.dev` beside it.
* **D12. Tests.** A test that only turned the fetch off loses that option. A test that served a document by URL parses the
  same bytes and hands them in, its expectations unchanged (R46). A test of the fetch goes with the fetch, named in the PLAN
  with the row it pinned. A test or golden that read the extract, by name or not (Phase 0 lists every one by a run at BASE
  with the extract emptied), hands in one committed fixture that states the extract's limits as a models.dev document
  (`llmprovider/catalog/testdata/extract-2026-10-12/api.json`), so that every G-wire golden is unchanged
  (`tools/goldendiff.py --rev BASE`: 0 findings). The expected values that change are these, and no others: (a) an evidence
  source "models.dev (bundled 2026-10-12)", with its stage suffixes, becomes "models.dev", "models.dev (canonical id)" or
  "models.dev (family)"; (b) that evidence's `Seen` becomes zero and its `Stale` false; (c) a test that keeps no document
  on purpose and asserted the extract's value now asserts MH16's, MH17's, MH18's or MH51's outcome; (d) `limitmemo`'s
  fallback cases go with the note (MH53). `AGENTS.md`'s test rule and `localonly`'s message say a test serves its own replies
  and hands in a document when it needs one.
  *(Amended 2026-11-24, A1. The fixture also carries, in its `opencode` and `opencode-go` sections, `provider.npm` for each
  model BASE's route table or heuristic routes off chat completions (73 entries: `@ai-sdk/anthropic` for messages,
  `@ai-sdk/openai` for responses, `@ai-sdk/google` for Google; none for chat completions), so that OpenCode routes every
  model as BASE did with no document (D6) and every request golden holds; the script that writes it checks both the limits
  and the routes against BASE. And (e): the five listing goldens `claude/listing.json` and
  `opencode-zen-{responses,messages,google,chat}/listing.json` are regenerated with the fixture handed in, their `result`
  the document's ranking (D5, MH11); `tools/goldendiff.py --rev BASE` takes them as a named exception, a change to their
  `result` only, and nothing else may differ (MH58). The no-document listing path stays held by MH11's ranking tests.)*
  *(Amended 2026-11-24, A2. (f): `TestOpencode_RoutesFromMetadata/anthropic` and
  `TestOpencode_RouteWithoutMetadataUsesTable/absent_from_document` serve a document with no limit in it and send a
  request on OpenCode's messages route, which read the extract's limit at BASE; a call has one document (D2), so each gives
  `WithMaxTokens`, its served document and its path assertion unchanged (MH59, question A). (g): the cold call of
  `TestCachedModel_NeverWaits_0033` hands in no document, its expectation "ok false within 50 ms" unchanged; its later
  calls hand the served document in (MH61, question C). A served test's assertion that counted the fetch is a test of the
  fetch, and goes as the third sentence says (MH60, question B).)*
* **D13. Tools and gates.** `tools/offline.py`'s program hands the document in with `WithMetadata` and nothing else; its
  plants become "no document handed in" and "a listing asked for"; for a `VERSION` before `v1.33.0` it exits 2, naming the
  revision whose copy of the tool checks that tag (b2917f0). `tools/cachecensus.py` drops its metadata anchor.
  `tools/catalogcheck.py` requires `--doc`. `scripts/check_api.py` accepts D10's eight against `v1.32.0`.
* **D14. Docs and records.** The SDK's docs name `https://models.dev/api.json` as the document a caller downloads and hands
  in; no SDK code holds an address as a string literal (MH23). The records change as "Relation to other records" says.
* **D15. v1.33.0.** This record's build lands before the gates' re-run; DL70's fetch and regeneration is dropped; then the
  short release review covers this change too, then the date step and the tag on the owner's word.
* **D16. Exports** (R55). `catalog.WithMetadata` (call sites: gobble's listings, lookups, limits calls and `provider.New`;
  the wizard's `ProviderOptions`; `tools/offline.py`'s program; `internal/cmd/catalogs`) and the field
  `Limits.RouteFallback` (call sites: `providers/claude/limit.go`, `providers/opencode`'s `limitInput`). `Limits` is new
  since v1.32.0, so its field adds no `apidiff` line.

### Consequences

* Good, because the SDK makes no request and reads no file for model metadata, and ships no extract.
* Good, because the harness decides where the document comes from, how old it may be and how its age is shown.
* Good, because a request reads only its provider's document, and no other call's document leaks into it (MH53, MH54).
* Neutral, because a caller that hands in nothing gets the static and name curators, as `WithoutModelMetadata` gave.
* Bad, because a program that never lists, hands in no document and sets no limit is refused on Claude's and OpenCode's
  messages route, and sends no limit to Kilo, where the extract answered before.
* Bad, because eight v1 names are removed in a minor release, by a recorded exception.
* Bad, because about 130 test files change, mostly by deleting one option, and 16 files' fetch tests go.
* Bad, because a provider built before the caller's refresh keeps its document until it is built again (D2).

### Confirmation

* The PLAN's tests, one per decided row, each seen red at BASE on its assertion (or, for a new name, red from its plant),
  each held by a plant through `tools/plant.py` on a scratch clone.
* `tools/apidiff.py v1.32.0` prints exactly MH42's lines; `tools/goldendiff.py --rev BASE` reports 0 findings.
* The gates: `make pre-add-check` and `go test ./...` through `run-heavy`; `-race` on the Linux machine through
  `run-on-host`; CI's make set; claims, recordlint and denyscan; an independent review of the build finds no blocker.

### State table

"Document" is the `Metadata` a call's or a provider's options hand in (D2). BASE is the build's base commit. Each decided
row's plant is the PLAN's, one to one; the test named first is the row's red test.

| Row | Input or state | Status | Outcome | Test | Decided by |
| :--- | :--- | :--- | :--- | :--- | :--- |
| MH1 | The hand-in's name and kind | decided | `catalog.WithMetadata(m Metadata)`, a shared option; built-ins and chatcompat read it; an outside provider ignores it; `For` scopes it | `TestWithMetadata_Shared_0093` | D1 |
| MH2 | Scope: calls and providers | decided | a call reads its own options' document; a provider the one given to `New`, for its life; no process-wide or cache-held document | `TestWithMetadata_Scope_0093` | D2 |
| MH3 | Given twice; a zero `Metadata` given last | decided | the last wins; a zero one given last drops the document (the way to drop one, `WithoutModelMetadata` being removed) | `TestWithMetadata_LastWins_0093` | D2; Q1 |
| MH4 | None; a zero `Metadata`; a document with no section | decided | BASE's behaviour with `WithoutModelMetadata` and the extract emptied (`limits_reads.go:378`), recorded at BASE as the baseline; no error; no request | `TestWithMetadata_NoneIsOff_0093` | D3 |
| MH5 | The caller swaps documents between calls, under `-race` | decided | each call wholly one document's (curation and menu alike); calls in flight unaffected | `TestWithMetadata_Swap_0093` | D4 |
| MH6 | `ModelInfo` merge | decided | the document lowest, then the listing, then the caller's layers; source "models.dev" | `TestWithMetadata_MergeRank_0093` | D5 |
| MH7 | `Limits` rank | decided | caller layers, service, registry ("models.dev" layer, then the document, by stage), remembered | `TestWithMetadata_LimitsRank_0093` | D5 |
| MH8 | The document's evidence time | decided | `Seen` zero, `Stale` false | `TestWithMetadata_EvidenceTime_0093` | D5 |
| MH9 | One document given as `WithMetadata` and as `WithModelLayer` | decided | read in both places | `TestWithMetadata_AlsoLayer_0093` | D5 |
| MH10 | `ListRaw` given `WithMetadata` | decided | accepted, unread (0041-MADR D9) | `TestListRaw_IgnoresMetadata_0093` | D2 |
| MH11 | A listing's ranking (every provider but Kilo and Ollama, which read no document: `discovery.go:334-339`) | decided | with a document, the sixes a fetched one gave on the committed snapshots; without, the fallback curators; no request for a document either way | `TestList_RanksHandedDocument_0093`; `TestList_NoDocumentRequest_0093` | D5, D6 |
| MH12 | The menu's facts and the static path; `Catalog.Search`'s fact words | decided | from the document, the cached listing and the caller's layers | `TestMenu_FactsFromHandedDocument_0093` | D5 |
| MH13 | Together's listed-only rule | decided | with a document, only its section's ids rank; without, `curateTogether` | `TestTogether_ListedOnlyHandedDocument_0093` | D5 |
| MH14 | A chatcompat server's facts by canonical id | decided | from the document; without one, the listing only | `TestCompat_CanonicalFromHandedDocument_0093` | D5 |
| MH15 | `LookupModel`, `CachedModel`, `CachedFacts` | decided | the document as the models.dev layer; nothing fetched; no document error | `TestLookupModel_HandedDocument_0093` | D5, D6 |
| MH16 | `LookupLimits`, `CachedLimits` with no document | decided | no extract evidence; a model only the extract knew is unknown | `TestLimits_NoExtract_0093` | D7 |
| MH17 | The route fallback with no document | decided | the cached listing's route models, then unknown; no extract | `TestRouteFallback_NoExtract_0093` | D7 |
| MH18 | A cold Claude or OpenCode messages-route request | decided | refused before sending, `ErrInvalidRequest`; nothing sent | `TestOutputLimit_ColdMessagesRefused_0093` | D8; Q2 |
| MH19 | DL51's refusal message | decided | names `WithMaxTokens` or `catalog.WithMetadata` as the fix, beside its existing fixes | `TestLimitRule_MessageNamesWithMetadata_0093` | D9; Q2 |
| MH20 | OpenCode's request route | decided | pinned, else the document's `provider.npm`, else the table, else the name rule; never waits; no request | `TestOpencode_RouteNoWait_0093`; `TestOpencode_RouteFromHandedDocument_0093` | D6 |
| MH21 | A provider's request-time facts (efforts, replay field, thinking shape, output limit) | decided | from the document given to `New` only | `TestProviders_FactsFromHandedDocument_0093` | D2; Q4 |
| MH22 | The wizard | decided | `ProviderOptions` carry the document to every listing it makes; without one, the curators; it fetches nothing | `TestWizard_HandedDocument_0093` | D2, D6 |
| MH23 | An address or fetch for model metadata in non-test code | decided | no string literal in a non-test Go file names models.dev, models.opencode.ai or `api.json` (a `go/parser` scan of literals; comments may name the address, as D14's docs do); no fetch | `TestNoMetadataFetch_0093` | D6, D14 |
| MH24 | Embedded models.dev limits | decided | none: no `limits_snapshot.json`, no `//go:embed` in `catalog` | `TestNoEmbeddedExtract_0093` | D7 |
| MH25 | The route snapshot and the test fixtures (F6) | decided | kept; `NOTICE` names each models.dev data file in the module, each exists, and `LICENSE.models.dev` holds the MIT text | `TestNoticeNamesModelsDevData_0093` | D11; Q3 |
| MH26 | `internal/cmd/catalogs` with no `-doc` | decided | a usage error, exit 2; it fetches nothing | `TestCatalogs_RequiresDoc_0093` | D6 |
| MH27 | The two environment variables | decided | no non-test Go file names them | `TestNoMetadataEnv_0093` | D10 |
| MH28 | Tests that only turned the fetch off | decided | the option deleted, nothing else changed | the PLAN's diff check | D12 |
| MH29 | Tests that served a document by URL | decided | the same bytes parsed and handed in; expectations unchanged | their own names; the PLAN's diff check | D12 |
| MH30 | Tests of the fetch itself | decided | deleted with it, each named in the PLAN with the record row it pinned *(A2: the list gains one test and three assertions, MH60)* | the PLAN's before-and-after test list | D12 |
| MH31 | Tests that name the extract | decided | MH55's fixture handed in; their expected values change only as D12 (a) to (c) say | their own names | D12 |
| MH32 | Live tests that rely on a document | decided | Phase 0 names each live assertion that needs one, with the fixture it will hand in, and checks the fixture's value against the assertion by hand (no live run unless the owner asks); a live test that needs none hands in none | the Phase 0 table; `go vet -tags live_gateways ./...` | D12 |
| MH33 | Coverage | decided | no package below its floor (R47); a fall stops the build | `make coverage-check` | R47 |
| MH34 | `tools/limitsnapshot.py`, its tests and README row | decided | removed | `test_no_limitsnapshot_0093` | D7 |
| MH35 | `tools/offline.py`'s program | decided | hands the document in with `WithMetadata`; plants "no document handed in" and "a listing asked for" each fail it | `test_offline_hands_document_0093` | D13 |
| MH36 | `tools/cachecensus.py` | decided | no metadata anchor; listing, Ollama show and probe kept | `test_cachecensus_no_metadata_0093` | D13 |
| MH37 | `tools/catalogcheck.py` | decided | `--doc` required | `test_catalogcheck_requires_doc_0093` | D13 |
| MH38 | The names of D10 | decided | removed in v1.33.0 | MH42's `apidiff` lines | D10; Q1 |
| MH39 | `make api-check` before v1.33.0's tag | decided | accepts exactly D10's eight against `v1.32.0`, each by name, citing this record; any other incompatible change fails | `scripts/test_gates.py`'s api-check case | D13; Q1 |
| MH40 | Docs (README, CHANGELOG, architecture, guides, package docs, tools README, AGENTS) | decided | each corrected as the PLAN's docs phase lists | claims rows in the execution amendment, each planted false | D14 |
| MH41 | Records (0009, 0013, 0015, 0021, 0028, 0033, 0041, 0048, 0058, 0061, 0065, 0071, 0078, 0088, 0089) | decided | a dated note each; the index gains this pair and its matrix row | `make records-check`; `tools/recordlint.py --changed` | D14 |
| MH42 | `tools/apidiff.py v1.32.0` | decided | exactly the lines under "The expected apidiff lines" below; exit 1 by the tool's design | the PLAN's gate table | D10, D16; Q1 |
| MH43 | `ParseMetadata`, `Metadata`, `Encode`, `Model`, `Models`, `Name`, `ReasoningEfforts`, `InterleavedField`, `NPM` | decided | kept, unchanged in behaviour; their doc comments say the document is the caller's | their existing tests | F2 |
| MH44 | A document past 32 MiB, or not a document | decided | `ParseMetadata` refuses it, as now | existing `ParseMetadata` tests | F2 |
| MH45 | The source address | decided | the docs name `https://models.dev/api.json`; no SDK code holds an address (MH23) | MH23's test; MH40's claims rows | the first ruling |
| MH46 | v1.33.0's remaining steps | decided | this build, then the gates again, then the short review (covering this change), the date step and the tag; DL70's fetch dropped | 0089-PLAN's dated entry | the fourth ruling; D15 |
| MH47 | gobble's 0071 Phase 2 and its 0072 draft | out of scope | gobble's records (0072-MADR D17, as the coordinator relays its revision): it hands the document to every listing, lookup and limits call and to `provider.New`, and drops its six uses of the names (F10) | | informed; Q4 |
| MH48 | gobble 0072's D4 fetch through `LookupMetadataWith` | out of scope | gobble's record: it fetches with its own request | | informed |
| MH49 | An SDK-side persisted store of the document | out of scope | contrary to the rulings; the harness keeps the copy | | rulings |
| MH50 | A document's age or source shown to a person | out of scope | the harness's ("presentation", the second ruling) | | rulings |
| MH51 | A cold Kilo request: nothing listed, no document, no limit given | decided | no limit sent (`LimitBasisNone`), where BASE sent the extract's | `TestKilo_ColdSendsNone_0093` | D8; Q2 |
| MH52 | A document with sections, but none for the provider | decided | a document: the canonical stage across sections, a chatcompat server's canonical facts and the first-party price fill read it; the provider's ranking and exact and family stages find nothing | `TestWithMetadata_OtherSectionsOnly_0093` (one `anthropic` section; an OpenCode model whose canonical id is in it) | D3 |
| MH53 | Two calls on one `Cache`, one with a document and one without, under `-race` | decided | each `Limits` carries its own `RouteFallback`; the memo holds no fallback; a request reads the fallback of the `Limits` it read | `TestLimits_RouteFallbackPerCall_0093` | D7; Q4 |
| MH54 | A document handed to `LookupLimits` (or any lookup), then a request of a provider built without one, on the same `Cache` | decided | the request sees no fact or fallback from that document (on Claude, refused as MH18); `LookupLimits`' doc names what it warms: the listing, the one-model and the served reads, not a document | `TestLookupLimits_DocumentNotSeenByRequest_0093` | D2; Q4 |
| MH55 | Tests and G-wire goldens that depend on the extract without naming it | decided | listed by a run at BASE with the extract emptied; each hands in `llmprovider/catalog/testdata/extract-2026-10-12/api.json`, the extract's limits as a document; every golden unchanged *(amended 2026-11-24, A1: the fixture's OpenCode sections also carry BASE's route as `provider.npm`; every request golden unchanged; the five listing goldens of D12 (e) regenerated)* | `tools/goldendiff.py --rev BASE`, 0 findings *(A1: with `--ranked-listings`, MH58)*; the listed tests | D12; Q3; A1 |
| MH56 | `tools/offline.py VERSION` with a tag before `v1.33.0` | decided | exit 2, naming b2917f0's copy of the tool for such a tag; nothing built | `test_offline_refuses_before_v1_33_0` | D13 |
| MH57 | The changelog's preamble (`CHANGELOG.md:5-7`) and `docs/architecture.md:1572-1576` | decided | each gains a dated note of D10's exception, naming this record | claims rows on both notes, planted false | D10; Q1 |
| MH58 | The G-wire goldens after the build, against BASE (added 2026-11-24, A1) | decided | `tools/goldendiff.py --rev BASE --ranked-listings` accepts a change to the `result` of exactly the five listing goldens of D12 (e); a change to any other golden, or to a request of one of the five, is a finding | `TestGoldenDiff.test_ranked_listings_0093` (`tools/test_tools.py`) | D12 (e); A1 |
| MH59 | A route test that serves its own document, with no limit in it, and sends a request on OpenCode's messages route: `TestOpencode_RoutesFromMetadata/anthropic` and `TestOpencode_RouteWithoutMetadataUsesTable/absent_from_document` (added 2026-11-24, A2) | decided | each gives `WithMaxTokens`; its served document and its path assertion unchanged: the document's `provider.npm` routes a model outside the table to `/messages`, and a model the document does not list keeps the table's `/messages` | `TestOpencode_RoutesFromMetadata/anthropic`; `TestOpencode_RouteWithoutMetadataUsesTable/absent_from_document` | D12 (f); A2, question A |
| MH60 | A served test's assertion that counted the fetch (added 2026-11-24, A2) | decided | deleted with the fetch, as MH30: `wizard`'s `TestConfigureLLM_ProviderOptionsReachTheListing` (it pinned 0015-PLAN S10; MH22 holds `ProviderOptions` reaching the listing) and the `metaHits` assertion of `TestListModels_HonoursTheMetadataURLOption` in `opencode`, `huggingface` and `together` (it pinned 0013-MADR A4); those three keep their ListModels-against-catalog comparison, renamed `TestListModels_HonoursTheHandedDocument` | MH30's before-and-after test list; the PLAN's assertion check of the three | D12; MH30; A2, question B |
| MH61 | `TestCachedModel_NeverWaits_0033`'s cold call (added 2026-11-24, A2) | decided | it hands in no document, and `CachedModel` returns `ok` false within 50 ms, waiting for nothing; MH15's test holds a handed document's answer | `TestCachedModel_NeverWaits_0033` | D12 (g); A2, question C |

Counts: 53 decided, 4 out of scope (MH47 to MH50), 0 open; 57 rows. *(2026-11-24, A1: 54 decided, 4 out of scope, 0 open;
58 rows.)* *(2026-11-24, A2: 57 decided, 4 out of scope, 0 open; 61 rows.)*

### The expected apidiff lines

`python3 -B tools/apidiff.py v1.32.0` after the build prints exactly these, and exits 1 (the tool exits 1 on any
incompatible change; F15):

```text
Incompatible changes:
- ./llmprovider.(*Settings).ModelMetadataDisabled: removed
- ./llmprovider.(*Settings).ModelMetadataURL: removed
- ./llmprovider.WithModelMetadataURL: removed
- ./llmprovider.WithoutModelMetadata: removed
- ./llmprovider/catalog.CachedMetadataWith: removed
- ./llmprovider/catalog.LookupMetadata: removed
- ./llmprovider/catalog.LookupMetadataWith: removed
- ./llmprovider/catalog.OptionsFromEnv: removed
Compatible changes:
- ./llmprovider.(*Settings).MaxTokensGiven: added
- ./llmprovider.APIError.Observed: added
- ./llmprovider.LimitBasis: added
- ./llmprovider.LimitBasisFact: added
- ./llmprovider.LimitBasisFallback: added
- ./llmprovider.LimitBasisNone: added
- ./llmprovider.LimitBasisOption: added
- ./llmprovider.LimitBasisRefusal: added
- ./llmprovider.LimitBasisRequest: added
- ./llmprovider.LimitObservation: added
- ./llmprovider.Response.OutputLimit: added
- ./llmprovider.SentLimit: added
- ./llmprovider.Warning.Observed: added
- ./llmprovider.WarningLimitLearned: added
- ./llmprovider.WarningToolCallCut: added
- ./llmprovider.WithLimitObservations: added
- ./llmprovider.WithoutLimitLearning: added
- ./llmprovider/catalog.CachedLimits: added
- ./llmprovider/catalog.Evidence: added
- ./llmprovider/catalog.Fact: added
- ./llmprovider/catalog.Limits: added
- ./llmprovider/catalog.LookupLimits: added
- ./llmprovider/catalog.WithMetadata: added
```

The order within each section is apidiff's; the PLAN's Phase 0 confirms it on a scratch clone, and a line that differs only
in order is recorded, not a deviation.

## Pros and Cons of the Options

### A. `catalog.WithMetadata(m Metadata)`

* Good, because it is one option, shaped as the shared options callers already pass (R17).
* Good, because the SDK holds no document between calls: nothing to refresh, expire or close.
* Bad, because a provider keeps the document it was built with until it is built again.

### B. `catalog.WithMetadataFunc(f func() Metadata)`

* Good, because a long-lived provider reads each refreshed document at once.
* Bad, because a caller's function runs inside every listing and request, and must not block, as `Layer.Model` must
  (`lookup.go:18-20`).
* Bad, because a call that read the source twice could mix two documents unless the SDK reads it once per call.

### C. `(*catalog.Cache).SetMetadata(m Metadata)`

* Good, because one swap serves every listing and provider sharing the cache.
* Bad, because a caller with no `Cache` cannot hand in a document (R29, R49).
* Bad, because it keeps the document in the SDK's state, the opposite of the rulings.

### D. Keep the fetch, at models.dev, without the extract

* Good, because callers change nothing.
* Bad, because the SDK still fetches, caches and ages the document, which the owner moved to the harness.

## More Information

### Relation to other records

A dated note is added to each, by the PLAN's records phase; no earlier text is rewritten.

* `0009-MADR-use-case-aware-default-model-ranking.md` §2 (the fetch beside a listing, OpenCode's default URL): superseded.
* `0013-MADR-remediate-debugging-pass-findings.md` A6, `0021-MADR-harden-and-tune-after-the-v1-1-review.md` C1 to C3,
  `0028-MADR-heuristics-and-performance-from-the-research-pass.md` D-A1, `0058-MADR-test-cache-isolation-for-reused-ports.md`
  D5, `0071-MADR-give-a-cache-a-lifetime-and-wait-for-a-refresh.md` D1 and D2,
  `0078-MADR-cached-metadata-refusal-test-uses-a-refused-option.md`: superseded as to the fetch.
* `0015-MADR-canonical-sdk-api-and-module-layout.md`: R17's common options, R30's example, and R42 and R48's exception (Q1).
* `0033-MADR-public-model-metadata-and-current-catalogs.md` D2 item 3 and D3: the caller persists and refreshes; the SDK
  parses and reads what it is given.
* `0041-MADR-decide-per-model-behaviour-from-metadata.md` D1, D3, D4, A3, D7.
* `0048-MADR-adopt-go-sdk-best-practices.md` D1 (R49): `catalog.Cache` holds no document.
* `0061-MADR-consumer-gaps-from-gobble-issue-9.md` E2 and its options amendment: `WithMetadata` joins the shared options.
* `0065-MADR-tests-reach-no-outside-host.md`: a test hands its document in.
* `0088-MADR-default-output-limit-from-model-facts.md` (F-line 72, ST8, ST9, ST91): "models.dev" is the handed document.
* `0089-MADR-model-output-limit-discovery.md` D1 to D3, D5 (the memo no longer carries the route fallback), DL23, DL24,
  DL29, DL50, DL51, DL70 (its annotation "One fetch, then ship (Recommended)": moot), DL71, DL74, DL77, its OQ2 ruling's
  "reads the cache and the embedded extract", and its claims rows C62now and C63now.
* gobble-cli `0072-MADR-keep-a-refreshed-models-dev-copy-on-disk.md` D17 (gobble's record): the caller of D1, at every
  listing, lookup, limits call and `provider.New`.

### What could not be established read-only

* Each fixture's provenance (F6), and whether models.opencode.ai's copy carries models.dev's licence.
* The full list of implicit extract dependents (MH55) and of live assertions that need a document (MH32): Phase 0's runs.
* gobble 0072's revised D17: the draft read here still says `provider.New` is handed none; the coordinator relays it as
  revised.

### Evidence index

Each row's text is on its line at the named revision (`tools/claims.py`).

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| C1 | the default document | `llmprovider/catalog/model_metadata.go:28` | `defaultModelMetadataURL = "https://models.opencode.ai/api.json"` |
| C2 | `LookupMetadata` | `llmprovider/catalog/model_metadata.go:270` | `func LookupMetadata(ctx context.Context, url string, client *http.Client) (Metadata, error) {` |
| C3 | `LookupMetadataWith` | `llmprovider/catalog/model_metadata.go:288` | `func LookupMetadataWith(ctx context.Context, id llmprovider.ProviderID, opts ...llmprovider.Option) (Metadata, error) {` |
| C4 | `CachedMetadataWith` | `llmprovider/catalog/model_metadata.go:308` | `func CachedMetadataWith(ctx context.Context, id llmprovider.ProviderID, opts ...llmprovider.Option) (Metadata, bool) {` |
| C5 | `OptionsFromEnv` | `llmprovider/catalog/model_metadata.go:437` | `func OptionsFromEnv() []llmprovider.Option {` |
| C6 | the fetch beside a listing | `llmprovider/catalog/discovery.go:343` | `meta = startModelMetadata(ctx, cfg)` |
| C7 | `WithModelMetadataURL` | `llmprovider/options.go:136` | `func WithModelMetadataURL(url string) Option {` |
| C8 | `WithoutModelMetadata` | `llmprovider/options.go:147` | `func WithoutModelMetadata() Option {` |
| C9 | `Settings.ModelMetadataURL` | `llmprovider/settings.go:379` | `func (st *Settings) ModelMetadataURL() string { return st.s.cfg.ModelMetadataURL }` |
| C10 | `Settings.ModelMetadataDisabled` | `llmprovider/settings.go:383` | `func (st *Settings) ModelMetadataDisabled() bool { return st.s.cfg.DisableModelMetadata }` |
| C11 | the extract's embed | `llmprovider/catalog/limits_reads.go:335` | `//go:embed limits_snapshot.json` |
| C12 | the extract in `Limits` | `llmprovider/catalog/limits.go:353` | `ext := bundledExtract()` |
| C13 | the extract in the route fallback | `llmprovider/catalog/limits.go:439` | `ext := bundledExtract()` |
| C14 | `LookupModel` fetches | `llmprovider/catalog/lookup.go:102` | `doc, err := loadModelMetadata(ctx, cfg)` |
| C15 | `CachedModel` starts a fetch | `llmprovider/catalog/lookup.go:137` | `if m, ok := cachedMetadata(ctx, cfg); ok {` |
| C16 | `CachedFacts` reads the cache | `llmprovider/catalog/lookup.go:175` | `if m, ok := peekMetadata(cfg); ok {` |
| C17 | `LookupLimits` fetches | `llmprovider/catalog/limits.go:176` | `if _, err := loadModelMetadata(ctx, cfg); err != nil {` |
| C18 | OpenCode's route reads the cache | `llmprovider/providers/opencode/opencode.go:722` | `meta, cached = catalog.CachedMetadataWith(ctx, p.gateway, p.listing...)` |
| C19 | and waits for a lookup | `llmprovider/providers/opencode/opencode.go:1003` | `return catalog.LookupMetadataWith(ctx, p.gateway, p.listing...)` |
| C20 | the route snapshot's embed | `llmprovider/internal/opencoderoute/opencoderoute.go:30` | `//go:embed routes_snapshot.json` |
| C21 | its source | `llmprovider/internal/opencoderoute/opencoderoute.go:27` | `"opencode-go" sections of models.opencode.ai/api.json (retrieved` |
| C22 | the catalogs command fetches | `llmprovider/internal/cmd/catalogs/main.go:105` | `return catalog.LookupMetadata(ctx, f.metadataURL, nil)` |
| C23 | `Metadata` is the models.dev layer | `llmprovider/catalog/metadata_document.go:70` | `func (m Metadata) Name() string { return sourceModelsDev }` |
| C24 | `WithModelLayer`, the shape followed | `llmprovider/catalog/lookup.go:34` | `func WithModelLayer(l Layer) llmprovider.Option {` |
| C25 | a "models.dev" layer is registry | `llmprovider/catalog/limits.go:336` | `if l.Name() == sourceModelsDev {` |
| C26 | `configFor` reads the off switch | `llmprovider/catalog/config.go:221` | `metadataOff:      st.ModelMetadataDisabled(),` |
| C27 | the wizard's advice | `wizard/configure.go:125` | `Pass catalog.OptionsFromEnv() here, or llmprovider.WithoutModelMetadata()` |
| C28 | the extract's generator | `tools/limitsnapshot.py:35` | `DEFAULT_OUT = ROOT / "llmprovider" / "catalog" / "limits_snapshot.json"` |
| C29 | `offline.py`'s program turns the fetch off | `tools/offline.py:93` | `llmprovider.WithoutModelMetadata(),` |
| C30 | `cachecensus.py`'s metadata anchor | `tools/cachecensus.py:50` | `c.metadataFetches[url] = fetch` |
| C31 | api-check fails any incompatible change | `scripts/check_api.py:110` | `return 1 if breaking else 0` |
| C32 | gobble turns the fetch off | `gobble-cli:llm/provider/provider.go:215` | `llmprovider.WithoutModelMetadata(),` |
| C33 | gobble's listing names a URL in tests | `gobble-cli:llm/provider/list.go:130` | `opts = append(opts, llmprovider.WithModelMetadataURL(o.ModelMetadataURL))` |
| C34 | the refusal's message | `llmprovider/internal/wire/limit_rule.go:80` | `"catalog.WithModelLayer, or pass WithMaxTokens or Request.MaxOutputTokens", llmprovider.ErrInvalidRequest,` |
| C35 | `localonly`'s advice | `internal/localonly/localonly.go:115` | `turns the models.dev document off when it serves none (0065-MADR)` |
| C36 | Kilo ranks from its own listing | `llmprovider/catalog/discovery.go:339` | `if id != llmprovider.ProviderKilo {` |
| C37 | the curation reads the document | `llmprovider/catalog/model_metadata.go:742` | `models, ok := res.doc.models(modelMetadataKey(provider))` |
| C38 | the document's evidence time | `llmprovider/catalog/limits.go:348` | `at := cfg.store.metadataFetchedAt(cfg)` |
| C39 | R42 | `docs/guides/api-standards.md:332` | `**R42. Deprecation is` |
| C40 | R48 | `docs/guides/api-standards.md:358` | `under the compatibility promise,** and` |
| C41 | R17's common options | `docs/guides/api-standards.md:214` | `reasoning, listing probes, the model metadata URL and turning its fetch` |
| C44 | `ReasoningEfforts` | `llmprovider/catalog/model_metadata.go:351` | `func (m Metadata) ReasoningEfforts(provider llmprovider.ProviderID, model string) []string {` |
| C45 | `InterleavedField` | `llmprovider/catalog/model_metadata.go:357` | `func (m Metadata) InterleavedField(provider llmprovider.ProviderID, model string) string {` |
| C46 | `NPM` | `llmprovider/catalog/model_metadata.go:363` | `func (m Metadata) NPM(provider llmprovider.ProviderID, model string) (string, bool) {` |
| C47 | Ollama lists before any fetch | `llmprovider/catalog/discovery.go:334` | `if id == llmprovider.ProviderOllama {` |
| C48 | canonical facts over every section | `llmprovider/catalog/canonical.go:106` | `func buildCanonical(m Metadata) map[string]ModelInfo {` |
| C49 | the first-party price fill | `llmprovider/catalog/metadata_document.go:89` | `func pricesByCanonical(doc *modelMetadataDoc, canonical string) (ModelRates, bool) {` |
| C50 | the canonical stage for any keyed provider | `llmprovider/catalog/limits.go:586` | `if info, ok := m.canonicalModel(model); ok && (cfg.compat` |
| C51 | Kilo reads its limits | `llmprovider/providers/kilo/kilo.go:545` | `limits := catalog.CachedLimits(ctx, llmprovider.ProviderKilo, model,` |
| C52 | the extract, parsed once | `llmprovider/catalog/limits_reads.go:378` | `var bundledExtract = sync.OnceValue(func() limitsExtract { return parseLimitsExtract(limitsSnapshot) })` |
| C53 | a golden holding the extract's value | `llmprovider/providers/claude/testdata/wire/claude/text.json:15` | `"max_tokens": 128000,` |
| C54 | G-wire turns the fetch off | `llmprovider/internal/wirecase/wirecase.go:398` | `extra := []llmprovider.Option{llmprovider.WithoutModelMetadata()}` |
| C55 | AGENTS' test rule | `AGENTS.md:239` | `or turns it off` |
| C56 | architecture's api-check | `docs/architecture.md:1574` | `fails on an incompatible change outside` |
| C57 | the changelog's promise | `CHANGELOG.md:6` | `a release that is not major only adds to the exported API, which` |
| C58 | the fallback goes to the memo | `llmprovider/catalog/limits.go:366` | `cfg.memo.SetFallback(memoKey(id, model, cfg, false), tokens, src)` |
| C59 | the memo's key | `llmprovider/internal/limitmemo/limitmemo.go:31` | `type Key struct {` |
| C60 | the memo's fallback note | `llmprovider/internal/limitmemo/limitmemo.go:170` | `func (m *Memo) SetFallback(k Key, tokens int, source string) {` |
| C61 | Claude reads it back | `llmprovider/providers/claude/limit.go:68` | `in.Fallback, in.FallbackSource = memo.Fallback(key)` |
| C62 | OpenCode reads it back | `llmprovider/providers/opencode/opencode.go:1073` | `in.Fallback, in.FallbackSource = p.limits.Fallback(p.memoKey(c, false))` |
| C63 | `LookupLimits` says it warms a request | `llmprovider/catalog/limits.go:140` | `LookupModel is, warms the caches a request reads, and returns what they` |

### Execution amendment (2026-11-24): the lines the docs phase wrote

0093-PLAN's Phase 5: one row for each edit its Phase 4 made, on a line that edit wrote; one for each changelog entry it added; and the two dated exception notes (MH57), the `NOTICE` and the licence (MH25) and `ExampleWithMetadata`. The ids end in "now", so that `tools/claims.py` does not read them against this record's `self=b2917f0`: each was checked with a plain id against a scratch clone's commit of the build's tree (`dc96bf0ce4e6`, uncommitted), 85 of 85, and planted false there with `tools/claims.py --plants`, 85 of 85 caught (MH40, MH45, MH57).

| Id | Claim | Where | Text |
| :--- | :--- | :--- | :--- |
| E1now | README.md: corrected by Phase 4 (D14, MH40) | `README.md:190` | `itself. Pass` |
| E2now | README.md: corrected by Phase 4 (D14, MH40) | `README.md:210` | `metadata" below), and a request reads only that document and what is` |
| E3now | README.md: corrected by Phase 4 (D14, MH40) | `README.md:277` | `wizard asks for a search; a blank search then offers the saved model, if` |
| E4now | README.md: corrected by Phase 4 (D14, MH40) | `README.md:292` | `[0093-MADR](docs/decisions/0093-MADR-take-model-metadata-from-the-caller.md)). Your` |
| E5now | CHANGELOG.md: the dated exception note (D10, MH57) | `CHANGELOG.md:12` | `2026-11-24: one recorded exception to that promise. v1.33.0 removes eight` |
| E6now | CHANGELOG.md: corrected by Phase 4 (D14, MH40) | `CHANGELOG.md:34` | `the models.dev document the call hands in with` |
| E7now | CHANGELOG.md: corrected by Phase 4 (D14, MH40) | `CHANGELOG.md:48` | `[0093-MADR](docs/decisions/0093-MADR-take-model-metadata-from-the-caller.md) D1, D2).` |
| E8now | CHANGELOG.md: corrected by Phase 4 (D14, MH40) | `CHANGELOG.md:84` | `the handed-in document states them, and where nothing states even that,` |
| E9now | CHANGELOG.md: corrected by Phase 4 (D14, MH40) | `CHANGELOG.md:153` | `- A request on a Messages route (Claude, OpenCode's messages route) whose` |
| E10now | AGENTS.md: corrected by Phase 4 (D14, MH40) | `AGENTS.md:238` | `and one that needs a models.dev document parses one and hands it in with` |
| E11now | docs/architecture.md: corrected by Phase 4 (D14, MH40) | `docs/architecture.md:180` | `, the models.dev document a caller hands in (0093-MADR D1),` |
| E12now | docs/architecture.md: corrected by Phase 4 (D14, MH40) | `docs/architecture.md:180` | `the models.dev document the call hands in (0093-MADR D2, D7),` |
| E13now | docs/architecture.md: corrected by Phase 4 (D14, MH40) | `docs/architecture.md:180` | `0093-MADR D7), the smallest limit a current model on the route is stated to take,` |
| E14now | docs/architecture.md: corrected by Phase 4 (D14, MH40) | `docs/architecture.md:197` | `names, required (0093-MADR D6), and each keyed listing,` |
| E15now | docs/architecture.md: corrected by Phase 4 (D14, MH40) | `docs/architecture.md:265` | `, with the models.dev document the call hands in with` |
| E16now | docs/architecture.md: corrected by Phase 4 (D14, MH40) | `docs/architecture.md:290` | `It filters, ranks and caveats nothing, reads no metadata document,` |
| E17now | docs/architecture.md: corrected by Phase 4 (D14, MH40) | `docs/architecture.md:298` | `Listing pages are read up to 8 MiB. The SDK fetches no metadata document` |
| E18now | docs/architecture.md: corrected by Phase 4 (D14, MH40) | `docs/architecture.md:311` | `1. **models.dev's document,** the one the call hands in with` |
| E19now | docs/architecture.md: corrected by Phase 4 (D14, MH40) | `docs/architecture.md:315` | `- A parsed document keeps each section's bytes.` |
| E20now | docs/architecture.md: corrected by Phase 4 (D14, MH40) | `docs/architecture.md:320` | `decoding every section now; it is the only way to make one.` |
| E21now | docs/architecture.md: corrected by Phase 4 (D14, MH40) | `docs/architecture.md:357` | `layers and its document without waiting,` |
| E22now | docs/architecture.md: corrected by Phase 4 (D14, MH40) | `docs/architecture.md:359` | `from the caches, the layers and the document given to the provider's` |
| E23now | docs/architecture.md: corrected by Phase 4 (D14, MH40) | `docs/architecture.md:370` | `These caches, and the probes', are the` |
| E24now | docs/architecture.md: corrected by Phase 4 (D14, MH40) | `docs/architecture.md:373` | `listings, probe results,` |
| E25now | docs/architecture.md: corrected by Phase 4 (D14, MH40) | `docs/architecture.md:378` | `fetches and probe runs that the cache would start,` |
| E26now | docs/architecture.md: corrected by Phase 4 (D14, MH40) | `docs/architecture.md:1134` | `the SDK fetches none and reads no variable for it (0093-MADR D6, D10).` |
| E27now | docs/architecture.md: corrected by Phase 4 (D14, MH40) | `docs/architecture.md:1157` | `auth.GrokFlowFromEnv` |
| E28now | docs/architecture.md: the dated exception note (D10, MH57) | `docs/architecture.md:1571` | `2026-11-24: one recorded exception. Against` |
| E29now | docs/guides/api-standards.md: corrected by Phase 4 (D14, MH40) | `docs/guides/api-standards.md:114` | `Without them, and with no document, the request decides (0093 D2, Q4).` |
| E30now | docs/guides/api-standards.md: corrected by Phase 4 (D14, MH40) | `docs/guides/api-standards.md:123` | `then the registry (a layer named "models.dev", then the document given` |
| E31now | docs/guides/api-standards.md: corrected by Phase 4 (D14, MH40) | `docs/guides/api-standards.md:128` | `Kilo cut one model at 128,000 of 131,072 stated), and none when nothing` |
| E32now | docs/guides/api-standards.md: corrected by Phase 4 (D14, MH40) | `docs/guides/api-standards.md:217` | `URL and turning its fetch off went with the fetch (0093 D10). A` |
| E33now | docs/guides/api-standards.md: corrected by Phase 4 (D14, MH40) | `docs/guides/api-standards.md:223` | `catalog.WithModelLayer` |
| E34now | docs/guides/api-standards.md: corrected by Phase 4 (D14, MH40) | `docs/guides/api-standards.md:288` | `environment calls an opt-in helper, such as` |
| E35now | docs/guides/api-standards.md: corrected by Phase 4 (D14, MH40) | `docs/guides/api-standards.md:336` | `until a major version. (0015 D13) *(2026-11-24: one recorded exception:` |
| E36now | docs/guides/api-standards.md: corrected by Phase 4 (D14, MH40) | `docs/guides/api-standards.md:365` | `tag fails an incompatible change. (0015 D13) *(2026-11-24: one recorded` |
| E37now | docs/guides/api-standards.md: corrected by Phase 4 (D14, MH40) | `docs/guides/api-standards.md:377` | `keep in memory between calls, the listings and their models' metadata,` |
| E38now | docs/guides/api-standards.md: corrected by Phase 4 (D14, MH40) | `docs/guides/api-standards.md:388` | `waits for it. It holds no models.dev document: a` |
| E39now | docs/guides/adding-a-provider.md: corrected by Phase 4 (D14, MH40) | `docs/guides/adding-a-provider.md:97` | `[0093-MADR](../decisions/0093-MADR-take-model-metadata-from-the-caller.md` |
| E40now | docs/guides/migrating-from-mcplib.md: corrected by Phase 4 (D14, MH40) | `docs/guides/migrating-from-mcplib.md:36` | `[0093-MADR](../decisions/0093-MADR-take-model-metadata-from-the-caller.md));` |
| E41now | docs/guides/migrating-from-mcplib.md: corrected by Phase 4 (D14, MH40) | `docs/guides/migrating-from-mcplib.md:295` | `The SDK fetches no document since v1.33.0: hand in a parsed one ([0093-MADR](../decisions/0093-MADR-take-model-metadata-from-the-caller.md) D10).` |
| E42now | docs/guides/migrating-from-mcplib.md: corrected by Phase 4 (D14, MH40) | `docs/guides/migrating-from-mcplib.md:397` | `and hand it in ([0093-MADR](../decisions/0093-MADR-take-model-metadata-from-the-caller.md) D10).` |
| E43now | tools/README.md: corrected by Phase 4 (D14, MH40) | `tools/README.md:29` | `of exactly the five listing goldens 0093-MADR D12 (e) regenerated (MH58); it prints each finding's path and values, and changes nothing` |
| E44now | tools/README.md: corrected by Phase 4 (D14, MH40) | `tools/README.md:32` | `, with no request, built for six platforms; two built-in plants must each fail it; a` |
| E45now | llmprovider/providers/claude/claude.go: corrected by Phase 4 (D14, MH40) | `llmprovider/providers/claude/claude.go:111` | `that ListModels or catalog.List read, and the models.dev document given` |
| E46now | llmprovider/providers/gemini/gemini.go: corrected by Phase 4 (D14, MH40) | `llmprovider/providers/gemini/gemini.go:74` | `that ListModels or catalog.List read, and the models.dev document given` |
| E47now | llmprovider/providers/grok/grok.go: corrected by Phase 4 (D14, MH40) | `llmprovider/providers/grok/grok.go:89` | `that ListModels or catalog.List read, and the models.dev document given` |
| E48now | llmprovider/providers/kilo/kilo.go: corrected by Phase 4 (D14, MH40) | `llmprovider/providers/kilo/kilo.go:86` | `that ListModels or catalog.List read, and the models.dev document given` |
| E49now | llmprovider/providers/ollama/ollama.go: corrected by Phase 4 (D14, MH40) | `llmprovider/providers/ollama/ollama.go:84` | `that ListModels or catalog.List read, and the models.dev document given` |
| E50now | llmprovider/providers/openai/openai.go: corrected by Phase 4 (D14, MH40) | `llmprovider/providers/openai/openai.go:107` | `that ListModels or catalog.List read, and the models.dev document given` |
| E51now | llmprovider/providers/together/together.go: corrected by Phase 4 (D14, MH40) | `llmprovider/providers/together/together.go:76` | `that ListModels or catalog.List read, and the models.dev document given` |
| E52now | llmprovider/providers/huggingface/huggingface.go: corrected by Phase 4 (D14, MH40) | `llmprovider/providers/huggingface/huggingface.go:79` | `// models.dev document given to New with catalog.WithMetadata (0093-MADR` |
| E53now | llmprovider/providers/huggingface/huggingface.go: corrected by Phase 4 (D14, MH40) | `llmprovider/providers/huggingface/huggingface.go:60` | `// ListModels returns the curated router listing, ranked by the models.dev` |
| E54now | llmprovider/providers/together/together.go: corrected by Phase 4 (D14, MH40) | `llmprovider/providers/together/together.go:56` | `// ListModels returns the curated chat models, ranked by the models.dev` |
| E55now | llmprovider/providers/opencode/opencode.go: corrected by Phase 4 (D14, MH40) | `llmprovider/providers/opencode/opencode.go:16` | `// models.dev document given to New with catalog.WithMetadata (0093-MADR D6),` |
| E56now | llmprovider/providers/opencode/opencode.go: corrected by Phase 4 (D14, MH40) | `llmprovider/providers/opencode/opencode.go:62` | `//     document is handed in (0062-MADR Q7 (a); it sent none, MADR 0009 §6).` |
| E57now | llmprovider/providers/opencode/opencode.go: corrected by Phase 4 (D14, MH40) | `llmprovider/providers/opencode/opencode.go:113` | `//     document given to New (0093-MADR D2), and never fetches.` |
| E58now | llmprovider/providers/chatcompat/chatcompat.go: corrected by Phase 4 (D14, MH40) | `llmprovider/providers/chatcompat/chatcompat.go:85` | `// given to New with catalog.WithMetadata (0093-MADR D2). The listing reads` |
| E59now | llmprovider/catalog/doc.go: corrected by Phase 4 (D14, MH40) | `llmprovider/catalog/doc.go:34` | `// caveat, static fallback or models.dev document, and nothing written to the` |
| E60now | llmprovider/catalog/doc.go: corrected by Phase 4 (D14, MH40) | `llmprovider/catalog/doc.go:41` | `// provider's requests read only the document given to its New. ModelInfo is one` |
| E61now | llmprovider/catalog/doc.go: corrected by Phase 4 (D14, MH40) | `llmprovider/catalog/doc.go:52` | `// backend's listing), then the registry (a layer named "models.dev", then` |
| E62now | llmprovider/catalog/doc.go: corrected by Phase 4 (D14, MH40) | `llmprovider/catalog/doc.go:56` | `// and OpenCode's messages route (0089-MADR D1 to D3); Limits.RouteFallback` |
| E63now | llmprovider/catalog/raw.go: corrected by Phase 4 (D14, MH40) | `llmprovider/catalog/raw.go:66` | `// static fallback; no models.dev document read; no listing cache read or write, so` |
| E64now | llmprovider/catalog/discovery.go: corrected by Phase 4 (D14, MH40) | `llmprovider/catalog/discovery.go:116` | `// modelListingTimeout bounds one model listing (MADR 0009 §2).` |
| E65now | llmprovider/catalog/cache.go: corrected by Phase 4 (D14, MH40) | `llmprovider/catalog/cache.go:67` | `// store is one cache's listings and their reads, each under its own lock.` |
| E66now | llmprovider/catalog/metadata_document.go: corrected by Phase 4 (D14, MH40) | `llmprovider/catalog/metadata_document.go:20` | `// Encode, and makes no request (0033-MADR D3). The document is the caller's:` |
| E67now | llmprovider/llmtest/llmtest.go: corrected by Phase 4 (D14, MH40) | `llmprovider/llmtest/llmtest.go:1011` | `// (R44, R40; 0026-MADR F57). The SDK fetches no metadata document` |
| E68now | internal/localonly/localonly.go: corrected by Phase 4 (D14, MH40) | `internal/localonly/localonly.go:95` | `// count: "example.com:443 (3)".` |
| E69now | internal/localonly/localonly.go: corrected by Phase 4 (D14, MH40) | `internal/localonly/localonly.go:115` | `"; a test serves its own replies, and hands in a models.dev document with catalog.WithMetadata when it " +` |
| E70now | wizard/configure.go: corrected by Phase 4 (D14, MH40) | `wizard/configure.go:126` | `// listings by a models.dev document; the wizard fetches none (0093-MADR` |
| E71now | llmprovider/catalog/listing_cache.go: corrected by Phase 4 (D14, MH40) | `llmprovider/catalog/listing_cache.go:16` | `// a refresh in the background past them. It is keyed as the probe cache is: by provider,` |
| E72now | NOTICE credits models.dev under MIT (D11, MH25); reworded 2026-11-24 (the review's notes closed, note 7): the data originates from models.dev, and the licence of models.opencode.ai's copy is not asserted (it stays unestablished, "What could not be established") | `NOTICE:11` | `the MIT License:` |
| E73now | LICENSE.models.dev holds models.dev's MIT text (D11, MH25) | `LICENSE.models.dev:1` | `MIT License` |
| E74now | ExampleWithMetadata's output (R50) | `llmprovider/catalog/example_metadata_0093_test.go:31` | `// Output: 200000 models.dev` |
| E75now | CHANGELOG Added: catalog.WithMetadata (D1, D2) | `CHANGELOG.md:44` | `: a shared option that hands in a models.dev` |
| E76now | CHANGELOG Added: NOTICE and LICENSE.models.dev (D11) | `CHANGELOG.md:50` | `(OpenCode's route snapshot and the test fixtures), credited under` |
| E77now | CHANGELOG Added: Limits.RouteFallback (D7) | `CHANGELOG.md:41` | `fallback, the smallest limit a current model on the route is stated to` |
| E78now | CHANGELOG Changed: only a handed-in document is read (D2 to D5, Q4) | `CHANGELOG.md:141` | `- Model metadata comes only from the caller. A listing's ranking and its` |
| E79now | CHANGELOG Changed: OpenCode's route never waits (D6) | `CHANGELOG.md:150` | `- OpenCode's request route is the pinned route, else the handed-in` |
| E80now | CHANGELOG Removed: the eight names (D10, Q1) | `CHANGELOG.md:169` | `the exception at the top of this file says: the SDK no longer downloads` |
| E81now | CHANGELOG Removed: the download, its cache and the two variables (D6, D10) | `CHANGELOG.md:175` | `- The download of` |
| E82now | CHANGELOG Changed: the cold-call refusal, against v1.32.0's `max_tokens` 8192 (D8, D9, Q2); re-based 2026-11-24 (the review's notes closed): this row held the Removed bullet for the extract, which is dropped, since the extract was never released | `CHANGELOG.md:158` | `where v1.32.0 sent 8192. The refusal names` |
| E83now | README: the model metadata section (D14, MH45) | `README.md:289` | `## Model metadata` |
| E84now | README: the SDK fetches and ships none (D6, D7) | `README.md:291` | `The SDK fetches no model metadata and ships none` |
| E85now | README: what happens without a document (MH4, MH18, MH51) | `README.md:312` | `models.dev facts. A Claude request, or one on OpenCode's messages route,` |
