---
status: in-progress
date: 2026-11-23
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
(1); the rest are not built.)* *(2026-11-23, deviation 6: Phases 2 and 3 are built with 0088's Phase 1 as one
phase, the "Merged phase" under Implementation Steps, after 0088's OQ10
measurement; the Kilo measurement follows it and gates no code.)*

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
  *(2026-11-23, deviation 7: the memo is in a new leaf package,
  `llmprovider/internal/limitmemo`, one per `catalog.Cache` through
  `internal/cacheref`, with its R2 and `docs/architecture.md` rows.)*
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

*(2026-11-23, deviation 6: built in the "Merged phase" below, with Phase 3 and 0088-PLAN's Phase 1.)*

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
   | `TestLimitRefusal_WindowForms_0089` | DL35 | Kilo, wire 16,384, C39's form with window 32,768 and about 40,000: the resend's room 9,152; Kilo's documented form; Together's split; after each, `CachedLimits` shows no window evidence and the memo is empty *(2026-11-23, confirmed by the coordinator from the merged phase's 0088 table, which builds ST55's first-payload resend on Kilo only because Together sends no limit the SDK chose (DL47): Together's case is a caller's `WithMaxTokens(16384)` refused in the split form: one send, the refusal returned, the memo empty, no window evidence)* | does not compile | the window kept in the memo *(2026-11-23: and V5's, a window not above the input it states read)* |
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

*(2026-11-23, deviation 6: built in the "Merged phase" below.)*

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

### Merged phase: Phases 2 and 3 with 0088-PLAN's Phase 1 (2026-11-23, deviation 6)

*(2026-11-23, deviation 6: the owner's rulings "Merge the phases" and "Measure first, then build (Recommended)", relayed by the coordinator. Phases 2 and 3 above, and [0088-PLAN-default-output-limit-from-model-facts.md](0088-PLAN-default-output-limit-from-model-facts.md)'s Phase 1, are built as this one phase, in one commit; their tables stand as the tests' definitions, read with the values below. Its base is `6ace750`.)*

**Why one phase.** Three of Phase 2's tests, `TestLimitMemo_Concurrent_0089` (DL43), `TestLimitRefusal_WindowForms_0089` (DL35) and `TestLimitRefusal_CallerLimit_0089` (DL37), assert the resend at a stated cap, which Phase 3 builds with 0088's Phase 1; and the resend tells a given limit from the default only through 0088's `Settings.MaxTokensGiven` (0088-MADR OQ5, ST35).

**Steps.**

0. **0088's OQ10 measurement, before the build** (the owner's ruling "Measure first, then build (Recommended)"), as 0088-PLAN's first amendment defines it: `TestLive_ClaudeWindowRefusal_0088`, built with `live_gateways`, switched on by `LLMPROVIDER_LIVE_0088` with `LLMPROVIDER_LIVE_0088_MODEL` (both named in AGENTS.md's "Live tests"; `make tables-check` passes), run once under the repository's live-run rules (the environment's API key only, checked by name; allowlisted output only), one request of about 192,000 input tokens with `max_tokens` 32,000 on a Claude model before 4.5; the refusal's status, error type and message, redacted, recorded under `llmprovider/providers/claude/testdata/window-0088/`, and quoted in a dated 0088-MADR amendment. If the message states no numbers, the build does not start: ST65's resend value then needs a ruling, since ST99 retired the 8192 resend and 0089-MADR D6 resends only at a stated number. A second run needs the coordinator's word. *(2026-11-23: run once on `claude-sonnet-4-20250514` by the owner's ruling "Try claude-sonnet-4 once": HTTP 404 `not_found_error`; ST65 out of scope, see 0088-MADR's sixth amendment.)*
1. **Re-ground** both MADRs at `6ace750` (`tools/claims.py --rev self=6ace750`), annotating each moved row; read the files Phase 2's and Phase 3's step 1 and 0088-PLAN's Phase 1 step 1 name; confirm 0081, 0083, 0085 and 0087 changed no line that computes or sends an output limit (0088 ST42); list the existing `Generate` tests whose servers answer plain JSON (0088 ST88e), and every other existing test, golden or fixture the change touches. One that no row below names is a stop.
2. **Tests first**, in the files the three tables name. A test of code that exists at the base is run red there (`tools/redgreen.py --expect fail --rev 6ace750`); a test of new API takes its red from its plant, as Phase 1 recorded; a build failure is not the expected red.
3. **The code**: the memo, the validator and the pass-back (Phase 2); the request rule (MADR D6), one helper in `internal/wire` that every body function calls, the resend at the validated stated number, and the report (Phase 3); `Settings.MaxTokensGiven`, the stream resend, Kilo's window form, the Generate over the stream, Grok's silence and ST65's form (0088's Phase 1).
4. **Green and plants**: every test `--expect pass`; every plant by `tools/plant.py` on a scratch clone, caught only by a failing test, its failing line quoted.
5. **Docs**: every new export's doc comment; `docs/architecture.md`; `docs/guides/api-standards.md` R6, R7 and R57 where the merged code makes them true; `CHANGELOG.md` under `[Unreleased]`; 0088's consumer note (`docs/guides/adding-a-provider.md`, `MaxTokensGiven`); AGENTS.md "Live tests" for `LLMPROVIDER_LIVE_0088`.
6. **Checks**: `make pre-add-check FILES="..."` and `go test ./...` through `run-heavy`; CI's set (parity, dep, coverage, api, contract, generate, records, tables, fixtures, changelog, docs, markdownlint, links, disclosure, gate-selftest, tools-test); `go vet -tags live_gateways ./...`; `-race` on the touched packages through `run-on-host` (a Linux host, not WSL; the Windows laptop where Windows code changes); `tools/apidiff.py v1.32.0`; `tools/claims.py` on both MADRs; `tools/recordlint.py --changed`; `tools/denyscan.py --changed`.
7. **The execution record**, in this PLAN and in 0088-PLAN, with the pass number; MADR annotations where a test's name or place differs.

**From this PLAN's Phase 2**, every row of its table: `TestLimitRefusal_Validator_0089` (DL31 to DL34, DL67), `TestLimitRefusal_FormsByWire_0089` (DL68), `TestLimitRefusal_WindowForms_0089` (DL35; 0088 ST92), `TestLimitRefusal_NamedModel_0089` (DL36), `TestLimitRefusal_CallerLimit_0089` (DL37; 0088 ST18), `TestLimitMemo_Scope_0089` and `TestLimitMemo_Bound_0089` (DL38), `TestLimitMemo_SelfHeal_0089` (DL39), `TestLimitMemo_Concurrent_0089` (DL43), `TestLimitObservations_PassBack_0089` (DL40, DL41), `TestLimitObservations_Invalid_0089` (DL42), and `TestDiscover_Offline_0089`'s passed-back case (DL24, deviation 1); each with the plants its row names.

