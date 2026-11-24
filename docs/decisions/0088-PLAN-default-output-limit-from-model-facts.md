---
status: in-progress
date: 2026-11-24
associated-madr: "0088-MADR-default-output-limit-from-model-facts.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Sending the Model's Own Output Limit When a Request Names None

Associated MADR: [0088-MADR-default-output-limit-from-model-facts.md](0088-MADR-default-output-limit-from-model-facts.md)

## Goal

After this PLAN, a request that names no output limit is sent the model's own
limit, held to the ceiling and the window rule the owner chooses, wherever
the model facts state one; a caller's limit, a listing probe and an unknown
model behave as today; a turn that fits at 8192 does not start to fail on a
strict service's window or limit; `Settings.MaxTokensGiven` tells a
caller's limit from the default; and the docs say so. The steps below follow
the owner's answers of 2026-11-21 to OQ1 to OQ8 (the MADR's "Owner
rulings"); R is the MADR's A2 value.

The MADR is `accepted`, with no open row. This PLAN is `proposed`. Nothing
below Phase 0 starts before the owner approves this PLAN and says
`proceed`, `execute the plan` or `do phase N`.
*(2026-11-22: the MADR is `proposed` again, reopened by the research amendment with six open rows; the owner's 2026-11-21 answers stand. See this PLAN's amendment of 2026-11-22.)*
*(Annotated 2026-11-22: the owner approved this PLAN and the start; it is `in-progress`. See the fifth amendment's execution entry of that date.)*

## Prerequisites

* The owner's answers to OQ1 to OQ8, recorded in the MADR as rows on
  2026-11-21 (done).
* By OQ8's answer, "Join this tag", this record joins the tag of 0081, 0083,
  0084, 0085 and 0087, whose scope becomes six. It is built last: Phase 1
  starts on the commit that carries 0087's code, after 0087's PLAN has
  passed its own review; Phase 4 is that tag's release, which waits for this
  PLAN's Phases 1 to 3.

## Scope

### In scope

* `llmprovider/options.go` and `llmprovider/settings.go`: a flag beside
  `MaxTokens` in `providerConfig`, set by `WithMaxTokens`'s own closure, so
  a `For(id, WithMaxTokens(n))` sets it for id only; the exported
  `Settings.MaxTokensGiven() bool`, after `HTTPClientGiven`, with its doc
  comment (OQ5 "Export MaxTokensGiven"); the doc comments of `MaxTokens`
  and `WithMaxTokens`; `Request.MaxOutputTokens`'s doc
  (`llmprovider/contract.go`).
* One shared, unexported helper for R, in `llmprovider/internal/wire`:
  `DefaultLimit(configured int, given bool, facts catalog.ModelInfo, ceiling int, localOnlyCallerFacts bool) (int, bool)`,
  returning the value and whether the raise set it (the name and signature
  re-read at Phase 1's start; the rule is the MADR's A2).
* Each provider's body function: Claude, OpenAI (the key branch only), Grok,
  Gemini, Kilo, Together, Hugging Face, Ollama, chatcompat, and OpenCode's
  `resolve`; each provider's constructor, where it reads `st.MaxTokens()`
  today (`kilo.go:225`, `claude.go:238`, `gemini.go:170`, `grok.go:206`,
  `openai.go:260`, `chatcompat.go:212`, `huggingface.go:159`,
  `together.go:155`, `ollama.go:161`, `opencode.go:272`, at 55a5dca), reads
  `st.MaxTokensGiven()`; each probe copy (Claude, chatcompat, Grok, Ollama, OpenAI) marks
  its limit given (MADR ST26).
* The resend (MADR A4, A5): each provider's retry check, as 0062-MADR D5's
  path takes it (`wire.Send`, `wire.StreamEventsRetry`), gains the limit
  check; `StreamEventsRetry` resends a stream whose first payload is an error
  before anything was yielded (ST55); the send that undoes the raise is not
  counted against `wire.MaxSends` (ST57).
* `llmprovider/context_overflow.go`: Kilo's documented window form as an
  overflow form (ST51). The Messages wire: Anthropic's limit form
  `max_tokens: N > M` recognised in the resend check of a raised request
  (ST49), with no change to its kind.
* Tests in `llmprovider/providers/default_limit_0088_test.go`,
  `llmprovider/internal/wire/stream_resend_0088_test.go`,
  `llmprovider/settings_given_0088_test.go` (package `llmprovider_test`) and
  `llmprovider/context_overflow_0088_test.go`, and the one existing test the
  MADR's ST32 names.
* Docs: the MADR's A8 lines; the Claude package doc's sentences on the
  output-per-minute reservation (ST38) and the `Generate` header wait
  (ST24); `docs/guides/adding-a-provider.md` (a provider outside the module
  reads `MaxTokensGiven` to apply the rule); `CHANGELOG.md` (`[Unreleased]`,
  **Added** for `Settings.MaxTokensGiven`, **Changed** for the default and
  the Kilo form). The dated annotations in 0041-MADR beside R7 and Q3 (ST31)
  and in 0062-MADR beside Q5 (ST57) were made on 2026-11-21 with the
  owner's answers.

### Exported names (R55)

* `Settings.MaxTokensGiven() bool` (OQ5 "Export MaxTokensGiven"). Call
  sites that need it: the ten built-in providers' constructors named above.
  No call site outside this module is named; R55's cost, an export only
  this module calls, is accepted by the owner's answer (MADR ST37).
  `make api-check` lists it.

### Out of scope

* gobble-cli's own code, including whether its lookup and its provider share
  a listing key (MADR ST58); billing, balance refusals and other services'
  rate limits (ST39, ST40, ST53); the wizard and sign-in records (ST42);
  option D's reasoning budget (ST45).

## Relation to Sibling Records

Read in the committed text at 55a5dca.

* 0084's and 0085's code is committed, and touched the same provider files
  (`claude.go`, `grok.go`, `openai.go`, `opencode.go`, `kilo.go` and the
  others' constructors); this record lands after them and moves nothing they
  still cite unbuilt. 0084's PLAN is `in-progress`: its remaining steps are
  records and the release.
* 0081, 0083 and 0087 change `wizard/` and the sign-in only; 0087's PLAN is
  `proposed`, its code unwritten. Shared with all five: `CHANGELOG.md`
  `[Unreleased]`, which also carries untagged entries of 0080 and 0082. By
  OQ8's answer this record is built after 0087 and joins their tag.
* 0041-MADR carries dated annotations beside R7 and its Q3 ruling, and
  0062-MADR one beside Q5, made on 2026-11-21 as additions only.
  `TestOutputLimit_Facts_0041` changes in Phase 1: its 128,000 case
  expects 16,384 (ST32), a change planned and approved with this record, not
  a deviation.

## Rules for Every Phase

* **Re-ground first.** Each phase starts by running
  `python3 -B tools/claims.py docs/decisions/0088-MADR-default-output-limit-from-model-facts.md --rev self=<start>`
  at the phase's start commit, and annotating each moved row in the MADR,
  dated, with its new line. A row whose text is gone is a stop.
* **Red, then green.** Each test marked red is run at the phase's base with
  `python3 -B tools/redgreen.py SPEC.json --expect fail --rev <base>` and must
  fail on its assertion; each marked green, `--expect pass`. A test that does
  not compile at the base goes in the spec's `drop` list, and its plant is
  its red. After the change, every test `--expect pass`.
* **Plants.** Each plant is the single production change named, made on a
  scratch clone by `python3 -B tools/plant.py PLANTS.json`, never in the tree,
  and each must turn its test red. The execution record quotes each failing
  line.
* **Heavy runs.** `python3 -B tools/gate.py`, `make pre-add-check`, and any
  `-race` run, `TestDefaultLimit_Concurrent_0088`'s included, go through
  `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>`, or
  to another host through `run-on-host` (with `--race` where the command
  runs `-race`), one at a time per host. A single package's `go test`
  without `-race` may stay on the session host.
* **No live run.** Every test serves its own replies; none reaches a service.
* **Stop conditions,** each a stop and a prompt to the owner with evidence and
  resolutions: an existing test, golden or fixture other than ST32's needs a
  change; `apidiff` against `v1.31.0` reports a name other than
  `Settings.MaxTokensGiven` and those `[Unreleased]` already lists; a red
  test cannot be red at the
  base without a production change; a row's outcome cannot be met as
  written; a third build pass of one phase.
* **Exit criterion of every phase:** no blocker, every row tested, plants
  caught.

## Implementation Steps

### Phase 0: records

1. **Re-ground** (the rule above) at HEAD when the owner takes up the drafts:
   the MADR's 125 claims, and the five sibling records' statuses.
2. This pair, drafted and revised on 2026-11-21 after record review 1. Its
   two index rows go in `docs/README.md`, with the record count, by
   `tools/records.py add`.
3. Done on 2026-11-21: the owner answered OQ1 to OQ8; each answer is a row of
   the MADR's "Owner rulings" table, quoting its label, and every open row
   it decided was rewritten `decided` with its test and plant from Phase 1's
   table. No row is open. *(2026-11-22: six rows are open again, ST63, ST65, ST66, ST67, ST71 and ST74, added by the MADR's research amendment.)*
4. gobble's model limit is cited from the SDK's 2026-10-12 snapshot (MADR
   C77, C78), not read live. Optional, if the owner wants a fresh reading:
   an agent reads Kilo's public listing with the environment's key, by a
   scratch script that prints only, for each `kilo-auto/` entry, its id and
   `top_provider.max_completion_tokens`; no other field, no key, no raw
   body. The result is quoted in the MADR, dated.
5. Done on 2026-11-21, additions only: 0041-MADR's dated annotations beside
   R7 and beside its Q3 ruling ("For a request that names no limit,
   reversed by 0088-MADR OQ1, 2026-11-21"), and 0062-MADR's beside its
   Q5 (a) ruling ("Amended for a raised request by 0088-MADR OQ6,
   2026-11-21: eight sends").
6. Done on 2026-11-21: the MADR is `accepted`. This PLAN goes to
   `in-progress` by `tools/records.py status` once the owner approves it.

Exit criterion: no blocker, every row tested, plants caught (here: every
decided row names its test and plant, and none is open). *(2026-11-22: not met again until OQ9 to OQ12 are answered; Phase 0 re-runs for them.)*

### Phase 1: the tests and the code, one commit

*(2026-11-23, deviation 6: built with 0089-PLAN's Phases 2 and 3 as one phase, after OQ10's measurement: [0089-PLAN-model-output-limit-discovery.md](0089-PLAN-model-output-limit-discovery.md), "Merged phase"; see this PLAN's sixth amendment.)*

1. **Re-ground** (the rule above) at the commit that carries 0087's code
   (OQ8), and re-read at the base: each provider's
   body function, constructor and probe copy; `AddThinkingFor`;
   `contextOverflow`, `completionFillsContext` and the input-alone forms;
   `wire.Send`, `wire.StreamEventsRetry` and `wire.MaxSends`; each
   provider's retry check under 0062-MADR D5; `catalog.CachedFacts` and
   `ModelInfo.SourceOf`; the provider with a seven-rung ladder named for
   `TestDefaultLimit_ResendLadder_0088`. Confirm that 0081, 0083, 0085 and
   0087 changed no line that computes or sends an output limit (ST42).
   Record it in the execution record.
2. **Tests first.** The provider tests use the harness of
   `TestOutputLimit_Facts_0041` (`wiretest`, a `limitLayer` stating the
   model's limit, and for the window a layer stating it, `sentLimit`), over
   its twelve provider and model pairs plus chatcompat and OpenCode Go, by
   `Generate` and by `Stream`, unless a row says otherwise. "Refused" means
   the test server answers with the named form; every other request is
   answered normally.

   | Test | Rows | What it does | At the base | Plant (on the fixed code), and why it turns red |
   | :--- | :--- | :--- | :--- | :--- |
   | `TestDefaultLimit_Facts_0088` | ST1, ST3, ST5 to ST8, ST10, ST14, ST24, ST25, ST28, ST29, ST38 | No caller limit, no window stated. `Stream`: L 128,000 sends 32,000; 20,000 sends 20,000; 8192 sends 8192; 4096 sends 4096. `Generate`: 128,000 sends 16,384; 12,000 sends 12,000. With reasoning off and at each of low, medium, high, xhigh and max where the provider takes it. | red: on `Stream`, 128,000 and 20,000 send 8192 | (a) the `Stream` ceiling removed: 128,000 sent, not 32,000; (b) the raise removed: 20,000 sends 8192; (c) `Generate` given the `Stream` ceiling: 128,000 sends 32,000, not 16,384; (d) `Stream` given the `Generate` ceiling: 16,384, not 32,000; (e) Kilo's body keeps the old cut: Kilo's 20,000 case sends 8192; (f) the raise only when reasoning is requested: the reasoning-off 128,000 case sends 8192; (g) the raise only at high and above: the low case sends 8192; (h) OpenCode's `resolve` keeps the old cut: its 20,000 cases send 8192 |
   | `TestDefaultLimit_BudgetShape_0088` | ST2 | Claude and OpenCode's messages route, the budget shape, L 128,000, no budget bounds stated, `Stream`. Budget 4096: `max_tokens` 32,000, `budget_tokens` 4096. Budget 40,000: `max_tokens` 44,096, `budget_tokens` 40,000. | red: budget 4096 sends 8192 (8192 is above 4096, so no raise, C26); budget 40,000 already sends 44,096 | (a) the budget path passes today's 8192 to `AddThinkingFor`: budget 4096 sends 8192, not 32,000; (b) the body's `max_tokens` set to R after `AddThinkingFor` returns: budget 40,000 sends 32,000, not 44,096 (and below its budget) |
   | `TestDefaultLimit_UnknownLimit_0088` | ST11, ST12, ST34, ST60 | No stated limit: 8192 with reasoning off and at high, by `Generate` and `Stream`. A `Generate` whose answer ends `length` with no text: `ErrEmptyAnswer`, one send. | green | (a) an unknown limit raised to the ceiling: 32,000 sent, not 8192; (b) a resend at the ceiling after an empty `length` answer: two sends, not one |
   | `TestDefaultLimit_StatedZero_0088` | ST13 | L stated as 0, marked known: 8192 sent. | green | the `limit > 0` guard dropped: R is not above 8192, so the cut sends min(8192, 0) = 0 |
   | `TestDefaultLimit_CallerGiven_0088` | ST15, ST16 | L 128,000: `WithMaxTokens(5000)` sends 5000; `WithMaxTokens(8192)` sends 8192; `For(id, WithMaxTokens(5000))` sends 5000. L 4096: `WithMaxTokens(20000)` sends 4096. | green | (a) the raise applied whatever the caller gave: the 5000 case sends 32,000; (b) "given" read as "differs from 8192": the 8192 case sends 32,000 |
   | `TestDefaultLimit_RequestGiven_0088` | ST17, ST18 | `MaxOutputTokens` 5000 with L 4096 sends 5000; 200,000 with L 128,000 sends 200,000; on Claude, its refusal in Anthropic's limit form is one send. | green | (a) the request's value cut to L: 4096, not 5000; (b) the resend run for a request's own limit: Claude's refusal case makes two sends |
   | `TestDefaultLimit_KiloListing_0088` | ST33, ST54 | Kilo, `WithoutModelMetadata`, a shared `catalog.WithCache`. Warm: `catalog.List` reads a fixture listing whose `kilo-auto/balanced` entry has `max_completion_tokens` 65,536; then `Stream` sends 32,000. Cold: a fresh cache; the first `Stream` sends 8192, and the server sees no listing request during it. | red: the warm case sends 8192 | (a) the helper raises only when `SourceOf` names a caller's layer: the warm case sends 8192; (b) the provider reads its facts by `catalog.LookupModel`, which fetches: the cold case sends 32,000 and the listing is read during the request |
   | `TestDefaultLimit_WindowFit_0088` | ST22, ST47 (OQ3) | `Stream`, no caller limit. L 29,491, W 32,768: 16,384. L 14,745, W 16,384: 8192. L 20,000, W 12,000: 8192. L 65,536, W 1,000,000: 32,000. L 128,000, W unstated: 32,000. | red: the first case sends 8192 | (a) the W/2 hold removed: the first case sends 29,491; (b) the floor at today's value removed: the third case sends 6,000 |
   | `TestDefaultLimit_OverflowResend_0088` | ST21, ST22, ST48, ST59, ST61 (OQ3) | Every provider and every OpenCode route, by `Generate` (refused in the reply) and `Stream` (refused at opening with a 400); L 128,000, W unstated. Each wire's overflow forms: OpenRouter's (C39), Together's split (C38), the completion-fills split (C35), OpenAI's `context_length_exceeded`, Kilo's documented form (C97). Cases: (1) a limit above 8192 refused, 8192 accepted: two sends, the second at 8192, the answer returned; (2) 8192 refused too: two sends, the overflow returned; (3) `WithMaxTokens(20000)` refused: one send; (4) L 8192 refused: one send; (5) an input-alone form (C99 to C101): one send; (6) a split whose input part is at or above the window: one send, and one below it: two; (7) 429, 500, a 0084 tool-schema refusal, and another `ErrInvalidRequest`: one send each, as today. | red: case 1 makes one send at 8192 | (a) the resend removed: case 1 returns the overflow after one send; (b) the resend run for a caller's limit: case 3 makes two; (c) a second resend allowed: case 2 makes three; (d) the helper's "raised" test made `>=`: case 4 makes two; (e) the input-alone check removed: case 5 makes two; (f) the resend run on any error of a raised request: case 7's tool-schema refusal makes two; (g) Claude's retry check without the limit check: Claude's case 1 makes one |
   | `TestDefaultLimit_LimitRefusal_0088` | ST49, ST50 (OQ3) | Claude and OpenCode's messages route, L 128,000: a server refusing any `max_tokens` above 16,000 with `max_tokens: 32000 > 16000` (`invalid_request_error`): two sends, the second at 8192, the answer returned. Kilo, L 65,536: a reply with `finish_reason` `length` and text: one send, the answer returned. | red: Claude's case makes one send at 8192 | (a) the limit form not recognised: one send, `ErrInvalidRequest`; (b) a resend on `FinishLength`: Kilo's case makes two |
   | `TestDefaultLimit_ClaudeWindowStop_0088` | ST23 (OQ3) | Claude, L 128,000, `Stream` and `Generate`: a Messages reply with `stop_reason` `model_context_window_exceeded` and text: `FinishLength`, the answer returned, one send. A 400 with "prompt is too long": one send, the overflow returned. | green | (a) a resend on `FinishLength`: two sends; (b) the input-alone check removed: the "prompt is too long" case makes two |
   | `TestClassify_KiloWindowForm_0088` | ST51 (OQ3) | Kilo's documented message (C97), with HTTP 400 and inside a 200: `ErrContextOverflow`. | red: `ErrInvalidRequest` | the form removed from the overflow forms: `ErrInvalidRequest` |
   | `TestDefaultLimit_ResendBudgetShape_0088` | ST52 (OQ3) | Claude, the budget shape, budget 10,000, L 128,000, `Stream`: the first send `max_tokens` 32,000 is refused in the limit form; the second sends `max_tokens` 14,096 and `budget_tokens` 10,000. | red: one send (14,096 accepted at the base) | (a) the resend's `max_tokens` set to 8192 after `AddThinkingFor`: 8192, below the budget; (b) the resend built without the thinking fields: no `budget_tokens` |
   | `TestStreamEventsRetry_FirstPayloadError_0088` | ST55 (OQ6) | Kilo and Together `Stream`, L 128,000: a 200 whose first payload is an error in C39's form for a limit above 8192, a normal stream at 8192: two sends, the answer streamed, no event before it. A 200 whose error follows a text delta: one send, the error yielded after the delta. | red: one send (8192 accepted at the base) | (a) the first-payload resend removed: one send, the error yielded; (b) the resend allowed after a yielded event: the second case makes two |
   | `TestDefaultLimit_ResendLadder_0088` | ST56, ST57 (OQ6) | The provider named at step 1, L 128,000, effort `max`. (1) A limit refusal, then an effort refusal: the third send at 8192 with the lower effort. (2) An effort refusal first: the second send keeps R with the lower effort. (3) A limit refusal, then refusals down the whole ladder, the last rung accepted: eight sends, the answer returned. | red: case 1's first send is 8192 | (a) the undo not sticky: case 1's third send is R; (b) the raise undone after any refusal: case 2's second send is 8192; (c) the undoing send counted against the bound: case 3 stops at seven, refused |
   | `TestDefaultLimit_Concurrent_0088` | ST46 | Under `-race`, 32 goroutines on one Kilo provider, L 128,000; the server refuses above 8192 for half of them (by a marker in the request) and accepts the rest: each refused request sends 32,000 then 8192; each other sends 32,000 once. | red: every request sends 8192 once | the undone raise stored on the provider: a request after an undo sends 8192, or the race detector reports the write |
   | `TestDefaultLimit_ListingProbe_0088` | ST26 | Claude and Ollama `ListModels` with probes on, a caller's layer stating L 128,000 for the probed model: each probe sends 8192. | green | the probe copy's limit not marked given: the probe sends 32,000 |
   | `TestDefaultLimit_LocalFacts_0088` | ST9 (OQ7) | chatcompat and Ollama: a cached models.dev document stating 32,768 for the canonical id: 8192 sent; a caller's layer stating 32,768: 32,000 sent. | red: the layer case sends 8192 | the source check removed: the models.dev case sends 32,000 |
   | `TestSettings_MaxTokensGiven_0088` | ST35, ST36 (OQ5) | In package `llmprovider_test`, through the exported API as an outside provider reads it: `ResolveOptions` with no `WithMaxTokens`: false; `WithMaxTokens(8192)`: true; `For(claude, WithMaxTokens(5000))`: true when resolving for Claude, false for Kilo. | dropped (does not compile at the base) | (a) `WithMaxTokens` does not set the flag: the 8192 case reads false; (b) the flag derived from `MaxTokens != 8192`: the 8192 case reads false |

   Rows pinned by existing tests, each with its plant: ST4
   `TestOpenAI_ChatGPTOmitsMaxOutputTokens` and `TestPlan_Request_0043` (the
   plan branch sets `max_output_tokens`); ST14 `TestOutputLimit_Facts_0041`,
   its 4096 case (one provider's cut removed); ST19
   `TestCapabilitiesCheck_RefusesInvalidValues` (the negative check removed);
   ST20 `TestResolveOptions_RefusesANonPositiveMaxTokens` (the positive check
   removed); ST26 also `TestListModels_ProbeIsTheOldRequest`, which states
   no facts and so cannot catch ST26's plant alone; ST27
   `TestDecode_EmptyIsErrEmptyAnswer_0061` on each wire (the Chat
   Completions empty check removed); ST30 `TestAddThinkingFor_0041` (the
   raise removed); ST41 `TestClassify_ErrorBodyLimit_0084` (0084's own
   plant); ST45 `TestGenerate_Reasoning` and
   `TestKilo_ReasoningEffortConfigured` (Kilo's reasoning object given a
   `max_tokens`); ST61 `TestKilo_ErrorIn200IsNotRetried` and
   `TestTogether_TypedErrorInside200SentOnce`, which stand because their
   models state no limit (the resend run on any overflow, raised or not:
   Kilo's test counts two requests).

   ST32: `TestOutputLimit_Facts_0041`'s 128,000 case expects 16,384, sent
   by `Generate`, and its doc comment says so; planned and approved with
   this record. `settings_test.go:83` and `concurrency_0048_test.go:83`
   stand unchanged.

   Rows pinned by checks, each with its plant: ST37 `make api-check` and
   `python3 -B tools/apidiff.py v1.31.0`, which list `MaxTokensGiven` beside
   `[Unreleased]`'s names (plant: a second exported name added on a scratch
   clone, which `apidiff` reports beyond that list); ST31, ST38, ST43 and
   ST44 a `tools/claims.py` row on each annotation, doc sentence and
   changelog entry, each planted false by `tools/claims.py --plants`, with
   `make changelog-check` and `make docs-check` (step 6).
3. **Red run at the base,** as the rules say; quote the failing lines.
4. **The code.**
   * `options.go` and `settings.go`: the flag, set by `WithMaxTokens`;
     `Settings.MaxTokensGiven`, with its doc comment.
   * The shared helper: with `given` false and a stated, positive L (on
     chatcompat and Ollama, only from a caller's layer), R = min(L,
     ceiling, W/2 where W is stated), held at least at today's value; it
     returns R and true when R is above today's value, else today's value
     and false.
   * Each provider's body function calls it with its ceiling (32,000 for a
     stream, 16,384 otherwise); each keeps `req.MaxOutputTokens` first, as
     today; the OpenAI ChatGPT and plan branches are untouched; each probe
     copy marks its limit given.
   * Kilo's documented window form in `contextOverflowMessages`.
   * The resend: each provider's retry check, when the helper raised the
     request's limit and the request has not yet been undone, returns
     "resend, undone" for an overflow that does not show the input alone
     over the window, an `ErrInvalidRequest` with the completion-fills
     reason, or, on Messages, the limit form; the request's next build uses
     today's value for the rest of the request; `wire.Send` and
     `StreamEventsRetry` do not count that send against `MaxSends`;
     `StreamEventsRetry` treats an error as the stream's first payload,
     before any yield, as a failure to open.
   * `GOOS=linux`, `darwin` and `windows` `go vet ./llmprovider/...`.
5. **Green and plants:** every test `--expect pass`; each plant red, on a
   scratch clone; `go test ./llmprovider/...` through `run-heavy`.
6. **Docs.** MADR A8's lines; the Claude package doc's sentences for ST24 and
   ST38; `docs/guides/adding-a-provider.md`'s sentence on `MaxTokensGiven`;
   `CHANGELOG.md`'s **Added** and **Changed** entries; a `tools/claims.py`
   row in the MADR's evidence index for each new sentence, for each
   changelog entry, and for the 0041 and 0062 annotations once committed,
   each planted false by `tools/claims.py --plants` and caught.
7. **Checks.** `make pre-add-check FILES="..."` through `run-heavy`; `make
   lint`; `make api-check` and `python3 -B tools/apidiff.py v1.31.0`
   (`MaxTokensGiven` beside `[Unreleased]`'s names, nothing else); `make records-check
   docs-check links-check markdownlint changelog-check tables-check
   disclosure-check`; `python3 -B tools/denyscan.py --changed`; `python3 -B
   tools/claims.py` on the MADR.
8. **Records.** The execution record: the pass number, the red and green
   output, each plant's failing line, each check's exit status taken before
   any filter, and anything not done.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 2: planted breaks and the gates

1. **Re-ground** at Phase 1's commit.
2. `python3 -B tools/plant.py PLANTS.json` with every plant of Phase 1's
   table and of the existing-test list, on a scratch clone; each caught,
   each failing line quoted.
3. `python3 -B tools/redgreen.py RED.json --expect fail --rev <base>` and
   `--expect pass` on Phase 1's commit, quoted.
4. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=1 ./llmprovider/...`
5. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B tools/gate.py --timeout 5400`
6. `make lint` and `make vuln`; `go vet -tags live_gateways ./...`; CI's set:
   `make parity-check dep-check coverage-check api-check contract-check
   generate-check records-check tables-check fixtures-check changelog-check
   docs-check markdownlint links-check disclosure-check gate-selftest`, and
   `make tools-test`.
7. `python3 -B tools/claims.py` on the MADR and this PLAN;
   `python3 -B tools/recordlint.py --changed`.
8. *(Added 2026-11-23, deviation 9: an open item.)* ST43's `tools/claims.py`
   row on each of the merged phase's changelog entries, planted false,
   which the merged phase did not add.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 3: an independent review

1. **Re-ground** at Phase 2's commit.
2. A reviewer who did not build the phase reads the MADR, this PLAN and the
   diff from the base, not the builder's summary, and reports findings by
   row: a row whose test does not pin its outcome, a case the table lacks,
   a cost the MADR misstates.
3. A finding that changes a row or adds one goes to the owner in one prompt,
   three options each, the recommended first; the ruling is recorded in the
   MADR as a row and in this PLAN as a dated deviation, and Phase 1's steps
   run again for it (a new pass, numbered).
4. *(Added 2026-11-23, deviation 9.)* The reviewer rules, with the owner,
   on the three items the MADR's tenth amendment leaves open from Phase 1b:
   R1 (the wire-general cut rule), R2 (Responses' incomplete or never-done
   call) and R3 (`wire.HeldCalls`).

Exit criterion: no blocker, every row tested, plants caught.

### Phase 4: the release (the owner's word)

No step runs under a standing approval: the tag and the GitHub Release each
wait for the owner's word in the same turn.

1. **Re-ground** at the release commit, the changelog entry included.
2. By OQ8's answer, "Join this tag": the release of the tag that carries
   0081, 0083, 0084, 0085, 0087 and 0088, after this PLAN's Phase 3.
   `CHANGELOG.md`'s `[Unreleased]` becomes the next minor version's section
   (the tag adds exported names), dated by its commit; `make
   changelog-check`; `python3 -B tools/apidiff.py v1.31.0` lists
   `MaxTokensGiven` beside the other records' additions and nothing else;
   the full gate on the release commit.
3. On the owner's word: the tag, pushed; CI on the tag in full.
4. On the owner's word: the GitHub Release with the section's notes.
5. `python3 -B tools/consumer.py` and `python3 -B tools/offline.py`, quoted.
6. `tools/records.py status` sets this PLAN `complete` once the tag's CI and
   step 5 are recorded.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 5: gobble-cli, after the release (gobble-cli's records)

Named here, not run under this PLAN: gobble-cli bumps the module and shows in
its own record that its model lookup and its provider share Kilo's listing
key, so that `kilo-auto/balanced`'s stated limit reaches the request (MADR
ST33, ST58), or passes the fact by `catalog.WithModelLayer`.

## Verification

* Every decided row has its test green and its plant caught, quoted in the
  execution record.
* The tests marked red are seen red at the base and green after.
* `TestOutputLimit_Facts_0041` passes with its one planned change; every
  other existing test passes unchanged.
* `apidiff` lists `MaxTokensGiven` beside `[Unreleased]`'s names and nothing
  else; the gates pass.

## Rollout and Rollback

* Rollout: a consumer takes it by bumping the module. A caller that wants
  today's behaviour passes `WithMaxTokens(8192)`, which this record keeps
  exactly (ST16). No configuration or data changes.
* Rollback: revert the phase's commit; the default returns to 8192, cut
  only, and Kilo's documented form returns to `ErrInvalidRequest`. Before
  the tag, `MaxTokensGiven` goes with the revert; after it, removing an
  exported name is a new record, not a revert.

## Execution Record

No phase has run: the PLAN awaits the owner's answers and approval.

### Drafting (2026-11-21)

* The pair was written from the coordinator's brief, this repository at
  55a5dca, gobble-cli at 4c3e397, and the upstream checkouts named in the
  MADR, all read only. No code, test, other document, index row, staging or
  commit.
* `python3 -B tools/claims.py` on the MADR: 74 of 74 rows pass, exit 0.

### Revision after record review 1 (2026-11-21)

* Every finding of review 1 was re-verified against the code, the snapshots
  and the records before it was written in, from the same revisions. The
  MADR's rows went from 13 decided, 6 out of scope and 27 open to 19, 5 and
  37 (61 rows); its questions from six to eight; its evidence index from 74
  to 125 rows. This PLAN's tests were rewritten to match, each plant's
  arithmetic worked against the fixed code.

### Owner answers recorded (2026-11-21)

* The owner's answers to OQ1 to OQ8, relayed by the coordinator, were
  recorded in the MADR's "Owner rulings" table with their labels quoted.
  OQ5 ("Export MaxTokensGiven") and OQ8 ("Join this tag") took option (2),
  not the recommended one; the other six took option (1). The MADR's 37
  open rows became decided (56 decided, 5 out of scope, 0 open), and the
  MADR went to `accepted`. This PLAN was amended for OQ5 (the exported
  accessor, its doc, its **Added** entry, the `apidiff` expectation) and
  OQ8 (built after 0087, released in that tag); it stays `proposed`.
* 0041-MADR and 0062-MADR gained dated annotations, additions only.

## Amendment (2026-11-22): the research's rows, and what Phase 1 waits for

Second draft, after record review 1 of 2026-11-22. Additions only; every step
above stands as written, annotated where the amendment touches it. The MADR's
amendment of 2026-11-22 adds ST62 to ST74 from
[0088-REPORT-output-limits-across-agents-and-vendors.md](../reports/0088-REPORT-output-limits-across-agents-and-vendors.md):
4 decided, 3 out of scope, 6 open, put to the owner as OQ9 to OQ12. The MADR
is `proposed` again (reopened 2026-11-22; the owner's 2026-11-21 answers
stand). This PLAN stays `proposed`.

### What waits for which answer

* **Phase 0 runs again** when OQ9 to OQ12 are answered: each answer becomes
  a row of the MADR's "Owner rulings", each open row is rewritten `decided`
  with its test and plant from the table below, and the MADR returns to
  `accepted`.
* **Phase 1 waits for OQ9 and OQ10.** OQ9 decides Grok's streamed request and
  whether a stream keeps the 32,000 ceiling, so it sets
  `TestDefaultLimit_Facts_0088`'s streamed expectations. OQ10 (1) needs a
  live measurement before Phase 1 (below); (2) adds a header; (3) adds
  nothing.
* **OQ11 and OQ12 do not block this PLAN** under option (1) of OQ11 or any
  option of OQ12: each is a new SDK record after this tag, with its own pair
  and its number from `python3 -B scripts/check_records.py --next` when
  written. Under OQ11 (2), Phase 1 grows by three wires' decoders, and their
  tests are written into this PLAN by a dated deviation before Phase 1 starts.
* **Phase 1 step 6's Claude sentence** is ST62's, not the one ST38 planned:
  it says Anthropic does not count `max_tokens` toward its output-per-minute
  limit. Step 6 also writes ST73's `Generate` sentence and, unless OQ9 is
  answered (1), nothing new for Grok.

### OQ10 (1): the live measurement, before Phase 1

Only if OQ10 is answered (1). `TestLive_ClaudeWindowRefusal_0088`, built with
`live_gateways` and switched on by `LLMPROVIDER_LIVE_0088` with
`LLMPROVIDER_LIVE_0088_MODEL` naming a Claude model before 4.5, sends one
request whose input is about 192,000 tokens with `max_tokens` 32,000 and
records the refusal's status, error type and message, redacted, under
`llmprovider/providers/claude/testdata/window-0088/`. It is run under the
repository's live-run rules (AGENTS.md "Live tests"), never by an agent that
lacks the key in its environment, and prints no key or body beyond the
message. Its cost: one large request; whether a refused request is billed
was not established. Phase 1's test then serves the recorded message.

### Tests added to Phase 1's table, by row and answer

| Test or check | Rows | What it does | At the base | Plant, and why it turns red |
| :--- | :--- | :--- | :--- | :--- |
| a `tools/claims.py` row on the Claude package doc's output-rate sentence (step 6) | ST62 | the sentence says Anthropic does not count `max_tokens` toward its output-per-minute limit | absent (added with the sentence) | the sentence planted to say the limit is reserved at the start: the row fails |
| `tools/claims.py` on C126, C127 and C134 | ST64, ST66's evidence | the amendment's evidence rows | green | each row's text planted false by `tools/claims.py --plants`: caught (seen 2026-11-22 for C126, C131 and C133) |
| `TestAddThinkingFor_0041` (existing) | ST68 | as ST30 | green | as ST30's: the raise removed |
| a `tools/claims.py` row on the `Generate` doc's advice to stream a request that reasons (step 6) | ST73 | the sentence cites OpenAI's 25,000 recommendation and the 16,384 ceiling | absent | the advice planted out of the sentence: the row fails |
| under OQ9 (1): `TestDefaultLimit_GrokStreamNoLimit_0088` | ST63 | Grok, a layer stating L 128,000: `Stream` sends no `max_output_tokens`; `Generate` sends 16,384; `WithMaxTokens(5000)` on `Stream` sends 5000 | red: the stream sends 8192 | (a) Grok's stream given R: 32,000 sent; (b) the omission applied to `Generate` too: no field on the plain request |
| under OQ9 (3): `TestDefaultLimit_Facts_0088`'s streamed cases | ST74, ST1, ST3, ST5 to ST8, ST10, ST25 | `Stream`, no caller limit: L 128,000 with no window stated sends 128,000; L 128,000 with W 200,000 sends 100,000; L 20,000 sends 20,000; `Generate` cases unchanged | red: 8192 at the base | the 32,000 ceiling kept on the stream: the first case sends 32,000 |
| under OQ9 (2) | ST63, ST74 | none: Phase 1's table as written | | |
| under OQ10 (1): `TestDefaultLimit_ClaudeOldWindowForm_0088` | ST65 | Claude, a layer stating L 32,000 and W 200,000, `Stream`; the server refuses any `max_tokens` above 8192 with HTTP 400 `invalid_request_error` and the message recorded by the live measurement: two sends, the second at 8192, the answer returned; the same refusal of a caller's `WithMaxTokens(20000)`: one send, the refusal returned | red: one send at 8192, which the server accepts, so the two-send assertion fails | the form not recognised: the raised case makes one send and returns the refusal |
| under OQ10 (2): `TestDefaultLimit_WindowStopHeader_0088` | ST65 | Claude, a layer stating L 32,000: a raised request carries `anthropic-beta: model-context-window-exceeded-2025-08-26`; a request at 8192, or with a caller's limit, carries none | red: no header | (a) the header on every request: the caller's case carries it; (b) the header removed: the raised case lacks it |
| under OQ10 (3) | ST65 | none; ST50's test covers the returned refusal | | |

Rows pinned by an out-of-scope statement, no test: ST69, ST70, ST72.

### Execution record entries (2026-11-22)

* First draft: records only, from the report above, this repository at
  18e8e72 (its `llmprovider/` unchanged since 55a5dca), gobble-cli at 0589454
  and the upstream checkouts, all read only. `python3 -B tools/claims.py` on
  the MADR: 133 of 133, exit 0.
* Second draft, after record review 1: the MADR's status set `proposed` with
  a dated note; ST62 and ST73 decided from the vendors' pages; ST63 opened
  (Grok) and ST74 added (a stream above 32,000), both under OQ9; OQ10's
  options rewritten (a live measurement; the header's name verified on
  Anthropic's stop-reasons page); OQ12's options regrouped; ST72's text
  corrected for Kilo's 503; C134 added; the OQ10 test's base outcome
  corrected. No code, test, index row, staging or commit.

## Amendment (2026-11-22, second): the owner's answers, OQ11's phase, and OQ9's tests by option

Additions only. The owner answered OQ10 "Measure, then recognise
(Recommended)", OQ11 "Inside 0088, this tag" and OQ12 "New record, all three
(Recommended)"; OQ9 is reworked in the MADR's second amendment and stays open.
The MADR now has 74 decided, 8 out of scope and 7 open rows (ST63, ST74,
ST85 to ST89). This PLAN stays `proposed`.

### What changes in the phases

* **Phase 0 runs again** for these answers: the MADR's rows are recorded
  (done in the MADR's second amendment); on the owner's approval, the dated
  annotations ST83 names are made in 0012-MADR (§1.5) and 0020-MADR (F3),
  each with a `tools/claims.py` row, planted false; when OQ9 is answered, its
  rows become decided with their tests from the table below, and the MADR
  returns to `accepted`.
* **OQ10.** The live measurement of the first amendment ("OQ10 (1): the live
  measurement, before Phase 1") is now a step of this PLAN, before Phase 1;
  `TestDefaultLimit_ClaudeOldWindowForm_0088` is in Phase 1's table.
* **OQ11: a new Phase 1b, "the cut answer, one way"**, built after Phase 1 in
  the same tag, before Phase 2. Its tests are below. Phase 1's stop condition
  "an existing test, golden or fixture other than ST32's needs a change" does
  not apply to the tests that pin a cut tool call or a cut Responses answer as
  `ErrIncomplete`: Phase 1b step 1 names each, and each changes as planned
  with this answer (ST84). `apidiff`'s expected list becomes
  `Settings.MaxTokensGiven` and `WarningToolCallCut` (ST82).
* **OQ12** adds no step here: its record is written after this tag, with its
  own pair; the MADR's second amendment states its scope.
* **Phase 1 waits for OQ9.** Its default-limit tests are the chosen option's
  rows below.

### Phase 1b: the cut answer, one way (OQ11)

1. **Re-ground** at Phase 1's commit: each wire's decoder and stream reader
   (`chatcompletions`, `messages`, `generatecontent`, Gemini's interactions,
   `responses`), the empty-answer helper (`wire.EmptyAnswer`), and every test
   that asserts `ErrIncomplete` for a cut answer; list them.
2. **Tests first**, red at Phase 1's commit:

   | Test | Rows | What it does | At the base | Plant, and why it turns red |
   | :--- | :--- | :--- | :--- | :--- |
   | `TestCutAnswer_Text_0088` | ST75 | each wire, `Generate` and `Stream`: a reply cut at the limit with text returns a response finishing `length` with the text | red on Responses (`ErrIncomplete`) | Responses' cut answer left an error: the Responses cases fail |
   | `TestCutAnswer_CompleteCalls_0088` | ST76, ST77, ST78 | each wire: a cut reply with one complete call and one cut call returns a response finishing `length` with the complete call only, and one `WarningToolCallCut` naming the cut call; Chat Completions: a call with arguments `{"path":"a"` is the cut one; an earlier call with empty arguments followed by another call is complete; *(third draft, ST94)* a last call with empty arguments on a `length` finish is the cut one; Gemini's Interactions answer with status `incomplete` and one completed call is returned finishing `length` (ST93) | red (`ErrIncomplete` on every wire) | (a) the complete calls dropped: no call; (b) the warning omitted: none; (c) the Chat Completions completeness check removed: the cut call is returned; (d) the empty last call kept: it is returned as `{}` |
   | `TestCutAnswer_NothingUsable_0088` | ST79 | each wire: a cut reply with only reasoning, and one with only a cut call: `ErrEmptyAnswer`, its reason the service's finish reason | red on Responses (`ErrIncomplete`) and on the cut-call-only case (`ErrIncomplete`) | an empty response returned: no error |
   | `TestCutAnswer_Stream_0088` | ST80 | each wire's `Stream`: the complete call yielded as an item, the cut call never yielded, `EventDone` with a response finishing `length` | red: the stream ends with an error | the stream ends with `ErrIncomplete`: no `EventDone` |
   | `TestCutAnswer_OtherIncomplete_0088` | ST81 | Responses incomplete with `content_filter`: as today | green | a content-filter incomplete returned as a response: the test fails |
   | `make api-check`, `python3 -B tools/apidiff.py v1.31.0` | ST82 | `WarningToolCallCut` and `Settings.MaxTokensGiven` beside `[Unreleased]`'s names | red until added | a third name on a scratch clone: reported |
   | `tools/claims.py` rows on the 0012 and 0020 annotations | ST83 | each annotation's text | absent | each planted false: caught |
   | `make changelog-check` and a claims row on the **Changed** entry | ST84 | the entry says a cut answer is a response finishing `length` | absent | the entry planted to say `ErrIncomplete`: caught |

3. **The code**: each decoder and stream reader returns the cut answer's
   usable part; the warning kind; the docs of `Response.FinishReason`,
   `ErrIncomplete` and `ErrEmptyAnswer`; the changelog entry.
4. Green, plants, checks and the execution record as Phase 1's steps 5 to 8.

Exit criterion: no blocker, every row tested, plants caught.

### OQ9's tests, by option (only the chosen option's are built)

| Answer | Test | Rows | What it does | At the base | Plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| OQ9 (1) *(third draft: see the third amendment's table, which replaces these three rows)* | `TestDefaultLimit_PerService_0088` | ST63, ST74, ST85, ST86 | no caller limit, `Stream`: Claude, L 128,000: `max_tokens` 128,000; Kilo, L 65,536: 65,536; Gemini, L 65,536: 65,536; Grok, L 128,000: no field; OpenAI with a key, L 128,000: no field; the same with a caller's layer stating 20,000: 20,000; Ollama, no caller layer: no field; Kilo with no stated limit: no field; Claude with no stated limit: 8192 | red: 8192 everywhere | (a) a 32,000 ceiling kept: Claude sends 32,000; (b) Grok given R: a field sent; (c) OpenAI's omission ignoring a caller's layer: the 20,000 case sends nothing; (d) an unknown limit sent as 8192 on Kilo: a field sent |
| OQ9 (1) *(third draft: replaced; this Kilo message mixed two forms, see the third amendment)* | `TestDefaultLimit_StatedRoomResend_0088` | ST87, ST89 | Kilo, L 40,000: the server refuses with "This endpoint's maximum context length is 131072 tokens. However, you requested about 140000 tokens (100000 in the messages, 40000 in the completion)": two sends, the second at 131,072 − 100,000 = 31,072; Claude, L 128,000: refused with `max_tokens: 128000 > 64000, which is the maximum allowed number of output tokens for` the model: the second send at 64,000; a refusal with no number: the second send at 8192 | red: one send | (a) the stated room ignored: the second send is 8192; (b) the stated maximum ignored: Claude's second send is 8192 |
| OQ9 (1) *(third draft: replaced by OQ15's rows)* | `TestGenerate_StreamsRaisedLimit_0088` | ST88 | `Generate`, no caller limit, Claude L 128,000: the request carries `"stream": true`, `max_tokens` 128,000, and the call returns the whole response; with `WithMaxTokens(4000)`: no `stream` field | red: a plain request at 8192 | the stream path skipped: a plain request at 128,000 |
| OQ9 (1) | `TestDefaultLimit_WindowFit_0088` | ST87 | removed from Phase 1's table: no half-window rule | | |
| OQ9 (2) | `TestDefaultLimit_GrokStreamNoLimit_0088` and Phase 1's table as written | ST63, ST74, ST85 to ST89 | as the first amendment's table states | | |
| OQ9 (3) | `TestDefaultLimit_Facts_0088`'s streamed cases, as the first amendment's table states for its option (3) | ST63, ST74, ST85 to ST89 | | | |

### Execution record entry (2026-11-22, second)

* Records only: the owner's answers to OQ10 to OQ12 and the reply on OQ9
  recorded in the MADR's second amendment; the report's round 2 section
  written from R4 to R7; this amendment. No code, test, index row, staging or
  commit.

## Amendment (2026-11-22, third): record review 2's rows, and tests for each split question

Additions only; the second amendment's rows that this one replaces are marked
in place. The MADR's third amendment decides ST63, ST91, ST93 to ST96 and,
under OQ15 (1), ST88a to ST88e; it opens ST90, ST92, ST97 and ST98 and splits
OQ9 into OQ9, OQ14, OQ15 and OQ17, with OQ16 for the new export. This PLAN
stays `proposed`; Phase 1 waits for OQ9, OQ14, OQ15 and OQ17, Phase 1b for
OQ16.

### Tests for the decided rows (Phase 1 and 1b)

| Test | Rows | What it does | At the base | Plant, and why it turns red |
| :--- | :--- | :--- | :--- | :--- |
| `TestDefaultLimit_GrokSilent_0088` | ST63 | Grok, a layer stating 128,000, no caller limit: `Stream` and `Generate` send no `max_output_tokens`; `WithMaxTokens(5000)`: 5000 | red: 8192 | Grok sent R: a field present |
| `TestDefaultLimit_WindowCap_0088` | ST91 | a layer stating L 16,384 and W 8,192: 8,192 sent, whatever OQ9 and OQ14 answer; L 20,000 and W 1,000,000 with a short input: 20,000 under every option; `WithMaxTokens(20000)` with W 8,192: 20,000, as given | red: the base sends 8192 in the second case (it never raises) | the window cap removed: under OQ14 (1) or (3) the first case sends 16,384; under OQ14 (2) half the window always lies below the window, so the cap is implied and pinned by `TestDefaultLimit_WindowFit_0088`'s plants *(2026-11-23, deviation 12, the owner's ruling "Expect 4,095; add a cap case (Recommended)": under ST87 the first case sends the estimate's room, 4,095; a new case, `WithMaxTokens(20000)` with L 16,384 and W 8,192, sends 8,192, and the window-cap plant moves to it: 16,384)* |
| `TestCutAnswer_Text_0088` and `TestCutAnswer_NothingUsable_0088`, Interactions cases | ST93 | Gemini's Interactions: status `incomplete` with text: a response finishing `length`; with nothing usable: `ErrEmptyAnswer`, reason `incomplete` | red: `ErrIncomplete` | Interactions left an error |
| `TestCutAnswer_CompleteCalls_0088`, the empty-last case | ST94 | Chat Completions, `length`: calls `a` (`{"x":1}`), `b` (empty), `c` (empty, last): `a` and `b` returned, `c` dropped and named | red | the empty last call kept: returned as `{}` |
| `TestCutAnswer_Text_0088`, an Interactions reply stopped at 65,536 | ST95 | the reply is returned as a cut answer; no second request is made | red | a continuation request sent: two requests |
| `TestDefaultLimit_PerService_0088`, OpenCode cases | ST96 | OpenCode's responses and google routes with no stated limit follow ST86's rule as OQ9 decides | as OQ9's table | as OQ9's table |

### Tests by answer (only the chosen option's are built)

| Answer | Test | Rows | What it does | At the base | Plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| OQ9 (1) | `TestDefaultLimit_PerService_0088` | ST74, ST85, ST86, ST96, ST98 | no caller limit, `Stream`; W stated large: Claude, L 128,000: 128,000; Kilo, L 65,536: 65,536; Gemini, L 65,536: 65,536; OpenAI with a key, L 128,000: no field, and with a caller's layer (not a models.dev copy) stating 20,000: 20,000; Ollama and chatcompat with no caller layer: no field. No stated limit: Kilo, Together, Hugging Face, chatcompat, Gemini, and OpenCode's chat, responses and google routes: no field; Claude and OpenCode's messages route: 8192 | red: 8192 everywhere | (a) a 32,000 ceiling kept: Claude sends 32,000; (b) OpenAI's silence ignoring the caller's layer: the 20,000 case sends nothing; (c) an unknown limit sent as 8192 on one optional wire, one plant per wire of ST86's list: that wire sends a field; (d) Ollama sent 8192: a field |
| OQ9 (2) | the same test, other expectations | ST74, ST85, ST86, ST98 | stated limits as in (1) on every wire, OpenAI included; no stated limit: 8192 on every wire | red | (a) a 32,000 ceiling kept; (b) OpenAI silent: no field |
| OQ9 (3) | Phase 1's table as written (32,000 streamed, 16,384 plain) | ST74, ST85, ST86, ST98 | | | |
| OQ14 (1) | `TestDefaultLimit_RefusalRoomResend_0088` | ST87, ST89, ST92 | Kilo, L 16,384, W 32,768, refused with C39's form "This endpoint's maximum context length is 32768 tokens. However, you requested about 40000 tokens.": input 40,000 − 16,384 = 23,616, second send 32,768 − 23,616 = 9,152. Kilo, L 65,536, W 200,000, refused with C97's form "This request exceeds the model's context window of 200000 tokens. Your request contains approximately 250000 tokens.": input 184,464, second send 15,536. Together, L 60,000, W 131,072, refused with OpenAI's split form "This model's maximum context length is 131072 tokens. However, you requested 160000 tokens (100000 in the messages, 60000 in the completion).": second send 31,072. C97 with 300,000: input 234,464 over the window: one send, the overflow returned. Claude, L 128,000, refused with `max_tokens: 128000 > 64000, which is the maximum allowed number of output tokens for` the model: second send 64,000. A refusal with no number: second send 8192 | red: one send | (a) the derived input ignored: the C39 case resends 8192; (b) the limit sent not subtracted: the C39 room is negative, so one send; (c) the stated maximum ignored: Claude's second send is 8192 |
| OQ14 (2) | `TestDefaultLimit_WindowFit_0088` (Phase 1's table) | ST87, ST89, ST92 | as Phase 1's table states, with ST91's cap | | |
| OQ14 (3) | `TestDefaultLimit_EstimatedFit_0088` | ST87, ST89, ST92 | a request whose built body holds 350,000 characters (100,000 tokens at 3.5), W 131,072, L 60,000: sent 131,072 − 100,000 − 4,096 = 26,976 *(2026-11-23, deviation 11: "built body" reads "request", its 350,000 characters of content counted as pi counts them, MADR ST87 as amended; ST105's and ST106's cases join it)* | red: 8192 *(2026-11-23: 60,000 at `f4efc5e`, since ST99 sends the fact; still a red on the assertion)* | the 4,096 removed: 31,072 |
| OQ15 (1) | `TestGenerateOverStream_0088` | ST88, ST88e | `Generate`, no caller limit: Claude, L 128,000: the request carries `"stream": true` and the call returns the whole response; OpenAI with a key, nothing sent: `"stream": true`; `WithMaxTokens(4000)`: a plain request | red: plain requests | the stream path skipped: a plain request at 128,000 |
| OQ15 (1) | `TestGenerateOverStream_Retry_0088` | ST88a | a stream refused at opening with 503: retried once; one failing after a text event: not resent | red | a resend after an event: two requests |
| OQ15 (1) | `TestGenerateOverStream_ReplyLimit_0088` | ST88b | a stream whose text events add up to more than 16 MiB: `ErrIncomplete` | red | no bound on the assembled answer: a response |
| OQ15 (1) | `TestGenerateOverStream_NoUsage_0088` | ST88c | a server that sends no usage chunk: usage zero | red | an invented usage: non-zero |
| OQ15 (1) | `TestGenerateOverStream_ErrorIn200_0088` | ST88d | a 200 whose first payload is C39's error: classified and, when raised, resent once at the room OQ14 decides | red | the error returned as an empty answer |
| OQ15 (2) | Phase 1's `Generate` cases (16,384) | ST88 | | | |
| OQ15 (3) | `TestDefaultLimit_Facts_0088`'s `Generate` cases at 10,666 | ST88 | L 128,000, `Generate`: 10,666 | red: 8192 | the 16,384 ceiling kept: 16,384 |
| OQ16 (1) | `TestCutAnswer_CompleteCalls_0088` (Phase 1b, as written) and `make api-check` | ST97, ST82 | the warning's kind is `WarningToolCallCut`; `apidiff` lists it beside `Settings.MaxTokensGiven` | | as Phase 1b's table |
| OQ16 (2) | the same test, kind text `tool_call_cut`; `make api-check` | ST97 | no new exported name; `apidiff` lists only `Settings.MaxTokensGiven` | | an exported constant added: `apidiff` reports it |
| OQ16 (3) | the same test without the warning | ST97 | no warning | | a warning emitted |
| OQ17 (1) | `TestDefaultLimit_CallerStatement_0088` | ST90 | OpenAI with a key: a models.dev document handed in as a caller's layer stating 128,000: no field; a layer named "models.json" stating 20,000: 20,000 | red | the source-name check removed: the models.dev case sends 128,000 |
| OQ17 (2) | the same test, other expectations | ST90 | both cases send their limit | | the check added: the models.dev case sends nothing |
| OQ17 (3) | none in this repository | ST90 | the SDK's rule is unchanged; gobble-cli's records carry the filter | | |

### Execution record entry (2026-11-22, third)

* Records only, after record review 2: the MADR's third amendment and this
  one; the second amendment's PLAN rows that mixed C39's and the split form,
  and that counted an empty last call complete, are marked replaced in place.
  No code, test, index row, staging or commit.

## Amendment (2026-11-22, fourth): OQ14 to OQ16 answered, OQ9 replaced by 0089, and the build order

Additions only. The MADR's fourth amendment records the owner's answers to
OQ14 ("we will be going with option 3, but only after the llmprovider-sdk
work related to model discovery and output limit work is complete."), OQ15
("Over the stream (Recommended)") and OQ16 ("Exported WarningToolCallCut
(Recommended)"), and the owner's reply on OQ9, which asked for model
output-limit discovery:
[0089-PLAN-model-output-limit-discovery.md](0089-PLAN-model-output-limit-discovery.md).
The MADR has eight open rows (ST43 under OQ18; ST74, ST85, ST86, ST90, ST92
and ST98 under 0089-MADR's OQ1; ST89 under its OQ3). *(Second draft, after
record review 3: ST90 reopened; ST89 moved to 0089's OQ3.)* This PLAN stays `proposed`.

### The build order, under OQ18 (1) or (2)

* **Phase 0 runs again** for OQ18 and 0089-MADR's OQ1: their answers become
  rows, and the MADR returns to `accepted` when none is open.
* **Phase 1b (the cut answer, OQ11)** depends on nothing in 0089: it is built
  first, with the tests the second and third amendments list and OQ16 (1)'s
  row (`WarningToolCallCut`, exported; `apidiff` lists it).
* **Phase 1 (the default and the resend)** is built in one commit with
  0089-PLAN's Phase 3, after 0089's Phases 1 and 2: the helper is 0089's
  (MADR D6), not the `DefaultLimit` step 4 names; its default-limit tests are
  0089-PLAN's "Tests by answer" for 0089's OQ1, which replace the first
  amendment's and the third's OQ9 rows; OQ15 (1)'s tests
  (`TestGenerateOverStream_0088` and ST88a to ST88e's) are in Phase 1; the
  live measurement for OQ10 runs before it, as the second amendment says.
* **Phase 1c (OQ14 (3), the estimate fit)** follows 0089-PLAN's Phase 3:
  `TestDefaultLimit_EstimatedFit_0088` (the third amendment's row), and
  0089's `TestOutputLimit_EstimateReadsLimits_0089` (its DL56: Claude, input
  1,000,000 stated and no window, a 350,000-character body: the limit held to
  1,000,000 less 100,000 less 4,096; plant: `Limits.Input` ignored, so no
  hold); plant of the first as there. The half-window hold
  (`TestDefaultLimit_WindowFit_0088`) is not built.
  *(2026-11-23, deviation 11: the "350,000-character body" is 350,000 characters of request content, MADR ST87 as amended; the hold shows only above a stated output of 895,904, so the test's layer states output 1,000,000 and input 1,000,000, no window, on a Claude model in no other source.)*
* **Phase 2 to Phase 4** run once, over this PLAN and 0089-PLAN together; the
  `apidiff` expectation is `Settings.MaxTokensGiven`, `WarningToolCallCut`
  and 0089-PLAN's list.

### Under OQ18 (3)

Phase 1 is built as written on 2026-11-21 (32,000 streamed; 8192 unknown;
half the window; a resend at 8192), with OQ15 (1)'s and Phase 1b's tests, and
released with the five; 0089 and Phase 1c follow in the next release, and
0089-PLAN's Phase 3 changes Phase 1's expectations as planned, not as a
deviation.

### Execution record entry (2026-11-22, fourth)

* Records only: the MADR's fourth amendment, this one, and the new 0089 pair.
  No code, test, index row, staging or commit.

## Amendment (2026-11-22, fifth): the release after the next tag, built with 0089

Additions only. The owner answered OQ18 "Five now; 0088+0089 next (Recommended)" and 0089-MADR's
OQ1 to OQ4 (recorded in the MADR's fifth amendment). The MADR is `accepted`;
this PLAN stays `proposed` until the owner approves it.

* **Order.** 0087's PLAN finishes first; the next tag carries 0081, 0083,
  0084, 0085 and 0087. This PLAN and 0089's are built after it: 0089's Phases
  1 and 2, its live measurement, then this PLAN's Phase 1b, then Phase 1 with
  0089's Phase 3 in one commit, then Phase 1c; Phases 2 to 4 run once over both
  PLANs. Phase 4 is the release after the next tag, not the next tag; its
  `apidiff` expectation is `Settings.MaxTokensGiven`, `WarningToolCallCut`,
  0089's list and `WithoutLimitLearning`.
* **Tests.** Phase 1's default-limit tests are 0089-PLAN's OQ1 (1), OQ2 (1)
  and OQ3 (1) rows (ST74, ST85, ST86, ST89, ST90, ST98, ST99); the third
  amendment's OQ9 (1) to (3), OQ14 (1) and (2) and OQ17 rows are not built;
  OQ15 (1)'s and OQ16 (1)'s are; ST92's is 0089's
  `TestLimitRefusal_WindowForms_0089`.
* **Execution record entry (2026-11-22, fifth):** records only.

### Execution record entry (2026-11-22): the owner's approval and start

The owner, relayed by the coordinator, approved this PLAN and 0089-PLAN and
the start, by the label "Approve both; start now (Recommended)". The status
went to `in-progress` by
`python3 -B tools/records.py status docs/decisions/0088-PLAN-default-output-limit-from-model-facts.md in-progress --date 2026-11-22`,
which wrote the front matter and the index row and exited 0
(`records: written; records-check: 0 problem(s)`). The build order of the
fifth amendment stands: 0089-PLAN's Phases 1 and 2 first, then its live
measurement, then this PLAN's Phase 1b, then Phase 1 with 0089-PLAN's Phase 3
in one commit, then Phase 1c. Nothing of this PLAN's phases is built yet; the
v1.32.0 tag is the owner's.

## Amendment (2026-11-23, sixth): Phase 1 built in 0089-PLAN's merged phase (deviation 6)

Additions only; every step above stands, read as this amendment says.

* **Deviation 6.** 0089-PLAN's Phase 2 stopped: three of its tests (DL35, DL37, DL43) assert the resend at a stated cap, which this PLAN's Phase 1 and 0089-PLAN's Phase 3 build, and the resend needs `Settings.MaxTokensGiven` (OQ5) to tell a given limit from the default. The owner's ruling, relayed by the coordinator: "Merge the phases". This PLAN's Phase 1 is built with 0089-PLAN's Phases 2 and 3 as one phase, in one commit:
  [0089-PLAN-model-output-limit-discovery.md](0089-PLAN-model-output-limit-discovery.md), "Merged phase: Phases 2 and 3 with 0088-PLAN's Phase 1". That section names which of Phase 1's tests are built, with which values (ST99), and which are not; its exports add `Settings.MaxTokensGiven` and no `WarningToolCallCut`.
* **OQ10's measurement first.** The owner's ruling (2026-11-23), relayed by the coordinator: "Measure first, then build (Recommended)". The first amendment's "OQ10 (1): the live measurement, before Phase 1" is the merged phase's step 0, run once, its refusal quoted in a dated MADR amendment; a message stating no numbers stops the build for a ruling on ST65's resend value.
* **Where the rest falls.** Phase 1b after the merged phase, in its own commit, before Phases 2 to 4 (no merged test depends on it); Phase 1c after it (it reads `catalog.Limits` and the merged rule); 0089's Kilo measurement gates no code; Phases 2 to 4 run once with 0089-PLAN's Phases 4 to 7.

### Execution record entry (2026-11-23, sixth)

* Re-grounding at `6ace750`: `python3 -B tools/claims.py docs/decisions/0088-MADR-default-output-limit-from-model-facts.md --rev self=6ace750`: 127 of 140 pass, exit 1; the 13 rows that moved, none gone, are in the MADR's sixth amendment, each at its line at `6ace750`.
* The merged phase's step 0 stopped before its run, on two findings put to the owner, recorded in 0089-PLAN's execution record of 2026-11-23: no Claude model before 4.5 is in the committed Claude listing or named by this pair (S1), and 241 G-wire goldens that record 8192 change under ST99 with no row naming them (S2). No live request was sent; nothing was built, staged or committed.
* The owner's rulings of 2026-11-23 on the merged phase's stop, relayed by the coordinator: S1 "Try claude-sonnet-4 once" (one run of the measurement on `claude-sonnet-4-20250514`; a not-found, or any refusal not in the window form, is the finding that no older model is served there, and ST65 is then ruled out of scope with that and the listing evidence); S2 "Same, plus explicit-8192 goldens" (a decided row naming the G-wire goldens as a planned change, checked by a script, and one golden per wire sent with an explicit `WithMaxTokens(8192)`). The measurement ran once: `LLMPROVIDER_LIVE_0088=1 LLMPROVIDER_LIVE_0088_MODEL=claude-sonnet-4-20250514 go test -tags live_gateways -count=1 -run '^TestLive_ClaudeWindowRefusal_0088$' -v -timeout 10m ./llmprovider/`, exit 0, HTTP 404 `not_found_error`; ST65 is out of scope and ST100 added (the MADR's sixth amendment); the merged phase drops `TestDefaultLimit_ClaudeOldWindowForm_0088` and gains ST100's goldens.

### Deviation 7 (2026-11-23): the merged phase built in ordered layers

* **The stop.** The merged phase's pass 1 built `Settings.MaxTokensGiven` (ST35, ST36) and its test, and stopped: the phase is more than one run can carry (ST88e: 335 `Generate` call sites in 120 test files move to the stream; ST100's 241 goldens; about 45 new tests over ten providers and OpenCode's four routes).
* **The owner's ruling**, relayed by the coordinator: "Ordered layers, one commit (Recommended)": the same phase and one commit, built as ordered layers, each its own run with a checkpoint: A, 0089's Phase 2 API, the memo, the validator and the pass-back; B, the request rule, the report and the resend, with Claude, Kilo and OpenCode's messages route; C, the optional wires and ST100's goldens (the diff-check script seen failing first, and the explicit-8192 goldens); D, ST88 and the ST88e test changes; E, this PLAN's remaining Phase 1 tests, the docs and all gates. The table "Layers" in [0089-PLAN-model-output-limit-discovery.md](0089-PLAN-model-output-limit-discovery.md)'s Merged phase section holds each layer's rows and status, and its execution record each checkpoint.
* **Pass counting.** A layer is not a build pass: only a rebuild after a failed gate or review counts toward the three-pass limit.

### Execution record entry (2026-11-23, seventh): ST101 and deviation 8

* The owner's rulings on layer B's stop, relayed by the coordinator: B1, first "Name it; add a caller layer (Recommended)", then, corrected after the build's probe showed a caller layer stating 8,192 sends 8,192 with a budget of 8,191, "Explicit WithMaxTokens(8192) (Recommended)": `TestThinkingWire_Claude`'s budget case gets the caller's `WithMaxTokens(8192)` (the MADR's seventh amendment, ST101). B2, "Merge layers B and C": layers B and C are one layer (0089-PLAN's deviation 8 and its Layers table).

### Execution record entry (2026-11-23, eighth): ST102

* The owner's ruling on the merged layer B+C's stop, relayed by the coordinator: "One class row, assert absent (Recommended)". The existing tests and goldens that pin the old 8,192 default on a route that now omits it are a planned change, the MADR's eighth amendment (ST102), its first member `TestTogether_RequestShapes`; each such assertion becomes "no limit sent", red at the base, with the plant "8192 sent" on that route.

### Execution record entry (2026-11-23, eighth): the merged layer B+C

* Built, unstaged, in [0089-PLAN-model-output-limit-discovery.md](0089-PLAN-model-output-limit-discovery.md)'s merged layer B+C, whose checkpoint records the runs: ST32's change to `TestOutputLimit_Facts_0041` with ST99's values, ST101's to `TestThinkingWire_Claude`, ST102's to `TestTogether_RequestShapes` (its one member), and ST100's goldens (234 regenerated, each differing from `6ace750` only in its output-limit field by the diff-check script, which was first seen failing on a planted unrelated difference; 21 explicit-8192 goldens added). Layers D and E follow.

### Execution record entry (2026-11-23, ninth): the merged layer D

* Built, unstaged, in [0089-PLAN-model-output-limit-discovery.md](0089-PLAN-model-output-limit-discovery.md)'s merged layer D, whose checkpoint records the runs: ST88 (a plain `Generate` whose limit the SDK chose or left out read over the service's stream and returned whole) with ST88a to ST88d, `TestGenerateOverStream_0088` and its four siblings, each red at `6ace750` and each plant caught; ST88e's changes to the existing `Generate` tests whose servers answered plain JSON (94 test files, by package and kind in that checkpoint); the 234 G-wire goldens in the stream form, checked by `tools/goldendiff.py --stream` (the diff-check moved into `tools/`). A5's first-payload resend, which ST88d needs, is built there; ST55's own test stays in layer E. ST102's list is unchanged. Layer E follows.

### Execution record entry (2026-11-23, tenth): ST51's guard tests and the plants with no site

* **The stop.** Layer E stopped at ST51: adding Kilo's documented form (C97) to `contextOverflowMessages` fails three existing guard tests of that table unless two test files change, which no row named (the MADR's ninth amendment quotes the failing lines). The owner's ruling, relayed by the coordinator: "Name the two test changes (Recommended)": one sample entry in `overflowSamples` (`llmprovider/context_overflow_test.go`) and the C97 pattern in `refContextOverflowMessages` (`llmprovider/context_overflow_ref_test.go`, whose generated header gains a dated line naming ST51). Recorded in the MADR's ninth amendment before the code.
* **The plants with no site in the merged code.** The owner's ruling, relayed by the coordinator: "Accept, noted in the record (Recommended)". Each replacement is caught only by its row's test failing (the run that row's spec names), and the merged phase's layer E checkpoint in [0089-PLAN-model-output-limit-discovery.md](0089-PLAN-model-output-limit-discovery.md) quotes each failing line.
  * *(2026-11-23)* ST23 (b) and ST48 (e), "the input-alone check removed": the merged code has no input-alone check to remove. The Messages wire reads no window form (0089-MADR DL68), so "prompt is too long" is never read for a resend; on Kilo, a window form whose input is at or over the window is held back twice, by V5 in `wire.ReadLimitRefusal` and by the room's floor in `LimitRetry.Check`, so removing either alone changes nothing (an equivalent plant). The replacement: `LimitRetry.Check` resends any overflow of a limit the SDK chose without reading it.
  * *(2026-11-23)* ST56 (b), "the raise undone after any refusal: case 2's second send is 8192": no 8192 resend exists since ST99. The replacement: `LimitRetry.Check` sets a sticky resend at 8192 on any refusal it cannot read.
  * *(2026-11-23)* ST61, "the resend runs on any overflow, raised or not": dropping `Check`'s sent and basis checks alone is caught by nothing, since V5 rejects the window form when no limit was sent (the input it implies is the whole total). The replacement: `LimitRetry.Check` resends any `ErrContextOverflow`.
  * *(2026-11-23)* ST57 (c), "the undoing send counted against the bound": planted in `internal/wire/sse.go`'s `StreamEventsRetry`, not `wire.Send`, since a `Generate` whose limit the SDK chose is read over the stream (ST88); `wire.Send`'s uncounted branch is no longer reached (a follow-up in 0089-PLAN).
* **ST56's test gains one case within its row:** case 1 again with a cap naming another model, which serves this request only (0089-MADR DL36), so that no memo masks the resend's stickiness. Plant (a) as first written was not caught: the cap kept in the memo bounded the third send to 64,000 by itself.

### Execution record entry (2026-11-23, eleventh): the merged layer E

* Built, unstaged, in [0089-PLAN-model-output-limit-discovery.md](0089-PLAN-model-output-limit-discovery.md)'s merged layer E, whose checkpoint records the runs: the rest of this PLAN's Phase 1 tests as the merged table builds them (ST23, ST26, ST46, ST48, ST50 to ST52, ST55 to ST57, ST59, ST61, ST63, ST91), each red at `6ace750` or green there as its row says, each green, and each plant caught, the replacements of the tenth entry included; ST51's form with its two planned test changes; the existing tests of Phase 1 step 2 (ST4, ST19, ST20, ST27, ST30, ST41, ST45, ST61), each with its plant caught; step 6's docs and the full checks. ST65 stays out of scope; the ST99-retired tests are not built.

### Execution record entry (2026-11-23, twelfth): Phase 1b started

* Phase 1b ("the cut answer, one way", OQ11), as the second amendment states it with the third's rows (ST93, ST94), the fourth's OQ16 (1) (`WarningToolCallCut`, exported) and the sixth's order (its own commit, after the merged phase, before Phase 1c and Phases 2 to 4). Base `247b745`, the merged phase committed by the owner; the tree clean at the start. Pass 1.
* *(2026-11-23)* **Stopped** at step 3, the code written and unstaged, before any plant, gate or doc: ST78's Messages cell ("its `tool_use` block closed and its input parsed") returns a call the limit cut on a plain Messages reply, since the service closes and parses the cut block's partial input there. The existing `TestDecode_StopReason` (`llmprovider/internal/wire/messages/answer_test.go:32`, `{"subject":"fix: tru"}` on `max_tokens`) and `TestFinish_Messages` (`llmprovider/internal/wire/messages/decode_0021_test.go:57`, `{"a":1}` on `model_context_window_exceeded`) then fail: `answer_test.go:34: a tool call cut by max_tokens = <nil>, want ErrIncomplete`; `decode_0021_test.go:60: model_context_window_exceeded with a call: <nil>; want ErrIncomplete, reason length`. Anthropic's stop-reasons page (read 2026-11-23) checks "if the last content block is an incomplete tool_use". Put to the owner; no row changed yet.

### Deviation 9 (2026-11-23): the owner's rulings on Phase 1b's stop

Recorded before more code; pass 1 continues (a ruling, not a new pass). Relayed by the coordinator, quoted as given:

* **ST78's Messages cell:** "Last tool block is the cut call (Recommended)". The MADR's tenth amendment adds ST103. Phase 1b's table gains, in `TestCutAnswer_CompleteCalls_0088`, a plain Messages case whose last block is `tool_use` with input `{"subject":"fix: tru"}` on `max_tokens`: dropped and named; plant: the last-block rule removed, caught only by that case. `TestDecode_StopReason` and `TestFinish_Messages` stand unchanged.
* **The builder's three code choices** (R1 the wire-general cut rule, R2 Responses' incomplete or never-done call, R3 `wire.HeldCalls`): "Defer to the review". Built as they are, each with a test and a plant; recorded in the MADR's tenth amendment as open for Phase 3. **Phase 3 (added 2026-11-23):** the reviewer rules on R1, R2 and R3 with the owner, each named and described in the MADR's tenth amendment.
* **Claims rows on uncommitted text** (ST83's annotations, ST84's changelog entry): the sixth amendment's "now" id convention, the coordinator's decision (this record's existing convention), each checked with plain ids on a scratch clone's commit of the tree and planted false there.
* **Open for the joint Phase 2 (added 2026-11-23):** the merged phase added no `tools/claims.py` rows for its own changelog entries, which ST43 names ("a `tools/claims.py` row on each entry, planted false"). Not built in Phase 1b.

### Execution record entry (2026-11-23, thirteenth): Phase 1b built (pass 1, then pass 2 after the coverage gate)

Base `247b745`; unstaged; no live call, every reply the tests' own. Pass 1 continued after deviation 9; pass 2 is the rebuild after `coverage-check` failed (below).

**Re-ground (step 1), at `247b745`:** each wire's decoder and stream reader: Chat Completions `response` (`chatcompletions.go:514`, C128 at `:533`) and `Events`/`done`; Messages `Decode` (`messages.go:310`, C129 at `:349`) and `Events` (`:440`); generateContent `Decode` (`generatecontent.go:179`, C132 at `:241`) and `Events` (`:298`); Gemini's Interactions `decodeInteraction` (`interactions.go:128`, C136 at `:167`) and `interactionEvents` (`:261`); Responses `decodeAs` (C130 at `responses.go:413`) and `readEvents` (C134 at `:700`); `wire.EmptyAnswer` (`finish.go:34`). `tools/claims.py` on the MADR at `self=247b745`: 82 of 140 (the rows Phase 1b rests on, C128 to C134, C136, C137, all pass; the others moved or were rewritten by the merged phase); at their pins, 140 of 140. The existing tests that pinned `ErrIncomplete` for a cut answer, found by the suite on the new code: `TestReadStream_FailureReturnsNoResponse` (`llmprovider/internal/wire/responses/decode_0021_test.go:41`, its incomplete case now a content filter, ST81) and `TestGeminiInteractions_Status` (`llmprovider/providers/gemini/interactions_test.go:154`, now a response finishing `length`, ST93), changed as ST84 plans; `TestDecode_StopReason` and `TestFinish_Messages` stand unchanged under ST103. Every other `ErrIncomplete` check of a cut answer, llmtest's W12 `Truncated` and `ReasoningCut` on all ten harnesses included, passes unchanged, since `ErrEmptyAnswer` matches `ErrIncomplete`.

**The code.** `wire.CutAnswer`, `CutCall`, `ToolCallCut`, `WholeObject` and `HeldCalls` (`llmprovider/internal/wire/finish.go`); each decoder and stream reader returns a cut answer's usable part through `CutAnswer` (Chat Completions, Messages with ST103's last block, generateContent, Interactions, Responses with its incomplete or never-done call); `llmprovider.WarningToolCallCut`; the docs of `Response.FinishReason`, `FinishLength`, `ErrIncomplete`, `ErrEmptyAnswer` and `APIError`.

**Tests**, in `llmprovider/providers/cut_answer_0088_test.go` and `llmprovider/internal/wire/cut_0088_test.go` (new):

| Test | Rows | At `247b745` |
| :--- | :--- | :--- |
| `TestCutAnswer_Text_0088` | ST75, ST93, ST95 | red: `cut_answer_0088_test.go:449: responses, Generate: 1 request(s), resp <nil>, err llmprovider: incomplete response: max_output_tokens; want one request, a response finishing length with hello` (and Interactions) |
| `TestCutAnswer_CompleteCalls_0088` | ST76 to ST78, ST93, ST94, ST103 | dropped (`WarningToolCallCut`); its red is its plants |
| `TestCutAnswer_NothingUsable_0088` | ST79, ST93 | red: `:500: chat completions, Generate, only a cut call: resp <nil>, err llmprovider: incomplete response: length; want ErrEmptyAnswer with reason "length" and no response` (and every wire) |
| `TestCutAnswer_Stream_0088` | ST80 | red: `:522: chat completions: items [], EventDone false, resp <nil>, err llmprovider: incomplete response: length; want the item lookup alone, then EventDone finishing length` (every wire) |
| `TestCutAnswer_OtherIncomplete_0088` | ST81 | green (`--expect pass --rev 247b745`, with `TestDecode_StopReason` and `TestFinish_Messages`: 3 of 3) |
| `TestCutAnswer_WireGeneral_0088`, `TestCutCall_0088` | R1 (open) | new API, dropped; red is the plants |
| `TestCutAnswer_ResponsesCutItem_0088` | R2 (open) | the same |
| `TestCutAnswer_HeldCalls_0088`, `TestHeldCalls_0088` | R3 (open) | the same |
| `TestCutAnswer_Helper_0088`, `TestToolCallCut_0088` | ST77, ST79 (the helpers in their package) | the same |

`tools/redgreen.py --expect fail --rev 247b745`: 3 of 3. `--expect pass` on the tree, the sixteen tests above and the four existing ones: 16 of 16.

**Plants**, by `tools/plant.py` on scratch clones, one spec per test with `-run` naming that test alone, 22 plants (21 by `tools/plant.py`, ST82's by `apidiff`; *2026-11-23, corrected from "25" by deviation 10's run, which counted them*), each caught by a failing assertion, none by a build failure (a first draft of R3's `Done` plant broke the build and was rewritten):

| Plant | Test | Its failing line |
| :--- | :--- | :--- |
| ST75 Responses' cut answer left an error | `Text` | `:449: responses, Generate: 1 request(s), resp <nil>, err llmprovider: incomplete response: max_output_tokens; ...` |
| ST93 Interactions left an error | `Text` | `:449: interactions, Generate: 1 request(s), resp <nil>, err llmprovider: incomplete response: incomplete; ...` |
| ST95 a continuation request sent | `Text` | `:449: interactions, Generate: 2 request(s), resp &{ID:int_1 ...FinishReason:length ...` |
| ST76 (a) the complete calls dropped | `CompleteCalls` | `:478: chat completions, Generate: calls []; want ["lookup {\"x\":1}"], the cut call dropped` |
| ST77 (b) the warning omitted | `CompleteCalls` | `:478: chat completions, Generate: warnings [{Kind: Model:m Detail:the output limit cut the call to "fetch" (id "call_b") ...` |
| ST78 (c) the Chat Completions completeness check removed | `CompleteCalls` | `:478: chat completions, Generate: calls ["lookup {\"x\":1}" "fetch {\"path\":\"a\""]; want ["lookup {\"x\":1}"], the cut call dropped` |
| ST94 (d) the empty last call kept | `CompleteCalls` | `:494: chat completions, the empty last call, Generate: calls ["lookup {\"x\":1}" "now {}" "list {}"]; ...` |
| ST103 the last-block rule removed | `CompleteCalls`, only its ST103 case | `:487: messages, the last block parsed (ST103), Generate: calls ["lookup {\"x\":1}" "commit {\"subject\":\"fix: tru\"}"]; ...` (and Stream) |
| ST79 an empty response returned | `NothingUsable`; `Helper` | `:542: chat completions, Generate, only a cut call: resp &{ID:c Model:m Output:[] FinishReason:length ...`; `cut_0088_test.go:44: only reasoning: &{...FinishReason:length ...` |
| ST80 the stream ends with `ErrIncomplete` | `Stream` | `:564: chat completions: items [], EventDone false, resp <nil>, err llmprovider: incomplete response: length; ...` |
| ST81 a content-filter incomplete returned as a response | `OtherIncomplete` | `:587: responses, Generate, a content filter: resp &{ID:resp_1 Model:gpt-5 Output:[{Role:assistant Text:hello}] FinishReason:length ...` |
| R1 (a) the last-empty rule removed | `WireGeneral`; `CutCall` | `:605: generatecontent, a trailing call with no arguments, Generate: calls ["lookup {\"x\":1}" "now {}"]; ...`; `cut_0088_test.go:61: CutCall("{}", last true) = false; want true` |
| R1 (b) any JSON taken as whole | `WireGeneral` | `:610: chat completions, arguments not an object, Generate: calls ["list [1]" "lookup {\"x\":1}"]; ...` |
| R2 (a) the incomplete-status check removed | `ResponsesCutItem` | `:624: responses, an incomplete call item, Generate: calls ["lookup {\"x\":1}" "fetch {\"y\":2}"]; ...` |
| R2 (b) a call added and never done not named | `ResponsesCutItem` | `:624: responses, an incomplete call item, Stream: warnings []; want one WarningToolCallCut naming fetch and call_b` |
| R3 no call held | `HeldCalls` (providers) | `:639: responses, Stream: items ["first" "lookup" "now"]; want first, then lookup, and never the last {} call` |
| R3 `Later` releases nothing | `TestHeldCalls_0088` | `cut_0088_test.go:100: Later yielded []; want now alone, the partial call kept` |
| R3 `Done` yields a dropped call | `TestHeldCalls_0088` | `cut_0088_test.go:109: Done yielded ["now"]; want nothing: the response dropped the call` |
| ST77 the name-only detail loses the name | `TestToolCallCut_0088` | `cut_0088_test.go:78: ToolCallCut("lookup", "") = {Kind:tool_call_cut ... Detail:the output limit cut a call before ...` |
| ST82 a third name exported | `tools/apidiff.py v1.32.0` against the expected list | `Error: apidiff reports ./llmprovider.WarningPlanted0088: added, beyond the expected list` |

**`apidiff`** (`python3 -B tools/apidiff.py v1.32.0`), exit 0: the merged phase's twenty lines and `./llmprovider.WarningToolCallCut: added`, nothing else.

**Checks**, each exit status taken before any filter:

| Command | Host | Exit |
| :--- | :--- | :--- |
| `go build ./...`; `go vet ./...`; `go vet -tags live_gateways ./...` | session (darwin/arm64) | 0; 0; 0 |
| `run-heavy -- make pre-add-check FILES="<the phase's 14 Go files>"` | session | 2 first (golangci-lint: `errcheck` on `HeldCalls.Later`'s type assertion, `gocognit` on Messages' `Events` (60) and Responses' `readEvents` (54), `nilerr` in Messages' `Events`; fixed without a suppression: `HeldCalls` holds `FunctionCallItem`s, `callHold` and `message.finish` in Messages, `streamEnd` in Responses); then 0, `14 file(s) clean` |
| `run-heavy -- go test -count=1 ./...` | session | 0, 46 packages ok |
| `run-on-host --need linux --race -- go test -race -count=1` (`llmprovider`, `internal/wire/...`, `providers/...`) | a fleet Linux host (linux/amd64), not WSL | 0, 18 packages ok; `internal/wire` again after its new test, 0 |
| `make parity-check dep-check coverage-check api-check contract-check generate-check records-check tables-check fixtures-check changelog-check docs-check markdownlint links-check disclosure-check gate-selftest tools-test`, each alone, through `run-heavy` | session | first run: `coverage-check` 2 and `gate-selftest` 2 (`llmprovider/internal/wire` at 77.0 %, below its 80 % floor: the new helpers had no test in their package); pass 2 added `cut_0088_test.go` (82.2 %); then 0 each |
| `tools/goldendiff.py` | | not run: no golden changed |
| `python3 -B tools/claims.py` on the MADR | session | 0, 140 of 140 |
| the "now" rows C144now to C148now, plain ids, on a scratch clone's tree of the working tree | session | 0, 5 of 5; `--plants`, all 5 caught |
| `python3 -B tools/recordlint.py --changed`; `python3 -B tools/denyscan.py --changed` | session | 0 (no rule rose); 0 (0 hits) |

**Docs.** `README.md` (the cut answer, its warning and its reasons); `docs/guides/api-standards.md` R7; `docs/guides/adding-a-provider.md` (`Truncated`); `docs/architecture.md` (`internal/wire`'s row, the answers bullet, the conformance `Truncated` line); `CHANGELOG.md` `[Unreleased]` **Added** (`WarningToolCallCut`) and **Changed** (the cut answer); the dated annotations of 0012-MADR §1.5 and 0020-MADR F3 (ST83).

**Not done:** Phases 1c and 2 to 4; R1 to R3 wait for Phase 3's review; ST43's claims rows wait for Phase 2.

### Deviation 10 (2026-11-23): a Responses stream cut with no `.done` events; Phase 1b pass 3

* **Evidence**, a live probe (round 7, allowlisted output only), relayed by the coordinator: on xAI `grok-4.20-0309-non-reasoning` with `max_output_tokens` 16, a streamed text cut sends `response.created`, `response.in_progress`, `response.output_item.added`, `response.content_part.added`, `response.output_text.delta`, then `response.incomplete` (reason `max_output_tokens`), with no `output_text.done`, `content_part.done` or `output_item.done`, its final message item `incomplete`; a plain text cut has the top-level `status` `incomplete` and its message item `completed`. OpenAI (`gpt-4o-mini`) sends every `.done` event, the item `incomplete`. `llmprovider/internal/wire/responses/responses.go` (`readEvents`, lines 736 to 752 of the tree after pass 2) built a stream's `result.Output` only from `output_item.done`, so a cut streamed Grok answer had nothing usable, against ST75 and ST80. xAI did not enforce `max_output_tokens` on a forced call (1042 tokens against 160): no Grok call cut was observed.
* **The owner's ruling**, relayed by the coordinator: "Fix it now in Phase 1b (Recommended)". The MADR's eleventh amendment adds ST104. This is Phase 1b's pass 3; a further failure is a stop, not a fourth pass.
* **Added to Phase 1b's table:** `TestCutAnswer_XAIShapes_0088` (ST104), replaying the measured stream and plain shapes, with its plants (the terminal response's items ignored; the cut decided by an item's status).

### Execution record entry (2026-11-23, fourteenth): Phase 1b pass 3 (ST104)

* **The code.** `llmprovider/internal/wire/responses/responses.go`: `outputItem.ID`; the terminal event's `Response.Output`; the stream's state as `streamState`, with `itemDone`, `delivered`, `takeTerminal` and `end`: on a cut, each item the terminal `response.incomplete` carries that no `output_item.done` delivered (by id, or by place for an item with none) is taken under the cut rules, a call it names no longer an open one, and a kept item is yielded before `EventDone`. The plain reader already decided the cut by the top-level `status`.
* **The test**, `TestCutAnswer_XAIShapes_0088` in `llmprovider/providers/cut_answer_0088_test.go`, on Grok: xAI's measured stream (no `.done` event) and plain shape (message item `completed`); and a stream whose terminal response carries a whole call and an incomplete one, never done (not measured). **Red** at the tree before this pass (its `responses.go` rebuilt from this pass's edits on a scratch clone, every other cut test there green): `cut_answer_0088_test.go:694: grok, Stream, xAI's cut: resp <nil>, EventDone false, err llmprovider: empty answer: max_output_tokens: responses: the answer has no content; want a response finishing length with its text`, `:705: grok, Stream, terminal calls: items []; want lookup alone, yielded before EventDone`; the plain case is green there, as the plain reader already met it. **Green** on the tree.
* **Plants**, each caught only by `TestCutAnswer_XAIShapes_0088`: (a) the terminal response's items ignored, `:694` as above; (b) the cut decided by an item's status, `:694: grok, Generate, xAI's cut: resp &{... Output:[{Role:assistant Text:Counting: 1, 2, 3,}] FinishReason:stop ...`; (c) a taken call left open, `:707: grok, Stream, terminal calls: warnings [{Kind:tool_call_cut ... "fetch" (id "call_b") ...}, ...]`, named twice; (d) the taken items not yielded, `:705: ... items []`. Every earlier plant re-run on this tree: all caught (25 by `tools/plant.py`, ST82's by `apidiff`).
* **Checks**, each exit taken before any filter: `go build`, `go vet`, `go vet -tags live_gateways`: 0, 0, 0; `run-heavy -- make pre-add-check FILES="<the 14 Go files>"`: 0, `14 file(s) clean`; `run-heavy -- go test -count=1 ./...`: 0, 46 packages; `run-on-host --need linux --race -- go test -race -count=1 ./llmprovider/internal/wire/... ./llmprovider/providers/...` on a fleet Linux host (linux/amd64), not WSL: 0, 17 packages; CI's set, each target alone through `run-heavy`: 0 each; `tools/apidiff.py v1.32.0`: 0, unchanged (21 lines); `tools/claims.py` on the MADR: 140 of 140; the "now" rows: 5 of 5, plants 5 caught; `tools/recordlint.py --changed` and `tools/denyscan.py --changed`: 0, 0.

### Execution record entry (2026-11-23, fifteenth): Phase 1c started, and stopped at its re-grounding

* **Start.** Phase 1c ("OQ14 (3), the estimate fit"), as the fourth amendment's bullet states it, with the third amendment's `TestDefaultLimit_EstimatedFit_0088` row and 0089-PLAN's `TestOutputLimit_EstimateReadsLimits_0089` (0089-MADR DL56), in the sixth amendment's order (after Phase 1b, before Phases 2 to 4). Base `f4efc5e`, Phase 1b committed by the owner; `git status` clean at the start. Pass 1.
* **Re-ground (step 1).** `tools/claims.py` on 0088-MADR at its pins: 140 of 140; on 0089-MADR: 49 of 49; on 0088-MADR at `self=f4efc5e`: 67 of 140, the rows the merged phase and Phase 1b moved, none that Phase 1c rests on. The limit rule at `f4efc5e`: `wire.LimitInput` and `wire.ChooseLimit` (`llmprovider/internal/wire/limit_rule.go:17`, `:51`) choose a request's value from the request, the option, the output fact (`catalog.Limits.Output` or `StreamedOutput`, already held to `Limits.Window`, `llmprovider/catalog/limits.go:450`) and the route's fallback; `LimitInput` carries no window, input limit or estimate. Its three callers build the body after choosing: Claude (`llmprovider/providers/claude/limit.go:48`, `claude.go:427`), Kilo (`kilo.go:528`, `:356`) and OpenCode's messages route (`opencode.go:1056`, `:595`). Claude's budget thinking shape raises the chosen value above the budget (`llmprovider/internal/wire/messages/messages.go:731`). At `f4efc5e` the third amendment's row sends L (60,000), not 8192: its "red: 8192" predates ST99; the red is still on the assertion.
* **Stopped** at step 2, before any test or code: ST87 (MADR line 1205) estimates "from the body it builds (3.5 characters a token, 4,096 spare, pi's)", and OQ14 (3) (MADR line 1137) "from the request it builds, as pi does". pi does not count a body: it counts the text of each message, its reasoning text, each call's name and arguments and the system text and tools, each image at 4,800 characters (`pi:packages/ai/src/utils/estimate.ts:15`, `:16`, `:34`, `:46` to `:68`). This SDK's body carries an image's bytes base64-encoded (`llmprovider/item.go:82`; `llmprovider/internal/wire/messages/messages.go:121`): a 1 MiB image is 1,398,104 characters, 399,459 tokens at 3.5, above a 200,000-token window, so a body-length estimate holds the limit to the wire's minimum, 1 (`MessagesLimits`, `llmprovider/internal/wire/limit_refusal.go:60`). No row decides how an image, a reasoning signature or the body's own JSON is counted; two further cases no row decides: whether the route's fallback (0089-MADR DL51) is held as the fact is (0089-MADR D6 names the fact only), and whether the hold applies before or after the budget shape's raise. Put to the owner; no row changed, nothing written but this entry.

### Deviation 11 (2026-11-23): the owner's rulings on Phase 1c's stop

Recorded before any test or code; pass 1 continues (a ruling, not a new pass). Relayed by the coordinator, quoted as given; the MADR's twelfth amendment holds the rows.

* **Q1, how the estimate counts:** "Count content, as pi does (Recommended)". ST87 is amended: the request's content, counted as pi counts it, each image at 4,800 characters, no signature, encrypted form or JSON overhead; the third amendment's "built body" reads "request". `TestEstimateInput_0088` (new, `llmprovider/internal/wire`) pins the count, with plants: an image counted by its base64; a reasoning signature counted; a deferred tool counted.
* **Q2, the fallback:** "Hold the fallback too (Recommended)". ST105: the route's fallback is held as the fact is; a fallback case joins `TestDefaultLimit_EstimatedFit_0088`, plant: the fallback not held.
* **Q3, the budget raise:** "Before the raise (Recommended)". ST106: the hold applies before `AddThinkingFor`'s raise; below 4,096 the default budget sends 8,192; a budget case joins `TestDefaultLimit_EstimatedFit_0088`, plant: the hold applied after the raise.
* **Recorded, not built:** R4 (the MADR's twelfth amendment), `ChooseLimit` sending the fallback without ST91's window cap (`llmprovider/internal/wire/limit_rule.go:66` to `:68`), for the joint Phase 3 review; the annotations on the third amendment's row (its red is 60,000 at `f4efc5e`) and on the fourth amendment's DL56 bullet (a stated output above 895,904).
* **Files added to the phase's scope:** `llmprovider/internal/wire/estimate_0088_test.go` (new); the providers' `limitInput` (Claude, Kilo, OpenCode) gain the window and the estimate.

### Execution record entry (2026-11-23, sixteenth): Phase 1c pass 1 after deviation 11, stopped at the suite

* **Tests, red at `f4efc5e`** (`tools/redgreen.py --expect fail --rev f4efc5e`, 2 of 2, each on its assertion): `TestDefaultLimit_EstimatedFit_0088` (`default_limit_0088_test.go:757: claude: sent [60000] ...; want one send at 26976`, and so each case: Kilo and OpenCode 60,000, the image case 60,000, the fallback case 64,000, the budget case 16,000); `TestOutputLimit_EstimateReadsLimits_0089` (`output_limit_0089_test.go:425: stream false: sent [1000000]; want one send at 895904`). `TestEstimateInput_0088` and `TestFit_0088` test new code; their red is their plants.
* **The code**, unstaged: `llmprovider/internal/wire/estimate.go` (new: `EstimateInput`, `fit`); `wire.LimitInput` gains `Window`, `Estimate`, `Minimum`, `ChooseLimit` holds the fact and the fallback, `LimitRetry.Choose` sets the wire's minimum; Claude's, Kilo's and OpenCode's `limitInput` pass the window (else the input limit) and the estimate. The four new tests green on the session host.
* **Plants**, by `tools/plant.py` on scratch clones, each run naming its test alone, 11 of 11 caught on an assertion: the 4,096 removed (`:757: claude: sent [31072]`); an image by its base64 (`:757: claude, a 1 MiB image: sent [1]`); Kilo's window not read and OpenCode's (`sent [60000]`); the fallback not held (`sent [64000]`); the hold after the raise (`sent [1904]`); `Limits.Input` ignored (`output_limit_0089_test.go:425: ... sent [1000000]`); a signature counted (`estimate_0088_test.go:43: ... = 2002; want 2`); a deferred tool counted (`= 2014; want 6`); bytes counted (`= 4; want 2`); the wire's minimum ignored (`estimate_0088_test.go:65: ... fit = 1; want 16`).
* **Stopped** at the suite (`run-heavy -- go test -count=1 ./...`, exit 1, 45 packages ok, one failing): the existing `TestDefaultLimit_WindowCap_0088` (ST91), its first case, `default_limit_0088_test.go:246: kilo, L 16,384, W 8,192: sent [4095]; want 8192`. ST87 holds every value the SDK chose to the window less the estimate less 4,096, always below the window, so ST91's cap on a fact can no longer be seen where a window is stated; the third amendment's ST91 row ("8,192 sent, whatever OQ9 and OQ14 answer") cannot be met under ST87, and no row names this change. Put to the owner; no gate beyond the suite, no doc and no record row written yet.

### Deviation 12 (2026-11-23): the owner's ruling on ST91's first case

Recorded before the change; pass 1 continues (a ruling, not a new pass). Relayed by the coordinator, quoted as given: "Expect 4,095; add a cap case (Recommended)". `TestDefaultLimit_WindowCap_0088`'s first case expects the estimate's room, 4,095, naming ST87; a new case, a caller's `WithMaxTokens(20000)` with L 16,384 and W 8,192, sends 8,192 (ST91's ninth-amendment annotation); the window-cap plant moves to the new case and must be caught only by it (16,384). The MADR's thirteenth amendment annotates ST91 and records R5 (no test but `TestFit_0088` reaches the Responses wire's minimum of 16) for the joint Phase 3 review.

### Execution record entry (2026-11-23, seventeenth): Phase 1c built (pass 1, after deviations 11 and 12)

Base `f4efc5e`; unstaged; no live call, every reply the tests' own.

**Rows to tests:**

| Row | Test | At `f4efc5e` | On the tree |
| :--- | :--- | :--- | :--- |
| ST87 (amended), its routes and the image | `TestDefaultLimit_EstimatedFit_0088` (Claude, Kilo, OpenCode's messages route, the 1 MiB image case) | red on the assertion: 60,000 each (sixteenth entry) | green: 26,976 each, 25,604 with the image |
| ST87's count | `TestEstimateInput_0088`, `TestFit_0088` (`llmprovider/internal/wire/estimate_0088_test.go`) | new code; red is its plants | green |
| ST105 | `TestDefaultLimit_EstimatedFit_0088`, the fallback case | red: 64,000 | green: 35,904, basis fallback |
| ST106 | `TestDefaultLimit_EstimatedFit_0088`, the budget case | red: 16,000 | green: 8,192, `budget_tokens` 4,096, reported 8,192 |
| DL56 | `TestOutputLimit_EstimateReadsLimits_0089`, `Generate` and `Stream` | red: 1,000,000 | green: 895,904 |
| ST91 (deviation 12) | `TestDefaultLimit_WindowCap_0088`, case 1 changed to 4,095, the cap case added (8,192) | not run there: case 1 sends 8,192 at the base, and the cap case's option cut is already built | green |

**Plants**, 12 of 12 by `tools/plant.py` on scratch clones, each spec running its test alone, each caught on an assertion: the eleven of the sixteenth entry, and ST91's window cap removed (`llmprovider/catalog/limits.go`, the window hold in `observedOutput`), caught only by the new cap case: `default_limit_0088_test.go:251: kilo, WithMaxTokens(20000), L 16,384, W 8,192: sent [16384]; want 8192`.

**`apidiff`** (`python3 -B tools/apidiff.py v1.32.0`), exit 0: the same 21 lines as Phase 1b's; `EstimateInput` is in `internal/wire`, not exported. **Goldens:** none changed, so `tools/goldendiff.py` was not run.

**Checks**, each exit taken before any filter:

| Command | Host | Exit |
| :--- | :--- | :--- |
| `go build ./...`; `go vet ./...`; `go vet -tags live_gateways ./...` | session (darwin/arm64) | 0; 0; 0 |
| `run-heavy -- make pre-add-check FILES="<the phase's 10 Go files>"` | session | 0, `10 file(s) clean` |
| `run-heavy -- go test -count=1 ./...` | session | 0, 46 packages ok (before and after the doc comments) |
| `run-on-host --need linux --race -- go test -race -count=1 ./llmprovider/internal/wire/... ./llmprovider/providers/...` | a fleet Linux host (linux/amd64), not WSL | 0, 17 packages ok |
| `run-heavy -- make -k parity-check dep-check coverage-check api-check contract-check generate-check records-check tables-check fixtures-check changelog-check docs-check markdownlint links-check disclosure-check gate-selftest tools-test` | session | 0; `internal/wire` coverage 82.6 % against 80 % |
| `python3 -B tools/claims.py` on 0088-MADR and 0089-MADR | session | 0, 140 of 140 and 49 of 49 |
| `python3 -B tools/recordlint.py --changed`; `python3 -B tools/denyscan.py --changed` | session | 0; 0, run after this entry |

**Docs.** The doc comments of `wire.LimitInput` and `wire.ChooseLimit`, of `Request.MaxOutputTokens` (`llmprovider/contract.go`), and the package docs of Claude, Kilo and OpenCode; `docs/guides/api-standards.md` R6 (the hold) and R57 (the estimate's numbers and their provenance); `docs/architecture.md` (`internal/wire`'s row); `docs/guides/adding-a-provider.md` (the output limit step); `README.md` ("Output limits"); `CHANGELOG.md` `[Unreleased]` **Changed** (the estimate fit).

**Not done:** Phases 2 to 4; R1 to R5 wait for Phase 3's review; ST43's claims rows on the changelog entries wait for Phase 2, as deviation 9 says.

## Amendment (2026-11-24, seventh): v1.33.0 re-scoped, Phase 3 a short release review

Additions only; every step above stands, read as this amendment says.

* **The owner's rulings of 2026-11-24**, relayed by the coordinator, quoted as given: "5 now, converge to 1 (Recommended)" (gobble-cli sends each model's known output cap from its own catalog now; this SDK's next release is fast-tracked in parallel; gobble-cli then owns the per-turn limit on this SDK's facts, in gobble-cli's own records), and "Yes, re-scope v1.33.0 (Recommended)".
* **What v1.33.0 carries.** It is cut from SDK main as it stands at `33e6f09`: this PLAN's Phase 1 as built in 0089-PLAN's merged phase, Phase 1b (the cut answer) and Phase 1c (the estimate fit); 0089-PLAN's Phases 1 to 3; 0087-MADR A-8's wizard fix (`c5b9912`); 0090's fix and tests; 0091's records and the docs of its Phase 1. It is a minor version: `apidiff` against `v1.32.0` lists 21 added names at `33e6f09`.
* **A scope addition (2026-11-24).** The owner, relayed by the coordinator, quoted as given: "scope this into v1.33.0: It also found something for the SDK release. When a service refuses a limit the caller set, the SDK learns the cap but never reports it, since that warning only travels on a successful reply." v1.33.0 also carries the caller-limit refusal report, its record to follow as a 0089-MADR amendment, built as its own phase after this release preparation; Phase 2's gates run again on that phase's commit before the tag, and its changelog entries join `[Unreleased]`.
* **What moves to v1.34.0.** R1 to R5 (the MADR's tenth, twelfth and thirteenth amendments; R1 to R3 are the three cut-answer choices Phase 1b deferred) are not ruled before v1.33.0: the behaviour ships as built, each with its tests and plants (the MADR's fourteenth amendment). 0091-PLAN's Phase 2 (`chatcompat.WithResponses()`) moves too (that PLAN's amendment of this date), and so does 0092 (the note below).
* **Phase 2** runs as written, over this PLAN's tables and 0089-PLAN's (its Phases 4 and 5), at `33e6f09`, step 8's ST43 rows included.
* **Phase 3** is a short release review, not the full review steps 2 and 3 describe: a reviewer who did not build the phases reads the diff from `v1.32.0` to the release commit and reports blockers only. A blocker goes to the owner as step 3 says; a finding that is not a blocker is recorded as a follow-up for v1.34.0. Step 4 (R1 to R3) moves to v1.34.0 with R4 and R5.
* **Phase 4** releases v1.33.0. Step 2's `apidiff` runs against `v1.32.0`, the newest tag, and expects the 21 lines the merged phase and Phase 1b recorded, nothing else, beside any the caller-limit refusal report's record names. `CHANGELOG.md`'s `[Unreleased]` becomes `[v1.33.0] - <date>`, dated by the release commit's day, with `README.md`'s "current release" line; this step is pending until the caller-limit refusal report's entries are in (the coordinator, 2026-11-24). *(2026-11-24: before the date step also come 0089's DL70 extract regeneration, the owner's "One fetch, then ship (Recommended)", and DL65's Kilo measurement, the owner's "Run before the tag", then the gates again; 0089-PLAN's entry of 2026-11-24 lists the remaining steps in order.)* The tag and the GitHub Release each wait for the owner's word in the same turn, as written.
* **0092 (a note).** `0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md` (`accepted`) and its PLAN (`proposed`) are drafted outside this tree (`<scratch>/m0092/`) and are not placed by this amendment. They target v1.34.0, where their drafts say v1.33.0 (the owner's "Own v1.33.0 phase (Recommended)", now superseded by "Yes, re-scope v1.33.0 (Recommended)"); placing them is their own Phase 0, which records the move by dated amendment.

### Execution record entry (2026-11-24, eighteenth): Phase 2, the joint gates, pass 1

Base `33e6f09`, the tree clean at the start (`git status --porcelain` empty); no code changed; no live call. Run with 0089-PLAN's Phases 4 and 5 as the seventh amendment says. Scripts and outputs: `<scratch>/rel133/`.

**Records (step 1 of the brief).** This PLAN's seventh amendment; the MADR's fourteenth (R1 to R5 to v1.34.0; ST43's rows); 0089-PLAN's entry of 2026-11-24 and 0089-MADR's amendment of 2026-11-24 (DL59's rows); 0091-PLAN's amendment and 0091-MADR's sixth (ST30); the 0092 note. No `docs/README.md` row changes: no title or status moved (`make records-check`: 193 records, 0 problems).

**Plants (step 2).** Every `tools/plant.py` spec the build phases wrote (0089-PLAN Phase 1, the merged phase's layers A to E, Phase 1b, Phase 1c), run again on scratch clones of the tree at `33e6f09`: 73 specs, 182 plants. 173 matched and were caught as written. 9 no longer matched once (8 sites, one plant in two specs), their text reworded by later layers and by Phases 1b and 1c; each was re-grounded to the same production change at the text `33e6f09` holds and caught on an assertion: DL19's window hold (`catalog/limits.go`, `observedOutput`), `TestDiscover_OutputAboveWindow_0089`; DL40 (d) the base URL ignored (`rememberedEvidence`), `limit_observations_0089_test.go:53`; DL39 the cap kept (`limitmemo.go`), `limitmemo_0089_test.go:47`; DL51 (d) 8192 sent where nothing states a limit (`limit_rule.go`), `output_limit_fallback_0089_test.go:196`; DL55 and DL67 (f) the pre-raise value reported (Claude, OpenCode's messages route), `output_limit_0089_test.go:383` (a first re-grounding left `sent` unused and broke the build; rewritten, then caught on the assertion); ST32/ST99 a 2026-11-21 value kept, `facts_0041_test.go:102: claude/claude-sonnet-5, a limit of 128000: sent 8192 ... want 128000`; ST27 the Chat Completions empty check removed, now the decoder's cut path skipping `wire.CutAnswer` (Phase 1b moved the check there), `empty_0061_test.go:21: cut while still reasoning: <nil>; want an error matching ErrEmptyAnswer and ErrIncomplete`. **182 of 182 caught, none by a build failure.** ST82's `apidiff` plant caught. `make changelog-check`'s plants: the newest section's heading undated, and removed: 2 of 2 caught.

**Claims rows (step 8; 0089-PLAN Phase 4).** ST43: 12 rows (C149now to C160now), one per 0088/0089 entry of `[Unreleased]`; DL59: 47 rows in 0089-MADR (C56now to C102now). Each checked with a plain id against a scratch clone's commit of the tree: 12 of 12 and 47 of 47 pass; each planted false: 12 of 12 and 47 of 47 caught.

**Checks**, each exit taken before any filter:

| Command | Host | Exit |
| :--- | :--- | :--- |
| `run-on-host --need linux --race -- go test -race -count=1 ./llmprovider/...` | the Linux machine (linux/amd64), not WSL | 0: 37 packages ok, 2 with no tests |
| `run-heavy -- python3 -B tools/gate.py --timeout 5400` | this Mac (darwin/arm64) | 0: `gate: 30 of 30 passed` (its clone held `[v1.33.0] - 2026-11-24`, since put back to `[Unreleased]`, Markdown only) |
| `run-heavy -- make -k lint vuln parity-check dep-check coverage-check api-check contract-check generate-check records-check tables-check fixtures-check changelog-check docs-check markdownlint links-check disclosure-check gate-selftest tools-test` | this Mac | 0 (lint 0 issues on each target; no vulnerabilities) |
| `go vet -tags live_gateways ./...` | this Mac | 0 |
| `python3 -B tools/apidiff.py v1.32.0` | this Mac | 0, 21 lines |
| `python3 -B tools/claims.py` on 0088-, 0089-, 0090- and 0091-MADR | this Mac | 0 each: 140 of 140, 49 of 49, 7 of 7, 109 of 109 |
| `python3 -B tools/recordlint.py --changed`; `python3 -B tools/denyscan.py --changed` | this Mac | 0 (6 records, no rule rose); 0 (0 hits) |
| `make changelog-check records-check` on the final tree | this Mac | 0 (46 sections; 193 records) |

**`apidiff`** (`python3 -B tools/apidiff.py v1.32.0`), exit 0: the 21 lines the merged phase and Phase 1b recorded, nothing else (compared line for line with Phase 1b's expected list).

**Release preparation (Phase 4 step 2, adapted).** `CHANGELOG.md`'s `[Unreleased]` was dated `[v1.33.0] - 2026-11-24` with its compare links, `make changelog-check` passed (47 sections), then put back to `[Unreleased]` at the coordinator's word: the caller-limit refusal report's entries are to follow, so the date step is pending. `README.md`'s "current release" line still reads `v1.31.0` (v1.32.0's preparation left it); it moves to `v1.33.0` with the date step. No other version string is kept. `tools/consumer.py --replace` (17 packages, six platforms) and `tools/offline.py --replace` (its plant caught): exit 0 each.

**Findings, recorded:** `make changelog-check` does not read the compare links at the file's end (a missing `[v1.33.0]:` link and an `[Unreleased]:` link left at `v1.32.0` both passed); a follow-up for v1.34.0. C146now and C147now are superseded by C160now and C153now (their lines moved with Phase 1c's entry).

**Not done:** Phase 3's short release review; the caller-limit refusal report's phase and the gates again on its commit; the date step; Phase 4's tag and GitHub Release (the owner's word); 0089's DL70 regeneration and DL65 measurement, put to the owner.
