---
status: in-progress
date: 2026-11-22
associated-madr: "0089-MADR-model-output-limit-discovery.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Model Output-Limit Discovery, and Send a Limit Only Where It Is Needed

Associated MADR: [0089-MADR-model-output-limit-discovery.md](0089-MADR-model-output-limit-discovery.md)

*(2026-11-22, second draft, after record review 3: the smaller API, `LookupLimits`, the embedded extract, the stored one-model and server reads, the validator's model match, the forms by wire, the expiring pass-back, the tests review 3 found unable to pin their rows, the Kilo measurement's verdict and bounds, and the prerequisite branched on 0088's OQ18.)*

## Goal

After this PLAN, a caller can ask the SDK what a model's output limit, window
and input limit are, with every source's value and time (`catalog.LookupLimits`,
`catalog.CachedLimits`); a validated refusal bounds a later request, kept for
one process and passed back only with an expiry; every response says what
output limit it carried and why (`Response.OutputLimit`); and a request that
names no limit carries what the owner's answers to OQ1 to OQ4 decide.

The MADR is `accepted` (2026-11-22): the owner answered OQ1 to OQ4 and 0088's
OQ18, and every row is decided or out of scope but DL65, which awaits this
PLAN's live measurement. This PLAN stays `proposed`: nothing below Phase 0
starts before the owner approves it and says `proceed`, `execute the plan` or
`do phase N`. *(Annotated 2026-11-22: the owner approved this PLAN and the
start; it is `in-progress`. See the execution record's entry of that date.)*

## Prerequisites

*(2026-11-22, the owner's answers: OQ18 "Five now; 0088+0089 next
(Recommended)". The next tag carries 0081, 0083, 0084, 0085 and 0087; 0087's
PLAN finishes first. This PLAN and 0088's are built after that tag, together:
this PLAN's Phases 1 and 2, then the live measurement, then Phase 3 with
0088's Phase 1 in one commit, then 0088's Phase 1c; Phases 4 to 7 run once
for both. The "Tests by answer" built are OQ1 (1), OQ2 (1), OQ3 (1) and OQ4
(1); the rest are not built.)*

* The owner's answers to this MADR's OQ1 to OQ4 and to 0088-MADR's OQ18,
  recorded as rows.
* Under 0088's OQ18 (1) or (2): 0088's Phase 1 is built with this PLAN's
  Phase 3 in one commit, and its Phase 1c after it. Under OQ18 (3): 0088 is
  built and released first on its 2026-11-21 rule; this PLAN's Phase 3 then
  changes 0088's Phase 1 expectations as planned, not as a deviation.
* Under OQ1 (1), one live measurement before Phase 3 ("OQ1 (1)'s
  measurement"); otherwise no live run: every fact the tests assert is served
  from a committed fixture or a test server.

## Scope

### In scope

* `llmprovider/catalog`: `Limits`, `Fact`, `Evidence`, `LookupLimits`,
  `CachedLimits`; the per-source reads (each layer, the listing, the document,
  the extract, kept apart, classified by source name); the new reads (Together
  `config.max_output_length`, Hugging Face per-provider `context_length`, the
  ChatGPT backend's `context_window`, Kilo `per_request_limits`, Claude's
  one-model read, Ollama's `/api/ps`), stored in their own map; the staged
  registry match; the embedded extract `limits_snapshot.json`.
* *(2026-11-22, deviation 2, the owner's ruling "Share the route table (Recommended)":)*
  `llmprovider/internal/opencoderoute` (new): OpenCode's route snapshot and
  table, moved from `providers/opencode`, read by `catalog` (for
  `Limits.Required`, DL74) and by `providers/opencode`, whose behaviour and
  existing tests stay unchanged. *(2026-11-23, deviation 4, the owner's
  ruling "Move it; fix the one path (Recommended)": `routes_snapshot.json` itself moves to
  `llmprovider/internal/opencoderoute`; `providers/opencode` keeps thin
  wrappers under today's names and types; the one existing test edited is the
  path at `llmprovider/live_0031_measure_test.go:108`.)*
* `tools/limitsnapshot.py` and its test in `tools/test_tools.py`: from a
  models.dev document given by path, it writes the extract, keeping only the
  `limit` fields of the sections the SDK reads.
* `llmprovider`: `SentLimit`, `LimitBasis` and its six constants,
  `Response.OutputLimit`, `LimitObservation`, `WithLimitObservations`,
  `WarningLimitLearned`, `Warning.Observed`; OQ3's option; under OQ2 (3)
  `WarningOutputLimitUnknown`; under OQ4 (2) or (3) `Request.OutputCeiling`.
* `llmprovider/internal/wire`: the refusal forms by wire and the validator
  (MADR D4, DL68); the memo (D5) beside 0062's; the request rule (D6), one
  helper every provider calls, replacing 0088's planned `DefaultLimit`.
* Every provider's body function, constructor and retry check (0088's list).
* Docs: `docs/guides/api-standards.md` R57, R6 and R7, and its Checks row;
  `docs/architecture.md`; `README.md`; `docs/guides/adding-a-provider.md`; the
  `catalog` and provider package docs (their output-limit lines and the
  reasoning-accounting table); `CHANGELOG.md`; under OQ1 (1), AGENTS.md "Live
  tests" names `LLMPROVIDER_LIVE_0089`.

### Exported names (R55), in `apidiff`'s form, each with its call site

`apidiff` reports a new type once and lists none of its fields (0044-MADR A1);
a field added to an existing type is expected as its own line, which Phase 4
confirms. The call sites are gobble-cli's
`docs/decisions/0066-MADR-set-each-session-model-up-from-discovered-limits.md`
unless named otherwise:

| `apidiff` line | Call site |
| :--- | :--- |
| `./llmprovider/catalog.LookupLimits: added` | session start, model switch, the title model (0066 D1, D2) |
| `./llmprovider/catalog.CachedLimits: added` | `gobble models`, the ACP model option, compaction's window (0066 D5, D6, D9); every provider's request (in-module) |
| `./llmprovider/catalog.Limits: added`, `.Fact: added`, `.Evidence: added` | `llm.Model.Limits` and its sources (0066 D4) |
| `./llmprovider.SentLimit: added`, `.LimitBasis: added`, and `.LimitBasisRequest`, `.LimitBasisOption`, `.LimitBasisFact`, `.LimitBasisRefusal`, `.LimitBasisFallback`, `.LimitBasisNone: added` | the debug log and the session report (0066 D8), the fallback notice (0066 D10) |
| `./llmprovider.Response.OutputLimit: added` | as `SentLimit` |
| `./llmprovider.LimitObservation: added`, `.WithLimitObservations: added`, `.WarningLimitLearned: added`, `.Warning.Observed: added` | learned caps saved and handed back (0066 D7) |
| OQ3's `WithLimitLearning` or `WithoutLimitLearning` | gobble opts in or not (0066 D7) |
| `./llmprovider.WarningOutputLimitUnknown: added` (OQ2 (3) only) | the notice (0066 D10) |
| `./llmprovider.Request.OutputCeiling: added` (OQ4 (2) or (3) only) | gobble's `outputLimit` setting (0066 S32) |

Eighteen lines, plus OQ3's option, plus up to two by OQ2 and OQ4.

### Out of scope

* New chatcompat dialects (MADR DL14, DL61); token counting (DL62); headers
  (DL63); probing (DL64).
* gobble-cli's code (its 0066 pair).
* 0088's cut answer (its Phase 1b), Generate over the stream (its OQ15) and
  the estimate (its Phase 1c): its own PLAN's.

## Rules for Every Phase

* **Re-ground first.** Each phase starts by running
  `python3 -B tools/claims.py docs/decisions/0089-MADR-model-output-limit-discovery.md --rev self=<start>`
  at the phase's start commit and annotating each moved row, dated, with its
  new line. A row whose text is gone is a stop.
* **Red, then green.** Each test marked red runs at the phase's base with
  `python3 -B tools/redgreen.py SPEC.json --expect fail --rev <base>` and must
  fail on its assertion; one that does not compile at the base goes in the
  spec's `drop` list, and its plant is its red. After the change, every test
  `--expect pass`.
* **Plants.** Each plant is the one production change named, made on a
  scratch clone by `python3 -B tools/plant.py PLANTS.json`, never in the tree;
  each must turn its test red; the execution record quotes each failing line.
* **Heavy runs.** `python3 -B tools/gate.py`, `make pre-add-check`, and any
  `-race` run go through
  `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>`, or to
  another host through `run-on-host` (with `--race` where the command runs
  `-race`), one at a time per host. A single package's `go test` without
  `-race` may stay on the session host.
* **No live run** but the one named in Prerequisites. Every other test serves
  its own replies and listings from the committed fixtures
  (`firstparty-2026-10-14`, `ranking-2026-10-12`, `rawlistings-2026-10-15`)
  and its own `httptest` servers; the localonly guard stays on.
* **Stop conditions**, each a stop and a prompt to the owner with evidence and
  resolutions: an existing test, golden or fixture needs a change no row
  names; `apidiff` reports a line outside the list above; a red test cannot be
  red at the base without a production change; a row cannot be met as
  written; a third build pass of one phase.
* **Exit criterion of every phase:** no blocker, every row tested, plants
  caught.

## Implementation Steps

### Phase 0: records

1. **Re-ground** at HEAD when the owner takes up the drafts.
2. Record the owner's answers to OQ1 to OQ4 as rows; each open row rewritten
   `decided` with its test and plant from "Tests by answer"; 0088's rows they
   decide (ST74, ST85, ST86, ST89, ST90, ST92, ST98) get dated decided rows in
   0088's MADR in the same change.
3. The two index rows in `docs/README.md`, by `tools/records.py add`.
4. The MADR goes to `accepted` on the owner's word; this PLAN to
   `in-progress` when approved.

Exit criterion: no blocker, every row tested, plants caught (here: every
decided row names its test and plant, and none is open).

### Phase 1: discovery

1. **Re-ground**, and read at the base: `lookup.go`, `listing_cache.go` (the
   key, `storeListing`, `cachedListing`, `claimListing`, last good),
   `listing_metadata.go`, `discovery.go`, `model_metadata.go`, `canonical.go`,
   `dialect.go`, the Ollama readers, `providers/opencode/route.go` (the embed
   precedent), and the fixtures named in the rules.
2. **Tests first**, in `llmprovider/catalog/limits_0089_test.go` (package
   `catalog`), `llmprovider/catalog/limits_example_0089_test.go` (package
   `catalog_test`, R50) and `tools/test_tools.py`:

   | Test | Rows | What it does | At the base | Plant, and why it turns red |
   | :--- | :--- | :--- | :--- | :--- |
   | `TestDiscover_ListingFacts_0089` | DL1, DL3, DL5 | Claude, Gemini and Kilo fixture listings: `claude-opus-4-6` output 128,000 from `listing:claude`, input 1,000,000, window unknown; a Gemini entry's two limits; `kilo-auto/balanced` 65,536 and window 1,000,000; a null Kilo cap: output unknown | does not compile | (a) `max_input_tokens` read as the window; (b) the null read as 0 and known |
   | `TestDiscover_ClaudeAlias_0089` | DL2 | listing without `claude-sonnet-4-5`; `GET /v1/models/claude-sonnet-4-5` answers the dated 4.5 entry (64,000): output 64,000, the resolved id kept; a 404: unknown | does not compile | the one-model read skipped |
   | `TestDiscover_ReadsSurviveRefresh_0089` | DL66 | an alias read and an `/api/ps` read stored; the listing refreshed after its TTL: both still read; a 404 read once, then a second `LookupLimits` within 10 minutes: no second request | does not compile | (a) stored in the listing's models: gone after the refresh; (b) the 404 not kept: a second request |
   | `TestDiscover_UnreadFields_0089` | DL4, DL8, DL9, DL10 | Kilo 65,536 and `per_request_limits` 32,768: 32,768; Together 40,000; Hugging Face windows 1,000,000 and 262,144: 262,144, and `:novita`: 1,000,000; ChatGPT `context_window` 272,000 | does not compile | (a) the larger Kilo value; (b) Together's field unread; (c) the largest HF window; (d) the ChatGPT field unread |
   | `TestDiscover_OllamaServed_0089` | DL12, DL13 | `/api/show` 131,072, `/api/ps` 8,192 for a loaded model: 8,192 from `server:ollama`, conflicting; a model not loaded: 131,072; the same through the dialect | does not compile | (a) trained first; (b) `/api/ps` unread |
   | `TestDiscover_EvidenceKinds_0089` | DL15, DL16, DL29 | a `catalog.Metadata` layer stating 64,000 and Claude's listing 128,000: 128,000 chosen, the layer's value kept; a "models.json" layer stating 20,000: 20,000 chosen; `WithoutModelMetadata`: the extract and the layer still present | does not compile | (a) the "models.dev" layer ranked as caller; (b) "models.json" ranked as registry; (c) the extract dropped with the document |
   | `TestDiscover_EmbeddedExtract_0089` | DL70, DL24 | the extract states 64,000, a served document 32,000: 32,000 from `models.dev`, the extract's kept, `Stale`; the document server closed: 64,000 from `models.dev (bundled <date>)` | does not compile | the extract ranked above the document |
   | `test_limitsnapshot_keeps_limits_only` (`tools/test_tools.py`) | DL70 | a document with costs, modalities and limits: the output has only `limit` per model | absent | a non-limit field kept: the test fails |
   | `TestDiscover_MatchStages_0089` | DL71 | an exact entry 128,000 and a family entry 64,000 for `claude-opus-4-6-20260205`: the exact one when present; absent, the family one under `models.dev (family)`; a canonical id match under `models.dev (canonical id)` | does not compile | (a) the family match chosen over the exact; (b) a family match named as exact |
   | `TestDiscover_Conflict_0089` | DL17, DL18 | listing 64,000 and document 128,000: conflicting, 64,000; two caller layers, 20,000 then 16,000: 16,000 | does not compile | (a) the lower rank chosen; (b) the earlier layer chosen |
   | `TestDiscover_OutputAboveWindow_0089` | DL19 | output 16,384, window 8,192: 8,192, conflicting | does not compile | the hold removed |
   | `TestDiscover_StatedZero_0089` | DL20 | a layer stating 0, marked known: unknown, the 0 kept | does not compile | 0 known |
   | `TestDiscover_Unknown_0089` | DL21 | no source: unknown, nil error (the extract stating nothing for the id) | does not compile | an error returned |
   | `TestDiscover_SourceFailure_0089` | DL22 | listing 503: the document's facts, the error wrapping the listing's `*APIError`; 401: `ErrAuthFailure` | does not compile | the error dropped |
   | `TestDiscover_Stale_0089` | DL23 | facts read, the clock moved past 10 minutes: still returned, `Stale` true, `Seen` unchanged | does not compile | `Stale` left false |
   | `TestDiscover_Offline_0089` | DL24 | every server answers an empty models.dev document or closes; a caller layer ~~and a passed-back cap~~: ~~those two~~ the layer's and the extract's facts; `Evidence` holds no entry from the empty document *(2026-11-22, deviation 1, "Split the test across phases (Recommended)": the passed-back cap's case is added to this test in Phase 2)* | does not compile | an empty document read as a fact |
   | `TestDiscover_Closed_0089` | DL25 | `Close`, then `LookupLimits`: the closed-cache error; the listing, document, one-model and `/api/ps` servers each count no request | does not compile | the check removed: the one-model or `/api/ps` server counts one |
   | `TestDiscover_Cancelled_0089` | DL26 | ctx cancelled while the listing stalls: ctx's error, the layers' facts | does not compile | the error swallowed |
   | `TestDiscover_Concurrent_0089` | DL27 | under `-race`, 16 goroutines `LookupLimits` on one cache for one model: the listing server counts one request; after all return, `CachedLimits` reads 128,000 | does not compile | a fetch per call |
   | `TestDiscover_InvalidProvider_0089` | DL28 | id `nope`: `ErrInvalidProvider` | does not compile | nil error |
   | `TestDiscover_WireFacts_0089` | DL6, DL7, DL11, DL30 | `Required` on Claude and OpenCode's messages route, and on no other provider or route | does not compile | (a) `Required` on Kilo; (b) the messages route not required |
   | `TestDiscover_RequestReadsDiscovery_0089` *(2026-11-22, deviation 3: in `llmprovider/providers/claude`, package `claude`, since a `catalog` test cannot build a Claude provider, which imports `catalog`)* | DL72 | Claude with a workspace: `LookupLimits`, then a request through a provider built with the same options and key: the request's facts are discovery's, no listing request during it | does not compile | the stored key without the scope: the request misses *(2026-11-23, deviation 5: it also pins DL75, `LookupLimits` given `catalog.WithClaudeWorkspace`, as the provider's listing is)* |
   | `TestDiscover_RequiredWithoutDocument_0089` *(2026-11-22, deviation 2)* | DL74 | OpenCode Zen, `WithoutModelMetadata`: a Claude model on the messages route reports `Required` true; a chat-route model false | does not compile | `Required` read from the models.dev document only: the Zen Claude model reports false |
   | `ExampleLookupLimits` | R50 | offline, `httptest`: prints the output fact and its source | does not compile | the `// Output:` line changed |

3. **Red run**: every test dropped (none compiles); plants are their red.
4. **The code**, then **green and plants**; `go test ./llmprovider/catalog/...`;
   the `-race` test through `run-heavy`; **checks**: `make pre-add-check
   FILES="..."` through `run-heavy`, `make lint`, `make docs-check`, `make
   tools-test`; the **execution record** (pass number, red and green output,
   each plant's failing line, each check's exit status taken before any
   filter, anything not done).

Exit criterion: no blocker, every row tested, plants caught.

### Phase 2: refusals as evidence, the memo and the pass-back

1. **Re-ground** at Phase 1's commit; read `context_overflow.go`,
   `api_error.go`'s classifiers, `internal/wire/learn.go`, `retry.go`,
   `sse.go`, `refusal.go`, `warning.go`, the Messages matcher, `AddThinkingFor`.
2. **Tests first**, in `llmprovider/internal/wire/limit_refusal_0089_test.go`,
   `llmprovider/limit_observations_0089_test.go` (package `llmprovider_test`)
   and the providers' `limit_refusal_0089_test.go`. The refusals' texts are the
   forum copies W7 names; none is a recorded reply.

   | Test | Rows | What it does | At the base | Plant, and why it turns red |
   | :--- | :--- | :--- | :--- | :--- |
   | `TestLimitRefusal_Validator_0089` | DL31 to DL34, DL67 | (1) Claude, wire value 128,000, refused `max_tokens: 128000 > 64000, which is the maximum allowed number of output tokens for claude-opus-4-5-20251101`: a bound of 64,000; (2) that text with OpenRouter's window form: rejected; (3) `100000 > 64000` when 128,000 was on the wire: rejected; (4) `64000 > 64000`: rejected; (5) Kilo (Chat Completions), OpenAI's form stating at most 8 with a 16 minimum test wire: rejected below the minimum; (6) OpenCode google, Gemini's range `65537 (exclusive)`: 65,536; (7) the budget shape raised the wire value to 44,096: a refusal stating `44096 > 32000` accepted, one stating the pre-raise value rejected | does not compile | (a) the form not recognised; (b) the double match read as a window; (c) N not compared; (d) M ≥ N accepted; (e) the minimum not applied; (f) N compared with the pre-raise value |
   | `TestLimitRefusal_FormsByWire_0089` | DL68 | OpenAI's "supports at most" text on OpenAI's Responses provider: not read; on Kilo: read; Gemini's range on Gemini's Interactions provider: not read | does not compile | OpenAI's form read on Responses |
   | `TestLimitRefusal_WindowForms_0089` | DL35 | Kilo, wire 16,384, C39's form with window 32,768 and about 40,000: the resend's room 9,152; Kilo's documented form; Together's split; after each, `CachedLimits` shows no window evidence and the memo is empty | does not compile | the window kept in the memo |
   | `TestLimitRefusal_NamedModel_0089` | DL36 | Claude alias `claude-sonnet-4-5`, refused naming `claude-sonnet-4-5-20250929` (the resolved id): kept; Kilo `anthropic/claude-sonnet-4.5`, refused naming `claude-sonnet-4-5`: kept (canonical tokens); Kilo `kilo-auto/balanced`, refused naming a Claude id: this request only, no memo entry | does not compile | (a) an exact-only compare: the alias case not kept; (b) any name kept: the routed case kept |
   | `TestLimitRefusal_CallerLimit_0089` | DL37 | Claude, `Request.MaxOutputTokens` 200,000, refused with a cap of 128,000: one send, `ErrInvalidRequest`; `CachedLimits` then shows the bound (under the OQ3 answer that keeps it) | does not compile | resent at 128,000 |
   | `TestLimitMemo_Scope_0089` | DL38 | two caches; two base URLs; two Kilo organizations; a streamed and a plain Gemini request: each apart | does not compile | (a) one memo for every cache; (b) stream and plain merged |
   | `TestLimitMemo_Bound_0089` | DL38 | 257 models kept: the first dropped | does not compile | no bound |
   | `TestLimitMemo_SelfHeal_0089` | DL39 | a kept cap of 4,096, then a response for its key reporting 10,000 output tokens: dropped | does not compile | kept |
   | `TestLimitMemo_Concurrent_0089` | DL43 | under `-race`, 32 requests on one Claude provider, half refused with the cap form: each refused request resent once; one memo entry | does not compile | the lock removed: the race detector reports it |
   | `TestLimitObservations_PassBack_0089` | DL40, DL41 | (1) OpenCode messages, a model no source states, a passed-back cap of 30,000 seen an hour ago: used, `remembered`; seen 25 hours ago: unused; for another Kilo organization: unused; (2) a registry fact of 64,000 and a passed-back 30,000: 64,000 chosen, 30,000 kept as evidence | does not compile | (a) no expiry; (b) the scope ignored; (c) the remembered value chosen over the registry |
   | `TestLimitObservations_Invalid_0089` | DL42 | no model; Tokens 0; zero `Seen`: `New` refuses each with `ErrInvalidRequest` | does not compile | accepted |
   | `TestDiscover_Offline_0089`, its passed-back case *(2026-11-22, deviation 1, "Split the test across phases (Recommended)")* | DL24 | offline, cold cache, a caller layer and a passed-back cap: those two and the extract's facts | red: the case asserts the passed-back evidence | the passed-back cap dropped offline |

3. **The code**, then green, plants, checks and the execution record as Phase
   1's step 4.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 3: the request's limit and its report

With 0088's Phase 1 in one commit under 0088's OQ18 (1) or (2); after 0088's
release under its OQ18 (3).

1. **Re-ground** at Phase 2's commit and at 0088's PLAN as amended. Confirm the
   answers to OQ1 to OQ4 and 0088's OQ18 are rows, and, under OQ1 (1), the
   measurement's verdict is recorded (DL65).
2. **Tests first**, in `llmprovider/providers/output_limit_0089_test.go` over
   `TestOutputLimit_Facts_0041`'s harness, by `Generate` and `Stream`, and the
   answers' tests:

   | Test | Rows | What it does | At the base | Plant |
   | :--- | :--- | :--- | :--- | :--- |
   | `TestOutputLimit_Reported_0089` | DL44, DL53, DL54 | `MaxOutputTokens` 5,000: {5,000, sent, request}; `WithMaxTokens(4000)`: option; `WithMaxTokens(200000)` with a stated 128,000: 128,000, option; a probe: option, 8192; the stream's final response carries it | does not compile | (a) basis fact for a caller's value; (b) zero on a stream |
   | `TestOutputLimit_BudgetReported_0089` | DL55 | Claude, the budget shape, budget 40,000 over a chosen 32,000: `OutputLimit.Tokens` 44,096, the wire's value | does not compile | the pre-raise value reported |
   | `TestOutputLimit_OutsideProvider_0089` | DL57 | `CachedLimits` for a chatcompat id built with `chatcompat.WithID` and a caller layer: the layer's fact | does not compile | the chatcompat id refused: unknown |
   | `TestOutputLimit_Default_0089` | DL45, and the OQ1 rows by its answer | the ChatGPT backend and plan: no field, basis none; the rest as "Tests by answer" | does not compile | a field on the plan route |

3. **The code**: the helper (MADR D6) in `internal/wire`, called by every body
   function; the resend's value from the validated refusal; the report.
4. Green, plants, checks and the execution record as Phase 1's step 4.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 4: exports, docs, standard and changelog

1. **Re-ground** at Phase 3's commit.
2. Checks, each with its plant:

   | Check | Rows | What it does | Plant |
   | :--- | :--- | :--- | :--- |
   | `make api-check`; `python3 -B tools/apidiff.py <newest tag>` | DL58 | lists exactly the lines of "Exported names" for the answers given | one more exported name on a scratch clone: reported |
   | `tools/claims.py` rows on R57, R6's and R7's new sentences, each package doc's output-limit line and each changelog entry | DL59, DL60 | each says what the code does | each planted false by `tools/claims.py --plants`: caught |
   | `make docs-check changelog-check records-check links-check tables-check` | DL59, DL60 | the docs and the variable table (under OQ1 (1), `LLMPROVIDER_LIVE_0089`) | the check's own plant where its PLAN names one |

3. The docs and the changelog (**Added**, **Changed**), R57 with its Checks row
   (`git grep -n` in non-test code finds no number 0088's "Every number" table
   retires but those this MADR's "Numbers" table keeps).
4. The execution record.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 5: planted breaks and the gates

1. **Re-ground** at Phase 4's commit.
2. Every plant of Phases 1 to 4 again with `tools/plant.py`, on one scratch
   clone, each caught, each failing line quoted.
3. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=1 ./llmprovider/...`
4. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B tools/gate.py --timeout 5400`
5. CI's set: `make lint vuln parity-check dep-check coverage-check api-check
   contract-check generate-check records-check tables-check fixtures-check
   changelog-check docs-check markdownlint links-check disclosure-check
   gate-selftest tools-test`; `go vet -tags live_gateways ./...`.
6. `python3 -B tools/claims.py` on the MADR; `python3 -B tools/recordlint.py
   --changed`.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 6: an independent review

1. **Re-ground** at Phase 5's commit.
2. A reviewer who did not build the phases reads the MADR, this PLAN and the
   diff from Phase 1's base, not the builder's summary, and reports by row.
3. A finding that changes or adds a row goes to the owner in one prompt,
   three options each, the recommended first; the ruling is a row in the MADR
   and a dated deviation here, and the phase it touches runs again as a new,
   numbered pass.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 7: the release (the owner's word)

The release 0088's OQ18 answer names. No step runs under a standing approval:
the tag and the GitHub Release each wait for the owner's word in the same
turn. Before the tag, `tools/limitsnapshot.py` regenerates the extract from a
public models.dev document, and the changed extract is committed with its
date. 0088's PLAN Phase 4 steps apply, with this record's lines added to the
`apidiff` expectation; `tools/records.py status` sets this PLAN `complete`
once the tag's CI and `tools/consumer.py` and `tools/offline.py` are recorded.

Exit criterion: no blocker, every row tested, plants caught.

## Tests by answer (only the chosen option's are built)

| Answer | Test | Rows | What it does | At the base | Plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| OQ1 (1) | `TestOutputLimit_Omitted_0089` | DL46 to DL48, DL69 | no caller limit: Claude, listing 128,000: 128,000, basis fact; OpenCode messages, a registry 64,000: 64,000; Gemini, Grok, OpenAI with a key, Together, Hugging Face, Ollama, chatcompat (a caller layer stating a limit included) and OpenCode's chat, responses and google routes: no field, basis none; Kilo as DL65's verdict says; `WithMaxTokens(5000)` on an optional wire: 5000 | red: 8192 everywhere | (a) a fact sent on Gemini; (b) Claude omitted; (c) a caller layer sent on Ollama; (d) the caller's value dropped on an optional wire |
| OQ1 (1) | `TestLive_OmittedLimit_0089` (live; "OQ1 (1)'s measurement") | DL65 | as that section says | not run offline | |
| OQ1 (2) | `TestOutputLimit_Default_0089`, per provider | DL46 to DL48 | Claude 128,000; Kilo 65,536 (window 1,000,000); OpenCode chat, a registry 20,000: 20,000; Kilo, nothing known: no field; Grok and OpenAI, a registry 128,000 only: no field, and with a "models.json" layer 20,000: 20,000; Ollama, registry only: no field; an output above the window: the window | red: 8192 everywhere | (a) a 32,000 ceiling kept; (b) the registry accepted on OpenAI; (c) 8192 sent on Kilo when unknown; (d) the window hold removed |
| OQ1 (3) | 0088's Phase 1 table as written | DL46 to DL48 | | | |
| OQ2 (1) | `TestOutputLimit_Fallback_0089` | DL50, DL51 | a fresh cache, Claude: the first request reads the extract (no listing request during it); a model in no listing and no extract entry, on a route whose current models state limits: 64,000 (the smallest current `max_tokens` in the fixture listing), basis fallback; then refused stating 4,096: resent at 4,096; a numberless refusal: returned. *(2026-11-22, DL51 "Refuse, say how to fix (Recommended)":)* a Messages-format request on a route where no source states any limit (an OpenCode messages model with no registry entry, no extract section stating limits, no layer): `ErrInvalidRequest` before any request reaches the server, and its message names the fix ("maxTokens" and "WithModelLayer") | red: 8192 sent | (a) the largest current limit chosen; (b) retired models counted; (c) a wait for the listing; (d) 8192 sent where no source states a limit: the server sees a request; (e) the refusal's message without the fix: the message assertion fails |
| OQ2 (2) | `TestOutputLimit_FirstUse_0089` | DL50, DL51 | the first request waits for the listing (one request) and sends its value; a concurrent second shares it; a listing failing: the request goes on, and a cold request within the next minute does not wait; then as OQ2 (1)'s fallback cases | red: no listing read | (a) no wait; (b) a wait per request; (c) a wait again within the minute |
| OQ2 (3) | `TestOutputLimit_Fallback_0089` | DL50, DL51 | a model in no source: 8192, basis fallback, one `WarningOutputLimitUnknown`; a numberless refusal: resent once at 8192 | red: no warning | the warning omitted |
| OQ3 (1) | `TestLimitLearning_Default_0089` | DL49, DL73 | a cap refusal: resent at the cap, kept, warned; with `WithoutLimitLearning()`: resent, not kept, no warning | does not compile | (a) the memory off by default; (b) the option ignored |
| OQ3 (2) | `TestLimitLearning_OptIn_0089` | DL49, DL73 | without the option: resent at the cap, nothing kept, no warning, and the next request refused again; with `WithLimitLearning()`: kept and warned | does not compile | (a) kept without the option; (b) the option ignored |
| OQ3 (3) | `TestLimitLearning_OptInAll_0089` | DL49, DL73 | without the option: one send, the refusal returned; with it: as OQ3 (2) with the option | does not compile | a resend without the option |
| OQ4 (1) | `apidiff` | DL52 | no `Request.OutputCeiling` | | the field added: reported |
| OQ4 (2), (3) | `TestOutputLimit_Ceiling_0089` | DL52 | `OutputCeiling` 20,000 with Claude 128,000: 20,000, basis fact, the resend still runs; 1,000: `ErrInvalidRequest` (below 1,024); above the fact: the fact; under OQ1 (1) on an optional wire: the ceiling sent | does not compile | (a) the ceiling counted as given: no resend; (b) the floor removed |

## OQ1 (1)'s measurement (only if OQ1 is answered (1))

`TestLive_OmittedLimit_0089`, in `llmprovider/live_0089_omitted_test.go`,
built with `live_gateways`, switched on by `LLMPROVIDER_LIVE_0089`, with
`LLMPROVIDER_LIVE_0089_MODELS` naming two Kilo models pinned in the test's doc
comment when it is written: the Claude model on Kilo with the smallest stated
limit (64,000 in the 2026-10-12 snapshot), and a non-Claude model whose stated
limit is at most 65,536. For each, one streamed request with no output limit,
asking for more output than its stated limit. It runs with `-timeout 30m` and
writes only an allowlisted line per model (route, model, the stated limit and
its source, HTTP status, finish reason, output and reasoning tokens) to
`tmp/live-0089/` (git-ignored), never a key or a body; it is run under the
repository's live-run rules (AGENTS.md "Live tests"): the environment's API
key only, by an agent or the owner, never a sign-in or a saved session. Its
cost: at most 64,000 plus 65,536 billed output tokens. The verdict decides
DL65 by a dated row: a `length` finish below the stated limit means Kilo fills
less, and it keeps sending the discovered limit; a `length` finish at or above
the stated limit means Kilo omits the field as the other optional wires do; a
`stop` decides nothing, and Kilo keeps sending until a measurement ends in
`length`.

## Verification

* Every decided row has its test green and its plant caught, quoted in the
  execution record; each test marked red was seen red at its base.
* `apidiff` lists the lines above for the answers given, and nothing else; the
  gates pass.

## Rollout and Rollback

* Rollout: in the release 0088's OQ18 names; a consumer takes it by bumping
  the module. A caller that wants a fixed limit passes
  `Request.MaxOutputTokens`, sent as given.
* Rollback: before the tag, revert the phase's commit; after it, removing an
  exported name is a new record, not a revert.

## Execution Record

No phase has run. *(Annotated 2026-11-22: approved; Phase 1 is recorded
below. Annotated 2026-11-23: Phase 1 pass 1 ran, below.)*

### Drafting (2026-11-22)

* First draft: from the owner's reply on 0088's OQ9, research rounds 1 to 3,
  record reviews 1 and 2, this repository at 751e970 (its `llmprovider/`
  unchanged from 18e8e72) and gobble-cli at dfaa1ac, all read only.
* Second draft, after record review 3: each finding re-verified against the
  code before it was written in (the MADR's note at its head); the owner's
  question on omission and design note folded in. Records only: no code,
  test, index row, staging or commit.
* The owner's answers (2026-11-22) to OQ1 to OQ4 and 0088's OQ18 recorded in
  the MADR's "Owner rulings"; the MADR set `accepted`; this PLAN's ordering
  and its chosen tests noted. Records only.
* The owner's DL51 ruling (2026-11-22), "Refuse, say how to fix (Recommended)", recorded in the MADR; `TestOutputLimit_Fallback_0089` asserts the refusal and its message, with plants (d) and (e). Records only.

### 2026-11-22: the owner's approval and start

The owner, relayed by the coordinator, approved this PLAN and 0088-PLAN and
the start, by the label "Approve both; start now (Recommended)". The status
went to `in-progress` by
`python3 -B tools/records.py status docs/decisions/0089-PLAN-model-output-limit-discovery.md in-progress --date 2026-11-22`,
which wrote the front matter and the index row and exited 0
(`records: written; records-check: 0 problem(s)`). The build order stands:
Phases 1 and 2 first, then OQ1 (1)'s measurement, then Phase 3 with
0088-PLAN's Phase 1. Phase 1's base is f86933c, the v1.32.0 release commit.

### 2026-11-22: deviations 1 to 3, before Phase 1's tests

Phase 1 pass 1 stopped at its step 2, before any test or code, on a row it
could not meet as written; the coordinator put three findings to the owner.
The rulings, relayed by the coordinator:

* **Deviation 1 (DL24).** `TestDiscover_Offline_0089` named a passed-back cap,
  which exists only through `llmprovider.WithLimitObservations`, an export of
  Phase 2. The owner's ruling: "Split the test across phases (Recommended)". Phase 1 builds the
  test with the caller layer, the embedded extract and no evidence from the
  empty document; Phase 2 adds the passed-back-cap case to the same test (its
  table's new row). DL24 is fully pinned after Phase 2.
* **Deviation 2 (DL74, a missing case).** `Limits.Required` on OpenCode's
  messages route could be read only from the models.dev document's
  `provider.npm`; the route table is in `providers/opencode`, which imports
  `catalog`, and gobble-cli passes `WithoutModelMetadata`. The owner's ruling:
  "Share the route table (Recommended)". OpenCode's route snapshot and table move to an
  internal package that `catalog` and `providers/opencode` both read (the
  Scope's new line), so `Required` is right with or without the document;
  the MADR's new row DL74, its test `TestDiscover_RequiredWithoutDocument_0089`
  and its plant are in Phase 1's table. OpenCode's behaviour and its existing
  tests stay unchanged; an existing test needing a change is still a stop.
* **Deviation 3 (DL72's test place).** `TestDiscover_RequestReadsDiscovery_0089`
  builds a Claude provider, which a `catalog` test cannot import (an import
  cycle). Decided by the coordinator from the precedent of
  `llmprovider/catalog/evidence_0041_test.go` (its assertion in
  `providers/grok`): the test is in `llmprovider/providers/claude`.

No build pass was used: the next run of Phase 1 is pass 1.

### 2026-11-23: deviations 4 and 5, before Phase 1's tests

* **Deviation 4 (the snapshot's place).** An internal package that `catalog`
  and `providers/opencode` both read cannot embed `routes_snapshot.json` while
  it stays in `providers/opencode/` (`go:embed` reaches only its package's
  directory and below), and `llmprovider/live_0031_measure_test.go:108`, an
  existing live-tagged test, reads the file by that path. The owner's ruling,
  relayed by the coordinator: "Move it; fix the one path (Recommended)". The file moves to
  `llmprovider/internal/opencoderoute`; the path at
  `llmprovider/live_0031_measure_test.go:108` is the one existing test edited,
  by the owner's leave; `providers/opencode` keeps thin wrappers with today's
  names and types (`routesSnapshot`, `routeTable`, `buildRouteTable`,
  `heuristicRoute`, `routeForNPM`), so every other existing test is
  unchanged; `docs/architecture.md`'s opencode row is updated in the docs
  step.
* **Deviation 5 (DL75, the workspace a lookup reads).** Decided by the
  coordinator from precedent: the catalog reads only
  `catalog.WithClaudeWorkspace` (`llmprovider/catalog/config.go:170-178`), and
  the Claude provider converts its own `WithWorkspace` into that option for
  its listing (`llmprovider/providers/claude/claude.go:270`); so
  `LookupLimits` follows the same rule, its doc comment tells a caller to pass
  `catalog.WithClaudeWorkspace`, and `TestDiscover_RequestReadsDiscovery_0089`
  pins it (the MADR's new row DL75).

### 2026-11-23: Phase 1, pass 1

Base f86933c (the v1.32.0 commit, which the tag `v1.32.0` names). No live
call, sign-in or provider API: every reply came from a committed fixture or a
test's own server.

**Re-grounding.** `python3 -B tools/claims.py docs/decisions/0089-MADR-model-output-limit-discovery.md --rev self=f86933c`:
`49 of 49 pass`, exit 0; no row moved. The files step 1 names were read at
the base.

**Red.** Every Go test of the table does not compile at the base (no
`LookupLimits`), so, by step 3, each was dropped from the red run and its
plant is its red. `test_limitsnapshot_keeps_limits_only` was run before the
tool existed: `ModuleNotFoundError: No module named 'limitsnapshot'`, 2
errors, exit 1. Within this pass the code and the Go tests were written
together; no test was seen to pass before its plant was seen to fail it.

**The code.** `llmprovider/catalog/limits.go` (the exports, the ranks,
`LookupLimits`, `CachedLimits`, the assembly, `Required`) and
`limits_reads.go` (the listing's extra fields, the one-model, served and
backend reads in their own maps, the embedded extract); the listing readers
(`discovery.go`, `listing_metadata.go`) emit Together's
`config.max_output_length`, Hugging Face's per-provider `context_length` and
Kilo's `per_request_limits` to a second sink, stored per listing key on each
fetch (`cache.go`, `config.go`); a chatcompat listing keeps its dialect's
name (`listing_cache.go`); `fetchClaudePage` shares `newClaudeRequest` with
the one-model read. `llmprovider/internal/opencoderoute` holds the moved
`routes_snapshot.json` (byte-identical, `cmp` silent) and the table;
`providers/opencode/route.go` keeps `routesSnapshot`, `routeTable`,
`buildRouteTable`, `heuristicRoute` and `routeForNPM` as wrappers, and every
existing OpenCode test passed unchanged; `llmprovider/live_0031_measure_test.go:108`
names the new path (deviation 4). `tools/limitsnapshot.py` wrote
`llmprovider/catalog/limits_snapshot.json` (775 models, nine sections, 86 KB)
from the committed models.dev fixture
`llmprovider/catalog/testdata/ranking-2026-10-12/metadata.json`, dated
2026-10-12: no document was fetched in this phase; Phase 7 regenerates it
from a public document before the tag.

Implementation choices within the rows, for review: the one-model read runs
only when the listing succeeded and did not name the model; a server's
`/api/ps` 404 is read as none loaded; the ChatGPT backend's and a plan
session's listing replace OpenAI's for such a session, their source named
`listing:openai`; the extract is matched by the same stages as the document,
named `models.dev (bundled <date>)`, `models.dev (bundled <date>, canonical
id)` and `models.dev (bundled <date>, family)`; a provider with no models.dev
section (Ollama) takes no registry evidence, so a model not loaded has its
trained context only (DL13); within the registry, a "models.dev" layer ranks
above the fetched document, which ranks above the extract.

**Green.** `go test -count=1 -v -run '_0089$|ExampleLookupLimits' ./llmprovider/catalog/`:
the 21 tests and the example each `--- PASS`, `ok`;
`TestDiscover_RequestReadsDiscovery_0089` in `providers/claude` `--- PASS`;
`TestLimitSnapshot` 2 tests `OK`; the new package's `TestOf`, `TestForNPM`,
`TestBuildTable` pass.

**Plants**, each by `python3 -B tools/plant.py` on a scratch clone at the
final code (catalog spec: `all 32 plant(s) caught`; Claude, tool and DL21
focused specs: each `all 1 plant(s) caught`; the new package's:
`all 3 plant(s) caught`):

| Plant | Test that fails | Its failing line |
| :--- | :--- | :--- |
| DL1, DL3 (a) `max_input_tokens` read as the window | `TestDiscover_ListingFacts_0089` | `limits_0089_test.go:248: claude input = {Tokens:0 Known:false ...}; want 1000000 from listing:claude` |
| DL5 (b) the null read as 0 and known | `TestDiscover_ListingFacts_0089` | `limits_0089_test.go:274: kilo null cap = {Tokens:0 Known:false ... Evidence:[{Source:listing:kilo Tokens:0 ...}]}; want unknown, no source stating it` |
| DL2 the one-model read skipped | `TestDiscover_ClaudeAlias_0089` | `limits_0089_test.go:288: alias output = {Tokens:0 Known:false ...}; want 64000 from listing:claude` |
| DL66 (a) the reads dropped with a listing refresh | `TestDiscover_ReadsSurviveRefresh_0089` | `limits_0089_test.go:355: the alias after a refresh = {Tokens:0 Known:false ...}; want 64000 from listing:claude` |
| DL66 (b) the 404 not kept | `TestDiscover_ReadsSurviveRefresh_0089` | `limits_0089_test.go:336: a 404, looked up twice within ten minutes: 2 requests; want 1` |
| DL4 (a) the larger Kilo value | `TestDiscover_UnreadFields_0089` | `limits_0089_test.go:377: kilo per-request output = {Tokens:65536 ...}; want 32768` |
| DL9 (b) Together's field unread | `TestDiscover_UnreadFields_0089` | `limits_0089_test.go:382: together output = {Tokens:0 Known:false ...}; want 40000 from listing:together` |
| DL10 (c) the largest Hugging Face window | `TestDiscover_UnreadFields_0089` | `limits_0089_test.go:388: hugging face window, the smallest = {Tokens:1000000 ...}` |
| DL8 (d) the ChatGPT field unread | `TestDiscover_UnreadFields_0089` | `limits_0089_test.go:395: chatgpt window = {Tokens:0 Known:false ...}; want 272000 from listing:openai` |
| DL12 (a) the trained context ranked first | `TestDiscover_OllamaServed_0089` | `limits_0089_test.go:411: loaded window = {Tokens:131072 Known:true Conflicting:true Evidence:[{Source:listing:ollama ...` |
| DL12 (b) `/api/ps` unread | `TestDiscover_OllamaServed_0089` | `limits_0089_test.go:411: loaded window = {Tokens:131072 Known:true Conflicting:false ...` |
| DL15 (a) the "models.dev" layer ranked as a caller's | `TestDiscover_EvidenceKinds_0089` | `limits_0089_test.go:470: listing over a models.dev layer = {Tokens:64000 ... Evidence:[{Source:models.dev ...` |
| DL16 (b) "models.json" ranked as registry | `TestDiscover_EvidenceKinds_0089` | `limits_0089_test.go:479: a caller's layer over the listing = {Tokens:128000 ... Evidence:[{Source:listing:claude ...` |
| DL29 (c) the extract dropped with the document | `TestDiscover_EvidenceKinds_0089` | `limits_0089_test.go:475: output = {Tokens:128000 ...}; want the extract's 50,000 kept with the document off` |
| DL70 the extract ranked above the document | `TestDiscover_EmbeddedExtract_0089` | `limits_0089_test.go:502: document over extract = {Tokens:64000 ... Evidence:[{Source:models.dev (bundled 2026-10-12) ...` |
| DL70 a non-limit field kept by the tool | `test_limitsnapshot_keeps_limits_only` | `FAIL: test_limitsnapshot_keeps_limits_only (test_tools.TestLimitSnapshot.test_limitsnapshot_keeps_limits_only)` |
| DL71 (a) the family match chosen over the exact | `TestDiscover_MatchStages_0089` | `limits_0089_test.go:535: exact = {Tokens:64000 ... Evidence:[{Source:models.dev (family) ...` |
| DL71 (b) a family match named as exact | `TestDiscover_MatchStages_0089` | `limits_0089_test.go:536: family = {Tokens:64000 ... Evidence:[{Source:models.dev ...` |
| DL17 (a) the lower rank chosen | `TestDiscover_Conflict_0089` | `limits_0089_test.go:550: listing over document = {Tokens:128000 ... Evidence:[{Source:models.dev Tokens:128000 ...` |
| DL18 (b) the earlier layer chosen | `TestDiscover_Conflict_0089` | `limits_0089_test.go:558: the later layer = {Tokens:20000 ... Evidence:[{Source:first ...` |
| DL19 the hold removed | `TestDiscover_OutputAboveWindow_0089` | `limits_0089_test.go:572: output = {Tokens:16384 Known:true Conflicting:false ...}` |
| DL20 0 known | `TestDiscover_StatedZero_0089` | `limits_0089_test.go:584: output = {Tokens:0 Known:true ...}; want unknown, the 0 kept` |
| DL21 an error returned | `TestDiscover_Unknown_0089` | `limits_0089_test.go:595: err = planted; want nil (a focused run; the full run's first line is another test's)` |
| DL22, DL26 the error dropped | `TestDiscover_SourceFailure_0089`, `TestDiscover_Cancelled_0089` | `limits_0089_test.go:615: 503: err = <nil>; want the listing's *APIError` |
| DL23 `Stale` left false | `TestDiscover_Stale_0089` | `limits_0089_test.go:639: evidence = {Source:listing:claude ... Stale:false}; want stale` |
| DL24 an empty document read as a fact | `TestDiscover_Offline_0089` | `limits_0089_test.go:667: evidence {Source:models.dev Tokens:0 ...} from the empty document` |
| DL25 the closed check removed | `TestDiscover_Closed_0089` | `limits_0089_test.go:696: ollama: 2 requests after Close; want none` |
| DL27 a fetch per call (`cachedOrFetched` replaced by `fetchClaimed`) | `TestDiscover_Concurrent_0089` | `limits_0089_test.go:753: listing requests = 16; want 1` |
| DL28 nil error | `TestDiscover_InvalidProvider_0089` | `limits_0089_test.go:763: err = <nil>; want ErrInvalidProvider` |
| DL30 (a) `Required` on Kilo | `TestDiscover_WireFacts_0089` | `limits_0089_test.go:792: kilo anthropic/claude-opus-4.6: Required = true; want false` |
| DL30 (b) the messages route not required | `TestDiscover_WireFacts_0089` | `limits_0089_test.go:792: opencode-zen claude-opus-4-6: Required = false; want true` |
| DL74 `Required` read from the document only | `TestDiscover_RequiredWithoutDocument_0089` | `limits_0089_test.go:810: Zen claude-opus-4-6 with no document: Required false; want true` |
| DL72, DL75 the stored key without the scope | `TestDiscover_RequestReadsDiscovery_0089` | `limits_discovery_0089_test.go:72: max_tokens = 200000; want 128000, the listing's limit the lookup stored` |
| R50 the `// Output:` line changed | `ExampleLookupLimits` | `--- FAIL: ExampleLookupLimits` |
| (the new package's own) gemini routed to messages on Zen | `TestOf` | `opencoderoute_test.go:33: Of(opencode-zen, "gemini-not-listed") = "messages", want "google"` |
| (the new package's own) google's package routed to responses | `TestForNPM` | `opencoderoute_test.go:43: ForNPM("@ai-sdk/google") = "responses", want "google"` |
| (the new package's own) a bad snapshot accepted | `TestBuildTable` | `opencoderoute_test.go:58: BuildTable accepted it` |

A first plant of the new package's own, "the table not read", was not
caught: the provider's existing `TestRouteHeuristic_AgreesWithEveryRow`
holds the heuristic equal to every table row, so the plant changed nothing
observable (an equivalent change); it was replaced by the `ForNPM` plant
above. It is not a Phase 1 row's plant.

**Checks**, each exit status taken before any filter:

| Command | Host | Exit |
| :--- | :--- | :--- |
| `run-heavy -- go test -count=1 ./...` | session (darwin/arm64) | 0, 45 packages ok, before and after the lint fix |
| `run-heavy -- make pre-add-check FILES="<the 15 Go files>"` | session (darwin/arm64) | 2 on the first run (goconst: "live" three times; gocritic weakCond in familyID), fixed; then 0: `15 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)` |
| `run-on-host --race -- go test -race -count=1 (catalog, internal/opencoderoute, providers/opencode, providers/claude)` | a fleet Linux host (linux/amd64), not WSL; again after the lint fix on a fleet Windows host (windows/amd64) | 0, 0 |
| `run-heavy -- make api-check contract-check generate-check docs-check coverage-check gate-selftest dep-check` | session (darwin/arm64) | 0 (api-check, docs-check, coverage-check, gate-selftest again after the lint fix: 0) |
| `make records-check tables-check changelog-check links-check disclosure-check markdownlint parity-check fixtures-check tools-test` | session (darwin/arm64) | 0 each; tables-check first exited 2 (R2's table had no `internal/opencoderoute` row), fixed in the docs step |
| `go vet -tags live_gateways ./...` | session (darwin/arm64) | 0 |
| `python3 -B tools/apidiff.py v1.32.0` | session (darwin/arm64) | 0, the five lines below |
| `python3 -B tools/claims.py docs/decisions/0089-MADR-model-output-limit-discovery.md` | session (darwin/arm64) | 0, 49 of 49 pass |
| `python3 -B tools/recordlint.py --changed; python3 -B tools/denyscan.py --changed` | session (darwin/arm64) | 0, 0 |

**`apidiff`** (`python3 -B tools/apidiff.py v1.32.0`), whole:

```text
Compatible changes:
- ./llmprovider/catalog.CachedLimits: added
- ./llmprovider/catalog.Evidence: added
- ./llmprovider/catalog.Fact: added
- ./llmprovider/catalog.Limits: added
- ./llmprovider/catalog.LookupLimits: added
```

Phase 1's five names, as the Scope's list names them, and nothing else.

**Docs.** Every new exported name has its doc comment, with R51's and R52's
words; `llmprovider/catalog/doc.go` gains the limits paragraph;
`docs/architecture.md`'s layout, `catalog` row and `opencode` row, and a row
for `internal/opencoderoute`; `docs/guides/api-standards.md` R2's table
(`catalog` may import `internal/opencoderoute`; its row) and R2's
citations; `tools/README.md`'s `limitsnapshot.py` row; `CHANGELOG.md`
**Added** under `[Unreleased]`. Not done, and why: R6 and R7 are unchanged,
since no request or response changes in Phase 1 and their text stays true;
R57 is not added, since its sentence ("No fixed number decides an output
limit a caller did not give") is false until Phase 3's request rule, and
Phase 4 step 3 adds it with its Checks row.

**Not done in Phase 1, as planned:** the passed-back cap's case of
`TestDiscover_Offline_0089` (Phase 2, deviation 1); Phases 2 to 7; the live
measurement. Nothing staged or committed.

Exit criterion: no blocker found by the builder; every Phase 1 row tested
(DL24 in its Phase 1 part); every plant caught. Pass count: 1.