**From this PLAN's Phase 3**, with the chosen "Tests by answer": `TestOutputLimit_Reported_0089` (DL44, DL53, DL54; 0088 ST15 to ST17, ST67), `TestOutputLimit_BudgetReported_0089` (DL55; 0088 ST2), `TestOutputLimit_OutsideProvider_0089` (DL57), `TestOutputLimit_Default_0089` (DL45, DL8's plan case; 0088 ST4), `TestOutputLimit_Omitted_0089` (OQ1 (1): DL46 to DL48, DL69, DL65's interim; 0088 ST33, ST74, ST85, ST86, ST90, ST98, ST99), `TestOutputLimit_Fallback_0089` (OQ2 (1): DL50, DL51; 0088 ST11, ST12, ST34, ST54), `TestLimitLearning_Default_0089` (OQ3 (1): DL49, DL73; 0088 ST49, ST89), and OQ4 (1)'s `apidiff` check (DL52: no `Request.OutputCeiling`); each with the plants its row names.

**From 0088-PLAN's Phase 1**, as its fifth amendment and 0088-MADR ST99 leave it:

| 0088 test | Rows | In the merged phase |
| :--- | :--- | :--- |
| `TestSettings_MaxTokensGiven_0088` | ST35, ST36 | built as written |
| `TestDefaultLimit_GrokSilent_0088` | ST63 | built as written (third amendment) |
| `TestDefaultLimit_WindowCap_0088` | ST91 | built on a route that sends a limit (Kilo, the discovered limit until DL65's verdict): a layer stating L 16,384 and W 8,192: 8,192; L 20,000 and W 1,000,000: 20,000; `WithMaxTokens(20000)` with W 8,192: 20,000, as given; plant: the window cap removed (16,384 sent) |
| `TestDefaultLimit_ClaudeOldWindowForm_0088` | ST65 | built from step 0's recorded refusal: a limit the SDK chose, refused in that form, resent once at the window less the input the form states (as ST92 derives it); a caller's `WithMaxTokens(20000)` so refused: one send; plant: the form not recognised *(2026-11-23: not built; ST65 is out of scope after the measurement's 404, the owner's ruling "Try claude-sonnet-4 once")* |
| `TestGenerateOverStream_0088`, `_Retry_0088`, `_ReplyLimit_0088`, `_NoUsage_0088`, `_ErrorIn200_0088` | ST88, ST88a to ST88e | built as the third amendment's OQ15 (1) rows; ST88d's resend is at the room the refusal states (ST92), not OQ14's; ST88e's existing tests listed at step 1 |
| `TestClassify_KiloWindowForm_0088` | ST51 | built as written |
| `TestStreamEventsRetry_FirstPayloadError_0088` | ST55 | built on Kilo only (Together sends no limit the SDK chose, DL47): the first payload in C39's form, resent once at the room it states; an error after a text delta: one send; plants as written |
| `TestDefaultLimit_ResendLadder_0088` | ST56, ST57 | built with the resend at the stated cap, not 8192 (ST99): case 1's third send at the cap with the lower effort; cases 2 and 3 as written; the provider named at step 1 |
| `TestDefaultLimit_Concurrent_0088` | ST46 | built on Kilo: each refused request sends its discovered limit, then the stated room; each other one send; plant as written |
| `TestDefaultLimit_ListingProbe_0088` | ST26 | built as written (also DL53) |
| `TestDefaultLimit_ClaudeWindowStop_0088` | ST23 | built as written |
| `TestDefaultLimit_OverflowResend_0088` | ST48, ST59, ST61 | built on the routes that send a limit the SDK chose (Claude, OpenCode's messages route, Kilo), cases 3, 5, 6 and 7: one send each; ST21's and ST22's resend at 8192 is retired by ST99 (a window form stating numbers is DL35's test; one stating none is returned, 0089-MADR D6); plants (b), (e), (f) |
| `TestDefaultLimit_LimitRefusal_0088` | ST50 | built for its Kilo `length` case only; ST49's resend is `TestLimitLearning_Default_0089`'s |
| `TestDefaultLimit_ResendBudgetShape_0088` | ST52 | built with ST99's value: Claude stated 128,000, budget 10,000, refused `max_tokens: 128000 > 64000, ...`: the second send `max_tokens` 64,000 and `budget_tokens` 10,000; plants (a) and (b) as written |
| `TestDefaultLimit_Facts_0088`, `_BudgetShape_0088`, `_UnknownLimit_0088`, `_StatedZero_0088`, `_CallerGiven_0088`, `_RequestGiven_0088`, `_KiloListing_0088`, `_LocalFacts_0088`, `_WindowFit_0088` | ST1 to ST3, ST5 to ST17, ST24, ST25, ST28, ST29, ST33, ST34, ST38, ST47, ST54, ST60 | not built: ST99 retires their values, and the fourth amendment the half-window hold; their rows that stand are pinned by this PLAN's tests named beside them above, ST13 by `TestDiscover_StatedZero_0089` with `TestOutputLimit_Fallback_0089`, ST16 by `TestSettings_MaxTokensGiven_0088`, ST60 by ST27's existing test |
| the existing tests of 0088-PLAN Phase 1 step 2 | ST4, ST19, ST20, ST27, ST30, ST41, ST45, ST61 | each with its plant, as written |
| `TestOutputLimit_Facts_0041` (existing) | ST32 | changed as ST32 planned, its values DL46 to DL48's (ST99) |
| `TestWireGoldens` in each provider package (existing, 2026-11-23) | ST100, ST16 | the G-wire goldens regenerated with `-update`, each diff checked by a script to touch only the output-limit field and, where ST88 applies, the stream form (the script seen failing on a planted unrelated difference); one new golden per wire sent with `WithMaxTokens(8192)`; plant: `WithMaxTokens(8192)` not counted as given (the 0088 owner's ruling "Same, plus explicit-8192 goldens") |
| existing tests and goldens pinning the old 8,192 default on a route that now omits it (2026-11-23, 0088-MADR ST102, the owner's ruling "One class row, assert absent (Recommended)") | ST102 | each assertion becomes "no limit sent", red at the base; plant: 8192 sent on that route. Members: `TestTogether_RequestShapes` (`llmprovider/providers/together/together_test.go:83`) |

**Exports**, in `apidiff`'s form, beyond Phase 1's five (R55: the call sites are this PLAN's "Exported names" and 0088-PLAN's): `./llmprovider.LimitObservation`, `.WithLimitObservations`, `.WarningLimitLearned`, `.Warning.Observed` (Phase 2); `./llmprovider.SentLimit`, `.LimitBasis`, `.LimitBasisRequest`, `.LimitBasisOption`, `.LimitBasisFact`, `.LimitBasisRefusal`, `.LimitBasisFallback`, `.LimitBasisNone`, `.Response.OutputLimit`, `.WithoutLimitLearning` (Phase 3, OQ3 (1)); `./llmprovider.Settings.MaxTokensGiven` (0088 OQ5). Fifteen lines; `tools/apidiff.py v1.32.0` lists these and Phase 1's five, and nothing else. *(2026-11-23, the owner's ruling "Add a stream dimension", 0089-MADR DL76: `catalog.Limits` gains `StreamedOutput`, its call site gobble-cli's 0066 D7 and every provider's streamed request (in-module). `Limits` is new since v1.32.0, so `apidiff` reports it once and lists none of its fields (0044-MADR A1): the field adds no line, and its expectation is `./llmprovider/catalog.Limits: added`, already listed; Phase 4 confirms the field by `go doc`.)* Not exported: `WarningToolCallCut` (0088's Phase 1b), `Request.OutputCeiling` (OQ4 (1)), `WarningOutputLimitUnknown` (OQ2 (1)).

**How the pieces meet, as Phase 2's analysis found.** `WithLimitObservations` is a shared option that `catalog` reads through `Settings.Values()`, as it reads `WithModelLayer`'s layers (`llmprovider/settings.go:377`, `llmprovider/catalog/config.go:218`), so `LookupLimits` and a request read the same passed-back caps. The id the one-model read resolved (V6) reaches a provider through the internal `cacheref` value that `catalog.WithCache` carries (`llmprovider/internal/cacheref/cacheref.go`), with no new export.

**What falls where**, by the PLANs' own dependencies:

* 0088's OQ10 measurement and ST65: step 0 above, before the build.
* 0088-PLAN's Phase 1b (the cut answer, OQ11, `WarningToolCallCut`): no test of the merged phase depends on it (its cut-answer cases fail the same way on the plain and the streamed path at the base), so it is built after the merged phase, in its own commit, before the shared Phases 2 to 4.
* 0088-PLAN's Phase 1c (OQ14 (3), the estimate fit, with DL56's `TestOutputLimit_EstimateReadsLimits_0089`): it reads `Limits` and the merged request rule, so it follows the merged phase and Phase 1b.
* OQ1 (1)'s Kilo measurement (DL65): it gates no code. Until its verdict, Kilo keeps sending its discovered limit (DL65's interim, pinned by `TestOutputLimit_Omitted_0089`'s Kilo case), so it runs after the merged phase and before Phase 7; a verdict that Kilo omits the field is a dated row and a step of its own.
* This PLAN's Phases 4 to 7 and 0088-PLAN's Phases 2 to 4: once, over both PLANs, after these.

**Layers (2026-11-23, deviation 7).** The owner's ruling "Ordered layers, one commit (Recommended)": the phase above is built as five ordered layers, each its own run with a checkpoint here, in one commit at the end. A layer is not a build pass: only a rebuild after a failed gate or review counts toward the three-pass limit. A test whose definition asserts the resend, the request rule or a provider's request is built in the layer that builds those, whichever phase's table names it.

| Layer | Rows and tests | Status |
| :--- | :--- | :--- |
| A: Phase 2's API, the memo, the validator and the pass-back | `LimitObservation`, `WithLimitObservations`, `WarningLimitLearned`, `Warning.Observed`; the memo in `llmprovider/internal/limitmemo`, one per `catalog.Cache` through `internal/cacheref`; the validator and the forms by route in `internal/wire`. `TestLimitRefusal_Validator_0089` (DL31 to DL34; DL67's case at the validator, which compares the value it is given), `TestLimitRefusal_FormsByWire_0089` (DL68, the cap forms by route), `TestLimitRefusal_NamedModel_0089` (DL36, V6's keep decision), `TestLimitMemo_Scope_0089` and `TestLimitMemo_Bound_0089` (DL38), `TestLimitMemo_SelfHeal_0089` (DL39, the memo's drop), `TestLimitObservations_PassBack_0089` (DL40, DL41), `TestLimitObservations_Invalid_0089` (DL42), `TestDiscover_Offline_0089`'s passed-back case (DL24) | built, unstaged; checkpoint in the execution record |
| B: the request rule, the report and the resend; Claude, Kilo and OpenCode's messages route *(2026-11-23: and DL76's `Limits.StreamedOutput`, the owner's ruling "Add a stream dimension")* | `SentLimit`, `LimitBasis` and its six constants, `Response.OutputLimit`, `WithoutLimitLearning`; the request-rule helper in `internal/wire` (plain values); the resend at a validated cap; the warning; each response's output fed to the memo's drop (DL39's wiring); the resolved id reaching a provider through `cacheref` (V6); the memo's caps in `Limits` (design note 5). Moved here from A by their definitions: `TestLimitRefusal_WindowForms_0089` (DL35), `TestLimitRefusal_CallerLimit_0089` (DL37), `TestLimitMemo_Concurrent_0089` (DL43, 32 requests on a Claude provider, each refused one resent), DL67's plant (f) (the provider passing the pre-raise value). Then Phase 3's tests on these routes and `TestLimitLearning_Default_0089` (DL49, DL73) | stopped after its code, before its tests (2026-11-23): two existing-test findings put to the owner; see the execution record *(2026-11-23, deviation 8, the owner's ruling "Merge layers B and C": built with layer C as one layer)* *(2026-11-23: built with C, unstaged; checkpoint in the execution record)* |
| C: the optional wires and ST100's goldens *(2026-11-23, deviation 8: one layer with B)* | the request rule on every other route; ST100's goldens regenerated, the diff-check script seen failing first on a planted unrelated difference, and the explicit-8192 goldens | not started; built with B (deviation 8) *(2026-11-23: built with B, unstaged; checkpoint in the execution record)* |
| D: ST88 and ST88e | `Generate` over the stream; the ST88e changes to the existing plain-JSON `Generate` tests | not started *(2026-11-23: built, unstaged; checkpoint in the execution record)* |
| E: 0088's remaining tests, the docs and all gates | the rest of the 0088 table above; step 5's docs; step 6's checks in full; step 7's record | not started *(2026-11-23: built, unstaged; checkpoint in the execution record)* |

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
| OQ2 (1) | `TestOutputLimit_Fallback_0089` *(2026-11-23: in `llmprovider/catalog`'s external test package, `catalog_test`, which can build the Claude and OpenCode providers and replace the embedded extract through a test hook; its OpenCode case pins DL77, plant: the extract's whole section read)* | DL50, DL51, DL77 | a fresh cache, Claude: the first request reads the extract (no listing request during it); a model in no listing and no extract entry, on a route whose current models state limits: 64,000 (the smallest current `max_tokens` in the fixture listing), basis fallback; then refused stating 4,096: resent at 4,096; a numberless refusal: returned. *(2026-11-22, DL51 "Refuse, say how to fix (Recommended)":)* a Messages-format request on a route where no source states any limit (an OpenCode messages model with no registry entry, no extract section stating limits, no layer): `ErrInvalidRequest` before any request reaches the server, and its message names the fix ("maxTokens" and "WithModelLayer") | red: 8192 sent | (a) the largest current limit chosen; (b) retired models counted; (c) a wait for the listing; (d) 8192 sent where no source states a limit: the server sees a request; (e) the refusal's message without the fix: the message assertion fails |
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

### 2026-11-23: deviation 6, Phase 2 stopped; the phases merged; measure first

**Phase 2 stopped** before its code (the tree at `6ace750` holds no Phase 2 test or code): three of its tests, `TestLimitMemo_Concurrent_0089` (DL43), `TestLimitRefusal_WindowForms_0089` (DL35) and `TestLimitRefusal_CallerLimit_0089` (DL37), assert the resend at a stated cap, which Phase 3 builds with 0088-PLAN's Phase 1, and the resend needs 0088's `Settings.MaxTokensGiven` (0088-MADR OQ5) to tell a given limit from the default. Re-grounding at `6ace750`: `python3 -B tools/claims.py docs/decisions/0089-MADR-model-output-limit-discovery.md --rev self=6ace750`, 42 of 49 pass, exit 1; the seven rows that moved, none gone, are annotated in the MADR (its "Re-grounding at `6ace750`").

**The owner's ruling**, relayed by the coordinator: "Merge the phases". Phases 2 and 3 and 0088-PLAN's Phase 1 are built as one phase, the "Merged phase" above, which names its tests, rows, plants and exports.

**The merged phase stopped before its build** (2026-11-23), on a row it could not meet as written: 0088-PLAN orders OQ10's live measurement before its Phase 1 (its second and fourth amendments), and `TestDefaultLimit_ClaudeOldWindowForm_0088` (ST65) serves the refusal that measurement records; at `6ace750` no such recording, live test or `LLMPROVIDER_LIVE_0088` exists. The owner's ruling, relayed by the coordinator: "Measure first, then build (Recommended)": the measurement is the merged phase's step 0.

**Step 0 stopped before its run** (2026-11-23), on an unexpected state, put to the owner; no live request was sent:

* **S1, no model to measure.** The step needs a Claude model before 4.5. Neither 0088 record names one. The committed Claude listing, `llmprovider/catalog/testdata/firstparty-2026-10-14/claude.json`, lists 14 models, the oldest `claude-sonnet-4-5-20250929`, `claude-haiku-4-5-20251001` and `claude-opus-4-5-20251101`, and no model before 4.5; `llmprovider/internal/retired/retired.go` marks even `claude-haiku-4-5` past its retirement floor (2026-10-15). OpenCode Zen's listing (`ranking-2026-10-12/zen.json`) holds no Claude model before 4.5 either; Kilo's (`ranking-2026-10-12/kilo.json`) still lists `anthropic/claude-opus-4.1` and `anthropic/claude-sonnet-4`, behind Kilo's gateway and its own refusal forms. A request to an unlisted model would most likely be answered not found, spending the one run on no evidence.
* **S2, existing goldens no row names.** Under DL47 and 0088 ST99, a request that names no limit sends none on an optional wire, and on Messages the discovered or fallback limit, never 8192. 241 committed G-wire goldens record `"max_tokens": 8192` or `"max_output_tokens": 8192` in a request built with no caller limit (`wirecase.Run`, no `WithMaxTokens`): claude 26, chatcompat 12, gemini 14, grok 28, huggingface 12, kilo 12, ollama 13, openai 28, opencode 84, together 12. Each changes. ST32 names one existing test (`TestOutputLimit_Facts_0041`) and ST88e the plain-JSON `Generate` tests; no row names these goldens, so the build's step 1 would stop on them.

Nothing was built, staged or committed; no build pass was used.

**The owner's rulings on S1 and S2** (2026-11-23), relayed by the coordinator: S1 "Try claude-sonnet-4 once"; S2 "Same, plus explicit-8192 goldens". Step 0 ran once: `LLMPROVIDER_LIVE_0088=1 LLMPROVIDER_LIVE_0088_MODEL=claude-sonnet-4-20250514 go test -tags live_gateways -count=1 -run '^TestLive_ClaudeWindowRefusal_0088$' -v -timeout 10m ./llmprovider/`, exit 0; the record is HTTP 404 `not_found_error`, message `model: claude-sonnet-4-20250514` (0088-MADR's sixth amendment quotes it whole). By the S1 ruling, ST65 is out of scope and `TestDefaultLimit_ClaudeOldWindowForm_0088` is not built; by S2, 0088-MADR's ST100 names the G-wire goldens as a planned change, in the merged phase's 0088 table. The recorder's offline twin, `TestWindowRecord_Twin_0088`, passed, and its two plants (the message not redacted; `max_tokens` not 32,000) were each caught by `tools/plant.py`.

### 2026-11-23: the merged phase, pass 1 (stopped after its first layer)

Base `6ace750`. Built, unstaged: `Settings.MaxTokensGiven` (0088 ST35, ST36; `llmprovider/options.go`, `llmprovider/settings.go`) and `TestSettings_MaxTokensGiven_0088` (`llmprovider/settings_given_0088_test.go`), green; its red is its plants, since it does not compile at the base: (a) `WithMaxTokens` does not set the flag, caught (`settings_given_0088_test.go:35: given 8192: MaxTokensGiven = false; want true`); (b) the flag derived from `MaxTokens != 8192`, caught (the same line). `go test -count=1 ./llmprovider/` exit 0; `go vet ./llmprovider/` exit 0.

Stopped there and put to the owner: the phase as recorded is larger than one build run can carry with its rows' red runs, plants and gates. By count at `6ace750`: ST88e changes the existing `Generate` tests whose servers answer plain JSON, 335 call sites in 120 test files (no shared test server answers a streamed request); ST100's 241 goldens; about 45 new tests over ten providers and OpenCode's four routes, each with its plants. Nothing else of the phase is built. Pass count: 1.

### 2026-11-23: deviation 7, the merged phase built in ordered layers

**The stop's evidence**, as the entry above records it: one run carries the phase's first piece only (`Settings.MaxTokensGiven` and its test); the rest is ST88e's 335 `Generate` call sites in 120 test files moving to the stream, ST100's 241 goldens, and about 45 new tests over ten providers and OpenCode's four routes, each with its plants.

**The owner's ruling**, relayed by the coordinator: "Ordered layers, one commit (Recommended)". The merged phase stays one phase and one commit, built as ordered layers, each its own run with a checkpoint (the table "Layers" in the Merged phase section):

* A: Phase 2's API, the memo, the validator and the pass-back, with their tests.
* B: the request rule, the report and the resend, plus Claude, Kilo and OpenCode's messages route.
* C: the optional wires and ST100's goldens (the diff-check script seen failing first, and the explicit-8192 goldens).
* D: ST88 (`Generate` over the stream) and the ST88e test changes.
* E: 0088's remaining tests, the docs, and all gates.

**Pass counting.** A layer is not a build pass: only a rebuild after a failed gate or review counts toward the three-pass limit. The merged phase's pass count stays 1.

**Design decided by the coordinator** for the layers (from the stopped pass's analysis): the memo in a new leaf package, `llmprovider/internal/limitmemo` (256 entries; keyed by provider, the raw `Settings.BaseURL()`, scope, model and stream), with its R2 and `docs/architecture.md` rows, in place of the Scope's "the memo (D5) beside 0062's" in `internal/wire`; `WithLimitObservations` a shared option whose value is `[]LimitObservation`, read by `catalog` through `Settings.Values()`; `WithoutLimitLearning` a shared marker value from `limitmemo`; the request-rule helper in `internal/wire` taking plain values (`wire` cannot import `catalog`); the memo's caps in `Limits` as evidence, named by the MADR's own source-naming convention, the MADR annotated where it names none.

**Tests by layer.** Three of Phase 2's tests assert the resend or a provider's request in their definitions, and are built in layer B, which builds those: `TestLimitRefusal_WindowForms_0089` (DL35), `TestLimitRefusal_CallerLimit_0089` (DL37) and `TestLimitMemo_Concurrent_0089` (DL43). DL67's plant (f), "N compared with the pre-raise value", is a provider's choice of the value it gives the validator, and is planted in layer B; layer A's validator case pins that the value given is the one compared.

### 2026-11-23: the merged phase, layer A (checkpoint)

Base `6ace750`, on top of the stopped pass's unstaged work, which is kept. No live call; every test serves its own values. Not a build pass (deviation 7); pass count 1.

**Re-grounding.** `tools/claims.py --rev self=6ace750`: 0089-MADR 42 of 49, 0088-MADR 127 of 140, each exit 1, the same rows the MADRs' 2026-11-23 annotations already record moved; none new, none gone. On the tree: 49 of 49 and 140 of 140, exit 0.

**The code.** `llmprovider/limit_observation.go`: `LimitObservation`; `WithLimitObservations`, a `SharedOption` whose value is `[]LimitObservation`, refused by `ResolveOptions` (so by every `New`) with `ErrInvalidRequest` for no model, Tokens below 1 or a zero Seen. `llmprovider/warning.go`: `WarningLimitLearned`, `Warning.Observed` (`omitzero`). `llmprovider/internal/limitmemo` (new, standard library only): `Memo` with `Keep`, `Cap` and `Observe` (the drop of DL39), `Key` (provider, base URL, scope, model, stream), `Bound` 256, oldest dropped, every method under one mutex, a nil `Memo` the process's. `internal/cacheref`: `Value.Limits` and `LimitMemo`, one memo per `catalog.Cache`. `internal/wire/limit_refusal.go`: `ReadLimitRefusal`, `LimitForms` (three cap forms, three window forms), `LimitWire` and the six routes' values by DL68, `LimitAsk`, `LimitReading`; V1 to V6, V6's canonical tokens by `catalog`'s rule, copied since `wire` cannot import `catalog`. `catalog`: the observations read from `Settings.Values()`; `rememberedEvidence`, source "remembered", ranked below every registry stage, used where the model (lower case), base URL and scope (Kilo organization, Claude workspace) match and Seen is within 24 hours (`lastGoodWindow`).

Choices within the rows, for review: an expired or other-scope observation is not evidence at all; the memo's `Keep` reports a change for a new key or a new value (layer B decides the warning's "once per key" from it); a window form's input is the stated input (the split) or the stated total less `Sent` (C39, C97), and V2 applies to the split's completion; window readings are built for V1 and layer B's DL35, whose test pins them.

**Red.** Every layer-A test uses API absent at `6ace750` (`TestDiscover_Offline_0089` included, through its new case), so none compiles there; each takes its red from its plants, as Phase 1 recorded.

**Green.** `python3 -B tools/redgreen.py SPEC.json --expect pass` on a scratch clone: `redgreen: 12 of 12 pass as expected` (the ten row tests below and the new package's `TestMemo_KeyAndValue` and `TestMemo_ConcurrentUse`; `cacheref`'s `TestLimitMemo_0089`).

**Plants**, each by `python3 -B tools/plant.py` on a scratch clone, each caught by a failing test (three first drafts broke the build by leaving an import or variable unused; they were rewritten to compile and rerun, and a build failure was not counted):

| Plant | Test that fails | Its failing line |
| :--- | :--- | :--- |
| DL31 (a) the form not recognised | `TestLimitRefusal_Validator_0089` | `limit_refusal_0089_test.go:35: case 1: {Form:0 Cap:0 ...}, false; want a kept cap of 64000` |
| DL32 (b) the double match read as a window | `TestLimitRefusal_Validator_0089` | `limit_refusal_0089_test.go:42: case 2: {Form:8 Cap:0 Window:200000 Input:22000 Keep:false}; want a text matching a cap form and a window form rejected (V1)` |
| DL33 (c) N not compared | `TestLimitRefusal_Validator_0089` | `limit_refusal_0089_test.go:50: case 3: {Form:1 Cap:64000 ...}; want N 100000 against a wire value of 128000 rejected (V2)` |
| DL34 (d) M at N accepted | `TestLimitRefusal_Validator_0089` | `limit_refusal_0089_test.go:55: case 4: {Form:1 Cap:64000 ...}; want M at N rejected (V3)` |
| DL34 (e) the minimum not applied | `TestLimitRefusal_Validator_0089` | `limit_refusal_0089_test.go:62: case 5: {Form:2 Cap:8 ...}; want a cap of 8 below the minimum 16 rejected (V4)` |
| DL68 OpenAI's form read on Responses | `TestLimitRefusal_FormsByWire_0089` | `limit_refusal_0089_test.go:99: OpenAI's form on Responses: {Form:2 Cap:4096 ...}; want not read` |
| DL36 (a) an exact-only compare | `TestLimitRefusal_NamedModel_0089` | `limit_refusal_0089_test.go:155: Claude's alias, refused naming its resolved id: Keep = false; want true` |
| DL36 (b) any name kept | `TestLimitRefusal_NamedModel_0089` | `limit_refusal_0089_test.go:155: Kilo's auto route, refused naming a Claude id: Keep = true; want false` |
| DL38 (a) one memo for every cache | `TestLimitMemo_Scope_0089` | `limit_memo_0089_test.go:34: the second cache's memo holds the first's cap 4096; want one memo per cache` |
| DL38 (b) stream and plain merged | `TestLimitMemo_Scope_0089` | `limit_memo_0089_test.go:54: a streamed and a plain Gemini request: ... reads ...'s cap 65536; want each apart` |
| DL38 no bound | `TestLimitMemo_Bound_0089` | `limitmemo_0089_test.go:22: after 257 keys, the first is still kept; want it dropped` |
| DL39 kept | `TestLimitMemo_SelfHeal_0089` | `limitmemo_0089_test.go:47: Observe(10000) over a cap of 4096 reported no drop` |
| DL40 (a) no expiry | `TestLimitObservations_PassBack_0089` | `limit_observations_0089_test.go:48: seen 25 hours ago: output {Tokens:30000 Known:true ... Source:remembered ...}; want unknown` |
| DL40 (b) the scope ignored | `TestLimitObservations_PassBack_0089` | `limit_observations_0089_test.go:62: another Kilo organization: output {Tokens:64000 ... {Source:remembered ...` |
| DL40 (d) the base URL ignored (added within the row) | `TestLimitObservations_PassBack_0089` | `limit_observations_0089_test.go:53: another base URL: output {Tokens:30000 ... Source:remembered ...}; want unknown` |
| DL41 (c) the remembered value chosen over the registry | `TestLimitObservations_PassBack_0089` | `limit_observations_0089_test.go:76: a registry fact beside a passed-back cap: output {Tokens:30000 ... Source:remembered ...` |
| DL42 accepted | `TestLimitObservations_Invalid_0089` | `limit_observations_0089_test.go:100: no model: ResolveOptions err = <nil>; want ErrInvalidRequest` |
| DL24 the passed-back cap dropped offline | `TestDiscover_Offline_0089` | `limits_0089_test.go:684: offline output, passed back = {Tokens:20000 ... }` (no remembered evidence) |
| (the new package's own) `Keep`'s lock removed | `TestMemo_ConcurrentUse`, `-race`, through `run-heavy` | `WARNING: DATA RACE` (eleven), then `fatal error: concurrent map read and map write` |

**Checks**, each exit status taken before any filter:

| Command | Host | Exit |
| :--- | :--- | :--- |
| `go build ./...`; `go vet ./...` | session (darwin/arm64) | 0; 0 |
| `go test -count=1` on `llmprovider`, `catalog`, `internal/wire`, `internal/limitmemo`, `internal/cacheref` | session (darwin/arm64) | 0 (coverage 98.7, 96.0, 94.2, 97.6, 100.0 %) |
| `run-heavy -- go test -count=1 ./...` | session (darwin/arm64) | 0, 46 packages ok |
| `run-on-host --need linux --race -- go test -race -count=1` (the five packages) | a fleet Linux host (linux/amd64), not WSL | 0 |
| `make tables-check dep-check api-check` | session | 0 each |
| `make records-check` | session | 2: only the two untracked 0090 records, not indexed, and the count they raise (another agent's; not touched) |
| `make markdownlint links-check parity-check docs-check` | session | 0 (a first run exited 2 on this layer's R2 row, unindented in its list, and a long citation line; both fixed) |
| `golangci-lint run -c .golangci.yml --build-tags live_gateways` (the five packages) | session | 0 |
| `python3 -B tools/claims.py` on both MADRs | session | 0, 0 (49 of 49; 140 of 140) |
| `python3 -B tools/recordlint.py --changed` | session | 1: one MD034 in the 0090 MADR (another agent's); none in 0088's or 0089's records |
| `python3 -B tools/denyscan.py --changed` | session | 0, 0 hits in 28 files |

**`apidiff`** (`python3 -B tools/apidiff.py v1.32.0`), whole:

```text
Compatible changes:
- ./llmprovider.(*Settings).MaxTokensGiven: added
- ./llmprovider.LimitObservation: added
- ./llmprovider.Warning.Observed: added
- ./llmprovider.WarningLimitLearned: added
- ./llmprovider.WithLimitObservations: added
- ./llmprovider/catalog.CachedLimits: added
- ./llmprovider/catalog.Evidence: added
- ./llmprovider/catalog.Fact: added
- ./llmprovider/catalog.Limits: added
- ./llmprovider/catalog.LookupLimits: added
```

Phase 1's five, `Settings.MaxTokensGiven` and layer A's four, and nothing else.

**Docs.** Each new export's doc comment (R41, R51, R52); `docs/architecture.md`'s layout line and rows for `internal/limitmemo` (new), `internal/cacheref`, `internal/wire` and `catalog`; `docs/guides/api-standards.md` R2's table (`internal/limitmemo`'s row; `internal/cacheref` may import it) and its citation. `WithLimitObservations`' doc names what `catalog` reads; a request's use of a passed-back cap is layer B's, which extends it. Not done here: `CHANGELOG.md`, R6, R7, R57 and the package docs, which step 5 does in layer E.

**Not done in layer A, by the table "Layers":** DL35, DL37 and DL43's tests, DL67's plant (f), the memo's caps in `Limits`, the wiring of each response's output to `Observe` and of the resolved id to V6: layer B. Nothing staged or committed.

### 2026-11-23: the merged phase, layer B (stopped after its code)

Base `77563af` (the owner's docs-only commit of the 0090 records over `6ace750`; the code is identical), on top of layer A's unstaged work, which is kept. No live call. Not a build pass (deviation 7); pass count 1.

**Recorded first.** The owner's ruling "Add a stream dimension", relayed by the coordinator: 0089-MADR's Owner rulings table and DL76 (`Limits.StreamedOutput`), its surface sketch, and the export list above (no `apidiff` line: `Limits` is a new type). DL68's window column confirmed against the code (0088 C38, C39, C97, ST51) and annotated in the MADR; the reading stands. The evidence name of a cap the memo keeps, "observed", added to the MADR's `Evidence` sketch (deviation 7's design note).

**Built, unstaged, untested beyond the existing suites.** `llmprovider/sent_limit.go` (`SentLimit`, `LimitBasis` and its six constants, `WithoutLimitLearning`, whose value is `limitmemo.Off`); `Response.OutputLimit` (`llmprovider/item.go`); `llmprovider/internal/wire/limit_rule.go` (`ChooseLimit`, the request rule in plain values; `LimitRetry`, the resend once at a validated cap or window room, sticky, its send not counted toward `MaxSends`, through `Retries`, `Send` and `StreamEventsRetry`); `internal/limitmemo` (each cap's time, for its evidence's `Seen`; per-model notes the catalog writes and a provider reads: the resolved id (V6) and the route's fallback (DL51)); `catalog` (`Limits.StreamedOutput`; a kept cap as "observed" evidence that bounds the chosen output; the route's fallback, the smallest current limit on the messages route, from the cached listing, else the document, else the extract, the retired list excluded; the resolved id kept by the one-model read; a base URL equal to the provider's default read as empty, so that a passed-back observation and a kept cap match a provider's own request); Claude, Kilo and OpenCode's messages route wired to them.

**Stopped** at the first test run, on two findings, put to the owner by the coordinator; no new test, plant, gate or docs row was run or written:

* **B1, an existing test no row names.** `TestThinkingWire_Claude` (`llmprovider/providers/claude/claude_test.go:336`), its budget case `claude-sonnet-5` with budget 9000, construction and request: `claude_test.go:369: max_tokens = 128000, want 13096`. The 13096 is today's 8192 default raised above the budget (0088 ST30); under 0088 ST2 and 0089 DL55 `AddThinkingFor` starts from the chosen fact, here the extract's 128,000. No row names the test.
* **B2, ST100's goldens fail in layer B.** Layer B's routes change 60 G-wire goldens (claude 12, claude-refuses-forced 12, kilo 12, opencode-zen-messages 12, opencode-go-messages 12); each differs only at `requests[0].body.max_tokens` (8192 to 128,000, or 131,072 on the Go messages case). ST100 names them, but the Layers table builds ST100 in layer C, so layer B's `go test ./...` gate cannot pass as written.

`TestOutputLimit_Facts_0041`'s three cases on B's routes (claude, opencode-zen `claude-sonnet-5`, kilo) fail as ST32 plans. `run-heavy -- go test -count=1 ./...` (session host): exit 1, 42 of 46 packages ok, the four failing (`providers`, `providers/claude`, `providers/kilo`, `providers/opencode`) only on the tests above; `go build ./...` and `go vet ./...` exit 0.

### 2026-11-23: deviation 8, layers B and C merged; the owner's rulings on B1 and the OpenCode fallback

The owner's rulings on layer B's stop, relayed by the coordinator:

* **B2: "Merge layers B and C".** Layer B's routes change 60 of ST100's goldens, which the Layers table built in layer C, so layer B's `go test ./...` gate could not pass as written. Layers B and C are built as one layer, one run, with one checkpoint (the Layers table annotated). A merged layer is not a build pass; the merged phase's pass count stays 1.
* **B1: first "Name it; add a caller layer (Recommended)", then, corrected, "Explicit WithMaxTokens(8192) (Recommended)".** The next run stopped before any write: a caller layer stating 8,192 cannot keep 13,096 pinned. Its probe on a scratch copy (`go test -run TestB1Probe -v`, exit 0): `layer: max_tokens 8192 thinking map[budget_tokens:8191 type:enabled]`; `option: max_tokens 13096 thinking map[budget_tokens:9000 type:enabled]`. The coordinator's question had wrongly described a caller layer; the owner re-ruled. `TestThinkingWire_Claude`'s budget case gets the caller's `WithMaxTokens(8192)`; 0088-MADR's ST101 names it.
* **The OpenCode fallback: "Models on opencode's route table (Recommended)".** Layer B's code took the fallback on OpenCode's messages route from the extract's whole section, which has no "current" data, and would pick `claude-3-5-haiku`'s 8,192 on Zen. The fallback is the smallest limit among the models the shared route table lists for the route: 0089-MADR DL77, its test `TestOutputLimit_Fallback_0089`'s OpenCode case.
* **Together's DL35 case**, confirmed by the coordinator: a caller's `WithMaxTokens(16384)` refused in the split form (Phase 2's table annotated).
* **Decided by the coordinator from evidence**, each annotated in the MADR: a refused `WithMaxTokens` value is kept, not resent (DL37); layer A's pass-back base-URL match, the default read as empty on both sides, stands, with a test (DL40); the memo's notes stay internal (D5); V6's copied canonical-token rule stays a review point (this checkpoint below).

### 2026-11-23: the merged layer B+C (stopped at its first test run)

Base `77563af`, on top of the unstaged work above, which is kept. No live call. Not a build pass (deviation 7); pass count 1.

**Built, unstaged, untested beyond the existing suites.** The records of deviation 8 above, 0089-MADR's DL77 and its annotations, 0088-MADR's seventh amendment (ST101). The OpenCode fallback by DL77 (`routeFallback` counts, on an OpenCode gateway, only the models the shared route table routes to messages). Layer C's wiring: `wire.LimitRetry.Optional` and `wire.DropLimit`; a `limit.go` in `gemini`, `grok`, `openai`, `together`, `huggingface`, `ollama` and `chatcompat` (the request rule on an optional wire, the report, the memo's keep and drop, the listing probe's 8192 given); OpenCode's responses, chat and google routes; each package doc's output-limit line.

**Stopped** at the first `go test ./llmprovider/...` (exit 1), on an existing test no row names: `TestTogether_RequestShapes` (`llmprovider/providers/together/together_test.go:83`, every case, at construction and on the request) asserts `body["max_tokens"] == 8192` on a request that names no limit, which DL47 and 0088 ST86 now send none of: `together_test.go:84: max_tokens <nil>, model openai/gpt-oss-120b`. The other failures are the planned ones: `TestOutputLimit_Facts_0041` (ST32), `TestThinkingWire_Claude`'s budget case (ST101) and each package's `TestWireGoldens` (ST100). On a scratch copy, the goldens regenerated with `-update` change 234 files, each only in its output-limit field (72 `max_tokens`, 76 `max_output_tokens`, 14 `generation_config.max_output_tokens` and 12 `generationConfig.maxOutputTokens` removed; 48 `max_tokens` 8192 to 128,000 and 12 to 131,072); the other 7 of ST100's 241 are listing goldens whose probe sends a given 8192 (ST26), unchanged. No new test, plant or gate was run.

### 2026-11-23: the merged layer B+C (checkpoint)

Base `77563af` (code identical to `6ace750`), on top of the unstaged work above, which is kept. No live call; every test serves its own replies. Not a build pass (deviations 7 and 8); pass count 1.

**Recorded first.** The owner's ruling on the stop above, relayed by the coordinator, "One class row, assert absent (Recommended)": 0088-MADR's eighth amendment, ST102, names the existing tests and goldens that pin the old 8,192 default on a route that now omits it, its one member `TestTogether_RequestShapes` (no other member was found: the full suite below is green); the 0088 table above gained its row.

**Tests**, each with its rows; files new unless named:

| Test | Rows | File |
| :--- | :--- | :--- |
| `TestLimitRefusal_WindowForms_0089` | DL35, V5 | `llmprovider/providers/limit_refusal_0089_test.go` |
| `TestLimitRefusal_CallerLimit_0089` | DL37 (both a request's value and `WithMaxTokens`') | the same |
| `TestLimitMemo_Concurrent_0089` | DL43 | the same |
| `TestLimitLearning_Default_0089` | DL49, DL73 | the same |
| `TestOutputLimit_Omitted_0089` | DL46 to DL48, DL65's interim, DL69 | `llmprovider/providers/output_limit_0089_test.go` |
| `TestOutputLimit_Reported_0089` | DL44, DL53, DL54; the basis of DL46 and DL47 | the same |
| `TestOutputLimit_BudgetReported_0089` | DL55, DL67's plant (f) | the same |
| `TestOutputLimit_OutsideProvider_0089` | DL57 | the same |
| `TestOutputLimit_Default_0089` | DL45 | `llmprovider/providers/openai/output_limit_0089_test.go` (the MADR's DL45 annotated: the openai package's helpers sign in the sessions) |
| `TestOutputLimit_Fallback_0089` | DL50, DL51, DL77 | `llmprovider/catalog/output_limit_fallback_0089_test.go` (`catalog_test`), with the hook `ReplaceExtract` in `export_0089_test.go` |
| `TestLimitMemo_StreamedOutput_0089` | DL76 | `llmprovider/catalog/limit_memo_0089_test.go` (added) |
| `TestLimitObservations_PassBack_0089`, its default-base case | DL40 (the coordinator's annotation) | `llmprovider/limit_observations_0089_test.go` (added) |
| `TestOutputLimit_Facts_0041` (existing, changed) | ST32 with ST99's values | `llmprovider/providers/facts_0041_test.go` |
| `TestThinkingWire_Claude` (existing, changed: the budget case's `WithMaxTokens(8192)`) | ST101 | `llmprovider/providers/claude/claude_test.go` |
| `TestTogether_RequestShapes` (existing, changed: no limit sent) | ST102 | `llmprovider/providers/together/together_test.go` |
| `TestWireGoldens` (existing; the scenario `explicit-8192` added to `llmprovider/internal/wirecase/wirecase.go`) | ST100, ST16 | each provider package's `testdata/wire` |

**Red.** `python3 -B tools/redgreen.py SPEC.json --expect fail --rev 6ace750`, the three files copied in and the three new-API functions of `output_limit_0089_test.go` dropped: `redgreen: 3 of 3 fail as expected`, each on its assertion: `output_limit_0089_test.go:143: claude, a listing stating 128,000: sent 8192 (found true); want 128000` (and the same for OpenCode's messages route, Kilo, and each of the ten optional wires: `sent 8192; want none`); `facts_0041_test.go:102: claude/claude-sonnet-5, a limit of 128000: sent 8192 (found true); want 128000`; `together_test.go:86: max_tokens 8192 (sent true; want none), model openai/gpt-oss-120b`. Every other test above uses API absent at the base, so it takes its red from its plants, as Phase 1 recorded; `TestThinkingWire_Claude`'s changed case passes at the base (a given 8192 raised to 13,096 there too), so its plant is its red, as ST101 says.

**Green.** `python3 -B tools/redgreen.py SPEC.json --expect pass`: `redgreen: 25 of 25 pass as expected` (the tests above and each provider package's `TestWireGoldens`).

**Plants**, each by `python3 -B tools/plant.py` on a scratch clone, each caught by a failing test, none by a build failure (providers spec: `all 21 plant(s) caught`; nine of them again with only their own row's test run, each `all 1 plant(s) caught`; catalog: `all 7`; the pass-back: `all 1`; the race plant through `run-heavy`: `all 1`):

| Plant | Test that fails | Its failing line |
| :--- | :--- | :--- |
| DL35 the window kept in the memo | `TestLimitRefusal_WindowForms_0089` | `limit_refusal_0089_test.go:138: kilo, OpenRouter's form: the memo holds 9152; want it empty after a window form` |
| DL35 V5 removed (a window not above the input it states read) | `TestLimitRefusal_WindowForms_0089` | `limit_refusal_0089_test.go:111: kilo, a stated total below the limit sent (V5): 2 send(s), err <nil>; want one send and the refusal returned` |
| DL37 a caller's limit resent | `TestLimitRefusal_CallerLimit_0089` | `limit_refusal_0089_test.go:174: the request's MaxOutputTokens: 2 send(s), err <nil>; want one send and ErrInvalidRequest` (and `WithMaxTokens:` the same) |
| DL43 the memo's lock removed (`-race`) | `TestLimitMemo_Concurrent_0089` | `race summary: 8 'WARNING: DATA RACE'; ... ['--- FAIL: TestLimitMemo_Concurrent_0089']; memo methods in the stacks ['Entry', 'Keep', 'Observe']` |
| DL67 (f) the provider passing the pre-raise value (OpenCode's messages route) | `TestOutputLimit_BudgetReported_0089` | `output_limit_0089_test.go:401: opencode-zen messages, refused stating 44096 > 32000: output {Tokens:128000 ... models.dev (bundled 2026-10-12) ...}` (nothing kept) |
| DL55 the pre-raise value reported (Claude) | `TestOutputLimit_BudgetReported_0089` | `output_limit_0089_test.go:391: claude: OutputLimit = ...` (Tokens 32000, not 44096) |
| DL44 (a) basis fact for a caller's value | `TestOutputLimit_Reported_0089` | `output_limit_0089_test.go:288: WithMaxTokens(4000): OutputLimit = {Tokens:4000 Sent:true Basis:fact Source:}; want {Tokens:4000 Sent:true Basis:option Source:}` |
| DL54 (b) zero on a stream | `TestOutputLimit_Reported_0089` | `output_limit_0089_test.go:309: the stream's final response: ...; want OutputLimit {Tokens:5000 Sent:true Basis:request Source:}` |
| DL53 the probe's 8192 not given | `TestOutputLimit_Reported_0089` | `output_limit_0089_test.go:348: a probe carried 128000 (found true); want 8192, given` |
| DL46 (b) Claude omitted | `TestOutputLimit_Omitted_0089` | `output_limit_0089_test.go:143: claude, a listing stating 128,000: sent 0 (found true); want 128000` |
| DL47 (a) a fact sent on Gemini | `TestOutputLimit_Omitted_0089`, `TestOutputLimit_Facts_0041` | `facts_0041_test.go:104: gemini/gemini-3.8-flash, a limit of 128000: sent 128000; want none (0089-MADR DL47, DL48)` |
| DL48 (c) a caller layer sent on Ollama | `TestOutputLimit_Omitted_0089`, `TestOutputLimit_Facts_0041` | `facts_0041_test.go:104: ollama/llama3.3:latest, a limit of 128000: sent 128000; want none (0089-MADR DL47, DL48)` |
| DL47 (d) the caller's value dropped on an optional wire | `TestOutputLimit_Omitted_0089` | `output_limit_0089_test.go:219: gemini, WithMaxTokens(5000): sent 0 (found false); want 5000` |
| DL65 Kilo omitted before the verdict | `TestOutputLimit_Omitted_0089` | `output_limit_0089_test.go:171: kilo, a layer stating 128,000: sent 0 (found false); want 128000 (DL65's interim)` |
| DL57 the chatcompat id refused | `TestOutputLimit_OutsideProvider_0089` | `output_limit_0089_test.go:418: CachedLimits for a chatcompat id: output {Tokens:0 Known:false ...}; want 20000 from the layer` |
| DL45 a field on the ChatGPT routes | `TestOutputLimit_Default_0089` | `output_limit_0089_test.go:42: the ChatGPT backend, a caller's value false: OutputLimit = {Tokens:8192 Sent:true Basis:option Source:}; want {Tokens:0 Sent:false Basis:none Source:}` |
| DL49, DL73 (a) the memory off by default | `TestLimitLearning_Default_0089` | `limit_refusal_0089_test.go:293: by default: warning <nil>, output {Tokens:128000 ...}; want 64000 kept and reported` |
| DL73 (b) the option ignored | `TestLimitLearning_Default_0089` | `limit_refusal_0089_test.go:295: WithoutLimitLearning: warning &{Kind:limit_learned ...}` |
| DL49 the resend dropped under the option | `TestLimitLearning_Default_0089` | `limit_refusal_0089_test.go:278: learning off true: 1 send(s), err ... max_tokens: 128000 > 64000 ...; want a resend` |
| DL76 the streamed cap read into Output | `TestLimitMemo_StreamedOutput_0089` | `limit_memo_0089_test.go:84: a streamed cap of 32768: output {Tokens:32768 ... Source:observed ...}` |
| DL51 (a) the largest current limit chosen | `TestOutputLimit_Fallback_0089` | `output_limit_fallback_0089_test.go:148: the fallback: sent [128000]; want [64000]` |
| DL51 (b) retired models counted | `TestOutputLimit_Fallback_0089` | `output_limit_fallback_0089_test.go:148: the fallback: sent [32000]; want [64000]` |
| DL50 (c) a wait for the listing (`LookupLimits` in the request) | `TestOutputLimit_Fallback_0089` | `output_limit_fallback_0089_test.go:106: a fresh cache: sent [128000], 2 listing request(s), err <nil>; want 128000 from the extract, no listing request` |
| DL51 (d) 8192 sent where no source states a limit | `TestOutputLimit_Fallback_0089` | `output_limit_fallback_0089_test.go:195: no source at all: 1 request(s), err <nil>; want none sent and ErrInvalidRequest` |
| DL51 (e) the refusal's message without the fix | `TestOutputLimit_Fallback_0089` | `output_limit_fallback_0089_test.go:198: the refusal's message "...: state the model's limit, ..."; want it to name maxTokens and WithModelLayer` |
| DL77 the extract's whole section read | `TestOutputLimit_Fallback_0089` | `output_limit_fallback_0089_test.go:180: opencode zen messages, no source: sent [8192], err <nil>; want 64000, basis fallback` |
| DL40 the default base URL not read as empty | `TestLimitObservations_PassBack_0089` | `limit_observations_0089_test.go:67: the default passed back, none looked up: output {Tokens:0 Known:false ...}; want the remembered 30000` |
| ST32 under ST99, a 2026-11-21 value kept (8192 on a route that sends one) | `TestOutputLimit_Facts_0041` | `facts_0041_test.go:102: claude/claude-sonnet-5, a limit of 128000: sent 8192 (found true); want 128000` |
| ST102 8192 sent on Together | `TestTogether_RequestShapes` | `together_test.go:86: max_tokens 8192 (sent true; want none), model openai/gpt-oss-120b` |
| ST101 `WithMaxTokens(8192)` not counted as given | `TestThinkingWire_Claude` | `claude_test.go:376: max_tokens = 128000, want 13096` |
| ST100 the same plant, on the explicit goldens | `TestWireGoldens`, `explicit-8192` | `explicit-8192 goldens failing: 19` (every case but `chatgpt` and `chatgpt-plan`, which send no limit whatever the caller gives, DL45, pinned by `TestOutputLimit_Default_0089`); e.g. `testdata/wire/chatcompat/explicit-8192.json: ... requests[0].body.max_tokens: want 8192, got (absent)` |

DL55's and DL67 (f)'s plants change the one datum both rows read, `LimitRetry.Sent`, the value the report names and V2 compares; each is planted on its own Messages route (Claude; OpenCode's messages route), and each fails both its report and its refusal case there.

**ST100's goldens.** The diff-check script (`golden_diffcheck.py`, scratch; allowed: a request body's `max_tokens`, `max_output_tokens`, `generation_config.max_output_tokens` or `generationConfig.maxOutputTokens` removed where it was 8192, or its 8192 changed to another number; ST88's stream form is layer D's and not allowed here; a new golden only an `explicit-8192.json` carrying 8192 or none) was first seen failing on a scratch copy of the goldens at `6ace750`: unchanged, exit 0; `max_tokens` removed, exit 0; plus a body's `model` changed, exit 1, `FINDING llmprovider/providers/claude/testdata/wire/claude/text.json: requests/0/body/model: 'claude-sonnet-5' -> 'claude-planted-0089'`, the allowed change not reported; 8192 to 9000, exit 0. Then `go test -run '^TestWireGoldens$' -update` on the ten provider packages (exit 0), the run again without `-update` (exit 0), and the script over the tree: exit 0, `diffcheck: 0 finding(s)`:

| Provider | Changed | Unchanged | New (`explicit-8192`) |
| :--- | ---: | ---: | ---: |
| chatcompat | 12 | 1 | 1 |
| claude | 24 | 2 | 2 |
| gemini | 14 | 0 | 1 |
| grok | 26 | 2 | 2 |
| huggingface | 12 | 1 | 1 |
| kilo | 12 | 1 | 1 |
| ollama | 12 | 1 | 1 |
| openai | 26 | 30 | 4 |
| opencode | 84 | 7 | 7 |
| together | 12 | 1 | 1 |
| total | 234 | 46 | 21 |

The 234 are the 241 of ST100 less the 7 listing goldens whose probe sends a given 8192 (ST26). The 21 explicit goldens carry `WithMaxTokens(8192)`'s 8192 on 19 cases and none on `chatgpt` and `chatgpt-plan` (DL45).

**Checks**, each exit status taken before any filter:

| Command | Host | Exit |
| :--- | :--- | :--- |
| `go build ./...`; `go vet ./...`; `go vet -tags live_gateways ./...` | session (darwin/arm64) | 0; 0; 0 |
| `run-heavy -- go test -count=1 ./...` | session (darwin/arm64) | 0, 46 packages ok |
| `run-on-host --need linux --race -- go test -race -count=1` (`llmprovider`, `catalog`, `internal/wire`, `internal/limitmemo`, `internal/cacheref`, `providers/...`) | a fleet Linux host (linux/amd64), not WSL | 0, 16 packages ok |
| `make tables-check dep-check api-check records-check markdownlint links-check docs-check fixtures-check` | session | 0 each; tables-check first exited 2 (the rows below missing), docs-check first exited 2 (R51: `LimitBasis`'s doc said nothing of concurrent use), each fixed in the docs step |
| `python3 -B tools/apidiff.py v1.32.0` | session | 0, the twenty lines below |
| `python3 -B tools/claims.py` on both MADRs | session | 0, 0 (49 of 49; 140 of 140) |
| `python3 -B tools/recordlint.py --changed`; `python3 -B tools/denyscan.py --changed` | session | 0 (no rule rose); 0 (0 hits) |

**`apidiff`** (`python3 -B tools/apidiff.py v1.32.0`), whole:

```text
Compatible changes:
- ./llmprovider.(*Settings).MaxTokensGiven: added
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
- ./llmprovider.WithLimitObservations: added
- ./llmprovider.WithoutLimitLearning: added
- ./llmprovider/catalog.CachedLimits: added
- ./llmprovider/catalog.Evidence: added
- ./llmprovider/catalog.Fact: added
- ./llmprovider/catalog.Limits: added
- ./llmprovider/catalog.LookupLimits: added
```

Phase 1's five and the merged phase's fifteen ("Exports" above), and nothing else; `Limits.StreamedOutput` adds no line (DL76).

**Docs.** `docs/guides/api-standards.md` R2: `llmprovider` may import `internal/limitmemo` (`WithoutLimitLearning`'s value), `catalog` `internal/limitmemo` and `internal/retired` (DL51's current models), with the citation; `docs/architecture.md`: the import column of `llmprovider`, `catalog` and the ten provider rows (`internal/limitmemo`; `catalog`'s `internal/retired`), and the holds of `llmprovider` (`SentLimit`, `LimitBasis`, `Response.OutputLimit`, `WithoutLimitLearning`), `internal/wire` (`ChooseLimit`, `LimitInput`, `LimitRetry`, `Optional`, `DropLimit`), `internal/limitmemo` (the notes, `Off`) and `catalog` (the observed cap, `StreamedOutput`, the fallback); `WithLimitObservations`' doc says what a provider's request reads of a passed-back cap and that the default base URL reads as empty; `LimitBasis`' doc says it is safe for concurrent use (R51). Each package doc's output-limit line was written with the code, before this checkpoint. Not done here, by the Layers table: `CHANGELOG.md`, R6, R7, R57, `README.md` and `docs/guides/adding-a-provider.md`, which step 5 does in layer E.

**Not done in layer B+C, by the Layers table:** layer D (ST88, `Generate` over the stream, and ST88e's test changes) and layer E (0088's remaining tests, the docs above, every gate of step 6, including `make pre-add-check`, `make lint` and `tools/gate.py`). Nothing staged or committed.

### 2026-11-23: the golden diff-check moves into `tools/` (the coordinator's decision)

Decided by the coordinator from the repository's rule (AGENTS.md "Tools": a script worth running twice belongs in `tools/`, with a test in `tools/test_tools.py`): the merged layer B+C's scratch diff-check moves into `tools/goldendiff.py`, with `TestGoldenDiff` in `tools/test_tools.py` and a row in `tools/README.md`, and gains the allowance ST88 needs (`--stream`): a golden's plain request becoming the stream form (`"stream": true`, `"stream_options": {"include_usage": true}`, `Accept: text/event-stream`, a `:generateContent` path becoming `:streamGenerateContent` with `alt=sse`); a new golden is an `explicit-8192.json`, plain unless its case's `text.json` already streamed at the revision (the ChatGPT routes). Without `--stream` it is layer B+C's check. Not a build pass.

### 2026-11-23: the merged phase, layer D (checkpoint)

Base `77563af` (code identical to `6ace750`), on top of the unstaged work above, which is kept. No live call; every test serves its own replies. Not a build pass (deviations 7 and 8); pass count 1.

**The tool.** `TestGoldenDiff`, seven cases (`make tools-test`'s command, `-k TestGoldenDiff`: `Ran 7 tests`, `OK`). Its plants, by `python3 -B tools/plant.py` on a scratch clone with that command, `all 4 plant(s) caught`: any request-body field read as a limit (`test_an_unrelated_change_is_a_finding`: `AssertionError: 0 != 1`); any value added under a stream key (`test_a_change_that_is_not_the_stream_form_is_a_finding [a stream that is not true]`: `AssertionError: 0 != 1`); the stream form allowed without `--stream` (`test_the_stream_form_needs_the_flag`); an explicit golden in the stream form accepted (`test_a_new_golden_is_an_explicit_one_kept_plain`). On the tree before this layer's goldens, `tools/goldendiff.py --rev 6ace750`: exit 0, layer B+C's table. On a scratch copy of this layer's goldens, `--stream`: as they are, exit 0; a body's `model` changed (an unrelated change), exit 1, `FINDING llmprovider/providers/claude/testdata/wire/claude/text.json: requests/0/body/model: 'claude-sonnet-5' -> 'claude-planted-0088'`; `"stream"` made the text `"true"` (not the stream form), exit 1, `FINDING .../claude/text.json: requests/0/body/stream: <absent> -> 'true'`; `stream_options`' `include_usage` made false, exit 1, `FINDING .../kilo/text.json: requests/0/body/stream_options: <absent> -> {'include_usage': False}`; restored, exit 0.

**The code.** `llmprovider/internal/wire/overstream.go` (new): `OverStream`, true when neither the request's `MaxOutputTokens` nor a given option (`Settings.MaxTokensGiven`, or a listing probe's 8192) names the limit; `Collect`, `Generate`'s response read from the provider's own `Stream`: a failure before the first event returned as the stream classified it, one after it marked `transport.AfterEvent` (new, `internal/transport/reply.go`), which `llmprovider`'s `retryable` does not retry (ST88a); the text and reasoning deltas and the calls' arguments bounded by `ReplyLimit`, `ErrIncomplete` past it (ST88b); `EventDone`'s response, its usage as the stream reported it (ST88c). `StreamEventsRetry` (`internal/wire/sse.go`) checks a stream that failed before yielding any event as it checks a failed opening, so an error inside a 200 as the first payload is resent while a retry check says so (0088-MADR A5; ST88d). Each provider's `Generate` reads over its `Stream` when `OverStream` holds: Claude, Gemini, Grok, Together, Hugging Face, Ollama, chatcompat, Kilo, OpenAI with a key (a ChatGPT session's request is streamed always and read as before), and OpenCode on its four routes (a route with no stream of its own stays plain, so its `Stream`'s fallback to `Generate` cannot loop).

Choices within the rows, for review:

* A5's first-payload resend, which 0088's table tests as ST55 in layer E (`TestStreamEventsRetry_FirstPayloadError_0088`, Kilo's `Stream`), is built here, since ST88d's resend is that path: `StreamEventsRetry` serves both. ST55's own test stays in layer E.
* ST88b's "assembled answer" is the text and reasoning deltas and the calls' arguments.
* A `Generate` read over the stream is a streamed request: a cap it keeps is under the memo's streamed key (DL38) and bounds `Limits.StreamedOutput` (DL76), not `Output`.
* Gemini's listing probe keeps the provider's limit, not given, so it is read over the stream; Gemini's listing golden takes the stream form.

**Tests (ST88, ST88a to ST88d)**, in `llmprovider/providers/generate_stream_0088_test.go` (new):

| Test | Rows | What it does |
| :--- | :--- | :--- |
| `TestGenerateOverStream_0088` | ST88, ST88e | Claude with a stated 128,000: the stream, at 128,000, the whole response; OpenAI with a key: the stream, no limit; `WithMaxTokens(4000)`, and the request's `MaxOutputTokens` 3000: plain |
| `TestGenerateOverStream_Retry_0088` | ST88a | under `WithRetry`: a 503 at the opening retried, both sends streamed; a stream cut after a text event: one send, the failure returned |
| `TestGenerateOverStream_ReplyLimit_0088` | ST88b | 17 text events of 1 MiB: `ErrIncomplete`, no response |
| `TestGenerateOverStream_NoUsage_0088` | ST88c | Together, a stream with no usage chunk: usage zero |
| `TestGenerateOverStream_ErrorIn200_0088` | ST88d | Kilo, its discovered 16,384, C39's window form as a 200's first payload: resent once over the stream at 9,152, the response returned; Together, no limit sent: one send, `ErrInvalidRequest` |

**Red.** `python3 -B tools/redgreen.py SPEC.json --expect fail --rev 6ace750`, the new file copied in, nothing dropped: `redgreen: 5 of 5 fail as expected`, each on its assertion: `generate_stream_0088_test.go:149: claude, a stated 128,000: stream false, limit 8192 (sent true); want stream true, limit 128000`; `:183: a 503 at the opening: 2 request(s), err <nil>; want it retried once over the stream, then hello`; `:232: 17 MiB of text events: a response of 5 bytes, err <nil>; want ErrIncomplete and no response`; `:253: Usage = {InputTokens:120 OutputTokens:40 ...}; want zero: the stream sent none`; `:297: kilo, its discovered 16,384: 1 request(s); want the first asking for the stream`.

**ST88e, the existing `Generate` tests whose servers answered plain JSON**, changed as planned; by package, the existing test files changed: `llmprovider` 5, `catalog` 2, `providers` 5, `chatcompat` 5, `claude` 11, `gemini` 7, `grok` 9, `huggingface` 7, `kilo` 10, `ollama` 7, `openai` 7, `opencode` 12, `together` 7: 94 files. By kind:

* A server answers a request that asks for the stream with its reply's stream form: the shared `wiretest.NewServer` (so `wirecase`'s cases, whose `usageFor` and `Reply` also read the `:streamGenerateContent` path); and, in the files listed, a test's own handler wrapped in `wiretest.StreamReplies`, or a fake transport's reply passed through `wiretest.StreamBody` (Grok's `httpResponse`, Kilo's `kiloCalls` and its rotating-token test). Each file was kept only where its package fails without it (five that pass without were left as they were). `wiretest`'s new `stream.go` writes a whole reply as its wire's stream (Messages, Chat Completions, Responses, Interactions, generateContent), each block, item, step or part whole, with the reply's id, model, stop reason and usage; a reply that is not a 200 JSON object, an event stream the handler writes itself, and a hijacked connection pass through.
* A test whose subject is a plain request names its own limit, which ST88e keeps plain: the ten conformance harnesses (`llmtest`'s `Generate` checks, W12 among them, with `WithMaxTokens(8192)`; `Stream` is checked as before), each provider's `TestGenerate_Unchanged` (nine, `WithMaxTokens(4000)`), and `TestWithRetry_StreamIsBuffered` (0031-MADR D6's retry of a plain reply cut part way).
* An assertion of the transport changed to the stream form: `TestWorkspace_Header_0064` (two streams and two posts), `TestOpencode_RoutePaths` and `TestOpencode_RoutesFromMetadata` (`:streamGenerateContent`), `TestOpencode_KeyInHeader` (the query `alt=sse` and nothing else), `TestLimitMemo_Concurrent_0089` (the streamed memo key) and `TestLimitLearning_Default_0089` (`StreamedOutput`). Red at `6ace750` (`tools/redgreen.py --expect fail`, the files and `internal/wiretest` copied in): `3 of 3 fail as expected`, `workspace_0064_test.go:124: stream: 1 requests ...; want 2`, `opencode_test.go:44: path = "/models/gemini-3.7-flash:generateContent", want "...:streamGenerateContent"`, `metadata_route_test.go:139: path = "/models/brand-new-model:generateContent", want "...:streamGenerateContent"`; the two 0089 tests use API absent at the base.
* The eleven examples' stand-in servers answer the event stream; each `// Output:` is unchanged.

The stream form is pinned by its own tests, `internal/wiretest/stream_0088_test.go` (new): `TestNewServer_StreamForm_0088`, `TestStreamReplies_0088` and `TestStreamBody_0088` (on each of the five wires, a plain `Generate` that names its limit and one read over the stream get the same output, id, finish reason and usage) and `TestStreamReply_Errors_0088` (each wire's failure in a 200 classified as its stream classifies it); the package's coverage 94.0 %. No test was found that pins the old 8,192 default on a route that omits it beyond ST102's member, so ST102's list is unchanged.

**Goldens.** 234 G-wire goldens change, each only in the stream form (`go test -run '^TestWireGoldens$'` before `-update`: 233 `requests[0]` and 1 `requests[1]` gaining `Accept: text/event-stream`, 221 `body.stream`, 84 `body.stream_options`, 12 `path` and 12 `query`, nothing else): chatcompat 12, claude 24, gemini 14, grok 26, huggingface 12, kilo 12, ollama 12, openai 26, opencode 84, together 12. The explicit-8192 goldens and the listing goldens whose probe sends a given 8192 stay plain. Then `-update` on the ten provider packages (exit 0), the run again without it (exit 0), and `python3 -B tools/goldendiff.py --rev 6ace750 --stream`: exit 0, `goldendiff: 0 finding(s)`, `total: changed 234, unchanged 46, new 21` (the same 234 as ST100's, now in the stream form too); without `--stream`, exit 1, 564 findings, each a part of the stream form.

**Green.** `python3 -B tools/redgreen.py SPEC.json --expect pass`: `redgreen: 22 of 22 pass as expected` (the five tests above, the four of `stream_0088_test.go`, the changed assertions, Claude's and OpenCode's `TestWireGoldens`, `TestConformance` and `TestGenerate_Unchanged`, and `TestWithRetry_StreamIsBuffered`).

**Plants**, each by `python3 -B tools/plant.py` on a scratch clone, each caught by a failing test, none by a build failure (`all 7 plant(s) caught`; `all 1`):

| Plant | Test that fails | Its failing line |
| :--- | :--- | :--- |
| ST88 the stream path skipped (`OverStream` false) | `TestGenerateOverStream_0088` (and the other four) | `generate_stream_0088_test.go:149: claude, a stated 128,000: stream false, limit 128000 (sent true); want stream true, limit 128000` |
| ST88a a resend after an event: the mark not set | `TestGenerateOverStream_Retry_0088` | `generate_stream_0088_test.go:199: a stream failing after a text event: 2 request(s), resp <nil>, err ... stream ended before message_stop; want one streamed send and the failure returned, not resent` |
| ST88a a resend after an event: `WithRetry` ignores the mark | `TestGenerateOverStream_Retry_0088` | the same line, 2 requests |
| ST88b no bound on the assembled answer | `TestGenerateOverStream_ReplyLimit_0088` | `generate_stream_0088_test.go:232: 17 MiB of text events: a response of 17825792 bytes, err <nil>; want ErrIncomplete and no response` |
| ST88c an invented usage | `TestGenerateOverStream_NoUsage_0088` | `generate_stream_0088_test.go:253: Usage = {InputTokens:0 OutputTokens:1 ...}; want zero: the stream sent none` |
| ST88d the error returned as an empty answer | `TestGenerateOverStream_ErrorIn200_0088` | `generate_stream_0088_test.go:301: together, nothing sent: 1 send(s), resp &{ ... }, err <nil>; want one send and the error, classified` |
| ST88d (A5) a first-payload error not resent | `TestGenerateOverStream_ErrorIn200_0088` | `generate_stream_0088_test.go:307: kilo, its discovered 16,384: 1 send(s), err llmprovider: context window exceeded: kilo stream 400: ...; want a resend, then hello` |
| ST88e the stream form loses the reply's usage (`StreamReply`, Messages) | `TestNewServer_StreamForm_0088`, Claude's `TestWireGoldens` | `stream_0088_test.go:126: messages: over the stream &{ID: ...` and `wirecase.go:415: Usage = {InputTokens:0 ...}, want the canned reply's {InputTokens:120 ...}` |

**Checks**, each exit status taken before any filter:

| Command | Host | Exit |
| :--- | :--- | :--- |
| `go build ./...`; `go vet ./...`; `go vet -tags live_gateways ./...` | session (darwin/arm64) | 0; 0; 0 |
| `run-heavy -- go test -count=1 ./...` | session (darwin/arm64) | 0, 46 packages ok (an earlier full run failed once on `wizard`'s `TestSecret_LateLineFeedDuringWait_0087`, a 10 ms timing case in a package no layer touches; it passed alone three times and in this run) |
| `run-on-host --need linux --race -- go test -race -count=1` (`llmprovider`, `internal/transport`, `internal/wire`, `internal/wirecase`, `internal/wiretest`, `catalog`, `providers/...`) | a fleet Linux host (linux/amd64), not WSL | 0, 16 packages ok |
| `make tables-check dep-check api-check records-check markdownlint links-check docs-check fixtures-check tools-test` | session | 0 each |
| `python3 -B tools/apidiff.py v1.32.0` | session | 0, the twenty lines of layer B+C's checkpoint, unchanged |
| `python3 -B tools/claims.py` on both MADRs; `tools/recordlint.py --changed`; `tools/denyscan.py --changed` | session | 0, 0 (49 of 49; 140 of 140); 0 (no rule rose); 0 (0 hits) |

**Docs.** `WithRetry`'s doc: a `Generate` read over the stream is retried until its first event and not after (ST88a), and its buffered `Stream` retries a failure at any point of a plain `Generate`; `StreamEventsRetry`'s doc (A5); each changed `Generate`'s doc comment; `docs/architecture.md`'s rows of `internal/wire` (`OverStream`, `Collect`, the first-payload resend), `internal/transport` (`AfterEvent`) and `internal/wiretest` (the stream form); `tools/README.md`'s row. Not done here, by the Layers table: `CHANGELOG.md`, R6, R7, R57, `README.md` and `docs/guides/adding-a-provider.md`, which step 5 does in layer E.

**Findings for review**, none changing a row:

* 0031-MADR D6 says `WithRetry` retries a failure at any point, through its buffered `Generate`; under ST88a that holds for a plain `Generate` only. 0031-MADR carries no dated annotation of it yet.
* `llmtest`'s W12 checks (an undecodable 200 is `ErrIncomplete` and sent once through `WithRetry`) describe a plain reply: read over the stream, an undecodable 200 is a stream that ended before its last event, retried once (0021-MADR D1). The module's ten harnesses name a limit; `llmtest` itself is unchanged, so a provider outside the module whose `Generate` reads over the stream would fail W12-garbled.
* With ST88, a cap learned by a `Generate` that names no limit bounds `Limits.StreamedOutput`; only a caller-limited plain request feeds `Limits.Output`.

**Not done in layer D, by the Layers table:** layer E (0088's remaining tests, ST55's among them; the docs above; every gate of step 6, including `make pre-add-check`, `make lint` and `tools/gate.py`). Nothing staged or committed.

### 2026-11-23: the merged phase, layer E (stopped at ST51, then the owner's rulings)

Base `77563af` (code identical to `6ace750`), on top of the unstaged work above, which is kept. No live call. Not a build pass (deviation 7); pass count 1.

**Stopped** at ST51, before its code was kept: Kilo's documented form (0088 C97) in `contextOverflowMessages` fails three existing guard tests of that table unless two test files change, which no row named (0088-MADR's ninth amendment quotes the lines). The tests written by then were kept; `llmprovider/context_overflow.go` was put back as it was. **The owner's rulings**, relayed by the coordinator: "Name the two test changes (Recommended)" (0088-MADR's ninth amendment, recorded before the code), and, on the four plants that have no site in the merged code, "Accept, noted in the record (Recommended)" (0088-PLAN's tenth execution entry).

### 2026-11-23: the merged phase, layer E (checkpoint)

Base `77563af` (code identical to `6ace750`), on top of the unstaged work above, which is kept. No live call; every test serves its own replies. Not a build pass (deviations 7 and 8); pass count 1.

**The code.** ST51: Kilo's documented window form in `contextOverflowMessages` (`llmprovider/context_overflow.go`), with the two planned changes of 0088-MADR's ninth amendment: its sample in `overflowSamples` (`llmprovider/context_overflow_test.go`) and its pattern in the frozen reference (`llmprovider/context_overflow_ref_test.go`, its header dated). `Request.MaxOutputTokens`' doc comment (`llmprovider/contract.go`). And golangci-lint's 20 findings over the phase's code, each fixed without a suppression: errcheck's unchecked type assertions of the built body's `max_tokens` (Claude, OpenCode's messages route), Kilo's unchecked `LimitRetry.Choose` (its refusal now goes into the call's `Refusal`, as Claude's does), `LimitRetry.Optional`'s (an error, unreached, sends none), and `internal/wiretest/stream.go`'s writes, type assertions and `json.Marshal` (a held reply's failed write is returned from `Hijack` and aborts the handler with `http.ErrAbortHandler` in `finish`); goconst's repeated keys in `stream.go` and `"context window"` in `context_overflow.go` (`anchorContextWindow`, beside `anchorMaxContextLength`); ineffassign's redundant closure in OpenCode's `Generate`; and, in tests, an unnecessary conversion, an unused helper (`limitServer.listings` and its count) and an unused constant, now read by `TestClassify_KiloWindowForm_0088`. Behaviour is unchanged: every test of the touched packages passes. And `llmprovider/internal/limitmemo`'s own tests, `TestMemo_Notes` and `TestLearningOff` (`limitmemo_0089_test.go`), for layer B's notes and `LearningOff`, which the gate's coverage floor found untested in their package; their plants (an empty resolved id kept; the notes unbounded; `Off` not read) are each caught: `limitmemo_0089_test.go:99: Resolved = ""; want the id kept, the empty one ignored`, `:116: after 257 more models: note-0 1000, 258 notes; want the oldest dropped, 256 held`, `:131: Off among the values: want learning off`.

**Tests**, files new unless named:

| Test | Rows | File | Red at `6ace750` |
| :--- | :--- | :--- | :--- |
| `TestDefaultLimit_GrokSilent_0088` | ST63 | `llmprovider/providers/default_limit_0088_test.go` | `:211: grok, stream false, options 0: sent [8192]; want -1 (-1: no max_output_tokens)` |
| `TestDefaultLimit_WindowCap_0088` | ST91 (on Kilo; a model no extract names) | the same | `:246: kilo, L 20,000, W 1,000,000: sent [8192]; want 20000` |
| `TestStreamEventsRetry_FirstPayloadError_0088` | ST55 (Kilo only) | the same | `:280: the error as the first payload: sent [8192], text "", err ... context window exceeded ...` |
| `TestDefaultLimit_ResendLadder_0088` | ST56, ST57 (Claude, effort max, `Generate` and `Stream`) | the same | `:356: (1) limit, then effort, stream false: sent [8192] at efforts ["max"] ...` |
| `TestDefaultLimit_Concurrent_0088` | ST46 (Kilo, 32 requests, `-race`) | the same | `:414: [map[content:plain 25 role:user]]: sent [8192]; want [16384]` |
| `TestDefaultLimit_ListingProbe_0088` | ST26 (Claude and Ollama) | the same | green, as the row says (`--expect pass --rev 6ace750`: 2 of 2) |
| `TestDefaultLimit_ClaudeWindowStop_0088` | ST23 | the same | green, as the row says (the same run) |
| `TestDefaultLimit_OverflowResend_0088` | ST48, ST59, ST61 (cases 3, 5, 6, 7 on Claude, OpenCode's messages route and Kilo) | the same | `:590: claude, (5) the input alone: sent [8192], err ...; want one send at 128000, the error returned` |
| `TestDefaultLimit_LimitRefusal_0088` | ST50 (its Kilo `length` case) | the same | `:612: kilo, a length finish: sent [8192], ...` |
| `TestDefaultLimit_ResendBudgetShape_0088` | ST52 (ST99's value) | the same | `:648: stream false: sent [14096], the second's budget_tokens <nil>, err <nil>; want 128000 then 64000, budget 10000` |
| `TestClassify_KiloWindowForm_0088` | ST51 | `llmprovider/context_overflow_0088_test.go` | `:22: llmprovider: invalid request: kilo HTTP 400: This request exceeds the model's context window ...; want ErrContextOverflow, beneath ErrInvalidRequest` |
| `TestLimitObservations_Request_0089` | DL40, DL41, at the request (Claude and OpenCode's messages route) | `llmprovider/providers/output_limit_0089_test.go` (added) | new API: its plant is its red |
| `TestContextOverflow_EveryFormHasASample`, `_PrefilterAdmitsEveryPattern`, `_MatchesReference` (existing) | ST51's two planned changes | `llmprovider/context_overflow_test.go`, `llmprovider/context_overflow_ref_test.go` | green at the base; with the form and without the changes, red (0088-MADR's ninth amendment) |

Red: `python3 -B tools/redgreen.py SPEC.json --expect fail --rev 6ace750`, the two new files and `generate_stream_0088_test.go` (its helpers) copied in, nothing dropped: `redgreen: 9 of 9 fail as expected`, each on its assertion, the lines above. `default_limit_0088_test.go` uses only API present at the base. Green: `--expect pass` on the tree, the new tests and the existing tests of 0088-PLAN Phase 1 step 2: `redgreen: 26 of 26 pass as expected`.

Choices within the rows, for review: `TestDefaultLimit_ResendLadder_0088` case 3 walks Claude's whole ladder as the Messages matcher reads it: a limit refusal, then max, xhigh, high, medium and low refused (each listing the efforts left, the last listing `'none'`), then the adaptive shape refused and the budget shape accepted: eight sends. Case 1 runs again with a cap naming another model (0089-MADR DL36), which no memo keeps, so that the resend's stickiness is what holds the third send (0088-PLAN's tenth entry). `TestDefaultLimit_OverflowResend_0088`'s case 6 is Kilo's two window forms with the input at or over the window; Messages reads no window form (DL68). `TestStreamEventsRetry_FirstPayloadError_0088` is in `llmprovider/providers`, not `internal/wire`, since it builds Kilo's provider.

**Plants**, each by `python3 -B tools/plant.py` on a scratch clone, one spec per row whose `-run` names only that row's test(s), each caught by a failing test and none by a build failure (`all 1`, fourteen specs; `all 2`, four; `all 3`, two: 28 plants). A first draft of ST55 (b) broke the build (a variable left unused) and was rewritten; a first ST56 (a) was not caught, since the kept cap held the third send by itself, and the case above was added. Plants marked * are the owner's accepted replacements (0088-PLAN's tenth entry).

| Plant | Test that fails | Its failing line |
| :--- | :--- | :--- |
| ST63 Grok sent R (the default counted as given) | `TestDefaultLimit_GrokSilent_0088` | `default_limit_0088_test.go:211: grok, stream false, options 0: sent [8192]; want -1 (-1: no max_output_tokens)` |
| ST91 the window cap removed (`catalog/limits.go`) | `TestDefaultLimit_WindowCap_0088` | `:246: kilo, L 16,384, W 8,192: sent [16384]; want 8192` |
| ST55 (a) the first-payload resend removed | `TestStreamEventsRetry_FirstPayloadError_0088` | `:280: the error as the first payload: sent [16384], text "", err llmprovider: context window exceeded: kilo stream 400: ...` |
| ST55 (b) the resend allowed after a yielded event | the same | `:285: the error after a text delta: sent [16384 9152], 4 event(s) then err <nil>; want one send, the delta, then the error` |
| ST56 (a) the resend not sticky | `TestDefaultLimit_ResendLadder_0088` | `:356: (1) limit naming another model, then effort, stream false: sent [128000 64000 128000] at efforts ["max" "max" "xhigh"] ...` |
| ST56 (b)* a sticky resend at 8192 on any refusal `Check` cannot read | the same | `:356: (2) effort first, stream false: sent [128000 8192] at efforts ["max" "xhigh"] ...` |
| ST57 (c)* the limit's resend counted against the bound (in `sse.go`) | the same | `:356: (3) limit, then the whole ladder, stream false: sent [128000 64000 64000 64000 64000 64000 64000] ... err llmprovider: invalid request: claude ...` |
| ST46 the resend stored on the provider (one shared `LimitRetry`) | `TestDefaultLimit_Concurrent_0088` | `:402: request 0: llmprovider: context window exceeded: kilo HTTP 400 ...` |
| ST26 Claude's probe limit not given | `TestDefaultLimit_ListingProbe_0088` | `:460: claude: a probe sent [128000]; want 8192, given` |
| ST26 Ollama's probe limit not given | the same | `:460: ollama: a probe sent [-1]; want 8192, given` |
| ST23 (a) a resend on `FinishLength` | `TestDefaultLimit_ClaudeWindowStop_0088` | `:490: stream false, stopped at the window: 2 send(s), resp &{... FinishReason:length ...}` |
| ST23 (b)* an overflow of a limit the SDK chose resent unread | the same | `:504: stream false, the prompt too long: 2 send(s), err llmprovider: context window exceeded: claude HTTP 400 ...` |
| ST48 (b) the resend run for a caller's limit | `TestDefaultLimit_OverflowResend_0088` | `:590: claude, (3) WithMaxTokens(20000) refused: sent [20000 16000], err ...` |
| ST48 (e)* an overflow of a limit the SDK chose resent unread | the same | `:590: claude, (5) the input alone: sent [128000 128000], err llmprovider: context window exceeded: ...` |
| ST59 (f) the resend run on any error of a limit the SDK chose | the same | `:590: claude, (5) the input alone: sent [128000 128000], ...` (and case 7's) |
| ST50 (b) a resend on `FinishLength` (Kilo) | `TestDefaultLimit_LimitRefusal_0088` | `:612: kilo, a length finish: sent [65536 65536], ...` |
| ST52 (a) the resend's `max_tokens` set to 8192 after `AddThinkingFor` | `TestDefaultLimit_ResendBudgetShape_0088` | `:648: stream false: sent [128000 8192], the second's budget_tokens 10000, ...` |
| ST52 (b) the resend built without the thinking fields | the same | `:648: stream false: sent [128000 64000], the second's budget_tokens <nil>, ...` |
| ST51 the form removed | `TestClassify_KiloWindowForm_0088` | `context_overflow_0088_test.go:22: llmprovider: invalid request: kilo HTTP 400: ...; want ErrContextOverflow, beneath ErrInvalidRequest` |
| DL40 at the request: the limits read without the caller's options (Claude) | `TestLimitObservations_Request_0089` | `output_limit_0089_test.go:458: claude, seen an hour ago: llmprovider: invalid request: claude: no source states an output limit for model "claude-pass-back-0089" ...` |
| ST4 the plan branch sets `max_output_tokens` | `TestPlan_Request_0043` | `plan_0043_test.go:82: the body sends max_output_tokens: 8192` |
| ST19 the negative check removed | `TestCapabilitiesCheck_RefusesInvalidValues` | `contract_test.go:86: Check = <nil>, want an error matching ErrInvalidRequest` |
| ST20 the positive check removed | `TestResolveOptions_RefusesANonPositiveMaxTokens` | `hygiene_test.go:30: WithMaxTokens(0) = <nil>, want ErrInvalidRequest` |
| ST27 the Chat Completions empty check removed | `TestDecode_EmptyIsErrEmptyAnswer_0061` (chatcompletions) | `empty_0061_test.go:21: cut while still reasoning: <nil>; want an error matching ErrEmptyAnswer and ErrIncomplete` |
| ST30 the raise removed | `TestAddThinkingFor_0041` | `thinking_0041_test.go:72: a budget cut to its maximum: ... "max_tokens":8192}; want ... "max_tokens":36096}` |
| ST41 the body limit back to 64 KiB | `TestClassify_ErrorBodyLimit_0084` | `toolschema_services_0084_test.go:125: 200 KiB: llmprovider: rate limited: openai HTTP 429: ...; want quota exhau...` |
| ST45 Kilo's reasoning object given a `max_tokens` | `TestKilo_ReasoningEffortConfigured` | `kilo_test.go:152: reasoning = map[effort:low max_tokens:1024], want map[effort:low]` |
| ST61* the resend run on any overflow, raised or not | `TestKilo_ErrorIn200IsNotRetried` | `error_in_200_test.go:39: 2 requests, want 1: an overflow cannot succeed on a retry` |

**Docs.** `docs/guides/api-standards.md`: the citation of 0088 and 0089; R6's output-limit sentences (the request's rule, the sources' rank, the stream for a `Generate` naming no limit), replacing "cuts the default and never raises it"; R7's `OutputLimit`; R57 in a new section "Output limits", with its Checks row. `CHANGELOG.md` `[Unreleased]`: **Added** (`Limits.StreamedOutput` in the `Limits` entry; `SentLimit`, `LimitBasis` and its constants, `Response.OutputLimit`; `WarningLimitLearned`, `Warning.Observed`, `LimitObservation`, `WithLimitObservations`, `WithoutLimitLearning`; `Settings.MaxTokensGiven`) and **Changed** (the default rule; the resend at a stated cap or window; a `Generate` naming no limit over the stream, Gemini's listing probe included, a cap learned from it bounding `Limits.StreamedOutput`; ST55's first-payload resend; ST51's kind). `README.md`: a bullet "Output limits", the model-facts bullet's output-limit words removed, and the `WithRetry` paragraph's stream sentence. `docs/guides/adding-a-provider.md`: `MaxTokensGiven` in the constructor and in "What `Settings` gives you", and the output-limit step of encoding. `Request.MaxOutputTokens`' doc comment; every other new export's doc comment was written in its layer, and `make docs-check` holds them (R50 to R52). [0031-MADR-native-streaming-on-the-responses-wire.md](0031-MADR-native-streaming-on-the-responses-wire.md) D6 carries a dated annotation: under 0088 ST88a, `WithRetry` retries a failure at any point only for a plain `Generate`; one read over the stream is retried until its first event. R57's check, run: `git grep -n -w -E '32000|32_000|16384|16_384|8192|8_192'` in non-test Go code finds only the listing probes' given 8192 (Claude, chatcompat, Grok, Ollama, OpenAI), `MaxTokens`' default (`llmprovider/settings.go:88`, which decides nothing unless given), `internal/wirecase`'s explicit-8192 scenario, and `catalog/search.go`'s size labels, none deciding a limit.

**Checks**, each exit status taken before any filter:

| Command | Host | Exit |
| :--- | :--- | :--- |
| `go build ./...`; `go vet ./...`; `go vet -tags live_gateways ./...` | session (darwin/arm64) | 0; 0; 0 |
| `golangci-lint run -c .golangci.yml --build-tags live_gateways ./...` (v2.15.0, the fleet's pin), host and `GOOS=windows` | session | 1 first (20 findings, fixed as "The code" says, none by a suppression); then 0 and 0 |
| `run-heavy -- make pre-add-check FILES="<the phase's 156 Go files>"` | session (darwin/arm64) | 2 first (the same lint findings); then 0: `156 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)` |
| `run-heavy -- go test -count=1 ./...` | session (darwin/arm64) | 0, 46 packages ok, before and after the lint fixes |
| `run-on-host --need linux --race -- go test -race -count=1 ./llmprovider/... ./internal/wiretest/` | a fleet Linux host (linux/amd64), not WSL | 0, 38 packages ok, before and after the lint fixes |
| `run-heavy -- python3 -B tools/gate.py --timeout 5400` | session (darwin/arm64) | 1 first: 28 of 30, `coverage-check` and `gate-selftest` (which runs it) on `llmprovider/internal/limitmemo` at 55.3 %, below its 80 % floor: layer B's notes and `LearningOff` had no test in their package; `TestMemo_Notes` and `TestLearningOff` added (98.7 %), each seen failing on a plant (`all 3 plant(s) caught`); then 0, `gate: 30 of 30 passed` |
| `make tables-check dep-check api-check records-check markdownlint links-check docs-check fixtures-check tools-test changelog-check parity-check disclosure-check` | session | 0 each |
| `python3 -B tools/apidiff.py v1.32.0`; `python3 -B tools/goldendiff.py --rev 6ace750 --stream` | session | 0; 0 |
| `python3 -B tools/claims.py` on 0088-MADR and 0089-MADR | session | 0, 0 (140 of 140; 49 of 49) |
| `python3 -B tools/recordlint.py --changed`; `python3 -B tools/denyscan.py --changed` | session | 0 (a first run exited 1 on one MD012 in this PLAN's 0088 sibling, fixed); 0 |

**`apidiff`** (`python3 -B tools/apidiff.py v1.32.0`): the twenty lines of layer B+C's checkpoint, unchanged; no `Request.OutputCeiling` (DL52). **`goldendiff`** (`python3 -B tools/goldendiff.py --rev 6ace750 --stream`): exit 0, `goldendiff: 0 finding(s)`, `total: changed 234, unchanged 46, new 21`.

**Findings, recorded at the coordinator's word:**

* `wire.Send`'s uncounted branch (`llmprovider/internal/wire/retry.go`, the `sends--` under `uncounted(replaced)`) is no longer reached: since ST88 every request whose limit the SDK chose, the only kind resent at a stated value, is read over the stream (`StreamEventsRetry`), and a plain send carries a caller's or a probe's limit, which is never resent. It is kept, harmless and untested; its removal is a follow-up below.
* ST91's "as given" holds for a `WithMaxTokens` value only where no source states the output limit; where one does, the option is cut to the output held to the window (DL19, DL44): annotated on ST91 in 0088-MADR's ninth amendment.
* A passed-back cap is used by a request only when its `BaseURL` is the provider's given base, the default read as empty (DL40's annotation), which is what a `WarningLimitLearned`'s `Observed` carries; `TestLimitObservations_Request_0089` pins it on Claude and OpenCode's messages route.
* `tools/claims.py` on 0031-MADR exits 1 with "0 of 0 pass (no evidence rows)": that record has no evidence index; its D6 annotation is held by review.

**Follow-ups, not built here:**

* `llmtest`'s W12 garbled check (an undecodable 200 is `ErrIncomplete` and sent once through `WithRetry`) assumes a plain reply: a provider outside the module whose `Generate` reads over the stream would fail it, since such a 200 is a stream ended before its last event, retried once (0021-MADR D1). The module's ten harnesses name a limit and so stay plain. A later record decides whether `llmtest` checks the stream form.
* `wizard`'s `TestSecret_LateLineFeedDuringWait_0087` (a 10 ms timing case) failed once in a full run during layer D. Layer E's runs that included it: ten (two `go test ./...` and, in each of two `tools/gate.py` runs, `test-race-cover`, `test-shuffle` and `test-repeat`'s two), none failing.
* Remove `wire.Send`'s uncounted branch, now unreached (the first finding), or give it a test that reaches it, in a later record.

**The merged phase's exit list**, row by row ("Merged phase" above):

| Item | Layer | State |
| :--- | :--- | :--- |
| Phase 2's tests (DL31 to DL43, DL24's passed-back case) | A; DL35, DL37, DL43 in B+C | built, green, plants caught; DL40 and DL41 at the request in E |
| Phase 3's tests (DL44 to DL57, DL76, DL77) and the chosen "Tests by answer" (OQ1 (1), OQ2 (1), OQ3 (1)) | B+C | built, green, plants caught |
| OQ4 (1)'s check (DL52): no `Request.OutputCeiling` | E | `apidiff` lists none |
| `TestSettings_MaxTokensGiven_0088` (ST35, ST36) | pass 1 | built, green, plants caught |
| `TestDefaultLimit_GrokSilent_0088`, `_WindowCap_0088`, `TestClassify_KiloWindowForm_0088`, `TestStreamEventsRetry_FirstPayloadError_0088`, `_ResendLadder_0088`, `_Concurrent_0088`, `_ListingProbe_0088`, `_ClaudeWindowStop_0088`, `_OverflowResend_0088`, `_LimitRefusal_0088`, `_ResendBudgetShape_0088` | E | built, red at the base or green there as each row says, green, plants caught |
| `TestDefaultLimit_ClaudeOldWindowForm_0088` (ST65) | | not built: out of scope (0088-MADR's sixth amendment) |
| `TestGenerateOverStream_0088` and its four siblings (ST88, ST88a to ST88d), ST88e's changes | D | built, green, plants caught |
| The ST99-retired tests (`TestDefaultLimit_Facts_0088` and eight more) | | not built, as the table says |
| The existing tests of 0088-PLAN Phase 1 step 2 (ST4, ST19, ST20, ST27, ST30, ST41, ST45, ST61) | E | green, each plant caught |
| `TestOutputLimit_Facts_0041` (ST32); `TestWireGoldens` (ST100, ST16); ST101; ST102 | B+C, D | changed as planned, green |
| Step 5's docs | A to E | done, as the Docs paragraphs say |
| Step 6's checks | E | the table above |
| Step 7's record | E | this entry, 0088-PLAN's tenth entry, 0088-MADR's ninth amendment |

Exit criterion: no blocker found by the builder; every row of the merged phase tested; every plant caught. Pass count: 1. Nothing staged or committed.
